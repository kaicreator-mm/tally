# P0-06 — Draft-to-frozen domain mapper: Execution Report

- **TASK:** P0-06
- **STATUS:** DONE
- **Date:** 2026-09-12
- **Phase / Wave:** P0 / W05
- **Execution/Review model:** LOW / HIGH (per task pack; HIGH review still required on this report)
- **Depends on:** P0-01, P0-03 — both satisfied
- **Hidden material:** none read; `docs/tasks/v1.4.1/hidden/` untouched (DL-17)
- **Authority chain read in order:** task contract → `DECISION_LOCKS.md` (DL-05, DL-06, DL-11, DL-18) → `PRD_AMENDMENTS.md` (A-02) → `SOURCE_BASELINE.md` (gap #4) → `PATH_OWNERSHIP.md` → current local source → L3 reference pack

## 1. Deliverable

New bounded pure-domain package `internal/intakefreeze` implementing the
confirmed-draft → frozen-engineering-artifacts mapping (PRD §3.2
"Structured Draft → Contract Validator → Frozen TaskContract"; PRD §15
"Confirm → Freeze"):

```go
func Map(confirmed ConfirmedDraft) (Frozen, error)
type ConfirmedDraft struct { TaskID, Profile, SourceRef string; Draft intake.Draft; RiskHints []string }
type Frozen struct { Intent intentevidence.FrozenIntent; Task taskcontract.TaskContract }
```

- `Frozen.Intent` is produced **only** through the P0-01 freeze entry
  `intentevidence.Freeze(intentevidence.Facts{...})` — the exact seam P0-01
  reserved for P0-06 ("Draft→Facts 的映射归 P0-06", `internal/intentevidence`
  package doc). No second freeze path, no schema knowledge duplicated.
- `Frozen.Task` is a plain `taskcontract.TaskContract` built by explicit
  field-by-field assignment, then validated by the existing authority
  (`Validate` + `ValidateStatic`) and hashed by content (`RevisionHash`),
  same discipline as `taskcontract.Parse`.

## 2. Mapping semantics (deterministic, fail-safe)

| Confirmed input | Frozen output | Rule |
|---|---|---|
| `TaskID` | `Intent.TaskID` + `Task.ID` | validated by `Freeze` (non-blank, no whitespace) and `taskcontract` validators (incl. `TASK-<digits>` static form) |
| `Profile` | `Task.Profile` | explicit governance input; must hit `taskcontract.IsValidProfile` exactly; leading/trailing whitespace folded as layout (same as `taskcontract.Parse` normalize), case variants rejected — **no default invented** |
| `Draft.TaskType` | `Intent.TaskType` | explicit two-arm `switch` translation intake → intentevidence vocabulary (compile-failure point if either side drifts); anything else → `ErrUnsupportedTaskType` |
| `SourceRef` | `Intent.SourceRef` only | never enters TaskContract (no such field) |
| `RiskHints` | `Intent.RiskHints` | **confirmed** hints only; `Draft.RiskSignals` are advisory candidates and are never auto-promoted to evidence (DL-06, L3 "Do"); blank entries rejected via `Freeze` |
| `Draft.Summary` / `Objective` | `Task.Title` / `Objective` | verbatim; blank objective rejected by `taskcontract.Validate` |
| `Draft.Scope` | `Task.Scope` | deep-copied; empty allowed_paths / blank entries / allowed×forbidden contradiction rejected by `Validate`/`ValidateStatic` |
| `Draft.Acceptance` | `Task.Acceptance` | id + statement verbatim; `IndependenceRequired` left undeclared (conservative `none` default — governance levels are not invented by mapping, intake-package boundary); duplicates / bad `AC-<digits>` form rejected by validators |
| `Draft.NonScope` / `Draft.Constraints` | — | **typed rejection** `ErrUnsupportedDraftSemantics` (see §3) |
| (draft has no refs/depends_on/documentation_impact) | `Task` zero values | no invented placeholder content |

## 3. Design decision: `non_scope` / `constraints` are rejected, not dropped

The frozen engineering schemas cannot carry these fields: `TaskContract` has
no such field (schema is closed; the package is not in this task's Allowed
Paths and DL-18 forbids duplicating the domain), and `FrozenIntent` is exactly
four fields (P0-01 ruling). The options were silent drop vs typed rejection.
Decision: **reject** (`ErrUnsupportedDraftSemantics`, any entry including
whitespace-only, field named in the message). Rationale:

- silently dropping owner-confirmed content would fork "what was confirmed"
  from "what is effective" — the exact divergence this codebase's strict
  parsers exist to prevent;
- the confirm UI (P0-04) renders both fields as editable textareas, so the
  owner's resolution path is real: fold the content into
  objective/scope/acceptance or delete it before freezing; the typed error
  names the field and says exactly that;
- this is the fail-safe-over-loose-normalization rule the task contract and
  AGENTS.md §"映射是纯领域变换" mandate.

"Presentation-only unresolved values" (L3 failure rule) are mechanically
covered by delegation: blank objective/statements, blank entries, placeholder
empty scopes are all rejected by the existing validators surfaced through
`ErrInvalidContract`/`ErrInvalidEvidence`. Text-heuristic placeholder linting
(`TBD` markers, cf. `requirement` L3 linter) is **not** duplicated here — it
is a lint/admission concern (`requirement.lint` idiom; P0-07 owns freeze
admission), and importing `internal/requirement` would drag its `llm`/`store`
dependency graph into a pure mapper.

## 4. Why this is the narrowest valid seam

- New package `internal/intakefreeze` only — an Allowed Path of this task,
  and the package name P0-01's report already reserved; P0-07 (verification
  authoring) and P0-08 (Git transaction) list the same package in their
  Allowed Paths, so downstream work extends it without colliding.
- No existing file touched: not `intake`, not `intentevidence`, not
  `taskcontract`, not the `v14_launch.go` hotspot (P0-04 → P0-08 serial
  ownership respected; the handler wiring is P0-08's).
- Dependencies: stdlib + `internal/intake` (input shape) +
  `internal/intentevidence` (freeze entry) + `internal/taskcontract`
  (validators/hash, read-only consumption). Test-only: `gopkg.in/yaml.v3`
  (whitelisted). No new module dependency (INV-6 whitelist unchanged).
- `internal/contract` (also an Allowed Path) was deliberately **not**
  touched: its `Compile` requires a frozen `verification.Contract`, which is
  P0-07's lane.

## 5. Exact source/tests read (L3 evidence)

- Task authority: `docs/tasks/v1.4.1/tasks/P0/P0-06_*.md`, `DECISION_LOCKS.md`,
  `PRD_AMENDMENTS.md`, `SOURCE_BASELINE.md`, `PATH_OWNERSHIP.md`,
  `MODEL_ASSIGNMENT.md`, `TASK_DAG.md` (P0-06/07/08 dependency rows), adjacent
  task contracts P0-07/P0-08 (to pin the output split: this task = FrozenIntent
  + TaskContract; P0-07 = verification admission; P0-08 = Git transaction).
- Tests first (reference order): `internal/taskcontract/parse_test.go`,
  `validate_test.go`, `types_test.go` (test inventory), `internal/verification/freeze_test.go`
  (read: standard-path freeze, hash equality with `Contract.Freeze`,
  author-side mutation isolation), `internal/intentevidence/*_test.go` inventory
  via P0-01 report.
- Interfaces/types: `intake.Draft`/`Intent`/`Scope`/`AcceptanceClause`/
  `ClarificationKind`, `intentevidence.Facts`/`Freeze`/`FrozenIntent`/
  `RiskSignalCandidates`, `taskcontract.TaskContract`/`Scope`/`AcceptanceClause`/
  `Validate`/`ValidateStatic`/`RevisionHash`, `contract.Compile` (read-only,
  to confirm non-ownership).
- Core implementation: `internal/intake/{intake,propose,clarification_schema}.go`,
  `internal/intentevidence/{intentevidence,artifact}.go`,
  `internal/taskcontract/{types,validate,parse,hash}.go`,
  `internal/verification/freeze.go`, `internal/contract/contract.go`,
  `internal/web/handlers/v14_launch.go` (read-only: `V14ConfirmDraft` /
  `v14DraftFromForm` / `v14FoldClarifications` — the confirmed-draft shape this
  mapper must accept), `internal/requirement/{doc,lint}.go` (unresolved-marker
  idiom, boundary reference only).
- External reference used: none (pack: current Tally source is canonical).

## 6. Semantic changes

- New production package `internal/intakefreeze` (1 file + 1 test file).
- No existing package, handler, worker, store, migration or template changed.
- No Decision Lock text touched; locks honored: DL-05 (risk stays derived —
  only confirmed hints are preserved, classification untouched), DL-06
  (confirmed facts survive Freeze as immutable evidence — via `Facts`), DL-11
  (presentation schema stays presentation — clarifications never reach this
  package; folded answers arrive inside the draft content), DL-18 (existing
  TaskContract/intentevidence domains extended by consumption, not duplicated).

## 7. Tests/commands executed (exact results)

Windows / Git Bash, Go `go1.27.1 windows/amd64`, clean tree before this change.

| # | Command | Result |
|---|---|---|
| 1 | `go test ./internal/taskcontract ./internal/verification ./internal/intake ./internal/intentevidence ./internal/contract ./internal/risk` (baseline, before change) | PASS — all `ok` |
| 2 | `gofmt -l internal/intakefreeze/` (after fixing one formatting nit found mid-development) | PASS — no output |
| 3 | `go test -count=1 -v ./internal/intakefreeze/` | PASS — 11/11 tests |
| 4 | `go test ./test/arch/...` | PASS — `ok tally/test/arch 1.096s` |
| 5 | `go build ./...` | PASS |
| 6 | `go vet ./internal/intakefreeze/` then `go vet ./...` | PASS |
| 7 | `go test -count=1 ./internal/...` (fresh) | PASS — 46 packages `ok` (2 with no test files), 0 failures |
| 8 | `go test -count=1 ./internal/intakefreeze/ ./internal/intake ./internal/intentevidence ./internal/taskcontract ./internal/contract ./test/arch/` (combined focused) | PASS — all `ok` |
| 9 | `make check` | NOT_RUN(reason: no `make` executable in the Git Bash environment; its constituent command families — arch guards, internal unit, integration — were executed directly as #4/#7 and noted below) |
| 10 | `go test -tags=integration ./test/e2e/...` | NOT_RUN(reason: integration path is opt-in and DB-gated; the task's Verification section requires golden tests + pre-commit rejection, both covered by #3) |

Negative/failure-path tests added (the pack requires at least one; eight
negative groups land):

- `TestMapRejectsInvalidProfile` (empty / case variant / unknown level;
  plus layout-whitespace acceptance variant);
- `TestMapRejectsUnsupportedTaskType` (empty, `BUG_FIX`, `Bug_Fix`, `bugfix`,
  `refactor` — no case folding, no guessing);
- `TestMapRejectsUnrepresentableDraftSemantics` (non_scope / constraints,
  substantive, multiple, and whitespace-only entries; error names the field);
- `TestMapSurfacesEvidenceValidationFailures` (blank identity, whitespace
  identity, blank hint entry → `intentevidence.ErrInvalidEvidence` via
  `errors.Is` through the wrapper);
- `TestMapRejectsInvalidContractSemantics` (blank objective, empty
  allowed_paths, duplicate clause id, `AC1` form, non-`TASK-<digits>` identity,
  allowed×forbidden scope contradiction → `ErrInvalidContract`);
- `TestMapFailureOrderIsDeterministic` (fixed validation order; profile first);
- `TestMapDoesNotMutateInput` (input untouched by Map; mutating input slices
  afterwards cannot move output hashes/fields — no shared backing arrays);
- `TestMapFreezesOnlyConfirmedRiskHints` (unconfirmed `Draft.RiskSignals`
  never leak into evidence).

Golden/determinism tests: `TestMapHappyPathDeterministicGolden` (two maps
deep-equal; field-by-field mapping; hash = recomputed revision; YAML
round-trip through the existing `taskcontract.Parse` authority yields the
same content hash — production artifact-family reachability in pure-function
form), `TestMapCanonicalizesOrderingVariants` (hint/scope/acceptance ordering
+ duplicate hints → identical revisions), `TestMapIntentMatchesDirectFreeze`
(mapped evidence identical to a direct `intentevidence.Freeze(Facts)`).

## 8. Baseline failures observed

None. Baseline command (#1) passed before the change; no pre-existing failure
was encountered or masked.

## 9. Acceptance criteria

- [x] **Mapper output is deterministic and contains no unresolved
  presentation-only fields** — determinism: #3 golden tests (identical input →
  identical output; ordering variants → identical identity; failure order
  fixed). No unresolved presentation-only fields: everything emitted is
  validated domain content (structural + static authority validation, #3);
  presentation-only/advisory values (clarification answers as such, candidate
  risk signals, confirmed-but-unrepresentable non_scope/constraints) never
  enter the output — the unrepresentable ones fail the map instead of being
  carried or dropped.

## 10. Production reachability

**N/A(owned integration tasks pending):** this task owns the pure domain
mapping only, per its contract ("No Git writes in pure mapper"). The
production paths that make it reachable are owned downstream: P0-07
(verification authoring/freeze admission in the same package) and P0-08
(confirm→freeze Git transaction wiring `intakefreeze.Map` behind
`V14ConfirmDraft`). No composition-root behavior changed in this task.

## 11. Completion report (per task contract format)

1. **status:** DONE
2. **changed files:** `internal/intakefreeze/intakefreeze.go` (new),
   `internal/intakefreeze/intakefreeze_test.go` (new),
   `docs/tasks/v1.4.1/tasks/P0/P0-06_draft_to_frozen_domain_mapper.md` (addendum),
   `docs/tasks/v1.4.1/reports/P0-06_draft_to_frozen_domain_mapper.md` (new),
   `docs/tasks/v1.4.1/STATUS.md` (10/57 → 11/57),
   `docs/architecture/CURRENT_CODEBASE.md`,
   `docs/implementation/CURRENT_IMPLEMENTATION_STATUS.md`
3. **semantic changes:** new pure draft→frozen mapping domain; confirmed-draft
   facts now have a deterministic, fail-safe path into `FrozenIntent` +
   `TaskContract`; unrepresentable confirmed semantics are typed-rejected
   instead of silently dropped; no existing behavior changed
4. **tests/commands:** see §7 (all PASS; `make check` and integration e2e
   NOT_RUN with reasons)
5. **baseline failures:** none
6. **remaining risks:** see §12
7. **Decision Lock touched:** none
8. **commit:** recorded in the task-file addendum / final message

## 12. Remaining risks / notes

- The `non_scope`/`constraints` rejection is a product-visible freeze gate:
  drafts carrying them cannot freeze until the owner resolves those fields in
  the confirm UI. If the program later wants them represented, that is a
  TaskContract schema change (P1-03 lane / ADR), not a mapper-local tweak.
- Blank `Summary` maps to an empty `Title` (title is optional in the
  TaskContract schema); the production confirm path already rejects blank
  summaries (`v14DraftFromForm`), and the mapper deliberately adds no extra
  requirement beyond the authority validators.
- Unresolved-marker linting (`TBD`-style) is intentionally out of scope here
  (see §3); if the program wants it on this boundary, it belongs to P0-07's
  freeze admission, reusing the `requirement` linter idiom.
- Intake↔intentevidence task-type vocabulary consistency is pinned by P0-01's
  test; the explicit `switch` here is an additional compile-time tripwire.
- HIGH review (per MODEL_ASSIGNMENT) has not yet executed; this report is the
  review input.
