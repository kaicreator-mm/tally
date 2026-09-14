# P2-02 Execution Report — Work list and task detail projection

- **Status:** DONE
- **Date:** 2026-09-12
- **Baseline:** P0/P1 complete, P2-01 landed (`b884b5e`); HEAD `33197fe` at start (progress 28/57)
- **Depends-on:** P2-01 (landed) — consumes its 研发 nav group and the `Page.Active` highlight mechanism; closes the two P2-01/P1-07 handovers listed in Risks (task-detail highlight refinement; task page basis lacking `compiled:<hash>`).

## 1. Seam choice (narrowest valid seam)

**The Task human view already has a complete read model: `v14_task.go` projects Git artifacts + fact rows + `taskstate.Derive` into one page, and the P1-07 change-detail lane already wired the compiled authority (`artifacts.LoadBundle → CompileAuthority`) plus revision-bound coverage derivation (`loadV13Change → deriveV13Coverage`) and the `VerifiedBasis` projection with its four-dimension rendering (`v14BasisDims`). What was missing was (a) the PRD §7/§29 human mental-model *presentation* of the Task (Overview/Plan/Changes/Verification/Reviews/Activity), (b) the same compiled/risk/assurance basis on the task page (P1-07 explicitly deferred it to "that page's owning lane" = this task), and (c) a Work list that can actually reach the work item. The landed change is therefore: re-group the existing task-page sections in `web/templates/v14_task.tmpl`, consume the two existing read paths in `v14_task.go` (no new projection logic, no taskstate change), and add one link + human heading in `v13_tasks.tmpl`.**

### Exact source/tests read (reference order per the L3 pack)

1. **Tests:** `v14_task_test.go` (page-test idiom, seed helpers, all surviving string assertions), `v14_change_test.go` (P1-07 compiled-basis expectations + the `v14cIntentRepo`/`v14cPassCandidate`/`v14cVerifyEvidence` fixtures reused as-is), `taskstatus_internal_test.go` (read; P0-22 exec-summary — confirms `/tasks` legacy view is a different domain, untouched), `v13_http_test.go` (Tasks-page assertions incl. the `"P1 任务"` heading check), `nav_http_test.go`/`nav_test.go` (P2-01 nav contract; no `Active`-key assertions on `/v14/tasks/{task}` — key change is regression-free).
2. **Contract/interface:** PRD §7 (Task/Work page: `Overview/Plan/Changes/Verification/Reviews/Activity` "对应现有领域对象 projection"; page top shows repository/base revision/contract state/risk/assurance/harness/execution status), §29 (UI IA: `Work → Task` with the same six sections; "系统状态仍使用当前 state machine"), §30.1 (users see Work, not internal state machine), §26.2; `taskstate.Result.VerifiedBasis` / `Verdict` / `Basis` (PRD §14, Decision Lock 12: derived read model); DL-19 (read/projection concern, no state-machine rewrite).
3. **Core implementation:** `v14_task.go` (existing derive assembly), `v14_change.go` (`v14ChangeBasis` — the P1-07 production wiring pattern consumed here: LoadBundle at the page's already-loaded pinned HEAD hash, coverage fed only when the existing revision-bound derivation is available, `CompiledNote` on typed failure, `v14BasisDims` four-dimension rows), `v13_changes.go` (`loadV13Change`, `deriveV13Coverage`, `v13CoverageView`), `internal/taskstate/basis.go` + `state.go` (projection is pure and complete; `Compiled` only adds `VerifiedBasis` + `compiled:<hash>` basis entry — re-derive with it is deterministic and additive, INV-STATE-01), `web/templates/v14_task.tmpl` / `v14_change.tmpl` / `v13_tasks.tmpl` / `base.tmpl`, `v14_launch.go` (v14 template stack funcMap), `v13_tasks.go` (Work list rows).
4. **Failure handling:** Compiled absent / `CompileAuthority` typed failure / derive failure on re-derive → explicit `已编译合同未接入:<reason>` note, first-derive conclusions preserved, never a fabricated verdict/level/`compiled:` reference; current-candidate coverage read failure → explicit `Coverage.Notice`, page never blocked.

### Why this is the narrowest valid seam

- The projection logic already exists and is production-wired (P1-07 on the change page). Re-implementing any of it in the task page would duplicate a derived read model (DL-18 extend-not-duplicate; Derive stays the sole production source). The task page composes the two existing seams — `LoadBundle → CompileAuthority` (worker's P1-03 seam, same pinned HEAD hash the page already loaded its snapshot from, DL-04) and `loadV13Change → deriveV13Coverage` (the change page's own read path) — for the *derived current candidate* located by the first `Derive`.
- The second `Derive` with `Snapshot.Compiled` (+coverage) is required by the data flow (the current candidate is only known after deriving) and is provably additive: `Compiled` feeds only `VerifiedBasis` and the `compiled:<hash>` basis entry in `taskstate.Derive`; state/class/gaps are identical between the two derives by construction.
- Presentation lands entirely in templates; `internal/taskstate/**` is zero-changed (acceptance/verdict semantics frozen per PATH_OWNERSHIP); `server.go` untouched (no new route needed — `/v14/tasks/{task}` and `/v13/tasks` already exist).
- Highlight refinement is the one-string `Active` split P2-01 handed over: the task detail page is a Work-domain view (PRD §29 `Work → Task`), so `Active "v14"` (matched nothing) → `"v13"` (工作 highlights on the work item's own detail page).

### External reference used

- None beyond the pack's EXT-GO-HTML (Go `html/template` context-aware escaping — stdlib behavior already relied on by every template; no external code copied, no new dependency).

## 2. Changed files

- `internal/web/handlers/v14_task.go`:
  - `V14TaskPage` gains projection-only fields mirroring the P1-07 change-page contract: `Coverage` (V13CoverageView), `BasisDims` ([]V14BasisDim), `HasVerdict`, `Verdict`, `VerdictLimited`, `VerdictOK`, `CompiledHash`, `CompiledNote`. No state field, no setter.
  - `v14TaskPage`: after the existing derive, loads the compiled authority (`v14TaskAuthority`) and the current-candidate coverage (`v14TaskCoverage` → `loadV13Change → deriveV13Coverage → v13CoverageView`), re-derives with `Snapshot.Compiled` (+ coverage reports only when the revision-bound derivation is available; typed re-derive failure keeps the first derive and records the reason), then projects `BasisDims` (existing `v14BasisDims`) and the verdict fields.
  - `V14Task`: `Active` key `"v14"` → `"v13"` (P2-01 handover; 任务详情高亮「工作」).
  - Header data-discipline comment records the P2-02 reading.
- `web/templates/v14_task.tmpl`: re-grouped into the PRD §7/§29 six human sections — 概览 Overview (identity + derived state + verdict line incl. `compiled:<hash>` + attention flash), 计划 Plan (frozen acceptance clauses), 变更 Changes (immutable candidates + supersession, unchanged content), 验证 Verification (basis dims, current-candidate coverage table, Basis references, gaps, ignored claims, exposures), 评审 Reviews (approval evidence + 需要我做什么 action block), 动态 Activity (repair chain, unchanged content). Every fact shown before is still shown; honest empty states preserved verbatim where tests pin them.
- `web/templates/v13_tasks.tmpl`: Work list presented in human terms — h1 `P1 任务` → `工作` (PRD §30.1; nav entry already says 工作 since P2-01), lede rewritten to the Work→Task mental model (derivation explanation kept), and each row's TaskID now links to `/v14/tasks/{task}` (previously the work list had no path to the work item).
- `internal/web/handlers/v14_task_test.go`: three new tests + section-header fixture + `wantStrings` helper (below).
- `internal/web/handlers/v13_http_test.go`: two assertion lines — `"P1 任务"` → `"<h1>工作</h1>"` and added `href="/v14/tasks/TASK-0001"` (intended-behavior update; all derivation assertions untouched).

`server.go`, `router.go`, `render.go`, `base.tmpl`, `internal/taskstate/**` (zero-changed — projection was already complete; its test suite passes untouched), `v14_change.go`, and every other-lane file: untouched.

## 3. Semantic changes

- The Task detail page now presents the PRD §7/§29 human model (Overview/Plan/Changes/Verification/Reviews/Activity) over the *same* derived facts as before — a pure read-side re-grouping; no route, no lifecycle field, no derive-semantics change (DL-19; the §29 note "系统状态仍使用当前 state machine" holds: derived state names remain visible in Overview).
- The task page's verification basis now carries the compiled/risk/assurance facts where available (task AC-2): verdict `VERIFIED` / `VERIFIED_LIMITED` / `NOT_VERIFIED` (limited never rendered as plain VERIFIED), `compiled:<hash>` in the basis list (INV-STATE-02 — closes the P1-07 report's noted gap for this page), required/achieved assurance, matched-policy count with unmatched/unknown risk signals verbatim (fail-safe L3 uplift visible), and the four basis dimensions with honest 未接入 notes when Compiled is absent. Assured statements are never made without Compiled; absence is an explicit typed note, never a downgrade of any gate.
- Coverage (functional + independence, §8.8.2 revision-bound) for the derived current candidate is now visible on the task page via the change page's own read path, so task and current-candidate pages show one consistent basis instead of two different truths.
- The Work list (工作) now navigates Work → Task (PRD §29): every row links to the task detail; the page speaks human terms (`工作`) instead of internal phase vocabulary (`P1 任务`), serving PRD §30.1 and the P2-05 gate direction.
- Work-list heading change is copy-only; derivation, counts, gaps, and approval rendering on that page are byte-identical.

## 4. Tests / commands executed (exact)

| Command | Result |
|---|---|
| `go build ./internal/web/...` | PASS |
| `go vet ./internal/web/...` | PASS |
| `go test ./internal/taskstate/... -count=1` | PASS (ok 0.847s) |
| `go test ./internal/web/ ./internal/taskstate/ -count=1` | PASS (ok 0.302s / ok 0.882s) |
| `go test ./test/arch/... -count=1` | PASS (ok 1.970s) |
| `go test ./internal/web/handlers/ -run "TestV14TaskUnavailable\|TestNav" -count=1 -v` | PASS (3/3) |
| `DATABASE_URL=postgresql://postgres:postgres@localhost:15432/postgres go test ./internal/web/handlers/ -run "TestV14Task\|TestV13TaskPage" -count=1 -v` | PASS (7/7: 3 new P2-02 + 2 pre-existing V14Task + V13TaskPage + NotFound) |
| `DATABASE_URL=postgresql://postgres:postgres@localhost:15432/postgres go test ./internal/web/handlers/ -count=1` (full package, DB-gated included) | PASS (ok 140.465s) |
| `DATABASE_URL=postgresql://postgres:postgres@localhost:15432/postgres go test ./internal/web/ -count=1` | PASS (ok 0.253s) |
| `go build ./...` (whole repo incl. parallel-lane files) | FAIL in `tally/runtimes/dsh` only (undefined symbols in P4-04's in-progress files); retried once after 60s per cross-lane rule, still in-flight — transient cross-lane noise originating ONLY outside my paths; `go build ./internal/web/... ./internal/taskstate/...` and the full handlers suite pass |
| `gofmt -l` on touched Go files | `v14_task.go`, `v14_task_test.go` clean; `v13_http_test.go` flagged only by the pre-existing whole-file CRLF checkout condition (present at baseline, same as P1-07's `v13_changes.go` note; normalizing it would be line-ending churn, §16) |
| `git diff --check` | clean (only pre-existing CRLF-checkout warnings on working-tree files; content diffs carry zero line-ending churn) |
| `make check` | NOT_RUN (no make binary in Git Bash; constituent families run individually as above) |

### New tests (all via real temp Git repo + real Postgres schema, SKIP-safe on DB absence; reuse the P1-07 `v14cIntentRepo` / `v14cPassCandidate` fixtures)

- `TestV14TaskCompiledBasisPlainVerified` — zero-policy, zero-signal intent: the six human sections render; verdict is plain VERIFIED (OK branch, never `VERIFIED_LIMITED`, no 未接入 note); basis carries `compiled:sha256:…` plus the real fact reference `fact:v13art:task_planned` (INV-11); assurance dim shows `必需 L1 · 达成 L1`; unmatched dim shows `无未匹配风险信号`; current-candidate coverage table renders (`针对冻结契约`); 工作 nav entry carries `aria-current="page"` on the task detail page.
- `TestV14TaskCompiledBasisVerifiedLimitedOnUnmatchedRisk` — owner-confirmed `auth_symbol_change` hint with no matching policy: verdict is `VERIFIED_LIMITED` with the 不得显示成普通 VERIFIED distinction, signal shown verbatim, `必需 L3 · 达成 L1` (fail-safe uplift visible), `compiled:sha256:…` present.
- `TestV14TaskCompiledAbsentKeepsHonestBasis` (negative) — v1.3-layout repo without FrozenIntent: page shows `已编译合同未接入` (typed reason) and all six sections with the real fact basis (`依据(Basis`, `fact:v13art:task_planned`, `没有暴露事实`); banned strings asserted absent: verdict line, `必需 L`, `达成 L`, `compiled:sha256:` — no assurance claimed, no basis fabricated (AC-2 failure path).

Negative/behavior-preservation coverage inherited and re-verified: `TestV14TaskDetailShowsRepairChainBasisAndUncertainty` (v1.3-layout repo, two-attempt chain) passes unchanged — proving the restructure kept every pinned fact rendering and that the Compiled-absent path adds only the explicit note; `TestV14TaskNotFound` / `TestV14TaskUnavailableWithoutSources` unchanged.

## 5. Baseline failures observed

None in owned scope. All pre-existing handler/taskstate/arch tests pass unchanged. Parallel-lane working-tree edits were present and active (`runtimes/dsh/**` P4-04, `internal/gitintegration/**` + `internal/gitwriter/**` P5-03, `Makefile` P6-06); none were touched; the only cross-lane effect is the `go build ./...` failure inside `runtimes/dsh` recorded above (retried once after 60s per the transient-noise rule).

## 6. Remaining risks

- The compiled basis on the task page reflects the pinned HEAD artifacts, same as every other derive input on the page; a request racing a freeze commit re-derives deterministically from the new HEAD.
- When Compiled is wired but the current candidate's coverage reports cannot bind the frozen revision (§8.8.2), the verdict is NOT_VERIFIED / assurance shows the L0 fail-safe while the coverage table shows its own explicit mismatch notice — honest, but reviewers should expect that combination on stale-revision candidates (same semantics as the P1-07 change page).
- The coverage *table* is duplicated as presentation markup between `v14_task.tmpl` and `v14_change.tmpl` (template stacks cannot share a partial without registering a shared define; facts come from one read path, so no semantic duplication).
- `/v14/changes/{csid}` still highlights nothing (pre-existing, unchanged) — its page was P1-07's; left to that lane/P2-03+ coordination rather than silent widening.
- 需人工 nav rename/extension to a Reviews aggregation remains P2-04's handover (unchanged from P2-01).

## 7. Decision Locks touched

None. All locks read; none bypassed, weakened, or reinterpreted. Relevant readings honored: DL-19 (pure read/projection; no state-machine change — no lifecycle transition, field, or setter added anywhere), DL-18 (existing domains extended in place; `internal/taskstate` zero-changed; P1-06/P1-07 acceptance/verdict semantics frozen and untouched), DL-04 (task page's snapshot, compiled authority, and coverage all read from the identical observed HEAD hash), DL-05 (risk stays derived; projection-only), §5 trust invariants (every VERIFIED-like rendering carries a real basis incl. `compiled:<hash>`; VERIFIED_LIMITED never flattened; unknown/unmatched signals preserved; nothing fabricated without Compiled; Agent claims still labeled claims). No shared hotspot touched (`server.go` unchanged; no §10 file edited). `hidden/` unread (DL-17).

## 8. Commit hash

Not committed — the orchestrator owns commits. Working tree contains exactly the five code/template files in §2 plus this report (plus unrelated parallel-lane files: `Makefile`, `internal/gitintegration/**`, `internal/gitwriter/**`, `runtimes/dsh/**`).
