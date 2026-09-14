# P1 Gate Report — P1-52 (P1 Vertical Trust Gate review)

- Date: 2026-09-06
- Auditor: P1-52 gate agent (read/verify/report only; no code changes)
- Base: `10e3448` (master, "P1-51: P1 metric baseline report generation") — branch `task/P1-52`, worktree `.worktrees/P1-52`; no commit in this range is authored by this gate except the report file itself
- Environment: Windows, Git Bash, Go 1.27.1 (`go1.27.1 windows/amd64`), git CLI; local Docker Postgres at `postgres://postgres:postgres@localhost:5432/tally?sslmode=disable` (reachable; per-test schemas self-migrate)
- Authority: `docs/tasks/v1.3/P1/P1-52.md`; PRD §36.3 (P1 Trust 验收), §37 (P1); `docs/tasks/v1.3/GATE_REVIEW_PROMPT.md`; `docs/tasks/v1.3/reports/P0_GATE.md`; `docs/tasks/v1.3/reports/P1_METRIC_BASELINE.md`

## Final gate verdict: PASS

The P1 gate asks one question: **is the Vertical Trust Loop trustworthy and measurable?** It does not ask whether Tally is faster/cheaper than bare Codex (PRD §36.3 final paragraph; Decision Lock 16). On the evidence re-executed and re-read in this audit, every task acceptance criterion (AC1–AC8) and every §36.3 Trust Gate item is satisfied by P1-scope code and tests, with no P2+ scope required or smuggled. The one scheduled command (`make check`) and one lint tool are unavailable in this environment; both are recorded exactly below, and every command they would have wrapped was executed individually and passed.

## Environment blockers (recorded exactly)

1. `make check` (the task's Required validation): **NOT_RUN(reason: `make` is not installed in this Git Bash environment; `which make` → not found, exit 1)**. All constituent commands were run individually instead (next section).
2. `golangci-lint run`: **NOT_RUN as a gate signal (toolchain incompatible with Go 1.27 export data)**. Reproduced firsthand in this gate: golangci-lint v1.64.8 (built with go1.27.1) fails with `can't run linter goanalysis_metalinter: buildssa: failed to load package context: could not load export data: internal error in importing "context" (cannot decode "context", export data version 4 is greater than maximum supported version 2)`. `go vet ./...` (PASS) and the LF-normalized `gofmt` check (PASS) were performed as the P0-gate-accepted equivalents (see `P0_GATE.md`).
3. Raw `gofmt -l .` on this Windows checkout flags files purely from CRLF working-tree artifacts (known from P0 gate). Equivalent normalized check executed instead: `git -c core.autocrlf=false -c core.eol=lf checkout-index --prefix=/tmp/gate-check/ -a && cd /tmp/gate-check && gofmt -l .` → **0 files flagged**.
4. `TestLiveCodexSurfaceAuthGate` (`runtimes/codex/runtime_test.go:1177`) is skip-gated on `TALLY_CODEX_LIVE`; the variable is not set here, so the live codex surface was not exercised in this gate run. This is by design: live model turns require operator `codex login`. The live-surface evidence of record is the P1-36 probe (`docs/tasks/v1.3/runtime/CODEX_RUNTIME_PROBE.md`), which captured the real binary's CLI surface and the 401 auth-gate behavior verbatim.

## Validation commands actually executed (this gate, fresh `-count=1` runs)

| Command | Result |
| --- | --- |
| `go build ./...` | PASS (exit 0) |
| `go vet ./...` | PASS (exit 0) |
| `go test -count=1 ./test/arch/... ./internal/... ./cmd/... ./runtimes/... ./test/runtimeconformance/...` | PASS (exit 0; every package `ok`, incl. `internal/verification`, `internal/evidence`, `internal/taskstate`, `internal/review`, `internal/runstore` 26.99s, `internal/measurement` 20.1s, `internal/workflow` 18.1s, `cmd/worker` 14.4s, `runtimes/codex` 3.9s) |
| `DATABASE_URL=… go test -count=1 -tags=integration ./...` (run twice: once filtered, once with full per-package capture) | PASS (exit 0 both; 44 packages `ok`, 0 FAIL; `tally/test/e2e` ok 79.8s) |
| `DATABASE_URL=… go test -count=1 -tags=integration -v -run 'TestV13' ./test/e2e/` | PASS — **17/17 E2E gate tests, none skipped** (list under AC6) |
| Normalized `gofmt -l .` (LF checkout-index tree) | PASS (0 files) |
| `golangci-lint run ./internal/holdout/` (probe of toolchain) | NOT_RUN as gate signal — export-data incompatibility quoted above |
| `make check` | NOT_RUN(`make` not installed; components run individually) |

The database-backed runs are real: E2E tests skip (not fail) when Postgres is unreachable, and none skipped — all 17 ran against live PostgreSQL with per-test schemas and self-migrated goose migrations.

## P1 execution audit — all 51 implementation tasks merged, in scope, no P2+ smuggled

Method: `git log --no-merges --name-only` over full history of `10e3448`, every commit mapped to its task, every changed file checked against the task's Allowed paths in `docs/tasks/v1.3/TASK_MANIFEST.json`.

- **Coverage: 51/51** non-gate P1 tasks (P1-01..P1-51) have exactly one non-merge implementation commit each, plus one gate commit per E2E/metric gate (P1-47..P1-50, P1-51) and this report (P1-52). No P1 task in the manifest is unimplemented or extra (52 P1 entries incl. P1-52).
- **Scope compliance:** every task's changed files match its manifest Allowed paths, with exactly two exceptions, both **declared integrator amendments** openly labeled in commit messages (the pattern the P0 gate already adjudicated as non-silent):
  - `2005a44` "integrate: resolve P1-27/P1-29 test import cycle — relocate ContextPack path-leak e2e proof from holdout to verification package (assertions unchanged)";
  - `4ae1b4b` "integrate: use external verification_test package for relocated path-leak e2e (breaks test import chain)".
  Both are test relocations only (−73/+90 lines across `internal/holdout/filesystem_test.go` → `internal/verification/holdout_path_leak_test.go`); assertions unchanged per commit message, verified by the green suites. No silent leaf-scope violation found. sqlc-generated files were regenerated only by the dedicated owner P1-31 (11 files, one commit). Hotspots respected: `cmd/worker/main.go` untouched by v13 work (vertical path lives in new `cmd/worker/v13.go`, owner P1-43); `internal/web/handlers/server.go` touched only by P1-44 (11 lines, registration).
- **No P2+ scope smuggled** (PRD §36.2, Decision Locks 12/14/15/16):
  - `runtimes/` contains exactly `fake` and `codex` — no Pi/OpenCode adapter (P3 scope).
  - No DAG scheduler / workflow DSL / autopilot: `internal/workflow` is the single-Task sequential kernel (P1-39); `cmd/worker/v13.go` states and implements "一次 job 恰好执行一个 standard TaskContract"; no such package or loop exists anywhere in `internal/`/`cmd/`.
  - No Saved Views, no Triage/Fast Path product, no holdout Recovery Ladder automation: keyword scan over all Go hits only (a) the §38.3 lifecycle milestone constant `MilestoneTriageCompleted` (recording, per PRD — not the P2 Triage product), (b) comments explicitly deferring these to P2, e.g. `v13_holdout_fail_test.go` "Recovery Ladder 自动化属 P2" and P1-48's AC3 proving **no** automatic second run after a block.
  - New migrations 0018–0021 are exactly the P1 SQL contracts (task/verification derived index, codegen/changeset, verification/evidence/exposure, approval/measurement) — no P2 fact model.

## AC1 — Primary worktree contamination = 0: PASS

- Structural: `internal/workspace/manager.go` refuses to place disposable worktrees inside the owner checkout and refuses to destroy the owner primary worktree; `internal/workspace/codegen.go` makes the owner worktree unaddressable as a run-sources target; `internal/workspace/cache.go` rejects dependency-cache roots inside or containing the owner checkout.
- End-to-end: `TestV13StandardTaskRevisionsTraceToRunChangeSetEvidence` family — after a full codegen + final-verification pipeline through real Git worktrees and real Postgres, `p47RunScenario` fails the test if `git status --porcelain` on the owner repo is non-empty ("Trust Gate 前提:整条流水线在 owner 主 worktree 零写入"); `TestV13HeldOutFailureStaysControlPlaneOnly` asserts the same plus holdout-store content-digest invariance. The production path (`cmd/worker/v13.go`) enforces the same via `workspace.New` failing closed on an owner-inside root.
- No contamination incident appears in any P1 task report or in this gate's runs.

## AC2 — Protected evidence silent modification = 0: PASS

- Database-side immutability is enforced by statement-level triggers (migrations 0019/0020) and asserted by rejection paths in E2E:
  - `TestV13HeldOutFailureEvidenceAppendOnly`: `UPDATE verification_evidence SET verdict='pass'`, `SET visibility_policy='visible'`, `DELETE`, `UPDATE verification_run SET status='pass'`, `DELETE` — all rejected; duplicate `fact_id` re-record is an error, not an update; the FAIL fact remains valid+applicable afterwards.
  - `TestV13ExposureFactAppendOnly`: exposure `UPDATE`/`DELETE` rejected (including blanket `DELETE`), fact read back byte-faithful through three projections; after exposure, the evidence row's `visibility_policy` stays `held_out` and the frozen contract is unchanged (AC2 of that file).
  - `TestV13SupersessionDoesNotMutatePriorSnapshot`: raw column-wise comparison of the superseded candidate's row before/after the supersession fact — unchanged; `changeset` UPDATE/DELETE and fact UPDATE/DELETE all rejected.
- Domain-side: holdout content is immutable in the Tally-controlled store (`Stat` digest re-check in P1-48); arch test INV-VER-03 (`test/arch/v13_verification_test.go`) pins "protected verification evidence is immutable to the authorized implementation run"; arch INV-VER-02 pins the frozen contract as read-only to implementation authority.
- No silent modification path was found; every mutation attempt in tests fails loudly.

## AC3 — Known false VERIFIED = 0: PASS

- `TestV13HeldOutFailureBlocksVerified`: with the visible surface fully green and only the held-out binding failing, the final verdict is `needs_review` — never verified; the `verification_run` row maps needs_review to `error` ("绝不是 pass"), so no pass fact exists that could be read as VERIFIED; functional coverage reports `GapBlockingFailure` with `Blocking=[HOLD-1]` while `SatisfiedBy=[VIS-1]` — a passing visible binding cannot mask an applicable FAIL (§8.8.1).
- `TestV13DerivedStateReachesVerifiedOnlyAfterEvidence`: state reaches VERIFIED only after functional + independence coverage bound to the *current* contract revision and the verification-run fact; coverage reports bound to a stale revision cannot produce VERIFIED (§8.8.2).
- Authority: arch INV-AUTH-01/02 — agent-channel lifecycle claims (e.g. self-declared merge) and agent-channel approvals are refused as facts, listed in auditable `IgnoredClaims`, and cannot drive state; `evidence.Confirm` requires a fact-capable independent recorder; structure test forbids `llm` imports on the fact/state/verification/contract/review surfaces.
- Anti-falsification details: `TestV13HeldOutFailureDerivesNonVerifiedTaskState` derives NEEDS_REVIEW with gaps traceable to the exact FAIL fact; duplicate fact-id overwrite attempt leaves the FAIL fact intact.

## AC4 — Traceability and hunk accounting = 100% for P1 scenarios: PASS

- Contract→Run→ChangeSet→Evidence: `TestV13StandardTaskRevisionsTraceToRunChangeSetEvidence` asserts `codegen_run.task_revision = TaskContract hash`, `verification_revision = frozen hash`, `context_pack_hash = ContextPack hash`; ChangeSet `ProducerRun()` round-trips to the stored Run primary key (`TestV13ChangeSetProducerRunReferencesStoredRun`); `verification_run` binds exact contract revision+hash and candidate identity; each evidence row's five-dimension scope (project/task/acceptance/contract/changeset) is asserted exact — in SQL (`count(*) … visibility_policy='held_out' AND verdict='pass'`), not just in memory.
- No cross-snapshot inheritance: `TestV13VerificationBoundToExactSnapshot` — a scope differing only in the ChangeSet dimension yields zero hits; superseded candidate A keeps its own immutable verification history; rebinding A's run to B via UPDATE is trigger-rejected.
- Hunk accounting: `internal/review/hunks.go` builds the Deterministic Hunk Inventory with fail-closed parsing ("解析失败一律报错,绝不静默丢弃条目"; added/deleted/renamed/binary all represented), content-derived stable hunk IDs, deterministic ordering (INV-STATE-01 rebuildable projection). `internal/review/accounting.go` partitions every entry, exposes `UnattributedCount`/`UnattributedEntries`, and `ApproveBlocked()` blocks on unattributed changes, unmet independence, and protected-evidence touches (INV-CHANGE-01/§19.2). The review assembly chain is exercised end-to-end in `TestV13ReviewShowsExposureGap` (same `InventoryFromDiff → RulesFor → BuildAccounting` chain as the review page).

## AC5 — Fake/Codex conformance evidence recorded: PASS (with recorded limits)

- Harness: `test/runtimeconformance` (P1-25) implements the PRD §35 checklist — 10 general scenarios (start/cwd/execute/result/events/cancel/crash/workspace/usage/capability_truth) plus capability-gated scenarios incl. `cap:session.isolation` (Session Isolation Canary), each with timeouts so a broken runtime fails rather than hangs.
- FakeRuntime: `TestFakeRuntimePassesSessionIsolationCanary` (`runtimes/fake/isolation_test.go`) — the canary passes on the reference adapter; FakeRuntime is the default runtime of the proven vertical path (all four E2E gates run on it).
- CodexRuntime: `TestCodexRuntimeConformance` runs the full harness against the adapter driving a substitute CLI binary compiled in-test that reproduces the **probed** surface vocabulary (thread.started/turn.started/item.completed/error JSONL, exec/resume/fork, `--version`) — protocol conformance without inventing flags. Capability declaration truthfulness is pinned (`TestDescriptorDeclaresExactlyProbedCapabilities`); the adapter refuses the `-a/--ask-for-approval` flag the live binary rejects (`runtime_test.go:640-647`, comment at `runtimes/codex/process.go:191`) — probe-doc §2.1 drift resolved in favor of the live surface (Decision Lock 13).
- Live-surface evidence of record: `docs/tasks/v1.3/runtime/CODEX_RUNTIME_PROBE.md` (P1-36) — every flag/verbatim from the installed `codex-cli 0.153.0`; live `exec --json` emits structured events then fails with 401 Unauthorized absent credentials. Recorded limit (not a gate failure): real model turns require operator `codex login`; the tested live behavior is the auth gate (`TestLiveCodexSurfaceAuthGate`, skip-gated on `TALLY_CODEX_LIVE`, not set in this environment).

## AC6 — Required holdout/exposure scenarios executed: PASS

All four gate scenarios were executed in this gate run against live Postgres (`go test -count=1 -tags=integration -v -run 'TestV13' ./test/e2e/` — 17/17 PASS, 0 skip):

| Scenario (PRD §36.3 item) | Tests | Result |
| --- | --- | --- |
| 至少一个真实 `standard + held_out` Task 端到端 (P1-47) | `TestV13StandardTaskRevisionsTraceToRunChangeSetEvidence`, `TestV13ImplementationContextCannotResolveHeldOutContent`, `TestV13VisibleIncrementalCheckRuns`, `TestV13HeldOutFinalPassContributesInformationIndependence`, `TestV13DerivedStateReachesVerifiedOnlyAfterEvidence` | PASS ×5 |
| 至少一次构造 holdout failure 阻断 VERIFIED (P1-48) | `TestV13HeldOutFailureBlocksVerified`, `TestV13HeldOutFailureStaysControlPlaneOnly`, `TestV13HeldOutFailureEvidenceAppendOnly`, `TestV13HeldOutFailureDerivesNonVerifiedTaskState` | PASS ×4 |
| 至少一次 exposure fact 使 effective information independence 变为 compromised 且 Review 可见 (P1-49) | `TestV13ExposureFactAppendOnly`, `TestV13ExposureFlipsInformationIndependence`, `TestV13ReviewShowsExposureGap` | PASS ×3 |
| Immutable ChangeSet + supersession (P1-50) | `TestV13RunWithoutCandidateCreatesNoChangeSet`, `TestV13CandidatesAreDistinctImmutableRecords`, `TestV13SupersessionDoesNotMutatePriorSnapshot`, `TestV13VerificationBoundToExactSnapshot`, `TestV13ChangeSetProducerRunReferencesStoredRun` | PASS ×5 |

Isolation details proven inside them: held-out content absent from ContextPack serialization, refs lexically path-free and unresolvable in the implementation workspace, redacted implementation-side signal allowlist-equal (no ref/digest/verdict/binding id), no `evidence_exposure` rows in the clean scenario; exposure anchored on lineage-root candidate A flips information independence for current candidate B via lineage expansion, and the review accounting shows exactly one `independence_gap` blocker with `ApproveBlocked()` (strict stops at "unmet and unwaived", never presented as satisfied).

## AC7 — Human/cost/latency baseline exists: PASS (capability proven; data honestly INSUFFICIENT)

- Fact paths exist and are exercised: `human_effort` / `llm_usage` append-only tables (migration 0021) written at call/effort time by `internal/measurement` (`Recorder`, P1-41); phase spans and runtime usage recorded per task; the worker vertical path and the E2E gates write through these paths; `cost_plane` is a call-time fact field (PRD §38.4).
- The baseline report exists: `internal/measurement/report.go` `BuildBaselineReport` (P1-51) produces every §38.2/§38.4/§38.5 readout family — human effort by kind, control/execution-plane cost and `control_plane_llm_cost_ratio`, ten lifecycle phase latencies, idea→verified latency, qualification-strength / authoring-mode / visibility distributions and their cross, acceptance-level planned vs effective independence with source-vs-information gap attribution, and first-pass `visible_only` vs `with_holdout` splits.
- Honesty: per `P1_METRIC_BASELINE.md` §3 — no accumulated persisted samples exist yet (all scenario data lives in ephemeral per-test schemas by design), so **every readout is marked insufficient (`Sufficient=false`, `MinSufficientSamples=2`), none reported as `0` or as "pass"**, and no readiness/value/superiority threshold appears anywhere in the report (Decision Lock 16; §36.3 does not preset ratios or require beating bare Codex). This is exactly what P1 should honestly produce: the measurement capability is proven, the accumulated numbers await real usage.
- 满足 §36.3 的对应项:"first-pass validation baseline 可产出;qualification strength、authoring mode、visibility、source/information independence 分布可产出;human/control/cost/latency 自动记录" — 可产出/自动记录 (capability) is what P1 requires and what is proven; the "INSUFFICIENT data" status is the honest current value of those readouts, not a missing capability.

## AC8 — No P2 scope is required for PASS: PASS

Every §36.3 Trust Gate item is satisfied by the P1-scope deliverables audited above: zero worktree contamination, zero silent protected-evidence modification, zero scope violations escaping the validator (TaskContract scope rules P1-14 + deterministic hunk inventory + unattributed/protected-touch approve blockers), zero known false VERIFIED, 100% traceability and hunk accounting in the gate scenarios, Session Isolation Canary passing on FakeRuntime (and CodexRuntime at protocol level against the probed surface), one real `standard + held_out` E2E, one constructed holdout failure blocking VERIFIED, one exposure fact compromising effective information independence visible in review accounting, and the producible measurement baseline. No DAG autopilot, no Recovery Ladder, no Saved Views/Triage/Fast Path, no Pi/OpenCode production adapter, no Runtime Router — none required by the gate, none present.

## Coordinator notes — re-verified by this gate (not trusted, reproduced)

1. **CodexRuntime live turns need operator login** — verified: probe §3 records live 401; `TestLiveCodexSurfaceAuthGate` is `TALLY_CODEX_LIVE`-gated and treats 401-family summaries as the expected no-credential `RunFailed` claim; FakeRuntime proves all model-independent paths (all four E2E gates run on it). Judged a recorded environment limit, not a gate hole: P1 requires a probed real surface, not operated credentials.
2. **Probe doc §2.1 drift on `-a` flag** — verified: §2.1 transcribed `-a, --ask-for-approval` from `--help`; the live binary rejects it; the adapter documents the drift and refuses the flag, with a test pinning argv (`runtime_test.go:640-647`). Adapter follows the live surface (Decision Lock 13). Doc drift is cosmetic; no action required for this gate.
3. **Production review page projects exposures per-candidate only** — verified at `internal/web/handlers/v13_changes.go`: `ExposuresForChangeSet(...)` collects only exposures anchored on the displayed candidate, and `deriveV13Coverage` passes a single-element `Lineage` (the candidate itself). An exposure anchored on an *ancestor* candidate is therefore not visible on the current candidate's production review page, although the derivation + review accounting handle lineage expansion when the caller supplies it — proven by E2E P1-49 (`Lineage: [B, A]` → flip + review gap). Candidate follow-up (wiring the supersession lineage into the review projection), not a trust hole: the derivation is conservative, and no path presents a compromised candidate as covered.
4. **Metric baseline readouts are INSUFFICIENT** — verified: `P1_METRIC_BASELINE.md` §3 states it exactly; `BuildBaselineReport` marks `Sufficient=false` below 2 samples; no `0`-substitution or pass/fail framing exists in the report code. Honest; matches AC7.
5. **Integrator amendments all declared in commit messages** — verified: full-history sweep found exactly the declared set (`budget.go`, `diagnose.go` under P0-13 `2af3195`; docs path relocation; P1 test relocations `2005a44`/`4ae1b4b`); no undeclared cross-boundary leaf edits in the P1 range beyond those two openly labeled test relocations.

## Gate disposition

- **Verdict: PASS.** P1 exits as a trustworthy, traceable, fully-reviewable, measurable single-Task vertical loop; the EXECUTION_PROTOCOL phase rule ("P2+ is explicitly deferred until P1-52 Trust Gate is reviewed") is hereby satisfiable — P2 planning may proceed on its own authority.
- Next allowed actions (candidate follow-ups, none blocking this gate, all outside this task's Allowed paths):
  1. Wire supersession-lineage exposure visibility into the production review projection (`internal/web/handlers/v13_changes.go`) so ancestor-anchored exposures surface on the current candidate page (derivation already supports it; E2E P1-49 is the spec).
  2. Accumulate real measurement samples through actual standard-profile task usage; re-read `BuildBaselineReport` outputs once any readout reaches sufficiency.
  3. On an operator-credentialed machine, run `TALLY_CODEX_LIVE=1 go test ./runtimes/codex/ -run TestLiveCodexSurfaceAuthGate` to renew live-surface evidence, and reconcile the probe doc §2.1 `-a` transcription.
  4. Repo hygiene (inherited P0-gate flag, since resolved): `*.log` test fixtures under `internal/logpipe/testdata/` are now tracked; keep the ignore rule from re-swallowing fixtures if new ones are added.
- Environment blockers to carry forward: `make` absent, golangci-lint/Go-1.27 export-data incompatibility (vet + normalized gofmt remain the accepted equivalents), CRLF gofmt artifacts on raw invocation.

## Notes

- No code was changed by this gate. The only file created is this report (`docs/tasks/v1.3/reports/P1_TRUST_GATE.md`), the task's sole Allowed path. The working tree of `C:\Dev\Code\tally` (primary checkout) was never written to; all work happened in `.worktrees/P1-52`.
- All quoted test names were executed in this environment during this gate; all file paths were read in this worktree. Agent/task-report claims were not accepted as evidence without re-execution or source verification.
