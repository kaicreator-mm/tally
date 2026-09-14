package runstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"

	"tally/internal/changeset"
	"tally/internal/core"
	"tally/internal/store"
)

// ChangeSet 持久化(migrations/0019;PRD §15.3/§6.4.2,INV-CHANGE-02):
//
//   - changeset 是不可变候选快照:只 INSERT(ON CONFLICT (id) DO NOTHING),
//     UPDATE/DELETE 由触发器拦截。快照身份 id 由写入方按 internal/changeset
//     推导("sha256:<hex>"),同一 Run 的同一候选同 ID,重复写入幂等忽略(AC3)。
//   - changeset_superseded_fact 是取代关系 append-only 事实:old/new/reason
//     只追加,永不修改;后续重做产生新快照 + 新取代事实,superseded_by 一律
//     从取代事实投影派生,快照本体无可变取代字段(PRD §15.3,AC1)。
//
// changed_files / entities 的 JSONB 编码形状由本适配器定义:
// changed_files = internal/changeset.ChangedFile 数组(字段 Path/Content/
// Executable/Hunks),entities = 字符串数组;空集合编码为 "[]"。
// 读取时经 changeset.New 重建领域值并重推内容身份,与行上存量不符即报错(AC4)。

// SaveChangeSet 固定一个候选快照:只 INSERT,身份冲突(同一快照已存在,
// 典型为写入重试/重放)按 SQL 合同幂等忽略,不报错也不更新。
func (s *Store) SaveChangeSet(ctx context.Context, cs *changeset.ChangeSet) error {
	if cs == nil {
		return errors.New("runstore: ChangeSet 为 nil")
	}
	// 表列 producer_run_id 是 Run 的数字主键;领域值是引用串,此处换算。
	runID, err := strconv.ParseInt(cs.ProducerRun(), 10, 64)
	if err != nil || runID <= 0 {
		return fmt.Errorf("runstore: ChangeSet %s 的 producer run %q 不是 Run 数字 id: %w", cs.ID(), cs.ProducerRun(), err)
	}
	files, err := encodeChangedFiles(cs.Files())
	if err != nil {
		return fmt.Errorf("runstore: 编码 ChangeSet %s 改动文件: %w", cs.ID(), err)
	}
	entities, err := encodeStrings(cs.Entities())
	if err != nil {
		return fmt.Errorf("runstore: 编码 ChangeSet %s entities: %w", cs.ID(), err)
	}
	if err := s.q.InsertChangeSet(ctx, store.InsertChangeSetParams{
		ID:              string(cs.ID()),
		ProjectID:       string(cs.Project()),
		BaseCommit:      cs.BaseRevision(),
		CandidateCommit: textPtr(cs.CandidateRevision()),
		PatchHash:       textPtr(cs.PatchIdentity()),
		ChangedFiles:    files,
		Entities:        entities,
		ProducerRunID:   runID,
		CreatedAt:       tsPtr(cs.CreatedAt()),
	}); err != nil {
		return fmt.Errorf("runstore: 写入 changeset %s: %w", cs.ID(), err)
	}
	return nil
}

// ChangeSet 按 id 读回快照并重建领域值(AC4)。未知 id 返回包着 pgx.ErrNoRows 的错误。
//
// 重建经 changeset.New 重推 patch 身份与快照身份:任一与行上存量不一致 =
// 存量行与内容推导身份矛盾(行被破坏),报错而不是把矛盾值当真。
func (s *Store) ChangeSet(ctx context.Context, id changeset.ChangeSetID) (*changeset.ChangeSet, error) {
	row, err := s.q.GetChangeSet(ctx, string(id))
	if err != nil {
		return nil, fmt.Errorf("runstore: 查询 changeset %s: %w", id, err)
	}
	return changeSetFromRow(row)
}

// ChangeSetsForTask 返回某 Task 的全部候选快照(经产生它们的 Run 归属),
// 按固定时刻定序。
func (s *Store) ChangeSetsForTask(ctx context.Context, project core.ProjectID, task string) ([]*changeset.ChangeSet, error) {
	rows, err := s.q.ListChangeSetsForTask(ctx, store.ListChangeSetsForTaskParams{
		ProjectID: string(project),
		TaskID:    task,
	})
	if err != nil {
		return nil, fmt.Errorf("runstore: 查询 task %s/%s 的 changeset: %w", project, task, err)
	}
	out := make([]*changeset.ChangeSet, 0, len(rows))
	for _, row := range rows {
		cs, err := changeSetFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, cs)
	}
	return out, nil
}

// AppendSupersededFact 追加一条取代事实(append-only,PRD §15.3,AC1):
// old 被 new 取代。同一事实(同 old/new/reason/时刻)重复追加按 SQL 合同
// 幂等忽略;不同时刻各自成行。
func (s *Store) AppendSupersededFact(ctx context.Context, f changeset.SupersededFact) error {
	if err := validateSupersededFact(f); err != nil {
		return err
	}
	if err := s.q.InsertChangeSetSupersededFact(ctx, store.InsertChangeSetSupersededFactParams{
		OldChangesetID: string(f.Old),
		NewChangesetID: string(f.New),
		Reason:         string(f.Reason),
		SupersededAt:   tsPtr(f.At),
	}); err != nil {
		return fmt.Errorf("runstore: 追加取代事实 %s→%s: %w", f.Old, f.New, err)
	}
	return nil
}

// ChangeSetSupersessions 返回某快照参与的取代事实(作为 old 或 new),
// 按发生时刻定序:派生 superseded_by / 取代链的投影源。
func (s *Store) ChangeSetSupersessions(ctx context.Context, id changeset.ChangeSetID) ([]changeset.SupersededFact, error) {
	rows, err := s.q.ListChangeSetSupersessionsForChangeSet(ctx, string(id))
	if err != nil {
		return nil, fmt.Errorf("runstore: 查询 changeset %s 的取代事实: %w", id, err)
	}
	out := make([]changeset.SupersededFact, 0, len(rows))
	for _, r := range rows {
		f, err := supersededFactFromRow(r.OldChangesetID, r.NewChangesetID, r.Reason, r.SupersededAt)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, nil
}

// AllChangeSetSupersessions 返回全部取代事实(排序确定):
// 取代投影整体重建 / 对账用(INV-STATE-01 可重建)。
func (s *Store) AllChangeSetSupersessions(ctx context.Context) ([]changeset.SupersededFact, error) {
	rows, err := s.q.ListChangeSetSupersessions(ctx)
	if err != nil {
		return nil, fmt.Errorf("runstore: 查询全部取代事实: %w", err)
	}
	out := make([]changeset.SupersededFact, 0, len(rows))
	for _, r := range rows {
		f, err := supersededFactFromRow(r.OldChangesetID, r.NewChangesetID, r.Reason, r.SupersededAt)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, nil
}

func supersededFactFromRow(oldID, newID, reason string, at pgtype.Timestamptz) (changeset.SupersededFact, error) {
	f, err := changeset.NewSupersededFact(
		changeset.ChangeSetID(oldID),
		changeset.ChangeSetID(newID),
		changeset.SupersedeReason(reason),
		tsVal(at),
	)
	if err != nil {
		return changeset.SupersededFact{}, fmt.Errorf("runstore: 重建取代事实 %s→%s(%s): %w", oldID, newID, reason, err)
	}
	return f, nil
}

func validateSupersededFact(f changeset.SupersededFact) error {
	switch {
	case f.Old == "" || f.New == "":
		return fmt.Errorf("%w: old/new 为空", changeset.ErrBadSupersede)
	case f.Old == f.New:
		return fmt.Errorf("%w: old 与 new 相同", changeset.ErrBadSupersede)
	case f.At.IsZero():
		return fmt.Errorf("%w: 时刻为零值", changeset.ErrBadSupersede)
	}
	switch f.Reason {
	case changeset.SupersedeEscalation, changeset.SupersedeRework, changeset.SupersedeSpecDefect:
	default:
		return fmt.Errorf("%w: 未知原因 %q", changeset.ErrBadSupersede, f.Reason)
	}
	return nil
}

// changeSetFromRow 从行重建领域快照并核对内容身份(AC3/AC4)。
func changeSetFromRow(row store.Changeset) (*changeset.ChangeSet, error) {
	files, err := decodeChangedFiles(row.ChangedFiles)
	if err != nil {
		return nil, fmt.Errorf("runstore: 解码 changeset %s 改动文件: %w", row.ID, err)
	}
	entities, err := decodeStrings(row.Entities)
	if err != nil {
		return nil, fmt.Errorf("runstore: 解码 changeset %s entities: %w", row.ID, err)
	}
	cs, err := changeset.New(changeset.Spec{
		Project:           core.ProjectID(row.ProjectID),
		BaseRevision:      row.BaseCommit,
		CandidateRevision: textVal(row.CandidateCommit),
		Files:             files,
		Entities:          entities,
		ProducerRun:       strconv.FormatInt(row.ProducerRunID, 10),
		CreatedAt:         tsVal(row.CreatedAt),
	})
	if err != nil {
		return nil, fmt.Errorf("runstore: 重建 changeset %s: %w", row.ID, err)
	}
	// 内容推导身份必须与存量一致:patch 身份与快照 id 双重对账。
	if got := cs.PatchIdentity(); got != textVal(row.PatchHash) {
		return nil, fmt.Errorf("runstore: changeset %s 的 patch 身份与存量不一致(重建 %q,行 %q)", row.ID, got, textVal(row.PatchHash))
	}
	if cs.ID() != changeset.ChangeSetID(row.ID) {
		return nil, fmt.Errorf("runstore: changeset %s 的快照身份与存量不一致(重建 %q)", row.ID, cs.ID())
	}
	return cs, nil
}

// ---- changed_files / entities 的 JSONB 编码 ----

func encodeChangedFiles(files []changeset.ChangedFile) ([]byte, error) {
	if len(files) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal(files)
}

func decodeChangedFiles(data []byte) ([]changeset.ChangedFile, error) {
	if isJSONNull(data) {
		return nil, nil
	}
	var files []changeset.ChangedFile
	if err := json.Unmarshal(data, &files); err != nil {
		return nil, err
	}
	return files, nil
}

func encodeStrings(xs []string) ([]byte, error) {
	if len(xs) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal(xs)
}

func decodeStrings(data []byte) ([]string, error) {
	if isJSONNull(data) {
		return nil, nil
	}
	var xs []string
	if err := json.Unmarshal(data, &xs); err != nil {
		return nil, err
	}
	return xs, nil
}

func isJSONNull(data []byte) bool {
	return string(data) == "null"
}
