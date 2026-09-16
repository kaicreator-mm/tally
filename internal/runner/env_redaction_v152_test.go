package runner

import (
	"context"
	"strings"
	"testing"
)

func TestV152SSHInvalidEnvErrorDoesNotExposeValue(t *testing.T) {
	const secret = "do-not-log-this-secret"
	r := &SSHRunner{RunRemote: func(context.Context, SSHConfig, string) (Result, error) {
		t.Fatal("invalid env must fail before transport")
		return Result{}, nil
	}}
	_, err := r.Run(context.Background(), Command{Argv: []string{"true"}, Env: []string{"BAD-NAME=" + secret}})
	if err == nil {
		t.Fatal("expected invalid env error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("secret leaked in SSH env validation error: %v", err)
	}
}

func TestV152DockerInvalidEnvErrorDoesNotExposeValue(t *testing.T) {
	const secret = "do-not-log-this-secret"
	f := &v152CaptureRunner{}
	d := &DockerTaskRunner{Exec: f, Profile: DockerTaskProfile{Image: "alpine"}}
	_, err := d.Run(context.Background(), Command{Argv: []string{"true"}, Env: []string{"BAD-NAME=" + secret}})
	if err == nil {
		t.Fatal("expected invalid env error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("secret leaked in Docker env validation error: %v", err)
	}
	if len(f.calls) != 0 {
		t.Fatal("invalid env reached Docker transport")
	}
}
