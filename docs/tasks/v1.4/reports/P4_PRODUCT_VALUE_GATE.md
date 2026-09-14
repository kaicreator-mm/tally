# P4 Product Value Gate (P4-04)

**Gate question:** did the P4-03 trial produce preregistered product-value metrics and an explicit kill/pivot signal evaluation, computed exactly per the frozen preregistration, without spin or post-hoc redefinition?
**Full derivations:** `docs/evals/v1.4/RESULTS.md` (every number traceable to named raw rows). Task: `docs/tasks/v1.4/P4/P4-04.md`.

## Verdict

```text
GATE VERDICT:        PASS_WITH_FOLLOWUPS   (phase gate: metrics computed per prereg, signals applied, decision delivered)
EXPERIMENT OUTCOME:  INCONCLUSIVE          (prereg §4 Inconclusive branch — the thesis is neither supported nor refuted at n=12)
DECISION:            CONTINUE              (per prereg §4: rerun with a larger pre-registered set; first results retained)
```

`PASS_WITH_FOLLOWUPS` is the *phase* verdict (the analysis task did what its contract requires). It must not be read as "the v1.4 value thesis passed" — the experiment outcome itself is INCONCLUSIVE, and the Continue recommendation is framed for the product owner, who adjudicates.

## Why INCONCLUSIVE (measured facts → signals)

**Facts (24 raw rows, commit `ed1c3ac`; none excluded):**

- Arm A (bare): 12/12 accepted; 4 tasks required one operator `prompt_reissue` each; median time-to-accepted 379.49 s (p90 743.27 s); M1(A) = 8/12 = 66.67%; M3(A) = 4/12 = 0.333.
- Arm B (tally): 6/12 accepted, all with zero recorded human events, median 244.54 s (p90 283.97 s), faster than bare on 5/6 paired bug-fix tasks (paired median −190.68 s); M1(B) row-governed = 50.00%, M3(B) = 0/6 = 0.000 (floor: 5 escalations defer human decisions).
- Arm B database_migration: 0/6 accepted — 5 escalations (walls 1352–1844 s) after **every** internal verify run failed (14/14) while the arm-neutral external judge accepted every last delivered candidate (5/5); 1 interrupted (runner infra death). db-06's final attempt additionally died on provider `402 Insufficient Balance`.
- G1 (14-day rework): **PENDING** — window closes 2026-09-23; reported as a named gap, not 0. G2 (slow guardrails): 0 recorded events in a short non-production window.

**Signals:** K1–K6 evaluated explicitly (RESULTS.md §7): **0 fired**, K3/K4/K5 not evaluable at this sample, K1/K2/K6 carry watch items — above all the migration verification disagreement (agent-claimed probe binding-name mismatch in db-04; recorded as a Claim, corroborated only by the 14/14-vs-5/5 pattern). **Threshold:** M1 reading-dependent (−16.67 pp row-governed vs +16.67/+25.00 pp under §31.1-endpoint readings that count delivered-candidate escalations as `READY_FOR_HUMAN_DECISION`); M3 passes; G1 not computable → Continue condition not evaluable as met → INCONCLUSIVE per the prereg's own branch.

## Follow-ups (required before/with the rerun)

1. **Root-cause the database_migration verification disagreement** (internal verify 0/14 vs external judge 5/5; probe binding-name mismatch claim) and fix or exonerate Tally's verification path. This decides whether the db-half of the trial measured Tally plumbing or candidate quality — the pivotal K1 evidence.
2. **Close the G1 window (2026-09-23)** and append the 14-day rework readout to `RESULTS.md` §5.4.
3. **Rerun on a larger pre-registered set** (new preregistration + manifest; this frozen set is never extended in place), retaining these first results.
4. Add loop resumption or an interrupted-row retry protocol (db-05 consumed a paired sample).
5. Provider-balance operational guard (db-06 attempt-3 402).
6. Rerun under an enforced writable-workspace sandbox when the codex-cli defect is fixed (removes the host-wide-read exposure class that required the `danger-full-access` adjudication).
7. Record observed runtime snapshot fields on bare rows (AC3 evidence symmetry; currently evidenced by identical codex binary path only).

## Acceptance criteria of P4-04

- **AC1** (both arms compared with sample counts and missing data): PASS — RESULTS.md §1–4 (12+12 rows; per-metric n; db-05 interrupted and bf-06 pre-trial void disclosed; zero exclusions).
- **AC2** (hands-off, time, intervention, assurance results explicit): PASS — RESULTS.md §5.1–5.3, §5.5, plus judge-vs-loop assurance facts in §4.
- **AC3** (14-day rework marked pending until window closes): PASS — RESULTS.md §5.4 (named gap; closes 2026-09-23).
- **AC4** (decision CONTINUE/SIMPLIFY/PIVOT/STOP with criteria): PASS — CONTINUE, per prereg §4 Inconclusive branch, with criteria and owner-adjudication framing (RESULTS.md §8).

## Validation

- `git diff --check`: PASS.
- Row-count cross-check (`ls …/raw/{bare,tally} | wc -l`): PASS (12 / 12).
- Deterministic recomputation of every number from the 24 raw rows: PASS (arithmetic embedded in RESULTS.md for hand-checking).
