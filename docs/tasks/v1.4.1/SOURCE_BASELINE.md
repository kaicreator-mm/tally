# Source Baseline

- Repository: Tally
- Archive: `tally-0911.7z`
- Git HEAD: `979a55ad3ba5e514bb80f0e012aa36e0e6577eba`
- v1.4 task status found in repository: `55/55 DONE`
- v1.4 Final Gate: `PASS_WITH_KNOWN_LIMITATIONS`
- v1.4.1 implementation: **NOT STARTED by this task pack**

## Key existing seams

- `internal/closedloop/**`
- `internal/agentruntime/**`
- `runtimes/codex/**`
- `internal/workspace/**`
- `internal/gitwriter/**`
- `internal/changeset/**`
- `internal/contract/**`
- `internal/engineeringpolicy/**`
- `internal/risk/**`
- `internal/assurance/**`
- `internal/taskstate/**`
- `internal/outcome/**`
- `internal/intake/**`
- `internal/web/handlers/v14_launch.go`

## Known production gaps verified during L2

1. `contract.Bind()` has no production worker call site.
2. `V14ConfirmDraft` does not Freeze TaskContract/VerificationContract.
3. production Intent AI executor injection is incomplete.
4. candidate intake risk signals do not survive into authoritative frozen execution inputs.
5. `VerifiedBasis` can express limited assurance, but current Closed Loop acceptance is driven by base verification status.
6. Outcome domain/store exists but production lifecycle writers are incomplete.
7. repository export currently includes local/runtime artifacts when the entire worktree is archived.

## Validation preflight

Attempted:

```bash
go test ./internal/contract ./internal/engineeringpolicy ./internal/intake ./internal/closedloop ./internal/agentruntime
```

Result: dependency download blocked by analysis-environment network/DNS access to Go module proxy.

This is `BASELINE_NOT_EXECUTED_NETWORK_BLOCKED`, not a source regression verdict.

Before implementation:
- run the focused test command in connected/cached CI;
- record exact baseline;
- do not start production changes if baseline failures are unexplained.
