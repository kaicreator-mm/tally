# Path Ownership / Conflict Matrix

## Serialized hotspots

| Path | Exclusive owner task(s) | Rule |
| --- | --- | --- |
| `internal/store/*.sql.go`, `models.go`, `querier.go` | P0-09 then P1-31 | sqlc-generated only |
| `cmd/worker/main.go` | P0-21 then P1-43 | never parallel inside a phase |
| `internal/web/handlers/server.go` | P1-44 | P1 UI integration only |
| `CLAUDE.md`, root docs | P1-46 | update after behavior is shipped |
| `internal/exec/exec.go` | P0-13/P0-20 in ordered integration | no unrelated refactor |

## Parallel-safe new bounded areas

P1 intentionally creates independent package/file ownership for TaskContract, Verification, Evidence, ChangeSet, Runtime Protocol, ContextPack, Review, Measurement, Approval, Policy and Holdout. Do not collapse these into one `service.go` during implementation.
