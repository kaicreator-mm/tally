# SEK-02R — SWEBOK v4.0a Source Identity and Rights Reconciliation

**Task:** #325 (follow-up to SEK-02 #302)
**Date:** 2026-11-06
**Baseline:** cf2dbc845bfc2dcbc110ed52e240e79953a4cba9
**Status:** PDF_METADATA_RESTORED / LANDING_PAGE_DATE_DISCREPANCY_OPEN / NATIVE_HASH_VERIFIED_2026-11-06

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
- Historical SEK-02.md is not rewritten. Its August-2026 release-date and ISBN assertions are re-confirmed against the current official PDF and retained; this observation supersedes only the interim in-PR correction that had wrongly treated them as unsupported, and it adds the landing-page date discrepancy and rights boundaries that SEK-02 did not record.

## Verification status

- Native download of the current official PDF: COMPLETED 2026-11-06 by the independent Exact-SHA reviewer (fresh curl fetch from the official PDF URL). SHA-256 `0c9db4da0b2425c58cf778b99131eb31bb6c188e98b67163a549a1366d96e511`; file size 6,015,672 bytes. The downloaded bytes are byte-identical to the retained verification copy (not a stale-cache reuse), and an independent pypdf 6.21.4 extraction of PDF page index 0 and page index 4 reproduces “Released August 2026” and “ISBN 978-0-7695-0000-3” with copyright © 2014–2025.
- Resolve or explicitly preserve the official landing-page date discrepancy; do not silently collapse the two source statements: STILL OPEN — both statements are preserved explicitly and IEEE clarification remains pending.
- Duplicate-key YAML parse and source-locator reference closure: COMPLETED 2026-11-06 (Python 3.14.7 / PyYAML 6.0.3 strict duplicate-key rejection; 21/21 checks PASS, exit 0; independent closure spot-checks also PASS).
- Historical receipt — then-current-tip re-verification at the reviewed HEAD `a1c752e54da5f64e3f2c0c5903bd4bbe4d1e46c7` (tree `8d5371c19c03ae60b161f1beea40297072325de6`; superseded as tip by the Builder-batch commit `9291675`, whose final-tip receipt is appended at the end of this file): the same 21-check strict parse/closure suite re-run 2026-11-06 — 21/21 PASS, exit 0, worktree clean before/after; the retained verification PDF re-confirmed bit-identical (SHA-256 `0c9db4da0b2425c58cf778b99131eb31bb6c188e98b67163a549a1366d96e511`, 6,015,672 bytes, 411 pages; title page “Released August 2026” and printed-page-iv ISBN re-extracted). Receipt `_tmp/i325-localval-r4/` (outside the write set).
- Human review of the corrected bibliographic record: PENDING.

No taxonomy, claim acceptance, schema, LinkML, Domain Package or DomainForge decision is changed by this metadata correction.


## Independent source re-check — 2026-11-06

A fresh read of the current official PDF URL exposes “Released August 2026” on
PDF page index 0 and ISBN 978-0-7695-0000-3 on PDF page index 4 / printed page iv.
The official IEEE landing page still records a 25 September 2025 update for
v4.0a. The earlier correction in this PR incorrectly treated the PDF's current
title-page date and printed ISBN as unsupported. This repair restores the
PDF-grounded values and records the landing-page conflict as unresolved.
Web extraction is not a substitute for the required native download. At the
time of this re-check the current PDF SHA-256 and byte size were still pending
and native validation and independent review were still outstanding; **that
pending state is superseded** — the native download with SHA-256/byte-size
verification, the strict duplicate-key YAML parse/reference closure, and the
fresh independent Exact-SHA review all completed on 2026-11-06 (see
“Verification status” above and the review section below). The PDF-versus-
landing-page release-date discrepancy itself remains OPEN pending IEEE
clarification; the merge gate now rests on that open discrepancy and human
review, not on any missing hash.


## Independent Exact-SHA review — 2026-11-06 (historical receipt at `f2785f4`)

A fresh independent review of PR #326 at HEAD `f2785f4603de46d1ae96a1c5125143b7adff8e40` re-verified from zero, without trusting prior receipts: branch/base identity and the exact five-path diff; the official PDF by native download (SHA-256 `0c9db4da0b2425c58cf778b99131eb31bb6c188e98b67163a549a1366d96e511`, 6,015,672 bytes, fresh fetch byte-identical to the retained copy); page-index-0 and page-index-4 text by independent extraction; registry, dossier, knowledge-record, README and observation cross-consistency; and strict duplicate-key YAML parse plus reference closure (21/21 PASS, exit 0, plus independent closure spot-checks). One residual contradiction in this file was repaired by the reviewer: the line "supersedes its incorrect August-2026/ISBN assertions" had described the superseded interim correction as if it were the final state — SEK-02's August-2026 and ISBN statements are confirmed by the current PDF and retained. The landing-page release-date discrepancy remains OPEN pending IEEE clarification; human review remains PENDING.

## Final-tip receipt — Builder batch `9291675` (2026-11-06 R2 repair append)

The R2 Exact-SHA review of PR #326 found that the receipts above were still bound to earlier heads. This section appends the missing final-tip receipt; the `f2785f4` independent review and the `a1c752e` re-verification above are historical receipts, preserved unchanged.

- Identity: HEAD `9291675ff3e0e6d31a1027bdd514a43318993b5c`, tree `3fb2ccc41281d78a9703b801ff18a98040d08d47`, base `cf2dbc845bfc2dcbc110ed52e240e79953a4cba9` (merge-base equals base; PR #326 headRefOid equalled this HEAD at the R2 review).
- Commands and exit codes at this exact tip — `validate_i325.py` (strict duplicate-key YAML parse + source/locator reference closure): 21/21 PASS, exit 0; retained-PDF recheck on both retained native copies (`pdf_verify.py`, pypdf 6.21.4): SHA-256 `0c9db4da0b2425c58cf778b99131eb31bb6c188e98b67163a549a1366d96e511`, 6,015,672 bytes, 411 pages, page-index-0 “Released August 2026”, page-index-4 (printed iv) ISBN 978-0-7695-0000-3 with © 2014–2025 — all PASS, exit 0. Clean worktree before/after: 0/0 dirty. Logs: Builder batch `_tmp/i325-localval-r4/logs/` (`validate-tip.log`, `validate-post.log`, `pdf-verify-r2.log`, `pdf-verify-r3.log`), independently re-run for this repair in `_tmp/i325-localval-r5/logs/` (`validate-pre.log`, `pdf-verify.log`, `identity-pre.txt`).
- Five-path SHA256 manifest at HEAD `9291675` / tree `3fb2ccc`:

```text
5eea9ca55f4a824307b2769b01d34742aab8eef797ea3e4db0a5657085cabef5  docs/versions/v2.1.1/authority/SOURCE_REGISTRY.yaml
af8b085f89cdd76a9f9b8384e021f18b016266dcc97bc37226855a31f22e6834  docs/versions/v2.1.1/authority/source-reviews/SWBOK-GUIDE-v4.0a.md
e4408a8b672d92967e7732462b9e5da985088722363eb1a1d1e8e137f4535f35  docs/versions/v2.1.1/knowledge/vertical-slice/KNOWLEDGE_RECORDS.yaml
cb4078fc271f741d21741618faa6a0b6d91e9a8b68c4dc6d3d53aaf72af6ee73  docs/versions/v2.1.1/knowledge/vertical-slice/README.md
ec320d086eae6afe07cd72413a11e7e5b8945a86f87954d891ff682ebf5e8a76  docs/versions/v2.1.1/methods/observations/SEK-02R.md
```

This append itself is an observation-only R2-repair commit and becomes the newest tip; its exact HEAD/TREE, the post-commit rerun (same commands, exit 0), the updated five-path manifest and clean-worktree evidence are delivered in the R2 repair delivery comment on PR #326 / coordinator #321, with logs in `_tmp/i325-localval-r5/`.

Still OPEN by design, unchanged by this receipt: the PDF-versus-landing-page release-date discrepancy (IEEE clarification pending) and human bibliographic review (PENDING). No DOI asserted; no taxonomy, claim-status, schema or rights change.
