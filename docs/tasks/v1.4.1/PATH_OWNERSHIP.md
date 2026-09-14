# Path Ownership / Hotspot Plan

The purpose of this file is to prevent low-cost parallel agents from colliding on central files.

| Path / semantic hotspot | Primary tasks | Rule |
|---|---|---|
| `internal/web/handlers/v14_launch.go` | P0-04 → P0-08 | SERIAL; P0-08 starts after P0-04 merge |
| `web/templates/v14_launch.tmpl` | P0-04 | Exclusive during P0-04 |
| `cmd/worker/v14.go` | P1-04 → P1-05/P1-06 | P1-04 first; P1-05/P1-06 use helper extraction and coordinate |
| `internal/contract/**` | P1-03; limited P0/P1 helpers | Contract semantics owner is P1 lane |
| `internal/engineeringpolicy/**` | P1/P3 | Runtime schema is frozen; P3 may consume, not extend governance fields |
| `internal/taskstate/**` | P1-06/P1-07 | Acceptance semantic helper must have one owner |
| `cmd/worker/**` runtime composition | P4-01 | Exclusive refactor owner |
| `runtimes/codex/**` | P4-02 | Regression-only unless defect discovered |
| `runtimes/dsh/**` | P4-04/05/06 | One runtime lane, sequential where overlapping |
| `internal/gitwriter/**` | P0-08/P3-07/P5 | Prefer composition; avoid modifying core writer concurrently |
| `internal/policygovernance/**` | P3 | P3 owns this new package |
| `migrations/**` | P3-02 | Migration numbering reserved before parallel work |
| `internal/outcome/**` | P6 | Outcome semantic enum is frozen |
| `web/templates/base.tmpl` | P2-01 | Exclusive navigation owner |
| `README.md`, current docs | P6-07 | Final sync only; earlier tasks write phase-local docs |

## Forbidden parallel edits

- Two tasks may not independently alter `engineeringpolicy.Policy`.
- Two tasks may not independently change Closed Loop verdict semantics.
- No task outside P4 may create a new runtime protocol.
- No task outside P6 may expand Outcome event kinds.
- No UI task may alter task lifecycle transitions simply to simplify rendering.
