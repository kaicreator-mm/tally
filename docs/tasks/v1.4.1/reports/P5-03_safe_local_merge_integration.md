# P5-03 Execution Report — Safe local merge integration

- **Status:** DONE
- **Date:** 2026-09-12
- **Baseline:** 25/57 (P5-01 + P5-02 landed); working tree shared with parallel lanes P2-01 / P4-01 / P6-04 / P6-06 (their files untouched).

## 1. Seam adjudication — why this is the narrowest valid seam

The task builds on the seams P5-01/P5-02 froze: the same package (`internal/gitintegration`), the same validation pipeline, zero new authority, no Decision Lock change.

**Ruling: safe local merge is one new typed operation on the existing `gitintegration.Service` — `Merge(ctx, MergeRequest)` — that merges the candidate's deterministic integration branch (P5-02's `refs/heads/tally/integration/<hex>`) into a caller-declared, format-guarded target ref. Conflicts are detected by a read-only `git merge-tree --write-tree` preflight (EXT-GIT-MERGE-TREE) and returned as a typed result; the only ref-write path is `git update-ref --stdin`'s `update <ref> <new> <old>` compare-and-swap primitive (EXT-GIT-UPDATE-REF). True merges are built with plumbing (`commit-tree` + two `-p`), so the work tree and index are never touched and no partial hidden state can exist.**

### Evidence read (in the mandated order)

1. Task contract `docs/tasks/v1.4.1/tasks/P5/P5-03_safe_local_merge_integration.md` including its L3 pack; then `DECISION_LOCKS.md` (DL-03, DL-13), `PRD_AMENDMENTS.md`, `SOURCE_BASELINE.md` (no merge-specific guidance — semantics defined by this task's contract).
2. Current source: `internal/gitintegration/service.go` + `branch.go` (frozen seam: `AcceptanceFact` pipeline, deterministic branch naming, CAS ref helpers) and their reports; `internal/gitwriter/writer.go` + `lock.go` (public API `New`/`Head`/`Commit`; `runGit`/`sanitizedGitEnv` unexported); `internal/changeset` API.
3. External references used: **EXT-GIT-MERGE-TREE** (`--write-tree -z --name-only`: exit 0 = clean with tree OID as first NUL field, exit 1 = conflict with conflicted filenames between the tree OID and the empty separator field — verified empirically against the installed git 2.55.0 before coding); **EXT-GIT-UPDATE-REF** (`update <ref> <new> <old>` refuses when the ref is not at `<old>` — verified empirically); **EXT-GIT-MERGE** (conflict is a non-success that must be surfaced, never auto-resolved — honored structurally: no merge strategy options, no `-X ours/theirs`, no work-tree merge at all).

### Why no narrower seam exists

1. **gitwriter has no merge/ref API and must not grow one for this task.** Its exported surface is HEAD-worktree commits only; a merge advances an arbitrary user-owned ref. P5-02 already established that ref plumbing gitwriter does not expose lives in `gitintegration` with `gitwriter.RunGit` discipline (see §2 — this task resolved the P5-02 handoff by exporting the helper instead of adding a third copy).
2. **Plumbing-only merge replaces a work-tree merge, not the write boundary.** `merge-tree` (preflight, read-only) + `commit-tree` (deterministic two-parent merge commit) + CAS `update-ref` never touch the work tree or index, which is what makes "conflict returns structured result without partial hidden state" structural rather than best-effort: there is no index to clean up and no `MERGE_HEAD` to abort. A porcelain `git merge` in the work tree was rejected precisely because its failure modes are work-tree state.
3. **No force / history rewrite exists structurally.** The only ref mutation is `update <ref> <new> <old>` with an explicit old value; there is no `delete`, no `update` without old, no `reset`, no `push --force`. Fast-forward (target is an ancestor of the branch tip) only advances the ref — both sides' commits remain ancestors of the new target (asserted by test).
4. **Both expected values are caller-declared and CAS-checked (base drift is typed, never a silent rebase).** `ExpectedTarget` (target ref) and `ExpectedBranchCommit` (integration branch tip, i.e. `Integrate`'s commit) are required; drift on either side — including a drift that happens between preflight and the ref update — is `ErrStaleBase`. A defense-in-depth ancestry check additionally verifies the integration branch actually carries the ChangeSet's fixed base before merging.
5. **Rejected alternative A — merge by re-applying ChangeSet files via `gitwriter.Commit` on the target.** That is what `Integrate` already does; it silently discards any concurrent target-side changes (last-writer-wins), which is exactly the history-destroying behavior this task forbids. Merge semantics require Git's three-way merge over a real base.
6. **Rejected alternative B — model/strategy-driven conflict resolution (`-X ours`, auto-retry with the other side).** Forbidden by the contract ("no autonomous conflict resolution", "fail safe on merge conflict"). No merge strategy options appear in any argv in this change.
7. **Rejected alternative C — allow merging into the checked-out branch of the integration worktree.** `git update-ref` would happily move a ref under a checked-out work tree, leaving that work tree and index silently desynchronized — a hidden partial state worse than a conflict. `Merge` enumerates `git worktree list --porcelain` (main + linked worktrees) and rejects a target ref that is any worktree's checked-out branch (`ErrTargetCheckedOut`), before any read/write of refs.
8. **Audit refs are preserved structurally.** The operation writes only the target ref; the `tally/` integration branch is read-only here, and `refs/heads/tally/**` is rejected as a merge target by `validateTargetRef`. Tests assert the integration branch is byte-identical after every outcome (success, conflict, drift, rejection).

### Resulting shape

- `Service.Merge(ctx, MergeRequest{ChangeSet, Acceptance, RepositoryPath, TargetRef, ExpectedTarget, ExpectedBranchCommit}) (MergeResult, error)`:
  1. frozen pipeline: identified candidate (`ErrCandidateUnidentified`), complete + matching acceptance (`ErrNotAccepted`/`ErrAcceptanceMismatch`), non-empty non-protected repository path (`ErrNoRepository`/`ErrProtectedTarget`), both expected values required (`ErrNoExpectedBase`), real repository (`gitwriter.ErrNotRepository`);
  2. `TargetRef` format-guarded: full `refs/heads/<name>` only, no leading `-`, no whitespace/control/ref metacharacters (`..`, `~^:?*[]\`, `@{`, `//`, trailing `/`, `.lock`), and never inside `refs/heads/tally/` (`ErrInvalidTargetRef`);
  3. target must not be checked out in any worktree (`ErrTargetCheckedOut`);
  4. CAS on both refs (`ErrStaleBase`); missing integration branch is `ErrIntegrationBranchMissing`; branch not carrying the candidate base is `ErrStaleBase`; branch already contained in target is `ErrAlreadyMerged` (typed non-success, target untouched);
  5. fast-forward when target is an ancestor (ref-only advance, `FastForward: true`); otherwise `merge-tree` preflight → `*MergeConflictError{Paths}` (errors.Is `ErrMergeConflict`) on conflict; clean → deterministic two-parent `commit-tree` with message `tally: merge changeset <id>` and explicit identity (`-c user.name/user.email`, never ambient config) → CAS ref update.
- New sentinels: `ErrMergeConflict`, `ErrInvalidTargetRef`, `ErrTargetCheckedOut`, `ErrIntegrationBranchMissing`, `ErrAlreadyMerged`; new typed error `*MergeConflictError` carrying conflict paths.
- Exit-code-classified commands (`merge-tree`, `merge-base --is-ancestor`) read the process exit code from `gitwriter.GitError` via `errors.As(*exec.ExitError)` — no output string matching anywhere in the merge path; ctx cancellation is checked first and passed through unwrapped.

## 2. Changed files

- `internal/gitintegration/merge.go` (new): merge sentinels + `MergeConflictError`, `MergeRequest`/`MergeResult`, `Merge`, target-ref validation, checked-out-target guard, CAS ref update, ancestry and `merge-tree` preflight helpers.
- `internal/gitintegration/merge_test.go` (new): 12 tests over real temporary Git repositories (existing conventions), covering clean true merge / fast-forward / conflict / target drift / branch drift / replay / checked-out target / branch without candidate base / missing branch / validation table / non-repository / cancelled context.
- `internal/gitintegration/branch.go`: removed the duplicated `runGit`/`sanitizedGitEnv` copies (P5-02 handoff risk); the local `runGit` is now a one-line forwarder to `gitwriter.RunGit` (single source of truth for argv/env/stdin discipline).
- `internal/gitwriter/writer.go` (authorized minimal export per P5-02 handoff): unexported `runGit` gained a `stdin` parameter (7 call sites updated, semantics unchanged — empty stdin) and a new exported `RunGit(ctx, dir, stdin, args...)` wrapper documents that it exists for the `gitintegration` semantic layer; no behavior, locking, or commit-path change.
- Everything else: **unchanged**.

## 3. Semantic changes

An accepted candidate can now be merged into a local target ref as one typed operation that: never auto-resolves conflicts (typed conflict result with the conflicted path list), never forces or rewrites history (CAS-only ref advance, both sides remain ancestors), preserves audit refs (integration branch read-only, `tally/` namespace unmergeable), types all base-drift as `ErrStaleBase`, refuses to move refs under any checked-out work tree, and never touches a work tree or index (no partial hidden state by construction). Acceptance criterion met: clean merge succeeds; conflict returns a structured result without partial hidden state.

## 4. Tests / commands executed

- `go test ./internal/gitintegration/ ./internal/gitwriter/` — **PASS** (ok 20.3s / 13.1s after `go clean -testcache`; gitintegration now 32 tests = 13 from P5-01 + 7 from P5-02 + 12 new; gitwriter unchanged-green with the `runGit` signature change)
- `go test ./internal/gitintegration/ -run TestMerge -v` — **PASS** (all 12 new tests individually listed)
- `go test ./test/arch/...` — **PASS** (dependency policy intact; no new dependency)
- `go vet ./internal/gitintegration/ ./internal/gitwriter/` — **PASS**
- `gofmt -l` on all touched files — clean (pre-existing baseline gofmt deltas in untouched `gitwriter/lock.go`, `lock_test.go`, `writer_test.go` were observed and deliberately not "cleaned up" — out of scope)
- `git diff --check -- internal/gitintegration internal/gitwriter` — clean (exit 0)
- New files LF-only (no CR bytes), per `.gitattributes`

Negative/failure-path tests added: conflict (asserts `ErrMergeConflict` via `errors.Is` and the exact conflicted path list via `errors.As`, target ref + integration branch + HEAD + work tree all untouched, no leftover `MERGE_HEAD`); target base drift; branch base drift (asserts neither ref moved); replay of an already-merged branch (`ErrAlreadyMerged`); checked-out target (`ErrTargetCheckedOut`, including the default-branch case); integration branch not carrying the candidate base (`ErrStaleBase`); missing integration branch; 11-case validation table (each asserts both refs unmoved); non-repository; cancelled context. Happy paths assert two parents, both sides' file content in the merge tree, the deterministic message, fast-forward creating no new commit, history-advance (ancestor checks), and audit-ref preservation.

## 5. Baseline failures observed

None in owned or neighbor packages. Cross-lane noise: `git diff --check` CRLF warnings for `internal/verification/*` (P6-04) and pre-existing `gitwriter` test-file gofmt deltas — not owned, not touched.

## 6. Remaining risks

- `git merge-tree --write-tree` requires git ≥ 2.38 (installed: 2.55.0). On older installs the preflight fails with a plain `gitwriter.GitError` (usage error) — fail-safe, but not a dedicated sentinel; no fallback merge strategy was implemented (the contract prefers fail-safe over a riskier work-tree merge).
- On conflict, `merge-tree --write-tree` writes an unreferenced (conflict-marked) tree object into the object DB. It is unreferenced garbage collectable by git and no ref can reach it; noted for completeness.
- The checked-out-target guard enumerates worktrees known to the target repository. A ref checked out in a worktree of a *different* repository that shares objects cannot be detected — not a configuration Tally creates.
- The final CAS-race classification path (update fails → re-read → `ErrStaleBase`) is structurally correct but not deterministically testable without injection hooks; covered by code review.
- Production wiring (who calls `Merge` after `Integrate`, with which target ref) is a later integration task; this task freezes the semantic operation, per the P5-01/P5-02 seam pattern.

## 7. Decision Locks touched

None edited, weakened, or routed around. DL-03 respected (no second Git-write authority: ref/merge plumbing sits in the pack-designated `gitintegration` layer over the single `gitwriter.RunGit` discipline; commits to a worktree still only via `gitwriter.Commit`). DL-13 respected (scope ends at controlled local integration; no remote, no force, no history-rewrite path exists). Trust invariants respected: unknown states fail safe (missing branch/expected values are typed errors, never guessed); cancellation is not fabricated into a merge result; the owner primary worktree can never be the target.

## 8. Commit hash

Not committed — orchestrator owns commits per task protocol.
