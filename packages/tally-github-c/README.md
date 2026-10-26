# Tally v2 GitHub C — first implementation slice

Source owner: [Issue #103](https://github.com/kaicreator-mm/tally/issues/103), Workfront B #88. Pure ESM consumer logic (no network, no GitHub writes, no K/Host control). This is an **actual Tally implementation module**, not a DomainHarness Kernel replacement or a complete selected C package deployment.

The external GitHub Capability transport **must** perform real paginated API reads and immutable source/permission verification. The caller must separately validate report data using the exactly pinned ADS v4.18 `schemas/github/v4.16/report.schema.json` (`ads/github/v5` report); the `validation.passed` input alone has **no cryptographic or authorization force**. This module declines to infer from that field any Task, Review, Release or GitHub write authority. Native snapshot IDs are supplied independently. Never dereference task-meta private manifest URLs.

`projectReport(observation)` returns a bounded source-qualified advisory with an explicit status and `authorized/taskPass/releasePass=false`. `projectReportSet()` detects conflicts where identical repo/Issue/namespace/report_id/revision/kind has different native comment, digest or source. It treats partial pages, revoked grant and stale sources conservatively. Future implementation must use the ADS validator (e.g. pinned schema + JCS digest), the actual GitHub transport and selected DomainHarness C/Host ports; they are distinct tasks.

Tests: `node --test test/*.test.mjs`. Source smoke on non-Node24 is possible but **v2 physical conformance requires Node24 plus exact installed DomainHarness v0.10.2 BUILD/Host consumer**. No end-to-end Task or remote Effect PASS is asserted. Version main integration remains blocked by Product #77, L2 #82 and Tally ADS v4.18 #85 independent acceptance.

`report_kind=correction` is observed but does not automatically revoke any report: correction target/authority, real comment pagination/currentness and provenance validation are a later GitHub transport/reconciliation contract. Treat unprocessed correction as a reason to invalidate stale cache in its owner; do not turn it into a privileged command.
