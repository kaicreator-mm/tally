# SEK-00 — Method Observation

**Task:** #300  
**Date:** 2026-11-04  
**Status:** OBSERVED / DRAFT

## Observations

1. The repository has a legacy root knowledge/ store whose admission contract is limited to verified repairs. It explicitly excludes unverified conclusions and analysis. Reusing it for a broad research corpus would mix incompatible trust and lifecycle semantics. The v2.1.1 assets therefore use docs/versions/v2.1.1/.
2. The current default branch is main at exact HEAD e988321439f65b55936a691c02250b6d62a29409. No version/v2 or version/v2.1.1 branch was present in the observed branch listing. A WIP integration branch was created from the exact main SHA; it is not an official version branch or release.
3. Repository .dev-standard/VERSION records ADS 4.18.0. The project override text is scoped to the prior v2 version branch; this draft does not silently extend its runtime-specific rules to the knowledge-only child version.
4. ADS #1359 is an open Delivery-First Product/L2 intake, not a frozen standard. The current Owner instruction is applied as the execution direction; it is not described as a normative ADS adoption.
5. Product PR #157 R9 and semantic L2 PR #158 R7 have independent source PASS, but Product_FROZEN remains NO pending the final Controller checkpoint and L2_FROZEN remains NO pending native Owner Freeze. The v2.1.1 work is documentation/research-only and does not claim either Freeze.
6. Task issues #300–#302 now hold separate bounded write sets. SEK-01 and SEK-02 are independent workfronts after SEK-00. The connected GitHub action set can create/read Issues and comments but does not expose a native Issue Dependency mutation action. Textual dependency references are recorded, but native dependency-edge readback remains NOT_MATERIALIZED. This must not be reported as a materialized live DAG.
7. The initial taxonomy work found that the ten-category hypothesis omits several SWEBOK KAs. This changes the expected discovery work and is a concrete method finding, not a frozen taxonomy decision.

## Negative / unresolved observations

- Native GitHub Issue Dependencies have not been created or read back.
- No formal Product/L2/DAG freeze has occurred.
- No independent review has occurred.
- No LinkML or graph tooling has been selected or executed.
- The WIP branch has no release or main-merge authority.

## Method adjustment

Use the exact main SHA as the WIP ancestry; retain the WIP-only branch name; keep task write sets disjoint; record textual dependency intent honestly; do not dispatch an external task as READY until native dependency facts are materialized and read back.
