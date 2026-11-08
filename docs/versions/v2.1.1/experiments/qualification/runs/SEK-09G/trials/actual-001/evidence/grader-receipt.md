# Independent blind-grader receipt

- Actual reviewer: `kimi-executor independent blind grading agent (no arm-key access)`.
- Reviewer kind: `INDEPENDENT_MODEL_GRADER`; this is **not** a human review and does not satisfy or claim G4.
- Completed UTC: `2026-11-08T08:39:46Z`.
- The reviewer reported reading only the three inputs below, without the hidden arm mapping, trial manifest, D/K-labelled requests, paired comparison or Builder summary. Its output was written outside the repository and copied byte-for-byte by the single evidence Writer after completion.
- `grader-completed.json` SHA256: `9b0680bb3b1991ebe81e21e5dd9e860614cb0e22ae1f03e7cef8b1398b4bfb9b`.
- All 14 blind IDs have correctness/scope/traceability scores, exact frozen criterion marks, unsupported-assertion annotations where applicable, and `observed_effort_seconds=null`. No reviewer effort was measured.

## Inputs made available to the grader

| File | SHA256 |
| --- | --- |
| `blind-TO-INDEPENDENT-SCORER.json` | `da7b7b430f23458a50e93164f688a6daf5225f63a70770b1c4e3a763c8493def` |
| `source-context-TO-INDEPENDENT-SCORER.json` | `61e11a8cc2f0764184a00f438cf8ae29a1c93d8622503b9a04a0008af56f7a9e` |
| `grader-blank.json` | `2ec898abf2219d6fff354af0057f130241741348465d403f352f781f3f5be05b` |

The hidden mapping was frozen at SHA256 `b133d9a1c213b55bf73d3752d3821218077c4b4add3cda1af8971bc17abaf045` and was not supplied. An earlier attempt to start a GLM grader failed before execution because the host reported a usage-limit reset; it produced no grades and is not the reviewer of record. The actual Kimi grader completed independently after that unavailable-role handoff.

The scoring validator subsequently confirmed the completed rating file against the frozen answers, original request/provider-response bytes, frozen criteria, blind IDs, hidden mapping and K retrieval hashes before creating `paired-comparison.json`. No score was supplied or edited by the Builder.
