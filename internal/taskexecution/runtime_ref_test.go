package taskexecution

import (
	"context"
	"errors"
	"testing"
)

func TestRuntimeExecutorRejectsSpoofedRuntimeReferenceBeforeStart(t *testing.T) {
	rt := &fakeRuntime{}
	executor, err := NewRuntimeExecutor(rt, LocalEnvironment{})
	if err != nil {
		t.Fatal(err)
	}

	request := req()
	request.Runtime.ID = "different-provider"
	if _, err := executor.Execute(context.Background(), request); !errors.Is(err, ErrRuntimeRefMismatch) {
		t.Fatalf("Execute error=%v, want ErrRuntimeRefMismatch", err)
	}
	if len(rt.starts) != 0 {
		t.Fatalf("runtime started despite spoofed RuntimeRef: starts=%d", len(rt.starts))
	}
}

func TestRuntimeExecutorRejectsMissingRuntimeReferenceBeforeStart(t *testing.T) {
	rt := &fakeRuntime{}
	executor, err := NewRuntimeExecutor(rt, LocalEnvironment{})
	if err != nil {
		t.Fatal(err)
	}

	request := req()
	request.Runtime.ID = ""
	if _, err := executor.Execute(context.Background(), request); !errors.Is(err, ErrRuntimeRefMismatch) {
		t.Fatalf("Execute error=%v, want ErrRuntimeRefMismatch", err)
	}
	if len(rt.starts) != 0 {
		t.Fatalf("runtime started despite missing RuntimeRef: starts=%d", len(rt.starts))
	}
}
