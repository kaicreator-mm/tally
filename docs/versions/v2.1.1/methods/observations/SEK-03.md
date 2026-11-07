# SEK-03 — Method Observation

**Task:** #304  
**Date:** 2026-11-04  
**Status:** OBSERVED / EXECUTED / HUMAN_REVIEW_PENDING / DRAFT

## Work performed

- Reopened the official SWEBOK v4.0a PDF, not just the landing-page result; inspected the title page, copyright/ISBN page, introduction, Tables I.1/I.2, hierarchy and depth-of-treatment sections.
- Reopened the official CS2023 Knowledge Model and Body of Knowledge PDFs; verified the 17-KA curriculum model, nine SE Knowledge Units, cross-KA references and the SE-Tools unit.
- Opened the complete official NIST SP 800-218 v1.1 PDF; inspected its abstract, audience, introduction, risk-based adoption discussion, four practice groups, practice elements and Table 1 ordering note.
- Checked the NIST SP 800-218A official publication page and abstract. It is a separate 2024 companion profile for generative AI and dual-use foundation models, intended to be used with SP 800-218.
- Produced 12 atomic source-grounded claims, 12 evidence records, 17 source locators and 12 typed support relationships. All remain pending human review.

## Material corrections found during primary-source reinspection

1. The official SWEBOK v4.0a title page states **Released August 2026**, not September 2025. The copyright page prints ISBN 978-0-7695-0000-3. The original provisional source ID encoded the incorrect year. The central registry now has a year-free canonical ID and preserves the old ID as an alias.
2. The CS2023 Software Engineering KU page locators in the initial registry were incorrect. The official Body of Knowledge PDF places the SE section at PDF page indices 174–178; the registry now records PDF page indices and named sections.
3. The initial NIST registry record had only metadata/abstract verification. The complete SP 800-218 v1.1 PDF is now inspected for the exact locators in the source dossier.
4. NIST SP 800-218A was not in the initial registry. It is a separate companion profile that adds AI-model-specific practices. The registry now contains it as a separate source with abstract-only verification.

These are meaningful source-verification corrections and should remain in the method retrospective; do not erase the initial mistakes.

## Modeling observations

- The first real slice demonstrates that source records need not be forced into a single authority ranking: SWEBOK is authoritative for its own BOK taxonomy; CS2023 for its curriculum model; NIST for its own recommended framework.
- CS2023's explicit cross-KA references provide a direct example of why topic duplication and perspective links must be represented separately from identity. A single tree is not enough for the curriculum's own model.
- NIST's practice ordering explicitly does not encode implementation order or relative importance. This is a useful counterexample to assuming that a framework's table order is a task DAG.
- NIST's risk-based applicability language shows that a practice record needs conditions/context, not only a title and description.
- The companion SP 800-218A demonstrates that a source family can have separately versioned profiles. A source registry needs related-source and supersession/companion relationships.

## Negative and unresolved findings

- SWEBOK is not comprehensive in citations; its reference list cannot be treated as a complete authority registry.
- CS2023 is a curriculum framework, not a professional-domain coverage guarantee.
- NIST SP 800-218A detailed practice content has not yet been reviewed; only identity and abstract are verified.
- No independent reviewer has accepted any claim.
- At the initial observation time (2026-11-04), native GitHub Issue Dependency edges were unmaterialized and the LinkML CLI had not been run. Both statements are historical and are superseded by the current-state reconciliation below.

## Method adjustment

Reinspect publisher PDFs for dates, ISBNs, page locators and content before creating canonical metadata. Record PDF page index and printed page number as separate fields when both matter. Check for companion profiles when using a framework as a boundary source. Keep claims atomic and preserve the claim → evidence → locator → source chain without duplicate editable evidence links.


## Static data integrity check

The committed vertical-slice fixture was parsed with PyYAML 6.0.3. The current data has 17 source locators, 12 evidence records, 12 atomic claims and 12 typed support relationships. All locator-to-source, evidence-to-locator, claim-to-source and relationship-to-claim/evidence references resolved against the current registry IDs and the preserved SWEBOK alias. No claim was left without a support relationship. This is a static YAML/reference check, not human acceptance of the claims.


## CS2023 author and SE-Tools locator correction — 2026-11-04

The official CS2023 report identifies Susan **L.** Epstein. The official Body of Knowledge PDF places “SE-Tools: Tools and Environments” at zero-based page index 178 (one-based PDF page 179); index 177 remains in SE-Teamwork. The registry, dossier, discovery report and knowledge locator were corrected together. The earlier index-177 correction was itself off by one and is superseded.


## Current-state reconciliation — 2026-11-05

- Native GitHub Issue Dependency edges are now MATERIALIZED and exact-readback verified: #301←#300; #302←#300; #304←#301/#302; #305←#301/#302/#304; #308←#305.
- The LinkML v1.12.1 L01–L15 CLI and instance-conversion experiment has since been executed in Windows-native Python 3.14.7 under SEK-04 builder PR #310. This does not accept any SEK-03 claim and does not imply LinkML adoption.
- All 12 claims remain PENDING / PENDING_LIMITED_EVIDENCE; all 12 evidence records remain EXTRACTED_PENDING_REVIEW. No claim is accepted corpus knowledge.
- The NIST SP 800-218A detailed-practice limitation remains current: only official identity and abstract are verified; detailed practices are not yet extracted.
- The independent Exact-SHA review of the current PR #303 remains BLOCK on SEK-04 integration findings; no claim-level human review has been recorded.


## Integrated WIP current-state supersession — 2026-11-05

The earlier current-state reconciliation above is retained as a dated execution snapshot. The prior statement that PR #303 remains BLOCK on SEK-04 integration is superseded:

- SEK-04 repair PR #311 passed terminal Exact-SHA R2 and was merged into builder PR #310; PR #310 was then independently reviewed and merged into WIP PR #303.
- The early LinkML PoF outcome is now explicitly `REPEAT_EXPERIMENT` in #305's experiment plan/results. This is not a LinkML production adoption or rejection.
- Current integrated WIP PR #303 HEAD at this checkpoint: `a2536324c0f3711229a7b3257d6f7f6a44e36b50`; it remains DRAFT/WIP and awaits fresh independent Exact-SHA review.
- All 12 claims remain PENDING / PENDING_LIMITED_EVIDENCE; all 12 evidence records remain EXTRACTED_PENDING_REVIEW. No claim is accepted corpus knowledge. This claim-level human-review boundary is unchanged by integration review.
- The NIST SP 800-218A detailed-practice limitation remains current: only official identity and abstract are verified; detailed practices are not yet extracted.
- SEK-05 #308 remains preplanned and not dispatchable until the post-integration PR #303 review and SEK-04 acceptance/dependency gates pass.
