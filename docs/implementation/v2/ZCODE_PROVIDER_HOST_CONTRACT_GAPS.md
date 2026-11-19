# ZCode Provider — Host Integration Contract & Gap Report (R105, Workspace A)

Status: `V21_CANDIDATE` / read-only host survey / base `version/v2@dff5329e` on the
`task/tally-i64-zcodeprovider` branch (Draft PR #470). Requirement refs: #64
(Agent registry / providers), R105 unified `TaskProvider` interface.

This report accompanies the new `createZcodeTaskProvider` facade in
`web/v2/zcode-provider.mjs`. It inventories the exact seams where a **Provider
Registry / Task Dispatch / UI provider selection** would mount the zcode
provider, and classifies every seam as:

- **[R102-WAIT]** — inside the R102 active write set
  (`web/v2/serve.mjs`, `web/v2/journey-server.mjs`,
  `experiments/v2/selected-business-build/local-codex-shim.mjs`,
  `experiments/v2/selected-business-build/bundle-local-codex.mjs`,
  `experiments/v2/selected-business-build/producer.mjs`); must wait for R102
  to release. Nothing in this delivery touches those files.
- **[INJECT-NOW]** — reachable today through injection or new files, without
  editing any existing file.

All `file:line` references were taken from this branch at delivery time.

## 1. Delivered facade surface (`web/v2/zcode-provider.mjs:369-421`)

`createZcodeTaskProvider({spawnFn,operations,now,killTree,host}={})` returns a
frozen five-member object — the unified TaskProvider shape. `host` is test
injection only (fake host, never a real spawn); by default it wraps
`createZcodeProviderHost({spawnFn,operations,now,killTree})`.

| Member | Maps to | Notes |
| --- | --- | --- |
| `providerId` | `ZCODE_PROVIDER_CONSTANTS.providerId` (`'zcode'`, line 22) | constant |
| `probe(config)` | `host.probeProvider(config)` (passthrough) | staged `VERSION` → `CREDENTIALS` → `CANARY`; any failing stage ⇒ `available:false` and dispatch never spawns the task process |
| `execute(input)` | `host.runAttempt(attemptConfig(input))` | unified `ProviderAttemptInput` → existing runAttempt config: `allowedWriteSet` and `taskPrompt` come from the unified input and **override** any copy inside `providerConfig`; every remaining `providerConfig` field is flattened (`cliPath`, `nodePath`, `repositoryPath`, `credentialsPath`, `gitPath`, `canaryTimeoutMs`, `mode`, `briefPath`, `emptyGitConfig`, …); `sourceSha`/`attemptId`/`worktreePath`/`issueRef`/`signal`/`timeoutMs` mapped from the unified input; `hostAuthorization` is carried on the config for future admission wiring (not consumed today — gap G1). Outcome = attempt + `providerId:'zcode'` + `taskId` + a **non-null unified `process` member** (the host returns `process:null` on paths that never reached a spawn; the facade normalizes it to `{status:'UNKNOWN',exitCode:null,pid:null,startedAt:null,terminatedAt:null}`). `taskPass:false` / `releasePass:false` on every path — execution evidence only, never a PASS grant. |
| `cancel(input)` | flips the abort channel only | `input.signal` is the only cancel channel and must expose `abort()` (AbortController-shaped — in Node v24 the abort affordance lives on the controller, while the host listens on `controller.signal`; verified on the runtime). Live channel ⇒ `abort()` + `true`; already aborted ⇒ `false`; missing/non-abortable channel ⇒ `false` with reason token `ZCODE_TASK_PROVIDER_CANCEL CANCEL_CHANNEL_ABSENT` on stderr; never throws. The process-tree kill itself is performed by the host's own abort listener (`zcode-provider.mjs:322-323,326`), never by the facade. |
| `readback(config,candidate)` | `host.joinVerifyCandidate(config,candidate)` (passthrough) | independent git readback (commit parent + tree + changed paths re-derived from the repository); agent-asserted values are never trusted. `taskPass`/`releasePass` stay `false`; verified state is `CANDIDATE_READBACK_VERIFIED` only when parent === `sourceSha` and all paths ∈ allowed set. |

Behavior invariants preserved from PR #470: default provider is `codex`;
zcode is only selectable via `resolveZcodeProviderSelection` (explicit request
AND enabled config AND validated probe); **no failure-driven fallback**; no
second scheduler/kernel/permission system/event log (git-anchored durability
only, candidate ref `refs/tally-zcode-candidates/<attemptId>`).

## 2. Host seam inventory

### 2.1 Provider Registry

| # | Seam | What it holds | Class |
| --- | --- | --- | --- |
| A1 | `web/v2/serve.mjs:291-298` — `startUiReadOnlyServer()`, `TALLY_V2_AGENT_DISPATCH==='host-mounted'` branch | the single provider mount: `readInstalledWriteJourneyEnvironment(env)` + `createInstalledWriteJourneyRunner({settings,env}).run`; claim SHA staleness gate at 294-295 | **[R102-WAIT]** |
| A2 | `web/v2/serve.mjs:440-448` — `startUiReadOnlyServer()`, `createAlmServer(config,{run:installedRun??failClosedStub,…})` | final injection point of the `run` seam into the server; the stub answers `UI_READ_ONLY_NO_AGENT_DISPATCH` | **[R102-WAIT]** |
| A3 | `web/v2/journey-server.mjs:369-380` — `createAlmServer(config,{run,…})` → `createJourneyController(config,{run})` | `run` is one function injected at composition time; there is no runtime provider registry anywhere | **[R102-WAIT]** (registry would extend this options surface) |
| A4 | `web/v2/journey-server.mjs:321-330` — `actionDefinitions(sourceSha)` | action metadata hardcodes `requiredCapability:'WINDOWS_CODEX_HOST'` (line 323) and reason text `'Run real Local Codex…'` (322) | **[R102-WAIT]** |
| A5 | `web/v2/journey-server.mjs:638-664` — `GET /api/v2/agent-attempts` | attempt read model hardcodes `agentId:'local-codex-windows'` (653) and `provider:'codex'` (658); derives rows only from the latest controller result | **[R102-WAIT]** |
| A6 | `web/v2/zcode-provider.mjs:423-433` — `resolveZcodeProviderSelection({requested,zcode,probe})` | the zcode selection policy (default codex; explicit + enabled + probe triple gate; no fallback). Delivered in PR #470; **grep-verified: zero callers** in `web/v2/` or `experiments/` outside the zcode module itself | **[INJECT-NOW]** — callable from any new composition file today |

### 2.2 Task Dispatch

| # | Seam | What it holds | Class |
| --- | --- | --- | --- |
| B1 | `web/v2/journey-controller.mjs:97-111` — `resultOf(kind,config,run)` | kind→script selection (`start` → `local-codex-windows-e2e.mjs`, else `concurrent-codex-windows-e2e.mjs`); an injected `run` takes precedence: line 100 `if(run)return run(kind,env,evidenceDir)` | **[INJECT-NOW]** — zcode mounts via injected run, no file change |
| B2 | `web/v2/journey-controller.mjs:148-156` — `createJourneyController(config,{run})` → `act(kind,sourceSha)` | owner of the `run(kind,env,evidenceDir)` contract; `STALE_SOURCE` gate (152); serializes runs through a pending promise chain (153); `normalize()` applied to every result | **[INJECT-NOW]** |
| B3 | `web/v2/journey-controller.mjs:117-126` — `normalize(result)` | PASS-downgrade rules keyed on `result.result.ackReceipt` / `nativeReadback`. zcode outcomes carry **neither** field ⇒ both downgrades are inert ⇒ a zcode PASS survives untouched. Semantically correct: a zcode candidate is execution evidence, not GitHub-publication evidence | **[INJECT-NOW]** (alignment note, see §3) |
| B4 | `web/v2/installed-write-journeys.mjs:44-92` — `readInstalledWriteJourneyEnvironment(env)`; `CONSUMER_PINS` at 36-40 | trusted-Host pin validation for the codex consumer (`TALLY_CODEX_CLI_PATH`, `TALLY_CODEX_SANDBOX_PATH`, `TALLY_BUILDER_*`, …). Not in the declared R102 five-file set; left untouched here | **[INJECT-NOW]** — a zcode variant is a new file reusing this validation shape with zcode pins (`TALLY_ZCODE_CLI_PATH`/node/credentials/canary budget) |
| B5 | `web/v2/installed-write-journeys.mjs:118-160` — `createInstalledWriteJourneyRunner({settings,env,spawnE2e}).run(kind)` | spawns `local-codex-windows-e2e.mjs` (155-157) and returns the run-contract result shape (146-148); `kind!=='start'` ⇒ `REVIEW_CONSUMER_NOT_MOUNTED` with **no evidenceDir** so the observed-runs projection cannot render a phantom attempt row (150-154) | **[INJECT-NOW]** via new zcode variant file |
| B6 | `experiments/v2/selected-business-build/bundle-local-codex.mjs:311-348` — sealed provider `runLocalCodexAttempt(input)` | K-consumer codex execution entry: claim/write-set/source-pair admission (319-324) then `createLocalCodexHost().runAttempt({…env pins})` (328-342); `TASK_BINDINGS` (51-72) is codex-only; replay lane 325-326 | **[R102-WAIT]** — a zcode path needs a parallel sealed provider entry |
| B7 | `experiments/v2/selected-business-build/local-codex-shim.mjs:84-121` — `computeLocalCodexAttempt(input)` | sealed C shape validation pins the codex result vocabulary **field-by-field** (`REAL_CODEX_CANDIDATE_RECORDED` + `process.builderPrincipal`/`builderSid`, 101-106; reason allowlist `REASONS`, 41-52). A zcode validator must pin the zcode vocabulary instead (`REAL_ZCODE_CANDIDATE_RECORDED`, no builder principal, `CREDENTIALS`/`CANARY` probe stages, git candidate ref fields) | **[R102-WAIT]** |
| B8 | `experiments/v2/selected-business-build/producer.mjs:56-67` — T06M sealing block | seals `local-codex-shim.mjs` bytes, substituting the `__PROVIDER_SHA256__` marker with the sha256 of `T06M_PROVIDER_FILE`; build refuses on digest marker miss | **[R102-WAIT]** |
| B9 | `experiments/v2/selected-business-build/producer.mjs:122-163, 164-204` | sealed build graph: capability module `capability-tally-local-codex@v1.0.0` (externalImports `@kaicreator/tally-local-codex-host-bundle` + `better-sqlite3`) and intent routes (`runLocalCodexAttempt` → `tally.runLocalCodexAttempt` → `tally-local-codex`, 169-170) | **[R102-WAIT]** — zcode would register a parallel capability/intent set here |
| B10 | `experiments/v2/selected-business-build/local-codex-host.mjs:238-254` — `createLocalCodexHost({spawnFn,operations,now,killTree}).runAttempt(config)` | the architectural mirror the zcode host copies (same injectable shape, same state vocabulary); admission differs: sqlite CAS store (`openSqliteCasStore`, 255) vs zcode's git-anchored durability (`attemptRecord`, `zcode-provider.mjs:182-201`). Reference only — no change needed | no seam |
| B11 | `web/v2/zcode-provider.mjs:265-367` — `createZcodeProviderHost(...)` (`runAttempt` 273-350, `joinVerifyCandidate` 351-364, `probeProvider` 365-366) + facade `createZcodeTaskProvider` 369-421 | the zcode execution engine and its new unified interface | delivered (PR #470 + this delivery) |
| B12 | `experiments/v2/selected-business-build/zcode-windows-e2e.mjs` | real-execution driver (produced the PR #470 receipt: CLI 0.16.9, candidate `434ac166`, parent `dff5329e`) | delivered (PR #470) |

### 2.3 UI provider-selection wiring

| # | Seam | What it holds | Class |
| --- | --- | --- | --- |
| C1 | `web/v2/serve.mjs:455-459` — `main()` startup print | `TALLY_UI_AGENT_DISPATCH=HOST_MOUNTED_SINGLE_CODEX\|DENIED` — codex-only label | **[R102-WAIT]** |
| C2 | `web/v2/journey-server.mjs:44` — `CLIENT` inline module | legacy console button label `'Start real Codex'` | **[R102-WAIT]** |
| C3 | `web/v2/journey-controller.mjs:26-27` — `CODEX` const | hardcoded Windows `cliPath`/`sandboxCliPath`. Not in the R102 set. zcode configuration does **not** go here — it travels through the facade's `providerConfig` (`cliPath`/`nodePath`/`credentialsPath`/`canaryTimeoutMs`/`gitPath`/…) | **[INJECT-NOW]** (config-only; nothing to edit) |
| C4 | `web/v2/journey-server.mjs:322-325` | `requiredCapability:'WINDOWS_CODEX_HOST'` (same seam as A4) | **[R102-WAIT]** |

## 3. `run(kind,env,evidenceDir)` contract vs `createZcodeJourneyRun`

Contract holders: `journey-controller.mjs:100` (`resultOf`) and `148-156`
(`act`); the same shape is produced by `installed-write-journeys.mjs:146-148`.
Expected result: `{status:'PASS'|'FAIL'|'UNKNOWN', reason?, exitCode, signal,
timeout:boolean, observedAt, result, evidenceDir}`.

`createZcodeJourneyRun({host,attemptConfig})` (`zcode-provider.mjs:438-453`,
delivered in PR #470) aligns field-for-field:

| Field | zcode mapping | Alignment verdict |
| --- | --- | --- |
| `status` | `LOCAL_CANDIDATE_READY`→`PASS`; `LOCAL_ATTEMPT_UNKNOWN`/`LOCAL_ATTEMPT_CANCELLED`→`UNKNOWN`; `LOCAL_ATTEMPT_FAILED`/`AGENT_NOT_EXECUTED`/`NOT_ADMITTED`→`FAIL` | matches controller semantics (`PASS` requires real effect; lost-ack stays `UNKNOWN`) |
| `reason` | `undefined` on `PASS`, otherwise `attempt.reason` | matches `resultOf`'s convention (`reason` only on non-PASS) |
| `exitCode` | `attempt.process?.exitCode ?? null` | aligned |
| `signal` | always `null` | aligned — the zcode tree kill is `taskkill /T /F` on Windows, no POSIX signal exists |
| `timeout` | `attempt.reason==='ATTEMPT_TIMEOUT_RECONCILED'` | aligned with the controller's `timeout` flag meaning |
| `observedAt` | ISO timestamp at seam return | aligned |
| `evidenceDir` | passthrough of the `run` argument | aligned |
| `result` | `{provider:'zcode', …attempt}` — the full ProviderAttemptOutcome | aligned (`resultOf` embeds `e2e-result.json` here; zcode embeds the in-process outcome — evidence remains git-anchored via the candidate ref) |
| `kind!=='start'` | `{status:'UNKNOWN',reason:'ZCODE_REVIEW_CONSUMER_NOT_MOUNTED',result:null}` | aligned with `installed-write-journeys.mjs:150-154`; like that seam it deliberately omits `evidenceDir` so `/api/v2/agent-attempts` (which requires `observedAt` && `evidenceDir`, `journey-server.mjs:645`) can never render a phantom attempt row |

Interaction with `normalize()` (`journey-controller.mjs:117-126`): a zcode
`PASS` result carries no `ackReceipt` (undefined ≠ `'LOST'`) and no
`nativeReadback`, so neither downgrade fires — `PASS` passes through unchanged.

**One misalignment (gap G5):** `snapshot()` → `task()` gates
`agentAttemptState` on `fresh?.status==='PASS' && summary?.realAgentEffect===true`
(`journey-controller.mjs:162-163`, consumed at 168). zcode attempt outcomes do
not emit `realAgentEffect`, so a zcode-mounted `PASS` would render
`agentAttemptState:'NOT_RUN'` in the snapshot. Resolution belongs to the
owning writer: either emit `realAgentEffect:true` from
`createZcodeJourneyRun`'s `result` when `agentLaunch===true`, or widen the
snapshot gate. `journey-controller.mjs` is outside the declared R102 write
set but was left untouched per workspace discipline (this workspace only
edits `web/v2/zcode-provider.*`).

## 4. Gap ledger

- **G1 — `hostAuthorization` not consumed.** `runAttempt` hardcodes its
  admission flags (`zcode-provider.mjs:274`:
  `claimGeneration:1,claimAccepted:true,admitted:true,authorized:false`).
  The unified input's `hostAuthorization:{claimKey,dispatchId,
  admissionGeneration}` is only carried on the config. Real admission
  (accepted-Claim registry row + publication witness + frozen write-set +
  current source pair — the codex chain at
  `bundle-local-codex.mjs:319-324`) has no zcode path yet. The facade keeps
  zcode default-off until this lands.
- **G2 — no sealed zcode capability.** `producer.mjs`/`local-codex-shim.mjs`/
  `bundle-local-codex.mjs` know only `capability-tally-local-codex`; the
  `runLocalCodexAttempt` intent routes there exclusively (B6-B9). A zcode
  sealed capability + validator (B7-shape) + producer registration must be
  added by the R102 owner.
- **G3 — no provider registry.** `createAlmServer` accepts exactly one `run`
  (A3); multi-provider selection needs a registry at composition time
  (`serve.mjs` A1/A2). `resolveZcodeProviderSelection` (A6) already provides
  the policy half.
- **G4 — UI hardcodes codex.** action metadata (A4), `agent-attempts`
  provider/agentId (A5), button label (C2), startup dispatch label (C1).
- **G5 — `realAgentEffect` snapshot gate** (see §3) — zcode PASS would show
  `agentAttemptState:'NOT_RUN'` until the adapter emits the flag or the gate
  widens.
- **G6 — pre-aborted signal not short-circuited.** `runAttempt` registers its
  abort listener after spawn (`zcode-provider.mjs:323`); an
  `AbortSignal` that is already aborted at dispatch never fires it (abort
  events do not replay), so the attempt runs to timeout. `cancel()` returning
  `false` for an already-aborted channel is consistent with this. A pre-spawn
  `signal.aborted` check inside `runAttempt` is a one-line future fix in the
  zcode module (not done here: no behavior change to existing functions in
  this delivery).
- **G7 — probe stage vocabulary divergence.** codex probe stages are
  `VERSION`/`AUTH` (`local-codex-host.mjs:101-106`); zcode stages are
  `VERSION`/`CREDENTIALS`/`CANARY` (`zcode-provider.mjs:141-181`). A registry
  normalizing probe results across providers must accept both vocabularies
  (the R105 unified zcode probe spec is the latter).
- **G8 — no async/`RUNNING` outcome model.** `runAttempt` awaits terminal
  process state before returning, so a zcode outcome never carries
  `process.status:'RUNNING'`; `cancel(input)` is only meaningful while
  `execute()`'s promise is in flight (the signal channel). A dispatch model
  that returns `RUNNING` outcomes first would need a host-level change
  (out of scope for the facade).
- **G9 — default-off with zero callers.** No existing entrypoint imports
  `zcode-provider.mjs`; the triple gate (explicit request + enabled config +
  validated probe) is fully implemented but dormant until A1/A2 are released.

## 5. What can proceed before R102 releases (new file + injection)

1. Host-local composition entrypoint (new file, e.g. a `serve` variant or a
   driver script) that imports `createAlmServer`/`createJourneyController`
   without editing them and injects
   `run: createZcodeJourneyRun({host:createZcodeProviderHost({...}),
   attemptConfig}).run` (seams B1/B2).
2. `resolveZcodeProviderSelection` + `provider.probe(providerConfig)` gating
   inside that composition (A6), with zcode config supplied via
   `providerConfig` (C3 note).
3. A zcode pin validator variant of `readInstalledWriteJourneyEnvironment`
   (B4) as a new module.
4. Everything already shipped in PR #470: `zcode-provider.mjs` (host +
   selection policy + journey-run adapter + facade), its 24-test matrix, and
   the real-execution driver `zcode-windows-e2e.mjs`.

## 6. Verification

```
node --test web/v2/zcode-provider.test.mjs   → 24/24 pass, EXIT=0
```

Includes the 5 new facade tests (five-member frozen interface + providerId,
execute mapping incl. providerConfig-flattening precedence, null-process
normalization with `taskPass`/`releasePass:false`, cancel three-state
contract incl. `CANCEL_CHANNEL_ABSENT`, readback passthrough). No real zcode
turn was run for this delivery — the PR #470 receipt
(candidate `434ac166`, parent `dff5329e`) stands.
