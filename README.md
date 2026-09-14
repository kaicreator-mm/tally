# Tally

**Software Engineering Harness for Coding Agents**

Tally turns planning output and software-engineering practice into executable control over coding agents.

```text
PlanningResult
  ↓
Intent / Contract
  ↓
Policy / Risk / Assurance
  ↓
Coding Harness Runtime
  ↓
ChangeSet
  ↓
Verification / Repair
  ↓
Acceptance
  ↓
Git Integration
  ↓
Outcome / Rule Learning
```

> Tally does not trust an Agent because it says “done”. Tally freezes what done means, executes in an isolated workspace, independently verifies the immutable candidate, controls durable Git state, and escalates only when judgment is required.

## Repository state

This is an existing Go brownfield system. The current source contains the v1.4.1 implementation plus closure-repair hardening; it is suitable as a **development baseline**, not yet as a fully release-qualified artifact because external Hidden Validation remains pending.

Current stack:

- Go 1.23;
- PostgreSQL + pgx + sqlc + goose;
- `net/http` + chi;
- Go templates + HTMX;
- Git-backed engineering artifacts;
- out-of-process Codex / DeepSeek Harness runtime adapters.

### v1.4.1 baseline

Implemented and strengthened:

- Intent → Dynamic Form → Confirm → atomic Freeze;
- Contract / Risk / Assurance production authority;
- risk-adjusted Acceptance;
- human-centered Work / Review / Rules / Integration UI;
- Rule-first EngineeringPolicy governance;
- local Git Control Plane L0–L3;
- append-only Outcome capture with atomic replay idempotency;
- deterministic Safe Export and repository hygiene gates;
- centralized transitional AI Runtime transport adapter.

DeepSeek Harness is **Strong-ready** but its real launcher/provider gate still requires external `TALLY_DSH_BIN` + `DEEPSEEK_API_KEY` evidence. P6-08 **Hidden Validation is PENDING_EXTERNAL**. See `docs/releases/v1.4.1_STRONG_BASELINE.md`.

## Runtime integration

Tally embeds the Harness protocol, not the Harness implementation.

```text
Tally Go
  ↓
internal/agentruntime.Runtime
  ├── runtimes/codex → external Codex CLI
  └── runtimes/dsh   → external DSH process / subprocess bridge
```

Tally keeps authority over:
- workspace isolation;
- ChangeSet;
- Git durability/integration;
- Verification;
- Acceptance;
- Outcome.

## Start here

### Coding agents

1. `AGENTS.md`
2. `docs/product/Tally_PRD_v1.4.1_Final.md`
3. `docs/tasks/v1.4.1/DECISION_LOCKS.md`
4. assigned v1.4.1 Task + appended L3 pack

### Current source/target/status

- `docs/STATUS.md`
- `docs/architecture/CURRENT_CODEBASE.md`
- `docs/architecture/V1.4.1_TARGET.md`
- `docs/implementation/CURRENT_IMPLEMENTATION_STATUS.md`
- `docs/implementation/V1.4.1_GAP_MAP.md`
- `docs/tasks/v1.4.1/README.md`

Historical v1.4 PRD/tasks/gates remain evidence and are intentionally retained.


## Build and validation

```bash
make migrate
make build
make test
make test-integration
make check
```

Useful focused commands:

```bash
go test ./test/arch/...
go test ./internal/...
go test ./runtimes/...
go test ./test/runtimeconformance/...
go test -tags=integration ./test/e2e/...
```

Integration tests require reachable PostgreSQL via `DATABASE_URL`.

**Never report PASS for a command that did not execute successfully.**

The v1.4 final gate recorded fresh full-suite PASS evidence on 2026-09-10. Any new v1.4.1 run must still record its own actual result; environment dependency failures are `NOT_RUN(reason)`, never inherited PASS.

## Run

```bash
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/tally?sslmode=disable'
export TALLY_PASSWORD='change-me'

make migrate
make build

./bin/server
./bin/worker
```

Default endpoints:

- server/UI: `http://localhost:8080`
- worker health: `http://localhost:8081/healthz`

See [`docs/guides/deployment.md`](docs/guides/deployment.md) and [`docs/guides/operations.md`](docs/guides/operations.md).

## Engineering authority model

```text
LLM output       = Proposal
Agent output     = Claim
Validator output = Evidence / Fact
Lifecycle state  = Derived
Human            = Judgment / authority where policy requires
```

A coding agent is never trusted merely because it produced the code.

Likewise, a Tally development task is not DONE merely because a class exists and a unit test passes. Current task completion should prove, as applicable:

```text
IMPLEMENTED
→ WIRED
→ TESTED
→ PRODUCTION_REACHABLE
→ E2E_PROVEN
```

## Product boundary

Tally owns the coding-task engineering loop.

AI Runtime owns Project Planning / Task Planning and can internally use DIRECT/AUTO/ORCHESTRATED planning. Tally consumes the resulting planning contract. Concrete Coding Task execution remains in Tally's `AgentRuntime`, with Codex and DeepSeek Harness running out-of-process behind `internal/agentruntime.Runtime`.

## Long-term direction

The long-term thesis remains:

> **Vibe Coding × Software Engineering**

But `Tally_Product_Direction.md` is deliberately non-binding. v1.4.1 only implements capabilities explicitly promoted by the Final PRD and task program.
