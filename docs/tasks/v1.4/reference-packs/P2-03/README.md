# P2-03 — L3 Reference Pack

## A. Task Understanding

**Verification failure classification** — reduce task-level implementation uncertainty without redesigning frozen v1.4 architecture.

## B. Implementation Research Questions

- RQ-01: Which stable failure classes are typed domain values?
- RQ-02: Which route to repair/provider retry/escalate/stop?
- RQ-03: How does unknown fail safe?
- RQ-04: How preserve FAIL vs ERROR?
- RQ-05: How avoid reinterpreting Acceptance?

## C. Reference Candidate Set

- **REF-TEMPORAL-ERR** — `temporalio/sdk-go@82c8087ecd03722e2b8e7c6c892784095fa94e73` — Uses typed application errors plus explicit non-retryable classification and RetryPolicy; useful for deterministic routing without string-matching error messages.

## D. Recommended References

1. **REF-TEMPORAL-ERR** — `temporal/error.go`, `internal/error.go`, `internal/internal_task_handlers.go`

## E. Reference Relevance Matrix

| Reference | Contract | Behavior | Failure Model | Tests | Version | License |
|---|---|---|---|---|---|---|
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

- String matching stderr.
- Repairing every FAIL.
- Treating verifier ERROR as candidate defect.
- Mutating acceptance in classifier.
- Unknown->repair default.

## O. Verification Mapping

- Each class has exactly one route.
- Unknown escalates.
- Tests distinguish FAIL vs ERROR/provider/environment.
- Classifier cannot mutate Acceptance.

## P. Reference Pack Manifest

See `manifest.yaml`.

## Q. Coding Agent Read Order

1. `docs/tasks/v1.4/P2/P2-03.md`
2. `docs/tasks/v1.4/DECISION_LOCKS.md`
3. `docs/tasks/v1.4/ARCHITECTURE_EVIDENCE.md`
4. `docs/tasks/v1.4/reference-packs/P2-03/notes/current-code.md`
5. `docs/tasks/v1.4/reference-packs/P2-03/notes/research-evidence.md`
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
