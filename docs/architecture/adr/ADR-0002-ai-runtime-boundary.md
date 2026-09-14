# ADR-0002 — Tally Domain AI Adapter Boundary over the Common AI Runtime

**Status:** Accepted (P0-09)
**Date:** 2026-09-09
**Authority inputs:** `Tally_PRD_v1.4_Final.md` §28 Runtime Boundary + §29 item 10; INV-V14-14; `docs/tasks/v1.4/DECISION_LOCKS.md` #17; task `docs/tasks/v1.4/P0/P0-09.md`.
**External contract evidence:** common AI Runtime repository (`ai-runtime`, sibling checkout; M1 frozen) — `api/openapi.yaml` **v0.4.0-m1-frozen** (network contract truth), `packages/runtime-contract/*.schema.json` (Budget/Operation/Trace schemas), `packages/generated/go` (generated transport), `packages/sdk-go` (thin SDK: `ExecuteOperation`/`GetOperation`/`GetTrace`/`AppendOutcome`, contract-required `Idempotency-Key` header).

## Context

`internal/llm` (legacy current source) owns BaseURL/APIKey/model chat-completions details and dials an OpenAI-compatible endpoint directly from Domain packages. The v1.4 cross-project baseline requires generic inference to go through the shared AI Runtime, which owns provider selection, model routing, tier escalation, common retry, cost, trace and rollout (PRD §28). The common AI Runtime is an external, already-frozen system: its OpenAPI contract, JSON schemas and generated Go SDK exist and are committed; its README states "OpenAPI is network contract truth; generated clients are transport-only" and "Domain declares Operation intent; Runtime chooses tier."

Two boundary failures must be prevented while migrating:

1. provider/model/tier/BaseURL/APIKey leaking back into Tally Domain declarations;
2. Coding Agent runtime semantics (workspace, worktree, session, patch, Codex process identity) crossing into the generic inference boundary — or generic inference creeping into AgentRuntime.

## Decision

### 1. `internal/airuntime` is the only Tally-side generic inference boundary (AC1)

One new package, `internal/airuntime`, defines the domain-facing request/result vocabulary, mirrored field-for-field on the frozen contract:

```text
Request{                       // → contract OperationRequest body
  Operation      string        //   operation        (minLength 1, caller-stable ID)
  Intent{Role, Instructions}   //   intent           (open object; typed here, migrated from llm.Role)
  Input{Prompt, OutputShape}   //   input            (any JSON; narrowed to Tally's prompt+shape shape)
  Budget{MaxAttempts, MaxCostCeilingUSD, DeadlineMs, MaxTokens}
  IdempotencyKey string        //   Idempotency-Key header (1..200), never a body field
}                              // NO provider / model / tier / baseURL / apiKey — anywhere

Result{                        // ← contract Operation (domain mapping)
  OperationID, Status (SUCCEEDED|UNCERTAIN|FAILED|CANCELLED),
  TerminationReason (ATTEMPTS_EXHAUSTED|…|PROVIDER_FAILURE),
  Output json.RawMessage, ProductionCostUSD, StrategyAssignment
}
```

Provider, model, tier, endpoint and credentials are Runtime-side configuration. They never appear as request fields, options, or executor knobs in Tally Domain code. `test/arch/ai_runtime_boundary_test.go` enforces this by vocabulary scan over identifiers and JSON tags (with one explicit exemption: the contract's own `Termination*` outcome enums, e.g. `PROVIDER_FAILURE`, are Runtime-reported facts, not domain routing declarations).

### 2. Consumer-owned `Executor` seam; no transport in the adapter (AC4)

```go
type Executor interface {
    Execute(ctx context.Context, req Request) (Result, error)
}
```

The adapter contains no HTTP, no SDK, no retry, no routing. The real transport is the common AI Runtime's `packages/sdk-go` client (or an equivalent composition-root adapter over the generated client), wired by consumer composition roots when consumers migrate (P0-10: `cmd/tally/req.go`, `cmd/tally/harvest.go`; P0-11: `cmd/worker/main.go`). The external SDK/contract was verified available (M1 frozen, generated Go client committed), so the task proceeds against it; rebuilding transport or routing inside Tally is forbidden and arch-enforced (`net/http` and external imports are rejected in `internal/airuntime`). Had the SDK/contract been unavailable, P0-09 would have returned BLOCKED instead of forging a local client.

### 3. Domain validation stays in Tally (AC2)

`Request.Validate()` is the mechanical Tally-side gate: operation present, intent complete (role/instructions — migrated `llm.ErrNoRole` semantics), idempotency key present and ≤200 chars, budget at contract minimums (`budget.schema.json`), input prompt non-empty and shape JSON valid, and the `logpipe.MustBeRedacted` gate over all free-text channels (migrated `llm.ErrRedacted`). `Validated(next)` wraps any `Executor` so the gate cannot be bypassed by a transport implementation.

The adapter never judges output: `Result.Output` is raw; interfaces expose only `Execute` (no Judge/Verify/Score/Evaluate/Check/Validate — INV-1 shape rule, arch-enforced). Semantic validation of proposals stays in `requirement`, `harvest`, `failure`/`exec`.

### 4. AgentRuntime ⇔ AI Runtime separation (AC3)

No workspace/worktree/session/patch/Codex-process semantics in `internal/airuntime` (identifier scan), and no `internal/airuntime` import from `internal/agentruntime` or `runtimes/**` (import scan, both directions). Generic-inference consumers in v1.4 are requirement, harvest, failure, exec — never the Coding Agent runtime wrapper.

### 5. Failure semantics crossing the boundary

Transport/infrastructure failures (network, auth, ctx cancellation) are `error` returns — no fabricated `Result`. `Status` is the Runtime's own verdict including `UNCERTAIN`; cancellation and provider failure surface as their contract enums (`CLIENT_CANCELLED`, `PROVIDER_FAILURE`) and must never be folded into candidate failures downstream (trust invariant; ADR-0001 §3 routing).

## Dependency status and ownership gap (honest note)

This task ships the contract-shaped boundary **without** a compile-time module dependency on `example.com/ai-runtime`:

- `go.mod`/`go.sum` are P0-09-owned, but adding `require example.com/ai-runtime` (module path is a local placeholder; verified buildable offline via a filesystem `replace`) surfaces as a **direct dependency** and fails `TestDepsWhitelist` in `test/arch/deps_test.go`, whose whitelist map is outside P0-09's Allowed paths (only `test/arch/ai_runtime_boundary_test.go` is owned; `docs/guides/code-standards.md`, the whitelist document, is likewise outside). A side effect also confirmed: the dependency forces the main module's `go` directive to 1.27.1.
- Consequence: the first task that wires the real SDK transport must, in one change: add the module dependency (+`replace` until the runtime module is published under its real path), add the whitelist entry to `test/arch/deps_test.go` and `docs/guides/code-standards.md`, and accept the `go` directive bump. P0-11 owns `test/arch/**` and is the natural owner; it needs `go.mod`/`go.sum` access granted (currently only P0-09 has it). The coordinator should sequence this explicitly.

Until that wiring task lands, `internal/airuntime` is a pure domain-boundary library (PRODUCTION_REACHABLE: N/A(reason: transport wiring belongs to P0-10/P0-11 composition roots; see above)), and `internal/llm` remains the legacy current source, untouched.

## Consequences

- P0-10/P0-11 migrate consumers onto `airuntime.Executor` with `Validated(...)` at their composition roots; domain schemas/validators are untouched (`FakeExecutor` covers deterministic consumer tests).
- Contract drift is caught in-repo: `internal/airuntime/contract_shape_test.go` pins JSON field names and enum values to `v0.4.0-m1-frozen`; renaming upstream breaks these tests visibly.
- `internal/llm` retirement stays with P0-11 once no production consumer remains (PRD §29 item 10).
