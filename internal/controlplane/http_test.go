package controlplane

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tally/internal/taskexecution"
)

func TestHTTPHandlerSnapshotAndDispatchPathIdentity(t *testing.T) {
	s, _ := newHarness(t, &fakeExecutor{}, &fakeWorkflow{})
	bootstrap(t, s)
	h, err := NewHTTPHandler(s)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v15/projects/project-1/control-plane", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("GET status=%d body=%s", res.Code, res.Body.String())
	}
	var snap ProjectSnapshot
	if err := json.NewDecoder(res.Body).Decode(&snap); err != nil {
		t.Fatal(err)
	}
	if snap.ProjectID != "project-1" || snap.ActiveDAG == nil {
		t.Fatalf("snapshot=%+v", snap)
	}

	body, err := json.Marshal(request("A"))
	if err != nil {
		t.Fatal(err)
	}
	// Body says A while path says B: task identity must not be ambiguous.
	req = httptest.NewRequest(http.MethodPost, "/api/v15/projects/project-1/tasks/B/dispatch", strings.NewReader(string(body)))
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("dispatch mismatch status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestHTTPHandlerRejectsUnknownJSONFields(t *testing.T) {
	s, _ := newHarness(t, &fakeExecutor{result: taskexecution.TaskExecutionResult{Status: taskexecution.ExecutionCompleted}}, &fakeWorkflow{})
	h, err := NewHTTPHandler(s)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v15/projects/p/requirements", strings.NewReader(`{"unknown":true}`))
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}
