# P0-07 — Verification authoring and freeze admission: Execution Report

- **TASK:** P0-07
- **STATUS:** DONE
- **Date:** 2026-09-12
- **Phase / Wave:** P0 / W05
- **Execution/Review model:** HIGH / HIGH (per task pack; HIGH review still required on this report)
- **Depends on:** P0-03, P0-06 — both satisfied
- **Hidden material:** none read; `docs/tasks/v1.4.1/hidden/` untouched (DL-17)
- **Authority chain read in order:** task contract → `DECISION_LOCKS.md` (DL-05, DL-06, DL-17, DL-18) → `PRD_AMENDMENTS.md` → `SOURCE_BASELINE.md` → `PATH_OWNERSHIP.md` → current local source → L3 reference pack (reference order honored: tests → interfaces → core implementation)

## 1. Deliverable

Freeze admission for the Confirm→Freeze chain, added to the same package P0-06
created (its report explicitly reserved this lane):

```go
func Freeze(confirmed ConfirmedDraft, plan VerificationPlan) (FrozenArtifacts, error)
type EvidenceDeclaration struct { AcceptanceID string; Kind verification.EvidenceType; Path, Selector string; HeldOut bool; Store, Ref string }
type VerificationPlan struct { Evidence []EvidenceDeclaration; RuntimeRequirements *verification.RuntimeRequirements }
type FrozenArtifacts struct { Intent intentevidence.FrozenIntent; Task taskcontract.TaskContract; Verification verification.Frozen }
```

- `Freeze` **reuses `Map` unchanged** (all P0-06 fail-safe gates fire first),
  then authors a `verification.Contract` whose acceptance set is 1:1 with the
  confirmed draft clauses, and freezes it through the **existing authority**
  `verification.FreezeContract` with the zero-value `FreezePolicy` (P1
  standard path). No second freeze implementation; `internal/verification`
  itself is zero-changed (read-only consumption, DL-18).
- `FrozenArtifacts` is the complete freeze-transaction content (FrozenIntent +
  TaskContract + VerificationContract, same task identity) that P0-08 commits
  atomically; this package still performs no Git writes/storage access.

## 2. Admission semantics (fail-safe, deterministic order)

Fixed order; every gate rejects with a typed sentinel and zero artifacts, so no
partial frozen state can escape:

1. **`Map` gates** (profile → task type → unrepresentable draft semantics →
   `intentevidence.Freeze` → `taskcontract.Validate`/`ValidateStatic`).
2. **Invention guard** — every declaration must route to a real confirmed
   clause (layout-trim on the routing key only): orphan declarations are
   invented bindings → `ErrUnknownEvidenceTarget`. Confirm→Freeze cannot
   invent evidence bindings.
3. **Duplicate guard** — equal declarations (content key, routing key folded)
   are rejected (`ErrInvalidVerification`), never deduplicated: declared
   content must remain traceable in the frozen output.
4. **Omission guard** — every acceptance clause needs ≥1 declaration; missing
   ones are listed ascending by clause id → `ErrMissingEvidence` (the typed
   clarification/blocker of implementation requirement 2; missing evidence
   prevents execution-ready freeze). `verification`'s own freeze rule (≥1
   binding per acceptance) remains as the independent backstop.
5. **Held-out isolation gate** — held-out is *only* the explicit
   `EvidenceDeclaration.HeldOut` flag (never inferred — the mapper has no rule
   that could auto-label anything held-out); any held-out declaration requires
   complete declared runtime isolation requirements (INV-VER-05) or →
   `ErrHeldOutWithoutIsolation`. The reverse direction is enforced by the
   authority: visible bindings referencing the holdout store are rejected
   (§8.4.1).
6. **Authority validation + freeze** — `verification.FreezeContract(c,
   verification.FreezePolicy{})`; failures wrap through `ErrInvalidVerification`
   (unknown evidence kind, held-out carrying a repo path, empty acceptance
   set, …).

### Authoring rules (mapping table)

| Input | verification.Contract | Rule |
|---|---|---|
| draft clause (post-Map) | `Acceptance{ID, Statement}` | 1:1, verbatim; set equality is structural (built from the mapped task), test-pinned |
| `AcceptanceID` | routing only | must hit a clause; trim-folded as layout |
| — | `Binding.ID = <clause>-EV<nn>` | mechanically derived (clause ids are unique `AC-<digits>`), cannot mismatch its clause |
| — | `AuthoringMode = pre_commit` | the only mode authorable at Confirm→Freeze; `post_hoc` is a runtime governance escalation (PRD §8.2) and is **not expressible** in the declaration shape |
| `HeldOut=true` | `VisibilityPolicy = held_out` | explicit declaration only; default is the safe direction (`visible`) |
| `Kind` | `Evidence.Type` | direct reuse of the closed `verification.EvidenceType` vocabulary; unknown → authority rejection |
| `Path`/`Selector`, `Store`/`Ref` | `EvidenceRef` fields | verbatim; held-out-with-path / missing store-ref rejected by §8.4.1 rules |
| `RuntimeRequirements` | `Contract.RuntimeRequirements` | required-complete iff any held-out declared; zero otherwise |
| — | `TaskID` = mapped task id, `Revision = 1` | shared identity; first freeze is revision 1 (re-confirmation = new freeze) |
| — | `IndependenceRequired` undeclared | conservative `none`; the confirm input has no independence channel and admission must not invent governance attributes (intake boundary; P0-06 report §2 pinned the same default) |

Determinism: clauses sorted by id, declarations within a clause by content key
(duplicates already rejected ⇒ total order) **before** binding-id assignment;
frozen hashes are invariant under input permutation (test-pinned).

## 3. Why this is the narrowest valid seam (and rejected alternatives)

- **Same package, new file (`internal/intakefreeze/verification.go`)** — an
  Allowed Path of this task, explicitly reserved by P0-06's report ("P0-07
  verification authoring/freeze admission in the same package"); reusing `Map`
  keeps a single draft→frozen mapping path, so no semantics can fork between
  the mapper and the admission entry.
- *Rejected:* extending `Map`/`ConfirmedDraft`/`Frozen` in place — would
  rewrite P0-06's delivered contract and its passing test suite; the two-entry
  split (mapper vs. execution-ready freeze admission) is the documented
  program plan and is what P0-08's reference pack expects ("new mapper
  outputs", plural).
- *Rejected:* putting admission inside `internal/verification` — that package
  is the contract domain (what a contract is / how it freezes), not the
  confirm-flow admission boundary; it must not depend on intake-side shapes.
- *Rejected:* a new `verificationauthoring` package — DL-18 anti-duplication
  and the narrow-surface rule; the same-package extension was already reserved.
- Fail-safe intact: zero-value `FreezePolicy` (post-hoc can never pass from
  this path), all gates pre-freeze, typed sentinels only, authority errors
  wrapped with `%w` (errors.Is transparent). No acceptance/assurance strength
  lowered anywhere; the new path can only *add* rejections.

## 4. Exact source/tests read (L3 evidence)

- Task authority: this task file, `DECISION_LOCKS.md`, `PRD_AMENDMENTS.md`,
  `SOURCE_BASELINE.md`, `PATH_OWNERSHIP.md`, `STATUS.md`, `TASK_DAG.md`
  (P0-07/P0-08 rows), adjacent contracts P0-06/P0-08 (output split: P0-07 =
  verification admission; P0-08 = Git transaction behind `V14ConfirmDraft`).
- Tests first (reference order): `internal/verification/freeze_test.go`,
  `internal/verification/holdout_path_leak_test.go`,
  `internal/intakefreeze/intakefreeze_test.go` (P0-06 inventory),
  `internal/taskcontract/validate_test.go` (inventory), baseline run of
  `internal/verification`, `internal/intakefreeze`, `internal/intake`,
  `internal/contract` (all `ok` before any change).
- Interfaces/types: `verification.FreezePolicy` / `ValidateFreeze` /
  `FreezeContract` / `Contract` / `Acceptance` / `Binding` / `EvidenceRef` /
  `RuntimeRequirements` / `Hash` (canonicalizing), `taskcontract.AcceptanceClause`
  / `ValidateStatic` (AC-<digits> grammar), `intentevidence.FrozenIntent`.
- Core implementation: `internal/verification/{freeze,types,hash}.go`,
  `internal/intakefreeze/intakefreeze.go`, `internal/taskcontract/validate.go`,
  `internal/contract/compile.go` (downstream consumer of the frozen contract,
  read-only), `internal/requirement/lint.go` (unresolved-marker linter idiom,
  boundary reference only), `internal/web/handlers/v14_launch.go`
  (`V14ConfirmDraft` / `v14DraftFromForm`, read-only: confirm path enforces
  ≥1 acceptance clause and currently carries no evidence declarations — its
  extension is P0-08's lane).
- External reference used: none (pack: current Tally source is canonical).

## 5. Semantic changes

- New file `internal/intakefreeze/verification.go` (+ test file): the freeze
  admission entry `Freeze`, declaration types, four typed sentinels,
  `FrozenArtifacts`.
- `internal/intakefreeze/intakefreeze.go`: doc-comment-only update (package doc
  now names the P0-07 admission entry; `Map` doc points to `Freeze`). No
  behavioral change to `Map`; all 11 P0-06 tests pass unchanged.
- No changes to `internal/verification`, `internal/intake`,
  `internal/taskcontract`, `internal/intentevidence`, handlers, worker,
  store, migrations, templates. No new dependency.

## 6. Tests/commands executed (exact results)

Windows / Git Bash, Go `go1.27.1 windows/amd64`, clean tree before this change.

| # | Command | Result |
|---|---|---|
| 1 | `go test ./internal/verification/ ./internal/intakefreeze/ ./internal/intake/ ./internal/contract/` (baseline, before change) | PASS — all `ok` |
| 2 | `go test -count=1 -v ./internal/intakefreeze/` | PASS — 19/19 test functions (11 pre-existing P0-06 + 8 new) |
| 3 | `gofmt -l internal/intakefreeze/` (after fixing one formatting nit and one test-field typo found by vet) | PASS — no output |
| 4 | `go vet ./internal/intakefreeze/`, then `go vet ./...` | PASS |
| 5 | `go test -count=1 ./internal/intakefreeze/ ./internal/intake/ ./internal/intentevidence/ ./internal/taskcontract/ ./internal/verification/ ./internal/contract/` | PASS — all `ok` |
| 6 | `go test ./test/arch/...` | PASS — `ok tally/test/arch` |
| 7 | `go build ./...` | PASS |
| 8 | `go test -count=1 ./internal/...` | PASS — 46 packages `ok`, 0 FAIL |
| 9 | `git diff --check` | PASS — clean |
| 10 | `make check` | NOT_RUN(reason: no `make` executable in this Git Bash environment, consistent with P0-04/P0-06; its constituent families were executed directly as #6/#8, integration below) |
| 11 | `go test -tags=integration ./test/e2e/...` | NOT_RUN(reason: opt-in, DB-gated; the task's Verification section requires unit tests only, covered by #2/#5) |

Task-contract Verification cases: normal (`TestFreezeHappyPathAuthorsVerification`,
incl. cross-check against a direct `verification.FreezeContract` of the authored
content, determinism and permutation invariance), missing-evidence
(`TestFreezeRejectsMissingEvidence`: single + empty plan listing all clauses
ascending, zero artifacts on failure), held-out
(`TestFreezeHeldOutExplicitDeclarationWithIsolationGate`: missing/incomplete
runtime requirements rejected, complete declaration freezes with store/ref-only
binding, and the never-auto-label proof), invalid binding
(`TestFreezeRejectsInvalidBindings`: unknown/missing evidence kind, visible
referencing holdout store, equal and routing-variant duplicates, held-out
carrying a repo path) plus negative extras: invented targets
(`TestFreezeRejectsUnknownEvidenceTarget`, incl. `ac-01`/`AC-1` variants and
layout-trim acceptance), clause-free light freeze
(`TestFreezeRejectsAcceptanceFreezeWithoutClauses`), Map-gate precedence
(`TestFreezeReusesMapGatesFirst`), input isolation
(`TestFreezeDoesNotMutateInputs`), and the structural pre-commit-only proof
inside the happy path (every frozen binding `AuthoringMode == pre_commit`).

## 7. Baseline failures observed

None. Baseline command (#1) passed before the change; no pre-existing failure
was encountered or masked.

## 8. Acceptance criteria

- [x] **Every frozen acceptance has a valid planned evidence binding** —
  admission requires ≥1 declaration per confirmed clause (`ErrMissingEvidence`
  otherwise), bindings are mapped onto the closed `verification` vocabulary and
  must pass `Contract.Validate` + freeze rules; the happy-path test asserts
  every frozen acceptance carries ≥1 valid binding and that the frozen hash
  equals a direct authority freeze of the same content.
- [x] **Missing evidence prevents execution-ready freeze** —
  `ErrMissingEvidence` (typed blocker naming the clauses) with zero artifacts;
  the `Map`-only mapper is documented as the non-execution-ready mapping hop.

## 9. Production reachability

**N/A(owned integration task pending):** this task owns the domain admission
per its Allowed Paths; the production wiring (handler confirm form carrying
evidence declarations, atomic Git commit of `FrozenArtifacts`) is P0-08's
owned lane (`v14_launch.go` is P0-04→P0-08 serial). No composition-root
behavior changed here.

## 10. Completion report (per task contract format)

1. **status:** DONE
2. **changed files:** `internal/intakefreeze/verification.go` (new),
   `internal/intakefreeze/verification_test.go` (new),
   `internal/intakefreeze/intakefreeze.go` (doc comments only),
   `docs/tasks/v1.4.1/tasks/P0/P0-07_verification_authoring_and_freeze_admission.md` (addendum),
   `docs/tasks/v1.4.1/reports/P0-07_verification_authoring_and_freeze_admission.md` (new),
   `docs/tasks/v1.4.1/STATUS.md` (11/57 → 12/57),
   `docs/architecture/CURRENT_CODEBASE.md`,
   `docs/implementation/CURRENT_IMPLEMENTATION_STATUS.md`
3. **semantic changes:** Confirm→Freeze now has a typed, fail-safe freeze
   admission that authors the frozen VerificationContract from owner-declared
   planned evidence — evidence cannot be invented (orphan declarations
   rejected), omitted (missing evidence blocks the freeze), or silently
   auto-labeled held-out (explicit declaration + isolation requirements only);
   authoring provenance is structurally pre_commit; no existing behavior
   changed
4. **tests/commands:** see §6 (all PASS; `make check` and integration e2e
   NOT_RUN with reasons)
5. **baseline failures:** none
6. **remaining risks:** see §11
7. **Decision Lock touched:** none
8. **commit:** recorded in the task-file addendum / final message

## 11. Remaining risks / notes

- **Independence authoring is deliberately not implemented:** the confirm
  input has no independence declaration channel, so admission freezes the
  conservative undeclared (`none`) default (same default P0-06 pinned for the
  TaskContract). Introducing source/information independence levels needs a
  confirmed-input channel first — a product decision for a later task, not a
  mapper-local tweak. `verification`'s planned-coverage rules stay intact and
  will apply the moment a contract declares a level.
- **Protection (protected evidence + baseline_hash) is not on the declaration
  surface:** its integrity anchor belongs to artifact bytes; admitting
  owner-typed hashes now would invite invented anchors. A later lane can add
  it without shape breakage (new optional declaration fields).
- The production confirm UI does not yet collect evidence declarations; until
  P0-08 lands, `Freeze` is reachable only from tests — by design (P0-04→P0-08
  serial ownership of `v14_launch.go`).
- HIGH review (per MODEL_ASSIGNMENT) has not yet executed; this report is the
  review input.
- Known sensitivity: binding ids derive from clause ids (`AC-01-EV01` style);
  clause-id grammar is pinned by `taskcontract.ValidateStatic`, so derivation
  cannot collide across clauses.

## 12. Decision Locks confirmation

No lock text edited and none bypassed: DL-05 (risk untouched), DL-06 (only
confirmed facts enter freeze — evidence enters only via explicit owner
declarations), DL-17 (no hidden material read), DL-18 (existing
verification/taskcontract/intentevidence domains consumed, not duplicated or
rewritten; `internal/verification` zero-changed). Held-out boundaries
preserved: explicit-declaration-only plus INV-VER-05 isolation gate; the
holdout path-leak suite in `internal/verification` remains green.
