# A0-03 — L3 Reference Pack

## A. Task Understanding

**Assurance L1/L2/L3 Bake-off** — reduce task-level implementation uncertainty without redesigning frozen v1.4 architecture.

## B. Implementation Research Questions

- RQ-01: How is the task/mutant set frozen before inspecting results?
- RQ-02: How are unique defects caught by L1/L2/L3 measured?
- RQ-03: How is mutation detection scored independently from held-out failure?
- RQ-04: How is visible/held-out split deterministic?
- RQ-05: What cost/latency/runtime identity is required for reproducibility?

## C. Reference Candidate Set

- **REF-SPECBENCH** — `WecoAI/SpecBench@08607352adc8abd78be2193dd9f725f1f032b8f0` — Separates visible validation from held-out evaluation and measures the gap; useful for L1/L3 capture comparisons without leaking evaluation evidence.
- **REF-EVILGENIE** — `JonathanGabor/evilgenie_inspect@f5d8a2ca5d92ecf5fd1e857695c54e609312e1bd` — Combines held-out tests, LLM judge, and test-file mutation detection; tests scorer dimensions independently and keeps dataset splits deterministic.

## D. Recommended References

1. **REF-SPECBENCH** — `benchmarks/spec_bench/evaluation/runner.py`, `benchmarks/spec_bench/run_loop.py`, `benchmarks/spec_bench/adapter.py`
2. **REF-EVILGENIE** — `categorical_scorer.py`, `tests/test_scorer.py`, `tests/test_split_determinism.py`, `reward_hacking.py`

## E. Reference Relevance Matrix

| Reference | Contract | Behavior | Failure Model | Tests | Version | License |
|---|---|---|---|---|---|---|
| REF-SPECBENCH | HIGH | HIGH | HIGH | MEDIUM | PINNED | Apache-2.0 |
| REF-EVILGENIE | HIGH | HIGH | HIGH | HIGH | PINNED | MIT |

## F. Test Evidence

See `tests/README.md` and pinned test/scorer paths in `notes/research-evidence.md`.

## G. Contract / Interface Evidence

Use the pinned references for state/retry/provenance semantics only. Local TaskContract and Decision Locks are authoritative.

## H. Core Implementation Evidence

Do not read whole repositories. Read only `manifest.yaml` paths.

## I. Failure Handling Evidence

Study typed failure, retry, cancellation, cleanup and finalization behavior from references, then map to Tally’s existing facts. External defaults are never local defaults.

## J. What to Learn

{cfg['pattern']}

## K. What NOT to Copy

- External persistence/data model.
- External provider/runtime coupling.
- External UI/telemetry.
- Retry/timeout/defaults conflicting with Tally Decision Locks.
- External code without a fresh license/attribution check.

## L. Current Code Gap

Read `notes/current-code.md`. Implement incrementally over v1.3 Trust Core.

## M. Recommended Implementation Pattern

{cfg['pattern']}

## N. Likely Implementation Mistakes

- Changing the corpus after results.
- Letting L2 see held-out names/failures.
- Equating held-out failure with malicious reward hacking.
- Reporting only aggregate pass rate.
- Comparing costs without model/runtime/environment identity.

## O. Verification Mapping

- AC1 -> manifest/hash frozen before results.
- AC2 -> unique caught defect IDs per layer.
- AC3 -> latency/cost include environment/model/runtime identity.
- AC4 -> no production selector changes.

## P. Reference Pack Manifest

See `manifest.yaml`.

## Q. Coding Agent Read Order

1. `docs/tasks/v1.4/A0/A0-03.md`
2. `docs/tasks/v1.4/DECISION_LOCKS.md`
3. `docs/tasks/v1.4/ARCHITECTURE_EVIDENCE.md`
4. `docs/tasks/v1.4/reference-packs/A0-03/notes/current-code.md`
5. `docs/tasks/v1.4/reference-packs/A0-03/notes/research-evidence.md`
6. Reference tests/contracts listed in `manifest.yaml`
7. Reference core path(s) listed in `manifest.yaml`
8. Implement only against local authority
9. Run the Task’s required validation commands

## R. Unknowns

- External references do not prove that their defaults fit Tally.
- Real Codex credentials/CLI behavior and model-dependent evaluations remain local Spike/E2E questions.
- On conflict, mark `CONFLICT` and follow local authority.

## Stop Condition

Stop researching when the HIGH-risk questions above have evidence and remaining uncertainty requires local code/test/spike.
