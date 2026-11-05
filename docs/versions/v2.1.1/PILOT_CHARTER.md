# Tally v2.1.1 — Software Engineering Domain Knowledge Pilot Charter

**Status:** DRAFT / EXPERIMENTAL / NOT_FROZEN  
**Prepared:** 2026-11-04  
**Parent index:** [Tally #269](https://github.com/kaicreator-mm/tally/issues/269)  
**Workstream:** [Tally #299](https://github.com/kaicreator-mm/tally/issues/299)  
**Task issues:** [SEK-00 #300](https://github.com/kaicreator-mm/tally/issues/300), [SEK-01 #301](https://github.com/kaicreator-mm/tally/issues/301), [SEK-02 #302](https://github.com/kaicreator-mm/tally/issues/302), [SEK-03 #304](https://github.com/kaicreator-mm/tally/issues/304), [SEK-04 #305](https://github.com/kaicreator-mm/tally/issues/305), [SEK-05 #308](https://github.com/kaicreator-mm/tally/issues/308)

## 1. Purpose

Produce a source-grounded, reviewable, reproducible Software Engineering Domain Knowledge Corpus and Knowledge Graph. Use the real knowledge-production work to test domain discovery, source authority, provenance, knowledge modeling, authoring formats, LinkML, graph export and human review.

This is an experiment, not a predetermined architecture adoption. A high-quality negative result is acceptable if the evidence shows that a proposed model, tool or method does not fit.

## 2. Ownership and boundaries

Both experimental asset families remain in Tally v2.1.1:

1. **Domain knowledge instances:** taxonomy, sources, source locators, evidence, concepts, atomic claims, principles, practices, patterns, anti-patterns, cases, counterexamples, typed relationships, review decisions and graph projections.
2. **Knowledge-production method evidence:** discovery procedures, source-verification rules, extraction/modeling methods, authoring experiments, graph validation, human-review practices, failures, corrections, limitations and retrospective.

This is an experimental placement decision, not a claim that Tally should become a universal knowledge-generation product.

DomainForge #204/#205 are reference and hypothesis links only. DomainForge does not own this implementation, does not automatically receive experimental records, and does not adopt LinkML or any resulting schema by implication. DomainForge #206 is historical; the detailed work is here. Any later promotion requires a separate evidence-backed decision after independent review.

## 3. Intended domain and consumer lens

**Domain:** Software Engineering as an engineering discipline: the systematic, disciplined and evidence-informed work of specifying, designing, constructing, testing, operating, maintaining and evolving software systems and the professional, organizational and economic practices required to do so.

**First consumer lens:** Tally Engineering Harness and its software-engineering decisions. This lens may help select useful cases and questions, but it must not turn the corpus into Tally implementation notes. Knowledge statements should remain broadly reusable and carry their applicability/context.

**Adjacent domains:** Computer Science, Systems Engineering, Cybersecurity, Data Engineering, AI/ML Engineering, HCI, platform/cloud engineering, embedded engineering and safety engineering are mapped as foundations, intersections, specialized contexts or explicit boundaries. They are not silently absorbed in full.

## 4. Completeness contract

“Complete” means **coverage-complete relative to a version-pinned, independently reviewed v2.1.1 taxonomy and acceptance/coverage profile**.

It does not mean all software engineering knowledge ever published, every vendor/framework/tool, every adjacent discipline in full, every possible case or empirical result, or unsupported 100% coverage claims.

Every included area must have a defined scope, expected knowledge types, source coverage, review state, graph coverage and explicit known gaps. A gap is a valid recorded result, not a reason to fabricate completeness.

## 5. Knowledge-production path

~~~text
Domain Discovery
  → Authority / Source Discovery
  → Source Identity and Version Verification
  → Content and Locator Verification
  → Evidence Extraction
  → Atomic Knowledge Claims
  → Knowledge Modeling
  → Typed Relationship Assertions
  → Cases / Counterexamples
  → Human Review
  → Graph Validation and Export
  → Coverage Review
  → Independent Qualification
  → Method Retrospective
~~~

Keep these states distinct:

~~~text
SOURCE_IDENTITY_VERIFIED
  != CONTENT_VERIFIED
  != LOCATOR_VERIFIED
  != EXTRACTED_CLAIM
  != REVIEWED_SYNTHESIS
  != ACCEPTED_KNOWLEDGE
~~~

Claims and material relationships require source/evidence provenance, applicable context, assumptions, exceptions, limitations, uncertainty and review status. Contradictory sources and counterexamples remain visible; do not average away disagreement.

## 6. Candidate knowledge entities

The initial modeling candidates are Domain and Subdomain; Source and SourceLocator; Evidence; Concept and atomic Claim; Principle and Practice; Pattern and AntiPattern; DecisionModel; Engineering Case and Counterexample; RelationshipAssertion; ReviewDecision and CoverageGap.

Candidate relationship types include SUPPORTED_BY, CONTRADICTED_BY, QUALIFIED_BY, APPLIES_TO, DEPENDS_ON, DERIVED_FROM, IMPLEMENTS, ILLUSTRATED_BY, VALIDATED_BY, SUPERSEDES and RELATED_TO. These are hypotheses, not a frozen ontology.

## 7. LinkML and authoring-format experiment

LinkML is **EXPERIMENTAL / NOT_ADOPTED**. The experiment must use authentic Software Engineering records and compare:

A. Markdown records with YAML front matter.  
B. Structured YAML instances paired with linked Markdown explanations.  
C. Standalone YAML records.

Evaluate schema expressiveness, IDs/references, evidence/provenance, cardinality, enum constraints, validation errors, JSON-LD/RDF projection, deterministic rebuild, schema evolution, Git Diff readability, human review effort and migration cost. Test a credible alternative if LinkML has material shortcomings.

The final experiment decision must be exactly one of ADOPT, ADOPT_WITH_AMENDMENTS, REPEAT_EXPERIMENT or REJECT. No outcome automatically changes DomainForge.

## 8. Deliverables

1. Software Engineering Domain Taxonomy and coverage profile.
2. Authority Source Registry and source-record template.
3. Source-grounded Knowledge Corpus.
4. Evidence and Provenance Records.
5. Typed Knowledge Relationships.
6. Engineering Cases and Counterexamples.
7. Reproducible JSON-LD/RDF Knowledge Graph.
8. Human-readable Knowledge Handbook.
9. Graph Validation and Coverage Reports.
10. LinkML Proof-of-fit Report.
11. Knowledge Production Method Retrospective.

All canonical source records, schemas, fixtures, evidence and generated artifacts are versioned in Git. Graph databases, vector stores and other query infrastructure are optional derived consumers, not canonical sources of truth.

## 9. Task DAG and safe parallelism

| Task | Work | Dependencies | Output |
|---|---|---|---|
| SEK-00 | Pilot Charter and acceptance profile | — | This charter and scope contract |
| SEK-01 | Domain Discovery and multi-view taxonomy | SEK-00 | Domain discovery report and proposed taxonomy |
| SEK-02 | Authority Research Framework | SEK-00 | Source verification framework and registry |
| SEK-03 #304 | Verified representative source research and initial evidence slice | SEK-01, SEK-02 | Source dossiers, locators, evidence and atomic claims |
| SEK-04 #305 | Early architecture review and LinkML Proof-of-fit | SEK-01, SEK-02, sample SEK-03 | Schema/format experiment, reproducible outputs and evidence-backed decision |
| SEK-05 #308 | Knowledge model, relationship catalog and authoring-format decision | SEK-04 #305 plus independent review of PR #303 | Selected experimental schema/format and validation profile |
| SEK-06 | Vertical-slice production | SEK-05 | Reviewed records in 2–3 contrasting areas |
| SEK-07 | Full scoped production | SEK-05, SEK-06 | Expanded knowledge records |
| SEK-08 | Graph integration and handbook | SEK-06, SEK-07 | Integrated graph and documentation |
| SEK-09 | Independent review and qualification | SEK-08 | Qualification report and gap register |
| SEK-10 | Method retrospective | SEK-09 | Findings, failures and promotion candidates |

SEK-01 and SEK-02 have disjoint write sets and may proceed concurrently after SEK-00 is accepted. SEK-03 requires both outputs. The later subdomain lanes may run in parallel only after taxonomy, schema and stable ID rules are sufficiently pinned. Graph generation/integration has one designated Writer.

The planning table is not the live execution DAG. Native GitHub Issue Dependencies remain the canonical live dependency mechanism; task text references do not substitute for those edges.

## 10. Quality gates

- **G0 Scope:** domain boundary, taxonomy, views, exclusions and scoped-completeness profile are explicit and reviewed.
- **G1 Sources:** source identity, edition/status, permitted access and exact locators are verified; unsupported citations are rejected.
- **G2 Knowledge:** atomic claims are grounded; assumptions, applicability, contrary evidence, exceptions and limitations are explicit.
- **G3 Graph:** unique IDs, typed relationships, valid references, provenance, deterministic exports and reproducible validation.
- **G4 Human review:** reviewers can trace knowledge → claim → evidence → locator → source; disputed items are not silently promoted.
- **G5 Utility:** representative questions compare knowledge-assisted answers with direct model responses; correctness, relevance, traceability, omissions, hallucinations and review effort are measured or marked UNKNOWN.

## 11. Storage and rights

Canonical experimental assets belong under docs/versions/v2.1.1/. Do not reuse the existing root knowledge/ store: it is a historical, narrowly scoped, verified-repair knowledge base with a different admission contract.

Source licensing and access restrictions are first-class metadata. Do not copy substantial copyrighted text. ISO publication pages state restrictions on using ISO content with AI/ML tools; this experiment records bibliographic metadata only for ISO sources unless a permitted human-access workflow is established. Do not prompt normative ISO text into an LLM or store generated clause summaries.

Do not store credentials, private telemetry, hidden evaluator material or private model chain-of-thought. Method evidence records observable inputs, outputs, decisions, revisions, errors and corrections only.

## 12. Current baseline and status

- Repository: kaicreator-mm/tally
- Observed default-branch baseline: main@e988321439f65b55936a691c02250b6d62a29409
- WIP integration branch: wip/v2.1.1-knowledge, created from that exact SHA.
- No official version/v2.1.1 branch or release is created by this charter.
- Repository .dev-standard/VERSION records ADS 4.18.0. The effective Tally-local ADS v4.18 adoption is documented for version/v2 through PR #85; this is not silently generalized to a new v2.1.1 version branch.
- Product PR #157 R9 exact source review (HEAD 93fe091aca5e8e367235f831bd416907bdf32d1d) is PASS with P0/P1/P2=0/0/0 and PRODUCT_FREEZE_READY=YES, but PRODUCT_FROZEN=NO because the final Controller checkpoint has not been published/read back.
- Semantic L2 PR #158 R7 exact source review (HEAD 1cd254694de9c6b36a1c66a193c10fe29901525f) is PASS with P0/P1/P2=0/0/0, but L2_FREEZE_READY=NO and L2_OWNER_FREEZE=NOT_PUBLISHED. This experiment does not modify Product/L2 semantics or claim their Freeze.
- This v2.1.1 work is documentation/research only: no runtime source, Host, Agent admission, CI workflow, or executable behavior is changed.
- ADS #1359 is a Delivery-First Product/L2 intake and remains NOT_FROZEN. The user's explicit Delivery-First direction is applied as an execution preference here; it is not represented as an adopted ADS amendment.
- SEK-00 #300, SEK-01 #301, SEK-02 #302, SEK-03 #304 and SEK-04 #305 exist as bounded Task Issues; SEK-05 #308 is preplanned and NOT_DISPATCHABLE until the SEK-04 repository checker has passed exact-HEAD local validation, its terminal independent review has passed, the builder/repair PRs are integrated into WIP PR #303, and the resulting #303 HEAD receives independent Exact-SHA review. Native dependency edges are now MATERIALIZED and exact-readback verified; the graph is live but not frozen.

**Current decision state:** taxonomy, authority framework, LinkML, authoring format and ontology remain PROPOSED / EXPERIMENTAL. This charter itself is DRAFT pending review.


## 13. Shared repository persistence — Controller-only exception

`docs/research/v2/OUTPUT_LEDGER.md` is a repository-wide shared persistence record, not a task-owned artifact. It is explicitly outside the individual SEK Task Pack write sets and is governed by a separate **Controller-only, single-writer exception**:

- Only the Delivery Controller may update the ledger after reading back the relevant GitHub task/PR outputs.
- Task Builders, Reviewers and Repairers must not modify the ledger.
- A Controller ledger update may be included in an integration PR, but the PR must identify it as the separate Controller persistence write set.
- This exception grants no additional write authority to any task and does not weaken task-level exact write sets.
