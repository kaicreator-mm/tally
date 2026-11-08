# SEK-09G execution runbook (live model required)

Task: #367 · Parent: #351 · Source prereg: #361 / PR #363.

**Status: ACTUAL-001 COMPLETED — 14 real localhost responses frozen, independent automated blind grading completed, descriptive paired comparison produced. G5 gate remains for separate review; G4 human review NOT_CONDUCTED.** See `trials/actual-001/README.md` for the run result and receipts. The mock tests below remain software checks only and are not empirical evidence.

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
4. Run: `node docs/versions/v2.1.1/experiments/qualification/runs/SEK-09G/run-paired-trial.mjs --repo . --model qwen2.5-3b-instruct-q4_k_m.gguf --outdir docs/versions/v2.1.1/experiments/qualification/runs/SEK-09G/trials/actual-NNN` (use a new empty directory; `actual-001` is frozen and must not be reused).
5. Check the manifest for **14 SUCCESS** with provider-reported model, all actual raw outputs and SHA digests. If no inference service exists, report exact `PROVIDER_UNAVAILABLE` and do not create synthetic benchmark answers.
6. Give only `blind-TO-INDEPENDENT-SCORER.json` and `source-context-TO-INDEPENDENT-SCORER.json` to a **different** grader, not the hidden key file or arm-labelled requests. The latter supplies the exact pinned claim→evidence→locator→source excerpts for source-checked ratings without revealing D/K allocation. Record actual scoring identity, citations, correctness/scope/traceability, omissions and unsupported claims, then unblind after complete blinded scores and compute paired descriptive comparison per the prereg. Publish actual receipts in this scoped directory through the same single Writer, rerun checks, and obtain independent Exact-SHA verification.

## Current factual constraints

`trials/actual-001` now holds the actual one-repeat 7×2 experiment: Qwen2.5-3B-Instruct Q4_K_M served by a loopback-only llama.cpp container, 14/14 successful calls, all raw bytes and SHA256 records frozen before scoring. A distinct automated grader (`kimi-executor`) scored only the blind packet plus pinned source context; its completed file was validated and unblinded by `score-paired-trial.mjs`. Results are descriptive for this small sample: no G5 automatic PASS, no statistical superiority, no measured reviewer effort, and no G4 human review. `trials/actual-001` must never be overwritten; a rerun needs a new `actual-NNN` directory and separate reporting.

The earlier attempt under #369 found no Ollama/LM Studio/llama.cpp provider and no cached models; that blocker was resolved for this run by an open-weight GGUF model and a local CPU inference container. The two new real Case nodes are already integrated under #368/PR #374, but this benchmark intentionally stays pinned to the **older, preregistered** corpus graph for no-leakage comparisons.

On found execution defects the SAME local Builder repairs this exact runner/results write set, reruns Node >=24 checks and actual calls as needed, then hands a final HEAD to an independent verifier. No `LOCAL_VALIDATION`-only dispatch and no change to corpus, model, prereg, main or DomainForge.


## Independent blinded scoring (only after 14 actual SUCCESS answers)

This is an executable and fail-closed scoring workflow, not a request for validation-only handoff. The model-run Writer owns all run repairs and evidence; an actually distinct grader fills the blind packet **without receiving the arm key**.

1. Verify scoring validator's own mock test: `node docs/versions/v2.1.1/experiments/qualification/runs/SEK-09G/score-paired-trial.mjs --self-test`. This creates **no empirical G5 grade**.
2. After the real `actual-001` run has exactly 14 SUCCESS responses, create the blank scoring template:
   `node docs/versions/v2.1.1/experiments/qualification/runs/SEK-09G/score-paired-trial.mjs --make-template --run-dir docs/versions/v2.1.1/experiments/qualification/runs/SEK-09G/trials/actual-001 --out docs/versions/v2.1.1/experiments/qualification/runs/SEK-09G/trials/actual-001/grader-blank.json`
3. Supply the independent grader **only** the frozen `blind-TO-INDEPENDENT-SCORER.json`, `source-context-TO-INDEPENDENT-SCORER.json` and `grader-blank.json`. Never supply `blind-key-WITHHOLD-FROM-SCORER.json`, the trial manifest or arm-labelled request files before grading. The source context contains the same pinned retrieval records used by K requests but no arm assignments; the scoring validator checks each K retrieval SHA256 against it. The grader must record actual identity/type/UTC, `independent_from_builder=true` only if truthful, complete all scores and each `criteria_marks` outcome from the original semicolon-delimited frozen criteria, and document any unsupported answer spans with source checks. If effort is not measured, leave it `null`.
4. Store their **actually completed** output as `grader-completed.json`, then validate original request/response bytes, frozen blind answers, grading fields and unblind before producing any paired comparison:
   `node docs/versions/v2.1.1/experiments/qualification/runs/SEK-09G/score-paired-trial.mjs --run-dir docs/versions/v2.1.1/experiments/qualification/runs/SEK-09G/trials/actual-001 --scored-json docs/versions/v2.1.1/experiments/qualification/runs/SEK-09G/trials/actual-001/grader-completed.json --out docs/versions/v2.1.1/experiments/qualification/runs/SEK-09G/trials/actual-001/paired-comparison.json`
5. The validator rejects missing/fabricated/null grades, duplicate or swapped IDs, mismatched raw SHA256, changed prereg criteria, unsupported unsupported-assertion annotations and incomplete responses. Results are exploratory descriptive measurements; **no automatic G5 gate PASS**, no statistical superiority and no fake G4 human review.

The runner now persists `G5-NN-{D,K}-request.json` and `G5-NN-{D,K}-provider-response.json` as exact hashed original bytes, logs real model `usage` when exposed, and refuses redirects even from localhost. The score verifier's seed/mapping is separate from the material given to the independent grader.
