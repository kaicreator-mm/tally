# SEK-02R — SWEBOK v4.0a Source Identity and Rights Reconciliation

**Task:** #325 (follow-up to SEK-02 #302)
**Date:** 2026-11-06
**Baseline:** cf2dbc845bfc2dcbc110ed52e240e79953a4cba9
**Status:** SOURCE_METADATA_CORRECTED / NATIVE_HASH_VERIFICATION_PENDING

## Triggering discrepancy

The WIP baseline's SOURCE_REGISTRY.yaml, SWEBOK source-review dossier and historical SEK-02.md stated that the official SWEBOK v4.0a title page said “Released August 2026” and that ISBN 978-0-7695-0000-3 appeared on the copyright page.

Reinspection of the current official sources found:
- Current official PDF title page: “Released Sept 2025”.
- Official IEEE landing page: 25 September 2025 update identifying the v4.0a release.
- Current PDF page index 4 / printed page iv: copyright © 2014–2025 and copyright/reprint permissions; the asserted ISBN is not displayed on this page.
- Official landing page and PDF copyright page provide the rights restrictions recorded in the updated source dossier.

Official references:
- PDF: https://ieeecs-media.computer.org/media/education/swebok/swebok-v4.pdf
- Landing page: https://www.computer.org/education/bodies-of-knowledge/software-engineering/resources/
- Download page: https://www.computer.org/education/bodies-of-knowledge/software-engineering/v4

## Correction

- Preserve canonical source ID SE-SRC-SWEBOK-GUIDE-4-0A; it is year-free and identifies the edition.
- Correct publication date to 2025-09, consistent with the title page and official landing page.
- Remove the unsupported ISBN and stable-identifier assertion; do not invent a replacement ISBN or DOI.
- Replace locator SE-LOC-SWEBOK-TITLE-2026-08 with SE-LOC-SWEBOK-TITLE-2025-09 and update all references in the initial vertical-slice fixture.
- Replace the ISBN/copyright locator claim with a copyright-only locator. The current copyright page does not establish the previously asserted ISBN.
- Preserve both the PDF copyright-page terms and the current landing-page restrictions. No PDF is committed or redistributed; only metadata, locators and concise original paraphrases are stored.
- Historical SEK-02.md is not rewritten. This observation supersedes its incorrect August-2026/ISBN assertions.

## Verification still required

- Native download of the current official PDF, SHA-256 and file size: PENDING.
- Confirm that the downloaded bytes match the current official URL and not an old cached copy.
- Duplicate-key YAML parse and source-locator reference closure: PENDING native validation.
- Human review of the corrected bibliographic record: PENDING.

No taxonomy, claim acceptance, schema, LinkML, Domain Package or DomainForge decision is changed by this metadata correction.
