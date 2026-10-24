# Tally v2 Product PRD — native Domain Package Engineering System Center (R0 candidate)

**Date:** 2026-10-24  
**Review state:** PRODUCT_DRAFT_R0 / INDEPENDENT_PRODUCT_REVIEW_NOT_RUN / FREEZE_READY=NO  
**Owning Issue:** [#76](https://github.com/kaicreator-mm/tally/issues/76), parent [#63](https://github.com/kaicreator-mm/tally/issues/63), [L2 #65](https://github.com/kaicreator-mm/tally/issues/65), [Domain Package-first correction #75](https://github.com/kaicreator-mm/tally/issues/75#issuecomment-6380546957)  
**Legacy Product evidence:** [F01–F47 matrix](./TALLY_V2_FUNCTIONAL_PARITY_R0.md), pinned v1.5.5 source b5e7b6e38ea38837e3bbcfc46c36586180705de0.  
**Standards adoption note:** Current Tally repo .dev-standard/VERSION is v1.5.0-specific ADS 1.2.1, **not a v2 authorization**. ADS repo main VERSION was 4.16.0 on inspection; v4.17 in preparation. Select and lawfully adopt **exact immutable qualified compatible ADS revision** for v2 before executable Task/ADS report publication. ADS v4.15 source-only reports may be test fixtures only until appropriate adoption. Do not use floating main as authority.  
**Platform dependency:** DomainHarness v0.10.2, actual independently installable Domain Packages/public ABI/BUILD/Host; physical qualification pending [DH T09 #1330](https://github.com/kaicreator-mm/domain-harness/issues/1330) and [R01 #1331](https://github.com/kaicreator-mm/domain-harness/issues/1331).

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
| P0-03 | Product Review/Freeze gate | Independent review of exact proposal and status with P0/P1/P2; no freeze from proposal alone; missing evidence shows BLOCKED/NOT_RUN |
| P0-04 | Architecture L2 UI and gate | Draft/revise L2, authority/dependency/negative scenarios, independent Review/Freeze, no direct Task admission while Product/L2 not approved |
| P0-05 | Task DAG and issue publication | Decompose tasks, show dependencies/ready/blocked, cycle rejection, bounded parallel candidates, GitHub Issue/Task contract/write and readback |
| P0-06 | Local-only Agent dispatch | Select declared capable Agent on Local workspace with explicit Claim/currentness, invoke genuine K-admitted C effect, preserve accepted ≠ terminal and UNKNOWN |
| P0-07 | Repair and independent verification | Explain evidence-backed failure, construct bounded retry/replan, protect accepted work, route reviewer/verifier independently and require real checks |
| P0-08 | GitHub authority and ADS reports | Consume Issue/PR/Review/CI/host facts and source-qualified ADS report, handle schema version, corrections, retries, stale source and true readback |
| P0-09 | SQLite derived-cache discipline | Faster GitHub projections and search; timestamp/source coverage/currentness and invalidation; no cache-based approval for irreversible native fact change |
| P0-10 | task-meta degraded execution | Collector fail/timeout/report gap does not block Task; missing token/cost/coverage=UNKNOWN; quality/effect gates unaffected by telemetry ACK |
| P0-11 | Version closure & release UI | Show Task vs version gates separately; full regression/Hidden/Critical Journeys, real RQ, PR merge and Tag under exact-source eligibility |
| P0-12 | Agent/role and authority controls | Agent registry/capability/currentness; logical role distinction, permission-bounded delegation, required user decision only where applicable |
| P0-13 | Safe default UI | All steps reachable without editing raw JSON/SQLite or shell scripts; actionable blocked reasons with source; no decorative success on mock |
| P0-14 | Cross-session GitHub durability | Every substantive decision/report/review/dispatch has canonical GitHub Issue/PR/document/ref, source SHA and recoverable handoff, independent of chat |
| P0-15 | Failure and data security | Secrets redacted, source/permission boundaries enforced, no direct bypass from UI/Agent to protected GitHub/Host effects; unsupported/UNKNOWN fail honestly |

### Domain-specific behavior to retain

- RequirementsBaseline, ProjectPlan/PlanningRevision, ArchitectureRevision and explicit provenance (audit F01–F05).
- TaskDAG DAG-cycles, readiness/parallel bounded scheduling, CompletionContract, required human authority and negative gate reporting (F06–F09, F13–F15).
- Evidence/assurance/verification coverage, hold-out isolation, candidate and workspace safety, repair and independent review (F10–F12, F21–F27, F47).
- Actual GitHub engineering operations with readback and source pin, stale-parent replan and version release audit (F28–F30).
- Conservative strategy/Context/error diagnosis and verified lessons when source evidence exists (F31–F36).
- Consistent UI experience from older natural-language launch and v1.5 lifecycle surfaces, **not** direct reuse of obsolete v14 job workflow (F42–F44).

## 4. P1 and deferred scope (not a core gate)

**P1 candidate:** supported Agent/Model cost comparisons with source coverage; error fingerprints/context suggestions, verified experience/skill selection and feedback; cross-project Blocked/Ready/Cost view; optional advanced multi-Agent parallel dispatch. Must not delay minimum single-project release.

**DEFER:** remote Runx Gateway/SSH instance broker as a core dependency, automated Package learning→promotion, broad recurrence/recursive optimization, configurable generic BPM, heavyweight distributed scheduler, full multi-project Portfolio/enterprise governance, rewriting old Go components, universal deployment environments, maintaining pre-1.0 Go ABI compatibility.

**No-goals:** Tally owns neither a second Kernel nor a generic Agent SDK/Tool runtime, ADS normative schema author, a private task-meta raw-data adapter, or GitHub's native Task truth.

## 5. Basic complete ADS UI journey (P0)

The UI is a **one-project/version workflow**, with stage navigator, a concise Markdown/structured edit pane, Issue/PR/evidence source links, review action strip and explicit missing gate reasons. It need not be a generic BPM designer.

| UI phase | User sees and can do | Source of authority | Gate / negative action |
|---|---|---|---|
| U01 Project setup / evidence | Select repo/ADS exact pin/version; collect intent, sources, open questions | GitHub repo + B draft via K | Missing repository grants or invalid pin blocks publication, not draft editing |
| U02 PRD | Structured requirement editing, acceptance and diff vs previous revision, issue linking | B Product revision + GitHub publication | Conflicting/unsourced critical assumptions stay OPEN |
| U03 Product Review/Freeze | Assign independent Product review, findings, corrections, exact accepted SHA | ADS-qualified review/freeze evidence | Reviewer-self-approval and P0/P1 unresolved cannot PASS |
| U04 L2 | Edit authority/Package boundary/host/execution architecture, negative contracts, independent L2 review | B Architecture + selected standards; GitHub readback | Package ABI marked OPEN until real DH; cannot freeze physical claim by guess |
| U05 Task DAG | Create task nodes/dependencies, split, mark ready, publish Issues/L3 contracts | B DAG proposal + native GitHub Issue dependencies | Cycle, stale accepted parent, missing admission are blocked |
| U06 Execute | View Agent Profiles and eligible Local host; dispatch/start/cancel/trace status | K admission/Effect + C Local + actual Host receipts | No runx required; accepted ≠ done; missing Claims/permissions block |
| U07 Repair / Verify | See actual failure fingerprint, Repair option, PR/review/verifier and narrowed rerun | GitHub/Host/independent review facts | Agent says PASS ≠ verifier PASS; no speculative self-approval |
| U08 Version closure | Full integration/regression/Hidden/CJ progress, RQ decision, merge/tag/baseline | Actual GitHub Release/RQ evidence | PR PASS ≠ Release PASS; missing witness remains NOT_RUN |
| U09 Experience | task-meta reported coverage/Token/Cost/learning feedback when qualified | Source-qualified ADS GitHub reports | Missing/unsupported evidence means UNKNOWN, never Task failure |

**UI simplification acceptance:** A new user can finish the basic journey from project selection to version qualified Tag using UI forms plus the genuine required Agent/reviewer/Host execution, without hand-authoring TaskExecutionRequest JSON. Verifications are genuinely performed; no UI click claims they occurred.

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

## 8. Critical Journeys and acceptance set

| Journey | Positive requirement | Essential negative case | Stage |
|---|---|---|---|
| CJ01 Product lifecycle | UI requirement→PRD→review→Freeze from exact source | Missing independent review cannot Freeze | Real runtime after DH; test spec now |
| CJ02 L2 and DAG | L2 Review/Freeze; DAG acceptance, Issue readback | Cycle/stale DAG/revision blocks publish/admission | Real after DH |
| CJ03 Local Task | No runx, selected B/S/K/C Local start→terminal→PR/verify | Lost ACK/duplicate effect/Agent self-PASS not accepted | Real after DH |
| CJ04 Collector loss | Same correct Task result with task-meta collector killed | Queue timeout/empty ADS report yields UNKNOWN only | Real after DH |
| CJ05 GitHub/SQLite | Correct Issue/PR/report projection and readback | Changed source, 403/429, stale cache, edited report | Real after DH |
| CJ06 Repair/review | Fail→bounded repair→independent exact-SHA review and checks | Stale evidence/self-review cannot complete | Real after DH |
| CJ07 Version Release | Integrated Tasks→Full Regression→Hidden→CJ→RQ→Tag | PR-only green cannot yield Release PASS | Real after DH |
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

## 10. Product review agenda — unresolved

- **P1 pending:** Is the **minimum ADS UI** feasible without large generic Workflow UI or a parallel Task state? Reviewer must inspect realistic navigation and error recovery.
- **P1 pending:** Define lawful ADS v2 adoption pin and report dialect. Existing project-local v1.5.0 pin is not future Task authorization.
- **P1 pending:** Clarify which planning/freeze records are actual GitHub ADS artifacts vs B application drafts; require one owner and source-currentness/readback.
- **P1 pending:** Prove task-meta failure cannot accidentally block a *verification* receipt or a GitHub-owned release gate due to a shared callback.
- **P2 pending:** Final physically published B groups/package granularity depends on DH Manifest/ABI; do not freeze four package names now.
- **P2 pending:** Legacy F02/F03/F33/F36 and low-ROI F37 relevance requires source/test ROI confirmation before final feature acceptance.

**Product R0 conclusion:** Product scope and boundaries can be independently reviewed **now**, but `FREEZE_READY=NO` pending review and approved ADS pin/gate disposition. Implementation completed: **0**; physical Domain Package E2E completed: **0**.
