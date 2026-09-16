# Tally v1.5.0 Validation Matrix

This is the executable-evidence ledger for Issue #9. It follows the v1.5.0 project override: **Local / Build Host commands only; no GitHub Actions and no CI systems**.

Status vocabulary is exact:

- `PASS` — the stated command/check actually executed successfully against the stated revision/environment.
- `FAIL` — it executed and failed.
- `NOT_RUN(reason)` — intentionally not executed for a stated non-environment reason.
- `ENV-BLOCKED(reason/evidence)` — required execution could not run because the environment/dependency was unavailable.

A PR-level or focused `PASS` is not a Release PASS.

## Evidence already executed

| Area | Revision / evidence | Environment | Command / check | Status | Notes |
|---|---|---|---|---|---|
| Engineering domain / TaskDAG / CompletionContract | PR #11, head `338584e20a1e987e1688605575bab132f3143213` before merge | ChatGPT Web isolated Go module | `go test ./internal/engineering/...` | PASS | Executed in isolated standard-library slice matching committed package. |
| Engineering domain static checks | PR #11 | same | `go vet ./internal/engineering/...` | PASS | Focused only. |
| Engineering domain race | PR #11 | same | `go test -race ./internal/engineering/...` | PASS | Focused only. |
| IntelligenceProvider + task execution contracts | PR #12, head `a6ab8358bb09d2850952b60f1f6f10c080fdefc2` | ChatGPT Web isolated Go module with mirrored current `agentruntime` public contract | `go test ./internal/intelligence/... ./internal/taskexecution/...` | PASS | Full private repository was not checked out. |
| Intelligence/task execution vet | PR #12 | same | `go vet ./internal/intelligence/... ./internal/taskexecution/...` | PASS | Focused only. |
| Intelligence/task execution race | PR #12 | same | `go test -race ./internal/intelligence/... ./internal/taskexecution/...` | PASS | Focused only. |
| Immutable worktree candidate freeze | PR #16 during review | ChatGPT Web isolated Go harness using real `git worktree add --detach` | isolated `go test ./...` | PASS | Covers modified + untracked, no-change => no candidate, deletion fail-closed, owner checkout rejection. |
| Worktree freeze vet | PR #16 | same | isolated `go vet ./...` | PASS | Focused harness only. |
| Worktree freeze race | PR #16 | same | isolated `go test -race ./...` | PASS | Focused harness only. |
| PR #16 manual contract review | merged as `83ca742f11f40aedb8dfa9bfff41e7d78b5e1fb5` | GitHub source review | review of `AgentRunResult`, `ChangeSet`, `closedloop.LoopConfig`, runstore and Control Plane seams | PASS | Static/source evidence, not executable full regression. |

## v1.5.0 closure candidate — required visible execution

The closure branch adds `scripts/validate-v1.5.0.sh` as the explicit carrier. The following was executed on a repository checkout of the frozen candidate on the Build Host.

**Execution record:** revision `0a5f7307ec5263240d726cd49b00232c39ebf684` · host `Nilar` (Windows 10.0.26200 x64) · Go 1.27.1 windows/amd64 · carrier `bash scripts/validate-v1.5.0.sh` with `DATABASE_URL=postgres://postgres:postgres@localhost:15432/tally_v150_validation?sslmode=disable` (disposable DB) · UTC `2026-09-15T23:31:24Z` → `2026-09-15T23:41Z` · evidence outside repo: `C:/Dev/xCode/tally-validation/v1.5.0/20260915T233124Z/{validation.log,summary.tsv}` · **no GitHub Actions / no CI**.

| Gate | Command | Status |
|---|---|---|
| Engineering domain | `go test ./internal/engineering/...` | PASS |
| Intelligence + execution | `go test ./internal/intelligence/... ./internal/taskexecution/...` | PASS |
| Control Plane + UI/API + Critical Journey | `go test ./internal/controlplane/...` + `go test ./internal/web/...` | PASS |
| Brownfield safety | `go test ./internal/changeset/... ./internal/workspace/... ./internal/closedloop/... ./internal/workflow/...` | PASS |
| Runtime adapters | `go test ./runtimes/...` | PASS |
| Runtime conformance | `go test ./test/runtimeconformance/...` | PASS |
| Architecture tests | `go test ./test/arch/...` | PASS |
| Build | `go build ./...` | PASS |
| Vet | `go vet ./...` | PASS |
| Focused race | `go test -race ./internal/engineering/... ./internal/intelligence/... ./internal/taskexecution/... ./internal/controlplane/...` | PASS |
| Full visible regression | `go test ./...` | PASS |
| Integration/E2E | `go test -tags=integration ./test/e2e/...` | PASS (disposable PostgreSQL `tally_v150_validation`) |
| Hidden Validation | `TALLY_V15_HIDDEN_ROOT=C:/Dev/vCode/hidden_validation/tally/v1.5.0 bash scripts/run-v1.5-hidden.sh` | PASS — 41/41 held-out scenarios (HV-REQ/HV-DAG/HV-COMP/HV-INT/HV-EXEC/HV-BROWNFIELD), same revision, UTC `2026-09-15T23:43:47Z`, sanitized evidence `C:/Dev/xCode/tally-validation/v1.5.0-hidden/20260915T234347Z/` |
| Real Codex/DSH CLI runtime journey variant (CJ-04 variant) | `exec.LookPath("codex"|"dsh")` availability probe | ENV-BLOCKED — no `codex` or `dsh` CLI on this Build Host; evidence `C:/Dev/xCode/tally-validation/v1.5.0/real-runtime-availability.txt` (`executable file not found in %PATH%`, UTC 2026-09-15T23:44:53Z). Deterministic fake-runtime carriers (real Git worktrees + real DB) executed instead. |

## Required Build Host run

On the frozen candidate:

```bash
bash scripts/validate-v1.5.0.sh
```

The generated `summary.tsv` and `validation.log` are the evidence. Do not translate missing commands into PASS. If a required executable gate is `FAIL` or `ENV-BLOCKED`, Release Qualification remains blocked.

## Closure interpretation

Current implementation work can be merged as a development baseline when manually reviewed and locally focused evidence is adequate for the PR. **v1.5.0 may not be marked release-qualified until the required visible commands, Critical Journeys and Hidden Validation execute against the same frozen candidate with acceptable results.**
