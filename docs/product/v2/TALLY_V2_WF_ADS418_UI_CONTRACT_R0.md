# Tally v2 W-F — minimum complete ADS v4.18 UI workflow R0

**Owner:** [#92](https://github.com/kaicreator-mm/tally/issues/92), Umbrella #86. **Status:** UI_REQUIREMENTS_AND_NEGATIVE_FIXTURE_SOURCE / NO_RUNNING_APP / NO_HOST_TEST / NO_RELEASE_PASS. This is an App Shell for **selected DomainHarness Engineering B + one K + C GitHub/Local**, not an independent UI Task controller. ADS project effective pin still pending reviewed #85, though upstream ADS v4.18 source RQ READY.

## Required operator journey — complete minimum, not one Task-only demo

| View | What operator can do without raw JSON | B/K/C real evidence and safe hold UI |
|---|---|---|
| Projects / Repo | choose owning GitHub repo, permissions, historic/effective ADS pin | native readback repo grant, status stale/partial/UNKNOWN, no cached authorization |
| Requirements & analysis | editable structured requirement form, source links and versions | B Product draft; GitHub versioned issue/doc after K→C readback |
| PRD and review | draft/revise PRD, compare source, initiate independent Review, see P0/P1/P2 | only real external Reviewer/source SHA permits Freeze; no author self-PASS |
| Architecture L2 | inspect Product-approved contract, selected DH public B/S/K/C/Host ABI, L2 Review | no physical L2 Freeze without real public ABI and current source |
| Task DAG/Scaffold | display **parallel workfronts**, stage steps/subtasks, agent/CI/Local/Human fit, handoffs and native blockers | distinguishes planning graph vs actual GitHub Issue Dependencies and accepted Claim |
| Task Pack / L3 | forms for Tests, Contract, Implementation Plan, Failure Handling, Reference, expected source | incomplete L3 visibly NOT_ADMITTED; not a giant JSON editor |
| Agent assignment | choose real Codex CLI Local, read capability/role and verifier availability | `LOCAL_AGENT_UNAVAILABLE`, `BUILDER_CONFINEMENT_UNAVAILABLE`, `HIDDEN_VERIFIER_UNAVAILABLE` are distinct |
| Execution console | show single active Task attempt P0, clean worktree, process receipts, cancel/retry/repair | K state and Host actual terminal; no acceptance on mere ACK; UNKNOWN side effects explicit |
| Task evidence / review | source SHA, diff, real tests/CI, errors, independent Reviewer P0/P1/P2 | real GitHub Review/Validation; protected path diff triggers deny |
| Full/Hidden/Critical | actual OS verifier principal, held-out availability, reusable test matrix | NOT_RUN cannot become PASS; no hidden fixture visibility to Builder |
| Version/Release | candidate source/tag baseline, RQ, exact PR merge and main readback | independent RQ and permission gated; version branch complete before main |
| Experience & learning | bounded Task outcome/cost/coverage feedback from GitHub ADS v5 reports | missing task-meta telemetry UNKNOWN; no direct task-meta API/manifest fetch |

## UI → security/Kernel/effect boundary

All user actions follow App Shell/Host authenticated session/Origin+CSRF → Engineering B repo/resource/protected-path decision → **one K Primary** typed admission/effect intent → selected C provider actual grant/command/remote readback. A shell is a view and command transport; it cannot mint independent Task status, claim authorship, Review/PASS or change K Journal. No direct `gh` or GitHub write token in Browser/Builder process. Operator may authorize a protected effect only after actual scope and human confirmation; no fake Owner-account identity inferred from transport login.

## Six frontend panels may be built separately, not serially

Parallel UI source components: (U1) Project/Authority/ADS-version context; (U2) Requirements/PRD/Review; (U3) L2/Task DAG/Task Packs/Agent fit; (U4) Run/Repair/Verifier; (U5) Release/Tag; (U6) GitHub Task telemetry/Experience advisory. All reuse a common **read-only projection/typed command contract** instead of carrying their own B decision logic. Each can have independent static Story fixtures and source Review now; production wiring waits for B/K/C/Host selected public ABI and qualified Release input.

## UI negative/critical journey fixture matrix

| ID | User interaction / adverse case | Expected view + action |
|---|---|---|
| UX-N01 | Repo scope revoked during open session | redact stale material, deny subsequent GitHub Effect |
| UX-N02 | Cross-origin/CSRF or expired Host session | Host rejection, no K admission |
| UX-N03 | Product PRD changed since accepted Reviewer SHA | explicit stale review, cannot click Freeze |
| UX-N04 | ADS 4.18 version selected in Draft #85 but not adopted | `STANDARD_PIN_PENDING`, no Task/Release governed publication |
| UX-N05 | Local fake/absent Codex | `LOCAL_AGENT_UNAVAILABLE`, safe Project/PRD form still works |
| UX-N06 | Genuine Codex but Builder can access gh/GCM/SSH token | `BUILDER_CONFINEMENT_UNAVAILABLE`, no Start |
| UX-N07 | Hidden verifier account/ACL missing | `HIDDEN_VERIFIER_UNAVAILABLE`, Hidden HOLD not green |
| UX-N08 | Two parallel DAG nodes Ready and user starts both | only one K-executing Attempt; second NOT_ADMITTED |
| UX-N09 | Task claim not current but SQLite says ready | stale native GitHub readback wins, action denied |
| UX-N10 | L3 incomplete despite Issue Ready | L3_NOT_ADMITTED with missing required section inline |
| UX-N11 | Task running then Agent CLI stops unexpectedly | UNKNOWN/recover controls, no automatic Task PASS |
| UX-N12 | Repair changes protected Hidden/ADS/auth file | path deny warning, cannot publish/call PASS |
| UX-N13 | Reviewer is same logical executor, only GitHub login differs | independent Reviewer still missing; show BLOCK |
| UX-N14 | Full or Critical Journey NOT_RUN but all PR tests green | Release HOLD, `PR PASS != RELEASE PASS` |
| UX-N15 | GitHub write ACK timeout | UNKNOWN_EFFECT, reconcile native readback before retry |
| UX-N16 | ADS Task Report v5 schema mismatch or edited correction | UNSUPPORTED/STALE, data advisory only, never Task truth |
| UX-N17 | task-meta stops publishing, UI shows no token cost | UNKNOWN cost, never 0 and no Task interruption |
| UX-N18 | Access to private manifest/Trace via report link | UI must not dereference/show private record; GitHub visible refs only |
| UX-N19 | User requests version/main merge before RQ/Freeze | disabled with concrete missing authority; no bypass |
| UX-N20 | Session restart or different browser | reconstruct real GitHub source facts; do not trust prior chat/task-local cache |

## Minimal v2 Product acceptance and UX quality thresholds

- A normal operator can complete project→requirements→PRD→L2→DAG/L3→real Task→Review→Release without editing arbitrary machine JSON or knowing GitHub API internals. Advanced portfolio and Strategy optimizer remain optional P1; minimum journey must expose each ADS owner gate.
- No dashboard derives its own truth from local Task percent or telemetry; status projection exposes last observed source SHA/revision/time, coverage/currentness/UNKNOWN and actionable failure reason.
- No fake green RQ, no `NOT_APPLICABLE` invented for unrun Host/Hidden tests; any external currentness/permission gap returns source-qualified refusal.
- UI can be independently source-reviewed/Story-fixtured now; real accessibility/screen/browser E2E and Host/Windows/Ubuntu denied permission tests remain `NOT_RUN` until installed app. Real Human Protected DENY requires operator participation only where effect materially exists.

**Next:** Design Review and bounded App Shell interface sketch (Contract only) → implement parallel UI panels on selected B/K/C after public ABI/accepted Product+ADS+L2 → real browser/Host/Critical Journey test at version qualification. `UI_SOURCE_READY=REVIEW_PENDING`, `UI_RUNTIME_NOT_RUN`.
