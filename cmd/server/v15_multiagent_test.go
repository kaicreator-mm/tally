package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"tally/internal/store"
)

func TestV15MultiAgentMaxConcurrencyFromEnv(t *testing.T) {
	t.Setenv("TALLY_V15_MULTI_AGENT_MAX_CONCURRENCY", "")
	if got, err := v15MultiAgentMaxConcurrencyFromEnv(); err != nil || got != 2 {
		t.Fatalf("default max concurrency=(%d,%v), want (2,nil)", got, err)
	}

	t.Setenv("TALLY_V15_MULTI_AGENT_MAX_CONCURRENCY", "4")
	if got, err := v15MultiAgentMaxConcurrencyFromEnv(); err != nil || got != 4 {
		t.Fatalf("configured max concurrency=(%d,%v), want (4,nil)", got, err)
	}

	for _, bad := range []string{"0", "-1", "not-an-int"} {
		t.Run(bad, func(t *testing.T) {
			t.Setenv("TALLY_V15_MULTI_AGENT_MAX_CONCURRENCY", bad)
			if _, err := v15MultiAgentMaxConcurrencyFromEnv(); err == nil {
				t.Fatalf("value %q unexpectedly accepted", bad)
			}
		})
	}
}

func TestV15ProductionCompositionEnablesMultiAgentOnlyWithExecutionRuntime(t *testing.T) {
	queries := &store.Queries{}
	contentRepo := t.TempDir()

	callPolicy := func(t *testing.T, runtime string) int {
		t.Helper()
		t.Setenv("TALLY_V15_RUNTIME", runtime)
		t.Setenv("TALLY_V15_STATE_ROOT", t.TempDir())
		t.Setenv("TALLY_V15_MULTI_AGENT_MAX_CONCURRENCY", "2")
		handler, _, err := v15ControlPlaneFromEnv(contentRepo, queries)
		if err != nil {
			t.Fatalf("compose v1.5 control plane: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/api/v15/projects/missing/tasks/T1/multi-agent/policy", bytes.NewBufferString(`{}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	if status := callPolicy(t, ""); status != http.StatusServiceUnavailable {
		t.Fatalf("without execution runtime multi-agent policy status=%d, want 503", status)
	}
	if status := callPolicy(t, "fake"); status != http.StatusNotFound {
		t.Fatalf("with fake runtime multi-agent policy status=%d, want 404 from repository after capability gate", status)
	}
}
