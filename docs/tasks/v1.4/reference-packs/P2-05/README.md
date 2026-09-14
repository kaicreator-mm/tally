# P2-05 — L3 Reference Pack

## A. Task Understanding

**Repair continuation from prior candidate with immutable supersession lineage** — reduce task-level implementation uncertainty without redesigning frozen v1.4 architecture.

## B. Implementation Research Questions

- RQ-01: What state does the next attempt start from?
- RQ-02: How materialize prior candidate without mutating owner or old ChangeSet?
- RQ-03: How preserve frozen Contract pins?
- RQ-04: When write supersession?
- RQ-05: What cleanup failures can leak writable workspace?

## C. Reference Candidate Set

- **REF-SWE-RETRY** — `SWE-agent/SWE-agent@3ea751c087f32b16e039a2233dd6eefecef325d5` — Defines an explicit outer Review/Retry loop with max_attempts and cost limits, distinct from the inner agent step loop; each attempt is finalized before deciding to start the next.
- **REF-INTOTO** — `in-toto/attestation@Attestation Framework v1.2` — Binds claims to immutable subjects by digest and keeps execution metadata/evidence as typed attestations; useful for ChangeSet-bound runtime/evidence provenance.

## D. Recommended References

1. **REF-SWE-RETRY** — `sweagent/agent/reviewer.py`, `sweagent/agent/agents.py`
2. **REF-INTOTO** — `spec/v1/statement.md`, `spec/predicates/link.md`, `spec/predicates/test-result.md`

## E. Reference Relevance Matrix

| Reference | Contract | Behavior | Failure Model | Tests | Version | License |
|---|---|---|---|---|---|---|
| REF-SWE-RETRY | HIGH | HIGH | HIGH | MEDIUM | PINNED | MIT |
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

- Editing owner primary worktree.
- Mutating old candidate under same ID.
- Changing Contract pins silently.
- Writing supersession before new candidate freeze.
- Deleting failed-attempt evidence needed for audit.

## O. Verification Mapping

- Initial repair workspace equals prior candidate per ADR.
- Contract pins identical.
- New ChangeSet ID distinct.
- One append-only rework supersession after freeze.
- Owner worktree untouched.

## P. Reference Pack Manifest

See `manifest.yaml`.

## Q. Coding Agent Read Order

1. `docs/tasks/v1.4/P2/P2-05.md`
2. `docs/tasks/v1.4/DECISION_LOCKS.md`
3. `docs/tasks/v1.4/ARCHITECTURE_EVIDENCE.md`
4. `docs/tasks/v1.4/reference-packs/P2-05/notes/current-code.md`
5. `docs/tasks/v1.4/reference-packs/P2-05/notes/research-evidence.md`
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
