# provider-dsh delivery receipts (v2.1 candidate)

Baseline: `origin/version/v2@dff5329eed76b88b768aa6802fd92e64fbc83881`.
Platform: Windows (win32 10.0.26300 x64), Node `v24.21.0` (fnm default).
All receipts below were produced from the exact working tree committed in
this PR (source under `packages/provider-dsh/`).

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
