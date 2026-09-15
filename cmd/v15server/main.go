// Command v15server runs the database-independent v1.5 Engineering Control
// Plane without the legacy v1.4 web, PostgreSQL sessions or DB job queue.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tally/internal/controlplane"
	"tally/internal/persistence/migratev150"
	"tally/internal/persistence/profile"
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

	service, err := controlplane.NewService(adapter, nil, nil)
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
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":      "ok",
			"component":   "tally-v1.5-core",
			"persistence": string(selected),
			"execution":   "not_configured",
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

	slog.Info("v1.5 Core server listening", "addr", addr, "persistence", selected, "execution", "not_configured")
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("v1.5 Core server failed", "err", err)
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
