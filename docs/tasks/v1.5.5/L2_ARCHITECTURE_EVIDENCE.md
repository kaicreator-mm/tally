# Tally v1.5.5 — L2 Architecture Evidence

Status: ACCEPTED FOR IMPLEMENTATION
Base revision: `d0c0f72390fb04d5891cfc82937f977729488f25`
Authority: frozen v1.5 PRD + frozen v1.5.5 executable PRD + v1.5.0 shared contracts

## 1. Compatibility conclusion

v1.5.5 is compatible with the frozen v1.5.0 architecture if it is implemented as an additive governed capability. It must not relax or redefine the v1.5.0 shared contracts.

The apparent conflict that `AcceptPlan` and `ActivateDAG` reject a second accepted value is intentional v1.5.0 immutability. v1.5.5 therefore adds a separate validated re-plan activation path. The initial lifecycle APIs remain immutable and unchanged.

## 2. Architectural invariants

1. `ActiveTaskDAG` remains the sole Project Flow authority.
2. A `ReplanProposal` is never active state.
3. Direct LLM and AI Runtime use the existing `intelligence.Provider` contract; provider output is proposal/assessment/recommendation only.
4. Engineering validation is deterministic for DAG structure, revision ancestry, task disposition, accepted-work preservation, authority requirements, and re-plan cycle policy.
5. Re-plan activation atomically replaces only the active PlanningRevision/TaskDAG view after validation and authority.
6. Previous PlanningRevision/TaskDAG baselines remain in append-only history carried by project state; filesystem snapshot generations are not treated as revision history.
7. Existing `outcome` and `policygovernance` remain the fact/governance seams for learning. v1.5.5 does not create a second rule engine.
8. v1.5.1 persistence internals, v1.5.2 SSH/Docker, v1.5.3 Git automation, and v1.5.4 Multi-Agent are optional adapters, not core dependencies.

## 3. Re-plan domain

Add `internal/replan` with:

- typed trigger kinds backed by accepted facts/evidence;
- `ReplanProposal` containing parent planning/DAG revision identity, proposed PlanningRevision, proposed TaskDAG, task dispositions, impact declaration, evidence/provenance, provider provenance, and fingerprint;
- task disposition enum `retained | superseded | invalidated | new`;
- deterministic validator that builds the candidate DAG, preserves compatible accepted/done work, rejects silent accepted-work loss, recalculates dependencies/readiness, and produces an activation decision;
- bounded cycle policy using proposal fingerprint/revision ancestry/repeat count;
- authority requirement derivation for requirements/scope change, architecture replacement, accepted-work invalidation and high-impact changes.

## 4. Control-plane integration

`controlplane.Service` receives an additive `ActivateReplan` method. It does not change `AcceptPlan` or initial `ActivateDAG`.

Activation is allowed only when:

- current snapshot identities match proposal parent identities;
- validator returned an accepted engineering decision;
- required human authority is authorized;
- new PlanningRevision number/parent are valid;
- all active DAG nodes have CompletionContracts;
- accepted/done work is retained or explicitly invalidated/superseded;
- cycle policy is satisfied.

On success, the old plan/DAG is appended to history, then the active Plan and ActiveDAG pointers are advanced together in one repository save.

## 5. Intelligence integration

Extend the existing operation enum additively with `replan_proposal` and `engineering_rule_proposal`. The `Provider` interface is unchanged. A parser/adapter converts provider JSON to the typed proposal; invalid JSON/domain content fails closed.

Provider disagreement is represented as evidence/assessment and cannot auto-activate a material plan revision.

## 6. Engineering Learning

Add `internal/engineeringlearning` as a thin governance layer over existing Outcome + policy governance concepts:

- `EngineeringRuleProposal` cites supporting outcome/evidence references, applicable scope, confidence, conflicts and provenance;
- validation rejects missing evidence, empty scope, unsupported confidence or unresolved conflicts;
- promotion policy derives Authority requirements for shared/high-impact rules;
- promotion cannot be authorized by the same AI/provider identity that proposed the rule;
- the adapter produces/uses the existing `policygovernance.Proposal`; runtime policy authority remains the pinned Git policy snapshot.

This is Tally-specific engineering learning, not Cairn/general knowledge governance.

## 7. Persistence

v1.5.0 `FileRepository` keeps only the newest compatibility snapshot after publication. Therefore re-plan history must live inside `ProjectSnapshot` (or an equivalent append-only ledger referenced by it), not in old snapshot files.

The v1.5.5 change stays DB-neutral and does not implement v1.5.1 storage migration.

## 8. Failure model

Fail closed on stale parent revision, malformed provider proposal, DAG cycle/missing dependency, unaccounted accepted work, invalid disposition, missing CompletionContract, missing required authority, repeated proposal fingerprint beyond policy, provider disagreement requiring escalation, evidence-free rule proposal, unresolved rule conflict, and unauthorized/self-authorized promotion.

## 9. Validation strategy

Focused deterministic tests cover trigger validation, proposal validation, preserved accepted work, explicit invalidation, dependency recalculation, stale revision rejection, cycle control, Direct/AI provider substitution, provider disagreement, scope/architecture authority, rule evidence/conflict/authority, self-promotion rejection and history immutability.

Full repository regression, real integration/E2E, held-out Hidden Validation and Critical Journeys are explicit Local/Build Host gates. No GitHub Actions/CI is used.