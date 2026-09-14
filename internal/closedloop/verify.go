// verify.go 实现 P2-08 的 Verifier 生产适配器(ports.go 的 Verifier 端口):
// 把 P2-01 抽取的候选验证服务(internal/candidateverify.Service)接入外层
// 闭环 ——
//
//   - 每次调用对恰好一个冻结候选做一次全新独立验证:冻结契约原样透传
//     (修复尝试用与原始验证相同的冻结契约复验,绝不放宽 —— ADR-0001
//     后果节;契约 pin 由构造时绑定,循环不接触契约材料);
//   - FinalOutcome 是适配器已记录的类型化事实面:Router 分类(P2-03
//     ClassResolver + ClassifyFinalStatus)与安全反馈构造(P2-04)从同一
//     账面读取,append-only(同身份不同结论拒绝覆盖 —— verdict 绑定精确
//     ChangeSet,INV-V14-06);
//   - verdict 映射(FAIL ≠ ERROR 端到端保持,ADR-0001 §3):
//     FinalVerified    → VerdictVerified(RunRef = 验证运行行 id;缺席 =
//     不可信结果,由 Engine fail-safe 升级,绝不伪造
//     basis);
//     FinalFailed      → VerdictFailed(可信负面 verdict;唯一进入路由的
//     词,P2-03 分类 repairable 才修复);
//     FinalNeedsReview → VerdictFailed(可信的非 VERIFIED 结论:它不是
//     "无法给出 verdict",而是"判定需要人/政策裁决"。
//     经 Router 按 ClassifyFinalStatus 分类为
//     requires_human_judgment → 确定性升级,结构上
//     不可修复 —— 绝不伪装成候选缺陷,也绝不冒充
//     ERROR);
//     其他/未知        → VerdictInconclusive(fail-safe:不可信、不可
//     accept、不可修复、不耗预算);
//   - ERROR 语义(service error):environment / provider 类(经 P2-03
//     ClassifyError 的 errors.Is 判因)走 infra 复验路径 —— **有界**重试
//     (本适配器自持,默认 2 次,硬上限 5),不算候选失败、绝不消耗修复
//     预算;取消优先于一切(取消的验证产出不构成 verdict,也不重试);
//     其余类(契约缺陷/纪律冲突/unknown)立即上抛,由 Engine fail-safe
//     升级(INV-V14-09:unknown 绝不静默选最弱路径)。
package closedloop

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"tally/internal/candidateverify"
	"tally/internal/changeset"
	"tally/internal/verification"
)

// 哨兵错误:调用方用 errors.Is 判因,不做字符串匹配。
var (
	// ErrInvalidVerifier:验证适配器构造输入不合法(服务/契约/runtime 缺席,
	// 复验次数越界)。
	ErrInvalidVerifier = errors.New("closedloop: 验证适配器构造不合法")
	// ErrInvalidVerifyCall:VerifyRequest 不完整或缺冻结快照/身份不一致。
	// Engine 的落点是 fail-safe 升级,绝不验证未知内容。
	ErrInvalidVerifyCall = errors.New("closedloop: 验证请求不完整或身份不一致")
	// ErrOutcomeAppendOnly:同一验证事实身份出现不同结论(append-only 违背)。
	ErrOutcomeAppendOnly = errors.New("closedloop: 验证结论事实只增不改")
)

// 复验预算界(ADR-0001 §3:infra 复验路径有自己的有界重试;默认值是策略
// 性质的选择,硬上限守住"有限性"不变量 —— 与 RepairPolicy 完全独立,两种
// 预算互不相通)。
const (
	// DefaultInfraVerifyAttempts 是 ERROR 复验的默认总次数(含首次)。
	DefaultInfraVerifyAttempts = 2
	// HardInfraVerifyAttempts 是复验次数的硬上限。
	HardInfraVerifyAttempts = 5
)

// CandidateVerification 是候选验证服务的消费侧窄缝(小接口由消费方定义;
// 生产面 *candidateverify.Service 原生满足,测试注入 fake)。
type CandidateVerification interface {
	Verify(ctx context.Context, req candidateverify.Request) (*verification.FinalOutcome, error)
}

// ServiceVerifierConfig 是 NewServiceVerifier 的依赖面。Service / Contract /
// Runtime 必填(candidateverify.Request 三者缺一即拒绝,构造期一并拒绝);
// InfraAttempts 为 0 时取默认,越界拒绝。
type ServiceVerifierConfig struct {
	// Service 是候选验证服务缝。
	Service CandidateVerification
	// Contract 是冻结验证契约(验证权威;执行/复验/反馈同一份,绝不放宽)。
	Contract *verification.Frozen
	// Runtime 是执行后端身份(verification_run 观测坐标;只作归因)。
	Runtime string
	// InfraAttempts 是 ERROR(environment/provider)复验的总次数上界
	// (含首次;0 = DefaultInfraVerifyAttempts)。
	InfraAttempts int
}

// ServiceVerifier 是 Verifier 端口的生产适配:candidateverify 服务 → verdict,
// 并持有本 Run 已记录的 FinalOutcome 账面(Router 分类与修复反馈的事实源)。
type ServiceVerifier struct {
	service       CandidateVerification
	contract      *verification.Frozen
	runtime       string
	infraAttempts int
	outcomes      *outcomeLedger
}

// Verifier 端口编译期钉住。
var _ Verifier = (*ServiceVerifier)(nil)

// NewServiceVerifier 构造验证适配器;输入不合法时拒绝构造(ErrInvalidVerifier)。
func NewServiceVerifier(cfg ServiceVerifierConfig) (*ServiceVerifier, error) {
	switch {
	case cfg.Service == nil:
		return nil, fmt.Errorf("%w: 候选验证服务为 nil", ErrInvalidVerifier)
	case cfg.Contract == nil:
		return nil, fmt.Errorf("%w: 冻结验证契约为 nil", ErrInvalidVerifier)
	case strings.TrimSpace(cfg.Runtime) == "":
		return nil, fmt.Errorf("%w: runtime 身份为空", ErrInvalidVerifier)
	}
	n := cfg.InfraAttempts
	if n == 0 {
		n = DefaultInfraVerifyAttempts
	}
	if n < 1 || n > HardInfraVerifyAttempts {
		return nil, fmt.Errorf("%w: infra 复验次数 %d 超界 [1,%d]", ErrInvalidVerifier, n, HardInfraVerifyAttempts)
	}
	return &ServiceVerifier{
		service:       cfg.Service,
		contract:      cfg.Contract,
		runtime:       cfg.Runtime,
		infraAttempts: n,
		outcomes:      newOutcomeLedger(),
	}, nil
}

// FailureRouter 返回以本适配器记录的 FinalOutcome 为分类来源的 Router
// (P2-03 的 NewFailureRouter;解析缺席 → unknown → 确定性升级,绝不默认修复)。
func (v *ServiceVerifier) FailureRouter() Router {
	return NewFailureRouter(v.outcomes.classify)
}

// FeedbackSource 返回以本适配器记录的 FinalOutcome 与冻结契约为材料源的
// 安全修复反馈构造器(P2-04 FeedbackBuilder;任何构造失败 → feedback_unavailable
// 升级,绝不降格转发)。
func (v *ServiceVerifier) FeedbackSource() FeedbackSource {
	return &contractFeedbackSource{ledger: v.outcomes, contract: v.contract}
}

// VerifyCandidate 实现 Verifier 端口:对精确的 (冻结契约, 冻结候选) 做一次
// 全新独立验证;ERROR 按类走有界复验;结论映射见文件头。
func (v *ServiceVerifier) VerifyCandidate(ctx context.Context, req VerifyRequest) (VerifyOutcome, error) {
	snap := req.Snapshot
	switch {
	case strings.TrimSpace(req.TaskID) == "":
		return VerifyOutcome{}, fmt.Errorf("%w: task id 为空", ErrInvalidVerifyCall)
	case snap == nil || snap.ID() != req.ChangeSet:
		return VerifyOutcome{}, fmt.Errorf("%w: changeset=%q 缺冻结快照或身份不一致", ErrInvalidVerifyCall, req.ChangeSet)
	}

	for attempt := 1; ; attempt++ {
		out, err := v.service.Verify(ctx, candidateverify.Request{
			Contract:  v.contract,
			ChangeSet: snap,
			Runtime:   v.runtime,
		})
		if err == nil {
			if out == nil {
				return VerifyOutcome{}, fmt.Errorf("%w: 服务返回空验证结论", ErrInvalidVerifyCall)
			}
			if rErr := v.outcomes.record(out); rErr != nil {
				return VerifyOutcome{}, rErr
			}
			return VerifyOutcome{
				Verdict: verdictForStatus(out.Status),
				RunRef:  verifyRunRef(out.RunID),
				// 功能覆盖观察原样投影(防御性拷贝):P1-06 验收守卫的事实面。
				// 缺席报告 = 无观察条目,守卫按 fail-safe 处理;本适配器只投影
				// 事实,验收裁决归 Engine 的 AcceptGuard(seam 裁决见 acceptguard.go)。
				Coverage: append([]verification.AcceptanceCoverage(nil), out.Functional.Acceptances...),
			}, nil
		}
		// 取消优先于一切:取消的验证产出不构成 verdict,也不进入复验。
		if ctx.Err() != nil {
			return VerifyOutcome{}, err
		}
		// ADR-0001 §3:只有 environment / provider 类 ERROR 走 infra 复验
		// 路径;契约缺陷/纪律冲突/unknown 立即上抛(fail-safe 升级)。
		switch ClassifyError(err) {
		case FailureEnvironment, FailureProvider:
		default:
			return VerifyOutcome{}, err
		}
		if attempt >= v.infraAttempts {
			return VerifyOutcome{}, err
		}
	}
}

// outcomeLedger 是本 Run 已记录的验证结论事实面(append-only):Router 的
// ClassResolver 与 FeedbackSource 的共享只读事实源。同一 (ChangeSet, RunRef)
// 身份出现不同结论即拒绝 —— verdict 只绑精确候选,事实绝不原地改写。
type outcomeLedger struct {
	mu          sync.Mutex
	byChangeSet map[changeset.ChangeSetID]*verification.FinalOutcome
	byRunRef    map[string]*verification.FinalOutcome
}

func newOutcomeLedger() *outcomeLedger {
	return &outcomeLedger{
		byChangeSet: map[changeset.ChangeSetID]*verification.FinalOutcome{},
		byRunRef:    map[string]*verification.FinalOutcome{},
	}
}

// record 追加一条结论事实(防御性浅拷贝:账面事实不受调用方后续改写影响);
// 任何身份的重复记录都是纪律违背,返回 ErrOutcomeAppendOnly(闭环内每个候选
// 恰好验证一次;重复即接线错误,fail-safe 升级),绝不静默覆盖。
func (l *outcomeLedger) record(out *verification.FinalOutcome) error {
	if out == nil {
		return fmt.Errorf("%w: 结论为 nil", ErrInvalidVerifyCall)
	}
	ref := verifyRunRef(out.RunID)
	l.mu.Lock()
	defer l.mu.Unlock()
	if prev, ok := l.byChangeSet[out.ChangeSetID]; ok {
		return fmt.Errorf("%w: changeset %s 已记录结论(运行 %s)", ErrOutcomeAppendOnly, out.ChangeSetID, verifyRunRef(prev.RunID))
	}
	if ref != "" {
		if _, ok := l.byRunRef[ref]; ok {
			return fmt.Errorf("%w: 验证运行 %s 已记录结论", ErrOutcomeAppendOnly, ref)
		}
	}
	cp := *out
	l.byChangeSet[cp.ChangeSetID] = &cp
	if ref != "" {
		l.byRunRef[ref] = &cp
	}
	return nil
}

// lookup 按 run 引用优先、候选身份兜底取回结论(任一命中即可:两者同属
// 一次失败验证的谱系身份)。
func (l *outcomeLedger) lookup(cs changeset.ChangeSetID, runRef string) *verification.FinalOutcome {
	l.mu.Lock()
	defer l.mu.Unlock()
	if out, ok := l.byRunRef[runRef]; ok && runRef != "" {
		return out
	}
	return l.byChangeSet[cs]
}

// classify 实现 P2-03 的 ClassResolver:已记录结论的类型化分类;结论缺席
// (接线断裂/身份漂移)按 unknown 处理 —— Router 投影为升级,绝不默认修复。
func (l *outcomeLedger) classify(f VerifyFailure) FailureClass {
	out := l.lookup(f.ChangeSet, f.RunRef)
	if out == nil {
		return FailureUnknown
	}
	return ClassifyFinalStatus(out.Status)
}

// verdictForStatus 把验证编排的类型化结论映射为闭环 verdict 词表(映射纪律
// 见文件头;非法值 fail-safe 落 Inconclusive,绝不猜成 verified/failed)。
func verdictForStatus(s verification.FinalStatus) Verdict {
	switch s {
	case verification.FinalVerified:
		return VerdictVerified
	case verification.FinalFailed, verification.FinalNeedsReview:
		return VerdictFailed
	default:
		return VerdictInconclusive
	}
}

// verifyRunRef 把验证运行行 id 投影为 basis/谱系引用:无可信行 id(≤0)时
// 返回空 —— Engine 对 "verified 而无运行引用" fail-safe 升级,绝不伪造 basis。
func verifyRunRef(runID int64) string {
	if runID <= 0 {
		return ""
	}
	return strconv.FormatInt(runID, 10)
}
