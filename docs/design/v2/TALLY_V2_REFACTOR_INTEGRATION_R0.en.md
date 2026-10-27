# Tally v2 — Agent-Native Refactor Integration / Product & L2 Delta R0 (Agent Edition)

**Status: PRODUCT/L2 INTAKE AND IMPLEMENTATION HANDOFF ONLY.** No Product Freeze, L2 Freeze, native Task DAG mutation, accepted Claim, implementation rights, Review PASS, Release or merge is implied. **2026-10-28 user requirement:** Tally governs vibe coding with software engineering; **diverse LLM coding Agents execute normal development work** and humans intervene only when necessary. This file reconciles the latest existing refactor with that outcome; Chinese primary counterpart `TALLY_V2_REFACTOR_INTEGRATION_R0.zh-CN.md`.

## 1. Target outcome and non-negotiable ownership

A user starts/imports a repository, describes software intent, supplies limited decision/permission/budget boundaries, then *inside Tally* qualified LLM Agents propose Product/L2/parallel Task DAG, lawfully claim and build, independently review, repair, run actual CI/Host/installed user validation, prepare authorized release and plan the next version. User clicks **only** genuine material decisions, credentials/physical Host steps or nondelegable approval—not every ADS stage or every Agent→Agent handoff.

- Native GitHub Task/Dependencies/Claim/PR/Review/CI/Release remain source of truth.
- Project-adopted exact ADS controls normative reviews/roles/changes/Release.
- One selected DomainHarness K Primary controls app State/Admission/CAS/Journal/Effect/replay/UNKNOWN; selected S Rule/Context/Workflow and C GitHub/Agent/Host/Model adapters are reused. Tally Engineering B owns project/product/planning/Agent coordination/assurance/improvement.
- Private task-meta traces are **not** a Tally connector; only qualified ADS-published GitHub report/coverage is read, and observer failure must not block engineering.
- Logical B contexts (Project/Product; Architecture/DAG; Agent Registry/Dispatch; Version Journey; Assurance/Release; Experience) **do not require six physical Business npm Packages**. Verify actual public DomainHarness BUILD/ABI before splitting.
- Do not reimplement GitHub, Model/provider SDK, K, private task-meta ingestion, generic BPM or another scheduler/approval engine.

## 2. Current real v2 baseline: reuse what is already built

| Native source | Latest verified candidate or accepted stage | Genuine unresolved boundary |
|---|---|---|
| [#86 v2 controller](https://github.com/kaicreator-mm/tally/issues/86) | Earlier readback of version/v2 `de5b937b...`, 14 independently reviewed stage PRs merged into VERSION | Version does not equal main/released Tally v2 |
| [#120 real GitHub C](https://github.com/kaicreator-mm/tally/pull/120) and [#127 native Issue dependency](https://github.com/kaicreator-mm/tally/pull/127) | MERGED in version/v2, real source readback and no-Claim denial | Not a real accepted Task Claim/whole Task |
| [#129 source event C](https://github.com/kaicreator-mm/tally/pull/129) | Open Draft `802719824bb3a4995fbec4d72da8d0818b506f70` | Review/accepted CAS unproven |
| [#131 Engineering B Claim diagnosis](https://github.com/kaicreator-mm/tally/pull/131) | Open Draft `be275dd43a97d5e3d37a3edb02de5c0c7ca9af9b` | Advisory projection, not K admission |
| [#133 Host Claim CAS](https://github.com/kaicreator-mm/tally/pull/133) | Open Draft `7cc08a62fc2c06b1b46b9a4be594f8874da6fecd`, public real SQLite/Host 6/6 Node24 | GitHub publisher/readback positive was mocked; real native Claim still unproven |
| [#77 Product candidate](https://github.com/kaicreator-mm/tally/pull/77) | Open Draft `b80594e2391617d9527f669e6b3d081ab98215fb`, 17 Product P0, 67 V vectors | Independent Product parity P1 BLOCK, no Product Freeze |
| [#82 L2 semantic](https://github.com/kaicreator-mm/tally/pull/82) | Open Draft `0a58f2dad881f5634575e2d0cf3107470ec97de2`; source R5 semantic PASS | L2 Freeze HOLD because Product and ADS adoption incomplete |
| [#85 ADS adoption](https://github.com/kaicreator-mm/tally/pull/85) | Open Draft exact v4.18 qualified source | No effective Tally project pin yet |
| [#135 upstream sync](https://github.com/kaicreator-mm/tally/pull/135) | Open Draft candidate Source compatibility for DH 0.10.3 | No released 0.10.3 tag at this observation; keep immutable qualified 0.10.2 until upgrade review |

This is a **snapshot from actual PR/Issue source**; reread current GitHub source before any effect. Do not overwrite in-flight writers or replay previous Reviews on new SHAs. Existing tests are reusable only to the actual applicable subject/profile and scope.

## 3. Required Product decision matrix before claiming complete Agent-native v2

| ID | Existing candidate | Revised intended requirement / owner decision |
|---|---|---|
| **D01 Agent-first P0** | PR #77 §1 describes a user “with Agents”, UI/operator semantics may be implemented as a click-through stage engine | User goal and bounded delegation should allow qualified LLM Planner/Architect/Builder/Reviewer/Verifier/Integrator to do ordinary legitimate work continuously. No mandatory user “continue”; not an automatic rights bypass. Add positive and false-HUMAN_REQUIRED negatives to #92/#145. |
| **D02 Safe parallel execution P0** | Product PRD §4, V63 and L2 §6 **explicitly limit P0 executing Task Attempts to 1**; concurrency is P1 | This materially conflicts with the established multi-Agent safe-parallel product direction. Recommend make **at least two disjoint genuinely concurrent Builder Agent attempts** a final v2 P0 acceptance, preserve one-Task strict path only as initial implementation slice. Product Owner must decide and amend frozen-candidate requirements; do NOT silently edit old V63 or claim its tests PASS. If kept at 1, describe v2 as a limited MVP, not fulfilled full Agent-native parallel development. |
| **D03 Multiple LLM Agent providers** | P0-06 initial real Codex CLI, Claude optional | Keep first real Codex provider for cheap vertical E2E; provider-neutral Agent registry/typed C adapter is P0 and at least one additional real selectable Agent CLI/role compatibility should be reviewed for final user outcome (Claude Code/ZCode where actual capability and tests exist). Product Owner decides whether a second physical Provider must release-block; fake/echo never qualifies. |
| **D04 Autonomous lawful Version Journey** | existing #92 J01–J08 UI and #145 nextActions Product intake | B must schedule lawful Agent→Agent work/review/repair/CI with true effective GitHub/K rights, pause only on source/Claim/budget/physical Host/Owner unknown and resume on exact readback; host/agent execution is real infrastructure, not a perpetual chat process. |
| **D05 Requirement-to-actual-behavior coverage** | #77 67 V tests mostly single Task, safety and UI; #92/#146 are later Product intake | Crosswalk existing vectors; minimally add actual selected installed user capability, inherited safety invariants, authorized RQ→Tag, and next-version real app journey. No duplicate test suite for every leaf. |
| **D06 Agent review independence** | existing logical Reviewer roles are present | Clarify genuinely independent **LLM Reviewer** context/role/subject and source provenance; a Builder changing role label is not independent. An independent Gate is not a mandatory HUMAN Gate. |
| **D07 Risk/Host proportionality** | P0 protected Builder credentials + separate Hidden verifier OS principal | Keep actual applicable P0 source/effect/Hidden promises unless lawfully amended; do not impose full hidden/physical Host per trivial leaf. Reuse qualified CI/Runner where possible. |
| **D08 Product parity P1** | #77 R3 retained 31 P0 of 47 legacy F domains and source mapping incomplete | Product Owner explicitly RETAIN/ADAPT/RETIRE/UNKNOWN with evidence. Resolve existing P1 through bounded parity proof or lawful risk/scoping amendment, not fabricated PASS, and do not rebuild obsolete Go subsystems. |
| **D09 Evolution scope** | #57/#58/#60/#139–#144 Product intake only | Basic casebook evidence/context + owner feedback as new-version workflow; advanced cross-project automated S Package promotion, broad A/B/Shadow and generic learning infrastructure remain conditional so core true software release is not blocked by speculative learning. |

**Critical conflict to force a real Product decision:** **V63 max-one-active Task** vs genuine maximum-safe Agent concurrency. Engineering write-set isolation, native Issue dependencies and single-writer CAS cannot be traded away for concurrency. Do not launch two physical concurrent source tasks until Product/L2/Task DAG authorizations and actual Host/Claims allow them.

## 4. Proposed minimum implementation topology (planning only)

```text
Product PR #77 parity P1 remediation + Agent-native D01..D09 amendment
           ├── ADS project adoption PR #85
           └── semantic L2 PR #82 bounded source/ABI reconciliation
                         → qualified source/authority join
     ┌───────────────────────────┼───────────────────────────┐
     ▼                           ▼                           ▼
B Project/Product + Planning   Agent Registry/C adapters    UX app shell
(#91, #141, casebook #143)     (#64, Host, real CLI)         (#92/#145 read models)
     └───────────────────────────┤───────────────────────────┘
                                 ▼
   Existing native Claim pipeline #125 (#127 merged, #129/#131/#133 review)
                                 ▼
    Real selected K → C eligible Agents + source-scoped WorkUnits
                                 ▼
    Two source-disjoint Builders (only after authorized parallel Product/DAG)
                                 ▼
    Independent Agent Reviewer → bounded Builder repair → exact re-review
                                 ▼
    CI/Host + selected external installed user behavior (#146)
                                 ▼
    Authorized version RQ/merge/tag → next version; experience feedback
```

The diagram is **not** itself a native Github dependency graph, task authorization or all-serial construction plan. Read-only/spec/UI work and independent implementation concerns may develop in parallel **where existing ADS source/Claim rules admit them**. Only genuine shared writer/contract and integration join force waiting. Do not steal source tasks already owned by PR #129/#131/#133/#135; use one shared-root integration owner.

## 5. User-facing critical journeys and negatives (candidate Product acceptance)

- **CJ-01** Real import/new project→natural language discovery, Product/L2/DAG candidate, source/status honest; no raw JSON; only material user decisions.
- **CJ-02** One physical eligible Codex CLI Task + real GitHub native accepted Claim → K-admitted selected C/Host → PR/independent Agent Reviewer + real CI; task-meta failure never blocks Task; unknown external Effect remains UNKNOWN.
- **CJ-03** **Recommended final Agent-first P0**: at least two genuinely concurrent disjoint legitimate Agent Builder attempts; one real independent Agent Reviewer finds a P1; scoped Builder repair and fresh review; actual selected installed user capability and Host verification. Must await approved Product concurrency change.
- **CJ-04** Agent autonomous ordinary next steps, user-only exception Inbox: scope/risk ambiguity, unavailable Host/credentials, exceeded real delegated rights/budget, applicable human Release duty; false human-required is a negative; no fake independence.
- **CJ-05** Qualified real RQ, authorized merge/tag readback, next version, actual retained/revised user capability, source-qualified practice feedback.

Other negatives: stale SHA/Claim; same protected write set claimed by competing agents; unqualified Agent Provider/Host; mock CAS publisher; Builder shell bypassing GitHub C rights; stale Reviewer source; borrowed GitHub login ≠ Agent identity; false Host PASS; lost ACK replay; automatic Product/Release self-approval; old Release rewritten; LLM model self-rating used as learning evidence. Metrics require measured task-meta ADS-published GitHub coverage; Issue lifetime is NOT active Agent time.

## 6. What is actionable immediately

1. Crosswalk the **present** #77 Product 17 P0/V01–V67 against D01–D09; formal Product Owner must revise/decide source pins and the unresolved legacy P1, then commission exact-HEAD independent Product review and lawful Freeze. These research documents cannot bypass it.
2. Rebase/synchronize #82 source contract **only after** appropriate Product decisions, select actual public DomainHarness ABI and genuine Agent roles/Tool bound effects, validate with independent L2 review.
3. Let #86 controller reconcile preexisting code/Task PRs and currently admitted native GitHub dependencies; only newly independent approved business concerns get new Task Issues and assigned source write sets. Keep already accepted source and the one physical K.
4. Complete a real Task pipeline, then safe independent parallel Agent integration, then real user-facing Journey/Host/RQ/next version. Track Product acceptance proof separately from code progress.
5. Qualification must be on actual pinned subjects and physical evidence, not mock-only/PR Draft/Design status.

**Current disposition:** `DESIGN_BASE=READY_FOR_FORMAL_PRODUCT_INTAKE`; `PRODUCT_FREEZE_READY=NO`; `L2_FREEZE_READY=NO`; `NEW_SCOPE_NATIVE_DAG_NOT_ADMITTED`; `FULL_AGENT_NATIVE_V2_NOT_RUN`; `MAIN_OR_RELEASE_AUTHORIZATION=NO`. This proposal is compatible with an **incremental continuation of existing Tally v2**; a wholesale restart would throw away valid source/Host/GitHub work and conflict with the observed goal of high-ROI engineering.
