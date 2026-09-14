# P1-07 — Compiled basis in task/read projections

- **Phase:** P1
- **Kind:** implementation
- **Wave:** W13
- **Complexity:** M
- **Execution model:** LOW
- **Depends on:** P1-04, P1-06
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-12)
- The VerifiedBasis projection and the `compiled:<hash>` basis reference already existed in `internal/taskstate` (audited v1.4 baseline, commit `22d9996`, old v1.4 numbering); taskstate is zero-changed. The landed change is the production read path only: the Change Detail read model (`v14_change.go`) now loads the compiled authority from the same pinned HEAD commit via the worker's P1-03 seam (`artifacts.LoadBundle → CompileAuthority`, no compiler bypass, DL-04), feeds `Snapshot.Compiled` plus the existing revision-bound functional/independence coverage reports (`deriveV13Coverage`, §8.8.2) into `taskstate.Derive`, and projects the verdict (`VERIFIED` vs `VERIFIED_LIMITED` distinctly rendered — limited names its gap classes and is never shown as plain VERIFIED — plus `NOT_VERIFIED`), the `compiled:<hash>` basis reference (INV-STATE-02), assurance required/achieved facts (achieved at the weakest clause) and unmatched/unknown risk signals verbatim (INV-V14-09).
- Requirement 2 held both directions: when the compiled authority is absent — repo without a FrozenIntent artifact, typed `LoadBundle`/`CompileAuthority` failure (`ErrPolicyRejections` on an invalid policy artifact), or unattributable candidate — the page keeps the exact v1.4 not-wired dimension notes, adds the typed failure reason, and renders no verdict, no level facts and no compiled reference (never fabricates assurance or basis).
- Read-path source-fact fix inside the owned file: the Change Detail path never filled `env.project` (v13env leaves it zero by contract; v14TaskPage always filled it), so task-scoped run/candidate/verification facts never reached the derive and state was pinned at READY; filled with the same one-line pattern as the task page.
- AC: UI/read result distinguishes VERIFIED vs VERIFIED_LIMITED and lists basis — PASS (`TestV14ChangeCompiledBasisPlainVerified`: plain VERIFIED + `compiled:sha256:` + `必需 L1 · 达成 L1`, never LIMITED; `TestV14ChangeCompiledBasisVerifiedLimitedOnUnmatchedRisk`: VERIFIED_LIMITED + signal verbatim + `必需 L3 · 达成 L1`). Negative/failure-path: `TestV14ChangeCompiledAbsentDoesNotClaimAssurance` and `TestV14ChangeInvalidPolicyKeepsCompiledAbsent` (no verdict/levels/compiled references rendered when Compiled is absent).
- Verification: `go build ./...`, `go vet ./...` PASS; touched files gofmt-clean; `go test ./internal/taskstate/...`, `./test/arch/...`, full `./internal/web/handlers/`, full `./internal/...` (DB at `postgresql://postgres:postgres@localhost:15432/postgres`) and `-tags=integration ./test/e2e/...` all PASS; `git diff --check` clean; `make check` NOT_RUN (no make binary in Git Bash). Full evidence: `reports/P1-07_compiled_basis_in_task_read_projections.md`. Decision Locks not bypassed; `hidden/` unread (DL-17). Known boundary: `v14_task.go` (outside Allowed Paths) intentionally untouched — the task-detail page's basis line does not yet carry `compiled:<hash>`.

## Goal

Show actual compiled policy/risk/required-vs-achieved assurance basis in read models/UI.


## Allowed / primary paths

- `internal/taskstate/**`
- `internal/web/handlers/v14_change.go`
- `web/templates/**`


## Implementation requirements

1. Projection only; no new mutable state.
2. Do not claim assurance when Compiled is absent.


## Acceptance criteria

- [ ] UI/read result distinguishes VERIFIED vs VERIFIED_LIMITED and lists basis.


## Verification

- Projection/handler tests.

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
1. **Tests:** Tests: `taskstate/basis_test.go`, `v14_task_test.go::TestV14TaskDetailShowsRepairChainBasisAndUncertainty`, `v14_change_test.go`
2. **Contract / Interface:** Interface: `VerifiedBasis`, `Compiled.Coverage`, projections only
3. **Core Implementation:** Core: existing v1.4 task/change handlers and templates
4. **Failure Handling:** Failure: absent Compiled must render unknown/not available, never fabricate normal VERIFIED

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
