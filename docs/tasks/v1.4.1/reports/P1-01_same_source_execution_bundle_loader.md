# P1-01 — Same-source execution bundle loader: Execution Report

- **TASK:** P1-01
- **STATUS:** DONE
- **Date:** 2026-09-12
- **Phase / Wave:** P1 / W08
- **Execution/Review model:** LOW / HIGH (per task pack; HIGH review pending — authority-boundary task)
- **Depends on:** P0-09 (DONE, commit `9d12cf8`) — satisfied
- **Hidden material:** none read; `docs/tasks/v1.4.1/hidden/` untouched (DL-17)
- **Authority chain read in order:** task contract → `DECISION_LOCKS.md` (DL-04 primary, DL-09/BI-01, DL-17, DL-18) → `PRD_AMENDMENTS.md` → `SOURCE_BASELINE.md` → `PATH_OWNERSHIP.md` → current local source → L3 reference pack

## 1. Deliverable

`artifacts.LoadBundle(repoPath, rev, taskID)` in the new file
`internal/artifacts/v141_bundle_loader.go`: opens the git repository, resolves
`rev` to **exactly one** commit, and from **that commit's tree snapshot**
(never the working tree, never a second resolution) loads all four classes of
execution authority for one task identity:

- FrozenIntent — `.tally/intents/<task_id>.yaml` via the shared `loadIntentTree`
  helper (identical adjudication to `artifacts.LoadIntent`);
- TaskContract — `.tally/tasks/<task_id>.yaml` via `taskcontract.Parse` +
  `ValidateStatic` (same chain as the v13 loader's per-file `loadTask`);
- frozen VerificationContract — `.tally/verifications/<task_id>.yaml` via
  `verification.Parse` + `verification.FreezeContract` at zero-value
  `FreezePolicy` (P1 standard path, same chain as the v13 loader's
  `loadVerification`);
- EngineeringPolicies — `.tally/policies/*.yaml` via the existing
  `loadPolicyTree` (unchanged: invalid policies recorded per-file in
  `PolicySnapshot.Rejected`, duplicate ids rejected as a group — the exact
  `LoadPolicies` discipline, DL-09/BI-01).

Result: `ExecutionBundle{TaskID, SourceCommit, Intent, Task, Verification,
Policies}` where `SourceCommit` is the **observed** commit sha (not the caller's
revision expression), all three task-scoped artifacts have passed existence +
content + identity adjudication against the requested identity, and `Identity()`
is a deterministic `sha256:<hex>` over canonical text `artifacts/bundle/v1` +
task id + the single source commit + all loaded revisions (`Revisions()`:
intent revision, task `RevisionHash`, frozen verification hash + revision, each
loaded policy's content revision — ascending). Same-commit reloads produce a
field-identical bundle with the same identity; any artifact version change or
source-commit change necessarily changes the identity (task AC: bundle identity
includes one source commit and all loaded revisions).

Typed rejections (new package-level sentinels, `errors.Is`-judgeable, mirroring
the intent loader's trio):

- `ErrTaskNotFound` / `ErrTaskInvalid` / `ErrTaskIdentityMismatch`;
- `ErrVerificationNotFound` / `ErrVerificationInvalid` /
  `ErrVerificationIdentityMismatch`.

Intent-level failures keep the existing `ErrIntentNotFound` / `ErrIntentInvalid`
/ `ErrIntentIdentityMismatch` sentinels. Load order is fixed
(intent → task → verification → policies) so the reported failure is
deterministic when several artifacts are missing. Mixed-source resolution is
impossible by construction: the bundle never calls the per-artifact entry
points (each would resolve `rev` again — for a movable revision like HEAD two
resolutions could straddle a concurrent commit); it resolves once and reads
everything from one tree. A pinned commit that is missing any artifact is a
typed rejection — never auto-upgraded to HEAD or any other source.

## 2. Why this is the narrowest valid seam

- Composes existing loaders instead of inventing a second semantics (pack:
  "Core: compose existing task/verification/policy loaders + FrozenIntent
  loader"): `LoadIntent` was refactored to extract `loadIntentTree`, and the
  bundle reuses that helper verbatim plus the existing `loadPolicyTree`; task /
  verification tree helpers replicate the v13 loader's per-file adjudication
  chains (`Parse` + `ValidateStatic`; `Parse` + `FreezeContract` zero policy)
  as single-identity reads. Both public entries (`LoadIntent`, `LoadBundle`)
  now share `openCommitTree`, so the open→resolve→tree prologue cannot drift.
- Not assembling `contract.Snapshot` / calling `contract.Compile`/`Bind`:
  that is P1-03's owned seam ("production contract compilation assembler …
  Use existing `contract.Compile/Bind`"). P1-01 only resolves the same-source
  bundle; the internal/contract package is zero-changed (its existing
  `Snapshot`/`Compile`/`Open`/`Bind` authority is untouched, DL-18).
- Policy loading reuses `loadPolicyTree` unchanged — rejected policies are
  carried, not fatal, exactly as `LoadPolicies` rules today; whether a
  rejection blocks a task is P1-03's fail-safe decision, not this loader's.
- No new dependency; go-git already whitelisted. No existing exported
  behavior changed: `LoadIntent`'s public signature and semantics are
  byte-identical (pure extraction refactor), proven by its untouched tests.

## 3. Exact source/tests read (L3 evidence)

- Tests first (per reference order): `internal/artifacts/v13_loader_test.go`
  (incl. `TestLoadReadsGitSnapshotNotWorkingTree`), `internal/artifacts/v14_policy_loader_test.go`.
- Interfaces: `artifacts.Load`, `artifacts.LoadPolicies`,
  `artifacts.LoadIntent` (single `sourceCommit` identity on artifact snapshots).
- Core: `internal/artifacts/v13_loader.go` (`loadTask`/`loadVerification`
  chains), `v14_policy_loader.go` (`loadPolicyTree`), `v141_intent_loader.go`,
  `v141_freeze_commit.go` (write-side artifact layout), and the consumption
  seam facts: `internal/contract/contract.go` (`Snapshot.Compile/Open` — the
  "P1-01 ContractSnapshot" the v1.4 compiler doc comment refers to already
  exists), `internal/contract/compile.go` (`Bind` — production call sites 0,
  wired by P1-03, per A0-05 finding), `internal/intentevidence/artifact.go`
  (`ArtifactPath`), `internal/taskcontract/validate.go` (`ValidateStatic` ⊃
  `Validate`), `internal/taskcontract/hash.go` (`RevisionHash` content-derived),
  `internal/verification/freeze.go` / `types.go` (`Frozen` accessors),
  `internal/intakefreeze/verification.go` (`FrozenArtifacts` shape).
- Downstream seams checked to avoid over-building: P1-02 (risk from
  FrozenIntent), P1-03 (Compile/Bind assembler), P1-04 (worker wiring) task
  contracts.
- External reference used: none (pack: current Tally source is canonical).

## 4. Semantic changes

- New production file `internal/artifacts/v141_bundle_loader.go`: one exported
  entry point (`LoadBundle`), one result type (`ExecutionBundle` with
  `Revisions()` / `Identity()`), six typed sentinels, shared `openCommitTree`.
- `internal/artifacts/v141_intent_loader.go`: pure refactor — `LoadIntent` body
  extracted into `loadIntentTree` + `openCommitTree` reuse; public signature,
  adjudication order, error wording and wrapping unchanged; imports narrowed.
- Read-only; no Git write, no DB, no runtime involvement; `gitwriter` (DL-03)
  untouched; `internal/contract`, `internal/intentevidence`,
  `internal/taskcontract`, `internal/verification`, `internal/engineeringpolicy`
  all zero-changed (DL-18).
- Decision Locks: none touched. DL-04 is the implemented lock (one pinned
  source commit for all authoritative artifacts of one task, observed sha,
  no mixed-source read, no auto-upgrade). DL-09/BI-01 inherited via
  `loadPolicyTree`. DL-17 respected.

## 5. Tests/commands executed (exact results)

Windows / Git Bash, clean tree before this change (HEAD `9d12cf8`). Changes are
2 new files + 1 pure-refactor file, so the package baseline equals
current-minus-new-code; no baseline failure existed to mask.

| # | Command | Result |
|---|---|---|
| 1 | `go test ./internal/artifacts/` | PASS — `ok tally/internal/artifacts 10.237s` (all pre-existing tests green on the refactored loader + 4 new top-level tests) |
| 2 | `go test -run 'TestLoadBundle' -v ./internal/artifacts/` | PASS — 4/4 top-level, 9/9 subtests (see §6) |
| 3 | `go test ./internal/artifacts/ -run 'TestLoadReadsGitSnapshotNotWorkingTree\|TestLoadPolicies' -v` | PASS — 7/7 (the reference pack's listed tests) |
| 4 | `go test ./test/arch/...` | PASS — `ok tally/test/arch 0.981s` (incl. `v14_contract_authority_test.go` DL-04 guards) |
| 5 | `go build ./...` | PASS |
| 6 | `go vet ./...` | PASS |
| 7 | `go test ./internal/...` | PASS — 46 packages `ok`, 0 failures (2 no-test-file pkgs) |
| 8 | `gofmt -l` on the three touched files | PASS — no output |
| 9 | `git diff --check` | PASS — clean |

NOT_RUN: `make check` (no make binary in Git Bash; constituent families run
individually per prior-task precedent); `-tags=integration` e2e suite (DB-gated
opt-in; this task's Verification section is commit-tree fixture tests, no
integration requirement).

Pre-existing worktree note (not introduced by this task): `gofmt -l
internal/artifacts/` lists `v13_loader.go` / `v13_loader_test.go` at HEAD too
(CRLF checkouts, `git ls-files --eol`: index LF). Left untouched per AGENTS.md
§16.

## 6. Negative/failure-path tests added (all PASS)

`internal/artifacts/v141_bundle_loader_test.go` (reuses the v13/v141 git repo
fixtures plus the real freeze admission + freeze commit production path):

- `TestLoadBundleAtFreezeCommit` — bundle loaded from the exact freeze commit
  equals the frozen values field-by-field (intent `Equal` + revision, task
  hash, verification hash + revision) and equals the independent loaders
  (`LoadIntent`, `LoadPolicies`); policies come from the same commit;
  `Revisions()` covers all five revision kinds; same-commit reload yields the
  same `Identity()` (task AC);
- `TestLoadBundlePinnedCommitNeverUpgrades` — **the task's divergent-commits
  verification**: freeze at commit2 → policy replaced at commit3 → unrelated
  advance at commit4. Bundle at commit2 carries freeze-time policy v1 and the
  frozen triple (HEAD's v2 invisible); bundle pinned **before** the freeze →
  typed `ErrIntentNotFound`, never auto-upgraded to HEAD; bundle at commit3
  carries v2 and its identity differs from commit2's; bundle at commit4 has
  **identical revisions** to commit3's but a different source commit — and a
  different identity, proving the identity includes the source commit itself;
- `TestLoadBundleTypedRejections` — 9 subtests: missing task / verification /
  intent each report their own typed sentinel (fixed load order, no
  misclassification); task and verification identity drift fail-closed
  (`ErrTaskIdentityMismatch` / `ErrVerificationIdentityMismatch`); invalid task
  (`brokenTaskYAML`) and invalid verification (`brokenVerificationYAML`)
  rejected `ErrTaskInvalid` / `ErrVerificationInvalid` without being confused
  with not-found/identity; unresolvable revision stays infrastructure (against
  all nine sentinels); empty/whitespace request identity passes through
  `intentevidence.ErrInvalidEvidence`;
- `TestLoadBundlePolicyRejectionsCarriedNotFatal` — invalid policy recorded in
  `Policies.Rejected` (`KindPolicy`) while the bundle still loads (fail-safe
  inheritance from `LoadPolicies`; blocking is P1-03's decision); a repo with
  the frozen triple and no policies at all loads a zero-policy bundle with a
  valid identity.

## 7. Baseline failures observed

None. All commands in §5 passed; no pre-existing failure encountered or masked.

## 8. Acceptance criteria

- [x] **Bundle identity includes one source commit and all loaded revisions** —
  `ExecutionBundle.SourceCommit` (observed sha) + `Revisions()` (all five
  revision kinds) feed the deterministic `Identity()`; pinned by
  `TestLoadBundleAtFreezeCommit` (same-commit determinism, full revision
  coverage) and `TestLoadBundlePinnedCommitNeverUpgrades` (policy change →
  identity change; content-identical different-commit bundle → identity
  change).

Task implementation requirements: (1) no mixed working-tree/source-commit
reads — single resolution, tree-only reads, working-tree divergence inherited
from the shared tree helpers and the existing snapshot tests; (2) typed
rejection for missing/mismatched artifact — six new + three reused sentinels,
pinned by `TestLoadBundleTypedRejections`.

## 9. Production reachability

**N/A(owned integration tasks pending):** the loader is production code in
`internal/artifacts`, but no composition root calls it yet by design — the
serial P1 authority chain continues with P1-02 (risk input derivation from the
bundle's FrozenIntent) and P1-03 (the production Compile/Bind assembler over
this bundle), and P1-04 wires the worker. No handler/worker file was touched
(`cmd/worker/v14.go` and `internal/web/handlers/server.go` are shared
hotspots owned by later tasks).

## 10. Completion report (per task contract format)

1. **status:** DONE
2. **changed files:** `internal/artifacts/v141_bundle_loader.go` (new),
   `internal/artifacts/v141_bundle_loader_test.go` (new),
   `internal/artifacts/v141_intent_loader.go` (pure extraction refactor),
   `docs/tasks/v1.4.1/tasks/P1/P1-01_...md` (appendix),
   `docs/tasks/v1.4.1/reports/P1-01_...md` (new),
   `docs/tasks/v1.4.1/STATUS.md` (14/57 → 15/57),
   `docs/architecture/CURRENT_CODEBASE.md`,
   `docs/implementation/CURRENT_IMPLEMENTATION_STATUS.md`
3. **semantic changes:** new same-source execution bundle loader resolving
   FrozenIntent + TaskContract + frozen VerificationContract +
   EngineeringPolicies from exactly one pinned source commit with typed
   missing/invalid/identity rejections and a deterministic
   commit+revisions identity; `LoadIntent` refactored onto the shared tree
   helper with unchanged public semantics
4. **tests/commands:** see §5 (all PASS; `make check` and integration e2e
   NOT_RUN with reasons)
5. **baseline failures:** none
6. **remaining risks:** see §11
7. **Decision Lock touched:** none
8. **commit:** recorded in the task-file addendum / final message

## 11. Remaining risks / notes

- Zero-policy bundles load successfully (the freeze triple does not include
  policies); whether an empty or rejected policy set must block a task is a
  P1-03 fail-safe decision — the bundle carries `Policies.Rejected` untouched
  so the assembler can rule on it.
- `tree.File` maps every `FindEntry` failure to `object.ErrFileNotFound`
  (go-git v5.12.0), the same taxonomy note as P0-02; a corrupt tree could
  surface as a typed not-found rather than an infrastructure error.
- HIGH review per MODEL_ASSIGNMENT still pending (authority-boundary task);
  the review should confirm the no-second-resolution argument in §1 and the
  decision to keep `contract.Snapshot` assembly out of this task (P1-03's
  seam).
- Known source-fact note for downstream tasks: the v1.4 doc comment in
  `internal/contract/compile.go` calls `contract.Snapshot` "P1-01 的
  ContractSnapshot" — that refers to the historical v1.4 program's P1-01; the
  aggregate already exists (`contract.Compile/Open`) and this v1.4.1 P1-01
  deliberately does not duplicate or rewrite it (DL-04/DL-18: read-only
  aggregates with content-derived pins are preserved).
