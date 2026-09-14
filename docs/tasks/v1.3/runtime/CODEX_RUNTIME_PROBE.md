# CODEX_RUNTIME_PROBE — P1-36 (spike)

Task: `docs/tasks/v1.3/P1/P1-36.md` (depends on P1-04). Kind: read-only probe; no production code changed.
Probe date: 2026-09-06 (local). Environment: Windows 11 Pro x64 (10.0.26200), Git Bash, no `codex` on PATH.
Authority: PRD §10 Runtime Protocol, §35 Conformance; DECISION_LOCKS #6 (structured CommandSpec), #13 ("CodexRuntime must use a probed real local surface; never invent CLI flags/protocol").

Everything below was executed on this machine; exit codes are quoted verbatim. No CLI/SDK flag in this
document is invented; each is copied from `--help` output or a generated protocol schema of the installed
binary.

## 1. Surface discovery

| Fact | Evidence |
| --- | --- |
| `codex` is NOT on PATH | `which codex` → not found (exit 127); `where.exe codex` → "用提供的模式无法找到文件。" |
| Codex desktop App is installed | `C:\Users\15549\AppData\Local\OpenAI\Codex\` exists (contents: `bin`, `runtimes`, `chrome-native-hosts-v2.json`) |
| App-bundled CLI binary | `C:\Users\15549\AppData\Local\OpenAI\Codex\bin\9ba750cce02d5e5c\codex.exe` (295,295,792 bytes); sibling helpers `codex-code-mode-host.exe`, `codex-command-runner.exe`, `codex-windows-sandbox-setup.exe`; second bin dir `e5208f10301fe5e6` contains only `rg.exe` |
| Binary path is content-hash-scoped | The `9ba750cce02d5e5c` directory name is a hash; the App rewrites it on update. The App itself publishes the current path as `CODEX_CLI_PATH` in `~/.codex/config.toml` (`[mcp_servers.node_repl.env]`) — an adapter must discover the path, not hardcode it |
| App build version | `26.901.22334` (from `BROWSER_USE_CODEX_APP_VERSION` in the same config block) |
| CLI version | `codex.exe --version` → `codex-cli 0.153.0`, exit 0 |
| CODEX_HOME | `C:\Users\15549\.codex` (App-managed `config.toml`, `sessions/`, sqlite state) |
| No standalone SDK | No `@openai/codex` / `codex-sdk` npm package installed (`npm ls -g`, `require.resolve` fail); no Go SDK exists; the process CLI + app-server protocol are the only local surfaces |

## 2. CLI surface (`codex.exe --help`, exit 0)

Subcommands relevant to a runtime adapter (verbatim from `--help`):

```text
  exec              Run Codex non-interactively [aliases: e]
  review            Run a code review non-interactively
  resume            Resume a previous interactive session (…)
  fork              Fork a previous interactive session (…)
  mcp-server        Start Codex as an MCP server (stdio)
  app-server        [experimental] Run the app server or related tooling
  exec-server       [EXPERIMENTAL] Run the standalone exec-server service
  apply             Apply the latest diff produced by Codex agent as a `git apply` …
  sandbox           Run commands within a Codex-provided sandbox
  doctor            Diagnose local Codex installation, config, auth, and runtime health
  features          Inspect feature flags
```

### 2.1 `codex exec` (non-interactive; `--help` exit 0)

Flags load-bearing for a Tally adapter (all verbatim):

```text
  -C, --cd <DIR>                  working root for the agent
  --add-dir <DIR>                 additional writable directories
  -s, --sandbox <read-only|workspace-write|danger-full-access>
  -a, --ask-for-approval <on-request|never>
  -m, --model <MODEL>
  --json                          Print events to stdout as JSONL
  --output-schema <FILE>          JSON Schema file constraining the model's final response
  -o, --output-last-message <FILE>
  --ephemeral                     Run without persisting session files to disk
  --skip-git-repo-check
  --ignore-user-config            Do not load $CODEX_HOME/config.toml
  --thread-source <SOURCE>        Source classification for new/forked threads
```

Sub-surfaces: `codex exec resume [<SESSION_ID>|--last] [PROMPT]` and `codex exec fork <SESSION_ID> [PROMPT]`
(both `--help` exit 0) — `session.resume` / `session.fork` exist on the one-shot exec surface too.

## 3. Live `codex exec --json` probe (auth-gated, in temp dir outside the repo)

Command: `codex.exe exec --json --skip-git-repo-check --sandbox read-only --ephemeral "Reply with the single word OK"`

Observed stdout JSONL (verbatim, errors interleaved on the same stream):

```json
{"type":"thread.started","thread_id":"01a07368-d440-7912-b025-44811ad11414"}
{"type":"turn.started"}
{"type":"error","message":"Reconnecting... 1/5 (unexpected status 401 Unauthorized: Missing bearer or basic authentication in header, url: wss://api.openai.com/v1/responses ...)"}
{"type":"item.completed","item":{"id":"item_0","type":"error","message":"Falling back from WebSockets to HTTPS transport. unexpected status 401 Unauthorized: ..."}}
```

Result: the exec surface is callable and emits structured JSONL events starting with `thread.started`
(UUID thread id) and `turn.started`; model execution is gated by credentials (401 after reconnect retries,
process exit 0 with an error item). Note: with no prompt argument and stdin piped, exec announces
`Reading additional input from stdin...` — the adapter must always pass the prompt or close stdin.

## 4. App-server protocol surface (structured, but `[experimental]`)

`codex app-server --help` (exit 0): transports `stdio://` (default), `unix://`, `ws://IP:PORT` with
`capability-token` / `signed-bearer-token` auth modes; subcommands `daemon`, `proxy`, `generate-ts`,
`generate-json-schema`.

`codex app-server generate-json-schema --out <DIR> --experimental` (exit 0) generated a schema bundle:
39 top-level files plus `v1/` (2 files) and `v2/` (265 files), combined schema
`codex_app_server_protocol.v2.schemas.json` with 622 definitions. Extracted facts (from the generated
schema, not guessed):

- **99 `ClientRequest` methods.** Minimal-Core mapping (PRD §10.2):
  - Descriptor/Start → `initialize`, `thread/start`, `thread/list`
  - Execute → `turn/start` (params: `threadId`, `input`, `cwd`, `sandboxPolicy`, `approvalPolicy`,
    `model`, `outputSchema`, `effort`, `serviceTier`; required: `input`, `threadId`)
  - Cancel → `turn/interrupt`; steer → `turn/steer`
  - Result → `turn/completed` notification + `thread/turns/list`, `thread/items/list`
  - resume/fork → `thread/resume`, `thread/fork`; persist → `thread/archive`/`thread/unarchive`
  - also `review/start` (code review as a method), `command/exec|write|terminate|resize`,
    `fs/readFile|writeFile|watch`, `model/list`, `modelProvider/capabilities/read`,
    `account/rateLimits/read`, `account/usage/read` (usage evidence)
- **81 `ServerNotification` methods**, incl. `thread/started`, `thread/status/changed`,
  `thread/tokenUsage/updated`, `turn/started`, `turn/completed`, `turn/diff/updated`, `item/started`,
  `item/completed`, `item/agentMessage/delta`, `item/commandExecution/outputDelta`,
  `item/fileChange/patchUpdated`, `error`, `warning`.
- **10 `ServerRequest` (server→client) methods** — the approval channel:
  `item/commandExecution/requestApproval`, `item/fileChange/requestApproval`,
  `item/permissions/requestApproval`, `item/tool/requestUserInput`, `item/tool/call`,
  `mcpServer/elicitation/request`, `account/chatgptAuthTokens/refresh`, `attestation/generate`, and
  legacy `execCommandApproval`, `applyPatchApproval`.

Live handshake probe: spawned `codex.exe app-server` (stdio), sent
`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"clientInfo":{"name":"tally-p1-36-probe","version":"0.0.1"}}}`
→ response received (exit path clean):

```json
{"userAgent":"tally-p1-36-probe/0.153.0 (Windows 10.0.26200; x86_64) …",
 "codexHome":"C:\\Users\\15549\\.codex","platformFamily":"windows","platformOs":"windows"}
```

The generated schema bundle was kept outside the repo (`%USERPROFILE%\.codex\.tmp\p1-36-schema`) because
this task's only Allowed path is this document; integrators regenerate it per installed version with the
command above instead of freezing it.

## 5. Auth state (deployment prerequisite)

- `codex login status` → `Not logged in` (exit 1).
- `codex doctor` (exit 0) → `✗ auth  no Codex credentials were found - Run codex login or provide an API
  key through a supported auth env var.` Also: ripgrep 14.1.1 detected; git 2.55.0 detected; runtime
  version 0.153.0, executable path as above.
- No `auth.json` in `CODEX_HOME`; the desktop App authenticates through its own channel, so
  CLI-driven headless runs need explicit credential provisioning (`codex login` or supported auth env var)
  before any real turn can succeed. 401 evidence in §3 confirms this is a hard gate, not a warning.

## 6. Session persistence / isolation-relevant facts (PRD §10.4, §35)

- Session rollouts persist under `~/.codex/sessions/YYYY/MM/DD/rollout-<ts>-<uuid>.jsonl`
  (files observed for 2026/09/04 and 2026/09/05). `--ephemeral` skips persistence.
- Resume/fork select sessions by UUID or name (`--last` supported), i.e. cross-session retrieval through
  Runtime-observable paths exists by design (`codex resume --last`, app-server `thread/list`,
  `thread/loaded/list`). A future CodexRuntime must not claim `session.isolation` unless the Session
  Isolation Canary passes — with a shared `CODEX_HOME`, cross-run retrieval of rollout files/history is
  protocol-observable, so an isolated per-run `CODEX_HOME` (or equivalent) is the likely isolation
  mechanism to conformance-test.
- Native sandbox tooling is present on this Windows install (`codex sandbox` subcommand,
  `codex-windows-sandbox-setup.exe`, app-server `windowsSandbox/setupStart|readiness`).

## 7. Surface evaluation vs PRD §10.2 / §10.3 (evidence-based)

| PRD §10.3 capability | Exec surface (`codex exec`) | App-server (v2 protocol) |
| --- | --- | --- |
| session.resume | yes — `exec resume <id|--last>` | yes — `thread/resume` |
| session.fork | yes — `exec fork <id>` | yes — `thread/fork` |
| session.isolation | unproven — Canary required; shared CODEX_HOME risk (§6) | same |
| turn.steer | no | yes — `turn/steer` |
| turn.interrupt | no (cancel = kill process) | yes — `turn/interrupt` |
| event.streaming | yes — `--json` JSONL on stdout | yes — 81 notification methods |
| output.structured | yes — `--output-schema`, `-o` | yes — `outputSchema` on `turn/start` |
| sandbox.native | yes (declared modes; Windows sandbox tooling present) — conformance needed | yes — `sandboxPolicy` param |
| tool.approval | no client-observable approval channel headless (policy is `on-request|never`) | yes — ServerRequest approval methods |
| tools.custom | no (no flag on exec) | partial — `item/tool/call`, dynamic tool schemas present (needs probe) |
| model.select | yes — `-m` | yes — `model`/`model/list` |
| session.persist | yes — rollout files; `--ephemeral` disables | yes — thread archive/list/read |
| memory.declared | no evidence → must not declare | no evidence → must not declare |

## 8. Conclusion: narrowest structured integration surface (AC2)

**Recommendation: implement CodexRuntime on the `codex exec` surface — one process per Execute, argv-only,
`--json` JSONL events on stdout, `--output-schema` for the final message, `-C` pinned to the disposable
worktree, `-s`/`-a` mapped from ExecutionPolicy, cancel via process termination.**

Reasons:

1. Non-experimental, documented CLI surface; every needed flag observed in `--help` (§2.1) — satisfies
   DECISION_LOCKS #6/#13 without inventing anything.
2. Covers PRD §10.2 Minimal Core with Tally-side simplicity: Start = spawn, Execute = one exec run,
   Events = stdout JSONL parse, Result = last message + terminal event, Cancel = process kill (mirrors
   `runner.Command`/structured argv precedent in this repo). Conformance items (§35: start, cwd,
   execute, events, cancel, result, crash handling, worktree isolation) are all directly testable.
3. No SDK dependency and no long-lived daemon; per-run process isolation is a better default fit for
   untrusted candidate execution than a shared app-server session.

**Recorded alternative (not chosen for v1):** `codex app-server` JSON-RPC over stdio is the richer surface
(turn steer/interrupt, approval routing back to Tally, usage/rate-limit reads, `review/start`). It is
functional here (§4) but self-labeled `[experimental]`; its 99-method schema can be regenerated per
installed version via `generate-json-schema`. If a later task needs `tool.approval` routing or mid-turn
steering, that task should re-probe and own the upgrade — not this one.

**Adapter prerequisites recorded for the implementing task:**

- Binary discovery: resolve `codex.exe` via config/env (the App publishes `CODEX_CLI_PATH`), never a
  hardcoded hash path (§1). `codex` is not on PATH on this machine.
- Auth provisioning is an operator/deployment prerequisite (`codex login` or supported auth env var);
  Tally must not invent a credential channel. Without it, exec fails 401 after `thread.started`/`turn.started`
  (§3, §5) — conformance tests must treat this as an environment precondition, and adapter tests should
  use a FakeRuntime for model-independent paths (DECISION_LOCKS #13).
- Structured argv only (no shell strings), `context.Context`-driven process cancellation, JSONL parse
  errors are explicit errors, usage numbers from `thread/tokenUsage/updated`-equivalent exec events are
  claims (INV-AUTH-01) — not facts.
- Capability declaration for CodexRuntime v1 (truthful, per §35): declare `session.resume`,
  `session.fork`, `event.streaming`, `output.structured`, `sandbox.native`, `model.select`,
  `session.persist`; do NOT declare `turn.steer`, `turn.interrupt`, `tool.approval`, `tools.custom`,
  `memory.declared`; declare `session.isolation` only after the Session Isolation Canary passes (§6).

## AC mapping

- **AC1** (exact installed command/SDK/app-server surface, version/help/schema evidence): PASS — §1–§5.
- **AC2** (narrowest structured integration surface chosen): PASS — §7–§8.
- **AC3** (BLOCKED if no callable surface): not triggered — a callable surface exists (§3, §4); recorded
  auth absence as a prerequisite rather than a blocker (§5).

## Validation transcript (exact commands executed)

| Command | Result |
| --- | --- |
| `codex --help` (as written in task file) | `codex` not found on PATH (exit 127 before `\|\| true`); help evidence captured by running the App-bundled binary directly — see next row |
| `<bundled>\codex.exe --version` | PASS — `codex-cli 0.153.0`, exit 0 |
| `<bundled>\codex.exe --help` | PASS — exit 0, §2 |
| `<bundled>\codex.exe exec --help` / `exec resume --help` / `exec fork --help` | PASS — exit 0, §2.1 |
| `<bundled>\codex.exe app-server --help` / `mcp-server --help` | PASS — exit 0, §4 |
| `<bundled>\codex.exe app-server generate-json-schema --out …` | PASS — exit 0, 39+2+265 schema files, §4 |
| JSON-RPC `initialize` against spawned `codex.exe app-server` | PASS — response received, §4 |
| `<bundled>\codex.exe exec --json …` (temp dir) | PASS (surface probe) — JSONL events then 401, §3 |
| `<bundled>\codex.exe login status` | exit 1 — `Not logged in`, §5 |
| `<bundled>\codex.exe doctor` | exit 0 — auth ✗, §5 |

No repository file other than this document was read-modified or written; probe artifacts live outside
the repo (`%USERPROFILE%\.codex\.tmp\p1-36-*`).
