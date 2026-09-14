// outcome.go 实现 P6-01:Accepted 终态的 outcome 事实出口(accepted 事件的
// 生产写入方)。
//
// 缝的选择(P6-01 的核心裁决):唯一耐久验收缝是 Engine 的 accept 转移 ——
// FinalVerified(verdict 事实)+ P1-06 验收守卫通过之后的 PhaseAccepted 转移。
// 本出口在**该转移发生之前**被调用:PhaseAccepted 这个终态只在 accepted
// outcome 事实已耐久落成之后才存在(出口已绑定时)。因此不产生第二个验收
// 权威:接受仍由 verdict + AcceptGuard 授权,本出口只观察并记录该裁决,
// 绝不参与授权(观察者,不是裁判)。
//
// 失败政策(显式 fail-safe):出口写入失败(取消除外 —— 写入用不继承取消的
// 派生 context,取消不可能伪造写入失败)→ EscalationOutcomeUnavailable 升级,
// 绝不静默接受、绝不静默丢事件:一次没有耐久 accepted outcome 的接受在事实面
// 上等于丢失了无法事后补采的 §23 事实。升级是终态事实,不进入修复、不耗预算
// (接受裁决已通过,这不是候选缺陷)。
//
// 幂等/重试:一次接受恰产出一条 accepted 事件。StoreAcceptSink 在追加前先
// 读该任务的既有事件流,同一 (task, changeset) 已有 accepted 事件时幂等返回
// 成功(重试/重放不产生重复语义事件);0023 无唯一约束(migrations 不归本
// 任务),检查-追加的原子性由闭环单写者纪律保证(一个 job 租约一个 Engine,
// 一次性 Run)。
package closedloop

import (
	"context"
	"errors"
	"fmt"
	"time"

	"tally/internal/changeset"
	"tally/internal/outcome"
)

// ErrInvalidAcceptSink:AcceptSink 持久化适配构造输入不合法(nil 事件面)。
var ErrInvalidAcceptSink = errors.New("closedloop: accepted outcome 出口构造输入不合法")

// AcceptFact 是一次已完成接受裁决的 outcome 事实面(accepted 事件载荷)。
// 由 Engine 在 accept 缝构造,字段与 Basis 同源:哪一个候选、由哪一次独立
// 验证运行确认、属于哪一次尝试 —— 接受依据与 outcome 事件绑定同一坐标。
type AcceptFact struct {
	// TaskID 是闭环任务 id(outcome 事件的 task 归属)。
	TaskID string
	// ChangeSet 是被接受的不可变候选身份(outcome 事件的 changeset 绑定)。
	ChangeSet changeset.ChangeSetID
	// VerifyRun 是确认该候选的独立验证运行引用(basis 来源,可追溯)。
	VerifyRun string
	// Attempt 是产出该候选的尝试序号。
	Attempt AttemptNo
	// RecordedAt 是接受事实的时刻(Engine 的时钟,非 sink 补记)。
	RecordedAt time.Time
}

// Validate 校验事实面硬约束:任务/候选/验证引用/序号缺一不可,时刻必须显式。
func (f AcceptFact) Validate() error {
	if f.TaskID == "" {
		return fmt.Errorf("closedloop: AcceptFact 缺少 task id")
	}
	if f.ChangeSet == "" {
		return fmt.Errorf("closedloop: AcceptFact 缺少 changeset")
	}
	if f.VerifyRun == "" {
		return fmt.Errorf("closedloop: AcceptFact 缺少验证运行引用(basis 或有,或没有接受)")
	}
	if !f.Attempt.IsValid() {
		return fmt.Errorf("closedloop: AcceptFact 尝试序号非法: %d", int(f.Attempt))
	}
	if f.RecordedAt.IsZero() {
		return fmt.Errorf("closedloop: AcceptFact 缺少事实时刻")
	}
	return nil
}

// AcceptSink 是 accepted outcome 事实的 append-only 出口(P6-01)。刻意只有
// 追加方法:outcome 事实只增不改(决策锁 14;internal/outcome 无任何改写
// API 是第一道防线)。实现必须幂等:同一 (task, changeset) 的重复投递不
// 产生第二条 accepted 语义事件。
type AcceptSink interface {
	// RecordAccepted 追加一条 accepted outcome 事实(绑定 task/changeset)。
	// 幂等:该 (task, changeset) 已有 accepted 事件时返回 nil,不重复落行。
	RecordAccepted(ctx context.Context, fact AcceptFact) error
}

// outcomeEvents 是 StoreAcceptSink 消费的窄事件面(*outcome.Store 原生满足,
// 测试用内存件替换)。只有读流与追加,没有改写入口。
type outcomeEvents interface {
	Record(ctx context.Context, e outcome.Event) (int64, error)
	ForTask(ctx context.Context, taskID string) ([]outcome.Event, error)
}

// StoreAcceptSink 是 AcceptSink 的持久化适配:内部/outcome 事件面之上的
// 幂等 accepted 写入方。
type StoreAcceptSink struct {
	events outcomeEvents
}

// NewStoreAcceptSink 构造持久化 accepted 出口;nil 事件面拒绝构造
// (ErrInvalidAcceptSink,绝不产出半成品出口)。
func NewStoreAcceptSink(events *outcome.Store) (*StoreAcceptSink, error) {
	if events == nil {
		return nil, fmt.Errorf("%w: 事件面为 nil", ErrInvalidAcceptSink)
	}
	return &StoreAcceptSink{events: events}, nil
}

// RecordAccepted 实现 AcceptSink。顺序:先校验事实面(不触库)→ 读该任务
// 既有事件流,同一 (task, changeset) 已有 accepted 事件则幂等返回 → 追加
// accepted 事件。读流失败与写入失败一样原样上抛(Engine 落显式升级,
// 绝不把「读不到」当「没有」—— 静默跳过等于丢事实)。
func (s *StoreAcceptSink) RecordAccepted(ctx context.Context, fact AcceptFact) error {
	if err := fact.Validate(); err != nil {
		return err
	}
	existing, err := s.events.ForTask(ctx, fact.TaskID)
	if err != nil {
		return fmt.Errorf("closedloop: 读任务 %s 的既有 outcome 事件(幂等检查): %w", fact.TaskID, err)
	}
	for _, e := range existing {
		if e.Kind == outcome.KindAccepted && e.ChangesetID == string(fact.ChangeSet) {
			// 幂等:该候选的 accepted 事实已落成,重试/重放不重复追加。
			return nil
		}
	}
	if _, err := s.events.Record(ctx, outcome.Event{
		TaskID:      fact.TaskID,
		ChangesetID: string(fact.ChangeSet),
		Kind:        outcome.KindAccepted,
		RecordedAt:  fact.RecordedAt,
	}); err != nil {
		return fmt.Errorf("closedloop: 追加任务 %s 候选 %s 的 accepted outcome: %w",
			fact.TaskID, string(fact.ChangeSet), err)
	}
	return nil
}
