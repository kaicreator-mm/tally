package controlplane

import (
	"context"
	"fmt"
	"strings"

	"tally/internal/engineering"
	"tally/internal/replan"
)

// ActivateReplan validates and activates a planning/DAG revision using the
// bounded v1.5.5 default policy. It deliberately does not relax AcceptPlan or
// the initial ActivateDAG immutability guaranteed by v1.5.0.
func (s *Service) ActivateReplan(ctx context.Context, projectID engineering.ID, proposal replan.Proposal, authority []engineering.AuthorityDecision) (ProjectSnapshot, error) {
	return s.ActivateReplanWithPolicy(ctx, projectID, proposal, authority, replan.DefaultCyclePolicy())
}

// ActivateReplanWithPolicy exists for explicit project policy configuration.
// The policy is still deterministic input; intelligence providers never choose
// or mutate it.
func (s *Service) ActivateReplanWithPolicy(ctx context.Context, projectID engineering.ID, proposal replan.Proposal, authority []engineering.AuthorityDecision, policy replan.CyclePolicy) (ProjectSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	snap, err := s.Repo.Load(ctx, projectID)
	if err != nil {
		return ProjectSnapshot{}, err
	}
	if snap.Plan == nil || snap.ActiveDAG == nil || snap.Architecture == nil || snap.Requirements == nil {
		return ProjectSnapshot{}, fmt.Errorf("%w: active planning baseline", ErrStageOrder)
	}
	current := replan.CurrentState{Plan: *snap.Plan, DAG: *snap.ActiveDAG, Contracts: snap.Contracts}
	ledger := replan.CycleLedger{Activations: uint64(len(snap.ReplanHistory))}
	ledger.Fingerprints = make([]string, 0, len(snap.ReplanHistory))
	for _, record := range snap.ReplanHistory {
		ledger.Fingerprints = append(ledger.Fingerprints, record.Fingerprint)
	}
	decision, err := replan.Validate(current, proposal, policy, ledger)
	if err != nil {
		return ProjectSnapshot{}, err
	}
	if err := replan.CheckAuthority(decision, authority); err != nil {
		return ProjectSnapshot{}, err
	}
	actor := strings.TrimSpace(s.Actor)
	if actor == "" {
		actor = "tally-controlplane"
	}
	at := s.now()
	record, err := replan.NewActivationRecord(current, decision, authority, actor, at)
	if err != nil {
		return ProjectSnapshot{}, err
	}

	// Build the entire next snapshot before publishing it. Repository.Save is
	// the single publication point, so Plan and ActiveTaskDAG never expose a
	// mixed revision.
	plan := *snap.Plan
	plan.AcceptedRevision = decision.Planning()
	dag := decision.DAG()
	snap.ReplanHistory = append(append([]replan.ActivationRecord(nil), snap.ReplanHistory...), record)
	snap.Plan = &plan
	snap.ActiveDAG = &dag
	snap.Contracts = decision.Contracts()
	if err := s.record(&snap, "replan.activated", "", "replan:"+decision.Fingerprint()); err != nil {
		return ProjectSnapshot{}, err
	}
	return snap, s.save(ctx, &snap)
}
