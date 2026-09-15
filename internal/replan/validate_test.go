package replan

import (
	"errors"
	"testing"
	"time"

	"tally/internal/engineering"
	"tally/internal/intelligence"
)

func TestValidatePreservesAcceptedWorkAndRecalculatesReadiness(t *testing.T) {
	current, proposal := fixture(t)
	decision, err := Validate(current, proposal, DefaultCyclePolicy(), CycleLedger{})
	if err != nil { t.Fatalf("Validate: %v", err) }
	if !decision.Validated() { t.Fatal("expected validated decision") }
	got := decision.DAG()
	if stateOf(got, "accepted") != engineering.TaskDone { t.Fatalf("accepted task state = %s", stateOf(got, "accepted")) }
	if stateOf(got, "new") != engineering.TaskReady { t.Fatalf("new task state = %s", stateOf(got, "new")) }
	if len(decision.RequiredAuthority()) != 0 { t.Fatal("compatible replan should not require authority") }
}

func TestValidateRejectsSilentAcceptedWorkChange(t *testing.T) {
	current, proposal := fixture(t)
	proposal.DAG.Nodes[0].VerificationRequirements = []string{"changed"}
	_, err := Validate(current, proposal, DefaultCyclePolicy(), CycleLedger{})
	if !errors.Is(err, ErrAcceptedWork) { t.Fatalf("expected ErrAcceptedWork, got %v", err) }
}

func TestValidateRejectsUnaccountedTask(t *testing.T) {
	current, proposal := fixture(t)
	proposal.Dispositions = proposal.Dispositions[1:]
	_, err := Validate(current, proposal, DefaultCyclePolicy(), CycleLedger{})
	if !errors.Is(err, ErrInvalidProposal) { t.Fatalf("expected ErrInvalidProposal, got %v", err) }
}

func TestCycleFingerprintCannotBeEvadedByNewIdentity(t *testing.T) {
	current, proposal := fixture(t)
	fp, err := Fingerprint(proposal)
	if err != nil { t.Fatal(err) }
	proposal.ID = "proposal-other"
	proposal.DAG.RevisionID = "dag-other"
	proposal.Provider.ProviderID = "another-provider"
	proposal.Provider.ObservedAt = proposal.Provider.ObservedAt.Add(time.Hour)
	fp2, err := Fingerprint(proposal)
	if err != nil { t.Fatal(err) }
	if fp != fp2 { t.Fatalf("equivalent proposal fingerprint changed: %s != %s", fp, fp2) }
	_, err = Validate(current, proposal, CyclePolicy{MaxReplans: 8, MaxSameFingerprint: 1}, CycleLedger{Activations: 1, Fingerprints: []string{fp}})
	if !errors.Is(err, ErrCycleBudget) { t.Fatalf("expected ErrCycleBudget, got %v", err) }
}

func TestMaterialReplanRequiresIndependentAuthority(t *testing.T) {
	current, proposal := fixture(t)
	proposal.Impact.ScopeChanged = true
	decision, err := Validate(current, proposal, DefaultCyclePolicy(), CycleLedger{})
	if err != nil { t.Fatal(err) }
	required := decision.RequiredAuthority()
	if len(required) != 1 { t.Fatalf("required authority = %d", len(required)) }
	self := engineering.AuthorityDecision{RequirementID: required[0].ID, Status: engineering.AuthorityAuthorized, Actor: proposal.Provider.ProviderID, At: time.Now()}
	if !errors.Is(CheckAuthority(decision, []engineering.AuthorityDecision{self}), ErrSelfAuthorization) {
		t.Fatal("provider self-authorization must be rejected")
	}
	human := self
	human.Actor = "human-owner"
	if err := CheckAuthority(decision, []engineering.AuthorityDecision{human}); err != nil { t.Fatalf("human authority: %v", err) }
}

func TestExplicitInvalidationRequiresDeclaredImpact(t *testing.T) {
	current, proposal := fixture(t)
	proposal.DAG.Nodes = proposal.DAG.Nodes[1:]
	proposal.DAG.Nodes[0].Dependencies = nil
	proposal.Dispositions = []TaskDisposition{
		{TaskID: "accepted", Kind: DispositionInvalidated, ReplacementID: "new", Reason: "architecture invalidated"},
		{TaskID: "new", Kind: DispositionNew},
	}
	delete(proposal.Contracts, "accepted")
	_, err := Validate(current, proposal, DefaultCyclePolicy(), CycleLedger{})
	if !errors.Is(err, ErrAcceptedWork) { t.Fatalf("expected declared accepted-work impact requirement, got %v", err) }
	proposal.Impact.AcceptedWorkInvalidated = true
	decision, err := Validate(current, proposal, DefaultCyclePolicy(), CycleLedger{})
	if err != nil { t.Fatalf("explicit invalidation: %v", err) }
	if len(decision.RequiredAuthority()) == 0 { t.Fatal("accepted-work invalidation must require authority") }
}

func fixture(t *testing.T) (CurrentState, Proposal) {
	t.Helper()
	now := time.Date(2026, 9, 15, 9, 0, 0, 0, time.UTC)
	plan1, err := engineering.NewPlanningRevision("plan-1", 1, "", []string{"build safely"}, artifact("plan-art-1", engineering.ArtifactProjectPlan), provenance("human", now))
	if err != nil { t.Fatal(err) }
	old, err := engineering.ActivateTaskDAG(engineering.TaskDAGProposal{RevisionID: "dag-1", Nodes: []engineering.TaskNode{{ID: "accepted", ParallelEligible: true, VerificationRequirements: []string{"unit"}}}})
	if err != nil { t.Fatal(err) }
	old, err = old.Transition("accepted", engineering.TaskRunning, "")
	if err != nil { t.Fatal(err) }
	old, err = old.Transition("accepted", engineering.TaskAccepted, "")
	if err != nil { t.Fatal(err) }
	old, err = old.Transition("accepted", engineering.TaskDone, "")
	if err != nil { t.Fatal(err) }
	contracts := map[engineering.ID]engineering.CompletionContract{"accepted": {RequiredState: engineering.NodeComplete, RequiredEvidence: []engineering.ID{"ev-unit"}}}
	plan2, err := engineering.NewPlanningRevision("plan-2", 2, "plan-1", []string{"build safely"}, artifact("plan-art-2", engineering.ArtifactProjectPlan), provenance("provider", now.Add(time.Minute)))
	if err != nil { t.Fatal(err) }
	proposalContracts := map[engineering.ID]engineering.CompletionContract{
		"accepted": contracts["accepted"],
		"new": {RequiredState: engineering.NodeComplete, RequiredEvidence: []engineering.ID{"ev-new"}},
	}
	proposal := Proposal{
		ID: "proposal-1", ParentPlanRevision: "plan-1", ParentDAGRevision: "dag-1", Planning: plan2,
		DAG: engineering.TaskDAGProposal{RevisionID: "dag-2", Nodes: []engineering.TaskNode{
			{ID: "accepted", ParallelEligible: true, VerificationRequirements: []string{"unit"}},
			{ID: "new", Dependencies: []engineering.ID{"accepted"}, ParallelEligible: true},
		}},
		Contracts: proposalContracts,
		Dispositions: []TaskDisposition{{TaskID: "accepted", Kind: DispositionRetained}, {TaskID: "new", Kind: DispositionNew}},
		Triggers: []Trigger{{Kind: TriggerExplicitHumanRequest, HumanActor: "owner", Provenance: provenance("human-request", now)}},
		Provider: intelligence.ProviderProvenance{Kind: intelligence.ProviderDirectLLM, ProviderID: "direct-test", ObservedAt: now},
		Provenance: provenance("intelligence:direct_llm", now),
	}
	return CurrentState{Plan: engineering.ProjectPlan{ID: "project-plan", AcceptedRevision: plan1}, DAG: old, Contracts: contracts}, proposal
}

func artifact(id engineering.ID, kind engineering.ArtifactKind) engineering.ArtifactRef {
	return engineering.ArtifactRef{ID: id, Kind: kind, URI: "git://tally/" + string(id)}
}
func provenance(source string, at time.Time) engineering.Provenance {
	return engineering.Provenance{Actor: "test", Source: source, At: at}
}
func stateOf(dag engineering.ActiveTaskDAG, id engineering.ID) engineering.TaskState {
	for _, node := range dag.Nodes { if node.ID == id { return node.State } }
	return ""
}
