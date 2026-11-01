# T06m Local Codex — L3 Contract, Implementation and Evidence

## Authority and immutable inputs

- Task: #172, T06m real Local Codex execution through selected DomainHarness K/Host.
- Accepted Claim: https://github.com/kaicreator-mm/tally/issues/172#issuecomment-6523851380
- Claim generation: 1; dispatch: `dispatch:172:t06m:local:20261030:1`.
- Intake source: `version/v2@2c6631dbdc3458d4dea43fdc7d78cd292c0aebe9`.
- Frozen DAG write-set SHA256: `d4947ee6cd3c60afee36cc384cb024ee420de5812b11ffd1fdacb2d1801ea100`.
- DomainHarness: immutable Git tag `v0.10.3@061376f4a023070f215f9feffad5c24d96d503cb`; public manifests remain `0.10.2-rc.0` and are packed from that Git checkout, not from an invented npm 0.10.3 release.
- Runtime: native Windows, Node 24.21.0, Codex CLI 0.166.0.
- #169 Claim and T06l historical DH v0.10.2 proof are not reused.

## Production wiring

`runLocalCodexAttempt` is the third Engineering B intent. Its `non-idempotent` effect carries `external: {permitTtlMs, await: terminal}` and routes only to the sealed `tally-local-codex` capability. The C adapter returns the required terminal external-effect envelope; B validates and projects only the bounded observation. No second K, scheduler, task engine, process journal or Agent bus exists.

The sealed provider re-reads the #172 `PUBLISHED_VERIFIED` CAS row and frozen DAG write-set, checks current native GitHub source/dependencies and the exact Claim publication witness, then enters one Tally-owned Local Host capability. Host verifies provider bytes before `Microkernel.load()` and all execution/effect state uses the selected public K/Host SQLite carrier.

The Host sequence is: reserve STARTING → real Codex/authentication probe → exact-base disposable worktree → OS confinement proof → PERMITTED → spawn/RUNNING → candidate write-set readback → terminal CAS. Existing nonterminal or ambiguous attempts return UNKNOWN and never blindly relaunch. A completed candidate is recovered without a new process.

## Builder and credential boundary

The real Builder is `NILAR\CodexSandboxOnline` (`S-1-5-21-2839165489-1832998833-1574237796-1004`), a non-administrator Windows principal. The outer full-package Codex sandbox launcher is distinct from the inner CLI process. Builder receives a dedicated default-profile model session and an explicit environment allowlist; operator GitHub tokens, Woodpecker tokens, SSH agent variables and unrelated environment are not inherited. Git system/global credential helpers are disabled and GitHub push is limited to a dry-run denial probe.

The executed confinement checks proved: exact principal/SID, no administrator/high-integrity token, denied operator GitHub/GCM/SSH/Codex-session reads, no returned GitHub credential, unauthorized remote dry-run, and denied Hidden fixture read. The verifier SID is distinct (`...-1003`). Hidden remains `HIDDEN_NOT_QUALIFIED`; no Hidden PASS is inferred. A CA bundle is explicitly supplied because the sandbox's Windows Schannel path cannot service the Codex TLS request without it.

Builder can only write the single candidate path `experiments/v2/selected-business-build/LOCAL_CODEX_L3.md`. Host independently enumerates changes and rejects any other path. Host, not the Agent, creates the candidate tree/commit/ref. No Agent-originated GitHub write, verification, Task PASS or Release PASS exists. Raw model output, private CoT and credential-bearing stderr are not journaled.

## Real execution evidence

Attempt `attempt:t06m:real:20261030:4` completed the real K-admitted Codex process with exit code 0 and `LOCAL_CANDIDATE_READY`:

```ini
CLAIM_ACCEPTED=true
K_EXTERNAL_PERMIT=EXECUTED
REAL_CODEX_PROCESS=EXIT0
BUILDER=NILAR\CodexSandboxOnline
CANDIDATE_TREE=e404da976b897efed3299195c002edd19da67d3b
CANDIDATE_COMMIT=2ec331fc237a260c74afecf8edbe1b3361b48f68
CANDIDATE_REF=refs/tally-candidates/attempt-t06m-real-20261030-4
K_SQLITE_REOPEN=PASS
SAME_MESSAGE_REPLAY=IDEMPOTENT_ZERO_NEW_DOCUMENTS
INDEPENDENT_PARENT_READBACK=2c6631dbdc3458d4dea43fdc7d78cd292c0aebe9
INDEPENDENT_TREE_READBACK=e404da976b897efed3299195c002edd19da67d3b
CANDIDATE_PATHS=experiments/v2/selected-business-build/LOCAL_CODEX_L3.md
TASK_PASS=false
HIDDEN_PASS=false
```

Evidence is preserved in `C:\xDev\kAiCreator\_tmp\tally-v2-local-agent\evidence-real-4`, including the sanitized consumer log, result JSON and durable K/Host SQLite. An earlier independent minimal spike also produced tree `b46b7ed89ab862fde6081a9ad9b69ab5e8d88953` / commit `554f8dc70936c2d877bc1c6aca99375f2984bfb5`; it was a probe only and is not substituted for the K-admitted attempt above.

## Validation and failure handling

- Engineering B: 43/43 PASS. Covers intent/rule binding, caller-forged authority, malformed attempts, real/replay/UNKNOWN projections and selected-C denial.
- Host: 8/8 PASS. Covers success/recovery, missing or unauthenticated provider, confinement denial, wrong write set, nonzero exit, timeout/process-tree kill, cancellation, lost ACK/restart and environment credential allowlisting.
- Public BUILD/DH v0.10.3 source build/pack and sealed-provider substitution denial: PASS.
- Public K/B/C/SQLite safe replay: `REPLAY_COMPLETE`, provider integrity PASS, SQLite reopen and same-message idempotent replay PASS. Replay has `agentLaunch=false` and is never called a real Agent effect.
- Real Windows attempt: PASS as bounded above; actual process and exact candidate readback, not a container fixture.
- Woodpecker profile: exact authored SHA, full PR-range Refs with shallow-clone recovery, Node24, B tests, immutable DH v0.10.3 build/pack, Host tests and safe replay. Pipeline evidence is recorded on the PR when executed; Docker results are not Windows OS-isolation evidence.

Cancellation is complete only after process-tree termination and reconciliation. Timeout/lost ACK/crash or ambiguous restart is durable UNKNOWN. Missing Claim, stale SHA/generation, blocked dependency, wrong write set, missing/unauthenticated provider and confinement failure all prevent dispatch. Agent output remains an unverified candidate claim. Full regression, Hidden validation, Critical Journeys, Packaging and Release Qualification are separate later gates and are not claimed here.

References: #62, #64, #86, #90, #92, #125, #132, #166, #169, #170, #171, #173, ADS #1241.
