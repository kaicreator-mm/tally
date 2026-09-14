// v14_rules.go 是策略治理的 Rules 投影页(P3-06):Proposed / Active /
// Conflicts / History 四个分区,全部从事实面派生、只读无写。
//
// 权威边界(改动前必读,AC:UI 不得把仅存在于 DB 的提案显示为 Active):
//   - Active 分区只从 artifacts.LoadPolicies(repo, HEAD) 的 Git 制品快照
//     派生(DL-09:运行时策略 = pinned 快照内容)。治理存储(append-only
//     事件面,P3-02)只提供 Proposed / History 与一致性核对 —— 治理面记为
//     ACTIVE 但 HEAD 快照不可见的提案,在「分歧」区显式呈现并注明运行时
//     不受其影响,绝不进入 Active 列表;
//   - 不确定性一等可见:HEAD 里存在但治理面无对应提案记录(或候选修订
//     不一致)的策略,逐条携带不确定标注(先于治理面存在/记录缺失,人裁决);
//   - Conflicts 从 Active 集合做确定性冲突分析(P3-05),只呈现不消解;
//   - History 是治理事件流(append-only)的倒序截断投影,零可变状态;
//   - 治理存储未接入时 Proposed/History/分歧区按未接入诚实呈现,Active
//     (纯 Git 面)照常 —— Git 权威面不因记录面缺席而缺席。
package handlers

import (
	"net/http"
	"sort"

	"tally/internal/artifacts"
	"tally/internal/core"
	"tally/internal/engineeringpolicy"
	"tally/internal/policygovernance"
)

// V14ActivePolicy 是 Active 分区的一行:HEAD 快照里的生效策略 + 与治理面的
// 一致性标注(不确定性一等可见)。
type V14ActivePolicy struct {
	Path        string
	ID          string
	Enforcement string
	Revision    string
	GateKeys    []string
	// GovernanceNote 是与治理面的一致性标注:提案一致(携带提案 ID)或
	// 不确定(治理面无对应记录/修订不一致)。
	GovernanceNote string
}

// V14DivergedPolicy 是治理分歧区的一行:治理面记为 ACTIVE,但 HEAD 快照不可
// 观察(同 id 不存在或候选修订不一致)—— 运行时不受其影响(DL-09)。
type V14DivergedPolicy struct {
	ProposalID   string
	PolicyID     string
	CandidateRev string
	ApprovedBy   string
	Reason       string // 不可观察的具体原因(id 缺席 / 修订不一致)
}

// V14ProposalRow 是 Proposed 分区的一行:治理面中未到终态/未激活的提案。
type V14ProposalRow struct {
	ID           string
	Status       string
	PolicyID     string
	CandidateRev string
	ProposedBy   string
	EvidenceN    int
	Scope        string
}

// V14ConflictRow 是 Conflicts 分区的一行(P3-05 发现的只读投影)。
type V14ConflictRow struct {
	Severity  string
	Kind      string
	PolicyIDs []string
	Detail    string
}

// V14GovEventRow 是 History 分区的一行(治理事件流倒序截断)。
type V14GovEventRow struct {
	ProposalID string
	ActionKind string
	ApprovedBy string
	OccurredAt string
}

// V14RulesPage 是 Rules 投影页数据。
type V14RulesPage struct {
	Page
	Unavailable bool // 数据源未接入
	LoadErr     string
	ProjectID   string
	// GovernanceAvailable 报告治理存储是否接入(未接入时 Proposed/History/
	// 分歧区诚实呈现,Active 照常)。
	GovernanceAvailable bool

	ActivePolicies []V14ActivePolicy
	Diverged       []V14DivergedPolicy
	Proposed       []V14ProposalRow
	Conflicts      []V14ConflictRow
	History        []V14GovEventRow
	// HistoryTotal 是治理事件流全长(截断呈现时用于如实声明)。
	HistoryTotal int
}

const v14RulesHistoryCap = 20

// V14Rules 渲染 Rules 投影页(P3-06)。
func (s *Server) V14Rules(w http.ResponseWriter, r *http.Request) {
	page := V14RulesPage{Page: s.page(r, "Rules · v1.4", "v14-rules")}
	env, ok := s.v13env()
	if !ok {
		page.Unavailable = true
		v14CARender(w, http.StatusOK, "v14_rules", page)
		return
	}
	g, err := openV13Git(env.repoPath)
	if err != nil {
		page.LoadErr = err.Error()
		v14CARender(w, http.StatusOK, "v14_rules", page)
		return
	}
	snap, err := artifacts.Load(env.repoPath, g.headHash)
	if err != nil {
		page.LoadErr = err.Error()
		v14CARender(w, http.StatusOK, "v14_rules", page)
		return
	}
	env.project = core.ProjectID(snap.ProjectID)
	page.ProjectID = snap.ProjectID
	page.GovernanceAvailable = s.Governance != nil

	ctx := r.Context()

	// Active:只从 HEAD 策略快照派生(DL-09)。
	policySnap, err := artifacts.LoadPolicies(env.repoPath, g.headHash)
	if err != nil {
		page.LoadErr = "策略快照装载失败:" + err.Error()
		v14CARender(w, http.StatusOK, "v14_rules", page)
		return
	}

	// 治理面:重建全部提案(历史不一致 = 一等的不确定性行,不静默跳过)。
	var proposals []policygovernance.Proposal
	var broken []string
	if s.Governance != nil {
		events, err := s.Governance.AllEvents(ctx)
		if err != nil {
			page.LoadErr = "治理事件读取失败:" + err.Error()
			v14CARender(w, http.StatusOK, "v14_rules", page)
			return
		}
		seen := map[string]bool{}
		order := make([]string, 0, len(events))
		for _, ev := range events {
			if !seen[ev.ProposalID] {
				seen[ev.ProposalID] = true
				order = append(order, ev.ProposalID)
			}
		}
		for _, pid := range order {
			p, err := s.Governance.Reconstruct(ctx, pid)
			if err != nil {
				broken = append(broken, pid)
				continue
			}
			proposals = append(proposals, p)
		}
		for _, pid := range broken {
			page.History = append(page.History, V14GovEventRow{ProposalID: pid, ActionKind: "治理历史不一致(重放失败)"})
		}
	}

	// Active 行 + 治理一致性标注;同 id 同修订的 ACTIVE 提案从分歧/在提区排除。
	activeByID := map[string]V14ActivePolicy{}
	for _, pa := range policySnap.Policies {
		row := V14ActivePolicy{
			Path:        pa.Path,
			ID:          pa.Policy.ID,
			Enforcement: string(pa.Policy.Enforcement),
			Revision:    pa.Policy.Revision,
			GateKeys:    pa.Policy.GateKeys(),
		}
		matched := ""
		for _, p := range proposals {
			if p.Status == policygovernance.StatusActive && p.Candidate.Revision() == pa.Policy.Revision {
				matched = "治理提案 " + p.ID + " 一致(ACTIVE)"
				break
			}
		}
		if matched == "" {
			matched = "不确定:治理面无对应 ACTIVE 提案记录(可能先于治理面存在,或治理记录缺失;需要人裁决)"
		}
		row.GovernanceNote = matched
		page.ActivePolicies = append(page.ActivePolicies, row)
		activeByID[pa.Policy.ID] = row
	}

	// 分歧区:治理 ACTIVE 但 HEAD 不可观察(AC:绝不进入 Active 列表)。
	for _, p := range proposals {
		if p.Status != policygovernance.StatusActive {
			continue
		}
		policyID := p.Candidate.Policy().ID
		if act, ok := activeByID[policyID]; ok && act.Revision == p.Candidate.Revision() {
			continue // 一致,已在 Active 区
		}
		reason := "HEAD 快照中不存在该 id 的策略"
		if act, ok := activeByID[policyID]; ok {
			reason = "HEAD 存在同 id 策略但内容修订不一致(快照 " + act.Revision + " vs 提案 " + p.Candidate.Revision() + ")"
		}
		page.Diverged = append(page.Diverged, V14DivergedPolicy{
			ProposalID:   p.ID,
			PolicyID:     policyID,
			CandidateRev: p.Candidate.Revision(),
			ApprovedBy:   p.ApprovedBy,
			Reason:       reason,
		})
	}
	sort.SliceStable(page.Diverged, func(i, j int) bool { return page.Diverged[i].ProposalID < page.Diverged[j].ProposalID })

	// Proposed 区:未激活的提案(PROPOSED/DRAFT;终态 REJECTED/SUPERSEDED 不占
	// 主列表 —— 全历史在 History 区可追溯)。
	for _, p := range proposals {
		if p.Status != policygovernance.StatusProposed && p.Status != policygovernance.StatusDraft {
			continue
		}
		page.Proposed = append(page.Proposed, V14ProposalRow{
			ID: p.ID, Status: string(p.Status), PolicyID: p.Candidate.Policy().ID,
			CandidateRev: p.Candidate.Revision(),
			ProposedBy:   p.ProposedBy, EvidenceN: len(p.Evidence), Scope: p.Scope,
		})
	}
	sort.SliceStable(page.Proposed, func(i, j int) bool { return page.Proposed[i].ID < page.Proposed[j].ID })

	// Conflicts:Active 集合的确定性冲突分析(P3-05;只呈现不消解)。
	pols := make([]engineeringpolicy.Policy, 0, len(policySnap.Policies))
	for _, pa := range policySnap.Policies {
		pols = append(pols, pa.Policy)
	}
	for _, c := range policygovernance.AnalyzeConflicts(pols) {
		page.Conflicts = append(page.Conflicts, V14ConflictRow{
			Severity: string(c.Severity), Kind: string(c.Kind), PolicyIDs: c.PolicyIDs, Detail: c.Detail,
		})
	}

	// History:治理事件流倒序截断(全长度如实声明)。
	if s.Governance != nil {
		events, _ := s.Governance.AllEvents(ctx)
		page.HistoryTotal = len(events)
		for i := len(events) - 1; i >= 0 && len(page.History) < v14RulesHistoryCap; i-- {
			ev := events[i]
			row := V14GovEventRow{ProposalID: ev.ProposalID, ActionKind: string(ev.Kind), OccurredAt: v13FormatTime(ev.OccurredAt)}
			if ev.Kind == policygovernance.EventActivate {
				row.ApprovedBy = ev.ApprovedBy
			}
			page.History = append(page.History, row)
		}
	}

	v14CARender(w, http.StatusOK, "v14_rules", page)
}
