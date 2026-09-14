# Tally v1.4 → Next Session HANDOFF

**Written:** 2026-09-10, by the v1.4 orchestrator (GLM-5.3) for the next model (256k context).
**State:** v1.4 COMPLETE — 55/55 tasks DONE, 6 gates PASS. Final gate `PASS_WITH_KNOWN_LIMITATIONS` (`docs/tasks/v1.4/reports/V1.4_FINAL_GATE.md`). Real-codex suites freshly re-validated PASS after credit restoration (commit `7ff9fe8`).

## 1. Read this first (authority order)

1. `AGENTS.md` (repo root — engineering instructions, still canonical)
2. `docs/tasks/v1.4/reports/V1.4_FINAL_GATE.md` (final closeout + known limitations + proposed next scope)
3. `docs/tasks/v1.4/STATUS.md` (all task→commit mappings)
4. `docs/evals/v1.4/RESULTS.md` + `docs/tasks/v1.4/reports/P4_PRODUCT_VALUE_GATE.md` (experiment outcome: INCONCLUSIVE / CONTINUE)
5. `docs/architecture/CURRENT_CODEBASE.md` (v1.4 package facts) + `docs/implementation/CURRENT_IMPLEMENTATION_STATUS.md`
6. ADRs: `docs/architecture/adr/ADR-000{1,2,3}-*.md`; locks: `docs/tasks/v1.4/DECISION_LOCKS.md` (#1-24)

## 2. Open items (all PROPOSED, none authorized — owner must adjudicate)

1. **Migration verification root-cause** (follow-up #1): 14/14 internal verify FAIL vs 5/5 external-judge accept on db tasks (`docs/evals/v1.4/results/raw/tally/db-0*.json`). Prerequisite to any rerun.
2. **Larger pre-registered trial rerun** (≥16 tasks; credits RESTORED and verified working 2026-09-10; frozen set hash `bbf381be…`, replacement policy TASK_SET.md §8/§11).
3. **G1 rework guardrail**: 14-day window closes 2026-09-23 — compute from outcome events then.
4. Intent→freeze wiring completion (P3-08 declared unwired: `SetV14ProposeExecutor` + draft→freeze chain).
5. Contract compiler production composition (risk-signal source + policy wiring into `taskstate.Snapshot.Compiled` — P3-09 declared 2 unbuilt basis dimensions).
6. L2 live-judge bake-off rerun (needs judge credentials; lock 21 keeps L2 experimental-only).

## 3. Environment facts (verified working)

- **codex binary:** `C:\Users\15549\AppData\Local\OpenAI\Codex\bin\9ba750cce02d5e5c\codex.exe` (codex-cli 0.153.0)
- **Provider (ADR-0003):** `ProviderEndpoint{ID "custom", Name "deepseek", BaseURL "https://api.deepseek.com", WireAPI "responses", RequiresOpenAIAuth true}`, model `deepseek-v4-flash`. Credentials RESTORED (verified 2026-09-10).
- **Credential file:** `C:\Dev\Code\tally\.e2e\deepseek-auth.json` — git-excluded via `.git/info/exclude`. NEVER print/commit. Tests read it via `TALLY_E2E_CODEX_AUTH_FILE`.
- **Postgres:** Docker + `postgres:16` image local. ALWAYS use port 5433 (`docker run -d --name <unique> -p 5433:5432 -e POSTGRES_PASSWORD=test postgres:16`), `DATABASE_URL=postgres://postgres:test@localhost:5433/postgres?sslmode=disable`, migrate via `go run ./cmd/tally migrate`, REMOVE container after. **localhost:5432 belongs to the USER's own containers (db03pg etc.) — never touch forge-saas-*/db03pg; wrong-password there breaks DB-gated tests that expect unreachable→SKIP.**
- **Working e2e invocation:** `DATABASE_URL=... TALLY_E2E_CODEX_AUTH_FILE=C:/Dev/Code/tally/.e2e/deepseek-auth.json go test -count=1 -tags=integration -timeout 25m ./test/e2e/...`

## 4. Operating protocol that worked (keep it)

- **Model strategy:** low-cost Flash subagents implement; strong model only for: trust-boundary code review (runtimes/codex, closedloop), gate execution, shared-hotspot minimal edits, freeze/blocked adjudication, final audits. Every task = subagent report → coordinator verifies scope (`git status`) + fresh tests (`-count=1`) → **one task one commit**.
- **Concurrency:** dispatch by dependency frontier; same-package tasks are serialized (or file-stem pinned + 60s sibling-retry discipline); same-port DB containers must not overlap (use 5433/5434 split).
- **Agents must never commit** — coordinator owns commits, review first.
- **Honesty:** NOT_RUN(reason)/SKIP recorded verbatim; never fabricate PASS; failing-test-first for any production fix; preregistered metrics/sets immutable (freeze changes need the §8 adjudication procedure — see `ecbf830` precedent).
- **Shell gotchas (Git Bash):** pipes swallow exit codes (`cmd | tail; echo $?` lies — capture to a file or use `; E=$?` directly); `cd` in compound commands breaks later relative paths.

## 5. Key lessons / gotchas on record

- Migration tasks must `go vet -tags=integration` their test surfaces (P0-10's harvestenv compile regression, fixed `0fb2250`).
- Test-package import cycles: pins belong on the CONSUMER side (fixed `8716d02`).
- Long-running agents can die on model-request failures mid-task — the resume pattern (audit leftover state → complete-not-redo → incremental materialization) is proven in P4-03.
- Bare-arm `prompt_reissue` counts as an intervention (prereg M1/M3); observed model stays `absent` unless the runtime reports it (lock 15).
- `tally.exe` at root was a stray build artifact (removed); rebuild via `go build ./cmd/tally` if needed.

## 6. Where the evidence lives

- Gates: `docs/tasks/v1.4/reports/{A0? – see STATUS, P0_TRUST_CLOSEOUT, P1_FOUNDATION_GATE, P2_CLOSED_LOOP_GATE, P3_MVP_GATE, P4_PRODUCT_VALUE_GATE, V1.4_FINAL_GATE}.md`
- Experiment: `docs/evals/v1.4/{PRE_REGISTRATION,TASK_SET,RUN_PROTOCOL,RESULTS}.md`, raw rows `docs/evals/v1.4/results/raw/**`, runners `scripts/v14_trial/{bare,tally}/**`
- Trial runs-root (local, uncommitted): `/tmp/tally-v14-trial/`
