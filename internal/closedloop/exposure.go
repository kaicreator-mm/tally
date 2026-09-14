// exposure.go 实现 P2-06:held-out 修复安全边界中的**暴露核算**半边
// (ADR-0001 §5;INV-V14-07/08;决策锁 10/23)。
//
// # 语义(ADR-0001 §5,逐条落进类型)
//
// 修复期间,任何超出 RepairFeedback allowlist 的 held-out 材料被蓄意暴露给
// 实现权威(误配置、未来特性、人的决定)时:
//
//   - 追加一条暴露事实(evidence.Exposure,append-only,INV-V14-08):
//     RecordExposure 是唯一入口,事实经注入的持久化出口落库获得身份,再
//     进入本 Run 的在账集合;持久化失败时事实不进账、调用方拿到类型化
//     error —— 未被记账的暴露绝不静默影响裁决(调用方负责 fail-safe 终止);
//   - 有效 assurance 只能保持或下降,绝不回升:账内任何 compromise 当前
//     候选谱系的暴露事实,都封顶其 assurance(engine.go 的 accept 守卫把它
//     变成机械行为 —— Verified 在先也不得 accept,fail-safe 升级);
//   - 谱系单调:事实的 compromise 判定用 evidence.Exposure.Compromises
//     (锚定谱系 / 持久通道 / 保守时间规则),祖先锚定的暴露命中所有后代
//     —— 后面的「干净」不能恢复前面的暴露;lineage 展开是 engine 的
//     provenance 责任(本文件的 exposureLineage);
//   - 「语义明确说它不构成 compromise」的显式出口就是既有领域规则本身:
//     锚定在谱系外、且候选可证明固化于暴露之前的事实 Compromises=false,
//     不触发降档 —— 不引入第二条豁免通道。
//
// 暴露事实本身不携带 held-out 内容、也不携带可识别 verdict;Reason 只是
// 追溯 provenance 且长度有界(INV-VER-05),越界一律类型化报错。
//
// 并发:验证/执行适配器可能在 Run 在途时蓄意披露并记账(这正是「修复期间
// 暴露」的物理位置),ExposureLedger 以互斥锁保护追加与读取;append-only
// 语义与 evidence.ExposureLog 一致(无删除/改写路径)。
package closedloop

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"tally/internal/changeset"
	"tally/internal/evidence"
)

// 哨兵错误:调用方用 errors.Is 判因,不做字符串匹配。
var (
	// ErrInvalidExposureSpec:蓄意暴露的构造输入不合法(task/binding/channel
	// 缺失、provenance 超界)。事实绝不带病落库。
	ErrInvalidExposureSpec = errors.New("closedloop: 暴露事实构造输入不合法")
	// ErrExposureSinkUnavailable:持久化出口缺席时记账。INV-V14-08 要求暴露
	// 必须成为 append-only 事实;没有出口的记账会静默丢失事实,拒绝执行
	// (fail-safe),绝不退化为只进内存账。
	ErrExposureSinkUnavailable = errors.New("closedloop: 暴露事实持久化出口缺席,拒绝记账")
	// ErrInvalidExposureLedger:构造在账集合时种子事实不合法(身份/通道/
	// 时刻缺失)或身份重复。账面从建立起就必须干净,绝不带病起步。
	ErrInvalidExposureLedger = errors.New("closedloop: 暴露在账集合构造不合法")
)

// EscalationExposureCompromise 是 accept 守卫的 fail-safe 升级原因:在账的
// 暴露事实 compromise 了被验证候选的谱系 —— strict 信息独立主张作废
// (INV-V14-07),若验收仍要求 strict 独立则不能自动接受(ADR-0001 §5),
// 有效 assurance 只能降档/升级,绝不静默保持(INV-V14-08)。
const EscalationExposureCompromise EscalationReason = "exposure_compromise"

// MaxExposureReasonLen 是暴露 provenance 的 rune 上界:provenance 只描述
// 「哪里发生了什么」,有界规模杜绝它被当成内容载荷通道(INV-VER-05)。
const MaxExposureReasonLen = 512

// RepairExposureSpec 是记录一条蓄意披露事实的类型化构造输入(修复期间,
// 超出 allowlist 的 held-out 材料进入了实现权威可见通道)。它不携带也不
// 能携带 held-out 内容;身份由持久化出口在追加时分配(P1-12 合同:身份
// 在落库前不存在,重复身份是错误不是更新)。
type RepairExposureSpec struct {
	// TaskID 是闭环任务 id(表列必填;binding ref 刻意不含 task)。
	TaskID string
	// Binding 是被暴露的 held-out binding 的消费键(§8.6.1)。
	Binding evidence.BindingRef
	// Acceptance 可空:binding 所属验收条款。
	Acceptance evidence.AcceptanceID
	// Anchor 可空:暴露锚定的候选(内容暴露可能先于任何具体候选被观测)。
	Anchor changeset.ChangeSetID
	// Channel 是材料进入的可见通道(evidence.ExposureChannel,开放词表;
	// 未知通道按持久/全局保守分类)。
	Channel evidence.ExposureChannel
	// Reason 是可空追溯 provenance:绝不得携带 held-out 内容或可识别
	// verdict,超界类型化报错(INV-VER-05)。
	Reason string
	// ExposedAt 是观测时刻;零值交由持久化出口取当下(与 runstore 合同
	// 一致),本层不发明时钟。
	ExposedAt time.Time
}

func (s RepairExposureSpec) validate() error {
	switch {
	case strings.TrimSpace(s.TaskID) == "":
		return fmt.Errorf("%w: task id 不能为空", ErrInvalidExposureSpec)
	case s.Binding.Project == "" || s.Binding.Verification == "" || s.Binding.BindingID == "":
		return fmt.Errorf("%w: binding ref 不完整", ErrInvalidExposureSpec)
	case s.Channel == "":
		return fmt.Errorf("%w: channel 不能为空", ErrInvalidExposureSpec)
	case utf8.RuneCountInString(s.Reason) > MaxExposureReasonLen:
		return fmt.Errorf("%w: provenance 超界(> %d rune)", ErrInvalidExposureSpec, MaxExposureReasonLen)
	default:
		return nil
	}
}

// ExposureSink 是蓄意披露事实的 append-only 持久化出口端口(组合根接线,
// 生产面是 runstore 的 evidence_exposure 适配)。刻意只有追加语义:暴露
// 只增不改(INV-V14-08);实现必须在追加成功后返回带已分配身份的领域
// 事实。
type ExposureSink interface {
	RecordExposure(ctx context.Context, spec RepairExposureSpec) (evidence.Exposure, error)
}

// ExposureSinkFunc 把函数适配为 ExposureSink(组合根/e2e 用闭包包装
// runstore.RecordExposure,避免领域包反向依赖持久化适配层)。
type ExposureSinkFunc func(ctx context.Context, spec RepairExposureSpec) (evidence.Exposure, error)

// RecordExposure 实现 ExposureSink。
func (f ExposureSinkFunc) RecordExposure(ctx context.Context, spec RepairExposureSpec) (evidence.Exposure, error) {
	return f(ctx, spec)
}

// ExposureLedger 是一次闭环 Run 的在账暴露集合:种子事实(Run 开始前已在
// force 的 append-only 投影,由组合根从存储读出)+ Run 在途经 RecordExposure
// 追加的事实。它在两类消费点之间保持同一账面:
//
//   - 写入面:执行/验证适配器在 Run 在途蓄意披露时调用 RecordExposure
//     (先持久化取得身份,再进账;持久化失败不进账并返回类型化 error);
//   - 裁决面:engine 的 accept 守卫用 Compromises 以候选谱系求值 —— 账内
//     任一 compromise 的事实都封顶该候选的 assurance。
//
// append-only:进账的事实没有删除/改写路径;重复身份是错误不是更新。
type ExposureLedger struct {
	mu   sync.Mutex
	sink ExposureSink
	log  *evidence.ExposureLog
}

// NewExposureLedger 构造在账集合。sink 可为 nil(账面只读:种子事实已在他
// 处落库,Run 在途记账会被拒绝);seed 是 Run 前已成立的暴露事实,逐条校验
// 并保持 append-only(身份重复拒绝构造)。
func NewExposureLedger(sink ExposureSink, seed []evidence.Exposure) (*ExposureLedger, error) {
	l := &ExposureLedger{sink: sink, log: evidence.NewExposureLog()}
	for _, e := range seed {
		if err := l.log.Append(e); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidExposureLedger, err)
		}
	}
	return l, nil
}

// RecordExposure 记账一条蓄意披露:先经持久化出口追加(身份在落库时分配),
// 成功后进入在账集合。持久化出口缺席或失败时事实不进账并返回类型化
// error —— 未被记账的暴露绝不静默影响裁决,也绝不伪装成已记账。
func (l *ExposureLedger) RecordExposure(ctx context.Context, spec RepairExposureSpec) (evidence.Exposure, error) {
	if err := spec.validate(); err != nil {
		return evidence.Exposure{}, err
	}
	if l.sink == nil {
		return evidence.Exposure{}, ErrExposureSinkUnavailable
	}
	exp, err := l.sink.RecordExposure(ctx, spec)
	if err != nil {
		return evidence.Exposure{}, err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.log.Append(exp); err != nil {
		return evidence.Exposure{}, err
	}
	return exp, nil
}

// Facts 返回在账事实的按追加顺序副本(含种子)。
func (l *ExposureLedger) Facts() []evidence.Exposure {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.log.Exposures()
}

// Compromises 以候选谱系与固化时刻求值在账事实,返回封顶该候选 assurance
// 的最重暴露(若存在)。scope 是最重通道分类(persistent > lineage):
// 任何持久通道事实封顶 persistent,否则 lineage —— 结果与事实顺序无关,
// 后续「干净」不能解除既有封顶(谱系单调,INV-V14-08)。无 compromise 时
// ok=false,assurance 不受暴露影响。
func (l *ExposureLedger) Compromises(lineage []evidence.ChangeSetID, fixedAt time.Time) (evidence.Exposure, evidence.ChannelScope, bool) {
	l.mu.Lock()
	facts := l.log.Exposures()
	l.mu.Unlock()
	var worstFact evidence.Exposure
	worstScope := evidence.ChannelScope("")
	found := false
	for _, e := range facts {
		if !e.Compromises(lineage, fixedAt) {
			continue
		}
		scope := evidence.ChannelScopeOf(e.Channel())
		if !found || (worstScope == evidence.ChannelScopeLineage && scope == evidence.ChannelScopePersistent) {
			worstFact, worstScope = e, scope
		}
		found = true
	}
	return worstFact, worstScope, found
}

// exposureLineage 把已完成尝试谱系投影为候选 change lineage(证据侧词表):
// 当前候选在前,被取代祖先按由近到远排后;无候选的尝试不参与。这是 engine
// 的 provenance 责任 —— 不完整谱系只会少报锚定 compromise,本循环的谱系由
// 取代事实机械完整(ADR-0001 §2)。
func exposureLineage(attempts []Attempt, current changeset.ChangeSetID) []evidence.ChangeSetID {
	out := []evidence.ChangeSetID{evidence.ChangeSetID(current)}
	for i := len(attempts) - 1; i >= 0; i-- {
		cs := attempts[i].ChangeSet
		if cs == "" || evidence.ChangeSetID(cs) == out[len(out)-1] {
			continue
		}
		out = append(out, evidence.ChangeSetID(cs))
	}
	return out
}
