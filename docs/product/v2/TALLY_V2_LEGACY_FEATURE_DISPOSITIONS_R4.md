# Tally v2 — 47 legacy functional domains: user-confirmed migration disposition (R4 scope)

> **Source qualification:** The project user expressly confirmed PR #152 on 2026-10-28 after review of the proposed 47-row / 19 RETAIN / 23 ADAPT / 5 RETIRE decision. This is a **user-confirmed Product migration scope** (Controller-transcribed on [PR #152](https://github.com/kaicreator-mm/tally/pull/152#issuecomment-6462259490), [Product #76](https://github.com/kaicreator-mm/tally/issues/76#issuecomment-6462261559) and [Product Owner #63](https://github.com/kaicreator-mm/tally/issues/63#issuecomment-6462263058)); independent R7 must still verify authorized owner attribution/risk and find P0/P1/P2. The confirmation is **NOT runtime compatibility PASS**, not historical code execution, not a replacement for real ALM acceptance, and NOT a publication of Product Freeze. Original 47-domain source matrix: [fixed v1.5.5 source audit](../../research/v2/TALLY_V2_FUNCTIONAL_PARITY_R0.md). Reviewed Product R4 [PR #152](https://github.com/kaicreator-mm/tally/pull/152) carried a parity P1 (unclosed retain/retire/acceptance dispositions). This bounded table addresses the **scope decision** without adding 47 separate engineering Tasks or reenacting an obsolete Go implementation.

## User-confirmed scope principles for independent Product Review

- **RETAIN** means preserve the externally testable software-engineering invariant; implementation is still a new selected DomainHarness Domain App and **physical v2 proof is NOT_RUN**.
- **ADAPT** means preserve the stated minimal user-observable software-engineering outcome with a new domain-package/ADS-native method; **do not port the old Go subsystem**. Advanced parts explicitly marked P1 are not v2 Release blockers.
- **RETIRE** means the *legacy specialized engine/UI/collector* is retired, **not** an independently required safety invariant: replacement users' visible behavior is named below where necessary. P0 selected DomainHarness K/admitted effects, ADS/GitHub authority, real Review and Release gates cannot be retired.
- **UNKNOWN** remains the evidence state when source production reachability, actual old test execution, selected SDK/Host ABI or current v2 behavior has not been physically verified. A deliberate Product scope choice is **not proof** of runtime reachability.
- Verification phase: **M1** Product/L2/authority review; **M2** native GitHub/Task/Claim/selected B→K→C/SQLite+one real Agent; **M3** separate Reviewer, Defect/Repair, QA/Host and two disjoint agent attempts; **M4** real installed ALM UI, RQ/Release, next-Task learning. This is a *coverage timetable*, not a fully serial Task DAG; unrelated concerns can run in parallel with legal worksets.

## Per-feature migration decision ledger

| Legacy domain | Product disposition | Minimal v2 behavior / explicit retirement boundary | V2 acceptance vector | Severity | Owner / existing Issue | Verification |
| --- | --- | --- | --- | --- | --- | --- |
| F01 | **RETAIN** | Requirements & acceptance baseline, typed source revision | V01,V68 | HIGH | Product B / #76 | M1 source review; M4 physical UX |
| F02 | **ADAPT** | Bounded ambiguity → question/answer → attributed requirement change | V31,V46,V68 | MED | Product B / #76 | M1 exact source; M4 UX |
| F03 | **ADAPT** | Frozen intent and risk changes invalidate stale review | V02,V32,V47 | HIGH | Product B / #76 | M1 source; M4 independent review |
| F04 | **RETAIN** | PlanningRevision and Project Plan remain revision-linked | V01,V68 | MED | Planning B / #91 | M1/M4 |
| F05 | **RETAIN** | Architecture constraints/L2 revision and qualified freeze | V01,V02,V68 | HIGH | Architecture B / #91 | M1/M4 |
| F06 | **RETAIN** | Task DAG proposal/activation distinct from real native authority | V03,V44,V69 | HIGH | DAG B/GitHub C / #125 | M2 source/K; M4 UI |
| F07 | **RETAIN** | Acyclic real dependencies and safe parallel readiness | V03,V04,V63 | HIGH | DAG B / #91 | M2 source; M3 two Agents |
| F08 | **RETAIN** | Completion contract and missing-gate explanation | V04,V06,V44,V45 | HIGH | Task B / #91 | M2/M3 |
| F09 | **RETAIN** | Human authority decision vs Agent automatic legal next action | V05,V50,V65,V82 | HIGH | Governance B/K / #145 | M2/M3/M4 |
| F10 | **RETAIN** | Source/lineage-derived independence; no role laundering | V06,V36,V74 | HIGH | Evidence B / #146 | M3 independent reviewer |
| F11 | **ADAPT** | Risk-bounded assurance/independent verifier selection | V33,V48,V76 | HIGH | Assurance B / #146 | M3/M4 |
| F12 | **ADAPT** | Current version ADS L3 contract validation, no old compiler | V34,V44,V49 | HIGH | ADS GitHub C / #125 | M2 exact pin |
| F13 | **ADAPT** | Project Task readiness/state projection, one selected K | V07,V26,V64,V70 | HIGH | Engineering B/K / #91 | M2 source/K; M4 workspace |
| F14 | **ADAPT** | Typed app commands/UI to authorized B→K→C operations | V22,V26,V65 | HIGH | App shell / #92 | M2/M4 |
| F15 | **ADAPT** | One-Task real slice → two legal disjoint concurrent Builders final v2 | V09,V10,V56,V63,V72,V73 | HIGH | Agent/DAG B+Host / #64/#125 | M2 first task; M3 real pair |
| F16 | **ADAPT** | External Agent capability, request/receipt currentness | V07,V08,V52,V70 | HIGH | Agent B+C / #64 | M2 real provider |
| F17 | **ADAPT** | Installed real Codex CLI as one governed adapter; fake test only | V52,V53,V66 | HIGH | Agent C/Host / #64 | M2 physical Local |
| F18 | **RETIRE** | Old Tally-owned direct model-runtime as mandatory execution path; use optional DH selected LLM advice and external ADS Agents | V27,V53,V81 | MED | Provider C (optional) / #150 | P1 model expansion; no P0 engine |
| F19 | **ADAPT** | Local Host/workspace execution mandatory first; SSH/Docker/runx optional | V07,V52,V54,V62 | HIGH | Agent Host/C / #64 | M2 physical Local |
| F20 | **RETIRE** | Generic legacy environment UP/Deploy/Rollback manager as universal P0 service; track release/host evidence through existing services | V23,V77,V78 | MED | Release B / #92 | P1 optional environment service; M4 gate |
| F21 | **RETAIN** | Per-Attempt disposable worktree and immutable candidate tree | V12,V54,V55,V56 | HIGH | Host/C Git / #64 | M2/M3 physical |
| F22 | **ADAPT** | Immutable source/Attempt/ChangeSet facts via K journal + GitHub receipts | V09,V10,V15,V70 | HIGH | K/Host/Evidence B / #125 | M2 restart/replay |
| F23 | **RETAIN** | SHA+protocol-bound verification & required independent review | V12,V14,V45,V76 | HIGH | Assurance B / #146 | M3/M4 real CI |
| F24 | **RETAIN** | Held-out isolation with actual separate-principal deny witness | V13,V57,V58,V67 | HIGH | Verifier Host/C / #146 | M3/M4 physical |
| F25 | **ADAPT** | Review→Repair→Recheck bounded loop coordinated via ADS, not a second closedloop engine | V14,V45,V74,V75 | HIGH | Journey B / #145 | M3 real Review |
| F26 | **RETAIN** | All changed/protected files accounted with reviewer independence | V35,V36,V74,V75 | HIGH | Review B / #146 | M3 exact SHA |
| F27 | **RETAIN** | External Agent role/identity/Review independence and repair | V36,V72,V74,V75 | HIGH | Agent B / #64 | M3 Agent/Reviewer |
| F28 | **ADAPT** | Native GitHub PR/CI/Issue & permission-currentness C adapter | V15,V16,V38,V41 | HIGH | GitHub C / #125 | M2/M4 native readback |
| F29 | **RETAIN** | Local Git change safety/conflict prevention and governed publisher | V54,V55,V56,V66 | HIGH | Host/C Git / #64 | M2/M3 source |
| F30 | **RETAIN** | Replan preserves accepted work, rebases only on current source | V17,V42,V75 | HIGH | Planning B / #91 | M2/M3 |
| F31 | **ADAPT** | P0 bounded real Task Learning → next Task context; old Go Rule engine not copied | V79,V80 | MED | Learning B / #57 | M4 two real Tasks; rule promotion P1 |
| F32 | **ADAPT** | Qualified rule/skill/procedure *candidate* governance only; no automatic package activation | V27,V43,V79 | HIGH | Governance B/S / #60 | M4 candidate; package activation P1 |
| F33 | **ADAPT** | P0 context selection from verifiable cases for later lawful Task | V79,V80,V81 | MED | Context B/S / #58 | M4 later Task |
| F34 | **RETAIN** | Reproduction and verified repair gates, avoid single flaky result | V14,V45,V74,V75 | HIGH | Failure B / #140 | M3 real failure |
| F35 | **ADAPT** | P0 secure error redaction/typed failure; advanced log fingerprinting P1 via utilities | V28,V34,V74 | MED | Evidence B/C / #140 | M3 core; P1 analytics |
| F36 | **ADAPT** | Learning context stale/revoked/contradictory check; old knowledge DB not authoritative | V43,V79,V80 | MED | Learning B/S / #57/#58 | M4 usage + invalidation |
| F37 | **RETIRE** | Broad legacy harvesting/dependency ingest workflow; native GitHub C chosen, optional ingestion later | V15,V16,V69 | MED | GitHub C / #125 | M2 native; P1 broad ingest |
| F38 | **ADAPT** | P0 source-backed progress/outcome, advanced cost/rework analytics optional | V19,V21,V70,V81 | MED | Evidence B / #139 | M2/M4 source |
| F39 | **RETIRE** | Old embedded Token/time collector; task-meta owns measurement, Tally consumes ADS GitHub summaries with fail-open | V18,V59,V60,V81 | HIGH | task-meta external / #57 | M2 negative; M4 metrics |
| F40 | **ADAPT** | One selected DH K/Host state store plus reconstructable SQLite GitHub projection; Postgres optional | V10,V11,V21 | HIGH | K/Host/cache / #125 | M2 physical SQLite |
| F41 | **ADAPT** | GitHub durable Project/evidence lineage for session restart; no alternate Git SoR | V25,V69,V70,V78 | MED | Source B/C / #139 | M2/M4 |
| F42 | **RETAIN** | Project, requirement, docs and decision UI in one ALM workspace | V01,V22,V68,V69 | MED | UX / #92 | M4 installed UI |
| F43 | **RETIRE** | Legacy v14-specific launch/attention UI implementation; replace with source-grounded v2 ALM Journeys | V22,V64,V82 | MED | UX / #92 | M4 UI new flow |
| F44 | **ADAPT** | v15 task lifecycle UI as v2 read-only/source-authorized task board | V22,V64,V70,V82 | MED | UX / #92 | M4 installed UI |
| F45 | **RETAIN** | Operator session, CSRF, provider credential/source scope, secret redaction | V28,V37,V38,V39,V40 | HIGH | App transport security / #92 | M3/M4 |
| F46 | **ADAPT** | Expose simple local app/install/Agent setup; do not require all legacy worker/CLI entries | V22,V29,V52 | MED | App Host/CLI / #92 | M4 clean consumer |
| F47 | **RETAIN** | Exact SHA candidate Full/Hidden/CJ, independent RQ, authorized merge/tag and real behavior | V23,V24,V42,V67,V76,V77 | HIGH | Release B/Host / #146 | M4 physical Release |

**Decision counts:** RETAIN=19; ADAPT=23; RETIRE=5; 47/47 exact source-domain IDs. All referenced vector requirements are *specifications* until independently executed at the proper selected DomainHarness/K/Host/GitHub/Agent gate.

## Explicit scope-versus-evidence and actual P0 cut

The 47-row previous audit already links historical source paths and many named test declarations, including the high-risk evidence/verification/Hidden / Review / Git-write cases. **Those source/test names are not historical Go PASS**. Independent reviewer should accept/reject the Product scope mapping on its merits; an unverified production caller does not warrant months of re-building obsolete Go innards. Conversely retained **HIGH** P0 safety/authority/release behavior cannot be marked qualified merely by an owner row: it must be demonstrated through current v2 physical selected K/C/Host/ADS and true source evidence at M2–M4.

**Minimum release Product value:** user in Tally creates/imports software project, adds requirements with traceable acceptance, reviews legitimate Product/L2 and Task DAG, coordinates qualified external Agents through accepted native ADS/GitHub Claims and authoritative source readback, tracks independent Review/Defect/Repair, actual Test Case/CI/installed acceptance, Release/RQ and post-release next Version, and gives verified historical Task context to a **different** later lawful Agent Task. No in-process Coding Agent, alternate CAS, second Kernel, task-meta raw reader, enterprise BPM, or universal legacy feature engine.

**Gate now:** `R4_LEGACY_SCOPE_DISPOSITION=USER_CONFIRMED_47_OF_47_2026_10_28`; `OWNER_ATTRIBUTION=INDEPENDENT_VERIFICATION_PENDING`; `LEGACY_RUNTIME_REACHABILITY=UNVERIFIED`; `V2_PHYSICAL_E2E=NOT_RUN`; `PRODUCT_PARITY_P1_CLOSURE=INDEPENDENT_R7_PENDING`; `PRODUCT_FREEZE=NO`. Do not silently close the P1 until an **independent exact-HEAD Product reviewer** validates authorized owner decision, severity/risk/retire rationale and absence of regressions. Do not promote optional P1 analytics, environment management, full original v14 UI or automatic rule/skill promotion to the v2 release critical path.
