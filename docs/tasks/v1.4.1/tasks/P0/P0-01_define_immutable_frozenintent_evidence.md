# P0-01 — Define immutable FrozenIntent evidence

- **Phase:** P0
- **Kind:** implementation
- **Wave:** W03
- **Complexity:** M
- **Execution model:** LOW
- **Depends on:** A0-05
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO


## Goal

Persist confirmed intake facts needed for deterministic downstream risk derivation without turning risk into mutable source state.


## Source facts

- Current intake risk signals are proposal-only.
- risk.Result is derived-only and requires basis refs.


## Allowed / primary paths

- `internal/intentevidence/**`


## Implementation requirements

1. Strict schema: task identity, task type, source ref, confirmed declared/risk hints only.
2. Deterministic canonicalization/hash/revision.
3. No provider/model prose blobs as authority.


## Acceptance criteria

- [ ] Round-trip is deterministic.
- [ ] Invalid/unknown authoritative fields fail closed.


## Verification

- Unit tests for canonicalization, validation, hash stability.

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
1. **Tests:** Tests: `internal/intake/propose_test.go::TestDraftHasNoAuthorityFields`, `internal/risk/risk_test.go`, `internal/contract/compile_test.go`
2. **Contract / Interface:** Interfaces: `intake.Draft` candidate `RiskSignals`; `risk.Result` derived facts; `contract.Bind(... risk.Result)`
3. **Core Implementation:** Core: `internal/intake/intake.go`, `internal/risk/risk.go`, `internal/risk/derive.go`
4. **Failure Handling:** Failure: unknown/unsupported confirmed facts must fail closed or remain explicit unknown; never silently disappear

## External reference pack

- No external implementation is required. Prefer current Tally source as the canonical implementation reference.

## Do

- Keep presentation, intent evidence and frozen engineering artifacts explicitly separated.
- Start by running the listed focused tests or the closest available subset.
- Make the smallest semantic change that satisfies this task; keep public surface area narrow.
- Add at least one negative/failure-path test for new behavior.

## Don't

- Do not let LLM-generated UI/risk text become authority without deterministic validation/freeze.
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

- **Status: DONE** (2026-09-12)
- Report: `docs/tasks/v1.4.1/reports/P0-01_define_immutable_frozenintent_evidence.md`
- Seam adjudication (owner ruling per DECISION_LOCKS "Open seams"): exact artifact path `.tally/intents/<task_id>.yaml` (flat, one `*.yaml` per task, same layout family as `.tally/{tasks,verifications,policies}`); exact schema = `task_id` / `task_type` (`bug_fix` | `db_migration`) / `source_ref` (optional) / `risk_hints` (optional, confirmed declared hints preserved verbatim) — nothing else; declared `revision` in YAML is rejected (hash is recomputed from content only). Ruling is pinned by `internal/intentevidence` constants + tests (`TestArtifactPathAdjudication`, `TestCanonicalPinsFormat`). `DECISION_LOCKS.md` itself was NOT edited (task contract grants no such authorization; ruling recorded in the report per the task's completion-report channel).
- Implemented: `internal/intentevidence` — immutable `FrozenIntent` (all fields unexported, value receivers only, no setters, defensive-copy accessors, zero value is not evidence), `Freeze(Facts)` + strict `Parse` as the only constructors, deterministic canonicalization + `sha256:` revision, canonical `Render` round-trip, `RiskSignalCandidates` projecting confirmed hints to `risk.Signal` with unified `intent:<task-id>` basis (out-of-enum hints preserved as Unknown via `risk.Derive` — DL-05/INV-V14-09).
- Validation: `go test ./internal/intentevidence/` PASS (16 tests, incl. negative/fail-closed: unknown YAML fields incl. provider/model prose and declared `revision`, unknown/uppercase task types, blank/whitespace identities and hints, empty/multi-document/non-mapping YAML, zero-value render); `go test -count=1 ./internal/intentevidence/ ./internal/risk ./internal/intake ./internal/contract ./test/arch/` PASS (fresh); `go build ./...` + `go vet ./...` PASS; `go run ./scripts/preflight` PASS — exit 0 `BASELINE_OK` (`unit_internal` = `go test ./internal/...` PASS; `integration` SKIP: NOT_RUN(reason: opt-in DB-gated, not requested)).
- Decision Locks touched: none (DL-06 satisfied by construction; DL-05 preserved — risk classification stays in `risk.Derive`); nothing under `docs/tasks/v1.4.1/hidden/` read (DL-17).
- Production reachability: N/A(owned integration tasks pending) — P0-02 Git loader, P0-06 draft→frozen mapper, P0-08 confirm→freeze transaction wire this domain; this task owns the type/schema/path ruling only.
