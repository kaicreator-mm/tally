# v1.4 Product Value Experiment — Frozen Task Set (P4-01)

**Status:** FROZEN at P4-01, 2026-09-09, before any arm executes.
**Authority:** `docs/evals/v1.4/PRE_REGISTRATION.md` (A0-05, tagged `eval-v1.4-preregistration`) and PRD §30. This document materializes the preregistration; it does not amend it.
**Manifest:** `docs/evals/v1.4/task_set.json` — SHA-256 `c4050efe1fe1d6017bcf310333698f2dcdf8a7ee76dd86561dad774c9c9f7420`
**Mechanical freeze enforcement:** `test/eval/v14_product` `TestManifestFrozen` (the manifest hash is pinned as a committed Go constant; any byte change to the manifest fails the test).

## 1. What was frozen

12 dogfood tasks — 6 `bug_fix` + 6 `database_migration` — each with deterministic, hash-pinned fixtures, a frozen acceptance contract, verbatim intent text, and a difficulty tier assigned at freeze time.

- PRD §30.2 first-round minimum: 10 tasks, ≥5 per harness type. Frozen at 12/6+6.
- Pre-registration targets 16 "if fixture authoring capacity allows". Capacity this round produced 12 deterministic tasks (each with executable oracle machinery); the count is fixed by this freeze. Growing beyond it is a NEW pre-registered set, not an extension of this one (see replacement policy).

## 2. The frozen tasks

| Task | Harness | Tier | Defect (deterministic) | Baseline / oracle assets |
| --- | --- | --- | --- | --- |
| `bf-01-token-bucket-offbyone` | bug_fix | S | boundary comparison admits one extra request after the bucket empties (from harness corpus TASK-2410) | `testdata/tasks/bf-01-…`, plus pinned `test/fixtures/v14/bugfix/{contracts,facts}/TASK-2410.yaml` |
| `bf-02-logout-session-invalidation` | bug_fix | L | logout does not revoke sessions (from harness corpus TASK-2411) | `testdata/tasks/bf-02-…`, plus pinned `test/fixtures/v14/bugfix/{contracts,facts}/TASK-2411.yaml` |
| `bf-03-pagination-partial-page` | bug_fix | M | floor division drops the final partial page; pages report full size | `testdata/tasks/bf-03-…` |
| `bf-04-date-validation-calendar` | bug_fix | M | flat 1..31 day check ignores month lengths and leap years | `testdata/tasks/bf-04-…` |
| `bf-05-currency-rounding-halfup` | bug_fix | M | truncation toward zero instead of rounding half away from zero | `testdata/tasks/bf-05-…` |
| `bf-06-email-dedupe-casefold` | bug_fix | S | case-sensitive dedupe lets case variants survive | `testdata/tasks/bf-06-…` |
| `db-01-orders-total-backfill` | database_migration | M | `ADD COLUMN … NOT NULL` without `DEFAULT` over seeded rows (materialized P3-02 corpus) | pinned `test/fixtures/v14/migration/**` + `testdata/tasks/db-01-…` |
| `db-02-account-email-unique` | database_migration | M | UNIQUE constraint over seeded duplicate addresses | `testdata/tasks/db-02-…` |
| `db-03-rename-breaks-downstream` | database_migration | L | post-rename revision still references the old column (fails both paths) | `testdata/tasks/db-03-…` |
| `db-04-orders-fk-orphan` | database_migration | M | FOREIGN KEY over seeded orphan reference | `testdata/tasks/db-04-…` |
| `db-05-drop-column-data-loss` | database_migration | L | `DROP COLUMN` destroys live seeded values while the upgrade SQL succeeds | `testdata/tasks/db-05-…` |
| `db-06-total-type-narrowing` | database_migration | M | `BIGINT → INTEGER` narrowing over out-of-range seeded value | `testdata/tasks/db-06-…` |

Every task directory contains: `intent.md` (verbatim intent bytes, identical for every participant), `acceptance.yaml` (frozen acceptance contract), `oracle/oracle.json` (frozen oracle cases and expected outcomes), and baseline fixtures (`baseline/**`). The `bf-01`/`bf-02`/`db-01` tasks additionally pin existing harness fixtures under `test/fixtures/v14/**`, tying the trial set to the real P3 harness corpus.

## 3. Difficulty tiers

Assigned once at freeze by defect locality: **S** = single comparison/logic point in one file; **M** = multi-case boundary, matrix, or data-path reasoning; **L** = cross-cutting semantics (auth surface, cross-revision coupling, expand/contract data preservation). Frozen distribution: S=2, M=7, L=3. Tiers are never re-assigned mid-trial.

## 4. Fixture roles (what reaches the implementing side)

The manifest schema records only `path → sha256`; roles are declared here and are frozen with the set:

- **Agent-facing (baseline):** `intent.md`, `baseline/**`, and the pinned `test/fixtures/v14/bugfix/{contracts,facts}/*.yaml` + `test/fixtures/v14/migration/migrations/*.sql` + `test/fixtures/v14/migration/seed/n_minus_1.sql`.
- **Verification-side only (never shipped to implementing agents):** `oracle/**` (frozen cases, expected outcomes, reference repairs) and the pinned `test/fixtures/v14/migration/repair/0003_orders_total_not_null.sql` for `db-01`, whose repair is supplied by the reference chain, not the candidate.

Repair candidates always exist as separate new content superseding the defect revision append-only (Decision Locks 8/23); defect files are never edited in place.

## 5. Oracles and their limits

- **bug_fix:** the verification harness evaluates frozen input/output cases (`oracle/oracle.json`) against the registered baseline and reference behaviors (`test/eval/v14_product/bugoracle.go`). The frozen expectations assert: the shipped baseline fails exactly the frozen failure set (`bug_repro_failed_before`) and the reference passes every case (stable acceptance, AC2). The baseline `.go` snapshots under `baseline/` are the representative carrier files; the registry functions are their executable oracle mirror.
- **database_migration:** `test/eval/v14_product/sqloracle.go` derives the three `database_migration` policy evidence keys (`fresh_database`, `n_minus_1_upgrade`, `data_integrity`) from the frozen SQL bytes with a deterministic PostgreSQL-dialect static analyzer (supported statement subset documented in that file), for both evidence paths and both candidates. The frozen expectations assert the shipped chain is repairable and the reference repair satisfies the full gate.
- **Limit:** this proves the frozen oracles coherent and deterministic before any arm runs. Live evidence execution — real regression checks and real PostgreSQL fresh/N-1/integrity runs per PRD §24.2 — happens at trial time under P4-03 against the same frozen contracts.

## 6. Arm neutrality (AC3)

No evaluator-facing asset — intent, acceptance, oracle, baseline, or the manifest itself — names or labels a trial condition. The same bytes are served to every participant, and acceptance is judged by the same external verification path for every candidate (a self-declared done is a Claim, never a pass — PRE_REGISTRATION §7). `TestNoArmLabelsInEvaluatorAssets` enforces this mechanically.

## 7. Exclusions

Only the PRE_REGISTRATION §5 exclusions are legal; nothing else, ever:

1. A fixture cannot materialize deterministically in the trial environment (infra defect, recorded with evidence) — excluded from both conditions, reported.
2. Environment outage mid-task (provider down > 30 min, DB unavailable) — task voided in both conditions, logged, not silently re-run.

At freeze **no task is excluded** (all `excluded: null`). No exclusion may be based on outcome, discovered difficulty, or condition-specific failure; a condition-differing exclusion list invalidates the trial.

## 8. Task replacement policy

- Tasks are **never replaced, re-tiered, or added after the first condition starts** on this set (PRD §30.2 不得中途替换; PRE_REGISTRATION §5).
- The freeze is byte-level: the SHA-256 of `docs/evals/v1.4/task_set.json` is `c4050efe1fe1d6017bcf310333698f2dcdf8a7ee76dd86561dad774c9c9f7420`, pinned in `test/eval/v14_product/manifest_test.go` (`frozenManifestHash`). Any manifest byte change after this commit makes `go test ./test/eval/v14_product/...` fail.
- If the manifest hash changes after the trial started, the trial is **invalidated** unless BOTH conditions re-run on the new set AND the first attempt's results are preserved and reported alongside.
- This document records the freeze; amendments to the experiment design are appended to `PRE_REGISTRATION.md` as dated notes that state whether the trial is invalidated. A different task set is a new preregistration (new manifest, new pinned hash, new freeze), never an edit of this one.

## 9. Bare-agent parity (documented here; implemented by P4-02)

How parity is achieved once arms run — not implemented in P4-01:

- **Same runtime / model:** identical coding runtime, adapter and CLI versions, and one model id for every participant. Requested configuration is never conflated with observed execution identity; observed runtime/model facts are recorded per task result row (Decision Lock 15, PRE_REGISTRATION §6).
- **Same repo baseline:** both conditions start from the identical fixture bytes recorded per task in `fixture_hashes`; the pinned hashes are the baseline identity.
- **Same context:** the verbatim `intent.md` bytes are given to every participant; the Tally-side Contract is generated from that same intent.
- **Same budgets:** equal per-task wall-clock ceiling and token/cost accounting window (cost is recorded for transparency, not a primary metric in trial 1).
- **Same judging:** every candidate passes the same external verification harness against the frozen acceptance; environment identity (OS, Go, DB, runtime/CLI, model, date) is recorded on every result row.

## 10. Freeze verification

```bash
go test ./test/eval/v14_product/...   # schema, freeze hash, fixtures, oracles, arm neutrality, policy doc
```

`trial_ref` records commit `3b389d8168ada40948b9c6566a69f85ae395a589` (HEAD at materialization). The operative freeze identity is the manifest hash above; the freeze is completed by the commit that carries the manifest and this file.

## 11. Freeze-owner adjudication (2026-09-10, pre-trial)

**Discovery:** the P4-02 bare-runner self-check (commit `6799b09`, RUN_PROTOCOL §7) found that bf-01's pinned baseline snapshot compiled and behaved as the FIXED boundary — all 6 oracle cases passed, contradicting `expected.baseline_failures` (a do-nothing candidate would have passed bf-01 in both arms). The code comment claimed an off-by-one bug that the `available >= 0` / 1-based-index arithmetic did not actually contain.

**Adjudication (freeze owner, per §8):** the trial had NOT started (no P4-03 condition had run), so no results existed to invalidate. The freeze was superseded by a re-freeze: the bf-01 baseline was repaired to the intended off-by-one defect (`+1`, failing exactly the frozen cases fourth-immediate / fifth-after-refill / capacity-two-third; the other three cases and every other frozen byte — acceptance, oracle, intent, all other tasks — are unchanged).

**Freeze identity change:**

- 2026-09-09 (P4-01 `8b9aad4`): `c4050efe1fe1d6017bcf310333698f2dcdf8a7ee76dd86561dad774c9c9f7420` (superseded, defective baseline)
- 2026-09-10 (this adjudication): `bbf381be07a0859328f8f39a3a5cdef4ce453b6fa05a244163438e2b6bcd4e4d` (operative)

Post-start amendments remain governed by §8 unchanged.
