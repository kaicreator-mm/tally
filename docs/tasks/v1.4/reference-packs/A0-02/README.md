# A0-02 — L3 Reference Pack

## A. Task Understanding

**Codex Session Isolation Spike** — reduce task-level implementation uncertainty without redesigning frozen v1.4 architecture.

## B. Implementation Research Questions

- RQ-01: Which CODEX_HOME state is observable by a new run and can contaminate session isolation?
- RQ-02: Can authentication remain usable when each run gets a unique CODEX_HOME?
- RQ-03: Does file auth behave differently from keyring auth when CODEX_HOME changes?
- RQ-04: What canary proves instructions/session history from run A are unobservable in run B?
- RQ-05: What minimal state may be handed off into an isolated home without copying prior session/instruction state?

## C. Reference Candidate Set

- **REF-CODEX-INSTR** — `openai/codex@553df1c691fe8bf7747e50da22f1342984495ae0` — Codex loads global AGENTS.md / AGENTS.override.md from CODEX_HOME; CODEX_HOME is observable instruction state, not merely a cache directory.
- **REF-CODEX-AUTH** — `openai/codex@553df1c691fe8bf7747e50da22f1342984495ae0` — File auth lives at CODEX_HOME/auth.json; keyring storage derives a key from the codex_home path, so per-run CODEX_HOME can change authentication behavior.
- **REF-CODEX-ROLLOUT** — `openai/codex@553df1c691fe8bf7747e50da22f1342984495ae0` — Active and archived session/rollout data are rooted under CODEX_HOME/sessions and CODEX_HOME/archived_sessions.

## D. Recommended References

1. **REF-CODEX-INSTR** — `codex-rs/codex-home/src/instructions/mod.rs`
2. **REF-CODEX-AUTH** — `codex-rs/login/src/auth/storage.rs`
3. **REF-CODEX-ROLLOUT** — `codex-rs/rollout/src/lib.rs`, `codex-rs/rollout/src/metadata.rs`

## E. Reference Relevance Matrix

| Reference | Contract | Behavior | Failure Model | Tests | Version | License |
|---|---|---|---|---|---|---|
| REF-CODEX-INSTR | HIGH | HIGH | HIGH | MEDIUM | PINNED | Apache-2.0 |
| REF-CODEX-AUTH | HIGH | HIGH | HIGH | MEDIUM | PINNED | Apache-2.0 |
| REF-CODEX-ROLLOUT | HIGH | HIGH | HIGH | MEDIUM | PINNED | Apache-2.0 |

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

- Symlinking/copying the entire CODEX_HOME.
- Assuming keyring auth is independent of CODEX_HOME.
- Treating successful login as proof of isolation.
- Declaring capability from configuration intent instead of the canary.

## O. Verification Mapping

- AC1 -> compare shared-home negative control and isolated-home candidates using the same canary.
- AC2 -> record file-auth/keyring/auth handoff behavior.
- AC3 -> no production capability is changed by the spike.
- AC4 -> report PASS architecture or exact BLOCKED conflict.

## P. Reference Pack Manifest

See `manifest.yaml`.

## Q. Coding Agent Read Order

1. `docs/tasks/v1.4/A0/A0-02.md`
2. `docs/tasks/v1.4/DECISION_LOCKS.md`
3. `docs/tasks/v1.4/ARCHITECTURE_EVIDENCE.md`
4. `docs/tasks/v1.4/reference-packs/A0-02/notes/current-code.md`
5. `docs/tasks/v1.4/reference-packs/A0-02/notes/research-evidence.md`
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
