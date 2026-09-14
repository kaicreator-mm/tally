# A0-03 — Validation and dependency preflight: Execution Report

- **TASK:** A0-03
- **STATUS:** DONE
- **Baseline HEAD:** `3c1e983d0305c51a8ed9fbb3a8c864e81e8a31da` (clean tree at start)
- **Date:** 2026-09-12 (runs executed 2026-09-11/12 UTC per JSON `generated_at`)
- **Execution/Review model:** LOW / LOW (per task pack; no source or API mismatch encountered)

## 1. Goal and deliverable

Provide a deterministic preflight that distinguishes dependency/network failures from
source test failures.

Delivered: `scripts/preflight/` — a stdlib-only Go preflight tool (no new `go.mod`
dependency, INV-6 whitelist untouched) that runs staged checks and emits a
machine-readable `tally.preflight.v1` JSON report on stdout plus human progress on
stderr. Exit codes carry the verdict for CI use:

| Exit | Status | Meaning | `is_code_regression` |
|---|---|---|---|
| 0 | `BASELINE_OK` | all executed gating checks passed | `false` |
| 1 | `SOURCE_FAILURE` | compile / vet / test failure **after** modules confirmed available | `true` |
| 2 | `DEPENDENCY_BLOCKED` | module download/network/state failure, or unclassified module-stage failure (fail-safe) | `false` |
| 3 | `TOOLCHAIN_BLOCKED` | go missing or older than `go.mod` directive | `false` |

Usage: `go run ./scripts/preflight [-integration] [-timeout 20m]`.
`-integration` additionally requires `DATABASE_URL` and runs
`go run ./cmd/tally migrate` + `go test -tags=integration ./...`.

## 2. Why the classification is deterministic (AC-1)

The tool stages its checks and stops at the first gating failure:
`toolchain → modules (go list -deps -test ./...) → build → vet → unit (arch + internal)
→ integration (opt-in)`. A failure is attributed to the stage where it occurred;
later stages never run, so their outcomes cannot be conflated.

Within the module stage, stderr is classified by precision-ordered vocabularies
(`ClassifyModuleStageError` in `scripts/preflight/preflight.go`):

1. dependency-state vocabulary first (`missing go.sum entry`, `checksum mismatch`,
   `go.mod requires go >=`, ...) — these can arrive **with** a `file.go:3:8:` prefix
   (measured on go1.27.1, see negative tests), so a bare position prefix must not
   claim "source error";
2. network vocabulary (`module lookup disabled`, `GOPROXY=off`, `dial tcp`,
   `no such host`, `404/410`, `Get "http(s)://`, ...) — download failures also embed
   the importer's position prefix;
3. source verdict requires **both** a `.go:<line>` diagnostic position **and**
   compiler vocabulary (`syntax error`, `undefined:`, `declared and not used`, ...);
4. anything else is `UNKNOWN` → fail-safe: `DEPENDENCY_BLOCKED`, flagged for human
   review — never `PASS`, never regression without evidence.

`StatusForModuleStage` maps only `SOURCE` to `SOURCE_FAILURE`; everything else in
the module stage is `DEPENDENCY_BLOCKED` with `is_code_regression=false`. This
directly encodes the acceptance criterion: *offline dependency failure is
classified, not reported as code regression*.

## 3. Reference order compliance (L3 evidence)

Exact sources/tests read, in the mandated order:

1. **Tests:** `test/arch/deps_test.go` (INV-6 dependency whitelist guard — confirms
   adding no direct dependency); focused package tests under the packages named by
   the Makefile `test` target and `docs/tasks/v1.4.1/SOURCE_BASELINE.md`.
2. **Contract/Interface:** `Makefile` targets `test`, `test-integration`, `check`,
   `lint`, `migrate` — the preflight unit baseline replicates `make test` exactly
   (`go test ./test/arch/...` then `go test ./internal/...`); the integration stage
   replicates `test-integration` (`go run ./cmd/tally migrate` +
   `go test -tags=integration ./...`).
3. **Core implementation:** `go.mod` dependency whitelist comment (lines 5–6;
   whitelist lives in `docs/04-code-standards.md` guarded by `deps_test.go`);
   `.github/workflows/reindex-check.yml` and `trailer-check.yml` (existing CI shape:
   self-hosted DB-gated job and PR trailer gate; preflight complements, not replaces).
4. **Failure handling:** `SOURCE_BASELINE.md` records the exact historical hazard —
   the v1.4 preflight was reported as
   `BASELINE_NOT_EXECUTED_NETWORK_BLOCKED`, "not a source regression verdict". The
   tool makes that distinction mechanical instead of prose.

Docs read first per task discipline: `docs/tasks/v1.4.1/DECISION_LOCKS.md`,
`docs/tasks/v1.4.1/PRD_AMENDMENTS.md`, `docs/tasks/v1.4.1/SOURCE_BASELINE.md`,
`docs/tasks/v1.4.1/PATH_OWNERSHIP.md`.

**External reference actually used:** EXT-UBER-GO-TEST (Uber Go Style Guide,
table-driven tests) — pattern only, for the classification matrix in
`scripts/preflight/preflight_test.go`. No external code copied.

**Why this is the narrowest valid seam:** the task's Allowed Paths are `scripts/**`
and `docs/tasks/v1.4.1/**`; the Makefile (repo root) is outside them, so
`scripts/preflight` is invoked via `go run ./scripts/preflight` and requires no
Makefile/CI edits. No production package, no new dependency, no interface added to
any internal package — the tool is a leaf `package main` under `scripts/`, invisible
to `make test`/`make check` semantics.

## 4. Changed files

- `scripts/preflight/preflight.go` — types, status/exit mapping, error classifier, command runner
- `scripts/preflight/main.go` — staged orchestration, JSON emission
- `scripts/preflight/preflight_test.go` — table-driven tests + two end-to-end failure-path tests
- `docs/tasks/v1.4.1/reports/A0-03_validation_and_dependency_preflight.md` — this report
- `docs/tasks/v1.4.1/tasks/A0/A0-03_validation_and_dependency_preflight.md` — status addendum appended

No production code, no `go.mod`/`go.sum`, no Makefile, no CI workflow, no shared
hotspot from `PATH_OWNERSHIP.md` was modified.

## 5. Validation executed (exact commands and results)

All runs on Windows / Git Bash, HEAD `3c1e983`, Go `go1.27.1`.

| # | Command | Result |
|---|---|---|
| 1 | `go build ./...` | PASS |
| 2 | `go vet ./scripts/preflight/...` | PASS |
| 3 | `gofmt -l scripts/preflight/` | PASS (no output; `scripts/v14_trial/tally/rows.go` was already unformatted before this task — pre-existing, not touched) |
| 4 | `go test ./scripts/preflight/...` | PASS (7 tests incl. 2 end-to-end failure-path tests) |
| 5 | `go run ./scripts/preflight` (connected/cached) | PASS — exit 0, `BASELINE_OK` |
| 6 | `GOPROXY=off GOSUMDB=off go run ./scripts/preflight` (deliberately offline, warm cache) | PASS — exit 0, `BASELINE_OK`; proves the module cache is complete and the baseline runs fully offline |
| 7 | `DATABASE_URL='postgres://postgres:test@localhost:5433/postgres?sslmode=disable' go run ./cmd/tally migrate` | PASS — 17 migrations applied |
| 8 | `DATABASE_URL='...5433...' go run ./scripts/preflight -integration` | PASS — exit 0, `BASELINE_OK` incl. `integration_migrate` + `integration_test` (`go test -tags=integration ./...`, 5m14.489s first run uncached) |
| 9 | `docker run -d --name tally-v141-pg -p 5433:5432 -e POSTGRES_PASSWORD=test postgres:16` + `docker rm -f tally-v141-pg` | PASS — container removed after use; port 5432 and all pre-existing containers untouched |

Task-required verification "run in one connected and one deliberately offline
environment": satisfied by runs 5/6 (repo-level offline) plus the package tests
(cold-cache offline classification, environment-independent).

### Machine-readable evidence — connected/cached run (run 5, exit 0)

```json
{
  "schema": "tally.preflight.v1",
  "generated_at": "2026-09-11T16:55:57Z",
  "go_version": "go1.27.1",
  "go_mod_directive": "1.23",
  "status": "BASELINE_OK",
  "is_code_regression": false,
  "checks": [
    {"name": "toolchain", "status": "PASS", "gating": true, "detail": "go go1.27.1 satisfies go.mod directive 1.23; optional tools: git=found docker=found golangci-lint=found"},
    {"name": "tools", "status": "PASS", "gating": false, "detail": "informational only; required-for-baseline tool is go itself"},
    {"name": "modules", "status": "PASS", "gating": true, "detail": "all module + test dependencies resolvable", "command": "...go.exe list -deps -test ./...", "elapsed": "1.5s"},
    {"name": "build", "status": "PASS", "gating": true, "command": "...go.exe build ./...", "elapsed": "3.03s"},
    {"name": "vet", "status": "PASS", "gating": true, "command": "...go.exe vet ./...", "elapsed": "1.698s"},
    {"name": "unit_arch", "status": "PASS", "gating": true, "detail": "ok  \ttally/test/arch\t0.968s", "command": "...go.exe test ./test/arch/...", "elapsed": "3.98s"},
    {"name": "unit_internal", "status": "PASS", "gating": true, "detail": "ok  \ttally/internal/workspace\t19.417s", "command": "...go.exe test ./internal/...", "elapsed": "53.85s"},
    {"name": "integration", "status": "SKIP", "gating": false, "detail": "NOT_RUN(reason: -integration not requested; DB-gated baseline is opt-in)"}
  ]
}
```

### Per-check summary — offline run (run 6) and integration run (run 8)

| Check | Offline (GOPROXY=off) | Integration (`-integration`, DB @5433) |
|---|---|---|
| toolchain | PASS | PASS |
| tools | PASS (non-gating) | PASS (non-gating) |
| modules | PASS | PASS (1.536s) |
| build | PASS | PASS (3.213s) |
| vet | PASS | PASS (1.811s) |
| unit_arch | PASS (cached) | PASS (1.059s) |
| unit_internal | PASS (cached) | PASS (21.337s) |
| integration_migrate | SKIP (not requested) | PASS (2.614s) |
| integration_test | SKIP (not requested) | PASS (33.893s; first uncached run 5m14.489s) |
| **overall** | **`BASELINE_OK`, exit 0** | **`BASELINE_OK`, exit 0** |

### Failure-path / negative tests (AC-1 evidence)

`go test ./scripts/preflight/... -v` key results:

- `TestClassifyModuleStageError` (13 cases): GOPROXY-off, dial-tcp, DNS, 410-Gone,
  go1.27-observed download-error-with-position-prefix → `DEPENDENCY_NETWORK`;
  missing-go.sum (with and without `main.go:3:8:` prefix), checksum mismatch,
  newer-toolchain-required → `DEPENDENCY_STATE`; unix/windows syntax and undefined
  symbol diagnostics → `SOURCE`; unrecognized/empty → `UNKNOWN`.
- `TestModuleStageFailureNeverReportsNetworkOrStateAsRegression`: network/state/unknown
  → `DEPENDENCY_BLOCKED` + `is_code_regression=false`; only `SOURCE` → `SOURCE_FAILURE`.
- `TestExitCodeFor`, `TestVersionAtLeast` (incl. fail-safe on unparseable toolchain
  string), `TestParseGoModDirective`, `TestHasGoFileLocation`.
- **End-to-end negative test 1** `TestModulesStageOfflineDependencyFailure`: temp
  module requiring a nonexistent module, `GOPROXY=http://127.0.0.1:9`
  (locally refused → fully offline, no external calls), empty `GOMODCACHE`,
  `GOFLAGS=-mod=mod` → download failure classified `DEPENDENCY_NETWORK`,
  overall `DEPENDENCY_BLOCKED`, not a regression. PASS.
- **End-to-end negative test 2** `TestModulesStageMissingGoSumEntryIsDependencyState`:
  read-only mode with missing `go.sum` entry reproduces go1.27's real message
  `main.go:3:8: missing go.sum entry ...` → classified `DEPENDENCY_STATE`,
  overall `DEPENDENCY_BLOCKED`, not a regression. PASS.

Both end-to-end tests existed in first draft form and **failed against the first
classifier draft** (which ordered the `.go:` position heuristic before dependency
vocabularies) — they caught two real misclassification shapes and drove the final
precision-ordered design. That is the negative-path test doing its job.

## 6. Acceptance criteria

- **AC-1 "Offline dependency failure is classified, not reported as code regression"**: **PASS**
  - Offline cold-cache dependency failure → `DEPENDENCY_NETWORK` /
    `DEPENDENCY_STATE` → overall `DEPENDENCY_BLOCKED`, exit 2, `is_code_regression=false`
    (proven by both end-to-end negative tests).
  - Offline warm-cache repo run → full baseline runs and passes (run 6), showing
    offline per se is not failure when the dependency set is complete.
  - `SOURCE_FAILURE` (exit 1, `is_code_regression=true`) is only reachable after
    `modules` PASS, i.e. after dependency availability is established.

## 7. Baseline vs introduced failures

- **Baseline failures observed at HEAD `3c1e983`:** none. `go build ./...`,
  `go vet ./...`, `go test ./test/arch/...`, `go test ./internal/...` (53.85s full
  uncached) and `go test -tags=integration ./...` (DB @5433) all PASS. This extends
  the coordinator's earlier 5-package run to the full unit + integration suites and
  supersedes the `BASELINE_NOT_EXECUTED_NETWORK_BLOCKED` state recorded in
  `SOURCE_BASELINE.md`.
- **Introduced failures:** none outstanding. Two defects found and fixed during
  development of the tool itself (see §8), both before any repo-baseline claim.

## 8. Known uncertainties / version sensitivity

- Error-message vocabulary is Go-version sensitive: go1.27.1 surfaces missing
  dependencies as `missing go.sum entry` (with importer position prefix) rather than
  older `module lookup disabled by GOPROXY=off` wording. The classifier covers both;
  unmatched future wording degrades to `UNKNOWN` → `DEPENDENCY_BLOCKED` (fail-safe,
  never a false regression and never a false PASS).
- Mid-run DB outage during the integration stage would be misattributed to source;
  this is documented in the JSON detail of `integration_test`. DB reachability is
  established by the migrate step first, which bounds the window.
- `golangci-lint`/`docker`/`git` availability is recorded informationally
  (non-gating); the Makefile `check` target's lint step remains a separate concern.
- Real-codex/DeepSeek e2e tests (`test/e2e/v14_codex_trust/**`) skip when
  `TALLY_E2E_CODEX_AUTH_FILE` is unset; credentials were intentionally not provided
  to the preflight run (LLM live paths are out of dependency-preflight scope, covered
  by P4 tasks). Their SKIP is included in the integration suite's exit 0.

## 9. Decision Locks

**DL_TOUCHED: no.** No Decision Lock was modified or bypassed. DL-17 respected:
nothing under `docs/tasks/v1.4.1/hidden/` was read, listed, or summarized. No new
dependency (INV-6 untouched), no production package touched, no protocol/runtime
change (DL-01/02/18 not implicated).

## 10. Remaining risks

- The unit baseline includes a known-slow package (`internal/workspace`, ~19s);
  full preflight ≈ 1–2 min unit / ≈ 6 min with integration. Acceptable for a
  pre-implementation gate; `-timeout` flag bounds runaway runs.
- Report generation date sensitivity: none — all evidence is embedded above with
  exact commands; re-runnable by any agent via the same commands.
