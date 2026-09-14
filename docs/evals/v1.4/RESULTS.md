# v1.4 Product Value Experiment — Results (P4-04)

**Status:** first results of the P4-03 trial, computed 2026-09-10 by task `docs/tasks/v1.4/P4/P4-04.md`.
**Authority:** `docs/evals/v1.4/PRE_REGISTRATION.md` (frozen, tagged `eval-v1.4-preregistration`) and PRD §30–33. The preregistration is immutable; every metric below is computed exactly as registered there. No metric was redefined, added to the primary set, or dropped after seeing the data.
**Inputs:** the 24 materialized raw rows under `results/raw/{bare,tally}/`, `results/raw/AUDIT_RESUME.md`, `results/raw/trial-environment.json` (commit `ed1c3ac`). Nothing else. Raw rows are read-only; this report only derives numbers from them.
**Derivation discipline:** every number is traceable to named rows; the arithmetic is printed so it can be checked by hand. The derivation re-implements the preregistered arithmetic in tables here; the metric *semantics* (endpoint set, breaking-event vocabulary, named-gap discipline: missing is never 0) follow PRD §31 and the P3-06 projection `internal/measurement/v14_metrics.go` (see §9).

---

## 1. Row inventory and cross-check

| Check | Command | Result |
| --- | --- | --- |
| Arm A rows | `ls docs/evals/v1.4/results/raw/bare \| wc -l` | 12 |
| Arm B rows | `ls docs/evals/v1.4/results/raw/tally \| wc -l` | 12 |
| Top-level raw entries | `ls docs/evals/v1.4/results/raw \| wc -l` | 4 (`bare/`, `tally/`, `AUDIT_RESUME.md`, `trial-environment.json`) |

Outcome census (rows, not tasks):

- Arm A (bare): 12 × `accepted`. Zero `rejected`, zero `escalated`, zero `interrupted`, zero `not_run`.
- Arm B (tally): 6 × `accepted` (bf-01…bf-06), 5 × `escalated` (db-01…db-04 `repair_exhausted`; db-06 `no_candidate`), 1 × `interrupted` (db-05; runner died mid-loop, no terminal state, no judge — AUDIT_RESUME §3). Zero `rejected`, zero `not_run`.

## 2. Sample, exclusions, missing data

**Frozen set:** 12 tasks (6 `bug_fix` + 6 `database_migration`), tiers S=2/M=7/L=3 (`TASK_SET.md` §2–3). Prereg minimum (≥10, ≥5 per harness) met.

**Exclusions applied: none.** The prereg (§5) allows exactly two exclusions (fixture cannot materialize; environment outage mid-task voided in BOTH arms). Neither was applied; every one of the 24 rows is retained in its arm's denominators. The exclusion lists are identical (empty) in both arms, so the prereg's trial-invalidation clause is not triggered.

Disclosed events that are NOT exclusions:

- **bf-06 arm-A pre-trial void:** 3 attempts under the broken `workspace-write` sandbox were VOIDED by coordinator adjudication *before* the trial proper and preserved untouched; both arms re-ran the full set under the adjudicated `danger-full-access` mode (trial-environment.json `sandbox_adjudication`). The materialized bf-06 arm-A row is from the corrected environment.
- **db-05 arm B `interrupted`:** the trial runner died (model-request infrastructure failure) after 2 completed inner runs and before any terminal state. Per the resume discipline it is retained and labeled, never dropped and never fabricated into a completed shape. It counts as **not completed** in every arm-B denominator below and as **missing data** in every per-metric n. Arm A's db-05 row is complete (accepted) and unaffected.
- **db-06 arm B:** executed fresh by the resume session with the identical runner/configuration (AUDIT_RESUME §4); attempt 3 failed with provider `402 Payment Required: Insufficient Balance` (recorded in the attempt and claim, never rewritten into a candidate failure).

**Per-metric sample sizes are stated with each metric.** No row was dropped anywhere.

## 3. Environment identity (prereg §6)

| Dimension | Value | Evidence |
| --- | --- | --- |
| Host OS / arch | Windows, amd64 | `identity.host_os/host_arch`, all 24 rows |
| Go | go1.27.1 | `identity.go_version`, all 24 rows |
| Coding runtime | codex-cli 0.153.0, adapter `1` | observed snapshot in arm-B row identity (`runtime_name/adapter_version/cli_version`); arm-A rows record only the binary path — see gap note below |
| Codex binary | identical across all 24 rows (normalized path `…\OpenAI\Codex\bin\9ba750cce02d5e5c\codex.exe`) | `requested.codex_bin` (A) / `identity.binary_path` (B), verified by normalized-path comparison |
| Model | requested `deepseek-v4-flash`, both arms. **Observed model not self-reported** by the codex exec JSONL surface (`model_observed=false` + `model_note`, every arm-B row; AUDIT_RESUME §6); never backfilled into observed facts | Decision Lock 15 |
| Database | PostgreSQL 16.15 (Debian 16.15-1.pgdg13+2), recorded in `judge.migration.db_version` of every db-task judge result, both arms | judge sections of db rows |
| Sandbox | `danger-full-access`, identical both arms (adjudicated deviation, disclosed; agents had host-wide read incl. verification-side assets) | trial-environment.json |
| Manifest | `bbf381be…` (operative re-freeze) in every judge result and every arm-B row; verified: 0 mismatches across 24 rows | row fields |
| Acceptance parity | `acceptance_sha256` equal across arms for all 11 tasks where both judge rows exist; arm-B db-05 has no judge row (interrupted) | judge sections |
| Judge | arm-neutral `go run ./scripts/v14_trial/bare judge` — identical command both arms | trial-environment.json, RUN_PROTOCOL §3 |
| Dates | Arm A: 2026-09-09 19:23–21:07Z. Arm B: bf-01…db-04 2026-09-09 21:22Z → 2026-09-10 00:08Z; db-05 interrupted 2026-09-10 00:16–00:37Z; db-06 fresh 2026-09-10 00:49–01:12Z | row timing |

Identity-recording gap (honest disclosure): arm-A rows carry `identity{host_os, host_arch, go_version, trial_date}` + `requested.codex_bin` but lack the observed runtime snapshot fields (`runtime_name/adapter_version/cli_version/model_observed`) that arm-B rows carry. Arm-A runtime parity is therefore evidenced by the identical binary path, not by a recorded CLI-version field. Follow-up: record the same snapshot fields on bare rows (RUN_PROTOCOL/runner change, not owned by this task).

## 4. Per-task table (all 24 rows)

Walls in seconds, exactly as recorded in `timing.wall_seconds`. "hum. ev." = recorded human-action events (prompt re-issue etc.; the `escalation` entries in arm-B rows are system→human handoffs, not human actions — see §5.4). "judge" = the arm-neutral external judge on the last delivered candidate.

| Task | Tier | Arm A outcome | A att | A wall (s) | A hum. ev. | Arm B outcome | B att | B wall (s) | B hum. ev. | B judge |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| bf-01-token-bucket-offbyone | S | accepted | 2 | 559.9533 | 1 prompt_reissue | accepted | 1 | 166.9376 | 0 | ACCEPTED |
| bf-02-logout-session-invalidation | L | accepted | 1 | 261.3119 | 0 | accepted | 1 | 307.2788 | 0 | ACCEPTED |
| bf-03-pagination-partial-page | M | accepted | 1 | 418.2208 | 0 | accepted | 1 | 249.3683 | 0 | ACCEPTED |
| bf-04-date-validation-calendar | M | accepted | 1 | 378.5745 | 0 | accepted | 1 | 166.0656 | 0 | ACCEPTED |
| bf-05-currency-rounding-halfup | M | accepted | 1 | 263.2973 | 0 | accepted | 1 | 239.7151 | 0 | ACCEPTED |
| bf-06-email-dedupe-casefold | S | accepted | 2 | 761.5329 | 1 prompt_reissue | accepted | 1 | 260.6665 | 0 | ACCEPTED |
| db-01-orders-total-backfill | M | accepted | 1 | 248.6655 | 0 | escalated (repair_exhausted) | 3 | 1600.5384 | 0 (1 escalation) | ACCEPTED |
| db-02-account-email-unique | M | accepted | 1 | 355.4359 | 0 | escalated (repair_exhausted) | 3 | 1579.5127 | 0 (1 escalation) | ACCEPTED |
| db-03-rename-breaks-downstream | L | accepted | 1 | 380.4106 | 0 | escalated (repair_exhausted) | 3 | 1735.0815 | 0 (1 escalation) | ACCEPTED |
| db-04-orders-fk-orphan | M | accepted | 2 | 578.8907 | 1 prompt_reissue | escalated (repair_exhausted) | 3 | 1843.7573 | 0 (1 escalation) | ACCEPTED |
| db-05-drop-column-data-loss | L | accepted | 2 | 1315.9728 | 1 prompt_reissue | **interrupted** | 2 | n/a (journal span 00:16:49→00:37:45Z; inner runs 470.8 + 615.4) | 0 | never executed |
| db-06-total-type-narrowing | M | accepted | 1 | 281.9250 | 0 | escalated (no_candidate) | 3 | 1352.1592 | 0 (1 escalation) | ACCEPTED (on attempt-2 candidate) |

Recorded verification disagreement (rows, verbatim facts): on all 6 arm-B `bug_fix` tasks the loop's internal verification is `verified` (verify runs 1–6). On all 5 escalated `database_migration` tasks **every** internal verify run is `failed` (db-01: runs 7,8,9; db-02: 11,12,13; db-03: 14,15,16; db-04: 17,18,19; db-06: 22,23 — 14/14 failed on bindings `fresh_database`/`n_minus_1_upgrade`/`data_integrity`) while the external arm-neutral judge **passed the same three gates** on the last delivered candidate of all 5 (5/5). db-04's last-attempt agent claim attributes the internal FAILs to a probe binding-name mismatch (`v14tallyprobe` implements `fresh`/`n1`/`integrity`; the loop invokes `fresh_database`/`n_minus_1_upgrade`/`data_integrity`). That attribution is an **agent Claim, not a Delivery Fact** — the rows do not contain the probe stderr — but the 14/14-vs-5/5 pattern is a recorded fact that section 7 depends on.

## 5. Metric derivations

### 5.1 M1 — Hands-off Completion Rate (prereg §3 primary; PRD §31.1)

Definition (prereg, verbatim): proportion of tasks reaching `ACCEPTABLE / READY_FOR_HUMAN_DECISION` with zero human orchestration events (**manual code fix, failure copy-back, prompt re-issue, verification restart**). Viewing a diff is NOT a failure event.

Typed breaking set used here (prereg M1 named events ∪ P3-06 `v14BreaksHandsOff`): `manual_repair`, `failure_copyback`, `prompt_reissue`, `verification_restart`, `manual_rerun`, `forced_retry`, `direct_code_correction`. Non-breaking per §31.1 (decision, not hand work): waiver, trade-off decision, clarification, escalation itself. `prompt_reissue` **is** an intervention and breaks hands-off in both vocabularies (trial note text: "operator re-issued the same verbatim frozen intent after the previous attempt was not accepted").

**Arm A.** Endpoint reached (accepted): 12/12 rows. Breaking events: `prompt_reissue` on bf-01, bf-06, db-04, db-05 (1 event each; see §4).
→ **M1(A) = (12 − 4) / 12 = 8/12 = 66.67%.** This reading does not depend on the endpoint interpretation below (all 12 arm-A rows are `accepted`).

**Arm B.** Endpoint reading is the one place the prereg text and the row vocabulary can diverge, because the trial's governing rule is "row outcomes are loop-governed for arm B with the external judge verdict recorded alongside; disagreements are retained, never resolved in favor of either signal" (trial-environment.json). All three readings, with exact row composition:

| Reading | Endpoint composition | Tasks | M1(B) | M1(B) − M1(A) |
| --- | --- | --- | --- | --- |
| R1 — row-governed (primary) | only rows with `outcome=accepted`, zero breaking events (bf-01…bf-06) | 6/12 | **50.00%** | 50.00 − 66.67 = **−16.67 pp** |
| R2 — §31.1 endpoint incl. `READY_FOR_HUMAN_DECISION`, conservative | R1 + the 4 `repair_exhausted` escalations that delivered a candidate judged ACCEPTED (db-01…db-04); `no_candidate` db-06 excluded | 10/12 | **83.33%** | **+16.67 pp** |
| R3 — §31.1 endpoint incl. `READY_FOR_HUMAN_DECISION`, full | R2 + db-06 (its delivered attempt-2 candidate is also judged ACCEPTED; the loop escalated only because attempt 3 produced no candidate) | 11/12 | **91.67%** | **+25.00 pp** |

Basis for counting escalations under R2/R3: §31.1 names `READY_FOR_HUMAN_DECISION` as a legal terminal state; db-01…db-04/db-06 escalations each ended with a frozen, materialized candidate awaiting a human decision, with **zero** recorded human orchestration events during the loop (the loop does the failure copy-back internally — that is the product thesis under test). The human decision at escalation is a decision, not one of the four named hand-work events (same distinction as `v14_metrics.go` `v14BreaksHandsOff`: "waiver / trade-off 不打破 — §31.1 明说 READY_FOR_HUMAN_DECISION 是合法终点"). db-05 (`interrupted`) is in the denominator and in the numerator of none: it reached no terminal state.

**The trial record cannot adjudicate between R1 and R2/R3** — the loop-vs-judge disagreement is retained by design, not resolved. So the honest M1(B) is the range **[50.00%, 91.67%]**, and M1(B) − M1(A) is **[−16.67 pp, +25.00 pp]**. Both bounds are carried into §6. n=12 per arm; no rows excluded.

### 5.2 M2 — Time to Accepted Change (prereg §3; PRD §31.2)

Wall-clock from intent recorded → accepted change, per task; median and p90 per arm. Values are the rows' `timing.wall_seconds`. Timing semantics (disclosed): arm-A wall = sum of its attempt walls (the operator's inter-attempt re-issue latency is *not* journaled in the rows, so arm-A times slightly understate true human-in-loop elapsed time); arm-B wall = loop wall including in-loop verification/repair (post-loop judge excluded). Percentiles: linear interpolation between order statistics (type 7; index = 0.9·(n−1)).

**Arm A — n=12** (all rows accepted; sorted, s):
248.6655, 261.3119, 263.2973, 281.9250, 355.4359, 378.5745, 380.4106, 418.2208, 559.9533, 578.8907, 761.5329, 1315.9728

- Median = (378.5745 + 380.4106) / 2 = **379.49 s** (6.32 min)
- p90 = 578.8907 + 0.9 × (761.5329 − 578.8907) = 578.8907 + 164.3779 = **743.27 s** (12.39 min)

**Arm B — n=6 (row-governed primary sample: the 6 accepted rows only;** the 5 escalated rows never reached an accepted change under the governing verdict and db-05 is interrupted, so they are outside the metric's definition, not silently dropped):
166.0656, 166.9376, 239.7151, 249.3683, 260.6665, 307.2788

- Median = (239.7151 + 249.3683) / 2 = **244.54 s** (4.08 min)
- p90 = 260.6665 + 0.5 × (307.2788 − 260.6665) = **283.97 s** (4.73 min)

**Survivorship caveat (mandatory reading for M2):** arm B's M2 sample is exactly its 6 bug-fix tasks. Its 6 database-migration tasks consumed 1352.16–1843.76 s each and produced **no accepted change**. A median over only-accepted tasks is structurally biased in arm B's favor; the like-for-like statement is the paired view, not the two medians.

**Paired per-task deltas (both arms accepted; tally − bare, s):**

| Task | bare | tally | delta |
| --- | --- | --- | --- |
| bf-01 | 559.9533 | 166.9376 | −393.0157 |
| bf-02 | 261.3119 | 307.2788 | +45.9669 |
| bf-03 | 418.2208 | 249.3683 | −168.8525 |
| bf-04 | 378.5745 | 166.0656 | −212.5089 |
| bf-05 | 263.2973 | 239.7151 | −23.5822 |
| bf-06 | 761.5329 | 260.6665 | −500.8664 |

Sorted deltas: −500.8664, −393.0157, −212.5089, −168.8525, −23.5822, +45.9669. Paired median = (−212.5089 + −168.8525)/2 = **−190.68 s** (tally faster); tally faster on **5 of 6** paired tasks.

**Paired db facts (arm B never accepted):** bare accepted in 248.67–1315.97 s; arm B spent 1352.16–1843.76 s across 3 attempts each and escalated (db-05: interrupted). No arm-B database_migration task produced an accepted change.

**Sensitivity (judge-governed, n=11:** 6 accepted + 5 escalated rows whose last delivered candidate is judged ACCEPTED**):** sorted: 166.0656, 166.9376, 239.7151, 249.3683, 260.6665, 307.2788, 1352.1592, 1579.5127, 1600.5384, 1735.0815, 1843.7573 → median = 307.28 s; p90 (index 9.0) = 1735.08 s. Reported for transparency; the governing primary stays row-governed.

### 5.3 M3 — Human Interventions per Accepted Change (prereg §3; PRD §31.3)

Discrete human-action events recorded in the rows: arm A `prompt_reissue` ×4 (bf-01, bf-06, db-04, db-05); arm B **zero** human-action events. Counting rules (stated, not retro-fitted):

- `prompt_reissue` is an intervention (prereg M1 names it; §31.3's closest typed kind is forced retry — the operator forces another agent attempt by re-issuing the intent). Counted once per occurrence.
- The `escalation` entries in arm-B rows are the **system** handing a decision to the human; no human action occurred inside the trial window, so they are not §31.3 events. They are reported below as deferred human attention, because 5 of 12 arm-B tasks ended with "human must now decide".
- The arm-neutral judge is trial measurement machinery, run identically for both arms; running it is not an arm intervention (otherwise it would add 1 to every task in both arms and cancel). The db-01…db-03 `rejudge` after the judge DSN-querystring bug is trial-infrastructure repair (disclosed, AUDIT_RESUME §2), same character.

| Arm | Interventions | Accepted changes (denominator) | M3 |
| --- | --- | --- | --- |
| A (bare) | 4 (`prompt_reissue`) | 12 | **0.333** |
| B (tally) | 0 | 6 (row-governed; 11 under the judge-governed sensitivity — same result) | **0.000** |

M3(B) < M3(A) on the recorded window. **Floor caveat:** arm B's 0 is a lower bound on human attention; its 5 escalations (0.42 per task) each require a future human decision + diff review that the trial window did not record. Arm A's rejections, by contrast, were resolved by the operator's single re-issue inside the window. Per-kind breakdown: arm A {prompt_reissue: 4}; arm B {} — all seven §31.3 kinds otherwise 0-recorded, and per the P3-06 named-gap discipline a recorded 0 in a 12-task single-day trial is a sample-size fact, not evidence of a stable rate (K3 needs weeks of use).

### 5.4 G1 — 14-day Rework Rate (fast guardrail; PRD §31.4)

**PENDING — not yet computable; reported as a named gap, not 0.** All 18 accepted changes (12 arm A + 6 arm B) are dated 2026-09-09 (UTC). The 14-day window per `V14ReworkWindow` closes **2026-09-23 (UTC)**. No re-modification of any accepted change has been recorded in-trial, but the preregistered metric requires the full window. No rework event may be inferred or fabricated before then. When the window closes, append the readout to this file (same derivations: re-modified accepted changes / total accepted changes per arm).

Attribution note carried from P3-06 (`V14ReworkAttributionNote`): the readout counts any post-acceptance re-modification; "same root cause" is not verifiable from the event facts, so the rate is an attribution upper bound.

### 5.5 G2 — Slow guardrails (PRD §31.5)

**Zero recorded events** in either arm of: escaped defect, rollback, hotfix, regression, production incident tied to an accepted change. Honest scope statement: nothing reached production in this trial; the observation surface is the trial itself over ~6 hours, so this is "no events recorded in a short, non-production window", not evidence of safety. Integrity disclosures that travel with these numbers (not guardrail events): the `danger-full-access` sandbox deviation (host-wide agent read in both arms), the five agent-executed judge invocations during attempts (db-02, db-03 ×2, db-05 ×2 — journalled, preserved, could not alter loop verdicts or recorded final judge results), the judge DSN-querystring bug and rejudge for db-01…db-03, and the leftover probe databases (AUDIT_RESUME §6).

## 6. Pre-registered thresholds (prereg §4)

```text
Continue:  M1(B) − M1(A) ≥ +15pp  AND  M3(B) < M3(A)  AND  G1(B) ≤ G1(A) + 5pp
Pivot:     any of K1–K6 fires
Inconclusive: anything between — rerun with a larger pre-registered set, first results retained
```

| Component | Evaluation | Result |
| --- | --- | --- |
| M1(B) − M1(A) ≥ +15 pp | −16.67 pp (R1 row-governed) / +16.67 pp (R2) / +25.00 pp (R3) | **reading-dependent**: passes only under the §31.1-endpoint readings; fails under the row-governed reading |
| M3(B) < M3(A) | 0.000 < 0.333 | passes (floor caveat: 5 deferred escalation decisions) |
| G1(B) ≤ G1(A) + 5 pp | window open until 2026-09-23 | **not computable** |

**Outcome: INCONCLUSIVE.** No kill/pivot signal fired (§7), and the Continue condition cannot be evaluated as met: one component is reading-dependent and one is not computable. Per the prereg, Inconclusive prescribes: **rerun with a larger pre-registered set; these first results are retained** (this file + the raw rows).

## 7. Kill / Pivot signals (PRD §33 via prereg K1–K6)

| Signal | Verdict at this trial | Evidence |
| --- | --- | --- |
| **K1** M1 persistently low in B AND failures dominated by Tally's own process/contract/verification noise → pivot | **NOT FIRED** — but the noise clause has material supporting evidence | "Persistently" cannot be established by one trial at n=12. Failure attribution: of arm B's 6 non-accepted tasks, 5 escalated with **all 14 internal verify runs failed while the external judge passed the same gates 5/5** (§4), and db-04's recorded claim attributes the internal FAILs to a probe binding-name mismatch (a Claim — probe stderr is not in the rows); 1 task was interrupted by runner infrastructure death; db-06's final attempt died on provider 402. Candidate *quality* failure is not established as the dominant cause; Tally-side verification-path noise is implicated but not proven. This is the single highest-priority investigation before any rerun. |
| **K2** M2 persistently Tally > bare + human loop without quality explanation → pivot | **NOT FIRED** (and direction currently opposite on comparable tasks) | Paired both-accepted tasks: tally faster on 5/6, paired median −190.68 s. Counter-fact: on all 6 database_migration tasks arm B consumed 1352–1844 s with no accepted change vs arm A's 249–1316 s. "Persistently" unmet at n=12; the db-task wall consumption must be explained (likely same root as K1's verification disagreement: 3 attempts × ~10 min each) |
| **K3** M3 not falling after weeks of use → core value unproven | **NOT EVALUABLE** | Single-shot trial; K3 requires weeks of usage telemetry. Recorded-window M3(B)=0 is a floor with 5 deferred decisions (§5.3) |
| **K4** G1 clearly increased by automation → stop pursuing hands-off | **NOT EVALUABLE** | G1 window open until 2026-09-23 (§5.4) |
| **K5** users bypass Tally at scale → friction too high | **NOT EVALUABLE** | No real users; one operator-run trial |
| **K6** policy noise (blocks/waivers/false positives) dominating attention → pivot policy defaults | **NOT FIRED** (0 events) | No policy blocks, waivers, or risk-signal false positives recorded in any row (arm-B compositions show matched policies, `confidence: full`, no unmatched risk signals). n too small to generalize; K6 remains a watch item |
| (PRD §33 also defines K7 — assurance-cost imbalance; the prereg's adopted signal set is K1–K6) | NOT EVALUABLE | Cost was recorded "for transparency" but no cost values exist in the raw rows |

Signal set summary: **0 fired, 3 not evaluable at this sample (K3/K4/K5), 3 not fired with watch items (K1/K2/K6).** The signal set is therefore inconclusive at n=12; the prereg's prescribed next step (§4 Inconclusive branch) is a larger pre-registered rerun with these first results retained.

## 8. Decision

**Measured facts first (no interpretation):**

1. Arm A: 12/12 accepted; 4 tasks needed one operator prompt re-issue each; walls 248.67–1315.97 s (median 379.49 s).
2. Arm B: 6/12 accepted, all with zero recorded human events, walls 166.07–307.28 s (median 244.54 s), faster than arm A on 5/6 paired bug-fix tasks.
3. Arm B: 0/6 database_migration tasks accepted; 5 escalated after every internal verify run failed (14/14) while the external judge accepted every last delivered candidate (5/5); 1 interrupted by runner death. Escalated walls 1352.16–1843.76 s. db-06's last attempt additionally died on provider 402.
4. G1 is not computable until 2026-09-23; G2 recorded 0 events in a short non-production window.

**Signal evaluation:** no preregistered kill/pivot signal fired; the Continue condition is partially met (M3 yes; M1 reading-dependent; G1 pending) → **INCONCLUSIVE** per prereg §4.

**Recommendation (for the product owner; final adjudication is theirs): CONTINUE** — continue v1.4 on the prereg's Inconclusive branch: rerun with a larger pre-registered set after the named fixes below, retaining these first results. Neither SIMPLIFY (no prereg mechanism selects it), PIVOT (K1's "persistently" and its dominance clause unproven at n=12), nor STOP (no K signal fired) is supported by this data. The single most decision-relevant open item is the database_migration verification disagreement (14/14 internal fail vs 5/5 external accept): if it confirms as Tally-side verification plumbing (as db-04's claim asserts), the db-half of this trial measured Tally's verification path, not the agent's candidates, and a fixed rerun is the honest successor; if it confirms as genuine candidate failure, K1 evidence strengthens materially.

Pre-rerun follow-ups (owned by their respective tasks, not by this analysis):

1. Root-cause the migration verification disagreement (probe binding-name mismatch claim in db-04; probe stderr not in rows) and fix or exonerate before rerunning the db harness.
2. Close the G1 window 2026-09-23 and append the 14-day rework readout to §5.4.
3. Rerun per prereg §4 with a larger pre-registered set (a new preregistration/manifest; this set is never extended in place).
4. Add loop resumption or an interrupted-row retry protocol so runner deaths stop consuming paired samples (db-05).
5. Provider-balance operational guard (db-06 attempt-3 402 mid-attempt).
6. Sandbox: rerun under an enforced writable-workspace sandbox if the codex-cli read-only defect is fixed, removing the host-wide-read exposure class (incl. verification-side assets) that required the adjudication.
7. Record observed runtime snapshot fields (`runtime_name/adapter_version/cli_version/model_observed`) on bare rows for full AC3 evidence symmetry.

## 9. Semantics alignment with `internal/measurement/v14_metrics.go` (P3-06)

The arithmetic here is re-derived from the raw rows (allowed by the task); the semantics deliberately match the P3-06 projection:

- **Breaking set:** prereg M1's four named events ∪ `v14BreaksHandsOff` (manual_repair/manual_rerun/forced_retry/direct_code_correction break; clarification/waiver/trade-off do not; diff review is never an event). The trial's `prompt_reissue` is in both.
- **Escalation ≠ intervention:** §31.1 makes `READY_FOR_HUMAN_DECISION` a legal endpoint; a human decision at escalation is not hand work — same distinction as the P3-06 comment on waiver/trade-off.
- **Named-gap discipline:** G1 pending is "explicit unavailable", not 0; arm-B db-05 is retained and labeled, never dropped; db-06's provider 402 stays a provider fact, never rewritten into a candidate failure; the agent's probe-mismatch root-cause stays a Claim, never promoted to Evidence.
- **Sufficiency:** as in P3-06, small-sample readouts are marked as such and never used to declare threshold victory — hence §6's INCONCLUSIVE rather than a pass declared on R2/R3.

## 10. Validation (Decision Lock 20 — every command with its real outcome)

| Command | Outcome |
| --- | --- |
| `ls docs/evals/v1.4/results/raw/bare \| wc -l` | PASS (12) |
| `ls docs/evals/v1.4/results/raw/tally \| wc -l` | PASS (12) |
| Deterministic recomputation of §4–§7 from the 24 rows (scripted extraction of `outcome/attempts/interventions/timing/judge`, medians/p90/deltas re-derived; all values re-issued twice, once from full-precision JSON, once from the tables above) | PASS (all values reproduced) |
| `git diff --check` (run on the working tree containing this report) | PASS (whitespace errors: none) |
| `go test ./...` | NOT_RUN(no production source changed; this task is analysis-only, allowed paths are the two report files) |
