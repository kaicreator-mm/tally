// v14_change.go 是 v1.4 P3-09 的 Change Detail 页(PRD §26.3):一个候选一页,
// 回答四个问题 —— 改了什么、为什么可信、哪里不确定、(谱系)它从哪来。
//
// 数据纪律(与 v13_changes.go / v14_task.go 完全同源,本文件只做只读取数与投影):
//   - 候选上下文复用 loadV13Change 既有读取路径(immutable 快照、验证 Run 与
//     逐条证据+裁定、取代 lineage、lineage 暴露投影、Git diff、HEAD 制品里的
//     TaskContract / 冻结 VerificationContract)—— 不新建第二套读取面;
//   - 「改了什么」= 快照固定文件 + 从 Git base..candidate 确定性重建的 hunk
//     清单(review.InventoryFromDiff);diff 不可得时显式说明(§19.2),绝不给
//     一份静默残缺的清单;
//   - 「为什么可信」= 本候选的验证事实(证据 fact 引用原样呈现,INV-11)+
//     覆盖推导(功能 + independence,P1-19/P1-20 seam 原样投影)+ 结论依据
//     Basis 引用(taskstate.Derive,INV-STATE-02)。v1.4.1 P1-07:已编译合同
//     经与 worker 同一生产 seam(LoadBundle → CompileAuthority)只读接入,
//     VerifiedBasis 投影给出验证结论(VERIFIED / VERIFIED_LIMITED 可区分)与
//     必需/达成保证档,依据列表携带 compiled:<hash>;Compiled 缺席(装配类型
//     失败或候选无从归属)时如实标注「未接入」及其 fail-safe 语义,绝不冒充
//     —— VERIFIED_LIMITED 只能由 VerifiedBasis 投影产生,展示缺席不放宽任何
//     门(PRD §14);
//   - 「哪里不确定」= 派生门槛缺口、被裁定失效的证据、当前/祖先暴露(P0-07:
//     祖先暴露在当前候选干净时也不消失)、派生失败原因;全部可见,绝不隐藏。
//
// 路由经本文件的 registerV14ChangeAttentionRoutes 挂载;接入点是 v14_launch.go
// RegisterV14Routes 末尾的一行调用(P3-09 的最小注册编辑)。
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

	"github.com/go-chi/chi/v5"

	"tally/internal/artifacts"
	"tally/internal/core"
	"tally/internal/evidence"
	"tally/internal/review"
	"tally/internal/taskstate"
	assets "tally/web"
)

/* ---------- 渲染:P3-09 两页(Change Detail / Attention)共用装载 ---------- */

var (
	v14CATmplOnce sync.Once
	v14CATmpl     map[string]*template.Template
	v14CATmplErr  error
)

// v14CATemplate 解析(一次)并返回 base + P3-09 页面("CA" = Change/Attention)
// 的模板克隆。与 v13Template/v14Template 同一组合方式;模板函数复用 v13Short /
// v13FormatTime(纯函数)。刻意不复用 v14Template 的页面注册表:那是 P3-08
// 文件持有的 sync.Once,本文件只增不改它。
func v14CATemplate(name string) (*template.Template, error) {
	v14CATmplOnce.Do(func() {
		funcs := template.FuncMap{
			"short": v13Short,
			"ftime": v13FormatTime,
			"join":  strings.Join,
		}
		base, err := template.New("base").Funcs(funcs).ParseFS(assets.TemplateFS, "templates/base.tmpl")
		if err != nil {
			v14CATmplErr = err
			return
		}
		pages := []string{"v14_change", "v14_attention", "v14_rules", "v14_integration"}
		v14CATmpl = make(map[string]*template.Template, len(pages))
		for _, p := range pages {
			clone, err := base.Clone()
			if err != nil {
				v14CATmplErr = err
				return
			}
			if _, err := clone.ParseFS(assets.TemplateFS, "templates/"+p+".tmpl"); err != nil {
				v14CATmplErr = err
				return
			}
			v14CATmpl[p] = clone
		}
	})
	if v14CATmplErr != nil {
		return nil, v14CATmplErr
	}
	t, ok := v14CATmpl[name]
	if !ok {
		return nil, errors.New("v14: 未知页面模板 " + name)
	}
	return t, nil
}

// v14CARender 渲染一个 P3-09 页面;模板执行出错只记日志(头已发出),与 v13Render 一致。
func v14CARender(w http.ResponseWriter, status int, page string, data any) {
	tmpl, err := v14CATemplate(page)
	if err != nil {
		http.Error(w, "v1.4 页面模板不可用: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = tmpl.ExecuteTemplate(w, "base", data)
}

/* ---------- 路由注册 ---------- */

// registerV14ChangeAttentionRoutes 挂载 P3-09 的两条只读路由。生产接入点位于
// v14_launch.go 的 RegisterV14Routes(经 server.go 的 s.RegisterV14Routes(r)
// 进入登录 + CSRF 保护组)。只读:本页没有任何写路由。
func (s *Server) registerV14ChangeAttentionRoutes(r chi.Router) {
	r.Get("/v14/changes/{csid}", s.V14Change) // Change Detail(按候选)
	r.Get("/v14/attention", s.V14Attention)   // Attention 投影(按任务集)
}

/* ---------- 页面视图 ---------- */

// V14BasisDim 是结论依据(Basis)的一个维度行(PRD §14 VerifiedBasis 的
// coverage / assurance / unmatched / waiver 四维度)。已接入的维度带事实值;
// 未接入的维度 Wired=false 并带明确原因 —— 缺席必须可见,绝不静默略过。
type V14BasisDim struct {
	Name string // 维度名(coverage / assurance / unmatched / waiver)
	// Value 是已接入时的事实值(来自既有 seam 投影);未接入时为空。
	Value string
	// Wired 报告该维度在本读取路径是否已有事实来源。
	Wired bool
	// Note 是未接入原因与 fail-safe 语义声明(Wired=true 时可空)。
	Note string
}

// V14ChangePage 是 Change Detail 页数据(PRD §26.3 四问)。
type V14ChangePage struct {
	Page
	Unavailable bool // 数据源未接入
	NotFound    bool // 事实面里没有这个 ChangeSet
	LoadErr     string

	// 候选身份(immutable 快照固定字段)。
	ID          string
	Project     string
	Base        string
	Candidate   string
	PatchID     string
	ProducerRun string
	CreatedAt   string
	Entities    []string

	// 改了什么:快照固定内容 + Git 确定性重建的 hunk 清单。
	Files           []V13ChangedFile
	Inventory       []V13InvFile
	FileCount       int
	HunkCount       int
	InventoryNotice string

	// 为什么可信:本候选的验证事实(证据 fact 引用 + validity)与计数。
	Runs            []V13ChangeRun
	VerifyPass      int
	VerifyFail      int
	VerifyError     int
	InvalidEvidence int // 被裁定失效的证据行数(不确定面,可见不隐藏)

	// 归属任务上下文(HEAD 制品快照)。TaskID 为空 = 无验证 Run 且产生 Run
	// 记录缺失,候选归属无从建立 —— 显式呈现,不猜。
	HasTask  bool
	TaskID   string
	TaskLink bool // 归属任务可链接到 /v14/tasks/{task}(闭环任务详情)
	// ContractMatched 报告候选验证 Run 绑定的冻结契约与 HEAD 冻结契约是同一
	// 内容修订(§8.8.2:跨 revision 不混算;false 时页面显式标注)。
	ContractMatched bool

	// 覆盖(功能 + independence;与冻结契约同 revision 才可推导)。
	Coverage V13CoverageView

	// 结论依据(派生;INV-11):派生结论 + Basis 引用 + 四维度行。
	BasisAvailable bool
	State          taskstate.State
	ClassText      string
	WaivedClass    bool // strict_waived:豁免放行,不得显示成普通满足
	Basis          []string
	Gaps           []taskstate.Gap
	BasisDims      []V14BasisDim
	DeriveErr      string
	RowErr         string

	// v1.4.1 P1-07:已编译合同(P1-03 CompileAuthority,与 worker 同一生产
	// seam)接入后的验证结论投影(PRD §14)。Compiled 缺席时 HasVerdict=false,
	// 页面保持「未接入」标注与 fail-safe 语义声明,绝不冒充档位。
	HasVerdict bool
	// Verdict 是验证结论(VERIFIED / VERIFIED_LIMITED / NOT_VERIFIED)。
	Verdict taskstate.Verdict
	// VerdictLimited:VERIFIED_LIMITED 是 limited 结论(存在未匹配风险、
	// 被豁免条款或保证缺口),必须与普通 VERIFIED 区分展示(PRD §14)。
	VerdictLimited bool
	// VerdictOK:普通 VERIFIED(验证门全部满足且无未消化缺口)。
	VerdictOK bool
	// CompiledHash 是结论所依据的已编译合同身份哈希(sha256:<hex>;依据列表
	// 同时携带 compiled:<hash> 引用,INV-STATE-02)。
	CompiledHash string
	// CompiledNote 是编译权威未接入时的显式类型化原因;接入成功时为空。
	CompiledNote string

	// Current 报告本候选是派生的当前候选(taskstate.Derive 规范指针)。
	Current bool

	// 谱系:取代事实(append-only)+ lineage + 暴露投影(当前/祖先,P0-07)。
	SupersededBy     []string // 取代本候选的候选身份
	Supersedes       []string // 本候选取代的候选身份
	Lineage          []string // 候选在前,取代祖先随后(P0-07 确定性展开)
	Exposures        []V13ExposureRow
	AncestorExposure int
}

// V14Change 渲染 Change Detail(PRD §26.3):改了什么 / 为什么可信 / 哪里不确定 /
// 谱系与暴露。全部只读派生,没有写动作,没有 Agent 会话转录。
func (s *Server) V14Change(w http.ResponseWriter, r *http.Request) {
	csid := v13CSIDParam(r)
	page := V14ChangePage{Page: s.page(r, "Change Detail · v1.4", "v14"), ID: string(csid), Coverage: V13CoverageView{}}
	env, ok := s.v13env()
	if !ok {
		page.Unavailable = true
		v14CARender(w, http.StatusOK, "v14_change", page)
		return
	}
	c, err := s.loadV13Change(r.Context(), env, csid)
	if err != nil {
		if errors.Is(err, errV13NotFound) {
			page.NotFound = true
			v14CARender(w, http.StatusNotFound, "v14_change", page)
			return
		}
		page.LoadErr = err.Error()
		v14CARender(w, http.StatusOK, "v14_change", page)
		return
	}
	s.v14ChangePage(r.Context(), env, c, &page)
	v14CARender(w, http.StatusOK, "v14_change", page)
}

// v14ChangePage 装配 Change Detail:immutable 身份 + 改动面 + 验证面 + 依据 + 谱系。
func (s *Server) v14ChangePage(ctx context.Context, env v13Env, c *v13ChangeCtx, page *V14ChangePage) {
	cs := c.cs
	page.Project = string(cs.Project())
	page.Base = cs.BaseRevision()
	page.Candidate = cs.CandidateRevision()
	page.PatchID = cs.PatchIdentity()
	page.ProducerRun = cs.ProducerRun()
	page.CreatedAt = v13FormatTime(cs.CreatedAt())
	page.Entities = cs.Entities()
	page.ContractMatched = c.matched

	// ---- 改了什么:快照固定内容。
	for _, f := range cs.Files() {
		page.Files = append(page.Files, V13ChangedFile{Path: f.Path, Executable: f.Executable, Hunks: f.Hunks})
	}

	// ---- 验证面:每个验证 Run 与逐条证据(validity 经裁定投影,§8.8.1)。
	for _, run := range c.runs {
		vr := V13ChangeRun{
			ID:               strconv.FormatInt(run.ID, 10),
			Status:           string(run.Status),
			ContractRevision: run.ContractRevision,
			ContractHash:     run.ContractHash,
			EnvName:          run.EnvName,
			Runtime:          run.Runtime,
			StartedAt:        v13FormatTime(run.StartedAt),
			FinishedAt:       v13FormatTime(run.FinishedAt),
		}
		for _, ev := range c.evid[run.ID] {
			switch ev.Verdict {
			case evidence.VerdictPass:
				page.VerifyPass++
			case evidence.VerdictFail:
				page.VerifyFail++
			default:
				page.VerifyError++
			}
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
				page.InvalidEvidence++
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

	// ---- 谱系与暴露(P0-07):取代事实 + lineage + 当前/祖先暴露。
	if facts, err := env.runs.ChangeSetSupersessions(ctx, cs.ID()); err == nil {
		for _, f := range facts {
			switch {
			case f.Old == cs.ID():
				page.SupersededBy = append(page.SupersededBy, string(f.New))
			case f.New == cs.ID():
				page.Supersedes = append(page.Supersedes, string(f.Old))
			}
		}
	} else {
		page.RowErr = "读取取代事实失败: " + err.Error()
	}
	for _, id := range c.lineage {
		page.Lineage = append(page.Lineage, string(id))
	}
	page.Exposures = v13ExposureRows(c.exposures)
	page.AncestorExposure = v13AncestorExposureCount(c.exposures)

	// ---- 覆盖推导(P1-19/P1-20 seam 原样投影;未匹配冻结契约时给显式 Notice)。
	cov, err := deriveV13Coverage(c)
	if err != nil {
		page.Coverage.Notice = "覆盖推导失败:" + err.Error()
	} else {
		page.Coverage = v13CoverageView(c, cov)
	}

	// ---- 确定性 hunk 清单:候选有 candidate revision 时从 Git 重建;只有 patch
	// 身份的候选没有可比对的 Git revision,显式说明(§19.2),绝不静默残缺。
	if c.git != nil && cs.CandidateRevision() != "" {
		diff, err := c.git.DiffText(cs.BaseRevision(), cs.CandidateRevision())
		switch {
		case err != nil:
			page.InventoryNotice = "无法从制品仓库重建 base..candidate diff:" + err.Error()
		default:
			inv, err := review.InventoryFromDiff(diff)
			if err != nil {
				page.InventoryNotice = "diff 无法解析为确定性清单:" + err.Error()
			} else {
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
		}
	} else if cs.CandidateRevision() == "" {
		page.InventoryNotice = "该候选没有固定 candidate revision(仅 patch 身份),无法从 Git 重建确定性 hunk 清单。"
	} else {
		page.InventoryNotice = "制品仓库不可用,无法从 Git 重建确定性 hunk 清单。"
	}

	// ---- 归属任务:优先验证 Run 记录的任务,回退产生 Run 的任务(与读取层
	// 同一判据);两者都缺席 = 归属无从建立,页面显式呈现,不猜。
	taskID := c.taskID
	if taskID == "" {
		taskID = c.producerTask
	}
	page.TaskID = taskID
	page.TaskLink = taskID != ""
	if c.task != nil {
		page.HasTask = true
	}

	// ---- 结论依据:沿闭环任务详情的同一装配派生(制品事实 + 事实面行)。
	s.v14ChangeBasis(ctx, env, c, taskID, cov, page)
}

// v14ChangeBasis 用与闭环任务详情(v14TaskPage)同一装配派生归属任务的结论,
// 投影成本候选的依据面:当前候选标记、结论类别(satisfied/waived 可区分)、
// Basis 引用(INV-11)、门槛缺口;四维度行(coverage/assurance/unmatched/
// waiver)在派生之前先装配 —— 未接入的维度不依赖派生成败。
//
// v1.4.1 P1-07:编译权威经与 worker 完全相同的 P1-03 生产 seam
// (artifacts.LoadBundle → ExecutionBundle.CompileAuthority)从同一 pinned
// HEAD 提交只读取数(PRD §16:读路径同样不旁路编译器)。Compiled 接入后
// VerifiedBasis 投影产出验证结论与保证档,依据列表携带 compiled:<hash>
// (INV-STATE-02);任何类型化装配失败 = 本读取路径没有已编译合同,页面
// 保持「未接入」标注,绝不冒充档位(展示缺席不放宽任何验证门)。
func (s *Server) v14ChangeBasis(ctx context.Context, env v13Env, c *v13ChangeCtx, taskID string, cov *v13Coverage, page *V14ChangePage) {
	page.BasisDims = v14BasisDims(page.Coverage, page.ClassText, page.WaivedClass, nil)

	if taskID == "" || c.git == nil {
		return // 无归属任务或 Git 缺席:派生面无从投影,页面已显式标注
	}
	snap, err := artifacts.Load(env.repoPath, c.git.headHash)
	if err != nil {
		page.RowErr = "制品装载失败: " + err.Error()
		return
	}
	// project 身份来自 Git 注册表(.tally/project.yaml)—— 与闭环任务详情
	// (v14TaskPage)同一判据。变更页此前从未填充它,v13env 按注释「不猜」
	// 留空,任务作用域的 run/candidate/verification 事实永远查不出来,派生
	// 只能停在 READY;读模型的结论要可追溯到当前候选,这里必须补齐。
	env.project = core.ProjectID(snap.ProjectID)
	var ta *artifacts.TaskArtifact
	for i := range snap.Tasks {
		if snap.Tasks[i].Contract.ID == taskID {
			ta = &snap.Tasks[i]
			break
		}
	}
	if ta == nil {
		return // HEAD 快照没有该任务(候选早于当前 HEAD 制品):不猜测归属
	}
	va := v13FrozenFor(snap, taskID)

	// ---- 编译权威(只读;与 worker 同一 seam,同一 pinned 提交)。失败是
	// 显式展示状态,不是候选失败,更不是「无风险/无保证」。
	var auth *artifacts.CompiledAuthority
	if bundle, err := artifacts.LoadBundle(env.repoPath, c.git.headHash, taskID); err != nil {
		page.CompiledNote = "已编译合同未接入:" + err.Error()
	} else if a, err := bundle.CompileAuthority(); err != nil {
		page.CompiledNote = "已编译合同未接入:" + err.Error()
	} else {
		auth = a
	}

	runs, err := env.runs.CodegenRunsForTask(ctx, env.project, taskID)
	if err != nil {
		page.RowErr = err.Error()
		return
	}
	css, err := env.runs.ChangeSetsForTask(ctx, env.project, taskID)
	if err != nil {
		page.RowErr = err.Error()
		return
	}
	vruns, err := env.runs.VerificationRunsForTask(ctx, env.project, taskID)
	if err != nil {
		page.RowErr = err.Error()
		return
	}
	approvals, err := env.approvals.ApprovalsForTask(ctx, env.project, taskID)
	if err != nil {
		page.RowErr = err.Error()
		return
	}
	derive := taskstate.Snapshot{
		Contract:  v13FrozenContract(va),
		Facts:     v13FactsFor(ta, va, c.git.headTime, runs, css, vruns),
		Approvals: v13ApprovalEvidence(approvals),
	}
	if auth != nil {
		derive.Compiled = auth.Compiled
		// 覆盖报告只在既有的同修订推导(cov,§8.8.2 修订绑定)可用时参与:
		// 报告缺席 = 达成档没有事实来源,投影按 L0 fail-safe,不冒充覆盖。
		if cov != nil {
			derive.Functional = &cov.Functional
			derive.Independence = &cov.Independence
		}
	}
	res, err := taskstate.Derive(derive)
	if err != nil {
		page.DeriveErr = err.Error()
		return
	}
	page.BasisAvailable = true
	page.State = res.State
	page.ClassText = string(res.VerificationClass)
	page.WaivedClass = res.VerificationClass == taskstate.ClassStrictWaived
	page.Basis = res.Basis
	page.Gaps = res.Gaps
	page.Current = res.CurrentCandidate != "" && res.CurrentCandidate == evidence.ChangeSetID(page.ID)
	// v1.4.1 P1-07:验证结论投影(仅 Compiled 接入时产出;taskstate 对
	// Compiled 与当前冻结契约跨修订显式报错,绝不静默降级)。
	if vb := res.VerifiedBasis; vb != nil {
		page.HasVerdict = true
		page.Verdict = vb.Verdict
		page.VerdictLimited = vb.Verdict == taskstate.VerdictVerifiedLimited
		page.VerdictOK = vb.Verdict == taskstate.VerdictVerified
		page.CompiledHash = vb.CompiledHash
	}
	// 派生后依据维度行要带上结论类别与(接入时)VerifiedBasis 事实值。
	page.BasisDims = v14BasisDims(page.Coverage, page.ClassText, page.WaivedClass, res.VerifiedBasis)
}

// v14BasisDims 装配结论依据的四维度行(PRD §14 的 coverage/assurance/unmatched/
// waiver)。已接入的维度给事实值;未接入的维度给明确原因与 fail-safe 语义声明
// —— 展示缺席绝不放宽任何验证门(unknown/unmatched 风险在上游仍按 fail-safe
// 升档,INV-V14-09)。
//
// v1.4.1 P1-07:vb 非 nil(已编译合同接入)时,assurance / unmatched 维度从
// VerifiedBasis 投影取事实值(必需/达成档、命中策略、未匹配与 unknown 信号
// 原样可见);vb 为 nil 时保持「未接入」标注 —— Compiled 缺席绝不冒充档位。
func v14BasisDims(cov V13CoverageView, classText string, waivedClass bool, vb *taskstate.VerifiedBasis) []V14BasisDim {
	// coverage:功能 + independence 覆盖推导(P1-19/P1-20 seam)已接入;与
	// 冻结契约同 revision 才可推导,不可推导时给显式 Notice(§8.8.2)。
	covDim := V14BasisDim{Name: "coverage"}
	if cov.Available {
		fc, ic := 0, 0
		for _, r := range cov.Functional {
			if r.Covered {
				fc++
			}
		}
		for _, r := range cov.Independence {
			if r.Covered {
				ic++
			}
		}
		covDim.Wired = true
		covDim.Value = fmt.Sprintf("功能 %d/%d · independence %d/%d(逐条款见覆盖表)",
			fc, len(cov.Functional), ic, len(cov.Independence))
	} else {
		covDim.Note = "覆盖不可推导:" + cov.Notice
	}

	// assurance:必需/达成保证档(L0-L3)聚合由 VerifiedBasis 投影给出;
	// Compiled 未接入时不冒充档位。
	assuranceDim := V14BasisDim{Name: "assurance"}
	if vb != nil {
		assuranceDim.Wired = true
		assuranceDim.Value = fmt.Sprintf("必需 %s · 达成 %s(达成按最弱条款诚实上报)",
			vb.AssuranceRequired, vb.AssuranceAchieved)
		assuranceDim.Note = "保证档来自已编译合同的 VerifiedBasis 投影(PRD §14),可经依据列表的 compiled:<hash> 单独复算。"
	} else {
		assuranceDim.Note = "保证档(required/achieved)由 VerifiedBasis(PRD §14)投影,需要已编译合同(contract.Bind)接入生产组合根;未接入,本页不冒充档位。"
	}

	// unmatched:未匹配风险信号来自已编译合同的策略匹配投影;unknown 信号
	// 原样保真(INV-V14-09)。
	unmatchedDim := V14BasisDim{Name: "unmatched"}
	if vb != nil {
		unmatchedDim.Wired = true
		switch {
		case len(vb.UnmatchedRiskSignals) > 0:
			ids := make([]string, 0, len(vb.UnmatchedRiskSignals))
			for _, id := range vb.UnmatchedRiskSignals {
				ids = append(ids, string(id))
			}
			unmatchedDim.Value = fmt.Sprintf("命中策略 %d;未匹配风险信号:%s(fail-safe:全部条款必需档升 L3)",
				vb.PolicyCoverage.MatchedCount, strings.Join(ids, ", "))
		case len(vb.PolicyCoverage.UnknownRiskSignals) > 0:
			ids := make([]string, 0, len(vb.PolicyCoverage.UnknownRiskSignals))
			for _, id := range vb.PolicyCoverage.UnknownRiskSignals {
				ids = append(ids, string(id))
			}
			unmatchedDim.Value = fmt.Sprintf("命中策略 %d;无未匹配风险信号;unknown 信号(冻结枚举之外,原样保真):%s",
				vb.PolicyCoverage.MatchedCount, strings.Join(ids, ", "))
		default:
			unmatchedDim.Value = fmt.Sprintf("命中策略 %d,无未匹配风险信号", vb.PolicyCoverage.MatchedCount)
		}
	} else {
		unmatchedDim.Note = "未匹配风险信号需要已编译合同的策略匹配投影;未接入。unknown/unmatched 风险 fail-safe(升 L3,INV-V14-09)不受展示缺席影响,VERIFIED_LIMITED 不会被显示成普通 VERIFIED。"
	}

	// waiver:已接入的部分是结论类别(strict_satisfied / strict_waived 可区分,
	// INV-07);waiver 证据本身在 UI 读取路径尚无投影来源(WaiverPermitted
	// 保守取不允许),被豁免条款永不得显示成普通满足。
	waiverDim := V14BasisDim{
		Name: "waiver",
		Note: "waiver 证据在 UI 读取路径尚无投影来源(WaiverPermitted 保守取不允许);strict_waived 与 strict_satisfied 永不合并,豁免条款不得显示成普通满足。",
	}
	switch {
	case classText == "":
		waiverDim.Value = "—(结论未到 VERIFIED,无 waiver 维度可判)"
		waiverDim.Wired = true
	case waivedClass:
		waiverDim.Value = "strict_waived(经 Waiver 放行,不是严格满足)"
		waiverDim.Wired = true
	default:
		waiverDim.Value = classText + "(无 Waiver 参与)"
		waiverDim.Wired = true
	}

	return []V14BasisDim{covDim, assuranceDim, unmatchedDim, waiverDim}
}
