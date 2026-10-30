# T06n Concurrent Local Codex — L3 Contract, Implementation and Evidence

## Authority and immutable inputs

- Task: #176 (T06n concurrent independent Builders with Reviewer Join and Repair), parent #86/#152/#90; depends on completed #172 (T06m) and #173 (Woodpecker-only CI). ADS v4.18 #1241.
- Implementation Claim: https://github.com/kaicreator-mm/tally/issues/176#issuecomment-6527320123 (generation 1, dispatch `dispatch:176:t06n:impl:20261030:1`).
- Builder A Claim: #177 comment 6527324131 (`dispatch:177:t06n:builder-a:20261030:1`); Builder B Claim: #178 comment 6527327548 (`dispatch:178:t06n:builder-b:20261030:1`); Reviewer Claim: #179 comment 6527331202 (`dispatch:179:t06n:reviewer:20261030:1`). All sourceSha `ca09511d49488785c9ae8ff436c5db98f31b6714`, generation 1.
- Candidate A is exactly `experiments/v2/selected-business-build/CONCURRENT_A.md`; candidate B is exactly `experiments/v2/selected-business-build/CONCURRENT_B.md`.
- DomainHarness: immutable `v0.10.3@061376f4a023070f215f9feffad5c24d96d503cb`, built and packed from Git source. The T06m single-Task attempt and #172 Claim are NOT reused as either concurrent Builder.

## Production wiring

The existing single K/Host and the sealed `tally-local-codex` capability are reused; no second scheduler, K, task engine, process journal or Agent bus is introduced. `runLocalCodexAttempt` remains the only non-idempotent external effect. A new fourth intent `joinReviewedCandidates` carries a separate **idempotent** effect (`effect:tally-reviewer-candidate-join`, with its own K external permit, `tally.joinReviewedCandidates` C operation). The producer selects the join route/handler only when `T06N_CONCURRENT=1`; default T06m/T06l build output is unchanged.

The sealed provider keeps a Host-owned task binding map for issues #172/#177/#178 (builder, one candidate path each, defaulting to the existing `LOCAL_CODEX_L3.md` for #172) and #179 (reviewer). Caller input may select only these pins; every claim/registry/write-set value comes from issue-keyed Host env (`TALLY_TASK_CLAIM_<issue>_*`, legacy `TALLY_CLAIM_*` for #172) or a Host-owned CAS registry row, never from caller authority. Each issue's accepted-Claim registry is a separate Host-prepared store (`TALLY_CLAIM_REGISTRY_PATH` for #172, `TALLY_TASK_CLAIM_<issue>_REGISTRY_PATH` for #177/#178/#179); claims are never copied into one synthetic shared authority. Each Builder Claim registry row, source pair, publication witness, operator/writer/comment/dispatch is independently verified. Per-attempt disposable worktrees are derived per attempt ID under `TALLY_LOCAL_WORKTREE_ROOT` (or the repository path), so two Builders never share a worktree; the Host additionally denies a new attempt when an active record holds the same worktree path (`CANDIDATE_WORKTREE_OVERLAP`) or the same candidate write grant (`CANDIDATE_WRITE_GRANT_OVERLAP`).

`consumer-concurrent-codex.mjs` loads the sealed App once, then uses ONE K facade with `Promise.all` over two distinct instance keys/messages/attempt IDs so the two real external Codex effects can overlap in process lifetimes. Host records and returns the actual process lifetime (`startedAt` captured at spawn ACK, `terminatedAt` at terminal close; the earlier attempt reservation timestamp stays a separate field) so overlap windows and `overlapMs` are computed from real process intervals, never from probe/reservation time. The Reviewer Join (`joinReviewedCandidates`) verifies the independent #179 reviewer Claim, rejects reviewer/builder self-assertion (`REVIEWER_SELF_ASSERTION`), then checks each candidate against its durable Host attempt record and an independent git readback: exact commit, tree, single changed path, exact parent `sourceSha`, record disjointness by attempt/path/worktree/ref (`CANDIDATE_NOT_FOUND`, `CANDIDATE_NOT_READY`, `CANDIDATE_STALE`, `CANDIDATE_READBACK_MISMATCH`, `CANDIDATE_WRITESET_OVERLAP`). Wrong/stale/missing/overlap Joins are demonstrated before any valid Join. A bounded Repair uses one new attempt ID and a fresh worktree; the prior FAILED/candidate record is preserved verbatim, and a re-Join consumes the repaired candidate. Candidate or review output is never Task PASS, Hidden PASS or Release PASS.

## Builder and credential boundary

Unchanged from T06m: both Builders run under the least-privilege Windows principal and sandbox boundary, with the explicit environment allowlist, disabled Git credential helpers, denied operator GitHub/GCM/SSH/Codex-session reads, denied Hidden fixture read, and no GitHub write credential. Each Builder writes only its own candidate path in its own disposable exact-base worktree at `ca09511d49488785c9ae8ff436c5db98f31b6714`. The Reviewer performs no process launch and no source write.

## Real execution evidence

Real Windows attempt set `attempt:t06n:real:a:3`, `attempt:t06n:real:b:3`, `attempt:t06n:real:repair:3` completed successfully. The two real Codex process intervals overlapped for **13149 ms**; overlap was computed from spawn/terminal timestamps, not reservation time.

```ini
REAL_PROCESS_OVERLAP_MS=13149
CANDIDATE_A_TREE=cfccc2b8fe31608588f39d81754caaefa6bccec3
CANDIDATE_A_COMMIT=aac503ff8f9e9c9bef820a9130ae4541bd1a15ec
CANDIDATE_B_TREE=a8c39ab847742399df162c40c643b0406fb2a3e8
CANDIDATE_B_COMMIT=7e0770d3780acbce8a5146226cef66eb0fdfe738
REVIEW_WRONG=CANDIDATE_STALE
REVIEW_STALE=CANDIDATE_STALE
REVIEW_MISSING=CANDIDATE_NOT_FOUND
REVIEW_OVERLAP=CANDIDATE_WRITESET_OVERLAP
REVIEW_JOIN_1=REVIEW_JOINED
REPAIR_RESULT=LOCAL_CANDIDATE_READY
REVIEW_JOIN_2=REVIEW_JOINED
K_SQLITE_REOPEN=PASS
SAME_MESSAGE_REPLAY=IDEMPOTENT_ZERO_NEW_DOCUMENTS
INDEPENDENT_CANDIDATE_READBACK=PASS
TASK_PASS=false
HIDDEN_PASS=false
```

Evidence is preserved in `C:\xDev\kAiCreator\_tmp\tally-v2-t06n\evidence-3`, including the sanitized consumer log, result JSON, independent readback and durable K/Host SQLite. Earlier wrong-Claim-pin and invalid-overlap-fixture runs remain failure provenance and were not rewritten as PASS.

## Validation and failure handling

- Engineering B: 49/49 PASS, including T06n reviewer-Join rule/binding/transition, K external permit declaration, idempotency grammar, input screening, replay/denial projections, forged-observation fail-closed and selected-C-only dispatch.
- Host: 14/14 PASS: existing T06m recovery/confinement/write-set/timeout/cancellation/lost-ACK/restart suites unchanged, plus candidate-path configuration, real overlapping lifetimes with actual timestamps, worktree/write-grant overlap denial, reviewer Join success/idempotency, all six denial classes, and bounded repair with preserved prior evidence and no relaunch.
- Public BUILD/DH v0.10.3 source build/pack and sealed-provider substitution denial: PASS (run-clean producer lane).
- Public K/B/C/SQLite safe replay: `REPLAY_COMPLETE` markers only, provider integrity PASS, SQLite reopen and same-message idempotent replay PASS. Safe replay is explicitly marked `T06N_SAFE_REPLAY_ONLY_NO_REAL_OVERLAP_NO_AGENT_EFFECT` and cannot be mistaken for real execution.
- Woodpecker profile `.woodpecker/v2-t06n-concurrent-agent.yaml`: exact authored SHA, PR-range `Refs:` trailers with shallow-clone recovery, Node 24, B tests, immutable DH v0.10.3 build/pack, Host tests and safe replay only. Docker is NOT Windows OS-isolation evidence and the profile says so explicitly.

Cancellation is complete only after process-tree termination and reconciliation; timeout/lost ACK/crash or ambiguous restart is durable UNKNOWN and never blindly relaunched, independently per Builder. Missing/stale Claims, blocked dependencies, overlapping write grants/worktrees, wrong write sets, missing/unauthenticated provider and confinement failure all prevent dispatch. Full regression, Hidden validation, Critical Journeys, Packaging and Release Qualification are separate later gates and are not claimed here.

References: #62, #64, #86, #90, #92, #125, #132, #152, #166, #169, #170, #171, #172, #173, #177, #178, #179, ADS #1241.
