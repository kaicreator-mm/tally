package web

import (
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"tally/internal/store"
	"tally/internal/web/handlers"
	assets "tally/web"
)

// Deps 是 web 需要的依赖。handler 编排业务包,不直接导入 llm。
type Deps struct {
	Sessions     SessionStore
	PasswordHash string // 单用户密码的 bcrypt 哈希

	// 内容仓库(写 git 用户内容:项目/需求/文档/笔记)。RepoPath 为空时写操作返回明确错误。
	ContentRepo string
	Identity    handlers.Identity

	// 任务状态来源(T05b/c 接执行数据时提供;nil → 任务列表为空并给行动提示)。
	Tasks handlers.TaskSource

	// 动作队列(T05b):界面触发动作只入队,worker 执行。nil → 动作按钮返回明确失败。
	Queue handlers.JobQueue
	// 执行数据只读来源(T05b):环境/部署/失败/指纹页从事实面与索引面读。nil → 空状态行动提示。
	Exec handlers.ExecSource

	// 验证库只读浏览来源(T05c,INV-2:界面只读)。nil → 验证库页空状态。
	Knowledge handlers.KnowledgeSource
	// 追溯/覆盖率读取来源(T05c):project/requirement 算法要 *store.Queries。nil → 视图/覆盖率页空状态。
	Trace *store.Queries

	// v1.5 Engineering Control Plane API + thin browser console. Both are
	// mounted inside the authenticated + CSRF protected browser group. nil
	// keeps the v1.5 control-plane surfaces disabled.
	ControlPlane http.Handler

	// T11d webhook 运输层:CI 构建结果 POST 进来即完成录入(D-d5 复用 ingest.IngestBuild)。
	IngestStore  *store.Store // 录入写事实面;nil → webhook 按未启用处理
	LogDir       string       // 原始日志留存目录(TALLY_LOG_DIR);空 → 不落盘,raw_ref 由请求自带
	WebhookToken string       // TALLY_WEBHOOK_TOKEN;空 → webhook 未启用,路由一律 503(D-d1)

	// T11e 自动诊断(默认关,D-e1):webhook 录入失败后按指纹自动入队 diagnose。
	DiagnoseAuto  bool             // TALLY_DIAGNOSE_AUTO=1 → true;缺省 false 行为与之前一致
	DiagnoseQueue DiagnoseEnqueuer // 入队出口(*jobs.Queue);nil → 开关开也不入队
}

// NewRouter 构建完整路由。
//
// 写边界(INV-9):公开路由仅登录 / 登出 / 健康检查 / 静态资源;其余一律挂在登录 + CSRF 之后。
// 所有 CRUD 与动作路由(T05a/b/c)都加在那个 Group 里,自动受保护。
// webhook(T11d)挂在登录组外:token 就是它的登录(D-d2),未启用/错 token 一律非 2xx,
// INV-9 行为测试不改而过;非浏览器交互,自然也不在 CSRF 组内(D-d3)。
func NewRouter(deps Deps) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)

	renderer := mustNewRenderer()

	// 静态资源(公开,GET):内嵌的 tally.css + vendored htmx.min.js。
	staticSub, err := fs.Sub(assets.StaticFS, "static")
	if err != nil {
		panic("web: 取静态资源子 FS: " + err.Error())
	}
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))

	// 公开路由
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/login", loginForm(renderer))
	r.Post("/login", deps.login)

	// webhook 运输层(T11d):token 鉴权,不进登录/CSRF 组。
	r.Post("/webhook/build", deps.webhookBuild)

	// 其余一切:先登录,再 CSRF。handler 路由(T05a)在此注册;T05b/c 的动作路由经
	// handlers.Server.RegisterRoutes 追加,自动落在同一保护组内。
	srv := handlers.New(
		handlers.Config{RepoPath: deps.ContentRepo, Identity: deps.Identity},
		renderer,
		deps.Tasks,
	)
	srv.Queue = deps.Queue // 动作入队出口;nil 时动作 POST 给明确失败
	srv.Exec = deps.Exec   // 执行数据只读来源;nil 时各页落空状态
	srv.Knowledge = deps.Knowledge
	srv.Trace = deps.Trace
	srv.ControlPlane = deps.ControlPlane != nil // 导航条目只在控制台真实挂载时渲染
	r.Group(func(r chi.Router) {
		r.Use(deps.requireAuth)
		r.Use(csrfMiddleware)

		srv.RegisterRoutes(r)
		if deps.ControlPlane != nil {
			// The stdlib mux inside controlplane rematches the full request path;
			// chi only provides the authenticated/CSRF-protected outer boundary.
			r.Handle("/api/v15/*", deps.ControlPlane)
			r.Handle("/v15/*", deps.ControlPlane)
		}
		r.Post("/logout", deps.logout)
	})

	return r
}

// mustNewRenderer 构建渲染器;内嵌模板是编译期已知的,解析失败属不可恢复的编程错误,直接 panic 让其尽早暴露。
func mustNewRenderer() *Renderer {
	rd, err := NewRenderer()
	if err != nil {
		panic("web: 模板解析失败: " + err.Error())
	}
	return rd
}

// loginForm 渲染登录页(公开页,无会话/无导航,模板自带极简外壳)。
func loginForm(renderer *Renderer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			Title string
			Error bool
		}{
			Title: "登录",
			Error: r.URL.Query().Get("error") == "1",
		}
		renderer.Render(w, http.StatusOK, "login", data)
	}
}
