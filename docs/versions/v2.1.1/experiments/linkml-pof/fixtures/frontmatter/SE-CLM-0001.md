---
id: se:SE-POF-CORPUS-0001
title: "SE vertical-slice LinkML proof-of-fit fixture"
corpus_status: DRAFT_PENDING_HUMAN_REVIEW
sources:
  - id: SE-SRC-SWEBOK-GUIDE-4-0A
    title: "Guide to the Software Engineering Body of Knowledge (SWEBOK Guide), Version 4.0a"
    source_class: PROFESSIONAL_BOK
    official_url: "https://ieeecs-media.computer.org/media/education/swebok/swebok-v4.pdf"
    edition: "Version 4.0a"
    publication_date: "2026-08"
source_locators:
  - id: SE-LOC-SWEBOK-KA-TABLE
    source_ref: se:SE-SRC-SWEBOK-GUIDE-4-0A
    locator_type: TABLE
    locator: "PDF page index 39; printed page xxxix; Introduction to the Guide, Table I.1"
    verification_state: LOCATOR_VERIFIED
evidence_records:
  - id: SE-EVD-0001
    evidence_type: DIRECT_SOURCE_PARAPHRASE
    source_locator_ref: se:SE-LOC-SWEBOK-KA-TABLE
    paraphrase: "SWEBOK v4.0a lists 18 Knowledge Areas and classifies the first 15 as Software Engineering KAs and the last three as foundation KAs."
    context: "The claim describes the Guide's own taxonomy, not an independently exhaustive classification of the discipline."
    extraction_status: EXTRACTED_PENDING_REVIEW
domain_nodes:
  - id: SE-KA-13
    label: Software Security
    view_id: SE-VIEW-DISCIPLINE
    source_ref: se:SE-SRC-SWEBOK-GUIDE-4-0A
    source_locator_ref: se:SE-LOC-SWEBOK-KA-TABLE
    taxonomy_ref: SE-TAXONOMY-0.1.0
knowledge_claims:
  - id: SE-CLM-0001
    claim_type: TAXONOMY_STATEMENT
    statement: "SWEBOK Guide v4.0a lists 18 Knowledge Areas; the first 15 are Software Engineering KAs and the final three are foundation KAs."
    scope: "The taxonomy as defined by SWEBOK Guide v4.0a."
    applicability: "Use as a reference mapping for domain discovery; do not claim it is the only valid taxonomy."
    limitations:
      - "Guide scope is not the entire body of knowledge."
    review_status: PENDING
    source_refs:
      - se:SE-SRC-SWEBOK-GUIDE-4-0A
    taxonomy_refs:
      - SE-TAXONOMY-0.1.0
    domain_node_refs:
      - se:SE-KA-13
relationship_assertions:
  - id: SE-REL-POF-0001
    subject_ref: se:SE-CLM-0001
    predicate: SUPPORTED_BY
    object_ref: se:SE-EVD-0001

    evidence_ref: se:SE-EVD-0001
    status: PENDING
  - id: SE-REL-POF-0002
    subject_ref: se:SE-CLM-0001
    predicate: INCLUDES
    object_ref: se:SE-KA-13

    evidence_ref: se:SE-EVD-0001
    status: PENDING
---
# SWEBOK taxonomy claim

This record states what the SWEBOK Guide v4.0a itself classifies. It is not a claim that SWEBOK is an exhaustive or exclusive ontology for Software Engineering.

**Review status:** pending human review.  
**Evidence:** `SE-EVD-0001`, located at Table I.1 on printed page xxxix.  
**Source:** IEEE Computer Society, SWEBOK Guide v4.0a, released August 2026.
