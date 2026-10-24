# Tally v2 Product acceptance vectors and falsification plan — R0

**Status:** SPECIFICATION_ONLY / NOT_RUN / NOT_PRODUCT_FROZEN  
**Ownership:** [Product #76](https://github.com/kaicreator-mm/tally/issues/76) / [PRD R0](./TALLY_V2_PRD_DOMAIN_PACKAGE_FIRST_R0.md).  
**Formal runtime gate:** Node24 clean installed external Tally Business Domain Package built by qualified DomainHarness v0.10.2 public BUILD/Host/SDK, then Local real E2E. **No mock-only gate can substitute.**

## Traceable journey vectors

| Vector | Source family | Stimulus / precondition | Expected independently observable result | Phase / evidence |
|---|---|---|---|---|
| V01 | F01–05 | User edits/revises requirements, PRD and L2 via UI | Typed revision; independent source-based review, separate Product and L2 freeze status | Spec now; physical NOT_RUN |
| V02 | F01–05 | User changes PRD after review at old source SHA | Freeze invalidated or requires new applicable review; no stale green label | Spec now; NOT_RUN |
| V03 | F06–07 | User submits cyclic Task DAG | Rejected with cycle/witness; no GitHub Task admission | Spec now; NOT_RUN |
| V04 | F06–08 | Accepted DAG has dependency blocked | Ready candidates exclude downstream node; explain reason | Spec now; NOT_RUN |
| V05 | F08–09 | User-level AUTHORITY_REQUIRED gate has no valid decision | Task/node cannot complete, UI shows missing decision | Spec now; NOT_RUN |
| V06 | F09–10 | Agent claims success without verifier/evidence | Completion remains UNSATISFIED/UNKNOWN, independent review pending | Spec now; NOT_RUN |
| V07 | F13–19 | Clean Local-only node, no runx or SSH config | Real Task can dispatch, verify and complete once with selected K/C permitted Effect | Requires DH; NOT_RUN |
| V08 | F13–19 | No accepted Claim or wrong Capability | Deny dispatch without external mutation; bounded clear reason | Requires DH; NOT_RUN |
| V09 | F13–19 | External Local Tool returns accepted but no terminal receipt | Remain RUNNING/UNKNOWN; no false PASS | Requires DH; NOT_RUN |
| V10 | F13–19 | Host crashes after accepted stateful operation and ACK lost | Journal read/reconcile; no blind duplicate effect | Requires DH; NOT_RUN |
| V11 | F13–19 | Same idempotency key reused with different Task input | Typed CONFLICT; no successful reuse | Requires DH; NOT_RUN |
| V12 | F21–27 | Candidate changes after test source SHA pinned | Old review/verification not current; retry before completion | Requires DH; NOT_RUN |
| V13 | F24–27 | Hidden/held-out fixture before candidate freeze | Block leak/contamination; independent gate evidence | Requires DH; NOT_RUN |
| V14 | F24–27 | Failing verifier, Agent requests automatic acceptance | No Task PASS; repair/escalation paths only | Requires DH; NOT_RUN |
| V15 | F28–30 | GitHub readback differs from accepted Issue/PR revision | Reconcile/UNKNOWN; no false mutation completion | Spec now; NOT_RUN |
| V16 | F28–30 | GitHub 403/429 or partial pagination | Coverage PARTIAL; bounded retry, never assume all native facts fetched | Spec now; NOT_RUN |
| V17 | F28–30 | Accepted Task already exists, proposed Replan drops it | Reject or preserve accepted evidence according to frozen policy | Spec now; NOT_RUN |
| V18 | F33–40 | task-meta collector process exits/timeout/queue-full | Same Agent/Verifier terminal engineering outcome; metrics UNKNOWN, not 0 | Requires DH; NOT_RUN |
| V19 | F33–40 | Task Report delayed, duplicated or corrected | Report identity/attempt/correction source pinned, no double counting | Spec now; NOT_RUN |
| V20 | F33–40 | GitHub Report uses incompatible ADS dialect | Typed UNSUPPORTED/INSUFFICIENT_EVIDENCE; never silently downgrade | Spec now; NOT_RUN |
| V21 | F40–44 | Cache has stale Issue/PR status while source changed | UI labels stale and refetches before any authority-changing Effect | Spec now; NOT_RUN |
| V22 | F40–44 | User does not know TaskExecutionRequest JSON | Can submit legal Task through ordinary UI structured forms | Requires DH; NOT_RUN |
| V23 | F42–47 | PR and local Task tests PASS but Full/Hidden/CJ absent | Version Closure BLOCKED/NOT_RUN; cannot publish qualified Tag | Requires DH; NOT_RUN |
| V24 | F42–47 | Independent Full/Hidden/CJ PASS on same qualified candidate | RQ decision with exact source, authorized tag/baseline, truthful required gates | Requires DH; NOT_RUN |
| V25 | F41–44 | A new chat has no prior context | All decisions, reports, PRD and exact SHA recover from GitHub ledger/Issue | GitHub persistence check |
| V26 | F13–19 | C GitHub/Local effect operation attempted directly from UI bypassing K | Physical package import/call path denial; no ungoverned mutation | Requires DH; NOT_RUN |
| V27 | F18/F31–36 | Incomplete telemetry used for Agent-cost comparison or Experience promotion | Advisory INSUFFICIENT_EVIDENCE, no qualified decision or Package promotion | Spec now; NOT_RUN |
| V28 | F45 | Secrets embedded in Agent output/error/log | Sanitized user projection and GitHub comment; no credential echo | Spec now; NOT_RUN |
| V29 | F46/F47 | Build contains package imports to private DH src or compiler in runtime | Clean installed external producer/consumer refuses; runtime BUILD closure fail | Requires DH; NOT_RUN |
| V30 | F13–19 | Package revision changes while previous occurrence runs | Old occurrence remains immutable selected pin; new BUILD separate | Requires DH; NOT_RUN |

| V31 | F02 | A Product ambiguity remains unresolved after preliminary clarification | UI records question/answer and marks critical requirement OPEN; Product Freeze blocked until applicable review | Spec now; NOT_RUN |
| V32 | F03 | Previously accepted intent source changes materially after freeze | Applicable frozen intent/review eligibility revoked or requalified, never stale green | Spec now; NOT_RUN |
| V33 | F11 | Declared high assurance Task lacks required verifier/authority scope | Visible required checks and missing gate; no readiness/completion | Spec now; NOT_RUN |
| V34 | F12 | Task Contract has unsupported adopted ADS dialect or missing required field | Typed UNSUPPORTED / L3_NOT_ADMITTED; no silent fallback to historical schema | Spec now; NOT_RUN |
| V35 | F26 | Builder modifies protected ADS or held-out/auth policy file without recorded grant | Guarded change is denied or escalated; no automatic acceptance | Requires DH; NOT_RUN |
| V36 | F26/F27 | Same execution lineage changes displayed role from Builder to Reviewer | Independent-review check rejects role laundering despite same/different GitHub login | Spec now; NOT_RUN |
| V37 | F45 | User UI session expired/revoked before GitHub/Local mutation | Reject prior to effect; no real provider operation | Requires DH; NOT_RUN |
| V38 | F28/F45 | Authenticated operator lacks grant for the target repo or request names another project | Wrong repo/scope denied despite a syntactically valid K intent | Requires DH; NOT_RUN |
| V39 | F45 | Cross-origin forged POST/CSRF attempts a protected UI action | Transport/action boundary blocks mutation and avoids publishing credentials | Requires DH; NOT_RUN |
| V40 | F28/F45 | Provider credential revoked after initial UI read, before K/C mutation | Provider denial and reconciliation; never infer success from stale UI permission | Requires DH; NOT_RUN |
| V41 | F28/F45 | Mutation request accepted locally but GitHub target cannot be read back | Maintain UNKNOWN; block related merge/tag/approval pending remote truth | Requires DH; NOT_RUN |
| V42 | F47 | Full RQ PASS at exact version head but target main advances or merge produces unverified tree | Compare candidate and target exact revision, revalidate/re-RQ changed merged scope; no tag without qualified final commit | Requires DH; NOT_RUN |
| V43 | F33–40 | Previously qualified task-meta report is corrected/revoked after Agent ranking/Context advice issued | Invalidate dependent advice and label STALE/REQUALIFY; retain independent Task/CI truth, no cost double-count | Spec now; NOT_RUN |
| V44 | F06/F08 | DAG Ready and Issue exists, but L3 Tests/Contract/Implementation/Failure Handling/Reference missing/stale | L3_NOT_ADMITTED; offer fix/publish/readback UI; no dispatch Effect | Requires DH; NOT_RUN |
| V45 | F23/F25 | L3 spec was complete before dispatch but real implementation/verification receipts absent afterward | Implementation Evidence remains NOT_RUN/UNKNOWN; no Task PASS from pre-run plan | Requires DH; NOT_RUN |

## Coverage and review protocol

- **Product evidence:** F01–F47 domain inventory and 45 vectors. These are test *requirements*, not executed test results.
- **Physical Node24 tests after actual DH qualification:** real public Contracts/Manifest/ABI, producer with a Tally Engineering B/Capability package from outside DH repo, immutable artifact SHA, selected BUILD, Local Host persistence, C GitHub and Local Agent with K Effect admission.
- **Source and authority:** actual latest GitHub Issue/PR/Review/CI and adopted ADS report versions, with transport pagination/error and cache staleness. A successful parser is neither evidence freshness nor Task PASS.
- **Separation:** Functional tests vs release-level Full Regression, Hidden Validation, Critical Journeys, independent Product/L2/Implementation review and Release Qualification. Do not replace any stage with historical v1.5.5 result.
- **Test outcomes:** PASS only with command/environment/exact SHA/exit/witness; otherwise FAIL/NOT_RUN/ENV-BLOCKED/UNKNOWN. Future tests must separately record mocked fixture assertions from real Provider/Host/Package runtime tests.
- **Reviewer blockers:** any contradiction about Domain Package First, collector independence, GitHub truth/cache, complete ADS UI, Source pin/revision gates, or full RQ before Tag should be classified P0/P1 before Product freeze.

## Consumer-only contract sketches (business semantics, not ABI)

- ProductDecision = `{projectRef, versionRef, requirementRevision, sourceRefs, decisionKind, authorityRef?, status}`.
- TaskProposal = `{taskRef, dagRevision, dependencyRefs, claimRef?, acceptance, requiredGates, sourcePin}`.
- VerificationObservation = `{candidateSha, checkRef, status, verifierRole, sourceRevision, coverage}`.
- EvidenceProjection = `{kind, sourceId, sourceShaOrRevision, adsDialect?, coverage, freshness, observedAt, payloadRef}`.
- TelemetryCoverage = one of `OBSERVED / PARTIAL / UNAVAILABLE / UNSUPPORTED`; never coerced to cost 0.
- RemoteOperationReceipt = `{idempotencyKey, accepted?, terminal?, receiptRef?, sourceRevision?, reconciliationState}` (actual typed C/K boundary **PENDING_DH**).
- PackageSelection = `{buildRef, selectedManifestDigests, occurrencePin, provenance}` (actual physical SDK manifest **PENDING_DH**).

These shape sketches are illustrative Product-level vocabulary and **MUST NOT be directly published as DomainHarness public ABI or executable payload before L2 review**.

## Future gates

1. P01 source parity Product reviewer checks rows/status and key call sites.
2. P02 Product reviewer checks P0-01…P0-15 plus V01…V30, fix P0/P1, Product Freeze if independently eligible.
3. Semantic L2 maps package boundaries; leave actual physical ABI open until qualified published DH artifacts exist.
4. Physical L2/API, executable Task DAG, production Domain Packages and UI/E2E only after gate.
5. Full Release Qualification / final tag on exact frozen version source.

**Current result:** `DESIGN_VECTORS=45/45_DRAFTED`, `PHYSICAL_TESTS_EXECUTED=0`, `PRODUCT_REVIEW=NOT_RUN`, `RELEASE_PASS=NO`.
