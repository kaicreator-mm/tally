# ADR-0003 — Structured Provider Endpoint Seeding for the Per-Run Codex Home

**Status:** Accepted (coordinator, P0-04 wave)
**Date:** 2026-09-09
**Authority inputs:** `Tally_PRD_v1.4_Final.md` §29 item 1-2 (real Codex path, truthful capability), INV-V14-07; `docs/tasks/v1.4/DECISION_LOCKS.md` #22; ADR context: A0-02 spike (`docs/tasks/v1.4/research/A0-02-findings.md`); task `docs/tasks/v1.4/P0/P0-03.md` (implemented), `docs/tasks/v1.4/P0/P0-04.md` (contract amendment below).
**Empirical basis (this machine, 2026-09-09, codex-cli 0.153.0):** isolated `CODEX_HOME` + `auth.json` only + `OPENAI_BASE_URL` env → CLI still dials `api.openai.com` (env var not honored) → 401. Isolated `CODEX_HOME` + Tally-generated minimal `config.toml` (custom provider endpoint) + `auth.json` → real inference succeeds (deepseek-v4-flash via responses-compatible endpoint, 9.5k tokens).

## Context

DECISION_LOCKS #22 froze the A0-02 isolation design: per-run fresh `CODEX_HOME`, cross-home handoff of **only `auth.json`**, never the whole home. P0-03 implemented exactly that (commit `11c3b1c`).

That handoff is sufficient for the built-in OpenAI provider with file auth. It is **insufficient for custom-provider deployments** (e.g. a DeepSeek responses-compatible endpoint selected via operator tooling): the provider endpoint lives in `config.toml`, which — correctly — never flows from the owner home into the run home. The isolated session therefore falls back to the default endpoint and fails 401 even though valid credential material was handed off. Authentication availability and endpoint configuration are separate concerns; #22 froze the former and did not consider the latter because the A0-02 substitute-CLI spike needed no provider config.

Without a bounded seam, P0-04 (real Codex standard + held-out E2E) cannot run on any custom-provider deployment, and `session.isolation` can never be truthfully declared there.

## Decision

### 1. `Config.Provider` — a structured, explicit, Tally-generated endpoint declaration

`runtimes/codex.Config` gains one optional field:

```go
type ProviderEndpoint struct {
    ID                 string // model_provider id written to the run home config
    Name               string
    BaseURL            string
    WireAPI            string // "responses" | "chat"
    RequiresOpenAIAuth bool   // true: key arrives via the auth.json handoff (#22, unchanged)
    EnvKey             string // env var name carrying the API key when RequiresOpenAIAuth=false
}
```

When set, session construction (Start/Fork, alongside `newSessionHome`) seeds the per-run home with a **Tally-generated minimal `config.toml`** containing exactly:

```text
model_provider = "<ID>"
model          = "<Config.Model>"          (only when set)

[model_providers.<ID>]
name                 = "<Name>"
base_url             = "<BaseURL>"
wire_api             = "<WireAPI>"
requires_openai_auth = <RequiresOpenAIAuth>
env_key              = "<EnvKey>"           (only when non-empty)
```

Nothing else is ever written.

### 2. What this is NOT (boundary preserved)

- It is **desired-configuration plumbing** (same class as `-m <model>`), not state handoff: the content is generated from typed fields Tally declares; no file is copied from the owner home.
- Instruction-bearing or session-bearing config (`instructions`, `notify`, `mcp_servers`, `profiles`, `history`, rollouts) is structurally impossible — there is no field for it.
- Credential material still crosses homes **only** via the #22 `auth.json` handoff, or via `Config.Env` env-var passthrough when `EnvKey` is used. No secret is ever written by the seeding.
- Owner-home `config.toml`/`AGENTS.md`/`sessions/` still never flow into run homes (P0-03 semantics unchanged).
- The isolation canary's channels (instruction leakage, rollout retrieval) are unaffected: the seeded file carries no instructions and no session state.
- `session.isolation` declaration remains evidence-gated by `Config.SessionIsolationVerified` (P0-03); this ADR changes no capability semantics.

### 3. Ownership and review

- Implementation lives in `runtimes/codex/**` (isolation.go family), owner task P0-04 (contract amended to allow this bounded seam; all other `runtimes/codex` behavior is out of P0-04's scope).
- The real-CLI conformance canary and the standard + held-out E2E that justify setting `SessionIsolationVerified` remain P0-04 acceptance items.

## Consequences

- Custom-provider deployments can run the real Codex path without weakening isolation: endpoint wiring is explicit, typed, generated per run, and disposable with the home.
- Official-provider deployments are unaffected (zero value = zero seeding, exactly P0-03 behavior).
- DECISION_LOCKS #24 records this as a refinement of #22 (endpoint configuration is not credential/session handoff).
