# A0-02 — Freeze v1.4.1 architecture decision locks: Execution Report

- **TASK:** A0-02
- **STATUS:** DONE
- **Baseline HEAD:** `310d961` (source tree clean at freeze; docs-only delta in progress)
- **Date:** 2026-09-12
- **Execution/Review model:** HIGH / HIGH (per task pack; authority-boundary work)
- **Hidden material:** none read; `docs/tasks/v1.4.1/hidden/` untouched (DL-17)

## 1. Goal and deliverable

Publish the implementation constraints that low-cost agents are forbidden to violate:
lock the existing Closed Loop / runtime / Git / policy / outcome authorities, and mark
unresolved seams as SPIKE_REQUIRED instead of inventing APIs.

Delivered (all in `docs/tasks/v1.4.1/DECISION_LOCKS.md`, +40 lines, no deletions —
the DL-01…DL-20 lock rows themselves are untouched):

1. **Frozen status header** — declares the document FROZEN (task A0-02, 2026-09-12),
   pins the source baseline (working tree at HEAD `310d961`; L2 archive baseline
   `979a55ad…`), and states the authority position of the locks (below the Final PRD
   with its incorporated L2 amendments and approved ADRs; above every task file,
   reference pack, and current source idiom), the change-control rule (locks change
   only by approved architecture decision; an agent that cannot satisfy a lock stops
   as `BLOCKED`), and the tension rule (lock + binding interpretation is the binding
   reading of raw PRD text).
2. **Binding interpretations BI-01 / BI-02** — frozen resolutions of the two known
   PRD-text/lock tensions identified and carried as risks R-POLICY / R-OUTCOME by
   A0-01:
   - BI-01 binds PRD §17 via DL-08 + DL-09 + A-04: no governance lifecycle fields in
     `engineeringpolicy.Policy` YAML/struct, no DB "runtime-active" flag;
     runtime-active = the exact policy blob in the pinned Git snapshot
     (`.tally/policies/*.yaml` via `artifacts.LoadPolicies`).
   - BI-02 binds PRD §27 via DL-12 + A-06: the Outcome domain stays exactly the five
     existing kinds; the PRD §27 lifecycle labels map onto them; `rejected` /
     `escalated` / `integration_failed` remain workflow/attention/integration facts.
3. **Open seams — SPIKE_REQUIRED** — the five seams L2 evidence §7 lists under "do
   not freeze yet" (FrozenIntent path/schema; acceptance-guard placement; DSH
   headless transport; replay storage/query; L3 integration service API), each with
   its owning task (P0-01, P1-06, P4-03 spike-gate-first, P3-04, P5-01) and the rule
   that a task needing an unfrozen seam stops as `BLOCKED` rather than inventing a
   parallel one.
4. **Machine enforcement index** — maps each executable guard at the frozen baseline
   (`test/arch/v14_*` suite, `internal/gitwriter`, `internal/closedloop/ports.go`,
   `internal/verification` holdout-leak tests, migration `0023`) to the locks it
   supports, and names A0-04 as owner of extending the guards for the v1.4.1-specific
   readings (governance-field exclusion, Outcome kind-set stability).

The locks DL-01…DL-20 themselves already existed in the task-pack baseline commit
`3c1e983`; this task freezes them (header + interpretations + seam register +
enforcement map) without altering any lock row.

## 2. Reference order compliance (L3 evidence)

Exact sources/tests read, in the mandated order:

1. **Tests:** `go test ./test/arch/...` executed (see §5); the four guard files named
   by the pack are covered by the run (`v14_runtime_boundary_test.go`,
   `v14_contract_authority_test.go`, `v14_assurance_failsafe_test.go`,
   `v14_outcome_event_test.go`).
2. **Contract/Interface:** `internal/closedloop/ports.go` (single `Executor` port),
   `internal/agentruntime/protocol.go` (single runtime protocol) — as cited by the
   enforcement index (DL-01, DL-02, DL-16).
3. **Core implementation:** `internal/engineeringpolicy/policy.go` (narrow schema;
   `Revision` is content-derived, `yaml:"-"`, no governance status/evidence/approval
   fields — BI-01 evidence), `internal/outcome/types.go` (closed five-kind enum,
   fail-safe unknown kind — BI-02 evidence), `migrations/0023_v14_outcome_event.sql`
   (`event_kind` CHECK constraint + `outcome_event_append_only` trigger — BI-02
   evidence), `internal/gitwriter/writer.go` (expected-base discipline — DL-03).
4. **Failure handling:** the arch suite already rejects trust-core/runtime and
   policy/ExecutionPolicy leakage; no new failure path introduced (docs-only task).

Docs read first per task discipline: `docs/tasks/v1.4.1/DECISION_LOCKS.md` (working
state), `docs/tasks/v1.4.1/PRD_AMENDMENTS.md` (A-01…A-10; BI-01/BI-02 cite A-04 and
A-06 verbatim-consistent), `docs/tasks/v1.4.1/SOURCE_BASELINE.md`, plus
`docs/tasks/v1.4.1/L2_ARCHITECTURE_EVIDENCE.md` §7 ("Do not freeze yet" list — the
open-seams table reproduces it exactly) and the A0-01 authority map (source facts
F1–F17, DL cross-reference §5.1, risks R-POLICY/R-OUTCOME).

**External reference actually used:** none.

**Why this is the narrowest valid seam:** the deliverable is the authority document
itself; the only semantic act is declaring existing locks frozen and recording
already-evidenced interpretations, not designing new architecture. Every
interpretation cites source evidence verified in this tree; every unresolved question
is routed to its owning task instead of being resolved by invention. No production
code, no test, no task file of other tasks, and no PRD text was touched.

## 3. Changed files

- `docs/tasks/v1.4.1/DECISION_LOCKS.md` — frozen header (status/baseline/authority/
  change control/tension rule), BI-01/BI-02, open-seams SPIKE_REQUIRED table,
  machine enforcement index (+40/−0)
- `docs/tasks/v1.4.1/reports/A0-02_freeze_architecture_decision_locks.md` — this report
- `docs/tasks/v1.4.1/tasks/A0/A0-02_freeze_v1_4_1_architecture_decision_locks.md` —
  execution-status addendum appended (A0-03 convention)
- `docs/tasks/v1.4.1/STATUS.md` — implementation progress 0/57 → 3/57

No production code, no test, no `go.mod`/`go.sum`, no shared hotspot from
`PATH_OWNERSHIP.md`, no PRD or other task file modified.

## 4. Semantic changes

- The decision-lock document changed status from an unordered constraint list to a
  frozen authority with explicit precedence, change control, and a PRD-tension rule.
- Two PRD-text/lock tensions (PRD §17 vs DL-08/A-04; PRD §27 vs DL-12/A-06) received
  frozen binding readings, so downstream tasks inherit the resolution instead of
  re-litigating it.
- Five seams are now formally registered as deliberately unfrozen (SPIKE_REQUIRED)
  with owning tasks — closing the "invent an API while spike is open" loophole.
- Lock enforcement became inspectable: a guard→lock index with an explicit
  review-enforced remainder and a named owner (A0-04) for the guard gap.

## 5. Tests/commands executed (exact results)

Windows / Git Bash, Go `go1.27.1 windows/amd64`, working tree = `310d961` + the
docs-only A0-02 delta.

| # | Command | Result |
|---|---|---|
| 1 | `go test ./test/arch/...` | PASS — `ok tally/test/arch (cached)` |
| 2 | `go test -count=1 ./test/arch/...` | PASS — `ok tally/test/arch 0.869s` (fresh run; confirms the machine-enforcement-index claim holds with the workspace delta applied) |
| 3 | `git diff --check` | PASS — clean (no whitespace/line-ending errors) |
| 4 | `grep -L DECISION_LOCKS docs/tasks/v1.4.1/tasks/*/*.md` | PASS — empty; 57/57 task files reference `DECISION_LOCKS` (AC-1 evidence) |
| 5 | Spot-checks of BI evidence: `policy.go` Revision/governance fields; `outcome/types.go` kind enum; `migrations/0023` CHECK/trigger; owning-task files P0-01/P1-06/P3-04/P4-03/P5-01 exist | PASS — all as cited in the frozen text |

No focused Go tests were added: docs-only task, no behavior introduced ("new behavior
needs focused tests" is N/A — no behavior change); the executable guards the document
references already exist and pass.

## 6. Acceptance criteria

- **AC-1 "Decision locks are referenced by every later task template"**: **PASS** —
  every one of the 57 task files under `docs/tasks/v1.4.1/tasks/**` references
  `DECISION_LOCKS` (its Mandatory implementation discipline requires reading it
  first); the frozen header additionally declares the locks binding over every task
  file and reference pack.
- **AC-2 "No lock contradicts current source semantics"**: **PASS** — A0-01 verified
  each DL-01…DL-20 against source facts F1–F17 in this tree on 2026-09-12; this task
  re-verified the BI-01/BI-02 evidence claims directly in source (narrow policy
  schema with content-derived revision; five-kind closed Outcome enum; migration 0023
  CHECK + append-only trigger) and the full arch suite passes at the frozen baseline
  (`ok tally/test/arch 0.869s`, exit 0).

## 7. Baseline failures observed

None. `go test -count=1 ./test/arch/...` passes at the frozen baseline with the
A0-02 docs delta applied; no other validation was in a failing state.

## 8. Remaining risks

- Locks without a machine guard (e.g. DL-06, DL-10, DL-11, DL-13, DL-19 and the
  v1.4.1-specific readings of DL-08/DL-12) remain review-enforced until A0-04 lands
  the guard extensions; the enforcement index states this explicitly so the gap is
  visible, not hidden.
- The frozen baseline hash `310d961` refers to the source tree at freeze time (A0-01/
  A0-03 convention of citing the pre-commit HEAD); the A0-02 commit itself postdates
  the header by necessity and contains only docs.
- Interpretation drift risk if a future task quotes raw PRD §17/§27 without the lock
  header's tension rule; mitigated by BI-01/BI-02 being frozen inside the locks
  document that every task must read first.

## 9. Decision Locks

**DL_TOUCHED: yes — by design, this is the freezing task.** Precisely: the DL-01…DL-20
lock rows were not modified (diff is +40/−0; no lock semantics changed, weakened, or
bypassed); the additions freeze their authority, record their binding interpretations
and enforcement map, and register unfrozen seams. No lock was routed around; nothing
under `docs/tasks/v1.4.1/hidden/` was read (DL-17).

## 10. Known uncertainty / version sensitivity

None material. The document is prose authority; the only executable claim it makes
(the arch suite green at the frozen baseline) was re-executed during this task and is
reproducible via the commands in §5.
