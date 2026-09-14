# v1.5.0 Path Ownership

One concern, one PR. Shared hotspots require the integration task owner.

| Task lane | Primary allowed paths | Shared/hotspot rule |
|---|---|---|
| T150-00 authority | `.dev-standard/**`, `AGENTS.md`, `CLAUDE.md`, `docs/tasks/v1.5.0/**`, current architecture/status docs | no source changes |
| T150-10/11/12 domain | `internal/engineering/**` | must not edit workflow/closedloop/store |
| T150-20 intelligence | `internal/intelligence/**` | existing llm/airuntime packages read-only unless adapter wiring is explicitly owned by T150-31 |
| T150-21/22 execution | `internal/taskexecution/**`, tests; `internal/agentruntime/**` only when a compatibility defect is proven | no new runtime lifecycle; runtime adapters are integration hotspots |
| T150-30 orchestration | `internal/controlplane/**` | persistence is port + bounded v1.5 compatibility store only |
| T150-31 integration | composition roots and bounded adapter files required to wire controlplane | owns `internal/web/handlers/server.go` only for additive v1.5 routing/dependency injection; do not rewrite existing handlers |
| T150-40+ validation/closure | v1.5 tests, validation fixtures/reports, current status docs | hidden fixtures remain withheld until candidate freeze |

Protected unless specifically owned: frozen PRDs, historical/archived docs, sqlc generated code, existing migrations, `.github/workflows/**`, core v1.4 verification semantics and owner primary-worktree safety.

`.github/workflows/**` is additionally prohibited by the v1.5.0 project override; no task may use it as a validation carrier.
