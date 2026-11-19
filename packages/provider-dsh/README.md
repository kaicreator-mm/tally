# @kaicreator/tally-provider-dsh

**V2_1_DSH_OPTIONAL_PROVIDER_CANDIDATE — NOT_A_V2_MVP_RELEASE_GATE.**

Isolated, default-off DeepSeek Harness (DSH) Agent Provider adapter for the
Tally v2 execution system, delivered as a v2.1 candidate (#64 requirement
track). It adds a third *optional* provider beside the Local Codex Host and
does not modify the v2 MVP: no R101 change, no second scheduler, no second
GitHub writer, no Business/Kernel route change (see `DSH_PROVIDER_SEAMS.md`
for the recorded integration seams).

## Governance positioning

- #392 freezes exactly ONE selected real Agent provider (Local Codex) for the
  v2 installed execution path: this package is **outside** that path and can
  never be selected implicitly.
- Selection is two-gate, default-off: `TALLY_ATTEMPT_PROVIDER=dsh` **and**
  `TALLY_DSH_ENABLED=1` (`resolveDshSelection`).
- Per `.dev-standard/PROJECT_OVERRIDES.md`, new isolated v2 packages carry no
  dedicated Woodpecker profile yet: run the package's own exact-SHA Node 24
  `npm test` and record it separately.

## What it implements

`createLocalDshHost` mirrors the Local Codex Host contract
(`experiments/v2/selected-business-build/local-codex-host.mjs`):

- fail-closed capability probe: `dsh --version` must equal the pinned version
  exactly (`0.2.2-rc.5`, npm latest as of 2026-11-19) and `DEEPSEEK_API_KEY`
  must be configured; mismatches refuse before any spawn;
- one disposable detached Git worktree per attempt; the Host (never the
  agent) creates the candidate `write-tree`/`commit-tree`/`update-ref` and
  re-reads it independently (`verifyCandidateReadback`);
- exact write-set enforcement after the run: exactly one changed path equal
  to the configured candidate note, otherwise `WRITE_SET_VIOLATION`;
- per-attempt throwaway `DSH_HOME` (sessions/storages/profiles never cross
  attempts), `DSH_TELEMETRY_MODE=DISABLED`, and a `--patch` overlay that
  disables the default-on DeepSeek session-log upload
  (`session-log-deepseek.enabled=false`, source-derived from the 0.2.2-rc.5
  config catalog; the smoke receipt labels its live-verification status);
- honest failure classification: cancel > orphan-pipe > write-set > nonzero
  exit > empty-final-answer; timeout/lost-ACK/ambiguous restart is durable
  `UNKNOWN` and never relaunches;
- a Go `WaitDelay` equivalent (output-pipe grace) that is *wired*, not just
  declared — tests assert the pipe ends are actually destroyed (the P4-04
  historical bug was exactly this missing one-line wiring);
- CAS-durable attempt records with atomic resource grants whose key
  derivation is byte-identical to the Codex Host, so DSH and Codex attempts
  can never hold the same worktree or candidate path concurrently.

## CLI contract (source-verified against @deepseek-ai/dsh 0.2.2-rc.5)

```
dsh --version
dsh --profile headless [--patch <yml>] "<task>"
```

- workspace root = process cwd (the adapter pins `cwd` to the worktree);
- stdout = final answer text; **exit 0 only for completed turns**;
- SIGTERM drains gracefully and exits 0, so cancellation is classified from
  the adapter's own intent, never from exit codes;
- credentials: `DEEPSEEK_API_KEY` in the launch environment (headless has no
  login store); the value is never logged or recorded by this package;
- `DSH_PERMISSION_MODE`: `read-only | workspace-write | danger-full-access`
  (danger-full-access additionally requires `allowDangerousFullAccess`).
  The CLI sandbox is a restriction layer, never the Host isolation boundary —
  real Host isolation is the OS-principal confinement proof
  (`proveConfinement`) plus the sandbox wrapper (`sandboxCliPath`), exactly as
  for Codex. Without `sandboxCliPath` commands spawn directly, which on a
  real Host fails closed at the principal check.

## Usage

```js
import {openSqliteCasStore} from '@kaicreator/domain-harness-host-node';
import {createLocalDshHostWithDhStore,resolveDshSelection,dshConfigFromEnv}
  from '@kaicreator/tally-provider-dsh';

const selection=resolveDshSelection(process.env);      // default: NOT_SELECTED
const config=dshConfigFromEnv(process.env);
// Pass the public DH store explicitly: ESM dynamic import resolves from the
// provider module's own location, so only the sealed-consumer lane (where
// this source is bundled next to the installed host-node package) may rely
// on the import fallback.
const host=await createLocalDshHostWithDhStore({openSqliteCasStore});
const result=await host.runAttempt(config);
// result.state: LOCAL_CANDIDATE_READY | LOCAL_ATTEMPT_FAILED |
//               LOCAL_ATTEMPT_CANCELLED | LOCAL_ATTEMPT_UNKNOWN |
//               AGENT_NOT_EXECUTED | NOT_ADMITTED
```

`createLocalDshHost` accepts `{spawnFn, operations, now, killTree, openStore}`
injection for tests and alternative wiring; `openStore` is required (the
public DomainHarness CAS store is injected, keeping this package dependency-
free).

## Environment variables

| Variable | Default | Meaning |
| --- | --- | --- |
| `TALLY_ATTEMPT_PROVIDER` | – | must be `dsh` (gate 1) |
| `TALLY_DSH_ENABLED` | – | must be `1` (gate 2) |
| `TALLY_DSH_BIN_PATH` | `dsh` on PATH | CLI path; on Windows prefer pointing at a real executable |
| `TALLY_DSH_BIN_ARGS_PREFIX` | – | `\|`-joined argv prefix (e.g. the `lib/bin.js` entry when `binPath` is node) |
| `TALLY_DSH_PINNED_VERSION` | `0.2.2-rc.5` | exact version pin |
| `TALLY_DSH_PERMISSION_MODE` | `workspace-write` | `read-only \| workspace-write \| danger-full-access` |
| `TALLY_DSH_ALLOW_DANGER_FULL_ACCESS` | – | must be `1` for `danger-full-access` |
| `TALLY_DSH_TIMEOUT_MS` | `300000` | attempt timeout |
| `TALLY_DSH_PIPE_GRACE_MS` | `5000` | output-pipe grace after process exit |
| `DEEPSEEK_API_KEY` | – | credential, presence-probed only |
| `TALLY_DSH_SANDBOX_CLI_PATH` | – | Host sandbox wrapper (Codex sandbox CLI) |

## Tests

```bash
npm test        # node --test test/*.test.mjs (Node >= 24; no npm dependencies)
```

Contract tests use an injected in-memory CAS double plus a fake DSH CLI that
is spawned for real (`node test/fake-dsh.mjs`), covering: CLI missing /
version drift / missing credential / confinement failure / happy path with
candidate readback / write-set violation / cancel / timeout / lost-ACK /
orphan pipes / empty final answer / secret redaction / grant races /
cross-provider exclusion.

## Real smoke (optional, gated)

```bash
# isolated install (no global mutation):
mkdir -p /tmp/dsh-smoke && cd /tmp/dsh-smoke &&
  npm install @deepseek-ai/dsh@0.2.2-rc.5 --registry=https://registry.npmmirror.com
TALLY_DSH_SMOKE_CONFIRM=1 \
TALLY_DSH_NODE="$(which node)" \
TALLY_DSH_BIN=/tmp/dsh-smoke/node_modules/@deepseek-ai/dsh/lib/bin.js \
  node smoke/run-dsh-smoke.mjs
```

Prints a JSON receipt; never prints the credential. Without the gates it
reports `REAL_DSH_SMOKE: NOT_RUN` and exits 0.
