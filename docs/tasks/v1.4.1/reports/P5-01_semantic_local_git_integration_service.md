# P5-01 Execution Report — Semantic local Git integration service

- **Status:** DONE
- **Date:** 2026-09-12
- **Baseline:** 22/57 (P0/P1 complete); working tree shared with parallel lanes P2-01 / P4-01 / P4-03 (their files untouched).

## 1. Seam adjudication (SPIKE_REQUIRED → frozen by this task)

DECISION_LOCKS.md "Open seams" row `L3 local Git integration service API` is owned and now closed by this task.

**Ruling: the local Git integration service is a new semantic facade package `internal/gitintegration` that composes the existing Git authorities — `internal/gitwriter` (single Git-write boundary, expected-base CAS, cross-process lock, atomic rollback) and `internal/changeset` (immutable accepted candidate snapshot). It adds zero Git plumbing and zero parallel authority.**

### Why this is the narrowest valid seam

1. **gitwriter already implements the CAS/lock discipline the reference pack points at.** EXT-GIT-UPDATE-REF's compare-and-swap ref update maps exactly onto `Writer.Commit`'s expected-base validation executed under the in-process per-repo mutex plus the cross-process lock file (`internal/gitwriter/writer.go` / `lock.go`). Re-implementing update-ref plumbing here would duplicate GitWriter and violate DL-03's spirit; composing it keeps one Git-write boundary.
2. **The semantic layer's job is typed invariants, not Git mechanics.** What GitWriter cannot express — "candidate must be identified", "candidate must carry a complete acceptance fact matching its identity", "integration target must never be the owner primary worktree", "empty expected base is fail-safe rejected" — is exactly what `gitintegration` adds as preflight validation before any GitWriter call.
3. **Rejected alternative A — extend `gitwriter` with acceptance awareness.** Wrong direction: gitwriter is a generic mechanical write boundary (harvest, knowledge, requirements all use it); acceptance/candidate semantics would leak Closed-Loop vocabulary into it and broaden its blast radius.
4. **Rejected alternative B — raw `git update-ref` / `git merge` calls in the service.** Re-exposes raw Git argv one layer up and loses the existing lock/rollback guarantees; also the model-facing surface must stay semantic-only.
5. **Rejected alternative C — store an `accepted` boolean on ChangeSet.** Violates ChangeSet immutability; acceptance is an external fact from the Closed Loop accept authority. Instead the service consumes a typed `AcceptanceFact` value (ChangeSetID + AcceptedAt + required Basis) supplied by the acceptance authority, and fails safe when it is missing, incomplete, or mismatched.

### Resulting shape

- `gitintegration.New(Config{PrimaryWorktree, Signature})` — constructor **requires** the owner primary worktree path; without it the service cannot enforce the trust invariant and refuses to construct (fail-safe).
- `gitintegration.Service.Integrate(ctx, IntegrateRequest{ChangeSet, Acceptance, RepositoryPath, ExpectedBase}) (IntegrationResult, error)` — the single semantic operation this task freezes:
  1. candidate identified (non-nil ChangeSet, non-empty ID and base) else `ErrCandidateUnidentified`;
  2. acceptance complete (ID + timestamp + **non-empty Basis** — every ACCEPTABLE result carries a basis) and matching the candidate identity, else `ErrNotAccepted` / `ErrAcceptanceMismatch`;
  3. applicable content present (full-content file model) else `ErrNoApplicableContent`;
  4. target is not the primary worktree (`ErrProtectedTarget`, Windows case-insensitive path compare) and expected base is explicit and non-empty (`ErrNoExpectedBase` — an unborn/unknown integration branch is never a legal integration start);
  5. read-only CAS preflight via `Writer.Head` → `ErrStaleBase` (wrapping `gitwriter.ErrBaseRevision`) when the target moved; conflict is an expected typed outcome;
  6. commit through `gitwriter.Commit` with deterministic semantic message `tally: integrate changeset <id>`; a base move between preflight and commit is still caught by gitwriter's in-lock revalidation and re-typed as `ErrStaleBase`; `gitwriter.ErrNoChanges` and `ErrNotRepository` pass through typed.
- **P5-02/P5-03 seam fit:** branch creation and merge extend this package on top of `Integrate` (they own branch/ref semantics); the request/result surface and the validation pipeline above are the frozen API they build against.

## 2. Changed files

- `internal/gitintegration/service.go` (new): package doc, sentinels, `AcceptanceFact`, `Config`, `Service`, `IntegrateRequest`/`IntegrationResult`, `Integrate`, primary-worktree guard.
- `internal/gitintegration/service_test.go` (new): 12 tests, real temporary Git repositories (gitwriter test conventions), happy path + all failure paths.
- `internal/gitwriter/**`, `internal/changeset/**`: **unchanged** (composition only; no core edits were needed).

## 3. Semantic changes

Controlled local integration is now reachable as one typed operation that cannot integrate an unaccepted, unidentified, basis-less, or base-stale candidate, and can never target the owner primary worktree. Decision Locks were not touched or bypassed (DL-03, DL-13 respected; ChangeSet immutability and supersession append-only semantics untouched).

## 4. Tests / commands executed

- `go test ./internal/gitintegration/...` — **PASS** (ok, 5.0s, 12 tests)
- `go test ./internal/gitwriter/... ./internal/changeset/...` — **PASS** (ok, unchanged neighbors, no regressions)
- `go vet ./internal/gitintegration/` — **PASS**
- `gofmt -l internal/gitintegration` — clean
- `git diff --check` — clean (new files only, LF per `.gitattributes`)

Negative/failure-path tests added: nil/unidentified candidate; zero/incomplete acceptance fact (4 variants); acceptance-identity mismatch; stale base (asserts `ErrStaleBase` wraps `gitwriter.ErrBaseRevision`); empty expected base on born and unborn targets; protected primary worktree; no applicable content; missing repository path; non-repository target (asserts no files written); no-op integration surfaces `gitwriter.ErrNoChanges` with a clean tree. Every rejection test also asserts HEAD did not move and the work tree stayed clean.

## 5. Baseline failures observed

None in owned or neighbor packages.

## 6. Remaining risks

- `AcceptanceFact` is currently supplied by the caller; wiring it from the real Closed Loop acceptance record is a later integration task (P5-02/P5-03 lane / worker wiring). Until then, production callers must construct facts only from the acceptance authority's records — the service validates shape and identity match, not provenance.
- Primary-worktree protection is path-based; symlink aliasing of the primary worktree is not resolved (pre-existing condition of path-based guards in this codebase).

## 7. Decision Locks touched

None. No lock was edited, weakened, or routed around; the open seam `L3 local Git integration service API` is closed by this task per DECISION_LOCKS §"Open seams".

## 8. Commit hash

Not committed — orchestrator owns commits per task protocol.
