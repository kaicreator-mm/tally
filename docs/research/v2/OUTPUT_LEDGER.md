# Tally v2 durable conversation output ledger

Purpose: **permanent discovery and handoff index**. GitHub-owned documents and Issues are the durable output; chat history is not an authoritative artifact.

Effective process: [GITHUB_OUTPUT_PERSISTENCE_PROTOCOL.md](./GITHUB_OUTPUT_PERSISTENCE_PROTOCOL.md). Product intake: [#63](https://github.com/kaicreator-mm/tally/issues/63).

| Date | Output | Canonical location | Source/reference pin | Status | Follow-up |
|---|---|---|---|---|---|
| 2026-10-24 | Full v1.5.5 source functional audit and 47-feature B/S/K/C migration map; Product constraints, high-ROI additions, ADS UI gaps, task-meta degradation, SQLite cache and closed-loop gates | [Source audit](./TALLY_V155_SOURCE_FUNCTIONAL_AUDIT_2026-10-24.md) | Tally `v1.5.5@b5e7b6e38ea38837e3bbcfc46c36586180705de0` (RQ candidate `9c71fa6f35f08bcc5bda4101263929f70f81202a`) | STATIC_AUDIT; v2 NOT_RUN; Product NOT_FROZEN | Create function-by-function parity/verification matrix; Product review and L2 |
| 2026-10-24 | User-required persistence: every substantive conversation output must be captured in GitHub, with durable handoff index and readback verification | [Persistence protocol](./GITHUB_OUTPUT_PERSISTENCE_PROTOCOL.md) | User Product decisions 2026-10-24; Tally #63 | USER_APPROVED intent; repository convention proposed for review | Apply to each next substantive Tally turn; keep ledger current |

## Adding the next entry

Add one row with the date, substantive topic, *exact GitHub artifact or Issue/PR link*, source SHA/ADS pin where relevant, honest status (e.g. DRAFT / USER_APPROVED / REVIEW_PENDING / PASS / NOT_RUN / UNKNOWN), and remaining follow-up. Do not count an assistant chat response, local-only file, or unpushed branch as persisted.

For generated specifications/reports, keep full text in `docs/research/v2/` or the applicable product/architecture/docs path, using a version branch and PR. For narrow operational updates, use Issue/PR comments; link them from here.

## Reference artifacts

- [Tally v1.5.5 source baseline](https://github.com/kaicreator-mm/tally/tree/b5e7b6e38ea38837e3bbcfc46c36586180705de0)
- [Tally v2 Product #63](https://github.com/kaicreator-mm/tally/issues/63)
- [Tally v2 L2 #65](https://github.com/kaicreator-mm/tally/issues/65)
- [v1.5.5 Release qualification record](https://github.com/kaicreator-mm/tally/issues/26#issuecomment-5692237137)
