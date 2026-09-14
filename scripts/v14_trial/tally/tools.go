// tools.go: small shared helpers for the Tally-arm runner — runtime
// construction, DB reachability, tooling builds.
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5"

	"tally/runtimes/codex"
)

func context2() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Minute)
}

// newCodexRuntime constructs the REAL CodexRuntime with the ADR-0003 provider
// endpoint (coordinator-provided environment) and the minimal owner home as
// the auth.json handoff source (Decision Lock 22). Per-session isolated homes
// and provider seeding are unconditional production behavior (P0-03/P0-04).
func newCodexRuntime(cfg trialConfig) *codex.Runtime {
	return codex.New(codex.Config{
		BinPath: cfg.CodexBin,
		Sandbox: cfg.Sandbox,
		Model:   cfg.Model,
		Env:     map[string]string{"CODEX_HOME": cfg.OwnerCodexHome},
		Provider: &codex.ProviderEndpoint{
			ID:                 providerID,
			Name:               providerName,
			BaseURL:            providerBaseURL,
			WireAPI:            providerWireAPI,
			RequiresOpenAIAuth: true,
		},
	})
}

func checkDB(ctx context.Context, dsn string) error {
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	return conn.Ping(ctx)
}

// ensurePsqlsh builds the minimal psql-compatible client the arm-neutral judge
// uses as --psql-bin on this host (no psql binary installed).
func ensurePsqlsh(toolsDir string) (string, error) {
	if err := os.MkdirAll(toolsDir, 0o755); err != nil {
		return "", err
	}
	out := filepath.Join(toolsDir, "psqlsh"+exeSuffix())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "build", "-o", out, "./scripts/v14_trial/tally/psqlsh") //nolint:gosec // structured argv
	b, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("go build psqlsh: %v: %s", err, truncateUTF8(string(b), 500))
	}
	return out, nil
}
