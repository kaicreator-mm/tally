package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"

	"tally/internal/policygovernance"
	"tally/internal/store"
)

// PageRenderer 是 handlers 对渲染器的要求(在使用方定义接口)。*web.Renderer 满足它。
// 经接口注入是为了 handlers 不反向导入 web(依赖方向:web → handlers)。
type PageRenderer interface {
	Render(w http.ResponseWriter, status int, page string, data any)
}

// Identity 是写 git 时的提交身份。Now 可注入,测试用固定时刻。
type Identity struct {
	Name  string
	Email string
	Now   func() time.Time // nil → time.Now
}

// Config 是内容仓库配置。RepoPath 为空时所有写操作返回明确错误(不 panic)。
type Config struct {
	RepoPath string
	Identity Identity
}

// Server 持有 handlers 的依赖。字段都由 web 在构建路由时注入。
type Server struct {
	Render  PageRenderer
	Content *ContentStore
	Tasks   TaskSource // 任务状态来源;nil → 任务列表为空(T05c 由 ExecTaskSource 接)

	// Queue 是动作入队出口(T05b):拉起/推倒/重建/回滚/重诊断一律只入队,不碰事实面(INV-5)。
	// nil → 动作 POST 返回明确失败,不 panic。
	Queue JobQueue
	// Exec 是执行引擎页面(环境/部署/失败/指纹)的只读数据来源(T05b)。
	// nil → 各页渲染空状态行动提示。
	Exec ExecSource

	// Knowledge 是已验证修复库的只读浏览来源(T05c,INV-2:界面只读,绝不写)。
	// nil → 验证库页渲染空状态行动提示。
	Knowledge KnowledgeSource
	// Trace 是追溯视图 A/B 与条款覆盖率的读取来源(T05c):project/requirement 包的
	// 算法函数签名要 *store.Queries,故持具体类型。nil → 视图/覆盖率页渲染空状态行动提示。
	Trace *store.Queries

	// Governance 是策略治理事实的 append-only 存储(P3-08「提议新规」动作的
	// 写入面;nil → 评审页的提议新规表单按未接入呈现,仅此一次决定不受影响)。
	// 运行时策略装载永远走 pinned Git 快照(artifacts.LoadPolicies),本存储
	// 不是运行时权威(DL-09/BI-01)。
	Governance *policygovernance.Store
}

// New 构建 Server。
func New(cfg Config, render PageRenderer, tasks TaskSource) *Server {
	return &Server{
		Render:  render,
		Content: &ContentStore{RepoPath: cfg.RepoPath, Identity: cfg.Identity},
		Tasks:   tasks,
	}
}

// RegisterRoutes 把 T05a 的全部路由挂到 r 上。
//
// 调用方(web.NewRouter)负责在 r 上先挂登录 + CSRF 中间件;这里只注册路由。
// 所有非 GET 路由都在该保护组内(INV-9),且没有任何路由指向状态( INV-3,架构测试强制)。
func (s *Server) RegisterRoutes(r chi.Router) {
	r.Get("/", s.Index)

	// 项目管理(写 git projects/)
	r.Get("/projects", s.ListProjects)
	r.Get("/projects/new", s.NewProjectForm)
	r.Post("/projects", s.CreateProject)
	r.Get("/projects/{slug}", s.ProjectDetail)
	r.Get("/projects/{slug}/edit", s.EditProjectForm)
	r.Post("/projects/{slug}", s.UpdateProject)

	// 需求条款(写 git .tally/requirements/)
	r.Get("/requirements", s.ListRequirements)
	r.Post("/requirements", s.CreateRequirement)
	r.Get("/requirements/{project}/{file}", s.RequirementForm)
	r.Post("/requirements/{project}/{file}", s.UpdateRequirement)

	// 任务(状态算出,只读,无任何控件)
	r.Get("/tasks", s.ListTasks)

	// 文档(写 git docs/)
	r.Get("/docs", s.ListDocs)
	r.Get("/docs/new", s.NewDocForm)
	r.Post("/docs", s.CreateDoc)
	r.Get("/docs/{slug}", s.ViewDoc)
	r.Get("/docs/{slug}/edit", s.EditDocForm)
	r.Post("/docs/{slug}", s.UpdateDoc)

	// Wiki 笔记层(写 git wiki/,标「笔记·未验证」)
	r.Get("/wiki", s.ListWiki)
	r.Get("/wiki/new", s.NewWikiForm)
	r.Post("/wiki", s.CreateWiki)
	r.Get("/wiki/{slug}", s.ViewWiki)
	r.Get("/wiki/{slug}/edit", s.EditWikiForm)
	r.Post("/wiki/{slug}", s.UpdateWiki)

	// 执行引擎(T05b):状态从事实面算出只读;动作 POST 一律只入队 jobs(INV-5),
	// 由 worker 执行时写新记录。破坏性动作(推倒/重建/回滚)在模板里带警示色 + 二次确认。
	r.Get("/environments", s.ListEnvironments)
	r.Post("/environments/up", s.EnvUp)
	r.Post("/environments/down", s.EnvDown)
	r.Post("/environments/rebuild", s.EnvRebuild)
	r.Get("/deploys", s.ListDeploys)
	r.Post("/deploys/rollback", s.DeployRollback)
	r.Get("/failures", s.ListFailures)
	r.Get("/fingerprint", s.FingerprintDetail)

	// T05c 追溯 + 知识 Wiki + 算出状态:全部只读,唯一写路由是重诊断入队。
	// 重诊断走 jobs(INV-5),三轮上限由后端守(failure.ConsultBudget),界面不提供绕过。
	r.Get("/knowledge", s.ListKnowledge)   // 已验证修复库,只读无编辑入口(INV-2)
	r.Get("/diagnose", s.DiagnoseStatus)   // 诊断状态(第几轮/五分类)
	r.Post("/diagnose", s.DiagnoseTrigger) // 触发重诊断:入队 jobs,后端守三轮上限
	r.Get("/coverage", s.Coverage)         // 条款覆盖率(无 project 列索引,有则详情)
	r.Get("/views/a", s.ViewA)             // 视图 A:条款累计失败
	r.Get("/views/b", s.ViewB)             // 视图 B:指纹影响面(关联条款 LEFT JOIN)

	// v1.3 P1 最小界面(P1-44,PRD §26.2):Tasks / Changes·Run Detail / Patch Review。
	// 数据源:事实面 s.Trace(*store.Queries)+ Git 制品 s.Content.RepoPath
	// (.tally TaskContract/VerificationContract 与 base..candidate diff)。
	// 二者缺席时页面渲染「未接入」空状态,不报 500。唯一写动作是 Patch Review
	// 的 Owner 决定,且只落 HumanApprovalEvidence(append-only 证据),不写任何
	// 任务状态字段(INV-STATE-01:状态由 State Engine 从证据派生)。
	r.Get("/v13/tasks", s.V13Tasks)
	r.Get("/v13/runs", s.V13Runs) // P2-03:Runs 投影(实现 Run 过程事实时间线,只读)
	r.Get("/v13/changes/{csid}", s.V13Change)
	r.Get("/v13/review/{csid}", s.V13Review)
	r.Post("/v13/review/{csid}/decide", s.V13Decide)
	r.Post("/v13/review/{csid}/propose-rule", s.V13ProposeRule) // P3-08:提议新规(治理草稿;绝不写审批证据、绝不改运行时策略)

	// v1.4 最小产品面(P3-08,PRD §33):Launch(意图受理)+ 闭环任务详情。
	// 路由实现全部在 v14_launch.go/v14_task.go;数据源为派生读模型,无写状态。
	s.RegisterV14Routes(r)
}

/* ---------- CSRF token(由 web 的 csrf 中间件写入 context) ---------- */

// ctxKey 是 CSRF token 在 context 里的键类型。
type ctxKey struct{}

// CSRFTokenKey 供 web 的 csrf 中间件写入当前请求的有效 token。
// 导出是因为键定义在 handlers(依赖方向:web → handlers,无反向导入)。
var CSRFTokenKey = ctxKey{}

// CSRFToken 从 context 取当前请求的 CSRF token(表单隐藏域用它)。
func CSRFToken(ctx context.Context) string {
	v, _ := ctx.Value(CSRFTokenKey).(string)
	return v
}

/* ---------- 页面公共数据 ---------- */

// Flash 是一条提示条(保存成功 / 冲突 / 错误)。Kind ∈ {ok,bad,warn,""}。
type Flash struct {
	Kind string
	Msg  string
}

// Page 是每个页面数据的公共部分(嵌入各页结构体,字段提升后模板直接 .Title 等)。
type Page struct {
	Title  string
	Active string // 导航高亮键:index/projects/requirements/tasks/docs/wiki
	CSRF   string
	Flash  Flash
}

// page 组装公共页面数据(含从 context 取 CSRF、从 query 取 flash)。
func (s *Server) page(r *http.Request, title, active string) Page {
	return Page{
		Title:  title,
		Active: active,
		CSRF:   CSRFToken(r.Context()),
		Flash:  flashFromQuery(r),
	}
}

// flashFromQuery 从重定向带回的 query 读提示条(?kind=&flash=)。
func flashFromQuery(r *http.Request) Flash {
	q := r.URL.Query()
	return Flash{Kind: q.Get("kind"), Msg: q.Get("flash")}
}

// redirectFlash 带提示条 303 重定向(PRG:POST 后总是重定向,刷新不重复提交)。
func redirectFlash(w http.ResponseWriter, r *http.Request, path, kind, msg string) {
	u := path + "?kind=" + url.QueryEscape(kind) + "&flash=" + url.QueryEscape(msg)
	http.Redirect(w, r, u, http.StatusSeeOther)
}

// serverError 记日志并回一个说清楚原因的错误页(不道歉不含糊,见 06-ui-spec 文案)。
func (s *Server) serverError(w http.ResponseWriter, r *http.Request, op string, err error) {
	slog.Error(op, "err", err, "path", r.URL.Path)
	http.Error(w, op+"失败: "+err.Error(), http.StatusInternalServerError)
}
