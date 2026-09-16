package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"tally/internal/agentruntime"
	"tally/internal/runner"
	"tally/internal/taskexecution"
	"tally/runtimes/codex"
	"tally/runtimes/dsh"
	"tally/runtimes/fake"
)

type executionComposition struct {
	Executor      taskexecution.AgentExecutor
	Kind          taskexecution.EnvironmentKind
	Target        string
	WorkspaceRoot string
}

func executionFromEnv(runtimeName, contentRepo string) (executionComposition, error) {
	runtimeAdapter, err := runtimeFromEnv(runtimeName)
	if err != nil {
		return executionComposition{}, err
	}
	kind := taskexecution.EnvironmentKind(strings.ToLower(strings.TrimSpace(envOr("TALLY_V15_ENVIRONMENT", string(taskexecution.EnvironmentLocal)))))
	workspaceRoot := strings.TrimSpace(os.Getenv("TALLY_V15_WORKSPACE_ROOT"))
	profile, err := environmentProfileFromEnv(runtimeName)
	if err != nil {
		return executionComposition{}, err
	}

	var environment taskexecution.ExecutionEnvironment
	var target string
	switch kind {
	case taskexecution.EnvironmentLocal:
		if workspaceRoot == "" {
			return executionComposition{}, errors.New("v1.5 core: TALLY_V15_WORKSPACE_ROOT is required for Local execution")
		}
		target = envOr("TALLY_V15_ENV_TARGET", "local")
		environment = taskexecution.LocalEnvironment{}

	case taskexecution.EnvironmentSSH:
		host := strings.TrimSpace(os.Getenv("TALLY_V15_SSH_HOST"))
		user := strings.TrimSpace(os.Getenv("TALLY_V15_SSH_USER"))
		if host == "" || user == "" {
			return executionComposition{}, errors.New("v1.5 core: SSH execution requires TALLY_V15_SSH_HOST and TALLY_V15_SSH_USER")
		}
		if workspaceRoot == "" {
			return executionComposition{}, errors.New("v1.5 core: SSH execution requires environment-local TALLY_V15_WORKSPACE_ROOT")
		}
		cfg := runner.SSHConfig{
			Host:           host,
			User:           user,
			Password:       os.Getenv("TALLY_V15_SSH_PASSWORD"),
			KeyFile:        os.Getenv("TALLY_V15_SSH_KEY_FILE"),
			KnownHostsFile: os.Getenv("TALLY_V15_SSH_KNOWN_HOSTS"),
		}
		if cfg.Password == "" && cfg.KeyFile == "" {
			return executionComposition{}, errors.New("v1.5 core: SSH execution requires TALLY_V15_SSH_KEY_FILE or TALLY_V15_SSH_PASSWORD")
		}
		target = envOr("TALLY_V15_ENV_TARGET", host)
		exec := runner.NewSSHRunner(cfg)
		recreateRepo := envOr("TALLY_V15_ENV_RECREATE_REPO", contentRepo)
		recreate, recreateErr := taskexecution.NewGitWorkspaceRecreator(exec, recreateRepo)
		if recreateErr != nil {
			return executionComposition{}, fmt.Errorf("v1.5 core: SSH workspace recovery: %w", recreateErr)
		}
		environment = taskexecution.NewSSHEnvironment(target, exec, profile, recreate)

	case taskexecution.EnvironmentDocker:
		image := strings.TrimSpace(os.Getenv("TALLY_V15_DOCKER_IMAGE"))
		source := strings.TrimSpace(os.Getenv("TALLY_V15_DOCKER_WORKSPACE_SOURCE"))
		if image == "" || source == "" {
			return executionComposition{}, errors.New("v1.5 core: Docker execution requires TALLY_V15_DOCKER_IMAGE and TALLY_V15_DOCKER_WORKSPACE_SOURCE")
		}
		targetRoot := strings.TrimSpace(os.Getenv("TALLY_V15_DOCKER_WORKSPACE_TARGET"))
		if targetRoot == "" {
			targetRoot = "/workspace"
		}
		workspaceRoot = targetRoot
		target = envOr("TALLY_V15_ENV_TARGET", "docker")
		network := strings.TrimSpace(os.Getenv("TALLY_V15_DOCKER_NETWORK"))
		profile.Network = network != "none"
		exec := &runner.DockerTaskRunner{
			Bin: os.Getenv("TALLY_V15_DOCKER_BIN"),
			Profile: runner.DockerTaskProfile{
				Image:           image,
				Workdir:         targetRoot,
				WorkspaceSource: source,
				WorkspaceTarget: targetRoot,
				Network:         network,
				User:            os.Getenv("TALLY_V15_DOCKER_USER"),
				ReadOnly:        envBool("TALLY_V15_DOCKER_READ_ONLY"),
			},
			NamePrefix: "tally-v152",
		}
		recreateRepo := envOr("TALLY_V15_ENV_RECREATE_REPO", contentRepo)
		recreate, recreateErr := taskexecution.NewGitWorkspaceRecreator(exec, recreateRepo)
		if recreateErr != nil {
			return executionComposition{}, fmt.Errorf("v1.5 core: Docker workspace recovery: %w", recreateErr)
		}
		environment = taskexecution.NewDockerEnvironment(target, exec, profile, recreate)

	default:
		return executionComposition{}, fmt.Errorf("v1.5 core: unsupported TALLY_V15_ENVIRONMENT %q (want local|ssh|docker)", kind)
	}

	if kind != taskexecution.EnvironmentLocal {
		if _, ok := runtimeAdapter.(taskexecution.EnvironmentRuntime); !ok {
			return executionComposition{}, fmt.Errorf("v1.5 core: runtime %q does not support non-local execution environment %q", runtimeName, kind)
		}
	}
	executor, err := taskexecution.NewRuntimeExecutor(runtimeAdapter, environment)
	if err != nil {
		return executionComposition{}, fmt.Errorf("v1.5 core: build %s AgentExecutor: %w", kind, err)
	}
	return executionComposition{Executor: executor, Kind: kind, Target: target, WorkspaceRoot: workspaceRoot}, nil
}

func runtimeFromEnv(runtimeName string) (agentruntime.Runtime, error) {
	isolationVerified := envBool("TALLY_V15_SESSION_ISOLATION_VERIFIED")
	switch runtimeName {
	case "fake":
		return fake.New(fake.Config{}), nil
	case "codex":
		return codex.New(codex.Config{
			BinPath:                  os.Getenv("TALLY_V15_CODEX_BIN"),
			Model:                    os.Getenv("TALLY_V15_MODEL"),
			SessionIsolationVerified: isolationVerified,
		}), nil
	case "dsh":
		env := map[string]string{}
		if key := os.Getenv("DEEPSEEK_API_KEY"); key != "" {
			env["DEEPSEEK_API_KEY"] = key
		}
		return dsh.New(dsh.Config{
			BinPath:                  os.Getenv("TALLY_V15_DSH_BIN"),
			Model:                    os.Getenv("TALLY_V15_MODEL"),
			Provider:                 os.Getenv("TALLY_V15_DSH_PROVIDER"),
			Env:                      env,
			SessionIsolationVerified: isolationVerified,
		}), nil
	default:
		return nil, fmt.Errorf("v1.5 core: unsupported TALLY_V15_RUNTIME %q (want fake|codex|dsh)", runtimeName)
	}
}

// DSH's observed headless protocol does not declare agentruntime model.select;
// its requested model/provider are applied by the DSH runtime's generated patch
// overlay. Propagating TALLY_V15_MODEL into StartRequest.ModelProfile would make
// CheckStart correctly reject DSH before execution, so DSH keeps this field empty.
func requestModelProfile(runtimeName string) string {
	if runtimeName == "dsh" {
		return ""
	}
	return os.Getenv("TALLY_V15_MODEL")
}

func environmentProfileFromEnv(runtimeName string) (taskexecution.EnvironmentProfile, error) {
	profile := taskexecution.EnvironmentProfile{
		Network:   envBool("TALLY_V15_ENV_NETWORK"),
		Secrets:   map[string]bool{},
		Resources: map[string]string{},
	}
	if runtimeName == "dsh" && os.Getenv("DEEPSEEK_API_KEY") != "" {
		profile.Secrets["DEEPSEEK_API_KEY"] = true
	}
	for _, raw := range strings.Split(os.Getenv("TALLY_V15_ENV_SECRETS"), ",") {
		name := strings.TrimSpace(raw)
		if name != "" {
			profile.Secrets[name] = true
		}
	}
	resources, err := parseKeyValueList(os.Getenv("TALLY_V15_ENV_RESOURCES"))
	if err != nil {
		return taskexecution.EnvironmentProfile{}, fmt.Errorf("v1.5 core: TALLY_V15_ENV_RESOURCES: %w", err)
	}
	profile.Resources = resources
	return profile, nil
}

func parseKeyValueList(raw string) (map[string]string, error) {
	out := map[string]string{}
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		key, value, ok := strings.Cut(item, "=")
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if !ok || key == "" || value == "" {
			return nil, fmt.Errorf("invalid entry %q (want name=value)", item)
		}
		out[key] = value
	}
	return out, nil
}

func envBool(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
