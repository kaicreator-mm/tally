# UI Web-First R1 Acceptance — v2 installed entrypoint (E lane #264)

Authoritative record of what the installed `web/v2/serve.mjs` entrypoint
actually serves and admits, per round. Evidence semantics are strict:
`NOT_RUN`/`UNKNOWN`/`STALE`/`FAILED` stay distinct; a clickable control never
asserts success before a fresh canonical receipt.

## R90 — installed entrypoint minimal repair (merged PR #451, base 62f18fd8, head 2bcf756c)

Delivered (receipt #392/6984159498):

- P0-ENTRYPOINT: the five unresolved `ui-*-host.mjs` static imports removed;
  real `node web/v2/serve.mjs` boot emits `TALLY_UI_READ_ONLY=<origin>/app`
  and `TALLY_UI_AGENT_DISPATCH=DENIED`.
- Real Task/DAG read models via `v2ReadModelHost` (no raw open-Issue fallback,
  no synthetic settings/metadata); absent providers answer NOT_CONNECTED.
- UI truthfulness DTOs (host.ts), MetadataRoute real arrays, honest
  `connected:false` for requirements/qa/maintenance/agents.
- Two real-browser defects fixed: CSP `style-src` for the Vite stylesheet;
  honest 204 for `/favicon.ico`.
- Installed Chrome smoke (Node 24, Chrome 157, playwright-core channel
  chrome): catalog connected, tasks/dag fail-closed
  FROZEN_DAG_HOST_NOT_CONNECTED, settings/metadata NOT_CONNECTED, release
  UNKNOWN×2; 0 console errors.
- Woodpecker pipeline #836 SUCCESS on the exact PR head.

NOT_RUN after R90: frozen-reader positive 51/29 roster read; J02/J04/J05 write
journeys; J06–J08 real installed journeys.

## R91 — installed write-journey seam (this branch)

Source facts at this exact HEAD:

- **J04 write path mounted (`run` seam)**: `serve.mjs` mounts the real
  selected single-Codex Windows consumer
  (`experiments/v2/selected-business-build/local-codex-windows-e2e.mjs`) as
  the journey-controller's sanctioned `run` hook, ONLY when the trusted Host
  process exports `TALLY_V2_AGENT_DISPATCH=host-mounted` plus the exact
  claim/lease pins (`TALLY_V2_AGENT_*`). The attempt binds to the exact
  served source SHA — a stale claim refuses START closed
  (`AGENT_CLAIM_SOURCE_SHA_STALE`). Without the pins the R90 fail-closed
  stub stays; boot prints `TALLY_UI_AGENT_DISPATCH=DENIED` truthfully
  (HOST_MOUNTED_SINGLE_CODEX when mounted).
- **Review seam honesty**: no two-builder review consumer is lawful under one
  protected claim key, so `review-repair` answers
  `UNKNOWN REVIEW_CONSUMER_NOT_MOUNTED` with no evidence directory — it never
  renders as an executed attempt. Independent source review stays the
  reviewer-conversation flow (R97 §2.C).
- **Agent read model**: new GET `/api/v2/agent-attempts` projects ONLY real
  executions of this server (observedAt + evidenceDir + exit/timeout). The
  R90 stub run and the unmounted review seam are invisible; missing
  telemetry (duration/tokens/cost/model) is null (UNKNOWN), never zero.
  `AgentsRoute` consumes it (previously hardcoded `attempts={[]}
  connected={false}`).
- **Release action seam**: `ReleaseRoute` submits through the real J07c
  admission (GET `/api/release/snapshot` for the exact source SHA, then POST
  `/api/release/actions` with the operator `?actionKey=`). `serve.mjs`
  supplies a real `releaseHost` evidence reader (native commit status when
  `TALLY_V2_QA_STATUS_CONTEXT` is configured). Without a mounted delivery
  seam the controller denies with
  `INDEPENDENT_HOST_ADMISSION_NOT_CONNECTED` after the real evidence read;
  permits stay null so no fake enabled button exists.
- **Release evidence**: `/api/v2/release` uses the mounted
  `v2ReadModelHost.release` (real woodpeckerCI gate + honest UNKNOWN
  independentRq); the advisory UNKNOWN×2 projection remains the documented
  default for an unmounted facade.

Regression suite: `web/v2/installed-write-journeys.test.mjs` covers the pin
admission matrix, the injected-runner start/ACK-lost/fail/timeout contract,
stale-claim START refusal, stub-invisibility of the attempts projection, the
release passthrough and the real fail-closed release admission — all offline,
zero credentials.

## Test matrix (round-level exit codes)

| Gate | Exit | Notes |
| --- | --- | --- |
| web/v2 + package tests (CI `alm-workspace-tests` command) | 0 | 343 pass / 0 fail / 3 skipped |
| `node web/v2/journey-browser-e2e.mjs --no-browser` witness | 0 | ok:true; INJECTED_RUN_SEAM; realWindowsCodex NOT_RUN; browser GUI_NOT_PROVEN |
| apps/tally-web `node --test` (pure TS) | 0 | 100 pass |
| `vite build` | 0 | 68 modules; new bundle hash |
| `tsc --noEmit` | 2 (baseline) | single pre-existing `task-actions/model.ts` null-narrow error, file untouched this branch; triaged per R97 §2.A |
| release-panel-negative greps (CI step) | 0 | all eight assertions hold |

Real installed Windows Host + Chrome journey: see the round receipt posted to
#264/#392 (exact SHA, browser actions, exit codes). This file never declares
Task PASS, RQ PASS or Release PASS — those remain Owner/Host gates.
