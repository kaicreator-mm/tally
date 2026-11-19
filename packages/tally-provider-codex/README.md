# @kaicreator/tally-provider-codex

Codex **AgentProvider/1** bridge adapter (#473). Wraps the existing real Codex
execution Host
(`experiments/v2/selected-business-build/local-codex-host.mjs`) as a
`@kaicreator/tally-agent-provider` provider **without re-implementing any host
authority**.

- **Status**: v2.1 candidate, `0.0.0-v2.1-candidate`, **default-off**. Not part
  of the v2 MVP release gate. Nothing wires this package into a pipeline yet;
  adoption requires an explicit `enabledProviderIds` selection by the Host.
- The bridge owns exactly three things: host terminal-state mapping,
  request/context adaptation, observation normalization/redaction. It contains
  no authorization, no CAS store, no sqlite, no git and no candidate
  qualification logic. Task admission, worktree grants, git candidate
  readback, review and Outcome remain solely with the existing
  selected B/K/Host/C chain.
- **Zero dependencies**: node built-ins plus a relative import of the contract
  package (`../../tally-agent-provider/src/index.mjs`).

## Lazy host loading

The host module imports `@kaicreator/domain-harness-host-node` at its top
level, which is not installed in every workspace. The bridge therefore loads
it lazily inside the factory via an injectable `loadHost` (default:
`defaultLoadHost()`, a dynamic `import()` resolved at
`../../../experiments/v2/selected-business-build/local-codex-host.mjs`
relative to `src/index.mjs`). A failed load is never a crash: it degrades to
`UNAVAILABLE/HOST_MODULE_UNAVAILABLE` (probe) or
`UNKNOWN/HOST_MODULE_UNAVAILABLE` (execute / reconcile with a replayable
config). All tests inject a fake host module; the suite passes without the
external package installed, which also proves the host is never loaded
statically.

## Usage

```js
import {createCodexAgentProvider, createAgentProviderRegistry} from '../tally-agent-provider/src/index.mjs';

const provider = createCodexAgentProvider({
  // loadHost: async () => await import('.../local-codex-host.mjs'), // optional
  config: {
    hostStorePath: 'C:\\tally\\host.sqlite',   // host CAS store (host-owned)
    repositoryPath: 'C:\\tally\\repo',
    cliPath: 'C:\\tools\\codex.exe',
    sandboxCliPath: 'C:\\tools\\codex.exe',
    // ...remaining host attempt fields (builder identity, hidden fixture, ...)
  },
});
const registry = createAgentProviderRegistry([provider]);
// explicit selection only — the registry default is codex, but every non-default
// provider needs explicit enabledProviderIds; there is no implicit fallback.
```

Per-attempt fields come from the trusted Host at `execute()` time:
`attemptId`, `sourceSha`, `worktreePath`, `timeoutMs`, `signal`. The bridge
supplies its own `AbortController` per execution and forwards host-side
cancellation through it.

## State mapping table (host → contract, tested item by item)

### probe

| Host `probeProvider` result | Contract observation |
| --- | --- |
| `{available: true, provider}` | `AVAILABLE` (evidence `{provider}`) |
| `{available: false, reason, stage, exitCode}` | `UNAVAILABLE` + host reason verbatim (evidence `{stage, exitCode}`) |
| malformed probe result | `UNKNOWN` + `PROBE_OBSERVATION_INVALID` |
| host module cannot be loaded | `UNAVAILABLE` + `HOST_MODULE_UNAVAILABLE` |
| factory config missing | `UNAVAILABLE` + `PROVIDER_CONFIG_UNAVAILABLE` |
| `probeProvider` throws | `UNKNOWN` + `PROBE_EXECUTION_FAILED` |

### execute (`runAttempt` terminal states, host vocabulary verbatim)

| Host state | Contract state | reason |
| --- | --- | --- |
| `LOCAL_CANDIDATE_READY` | `COMPLETED` | `null` |
| `LOCAL_ATTEMPT_FAILED` (`WRITE_SET_VIOLATION`, `ATTEMPT_PROCESS_FAILED`, candidate error code) | `FAILED` | host reason verbatim |
| `LOCAL_ATTEMPT_CANCELLED` (`ATTEMPT_CANCELLED`) | `CANCELLED` | host reason verbatim |
| `LOCAL_ATTEMPT_UNKNOWN` (`ATTEMPT_STATUS_UNKNOWN`, `ATTEMPT_TIMEOUT_RECONCILED`) | `UNKNOWN` | host reason verbatim |
| `AGENT_NOT_EXECUTED` (`LOCAL_AGENT_UNAVAILABLE`, `BUILDER_CONFINEMENT_UNAVAILABLE`, `HIDDEN_VERIFIER_UNAVAILABLE`, stored reason) | `FAILED` | host reason verbatim |
| `NOT_ADMITTED` (`CANDIDATE_WORKTREE_OVERLAP`, `CANDIDATE_WRITE_GRANT_OVERLAP`) | `FAILED` | host reason verbatim |
| any other/future host state | `UNKNOWN` + `HOST_STATE_MAPPING_MISS` | fail-closed |
| host throws a coded error (e.g. `CANDIDATE_WORKTREE_ALREADY_EXISTS`) | `UNKNOWN` | error code, else `HOST_EXECUTION_ERROR` |
| host module cannot be loaded | `UNKNOWN` + `HOST_MODULE_UNAVAILABLE` | |
| cancellation before the host launched | `CANCELLED` + `CANCELLED_BEFORE_LAUNCH` | bridge-local; nothing durable was recorded and the host is never invoked |

`AGENT_NOT_EXECUTED` and `NOT_ADMITTED` are deterministic terminal
non-successes durably recorded by the host, so they map to `FAILED` (never
`UNKNOWN`); `UNKNOWN` is reserved for genuinely undeterminable outcomes that
need reconciliation.

Host reasons are passed through only when they already satisfy the contract
reason shape (`/^[A-Z][A-Z0-9_]{0,95}$/`); anything else degrades to
`HOST_REASON_UNSAFE` (UNKNOWN) or `null` (FAILED).

### cancel

The host has no dedicated cancel API — cancellation is the `AbortSignal` path
of `runAttempt` (the host terminates the process tree and durably records
`LOCAL_ATTEMPT_CANCELLED`). The bridge maps:

| Situation | Contract observation |
| --- | --- |
| in-flight attempt, pin matches | `CANCEL_REQUESTED`; execute then resolves `CANCELLED`/`ATTEMPT_CANCELLED` |
| in-flight attempt, pin mismatch (taskId/sourceSha/providerId) | `UNKNOWN` + `ATTEMPT_PIN_MISMATCH` (no abort) |
| no in-flight handle for that attemptId | `NOT_RUNNING` + `NO_INFLIGHT_ATTEMPT` (the bridge cannot know the durable record state and does not guess) |
| malformed pin | `UNKNOWN` + `ATTEMPT_PIN_INVALID` |

### reconcile

Reconcile replays through `runAttempt`, which is read-only for any attempt
that already has a durable record (the host never relaunches an existing
record). The bridge replays only configs it can vouch for: the config recorded
when `execute()` launched the attempt in this process, or one explicitly
supplied by the trusted Host via `context.attemptConfig` (pin identity is
re-checked). With neither, it fails closed instead of risking a launch side
effect.

| Host replay result | Contract observation |
| --- | --- |
| `LOCAL_CANDIDATE_READY` | `COMPLETED` |
| `LOCAL_ATTEMPT_UNKNOWN` + `process.status === 'RUNNING'` | `RUNNING` |
| `LOCAL_ATTEMPT_UNKNOWN` otherwise | `UNKNOWN` + host reason |
| `LOCAL_ATTEMPT_FAILED` | `FAILED` + host reason |
| `LOCAL_ATTEMPT_CANCELLED` | `CANCELLED` + host reason |
| `AGENT_NOT_EXECUTED` | `FAILED` + host reason |
| `NOT_ADMITTED` | `UNKNOWN` + host reason (the attempt's own record state could not be established) |
| unknown state | `UNKNOWN` + `HOST_STATE_MAPPING_MISS` |
| no replayable attempt config | `UNKNOWN` + `RECONCILE_STATE_UNAVAILABLE` |
| host module cannot be loaded | `UNKNOWN` + `HOST_MODULE_UNAVAILABLE` |

The host's own replay collapses FAILED/CANCELLED/UNKNOWN/RUNNING records into
`LOCAL_ATTEMPT_UNKNOWN` and preserves only `process.status`; the bridge never
fabricates `FAILED`/`CANCELLED` for a replay the host could not classify.

## Evidence policy

Execute/reconcile evidence is adapter-redacted to **execution observations
only**: `agentLaunch`, `exitCode`, `pid`, `startedAt`, `terminatedAt` and the
derived `durationMs`. Candidate git facts (`candidateTree`,
`candidateCommit`, `candidateRef`) are deliberately dropped: provider results
are untrusted observations, and the Host independently reads the exact
candidate back from the repository (`verifyCandidateReadback`) before any
review decision. Authority fields (`taskPass`, `reviewPass`, `releasePass`,
`claimAccepted`, `authorized`, `candidateCommit` as a top-level field) are
structurally impossible: the contract normalizer rejects unexpected fields,
and the redaction keeps them out of evidence even when a host result carries
them.

## Intentionally ignored inputs

- `request.prompt`: the local Codex host owns the candidate-note prompt bound
  to its write-set protocol; a caller-supplied prompt is never forwarded.
- `context.env`: the host builder environment allowlist is the only authority
  for what the agent process sees; nothing is merged.

## In-memory state (and why it is not a journal)

The bridge keeps only: one memoized host-module load promise,
`AbortController` handles for execute calls currently in flight, and the
per-attempt host configs needed to ask the host for a replay. It stores no
results and makes no decisions from them — durable attempt state lives solely
in the host CAS store.

## Tests

```bash
cd packages/tally-provider-codex && npm test
# exit code 0; 28 tests, all fake-host injected, no real CLI touched
```

## Honest NOT_RUN list

Real-environment behaviors were **NOT_RUN** in this round (fake-host injection
only); the existing real Codex evidence is reused instead of re-executed:

1. Real `codex-cli` execution through `runAttempt` (spawn, exit, candidate
   write) — reuse: `experiments/v2/selected-business-build/LOCAL_CODEX_L3.md`
   receipts and the t06m CI lane.
2. Real Windows builder confinement (principal/SID/ACL/denied reads) — reuse:
   same L3 receipts; the bridge maps confinement failures verbatim but did not
   exercise them.
3. Real CAS store persistence and restart replay against SQLite — the fake
   host mimics the replay surface only.
4. Real `@kaicreator/domain-harness-host-node` module load — not installed in
   this workspace; the failure path is tested via injected rejection.
5. Concurrent multi-provider execution through a live registry in a real
   pipeline — the package is not wired anywhere yet (default-off).
