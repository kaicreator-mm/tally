# A0-03 — Assurance L1/L2/L3 Bake-off findings

**Date:** 2026-09-08
**Task:** `docs/tasks/v1.4/A0/A0-03.md`
**Harness:** `test/eval/v14_assurance/**` (spike; no production selector changed)
**Artifacts:** `A0-03-results.jsonl` (36 条逐结果记录), `A0-03-summary.json`

## Corpus freeze record (AC1)

| Version | SHA-256 | Status |
|---|---|---|
| v1 | `bf4cae9d1896763b007e1478f3b7ffc051506db7b2af75d124bf74ae2d7cab00` | superseded by amendment |
| v2 (frozen) | `e83b8bd3c7c2c61954dc8898ef1c08cc71358e273bfa56a9a50a6534cdf5f51c` | current |

**Amendment v1→v2 (disclosed, before any layer-comparison conclusion was consumed):**
`TestHeldOutOnlyDefectsExist` (a corpus-design sanity test) found that t3-dedupe's visible case `"b,a,b"` already caught every designed mutant, so the task had no held-out-only defect — a corpus design error, not a result. Fix: move the non-adjacent-repeat case into held-out; visible becomes a no-duplicate sample. The v1 result artifacts were deleted, never used for any mapping decision. `TestCorpusFrozen` mechanically enforces the v2 hash; any further corpus change fails the suite.

Corpus: 6 tasks × 2 mutants = 12 single-defect candidates, each task containing at least one held-out-only defect (verified by test). Frozen before results inspection.

## Results (deterministic layers, executed 2026-09-08)

| Layer | Definition | Caught | Unique catches |
|---|---|---|---|
| L1 | deterministic visible checks | 5/12 | — |
| L2 | L1 + independent LLM judge | **NOT_RUN** (no LLM judge credentials in this environment) | recorded as lower bound = L1 |
| L3 | L1 + held-out checks | 12/12 | **L3\L1 = 7** (`t1-m1, t2-m1, t3-m1, t3-m2, t4-m1, t5-m1, t6-m1`) |

L2\L1 = `null` — judge not executed; the harness marks `l2_complete: false` and refuses to treat L2 numbers as mapping evidence.

Cost/latency (per `A0-03-results.jsonl`): deterministic checks cost 0 (known), latency in microseconds per record, runtime identity `harness-local`, model `none`, environment `windows/amd64 go1.27.1` on every record (AC3).

## Interpretation

1. **L1 alone is not a safe universal default**: it missed 58% of single-defect candidates in a frozen corpus designed so that half the defects live only beyond visible behavior.
2. **Held-out adds real capture** (L3\L1 = 7/12 unique) at ~2× deterministic latency and zero marginal external cost in this harness.
3. **L2 remains an open empirical question locally**: no live judge was available, so the EvilGenie-style "cheap judge catches most reward-hacking" external claim is NOT locally validated and must not drive defaults yet (Decision Lock 11: no opinion-based mapping).

## Decision handed to A0-06 (conservative default per L2 §10 blocker 2)

```text
default floor:        L1 (deterministic visible + protected checks) everywhere
risk-adjusted uplift: L3 held-out for any acceptance clause above minimal risk
L2 (independent judge): experimental tier only — NOT a default until a live
                       judge bake-off reruns (P1-06 revisits with real AI Runtime)
```

This is the safe conservative default the freeze gate allows when L2 is unmeasured. It matches Decision Lock 11 (no hardcoded held-out-as-universal; exact mapping empirical) — held-out is uplift-by-risk, not universal.

## Required validation

```text
go test ./test/eval/v14_assurance/...: PASS (6/6, 2026-09-08 local run)
```
