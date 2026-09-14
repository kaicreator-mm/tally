# P1-06 — Risk-adjusted acceptance guard

- **Phase:** P1
- **Kind:** implementation
- **Wave:** W12
- **Complexity:** XL
- **Execution model:** HIGH
- **Depends on:** P1-04
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-12)
- Seam adjudication (DECISION_LOCKS open seam owned by this task): **guard enforcement in the Closed Loop Engine accept transition** — `closedloop.Config.AcceptGuard` consulted on `VerdictVerified` (after the P2-06 exposure guard, before `EventVerified`), denial escalates typed `assurance_insufficient`; ServiceVerifier rejected as enforcement point (it would fabricate candidate-failure/ERROR facts from a true `FinalVerified`, and the Router classifies `FinalVerified → unknown → escalate` anyway); one pure semantic evaluator in `taskstate.EvaluateAcceptanceGuard` (required from `contract.Compiled.Clauses()`, achieved via `assurance.Achieved`, VERIFIED_LIMITED gap set identical to the VerifiedBasis projection; no duplicated assurance math). `NewLoop` wires `CompiledAcceptanceGuard` from `LoopConfig.Authority`; `cmd/worker/v14.go` always supplies the mandatory compiled authority, so production acceptance is guard-gated (DL-07/A-03 mechanical).
- AC: Required L3 + achieved L1 never returns Accepted — PASS (taskstate evaluator test; closedloop `TestLoopGuardBlocksRequiredL3WithAchievedL1` through real NewLoop composition; worker `TestV14AcceptanceGuardBlocksInsufficientAssurance` on the production path). Unmatched high-risk signal never silently passes — PASS (evaluator test isolating unmatched with assurance satisfied; worker `TestV14AcceptanceGuardBlocksUnmatchedRisk` surfaces the signal verbatim in the escalation detail).
- Verification: focused adversarial tests incl. VERIFIED_LIMITED semantics — done (waiver distinctness, missing-coverage L0 fail-safe, unknown-visibility no-facts, nil-authority rejection, repaired-candidate re-check, Router bypass/no-budget consumption). Full evidence: `reports/P1-06_risk_adjusted_acceptance_guard.md`. Arch suite, focused packages, full internal suite, DB-backed worker production tests and integration e2e v14 harnesses all PASS; `git diff --check` clean; Decision Locks not bypassed (open-seam row frozen by this owning task).


## Goal

Prevent a base FinalVerified candidate from becoming Accepted when compiled assurance/risk coverage is insufficient.


## Source facts

- VerifiedBasis already represents required/achieved/unmatched semantics; current Closed Loop verifier maps FinalVerified to Verified.


## Allowed / primary paths

- `internal/closedloop/**`
- `internal/taskstate/**`
- `internal/assurance/**`
- `cmd/worker/v14.go`


## Forbidden boundary

- Do not duplicate assurance math in a second implementation.


## Implementation requirements

1. Reuse or extract one pure semantic evaluator.
2. Insufficient assurance/unmatched required risk must fail-safe to non-Accept path.
3. Do not rewrite Closed Loop engine.


## Acceptance criteria

- [ ] Required L3 + achieved L1 never returns Accepted.
- [ ] Unmatched high-risk signal never silently passes.


## Verification

- Focused adversarial tests including VERIFIED_LIMITED semantics.

## Mandatory implementation discipline

- Read `../../DECISION_LOCKS.md`, `../../PRD_AMENDMENTS.md`, and `../../SOURCE_BASELINE.md` first.
- Do not broaden scope to “clean up” adjacent architecture.
- Do not weaken fail-safe behavior to make tests pass.
- Preserve source-commit identity and held-out verification boundaries.
- If an expected seam does not exist, stop this task as `BLOCKED_BY_SOURCE_FACT` and report the exact source evidence instead of inventing a parallel subsystem.
- New behavior needs focused tests.
- Existing baseline failures must be distinguished from introduced failures.

## Completion report

Report exactly:

1. status: `DONE | BLOCKED | PARTIAL`;
2. changed files;
3. semantic changes;
4. tests/commands executed with exact results;
5. baseline failures observed;
6. remaining risks;
7. whether any Decision Lock was touched;
8. commit hash (if committed).

---

# L3 Task-Level Implementation Reference Pack

## Model assignment

- **Execution:** `HIGH`
- **Review:** `HIGH`
- **Reason:** Direct acceptance-safety change: FinalVerified must not bypass required assurance or unmatched risk.
- **Review rule:** HIGH review required because the task touches an authority boundary, release gate, or high-blast-radius projection even when implementation is deterministic.

## Reference order — mandatory

Use evidence in this order. Do not jump directly to an external example before reading Tally's tests/contracts.
1. **Tests:** Tests first: `taskstate/basis_test.go::{TestBasisVerifiedLimitedUnmatchedRisk,TestBasisVerifiedLimitedWaiver}`, `test/arch/v14_assurance_failsafe_test.go`
2. **Contract / Interface:** Interfaces: `assurance.Select` = required; `assurance.Achieved` = achieved; `taskstate.VerifiedBasis.Verdict`
3. **Core Implementation:** Core seam to adapt: `closedloop/verify.go::verdictForStatus`, worker verifier composition; reuse `taskstate` semantic evaluator/extracted pure helper
4. **Failure Handling:** Failure: `FinalVerified` with required>achieved, unmatched risk or valid waiver must never return normal Accepted

## External reference pack

- No external implementation is required. Prefer current Tally source as the canonical implementation reference.

## Do

- Fail before runtime start when authority cannot be reproduced from one source snapshot.
- Start by running the listed focused tests or the closest available subset.
- Make the smallest semantic change that satisfies this task; keep public surface area narrow.
- Add at least one negative/failure-path test for new behavior.

## Don't

- Do not compute policy/risk/assurance from mixed commits or silently downgrade unknowns.
- Do not copy external code wholesale. Extract the pattern and adapt it to Tally's existing interfaces and dependency whitelist.
- Do not add a new dependency unless `test/arch/deps_test.go` and the dependency policy are intentionally updated by an approved architecture change.

## High-model execution requirement

This task should be performed by a high-capability model because success requires semantic judgment rather than mechanical editing. The model must explain the chosen seam, enumerate rejected alternatives, and demonstrate that fail-safe behavior remains intact before coding.

## L3 completion evidence required from implementing agent

- exact source/tests read;
- any external reference actually used;
- why the chosen seam is the narrowest valid seam;
- changed files;
- exact test commands/results;
- negative tests added;
- known uncertainty or version sensitivity;
- confirmation that Decision Locks were not bypassed.
