// Package approval 实现 PRD §18.1/§19 的 HumanApprovalEvidence 领域(P1-23):
// Owner 决定只以「证据」存在(§18.1:Owner decision → HumanApprovalEvidence →
// State Engine → APPROVED,而不是 UPDATE task SET status='APPROVED'),以及
// §8.10 的 Independence Waiver —— 运行期 independence 损失在 Project
// Verification Policy 允许时的显式风险接受 provenance。
//
// 边界(P1):
//   - 本包只定义决定/豁免证据的类型、合法枚举与确定性校验,以及 strict 结论
//     (strict_satisfied|strict_waived,§8.10)的机械判定;不写库(落库归
//     持久化适配任务),不派生 TaskExecutionState,不解释 Review 内容。
//   - 决定与豁免证据一经构造只读(append-only 事实,INV-FACT-01);同一候选
//     后来追加的新决定是新证据,历史证据永不改写(后来 pass 不洗白更早的
//     记录,反之亦然)。
//   - Independence Waiver 是独立的风险接受标注:它永远不能把 compromised
//     证据重新标注为 independent(§8.10:exposure 是 observed fact,policy
//     可以容忍但不能改标签),也不能修复 design-time planned coverage 缺失
//     —— 那是 Freeze 时(§8.5.1)就能发现的 FREEZE FAIL,不属于运行期损失。
package approval

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"tally/internal/core"
)

// Decision 是 Owner 决定方向(§19:migrations/0021 human_approval.decision)。
// 未批准/否决一律显式记 reject;空值非法 —— 决定方向必须由 Owner 显式给出,
// 缺席的决定不是「默认批准」。
type Decision string

const (
	DecisionApprove Decision = "approve"
	DecisionReject  Decision = "reject"
)

// IsValid 报告该值是否是合法决定方向。
func (d Decision) IsValid() bool {
	switch d {
	case DecisionApprove, DecisionReject:
		return true
	}
	return false
}

// 哨兵错误:调用方用 errors.Is 区分,包装文案只提供字段级上下文。
var (
	// ErrInvalidApproval 拒绝身份/决定/原因/裁决人不完整,或与所附豁免矛盾
	// (reject 携带豁免)的决定证据。
	ErrInvalidApproval = errors.New("approval: invalid human approval evidence")
)

// ApprovalSpec 是 NewApproval 的构造输入;构造成功后与产物彻底脱钩。
type ApprovalSpec struct {
	// Project 是决定归属项目(lifecycle 证据坐标,migrations/0021 必填)。
	Project core.ProjectID
	// TaskID 是决定归属任务(必填)。
	TaskID string
	// ChangeSetID 是被审候选(§19:Review 输入是固定 revision 的 ChangeSet)。
	// 允许为空:非候选裁决(如合同评审类决定)没有候选坐标,不编造。
	ChangeSetID string
	// Decision 是决定方向(approve|reject,必填)。
	Decision Decision
	// Reason 是决定原因(事实原样记录,必填;对应 human_approval.reason)。
	Reason string
	// DecidedBy 是裁决人身份(证据的 recorder 侧写;人工决定是 fact-capable
	// 通道,必填)。
	DecidedBy string
	// DecidedAt 是决定时刻(必填;构造时归一化为 UTC)。
	DecidedAt time.Time
	// Waiver 是随批准一并记录的 independence 豁免 provenance(§8.10;可选)。
	// 仅 approve 可携带:reject 与「接受风险放行」矛盾,reject 行永远不应
	// 带 waived_independence 标记。
	Waiver *IndependenceWaiver
}

// HumanApprovalEvidence 是一条 append-only 的 Owner 决定证据(§18.1/§19,
// migrations/0021 human_approval 行的领域投影)。字段全部私有、只读访问;
// 状态权威由 State Engine 从这类证据派生,本类型不携带任何任务状态。
type HumanApprovalEvidence struct {
	project     core.ProjectID
	taskID      string
	changeSetID string
	decision    Decision
	reason      string
	decidedBy   string
	decidedAt   time.Time
	waiver      *IndependenceWaiver
}

// NewApproval 记录一条 Owner 决定证据。project/task/decision/reason/
// decided_by/decided_at 任一缺失、decision 非法、或所附 waiver 非法/
// 与 reject 矛盾时返回 error 且不产生证据;成功时 DecidedAt 归一化为 UTC,
// 后续对 spec 的改动不影响已构造证据。
func NewApproval(spec ApprovalSpec) (HumanApprovalEvidence, error) {
	if spec.Project == "" {
		return HumanApprovalEvidence{}, fmt.Errorf("%w: 缺少 project", ErrInvalidApproval)
	}
	if strings.TrimSpace(spec.TaskID) == "" {
		return HumanApprovalEvidence{}, fmt.Errorf("%w: 缺少 task_id", ErrInvalidApproval)
	}
	if !spec.Decision.IsValid() {
		return HumanApprovalEvidence{}, fmt.Errorf("%w: decision %q 非法,必须是 approve|reject(决定方向必须显式给出)", ErrInvalidApproval, string(spec.Decision))
	}
	if strings.TrimSpace(spec.Reason) == "" {
		return HumanApprovalEvidence{}, fmt.Errorf("%w: 缺少 reason(决定原因必填,§19)", ErrInvalidApproval)
	}
	if strings.TrimSpace(spec.DecidedBy) == "" {
		return HumanApprovalEvidence{}, fmt.Errorf("%w: 缺少 decided_by(裁决人身份必填)", ErrInvalidApproval)
	}
	if spec.DecidedAt.IsZero() {
		return HumanApprovalEvidence{}, fmt.Errorf("%w: 缺少 decided_at(决定证据必须可定位到决定时刻)", ErrInvalidApproval)
	}
	if spec.Waiver != nil {
		if err := spec.Waiver.validate(); err != nil {
			return HumanApprovalEvidence{}, err
		}
		if spec.Decision != DecisionApprove {
			return HumanApprovalEvidence{}, fmt.Errorf("%w: reject 决定不得携带 independence 豁免(豁免是接受风险放行的 provenance,与否决矛盾)", ErrInvalidApproval)
		}
	}
	return HumanApprovalEvidence{
		project:     spec.Project,
		taskID:      spec.TaskID,
		changeSetID: spec.ChangeSetID,
		decision:    spec.Decision,
		reason:      spec.Reason,
		decidedBy:   spec.DecidedBy,
		decidedAt:   spec.DecidedAt.UTC(),
		waiver:      spec.Waiver,
	}, nil
}

// Project 返回决定归属项目。
func (a HumanApprovalEvidence) Project() core.ProjectID { return a.project }

// TaskID 返回决定归属任务。
func (a HumanApprovalEvidence) TaskID() string { return a.taskID }

// ChangeSetID 返回被审候选;第二个返回值为 false 表示非候选裁决(坐标缺席)。
func (a HumanApprovalEvidence) ChangeSetID() (string, bool) {
	return a.changeSetID, a.changeSetID != ""
}

// Decision 返回决定方向。
func (a HumanApprovalEvidence) Decision() Decision { return a.decision }

// Reason 返回决定原因(事实原样记录)。
func (a HumanApprovalEvidence) Reason() string { return a.reason }

// DecidedBy 返回裁决人身份。
func (a HumanApprovalEvidence) DecidedBy() string { return a.decidedBy }

// DecidedAt 返回决定时刻(UTC)。
func (a HumanApprovalEvidence) DecidedAt() time.Time { return a.decidedAt }

// Waiver 返回随本证据记录的豁免 provenance;nil 表示无豁免。
func (a HumanApprovalEvidence) Waiver() *IndependenceWaiver { return a.waiver }

// WaivedIndependence 报告本行是否携带 independence 豁免(对应 human_approval
// 行内 waived_independence 原值)。strict_waived 行与 strict_satisfied 行在
// 证据行级即可区分;任何报表不得把本值为真的行并入 strict_satisfied 呈现
// (§8.10:strict_waived 不得表述为「Strict requirement 已满足」)。
func (a HumanApprovalEvidence) WaivedIndependence() bool { return a.waiver != nil }
