package measurement

// report.go(P1-51)是 P1 基线度量报告的纯读侧组装:从已落成的度量事实投影
// (TaskBaseline)组装 PRD §25/§38/§36.3 要求的基线分布/读数,不做任何采集、
// 落库或任务状态解释。P0-08 起附带确定性运维呈现(RenderBaselineReport),
// 把报告从测试专用能力接成 CLI 可用的操作员只读输出。
//
// 边界(P1):
//   - 本文件是事实投影上的纯函数:输入由调用方(组合根/门任务)从事实存储、
//     冻结 VerificationContract 与 effective independence 推导报告映射而来;
//     internal/measurement 保持叶子包,不依赖 internal/verification 或
//     internal/evidence。authoring mode / visibility / independence requirement
//     的取值与 verification 域枚举逐字一致(pre_commit|post_hoc、visible|
//     held_out、none|source|information),qualification strength 用 PRD §38.5
//     的报告词表;映射责任在调用方,本包不解释 VerificationContract。
//   - P1 只记录基线分布,不设任何产品价值/就绪阈值(决策锁 16;§25.4:
//     readiness gate 归 P2;§36.3:P1 不预设 held-out/strong-evidence 比率
//     阈值)。本文件唯一的常量 MinSufficientSamples 是报告呈现的样本充分性
//     规则(AC4),它不评判 Tally 的任何表现、不构成通过/失败语义:样本不足
//     的读数必须呈现为「样本不足」,不得当作 0、当作通过,也不得据此做任何
//     优劣或就绪结论。
//   - 全部聚合只读输入事实;不存在对历史的重算或改写:成本归属只按每条记录
//     写入时落成的 cost_plane 累计(§25.3),人工开销按 active/wall 双口径
//     分别累计(§25),缺失的观测保持缺失(AC4),不被补成 0。

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// MinSufficientSamples 是把一个读数呈现为「分布」所需的最低样本数(AC4)。
// 这是报告呈现充分性规则,不是产品价值/就绪阈值:样本数低于它的读数一律
// 标记 Sufficient=false,由消费方呈现为「样本不足」,不得参与任何优劣比较
// 或阈值判断(P1 不设阈值,决策锁 16)。单个样本是个案,不是分布。
const MinSufficientSamples = 2

// ---- 样本充分性与分布读数原语(AC4) ----

// Samples 是一个读数的样本量与充分性标注。Sufficient=false 表示缺失或样本
// 不足:此时读数不构成分布结论,消费方必须按「样本不足」呈现,不得当作 0、
// 当作通过,也不得据此下任何结论。
type Samples struct {
	Count      int
	Sufficient bool
}

func sampleSufficiency(n int) Samples {
	return Samples{Count: n, Sufficient: n >= MinSufficientSamples}
}

// ScalarDistribution 是标量样本的分布读数。Min/Median/Max 仅在 Count>0 时
// 有意义;单位由使用处的字段注释固定(人工开销与延迟为分钟,成本为记账
// 货币 micros)。Sufficient 遵循 AC4:样本不足时读数不构成分布结论。
type ScalarDistribution struct {
	Samples
	Min    float64
	Median float64
	Max    float64
}

func newScalarDistribution(values []float64) ScalarDistribution {
	d := ScalarDistribution{Samples: sampleSufficiency(len(values))}
	if len(values) == 0 {
		return d
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	d.Min = sorted[0]
	d.Max = sorted[len(sorted)-1]
	d.Median = sorted[len(sorted)/2]
	if len(sorted)%2 == 0 {
		d.Median = (sorted[len(sorted)/2-1] + sorted[len(sorted)/2]) / 2
	}
	return d
}

// CategoryDistribution 是封闭词表上的计数分布。词表全部取值都显式出现
// (未观测的取值计数为 0 —— 「没观测到」与「观测到 0 个」都按 0 呈现,
// 但整体 Sufficient=false 标记该维度尚不构成分布,AC4)。
type CategoryDistribution struct {
	// Counts 是词表值 → 观测数(含 0 计数的词表全量取值)。
	Counts     map[string]int
	Total      int
	Sufficient bool
}

func newCategoryDistribution(vocab []string, observed []string) (CategoryDistribution, error) {
	d := CategoryDistribution{
		Counts:     make(map[string]int, len(vocab)),
		Total:      len(observed),
		Sufficient: len(observed) >= MinSufficientSamples,
	}
	allowed := make(map[string]bool, len(vocab))
	for _, v := range vocab {
		d.Counts[v] = 0
		allowed[v] = true
	}
	for _, v := range observed {
		if !allowed[v] {
			return CategoryDistribution{}, fmt.Errorf("measurement: 报告观测值 %q 不在封闭词表内(调用方映射错误,拒绝静默归类)", v)
		}
		d.Counts[v]++
	}
	return d, nil
}

// CrossDistribution 是 strength × visibility 的交叉计数分布(§38.5)。只含
// 观测到的组合;边际分布(QualificationStrength/Visibility)已显式呈现零
// 计数取值。Sufficient 遵循 AC4。
type CrossDistribution struct {
	// Counts 是 strength 词表值 → visibility 词表值 → 观测数。
	Counts     map[string]map[string]int
	Total      int
	Sufficient bool
}

// ---- 报告词表(PRD §38.5;与 internal/verification 域枚举逐字一致) ----

// AuthoringMode 是验证权威的 authoring provenance 词表(INV-VER-01;§38.5
// authoring mode 分布)。取值与 verification 包的枚举一致,映射责任在调用方。
type AuthoringMode string

const (
	ModePreCommit AuthoringMode = "pre_commit"
	ModePostHoc   AuthoringMode = "post_hoc"
)

// IsValid 报告该值是否是已定义的 authoring mode。
func (m AuthoringMode) IsValid() bool {
	switch m {
	case ModePreCommit, ModePostHoc:
		return true
	}
	return false
}

// Visibility 是 binding 可见性词表(§8.4.1;§38.5 visibility 分布)。
type Visibility string

const (
	VisibilityVisible Visibility = "visible"
	VisibilityHeldOut Visibility = "held_out"
)

// IsValid 报告该值是否是已定义的可见性。
func (v Visibility) IsValid() bool {
	switch v {
	case VisibilityVisible, VisibilityHeldOut:
		return true
	}
	return false
}

// IndependenceLevel 是 Acceptance 级独立性要求词表(Decision Lock 10:
// none < source < information;§38.5 planned/effective coverage)。
type IndependenceLevel string

const (
	LevelNone        IndependenceLevel = "none"
	LevelSource      IndependenceLevel = "source"
	LevelInformation IndependenceLevel = "information"
)

// IsValid 报告该值是否是已定义的独立性要求级别。
func (l IndependenceLevel) IsValid() bool {
	switch l {
	case LevelNone, LevelSource, LevelInformation:
		return true
	}
	return false
}

// QualificationStrength 是 PRD §38.5 的 qualification strength 报告词表。
// 它是 §8.3.1 baseline class × mechanical sensitivity 的组合投影
// (assertion→assertion_strong;compile_error→compile_error_weak;panic→
// panic_error_weak;pass→baseline_pass;非 test 证据→n_a),映射责任在调用方。
// ErrorWeak 是 §8.3.1 error 行的弱类事实桶:§38.5 词表未单列,但丢弃已观测
// 的 class 比多一个事实桶更失真,故显式收录。
type QualificationStrength string

const (
	StrengthAssertionStrong  QualificationStrength = "assertion_strong"
	StrengthCompileErrorWeak QualificationStrength = "compile_error_weak"
	StrengthPanicErrorWeak   QualificationStrength = "panic_error_weak"
	StrengthErrorWeak        QualificationStrength = "error_weak"
	StrengthBaselinePass     QualificationStrength = "baseline_pass"
	StrengthNA               QualificationStrength = "n_a"
)

// IsValid 报告该值是否是已定义的 qualification strength。
func (s QualificationStrength) IsValid() bool {
	switch s {
	case StrengthAssertionStrong, StrengthCompileErrorWeak, StrengthPanicErrorWeak,
		StrengthErrorWeak, StrengthBaselinePass, StrengthNA:
		return true
	}
	return false
}

// effortKinds 是 PRD §25 人工开销条目的全量词表(报告呈现顺序;types.go 的
// HumanEffortKind 常量是它的元素)。报告按词表全量呈现,未观测条目显式 0。
var effortKinds = []HumanEffortKind{
	EffortContractReview,
	EffortVerificationReview,
	EffortDiffReview,
	EffortManualCorrection,
	EffortSpecDefectDecision,
}

// 词表固定顺序(新取值只能追加,不能改名或复用:词表值进入报告即是记录面)。
var (
	authoringModeVocab = []AuthoringMode{ModePreCommit, ModePostHoc}
	visibilityVocab    = []Visibility{VisibilityVisible, VisibilityHeldOut}
	independenceVocab  = []IndependenceLevel{LevelNone, LevelSource, LevelInformation}
	strengthVocab      = []QualificationStrength{
		StrengthAssertionStrong, StrengthCompileErrorWeak, StrengthPanicErrorWeak,
		StrengthErrorWeak, StrengthBaselinePass, StrengthNA,
	}
)

func stringVocab[T ~string](vocab []T) []string {
	out := make([]string, len(vocab))
	for i, v := range vocab {
		out[i] = string(v)
	}
	return out
}

// ---- 输入:已落成事实的任务级投影 ----

// BindingFact 是一条验证 binding 的观测事实投影(§38.5 binding 级分布的
// 输入)。authoring mode 与 visibility 来自冻结契约声明;strength 来自
// §8.3.1 qualification 评估的 class×sensitivity 投影。本包不解释契约。
type BindingFact struct {
	BindingID     string
	AuthoringMode AuthoringMode
	Visibility    Visibility
	Strength      QualificationStrength
}

// AcceptanceFact 是 acceptance 级 independence 覆盖的观测投影(§38.5
// planned/effective coverage 与 source/information gaps 的输入;Decision
// Lock 10:independence requirement 属于 Acceptance Clause)。Covered 与
// SourceSatisfied 来自 effective independence 推导(§8.5.2):Covered 报告
// effective coverage 是否成立;SourceSatisfied 报告是否存在 effective source
// independent 的 applicable evidence(仅用于 information 级缺口的维度归因:
// §8.5 规定同一 evidence 必须同时满足 source+information,存在 source 满足
// 而整体不成立即 information 维度缺口)。
type AcceptanceFact struct {
	AcceptanceID string
	// Required 是该 acceptance 的独立性要求;空 = 未声明,按 none 计(§7.3 缺省)。
	Required IndependenceLevel
	Covered  bool
	// SourceSatisfied 仅在 Required=information 且 Covered=false 时参与缺口归因。
	SourceSatisfied bool
}

// FirstPassFact 是一个任务首次最终验证的 first-pass 观测(§38.5 first-pass
// final validation:visible_only / with_holdout)。WithHoldout 报告被验证契约
// 是否含 held_out binding;Passed 报告该次最终验证是否得到 verified(failed
// 与 needs_review 都不是 first-pass PASS;§38.5 first-pass 只记录 pass 方向)。
type FirstPassFact struct {
	TaskID      string
	WithHoldout bool
	Passed      bool
}

// TaskBaseline 是一个任务已落成度量事实的投影(BuildBaselineReport 的输入)。
// 全部字段都是事实的读侧投影;本包不做采集,也不回填任何任务状态字段。
type TaskBaseline struct {
	// TaskID 归属任务(必填,报告内唯一)。
	TaskID string
	// Profile 是任务 profile(§7.2;§38.2 人工开销按 profile 分维)。原样
	// 记录(空 = 未声明);本包不解释 profile 语义,P1 主要产生 standard 数据。
	Profile string
	// Efforts 是该任务的人工控制开销事实(active/wall 双口径分段行)。
	Efforts []HumanEffort
	// Usages 是该任务的 LLM/Agent 调用使用/成本事实(cost_plane 已在调用时落成)。
	Usages []LLMUsage
	// Spans 是该任务的生命周期阶段计时区间(§25 阶段分解;可空)。
	Spans []PhaseSpan
	// Milestones 是该任务已观测的里程碑时刻(可空;idea_to_verified 的输入)。
	Milestones MilestoneTimes
	// Bindings 是该任务冻结契约的 binding 级观测(§38.5;可空)。
	Bindings []BindingFact
	// Acceptances 是该任务 acceptance 级 independence 覆盖观测(可空)。
	Acceptances []AcceptanceFact
	// FirstPass 是该任务首次最终验证的 first-pass 观测(nil = 尚无)。
	FirstPass *FirstPassFact
}

// ---- 报告:基线分布/读数 ----

// BaselineReport 是 P1 基线分布/读数(§25/§38/§36.3)。全部数值都从输入
// 事实推导;样本不足的读数 Sufficient=false(AC4),消费方必须呈现为
// 「样本不足」,不得当作 0、当作通过,也不得据此做任何就绪/价值结论
// (P1 只记录分布,不设阈值 —— 决策锁 16)。
type BaselineReport struct {
	// GeneratedAt 是报告组装时刻(显式传入,报告必须可定位到生成时间)。
	GeneratedAt time.Time
	// Tasks 是进入报告的任务观测数。
	Tasks int
	// SourcePeriodFrom / SourcePeriodTo 是输入事实观测时刻的最早/最晚值
	// (P0-08 AC3:报告必须可定位到覆盖的历史期间)。观测时刻取事实行自带的
	// 时间:effort/usage 的 recorded_at、phase span 的 start/end、里程碑时刻。
	// SourcePeriodValid=false 表示输入不含任何带时刻的事实 —— 覆盖期间未知,
	// 消费方必须显式呈现「期间未知」,不得用 GeneratedAt 或 0 伪充(AC2)。
	SourcePeriodFrom  time.Time
	SourcePeriodTo    time.Time
	SourcePeriodValid bool

	Human     HumanReport
	Cost      CostReport
	Phases    PhaseReport
	Latency   LatencyReport
	Trust     TrustReport
	FirstPass FirstPassReport
}

// HumanReport 是人工控制开销读数(§25「至少记录」条目;§38.2 human effort)。
type HumanReport struct {
	// ByKind 按条目汇总 active/wall;PRD §25 全部条目都出现(未观测条目为
	// 零值、Sufficient=false),不省略、不补 0 充数(AC4)。
	ByKind map[HumanEffortKind]KindEffort
	// PerTaskActiveMinutes 是每任务人工 active 分钟合计的样本分布
	// (§38.2 human minutes / verified task 的样本面;单位:分钟)。
	PerTaskActiveMinutes ScalarDistribution
}

// KindEffort 是一条目(如 contract_review_minutes)的双口径汇总。
type KindEffort struct {
	Active  time.Duration // active 合计(North Star 分母口径:人实际投入)
	Wall    time.Duration // wall 合计(端到端等待口径)
	Samples Samples       // 分段事实行数及其充分性
}

// CostReport 是 LLM/Agent 使用与成本读数(§25.3/§38.4)。
type CostReport struct {
	// ByPlane 按每条记录写入时落成的 cost_plane 聚合(两面都出现;未观测
	// 面为零值、Sufficient=false)。历史永不重算:只读存储的归属面。
	ByPlane map[CostPlane]PlaneTotals
	// ControlRatio 是 control_plane_llm_cost_ratio(§25.3/§38.4);
	// RatioValid=false 表示总成本为 0,比率未定义 —— 不猜、不补 0(AC4)。
	ControlRatio float64
	RatioValid   bool
	// PerTaskCostMicros 是每任务成本合计的样本分布(§38.4 cost per verified
	// task 的样本面;单位:记账货币 micros)。
	PerTaskCostMicros ScalarDistribution
}

// PlaneTotals 是一个成本归属面的聚合读数。
type PlaneTotals struct {
	CostMicros   int64 // 记账货币 1e-6 整数(与事实行同口径)
	InputTokens  int64
	OutputTokens int64
	Calls        int64
	Samples      Samples
}

// PhaseReport 是生命周期阶段墙钟分解(§25 阶段分解;§38.3 阶段 latency 的
// 记录面)。全部 §25 阶段都出现(未观测阶段为零值、Sufficient=false)。
type PhaseReport struct {
	WallByPhase map[LifecyclePhase]PhaseTotals
}

// PhaseTotals 是一个阶段的墙钟合计读数。
type PhaseTotals struct {
	Wall    time.Duration
	Samples Samples // 计时区间段数及其充分性
}

// LatencyReport 是里程碑跨度读数。
type LatencyReport struct {
	// IdeaToVerifiedMinutes 是 idea_to_verified_latency 的样本分布(§25.1
	// North Star 的 guardrail;单位:分钟)。两个端点里程碑齐备的任务才有样本。
	IdeaToVerifiedMinutes ScalarDistribution
}

// TrustReport 是工程信任/验证独立性分布(§38.5;§36.3「可产出」项)。
type TrustReport struct {
	// QualificationStrength 是 qualification strength 分布(assertion_strong /
	// compile_error_weak / panic_error_weak / error_weak / baseline_pass / n_a)。
	QualificationStrength CategoryDistribution
	// AuthoringMode 是 authoring mode 分布(pre_commit / post_hoc)。
	AuthoringMode CategoryDistribution
	// Visibility 是 visibility 分布(visible / held_out)。
	Visibility CategoryDistribution
	// StrengthByVisibility 是 strength × visibility 交叉分布(§38.5)。
	StrengthByVisibility CrossDistribution
	// Independence 是 acceptance 级 planned/effective independence coverage
	// 与 source/information 缺口(Decision Lock 10)。
	Independence IndependenceReport
}

// IndependenceReport 是 acceptance 级独立性覆盖读数。
type IndependenceReport struct {
	// Planned 是 independence_required 分布(none/source/information;§38.5
	// planned coverage 的记录面)。
	Planned CategoryDistribution
	// Source / Information 是有该级要求的 acceptance 的 effective coverage
	// 与缺口(none 级不列:none 不要求独立证据,「覆盖」没有对照意义)。
	Source      IndependenceCoverage
	Information IndependenceCoverage
}

// IndependenceCoverage 是一个要求级别的 effective coverage 读数(§38.5)。
// 缺口维度归因遵循 §8.5(同一 evidence 必须同时满足 source+information):
// source 级缺口全记 SourceGaps;information 级缺口在存在 source 满足而整体
// 不成立时记 InformationGaps,否则记 SourceGaps。
type IndependenceCoverage struct {
	Required        int // 有该级要求的 acceptance 观测数
	Covered         int // 其中 effective coverage 成立数
	SourceGaps      int
	InformationGaps int
	Samples         Samples // Required 的样本充分性
}

// FirstPassReport 是首次最终验证的 first-pass 分布(§38.5 first-pass final
// validation),按被验证契约是否含 held-out binding 分桶。两桶都出现
// (未观测桶为零值、Sufficient=false;「数据存在才报告」由 Sufficient
// 标注承载,不靠省略桶,AC4)。
type FirstPassReport struct {
	VisibleOnly FirstPassBucket // 契约无 held-out binding
	WithHoldout FirstPassBucket // 契约含 held-out binding
}

// FirstPassBucket 是一个 first-pass 分桶的读数。
type FirstPassBucket struct {
	Tasks   int     // 有 first-pass 观测的任务数
	Passed  int     // 首次最终验证 verified 的任务数
	Samples Samples // Tasks 的样本充分性
}

// BuildBaselineReport 从已落成的度量事实投影组装 P1 基线报告(§25/§38/§36.3)。
//
// 输入校验是硬边界:任务 id 必填且唯一;effort/usage/first-pass 的任务归属
// 必须与所在任务一致;枚举取值必须在封闭词表内;阶段区间与里程碑必须自洽。
// 映射错误(而非数据缺失)一律报错 —— 报告丢行或静默归类比失败更危险。
// 数据缺失不是错误:缺失维度以零值 + Sufficient=false 呈现(AC4)。
func BuildBaselineReport(tasks []TaskBaseline, generatedAt time.Time) (BaselineReport, error) {
	if generatedAt.IsZero() {
		return BaselineReport{}, fmt.Errorf("measurement: 基线报告缺少生成时刻")
	}
	rep := BaselineReport{
		GeneratedAt: generatedAt.UTC(),
		Tasks:       len(tasks),
		Human:       HumanReport{ByKind: make(map[HumanEffortKind]KindEffort, len(effortKinds))},
		Cost:        CostReport{ByPlane: make(map[CostPlane]PlaneTotals, 2)},
		Phases:      PhaseReport{WallByPhase: make(map[LifecyclePhase]PhaseTotals, len(lifecyclePhases))},
	}
	rep.Trust.StrengthByVisibility.Counts = make(map[string]map[string]int)
	// 词表全量初始化:未观测取值显式 0 计数,不靠省略表达缺失(AC4)。
	for _, k := range effortKinds {
		rep.Human.ByKind[k] = KindEffort{}
	}
	for _, p := range []CostPlane{CostPlaneControl, CostPlaneExecution} {
		rep.Cost.ByPlane[p] = PlaneTotals{}
	}
	for _, ph := range lifecyclePhases {
		rep.Phases.WallByPhase[ph] = PhaseTotals{}
	}

	seen := make(map[string]bool, len(tasks))
	var activeMinutes, costMicros, latencies []float64
	var strengths, modes, visibilities, planned []string
	// 覆盖期间从输入事实的观测时刻推导(AC3);没有任何带时刻的事实时保持
	// SourcePeriodValid=false:期间未知必须显式呈现,不用 GeneratedAt 伪充(AC2)。
	var periodFrom, periodTo time.Time
	observe := func(at time.Time) {
		if at.IsZero() {
			return
		}
		if !rep.SourcePeriodValid {
			periodFrom, periodTo, rep.SourcePeriodValid = at, at, true
			return
		}
		if at.Before(periodFrom) {
			periodFrom = at
		}
		if at.After(periodTo) {
			periodTo = at
		}
	}

	for i := range tasks {
		t := &tasks[i]
		if strings.TrimSpace(t.TaskID) == "" {
			return BaselineReport{}, fmt.Errorf("measurement: 基线报告输入 tasks[%d] 缺少 task_id", i)
		}
		if seen[t.TaskID] {
			return BaselineReport{}, fmt.Errorf("measurement: 基线报告输入任务重复: %s", t.TaskID)
		}
		seen[t.TaskID] = true

		// ---- 人工控制开销(§25/§38.2) ----
		taskActive := time.Duration(0)
		for _, h := range t.Efforts {
			if err := h.Validate(); err != nil {
				return BaselineReport{}, fmt.Errorf("measurement: 任务 %s 的 effort: %w", t.TaskID, err)
			}
			if h.TaskID != t.TaskID {
				return BaselineReport{}, fmt.Errorf("measurement: effort 行归属任务 %s,却出现在任务 %s 的投影里", h.TaskID, t.TaskID)
			}
			observe(h.RecordedAt)
			k := rep.Human.ByKind[h.Kind]
			k.Active += h.Active
			k.Wall += h.Wall
			k.Samples.Count++
			rep.Human.ByKind[h.Kind] = k
			taskActive += h.Active
		}
		if len(t.Efforts) > 0 {
			activeMinutes = append(activeMinutes, Minutes(taskActive))
		}

		// ---- LLM/Agent 使用与成本(§25.3/§30/§38.4) ----
		var taskCost int64
		for _, u := range t.Usages {
			if err := u.Validate(); err != nil {
				return BaselineReport{}, fmt.Errorf("measurement: 任务 %s 的 usage: %w", t.TaskID, err)
			}
			if u.TaskID != t.TaskID {
				return BaselineReport{}, fmt.Errorf("measurement: usage 行归属任务 %s,却出现在任务 %s 的投影里", u.TaskID, t.TaskID)
			}
			observe(u.RecordedAt)
			p := rep.Cost.ByPlane[u.CostPlane]
			p.CostMicros += u.CostMicros
			p.InputTokens += u.InputTokens
			p.OutputTokens += u.OutputTokens
			p.Calls++
			p.Samples.Count++
			rep.Cost.ByPlane[u.CostPlane] = p
			taskCost += u.CostMicros
		}
		if len(t.Usages) > 0 {
			costMicros = append(costMicros, float64(taskCost))
		}

		// ---- 生命周期阶段墙钟(§25 阶段分解) ----
		for _, s := range t.Spans {
			if err := s.Validate(); err != nil {
				return BaselineReport{}, fmt.Errorf("measurement: 任务 %s 的 phase span: %w", t.TaskID, err)
			}
			observe(s.Start)
			observe(s.End)
			p := rep.Phases.WallByPhase[s.Phase]
			p.Wall += s.Duration()
			p.Samples.Count++
			rep.Phases.WallByPhase[s.Phase] = p
		}

		// ---- 里程碑跨度(§25.1 guardrail) ----
		for m, at := range t.Milestones {
			if !m.IsValid() {
				return BaselineReport{}, fmt.Errorf("measurement: 任务 %s 的里程碑 %q 非法", t.TaskID, string(m))
			}
			if at.IsZero() {
				return BaselineReport{}, fmt.Errorf("measurement: 任务 %s 的里程碑 %s 缺少时刻", t.TaskID, string(m))
			}
			observe(at)
		}
		if d, ok := IdeaToVerifiedLatency(t.Milestones); ok {
			latencies = append(latencies, Minutes(d))
		}

		// ---- §38.5 binding 级分布 ----
		for _, b := range t.Bindings {
			if strings.TrimSpace(b.BindingID) == "" {
				return BaselineReport{}, fmt.Errorf("measurement: 任务 %s 的 binding 投影缺少 binding_id", t.TaskID)
			}
			if !b.AuthoringMode.IsValid() {
				return BaselineReport{}, fmt.Errorf("measurement: binding %s 的 authoring mode %q 非法", b.BindingID, string(b.AuthoringMode))
			}
			if !b.Visibility.IsValid() {
				return BaselineReport{}, fmt.Errorf("measurement: binding %s 的 visibility %q 非法", b.BindingID, string(b.Visibility))
			}
			if !b.Strength.IsValid() {
				return BaselineReport{}, fmt.Errorf("measurement: binding %s 的 qualification strength %q 非法", b.BindingID, string(b.Strength))
			}
			strengths = append(strengths, string(b.Strength))
			modes = append(modes, string(b.AuthoringMode))
			visibilities = append(visibilities, string(b.Visibility))
			inner := rep.Trust.StrengthByVisibility.Counts[string(b.Strength)]
			if inner == nil {
				inner = make(map[string]int, len(visibilityVocab))
				rep.Trust.StrengthByVisibility.Counts[string(b.Strength)] = inner
			}
			inner[string(b.Visibility)]++
		}

		// ---- §38.5 acceptance 级 planned/effective coverage 与缺口 ----
		for _, a := range t.Acceptances {
			if strings.TrimSpace(a.AcceptanceID) == "" {
				return BaselineReport{}, fmt.Errorf("measurement: 任务 %s 的 acceptance 投影缺少 acceptance_id", t.TaskID)
			}
			req := a.Required
			if req == "" {
				req = LevelNone // §7.3 缺省:未声明 = none
			}
			if !req.IsValid() {
				return BaselineReport{}, fmt.Errorf("measurement: acceptance %s 的 independence level %q 非法", a.AcceptanceID, string(a.Required))
			}
			planned = append(planned, string(req))
			switch req {
			case LevelSource:
				c := &rep.Trust.Independence.Source
				c.Required++
				if a.Covered {
					c.Covered++
				} else {
					c.SourceGaps++
				}
			case LevelInformation:
				c := &rep.Trust.Independence.Information
				c.Required++
				if a.Covered {
					c.Covered++
				} else if a.SourceSatisfied {
					// source 维度已有满足的 evidence,缺的是同一条 evidence 上的
					// information 维度(§8.5:两维度必须在同一 evidence 上成立)。
					c.InformationGaps++
				} else {
					c.SourceGaps++
				}
			}
		}

		// ---- §38.5 first-pass(AC3) ----
		if fp := t.FirstPass; fp != nil {
			if fp.TaskID != t.TaskID {
				return BaselineReport{}, fmt.Errorf("measurement: first-pass 观测归属任务 %s,却出现在任务 %s 的投影里", fp.TaskID, t.TaskID)
			}
			bucket := &rep.FirstPass.VisibleOnly
			if fp.WithHoldout {
				bucket = &rep.FirstPass.WithHoldout
			}
			bucket.Tasks++
			if fp.Passed {
				bucket.Passed++
			}
		}
	}

	// ---- 样本充分性标注(AC4)与分布组装 ----
	for kind, k := range rep.Human.ByKind {
		k.Samples.Sufficient = k.Samples.Count >= MinSufficientSamples
		rep.Human.ByKind[kind] = k
	}
	for plane, p := range rep.Cost.ByPlane {
		p.Samples.Sufficient = p.Samples.Count >= MinSufficientSamples
		rep.Cost.ByPlane[plane] = p
	}
	for phase, p := range rep.Phases.WallByPhase {
		p.Samples.Sufficient = p.Samples.Count >= MinSufficientSamples
		rep.Phases.WallByPhase[phase] = p
	}
	rep.Human.PerTaskActiveMinutes = newScalarDistribution(activeMinutes)
	rep.Cost.PerTaskCostMicros = newScalarDistribution(costMicros)

	control := rep.Cost.ByPlane[CostPlaneControl].CostMicros
	execution := rep.Cost.ByPlane[CostPlaneExecution].CostMicros
	if total := control + execution; total > 0 {
		rep.Cost.ControlRatio = float64(control) / float64(total)
		rep.Cost.RatioValid = true
	}

	rep.Latency.IdeaToVerifiedMinutes = newScalarDistribution(latencies)

	var err error
	if rep.Trust.QualificationStrength, err = newCategoryDistribution(stringVocab(strengthVocab), strengths); err != nil {
		return BaselineReport{}, err
	}
	if rep.Trust.AuthoringMode, err = newCategoryDistribution(stringVocab(authoringModeVocab), modes); err != nil {
		return BaselineReport{}, err
	}
	if rep.Trust.Visibility, err = newCategoryDistribution(stringVocab(visibilityVocab), visibilities); err != nil {
		return BaselineReport{}, err
	}
	rep.Trust.StrengthByVisibility.Total = len(strengths)
	rep.Trust.StrengthByVisibility.Sufficient = len(strengths) >= MinSufficientSamples

	if rep.Trust.Independence.Planned, err = newCategoryDistribution(stringVocab(independenceVocab), planned); err != nil {
		return BaselineReport{}, err
	}
	rep.Trust.Independence.Source.Samples = sampleSufficiency(rep.Trust.Independence.Source.Required)
	rep.Trust.Independence.Information.Samples = sampleSufficiency(rep.Trust.Independence.Information.Required)

	rep.FirstPass.VisibleOnly.Samples = sampleSufficiency(rep.FirstPass.VisibleOnly.Tasks)
	rep.FirstPass.WithHoldout.Samples = sampleSufficiency(rep.FirstPass.WithHoldout.Tasks)

	rep.SourcePeriodFrom, rep.SourcePeriodTo = periodFrom.UTC(), periodTo.UTC()
	return rep, nil
}

// ---- 运维呈现(P0-08:基线报告的操作员只读呈现) ----

// RenderBaselineReport 把基线报告渲染成确定性的运维文本(P0-08 AC1-AC3)。
//
// 契约:
//   - 纯函数:不读时钟、不依赖环境;同一报告两次渲染逐字节相同(可 diff、可存档)。
//   - 全部迭代走固定词表顺序(effortKinds/lifecyclePhases/报告词表),不迭代
//     map —— 渲染行序是稳定契约。
//   - 每个读数行都带样本数与充分性标注(AC3):count=0 必须伴随 insufficient
//     呈现,不得被误读为「观测到 0」或「通过」(AC2);比率未定义、标量分布
//     无样本时输出显式文字,不输出数值。
//   - 本函数不解释、不评价任何数值:P1 只记录分布,不设阈值(决策锁 16)。
func RenderBaselineReport(rep BaselineReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "基线报告(P1 度量基线,只读)\n")
	fmt.Fprintf(&b, "generated_at: %s\n", rep.GeneratedAt.Format(time.RFC3339))
	fmt.Fprintf(&b, "tasks: %d\n", rep.Tasks)
	if rep.SourcePeriodValid {
		fmt.Fprintf(&b, "source_period: %s .. %s\n",
			rep.SourcePeriodFrom.Format(time.RFC3339), rep.SourcePeriodTo.Format(time.RFC3339))
	} else {
		fmt.Fprintf(&b, "source_period: 未知(输入不含任何带观测时刻的事实,AC2:不伪造期间)\n")
	}
	fmt.Fprintf(&b, "min_sufficient_samples: %d(呈现充分性规则:样本不足 ≠ 观测到 0,也 ≠ 通过)\n\n", MinSufficientSamples)

	// ---- 人工控制开销(§25/§38.2) ----
	b.WriteString("[human] 人工控制开销(§25/§38.2;active=人实际投入,wall=端到端等待)\n")
	for _, k := range effortKinds {
		ke := rep.Human.ByKind[k]
		fmt.Fprintf(&b, "  %s: active=%s wall=%s samples=%d %s\n",
			string(k), minutesMark(ke.Active), minutesMark(ke.Wall), ke.Samples.Count, sufficiencyMark(ke.Samples))
	}
	b.WriteString("  per_task_active_minutes: " + scalarMark(rep.Human.PerTaskActiveMinutes) + "\n\n")

	// ---- LLM/Agent 使用与成本(§25.3/§38.4) ----
	b.WriteString("[cost] LLM/Agent 使用与成本(§25.3/§38.4;按写入时落成的 cost_plane 聚合,历史不重算)\n")
	for _, p := range []CostPlane{CostPlaneControl, CostPlaneExecution} {
		pt := rep.Cost.ByPlane[p]
		fmt.Fprintf(&b, "  %s: cost=%dmicros in=%d out=%d calls=%d samples=%d %s\n",
			string(p), pt.CostMicros, pt.InputTokens, pt.OutputTokens, pt.Calls, pt.Samples.Count, sufficiencyMark(pt.Samples))
	}
	if rep.Cost.RatioValid {
		fmt.Fprintf(&b, "  control_plane_llm_cost_ratio: %.4f\n", rep.Cost.ControlRatio)
	} else {
		b.WriteString("  control_plane_llm_cost_ratio: 未定义(总成本为 0;不猜,AC2)\n")
	}
	b.WriteString("  per_task_cost_micros: " + scalarMark(rep.Cost.PerTaskCostMicros) + "\n\n")

	// ---- 生命周期阶段墙钟(§25) ----
	b.WriteString("[phases] 生命周期阶段墙钟(§25)\n")
	for _, ph := range lifecyclePhases {
		pt := rep.Phases.WallByPhase[ph]
		fmt.Fprintf(&b, "  %s: wall=%s samples=%d %s\n",
			string(ph), minutesMark(pt.Wall), pt.Samples.Count, sufficiencyMark(pt.Samples))
	}
	b.WriteString("\n")

	// ---- 里程碑跨度(§25.1) ----
	b.WriteString("[latency] 里程碑跨度(§25.1 guardrail)\n")
	b.WriteString("  idea_to_verified_minutes: " + scalarMark(rep.Latency.IdeaToVerifiedMinutes) + "\n\n")

	// ---- 验证独立性分布(§38.5) ----
	b.WriteString("[trust] 验证独立性分布(§38.5)\n")
	b.WriteString("  qualification_strength: " + categoryMark(rep.Trust.QualificationStrength, stringVocab(strengthVocab)) + "\n")
	b.WriteString("  authoring_mode: " + categoryMark(rep.Trust.AuthoringMode, stringVocab(authoringModeVocab)) + "\n")
	b.WriteString("  visibility: " + categoryMark(rep.Trust.Visibility, stringVocab(visibilityVocab)) + "\n")
	b.WriteString("  strength_x_visibility: " + crossMark(rep.Trust.StrengthByVisibility) + "\n")
	b.WriteString("  independence_planned: " + categoryMark(rep.Trust.Independence.Planned, stringVocab(independenceVocab)) + "\n")
	fmt.Fprintf(&b, "  independence_source: required=%d covered=%d source_gaps=%d information_gaps=%d %s\n",
		rep.Trust.Independence.Source.Required, rep.Trust.Independence.Source.Covered,
		rep.Trust.Independence.Source.SourceGaps, rep.Trust.Independence.Source.InformationGaps,
		sufficiencyMark(rep.Trust.Independence.Source.Samples))
	fmt.Fprintf(&b, "  independence_information: required=%d covered=%d source_gaps=%d information_gaps=%d %s\n\n",
		rep.Trust.Independence.Information.Required, rep.Trust.Independence.Information.Covered,
		rep.Trust.Independence.Information.SourceGaps, rep.Trust.Independence.Information.InformationGaps,
		sufficiencyMark(rep.Trust.Independence.Information.Samples))

	// ---- 首次最终验证(§38.5) ----
	b.WriteString("[first_pass] 首次最终验证(§38.5;按契约是否含 held-out binding 分桶)\n")
	fmt.Fprintf(&b, "  visible_only: tasks=%d passed=%d %s\n",
		rep.FirstPass.VisibleOnly.Tasks, rep.FirstPass.VisibleOnly.Passed, sufficiencyMark(rep.FirstPass.VisibleOnly.Samples))
	fmt.Fprintf(&b, "  with_holdout: tasks=%d passed=%d %s\n",
		rep.FirstPass.WithHoldout.Tasks, rep.FirstPass.WithHoldout.Passed, sufficiencyMark(rep.FirstPass.WithHoldout.Samples))
	return b.String()
}

// sufficiencyMark 把样本充分性渲染为显式标注:不足必须可见(AC2)。
func sufficiencyMark(s Samples) string {
	if s.Sufficient {
		return "sufficient"
	}
	return "insufficient"
}

// boolMark 是无样本数、只有充分性布尔的分布(分类/交叉分布)的标注。
func boolMark(sufficient bool) string {
	if sufficient {
		return "sufficient"
	}
	return "insufficient"
}

// minutesMark 按度量口径(分钟,一位小数)渲染时长。
func minutesMark(d time.Duration) string {
	return fmt.Sprintf("%.1fm", Minutes(d))
}

// scalarMark 渲染标量分布:无样本时输出显式文字,不输出 min/median/max 数值
// (没有样本就没有分布,AC2)。
func scalarMark(d ScalarDistribution) string {
	if d.Count == 0 {
		return fmt.Sprintf("n=0 %s(无样本)", sufficiencyMark(d.Samples))
	}
	return fmt.Sprintf("n=%d min=%.1f median=%.1f max=%.1f %s",
		d.Count, d.Min, d.Median, d.Max, sufficiencyMark(d.Samples))
}

// categoryMark 渲染封闭词表计数分布:词表全量取值按固定顺序出现(未观测为
// 显式 0 计数),整体充分性随行标注。
func categoryMark(d CategoryDistribution, vocab []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "total=%d %s |", d.Total, boolMark(d.Sufficient))
	for _, v := range vocab {
		fmt.Fprintf(&b, " %s=%d", v, d.Counts[v])
	}
	return b.String()
}

// crossMark 渲染 strength×visibility 交叉分布:只列已观测组合(按词表固定
// 顺序),无观测时显式说明;边际分布的显式 0 由上方单维分布承载。
func crossMark(d CrossDistribution) string {
	if d.Total == 0 {
		return fmt.Sprintf("total=0 %s(无已观测组合)", boolMark(d.Sufficient))
	}
	var b strings.Builder
	fmt.Fprintf(&b, "total=%d %s |", d.Total, boolMark(d.Sufficient))
	for _, s := range strengthVocab {
		for _, v := range visibilityVocab {
			if n := d.Counts[string(s)][string(v)]; n > 0 {
				fmt.Fprintf(&b, " %s/%s=%d", string(s), string(v), n)
			}
		}
	}
	return b.String()
}
