// v13_tasks.go 是 v1.3 P1 最小界面的共享装配层与 Tasks 页(P1-44,PRD §26.2)。
//
// 边界与语义(全部照抄既有领域包,本文件只做只读取数与投影,不发明规则):
//   - TaskContract / VerificationContract 从 Git 制品(.tally/,PRD §6.1)经
//     internal/artifacts 装载 —— Git 是 desired 侧唯一来源;
//   - CodegenRun / ChangeSet / VerificationRun / Evidence / HumanApproval 从
//     事实面(*store.Queries,经 internal/runstore 与 internal/approval)只读;
//   - TaskExecutionState 用 internal/taskstate.Derive 从证据投影派生
//     (INV-STATE-01:状态是派生的,本页没有任何可写状态字段);
//   - 旧的 /tasks「执行摘要」是 legacy 项目执行视图(见 tasks.go 的 P0-22
//     语义边界),本页是 P1 的 Task(TaskContract)视图;页面明确标注两者关系。
//
// 渲染:共享 PageRenderer(internal/web/render.go 的 pageNames 登记表)不含
// v1.3 页面,render.go 不在本任务 Allowed paths 内;因此 v1.3 页面用同一
// base.tmpl + 各自 .tmpl 在本包内自行解析渲染(tally/web 内嵌 FS,零新依赖)。
package handlers

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	"tally/internal/approval"
	"tally/internal/artifacts"
	"tally/internal/changeset"
	"tally/internal/core"
	"tally/internal/evidence"
	"tally/internal/runstore"
	"tally/internal/taskstate"
	"tally/internal/verification"
	assets "tally/web"
)

/* ---------- 渲染:v1.3 页面模板(共享 base.tmpl) ---------- */

var (
	v13TmplOnce sync.Once
	v13Tmpl     map[string]*template.Template
	v13TmplErr  error
)

// v13Template 解析(一次)并返回 base + 指定 v1.3 页面的模板克隆。
// 与 internal/web.Renderer 同一组合方式:base 定义外壳,content 由页面覆盖。
func v13Template(name string) (*template.Template, error) {
	v13TmplOnce.Do(func() {
		funcs := template.FuncMap{
			"short": v13Short,      // 长身份(sha256:…)→ 短显示
			"ftime": v13FormatTime, // 时间 → 人读格式;零值 → 「—」
			"join":  strings.Join,  // 列表 → 逗号串
		}
		base, err := template.New("base").Funcs(funcs).ParseFS(assets.TemplateFS, "templates/base.tmpl")
		if err != nil {
			v13TmplErr = fmt.Errorf("解析 base 模板: %w", err)
			return
		}
		pages := []string{"v13_tasks", "v13_change", "v13_review", "v13_runs"}
		v13Tmpl = make(map[string]*template.Template, len(pages))
		for _, p := range pages {
			clone, err := base.Clone()
			if err != nil {
				v13TmplErr = fmt.Errorf("克隆 base(%s): %w", p, err)
				return
			}
			if _, err := clone.ParseFS(assets.TemplateFS, "templates/"+p+".tmpl"); err != nil {
				v13TmplErr = fmt.Errorf("解析页面 %s: %w", p, err)
				return
			}
			v13Tmpl[p] = clone
		}
	})
	if v13TmplErr != nil {
		return nil, v13TmplErr
	}
	t, ok := v13Tmpl[name]
	if !ok {
		return nil, fmt.Errorf("v13: 未知页面模板 %q", name)
	}
	return t, nil
}

// v13Render 渲染一个 v1.3 页面。模板执行出错只记日志(头已发出),与 Renderer 一致。
func v13Render(w http.ResponseWriter, status int, page string, data any) {
	tmpl, err := v13Template(page)
	if err != nil {
		http.Error(w, "v1.3 页面模板不可用: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = tmpl.ExecuteTemplate(w, "base", data)
}

// v13Short 取长身份的短显示:去掉 sha256: 前缀后截前 12 位。
func v13Short(s string) string {
	s = strings.TrimPrefix(s, "sha256:")
	if len(s) > 12 {
		s = s[:12]
	}
	return s
}

// v13FormatTime 是模板用的时间格式化;零值显示「—」。
func v13FormatTime(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Format("2006-01-02 15:04:05")
}

/* ---------- 数据源守卫与只读 Git 访问 ---------- */

// v13Env 是 v1.3 页面的一次请求读取环境。事实面(s.Trace)与制品仓库
// (s.Content.RepoPath)任一缺席 → 页面渲染「未接入」空状态(T05b 同一模式)。
type v13Env struct {
	runs      *runstore.Store
	approvals *approval.Store
	repoPath  string
	project   core.ProjectID
}

// v13env 从 Server 取 v1.3 读取环境。project 身份来自 Git 注册表
// (.tally/project.yaml),由调用方在装载制品后填入;这里先不猜。
func (s *Server) v13env() (v13Env, bool) {
	if s.Trace == nil || s.Content == nil || strings.TrimSpace(s.Content.RepoPath) == "" {
		return v13Env{}, false
	}
	return v13Env{
		runs:      runstore.New(s.Trace),
		approvals: approval.New(s.Trace),
		repoPath:  s.Content.RepoPath,
	}, true
}

// v13Git 是一次页面请求内的只读 Git 访问(对象库直读;v1.3 界面绝不写 Git,
// 生产 Git 写归 internal/gitwriter,DECISION_LOCKS 5)。
type v13Git struct {
	repo     *git.Repository
	headHash string
	headTime time.Time
}

// openV13Git 打开制品仓库并解析 HEAD。HEAD 缺失(空仓库)返回 error:
// 没有 Git 提交就没有 desired 制品,v1.3 视图无从谈起。
func openV13Git(repoPath string) (*v13Git, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("v13: 打开 git 仓库 %s: %w", repoPath, err)
	}
	ref, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("v13: 解析 HEAD: %w", err)
	}
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("v13: 读 HEAD 提交: %w", err)
	}
	return &v13Git{repo: repo, headHash: commit.Hash.String(), headTime: commit.Committer.When}, nil
}

// DiffText 返回 base..candidate 两个 revision 的 git diff 统一文本。
// 任一 revision 解析不了返回 error(调用方显式降级,不静默给空清单)。
func (g *v13Git) DiffText(baseRev, candidateRev string) (string, error) {
	baseHash, err := g.repo.ResolveRevision(plumbing.Revision(baseRev))
	if err != nil {
		return "", fmt.Errorf("v13: 解析 base revision %q: %w", baseRev, err)
	}
	candHash, err := g.repo.ResolveRevision(plumbing.Revision(candidateRev))
	if err != nil {
		return "", fmt.Errorf("v13: 解析 candidate revision %q: %w", candidateRev, err)
	}
	baseCommit, err := g.repo.CommitObject(*baseHash)
	if err != nil {
		return "", fmt.Errorf("v13: 读 base 提交 %s: %w", baseHash, err)
	}
	candCommit, err := g.repo.CommitObject(*candHash)
	if err != nil {
		return "", fmt.Errorf("v13: 读 candidate 提交 %s: %w", candHash, err)
	}
	baseTree, err := baseCommit.Tree()
	if err != nil {
		return "", fmt.Errorf("v13: 读 base 树: %w", err)
	}
	candTree, err := candCommit.Tree()
	if err != nil {
		return "", fmt.Errorf("v13: 读 candidate 树: %w", err)
	}
	changes, err := object.DiffTree(baseTree, candTree)
	if err != nil {
		return "", fmt.Errorf("v13: 计算 base..candidate diff: %w", err)
	}
	patch, err := changes.Patch()
	if err != nil {
		return "", fmt.Errorf("v13: 生成 base..candidate patch: %w", err)
	}
	var sb strings.Builder
	if err := patch.Encode(&sb); err != nil {
		return "", fmt.Errorf("v13: 编码 diff: %w", err)
	}
	return sb.String(), nil
}

/* ---------- 证据投影:Git 制品 + 事实面行 → taskstate 生命周期事实 ---------- */

// v13Recorder 是本 UI 读取模型的记录 provenance:界面只做确定性只读投影
// (validator 通道,fact-capable),绝不代表 Agent 声明事实(INV-AUTH-01/02)。
var v13Recorder = evidence.Recorder{Kind: evidence.RecorderValidator, Actor: "tally.v13-ui"}

// v13FactsFor 把一个 Task 的 Git 制品与事实面行投影成 taskstate 生命周期事实。
//
// 投影规则(逐条对应 FactKind 的字段合同,见 taskstate.Fact.validate):
//   - task_planned / verification_frozen:来自 Git 制品;发生时刻取制品快照
//     所在 HEAD 提交时刻 —— 同一提交重算必得同一事实(确定性)。
//   - codegen_run_started:每一行 run 记录;status=failed 的行追加
//     codegen_run_failed(FAILED 由终态失败事实推导)。escalated/cancelled
//     行不投影 —— 引擎没有对应生命周期事实种类,行本身仍在 Run 时间线展示,
//     不编造事实种类。
//   - changeset_produced / verification_run_started:每一行候选 / 验证记录。
//
// 审批投影:HumanApprovalEvidence 中绑定候选、经人类通道记录的决定进入
// Snapshot.Approvals;无候选坐标的非候选裁决不投影(taskstate.ApprovalEvidence
// 的字段合同要求候选)。Waiver 不在 P1 UI 投影(WaiverPermitted 的 Policy
// 投影尚无来源,保守取「不允许」,见 taskstate.Snapshot 注释)。
func v13FactsFor(
	ta *artifacts.TaskArtifact,
	va *artifacts.VerificationArtifact,
	headTime time.Time,
	runs []runstore.Run,
	css []*changeset.ChangeSet,
	vruns []runstore.VerificationRun,
) []taskstate.Fact {
	facts := make([]taskstate.Fact, 0, len(runs)+len(css)+len(vruns)+2)
	if ta != nil {
		facts = append(facts, taskstate.Fact{
			ID:          "v13art:task_planned:" + ta.Contract.Hash,
			Kind:        taskstate.FactTaskPlanned,
			At:          headTime,
			Recorder:    v13Recorder,
			RevisionRef: ta.Contract.Hash,
		})
	}
	if va != nil {
		h := va.Frozen.Hash()
		facts = append(facts, taskstate.Fact{
			ID:          "v13art:verification_frozen:" + h,
			Kind:        taskstate.FactVerificationFrozen,
			At:          headTime,
			Recorder:    v13Recorder,
			RevisionRef: h,
		})
	}
	for _, r := range runs {
		switch r.Status {
		case runstore.RunFailed:
			facts = append(facts, taskstate.Fact{
				ID:       fmt.Sprintf("v13run:%d:started", r.ID),
				Kind:     taskstate.FactCodegenRunStarted,
				At:       r.StartedAt,
				Recorder: v13Recorder,
				RunID:    strconv.FormatInt(r.ID, 10),
			})
			failedAt := r.FinishedAt
			if failedAt.IsZero() {
				failedAt = r.StartedAt
			}
			facts = append(facts, taskstate.Fact{
				ID:       fmt.Sprintf("v13run:%d:failed", r.ID),
				Kind:     taskstate.FactCodegenRunFailed,
				At:       failedAt,
				Recorder: v13Recorder,
				RunID:    strconv.FormatInt(r.ID, 10),
			})
		case runstore.RunRunning, runstore.RunCompleted:
			facts = append(facts, taskstate.Fact{
				ID:       fmt.Sprintf("v13run:%d:started", r.ID),
				Kind:     taskstate.FactCodegenRunStarted,
				At:       r.StartedAt,
				Recorder: v13Recorder,
				RunID:    strconv.FormatInt(r.ID, 10),
			})
		default:
			// escalated / cancelled:不编造生命周期事实种类(见函数注释)。
		}
	}
	for _, cs := range css {
		facts = append(facts, taskstate.Fact{
			ID:        "v13cs:" + string(cs.ID()),
			Kind:      taskstate.FactChangeSetProduced,
			At:        cs.CreatedAt(),
			Recorder:  v13Recorder,
			RunID:     cs.ProducerRun(),
			ChangeSet: evidence.ChangeSetID(cs.ID()),
		})
	}
	for _, vr := range vruns {
		facts = append(facts, taskstate.Fact{
			ID:        fmt.Sprintf("v13vrun:%d", vr.ID),
			Kind:      taskstate.FactVerificationRunStarted,
			At:        vr.StartedAt,
			Recorder:  v13Recorder,
			RunID:     strconv.FormatInt(vr.ID, 10),
			ChangeSet: evidence.ChangeSetID(vr.ChangeSetID),
		})
	}
	return facts
}

// v13ApprovalEvidence 把审批行投影成 taskstate.ApprovalEvidence(只取绑定
// 候选的行;决定方向原样保留,approve/reject 都投影,方向语义归引擎)。
func v13ApprovalEvidence(rows []approval.RecordedApproval) []taskstate.ApprovalEvidence {
	out := make([]taskstate.ApprovalEvidence, 0, len(rows))
	for _, ra := range rows {
		csID, ok := ra.Approval.ChangeSetID()
		if !ok {
			continue // 非候选裁决没有候选坐标,不投影(不编造坐标)
		}
		out = append(out, taskstate.ApprovalEvidence{
			ID:        fmt.Sprintf("v13approval:%d", ra.ID),
			Decision:  taskstate.ApprovalDecision(ra.Approval.Decision()),
			ChangeSet: evidence.ChangeSetID(csID),
			Recorder:  evidence.Recorder{Kind: evidence.RecorderHuman, Actor: ra.Approval.DecidedBy()},
			At:        ra.Approval.DecidedAt(),
		})
	}
	return out
}

/* ---------- Tasks 页(PRD §26.1 Tasks / AC1) ---------- */

// V13TaskRow 是 Tasks 页的一个 TaskContract 行:派生状态 + 当前 Run +
// 验证摘要 + 审批现状,全部从事实面推导,没有任何手动状态字段。
type V13TaskRow struct {
	TaskID  string
	Title   string
	Profile string
	// ContractHash 是 TaskContract 内容修订哈希(desired 侧身份,可追溯)。
	ContractHash string
	// HasFrozen 报告该任务在 HEAD 快照里有冻结 VerificationContract。
	HasFrozen bool

	// State / Class 是 taskstate.Derive 的派生结论;DeriveErr 非空时为空值。
	State taskstate.State
	Class taskstate.VerificationClass
	// Gaps 是主线未达 VERIFIED 时的门槛缺口(= blocked reason 的确定性来源)。
	Gaps []taskstate.Gap
	// DeriveErr 是派生失败的原始原因(行内降级展示,不整页 500)。
	DeriveErr string

	// CurrentRun 是派生的当前实现 Run;RunCount 是该任务 Run 总数。
	CurrentRun  string
	CurrentStat string // 当前 Run 的观测状态(running/completed/…)
	RunCount    int

	// CandidateID 是派生的当前候选 ChangeSet;CandidateCount 是候选总数。
	CandidateID    string
	CandidateCount int

	// VerifySummary 是验证 Run 观测结果计数(pass/fail/error)。
	VerifyPass, VerifyFail, VerifyError int
	VerifyTotal                         int

	// ApprovalText 是最近一次审批决定的记录式呈现(证据行,不是状态)。
	ApprovalText string

	// NeedsAttention 报告派生状态处于异常态(需要人工关注)。
	NeedsAttention bool

	// RowErr 是装配该行时事实面读取失败的原始原因(该行降级)。
	RowErr string
}

// V13TasksPage 是 v1.3 Tasks 页数据。
type V13TasksPage struct {
	Page
	// Unavailable:v1.3 数据源(事实面 / 制品仓库)未接入。
	Unavailable bool
	// LoadErr:Git 制品装载失败(如 HEAD 缺失、注册表缺失)。
	LoadErr string
	// ProjectID 来自 Git 注册表;空 = 未装载。
	ProjectID    string
	SourceCommit string
	Tasks        []V13TaskRow
}

// V13Tasks 渲染 v1.3 Task(TaskContract)列表:派生状态、当前 Run、验证摘要。
func (s *Server) V13Tasks(w http.ResponseWriter, r *http.Request) {
	page := V13TasksPage{Page: s.page(r, "P1 任务", "v13")}
	env, ok := s.v13env()
	if !ok {
		page.Unavailable = true
		v13Render(w, http.StatusOK, "v13_tasks", page)
		return
	}
	s.v13TasksPage(r, env, &page)
	v13Render(w, http.StatusOK, "v13_tasks", page)
}

func (s *Server) v13TasksPage(r *http.Request, env v13Env, page *V13TasksPage) {
	ctx := r.Context()
	g, err := openV13Git(env.repoPath)
	if err != nil {
		page.LoadErr = err.Error()
		return
	}
	snap, err := artifacts.Load(env.repoPath, g.headHash)
	if err != nil {
		page.LoadErr = err.Error()
		return
	}
	env.project = core.ProjectID(snap.ProjectID)
	page.ProjectID = snap.ProjectID
	page.SourceCommit = g.headHash

	for _, ta := range snap.Tasks {
		row := s.v13TaskRow(ctx, env, g, snap, ta)
		page.Tasks = append(page.Tasks, row)
	}
}

// v13FrozenFor 取某任务的冻结 VerificationContract 制品(HEAD 快照内 task_id 唯一)。
func v13FrozenFor(snap *artifacts.Snapshot, taskID string) *artifacts.VerificationArtifact {
	for i := range snap.Verifications {
		if snap.Verifications[i].Frozen.TaskID() == taskID {
			return &snap.Verifications[i]
		}
	}
	return nil
}

func (s *Server) v13TaskRow(ctx context.Context, env v13Env, g *v13Git, snap *artifacts.Snapshot, ta artifacts.TaskArtifact) V13TaskRow {
	row := V13TaskRow{
		TaskID:       ta.Contract.ID,
		Title:        ta.Contract.Title,
		Profile:      ta.Contract.Profile,
		ContractHash: ta.Contract.Hash,
	}
	va := v13FrozenFor(snap, ta.Contract.ID)
	row.HasFrozen = va != nil

	runs, err := env.runs.CodegenRunsForTask(ctx, env.project, ta.Contract.ID)
	if err != nil {
		row.RowErr = err.Error()
		return row
	}
	css, err := env.runs.ChangeSetsForTask(ctx, env.project, ta.Contract.ID)
	if err != nil {
		row.RowErr = err.Error()
		return row
	}
	vruns, err := env.runs.VerificationRunsForTask(ctx, env.project, ta.Contract.ID)
	if err != nil {
		row.RowErr = err.Error()
		return row
	}
	approvals, err := env.approvals.ApprovalsForTask(ctx, env.project, ta.Contract.ID)
	if err != nil {
		row.RowErr = err.Error()
		return row
	}

	row.RunCount = len(runs)
	if len(runs) > 0 {
		// CodegenRunsForTask 按开始时刻定序,末位即最新。
		latest := runs[len(runs)-1]
		row.CurrentRun = strconv.FormatInt(latest.ID, 10)
		row.CurrentStat = string(latest.Status)
	}
	row.CandidateCount = len(css)
	if len(css) > 0 {
		row.CandidateID = string(css[len(css)-1].ID())
	}
	row.VerifyTotal = len(vruns)
	for _, vr := range vruns {
		switch vr.Status {
		case runstore.VerificationPass:
			row.VerifyPass++
		case runstore.VerificationFail:
			row.VerifyFail++
		case runstore.VerificationError:
			row.VerifyError++
		}
	}
	if len(approvals) > 0 {
		latest := approvals[len(approvals)-1]
		row.ApprovalText = string(latest.Approval.Decision()) + " · " + latest.Approval.DecidedBy() + " · " + v13FormatTime(latest.Approval.DecidedAt())
	}

	res, err := taskstate.Derive(taskstate.Snapshot{
		Contract:  v13FrozenContract(va),
		Facts:     v13FactsFor(&ta, va, g.headTime, runs, css, vruns),
		Approvals: v13ApprovalEvidence(approvals),
	})
	if err != nil {
		row.DeriveErr = err.Error()
		return row
	}
	row.State = res.State
	row.Class = res.VerificationClass
	row.Gaps = res.Gaps
	// 当前 Run / 候选以派生结论为准(规范顺序的指针);派生不出指针时
	// 连同观测状态一并留空,不给「有状态无对象」的残行。
	if res.CurrentRun != "" {
		row.CurrentRun = res.CurrentRun
	} else {
		row.CurrentStat = ""
	}
	if res.CurrentCandidate != "" {
		row.CandidateID = string(res.CurrentCandidate)
	} else {
		row.CandidateID = ""
	}
	row.NeedsAttention = res.State.IsException()
	return row
}

// v13FrozenContract 取冻结契约值;va 为 nil 时返回 nil(尚无冻结契约)。
func v13FrozenContract(va *artifacts.VerificationArtifact) *verification.Frozen {
	if va == nil {
		return nil
	}
	return va.Frozen
}

// v13CSIDParam 取 URL 里的 ChangeSet 身份参数。
func v13CSIDParam(r *http.Request) changeset.ChangeSetID {
	return changeset.ChangeSetID(chi.URLParam(r, "csid"))
}

// v13NotLoadable 是 v1.3 页面通用「数据源未接入」空状态文案。
const v13NotLoadable = "v1.3 数据源未接入:需要事实面(数据库)与 Git 制品仓库。"

// errV13NotFound 标记目标不存在(404 语义);用类型而非字符串匹配。
var errV13NotFound = errors.New("v13: 目标不存在")
