package airuntimelegacy

import (
	"context"
	"errors"
	"testing"

	"tally/internal/airuntime"
)

func TestNilClientFailsClosed(t *testing.T) {
	_, err := (executor{}).Execute(context.Background(), airuntime.Request{})
	if !errors.Is(err, ErrNilClient) {
		t.Fatalf("nil client should fail closed with ErrNilClient, got %v", err)
	}
}
