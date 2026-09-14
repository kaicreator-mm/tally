# P2-01 Execution Report — Human-centered navigation projection

- **Status:** DONE
- **Date:** 2026-09-12
- **Baseline:** A0/P0/P1 complete (progress 22/57 at start); HEAD `c6767d7`
- **Depends-on:** P1-08 (landed) — no runtime dependency taken; this task is a pure read-side navigation projection.

## 1. Seam choice (narrowest valid seam)

**The navigation shell is `web/templates/base.tmpl` — already the single shared shell parsed by all four template stacks (`internal/web/render.go`, `handlers/v13_tasks.go`, `handlers/v14_launch.go`, `handlers/v14_change.go`) and the file PATH_OWNERSHIP assigns to P2-01 as "Exclusive navigation owner". Every page data struct embeds `handlers.Page`, whose `Active` field is the existing navigation-highlight key. The projection therefore lands entirely in base.tmpl plus two one-string `Active` key disambiguations in the two handlers whose pages the new entries target.** No route, no read model, no state machine, and no shared hotspot (`server.go` untouched) is changed.

### Exact source/tests read (reference order per the L3 pack)

1. **Tests:** `internal/web/handlers/handlers_http_test.go` (authed test-server harness: `newTestServer`/`newTestServerFull`, fake sessions, CSRF priming — reused as-is by the new HTTP test); `internal/web/render_test.go`; `internal/web/handlers/v14_task_test.go` / `v14_change_test.go` / `v14_attention_test.go` / `v13_http_test.go` (page-test idioms; none assert on `Active` keys or nav content — verified by grep, so the key disambiguation is regression-free).
2. **Contract/interface:** PRD §3.1 (human nav = projection/view over existing domain objects; "Domain State ≠ Navigation Structure"), §6.1 (human mental model Project→Work→Plan→Run→Review), §26.2/§26.4 (v1.3 Task view / Attention projection), §33 Phase 3 (Work/Runs/Reviews/Rules as projections, no state-machine change); DL-19 ("Human-centered navigation is a read/projection concern and must not rewrite the state machine"); DECISION_LOCKS table and binding interpretations read in full; `SOURCE_BASELINE.md`, `PRD_AMENDMENTS.md` read per task discipline.
3. **Core implementation:** `web/templates/base.tmpl` (existing rail structure, `aria-current` highlight idiom, honesty sub-labels), `internal/web/router.go` + `internal/web/handlers/server.go` `RegisterRoutes` (full existing GET-route inventory), `handlers/v13_tasks.go` (`/v13/tasks` — Task(TaskContract) view over Git artifacts + fact store, derived state, nil-safe "未接入" empty state), `handlers/v14_attention.go` + `v14_change.go` (`/v14/attention` — Attention projection from derived blocking facts + exposure ledger, nil-safe), `handlers/v14_launch.go` (`/v14/launch` — structured intent intake + closed-loop task list, nil-safe `Unavailable` path), `internal/web/render.go` (pageNames registry, funcMap).
4. **Failure handling:** a nav entry that would point at a concept without a real, read-model-backed route must not exist (task requirement 2) — enforced by test rather than by convention.

### Route audit backing each entry (evidence for "backed by real data")

- **发起 → `/v14/launch`** — existing GET route (`RegisterV14Routes`); page renders the structured (non-chat) intent entry and the closed-loop task list read from Git artifacts + fact store; with sources nil it renders its documented `Unavailable` state (200, honest empty state, no fake data).
- **工作 → `/v13/tasks`** — existing GET route; the v1.3 engineering Task view: `.tally` TaskContracts from the pinned HEAD snapshot + CodegenRun/ChangeSet/VerificationRun/HumanApproval fact rows, state derived by `taskstate` (INV-STATE-01); nil-safe empty state.
- **需人工 → `/v14/attention`** — existing GET route; PRD §26.4 Attention projection: only genuinely human-needing items (needs_review / blocked / failed_run / cancelled / exposure_compromise / honest projection_error), derived, read-only.
- **Runs — deliberately NOT added:** no parameter-less read route over run facts exists today (runs are only visible per-candidate under `/v13/changes/{csid}` / `/v14/changes/{csid}`); the Runs projection page is owned by P2-03 (W16).
- **Rules — deliberately NOT added:** no EngineeringPolicy read route exists anywhere in the web tier; rule governance UI is owned by the P3 lane.
- **Reviews as a full aggregation — deliberately NOT fabricated:** `/v14/attention` is the existing backed read model closest to the PRD "Reviews" concept (P2-04 states it will reuse "approval/review/attention" facts); the entry is labeled 需人工 (needs-human) — exactly what the page shows — not "Reviews", so no capability is implied that does not exist. P2-04 may rename/extend when its aggregation page lands.

### Why this is the narrowest valid seam

- `base.tmpl` is the one file every rendered page already flows through; adding entries there requires zero new plumbing, zero new data flow, zero new types.
- Highlight correctness uses the existing `Page.Active` mechanism. The two v1.4 pages targeted by new entries shared the coarse key `"v14"` with pages they are not (`/v14/tasks/{task}`, `/v14/changes/{csid}`), which would have made 发起 and 需人工 highlight simultaneously on either page; two one-string key splits (`"v14-launch"`, `"v14-attention"`) fix that without touching any other handler (`v14_task.go`/`v14_change.go` keep `"v14"`, which matched nothing before and matches nothing now — status quo preserved).
- Work keeps the existing key `"v13"`: all three v1.3 pages (tasks list / change detail / review) are Work-domain views, so 工作 highlighting on them is semantically correct.

### External reference used

- None beyond the pack's EXT-GO-HTML (Go `html/template` context-aware escaping — standard library behavior already relied on by every template; no new code pattern was needed, no external code copied, no new dependency).

## 2. Changed files

- `web/templates/base.tmpl`: new 研发 nav group (after 项目, matching the PRD Project→Work flow) with three entries — 发起 `/v14/launch`, 工作 `/v13/tasks`, 需人工 `/v14/attention` — plus a comment recording the projection discipline (read-only, DL-19, entries only for real read-model-backed routes, Runs/Rules deferred to their owning tasks). No existing line changed; existing groups byte-identical.
- `internal/web/handlers/v14_launch.go`: one string — `v14LaunchPage` Active key `"v14"` → `"v14-launch"` (launch page highlights 发起 only).
- `internal/web/handlers/v14_attention.go`: one string — `V14Attention` Active key `"v14"` → `"v14-attention"` (attention page highlights 需人工 only).
- `internal/web/nav_test.go` (new): template-level contract tests (below).
- `internal/web/handlers/nav_http_test.go` (new): HTTP-level route-integrity and highlight-wiring tests (below).

`server.go`, `router.go`, `render.go`, all page templates, and every domain package: untouched.

## 3. Semantic changes

- The v1.3/v1.4 engineering loop (structured intent intake → Task work view → needs-human projection), previously unreachable from the sidebar, is now navigable under human mental-model labels (PRD §6.1) instead of internal vocabulary — satisfying the direction P2-05 will gate ("structured decision flow is primary; internal state-machine terminology does not dominate navigation").
- Navigation is a pure projection: read-only links; no new route, handler state, template state, or persistence; no state machine touched (DL-19; PATH_OWNERSHIP "No UI task may alter task lifecycle transitions simply to simplify rendering").
- Highlight is truthful per page: exactly one 研发 entry is current on `/v13/tasks`, `/v14/launch`, `/v14/attention` respectively; v1.4 task/change pages keep the pre-existing no-highlight behavior.
- Absence is explicit: Runs and Rules have no entries, with the reason and owning tasks recorded in the template comment — a missing capability is visible as an absent entry, never as a placeholder.

## 4. Tests / commands executed (exact)

| Command | Result |
|---|---|
| `go build ./internal/web/...` | PASS |
| `go vet ./internal/web/...` | PASS |
| `go build ./...` (whole repo incl. parallel-lane files) | PASS (exit 0) |
| `go test ./internal/web/... ` | PASS (ok 0.295s web; ok 17.192s handlers) |
| `go test ./internal/web/ -run "TestNav" -v` | PASS (5/5 new tests) |
| `go test ./internal/web/handlers/ -run "TestNav" -v` | PASS (2/2 new tests) |
| `go test ./test/arch/...` | PASS (ok 1.3s) |
| `DATABASE_URL=postgresql://postgres:postgres@localhost:15432/postgres go test ./internal/web/handlers/ -count=1` (full package, DB-gated included) | PASS (ok 116.1s) |
| `DATABASE_URL=postgresql://postgres:postgres@localhost:15432/postgres go test ./internal/web/ -count=1` | PASS (ok 0.26s) |
| `gofmt -l` on the four touched/new Go files | clean (no output) |
| `git diff --check` | clean (only pre-existing CRLF-checkout warnings on working-tree files; `git diff` confirms zero line-ending churn — the only change to base.tmpl is the 10 inserted lines) |
| `make check` | NOT_RUN (no make binary in Git Bash; constituent families run individually as above) |

### New tests

Template level (`internal/web/nav_test.go`, renders base.tmpl directly through `assets.TemplateFS`):

- `TestNavProjectsHumanEntriesOntoExistingRoutes` — the three entries exist with their human labels bound to the correct hrefs; no empty hrefs.
- `TestNavHighlightFollowsActiveKey` — for `Active` ∈ {`v13`, `v14-launch`, `v14-attention`}, `aria-current="page"` lands on exactly the corresponding entry and on no other 研发 entry.
- `TestNavHasNoPlaceholderEntries` (negative) — no `/runs`, `/v14/runs`, `/rules`, `/policies`, `/reviews` entries and no placeholder copy ("即将上线"/"敬请期待"/"coming soon"/"占位") may appear in the nav.
- `TestNavHrefsAreKnownRoutes` (negative/structural) — every href rendered in the nav must belong to a declared set of real routes; adding a nav entry without a real route turns this red first.
- `TestNavTemplateParseFSIsComplete` — base.tmpl defines both `base` and `content` and exists in the embedded FS (guards against silent-empty ParseFS matches).

HTTP level (`internal/web/handlers/nav_http_test.go`, real router via the existing `newTestServer` harness):

- `TestNavLinksResolveToRealRoutes` — extracts every href from the rendered rail of `GET /` and GETs each through the real router: any 404/405/5xx fails. This is the executable form of the acceptance criterion "Existing routes remain valid; navigation maps to existing/read-model data" and simultaneously the placeholder guard: a nav entry with no backing route cannot pass. Also asserts the three 研发 entries are present, and (with nil sources) that every target answers 200 via its documented honest empty state.
- `TestNavHighlightWiringOverHTTP` — end-to-end highlight wiring: on `/v13/tasks`, `/v14/launch`, `/v14/attention` respectively, exactly the matching entry carries `aria-current="page"` (this also exercises all three alternate template stacks — v13 funcs, v14 launch funcs, v14 change/attention funcs — parsing the modified base.tmpl).

## 5. Baseline failures observed

None. All pre-existing tests pass unchanged; no baseline failure needed distinguishing. (Parallel-lane working-tree edits — `cmd/worker/**`, `internal/gitintegration/**`, `internal/verification/*_test.go` — were present in `git status`; none were touched, and the repo-wide build/test commands above included them without error.)

## 6. Remaining risks

- **Labels are Chinese human terms (发起/工作/需人工).** P2-02/03/04 will land the dedicated Work/Runs/Reviews pages; P2-04 may want 需人工 renamed (e.g. to a Reviews aggregation) and P2-03 will add the Runs entry. The template comment marks those extension points so the owning tasks find them.
- **/v14/tasks/{task} and /v14/changes/{csid} highlight nothing** (pre-existing behavior, unchanged). P2-02 owns the task-detail projection and may introduce finer keys.
- The known P1-07 report note that the v14 task page's basis line lacks `compiled:<hash>` is untouched here (its owning lane's follow-up).

## 7. Decision Locks touched

None. DL-19 is complied with (navigation is read/projection-only; no state machine change). All locks were read; none bypassed, weakened, or reinterpreted. No shared hotspot touched (`server.go` unchanged, per §10).

## 8. Commit hash

Not committed — the orchestrator owns commits. Working tree contains exactly the five files listed in §2 (plus unrelated parallel-lane files).
