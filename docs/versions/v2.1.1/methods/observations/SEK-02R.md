# SEK-02R — SWEBOK v4.0a Source Identity and Rights Reconciliation

**Task:** #325 (follow-up to SEK-02 #302)
**Date:** 2026-11-06
**Baseline:** cf2dbc845bfc2dcbc110ed52e240e79953a4cba9
**Status:** PDF_METADATA_RESTORED / LANDING_PAGE_DATE_DISCREPANCY_OPEN / NATIVE_HASH_VERIFICATION_PENDING

## Triggering discrepancy

The WIP baseline's SOURCE_REGISTRY.yaml, SWEBOK source-review dossier and historical SEK-02.md stated that the official SWEBOK v4.0a title page said “Released August 2026” and that ISBN 978-0-7695-0000-3 appeared on the copyright page.

Reinspection of the current official sources found:
- Current official PDF title page: “Released August 2026”.
- Official IEEE landing page: 25 September 2025 update identifying the v4.0a release. This conflicts with the current PDF title-page release statement; the relationship is unresolved.
- Current PDF page index 4 / printed page iv: copyright © 2014–2025, ISBN 978-0-7695-0000-3, and copyright/reprint permissions.
- The PDF copyright page and official landing page provide source-specific rights restrictions; preserve both without inferring broad reuse permission.

Official references:
- PDF: https://ieeecs-media.computer.org/media/education/swebok/swebok-v4.pdf
- Landing page: https://www.computer.org/education/bodies-of-knowledge/software-engineering/resources/
- Download page: https://www.computer.org/education/bodies-of-knowledge/software-engineering/v4

## Correction

- Preserve canonical source ID SE-SRC-SWEBOK-GUIDE-4-0A; it is year-free and identifies the edition.
- Retain publication_date 2026-08 as the exact current PDF title-page statement, while separately recording the landing page's 2025-09 update statement as an unresolved cross-source discrepancy.
- Retain ISBN 978-0-7695-0000-3 as printed on PDF page index 4 / printed page iv; do not assert a DOI.
- Keep locator SE-LOC-SWEBOK-TITLE-2026-08 anchored to the current PDF title page and update all in-scope references to that exact locator.
- Keep the ISBN/copyright locator tied to the PDF page that actually prints the ISBN.
- Preserve both the PDF copyright-page terms and the current landing-page restrictions. No PDF is committed or redistributed; only metadata, locators and concise original paraphrases are stored.
- Historical SEK-02.md is not rewritten. This observation supersedes its incorrect August-2026/ISBN assertions.

## Verification still required

- Native download of the current official PDF, SHA-256 and file size: PENDING.
- Confirm that the downloaded bytes match the current official URL and not an old cached copy; compare the downloaded title/copyright pages against the current PDF extraction.
- Resolve or explicitly preserve the official landing-page date discrepancy; do not silently collapse the two source statements.
- Duplicate-key YAML parse and source-locator reference closure: PENDING native validation.
- Human review of the corrected bibliographic record: PENDING.

No taxonomy, claim acceptance, schema, LinkML, Domain Package or DomainForge decision is changed by this metadata correction.


## Independent source re-check — 2026-11-06

A fresh read of the current official PDF URL exposes “Released August 2026” on
PDF page index 0 and ISBN 978-0-7695-0000-3 on PDF page index 4 / printed page iv.
The official IEEE landing page still records a 25 September 2025 update for
v4.0a. The earlier correction in this PR incorrectly treated the PDF's current
title-page date and printed ISBN as unsupported. This repair restores the
PDF-grounded values and records the landing-page conflict as unresolved.
Web extraction is not a substitute for the required native download: the
current PDF SHA-256 and byte size remain pending, and no merge is permitted
until native validation and fresh independent Exact-SHA review pass.
