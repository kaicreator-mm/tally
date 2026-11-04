# SEK-02 — Authority Research and Provenance Framework

**Status:** DRAFT / EXPERIMENTAL / NOT_FROZEN  
**Task:** [#302](https://github.com/kaicreator-mm/tally/issues/302)  
**Parent:** [#299](https://github.com/kaicreator-mm/tally/issues/299)  
**Research date:** 2026-11-04

## 1. Purpose

Define how the corpus discovers, verifies, classifies and uses sources. A source's reputation or publication type is not a universal quality score. **Authority is claim-specific:** a standard may be authoritative for a normative requirement in its scope, a peer-reviewed study for a measured result in its population, an official vendor manual for current product behavior, and a postmortem for a particular incident history.

A source record is not itself a knowledge claim. A verified source identity does not prove that a claim was extracted correctly or that a synthesis is accepted.

## 2. Source classes

| Class | Typical source | Best suited to support | Limitations |
|---|---|---|---|
| NORMATIVE_STANDARD | ISO/IEC/IEEE, IEEE, IETF, W3C, national standards | Normative requirements, defined terms and conformance scope | Applicability, edition, jurisdiction/contract and licensing matter; not proof of universal effectiveness |
| PROFESSIONAL_BOK | SWEBOK and professional consensus guides | Discipline taxonomy, generally accepted knowledge and topic maps | Guide is not the complete literature or an exhaustive ontology |
| CURRICULUM_FRAMEWORK | ACM/IEEE-CS/AAAI curricula | Educational scope, competency and curriculum boundaries | Curriculum coverage is not professional-domain completeness |
| PEER_REVIEWED_EMPIRICAL | Journal/conference paper, systematic review, replication | Empirical findings, measured associations and tested interventions | Population, method, bias, confidence and replication limit generalization |
| TEXTBOOK_MONOGRAPH | Recognized academic/professional books | Explanations, established theories, methods and worked examples | Edition, author expertise, citation trail and publication date matter |
| OFFICIAL_TECHNICAL_DOCUMENTATION | Maintainer/vendor/platform documentation | Product-specific APIs, behavior, supported versions and migration guidance | Product-specific; may be promotional or version-bound |
| PRACTITIONER_REPORT | Engineering report, DORA report, industry survey | Observed practice, trends and field measurements | Selection bias, sampling, self-report and causal inference limitations |
| CASE_POSTMORTEM | Incident report, outage review, public engineering case | Event-specific causal chain, context and lessons | One case does not establish universal causality |
| SECONDARY_SYNTHESIS | Systematic mapping, literature review, meta-analysis | Synthesis across primary research | Inclusion/exclusion rules, search date and publication bias matter |
| INDEX_CATALOG | DOI registry, ISBN catalog, publisher catalog | Identity resolution and bibliographic metadata | Not evidence for the substantive claim |

Keep source class, source role (primary/secondary/tertiary), evidence type and claim-authority fit as separate fields.

## 3. Claim-authority fit

| Claim type | Preferred evidence | Required caution |
|---|---|---|
| Normative requirement | Current applicable standard, regulation, contract or official specification | Record exact clause only after permitted access; identify jurisdiction and edition |
| Discipline taxonomy | Professional BOK plus curriculum and literature cross-check | Do not mistake a guide or curriculum for the complete domain |
| Conceptual definition | Original theory/seminal paper, authoritative textbook, standards vocabulary | Record competing definitions and context |
| Empirical association/effect | Primary empirical study and independent replications/systematic review | Do not infer causation from correlation |
| Tool/API behavior | Current official documentation plus reproducible test where feasible | Pin product/version/date; docs may be stale |
| Historical case fact | Original incident/postmortem and independent corroboration | Separate observed facts from the author's causal interpretation |
| Practice effectiveness | Empirical studies, controlled comparisons, repeated field evidence | State context and strength; do not elevate anecdotes to universal rules |
| Curriculum/competency | Official curriculum and accreditation material | Do not infer professional completeness from course hours |

Authority dimensions are assessed independently: relevance to the exact claim; author/organization competence and editorial process; publication/review process; currency and edition status; methods and transparency; independent corroboration/replication; applicability to the target context; conflicts of interest and known bias; exact locator and provenance; access and reuse rights.

Do not collapse these into a single numeric “authority score” unless a later experiment demonstrates that such a score is meaningful and reviewable.

## 4. Verification states

The following states are distinct and must not be silently collapsed:

1. **CANDIDATE** — discovered but not identity-verified.
2. **IDENTITY_VERIFIED** — title, author/organization, edition/version and official identifier/URL checked against an authoritative catalog or publisher.
3. **CONTENT_VERIFIED** — the permitted source content relevant to the proposed claim was inspected.
4. **LOCATOR_VERIFIED** — the exact chapter/section/page/clause/paragraph was inspected and recorded.
5. **CLAIM_EXTRACTED** — an atomic, attributed claim has been written with a link to its evidence.
6. **REVIEWED** — a human reviewer checked the source, locator, interpretation and context.
7. **ACCEPTED** — the claim is accepted for this corpus scope and version.

Additional states/flags: ACCESS_RESTRICTED, BIBLIOGRAPHIC_ONLY, DISPUTED, SUPERSEDED, WITHDRAWN and STALE.

A source can be identity-verified while its content is not. A content-verified abstract is not full-text verification.

## 5. Stable IDs and source record

Candidate source ID pattern:

**SE-SRC-<ORGANIZATION>-<SHORT-TITLE>-<EDITION-OR-YEAR>**

Rules:
- IDs are immutable once referenced; title/edition corrections do not recycle IDs.
- A new edition receives a new source ID and a SUPERSEDES relation.
- Do not encode mutable URL fragments or a Git branch name as the only identity.
- DOI, ISBN, standard number, report number or official catalog identifier is preferred when actually verified.
- When no stable external identifier is verified, retain the official publisher URL and mark the missing identifier as null/unknown rather than inventing one.

Required record fields:
- stable ID, source class and source role;
- exact title and author/editor/organization;
- edition, revision, publication date and status;
- DOI, ISBN, standard/report number and official URL where verified;
- publication venue/publisher and peer-review/consensus process where relevant;
- relevant domain areas and intended claim types;
- authority rationale for those claim types;
- exact source locator and locator-verification status;
- access date and permitted access method;
- license/copyright/AI-use constraints;
- scope, assumptions, known limitations, conflicts and supersession;
- verification state and reviewer decision.

## 6. Evidence and provenance chain

Canonical traceability path:

~~~text
Knowledge Entity / Claim
  → Evidence Record
  → Source Locator
  → Source Record
~~~

Evidence record fields should include:
- evidence ID and evidence type (normative text, empirical result, case observation, secondary synthesis, official behavior, etc.);
- source ID and exact locator ID;
- concise paraphrase or permitted short quotation;
- context, population/system/version, assumptions and boundary conditions;
- method/design and result when empirical;
- limitations, bias, uncertainty and contrary evidence;
- extraction date, extractor, reviewer and status;
- optional content digest only where the source may lawfully be retained and hashed.

A relationship assertion is itself a claim: it needs a predicate, subject/object IDs, context, provenance, status and (when material) evidence. A graph edge must not become more authoritative than the evidence supporting it.

## 7. Conflict and change handling

- Keep contradictory claims as separate records with their own evidence and context.
- Do not silently average incompatible findings.
- Distinguish factual disagreement, scope mismatch, definition mismatch, method differences and temporal change.
- A newer edition may supersede the applicability of an older source without erasing historical evidence.
- If a source is withdrawn/retracted, preserve the record and mark affected claims for review.
- If a locator cannot be verified, record LOCATOR_UNKNOWN; do not guess page or clause numbers.
- If only a public abstract is accessible, do not claim the entire paper/standard/book was inspected.

## 8. Rights and access controls

Copyright, license, subscription and source-specific AI-use restrictions are part of provenance, not an afterthought.

Current verified constraints:
- SWEBOK Guide v4.0a is downloadable for personal/academic use; its copyright notice requires permission for commercial reprint/redistribution or new collective works. This corpus stores concise facts, bibliographic references and original synthesis, not substantial reproduced text. See the PDF copyright/reprint notice, printed page iv.
- ISO publication pages state that ISO content must not be used to train or prompt AI/ML tools except content expressly available through ISO Open Data. For this experiment, ISO records are bibliographic-only unless a permitted human-access and evidence-recording procedure is established. Do not ingest ISO normative text into an LLM prompt, reproduce clauses, or store generated clause summaries.
- For other publishers, verify the applicable rights before storing excerpts or redistributing full text. Public availability is not equivalent to permission for unrestricted reuse.

## 9. Human review protocol

A reviewer should be able to answer:
1. Does the source exist and is the edition/version correct?
2. Is this source authoritative for this specific claim type?
3. Does the locator actually support the claim?
4. Is the paraphrase faithful and appropriately scoped?
5. Are assumptions, uncertainty and counterevidence visible?
6. Is the claim atomic enough to be independently accepted or rejected?
7. Are rights/access restrictions respected?
8. Does the graph relationship assert more than the evidence supports?

Reviewer outcomes: ACCEPT, ACCEPT_WITH_QUALIFICATION, REJECT, DISPUTED, NEEDS_SOURCE and NEEDS_LOCATOR.

## 10. Negative rules

- Never invent a book, author, edition, DOI, ISBN, standard number, URL, quote, page, clause or study result.
- Never treat search snippets as full-source verification.
- Never treat an official logo, publisher or standard number as proof that every claim is supported.
- Never treat a vendor's marketing page as independent empirical evidence.
- Never treat a single case study as universal causal proof.
- Never remove a counterexample because it complicates a preferred principle.
- Never store hidden model chain-of-thought; record observable method decisions and evidence only.

## 11. Initial registry status

The first registry is deliberately small and representative. It contains one primary professional BOK taxonomy source (SWEBOK), one curriculum comparator (CS2023), multiple standards by bibliographic identity for process, systems, quality, architecture and testing views, and one official secure-development framework (NIST SSDF).

The registry records exactly what has been verified. It does not imply full normative text access or that each source has already contributed an accepted knowledge claim.
