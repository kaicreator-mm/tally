# Tally v2 durable conversation output ledger

Purpose: **permanent discovery and handoff index**. GitHub-owned documents and Issues are the durable output; chat history is not an authoritative artifact.

Effective process: [GITHUB_OUTPUT_PERSISTENCE_PROTOCOL.md](./GITHUB_OUTPUT_PERSISTENCE_PROTOCOL.md). Product intake: [#63](https://github.com/kaicreator-mm/tally/issues/63).

| Date | Output | Canonical location | Source/reference pin | Status | Follow-up |
|---|---|---|---|---|---|
| 2026-10-24 | Full v1.5.5 source functional audit and 47-feature B/S/K/C migration map; Product constraints, high-ROI additions, ADS UI gaps, task-meta degradation, SQLite cache and closed-loop gates | [Source audit](./TALLY_V155_SOURCE_FUNCTIONAL_AUDIT_2026-10-24.md) | Tally `v1.5.5@b5e7b6e38ea38837e3bbcfc46c36586180705de0` (RQ candidate `9c71fa6f35f08bcc5bda4101263929f70f81202a`) | STATIC_AUDIT; v2 NOT_RUN; Product NOT_FROZEN | Create function-by-function parity/verification matrix; Product review and L2 |
| 2026-10-24 | User-required persistence: every substantive conversation output must be captured in GitHub, with durable handoff index and readback verification | [Persistence protocol](./GITHUB_OUTPUT_PERSISTENCE_PROTOCOL.md) | User Product decisions 2026-10-24; Tally #63 | USER_APPROVED intent; repository convention proposed for review | Apply to each next substantive Tally turn; keep ledger current |
| 2026-10-24 | Cross-project ADS intake to require GitHub persistence for all substantive Web/Agent conversation outputs, expanding existing task-prompt rule #64 | [ADS #1316](https://github.com/kaicreator-mm/ai-development-standard/issues/1316) | ADS previous rule #64; Tally #73, [draft PR #74](https://github.com/kaicreator-mm/tally/pull/74) | PROPOSED STANDARD INTAKE; not adopted ADS norm | Independent ADS Product review; propagate to relevant repos after approval |
| 2026-10-24 | Pre-DomainHarness v0.10.2 delivery lanes, Product-first dependencies, complete simple ADS UI P0, GitHub/SQLite/pure B parallelization and runtime blocking gates | [Tally #75](https://github.com/kaicreator-mm/tally/issues/75) · [#63 Product handoff](https://github.com/kaicreator-mm/tally/issues/63#issuecomment-6380289571) · [#65 L2 handoff](https://github.com/kaicreator-mm/tally/issues/65#issuecomment-6380290766) · [#66 UI priority](https://github.com/kaicreator-mm/tally/issues/66#issuecomment-6380290057) | DH T00 #1319 / A2 #1341 current; task-meta #14; Tally v1.5.5 audit tag pinned | PLANNING_PERSISTED; Product/L2 NOT_FROZEN; DH real E2E NOT_RUN | P01 feature parity → P02 Product PRD → A01 L2, then independent parallel G01/D01/B01/A02/U01/M01; I01 after DH |
| 2026-10-24 | **P0 correction:** Tally v2 is native independently BUILD-selected Domain Packages, **not TS service built first then wrapped**. Prior #75 implementation-early interpretation superseded; before DH v0.10.2 do only audit, Product/semantic L2, specifications/fixtures/wireframes, no production B/C/UI source. Physical Package_READY admission bound to real DH external producer/clean install/RQ. | [#75 exact precedence](https://github.com/kaicreator-mm/tally/issues/75#issuecomment-6380546957) · [#63 Product](https://github.com/kaicreator-mm/tally/issues/63#issuecomment-6380554722) · [#65 L2](https://github.com/kaicreator-mm/tally/issues/65#issuecomment-6380555091) · [#66 UI](https://github.com/kaicreator-mm/tally/issues/66#issuecomment-6380555404) | DomainHarness #1330 external package consumer, #1331 RQ; v0.10.2 source pending | USER_ARCHITECTURE_LOCK_PERSISTED; PRODUCTION_CODE_NOT_STARTED | Complete F01–F47 parity/PRD/UI specs; wait for public Package_READY before executable Tally package DAG |

## Adding the next entry

Add one row with the date, substantive topic, *exact GitHub artifact or Issue/PR link*, source SHA/ADS pin where relevant, honest status (e.g. DRAFT / USER_APPROVED / REVIEW_PENDING / PASS / NOT_RUN / UNKNOWN), and remaining follow-up. Do not count an assistant chat response, local-only file, or unpushed branch as persisted.

For generated specifications/reports, keep full text in `docs/research/v2/` or the applicable product/architecture/docs path, using a version branch and PR. For narrow operational updates, use Issue/PR comments; link them from here.

## Reference artifacts

- [Tally v1.5.5 source baseline](https://github.com/kaicreator-mm/tally/tree/b5e7b6e38ea38837e3bbcfc46c36586180705de0)
- [Tally v2 Product #63](https://github.com/kaicreator-mm/tally/issues/63)
- [Tally v2 L2 #65](https://github.com/kaicreator-mm/tally/issues/65)
- [v1.5.5 Release qualification record](https://github.com/kaicreator-mm/tally/issues/26#issuecomment-5692237137)
