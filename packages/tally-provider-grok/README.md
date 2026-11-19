# Tally Provider Grok (v2.1 candidate, default DISABLED)

AgentProvider/1 Node adapter for the xAI official **Grok Build CLI**
(`@xai-official/grok`, native Rust binary; bin: `grok`). Source-only candidate
for [#473](https://github.com/kaicreator-mm/tally/issues/473), translating the
observed CLI contract of Go candidate PR [#468](https://github.com/kaicreator-mm/tally/pull/468)
(`runtimes/grok/`) into Node **without Go bridging**.

**Status: DISABLED by default.** The registry default provider stays `codex`;
`grok` never appears in any default enabled list, and the authoring machine
holds **no real Grok credentials**, so `probe` reports `UNAVAILABLE` /
`GROK_NOT_AUTHENTICATED` and `execute` refuses to spawn. Nothing here fakes a
real execution PASS.

## Evidence source (observed CLI facts)

All CLI surfaces are observations carried from #468, taken on the pinned
binary **grok 1.0.87 (commit 929e6dd187b1, win32-x64, observed 2026-11-19)**:

| Observed fact | Adapter behaviour |
|---|---|
| One execute = one one-shot process `grok --no-auto-update [-m <model>] [--permission-mode <mode>] --single=<instruction>` | exact argv shape; zero overlay (flags only when configured) |
| Attached `--single=<text>` form is safe for any text; bare `-p "-dash"` is misparsed by clap (exit 2) | instruction always rides `--single=` as a single argv element; `-p` never emitted |
| stdout = final answer (plain); exit 0 completed / 1 runtime error incl. auth / 2 usage | `COMPLETED` (answer in redacted evidence) / `FAILED` (`GROK_RUN_FAILED`, stderr summary) / `FAILED` (`GROK_USAGE_ERROR`) |
| Auth = `XAI_API_KEY` env; keyless observed: empty stdout, stderr `Not signed in. ...`, exit 1 | probe `UNAVAILABLE`+`GROK_NOT_AUTHENTICATED` without the key; execute typed refusal **before spawn**; forced-keyless stand-in run maps exit 1 to `FAILED` with the observed stderr text |
| `--no-auto-update` mandatory for scripted environments (hidden flag observed accepted) | always passed |
| `GROK_HOME` redirects the per-attempt home (empty temp home observed usable, no `~/.grok` writes) | fresh temp `GROK_HOME` per attempt, forced over any same-named key, removed after the run |
| Strict spawn-env allowlist (OS base keys; other owner credentials structurally unreachable) | 1:1 allowlist from #468 `isolation.go`; tested with a planted `DEEPSEEK_API_KEY` that never reaches the CLI |
| npm dist-tag latest stale (0.1.4) vs real 1.0.x line | explicit `binPath` required (no implicit discovery); pinned `1.0.87` version probe, fail-closed on mismatch/unobservability |
| cancel = whole-process-tree termination (taskkill `/T /F` stand-in proof; in-flight real cancel unobserved) | `taskkill /T /F` (Windows) / process-group kill (POSIX); `CANCELLED` on cancel/abort, `UNKNOWN`+`GROK_RUN_TIMEOUT` on timeout |

Capabilities stay the **honest empty set** (no streaming, resume, fork,
structured output, tool approval, native sandbox, model report has been
observed). Workspace pinning is the spawn cwd (the `-c/--cwd` flag's effect
was never observed; not used).

## Honesty ledger

- `REAL_EXECUTION=NOT_RUN` — no `XAI_API_KEY` on this machine; no real grok
  binary was spawned; **no real completion turn has ever been observed by
  this adapter**. The keyless refusal (`execute` throws
  `GrokProviderRefusalError`(`GROK_NOT_AUTHENTICATED`) before any spawn) is
  itself an asserted behaviour backed by #468's observed keyless contract.
- The pinned version probe only proves the stand-in/binary identity shape;
  with a real binary it proves the pin gate. It is **not** an execution
  receipt.
- No canary was run (nothing to run it against without credentials).
- All process-level tests run against `test/fake-grok.mjs`, a stand-in
  mimicking only the observed surface (ported from #468's
  `testdata/fakegrok/main.go`), at a real process boundary — no in-memory
  mocks for argv/env/exit/cancel paths.
- Provider observations are claims, never Task/Review/Release facts; the
  contract normalization refuses authority fields, `UNKNOWN` always carries a
  reason, there is no fallback path.

## Test matrix (27 tests, all green)

`npm test` → `node --test test/*.test.mjs` (Node 24, no dependencies):

- argv shape: `--no-auto-update` + attached `--single=` (incl. leading-dash
  instruction), zero overlay, model/`--permission-mode` pass-through, never
  bare `-p`; cwd pinning proven by the stand-in really reading a file from
  the spawn cwd.
- env allowlist: Host `XAI_API_KEY` reaches the CLI; owner `DEEPSEEK_API_KEY`
  does not; `GROK_HOME` forced fresh per attempt (context-env override
  defeated), removed after the run.
- probe: AVAILABLE (pin + credential), `GROK_NOT_AUTHENTICATED`,
  `GROK_VERSION_MISMATCH` (fail closed), `GROK_PROBE_FAILED`
  (unobservable/missing binary).
- execute mapping: exit 0 → `COMPLETED` (answer trimmed + redacted); exit 0
  without answer → `FAILED`/`GROK_EMPTY_OUTPUT` (fail safe); exit 1 keyless
  stderr → `FAILED`/`GROK_RUN_FAILED`; exit 2 → `GROK_USAGE_ERROR`; exit 7
  crash → `FAILED`, partial stdout never surfaces; CLI-echoed credential →
  `[REDACTED]` in evidence.
- lifecycle: keyless typed refusal with zero spawn; timeout → tree killed,
  `UNKNOWN`+`GROK_RUN_TIMEOUT`; AbortSignal → `CANCELLED`;
  `cancel()` → `CANCEL_REQUESTED`/`NOT_RUNNING`, whole-tree teardown proven by
  pid liveness polling; `reconcile` → `UNKNOWN`+`NOT_RECONCILED`.
- contract: registry default is `codex` and grok resolves only behind an
  explicit `enabledProviderIds` + matching attempt pin (`PROVIDER_DISABLED`
  otherwise); pin rebinding refused; liar/authority fields refused;
  construction fails closed on blank `binPath`, non-enum permission mode,
  blank model.

## Enablement conditions (all required before flipping on)

1. A real `XAI_API_KEY` credential provisioned through the trusted Host
   context env (never stored in config, never on disk).
2. Explicit enablement: `enabledProviderIds` containing `grok` plus a
   matching `createProviderAttemptPin` (the registry refuses otherwise).
3. The real grok 1.0.87 binary at an explicit `config.binPath` passing the
   pinned version probe (a drifted CLI fails closed; npm dist-tag is not a
   version source).
4. A recorded real-execution receipt: at least one real credentialed turn
   (harmless echo instruction) plus a shared conformance canary before any
   capability declaration beyond the empty set is even considered.

Until then this package is a source-only candidate: it does not mount into
the v2 Host, is not a Core MVP or Release Qualification gate, and a
`COMPLETED` observation would still only ever be an untrusted provider claim.
