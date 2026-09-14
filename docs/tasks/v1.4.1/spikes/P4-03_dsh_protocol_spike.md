# P4-03 — DeepSeek Harness (DSH) integration protocol spike

- **Date:** 2026-09-12
- **Probed artifact:** npm `@deepseek-ai/dsh` **0.1.5-rc.1** (dist-tag `latest`; `alpha` = 0.1.5-alpha.2, `next` = 0.1.5-rc.2). Docs site tracks source `0.1.5-alpha.1`. Upstream: github.com/deepseek-ai/deepseek-harness, master, MIT, **developer preview — "THERE WILL BE COMPATIBILITY-BREAKING CHANGES"**. No commit hash observable from the npm artifact; version string is the only pin available to an out-of-process consumer.
- **Environment:** Windows 10 / Git Bash, Node v26.8.1, npm 11.19.0. `DEEPSEEK_API_KEY` present in env (referenced by name only; never recorded).
- **Reproduction:** `tools/dsh-spike/probe-headless.sh` (env-read only; no key at build time).

## 1. Evidence classes

- **[DOC]** official/community docs (URL cited); **[OBS]** observed live probe output (transcripts below or in this file); **[ASM]** assumption / open risk — not a fact.

## 2. Protocol facts (per spike question)

### 2.1 CLI / SDK / process protocol — [OBS] + [DOC]

Three out-of-process surfaces shipped in the single `dsh` launcher:

| Surface | argv | Nature | Evidence |
|---|---|---|---|
| headless | `dsh --profile headless "<task>"` | one-shot process: submit one task as "an ordinary user message", "last non-empty assistant text is written to stdout", exit | [OBS] probes §3; [DOC] https://deepseekdocs.com/en/docs/user-guide/cli (tracks 0.1.5-alpha.1) |
| sdk | `dsh --profile sdk` | "Serve DeepSeek Harness SDK clients over stdio JSON-RPC" until client disconnect | [OBS] `--profile sdk --help`; upstream `packages/sdk` = "Out-of-process SDK: JSON-RPC protocol and TypeScript client/server" |
| acp | `dsh --profile acp` | "Serve automation clients over Agent Client Protocol stdio" until disconnect | [OBS] `--profile acp --help`; upstream `packages/acp` |

Launcher grammar [OBS `dsh --help`]: `-V/--version`, `--profile <name>`, `--from-default-profile`, `--patch <path>` (repeatable YAML overlay, highest precedence), `--dump-config` / `--dump-default-config`; first unrecognized token starts app args; app flags vary per profile. `dsh --profile headless --help` [OBS] shows **only `-h/--help` + the task argument** — no model flag, no cwd flag, no output flag.

Historical breakage data point [DOC]: the `dsh run` subcommand **has been removed**; one-off tasks moved to `--profile headless <task>`. Confirms developer-preview volatility.

The in-harness subagent seam (`ctx.subagents` providers: spawn/fork/acp/dsh-sdk/Codex/Claude-Code) is an **in-process TypeScript API** inside the harness ([DOC] upstream `docs/subsystems/subagent.md`, `packages/subagent/tool-subagent/src/index.ts`) — **not** reachable from Go except through the sdk/acp stdio surfaces or a subprocess bridge. This matches the AGENTS.md Runtime lock: `Tally Go → runtimes/dsh → external DeepSeek Harness process`, and if SDK semantics are needed, `→ subprocess bridge → DSH SDK` (TS).

### 2.2 cwd semantics — [OBS] decisive

Default config dump [OBS, `--profile headless --dump-default-config`]:

```yaml
- id: permission-core        # (workspace preset block)
    mode: !!js process.env.DSH_PERMISSION_MODE ?? 'workspace-write'
    workspaceRoot: !!js process.cwd()
- id: system-prompt
  config:
    personaSuffix: Your working directory is {{cwd}}.
```

- workspace root = **process working directory** (also stated [DOC]: "The invoking directory is always the default workspace root").
- [OBS] Credentialed probe in an empty temp dir containing only `hello.txt`: task "read hello.txt and reply its contents" → stdout `probe file`, exit 0. File access confirmed bound to cwd.
- No CLI flag to override workspace; Tally must spawn with cwd = the disposable candidate worktree (same pattern as CodexRuntime's `-C` pinning, achieved via process cwd instead of a flag).
- Permission default `workspace-write` (matches CodexRuntime's conservative `defaultSandbox`); escalation to `danger-full-access` only via env [OBS dump: approval preset `never` only under `DSH_PERMISSION_MODE=danger-full-access`].

### 2.3 Output / result shape — [OBS]

- stdout: **final assistant message only, plain text** — "not JSON events" [DOC], confirmed [OBS]: `probe file\n`, nothing else on stdout.
- stderr: reasoning stream under a `dsh: reasoning:` heading [OBS seen in cancelled run].
- Exit code: turn `completed` → 0, otherwise 1 [DOC]; [OBS] success = 0; keyless failure = 1; SIGTERM kill = 143 (128+15, default termination — see §2.4).
- **No per-run JSONL event vocabulary** on the headless surface (unlike `codex exec --json`): `event.streaming`, structured output schema, usage reporting are **not available** headless. Persisted session log exists but is `session.v3.jsonl.zstd` under `$DSH_HOME/sessions/<workspace-slug>/` [OBS] — an internal format, not a contract; do not parse.
- Result truth: headless output is an **Agent Claim** (INV-AUTH-01) — same treatment as CodexRuntime results.

### 2.4 Cancel / kill semantics — [OBS] partial, one open risk

- [OBS] SIGTERM mid-run (bash `sleep 90` task): process died with exit 143, stdout empty, no final message — cancellation = process termination, no graceful "aborted" final answer observed on the headless surface. Consistent with [DOC] upstream note that *inside* the harness, cancellation is an `AbortSignal` that settles as `aborted`; externally we only see process death.
- **Open risk [ASM]:** whole-tree teardown when Tally kills the `dsh` launcher (Node) on Windows is **unverified** — the probe host runs other Node processes so orphan attribution was inconclusive (`tasklist node.exe` count includes unrelated processes). Upstream internally waits on whole-tree exit via `dsh-subprocess` when *it* is the parent ([DOC] design note 2026-08-04), but that guarantee does not transfer to Tally-as-parent. P4-04 must use Windows Job Objects / process-group kill and prove teardown.

### 2.5 Provider / model config — [OBS]

- No CLI model flag. Default route from config dump [OBS]:

```yaml
- id: agent-default-model
  name: '@deepseek-ai/dsh-agent-default-model'
  config:
    provider: deepseek-official
    model: deepseek-flash
```

- Model/provider/reasoning overrides are **profile-layer config** (`--patch <yaml>` overlay or a custom profile via `--from-default-profile`), not argv. Tally can therefore pin desired config as a generated minimal YAML patch (desired-config pipeline, mirroring CodexRuntime's seeded `config.toml`), and must record observed model separately (§13 provenance rule: no `requested == actual` without observation — headless does not report the actual model on stdout, so actual-model is **not observable** headless; must stay unclaimed or come from the sdk/acp seam).
- Reasoning-effort / max-tokens are subagent `agentOptions` inside the harness [DOC]; headless surface exposure unverified.

### 2.6 Auth shape — [OBS]

- Keyless probe: exit 1, empty stdout, stderr:

```
dsh: MISSING_CREDENTIAL: llm-deepseek: no API key for provider route "deepseek-official"; store DEEPSEEK_API_KEY through the credentials service (the web Models page writes it), or export DEEPSEEK_API_KEY in the launching environment
```

  → fail-safe, structured, names the exact env var. Good negative path.
- Credentialed probe with only `DEEPSEEK_API_KEY` exported: success. Auth = `DEEPSEEK_API_KEY` env (or credentials file under `$DSH_HOME` via the web UI / `dsh-credentials-local`). Web-search plugin separately reads `apiKeyEnv: DEEPSEEK_API_KEY` [OBS dump].
- Default `DSH_HOME` = `~/.dsh` [DOC]; profiles auto-initialize there [OBS: `~/.dsh/profiles/headless`, `node_modules`, `sessions/`, `storages/` created by probes].

## 3. Probe transcript (minimal, reproducible)

Commands + observed results (Windows/Git Bash; `DSH` = `npx -y @deepseek-ai/dsh@0.1.5-rc.1`):

| # | Command | Observed |
|---|---|---|
| 1 | `$DSH --version` | `0.1.5-rc.1` |
| 2 | `$DSH --help` | launcher flags as §2.1; only `web`/`plugin` subcommands |
| 3 | `$DSH --profile headless --help` | only `-h` + `[task...]` |
| 4 | `$DSH --profile headless --dump-default-config` | 348-line plugin tree; key extracts quoted in §2.2/§2.5/§2.6; telemetry block defaults `FEEDBACK_ONLY` with OTLP URL `https://harness-telemetry.deepseeksvc.com/v1/logs` |
| 5 | keyless (env unset) `--profile headless "Reply PONG"` in temp ws | exit 1, stdout empty, `MISSING_CREDENTIAL` stderr (§2.6) |
| 6 | credentialed `--profile headless "read hello.txt..."` in temp ws | exit 0, stdout `probe file` |
| 7 | SIGTERM kill mid long task | exit 143, stdout empty (§2.4) |
| 8 | `$DSH --profile sdk --help` / `--profile acp --help` | stdio servers, no app flags (§2.1) |
| 9 | `$DSH --profile sdk < /dev/null` | immediate clean exit 0, no banner (EOF = disconnect) |
| 10 | `npm view @deepseek-ai/dsh-sdk` | **404** — the TS client SDK is not published on public npm as of today; sdk-profile protocol docs are not externally published |

Rate budget: 2 model-backed turns total (one keyless-fail, one success) + one killed turn; all other probes are local.

Process cleanup: probes ran under `timeout`; launcher exits observed for every run; orphan attribution inconclusive as noted (§2.4 risk).

## 4. Mapping to `agentruntime.Runtime/Session` needs (vs runtimes/codex)

| Protocol need | CodexRuntime today | DSH headless (0.1.5-rc.1) | Fit |
|---|---|---|---|
| one Execute = one process | `codex exec` | `--profile headless "<task>"` | FIT |
| workspace pinning | `-C <ws>` flag | spawn with cwd = worktree (no flag) | FIT (via cwd) |
| stdout final result | yes (item.completed JSONL) | yes (plain text only) | FIT |
| event streaming | `--json` JSONL | **NONE** | GAP (claim-only; no live events) |
| session resume/fork | `exec resume/fork` | **NONE observed** headless; sessions persist internally but no resume argv | GAP |
| model select | `-m` | config overlay only; actual model unreported | PARTIAL (desired-only) |
| cancellation | kill process | kill process (exit 143) | FIT (teardown unproven on Win) |
| auth | auth.json handover, per-run CODEX_HOME | `DEEPSEEK_API_KEY` env | FIT (env injection; simpler) |
| isolation | per-run CODEX_HOME | default shared `~/.dsh` (sessions, storages, profiles persist across runs) | GAP → needs per-run `DSH_HOME` |
| structured output | `--output-schema` | none | GAP |

**Recommended seam for P4-04:** `Tally Go → runtimes/dsh → external `dsh --profile headless` process`, with (a) per-run `DSH_HOME` for isolation, (b) `DSH_TELEMETRY_MODE=DISABLED` (default FEEDBACK_ONLY ships telemetry to a DeepSeek OTLP endpoint — unacceptable for candidate execution), (c) generated minimal `--patch` YAML pinning provider/model, (d) job-object/process-tree kill, (e) capability set declaring only what headless proves (no event.streaming, no session.resume, no output.structured, no model.select-as-observed). If events/steering become requirements, the `sdk`/`acp` stdio profiles are the growth path, but their wire protocols are currently undocumented externally and the TS SDK is unpublished (npm 404) — a subprocess bridge would be required per the Runtime lock and is out of scope for P4-04.

## 5. Upstream version / breaking-change risk

- Developer preview, 0.1.x versioning, breaking changes promised in README [DOC].
- Already-observed break: `dsh run` removed in favor of `--profile headless` [DOC].
- Docs site version (0.1.5-alpha.1) ≠ npm latest (0.1.5-rc.1) ≠ dist-tags — pin an exact version and probe identity at runtime (`--version`), never trust ranges.
- SAFETY.md [DOC]: no security audit; sandbox/approval "do not guarantee isolation"; not sole security control → aligns with Tally's disposable-worktree + fail-safe stance; adapter must keep DSH inside the candidate sandbox boundary.

## 6. Verdict

**GO-WITH-CONDITIONS** — see report `docs/tasks/v1.4.1/reports/P4-03_deepseek_harness_integration_spike.md`.
