# A0-04 — L3 Reference Pack

## A. Task Understanding

**Closed-loop Repair and held-out feedback ADR** — reduce task-level implementation uncertainty without redesigning frozen v1.4 architecture.

## B. Implementation Research Questions

- RQ-01: What separates inner agent iteration from outer repair attempt?
- RQ-02: What candidate/base must repair start from?
- RQ-03: Which verifier feedback is safe to expose?
- RQ-04: How do cancellation/infra/contract/exhaustion terminate?
- RQ-05: Which limits are policy defaults, not architecture constants?

## C. Reference Candidate Set

- **REF-SWE-RETRY** — `SWE-agent/SWE-agent@3ea751c087f32b16e039a2233dd6eefecef325d5` — Defines an explicit outer Review/Retry loop with max_attempts and cost limits, distinct from the inner agent step loop; each attempt is finalized before deciding to start the next.
- **REF-TEMPORAL-ERR** — `temporalio/sdk-go@82c8087ecd03722e2b8e7c6c892784095fa94e73` — Uses typed application errors plus explicit non-retryable classification and RetryPolicy; useful for deterministic routing without string-matching error messages.
- **REF-SPECBENCH** — `WecoAI/SpecBench@08607352adc8abd78be2193dd9f725f1f032b8f0` — Separates visible validation from held-out evaluation and measures the gap; useful for L1/L3 capture comparisons without leaking evaluation evidence.
- **REF-EVILGENIE** — `JonathanGabor/evilgenie_inspect@f5d8a2ca5d92ecf5fd1e857695c54e609312e1bd` — Combines held-out tests, LLM judge, and test-file mutation detection; tests scorer dimensions independently and keeps dataset splits deterministic.

## D. Recommended References

1. **REF-SWE-RETRY** — `sweagent/agent/reviewer.py`, `sweagent/agent/agents.py`
2. **REF-TEMPORAL-ERR** — `temporal/error.go`, `internal/error.go`, `internal/internal_task_handlers.go`
3. **REF-SPECBENCH** — `benchmarks/spec_bench/evaluation/runner.py`, `benchmarks/spec_bench/run_loop.py`, `benchmarks/spec_bench/adapter.py`
4. **REF-EVILGENIE** — `categorical_scorer.py`, `tests/test_scorer.py`, `tests/test_split_determinism.py`, `reward_hacking.py`

## E. Reference Relevance Matrix

| Reference | Contract | Behavior | Failure Model | Tests | Version | License |
|---|---|---|---|---|---|---|
| REF-SWE-RETRY | HIGH | HIGH | HIGH | MEDIUM | PINNED | MIT |
| REF-TEMPORAL-ERR | HIGH | HIGH | HIGH | MEDIUM | PINNED | MIT |
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

- Reusing ExecutionPolicy Retry.MaxIterations as repair budget.
- Repairing in place.
- Feeding raw held-out failures back then still claiming independence.
- Letting repair amend acceptance/evidence.
- Hard-coding max repair attempts=2 as architecture.

## O. Verification Mapping

- Nested loops and identities explicit.
- Each repair creates new run/candidate lineage.
- Held-out feedback allowlist explicit.
- Cancellation/exhaustion/escalation explicit.
- Default 2 remains policy default.

## P. Reference Pack Manifest

See `manifest.yaml`.

## Q. Coding Agent Read Order

1. `docs/tasks/v1.4/A0/A0-04.md`
2. `docs/tasks/v1.4/DECISION_LOCKS.md`
3. `docs/tasks/v1.4/ARCHITECTURE_EVIDENCE.md`
4. `docs/tasks/v1.4/reference-packs/A0-04/notes/current-code.md`
5. `docs/tasks/v1.4/reference-packs/A0-04/notes/research-evidence.md`
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
