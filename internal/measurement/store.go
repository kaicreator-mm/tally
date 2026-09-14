package measurement

// 度量事实持久化适配器(P1-34,migrations/0021 llm_usage / human_effort,
// PRD §25/§30/§33.2,INV-FACT-01):
//
//   - AC2:llm_usage 行在写入时落成 cost_plane 事实字段(本适配器在写入前
//     经 Validate 拒绝非法归属面;0021 CHECK 是第二道防线);全部成本聚合
//     只按存储的 cost_plane 列分组(§25.3/§30:不存在、也不允许出现任何
//     role/use_case→plane 的查询期重算,历史永不重算)。
//   - AC3:human_effort 是人工计时的事实面(active/wall 双口径分段行只增
//     不改),聚合在读侧按 kind 完成(*_minutes 指标由聚合结果换算,
//     measurement.Minutes)。任务上不存在、也不需要任何手工记账字段。
//   - AC4:本适配器不携带、不触碰任何任务状态:两张表都没有任务状态列,
//     本文件不存在任何指向任务表的语句,也不提供任何 Update/Delete API
//     (事实面只追加,行级触发器是第二道防线)。
//
// 编码合同(migrations/0021;0022 演进用量可用性):时长一律为微秒整数
// (latency/active/wall,与 time.Duration 的换算口径固定);cost 为记账货币
// 1e-6 整数(CostMicros);llm_usage.task_id 可空(任务外控制面调用),读回为
// 空串。token/cost 三列可空:NULL = 未观测(未知,领域侧 0 值 +
// UsageReported=false),非 NULL = 已上报观测(观测到 0 合法);读取时逐行经
// Validate 与 0022 CHECK 的读侧镜像重建领域值(重建即验证,不把矛盾值当真)。
//
// 错误语义:未命中包装 pgx.ErrNoRows;聚合行的枚举列越出 CHECK 取值视为
// 行破坏,返回 error 而不是静默跳过 —— 成本报表丢行比报错更危险。

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"tally/internal/store"
)

// Store 是 LLM 使用/成本与人工计时事实的持久化适配器。只在构造时绑定
// 查询集,本身无状态;用 *store.Queries 构造(事务内传 WithTx 的查询集)。
type Store struct {
	q *store.Queries
}

// New 用 sqlc 生成的查询集构造适配器。
func New(q *store.Queries) *Store {
	return &Store{q: q}
}

// ---- llm_usage(PRD §30 llm_run 度量面) ----

// RecordLLMUsage 追加一条 LLM/Agent 调用使用/成本事实,返回新行 id。
// cost_plane 在此处落成历史事实字段:非法归属面/终态/负值等在写入前被
// Validate 拒绝,不触库(AC2)。
//
// 用量可用性(锁 16,migrations/0022):UsageReported=false 的调用照样落行,
// token/cost 三值写 NULL(领域侧零值 ↔ 库内 NULL);true 时三值是已上报观测
// (观测到 0 合法)。适配器不把未知合成为假 0,也不把已上报 0 抹成未知。
func (s *Store) RecordLLMUsage(ctx context.Context, u LLMUsage) (int64, error) {
	if err := u.Validate(); err != nil {
		return 0, err
	}
	var inputTokens, outputTokens, costMicros pgtype.Int8
	if u.UsageReported {
		inputTokens = int8Ptr(u.InputTokens)
		outputTokens = int8Ptr(u.OutputTokens)
		costMicros = int8Ptr(u.CostMicros)
	}
	id, err := s.q.InsertLLMUsage(ctx, store.InsertLLMUsageParams{
		TaskID:              textPtr(u.TaskID),
		UseCase:             u.UseCase,
		CostPlane:           string(u.CostPlane),
		Provider:            u.Provider,
		Runtime:             u.Runtime,
		Model:               u.Model,
		PromptRevision:      u.PromptRevision,
		BehaviorRevision:    u.BehaviorRevision,
		ContextHash:         u.ContextHash,
		InputTokens:         inputTokens,
		OutputTokens:        outputTokens,
		LatencyMicros:       u.Latency.Microseconds(),
		CostMicros:          costMicros,
		UsageReported:       u.UsageReported,
		Status:              string(u.Status),
		FailureType:         u.FailureType,
		HumanIntervened:     u.HumanIntervened,
		AuthoringProvenance: u.AuthoringProvenance,
		RecordedAt:          pgtype.Timestamptz{Time: u.RecordedAt.UTC(), Valid: true},
	})
	if err != nil {
		return 0, fmt.Errorf("measurement: 写入 llm usage: %w", err)
	}
	return id, nil
}

// LLMUsageForTask 返回某任务全部调用记录(旧→新),逐行重建领域值。
func (s *Store) LLMUsageForTask(ctx context.Context, taskID string) ([]LLMUsage, error) {
	rows, err := s.q.ListLLMUsageByTask(ctx, textPtr(taskID))
	if err != nil {
		return nil, fmt.Errorf("measurement: 查询任务 %s 的 llm usage: %w", taskID, err)
	}
	out := make([]LLMUsage, 0, len(rows))
	for _, row := range rows {
		u, err := llmUsageFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, nil
}

func llmUsageFromRow(row store.ListLLMUsageByTaskRow) (LLMUsage, error) {
	u := LLMUsage{
		TaskID:              textVal(row.TaskID),
		UseCase:             row.UseCase,
		CostPlane:           CostPlane(row.CostPlane),
		Provider:            row.Provider,
		Runtime:             row.Runtime,
		Model:               row.Model,
		PromptRevision:      row.PromptRevision,
		BehaviorRevision:    row.BehaviorRevision,
		ContextHash:         row.ContextHash,
		InputTokens:         int8Val(row.InputTokens),
		OutputTokens:        int8Val(row.OutputTokens),
		Latency:             time.Duration(row.LatencyMicros) * time.Microsecond,
		CostMicros:          int8Val(row.CostMicros),
		UsageReported:       row.UsageReported,
		Status:              UsageStatus(row.Status),
		FailureType:         row.FailureType,
		HumanIntervened:     row.HumanIntervened,
		AuthoringProvenance: row.AuthoringProvenance,
		RecordedAt:          tsVal(row.RecordedAt),
	}
	// 重建即验证(含 0022 CHECK 的读侧镜像):usage_reported 与 NULL/非 NULL
	// 数值不自洽的行视为行破坏,报错而不是把矛盾值当真。
	if err := usageAvailabilityConsistent(row); err != nil {
		return LLMUsage{}, fmt.Errorf("measurement: 重建 llm_usage 行 %d: %w", row.ID, err)
	}
	if err := u.Validate(); err != nil {
		return LLMUsage{}, fmt.Errorf("measurement: 重建 llm_usage 行 %d: %w", row.ID, err)
	}
	return u, nil
}

// usageAvailabilityConsistent 是 migrations/0022 llm_usage_usage_reported_consistent
// CHECK 的读侧镜像:已上报 ⇒ 三列全部非 NULL;未上报 ⇒ 三列全部 NULL。库里
// 正常不可能出现违反行(CHECK 拦截写入),读侧再验一次是为了把「绕过 CHECK
// 的行破坏」显式报错,而不是静默归零或当真。
func usageAvailabilityConsistent(row store.ListLLMUsageByTaskRow) error {
	allPresent := row.InputTokens.Valid && row.OutputTokens.Valid && row.CostMicros.Valid
	anyPresent := row.InputTokens.Valid || row.OutputTokens.Valid || row.CostMicros.Valid
	if row.UsageReported && !allPresent {
		return fmt.Errorf("usage_reported 为真但 token/cost 存在 NULL(行破坏)")
	}
	if !row.UsageReported && anyPresent {
		return fmt.Errorf("usage_reported 为假但 token/cost 存在非 NULL(行破坏)")
	}
	return nil
}

// PlaneCost 是按写入时落成的 cost_plane 聚合的一行结果(§25.3/§38.4:
// control/execution plane 成本、token 与调用数的唯一合法来源)。
//
// 调用计数三列(P0-06,锁 16:未知用量仍是一次调用):Calls = calls_total
// 全部调用(含未上报用量的);CallsReported = usage_reported 为真的行;
// CallsMissingUsage = 用量未上报的行。数值合计只累计已上报观测 —— 未上报行
// 对 CostMicros/InputTokens/OutputTokens 无贡献,也不被当作 0 计入已上报
// 合计;三者全 0 且 CallsMissingUsage>0 时读侧必须按「不完整」呈现,不得
// 读作「观测到 0 成本」。
type PlaneCost struct {
	Plane             CostPlane
	CostMicros        int64
	InputTokens       int64
	OutputTokens      int64
	Calls             int64 // calls_total = CallsReported + CallsMissingUsage
	CallsReported     int64
	CallsMissingUsage int64
}

// LLMCostByPlane 返回全库按 cost_plane 的聚合(每面至多一行,control 在前)。
func (s *Store) LLMCostByPlane(ctx context.Context) ([]PlaneCost, error) {
	rows, err := s.q.SumLLMCostByPlane(ctx)
	if err != nil {
		return nil, fmt.Errorf("measurement: 按 cost_plane 聚合 llm 成本: %w", err)
	}
	out := make([]PlaneCost, 0, len(rows))
	for _, row := range rows {
		c, err := planeCostFromRow(row.CostPlane, row.CostMicros, row.InputTokens, row.OutputTokens,
			row.CallsTotal, row.CallsReported, row.CallsMissingUsage)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

// LLMCostByPlaneForTask 返回单任务按 cost_plane 的聚合(cost per verified
// task 与按任务的成本分维读侧用)。
func (s *Store) LLMCostByPlaneForTask(ctx context.Context, taskID string) ([]PlaneCost, error) {
	rows, err := s.q.SumLLMCostByPlaneForTask(ctx, textPtr(taskID))
	if err != nil {
		return nil, fmt.Errorf("measurement: 聚合任务 %s 的 llm 成本: %w", taskID, err)
	}
	out := make([]PlaneCost, 0, len(rows))
	for _, row := range rows {
		c, err := planeCostFromRow(row.CostPlane, row.CostMicros, row.InputTokens, row.OutputTokens,
			row.CallsTotal, row.CallsReported, row.CallsMissingUsage)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func planeCostFromRow(plane string, costMicros, inputTokens, outputTokens, callsTotal, callsReported, callsMissing int64) (PlaneCost, error) {
	p := CostPlane(plane)
	if !p.IsValid() {
		return PlaneCost{}, fmt.Errorf("measurement: 聚合行 cost_plane %q 非法(行破坏,拒绝计入成本报表)", plane)
	}
	if callsTotal != callsReported+callsMissing {
		return PlaneCost{}, fmt.Errorf("measurement: 聚合行 cost_plane %q 的 calls_total=%d ≠ reported=%d + missing=%d(行破坏)",
			plane, callsTotal, callsReported, callsMissing)
	}
	return PlaneCost{
		Plane:             p,
		CostMicros:        costMicros,
		InputTokens:       inputTokens,
		OutputTokens:      outputTokens,
		Calls:             callsTotal,
		CallsReported:     callsReported,
		CallsMissingUsage: callsMissing,
	}, nil
}

// ControlPlaneRatio 从聚合行推导 control_plane_llm_cost_ratio =
// control_plane_llm_cost / total_llm_cost(§25.3):只读存储的 cost_plane
// 分组值在读侧相除,不做任何归属重算;总成本为 0(无记录或全部 0 成本)
// 时返回 (0, false):比率未定义,不猜。
func ControlPlaneRatio(costs []PlaneCost) (float64, bool) {
	var control, total int64
	for _, c := range costs {
		switch c.Plane {
		case CostPlaneControl:
			control += c.CostMicros
			total += c.CostMicros
		case CostPlaneExecution:
			total += c.CostMicros
		}
	}
	if total == 0 {
		return 0, false
	}
	return float64(control) / float64(total), true
}

// ---- human_effort(PRD §25 至少记录条目、§19.4 审核计时) ----

// RecordHumanEffort 追加一条人工控制开销事实,返回新行 id。分段上报各自
// 成行,只增不改;非法 kind/负时长/缺任务归属在写入前被 Validate 拒绝,
// 不触库(AC3)。
func (s *Store) RecordHumanEffort(ctx context.Context, h HumanEffort) (int64, error) {
	if err := h.Validate(); err != nil {
		return 0, err
	}
	id, err := s.q.InsertHumanEffort(ctx, store.InsertHumanEffortParams{
		TaskID:       h.TaskID,
		Kind:         string(h.Kind),
		ActiveMicros: h.Active.Microseconds(),
		WallMicros:   h.Wall.Microseconds(),
		RecordedAt:   pgtype.Timestamptz{Time: h.RecordedAt.UTC(), Valid: true},
	})
	if err != nil {
		return 0, fmt.Errorf("measurement: 写入 human effort: %w", err)
	}
	return id, nil
}

// HumanEffortForTask 返回某任务全部人工计时行(旧→新),逐行重建领域值:
// 分段上报的原始区间全部可见,跨行聚合在读侧(HumanEffortByKindForTask)做。
func (s *Store) HumanEffortForTask(ctx context.Context, taskID string) ([]HumanEffort, error) {
	rows, err := s.q.ListHumanEffortByTask(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("measurement: 查询任务 %s 的 human effort: %w", taskID, err)
	}
	out := make([]HumanEffort, 0, len(rows))
	for _, row := range rows {
		h := HumanEffort{
			TaskID:     row.TaskID,
			Kind:       HumanEffortKind(row.Kind),
			Active:     time.Duration(row.ActiveMicros) * time.Microsecond,
			Wall:       time.Duration(row.WallMicros) * time.Microsecond,
			RecordedAt: tsVal(row.RecordedAt),
		}
		if err := h.Validate(); err != nil {
			return nil, fmt.Errorf("measurement: 重建 human_effort 行 %d: %w", row.ID, err)
		}
		out = append(out, h)
	}
	return out, nil
}

// EffortSummary 是某任务按 kind 聚合的人工计时结果(§25/§38.2 的
// *_minutes 指标源):active/wall 各自累加,样本数即分段行数。
type EffortSummary struct {
	Kind    HumanEffortKind
	Active  time.Duration
	Wall    time.Duration
	Samples int64
}

// ActiveMinutes 返回 active 口径的分钟值(North Star 分母口径)。
func (e EffortSummary) ActiveMinutes() float64 { return Minutes(e.Active) }

// WallMinutes 返回 wall 口径的分钟值。
func (e EffortSummary) WallMinutes() float64 { return Minutes(e.Wall) }

// HumanEffortByKindForTask 按 kind 聚合某任务的全部人工计时行(每 kind
// 至多一行,kind 字典序):contract/verification/diff review 的 active/wall
// 分钟、manual_correction、spec_defect_decision 等指标由此换算,不在任务上
// 落任何手工记账字段(AC3)。
func (s *Store) HumanEffortByKindForTask(ctx context.Context, taskID string) ([]EffortSummary, error) {
	rows, err := s.q.SumHumanEffortByKindForTask(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("measurement: 聚合任务 %s 的 human effort: %w", taskID, err)
	}
	out := make([]EffortSummary, 0, len(rows))
	for _, row := range rows {
		kind := HumanEffortKind(row.Kind)
		if !kind.IsValid() {
			return nil, fmt.Errorf("measurement: 聚合行 kind %q 非法(行破坏,拒绝计入人工计时报表)", row.Kind)
		}
		out = append(out, EffortSummary{
			Kind:    kind,
			Active:  time.Duration(row.ActiveMicros) * time.Microsecond,
			Wall:    time.Duration(row.WallMicros) * time.Microsecond,
			Samples: row.Samples,
		})
	}
	return out, nil
}

// ---- pgtype 换算(本包私有,与 runstore 的辅助同义) ----

// int8Ptr/int8Val 是 token/cost 可空整数列的领域换算(锁 16):NULL ↔ 未上报
// (领域侧 0 值 + UsageReported=false),非 NULL ↔ 已上报观测(观测到 0 合法,
// 换算保留 Valid 区分,不与 NULL 混同)。
func int8Ptr(n int64) pgtype.Int8 {
	return pgtype.Int8{Int64: n, Valid: true}
}

func int8Val(n pgtype.Int8) int64 {
	if !n.Valid {
		return 0
	}
	return n.Int64
}

func textPtr(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

func textVal(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

func tsVal(t pgtype.Timestamptz) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time
}
