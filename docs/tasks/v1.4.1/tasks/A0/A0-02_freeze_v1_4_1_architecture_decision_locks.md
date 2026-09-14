# A0-02 — Freeze v1.4.1 architecture decision locks

- **Phase:** A0
- **Kind:** architecture
- **Wave:** W00
- **Complexity:** M
- **Execution model:** HIGH
- **Depends on:** A0-01
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO


## Goal

Publish the implementation constraints that low-cost agents are forbidden to violate.


## Allowed / primary paths

- `docs/tasks/v1.4.1/**`
- `docs/architecture/**`


## Implementation requirements

1. Lock existing Closed Loop/runtime/Git/policy/outcome authorities.
2. Mark unresolved seams as SPIKE_REQUIRED instead of inventing APIs.


## Acceptance criteria

- [ ] Decision locks are referenced by every later task template.
- [ ] No lock contradicts current source semantics.


## Verification

- Architecture review.

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

- **Execution:** `HIGH`
- **Review:** `HIGH`
- **Reason:** Architecture authority and cross-package invariants; a wrong lock creates downstream rework.
- **Review rule:** HIGH review required because the task touches an authority boundary, release gate, or high-blast-radius projection even when implementation is deterministic.

## Reference order — mandatory

Use evidence in this order. Do not jump directly to an external example before reading Tally's tests/contracts.
1. **Tests:** Tests first: `test/arch/v14_runtime_boundary_test.go`, `v14_contract_authority_test.go`, `v14_assurance_failsafe_test.go`, `v14_outcome_event_test.go`
2. **Contract / Interface:** Interfaces: `internal/closedloop/ports.go`, `internal/agentruntime/protocol.go`
3. **Core Implementation:** Core authority: `internal/engineeringpolicy/policy.go`, `internal/outcome/types.go`, `internal/gitwriter/writer.go`
4. **Failure Handling:** Failure handling: architecture tests already reject trust-core/runtime and policy/execution-policy leakage

## External reference pack

- No external implementation is required. Prefer current Tally source as the canonical implementation reference.

## Do

- Prefer source/AST evidence and executable guards over prose assertions.
- Start by running the listed focused tests or the closest available subset.
- Make the smallest semantic change that satisfies this task; keep public surface area narrow.
- Add at least one negative/failure-path test for new behavior.

## Don't

- Do not create implementation APIs while doing architecture/readiness work.
- Do not copy external code wholesale. Extract the pattern and adapt it to Tally's existing interfaces and dependency whitelist.
- Do not add a new dependency unless `test/arch/deps_test.go` and the dependency policy are intentionally updated by an approved architecture change.

## High-model execution requirement

This task should be performed by a high-capability model because success requires semantic judgment rather than mechanical editing. The model must explain the chosen seam, enumerate rejected alternatives, and demonstrate that fail-safe behavior remains intact before coding.

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

- **Status: DONE** (frozen at source HEAD `310d961`, 2026-09-12)
- Deliverable: `docs/tasks/v1.4.1/DECISION_LOCKS.md` frozen — status/authority/change-control header, binding interpretations BI-01 (PRD §17 via DL-08+DL-09+A-04) and BI-02 (PRD §27 via DL-12+A-06), open-seams SPIKE_REQUIRED register (5 seams with owning tasks), machine enforcement index (+40/−0; DL-01…DL-20 lock rows untouched)
- Report: `docs/tasks/v1.4.1/reports/A0-02_freeze_architecture_decision_locks.md`
- Acceptance: 57/57 task templates reference `DECISION_LOCKS` (AC-1); no lock contradicts source semantics — A0-01 facts F1–F17 + direct BI evidence checks + `go test -count=1 ./test/arch/...` → `ok tally/test/arch 0.869s` at the frozen baseline (AC-2)
- Baseline failures: none
