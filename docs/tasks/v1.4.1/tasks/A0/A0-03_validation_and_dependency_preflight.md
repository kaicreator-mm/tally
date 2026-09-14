# A0-03 — Validation and dependency preflight

- **Phase:** A0
- **Kind:** test-infra
- **Wave:** W01
- **Complexity:** S
- **Execution model:** LOW
- **Depends on:** none
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO


## Goal

Provide a deterministic preflight that distinguishes dependency/network failures from source test failures.


## Allowed / primary paths

- `scripts/**`
- `docs/tasks/v1.4.1/**`


## Implementation requirements

1. Check Go/tool versions and module availability.
2. Run focused baseline when dependencies are available.
3. Emit machine-readable status.


## Acceptance criteria

- [ ] Offline dependency failure is classified, not reported as code regression.


## Verification

- Run in one connected and one deliberately offline environment where possible.

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

- **Execution:** `LOW`
- **Review:** `LOW`
- **Reason:** The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred.
- **Review rule:** LOW review is sufficient if all focused tests and task acceptance criteria pass; escalate only on source/API mismatch.

## Reference order — mandatory

Use evidence in this order. Do not jump directly to an external example before reading Tally's tests/contracts.
1. **Tests:** Tests: `test/arch/deps_test.go` plus focused package tests
2. **Contract / Interface:** Interface: `Makefile` targets `test`, `test-integration`, `check`
3. **Core Implementation:** Core: `go.mod` dependency whitelist comment; `.github/workflows/*`
4. **Failure Handling:** Failure: classify module-fetch/network errors separately from compile/test assertion failures

## External reference pack
- **EXT-UBER-GO-TEST — Uber Go Style Guide — table-driven tests**  
  https://github.com/uber-go/guide/blob/master/style.md  
  License/version: CC BY 3.0 / repository terms; pattern only; master.  
  Use: Reference table-driven test structure for deterministic validation matrices.

## Do

- Prefer source/AST evidence and executable guards over prose assertions.
- Start by running the listed focused tests or the closest available subset.
- Make the smallest semantic change that satisfies this task; keep public surface area narrow.
- Add at least one negative/failure-path test for new behavior.

## Don't

- Do not create implementation APIs while doing architecture/readiness work.
- Do not copy external code wholesale. Extract the pattern and adapt it to Tally's existing interfaces and dependency whitelist.
- Do not add a new dependency unless `test/arch/deps_test.go` and the dependency policy are intentionally updated by an approved architecture change.

## Low-cost model execution recipe

1. Read the exact tests listed above.
2. Read the interface/types used by those tests.
3. Read only the nearest implementation seam, not the whole repository.
4. Implement one bounded change.
5. Run focused tests.
6. Run relevant `test/arch` guards.
7. If expected source facts differ from this pack, stop with `BLOCKED_BY_SOURCE_FACT`; do not redesign the architecture.

## L3 completion evidence required from implementing agent

- exact source/tests read;
- any external reference actually used;
- why the chosen seam is the narrowest valid seam;
- changed files;
- exact test commands/results;
- negative tests added;
- known uncertainty or version sensitivity;
- confirmation that Decision Locks were not bypassed.

---

## Execution status

- **Status: DONE** (executed at HEAD `3c1e983`, 2026-09-12)
- Deliverable: `scripts/preflight/` (Go, stdlib-only, staged deterministic preflight; JSON `tally.preflight.v1` + exit-code contract)
- Report: `docs/tasks/v1.4.1/reports/A0-03_validation_and_dependency_preflight.md`
- Acceptance: offline dependency failure classified (`DEPENDENCY_NETWORK` / `DEPENDENCY_STATE` → `DEPENDENCY_BLOCKED`, `is_code_regression=false`), never reported as code regression — proven by two end-to-end negative tests and connected + deliberately-offline repo runs
- Baseline at HEAD `3c1e983`: full unit suite (`./test/arch/...`, `./internal/...`) and DB-gated integration suite (Postgres 16 on port 5433) all PASS; no baseline failures
