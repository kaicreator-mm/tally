package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"tally/internal/controlplane"
	"tally/internal/engineering"
	"tally/internal/intelligence"
	"tally/internal/replan"
)

// This is the cross-slice compatibility test between v1.5.5 Re-plan and the
// later v1.5.1 SQLite-first composition. SQLite stores the complete governed
// ProjectSnapshot, so immutable re-plan ancestry must survive a real close/open
// cycle together with the newly active Plan and TaskDAG.
func TestReplanHistoryRoundTripAcrossSQLiteRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "replan.db")
	store, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	service, err := controlplane.NewService(store, nil, nil)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	at := time.Date(2026, 9, 16, 0, 30, 0, 0, time.UTC)
	prov := engineering.Provenance{Actor: "owner", Source: "sqlite-v155-test", At: at}
	requirements, err := engineering.NewRequirementsBaseline(
		"req-v1", "r1",
		[]engineering.Requirement{{ID: "REQ-1", Statement: "build", AcceptanceCriteria: []string{"done"}}},
		v155Artifact("req-art", engineering.ArtifactRequirements), prov,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.AcceptRequirements(ctx, "project-v155", requirements); err != nil {
		t.Fatalf("AcceptRequirements: %v", err)
	}

	planR1, err := engineering.NewPlanningRevision("plan-r1", 1, "", []string{"build"}, v155Artifact("plan-art-r1", engineering.ArtifactProjectPlan), prov)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.AcceptPlan(ctx, "project-v155", engineering.ProjectPlan{ID: "plan", AcceptedRevision: planR1}); err != nil {
		t.Fatalf("AcceptPlan: %v", err)
	}

	architecture, err := engineering.NewArchitectureRevision("arch-r1", engineering.ArchitectureDesign, "design", nil, v155Artifact("arch-art", engineering.ArtifactArchitecture), prov)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.AcceptArchitecture(ctx, "project-v155", architecture); err != nil {
		t.Fatalf("AcceptArchitecture: %v", err)
	}

	contractA := engineering.CompletionContract{RequiredState: engineering.NodeAccepted}
	if _, err := service.ActivateDAG(
		ctx,
		"project-v155",
		engineering.TaskDAGProposal{RevisionID: "dag-r1", Nodes: []engineering.TaskNode{{ID: "A"}}},
		map[engineering.ID]engineering.CompletionContract{"A": contractA},
	); err != nil {
		t.Fatalf("ActivateDAG: %v", err)
	}

	planR2, err := engineering.NewPlanningRevision("plan-r2", 2, "plan-r1", []string{"build", "re-plan"}, v155Artifact("plan-art-r2", engineering.ArtifactProjectPlan), prov)
	if err != nil {
		t.Fatal(err)
	}
	proposal := replan.Proposal{
		ID:                 "replan-r2",
		ParentPlanRevision: "plan-r1",
		ParentDAGRevision:  "dag-r1",
		Planning:           planR2,
		DAG: engineering.TaskDAGProposal{RevisionID: "dag-r2", Nodes: []engineering.TaskNode{
			{ID: "A"},
			{ID: "B", Dependencies: []engineering.ID{"A"}},
		}},
		Contracts: map[engineering.ID]engineering.CompletionContract{
			"A": contractA,
			"B": {RequiredState: engineering.NodeAccepted},
		},
		Dispositions: []replan.TaskDisposition{
			{TaskID: "A", Kind: replan.DispositionRetained},
			{TaskID: "B", Kind: replan.DispositionNew},
		},
		Triggers: []replan.Trigger{{
			Kind:       replan.TriggerExplicitHumanRequest,
			HumanActor: "owner",
			Provenance: prov,
		}},
		Provider: intelligence.ProviderProvenance{
			Kind:       intelligence.ProviderDirectLLM,
			ProviderID: "direct-test",
			ObservedAt: at,
		},
		Provenance: engineering.Provenance{Actor: "direct-test", Source: "intelligence:direct_llm", At: at},
	}
	activated, err := service.ActivateReplan(ctx, "project-v155", proposal, nil)
	if err != nil {
		t.Fatalf("ActivateReplan: %v", err)
	}
	if len(activated.ReplanHistory) != 1 {
		t.Fatalf("history before restart=%d, want 1", len(activated.ReplanHistory))
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	store, err = Open(ctx, path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer store.Close()
	got, err := store.Load(ctx, "project-v155")
	if err != nil {
		t.Fatalf("Load after restart: %v", err)
	}
	if got.Plan == nil || got.Plan.AcceptedRevision.ID != "plan-r2" || got.ActiveDAG == nil || got.ActiveDAG.RevisionID != "dag-r2" {
		t.Fatalf("active replan state lost across restart: plan=%+v dag=%+v", got.Plan, got.ActiveDAG)
	}
	if len(got.ReplanHistory) != 1 {
		t.Fatalf("history after restart=%d, want 1", len(got.ReplanHistory))
	}
	history := got.ReplanHistory[0]
	if history.PreviousPlanning.ID != "plan-r1" || history.PreviousDAG.RevisionID != "dag-r1" || history.ActivatedPlanning.ID != "plan-r2" || history.ActivatedDAG.RevisionID != "dag-r2" {
		t.Fatalf("replan ancestry changed across SQLite restart: %+v", history)
	}
}

func v155Artifact(id engineering.ID, kind engineering.ArtifactKind) engineering.ArtifactRef {
	return engineering.ArtifactRef{ID: id, Kind: kind, URI: "git://repo/" + string(id)}
}
