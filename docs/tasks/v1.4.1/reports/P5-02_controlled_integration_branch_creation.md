# P5-02 Execution Report — Controlled integration branch creation

- **Status:** DONE
- **Date:** 2026-09-12
- **Baseline:** 23/57 (P5-01 landed); working tree shared with parallel lanes P2-01 / P4-01 / P4-03 (their files untouched).

## 1. Seam adjudication — why this is the narrowest valid seam

The task builds directly on the seam P5-01 froze: the same package (`internal/gitintegration`), the same validation pipeline, zero new authority. No open seam row remained to close; no Decision Lock needed changing.

**Ruling: integration branch creation is one new typed operation on the existing `gitintegration.Service` — `EnsureIntegrationBranch` — with both the branch name and the branch point derived deterministically from the accepted ChangeSet. It adds ref plumbing only where `gitwriter` has no API, and composes `gitwriter` for everything else.**

### Evidence read (in the mandated order)

1. Task contract `docs/tasks/v1.4.1/tasks/P5/P5-02_controlled_integration_branch_creation.md`; then `DECISION_LOCKS.md` (DL-03, DL-13; gitwriter lock discipline row), `PRD_AMENDMENTS.md`, `SOURCE_BASELINE.md` (no branch-naming or branch-ref guidance in either — naming is defined by this task).
2. Current source: `internal/gitintegration/service.go` + `service_test.go` (frozen seam, test conventions), `internal/gitwriter/writer.go` + `lock.go` (public API = `New`/`Head`/`Commit`; lock primitives unexported), `internal/changeset/types.go` + `identity.go` (ID = `sha256:<64 lowercase hex>`, deterministic).
3. Downstream task goals only: P5-03 (`safe_local_merge_integration.md` — "preserve audit refs": the branch *is* a durable named integration ref) and P5-04 (protected-branch policy: the `tally/` namespace makes ownership explicit).
4. External pack references used conceptually: **EXT-GIT-UPDATE-REF** (compare-and-swap ref update semantics — mapped onto `git update-ref --stdin`'s `create` primitive); **EXT-GIT-WORKTREE** (read for linked-worktree context; not needed — the operation is HEAD-independent and never checks out).

### Why no narrower seam exists

1. **`gitwriter` has no ref API and must not be extended by this task.** Its exported surface is `New`/`Head`/`Commit` (HEAD commits only); its lock primitives are unexported. Allowed paths forbid editing `internal/gitwriter/**`, and the L3 pack explicitly places this task's plumbing in gitintegration ("Core Implementation: Git command/plumbing isolated in gitintegration package"). Composing `gitwriter.New`/`Head` for repository validation keeps one typed entry for repo-ness and context handling.
2. **CAS at the ref level replaces a Tally-level lock, not weakens it.** `git update-ref --stdin` with the `create` primitive ("Create <ref> after verifying that it does not exist") is atomic under git's own ref lockfiles, so concurrent creators of the same deterministic branch cannot both win and neither can overwrite — the concurrency backstop is structural, and no second lock implementation is duplicated.
3. **No caller-supplied ref text exists anywhere.** Branch name = `refs/heads/tally/integration/<sha256 hex of ChangeSetID>` (colon-bearing `sha256:` prefix stripped; hex is ref-safe); branch point = the ChangeSet's own fixed `BaseRevision`, resolved once via `rev-parse --verify --quiet <rev>^{commit}` to an immutable OID (a movable expression like a branch name would be non-deterministic, and a `-`-prefixed value is rejected before it can be parsed as a git option). `project` is deliberately *not* embedded: ProjectID is arbitrary text (ref-format violations, sanitization collisions); the ID already encodes project/base/candidate/run. Nothing model-controlled can reach Git argv or stdin.
4. **Rejected alternative A — put `ExpectedBase` in the request and validate it against `cs.BaseRevision()`.** Strictly wider surface for the same guarantee: the only legal branch point *is* the accepted change's base, so it is derived, eliminating the mismatch error class entirely.
5. **Rejected alternative B — plain `git branch <name> <base>`.** Porcelain without compare-and-swap semantics; detecting "already exists" would require string-matching localized git output (forbidden). The plumbing `create` primitive decides by exit status plus a read-back of the ref value, never by message text.
6. **Rejected alternative C — skip the acceptance requirement since "it is only a ref".** Branch creation is step one of the controlled integration flow (Accept → integration), so it runs the exact frozen pipeline of `Integrate`: identified candidate (`ErrCandidateUnidentified`), complete + matching acceptance fact with non-empty basis (`ErrNotAccepted`/`ErrAcceptanceMismatch`), non-primary-worktree target (`ErrProtectedTarget`), real repository (`gitwriter.ErrNotRepository`).

### Resulting shape

- `Service.EnsureIntegrationBranch(ctx, IntegrationBranchRequest{ChangeSet, Acceptance, RepositoryPath}) (IntegrationBranchResult, error)`:
  1. pipeline as in (6) above;
  2. deterministic naming, format-guarded — an unmappable identity fails safe (`ErrCandidateUnidentified`), never "cleaned" into a ref name;
  3. base resolution in the target repo → `ErrBaseNotFound` when the base is absent (ctx cancellation is re-checked first and passed through unwrapped, never fabricated into a candidate failure);
  4. CAS create: success → `Created: true`; ref already at the same base → idempotent success, `Created: false`, zero mutation; ref elsewhere → `ErrIntegrationBranchExists` typed conflict. On a lost race the ref value is re-read and classified by value, not by git's message.
- New sentinels: `ErrIntegrationBranchExists`, `ErrBaseNotFound`. Errors reuse `gitwriter.GitError` for argv/output attachment; `runGit`/`sanitizedGitEnv` mirror gitwriter's discipline (structured argv, no shell, ctx kill, `GIT_*` stripped) — duplicated only because gitwriter's helpers are unexported.
- The operation never moves HEAD, never checks out, never creates a commit (commits remain `Integrate` via the gitwriter write boundary), and has no force/delete/reset path.

## 2. Changed files

- `internal/gitintegration/branch.go` (new): branch sentinels, `IntegrationBranchRequest`/`IntegrationBranchResult`, `EnsureIntegrationBranch`, deterministic naming helper, CAS ref helpers, git invocation helper.
- `internal/gitintegration/branch_test.go` (new): 7 tests over real temporary Git repositories (service_test.go / gitwriter test conventions).
- `internal/gitwriter/**`, `internal/changeset/**`, everything else: **unchanged** (composition only; no core edits were needed).

## 3. Semantic changes

A controlled integration branch is now reachable as one typed operation that always names and roots the branch deterministically from an accepted candidate, can never overwrite an existing branch (CAS create), treats base drift as a typed conflict, and never touches the owner primary worktree or HEAD. Acceptance criterion met: base drift is typed conflict; no silent branch overwrite.

## 4. Tests / commands executed

- `go test ./internal/gitintegration/...` — **PASS** (ok, 7.2s; 19 tests total = 12 from P5-01 + 7 new)
- `go test ./internal/gitintegration/ -run TestEnsureIntegrationBranch -v` — **PASS** (all 7 new tests listed individually above)
- `go test ./internal/gitwriter/... ./internal/changeset/...` — **PASS** (unchanged neighbors, no regressions)
- `go test ./test/arch/...` — **PASS** (relevant arch guards, dependency policy intact)
- `go vet ./internal/gitintegration/` — **PASS**
- `gofmt -l internal/gitintegration` — clean
- `git diff --check` — clean (exit 0; CRLF warnings emitted for other lanes' files only)
- CR check on both new files — 0 CRLF (LF per `.gitattributes`)

Negative/failure-path tests added: drifted existing branch (`ErrIntegrationBranchExists`, asserts the ref still points at the old value — no overwrite — and HEAD unmoved); missing base in target repo (`ErrBaseNotFound`, asserts no branch created); validation-pipeline table (nil changeset / zero acceptance / mismatched acceptance / missing repository path / protected primary worktree — each asserts no branch created and HEAD unmoved); non-repository target (`gitwriter.ErrNotRepository`); cancelled context (returns `context.Canceled` unwrapped, no branch created). Happy-path tests assert the exact deterministic ref name, branch point, `Created` flag, HEAD immobility, and clean work tree; idempotency test asserts the second call reports `Created: false` with zero mutation.

## 5. Baseline failures observed

None in owned or neighbor packages. Cross-lane noise: `git diff --check` printed CRLF warnings for `cmd/worker/*` (P4-01) and `web/templates/base.tmpl` (P2-01) — not owned by this task, not touched.

## 6. Remaining risks

- `runGit`/`sanitizedGitEnv` duplicate two small gitwriter internals (they are unexported). If a later P5 task (merge, P5-03) needs the same helpers, consider a small authorized gitwriter export/refactor in a task that owns `internal/gitwriter/**`.
- The concurrent "lost race" classification path (create fails → read-back → idempotent success) is structurally correct but not deterministically testable without injection hooks; it is covered by code review, not by a test.
- The identity-format guard in `integrationBranchRef` (non-`sha256:<64 hex>` IDs) is defensive and unreachable through `changeset.New`'s public API today, hence untested.
- Symlink aliasing of the primary worktree is not resolved — pre-existing condition of path-based guards (inherited unchanged from P5-01).

## 7. Decision Locks touched

None. No lock was edited, weakened, or routed around. DL-03 respected (no second Git-write authority for commits; ref CAS is the plumbing gitwriter does not expose, placed in the pack-designated package); DL-13 respected (scope ends at local integration; no remote/force/history-rewrite path exists in this change).

## 8. Commit hash

Not committed — orchestrator owns commits per task protocol.
