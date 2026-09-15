package controlplane

import (
	"context"
	"errors"
	"testing"
	"time"

	"tally/internal/engineering"
	"tally/internal/intelligence"
	"tally/internal/replan"
)

func TestActivateReplanAdvancesPlanAndDAGAtomically(t *testing.T) {
	s, repo := newHarness(t, &fakeExecutor{}, &fakeWorkflow{})
	bootstrap(t, s)
	before, err := s.Snapshot(context.Background(), "project-1")
	if err != nil { t.Fatal(err) }
	proposal := replanProposal(t, before)

	after, err := s.ActivateReplan(context.Background(), "project-1", proposal, nil)
	if err != nil { t.Fatalf("ActivateReplan: %v", err) }
	if after.Plan.AcceptedRevision.ID != "plan-r2" || after.ActiveDAG.RevisionID != "dag-r2" {
		t.Fatalf("mixed or wrong revision: plan=%s dag=%s", after.Plan.AcceptedRevision.ID, after.ActiveDAG.RevisionID)
	}
	if len(after.ReplanHistory) != 1 { t.Fatalf("history=%d", len(after.ReplanHistory)) }
	if after.ReplanHistory[0].PreviousPlanning.ID != "plan-r1" || after.ReplanHistory[0].PreviousDAG.RevisionID != "dag-r1" {
		t.Fatalf("old baseline not retained: %+v", after.ReplanHistory[0])
	}
	reloaded, err := repo.Load(context.Background(), "project-1")
	if err != nil { t.Fatalf("reload: %v", err) }
	if reloaded.Plan.AcceptedRevision.ID != "plan-r2" || len(reloaded.ReplanHistory) != 1 {
		t.Fatalf("reloaded state lost replan: %+v", reloaded)
	}
	if _, err := s.AcceptPlan(context.Background(), "project-1", plan(t)); !errors.Is(err, ErrImmutableStage) {
		t.Fatalf("v1.5.0 AcceptPlan immutability weakened: %v", err)
	}
}

func TestActivateReplanRejectsStaleParent(t *testing.T) {
	s, _ := newHarness(t, &fakeExecutor{}, &fakeWorkflow{})
	bootstrap(t, s)
	before, _ := s.Snapshot(context.Background(), "project-1")
	proposal := replanProposal(t, before)
	if _, err := s.ActivateReplan(context.Background(), "project-1", proposal, nil); err != nil { t.Fatal(err) }
	if _, err := s.ActivateReplan(context.Background(), "project-1", proposal, nil); !errors.Is(err, replan.ErrStaleParent) {
		t.Fatalf("expected stale parent, got %v", err)
	}
}

func TestActivateReplanRequiresIndependentAuthorityForScopeChange(t *testing.T) {
	s, _ := newHarness(t, &fakeExecutor{}, &fakeWorkflow{})
	bootstrap(t, s)
	before, _ := s.Snapshot(context.Background(), "project-1")
	proposal := replanProposal(t, before)
	proposal.Impact.ScopeChanged = true
	if _, err := s.ActivateReplan(context.Background(), "project-1", proposal, nil); !errors.Is(err, replan.ErrAuthorityRequired) {
		t.Fatalf("expected authority requirement, got %v", err)
	}
	requirementID := engineering.ID("replan-authority-" + string(proposal.ID))
	self := engineering.AuthorityDecision{RequirementID: requirementID, Status: engineering.AuthorityAuthorized, Actor: proposal.Provider.ProviderID, At: time.Now()}
	if _, err := s.ActivateReplan(context.Background(), "project-1", proposal, []engineering.AuthorityDecision{self}); !errors.Is(err, replan.ErrSelfAuthorization) {
		t.Fatalf("expected self authorization rejection, got %v", err)
	}
	human := self
	human.Actor = "product-owner"
	if _, err := s.ActivateReplan(context.Background(), "project-1", proposal, []engineering.AuthorityDecision{human}); err != nil {
		t.Fatalf("authorized replan: %v", err)
	}
}

func replanProposal(t *testing.T, snap ProjectSnapshot) replan.Proposal {
	t.Helper()
	now := time.Date(2026, 9, 15, 9, 30, 0, 0, time.UTC)
	r2, err := engineering.NewPlanningRevision("plan-r2", snap.Plan.AcceptedRevision.Number+1, snap.Plan.AcceptedRevision.ID, []string{"build", "re-plan"}, artifact("plan-art-r2", engineering.ArtifactProjectPlan), engineering.Provenance{Actor: "direct-test", Source: "test", At: now})
	if err != nil { t.Fatal(err) }
	contracts := map[engineering.ID]engineering.CompletionContract{}
	for id, c := range snap.Contracts { contracts[id] = c }
	contracts["C"] = engineering.CompletionContract{RequiredState: engineering.NodeAccepted}
	return replan.Proposal{
		ID: "proposal-r2",
		ParentPlanRevision: snap.Plan.AcceptedRevision.ID,
		ParentDAGRevision: snap.ActiveDAG.RevisionID,
		Planning: r2,
		DAG: engineering.TaskDAGProposal{RevisionID: "dag-r2", Nodes: []engineering.TaskNode{
			{ID: "A"},
			{ID: "B", Dependencies: []engineering.ID{"A"}},
			{ID: "C", Dependencies: []engineering.ID{"A"}, ParallelEligible: true},
		}},
		Contracts: contracts,
		Dispositions: []replan.TaskDisposition{
			{TaskID: "A", Kind: replan.DispositionRetained},
			{TaskID: "B", Kind: replan.DispositionRetained},
			{TaskID: "C", Kind: replan.DispositionNew},
		},
		Triggers: []replan.Trigger{{Kind: replan.TriggerExplicitHumanRequest, HumanActor: "owner", Provenance: engineering.Provenance{Actor: "owner", Source: "human-request", At: now}}},
		Provider: intelligence.ProviderProvenance{Kind: intelligence.ProviderDirectLLM, ProviderID: "direct-test", ObservedAt: now},
		Provenance: engineering.Provenance{Actor: "direct-test", Source: "intelligence:direct_llm", At: now},
	}
}
