# P2-03 Execution Report — Runs projection

- **Status:** DONE
- **Date:** 2026-09-12
- **Baseline:** P0/P1 complete; P2-01/P2-02 landed; HEAD `14961b0` at start (progress 33/57)
- **Depends-on:** P2-01 (landed) — consumes its 研发 nav group, its `Page.Active` highlight mechanism, and the extension point its template comment reserved for the Runs entry (closed here).

## 1. Seam choice (narrowest valid seam)

**The run facts already have a complete typed read model: `internal/runstore.Run` (one row per CodegenRun, status is the row's run-control fact from migrations/0019) and an established web-tier read path `runstore.Store.CodegenRunsForTask(project, task)` already used by the work list (`v13_tasks.go`) and Attention (`v14_attention.go`). What did not exist was (a) a parameter-less read route over run facts (P2-01 correctly refused to fabricate a nav entry without one) and (b) any enumeration of runs across tasks. The landed change is therefore: a new read-only page `/v13/runs` (`V13Runs`) that enumerates the task inventory from the HEAD artifact snapshot (Git as the sole desired-side source, same as the work list) and expands each task's run timeline through the existing `CodegenRunsForTask` path; plus the one nav entry (运行), the one route line, the one template-registry string, and the nav-contract test updates that P2-01's extension point anticipated. No store query, no domain package, and no state machine is touched.**

### Exact source/tests read (reference order per the L3 pack)

1. **Tests:** `v13_http_test.go` (the v13 test harness: `v13NewServer`/`v13Repo`/`v13TestDB`/`v13Seed` — reused as-is; `TestV13PagesRenderEmptyStateWithoutSources` extended by one path), `nav_http_test.go` + `nav_test.go` (P2-01 nav contract — entries map, highlight cases, known-routes set, placeholder guards, all extended), `runstore_test.go` (runstore fixture idioms; `StartRunSpec`/`FinishRunSpec` field contracts), `internal/workflow/v13_test.go` (kernel run-lifecycle semantics: status vocabulary, escalation chain, `escalated_to:<profile>` reason words — the page renders these verbatim), `handlers_http_test.go` (authed harness behind `TestNavLinksResolveToRealRoutes`, untouched).
2. **Contract/interface:** PRD §6.1 (human mental model Project→Work→Plan→**Run**→Review), §29 (top-level Runs; "系统状态仍使用当前 state machine"), §30.1 (users see Work, not the internal state machine), §33 Phase 3 (Work/**Runs**/Reviews/Rules as projections, 不改状态机), §15.2 (run identity coordinates fixed at start), Decision Lock 15 (`internal/workflow` RunBegin comment: expected Model/RuntimeName vs `ObservedIdentity` recorded separately, never backfilled); `runstore.RunStatus` (migrations/0019 CHECK vocabulary: running/completed/failed/escalated/cancelled; terminal one-way); DL-19 (read/projection concern, no state-machine rewrite).
3. **Core implementation:** `internal/runstore/codegen.go` (Store read API: only `CodegenRun(id)` and `CodegenRunsForTask(project, task)` — there is **no** list-all-runs query, and `internal/store` sqlc files are a §10 hotspot outside this task's allowed paths; `Run.RuntimeCapabilitySnapshot` carries the runstore-injected `observed_identity` key when observed_* inputs were nonzero), `internal/runstore/changeset.go`/`verification.go` (method inventory confirming the task-scoped read surface), `internal/workflow/v13.go` (RunRecord/RunFinish semantics: failed/escalated/cancelled runs produce no ChangeSet; `ResultPatchHash` = ChangeSet identity; FailureReason vocabulary incl. `escalated_to:<profile>`), `v13_tasks.go` (v13 template stack + `v13Template` page registry + `v13env` guard), `v13_changes.go` (producer-run read precedent), `v14_attention.go` (`CodegenRunsForTask` consumption + projection-error honesty pattern), `v13_tasks.tmpl`/`base.tmpl` (template idioms), `web/static/tally.css` (st-ok/st-warn/st-bad/st-off chip classes).
4. **Failure handling:** sources nil → documented 未接入 empty state (200); Git load failure → explicit `LoadErr`; per-task run-read failure → explicit per-task flash (`V13TaskRunErr`), the task never silently disappears; no runs → honest absence empty state; observed-identity snapshot present but unparseable → explicit note (never silently downgraded to "未观察").

### Why this is the narrowest valid seam

- The AC is "Run status is derived from existing workflow/runtime facts; no duplicate run state." The page derives everything from rows it reads through the **existing** store adapter; status is rendered verbatim from `runstore.RunStatus` — no second status enum, no state field, no setter, no write route, no transition logic, no new SQL. The alternative "narrower" page (one list-all query) would have required widening into `internal/store` (§10 hotspot, outside allowed paths) and would have added zero AC value.
- Enumeration via the HEAD snapshot's task inventory is the same composition the work list and Attention already use (DL-04: one pinned source commit per page; `artifacts.Load(repoPath, headHash)` once). The honest consequence — runs of tasks absent from the HEAD snapshot are not listed — is stated explicitly in the page copy instead of being silently papered over.
- `server.go` change is the single route line plus one comment word (§10 shared hotspot: minimal registration seam, declared here). Template registration is one string in `v13Template`'s page list (`v13_tasks.go`, an owned file). Everything else is a new file or the nav-contract tests P2-01 built for exactly this extension.
- Provenance discipline (§13 / Decision Lock 15) is honored in presentation: Runtime/Model columns are labeled 期望坐标(开跑写定); the runtime-reported `observed_identity` (only when the run actually reported it) renders on a separate line as 实际…(runtime 自报); absence renders 实际身份未观察 — expected and observed never presented as each other.

### External reference used

- None beyond the pack's EXT-GO-HTML (Go `html/template` context-aware escaping — stdlib behavior already relied on by every template; no external code copied, no new dependency).

## 2. Changed files

- `internal/web/handlers/v13_runs.go` (new): `V13Runs` handler + `V13RunsPage`/`V13RunRow`/`V13TaskRunErr` view models. Row fields are verbatim row facts: status (+display-color class only), runtime/model expected coordinates, observed identity decoded read-only from the snapshot's `observed_identity` key (with explicit unparseable-snapshot note), escalated-from, iterations, tokens (`nil` semantics preserved as 用量未上报), started/finished (`—` = running), failure reason verbatim, candidate link when `ResultPatchHash` is set (failed/escalated/cancelled runs have none by §11.2.1). Deterministic newest-first ordering (StartedAt desc, row id desc tie-break). Summary counts derived from the rendered rows only.
- `web/templates/v13_runs.tmpl` (new): human Runs page — lede separating run process facts from Work-derived task state, unavailable/LoadErr/empty honest states, per-task read-failure flashes, one table (Run/任务/状态/Runtime·模型/迭代·用量/升级链/失败原因/固定候选), provenance footnote (期望 vs 自报实际; task derived state lives on 工作). Every fact is a stored fact; no placeholder copy.
- `internal/web/handlers/server.go` (§10 hotspot — declared minimal seam): one line `r.Get("/v13/runs", s.V13Runs)` in the existing v1.3 route block + one comment word. No other route/handler change.
- `web/templates/base.tmpl`: one nav entry `运行 → /v13/runs` (after 工作, matching PRD §6.1 Work→Run order) with new key `v13-runs`; group comment updated (Runs landed by its owner P2-03; Rules still deferred to P3). Existing entries byte-identical.
- `internal/web/handlers/v13_tasks.go`: one string — `"v13_runs"` added to the `v13Template` page registry.
- `internal/web/nav_test.go`: `humanNavEntries` gains `/v13/runs: 运行`; highlight case `{"v13-runs", "/v13/runs"}`; known-routes set gains `/v13/runs`; placeholder-guard comment updated (the never-existed `/runs` and `/v14/runs` phantoms stay banned).
- `internal/web/handlers/nav_http_test.go`: `humanNavEntries` gains `/v13/runs`; highlight-wiring table gains `{"/v13/runs", "/v13/runs"}` (the shared loop now iterates the entries slice).
- `internal/web/handlers/v13_http_test.go`: one path added to the empty-state loop (`"/v13/runs"` → renders 未接入 with nil sources).
- `internal/web/handlers/v13_runs_http_test.go` (new): two DB-gated tests over real run fixtures (below).

`internal/workflow/**` zero-changed (runs are read, never re-modeled), `internal/runstore/**` zero-changed, `internal/taskstate/**` zero-changed, all other templates and lanes untouched.

## 3. Semantic changes

- Run facts are now visible **separately from Work/Task state** (task goal): `/v13/runs` lists each CodegenRun as a process fact — status verbatim from the row's run-control fact, iterations, token usage, escalation chain (升级自 Run #N / `escalated_to:<profile>`), failure reason words verbatim (`runtime_error`, `wall_time_budget_exhausted`, …), and a link to the candidate a completed run fixed. The page states no task conclusion; the task state machine (derived on 工作) is untouched (PRD §29 note honored; DL-19).
- No duplicate run state anywhere: no new status vocabulary (display color classes are presentation only; text stays verbatim), no writable field, no route that mutates, no second store query, counts derived from the rows on the page.
- The P2-01 handover is closed: the 研发 nav now carries 运行 bound to a real, read-model-backed route — the nav no longer implies a missing capability, and `TestNavLinksResolveToRealRoutes` GETs it through the real router.
- Provenance presentation (§13): expected runtime/model and runtime-reported actual identity are separate, labeled lines; unobserved is rendered as such; a corrupt identity snapshot is an explicit error note, never silently "unobserved".

## 4. Tests / commands executed (exact)

| Command | Result |
|---|---|
| `go build ./internal/web/...` | PASS |
| `go vet ./internal/web/...` | PASS |
| `gofmt -l` on touched/new Go files | new `v13_runs.go`, `v13_runs_http_test.go`, `nav_test.go`, `nav_http_test.go` clean; `server.go`/`v13_tasks.go`/`v13_http_test.go` flagged only by the pre-existing whole-file CRLF checkout condition (present at baseline, same as P2-02's `v13_http_test.go` note; normalizing would be line-ending churn, §16) |
| `go test ./internal/web/ -run "TestNav" -v -count=1` | PASS (5/5) |
| `go test ./internal/web/handlers/ -run "TestNav\|TestV13PagesRenderEmptyState" -v -count=1` | PASS (3/3, incl. `TestNavLinksResolveToRealRoutes` now GETting `/v13/runs` through the real router) |
| `DATABASE_URL=postgresql://postgres:postgres@localhost:15432/postgres go test ./internal/web/handlers/ -run "TestV13Runs" -v -count=1` | PASS (2/2 new tests) |
| `rtk go test ./internal/workflow/... ./test/arch/... -count=1` | PASS (141 tests, 2 packages) |
| `DATABASE_URL=postgresql://postgres:postgres@localhost:15432/postgres go test ./internal/web/handlers/ -count=1` (full package, DB-gated included) | PASS (ok 153.412s) |
| `DATABASE_URL=postgresql://postgres:postgres@localhost:15432/postgres go test ./internal/web/... -count=1` | PASS (web ok 0.263s; handlers ok 149.791s) |
| `go build ./...` (whole repo incl. parallel-lane files) | PASS (exit 0) |
| `git diff --check` | clean (only the pre-existing CRLF-checkout warnings on four working-tree files; content diffs carry zero line-ending churn) |
| `make check` | NOT_RUN (no make binary in Git Bash; constituent families run individually as above) |

### New tests (real temp Git repo + real Postgres schema, SKIP-safe on DB absence)

- `TestV13RunsPageShowsRunTimelineFacts` — seeds a deterministic five-run timeline for TASK-0001 (completed fixing a candidate with usage; escalated with `escalated_to:strong`; its escalation successor completed fixing the second candidate **with** runtime-reported observed identity; failed `runtime_error`; one left running with no usage report) and asserts: statuses render verbatim (`>completed< >escalated< >failed< >running<`), escalation chain (升级自 Run #<id>), failure reasons verbatim, candidate links to `/v13/changes/{csid}` only for candidate-producing runs, task links to `/v14/tasks/TASK-0001`, 期望坐标/实际(runtime 自报)/实际身份未观察 three-way provenance presentation, 用量未上报 for unreported usage, `共 5 次运行` row-derived summary, newest-first ordering (late run's usage appears before early run's), no `>cancelled<` anywhere (not seeded — status comes from rows, not a fixed palette), and the nav highlighting exactly 运行 (not 工作) on the page.
- `TestV13RunsPageWithoutRunsStatesAbsenceHonestly` (negative) — same repo + fresh schema, zero runs: page asserts the honest-absence empty state and the banned-strings set (`>completed<` `>failed<` `>escalated<` `>cancelled<` `>running<`, 升级自, escalated_to, runtime_error, `/v13/changes/` links, 读取失败) — no status, candidate, or failure fact can be fabricated when no rows exist.

Inherited coverage re-verified: `TestV13PagesRenderEmptyStateWithoutSources` (now including `/v13/runs`), `TestNavLinksResolveToRealRoutes` / `TestNavHighlightWiringOverHTTP` / `TestNavHasNoPlaceholderEntries` / `TestNavHrefsAreKnownRoutes` (extended for the new entry, all green), plus the full DB-gated handlers package (140s family) passing unchanged.

## 5. Baseline failures observed

None in owned scope. All pre-existing handler/nav/workflow/arch tests pass unchanged. Parallel-lane working-tree edits were present and active (`runtimes/dsh/**` P4-04, `internal/gitintegration/**` P5-04, `cmd/worker/**` + `test/**` P4-02); none were touched, and `go build ./...` over the whole tree exited 0 at validation time. One implementation-time source-fact correction (not a baseline failure): the first handler draft read a `Profile` field from `runstore.Run`; the source fact is that codegen_run stores no profile column (the TaskContract carries the profile; `workflow.RunBegin.Profile` is consumed by hooks/records, not persisted on the run row) — the field was removed rather than synthesized.

## 6. Remaining risks

- Runs of tasks not present in the HEAD artifact snapshot are not listed (the only cross-task enumeration available within allowed paths is the Git task inventory; the same discipline as the work list). The page copy states this explicitly. If a list-all-runs read is ever wanted, it is a sqlc/store addition and belongs to a task that owns `internal/store`.
- The page reads per-task (N tasks → N queries per request, plus one `artifacts.Load`); fine at current scale, but a future list-all query would collapse it — noted for the P3 lane, not implemented here.
- Run statuses are shown verbatim (running/completed/…): they are run-control facts, not the task state machine, so this does not conflict with PRD §30.1; if P2-05's UX gate wants human synonyms for these words, that is a copy change in `v13_runs.tmpl` only.
- `/v14/changes/{csid}` still highlights nothing (pre-existing, unchanged — P2-02 recorded it; its page belongs to the P1-07 lane, not this task).
- The escalated-status display color (st-warn) is a presentation judgment; text remains the row's verbatim status.

## 7. Decision Locks touched

None. All locks read; none bypassed, weakened, or reinterpreted. Relevant readings honored: DL-19 (pure read/projection: no lifecycle transition, field, setter, or new status vocabulary anywhere; `internal/workflow`/`internal/runstore`/`internal/taskstate` zero-changed), DL-18 (existing domains consumed in place; no second run/task/review state), DL-04 (one page, one pinned HEAD: task inventory and every run read resolve under the same snapshot commit), Decision Lock 15 (expected vs observed identity rendered separately, never merged), §5 trust invariants (nothing on the page claims verification or acceptance; agent claims are not rendered as facts; absence is explicit). Shared hotspot: `server.go` touched only at the declared one-line route seam (§10). `hidden/` unread (DL-17).

## 8. Commit hash

Not committed — the orchestrator owns commits. Working tree contains exactly the nine code/template/test files in §2 plus this report (plus unrelated parallel-lane files: `runtimes/dsh/**`, `internal/gitintegration/**`, `cmd/worker/**`).
