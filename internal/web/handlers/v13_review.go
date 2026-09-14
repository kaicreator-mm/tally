// v13_review.go 是 v1.3 P1 界面的 Patch Review 页与 Owner 决定入口
// (P1-44,PRD §19 / §26.3)。
//
// 语义边界(全部由既有领域包承载,本文件不发明规则):
//   - Review 输入是固定 revision 的 ChangeSet + 确定性 Hunk Inventory(P1-07)
//   - 确定性归账(review.BuildAccounting,P1 评审内核);规则只归账,
//     「这段代码服务哪个 AC」的残差在 P1 UI 没有归属输入通道,如实落在
//     Unattributed(§19.2:Unattributed 非空 ⇒ 不得 Approve,AC3)。
//   - Owner 决定只写 HumanApprovalEvidence(append-only 证据,§18.1/§19),
//     不写任何任务状态字段(AC4);批准被阻断时服务端拒绝落证据,
//     与页面按钮禁用是同一判定(review.ApproveBlocked)。
//   - 不提供 Independence Waiver 投影:Project Verification Policy 的投影
//     尚无事实来源,WaiverPolicy 保守取零值(不允许豁免),缺口如实阻断。
package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"tally/internal/approval"
	"tally/internal/policygovernance"
	"tally/internal/review"
)

/* ---------- Patch Review 页(§19 / §26.3) ---------- */

// V13ReviewEntry 是一个记账条目(hunk 或零 hunk 文件条目)的展示行。
type V13ReviewEntry struct {
	HunkID       string
	Path         string
	Kind         string
	Content      string
	Bucket       string
	Tags         []string
	ByRule       bool
	Unattributed bool
}

// V13AcceptanceReview 是一条验收条款的 review 视图(§19.3 Acceptance Coverage)。
type V13AcceptanceReview struct {
	ID                      string
	Statement               string
	RelatedHunks            []string
	FunctionalCovered       bool
	FunctionalGap           string
	FunctionalSatisfiedBy   []string
	FunctionalBlocking      []string
	IndependenceRequired    string
	IndependenceCovered     bool
	IndependenceSatisfiedBy []string
	Strict                  string
}

// V13ProtectedTouch 是被候选触及的一个 protected evidence 文件(§19.3)。
type V13ProtectedTouch struct {
	Path         string
	BindingID    string
	AcceptanceID string
	ExpectedHash string
}

// V13Blocker 是一个 approve 阻断项的展示行。
type V13Blocker struct {
	Kind     string
	Describe string
}

// V13Report 是确定性归账报告的展示投影。
type V13Report struct {
	Available         bool
	Notice            string // 报告不可推导时的显式说明
	Entries           []V13ReviewEntry
	UnattributedCount int
	Acceptances       []V13AcceptanceReview
	Protected         []V13ProtectedTouch
	Blockers          []V13Blocker
	ApproveBlocked    bool
	StrictOutcome     string
}

// V13ReviewPage 是 Patch Review 页数据。
type V13ReviewPage struct {
	Page
	Unavailable bool
	NotFound    bool
	LoadErr     string

	ID        string
	Project   string
	TaskID    string
	Base      string
	Candidate string

	Report V13Report

	// Exposures 是候选 lineage(当前 + 全部取代祖先)上的暴露投影(P0-07):
	// 评审输入必须看见祖先暴露 —— 当前候选自身干净不能让历史泄漏消失(AC2/AC3)。
	// 与归账报告独立呈现:报告不可用时暴露事实照样可见。
	Exposures        []V13ExposureRow
	AncestorExposure int

	// CanDecide 报告页面是否呈现可用的 Owner 决定表单;false 时按钮禁用,
	// 且服务端同判定拒绝落证据(两层一致)。
	CanDecide     bool
	DecideBlocker string

	// GovernanceAvailable 报告「提议新规」表单是否可用(P3-08):治理存储
	// 未接入时表单按未接入呈现,仅此一次决定不受影响。
	GovernanceAvailable bool
}

// V13Review 渲染 Patch Review 页(确定性归账 + approve 阻断呈现)。
func (s *Server) V13Review(w http.ResponseWriter, r *http.Request) {
	page := V13ReviewPage{Page: s.page(r, "Patch Review", "v13")}
	env, ok := s.v13env()
	if !ok {
		page.Unavailable = true
		v13Render(w, http.StatusOK, "v13_review", page)
		return
	}
	s.v13ReviewPage(r, env, &page)
	v13Render(w, http.StatusOK, "v13_review", page)
}

func (s *Server) v13ReviewPage(r *http.Request, env v13Env, page *V13ReviewPage) {
	ctx := r.Context()
	page.GovernanceAvailable = s.Governance != nil
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
	page.TaskID = c.taskID
	page.Base = cs.BaseRevision()
	page.Candidate = cs.CandidateRevision()

	// Lineage 暴露投影(P0-07):先于归账装配填充 —— 报告不可用(无契约/
	// 清单失败)时祖先暴露也必须可见,不随报告一起缺席。
	page.Exposures = v13ExposureRows(c.exposures)
	page.AncestorExposure = v13AncestorExposureCount(c.exposures)

	// 归账输入逐项装配;任一缺席都让报告显式不可用(不静默残缺)。
	if c.task == nil {
		page.Report.Notice = "HEAD 制品快照中没有该任务的 TaskContract,无法建立归账规则(scope/契约 evidence 路径)。"
		return
	}
	if c.frozen == nil || !c.matched {
		page.Report.Notice = "候选验证记录与 HEAD 冻结契约不属于同一修订(§8.8.2),不能以当前契约做验收归账。"
		return
	}
	if c.git == nil || cs.CandidateRevision() == "" {
		page.Report.Notice = "确定性 hunk 清单不可得(无 candidate revision 或制品仓库不可读),§19.2 要求无完整清单不得 Approve。"
		page.Report.ApproveBlocked = true
		page.DecideBlocker = page.Report.Notice
		return
	}
	diff, err := c.git.DiffText(cs.BaseRevision(), cs.CandidateRevision())
	if err != nil {
		page.Report.Notice = "无法重建 base..candidate diff:" + err.Error()
		page.Report.ApproveBlocked = true
		page.DecideBlocker = page.Report.Notice
		return
	}
	inv, err := review.InventoryFromDiff(diff)
	if err != nil {
		page.Report.Notice = "diff 无法解析为确定性清单:" + err.Error()
		page.Report.ApproveBlocked = true
		page.DecideBlocker = page.Report.Notice
		return
	}
	rules, err := review.RulesFor(c.task.Scope, c.frozen)
	if err != nil {
		page.Report.Notice = "归账规则集构建失败:" + err.Error()
		page.Report.ApproveBlocked = true
		page.DecideBlocker = page.Report.Notice
		return
	}
	cov, err := deriveV13Coverage(c)
	if err != nil || cov == nil {
		page.Report.Notice = "覆盖推导不可用,无法做验收归账。"
		page.Report.ApproveBlocked = true
		page.DecideBlocker = page.Report.Notice
		return
	}

	// WaiverPolicy 零值 = 不允许豁免(保守默认);Attributions 为空:P1 UI
	// 没有残差归属输入通道,规则之外的条目如实落入 Unattributed。
	rep, err := review.BuildAccounting(review.AccountingInput{
		ChangeSet:    cs,
		Inventory:    inv,
		Rules:        rules,
		Frozen:       c.frozen,
		Functional:   cov.Functional,
		Independence: cov.Independence,
	})
	if err != nil {
		page.Report.Notice = "确定性归账失败:" + err.Error()
		page.Report.ApproveBlocked = true
		page.DecideBlocker = page.Report.Notice
		return
	}

	page.Report.Available = true
	page.Report.StrictOutcome = string(rep.StrictOutcome)
	for _, e := range rep.Entries {
		page.Report.Entries = append(page.Report.Entries, V13ReviewEntry{
			HunkID:       string(e.HunkID),
			Path:         e.Path,
			Kind:         string(e.Kind),
			Content:      string(e.Content),
			Bucket:       string(e.Bucket),
			Tags:         e.Tags,
			ByRule:       e.ByRule,
			Unattributed: e.Bucket == review.BucketUnattributed,
		})
	}
	page.Report.UnattributedCount = rep.UnattributedCount()
	for _, a := range rep.Acceptances {
		page.Report.Acceptances = append(page.Report.Acceptances, V13AcceptanceReview{
			ID:                      a.AcceptanceID,
			Statement:               a.Statement,
			RelatedHunks:            v13HunkIDs(a.RelatedHunks),
			FunctionalCovered:       a.FunctionalCovered,
			FunctionalGap:           string(a.FunctionalGap),
			FunctionalSatisfiedBy:   a.FunctionalSatisfiedBy,
			FunctionalBlocking:      a.FunctionalBlocking,
			IndependenceRequired:    string(a.IndependenceRequired),
			IndependenceCovered:     a.IndependenceCovered,
			IndependenceSatisfiedBy: a.IndependenceSatisfiedBy,
			Strict:                  string(a.Strict),
		})
	}
	for _, p := range rep.ProtectedTouched {
		page.Report.Protected = append(page.Report.Protected, V13ProtectedTouch{
			Path:         p.Path,
			BindingID:    p.BindingID,
			AcceptanceID: p.AcceptanceID,
			ExpectedHash: p.ExpectedHash,
		})
	}
	for _, b := range rep.Blockers {
		page.Report.Blockers = append(page.Report.Blockers, V13Blocker{Kind: string(b.Kind), Describe: v13BlockerText(b)})
	}
	page.Report.ApproveBlocked = rep.ApproveBlocked()
	if rep.ApproveBlocked() {
		page.DecideBlocker = fmt.Sprintf("存在 %d 个 Approve 阻断项(见阻断清单);§19.2:Unattributed 非空不得 Approve。", len(rep.Blockers))
	}
	// 决定资格与页面呈现同判定:报告可用且未被阻断才开放批准;否决永远可用。
	page.CanDecide = page.Report.Available
}

func v13HunkIDs(ids []review.HunkID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, string(id))
	}
	return out
}

// v13BlockerText 把阻断项转成确定性人读文案(按种类分支,不做字符串猜测)。
func v13BlockerText(b review.Blocker) string {
	switch b.Kind {
	case review.BlockerUnattributed:
		if b.HunkID != "" {
			return fmt.Sprintf("未归账 hunk %s(%s)", v13Short(string(b.HunkID)), b.Path)
		}
		return "未归账文件条目 " + b.Path
	case review.BlockerIndependenceGap:
		return fmt.Sprintf("验收条款 %s 的 strict 独立性要求未被满足且无有效豁免", b.AcceptanceID)
	case review.BlockerStrictResolution:
		return "豁免判定本身不可用:" + b.Err.Error()
	default:
		return string(b.Kind)
	}
}

/* ---------- Owner 决定(AC4:只写 HumanApprovalEvidence) ---------- */

// V13Decide 处理 Patch Review 的 Owner 决定(approve|reject)。
//
// 批准与页面同判定:归账不可用或 ApproveBlocked 时不落证据,带提示回评审页;
// 否决不受阻断限制(§19:否决永远可用)。成功只追加 human_approval 证据行,
// 不触碰任何状态字段 —— 状态由 State Engine 从证据派生(INV-STATE-01)。
func (s *Server) V13Decide(w http.ResponseWriter, r *http.Request) {
	env, ok := s.v13env()
	csid := v13CSIDParam(r)
	back := "/v13/review/" + string(csid)
	if !ok {
		redirectFlash(w, r, back, "bad", v13NotLoadable)
		return
	}
	decision := strings.TrimSpace(r.FormValue("decision"))
	reason := strings.TrimSpace(r.FormValue("reason"))
	decidedBy := strings.TrimSpace(r.FormValue("decided_by"))
	if decision != string(approval.DecisionApprove) && decision != string(approval.DecisionReject) {
		redirectFlash(w, r, back, "bad", "决定方向必须是 approve 或 reject,缺席的决定不是默认批准。")
		return
	}

	c, err := s.loadV13Change(r.Context(), env, csid)
	if err != nil {
		if errors.Is(err, errV13NotFound) {
			http.Error(w, "候选不存在: "+string(csid), http.StatusNotFound)
			return
		}
		redirectFlash(w, r, back, "bad", "读取候选失败:"+err.Error())
		return
	}
	taskID := v13ApprovalTaskID(c)
	if taskID == "" {
		redirectFlash(w, r, back, "bad", "无法确定候选归属任务(候选没有产生它的实现 Run 记录),不编造决定坐标。")
		return
	}

	dec := approval.Decision(decision)
	if dec == approval.DecisionApprove {
		blocked, why := s.v13ApproveBlocked(r.Context(), env, c)
		if blocked {
			redirectFlash(w, r, back, "bad", "Approve 被阻断:"+why)
			return
		}
	}

	ev, err := approval.NewApproval(approval.ApprovalSpec{
		Project:     c.cs.Project(),
		TaskID:      taskID,
		ChangeSetID: string(c.cs.ID()),
		Decision:    dec,
		Reason:      reason,
		DecidedBy:   decidedBy,
		DecidedAt:   time.Now(),
	})
	if err != nil {
		redirectFlash(w, r, back, "bad", "决定证据构造被拒绝:"+err.Error())
		return
	}
	if _, err := env.approvals.RecordApproval(r.Context(), ev); err != nil {
		redirectFlash(w, r, back, "bad", "决定证据落库失败:"+err.Error())
		return
	}
	redirectFlash(w, r, back, "ok",
		"已记录 HumanApprovalEvidence("+decision+"):决定是 append-only 证据;任务状态由 State Engine 派生,本界面不写状态。")
}

// v13ApprovalTaskID 取候选归属任务:验证 Run 记录优先,回退产生它的实现 Run。
// 两者都缺席返回空(调用方拒绝落证据,不编造坐标)。
func v13ApprovalTaskID(c *v13ChangeCtx) string {
	if c.taskID != "" {
		return c.taskID
	}
	return c.producerTask
}

// v13ApproveBlocked 用与页面渲染完全相同的装配链重算归账,报告批准是否被阻断
// (服务端防线;页面按钮禁用只是呈现层)。
func (s *Server) v13ApproveBlocked(ctx context.Context, env v13Env, c *v13ChangeCtx) (bool, string) {
	switch {
	case c.task == nil:
		return true, "没有 TaskContract,无法建立归账规则。"
	case c.frozen == nil || !c.matched:
		return true, "候选与 HEAD 冻结契约不属于同一修订,不能归账。"
	}
	if c.git == nil || c.cs.CandidateRevision() == "" {
		return true, "确定性 hunk 清单不可得。"
	}
	diff, err := c.git.DiffText(c.cs.BaseRevision(), c.cs.CandidateRevision())
	if err != nil {
		return true, "无法重建 diff:" + err.Error()
	}
	inv, err := review.InventoryFromDiff(diff)
	if err != nil {
		return true, "清单解析失败:" + err.Error()
	}
	rules, err := review.RulesFor(c.task.Scope, c.frozen)
	if err != nil {
		return true, "归账规则构建失败:" + err.Error()
	}
	cov, err := deriveV13Coverage(c)
	if err != nil || cov == nil {
		return true, "覆盖推导不可用。"
	}
	rep, err := review.BuildAccounting(review.AccountingInput{
		ChangeSet:    c.cs,
		Inventory:    inv,
		Rules:        rules,
		Frozen:       c.frozen,
		Functional:   cov.Functional,
		Independence: cov.Independence,
	})
	if err != nil {
		return true, "确定性归账失败:" + err.Error()
	}
	if rep.ApproveBlocked() {
		return true, fmt.Sprintf("存在 %d 个阻断项(如未归账 hunk %d 个);Unattributed 非空不得 Approve。", len(rep.Blockers), rep.UnattributedCount())
	}
	return false, ""
}

// V13ProposeRule 处理评审页的「提议新规」动作(P3-08):把人本规则想法落成
// 一条治理提案草稿(policygovernance,DRAFT),证据引用指向本案(候选/任务)。
//
// 边界(AC:一次性决定绝不自动改策略;提议新规绝不自动批准):
//   - 本动作绝不写 HumanApprovalEvidence —— 提案不改变本案的任何决定事实,
//     案子的解决仍走既有 decide 流(仅此一次,既有审批证据语义);
//   - 本动作绝不修改/激活任何运行时策略 —— 候选 YAML 出生即正典解析校验
//     (P3-01/P3-03),但激活只能经治理链(P3-07)把正典 YAML 提交进 Git
//     (DL-09/BI-01);治理存储未接入时显式失败,不静默假装成功;
//   - 治理存储 nil → 明确失败(与页面表单不可用呈现两层一致)。
func (s *Server) V13ProposeRule(w http.ResponseWriter, r *http.Request) {
	csid := v13CSIDParam(r)
	back := "/v13/review/" + string(csid)
	if s.Governance == nil {
		redirectFlash(w, r, back, "bad", "治理存储未接入,无法记录规则提案(只此一次的决定不受影响)。")
		return
	}
	env, ok := s.v13env()
	if !ok {
		redirectFlash(w, r, back, "bad", v13NotLoadable)
		return
	}
	c, err := s.loadV13Change(r.Context(), env, csid)
	if err != nil {
		if errors.Is(err, errV13NotFound) {
			http.Error(w, "候选不存在: "+string(csid), http.StatusNotFound)
			return
		}
		redirectFlash(w, r, back, "bad", "读取候选失败:"+err.Error())
		return
	}

	ruleYAML := strings.TrimSpace(r.FormValue("rule_yaml"))
	proposedBy := strings.TrimSpace(r.FormValue("proposed_by"))
	note := strings.TrimSpace(r.FormValue("rule_note"))
	if ruleYAML == "" || proposedBy == "" {
		redirectFlash(w, r, back, "bad", "候选策略 YAML 与提案人必填;缺席的提案不是提案。")
		return
	}

	// 候选出生即正典解析校验(不建第二 DSL;DL-08:治理字段混入即拒绝)。
	cand, err := policygovernance.NewCandidate([]byte(ruleYAML))
	if err != nil {
		redirectFlash(w, r, back, "bad", "候选策略不是合法 EngineeringPolicy:"+err.Error())
		return
	}

	// 治理身份铸造:每次提议一条新提案(append-only;同案可多次提议,各自成案)。
	id := fmt.Sprintf("prop-%s-%d", strings.ToLower(string(csid)), time.Now().UnixNano())
	draft, err := policygovernance.NewDraft(id, cand, proposedBy)
	if err != nil {
		redirectFlash(w, r, back, "bad", "提案构造被拒绝:"+err.Error())
		return
	}
	draft.Scope = note
	draft.Supersedes = strings.TrimSpace(r.FormValue("supersedes"))
	ev := policygovernance.Event{
		ProposalID:        draft.ID,
		Kind:              policygovernance.EventDraft,
		CandidateRevision: draft.Candidate.Revision(),
		CandidateYAML:     string(draft.Candidate.Raw()),
		ProposedBy:        draft.ProposedBy,
		Scope:             draft.Scope,
		Supersedes:        draft.Supersedes,
		OccurredAt:        time.Now(),
	}
	ev.Evidence = append(ev.Evidence, policygovernance.EvidenceRef{
		Kind: policygovernance.EvidenceChangeSet,
		Ref:  string(c.cs.ID()),
	})
	if taskID := v13ApprovalTaskID(c); taskID != "" {
		ev.Evidence = append(ev.Evidence, policygovernance.EvidenceRef{
			Kind: policygovernance.EvidenceTask,
			Ref:  taskID,
		})
	}
	if _, err := s.Governance.Append(r.Context(), ev); err != nil {
		redirectFlash(w, r, back, "bad", "治理提案落库失败:"+err.Error())
		return
	}
	redirectFlash(w, r, back, "ok",
		"治理提案草稿已记("+draft.ID+",DRAFT):本动作绝不写审批证据、绝不批准本候选;运行时策略不受影响(激活须走治理链 + Git 提交)。")
}
