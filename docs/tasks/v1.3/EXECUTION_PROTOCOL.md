# Low-cost Parallel Execution Protocol

## Unit of work

- One task = one branch/worktree = one implementation agent.
- Start only when every listed dependency is merged into the task base.
- Read only the PRD sections referenced by the task plus `SOURCE_BASELINE.md`, `DECISION_LOCKS.md`, this protocol and the assigned task file.
- Do not read old task plans as product authority.

## Scope discipline

- Change only `Allowed paths`.
- If a required edit is outside scope, return `BLOCKED`; do not widen scope.
- Do not perform unrelated cleanup, package moves, style normalization or dependency upgrades.
- New external dependencies require an explicit task; otherwise do not modify `go.mod`.

## Shared hotspots

Leaf agents MUST NOT touch shared generated/integration hotspots unless their task owns them:

- `internal/store/*.sql.go`, `internal/store/models.go`, `internal/store/querier.go`
- `cmd/worker/main.go`
- `internal/web/handlers/server.go`
- root `CLAUDE.md`, `README.md` and legacy docs

SQL contract tasks edit migration/query files only. `P0-09` and `P1-31` are the only sqlc generation owners.

## Git / merge discipline

- Never merge another task branch from a leaf task.
- Do not resolve cross-task conflicts by redesigning interfaces. Integrator tasks own named integration points.
- Keep commits task-scoped and use the task ID in the commit message when practical.

## Validation

- Run focused validation listed in the task.
- Integration/gate tasks run broader validation.
- Never claim a command passed unless executed.
- Missing DB/Docker/tool/network is `NOT_RUN(reason)` or `BLOCKED`, never PASS.

## Stop conditions

Stop and return BLOCKED when:

- an acceptance criterion requires a product/architecture decision not in PRD or `DECISION_LOCKS.md`;
- required file is outside Allowed paths;
- generated sqlc output would need manual editing;
- a task would require P2+ scope;
- the actual Codex runtime surface contradicts the adapter assumptions.

## Phase rule

P1 starts only after `P0-23` PASS. P2+ is explicitly deferred until `P1-52` Trust Gate is reviewed.
