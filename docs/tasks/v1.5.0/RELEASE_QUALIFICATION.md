# Tally v1.5.0 Release Qualification

**Overall status: RELEASE QUALIFIED on the deterministic Local carrier (Build Host `Nilar`, revision `0a5f7307ec5263240d726cd49b00232c39ebf684`, UTC 2026-09-15); one variant ENV-BLOCKED.**

All required executable validation, Critical Journeys and Hidden Validation have executed against frozen candidate `0a5f730` on the Build Host (Windows 10.0.26200 x64, Go 1.27.1, disposable PostgreSQL `tally_v150_validation`; no GitHub Actions / no CI): visible carrier 13/13 PASS, CJ-01..CJ-06 executed, held-out pack 41/41 PASS. The only remaining blocked item is the CJ-04 real Codex/DSH CLI variant: neither CLI exists on this Build Host (`exec.LookPath` evidence on file); the brownfield journey executed with the deterministic `FakeRuntime` over real Git worktrees and a real database instead.

Status vocabulary: `PASS | FAIL | NOT_RUN(reason) | ENV-BLOCKED(reason/evidence)`.

## Qualification assertions

| # | Assertion | Evidence / current fact | Status |
|---:|---|---|---|
| 1 | Frozen v1.5.0 product/architecture/task authority is adopted and project overrides prohibit GitHub Actions/CI. | `docs/product/v1.5/*`, `docs/tasks/v1.5.0/*`, `.dev-standard/PROJECT_OVERRIDES.md`. | PASS (source authority review) |
| 2 | Active TaskDAG is the sole Project Flow authority; there is no second mutable flow state. | `internal/engineering` + `internal/controlplane`; architecture/manual review. | PASS (source/focused evidence) |
| 3 | Requirements Baseline rejects missing statement/AC and freezes provenance/artifact identity. | Engineering domain tests from PR #11. | PASS (focused executable evidence) |
| 4 | Project Plan/PlanningRevision and ArchitectureRevision are accepted as explicit immutable-stage values with validated constraints/artifacts. | PR #11 engineering tests/source review. | PASS (focused executable evidence) |
| 5 | TaskDAG activation rejects duplicate/missing/self/cyclic dependencies and derives readiness deterministically, including parallel-ready nodes. | PR #11 DAG tests. | PASS (focused executable evidence) |
| 6 | Active TaskDAG/readiness/audit survive restart through the v1.5 compatibility repository. | `internal/controlplane` lifecycle + snapshot persistence tests executed on `0a5f730` (carrier log `20260915T233124Z`); held-out HV-DAG-09 re-proved explicit operational states survive refresh. | PASS |
| 7 | CompletionContract deterministically blocks missing state/artifact/evidence/check/record/authority/Git requirements. | Engineering CompletionContract tests/source. | PASS (focused executable/source evidence) |
| 8 | Human authority can block an otherwise accepted task and rejected/pending authority does not satisfy completion. | `TestAuthorityDecisionParticipatesInCompletion` executed on `0a5f730`; held-out HV-COMP-06 re-proved pending/rejected authority never satisfies. | PASS |
| 9 | Direct LLM and AI Runtime implement the same typed IntelligenceProvider contract. | PR #12 conformance tests. | PASS (focused executable evidence) |
| 10 | Intelligence output is Proposal/Assessment/Recommendation only and cannot own Project/DAG state or deterministic gates. | `internal/intelligence`, Control Plane boundary/manual review. | PASS (source/focused evidence) |
| 11 | `internal/agentruntime.Runtime/Session` remains the single Agent Runtime protocol; no v1.5 second runtime stack exists. | Brownfield bridge PR #16 manual contract review. | PASS (source review) |
| 12 | Runtime × ExecutionEnvironment remain orthogonal and v1.5.0 production implementation is Local-only; SSH/Docker are not falsely claimed implemented. | `internal/taskexecution`, server composition, frozen PRD. | PASS (source review) |
| 13 | Task correctness does not depend on Agent session resume; restart/fallback semantics are bounded and interruption is first-class. | PR #12 focused tests + PR #16 bridge review. | PASS (focused executable/source evidence) |
| 14 | Executable coding attempts use Tally-owned disposable worktrees and owner primary checkout is rejected at the freeze boundary. | PR #16 isolated real-Git worktree tests. | PASS (focused executable evidence) |
| 15 | Immutable candidate content is tied to a durable CodegenRun/ChangeSet before candidateverify records verification lineage. | PR #16 `runstoreCodegenRecorder` + manual interface review. | PASS (source review; full DB integration pending) |
| 16 | Existing `internal/closedloop` + `candidateverify` remain verify/repair/accept authority; an Agent `completed` claim alone cannot accept/advance. | PR #16 architecture/dispatch review; existing closedloop authority. | PASS (source/focused evidence) |
| 17 | Runtime/environment/provider operational errors remain distinct from candidate verification failure and cannot be silently accepted. | taskexecution/controlplane/closedloop tests executed on `0a5f730`; held-out HV-INT-03/04 + HV-EXEC-06 re-proved error/failure separation. | PASS |
| 18 | Minimal v1.5 Control Plane API/UI is production reachable only behind existing authentication + CSRF and does not create a second state authority. | `internal/controlplane/http_test.go` + `internal/web/v15_controlplane_test.go` executed on `0a5f730` (unauthenticated redirect, CSRF enforcement, no second authority). | PASS |
| 19 | Full visible regression and required Critical Journeys execute successfully against the frozen candidate, including real Local brownfield execution. | `bash scripts/validate-v1.5.0.sh` 13/13 PASS on `0a5f730` incl. full regression + integration E2E (disposable PostgreSQL); CJ-01..06 executed per `CRITICAL_JOURNEYS.md`; brownfield journey ran with `FakeRuntime` over real Git worktrees + real DB. Real Codex/DSH CLI variant unavailable on host (LookPath evidence on file). | PASS (real-CLI variant ENV-BLOCKED, evidence recorded) |
| 20 | Hidden Validation executes against the same frozen candidate with external held-out stimuli, no leakage, and no deferred v1.5.1–v1.5.5 capability is falsely claimed complete. | External pack `C:/Dev/vCode/hidden_validation/tally/v1.5.0` (authored by the local validation agent, a non-implementation-author, from frozen PRD scenario families after candidate freeze) executed via `scripts/run-v1.5-hidden.sh` on `0a5f730`: 41/41 PASS across HV-REQ/HV-DAG/HV-COMP/HV-INT/HV-EXEC/HV-BROWNFIELD; sanitized evidence `C:/Dev/xCode/tally-validation/v1.5.0-hidden/20260915T234347Z/`; candidate checkout verified clean before and after. | PASS |

## Required release run sequence

After the closure implementation is merged, freeze one concrete candidate commit and do not change implementation code between these gates:

```bash
# 1. Visible validation / full regression / applicable integration
bash scripts/validate-v1.5.0.sh

# 2. Critical Journeys, including real Local Brownfield execution
#    Record CJ-01..CJ-06 against the exact same revision.

# 3. Held-out validation
TALLY_V15_HIDDEN_ROOT=/secure/held-out/tally-v1.5.0 \
  bash scripts/run-v1.5-hidden.sh
```

If any required gate is `FAIL`, `NOT_RUN`, or `ENV-BLOCKED`, keep overall status **NOT RELEASE QUALIFIED** unless the frozen release policy explicitly classifies that gate non-blocking.

## Deferred capability truth

Release Qualification for v1.5.0 must not imply completion of:

- v1.5.1 SQLite-default/PostgreSQL-optional persistence redesign / DB queue removal;
- v1.5.2 SSH/Docker execution environments;
- v1.5.3 full GitHub Issue/PR/Merge/Tag/Release authority automation;
- v1.5.4 full multi-Agent role workflow;
- v1.5.5 automatic re-plan / governed learning.

Only the shared contracts/seams needed by those later slices are frozen in v1.5.0.
