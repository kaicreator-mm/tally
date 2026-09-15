// Command server 是 tally 的 HTTP 服务:界面、接收 webhook、入队任务。
//
// 主要持久状态仍在 Postgres 和 git；v1.5.0 Engineering Control Plane 的
// compatibility snapshot 明确落在 TALLY_V15_STATE_ROOT，供 v1.5.1 存储迁移前
// 保证 Active TaskDAG/restart 状态可恢复。路由(含登录/CSRF)在 internal/web。
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/crypto/bcrypt"

	"tally/internal/jobs"
	"tally/internal/knowledge"
	"tally/internal/store"
	"tally/internal/web"
	"tally/internal/web/handlers"
)

const defaultDSN = "postgres://postgres:postgres@localhost:5432/tally?sslmode=disable"

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	ctx := context.Background()

	st, err := store.Open(ctx, databaseURL())
	if err != nil {
		slog.Error("连接数据库失败", "err", err)
		os.Exit(1)
	}
	defer st.Close()

	// 内容仓库:写 git 用户内容(项目/需求/文档/笔记)。启动时确保是 git 仓库(幂等)。
	contentRepo := envOr("TALLY_CONTENT_REPO", "./tally-content")
	identity := handlers.Identity{Name: "tally-web", Email: "tally-web@localhost"}
	if err := handlers.EnsureRepo(contentRepo, identity); err != nil {
		slog.Error("准备内容仓库失败", "path", contentRepo, "err", err)
		os.Exit(1)
	}

	// server 端旧路径只入队、不执行(执行在 cmd/worker):handlers 表为空即可。
	queue := jobs.New(st, nil)

	// P0-05:v1.4 意图入口的生产推理出口(PRD §28 / DECISION_LOCKS #17 /
	// ADR-0002)。已配置 → 组合根注入 airuntime.Executor(过渡传输见
	// aiexecutor.go);未配置 → 显式告警并保持未接入:意图入口如实渲染
	// 「AI Runtime 未接入」,不假装可用、不 panic —— 与 TALLY_WEBHOOK_TOKEN
	// 同一装配纪律(明确未启用,不做默认放行)。
	proposeExec, err := aiProposeExecutorFromEnv()
	switch {
	case err == nil:
		handlers.SetV14ProposeExecutor(proposeExec)
		slog.Info("v1.4 意图 AI 执行器已接入", "transport", "interim-llm(ADR-0002,真实 SDK 接线后替换)")
	case errors.Is(err, errAIRuntimeNotConfigured):
		slog.Warn("AI Runtime 未配置,v1.4 意图入口保持未接入(功能显式不可用,服务照常启动):" +
			"设置 TALLY_LLM_API_KEY(或 OPENAI_API_KEY)或 TALLY_LLM_BASE_URL 后重启")
	default:
		slog.Error("AI Runtime 执行器装配失败", "err", err)
		os.Exit(1)
	}

	// v1.5 lifecycle/state API 始终可用。执行仍需显式选择 runtime；一旦
	// 启用，dispatch 通过 Engineering Harness 复用既有 brownfield
	// Closed Loop + candidateverify，并使用 Tally-owned disposable worktree。
	controlPlane, v15Runtime, err := v15ControlPlaneFromEnv(contentRepo, st.Queries())
	if err != nil {
		slog.Error("v1.5 Engineering Control Plane 装配失败", "err", err)
		os.Exit(1)
	}
	if v15Runtime == "" {
		slog.Warn("v1.5 Control Plane 已启用生命周期/状态 API，但 Engineering Harness 未启用；设置 TALLY_V15_RUNTIME 后才能 dispatch")
	} else {
		slog.Info("v1.5 Engineering Harness 已接入", "runtime", v15Runtime, "environment", "local")
	}

	deps := web.Deps{
		Sessions:     web.PGSessions(st.Queries()),
		PasswordHash: passwordHash(),
		ContentRepo:  contentRepo,
		Identity:     identity,
		Queue:        queue,
		// 执行引擎页面的只读数据来源(环境/部署/失败/指纹)。
		Exec: st.Queries(),

		// T05c:任务状态从执行链算出(INV-3);已验证修复库只读浏览(INV-2);追溯/覆盖率读索引面。
		Tasks:     handlers.NewExecTaskSource(st.Queries()),
		Knowledge: knowledge.NewStore(contentRepo, st.Queries()),
		Trace:     st.Queries(),

		ControlPlane: controlPlane,

		// T11d webhook 运输层:CI 构建结果 POST 进来即完成录入。
		// TALLY_WEBHOOK_TOKEN 未设置 → 路由整体 503(明确未启用,不做默认放行)。
		IngestStore:  st,
		LogDir:       envOr("TALLY_LOG_DIR", "./tally-logs"),
		WebhookToken: os.Getenv("TALLY_WEBHOOK_TOKEN"),
		// T11e 自动诊断(默认关,D-e1):webhook 录入失败后按指纹自动入队 diagnose,worker 消费。
		DiagnoseAuto:  os.Getenv("TALLY_DIAGNOSE_AUTO") == "1",
		DiagnoseQueue: queue,
	}

	addr := envOr("TALLY_ADDR", ":8080")
	srv := &http.Server{Addr: addr, Handler: web.NewRouter(deps)}

	// 优雅关停
	sctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-sctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(sctx), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	slog.Info("server 监听中", "addr", addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server 出错", "err", err)
		os.Exit(1)
	}
	slog.Info("server 已停止")
}

// passwordHash 返回单用户密码的 bcrypt 哈希。
// 从 TALLY_PASSWORD 读明文密码;未设置时用开发默认值并告警(生产必须设置)。
func passwordHash() string {
	pw := os.Getenv("TALLY_PASSWORD")
	if pw == "" {
		pw = "tally-dev"
		slog.Warn("未设置 TALLY_PASSWORD,使用开发默认密码 tally-dev(生产环境请设置)")
	}
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("生成密码哈希失败", "err", err)
		os.Exit(1)
	}
	return string(h)
}

func databaseURL() string {
	if v := os.Getenv("DATABASE_URL"); v != "" {
		return v
	}
	return defaultDSN
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
