// phase.go 实现 P2-02 的外层闭环状态机(PRD §5 主链路的 Execute → Verify →
// Repair → Accept/Escalate 段;ADR-0001 §1/§3)。
//
// 纪律:
//
//   - 转移表是唯一权威:任何状态迁移都必须经 step(或 CanTransition 预检),
//     不存在第二条改相位的路径;表是编译期常量,确定性、无时钟、无随机;
//   - 终态吸收:Accepted / Escalated / Cancelled 没有任何出边;
//   - 没有通用 workflow DSL:边集固定,不可配置、不可运行期增删
//     (P2-02 AC1);修复只允许从 Verifying 进入(可信 FAIL 且裁决允许且
//     预算允许时,由 Engine 裁量后走 EventRepair),Executing 永远不能直接
//     跳到 Accepted —— 候选必须先过独立验证。
package closedloop

// Phase 是外层闭环的一个状态。
type Phase string

const (
	// PhasePending:已构造、未启动。唯一入口状态。
	PhasePending Phase = "pending"
	// PhaseExecuting:一次尝试的 CodegenRun 在途(第 1 次尝试;内层迭代
	// 对外层不可见,ADR-0001 §1)。
	PhaseExecuting Phase = "executing"
	// PhaseVerifying:当前候选在独立验证服务下在途。
	PhaseVerifying Phase = "verifying"
	// PhaseRepairing:一次修复尝试的 CodegenRun 在途(第 ≥2 次尝试;
	// 必然是新 Run + 新一次性 worktree,ADR-0001 §2)。
	PhaseRepairing Phase = "repairing"
	// PhaseAccepted(终态):候选通过独立验证,可接受;结果必须携带
	// basis(INV:每个 VERIFIED-like 结果都带依据)。
	PhaseAccepted Phase = "accepted"
	// PhaseEscalated(终态):带类型化原因升级给人;fail-safe 落点。
	PhaseEscalated Phase = "escalated"
	// PhaseCancelled(终态):调用方取消;无 verdict、无预算消耗,
	// 半途工作随一次性 worktree 丢弃(ADR-0001 §3)。
	PhaseCancelled Phase = "cancelled"
)

// IsValid 报告 p 是否是状态机词表内的合法状态。
func (p Phase) IsValid() bool {
	switch p {
	case PhasePending, PhaseExecuting, PhaseVerifying, PhaseRepairing,
		PhaseAccepted, PhaseEscalated, PhaseCancelled:
		return true
	default:
		return false
	}
}

// Terminal 报告 p 是否是终态(终态没有出边,Engine 落入即返回)。
func (p Phase) Terminal() bool {
	switch p {
	case PhaseAccepted, PhaseEscalated, PhaseCancelled:
		return true
	default:
		return false
	}
}

// Active 报告 p 是否是需要 Engine 推进的非终态(含唯一入口 Pending:
// 它有待执行的 EventStart 出边)。
func (p Phase) Active() bool { return p.IsValid() && !p.Terminal() }

// Event 是驱动状态机的一次确定性事件(Engine 的内部裁决词汇)。
type Event uint8

const (
	// EventStart:启动首次尝试(Pending → Executing)。
	EventStart Event = iota
	// EventCandidateReady:本次尝试固定了一个候选快照(Executing/Repairing
	// → Verifying);候选必须全新(身份从未出现在谱系中)。
	EventCandidateReady
	// EventVerified:独立验证给出可信正面 verdict(Verifying → Accepted)。
	EventVerified
	// EventRepair:可信 FAIL 被裁决为可修复、且修复预算允许下一次尝试
	// (Verifying → Repairing;新 Run + 新 ChangeSet + 取代事实)。
	EventRepair
	// EventEscalate:终止循环并升级(任一活动态 → Escalated;升级原因由
	// Escalation 载荷区分:无可验证物、基础设施失败、路由裁决、fail-safe)。
	EventEscalate
	// EventCancel:调用方取消(任一活动态 → Cancelled)。
	EventCancel
)

// String 返回事件的规范名(诊断/测试用途)。
func (ev Event) String() string {
	switch ev {
	case EventStart:
		return "start"
	case EventCandidateReady:
		return "candidate_ready"
	case EventVerified:
		return "verified"
	case EventRepair:
		return "repair"
	case EventEscalate:
		return "escalate"
	case EventCancel:
		return "cancel"
	default:
		return "Event(unknown)"
	}
}

// transitions 是外层闭环的确定性转移表(唯一权威;ADR-0001 §1/§2/§3)。
// 终态刻意缺席:终态没有任何出边。升级(EventEscalate)从任一活动态可达
// —— 升级原因由 Escalation 载荷类型化区分,不靠状态爆炸表达。
var transitions = map[Phase]map[Event]Phase{
	PhasePending: {
		EventStart:  PhaseExecuting,
		EventCancel: PhaseCancelled,
	},
	PhaseExecuting: {
		EventCandidateReady: PhaseVerifying,
		EventEscalate:       PhaseEscalated,
		EventCancel:         PhaseCancelled,
	},
	PhaseVerifying: {
		EventVerified: PhaseAccepted,
		EventRepair:   PhaseRepairing,
		EventEscalate: PhaseEscalated,
		EventCancel:   PhaseCancelled,
	},
	PhaseRepairing: {
		EventCandidateReady: PhaseVerifying,
		EventEscalate:       PhaseEscalated,
		EventCancel:         PhaseCancelled,
	},
}

// step 按转移表推进一步;无边可走时返回 (from, false)(调用方按
// fail-safe 处理;Engine 内部事件按构造只发合法事件,miss 属于不变量
// 违背,同样落 fail-safe 升级)。
func step(from Phase, ev Event) (Phase, bool) {
	if !from.IsValid() || !from.Active() {
		return from, false
	}
	to, ok := transitions[from][ev]
	if !ok || !to.IsValid() {
		return from, false
	}
	return to, true
}

// CanTransition 报告 from → to 是否是转移表中的一条边(纯查询,不改状态)。
func CanTransition(from, to Phase) bool {
	if !from.Active() {
		return false
	}
	for _, edge := range transitions[from] {
		if edge == to {
			return true
		}
	}
	return false
}
