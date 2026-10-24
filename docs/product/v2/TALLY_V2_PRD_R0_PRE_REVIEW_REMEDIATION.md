# Tally v2 Product R0 — GPT-6 adversarial pre-review remediation ledger

**Status:** AUTHOR_REPAIR_CANDIDATE / CLAUDE_MODEL_DIVERSITY_REVIEW_NOT_RUN / FORMAL_PRODUCT_R1_NOT_RUN / PRODUCT_FREEZE_READY=NO  
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
- **R0 preparation not implementation:** 47 mapped feature domains (legacy `v1.5.5@b5e7b6e38ea38837e3bbcfc46c36586180705de0`), 49 Product vectors now specified; no Go/Node/Host physical tests executed this turn, no formal reviewer PASS or Release Qualification.

## Next independent checks

1. Actual Claude agent, if required for model diversity, reviews **latest refreshed exact PR #77 HEAD**, not the already reviewed `3935e2d6...` source. Record role as CLAUDE only if an actual Claude execution occurs; full findings and closeout posted to #77 and #76.
2. A different independent Product R1 reviewer inspects newest exact SHA and this remediation ledger, verifies each P1/P2 with source references, then issues P0/P1/P2, PASS/BLOCK, PRODUCT_FREEZE_READY and MERGE_ELIGIBLE **separately**.
3. Before L2 independent R1, reconcile approved Product changes into stacked #79 semantic L2 candidate; **physical** L2 Freeze still pending actual DomainHarness v0.10.2 public Package API and independently proven clean external producer/T09 + R01.
4. If Claude pre-review or formal Reviewer finds new P0/P1, fix Product docs in PR #77 with a **new exact SHA** and repeat narrowed independent review. Keep all GitHub output durable.

**Never infer a reviewer verdict from this author change ledger; `P1_CLOSED_BY_INDEPENDENT_REVIEW=UNKNOWN`.**
