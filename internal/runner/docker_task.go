package runner

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
	"time"
)

// DockerTaskProfile is the reproducible execution profile for one task runner.
// WorkspaceSource may be a host path or a named volume; WorkspaceTarget is the
// environment-local path visible to the runtime inside the container.
type DockerTaskProfile struct {
	Image           string
	Workdir         string
	WorkspaceSource string
	WorkspaceTarget string
	Network         string
	User            string
	ReadOnly        bool
	ExtraArgs       []string
}

// DockerTaskRunner adapts the existing docker CLI transport to task execution.
// It deliberately remains a Runner so runtimes do not acquire Docker-specific types.
type DockerTaskRunner struct {
	Bin        string
	Exec       Runner
	Profile    DockerTaskProfile
	NamePrefix string
	seq        atomic.Uint64
}

func (d *DockerTaskRunner) bin() string {
	if d.Bin == "" {
		return "docker"
	}
	return d.Bin
}

func (d *DockerTaskRunner) exec() Runner {
	if d.Exec == nil {
		return LocalRunner{}
	}
	return d.Exec
}

func (d *DockerTaskRunner) Run(ctx context.Context, cmd Command) (Result, error) {
	if strings.TrimSpace(d.Profile.Image) == "" {
		return Result{}, errors.New("runner: Docker task image is required")
	}
	if len(cmd.Argv) == 0 {
		return Result{}, errors.New("runner: 空命令")
	}
	if (d.Profile.WorkspaceSource == "") != (d.Profile.WorkspaceTarget == "") {
		return Result{}, errors.New("runner: Docker workspace source and target must be provided together")
	}

	prefix := d.NamePrefix
	if prefix == "" {
		prefix = "tally-task"
	}
	name := fmt.Sprintf("%s-%d", prefix, d.seq.Add(1))
	args := []string{d.bin(), "run", "--rm", "--name", name}
	if d.Profile.WorkspaceSource != "" {
		args = append(args, "--volume", d.Profile.WorkspaceSource+":"+d.Profile.WorkspaceTarget)
	}
	workdir := cmd.Dir
	if workdir == "" {
		workdir = d.Profile.Workdir
	}
	if workdir != "" {
		args = append(args, "--workdir", workdir)
	}
	if d.Profile.Network != "" {
		args = append(args, "--network", d.Profile.Network)
	}
	if d.Profile.User != "" {
		args = append(args, "--user", d.Profile.User)
	}
	if d.Profile.ReadOnly {
		args = append(args, "--read-only")
	}

	// Keep environment values (which may contain credentials) out of Docker CLI
	// argv. `docker run --env KEY` copies KEY from the docker client's environment,
	// so values travel through exec.Cmd.Env rather than process arguments/loggable
	// command lines. Duplicate keys are deterministic last-wins.
	envByKey := map[string]string{}
	for _, entry := range cmd.Env {
		key, _, ok := strings.Cut(entry, "=")
		if !ok {
			return Result{}, errors.New("runner: environment entry must use KEY=VALUE form")
		}
		if !envNameRE.MatchString(key) {
			return Result{}, fmt.Errorf("runner: invalid environment variable name %q", key)
		}
		envByKey[key] = entry
	}
	keys := make([]string, 0, len(envByKey))
	for key := range envByKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	outerEnv := make([]string, 0, len(keys))
	for _, key := range keys {
		args = append(args, "--env", key)
		outerEnv = append(outerEnv, envByKey[key])
	}

	args = append(args, d.Profile.ExtraArgs...)
	args = append(args, d.Profile.Image)
	args = append(args, cmd.Argv...)

	res, err := d.exec().Run(ctx, Command{Argv: args, Env: outerEnv})
	if err != nil && ctx.Err() != nil {
		d.cleanup(name)
	}
	return res, err
}

func (d *DockerTaskRunner) cleanup(name string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = d.exec().Run(ctx, Command{Argv: []string{d.bin(), "rm", "-f", name}})
}
