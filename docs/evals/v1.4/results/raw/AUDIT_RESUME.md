# P4-03 Trial Resume Audit (AUDIT_RESUME)

**Written by:** the resume session for task `docs/tasks/v1.4/P4/P4-03.md`, 2026-09-10 (UTC).
**Why this file exists:** the original P4-03 execution session built both runners and executed most of
both arms over ~6 hours, then died from an infrastructure (model-request) failure **before
materializing raw result rows**. Per the resume discipline, this session audited every arm × task
record first; this document is that audit. No metric computation and no conclusions live here
(P4-04 owns aggregation); only factual per-task state, interruptions and provenance.

**Record locations (never committed):**

- Arm A (bare): `%TEMP%\tally-v14-trial\bare\<task_id>\` — `attempt-N/{attempt.json, codex-stdout.jsonl, last-message.txt, ws/, judge-result.json}` + `events.jsonl`.
- Arm B (tally): `%TEMP%\tally-v14-trial\tally\<task_id>\` — `run-record.json` (loop + embedded judge result) + `events.jsonl` + `candidate-ws/` + `repo/` + `worktrees/` + `verify-worktrees/`.
- Judge identity: arm-neutral `go run ./scripts/v14_trial/bare judge` — the identical command for both arms (RUN_PROTOCOL §3).

**Row outcome vocabulary used in `bare/<task>.json` and `tally/<task>.json`:**
`accepted | rejected | escalated | interrupted | not_run`.
`interrupted` = the arm's trial for the task started but never reached a terminal trial state
(no complete loop verdict / no external judge); it is retained and labeled, never dropped and never
fabricated into a completed shape.

## 1. Per-task audit — Arm A (bare)

Every task: `exit_status=completed` on every attempt, judge `complete=true` on the last attempt,
journal consistent (agent_attempt → verification events, plus operator events where listed).
**Nothing was missing or partial on the bare arm at resume time.**

| Task | Attempts | Last-attempt judge | Outcome (row) | Interventions (journal) |
| --- | --- | --- | --- | --- |
| bf-01-token-bucket-offbyone | 2 | accepted (a1 NOT_ACCEPTED → operator `prompt_reissue` → a2) | accepted | 1 prompt_reissue |
| bf-02-logout-session-invalidation | 1 | accepted | accepted | 0 |
| bf-03-pagination-partial-page | 1 | accepted | accepted | 0 |
| bf-04-date-validation-calendar | 1 | accepted | accepted | 0 |
| bf-05-currency-rounding-halfup | 1 | accepted | accepted | 0 |
| bf-06-email-dedupe-casefold | 2 | accepted (a1 NOT_ACCEPTED → `prompt_reissue` → a2) | accepted | 1 prompt_reissue |
| db-01-orders-total-backfill | 1 | accepted | accepted | 0 |
| db-02-account-email-unique | 1 | accepted | accepted | 0 |
| db-03-rename-breaks-downstream | 1 | accepted | accepted | 0 |
| db-04-orders-fk-orphan | 2 | accepted (a1 NOT_ACCEPTED → `prompt_reissue` → a2) | accepted | 1 prompt_reissue |
| db-05-drop-column-data-loss | 2 | accepted (a1 NOT_ACCEPTED → `prompt_reissue` → a2) | accepted | 1 prompt_reissue |
| db-06-total-type-narrowing | 1 | accepted | accepted | 0 |

Pre-trial environment deviation, preserved untouched (disclosed in `trial-environment.json`
`sandbox_adjudication`): bf-06 arm A was first attempted under the broken `workspace-write`
(codex-cli 0.153.0 enforces it read-only on this host), VOIDED by coordinator adjudication, and
preserved at `bare/superseded-environment-workspace-write-readonly-defect/` (3 attempts + 2
prompt_reissue + ADJUDICATION.md). The row above and the materialized `bare/bf-06-….json` come from
the re-run under the adjudicated `danger-full-access` mode.

## 2. Per-task audit — Arm B (tally)

At resume time: 10 of 12 tasks had a complete `run-record.json` (loop verdict + judge), one task
(db-05) was interrupted mid-loop by the runner's death, one task (db-06) had never started.

| Task | Loop outcome (recorded) | Attempts | Judge (arm-neutral, on last candidate) | State at resume |
| --- | --- | --- | --- | --- |
| bf-01-token-bucket-offbyone | accepted (phase=accepted) | 1 | ACCEPTED | complete |
| bf-02-logout-session-invalidation | accepted | 1 | ACCEPTED | complete |
| bf-03-pagination-partial-page | accepted | 1 | ACCEPTED | complete |
| bf-04-date-validation-calendar | accepted | 1 | ACCEPTED | complete |
| bf-05-currency-rounding-halfup | accepted | 1 | ACCEPTED | complete |
| bf-06-email-dedupe-casefold | accepted | 1 | ACCEPTED | complete |
| db-01-orders-total-backfill | escalated (repair_exhausted, 3 attempts) | 3 | ACCEPTED (re-run; loop verdict governs the row) | complete |
| db-02-account-email-unique | escalated (repair_exhausted, 3 attempts) | 3 | ACCEPTED (re-run; loop verdict governs) | complete |
| db-03-rename-breaks-downstream | escalated (repair_exhausted, 3 attempts) | 3 | ACCEPTED (re-run; loop verdict governs) | complete |
| db-04-orders-fk-orphan | escalated (repair_exhausted, 3 attempts) | 3 | ACCEPTED | complete |
| db-05-drop-column-data-loss | — (no terminal state reached) | 2 inner runs completed | never executed by the runner | **interrupted** (see §3) |
| db-06-total-type-narrowing | escalated (no_candidate, 3 attempts) | 3 | ACCEPTED (on the last delivered candidate; loop verdict governs) | was missing at resume; run fresh by this session (see §4) |

Judge-infrastructure event inside arm B (before this resume): the first judge pass for db-01/db-02/db-03
hit a maintenance-URI query-string bug that made every `database_migration` gate `NOT_RUN`
(incomplete evidence). The superseded incomplete results are preserved per task as
`judge-result-incomplete-dsn-querystring-bug.json`; the runner's `rejudge` path re-ran the same
arm-neutral judge on the untouched candidate workspaces at 23:36:32Z / 23:37:05Z / 23:37:38Z
(2026-09-09) and the run records were updated by that code path only. Loop facts were never touched.
db-04 onward judged directly without the bug.

Escalation detail recorded on every escalated db task: reason `repair_exhausted`
("修复预算拒绝下一次修复尝试" — the bounded repair policy refused the next repair attempt), 3 attempts each,
every attempt `status=completed` with a candidate ChangeSet (append-only lineage in the run records).

## 3. Interruption forensics — db-05 (arm B), recorded as `interrupted`

Journal evidence (`tally/db-05-drop-column-data-loss/events.jsonl`, all times UTC):

- 00:16:49.057Z — inner run `TASK-9110/attempt/001` completed (1 iteration, 470.8 s) with a delivered candidate claim.
- 00:27:05.012Z — inner run `TASK-9110/attempt/002` completed (1 iteration, 615.4 s) with a delivered candidate claim.
- 00:34:32.031Z / 00:37:45.157Z — the implementing agent executed the trial's own arm-neutral judge
  during its attempts (`…\Temp\judge-db05-current.json`, `…\Temp\judge-db05-reworked.json`, both
  preserved; journalled as `verification` events — agent-produced evidence, **not** the trial judge).
- After 00:37:45Z the trial-runner process died (coordinator-confirmed model-request infrastructure
  failure) before: the loop's terminal verdict (attempt 3 or escalation), candidate-workspace
  materialization, and the runner's own judge invocation. No `run-record.json` exists.

Preserved untouched under `tally/db-05-drop-column-data-loss/`: `events.jsonl`, `repo/` (2 commits:
agent base + verification authority), `worktrees/wt-6139054c-7028fd55/` (contains the last attempt's
modified `…/db-05-drop-column-data-loss/baseline/migrations/0003_drop_legacy_token.sql`),
`verify-worktrees/`.

**Why not resumed or re-run:** the tally runner has no loop-resumption support — `runTask` refuses a
task whose fixture repository already exists, and a loop always restarts at attempt 1 (fresh
CodegenRun/ChangeSet discipline). Per the resume discipline ("finish partial attempts ONLY if the
protocol supports resumption; otherwise record them honestly as interrupted rows"), db-05 arm B is
materialized as an `interrupted` row carrying the journal-derived facts above. It was not re-run,
and no terminal verdict was fabricated. (Arm A db-05 is complete and unaffected.)

## 4. Fresh execution by this resume session — db-06 (arm B)

db-06-total-type-narrowing had no directory, no journal and no records at resume time (never
started). Executed fresh by this session with the identical runner and configuration surface as the
rest of arm B (`go run ./scripts/v14_trial/tally run --task db-06-total-type-narrowing`, started
00:49:51Z, finished 01:12:08Z 2026-09-10, model `deepseek-v4-flash` (requested), sandbox
`danger-full-access`, same codex binary 0.153.0, same facts DB container).

Recorded outcome (run record `tally/db-06-total-type-narrowing/run-record.json`):

- Attempt 1: completed, candidate ChangeSet (295.8 s). Attempt 2 (repair): completed, new candidate
  ChangeSet (787.2 s). Attempt 3 (repair): FAILED — inner iterations exhausted; the attempt's
  recorded agent claim shows the provider returned `402 Payment Required: Insufficient Balance`
  (deepseek account credit exhaustion mid-attempt), so no third candidate was produced. The loop's
  typed escalation is `no_candidate` ("尝试结束但没有固定候选 ChangeSet") — the provider error is
  recorded in the attempt record and the claim, never rewritten into a candidate failure (AGENTS §5
  trust invariant).
- External arm-neutral judge on the last delivered candidate (attempt 2, materialized to
  `candidate-ws`): ACCEPTED — `fresh_database` pass, `n_minus_1_upgrade` pass, `data_integrity` pass
  on live PostgreSQL 16.15 (basis in `judge-result.json`).
- Row outcome: `escalated` (the loop verdict governs; the judge verdict is recorded alongside for
  transparency per the runner's documented semantics). Wall 1352.2 s; attempts [295.8, 787.2, 229.3].

## 5. Superseded infra artifacts preserved inside the tally runs root (disclosed, not rows)

- `tally/bf-01-token-bucket-offbyone/not_run-infra/run-record.json` — an `outcome=not_run` row
  ("probe build failed: mkdir …") from 21:17:23Z 2026-09-09, superseded by the clean bf-01 loop
  re-run 5 minutes later; preserved.
- `tally/db-02-account-email-unique/not_run-infra/{repo,verify-worktrees,worktrees}-runner-killed-midloop/`
  — partial composition artifacts of a runner killed mid-loop at ~22:10–22:15Z, superseded by the
  clean db-02 loop re-run; preserved.

These superseded artifacts are provenance, not trial rows; the corresponding task rows come from the
completed re-runs.

## 6. Honest-accounting notes that travel with the rows

- **Observed model:** the codex exec JSONL surface does not self-report the model on any attempt in
  either arm (observed identity records `model_observed=false`, runtime `codex-cli 0.153.0`). The
  requested model id (`deepseek-v4-flash`) is recorded separately in every row and never backfilled
  into observed facts (Decision Lock 15, RUN_PROTOCOL §5).
- **Agent self-judging inside arm B (sandbox deviation consequence):** under the adjudicated
  `danger-full-access` mode, the implementing agents of db-02 (22:51:05Z), db-03 (23:20:52Z,
  23:28:47Z) and db-05 (00:34:32Z, 00:37:45Z) discovered and executed the trial's own judge with
  custom `--out` paths during their attempts. All invocations are journalled; the preserved outputs
  are `%TEMP%\tally-v14-trial\judge-db02-repair1.json`, `judge-db03-repair1.json`,
  `judge-db03-repair1-final.json`, `%TEMP%\judge-db05-current.json`, `%TEMP%\judge-db05-reworked.json`.
  These could not alter the loop verdicts (probe-based, in-runstore) nor the recorded final judge
  results (each task's `judge-result.json` under the tally runs root was produced by the runner's own
  invocation after the loop). The pre-existing `trial-environment.json` disclosure (which named only
  db-02) is generalized by this resume to cover all three tasks.
- **Leftover probe databases:** the `tally-p403-pg` container holds `v14tally_*_{fresh,n1,integrity}`
  databases created by the verification probe's live-DB evidence path during the db-task loops (the
  probe does not drop them) plus `v14tally_manual_*` from a manual probe invocation. The bare judge's
  own `v14trial_*` temp databases are dropped after each judging. The container is removed only at
  the very end of this resume session.
- **Arm A attempt-1 rejections are outcomes, not failures of the trial:** bf-01/bf-06/db-04/db-05
  bare rows record 2 attempts each; attempt 1 was judged NOT_ACCEPTED and the operator's
  `prompt_reissue` is the single journaled intervention. Trial-validity events (the bf-06 void and
  re-run, the rejudge supersessions) are disclosed here and in `trial-environment.json`, never
  hidden behind outcome rewriting.

## 7. Materialization provenance (this session)

Raw rows are assembled by the runner itself (`go run ./scripts/v14_trial/tally rows --out
docs/evals/v1.4/results/raw`), which reads only recorded artifacts — no metric computation. The
resume session extended `rows` in exactly two ways: (1) a task with a journal but no run-record
materializes as an `interrupted` row (db-05) instead of being skipped; (2) tasks with preserved
`not_run-infra/` artifacts get a provenance note.

Exact disposition of every arm × task by this session:

- **Found complete, never re-executed (timing/identity bytes untouched):** bare ×12 (all tasks);
  tally ×10 (bf-01…bf-06, db-01…db-04).
- **Executed fresh by this session:** tally db-06 only (§4).
- **Recorded as interrupted, not re-run:** tally db-05 only (§3).

Final raw row inventory (24 rows + `trial-environment.json` + this audit):
bare 12×`accepted`; tally 6×`accepted`, 5×`escalated` (db-01…db-04 `repair_exhausted`, db-06
`no_candidate`), 1×`interrupted` (db-05). Zero `rejected`, zero `not_run` rows in the materialized
record; every not-executed surface (db-05 terminal state + judge) is retained inside the
`interrupted` row's `not_run_reasons`.
