# SEK-09G actual-001 — frozen 7×2 paired trial and independent blind grading

**Status: ACTUAL_7X2_COMPLETED / ANSWERS_FROZEN_BEFORE_SCORING / INDEPENDENT_MODEL_BLIND_GRADING_COMPLETED / DESCRIPTIVE_RESULTS_AVAILABLE. G5 gate acceptance is not automatic; G4 human review is NOT_CONDUCTED.**

This directory contains the actual 14 localhost model responses, their original request/provider bytes, frozen preregistered inputs, the blind grading inputs and completed ratings, and the seven paired comparisons. It is an exploratory one-repeat sample, not a statistical superiority claim, production acceptance, or DomainForge promotion.

## Reproducible identities

- Model: Qwen2.5-3B-Instruct GGUF Q4_K_M, file SHA256 `626b4a6678b86442240e33df819e00132d3ba7dddfe1cdc4fbb18e0a9615c62d`.
- Provider: local llama.cpp container, loopback-only endpoint, image digest `sha256:ca40726778c25a7fadce42c09bada8484eab22e28bc283bdbf212e87b1555390`; all responses report system fingerprint `b12199-372dc5ca1` and `finish_reason=stop`.
- Trial UTC: `2026-11-08T08:30:22.921Z` through `2026-11-08T08:34:39.616Z`; 14/14 success, 6,069 total tokens. No paid API or credential was used.
- Frozen prereg blob `5498f830ab7e832e97c9b6a879bf1c1d6d8fddcd`; frozen graph blob `2d4498dfa5f90bcf57cc81e9c155fb9ede31bf00`.
- Frozen trial manifest SHA256 `dbfc43f02a889244aa5590c7db38672994dedbfbefbd44094c0b492a6bac8d8e`.
- Independent automated grader: `kimi-executor`, no arm-key access; completed `2026-11-08T08:39:46Z`. Grader output SHA256 `9b0680bb3b1991ebe81e21e5dd9e860614cb0e22ae1f03e7cef8b1398b4bfb9b`.
- See [provider receipt](evidence/provider-receipt.json), [grader receipt](evidence/grader-receipt.md), [native validation receipt](evidence/native-validation.md), and [server log](evidence/llama-server.log).

## All seven questions, raw responses and grades

Scores below are `correctness / scope / traceability`, each 0..2; `U` is the count of annotated unsupported assertions. The answer record contains the exact answer text and request/response digests; the raw provider file is the original HTTP response body. All individual criterion marks and unsupported spans are in [grader-completed.json](grader-completed.json), and all per-question deltas and omissions are in [paired-comparison.json](paired-comparison.json).

| Q | Frozen question | Direct answer / raw provider bytes | D score; U | Corpus answer / raw provider bytes | K score; U |
| --- | --- | --- | --- | --- | --- |
| G5-01 | In CS2023's requirements curriculum, which functional-requirement description approaches are named, and is one ranked best? | [answer](G5-01-D.json) · [raw](G5-01-D-provider-response.json) | 1/1/1; U=2 | [answer](G5-01-K.json) · [raw](G5-01-K-provider-response.json) | 2/2/1; U=0 |
| G5-02 | Does CS2023 distinguish architectural from detailed design, and specify exact criteria for their boundary? | [answer](G5-02-D.json) · [raw](G5-02-D-provider-response.json) | 0/1/1; U=1 | [answer](G5-02-K.json) · [raw](G5-02-K-provider-response.json) | 1/2/1; U=0 |
| G5-03 | How does the CS2023 SE-Validation unit distinguish verification from validation? | [answer](G5-03-D.json) · [raw](G5-03-D-provider-response.json) | 2/1/1; U=1 | [answer](G5-03-K.json) · [raw](G5-03-K-provider-response.json) | 2/2/1; U=0 |
| G5-04 | Does SWEBOK v4.0a's construction definition include verification, tests and debugging, and establish fixed process boundaries? | [answer](G5-04-D.json) · [raw](G5-04-D-provider-response.json) | 0/2/1; U=1 | [answer](G5-04-K.json) · [raw](G5-04-K-provider-response.json) | 2/2/1; U=0 |
| G5-05 | Does CS2023's Software Reliability unit specify a measurement procedure for its reliability quantities? | [answer](G5-05-D.json) · [raw](G5-05-D-provider-response.json) | 2/2/1; U=1 | [answer](G5-05-K.json) · [raw](G5-05-K-provider-response.json) | 2/2/1; U=0 |
| G5-06 | Is maintenance in SWEBOK v4.0a limited to post-delivery, and was the cited ISO 14764 text inspected here? | [answer](G5-06-D.json) · [raw](G5-06-D-provider-response.json) | 0/2/1; U=1 | [answer](G5-06-K.json) · [raw](G5-06-K-provider-response.json) | 2/2/1; U=0 |
| G5-07 | What activities fall under software engineering management according to SWEBOK v4.0a, and does that establish universal consensus? | [answer](G5-07-D.json) · [raw](G5-07-D-provider-response.json) | 1/2/1; U=2 | [answer](G5-07-K.json) · [raw](G5-07-K-provider-response.json) | 2/2/1; U=0 |

## Descriptive paired result

| Metric | Direct (7) | Corpus-grounded (7) | K−D interpretation boundary |
| --- | ---: | ---: | --- |
| Mean correctness | 0.8571428571 | 1.8571428571 | Descriptive sample only |
| Mean source scope | 1.5714285714 | 2 | Descriptive sample only |
| Mean traceability | 1 | 1 | No improvement; no answer earned a complete explicit chain score of 2 |
| Unsupported assertions | 9 | 0 | Grader-annotated count, not an inferential statistic |

Per-question correctness K−D deltas are `+1, +1, 0, +2, 0, +2, +1` for G5-01..07 respectively. Omission-rate deltas and every raw grade are preserved in the comparison artifact. Reviewer effort was not measured (`0/14` timed records), so no effort comparison is made. No confidence interval, significance test, universal claim, automatic G5 PASS, G4 human-review claim, or production acceptance is inferred from this single small sample.

## Freeze and verification layout

- `frozen-answer-SHA256SUMS`: pre-grading freeze of all raw requests/responses, answer records, manifest, questions, blind packet, hidden mapping and source context.
- `blind-TO-INDEPENDENT-SCORER.json` plus `source-context-TO-INDEPENDENT-SCORER.json`: the only answer/source inputs supplied to the grader, together with the blank template.
- `blind-key-WITHHOLD-FROM-SCORER.json`: hidden arm mapping, used only after grading for unblinding.
- `grader-completed.json`: unchanged independent grader output.
- `paired-comparison.json`: generated only after fail-closed validation of all 14 ratings and original bytes.
- `SHA256SUMS`: final whole-run integrity manifest, including grading and evidence files.
