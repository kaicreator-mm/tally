# Source Review — SWEBOK Guide v4.0a

**Source ID:** `SE-SRC-SWEBOK-GUIDE-4-0A`  
**Source class:** PROFESSIONAL_BOK  
**Review state:** CONTENT_AND_LOCATORS_VERIFIED / HUMAN_REVIEW_PENDING  
**Accessed:** 2026-11-04

## 1. Identity verification

- **Exact title:** *Guide to the Software Engineering Body of Knowledge (SWEBOK Guide), Version 4.0a*
- **Editor:** Hironori Washizaki
- **Organization / publisher:** IEEE Computer Society
- **Edition:** Version 4.0a
- **Release statement:** The official PDF title page says **Released August 2026**.
- **ISBN:** 978-0-7695-0000-3 (printed on the official PDF copyright page)
- **Official PDF:** https://ieeecs-media.computer.org/media/education/swebok/swebok-v4.pdf
- **Official landing page:** https://www.computer.org/education/bodies-of-knowledge/software-engineering

The prior provisional ID suffix `-2025` was based on an incorrect initial date assumption. The old ID is retained as a non-canonical alias in the Source Registry; new references use the canonical year-free ID above. The suffix is not publication metadata.

## 2. Authority fit

SWEBOK is an IEEE Computer Society professional body-of-knowledge guide. The introduction reports a development/review process involving 130+ reviewers from multiple countries and states that the Guide organizes the generally accepted portion of the discipline and provides topical access.

It is appropriate as:
- a primary reference for what SWEBOK v4.0a itself classifies as Software Engineering Knowledge Areas and related disciplines;
- a professional-consensus-oriented taxonomy reference;
- a topic discovery map for further source research.

It is not, by itself:
- the entire Software Engineering body of knowledge;
- a comprehensive bibliography or evidence review;
- a normative lifecycle standard;
- empirical proof that every listed practice is effective in every context;
- a complete characterization of related disciplines.

## 3. Verified content and exact locators

Locators below use the PDF's zero-based page index where stated, plus the document's printed Roman-numeral page labels.

| Locator ID | Exact locator | Verified content |
|---|---|---|
| SE-LOC-SWEBOK-TITLE-2026-08 | PDF page index 0, title page | Version 4.0a; editor; Released August 2026 |
| SE-LOC-SWEBOK-ISBN-COPYRIGHT | PDF page index 4; printed page iv | ISBN and copyright/reprint permission notice |
| SE-LOC-SWEBOK-SE-DEFINITION | PDF page indices 37–38; printed pages xxxvii–xxxviii; Introduction, “What Is Software Engineering?” | The Guide attributes a definition to ISO/IEC/IEEE SEVOCAB; historical definitions and scope discussion |
| SE-LOC-SWEBOK-KA-TABLE | PDF page index 39; printed page xxxix; Table I.1 | 18 Knowledge Areas; Chapters 1–15 are Software Engineering KAs and Chapters 16–18 are foundation KAs |
| SE-LOC-SWEBOK-RELATED-DISCIPLINES | PDF page index 39; printed page xxxix; Table I.2 | Eleven related disciplines; the Guide says characterizing their knowledge is not its objective |
| SE-LOC-SWEBOK-HIERARCHY | PDF page index 40; printed page xl; “Hierarchical Organization” | Two- or three-level topic decomposition; not designed around particular application domains, business uses or management philosophies |
| SE-LOC-SWEBOK-CITATION-LIMIT | PDF page index 40; printed page xl; “Reference Material and Matrix” | The Guide does not attempt comprehensive citations |
| SE-LOC-SWEBOK-DEPTH | PDF page index 40; printed page xl; “Depth of Treatment” | “Generally accepted” does not mean uniform application to all software engineering endeavors; project needs determine what to apply and how |

### 18 Knowledge Areas

Table I.1 lists:
1. Software Requirements
2. Software Architecture
3. Software Design
4. Software Construction
5. Software Testing
6. Software Engineering Operations
7. Software Maintenance
8. Software Configuration Management
9. Software Engineering Management
10. Software Engineering Process
11. Software Engineering Models and Methods
12. Software Quality
13. Software Security
14. Software Engineering Professional Practice
15. Software Engineering Economics
16. Computing Foundations
17. Mathematical Foundations
18. Engineering Foundations

The Guide explicitly groups the first 15 as Software Engineering KAs and the final three as foundation KAs. These are source labels, not Tally-owned classifications.

### Related disciplines

Table I.2 lists Business Analysis, Computer Engineering, Computer Science, Cybersecurity, Data Science, General Management, Information Systems and Technology, Mathematics, Project Management, Quality Management and Systems Engineering. The Guide explicitly says characterizing the knowledge of these related disciplines is not an objective of the Guide.

## 4. Scope, assumptions and limitations

- The Guide is a map to the body of knowledge, not the complete body of knowledge.
- Its references are intentionally non-comprehensive; the absence of a reference is not evidence that a topic lacks authoritative literature.
- Its two/three-level hierarchy is a topical navigation structure. It does not claim that KAs are disjoint or that cross-KA relationships are absent.
- The Guide states that application domain, business use, management philosophy and development method are not the organizing basis of its topic breakdown.
- Generally accepted knowledge is not a universal prescription. Context and project needs govern applicability.
- The Guide's definition section attributes the SEVOCAB definition to ISO/IEC/IEEE 24765; do not reattribute that definition as an original SWEBOK definition.

## 5. Rights and handling

The official copyright page permits educational or personal use subject to conditions. Commercial reprint/republishing, advertising/promotional use, and creating new collective works for resale or redistribution require permission from IEEE. The corpus therefore stores bibliographic facts, exact locators, short original paraphrases and original analysis only; it does not reproduce substantial Guide text.

## 6. Verification record

- Identity: VERIFIED against the official PDF title and copyright pages.
- Content: VERIFIED for the introduction, Tables I.1/I.2 and stated scope limitations listed above.
- Locator: VERIFIED for the listed PDF page indices and printed page labels.
- Claim extraction: separate records in `knowledge/vertical-slice/KNOWLEDGE_RECORDS.yaml`.
- Human review: PENDING.
- Acceptance into final corpus: NOT_YET_ACCEPTED.

## 7. Open questions

- The exact relationship between the SWEBOK reference spine and a Tally-owned multi-view taxonomy requires independent review.
- SWEBOK's own references should be researched as leads, not treated as automatically verified source records.
- The 18 KAs are a reference classification, not a claim that every knowledge item has exactly one parent.
