package taskexecution

import (
	"context"
	"errors"
	"testing"

	"tally/internal/runner"
)

func TestV152WorkspaceRecreateTransportFailureIsEnvironmentUnavailable(t *testing.T) {
	f := &v152ScriptedRunner{errs: []error{errors.New("ssh transport down")}}
	recreate, err := NewGitWorkspaceRecreator(f, "git@example.invalid:repo.git")
	if err != nil {
		t.Fatal(err)
	}
	err = recreate(context.Background(), v152RemoteReq().Workspace)
	var typed *EnvironmentError
	if !errors.As(err, &typed) || typed.Kind != FailureEnvironmentUnavailable {
		t.Fatalf("err=%T %v", err, err)
	}
	if typed.Operation != "workspace.recreate.cleanup" {
		t.Fatalf("operation=%q", typed.Operation)
	}
}

func TestV152WorkspaceRecreateCommandFailureIsWorkspaceUnavailable(t *testing.T) {
	f := &v152ScriptedRunner{results: []runner.Result{{ExitCode: 0}, {ExitCode: 128, Stderr: "clone failed"}}}
	recreate, err := NewGitWorkspaceRecreator(f, "git@example.invalid:repo.git")
	if err != nil {
		t.Fatal(err)
	}
	err = recreate(context.Background(), v152RemoteReq().Workspace)
	var typed *EnvironmentError
	if !errors.As(err, &typed) || typed.Kind != FailureWorkspaceUnavailable {
		t.Fatalf("err=%T %v", err, err)
	}
	if typed.Operation != "workspace.recreate.clone" {
		t.Fatalf("operation=%q", typed.Operation)
	}
}
