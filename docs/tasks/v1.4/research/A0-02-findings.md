# A0-02 — Codex Session Isolation Spike  findings

**Date:** 2026-09-08
**Task:** `docs/tasks/v1.4/A0/A0-02.md`
**Evidence code:** `runtimes/codex/isolation_spike_test.go` (spike only; no production change)

## Verdict: PASS path identified

A production-viable per-run Codex home/session boundary exists:

```text
per-run CODEX_HOME (fresh empty directory)
+ hand off ONLY auth.json (credential material) into it
+ never copy/symlink sessions/, archived_sessions/, AGENTS.md, history
+ declare session.isolation only after the conformance canary passes
  against the real CLI
```

## RQ answers (from L3 pack)

| RQ | Answer | Evidence |
|---|---|---|
| RQ-01 observable CODEX_HOME state | `sessions/`, `archived_sessions/` rollouts and global `AGENTS.md` are protocol-observable instruction/session state, not cache | REF-CODEX-ROLLOUT, REF-CODEX-INSTR; `TestSpikeSharedHomeNegativeControl` reproduces cross-run retrieval |
| RQ-02 auth usable with unique per-run home | Yes — with explicit auth handoff | `TestSpikeIsolatedHomeCandidate` |
| RQ-03 file auth vs keyring under changed home | File auth (`CODEX_HOME/auth.json`) does not follow a new home (401 without handoff); keyring derives its key from the codex_home path, so per-run home changes keyring behavior too | REF-CODEX-AUTH; `TestSpikeIsolatedHomeWithoutAuthHandoff` |
| RQ-04 canary | Existing `test/runtimeconformance` Session Isolation Canary: inject unique token in a Verification-role session, attempt retrieval from fresh coding/debug sessions; any retrieval = isolation failure | `TestSpikeCanaryFakeVsCodex` |
| RQ-05 minimal handoff state | `auth.json` only (or an equivalent explicit credential handoff). Sessions/rollouts/instructions must NOT be handed off | `TestSpikeIsolatedHomeCandidate` |

## Spike results (all executed locally, 2026-09-08)

| Test | Result | Meaning |
|---|---|---|
| `TestSpikeCanaryFakeVsCodex` | PASS | Fake: declared+available; Codex: undeclared → unavailable with explicit reason. Same canary, honest gating |
| `TestSpikeSharedHomeNegativeControl` | PASS (leak reproduced) | Shared CODEX_HOME: run B retrieved run A's canary — the risk is real, not theoretical |
| `TestSpikeIsolatedHomeCandidate` | PASS | Per-run home + auth.json-only handoff: auth works, canary not retrievable |
| `TestSpikeIsolatedHomeWithoutAuthHandoff` | PASS | Isolated home without auth handoff fails with the probed 401 shape — authentication availability and session isolation are separate concerns |

## Constraints for P0-03 (implementation task)

1. `Config.Env["CODEX_HOME"]` already threads to the spawned process (`runtimes/codex/process.go:221`); the seam exists. P0-03 must construct the per-run home at run setup (worker/orchestration side owns lifecycle + cleanup), not by mutating global config.
2. The auth handoff copies **only** `auth.json`. Never copy/symlink the whole `CODEX_HOME` (pack §N likely-mistake).
3. `session.isolation` may be declared only after the conformance canary passes **against the real Codex CLI**; the substitute-CLI spike proves the mechanism, not the real binary.
4. Keyring-backed auth environments need an explicit decision: per-run home changes the derived keyring key, so either file auth must be present for handoff or the deployment must document keyring as unsupported for isolated runs.
5. Residual risk stands as declared by the canary: provider-internal opaque retention is out of proof scope (`runtimeconformance.ResidualRiskNote`).

## Real-CLI validation status

```text
NOT_RUN(codex CLI not installed in this environment; auth unavailable)
```

`codex` is not on PATH. Real-binary canary validation is deferred to P0-03/P0-04, which must rerun the conformance canary against the real CLI before declaring `session.isolation`.
