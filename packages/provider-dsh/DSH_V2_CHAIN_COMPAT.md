# DSH ↔ v2 execution chain compatibility matrix (R105 workspace B)

As-of: 2026-11-19. Baseline `origin/version/v2 @ dff5329eed76b88b768aa6802fd92e64fbc83881`
(merge-base of this candidate branch). Worktree: `tally-i64-dsh-zcode`, branch
`task/tally-i64-dsh-zcode` (Draft PR #467).

Scope: verify that `packages/provider-dsh` (PR #467) is compatible with the
existing v2 execution chain — K/Host composition, Claim admission, worktree
discipline, CAS, Candidate Readback — and add the unified R105 TaskProvider
facade. All 对照 (reference) files are READ-ONLY; every fix below stays inside
`packages/provider-dsh/**`.

Verdict vocabulary: **OK** = verified compatible, no change needed; **FIX** =
in-package gap found and fixed in this round (with tests); **REMAINING_INTEGRATION**
= lawful seam that must land outside this package (see `DSH_PROVIDER_SEAMS.md`).

| # | Chain element | 对照 file(s) | Verdict |
|---|---------------|--------------|---------|
| 1 | K/Host 组合链 | `bundle-local-codex.mjs` + `local-codex-host.mjs` | **FIX + REMAINING_INTEGRATION** (host shape OK after facade; probe tier fixed) |
| 2 | Claim admission | `consumer-claim-admission.mjs`, `claim-admission-shim.mjs` | **OK** (layer-equivalent) + **REMAINING_INTEGRATION** (claim wiring) |
| 3 | 工作树 (worktree + refs) | `local-codex-host.mjs`, `bundle-local-codex.mjs` | **FIX** (ref namespace) / worktree discipline **OK** |
| 4 | CAS | `local-codex-host.mjs`, `bundle-local-codex.mjs` | **OK** (namespaces disjoint; shared grant derivation by design) |
| 5 | Candidate Readback | `local-codex-host.mjs` | **OK** (line-equivalent) + facade `ready` gate added |

---

## 1. K/Host 组合链 — how the Codex chain assembles the host, and the DSH shape

Reference (all read-only):

- `bundle-local-codex.mjs:29-45` — the sealed provider source imports
  `createLocalCodexHost` from `local-codex-host.mjs` and lets esbuild bundle it
  (externals pinned to `better-sqlite3` + `@kaicreator/domain-harness-host-node`,
  `bundle-local-codex.mjs:859-881`).
- `bundle-local-codex.mjs:328-342` — `runLocalCodexAttempt` composes the host
  with **no injection at all**: `createLocalCodexHost()` bare (default `spawn`,
  default operations, `now=Date.now`, default `taskkill` terminate; the SQLite
  store is hard-imported at `local-codex-host.mjs:6,255`), then calls exactly
  one member: `.runAttempt({issueRef,sourceSha,attemptId,candidateRelativePath,
  worktreePath,hostStorePath,repositoryPath,cliPath,...builderEnv})`.
- `bundle-local-codex.mjs:343-347` — the catch mapping: any host throw is
  flattened to `LOCAL_ATTEMPT_UNKNOWN / ATTEMPT_STATUS_UNKNOWN / agentLaunch:true`
  with a fully-shaped UNKNOWN `process` record.

Findings:

- **Injectable shape is同构 (isomorphic).** `createLocalCodexHost({spawnFn,
  operations,now,killTree})` (`local-codex-host.mjs:238-240`) vs
  `createLocalDshHost({spawnFn,operations,now,killTree,openStore})`
  (`dsh-host.mjs`) — same four ports plus the store factory the Codex host
  hard-imports. The injected `operations` surfaces have identical member names
  and signatures on both sides: `{probeProvider,proveConfinement,prepareWorktree,
  finalizeCandidate,verifyCandidateReadback,sandbox}` (`local-codex-host.mjs:230`
  vs `dsh-host.mjs` default ops return). The DSH host is therefore drop-in
  compatible with the bundle's composition pattern.
- **Outcome flag parity.** Both hosts emit the same base flags per attempt
  (`claimGeneration:1, claimAccepted:true, admitted:true, authorized:false,
  taskPass:false, releasePass:false`) and the same `overlapDenied` shape
  (`local-codex-host.mjs:233-236,247-249` = `dsh-host.mjs`).
- **FIX (probe tiers).** R105 requires a three-tier `probe`:
  VERSION → CREDENTIALS → CANARY, any failed tier ⇒ `available:false` and the
  attempt never spawns. The delivered package had only two tiers (VERSION pin +
  AUTH credential). Fixed in `dsh-host.mjs:284-304`: a third CANARY tier re-runs
  the pinned `--version` round-trip through the credential-carrying
  execution-shaped environment (`DSH_TELEMETRY_MODE=DISABLED`,
  `DSH_PERMISSION_MODE`, `DEEPSEEK_API_KEY` present) and the same argv
  composition an attempt takes (sandbox wrapper respected). `--version` is the
  only source-verified no-model-call invocation of dsh 0.2.x, so the canary
  proves the real exec path without touching the model API. Failure returns
  `{available:false, stage:'CANARY', canaryMatched:false}`; success adds
  `canaryVerified:true`. The facade maps host stage `AUTH` → tier
  `CREDENTIALS` and exposes `tiers:{VERSION,CREDENTIALS,CANARY}` with
  PASS/FAIL/NOT_RUN per tier.
- **FIX (outcome parity).** The unified outcome spec includes
  `candidateRef`/candidate paths; the host recorded `candidateRef` in the
  durable record but never returned it. `runAttempt` outcomes now surface
  `candidateRef` everywhere and `candidatePaths` on `LOCAL_CANDIDATE_READY`
  (`dsh-host.mjs`), matching the shim's candidate shape discipline
  (`local-codex-shim.mjs` join output carries `candidateRef ?? null`).
- **FIX (cancel channel).** A signal aborted *before* listener registration
  never fires its `abort` event, so the attempt would run to timeout under a
  dead cancel channel. The host now classifies from intent immediately
  (`dsh-host.mjs:596`, `if(config.signal?.aborted)abort()`), consistent with
  the "SIGTERM drains gracefully, classify from adapter intent" contract.
- **REMAINING_INTEGRATION.** No DSH dispatch branch exists in the sealed
  bundle/shim lane (`local-codex-shim.mjs` exposes only
  `tally.runLocalCodexAttempt` et al.). The one-branch seam (Host env gates +
  `dshConfigFromEnv` + `createLocalDshHostWithDhStore`) is recorded in
  `DSH_PROVIDER_SEAMS.md` Seam A and must be executed by the lawful serialized
  writer of `bundle-local-codex.mjs` / `local-codex-shim.mjs` (R102/R264 write
  set) — not in this candidate. The unified facade (`dsh-task-provider.mjs`)
  gives that seam a ready-made provider object.

## 2. Claim admission — aligning the admitted Claim context

Reference (read-only):

- `consumer-claim-admission.mjs:53-57` + `:96-97` — the accepted Claim registry
  row: `phase:'PUBLISHED_VERIFIED'`, `generation:1`, `protectedClaimKey`
  (`owner/repo#<issue>:<role>:__default__`), `sourceSha`, `dispatchId`,
  `operatorKind/operatorId/writerId`, `commentId/commentRef`.
- `claim-admission-shim.mjs:56-94` — the sealed C verifies exactly that shape
  and emits `ADMITTED_PENDING_AGENT_EFFECT` with `dispatchId`, `generation:1`,
  `protectedClaimKey`; authority flags all false.
- `bundle-local-codex.mjs:119-189` — the Codex chain verifies the accepted
  Claim (durable registry row + native GitHub publication witness) **upstream
  of the host**: `verifyAcceptedClaim` → `verifyFrozenWriteSet` (`:237-305`) →
  `verifyCurrentSourcePair` (`:190-202`) → only then `createLocalCodexHost().runAttempt`.

Findings:

- **Layer-equivalence: OK.** The DSH host sits at exactly the same layer as the
  Codex host: downstream of Claim admission, upstream of any Task/Release
  semantics. Neither host receives or verifies claim rows; both hardcode the
  attempt-level flags `claimAccepted:true, admitted:true, claimGeneration:1`
  (the *Claim* admission is the upstream chain's job). The DSH host reuses this
  unchanged — no new claim semantics invented.
- **Facade alignment: OK.** The R105 unified input carries
  `hostAuthorization:{claimKey?,dispatchId?,admissionGeneration?}`. The facade
  accepts it and echoes it onto every outcome (`hostAuthorization`) as
  caller-supplied evidence, and refuses a mismatched `providerId` envelope
  (`DSH_PROVIDER_ID_MISMATCH`). It does **not** verify claims — correctly:
  claim verification belongs to the sealed upstream chain.
- **REMAINING_INTEGRATION.** The DSH equivalent of
  `runLocalCodexAttempt`'s upstream gate (`verifyAcceptedClaim` +
  `verifyFrozenWriteSet` + `verifyCurrentSourcePair` before
  `host.runAttempt`) does not exist; those functions live inside the sealed
  bundle (`bundle-local-codex.mjs:119-305`) and cannot be duplicated in this
  package without creating a second admission authority. Until Seam A/B lands,
  a DSH attempt must only be dispatched by a chain that has already admitted
  the Claim (enforced by the two-gate selection + probe, never assumed).

## 3. 工作树 — worktree discipline and ref namespaces

Reference (read-only):

- `local-codex-host.mjs:161-179` (`prepareWorktree`: one-shot
  `git worktree add --detach <path> <sha>`, `CANDIDATE_WORKTREE_ALREADY_EXISTS`
  on occupied path, HEAD readback = `CANDIDATE_BASE_MISMATCH` guard,
  icacls grants); `:205` candidate ref
  `refs/tally-candidates/<sanitizeId(attemptId)>`.
- `bundle-local-codex.mjs:306-310` — codex default worktree root
  `<root>/.tally-t06n-worktrees/<safe>`.
- `local-codex-host.mjs:274-306` — atomic grants (`mustBeAbsent` put) acquired
  BEFORE the attempt record is created, released once a terminal record exists,
  retained on crash for operator reconciliation; plus the active-record overlap
  scan within the provider's own attempt prefix.
- R105 brief: the ZCode provider (workspace C) owns `refs/tally-zcode-candidates/`.

Findings:

- **One-shot worktree semantics: OK.** `dsh-host.mjs` `prepareWorktree` is the
  Codex algorithm plus DSH-specific additions (throwaway `DSH_HOME` beside the
  worktree, session-log disable patch, ACL for `dshHome`/sandbox home). Same
  "occupied path ⇒ refuse" and "HEAD ≠ sourceSha ⇒ CANDIDATE_BASE_MISMATCH"
  behavior.
- **Path occupancy lock equivalence: OK.** Grant acquisition ordering,
  `mustBeAbsent` reservation, revision-matched release, and crash-retention are
  code-equivalent to the Codex host. The grant key derivation is byte-identical
  (`tally:v2:local-agent-grant:`), so a DSH and a Codex attempt can never hold
  the same physical worktree or candidate path concurrently — asserted by the
  in-package test `grant derivation excludes cross-provider (Codex) resource
  holders`, which pre-seeds a Codex-derived grant key and asserts
  `NOT_ADMITTED / CANDIDATE_WORKTREE_OVERLAP` with `agentLaunch:false`.
- **FIX (ref namespace).** The delivered package wrote candidates to
  `refs/tally-candidates/<id>` — the **Codex** prefix. Attempt records live
  under disjoint CAS prefixes (`tally:v2:local-dsh-attempt:` vs
  `tally:v2:local-agent-attempt:`), so neither host could detect a collision,
  and `git update-ref` overwrites unconditionally: a DSH and a Codex attempt
  with the same sanitized attemptId would silently destroy one provider's
  candidate ref. ZCode already owns a provider-unique prefix
  (`refs/tally-zcode-candidates/`); DSH now owns
  `refs/tally-dsh-candidates/` (`dsh-host.mjs:76`, applied at `:414`, exposed as
  `LOCAL_DSH_CONSTANTS.candidateRefPrefix`). A dedicated test asserts the
  prefix differs from both the Codex and Zcode namespaces; the e2e test asserts
  real candidates land under it. Codex/zcode files untouched.
- **Default worktree root: OK (by construction).** Codex derives
  `.tally-t06n-worktrees/` in the bundle; DSH takes the worktree path from Host
  config (`TALLY_LOCAL_WORKTREE_PATH`, no in-package default derivation) —
  distinct by configuration, and physical collisions are excluded by the shared
  grants regardless of prefix conventions.

## 4. CAS — store identity, key namespaces, concurrency

Reference (read-only):

- `local-codex-host.mjs:6,255,405` — `openSqliteCasStore` from
  `@kaicreator/domain-harness-host-node`, `busyTimeoutMs:5000`, per-attempt
  open/close.
- `local-codex-host.mjs:12-14` — attempt keys
  `tally:v2:local-agent-attempt:<id>`, grants
  `tally:v2:local-agent-grant:<kind>:<sha256(value)>`; join keys
  `tally:v2:reviewer-join:` (`:457`).
- `bundle-local-codex.mjs:75-76,461-462,669-672` — other chain namespaces in
  the same store family: `tally:v2:claim-admission:`, `tally:v2:task-writeset:`,
  `tally:v2:version-dag-effect-fence:`, `tally:v2:release-rq:`,
  `tally:v2:release-grant:`.

Findings:

- **What `createLocalDshHostWithDhStore` is: OK.** It wraps exactly the public
  `openSqliteCasStore` from `@kaicreator/domain-harness-host-node` — the same
  store factory the Codex host hard-imports — adapting its
  `{path,busyTimeoutMs}` call shape (`dsh-host.mjs:666-671`). Standalone
  (non-npm) usage injects any store with the get/put/list/remove +
  `mustBeAbsent`/`mustMatchRevision` surface; the test double
  (`test/memory-cas-store.mjs`) mirrors that contract.
- **Key prefix disjointness: OK.** DSH attempt records use
  `tally:v2:local-dsh-attempt:` (`dsh-host.mjs:66`) — disjoint from the Codex
  attempt prefix, join prefix, claim/writeset/release/fence/dag namespaces
  above. The ONLY deliberate overlap is the grant prefix
  `tally:v2:local-agent-grant:` (`dsh-host.mjs:69`), byte-identical by design:
  grants guard *physical* resources (worktree paths, candidate paths) and must
  exclude across providers, not just within one.
- **Same-store concurrency: OK.** Both hosts open the store per attempt with
  `busyTimeoutMs:5000` and rely on CAS conditions (`mustBeAbsent` for
  reservations/grants, `mustMatchRevision` for record updates) — the DH
  SQLite store's documented concurrency contract; the overlap-scan asymmetry
  (each host scans only its own attempt prefix) is compensated exactly by the
  shared grant derivation, and a crash before any terminal record retains the
  grants for operator reconciliation in both implementations.
- **REMAINING_INTEGRATION.** The in-package tests run on the memory double;
  the real SQLite receipt (same DB file concurrently written by a Codex attempt
  and a DSH attempt) belongs to the Phase 5 real-execution round together with
  the DeepSeek API capacity.

## 5. Candidate Readback — independent git readback

Reference (read-only):

- `local-codex-host.mjs:212-229` — `verifyCandidateReadback`: re-derives from
  the repository `commit^{tree}`, `commit^`, `diff --name-only sourceSha commit`;
  returns `{verified,candidateTree,parent,paths}` — "never from Agent-asserted
  values"; used by the Reviewer Join only (`:434-438`, where the full gate
  `readback.verified && candidateTree===record.candidateTree &&
  parent===config.sourceSha && paths===1 && paths[0]===candidateRelativePath`
  denies with `CANDIDATE_READBACK_MISMATCH`).

Findings:

- **Host-level equivalence: OK.** `dsh-host.mjs` `verifyCandidateReadback` is
  line-equivalent to the Codex implementation (same three git queries, same
  derived fields, `verified:false` on any git failure or non-SHA commit).
  The DSH `runAttempt` additionally does its own in-worktree readback inside
  `finalizeCandidate` (`rev-parse commitSha^{tree}` must equal the written
  tree) before recording `CANDIDATE_READY` — same as Codex.
- **"Parent/tree/paths 全匹配才算 READY" gate: OK at facade level.** The Codex
  host applies that full-match gate only inside `joinReviewedCandidates`; DSH
  has no join (deliberate, `DSH_PROVIDER_SEAMS.md` §5). The R105 facade
  therefore exposes `readback(config,candidate)` = independent ops readback
  **plus** the ready verdict: `ready=true` requires `verified`,
  `candidateTree` match, `parent===config.sourceSha`, and paths exactly
  `[candidatePath]` (default
  `packages/provider-dsh/DSH_CANDIDATE_NOTE.md`); anything else (tree/parent
  mismatch, extra path, missing readback port) is `ready:false`, verified only
  from git — never from agent-asserted values. A full DSH Reviewer Join that
  consumes this verdict stays **REMAINING_INTEGRATION** (Seam B).

---

## Fixes applied in this round (all inside `packages/provider-dsh/`)

1. `src/dsh-host.mjs` — three-tier probe (VERSION → AUTH/CREDENTIALS → CANARY),
   canary = credential-carrying execution-shaped `--version` round-trip, no
   model API call (`probeProvider`, `dsh-host.mjs:284-304`).
2. `src/dsh-host.mjs` — candidate ref namespace
   `refs/tally-dsh-candidates/` (constant `:76`, use `:414`,
   `LOCAL_DSH_CONSTANTS.candidateRefPrefix` `:675`), removing the silent
   cross-provider ref collision with the Codex prefix.
3. `src/dsh-host.mjs` — `runAttempt` outcomes surface `candidateRef` (all
   branches) and `candidatePaths` (READY), matching the R105 unified outcome
   shape and the shim's candidate shape discipline.
4. `src/dsh-host.mjs` — pre-aborted cancel signals now cancel immediately
   (`dsh-host.mjs:596`) instead of running to timeout under a dead channel.
5. `src/dsh-task-provider.mjs` (new, 90 lines) — unified R105 TaskProvider
   facade: `providerId:'dsh'`, tiered `probe`, `execute` envelope mapping
   (single-candidate `allowedWriteSet` enforcement, `taskId`/`hostAuthorization`
   echo, UNKNOWN-on-throw mapping mirroring the sealed bundle's catch),
   `cancel` three-state semantics via the execute-time signal only, `readback`
   with the full-match ready gate; frozen five-member object;
   `taskPass`/`releasePass` are false in every outcome shape (envelope denials,
   host outcomes, UNKNOWN catch — asserted per test).
6. `src/index.mjs` — exports `createDshTaskProvider`.
7. `test/fake-dsh.mjs` — `--fake-counter` / `--fake-version-fail-from` to make
   the canary tier fail-closable in tests.
8. `test/dsh-host.test.mjs` — canary tier test, ref-namespace test,
   pre-abort cancellation test, `candidateRef`/`candidatePaths` assertions
   (fake + real e2e).
9. `test/dsh-task-provider.test.mjs` (new) — 11 facade tests: frozen members,
   execute mapping, envelope refusals (no spawn), pre-abort refusal, UNKNOWN
   catch mapping, cancel three states, probe tier mapping (fake + real CLI),
   readback ready gate (4 mismatch classes + fail-closed port), selection
   semantics reuse.

## Remaining integration gaps (outside `packages/provider-dsh/**`)

- **Seam A/B** (`DSH_PROVIDER_SEAMS.md` §2-3): DSH dispatch branch in the
  sealed bundle/shim lane, incl. the upstream Claim/FSW/source-pair gate that
  mirrors `bundle-local-codex.mjs:311-348`; the new Business intent
  (`runLocalDshAttempt`) + producer sealing under `TDSH_LOCAL=1`.
- **Selection wiring**: `resolveDshSelection` is the pure decision, but nothing
  in the v2 path consults it yet (R102 frozen write set).
- **Reviewer Join** for DSH candidates (consume the facade's `readback.ready`
  verdict) — v2 join reads Codex-prefixed records today.
- **Real receipts**: real SQLite same-store concurrency with a parallel Codex
  attempt, and real DeepSeek API execution receipts — Phase 5 (controller).
