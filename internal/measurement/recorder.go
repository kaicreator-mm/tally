package measurement

// recorder.go(P1-41)是 P1 度量事实的自动录制入口:执行组件在阶段切换、
// 人工控制完成、LLM/runtime 调用发生时调用 Recorder,由它统一落成
// PRD §25/§30/§38 的事实面。
//
// 边界(P1):
//   - 人工计时(active/wall)与 LLM 使用/成本事实即时经 P1-34 的 *Store
//     落库(llm_usage / human_effort,只追加,INV-FACT-01);
//   - 生命周期阶段 start/end(PhaseSpan)是录制器按任务持有的事实记录:
//     P1 存储合同(migrations/0021)只有 llm_usage / human_effort 两张度量
//     事实表,阶段计时没有落库面,本文件不越权加表;区间按 P1-08 事实模型
//     累积,经 PhaseTiming 交付(AC1),各阶段墙钟耗时由 DurationByPhase 汇总;
//   - runtime 自报用量是 claim(INV-AUTH-01):provider 未上报用量时不存在可
//     落成的 token 数值,Recorder 拒绝伪造「观测到 0」的数值(P0-06/锁 16:
//     未知用量仍是一次调用,0022 起照落 usage_reported=false + NULL 数值的
//     事实行),不可用性表示在 RuntimeUsageTotals.UsageAvailable 上;
//
// 归属口径(AC2/AC3):cost_plane 由调用方在调用发生时给出并随之落成事实
// 字段,本文件不做任何 role→plane 推导;全部指标(阶段耗时分解、*_minutes、
// plane 成本比率)只从录制的事实推导,任务上没有任何手工记账字段参与。

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// errNoStore 表示 Recorder 未绑定事实存储却收到需要落库的记录。纯事实路径
// (阶段计时)不需要存储;用量事实路径(含未上报用量的调用,P0-06 起必须
// 落行)与人工计时路径必须显式绑定存储。
var errNoStore = errors.New("measurement: recorder 未绑定事实存储,无法落库")

// Recorder 自动录制 P1 度量事实(AC1-AC4)。一个实例服务一个进程内的录制面:
//   - 持久事实(人工计时、LLM 使用/成本)即时经 *Store 落库;
//   - 阶段区间与 runtime 用量聚合按任务在内存累积,经 PhaseTiming /
//     RuntimeUsageForTask 交付。
//
// 全部方法并发安全;内部互斥锁只护内存状态,不跨存储调用持有。
// store 允许为 nil(纯事实路径可用);需要落库的路径遇到 nil 返回 errNoStore。
type Recorder struct {
	store *Store

	mu    sync.Mutex
	open  map[phaseKey]time.Time  // 已开启未关闭的阶段计时区间
	spans map[string][]PhaseSpan  // 已落成的阶段区间事实(按任务)
	usage map[string]*usageTotals // runtime 用量聚合(按任务)
}

// phaseKey 定位一个任务某阶段的开启中区间。
type phaseKey struct {
	taskID string
	phase  LifecyclePhase
}

// NewRecorder 构造录制器。store 为 nil 时只支持纯事实路径(见 errNoStore)。
func NewRecorder(store *Store) *Recorder {
	return &Recorder{
		store: store,
		open:  make(map[phaseKey]time.Time),
		spans: make(map[string][]PhaseSpan),
		usage: make(map[string]*usageTotals),
	}
}

// ---- 生命周期阶段计时(AC1:phase start/end/wall) ----

// PhaseStarted 开启某任务某阶段的一段计时区间(start)。同一阶段允许先后
// 多段(重试/分段计时),但同一 (task, phase) 不允许并发开启:重复开启说明
// 调用方状态机有缺陷,报错而不是静默覆盖。
func (r *Recorder) PhaseStarted(taskID string, phase LifecyclePhase, at time.Time) error {
	if strings.TrimSpace(taskID) == "" {
		return fmt.Errorf("measurement: phase start 缺少 task_id")
	}
	if !phase.IsValid() {
		return fmt.Errorf("measurement: phase start 的阶段名 %q 非法,必须是 PRD §25 定义的阶段名", string(phase))
	}
	if at.IsZero() {
		return fmt.Errorf("measurement: phase start 缺少时刻")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := phaseKey{taskID: taskID, phase: phase}
	if _, open := r.open[key]; open {
		return fmt.Errorf("measurement: 任务 %s 的阶段 %s 已在计时中,不能重复开启", taskID, string(phase))
	}
	r.open[key] = at
	return nil
}

// PhaseEnded 关闭开启中的区间并落成一条 PhaseSpan 事实(end;End-Start 即
// 该段墙钟时长,AC1)。返回已落成的区间事实。
func (r *Recorder) PhaseEnded(taskID string, phase LifecyclePhase, at time.Time) (PhaseSpan, error) {
	if strings.TrimSpace(taskID) == "" {
		return PhaseSpan{}, fmt.Errorf("measurement: phase end 缺少 task_id")
	}
	if !phase.IsValid() {
		return PhaseSpan{}, fmt.Errorf("measurement: phase end 的阶段名 %q 非法,必须是 PRD §25 定义的阶段名", string(phase))
	}
	if at.IsZero() {
		return PhaseSpan{}, fmt.Errorf("measurement: phase end 缺少时刻")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := phaseKey{taskID: taskID, phase: phase}
	start, open := r.open[key]
	if !open {
		return PhaseSpan{}, fmt.Errorf("measurement: 任务 %s 的阶段 %s 没有开启中的计时区间", taskID, string(phase))
	}
	if at.Before(start) {
		return PhaseSpan{}, fmt.Errorf("measurement: 任务 %s 的阶段 %s 结束时刻早于开启时刻", taskID, string(phase))
	}
	delete(r.open, key)
	span := PhaseSpan{Phase: phase, Start: start, End: at}
	r.spans[taskID] = append(r.spans[taskID], span)
	return span, nil
}

// PhaseTiming 返回该任务已落成的阶段区间事实集合(副本;AC1)。ok=false 表示
// 尚无区间。各阶段墙钟耗时分解用 PhaseTiming.DurationByPhase,minutes 口径用
// measurement.Minutes 换算。
func (r *Recorder) PhaseTiming(taskID string) (PhaseTiming, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	spans := r.spans[taskID]
	if len(spans) == 0 {
		return PhaseTiming{}, false
	}
	out := make([]PhaseSpan, len(spans))
	copy(out, spans)
	return PhaseTiming{TaskID: taskID, Spans: out}, true
}

// ---- 人工控制开销(AC1:active/wall)与 LLM 使用/成本(AC2)落库 ----

// RecordEffort 落一条人工控制开销事实(active/wall 双口径,AC1;§25 至少
// 记录条目)。即时经 P1-34 适配器落库,只追加;*_minutes 指标由读侧按事实行
// 聚合换算(AC3:任务上没有、也不需要任何手工记账字段)。
func (r *Recorder) RecordEffort(ctx context.Context, h HumanEffort) (int64, error) {
	if r.store == nil {
		return 0, errNoStore
	}
	return r.store.RecordHumanEffort(ctx, h)
}

// RecordLLMUsage 落一条 LLM/Agent 调用使用/成本事实(AC2:cost_plane 由调用
// 方在调用发生时给出并落成事实字段;§30)。原样委托 P1-34 适配器:非法归属
// 面/缺 use_case 等在写入前被拒,不触库。
func (r *Recorder) RecordLLMUsage(ctx context.Context, u LLMUsage) (int64, error) {
	if r.store == nil {
		return 0, errNoStore
	}
	return r.store.RecordLLMUsage(ctx, u)
}

// ---- runtime 用量观测(AC4:不可用 ≠ 0) ----

// TokenUsage 是一次调用 provider token 用量的观测结果(AC4):
//
//   - Reported=true:provider 上报了用量,InputTokens/OutputTokens 是观测值
//     (观测到 0 是合法事实);
//   - Reported=false:provider 未上报用量 —— 这是「不可用」,与「观测到 0」
//     是两种不同状态,不得互化;此时不得携带数值(未观测就没有值)。
type TokenUsage struct {
	Reported     bool
	InputTokens  int64
	OutputTokens int64
}

// ReportedTokens 构造一条已上报的用量观测。
func ReportedTokens(input, output int64) TokenUsage {
	return TokenUsage{Reported: true, InputTokens: input, OutputTokens: output}
}

// UnavailableTokens 构造一条「provider 未上报用量」的观测(AC4)。
func UnavailableTokens() TokenUsage { return TokenUsage{} }

// Validate 校验观测自洽:数值非负;未上报时不得携带数值(把「未知」伪装成
// 「已知 0」的邻近值会污染全部 token 指标,直接拒绝)。
func (t TokenUsage) Validate() error {
	if t.InputTokens < 0 || t.OutputTokens < 0 {
		return fmt.Errorf("measurement: token 用量为负")
	}
	if !t.Reported && (t.InputTokens != 0 || t.OutputTokens != 0) {
		return fmt.Errorf("measurement: token 用量未上报时不得携带数值(AC4:不可用不是 0)")
	}
	return nil
}

// RuntimeUsage 是一次 Agent runtime 调用的用量观测(调用发生时录制;
// §30/§38.4)。Tokens 未上报时调用本身照样落成 llm_usage 事实行(P0-06/锁 16:
// usage_reported=false,token/cost 三列 NULL —— 未观测不写成观测到 0),
// 并计入 RuntimeUsageTotals.Calls 与 CallsMissingUsage。
type RuntimeUsage struct {
	// TaskID 归属任务(runtime 调用都发生在任务执行内,必填)。
	TaskID string
	// UseCase 是调用用途(如 implement_patch);CostPlane 是成本归属面,调用
	// 发生时必填(AC2),本包不做任何 role→plane 推导。
	UseCase   string
	CostPlane CostPlane

	// 调用面标识(§30 provider/runtime/model);探测面拿不到就留空,不猜。
	Provider string
	Runtime  string
	Model    string

	// BehaviorRevision/ContextHash 原样记录(§30 behavior revision 与
	// context hash 的 runtime 侧字段;prompt 侧字段走 RecordLLMUsage 全量面)。
	BehaviorRevision string
	ContextHash      string

	// Latency 是调用墙钟耗时;失败/取消调用照样按实际发生值记录。
	Latency time.Duration
	// Status 是调用终态;FailureType 仅失败时给出类型描述。
	Status      UsageStatus
	FailureType string

	// Tokens 是 provider 用量观测(AC4:未上报用 UnavailableTokens() 表示)。
	Tokens TokenUsage
	// CostMicros 是已确立的调用成本(记账货币 1e-6 整数);仅在 Tokens 上报时
	// 随行落库(没有 token 观测就没有可落成的成本数值;未上报却携带成本
	// 会在写入前被拒,不静默丢弃也不伪装成 0)。
	CostMicros int64

	// RecordedAt 是观测发生时刻。
	RecordedAt time.Time
}

// Validate 校验观测:任务归属、用途、归属面(AC2)、终态、非负、时刻与
// Tokens 自洽(AC4)。
func (o RuntimeUsage) Validate() error {
	if strings.TrimSpace(o.TaskID) == "" {
		return fmt.Errorf("measurement: runtime usage 缺少 task_id")
	}
	if strings.TrimSpace(o.UseCase) == "" {
		return fmt.Errorf("measurement: runtime usage 缺少 use_case")
	}
	if !o.CostPlane.IsValid() {
		return fmt.Errorf("measurement: runtime usage 的 cost_plane %q 非法,必须是 control|execution(调用时必填,PRD §25.3/§30)", string(o.CostPlane))
	}
	if !o.Status.IsValid() {
		return fmt.Errorf("measurement: runtime usage 的 status %q 非法,必须是 succeeded|failed|cancelled 之一", string(o.Status))
	}
	if o.Latency < 0 {
		return fmt.Errorf("measurement: runtime usage 的 latency 为负")
	}
	if o.CostMicros < 0 {
		return fmt.Errorf("measurement: runtime usage 的 cost 为负")
	}
	if o.RecordedAt.IsZero() {
		return fmt.Errorf("measurement: runtime usage 缺少 recorded_at(度量事实必须可定位到发生时间)")
	}
	if !o.Tokens.Reported && o.CostMicros != 0 {
		return fmt.Errorf("measurement: runtime usage 用量未上报时不得携带成本(没有 token 观测就没有可落成的成本数值)")
	}
	if err := o.Tokens.Validate(); err != nil {
		return fmt.Errorf("measurement: runtime usage 的 tokens: %w", err)
	}
	return nil
}

// fact 把观测转成可落库的 LLMUsage 事实行。Tokens 上报时携带观测数值;
// 未上报时 UsageReported=false、token/cost 留零(库内落 NULL)—— 调用事实
// 照落,数值保持未知(锁 16:未知不是 0)。
func (o RuntimeUsage) fact() LLMUsage {
	u := LLMUsage{
		TaskID:           o.TaskID,
		UseCase:          o.UseCase,
		CostPlane:        o.CostPlane,
		Provider:         o.Provider,
		Runtime:          o.Runtime,
		Model:            o.Model,
		BehaviorRevision: o.BehaviorRevision,
		ContextHash:      o.ContextHash,
		Latency:          o.Latency,
		Status:           o.Status,
		FailureType:      o.FailureType,
		UsageReported:    o.Tokens.Reported,
		RecordedAt:       o.RecordedAt,
	}
	if o.Tokens.Reported {
		u.InputTokens = o.Tokens.InputTokens
		u.OutputTokens = o.Tokens.OutputTokens
		u.CostMicros = o.CostMicros
	}
	return u
}

// usageTotals 是某任务 runtime 用量聚合的内部累积。
type usageTotals RuntimeUsageTotals

// RuntimeUsageTotals 是某任务 runtime 调用用量的聚合视图(AC4)。
type RuntimeUsageTotals struct {
	// Calls 是观测到的调用数(calls_total,含未上报用量的调用)。
	Calls int64
	// CallsMissingUsage 是其中用量未上报的调用数;Calls-CallsMissingUsage 即
	// 已上报调用数(P0-06 AC1:total/reported/missing 可重建)。
	CallsMissingUsage int64
	// InputTokens/OutputTokens 是已上报用量的合计;CostMicros 是已落成事实的
	// 成本合计。UsageAvailable=false 时这些合计不完整,消费方必须按「不可用」
	// 呈现,不得当作完整事实。
	InputTokens  int64
	OutputTokens int64
	CostMicros   int64
	// UsageAvailable 报告是否所有调用都上报了用量(P0-06 AC2:由
	// CallsMissingUsage 单调推导,不由最后一次写入者决定 —— 一旦存在未上报
	// 调用,后续已上报调用不会把它翻回 true);true 时上面的合计是完整的
	// 观测事实,false 时存在未上报用量(不可用 ≠ 0)。
	UsageAvailable bool
}

// RecordRuntimeUsage 录制一次 runtime 调用的用量观测(AC2/AC4/P0-06 锁 16):
//
//   - 无论 Tokens 是否上报,都立即落一条 llm_usage 事实行(cost_plane 在调用
//     时落成)并计入聚合,返回行 id;
//   - Tokens 上报:行携带观测数值(观测到 0 合法);
//   - Tokens 未上报:行以 usage_reported=false 落库,token/cost 为 NULL
//     (领域侧零值)—— 调用事实保留,数值保持未知,不落假 0;
//   - 落库失败(含未绑定存储)返回错误,聚合不变(事实未落成,不计入)。
func (r *Recorder) RecordRuntimeUsage(ctx context.Context, o RuntimeUsage) (int64, error) {
	if err := o.Validate(); err != nil {
		return 0, err
	}
	if r.store == nil {
		return 0, errNoStore
	}
	id, err := r.store.RecordLLMUsage(ctx, o.fact())
	if err != nil {
		return 0, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	totals, ok := r.usage[o.TaskID]
	if !ok {
		totals = &usageTotals{}
		r.usage[o.TaskID] = totals
	}
	totals.Calls++
	if o.Tokens.Reported {
		totals.InputTokens += o.Tokens.InputTokens
		totals.OutputTokens += o.Tokens.OutputTokens
		totals.CostMicros += o.CostMicros
	} else {
		totals.CallsMissingUsage++
	}
	// AC2:可用性由缺失计数单调推导 —— 出现过未上报调用即恒为 false。
	totals.UsageAvailable = totals.CallsMissingUsage == 0
	return id, nil
}

// RuntimeUsageForTask 返回该任务的 runtime 用量聚合(AC4)。ok=false 表示
// 尚无观测。
func (r *Recorder) RuntimeUsageForTask(taskID string) (RuntimeUsageTotals, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	totals, ok := r.usage[taskID]
	if !ok {
		return RuntimeUsageTotals{}, false
	}
	return RuntimeUsageTotals(*totals), true
}
