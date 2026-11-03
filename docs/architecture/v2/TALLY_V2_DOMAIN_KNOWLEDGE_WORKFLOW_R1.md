# Tally v2 — Engineering Domain Data & Bounded Workflow Contract R1

> **Status:** DESIGN_ONLY / SOURCE_AUDIT / IMPLEMENTATION_NOT_ADMITTED. This is the detailed Lane F design artifact proposed under [Issue #276](https://github.com/kaicreator-mm/tally/issues/276), carried by subordinate docs-only [PR #277](https://github.com/kaicreator-mm/tally/pull/277). It is not a Product/L2 Freeze, executable Task Pack, CAS Claim, independent review, BUILD qualification, Host acceptance or Release decision.
>
> **Source pins:** Tally `version/v2@7ebb1cdd0a79fc82cc938e35f852a80dd8e15332`; DomainHarness source tag `v0.10.3@061376f4a023070f215f9feffad5c24d96d503cb`. ABI, BUILD profile, installed Runtime and Host remain distinct pins.

## 1. Purpose and architectural decision

Tally v2 is an agent-native Engineering ALM Domain App. Lane F must make the Engineering domain model explicit without turning Tally into a generic BPM engine or creating a second Kernel, scheduler, Journal, Task database or Agent runtime.

Keep one physical `packages/tally-engineering-business` package and its current public Business ABI unless a separately reviewed compatibility change is required. Organize its domain semantics into five logical bounded contexts:

| Context | Responsibility | Workflows |
|---|---|---|
| B1 Product / Requirement | Project identity, requirement intent, Product revision and acceptance | WF01, part of WF02, WF07 |
| B2 Architecture / Planning | Architecture/L2 decisions, Version and PlanRevision, DAG proposals | WF02, WF03, WF07 |
| B3 Execution / Journey | WorkUnit, accepted Claim, Attempt, Agent/Host handoff | WF04, WF05 |
| B4 Assurance / Release | Independent Review, Repair, Verification, QA and Release qualification | WF05, WF06 |
| B5 Improvement / Experience | Maintenance feedback, Experience qualification and controlled reuse | WF07, WF08 |

These are logical business boundaries, not five independently installable Business Packages or five K instances.

### 1.1 Non-interchangeable concepts

- **Domain Process:** a Tally-owned business lifecycle and its invariants.
- **Workflow Definition:** versioned declarative description of triggers, inputs, states, guards, waits, evidence and recovery.
- **K Execution:** the one selected DomainHarness K owns admission, durable state/CAS, Journal, Effect Permit, replay and UNKNOWN.
- **UI Journey:** a human-facing projection of the next actionable step; it is not the business process or its authority.
- **Agent Skill Flow:** a bounded proposal/execution technique; it cannot grant permission or change the domain state.
- **C Operation:** a narrow selected capability. Its handler may use an approved remote MCP/HTTP/CLI or local Host adapter; C is not itself an external Tool.
- **Host Port:** trusted provider credentials, environment, process execution and physical persistence.

No assumption is made that DomainHarness v0.10.3 automatically parses `domain/workflows/*.yaml`, loads every file under `domain/`, or executes arbitrary Workflow YAML. Assets enter a runtime only through an explicit, tested BUILD selection and consumer contract.

### 1.2 Authority layers

| Fact / decision | Authority | Tally B responsibility |
|---|---|---|
| GitHub Repository, Issue, PR, Commit, Review, Check, Release/Tag and native Issue dependency | GitHub native source | Interpret and project; never mirror as a competing source of truth |
| ADS Claim, Agent role, Review, Task Learning and report schema semantics | Exact adopted ADS version/profile | Apply the normative contract; do not redefine it |
| Admission, accepted CAS, K State, Effect Permit, Journal, replay and UNKNOWN | One selected DomainHarness K Primary | Request and interpret; never create a second authority |
| Process, local Agent, filesystem, SQLite physical writes, provider/session and environment result | Trusted Host / actual provider | Consume verified receipts; do not infer physical success |
| Curated engineering vocabulary, domain rules, applicability, counterexamples and Workflow definitions | Tally Engineering B | Own and version the business meaning |
| Private execution telemetry / collector internals | task-meta | Do not query private API, database, manifest or Trace; consume only eligible ADS-published GitHub evidence |

A GitHub comment that says “Claim accepted” is not by itself proof of the K CAS row. A Task Issue is not automatically an executable WorkUnit. An Agent report is not an independent Review. A merged PR is not Release PASS. A missing metric is UNKNOWN, not zero.

## 2. Canonical domain glossary and identity model

Use a stable, Tally-owned namespace for new Domain Data IDs. Proposed pattern: `tally.domain.<concept>/v1`, `tally.workflow.<process>/v1`, and `tally.rule.<invariant>/v1`. The `v1` here versions the Tally domain contract; it is not Tally app v1, DH v0.1, Business ABI v08 or V10 BUILD.

| Canonical concept ID | Meaning / minimum identity | Source of authoritative facts | Tally B may own |
|---|---|---|---|
| `tally.domain.project/v1` | Project scope anchored to repository identity and an admitted Project record | GitHub repository + selected K/Host admission receipt | Goal, product scope, constraints, display projection |
| `tally.domain.requirement/v1` | Requirement statement, acceptance criteria, revision and source refs | Versioned GitHub issue/PR/commit evidence | Requirement semantics, decomposition, acceptance interpretation |
| `tally.domain.product-revision/v1` | Immutable Product intent revision, digest and predecessor | Exact GitHub source commit/PR and qualified Owner/Product decision | Product meaning and change-impact rules |
| `tally.domain.architecture-decision/v1` | ADR/L2 decision, alternatives, constraints and consequences | Versioned GitHub design artifact + independent review | Architecture vocabulary and domain constraints |
| `tally.domain.version/v1` | Version identity and exact source/Product/L2 pins | Git ref/commit and approved Product/L2 sources | Version lifecycle and compatibility interpretation |
| `tally.domain.plan-revision/v1` | Append-only planning revision and predecessor | K activation receipt + exact source and Owner-approved candidate | Replan semantics, accepted-work preservation rules |
| `tally.domain.task-pack/v1` | Exact Task roster, issue/body identity, write sets, dependency snapshot and digest | GitHub native Issues/dependencies + committed candidate blob + K freeze receipt | Pack construction rules and stale/complete interpretation |
| `tally.domain.work-unit/v1` | Typed unit derived from a Task Issue Container, including key, kind, write set and source | GitHub Issue body and exact source refs; Claim remains separate | WorkUnit typing, collision/overlap analysis |
| `tally.domain.task-issue-container/v1` | GitHub Issue container that may describe a task, review, repair, campaign or other work | GitHub Issue | Classification/projection only |
| `tally.domain.native-dependency/v1` | One real GitHub `blocked_by` relation | GitHub native dependency API/readback | Readiness rules; never promote a textual edge to native |
| `tally.domain.claim/v1` | ADS-compliant Claim request/identity and accepted execution scope | ADS protocol + K CAS acceptance; GitHub evidence is a published reference | Business eligibility and display projection |
| `tally.domain.task-attempt/v1` | One immutable execution attempt bound to WorkUnit, Claim, actor, environment and source SHA | K/Host event and physical process receipt; ADS published summary where available | Attempt classification/read model, not fabricated execution |
| `tally.domain.review/v1` | Independent review decision for one exact source SHA/tree | GitHub PR review / ADS-compliant review record | Review requirements and applicability |
| `tally.domain.finding/v1` | P0/P1/P2 finding with source range, evidence and disposition | Exact review/issue/PR record | Finding taxonomy and remediation rules |
| `tally.domain.verification/v1` | Command, environment, exact source, exit code and test/CI evidence | Actual Woodpecker/Host/Browser result | Required verification policy and evidence projection |
| `tally.domain.release/v1` | Release candidate, exact merged commit/tree, tag and qualification | GitHub PR/commit/tag/Release + independent RQ/Owner/Host evidence | Release eligibility rules; never mint external authority |
| `tally.domain.defect/v1` | Defect and affected release/requirement/source | GitHub Issue + exact reproduction/verification refs | Defect classification and impact |
| `tally.domain.experience-ref/v1` | Curated reference to an evidence-backed Task/Attempt/Review/Outcome | ADS-published GitHub report and original GitHub/CI/Host facts | Applicability, exclusions, counterexamples and use/reject decision |
| `tally.domain.asset/v1` | Versioned Rule/Pattern/Recipe/Skill/Workflow/Validator candidate or qualified asset | Tally-owned versioned source plus source/validation refs | Asset meaning and qualification lifecycle |
| `tally.domain.package-build-pin/v1` | Exact selected Package/ABI/BUILD/runtime dependency closure and digest | BUILD manifest, source/tag, package/tarball digest and installed Host receipt | Compatibility policy and currentness projection |

### 2.1 Identity distinctions that must remain explicit

1. `TaskIssueContainer` ≠ `WorkUnit` ≠ `Claim` ≠ `Attempt`. A single Issue may contain several WorkUnits; an accepted Claim is a separately verified authorization; an Attempt is one execution instance.
2. `PlanRevisionCandidate` ≠ activated `PlanRevision`; a proposed dependency ≠ native GitHub `blocked_by`; an exact Task Pack blob ≠ a frozen DAG without native readback and Owner/K evidence.
3. `Review` ≠ `Verification` ≠ `RQ` ≠ `Release`. Each has a distinct authority and exact-source binding.
4. `ExperienceRef` ≠ promoted `DomainAsset` ≠ BUILD-selected asset. Promotion, BUILD qualification, activation and actual usage are separate events.
5. `PackageBuildPin` must distinguish Business ABI, DomainHarness source tag, V10 BUILD profile, package dependency closure and installed Host. Never collapse them into one version string.

## 3. Domain Data asset contract

### 3.1 File-first layout

The following is a target organization, not a magical Runtime loader contract:

```text
packages/tally-engineering-business/
  domain/
    glossary/
    authority/
    schemas/
    rules/
    knowledge/
    patterns/
    recipes/
    templates/
    skills/
    workflows/
    tool-policies/
    examples/
      positive/
      negative/
      edge-cases/
    validation/
  src/
    domain-model/
  test/
    domain-model/
    workflows/
    domain-assets/
```

Create only directories with real, source-linked assets and consumers. Do not add empty folders or copy entire ADS schemas, GitHub records, task-meta traces or Kernel state into B. Existing public `getPolicy()`/BUILD contract remains the compatibility boundary unless a separately reviewed change says otherwise.

### 3.2 Minimum metadata for a Domain Asset

Each asset should carry:
- `assetId`, `assetKind`, `assetVersion`, `ownerContext`;
- `status`: `OBSERVED | CANDIDATE | VALIDATED | SHADOW_VERIFIED | REJECTED | QUALIFIED`;
- `sourceRefs[]`: exact GitHub/ADS/CI/Host references and source SHA where applicable;
- `authorityRefs[]`: pointer to the owner of each fact/decision;
- `appliesTo[]`, `exclusions[]`, `counterexamples[]`;
- `validationRefs[]`: positive, negative, boundary, recovery evidence;
- `supersedes[]` and `supersededBy` where applicable;
- `buildQualification`: `NOT_SELECTED | SELECTED_UNQUALIFIED | QUALIFIED_ON_PIN` plus exact BUILD/package pin.

Do not add runtime mutable state to a Domain Data file. K owns live Workflow occurrence state and CAS. Domain Data is versioned definition and evidence-linked knowledge.

### 3.3 Schema and rule discipline

- Use JSON Schema Draft 2020-12 for structure and typed boundaries; validate deterministically in the Producer/BUILD or a tested B helper.
- Use pure B functions for business invariants: source-currentness, accepted-work preservation, DAG shape, WorkUnit identity/collision, evidence qualification and Experience applicability.
- Keep source facts in their native authority. A B schema may validate a GitHub Issue projection but cannot become the canonical GitHub Issue store.
- A Rule is not a paragraph; it needs deterministic input/output, positive/negative/boundary fixtures and a named owner.
- A Skill is bounded semantic guidance, not automatically a Tool, C Operation, Workflow executor or K Permit.
- A Recipe is a reproducible procedure with prerequisites, side-effect classification, verification and recovery.
- A Pattern/Knowledge item must retain applicability, exclusions, counterexamples and source evidence; one success narrative cannot become a universal rule.

## 4. Eight bounded Engineering Workflows

The following states are the proposed **domain lifecycle vocabulary**. They are not claims that current K has eight independently installed Workflow instances. The live state must remain in the one K Primary; UI Journey state is only a projection.

### WF01 — Project Intake

- **ID / owner:** `tally.workflow.project-intake/v1` / B1 Product & Requirement.
- **Trigger:** a user requests a new Project or submits a new/changed Product/Requirement baseline.
- **Inputs:** repository identity, goal/non-goals, constraints, requirement candidates, acceptance criteria, owner/operator scope, source refs and current source SHA.
- **Authority:** GitHub repository/Issue/PR facts; Product Owner decision; independent Product Review; ADS role/review contract; K CAS for admitted effects.
- **Target states:** `DRAFT → EVIDENCE_GATHERING → PRODUCT_REVIEW_PENDING → PRODUCT_REVIEWED → PRODUCT_FROZEN → PROJECT_ADMISSION_PENDING → ADMITTED`. Side outcomes: `REJECTED | BLOCKED | STALE | UNKNOWN`.
- **Business mapping:** existing `freezeProductIntake` and `admitProjectIntake` are the candidate Business Intents (T210 selected route); `guided-intake.mjs` describes a UI-guided step sequence, not the canonical Project state machine.
- **C / Host mapping:** GitHub Project/Issue observation via `packages/tally-github-c/src/project-intake.mjs`; physical freeze/admission experiment via `experiments/v2/project-intake/host-project.mjs`; Host controller `web/v2/intake-controller.mjs`. Any native write must pass the selected K/Host path and canonical readback.
- **K ownership:** freeze/admission CAS, idempotency, replay and effect receipts. Product frozen and Project admitted are different facts.
- **Completion evidence:** exact Product digest/source SHA, independent reviewer identity, freeze CAS receipt, accepted Claim/write-set where required, Project identity and canonical GitHub readback.
- **Wait / reject / recovery:** wait for missing Product evidence or independent Review; same author/reviewer, stale source, missing Claim or mismatched digest blocks; lost ACK becomes UNKNOWN and is reconciled by reading the existing Host/GitHub record, never blind duplicate creation.
- **Current evidence:** B/Host/C source and tests exist; selected T210 positive clean-consumer E2E was not found in the inspected Woodpecker profiles. `IMPLEMENTATION=PARTIAL`, `SELECTED_E2E=NOT_PROVEN`.

### WF02 — Architecture / L2

- **ID / owner:** `tally.workflow.architecture-l2/v1` / B1+B2.
- **Trigger:** Product baseline is frozen, a material architecture constraint changes, or a new Version requires an L2 decision.
- **Inputs:** ProductRevision, requirements and risks, architecture constraints, alternatives, ADR/design refs, prior L2 revision and exact source SHA.
- **Authority:** Product Owner scope; independent architecture/L2 Reviewer; GitHub design artifacts; ADS review semantics. A UI click or author self-review is not an L2 PASS.
- **Target states:** `DRAFT → OPTIONS_RECORDED → L2_REVIEW_PENDING → INDEPENDENTLY_REVIEWED → L2_FROZEN`; side outcomes: `REJECTED | STALE | BLOCKED | UNKNOWN`.
- **Business mapping:** no dedicated Architecture/L2 Business Intent was found among the current ten `INTENT_TYPES`. Do not reuse `freezeProductIntake` as a generic L2 freeze and do not infer that a Product Review implies L2 Review.
- **C / Host mapping:** GitHub read/PR/review evidence can be consumed through selected C; any durable L2 freeze requires a separately explicit authorized B→K→C contract. The current bounded scan did not identify a complete selected Architecture/L2 effect path.
- **K ownership:** if an approved L2 revision is made durable in the App, K owns the CAS/versioned activation receipt; GitHub remains authoritative for the underlying review/source artifact.
- **Completion evidence:** ADR/L2 revision digest, alternatives and consequences, independent exact-source review, owner decision and version pin.
- **Wait / reject / recovery:** missing reviewer, changed Product source, contradictory constraint or stale L2 approval blocks; preserve the prior L2 revision and mark the new proposal stale rather than overwriting.
- **Current evidence:** Product/L2 design documents and historical review artifacts exist, but a dedicated selected B/K Workflow route and positive clean-consumer proof are not established. `IMPLEMENTATION=DESIGN_ONLY`, `SELECTED_E2E=NOT_PROVEN`.

### WF03 — Version Planning / Task DAG

- **ID / owner:** `tally.workflow.version-planning-dag/v1` / B2 Architecture & Planning.
- **Trigger:** Product/L2 freeze, new Version planning, or a governed Replan request.
- **Inputs:** repository/version source SHA, Product/L2 pins, prior PlanRevision, native Task Issues, Issue body/write-set identities, native dependencies, WorkUnits, accepted/terminal work and supersession candidates.
- **Authority:** GitHub Issues and native dependencies; exact Task Pack blob/commit; Owner approval bound to that exact blob; ADS Task metadata; K CAS activation receipt.
- **Target states:** `PLAN_CANDIDATE → SOURCE_VALIDATION → NATIVE_EDGE_RECONCILIATION → WRITESET_REVIEW → OWNER_APPROVAL_PENDING → FREEZE_PENDING → FROZEN`. Side outcomes: `STALE | CYCLIC | BLOCKED | CONFLICT | UNKNOWN`.
- **Business mapping:** current `planNextVersion()` is advisory-only and emits candidates; `inspectTaskReadiness` is observational. Neither activates a PlanRevision. #273 defines the missing governed Replan contract; #260 owns the current canonical DAG/Claim lane.
- **C / Host mapping:** native GitHub Issue/dependency reads and writes must be exposed through the existing selected Controller→K→Host/GitHub C path. A text edge in an Issue body is not a native dependency.
- **K ownership:** append-only PlanRevision activation/freeze and replay-safe operation receipts. GitHub owns the actual native Issue/dependency facts.
- **Completion evidence:** complete roster, per-Issue current body/write-set digests, exact Task Pack blob, all dependency readbacks, acyclic graph, preserved accepted work, exact Owner approval, accepted Claim and K freeze receipt.
- **Wait / reject / recovery:** missing native edge, changed `updatedAt`, moved source, stale Owner approval, cycle or write-set conflict blocks. Never silently rebase or delete accepted work; lost ACK requires exact readback.
- **Current evidence:** pure candidate planning exists; canonical DAG writer is still in #260 closeout; governed Replan #273 is planned. `IMPLEMENTATION=ADVISORY_ONLY`, `GOVERNED_ACTIVATION=NOT_IMPLEMENTED`.

### WF04 — Task Execution / Attempt

- **ID / owner:** `tally.workflow.task-execution-attempt/v1` / B3 Execution.
- **Trigger:** a WorkUnit is source-current, dependencies are natively clear, and a lawful Claim request is eligible.
- **Inputs:** Task Issue/WorkUnit identity, exact source SHA, native dependencies, allowed write set, accepted Claim witness, actor/role, Host/environment target and unique Attempt ID.
- **Authority:** GitHub Task/dependency facts; ADS Claim/role protocol; K CAS accepted Claim and Attempt state; actual Host/Agent receipt.
- **Target states:** `PLANNED → READY → CLAIM_PENDING → CLAIM_ACCEPTED → ATTEMPT_RUNNING → CANDIDATE_READY`; terminal/uncertain Attempt outcomes: `FAILED | CANCELLED | INTERRUPTED | RECOVERING | UNKNOWN`. Task completion is a later Review/QA decision, not an Attempt state.
- **Business mapping:** current `admitClaimedTask` and `runLocalCodexAttempt` are the candidate Intent routes. `joinReviewedCandidates` belongs to WF05's candidate join, not the Attempt's own completion.
- **C / Host mapping:** native Claim event/registry read and the selected Local Agent capability; Host owns actual process/workspace execution and receipt. No browser or Agent self-report can mint a Claim.
- **K ownership:** Claim admission, one active writer CAS, Attempt event history, idempotency, replay and UNKNOWN.
- **Completion evidence:** exact Attempt ID, Claim generation, actor/role, source SHA, Host/runtime/environment, process exit and candidate tree; a zero exit alone does not prove Task PASS.
- **Wait / reject / recovery:** missing/ambiguous Claim, stale source, blocked dependency, duplicate active writer, Host loss or lost ACK must deny or remain UNKNOWN. Reconcile an existing Attempt before retrying an effectful launch.
- **Current evidence:** selected K safe-replay profiles exist for Local Codex and concurrent Join on DH v0.10.3; actual Windows/installed Host and complete Attempt read model remain separate gates. #275 owns the source-backed read-only Attempt model. `IMPLEMENTATION=PARTIAL`, `REAL_HOST_ACCEPTANCE=NOT_PROVEN`.

### WF05 — Independent Review / Repair

- **ID / owner:** `tally.workflow.independent-review-repair/v1` / B3+B4.
- **Trigger:** a candidate source/tree is published and ready for independent review, or an existing exact-source Review requests changes.
- **Inputs:** candidate SHA/tree, full diff, declared write set, protected paths, test/CI receipts, reviewer identity/role, prior Findings and repair lineage.
- **Authority:** GitHub PR/commit/review facts; ADS reviewer independence/role semantics; actual verifier/Host results; K controls any effectful Agent dispatch or publication.
- **Target states:** `CANDIDATE_READY → REVIEW_PENDING → REVIEWED_PASS | CHANGES_REQUESTED | REJECTED`; repair branch: `REPAIR_PENDING → REPAIR_RUNNING → RECHECK_PENDING → RE_REVIEW_PENDING → RE_REVIEWED`. Every new source SHA requires fresh applicable review/verification.
- **Business mapping:** `joinReviewedCandidates` joins source-qualified candidate results; `publishReviewedCandidate` publishes a candidate PR. Neither function alone is the independent Review decision or full Review/Repair state machine.
- **C / Host mapping:** selected GitHub C publishes/reads the candidate PR; Host verifier runs tests and collects real output; independent Reviewer uses a separate principal. Repair and re-review must remain separately attributable.
- **K ownership:** permitted Agent/Repair effects, idempotency and state receipts. GitHub remains authoritative for commits and PR reviews.
- **Completion evidence:** independent reviewer identity, exact SHA/tree, findings/dispositions, verifier command/exit, all changed paths accounted, repair diff and re-review on the repaired SHA.
- **Wait / reject / recovery:** self-review, stale Review, changed protected paths, missing verifier, incomplete diff, ambiguous candidate join or lost publish ACK cannot PASS. Reconcile publication by canonical PR readback.
- **Current evidence:** Join and candidate publication code/tests exist, plus historical concurrent Builder/Reviewer evidence; this does not prove the new installed UI/Host J01–J08. Full current selected Review→Repair→Re-Review E2E remains `NOT_PROVEN`.

### WF06 — QA / Release (Deployment remains a separate effect)

- **ID / owner:** `tally.workflow.qa-release/v1` / B4 Assurance & Release.
- **Trigger:** a source candidate has applicable independent Review and enters QA/release qualification.
- **Inputs:** exact source SHA/tree, required Node24 tests/build, Woodpecker run, Windows/Chromium Host acceptance, Full/Hidden/Critical vectors, independent RQ, Owner grant, target branch and tag/commit identity.
- **Authority:** actual CI/Host/Browser receipts; independent RQ; exact Owner authorization; GitHub merge/tag readback; K CAS reservation.
- **Target states:** `QA_PENDING → CI_VERIFIED → HOST_UI_VERIFIED → RQ_PENDING → OWNER_GRANT_PENDING → RELEASE_PREFLIGHT → MERGE_PENDING → MERGED_REQUALIFY_REQUIRED → POST_MERGE_RQ → TAG_PENDING → TAGGED`. Any missing/stale gate yields `BLOCKED | STALE | UNKNOWN`. **Deployment is a separate Host/Owner-authorized lifecycle**; a tag does not imply deployment or rollback success.
- **Business mapping:** `deliverQualifiedRelease` supports bounded `PREFLIGHT | MERGE | TAG` requests. `projectReleaseReadiness()` is advisory evidence projection and never signs an RQ or Owner grant.
- **C / Host mapping:** `packages/tally-github-c/src/release-delivery.mjs` plus `experiments/v2/release/host-release.mjs`; Host CAS reservation, exact GitHub readback and post-merge requalification are mandatory.
- **K ownership:** release-operation reservation, CAS, effect permission, replay/UNKNOWN and receipt. GitHub owns merge/tag facts; independent RQ and Owner own their own authority.
- **Completion evidence:** exact CI and Host receipts, independent RQ at the exact post-merge source, fresh Owner grant, merge readback, post-merge qualification and tag readback. Release PASS is not inferred from a successful merge.
- **Wait / reject / recovery:** stale head/tree, missing RQ, missing grant, Host CAS conflict, access denial, rate limit or lost ACK blocks/UNKNOWN. Never blindly retry merge/tag; reconcile the exact operation key and native readback.
- **Current evidence:** source/Host/C code and default-deny checks exist; the inspected Release profile did not prove a selected positive T222 clean consumer. `SELECTED_POSITIVE_E2E=NOT_PROVEN`.

### WF07 — Maintenance / Replan

- **ID / owner:** `tally.workflow.maintenance-replan/v1` / B1+B2+B5.
- **Trigger:** a defect/maintenance report against an immutable prior release, changed requirement, failed verification, or newly discovered source/dependency condition.
- **Inputs:** prior Release tag/commit, defect Issue, affected Requirement/source SHA, reproduction and evidence refs, current PlanRevision, accepted/terminal WorkUnits and unfinished work.
- **Authority:** GitHub Release/Issue/CI facts; Product Owner scope; ADS evidence semantics; K CAS for maintenance publication and Replan activation.
- **Target states (two related but distinct sub-processes):** maintenance `OBSERVED → DEFECT_TRIAGED → MAINTENANCE_CANDIDATE → CANDIDATE_REVIEWED → ISSUE_PUBLICATION_PENDING → ISSUE_PUBLISHED`; Replan `REPLAN_CANDIDATE → PRESERVATION_CHECK → OWNER_APPROVAL_PENDING → ACTIVATION_PENDING → PLAN_REVISION_ACTIVATED`. Historical Release remains immutable.
- **Business mapping:** `projectMaintenanceIntake()` and `planNextVersion()` are advisory projections; `publishMaintenanceIssue` and `recordTaskLearningOutcome` are current maintenance/learning effects. #273 adds the missing governed Replan candidate/activation contract; it is not implemented by calling `planNextVersion()`.
- **C / Host mapping:** maintenance Issue publication via `maintenance-effects.mjs` and `host-maintenance.mjs`; Replan native dependency/Task Pack effects must use the existing selected K→Host/GitHub C path.
- **K ownership:** immutable maintenance candidate/decision receipts, idempotent publication, and append-only PlanRevision activation. Do not create a second DAG scheduler or history store.
- **Completion evidence:** exact defect/prior release refs, candidate digest and independent review, native maintenance Issue readback; for Replan, exact new PlanRevision plus proof accepted/terminal work and prior evidence were preserved.
- **Wait / reject / recovery:** stale prior release, unverified defect, duplicate candidate, lost publication ACK, changed dependencies or unapproved Replan remains UNKNOWN/BLOCKED; reconcile before retry.
- **Current evidence:** maintenance projection/publication and learning record code exists; governed Replan activation #273 remains planned. `MAINTENANCE=PARTIAL`, `REPLAN_ACTIVATION=NOT_IMPLEMENTED`.

### WF08 — Experience Qualification / Promotion

- **ID / owner:** `tally.workflow.experience-qualification/v1` / B5 Improvement & Experience.
- **Trigger:** a Task/Attempt/Review/Verification outcome is available, or a later Task reports use/reject of a previously qualified ExperienceRef.
- **Inputs:** ADS-published GitHub report, original Task/Attempt/Review/CI/Host refs, exact source SHA, applicability, exclusions, counterexamples, validation fixtures, human decision and later independently attested outcome where available.
- **Authority:** original GitHub/CI/Host facts and ADS v5 report/profile; Tally B owns curation and qualification policy; K owns durable ADOPT/REJECT/feedback effect; Owner governs any Package promotion/build selection.
- **Target states:** `OBSERVED → EXTRACTED → CANDIDATE → VALIDATING → SHADOW_VERIFIED | REJECTED → GOVERNED_REVIEW → PROMOTED → BUILD_QUALIFIED → SELECTED_IN_NEW_BUILD → USED | REJECTED_BY_LATER_TASK → OUTCOME_PENDING → OUTCOME_ATTESTED`. A running occurrence remains pinned to its original build.
- **Business mapping:** current `recordTaskLearningOutcome` records a use/reject decision and optional later attested outcome; `casebook.mjs` and `feedback.mjs` are projections. #274 owns the missing ADS v5 EXPERIENCE_USE_FEEDBACK publisher; #262 consumes the Experience context; #60 defines the broader candidate lifecycle. No current Intent proves full DomainAsset promotion and BUILD activation.
- **C / Host mapping:** ADS feedback Publisher is a distinct effectful C operation with K Permit, idempotency, lost-ACK reconciliation and exact GitHub readback. Read-only ADS report consumption remains separate from effectful publishing.
- **K ownership:** durable use/reject decision and authorized publication/adoption receipts. Package activation is a new exact BUILD/Host pin, not a mutation of the active instance.
- **Completion evidence:** source-qualified candidate, positive/negative/counterexample validation, independent qualification, Owner-governed promotion, new BUILD pin, actual next-Task Context Pack use/reject receipt and later outcome evidence. No statistically measured gain without independent measured outcomes.
- **Wait / reject / recovery:** missing telemetry remains UNKNOWN; contradictory evidence is retained; stale/revoked asset is excluded; lost feedback ACK is reconciled; unqualified candidate cannot be selected; no Agent ranking, model-weight training or automatic promotion.
- **Confirmed P0 gap:** `experiments/v2/maintenance/host-maintenance.mjs` currently requires the suggestion's qualification to equal `REPORTED_QUALIFIED` before recording either `ADOPTED` or `REJECTED` (see `hostRecordTaskLearningOutcome`). This blocks the required ability to record a source-qualified human REJECT for stale/unqualified evidence while keeping ADOPT denied. #219 comment 6594480790 explicitly requires this distinction; #262 LOCAL_VALIDATION R4 also reports `#219_HOST_REJECT_CONTRACT`. Repair and positive REJECT + negative ADOPT tests belong inside #219's existing Host/test write set, with a fresh valid Claim/ownership check before source mutation.
- **Minimum #219 repair contract (proposal, not applied):** first require a source-identifiable ledger suggestion with a valid ADS publication provenance ref. For `ADOPTED`, require `qualification === 'REPORTED_QUALIFIED'` plus the existing currentness/verifier/Owner gates. For `REJECTED`, permit a source-identifiable `UNKNOWN`/stale/unqualified suggestion and persist the exact `sourceStatus`, `qualification`, verifier/currentness evidence and provenance in the K/Host CAS record; do not make it eligible for a next-Task Context Pack. Suggestions with no valid source identity/provenance remain unrecordable. Tests: positive REJECT on UNKNOWN, positive REJECT on STALE, negative ADOPT on both, negative REJECT with missing/malformed provenance, and replay/idempotency for each.
- **Current evidence:** candidate/read projections and local learning decisions exist; ADS v5 Publisher #274 and complete promotion/BUILD-selection proof remain outstanding. `UNQUALIFIED_REJECT=BLOCKED_BY_HOST_GUARD`, `UNQUALIFIED_ADOPT=DENY_EXPECTED`, `FEEDBACK_PUBLICATION=NOT_PROVEN`, `PROMOTION_BUILD=NOT_PROVEN`.

## 5. Current Business Intent / C / Host crosswalk

The ten Intent declarations in `src/index.mjs` are not ten complete ALM Workflows. They are operation/effect entry points. The bounded mapping below prevents an implementation from claiming a complete Workflow from one Intent or unit test.

| Existing Business Intent | Domain Workflow | C / Host path (current source) | What it does not prove |
|---|---|---|---|
| `inspectTaskReadiness` | WF03 / WF04 observation | Native read / selected K readiness observation | Claim acceptance, Task PASS or DAG freeze |
| `admitClaimedTask` | WF04 | Native Claim observation + K admission | Actual Agent process completion |
| `runLocalCodexAttempt` | WF04 | Selected Local Codex capability + Host process | Independent Review or Task PASS |
| `joinReviewedCandidates` | WF05 | Candidate/Attempt join | Independent Reviewer identity or re-review |
| `publishReviewedCandidate` | WF05 | GitHub candidate PR publication/readback | Review PASS or merge permission |
| `freezeProductIntake` | WF01 | Host CAS Product freeze | Project admission or L2 freeze |
| `admitProjectIntake` | WF01 | Project creation/admission and GitHub readback | Product/L2 Freeze or version DAG freeze |
| `publishMaintenanceIssue` | WF07 | Native maintenance Issue publication/readback | Governed Replan activation |
| `recordTaskLearningOutcome` | WF07 / WF08 | Host CAS ADOPT/REJECT + later outcome record | ADS v5 feedback publication or DomainAsset promotion |
| `deliverQualifiedRelease` | WF06 | Release PREFLIGHT/MERGE/TAG | Deployment, post-merge RQ completion or automatic Release PASS |

No current Intent in this list is a dedicated Architecture/L2 freeze, governed Replan activation or Experience DomainAsset promotion contract. Add one only after the exact existing ABI and downstream source gap are reviewed; do not add a generic intent merely to mirror a workflow label.

## 6. Existing B module disposition — do not refactor by line count

| Current module | Current source responsibility | Proposed ownership / action | Evidence boundary |
|---|---|---|---|
| `src/index.mjs` | Public Business ABI, policy, 10 Intent/Effect declarations, assess/project handlers | Preserve root ABI; separate private domain logic only when the selected producer explicitly bundles it | V10 producer currently selects one B source file; zero imports in index |
| `src/acceptance.mjs` | Acceptance item and coverage projection | B4 acceptance read model; map its inputs to Verification/Review/Release refs | Unit test exists; selected runtime callsite not found in bounded scan |
| `src/casebook.mjs` | Case replay and affected-case projection | B5 knowledge/counterexample read model | Unit test exists; selected runtime callsite not found in bounded scan |
| `src/diagnosis.mjs` | Workfront diagnosis | B2/B5 deterministic diagnosis projection | Unit test exists; selected runtime callsite not found in bounded scan |
| `src/feedback.mjs` | Feedback-case stage and duplicate projection | B5 experience/feedback read model | Unit test exists; selected runtime callsite not found in bounded scan |
| `src/guided-intake.mjs` | Guided step schema/validation/next-step | Keep out of sealed B as-is until Node/AJV/fs/crypto dependency boundary is explicit; likely Host Journey adapter or build-time validation | Direct Intake Host import; not in selected B module bytes |
| `src/journey.mjs` | Project/Version/Task Journey snapshot and next actions | UI Journey/read model; never owns K durable state | Unit test exists; selected runtime callsite not found in bounded scan |
| `src/lineage.mjs` | Version/evidence lineage reconstruction | B4/B5 lineage read model over GitHub refs | Unit test exists; selected runtime callsite not found in bounded scan |
| `src/maintenance-intake.mjs` | Prior Release/Defect/Learning candidate projection | B5 pure domain projection; expose through an explicit version-pinned internal seam before Host consumes it | Direct Maintenance Host controller import |
| `src/planning.mjs` | Advisory Product/L2/Task DAG candidate planning | B2 pure candidate calculation; no native dependency or Claim authority | Direct Intake Host controller import |
| `src/release-readiness.mjs` | Reported gate/readiness projection | B4 read model; reported PASS remains unverified until independent source attestation | Unit test exists; selected runtime callsite not found in bounded scan |
| `src/work-units.mjs` | IssueContainer/WorkUnit identity, Claim history and collision projection | B3 derived read model; no Claim/CAS authority | Unit test exists; selected runtime callsite not found in bounded scan |
| `src/attempts.mjs` | Not present at baseline | #275 owns initial read-only Attempt model; F may touch only after #275 source ownership completes | Explicit dependency; do not create a competing Attempt writer |

**Critical closure finding:** the V10 Producer consumes one `TALLY_BUSINESS_SOURCE` file as selected B `moduleSource`; current `index.mjs` has zero imports. Host controllers directly deep-import some sibling helpers. The B package is private and exports only its root. This is a monorepo-internal coupling and source-pin/version-alignment obligation, not evidence of direct Provider/Permit bypass. Do not split modules until the Producer/BUILD closure is deliberately updated and tested. Do not delete “no callsite found” modules without a repository-wide usage/ownership audit.

## 7. Validation matrix and acceptance gates

Every Workflow and promoted Rule must be tested through the exact layers it claims to use:

1. **Definition:** schema parses; IDs and versions are stable; owner, source authority, applicability and exclusions are explicit.
2. **Positive:** one eligible real or isolated clean-consumer path executes the selected B Intent → one K → selected C → trusted Host and returns the authoritative receipt.
3. **Negative:** missing/stale source, invalid role, absent Claim, wrong repository, changed write set, unauthorized effect and unselected route all refuse; effectful Provider dispatch count is zero on denial.
4. **Boundary:** duplicate IDs, cross-repository input, empty/oversized evidence, cyclic DAG, same author/reviewer, protected-path edits and conflicting dependencies remain blocked/UNKNOWN.
5. **Recovery:** lost ACK, restart, timeout, retry, replay and concurrent writer conflict reconcile by exact operation key and canonical readback; no duplicate external action.
6. **Currentness:** bind the evidence to exact source SHA/tree, Task Pack blob, selected Package/ABI, DH source tag, V10 BUILD profile, Host source and installed dependency digest.
7. **UI:** dynamic form is a projection of an issued/validated Workflow step; stale step/source and CSRF fail closed; UI cannot mint Claim, alter K state or write GitHub directly.
8. **Experience:** positive/negative/counterexample cases, independent qualification, promotion approval, new BUILD pin, later Task consumption and later outcome are separately recorded.

### 7.1 Exact-source qualification matrix

| Evidence profile | Current pin / scope | Allowed conclusion |
|---|---|---|
| T02B default clean consumer | DH v0.10.2 | Historical selected inspect route only on that pin |
| T06L Claim admission | DH v0.10.2 | Selected Claim safe-replay evidence on v0.10.2; not v0.10.3 |
| T06M Local Codex | DH v0.10.3 SHA `061376f4a023070f215f9feffad5c24d96d503cb` | Selected Local Codex safe replay on this pin |
| T06N concurrent Join | DH v0.10.3 SHA `061376f4a023070f215f9feffad5c24d96d503cb` | Selected Join safe replay on this pin; real Windows overlap remains separate |
| T06P candidate publication | No selected positive consumer found in inspected profile | B/C/Host source tests only; selected K E2E NOT_PROVEN |
| T210 Product/Project Intake | No selected positive consumer found in inspected profile | B/C/Host source tests only; selected K E2E NOT_PROVEN |
| T219 Maintenance/Learning | No selected positive consumer found in inspected profile | B/C/Host source tests only; selected K E2E NOT_PROVEN |
| T222 Release | Static default-deny profile only | Refusal when unselected; positive selected Release E2E NOT_PROVEN |

Before any Woodpecker CI:
`node apps/tally-web/scripts/syntax-preflight.mjs` → `git diff --check` → scoped Node24 tests/typecheck/build → Woodpecker. The current docs-only audit does not run CI.

## 8. Source write-set overlap and BUILD-closure decision

The current Task bodies show that F is a **late serialized source join**, not a parallel writer for every existing B file.

| Task | Exact overlap with #276 candidate write set | Required ordering / boundary |
|---|---|---|
| #219 Maintenance/Learning | `src/index.mjs`, `test/business-policy.test.mjs`, and (under the proposed selected-BUILD amendment) `experiments/v2/selected-business-build/producer.mjs`; #219 also owns maintenance C/Host/controller paths | #219 must finish its B policy and Producer edits first; F reconciles the final source rather than racing it |
| #273 Governed Replan | `src/index.mjs`, `test/business-policy.test.mjs`, `src/planning.mjs` | #273 is after #219; F must consume its final Replan contract/source and not rewrite it concurrently |
| #275 Attempt read model | `src/attempts.mjs`, `test/attempts.test.mjs` | #275 owns initial creation; F may reconcile only after #275 completes and the Attempt source contract is frozen |
| #262 Experience | No current exact B path in its admitted write set; it reserves Experience/feedback integration semantics and allows existing learning/Casebook B files only by separate amendment | F must align with the delivered ExperienceRef/Qualification contract; no opportunistic Casebook/feedback edit |
| #261 Action bridge | UI/Host action paths, not F's current B write set | Preserve B Intent and K/Host action contract; do not edit #261-owned paths |
| #274 ADS feedback Publisher | C/native-C/Host Publisher paths, not F's current B write set | Align ExperienceUseFeedback vocabulary and read/effect split; do not edit C paths |
| #264 Final integration | Shared Host/UI/entrypoint/Woodpecker hotspots | E remains the sole shared integration writer and waits for F |

### 8.1 Required decision: how Domain Data enters the selected BUILD

At the pinned source baseline:
- V10 Producer reads the single `TALLY_BUSINESS_SOURCE` file and passes those bytes as the selected B `moduleSource`.
- `src/index.mjs` has zero imports.
- B `package.json` exposes only its root entrypoint and the package is private.
- `domain/**` assets and sibling source modules are therefore **not proven to enter the sealed selected B** merely because they exist in the repository.

Before freezing the final #276 write set, choose and evidence one of these explicit outcomes:

1. **Authoring-time/reference-only Domain Data for v2:** assets are versioned and validated by tests/Producer-side tooling but are not claimed as selected Runtime inputs. Existing B root remains the selected source. State clearly which semantics remain compiled/duplicated in `index.mjs`.
2. **Selected Runtime Domain Data for v2:** the BUILD must deliberately include/inline/compile the required assets into the selected B module and bind the result to a digest. This is the recommended outcome for core executable schemas/rules/workflow definitions; non-executable Knowledge/Patterns/Recipes/Examples/Skills may remain authoring-only until a real consumer exists.

Issue #276 now records a proposed but not admitted write-set amendment: edit `experiments/v2/selected-business-build/producer.mjs` serially after #219; add `domain-data-compiler.mjs`, `domain-data-compiler.test.mjs`, `consumer-domain-data.mjs` and `consumer-domain-data.test.mjs`. F supplies the deterministic consumer/test entrypoint; E/#264 owns the shared Woodpecker profile. These paths require independent review, exact Task Pack reconciliation and Owner approval before any source Claim.

A Host controller reading `domain/**` directly is not an equivalent solution for B-owned business semantics: it reintroduces the existing Host deep-import/source-drift risk and does not prove selected B/K execution. Do not silently add `fs` reads or a dynamic file loader to the Business ABI.

### 8.2 Recommended staged execution (inside the admitted F task)

- **F0 — Design and inventory:** completed as docs-only analysis in PR #277; not source admission.
- **F1 — Disjoint Domain Data assets:** after a fresh #276 Task Pack/Claim, add only the minimal glossary/authority/schema/rule/workflow assets and deterministic tests under F's new directories.
- **F2 — B source reconciliation:** after #219/#273/#275 and the relevant #262 Experience contract are complete; one writer edits the overlapping B modules/tests.
- **F3 — BUILD qualification:** only if the selected-runtime outcome above is chosen and the write set is separately amended; run exact clean-consumer positive/negative tests for every selected asset/route.
- **F4 — Integration:** E/#264 alone composes shared Host/UI/Woodpecker paths after independent F review.

This ordering avoids overlapping writers and prevents a documentation-only Domain Data tree from being mistaken for selected executable Business logic.

## 8. Implementation sequence for Lane F

1. Complete the source-to-callsite/BUILD closure inventory and preserve the module disposition table above.
2. Establish the minimal glossary and authority map; do not create all directories/assets in bulk.
3. Add only the schemas and deterministic rules required by a real Workflow consumer. Each schema/rule must have a positive, negative, boundary and recovery fixture.
4. Define the eight Workflow assets and map each state/transition to an existing or explicitly missing Business Intent/C Operation/K effect/Host port.
5. Resolve overlapping writers #219, #261, #262, #273, #274 and #275 before changing their owned source files. #276 stays PLANNED until exact Task Pack, native dependencies, accepted CAS Claim and JIT branch are verified.
6. Reconcile the selected B source closure: no deep import into private package internals without an explicit contract; no assumption that sibling files are bundled; no generic SDK change without a reproducible clean-consumer gap.
7. Add tests in the exact F write set; syntax preflight first; then scoped Node24 tests, typecheck/build and the approved Woodpecker profile.
8. Independent exact-SHA review; only then may #260 refresh/freeze the final DAG and #264 perform the sole shared integration.

## 9. Adversarial acceptance checklist

- [ ] No eight independent K instances or second Workflow Engine.
- [ ] No conflation of Domain Process, K Execution, UI Journey and Agent Skill Flow.
- [ ] No inference that 10 Intent declarations mean 10 selected routes or eight implemented Workflows.
- [ ] No arbitrary YAML auto-loading or implicit sibling-module BUILD inclusion.
- [ ] No direct Provider/Host side effect from B or browser.
- [ ] No GitHub/ADS/K/Host/task-meta fact mirrored as a competing authority.
- [ ] No Issue automatically treated as executable Task/WorkUnit.
- [ ] No accepted Claim inferred from a self-reported comment.
- [ ] No Task PASS from an Attempt receipt, PR merge, static test or reported CI status alone.
- [ ] No accepted/terminal work, historical Review or Release evidence silently rewritten during Replan.
- [ ] No Experience Candidate automatically promoted or hot-swapped into a running build.
- [ ] No quality gain, cost, duration or Agent ranking fabricated from missing telemetry.
- [ ] No version claim that confuses Tally v2, Business ABI v08, DH v0.10.3, V10 BUILD and installed Host.

## 10. Current disposition

```ini
DOMAIN_DATA_INVENTORY=STATIC_DESIGN_COMPLETE
CANONICAL_GLOSSARY=PROPOSED
EIGHT_WORKFLOW_CONTRACTS=DESIGN_COMPLETE_NOT_IMPLEMENTED
SOURCE_CALLSITE_SCAN=BOUNDED
SELECTED_BUILD_CLOSURE=PARTIAL
INTENT_ROUTE_E2E=4_ROUTES_ACROSS_MIXED_DH_PINS
DH_V0103_SELECTED_SAFE_REPLAY=T06M_T06N_ONLY
PRODUCT_INTAKE_SELECTED_E2E=NOT_PROVEN
ARCHITECTURE_L2_SELECTED_ROUTE=NOT_FOUND
GOVERNED_REPLAN=NOT_IMPLEMENTED
REVIEW_REPAIR_FULL_E2E=NOT_PROVEN
RELEASE_POSITIVE_SELECTED_E2E=NOT_PROVEN
EXPERIENCE_FEEDBACK_PUBLISHER=NOT_PROVEN
DOMAIN_ASSET_PROMOTION_BUILD=NOT_PROVEN
NODE24_TEST=NOT_RUN
WOODPECKER=NOT_RUN
INDEPENDENT_REVIEW=NOT_RUN
TASK_276=PLANNED
SOURCE_CLAIM=NOT_AUTHORIZED
VERSION_V2_MUTATION=NONE
MAIN_MERGE=NONE
RELEASE=NONE
```
