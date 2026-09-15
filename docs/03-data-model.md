# Compatibility Pointer — Data Model

Current v1.5.0 domain/data-model authority is split intentionally:

- project lifecycle / Active TaskDAG / CompletionContract / authority / audit: `internal/engineering` + `internal/controlplane`;
- task execution contracts: `internal/taskexecution` + existing `internal/agentruntime`;
- immutable candidate and brownfield machine facts: existing `internal/changeset`, `internal/runstore`, `internal/verification`, `internal/evidence`, `internal/outcome`.

See [`architecture/CURRENT_CODEBASE.md`](architecture/CURRENT_CODEBASE.md) and [`implementation/CURRENT_IMPLEMENTATION_STATUS.md`](implementation/CURRENT_IMPLEMENTATION_STATUS.md).

v1.5.0 deliberately keeps a DB-neutral/file compatibility repository for project-control restart state while reusing PostgreSQL fact stores for brownfield execution. Storage unification/redesign is deferred to v1.5.1.
