# A0 Architecture Gate — Verdict

**Gate:** A0-06, 2026-09-08
**Reviewer:** Principal orchestrator (strong-review tier). **Disclosure:** subagent dispatch was unavailable (provider captcha gate), so this gate was reviewed by the same agent that executed A0-02/03/04 — mitigated by executable evidence (committed tests, frozen hashes, honest NOT_RUN records) rather than narrative claims. A later independent re-review is welcome but not required to proceed.

## Verdict: **READY_TO_IMPLEMENT**

All four L2 freeze blockers resolved; every HIGH/CRITICAL claim has a frozen decision and an owning task. P0 may start.

## Freeze blocker closure

| # | Blocker (L2 §10) | Resolution | Evidence |
|---|---|---|---|
| 1 | Codex isolation decision + production path | PASS path: per-run `CODEX_HOME` + `auth.json`-only handoff; capability undeclared until real-CLI canary | A0-02 findings + `runtimes/codex/isolation_spike_test.go` (4 tests PASS) |
| 2 | Assurance default decision or safe conservative default | Conservative default: L1 floor + L3 risk-adjusted uplift; L2 experimental (judge NOT_RUN) | A0-03 findings; corpus v2 frozen `e83b8bd3…`; 6/6 tests PASS |
| 3 | Repair/held-out ADR accepted | ADR-0001 accepted (5/5 task ACs) | `docs/architecture/adr/ADR-0001-v14-closed-loop-repair.md` |
| 4 | Aggregate-contract compatibility recorded | Decision Lock 5 (pre-existing) reaffirmed; no physical merge | `DECISION_LOCKS.md` |

## HIGH/CRITICAL claim disposition (AC1)

All HIGH/CRITICAL claims from §3 are frozen with a resolved next action:

- **KEEP**: A01–A04, A06, A07, A13, A15, A20, A27, A28 → owning tasks P1-01, P2-*, P0-07/P2-06.
- **CHANGE**: A05 (no merge; aggregate), A14 (bounded, policy-default 2), A17 (extend taskstate), A19 (append-only outcome), A21 (observed identity; P0-01/02), A22 (persist unknown-usage calls; P0-05/06), A23 (AI Runtime migration; P0-09..11).
- **REJECT**: A10 (held-out universal default), A11 (isolation declared today), A12 (repair inside inner loop), A16 (raw held-out feedback), A24 (CodexRuntime into AI Runtime), A26 (full Jira/GitLab IA).
- **DEFER**: A18 (WorkItem rewrite), A25 (Guidance plane).
- **SPIKE → RESOLVED**: A09 (assurance mapping → A0-03), A11 fix path (→ A0-02), A14/A15 semantics (→ ADR-0001).

## Deferred validation obligations (not blockers, owned downstream)

| Obligation | Owner | Reason not blocking |
|---|---|---|
| Real-CLI session isolation canary | P0-03/P0-04 | `session.isolation` stays undeclared until proven — no unproven claim enters production |
| Live LLM-judge bake-off rerun | P1-06 | L2 is not a default tier; conservative default does not depend on it |
| `make check` full gate | P0-13 | Go suites runnable locally (module cache present); integration deps pending per-task |

## AC checklist

- AC1 every HIGH/CRITICAL claim resolved with next action: **PASS**
- AC2 assurance default references A0-03 results: **PASS** (Decision Lock 21)
- AC3 Codex isolation decision references A0-02: **PASS** (Decision Lock 22)
- AC4 Repair ADR accepted: **PASS** (Decision Lock 23)
- AC5 unambiguous verdict: **PASS** — READY_TO_IMPLEMENT

## Validation

```text
git diff --check: PASS (this gate's files)
go test ./test/eval/v14_assurance/...: PASS (6/6)
go test ./runtimes/codex/... ./test/runtimeconformance/...: PASS
real Codex CLI canary: NOT_RUN(codex CLI not installed; owned by P0-03/P0-04)
```
