package runstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"tally/internal/changeset"
	"tally/internal/command"
	"tally/internal/core"
	"tally/internal/evidence"
	"tally/internal/store"
	"tally/internal/verification"
)

// Evidence / Exposure 持久化(migrations/0020;PRD §6.4.3、§8.6、§8.8、§8.13,
// INV-FACT-01 / INV-AUTH-01/02):
//
//   - verification_evidence 与 verification_evidence_ruling、evidence_exposure
//     全部只追加(语句级触发器拦截 UPDATE/DELETE):evidence 诞生即 valid,
//     失效只能追加带 provenance 的裁定行,任何行都不改写(AC1;合法 FAIL
//     不被后来的 PASS 抹掉,§8.8.1)。
//   - recorder_kind 刻意排除 implementation_agent:实现侧通道不能确认机器事实。
//     本适配器经 internal/evidence 的领域构造器做同一校验,拒绝在数据库约束
//     之前放行非法 verdict / 非法 recorder / 不完整 scope。
//   - applicability 由行上的精确 scope 引用(project/task/acceptance/contract
//     revision/ChangeSet)相对当前 desired 上下文判定;validity 与 applicability
//     是两条独立的轴(§8.8.1/8.8.2)。
//
// 读取时重建领域值(AC4):evidence 行联其 verification_run 行取得 scope 的
// Verification/ChangeSet 两维(Run 是 scope 的权威坐标,写入时不重复声明);
// exposure 的领域身份由数据库 serial 承载,读回时 ID = serial 十进制串。

// EvidenceRecord 是写入一条 evidence 事实的输入(§6.4.3/§14.3)。
// Project 走方法参数;scope 的 Verification/ChangeSet 两维取自 RunID 指向的
// verification_run 行。
type EvidenceRecord struct {
	// FactID 是证据事实身份,唯一且永不复用;重复记录=错误,不是更新。
	FactID evidence.FactID
	// RunID 是产生本证据的 verification_run 行 id。
	RunID      int64
	Task       string
	Acceptance string
	BindingID  string
	// VisibilityPolicy 是冻结契约策略在执行时的快照(§8.4.1)。
	VisibilityPolicy verification.VisibilityPolicy
	Verdict          evidence.Verdict
	// Recorder 是记录 provenance;kind 必须 fact-capable(INV-AUTH-01/02)。
	Recorder evidence.Recorder
	// ValidatorVersion 可空(非 validator 通道)。
	ValidatorVersion string
	// Command 是结构化 CommandSpec(canonical JSON:argv/cwd/env/timeout/shell,
	// 0016 合同),拒绝 shell 字符串;nil = 未记录。
	Command *command.CommandSpec
	// ExitCode nil = 未记录。
	ExitCode     *int32
	StdoutDigest string
	StderrDigest string
	// RawOutputRef 是完整输出在 Tally 控制存储中的引用;held_out 的原始失败
	// 输出不得进实现侧可见通道(INV-VER-05、§14.3)。
	RawOutputRef string
	// RecordedAt 为零值时取当下(UTC)。
	RecordedAt time.Time
}

// RecordEvidence 记一条 observed evidence 事实(result + provenance + 精确
// scope 引用),返回行 id(追加裁定时引用)。fact_id 重复(同一 FactID 二次
// 记录)是数据库唯一约束错误:事实身份重复是错误,不是更新(INV-FACT-01)。
func (s *Store) RecordEvidence(ctx context.Context, project core.ProjectID, rec EvidenceRecord) (int64, error) {
	switch {
	case project == "":
		return 0, errors.New("runstore: 缺少 project")
	case rec.FactID == "":
		return 0, errors.New("runstore: 缺少 fact_id")
	case rec.RunID <= 0:
		return 0, errors.New("runstore: 缺少 run_id")
	case rec.Task == "":
		return 0, errors.New("runstore: 缺少 task")
	case rec.Acceptance == "":
		return 0, errors.New("runstore: 缺少 acceptance_id")
	case rec.BindingID == "":
		return 0, errors.New("runstore: 缺少 binding_id")
	}
	switch rec.VisibilityPolicy {
	case verification.VisibilityVisible, verification.VisibilityHeldOut:
	default:
		return 0, fmt.Errorf("runstore: 非法 visibility_policy %q", rec.VisibilityPolicy)
	}
	if rec.Command != nil {
		if err := rec.Command.Validate(); err != nil {
			return 0, fmt.Errorf("runstore: evidence %s command: %w", rec.FactID, err)
		}
	}

	// Run 行是 scope 的权威坐标:run 不存在即报错(证据不能脱离 Run 存在)。
	run, err := s.VerificationRun(ctx, project, rec.RunID)
	if err != nil {
		return 0, err
	}
	// 领域构造器做 recorder/verdict/scope 完整性校验(与表 CHECK 同一约束面)。
	recordedAt := rec.RecordedAt
	if recordedAt.IsZero() {
		recordedAt = time.Now()
	}
	if _, err := evidence.NewFact(rec.FactID, scopeOf(project, rec, run), rec.Verdict, recordedAt, rec.Recorder); err != nil {
		return 0, err
	}

	var cmdJSON []byte
	if rec.Command != nil {
		if cmdJSON, err = json.Marshal(rec.Command); err != nil {
			return 0, fmt.Errorf("runstore: 编码 evidence %s command: %w", rec.FactID, err)
		}
	}
	id, err := s.q.InsertVerificationEvidence(ctx, store.InsertVerificationEvidenceParams{
		FactID:           string(rec.FactID),
		RunID:            rec.RunID,
		ProjectID:        string(project),
		TaskID:           rec.Task,
		AcceptanceID:     rec.Acceptance,
		BindingID:        rec.BindingID,
		VisibilityPolicy: string(rec.VisibilityPolicy),
		Verdict:          string(rec.Verdict),
		RecorderKind:     string(rec.Recorder.Kind),
		RecorderActor:    rec.Recorder.Actor,
		ValidatorVersion: textPtr(rec.ValidatorVersion),
		Command:          cmdJSON,
		ExitCode:         int4Ptr(rec.ExitCode),
		StdoutDigest:     textPtr(rec.StdoutDigest),
		StderrDigest:     textPtr(rec.StderrDigest),
		RawOutputRef:     textPtr(rec.RawOutputRef),
		RecordedAt:       tsPtr(recordedAt),
	})
	if err != nil {
		return 0, fmt.Errorf("runstore: 写入 evidence %s: %w", rec.FactID, err)
	}
	return id, nil
}

func scopeOf(project core.ProjectID, rec EvidenceRecord, run *VerificationRun) evidence.Scope {
	return evidence.Scope{
		Project:      project,
		Task:         evidence.TaskID(rec.Task),
		Acceptance:   evidence.AcceptanceID(rec.Acceptance),
		Verification: evidence.VerificationRevision(run.ContractHash),
		ChangeSet:    evidence.ChangeSetID(run.ChangeSetID),
	}
}

// Evidence 是一条 evidence 事实的读取模型(领域值已重建;AC4)。
type Evidence struct {
	// ID 是行 id(代理主键);追加裁定时引用。
	ID               int64
	FactID           evidence.FactID
	RunID            int64
	Scope            evidence.Scope
	BindingID        string
	VisibilityPolicy verification.VisibilityPolicy
	Verdict          evidence.Verdict
	Recorder         evidence.Recorder
	ValidatorVersion string
	Command          *command.CommandSpec // nil = 未记录
	ExitCode         *int32
	StdoutDigest     string
	StderrDigest     string
	RawOutputRef     string
	RecordedAt       time.Time
}

// Fact 重建纯领域 fact 值(不含裁定序列;validity 由 EvidenceRulings 派生)。
func (e Evidence) Fact() (evidence.Fact, error) {
	return evidence.NewFact(e.FactID, e.Scope, e.Verdict, e.RecordedAt, e.Recorder)
}

// Evidence 按 fact_id 读回单条证据并重建领域值(联 Run 行取完整 Scope)。
// 未知 fact_id 返回包着 pgx.ErrNoRows 的错误。
func (s *Store) Evidence(ctx context.Context, project core.ProjectID, factID evidence.FactID) (*Evidence, error) {
	row, err := s.q.GetVerificationEvidenceByFactID(ctx, store.GetVerificationEvidenceByFactIDParams{
		ProjectID: string(project),
		FactID:    string(factID),
	})
	if err != nil {
		return nil, fmt.Errorf("runstore: 查询 evidence %s: %w", factID, err)
	}
	run, err := s.VerificationRun(ctx, project, row.RunID)
	if err != nil {
		return nil, err
	}
	e, err := evidenceFromRow(row, run)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// EvidenceForRun 返回一次 Run 产生的全部 evidence(§6.4.3:Run → Evidence 一对多)。
func (s *Store) EvidenceForRun(ctx context.Context, project core.ProjectID, runID int64) ([]Evidence, error) {
	run, err := s.VerificationRun(ctx, project, runID)
	if err != nil {
		return nil, err
	}
	rows, err := s.q.ListVerificationEvidenceForRun(ctx, store.ListVerificationEvidenceForRunParams{
		RunID:     runID,
		ProjectID: string(project),
	})
	if err != nil {
		return nil, fmt.Errorf("runstore: 查询 run %d 的 evidence: %w", runID, err)
	}
	return evidenceFromRows(rows, run)
}

// ApplicableValidEvidence 返回当前 desired 上下文的 gate 输入(§8.8.1/8.8.2):
// evidence 必须精确落在 scope 五维上(applicability),且未被任何失效裁定命中
// (validity)。contractRevision 是 scope.Verification 哈希对应的冻结 revision 号
// (scope 只携带内容哈希,revision 号由调用方从冻结契约一并给出)。
func (s *Store) ApplicableValidEvidence(ctx context.Context, scope evidence.Scope, contractRevision int32) ([]Evidence, error) {
	rows, err := s.q.ListApplicableValidEvidence(ctx, store.ListApplicableValidEvidenceParams{
		ProjectID:        string(scope.Project),
		TaskID:           string(scope.Task),
		AcceptanceID:     string(scope.Acceptance),
		ContractRevision: contractRevision,
		ContractHash:     string(scope.Verification),
		ChangesetID:      string(scope.ChangeSet),
	})
	if err != nil {
		return nil, fmt.Errorf("runstore: 查询 applicable valid evidence: %w", err)
	}
	// 命中查询的行共享查询给出的精确 scope(该查询按五维过滤)。
	run := &VerificationRun{ContractHash: string(scope.Verification), ChangeSetID: changeset.ChangeSetID(scope.ChangeSet)}
	return evidenceFromRows(rows, run)
}

// AppendEvidenceRuling 追加一条失效裁定(append-only;不改写被裁定的 evidence
// 行,§8.8.1),返回裁定行 id。裁定通道本身必须 fact-capable:agent 提案不能
// 使证据失效。
func (s *Store) AppendEvidenceRuling(ctx context.Context, project core.ProjectID, evidenceID int64, r evidence.Ruling) (int64, error) {
	if project == "" {
		return 0, errors.New("runstore: 缺少 project")
	}
	if evidenceID <= 0 {
		return 0, errors.New("runstore: 缺少 evidence_id")
	}
	// 领域构造器做 kind/reason/recorder 校验并归一时间。
	validated, err := evidence.NewRuling(r.Kind, r.Reason, r.RuledAt, r.Recorder)
	if err != nil {
		return 0, err
	}
	id, err := s.q.InsertVerificationEvidenceRuling(ctx, store.InsertVerificationEvidenceRulingParams{
		EvidenceID:    evidenceID,
		ProjectID:     string(project),
		Kind:          string(validated.Kind),
		Reason:        validated.Reason,
		RecorderKind:  string(validated.Recorder.Kind),
		RecorderActor: validated.Recorder.Actor,
		RuledAt:       tsPtr(validated.RuledAt),
	})
	if err != nil {
		return 0, fmt.Errorf("runstore: 追加 evidence %d 的裁定: %w", evidenceID, err)
	}
	return id, nil
}

// EvidenceRulings 返回某条 evidence 的裁定序列(append-only,旧到新;
// validity 由此派生:存在失效裁定 = 已失效)。
func (s *Store) EvidenceRulings(ctx context.Context, project core.ProjectID, evidenceID int64) ([]evidence.Ruling, error) {
	rows, err := s.q.ListVerificationEvidenceRulings(ctx, store.ListVerificationEvidenceRulingsParams{
		EvidenceID: evidenceID,
		ProjectID:  string(project),
	})
	if err != nil {
		return nil, fmt.Errorf("runstore: 查询 evidence %d 的裁定: %w", evidenceID, err)
	}
	out := make([]evidence.Ruling, 0, len(rows))
	for _, row := range rows {
		r, err := evidence.NewRuling(
			evidence.RulingKind(row.Kind),
			row.Reason,
			tsVal(row.RuledAt),
			evidence.Recorder{Kind: evidence.RecorderKind(row.RecorderKind), Actor: row.RecorderActor},
		)
		if err != nil {
			return nil, fmt.Errorf("runstore: 重建 evidence %d 的裁定 %d: %w", evidenceID, row.ID, err)
		}
		out = append(out, r)
	}
	return out, nil
}

func evidenceFromRows(rows []store.VerificationEvidence, run *VerificationRun) ([]Evidence, error) {
	out := make([]Evidence, 0, len(rows))
	for _, row := range rows {
		e, err := evidenceFromRow(row, run)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

func evidenceFromRow(row store.VerificationEvidence, run *VerificationRun) (Evidence, error) {
	scope := evidence.Scope{
		Project:      core.ProjectID(row.ProjectID),
		Task:         evidence.TaskID(row.TaskID),
		Acceptance:   evidence.AcceptanceID(row.AcceptanceID),
		Verification: evidence.VerificationRevision(run.ContractHash),
		ChangeSet:    evidence.ChangeSetID(run.ChangeSetID),
	}
	visibility := verification.VisibilityPolicy(row.VisibilityPolicy)
	switch visibility {
	case verification.VisibilityVisible, verification.VisibilityHeldOut:
	default:
		return Evidence{}, fmt.Errorf("runstore: evidence %s 的 visibility_policy 非法: %q", row.FactID, row.VisibilityPolicy)
	}
	var cmd *command.CommandSpec
	if len(row.Command) > 0 && !isJSONNull(row.Command) {
		cmd = &command.CommandSpec{}
		if err := json.Unmarshal(row.Command, cmd); err != nil {
			return Evidence{}, fmt.Errorf("runstore: 解码 evidence %s command: %w", row.FactID, err)
		}
	}
	e := Evidence{
		ID:               row.ID,
		FactID:           evidence.FactID(row.FactID),
		RunID:            row.RunID,
		Scope:            scope,
		BindingID:        row.BindingID,
		VisibilityPolicy: visibility,
		Verdict:          evidence.Verdict(row.Verdict),
		Recorder:         evidence.Recorder{Kind: evidence.RecorderKind(row.RecorderKind), Actor: row.RecorderActor},
		ValidatorVersion: textVal(row.ValidatorVersion),
		Command:          cmd,
		ExitCode:         int4Val(row.ExitCode),
		StdoutDigest:     textVal(row.StdoutDigest),
		StderrDigest:     textVal(row.StderrDigest),
		RawOutputRef:     textVal(row.RawOutputRef),
		RecordedAt:       tsVal(row.RecordedAt),
	}
	// 重建即验证:非法 verdict / 非法 recorder / 不完整 scope 无法从库里读回来
	// (行被破坏时报错,不把矛盾值当真,与 ChangeSet 重建同一策略)。
	if _, err := e.Fact(); err != nil {
		return Evidence{}, fmt.Errorf("runstore: 重建 evidence %s: %w", row.FactID, err)
	}
	return e, nil
}

func int4Ptr(p *int32) pgtype.Int4 {
	if p == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: *p, Valid: true}
}

// ---- Exposure(PRD §8.6,AC1:append-only 暴露事实) ----

// ExposureRecord 是写入一条暴露事实的输入。exposure 的领域身份由数据库 serial
// 承载:写入不带自选 ID(重复身份是错误不是更新,身份在落库前不存在),
// 返回值是带已分配身份的领域 Exposure。detail 列只放引用/坐标类上下文
// (本适配器仅编码 Reason),不得携带 held-out 内容或可识别 verdict(INV-VER-05)。
type ExposureRecord struct {
	Task string // 暴露所属任务(表列必填;BindingRef 刻意不含 task,重挂任务不重置暴露账)
	// ContractRevision 是与 Binding().Verification 哈希同一冻结契约的 revision 号。
	ContractRevision int32
	Binding          evidence.BindingRef
	// Acceptance 可空:binding 所属验收条款。
	Acceptance evidence.AcceptanceID
	// ChangeSet 可空:内容暴露可能先于任何具体 candidate 被观测(P1-12 合同)。
	ChangeSet evidence.ChangeSetID
	Channel   evidence.ExposureChannel
	Reason    string // 可空追溯 provenance
	// ExposedAt 为零值时取当下(UTC)。
	ExposedAt time.Time
}

// RecordExposure 追加一条暴露事实,返回带已分配身份的领域 Exposure。
func (s *Store) RecordExposure(ctx context.Context, rec ExposureRecord) (evidence.Exposure, error) {
	switch {
	case rec.Task == "":
		return evidence.Exposure{}, errors.New("runstore: 缺少 task")
	case rec.ContractRevision < 1:
		return evidence.Exposure{}, fmt.Errorf("runstore: contract_revision 必须 >= 1(当前 %d)", rec.ContractRevision)
	case rec.Binding.Project == "" || rec.Binding.Verification == "" || rec.Binding.BindingID == "":
		return evidence.Exposure{}, fmt.Errorf("%w: binding ref 不完整", evidence.ErrInvalidExposure)
	case rec.Channel == "":
		return evidence.Exposure{}, fmt.Errorf("%w: channel 为空", evidence.ErrInvalidExposure)
	}
	exposedAt := rec.ExposedAt
	if exposedAt.IsZero() {
		exposedAt = time.Now()
	}
	detail, err := encodeExposureDetail(rec.Reason)
	if err != nil {
		return evidence.Exposure{}, err
	}
	id, err := s.q.InsertEvidenceExposure(ctx, store.InsertEvidenceExposureParams{
		ProjectID:        string(rec.Binding.Project),
		TaskID:           rec.Task,
		ContractRevision: rec.ContractRevision,
		ContractHash:     string(rec.Binding.Verification),
		BindingID:        rec.Binding.BindingID,
		AcceptanceID:     textPtr(string(rec.Acceptance)),
		ChangesetID:      textPtr(string(rec.ChangeSet)),
		Channel:          string(rec.Channel),
		Detail:           detail,
		ExposedAt:        tsPtr(exposedAt),
	})
	if err != nil {
		return evidence.Exposure{}, fmt.Errorf("runstore: 追加暴露事实(binding=%s channel=%s): %w", rec.Binding.BindingID, rec.Channel, err)
	}
	return reconstructExposure(id, rec.Binding, rec.Acceptance, rec.ChangeSet, rec.Channel, rec.Reason, exposedAt)
}

// ExposuresForTask 返回某任务的暴露事实时间线(Review 面按任务展开)。
func (s *Store) ExposuresForTask(ctx context.Context, project core.ProjectID, task string) ([]evidence.Exposure, error) {
	rows, err := s.q.ListEvidenceExposureForTask(ctx, store.ListEvidenceExposureForTaskParams{
		ProjectID: string(project),
		TaskID:    task,
	})
	if err != nil {
		return nil, fmt.Errorf("runstore: 查询 task %s/%s 的暴露事实: %w", project, task, err)
	}
	return exposuresFromRows(rows)
}

// ExposuresForBinding 返回某条 held-out binding 的全部暴露事实
// (§8.10:Waiver 需引用 compromised bindings 的 exposure refs)。
func (s *Store) ExposuresForBinding(ctx context.Context, project core.ProjectID, contractHash, bindingID string) ([]evidence.Exposure, error) {
	rows, err := s.q.ListEvidenceExposureForBinding(ctx, store.ListEvidenceExposureForBindingParams{
		ProjectID:    string(project),
		ContractHash: contractHash,
		BindingID:    bindingID,
	})
	if err != nil {
		return nil, fmt.Errorf("runstore: 查询 binding %s 的暴露事实: %w", bindingID, err)
	}
	return exposuresFromRows(rows)
}

// ExposuresForChangeSet 返回某候选 lineage 上的暴露事实
// (selection/content 污染判定,§8.6.2)。
func (s *Store) ExposuresForChangeSet(ctx context.Context, project core.ProjectID, cs changeset.ChangeSetID) ([]evidence.Exposure, error) {
	rows, err := s.q.ListEvidenceExposureForChangeSet(ctx, store.ListEvidenceExposureForChangeSetParams{
		ProjectID:   string(project),
		ChangesetID: textPtr(string(cs)),
	})
	if err != nil {
		return nil, fmt.Errorf("runstore: 查询 changeset %s 的暴露事实: %w", cs, err)
	}
	return exposuresFromRows(rows)
}

// ---- 取代 lineage 暴露投影(P0-07,PRD §29 item 8 / INV-V14-08) ----------
//
// 取代事实(evidence 侧只读消费 changeset_superseded_fact)构成候选的祖先
// 图:old 被 new 取代,重做产生新快照 + 新事实。暴露事实只追加、永不改写;
// 替换候选不得重置暴露账 —— 锚定祖先的暴露必须仍在当前候选的读取面里
// (祖先暴露在当前候选干净时也不消失,AC3)。本节只新增读取投影,不新增
// 任何可变暴露状态(AC4)。

// LineageExposure 是一条暴露事实在候选 lineage 投影中的定位:事实本体 +
// 它锚定的候选 + 该候选相对被求值候选的代数。定位是读取时派生的坐标,
// 不落库:暴露事实本身始终 append-only,祖先身份由取代事实派生。
type LineageExposure struct {
	Exposure evidence.Exposure
	// ChangeSet 是本条暴露锚定的候选(lineage 投影只含锚定事实)。
	ChangeSet changeset.ChangeSetID
	// AncestorDepth:0 = 锚定在被求值候选本身;N = 锚定在其第 N 代取代祖先
	// (1 = 被 directly 取代的快照)。
	AncestorDepth int
}

// ChangeSetAncestors 确定性展开候选的取代祖先(P0-07 AC1):被该候选直接
// 取代(old→new 的 new 侧命中)的候选为第 1 层,逐层向外;同层按候选身份
// 排序,层间由近到远 —— 同一事实集合恒得同一输出,与行返回顺序无关。
// 取代事实是 append-only 的图而不是树(同 old 可被多次取代、多父、乃至
// 环),visited 去重保证遍历有限;old 侧方向(该候选被谁取代)是后代方向,
// 不参与祖先展开。纯计算不触库时用 walkSupersessionAncestors。
func (s *Store) ChangeSetAncestors(ctx context.Context, cs changeset.ChangeSetID) ([]changeset.ChangeSetID, error) {
	return walkSupersessionAncestors(cs, func(id changeset.ChangeSetID) ([]changeset.SupersededFact, error) {
		return s.ChangeSetSupersessions(ctx, id)
	})
}

// walkSupersessionAncestors 是 ChangeSetAncestors 的纯遍历核:facts 返回某
// 候选参与的全部取代事实(与 ChangeSetSupersessions 同合同,只消费
// f.New == 查询点 的边)。层内排序是确定性的唯一来源。
func walkSupersessionAncestors(cs changeset.ChangeSetID, facts func(changeset.ChangeSetID) ([]changeset.SupersededFact, error)) ([]changeset.ChangeSetID, error) {
	if cs == "" {
		return nil, errors.New("runstore: 缺少 changeset id,无法展开取代祖先")
	}
	visited := map[changeset.ChangeSetID]bool{cs: true}
	var out []changeset.ChangeSetID
	frontier := []changeset.ChangeSetID{cs}
	for len(frontier) > 0 {
		var next []changeset.ChangeSetID
		for _, node := range frontier {
			fs, err := facts(node)
			if err != nil {
				return nil, err
			}
			for _, f := range fs {
				if f.New != node || visited[f.Old] {
					continue
				}
				visited[f.Old] = true
				next = append(next, f.Old)
			}
		}
		sort.Slice(next, func(i, j int) bool { return next[i] < next[j] })
		out = append(out, next...)
		frontier = next
	}
	return out, nil
}

// ExposuresForChangeSetLineage 投影候选 lineage 上的暴露事实(P0-07 AC3):
// 当前候选(AncestorDepth=0)+ 全部取代祖先上的暴露,输出顺序确定 ——
// 当前候选在前,祖先按代数由近到远,同锚点内保持 ExposuresForChangeSet
// 的时间序。祖先暴露在当前候选自身干净(候选局部查询为空)时也不消失:
// 读取侧的「干净」只能由 lineage 投影定义,取代不重置暴露账。未锚定暴露
// 没有 lineage 坐标,不在本投影(按 task / binding 投影另行读取)。
func (s *Store) ExposuresForChangeSetLineage(ctx context.Context, project core.ProjectID, cs changeset.ChangeSetID) ([]LineageExposure, error) {
	if project == "" {
		return nil, errors.New("runstore: 缺少 project")
	}
	var out []LineageExposure
	appendAnchor := func(anchor changeset.ChangeSetID, depth int) error {
		rows, err := s.ExposuresForChangeSet(ctx, project, anchor)
		if err != nil {
			return err
		}
		for _, e := range rows {
			out = append(out, LineageExposure{Exposure: e, ChangeSet: anchor, AncestorDepth: depth})
		}
		return nil
	}
	if err := appendAnchor(cs, 0); err != nil {
		return nil, err
	}
	ancestors, err := s.ChangeSetAncestors(ctx, cs)
	if err != nil {
		return nil, err
	}
	for i, ancestor := range ancestors {
		if err := appendAnchor(ancestor, i+1); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func exposuresFromRows(rows []store.EvidenceExposure) ([]evidence.Exposure, error) {
	out := make([]evidence.Exposure, 0, len(rows))
	for _, row := range rows {
		e, err := exposureFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

func exposureFromRow(row store.EvidenceExposure) (evidence.Exposure, error) {
	reason, err := decodeExposureDetail(row.Detail)
	if err != nil {
		return evidence.Exposure{}, fmt.Errorf("runstore: 解码暴露事实 %d detail: %w", row.ID, err)
	}
	binding := evidence.BindingRef{
		Project:      core.ProjectID(row.ProjectID),
		Verification: evidence.VerificationRevision(row.ContractHash),
		BindingID:    row.BindingID,
	}
	return reconstructExposure(
		row.ID,
		binding,
		evidence.AcceptanceID(textVal(row.AcceptanceID)),
		evidence.ChangeSetID(textVal(row.ChangesetID)),
		evidence.ExposureChannel(row.Channel),
		reason,
		tsVal(row.ExposedAt),
	)
}

func reconstructExposure(id int64, binding evidence.BindingRef, acceptance evidence.AcceptanceID, cs evidence.ChangeSetID, channel evidence.ExposureChannel, reason string, at time.Time) (evidence.Exposure, error) {
	e, err := evidence.NewExposure(evidence.ExposureSpec{
		ID:         evidence.ExposureID(strconv.FormatInt(id, 10)),
		Binding:    binding,
		Acceptance: acceptance,
		ChangeSet:  cs,
		Channel:    channel,
		Reason:     reason,
		ExposedAt:  at,
	})
	if err != nil {
		return evidence.Exposure{}, fmt.Errorf("runstore: 重建暴露事实 %d: %w", id, err)
	}
	return e, nil
}

// exposureDetail 是 detail JSONB 列的编码形状(对象;CHECK 拒绝非对象)。
type exposureDetail struct {
	Reason string `json:"reason,omitempty"`
}

func encodeExposureDetail(reason string) ([]byte, error) {
	if reason == "" {
		return nil, nil // JSONB NULL = 无 detail
	}
	data, err := json.Marshal(exposureDetail{Reason: reason})
	if err != nil {
		return nil, fmt.Errorf("runstore: 编码暴露 detail: %w", err)
	}
	return data, nil
}

func decodeExposureDetail(data []byte) (string, error) {
	if len(data) == 0 || isJSONNull(data) {
		return "", nil
	}
	var d exposureDetail
	if err := json.Unmarshal(data, &d); err != nil {
		return "", err
	}
	return d.Reason, nil
}
