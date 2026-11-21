# SEK-07C — Security Engineering lane

Status: DRAFT_PENDING_HUMAN_REVIEW. All knowledge records are PROPOSED or
PENDING_REVIEW; none is ACCEPTED. No ReviewDecision records exist in this lane.

## Scope and authority fit

Software security engineering across threat/risk analysis, security requirements
linkage, secure architecture/design, secure implementation, security verification,
vulnerability handling and security assurance, with emphasis on traceable
applicability, threat assumptions and lifecycle integration.

Three authority tiers are kept distinct on every record — no authority inflation:

- **Professional guidance** — SWEBOK Guide v4.0a, KA13 Software Security
  (`se:SRC-SWEBOK-GUIDE-4-0A`; a guide to generally accepted knowledge, not a
  prescription; its topic citations are not comprehensive).
- **Government recommendation** — NIST SP 800-218 v1.1 SSDF
  (`se:SRC-NIST-SP-800-218-V1-1`; recommendation framework, risk-based adoption,
  not an ISO/IEC/IEEE International Standard and not a universal mandatory
  sequence).
- **Curriculum framing** — CS2023 Security knowledge area, SEC-Engineering and
  SEC-Foundations units (`se:SRC-CS2023`; curriculum model, not a professional
  practice taxonomy).

NIST SP 800-218A (`se:SRC-NIST-SP-800-218A-2024`) is registry-verified at
ABSTRACT_VERIFIED (metadata/abstract only); it backs no content claim here.
ISO sources remain metadata-only in the registry and are not used by this lane.

Out of scope (owned by other lanes): generic construction/coding-style content
(07B), reliability/performance (07D), delivery pipeline mechanics (07E), testing
execution mechanics (SEK-06C, whose NIST locators cover PW.7, PW.8 and
RV.1-RV.3 — this lane cites different SSDF sections: PO.1, PW.1, PW.2).

## Taxonomy view

`se:VIEW-07C-SECURITY-ENGINEERING` (view_kind SECURITY, coverage PARTIAL)
organizes the lane as a lifecycle-cross-cutting lens with five nodes:
`se:NODE-07C-SECURITY-LIFECYCLE` (organizing concern) containing
`se:NODE-07C-SECURITY-REQUIREMENTS`, `se:NODE-07C-SECURE-DESIGN`,
`se:NODE-07C-SECURE-IMPLEMENTATION` and `se:NODE-07C-SECURITY-VERIFICATION`
(PART_OF edges within the single view). The shared CS2023 curriculum view
`se:VIEW-CS2023-SE` is carried verbatim; the lane does not force these views
into one hierarchy.

## Sources, editions, access and rights limits

- **SWEBOK v4.0a** — official PDF re-verified by native SHA-256
  `0c9db4da0b2425c58cf778b99131eb31bb6c188e98b67163a549a1366d96e511`
  (411 pages). KA13 chapter starts at PDF page index 263 (1-based page 264).
  Rights: educational/personal-use conditions with IEEE permission required
  beyond them; this lane stores bibliographic facts, exact locators and concise
  original paraphrases only; no PDF mirroring, no substantial reproduction, no
  inferred redistribution permission.
- **NIST SP 800-218 v1.1** — official PDF verified by SHA-256
  `617746e553a9e2da49bfbd4eef0dfc3094758a39b869314e4173ac36605cde22`
  (739,891 bytes, 36 pages). US-government work, not subject to US copyright;
  attribution appreciated; concise original paraphrase with official citation
  is used regardless.
- **CS2023** — official Body of Knowledge PDF full text obtained via a
  text-extraction service (direct binary download unavailable from this
  network; no binary hash recorded); heading-anchored locators only, no page
  numbers asserted beyond registry-verified anchors. Permission scope covers
  educational materials and programs; other use requires specific permission —
  see the shared `se:GAP-CS2023-RIGHTS-CLEARANCE` (HIGH/OPEN), carried in this
  lane.

## Record counts

| Collection | Count |
|---|---|
| domains (shared, verbatim) | 1 |
| taxonomy_views (1 shared + 1 lane-local) | 2 |
| taxonomy_nodes | 5 |
| source_locators (5 SWEBOK, 3 NIST, 2 CS2023) | 10 |
| evidence_records (13 direct paraphrase + 1 marked synthesis) | 14 |
| concepts | 3 |
| knowledge_claims | 9 |
| principles | 1 |
| practices | 3 |
| patterns | 1 |
| anti_patterns | 1 |
| decision_models | 1 |
| coverage_gaps (3 lane-local + 1 shared verbatim) | 4 |
| relationship_assertions | 30 |

Every knowledge claim has a SUPPORTED_BY edge to a lane Evidence record; every
Evidence record chains to a lane SourceLocator that names a registry Source.

## Gaps and limitations

- `se:GAP-07C-SWEBOK-KA13-UNINSPECTED-SECTIONS` — KA13 sections 1.3, 2 (2.1-2.3),
  3.2, 5.2 and 6 were read during extraction but are not admitted as lane
  content (MEDIUM/OPEN).
- `se:GAP-07C-NO-CASE-EVIDENCE` — no incident/postmortem/case-study source is
  registered; the lane has no Case or Counterexample records and the
  anti-pattern's consequences are inference, not observed outcomes (MEDIUM/OPEN).
- `se:GAP-07C-NIST-218A-ABSTRACT-ONLY` — the AI companion profile's practices
  are not represented; lane practices must not be generalized to generative-AI
  development (MEDIUM/OPEN).
- `se:GAP-CS2023-RIGHTS-CLEARANCE` — shared HIGH/OPEN rights gap, carried
  verbatim.
- The anti-pattern `se:ANT-07C-SECURITY-AFTERTHOUGHT` rests on the explicitly
  marked synthesis evidence `se:EVD-07C-SYNTHESIS-AFTERTHOUGHT`, not on any
  source sentence. HUMAN_REVIEW_BURDEN for this lane is NOT_MEASURED.

## Reproduction steps

1. Sources: SWEBOK PDF at `_tmp/tally-sek07-lanes/sources/swebok-v4-bust.pdf`
   (hash above), NIST PDF at `_tmp/tally-sek07-lanes/sources/nist-sp-800-218.pdf`
   (hash above), CS2023 SEC verbatim selections at
   `_tmp/tally-sek07-lanes/sources/cs2023-bok-sek07-extract.md`.
2. Extract with the pinned venv Python (3.14.7, pypdf 7.0.0):
   `scripts/extract_pages.py <pdf> <start> <end>` (1-based inclusive pages);
   SWEBOK KA13 = pages 264-273; NIST PO.1 = pages 14-15, PW.1 = page 20,
   PW.2 = page 21.
3. Validate at the lane tip:
   `docs/versions/v2.1.1/knowledge/model/validate_knowledge_graph.py --self-test`
   (66/66) and
   `_tmp/tally-sek07-lanes/scripts/assemble_and_validate.py <worktree> 07C
   docs/versions/v2.1.1/knowledge/vertical-slice/lanes/security-engineering/records.yaml
   <outdir>`.
4. Validation receipts live in `_tmp/tally-sek07-lanes/validation/07C/`
   (outside the repository).

This lane is independently reviewable; it is not the integrated graph root.
SEK-08 performs single-writer graph materialization/export. No merge, freeze,
release or production adoption is authorized by this lane; records await
independent Exact-SHA human review.
