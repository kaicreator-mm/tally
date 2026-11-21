# Native execution and validation receipt

## Environment and exact source

- Windows-native Git Bash; Node `v24.21.0`; Python `3.14.7` (recorded, not used to generate answers or grades).
- Initial PR candidate: HEAD `3df197954a5102befc1c6ac35f43454cf304e755`, TREE `463dea5b054b9256337febbb480df4c1d83a660a`.
- Source-context repair candidate: HEAD `22c7e95db81c9512580d72fefcdfc5be38ce162b`, TREE `f3187f94193548eafd7bac954f6f86d23ce44e41`.
- Actual answer collection used the `22c7e95` source. Final evidence HEAD/TREE and independent Exact-SHA disposition are recorded on PR #375 after the evidence commit; this file does not invent a self-referential final SHA.

## Commands actually run

All commands below used the repository root as working directory unless shown otherwise. `RUN=docs/versions/v2.1.1/experiments/qualification/runs/SEK-09G` is only an abbreviation in this receipt; actual commands used the full path.

| Check | Actual result / exit |
| --- | --- |
| `node --check $RUN/run-paired-trial.mjs` at initial HEAD | exit 0, Node `v24.21.0` |
| `node --check $RUN/score-paired-trial.mjs` at initial HEAD | exit 0 |
| `node $RUN/run-paired-trial.mjs --self-test` at initial HEAD | 14 mock network calls; `SELF_TEST=PASS MOCK_ONLY REAL_MODEL_CALLS=0`; exit 0 |
| `node $RUN/score-paired-trial.mjs --self-test` at initial HEAD | `SCORER_SELF_TEST=PASS FIXTURE_ONLY ACTUAL_GRADES=0`; exit 0 |
| `node $RUN/run-paired-trial.mjs --repo . --preflight` at initial HEAD | exact prereg/graph blobs and seven chains `FROZEN_INPUTS=PASS`; exit 0 |
| Same runner syntax/self-test/preflight after source-context repair | all exit 0; mock-only output unchanged in meaning |
| Same scorer syntax/self-test after source-context repair | `SCORER_SELF_TEST=PASS FIXTURE_ONLY ACTUAL_GRADES=0`; exit 0 |
| `../../tools/task-check.sh` before repair push | `无 package.json, 无可运行检查(通过)`; exit 0 |
| `node $RUN/run-paired-trial.mjs --repo . --endpoint http://127.0.0.1:11434/v1/chat/completions --model qwen2.5-3b-instruct-q4_k_m.gguf --outdir $RUN/trials/actual-001 --timeout-ms 240000` | 7 D + 7 K real localhost calls, all `SUCCESS`; `ACTUAL_7X2_RUN_EXIT=0` |
| `sha256sum -c frozen-answer-SHA256SUMS` in `actual-001` | every frozen request, provider response, answer, manifest, blind packet, hidden key and source context `OK`; exit 0 |
| `node $RUN/score-paired-trial.mjs --make-template --run-dir $RUN/trials/actual-001 --out $RUN/trials/actual-001/grader-blank.json` | `BLIND_SCORING_TEMPLATE_CREATED=14 RATINGS_COMPLETED=0`; exit 0 |
| `node $RUN/score-paired-trial.mjs --run-dir $RUN/trials/actual-001 --scored-json $RUN/trials/actual-001/grader-completed.json --out $RUN/trials/actual-001/paired-comparison.json` | `PAIRED_SCORE_VALIDATION=PASS`; seven pairs/14 responses; exit 0 |
| `docker logs sek09g-llama-qwen3b > .../evidence/llama-server.log 2>&1` | exit 0 |

The mock tests are software checks only. The empirical result comes exclusively from the later 14 real localhost responses whose bytes and digests are preserved here.

## Frozen source identities

- Prereg commit `83a47d404311d2b1ac0bc06616ddd477bf13af56`, blob `5498f830ab7e832e97c9b6a879bf1c1d6d8fddcd`.
- Corpus commit `9cbf7983c5b47c146bd934f39e10517c3a828b66`, graph blob `2d4498dfa5f90bcf57cc81e9c155fb9ede31bf00`, graph SHA256 `21755df44c722c4bdd9fccb28446dd6aa12bbcfad168931344502f37d2aab7a4`.
- Frozen trial manifest SHA256 `dbfc43f02a889244aa5590c7db38672994dedbfbefbd44094c0b492a6bac8d8e`.
- Grader output SHA256 `9b0680bb3b1991ebe81e21e5dd9e860614cb0e22ae1f03e7cef8b1398b4bfb9b`.

## CI at source-repair candidate

GitHub `trailer-check` on `22c7e95db81c9512580d72fefcdfc5be38ce162b`: `SUCCESS` ([run 42461949507](https://github.com/kaicreator-mm/tally/actions/runs/42461949507)). Final evidence-commit CI is checked separately before merge. No Woodpecker result is claimed for this docs/experiment-only write set.

## Boundaries

No paid API, API key, remote inference endpoint, main merge, release, tag, deployment or DomainForge promotion was used. G4 human review remains `NOT_CONDUCTED`; reviewer effort is `NOT_MEASURED`; G5 remains subject to separate gate review rather than automatic PASS.
