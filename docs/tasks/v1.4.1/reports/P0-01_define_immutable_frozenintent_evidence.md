# P0-01 — Define immutable FrozenIntent evidence: Execution Report

- **TASK:** P0-01
- **STATUS:** DONE
- **Date:** 2026-09-12
- **Phase / Wave:** P0 / W03
- **Execution/Review model:** LOW / HIGH (per task pack: authority boundary — HIGH review still required on this report)
- **Depends on:** A0-05 (READY_TO_IMPLEMENT) — satisfied
- **Hidden material:** none read; `docs/tasks/v1.4.1/hidden/` untouched (DL-17)
- **Authority chain read in order:** task contract → `DECISION_LOCKS.md` (DL-05, DL-06, open-seams table) → `PRD_AMENDMENTS.md` (A-02) → `SOURCE_BASELINE.md` (gap #4) → current local source → L3 reference pack

## 1. Deliverable

The P0-01 ruling of the `DECISION_LOCKS.md` open seam **"FrozenIntent exact file
path / schema"** (owner: P0-01), implemented as the new bounded domain package
`internal/intentevidence` with tests:

- **Exact file path ruling:** `.tally/intents/<task_id>.yaml` — flat directory,
  one `*.yaml` per task, the same layout family as `.tally/tasks/TASK-*.yaml`,
  `.tally/verifications/TASK-*.yaml`, `.tally/policies/*.yaml` (INV-ART-01
  Git-first artifact discipline). Artifact identity is content-derived
  (`task_id` + content revision hash), never the filename — the same rule the
  policy loader applies. Authority: `ArtifactDir` / `ArtifactPath` constants.
- **Exact schema ruling (four fields, nothing else):**
  - `task_id` — required, no whitespace (task identity);
  - `task_type` — required, exactly `bug_fix` | `db_migration` (the frozen MVP
    vocabulary, proven equal to `intake`'s by test);
  - `source_ref` — optional external reference (blank = undeclared);
  - `risk_hints` — optional list of the owner-confirmed declared risk hints,
    preserved verbatim (sorted/deduped only).
  - Nothing else is a field: provider/model/prompt/raw-output prose and any
    governance/lifecycle vocabulary are rejected as unknown fields; a declared
    `revision` is rejected too — the hash is always recomputed from content
    (same anti-declaration rule as `engineeringpolicy`).

## 2. Immutability semantics (type level)

- `FrozenIntent` fields are all unexported; there is no setter and no
  pointer-receiver method (both pinned by reflection in
  `TestFrozenIntentImmutableByType`).
- The only constructors are `Freeze(Facts)` and `Parse([]byte)`; both validate
  fail-closed first, then compute the revision. There is no partial-freeze
  state: failures return the zero value, and the zero value is not evidence
  (`IsZero`); rendering it is refused.
- Collection accessors return defensive copies; mutating input `Facts` after
  `Freeze` cannot change frozen evidence (`TestFreezeIsolatesInputFacts`).
- Revision is `sha256:<hex>` over a versioned canonical text
  (`intent/v1`, format pinned by `TestCanonicalPinsFormat`); any content
  change changes the revision, layout/ordering/whitespace changes never do
  (`TestRevisionChangesOnlyWithContent`).

## 3. Risk stays derived (DL-05) and confirmed facts survive freeze (DL-06)

`FrozenIntent` stores confirmed declared hints verbatim; classification into
known/unknown is left to `risk.Derive`. `RiskSignalCandidates()` projects each
hint to a `risk.Signal` with the unified basis format `intent:<task-id>` (the
`<kind>:<id>` form the risk package documents). A test proves an
out-of-enum confirmed hint survives into `Result.Unknown`, keeps `HasUnknown()`
and `AboveMinimal()` true — it never silently disappears and is never read as
low risk (INV-V14-09 fail-safe).

## 4. Why this is the narrowest valid seam

- New package only under the task's Allowed Paths (`internal/intentevidence/**`);
  no existing package, handler, worker or store file touched.
- Dependencies: stdlib + `gopkg.in/yaml.v3` (already whitelisted) +
  `tally/internal/risk` (dependency-free derived model). No dependency on
  `intake`/`airuntime` — confirmed facts enter via the explicit `Facts` value,
  so AI-Runtime-facing input code cannot leak into the evidence schema.
- Downstream tasks keep their own ownership: P0-02 (Git artifact loader in
  `internal/artifacts`) consumes `Parse` + `ArtifactPath` + accessors for
  identity checks; P0-06 (draft→frozen mapper in `internal/intakefreeze`)
  consumes `Freeze(Facts)`; P0-08 wires the transaction. This task neither
  duplicates nor pre-empts them.
- Idioms follow existing authority code (`taskcontract.Canonical/RevisionHash`,
  `taskcontract.Parse` strict KnownFields/single-doc, `engineeringpolicy`
  content-only revision) so reviewers and downstream loaders meet no new
  convention.

## 5. Exact source/tests read (L3 evidence)

- Tests first (per reference order): `internal/intake/propose_test.go::TestDraftHasNoAuthorityFields`, `internal/risk/risk_test.go` (suite), `internal/contract/compile_test.go` (suite), plus baseline run of all three packages.
- Interfaces/types: `intake.Draft` (+`Intent`, `TaskType`), `risk.Result`/`Signal`/`Derive`, `contract` Pins/hash.
- Core: `internal/intake/intake.go`, `internal/risk/risk.go`, `internal/risk/derive.go`, `internal/contract/hash.go`, `internal/taskcontract/{types,hash,parse}.go`, `internal/artifacts/v14_policy_loader.go`, `internal/engineeringpolicy/policy.go` (revision pattern), `test/arch/v141_decision_lock_guards_test.go`, `test/arch/deps_test.go`, `go.mod`.
- External reference used: none (pack says current Tally source is canonical).
- Docs updated per AGENTS.md §15: `docs/architecture/CURRENT_CODEBASE.md` (new package in topology; gap #3 restated with the landed domain and the open wiring tasks), `docs/implementation/CURRENT_IMPLEMENTATION_STATUS.md` (progress note 6/57).

## 6. Semantic changes

- New production package `internal/intentevidence` (3 files: value type +
  artifact form; no existing file's semantics changed).
- New tests: 16 (2 files), including the negative/failure-path tests required
  by the pack (see §7).
- `DECISION_LOCKS.md` was NOT edited. The task contract grants no authorization
  to touch it (its completion report instead asks whether a lock was touched),
  and its change-control paragraph reserves lock changes for ADR/amendment.
  The seam ruling is therefore recorded here and enforced in code+tests
  (`ArtifactDir`/`ArtifactPath` constants, schema via strict parser, pinned
  canonical format). Downstream owners (P0-02 etc.) and reviewers should treat
  this report + the guard tests as the seam's authoritative record.

## 7. Tests/commands executed (exact results)

Windows / Git Bash, Go `go1.27.1 windows/amd64`, clean tree before this change.

| # | Command | Result |
|---|---|---|
| 1 | `go test ./internal/risk ./internal/intake ./internal/contract` (baseline, before change) | PASS — all `ok` |
| 2 | `go test ./internal/intentevidence/` | PASS — `ok tally/internal/intentevidence 0.743s` (16/16) |
| 3 | `go test -count=1 ./internal/intentevidence/ ./internal/risk ./internal/intake ./internal/contract ./test/arch/` | PASS — all `ok` (fresh, uncached) |
| 4 | `go test ./test/arch/...` | PASS — `ok tally/test/arch 0.922s` (incl. A0-04 v1.4.1 guards) |
| 5 | `go test ./internal/taskcontract ./internal/engineeringpolicy ./internal/artifacts` (adjacent idiom packages) | PASS — all `ok` |
| 6 | `go build ./...` | PASS |
| 7 | `go vet ./...` | PASS |
| 8 | `go run ./scripts/preflight` | PASS — exit 0, `BASELINE_OK`, `is_code_regression=false`; `unit_internal` = `go test ./internal/...` PASS (24.3s); `integration`: SKIP — NOT_RUN(reason: opt-in DB-gated, not requested by this task's Verification section) |
| 9 | `gofmt -l internal/intentevidence/` | PASS — no output |
| 10 | `git diff --check` (pre-commit) | PASS — clean |

Negative/failure-path tests added: `TestFreezeFailsClosed` (blank/whitespace
task identity, empty/out-of-vocabulary/uppercase task types, blank hint
entries), `TestParseFailsClosed` (11 unknown-field probes incl.
provider/model/prompt/raw_output/status/frozen/declared revision; empty doc,
multi-doc, missing/whitespace identity, blank hints, out-of-vocabulary and
case-variant task types), `TestParseRejectsNonMapping`,
`TestRenderZeroValueFailsClosed`, `TestRiskSignalCandidatesPreserveUnknown`
(out-of-enum hint stays Unknown and above the minimal-risk floor),
`TestTaskTypeVocabularyMatchesIntake` (vocabulary-drift tripwire).

One test initially failed during development (`ArtifactPath("TASK-0023\n")`):
the test had mis-encoded the house rule — leading/trailing whitespace is layout
and is collapsed (same as `taskcontract.normalize`), internal whitespace is
rejected. Design kept, test corrected; no fail-safe behavior was weakened.

## 8. Baseline failures observed

None. All baseline commands in §7 #1 passed before the change; no pre-existing
failure was encountered or masked.

## 9. Acceptance criteria

- [x] **Round-trip is deterministic** — `TestRenderRoundTripDeterministic`
  (Freeze → Render → Parse equal in content and revision; Render is a byte
  fixed point), `TestParseFormattingVariantsSameRevision` (key order, flow vs
  block lists, quoting variants → same revision), `TestFreezeCanonicalizesDeterministically`.
- [x] **Invalid/unknown authoritative fields fail closed** —
  `TestParseFailsClosed`, `TestFreezeFailsClosed`, `TestParseRejectsNonMapping`,
  `TestRenderZeroValueFailsClosed` (typed sentinels `ErrInvalidEvidence`,
  `ErrUnsupportedTaskType`, matched via `errors.Is`).

## 10. Production reachability

**N/A(owned integration tasks pending):** this task owns the domain type,
schema and path ruling only. The production paths that make it reachable are
owned downstream — P0-02 (pinned-commit Git loader), P0-06 (confirmed-draft →
`Freeze(Facts)` mapping), P0-08 (confirm→freeze Git transaction). No
composition-root behavior changed in this task.

## 11. Completion report (per task contract format)

1. **status:** DONE
2. **changed files:** `internal/intentevidence/intentevidence.go` (new), `internal/intentevidence/artifact.go` (new), `internal/intentevidence/intentevidence_test.go` (new), `internal/intentevidence/artifact_test.go` (new), `docs/tasks/v1.4.1/tasks/P0/P0-01_...md` (addendum), `docs/tasks/v1.4.1/reports/P0-01_...md` (new), `docs/tasks/v1.4.1/STATUS.md` (5/57 → 6/57), `docs/architecture/CURRENT_CODEBASE.md`, `docs/implementation/CURRENT_IMPLEMENTATION_STATUS.md`
3. **semantic changes:** new immutable-evidence domain; FrozenIntent exact path/schema adjudicated (`.tally/intents/<task_id>.yaml`; 4-field closed schema); risk classification remains in `risk.Derive` (DL-05); no existing behavior changed
4. **tests/commands:** see §7 (all PASS; integration NOT_RUN: opt-in, not required by task)
5. **baseline failures:** none
6. **remaining risks:** see §12
7. **Decision Lock touched:** none
8. **commit:** recorded in the task-file addendum / final message

## 12. Remaining risks / notes

- The seam ruling lives in code + tests + this report, not in
  `DECISION_LOCKS.md` (no authorization to edit it). If the program wants the
  open-seams table row flipped to RESOLVED, that is a one-line docs decision
  above this task's authority (record it via PRD_AMENDMENTS/ADR or grant it to
  a docs-owning task).
- `intentevidence → risk` is a new internal import edge (stdlib-only target);
  it exists to unify the `intent:<task-id>` basis format and to keep the
  unknown-hint fail-safe testable at the evidence boundary. No arch guard
  restricts this edge.
- YAML rendering uses `gopkg.in/yaml.v3` (pinned in go.mod); its escaping is
  deterministic, but a future dependency bump could alter byte layout. The
  revision hash does not depend on rendering (only on canonical text), and the
  round-trip/fixed-point tests would catch any behavior drift loudly.
- Task-type vocabulary is deliberately re-declared in this package (no
  production import of `intake`); a vocabulary extension (third task type)
  must update both sides — `TestTaskTypeVocabularyMatchesIntake` fails loudly
  if they drift.
- HIGH review (per MODEL_ASSIGNMENT) has not yet executed; this report is the
  review input.
