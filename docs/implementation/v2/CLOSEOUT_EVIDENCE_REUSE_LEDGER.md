# Closeout Evidence Reuse Ledger — v2 Core MVP closeout (#392/#259/#264)

What evidence each round REUSES (unchanged bytes/contract) versus re-runs
fresh. R97 §3: reuse unchanged exact Blob/contract reviews; no repetitive
review cycles; no automatic full-suite reruns without affected-risk evidence.

## R90 (PR #451, merged ccf7e69b) — as recorded in receipt #392/6984159498

- Reused: E0 version-task-dag host module + its http regression (contract
  unchanged, one guard tightened); #449 U1/U3/U4 merged UI fixes;
  journey-browser-e2e harness (no-browser witness, injected run seam).
- Fresh: serve.mjs composition, journey-server v2 models, host.ts/App.tsx
  DTOs, installed Chrome smoke (provisional), CI #836 on the exact head.

## R91 (this branch, task/i264-r91-installed-write from c1523a23)

Reused unchanged (source identity intact — no re-review required):

- `web/v2/journey-controller.mjs` — the sanctioned `run` seam contract and
  normalize()/snapshot() semantics (NOT in this round's write set; zero diff).
- `web/v2/journey-browser-e2e.mjs` + test — harness unchanged; witness re-run
  as the CI-prescribed regression (ok:true ×13).
- `packages/tally-github-native-c/src/ui-version-readonly.mjs` — native
  catalog/qa readers reused as-is; `nativeC.qa()` now feeds the real release
  evidence projection (read-only reuse of an existing contract).
- E0 `version-task-dag-host-http.test.mjs` import-resolution guard — untouched;
  passes on this branch (per-package install as in CI).
- Frozen DAG read composition, roster host, intake/maintenance controllers —
  untouched.

Fresh (reviewed at this exact HEAD):

- NEW `web/v2/installed-write-journeys.mjs` — the E1 installed write-journey
  runner (Host pin admission + existing consumer spawn + result contract).
- `web/v2/serve.mjs` — run-seam mount (env-pin gated), releaseHost, real
  release projection, truthful boot output.
- `web/v2/journey-server.mjs` — GET `/api/v2/agent-attempts`;
  `/api/v2/release` mounted-provider passthrough.
- `apps/tally-web/src/adapters/host.ts`, `src/app/App.tsx` — agent attempts
  read model + release action submission (same-origin, operator action key).
- `web/v2/installed-write-journeys.test.mjs` — extended seam regression.

Lease continuity: fresh current-source witness dispatch
`dispatch:264:e1:integration:host:20261119:2` at c1523a23 (comment
6994840214, supersedes the r90 witness at 6e7209e7); the 13-path FSW write
set rebinds hash-identically (`70cde9c9…` verified against the compact-JSON
recipe). Host-side lease registries live outside the repo
(`host-registries/i264-e1-claim/*.sqlite`, claim rev 2 / FSW rev 1,
planIdentity `376ce6047286d9b1…` over the documented recipe).

Known NOT_RUN at this HEAD (honest queue, not silent gaps): real Windows
Codex attempt through the mounted seam (requires the Host-supplied consumer
pins + DH checkout at `061376f4…`); two-builder T06N review join (not lawful
under one protected claim key); release delivery sealed consumer (none exists
at this source); #208 exact-SHA RQ; native predecessor dispositions
(#260/#261/#273 remain open blocked_by edges of #264; #276 closed not_planned).
