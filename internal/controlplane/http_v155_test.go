package controlplane

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tally/internal/replan"
)

func TestHTTPHandlerReplanUsesGovernedServicePath(t *testing.T) {
	s, _ := newHarness(t, &fakeExecutor{}, &fakeWorkflow{})
	bootstrap(t, s)
	before, err := s.Snapshot(t.Context(), "project-1")
	if err != nil {
		t.Fatal(err)
	}
	proposal := replanProposal(t, before)
	body, err := json.Marshal(replanActivationRequest{Proposal: proposal})
	if err != nil {
		t.Fatal(err)
	}
	h, err := NewHTTPHandler(s)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v15/projects/project-1/replan", strings.NewReader(string(body)))
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("first replan status=%d body=%s", res.Code, res.Body.String())
	}
	var after ProjectSnapshot
	if err := json.NewDecoder(res.Body).Decode(&after); err != nil {
		t.Fatal(err)
	}
	if after.Plan == nil || after.Plan.AcceptedRevision.ID != "plan-r2" || after.ActiveDAG == nil || after.ActiveDAG.RevisionID != "dag-r2" {
		t.Fatalf("HTTP replan did not publish one governed plan/DAG revision: %+v", after)
	}
	if len(after.ReplanHistory) != 1 {
		t.Fatalf("HTTP replan history=%d, want 1", len(after.ReplanHistory))
	}

	// Replaying the exact same parent through HTTP must hit the service's stale
	// parent guard. The handler cannot bypass/reimplement re-plan validation.
	req = httptest.NewRequest(http.MethodPost, "/api/v15/projects/project-1/replan", strings.NewReader(string(body)))
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusConflict {
		t.Fatalf("stale replan status=%d body=%s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), replan.ErrStaleParent.Error()) {
		t.Fatalf("stale response lost typed service error: %s", res.Body.String())
	}
}
