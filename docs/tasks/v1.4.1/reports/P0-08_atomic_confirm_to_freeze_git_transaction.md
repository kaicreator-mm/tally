# P0-08 — Atomic confirm-to-freeze Git transaction: Execution Report

- **TASK:** P0-08
- **STATUS:** DONE
- **Date:** 2026-09-12
- **Phase / Wave:** P0 / W06
- **Execution/Review model:** HIGH / HIGH (per task pack; HIGH review still required on this report)
- **Depends on:** P0-02, P0-04, P0-05, P0-06, P0-07 — all satisfied
- **Hidden material:** none read; `docs/tasks/v1.4.1/hidden/` untouched (DL-17)
- **Authority chain read in order:** task contract → `DECISION_LOCKS.md` (DL-03, DL-04, DL-06, DL-17, DL-18) → `PRD_AMENDMENTS.md` → `SOURCE_BASELINE.md` → `PATH_OWNERSHIP.md` → current local source → L3 reference pack (reference order honored: tests → interfaces → core implementation)

## 1. Deliverable

User confirmation is now turned into immutable Git-backed FrozenIntent + TaskContract + VerificationContract at one expected base:

```text
POST /v14/launch/confirm (v14_launch.go, PRG)
  → v14DraftFromForm (typed draft + freeze inputs: task_id / profile / risk_hint / per-clause evidence)
  → v14FreezeInputsOf → ConfirmedDraft + VerificationPlan
  → intakefreeze.Freeze (P0-07 admission, unchanged)
  → artifacts.FreezeCommit (new, single expected-base CAS boundary)
      → gitwriter.Writer.Commit — exactly ONE commit:
         .tally/intents/<task_id>.yaml
         .tally/tasks/<task_id>.yaml
         .tally/verifications/<task_id>.yaml
  → 303 flash outcome: frozen (sha) | idempotent no-op | base conflict | typed failure
  → reloadable: artifacts.Load / artifacts.LoadIntent from the pinned commit (proven by tests)
```

No composition-root change was needed: the confirm route is already registered in production (`RegisterV14Routes` → `server.go` `RegisterRoutes` → `web.NewRouter` in `cmd/server`), and the handler obtains the artifact repo + commit identity from the already-injected `Server.Content` (`RepoPath`/`Identity`), the same source every v13/v14 read path uses. `cmd/server/main.go` and `internal/web/handlers/server.go` are zero-touched.

## 2. Semantics

- **One expected-base CAS boundary (requirement 1):** `FreezeCommit` performs exactly one `gitwriter.Commit` with an explicit `ExpectedBase`. `internal/gitwriter` validates the base under its per-repo mutex + cross-process lock (DL-03 discipline; no second Git-write path anywhere in this task).
- **Same task/source commit semantics (requirement 2):** the three file paths derive from the single task identity; `FreezeCommit` re-checks intent/task/verification identity equality with typed `ErrFreezeIdentityMismatch` before any I/O (fail-closed even for hand-assembled `FrozenArtifacts`).
- **Failure leaves no partial durable artifact set (requirement 3):** all three files render in memory before the repo is touched; any render/admission failure writes nothing. `gitwriter.Commit` applies files atomically and rolls back work tree + index on every failure path; the partial-failure test proves a mid-apply failure leaves zero of the three files and no commit.
- **Stale HEAD conflicts, never silently rebases (AC1):** typed `gitwriter.ErrBaseRevision` passthrough; the handler surfaces it as an explicit conflict flash telling the owner to re-confirm on the latest state. Proven at the transaction seam with an explicit stale base (the handler always reads fresh HEAD, so the race window cannot be injected losslessly over HTTP).
- **Duplicate confirm idempotency:** identical content at HEAD → `gitwriter.ErrNoChanges` → rendered as a non-failing "already frozen, no new commit" outcome; no second commit is created (test-pinned).
- **PRG-safe handler behavior (requirement 4):** every Git-commit outcome (success / idempotent / conflict / failure / repo-unavailable) is expressed as a flash + 303 redirect; admission/form failures re-render 400 with all owner-entered content preserved (task id, profile, clauses, evidence, risk-hint checks, non_scope/constraints).
- **No invented authority:** profile, task id, kept risk hints, and evidence declarations are explicit owner form inputs; admission failures are mapped from typed sentinels (`errors.Is` only). Evidence kind/path must be declared together (bounded UI surface: repo-path visible evidence; half-declarations are typed-rejected). `non_scope`/`constraints` content still blocks freeze via the P0-06 gate (`ErrUnsupportedDraftSemantics`) instead of being dropped.

## 3. Why this is the narrowest valid seam (and rejected alternatives)

- **`internal/artifacts/v141_freeze_commit.go` (chosen):** the write-side complement of P0-02's `v141_intent_loader.go` in the package that already owns the `.tally` artifact layout constants (`TasksDir`, `VerificationsDir`) and pinned-commit semantics; `intentevidence.ArtifactPath` stays the single source of the intent path. Rendering is `intentevidence.FrozenIntent.Render()`, `yaml.Marshal(TaskContract)`, `yaml.Marshal(verification.Frozen.Contract())` — all strict-parse-reversible by their existing authorities (roundtrip hash equality is test-pinned); no serialization is invented here.
- *Rejected:* putting the transaction inside `internal/intakefreeze` — that package's documented contract (P0-06/P0-07) is a pure domain boundary with no Git writes/storage; adding a Git-writing file would break a declared boundary that other tasks rely on.
- *Rejected:* implementing the transaction in the handler — CAS/identity/atomicity semantics would be testable only over HTTP and the layout knowledge would leak into the presentation layer.
- *Rejected:* extending `internal/gitwriter` with artifact knowledge — it is the single Git-write authority and must stay artifact-format-agnostic (PATH_OWNERSHIP: "prefer composition; avoid modifying core writer"; core writer is zero-changed).
- *Template edit (`web/templates/v14_launch.tmpl`):* outside the listed primary paths, but PATH_OWNERSHIP releases the file after P0-04 ("exclusive **during** P0-04"), and P0-07's completion report explicitly assigns "the handler confirm form carrying evidence declarations" to P0-08's serial lane. Without the task-id/profile/evidence inputs the Goal (turn confirmation into frozen artifacts) is unreachable — freeze admission would always fail with `ErrMissingEvidence`.

## 4. Exact source/tests read (L3 evidence)

- Task authority: this task file, `DECISION_LOCKS.md`, `PRD_AMENDMENTS.md`, `SOURCE_BASELINE.md`, `PATH_OWNERSHIP.md`, `TASK_DAG.md` (P0-08 row), prior reports P0-04/05/06/07 (serial-lane handoffs).
- Tests first (reference order): `internal/gitwriter/writer_test.go` (`TestCommitValidatesExpectedBase` family, fixtures), `internal/web/handlers/v14_launch_test.go`, `internal/artifacts/v141_intent_loader_test.go` + `v13_loader_test.go` (fixtures), `internal/intakefreeze/intakefreeze_test.go` (fixture shape).
- Interfaces: `gitwriter.CommitRequest{ExpectedBase, Files}` / `Writer.Head` / `ErrBaseRevision` / `ErrNoChanges`; `intentevidence.ArtifactPath/Render/Parse`; `taskcontract.Parse` (strict KnownFields, `Hash` is `yaml:"-"`); `verification.Frozen.Contract()/Hash()` + `verification.Parse`.
- Core implementation: `internal/gitwriter/writer.go` (base validation under locks, apply/revert rollback), `internal/web/handlers/v14_launch.go`, `internal/intakefreeze/{intakefreeze,verification}.go`, `internal/artifacts/v13_loader.go` (path/glob/manifest rules), `internal/web/handlers/content.go` (`ContentStore` — the existing UI→gitwriter composition precedent), `cmd/server/main.go` (production injection of `ContentRepo`/`Identity`).
- External reference used: none (pack: current Tally source is canonical; git update-ref/worktree docs not needed — CAS semantics already provided by gitwriter).

## 5. Changed files / semantic changes

- `internal/artifacts/v141_freeze_commit.go` (new): `FreezeCommitRequest`, `FreezeCommit`, `freezeCommitFiles`, typed `ErrFreezeIdentityMismatch`; typed gitwriter sentinels pass through unwrapped.
- `internal/artifacts/v141_freeze_commit_test.go` (new): 5 tests — success + triple reload (hash/revision equality via `Load`/`LoadIntent`), stale base conflict, partial-failure rollback (blocked apply mid-set), duplicate-confirm `ErrNoChanges`, identity-mismatch rejection.
- `internal/web/handlers/v14_launch.go`: `V14ConfirmDraft` rewired to freeze (PRG outcomes); new `v14CommitFrozen`, `v14FreezeInputsOf`, `v14FreezeErrText`, `v14StringIn` (template func `has`); `V14DraftView` gains `TaskID`/`Profile`/`RiskHints`, `V14AcceptanceEdit` gains evidence fields, `Confirmed` receipt field removed (success is now a redirect, the fake "confirmed on this page only" state no longer exists); package doc updated to the wired chain.
- `internal/web/handlers/v14_launch_test.go`: boundary test updated to the wired semantics (unwired-receipt assertions replaced; freeze-input fields asserted; admission-failure re-render preserves edits); two draft-header assertions updated.
- `internal/web/handlers/v14_confirm_freeze_test.go` (new): 4 handler tests — success + reload from HEAD, duplicate idempotency (no new commit), admission failures block freeze (missing evidence / half-declared evidence / blank-containing task id; HEAD untouched), repo-unavailable honest degradation.
- `web/templates/v14_launch.tmpl`: confirm form gains task-id input, explicit profile select, risk-hint checkboxes (unchecked by default — advisory never preselected), per-clause evidence kind/path/selector inputs; stale "not yet wired" receipt copy replaced with the true atomic-commit semantics; candidates keep round-tripping via hidden fields so re-renders preserve the full form.
- No changes to `internal/gitwriter`, `internal/intakefreeze`, `internal/intake`, `internal/verification`, `internal/taskcontract`, `internal/intentevidence`, `cmd/server/**`, `server.go`, worker, store, migrations. No new dependency.

## 6. Tests/commands executed (exact results)

Windows / Git Bash, Go `go1.27.1 windows/amd64`, clean tree before this change.

| # | Command | Result |
|---|---|---|
| 1 | `go test -count=1 -run FreezeCommit -v ./internal/artifacts/` | PASS — 5/5 new transaction tests |
| 2 | `go test -count=1 -run 'TestV14Confirm' -v ./internal/web/handlers/` | PASS — 4/4 new handler tests |
| 3 | `go test -count=1 ./internal/artifacts/ ./internal/intakefreeze/ ./internal/gitwriter/ ./internal/intentevidence/ ./internal/web/handlers/` | PASS — all `ok` |
| 4 | `go test ./test/arch/...` | PASS — `ok tally/test/arch` (DL guard suite, incl. v141 guards) |
| 5 | `go build ./...` + `go vet ./internal/artifacts/ ./internal/web/handlers/` | PASS |
| 6 | `go test -count=1 ./internal/...` | PASS — 46 packages `ok`, 0 FAIL |
| 7 | `DATABASE_URL=postgresql://postgres:postgres@localhost:15432/postgres go test -count=1 -tags=integration ./test/e2e/... ./internal/web/handlers/` | PASS — `test/e2e` ok (206s), handlers ok (97s, DB-gated tests ran against the shared local Postgres) |
| 8 | `gofmt -l` on all five changed/new Go files | PASS — no output |
| 9 | `git diff --check` | PASS — clean |

`make check` NOT_RUN(reason: no `make` executable in this Git Bash environment, consistent with P0-04/05/06/07; its constituent families were executed directly as #4/#6/#7).

Task-contract Verification coverage: success (`TestFreezeCommitCommitsTripleAndReloads`, `TestV14ConfirmFreezesTripleAtomicallyAndReloads` — AC2 incl. hash/revision roundtrip equality), stale base (`TestFreezeCommitStaleBaseConflicts` — AC1, plus handler conflict-flash branch), partial failure rollback (`TestFreezeCommitPartialFailureLeavesNoArtifacts` — mid-apply failure, zero-file residue, tree unchanged), duplicate confirm idempotency (`TestFreezeCommitDuplicateConfirmIsErrNoChanges`, `TestV14ConfirmDuplicateIsIdempotent` — no second commit, honest flash).

## 7. Baseline failures observed

None. All focused baselines passed before/after; two test expectations in `v14_launch_test.go` asserted the pre-P0-08 "freeze not wired" receipt and were updated with the behavior they documented (that receipt no longer exists by design).

## 8. Acceptance criteria

- [x] **Stale HEAD conflicts instead of silently rebasing** — explicit `ExpectedBase` validated by gitwriter under lock; stale base → typed `ErrBaseRevision`, empty result, HEAD unchanged, work tree clean, zero artifact files on disk (transaction test) and an explicit conflict flash with no write at the handler layer.
- [x] **Successful confirm is reloadable from committed artifacts** — after `FreezeCommit`, `artifacts.Load` at the new HEAD returns the TaskContract with the frozen content hash and the frozen VerificationContract with the frozen hash, and `artifacts.LoadIntent(repo, commit, taskID)` returns the FrozenIntent with equal revision/source-ref/risk-hints; proven at both transaction and handler level.

## 9. Production reachability

**PASS** — the confirm POST route is already mounted in the production composition root (`cmd/server` → `web.NewRouter` → `RegisterRoutes` → `RegisterV14Routes`), and the freeze path uses only already-injected production dependencies (`Server.Content.RepoPath`, `Server.Content.Identity`). No composition-root edit was required or made; the e2e/handler integration run (#7) exercised the real handler stack.

## 10. Completion report (per task contract format)

1. **status:** DONE
2. **changed files:** `internal/artifacts/v141_freeze_commit.go` (new), `internal/artifacts/v141_freeze_commit_test.go` (new), `internal/web/handlers/v14_launch.go`, `internal/web/handlers/v14_launch_test.go`, `internal/web/handlers/v14_confirm_freeze_test.go` (new), `web/templates/v14_launch.tmpl`, plus docs: this report, task-file addendum, `STATUS.md`, `docs/architecture/CURRENT_CODEBASE.md`, `docs/implementation/CURRENT_IMPLEMENTATION_STATUS.md`
3. **semantic changes:** user confirmation now atomically freezes the confirmed draft into one Git commit containing FrozenIntent + TaskContract + frozen VerificationContract at a single expected base; stale bases conflict (never rebase), duplicate confirms are content-idempotent, partial failures leave nothing, and the PRG contract is enforced on every Git outcome
4. **tests/commands:** see §6 (all PASS; `make check` NOT_RUN with reason)
5. **baseline failures:** none
6. **remaining risks:** see §11
7. **Decision Lock touched:** none
8. **commit:** recorded in the task-file addendum / final message

## 11. Remaining risks / notes

- **The confirm UI declares one visible repo-path evidence per acceptance clause.** The domain (`intakefreeze.VerificationPlan`) supports N declarations per clause, held-out declarations with store/ref, and runtime isolation requirements; the UI surface bounds itself to the single-declaration visible case (multiple declarations, held-out, `validator` kind with its `validator` ref, and `protection` remain programmatic-API concerns). `validator` is deliberately not offered in the select because this declaration surface cannot express its required `validator` field — offering it would guarantee a typed rejection.
- **Stale-base conflict is proven at the transaction seam,** not over HTTP: the handler reads fresh HEAD immediately before committing, so the only race window is between that read and gitwriter's locked re-validation — exactly what `gitwriter.ErrBaseRevision` covers and what the transaction test pins.
- The freeze commit overwrites the three artifact paths if the same task id is re-confirmed with *changed* content (new content-derived revisions; history remains in Git). Constant-content re-confirmation is the tested idempotent no-op. Append-only supersession semantics for *revisions* of frozen contracts remain the province of later lanes (P1/P3); this task does not add revision bookkeeping beyond content-derived hashes, consistent with P0-07 ("re-confirmation = new freeze").
- HIGH review (per MODEL_ASSIGNMENT) has not yet executed; this report is the review input.

## 12. Decision Locks confirmation

No lock text edited and none bypassed: **DL-03** (all Git writes via `internal/gitwriter`, which is zero-changed; no Runtime/agent involvement in this path), **DL-04** (the three authoritative artifacts land in and are reloaded from one pinned commit; identity gate enforces shared task identity), **DL-06** (only owner-confirmed facts enter freeze: explicit risk-hint selection, declared evidence; candidate signals never auto-promoted), **DL-17** (no hidden material read), **DL-18** (existing domains consumed read-only; `gitwriter`/`intakefreeze`/`verification`/`taskcontract`/`intentevidence` all zero-changed). `test/arch` guards green.
