# SEK-09G — G5 Paired Trial Run (answer collection attempt #1)

**Work item:** #367 · **Design:** `docs/versions/v2.1.1/experiments/qualification/SEK-09E-UTILITY-PREREGISTRATION.md` (prereg blob `5498f830ab7e832e97c9b6a879bf1c1d6d8fddcd` at HEAD `55ab06a8a09b8aa850e1c75743d66420d8c57b56`)

**Status: BLOCKED / PROVIDER_UNAVAILABLE. EXPERIMENT=NOT_RUN. G5=NOT_MEASURED. HUMAN_OR_INDEPENDENT_SCORING=NOT_RUN.**

No answers, scores, blinded packages or trial outputs exist in this run directory. Nothing was fabricated.

## What was done in this attempt (2026-11-08T00:19:57Z)

1. **Pinned-input verification (passed, exit 0).** Before any provider check, the frozen basis was re-verified read-only by `C:\xDev\kAiCreator\_tmp\tally-i367-k3\verify_frozen_inputs.py` (report: `evidence/frozen-input-verification.json`):
   - Pinned corpus baseline `wip/v2.1.1-knowledge` HEAD `9cbf7983c5b47c146bd934f39e10517c3a828b66`, TREE `fe8f5e106f349bb0c814696fcd231732f7568250` — verified.
   - Underlying corpus TREE `a7b972fea96b9f12bbb8066fe6c88c13beb5f8d9` (SEK-08 knowledge-graph integration) — verified.
   - Current WIP HEAD `55ab06a8a09b8aa850e1c75743d66420d8c57b56`, TREE `c20283b53d724fe9caccc01e5a1a831593850332` — verified.
   - All 7 frozen claim→evidence→locator→source chains (G5-01..G5-07) resolve in the pinned corpus tree: claim present, evidence present, SUPPORTED_BY relation present, evidence→locator ref match, locator→source ref match, source present in `SOURCE_REGISTRY.yaml`.
   - The 7 frozen question texts (hashes in the verification report) were captured from the prereg blob **before** any model interaction; no question, criterion or rubric was changed.
2. **Provider probe (failed).** No non-billable local inference provider is available on this machine — full command/exit-code evidence in `evidence/provider-probe-2026-11-08.json`. Summary: Ollama not installed and port 11434 refused (curl exit 7); LM Studio not installed and port 1234 refused (exit 7); llama.cpp/llamafile binaries absent; no Python inference stack (torch/transformers/llama-cpp/vllm) installed; no inference Docker container running; common local inference ports (8080/8000/5000/5001/11435/4891) all unreachable; no Hugging Face model cache. A DeepSeek key exists in the environment for other test purposes, but it is a paid API and was deliberately not used per #367.
3. **Trial capture.** All 14 planned calls (7 direct + 7 corpus-grounded, repeat=1) are recorded as `NOT_RUN` with reason `PROVIDER_UNAVAILABLE` in `trials/trial-capture.json`.

## Concrete next action

Install a local open-weights inference runtime (Ollama or LM Studio with a 7B-class model such as qwen2.5:7b / llama3.1:8b), then re-run this stage: 14 independent stateless calls with recorded provider/model/revision, parameters, UTC timestamps, request/response/retrieval/input hashes and statuses; freeze raw answers; build the seeded blind package and mapping; then hand off to the independent grader. Any paid-API route requires a separate budget authorization and was not taken.
