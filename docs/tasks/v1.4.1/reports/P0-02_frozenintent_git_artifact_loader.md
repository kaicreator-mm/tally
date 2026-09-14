# P0-02 — FrozenIntent Git artifact loader: Execution Report

- **TASK:** P0-02
- **STATUS:** DONE
- **Date:** 2026-09-12
- **Phase / Wave:** P0 / W04
- **Execution/Review model:** LOW / LOW (per task pack; review passes on green focused tests + AC — no source/API mismatch encountered)
- **Depends on:** P0-01 (DONE, commit `9b49274`) — satisfied
- **Hidden material:** none read; `docs/tasks/v1.4.1/hidden/` untouched (DL-17)
- **Authority chain read in order:** task contract → `DECISION_LOCKS.md` (DL-04, DL-09 pattern, open-seams ownership) → `PRD_AMENDMENTS.md` (A-02) → `SOURCE_BASELINE.md` → current local source → L3 reference pack

## 1. Deliverable

`artifacts.LoadIntent(repoPath, rev, taskID)` in the new file
`internal/artifacts/v141_intent_loader.go`: opens the git repository, resolves
`rev` to a commit (same go-git revision syntax as `Load` / `LoadPolicies`),
reads `.tally/intents/<task_id>.yaml` **from that commit's tree snapshot**
(never the working tree — INV-ART-01, DL-04's pinned-commit discipline), and
returns a deterministic `IntentArtifact{Path, SourceCommit, Intent}` where
`Intent` is the immutable `intentevidence.FrozenIntent` produced by
`intentevidence.Parse` (revision recomputed from the Git blob's actual
content — no declared/observed drift channel).

Typed rejections (new package-level sentinels, `errors.Is`-judgeable):

- `ErrIntentNotFound` — the artifact for the requested identity is not in the
  pinned commit's tree (never frozen, or commit predates/postdates the freeze);
- `ErrIntentInvalid` — the artifact exists but fails `intentevidence.Parse`
  (malformed YAML, unknown schema fields, declared `revision`, out-of-vocabulary
  `task_type`, empty/multi-doc, whitespace identity). The intentevidence typed
  sentinels (`ErrInvalidEvidence` / `ErrUnsupportedTaskType`) are wrapped with
  `%w` through, so both loader-level and evidence-level `errors.Is` classification
  work;
- `ErrIntentIdentityMismatch` — the artifact's content declares a different
  `task_id` than the requested/path-derived identity (identity is
  content-derived; a single-artifact load has no peers to arbitrate with, so
  any drift fails closed instead of guessing).

Infrastructure failures (repo open, revision resolution, object read) remain
plain wrapped errors, exactly matching the existing `Load` / `LoadPolicies`
taxonomy: artifact-level invalidity never masquerades as infrastructure
failure or vice versa (pinned by test).

## 2. Why this is the narrowest valid seam

- Reuses the existing loader conventions instead of inventing a parallel
  mechanism: same open→resolve→commit→tree prologue and error wording as
  `v13_loader.go` / `v14_policy_loader.go`; same go-git dependency (already
  whitelisted); `intentevidence.ArtifactPath` is the only source of the path
  literal (no scattered `.tally/intents` strings); content validation is
  entirely `intentevidence.Parse` — the loader adds only existence and
  identity adjudication.
- Single-artifact entry point (not a snapshot iterator): the task goal and AC
  are per-identity ("load `.tally/intents/<task_id>.yaml`"), the consuming
  tasks (P0-08 reload-after-commit, P0-09 e2e gate) reload one task's intent at
  the pinned commit, and a snapshot variant would add unowned surface. If a
  batch need appears, it is a later task's seam.
- No existing file's semantics changed: `v13_loader.go`,
  `v14_policy_loader.go` and their tests are untouched. Package-level helpers
  (`repoFixture`, `writeWorktreeFile`) are reused from the existing test file
  in the same package.
- No new dependency (`test/arch/deps_test.go` whitelist unchanged and green).

## 3. Exact source/tests read (L3 evidence)

- Tests first (per reference order): `internal/artifacts/v13_loader_test.go`
  (incl. `TestLoadReadsGitSnapshotNotWorkingTree`), `internal/artifacts/v14_policy_loader_test.go`.
- Interfaces: `artifacts.Load(repoPath, rev)`, `artifacts.LoadPolicies(repoPath, rev)`.
- Core pattern: `internal/artifacts/v13_loader.go`, `internal/artifacts/v14_policy_loader.go`.
- P0-01 domain consumed: `internal/intentevidence/intentevidence.go`
  (`Freeze`, `Parse`, sentinels), `internal/intentevidence/artifact.go`
  (`ArtifactDir`, `ArtifactPath`), its tests (fixture/fail-closed idioms).
- External reference used: none (pack says current Tally source is canonical).
  go-git v5.12.0 `plumbing/object/tree.go` (`Tree.File` /
  `object.ErrFileNotFound`) inspected in the module cache for the
  existence-error taxonomy.

## 4. Semantic changes

- New production file `internal/artifacts/v141_intent_loader.go`: one exported
  entry point (`LoadIntent`), one result type (`IntentArtifact`), three typed
  sentinel errors. Read-only; no Git write, no DB, no runtime involvement.
- No behavior of any existing function changed; `gitwriter` (DL-03) untouched.
- Decision Locks: none touched. DL-04 reinforced (loader only reads a resolved
  pinned commit; `SourceCommit` reports the **observed** commit sha, not the
  caller's revision expression — §13 desired/observed discipline). DL-17
  respected.

## 5. Tests/commands executed (exact results)

Windows / Git Bash, Go `go1.27.1 windows/amd64`, clean tree before this change
(HEAD `9b49274`). Changes are purely additive (2 new files), so the package
baseline equals current-minus-new-files; no baseline failure existed to mask.

| # | Command | Result |
|---|---|---|
| 1 | `go test ./internal/artifacts/` (after change) | PASS — `ok tally/internal/artifacts 6.314s` (13 existing + 6 new top-level tests, all green) |
| 2 | `go test -count=1 -run TestLoadIntent -v ./internal/artifacts/` | PASS — 6/6 top-level, 7/7 subtests (see §6) |
| 3 | `go test -count=1 ./internal/artifacts/ ./internal/intentevidence/ ./internal/taskcontract ./internal/engineeringpolicy` | PASS — all `ok` (fresh, uncached; P0-01 domain + idiom packages unaffected) |
| 4 | `go test ./test/arch/...` | PASS — `ok tally/test/arch 0.894s` (incl. deps whitelist + A0-04 v1.4.1 guards) |
| 5 | `go build ./...` | PASS |
| 6 | `go vet ./internal/artifacts/` (and `go vet ./...` inside preflight) | PASS |
| 7 | `go run ./scripts/preflight` | PASS — exit 0, `BASELINE_OK`, `is_code_regression=false`; `unit_internal` = `go test ./internal/...` PASS (34.5s); `integration`: SKIP — NOT_RUN(reason: opt-in DB-gated; not required by this task's Verification section, which is commit-tree fixture tests) |
| 8 | `gofmt -l` on the two new files | PASS — no output |
| 9 | `git diff --check` (pre-commit) | PASS — clean |

Pre-existing worktree note (not introduced by this task): `gofmt -l
internal/artifacts/` also lists `v13_loader.go` / `v13_loader_test.go` because
those tracked files are checked out CRLF locally (`git ls-files --eol`: `i/lf
w/crlf`, index is LF). Per AGENTS.md §16 they were left untouched.

## 6. Negative/failure-path tests added (all PASS)

`internal/artifacts/v141_intent_loader_test.go` (reuses the v13 git repo
fixture):

- `TestLoadIntentReadsCommitTreeNotWorkingTree` — **working-tree divergence**
  (task Verification requirement): uncommitted worktree rewrite of the
  committed intent and an untracked intent file are invisible at HEAD; after
  committing v2, historical sha1 re-loads v1 and HEAD loads v2, revisions
  content-bound and non-drifting;
- `TestLoadIntentRejectsInvalidDeterministically` — 7 subtests (unknown field,
  declared `revision`, empty doc, multi-doc, non-mapping, out-of-vocabulary
  `task_type`, whitespace content identity) each typed
  `ErrIntentInvalid` with intentevidence sentinels穿透 and no misclassification
  as not-found/identity-mismatch;
- `TestLoadIntentRejectsIdentityMismatch` — content `task_id` ≠ requested
  identity → `ErrIntentIdentityMismatch`, fail-closed;
- `TestLoadIntentNotFound` — uncommitted-for identity → `ErrIntentNotFound`;
  unopenable repo stays infrastructure (not NotFound); empty/internal-whitespace
  request identity surfaces `intentevidence.ErrInvalidEvidence`;
- `TestLoadIntentUnresolvableRevision` — bad revision is infrastructure, not
  any of the three artifact-level rejections;
- `TestLoadIntentAtHead` — positive path: content/revision bound to the Git
  blob (`Revision()` equals a locally re-frozen evidence's revision),
  `SourceCommit` = HEAD sha, per-task isolation, byte-deterministic re-load
  (Equal + same Revision + same SourceCommit).

## 7. Baseline failures observed

None. All commands in §5 passed; no pre-existing failure encountered or masked.

## 8. Acceptance criteria

- [x] **Given source commit, loader returns deterministic FrozenIntent or
  typed rejection** — `TestLoadIntentAtHead` (deterministic re-load, revision
  bound to blob content, observed source commit) vs.
  `TestLoadIntentRejectsInvalidDeterministically` / `TestLoadIntentNotFound` /
  `TestLoadIntentRejectsIdentityMismatch` (typed rejections), plus
  working-tree divergence proof (`TestLoadIntentReadsCommitTreeNotWorkingTree`).

## 9. Production reachability

**N/A(owned integration tasks pending):** the loader is production code in
`internal/artifacts` but no composition root calls it yet by design — P0-08
(confirm→freeze transaction) is its production consumer (reload-after-commit
proof) and P0-09 owns the e2e gate. This task owns the pinned-commit read path
only; no handler/worker file was touched.

## 10. Completion report (per task contract format)

1. **status:** DONE
2. **changed files:** `internal/artifacts/v141_intent_loader.go` (new), `internal/artifacts/v141_intent_loader_test.go` (new), `docs/tasks/v1.4.1/tasks/P0/P0-02_...md` (appendix), `docs/tasks/v1.4.1/reports/P0-02_...md` (new), `docs/tasks/v1.4.1/STATUS.md` (6/57 → 7/57), `docs/architecture/CURRENT_CODEBASE.md`, `docs/implementation/CURRENT_IMPLEMENTATION_STATUS.md`
3. **semantic changes:** new pinned-commit loader for FrozenIntent artifacts with typed not-found/invalid/identity-mismatch rejections; no existing behavior changed
4. **tests/commands:** see §5 (all PASS; integration NOT_RUN: opt-in DB-gated, not required by task)
5. **baseline failures:** none
6. **remaining risks:** see §11
7. **Decision Lock touched:** none
8. **commit:** recorded in the task-file addendum / final message

## 11. Known uncertainty / version sensitivity

- `tree.File` maps every `FindEntry` failure to `object.ErrFileNotFound`
  (go-git v5.12.0), so a corrupt-tree read could surface as
  `ErrIntentNotFound` rather than an infrastructure error. A go-git upgrade
  that changes this taxonomy would only shift which typed sentinel a corrupt
  repo reports; the normal-path classification is pinned by tests.
- Multiple `%w` wrapping requires Go ≥ 1.20 (module directive 1.23) — fine.
- HIGH/independent review per MODEL_ASSIGNMENT is LOW for this task; the
  review rule ("escalate only on source/API mismatch") was met with no
  mismatch.

## 12. Remaining risks / notes

- No batch/snapshot intent loader exists; if a future task needs
  "all intents in a commit", that is new surface to be owned then (single
  `LoadIntent` kept the public area narrow per the pack's Do).
- The identity-mismatch rule is stricter than filename-based convention: an
  artifact committed under a "wrong" filename is unreachable through
  `LoadIntent` even if its content is valid — deliberate fail-closed (P0-08's
  writer derives the path from the same `ArtifactPath`, so honest writes
  always round-trip).
