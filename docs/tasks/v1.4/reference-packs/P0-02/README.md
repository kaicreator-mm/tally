# P0-02 — L3 Reference Pack

## A. Task Understanding

**Persist actual Codex runtime/model/session snapshot** — reduce task-level implementation uncertainty without redesigning frozen v1.4 architecture.

## B. Implementation Research Questions

- RQ-01: Which Codex facts are observed vs declared?
- RQ-02: How map CLI/binary/session/model/capabilities into generic snapshot?
- RQ-03: Which facts are process measurements vs claims?
- RQ-04: How prevent adapter version from being called CLI version?
- RQ-05: Can existing codegen_run fields hold them without migration?

## C. Reference Candidate Set

- **REF-CODEX-AUTH** — `openai/codex@553df1c691fe8bf7747e50da22f1342984495ae0` — File auth lives at CODEX_HOME/auth.json; keyring storage derives a key from the codex_home path, so per-run CODEX_HOME can change authentication behavior.
- **REF-CODEX-ROLLOUT** — `openai/codex@553df1c691fe8bf7747e50da22f1342984495ae0` — Active and archived session/rollout data are rooted under CODEX_HOME/sessions and CODEX_HOME/archived_sessions.
- **REF-SLSA-PROV** — `slsa-framework/slsa@spec v1.1/v1.2 family` — Separates builder identity from per-invocation metadata such as invocationId/start/finish; useful for observed runtime identity versus desired configuration.
- **REF-INTOTO** — `in-toto/attestation@Attestation Framework v1.2` — Binds claims to immutable subjects by digest and keeps execution metadata/evidence as typed attestations; useful for ChangeSet-bound runtime/evidence provenance.

## D. Recommended References

1. **REF-CODEX-AUTH** — `codex-rs/login/src/auth/storage.rs`
2. **REF-CODEX-ROLLOUT** — `codex-rs/rollout/src/lib.rs`, `codex-rs/rollout/src/metadata.rs`
3. **REF-SLSA-PROV** — `spec/build-provenance.md`
4. **REF-INTOTO** — `spec/v1/statement.md`, `spec/predicates/link.md`, `spec/predicates/test-result.md`

## E. Reference Relevance Matrix

| Reference | Contract | Behavior | Failure Model | Tests | Version | License |
|---|---|---|---|---|---|---|
| REF-CODEX-AUTH | HIGH | HIGH | HIGH | MEDIUM | PINNED | Apache-2.0 |
| REF-CODEX-ROLLOUT | HIGH | HIGH | HIGH | MEDIUM | PINNED | Apache-2.0 |
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

- Writing DescriptorVersion='1' as Codex CLI version.
- Persisting requested model as observed model.
- Storing auth/CODEX_HOME secrets in identity.
- Adding Codex-specific DB columns prematurely.

## O. Verification Mapping

- AC1 -> adapter and CLI versions distinct.
- AC2 -> binary/session facts come from runtime observation.
- AC3 -> no Codex-only DB path.
- AC4 -> unknown observed model does not become false data.

## P. Reference Pack Manifest

See `manifest.yaml`.

## Q. Coding Agent Read Order

1. `docs/tasks/v1.4/P0/P0-02.md`
2. `docs/tasks/v1.4/DECISION_LOCKS.md`
3. `docs/tasks/v1.4/ARCHITECTURE_EVIDENCE.md`
4. `docs/tasks/v1.4/reference-packs/P0-02/notes/current-code.md`
5. `docs/tasks/v1.4/reference-packs/P0-02/notes/research-evidence.md`
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
