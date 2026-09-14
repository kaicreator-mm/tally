# P1-06 Execution Report — Risk-adjusted acceptance guard

- **Status:** DONE
- **Date:** 2026-09-12
- **Baseline:** A0+P0 complete, P1-01…P1-05 DONE (progress 19/57 at start)
- **Release blocker:** `P1-06 assurance acceptance guard` — RESOLVED by this task

## 1. Seam adjudication (SPIKE_REQUIRED → frozen by this task)

DECISION_LOCKS.md "Open seams" row `Acceptance-guard exact placement (Closed Loop Engine vs ServiceVerifier)` is owned and now closed by this task.

**Ruling: the guard's enforcement point is the Closed Loop Engine's accept transition; the ServiceVerifier is rejected as the enforcement point and remains a verification-fact passthrough.** The evaluation semantics live in one pure evaluator in `internal/taskstate` (PATH_OWNERSHIP assigns the taskstate acceptance semantic helper to P1-06/P1-07; this task lands it).

### Source facts the ruling is based on

1. **Accept authority is the Engine's (DL-01).** The only place a verified candidate becomes Accepted is the `VerdictVerified` branch of `internal/closedloop/engine.go` (`step(PhaseVerifying, EventVerified) → PhaseAccepted`); the non-Accept fail-safe path (typed `Escalation` vocabulary) is likewise an Engine capability. A guard that must convert "verified" into "not accepted" needs exactly these two powers.
2. **Precedent: the P2-06 exposure guard.** `engine.go` already evaluates an accept-time guard at the same spot (`l.e.ledger.Compromises(lineage, out.Candidate.CreatedAt())` → escalate `exposure_compromise`). The assurance guard is the same shape; placing it elsewhere would make two accept guards live at two different layers.
3. **`verdictForStatus` is a fact-preserving projection (internal/closedloop/verify.go).** `verification.FinalVerified` is a true statement of the verification authority. Re-mapping it inside `ServiceVerifier.VerifyCandidate` would:
   - `→ VerdictFailed`: fabricate a candidate failure the verifier never issued (ADR-0001 §3 defines FAIL as "验证器给出了可信失败判定"), and the failure router classifies from the recorded `FinalOutcome` — `ClassifyFinalStatus(FinalVerified) = FailureUnknown → escalate` (internal/closedloop/failure.go), i.e. the loop would escalate anyway, only with corrupted verdict facts en route;
   - `→ VerdictInconclusive`: fabricate an ERROR ("无法给出可信 verdict") and misuse the inconclusive channel as an acceptance block;
   - a fourth verdict word (`VerdictLimited`): rewrites the closed verdict vocabulary and still requires an Engine-side non-accept ruling — so the Engine seam is unavoidable in every variant.
4. **The required-assurance authority is `contract.Compiled`, not the frozen verification contract.** Required levels (risk-adjusted uplift, unmatched-signal failsafe) exist only in the compiled authority (P1-03 `CompileAuthority` → `Clauses()`/`Coverage()`), which the worker already produces as a mandatory pre-execution gate (P1-04). `ServiceVerifier` is constructed from `LoopConfig{Verify, Contract, Runtime}` and is by role an adapter for the *verification* service; making it hold the compiled authority would turn a verification adapter into an acceptance authority (blurs "Runtime output is not Verification/Acceptance").
5. **Rejected alternative "evaluate only in taskstate.Derive (read model)":** the projection has no authority over the runtime loop; it is P1-07's read surface.

### Resulting shape

- `closedloop.Config.AcceptGuard` (consumer-defined narrow interface, pure function) consulted on the accept transition, after the exposure guard, before `EventVerified`; denial → escalate `assurance_insufficient` (new typed `EscalationReason`).
- `taskstate.EvaluateAcceptanceGuard` — the one pure semantic evaluator; assurance math delegated verbatim to `assurance.Achieved` + `Level.Satisfies` over `contract.Compiled.Clauses()` (Forbidden boundary respected: no second assurance implementation; the gap conditions are exactly the `VerifiedBasis` VERIFIED_LIMITED triggers from `projectVerifiedBasis`).
- `closedloop.CompiledAcceptanceGuard` — adapter binding the compiled authority + frozen contract visibility; wired by `NewLoop` whenever `LoopConfig.Authority` is present; `cmd/worker/v14.go` always passes it (authority is mandatory there since P1-04).
- `ServiceVerifier` gains only a defensive projection: `VerifyOutcome.Coverage` = `FinalOutcome.Functional.Acceptances` (facts, not rulings).

## 2. Changed files

- `internal/taskstate/acceptguard.go` (new): `AcceptanceGuardInput` / `AcceptanceGuardDecision` / `EvaluateAcceptanceGuard` (+ `clauseFacts` coverage→`assurance.ExecutedFacts` projection).
- `internal/taskstate/acceptguard_test.go` (new): focused adversarial tests (AC1, AC2, waiver, fail-safes).
- `internal/closedloop/acceptguard.go` (new): `AcceptGuard` port, `AcceptGuardRequest`/`AcceptGuardRuling`, `CompiledAcceptanceGuard` adapter, `ErrInvalidAcceptanceGuard`.
- `internal/closedloop/acceptguard_test.go` (new): engine-level denial/allow/repair-attempt tests, adapter constructor + ServiceVerifier projection tests, NewLoop production-composition tests with a real `Compile→Bind` authority.
- `internal/closedloop/engine.go`: `Config.AcceptGuard` field, Engine wiring, guard evaluation on the accept transition.
- `internal/closedloop/ports.go`: `VerifyOutcome.Coverage` field (typed observation face).
- `internal/closedloop/verify.go`: `ServiceVerifier` projects `FinalOutcome.Functional.Acceptances` defensively into `VerifyOutcome.Coverage`.
- `internal/closedloop/result.go`: `EscalationAssuranceInsufficient = "assurance_insufficient"`.
- `internal/closedloop/wiring.go`: `LoopConfig.Authority *contract.Compiled`; `NewLoop` constructs and injects the guard when the authority is present.
- `internal/closedloop/integration_test.go`: `lpVerifyStep.functional` additive field (fake can now mirror a real `FinalOutcome`).
- `cmd/worker/v14.go`: `LoopConfig.Authority: authority.Compiled` (guard becomes a production composition fact) + comment updates.
- `cmd/worker/v14_test.go`: `v14FakeVerify` now mirrors the real orchestrator contract (FinalVerified ⇒ complete functional coverage report, synthesized from the frozen contract; explicit override field added); two new P1-06 production-path tests; `TestV14PreventionContextReachesRuntime` terminal-state expectation updated (see §6).

## 3. Semantic changes

- A base `FinalVerified` candidate can no longer become Accepted when the compiled authority requires a higher assurance level than the observed verification facts achieve, when unmatched risk signals exist, or when waived requirements exist: the loop escalates with `assurance_insufficient` (fail-safe, human/policy adjudication; not a candidate failure, no repair budget consumed, no verdict fabricated — the guard runs before the Router).
- VERIFIED_LIMITED semantics now gate the runtime accept path, with gap vocabulary identical to the read projection (satisfied/waived remain distinct; unknown/unmatched signals verbatim in the escalation detail).
- Missing coverage observation = no verification facts = L0 (never treated as satisfied); unknown binding visibility contributes no facts.
- Guard denial detail is deterministic and carries clause ids, required vs achieved levels, waived ids and unmatched signal ids (no held-out material exists in the input space).

## 4. Tests / commands executed (exact)

| Command | Result |
|---|---|
| `go build ./...` | PASS |
| `go vet ./...` | PASS |
| `gofmt -l` on all touched files | clean (two pre-existing unformatted neighbors untouched) |
| `go test ./internal/taskstate` | PASS |
| `go test ./internal/closedloop` | PASS |
| `go test ./internal/assurance ./internal/contract` | PASS |
| `go test ./test/arch/...` | PASS (DL-01/DL-05/DL-07 guard suite green) |
| `go test ./cmd/worker -run TestV14` | PASS (13.8s) |
| `go test ./internal/...` | PASS (all packages; only `core`/`store` have no tests) |
| `DATABASE_URL=postgresql://postgres:postgres@localhost:15432/postgres go test ./cmd/worker` | PASS (49.2s; includes DB-gated `TestV14ClosedLoopPersistsFacts`, `TestV14RepairSupersessionPersists` — production composition with real candidateverify + runstore + guard) |
| `DATABASE_URL=… go test -tags=integration ./test/e2e -run TestV14` | PASS (187s; v14 closed-loop e2e harnesses on real temp Git repos) |
| `git diff --check` | clean |
| `make check` | NOT_RUN (no make binary in Git Bash; constituent families run individually as above — same ruling as P1-03/04/05) |

Key new tests (negative/failure paths per contract):

- taskstate: `TestAcceptanceGuardRequiredL3AchievedL1NeverAcceptable` (AC1), `TestAcceptanceGuardUnmatchedRiskDeniedEvenWhenAssuranceSatisfied` (AC2, assurance satisfied yet denied, signal verbatim), `TestAcceptanceGuardWaivedIsLimitedDistinctFromUnmet` (VERIFIED_LIMITED/waiver semantics), `TestAcceptanceGuardMissingCoverageFailsSafe`, `TestAcceptanceGuardUnknownVisibilityYieldsNoFacts`, `TestAcceptanceGuardNilCompiledRejected`, plus a positive plain-accept control.
- closedloop: `TestEngineAcceptanceGuardDeniesInsufficientAssurance` (typed escalate, no basis, single attempt, Router bypassed), `TestEngineAcceptanceGuardAllowsSufficientCandidate`, `TestEngineAcceptanceGuardFiresOnRepairedCandidate` (FAIL→repair→verified-with-gap still blocked, supersession lineage preserved), `TestNewCompiledAcceptanceGuardRejectsNilInputs`, `TestServiceVerifierProjectsCoverageForGuard` (defensive copy), `TestLoopGuardBlocksRequiredL3WithAchievedL1` (AC1 through real `NewLoop` composition), `TestLoopGuardBlocksMissingCoverageObservation` (fail-safe), `TestLoopGuardAllowsSufficientCandidate`, `TestLoopWithoutAuthorityKeepsLegacyAccept` (explicitly pins the nil-authority composition semantics).
- worker (production path, real repos): `TestV14AcceptanceGuardBlocksInsufficientAssurance` (matched policy → L3 required; FinalVerified + visible facts → escalated `assurance_insufficient`, detail carries `AC-01 required L3 achieved L1`, no repair), `TestV14AcceptanceGuardBlocksUnmatchedRisk` (unmatched `auth_symbol_change` surfaced verbatim in the escalation detail); first-pass/repair/persist suites remain green (guard allows the L1-sufficient control).

## 5. Baseline failures observed

None. Pre-existing gofmt deviations in untouched neighbor files (`cmd/worker/main.go`, `v13*.go`, `internal/closedloop/exposure_test.go`, `internal/taskstate/state*.go`, …) were already present at the baseline and were deliberately not reformatted (no line-ending normalization pass; `.gitattributes` untouched).

## 6. Known semantic overlay on a P1-05 test

`TestV14PreventionContextReachesRuntime` uses the policy-matched fixture (required L3) whose candidate can now never be auto-accepted from visible-only facts (achieved L1) — exactly the product semantics this task introduces. Its Prevention-context assertions (instruction snapshot, compiled-aware hash re-derivation) are unchanged and still prove execution happened; only the terminal-state assertion changed from `accepted` to `escalated/assurance_insufficient`, with an in-test comment dated 2026-09-12 explaining the P1-06 overlay. This is an intended semantic consequence, not a regression.

## 7. Remaining risks / scope notes

- `LoopConfig.Authority == nil` keeps v1.4 acceptance behavior (no guard bound) so that existing test/trial compositions (`test/e2e` harnesses, `scripts/v14_trial`) outside this task's allowed paths keep compiling and expressing their own semantics. The production composition (`cmd/worker`) always supplies the authority; the repo-wide no-bypass gate is P1-08's owned task (`compiled_authority_no_bypass_gate`).
- The closed loop has no waiver channel in v1.4.1; the evaluator keeps `Waived` as an input (single-owner semantics) and the runtime wiring passes none — waiver read-model consumption remains P1-07/P3 work.
- A policy-matched (elevated-risk) task with a visible-only verification contract now deterministically escalates instead of auto-accepting; recovering the auto-accept path requires held-out (L3) evidence facts passing — by design (Decision Lock #21).

## 8. Decision Locks

Not bypassed, none modified. DL-01 (engine keeps Accept authority; no engine rewrite — one guarded branch mirroring the exposure guard), DL-05 (risk.Result untouched), DL-07 (now mechanically enforced), DL-12 (Outcome domain untouched), DL-17 (hidden/ unread), DL-18 (extend-not-duplicate: taskstate/closedloop extended; the open-seams row `Acceptance-guard exact placement` is now frozen by its owning task as the locks prescribe).

## 9. Completion report (task contract format)

1. **status:** DONE
2. **changed files:** see §2 (4 new files, 8 modified)
3. **semantic changes:** see §3
4. **tests/commands:** see §4
5. **baseline failures:** none (see §5)
6. **remaining risks:** see §7
7. **Decision Locks touched:** none
8. **commit:** recorded in git log (`P1-06: …`)
