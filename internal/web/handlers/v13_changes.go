// v13_changes.go 是 v1.3 P1 界面的 Changes / Run Detail 页(P1-44,PRD §26.2
// Changes,§26.1 的 Changes 工作区子集)。
//
// 页面语义(AC2):
//   - immutable ChangeSet:身份与全部固定字段(base/candidate/patch 身份、
//     producer run、created_at、改动文件)从事实面重建,只读;
//   - VerificationRun / Evidence:候选上的全部历史验证 Run 与逐条证据事实
//     (含 validity —— 经裁定失效的证据如实标注,§8.8.1);
//   - Deterministic Hunk Inventory(P1-07):从 Git base..candidate diff 确定性
//     重建;diff 不可得时显式说明,绝不给一份静默残缺的清单(§19.2);
//   - functional + independence coverage(P1-19/P1-20):与冻结契约同 revision
//     的覆盖推导;契约身份对不上时显式标注 mismatch,不跨 revision 混算(§8.8.2)。
//
// 本文件同时持有 change/review 两页共用的候选上下文装载与覆盖推导。
package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/jackc/pgx/v5"

	"tally/internal/artifacts"
	"tally/internal/changeset"
	"tally/internal/core"
	"tally/internal/evidence"
	"tally/internal/review"
	"tally/internal/runstore"
	"tally/internal/taskcontract"
	"tally/internal/verification"
)

/* ---------- 候选上下文装载(change / review 共用) ---------- */

// v13ChangeCtx 是一个候选的全部只读上下文:快照本体、其验证 Run 与逐条证据
// (含裁定投影)、以及 HEAD 制品快照里对应的 TaskContract 与冻结 VerificationContract。
type v13ChangeCtx struct {
	cs   *changeset.ChangeSet
	runs []runstore.VerificationRun
	// evid 按 verification_run 行 id 分组的证据行(新到在前,保查询序)。
	evid map[int64][]runstore.Evidence
	// rulings 按 evidence 行 id 的裁定投影(validity 派生输入)。
	rulings map[int64][]evidence.Ruling
	// lineage 是候选的 change lineage:候选在前,取代祖先随后(最近在前,
	// P0-07 确定性展开)。independence 推导与暴露呈现共用同一条 lineage。
	lineage []evidence.ChangeSetID
	// exposures 是该候选 lineage(候选 + 全部取代祖先)上的 exposure 投影
	// (independence 推导输入;当前/祖先由 AncestorDepth 区分,P0-07)。
	exposures []runstore.LineageExposure

	// git 提供确定性 diff(base..candidate);打开失败时为 nil(显式降级)。
	git *v13Git

	task   *taskcontract.TaskContract // HEAD 制品里匹配的 TaskContract(nil = 无)
	frozen *verification.Frozen       // HEAD 制品里匹配的冻结契约(nil = 无)

	// matchedTask 报告 frozen 与候选验证 Run 是否指向同一冻结契约修订
	// (§8.8.2:跨 revision 的报告不混算)。无 Run 或无冻结契约时不匹配。
	matched  bool
	contract string // 候选验证 Run 记录的冻结契约哈希(无 Run 为空)
	taskID   string // 候选验证 Run 记录的任务(无 Run 为空)

	// producerTask 是产生候选的实现 Run 记录的任务(验证 Run 缺席时的
	// 归属回退;两者都缺席为空,决定入口拒绝编造坐标)。
	producerTask string
}

// loadV13Change 装载候选上下文。目标不存在返回 errV13NotFound;其余读取失败
// 原样返回(调用方页面级降级,不静默给半份上下文)。
func (s *Server) loadV13Change(ctx context.Context, env v13Env, csid changeset.ChangeSetID) (*v13ChangeCtx, error) {
	cs, err := env.runs.ChangeSet(ctx, csid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errV13NotFound
		}
		return nil, fmt.Errorf("读取 ChangeSet: %w", err)
	}
	project := cs.Project()

	runs, err := env.runs.VerificationRunsForChangeSet(ctx, project, csid)
	if err != nil {
		return nil, fmt.Errorf("读取验证 Run: %w", err)
	}
	evid := make(map[int64][]runstore.Evidence, len(runs))
	rulings := make(map[int64][]evidence.Ruling)
	for _, run := range runs {
		rows, err := env.runs.EvidenceForRun(ctx, project, run.ID)
		if err != nil {
			return nil, fmt.Errorf("读取 Run %d 的证据: %w", run.ID, err)
		}
		evid[run.ID] = rows
		for _, ev := range rows {
			rs, err := env.runs.EvidenceRulings(ctx, project, ev.ID)
			if err != nil {
				return nil, fmt.Errorf("读取证据 %s 的裁定: %w", ev.FactID, err)
			}
			rulings[ev.ID] = rs
		}
	}

	c := &v13ChangeCtx{cs: cs, runs: runs, evid: evid, rulings: rulings}
	if len(runs) > 0 {
		// VerificationRunsForChangeSet 新到在前;首行即最新观测坐标。
		c.contract = runs[0].ContractHash
		c.taskID = runs[0].Task
	}
	if producerID, perr := strconv.ParseInt(cs.ProducerRun(), 10, 64); perr == nil {
		producer, perr := env.runs.CodegenRun(ctx, producerID)
		switch {
		case perr == nil:
			c.producerTask = producer.Task
		case errors.Is(perr, pgx.ErrNoRows):
			// 产生它的 Run 行不存在:归属回退不到,保持为空(调用方显式处理)。
		default:
			return nil, fmt.Errorf("读取产生候选的实现 Run: %w", perr)
		}
	}
	// 取代 lineage 与其上的暴露投影(P0-07):候选 + 全部取代祖先。祖先暴露
	// 在当前候选干净时也不得消失(AC3);祖先展开失败是候选读取失败,页面
	// 显式报错,不静默退回 candidate-local 视图。
	ancestors, err := env.runs.ChangeSetAncestors(ctx, csid)
	if err != nil {
		return nil, fmt.Errorf("展开取代祖先: %w", err)
	}
	c.lineage = make([]evidence.ChangeSetID, 0, len(ancestors)+1)
	c.lineage = append(c.lineage, evidence.ChangeSetID(csid))
	for _, a := range ancestors {
		c.lineage = append(c.lineage, evidence.ChangeSetID(a))
	}
	exposures, err := env.runs.ExposuresForChangeSetLineage(ctx, project, csid)
	if err != nil {
		return nil, fmt.Errorf("读取 lineage exposure: %w", err)
	}
	c.exposures = exposures

	g, err := openV13Git(env.repoPath)
	if err == nil {
		c.git = g
		snap, err := artifacts.Load(env.repoPath, g.headHash)
		if err == nil {
			for i := range snap.Tasks {
				if snap.Tasks[i].Contract.ID == c.taskID {
					c.task = &snap.Tasks[i].Contract
					break
				}
			}
			if c.taskID != "" {
				for i := range snap.Verifications {
					if snap.Verifications[i].Frozen.TaskID() == c.taskID {
						c.frozen = snap.Verifications[i].Frozen
						break
					}
				}
			}
		}
		// 制品装载失败不是候选读取失败:task/frozen 保持 nil,页面显式说明。
	}
	c.matched = c.frozen != nil && c.contract != "" && c.frozen.Hash() == c.contract
	return c, nil
}

// v13EvidenceValid 报告一条证据当前是否 valid:存在任一 invalidating 裁定
// (holdout_defect / spec_defect / invalid_evidence,§8.8.1)即失效。
func v13EvidenceValid(rs []evidence.Ruling) bool {
	for _, r := range rs {
		if r.Kind.Invalidates() {
			return false
		}
	}
	return true
}

/* ---------- 覆盖推导(P1-19 / P1-20,仅用于与冻结契约同 revision 的证据) ---------- */

// v13FunctionalRecords 把与冻结契约同 revision 的证据行投影成功能覆盖输入。
// verdict=error 无可用判定(§8.8.3 同性质),不进入 pass/fail 推导 —— 页面在
// 证据表里原样展示 error 行,不静默丢弃。
func v13FunctionalRecords(
	c *v13ChangeCtx,
	frozen *verification.Frozen,
) []verification.EvidenceRecord {
	var records []verification.EvidenceRecord
	for _, run := range c.runs {
		if run.ContractHash != frozen.Hash() {
			continue // §8.8.2:其它 revision 的证据是历史事实,不参与当前推导
		}
		for _, ev := range c.evid[run.ID] {
			var status verification.EvidenceStatus
			switch ev.Verdict {
			case evidence.VerdictPass:
				status = verification.EvidenceStatusPass
			case evidence.VerdictFail:
				status = verification.EvidenceStatusFail
			default:
				continue // error:无可用判定,不进 pass/fail 推导
			}
			validity := verification.EvidenceValidityValid
			if !v13EvidenceValid(c.rulings[ev.ID]) {
				validity = verification.EvidenceValidityInvalidated
			}
			records = append(records, verification.EvidenceRecord{
				BindingID: ev.BindingID,
				Revision:  frozen.Revision(),
				Status:    status,
				Validity:  validity,
			})
		}
	}
	return records
}

// v13IndependenceInput 把证据与 exposure 事实投影成 effective independence
// 覆盖(P1-20)的求值输入。authoring provenance 的活动接触维度
// (CandidateBlind/AuthoredInLineage/ChangedInLineage)与实现权威 provenance
// (IndependenceContext.Implementation)尚无事实来源,保守取零值 —— 缺省的
// provenance 不构成独立证据(§8.5.2),推导自然落在不满足一侧,不制造假覆盖。
func v13IndependenceInput(
	c *v13ChangeCtx,
	frozen *verification.Frozen,
	project core.ProjectID,
) ([]verification.IndependenceEvidence, error) {
	ref := func(bindingID string) evidence.BindingRef {
		return evidence.BindingRef{
			Project:      project,
			Verification: evidence.VerificationRevision(frozen.Hash()),
			BindingID:    bindingID,
		}
	}
	// 同一 binding 的全部判定投影(按契约 binding 分组;derivation 自行按
	// revision/candidate 选择,这里不预筛 error/validity)。
	byBinding := map[string]*verification.IndependenceEvidence{}
	for _, run := range c.runs {
		if run.ContractHash != frozen.Hash() {
			continue
		}
		for _, ev := range c.evid[run.ID] {
			ie, ok := byBinding[ev.BindingID]
			if !ok {
				ie = &verification.IndependenceEvidence{
					BindingID: ev.BindingID,
					Revision:  frozen.Revision(),
				}
				byBinding[ev.BindingID] = ie
			}
			ie.Verdicts = append(ie.Verdicts, evidence.VerdictRecord{
				Binding:   ref(ev.BindingID),
				Task:      evidence.TaskID(c.taskID),
				ChangeSet: evidence.ChangeSetID(c.cs.ID()),
				Verdict:   ev.Verdict,
				Valid:     v13EvidenceValid(c.rulings[ev.ID]),
			})
		}
	}
	// Exposure facts 整体交给推导:它按 BindingRef 选择,其它 binding 的
	// exposure 永远不会跨 binding 泄漏(P1-18 的输入合同)。输入含全部取代
	// 祖先上的暴露(P0-07):lineage 单调,祖先泄漏不得被新候选的干净记录
	// 抹掉 —— 保证只能维持或降低 assurance(INV-V14-08)。
	for _, le := range c.exposures {
		e := le.Exposure
		b := e.Binding()
		ie, ok := byBinding[b.BindingID]
		if !ok {
			ie = &verification.IndependenceEvidence{
				BindingID: b.BindingID,
				Revision:  frozen.Revision(),
			}
			byBinding[b.BindingID] = ie
		}
		ie.Exposures = append(ie.Exposures, e)
	}

	ids := make([]string, 0, len(byBinding))
	for id := range byBinding {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]verification.IndependenceEvidence, 0, len(ids))
	for _, id := range ids {
		out = append(out, *byBinding[id])
	}
	return out, nil
}

// v13Coverage 是候选页/评审页共用的覆盖推导输出(与冻结契约同 revision)。
type v13Coverage struct {
	Functional   verification.FunctionalCoverageReport
	Independence verification.IndependenceCoverageReport
}

// deriveV13Coverage 对上下文做 functional + independence 覆盖推导。
// 上下文未匹配冻结契约时返回 nil(matched 由调用方先行判断)。
func deriveV13Coverage(c *v13ChangeCtx) (*v13Coverage, error) {
	if !c.matched || c.frozen == nil {
		return nil, nil
	}
	project := c.cs.Project()
	functional, err := verification.DeriveFunctionalCoverage(c.frozen, v13FunctionalRecords(c, c.frozen))
	if err != nil {
		return nil, fmt.Errorf("功能覆盖推导: %w", err)
	}
	indepInput, err := v13IndependenceInput(c, c.frozen, project)
	if err != nil {
		return nil, fmt.Errorf("independence 求值输入投影: %w", err)
	}
	independence, err := verification.DeriveIndependenceCoverage(c.frozen, verification.IndependenceContext{
		Project:          project,
		Verification:     evidence.VerificationRevision(c.frozen.Hash()),
		ChangeSet:        evidence.ChangeSetID(c.cs.ID()),
		CandidateFixedAt: c.cs.CreatedAt(),
		// 候选在前、取代祖先随后(P0-07):锚定祖先的 lineage 局部暴露经完整
		// lineage 命中当前候选,不再退化为 candidate-local。
		Lineage: c.lineage,
		// Implementation / Precommit 零值 = 保守推导(见 v13IndependenceInput)。
	}, indepInput)
	if err != nil {
		return nil, fmt.Errorf("independence 覆盖推导: %w", err)
	}
	return &v13Coverage{Functional: functional, Independence: independence}, nil
}

/* ---------- Changes / Run Detail 页 ---------- */

// V13ChangedFile 是快照内一个改动文件的展示行(immutable 内容,含 hunk 坐标)。
type V13ChangedFile struct {
	Path       string
	Executable bool
	Hunks      []changeset.Hunk
}

// V13EvidenceRow 是一条验证证据的展示行(validity 是裁定投影后的现状)。
type V13EvidenceRow struct {
	FactID      string
	BindingID   string
	Acceptance  string
	Visibility  string
	Verdict     string
	Recorder    string
	Valid       bool
	RecordedAt  string
	Invalidated string // 失效裁定的种类说明;valid 时为空
}

// V13ChangeRun 是一个验证 Run 及其证据的展示行。
type V13ChangeRun struct {
	ID               string
	Status           string
	ContractRevision int32
	ContractHash     string
	EnvName          string
	Runtime          string
	StartedAt        string
	FinishedAt       string
	Evidence         []V13EvidenceRow
}

// V13InvHunk 是确定性清单里一个 hunk 的展示行。
type V13InvHunk struct {
	ID    string
	Coord string // 「oldStart..+newStart」形态的落点摘要
	Body  []string
}

// V13InvFile 是确定性清单里一个改动文件的展示行。
type V13InvFile struct {
	Path    string
	Kind    string
	Content string
	Hunks   []V13InvHunk
}

// V13CoverageView 是覆盖推导的展示投影(逐条款行)。
type V13CoverageView struct {
	Available    bool
	Notice       string // 不可用/不匹配时的显式说明
	ContractRef  string
	Functional   []V13FuncRow
	Independence []V13IndepRow
}

// V13FuncRow 是一条验收条款的功能覆盖行。
type V13FuncRow struct {
	ID          string
	Statement   string
	Covered     bool
	Gap         string
	SatisfiedBy []string
	Blocking    []string
}

// V13IndepRow 是一条验收条款的 independence 覆盖行(与功能覆盖分开呈现,§8.8.3)。
type V13IndepRow struct {
	ID          string
	Required    string
	Covered     bool
	SatisfiedBy []string
}

// V13Supersede 是一条取代事实的展示行。
type V13Supersede struct {
	Old, New, Reason, At string
}

// V13ExposureRow 是 lineage 暴露投影的展示行(P0-07):当前候选与取代祖先
// 上的暴露分列呈现 —— 祖先暴露明确标注代数与锚点,修复(repair)不能掩盖
// 既有信息泄漏(AC2/AC3)。行是只读投影,不含任何可变暴露状态(AC4)。
type V13ExposureRow struct {
	ID         string
	BindingID  string
	Acceptance string
	Channel    string
	Scope      string // persistent(binding-wide)| lineage(lineage 局部)
	Reason     string
	ExposedAt  string
	Anchor     string // 锚定候选身份
	Depth      int    // 0 = 当前候选;N = 第 N 代取代祖先
	Current    bool
}

// v13ExposureRows 把 lineage 暴露投影成展示行(change / review 两页共用)。
func v13ExposureRows(exps []runstore.LineageExposure) []V13ExposureRow {
	rows := make([]V13ExposureRow, 0, len(exps))
	for _, le := range exps {
		e := le.Exposure
		scope := "lineage"
		if e.BindingWide() {
			scope = "persistent"
		}
		acc, _ := e.Acceptance()
		rows = append(rows, V13ExposureRow{
			ID:         string(e.ID()),
			BindingID:  e.Binding().BindingID,
			Acceptance: string(acc),
			Channel:    string(e.Channel()),
			Scope:      scope,
			Reason:     e.Reason(),
			ExposedAt:  v13FormatTime(e.ExposedAt()),
			Anchor:     string(le.ChangeSet),
			Depth:      le.AncestorDepth,
			Current:    le.AncestorDepth == 0,
		})
	}
	return rows
}

// v13AncestorExposureCount 统计锚定在取代祖先(≥1 代)上的暴露条数。
func v13AncestorExposureCount(exps []runstore.LineageExposure) int {
	n := 0
	for _, le := range exps {
		if le.AncestorDepth > 0 {
			n++
		}
	}
	return n
}

// V13ChangePage 是 v1.3 Changes / Run Detail 页数据。
type V13ChangePage struct {
	Page
	Unavailable bool
	NotFound    bool
	LoadErr     string

	ID          string
	Project     string
	Base        string
	Candidate   string
	PatchID     string
	ProducerRun string
	CreatedAt   string
	Entities    []string
	Superseded  []V13Supersede

	// Lineage 是候选 + 取代祖先的身份列表(候选在前,P0-07);Exposures 是
	// lineage 上的暴露投影,祖先暴露在当前候选干净时也不消失(AC2/AC3)。
	Lineage          []string
	Exposures        []V13ExposureRow
	AncestorExposure int

	Files []V13ChangedFile

	Runs []V13ChangeRun

	// Inventory 是确定性 hunk 清单;InventoryNotice 非空时说明清单为何缺席。
	Inventory       []V13InvFile
	FileCount       int
	HunkCount       int
	InventoryNotice string

	Coverage V13CoverageView
}

// V13Change 渲染候选 ChangeSet 详情:immutable 快照、验证 Run/证据、
// 确定性 hunk 清单、功能+independence 覆盖(AC2)。
func (s *Server) V13Change(w http.ResponseWriter, r *http.Request) {
	page := V13ChangePage{Page: s.page(r, "Change 详情", "v13"), Coverage: V13CoverageView{}}
	env, ok := s.v13env()
	if !ok {
		page.Unavailable = true
		v13Render(w, http.StatusOK, "v13_change", page)
		return
	}
	s.v13ChangePage(r, env, &page)
	v13Render(w, http.StatusOK, "v13_change", page)
}

func (s *Server) v13ChangePage(r *http.Request, env v13Env, page *V13ChangePage) {
	ctx := r.Context()
	c, err := s.loadV13Change(ctx, env, v13CSIDParam(r))
	if err != nil {
		if errors.Is(err, errV13NotFound) {
			page.NotFound = true
			return
		}
		page.LoadErr = err.Error()
		return
	}
	cs := c.cs
	page.ID = string(cs.ID())
	page.Project = string(cs.Project())
	page.Base = cs.BaseRevision()
	page.Candidate = cs.CandidateRevision()
	page.PatchID = cs.PatchIdentity()
	page.ProducerRun = cs.ProducerRun()
	page.CreatedAt = v13FormatTime(cs.CreatedAt())
	page.Entities = cs.Entities()

	for _, f := range cs.Files() {
		page.Files = append(page.Files, V13ChangedFile{Path: f.Path, Executable: f.Executable, Hunks: f.Hunks})
	}

	if facts, err := env.runs.ChangeSetSupersessions(ctx, cs.ID()); err == nil {
		for _, f := range facts {
			page.Superseded = append(page.Superseded, V13Supersede{
				Old: string(f.Old), New: string(f.New), Reason: string(f.Reason), At: v13FormatTime(f.At),
			})
		}
	}

	// Lineage 暴露投影(P0-07):当前候选 + 全部取代祖先,祖先行带代数与锚点。
	for _, id := range c.lineage {
		page.Lineage = append(page.Lineage, string(id))
	}
	page.Exposures = v13ExposureRows(c.exposures)
	page.AncestorExposure = v13AncestorExposureCount(c.exposures)

	for _, run := range c.runs {
		vr := V13ChangeRun{
			ID:               fmt.Sprintf("%d", run.ID),
			Status:           string(run.Status),
			ContractRevision: run.ContractRevision,
			ContractHash:     run.ContractHash,
			EnvName:          run.EnvName,
			Runtime:          run.Runtime,
			StartedAt:        v13FormatTime(run.StartedAt),
			FinishedAt:       v13FormatTime(run.FinishedAt),
		}
		for _, ev := range c.evid[run.ID] {
			row := V13EvidenceRow{
				FactID:     string(ev.FactID),
				BindingID:  ev.BindingID,
				Acceptance: string(ev.Scope.Acceptance),
				Visibility: string(ev.VisibilityPolicy),
				Verdict:    string(ev.Verdict),
				Recorder:   string(ev.Recorder.Kind) + "/" + ev.Recorder.Actor,
				Valid:      v13EvidenceValid(c.rulings[ev.ID]),
				RecordedAt: v13FormatTime(ev.RecordedAt),
			}
			if !row.Valid {
				for _, ru := range c.rulings[ev.ID] {
					if ru.Kind.Invalidates() {
						row.Invalidated = string(ru.Kind)
						break
					}
				}
			}
			vr.Evidence = append(vr.Evidence, row)
		}
		page.Runs = append(page.Runs, vr)
	}

	// 确定性 hunk 清单:候选有 candidate revision 时从 Git 重建;
	// 只有 patch 身份的候选没有可比对的 Git revision,显式说明(§19.2:
	// 不允许一份静默残缺的清单进入 review)。
	if c.git != nil && cs.CandidateRevision() != "" {
		diff, err := c.git.DiffText(cs.BaseRevision(), cs.CandidateRevision())
		switch {
		case err != nil:
			page.InventoryNotice = "无法从制品仓库重建 base..candidate diff:" + err.Error()
		default:
			inv, err := review.InventoryFromDiff(diff)
			if err != nil {
				page.InventoryNotice = "diff 无法解析为确定性清单:" + err.Error()
				break
			}
			page.FileCount = inv.FileCount()
			page.HunkCount = inv.HunkCount()
			for _, f := range inv.Files {
				vf := V13InvFile{Path: f.Path, Kind: string(f.Kind), Content: string(f.Content)}
				for _, h := range f.Hunks {
					vf.Hunks = append(vf.Hunks, V13InvHunk{
						ID:    string(h.ID),
						Coord: fmt.Sprintf("-%d,%d +%d,%d", h.OldStart, h.OldLines, h.NewStart, h.NewLines),
						Body:  h.Body,
					})
				}
				page.Inventory = append(page.Inventory, vf)
			}
		}
	} else {
		page.InventoryNotice = "该候选没有固定 candidate revision(仅 patch 身份),无法从 Git 重建确定性 hunk 清单。"
	}

	cov, err := deriveV13Coverage(c)
	if err != nil {
		page.Coverage.Notice = "覆盖推导失败:" + err.Error()
		return
	}
	page.Coverage = v13CoverageView(c, cov)
}

// v13CoverageView 把覆盖推导结果投影成展示行;不可用/不匹配时给显式 Notice。
func v13CoverageView(c *v13ChangeCtx, cov *v13Coverage) V13CoverageView {
	view := V13CoverageView{}
	switch {
	case c.frozen == nil:
		view.Notice = "HEAD 制品快照中没有该任务的冻结 VerificationContract,无法推导覆盖。"
		return view
	case !c.matched:
		view.Notice = fmt.Sprintf(
			"候选的验证记录绑定契约 %s,而 HEAD 制品快照冻结契约是 %s:跨 revision 不混算(§8.8.2),覆盖不可推导。",
			v13Short(c.contract), v13Short(c.frozen.Hash()))
		view.ContractRef = c.frozen.Hash()
		return view
	}
	view.Available = true
	view.ContractRef = c.frozen.Hash()
	for _, a := range cov.Functional.Acceptances {
		view.Functional = append(view.Functional, V13FuncRow{
			ID:          a.AcceptanceID,
			Covered:     a.Covered,
			Gap:         string(a.Gap),
			SatisfiedBy: a.SatisfiedBy,
			Blocking:    a.Blocking,
		})
	}
	for _, a := range cov.Independence.Acceptances {
		view.Independence = append(view.Independence, V13IndepRow{
			ID:          a.AcceptanceID,
			Required:    string(a.Required),
			Covered:     a.Covered,
			SatisfiedBy: a.SatisfiedBy,
		})
	}
	return view
}
