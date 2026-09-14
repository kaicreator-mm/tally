# A0-01 — Current Source-to-PRD Authority Map (v1.4.1)

- **Task:** A0-01 (phase A0, docs)
- **Status:** DONE
- **Source baseline verified against:** working tree at HEAD `3c1e983` (clean); SOURCE_BASELINE.md records the L2 archive baseline `979a55ad3ba5e514bb80f0e012aa36e0e6577eba` with v1.4 55/55 DONE and Final Gate `PASS_WITH_KNOWN_LIMITATIONS`.
- **PRD mapped:** `docs/product/Tally_PRD_v1.4.1_Final.md` (§1–§40, "Final / L2+L3 Aligned / Implementation Authorized").
- **Cross-reviewed against:** `docs/tasks/v1.4.1/L2_ARCHITECTURE_EVIDENCE.md`, `docs/tasks/v1.4.1/DECISION_LOCKS.md` (DL-01…DL-20), `docs/tasks/v1.4.1/PRD_AMENDMENTS.md` (A-01…A-10), `docs/tasks/v1.4.1/SOURCE_BASELINE.md`, `docs/tasks/v1.4/STATUS.md`.
- **Hidden material:** none read; `docs/tasks/v1.4.1/hidden/` untouched (DL-17).

Every owner below was verified in the current working tree on 2026-09-12. Status vocabulary:

- **reuse** — existing seam already owns this; v1.4.1 must not duplicate it.
- **change** — existing seam owns it, but a bounded v1.4.1 extension/wiring is required (the named gap is part of the v1.4.1 scope).
- **new** — no current owner exists; a new bounded object/package is required (only where the PRD/L2 authorizes one).
- **defer** — explicitly outside the v1.4.1 release gate (NEXT / ADR-gated).

---

## 1. Verified source-fact inventory (evidence base)

These are the load-bearing source facts this map rests on. Each was checked directly (file read or exact grep); the L3 arch tests pin several of them as machine checks.

| # | Source fact | Evidence |
|---|---|---|
| F1 | `internal/closedloop` owns Execute→Verify→Repair→Accept/Escalate: `engine.go` (phase machine, accept guard = `VerdictVerified` + exposure-compromise guard), `ports.go` (Executor/Verifier/Router/RepairBudget/LineageSink), `wiring.go` (`NewLoop`), `executor.go` (`WorkspaceExecutor`), `verify.go` (`ServiceVerifier`), `failure.go`, `feedback.go`, `repair_policy.go`, `exposure.go`, `lineage.go`, `phase.go`, `result.go`. | Read; `test/arch/v14_*` suite passes at baseline |
| F2 | Acceptance in the loop is driven by base verification status only: `verify.go` `verdictForStatus` maps `verification.FinalVerified → VerdictVerified`; `engine.go` accepts on `VerdictVerified` (+ RunRef + exposure guard). No `contract.Compiled`/assurance input exists in `LoopConfig` or `Engine`. | Read `internal/closedloop/engine.go`, `verify.go`, `wiring.go` |
| F3 | Production loop composition is `cmd/worker/v14.go`: loads `.tally` TaskContract + frozen VerificationContract from Git HEAD via `artifacts.Load`, selects runtime by config (`v13Runtime`, default `fake`), builds `v14CodegenRunner` over `agentruntime.Session.Execute`, freezes candidates via `changeset.New`, records facts via `workflow.RunSink`/`runstore`. Registered unconditionally in `cmd/worker/main.go:132`. | Read `cmd/worker/v14.go`, `cmd/worker/main.go` |
| F4 | `contract.Bind(...)` (and `Compiled` generally) has **no production worker call site**. Only callers: `scripts/v14_trial/tally/trialrun.go:385`, `scripts/v14_trial/tally/selfcheck.go:273` (trial harness, not `cmd/worker`) and tests. `contract.Compile`/`Open`/`Bind`/`Compiled.{Coverage,Clauses,FrozenGates,Requires,GateKeys}` all exist in `internal/contract/{contract.go,compile.go}`. | grep `contract.Bind(`; read `internal/contract/compile.go` |
| F5 | `V14ConfirmDraft` (`internal/web/handlers/v14_launch.go`) validates and re-renders the draft only; it does **not** write any contract state. Source comment: "草案→TaskContract/VerificationContract 的冻结链路…v1.4 尚未接线;本 handler 绝不伪造冻结". | Read `v14_launch.go:435-504` |
| F6 | Intent AI executor injection is a settable seam with no production caller: `SetV14ProposeExecutor` defined in `v14_launch.go`; grep over `cmd/` + `internal/` finds no composition-root call. `intake.Propose` consumes `airuntime.Executor`. | grep `SetV14ProposeExecutor` |
| F7 | Intake `Draft.RiskSignals` is explicitly candidate-only ("只是候选,不作数") and nothing persists it as an execution input; no `FrozenIntent`/`IntentEvidence` artifact exists anywhere. `TaskContract` carries no task_type/risk-hint fields consumed by risk derivation; `contract.Bind` requires a `risk.Result` supplied by the caller. | Read `internal/intake/intake.go`; grep `FrozenIntent` (0 hits) |
| F8 | `taskstate.VerifiedBasis` (`internal/taskstate/basis.go`) already derives `VERIFIED` / `VERIFIED_LIMITED` / `NOT_VERIFIED`, required vs achieved assurance (weakest link), unmatched risk signals, satisfied vs waived, residual risk — purely from `contract.Compiled` + observed verification facts. | Read `internal/taskstate/basis.go` |
| F9 | `artifacts.LoadPolicies` (`internal/artifacts/v14_policy_loader.go`) loads `.tally/policies/*.yaml` from a **pinned Git commit tree** (test-pinned: reads commit tree, not working tree). It has **no production consumer** outside its own tests. `engineeringpolicy.Policy` (`policy.go`) is a narrow runtime schema; revision derives from content; parser rejects ExecutionPolicy fields (machine-checked by `test/arch/v14_runtime_boundary_test.go`). | grep `LoadPolicies`; read loader + tests |
| F10 | Outcome domain is exactly five append-only event kinds (`accepted`, `reverted`, `hotfixed`, `regression_found`, `human_correction`): `internal/outcome/types.go`, migration `0023_v14_outcome_event.sql` (append-only trigger), sqlc surface. Machine-checked by `test/arch/v14_outcome_event_test.go`. The closed loop (`engine.go`/`cmd/worker/v14.go`) writes **no** outcome events today; no production lifecycle writer exists. | Read `internal/outcome/types.go`; grep `outcome.` in `cmd/worker` |
| F11 | `internal/agentruntime/protocol.go` is the single coding-runtime protocol: `Runtime` (Descriptor/Start), `Session` (Execute/Cancel/Events/Result/Close), `StartRequest{Workspace,ResumeOf,ModelProfile}`, `ExecuteRequest{Instruction}`, `AgentRunResult` (claim, INV-AUTH-01), `CapabilitySet`/`Negotiate` (`capability.go`), `ObservedIdentity`/`IdentityReporter` (`identity.go`). Control flow branches on capabilities, never runtime name. Implementations: `runtimes/codex`, `runtimes/fake` only. No `runtimes/dsh` exists. | Read `protocol.go`; `ls runtimes/` |
| F12 | AI Runtime boundary is `internal/airuntime` (`Executor`, `Request`, `Result` with `OperationID`/`TerminationReason`/`StrategyAssignment` — runtime-observed rollout facts). Mutual isolation from `agentruntime`/`runtimes/*`/`internal/llm` is machine-checked (`test/arch/v14_runtime_boundary_test.go` AC2a/AC2b). `internal/llm` is legacy, outside both sides of the coding path. | Read `internal/airuntime/result.go`; arch test |
| F13 | Git durability primitives exist: `internal/gitwriter` (expected-base commit, `ErrBaseRevision`, lock file; production consumers today: `internal/requirement/sign.go`, `internal/harvest/catalog.go`, `internal/knowledge/store.go`), `internal/workspace` (disposable worktree manager, refuses owner-checkout as worktree root), `internal/changeset` (immutable identity + supersession lineage). No L3 integration (branch/merge/conflict) service and no remote (L4) path exists. | grep `gitwriter.`; read package listings |
| F14 | Current contract-freeze production path is the requirement/CLI chain: `cmd/tally req` (lint/interview/sign) → `internal/requirement/sign.go` (atomic `gitwriter` commit of `.tally/requirements/*`); `internal/artifacts/v13_loader.go` defines the `.tally` layout (`project.yaml`, `tasks/TASK-*.yaml`, `verifications/TASK-*.yaml`) read from Git snapshots only. | Read `cmd/tally/req.go`, `internal/artifacts/v13_loader.go` |
| F15 | No code exists for: `DynamicForm*`, `RuleProposal`, `RuleReplayResult`, `ReviewDecision`, `PlanningResult`/`PlanningProvenance`, DSH adapter, safe-export packaging, L3 integration service. (grep over `internal/ cmd/ runtimes/`: 0 hits.) These are genuinely new v1.4.1 objects. | grep |
| F16 | Verification orchestration exists (`internal/verification/orchestrator.go`: `FinalVerified`/`FinalFailed`/`FinalNeedsReview`, holdout/visible executors, holdout path-leak test); production candidate verification service is `internal/candidateverify` (`service.go`, `holdout.go`, `sink.go`). | Package listing; `closedloop/verify.go` |
| F17 | v1.3 vertical path (`cmd/worker/v13.go`), measurement (`internal/measurement` incl. `v14_metrics.go`), evidence independence/exposure (`internal/evidence`), context packs (`internal/contextpack`), holdout (`internal/holdout`) are all present and are reuse seams, not rebuild targets. | Package listings |

Baseline validation: `go test ./test/arch/ -run 'TestV14' -count=1` → `ok tally/test/arch 0.506s` (Go 1.27.1 windows/amd64, exit 0). All L2-era "known production gaps" (SOURCE_BASELINE.md items 1–7) were re-confirmed as still-true source facts F3–F7, F10, and the export gap (§32 below).

---

## 2. PRD section → owner seam map

The table covers every section of `Tally_PRD_v1.4.1_Final.md`. "Owner" names the package/file that must remain the single authority for that requirement. Duplicate-risk markers (R#) refer to §4.

| PRD § | Requirement (short) | Owner seam | Status | Notes |
|---|---|---|---|---|
| §1 | Executive summary; five closures | narrative; realized via §14–§32 rows | — | No code owner. |
| §2 | Current code baseline | this map + `docs/architecture/CURRENT_CODEBASE.md` | reuse | Facts F1–F17. PRD §2 statements match source; no action. |
| §3.1 | Navigation ≠ state machine | `internal/taskstate` (state machine, untouched) + `internal/web/handlers/*` (projection) | reuse | DL-19. New nav is read-only projection over existing facts. R-UI. |
| §3.2 | Dynamic form never mutates Contract authority | `internal/taskcontract` + `internal/verification/freeze.go` stay sole contract authorities; form DTO lives with intake/UI layer | change | DL-11. R-FORM. |
| §3.3 | No second rule engine; Rules UI = EngineeringPolicy governance | `internal/engineeringpolicy` (runtime schema) + new governance objects (see §11–§13 rows) | change | DL-18, DL-08. R-RULE. |
| §3.4 | Harness replaces only the Execute primitive | `internal/closedloop` (loop authority) + `internal/agentruntime` (protocol) | reuse | DL-01, DL-02, DL-03. R-HARNESS. |
| §3.5 | DIRECT/AUTO/ORCHESTRATED outside Tally | `internal/airuntime` boundary seam only; Tally consumes results | reuse / defer | DL-20. PlanningResult consumption is §5.1/§28 scope. |
| §3.6 | Git control plane = incremental extension of gitwriter/workspace/changeset | `internal/gitwriter`, `internal/workspace`, `internal/changeset` | reuse (+ new L3 service later, see §18) | R-GIT. |
| §3.7 | Rule evidence comes from OutcomeLog et al., no second case history | `internal/outcome` + `internal/runstore` facts | reuse | R-OUTCOME (no parallel CaseHistory store). |
| §4 | Product definition | narrative | — | — |
| §5.1 | AI Runtime responsibilities; Tally consumes PlanningResult | `internal/airuntime` (boundary); **new** bounded PlanningResult/Provenance record at consumption point | new (bounded) | DL-20: do not implement planner modes. |
| §5.2 | Tally responsibilities (intake…Outcome) | distributed: this table §15–§27 rows | — | — |
| §5.3 | Harness responsibilities | `runtimes/codex`, **new** `runtimes/dsh` | reuse + new | DL-14/15/16. Out-of-process adapters only. |
| §6 | UX mental model, nav list | `internal/web/handlers` + `web/templates` (new projection pages) | new (projection) | DL-19: no state-machine rewrite; no SPA rewrite. |
| §7 | Work UI task detail | `internal/web/handlers/v14_task.go`, `v14_change.go` (exist: detail/change/attention) extended with plan/verification/review projections | change | Read-only over taskstate/runstore facts. |
| §8 | LLM dynamic forms | `internal/intake` (Draft/Clarification exist) + **new** bounded `DynamicFormDefinition` DTO + schema validation, surfaced via `internal/web/handlers/v14_launch.go` | change + new | DL-11: bounded DSL, no HTML/JS, no domain schema. Seed exists: `V14QuestionView`, `v14FoldClarifications`, `v14DraftFromForm`. |
| §8.3 | Advisory generated choices | same as §8; recommendation stays advisory in DTO | new (bounded) | Server-side validation mandatory (L2 F4). |
| §8.4 | Form schema constraints | validator in intake/UI seam; contract schema untouched | new (bounded) | Machine-checkable: reject arbitrary markup fields. |
| §9 | Rule-first review evolution | governance layer (§11–§13) over `internal/engineeringpolicy` | change | No new execution rule language (DL-18). |
| §10 | Reviews vs Rules pages | `internal/web/handlers` (Reviews = existing attention/escalation facts; Rules = new governance projections) | new (projection) | Reviews read `closedloop` escalation/runstore facts; no new authority. |
| §11 | Rule proposal display + actions | **new** `RuleProposal`/`RuleVersion`/`RuleEvidence`/`RuleApproval` governance objects (new store/queries + handlers) | new | Governance facts are separate from `engineeringpolicy.Policy` (DL-08). Activation = `gitwriter` commit of canonical YAML (F13). |
| §12 | Historical replay / dry run | **new** `RuleReplayResult` projection computed from recorded facts (`outcome`, runstore, changeset lineage) + `engineeringpolicy.Match` | new | DL-10: evidence-bounded; `INSUFFICIENT_EVIDENCE`/`LOW_EVIDENCE` fail-safe; no counterfactual outcomes. |
| §13 | Exception → Rule ("Apply Once" / "Apply and Propose Rule") | ReviewDecision (new) + RuleProposal (new); evidence from outcome/runstore | new | Does not activate policy directly (PRD invariant). |
| §14 | Production closure chain end-to-end | `internal/web/handlers/v14_launch.go` → `internal/intake` → freeze path (F14) → `cmd/worker/v14.go` → `internal/closedloop` → `internal/outcome` | change | The §15/§16/§27 gaps below are exactly this chain's missing links. |
| §15 | Intent / Confirm / Freeze + contract identity | `internal/web/handlers/v14_launch.go` (`V14ConfirmDraft`), `internal/intake`, `internal/taskcontract` + `internal/verification/freeze.go`, artifacts layout `internal/artifacts/v13_loader.go` | change | Gaps: F5 (Confirm does not freeze), F6 (executor not injected). Invariant "confirmed UI state ≠ executable task" already holds (launch only enqueues frozen tasks, `V14LaunchRun` guards). |
| §15.1 | FrozenIntent/IntentEvidence immutable intake facts (L2 addendum, PRD A-02) | **new** minimal immutable companion artifact persisted at Confirm/Freeze; consumed by risk derivation at compile time | new | DL-05/DL-06. Not a mutable RiskProfile. Fills gap F7. Exact path/schema deliberately unfrozen (L2 §7). |
| §16 | Compiler authority: worker must consume CompiledContract | `internal/contract` (`Compile`/`Open`/`Bind`, all exist) wired into `cmd/worker/v14.go` + `internal/closedloop` (verifier/accept guard inputs) | change | Gap F4: Bind has zero production call sites (only `scripts/v14_trial/*`). No-bypass tests required (PRD Phase 2). R-COMPILE (don't fork a second compiler). |
| §17 | EngineeringPolicy evolution & governance metadata | runtime schema: `internal/engineeringpolicy/policy.go` (stays narrow; identity = existing `ID` + content-derived `Revision`); governance metadata (status DRAFT/PROPOSED/ACTIVE/SUPERSEDED/REJECTED, evidence_refs, approved_by, supersedes, scope): **new** governance objects from §11 | change (governance) | **Conflict resolved per DL-08/A-04:** lifecycle/status/evidence fields must NOT be added to `engineeringpolicy.Policy` YAML/struct. "Runtime consumes ACTIVE" = the policy blob in the pinned Git snapshot (`artifacts.LoadPolicies`, F9). See §4 R-POLICY. |
| §18 | Git Control Plane L0–L5 | L0/L1/L2: `internal/gitwriter` + `internal/workspace` + `internal/changeset` (exist, F13). L3 controlled integration: **new** integration service composing those (owner to be fixed by its own task). L4: optional capability-gated adapter. L5: human approval, not automated | reuse + new(L3) + defer(L4/L5) | DL-13: release gate ends at L3. R-GIT. |
| §19 | Harness abstraction: `agentruntime.Runtime` is the only protocol | `internal/agentruntime/protocol.go` + `capability.go`; adapters in `runtimes/*` | reuse | DL-02. R-BACKEND: the PRD draft `ExecutionBackend` is deleted (A-01). |
| §20 | Execution request reuse | `agentruntime.StartRequest`/`ExecuteRequest` (exist) | reuse | Hidden/held-out material must not enter ExecuteRequest (existing INV; `v14Instruction` renders only safe feedback projection, F3). |
| §21 | Execution result = claim; ChangeSet by freezer | `agentruntime.AgentRunResult` + `cmd/worker/v14.go` `v14CodegenRunner` (changeset freeze) | reuse | "done ≠ accepted" invariant machine-grounded (INV-AUTH-01). Provenance: observed identity optional-only (`identity.go`), never desired==actual. |
| §22 | Codex primary adapter | `runtimes/codex` | reuse | Preserve + prove no regression; composition normalization allowed. Cross-provider subagents not a gate (DL-14). |
| §23 | DeepSeek Harness adapter (secondary/canary) | **new** `runtimes/dsh` implementing `agentruntime.Runtime/Session`; exact-version Spike gate first | new + defer(gate) | DL-15. No DSH Git/verification/acceptance authority; out-of-process bridge allowed. F15: nothing exists yet. |
| §24 | Harness routing by capability, not name | `internal/agentruntime/capability.go` (`Negotiate`, `CapabilitySet`); composition root selects adapter | reuse | DL-16. DSH-internal capabilities stay adapter-local (PRD §24 caution). |
| §25 | One primary harness per task | `internal/closedloop/engine.go` (single Executor port) + `cmd/worker/v14.go` (one runtime session per attempt) | reuse | Cancellation convergence already enforced (engine ctx discipline). |
| §26 | Verification hardening / migration false-negative taxonomy | `internal/verification` (orchestrator, qualification, holdout/visible executors) + `internal/candidateverify` | change | Harness-local tests remain evidence-only. Classification taxonomy work is owned here; no second verifier. |
| §27 | Outcome production closure | `internal/outcome` (5 kinds, F10) + **new production writers** wired from `internal/closedloop` terminal states via `cmd/worker/v14.go`; rejected/escalated/integration-failed as workflow/attention/integration facts (`internal/runstore`, `internal/web/handlers/v14_attention.go`) | change | **Conflict resolved per DL-12/A-06:** the Final PRD §27 sentence lists 8 lifecycle labels; the Outcome *domain* stays the five existing kinds. accepted→`KindAccepted`, reverted→`KindReverted`, human_corrected→`KindHumanCorrection`, hotfix_required→`KindHotfixed` (hotfix event), regression_detected→`KindRegressionFound`; rejected/escalated/integration_failed remain workflow facts unless a future ADR changes the Outcome domain. See §4 R-OUTCOME. |
| §28 | Data/domain changes; minimal new objects | reuse: TaskContract, VerificationContract, CompiledContract, ChangeSet, EngineeringPolicy, Outcome, Workspace, execution state (all exist, F1–F17). New: `DynamicFormDefinition`, `PlanningProvenance`, `RuleProposal`, `RuleReplayResult`, `ReviewDecision`, FrozenIntent (§15.1). Runtime identity: reuse `agentruntime.RuntimeDescriptor`/`ObservedIdentity` (F11) | as marked | Forbidden duplicates (second Task/Contract/Policy engine/Closed Loop) are Gate A + §4 risks. |
| §29 | UI information architecture | `internal/web/handlers` + `web/templates` (new pages; state machine untouched) | new (projection) | DL-19. |
| §30 | UI design rules | same as §29 (projection discipline incl. recommendation evidence links) | new (projection) | — |
| §31 | Failure taxonomy | `internal/closedloop/failure.go` (`FailureClass`, `ClassifyFinalStatus`, `ClassifyError`) + `internal/airuntime/result.go` (`TerminationReason`); IntegrationFailure realized with the L3 service (§18) | reuse + change | No second taxonomy; do not fabricate provider/cancel errors into candidate failures (existing engine discipline). |
| §32 | Safe export | **new** export packaging that excludes `.git/`, `.worktrees/`, `.e2e/`, `.codegraph/`, credentials, binaries, caches, local execution artifacts; must not rely on `.gitignore` | new | Gap confirmed: no export code exists today (F15); whole-worktree archive currently leaks local/runtime artifacts (SOURCE_BASELINE gap 7). |
| §33 | Non-goals | `DECISION_LOCKS.md` + this map | — | Enforced by review, not code. |
| §34 | Implementation phases 0–8 | task pack `docs/tasks/v1.4.1/tasks/{A0,P0..P6}` | — | Phase 0 = this task pack. |
| §35 | Release gates A–J | Gate A: `test/arch/v14_*` (existing) + new no-duplication guards per §4. Gates B–J: owned by the phase tasks implementing §14/§16/§27/§29/§11-13/§19-25/§18/§26/§32 | — | Each gate's proof obligation lands in the owning task; no gate may be marked PASS without its command (AGENTS §9). |
| §36 | Acceptance scenarios 1–6 | e2e layer `test/e2e/` (integration-tagged), composed from the §15/§16/§18/§27/§11-13 seams | — | Scenario 6 (base drift) reuses existing expected-base rejection (`gitwriter.ErrBaseRevision` discipline). |
| §37 | Metrics | `internal/measurement` (incl. `v14_metrics.go`) + read-side projections | change | Engineering metrics partially exist; UX/rule-governance metrics defer to their features' tasks. |
| §38 | Definition of Done | §35 gates + this map as the code-to-PRD correspondence record | — | Items 1–15 all resolve to rows above. |
| §39 | Final architecture diagram | realized by rows §5–§28 | — | — |
| §40 | Product statement | narrative | — | — |

**Ambiguity check (AC-1):** every PRD section above carries exactly one owner seam or is explicitly narrative/deferred. The two literal-text conflicts (§17 vs DL-08, §27 vs DL-12) are resolved above by the Decision Locks + PRD_AMENDMENTS (which the PRD itself declares incorporated) and are flagged as risks R-POLICY / R-OUTCOME so no implementing agent can read the raw PRD sentences as license to duplicate architecture.

---

## 3. L2 amendments → source rationale

| Amendment | Decision | Source rationale (verified facts) |
|---|---|---|
| A-01 Runtime abstraction | Delete generic `ExecutionBackend`; reuse `agentruntime.Runtime`; DSH implements an adapter | F11: protocol with Descriptor/Start/Execute/Cancel/Events/Result + capability negotiation already exists and is conformance-guarded; `runtimes/codex` implements it. A parallel backend would create lifecycle/cancel/result ownership ambiguity (L2 F5). Machine guard: `test/arch/v14_runtime_boundary_test.go` AC2a. |
| A-02 FrozenIntent evidence | Add minimal immutable intake-facts artifact; never persist mutable `risk.Result` | F7: `Draft.RiskSignals` is candidate-only and dies at render time; `TaskContract` carries no risk hints; `contract.Bind` needs a `risk.Result` but production has no trustworthy input (F4). `risk.Result` is derived-only (`internal/risk/derive.go`), DL-05. |
| A-03 Acceptance authority | `FinalVerified` alone insufficient; `VERIFIED_LIMITED` must not become Accepted | F2: engine accepts on `VerdictVerified` mapped from `FinalVerified` with no Compiled/assurance input. F8: `VerifiedBasis` already computes required-vs-achieved assurance and unmatched risk, so the guard has a seam to consume — it is just not wired into the loop. |
| A-04 Rule governance | Governance lifecycle outside `engineeringpolicy.Policy`; ACTIVE = policy blob in pinned Git snapshot | F9: `Policy` schema is narrow with content-derived revision; parser rejects ExecutionPolicy vocabulary (machine-checked); `artifacts.LoadPolicies` already implements Git-snapshot authority but is unwired. DB-side status fields would split policy authority (L2 F3). |
| A-05 Historical replay | Replay is evidence-bounded; no fabricated counterfactual outcomes | Outcome/runstore/changeset facts are append-only event/lineage records (F10, F13); counterfactual code/test outcomes have no recorded basis, and inventing them would violate the evidence invariants (DL-10). |
| A-06 Outcome semantics | Keep exactly five kinds; rejected/escalated/integration failure stay workflow facts | F10: closed enum enforced at three layers (domain constants, migration CHECK, append-only trigger) with fail-safe `ErrUnknownKind`; widening would require an ADR and would corrupt rule-learning evidence (L2 F10). |
| A-07 Git scope | v1.4.1 = L0–L3; remote is optional/NEXT | F13: L0–L2 primitives exist and are proven; L3 service and L4 adapter do not exist; scope explosion would reopen credential/security surface (L2 F9). |
| A-08 DSH maturity | Secondary/canary; no protected-branch or durable Git authority | F11/F15: DSH adapter does not exist; L2 external evidence (DSH SAFETY.md: developer preview, not a security control). Tally-owned isolation already required by protocol (`StartRequest.Workspace` must be disposable worktree). |
| A-09 UI as projection | No state-machine rewrite; dynamic forms are presentation-only | F5/F14: lifecycle authority is derived state + Git-frozen contracts; UI handlers today are read-only fact projections plus two bounded write exits (enqueue, none for freeze). DL-19/DL-11. |
| A-10 AI Runtime planning | Tally consumes unified planning result/provenance only | F12: `airuntime.Result` already carries runtime-observed `OperationID`/`TerminationReason`/`StrategyAssignment`; boundary isolation is machine-checked. Implementing DIRECT/AUTO/ORCHESTRATED inside Tally would duplicate the planner (DL-20). |

---

## 4. Duplicate-architecture risks (explicit register)

Gate A (PRD §35) makes "no second X" a release gate. Each risk below names the temptation, the existing authority it would duplicate, and the guard.

| ID | Risk (do not build…) | Existing single authority | Guard |
|---|---|---|---|
| R-BACKEND | A new generic `ExecutionBackend`/core runtime protocol alongside `agentruntime` | `internal/agentruntime/protocol.go` (F11) | DL-02; arch test AC2a; A-01 |
| R-LOOP | A second orchestration/repair loop or moving Verify/Repair/Accept into adapters | `internal/closedloop` (F1) | DL-01; adapters only replace the Execute primitive (PRD §3.4) |
| R-COMPILE | A second "compiler" or letting the worker execute uncompiled contracts / re-deriving policy matches ad hoc | `internal/contract` (`Compile`/`Open`/`Bind`) (F4) | §16 wiring task + no-bypass tests; trial-script callers in `scripts/v14_trial/*` are not production authority |
| R-POLICY | Governance status/lifecycle fields inside `engineeringpolicy.Policy` YAML/struct, or a DB "ACTIVE" flag split from Git content | `internal/engineeringpolicy` + `artifacts.LoadPolicies` Git-snapshot authority (F9) | DL-08, DL-09; A-04. **PRD §17 must be read through this lock** |
| R-OUTCOME | Adding rejected/escalated/integration_failed kinds to OutcomeLog, or a mutable outcome status column, or a parallel CaseHistory store | `internal/outcome` 5-kind closed enum + append-only trigger (F10); workflow facts live in runstore/attention | DL-12; `test/arch/v14_outcome_event_test.go` AC1/AC5; **PRD §27 must be read through this lock** |
| R-RISK | A mutable RiskProfile/state table as risk truth, or persisting derived `risk.Result` | `internal/risk` derived-only + FrozenIntent as the immutable *fact* input (A-02) | DL-05, DL-06 |
| R-CONTRACT | A second contract domain or LLM/dynamic-form-driven contract schema | `internal/taskcontract` + `internal/verification/freeze.go` (F14) | PRD §3.2; DL-11 |
| R-GIT | A new `internal/git` mega-subsystem re-implementing commit/worktree/lineage | `internal/gitwriter` + `internal/workspace` + `internal/changeset` (F13) | PRD §3.6; DL-03; L3 service must compose, not relocate |
| R-HARNESS | Harness adapters owning retry/verification/acceptance, or DSH receiving durable Git authority | closedloop (verdict/accept) + gitwriter (durable writes) (F1, F13) | DL-03, DL-15; PRD §23 prohibitions |
| R-AIRUNTIME | Re-implementing the AI Runtime planner (DIRECT/AUTO/ORCHESTRATED) or crossing the airuntime/agentruntime boundary | `internal/airuntime` seam + arch isolation test (F12) | DL-20; arch tests AC2a/AC2b |
| R-UI | State-machine rewrite, SPA/framework rewrite, chat-first IA, arbitrary LLM HTML/JS | taskstate derived state + handlers-as-projections (F5, F14) | DL-19, DL-11; PRD §3.1/§33 |
| R-IDENTITY | A second runtime identity model (desired vs observed conflation) | `agentruntime.RuntimeDescriptor`/`ObservedIdentity`; `airuntime.Result.Strategy` (F11, F12) | AGENTS §13; PRD §28 explicitly says reuse |
| R-EXPORT | Treating `.gitignore`/worktree archiving as safe export | new export packager (§32) is the only sanctioned path | SOURCE_BASELINE gap 7 |
| R-SPLIT-BRAIN | Loading task/verification from commit A while policies/intent evidence come from B | `artifacts.Load`/`LoadPolicies` are rev-pinned; all v1.4.1 wiring must resolve from one pinned commit | L2 F1; DL-04, DL-09 |

---

## 4.1 Explicitly deferred (not v1.4.1 gates)

- L4 remote (fetch/push/PR/MR), L5 destructive automation — DL-13, PRD §18.
- Native Codex cross-provider subagent routing — DL-14.
- AI Runtime planning modes — DL-20.
- WorkItem identity rewrite (outcome events keep `task_id`) — DL-13 note in `internal/outcome/types.go`.
- Recursive harness trees (Tally→DSH→Codex→subagent) — PRD §25, ADR-gated.
- Outcome-domain expansion beyond five kinds — DL-12, ADR-gated.

---

## 5. Review against L2 evidence and Decision Locks

### 5.1 Decision Lock cross-reference (DL-01…DL-20, all accounted for)

| Lock | Where it binds this map |
|---|---|
| DL-01 closedloop authority | §2 rows §3.4/§14; R-LOOP |
| DL-02 single runtime protocol | §2 rows §19/§5.3; R-BACKEND; A-01 |
| DL-03 runtimes own no Git durability/acceptance | §2 rows §18/§21/§23; R-GIT, R-HARNESS |
| DL-04 one pinned source commit | §4 R-SPLIT-BRAIN |
| DL-05 risk.Result stays derived | §4 R-RISK; A-02 |
| DL-06 confirmed intake facts survive Freeze | §2 row §15.1 (FrozenIntent) |
| DL-07 insufficient assurance never Accepted | §2 row §16 + A-03 (assurance/acceptance guard; guard placement deliberately open, §5.3) |
| DL-08 no governance fields in Policy struct | §2 row §17; R-POLICY; A-04 |
| DL-09 runtime-active policy = pinned Git snapshot content | §2 row §17; R-POLICY, R-SPLIT-BRAIN |
| DL-10 replay is evidence-bounded | §2 row §12; A-05 |
| DL-11 bounded presentation schema | §2 rows §3.2/§8/§8.4; R-CONTRACT, R-UI |
| DL-12 five Outcome kinds | §2 row §27; R-OUTCOME; A-06 |
| DL-13 Git scope ends at L3 | §2 row §18; §4.1 deferred |
| DL-14 Codex primary; no cross-provider subagent gate | §2 row §22; §4.1 |
| DL-15 DSH secondary/canary, Tally-owned isolation | §2 row §23; R-HARNESS; A-08 |
| DL-16 one primary runtime owner per task | §2 rows §24/§25 |
| DL-17 hidden material isolation | header note; §5.3; nothing under `hidden/` read |
| DL-18 extend, don't duplicate Task/Contract/Policy/Outcome | §2 rows §3.3/§9/§28; R-RULE, R-CONTRACT, R-OUTCOME |
| DL-19 navigation is projection only | §2 rows §3.1/§6/§29; R-UI |
| DL-20 AI Runtime planning modes out of scope | §2 rows §3.5/§5.1; R-AIRUNTIME; A-10 |

### 5.2 L2 verdict coverage

- Every L2 verdict row (L2_ARCHITECTURE_EVIDENCE.md §2) is represented either as a `reuse` row (Closed Loop, worktree/ChangeSet, Harness abstraction, Codex, UI projection, Git scope, AI Runtime) or a `change`/`new` row with the named gap (Confirm→Freeze, Compiler wiring, assurance guard, FrozenIntent, dynamic form, rule governance, replay limits, outcome closure, safe export). No L2 verdict is contradicted by this map.
- Every DL-01…DL-20 is cited in §2/§4 where it binds an owner. DL-17 (hidden material) honored: nothing under `docs/tasks/v1.4.1/hidden/` was read or listed.
- The two PRD-text/lock tensions (§17 vs DL-08, §27 vs DL-12) are resolved with the locks as the binding interpretation, consistent with PRD_AMENDMENTS.md's statement that the amendments are incorporated into the Final PRD; both are carried as risks R-POLICY and R-OUTCOME so downstream tasks inherit the resolution rather than re-litigating it.
- Deliberately left unfrozen (per L2 §7, so this map does not over-specify): FrozenIntent exact file path/schema; acceptance-guard exact placement (Engine vs ServiceVerifier — both named as candidate seams, decision belongs to the guard task with tests); DSH transport shape; replay storage; L3 integration service API.

## 6. Validation

- `go test ./test/arch/ -run 'TestV14' -count=1`: PASS (`ok tally/test/arch 0.506s`, exit 0, Go 1.27.1 windows/amd64) — run before writing this map, as the L3 pack requires.
- `grep` fact checks listed in §1 (contract.Bind call sites; SetV14ProposeExecutor callers; LoadPolicies consumers; FrozenIntent/DynamicForm/RuleProposal/PlanningResult existence): executed, results as recorded.
- No production code changed; no tests added (docs-only task; "new behavior needs focused tests" is N/A — no behavior introduced).
