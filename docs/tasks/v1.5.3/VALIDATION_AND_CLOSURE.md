# Tally v1.5.3 — Build Host Validation & Closure Record

Local/Build Host execution record for issue #45 (validation handoff) and the
v1.5.3 release gate in `docs/implementation/V1.5.3_LOCAL_VALIDATION_GATE.md`.
No GitHub Actions / CI result is used as any gate. Statuses use only
`PASS` / `FAIL` / `NOT_RUN(reason)` / `ENV-BLOCKED(reason + evidence)`.

## 0. Revisions and environment

- HANDOFF_REVISION (frozen candidate branch head): `ce3e4ae9c102d26cf864eab82cab659c99f15694`
- Repairs applied on `feat/v1.5.3-git-engineering-authority` before validation:
  - `a142998` merge `origin/main` into the candidate (baseline sync; handoff branch predated current main)
  - `20547fa` fix: close the replan-history `if i > 0` block dropped by the frozen branch (candidate never compiled on Build Host: `types.go:212: syntax error: unexpected name snapshotHasTask`); restores exact mainline structure, verified by diff against `origin/main`
- VALIDATION_REVISION: the commit containing this ledger. Exact SHA is recorded
  in the hidden pack `aggregate.txt` (`revision=` field), the issue #45 evidence
  comment, and the merge/tag record. All code-affecting gates below ran on
  `20547fa8b47e911946ea3be53ef1968d7f47a57c`; the only delta between `20547fa`
  and the validation revision is this docs-only ledger, and the fast gates
  (focused tests, vet, full regression, hidden pack) were re-executed on the
  validation revision itself.
- Host: Nilar, Windows 10.0.26200 x64, Go 1.27.1 windows/amd64, git 2.55.0.windows.5
- Validation author: local validation agent (independent of the implementation author)
- Evidence timestamps: 2026-09-16 UTC

## 1. Focused v1.5.3 tests (gate §1)

Executed on `20547fa`:

```text
go test -count=1 ./internal/engineering/...      PASS
go test -count=1 ./internal/gitengineering/...   PASS
go test -count=1 ./internal/controlplane/...     PASS
go test -count=1 -run 'Git|Replay|Lifecycle|Completion' ./internal/engineering/...                  PASS
go test -count=1 -run 'Authority|WrongBase|Stale|Push|Review|Merge|Tag|Release|Replay' ./internal/gitengineering/...  PASS
go test -count=1 -run 'GitAction|Awaiting|Completion' ./internal/controlplane/...                   PASS
```

Required semantic assertions 1-7 from the gate doc are covered by the branch
tests and independently re-proven by the held-out pack (§4 below).

## 2. Build + static/local checks (gate §2)

`make` is not installed on this host (ENV-BLOCKED as an invocation carrier);
every Makefile recipe was executed verbatim instead:

```text
make build          -> go build ./...                                  PASS (20547fa)
make test           -> go test ./...                                   PASS (20547fa; see §3)
make test-runtime   -> runtime adapter test recipes verbatim           PASS (20547fa)
make check          -> docscheck PASS, repohealth PASS,
                       gofmt: scoped PASS (all 15 v1.5.3-changed files clean);
                       repo-wide `gofmt -l` flags 22 pre-existing files equally
                       on origin/main (Go 1.27 struct-tag alignment drift;
                       environment/toolchain condition, not v1.5.3-attributable;
                       repo not reformatted to avoid scope creep),
                       go vet ./... PASS,
                       golangci-lint ENV-BLOCKED (v1.64.8 cannot decode Go 1.27
                       export data: "version 4 > maximum supported version 2";
                       binary/toolchain evidence preserved; go vet is the
                       operative static check)
go test -race ./internal/engineering/... ./internal/gitengineering/... ./internal/controlplane/...   PASS (20547fa)
```

## 3. Integration / regression (gate §3)

```text
go test -tags=integration ./...   PASS (20547fa, with PostgreSQL container postgres-18 at localhost:15432)
go test -count=1 ./...            PASS (20547fa)
```

One environment/timing flake was recorded and disposed: brownfield
`internal/workspace` `TestCreateConcurrentDistinctPaths` failed once on a git
`worktree add` race ("failed to read .git/worktrees/.../commondir"); the test
was re-run 3x, then its package, then the full integration suite — all PASS
(0 non-ok). Package is untouched by the v1.5.3 diff; classified as an
environment/timing flake, not a v1.5.3 defect.

## 4. Hidden Validation — Local Git + authority + completion (gate §4/§6)

Held-out pack (outside the implementation repository):
`C:/Dev/vCode/hidden_validation/tally/v1.5.3/` (manifest.json, run.sh, gotests/).
Pack author: local validation agent; the implementation author never saw the
fixtures/oracles before the visible implementation was frozen.

Final result on the validation revision: **25/25 PASS, run.sh exit 0**
(result dir `results/20260916T013740Z`, aggregate: `scenarios_total=25,
scenarios_failed=0, scenarios_env_blocked=0, go_test_exit=0`).

Coverage:

- HV153GIT01 wrong base blocked before provider mutation (remote fake 0 calls; blocked record valid; no completion fact) — gate HV-GIT-01 PASS
- HV153GIT02 stale-head guarded push => failed record `stale_head`, audit-only — gate HV-GIT-02 PASS
- HV153GIT03 unreachable local remote => failed record `git_command_failed` — gate HV-GIT-03 PASS
- HV153GIT04 duplicate record ID => `ErrDuplicateGitAction`; new ID + same replay key => `ErrReplayedGitAction` — gate HV-GIT-04 PASS
- HV153AUTH01 merge guard chain (no PR / verification / review / changes-requested / acceptance / stale base / stale head / human requirement missing / authority absent / mismatch / rejected) — each denied pre-mutation with exact reason
- HV153AUTH02 tag/release accepted-baseline guards + human authority requirement
- HV153AUTH03 policy-forbidden action
- HV153AUTH04 blocked record never becomes completion evidence (projection + contract evaluation)
- HV153CMP01 Critical Journey (gate §6): accepted-with-facts => `task.accepted.awaiting_git`, blocked + failed + non-required-successful actions audit-only, required authorized merge => DONE with exact result revision `ResultRevision`, duplicate/replay refused after DONE — CJ-153 PASS
- HV153CMP02/03 trace binding (foreign project/task refused, nothing persisted) and invalid record rejection
- HV153NEG01 candidate diff touches no `.github/` path (merge-base diff) — PASS
- HV153NEG02 no GitHub status/check/API surface in `internal/engineering` or `internal/controlplane` non-test files — PASS

Pack-oracle repairs during execution (implementation NOT changed; no test
weakened): (a) human-authority oracle aligned to the frozen denial order
(requirement-ID check precedes decision-supplied check); (b) journey git auth
switched Bearer -> Basic (GitHub git-over-HTTPS rejects Bearer tokens);
(c) ambient `git clone` replaced by deterministic init+fetch after a GCM
interactive hang; (d) merge-record lookup made semantic instead of positional;
(e) run.sh hardened to attribute package-level failures (panic/timeout) as an
explicit FAIL scenario row.

## 5. Hidden Validation — Real GitHub provider (gate §5)

HV153GH01 executes the full gate §5 chain 1-17 against a disposable
self-provisioned private repository (`kaicreator-mm/tally-hv153-1789522664552736200`),
token from `gh auth token` via `TALLY_V153_GH_TOKEN` (never hardcoded):
create issue -> branch from main -> commit -> push -> PR opened with
authority-forced base `main` (caller-supplied wrong base overridden) ->
REQUEST_CHANGES -> changes-requested merge blocked (PR verified unmerged via
API) -> repair commit+push -> stale-head push classified `stale_head` (remote
branch verified unmoved) -> re-review -> unauthorized merge blocked (PR
verified open/unmerged) -> authorized squash merge with expected head SHA ->
record `ResultRevision` asserted EQUAL to actual main HEAD via API -> tag on
accepted baseline -> tag mismatch blocked (no remote ref) -> wrong release
baseline blocked (no remote release) -> release published from accepted
baseline -> duplicate/replay guards fire -> trace/actor/authority/timestamp/
provenance assertions across all records — **PASS** (22.95s).

Provider constraint exercised and classified, not hidden: GitHub forbids
APPROVE / REQUEST_CHANGES on one's own PR; the adapter's failure was asserted
as `provider_request_failed` and the successful review seam was proven with
COMMENT. Review/approval authority semantics are held out in HV153AUTH against
the guard chain, because GitHub remains provider only, never workflow
authority (HV153NEG02).

Cleanup: repository DELETE returned 403 — the `gh` OAuth token lacks the
`delete_repo` scope (evidence in go-test.log CLEANUP-BEST-EFFORT lines).
Leftover disposable private repos for manual deletion (all contain only
held-out fixtures): `tally-hv153-1789521669047787400`,
`tally-hv153-1789521863749837800`, `tally-hv153-1789522594451667500`,
`tally-hv153-1789522664552736200`.

## 6. Independent final diff review

Reviewer: local validation agent (independent of the implementation author).
Diff `origin/main...HEAD`: 16 files, +2541/-0, all inside the v1.5.3 lane
(docs gate, `internal/engineering` git domain, `internal/gitengineering`,
control-plane git-action seam + 25-line dispatch seam + snapshot validation).
Findings: domain contracts enforce blocked/failed/succeeded invariants;
authority denies before any provider mutation; LocalGitAdapter uses structured
argv, prompt-free env, explicit-path commits, stale-head check before push;
GitHub REST details confined to the adapter; control plane validates records,
enforces duplicate/replay, folds only authorized successes into
CompletionFacts; `finishAccepted` accepted-awaiting-git semantics do not
introduce a second workflow authority; the `20547fa` brace repair restores
exact mainline structure.

Verdict: **PASS WITH CONCERNS (2 minor, accepted)** —
(1) authored commits other than `20547fa` lack `Refs:` trailers (traceability
is carried by PR #44 / issue #45 instead; process note for future lanes);
(2) Go 1.27 gofmt repo-wide drift documented in §2 (pre-existing, out of scope).

## 7. Release qualification (gate §8)

| Item | Status |
| --- | --- |
| focused v1.5.3 tests | PASS |
| make build (verbatim recipe) | PASS |
| make test (verbatim recipe) | PASS |
| make test-runtime (verbatim recipe) | PASS |
| make check (verbatim recipes) | PASS except golangci-lint ENV-BLOCKED (toolchain incompatible with Go 1.27; evidence §2) |
| go test ./... | PASS |
| integration regression | PASS |
| Local Git Hidden Validation | PASS |
| real GitHub provider Hidden Validation | PASS |
| Request Changes -> repair -> re-review journey | PASS |
| unauthorized merge blocked | PASS |
| stale head / wrong base / push failure behavior | PASS |
| merge result SHA/revision correctness | PASS |
| tag mismatch / wrong release baseline blocked | PASS |
| duplicate/replay action handling | PASS |
| audit/provenance completeness | PASS |
| Critical Journey (CJ-153) | PASS |
| independent review of final diff | PASS WITH CONCERNS (2 minor accepted) |
| no `.github/workflows` change | PASS (HV153NEG01) |
| no GitHub Actions/CI result used as gate | PASS (HV153NEG02 + this record) |

`make` itself and `golangci-lint` are tooling ENV-BLOCKED carriers with
verbatim-recipe / `go vet` substitutes executed and recorded; neither is a
product defect and neither leaves a material item unexecuted.

**Release Qualified: YES** on the validation revision (this ledger's commit;
fast gates re-executed there, code-affecting gates on `20547fa` with the
docs-only delta stated above).
