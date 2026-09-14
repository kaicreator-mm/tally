# P0-01 — L3 Reference Pack

## A. Task Understanding

**Propagate requested ModelProfile and capture generic runtime identity** — reduce task-level implementation uncertainty without redesigning frozen v1.4 architecture.

## B. Implementation Research Questions

- RQ-01: How are desired and observed runtime/model identity separated?
- RQ-02: Which neutral fields belong in agentruntime?
- RQ-03: How is observed identity bound to execution/candidate provenance?
- RQ-04: What remains optional for FakeRuntime?
- RQ-05: How do tests prove desired does not silently become actual?

## C. Reference Candidate Set

- **REF-SLSA-PROV** — `slsa-framework/slsa@spec v1.1/v1.2 family` — Separates builder identity from per-invocation metadata such as invocationId/start/finish; useful for observed runtime identity versus desired configuration.
- **REF-INTOTO** — `in-toto/attestation@Attestation Framework v1.2` — Binds claims to immutable subjects by digest and keeps execution metadata/evidence as typed attestations; useful for ChangeSet-bound runtime/evidence provenance.

## D. Recommended References

1. **REF-SLSA-PROV** — `spec/build-provenance.md`
2. **REF-INTOTO** — `spec/v1/statement.md`, `spec/predicates/link.md`, `spec/predicates/test-result.md`

## E. Reference Relevance Matrix

| Reference | Contract | Behavior | Failure Model | Tests | Version | License |
|---|---|---|---|---|---|---|
| REF-SLSA-PROV | HIGH | HIGH | HIGH | MEDIUM | PINNED | Study-only; verify exact spec terms before copying |
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

- Recording requested model as actual model.
- Adding Codex-only CLI fields to generic control flow.
- Making observed identity mandatory for FakeRuntime.
- Using adapter version as actual CLI version.

## O. Verification Mapping

- AC1 -> selected ModelProfile reaches StartRequest.
- AC2 -> no runtime-name branch.
- AC3 -> missing optional observed data is safe.
- AC4 -> requested/observed mismatch test stays distinguishable.

## P. Reference Pack Manifest

See `manifest.yaml`.

## Q. Coding Agent Read Order

1. `docs/tasks/v1.4/P0/P0-01.md`
2. `docs/tasks/v1.4/DECISION_LOCKS.md`
3. `docs/tasks/v1.4/ARCHITECTURE_EVIDENCE.md`
4. `docs/tasks/v1.4/reference-packs/P0-01/notes/current-code.md`
5. `docs/tasks/v1.4/reference-packs/P0-01/notes/research-evidence.md`
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
