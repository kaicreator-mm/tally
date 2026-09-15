// Command v15server runs the database-independent v1.5 Engineering Control
// Plane without the legacy v1.4 web, PostgreSQL sessions or DB job queue.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"tally/internal/controlplane"
	"tally/internal/persistence/migratev150"
	"tally/internal/persistence/profile"
	"tally/internal/taskexecution"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	adapter, selected, err := profile.Open(ctx, profile.Config{
		Name:        profile.Name(envOr("TALLY_V15_PERSISTENCE", string(profile.SQLite))),
		SQLitePath:  envOr("TALLY_V15_SQLITE_PATH", profile.DefaultSQLitePath),
		PostgresDSN: os.Getenv("TALLY_V15_POSTGRES_URL"),
	})
	if err != nil {
		slog.Error("v1.5 persistence startup failed", "err", err)
		os.Exit(1)
	}
	defer adapter.Close()

	if sourceRoot := os.Getenv("TALLY_V15_IMPORT_V150_STATE_ROOT"); sourceRoot != "" {
		report, err := migratev150.Import(ctx, sourceRoot, adapter)
		if err != nil {
			slog.Error("v1.5.0 snapshot import failed", "source_root", sourceRoot, "err", err)
			os.Exit(1)
		}
		slog.Info("v1.5.0 snapshot import completed", "source_root", sourceRoot, "imported", len(report.Imported), "skipped_existing", len(report.Skipped))
	}

	runtimeName := strings.TrimSpace(os.Getenv("TALLY_V15_RUNTIME"))
	contentRepo := strings.TrimSpace(os.Getenv("TALLY_V15_CONTENT_REPO"))
	baseRevision := strings.TrimSpace(os.Getenv("TALLY_V15_BASE_REVISION"))
	var executor taskexecution.AgentExecutor
	var scheduler *controlplane.LocalScheduler
	environmentKind := taskexecution.EnvironmentLocal
	environmentTarget := "local"
	workspaceRoot := strings.TrimSpace(os.Getenv("TALLY_V15_WORKSPACE_ROOT"))
	executionState := "disabled"
	verificationState := "not_configured"
	if runtimeName != "" {
		composition, composeErr := executionFromEnv(runtimeName, contentRepo)
		if composeErr != nil {
			slog.Error("v1.5 AgentExecutor/environment startup failed", "err", composeErr)
			os.Exit(1)
		}
		executor = composition.Executor
		environmentKind = composition.Kind
		environmentTarget = composition.Target
		workspaceRoot = composition.WorkspaceRoot
	}

	var service *controlplane.Service
	if executor == nil {
		service, err = controlplane.NewService(adapter, nil, nil)
	} else {
		// The production path stays behind the frozen TaskHarness authority.
		// ExecutionEnvironment only changes where workspace/process execution
		// occurs; it does not become a second scheduler or verification loop.
		harness, harnessErr := controlplane.NewExecutionOnlyHarness(executor)
		if harnessErr != nil {
			slog.Error("v1.5 execution harness startup failed", "err", harnessErr)
			os.Exit(1)
		}
		service, err = controlplane.NewServiceWithHarness(adapter, harness)
	}
	if err != nil {
		slog.Error("v1.5 control plane startup failed", "err", err)
		os.Exit(1)
	}
	controlPlane, err := controlplane.NewHTTPHandler(service)
	if err != nil {
		slog.Error("v1.5 HTTP handler startup failed", "err", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.Handle("/", controlPlane)

	if executor != nil {
		requests, reqErr := controlplane.NewGitTaskRequestFactory(controlplane.GitTaskRequestFactoryConfig{
			Repo:              contentRepo,
			BaseRevision:      baseRevision,
			RuntimeID:         runtimeName,
			WorkspaceRoot:     workspaceRoot,
			EnvironmentKind:   environmentKind,
			EnvironmentTarget: environmentTarget,
			ModelProfile:      requestModelProfile(runtimeName),
		})
		if reqErr != nil {
			slog.Error("v1.5 scheduler request composition failed", "err", reqErr)
			os.Exit(1)
		}
		maxConcurrent, parseErr := schedulerConcurrencyFromEnv()
		if parseErr != nil {
			slog.Error("v1.5 scheduler configuration failed", "err", parseErr)
			os.Exit(1)
		}
		scheduler, err = controlplane.NewLocalScheduler(service, requests, maxConcurrent)
		if err != nil {
			slog.Error("v1.5 local scheduler startup failed", "err", err)
			os.Exit(1)
		}
		schedulerHTTP, handlerErr := controlplane.NewSchedulerHTTPHandler(scheduler)
		if handlerErr != nil {
			slog.Error("v1.5 scheduler HTTP startup failed", "err", handlerErr)
			os.Exit(1)
		}
		mux.Handle("POST /api/v15/projects/{project}/scheduler/recover", schedulerHTTP)
		mux.Handle("POST /api/v15/projects/{project}/scheduler/run-once", schedulerHTTP)
		mux.Handle("POST /api/v15/projects/{project}/scheduler/run", schedulerHTTP)
		executionState = "local_scheduler+harness+agent_executor:" + string(environmentKind)
		verificationState = "fail_closed_unavailable"
	}

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":             "ok",
			"component":          "tally-v1.5-core",
			"persistence":        string(selected),
			"execution":          executionState,
			"runtime":            runtimeName,
			"environment":        string(environmentKind),
			"environment_target": environmentTarget,
			"verification":       verificationState,
		})
	})

	addr := envOr("TALLY_V15_ADDR", "127.0.0.1:8081")
	srv := &http.Server{Addr: addr, Handler: mux}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	slog.Info("v1.5 Core server listening", "addr", addr, "persistence", selected, "execution", executionState, "runtime", runtimeName, "environment", environmentKind, "environment_target", environmentTarget, "verification", verificationState)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("v1.5 Core server failed", "err", err)
		os.Exit(1)
	}
}

func schedulerConcurrencyFromEnv() (int, error) {
	raw := strings.TrimSpace(os.Getenv("TALLY_V15_SCHEDULER_MAX_CONCURRENT"))
	if raw == "" {
		return 2, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("TALLY_V15_SCHEDULER_MAX_CONCURRENT must be a positive integer")
	}
	return n, nil
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
