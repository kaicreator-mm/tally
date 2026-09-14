# Current Codebase Baseline

**HEAD audited:** `979a55ad3ba5e514bb80f0e012aa36e0e6577eba`  
**Audit date:** 2026-09-11  
**Meaning:** source facts before v1.4.1 implementation.

**v1.4.1 current state (2026-09-13):** 56/57 tasks DONE (`docs/tasks/v1.4.1/STATUS.md` is the live progress authority). All v1.4.1 lanes have landed and gated: P0 authority chain, P1 compiled contract + risk-adjusted acceptance + no-bypass gate, P2 human-centered UX (gate PASS), P3 rule governance (gate PASS — no split-brain), P4 DSH canary adapter (gate LIMITED_READY), P5 local Git integration control plane (gate PASS), P6 Outcome/export/migration-fix. P6-08 (final hidden validation) remains with the external process (hidden pack withheld per DL-17).

## 1. Repository technology

- Go 1.23
- PostgreSQL
- pgx/v5 + sqlc + goose
- net/http + chi
- Go templates + HTMX
- Git CLI/go-git where appropriate
- no frontend build pipeline
- architecture dependency allow-list tests

## 2. v1.4 status

v1.4 is implemented and final-gated.

`docs/tasks/v1.4/STATUS.md` records 55/55 DONE.

`docs/tasks/v1.4/reports/V1.4_FINAL_GATE.md` records:
`PASS_WITH_KNOWN_LIMITATIONS`.

## 3. Core runtime topology

```text
cmd/server
cmd/worker
cmd/tally

internal/agentruntime
internal/closedloop
internal/workspace
internal/gitwriter
internal/changeset
internal/contract
internal/engineeringpolicy
internal/risk
internal/intentevidence
internal/intakefreeze
internal/assurance
internal/verification
internal/taskstate
internal/outcome
internal/gitintegration
internal/policygovernance

runtimes/fake
runtimes/codex
runtimes/dsh
```

### Coding Runtime

`internal/agentruntime.Runtime/Session` is Tally's runtime-neutral core protocol.

Current Codex implementation is **out-of-process**:
`runtimes/codex` spawns/controls Codex CLI processes.

`cmd/worker/v14.go` bridges:
`closedloop.CodegenRunner → v14CodegenRunner → agentruntime.Runtime`.

Runtime output is an Agent Claim. Tally separately freezes ChangeSet, verifies it and determines Acceptance.

A second out-of-process adapter exists: `runtimes/dsh` (DeepSeek Harness, developer-preview). It is **canary-gated and disabled by default** (P4-06: runtime=dsh + TALLY_DSH_CANARY=1 + explicit TALLY_DSH_BIN; limited-readiness verdict LIMITED_READY per P4-07 — see `docs/guides/dsh-canary.md`). Same Runtime/Session protocol, one Execute = one `--profile headless` process, pinned upstream version with a live `--version` identity gate.

## 4. Closed Loop

Current source has a real outer loop:
`Execute → Verify → Repair → re-Verify → Accept/Escalate`.

Disposable worktrees and immutable ChangeSets are production boundaries.

## 5. v1.4.1 gaps visible in source

- ~~`V14ConfirmDraft` remains a confirm/render boundary rather than full artifact freeze.~~ RESOLVED (v1.4.1 P0-08): `V14ConfirmDraft` is now the production confirm→freeze path — `intakefreeze.Freeze` admission followed by `artifacts.FreezeCommit`, which renders the frozen triple (`.tally/intents/<task_id>.yaml`, `.tally/tasks/<task_id>.yaml`, `.tally/verifications/<task_id>.yaml`) and commits them through exactly one `gitwriter.Writer.Commit` at an explicit expected base (one CAS boundary; stale base → typed `gitwriter.ErrBaseRevision`, never silently rebased; duplicate confirm → content-idempotent `gitwriter.ErrNoChanges`; mid-apply failure rolls back with zero artifact residue). Successful confirms are reloadable from the pinned commit via `artifacts.Load`/`artifacts.LoadIntent` (hash/revision equality test-pinned). No composition-root change was needed: the confirm route is already production-mounted and the repo/commit identity come from the injected `Server.Content`.
- ~~`contract.Bind` is implemented but not authoritative in the normal worker composition path.~~ PARTIALLY RESOLVED (v1.4.1 P1-03): the production compilation seam landed as `(*artifacts.ExecutionBundle).CompileAuthority() → *artifacts.CompiledAuthority` (`internal/artifacts/v141_contract_assembler.go`) — the single entry from the same-source bundle (P1-01 `LoadBundle`) to the authoritative compiled contract, composing the existing authorities without a parallel subsystem (DL-18): source-commit consistency re-check at the assembly entry (`ErrSourceMismatch`), policy-loader rejections fail-safe (`ErrPolicyRejections` blocks execution — the P1-01 deferred adjudication), `DeriveRisk` typed errors pass through verbatim (invalid risk basis stops execution), then `contract.Compile` + `contract.Bind` unchanged (DL-04: pure projection, never rewrites authority); unmatched/unknown risk does not block — INV-V14-09 semantics preserved (visible coverage + L3 failsafe uplift); zero-policy bundles compile as `ConfidenceNone` (L1 floor when no signals). The result exposes task id, observed source commit, bundle identity and the authoritative `risk.Result` next to `Compiled.Hash/Pins/Coverage/Clauses/GateKeys`. The worker does not call it yet — call-site timing (compile before workspace/runtime start, no-session-on-failure) is P1-04's owned wiring.
- Intake risk candidates do not yet form immutable execution facts in the production freeze path; the immutable evidence domain itself landed since this audit as `internal/intentevidence` (v1.4.1 P0-01: `.tally/intents/<task_id>.yaml` artifact schema, strict fail-closed parse, content-derived `sha256:` revision), the pinned-commit Git loader landed as `artifacts.LoadIntent` (v1.4.1 P0-02: commit-tree-only read, typed not-found/invalid/identity-mismatch rejections), the pure confirmed-draft→frozen mapper landed as `intakefreeze.Map` (v1.4.1 P0-06: `ConfirmedDraft → {FrozenIntent, TaskContract}`, deterministic, fail-safe typed rejection of unrepresentable draft semantics, confirmed risk hints only), and verification authoring/freeze admission landed as `intakefreeze.Freeze(ConfirmedDraft, VerificationPlan) → FrozenArtifacts` (v1.4.1 P0-07: owner-declared planned evidence per acceptance clause → frozen `verification.Frozen` via `verification.FreezeContract` at zero-value `FreezePolicy`; typed fail-safe admission — orphan declaration `ErrUnknownEvidenceTarget`, missing evidence `ErrMissingEvidence`, held-out without declared runtime isolation `ErrHeldOutWithoutIsolation`, authority-validation failures `ErrInvalidVerification`; held-out visibility only from explicit declaration and never auto-labeled; authoring provenance structurally fixed to `pre_commit`). and the confirm→freeze Git transaction landed as `artifacts.FreezeCommit` (v1.4.1 P0-08: renders `intakefreeze.FrozenArtifacts` into the three `.tally` artifact paths and commits them via exactly one `gitwriter.Commit` at an explicit expected base — single CAS boundary, cross-artifact identity gate `ErrFreezeIdentityMismatch`, stale base conflicts instead of rebasing, content-idempotent duplicate confirms, gitwriter rollback inheritance) wired behind `V14ConfirmDraft` with PRG flash outcomes (frozen sha / idempotent / conflict / typed failure). The same-source execution bundle loader landed as `artifacts.LoadBundle` (v1.4.1 P1-01, `internal/artifacts/v141_bundle_loader.go`): resolves the revision to exactly one commit and loads FrozenIntent (shared tree helper with `LoadIntent`), TaskContract, frozen VerificationContract and EngineeringPolicies (existing `LoadPolicies` rejection discipline) from that single commit tree (DL-04; no second resolution, no auto-upgrade to HEAD), with typed missing/invalid/identity-mismatch rejections per artifact class and a deterministic bundle `Identity()` over the observed source commit plus all loaded revisions; the authoritative risk input derivation landed as `intentevidence.FrozenIntent.DeriveRisk()` (v1.4.1 P1-02, `internal/intentevidence/riskderive.go`): pure derivation of the authoritative `risk.Result` from the frozen intent only — confirmed risk hints via the existing `RiskSignalCandidates` (basis `intent:<task-id>`) plus the deterministic task-type fact via `TaskTypeRiskCandidates` (`db_migration` ⇒ `schema_migration`, basis `tasktype:<task-id>`; `bug_fix` ⇒ none) — delegated verbatim to `risk.Derive` (classification stays in the risk package, DL-05; unknown hints stay visible, INV-V14-09; zero-value intent rejects `ErrInvalidEvidence`; missing basis fails the whole derivation `risk.ErrMalformedSignal`; nothing persisted); production consumption assembly landed as `(*artifacts.ExecutionBundle).CompileAuthority()` (v1.4.1 P1-03, `internal/artifacts/v141_contract_assembler.go`): the bundle → authoritative-compiled-contract seam with fail-safe rulings — policy-loader rejections block (`ErrPolicyRejections`), source mismatch blocks (`ErrSourceMismatch`), `DeriveRisk` typed errors stop execution, `contract.Compile`/`Bind` reused unchanged; worker wiring landed (v1.4.1 P1-04: `runV14Task` loads `artifacts.LoadBundle(repo, "HEAD", taskID)` and calls `bundle.CompileAuthority()` as the first execution gate — before runtime selection, held-out enforcement and any runtime session, failures typed pre-execution errors that never become candidate failures; v1.4.1 P1-05: the authority's `Compiled.Requires()` prevention context enters every iteration instruction and the compiled-aware `ContextPackHash`, so repair/replay cannot silently swap authority; v1.4.1 P1-06: the same authority feeds the acceptance guard).
- ~~taskstate can represent limited assurance, but acceptance needs a no-bypass compiled/assurance guard.~~ RESOLVED (v1.4.1 P1-06): acceptance is now risk-adjustment-gated in the Closed Loop Engine — `closedloop.Config.AcceptGuard` is consulted on the `VerdictVerified` accept transition (after the P2-06 exposure guard, before `EventVerified`); denial escalates with the typed reason `assurance_insufficient` (never Accepted, never a candidate failure, never repair budget). The single semantic evaluator is the pure `taskstate.EvaluateAcceptanceGuard` (required levels from `contract.Compiled.Clauses()`, achieved via `assurance.Achieved` over the projected functional coverage, VERIFIED_LIMITED triggers identical to the `VerifiedBasis` projection: assurance gap, unmatched risk signals, waived clauses — all fail-safe, missing coverage observation = no facts = L0). The production composition `closedloop.NewLoop` wires a `CompiledAcceptanceGuard` whenever `LoopConfig.Authority` (`*contract.Compiled`, the P1-04 mandatory product) is present, and `cmd/worker/v14.go` always passes it, so `verification.FinalVerified` no longer authorizes acceptance by itself (PRD §13/A-03/DL-07); `LoopConfig.Authority == nil` (test/trial compositions without a compiled authority) keeps v1.4 behavior, with the production no-bypass gate owned by P1-08. `ServiceVerifier` remains a fact-passthrough (`VerifyOutcome.Coverage` is a defensive projection of `FinalOutcome.Functional`); the verdict mapping `verdictForStatus` is unchanged — the spike adjudication (Engine vs ServiceVerifier) is recorded in `docs/tasks/v1.4.1/reports/P1-06_risk_adjusted_acceptance_guard.md`.
- ~~EngineeringPolicy runtime schema exists; rule governance/proposal/replay UI does not.~~ RESOLVED (v1.4.1 P3-01..P3-09): governance proposal domain + append-only event store (0024, `action_kind` CHECK) + candidate canonicalization/verification + conflict analysis + evidence-bounded historical replay + approve/activate via GitWriter + Rules UI projection (`/v14/rules`, Active derived only from the Git snapshot) + apply-once-vs-propose-rule review action; the P3-09 split-brain gate PASSED (runtime authority stays the Git snapshot; governance events are bookkeeping).
- ~~no `runtimes/dsh` adapter exists yet.~~ RESOLVED (v1.4.1 P4-03..P4-07): the DSH adapter landed (P4-03 spike GO-WITH-CONDITIONS, P4-04 skeleton with honest capability set + pinned-version identity gate + kill-tree, P4-05 lifecycle matrix with zero-orphan cancellation/timeout, P4-06 canary registration disabled-by-default, P4-07 gate verdict LIMITED_READY — live content production is a documented limitation).
- ~~Outcome domain/store exists; lifecycle writers are incomplete.~~ RESOLVED (v1.4.1 P6-01/P6-02): the accepted Outcome writer attaches inside the Closed Loop Engine accept transition (append-only, idempotent, fail-safe `outcome_unavailable` escalation), and the four remaining kinds have typed capture APIs with documented authoritative producers (production wiring of those producers is a known gap recorded for the final validation).
- ~~migration verification disagreement remains an open product-trust issue.~~ RESOLVED (v1.4.1 P6-04/P6-05): root cause reproduced deterministically (trial fixture as trigger + verifier semantics as amplifier) and fixed on both surfaces (opt-in infra-exit-code channel yielding needs_review instead of fabricated FAIL; compose.go gate-key→probe-subcommand mapping), exit-1 trusted-FAIL semantics pinned at three levels.

## 6. Preserve these brownfield seams

Do not replace:
- Closed Loop;
- agentruntime protocol;
- GitWriter/workspace/ChangeSet;
- EngineeringPolicy runtime schema;
- Outcome semantics;
- held-out evidence boundary.
