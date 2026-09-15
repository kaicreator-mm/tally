package runner

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type v152CaptureRunner struct {
	calls []Command
	err   error
}

func (r *v152CaptureRunner) Run(_ context.Context, c Command) (Result, error) {
	r.calls = append(r.calls, c)
	return Result{}, r.err
}

func TestV152SSHRunnerPreservesCwdEnvAndArgv(t *testing.T) {
	var got string
	r := &SSHRunner{Config: SSHConfig{Host: "h"}, RunRemote: func(_ context.Context, _ SSHConfig, command string) (Result, error) {
		got = command
		return Result{}, nil
	}}
	_, err := r.Run(context.Background(), Command{Argv: []string{"tool", "a b", "it's"}, Dir: "/srv/a b", Env: []string{"A=x y", "B=it's"}})
	if err != nil {
		t.Fatal(err)
	}
	want := "cd -- '/srv/a b' && exec env 'A=x y' 'B=it'\"'\"'s' tool 'a b' 'it'\"'\"'s'"
	if got != want {
		t.Fatalf("remote command\n got: %s\nwant: %s", got, want)
	}
}

func TestV152SSHRunnerRejectsInvalidEnvName(t *testing.T) {
	r := &SSHRunner{RunRemote: func(context.Context, SSHConfig, string) (Result, error) {
		t.Fatal("must not run")
		return Result{}, nil
	}}
	if _, err := r.Run(context.Background(), Command{Argv: []string{"true"}, Env: []string{"BAD-NAME=x"}}); err == nil {
		t.Fatal("expected error")
	}
}

func TestV152DockerTaskRunnerBuildsProfileArgs(t *testing.T) {
	f := &v152CaptureRunner{}
	d := &DockerTaskRunner{Exec: f, NamePrefix: "case", Profile: DockerTaskProfile{Image: "golang:1.23", Workdir: "/workspace", WorkspaceSource: "volume-name", WorkspaceTarget: "/workspace", Network: "none", User: "1000:1000", ReadOnly: true, ExtraArgs: []string{"--cpus", "2"}}}
	if _, err := d.Run(context.Background(), Command{Argv: []string{"go", "test", "./..."}, Env: []string{"Z=2", "A=1"}}); err != nil {
		t.Fatal(err)
	}
	want := []string{"docker", "run", "--rm", "--name", "case-1", "--volume", "volume-name:/workspace", "--workdir", "/workspace", "--network", "none", "--user", "1000:1000", "--read-only", "--env", "A=1", "--env", "Z=2", "--cpus", "2", "golang:1.23", "go", "test", "./..."}
	if !reflect.DeepEqual(f.calls[0].Argv, want) {
		t.Fatalf("got %#v\nwant %#v", f.calls[0].Argv, want)
	}
}

func TestV152DockerTaskRunnerCancelsAndCleansContainer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f := &v152CaptureRunner{err: context.Canceled}
	d := &DockerTaskRunner{Exec: f, NamePrefix: "case", Profile: DockerTaskProfile{Image: "alpine"}}
	_, err := d.Run(ctx, Command{Argv: []string{"sleep", "10"}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
	if len(f.calls) != 2 {
		t.Fatalf("calls=%d", len(f.calls))
	}
	want := []string{"docker", "rm", "-f", "case-1"}
	if !reflect.DeepEqual(f.calls[1].Argv, want) {
		t.Fatalf("cleanup=%v", f.calls[1].Argv)
	}
}
