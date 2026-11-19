# @kaicreator/tally-provider-opencode

Tally **v2.1 candidate** — `AgentProvider/1` adapter for the **OpenCode CLI**
(`opencode-ai@1.19.6`), implemented in **Node**.

Status: **candidate, default-off**. This package is not registered anywhere
and grants no admission; the Host must explicitly select it via
`enabledProviderIds: ['opencode']`. All observations are untrusted; Task
admission, worktree grants, git candidate readback, review and Outcome remain
solely with the existing selected B/K/Host/C chain.

Evidence provenance: the CLI contract below was **field-verified on
opencode-ai@1.19.6 (2026-11-19)** and captured in **PR #471**
(`runtimes/opencode` Go runtime, its tests and fake CLI). The Go source is
**input evidence only** — this package never ports or bridges the Go worker;
v2 Tasks execute exclusively through the CLI process surface. The Node adapter
itself is the deliverable of #473 (sub-agent D).

## CLI contract map (all flags/env observed, nothing invented)

| Concern | Adapter behavior | Evidence |
| --- | --- | --- |
| One-shot turn | `opencode run --format json [--model provider/model] [--agent <name>] --pure "<instruction>"` — instruction last | #471 argv assembly + tests; 1.19.6 field runs |
| Never passed | `--attach` (shared-server face), `--standalone` (absent in 1.19.6), `--auto`, `OPENCODE_CONFIG` env (single config pipeline = flags) | #471 runtime docs + `run --help` observation |
| Process shape | without `--attach` each run starts its own temporary in-process local server on a random port; single process tree, zero residue after exit | 1.19.6 field observation (`tasklist`) |
| **PWD trap** | opencode resolves the project dir from the `PWD` env var, **not** the process cwd → adapter pins `PWD`/`OLDPWD` to the worktree, last-wins over `context.env`; spawn cwd = worktree | live-verified on Windows, 1.19.6 |
| Isolation | `XDG_CONFIG_HOME` / `XDG_DATA_HOME` / `XDG_CACHE_HOME` / `XDG_STATE_HOME` (+ platform `TMP`/`TEMP`) redirected to a fresh per-attempt temp tree (`tally-opencode-attempt-*` with `config/data/cache/state/tmp`), removed when the run ends; owner config/`auth.json`/sessions never read or written | `opencode debug paths` observation |
| Plugin face | `--pure` always passed (workspace-level plugin injection closed) | `run --help` observation |
| Credentials | flow only through the Host-supplied `context.env` (e.g. `DEEPSEEK_API_KEY`); never written to disk; **no MISSING_CREDENTIAL error is invented** — a keyless run falls back to the CLI's keyless default model and is classified by real exit code/output; probe gates `CREDENTIALS` by explicit env presence only | `providers list` observation (Credentials 0 + Environment 1) |
| stdout | NDJSON envelope `{type, timestamp, sessionID:"ses_*", part}`; observed types `step_start` / `text` / `tool_use` / `step_finish`; unknown types ignored (fail-safe); summary = last `text` part; usage = Σ `step_finish` tokens | #471 output observation |
| Degrade | exit 0 with bare non-JSON stdout → COMPLETED with the raw text as summary and zero usage (`degraded: true`) | plainok scenario (#471) |
| Exit codes | 0 + observed answer ⇒ `COMPLETED`; 0 + no answer ⇒ `FAILED/NO_OUTPUT` (fail-safe); non-zero ⇒ `FAILED/EXIT_NONZERO` (stderr excerpt only — partial stdout never masquerades as an answer); timeout ⇒ tree killed, `UNKNOWN/TIMEOUT_KILLED`; AbortSignal/`cancel()` ⇒ `CANCELLED`; spawn/version-gate environment failures ⇒ `UNKNOWN` (never a FAILED claim for the candidate) | #471 + v2 classification discipline |
| Version gate | live `--version` must print exactly `1.19.6` (fail-closed, cached successful observation only); mismatch/unobservable ⇒ probe `UNAVAILABLE`, execute `UNKNOWN` | #471 pin-gate |
| Cancellation | whole-tree kill: Windows `taskkill /pid <pid> /T /F`, POSIX process-group `SIGKILL`; pipe-holding orphans bounded by a 5 s wait delay | 1.19.6 `taskkill /T /F` zero-residue observation |
| Reconcile | no persistent execution journal exists ⇒ default `UNKNOWN/NOT_RECONCILED`; in-process in-flight attempts report `RUNNING`; an injectable `attemptRecords.get()` lookup is honored under the same contract normalization | contract: registry keeps no journal |
| Capabilities | honest empty set (streaming/resume/fork/model.select undeclared; `session.isolation` stays undeclared until the shared conformance canary passes) | #471 capability discipline |

## Probe ladder

1. `VERSION` — binary stat + live `--version` vs pinned `1.19.6`.
2. `CREDENTIALS` — `context.env.DEEPSEEK_API_KEY` present (explicit env only;
   adapter env is exactly the Host snapshot plus forced isolation keys —
   the parent process env is never merged in).
3. Conformance canary: **not executed** — `canary: 'NOT_RUN'` in evidence.

Any failure ⇒ `UNAVAILABLE` with a machine reason (`CLI_NOT_FOUND`,
`VERSION_UNOBSERVABLE`, `VERSION_MISMATCH`, `CREDENTIALS_MISSING`).

## Usage sketch

```js
import {createAgentProviderRegistry} from '@kaicreator/tally-agent-provider';
import {createOpenCodeAgentProvider} from '@kaicreator/tally-provider-opencode';

const opencode = createOpenCodeAgentProvider({
  cliPath: 'C:\\path\\to\\opencode.exe', // explicit, directly spawnable; no implicit discovery
  // model: 'deepseek/deepseek-v4-pro',  // optional --model expectation
  // agent: 'build',                     // optional --agent expectation
});

const registry = createAgentProviderRegistry([opencode]);
const provider = registry.resolve({
  providerId: 'opencode',
  enabledProviderIds: ['opencode'], // default-off: explicit opt-in required
});
```

Until a release wiring exists, the candidate resolves the contract package via
a workspace-relative import (`packages/tally-agent-provider`) and carries zero
npm dependencies; `npm test` needs no install.

## Tests

```bash
cd packages/tally-provider-opencode && npm test
```

All process behavior is exercised against `test/fake-opencode.mjs` (a Node
test double mirroring the observed 1.19.6 surface, launched via
`process.execPath`). **No real opencode process is ever spawned by the tests.**

## NOT_RUN (honest gaps)

- **Real opencode turn** — intentionally not executed here; the controller
  runs unified real-CLI acceptance for #473.
- **Live `--version` pin gate against the real CLI** — tests use the fake CLI
  only.
- **Session-isolation conformance canary** — not run; `session.isolation`
  therefore remains undeclared.
- **Keyless-default-model real routing** (real CLI fallback to `bigpickle`
  without credentials) — classified per evidence, not re-executed in Node.
- **Bad-model live classification** (`exit 1` + stderr structured error) —
  taken from #471 Go-side observation; Node-side covered by fake only.
