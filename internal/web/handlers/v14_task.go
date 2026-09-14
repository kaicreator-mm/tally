// v14_task.go 是 v1.4 P3-08 的闭环任务详情面(PRD §26.2):一个页面回答
// 当前执行到哪(相位/派生状态)、为什么可信(依据)、哪里不确定、需要人做什么。
//
// 数据纪律(与 v13_tasks.go 完全同源,本文件只做只读取数与投影):
//   - TaskContract / 冻结 VerificationContract 从 Git 制品(artifacts.Load)装载
//     —— Git 是 desired 侧唯一来源;
//   - CodegenRun / ChangeSet / VerificationRun / 审批 / 暴露事实从事实面
//     (*store.Queries 经 runstore/approval)只读;
//   - TaskExecutionState 用 taskstate.Derive 从证据投影派生(INV-STATE-01);
//     每个 VERIFIED 类状态旁必须呈现其依据引用(INV-11),strict_waived 必须
//     与 strict_satisfied 可区分(豁免不得显示成普通满足);
//   - 尝试链是外层闭环的事实:修复尝试 = 新 CodegenRun 且 EscalatedFrom 指向
//     前一次尝试(ADR-0001 §2:attempt-numbered, parent-linked);序号由父链
//     推导,不是任何可写状态字段;
//   - v1.4.1 P2-02:页面按人类心智模型呈现(PRD §7/§29:Overview / Plan /
//     Changes / Verification / Reviews / Activity,全部是既有领域对象的
//     projection,不改状态机,DL-19);验证结论依据面消费 P1-07 已接线的
//     编译权威读路径(LoadBundle → CompileAuthority,与 worker 同一生产
//     seam、同一 pinned HEAD 提交),Compiled 缺席时如实标注,绝不冒充档位。
package handlers

import (
	"context"
	"net/http"
	"sort"
	"strconv"

	"github.com/go-chi/chi/v5"

	"tally/internal/artifacts"
	"tally/internal/changeset"
	"tally/internal/core"
	"tally/internal/evidence"
	"tally/internal/runstore"
	"tally/internal/taskstate"
)

// v14MaxChainDepth 是尝试链推导的防御性深度上限:事实面出现环/超长链时停止
// 上溯并标记截断,绝不做无界遍历。
const v14MaxChainDepth = 64

// V14AttemptView 是尝试链里的一次 CodegenRun(闭环第 1 次尝试或第 N 次修复)。
type V14AttemptView struct {
	// No 是尝试链序号(父链推导,1 = 根尝试;≥2 = 修复尝试)。
	No int
	// ChainTruncated 报告上溯深度超限(异常事实面,如实呈现)。
	ChainTruncated bool
	ID             string // 持久化 Run 行身份
	Status         string // 观测状态(running/completed/failed/…)
	FailureReason  string // 终态失败原因(如 no_candidate / runtime_error)
	Iterations     int32
	StartedAt      string
	FinishedAt     string
	// Claim 是 runtime 自述摘要:永远是 claim,不是事实(INV-AUTH-01/02)。
	Claim string
	// CandidateID 是该 Run 固定的候选身份(空 = 没有固定候选)。
	CandidateID string
}

// V14CandidateView 是一个不可变候选快照及其验证现状。
type V14CandidateView struct {
	ID            string // 完整 ChangeSet 身份(链接去既有 v1.3 Change 详情页)
	CreatedAt     string
	ProducerRun   string
	FileCount     int
	VerifyPass    int
	VerifyFail    int
	VerifyError   int
	VerifyRunning bool
	// Current 报告这是派生的当前候选(taskstate.Derive 的规范指针)。
	Current bool
	// SupersededBy / Supersedes 是取代事实的投影(append-only,不合并)。
	SupersededBy []string
	Supersedes   []string
}

// V14ExposureView 是一条暴露事实的展示(held-out 内容进入实现侧可见通道的
// append-only 账面;不确定性必须可见,不得隐藏)。
type V14ExposureView struct {
	BindingID  string
	Acceptance string
	Channel    string
	ChangeSet  string
	At         string
	Reason     string
}

// V14ApprovalView 是一条人类审批证据的展示(记录式呈现,不是状态)。
type V14ApprovalView struct {
	Decision string
	By       string
	At       string
}

// V14ClauseView 是合同验收条款的展示行。
type V14ClauseView struct {
	ID        string
	Statement string
}

// V14TaskPage 是闭环任务详情页数据。
type V14TaskPage struct {
	Page
	Unavailable bool // 数据源未接入
	NotFound    bool // HEAD 快照里没有这个 TaskContract
	LoadErr     string
	RowErr      string // 事实面读取失败的原始原因(页面降级)

	TaskID  string
	Title   string
	Profile string
	// ContractHash / FrozenHash 是 desired 侧内容身份(可追溯)。
	ContractHash string
	HasFrozen    bool
	FrozenHash   string
	ProjectID    string
	SourceCommit string
	Objective    string
	Acceptance   []V14ClauseView

	// State / Class 是 taskstate.Derive 的派生结论;DeriveErr 非空时为空值。
	State            taskstate.State
	WaivedClass      bool // strict_waived:豁免放行,不得显示成普通满足
	ClassText        string
	DeriveErr        string
	Gaps             []taskstate.Gap
	Basis            []string // INV-11:VERIFIED 类结论必须携带的依据引用
	IgnoredClaims    []taskstate.IgnoredClaim
	CurrentRun       string
	CurrentCandidate string
	NeedsAttention   bool

	Attempts   []V14AttemptView
	Candidates []V14CandidateView
	Exposures  []V14ExposureView
	Approvals  []V14ApprovalView

	// ---- v1.4.1 P2-02:验证结论依据面(PRD §14)。字段语义与 Change Detail 页
	// (V14ChangePage 的 P1-07 同名字段)一致:消费同一编译权威读路径,不在本页
	// 重复投影实现(DL-18:extend-not-duplicate)。Compiled 缺席时 HasVerdict=false,
	// 页面保持「未接入」标注,绝不渲染结论、档位或 compiled 引用。
	// Coverage 是当前候选的功能 + independence 覆盖展示(§8.8.2 同修订推导,
	// 与 Change Detail 同一装载/推导);BasisDims 是依据四维度行
	// (coverage/assurance/unmatched/waiver)。
	Coverage   V13CoverageView
	BasisDims  []V14BasisDim
	HasVerdict bool
	// Verdict 是验证结论(VERIFIED / VERIFIED_LIMITED / NOT_VERIFIED);
	// VERIFIED_LIMITED 是 limited 结论,必须与普通 VERIFIED 区分展示。
	Verdict        taskstate.Verdict
	VerdictLimited bool
	VerdictOK      bool
	// CompiledHash 是结论所依据的已编译合同身份哈希;CompiledNote 是编译权威
	// 未接入时的显式类型化原因(接入成功时为空)。
	CompiledHash string
	CompiledNote string
}

// V14Task 渲染闭环任务详情(PRD §26.2):Intent/Contract/Current Run/Current
// ChangeSet/Verification/Repair attempts/Verdict/Human action 全部从事实派生。
func (s *Server) V14Task(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "task")
	// Active key "v13"(P2-01 移交的高亮细化):任务详情是 Work 域视图
	// (PRD §29:Work → Task),左侧「工作」入口在详情页保持高亮。
	page := V14TaskPage{Page: s.page(r, "闭环任务 · v1.4", "v13"), TaskID: taskID}
	env, ok := s.v13env()
	if !ok {
		page.Unavailable = true
		v14Render(w, http.StatusOK, "v14_task", page)
		return
	}
	g, err := openV13Git(env.repoPath)
	if err != nil {
		page.LoadErr = err.Error()
		v14Render(w, http.StatusOK, "v14_task", page)
		return
	}
	snap, err := artifacts.Load(env.repoPath, g.headHash)
	if err != nil {
		page.LoadErr = err.Error()
		v14Render(w, http.StatusOK, "v14_task", page)
		return
	}
	env.project = core.ProjectID(snap.ProjectID)
	page.ProjectID = snap.ProjectID
	page.SourceCommit = g.headHash

	var ta *artifacts.TaskArtifact
	for i := range snap.Tasks {
		if snap.Tasks[i].Contract.ID == taskID {
			ta = &snap.Tasks[i]
			break
		}
	}
	if ta == nil {
		page.NotFound = true
		v14Render(w, http.StatusNotFound, "v14_task", page)
		return
	}
	s.v14TaskPage(r.Context(), env, g, snap, ta, &page)
	v14Render(w, http.StatusOK, "v14_task", page)
}

// v14TaskPage 装配详情页:desired 合同 + 事实面 + 派生结论。
func (s *Server) v14TaskPage(ctx context.Context, env v13Env, g *v13Git, snap *artifacts.Snapshot, ta *artifacts.TaskArtifact, page *V14TaskPage) {
	page.Title = ta.Contract.Title
	page.Profile = ta.Contract.Profile
	page.ContractHash = ta.Contract.Hash
	page.Objective = ta.Contract.Objective
	for _, a := range ta.Contract.Acceptance {
		page.Acceptance = append(page.Acceptance, V14ClauseView{ID: a.ID, Statement: a.Statement})
	}
	va := v13FrozenFor(snap, ta.Contract.ID)
	page.HasFrozen = va != nil
	if va != nil {
		page.FrozenHash = va.Frozen.Hash()
	}

	runs, err := env.runs.CodegenRunsForTask(ctx, env.project, ta.Contract.ID)
	if err != nil {
		page.RowErr = err.Error()
		return
	}
	css, err := env.runs.ChangeSetsForTask(ctx, env.project, ta.Contract.ID)
	if err != nil {
		page.RowErr = err.Error()
		return
	}
	vruns, err := env.runs.VerificationRunsForTask(ctx, env.project, ta.Contract.ID)
	if err != nil {
		page.RowErr = err.Error()
		return
	}
	approvals, err := env.approvals.ApprovalsForTask(ctx, env.project, ta.Contract.ID)
	if err != nil {
		page.RowErr = err.Error()
		return
	}
	exposures, err := env.runs.ExposuresForTask(ctx, env.project, ta.Contract.ID)
	if err != nil {
		page.RowErr = err.Error()
		return
	}

	page.Attempts = v14AttemptViews(runs)
	page.Candidates = s.v14CandidateViews(ctx, env.runs, css, vruns)
	for _, e := range exposures {
		row := V14ExposureView{
			BindingID: e.Binding().BindingID,
			Channel:   string(e.Channel()),
			ChangeSet: "—",
			At:        v13FormatTime(e.ExposedAt()),
			Reason:    e.Reason(),
		}
		if ac, ok := e.Acceptance(); ok {
			row.Acceptance = string(ac)
		}
		if cs, ok := e.ChangeSet(); ok {
			row.ChangeSet = v13Short(string(cs))
		}
		page.Exposures = append(page.Exposures, row)
	}
	for _, ra := range approvals {
		page.Approvals = append(page.Approvals, V14ApprovalView{
			Decision: string(ra.Approval.Decision()),
			By:       ra.Approval.DecidedBy(),
			At:       v13FormatTime(ra.Approval.DecidedAt()),
		})
	}

	derive := taskstate.Snapshot{
		Contract:  v13FrozenContract(va),
		Facts:     v13FactsFor(ta, va, g.headTime, runs, css, vruns),
		Approvals: v13ApprovalEvidence(approvals),
	}
	res, err := taskstate.Derive(derive)
	if err != nil {
		page.DeriveErr = err.Error()
		return
	}

	// ---- v1.4.1 P2-02:验证结论依据面(PRD §14)。编译权威经与 worker 完全
	// 相同的 P1-03 生产 seam(LoadBundle → CompileAuthority)从同一 pinned
	// HEAD 提交只读;当前候选的覆盖沿 Change Detail 同一读路径(§8.8.2 同
	// 修订推导)。第一次 Derive 的当前候选定位覆盖推导对象;Derive 是纯函数,
	// Compiled / 覆盖报告只是增量投影输入,重算的其余结论与第一次完全一致
	// (INV-STATE-01)。Compiled 缺席或装配类型化失败是显式展示状态
	// (CompiledNote),不阻断页面,更不冒充档位(AC:依据哪里有就显示到哪里)。
	auth := v14TaskAuthority(env, g, ta.Contract.ID, page)
	cov := s.v14TaskCoverage(ctx, env, res.CurrentCandidate, page)
	if auth != nil {
		derive.Compiled = auth.Compiled
		// 覆盖报告只在同修订推导可用时参与:报告缺席 = 达成档没有事实来源,
		// 投影按 L0 fail-safe,不冒充覆盖(与 Change Detail 同一纪律)。
		if cov != nil {
			derive.Functional = &cov.Functional
			derive.Independence = &cov.Independence
		}
		res2, err := taskstate.Derive(derive)
		if err != nil {
			// Compiled 与冻结契约跨修订等类型化失败:保持第一次推导的结论,
			// 显式记录原因,绝不静默降级或伪造结论。
			page.CompiledNote = "已编译合同未接入:" + err.Error()
		} else {
			res = res2
		}
	}
	page.State = res.State
	page.ClassText = string(res.VerificationClass)
	page.WaivedClass = res.VerificationClass == taskstate.ClassStrictWaived
	page.Gaps = res.Gaps
	page.Basis = res.Basis
	page.IgnoredClaims = res.IgnoredClaims
	page.CurrentRun = res.CurrentRun
	page.CurrentCandidate = string(res.CurrentCandidate)
	page.NeedsAttention = res.State.IsException()
	// 依据四维度行(coverage/assurance/unmatched/waiver)与验证结论投影;
	// 维度未接入的如实标注,展示缺席不放宽任何验证门(PRD §14)。
	page.BasisDims = v14BasisDims(page.Coverage, page.ClassText, page.WaivedClass, res.VerifiedBasis)
	if vb := res.VerifiedBasis; vb != nil {
		page.HasVerdict = true
		page.Verdict = vb.Verdict
		page.VerdictLimited = vb.Verdict == taskstate.VerdictVerifiedLimited
		page.VerdictOK = vb.Verdict == taskstate.VerdictVerified
		page.CompiledHash = vb.CompiledHash
	}
}

// v14TaskAuthority 只读装载本任务的编译权威(与 worker 同一 P1-03 生产 seam、
// 同一 pinned HEAD 提交,DL-04:读路径同样不旁路编译器、不重解析可动引用)。
// 任何类型化失败都是显式展示状态(page.CompiledNote),不是任务失败,更不是
// 「无风险/无保证」。
func v14TaskAuthority(env v13Env, g *v13Git, taskID string, page *V14TaskPage) *artifacts.CompiledAuthority {
	bundle, err := artifacts.LoadBundle(env.repoPath, g.headHash, taskID)
	if err != nil {
		page.CompiledNote = "已编译合同未接入:" + err.Error()
		return nil
	}
	auth, err := bundle.CompileAuthority()
	if err != nil {
		page.CompiledNote = "已编译合同未接入:" + err.Error()
		return nil
	}
	return auth
}

// v14TaskCoverage 沿 Change Detail 同一读路径(loadV13Change → deriveV13Coverage)
// 推导当前候选的功能 + independence 覆盖(PRD §14 coverage 维度的事实来源)。
// 候选缺席或读取失败都给显式 Notice,不阻断页面,更不伪造覆盖。
func (s *Server) v14TaskCoverage(ctx context.Context, env v13Env, csid evidence.ChangeSetID, page *V14TaskPage) *v13Coverage {
	if csid == "" {
		page.Coverage.Notice = "没有当前候选:候选由事实产生后,覆盖在此推导显示。"
		return nil
	}
	c, err := s.loadV13Change(ctx, env, changeset.ChangeSetID(csid))
	if err != nil {
		page.Coverage.Notice = "当前候选覆盖读取失败:" + err.Error()
		return nil
	}
	cov, err := deriveV13Coverage(c)
	if err != nil {
		page.Coverage = V13CoverageView{Notice: "覆盖推导失败:" + err.Error()}
		return nil
	}
	page.Coverage = v13CoverageView(c, cov)
	return cov
}

// v14AttemptViews 把 Run 行投影成尝试链:序号沿 EscalatedFrom 父链推导
// (根 = 1,修复尝试递增);父链断口(父行不在本任务的事实面里)按根处理
// 并保持确定性显示。
func v14AttemptViews(runs []runstore.Run) []V14AttemptView {
	byID := make(map[int64]runstore.Run, len(runs))
	for _, r := range runs {
		byID[r.ID] = r
	}
	depth := func(r runstore.Run) (int, bool) {
		n := 1
		cur := r
		for cur.EscalatedFrom != 0 {
			n++
			if n > v14MaxChainDepth {
				return n, true
			}
			parent, ok := byID[cur.EscalatedFrom]
			if !ok {
				break // 父行不在(跨任务/已被清理):按当前深度呈现
			}
			cur = parent
		}
		return n, false
	}

	out := make([]V14AttemptView, 0, len(runs))
	for _, r := range runs {
		n, trunc := depth(r)
		row := V14AttemptView{
			No:             n,
			ChainTruncated: trunc,
			ID:             strconv.FormatInt(r.ID, 10),
			Status:         string(r.Status),
			FailureReason:  r.FailureReason,
			Iterations:     r.Iterations,
			StartedAt:      v13FormatTime(r.StartedAt),
			FinishedAt:     v13FormatTime(r.FinishedAt),
			Claim:          r.AgentClaim,
			CandidateID:    r.ResultPatchHash,
		}
		out = append(out, row)
	}
	// 显示顺序:先按尝试序号,再按开始时刻;深度相同(独立根)时早的在前。
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].No != out[j].No {
			return out[i].No < out[j].No
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// v14CandidateViews 投影候选清单:验证计数、当前标记与取代事实。
// css 由 ChangeSetsForTask 给出(按创建时刻定序,末位即最新)。
func (s *Server) v14CandidateViews(ctx context.Context, runs *runstore.Store, css []*changeset.ChangeSet, vruns []runstore.VerificationRun) []V14CandidateView {
	out := make([]V14CandidateView, 0, len(css))
	for _, cs := range css {
		row := V14CandidateView{
			ID:          string(cs.ID()),
			CreatedAt:   v13FormatTime(cs.CreatedAt()),
			ProducerRun: cs.ProducerRun(),
			FileCount:   len(cs.Files()),
		}
		for _, vr := range vruns {
			if vr.ChangeSetID != cs.ID() {
				continue
			}
			switch vr.Status {
			case runstore.VerificationPass:
				row.VerifyPass++
			case runstore.VerificationFail:
				row.VerifyFail++
			case runstore.VerificationError:
				row.VerifyError++
			default:
				row.VerifyRunning = true
			}
		}
		facts, err := runs.ChangeSetSupersessions(ctx, cs.ID())
		if err == nil { // 取代事实读取失败不遮蔽候选本身,降级为空投影
			for _, f := range facts {
				if f.Old == cs.ID() {
					row.SupersededBy = append(row.SupersededBy, v13Short(string(f.New)))
				}
				if f.New == cs.ID() {
					row.Supersedes = append(row.Supersedes, v13Short(string(f.Old)))
				}
			}
		}
		out = append(out, row)
	}
	return out
}
