# SEK-00 — Method Observation

**Task:** #300  
**Date:** 2026-11-04  
**Status:** OBSERVED / CURRENTNESS_RECONCILED / DRAFT

## Observations

1. The repository has a legacy root knowledge/ store whose admission contract is limited to verified repairs. It explicitly excludes unverified conclusions and analysis. Reusing it for a broad research corpus would mix incompatible trust and lifecycle semantics. The v2.1.1 assets therefore use docs/versions/v2.1.1/.
2. The current default branch is main at exact HEAD e988321439f65b55936a691c02250b6d62a29409. No version/v2 or version/v2.1.1 branch was present in the observed branch listing. A WIP integration branch was created from the exact main SHA; it is not an official version branch or release.
3. Repository .dev-standard/VERSION records ADS 4.18.0. The project override text is scoped to the prior v2 version branch; this draft does not silently extend its runtime-specific rules to the knowledge-only child version.
4. ADS #1359 is an open Delivery-First Product/L2 intake, not a frozen standard. The current Owner instruction is applied as the execution direction; it is not described as a normative ADS adoption.
5. Product PR #157 R9 and semantic L2 PR #158 R7 have independent source PASS, but Product_FROZEN remains NO pending the final Controller checkpoint and L2_FROZEN remains NO pending native Owner Freeze. The v2.1.1 work is documentation/research-only and does not claim either Freeze.
6. At the initial observation time (2026-11-04), task issues #300–#302 held the initial bounded write sets and the connected GitHub action surface had not materialized native Issue Dependency edges. This was an initial-state finding only; see the current-state reconciliation below.
7. The initial taxonomy work found that the ten-category hypothesis omits several SWEBOK KAs. This changes the expected discovery work and is a concrete method finding, not a frozen taxonomy decision.

## Historical negative / unresolved observations — initial state on 2026-11-04

- At the initial observation time, native GitHub Issue Dependencies had not been created or read back.
- No formal Product/L2/DAG freeze has occurred.
- At the initial observation time, no independent review had occurred.
- At the initial observation time, no LinkML or graph tooling had been selected or executed.
- The WIP branch has no release or main-merge authority.

## Method adjustment

Use the exact main SHA as the WIP ancestry; retain the WIP-only branch name; keep task write sets disjoint; record textual dependency intent honestly; do not dispatch an external task as READY until native dependency facts are materialized and read back.


## Task materialization update — 2026-11-04

- SEK-03 is tracked by Issue #304 and owns the first source-review dossiers, vertical-slice claims/evidence and its method observation.
- SEK-04 is tracked by Issue #305 and owns the LinkML schema/fixtures/experiment plan/results.
- Their artifact write sets are disjoint. SEK-04 may begin its experiment design using the real sample from SEK-03, while actual LinkML CLI execution remains dependent on a LinkML-enabled builder environment.
- At the initial task-materialization checkpoint, native GitHub Issue dependency edges had not yet been created/read back; this statement is historical and is superseded by the current-state reconciliation below.


## Current-state reconciliation — 2026-11-05

The initial-state observations above are retained as method history and must not be read as current status.

- Native GitHub Issue Dependencies are now MATERIALIZED and exact-readback verified: #301 blocked by #300; #302 blocked by #300; #304 blocked by #301 and #302; #305 blocked by #301, #302 and #304; #308 blocked by #305. The graph is live but not frozen.
- The WIP integration remains `wip/v2.1.1-knowledge`, represented by Draft PR #303 at HEAD `59054299271b1447ceaad212251bc49da0e8dcb3`; it is not an official version branch and has no main/release/deployment authority.
- LinkML/linkml-runtime v1.12.1 L01–L15 has since been executed in Windows-native Python 3.14.7 under builder PR #310. This is experimental evidence only; LinkML remains NOT_ADOPTED.
- The repository-contained checker repair is in PR #311. Its current 32-case source/locator coherence revision awaits exact-HEAD Windows-native validation and terminal independent review.
- An independent Exact-SHA review of the current integration PR #303 exists, but its overall result remains BLOCK because SEK-04 builder/repair integration gates are still open. No task acceptance or product freeze is implied.
