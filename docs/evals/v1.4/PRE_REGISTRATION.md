# v1.4 Product Value Experiment — Pre-Registration

**Status:** FROZEN before trial (A0-05, 2026-09-08)
**Authority:** PRD §30-33; Decision Lock 20 (no validation bluffing).
**Trial owner:** P4-03 executes; P4-04 reports. This document may not be edited after trial start except by appending dated amendment notes that explicitly state whether the trial is invalidated.

PRD §30 requires pre-registration to be saved in Git and tagged. This file is that registration; the accompanying `task-set-manifest.schema.json` defines the manifest every task set must satisfy before arms run.

## 1. Research question

Does **Same Coding Agent + Tally Closed Loop** reduce human orchestration labor and time-to-accepted-change relative to **the same agent run bare**, without worsening 14-day rework or escaping quality guardrails?

## 2. Arms (PRD §30.1)

| Arm | Definition |
|---|---|
| **A — Bare** | The coding agent driven directly by a human operator: same intent text, same repo, same model/runtime. Human does all orchestration (copy failures back, rerun verification, restart) as they normally would. |
| **B — Tally** | Same agent runtime inside the Tally closed loop: Intent → Contract → Execute → Verify → Repair(bounded) → Accept/Escalate. Human only receives escalations. |

## 3. Metrics (fixed before trial; PRD §31)

### Primary

| Metric | Definition | Direction |
|---|---|---|
| M1 Hands-off Completion Rate | proportion of tasks reaching `ACCEPTABLE / READY_FOR_HUMAN_DECISION` with zero human orchestration events (manual code fix, failure copy-back, prompt re-issue, verification restart). **Viewing a diff is NOT a failure event.** | higher better |
| M2 Time to Accepted Change | wall-clock from Intent recorded → accepted change (per task; report median and p90 per arm) | lower better |
| M3 Human Interventions per Accepted Change | discrete events: clarification, manual repair, manual rerun, waiver, trade-off decision, forced retry, direct code correction | lower better |

### Guardrails

| Metric | Definition | Trigger |
|---|---|---|
| G1 14-day Rework Rate (fast) | same WorkItem/Change re-modified within 14 days for the same root cause | report; K4 if clearly higher in arm B |
| G2 Slow guardrails | escaped defect, rollback, hotfix, regression, production incident tied to an accepted change | any occurrence logged with root-cause note |

### Kill/Pivot signals (PRD §33, verbatim adoption)

- **K1** M1 persistently low in arm B AND failures dominated by Tally's own process/contract generation/verification noise → pivot.
- **K2** M2 persistently `Tally > bare + human loop` without quality explanation → pivot.
- **K3** M3 not falling after weeks of use → core value unproven.
- **K4** G1 clearly increased by automation → stop pursuing hands-off.
- **K5** users bypass Tally at scale → friction too high.
- **K6** policy noise (per PRD §33 K6) dominating attention → pivot policy defaults.

## 4. Thresholds (pre-committed)

The first trial is sized for trend detection, not significance (PRD §30.2 "达到足以判断趋势的一组固定任务"):

```text
Continue (v1.4 thesis supported):   M1(B) − M1(A) ≥ +15pp  AND  M3(B) < M3(A)  AND  G1(B) ≤ G1(A) + 5pp
Pivot signal:                        any of K1–K6 fires
Inconclusive:                        anything between — rerun with a larger pre-registered set, first results retained
```

## 5. Task set rules (fixed before trial)

### Inclusion (frozen at P4-01)

- Harness types: **Bug Fix** and **Database Migration** only (PRD §30.2 first round; no scope expansion before trial).
- Each task must have: reproducible failing baseline, frozen acceptance (Contract), fixtures hash-pinned in the manifest, and a difficulty tier (S/M/L) assigned at freeze time by the inclusion rule below.
- Minimum first-round size: 10 tasks (≥5 per harness type), target 16 if fixture authoring capacity allows; the count is fixed at P4-01 freeze and published in the manifest before arm A runs.

### Exclusion (pre-declared, the only allowed ones)

- Task fixture cannot materialize deterministically in the trial environment (infra defect, recorded with evidence) — excluded from BOTH arms and reported.
- Environment outage mid-task (provider down > 30 min, DB unavailable) — task voided in BOTH arms, logged, not re-run silently.
- No exclusion based on outcome, difficulty discovered mid-trial, or arm-specific failure. An exclusion list that differs between arms invalidates the trial.

### No post-hoc replacement (AC4)

Tasks are never replaced, re-tiered, or added after arm A starts on the set. Any manifest byte change after freeze (verified by manifest hash) invalidates the trial unless BOTH arms re-run on the new set AND the first attempt's results are preserved and reported alongside (PRD §30.2 不得中途替换).

## 6. Arm/environment/model parity (AC3)

- **Same runtime**: identical coding runtime, adapter version, and CLI version across arms; recorded per task via observed runtime snapshot (Decision Lock 15).
- **Same model**: identical model id for both arms, actual observed model recorded, not requested.
- **Same repo baseline**: both arms start from the same commit of the target task fixture repo; fixture hashes pinned in the manifest.
- **Same context**: identical intent text; arm B's Contract is generated from that intent by Tally; arm A receives the same intent verbatim.
- **Same budgets**: equal per-task wall-clock ceiling and equal token/cost accounting window; measurement records cost per arm for transparency but cost is not a primary metric in trial 1.
- **Environment identity** (recorded in every result row): host OS/arch, Go version, DB version (migration tasks), runtime/CLI version, model id, date.

## 7. Judging and bias controls

- Acceptance in BOTH arms is judged against the frozen Contract acceptance by a verifier path that is not the implementing agent (arm A's acceptance is run through the same verification harness as arm B's — a bare agent's self-declared done is a Claim, never a pass).
- Human review events (diff viewing) are logged but do not count as interventions (PRD §31.1).
- Trial operator may not modify acceptance, fixtures, or thresholds after freeze; all operator actions on trial tasks are journaled.
- Self-evaluation bias (prior adversarial-review finding): no metric in this trial is computed by the implementing agent or by arm-side tooling; all metric computation consumes recorded event logs only.

## 8. Analysis plan

- Per-task paired comparison (same task, both arms); report M1 difference in percentage points, M2/M3 per-arm medians + p90, G1/G2 event tables.
- Every executed command and its `PASS | FAIL | NOT_RUN(reason)` recorded in the trial report (Decision Lock 20).
- First results are published regardless of direction; negative or inconclusive results are never suppressed or recomputed against a changed corpus.

## 9. Freeze

This registration is frozen by Git tag `eval-v1.4-preregistration` at the commit containing this file. P4-01 freezes the task-set manifest (hash recorded in its own committed file) before P4-02/P4-03 execute arms.

## Dated note (2026-09-10, freeze-owner adjudication — pre-trial)

Trial status at amendment: NOT STARTED (no condition of P4-03 had run). The frozen task set was found to contain a defective fixture (bf-01 baseline pinned as fixed code; discovered by the P4-02 self-check before any trial run). Per TASK_SET.md §8 the freeze was superseded by a re-freeze with the bf-01 baseline repaired to the intended defect; acceptance, oracle, intent and all other task bytes are unchanged. **This amendment does NOT invalidate any trial result — none existed.** Freeze hash `c4050efe…` → `bbf381be…` (recorded in TASK_SET.md §11). Metrics, thresholds, exclusions and parity rules of this preregistration are unchanged.
