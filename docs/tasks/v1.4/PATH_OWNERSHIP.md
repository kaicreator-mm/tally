# Tally v1.4 Path Ownership

## Serialized hotspots

| Path | Primary task(s) | Rule |
|---|---|---|
| `cmd/worker/main.go` | P0-11 -> P2-09 | Never parallelize writes to this file. |
| `internal/workflow/v13.go` | P0-01 | P2 must wrap/reuse it, not rewrite it. |
| `runtimes/codex/**` | P0-02 -> P0-03 | Identity then isolation, sequential. |
| `internal/store/*.sql.go` | P0-06, P3-05 | Generated ownership is serialized after SQL contract changes. |
| `internal/artifacts/**` | P1-01 -> P1-03 | Aggregate facade first, policy loader second. |
| `internal/closedloop/**` | P2 tasks | Follow dependencies; P2-08 is the integration owner. |
| `internal/web/handlers/v14_*.go` | P3-08 -> P3-09 | Launch/task surface first, change/attention second. |
| Root `README/AGENTS/CLAUDE` | P0-12 -> P4-05 | Baseline docs then final closeout only. |

## Forbidden ownership shortcuts

- No task may edit generated sqlc files unless its task explicitly owns generation.
- No UI task may add lifecycle state columns/tables.
- No EngineeringPolicy task may change `ExecutionPolicy` retry/network/shell semantics.
- No closed-loop task may weaken `internal/verification` qualification to make E2E pass.
- No AI Runtime task may move `runtimes/codex` into common inference runtime.
