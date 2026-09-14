# P0-02 — FrozenIntent Git artifact loader

- **Phase:** P0
- **Kind:** implementation
- **Wave:** W04
- **Complexity:** M
- **Execution model:** LOW
- **Depends on:** P0-01
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO


## Goal

Load FrozenIntent from the exact pinned Git commit using the same artifact-authority pattern as existing v1.4 artifacts.


## Allowed / primary paths

- `internal/artifacts/**`
- `internal/intentevidence/**`


## Implementation requirements

1. Do not read working tree as runtime authority.
2. Reject malformed or identity-mismatched artifacts.


## Acceptance criteria

- [ ] Given source commit, loader returns deterministic FrozenIntent or typed rejection.


## Verification

- Commit-tree fixture tests incl. working-tree divergence.

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
1. **Tests:** Tests first: `internal/artifacts/v13_loader_test.go::TestLoadReadsGitSnapshotNotWorkingTree`, `v14_policy_loader_test.go`
2. **Contract / Interface:** Interface: `artifacts.Load(repoPath, rev)`, `LoadPolicies(repoPath, rev)`
3. **Core Implementation:** Core pattern: `internal/artifacts/v13_loader.go`, `v14_policy_loader.go`
4. **Failure Handling:** Failure: unresolved revision, duplicate identity, malformed blob, working-tree divergence

## External reference pack

- No external implementation is required. Prefer current Tally source as the canonical implementation reference.

## Do

- Keep presentation, intent evidence and frozen engineering artifacts explicitly separated.
- Start by running the listed focused tests or the closest available subset.
- Make the smallest semantic change that satisfies this task; keep public surface area narrow.
- Add at least one negative/failure-path test for new behavior.

## Don't

- Do not let LLM-generated UI/risk text become authority without deterministic validation/freeze.
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

- **Status: DONE** (2026-09-12)
- Report: `docs/tasks/v1.4.1/reports/P0-02_frozenintent_git_artifact_loader.md`
- Implemented: `internal/artifacts/v141_intent_loader.go` — `LoadIntent(repoPath, rev, taskID)` reads `.tally/intents/<task_id>.yaml` from the resolved pinned commit's tree only (git object store; working tree invisible), parses via `intentevidence.Parse` (path derived solely from `intentevidence.ArtifactPath`; revision recomputed from the Git blob's actual content) and returns `IntentArtifact{Path, SourceCommit, Intent}` with the **observed** commit sha. Typed rejections: `ErrIntentNotFound` / `ErrIntentInvalid` (intentevidence `ErrInvalidEvidence`/`ErrUnsupportedTaskType` wrapped through for two-level `errors.Is`) / `ErrIntentIdentityMismatch` (content `task_id` ≠ requested/path identity, fail-closed). Infrastructure failures (repo open, revision resolve, object read) stay plain errors, matching the `Load`/`LoadPolicies` taxonomy.
- Requirement 1 (no working-tree authority) and requirement 2 (reject malformed or identity-mismatched artifacts) are both test-pinned; no existing loader file or test was modified; no new dependency.
- Validation: `go test -count=1 -run TestLoadIntent -v ./internal/artifacts/` PASS (6 top-level / 7 subtests, incl. working-tree divergence, 7 malformed-content cases, identity mismatch, untracked-invisible, historical re-load); `go test -count=1 ./internal/artifacts/ ./internal/intentevidence/ ./internal/taskcontract ./internal/engineeringpolicy` PASS; `go test ./test/arch/...` PASS; `go build ./...` + `go vet ./...` PASS; `go run ./scripts/preflight` PASS — exit 0 `BASELINE_OK` (`integration` SKIP: NOT_RUN(reason: opt-in DB-gated, not requested)); `git diff --check` clean.
- Baseline failures: none.
- Decision Locks touched: none (DL-04 reinforced: single pinned source commit, observed-vs-requested identity distinguished; DL-03 untouched — read-only loader, no Git-write authority; DL-17 respected).
- Production reachability: N/A(owned integration tasks pending) — P0-08 (confirm→freeze transaction, reload-after-commit) and P0-09 (e2e gate) are the production consumers; no composition-root behavior changed here.
