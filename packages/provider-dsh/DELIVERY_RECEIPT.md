# provider-dsh delivery receipts (v2.1 candidate)

Baseline: `origin/version/v2@dff5329eed76b88b768aa6802fd92e64fbc83881`.
Platforms: Windows (win32 10.0.26300 x64, Node v24.21.0) and Linux
(CC-LAN200 Ubuntu, Node v24.21.0). All receipts below were produced from the
exact working tree committed in this PR (source under `packages/provider-dsh/`).

## 1. Package contract tests — PASS

Command (package directory):

```
npm test            # node --test test/*.test.mjs
```

Result (2026-11-19, local Build-Host execution per
`.dev-standard/PROJECT_OVERRIDES.md` "new isolated package" CI row — no
dedicated Woodpecker profile exists for this package yet):

```
ℹ tests 26
ℹ pass 25
ℹ fail 0
ℹ skipped 1
ℹ duration_ms ≈ 3.8–4.7s     exit code 0
```

The single skip is the POSIX-only real orphan-pipe observation: on Windows
an inherited grandchild handle does not keep the child stdout open
(empirically probed 2026-11-19), so that live assertion is skipped on win32
with the platform reason recorded in the test; the grace mechanism's WIRING
is asserted by the injected fake-child test on every platform.

Coverage mapping to the required scenarios: CLI missing / version
incompatibility / missing credential / confinement failure never spawn;
successful exit with candidate file + independent git readback; write-set
violation refused; cancel / timeout / process-tree termination; lost-ACK
UNKNOWN without relaunch; orphan-pipe grace wiring; empty-final-answer
fail-safe; secret redaction in records; grant races; cross-provider (Codex)
grant exclusion; default-off two-gate selection.

## 2. Real DomainHarness v0.10.3 store integration — PASS

`createLocalDshHostWithDhStore` (explicit `openSqliteCasStore` injection)
drove one full attempt against the real public SQLite CAS store, packed from
the DomainHarness checkout at the CI-pinned SHA `061376f4a023070f215f9feffad5c24d96d503cb`
(tag v0.10.3):

```json
{"REAL_DH_STORE_CHECK":{"state":"LOCAL_CANDIDATE_READY",
 "reason":"REAL_DSH_CANDIDATE_RECORDED","candidateCommitShaShaped":true,
 "finalAnswerSha256Shaped":true,"realStoreRecordPhase":"CANDIDATE_READY",
 "recordHasPermissionMode":true,"recordNoSecret":true,
 "node":"v24.21.0","platform":"win32"},"PASS":true}
```

## 3. REAL DSH CLI smoke — PASS (gated one-shot)

CLI: `@deepseek-ai/dsh@0.2.2-rc.5` installed isolated under `_tmp` (no
global mutation), invoked via `lib/bin.js` under Node. Gates:
`TALLY_DSH_SMOKE_CONFIRM=1` + `DEEPSEEK_API_KEY` present (value never
printed/recorded). Receipt (`smoke/run-dsh-smoke.mjs`):

```json
{"REAL_DSH_SMOKE":"PASS","pinnedVersion":"0.2.2-rc.5",
 "versionCheck":{"exitCode":0,"observedVersion":"0.2.2-rc.5","pinMatched":true},
 "positiveRun":{"exitCode":0,"tokenMatched":true,"answerBytes":19,
   "stderrTail":"[llm-deepseek-api-key] model listing from https://api.deepseek.com/models loaded 2 model(s), skipped 0 row(s)\n"},
 "negativeKeylessRun":{"exitCode":1,"missingCredentialShape":true,
   "stderrTail":"dsh: MISSING_CREDENTIAL: llm-deepseek: no API key for provider route \"deepseek-official\"; ..."},
 "sessionLogDisable":"ARRAY_FORMAT_LIVE_ACCEPTED_UPLOAD_OFF_SOURCE_DERIVED",
 "isolation":"per-attempt throwaway DSH_HOME"}
```

Live findings encoded back into the adapter this round:
- 0.2.x `--patch` overlays must be a **top-level YAML array** of loader patch
  entries (the spike-era mapping form is rejected with
  "must be a top-level YAML array");
- `api.deepseek.com` is directly reachable with no proxy (the adapter's
  proxy is opt-in, unlike the Codex builder env default).

NOT proven by the smoke (and not claimed): Host-isolated Builder execution,
Task/Review/Release PASS, session-log upload-off effect (source-derived
only), Windows OS-principal confinement against the real CLI.

## 4. CI carrier

Per `.dev-standard/PROJECT_OVERRIDES.md` (v2 CI applicability table, row
"New isolated v0.10.3 contract-first Tally source"), this package has **no
dedicated Woodpecker profile installed yet**; the lawful carrier is the
exact-SHA Node 24 package `npm test` recorded above. Installing a dedicated
profile is a CI-policy scope change reserved for the ADS project-verifier
review (see DSH_PROVIDER_SEAMS.md §4).

## 5. Follow-up verification round (2026-11-19, same-day gap closure)

### 5.1 POSIX test coverage — PASS (26/26, zero skips)

Full suite on CC-LAN200 (`cc@192.168.100.200`, Ubuntu, Node v24.21.0,
`node --test test/*.test.mjs`): `tests 26 / pass 26 / fail 0 / skipped 0`,
exit 0, ~2.4s. The POSIX-only real orphan-pipe test ran live: the grandchild
really holds the pipe, the grace timer really destroys the host's pipe ends,
and the process-group termination really reaps the orphan. (Note: on WSL
Node v22.23.2 the same suite aborts mid-run with node:test
"Promise resolution is still pending but the event loop has already
resolved" — a Node-22 runner behavior with fake children and unref'd timers;
Node 24, as required by this package, runs it green.)

### 5.2 Session-log upload disable — LIVE VERIFIED (was source-derived)

`smoke/run-dsh-session-log-verify.mjs`: a local Anthropic-protocol capture
server is injected via `DEEPSEEK_BASE_URL` (the documented llm-deepseek
fallback; dummy key only, zero real API calls), then the real
`@deepseek-ai/dsh@0.2.2-rc.5` runs headless twice:

- control (no overlay): 2/2 `/messages` requests carry `dsh_session_log`,
  ~41,426 bytes each — the default-on upload ships the session log suffix
  with **every model request**;
- with the shipped overlay (`session-log-deepseek.enabled=false`): 0/2
  requests carry the field; both turns complete normally (token matched,
  exit 0).

Verdict: `UPLOAD_OFF_LIVE_VERIFIED`. This upgrades the overlay from
"source-derived" to live-verified, and quantifies the outbound data the
v2.1 Host must not leak.

### 5.3 REAL Windows-principal verification — PASS (after interactive repair)

Machine: `nilar`, builder principal `nilar\codexsandboxonline`
(S-…-1004), verifier S-…-1003, Codex CLI 0.166.0 elevated backend.

Findings and repairs along the way (all machine-level, recorded):

- The new `OPERATOR_DSH` denied-read check initially refused the run on a
  real finding: `C:\Users\15549\.dsh` carried an explicit
  `Nilar\CodexSandboxUsers:(OI)(CI)(RX)` ACE, letting the confined builder
  enumerate operator DSH state. Removing the ACE proved non-durable — the
  authorized elevated setup re-run **re-granted** it — so the fix mirrors
  the Codex baseline's own technique (its `auth.json` carries a builder
  file-level DENY): `icacls ~\.dsh /deny *S-…-1004:(OI)(CI)(R)`. A DENY
  always wins over the group grant and survives setup re-runs.
- The elevated orchestrator mid-round required the documented one-time
  interactive repair (`run-elevated-setup.ps1` via UAC, operator-approved);
  alternatives were verified contract-ineligible live: `unelevated` runs as
  the operator principal (`nilar\15549`), `mxc` runs with Administrators.

Final receipt (`smoke/run-dsh-windows-confinement.mjs`, `PASS: true`):

- Stage 1 probe: `dsh --version` **inside the sandbox, as the confined
  builder** — `available:true`, observed `0.2.2-rc.5` == pinned;
- Stage 1 confinement matrix: `verified:true` (principal, SID, no-admin,
  verifier-distinct, hidden-fixture deny, operator `~/.dsh` + `~/.ssh`
  denied reads, GCM refused, remote-write refused);
- Stage 2 full confined attempt: `LOCAL_CANDIDATE_READY /
  REAL_DSH_CANDIDATE_RECORDED`, one real model turn via the Host sandbox
  wrapper (`sandboxWrapper: HOST_SANDBOX`), process exited 0 under
  `builderPrincipal: nilar\codexsandboxonline`; the real model wrote exactly
  `packages/provider-dsh/DSH_CANDIDATE_NOTE.md` inside the disposable
  worktree; the Host-created candidate tree/commit verified by independent
  git readback (`parent` == source SHA, single path); the durable record
  contains the 24-key env allowlist and **no credential material**.

Honest scope: this proves the provider under real Windows principal
confinement at Host level. It is still NOT a Task/Review/Release PASS and
not a v2 installed-path execution (the package remains default-off, outside
#392's single-provider freeze).
