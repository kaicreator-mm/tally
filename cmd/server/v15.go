package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"tally/internal/agentruntime"
	"tally/internal/closedloop"
	"tally/internal/controlplane"
	"tally/internal/engineering"
	"tally/internal/multiagent"
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
		isolationVerified := os.Getenv("TALLY_V15_SESSION_ISOLATION_VERIFIED") == "1"
		switch runtimeName {
		case "fake":
			runtime = fake.New(fake.Config{})
		case "codex":
			runtime = codex.New(codex.Config{
				BinPath:                  os.Getenv("TALLY_V15_CODEX_BIN"),
				Model:                    os.Getenv("TALLY_V15_MODEL"),
				SessionIsolationVerified: isolationVerified,
			})
		case "dsh":
			runtime = dsh.New(dsh.Config{
				BinPath:                  os.Getenv("TALLY_V15_DSH_BIN"),
				Model:                    os.Getenv("TALLY_V15_MODEL"),
				Provider:                 os.Getenv("TALLY_V15_DSH_PROVIDER"),
				SessionIsolationVerified: isolationVerified,
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
		worktreeRoot := envOr("TALLY_V15_WORKTREE_ROOT", filepath.Join(os.TempDir(), "tally-v15-worktrees"))
		envName := envOr("TALLY_V15_ENV", "server-local")
		harness, harnessErr := controlplane.NewBrownfieldHarness(controlplane.BrownfieldHarnessConfig{
			Repo:         contentRepo,
			WorktreeRoot: worktreeRoot,
			HoldoutRoot:  os.Getenv("TALLY_V15_HOLDOUT_ROOT"),
			EnvName:      envName,
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

		// v1.5.4 composes above the same frozen AgentExecutor used by the
		// Engineering Harness. This deliberately does not introduce a second
		// runtime stack or worker queue: Local/Codex/DSH execution remains owned
		// by taskexecution.RuntimeExecutor, and the verified-role guard reuses
		// the existing verification-enforcement abstraction for review roles.
		maxConcurrency, parseErr := v15MultiAgentMaxConcurrencyFromEnv()
		if parseErr != nil {
			return nil, "", parseErr
		}
		roleExecutor, roleErr := composeV15MultiAgentExecutor(executor, multiagent.VerifiedRoleExecutorConfig{
			Delegate:     executor,
			Repo:         contentRepo,
			WorktreeRoot: worktreeRoot,
			EnvName:      envName,
			Runtime:      runtime.Descriptor(),
			Queries:      queries,
		})
		if roleErr != nil {
			return nil, "", fmt.Errorf("v1.5 control plane: build verified role executor: %w", roleErr)
		}
		multiAgentEngine, engineErr := multiagent.NewEngine(roleExecutor, nil, multiagent.Config{MaxConcurrency: maxConcurrency})
		if engineErr != nil {
			return nil, "", fmt.Errorf("v1.5 control plane: build Multi-Agent Engine: %w", engineErr)
		}
		service.MultiAgent = multiAgentEngine
	}

	handler, err := controlplane.NewHTTPHandler(service)
	if err != nil {
		return nil, "", err
	}
	return handler, runtimeName, nil
}

// composeV15MultiAgentExecutor wires the v1.5.4 multi-agent role executor
// above the frozen AgentExecutor: reviewer/verifier/specialist role runs go
// through the existing VerifiedRoleExecutor verification enforcement
// (candidate freeze, read-only drift rejection, candidateverify evidence),
// while implementer/repairer runs keep the raw delegate behavior.
func composeV15MultiAgentExecutor(raw taskexecution.AgentExecutor, cfg multiagent.VerifiedRoleExecutorConfig) (taskexecution.AgentExecutor, error) {
	cfg.Delegate = raw
	guarded, err := multiagent.NewVerifiedRoleExecutor(cfg)
	if err != nil {
		return nil, err
	}
	return &v15GuardedRoleExecutor{raw: raw, guarded: guarded}, nil
}

type v15GuardedRoleExecutor struct {
	raw     taskexecution.AgentExecutor
	guarded *multiagent.VerifiedRoleExecutor
}

func (e *v15GuardedRoleExecutor) Execute(ctx context.Context, req taskexecution.TaskExecutionRequest) (taskexecution.TaskExecutionResult, error) {
	if multiagent.IsGuardedRoleInstruction(req.Instruction) {
		return e.guarded.Execute(ctx, req)
	}
	return e.raw.Execute(ctx, req)
}

func (e *v15GuardedRoleExecutor) Cancel(id engineering.ID) error {
	// VerifiedRoleExecutor only decorates the same delegate for cancellation,
	// so the raw executor stays the single cancellation authority.
	return e.raw.Cancel(id)
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
	if _, err := closedloop.NewRepairPolicy(n); err != nil {
		return 0, fmt.Errorf("v1.5 control plane: invalid TALLY_V15_MAX_REPAIR: %w", err)
	}
	return n, nil
}

func v15MultiAgentMaxConcurrencyFromEnv() (int, error) {
	raw := strings.TrimSpace(os.Getenv("TALLY_V15_MULTI_AGENT_MAX_CONCURRENCY"))
	if raw == "" {
		return 2, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("v1.5 control plane: TALLY_V15_MULTI_AGENT_MAX_CONCURRENCY must be an integer: %w", err)
	}
	if n < 1 {
		return 0, fmt.Errorf("v1.5 control plane: TALLY_V15_MULTI_AGENT_MAX_CONCURRENCY must be >= 1")
	}
	return n, nil
}
