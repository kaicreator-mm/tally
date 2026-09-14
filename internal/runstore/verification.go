package runstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"tally/internal/changeset"
	"tally/internal/core"
	"tally/internal/store"
)

// VerificationRun 持久化(migrations/0020;PRD §6.4.3/§8.13,INV-FACT-01):
// verification_run 是某冻结 contract revision 对某 immutable ChangeSet 的一次
// 实际执行,事实面只 INSERT/SELECT(表上 UPDATE/DELETE 触发器拦截):
// 历史 Run 全量保留(§8.13),同一 (contract revision, ChangeSet) 的多次执行
// 各自成行,禁止 cherry-pick 靠全量留痕(§8.7.2)。
//
// 行落库时一次写入最终观测结果(status pass/fail/error);生命周期状态
// (VERIFIED 等)另行派生(INV-STATE-01),不是本表语义。

// VerificationStatus 是 verification_run.status:落行时一次写入的观测结果。
type VerificationStatus string

const (
	// VerificationPass 观测通过。
	VerificationPass VerificationStatus = "pass"
	// VerificationFail 观测失败。
	VerificationFail VerificationStatus = "fail"
	// VerificationError 执行出错(无可用 verdict)。
	VerificationError VerificationStatus = "error"
)

// IsValid 报告状态是否在迁移 0020 的 CHECK 取值内。
func (s VerificationStatus) IsValid() bool {
	switch s {
	case VerificationPass, VerificationFail, VerificationError:
		return true
	default:
		return false
	}
}

// VerificationRunSpec 是记一次验证执行的输入:落行时带最终观测结果,
// 行一旦落库不可改(INV-FACT-01)。
type VerificationRunSpec struct {
	Project           core.ProjectID
	Task              string
	ContractRevision  int32                 // 冻结 VerificationContract revision 号,>= 1
	ContractHash      string                // 冻结契约内容哈希(sha256:<hex>)
	ChangeSetID       changeset.ChangeSetID // immutable ChangeSet 身份(值引用 P1-11 快照面)
	EnvName           string                // 验证环境身份 = 名称 + 定义内容哈希
	EnvDefinitionHash string
	// VersionCombo 是当时环境版本组合(JSONB 冗余自包含);必须非空,
	// 空组合无法自证执行环境,拒绝落行。
	VersionCombo core.VersionCombo
	Runtime      string    // 执行后端身份(FakeRuntime/…)
	StartedAt    time.Time // 零值 = 当下(UTC)
	FinishedAt   time.Time // 零值 = NULL(未记结束时刻)
	Status       VerificationStatus
}

// RecordVerificationRun 记一次验证执行,返回行 id。
func (s *Store) RecordVerificationRun(ctx context.Context, spec VerificationRunSpec) (int64, error) {
	switch {
	case spec.Project == "":
		return 0, errors.New("runstore: 缺少 project")
	case spec.Task == "":
		return 0, errors.New("runstore: 缺少 task")
	case spec.ContractRevision < 1:
		return 0, fmt.Errorf("runstore: contract_revision 必须 >= 1(当前 %d)", spec.ContractRevision)
	case spec.ContractHash == "":
		return 0, errors.New("runstore: 缺少 contract_hash")
	case spec.ChangeSetID == "":
		return 0, errors.New("runstore: 缺少 changeset_id")
	case spec.EnvName == "":
		return 0, errors.New("runstore: 缺少 env_name")
	case spec.EnvDefinitionHash == "":
		return 0, errors.New("runstore: 缺少 env_definition_hash")
	case len(spec.VersionCombo) == 0:
		return 0, errors.New("runstore: version_combo 不能为空(执行环境必须自包含)")
	case spec.Runtime == "":
		return 0, errors.New("runstore: 缺少 runtime")
	case !spec.Status.IsValid():
		return 0, fmt.Errorf("runstore: 非法 verification_run status %q", spec.Status)
	}
	combo, err := json.Marshal(spec.VersionCombo)
	if err != nil {
		return 0, fmt.Errorf("runstore: 编码 version_combo: %w", err)
	}
	startedAt := spec.StartedAt
	if startedAt.IsZero() {
		startedAt = time.Now()
	}
	id, err := s.q.InsertVerificationRun(ctx, store.InsertVerificationRunParams{
		ProjectID:         string(spec.Project),
		TaskID:            spec.Task,
		ContractRevision:  spec.ContractRevision,
		ContractHash:      spec.ContractHash,
		ChangesetID:       string(spec.ChangeSetID),
		EnvName:           spec.EnvName,
		EnvDefinitionHash: spec.EnvDefinitionHash,
		VersionCombo:      combo,
		Runtime:           spec.Runtime,
		StartedAt:         tsPtr(startedAt),
		FinishedAt:        tsPtr(spec.FinishedAt),
		Status:            string(spec.Status),
	})
	if err != nil {
		return 0, fmt.Errorf("runstore: 写入 verification_run(task=%s changeset=%s): %w", spec.Task, spec.ChangeSetID, err)
	}
	return id, nil
}

// VerificationRun 是 verification_run 的读取模型(纯 Go 类型)。
type VerificationRun struct {
	ID                int64
	Project           core.ProjectID
	Task              string
	ContractRevision  int32
	ContractHash      string
	ChangeSetID       changeset.ChangeSetID
	EnvName           string
	EnvDefinitionHash string
	VersionCombo      core.VersionCombo
	Runtime           string
	StartedAt         time.Time
	FinishedAt        time.Time // 零值 = NULL
	Status            VerificationStatus
}

// VerificationRun 按 id 读单条 Run(项目命名空间内)并重建值(AC4)。
// 未知 id 返回包着 pgx.ErrNoRows 的错误。
func (s *Store) VerificationRun(ctx context.Context, project core.ProjectID, id int64) (*VerificationRun, error) {
	row, err := s.q.GetVerificationRun(ctx, store.GetVerificationRunParams{
		ProjectID: string(project),
		ID:        id,
	})
	if err != nil {
		return nil, fmt.Errorf("runstore: 查询 verification_run %s/%d: %w", project, id, err)
	}
	run, err := verificationRunFromRow(row)
	if err != nil {
		return nil, err
	}
	return run, nil
}

// VerificationRunsForChangeSet 返回某 immutable ChangeSet 上的全部历史 Run
// (§8.13:保留历史 Run;新到在前)。
func (s *Store) VerificationRunsForChangeSet(ctx context.Context, project core.ProjectID, cs changeset.ChangeSetID) ([]VerificationRun, error) {
	rows, err := s.q.ListVerificationRunsForChangeSet(ctx, store.ListVerificationRunsForChangeSetParams{
		ProjectID:   string(project),
		ChangesetID: string(cs),
	})
	if err != nil {
		return nil, fmt.Errorf("runstore: 查询 changeset %s 的 verification_run: %w", cs, err)
	}
	return verificationRunsFromRows(rows)
}

// VerificationRunsForTask 返回某任务的全部历史 Run(任务页 verification timeline)。
func (s *Store) VerificationRunsForTask(ctx context.Context, project core.ProjectID, task string) ([]VerificationRun, error) {
	rows, err := s.q.ListVerificationRunsForTask(ctx, store.ListVerificationRunsForTaskParams{
		ProjectID: string(project),
		TaskID:    task,
	})
	if err != nil {
		return nil, fmt.Errorf("runstore: 查询 task %s/%s 的 verification_run: %w", project, task, err)
	}
	return verificationRunsFromRows(rows)
}

func verificationRunsFromRows(rows []store.VerificationRun) ([]VerificationRun, error) {
	out := make([]VerificationRun, 0, len(rows))
	for _, row := range rows {
		run, err := verificationRunFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, *run)
	}
	return out, nil
}

func verificationRunFromRow(row store.VerificationRun) (*VerificationRun, error) {
	combo := core.VersionCombo{}
	if err := json.Unmarshal(row.VersionCombo, &combo); err != nil {
		return nil, fmt.Errorf("runstore: 解码 verification_run %d 的 version_combo: %w", row.ID, err)
	}
	status := VerificationStatus(row.Status)
	if !status.IsValid() {
		return nil, fmt.Errorf("runstore: verification_run %d 的 status 非法: %q", row.ID, row.Status)
	}
	return &VerificationRun{
		ID:                row.ID,
		Project:           core.ProjectID(row.ProjectID),
		Task:              row.TaskID,
		ContractRevision:  row.ContractRevision,
		ContractHash:      row.ContractHash,
		ChangeSetID:       changeset.ChangeSetID(row.ChangesetID),
		EnvName:           row.EnvName,
		EnvDefinitionHash: row.EnvDefinitionHash,
		VersionCombo:      combo,
		Runtime:           row.Runtime,
		StartedAt:         tsVal(row.StartedAt),
		FinishedAt:        tsVal(row.FinishedAt),
		Status:            status,
	}, nil
}
