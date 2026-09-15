# Tally v1.5.5 — L3 Implementation Evidence

Status: READY
Authority: `Tally_PRD_v1.5.5_Replan_Engineering_Learning.md`, v1.5.0 shared contracts, `DECISION_LOCKS.md`

## T155-01 Replan domain

### Tests first
- reject trigger without accepted fact/evidence except explicit human request;
- accept all frozen trigger kinds;
- reject missing parent plan/DAG identity;
- reject duplicate/missing task dispositions.

### Contract
Create `internal/replan` values only. Reuse `engineering.ID`, `PlanningRevision`, `TaskDAGProposal`, `ActiveTaskDAG`, `EvidenceRef`, `Provenance`, `AuthorityRequirement` and intelligence provider provenance. Do not create peer copies of these contracts.

### Failure handling
Typed errors with `errors.Is`; malformed proposals fail closed.

## T155-02 Cycle control

### Tests first
- stable equivalent proposal fingerprint;
- materially different proposal changes fingerprint;
- repeated equivalent proposal over configured budget escalates/rejects;
- total replan budget exhaustion rejects activation.

### Implementation
Canonicalize proposal-owned deterministic fields and SHA-256 hash them. Provider prose/observation timestamps must not make an otherwise equivalent engineering proposal unique.

## T155-03 Engineering validation

### Tests first
- retained accepted/done task preserves terminal state;
- retained task with changed incompatible dependencies/requirements rejects;
- accepted/done task silently absent rejects;
- explicit invalidation/supersession is represented and can require authority;
- new/replaced task starts from deterministic readiness;
- DAG cycle/missing dependency rejects through existing `engineering.ActivateTaskDAG`;
- dependency readiness recalculates after preservation.

### Implementation
Use `engineering.ActivateTaskDAG` as structural authority, then copy only explicitly retained compatible terminal state from the current DAG and call `RefreshReadiness`. Never mutate the current DAG while validating.

## T155-04 Intelligence adapter

### Tests first
- Direct LLM and AI Runtime parse the same typed output shape;
- invalid provider JSON/domain values reject;
- provider result cannot activate state;
- disagreement produces escalation evidence rather than selecting a provider silently.

### Implementation
Add enum operations to `internal/intelligence` only; the `Provider` interface remains unchanged. `internal/replan` owns request/output encoding because it is the consumer domain.

## T155-05 Control Plane activation

### Tests first
- initial `AcceptPlan`/`ActivateDAG` remain immutable;
- stale parent plan/DAG rejects;
- validated proposal with no required human authority activates atomically;
- material scope/architecture/accepted-work invalidation without authority rejects;
- authorized activation advances Plan + ActiveDAG together;
- old PlanningRevision/DAG appears exactly once in immutable history;
- activation audit record references proposal/fingerprint.

### Implementation
Extend `ProjectSnapshot` additively with replan history and cycle ledger using JSON `omitempty`. Preserve schema compatibility with current v1.5.0 snapshots. `ActivateReplan` is a separate method guarded by the service mutation mutex and publishes one repository snapshot.

## T155-06/07 Engineering Learning

### Tests first
- rule proposal without supporting outcome/evidence rejects;
- scope/confidence/provenance required;
- duplicate/conflicting evidence is deterministic;
- unresolved conflicting rule cannot be promoted;
- Tally-specific scope is enforced by policy/adaptor, not by a second knowledge store.

### Contract
`EngineeringRuleProposal` wraps governance metadata and an existing canonical `policygovernance.Candidate`/proposal path. Outcome IDs/evidence are references, not copied mutable truth.

## T155-08 Promotion authority

### Tests first
- shared/high-impact rule requires authority;
- missing/rejected authority blocks promotion;
- AI/provider actor cannot authorize its own proposal;
- authorized human promotion produces existing policy-governance activation input without changing runtime policy authority.

### Implementation
Keep existing `policygovernance` lifecycle and Git-backed runtime authority. v1.5.5 adds the authority/evidence gate before activation; it does not modify `engineeringpolicy.Policy` schema.

## T155-09 Integration

Prefer bounded additive endpoints/service methods. No generic workflow/BPM API. API handlers must invoke the same control-plane service path as non-HTTP callers.

## Validation commands

Focused target commands when the repository is available on a Local/Build Host:

```bash
go test ./internal/replan/...
go test ./internal/engineeringlearning/...
go test ./internal/intelligence/...
go test ./internal/controlplane/...
go test ./internal/policygovernance/...
```

Full regression must use the repository's explicit local validation carrier, never GitHub Actions/CI. Hidden fixtures are withheld until visible implementation freezes.

## Review checklist

- no mutation of frozen PRDs;
- no second Project Flow/runtime/provider/rule engine;
- no weakening of v1.5.0 immutability;
- provider output never becomes accepted fact directly;
- retained accepted work has compatibility proof;
- invalidated accepted work always explicit and authority-governed;
- history survives latest-snapshot cleanup;
- no AI self-authority;
- no fabricated validation PASS.