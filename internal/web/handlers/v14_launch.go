// v14_launch.go 是 v1.4 P3-08 的最小 Launch 面(PRD §26.1 / §27;决策锁 19:
// minimal UI —— 只有 Launch / 闭环详情 / Change / Attention,没有 Jira 式 IA)。
//
// 组成(全部只读事实 + 两个有界写出口,不发明任何新状态权威):
//  1. 闭环任务列表:Git HEAD 快照(.tally TaskContract / 冻结 VerificationContract,
//     经 internal/artifacts 装载)+ 每行 taskstate 派生状态(复用 v13_tasks.go
//     同一装配 v13TaskRow,INV-STATE-01:状态是派生的)。standard profile 且
//     已冻结验证契约的任务可以 POST 入队 v14_task 闭环 job(P2-09,cmd/worker
//     的 KindV14Task 线格式;界面只入队不执行,INV-5)。
//  2. 自然语言意图入口(P3-07 seam):一句意图(+可选 source_ref)经
//     internal/intake.Propose 走 AI Runtime 边界结构化;输出要么是给 owner
//     确认/编辑的 DRAFT 草案,要么是澄清问题 —— 二者在 owner 确认(第 3 条)
//     之前都绝不是权威合同(INV-02:验收在执行前冻结)。
//  3. 确认→冻结(v1.4.1 P0-08):owner 对草案的确认经 internal/intakefreeze
//     的冻结准入(Freeze:确认事实 + owner 声明的 planned evidence →
//     FrozenIntent + TaskContract + 冻结 VerificationContract),再经
//     internal/artifacts.FreezeCommit 以**单一期望基线**经 internal/gitwriter
//     原子提交为**一个** Git 提交(.tally/intents + .tally/tasks +
//     .tally/verifications 各一份,共享同一任务身份)。Git 写入只走
//     gitwriter 单一写边界(DL-03),失败不留半套制品,基线冲突类型化报错
//     而不是静默改基;重复确认按内容幂等(ErrNoChanges)。提交成功后即可由
//     artifacts.Load/LoadIntent 从 pinned 提交确定性读回(P0-02)。
//
// 渲染与 v1.3 页面同一模式(共享 base.tmpl,本文件内自行解析;internal/web
// 的 render.go 不在本任务 Allowed paths 内)。路由注册提供 RegisterV14Routes,
// 由 composition root 挂进登录 + CSRF 组(见函数注释)。
package handlers

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"

	"tally/internal/airuntime"
	"tally/internal/artifacts"
	"tally/internal/core"
	"tally/internal/gitwriter"
	"tally/internal/intake"
	"tally/internal/intakefreeze"
	"tally/internal/intentevidence"
	"tally/internal/jobs"
	"tally/internal/verification"
	assets "tally/web"
)

/* ---------- 渲染:v1.4 页面模板(共享 base.tmpl) ---------- */

var (
	v14TmplOnce sync.Once
	v14Tmpl     map[string]*template.Template
	v14TmplErr  error
)

// v14Template 解析(一次)并返回 base + 指定 v1.4 页面的模板克隆。
// 与 v13Template 同一组合方式;模板函数复用 v13Short / v13FormatTime(纯函数)。
func v14Template(name string) (*template.Template, error) {
	v14TmplOnce.Do(func() {
		funcs := template.FuncMap{
			"short": v13Short,
			"ftime": v13FormatTime,
			"join":  strings.Join,
			"has":   v14StringIn,
		}
		base, err := template.New("base").Funcs(funcs).ParseFS(assets.TemplateFS, "templates/base.tmpl")
		if err != nil {
			v14TmplErr = err
			return
		}
		pages := []string{"v14_launch", "v14_task"}
		v14Tmpl = make(map[string]*template.Template, len(pages))
		for _, p := range pages {
			clone, err := base.Clone()
			if err != nil {
				v14TmplErr = err
				return
			}
			if _, err := clone.ParseFS(assets.TemplateFS, "templates/"+p+".tmpl"); err != nil {
				v14TmplErr = err
				return
			}
			v14Tmpl[p] = clone
		}
	})
	if v14TmplErr != nil {
		return nil, v14TmplErr
	}
	t, ok := v14Tmpl[name]
	if !ok {
		return nil, errors.New("v14: 未知页面模板 " + name)
	}
	return t, nil
}

// v14Render 渲染一个 v1.4 页面;模板执行出错只记日志(头已发出),与 v13Render 一致。
func v14Render(w http.ResponseWriter, status int, page string, data any) {
	tmpl, err := v14Template(page)
	if err != nil {
		http.Error(w, "v1.4 页面模板不可用: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = tmpl.ExecuteTemplate(w, "base", data)
}

/* ---------- AI Runtime 推理出口(组合根注入缝) ---------- */

// intake.Propose 需要 airuntime.Executor(P3-07 合同)。handlers.Server 的字段
// 集合由 server.go 定义(共享热点,本任务不持有),因此推理出口走包级注入缝:
// composition root 启动时调用 SetV14ProposeExecutor;缺席(nil)时意图入口
// 渲染「AI Runtime 未接入」状态并给出明确行动提示,绝不假装可用。
var (
	v14ProposeMu  sync.RWMutex
	v14ProposeRun airuntime.Executor
)

// SetV14ProposeExecutor 注入/清除(intake.Propose 的)通用推理出口。
// 生产装配点:server 组合根;测试注入 airuntime.FakeExecutor。
func SetV14ProposeExecutor(x airuntime.Executor) {
	v14ProposeMu.Lock()
	defer v14ProposeMu.Unlock()
	v14ProposeRun = x
}

func v14ProposeExecutor() airuntime.Executor {
	v14ProposeMu.RLock()
	defer v14ProposeMu.RUnlock()
	return v14ProposeRun
}

/* ---------- Launch 页数据 ---------- */

// v14TaskJobKind 是 v1.4 闭环 job 的线格式 kind,与 cmd/worker/v14.go 的
// KindV14Task("v14_task")是同一字符串契约:界面只入队,worker 恒能执行
// (注册无开关)。kind 字符串由两侧各自声明,jobs.job.kind 是自由 TEXT。
const v14TaskJobKind jobs.Kind = "v14_task"

// v14RunJobPayload 是闭环 job 的入队参数,与 cmd/worker 的 V14TaskPayload
// 同一线格式:合同本体以 Git HEAD 快照为权威,payload 只带坐标。
type v14RunJobPayload struct {
	Task string `json:"task"`
}

// V14LaunchTaskRow 是 Launch 列表的一行:复用 v1.3 Tasks 页的同一派生装配
// (V13TaskRow:taskstate 派生状态 / 当前 Run / 当前候选 / 验证摘要),
// 附加本页的闭环可启动标记。
type V14LaunchTaskRow struct {
	V13TaskRow
	// Launchable 报告该行可以入队 v14_task 闭环 job:
	// standard profile 且 HEAD 快照内有冻结 VerificationContract
	// (闭环路径的执行前提,cmd/worker runV14Task 的同一判据)。
	Launchable bool
}

// V14QuestionView 是一条澄清问题的展示(结构化选项优先,PRD §27)。v1.4.1 起
// 它承载有界 Dynamic Clarification Schema 的渲染面(DL-11 / PRD §3.2):每种
// 字段类型渲染为熟悉的原生表单控件,advisory 推荐只标注不预选(PRD §8.3:
// 建议不是决定,提交值永远来自 owner),number 渲染声明区间。全部值都是纯文本
// 数据(html/template 自动转义),没有任何 markup/脚本通道。
type V14QuestionView struct {
	Index      int
	Question   string
	Candidates []string
	// Kind 是有界字段类型(封闭词表,见 intake.ClarificationKind;
	// 空 = 未携带 schema 的旧简单形态,照旧渲染候选 + 自由作答)。
	Kind intake.ClarificationKind
	// KindLabel 是类型的人读标签(仅展示)。
	KindLabel string
	// advisory 推荐项(HasRecommended 才有效;RecommendedChoice 是对应候选
	// 文本,模板不按下标取值,避免越界求值)。
	HasRecommended       bool
	Recommended          int
	RecommendedChoice    string
	RecommendationReason string
	// KindNumber 的声明闭区间(HasMin/HasMax 才有效):既渲染为 input 约束,
	// 也是提交后服务端复核的依据。
	HasMin bool
	Min    float64
	HasMax bool
	Max    float64
}

// V14AcceptanceEdit 是草案确认表单里一条可编辑的验收条款(id 固定,陈述可改),
// 以及 owner 为该条款声明的 planned evidence(P0-08 冻结准入输入;空 = 未声明,
// 缺失证据会被 intakefreeze 的遗漏防护类型化阻断)。
type V14AcceptanceEdit struct {
	ID        string
	Statement string
	// EvidenceKind/EvidencePath/EvidenceSelector 是确认时声明的可见证据定位
	// (类型取自 verification 的封闭词表;本确认面只声明 repo 路径型证据,
	// held-out 与 validator 形态不在本面上,由程序化 API 承载)。
	EvidenceKind     string
	EvidencePath     string
	EvidenceSelector string
}

// V14DraftView 是 intake 草案的可确认/可编辑展示(AC2:structured Contract diff
// confirmable/editable)。多值字段压成多行文本供 textarea 编辑;它只是给 owner
// 的 DRAFT,owner 确认提交后才经冻结准入成为 Git 里的权威冻结制品。
type V14DraftView struct {
	TaskType       string
	Summary        string
	Objective      string
	AllowedPaths   string // 多行文本(每行一条)
	ForbiddenPaths string
	NonScope       string
	Constraints    string
	Acceptance     []V14AcceptanceEdit
	RiskSignals    []string // 候选风险信号:只展示,不作数
	SourceRef      string
	// TaskID 是确认时指定的目标任务身份(TASK-<数字>;冻结三件套共享)。
	TaskID string
	// Profile 是确认时显式选择的治理等级(light/standard/strict);映射器
	// 不设默认值,缺席会被冻结准入类型化拒绝。
	Profile string
	// RiskHints 是 owner 确认保留的风险提示(勾选;未勾选的候选只是展示,
	// 绝不因确认而自动成为证据,DL-06)。
	RiskHints []string
}

// V14LaunchPage 是 Launch 页数据。
type V14LaunchPage struct {
	Page
	// Unavailable:v1.4 数据源(事实面 / 制品仓库)未接入。
	Unavailable bool
	// LoadErr:Git 制品装载失败(HEAD 缺失、注册表缺失等)。
	LoadErr      string
	ProjectID    string
	SourceCommit string
	Tasks        []V14LaunchTaskRow

	// ExecutorWired 报告 AI Runtime 推理出口是否已注入;缺席时意图入口
	// 渲染不可用状态而不是假装提交。
	ExecutorWired bool
	// ProposeErr 是意图结构化失败的类型化原因(inline 呈现,不整页 500)。
	ProposeErr string
	// Intent / SourceRef 回填表单。
	Intent    string
	SourceRef string
	// Questions 非空 = 上次提案是澄清请求(AC3:澄清是一等产物)。
	Questions []V14QuestionView
	// Draft 非空 = 上次提案产出了完整草案(可确认/可编辑)。
	Draft *V14DraftView
}

// RegisterV14Routes 注册 v1.4 最小界面的全部路由。挂在登录 + CSRF 保护组内
// (与 handlers.Server.RegisterRoutes 同一要求);composition root 在
// RegisterRoutes 末尾调用本函数即可 —— 该文件是共享热点,不在 P3-08 的
// Allowed paths 内,接入编辑由持有它的任务/整合者完成。
func (s *Server) RegisterV14Routes(r chi.Router) {
	r.Get("/v14/launch", s.V14Launch)                // Launch 列表 + 意图入口
	r.Post("/v14/launch/intent", s.V14ProposeIntent) // 意图结构化(P3-07 seam)
	r.Post("/v14/launch/confirm", s.V14ConfirmDraft) // 草案确认(仅本页生效)
	r.Post("/v14/launch/run", s.V14LaunchRun)        // 入队 v14_task 闭环 job
	r.Get("/v14/tasks/{task}", s.V14Task)            // 闭环任务详情
	r.Get("/v14/rules", s.V14Rules)                  // P3-06:Rules 投影(Active 只从 Git 快照派生;治理面分歧/不确定性一等可见)
	r.Get("/v14/integration", s.V14Integration)      // P5-05:集成审计投影(验收与集成是两个可分别观察的状态)
	s.registerV14ChangeAttentionRoutes(r)            // P3-09: Change Detail + Attention(最小注册接入点,实现见 v14_change.go)
}

// V14Launch 渲染 Launch 页(PRD §26.1):闭环任务列表 + 自然语言意图入口。
func (s *Server) V14Launch(w http.ResponseWriter, r *http.Request) {
	page := s.v14LaunchPage(r)
	v14Render(w, http.StatusOK, "v14_launch", page)
}

// v14LaunchPage 组装 Launch 页公共数据:AI Runtime 接入状态 + 尽力而为的
// 闭环任务列表。数据源缺席时只把列表区标为未接入(Unavailable),意图 /
// 草案面板照常渲染 —— 它们不依赖本地事实面。
func (s *Server) v14LaunchPage(r *http.Request) *V14LaunchPage {
	page := &V14LaunchPage{Page: s.page(r, "Launch · v1.4 闭环", "v14-launch")}
	page.ExecutorWired = v14ProposeExecutor() != nil
	env, ok := s.v13env()
	if !ok {
		page.Unavailable = true
		return page
	}
	g, err := openV13Git(env.repoPath)
	if err != nil {
		page.LoadErr = err.Error()
		return page
	}
	snap, err := artifacts.Load(env.repoPath, g.headHash)
	if err != nil {
		page.LoadErr = err.Error()
		return page
	}
	env.project = core.ProjectID(snap.ProjectID)
	page.ProjectID = snap.ProjectID
	page.SourceCommit = g.headHash

	ctx := r.Context()
	for _, ta := range snap.Tasks {
		if ta.Contract.Profile != "standard" {
			continue // 闭环路径只执行 standard TaskContract(cmd/worker 同一判据)
		}
		row := V14LaunchTaskRow{V13TaskRow: s.v13TaskRow(ctx, env, g, snap, ta)}
		row.Launchable = row.HasFrozen && row.DeriveErr == ""
		page.Tasks = append(page.Tasks, row)
	}
	return page
}

/* ---------- 意图结构化(P3-07 seam) ---------- */

// V14ProposeIntent 把一句自然语言意图结构化为草案或澄清请求(intake.Propose;
// AI Runtime 边界)。POST 后直接渲染结果页:提案调用有幂等键,刷新重复提交
// 会在 Runtime 侧命中同一 Operation,不产生第二份权威状态。
func (s *Server) V14ProposeIntent(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "表单解析失败", http.StatusBadRequest)
		return
	}
	page := s.v14LaunchPage(r)
	page.Intent = strings.TrimSpace(r.FormValue("intent"))
	page.SourceRef = strings.TrimSpace(r.FormValue("source_ref"))

	// 澄清问答折叠:结构化选项优先(PRD §27),自由文本兜底;答案原样并回
	// 意图文本 —— 新文本 = 新幂等键 = 新的逻辑提案调用(intake 的合同)。
	folded, qs, qErr := v14FoldClarifications(r)
	if qErr != "" {
		page.ProposeErr = qErr
		page.Questions = qs
		v14Render(w, http.StatusBadRequest, "v14_launch", page)
		return
	}
	intentText := page.Intent + folded

	switch {
	case intentText == "":
		page.ProposeErr = "意图文本为空:一句自然语言意图是唯一必填项(PRD §27:自然语言优先,不做 Jira 式必填表单)。"
		v14Render(w, http.StatusBadRequest, "v14_launch", page)
		return
	case !page.ExecutorWired:
		page.ProposeErr = "AI Runtime 推理出口未接入:意图结构化需要组合根注入 airuntime.Executor(P3-07 合同)。接入前本页不会假装提案可用。"
		v14Render(w, http.StatusOK, "v14_launch", page)
		return
	}

	proposal, err := intake.Propose(r.Context(), v14ProposeExecutor(), intake.Intent{
		Text:      intentText,
		SourceRef: page.SourceRef,
	})
	if err != nil {
		page.ProposeErr = v14ProposeErrText(err)
		v14Render(w, http.StatusBadRequest, "v14_launch", page)
		return
	}
	switch p := proposal.(type) {
	case intake.DraftProposal:
		page.Draft = v14DraftViewOf(p.Draft, page.SourceRef)
	case intake.ClarificationProposal:
		page.Questions = v14QuestionViews(p.Questions)
	}
	v14Render(w, http.StatusOK, "v14_launch", page)
}

// v14ProposeErrText 把 intake 的类型化错误映射成人读原因(errors.Is 判类,
// 不做字符串匹配);未映射的错误原样透出(其文本本就面向操作者)。
func v14ProposeErrText(err error) string {
	switch {
	case errors.Is(err, intake.ErrEmptyIntent):
		return "意图文本为空:一句自然语言意图是唯一必填项。"
	case errors.Is(err, intake.ErrIntentTooLong):
		return "意图文本超过长度上限(有界输入,防失控提示词)。"
	case errors.Is(err, intake.ErrNoExecutor):
		return "AI Runtime 推理出口未注入,无法提案。"
	case errors.Is(err, intake.ErrUnsupportedTaskType):
		return "推理给出的任务类型不在 MVP 支持范围(bug_fix / db_migration),已按 fail-safe 拒绝,不猜测归类。"
	case errors.Is(err, intake.ErrInvalidProposal):
		return "提案输出形状非法,已按 fail-safe 拒绝(含糊时不得发明验收):" + err.Error()
	case errors.Is(err, intake.ErrProposeUncertain):
		return "AI Runtime 自报拿不准(UNCERTAIN),结果未采信:请补充意图后重试。"
	case errors.Is(err, intake.ErrProposeCancelled):
		return "提案操作被取消(取消是一等终态,未被编造成失败):可重新提交。"
	case errors.Is(err, intake.ErrProposeFailed):
		return "提案操作被 AI Runtime 判定失败:" + err.Error()
	default:
		return err.Error()
	}
}

// 澄清折叠的有界上限(与 intake 侧 schema 同一量纲;超限类型化拒绝,不截断):
// 答案文本 ≤ 500 B,多选最多勾 16 项。
const (
	maxClarificationAnswerLen = 500
	maxClarificationSelected  = 16
)

// v14FoldClarifications 从表单读澄清问答(q<i> 问题 + k<i> 字段类型 + c<i>
// 候选 + min/max 区间 + r/rr advisory + a<i> 选项答案 / f<i> 自由答案),按
// 有界 schema **服务端复核每个提交值**(任务合同:Server revalidates all
// submitted values),复核通过才折叠成追加到意图文本的段落。未携带 k<i> 的
// 旧形态表单保持既有行为(自由文本优先,选项原样作答)。
//
// 返回追加文本与(未答/复核失败时)需要重答的问题列表 —— 重答视图从表单隐藏
// 域重建,owner 已看到的 schema(候选/区间/推荐提示)不丢。
func v14FoldClarifications(r *http.Request) (extra string, qs []V14QuestionView, errMsg string) {
	idx := v14HiddenIndices(r.Form, "v14q")
	if len(idx) == 0 {
		return "", nil, ""
	}
	var (
		b        strings.Builder
		reanswer []V14QuestionView
		errs     []string
	)
	b.WriteString("\n\n[澄清问答]\n")
	for _, i := range idx {
		q := strings.TrimSpace(r.FormValue("v14q." + i))
		if q == "" {
			continue
		}
		cf, perr := v14ClarificationFormOf(r, i, q)
		if perr != "" {
			errs = append(errs, "问题 #"+i+": "+perr)
			continue
		}
		answer, aerr := v14ClarificationAnswer(r, cf)
		switch {
		case aerr != "":
			errs = append(errs, "问题 #"+i+": "+aerr)
			reanswer = append(reanswer, cf.view)
		case answer == "":
			reanswer = append(reanswer, cf.view)
		default:
			b.WriteString("Q: " + q + "\nA: " + answer + "\n")
		}
	}
	if len(errs) > 0 {
		return "", reanswer, "澄清答案未通过服务端复核(有界 schema,fail-safe,不猜测修正):" + strings.Join(errs, "; ")
	}
	if len(reanswer) > 0 {
		return "", reanswer, "还有澄清问题未回答;回答全部问题后才会产出草案(含糊时绝不发明验收)。"
	}
	return b.String(), nil, ""
}

// v14ClarificationForm 是从提交表单重建的单条澄清问答 schema。提交侧没有权威
// schema 可查(每轮澄清都由推理新生成),因此表单隐藏域携带上一轮渲染的
// schema;服务端先对 schema 本身做 fail-safe 校验(词表/区间/可解析),再用它
// 复核提交的答案。
type v14ClarificationForm struct {
	view       V14QuestionView          // 未答/复核失败时的重渲染视图
	kind       intake.ClarificationKind // "" = 未携带 schema 的旧形态表单
	candidates []string
	min, max   *float64
}

// v14ClarificationFormOf 重建下标 idx 的问题 schema;schema 本身非法(词表外
// kind、区间倒挂、界值坏值)返回类型化原因 —— 绝不猜测修正(DL-11)。
func v14ClarificationFormOf(r *http.Request, idx, question string) (v14ClarificationForm, string) {
	cf := v14ClarificationForm{view: V14QuestionView{Question: question}}
	n, err := strconv.Atoi(idx)
	if err != nil {
		return cf, "问题下标非法"
	}
	cf.view.Index = n

	kind := strings.TrimSpace(r.FormValue("v14k." + idx))
	if kind == "" {
		return cf, "" // 旧形态表单:保持既有行为,答案按自由文本折叠
	}
	k := intake.ClarificationKind(kind)
	switch k {
	case intake.KindChoice, intake.KindMultiChoice, intake.KindBoolean, intake.KindNumber,
		intake.KindRepository, intake.KindBranch, intake.KindRiskChoice, intake.KindConfirmation, intake.KindFreeText:
		cf.kind = k
	default:
		return cf, fmt.Sprintf("字段类型 %q 不在有界词表内(DL-11 fail-safe)", kind)
	}
	cf.view.Kind = k
	cf.view.KindLabel = v14KindLabel(k)

	for _, c := range r.Form["v14c."+idx] {
		if c = strings.TrimSpace(c); c != "" {
			cf.candidates = append(cf.candidates, c)
		}
	}
	cf.view.Candidates = cf.candidates

	// advisory 推荐只是渲染提示:任何坏值都直接丢弃提示,不影响答案复核。
	if s := strings.TrimSpace(r.FormValue("v14r." + idx)); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 0 && n < len(cf.candidates) {
			cf.view.HasRecommended = true
			cf.view.Recommended = n
			cf.view.RecommendedChoice = cf.candidates[n]
			if reason := strings.TrimSpace(r.FormValue("v14rr." + idx)); len(reason) <= maxClarificationAnswerLen {
				cf.view.RecommendationReason = reason
			}
		}
	}

	if k == intake.KindNumber {
		if s := strings.TrimSpace(r.FormValue("v14min." + idx)); s != "" {
			v, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return cf, "数值下界不是合法数字"
			}
			cf.min = &v
			cf.view.HasMin, cf.view.Min = true, v
		}
		if s := strings.TrimSpace(r.FormValue("v14max." + idx)); s != "" {
			v, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return cf, "数值上界不是合法数字"
			}
			cf.max = &v
			cf.view.HasMax, cf.view.Max = true, v
		}
		if cf.min != nil && cf.max != nil && *cf.min > *cf.max {
			return cf, fmt.Sprintf("数值区间非法(min %v > max %v)", *cf.min, *cf.max)
		}
	}
	return cf, ""
}

// v14ClarificationAnswer 按字段类型复核并归并一条提交答案:
//   - 自由文本(f<i>)是所有类型共用的逃生舱(PRD §8.1,可选):除 number /
//     free_text 外,填写后优先于结构化选项(与既有折叠语义一致,输入框有标注);
//   - 选项值必须属于渲染时声明的候选集,boolean/confirmation 必须取规范值,
//     number 必须可解析且落在声明区间 —— 不符合即类型化拒绝,绝不猜测修正;
//   - 返回空串表示该问题未作答(交由调用方要求重答)。
func v14ClarificationAnswer(r *http.Request, cf v14ClarificationForm) (string, string) {
	idx := strconv.Itoa(cf.view.Index)
	free := strings.TrimSpace(r.FormValue("v14f." + idx))
	selected := make([]string, 0, len(r.Form["v14a."+idx]))
	for _, s := range r.Form["v14a."+idx] {
		if s = strings.TrimSpace(s); s != "" {
			selected = append(selected, s)
		}
	}

	switch cf.kind {
	case "":
		// 旧形态表单:既有语义(自由文本优先,否则取首个选项原样作答)。
		a := free
		if a == "" && len(selected) > 0 {
			a = selected[0]
		}
		return v14BoundedAnswer(a)
	case intake.KindNumber:
		if free == "" {
			return "", ""
		}
		v, err := strconv.ParseFloat(free, 64)
		if err != nil {
			return "", fmt.Sprintf("数值答案 %q 不是合法数字", free)
		}
		if cf.min != nil && v < *cf.min {
			return "", fmt.Sprintf("数值答案 %v 低于声明下界 %v", v, *cf.min)
		}
		if cf.max != nil && v > *cf.max {
			return "", fmt.Sprintf("数值答案 %v 高于声明上界 %v", v, *cf.max)
		}
		return v14BoundedAnswer(free)
	case intake.KindFreeText:
		return v14BoundedAnswer(free)
	}

	if free != "" { // 结构化类型的自由文本逃生舱(明确声明的覆盖)
		return v14BoundedAnswer(free)
	}
	switch cf.kind {
	case intake.KindBoolean:
		if len(selected) == 0 {
			return "", ""
		}
		if len(selected) > 1 || (selected[0] != "是" && selected[0] != "否") {
			return "", fmt.Sprintf("是否答案只能取「是」或「否」,得到 %v", selected)
		}
		return selected[0], ""
	case intake.KindConfirmation:
		if len(selected) == 0 {
			return "", ""
		}
		if len(selected) > 1 || (selected[0] != "确认" && selected[0] != "否决") {
			return "", fmt.Sprintf("确认答案只能取「确认」或「否决」,得到 %v", selected)
		}
		return selected[0], ""
	case intake.KindMultiChoice:
		if len(selected) == 0 {
			return "", ""
		}
		if len(selected) > maxClarificationSelected {
			return "", fmt.Sprintf("多选答案最多 %d 项,得到 %d 项", maxClarificationSelected, len(selected))
		}
		for _, s := range selected {
			if !v14InCandidates(s, cf.candidates) {
				return "", fmt.Sprintf("多选答案 %q 不在候选中", s)
			}
		}
		return strings.Join(selected, "、"), ""
	default: // choice / repository / branch / risk_choice:单选
		if len(selected) == 0 {
			return "", ""
		}
		if len(selected) > 1 {
			return "", fmt.Sprintf("该类型只能选一项,得到 %d 项", len(selected))
		}
		if !v14InCandidates(selected[0], cf.candidates) {
			return "", fmt.Sprintf("选项答案 %q 不在候选中", selected[0])
		}
		return selected[0], ""
	}
}

// v14BoundedAnswer 对答案做长度上限校验;超限类型化拒绝(不静默截断)。
func v14BoundedAnswer(a string) (string, string) {
	if a == "" {
		return "", ""
	}
	if len(a) > maxClarificationAnswerLen {
		return "", fmt.Sprintf("答案长度 %d 超上限 %d", len(a), maxClarificationAnswerLen)
	}
	return a, ""
}

func v14InCandidates(s string, cs []string) bool {
	for _, c := range cs {
		if c == s {
			return true
		}
	}
	return false
}

// v14HiddenIndices 取 name 前缀形如 "<prefix>.<i>" 的全部下标,按数值升序。
func v14HiddenIndices(form map[string][]string, prefix string) []string {
	seen := map[int]bool{}
	for k := range form {
		if !strings.HasPrefix(k, prefix+".") {
			continue
		}
		n := 0
		ok := false
		for _, c := range k[len(prefix)+1:] {
			if c < '0' || c > '9' {
				ok = false
				break
			}
			n = n*10 + int(c-'0')
			ok = true
		}
		if ok {
			seen[n] = true
		}
	}
	out := make([]int, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Ints(out)
	strs := make([]string, len(out))
	for i, n := range out {
		strs[i] = strconv.Itoa(n)
	}
	return strs
}

// v14QuestionViews 把澄清请求投影成展示行(P0-04:按 Kind/advisory/区间完整
// 投影)。输入已经 intake.validateClarifications 归一(词表/越界/区间都有
// 保证),这里只做展示转换,不再重复校验。
func v14QuestionViews(qs []intake.Clarification) []V14QuestionView {
	out := make([]V14QuestionView, 0, len(qs))
	for i, q := range qs {
		v := V14QuestionView{
			Index:                i,
			Question:             q.Question,
			Candidates:           q.Candidates,
			Kind:                 q.Kind,
			KindLabel:            v14KindLabel(q.Kind),
			RecommendationReason: q.RecommendationReason,
		}
		if q.Recommended != nil && *q.Recommended >= 0 && *q.Recommended < len(q.Candidates) {
			v.HasRecommended = true
			v.Recommended = *q.Recommended
			v.RecommendedChoice = q.Candidates[*q.Recommended]
		}
		if q.Min != nil {
			v.HasMin, v.Min = true, *q.Min
		}
		if q.Max != nil {
			v.HasMax, v.Max = true, *q.Max
		}
		out = append(out, v)
	}
	return out
}

// v14KindLabel 把字段类型映射为人读标签(仅展示;空/词表外归为自由作答)。
func v14KindLabel(k intake.ClarificationKind) string {
	switch k {
	case intake.KindChoice:
		return "单选"
	case intake.KindMultiChoice:
		return "多选"
	case intake.KindBoolean:
		return "是否"
	case intake.KindNumber:
		return "数值"
	case intake.KindRepository:
		return "仓库路径"
	case intake.KindBranch:
		return "分支"
	case intake.KindRiskChoice:
		return "风险取舍"
	case intake.KindConfirmation:
		return "确认"
	case intake.KindFreeText:
		return "自由文本"
	default:
		return "自由作答"
	}
}

// v14DraftViewOf 把 intake 草案投影成可确认/可编辑的展示(多值字段 → 多行文本)。
func v14DraftViewOf(d intake.Draft, sourceRef string) *V14DraftView {
	acs := make([]V14AcceptanceEdit, 0, len(d.Acceptance))
	for _, a := range d.Acceptance {
		acs = append(acs, V14AcceptanceEdit{ID: a.ID, Statement: a.Statement})
	}
	return &V14DraftView{
		TaskType:       string(d.TaskType),
		Summary:        d.Summary,
		Objective:      d.Objective,
		AllowedPaths:   strings.Join(d.Scope.AllowedPaths, "\n"),
		ForbiddenPaths: strings.Join(d.Scope.ForbiddenPaths, "\n"),
		NonScope:       strings.Join(d.NonScope, "\n"),
		Constraints:    strings.Join(d.Constraints, "\n"),
		Acceptance:     acs,
		RiskSignals:    d.RiskSignals,
		SourceRef:      sourceRef,
	}
}

/* ---------- 草案确认 → 冻结(P0-08) ---------- */

// errV14BadDraft 标记确认表单字段非法(类型化哨兵,errors.Is 判类)。
var errV14BadDraft = errors.New("v14: 草案字段非法")

// v14ConfirmPath 是确认 POST 的 PRG 回跳目标。
const v14ConfirmPath = "/v14/launch"

// V14ConfirmDraft 受理 owner 对草案的确认/编辑(AC2),并把确认变成不可变
// 的 Git 冻结(P0-08):表单 → 类型化草案 → intakefreeze.Freeze 冻结准入
// (确认事实 + owner 声明的 planned evidence → 三件套)→ artifacts.FreezeCommit
// 以单一期望基线经 gitwriter 原子提交。任何准入失败都类型化拒绝并原样退回
// 表单,绝不猜测修正;Git 提交的失败语义全部如实转述(基线冲突 / 幂等 /
// 失败),确认成功后 303 回本页(PRG,刷新不重复提交)。
func (s *Server) V14ConfirmDraft(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "表单解析失败", http.StatusBadRequest)
		return
	}
	draft, err := v14DraftFromForm(r)
	if err != nil {
		page := s.v14LaunchPage(r)
		page.Draft = draft
		page.ProposeErr = err.Error()
		v14Render(w, http.StatusBadRequest, "v14_launch", page)
		return
	}

	// 冻结准入:纯领域校验(Map 门 + 发明/遗漏/held-out 防护 + 既有权威
	// 校验)。失败即无冻结产物,原样退回表单让 owner 补充/修正;准入通过后
	// 的 Git 提交终局(成功/幂等/冲突/失败)一律在 v14CommitFrozen 内以
	// flash 303 重定向表达(PRG:刷新不会重复提交)。
	confirmed, plan, perr := v14FreezeInputsOf(r, draft)
	if perr == nil {
		var frozen intakefreeze.FrozenArtifacts
		frozen, perr = intakefreeze.Freeze(confirmed, plan)
		if perr == nil {
			s.v14CommitFrozen(w, r, frozen, draft.TaskID)
			return
		}
	}
	page := s.v14LaunchPage(r)
	page.Draft = draft
	page.ProposeErr = v14FreezeErrText(perr)
	v14Render(w, http.StatusBadRequest, "v14_launch", page)
}

// v14CommitFrozen 以单一期望基线把冻结三件套原子提交进制品仓库,并把四种
// 结局作为 flash 303 重定向表达(PRG):
//   - 成功:三件套已原子提交为一个提交,提示提交 sha;
//   - 重复确认:内容与 HEAD 一致 → 幂等(ErrNoChanges),不产生新提交;
//   - 基线冲突:HEAD 已离开期望基线 → 类型化冲突,绝不静默改基;
//   - 其他失败:如实转述,不编造成成功。
//
// 仓库未接入或读不到基线时同样重定向并说明原因。
func (s *Server) v14CommitFrozen(w http.ResponseWriter, r *http.Request, frozen intakefreeze.FrozenArtifacts, taskID string) {
	back := v14ConfirmPath
	env, ok := s.v13env()
	if !ok {
		redirectFlash(w, r, back, "bad", "未能冻结任务 "+taskID+":"+v13NotLoadable)
		return
	}
	wr, err := gitwriter.New(env.repoPath)
	if err != nil {
		redirectFlash(w, r, back, "bad", "未能冻结任务 "+taskID+":定位制品仓库失败 "+err.Error())
		return
	}
	base, err := wr.Head(r.Context())
	if err != nil {
		redirectFlash(w, r, back, "bad", "未能冻结任务 "+taskID+":读取制品仓库基线失败 "+err.Error())
		return
	}
	res, cerr := artifacts.FreezeCommit(r.Context(), env.repoPath, artifacts.FreezeCommitRequest{
		ExpectedBase: base,
		Artifacts:    frozen,
		Message:      "freeze: 确认冻结任务 " + taskID + "(FrozenIntent + TaskContract + VerificationContract)",
		Signature:    s.Content.Identity.gitSignature(),
	})
	switch {
	case errors.Is(cerr, gitwriter.ErrNoChanges):
		redirectFlash(w, r, back, "ok", "任务 "+taskID+" 已冻结(内容与制品仓库 HEAD 一致,未产生新提交)。")
	case errors.Is(cerr, gitwriter.ErrBaseRevision):
		redirectFlash(w, r, back, "bad", "冻结基线冲突:制品仓库 HEAD 已离开期望基线 "+v13Short(base)+
			",未写入任何冻结制品(绝不静默改基);请刷新后基于最新状态重新确认。")
	case cerr != nil:
		redirectFlash(w, r, back, "bad", "冻结提交失败(未产生冻结提交):"+cerr.Error())
	default:
		redirectFlash(w, r, back, "ok", "任务 "+taskID+" 冻结完成:FrozenIntent + TaskContract + VerificationContract 已原子提交为 "+
			v13Short(res.Commit)+",可从该提交确定性读回。")
	}
}

// v14DraftFromForm 从确认表单重建类型化草案视图 + 在场校验(空摘要/目标、
// 空边界、空验收条款一律拒绝;任务类型必须是 MVP 两个值之一)。冻结输入
// (task_id/profile/risk_hint/逐条款证据)一并回收,值本身交给冻结准入做
// 类型化校验 —— handler 不复制领域规则。
func v14DraftFromForm(r *http.Request) (*V14DraftView, error) {
	d := &V14DraftView{
		TaskType:       strings.TrimSpace(r.FormValue("task_type")),
		Summary:        strings.TrimSpace(r.FormValue("summary")),
		Objective:      strings.TrimSpace(r.FormValue("objective")),
		AllowedPaths:   strings.TrimSpace(r.FormValue("allowed_paths")),
		ForbiddenPaths: strings.TrimSpace(r.FormValue("forbidden_paths")),
		NonScope:       strings.TrimSpace(r.FormValue("non_scope")),
		Constraints:    strings.TrimSpace(r.FormValue("constraints")),
		SourceRef:      strings.TrimSpace(r.FormValue("source_ref")),
		TaskID:         strings.TrimSpace(r.FormValue("task_id")),
		Profile:        strings.TrimSpace(r.FormValue("profile")),
	}
	for _, h := range r.Form["risk_hint"] {
		if h = strings.TrimSpace(h); h != "" {
			d.RiskHints = append(d.RiskHints, h)
		}
	}
	d.RiskSignals = v14Lines(strings.Join(r.Form["risk"], "\n"))
	// 先回收验收条款再校验其它字段:校验失败重渲染时,用户已填的陈述不丢。
	idx := v14HiddenIndices(r.Form, "acid")
	for _, i := range idx {
		id := strings.TrimSpace(r.FormValue("acid." + i))
		stmt := strings.TrimSpace(r.FormValue("acstmt." + i))
		if id == "" || stmt == "" {
			return d, fmt.Errorf("%w: 验收条款 #%s 缺少 id 或陈述", errV14BadDraft, i)
		}
		d.Acceptance = append(d.Acceptance, V14AcceptanceEdit{
			ID:               id,
			Statement:        stmt,
			EvidenceKind:     strings.TrimSpace(r.FormValue("ev_kind." + i)),
			EvidencePath:     strings.TrimSpace(r.FormValue("ev_path." + i)),
			EvidenceSelector: strings.TrimSpace(r.FormValue("ev_selector." + i)),
		})
	}
	switch intake.TaskType(d.TaskType) {
	case intake.TaskTypeBugFix, intake.TaskTypeMigration:
	default:
		return d, fmt.Errorf("%w: 任务类型 %q 不在 MVP 支持范围(bug_fix/db_migration)", errV14BadDraft, d.TaskType)
	}
	if d.Summary == "" {
		return d, fmt.Errorf("%w: 意图摘要为空", errV14BadDraft)
	}
	if d.Objective == "" {
		return d, fmt.Errorf("%w: 目标为空", errV14BadDraft)
	}
	if len(v14Lines(d.AllowedPaths)) == 0 {
		return d, fmt.Errorf("%w: 实现边界(allowed_paths)为空:合同必须声明实现边界", errV14BadDraft)
	}
	if len(d.Acceptance) == 0 {
		return d, fmt.Errorf("%w: 验收条款为空:草案必须给出可确认的初始验收", errV14BadDraft)
	}
	return d, nil
}

// v14FreezeInputsOf 把确认表单装配为冻结准入(P0-07 Freeze)的输入:
// ConfirmedDraft(确认事实 + owner 勾选保留的风险提示)与 VerificationPlan
// (owner 逐条款声明的可见 planned evidence)。本确认面的证据是 repo 路径型:
// 类型与路径必须同时声明或同时缺席,半声明是写错的输入,类型化拒绝。
func v14FreezeInputsOf(r *http.Request, draft *V14DraftView) (intakefreeze.ConfirmedDraft, intakefreeze.VerificationPlan, error) {
	confirmed := intakefreeze.ConfirmedDraft{
		TaskID:    draft.TaskID,
		Profile:   draft.Profile,
		SourceRef: draft.SourceRef,
		Draft: intake.Draft{
			TaskType:  intake.TaskType(draft.TaskType),
			Summary:   draft.Summary,
			Objective: draft.Objective,
			Scope: intake.Scope{
				AllowedPaths:   v14Lines(draft.AllowedPaths),
				ForbiddenPaths: v14Lines(draft.ForbiddenPaths),
			},
			NonScope:    v14Lines(draft.NonScope),
			Constraints: v14Lines(draft.Constraints),
			Acceptance:  make([]intake.AcceptanceClause, 0, len(draft.Acceptance)),
			RiskSignals: draft.RiskSignals,
		},
		RiskHints: draft.RiskHints,
	}
	for _, a := range draft.Acceptance {
		confirmed.Draft.Acceptance = append(confirmed.Draft.Acceptance, intake.AcceptanceClause{ID: a.ID, Statement: a.Statement})
	}

	plan := intakefreeze.VerificationPlan{}
	for _, a := range draft.Acceptance {
		kind, path, sel := a.EvidenceKind, a.EvidencePath, a.EvidenceSelector
		if kind == "" && path == "" && sel == "" {
			continue // 本条款未声明证据 → 交给准入的遗漏防护类型化阻断
		}
		if kind == "" || path == "" {
			return confirmed, plan, fmt.Errorf("%w: 条款 %s 的证据类型与仓库路径必须同时声明(本确认面只接受 repo 路径型可见证据)", errV14BadDraft, a.ID)
		}
		// 词表外的类型由 verification 既有校验拒绝(ErrInvalidVerification),
		// handler 不复制词表。
		plan.Evidence = append(plan.Evidence, intakefreeze.EvidenceDeclaration{
			AcceptanceID: a.ID,
			Kind:         verification.EvidenceType(kind),
			Path:         path,
			Selector:     sel,
		})
	}
	return confirmed, plan, nil
}

// v14FreezeErrText 把冻结准入的类型化错误映射成人读原因(errors.Is 判类,
// 不做字符串匹配);未映射的错误原样透出。
func v14FreezeErrText(err error) string {
	switch {
	case errors.Is(err, errV14BadDraft):
		return err.Error()
	case errors.Is(err, intakefreeze.ErrInvalidProfile):
		return "治理等级缺失或不在词表内(light/standard/strict):profile 是治理决策,必须由 owner 显式选择,不设默认值。" + err.Error()
	case errors.Is(err, intakefreeze.ErrUnsupportedTaskType):
		return "任务类型不在 MVP 支持范围(bug_fix/db_migration),已按 fail-safe 拒绝。" + err.Error()
	case errors.Is(err, intakefreeze.ErrUnsupportedDraftSemantics):
		return "草案携带冻结制品无法承载的语义(non_scope/constraints):请先把它们解析进目标/边界/验收,或在确认时删除 —— 冻结绝不静默丢弃 owner 确认过的内容。" + err.Error()
	case errors.Is(err, intakefreeze.ErrUnknownEvidenceTarget):
		return "证据声明指向不存在的验收条款:Confirm→Freeze 不能发明证据绑定。" + err.Error()
	case errors.Is(err, intakefreeze.ErrMissingEvidence):
		return "还有验收条款没有 planned evidence 声明:缺失证据阻止执行就绪的冻结,请为每条验收声明至少一条证据。" + err.Error()
	case errors.Is(err, intakefreeze.ErrInvalidVerification):
		return "证据声明未通过冻结校验(类型词表/定位形状/重复声明)。" + err.Error()
	case errors.Is(err, intakefreeze.ErrInvalidContract):
		return "映射产物未通过任务合同校验。" + err.Error()
	case errors.Is(err, intentevidence.ErrInvalidEvidence):
		return "冻结意图证据非法(任务 id 必填且不含空白)。" + err.Error()
	default:
		return err.Error()
	}
}

// v14StringIn 报告 v 是否在列表里(模板勾选回显用;纯展示辅助)。
func v14StringIn(list []string, v string) bool {
	return v14InCandidates(v, list)
}

// v14Lines 把多行文本拆成去空白、去空行的行列表(顺序保留)。
func v14Lines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if v := strings.TrimSpace(line); v != "" {
			out = append(out, v)
		}
	}
	return out
}

/* ---------- 闭环启动(入队 v14_task) ---------- */

// V14LaunchRun 为一个 standard 且已冻结验证契约的 TaskContract 入队 v14_task
// 闭环 job(P2-09)。界面只入队不执行(INV-5):闭环事实由 worker 写,
// 结果在任务详情页从事实面派生展示。POST 后 303 回本页(PRG)。
func (s *Server) V14LaunchRun(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "表单解析失败", http.StatusBadRequest)
		return
	}
	taskID := strings.TrimSpace(r.FormValue("task"))
	back := "/v14/launch"
	env, ok := s.v13env()
	if !ok {
		redirectFlash(w, r, back, "bad", "未能启动闭环:"+v13NotLoadable)
		return
	}
	g, err := openV13Git(env.repoPath)
	if err != nil {
		redirectFlash(w, r, back, "bad", "未能启动闭环:Git 制品装载失败 "+err.Error())
		return
	}
	snap, err := artifacts.Load(env.repoPath, g.headHash)
	if err != nil {
		redirectFlash(w, r, back, "bad", "未能启动闭环:Git 制品装载失败 "+err.Error())
		return
	}
	var found *artifacts.TaskArtifact
	for i := range snap.Tasks {
		if snap.Tasks[i].Contract.ID == taskID {
			found = &snap.Tasks[i]
			break
		}
	}
	switch {
	case found == nil:
		redirectFlash(w, r, back, "bad", "未能启动闭环:HEAD 快照里没有 TaskContract "+taskID+"(Git 是 desired 侧唯一来源,不执行未提交的合同)。")
		return
	case found.Contract.Profile != "standard":
		redirectFlash(w, r, back, "bad", "未能启动闭环:task "+taskID+" 的 profile="+found.Contract.Profile+",闭环路径只执行 standard TaskContract。")
		return
	case v13FrozenFor(snap, taskID) == nil:
		redirectFlash(w, r, back, "bad", "未能启动闭环:task "+taskID+" 没有冻结 VerificationContract(验收必须在执行前冻结,INV-02)。")
		return
	}
	s.enqueue(w, r, v14TaskJobKind, v14RunJobPayload{Task: taskID}, back)
}
