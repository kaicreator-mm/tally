package approval

// HumanApprovalEvidence 持久化适配器(P1-34,migrations/0021 human_approval,
// PRD §18.1/§19/§8.10,INV-FACT-01):
//
//   - AC1:approval 是 append-only 证据。写路径只有 INSERT(0021 行级触发器
//     拦截 UPDATE/DELETE 是第二道防线);同一候选后来的新决定追加新行,历史
//     行永不改写 —— 后来 pass 不洗白更早的记录,反之亦然。
//   - AC4:本适配器不携带、不触碰任何任务状态。APPROVED 等权威状态由状态
//     引擎从本表证据派生(§18.1:不是 UPDATE task SET status='APPROVED');
//     本文件不存在任何指向任务表的语句。
//
// 编码合同(migrations/0021):
//   - waiver provenance 拆两列:waiver_reason_code 存原因代码原值;
//     waiver_refs(JSONB 数组)按写入顺序存其余 provenance 的自描述条目
//     (category/gaps/compromised acceptances/compromised bindings/exposure
//     refs/owner 接受),读取时重建 *IndependenceWaiver 并经 NewWaiver 全量
//     校验(重建即验证,不把矛盾值当真)。
//   - refs 列(JSON 对象)在 P1 领域类型 HumanApprovalEvidence 中没有对应
//     坐标,写入固定为 '{}',不做领域外的自由文本扩展;读侧忽略该列。
//
// 错误语义:查询未命中包装 pgx.ErrNoRows(证据缺席不是错误状态,调用方用
// errors.Is 判因);重建失败(行被破坏/矛盾)返回 error,绝不静默修正。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"tally/internal/core"
	"tally/internal/evidence"
	"tally/internal/store"
)

// Store 是 HumanApprovalEvidence 的持久化适配器。只在构造时绑定查询集,
// 本身无状态;用 *store.Queries 构造(事务内传 WithTx 的查询集)。
type Store struct {
	q *store.Queries
}

// New 用 sqlc 生成的查询集构造适配器。
func New(q *store.Queries) *Store {
	return &Store{q: q}
}

// RecordedApproval 是一条已落库的决定证据:行 id + 重建的领域值。
type RecordedApproval struct {
	// ID 是 human_approval 行 id(代理主键);历史行按 id 仍可回查。
	ID int64
	// Approval 是从行重建并经领域构造器全量校验的证据值。
	Approval HumanApprovalEvidence
}

// RecordApproval 追加一条决定证据(AC1),返回新行 id。零值或以其他方式
// 破坏的证据在写入前经领域构造器重校验并拒绝(不碰库)。
func (s *Store) RecordApproval(ctx context.Context, a HumanApprovalEvidence) (int64, error) {
	validated, err := a.validatedForStorage()
	if err != nil {
		return 0, err
	}
	waiverRefs, err := waiverRefsJSON(validated.Waiver())
	if err != nil {
		return 0, err
	}
	var reasonCode pgtype.Text
	if w := validated.Waiver(); w != nil {
		reasonCode = pgtype.Text{String: w.ReasonCode(), Valid: true}
	}
	var csID pgtype.Text
	if cs, ok := validated.ChangeSetID(); ok {
		csID = pgtype.Text{String: cs, Valid: true}
	}
	id, err := s.q.InsertHumanApproval(ctx, store.InsertHumanApprovalParams{
		ProjectID:          string(validated.Project()),
		TaskID:             validated.TaskID(),
		ChangesetID:        csID,
		Decision:           string(validated.Decision()),
		Reason:             validated.Reason(),
		DecidedBy:          validated.DecidedBy(),
		DecidedAt:          pgtype.Timestamptz{Time: validated.DecidedAt(), Valid: true},
		WaivedIndependence: validated.WaivedIndependence(),
		WaiverReasonCode:   reasonCode,
		WaiverRefs:         waiverRefs,
		Refs:               []byte("{}"),
	})
	if err != nil {
		return 0, fmt.Errorf("approval: 写入决定证据: %w", err)
	}
	return id, nil
}

// Approval 按 id 读回一条决定证据并重建领域值。未知 id 返回包着
// pgx.ErrNoRows 的错误。
func (s *Store) Approval(ctx context.Context, id int64) (*RecordedApproval, error) {
	row, err := s.q.GetHumanApproval(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("approval: 查询决定证据 %d: %w", id, err)
	}
	out, err := approvalFromRow(row)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ApprovalsForTask 返回某任务全部决定证据(新→旧;AC1:历史行全部可见,
// 不覆盖、不瘦身 —— 状态引擎的时间线视图)。
func (s *Store) ApprovalsForTask(ctx context.Context, project core.ProjectID, taskID string) ([]RecordedApproval, error) {
	rows, err := s.q.ListHumanApprovalsByTask(ctx, store.ListHumanApprovalsByTaskParams{
		ProjectID: string(project),
		TaskID:    taskID,
	})
	if err != nil {
		return nil, fmt.Errorf("approval: 查询任务 %s/%s 的决定证据: %w", project, taskID, err)
	}
	return approvalsFromRows(rows)
}

// LatestApprovalForChangeSet 返回某 ChangeSet 的最新一条决定(§19:Review
// 输入是固定 revision 的 ChangeSet;状态引擎据此派生该候选当前是否带有效
// APPROVED 证据)。无任何决定行时返回包着 pgx.ErrNoRows 的错误(未决定是
// 证据缺席);更早的行仍可经 ApprovalsForTask/Approval 回查。
func (s *Store) LatestApprovalForChangeSet(ctx context.Context, project core.ProjectID, taskID, changesetID string) (*RecordedApproval, error) {
	if changesetID == "" {
		return nil, errors.New("approval: 缺少 changeset_id(候选裁决必须指向固定 revision 的 ChangeSet)")
	}
	row, err := s.q.GetLatestHumanApprovalByChangeset(ctx, store.GetLatestHumanApprovalByChangesetParams{
		ProjectID:   string(project),
		TaskID:      taskID,
		ChangesetID: pgtype.Text{String: changesetID, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("approval: 查询候选 %s 的最新决定: %w", changesetID, err)
	}
	out, err := approvalFromRow(row)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// WaivedApprovalsForProject 返回某项目全部豁免行(新→旧):strict_waived 与
// strict_satisfied 的分开统计源(§38.5);strict_satisfied 行不进本投影。
func (s *Store) WaivedApprovalsForProject(ctx context.Context, project core.ProjectID) ([]RecordedApproval, error) {
	rows, err := s.q.ListWaivedHumanApprovalsByProject(ctx, string(project))
	if err != nil {
		return nil, fmt.Errorf("approval: 查询项目 %s 的豁免行: %w", project, err)
	}
	return approvalsFromRows(rows)
}

// validatedForStorage 经领域构造器重校验并返回归一化(UTC)的等值证据:
// 零值/矛盾值(如 reject 携带豁免、不完整 provenance)在此拒绝,不触库。
func (a HumanApprovalEvidence) validatedForStorage() (HumanApprovalEvidence, error) {
	spec := ApprovalSpec{
		Project:   a.Project(),
		TaskID:    a.TaskID(),
		Decision:  a.Decision(),
		Reason:    a.Reason(),
		DecidedBy: a.DecidedBy(),
		DecidedAt: a.DecidedAt(),
		Waiver:    a.Waiver(),
	}
	if cs, ok := a.ChangeSetID(); ok {
		spec.ChangeSetID = cs
	}
	validated, err := NewApproval(spec)
	if err != nil {
		return HumanApprovalEvidence{}, err
	}
	return validated, nil
}

func approvalsFromRows(rows []store.HumanApproval) ([]RecordedApproval, error) {
	out := make([]RecordedApproval, 0, len(rows))
	for _, row := range rows {
		a, err := approvalFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

func approvalFromRow(row store.HumanApproval) (RecordedApproval, error) {
	// 行契约矛盾:waived=false 却带豁免 reason code(适配器写路径不可能产生,
	// 出现即行被破坏)—— 不把矛盾值当真。
	if !row.WaivedIndependence && row.WaiverReasonCode.Valid {
		return RecordedApproval{}, fmt.Errorf("approval: 行 %d 矛盾:waived_independence=false 却带 waiver_reason_code %q", row.ID, row.WaiverReasonCode.String)
	}
	spec := ApprovalSpec{
		Project:   core.ProjectID(row.ProjectID),
		TaskID:    row.TaskID,
		Decision:  Decision(row.Decision),
		Reason:    row.Reason,
		DecidedBy: row.DecidedBy,
		DecidedAt: tsVal(row.DecidedAt),
	}
	if row.ChangesetID.Valid {
		spec.ChangeSetID = row.ChangesetID.String
	}
	if row.WaivedIndependence {
		w, err := waiverFromRefs(row.WaiverRefs, textVal(row.WaiverReasonCode))
		if err != nil {
			return RecordedApproval{}, fmt.Errorf("approval: 重建行 %d 的豁免 provenance: %w", row.ID, err)
		}
		spec.Waiver = w
	}
	a, err := NewApproval(spec)
	if err != nil {
		return RecordedApproval{}, fmt.Errorf("approval: 重建行 %d 的决定证据: %w", row.ID, err)
	}
	return RecordedApproval{ID: row.ID, Approval: a}, nil
}

// ---- waiver_refs 的 JSONB 编码(JSON 数组,自描述条目,写入顺序保留) ----

// waiverRefEntry 是 waiver_refs 数组的一个条目。条目集合覆盖 NewWaiver 的
// 全部 provenance 输入,读取时据此重建并通过 NewWaiver 重校验;未知 kind
// 视为行破坏,拒绝而非跳过(provenance 静默丢失比报错更危险)。
type waiverRefEntry struct {
	Kind         string `json:"kind"`
	Value        string `json:"value,omitempty"`
	AcceptanceID string `json:"acceptance,omitempty"`
	Planned      bool   `json:"planned,omitempty"`
	Project      string `json:"project,omitempty"`
	Verification string `json:"verification,omitempty"`
}

const (
	waiverRefCategory              = "category"
	waiverRefGap                   = "gap"
	waiverRefCompromisedAcceptance = "compromised_acceptance"
	waiverRefCompromisedBinding    = "compromised_binding"
	waiverRefExposureRef           = "exposure_ref"
	waiverRefAcceptedBy            = "accepted_by"
	waiverRefAcceptedAt            = "accepted_at"
)

// waiverRefsJSON 把豁免 provenance 编码为 waiver_refs JSON 数组;nil(无豁免)
// 编码为 []。字段顺序固定,同一豁免的编码逐字节确定。
func waiverRefsJSON(w *IndependenceWaiver) ([]byte, error) {
	if w == nil {
		return []byte("[]"), nil
	}
	entries := []waiverRefEntry{{Kind: waiverRefCategory, Value: string(w.Category())}}
	for _, g := range w.Gaps() {
		entries = append(entries, waiverRefEntry{Kind: waiverRefGap, AcceptanceID: g.AcceptanceID, Planned: g.Planned})
	}
	for _, acc := range w.CompromisedAcceptances() {
		entries = append(entries, waiverRefEntry{Kind: waiverRefCompromisedAcceptance, Value: acc})
	}
	for _, b := range w.CompromisedBindings() {
		entries = append(entries, waiverRefEntry{Kind: waiverRefCompromisedBinding, Project: string(b.Project), Verification: string(b.Verification), Value: b.BindingID})
	}
	for _, e := range w.ExposureRefs() {
		entries = append(entries, waiverRefEntry{Kind: waiverRefExposureRef, Value: string(e)})
	}
	entries = append(entries,
		waiverRefEntry{Kind: waiverRefAcceptedBy, Value: w.AcceptedBy()},
		waiverRefEntry{Kind: waiverRefAcceptedAt, Value: w.AcceptedAt().UTC().Format(time.RFC3339Nano)},
	)
	data, err := json.Marshal(entries)
	if err != nil {
		return nil, fmt.Errorf("approval: 编码 waiver_refs: %w", err)
	}
	return data, nil
}

// waiverFromRefs 从 waiver_refs 行值 + waiver_reason_code 列重建并重校验
// 豁免 provenance(reason code 单独成列,是 provenance 的一部分)。
// 空数组/非法 JSON/未知条目/缺失 reason code 都是行破坏:豁免行必须携带
// 完整 provenance。
func waiverFromRefs(data []byte, reasonCode string) (*IndependenceWaiver, error) {
	var entries []waiverRefEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("approval: 解码 waiver_refs: %w", err)
	}
	spec := WaiverSpec{ReasonCode: reasonCode}
	for _, e := range entries {
		switch e.Kind {
		case waiverRefCategory:
			spec.Category = RiskCategory(e.Value)
		case waiverRefGap:
			spec.Gaps = append(spec.Gaps, CoverageGap{AcceptanceID: e.AcceptanceID, Planned: e.Planned})
		case waiverRefCompromisedAcceptance:
			spec.CompromisedAcceptances = append(spec.CompromisedAcceptances, e.Value)
		case waiverRefCompromisedBinding:
			spec.CompromisedBindings = append(spec.CompromisedBindings, evidence.BindingRef{
				Project:      core.ProjectID(e.Project),
				Verification: evidence.VerificationRevision(e.Verification),
				BindingID:    e.Value,
			})
		case waiverRefExposureRef:
			spec.ExposureRefs = append(spec.ExposureRefs, evidence.ExposureID(e.Value))
		case waiverRefAcceptedBy:
			spec.AcceptedBy = e.Value
		case waiverRefAcceptedAt:
			at, err := time.Parse(time.RFC3339Nano, e.Value)
			if err != nil {
				return nil, fmt.Errorf("approval: waiver_refs accepted_at %q 非法: %w", e.Value, err)
			}
			spec.AcceptedAt = at
		default:
			return nil, fmt.Errorf("approval: waiver_refs 未知条目 kind %q", e.Kind)
		}
	}
	w, err := NewWaiver(spec)
	if err != nil {
		return nil, err
	}
	return w, nil
}

// ---- pgtype 换算(本包私有,与 runstore 的辅助同义) ----

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
