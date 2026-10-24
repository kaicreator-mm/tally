# Tally conversation-to-GitHub evidence persistence protocol (v2 Product intake)

Status: **user-directed requirement (2026-10-24), proposed repository convention; not a frozen ADS normative amendment, nor a Release PASS**.

## Non-negotiable outcome

GitHub is the durable handoff and provenance source for **all substantive Tally development outputs produced in conversations**. A conversation ending, moving to a new chat, or an agent being replaced must not erase decisions, research, review findings, source audits, specs, implementation evidence, task breakdowns, progress, blocker diagnoses, next actions or locally-required agent instructions.

A chat reply is a presentation of an already-written GitHub artifact or an explicit failure report, **not** the only storage location for substantive work.

## For every subsequent substantive Tally conversation

1. **Read GitHub first:** inspect the active parent Issue, effective ADS pin and repository instructions, current Product/L2 state, and the [output ledger](./OUTPUT_LEDGER.md). Reconcile newer Issue/PR facts rather than inheriting stale chat summaries.
2. **Choose an owner before work:** use a specific existing Issue or create an appropriately-scoped Issue for each concern. Use a branch/PR for substantive versioned Markdown/spec/test/code artifacts; Issue comments for narrow decisions, status, evidence and dispatch handoff. One concern, one PR; do not merge a development/version branch into `main` until applicable version gates are met.
3. **Write back before declaring completion:** persist the *complete substantive deliverable*, not merely "done" or a vague chat summary, in GitHub. Include exact source SHA/tag when reviewing code; link reference issues/PRs; report P0/P1/P2, PASS/FAIL/NOT_RUN/UNKNOWN distinctly.
4. **Record decisions accurately:** distinguish USER_APPROVED, PROPOSED, ASSUMPTION, OPEN_QUESTION, and SUPERSEDED. Do not silently convert assistant analysis into an adopted ADS norm, frozen PRD, approved L2 or implementation Task.
5. **Keep the index current:** add a dated entry to [OUTPUT_LEDGER.md](./OUTPUT_LEDGER.md) pointing to the canonical artifact/Issue/PR, relevant Git SHA, review/qualification state, and follow-ups; add an Issue comment on the v2 Product intake [#63](https://github.com/kaicreator-mm/tally/issues/63) for consequential requirements/decisions.
6. **Verify remote persistence:** read back the written GitHub path/Issue or check the PR/commit result. If permission, quota or connectivity prevents the write, explicitly report `NOT_PERSISTED` with the complete text available for later recovery; never imply GitHub persistence or completion without a receipt.
7. **Safe persistence:** do not publish credentials, raw private telemetry, model hidden thoughts, secret test fixtures or sensitive personal data to GitHub. Use source-qualified redacted evidence and link permitted originals.

## User-approved v2 boundaries (2026-10-24)

- `task-meta`: mandatory telemetry ecosystem component when available, but collector/hook/publisher failure **must not block** starting, running, verifying or completing a Task. Telemetry coverage becomes UNKNOWN/partial; never fake zero metrics. Tally consumes the ADS-qualified results **through GitHub**, not through task-meta's private database.
- Execution: Local provider is sufficient for normal operation and real v2 closure; SSH/Docker/runx are optional extensions, not hard requirements.
- GitHub owns native Issue/PR/Review/ADS facts; SQLite may cache/index/prefetch/queue reconciliation and hold legitimate K-owned app state. Stale cached GitHub state cannot release authority gates or count a GitHub write as confirmed.
- Tally UI must support a **simple complete ADS flow**: Product evidence and requirements → PRD / Review / Freeze → L2 / Review / Freeze → Task DAG/Issues → Agent execution → diagnose/repair/independent verification → Version Closure / Release Qualification / Tag.
- v2 is a new TypeScript/Node-target DomainHarness K/S/B/C implementation using existing business semantics from the v1.5.5 source audit. No Go compatibility/bridge. Product/L2 still requires independent freeze.

## Artifacts, status and handoff

- Canonical source audit: [TALLY_V155_SOURCE_FUNCTIONAL_AUDIT_2026-10-24.md](./TALLY_V155_SOURCE_FUNCTIONAL_AUDIT_2026-10-24.md), pinned to `v1.5.5@b5e7b6e38ea38837e3bbcfc46c36586180705de0`.
- Product owner: [Tally #63](https://github.com/kaicreator-mm/tally/issues/63).
- L2 intake: [Tally #65](https://github.com/kaicreator-mm/tally/issues/65).
- GitHub-only task-meta ADS report interchange is a data-source constraint, **not** an instruction that missing telemetry blocks Task execution.
- Further analysis must not be represented as an executed test. This audit is source-static; new v2 DomainHarness E2E remains `NOT_RUN`.

## Resumption checklist

Read this file → OUTPUT_LEDGER → Tally #63 and linked Issues → current repository SHA / branch → compare previous output scope → perform work → write GitHub → verify result → respond with links, completion status and remaining blockers.
