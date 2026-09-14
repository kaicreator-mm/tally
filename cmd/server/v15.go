package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"tally/internal/agentruntime"
	"tally/internal/controlplane"
	"tally/internal/taskexecution"
	"tally/runtimes/codex"
	"tally/runtimes/dsh"
	"tally/runtimes/fake"
)

func v15ControlPlaneFromEnv() (http.Handler, string, error) {
	repo, err := controlplane.NewFileRepository(envOr("TALLY_V15_STATE_ROOT", "./tally-state/v1.5"))
	if err != nil {
		return nil, "", err
	}

	runtimeName := strings.TrimSpace(os.Getenv("TALLY_V15_RUNTIME"))
	var executor taskexecution.AgentExecutor
	if runtimeName != "" {
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
		executor, err = taskexecution.NewRuntimeExecutor(runtime, taskexecution.LocalEnvironment{})
		if err != nil {
			return nil, "", fmt.Errorf("v1.5 control plane: build Local AgentExecutor: %w", err)
		}
	}

	// Verification/review is deliberately not fabricated here. Until the
	// brownfield verifier adapter is injected, a completed Agent claim is persisted
	// as blocked with ErrWorkflowUnavailable rather than promoted to project fact.
	service, err := controlplane.NewService(repo, executor, nil)
	if err != nil {
		return nil, "", err
	}
	handler, err := controlplane.NewHTTPHandler(service)
	if err != nil {
		return nil, "", err
	}
	return handler, runtimeName, nil
}
