# @kaicreator/tally-provider-zcode

Tally v2.1 **candidate** AgentProvider/1 adapter for the ZCode agent CLI.
**Default-off**: nothing in the product registers this package; it becomes
usable only when a Host explicitly installs it into
`createAgentProviderRegistry(...)` and selects it via `enabledProviderIds`
(the default provider stays `codex`). There is no fallback path.

Ported from the #470 Windows-proven ZCode Node Provider
(`web/v2/zcode-provider.mjs`), reshaped onto the AgentProvider/1 contract
(`packages/tally-agent-provider`).

## Verification scope (honest)

- **Real ZCode execution: NOT_RUN in this change.** The real Windows turn
  receipt is reused from #470 verbatim — candidate commit
  `434ac166716163507460b78a527a43144eb6fba0` (parent
  `dff5329eed76b88b768aa6802fd92e64fbc83881`), CLI core `0.16.9` on runtime
  `3.14.3` via node v24.21.0, process pid 19192 exit 0, ~14.5 min turn,
  independent git readback verified (`ZCODE_PROVIDER_V21_CANDIDATE.md`).
  The process machinery (argv, env allowlist, probe staging, parsing,
  timeout/kill, cancel) is ported **verbatim**; only the result-shaping layer
  is new.
- **Newly verified here: the interface differences only** — contract
  compliance (defineAgentProvider wrapping, registry/resolve/pin/default
  semantics, no fallback), contract-observation state mapping, the new
  `cancel`/`reconcile` operations, and the Host-admitted-worktree rule.
  All via injected-fake process matrices (`npm test`); no real zcode turn.

## Contract mapping (AgentProvider/1)

| Operation | States produced | Notes |
| --- | --- | --- |
| `probe()` | `AVAILABLE` / `UNAVAILABLE` | staged VERSION → CREDENTIALS → CANARY; any stage failure ⇒ `UNAVAILABLE` (`LOCAL_AGENT_UNAVAILABLE`) |
| `execute(request, context)` | `COMPLETED` / `FAILED` / `CANCELLED` / `UNKNOWN` | exit 0 ⇒ `COMPLETED` (`AGENT_TURN_COMPLETED`); nonzero ⇒ `FAILED` (`ATTEMPT_PROCESS_FAILED`, strict — an exit-127 healthy turn was observed in #470 and stays FAILED); timeout ⇒ `UNKNOWN` (`ATTEMPT_TIMEOUT_RECONCILED`); spawn error ⇒ `UNKNOWN`; AbortSignal cancel ⇒ `CANCELLED` (`ATTEMPT_CANCELLED`); probe unavailable / invalid inputs / missing worktree ⇒ `FAILED`, never spawns |
| `cancel(attempt)` | `CANCEL_REQUESTED` / `NOT_RUNNING` / `UNKNOWN` | aborts the live attempt (same path as a Host AbortSignal); idle ⇒ `NOT_RUNNING` |
| `reconcile(attempt)` | `RUNNING` / `COMPLETED` / `FAILED` / `CANCELLED` / `UNKNOWN` | live slot ⇒ `RUNNING`; git candidate ref (`refs/tally-zcode-candidates/<attemptId>`, read-only) ⇒ `COMPLETED`/`FAILED CANDIDATE_STALE`; occupied worktree without ref or absent record ⇒ `UNKNOWN` — never a blind relaunch |

The CLI facts were verified in #470 and are kept unchanged: binary
`~/.zcode/runtime/releases/<ver>/bin/zcode.mjs` via node; version from the
`version` subcommand (`--version` prints the bundle stub); non-interactive
turn `-p <prompt> --cwd <dir> --mode yolo --json --no-color`; final JSON
parsed from the first `{` (noise lines may precede); no built-in timeout ⇒
the adapter enforces one and kills the process tree
(`taskkill /PID <pid> /T /F`); auth failure stalls silently ⇒ the probe
canary (`-p "Reply with exactly: OK"`, default budget 180 s, configurable)
is the only definitive health check.

## Architecture split under AgentProvider/1

The provider owns ONLY its CLI/protocol execution and reporting. Unlike
#470, this adapter **never creates the worktree** (the Host supplies the
already admitted `request.worktreePath`; a missing directory ⇒ `FAILED
WORKTREE_ABSENT`, never `git worktree add`) and **never creates the
candidate ref** — commit/finalize/readback and Outcome remain solely with
the existing selected B/K/Host/C chain. `reconcile` only READS the durable
candidate ref. The `live` map is a control-plane handle (pid + abort) for
in-flight attempts, deleted at the terminal observation — not an execution
journal; durability stays git-anchored.

Evidence is an untrusted, adapter-redacted blob (`agent` session/response
bounded to 2000 chars, `stdoutTail`/`stderrTail` bounded tails,
`process`, `providerProbe`); the builder env allowlist strips all `ZCODE_*`
desktop pointers and carries no GitHub/CI credentials
(`GIT_CONFIG_GLOBAL` empty, `GCM_INTERACTIVE=never`). Observations can
never carry authority fields — the contract normalization rejects them.

## Injectable seams

`createZcodeAgentProvider({spawnFn, operations, now, killTree, config})` —
same injection shape as #470: `operations` (probeProvider / attemptRecord /
runAgentProcess) and `spawnFn` drive the whole test matrix without a real
CLI; `config` carries local facts (`cliPath`, `nodePath`, `gitPath`,
`credentialsPath`, `repositoryPath`, `canaryTimeoutMs`, ...).

## Honest boundaries (inherited from #470)

- No OS-level confinement: the CLI runs as the operator user; the write-set
  fence is enforced Host-side (post-hoc, inside the disposable worktree).
- Auth failure = silent stall; the bounded canary is the only honest
  health signal.
- Nonzero exit never reads COMPLETED even when the printed answer looks
  healthy (stdout and the git record stay as reconciliation evidence).

## Testing

```
cd packages/tally-provider-zcode && npm test
```

`node --test` matrices: `test/zcode-agent-provider.test.mjs` (ported
#470 process matrix with fake spawn) and `test/contract.test.mjs`
(contract compliance against `packages/tally-agent-provider`).
