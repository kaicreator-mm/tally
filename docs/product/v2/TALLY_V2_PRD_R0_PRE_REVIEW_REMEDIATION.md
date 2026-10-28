# Tally v2 Product R0 — GPT-6 and Claude pre-review remediation ledger

**Status:** AUTHOR_REPAIR_CANDIDATE / CLAUDE_PRE_REVIEW_REWORK_P1_5_P2_5 / FORMAL_PRODUCT_R1_NOT_RUN / PRODUCT_FREEZE_READY=NO  
**Source review:** [PR #77 GPT-6 pre-review comment #6381193617](https://github.com/kaicreator-mm/tally/pull/77#issuecomment-6381193617) on `3935e2d6d8a499fd010fbe82a5d9485bf7234c58`.  
**Review identity:** reviewer actually identified as `GPT6_ADVERSARIAL_PRD_PRE_REVIEW` even though originally requested `CLAUDE_PRD_PRE_REVIEW`. Claude execution is **not** observed; do not translate one into the other.  
**Findings:** P0=0 / P1=5 / P2=2, verdict `PRE_REVIEW_REWORK`. **Author's patch != independent closure.**

## Exact remediation mapping

| Finding | Owner | Author change submitted for independent verification | Future falsifying evidence | Author state |
|---|---|---|---|---|
| P1-01 L3 admission absent | Product now; semantic L2 downstream | PRD P0-05 and U05-L3 specify separate L3 Tests, Contract, Implementation, Failure Handling, Reference source, GitHub readback, missing/stale blocker, and post-run Implementation Evidence; V44/V45 added | Dispatch succeeds with DAG Ready while current L3 missing, or post-run PASS without real Implementation Evidence | FIX_CANDIDATE |
| P1-02 F26 incorrectly P1 | Product | F26 disposition now P0 security minimum, rich UI remains P1; PRD P0-07/15 and guard/reviewer policy; V35/V36 | Unpermitted protected change or same-lineage role swap accepted as independent review | FIX_CANDIDATE |
| P1-03 UI operator Auth/CSRF lacks negative tests | Product with physical API L2 TBD | PRD P0-15 plus session/provider/scope/CSRF/origin result contracts; V37–V41 | Wrong repo, expired session, forged origin, revoked grant or missing readback changes real GitHub/Host state | FIX_CANDIDATE |
| P1-04 version RQ→merge→tag exact-tree gate missing | Product with L2 | PRD P0-11, U08 and version-branch post-merge tree/tag revalidation, currentness and required re-RQ; V42 | Main drift or merged tree differs from qualified candidate and UI still creates verified Tag | FIX_CANDIDATE |
| P1-05 bounded F02/F03/F11/F12 retained semantics unclear | Product | Parity labels narrowed to P0_BOUNDED_* with explicit exclusions; PRD minimum behavior table; V31–V34 deny and V46–V49 allow | Needed clarification/intents/assurance/contract behavior absent or generalized Go legacy behavior treated mandatory | FIX_CANDIDATE |
| P2-01 PRD parity relative path broken | Product | PRD source ref fixed to `../../research/v2/TALLY_V2_FUNCTIONAL_PARITY_R0.md` | Link cannot resolve within doc tree | FIX_CANDIDATE |
| P2-02 report correction does not revoke advice | Product then L2 currentness | PRD explicit source correction → derived Context/Cost/Agent Decision STALE/REQUALIFY; V43 | Revoked source yields still-applicable recommendation or changes independent Task truth | FIX_CANDIDATE |

## Preserved invariants

- **Domain Package First:** Tally implementation is genuinely BUILT/selected installable Business/Capability Domain Packages using DomainHarness public K/S/Host, not standalone TS service later wrapped. No guessed public Manifest/ABI.
- **GitHub native fact and telemetry differ:** GitHub's actual Task/Claim/PR/Review/CI facts and valid ADS report revision carry their source-specific truth; task-meta is nonblocking observational producer, Tally reads its reports **only through GitHub**. GitHub source authority unavailability still blocks the dependent gate.
- **Local-only baseline:** runx/SSH/Docker remote provider optional, never a precondition for qualified Local single-Task+version completion.
- **Protected File and independence:** Logical execution provenance is not GitHub account, protected source path exception not automatic owner check.
- **Version closeout:** Finish version branch first, qualify exact candidate, then merge main only under source-currentness rules, reverify merged tree and target Tag.
- **R0 preparation not implementation:** 47 mapped feature domains (legacy `v1.5.5@b5e7b6e38ea38837e3bbcfc46c36586180705de0`), **49 Product vectors at older pre-review HEAD a97e6fa3**; current revised vectors are **67**; no Go/Node/Host physical tests executed this turn, no formal reviewer PASS or Release Qualification.

## Actual Claude adversarial pre-review (newer than GPT-6) — author remediation

[Full Claude PR #77 review #6381544471](https://github.com/kaicreator-mm/tally/pull/77#issuecomment-6381544471) inspected exact former head a97e6fa3e1c36a400d6912c6649b780eb0678c97. Reviewer declared Claude Opus 5.5, independent of author; P0=0/P1=5/P2=5, PRE_REVIEW_REWORK. **This is an actual Claude pre-review, not formal Product R1, and the source verdict remains REWORK**. The author-proposed changes below require independent recheck, not an automatic PASS.

| Finding | Product fix candidate | Negative/positive spec | Author status |
|---|---|---|---|
| P1-C1 Freeze authority and open P1 agenda | Drafts B/K; actual eligible Product/L2 review and Freeze are GitHub ADS source-qualified read-back artifacts. Project v2 exact ADS pin must be adopted before Freeze publication; unadopted pin is external HOLD, not silently resolved. §10 turns undefined P1 agenda into decisions/holds/deferred. | V50/V51/V65 | FIX_CANDIDATE |
| P1-C2 Real Local Agent unspecified | Codex CLI mandatory real Local Agent C provider; actual start/cancel/timeout/exit/receipt/workspace, optional Claude Code; Fake test-only. | V52/V53 | FIX_CANDIDATE |
| P1-C3 F21/F29 workspace/git isolation missing | P0-16 isolated clean-pinned per-attempt disposable worktree, candidate exact commit/tree freeze, disallow dirty/conflicting concurrent writers. | V54/V55/V56 | FIX_CANDIDATE |
| P1-C4 Hidden Local host trust boundary missing | P0-17 verifier-only fixtures after candidate freeze and Builder denial witness; same-user unrestricted read invalidates Hidden PASS. | V57/V58 | FIX_CANDIDATE |
| P1-C5 task-meta in-process hook stalls/exit nonzero | P0-10 bounded fail-open hook and decoupled publisher; preserve Agent actual exit and verifier; disable invalid Hook if unable to prove. | V59/V60 | FIX_CANDIDATE |
| P2-C1 Cancel and partial side effects | CANCEL_REQUESTED until process/child and remote receipts reconciled, no blind rollback/PASS. | V61/V62 | FIX_CANDIDATE |
| P2-C2 Parallel scope contradiction | P0 compute parallel-ready set but only N=1 executing Task; concurrent multi-Agent is P1. | V63 | FIX_CANDIDATE |
| P2-C3 Broken GitHub markdown table V31–V49 | One contiguous table with V01–V67, no blank-separated unheaded rows. | Static markdown verification | FIX_CANDIDATE |
| P2-C4 Counts/old Go HTTP source citation | Current V01–V67; v1.5.5 http.go 62–65 main routes, dispatch line 72, historic comments not rewritten. | Static source readback | FIX_CANDIDATE |
| P2-C5 App shell could hide second Business engine | Explicit domain-shell business state/rule leak prohibition, vector V64. | V64 | FIX_CANDIDATE |

Current authored evidence: **47 legacy parity domains, 67 Product acceptance vectors, P0-01..17** (all test vectors NOT_RUN). New source HEAD must be pinned after Output Ledger and PR metadata. No physical Node24 tests performed; no DomainHarness v0.10.2 ABI or npm Manifest freeze.

**Gate:** Distinct independent reviewer must check the newest exact PR #77 SHA, deliver P0/P1/P2 and Product PASS/BLOCK/CONDITIONAL, and separately determine PRODUCT_FREEZE_READY vs actual PRODUCT_FROZEN. While the lawful exact v2 ADS adoption pin is unresolved, a document may be SEMANTIC_FREEZE_READY_CONDITIONAL but **not published as PRODUCT_FROZEN**. Do not replace GitHub ADS fact with K draft state.

## Product R1 candidate review at earlier HEAD bda4fe71 — P1×2/P2×3 FIX_CANDIDATE

[Claude Opus 5.5 source-only Product R1 candidate review #6382724644](https://github.com/kaicreator-mm/tally/pull/77#issuecomment-6382724644) explicitly declares **not independent of prior Claude pre-review model/toolchain** and returned `PRODUCT_R1=BLOCK, P0=0/P1=2/P2=3` on `bda4fe717a928ad00941c6c6d28dc44e42755043`. This verdict remains historic and must not be misreported as a fully independent formal Product PASS.

| Finding | Repair decision / exact Product evidence | Status |
|---|---|---|
| **P1-R1** Builder shell bypasses K with inherited GitHub write credentials | **Choose model A:** Builder subprocess **must not carry GitHub write credentials**; only K-admitted C GitHub with immutable source/effect receipt may publish branch/PR/merge/tag. Unauthorized shell push/merge => `UNGOVERNED_EFFECT` and Task BLOCK. PRD P0-16 and Builder confinement; V66. Modify V61 to use the only lawful K→C remote publish path. | AUTHOR_FIX_CANDIDATE / no physical OS/network denial proof |
| **P1-R2** Hidden Local-only provider missing | Default P0 Local verification uses **distinct verifier OS account/principal + filesystem ACL** with **actual Builder deny-read witness**. Separate workspace names alone are inadequate. No Docker/runx needed; `HIDDEN_VERIFIER_UNAVAILABLE` blocks Hidden PASS but not ordinary editing; V67. | AUTHOR_FIX_CANDIDATE / actual isolation NOT_RUN |
| P2-R1 stale active counts | Gate references `P0-01..P0-17`, vectors current `V01..V67`; prior `49` is historically qualified to reviewed a97e6fa3. | AUTHOR_FIX_CANDIDATE |
| P2-R2 F39 disposition mismatch | Main matrix `F39` is now `REPLACE; P0_HOOK_FAIL_OPEN`; Measurement implementation fully replaced by task-meta/GitHub-only advisory, but Hook nonblocking P0 persists. | AUTHOR_FIX_CANDIDATE |
| P2-R3 stale weak Hidden V13 | V13 now directly defines Builder-readable fixture before candidate freeze => `HIDDEN_NOT_QUALIFIED`, harmonized with V57/V58; not a generic leak label. | AUTHOR_FIX_CANDIDATE |

**Execution evidence:** zero new Go/Node24/build/Host/GitHub Effect/Hidden tests. Product R1 narrow independent re-review must inspect exact latest SHA and these 5 amendments; only then can decide `SEMANTIC_FREEZE_READY`. The **separate** [#81 ADS v4.17.0 adoption PR](https://github.com/kaicreator-mm/tally/pull/81) was submitted in a *stacked version-staging branch* and is NOT merged into Product or main. A root `.dev-standard/VERSION` commit on this draft staging branch does not itself make Product Freeze, nor does it retroactively change historical Tally v1.5 baseline. Latest ADS v4.17 source qualification CONDITIONAL P2×3 remains disclosed.

## Product R2 repeated-Claude review and remaining P2 author repair

[Product R2 candidate review #6382926918](https://github.com/kaicreator-mm/tally/pull/77#issuecomment-6382926918) on prior exact `8115759cea3e56569b4f131f77964aca60c4a7bb` reports `P0=0/P1=0/P2=4`, `SEMANTIC_FREEZE_READY=CONDITIONAL_ON_ADS_PIN`, `PRODUCT_FROZEN=NO`, `MERGE_ELIGIBLE=NO`. Reviewer explicitly uses the **same Claude model/toolchain** as pre-review and R1, so it **does not qualify as distinct formal Product approval**. Preserve both earlier BLOCK and current conditional review history.

Author submitted all four P2 refinements, NOT independent closure:

| R2 P2 | Candidate fix | Evidence limit |
|---|---|---|
| P2-1 Markdown V66/V67 detached | Remove V65→V66 blank and add normal table→Coverage separation; total 67 unique rows | Static source review, not rendered UI test |
| P2-2 Remediation ledger says 65 current | Correct current to 67 while retaining original 49/65 historically pinned counts | Static source review |
| P2-3 Noninteractive elevation and token store gap | PRD Builder and verifier witness must probe sudo -n, admin/backup privilege, GH config/GCM/ssh-agent/delegated credential reachability, not merely scrub environment; strengthen V66/V67 | No actual ACL/privilege/credential denial test performed |
| P2-4 Builder confinement unavailable not named | Add `BUILDER_CONFINEMENT_UNAVAILABLE`; disallow CJ03/Release PASS, preserve Product read-only/editing UI | Real Agent/Host test NOT_RUN |

**Next:** only a distinct authorized independent Product reviewer may adjudicate the corrected source at newest exact PR #77 HEAD. Separate ADS v4.17 pin [PR #81](https://github.com/kaicreator-mm/tally/pull/81) has now been repaired to canonical format + A1 root override, yet downstream adoption remains explicitly blocked by `PROJECT_ADOPTION.md` §2.1C literal `RQ READY` vs actual independent `CONDITIONAL`; [ADS normative interpretation Issue #1318](https://github.com/kaicreator-mm/ai-development-standard/issues/1318) and [Tally #80](https://github.com/kaicreator-mm/tally/issues/80) own this gate. Neither PR may be merged into historical main as an incomplete Tally v2 Release.

**State:** `AUTHOR_R2_P2_FIX_CANDIDATE`, `FORMAL_DISTINCT_PRODUCT_REVIEW=NOT_RUN`, `PHYSICAL_TESTS=NOT_RUN`, `ADS_417_ADOPTION_PENDING_NORMATIVE_READY_GATE`, `PRODUCT_FROZEN=NO`.
## Next independent checks

1. Actual Claude pre-review DONE on earlier exact a97e6fa3 and returned REWORK; author fixes must be independently checked on newer source.
2. A different independent Product R1 reviewer inspects newest exact SHA and this remediation ledger, verifies each P1/P2 with source references, then issues P0/P1/P2, PASS/BLOCK, PRODUCT_FREEZE_READY and MERGE_ELIGIBLE **separately**.
3. Before L2 independent R1, reconcile approved Product changes into stacked #79 semantic L2 candidate; **physical** L2 Freeze still pending actual DomainHarness v0.10.2 public Package API and independently proven clean external producer/T09 + R01.
4. If Claude pre-review or formal Reviewer finds new P0/P1, fix Product docs in PR #77 with a **new exact SHA** and repeat narrowed independent review. Keep all GitHub output durable.

**Never infer a reviewer verdict from this author change ledger; `P1_CLOSED_BY_INDEPENDENT_REVIEW=UNKNOWN`.**


## 2026-10-28 Product Owner ALM full-v2 amendment (current HEAD candidate, NOT reviewed)

**Do not rewrite the original R0/R1/R2 history above.** Those rows, including historic P2-C2 `max1 active Task P0`, applied to their exact earlier HEADs. The current Product Owner direction explicitly **supersedes that old scope decision for full v2**: first authentic single Task is a delivery *slice*, while complete v2 P0 requires at least two actually authorized independent active Builder Task Attempts. The adopted ADS Claim/Host/K and nonoverlapping write-set requirements remain; this does **not** authorize concurrency before native DAG admission. Previous reviews/Freeze claims do not transfer to the new HEAD.

Updated authoritative **author candidate** source in this Product PR:
- [PRD](./TALLY_V2_PRD_DOMAIN_PACKAGE_FIRST_R0.md): P0-01..P0-24, extending core to agent-native ZenTao/Jira-style requirements, design, development, independent review/defect, QA/acceptance, source-qualified RQ/release, maintenance, evidence-grounded Task Learning and next-task adoption. Explicit Tally management role vs external ADS-speaking Agents; real selected DomainHarness B/S/K/C/Host, no second Agent runtime/Kernel/SoR.
- [Product vectors](./TALLY_V2_PRODUCT_ACCEPTANCE_VECTORS_R0.md): **82 unique contiguous V01–V82** (67 historic + 15 new ALM); V63 now validates real safe parallelism rather than old max1 P0. New vectors are `NOT_RUN`, not physical or QA successes.
- Source-linked bilingual ALM design intake remains [Draft PR #147](https://github.com/kaicreator-mm/tally/pull/147); formal Product #76 and L2 #82 must each independently decide/adopt final accepted scope.

**Legacy 47-feature parity R3 P1 still open.** This change does not dismiss original source-parity evidence. Formal reviewer must either verify a required historical behavior or obtain an explicit Product Owner `RETAIN/ADAPT/RETIRE/UNKNOWN` disposition for each genuinely contested capability, without shrinking unmodified P0 security/release evidence by assumption.

**Next required gate:** independent exact-HEAD Product reviewer at latest PR #77 head, including P0-18..24, V63 and V68..82, ADS adoption currentness, source parity R3 P1, and feasibility/independence of B/S/K/C/Host. Only after Review+Freeze/owner decisions can L2 #82 and #86 authorized DAG adapt. Until then `PRODUCT_FREEZE_READY=NO / NEW_ALM_SOURCE_CLAIM=NO / MAIN_MERGE=HOLD`.
