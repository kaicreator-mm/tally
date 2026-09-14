# P2-08 — L3 Reference Pack

## A. Task Understanding

**Integrate Execute -> Verify -> Repair closed-loop engine** — reduce task-level implementation uncertainty without redesigning frozen v1.4 architecture.

## B. Implementation Research Questions

- RQ-01: What minimal state machine composes TaskExecution and verification?
- RQ-02: When is attempt finalized before next repair?
- RQ-03: Which routes cause repair/provider handling/escalation/cancellation/accept?
- RQ-04: How guarantee terminal basis/reason?
- RQ-05: How preserve immutable candidate/evidence bindings?

## C. Reference Candidate Set

- **REF-SWE-RETRY** — `SWE-agent/SWE-agent@3ea751c087f32b16e039a2233dd6eefecef325d5` — Defines an explicit outer Review/Retry loop with max_attempts and cost limits, distinct from the inner agent step loop; each attempt is finalized before deciding to start the next.
- **REF-TEMPORAL-ERR** — `temporalio/sdk-go@82c8087ecd03722e2b8e7c6c892784095fa94e73` — Uses typed application errors plus explicit non-retryable classification and RetryPolicy; useful for deterministic routing without string-matching error messages.
- **REF-INTOTO** — `in-toto/attestation@Attestation Framework v1.2` — Binds claims to immutable subjects by digest and keeps execution metadata/evidence as typed attestations; useful for ChangeSet-bound runtime/evidence provenance.

## D. Recommended References

1. **REF-SWE-RETRY** — `sweagent/agent/reviewer.py`, `sweagent/agent/agents.py`
2. **REF-TEMPORAL-ERR** — `temporal/error.go`, `internal/error.go`, `internal/internal_task_handlers.go`
3. **REF-INTOTO** — `spec/v1/statement.md`, `spec/predicates/link.md`, `spec/predicates/test-result.md`

## E. Reference Relevance Matrix

| Reference | Contract | Behavior | Failure Model | Tests | Version | License |
|---|---|---|---|---|---|---|
| REF-SWE-RETRY | HIGH | HIGH | HIGH | MEDIUM | PINNED | MIT |
| REF-TEMPORAL-ERR | HIGH | HIGH | HIGH | MEDIUM | PINNED | MIT |
| REF-INTOTO | HIGH | HIGH | HIGH | HIGH | PINNED | Apache-2.0 |

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

- Embedding verification implementation inside closedloop.
- Starting repair before prior evidence is durable.
- Acceptable without VerifiedBasis.
- Turning cancellation/verifier ERROR into fake FAIL.
- Unbounded while loop.

## O. Verification Mapping

- First-pass PASS has zero repair attempts.
- Repairable FAIL creates next attempt and re-verifies.
- Non-repairable/unknown escalate.
- Cancellation yields no fake verdict.
- Every terminal result carries typed basis/reason.

## P. Reference Pack Manifest

See `manifest.yaml`.

## Q. Coding Agent Read Order

1. `docs/tasks/v1.4/P2/P2-08.md`
2. `docs/tasks/v1.4/DECISION_LOCKS.md`
3. `docs/tasks/v1.4/ARCHITECTURE_EVIDENCE.md`
4. `docs/tasks/v1.4/reference-packs/P2-08/notes/current-code.md`
5. `docs/tasks/v1.4/reference-packs/P2-08/notes/research-evidence.md`
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
