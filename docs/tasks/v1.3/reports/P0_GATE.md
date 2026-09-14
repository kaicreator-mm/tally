# P0 Gate Report — P0-23 (P0 correctness integration gate)

- Date: 2026-09-05
- Auditor: P0-23 gate agent (read/verify/report only; no code fixes)
- Base: `55b8f81` (v1.3 documentation baseline) → HEAD: `e93807d` (branch `task/P0-23`, worktree `.worktrees/P0-23`)
- Environment: Windows, Git Bash, Go 1.27, git CLI; local Docker Postgres at `postgres://postgres:postgres@localhost:5432/tally?sslmode=disable` (reachable; shared db migrated 0001..0017 via `go run ./cmd/tally migrate`, reported "已是最新,无新迁移")

## Final gate verdict: FAIL

P1 must not start (EXECUTION_PROTOCOL phase rule: "P1 starts only after P0-23 PASS"). Exactly one acceptance criterion fails (AC3), and it is **not** a production correctness hole: it is one stale integration test asserting the retired pre-0016 legacy repro columns. All ten PRD §37 P0 correctness items are verified closed in merged code (AC1 PASS). The remediation is a single declared test adaptation (same class as the already-merged adaptations `e93807d` / `f7b42e0`) and is outside this gate task's Allowed paths.

## Environment blockers (recorded exactly)

1. `make check`: **NOT_RUN(reason: `make` is not installed in this Git Bash environment; exit 127 "make: command not found")**. All constituent commands were run individually instead (see VALIDATION section).
2. `golangci-lint run`: **NOT_RUN as a gate signal (toolchain incompatible with Go 1.27)**. Confirmed locally: golangci-lint v1.64.8 (built with go1.27.1) fails typechecking with `could not load export data: internal error in importing "unicode" (cannot decode "unicode", export data version 4 is greater than maximum supported version 2)` plus cascading spurious `undefined:` errors for imported packages. `go vet ./...` (PASS) and `gofmt` (PASS) performed instead.
3. Raw `gofmt -l .` on this Windows checkout flags 238 files purely from `core.autocrlf` CRLF working-tree artifacts. Equivalent normalized check run instead: `git -c core.autocrlf=false -c core.eol=lf checkout-index --prefix=/tmp/gate-check/ -a && cd /tmp/gate-check && gofmt -l .` → **0 files flagged**.

## Validation commands actually executed

| Command | Result |
| --- | --- |
| `go build ./...` | PASS (exit 0) |
| `go vet ./...` | PASS (exit 0) |
| Normalized `gofmt -l .` (LF checkout-index tree) | PASS (0 files) |
| `go test -count=1 ./test/arch/... ./internal/... ./cmd/...` | PASS (exit 0) |
| `go run ./cmd/tally migrate` (DATABASE_URL set) | PASS (idempotent; no new migrations) |
| `go test -count=1 -tags=integration ./...` (DATABASE_URL set) | FAIL — exactly 1 failing test, `TestWebhookBuildFailedIntegration` (`tally/internal/web`); all other packages ok |
| Targeted: `go test -tags=integration ./test/harvestenv/ -run TestHarvestEndToEndIntegration -v` | PASS (11.16s) — coordinator follow-up item re-verified |
| Targeted: `TestWebhookBuildAutoDiagnoseEnqueues` | PASS inside the full integration run (not among failures) |
| `make check` | NOT_RUN(`make` not installed; components run individually) |
| `golangci-lint run` | NOT_RUN(toolchain incompatible with Go 1.27 export data; see above) |

### The one integration failure, root-caused

`TestWebhookBuildFailedIntegration` (internal/web/webhook_integration_test.go:172-181) deterministically fails:

```text
webhook_integration_test.go:173: 复现上下文 = ("", ""),想要 (/tmp/demo, make build)
webhook_integration_test.go:180: other_targets = [],想要 [make other-a make other-b]
```

- Root cause: the test reads the raw legacy columns `workdir` / `repro` / `other_targets` from `GetLatestFailureEventByFingerprint` and asserts they carry the webhook payload. Since migration `0016_failure_command_spec.sql` (contract P0-04 `1fdf028`, end-to-end P0-12 `b223a94`, sqlc regen P0-09 `4404cbc`), new writes populate **only** the structured columns `repro_command` (CommandSpec canonical JSON) and `other_target_commands`; the legacy text three columns are `NOT NULL DEFAULT ''` and are read-compat sources for pre-0016 rows only. The dual-track reconstruction happens at load time in `internal/exec/diagnose.go:78-97` (structured JSON, else `failure.LegacyReproCommand(workdir, repro)`).
- Proof this is test staleness, not data loss: `internal/ingest/ingest_integration_test.go:255-331` (`TestIngestBuildPersistsReproContext`, passing in this run) verifies the **same** `IngestBuild → RecordFailure` path the webhook wraps, asserting `repro_command = {argv:[make,build], cwd:/tmp/demo, shell:false}`-equivalent structured persistence AND explicitly asserting the legacy columns are empty on new writes (`新写入不应再落旧文本列`). Both tests cannot pass simultaneously; the ingest test encodes the merged contract, the webhook test encodes the retired one.
- The webhook handler itself passes the fields through correctly (`internal/web/webhook.go:135-137` → `ingest.BuildInput{Workdir, Repro, OtherTargets}`), and per-test schemas are fully isolated (`internal/store/testdb.go`, one schema per test), so this is not pollution or a dropped field.
- Note on coordinator note 5: the earlier diagnosis of this test ("fixed by seeding project_index") was incomplete — the seeding fix (merged in `e93807d`) cleared the ghost-project guard, exposing the stale legacy-column assertions at a later assertion line. `TestWebhookBuildAutoDiagnoseEnqueues` (the other seeding-fixed test) passes. `TestHarvestEndToEndIntegration` passes.
- Required remediation (owner/integrator, outside gate Allowed paths): adapt `TestWebhookBuildFailedIntegration` lines 164-181 to the 0016 structured contract — read `ReproCommand` / `OtherTargetCommands` (or the dual-track loader) instead of the raw legacy columns. Acceptance property ("复现上下文与版本组合原样持久化", T11b) remains fully verified, just via the authoritative columns. No production code change is indicated.

## AC1 — No known P0 hole remains from SOURCE_BASELINE: PASS

Audited against SOURCE_BASELINE.md "Audited source facts" 1..15 and PRD §37 P0 item list (trace tuple, project namespace, LLM role separation, failure worktree, structured repro, lease recovery, Git serialization, spec_defect closure, false-green, deploy inconsistency). Each hole verified closed in merged code:

| # | Source fact / PRD §37 item | Closure evidence (file + commit) |
| --- | --- | --- |
| 1 | Legacy `/tasks` one-project-one-task false-green | `internal/web/handlers/taskstatus.go` rewritten ("P0-22 重述语义"): columns derived only from execution records, no manual status fields, coverage shown as coverage fact not verification verdict — `cb3392b`; also `tasks.go`, `taskstatus_internal_test.go`, `web/templates/tasks.tmpl`, `base.tmpl` |
| 2 | Jobs: no lease owner/heartbeat/stale recovery | `migrations/0014_job_lease.sql` + `internal/store/queries/job.sql` (`b4bc792`); `internal/jobs/jobs.go` Claim uses `FOR UPDATE SKIP LOCKED`, writes owner+lease, claims pending-or-stale rows, `HeartbeatJob` fencing renewal, stale-predicate reclaim (`7cedc42`, contract test `test/arch/job_lease_contract_test.go`) |
| 3 | Trace: `DeleteStaleTraceEdges` column-wise compare; no project namespace | `internal/store/queries/trace.sql:144` now deletes by full tuple via `NOT EXISTS` on (project_id, from_type, from_key, to_type, to_key); `migrations/0015_trace_project_namespace.sql` (`aaecce7`); reindex project-namespaced `internal/project/reindex.go`, `internal/project/trace.go` (`a09b02c`); contract test `test/arch/trace_namespace_contract_test.go` |
| 4 | `internal/llm/llm.go` hardcoded interview prompt for all callers | Role-aware proposal client: `Role` / `WithRole` / `ErrNoRole` gate — Propose refuses without a caller-declared role; no built-in persona (`39c2c29`) |
| 5 | Worker writes candidate files into supplied workdir + `strings.Fields(command)` | `cmd/worker/main.go:171-267` `failureRunner`: structured `CommandSpec` executed by Argv (no `strings.Fields`), disposable worktrees via `internal/workspace` (P0-05 `a17c214`), root forced outside owner checkout, never falls back to running in the owner directory (INV-EXEC-01) — wired by P0-21 `766b5ca` |
| 6 | `runner.Command` structured (Argv/Dir/Env) — reuse, don't replace | Preserved and reused (e.g. `internal/runner/docker.go`); not replaced |
| 7 | `failure_event` stores `workdir`/`repro` TEXT, `other_targets` | `migrations/0016_failure_command_spec.sql` + `internal/store/queries/failure_event.sql` header contract: new writes only `repro_command` (CommandSpec JSON) + `other_target_commands` (`1fdf028`, regen `4404cbc`); write path `failure.RecordEvent` (`b223a94`); ingest boundary converts legacy text via `failure.LegacyReproCommand` / `legacyOtherTargets` (`internal/ingest/ingest.go:116-159`); read side dual-track in `internal/exec/diagnose.go` |
| 8 | Git writes bypass serialization (content.go, sign.go, catalog.go, knowledge/store.go) | `internal/gitwriter` lock + writer (`1c7a548`); all four call sites route through it: `internal/web/handlers/content.go` (`3a3bbfc`), `internal/requirement/sign.go` (`b38f5b8`), `internal/harvest/catalog.go` (`539cca5`), `internal/knowledge/store.go` (`d0a149a`) |
| 9 | `SpecDefectSink` nil in production wiring | `internal/requirement/specdefect.go` production adapter (`afb84e6`); wired in `cmd/worker/main.go:88-91` (`SpecSink: requirement.SpecDefectSink{...}`, `766b5ca`) |
| 10 | `jobs.KindDeploy` exists but no deploy handler registered | `internal/exec/exec.go:61` registers `jobs.KindDeploy: cfg.handleDeploy()` (plus rollback) — `822f8e0` |
| 11 | Failure gates are valuable deterministic assets — preserve | Preserved on structured specs: `internal/failure/gates.go` G1 `ConfirmRepro` (5x red), G2 repeated green, G3 other targets, G4 scope (`b223a94`) |
| 12 | Append-only fact triggers + arch tests must extend | `migrations/0001_facts.sql` (5 triggers) + `migrations/0011_append_only_statement.sql` (10 statement triggers); `test/arch/append_only_test.go` dual enforcement (sqlc query scan + DB trigger presence); four new contract arch tests added with the P0 SQL contracts |
| 13 | Rich existing tests; avoid directory-only reorg | No cosmetic reorg; new packages (`internal/command`, `internal/gitwriter`, `internal/workspace`) are functional P0 deliverables |
| 14 | sqlc-generated files are shared hotspots; only dedicated tasks regenerate | `git log 55b8f81..HEAD -- internal/store/querier.go` → exactly one commit, `4404cbc` (P0-09, "regenerate sqlc for P0 contracts, sqlc v1.31.1"); no leaf task edited generated files |
| 15 | `cmd/worker/main.go` and `internal/web/handlers/server.go` are serialized hotspots | `internal/web/handlers/server.go`: **0 changes** in range; `cmd/worker/main.go` touched only by owner P0-21 (`766b5ca`) |

PRD §37 P0 checklist: trace tuple (3) ✓, project namespace (3 + P0-08 project index `5439ca4` + P0-13 ghost-project guard `2af3195` — unknown projects fail-closed rejected at `internal/ingest/ingest.go:ensureKnownProject` before any fact INSERT) ✓, LLM role separation (4) ✓, failure worktree (5) ✓, structured repro (7) ✓, lease recovery (2) ✓, Git serialization (8) ✓, spec_defect closure (9) ✓, false-green (1) ✓, deploy inconsistency (10) ✓.

## AC2 — All changed paths stay within task scopes: PASS (with flagged items)

Method: `git log --no-merges --name-only 55b8f81..HEAD`; every non-merge commit's file list checked against its task's Allowed paths.

- Leaf-task commits match their Allowed paths for P0-01/02/03/04/05/06/07/08/09/10/11/14/15/16/17/18/19/21/22 (verified per-commit, spot-checked against task files).
- sqlc generated files: only `4404cbc` (P0-09, dedicated generation owner). `internal/web/handlers/server.go`: untouched. `cmd/worker/main.go`: only `766b5ca` (owner).
- **Flagged — declared integrator/coordinator amendments** (cross-boundary touches, openly labeled in commit messages, performed by the serialized integrator role per EXECUTION_PROTOCOL "Integrator tasks own named integration points"):
  - `44d58fa` "P0-11 integrator amendments": `internal/project/{annotation.go,trailer.go,helpers_test.go,trailer_integration_test.go,views_integration_test.go}` (test adaptation to 0015/P0-04 contracts).
  - `2af3195` P0-13, declared "worker + coordinator completion": touches P0-21-owned `internal/exec/diagnose.go` (dual-track read per P0-12 fix map) and P0-12-owned `internal/failure/budget.go` (JSON-null sentinel fix) + test adaptations to 0016.
  - `31fcd83` "integrate: ... (scope amendment: unowned file)": `internal/web/handlers/trace_integration_test.go`.
  - `f7b42e0` "integrate: W03 downstream adaptations": `internal/deps/retire.go` + store/web/deps integration tests.
  - `766b5ca` "P0-21 ... coordinator test-timing completion": `cmd/worker/main_test.go` timing adaptation.
  - `e93807d` "integrate: webhook integration tests seed project_index": `internal/web/webhook_integration_test.go`.
- **Flagged — two mechanical deviations in leaf-labeled commits** (recorded for integrator adjudication; judged non-silent and functionally forced, hence not scope violations):
  - `b223a94` (P0-12) additionally touches `internal/failure/budget.go`, which is not in P0-12's Allowed list. `RecordEvent` — the exact function P0-12 rewrites for structured persistence — lives in that file; the change is the task's own core subject, and the same file later received the declared coordinator sentinel fix in `2af3195`.
  - `822f8e0` (P0-20) edits `docs/guides/operations.md` while its Allowed list says `docs/07-operations.md`. That path no longer exists: the v1.3 docs restructure in the gate base `55b8f81` relocated it to `docs/guides/operations.md`. Same logical document; the task-file path is stale, not the edit.
- No silent leaf-scope violations found.

## AC3 — Suites green, or environment blockers recorded exactly: FAIL

- Architecture suite: PASS. Unit suites (`./internal/...`, `./cmd/...`): PASS (fresh, `-count=1`).
- Integration suite (`go test -count=1 -tags=integration ./...` with DATABASE_URL): **FAIL — exactly one deterministic failure**, `TestWebhookBuildFailedIntegration`. Root cause, proof, and required remediation documented above; it is a stale test asserting retired pre-0016 legacy columns, **not** an environment blocker and **not** an unclosed P0 hole (the correctness property is independently verified by the passing `TestIngestBuildPersistsReproContext` on the same ingest path, plus the dual-track loader in `internal/exec/diagnose.go`).
- Coordinator follow-ups re-verified: `TestHarvestEndToEndIntegration` PASS (targeted rerun, 11.16s); `TestWebhookBuildAutoDiagnoseEnqueues` PASS in full run.
- Environment blockers recorded exactly in their section above (`make` missing; golangci-lint/Go 1.27 export-data incompatibility; CRLF gofmt artifacts).

## AC4 — No P1 domain implementation smuggled into P0: PASS

- `internal/` contains no `taskcontract`, `verification`, `evidence`, `changeset`, `codegen`, `runtime`, `review`, `holdout`, or any other P1-domain package; `cmd/` is still only `server`, `tally`, `worker`.
- New packages in range are exactly the P0-scoped ones: `internal/command` (P0-03), `internal/gitwriter` (P0-06), `internal/workspace` (P0-05).
- New migrations 0014-0017 are the P0 SQL contracts (job lease, trace namespace, failure CommandSpec, project index) — no P1 fact tables, no workflow/DAG/runtime adapters, no P2+ features.

## Gate disposition

- Gate verdict: **FAIL**, solely on AC3 (one stale integration test).
- Single required action before P1: owner/integrator adapts `TestWebhookBuildFailedIntegration` to the 0016 structured repro contract (test-only change; the acceptance property is preserved, not weakened). Then rerun `go test -tags=integration ./...` and re-issue this gate.
- No production code changes are indicated by this audit.

## Second issue (re-verification)

- Date: 2026-09-05
- Agent: P0-23 gate re-verification agent (report file is the only tracked path touched; no code changes)
- Scope: verify the single remediation required by the first issue, re-run the full integration suite, re-confirm build/vet/unit suites, and issue final per-AC verdicts.
- Range since gate base `e93807d`: `b38c7d1` (this report, first issue), `ac15125` (the AC3 remediation), `2dbcfa6` (report merge). Nothing else changed.

### Remediation verified

`ac15125` "integrate: adapt TestWebhookBuildFailedIntegration to 0016 structured contract (P0-23 AC3 remediation, test-only)" addresses the first issue's single FAIL item exactly as prescribed:

- **Test-only**: touches exactly `internal/web/webhook_integration_test.go` (+19/−7). No production code changed.
- **Structured contract asserted**: `ev.ReproCommand` unmarshalled as `command.CommandSpec` with `cwd=/tmp/demo`, `argv=make build`; `ev.OtherTargetCommands` unmarshalled as `[]failure.OtherTarget` with `make other-a`, `make other-b` — read from the authoritative 0016 columns, as the first issue required.
- **Legacy columns asserted empty on new rows**: adds `ev.Workdir == "" && ev.Repro == ""` assertion (新行旧文本列恒空) — strengthens, not weakens, the check.
- **Acceptance property preserved (T11b)**: build_run_id pairing, fp_exact vs go extractor, raw_log_ref, and build_run (status/exit_code/commit) assertions unchanged; the repro-context persistence property is now verified via the authoritative structured columns, consistent with the passing `TestIngestBuildPersistsReproContext`.

### Environment notes (recorded exactly)

1. First integration attempt in this worktree failed 7 tests across 3 packages — all environmental, none related to the remediation, and none reproducible against the canonical checkout:
   - `internal/logpipe` + `internal/logpipe/toolchain` (7 tests): the `.log` sample fixtures under `internal/logpipe/testdata/` are matched by the `.gitignore` rule `*.log` and were never tracked (`git log --all --diff-filter=A` finds no `.log` fixture ever committed), so a fresh worktree does not have them. Resolution: the 16 canonical fixture files were restored into the worktree from the owner checkout (untracked, ignored — `git status` remains clean; no tracked path touched). Repo hygiene flag for a future task (outside this gate's allowed paths): the `*.log` ignore rule silently excludes test fixtures that `TestTestdataIsRedacted` explicitly requires to exist.
   - `internal/project` `TestReindexWithGitRepoThenVerify` (attempt 1 only): `CREATE SCHEMA tally_it_<nanots>` failed once with `duplicate key value violates unique constraint "pg_namespace_nspname_index"` — the known Postgres concurrent-DDL catalog race when parallel test packages create schemas simultaneously (per-test schema isolation in `internal/store/testdb.go`). Transient flake; passed on the full re-run below. Not a code or data defect.
2. `make` still not installed; `golangci-lint` still v1.64.8 (built with go1.27.1) — same toolchain incompatibility recorded in the first issue. Both remain NOT_RUN with the first issue's recorded reasons; all constituent commands were run individually.

### Validation commands actually executed (this issue)

| Command | Result |
| --- | --- |
| `export DATABASE_URL='postgres://postgres:postgres@localhost:5432/tally?sslmode=disable' && go test -count=1 -tags=integration ./...` | **PASS** (exit 0) — every package `ok`, including `internal/web` (remediated test inside), `internal/project`, `internal/logpipe`, `internal/logpipe/toolchain`, `test/arch`, `test/harvestenv` |
| `go build ./...` | PASS (exit 0) |
| `go vet ./...` | PASS (exit 0) |
| `go test -count=1 ./test/arch/... ./internal/... ./cmd/...` | PASS (exit 0; 23 packages ok, 0 failures) |
| `make check` | NOT_RUN(`make` not installed; components run individually — unchanged from first issue) |
| `golangci-lint run` | NOT_RUN(toolchain incompatible with Go 1.27 export data — unchanged from first issue) |

### Final per-AC verdicts

- **AC1 — No known P0 hole remains: PASS** (unchanged; no production code changed since the first issue verified all ten PRD §37 closures).
- **AC2 — All changed paths stay within task scopes: PASS** (only new commit `ac15125` is the gate-prescribed test-only remediation, openly labeled as an integrator amendment in the established `integrate:` pattern; `b38c7d1`/`2dbcfa6` are this report's own files).
- **AC3 — Suites green, or environment blockers recorded exactly: PASS** (integration suite fully green on `-count=1`; build/vet/unit green; remaining blockers are tool-availability NOT_RUNs recorded exactly above).
- **AC4 — No P1 domain implementation smuggled into P0: PASS** (no new packages, no new migrations, no P1-domain code since the first issue's audit).

### Final gate verdict: PASS

The single FAIL item from the first issue is remediated test-only, with the acceptance property preserved and strengthened. All suites are green. P1 may start per the EXECUTION_PROTOCOL phase rule.
