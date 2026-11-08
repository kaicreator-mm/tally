# SEK-09G execution runbook (live model required)

Task: #367 · Parent: #351 · Source prereg: #361 / PR #363.

**Status: CODE_READY / PROVIDER_UNAVAILABLE / REAL_TRIALS=NOT_RUN / G5=BLOCK.** Do not treat the included mock network test as a benchmark.

The Node 24 zero-dependency runner in this folder:
- Verifies the frozen prereg question table and source graph as Git blobs using exact commits (before contacting any LLM); fails closed on changed content.
- Resolves all seven Claim→Evidence→Locator→Source chains and their `SUPPORTED_BY` links from the pinned JSON-LD.
- Uses only a local OpenAI-compatible HTTP endpoint; refuses non-loopback HTTP addresses or paid remote API URLs. No token or API key is read.
- Runs 7 direct and 7 corpus-grounded stateless calls to the same model/parameters, with independent requests; direct prompts receive **no corpus details**, grounded prompts receive pinned graph text, neither receives expected criteria.
- Captures real provider response text, request/response digests, model string and elapsed time; writes incrementally and never overwrites a prior run; incomplete runs cannot be scored as successes.
- Only after 14 successful calls, freezes the answers and creates a blinded review packet and separate hidden arm mapping. All rating fields remain NULL until a distinct real scorer reviews them.

## Environment and execution

Windows-native Node **>=24** and Git; no WSL or npm dependencies.

1. Verify runner with a fake localhost HTTP server: `node docs/versions/v2.1.1/experiments/qualification/runs/SEK-09G/run-paired-trial.mjs --self-test` . Expect `SELF_TEST=PASS MOCK_ONLY REAL_MODEL_CALLS=0`.
2. Verify exact freeze from this Git checkout: `node docs/versions/v2.1.1/experiments/qualification/runs/SEK-09G/run-paired-trial.mjs --repo . --preflight`.
3. If Ollama not installed, install it on the allowed local execution workstation, pull an open-weight model with sufficient capacity (example `ollama pull qwen2.5:7b`), then start its local API. This installation/download is a *real local side effect* that the local Builder must own. Prefer the existing device's capacity and no paid APIs.
4. Run: `node docs/versions/v2.1.1/experiments/qualification/runs/SEK-09G/run-paired-trial.mjs --repo . --model qwen2.5:7b --outdir docs/versions/v2.1.1/experiments/qualification/runs/SEK-09G/trials/actual-001`.
5. Check the manifest for **14 SUCCESS** with provider-reported model, all actual raw outputs and SHA digests. If no inference service exists, report exact `PROVIDER_UNAVAILABLE` and do not create synthetic benchmark answers.
6. Give only `blind-TO-INDEPENDENT-SCORER.json` to a **different** grader, not the hidden key file. Record actual scoring identity, citations, correctness/scope/traceability, omissions and unsupported claims, then unblind after complete blinded scores and compute paired descriptive comparison per the prereg. Publish actual receipts in this scoped directory through the same single Writer, rerun checks, and obtain independent Exact-SHA verification.

## Current factual constraints

Previous actual local attempt #369 found no Ollama/LM Studio/llama.cpp provider, no cached models, and did not use a paid DeepSeek key. The assistant execution container also has no reachable model-download network or pretrained model; only a mock HTTP self-test was executed locally during runner development. **No actual 14-call result or scoring exists yet**, so no empirical G5 result or corpus advantage is claimed. The two new real Case nodes are already integrated under #368/PR #374, but this benchmark intentionally stays pinned to the **older, preregistered** corpus graph for no-leakage comparisons.

On found execution defects the SAME local Builder repairs this exact runner/results write set, reruns Node >=24 checks and actual calls as needed, then hands a final HEAD to an independent verifier. No `LOCAL_VALIDATION`-only dispatch and no change to corpus, model, prereg, main or DomainForge.
