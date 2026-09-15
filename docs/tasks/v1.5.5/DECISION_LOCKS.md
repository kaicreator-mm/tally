# Tally v1.5.5 Decision Locks

Status: ACTIVE
Authority: frozen v1.5 PRD + frozen v1.5.5 PRD + v1.5.0 Decision Locks

## DL-155-01 Additive slice
v1.5.5 extends v1.5.0 behavior without redefining the frozen shared contracts.

## DL-155-02 Initial lifecycle remains immutable
`AcceptPlan` and initial `ActivateDAG` retain their v1.5.0 one-time semantics. Re-plan uses a separate governed activation operation.

## DL-155-03 Single Project Flow
`ActiveTaskDAG` remains the only active Project Flow authority. ReplanProposal/history are proposals/history, never peer mutable flow state.

## DL-155-04 Proposal is not activation
Direct LLM/AI Runtime can only produce proposal/assessment/recommendation. Engineering validation and authority gates precede activation.

## DL-155-05 Accepted work preservation
Existing accepted/done work is retained when compatible. Any accepted/done task not retained requires an explicit `invalidated` or `superseded` disposition and impact/authority evaluation.

## DL-155-06 Revision ancestry
Every activated PlanningRevision has a non-zero monotonic number and parent equal to the currently accepted PlanningRevision. Every activated TaskDAG revision is tied to that activation. Stale-parent proposals fail closed.

## DL-155-07 History is append-only
Previous active PlanningRevision/TaskDAG baselines and activation records are retained. Current compatibility snapshot file generations are not revision history.

## DL-155-08 Deterministic re-plan validation
DAG structure, accepted-work accounting, disposition legality, dependency recalculation, revision ancestry, cycle-control fingerprint, authority requirements and activation readiness are deterministic code.

## DL-155-09 Bounded cycles
Repeated equivalent proposals are fingerprinted. Policy bounds re-plan count and repeated fingerprints; exhaustion escalates rather than looping indefinitely.

## DL-155-10 Trigger evidence
Re-plan triggers come from accepted facts/events/evidence. LLM periodic speculation alone is not a valid trigger. Explicit human request is an accepted trigger with actor/provenance.

## DL-155-11 Authority
Requirements/scope material change, architecture replacement, accepted-work invalidation, high-cost/high-risk change and shared Engineering Rule promotion require an Authority gate according to policy.

## DL-155-12 Intelligence substitution
Direct LLM and AI Runtime use the same `intelligence.Provider` interface. New operations are enum additions only; no second provider abstraction.

## DL-155-13 Learning reuse
Engineering Learning consumes existing Outcome/evidence and delegates policy candidate governance to existing `policygovernance`; it does not create a second runtime rule authority.

## DL-155-14 No AI self-promotion
AI/provider identities may propose and analyze rules but cannot satisfy the human/shared-rule promotion authority they caused.

## DL-155-15 Optional neighboring slices
Core v1.5.5 must compile conceptually against v1.5.0 only; v1.5.1–v1.5.4 capabilities integrate through contracts if present and are not prerequisites.

## DL-155-16 Validation carrier
No GitHub Actions or CI. Focused tests, integration, full regression, Hidden Validation, Critical Journeys and Release Qualification use explicit Local/Build Host execution with `PASS | FAIL | NOT_RUN | ENV-BLOCKED` evidence.