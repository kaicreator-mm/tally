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
	env := append([]string(nil), cmd.Env...)
	for _, entry := range env {
		key, _, ok := strings.Cut(entry, "=")
		if !ok || !envNameRE.MatchString(key) {
			return Result{}, fmt.Errorf("runner: 无效环境变量 %q", entry)
		}
	}
	sort.Strings(env)
	for _, entry := range env {
		args = append(args, "--env", entry)
	}
	args = append(args, d.Profile.ExtraArgs...)
	args = append(args, d.Profile.Image)
	args = append(args, cmd.Argv...)

	res, err := d.exec().Run(ctx, Command{Argv: args})
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
