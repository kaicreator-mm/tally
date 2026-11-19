# @kaicreator/tally-provider-dsh-adapter

**V2_1_CANDIDATE — DEFAULT-OFF — NOT_A_V2_MVP_RELEASE_GATE.**

AgentProvider/1 contract adapter for the DeepSeek Harness (DSH) CLI, adapted
from the #467 Node DSH Provider package (`packages/provider-dsh` on
`refs/pull/467/head`, whose CLI facts, working-directory behavior, credential
isolation and error classification were verified against the real
`@deepseek-ai/dsh` 0.2.2-rc.5 — see that package's `DELIVERY_RECEIPT.md`,
`DSH_PROVIDER_SEAMS.md` and its own test suite). Delivered under the #473
provider-unification effort (`Refs: #473 #474 #467`).

Zero npm dependencies (Node >= 24; the real CLI additionally requires
`engines.node >= 24.9.0` per the @deepseek-ai/dsh 0.2.2-rc.5 metadata).

## Selection: registry-gated, DEFAULT-DISABLED

The authoritative door is the AgentProvider/1 registry: the provider is
installed under id `dsh` but is **never** resolved unless the Host passes an
`enabledProviderIds` list that explicitly contains `dsh`:

```js
import {createAgentProviderRegistry} from '@kaicreator/tally-agent-provider';
import {createDshAgentProvider} from '@kaicreator/tally-provider-dsh-adapter';

const dsh=await createDshAgentProvider({hostEnv, config});
const registry=createAgentProviderRegistry([codexProvider, dsh]);

registry.resolve({providerId:'dsh'});                          // PROVIDER_DISABLED
registry.resolve({providerId:'dsh',
  enabledProviderIds:['codex','dsh']});                        // the dsh adapter
```

There is **no fallback**: an enabled list without `dsh` refuses with
`PROVIDER_DISABLED` and never silently substitutes the default provider.

The #467 env two-gate (`TALLY_ATTEMPT_PROVIDER=dsh` **and**
`TALLY_DSH_ENABLED=1`) is retained as **defense-in-depth only**: if a Host
sets either legacy env field, both must agree before the adapter acts
(`DSH_ENV_GATE_CLOSED` otherwise). A Host that never touches those fields is
governed by the registry gate alone.

## Contract surface

`createDshAgentProvider(deps)` returns a frozen AgentProvider/1 provider
(bound via `defineAgentProvider` from `@kaicreator/tally-agent-provider`;
injectable for tests, resolved by dynamic import from the sibling contract
package otherwise — same injected-dependency discipline as #467's
`openStore`).

deps: `{spawnFn, killTree, now, hostEnv, config, defineAgentProvider}` —
all optional except that `hostEnv` defaults to `{}` (the adapter never reads
`process.env`; credentials must come from the Host-supplied `context.env`).

### probe(context)

1. **VERSION** — `dsh --version` must equal the pinned version exactly
   (`0.2.2-rc.5`); missing binary or drift ⇒ `UNAVAILABLE /
   LOCAL_AGENT_UNAVAILABLE` with `stage:'VERSION'`, never a spawn;
2. **CREDENTIALS** — `DEEPSEEK_API_KEY` must be present in the Host-supplied
   context env (the value is never read from `process.env`, never logged,
   never recorded);
3. **CANARY** — optional bounded canary; **default OFF, reported
   `canary:'NOT_RUN'`**: the probe makes NO real DSH turn (real turns are
   controller-owned acceptance, #473).

All three stages before any failure ⇒ `AVAILABLE`.

### execute(request, context)

One-shot `dsh --profile headless --patch <overlay> "<prompt>"` process,
`cwd` pinned to the Host-supplied `request.worktreePath` (workspace root =
process cwd). Outcome classification (order preserved from #467):

| Condition | State | Reason |
| --- | --- | --- |
| timeout / spawn error | `UNKNOWN` | `ATTEMPT_TIMEOUT_RECONCILED` / `ATTEMPT_STATUS_UNKNOWN` |
| adapter cancellation intent (AbortSignal or `cancel()`) | `CANCELLED` | `ATTEMPT_CANCELLED` |
| orphan pipe after grace | `FAILED` | `DSH_OUTPUT_PIPE_NOT_CONVERGED` |
| nonzero exit | `FAILED` | `ATTEMPT_PROCESS_FAILED` |
| exit 0 + empty stdout | `FAILED` | `DSH_EMPTY_FINAL_ANSWER` |
| otherwise | `COMPLETED` | `REAL_DSH_ATTEMPT_COMPLETED` |

- **Cancellation is classified from the adapter's own intent, never from
  exit codes**: the real 0.2.x CLI drains SIGTERM gracefully and exits 0
  (#467-verified trap); tests assert an exit-0 close still yields
  `CANCELLED` once cancel was intended.
- Timeout kills the process tree (Windows `taskkill /T /F`; POSIX detached
  process-group `SIGKILL`) and reports honest `UNKNOWN`.
- `spawnCapture` carries the Go `WaitDelay` equivalent: orphan descendants
  holding the output pipes past `pipeGraceMs` get the host pipe ends
  destroyed so the wait converges, then fail closed
  (`DSH_OUTPUT_PIPE_NOT_CONVERGED`) — the mechanism is *wired*, asserted by
  tests (the historical P4-04 bug was exactly this missing wiring).
- Evidence is untrusted, adapter-redacted, tail-truncated material:
  `exitCode`, `pid`, `timedOut`, `cancelled`, `orphanPipe`,
  `permissionMode`, `pinnedVersion`; `stdoutSha256`/`stdoutBytes`/
  `stdoutTail` (200 runes) on COMPLETED; `stderrTail` (500 runes) on
  FAILED/UNKNOWN. The credential value is used as a redaction secret and
  never appears in any observation, log or record.
- Refusals before spawn map to `FAILED` with `LOCAL_AGENT_UNAVAILABLE`
  (probe failed: version/credential), `DSH_CONFIG_INVALID` (request shape,
  illegal `DSH_PERMISSION_MODE`, ungated `danger-full-access`),
  `PROVIDER_WORKSPACE_UNAVAILABLE` (missing Host worktree),
  `DSH_ENV_GATE_CLOSED` (defense-in-depth), `ATTEMPT_ALREADY_ACTIVE`.

### cancel(attempt) / reconcile(attempt)

- `cancel` on an in-flight attempt marks the adapter's cancellation intent
  **before** killing, terminates the tree ⇒ `CANCEL_REQUESTED`; otherwise
  `NOT_RUNNING`.
- `reconcile` reports `RUNNING` only for attempts this process still holds;
  anything else is honest `UNKNOWN / ATTEMPT_STATUS_UNKNOWN` with
  `evidence.durableJournal:false` — the one-shot process model keeps no
  durable execution journal (durable attempt records stay with the Host
  chain, as in #467's CAS store, which is intentionally NOT part of this
  adapter).

## What is deliberately NOT in this adapter

Under AgentProvider/1, task admission, worktree acquisition/grants,
OS-principal confinement proof, candidate write-tree/commit-tree/readback,
review and Outcome remain solely with the existing selected B/K/Host/C
chain. The #467 Host functions for those (`prepareWorktree`,
`proveConfinement`, `finalizeCandidate`, `verifyCandidateReadback`, atomic
`tally:v2:local-agent-grant:` grants, CAS-durable attempt records) are
therefore NOT ported; the Host supplies an already admitted workspace and
this adapter owns only CLI execution and reporting. Wire the #467 host
package directly where those duties are needed.

## Behavior retained from #467 (verified points)

- pinned-version fail-closed probe (`0.2.2-rc.5`, one bare version line);
- credential = `DEEPSEEK_API_KEY` presence only, injected exclusively into
  the allowlisted execution environment;
- credential-free allowlisted builder env (no GitHub/SSH/token material;
  `GIT_*` hygiene; `DSH_TELEMETRY_MODE=DISABLED`; proxy opt-in);
- per-attempt throwaway `DSH_HOME` (fresh `mkdtemp` per execute; sessions,
  storages and profiles never cross attempts) with the `--patch` overlay
  disabling the default-on DeepSeek session-log upload (top-level YAML
  array form, live-verified in #467 §5.2);
- `DSH_PERMISSION_MODE` three values with the explicit
  `danger-full-access` allowance gate;
- headless argv shape, `cwd` = workspace root, stdout = final answer,
  exit 0 only on completed turns;
- SIGTERM-drain-exit-0 cancellation trap (intent-based classification);
- `spawnCapture` pipe-grace (WaitDelay equivalent) wiring;
- process-tree termination defaults for win32/POSIX;
- exit 0 + empty stdout fails safe; reason order as documented above.

## Tests

```bash
npm test        # node --test test/*.test.mjs (Node >= 24; no npm dependencies)
```

Matrix: probe AVAILABLE / version drift / missing binary (no spawn) /
missing credential (no headless spawn); defense-in-depth env gate; happy
one-shot turn with real fake-CLI spawn contract observation (cwd, env
isolation, argv, overlay patch); nonzero exit; leaked-key redaction;
empty-final-answer; SIGTERM-exit-0 trap (pre-aborted signal and in-flight
`cancel()`); timeout tree-kill (injected + real process); orphan-pipe
wiring; reconcile honesty; registry default-disabled / no-fallback /
pin-mismatch / duplicate id; liar-observation rejection; credential-free
evidence across COMPLETED/FAILED/CANCELLED/UNKNOWN.

## Honest NOT_RUN list (this delivery)

- **Real DSH turn**: not executed here (controller-owned acceptance per
  #473); the probe's CANARY is bounded and OFF by default
  (`evidence.canary:'NOT_RUN'`).
- **Real DeepSeek API call**: none; all tests run against the ported fake
  CLI (`test/fake-dsh.mjs`) or injected fake children.
- **Real `dsh --version` probe**: not exercised here (tests use the fake
  CLI); the pin itself was live-verified in #467 against 0.2.2-rc.5.
- **Real Windows-principal confinement / sandbox-wrapper execution**: not
  exercised here (Host-duty, #467 §5.3 receipt stands for the machinery
  this adapter ported).
- **Durable reconcile**: not implemented (no CAS store in the adapter);
  `UNKNOWN / ATTEMPT_STATUS_UNKNOWN` + `durableJournal:false` is the honest
  contract answer.
- **CI carrier**: per the #467 CI row (`.dev-standard/PROJECT_OVERRIDES.md`),
  no dedicated Woodpecker profile for this package; the exact-SHA Node 24
  package `npm test` receipt accompanies the delivery report.
