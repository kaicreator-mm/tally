// Package runstore 为 P1 观测面持久化提供薄类型化适配器(P1-33,PRD §33.2):
// CodegenRun 过程记录、ChangeSet 不可变快照、VerificationRun / Evidence /
// Exposure / 取代事实等 append-only 机器事实。领域包(internal/changeset、
// internal/verification、internal/evidence)不 import 生成代码;本包是它们与
// internal/store 生成查询之间唯一的编码边界:写入时把领域值 / typed 入参编码成
// SQL 合同行,读取时从行重建领域值(AC4)。
//
// 不变式落点(INV-FACT-01 / INV-CHANGE-02):
//   - 事实面(changeset / supersession / verification_run / evidence / ruling /
//     exposure)只有 Insert/Append/Record 写入与读路径,不存在任何 Update/Delete
//     API(AC1);数据库层 append-only 触发器(migrations/0019、0020)是第二道
//     防线,runstore_test.go 覆盖触发器拒绝路径。
//   - codegen_run 是过程记录而非事实面:唯一允许的变更是两条带谓词的 run-control
//     迁移(进度上报与终态迁移,均限定 status='running',命中 0 行即失败,
//     终态一次性不可回退);身份坐标字段(PRD §15.2 可追溯字段)开跑时写定,
//     此后不可变(AC2)。
//   - ChangeSet 以内容推导身份 insert-once(ON CONFLICT DO NOTHING 幂等);
//     读取时经 changeset.New 重推身份,与存量行不符即报错,不把矛盾值当真(AC3)。
//
// 本包只做编码、最小校验与错误包装,不持有业务策略;调用方用 errors.Is 判因
// (pgx.ErrNoRows = 记录缺席,是证据缺席不是错误状态)。
package runstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"tally/internal/core"
	"tally/internal/measurement"
	"tally/internal/store"
)

// Store 持有 P1 观测面的持久化适配器。只在构造时绑定查询集,本身无状态。
// 用 *store.Store.Queries() 构造;事务内可用 InTx 提供的 *store.Queries。
type Store struct {
	q *store.Queries
}

// New 用 sqlc 生成的查询集构造适配器。
func New(q *store.Queries) *Store {
	return &Store{q: q}
}

// RunStatus 是 CodegenRun 的生命周期状态(migrations/0019 CHECK;§11.2.1/§17)。
type RunStatus string

const (
	// RunRunning 运行中:唯一允许 run-control 迁移的状态。
	RunRunning RunStatus = "running"
	// RunCompleted 正常完成。
	RunCompleted RunStatus = "completed"
	// RunFailed 失败终止。
	RunFailed RunStatus = "failed"
	// RunEscalated 升级终止(治理升级后另开新 Run 记 escalated_from)。
	RunEscalated RunStatus = "escalated"
	// RunCancelled 取消终止。
	RunCancelled RunStatus = "cancelled"
)

// IsValid 报告状态是否在迁移 0019 的 CHECK 取值内。
func (s RunStatus) IsValid() bool {
	switch s {
	case RunRunning, RunCompleted, RunFailed, RunEscalated, RunCancelled:
		return true
	default:
		return false
	}
}

// Terminal 报告是否终态(合法且非 running)。终态一次性,不可回退重写。
func (s RunStatus) Terminal() bool {
	return s.IsValid() && s != RunRunning
}

// StartRunSpec 是开跑输入。身份坐标(PRD §15.2 可追溯字段)此时全部已知,
// 写入后永不可变:升级不得在同一 Run 内替换 runtime/model,只能终止旧 Run、
// 新 Run 记 EscalatedFrom(§11.2.1)。
type StartRunSpec struct {
	Project              core.ProjectID
	Task                 string // TaskContract 身份(Git 持有意图,这里只存引用)
	Role                 string
	Runtime              string
	RuntimeVersion       string
	Model                string
	BehaviorRevision     string
	CostPlane            measurement.CostPlane // 成本归属面,与 internal/measurement 取值一致(PRD §30)
	BaseCommit           string
	TaskRevision         string // 执行所对 TaskContract 修订哈希
	VerificationRevision string // 执行所对 VerificationContract 修订哈希
	ContextPackHash      string
	WorkspaceID          string // 一次性 Git worktree(P0-05/P1-35)
	// AgentSessionID 是 Runtime 会话身份;会话未及建立即崩溃时为空。
	AgentSessionID string
	// ObservedRuntimeName / ObservedRuntimeVersion / ObservedModel 是 runtime
	// 现场自报的实际身份事实(P0-02;PRD §11.3;Decision Lock 15):与上面的
	// 期望坐标(Runtime/RuntimeVersion/Model)分开记录,互不回填。全部零值
	// = 完全未观察(合法;未观察 ≠ 期望值)。这些事实没有独立列,非空时以
	// 专属 observed_identity 键并入 RuntimeCapabilitySnapshot JSONB 落库
	//(该列即自包含 runtime 快照;期望坐标列语义不变,两类事实一行可区分)。
	ObservedRuntimeName    string // 实际 runtime 自报标识(如 "codex-cli");空 = 未观察
	ObservedRuntimeVersion string // 实际 runtime/CLI 版本(现场实测,非适配器版本);空 = 未观察
	ObservedModel          string // 实际生效模型;空 = 未观察
	// RuntimeCapabilitySnapshot 是运行时声明能力快照(§10.3),必须是非空 JSON
	// 对象;原样存 JSONB,本包不解释其内容 —— 唯一例外:observed_* 输入非零时
	// 由本包注入顶层 "observed_identity" 键(该键为本包保留,调用方不得预置)。
	RuntimeCapabilitySnapshot json.RawMessage
	// EscalatedFrom 是被升级终止的旧 Run id;0 = 非升级 Run。
	EscalatedFrom int64
	// StartedAt 为零值时取当下(UTC)。
	StartedAt time.Time
}

func (spec StartRunSpec) validate() error {
	switch {
	case spec.Project == "":
		return errors.New("runstore: 缺少 project")
	case spec.Task == "":
		return errors.New("runstore: 缺少 task")
	case spec.Role == "":
		return errors.New("runstore: 缺少 role")
	case spec.Runtime == "":
		return errors.New("runstore: 缺少 runtime")
	case spec.RuntimeVersion == "":
		return errors.New("runstore: 缺少 runtime_version")
	case spec.Model == "":
		return errors.New("runstore: 缺少 model")
	case spec.BehaviorRevision == "":
		return errors.New("runstore: 缺少 behavior_revision")
	case !spec.CostPlane.IsValid():
		return fmt.Errorf("runstore: 非法 cost_plane %q", spec.CostPlane)
	case spec.BaseCommit == "":
		return errors.New("runstore: 缺少 base_commit")
	case spec.TaskRevision == "":
		return errors.New("runstore: 缺少 task_revision")
	case spec.VerificationRevision == "":
		return errors.New("runstore: 缺少 verification_revision")
	case spec.ContextPackHash == "":
		return errors.New("runstore: 缺少 context_pack_hash")
	case spec.WorkspaceID == "":
		return errors.New("runstore: 缺少 workspace_id")
	case len(spec.RuntimeCapabilitySnapshot) == 0 || !json.Valid(spec.RuntimeCapabilitySnapshot):
		return errors.New("runstore: runtime_capability_snapshot 必须是非空 JSON")
	case !isJSONObject(spec.RuntimeCapabilitySnapshot):
		return errors.New("runstore: runtime_capability_snapshot 必须是 JSON 对象")
	case hasTopLevelKey(spec.RuntimeCapabilitySnapshot, observedIdentityKey):
		return fmt.Errorf("runstore: runtime_capability_snapshot 顶层键 %q 为本包保留(observed_* 输入的落库位置),调用方不得预置", observedIdentityKey)
	case spec.EscalatedFrom < 0:
		return errors.New("runstore: escalated_from 不能为负")
	default:
		return nil
	}
}

// StartCodegenRun 记一次正式 Agent 执行的开始:写入过程记录,status 恒以
// 'running' 起步,返回代理主键 id。
func (s *Store) StartCodegenRun(ctx context.Context, spec StartRunSpec) (int64, error) {
	if err := spec.validate(); err != nil {
		return 0, err
	}
	snapshot := spec.RuntimeCapabilitySnapshot
	if enriched, err := attachObservedIdentity(snapshot, spec.ObservedRuntimeName, spec.ObservedRuntimeVersion, spec.ObservedModel); err != nil {
		return 0, err
	} else {
		snapshot = enriched
	}
	startedAt := spec.StartedAt
	if startedAt.IsZero() {
		startedAt = time.Now()
	}
	session := pgtype.Text{}
	if spec.AgentSessionID != "" {
		session = pgtype.Text{String: spec.AgentSessionID, Valid: true}
	}
	var escalated pgtype.Int8
	if spec.EscalatedFrom != 0 {
		escalated = pgtype.Int8{Int64: spec.EscalatedFrom, Valid: true}
	}
	id, err := s.q.InsertCodegenRun(ctx, store.InsertCodegenRunParams{
		ProjectID:                 string(spec.Project),
		TaskID:                    spec.Task,
		Role:                      spec.Role,
		Runtime:                   spec.Runtime,
		RuntimeVersion:            spec.RuntimeVersion,
		Model:                     spec.Model,
		BehaviorRevision:          spec.BehaviorRevision,
		CostPlane:                 string(spec.CostPlane),
		BaseCommit:                spec.BaseCommit,
		TaskRevision:              spec.TaskRevision,
		VerificationRevision:      spec.VerificationRevision,
		ContextPackHash:           spec.ContextPackHash,
		WorkspaceID:               spec.WorkspaceID,
		AgentSessionID:            session,
		RuntimeCapabilitySnapshot: []byte(snapshot),
		EscalatedFrom:             escalated,
		StartedAt:                 tsPtr(startedAt),
	})
	if err != nil {
		return 0, fmt.Errorf("runstore: 写入 codegen_run(project=%s task=%s): %w", spec.Project, spec.Task, err)
	}
	return id, nil
}

// observedIdentityKey 是 observed_* 身份事实在 runtime_capability_snapshot
// JSONB 内的落库键(本包保留;validate 拒绝调用方预置)。
const observedIdentityKey = "observed_identity"

// observedIdentityJSON 是 observed_identity 键的编码形状(omitempty:缺即未
// 观察,不落假空值)。键名是 runtime 中立词汇,不带 runtime 名分支
// (P0-02:无 Codex 专用 DB 路径)。
type observedIdentityJSON struct {
	RuntimeName    string `json:"runtime_name,omitempty"`
	RuntimeVersion string `json:"runtime_version,omitempty"`
	Model          string `json:"model,omitempty"`
}

// attachObservedIdentity 把 observed_* 实际身份事实并入能力快照 JSON:全部
// 未观察(零值)时原样返回 data(字节不动,既有调用方行为不变);否则注入
// 顶层 observed_identity 键。快照的其余键原样保留(调用方内容不丢失)。
// 编码用 map 重排:encoding/json 对 map 键排序,输出确定。
func attachObservedIdentity(data json.RawMessage, name, version, model string) (json.RawMessage, error) {
	obs := observedIdentityJSON{RuntimeName: name, RuntimeVersion: version, Model: model}
	if obs == (observedIdentityJSON{}) {
		return data, nil
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(data, &top); err != nil {
		return nil, fmt.Errorf("runstore: 解码 runtime_capability_snapshot 以注入 %s: %w", observedIdentityKey, err)
	}
	enc, err := json.Marshal(obs)
	if err != nil {
		return nil, fmt.Errorf("runstore: 编码 %s: %w", observedIdentityKey, err)
	}
	top[observedIdentityKey] = enc
	out, err := json.Marshal(top)
	if err != nil {
		return nil, fmt.Errorf("runstore: 编码含 %s 的 runtime_capability_snapshot: %w", observedIdentityKey, err)
	}
	return out, nil
}

// hasTopLevelKey 报告 JSON 对象是否带指定顶层键(validate 已保证是 JSON 对象)。
func hasTopLevelKey(data json.RawMessage, key string) bool {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(data, &top); err != nil {
		return false
	}
	_, ok := top[key]
	return ok
}

// RunProgress 是运行中的一次进度/用量上报(AC2 的 run-control 语义):
// 只允许动 iterations 与 usage 列,且只对仍处 running 的 Run 生效。
type RunProgress struct {
	ID         int64
	Iterations int32
	// InputTokens / OutputTokens / CostMicros:nil = 尚未上报(NULL)。
	InputTokens  *int64
	OutputTokens *int64
	CostMicros   *int64
}

// UpdateRunProgress 上报运行进度。返回 false = Run 不存在或已终态
// (谓词 status='running' 命中 0 行),调用方必须放弃,不得另开写入路径。
func (s *Store) UpdateRunProgress(ctx context.Context, p RunProgress) (bool, error) {
	switch {
	case p.ID <= 0:
		return false, errors.New("runstore: 进度上报缺少 Run id")
	case p.Iterations < 0:
		return false, errors.New("runstore: iterations 不能为负")
	case int64LessThan(p.InputTokens, 0), int64LessThan(p.OutputTokens, 0), int64LessThan(p.CostMicros, 0):
		return false, errors.New("runstore: usage 计数不能为负")
	}
	n, err := s.q.UpdateCodegenRunProgress(ctx, store.UpdateCodegenRunProgressParams{
		ID:           p.ID,
		Iterations:   p.Iterations,
		InputTokens:  int8Ptr(p.InputTokens),
		OutputTokens: int8Ptr(p.OutputTokens),
		CostMicros:   int8Ptr(p.CostMicros),
	})
	if err != nil {
		return false, fmt.Errorf("runstore: 更新 codegen_run %d 进度: %w", p.ID, err)
	}
	return n > 0, nil
}

// FinishRunSpec 是终态迁移输入(§11.2.1/§17):一次性,谓词限定 status='running',
// 二次迁移或对未知 id 命中 0 行。
type FinishRunSpec struct {
	ID         int64
	Status     RunStatus // 必须是终态(completed/failed/escalated/cancelled)
	FinishedAt time.Time // 零值 = 当下(UTC)
	// Iterations 取终值一并落定。
	Iterations int32
	// InputTokens / OutputTokens / CostMicros:nil = 尚未上报(NULL)。
	InputTokens  *int64
	OutputTokens *int64
	CostMicros   *int64
	// ResultPatchHash 仅在有固定 candidate 的 Run 非空(PRD §11.2.1)。
	ResultPatchHash string
	// AgentClaim 是 Agent 的 Claim 原样记录;Claim 永非 Project Fact(INV-AUTH-01/02)。
	AgentClaim string
	// ValidationResult 是确定性 validation 的记录(写入方编码)。
	ValidationResult string
	// FailureReason 是终态为 failed/escalated/cancelled 时的原因。
	FailureReason string
}

// FinishCodegenRun 执行终态迁移。返回 false = Run 不存在或已终态;
// 终态一次性,不可回退重写。
func (s *Store) FinishCodegenRun(ctx context.Context, spec FinishRunSpec) (bool, error) {
	switch {
	case spec.ID <= 0:
		return false, errors.New("runstore: 终态迁移缺少 Run id")
	case !spec.Status.IsValid():
		return false, fmt.Errorf("runstore: 非法 Run 状态 %q", spec.Status)
	case !spec.Status.Terminal():
		return false, fmt.Errorf("runstore: 终态迁移目标不能是 %q(必须是 completed/failed/escalated/cancelled)", spec.Status)
	case int64LessThan(spec.InputTokens, 0), int64LessThan(spec.OutputTokens, 0), int64LessThan(spec.CostMicros, 0):
		return false, errors.New("runstore: usage 计数不能为负")
	}
	finishedAt := spec.FinishedAt
	if finishedAt.IsZero() {
		finishedAt = time.Now()
	}
	n, err := s.q.FinishCodegenRun(ctx, store.FinishCodegenRunParams{
		ID:               spec.ID,
		Status:           string(spec.Status),
		FinishedAt:       tsPtr(finishedAt),
		Iterations:       spec.Iterations,
		InputTokens:      int8Ptr(spec.InputTokens),
		OutputTokens:     int8Ptr(spec.OutputTokens),
		CostMicros:       int8Ptr(spec.CostMicros),
		ResultPatchHash:  textPtr(spec.ResultPatchHash),
		AgentClaim:       textPtr(spec.AgentClaim),
		ValidationResult: textPtr(spec.ValidationResult),
		FailureReason:    textPtr(spec.FailureReason),
	})
	if err != nil {
		return false, fmt.Errorf("runstore: 迁移 codegen_run %d 到终态 %s: %w", spec.ID, spec.Status, err)
	}
	return n > 0, nil
}

// Run 是 CodegenRun 的读取模型:纯 Go 类型,pgtype 不外泄;可空列按零值/nil 语义。
type Run struct {
	ID                        int64
	Project                   core.ProjectID
	Task                      string
	Role                      string
	Runtime                   string
	RuntimeVersion            string
	Model                     string
	BehaviorRevision          string
	CostPlane                 measurement.CostPlane
	BaseCommit                string
	TaskRevision              string
	VerificationRevision      string
	ContextPackHash           string
	WorkspaceID               string
	AgentSessionID            string // 空 = 会话未及建立即崩溃
	StartedAt                 time.Time
	FinishedAt                time.Time // 零值 = 未结束(running)
	Status                    RunStatus
	Iterations                int32
	EscalatedFrom             int64  // 0 = 非升级 Run
	InputTokens               *int64 // nil = 尚未上报
	OutputTokens              *int64
	CostMicros                *int64
	ResultPatchHash           string // 空 = 没有固定 candidate 的 Run
	AgentClaim                string
	ValidationResult          string
	FailureReason             string
	RuntimeCapabilitySnapshot json.RawMessage // 可能含本包注入的 observed_identity 键(开跑时的实际身份观察)
	ConformanceEvidenceRef    string          // 空 = 本 Run 未绑定
}

// CodegenRun 按 id 读整行(全部 §15.1 字段)。未知 id 返回包着 pgx.ErrNoRows 的错误。
func (s *Store) CodegenRun(ctx context.Context, id int64) (*Run, error) {
	row, err := s.q.GetCodegenRun(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("runstore: 查询 codegen_run %d: %w", id, err)
	}
	run := runFromRow(row)
	return &run, nil
}

// CodegenRunsForTask 返回某 Task 的全部 Run 时间线(PRD §29),按开始时刻定序。
func (s *Store) CodegenRunsForTask(ctx context.Context, project core.ProjectID, task string) ([]Run, error) {
	rows, err := s.q.ListCodegenRunsForTask(ctx, store.ListCodegenRunsForTaskParams{
		ProjectID: string(project),
		TaskID:    task,
	})
	if err != nil {
		return nil, fmt.Errorf("runstore: 查询 task %s/%s 的 codegen_run: %w", project, task, err)
	}
	runs := make([]Run, 0, len(rows))
	for _, row := range rows {
		runs = append(runs, runFromRow(row))
	}
	return runs, nil
}

func runFromRow(row store.CodegenRun) Run {
	return Run{
		ID:                        row.ID,
		Project:                   core.ProjectID(row.ProjectID),
		Task:                      row.TaskID,
		Role:                      row.Role,
		Runtime:                   row.Runtime,
		RuntimeVersion:            row.RuntimeVersion,
		Model:                     row.Model,
		BehaviorRevision:          row.BehaviorRevision,
		CostPlane:                 measurement.CostPlane(row.CostPlane),
		BaseCommit:                row.BaseCommit,
		TaskRevision:              row.TaskRevision,
		VerificationRevision:      row.VerificationRevision,
		ContextPackHash:           row.ContextPackHash,
		WorkspaceID:               row.WorkspaceID,
		AgentSessionID:            textVal(row.AgentSessionID),
		StartedAt:                 tsVal(row.StartedAt),
		FinishedAt:                tsVal(row.FinishedAt),
		Status:                    RunStatus(row.Status),
		Iterations:                row.Iterations,
		EscalatedFrom:             int8Scalar(row.EscalatedFrom),
		InputTokens:               int8Val(row.InputTokens),
		OutputTokens:              int8Val(row.OutputTokens),
		CostMicros:                int8Val(row.CostMicros),
		ResultPatchHash:           textVal(row.ResultPatchHash),
		AgentClaim:                textVal(row.AgentClaim),
		ValidationResult:          textVal(row.ValidationResult),
		FailureReason:             textVal(row.FailureReason),
		RuntimeCapabilitySnapshot: append(json.RawMessage(nil), row.RuntimeCapabilitySnapshot...),
		ConformanceEvidenceRef:    textVal(row.ConformanceEvidenceRef),
	}
}

// ---- pgtype ↔ 纯 Go 类型的小转换(包内共用) ----

func textPtr(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

func textVal(t pgtype.Text) string {
	if t.Valid {
		return t.String
	}
	return ""
}

func int8Ptr(p *int64) pgtype.Int8 {
	if p == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: *p, Valid: true}
}

func int8Val(t pgtype.Int8) *int64 {
	if !t.Valid {
		return nil
	}
	v := t.Int64
	return &v
}

func int8Scalar(t pgtype.Int8) int64 {
	if !t.Valid {
		return 0
	}
	return t.Int64
}

func int4Val(t pgtype.Int4) *int32 {
	if !t.Valid {
		return nil
	}
	v := t.Int32
	return &v
}

func tsPtr(t time.Time) pgtype.Timestamptz {
	if t.IsZero() {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func tsVal(t pgtype.Timestamptz) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time
}

func int64LessThan(p *int64, v int64) bool {
	return p != nil && *p < v
}

// isJSONObject 报告 data 的首个有效字节是否开一个 JSON 对象
// (runtime_capability_snapshot / detail 列的形状护栏:JSONB 对象,不是标量)。
func isJSONObject(data []byte) bool {
	for _, b := range data {
		switch b {
		case ' ', '\t', '\r', '\n':
			continue
		case '{':
			return true
		default:
			return false
		}
	}
	return false
}
