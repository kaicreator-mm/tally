# Development Guide

## Prerequisites

- Go 1.23+
- PostgreSQL
- git CLI
- `sqlc` and `golangci-lint` when using their Make targets
- optional Docker/SSH/deployment tools only for paths that need them

## Common commands

```bash
make migrate
make build
make test
make test-integration
make check
make sqlc
make reindex
```

`make dev-db` currently references a `docker-compose.dev.yml` that is not present in the supplied source. Provision PostgreSQL yourself instead of assuming that target works.

## Current task program

```text
docs/tasks/v1.4/
```

Read:

1. `DECISION_LOCKS.md`
2. `TASK_DAG.md`
3. `PATH_OWNERSHIP.md`
4. assigned task
5. L3 Reference Pack if the task has one

A0 architecture spikes execute before downstream implementation assumptions are treated as frozen.

## Brownfield rule

The source already has the v1.3 Trust Core.

Do not implement v1.4 by duplicating it under new package names.

Prefer small aggregation/orchestration seams over broad physical migrations.

## Task completion

A task may need to prove:

```text
IMPLEMENTED
→ WIRED
→ TESTED
→ PRODUCTION_REACHABLE
→ E2E_PROVEN
```

A pure domain/library task can mark wiring/E2E `N/A(reason)` if a later owned integration task is explicit.

## Integration tests

Integration tests require a reachable PostgreSQL `DATABASE_URL`.

The existing test helpers create isolated schemas.

## Line endings

If Git reports mass modifications, first inspect:

```bash
git diff --ignore-space-at-eol
```

The supplied baseline had CRLF/LF-only worktree noise.

Do not commit hundreds of line-ending-only changes as part of a feature task.
