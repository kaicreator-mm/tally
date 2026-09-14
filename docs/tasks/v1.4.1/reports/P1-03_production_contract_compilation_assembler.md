# P1-03 — Production contract compilation assembler: Execution Report

- **Status:** DONE
- **Commit:** recorded in the task-file addendum / final message
- **Date:** 2026-09-12

## 1. Deliverable

`internal/artifacts/v141_contract_assembler.go` — the production seam (PRD §16:
Frozen Contract → Compiler → Policy Match → Risk → Assurance → CompiledContract
→ Execution; the worker must not bypass the compiler) from the same-source
execution bundle (P1-01 `LoadBundle`) to the authoritative compiled contract:

```go
func (b *ExecutionBundle) CompileAuthority() (*CompiledAuthority, error)

type CompiledAuthority struct {
    TaskID         string              // bundle-consistent task identity
    SourceCommit   string              // observed single source commit (DL-04)
    BundleIdentity string              // ExecutionBundle.Identity()
    Risk           risk.Result         // authoritative derived risk input (DL-05: derived value, never persisted)
    Compiled       *contract.Compiled  // Hash/Pins/Coverage/Clauses/FrozenGates/Requires/GateKeys
}
```

Assembly order (fixed, fail-safe first):

1. nil-intent guard (hand-built bundle → typed error, no panic);
2. source-commit consistency re-check — `b.Policies.SourceCommit != b.SourceCommit`
   → typed `ErrSourceMismatch` (DL-04 defense at the assembly entry; loader
   output is always equal, this catches bundles assembled by hand);
3. policy-loader rejections are fail-safe — `len(b.Policies.Rejected) > 0` →
   typed `ErrPolicyRejections`, no `Compiled` produced (this is the adjudication
   P1-01 explicitly deferred to P1-03);
4. authoritative risk derivation — `b.Intent.Intent.DeriveRisk()`; typed errors
   (`intentevidence.ErrInvalidEvidence`, `risk.ErrMalformedSignal`) pass through
   verbatim wrapped (`%w`): invalid risk basis stops execution (the exact
   delegation P1-02 recorded), never "no risk" derived from non-evidence;
5. `contract.Compile(b.Task.Contract, b.Verification.Frozen)` — the existing
   contract authority; content-derived pins act as the second identity defense
   (its typed sentinels, e.g. `contract.ErrTaskMismatch`, pass through);
6. `contract.Bind(snap, policies, riskFacts)` — the existing compiler, zero
   semantic change (DL-04: pure projection over the snapshot; policies passed in
   load order with their Parse-filled revision pins);
7. result exposes task id, observed source commit, bundle identity and the
   authoritative `risk.Result` next to `Compiled.Hash()` (implementation
   requirement 3: compiled identity/hash exposed to the caller).

## 2. Why this is the narrowest valid seam (and rejected alternatives)

- **`internal/artifacts` (chosen, method on `*ExecutionBundle`):** the bundle
  already carries every input and its identity; the assembler is the bundle's
  own "compile my authority" operation, living next to the loader that
  guarantees DL-04. `artifacts` already imports `engineeringpolicy`,
  `intentevidence`, `taskcontract`, `verification`; adding `contract` creates
  no cycle and follows the existing layering (loader composes domain
  authorities). P1-04's reference pack names "compiled assembler result" as its
  expected interface, and `internal/artifacts/**` is in P1-04's allowed paths —
  the plan itself points here.
- *Rejected — `internal/contract.Assemble(bundle)`:* would add an
  `internal/artifacts` (and transitively `intentevidence`) dependency to the
  contract authority package — the exact direction P1-02's report already
  rejected ("Deriving facts inside the compiler would … add an `intentevidence`
  dependency to `contract`"). Contract is a consumed domain, not a consumer of
  Git loaders.
- *Rejected — new package (e.g. `internal/executionauthority`):* a new
  abstraction with a single consumer; AGENTS.md §11 forbids it; DL-18 requires
  extending, not duplicating.
- *Rejected — assembling directly in `cmd/worker/v14.go`:* that file is a
  shared hotspot (§10) and worker wiring is P1-04's owned task; the assembler
  must exist and be testable before the worker consumes it.
- *Rejected — inventing a second compile path or widening `contract.Bind`:*
  the existing authority already covers matching, assurance selection, evidence
  compilation, requires context and identity hashing (guard:
  `test/arch/v14_contract_authority_test.go` AC4a–e). The assembler only
  feeds it; it reimplements nothing.

## 3. Fail-safe rulings (the part P1-01/P1-02 explicitly deferred here)

| Trigger | Ruling | Basis |
|---|---|---|
| `Policies.Rejected` non-empty | **block** (`ErrPolicyRejections`) | Task implementation requirement 2 ("Policy loader rejections are fail-safe"). DL-09/BI-01: runtime-active policy is the exact policy content of the pinned snapshot; an artifact that exists but failed validation means the runtime-visible rule set is incomplete — running would be a silent policy false-negative (same ruling as the match layer: invalid policies are never silently skipped). |
| `DeriveRisk` error | **block**, typed passthrough (`ErrInvalidEvidence` / `risk.ErrMalformedSignal`) | P1-02 delegation: "Downstream (P1-03) must feed DeriveRisk's typed errors into execution blocking ('invalid risk basis stop execution')". Unreachable via the loader (`Parse` rejects invalid intents) — it is the assembler's defense against hand-built bundles. |
| Source-commit mismatch | **block** (`ErrSourceMismatch`) | Reference pack failure handling ("source mismatch … stop execution"); DL-04. |
| Unmatched / unknown risk signals | **do not block** | Existing pinned compiler semantics (INV-V14-09, `TestBindUnmatchedRiskFailsafeVisible`): signals stay visible via `Coverage()`, every clause is fail-safe uplifted to L3 (`ReasonUnmatchedRiskFailsafe`). The assembler must not change the authority it consumes. |
| Zero policies | **do not block** | P1-03's second deferred ruling (P1-01 report: "零策略或含拒收策略的 bundle 可装载成功，是否阻断执行归 P1-03 的 fail-safe 裁决"). The task contract's fail-safe requirement covers *rejections*, not an empty set; a policy-less project with no risk signals compiles to the L1 default floor, and with risk signals falls into the unmatched → L3 failsafe. Blocking zero-policy repositories is required by no authority document (PRD §16/§17, DL-09, locks); inventing that rule would be scope broadening. Test-pinned so the ruling is visible. |
| `contract.Compile`/`Bind` errors | **block**, typed passthrough | The contract authority's own gates (identity, pin, unpinned-policy, match-input rejections) are inherited, not duplicated. |

## 4. Exact source/tests read (L3 evidence)

Local authority first, then reference pack:

- Task contract + reference pack; `DECISION_LOCKS.md` (DL-04/05/09/18, BI-01),
  `PRD_AMENDMENTS.md` (A-02/A-04), `SOURCE_BASELINE.md` (gap 1:
  "`contract.Bind()` has no production worker call site"), `PATH_OWNERSHIP.md`
  (`internal/contract/**` owner = P1 lane), PRD §16 (Contract Compiler
  Authority), PRD §17.
- `internal/contract/compile.go`, `contract.go` (Compile/Open/Bind, pins,
  sentinels); `internal/contract/compile_test.go` — all reference-listed tests
  read in full: `TestBindPinsSnapshotAndIsDeterministic`,
  `TestBindCompilesRequiresDoubleDuty`, plus the seven siblings;
  `test/arch/v14_contract_authority_test.go` (AC4a–e guard, in place).
- `internal/artifacts/v141_bundle_loader.go`, `v141_intent_loader.go`,
  `v14_policy_loader.go`, `v13_loader.go` (types, rejection discipline,
  `PolicySnapshot.Rejected`); `v141_bundle_loader_test.go` + fixture files
  (`v13_loader_test.go`, `v141_freeze_commit_test.go`,
  `v141_intent_loader_test.go`, `v14_policy_loader_test.go`).
- `internal/intentevidence/riskderive.go`, `intentevidence.go`
  (`RiskSignalCandidates`, basis format); `internal/risk/risk.go`/`derive.go`
  (frozen enum, `Result`); `internal/engineeringpolicy/match.go` (coverage,
  confidence, fail-safe visibility); `internal/assurance/select.go` reasons;
  `internal/intake` task-type vocabulary; `internal/intakefreeze` (production
  freeze path used by tests).
- External references used: **none** (the pack says none required; current
  Tally source is the canonical reference).

## 5. Semantic changes / changed files

Changed files (all inside Allowed Paths `internal/artifacts/**` + task/report
docs; `internal/contract/**` and `internal/engineeringpolicy/**` zero-changed —
their authority is consumed, not modified):

- `internal/artifacts/v141_contract_assembler.go` (new): the seam above.
- `internal/artifacts/v141_contract_assembler_test.go` (new): 7 test functions.
- `docs/tasks/v1.4.1/tasks/P1/P1-03_…md` (appendix: Execution status / DONE).
- `docs/tasks/v1.4.1/reports/P1-03_…md` (this file).
- `docs/tasks/v1.4.1/STATUS.md` (16/57 → 17/57 + P1-03 summary).
- `docs/architecture/CURRENT_CODEBASE.md` (gap row updated: Bind now has a
  production seam; worker call-site still open → P1-04).
- `docs/implementation/CURRENT_IMPLEMENTATION_STATUS.md` (P1-03 entry).

Semantic changes: exactly one new public entry point
(`ExecutionBundle.CompileAuthority`) plus the `CompiledAuthority` result type
and two typed sentinels (`ErrPolicyRejections`, `ErrSourceMismatch`). No
existing behavior changed; `contract`/`engineeringpolicy`/`intentevidence`/
`risk` are byte-identical.

## 6. Tests/commands executed (exact results)

- `go build ./...` — PASS
- `go vet ./internal/artifacts/` — PASS
- `go test ./internal/artifacts/ -run 'TestCompileAuthority' -v` — **7/7 PASS**
  (TestCompileAuthorityPolicyMatch, TestCompileAuthorityNoMatchKeepsFailSafe,
  TestCompileAuthorityUnknownRiskVisible, TestCompileAuthorityRejectsInvalidPolicy,
  TestCompileAuthorityRejectsInvalidRiskBasis,
  TestCompileAuthorityRejectsSourceMismatch,
  TestCompileAuthorityRejectsMixedIdentity)
- `go test ./internal/artifacts/... ./internal/contract/... ./internal/engineeringpolicy/... ./internal/intentevidence/... ./internal/risk/... ./internal/assurance/...` — all `ok` (PASS)
- `go test ./test/arch/...` — PASS (DL-04/DL-18 contract-authority guard intact)
- `go test ./internal/...` — PASS (all packages, no failures)
- `gofmt -l` on both new files — clean
- `git diff --check` — clean
- `make check` — NOT_RUN (no make binary in Git Bash; constituent families run
  individually above, same as P0/P1 predecessor tasks)
- integration e2e (`go test -tags=integration ./test/e2e/...`) — NOT_RUN
  (opt-in DB-gated suite; the task's "Integration tests" verification is the
  policy match/no-match/invalid/unknown matrix, which runs against real
  temporary Git repositories through the real production freeze path
  (`intakefreeze.Freeze` + `FreezeCommit`) — commit-tree fixtures, no DB
  involvement)

All tests were executed; no PASS is claimed for any unrun command.

## 7. Negative/failure-path tests added (all PASS)

- `TestCompileAuthorityRejectsInvalidPolicy` — one valid + one invalid policy in
  the pinned commit: bundle still loads (P1-01 boundary), assembly fails with
  `ErrPolicyRejections`, no `Compiled` produced, error names the rejected file,
  not misclassified as `ErrSourceMismatch`.
- `TestCompileAuthorityRejectsInvalidRiskBasis` — zero-value intent (hand-built
  bundle) → `intentevidence.ErrInvalidEvidence` passthrough; nil intent → typed
  error, no panic.
- `TestCompileAuthorityRejectsSourceMismatch` — hand-mutated policy snapshot
  commit → `ErrSourceMismatch`.
- `TestCompileAuthorityRejectsMixedIdentity` — two tasks frozen in one repo,
  hand-assembled mixed-identity bundle → `contract.ErrTaskMismatch` passthrough
  (the contract authority's identity gate is inherited, not bypassed).
- No-match/unmatched-risk and unknown-risk sub-scenarios double as the
  "fail-safe intact" proofs: compilation succeeds but every clause is L3 with
  `ReasonUnmatchedRiskFailsafe` and signals stay visible in `Coverage()` — the
  assembler never downgrades or hides them.

## 8. Baseline failures observed

None. All suites pass at HEAD before and after the change (the only touched
packages are `internal/artifacts` plus docs).

## 9. Acceptance criteria

- **AC1 — No supported task starts execution without Compiled contract: PASS
  (seam-level).** `CompileAuthority` is the only path from a loaded bundle to
  the execution authority; every fail-safe trigger (rejected policy, invalid
  risk basis, source mismatch, mixed identity, any contract/compile rejection)
  produces an error and no `Compiled`. Worker call-site timing — obtain the
  compiled authority *before* creating a workspace/runtime session, and record
  its identity in run facts — is P1-04's owned wiring (`cmd/worker/v14.go` is a
  §10 hotspot this task must not touch); the spy-based proof ("no session starts
  before successful compilation") is P1-04's AC.
- **AC2 — Inputs are reproducible from source commit: PASS.** The assembler is
  a pure function of the bundle; test-pinned: same-bundle re-assembly → same
  compiled hash; `LoadBundle` at the same commit → re-assembly → same compiled
  hash and same bundle identity; policy-content change at a later commit →
  compiled hash changes (input sensitivity); pins byte-equal the bundle's
  contracts (the compiler rewrites nothing — AC4d discipline).

## 10. Production reachability

**N/A(owned integration task pending):** the assembler is production code in
`internal/artifacts`, but no composition root calls it yet by design — the
worker call site belongs to P1-04 (`cmd/worker/v14.go` is a shared hotspot;
P1-04's goal is exactly "obtain CompiledContract before creating an execution
workspace/runtime session"). PRD §16's "worker does not bypass the compiler"
becomes a runtime fact when P1-04 lands; from P1-03's seam the bypass is
already structurally closed (the only bundle → compiled-contract entry refuses
to produce a `Compiled` on any fail-safe trigger).

## 11. Completion report (per task contract format)

1. **status:** DONE
2. **changed files:** see §5
3. **semantic changes:** new production compilation assembler seam
   (`ExecutionBundle.CompileAuthority` → `CompiledAuthority`); fail-safe
   rulings: rejected policies block, invalid risk basis blocks (typed
   passthrough), source mismatch blocks; unmatched/unknown risk keeps the
   existing visible + L3 failsafe semantics; zero-policy bundles compile
   (ruling documented and test-pinned); no existing behavior changed
4. **tests/commands:** see §6 (all executed as listed; focused 7/7, arch,
   neighbors, full internal suite PASS)
5. **baseline failures:** none
6. **remaining risks:** see §12
7. **Decision Lock touched:** none (DL-04/05/09/18 satisfied by reuse;
   `DECISION_LOCKS.md` file untouched; hidden/ unread, DL-17)
8. **commit:** recorded in the task-file addendum / final message

## 12. Remaining risks / notes

- The zero-policy ruling (compile, not block) is the one judgment call this
  task owned. It is documented in code, pinned by
  `TestCompileAuthorityNoMatchKeepsFailSafe`, and justified against the
  authority chain; if product later wants policy-less repos blocked, that is a
  one-line change at this seam with its test flipped — not a structural one.
- P1-04 must call `CompileAuthority` before workspace/runtime creation and
  record `BundleIdentity`/`Compiled.Hash()`/risk basis in run facts; until
  then PRD §16's no-bypass property is seam-level, not composition-level.
- `risk.Result` is carried on `CompiledAuthority` as a derived value (DL-05);
  downstream tasks (P1-06 acceptance guard, P1-07 projections) should consume
  it from here or re-derive from the pinned commit — never persist it as
  authoritative.
- HIGH review per MODEL_ASSIGNMENT still pending; suggested review focus: the
  fail-safe rulings table (§3) — especially the zero-policy ruling — and the
  chosen seam direction (artifacts → contract, §2).
- Known source-fact note: SOURCE_BASELINE gap 1 ("`contract.Bind()` has no
  production worker call site") is now half-closed: the production seam exists
  and `Bind` is reachable through it; the call site itself is P1-04.
