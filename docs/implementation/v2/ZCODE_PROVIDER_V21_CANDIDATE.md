# ZCode Agent Provider — v2.1 Candidate Delivery (NOT for v2 MVP)

Status: `V21_CANDIDATE` / default-off / base `version/v2@dff5329eed76b88b768aa6802fd92e64fbc83881`.
Requirement refs: #64 (Agent registry/providers), #234 (Settings — `AgentProviderConfig`
id union already contains `'zcode'`). This delivery is Owner-permitted v2.1 candidate
preparation per #392; it changes no v2 MVP completion condition, adds no second
scheduler/kernel/permission system/event log, and does not modify the selected v2
Codex execution path — not a single existing file changed in this PR.

## Delivered surface (all NEW files)

| File | Role |
| --- | --- |
| `web/v2/zcode-provider.mjs` | Provider adapter: capability probe, execution host, selection policy, journey-run seam adapter. Node builtins only (no new dependency; runs in the alm-workspace CI lane as-is). |
| `web/v2/zcode-provider.test.mjs` | 19-test matrix (see below), including a real-git + real-process integration test with a fake ZCode binary. |
| `experiments/v2/selected-business-build/zcode-windows-e2e.mjs` | Host-local Windows real-execution driver (mirrors `local-codex-windows-e2e.mjs`); produced the receipt below. Not wired into CI lanes. |
| `docs/implementation/v2/ZCODE_PROVIDER_V21_CANDIDATE.md` | This document. |

### Exports

- `createZcodeProviderHost({spawnFn, operations, now, killTree})` — same injectable
  shape as `createLocalCodexHost`; `runAttempt(config)` returns the codex-host state
  vocabulary (`LOCAL_CANDIDATE_READY` / `LOCAL_ATTEMPT_FAILED` / `LOCAL_ATTEMPT_UNKNOWN` /
  `LOCAL_ATTEMPT_CANCELLED` / `AGENT_NOT_EXECUTED` / `NOT_ADMITTED`) with zcode-specific
  reasons (`REAL_ZCODE_CANDIDATE_RECORDED`, `WRITE_SET_VIOLATION`,
  `ATTEMPT_TIMEOUT_RECONCILED`, `ATTEMPT_CANCELLED`, `LOCAL_AGENT_UNAVAILABLE`, …).
- `host.probeProvider(config)` — staged capability probe: `VERSION` (`zcode version`
  subcommand; the `--version` flag prints the bundle stub and is NOT used) →
  `CREDENTIALS` (`~/.zcode/v2/credentials.json` with a non-empty `oauth:active_provider`)
  → `CANARY` (one bounded `-p "Reply with exactly: OK"` turn; invalid credentials do
  not fail fast in the real CLI — they stall silently, so a bounded stall is reported
  honestly as unavailable instead of being retried into a fake PASS).
- `resolveZcodeProviderSelection({requested, zcode, probe})` — policy: default
  `codex`; `zcode` only with an explicit request AND an enabled provider row AND a
  validated probe. There is deliberately NO failure-driven fallback: a Codex error
  never switches provider.
- `createZcodeJourneyRun({host, attemptConfig})` — adapts `runAttempt` onto the
  existing `run(kind, env, evidenceDir) → {status,reason,exitCode,signal,timeout,
  observedAt,result,evidenceDir}` controller seam, so mounting is a one-injection
  change for the E writer.
- `zcodeBuilderEnvironment(config)` — explicit env allowlist: no `ZCODE_*` inheritance
  (the desktop-inherited provider pointers are a proven stale-path trap), no
  GitHub/CI credentials, git credential helpers disabled (`GIT_CONFIG_GLOBAL` empty,
  `GCM_INTERACTIVE=never`). The builder never pushes; the Host finalizes the
  candidate ref itself.

### Durability (no second store)

The attempt record is git-anchored: the candidate commit ref
`refs/tally-zcode-candidates/<attemptId>` is the durable candidate record, and
worktree path occupancy is the physical single-writer lock (`git worktree add`
is atomic on path registration). A completed attempt replays from git and is never
relaunched; an occupied worktree without a ref stays `LOCAL_ATTEMPT_UNKNOWN`
(possible running/crashed attempt — honest, no blind relaunch).

## Verified ZCode CLI contract (2026-11-19, local probes)

- Binary: `~/.zcode/runtime/releases/<ver>/bin/zcode.mjs` launched via node;
  CLI core version from `zcode version` (0.16.9 at probe time).
- Non-interactive execution: `-p <prompt> --cwd <worktree> --mode yolo --json
  --no-color`; exit 0 = turn complete with a final JSON object on stdout
  (`sessionId`, `response`, `usage`, `projection`); parse from the first `{`
  (noise lines can precede it). Long prompts must use `--attach` (argv ~8 KB cap).
- No built-in timeout — the host enforces its own and terminates the process
  tree (`taskkill /PID <pid> /T /F`), same as the codex host.
- Agent output is never evidence: the candidate commit, tree and changed paths
  are re-derived from git (`verifyCandidateReadback`), mirroring the codex host.

## Minimal test matrix (`web/v2/zcode-provider.test.mjs`, 19 tests)

1. Available provider positive (probe stages + version + canary).
2. Missing CLI (`MISSING_CLI`), absent credentials (`CREDENTIALS_ABSENT`), stalling
   canary (`CANARY`) — honest unavailable probes; dispatch never spawns.
3. Real execution positive: prompt/cwd/mode/json argv, agent result capture,
   `LOCAL_CANDIDATE_READY`.
4. Write-set fence: changed paths outside the allowed set ⇒ `WRITE_SET_VIOLATION`,
   never a candidate.
5. Nonzero exit ⇒ `LOCAL_ATTEMPT_FAILED/ATTEMPT_PROCESS_FAILED`.
6. Timeout ⇒ kills process tree, `LOCAL_ATTEMPT_UNKNOWN/ATTEMPT_TIMEOUT_RECONCILED`.
7. Cancel (AbortSignal) ⇒ `LOCAL_ATTEMPT_CANCELLED`, no PASS.
8. Spawn error / lost result ⇒ `UNKNOWN`; occupied worktree blocks blind relaunch.
9. Replay from git on a completed attempt (no relaunch); stale ref ⇒ `NOT_ADMITTED`.
10. Env allowlist: no token/credential inheritance, no `ZCODE_*` pointers.
11. Selection policy: codex default, zcode only configured+validated, no fallback.
12. Journey seam mapping incl. `REVIEW_CONSUMER_NOT_MOUNTED` for non-start kinds.
13. Real-git integration: temp repository, real `git worktree add`, real child
    process fake CLI, full chain incl. independent readback and the write-set
    violation variant.

## Real ZCode execution receipt (2026-11-19, Windows host-local)

Driver: `experiments/v2/selected-business-build/zcode-windows-e2e.mjs`
(v2.1 candidate lane, not CI). One-off disposable worktree at
`version/v2@dff5329eed76b88b768aa6802fd92e64fbc83881`.

<!-- ZCODE_RECEIPT:BEGIN — filled from evidence/e2e-result.json -->

- **Result: PASS / realAgentEffect: true** (`e2e-result.json`, status PASS,
  joinState `CANDIDATE_READBACK_VERIFIED`).
- CLI: `~/.zcode/runtime/releases/3.14.3/bin/zcode.mjs` via node v24.21.0,
  core version `0.16.9` (from `zcode version`).
- Probe: available; canary session `sess_72ed3c17…` answered exactly `OK`.
- Execution: one-off worktree at `dff5329eed76b88b768aa6802fd92e64fbc83881`;
  real agent turn, process `EXITED` code `0`, pid 19192, 09:21:42Z→09:36:11Z
  (~14.5 min); agent session `sess_8da15bab…`, 3 model requests, 96,662 tokens.
- Candidate: commit `434ac166716163507460b78a527a43144eb6fba0`
  (parent `dff5329e…`, tree `a6a1c578d01afcfe28903423a6a0ebbca1997201`,
  ref `refs/tally-zcode-candidates/attempt-zcode-i64-20261119-1`), exactly one
  changed path: `experiments/v2/selected-business-build/LOCAL_ZCODE_L3.md`
  (+5 lines). The committed file in this PR is that candidate verbatim.
- Independent readback (re-derived from git, not provider memory): verified —
  parent/tree/paths all match.
- Honest negatives from the same day: two earlier driver rounds failed the probe
  canary honestly (`LOCAL_AGENT_UNAVAILABLE`, stage `CANARY`,
  `CREDENTIALS_OR_PROVIDER_STALL`) while the provider was slow under concurrent
  desktop-session load — the adapter refused to fabricate success; the known-good
  runner baseline stalled identically (exit 124), proving the stall was provider
  slowness, not an adapter/env defect. Canary budget is configurable
  (`canaryTimeoutMs`, default 180 s; this receipt used 300 s).

<!-- ZCODE_RECEIPT:END -->

Evidence (host-local, not committed): `_tmp/tally-i64-zcodeprovider/real-e2e/`
(`e2e-result.json`, `agent-stdout-tail.txt`, `agent-stderr-tail.txt`, driver log).

## Selection & configuration (for the E writer)

```js
// settings row (AgentProviderConfig from apps/tally-web settings model #234)
const zcodeRow={id:'zcode',enabled:false /* explicit opt-in */,cliPath:'C:/Users/<user>/.zcode/runtime/releases/<v>/bin/zcode.mjs',...};
const probe=await host.probeProvider(config);
const selection=resolveZcodeProviderSelection({requested:taskProvider, zcode:zcodeRow, probe});
// selection.selected === 'zcode' only when requested AND enabled AND probe.available
```

## Remaining integration (deliberately NOT done here — #264's exclusive write set)

The provider registry/mount points live in #264-owned files; this PR is
new-file-only by coordination decision (no writer conflict with R101):

1. `web/v2/serve.mjs` / `web/v2/journey-server.mjs` (E writer): mount
   `createZcodeJourneyRun(...)` as the `run` seam when selection admits zcode
   (fail-closed stub otherwise), and label `provider:'zcode'` where journey-server
   hardcodes `provider:'codex'` (journey-server.mjs:653-658).
2. Settings persistence (#234): the settings model already accepts the zcode row;
   wiring `GET/PUT /api/v2/settings` to store it is #234/#264 surface.
3. `apps/tally-web` UI labels for a zcode provider row (settings.test.ts already
   fixtures `{id:'zcode', enabled:false, buildSelected:false}`).

## Known limitations (honest boundaries)

- **OS-level confinement**: unlike the Codex elevated sandbox (separate low-privilege
  principal + ACLs + confinement proofs), the ZCode CLI runs as the operator user.
  The write-set fence is enforced post-hoc inside the disposable worktree (git diff
  + untracked listing), and the builder env carries no GitHub/CI credentials with
  git credential helpers disabled — but writes OUTSIDE the worktree by a
  misbehaving agent are not machine-enforced. A future zcode sandbox host
  (codex-windows-sandbox-style) is v2.1 follow-up work.
- **Auth failure = silent stall**: the real CLI hangs instead of erroring on bad
  credentials; the probe's bounded canary distinguishes "verified healthy" from
  "unavailable (credentials/provider/stall — indistinguishable from outside)".
- **Exit-code anomaly**: one observed healthy turn exited 127 after printing its
  answer; the adapter stays strict (nonzero exit ⇒ `LOCAL_ATTEMPT_FAILED`) — the
  agent's stdout and the git candidate are captured as evidence for reconciliation.
- CI (alm-workspace lane) exercises the fake-binary matrix only; real-CLI runs are
  host-local Windows evidence, same doctrine as the Codex lanes.
