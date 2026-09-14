# P1-02 — Authoritative risk input derivation: Execution Report

- **TASK:** P1-02
- **STATUS:** DONE
- **Date:** 2026-09-12
- **Phase / Wave:** P1 / W09
- **Execution/Review model:** HIGH / HIGH (per task pack; HIGH review pending — the pack requires it because the task defines an authority-boundary derivation chain)
- **Depends on:** P1-01 (DONE) — satisfied: the bundle's FrozenIntent is this task's upstream fact source
- **Hidden material:** none read; `docs/tasks/v1.4.1/hidden/` untouched (DL-17)
- **Authority chain read in order:** task contract → `DECISION_LOCKS.md` (DL-05 primary, DL-06, DL-18) → `PRD_AMENDMENTS.md` (A-02) → `SOURCE_BASELINE.md` → `PATH_OWNERSHIP.md` → current local source → L3 reference pack

## 1. Deliverable

`intentevidence.FrozenIntent.DeriveRisk()` in the new file
`internal/intentevidence/riskderive.go`: a pure, deterministic derivation of the
authoritative `risk.Result` from **the frozen intent evidence only**:

- confirmed risk hints project through the existing
  `FrozenIntent.RiskSignalCandidates()` (basis `"intent:<task-id>"`, the P0-01
  adjudicated format) — candidate LLM risk output (`intake.Draft.RiskSignals`,
  reasoning prose) has **no parameter path** into the derivation; only
  owner-confirmed facts frozen into evidence are read (implementation
  requirement 1, DL-06, A-02);
- the frozen task-type fact projects through the new
  `FrozenIntent.TaskTypeRiskCandidates()`: `db_migration` (PRD §24.2 closed
  loop) deterministically implies `schema_migration` (PRD §13.1/§24.2) with
  basis `"tasktype:<task-id>"`; `bug_fix` implies no frozen signal and yields
  zero candidates. The `tasktype:` kind is deliberately distinct from
  `intent:` so owner-confirmed declarations stay distinguishable from
  deterministic derivations; when both sources produce the same signal
  (db_migration task that also confirmed the `schema_migration` hint),
  `risk.Derive` merges them into one signal with the sorted deduplicated
  union of both basis refs — both provenances preserved;
- the composed candidates go **verbatim** into the existing `risk.Derive`:
  known/unknown classification stays entirely in the risk package (DL-05,
  extend-not-duplicate — this file adds no classification semantics), unknown
  hints return in `Result.Unknown` verbatim with basis (INV-V14-09: never
  dropped, renamed or read as low risk), and `HasUnknown`/`AboveMinimal` keep
  the fail-safe floor semantics for the downstream unmatched-risk → L3
  uplift (`contract.Bind` `riskBand`, guarded by
  `test/arch/v14_assurance_failsafe_test.go`).

Typed rejections (`errors.Is`-judgeable):

- zero-value receiver → `ErrInvalidEvidence`: a zero `FrozenIntent` is not
  evidence, and deriving "no risk" from non-evidence would be a silent
  under-classification;
- any structurally invalid candidate (missing/blank basis, blank/padded id)
  → `risk.ErrMalformedSignal`, failing the **whole** derivation — the risk
  invariants are preserved unchanged at the same `risk.Derive` entry the
  derivation delegates to; there is no partial output.

Nothing is persisted: no RiskProfile, no storage, no state (implementation
requirement 3, DL-05). The result is recomputable at any time, field-identical,
from the evidence reloadable off the pinned commit (same Revision ⇒ same
`Result`).

## 2. Why this is the narrowest valid seam (and rejected alternatives)

- **`internal/intentevidence` (chosen):** it owns the confirmed-facts domain,
  already holds the P0-01-adjudicated `intentevidence → risk` import edge, and
  "which candidates enter the derivation" is a ruling about the facts — its
  owner's side. Extends the existing domain (DL-18); `internal/risk` and
  `internal/contract` are zero-changed.
- *Rejected — `internal/risk` (e.g. `risk.FromIntent`):* the risk package is
  deliberately stdlib-only with basis refs "owned by callers" (its package
  doc); an intent-aware derivation would add an internal dependency and put a
  fact-domain ruling inside the classification package.
- *Rejected — `internal/artifacts` (method on `ExecutionBundle`):* the bundle
  loader's owned seam stopped at same-source loading (P1-01); derivation is a
  pure domain transform over the `FrozenIntent` value and needs no Git access.
  Keeping it off the bundle lets P1-03 compose `LoadBundle` + `DeriveRisk` +
  `Bind` without the loader growing derivation authority.
- *Rejected — `internal/contract`:* the compiler consumes `risk.Result`
  (`Bind`'s `riskFacts`); its own doc comment defers "task-type signal
  collection … through the same entry" (risk.Derive) to later tasks. Deriving
  facts inside the compiler would blur the DL-04 authority inputs with domain
  derivation and add an `intentevidence` dependency to `contract`.
- *Rejected — variadic `...risk.Signal` parameter on `DeriveRisk`:* it would
  open a bypass channel for unconfirmed (LLM) candidates, violating
  requirement 1's spirit; the public surface stays two narrow methods.
- *Change-path / repository-scan facts:* deliberately not derived here —
  change paths do not exist before execution (INV-V14-02); candidate-change
  signal reuse is P1-05's matching surface. Compile-time facts are exactly the
  frozen 4-field schema.

Fail-safe intactness argument: the derivation interprets nothing — it only
composes candidate projections (each with mandatory basis by construction) and
delegates to `risk.Derive`, whose fail-safe behavior (unknown visible,
`AboveMinimal`, malformed-signal total rejection) is already pinned by
`internal/risk/risk_test.go` and re-pinned at this seam by the new negative
tests. Downstream interpretation (unmatched → L3) is unchanged code guarded by
`test/arch/v14_assurance_failsafe_test.go`.

## 3. Exact source/tests read (L3 evidence)

- Tests first (per reference order): `internal/risk/risk_test.go`,
  `internal/engineeringpolicy/match_test.go`,
  `internal/contract/compile_test.go` (fixtures + `Bind` determinism/pin
  coverage), `test/arch/v14_assurance_failsafe_test.go` (AC5a–c guards).
- Interfaces: `risk.Signal`/`risk.Result`/`Derive` (basis contract),
  `engineeringpolicy.MatchInput`/`PolicyMatchCoverage`,
  `contract.Bind(snap, policies, riskFacts risk.Result)`,
  `intentevidence.FrozenIntent` facts + `RiskSignalCandidates`.
- Core: `internal/risk/derive.go`, `internal/risk/risk.go`,
  `internal/engineeringpolicy/match.go` (consumes the derived `risk.Result`),
  `internal/contract/compile.go` (`riskBand`: unmatched → fail-safe band;
  `AboveMinimal` → elevated), `internal/intentevidence/intentevidence.go` +
  `artifact.go`, `internal/intakefreeze/intakefreeze.go` (confirmed hints vs
  draft candidates), `internal/intake/intake.go` (candidate `RiskSignals`
  provenance), `internal/taskcontract/types.go` (no risk facts on the task
  contract — checked before rejecting it as a fact source),
  `internal/artifacts/v141_intent_loader.go` + `v141_bundle_loader.go`
  (upstream bundle shape, P1-01 report's downstream note).
- Prior task evidence: `reports/P1-01_same_source_execution_bundle_loader.md`
  (names P1-02 as the downstream consumer), `reports/P0-01_…md` (basis-format
  adjudication `intent:<task-id>`).
- External reference used: none (pack: current Tally source is canonical).

## 4. Semantic changes / changed files

- `internal/intentevidence/riskderive.go` (new): `TaskTypeRiskCandidates`,
  `DeriveRisk`, `taskTypeBasisKind` constant; package doc boundary statement.
- `internal/intentevidence/riskderive_test.go` (new): derivation table tests,
  determinism, purity, and negative-path tests.
- `internal/intentevidence/intentevidence.go`: package-doc bullet extended —
  the derivation entry lives in this package but classification remains
  entirely in `risk.Derive` (documentation of a current fact, §15; no code
  change).
- `internal/risk`, `internal/contract`, `internal/engineeringpolicy`,
  `internal/artifacts`: zero-changed (DL-18; consumers consume).
- Decision Locks touched: none. DL-05 is the implemented lock (derived risk,
  no persisted RiskProfile); DL-06 inherited (only frozen confirmed facts are
  read); DL-18 respected (existing domains extended, no parallel risk system);
  DL-17 respected.

## 5. Tests/commands executed (exact results)

Windows / Git Bash, clean tree before this change. Changes are 1 new production
file + 1 new test file + 1 doc-only comment line, so the package baseline equals
current-minus-new-code; no baseline failure existed to mask.

| # | Command | Result |
|---|---|---|
| 1 | `gofmt -l internal/intentevidence/` | PASS — no output |
| 2 | `go test ./internal/intentevidence/ ./internal/risk/` | PASS — both `ok` |
| 3 | `go test -run 'TestDeriveRisk\|TestTaskTypeRiskCandidates' -v ./internal/intentevidence/` | PASS — 6/6 (the task Verification: risk derivation table tests, §6) |
| 4 | `go test ./internal/risk/ ./internal/engineeringpolicy/ ./internal/contract/ ./test/arch/` | PASS — 4/4 `ok` (reference-pack listed test families, incl. `v14_assurance_failsafe_test.go`) |
| 5 | `go build ./...` | PASS |
| 6 | `go vet ./...` | PASS |
| 7 | `go test ./internal/...` | PASS — 46 packages `ok`, 0 failures (2 no-test-file pkgs) |
| 8 | `git diff --check` | PASS — clean |

NOT_RUN: `make check` (no make binary in Git Bash; constituent families run
individually per prior-task precedent); `-tags=integration` e2e suite (DB-gated
opt-in; this task's Verification section is table tests over a pure transform,
no integration requirement).

## 6. Negative/failure-path tests added (all PASS)

`internal/intentevidence/riskderive_test.go`:

- `TestDeriveRiskKnownFactsProduceDeterministicSignals` — the AC table: bug_fix
  + no hints ⇒ empty set (legitimate, not error); db_migration ⇒ deterministic
  `schema_migration` with `tasktype:` basis (above-minimal floor); confirmed
  auth/protected hints ⇒ signals with `intent:` basis; hint+task-type same
  signal ⇒ merged sorted union basis; all three signals together; unknown hint
  ⇒ `Unknown` verbatim, `HasUnknown`/`AboveMinimal` true (fail-safe visible);
- `TestDeriveRiskDeterministicAcrossEqualEvidence` — equal-revision intents
  (shuffled/typographic variants) derive field-identical `Result`s, repeatedly;
- `TestDeriveRiskIsPure` — caller mutation of the result (and of candidate
  projection slices) cannot leak into evidence or re-derivation;
- `TestDeriveRiskRejectsZeroValue` — **negative**: zero-value intent →
  `ErrInvalidEvidence`, no partial result;
- `TestDeriveRiskChainRejectsMissingBasis` — **negative (AC: missing basis is
  rejected per risk invariants)**: a basisless candidate injected into the
  composed derivation input fails the whole derivation with
  `risk.ErrMalformedSignal` (no partial result), and the positive round-trip:
  `DeriveRisk` output re-fed into `risk.Derive` re-derives identically (every
  produced signal carries basis);
- `TestTaskTypeRiskCandidatesBoundaries` — bug_fix/zero ⇒ nil; db_migration ⇒
  the exact single candidate.

## 7. Baseline failures observed

None. All commands in §5 passed; no pre-existing failure encountered or masked.

## 8. Acceptance criteria

- [x] **Known migration/auth/protected-surface facts produce deterministic
  signals+basis** — `TestDeriveRiskKnownFactsProduceDeterministicSignals`
  (signals + verbatim basis from confirmed hints and the deterministic
  db_migration task-type fact; merged union basis proves provenance
  preservation) + `TestDeriveRiskDeterministicAcrossEqualEvidence`.
- [x] **Missing basis is rejected per risk invariants** —
  `TestDeriveRiskChainRejectsMissingBasis` (typed `ErrMalformedSignal`, whole
  derivation, no partial output) + `TestDeriveRiskRejectsZeroValue`
  (non-evidence never derives "no risk").

Implementation requirements: (1) LLM candidates enter only via frozen evidence
— no parameter path exists; (2) unknown/unmatched stay visible — `Unknown`
verbatim + `HasUnknown`/`AboveMinimal`, consumed unchanged by the guarded
fail-safe uplift; (3) no persisted mutable RiskProfile — pure function, zero
storage/state added.

## 9. Production reachability

**N/A(owned integration tasks pending):** the derivation is production code in
`internal/intentevidence`, but no composition root calls it yet by design —
P1-03 owns assembling `LoadBundle` + `DeriveRisk` + `contract.Bind` (its
failure handling: "invalid risk basis stop execution" maps to the typed
`risk.ErrMalformedSignal` / `ErrInvalidEvidence` rejections), and P1-04 wires
the worker. No handler/worker file touched (`cmd/worker/v14.go`,
`internal/web/handlers/server.go` are shared hotspots).

## 10. Completion report (per task contract format)

1. **status:** DONE
2. **changed files:** `internal/intentevidence/riskderive.go` (new),
   `internal/intentevidence/riskderive_test.go` (new),
   `internal/intentevidence/intentevidence.go` (package-doc bullet only),
   `docs/tasks/v1.4.1/tasks/P1/P1-02_…md` (appendix),
   `docs/tasks/v1.4.1/reports/P1-02_…md` (new),
   `docs/tasks/v1.4.1/STATUS.md` (15/57 → 16/57),
   `docs/architecture/CURRENT_CODEBASE.md`,
   `docs/implementation/CURRENT_IMPLEMENTATION_STATUS.md`
3. **semantic changes:** new authoritative risk-input derivation on
   FrozenIntent (confirmed hints + deterministic task-type fact → `risk.Derive`
   → `risk.Result`); typed zero-value rejection; no existing behavior changed
4. **tests/commands:** see §5 (all PASS; `make check` and integration e2e
   NOT_RUN with reasons)
5. **baseline failures:** none
6. **remaining risks:** see §11
7. **Decision Lock touched:** none
8. **commit:** recorded in the task-file addendum / final message

## 11. Remaining risks / notes

- The only deterministic task-type mapping is `db_migration → schema_migration`;
  `bug_fix` intentionally yields none. Extending the mapping (or the signal
  enum) is a task-contract-level change, not runtime data — same discipline as
  the frozen signal enum.
- Basis kinds are opaque strings by design; the `tasktype:` vs `intent:`
  distinction is documentation + test-pinned, not compiler-enforced.
- Downstream (P1-03) must feed `DeriveRisk`'s typed errors into execution
  blocking ("invalid risk basis stop execution"); the derivation itself only
  returns them.
- HIGH review per MODEL_ASSIGNMENT still pending; review focus: the seam choice
  (§2 alternatives) and the zero-value rejection semantics.
- Known source-fact note: `internal/contract/compile.go`'s doc comment already
  anticipated this task ("任务类型/历史需求的信号采集属后续任务,经同一入口接入")
  — the task-type half is now connected via the same `risk.Derive` entry;
  "historical requirement" facts remain future tasks (P3 replay lane), and
  this task deliberately did not pre-build an input for them.
