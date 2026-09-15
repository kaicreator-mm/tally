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

The closure branch adds `scripts/validate-v1.5.0.sh` as the explicit carrier. The following must be executed on a repository checkout of the frozen candidate. At the time this document was authored, ChatGPT Web does not have an authenticated local checkout of the private repository, so these remain blocked rather than inherited from earlier revisions.

| Gate | Command | Current status |
|---|---|---|
| Engineering domain | `go test ./internal/engineering/...` | ENV-BLOCKED(private repo checkout unavailable in current execution environment) |
| Intelligence + execution | `go test ./internal/intelligence/... ./internal/taskexecution/...` | ENV-BLOCKED(private repo checkout unavailable) |
| Control Plane + UI/API + Critical Journey | `go test ./internal/controlplane/... ./internal/web/...` | ENV-BLOCKED(private repo checkout unavailable) |
| Brownfield safety | `go test ./internal/changeset/... ./internal/workspace/... ./internal/closedloop/... ./internal/workflow/...` | ENV-BLOCKED(private repo checkout unavailable) |
| Runtime adapters | `go test ./runtimes/...` | ENV-BLOCKED(private repo checkout / runtime dependencies unavailable) |
| Runtime conformance | `go test ./test/runtimeconformance/...` | ENV-BLOCKED(private repo checkout unavailable) |
| Architecture tests | `go test ./test/arch/...` | ENV-BLOCKED(private repo checkout unavailable) |
| Build | `go build ./...` | ENV-BLOCKED(private repo checkout unavailable) |
| Vet | `go vet ./...` | ENV-BLOCKED(private repo checkout unavailable) |
| Focused race | `go test -race ./internal/engineering/... ./internal/intelligence/... ./internal/taskexecution/... ./internal/controlplane/...` | ENV-BLOCKED(private repo checkout unavailable) |
| Full visible regression | `go test ./...` | ENV-BLOCKED(private repo checkout unavailable) |
| Integration/E2E | `go test -tags=integration ./test/e2e/...` | ENV-BLOCKED(Build Host + reachable PostgreSQL not available here) |

## Required Build Host run

On the frozen candidate:

```bash
bash scripts/validate-v1.5.0.sh
```

The generated `summary.tsv` and `validation.log` are the evidence. Do not translate missing commands into PASS. If a required executable gate is `FAIL` or `ENV-BLOCKED`, Release Qualification remains blocked.

## Closure interpretation

Current implementation work can be merged as a development baseline when manually reviewed and locally focused evidence is adequate for the PR. **v1.5.0 may not be marked release-qualified until the required visible commands, Critical Journeys and Hidden Validation execute against the same frozen candidate with acceptable results.**
