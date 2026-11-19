# DSH Provider integration seams (v2.1 candidate record)

As-of: 2026-11-19. Baseline `origin/version/v2@dff5329eed76b88b768aa6802fd92e64fbc83881`.

This file records the seams required to wire `packages/provider-dsh` into the
v2 execution system, **without touching them in this candidate**. Rationale:
the shared entry files are either inside the active #264 R101 writer's frozen
write set (`local-codex-shim.mjs` is literally an FSW rev2 path held today)
or inside #392's frozen single-provider semantics; per the candidate's
constraints the seams are recorded here for the lawful serialized writer.

## 1. Why no wiring is included

- `#392` (frozen amendment): "Exactly ONE selected real Agent provider for
  the v2 installed execution path … do not build a second provider merely to
  satisfy vendor count; no new provider adapter for v2" (multi-provider
  deferred to v2.1). Any route/wiring change would contradict it before a
  lawful v2.1 child version adopts the scope (#283 is INDEX_ONLY).
- Active writer conflict (as-of 2026-11-19): #264 R101 writer
  `zcode:i264:e1:integration:1` holds FSW rev2 (14 paths) including
  `experiments/v2/selected-business-build/local-codex-shim.mjs`, and
  `bundle-local-codex.mjs` was modified by merged #464/#465 the same day.
  `web/v2/**` and `apps/tally-web/src/adapters/**` are lane-E exclusive.
- The candidate's write set is exactly `packages/provider-dsh/**` — zero
  intersection with every active claim/lease as of the baseline above.

## 2. Seam A — provider selection (smallest wiring)

Mirror of the Codex bundle's Host-env authority
(`bundle-local-codex.mjs:328-342` pattern): the **existing** sealed
`tally.runLocalCodexAttempt` provider would read two additional Host env
gates and dispatch to `createLocalDshHost` when both hold:

```
TALLY_ATTEMPT_PROVIDER=dsh AND TALLY_DSH_ENABLED=1
```

`resolveDshSelection()` in `src/dsh-config.mjs` already implements the pure
decision; the seam is one branch in the bundle's attempt orchestration that
constructs `dshConfigFromEnv(env)` and calls `createLocalDshHostWithDhStore()`
instead of `createLocalCodexHost()`. Everything downstream (Task binding,
accepted-Claim verification, FSW verification, source-pair check, CAS store
path, candidate publication) is provider-agnostic and stays unchanged.

Files this seam would touch (today: do not):
- `experiments/v2/selected-business-build/bundle-local-codex.mjs` (#264 R91+
  hot file — serialize behind that writer).

## 3. Seam B — separate capability (v2.1 lawful route)

When a v2.1 child version adopts multi-provider scope, add a dedicated
capability instead of overloading the Codex one:

- `packages/tally-engineering-business/src/index.mjs`: new intent
  `runLocalDshAttempt` (+ `INTENT_TYPES`, policy block, effect block) — the
  producer deep-equals the intent list, so this must land together with
  producer.mjs changes in one review;
- `experiments/v2/selected-business-build/producer.mjs`: seal
  `local-dsh-shim.mjs` + `bundle-local-dsh.mjs` behind a `TDSH_LOCAL=1`
  env-gated module/handler/route branch (append-only pattern used by
  T06N/T06P/T210/T312/T219/T222/T284);
- `experiments/v2/selected-business-build/run-clean.sh`: copy + `node --test`
  lane mirroring lines 152-153;
- the sealed shim would re-apply the Codex shim's shape discipline (state/
  reason whitelists, `authorized/taskPass/releasePass:false`, provider-error
  compression) under `capability-tally-local-dsh@…`.

## 4. Seam C — CI carrier

`.dev-standard/PROJECT_OVERRIDES.md` (v2 CI applicability table) currently
maps new isolated packages (`packages/tally-guided-intake/**`,
`packages/tally-llm-contract/**`) to "dedicated Woodpecker profile not yet
installed; run the package's own exact-SHA Node24 `npm test` in a manual
Woodpecker run or Local/Build Host execution and record it separately".
This candidate follows that row exactly (recorded test receipt, no new
`.woodpecker/` profile). Installing a dedicated profile later is a CI-policy
scope change: route it through `v2-ads-project-verifier.yaml` review.

## 5. Known limitations / deliberate gaps

- Reviewer Join (`joinReviewedCandidates`) is not duplicated here; the
  v2 Reviewer Join consumes Codex-prefixed records today. A v2.1 join should
  read records by grant identity rather than provider prefix (the shared
  `tally:v2:local-agent-grant:` derivation already guarantees physical
  mutual exclusion across providers).
- Cross-provider active-record scanning reads only the DSH prefix; the
  shared grants are the authoritative exclusion mechanism.
- Output-pipe grace is a POSIX `WaitDelay` equivalent: empirically (probed
  2026-11-19), on Windows an inherited grandchild handle does not keep the
  child's stdout open, so `close` fires with the direct child's exit and the
  grace path does not trigger there. Descendants surviving a timeout/cancel
  tree-kill can still occur when the root is already dead (`taskkill /T`
  cannot attach it); bounded by the disposable worktree/DSH_HOME and kept
  visible via kill/termination evidence (same gap class as the Go adapter's
  taskkill snapshot, P4-04 §7).
- The default confinement proof is Windows-principal based; POSIX Host
  wiring (and its confinement story) is future work — it fails closed today.
- `--json` NDJSON events and `--session-id` resume exist in 0.2.x and are
  deliberately unused (projection-only stream; resume requires cwd-stable
  sessions); the adapter keeps the plain headless surface the pin verifies.
- The session-log upload disable overlay is **live-verified**: 0.2.x
  `--patch` overlays must be a top-level YAML ARRAY of loader patch entries
  (the spike-era mapping form is rejected); by default every model request
  carries a ~41 KB `dsh_session_log` payload, and the shipped
  `session-log-deepseek.enabled=false` overlay removes it while turns still
  complete (`smoke/run-dsh-session-log-verify.mjs`, DELIVERY_RECEIPT §5.2).

## 6. Legacy Go adapter (`runtimes/dsh/**`)

Behavioral reference only (crash classification order, WaitDelay necessity,
per-session DSH_HOME, telemetry disabling, honest empty-capability set).
No source reuse — it stays out-of-scope for v2 per PROJECT_OVERRIDES
(OUT_OF_SCOPE_FOR_V2 row) and its pinned version (0.1.5-rc.1) is ~30
releases behind; its two-value permission enum is already stale (0.2.x adds
`read-only`).
