# P4-03 — DeepSeek Harness integration spike: Execution Report

- **TASK:** P4-03
- **STATUS:** DONE
- **Spike verdict:** **GO-WITH-CONDITIONS**
- **Date:** 2026-09-12
- **Execution/Review model:** HIGH / HIGH (per task pack: external runtime is developer-preview; authority/release-gate adjacency)
- **Hidden material:** none read (DL-17)

## 1. Status / changed files

Status: DONE (research spike; no production code changed).

Created files (all inside this task's Allowed paths):

- `docs/tasks/v1.4.1/spikes/P4-03_dsh_protocol_spike.md` — the spike document: probed artifact/version, evidence-tagged protocol facts (CLI/SDK/process protocol, cwd, output/result, cancel/kill, provider/model config, auth), probe transcript, mapping to `agentruntime.Runtime/Session` needs vs `runtimes/codex`, version/breaking-change risk, verdict basis.
- `tools/dsh-spike/probe-headless.sh` — minimal reproducible probe (version, help, keyless fail-safe, credentialed cwd-bound run). Reads `DEEPSEEK_API_KEY` from env only; no key material in any file, log, or this report.
- this report.

## 2. Semantic changes

None to production source. Knowledge/artifact changes only: the DSH integration protocol is now pinned and evidenced; P4-04's seam choice is constrained by this verdict.

## 3. Commands executed and exact results

| # | Command | Result |
|---|---|---|
| 1 | `npm view @deepseek-ai/dsh version dist-tags` | `latest 0.1.5-rc.1`, `alpha 0.1.5-alpha.2`, `next 0.1.5-rc.2` |
| 2 | `npx -y @deepseek-ai/dsh@0.1.5-rc.1 --version` | `0.1.5-rc.1` (PASS) |
| 3 | `... --profile headless --help` | only `-h/--help` + `[task...]` — no model/cwd/output flags (PASS) |
| 4 | `... --profile headless --dump-default-config` | 348-line plugin tree; extracts: `workspaceRoot: process.cwd()`, `DSH_PERMISSION_MODE ?? 'workspace-write'`, `agent-default-model: provider deepseek-official / model deepseek-flash`, `apiKeyEnv: DEEPSEEK_API_KEY`, telemetry default `FEEDBACK_ONLY` → OTLP `harness-telemetry.deepseeksvc.com` (PASS) |
| 5 | keyless headless run (env unset) | exit 1, stdout empty, `dsh: MISSING_CREDENTIAL: ... export DEEPSEEK_API_KEY ...` (PASS — fail-safe confirmed) |
| 6 | credentialed headless run in temp ws | exit 0, stdout = exact file content read from cwd; stderr = `dsh: reasoning:` stream (PASS) |
| 7 | SIGTERM kill mid long tool task | exit 143, empty stdout — cancel = process termination; whole-tree teardown on Windows NOT conclusively verified (open risk) |
| 8 | `--profile sdk` / `--profile acp` `--help`, sdk stdin-EOF | stdio servers confirmed; EOF → clean exit 0 (PASS) |
| 9 | `npm view @deepseek-ai/dsh-sdk` | 404 — TS client SDK not on public npm (PASS as evidence) |
| 10 | `bash tools/dsh-spike/probe-headless.sh` | full repro: keyless exit 1 + MISSING_CREDENTIAL, credentialed exit 0 + cwd-bound answer, cleanup done (PASS) |
| 11 | `git status --porcelain` (self-check) | only the three created files under Allowed paths (plus other lanes' in-progress files, untouched) |

Docs consulted (key citations in spike doc): official-doc mirrors deepseekdocs.com CLI page (tracks 0.1.5-alpha.1), upstream repo README/SAFETY.md, `docs/subsystems/subagent.md`, `packages/subagent/tool-subagent/src/index.ts`, design note `2026-08-04-claude-code-and-codex-subagent-backends.md`, dsh-plugin.org run-modes tutorial. All L3 pack references read after local contract/locks/source, per §8 order.

## 4. Baseline failures observed

None. No Go test suites required (no Go code added); the probe is a shell script executed successfully (command 10).

## 5. Acceptance criteria

- **AC-1 "Report returns GO / GO_WITH_LIMITS / NO_GO and a minimal protocol transcript": PASS.**
  Verdict **GO-WITH-CONDITIONS** (task-file vocabulary GO_WITH_LIMITS is not used by this program's gate language; the meaning is identical). Transcript: spike doc §3 (10 probes, 2 model-backed turns total, rate-limited) + reproducible script.

## 6. Verdict: GO-WITH-CONDITIONS

**GO because (decisive evidence):**

1. A real out-of-process integration surface exists and was proven live against the pinned version 0.1.5-rc.1: `Tally Go → external dsh process` satisfies the Runtime lock with **no SDK bridge required** for the baseline seam. One Execute = one `--profile headless` process, stdout = final assistant message, exit 0/1 = completed/not — the exact shape CodexRuntime's `codex exec` probe record established for the codex mode.
2. cwd semantics confirmed executable: workspace = process cwd (config literal `workspaceRoot: process.cwd()` + live probe reading a cwd file), so Tally pins the disposable candidate worktree the same way it pins `-C` today.
3. Auth is a clean env injection (`DEEPSEEK_API_KEY`), fail-safe keyless (structured exit-1 error), no credential files needed.
4. Cancellation = process kill, compatible with agentruntime cancel semantics.

**Conditions (binding for P4-04 before the seam is called production):**

1. **Pin exact version** and discover identity at runtime via `--version` (never ranges; `dsh run` was already removed upstream — breaking changes are promised in the developer-preview README).
2. **Per-run `DSH_HOME` isolation** is mandatory: default `~/.dsh` persists sessions/storages across runs (protocol-observable state leakage risk analogous to the pre-P0-03 shared `CODEX_HOME`); probe evidence shows session dirs keyed by workspace slug surviving runs.
3. **`DSH_TELEMETRY_MODE=DISABLED`** must be forced in spawn env: default FEEDBACK_ONLY ships telemetry to `harness-telemetry.deepseeksvc.com` — unacceptable egress from candidate execution.
4. **Kill-tree hardening + proof**: Windows whole-tree teardown when killing the Node launcher is unverified (open risk R-1); P4-04 must use Job Objects/process-group kill with a teardown test.
5. **Honest capability set**: headless exposes **no event streaming, no session resume/fork, no structured output, no model flag** — the DSH adapter must not declare `event.streaming`/`session.resume`/`session.fork`/`output.structured`/`model.select` (actual model is not reported on stdout; requested ≠ observed must stay unclaimed, §13 provenance).
6. **Desired model/provider via generated minimal `--patch` YAML** (config overlay), never argv; SAFETY.md constraints honored (DSH is never the sole security control; stays inside the disposable worktree sandbox).

**NO-GO alternative rejected:** `sdk`/`acp` stdio profiles are the richer seam (JSON-RPC events, model observation) but their wire protocols are externally undocumented and the TS client SDK is unpublished on npm (404) — using them now would require inventing a protocol or an unversioned bridge. Registered as the P4-04+ growth path, not the baseline.

## 7. Remaining risks

- **R-1** Windows whole-tree teardown on launcher kill unverified (condition 4).
- **R-2** Developer-preview volatility: any 0.1.x bump may move/replace `--profile headless`; mitigated by version pinning + runtime identity probe + conformance canary requirement carried into P4-04.
- **R-3** Headless surface gives no per-run usage/actual-model observation → Usage stays zero-valued and model stays desired-only in the adapter (same "recorded when available → unavailable" honesty pattern CodexRuntime uses for usage).
- **R-4** Upstream commit hash not observable from the npm artifact; only version-string pinning is available to an out-of-process consumer.
- **R-5** Internal `session.v3.jsonl.zstd` format may change; must never be parsed as a contract.

## 8. Decision Locks

None touched or bypassed. Runtime lock respected (external process mode; no SDK bridge needed for baseline; bridge remains the mandated path only if SDK semantics are later required). No production code, no `runtimes/dsh/**`, no `internal/agentruntime/**` changes (PATH_OWNERSHIP respected).

## 9. Commit

Not committed — orchestration rule (implementation agent is not the committer). Working tree contains only the three created files from this task.

## 10. L3 completion evidence

- Local sources read: task file P4-03 (contract + L3 pack), `AGENTS.md`, `runtimes/codex/runtime.go` (integration-pattern reference, read-only), report template `reports/A0-05_architecture_readiness_gate.md`.
- External references actually used: the four L3 pack URLs (SAFETY.md, subagent.md, tool-subagent/src/index.ts, codex/claude-code backends note) plus deepseekdocs.com CLI page, dsh-plugin.org run-modes, upstream repo/packages listing — all cited in the spike doc.
- Chosen seam and rejected alternatives: §6 above; narrowest valid seam = one-shot headless process with env-injected auth/cwd-pinned workspace, because it requires no protocol invention, no new Go dependencies, and maps 1:1 onto the proven CodexRuntime exec pattern.
- Negative/failure-path evidence: keyless MISSING_CREDENTIAL probe (structured fail-safe) and SIGTERM cancellation probe (exit 143, no fabricated result).
- Version sensitivity: §7 R-2/R-4.
