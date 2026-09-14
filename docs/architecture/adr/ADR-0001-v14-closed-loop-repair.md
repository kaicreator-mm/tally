# ADR-0001 — v1.4 Closed-loop Repair and Held-out Feedback Boundary

**Status:** Proposed (A0-04, Architecture Freeze input)
**Date:** 2026-09-08
**Authority inputs:** `Tally_PRD_v1.4_Final.md` §17 Repair + Invariants 2-12; `docs/tasks/v1.4/DECISION_LOCKS.md` #7/#8/#9/#10; `ARCHITECTURE_EVIDENCE.md` §8 INV-V14-03..08, §9 SPIKE-3; A0-02 findings; A0-03 findings.
**Source anchors:** `internal/workflow/v13.go` (inner loop), `cmd/worker/v13.go` (composition, no outer loop), `internal/changeset` supersession fact `reason=rework` (migration 0019), `internal/verification` FAIL/ERROR surface, `internal/agentruntime` `ExecutionPolicy.Retry`.

## Context

v1.3 executes Codegen (bounded inner iterations) then Verification once, sequentially, with no path from a verification failure back to a bounded rework attempt. A human currently plays that retry orchestrator. v1.4 adds the outer closed loop (PRD §17). The dangerous failure modes are documented in `ARCHITECTURE_EVIDENCE.md` §5: repair mutating a verified candidate, raw held-out diagnostics leaking to the implementation authority, inner/outer budget conflation, and unbounded repair.

## Decision

### 1. Inner execution retry ≠ outer Repair (AC1)

Two distinct budgets, never shared, never converted:

| | Inner execution loop | Outer Repair loop |
|---|---|---|
| Owner | `workflow.TaskExecution` (v13.go) | new v1.4 closed-loop orchestrator (P2-02/P2-08) |
| Budget | `ExecutionPolicy.Retry.MaxIterations` | `RepairPolicy.MaxAttempts` |
| Scope | same CodegenRun: agent plan/code/test iteration on runtime/codegen failures | verification-driven rework: new CodegenRun per attempt |
| Trigger | agent/runtime process failure | typed verification failure classified `repairable` |
| Termination | existing v13 semantics | hard stop at budget → escalate |

`ExecutionPolicy.Retry.MaxIterations` is **never** read by the repair loop and `MaxRepairAttempts` never extends inner iterations. A task may exhaust inner iterations and still have full repair budget; conversely repair budget is not refillable by inner failures.

### 2. Repair candidate identity (AC2)

Every outer Repair attempt creates:

```text
new CodegenRun (attempt-numbered, parent-linked)
+ new disposable worktree (INV-V14-15)
+ new immutable ChangeSet
+ supersession fact (reason=rework, append-only; existing 0019 surface)
+ fresh independent Verification
```

Never mutated in place: a failed or verified candidate is immutable; a verdict binds one exact ChangeSet (INV-V14-06). Repair base = latest non-superseded candidate's ChangeSet content and its workspace base revision; the new run re-applies intent on top of that base under a fresh contract compilation. Prior candidates remain queryable history; supersession is append-only (INV-V14-06/16). Attempt identity is idempotent per (task, attempt_no); a crashed repair attempt resumes by idempotency key, never by reusing the dead attempt's ChangeSet.

### 3. Failure classification and routing

Typed classification (no `strings.Contains` domain routing). Minimum closed set:

```text
repairable                → outer repair (consumes RepairPolicy budget)
contract_defect           → escalate (acceptance/contract needs human re-authoring)
policy_conflict           → escalate (policy contradiction, not code)
environment_failure       → infrastructure route (retryable infra path, NOT repair budget)
provider_failure          → provider route (AI Runtime/provider path, NOT repair budget)
verification_inconclusive → ERROR semantics: infra re-verify path, never counted as candidate FAIL
requires_human_judgment   → attention/escalate
unknown                   → escalate (fail safe; never silently weakest assurance, INV-V14-09)
```

Verification **FAIL** (verifier produced a trustworthy negative verdict) and **ERROR** (verifier could not produce a trustworthy verdict: timeout, network, runner crash, DB unavailable) remain distinct end-to-end: ERROR never becomes a candidate failure fact, never consumes repair budget, and routes to the infrastructure re-verify path with its own bounded retry. Only FAIL classified `repairable` enters the repair loop. Cancellation is a first-class terminal outcome at every stage: cancelled runs produce no verdict and consume no repair attempt; partial work is discarded with the disposable worktree.

### 4. Safe RepairFeedback for held-out failures (AC3)

Repair context is built through a structured, allowlisted `RepairFeedback` projection. Allowlist (fixed; additive changes require ADR amendment):

```text
acceptance_clause id + semantic text (already agent-visible pre-execution)
failure category (typed classification)
affected behavior description (human/authority-authored phrasing)
safe diagnostic summary (authority-redacted, no raw outputs)
visible evidence refs (already implementation-visible evidence only)
attempt lineage (run/changeset ids)
```

Structurally excluded (never present in the projection type): protected test path/source, hidden test content, exact assertions, expected values, fixture internals, oracle implementation, raw held-out stdout/stderr, content digests of hidden assets, verdict details of held-out bindings beyond pass/fail category. The projection composes the existing `RedactedHoldoutOutcome` allowlist discipline (`internal/verification/holdout_executor.go`) — the repair builder may not widen it.

### 5. Exposure accounting: downgrade/escalate (AC4)

If any held-out material beyond the allowlist is exposed to the implementation authority during repair (misconfiguration, future feature, human decision):

```text
exposure fact appended (evidence.Exposure, append-only, INV-V14-08)
effective assurance for that acceptance: can only stay or decrease — never increase
strict information-independence claims for the affected binding are void
if the acceptance still requires strict independence → escalate (cannot auto-accept)
```

Exposure is monotonic across the repair lineage (all ancestor ChangeSets, claim A28): the worst exposure in the lineage bounds the assurance of every descendant. Assurance never "recovers" by later non-exposure.

### 6. Repair budget default (AC5)

`RepairPolicy.MaxAttempts` default = **2**. This is a policy default (configurable per policy revision with rationale), **not** an architecture constant. The invariants that ARE architectural: the loop is finite, the budget is a separate type from inner retry, exhaustion escalates to human attention rather than looping or auto-accepting.

## Alternatives considered

- **Reuse inner iteration budget for repair** — rejected (A12): conflates codegen/runtime failure semantics with acceptance failure semantics; the inner loop ends at ChangeSet creation, verification happens outside it.
- **Mutate the failed candidate in place** — rejected: breaks verdict↔ChangeSet binding and append-only supersession.
- **Feed raw held-out failure output to the repair agent** — rejected (A16): destroys information independence by construction; strict assurance becomes false.
- **Unbounded repair until pass** — rejected: verification-fix cycles can oscillate; bounded retry with escalation is the established pattern.

## Consequences

- P2-02 (closed-loop skeleton), P2-03 (classification), P2-04 (RepairFeedback), P2-05 (continuation/supersession), P2-06 (exposure accounting), P2-07 (RepairPolicy) each implement exactly one section above.
- `workflow/v13.go` inner loop is retained unchanged (PATH_OWNERSHIP: P2 wraps, never rewrites).
- A0-03's conservative assurance default (L1 floor + L3 uplift by risk) applies to re-verification identically: a repair attempt is verified under the same frozen acceptance and assurance mapping as the original — never weaker (INV-V14-03).
