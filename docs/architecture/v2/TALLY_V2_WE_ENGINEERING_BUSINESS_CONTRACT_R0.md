# Tally v2 W-E — Engineering Business Domain Package contract and ADS v4.18 Task DAG/L3 R0

**Owner:** [#91](https://github.com/kaicreator-mm/tally/issues/91). **State:** SOURCE_CONTRACT_CANDIDATE / PUBLIC_BUILD_IMPORT_PHYSICAL_NOT_RUN / CLAIM_NOT_ADMITTED. Product [#77](https://github.com/kaicreator-mm/tally/pull/77) 17 P0/47 Fxx/67 Vxx, semantic L2 [#82](https://github.com/kaicreator-mm/tally/pull/82), ADS v4.18 target [#85](https://github.com/kaicreator-mm/tally/pull/85), public DH [#87](https://github.com/kaicreator-mm/tally/issues/87). Not a standalone Go/TS server wrapped as Package.

## Exactly one selected K; bounded B modules

| Business concern | B-owned source contract | K/External truth unchanged | C/S connection |
|---|---|---|---|
| Engineering Product | requirements/PRD revisions, proposed Product review/freeze, source impact and validated operator intent | GitHub real Issue/PR/review and accepted frozen source; K app intent only, cannot mint GitHub Review | C GitHub actual source readback, S Context optional |
| Engineering Task | DAG plan/topology/TaskContract/L3, Agent suitability and pending/frozen Task intent, retry/repair decision | Native GitHub Dependencies, Claim and Issue status, K Admission/State/CAS/Effect; no parallel business runner | S Decision/Workflow optional, C Local + C GitHub |
| Engineering Assurance | tests/contract/implementation plan/review/validation/Hidden/Critical/Release proposals; source-qualified failure classifications | Real independent Reviewer, CI/Host run, exact Git commit/tree/RQ/Tag. B cannot self-sign PASS | C verification/Host/GitHub; K permits only valid effects |
| Engineering Experience | bounded rule/skill/procedure/workflow Candidate, applicability and feedback recommendation | GitHub Task Report/Task Learning qualified from ADS; missing task-meta metric UNKNOWN and cannot block actual Task | S Context/Shadow optional, C GitHub only for telemetry |

**Packaging rule:** the four responsibilities are **logical B capability modules**, not automatically four npm packages. Build-selected Business Package split must maximize small independent task ownership while satisfying actual public DomainHarness BUILD/Tiny/Rich module signatures. No guessed public SDK interfaces, no direct SQL or native GitHub credential in B, no private `src/**`/compiler import. If K/S/C unavailable or not selected, deny source effect, do not implement alternative mutable scheduler.

## Proposed typed B command/effect intent vocabulary (not public DH API names)

| Proposed B input | Reasoned output | External side effect and gate |
|---|---|---|
| `ProposeRequirement` / `RevisePRD` | source-qualified PRD draft/impact recommendation | GitHub publish only K→C with expected SHA |
| `ProposeFreeze` | validity of real Reviewer/source/standards pin predicates | Cannot mutate Freeze without true GitHub Review/authority/readback |
| `PlanWorkfronts` | graph with independent lanes, dependency reasons, agent/Host/CI/Human fit | Graph stays proposal until canonical native GitHub Issue Dependencies and Claims |
| `ProposeTaskAdmission` | check L3 Tests/Contract/Implementation/Failure Handling/Reference, current Claim/repo rights | K admits N=1 executing Attempt, C Local real process only |
| `RecordAttemptObservation` | exact process/Verifier/Effect receipts or UNKNOWN | Observation not Task PASS on Agent assertion alone |
| `ProposeRepair` | bounded retry/alternate plan preserving first Attempt, accepted work and source | Protected files, role separation, original failed tests still gate |
| `ProposeRelease` | applicable Full/Hidden/CJ/Packaging + independent RQ source tuple | GitHub version/main/Tag only after owner grant/currentness; PR PASS ≠ Release PASS |
| `ProposeLearningFeedback` | bounded source-linked experience feedback, missing metrics UNKNOWN | C GitHub emits ADS v5 feedback after K permit, never task-meta private API |

## ADS v4.18 Task DAG topology and live authority

Plan **step/stage flow plus parallel sub-requirements**: Workfront A public Host ABI; B GitHub Reader; C SQLite; D Local Agent; E Business; F UI all have independent *source contract* tasks; actual runtime E2E requires a join of applicable accepted Host, B/C/D/E contracts, Product+ADS+L2 currentness, DH Release and native Task L3. UI joins before minimal complete ADS Critical Journeys but UI source prototype is not forced to wait for C/D runtime. Avoid artificial linear T01→T02→… if capability splits don't actually depend.

At runtime native GitHub Issues and `blocked_by` edges are authoritative; planning DAG is a *document* and never itself a durable Claim. `ready_builder` check must re-read native Issue/dependencies and exact Task Pack, verify accepted CAS Claim and build/Host selected Package source before running. Role is self-declared and evidence-checked, never determined only by GitHub account.

## Negative contract and acceptance

| ID | Scenario | Required disposition |
|---|---|---|
| BE-N01 | Product Freeze projected from local K cache without independent GitHub Review | HOLD / NOT_FROZEN |
| BE-N02 | DAG document has edge but native Issue dependency differs | do not dispatch, re-read native |
| BE-N03 | Workfront independently Ready but unrelated blocker added to another lane | only affected lane blocked, not all planning serialized |
| BE-N04 | Execution Task lacks L3 required Test or Failure Handling | L3_NOT_ADMITTED |
| BE-N05 | Builder role renames as Reviewer, claims self-PASS | independent Review requirement remains blocked |
| BE-N06 | Fake Agent returned success without actual terminal | LOCAL_AGENT_UNAVAILABLE/NOT_QUALIFIED |
| BE-N07 | More than one concurrently EXECUTING Task Attempt | second K NOT_ADMITTED |
| BE-N08 | Native GitHub merge unsuccessful but local B says integrated | external currentness UNKNOWN, no Release PASS |
| BE-N09 | Correction of Task Report after cached learner output | stale advisory invalidated; no retroactive Task status |
| BE-N10 | task-meta collector fails/hangs mid Task | telemetry UNKNOWN; Task/Verifier continues |
| BE-N11 | repaired code includes protected ADS/Hidden fixture edit | B protected-path deny + actual verifier and independent Review |
| BE-N12 | same Claim/messageId replay changed input | typed conflict, K controls outcome |
| BE-N13 | group different B modules share single instanceKey | explicit target selection, no cross-B read authority |
| BE-N14 | disabled optional S Feature | B uses legal minimal behavior or typed unavailable, no auto feature import |
| BE-N15 | GitHub grant revoked between K Permit and C write | deny/reconcile, no assumed side effect |
| BE-N16 | runtime/Host lost after accepted queue ACK | UNKNOWN until real Host observation; not terminal |
| BE-N17 | attempting cross-version Task pin upgrade mid execution | preserve accepted immutable source; fail closed |
| BE-N18 | Product source candidate changed after reviewer PASS | new SHA requires applicable delta review/re-Freeze |

## Release versus implementation

**Source contract review NOW:** independent check of real public DH selected B manifest form, business separation, v4.18 DAG topology/agent fit, 18 negative cases. **Future production:** L3 on each concern, native Issue/Claim and exact ads pin, true DH public package install/BUILD. Local Node24 tiny/Rich contract smoke, actual verifier and governed C GitHub e2e. Full/Hidden/CJ/RQ once at version closure with reuse of unaffected validations. No production B class/npm Package interface claimed to exist yet.

**Review output:** P0/P1/P2, source Business/Kernel boundaries, mapping to Product P0/Fxx/Vxx, DAG parallelism and legal effect lines. `SEMANTIC_B_CONTRACT=REVIEW_PENDING`, `IMPLEMENTATION=NOT_RUN`.
