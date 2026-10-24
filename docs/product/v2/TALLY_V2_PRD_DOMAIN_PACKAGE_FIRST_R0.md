# Tally v2 Product PRD — native Domain Package Engineering System Center (R0 candidate)

**Date:** 2026-10-24  
**Review state:** PRODUCT_DRAFT_R0 / INDEPENDENT_PRODUCT_REVIEW_NOT_RUN / FREEZE_READY=NO  
**Owning Issue:** [#76](https://github.com/kaicreator-mm/tally/issues/76), parent [#63](https://github.com/kaicreator-mm/tally/issues/63), [L2 #65](https://github.com/kaicreator-mm/tally/issues/65), [Domain Package-first correction #75](https://github.com/kaicreator-mm/tally/issues/75#issuecomment-6380546957)  
**Legacy Product evidence:** [F01–F47 matrix](../../research/v2/TALLY_V2_FUNCTIONAL_PARITY_R0.md), pinned v1.5.5 source b5e7b6e38ea38837e3bbcfc46c36586180705de0.  
**Standards adoption note:** Tally historical main pin is v1.5.0-specific ADS 1.2.1; this prospective Tally v2 source branch now contains a DRAFT-only canonical ADS v4.17.0 root pin via [PR #81](https://github.com/kaicreator-mm/tally/pull/81), **not an authorized effective v2 adoption or Product Freeze**. ADS v4.17.0 is now SOURCE_RELEASE_CLOSED_CONDITIONAL at a45644d734fdf94da3e6f9c3f90ffe4ce3c25fda; Tally v2 proposed adoption #80 remains NOT_ADOPTED. v4.16.0 is fully qualified conservative fallback. Select and lawfully adopt **exact immutable qualified compatible ADS revision** for v2 before executable Task/ADS report publication. ADS v4.15 source-only reports may be test fixtures only until appropriate adoption. Do not use floating main as authority.  
**Platform dependency:** DomainHarness v0.10.2, actual independently installable Domain Packages/public ABI/BUILD/Host; physical qualification pending [DH T09 #1330](https://github.com/kaicreator-mm/domain-harness/issues/1330) and [R01 #1331](https://github.com/kaicreator-mm/domain-harness/issues/1331).

### Candidate ADS standard decision — explicit source release and project adoption separation

[Tally v2 adoption gate #80](https://github.com/kaicreator-mm/tally/issues/80) proposes the **latest source-qualified ADS v4.17.0** on immutable integration `main@a45644d734fdf94da3e6f9c3f90ffe4ce3c25fda`, qualified tree `d1059bef9d5f74ad607c68c0a08d5fb9836c6fa5`. Upstream RQ at [ADS #1311](https://github.com/kaicreator-mm/ai-development-standard/issues/1311#issuecomment-6381478181) is **CONDITIONAL P0/P1/P2=0/0/3**, not full runtime PASS. Residual P2 covers legacy pinned Python 3.14 portability, manifest coverage of new docs/tests, and **unproven external task-meta/Tally/Host/native/human-DENY E2E**. Qualified ADS **v4.16.0@718fb02e33b31f568736447b2b905883cc6fcc3a**, source-release PASS P0/P1/P2=0/0/0, is a conservative compatible fallback if required.

`ADS_V2_CANDIDATE_PIN=4.17.0@a45644d734fdf94da3e6f9c3f90ffe4ce3c25fda` is a **Product preference, not an adopted project standard**. Historical Tally main `.dev-standard/VERSION` is ADS 1.2.1 for v1.5.0; this stacked candidate includes prospective v2 ADS 4.17.0 canonical identity and one A1 manual adoption level, but **PR #81 remains DRAFT**, §2.1C(2) demands `READY` while the observed upstream v4.17 RQ is `CONDITIONAL`, and independent project compatibility/activation remain unresolved. Before actual v2 Product Freeze, [#80](https://github.com/kaicreator-mm/tally/issues/80) must receive independent source/profile/P2 review and the legal project-local version adoption action, followed by actual qualified GitHub Freeze artifact readback. No controller may silently switch pins or claim v4.17 downstream runtime E2E PASS from an ADS standards-source release.

## 1. Product objective and delivery definition

Provide one approachable engineering cockpit in which users (with Agents) can take a repository and version through the **entire simplified ADS development lifecycle**, from requirements through independently verified release, while preserving actual GitHub authority and honest verification evidence.

**Hard delivery constraint:** The **executable** Tally v2 application is composed from **DomainHarness selected Domain Packages**, including Tally-owned Engineering Business Package(s) and Tally-owned or selected Capability Packages, with DomainHarness K Primary/selected Standard S/Host. It is **not** a standalone TypeScript service later adapted/wrapped. Existing Go code is evidence for user-visible capability, not a source compatibility requirement.

PRD success requires an actual Node24 clean installed **external producer → Package BUILD → selected Domain App → K-admitted Local Agent/Tool → independent verification/GitHub fact readback** path. A source-only model, Demo/Mock, UI click sequence or packaged tarball alone is not v2 Runtime PASS.

## 2. Actors, authorization and project scope

- **Operator** chooses project/version/ADS revision, confirms requirements, responds to actual required authority decisions, inspects progress/blocks and release gates.
- **Logical Planner/Builder/Reviewer/Verifier/Repairer Agents** are distinct by declared task role/run/session and pinned provenance; same GitHub transport identity must not collapse them, but real provider grants still apply.
- **GitHub provider** is authority of actual Issue/PR/Review/merge/commit/tag and version-qualified ADS-published facts.
- **DomainHarness K Primary** owns Domain App State, Admission/CAS, Journal/Effect, replay, UNKNOWN; **selected S** applies reusable Rule/Context/Decision/Workflow as bounded services. Engineering B owns domain rules and user decisions, not duplicate platform effects.
- **task-meta** is independent collector/publisher of optional-at-execution telemetry: its failure affects only coverage, cost/learning/optimization advisory, **never Task start/run/verification/completion**. Tally does not directly read task-meta API/DB/Manifest/Trace.
- **Local host** is mandatory target for the basic single-Task and version journey. SSH/Docker/runx/remote Host are *optional* Capability providers.
- Single repository / single version is the mandatory basic case. Cross-project portfolio and multi-Agent broad scheduling may be added without making core closure depend on them.

## 3. P0 in-scope requirements (MUST)

| PRD ID | Product requirement | Minimum externally observable acceptance |
|---|---|---|
| P0-01 | Package-native composition | Real public DomainHarness-installed B/S/K/C/Host package/BUILD separation, selected pin, no private source deep imports, no alternate K implementation |
| P0-02 | Product & requirements UI | Capture project intent, facts, constraints, drafts, clarifications, requirements/PRD revisions, explicit decisions and source links in UI |
| P0-03 | Product Review/Freeze gate | Drafts live in B/K only; only an applicable independent, source-qualified ADS review/Freeze artifact **published to GitHub and read back** establishes the external Freeze fact; unresolved P0/P1 or no applicable adopted governing standard ⇒ HOLD, never pretend to Freeze |
| P0-04 | Architecture L2 UI and gate | Draft/revise L2, authority/dependency/negative scenarios, independent Review/Freeze, no direct Task admission while Product/L2 not approved |
| P0-05 | Task DAG, L3 contract and Issue publication | Decompose Tasks/dependencies, reject cycles and stale DAG, publish Issue and version-qualified L3 Tests / Contract / Implementation / Failure Handling / Reference evidence, re-read exact accepted source before executable dispatch |
| P0-06 | Named real Local Agent dispatch | **Codex CLI** is the mandatory first real Local Agent C provider class (preinstalled/authenticated by operator); start, cancel, timeout, terminal status/exit, log/receipt and workspace identity must be observable under actual K; Claude Code may be optional. Fake/echo is TEST_FIXTURE_ONLY and cannot satisfy CJ03 |
| P0-07 | Repair, protected-path controls and independent verification | Explain evidence-backed failure, construct bounded retry/replan, preserve accepted work, reject unauthorized edits to protected ADS/hidden-test/auth files, require provenance-qualified independently performed review and real checks |
| P0-08 | GitHub authority and ADS reports | Consume Issue/PR/Review/CI/host facts and source-qualified ADS report, handle schema version, corrections, retries, stale source and true readback |
| P0-09 | SQLite derived-cache discipline | Faster GitHub projections and search; timestamp/source coverage/currentness and invalidation; no cache-based approval for irreversible native fact change |
| P0-10 | task-meta degraded execution | Collector fail/timeout/report gap **and in-process/wrapper hook hang/nonzero exit** do not block Task; fail-open bounded observation with no awaited telemetry publisher and no changed Agent exit status, metrics UNKNOWN; quality/effect gates unaffected by telemetry ACK |
| P0-11 | Version closure & source-pinned release UI | Keep Task vs version gates separate; Full/Hidden/CJ/RQ must bind exact candidate commit/tree and coverage, verify target branch currentness before merge, revalidate merged commit/tree and tag target, re-RQ when verified content changes |
| P0-12 | Agent/role and authority controls | Agent registry/capability/currentness; logical role distinction, permission-bounded delegation, required user decision only where applicable |
| P0-13 | Safe default UI | All steps reachable without editing raw JSON/SQLite or shell scripts; actionable blocked reasons with source; no decorative success on mock |
| P0-14 | Cross-session GitHub durability | Every substantive decision/report/review/dispatch has canonical GitHub Issue/PR/document/ref, source SHA and recoverable handoff, independent of chat |
| P0-15 | Operator/session, repository scope and data security | Authenticated operator action+resource authorization, session expiry, origin/CSRF protection, actual GitHub provider grant AND K permit, protected-file restrictions and secret redaction; neither valid JSON nor K-only permit bypasses provider/scope checks |
| P0-16 | Safe Local workspace and Git writes | One isolated disposable worktree per Task Attempt from pinned clean base; no dirty source/concurrent writer, frozen candidate SHA/tree before verifier; **the Builder subprocess has no GitHub write credential or ungoverned push/PR/merge/tag path**, with K-admitted C GitHub as the sole publisher and exact readback | 
| P0-17 | Honest held-out/Hidden verification isolation | Default single-machine Local verifier is a **separate OS account/principal with filesystem ACL deny-read witnessed against Builder**, with fixtures staged only after candidate freeze; no Docker/runx required. Missing isolated verifier ⇒ HIDDEN_VERIFIER_UNAVAILABLE / no Hidden PASS; container/remote alternatives optional |

### Domain-specific behavior to retain

- RequirementsBaseline, ProjectPlan/PlanningRevision, ArchitectureRevision and explicit provenance (audit F01–F05).
- TaskDAG DAG-cycles, readiness/parallel bounded scheduling, CompletionContract, required human authority and negative gate reporting (F06–F09, F13–F15).
- Evidence/assurance/verification coverage, hold-out isolation, candidate and workspace safety, repair and independent review (F10–F12, F21–F27, F47).
- Actual GitHub engineering operations with readback and source pin, stale-parent replan and version release audit (F28–F30).
- Conservative strategy/Context/error diagnosis and verified lessons when source evidence exists (F31–F36).
- Consistent UI experience from older natural-language launch and v1.5 lifecycle surfaces, **not** direct reuse of obsolete v14 job workflow (F42–F44).

### Source of authority for lifecycle artifacts (Product decision; not guessed ABI)

| Object/status | Authoritative fact | Projection/draft | Required proof / applicable stage |
|---|---|---|---|
| Work-in-progress Product/PRD/L2/DAG proposal | B application intention under selected K State | UI may show DRAFT; GitHub links optional until publish | Cannot assert reviewed or frozen |
| Published PRD/L2/Task specification | GitHub commit/Issue/PR/source-qualified ADS artifact and actual readback | B/K holds revision/reference for UI | Current exact SHA, correct project/version and GitHub object identity |
| Independent Product/L2 Review and Freeze | **Actual applicable GitHub ADS-governed review/Freeze evidence** plus accepted source SHA and reviewer provenance | B/K stores projection and requested command, not a second Freeze authority | If GitHub source absent/stale ⇒ REVIEW_PENDING/UNKNOWN; cannot infer from K State or just comment text |
| Task/Claim/L3/Execution evidence | Actual native Issue/PR/report facts scoped to selected ADS version plus actual K/Host receipts | Tally may produce drafts before publication | Must pass version-qualified readback/currentness/authority gate |
| Version Release/RQ/merge/tag | Source-qualified independent RQ and actual GitHub merge/tag commit/tree | B/K only project/release projection | RQ complete on eligible source; merge/tag readback |

**Standards pin stage rule:** At Product **draft/pre-review**, reference v2's *proposed exact compatible* ADS pin and evaluate candidate without claiming an effective normative Freeze. **Publishing a Product Freeze** requires project-level legal adoption/qualification of the exact ADS governance revision applicable to v2 at that stage; if not yet adopted, status is `PRODUCT_REVIEW_CANDIDATE` / `STANDARD_PIN_PENDING`, **not** `FROZEN`. Formal reviewer may report `FREEZE_READY_CONDITIONAL_ON_ADS_PIN` with explicit blocker, but no controller may turn it into `PRODUCT_FROZEN` by updating B/K alone. Task and Release publication later require their own adopted compatible report/protocol revision. Do not retroactively apply an unqualified v4.17 draft or assume v1.5.0's 1.2.1 standard is valid for v2.

### Minimal retained legacy behaviors — bounded P0, not whole Go subsystems

These are **new v2 user-observable requirements**, not evidence that historic Go implementation is fully reachable. Source reachability of these legacy domains remains `UNKNOWN` in the F01–F47 evidence matrix.

| Legacy domain | MUST preserve in P0 | Positive / negative acceptance | Explicitly NOT required in P0 |
|---|---|---|---|
| F02 requirement clarification | One concise clarification/question→answer→requirement revision with unresolved decision list and proposer/reviewer provenance | User resolves one blocking ambiguity and sees changed PRD; unresolved critical question blocks Product Freeze (V31/V46) | Full transcript interview engine, automated signature matching |
| F03 frozen intent/risk | Frozen accepted intent/source revision with scoped risks; change of materially accepted intent invalidates applicable acceptance until reviewed | Intent changed after accepted freeze ⇒ new eligible review/revalidation (V32/V47) | Comprehensive intent-classifier or legacy risk-scoring internals |
| F11 risk/assurance | Select a bounded assurance level from declared task risk and expose resulting verification/authority gates | High-risk scope with inadequate verifier ⇒ gate blocked (V33/V48) | General-purpose assurance optimizer or automatic risk model |
| F12 contract/policy | Validate TaskContract against selected **actually adopted** ADS dialect and required evidence fields, reject missing/unsupported schema | Unsupported ADS/L3 fields at admission ⇒ UNSUPPORTED/BLOCKED (V34/V49) | Migrating a generic Go policy compiler or all historic compatibility profiles |
| F26 reviewer & protected paths | Protected source paths/Review independence/provenance are nonoptional; denial and escalation visible | Builder edits guarded file without authorization or reviewer self-approves ⇒ BLOCK (V35/V36) | Rich reviewer dashboards, AI review rankings, extended UX |

Existing F02/F03/F11/F12 `LEGACY` files are **not required to be copied**. If a later source audit shows a valuable omitted legacy subcapability, handle it through a Product amendment, not silent expansion of P0. Preserve their current `PROD_REACHABILITY_UNKNOWN` until qualified evidence exists.

### Protected engineering sources and logical reviewer independence (P0)

Treat ADS governance files, hidden/held-out fixtures, authentication/authorization policy and other project-declared protected paths as **protected-by-policy**, with exact scope and authorized exception. Any unauthorized change requires a deny/escalation record, not silent acceptance. Independent review is based on **declared logical execution identity and subject/source evidence**, not merely different GitHub accounts or a switch in displayed Agent role; self-review via role swapping within the same execution lineage is invalid. Actual GitHub protection and grants remain binding; Tally does not invent GitHub owner identity approval.

### Operator-session and provider effect authorization (P0)

All UI mutations require valid operator session, correct repo/project scope, intended action authorization, origin/CSRF protection where browser transport applies, matching GitHub provider permissions **and** a valid K-governed effect grant. Reject wrong repo grant, expired/revoked session/credentials, cross-origin forgery and mismatched scope **before any external mutation**, even where the request body is valid. Re-read actual remote target/current source prior to applicable irreversible actions. Architecture chooses mechanism; Product contracts only specify enforceable outcomes.


## 4. P1 and deferred scope (not a core gate)

**P1 candidate:** supported Agent/Model cost comparisons with source coverage; error fingerprints/context suggestions, verified experience/skill selection and feedback; cross-project Blocked/Ready/Cost view; **concurrent** multi-Agent dispatch beyond one executing Task at a time. P0 may compute/display the full parallel-ready set but **max simultaneously executing Tasks = 1** for minimal version. Must not delay minimum single-project release.

**DEFER:** remote Runx Gateway/SSH instance broker as a core dependency, automated Package learning→promotion, broad recurrence/recursive optimization, configurable generic BPM, heavyweight distributed scheduler, full multi-project Portfolio/enterprise governance, rewriting old Go components, universal deployment environments, maintaining pre-1.0 Go ABI compatibility.

**No-goals:** Tally owns neither a second Kernel nor a generic Agent SDK/Tool runtime, ADS normative schema author, a private task-meta raw-data adapter, or GitHub's native Task truth.

### P0 Local Agent provider, workspace and concurrency contract

**Mandatory production provider:** a real installed and authenticated **Codex CLI** local agent via a Tally-owned governed C adapter (new implementation, not a Go adapter). It must support capability probing and explicit `start/cancel/timeout/terminal receipt/exit code/stdout+stderr redaction/workspace identity`. If user credentials/provider binary are missing, show `LOCAL_AGENT_UNAVAILABLE` and do not claim a completed Task; basic Product editing and other lawful UI views still operate. Claude Code may be added as optional alternative, but an echo/Fake provider is **test-only** and does not qualify the first Local E2E. This is a Product provider-class commitment; actual DH C SDK API and process isolation mechanism remain L2/BUILD pending.

**Workspace safety:** each Task **Attempt** gets an isolated disposable Git worktree created from the exact accepted clean base SHA; no reuse of the mutable primary working checkout as an execution target. Reject unexpected uncommitted changes, wrong upstream SHA, overlapping write grants and a second concurrent writer to one checkout, with a recoverable BLOCK explanation. Candidate source changes become frozen commit/tree before Verifier access; accepted worktree provenance must be preserved on Repair/Replan and any changed tree demands fresh verification. The initial minimal product may show many parallel-ready Task nodes but schedules **exactly one executing Task Attempt at a time** (P0); concurrent multi-Agent scheduling is P1.

**Cancel and partial side effects:** after Start, Cancel is only `CANCEL_REQUESTED` until actual Agent/process-child termination and K/Host/GitHub effect receipts are reconciled. The Agent may have locally produced commits, while **only previously K-admitted C GitHub** may have published remote branches/PRs; B must show these receipts and preserve UNKNOWN where terminal outcome cannot be established. Cancel/timeout never auto-confers PASS nor assumes rollback of external writes. Actual OS process/group cleanup is a C/Host mechanism to be verified physically.

### Builder GitHub effect confinement (P0, chosen model A)

**Selected Product rule:** The Codex CLI/Builder child process receives **no GitHub write token, credential helper, SSH push key or inherited privileged `gh` session**. Its allowed output is local workspace changes and a frozen candidate commit/tree; changes to GitHub native Issue/branch/PR/Review/merge/Tag flow only via the **K-permitted C GitHub publisher** with scoped target, expected source SHA, immutable effect receipt and remote readback. Forbid direct `git push`, `gh pr create/merge`, branch force-push, or alternate credential mechanisms from the Builder process; treat an attempted or observed bypass as `UNGOVERNED_EFFECT`, block Task acceptance, revoke/reconcile scope and retain incident evidence. Denial must be actually witnessed in the host environment, not assumed merely because environment variables were scrubbed. **Probe must include the actual Builder principal's noninteractive privilege escalation and reachable credential stores/agents** (e.g. sudo -n, administrator/backup privileges, GH config, Git credential manager, SSH agent and delegated tokens); Builder must be a least-privileged account/security principal without a path to verifier fixtures or GitHub write credentials. If this isolation cannot be witnessed, the provider reports `BUILDER_CONFINEMENT_UNAVAILABLE` and Task/CJ03/Release may not claim PASS, while Product editing and other permitted read-only UI functions stay usable. Actual credential isolation/network subprocess policy and GitHub effect implementation remain pending qualified L2 Host/Capability integration. No privileged GitHub operation can be silently performed by an Agent's shell.

### Hidden validation trust boundary on a shared Local host

**Minimum Local verifier class:** operator-provisioned **separate OS account/principal with ACL-protected verifier-only workspace and an actual denial witness** (on Ubuntu separate UID/filesystem permissions; on Windows separate security principal/ACL). This is the default P0 **single-machine path without Docker, SSH or runx**. Product setup checks that the verifier identity/workspace is available; otherwise it displays `HIDDEN_VERIFIER_UNAVAILABLE`, continues requirements and other valid UI tasks, but does **not** claim Hidden PASS or release eligibility. An appropriately isolated container or remote verifier may substitute later, but is never a mandatory dependency for Local.

After the candidate commit/tree is frozen, materialize held-out fixtures solely in that verifier-owned workspace/identity not exposed via Agent worktree, environment, logs, cache, command-line or reachable filesystem paths. A Builder that can read any hidden fixture before/during its run **including via noninteractive sudo/admin token/backup privilege or inherited credential agents** invalidates Hidden evidence. With a same-OS-user shell that can freely read verifier-owned paths, simply using two directory names does **not** satisfy this guarantee: use enforceable filesystem/permission/user/container/remote verifier isolation with witnessed denial, otherwise `HIDDEN_NOT_QUALIFIED` and RQ cannot treat Hidden as PASS. Any project-specific weaker assurance must be expressly allowed by the adopted ADS, identified in its release gate and independent RQ, never silently re-labelled PASS. Local-only stays mandatory without requiring runx.

### Telemetry hook fail-open requirement independent of Agent process

A task-meta observer may run as an Agent hook, wrapper, separate collector or report publisher. **All** observation seams must be non-authoritative and fail open under crash, hook non-zero exit, queue-full and **indefinitely hanging in-process/CLI hook**. A bounded adapter execution timeout/non-blocking queue must keep the Agent's own start/exit status and independent verification path intact. Do not synchronously await a telemetry reporter before dispatch, before Agent terminal receipt, in the verifier callback or before Task/RQ eligibility; provider hook failures must not be reclassified as legitimate Agent command failures. If this guarantee cannot be shown for a selected Agent integration, disable/bypass that telemetry hook and report `TELEMETRY_UNAVAILABLE`, or leave the production provider unqualified rather than ship an execution-blocking observer. Positive/negative physical tests required later; no proposed hook mechanism here pretends to be tested.

## 5. Basic complete ADS UI journey (P0)

The UI is a **one-project/version workflow**, with stage navigator, a concise Markdown/structured edit pane, Issue/PR/evidence source links, review action strip and explicit missing gate reasons. It need not be a generic BPM designer.

| UI phase | User sees and can do | Source of authority | Gate / negative action |
|---|---|---|---|
| U01 Project setup / evidence | Select repo/ADS exact pin/version; collect intent, sources, open questions | GitHub repo + B draft via K | Missing repository grants or invalid pin blocks publication, not draft editing |
| U02 PRD | Structured requirement editing, acceptance and diff vs previous revision, issue linking | B Product revision + GitHub publication | Conflicting/unsourced critical assumptions stay OPEN |
| U03 Product Review/Freeze | Assign independent Product review, correct findings, record exact accepted SHA and GitHub readback of applicable Freeze artifact | **GitHub source-qualified review/freeze artifact**, never a B/K-only draft | No qualifying governing ADS pin, independent review or remote readback ⇒ HOLD; reviewer-self-approval/P0/P1 unresolved cannot PASS |
| U04 L2 | Edit authority/Package boundary/host/execution architecture, negative contracts, independent L2 review | B Architecture + selected standards; GitHub readback | Package ABI marked OPEN until real DH; cannot freeze physical claim by guess |
| U05 Task DAG | Create nodes/dependencies, split, cycle-check, publish Issue, see actual GitHub readback | B DAG proposal + native GitHub Issue graph | Cycle, missing authority/currentness or partial native graph blocks Task admission |
| U05-L3 Task specification | Generate/edit version-qualified L3 Tests, Contract, Implementation plan, Failure Handling and Reference; fix stale/missing fields and read back durable GitHub evidence | B TaskContract and native ADS GitHub L3 evidence; K/C publication | **Ready DAG ≠ executable L3**; missing, unreviewed where required or stale pre-admission L3 blocks Dispatch; post-run Implementation Evidence is a different completion receipt |
| U06 Execute | Select real Codex CLI Agent/isolated Local attempt workspace, review L3, dispatch/start/cancel/reconcile partial effects and inspect actual exit/receipts | K admission/Effect + C Local + actual Host receipts | No runx required; L3+Claim+scope+clean current source required; cancel/timeout leaves reconciliation, not PASS; no Fake Agent for release |
| U07 Repair / Verify | See actual failure fingerprint, Repair option, PR/review/verifier and narrowed rerun | GitHub/Host/independent review facts | Agent says PASS ≠ verifier PASS; no speculative self-approval |
| U08 Version closure | Show exact candidate SHA/tree, full regression/Hidden/CJ and independent RQ, perform eligible version-branch→main merge, compare merged SHA/tree and tag target | Actual GitHub native merge/tag facts and independently qualified Release evidence | **Main drift / changed merge tree / stale RQ ⇒ revalidate or BLOCK before tag**; PR PASS ≠ Release PASS |
| U09 Experience | task-meta reported coverage/Token/Cost/learning feedback when qualified | Source-qualified ADS GitHub reports | Missing/unsupported evidence means UNKNOWN, never Task failure |

**UI simplification acceptance:** A new user can finish the basic journey from project selection to version qualified Tag using UI forms plus the genuine required Agent/reviewer/Host execution, without hand-authoring TaskExecutionRequest JSON. Verifications are genuinely performed; no UI click claims they occurred.

### L3 before dispatch; Implementation Evidence after execution

Task DAG readiness is necessary but insufficient. Before dispatch, the selected ADS dialect's **L3 specification** must be attached/published under its actual GitHub source, with Tests, Contract, Implementation plan, Failure Handling and Reference material and verifiable accepted revision. Missing material, expired reference, unsupported report dialect or required review gap exposes a recoverable `L3_NOT_ADMITTED` reason in UI. A Product-style additional approval layer is **not** invented where ADS only requires admissible evidence. After real execution, Implementation Evidence includes actual source changes, commands, exit codes/receipts and failure dispositions; it cannot be pre-certified by the plan. The separate verifier/PR/CI gates still apply.

## 6. Business/Package semantic ownership (not the physical v0.10.2 ABI)

- **Engineering Business B / Product & Lifecycle:** requirements revisions, policy/ADS applicability, PRD review/freeze intentions, L2 proposal/review/freeze record, stages.
- **Engineering Business B / Task & Execution Coordination:** Task/DAG/Ready/Replan, Agent role/config matching and TaskContract, timeout/retry budgets, external operation *intents*, lifecycle projection.
- **Engineering Business B / Verification & Release:** requirement/verification closure, candidate/repair/review, accepted vs reverted, independent version gates, RQ proposal.
- **Engineering Business B / Evidence & Learning (bounded):** source-qualified GitHub report interpretation, coverage/Context references, advisory suggestion without self-promotion.
- **Selected S:** generic Context/Rule/Decision/Workflow/Evidence/Shadow where actual public package exists. B defines engineering-specific facts and obligations; do not copy a selected S interpreter.
- **Exactly one selected DH K Primary:** app State/Admission/Journal/CAS/Effect/UNKNOWN/Replay. Tally's B never writes a second authoritative state journal.
- **C GitHub / C Local Agent / C Tool/Model:** externally permissioned capabilities invoked only under K permit for business-changing effects. Read-only GitHub projection still source-qualified.
- **Host Node:** K-authoritative persistent storage and selected infrastructure per actual public Host API; SQLite GitHub cache is separate **derived read model** with proven invalidation.
- **UI/app shell:** view/form/command projection over B/K; not independently authoritative; packaging/lifecycle of shell held OPEN for public DH manifest contract.

Physical Manifest/exports/package names/deployment route are deliberately **PENDING_DH_v0.10.2** and must be filled at reviewed L2 physical design. The conceptual four B groups are **organizational** and do not require four physical independently published packages.

## 7. Critical failure contracts

**Telemetry failure:** Agent→task-meta capture/export/publish exceptions, timeout, queue-full and delayed reports must not gate Task start, effect completion, verification or later RQ. Only observed metrics/learning advisory go unavailable/partial; no false zero, no source-less Agent ranking.

**GitHub fault is different:** If required GitHub Issue/Claim/PR/Review evidence cannot be read or written due to network/403/429, native GitHub authority gates stay WAIT/UNKNOWN; already-authorized Local computation may continue only where policy permits. Do not falsely claim the whole Task passed simply because task-meta is nonblocking.

**SQLite stale cache:** Show staleness; no irreversible authorization based only on a warm cache. Revalidate expected GitHub SHA/revision and permission; write ACK is not terminal without qualified readback/reconciliation.

**Local-only configuration:** No runx module must be loaded or remote endpoint configured to execute one real Task. SSH/Docker/runx adapters are optional BUILD-selected or capability-selected as qualified.

**Agent/Review:** Agent output is an unverified claim; review/validation must bind exact candidate SHA, role provenance and independent checks. Human authority only when required by ADS/Task contract, not by GitHub account == Agent identity.

**Process crash / duplicate:** K-selected physical State/Effect/Journal/UNKNOWN and actual external C reconciliation control idempotency. No Tally-specific second Effect engine. Duplicate start with incompatible intent is typed CONFLICT.

**Package selection/release:** New versions require qualified BUILD/deploy; no runtime hot swap of selected B/S/C or prior in-flight occurrence.

### Source-qualified version-branch release and merge

Version Release Qualification binds exact candidate commit/tree, selected BUILD and required Full Regression/Hidden/CJ/packaging evidence. Finish the version branch before merging main. Immediately before merge, verify both candidate and main target expected HEAD/currentness; detect base drift and recalculate eligibility as required by the adopted ADS policy. After merge, verify actual merged commit/tree against qualified scope and evidence (including any new merge changes); if not equivalently qualified, **re-run required validation/re-issue RQ** before declaring baseline or Tag. Tag must point to the qualified final exact commit/tree, not merely the prior PR HEAD or local optimistic cache. Record genuine GitHub readbacks of merge and tag. `PR_PASS != RELEASE_PASS`, including if all leaf CI checks passed.

### Observational report correction invalidates derived advice

When a GitHub-published ADS Task Report is corrected, revoked, superseded or loses qualification, invalidate any derived Context Pack, estimated cost comparison, Agent ranking or Decision recommendation that depended on its previous source revision. Show `STALE/REQUALIFY/INSUFFICIENT_EVIDENCE` and re-derive only from current qualified facts. An advisory observation never retroactively changes a separately qualified Task/Host/CI terminal fact or silently mutates a selected Domain Package; auto Package promotion remains deferred.

## 8. Critical Journeys and acceptance set

| Journey | Positive requirement | Essential negative case | Stage |
|---|---|---|---|
| CJ01 Product lifecycle | UI requirement→PRD→review→Freeze from exact source | Missing independent review cannot Freeze | Real runtime after DH; test spec now |
| CJ02 L2/DAG/L3 | L2 Review/Freeze; DAG+Issue source readback and L3 pre-admission source/contract completeness | Cycle, stale revision, missing L3 Tests/Contract/Failure Handling blocks dispatch | Real after DH |
| CJ03 Local Task | Installed/authorized Codex CLI **without GitHub write credentials**, no runx, isolated clean workspace; selected B/S/K/C start→terminal→candidate freeze→K-admitted GitHub publish/independent verify | Shell credential push/merge bypass, dirty checkout, shared writer, fake provider, orphan/cancel, lost ACK/self-PASS blocked | Real after DH |
| CJ04 Telemetry loss | Same correct Task outcome with collector down **and Agent in-process hook hung/exit-nonzero** | Hook must timeout/fail open without blocking worker, altering Agent exit code or withholding verifier evidence; telemetry UNKNOWN | Real after DH |
| CJ05 GitHub/SQLite | Correct Issue/PR/report projection and readback | Changed source, 403/429, stale cache, edited report | Real after DH |
| CJ06 Repair/review | Fail→bounded repair→independent exact-SHA review and checks | Stale evidence/self-review cannot complete | Real after DH |
| CJ07 Version Release | Local Codex Task plus **separate OS-principal verifier with ACL-denied Builder read**, then version branch Full/Hidden/CJ→exact-tree RQ→main merge source verification→Tag | Missing OS verifier/Builder-readable Hidden path, main drift, unqualified merge tree ⇒ no Release PASS | Real after DH |
| CJ08 Experience | ADS GitHub Task report→source-qualified advisory feedback | Unsupported dialect/partial coverage cannot promote | Read-only fixtures can be specified now |

**Specifications are authored before DH; every real test execution is NOT_RUN in this document.** Mock/prototype tests may validate fixture interpretation but do not close a real K/Host/Package gate.

## 9. Dependencies, gates and version sequencing

### Phase 0: now — documents/evidence only

- Source-parity R0 with 47/47 feature domains; prioritized missing call-site/test mapping, not full low-ROI legacy revalidation.
- Product PRD R0 (this doc), UI journey specification, user decision locks, and independently reviewed Product Freeze candidate.
- Conceptual L2 boundaries, candidate Domain Packages, source-truth matrix and detailed test-vector specifications. **Physical ABI kept OPEN**, independent L2 Freeze cannot claim it verified.
- All conversational deliverables persisted to GitHub with auditable exact refs/ledger.

### Phase 1: Domain Package physical readiness before Tally production source

Require reviewed, **actual public** contracts and manifest, independent packed K/S/Host/BUILD runtime, external selected Package producer, clean Node24 npm install, real K-admitted Local Effect/replay/UNKNOWN, source pin and review. DH [#1319](https://github.com/kaicreator-mm/domain-harness/issues/1319), [#1323](https://github.com/kaicreator-mm/domain-harness/issues/1323), [#1324](https://github.com/kaicreator-mm/domain-harness/issues/1324), [#1325](https://github.com/kaicreator-mm/domain-harness/issues/1325), [#1330](https://github.com/kaicreator-mm/domain-harness/issues/1330), [#1331](https://github.com/kaicreator-mm/domain-harness/issues/1331). **Do not substitute draft packages/private deep imports as release evidence.**

Then finish independently reviewed physical L2, produce executable Task DAG, implement Tally B/C Domain Packages and UI shell with one concern/PR, test Local Task, then version RQ.

### Phase 2: release qualification

Actual replay/Claim permissions, GitHub/Local integration, task-meta down negative, missing telemetry, full regression/Hidden/CJ, packaging and release qualification, then qualified tag/baseline according to pinned ADS and version policy. **PR PASS ≠ Release PASS.**

## 10. Product decisions and external qualification gates (not self-declared P1 defects)

- **DECIDED / Product P0:** Minimal ADS UI is a stage navigator + structured forms + action/review panel; it is a B/K projection, not BPM or parallel Task State. Real UI construction, usability and end-to-end evidence remain NOT_RUN until physical Package support is qualified.
- **DECIDED / Product P0 authority:** GitHub source-qualified applicable review/Freeze artifact is the sole external lifecycle Freeze fact; K State contains user drafts and source projections only. Absence of actual GitHub Freeze evidence cannot be replaced by local B/K status.
- **HOLD / external standards pin:** v2's exact *adopted* compatible ADS revision/report dialect must be qualified at the lifecycle stage before **Product Freeze publication** or Task/Release effects, rather than silently inheriting Tally v1.5.0's ADS 1.2.1 or following floating ADS main. The independent Product reviewer may assess semantic Product readiness conditionally, **but no PRODUCT_FROZEN flag may be published while STANDARD_PIN_PENDING**. Pin selection/adoption owner: Tally Product controller with ADS upstream version review, not independent Reviewer by assumption.
- **DECIDED / Product P0 telemetry:** Hook hang and non-zero exit are fail-open, with bounded timeout and no awaited publisher on Task/Verifier/Release path; unqualified hook disabled and coverage UNKNOWN. Future physical L2 and K/C implementation must prove this; absence of real tests now is a test gate, **not an undecided requirement**.
- **DECIDED / Product P0 Local:** Codex CLI is first real provider and isolated worktrees, hidden verifier grant, N=1 executing Task Attempt; Fake only for tests.
- **DEFER / L2 physical:** Final npm Business/Capability package granularity, public Manifest/ABI and selected Host transport stay OPEN_PENDING_DH_v0.10.2; conceptual B groups are not four promised npm packages.
- **DEFER / low ROI:** Legacy F33/F36 extended retrieval, full F37 harvest and other older Go subsystems beyond explicitly scoped F02/F03/F11/F12 are P1/POST_MVP unless later independently justified by critical Product evidence.
- **Formal Product Reviewer still required:** This list states design decisions and clearly identified external gates only; a draft Product author does not independently close Product R1 P0/P1/P2 or assert Product Freeze / implementation authorization.

**Product R0 conclusion:** Product scope and boundaries can be independently reviewed **now**, but `FREEZE_READY=NO` pending review and approved ADS pin/gate disposition. Implementation completed: **0**; physical Domain Package E2E completed: **0**.
