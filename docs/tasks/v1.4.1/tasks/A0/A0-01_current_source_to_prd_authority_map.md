# A0-01 — Current source-to-PRD authority map

- **Phase:** A0
- **Kind:** docs
- **Wave:** W00
- **Complexity:** S
- **Execution model:** LOW
- **Depends on:** none
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO


## Goal

Create a precise map from every v1.4.1 requirement to the existing source seam and mark reuse/new/change/defer.


## Source facts

- v1.4 already has Closed Loop, agentruntime, Git writer, Contract/Policy/Risk/Assurance/Outcome modules.
- L2 found several PRD items that would duplicate existing abstractions.


## Allowed / primary paths

- `docs/tasks/v1.4.1/**`


## Implementation requirements

1. Map every PRD section to package/file owners.
2. List all L2 amendments and their source rationale.
3. No production code changes.


## Acceptance criteria

- [ ] No requirement is left with an ambiguous owner.
- [ ] Duplicate-architecture risks are explicitly marked.


## Verification

- Review against L2_ARCHITECTURE_EVIDENCE.md and DECISION_LOCKS.md.

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
1. **Tests:** Tests: `test/arch/v14_contract_authority_test.go`, `v14_runtime_boundary_test.go`, `v14_outcome_event_test.go`
2. **Contract / Interface:** Contract: `DECISION_LOCKS.md`, `PRD_AMENDMENTS.md`, source `docs/tasks/v1.4/STATUS.md`
3. **Core Implementation:** Core: `cmd/worker/v14.go`, `internal/closedloop/*`, `internal/agentruntime/*`, `internal/contract/*`
4. **Failure Handling:** Failure evidence: stale current docs vs v1.4 final gate; record only, do not alter production code

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
