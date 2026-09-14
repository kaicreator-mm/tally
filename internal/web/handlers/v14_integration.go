// v14_integration.go 是集成审计投影页(P5-05):把 Accepted(验收)与
// Integrated(已集成)呈现为两个可分别观察的工程状态 —— 绝不互相等同
// (AC:UI 永不把候选验收等同于成功集成)。
//
// 权威边界(改动前必读):
//   - Accepted 分区 = 既有审批证据事实面(HumanApprovalEvidence,append-only;
//     按 HEAD 快照的 standard 任务逐个读取最近一条决定);
//   - Integrated 分区 = gitintegration.ListIntegrationRefs 的 Git 事实
//     (P5-02 确定性集成分支,refs/heads/tally/integration/*);
//   - 两个分区数据来源独立、互不推导:有验收无集成分支的任务在 Accepted 区
//     如实呈现「尚未集成」;集成分支的存在不暗示任何验收结论;
//   - 全页只读零写路由(DL-19)。
package handlers

import (
	"net/http"
	"sort"

	"tally/internal/artifacts"
	"tally/internal/changeset"
	"tally/internal/core"
	"tally/internal/gitintegration"
)

// V14AcceptedRow 是 Accepted 分区的一行:一次已录得的验收决定(观测值原样)。
type V14AcceptedRow struct {
	TaskID    string
	ChangeSet string
	Decision  string
	DecidedBy string
	Reason    string
	At        string
	// Integrated 是该候选的集成事实标注:有对应集成分支 = 已集成(分支名),
	// 否则「尚未集成」—— 两个字段永不互相推导。
	Integrated string
}

// V14IntegratedRow 是 Integrated 分区的一行:一条 Git 集成分支事实。
type V14IntegratedRow struct {
	Ref     string
	Commit  string
	Subject string
	Date    string
}

// V14IntegrationPage 是集成审计投影页数据。
type V14IntegrationPage struct {
	Page
	Unavailable bool
	LoadErr     string
	ProjectID   string
	// TaskCount 是扫描的 standard TaskContract 数。
	TaskCount int

	Accepted   []V14AcceptedRow
	Integrated []V14IntegratedRow
}

// V14Integration 渲染集成审计投影页(P5-05)。
func (s *Server) V14Integration(w http.ResponseWriter, r *http.Request) {
	page := V14IntegrationPage{Page: s.page(r, "Integration · v1.4", "v14-integration")}
	env, ok := s.v13env()
	if !ok {
		page.Unavailable = true
		v14CARender(w, http.StatusOK, "v14_integration", page)
		return
	}
	g, err := openV13Git(env.repoPath)
	if err != nil {
		page.LoadErr = err.Error()
		v14CARender(w, http.StatusOK, "v14_integration", page)
		return
	}
	snap, err := artifacts.Load(env.repoPath, g.headHash)
	if err != nil {
		page.LoadErr = err.Error()
		v14CARender(w, http.StatusOK, "v14_integration", page)
		return
	}
	env.project = core.ProjectID(snap.ProjectID)
	page.ProjectID = snap.ProjectID
	ctx := r.Context()

	// Integrated 分区:Git 集成分支事实(P5-02/P5-03 落地的确定性引用)。
	refs, err := gitintegration.ListIntegrationRefs(ctx, env.repoPath)
	if err != nil {
		page.LoadErr = "集成分支列举失败:" + err.Error()
		v14CARender(w, http.StatusOK, "v14_integration", page)
		return
	}
	integratedByRef := map[string]bool{}
	for _, ref := range refs {
		page.Integrated = append(page.Integrated, V14IntegratedRow{
			Ref: ref.Ref, Commit: ref.Commit, Subject: ref.Subject, Date: ref.Date,
		})
		integratedByRef[ref.Ref] = true
	}

	// Accepted 分区:逐 standard 任务读最近一条审批决定(既有审批证据事实面)。
	for _, ta := range snap.Tasks {
		if ta.Contract.Profile != "standard" {
			continue
		}
		page.TaskCount++
		approvals, err := env.approvals.ApprovalsForTask(ctx, env.project, ta.Contract.ID)
		if err != nil {
			page.LoadErr = "审批证据读取失败(任务 " + ta.Contract.ID + "):" + err.Error()
			v14CARender(w, http.StatusOK, "v14_integration", page)
			return
		}
		if len(approvals) == 0 {
			continue // 无验收决定的任务不是本页对象
		}
		latest := approvals[len(approvals)-1]
		a := latest.Approval
		csID, _ := a.ChangeSetID()
		row := V14AcceptedRow{
			TaskID:     ta.Contract.ID,
			ChangeSet:  csID,
			Decision:   string(a.Decision()),
			DecidedBy:  a.DecidedBy(),
			Reason:     a.Reason(),
			At:         v13FormatTime(a.DecidedAt()),
			Integrated: "尚未集成",
		}
		if csID != "" {
			if want, err := gitintegration.IntegrationBranchRefFor(changeset.ChangeSetID(csID)); err == nil && integratedByRef[want] {
				row.Integrated = "已集成:" + want
			}
		}
		page.Accepted = append(page.Accepted, row)
	}
	sort.SliceStable(page.Accepted, func(i, j int) bool { return page.Accepted[i].TaskID < page.Accepted[j].TaskID })

	v14CARender(w, http.StatusOK, "v14_integration", page)
}
