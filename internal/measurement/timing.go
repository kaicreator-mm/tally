package measurement

import (
	"fmt"
	"strings"
	"time"
)

// LifecyclePhase 是 PRD §25「生命周期阶段分解」的稳定命名枚举(AC3)。
// 这些名字是 §25 阶段分解与 §38.3 阶段 latency 的 join 键:一旦落库即是历史
// 事实的一部分,只能新增,不能改名或复用旧名。
//
//	PhaseIntake                 ← Intake
//	PhaseProductPlanning        ← Product/Planning
//	PhaseVerificationPlanning   ← Verification Planning
//	PhaseQueue                  ← Queue
//	PhaseWorkspaceBootstrap     ← Workspace Bootstrap
//	PhaseAgentExecution         ← Agent Execution
//	PhaseIncrementalValidation  ← Incremental Validation
//	PhaseFinalValidation        ← Final Validation
//	PhaseHumanReview            ← Human Review
//	PhaseIntegration            ← Integration
type LifecyclePhase string

const (
	PhaseIntake                LifecyclePhase = "intake"
	PhaseProductPlanning       LifecyclePhase = "product_planning"
	PhaseVerificationPlanning  LifecyclePhase = "verification_planning"
	PhaseQueue                 LifecyclePhase = "queue"
	PhaseWorkspaceBootstrap    LifecyclePhase = "workspace_bootstrap"
	PhaseAgentExecution        LifecyclePhase = "agent_execution"
	PhaseIncrementalValidation LifecyclePhase = "incremental_validation"
	PhaseFinalValidation       LifecyclePhase = "final_validation"
	PhaseHumanReview           LifecyclePhase = "human_review"
	PhaseIntegration           LifecyclePhase = "integration"
)

// lifecyclePhases 是 PRD §25 的全部阶段,按固定顺序排列;顺序本身也是稳定契约,
// 阶段分解报表按此序呈现。
var lifecyclePhases = []LifecyclePhase{
	PhaseIntake,
	PhaseProductPlanning,
	PhaseVerificationPlanning,
	PhaseQueue,
	PhaseWorkspaceBootstrap,
	PhaseAgentExecution,
	PhaseIncrementalValidation,
	PhaseFinalValidation,
	PhaseHumanReview,
	PhaseIntegration,
}

// LifecyclePhases 返回全部生命周期阶段(PRD §25 顺序的副本)。调用方修改返回
// 切片不影响包内状态。
func LifecyclePhases() []LifecyclePhase {
	out := make([]LifecyclePhase, len(lifecyclePhases))
	copy(out, lifecyclePhases)
	return out
}

// IsValid 报告该值是否是已定义的生命周期阶段(AC3:名字是封闭枚举,不是自由文本)。
func (p LifecyclePhase) IsValid() bool {
	for _, known := range lifecyclePhases {
		if p == known {
			return true
		}
	}
	return false
}

// Milestone 是 PRD §38.3 要求至少记录的生命周期里程碑事件。时间戳在事件发生时
// 写入一次,之后只读;idea_to_verified_latency 等跨度由里程碑时间差推导。
type Milestone string

const (
	MilestoneIdeaReceived          Milestone = "idea_received"
	MilestoneTriageCompleted       Milestone = "triage_completed"
	MilestonePlanCompleted         Milestone = "plan_completed"
	MilestoneAgentStarted          Milestone = "agent_started"
	MilestoneAgentCompleted        Milestone = "agent_completed"
	MilestoneVerificationStarted   Milestone = "verification_started"
	MilestoneVerificationCompleted Milestone = "verification_completed"
	MilestoneReviewStarted         Milestone = "review_started"
	MilestoneReviewCompleted       Milestone = "review_completed"
	MilestoneMerged                Milestone = "merged"
	MilestoneReleased              Milestone = "released"
	MilestoneDeployed              Milestone = "deployed"
	MilestoneStable                Milestone = "stable"
)

// IsValid 报告该值是否是 PRD §38.3 定义的里程碑事件名。
func (m Milestone) IsValid() bool {
	switch m {
	case MilestoneIdeaReceived, MilestoneTriageCompleted, MilestonePlanCompleted,
		MilestoneAgentStarted, MilestoneAgentCompleted,
		MilestoneVerificationStarted, MilestoneVerificationCompleted,
		MilestoneReviewStarted, MilestoneReviewCompleted,
		MilestoneMerged, MilestoneReleased, MilestoneDeployed, MilestoneStable:
		return true
	}
	return false
}

// PhaseSpan 是某任务某阶段的一次连续计时区间。同一阶段允许有多个区间
// (重试/分段计时),汇总时累加。
type PhaseSpan struct {
	Phase LifecyclePhase
	Start time.Time
	End   time.Time
}

// Validate 校验区间:阶段名合法(AC3)、Start/End 显式给出、End 不早于 Start。
func (s PhaseSpan) Validate() error {
	if !s.Phase.IsValid() {
		return fmt.Errorf("measurement: phase span 的 phase %q 非法,必须是 PRD §25 定义的阶段名", string(s.Phase))
	}
	if s.Start.IsZero() || s.End.IsZero() {
		return fmt.Errorf("measurement: phase span %s 缺少 start/end", string(s.Phase))
	}
	if s.End.Before(s.Start) {
		return fmt.Errorf("measurement: phase span %s 的 end 早于 start", string(s.Phase))
	}
	return nil
}

// Duration 返回区间时长。
func (s PhaseSpan) Duration() time.Duration { return s.End.Sub(s.Start) }

// PhaseTiming 是一个任务的生命周期阶段耗时分解记录(PRD §25「同时记录生命周期
// 阶段分解」、§38.3 各阶段 latency 的记录面)。
type PhaseTiming struct {
	TaskID string
	Spans  []PhaseSpan
}

// Validate 校验记录:任务归属必填,每个区间自身合法。
func (t PhaseTiming) Validate() error {
	if strings.TrimSpace(t.TaskID) == "" {
		return fmt.Errorf("measurement: phase timing 缺少 task_id")
	}
	for i, s := range t.Spans {
		if err := s.Validate(); err != nil {
			return fmt.Errorf("measurement: phase timing spans[%d]: %w", i, err)
		}
	}
	return nil
}

// DurationByPhase 按阶段累加区间时长。phase 非法的区间不参与汇总(写入方应先
// Validate;此处保守跳过而不是 panic)。没有区间的阶段不出现在结果里。
func (t PhaseTiming) DurationByPhase() map[LifecyclePhase]time.Duration {
	out := make(map[LifecyclePhase]time.Duration)
	for _, s := range t.Spans {
		if !s.Phase.IsValid() {
			continue
		}
		out[s.Phase] += s.Duration()
	}
	return out
}

// MilestoneTimes 是一个任务已观测到的里程碑时间戳集合(写入时的事实)。
type MilestoneTimes map[Milestone]time.Time

// IdeaToVerifiedLatency 返回 idea_to_verified_latency(PRD §25 至少记录条目,
// §25.1 North Star 的 guardrail):IDEA_RECEIVED → VERIFICATION_COMPLETED 的
// 墙钟时长。任一里程碑缺失或为零值时返回 (0, false):跨度未定义,不猜、
// 不部分计算。
func IdeaToVerifiedLatency(ms MilestoneTimes) (time.Duration, bool) {
	start, ok := ms[MilestoneIdeaReceived]
	if !ok || start.IsZero() {
		return 0, false
	}
	end, ok := ms[MilestoneVerificationCompleted]
	if !ok || end.IsZero() {
		return 0, false
	}
	return end.Sub(start), true
}

// Minutes 把时长换算为 PRD §25 度量口径的单位(分钟,含小数)。
func Minutes(d time.Duration) float64 { return d.Minutes() }
