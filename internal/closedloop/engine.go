// engine.go 实现 P2-02 的确定性闭环驱动器:按 phase.go 的转移表推进
// Pending → Executing → Verifying →(Repairing →)Accepted/Escalated/Cancelled,
// 端口(ports.go)全部注入。每次 Run 恰好落入一个终态(AC2:显式取消与
// 终态);领域落点(接受/升级/取消)不是 Go error,error 只表示 misuse。
//
// 取消纪律(ADR-0001 §3,确定性顺序):
//  1. 每次端口调用前检查 ctx:已取消 → 取消终态,被取消的尝试不入谱系;
//  2. 每次端口调用后先于返回值检查 ctx:已取消 → 取消终态,即使端口返回了
//     合法的候选/verdict 也不得采信 —— 取消绝不伪造 verdict;
//  3. ctx 存活下的端口 error → 基础设施 fail-safe 升级,绝不伪装成候选失败。
//
// 修复纪律(ADR-0001 §1/§2):只有可信 FAIL 经 Router 裁决 repair、且
// RepairBudget 允许下一次尝试、且取代事实出口在场,才进入 Repairing;修复
// 候选就位时同步追加 reason=rework 取代事实;候选身份在谱系中出现过的
// 一律 fail-safe 升级(候选绝不复用/改写)。
//
// 修复安全边界(P2-06,ADR-0001 §4/§5;INV-V14-07/08):反馈缝在场时,
// 安全 RepairFeedback 投影构造成功才开始修复 Run,失败一律
// feedback_unavailable 升级(修复编码指令里没有第二条失败材料来路);蓄意
// 诊断暴露经 ExposureLedger 记账(append-only),账内 compromise 候选谱系
// 的暴露封顶其 assurance —— verified 在先也不得 accept,fail-safe 升级,
// strict 主张绝不带病存活。
//
// 继续纪律(ADR-0001 §2,P2-05):每次修复尝试从冻结的前候选继续 ——
// ExecuteRequest.RepairBase 携带最新未被取代候选的内容快照及其工作区基线,
// 适配器在全新一次性 worktree 内确定性物化它;前候选与其工作区绝不被
// 改写,owner 主 worktree 永远不是执行沙箱。
package closedloop

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"tally/internal/changeset"
	"tally/internal/evidence"
)

// Engine 的 misuse 哨兵(errors.Is 判因,不做字符串匹配)。
var (
	// ErrInvalidConfig:构造依赖缺失(Executor / Verifier 为 nil)。没有
	// 执行与验证服务就不存在闭环,拒绝构造半成品。
	ErrInvalidConfig = errors.New("closedloop: Executor 与 Verifier 为必填端口")
	// ErrInvalidRequest:Run 请求不合法(任务 id 为空)。
	ErrInvalidRequest = errors.New("closedloop: task id 不能为空")
	// ErrAlreadyRun:对同一 Engine 二次调用 Run。一次 Run 对象恰好驱动
	// 一个任务的一次闭环执行,不可复用(与 workflow.TaskExecution 同纪律)。
	ErrAlreadyRun = errors.New("closedloop: Engine 已执行,不可复用")
)

// Config 是 NewEngine 的依赖面。Executor / Verifier 必填;Router /
// RepairBudget / LineageSink / FeedbackSource / ExposureSink 缺席时的落点
// 是确定性 fail-safe(见各类型注释),绝不静默放宽。
type Config struct {
	// Executor 是注入的 codegen 服务(必填)。
	Executor Executor
	// Verifier 是注入的独立验证服务(必填)。
	Verifier Verifier
	// Router 是失败路由缝;nil = 失败一律 router_unavailable 升级。
	Router Router
	// RepairBudget 是修复预算缝;nil = 修复结构性不可用(repair_unavailable)。
	RepairBudget RepairBudget
	// LineageSink 是取代事实出口;nil = 修复结构性不可用(没有取代事实的
	// 修复非法,ADR-0001 §2)。
	LineageSink LineageSink
	// FeedbackSource 是修复反馈缝(P2-06);nil = 修复不带反馈材料继续
	// (结构上没有失败材料进入修复 Run 的通道,不存在泄漏面);非 nil 时
	// 构造失败一律 feedback_unavailable 升级,绝不降格转发。
	FeedbackSource FeedbackSource
	// ExposureSink 是蓄意披露事实的持久化出口(P2-06);nil = Run 在途记账
	// 被拒绝(exposure sink unavailable),种子事实仍然参与 accept 守卫。
	ExposureSink ExposureSink
	// Exposures 是 Run 开始前已成立的暴露事实(append-only 投影,组合根
	// 职责);它们与 Run 在途记账的事实一起,按谱系单调规则参与 accept
	// 守卫 —— 后来的验证通过不能解除先前暴露的封顶(INV-V14-08)。
	Exposures []evidence.Exposure
	// AcceptGuard 是验收守卫缝(P1-06,DL-07/A-03):VerdictVerified 的
	// accept 转移上,保证缺口/未匹配风险/豁免条款使候选 fail-safe 升级,
	// 绝不普通接受。nil = 未绑定验收权威(组合根职责;生产组合根 NewLoop
	// 在编译权威在位时恒绑定,见 wiring.go)。实现必须是纯函数。
	AcceptGuard AcceptGuard
	// AcceptSink 是 accepted outcome 事实的 append-only 出口(P6-01):在
	// accept 转移落成之前追加 kind=accepted 事件 —— 出口已绑定时,
	// PhaseAccepted 只在 outcome 事实耐久落成之后存在。nil = 未绑定出口
	// (v1.4 行为;生产组合根在有事实存储时恒绑定,见 wiring.go)。写入
	// 失败 → EscalationOutcomeUnavailable 显式升级,绝不静默接受、绝不
	// 静默丢事件。实现必须幂等(同一 task/changeset 不重复落 accepted)。
	AcceptSink AcceptSink
	// Now 返回取代事实的时刻;nil 则用 time.Now。测试注入固定时钟。
	Now func() time.Time
}

// Engine 驱动一个任务的外层闭环。一次性使用:Run 恰好可调用一次。
type Engine struct {
	exec   Executor
	verify Verifier
	router Router
	budget RepairBudget
	facts  LineageSink
	// feedback 是修复反馈缝(P2-06);nil = 修复不带失败材料。
	feedback FeedbackSource
	// guard 是验收守卫缝(P1-06);nil = 组合根未绑定验收权威(裁决缺席,
	// 不由 Engine 替组合根补位 —— 生产组合根在有编译权威时恒绑定)。
	guard AcceptGuard
	// accept 是 accepted outcome 事实出口(P6-01);nil = 组合根未绑定
	// (v1.4 行为)。绑定时不改变接受裁决(verdict+guard 仍是唯一权威),
	// 只把已成立的接受落成耐久 outcome 事实。
	accept AcceptSink
	// ledger 是本 Run 的暴露在账集合(P2-06,accept 守卫的裁决面);恒非
	// nil —— 无 sink 无种子时为空账,守卫恒通过。
	ledger *ExposureLedger
	now    func() time.Time

	mu   sync.Mutex
	done bool
}

// NewEngine 校验并构造一个闭环驱动器。依赖缺失时拒绝构造(ErrInvalidConfig)。
func NewEngine(cfg Config) (*Engine, error) {
	if cfg.Executor == nil || cfg.Verifier == nil {
		return nil, ErrInvalidConfig
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	ledger, err := NewExposureLedger(cfg.ExposureSink, cfg.Exposures)
	if err != nil {
		return nil, err
	}
	return &Engine{
		exec:     cfg.Executor,
		verify:   cfg.Verifier,
		router:   cfg.Router,
		budget:   cfg.RepairBudget,
		facts:    cfg.LineageSink,
		feedback: cfg.FeedbackSource,
		guard:    cfg.AcceptGuard,
		accept:   cfg.AcceptSink,
		ledger:   ledger,
		now:      now,
	}, nil
}

// ExposureLedger 返回本 Run 的暴露在账集合(P2-06):执行/验证适配器在 Run
// 在途蓄意披露 held-out 材料时,经它记账(先持久化取得身份,再进账);
// engine 的 accept 守卫以同一账面裁决。Run 开始前记录即种子语义应改走
// Config.Exposures —— 账面只增不改。
func (e *Engine) ExposureLedger() *ExposureLedger { return e.ledger }

// Run 执行该任务的一次完整外层闭环,返回终态结果。Go error 仅表示 misuse
// (空任务 id、二次执行);领域落点(接受/升级/取消)全部在 Result 里。
func (e *Engine) Run(ctx context.Context, taskID string) (Result, error) {
	if strings.TrimSpace(taskID) == "" {
		return Result{}, ErrInvalidRequest
	}
	e.mu.Lock()
	if e.done {
		e.mu.Unlock()
		return Result{}, ErrAlreadyRun
	}
	e.done = true
	e.mu.Unlock()

	l := &loop{
		e:      e,
		taskID: taskID,
		seen:   map[changeset.ChangeSetID]bool{},
	}
	return l.run(ctx)
}

// loop 是单次闭环执行的推进内芯(Engine 的一次性状态,与可复用配置分离)。
type loop struct {
	e      *Engine
	taskID string

	phase     Phase
	attempts  []Attempt                      // 已完成尝试谱系(append-only)
	seen      map[changeset.ChangeSetID]bool // 谱系中出现过的候选身份(新鲜性强制)
	lastCand  changeset.ChangeSetID          // 最新未被取代的候选(修复取代对象,ADR-0001 §2)
	lastSnap  *changeset.ChangeSet           // 最新未被取代候选的内容快照(修复继续基 RepairBase)
	parentRun RunID                          // 紧邻前一次尝试的 Run 身份
	attemptNo AttemptNo                      // 在途尝试序号
	nextFedbk *RepairFeedback                // 待交给下一次修复尝试的安全反馈(P2-06)
}

// run 从 Pending 推进到一个终态(唯一入口,恰好调用一次)。
func (l *loop) run(ctx context.Context) (Result, error) {
	base := Result{TaskID: l.taskID}

	// 启动前已取消:无尝试、无 verdict。
	if ctx.Err() != nil {
		return l.cancel(base), nil
	}
	// EventStart:首次尝试总是被允许;修复预算只治理修复(ADR-0001 §1)。
	next, ok := step(PhasePending, EventStart)
	if !ok {
		// 不可达(静态表边)。防御性 fail-safe:进终态,绝不带病推进。
		return l.escalate(base, EscalationInvalidOutcome, ""), nil
	}
	l.phase = next
	l.attemptNo = 1

	for {
		// ---- Execute / Repair:一次全新 CodegenRun(内层对循环不透明) ----
		if ctx.Err() != nil {
			return l.cancel(base), nil
		}
		// 继续纪律(ADR-0001 §2):修复尝试携带冻结的前候选作为继续基;
		// 首次尝试没有前候选。
		var repairBase *changeset.ChangeSet
		if l.attemptNo.IsRepair() {
			if l.lastSnap == nil {
				// 不可达:修复尝试只会在持有一个候选快照后开始。防御性 fail-safe。
				return l.escalate(base, EscalationInvalidOutcome, "修复尝试缺少前一候选快照"), nil
			}
			repairBase = l.lastSnap
		}
		runID := RunIDFor(l.taskID, l.attemptNo)
		// 修复反馈只消费一次:它属于且只属于触发它的那次失败(P2-06)。
		feedback := l.nextFedbk
		l.nextFedbk = nil
		out, err := l.e.exec.ExecuteCandidate(ctx, ExecuteRequest{
			TaskID:     l.taskID,
			Attempt:    l.attemptNo,
			RunID:      runID,
			ParentRun:  l.parentRun,
			RepairBase: repairBase,
			Feedback:   feedback,
		})
		if ctx.Err() != nil {
			// 取消优先于返回值:半途尝试不采信、不入谱系、不耗预算。
			return l.cancel(base), nil
		}
		if err != nil {
			return l.escalate(base, EscalationInfrastructure, err.Error()), nil
		}
		if out.Candidate == nil {
			// 尝试终结但无可固定候选:无可验证物,升级(不是 error,也不
			// 是候选失败事实;该次尝试已完成,入谱系)。
			l.attempts = append(l.attempts, Attempt{No: l.attemptNo, RunID: runID, ParentRun: l.parentRun})
			return l.escalate(base, EscalationNoCandidate, "尝试结束但没有固定候选 ChangeSet"), nil
		}
		cand := out.Candidate.ID()
		if cand == "" || l.seen[cand] {
			// 候选身份为空或复用:候选不可信/谱系被违背(ADR-0001 §2)。
			// 违规尝试不入谱系(重复身份会让谱系歧义),fail-safe 升级。
			return l.escalate(base, EscalationIdentityReuse, "候选身份为空或已在谱系中出现: "+string(cand)), nil
		}

		// ---- 修复候选:先追加取代事实,再入谱系(append-only) ----
		att := Attempt{No: l.attemptNo, RunID: runID, ParentRun: l.parentRun, ChangeSet: cand}
		if l.attemptNo.IsRepair() {
			if l.lastCand == "" {
				// 不可达:Repairing 只能从持有候选的 Verifying 进入。防御性 fail-safe。
				return l.escalate(base, EscalationInvalidOutcome, "修复尝试缺少被取代候选"), nil
			}
			att.Supersedes = l.lastCand
			fact, fErr := changeset.NewSupersededFact(l.lastCand, cand, changeset.SupersedeRework, l.e.now())
			if fErr != nil {
				return l.escalate(base, EscalationInvalidOutcome, fErr.Error()), nil
			}
			if err := l.e.facts.RecordSuperseded(ctx, fact); err != nil {
				if ctx.Err() != nil {
					return l.cancel(base), nil
				}
				return l.escalate(base, EscalationSupersessionFailed, err.Error()), nil
			}
		}

		// ---- 候选入谱系 → Verifying ----
		l.attempts = append(l.attempts, att)
		l.seen[cand] = true
		l.lastCand = cand
		l.lastSnap = out.Candidate // 冻结快照原样保留:修复尝试的继续基
		l.parentRun = runID
		next, ok := step(l.phase, EventCandidateReady)
		if !ok {
			return l.escalate(base, EscalationInvalidOutcome, ""), nil
		}
		l.phase = next

		// ---- Verify:全新独立验证(同一冻结契约,绝不放宽) ----
		if ctx.Err() != nil {
			return l.cancel(base), nil
		}
		vo, err := l.e.verify.VerifyCandidate(ctx, VerifyRequest{
			TaskID:    l.taskID,
			Attempt:   l.attemptNo,
			ChangeSet: cand,
			Snapshot:  out.Candidate,
		})
		if ctx.Err() != nil {
			// 取消优先:已完成尝试保留为事实,验证产出不构成 verdict。
			return l.cancel(base), nil
		}
		if err != nil {
			return l.escalate(base, EscalationInfrastructure, err.Error()), nil
		}
		if !vo.Verdict.IsValid() {
			return l.escalate(base, EscalationInvalidOutcome, "未知 verdict: "+string(vo.Verdict)), nil
		}

		switch vo.Verdict {
		case VerdictVerified:
			// 每个可接受结果必须携带 basis:验证运行引用由验证服务如实报告,
			// 缺席即不可信结果,绝不伪造(INV:basis 或有,或升级)。
			if strings.TrimSpace(vo.RunRef) == "" {
				return l.escalate(base, EscalationInvalidOutcome, "verified 缺少验证运行引用"), nil
			}
			// 暴露守卫(P2-06,INV-V14-07/08;ADR-0001 §5):在账的暴露事实
			// compromise 当前候选谱系时,strict 信息独立主张作废 —— 有效
			// assurance 只能保持或下降,Verified 在先也不得 accept,fail-safe
			// 升级交人裁决(绝不静默保持或自动豁免)。守卫以候选固化时刻与
			// 完整取代谱系求值,后续「干净」不能解除先前封顶。
			lineage := exposureLineage(l.attempts, cand)
			if exp, scope, hit := l.e.ledger.Compromises(lineage, out.Candidate.CreatedAt()); hit {
				return l.escalate(base, EscalationExposureCompromise,
					fmt.Sprintf("exposure %s(channel %s,scope %s)compromises the accepted candidate lineage", exp.ID(), exp.Channel(), scope)), nil
			}
			// 验收守卫(P1-06,DL-07/A-03):FinalVerified 只是验证事实,不是
			// Acceptance —— 编译权威的必需保证档未达成、存在未匹配风险信号或
			// 被豁免条款时,候选不得成为普通 Accepted,fail-safe 升级交人/政策
			// 裁决(绝不伪装成候选失败进入修复:升级发生在 Router 之前,不产
			// verdict 事实、不耗预算)。守卫是纯函数,无可取消 I/O;守卫缺席 =
			// 组合根未绑定验收权威(wiring.go 的职责,不由 Engine 补位)。
			if l.e.guard != nil {
				ruling := l.e.guard.AllowAccept(ctx, AcceptGuardRequest{
					TaskID:    l.taskID,
					Attempt:   l.attemptNo,
					ChangeSet: cand,
					VerifyRun: vo.RunRef,
					Coverage:  vo.Coverage,
				})
				if !ruling.Allowed {
					return l.escalate(base, EscalationAssuranceInsufficient, ruling.Reason), nil
				}
			}
			// accepted outcome 事实(P6-01,决策锁 14/PRD §23):出口绑定时,
			// 在 accept 转移落成之前追加 kind=accepted 事件 —— 顺序语义是
			// 「PhaseAccepted 蕴含耐久 accepted outcome」:事实写不进去,
			// 接受就没有耐久成立,显式 fail-safe 升级(绝不静默接受、绝不
			// 静默丢无法事后补采的事实)。该出口只观察/记录接受裁决,不参与
			// 授权(verdict + AcceptGuard 仍是唯一验收权威)。事实写入用不
			// 继承取消的派生 context:队列取消不可能把已开始的追加伪造成立
			// 失败;写入成功即接受完成,取消不再改写终态。写入前的取消仍是
			// 一等取消终态(接受未最终落成,无 outcome、无 verdict 采信)。
			if l.e.accept != nil {
				if ctx.Err() != nil {
					return l.cancel(base), nil
				}
				if err := l.e.accept.RecordAccepted(context.WithoutCancel(ctx), AcceptFact{
					TaskID:     l.taskID,
					ChangeSet:  cand,
					VerifyRun:  vo.RunRef,
					Attempt:    l.attemptNo,
					RecordedAt: l.e.now(),
				}); err != nil {
					return l.escalate(base, EscalationOutcomeUnavailable, err.Error()), nil
				}
			}
			if next, ok := step(l.phase, EventVerified); ok {
				l.phase = next
			} else {
				return l.escalate(base, EscalationInvalidOutcome, ""), nil
			}
			base.Phase = l.phase
			base.Basis = &Basis{ChangeSet: cand, VerifyRun: vo.RunRef, Attempt: l.attemptNo}
			base.Attempts = CloneAttempts(l.attempts)
			return base, nil

		case VerdictInconclusive:
			// ERROR 语义:不算候选失败、不进修复、不耗预算;infra 复验
			// 路径归 P2-03/P2-08,骨架落点 fail-safe 升级(ADR-0001 §3)。
			return l.escalate(base, EscalationInconclusive, "验证器无法给出可信 verdict"), nil

		case VerdictFailed:
			// 唯一进入路由的 verdict(只有 FAIL 分类 repairable 才进修复)。
		}

		// ---- Route:类型化失败路由(P2-03 实现);无法裁决 = fail-safe ----
		if l.e.router == nil {
			return l.escalate(base, EscalationRouterUnavailable, "未注入失败路由端口"), nil
		}
		ruling, err := l.e.router.RouteFailure(ctx, VerifyFailure{
			TaskID:    l.taskID,
			Attempt:   l.attemptNo,
			ChangeSet: cand,
			RunRef:    vo.RunRef,
		})
		if ctx.Err() != nil {
			return l.cancel(base), nil
		}
		if err != nil {
			return l.escalate(base, EscalationRouterUnavailable, err.Error()), nil
		}
		switch ruling.Route {
		case RouteRepair:
			// 修复前提逐一核对;任何一项不满足都是显式升级,绝不静默改道。
			nextNo := l.attemptNo + 1
			if l.e.facts == nil {
				return l.escalate(base, EscalationRepairUnavailable, "未注入取代事实出口;没有取代事实的修复非法"), nil
			}
			if l.e.budget == nil {
				return l.escalate(base, EscalationRepairUnavailable, "未注入修复预算端口"), nil
			}
			if !l.e.budget.CanRepair(nextNo) {
				return l.escalate(base, EscalationRepairExhausted, "修复预算拒绝下一次修复尝试"), nil
			}
			// 修复反馈边界(P2-06,ADR-0001 §4;INV-V14-07):反馈缝在场时,
			// 安全投影构造成功才开始修复 Run;构造失败(不合法/不安全/缺材
			// 料)或反馈身份与本次失败候选不一致,一律 feedback_unavailable
			// 升级 —— 修复编码指令没有第二条失败材料来路,绝不降格转发、
			// 绝不带着裸 held-out 材料修复(AC4:insufficient safe feedback
			// escalates rather than leaks)。反馈缝缺席 = 修复不带反馈材料
			// (结构上没有失败材料进入通道,不存在泄漏面;组合根 P2-08 决定
			// 是否接线)。
			if l.e.feedback != nil {
				fb, fbErr := l.e.feedback.RepairFeedback(ctx, VerifyFailure{
					TaskID:    l.taskID,
					Attempt:   l.attemptNo,
					ChangeSet: cand,
					RunRef:    vo.RunRef,
				})
				if ctx.Err() != nil {
					return l.cancel(base), nil
				}
				if fbErr != nil {
					return l.escalate(base, EscalationFeedbackUnavailable, fbErr.Error()), nil
				}
				if fb.Failure != FailureRepairable || fb.TaskID != l.taskID || fb.Attempt != l.attemptNo || fb.ChangeSet != cand {
					return l.escalate(base, EscalationFeedbackUnavailable, "反馈身份与本次失败候选不一致"), nil
				}
				l.nextFedbk = &fb
			}
			nextPhase, ok := step(l.phase, EventRepair)
			if !ok {
				return l.escalate(base, EscalationInvalidOutcome, ""), nil
			}
			l.phase = nextPhase
			l.attemptNo = nextNo
			// 循环继续:全新 CodegenRun(新 worktree/新候选由端口契约约束,
			// 身份新鲜性由 seen 强制),候选就位后追加取代事实并再次验证。

		case RouteEscalate:
			if !ruling.Reason.IsValid() {
				return l.escalate(base, EscalationRouterUnavailable, "路由裁决升级但缺少原因"), nil
			}
			return l.escalate(base, ruling.Reason, ""), nil

		default:
			return l.escalate(base, EscalationRouterUnavailable, "未知路由裁决"), nil
		}
	}
}

// escalate 经 EventEscalate 收束为 Escalated 终态(升级原因类型化载荷;
// 表边按构造可达,step 失败时防御性直落终态,绝不带病停在活动态)。
// detail 为空时不携带补充说明。
func (l *loop) escalate(base Result, reason EscalationReason, detail string) Result {
	if next, ok := step(l.phase, EventEscalate); ok {
		l.phase = next
	}
	if !l.phase.Terminal() {
		l.phase = PhaseEscalated
	}
	if !reason.IsValid() {
		reason = EscalationInvalidOutcome
	}
	base.Phase = l.phase
	base.Escalation = &Escalation{Reason: reason, Detail: detail}
	base.Basis = nil
	base.Attempts = CloneAttempts(l.attempts)
	return base
}

// cancel 经 EventCancel 收束为取消终态:无 verdict、无升级载荷、无预算
// 消耗;已完成尝试保留为事实,半途尝试已确保不入谱系。
func (l *loop) cancel(base Result) Result {
	if next, ok := step(l.phase, EventCancel); ok {
		l.phase = next
	}
	if !l.phase.Terminal() {
		l.phase = PhaseCancelled
	}
	base.Phase = l.phase
	base.Basis = nil
	base.Escalation = nil
	base.Attempts = CloneAttempts(l.attempts)
	return base
}
