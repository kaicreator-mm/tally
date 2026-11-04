# SEK-03 — Method Observation

**Task:** #304  
**Date:** 2026-11-04  
**Status:** OBSERVED / DRAFT

## Observations

1. SWEBOK v4.0a can support exact source locators for the 18-KA table, related-discipline table and hierarchy/scope discussion. The review dossier distinguishes printed page numbers from PDF viewer page indices.
2. Directly opening the CS2023 Body of Knowledge PDF materially improved verification: the nine SE Knowledge Units and SE-Tools content are now tied to printed pages 166–170 rather than only search-result snippets.
3. The nine CS2023 SE KUs demonstrate a different granularity and purpose from SWEBOK's 18 KAs. The curriculum includes a Tools and Environments unit with SCM/version control, release management, testing tools and CI/process automation; this does not make it equivalent to a SWEBOK KA.
4. NIST SP 800-218 v1.1 is verified only at official metadata and abstract level in this slice. No detailed secure-development practice claim is extracted yet.
5. ISO standards remain bibliographic-only; no normative clauses or abstract-derived claims are included.
6. The first evidence set is intentionally small: seven atomic claims, seven evidence records, eight locators and three source references. It is sufficient to test basic provenance structures, but not to decide the final ontology.

## Negative / unresolved observations

- No full-text review of every SWEBOK chapter or every CS2023 Knowledge Unit has occurred.
- No full NIST SP 800-218 practice-level extraction has occurred.
- SWEBOK commercial reuse conditions require separate rights review before broader redistribution or commercial republication.
- The YAML fixture has not yet been validated with LinkML CLI.
- All claims remain PROPOSED pending human review.

## Method adjustment

Prefer direct official PDFs and exact printed locators over search snippets. Keep source identity, content verification, locator verification and claim acceptance as separate states. Use a deliberately small but heterogeneous sample before scaling the corpus.
