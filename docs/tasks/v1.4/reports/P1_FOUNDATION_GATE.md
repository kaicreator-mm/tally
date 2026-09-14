# P1_FOUNDATION_GATE — Contract/Policy/Assurance Foundation Gate (P1-10)

**Date:** 2026-09-09 (Beijing), executed by coordinator
**Task:** `docs/tasks/v1.4/P1/P1-10.md`
**Verdict:** **PASS — v1.4 foundation is deterministic and v1.3-compatible; P2 (Closed Repair Loop) may proceed.**

## Validation (fresh, executed this gate)

```text
go test -count=1 ./internal/contract/... ./internal/engineeringpolicy/... ./internal/risk/... ./internal/assurance/... ./internal/taskstate/...   PASS (5/5 ok)
go test -count=1 ./test/arch/...                                                                                                                  PASS
go test -count=1 -tags=integration ./test/e2e/...                                                                                                 PASS (exit 0; v13 regression + v14 real-codex suites; Postgres via Docker, real codex CLI via TALLY_E2E_CODEX_AUTH_FILE)
```

## Acceptance criteria

- **AC1 (all P1 acceptance checked): PASS** — P1-01 `c680ff9`, P1-02 `7bfa191`, P1-03 `971c728`, P1-04 `783b32a`, P1-05 `17df908`, P1-06 `1bd8d9e`, P1-07 `ffc8833`, P1-08 `22d9996`, P1-09 `46fa186`+`b00b28b`: each subagent report's AC table was coordinator-reviewed, tests re-run fresh at commit time, and commits are task-scoped.
- **AC2 (v1.3 E2E remains green): PASS** — `test/e2e` v13 suites (standard+holdout, holdout-fail, exposure, changeset-supersession) green in the same run as the v14 suites.
- **AC3 (no physical contract migration): PASS** — `internal/taskcontract`/`internal/verification` untouched by all P1 tasks (verified per-task via git status scope checks; v1.4 aggregate lives in `internal/contract` as a new seam; P1-09 pins `TestV14ContractAggregateExposesUnderlyingAuthorityUnchanged`).
- **AC4 (no new mutable lifecycle state): PASS** — `TestV14LifecycleStateHasNoWriteAuthority` + `TestV14NoSecondStateAuthorityTable` (no state-shaped migration/table/query path; VerifiedBasis produced only by `Derive`); the v13 allowlist extension (`46fa186`, `b00b28b`) admits only pure-domain derivation seams per Decision Lock 12 — the no-persistence/no-write-channel invariant itself is unchanged and double-pinned by `TestV14TaskstateStaysPureDerivation`.

## Incidents / decisions during the gate

1. `TestV13StateHasNoDirectWritePath` failed after P1-08 (its v1.3-era import allowlist predated the Decision-Lock-12 derivation seams). Fixed by extending the allowlist to the pure-domain seams (contract/assurance/engineeringpolicy/risk + test-fixture-only taskcontract) with a pointer to the v1.4 pin — an authorized refinement, not a weakening; the write-path and persistence prohibitions are intact and machine-checked.
2. Process note recorded honestly: one commit (`46fa186`) landed before its final test rerun due to a piped exit-code being swallowed in the coordinator's shell; the failure was caught immediately and fixed in `b00b28b`; final suite state verified green with explicit exit code.

## Decision

**Gate PASS. P2 (Closed Repair Loop: P2-01/02/03 wave) is unblocked.**
