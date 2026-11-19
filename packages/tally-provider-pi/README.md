# @kaicreator/tally-provider-pi (v2.1 candidate, default-off)

Node AgentProvider/1 adapter for the **pi coding agent CLI, pinned 0.85.1**.
Source-only candidate for kaicreator-mm/tally#473, ported from the PR #469 Go
candidate (`runtimes/pi`) whose argv/env/NDJSON/exit-code contract is the sole
evidence source (host observations of the real pi 0.85.1 binary, 2026-11-19).
The Go source is evidence input only — nothing is bridged, imported or
executed from it.

**Default-off**: not mounted into the selected v2 Host. It participates only
through an explicit `enabledProviderIds` selection and actual Host
qualification; there is no fallback path. A provider `COMPLETED` result is an
untrusted observation — never a Task PASS, Review, Release or Candidate fact.

## Ownership

Provider owns the pi process lifecycle: pinned-version probe, credential
gate, hardened argv/env construction, NDJSON parsing, exit-code
classification, tree-terminating cancel. Host owns Claim/FSW admission, K
permission, worktree grants, independent git candidate readback, Review, CI
and Task Outcome. The in-flight map inside the adapter holds process handles
only (pid + cancel flag) — it is not an execution journal and no outcome is
persisted. `reconcile` is always `UNKNOWN`/`NOT_RECONCILED` (no second
persistence layer).

## CLI contract mapping (evidence: PR #469, pi 0.85.1)

| Concern | Value (all observed, none invented) |
| --- | --- |
| Invocation | one-shot process: `pi -p --mode json --no-session --session-dir <per-attempt tmp> --no-extensions --no-skills --no-prompt-templates --no-themes --no-context-files --no-approve [--provider p] [--model m] [--thinking t] -- "<instruction>"` |
| Trailing separator | instruction is always the single positional argument after `--` |
| Workspace | spawn `cwd` = Host-provided worktree (pi has no cwd flag; tools write inside the process cwd) |
| stdout (`--mode json`) | NDJSON: first `{type:"session",id:<uuid>,cwd}` line, then `message_end` events; the assistant line carries content text, provider, model, `usage{input,output,cacheRead,...}`, stopReason |
| stdout (`--mode text`) | final answer plain text; usage stays unobserved (zero) |
| Exit codes | 0 = success; non-zero = incomplete (keyless: stderr `No API key found for <provider>.`; bad model: provider 400 text); both mapped to `FAILED` with redacted stderr tail |
| Contract violation | exit 0 but no text-bearing assistant `message_end` ⇒ `FAILED` `PI_CONTRACT_NO_ANSWER` (fail-safe, success never guessed) |
| Cancelled stream | truncated tail lines are skipped by the parser; cancellation classifies `CANCELLED`, never success |
| Version gate | `pi --version` → bare version line; mismatch or unobservable ⇒ fail-closed (`UNAVAILABLE` on probe, `FAILED` on execute, nothing spawns) |
| Credentials | env-only (observed deepseek readiness flips with `DEEPSEEK_API_KEY`); `--api-key` exists but is never used — credentials never enter argv or evidence |

## Environment hardening (`src/pi-env.mjs`, pure functions)

- All **42** provider credential env vars (ANTHROPIC_*/OPENAI_*/DEEPSEEK_*/
  XAI_*/AWS_*/… exactly as observed in #469 `isolation.go`) are stripped from
  the inherited environment case-insensitively.
- The **6** pi config switches (`PI_CODING_AGENT_DIR`,
  `PI_CODING_AGENT_SESSION_DIR`, `PI_PACKAGE_DIR`, `PI_OFFLINE`,
  `PI_TELEMETRY`, `PI_SHARE_VIEWER_URL`) are stripped and re-imposed by the
  isolation semantics.
- Six forced keys override everything (case-insensitive; no case-variant
  duplicates): `HOME`/`USERPROFILE` → per-attempt `home/`,
  `PI_CODING_AGENT_DIR` → `agent-config/`, `PI_CODING_AGENT_SESSION_DIR` →
  `sessions/`, `PI_OFFLINE=1`, `PI_TELEMETRY=0`.
- The only explicit injection channel is the Host `context.env` (e.g. the
  provider API key). Forced keys reject same-named injections. A fully empty
  env is deliberately not used (unobserved OS survival surface).
- Per-attempt isolation group (`tally-pi-attempt-*` temp root + agent-config
  + sessions + home) is created before spawn and removed when the attempt
  settles; the owner's `~/.pi` is never touched (`--no-session` plus the
  redirected session dir is defense in depth).

## Failure / cancellation classification

Priority: cancellation (`CANCELLED` `PI_EXEC_CANCELLED`, or
`PI_CANCELLED_BEFORE_START` when the signal is already aborted) > timeout
(tree killed, `UNKNOWN` `PI_EXEC_TIMEOUT`) > spawn failure
(`PI_SPAWN_FAILED`) > orphaned output pipes (`PI_OUTPUT_PIPE_UNRESOLVED`,
WaitDelay-style bound) > non-zero exit (`PI_EXIT_NONZERO`) > exit 0 without
answer (`PI_CONTRACT_NO_ANSWER`) > `COMPLETED` with stream-observed
evidence. Every `UNKNOWN` carries a reason code. Failure stderr is redacted
with the logpipe `RedactFine` pattern set before it enters evidence.
Tree termination: `taskkill /PID <pid> /T /F` on Windows, SIGKILL of the
child's own POSIX process group elsewhere.

## Usage

```js
import {createPiAgentProvider} from '@kaicreator/tally-provider-pi';
import {createAgentProviderRegistry, createProviderAttemptPin} from '@kaicreator/tally-agent-provider';

const pi = createPiAgentProvider({
  binPath: 'C:\\Application\\pi\\pi', // explicit; pinned 0.85.1 gate
  provider: 'deepseek',               // expected flags, never observations
});
const registry = createAgentProviderRegistry([pi], {defaultProviderId: 'codex'});
const pin = createProviderAttemptPin({
  taskId: '473', attemptId: 'attempt:473:r1',
  sourceSha: '<40-hex>', providerId: 'pi',
});
const provider = registry.resolve({pin, enabledProviderIds: ['pi']});
const probe = await provider.probe({env: {DEEPSEEK_API_KEY: key}}); // AVAILABLE
const done = await provider.execute(
  {taskId: '473', attemptId: 'attempt:473:r1', sourceSha: pin.sourceSha,
   prompt: 'add a test', worktreePath: worktree, timeoutMs: 600000},
  {env: {DEEPSEEK_API_KEY: key}, signal: controller.signal},
); // {state, reason?, evidence?} — an observation, not a Task fact
```

The `@kaicreator/tally-agent-provider` contract is imported monorepo-relative
(`../../tally-agent-provider/src/index.mjs`) — this is a source-only
candidate with no workspace install yet; the package name import above works
once the candidate is mounted.

## Tests

`npm test` runs `node --test test/*.test.mjs` against `test/fake-pi.mjs`, a
Node stand-in CLI that imitates only the observed 0.85.1 surface (version
line, NDJSON stream, exit codes, keyless stderr, hang scenario, cwd-bound
read instruction). All process boundaries (argv shape, env strip/force,
stream parsing, exit-code classification, timeout/abort tree termination,
isolation cleanup) are exercised against real child processes. No real pi
turn is ever run by this package.

## NOT_RUN (honest list)

- **Real pi turn / canary**: never executed here (`canaryTurn: 'NOT_RUN'` in
  probe evidence). Live acceptance against the pinned CLI is the
  controller's gate.
- **session.isolation evidence gate**: the Go candidate declares
  session.isolation only behind a conformance canary; this adapter declares
  an empty capability set and runs no isolation canary.
- **Live CLI smoke / Host qualification / CI**: not run; default-off until
  the Host wires and qualifies the adapter.
- **Windows live tree-termination receipt against real pi**: covered by the
  fake CLI's hang scenario only.

## Boundaries

Observe-only, no authority fields, no fallback, no scheduler, UNKNOWN always
reasoned. Instructions beginning with `@` are pi's file-attachment syntax
(observed in its help text, not further verified) — a known, documented
boundary, not worked around here. The adapter contract is valid only for the
pinned 0.85.1 CLI; version drift fails closed.

Refs: #473 #474 #469
