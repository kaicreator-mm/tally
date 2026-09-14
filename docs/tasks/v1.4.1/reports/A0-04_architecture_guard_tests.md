# A0-04 — Architecture guard tests: Execution Report

- **TASK:** A0-04
- **STATUS:** DONE
- **Baseline HEAD:** `c0f849860f90f70bbb42f9ea069fefda28adeb1f` (clean tree at start; A0-02 freeze commit)
- **Date:** 2026-09-12
- **Execution/Review model:** LOW / HIGH (per task pack; HIGH review of an authority-boundary guard suite is the pack's own rule — no source or API mismatch was encountered)

## 1. Goal and deliverable

Add low-cost structural tests that detect accidental creation of duplicate
authorities, covering the v1.4.1-specific readings that the frozen
`DECISION_LOCKS.md` "Machine enforcement index" assigns to A0-04:

| Guard (new) | Lock reading | What fails it |
|---|---|---|
| `TestV141SingleCoreRuntimeProtocol` | DL-02 / A-01 (v141-DL02) | (a) any type named `ExecutionBackend` (the A-01-retired abstraction) anywhere in `internal/`+`runtimes/`+`cmd/`; (b) an exported *interface* named `Runtime` or `Session` outside `internal/agentruntime` (concrete adapter structs like `runtimes/codex.Runtime` and the sqlc row model `store.Session` remain legal); (c) an exported interface outside `internal/agentruntime` that copies the protocol opening shape (`Descriptor()` + `Start`) while referencing no `agentruntime.*` vocabulary — a shape copy in a different vocabulary is a second protocol |
| `TestV141PolicySchemaExcludesGovernanceFields` | DL-08 / A-04 via BI-01 (v141-BI01 structural) | any struct field or yaml key in `internal/engineeringpolicy` matching the PRD §17 governance vocabulary (`status`, `version`, `policy_id`, `evidence_refs`, `created_from_cases`, `approved_by`, `approved_at`, `supersedes`, `scope`), or a const string equal (case-insensitively) to a governance lifecycle value (`DRAFT`/`PROPOSED`/`ACTIVE`/`SUPERSEDED`/`REJECTED`) |
| `TestV141PolicyParseRejectsGovernanceFields` | BI-01 behavioral | `engineeringpolicy.Parse` accepting a policy YAML that carries any governance field (negative test injecting each field into a valid gate-policy skeleton) |
| `TestV141NoDatabasePolicyActivationFlag` + `TestV141PolicyActivationGuardDetectsViolations` | BI-01 persistence clause (v141-BI01) | any migration creating or altering a policy-named table with an activation flag column (`active`/`is_active`/`runtime_active`/`is_runtime_active`/`activation`) — activation may only be a `gitwriter` commit of canonical policy YAML; a separate governance table's `status` column is deliberately NOT flagged (BI-01 allows governance projections to own their lifecycle) |
| `TestV141OutcomeKindSetStaysFive` | DL-12 / A-06 via BI-02 (v141-BI02 structural) | the migration `0023` CHECK enum containing anything beyond the five kinds (the existing AC1 test only pins presence — this pins closure on the DB side too); a sixth `Kind` constant in `internal/outcome`; PRD §27 labels (`rejected`, `escalated`, `integration_failed`, and raw spellings `human_corrected`, `hotfix_required`, `regression_detected`) appearing as kind constants, query literals, or sqlc-generated literals; an `event_kind` column in any table other than `outcome_event` (parallel case-history store) |
| `TestV141OutcomeParseKindRejectsPRD27Labels` + `TestV141OutcomeGuardsDetectViolations` | BI-02 behavioral | `outcome.ParseKind` accepting a §27 label or a case variant (fail-safe: unknown kinds are rejected, never folded into a known kind) |

All new guards live in one file, `test/arch/v141_decision_lock_guards_test.go`
(package `arch`), following the existing `test/arch` idiom (AST/source-inspection
over `go/parser`, SQL structural checks over `readAllMigrations`/`createTableBody`-
style regexes, behavior checks through the real `engineeringpolicy.Parse` /
`outcome.ParseKind` entry points, Chinese commentary citing the lock ids).

## 2. Requirement 4 — checks that cannot be automated (review-enforced remainder)

Documented in the guard file header and repeated here:

- **DL-02 composition-root facts:** that `closedloop.Executor` adapters actually
  route through `agentruntime.Runtime` and that one attempt uses one runtime
  session (`cmd/worker/v14.go`) are wiring-level facts; static AST can only pin
  protocol *declaration* uniqueness. Wiring correctness belongs to runtime
  conformance (P1-25 precedent) and review.
- **BI-01 semantic face:** "runtime consumes ACTIVE" = "the exact policy blob at
  `.tally/policies/*.yaml` in the pinned commit, loaded via
  `artifacts.LoadPolicies`" is a data-flow property; the separation of governance
  facts into the P3 lane and the "activation = gitwriter commit" process
  invariant are not machine-checkable before P3 lands. The guards pin the
  negative space (governance fields / lifecycle vocabulary / activation flags
  must not appear).
- **BI-02 full generality:** a deliberate parallel event store that avoids the
  `event_kind` column name would be invisible to static scanning; column-name
  uniqueness is the machine-guardable lower bound, the rest is review.

## 3. Reference order compliance (L3 evidence)

Exact sources/tests read, in the mandated order (local TaskContract → Decision
Locks → current local source → references):

1. **Task discipline docs:** `docs/tasks/v1.4.1/DECISION_LOCKS.md` (locks table,
   BI-01/BI-02, machine enforcement index), `PRD_AMENDMENTS.md` (A-01/A-04/A-06),
   `SOURCE_BASELINE.md`, `PATH_OWNERSHIP.md`; PRD §17 and §27 sections
   referenced by BI-01/BI-02.
2. **Tests (contract/idiom):** `test/arch/v14_runtime_boundary_test.go`
   (AC2a/2b/3a/3b — DL-02/DL-08's existing guards; the new DL-02 guard
   complements, does not overlap: AC2 pins *isolation between the two runtime
   systems*, v141 pins *protocol declaration uniqueness*), `v14_outcome_event_test.go`
   (AC1–AC5; reused fixtures `outcomeKinds`, `readOutcomeMigration`,
   `readOutcomeQueries`, `readOutcomeFile`, `outcomeGeneratedPath`),
   `v14_contract_authority_test.go`, `v14_lifecycle_state_test.go`
   (`v14ParsePackage`), `helpers_test.go` (`goFiles`, `readAllMigrations`),
   `job_lease_contract_test.go` (`sqlcQuery`, `normSQL`).
3. **Core implementation (nearest seams only):** `internal/agentruntime/protocol.go`
   (Runtime/Session shape), `internal/closedloop/ports.go` (consumer ports),
   `internal/engineeringpolicy/policy.go` + `parse.go` (Policy schema,
   `KnownFields(true)` strict parsing), `internal/outcome/types.go`
   (closed enum + `ParseKind`), `internal/artifacts/v14_policy_loader.go`
   (LoadPolicies), `migrations/0023_v14_outcome_event.sql` (CHECK + append-only
   trigger), `cmd/worker/v14.go` (runtime field usage).
4. **Calibration facts that shaped the guards:** `runtimes/codex.Runtime` /
   `runtimes/fake.Runtime` / `store.Session` are structs (so the interface-name
   check does not false-positive on adapters); no `polic*` table exists in
   migrations today; `event_kind` exists only in `outcome_event`.

**External reference actually used:** none consulted. The pack lists the
in-repo `test/arch` idiom as the primary reference and EXT-UBER-GO-TEST as
pattern-only; the existing repo style (loop-per-vocabulary, fail-fast
`t.Errorf` with lock citations) was followed directly. No external code copied.

**Why this is the narrowest valid seam:** the pack's own reference order
designates `test/arch/*.go` as the contract/idiom location and the frozen
machine enforcement index expects v1.4.1 guards to extend that suite; a single
new `v141_`-prefixed test file adds guards with zero production changes, zero
new dependencies (stdlib + two existing internal packages already imported by
sibling test files), and no modification to any existing test or shared hotspot.
The task file's "Allowed / primary paths" (`internal/**`, `cmd/**`,
`docs/tasks/v1.4.1/**`) was read together with the same file's L3 pack; the
pack and the machine enforcement index both place guard tests in `test/arch`,
so the new file lives there. `internal/**` and `cmd/**` were **not** modified —
the task needed no production change, and none was invented.

## 4. Changed files

- `test/arch/v141_decision_lock_guards_test.go` — new guard suite (8 tests + helpers)
- `docs/tasks/v1.4.1/reports/A0-04_architecture_guard_tests.md` — this report
- `docs/tasks/v1.4.1/tasks/A0/A0-04_architecture_guard_tests.md` — status addendum appended
- `docs/tasks/v1.4.1/STATUS.md` — progress 3/57 → 4/57

No production code, no migration, no existing test, no Decision Lock file, no
`go.mod`/`go.sum`, no shared hotspot was modified. Existing guards were neither
weakened nor deleted; no existing file was touched at all.

## 5. Validation executed (exact commands and results)

Windows / Git Bash, HEAD `c0f8498`, Go `go1.27.1`.

| # | Command | Result |
|---|---|---|
| 1 | `go test -count=1 ./test/arch/ -run "TestV141" -v` | PASS — 8/8 new tests |
| 2 | `go test -count=1 ./test/arch/...` (task-required architecture guard suite, uncached) | PASS |
| 3 | `go vet ./test/arch/...` | PASS |
| 4 | `gofmt -l test/arch/v141_decision_lock_guards_test.go` | PASS (no output; note: several pre-existing arch test files are flagged by `gofmt -l` before this task — untouched, not this task's scope) |
| 5 | `git diff --check` | PASS (no whitespace/line-ending issues; only the new files staged, no normalization churn) |
| 6 | **Negative-path probe (DL-02):** temporary `internal/workspace/zz_v141_probe.go` declaring `ExecutionBackend`, `Runtime`/`Session` interfaces, and a non-agentruntime `Descriptor+Start` shape → `go test ./test/arch/ -run TestV141SingleCoreRuntimeProtocol` | FAIL as expected — all three violation shapes flagged with file-precise messages; probe deleted afterwards, `git status` clean |
| 7 | **Negative-path probe (BI-01 structural):** temporary `internal/engineeringpolicy/zz_v141_probe.go` with `Status`/`ApprovedBy` fields + `yaml:"status"`/`yaml:"approved_by"` tags + const `"ACTIVE"` → `go test ./test/arch/ -run TestV141PolicySchemaExcludesGovernanceFields` | FAIL as expected — field-name, yaml-key, and lifecycle-const checks all fired; probe deleted afterwards |
| 8 | Re-run after probe removal: `go test -count=1 ./test/arch/` | PASS |

Permanent negative tests inside the suite (no source mutation needed):
`TestV141PolicyParseRejectsGovernanceFields` (10 governance fields must be
rejected by `Parse`), `TestV141OutcomeParseKindRejectsPRD27Labels` (6 §27 labels
+ `""`/`"Accepted"` must yield `ErrUnknownKind`; the five kinds must parse —
fixture same-source check), `TestV141PolicyActivationGuardDetectsViolations`
(synthetic `is_active`/`runtime_active` migrations must be flagged; a governance
`status` column must NOT be flagged), `TestV141OutcomeGuardsDetectViolations`
(synthetic sixth CHECK value and a synthetic parallel `event_kind` table must be
detected).

## 6. Acceptance criteria

- **AC "Known forbidden architecture changes fail a guard or documented review
  gate":** **PASS**
  - Forbidden change "second core runtime protocol" → fails
    `TestV141SingleCoreRuntimeProtocol` (proven by probe: ExecutionBackend
    revival, shadow `Runtime`/`Session` interfaces, and vocabulary-free
    `Descriptor+Start` shape copy each flagged).
  - Forbidden change "governance fields on `engineeringpolicy.Policy`" → fails
    `TestV141PolicySchemaExcludesGovernanceFields` (probe-proven) and
    `TestV141PolicyParseRejectsGovernanceFields` (behavioral).
  - Forbidden change "DB activation flag / runtime-active declaration" → fails
    `TestV141NoDatabasePolicyActivationFlag` (negative-proven by synthetic
    migration).
  - Forbidden change "Outcome kind-set expansion / §27 labels as kinds /
    parallel event store" → fails `TestV141OutcomeKindSetStaysFive` and
    `TestV141OutcomeParseKindRejectsPRD27Labels` (negative-proven).
  - The remainder that cannot fail a guard (wiring, data-flow, activation-as-
    commit process) is explicitly documented as review-enforced (§2) —
    satisfying the AC's "or documented review gate" branch.

## 7. Baseline vs introduced failures

- **Baseline failures observed at HEAD `c0f8498`:** none. `go test ./test/arch/...`
  passes uncached; `go vet` clean.
- **Introduced failures:** none outstanding. The only FAILs during the task were
  the two deliberate probe runs (§5 #6/#7), which are the negative path working
  as designed; probes were deleted and re-verified green.

## 8. Known uncertainties / version sensitivity

- The `Descriptor+Start` shape-copy check requires the pair and an
  agentruntime-vocabulary-free signature; a future protocol copy that changes
  the opening method names entirely (neither `Descriptor` nor `Start`) would
  evade check (c) — it would still have to survive (a)/(b) and review. This is
  the documented precision/fragility tradeoff (pack: "avoid brittle
  exact-source-string tests where semantic test is possible").
- Activation-flag vocabulary is name-based; a future column named differently
  (e.g. `enabled`) on a policy table would evade the regex — BI-01's semantic
  clause remains review-enforced (§2).
- `gofmt -l` flags several pre-existing `test/arch` files (formatting drift
  predating this task, likely CRLF-era); not touched, not counted as baseline
  failures since `go test`/`go vet` pass.
- `sqlcQuery`'s error message cites `jobLeaseQueriesPath` (existing helper
  quirk); irrelevant on the happy path used here.

## 9. Decision Locks

**DL_TOUCHED: no.** No lock was modified, weakened, or bypassed; the guards
*enforce* DL-02/DL-08/DL-12 readings BI-01/BI-02 exactly as frozen in A0-02.
`DECISION_LOCKS.md` was not edited (the machine enforcement index's status
annotations were left as frozen; the contract does not assign A0-04 to update
them). DL-17 respected: nothing under `docs/tasks/v1.4.1/hidden/` was read,
listed, or summarized.

## 10. Remaining risks

- Guard coverage is intentionally bounded to the frozen v1.4.1 readings; future
  ADRs that legitimately widen the Outcome domain or add governance facts (P3)
  will need to update these tests together with the locks — the tests fail
  loudly by design, which is the intended friction.
- The protocol-uniqueness scan parses `internal/`, `runtimes/`, `cmd/` fully
  AST-once per run (~0.1s observed); cost is negligible in the suite (0.9s total).
