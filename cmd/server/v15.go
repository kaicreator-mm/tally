package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"tally/internal/agentruntime"
	"tally/internal/controlplane"
	"tally/internal/store"
	"tally/internal/taskexecution"
	"tally/runtimes/codex"
	"tally/runtimes/dsh"
	"tally/runtimes/fake"
)

func v15ControlPlaneFromEnv(contentRepo string, queries *store.Queries) (http.Handler, string, error) {
	repo, err := controlplane.NewFileRepository(envOr("TALLY_V15_STATE_ROOT", "./tally-state/v1.5"))
	if err != nil {
		return nil, "", err
	}

	runtimeName := strings.TrimSpace(os.Getenv("TALLY_V15_RUNTIME"))
	var service *controlplane.Service
	if runtimeName == "" {
		// Lifecycle/state is usable without an execution runtime. Dispatch stays
		// explicitly unavailable; no FakeRuntime is silently selected.
		service, err = controlplane.NewService(repo, nil, nil)
		if err != nil {
			return nil, "", err
		}
	} else {
		var runtime agentruntime.Runtime
		switch runtimeName {
		case "fake":
			runtime = fake.New(fake.Config{})
		case "codex":
			runtime = codex.New(codex.Config{
				BinPath: os.Getenv("TALLY_V15_CODEX_BIN"),
				Model:   os.Getenv("TALLY_V15_MODEL"),
			})
		case "dsh":
			runtime = dsh.New(dsh.Config{
				BinPath:  os.Getenv("TALLY_V15_DSH_BIN"),
				Model:    os.Getenv("TALLY_V15_MODEL"),
				Provider: os.Getenv("TALLY_V15_DSH_PROVIDER"),
			})
		default:
			return nil, "", fmt.Errorf("v1.5 control plane: unsupported TALLY_V15_RUNTIME %q (want fake|codex|dsh)", runtimeName)
		}

		executor, execErr := taskexecution.NewRuntimeExecutor(runtime, taskexecution.LocalEnvironment{})
		if execErr != nil {
			return nil, "", fmt.Errorf("v1.5 control plane: build Local AgentExecutor: %w", execErr)
		}
		maxRepair, parseErr := v15MaxRepairFromEnv()
		if parseErr != nil {
			return nil, "", parseErr
		}
		harness, harnessErr := controlplane.NewBrownfieldHarness(controlplane.BrownfieldHarnessConfig{
			Repo:         contentRepo,
			WorktreeRoot: envOr("TALLY_V15_WORKTREE_ROOT", filepath.Join(os.TempDir(), "tally-v15-worktrees")),
			HoldoutRoot:  os.Getenv("TALLY_V15_HOLDOUT_ROOT"),
			EnvName:      envOr("TALLY_V15_ENV", "server-local"),
			Runtime:      runtime.Descriptor(),
			Executor:     executor,
			Queries:      queries,
			MaxRepair:    maxRepair,
		})
		if harnessErr != nil {
			return nil, "", fmt.Errorf("v1.5 control plane: build Engineering Harness: %w", harnessErr)
		}
		service, err = controlplane.NewServiceWithHarness(repo, harness)
		if err != nil {
			return nil, "", err
		}
	}

	handler, err := controlplane.NewHTTPHandler(service)
	if err != nil {
		return nil, "", err
	}
	return handler, runtimeName, nil
}

func v15MaxRepairFromEnv() (int, error) {
	raw := strings.TrimSpace(os.Getenv("TALLY_V15_MAX_REPAIR"))
	if raw == "" {
		return 0, nil // brownfield closed-loop policy default
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("v1.5 control plane: TALLY_V15_MAX_REPAIR must be an integer: %w", err)
	}
	return n, nil
}
