# A0-04 — Architecture guard tests

- **Phase:** A0
- **Kind:** tests
- **Wave:** W01
- **Complexity:** M
- **Execution model:** LOW
- **Depends on:** A0-02
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO


## Goal

Add low-cost structural tests/checks that detect accidental creation of duplicate authorities.


## Allowed / primary paths

- `internal/**`
- `cmd/**`
- `docs/tasks/v1.4.1/**`


## Implementation requirements

1. Guard no second runtime core protocol.
2. Guard EngineeringPolicy runtime schema from governance fields where practical.
3. Guard Outcome kind set from accidental expansion in v1.4.1.
4. Document checks that cannot be automated.


## Acceptance criteria

- [ ] Known forbidden architecture changes fail a guard or documented review gate.


## Verification

- Run architecture guard suite.

## Mandatory implementation discipline

- Read `../../DECISION_LOCKS.md`, `../../PRD_AMENDMENTS.md`, and `../../SOURCE_BASELINE.md` first.
- Do not broaden scope to “clean up” adjacent architecture.
- Do not weaken fail-safe behavior to make tests pass.
- Preserve source-commit identity and held-out verification boundaries.
- If an expected seam does not exist, stop this task as `BLOCKED_BY_SOURCE_FACT` and report the exact source evidence instead of inventing a parallel subsystem.
- New behavior needs focused tests.
- Existing baseline failures must be distinguished from introduced failures.

## Completion report

Report exactly:

1. status: `DONE | BLOCKED | PARTIAL`;
2. changed files;
3. semantic changes;
4. tests/commands executed with exact results;
5. baseline failures observed;
6. remaining risks;
7. whether any Decision Lock was touched;
8. commit hash (if committed).

---

# L3 Task-Level Implementation Reference Pack

## Model assignment

- **Execution:** `LOW`
- **Review:** `HIGH`
- **Reason:** The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred.
- **Review rule:** HIGH review required because the task touches an authority boundary, release gate, or high-blast-radius projection even when implementation is deterministic.

## Reference order — mandatory

Use evidence in this order. Do not jump directly to an external example before reading Tally's tests/contracts.
1. **Tests:** Tests: use existing architecture-test idiom in `test/arch/*.go`
2. **Contract / Interface:** Contracts: `v14_runtime_boundary_test.go`, `v14_outcome_event_test.go`, `v14_contract_authority_test.go`
3. **Core Implementation:** Core: AST/source-inspection helpers in `test/arch/helpers_test.go`
4. **Failure Handling:** Failure: guard only stable architecture locks; avoid brittle exact-source-string tests where semantic test is possible

## External reference pack
- **EXT-UBER-GO-TEST — Uber Go Style Guide — table-driven tests**  
  https://github.com/uber-go/guide/blob/master/style.md  
  License/version: CC BY 3.0 / repository terms; pattern only; master.  
  Use: Reference table-driven test structure for deterministic validation matrices.

## Do

- Prefer source/AST evidence and executable guards over prose assertions.
- Start by running the listed focused tests or the closest available subset.
- Make the smallest semantic change that satisfies this task; keep public surface area narrow.
- Add at least one negative/failure-path test for new behavior.

## Don't

- Do not create implementation APIs while doing architecture/readiness work.
- Do not copy external code wholesale. Extract the pattern and adapt it to Tally's existing interfaces and dependency whitelist.
- Do not add a new dependency unless `test/arch/deps_test.go` and the dependency policy are intentionally updated by an approved architecture change.

## Low-cost model execution recipe

1. Read the exact tests listed above.
2. Read the interface/types used by those tests.
3. Read only the nearest implementation seam, not the whole repository.
4. Implement one bounded change.
5. Run focused tests.
6. Run relevant `test/arch` guards.
7. If expected source facts differ from this pack, stop with `BLOCKED_BY_SOURCE_FACT`; do not redesign the architecture.

## L3 completion evidence required from implementing agent

- exact source/tests read;
- any external reference actually used;
- why the chosen seam is the narrowest valid seam;
- changed files;
- exact test commands/results;
- negative tests added;
- known uncertainty or version sensitivity;
- confirmation that Decision Locks were not bypassed.

---

## Execution status

- **Status: DONE** (executed at HEAD `c0f8498`, 2026-09-12)
- Deliverable: `test/arch/v141_decision_lock_guards_test.go` — 8 guard tests (package `arch`, existing test idiom) covering the v1.4.1-specific readings the frozen DECISION_LOCKS machine enforcement index assigns to A0-04:
  - **v141-DL02** protocol uniqueness (DL-02/A-01): no `ExecutionBackend` revival; exported `Runtime`/`Session` *interfaces* only in `internal/agentruntime` (adapter structs and the sqlc `Session` row model stay legal); no vocabulary-free `Descriptor()+Start` shape copy outside the protocol package
  - **v141-BI01** governance-field exclusion (DL-08/A-04, binding PRD §17): `internal/engineeringpolicy` struct fields / yaml keys / const values free of the governance vocabulary (`status`, `version`, `policy_id`, `evidence_refs`, `created_from_cases`, `approved_by`, `approved_at`, `supersedes`, `scope`; `DRAFT`/`PROPOSED`/`ACTIVE`/`SUPERSEDED`/`REJECTED`); `engineeringpolicy.Parse` behaviorally rejects all ten injected governance fields; no migration may put an activation flag column (`active`/`is_active`/`runtime_active`/`is_runtime_active`/`activation`) on a policy-named table — a separate governance table's `status` column is deliberately not flagged (BI-01: governance facts are a separate P3 lane)
  - **v141-BI02** Outcome kind-set stability (DL-12/A-06, binding PRD §27): `0023` CHECK enum contains exactly the five kinds (closure, not just presence); no sixth `Kind` constant; §27 labels (`rejected`/`escalated`/`integration_failed`, raw spellings `human_corrected`/`hotfix_required`/`regression_detected`) absent as kind constants/query literals/generated literals; `event_kind` column exists only in `outcome_event` (no parallel case-history store); `outcome.ParseKind` fail-safe-rejects §27 labels, `""`, and case variants
- Report: `docs/tasks/v1.4.1/reports/A0-04_architecture_guard_tests.md`
- Acceptance: known forbidden architecture changes fail a guard — probe-proven negative paths (temporary probe files made `TestV141SingleCoreRuntimeProtocol` and `TestV141PolicySchemaExcludesGovernanceFields` FAIL with file-precise messages, then removed) plus permanent synthetic negative tests (activation-flag migrations, extra CHECK value, parallel `event_kind` table, `Parse`/`ParseKind` rejections); the non-automatable remainder (composition-root wiring, LoadPolicies data-flow / activation-as-commit process, renamed-column parallel store) is documented as review-enforced per implementation requirement 4
- Validation: `go test -count=1 ./test/arch/...` PASS (uncached); `go vet ./test/arch/...` PASS; `gofmt -l` clean on the new file; `git diff --check` clean
- Changed files: `test/arch/v141_decision_lock_guards_test.go` (new), this task file (addendum), `docs/tasks/v1.4.1/reports/A0-04_architecture_guard_tests.md` (new), `docs/tasks/v1.4.1/STATUS.md` (4/57). No production code, migration, existing test, or shared hotspot touched
- Decision Locks touched: none (guards enforce frozen readings; no lock edited)
