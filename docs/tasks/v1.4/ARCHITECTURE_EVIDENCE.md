# Tally v1.4 — L2 Architecture Evidence Review

**Status:** `READY_AFTER_FIXES`  
**Source ZIP SHA-256:** `93577bdd56db181e8a81bd36c0f49b991f5dea599767000a74a8c0f66ca820ba`  
**Source HEAD:** `18193d5ff9ff9b6699a4d9b16b6429e2e084094e`  
**PRD SHA-256:** `35474f42422222d339214a69c3153547e4a4c8d616f573f1d50af9c371b38907`  
**L2 Prompt SHA-256:** `8285fd567bc1ad4198fc8b25ab3c7724f866321e82f74adbc6ebe4efbc4c5115`  
**Review date:** 2026-09-08

## 1. Executive Architecture Verdict

**Verdict: `SUPPORTED WITH CHANGES` / Freeze recommendation: `READY_AFTER_FIXES`.**

The v1.4 direction does **not** require a greenfield rewrite. The current source already implements most of the v1.3 Vertical Trust Core: immutable ChangeSet, append-only evidence/exposure/supersession facts, VerificationContract freeze, held-out execution, derived task state, deterministic hunk accounting, disposable worktrees, Runtime abstraction, a PostgreSQL lease queue, and a worker path that executes Codegen then Verification.

The missing architectural center is an **outer closed-loop orchestrator**:

```text
ContractSnapshot
  -> CodegenRun
  -> immutable ChangeSet
  -> Verification
  -> classify failure
  -> bounded Repair (new Run + new ChangeSet)
  -> re-Verification
  -> Accept / Escalate
```

Three items must be resolved before the v1.4 implementation architecture is treated as frozen:

1. **Codex session isolation** must become real and conformance-tested before `session.isolation` is declared.
2. **Assurance L1/L2/L3 defaults** require a local bake-off; external evidence supports both held-out verification and cheaper independent-judge approaches, so the default cannot be selected by opinion.
3. **Repair + held-out feedback semantics** require an ADR: raw held-out content must not be leaked to the implementation authority during Repair; any exposure must be recorded and assurance degraded or the task escalated.

## 2. Current Source Baseline

### Strong existing foundations

- `internal/taskcontract/**` — versioned TaskContract parsing, validation and content identity.
- `internal/verification/**` — frozen VerificationContract, visible/held-out execution, qualification and coverage.
- `internal/changeset/**` + migration `0019` — immutable ChangeSet and append-only supersession facts.
- `internal/evidence/**` + migration `0020` — evidence, exposure, source/information independence.
- `internal/taskstate/state.go` — evidence-driven derived state, `Basis`, `Gaps`, strict satisfied vs waived semantics.
- `internal/workflow/v13.go` — bounded *inner* codegen iterations and model/runtime escalation.
- `cmd/worker/v13.go` — production composition of TaskContract -> Codegen -> ChangeSet -> Verification.
- `internal/workspace/**` — disposable Git worktree execution.
- `internal/jobs/**` + migration `0014` — durable Postgres job lease/heartbeat/reclaim.
- `internal/measurement/**` + migration `0021` — usage/human effort facts and reports.
- `runtimes/fake/**`, `runtimes/codex/**` — Runtime abstraction implementations.

### Known current gaps from source audit

- `runtimes/codex` intentionally does **not** declare `session.isolation`; comments identify shared `CODEX_HOME` state as an observable isolation risk.
- `internal/workflow/v13.go` passes `StartRequest{Workspace: ...}` without the configured `ModelProfile`, although Codex `Start` supports it.
- Codex has an actual `RuntimeSnapshot`, but the main workflow persists descriptor-level identity rather than the full actual CLI/binary/session identity.
- `measurement.RecordRuntimeUsage` can move aggregate `UsageAvailable` back to true after an earlier missing-usage call.
- Calls with unknown token counts are not durably represented in `llm_usage` because token/cost columns are non-null and the current writer skips unavailable usage.
- `cmd/worker/v13.go` runs Codegen and Verification sequentially but has no Verification-failure -> Repair outer loop.
- Current `ExecutionPolicy.Retry.MaxIterations` is an **inner codegen retry budget** and must not be reused as the v1.4 outer Repair budget.
- Generic `internal/llm` still owns provider/base-URL/model details; this conflicts with the cross-project AI Runtime boundary.
- Current archive has 229 Git-dirty paths caused by line-ending normalization only; `git diff --ignore-cr-at-eol` shows no content delta. `.gitattributes` is absent.

### Validation limitation of this review

A fresh Go validation run could not be completed in the review environment because required modules were not locally cached and outbound access to `proxy.golang.org` was unavailable. Therefore current-source runtime validation is `NOT_RUN(dependency fetch/network unavailable)`. Historical source evidence remains in `docs/tasks/v1.3/reports/P1_TRUST_GATE.md`, but it is not treated as a substitute for the v1.4 closeout gate.

## 3. Architecture Claim Register / Evidence Matrix

| Claim | Decision | Risk | Confidence | Main basis | If wrong |
|---|---|---:|---|---|---|
| A01 Go modular monolith + Git monorepo + PostgreSQL remain appropriate | **KEEP** | MED | HIGH | Current implementation cohesion; no observed scale requiring distribution | Premature service split / ops burden |
| A02 Existing PostgreSQL lease queue is sufficient; no Temporal/Kafka now | **KEEP** | MED | HIGH | `internal/jobs`, migration 0014; PostgreSQL `SKIP LOCKED` is explicitly suitable for queue-like multi-consumer access | Duplicate/lost scheduling if existing lease semantics are broken |
| A03 Disposable Git worktree per coding run remains the execution boundary | **KEEP** | HIGH | HIGH | Current workspace package; Git linked-worktree model; v1.3 invariants | Owner tree contamination / candidate ambiguity |
| A04 Immutable ChangeSet + append-only facts + derived state remain the trust core | **KEEP** | CRITICAL | HIGH | Source migrations 0019/0020/0021 and taskstate | False authoritative state / unverifiable history |
| A05 Physically merge TaskContract + VerificationContract in v1.4 | **CHANGE** | HIGH | HIGH | Both are mature/frozen and widely referenced in source | Large regression surface with little product value |
| A06 Introduce a v1.4 `ContractSnapshot` aggregate over the two existing contracts | **KEEP** | MED | HIGH | Minimizes migration while matching PRD semantics | Duplicate authority if aggregate is allowed to override pins |
| A07 Engineering Policy should be separate from existing runtime `ExecutionPolicy` | **KEEP** | MED | HIGH | Existing `internal/policy` owns network/shell/budget/retry; new policy owns acceptance/gates | Policy authority confusion |
| A08 `Policy.requires[]` may be shown to Agent before execution and compiled to required evidence | **KEEP WITH CONSTRAINTS** | HIGH | MED-HIGH | Contract-before-execution semantics; no hidden verifier content may leak | Guidance/gate divergence or test leakage |
| A09 Exact L0/L1/L2/L3 default mapping is already known | **SPIKE REQUIRED** | CRITICAL | LOW | External studies disagree on held-out incremental value | Excess verification cost or weak assurance |
| A10 Held-out should be universal default | **REJECT** | HIGH | MED-HIGH | SpecBench supports held-out; EvilGenie finds cheaper judge/edit checks highly effective in unambiguous cases | Cost/latency without proportional benefit |
| A11 Codex can safely claim `session.isolation` today | **REJECT / FIX** | CRITICAL | HIGH | Source comment and missing capability declaration; Codex instruction aggregation includes `$CODEX_HOME` files | Held-out information independence false-positive |
| A12 v1.4 Auto Repair belongs inside existing `workflow.TaskExecution` inner retry loop | **REJECT** | HIGH | HIGH | Existing loop ends at ChangeSet; verification occurs outside it | Conflates runtime/codegen failures with acceptance failures |
| A13 v1.4 needs an outer closed-loop orchestrator above codegen + verification | **KEEP** | HIGH | HIGH | Source composition gap; loop-engineering evidence | Human remains the failure-copy/retry orchestrator |
| A14 Outer Repair attempts should be bounded, default 2 | **CHANGE** | MED | HIGH | Bounded retry is strong practice; exact count is empirical | Infinite repair loops or premature escalation |
| A15 Structured verification failure feedback should drive Repair | **KEEP** | MED | MED | 2026 repair-loop preprint shows structured external feedback can materially improve bounded repair; local validation still required | Repair ignores evidence or chases noise |
| A16 Raw held-out diagnostics can be given to Repair Agent without changing assurance | **REJECT** | CRITICAL | HIGH | Violates information-independence semantics by construction | Strict assurance becomes false |
| A17 `VerifiedBasis` should be a new mutable/persisted lifecycle object | **CHANGE** | MED | HIGH | `taskstate.Result` already has derived `Basis`/`Gaps`; facts already persist | Duplicate state and reconciliation bugs |
| A18 Full WorkItem/Requirement rewrite is necessary before closed loop | **DEFER** | MED | HIGH | Existing TaskContract is enough to test closed-loop hypothesis | AI-Jira scope before product proof |
| A19 Outcome should be one mutable row with boolean flags | **CHANGE** | MED | HIGH | Existing fact discipline is append-only | Lost event ordering / overwritten history |
| A20 Outcome capture should start now as append-only events | **KEEP** | MED | HIGH | Cannot reconstruct reliably later | Lost product evidence |
| A21 Adapter descriptor is enough to represent actual Runtime/model identity | **CHANGE** | HIGH | HIGH | Codex snapshot already exposes actual CLI/binary; workflow can record desired model without executing it | Misattributed results/costs |
| A22 Unknown token usage may be omitted from durable call history | **CHANGE** | HIGH | HIGH | Current measurement path loses calls; aggregate availability can flip incorrectly | Cost/latency evidence is biased |
| A23 Generic LLM provider routing belongs in Tally | **CHANGE** | MED | HIGH | Cross-project AI Runtime baseline | Duplicated provider/routing/fallback logic |
| A24 AgentRuntime/CodexRuntime should move into common AI Runtime | **REJECT** | CRITICAL | HIGH | Coding runtime owns workspace/session/patch semantics, not generic inference | Domain/runtime boundary collapse |
| A25 Full Guidance Plane is required in v1.4 | **DEFER** | MED | HIGH | Existing project docs can be passed through; Contract requirements supply immediate prevention context | Large authoring/UI scope |
| A26 Full Jira/GitLab-style IA is required in v1.4 | **REJECT/DEFER** | MED | HIGH | Product hypothesis is closed-loop execution, not migration to a new SoR | Adoption friction |
| A27 Current source is a viable brownfield base for v1.4 | **KEEP** | CRITICAL | HIGH | Most v1.3 primitives already exist | Rewrite would discard validated trust core |
| A28 Repair lineage must include exposure lineage across ancestor ChangeSets | **KEEP** | HIGH | HIGH | Existing supersession + exposure facts, but current UI/read model is candidate-local | Hidden leakage across repair attempts |

## 4. Evidence Summary

### External evidence supporting current direction

- **OpenAI — Running Codex safely at OpenAI (2026-05-08):** bounded sandbox, managed config, network policy, audit telemetry, low-risk frictionless / high-risk explicit review. Supports bounded execution and risk-adjusted intervention.
- **OpenAI — Unrolling the Codex agent loop (2026):** Codex instructions aggregate from `$CODEX_HOME` and project `AGENTS*` files. This directly makes shared `CODEX_HOME` relevant to session-isolation claims.
- **PostgreSQL 18 SELECT documentation:** `SKIP LOCKED` is unsuitable for general consistent reads but explicitly useful to avoid contention among multiple consumers of a queue-like table. Supports the existing queue pattern under current scale.
- **SpecBench (2026 preprint):** visible-test saturation can coexist with held-out failures; supports held-out as a meaningful high-strength assurance mechanism.
- **EvilGenie (benchmark):** independent LLM judge + test-file mutation detection are highly effective for clear reward-hacking cases and held-out gives only small additional gain in that setting. Supports a bake-off rather than universal held-out.
- **Self-Authored Verification Is Unreliable (2026 preprint):** verification controlled by the same improving agent can diverge from deployment performance. Supports external acceptance authority.
- **Structured Feedback Improves Repair in an LLM Agent Loop (2026 preprint):** structured validator feedback substantially improved terminal repair success in a bounded agent loop in the studied environment. Supports structured RepairFeedback, but production external validity remains to be validated locally.
- **LoopsBench (2026 preprint):** long-horizon coding success depends on loop engineering and regression obligations, supporting a first-class outer loop rather than a one-shot agent result.

## 5. Failure Pattern Catalog

| Failure pattern | Trigger | Weakness | Required defense |
|---|---|---|---|
| False held-out independence | Shared Codex state / leaked hidden diagnostics | Runtime/session boundary not real | Per-run isolation canary + exposure accounting |
| Green by verifier mutation | Agent changes protected tests/config | Maker controls checker | Hash/integrity checks + immutable/protected assets |
| Repair games the verifier | Raw hidden failure tells implementation what hidden test expects | Repair crosses authority boundary | Sanitized feedback or escalation; exposure downgrades assurance |
| Infinite repair loop | Verification fails repeatedly | Retry lacks semantic bound | Separate outer repair budget + classification + hard stop |
| Wrong-model attribution | Desired profile logged but not passed to Runtime | Observability records intent as fact | Record actual runtime/model snapshot |
| Missing-cost bias | Usage unavailable -> call omitted | Measurement stores only complete token rows | Persist call with usage availability explicitly represented |
| Policy false-negative | Risk signal not matched to a gate | `required = empty` can look complete | `unmatched_risk_signals` + assurance upgrade/escalation |
| Policy false-positive / waiver inflation | Trigger too broad | Only escape is manual waiver | Evidence-based risk reclassification + monitor overrides |
| Inner/outer retry conflation | Reuse `MaxIterations` for verification repair | Different failure semantics share one budget | Separate `MaxRepairAttempts` |
| Candidate lineage ambiguity | Repair mutates prior candidate in place | Verdict no longer binds a fixed result | New Run + new ChangeSet + supersession fact every attempt |

## 6. Complexity Reduction Opportunities

1. **Do not introduce Temporal/Kafka/microservices.** Existing PostgreSQL job leases + in-process modular orchestration are sufficient for the current observed scale.
2. **Do not physically merge the two existing contract artifact formats yet.** Add an aggregate facade and defer storage migration.
3. **Do not add a persisted VerifiedBasis table.** Extend the current deterministic read model.
4. **Do not add WorkItem/Requirement authoring as a prerequisite.** Use current Task identity/source refs until closed-loop value is proven.
5. **Do not build full Guidance authoring.** Pass through existing project context and expose frozen requirements to the Agent.
6. **Do not build a second Runtime in v1.4 NOW.** First prove Codex closed-loop correctness; second Runtime is NEXT.
7. **Do not build full Metrics/Quality dashboards.** Persist facts and add only the minimal closed-loop read projections.

## 7. Missing Engineering Obligations

- Explicit outer-loop cancellation semantics.
- Repair attempt identity and idempotency.
- Previous-candidate materialization/base revision rules.
- Safe feedback schema for held-out failures.
- Policy match coverage and unmatched-risk handling.
- Actual Runtime/model identity persistence.
- Unknown-usage durable measurement semantics.
- Outcome event schema.
- Repair lineage exposure projection.
- Separate limits for inner codegen retry vs outer verification-driven repair.

## 8. Architecture Invariants to Freeze

- **INV-V14-01** Agent Claim never changes authoritative delivery state.
- **INV-V14-02** Contract semantics are frozen before a production Coding Run begins.
- **INV-V14-03** A Repair never weakens frozen acceptance, gate requirements, or required assurance.
- **INV-V14-04** Each Repair attempt creates a new CodegenRun and a new immutable ChangeSet.
- **INV-V14-05** Outer Repair budget is distinct from inner Codegen iteration/retry budget.
- **INV-V14-06** Every verdict binds one exact ChangeSet and exposes its basis.
- **INV-V14-07** Held-out raw content may not cross into implementation authority while strict information independence is claimed.
- **INV-V14-08** Any held-out exposure is append-only evidence and can only preserve or reduce assurance, never increase it.
- **INV-V14-09** `unmatched_risk_signals != []` cannot silently select the lowest assurance path.
- **INV-V14-10** `satisfied` and `waived` remain distinguishable in facts and UI.
- **INV-V14-11** Actual Runtime/model identity is observed and recorded separately from desired configuration.
- **INV-V14-12** Unknown token/cost usage does not erase the fact that a Runtime/LLM call occurred.
- **INV-V14-13** Derived lifecycle/assurance projections are rebuildable from authoritative records.
- **INV-V14-14** Generic LLM inference uses the common AI Runtime boundary; Coding Agent Runtime remains Tally-owned.
- **INV-V14-15** Owner primary worktree is never the writable Agent execution workspace.
- **INV-V14-16** Outcome history is append-only.

## 9. Required Spikes Before Architecture Freeze

### SPIKE-1 — Codex Session Isolation

Hypothesis: one Codex Coding Run can receive an isolated home/session state such that another run cannot retrieve its rollout/instruction/session artifacts while authentication still works.

Pass: same conformance canary passes for FakeRuntime and CodexRuntime; no cross-run artifact retrieval through protocol-observable channels.

Fail: isolation cannot be achieved without unacceptable auth/config coupling -> L3 held-out with Codex remains blocked and architecture must use a separate verifier boundary.

### SPIKE-2 — Assurance Bake-off

Hypothesis: a cost-appropriate default can be selected among L1 deterministic, L2 independent judge + mutation detection, and L3 held-out.

Measure: issue capture set, unique captures, latency, model/tool cost, false-positive review, task type.

Decision: exact default mapping is frozen only after results.

### SPIKE-3 — Repair / Held-out Feedback Boundary

Hypothesis: verification failures can be converted to bounded, useful RepairFeedback without leaking hidden acceptance assets.

Pass: a stable feedback schema exists; L3 hidden content remains sealed or exposure is explicitly recorded and assurance downgraded.

Fail: Repair requires raw held-out contents to succeed -> automatic L3 repair is disabled; escalate instead.

## 10. Freeze Recommendation

`READY_AFTER_FIXES`

**Freeze blockers:**

1. Codex isolation spike has a decision and production path.
2. Assurance bake-off produces a default-selection decision or explicitly leaves a safe conservative default.
3. Repair/held-out ADR is accepted.
4. v1.4 Decision Locks record aggregate-contract compatibility rather than physical merge.

Once these are resolved, implementation should proceed with the Task DAG in this pack. The remaining P0 trust-closeout items are implementation gates, not reasons to redesign the architecture.

## 11. A0 spike resolutions (2026-09-08, gate A0-06)

Status of the four freeze blockers above:

1. **Codex isolation spike** — RESOLVED. A0-02 (`docs/tasks/v1.4/research/A0-02-findings.md`): per-run `CODEX_HOME` + `auth.json`-only handoff proven at mechanism level (leak reproduced under shared home; isolation candidate passes); real-CLI canary deferred to P0-03/P0-04, `session.isolation` remains undeclared until then.
2. **Assurance bake-off** — RESOLVED as conservative default. A0-03 (`A0-03-findings.md`): frozen corpus v2, L3\L1 = 7/12 unique catches, L2 judge NOT_RUN(no credentials). Default = L1 floor + L3 risk-adjusted uplift; L2 experimental only.
3. **Repair/held-out ADR** — ACCEPTED. ADR-0001 `docs/architecture/adr/ADR-0001-v14-closed-loop-repair.md`.
4. **Aggregate-contract compatibility in Decision Locks** — ALREADY RECORDED (lock 5) and reaffirmed; no physical merge of TaskContract + VerificationContract.

Gate verdict: see `docs/tasks/v1.4/reports/A0_ARCHITECTURE_GATE.md`.
