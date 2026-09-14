# P6-02 Execution Report — Remaining existing Outcome event capture

- **Status:** DONE
- **Date:** 2026-09-12
- **Baseline:** HEAD 07c3184 on master (audited baseline 979a55a); progress 35/57 at start
- **Execution model:** LOW (HIGH review)

## 1. Seam adjudication (chosen before code, per task contract)

**Ruling: current source has NO automatic production hook for any of `reverted` / `hotfixed` / `regression_found` / `human_correction` — per the task contract ("real hooks **or explicit capture APIs**") and the L3 pack ("expose explicit capture API if automatic hook does not exist"), the change lands as four explicit capture APIs in `internal/outcome`, each with its kind fixed by the entry point and its authoritative producer documented on the fact type. No producer site was faked, and `accepted` is deliberately NOT capturable through this surface (its only writer stays the P6-01 closedloop accept seam).**

### Source facts the ruling is based on (per-kind negative evidence)

1. **reverted — no hook.** The only "rollback" in production is `jobs.KindDeployRollback → deploy.Deployer.Rollback` (`internal/deploy/rollback.go`, `internal/deploy/deploy.go`): a project/env-scoped `deploy_run` accounting row whose payload carries **no task/changeset identity** (`deploy.Definition` = Project/Digest/BuildRun/TargetEnv), and `outcome.Event.Validate` requires at least one binding — a deploy rollback cannot be attributed as "an accepted candidate was reverted". `gitwriter.revert` (`internal/gitwriter/writer.go`) is transactional cleanup of a failed apply, not a product revert. Per the L3 failure discipline, `reverted` is never inferred from names or generic workflow failure.
2. **hotfixed — no hook.** No hotfix path exists anywhere in the repo (codegraph queries + string sweep): a hotfix is by definition an out-of-band fix bypassing the candidate loop; v1.4.1 source has no such action.
3. **regression_found — no hook.** The verification-side regression gate (`internal/verification/coverage_independence.go`) is a **pre-acceptance** gate; there is no post-acceptance re-verification of accepted candidates in current source. `internal/measurement/v14_metrics.go` only *reads* the event kind (§31.4 rework / §31.5 guardrail projections).
4. **human_correction — no hook.** `human_approval` / waiver facts (`internal/approval`, §19) are direction-ruling evidence, not rewrites of agent output; `internal/project/trace.go` is explicitly "无人工标注字段"; `internal/gitintegration` (P5-01..04) exposes only Integrate/Merge/EnsureIntegrationBranch — no human-edit capture.

### Why the capture API lives in `internal/outcome` (narrowest valid seam)

The four events happen **outside** the Closed Loop Engine — no Engine transition represents them, so a `closedloop`-side sink (the P6-01 shape) would be an invented authority with no caller. `internal/outcome/**` is P6-owned (PATH_OWNERSHIP) and already holds the frozen enum + the append-only store; the capture API is a typed, controlled entry over the existing single insert path (`Store.Record`), not a parallel subsystem. Read side (P6-03) is untouched.

### Trust boundaries preserved

- **`accepted` structurally excluded:** no `RecordAccepted` on this surface — `TestCaptureAPINeverExposesAccepted` pins that no `*Store` method name contains "Accepted", keeping P6-01's ruling intact (the Engine accept transition remains the only acceptance authority; no second acceptance-fact writer).
- **Kind fixed by entry point:** callers cannot record a wrong/unknown kind through the capture API (`RecordReverted`/`RecordHotfixed`/`RecordRegressionFound`/`RecordHumanCorrection` each hardwire the kind; `Event.Validate` still runs inside `Record` as the second line of defense).
- **Kind set untouched (DL-12/BI-02):** zero changes to the five-kind enum, to `Event`, to the store schema, or to any migration; `rejected`/`escalated`/`integration_failed` were not added (task requirement 2).
- **Failure policy (explicit fail-safe):** capture failures return coordinate-carrying errors; "fact not captured" is visible to the producer and is never interpreted as any domain verdict/candidate failure. Read failure during the replay check surfaces identically to write failure — "cannot read" is never folded into "nothing there".

### Idempotency / occurrence semantics (deliberately different from P6-01, and why)

P6-01 dedupes `accepted` on `(task, changeset)` because one logical acceptance must yield exactly one event. For the four captured kinds, deduping on `(task, changeset, kind)` would be **wrong**: two regressions on the same accepted changeset at different times are two facts, and the event surface has no payload to distinguish occurrences — **RecordedAt is the occurrence coordinate**. So the capture API suppresses only *exact replays* (same kind + coordinates + same instant, timezone-normalized via `Equal`); a different instant always appends. Check-then-append atomicity rests on the same single-writer caller discipline documented in P6-01 (no DB unique constraint; migrations outside this task's paths).

## 2. Changed files

- `internal/outcome/capture.go` (new): per-kind typed facts (`RevertFact`, `HotfixFact`, `RegressionFact`, `HumanCorrectionFact`, each with `Validate`: at-least-one non-blank binding, explicit `RecordedAt`), shared `validateCaptureFact`, four capture methods on `*Store` (kind fixed per entry), shared `capture` replay-safe append path + `existingStream` read (ForTask when task-bound, ForChangeset for out-of-band changeset-only hotfix), per-kind authoritative-producer documentation in type comments, header adjudication with the negative evidence above.
- `internal/outcome/capture_test.go` (new): unit + DB-backed tests (below).

No other file touched: `internal/closedloop/**`, `cmd/**`, `internal/web/**`, `runtimes/**`, `internal/workflow/**`, migrations — all untouched (git status shows only the other lanes' in-flight files: `internal/web/**`, `web/templates/**`, `runtimes/dsh/**`).

## 3. Semantic changes

- `internal/outcome` now exposes four explicit capture APIs for the four pre-existing non-accepted kinds, each with a documented authoritative producer (AC: "Each supported event has a documented authoritative producer" — producer = the control-plane role named on each fact type; the *hook* is explicitly declared not to exist in v1.4.1 source, per requirement 1).
- Producers that will call these APIs are future tasks (post-acceptance verification runs for `regression_found`, a control-plane revert action for `reverted`, incident-response registration for `hotfixed`, human-edit registration for `human_correction`); wiring a fake producer today was explicitly rejected.
- Read-side consumers (`measurement` §31 metrics, P6-03 governance projections) unchanged; they already tolerate partial binding (`UnboundAccepted`/`UnattributedSignals`) so capture facts follow the domain floor (at least one binding), not invented stricter per-kind rules.
- No behavior change anywhere else in the product; the worker composition (P6-01) is untouched.

## 4. Tests / commands executed (exact)

| Command | Result |
|---|---|
| `gofmt -l internal/outcome/` (after `gofmt -w` on both new files) | clean |
| `go vet ./internal/outcome/` | PASS |
| `go build ./internal/... ./cmd/...` | PASS |
| `DATABASE_URL=postgresql://postgres:postgres@localhost:15432/postgres go test ./internal/outcome/` | PASS (ok 54.1s; DB-backed tests ran against the shared dev Postgres, not skipped) |
| `DATABASE_URL=postgresql://postgres:postgres@localhost:15432/postgres go test -v -run "TestCapture" ./internal/outcome/` | PASS — 6/6: TestCaptureFactValidate, TestCaptureAPINeverExposesAccepted (unit), TestCaptureRoundtripEachKind, TestCaptureReplayIsIdempotent, TestCaptureDistinctOccurrenceAppends, TestCaptureRejectsInvalidFactWithoutRows (each DB-backed ~6.7s: per-test schema + goose migrations) |
| `go test ./internal/closedloop/` | PASS (ok 9.8s; package untouched, no regression) |
| `go test ./test/arch/` | PASS (ok 1.3s; DL-12 Outcome kind-set guard and P1-08 no-bypass guard stay green) |
| `git diff --check -- internal/outcome/` | clean |
| new-file line endings | LF (0 CR bytes; `.gitattributes` `* text=auto eol=lf` honored) |

`cmd/worker` suite not run: no `cmd/**` file modified (build proves compilation); the P6-01 worker tests exercise the accepted seam, which this task did not touch.

## 5. Baseline failures observed

None. Working-tree noise is confined to other lanes' paths (`internal/web/handlers/*`, `web/templates/*`, `runtimes/dsh/` — P2-03/P4-04), untouched and unaffected by this task.

## 6. Negative / failure-path tests added

- `TestCaptureFactValidate` — double-blank bindings, whitespace-only coordinates, missing `RecordedAt` rejected per kind; task-only and changeset-only legal forms accepted.
- `TestCaptureRejectsInvalidFactWithoutRows` (DB) — five invalid captures rejected with zero rows written (fail-safe: invalid input never becomes a silent empty success).
- `TestCaptureAPINeverExposesAccepted` — API-surface guard: no accepted writer on the capture face (P6-01 authority preserved).
- Replay-vs-occurrence pins: `TestCaptureReplayIsIdempotent` (same instant in a different timezone = same fact, one row) and `TestCaptureDistinctOccurrenceAppends` (different instant = second row, ordered).

## 7. Decision Locks

- **None touched or bypassed.** DL-12/BI-02: the five-kind closed enum, `Event`, store schema, migration 0023 surface and the append-only trigger are all unchanged; no new kind added. DL-13/DL-14 (event identity/append-only discipline) preserved. DL-10 untouched (no counterfactual outcomes invented — the whole point of not faking producers).

## 8. Remaining risks / notes

- **Documented-producer vs wired-producer gap:** the AC is satisfied by documentation + explicit API; actual production callers for the four kinds do not exist yet by design. When a producer lands (e.g., a future post-acceptance verification task), it must call these APIs and never infer events from deploy/workflow facts.
- Exact-replay idempotency is caller-discipline-based (check-then-append, no DB unique index) — same limitation class as P6-01's accepted sink, adjudicated there; a partial unique index would need migrations (P3-02 numbering, outside this task's paths).
- `docs/architecture/CURRENT_CODEBASE.md` / implementation-status updates are owned by P6-06 (docs lane); this report is the implementation fact record.
- Not committed (orchestrator owns commits); no commit hash.
