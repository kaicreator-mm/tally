# P2-07 — L3 Reference Pack

## A. Task Understanding

**Separate bounded outer RepairPolicy from ExecutionPolicy retry** — reduce task-level implementation uncertainty without redesigning frozen v1.4 architecture.

## B. Implementation Research Questions

- RQ-01: What belongs to repair policy vs execution retry?
- RQ-02: How represent max attempts/cancellation/exhaustion?
- RQ-03: How reject zero/negative/unbounded?
- RQ-04: How avoid final-attempt off-by-one?
- RQ-05: What typed exhaustion reason is emitted?

## C. Reference Candidate Set

- **REF-SWE-RETRY** — `SWE-agent/SWE-agent@3ea751c087f32b16e039a2233dd6eefecef325d5` — Defines an explicit outer Review/Retry loop with max_attempts and cost limits, distinct from the inner agent step loop; each attempt is finalized before deciding to start the next.
- **REF-TEMPORAL-ERR** — `temporalio/sdk-go@82c8087ecd03722e2b8e7c6c892784095fa94e73` — Uses typed application errors plus explicit non-retryable classification and RetryPolicy; useful for deterministic routing without string-matching error messages.

## D. Recommended References

1. **REF-SWE-RETRY** — `sweagent/agent/reviewer.py`, `sweagent/agent/agents.py`
2. **REF-TEMPORAL-ERR** — `temporal/error.go`, `internal/error.go`, `internal/internal_task_handlers.go`

## E. Reference Relevance Matrix

| Reference | Contract | Behavior | Failure Model | Tests | Version | License |
|---|---|---|---|---|---|---|
| REF-SWE-RETRY | HIGH | HIGH | HIGH | MEDIUM | PINNED | MIT |
| REF-TEMPORAL-ERR | HIGH | HIGH | HIGH | MEDIUM | PINNED | MIT |

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

- Aliasing to ExecutionPolicy.Retry.MaxIterations.
- Mixing attempts vs retries semantics.
- Exhaustion check before final success check.
- 0/negative meaning unlimited.
- Waiver silently extending budget.

## O. Verification Mapping

- Tests: first-pass success, success on final attempt, exhaustion, cancellation, invalid bounds.
- Default=2 lives in policy construction.
- Exhaustion is typed escalation reason.

## P. Reference Pack Manifest

See `manifest.yaml`.

## Q. Coding Agent Read Order

1. `docs/tasks/v1.4/P2/P2-07.md`
2. `docs/tasks/v1.4/DECISION_LOCKS.md`
3. `docs/tasks/v1.4/ARCHITECTURE_EVIDENCE.md`
4. `docs/tasks/v1.4/reference-packs/P2-07/notes/current-code.md`
5. `docs/tasks/v1.4/reference-packs/P2-07/notes/research-evidence.md`
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
