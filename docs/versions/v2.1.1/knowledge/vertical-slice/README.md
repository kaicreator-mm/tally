# Vertical Slice — Software Engineering Knowledge Records

**Slice:** `SE-VERTICAL-SLICE-0.1.0`  
**Status:** DRAFT / PENDING HUMAN REVIEW  
**Task:** [SEK-03 #304](https://github.com/kaicreator-mm/tally/issues/304)  
**Downstream experiment:** [SEK-04 #305](https://github.com/kaicreator-mm/tally/issues/305)

## Purpose

This is the first real-data fixture for the provenance chain and early LinkML Proof-of-fit. It intentionally uses three source families with different authority roles:

- SWEBOK v4.0a: professional BOK taxonomy and explicit scope limitations.
- CS2023: curriculum boundary, Knowledge Unit model and cross-area references.
- NIST SP 800-218 v1.1 plus the SP 800-218A companion abstract: government-published recommended secure-development practices and AI-specific companion scope.

## Contents

`KNOWLEDGE_RECORDS.yaml` contains:
- 17 exact source locators;
- 12 evidence records;
- 12 atomic claims;
- 12 typed `SUPPORTED_BY` relationship assertions.

Canonical traceability is expressed as:

`KnowledgeClaim --SUPPORTED_BY--> Evidence --located_at--> SourceLocator --identifies--> Source`

The source locator references a source ID from the central Source Registry. The relationship assertion is the canonical claim-to-evidence link; the YAML claim does not duplicate the same edge in an `evidence_refs` list.

## Admission status

- Every claim remains `PENDING` or `PENDING_LIMITED_EVIDENCE`.
- Every evidence record remains `EXTRACTED_PENDING_REVIEW`.
- No claim is accepted corpus knowledge.
- NIST SP 800-218A is supported only by its official metadata and abstract in this slice; its detailed practices are not represented.
- ISO records are not used for content claims because the current corpus's ISO access restriction remains in force.

## Known limits

- The slice is deliberately small and not coverage-complete.
- No empirical software-engineering outcome study is included yet.
- These records test traceability and authoring structure; they do not validate the entire proposed taxonomy.
- The source registry and taxonomy are experimental inputs and still require independent review.

## Reproduction and review

Review source dossiers in `authority/source-reviews/`, then follow each claim to its evidence, exact locator and source. A reviewer should be able to reject or qualify any one claim without editing unrelated records.


## Current source coverage and gaps

| Evidence family | Current coverage | Status |
|---|---|---|
| SWEBOK v4.0a | Discipline taxonomy, related-discipline boundary, Guide scope and applicability limitations | Source metadata corrected by SEK-02R: Released Sept 2025; no ISBN/DOI asserted; current rights limits recorded; human review pending |
| CS2023 | CS curriculum boundary, nine SE Knowledge Units, explicit cross-KA perspective references | Selected content/locators verified |
| NIST SP 800-218 v1.1 | Secure-development practice groups, SDLC integration, risk-based applicability and ordering caveat | Selected PDF sections verified |
| NIST SP 800-218A | AI-model-specific companion profile relationship | Metadata and abstract only |
| ISO/IEC/IEEE 12207, 15288, 25010, 42010, 29119-2 | Bibliographic identity and boundary candidates only | No ISO normative content admitted |

The first slice does **not** yet cover authoritative detailed claims for requirements, architecture/design, construction, testing, operations, configuration management, maintenance, quality, process/methods, engineering management/economics, professional practice or the three foundation KAs. These gaps are intentional and must drive the next source-research lanes. SWEBOK's reference list is a lead set, not a complete authority registry.


## SWEBOK source identity correction — SEK-02R

The current official IEEE PDF and landing page identify SWEBOK v4.0a as released/updated in September 2025. Earlier repository claims of “Released August 2026” and ISBN 978-0-7695-0000-3 are superseded; the current PDF copyright page does not display that ISBN. The canonical Source ID remains year-free. Do not reuse the superseded metadata. See the SEK-02R source dossier and observation for the exact official locators and rights boundary.
