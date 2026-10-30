# Tally v2 Cross-version Casebook R0

Bounded public fixtures for Tally Issue #143. This dataset evaluates existing #139 lineage, #140 diagnosis, #141 planning and #142 WorkUnit projections; it is not another analyzer or ADS rule set.

## Dataset

`tally-v2-casebook-r0.json` contains eight source-linked cases across DomainHarness v0.7/v0.8/v0.8.1/v0.10/v0.10.2, task-meta v0.4 and Tally v2:

- #530 false-serialization candidate; native dependencies remain unchanged.
- #586 one campaign with checkpoint history, not one feature per comment.
- #1013 integration seam gap and real WorkUnit collision, not version completion.
- #1100 required P0 CAS/security repair, never waived for cost.
- #1264 loaded-but-not-invoked selected behavior; incomplete historical SHA remains not reconstructable.
- #1354 one concern converging through reviews, with scoped Fast Path and active time UNKNOWN.
- task-meta #64 global DAG Freeze vs per-Task currentness and calendar-only observability.
- Tally #86 draft/source progress vs installed K/B/C/Host acceptance.

Every case carries its native Issue and public PR/comment/commit refs, evidence cutoff, permitted planning-time inputs, expected/forbidden decisions, missing evidence and joint projection labels. No private Trace, chain-of-thought or task-meta API/DB data is included.

## Replay

From the repository root on Windows Node 24:

```sh
node --test packages/tally-engineering-business/test/casebook.test.mjs
```

The automatic checks run both modes:

- `RETROSPECTIVE_DIAGNOSIS` evaluates all currently public findings.
- `HISTORICALLY_BLIND` requires complete source/timestamp-qualified planning-time inputs. Expected decisions must be a subset of `permitted_inputs.visibleDecisions`; actual decisions outside that set fail with `leakedDecisions`, even if timestamps look valid. Cases #1264 and task-meta #64 are `NOT_RECONSTRUCTABLE`; a projection observed after the case cutoff is also excluded from predictive counts.

Reports count source-grounded correctness, conflict safety, unknown honesty, false blockers, denominator consistency and next-action usefulness. Naive baseline errors and candidate errors remain counts, not percentages. Active hours and production A/B ROI are `NOT_MEASURED`.

Use `affectedCases(casebook, changedSourceRefs)` to replay only cases linked to changed Issue/PR/commit sources. A failed or missing projection is visible as FAIL/UNKNOWN, never silently treated as PASS. Casebook PASS is fixture conformance only, not TASK_PASS, HIDDEN_PASS, Release Qualification or main-merge authorization.
