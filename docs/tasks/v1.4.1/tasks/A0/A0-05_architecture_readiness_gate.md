# A0-05 — Architecture readiness gate

- **Phase:** A0
- **Kind:** gate
- **Wave:** W02
- **Complexity:** S
- **Execution model:** LOW
- **Depends on:** A0-02, A0-03, A0-04
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO


## Goal

Issue the pre-implementation verdict and freeze the brownfield baseline.


## Allowed / primary paths

- `docs/tasks/v1.4.1/reports/**`


## Implementation requirements

1. Implement only the minimum change required by this task and preserve all v1.4 invariants.


## Acceptance criteria

- [ ] Report says READY_TO_IMPLEMENT or BLOCKED with exact blocker.
- [ ] Baseline commit and unresolved L3 items recorded.


## Verification

- Human/strong-model review of A0 evidence.

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
- **Review:** `HIGH`
- **Reason:** The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred.
- **Review rule:** HIGH review required because the task touches an authority boundary, release gate, or high-blast-radius projection even when implementation is deterministic.

## Reference order — mandatory

Use evidence in this order. Do not jump directly to an external example before reading Tally's tests/contracts.
1. **Tests:** Evidence inputs: outputs from A0-01..04
2. **Contract / Interface:** Tests: `make test`/focused arch suite result
3. **Core Implementation:** Contract: `DECISION_LOCKS.md`
4. **Failure Handling:** Failure: gate must emit exact blocker IDs; no vague READY verdict

## External reference pack

- No external implementation is required. Prefer current Tally source as the canonical implementation reference.

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

- **Status: DONE** (gate issued at HEAD `6946bb5`, clean tree, 2026-09-12)
- Verdict: **READY_TO_IMPLEMENT** — no blocker (exact blocker IDs: none)
- Report: `docs/tasks/v1.4.1/reports/A0-05_architecture_readiness_gate.md`
- Evidence reviewed: A0-01..A0-04 reports and task-file addenda (DAG prerequisites A0-02/A0-03/A0-04 all DONE); frozen `DECISION_LOCKS.md` content intact; `PRD_AMENDMENTS.md` and `SOURCE_BASELINE.md` read first per discipline
- Gate-time validation: `go test -count=1 ./test/arch/...` PASS (`ok tally/test/arch 1.055s`, uncached); `go run ./scripts/preflight` PASS — exit 0 `BASELINE_OK` (toolchain/modules/build/vet/unit_arch/unit_internal all PASS; `-integration` opt-in not requested at gate time, carried forward from A0-03's DB-gated proof at source-identical `3c1e983` — verified via `git diff --stat 3c1e983..HEAD`: docs + `scripts/preflight` + `test/arch` guards only); spot re-checks: `runtimes/` = codex+fake only (no dsh), `contract.Bind(` production call sites = 0 (only `internal/taskstate/basis_test.go`), 57/57 task files reference `DECISION_LOCKS`
- Baseline recorded: audited L2 archive baseline `979a55ad3ba5e514bb80f0e012aa36e0e6577eba` unchanged; gate-time HEAD `6946bb5`; broad parallel implementation starts at/after the A0-05 docs commit
- Unresolved L3 items recorded (all non-blocking): 5 SPIKE_REQUIRED open seams with owning tasks (P0-01, P1-06, P4-03, P3-04, P5-01); review-enforced remainder (A0-04 §2: wiring-level DL-02 facts, BI-01 semantic face, BI-02 full generality; unguarded locks DL-06/10/11/13/19); SOURCE_BASELINE.md stale `BASELINE_NOT_EXECUTED_NETWORK_BLOCKED` preflight note superseded by A0-03 `BASELINE_OK` (doc outside this task's Allowed Paths — recorded, not edited); A0-01 task file lacks an `## Execution status` addendum (same constraint); environment sensitivities (preflight classifier Go-version sensitivity, live-LLM e2e skips without `TALLY_E2E_CODEX_AUTH_FILE`, pre-existing gofmt drift in some `test/arch` files)
- Changed files: `docs/tasks/v1.4.1/reports/A0-05_architecture_readiness_gate.md` (new), this task file (addendum), `docs/tasks/v1.4.1/STATUS.md` (4/57 → 5/57; hard blocker "A0 baseline/readiness gate" resolved by this READY verdict)
- Decision Locks touched: none; nothing under `docs/tasks/v1.4.1/hidden/` read (DL-17)
