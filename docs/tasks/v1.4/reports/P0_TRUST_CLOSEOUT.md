# P0_TRUST_CLOSEOUT — v1.3.1 Trust Closeout Gate (P0-13)

**Date:** 2026-09-09 (Beijing), executed by coordinator
**Task:** `docs/tasks/v1.4/P0/P0-13.md`
**Verdict:** **PASS — v1.4 foundation (P1) may proceed**, with the follow-ups recorded below (none blocks P1; the frozen-adapter items gate only later real-loop phases, see §Follow-ups).

## 1. The 11 closeout items (PRD §29)

| # | Item | Evidence (fresh unless noted) | Status |
|---|---|---|---|
| 1 | CodexRuntime Session Isolation | P0-03 `11c3b1c`: per-session fresh `CODEX_HOME`, auth.json-only handoff, evidence-gated declaration; unit+conformance suites green (fresh rerun this gate) | PASS |
| 2 | Fake + Codex 同一 isolation canary | Shared `test/runtimeconformance` canary: Fake declared+available (historical+fresh); **real codex CLI 0.153.0 PASS through production adapter** (P0-04 `0aa1607`, fresh rerun by coordinator 2026-09-09, ~54s) | PASS |
| 3 | real Codex standard + held-out E2E | P0-04 `0aa1607`: `go test -count=1 -tags=integration ./test/e2e/v14_codex_trust/...` PASS (agent ~144s; coordinator fresh rerun 82.9s); report `docs/tasks/v1.4/reports/P0_CODEX_TRUST.md` | PASS |
| 4 | actual ModelProfile 真正传给 Runtime | P0-01 `8ada9fc` (capability negotiation + ModelProfile propagation) | PASS |
| 5 | actual runtime/model/CLI/session/conformance trace | P0-01 `8ada9fc`, P0-02 `a0cc7bf` (IdentityReporter, reserved JSONB key), P0-04 evidence: CLI version `codex-cli 0.153.0` observed live, binary path, thread id asserted vs claim, `observed_identity` persisted; observed model honestly `unknown` where the runtime surface does not report it (never backfilled from requested `deepseek-v4-flash`) | PASS |
| 6 | Measurement `UsageAvailable` 修复 | P0-06 `696d19c` (monotonic availability; sqlc regen for 0022) | PASS |
| 7 | token unknown 时 runtime call 仍持久化 | P0-05 `d02011c` + P0-06 `696d19c` (durable unknown-usage calls; P0-04 confirms adapter usage stays zero-not-fabricated) | PASS |
| 8 | Exposure ancestor lineage 可见 | P0-07 `2b1df9f` | PASS |
| 9 | Baseline readout operationalization | P0-08 `703f128` (`tally report baseline` on real history) | PASS |
| 10 | 通用 LLM client 迁入 AI Runtime | P0-09 `aaaba60` (ADR-0002 adapter), P0-10 `1d792c4` (requirement/harvest), P0-11 `1795525` (failure/exec/worker; legacy `internal/llm` deprecated + arch-enforced `test/arch/ai_runtime_domain_migration_test.go`) | PASS |
| 11 | 文档与代码状态同步 | P0-12 `740938a` (EOL policy + baseline docs); docs corpus landed `e0e0664` (207 files: v1.4 PRD/DAG/tasks/reference-packs + v1.3 archive); `CURRENT_IMPLEMENTATION_STATUS.md` updated this gate | PASS |

## 2. Fresh validation record (executed this gate, 2026-09-09)

```text
go build ./...                          PASS
go vet ./...                            PASS
go test ./...                           PASS   (exit 0)
go test -tags=integration ./...         PASS   (exit 0; 47 packages ok)
```

Environment: local Docker postgres:16 (`DATABASE_URL`), real codex CLI `codex-cli 0.153.0` via `TALLY_E2E_CODEX_AUTH_FILE` (credential git-excluded, never printed/committed; tests SKIP without it).

## 3. Incidents during the gate (honest record)

1. `test/harvestenv` integration build was broken by P0-10 (removed `harvest.Proposer`; integration-tagged compile not covered by P0-10's unit-only validation). Fixed by `0fb2250` (`fakeProposer` → `airuntime.FakeExecutor`, same shape as the P0-10-updated unit tests). Lesson recorded: migration tasks must `go vet -tags=integration` their touched test surfaces.
2. First full-parallel integration run had two load-sensitive flakes: `internal/failure` schema-name nanosecond collision and `internal/workspace` concurrent-worktree Windows git error. Both passed on individual rerun AND the final full suite (post-migration, exit 0). No gate item depends on the flaky path; tracked as test-infra follow-up.
3. `test/harvestenv` expects a pre-migrated database (no self-run goose). The gate DB was migrated via `go run ./cmd/tally migrate` before the suite. Legacy test-infra behavior, documented here.

## 4. Known limitations (not gate items; owned follow-ups)

- **Frozen-adapter findings (owner: `runtimes/codex`, from P0-04 report §6):** (a) `classifyTurn` marks a turn RunFailed on codex's non-fatal `item.type=="error"` metadata warnings (unknown-model warning fires every turn for custom models); (b) frozen argv places flags after `exec resume <id>` which 0.153.0 rejects. P0-04's E2E drove production primitives directly; the P1-39 kernel path with real Codex needs these fixed before P2-09/P2-10 and P4-03.
- Keyring-auth deployments are isolation-incompatible (key derives from home path); file-auth handoff or unsupported marking per DECISION_LOCKS #22 (documented in `runtimes/codex/isolation.go`).
- Windows codex 0.153.0 `workspace-write` sandbox mounts the workspace read-only and rejects all commands; `danger-full-access` documented as the least working mode for write fixtures (P0-04 harness).
- `internal/llm` legacy remains as deprecated composition-root transport until the shared AI Runtime real SDK wiring task replaces it (ADR-0002 interim adapter).

## 5. Decision

All 11 items PASS with fresh evidence; no historical PASS was relabeled fresh. **The gate passes: P1 (Contract/Policy/Assurance foundation) may start.**
