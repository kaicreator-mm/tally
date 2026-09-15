package replan

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"tally/internal/engineering"
)

func Validate(current CurrentState, proposal Proposal, policy CyclePolicy, ledger CycleLedger) (Decision, error) {
	if err := proposal.ValidateBasic(); err != nil {
		return Decision{}, err
	}
	if current.Plan.AcceptedRevision.ID == "" || current.DAG.RevisionID == "" {
		return Decision{}, fmt.Errorf("%w: current plan and DAG are required", ErrInvalidProposal)
	}
	if proposal.ParentPlanRevision != current.Plan.AcceptedRevision.ID || proposal.ParentDAGRevision != current.DAG.RevisionID {
		return Decision{}, ErrStaleParent
	}
	if proposal.Planning.Number != current.Plan.AcceptedRevision.Number+1 {
		return Decision{}, fmt.Errorf("%w: planning revision must advance exactly by one", ErrStaleParent)
	}

	candidate, err := engineering.ActivateTaskDAG(proposal.DAG)
	if err != nil {
		return Decision{}, fmt.Errorf("%w: proposed DAG: %v", ErrInvalidProposal, err)
	}
	if err := validateContracts(candidate, proposal.Contracts); err != nil {
		return Decision{}, err
	}

	oldByID := make(map[engineering.ID]engineering.TaskNode, len(current.DAG.Nodes))
	for _, node := range current.DAG.Nodes {
		if node.State == engineering.TaskRunning {
			return Decision{}, fmt.Errorf("%w: task %s is running; interrupt/block it before re-plan activation", ErrInvalidProposal, node.ID)
		}
		oldByID[node.ID] = node
	}
	newByID := make(map[engineering.ID]engineering.TaskNode, len(candidate.Nodes))
	for _, node := range candidate.Nodes {
		newByID[node.ID] = node
	}
	dispositions := make(map[engineering.ID]TaskDisposition, len(proposal.Dispositions))
	for _, d := range proposal.Dispositions {
		dispositions[d.TaskID] = d
		if _, oldOK := oldByID[d.TaskID]; !oldOK {
			if _, newOK := newByID[d.TaskID]; !newOK {
				return Decision{}, fmt.Errorf("%w: disposition references unknown task %s", ErrInvalidProposal, d.TaskID)
			}
		}
	}

	acceptedWorkInvalidated := false
	for id, old := range oldByID {
		d, ok := dispositions[id]
		if !ok {
			return Decision{}, fmt.Errorf("%w: old task %s has no disposition", ErrInvalidProposal, id)
		}
		_, existsInNew := newByID[id]
		switch d.Kind {
		case DispositionRetained:
			if !existsInNew {
				return Decision{}, fmt.Errorf("%w: retained task %s is missing from proposed DAG", ErrInvalidProposal, id)
			}
		case DispositionInvalidated, DispositionSuperseded:
			if existsInNew {
				return Decision{}, fmt.Errorf("%w: %s task %s cannot keep the same active node identity", ErrInvalidProposal, d.Kind, id)
			}
			if d.ReplacementID != "" {
				if _, ok := newByID[d.ReplacementID]; !ok {
					return Decision{}, fmt.Errorf("%w: replacement task %s is not in proposed DAG", ErrInvalidProposal, d.ReplacementID)
				}
				if rd, ok := dispositions[d.ReplacementID]; !ok || rd.Kind != DispositionNew {
					return Decision{}, fmt.Errorf("%w: replacement task %s must be marked new", ErrInvalidProposal, d.ReplacementID)
				}
			}
			if old.State == engineering.TaskAccepted || old.State == engineering.TaskDone {
				acceptedWorkInvalidated = true
			}
		case DispositionNew:
			return Decision{}, fmt.Errorf("%w: existing task %s cannot be marked new", ErrInvalidProposal, id)
		default:
			return Decision{}, fmt.Errorf("%w: invalid disposition for %s", ErrInvalidProposal, id)
		}
	}
	for id := range newByID {
		d, ok := dispositions[id]
		if !ok {
			return Decision{}, fmt.Errorf("%w: proposed task %s has no disposition", ErrInvalidProposal, id)
		}
		_, existed := oldByID[id]
		if existed && d.Kind != DispositionRetained {
			return Decision{}, fmt.Errorf("%w: existing proposed task %s must be retained", ErrInvalidProposal, id)
		}
		if !existed && d.Kind != DispositionNew {
			return Decision{}, fmt.Errorf("%w: new proposed task %s must be marked new", ErrInvalidProposal, id)
		}
	}

	// Retained accepted/done work is reusable only when its engineering node and
	// completion contract are unchanged. Re-planning may freely reshape pending
	// work, but it cannot relabel already accepted work as compatible.
	for i := range candidate.Nodes {
		old, existed := oldByID[candidate.Nodes[i].ID]
		if !existed || dispositions[old.ID].Kind != DispositionRetained {
			continue
		}
		if old.State != engineering.TaskAccepted && old.State != engineering.TaskDone {
			continue
		}
		if !compatibleAcceptedNode(old, candidate.Nodes[i]) {
			return Decision{}, fmt.Errorf("%w: retained accepted task %s changed engineering requirements", ErrAcceptedWork, old.ID)
		}
		oldContract, oldOK := current.Contracts[old.ID]
		newContract, newOK := proposal.Contracts[old.ID]
		if !oldOK || !newOK || !reflect.DeepEqual(normalizeContract(oldContract), normalizeContract(newContract)) {
			return Decision{}, fmt.Errorf("%w: retained accepted task %s changed CompletionContract", ErrAcceptedWork, old.ID)
		}
		candidate.Nodes[i].State = old.State
		candidate.Nodes[i].Acceptance = old.Acceptance
		candidate.Nodes[i].BlockedReason = ""
	}
	candidate = candidate.RefreshReadiness()

	impact := proposal.Impact
	if acceptedWorkInvalidated && !impact.AcceptedWorkInvalidated {
		return Decision{}, fmt.Errorf("%w: proposal invalidates accepted work but impact does not declare it", ErrAcceptedWork)
	}

	fingerprint, err := Fingerprint(proposal)
	if err != nil {
		return Decision{}, err
	}
	if err := checkCycle(policy, ledger, fingerprint); err != nil {
		return Decision{}, err
	}

	requiredAuthority := deriveAuthority(proposal.ID, impact)
	return Decision{
		proposalID: proposal.ID, parentPlan: proposal.ParentPlanRevision, parentDAG: proposal.ParentDAGRevision,
		planning: proposal.Planning, dag: candidate, contracts: cloneContracts(proposal.Contracts),
		dispositions: append([]TaskDisposition(nil), proposal.Dispositions...), impact: impact,
		fingerprint: fingerprint, requiredAuthority: requiredAuthority, provider: proposal.Provider, validated: true,
	}, nil
}

func validateContracts(dag engineering.ActiveTaskDAG, contracts map[engineering.ID]engineering.CompletionContract) error {
	if len(contracts) != len(dag.Nodes) {
		return fmt.Errorf("%w: every proposed DAG task requires exactly one CompletionContract", ErrInvalidProposal)
	}
	for _, node := range dag.Nodes {
		if _, ok := contracts[node.ID]; !ok {
			return fmt.Errorf("%w: task %s is missing CompletionContract", ErrInvalidProposal, node.ID)
		}
	}
	return nil
}

func compatibleAcceptedNode(a, b engineering.TaskNode) bool {
	a.State, b.State = "", ""
	a.Acceptance, b.Acceptance = "", ""
	a.BlockedReason, b.BlockedReason = "", ""
	a.Dependencies = sortedIDs(a.Dependencies)
	b.Dependencies = sortedIDs(b.Dependencies)
	a.RuntimeRequirements = sortedStrings(a.RuntimeRequirements)
	b.RuntimeRequirements = sortedStrings(b.RuntimeRequirements)
	a.EnvironmentRequirements = sortedStrings(a.EnvironmentRequirements)
	b.EnvironmentRequirements = sortedStrings(b.EnvironmentRequirements)
	a.VerificationRequirements = sortedStrings(a.VerificationRequirements)
	b.VerificationRequirements = sortedStrings(b.VerificationRequirements)
	return reflect.DeepEqual(a, b)
}

func normalizeContract(c engineering.CompletionContract) engineering.CompletionContract {
	c.RequiredArtifacts = sortedIDs(c.RequiredArtifacts)
	c.RequiredEvidence = sortedIDs(c.RequiredEvidence)
	c.RequiredChecks = sortedIDs(c.RequiredChecks)
	c.RequiredRecords = sortedIDs(c.RequiredRecords)
	c.RequiredAuthority = sortedIDs(c.RequiredAuthority)
	c.RequiredGitOperations = sortedIDs(c.RequiredGitOperations)
	return c
}

func sortedIDs(in []engineering.ID) []engineering.ID {
	out := append([]engineering.ID(nil), in...)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
func sortedStrings(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func deriveAuthority(proposalID engineering.ID, impact Impact) []engineering.AuthorityRequirement {
	if !impact.Material() {
		return nil
	}
	var reasons []string
	if impact.RequirementsChanged { reasons = append(reasons, "requirements changed") }
	if impact.ScopeChanged { reasons = append(reasons, "scope changed") }
	if impact.ArchitectureReplaced { reasons = append(reasons, "architecture replacement") }
	if impact.AcceptedWorkInvalidated { reasons = append(reasons, "accepted work invalidation") }
	if impact.HighCost { reasons = append(reasons, "high cost") }
	if impact.HighRisk { reasons = append(reasons, "high risk") }
	return []engineering.AuthorityRequirement{{
		ID: engineering.ID("replan-authority-" + string(proposalID)),
		Mode: engineering.ModeAuthorityRequired,
		Scope: "replan.activation",
		Reason: strings.Join(reasons, ", "),
	}}
}

func CheckAuthority(decision Decision, decisions []engineering.AuthorityDecision) error {
	if !decision.validated {
		return fmt.Errorf("%w: decision was not produced by validator", ErrInvalidProposal)
	}
	if len(decision.requiredAuthority) == 0 {
		return nil
	}
	byID := map[engineering.ID][]engineering.AuthorityDecision{}
	for _, d := range decisions {
		if err := d.Validate(); err != nil {
			return fmt.Errorf("%w: invalid authority decision: %v", ErrAuthorityRequired, err)
		}
		byID[d.RequirementID] = append(byID[d.RequirementID], d)
	}
	for _, requirement := range decision.requiredAuthority {
		authorized := false
		for _, d := range byID[requirement.ID] {
			if d.Status != engineering.AuthorityAuthorized {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(d.Actor), strings.TrimSpace(decision.provider.ProviderID)) {
				return ErrSelfAuthorization
			}
			authorized = true
		}
		if !authorized {
			return fmt.Errorf("%w: %s", ErrAuthorityRequired, requirement.ID)
		}
	}
	return nil
}

func checkCycle(policy CyclePolicy, ledger CycleLedger, fingerprint string) error {
	if policy.MaxReplans == 0 { policy.MaxReplans = DefaultCyclePolicy().MaxReplans }
	if policy.MaxSameFingerprint == 0 { policy.MaxSameFingerprint = DefaultCyclePolicy().MaxSameFingerprint }
	if ledger.Activations >= policy.MaxReplans {
		return fmt.Errorf("%w: max replans %d reached", ErrCycleBudget, policy.MaxReplans)
	}
	var repeated uint64
	for _, fp := range ledger.Fingerprints {
		if fp == fingerprint { repeated++ }
	}
	if repeated >= policy.MaxSameFingerprint {
		return fmt.Errorf("%w: equivalent proposal repeated %d times", ErrCycleBudget, repeated)
	}
	return nil
}

// Fingerprint detects semantically repeated re-plans. Revision IDs, proposal
// IDs, provider IDs and timestamps are intentionally excluded so minting a new
// identity cannot evade bounded-cycle control.
func Fingerprint(p Proposal) (string, error) {
	dag, err := engineering.ActivateTaskDAG(p.DAG)
	if err != nil {
		return "", fmt.Errorf("%w: fingerprint DAG: %v", ErrInvalidProposal, err)
	}
	nodes := dag.Nodes
	for i := range nodes {
		nodes[i].State = ""
		nodes[i].Acceptance = ""
		nodes[i].BlockedReason = ""
		nodes[i].Dependencies = sortedIDs(nodes[i].Dependencies)
		nodes[i].RuntimeRequirements = sortedStrings(nodes[i].RuntimeRequirements)
		nodes[i].EnvironmentRequirements = sortedStrings(nodes[i].EnvironmentRequirements)
		nodes[i].VerificationRequirements = sortedStrings(nodes[i].VerificationRequirements)
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	dispositions := append([]TaskDisposition(nil), p.Dispositions...)
	sort.Slice(dispositions, func(i, j int) bool { return dispositions[i].TaskID < dispositions[j].TaskID })
	contracts := cloneContracts(p.Contracts)
	for id, c := range contracts { contracts[id] = normalizeContract(c) }
	canonical := struct {
		Strategy []string `json:"strategy"`
		Nodes []engineering.TaskNode `json:"nodes"`
		Contracts map[engineering.ID]engineering.CompletionContract `json:"contracts"`
		Dispositions []TaskDisposition `json:"dispositions"`
		Impact Impact `json:"impact"`
	}{append([]string(nil), p.Planning.Strategy...), nodes, contracts, dispositions, p.Impact}
	raw, err := json.Marshal(canonical)
	if err != nil { return "", fmt.Errorf("replan: fingerprint marshal: %w", err) }
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func NewActivationRecord(current CurrentState, decision Decision, authority []engineering.AuthorityDecision, actor string, at time.Time) (ActivationRecord, error) {
	if !decision.validated {
		return ActivationRecord{}, fmt.Errorf("%w: decision not validated", ErrInvalidProposal)
	}
	if strings.TrimSpace(actor) == "" || at.IsZero() {
		return ActivationRecord{}, errors.New("replan: activation actor/time are required")
	}
	r := ActivationRecord{
		ProposalID: decision.proposalID, Fingerprint: decision.fingerprint,
		PreviousPlanning: current.Plan.AcceptedRevision, PreviousDAG: cloneDAG(current.DAG), PreviousContracts: cloneContracts(current.Contracts),
		ActivatedPlanning: decision.planning, ActivatedDAG: cloneDAG(decision.dag), ActivatedContracts: cloneContracts(decision.contracts),
		Dispositions: append([]TaskDisposition(nil), decision.dispositions...), Impact: decision.impact,
		Authority: append([]engineering.AuthorityDecision(nil), authority...), Actor: actor, ActivatedAt: at.UTC(),
	}
	return r, r.Validate()
}
