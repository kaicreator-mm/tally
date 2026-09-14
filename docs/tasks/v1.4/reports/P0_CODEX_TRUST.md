# P0_CODEX_TRUST — Real Codex standard + held-out trust E2E evidence (P0-04)

**Date:** 2026-09-09
**Task:** `docs/tasks/v1.4/P0/P0-04.md` (amended 2026-09-09 with the ADR-0003 carve-out)
**Evidence:** `go test -tags=integration ./test/e2e/v14_codex_trust/...` — PASS (executed 2026-09-09, this machine)
**Runtime identity (this machine):** codex binary `C:\Users\15549\AppData\Local\OpenAI\Codex\bin\9ba750cce02d5e5c\codex.exe`, self-reported `codex-cli 0.153.0` (live `--version` probe via `codex.Runtime.Snapshot`); adapter version `1`.

## 1. Runtime / provider / model

| Fact | Value | Source |
| --- | --- | --- |
| Binary path | `C:\Users\15549\AppData\Local\OpenAI\Codex\bin\9ba750cce02d5e5c\codex.exe` | `RuntimeSnapshot.BinaryPath` (== `Config.BinPath`) |
| CLI version (observed) | `0.153.0` (`codex-cli` self-report) | live `codex --version` via Snapshot |
| Adapter version | `1` (`DescriptorVersion` — distinct from CLI version) | Snapshot |
| Provider endpoint | ID `custom`, Name `deepseek`, BaseURL `https://api.deepseek.com`, WireAPI `responses`, RequiresOpenAIAuth `true`, EnvKey empty | ADR-0003 `Config.Provider` (typed, seeded into per-run home as minimal `config.toml`; key material arrived **only** via the #22 `auth.json` handoff from a temp owner home built from `TALLY_E2E_CODEX_AUTH_FILE`) |
| Model requested | `deepseek-v4-flash` (config seeding `-m` flag, both) | desired configuration |
| Model observed | **unknown** — the exec JSONL surface emits no model self-report; never backfilled from the requested value (Decision Lock 15) | `ObservedIdentity.Model` == "" asserted in test |
| Session observed | real codex thread ids per turn (e.g. `01a0826a-aa7d-…`), captured from `thread.started` and asserted `==` claim SessionID; persisted as `codegen_run.agent_session_id` | `IdentityReporter` + runstore row |
| Usage observed | unavailable — adapter does not parse exec usage events; recorded as unavailable, not fabricated zeros (lock 16). For reference, raw CLI JSONL `turn.completed` events on this machine reported ~9.5k–174k input / 19–8.8k output tokens per turn (claims, not Tally facts) | adapter claim plane |

## 2. Session Isolation Canary (declaration gate)

**Result: PASS** — `runtimeconformance.CheckSessionIsolation` against the real codex CLI through the production adapter (`runtimes/codex`, per-run `CODEX_HOME` + auth.json-only handoff), executed 2026-09-09, ~54s wall.

- `CapabilityDeclared=true`, `CapabilityAvailable=true`, zero Failures.
- `InjectionObserved=false` (injection echo not protocol-observable; noted by the canary itself — the retrieval verdict is unaffected: no retrieval channel returned the token).
- Residual risk carried verbatim: provider-internal opaque retention remains declared residual risk (`ResidualRiskNote`).
- Only after this PASS did the E2E construct runtimes with `SessionIsolationVerified=true` and assert `session.isolation` in `Descriptor()` and `Snapshot()`. Declaration is now evidence-backed for this exact CLI + adapter (DECISION_LOCKS #22/#24).

## 3. Standard path (AC1 — real candidate in disposable workspace)

**Result: PASS.** Seeded fixture repo (tiny Go module, one visible red test) → real codex turn in a `workspace.Manager` disposable worktree under a test-owned root → candidate fixed **from worktree facts only** (`git status --porcelain` + content read-back; zero content injection) → immutable ChangeSet via `changeset.New` → persisted via production `runstore` APIs.

- Real turn duration ~60–120s; candidate = `greeting/greeting.go` with the delivered constant.
- `codegen_run` row: runtime `codex`, model `deepseek-v4-flash` (desired plane) with `observed_identity` (`codex-cli` / `0.153.0` / observed session id) persisted in the capability snapshot JSONB, `agent_session_id` = observed thread id, `result_patch_hash` = ChangeSet identity.
- The owner fixture repo stayed clean for the whole run (`git status --porcelain` empty); INV-12 held — every execution workspace was a created temp worktree, never an owner worktree.

## 4. Held-out final verification (AC2 — under required isolation)

**Result: PASS.** Frozen VerificationContract (visible `go test ./greeting/` + held_out `git grep` binding via `holdout.FilesystemStore`, content protected by baseline digest) executed by the P1-38 orchestrator in a disposable verification workspace; held-out content materialized only inside a Tally-controlled one-shot boundary directory.

- FinalOutcome `verified`; independence coverage complete (held-out PASS supplies information independence); visible binding exit 0.
- Held-out verdict record precisely bound to candidate id, contract hash/revision, and content digest; `verification_evidence` has exactly one `held_out`/`pass` row.
- Isolation redaction checks pass: the implementation-side ContextPack projection carries no held-out locating fields, ref, digest, or content tokens; the redacted implementation signal carries no held-out anchors.

## 5. Workspace / credential disposal

- Codegen worktree removed on run teardown (asserted gone); verification worktree removed (asserted gone); no new `tally-codex-home-*` or `tally-p04-holdout-*` directories remained after the suite (before/after temp-dir diff asserted in test).
- Credential: read from `TALLY_E2E_CODEX_AUTH_FILE` (git-excluded `.e2e/deepseek-auth.json`), copied byte-only into a temp owner home, handed off per #22; never printed, never written to any tracked file, never logged. Tests SKIP when the env var is unset (AC5 path).

## 6. Findings (frozen adapter surfaces, for their owners — NOT changed by P0-04)

Both findings are empirically reproduced on codex-cli 0.153.0 (Windows, 2026-09-09) and block the P1-39 workflow kernel from completing a real turn; `runtimes/codex` behavior outside the ADR-0003 carve-out is frozen, so P0-04's E2E drives the production primitives directly (adapter session → worktree-fact candidate → runstore → orchestrator) and records claims as claims:

1. **Error-item classification (process.go `classifyTurn`):** codex emits a **non-fatal** `item.type=="error"` item (`"Model metadata for deepseek-v4-flash not found. Defaulting to fallback metadata..."`) at the start of **every** turn for any model name unknown to codex; the turn then completes normally (exit 0). The adapter classifies any error item as `RunFailed`, so the claim plane reports failed while the work factually completed. Suggested follow-up (owner of `runtimes/codex` classification): treat terminal `turn.completed`/last-message presence as authoritative over a leading metadata warning item, or whitelist that item text — an adapter-side decision, not P0-04's.
2. **Resume argv surface (`process.go buildExecSpec`):** `codex exec resume <id>` in 0.153.0 rejects the flags the adapter appends after the subcommand (clap: `unexpected argument '-C' found` → exit 2). Any same-session second turn therefore fails against the real CLI. Owner: P0-03/follow-up; flags would need to move before the `resume` subcommand or the resume path stay unused for this CLI version.
3. **Windows sandbox (harness-level, documented in `test/e2e/v14_codex_trust/harness_test.go`):** `workspace-write` on Windows mounts the workspace read-only and rejects **all** command execution ("blocked by policy", including `pwd`); `danger-full-access` is the least mode that lets a fixture needing writes+`go test` run. Mitigations: workspace is a disposable worktree (INV-EXEC-01), session home is a per-run isolated home (#22).

## 7. Validation record

| Command | Result |
| --- | --- |
| `go test ./runtimes/codex/...` | PASS (includes new ADR-0003 seam unit tests) |
| `go test ./test/runtimeconformance/...` | PASS |
| `go test -tags=integration ./test/e2e/v14_codex_trust/...` (DATABASE_URL + TALLY_E2E_CODEX_AUTH_FILE set) | PASS (~144s wall; canary 3 real turns + 1 real standard turn) |

Model usage bounded: 3 canary turns + 1 standard turn + diagnostic probes during bring-up; no other model consumption.
