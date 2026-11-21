# Builder mechanical audit — not independent review

`role: BUILDER_SELF_CHECK_NOT_INDEPENDENT_REVIEW`

This audit was produced by the same Writer that ran the trials and recorded the evidence. It is a reproducibility self-check, **not** the independent Exact-SHA review required for WIP merge. Independent reviewer dispatch was attempted and failed before execution because the host reported model usage limits (GLM reset `2026-11-08T20:24:27` local; Kimi weekly limit reached). No independent review result exists as of this commit.

## What the machine check covered

`evidence/builder-mechanical-audit.json` records 242 individual checks, all `pass`, run with Node `v24.21.0`:

- Exact HEAD/TREE and clean worktree; 59 changed paths, all inside `runs/SEK-09G/`; 10 commits, all with `Refs: #367`.
- `sha256sum -c SHA256SUMS` (56 files) and `sha256sum -c frozen-answer-SHA256SUMS` (47 pre-grading files): all OK.
- 14 trials all `SUCCESS`, 7 D / 7 K, one returned model, one `system_fingerprint`, all `finish_reason=stop`.
- Every request/response digest recomputed and matched; every provider response body's answer equals the frozen answer record.
- Every D user message is exactly the frozen question; every K user message is exactly the question plus the fixed prefix plus pinned retrieval JSON; K retrieval hashes match `source-context-TO-INDEPENDENT-SCORER.json`; D retrieval hashes are null.
- No frozen criteria or trap text appears in any prompt; 14 answers are distinct and non-empty.
- Independent grader file: 14 complete ratings, integer 0..2 scores, criterion text/order matches the blank template, `INDEPENDENT_MODEL_GRADER`, Kimi reference.
- `paired-comparison.json` embeds the actual SHA256 of both the grader output and the trial manifest, has status `ACTUAL_BLIND_GRADING_VERIFIED_NO_AUTO_PASS`, and explicitly withholds superiority, G4 and automatic G5 claims.
- Node 24 self-tests and frozen-input preflight all exit 0; GitHub `trailer-check` on `6406b6e84d3ef74b7cd1e04bd29b59456b851f36` is `SUCCESS`.
- Provider receipt declares non-billable loopback-only local inference, no API key, no paid API.

## What this does not establish

- It does not substitute for a distinct reviewer's Exact-SHA review.
- It does not establish G5 gate acceptance, statistical superiority or G4 human review.
- It does not authorize merge to `wip/v2.1.1-knowledge` by itself.
