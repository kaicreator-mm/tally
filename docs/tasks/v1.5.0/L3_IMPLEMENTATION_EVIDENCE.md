# v1.5.0 L3 Implementation Evidence

This document is the implementation reference for Issues #6-#9. Local Task contracts and Decision Locks remain higher authority.

## T150-10 Requirements / Planning / Architecture
**Tests**: reject empty requirement/AC; freeze revisions with stable IDs/provenance; architecture constraints are explicit and validated.  
**Contract**: immutable accepted values; AI output remains proposal.  
**Implementation**: typed values in `internal/engineering`, constructors validate required fields, clone input slices to avoid post-freeze mutation.  
**Failure handling**: typed sentinel/validation errors, never silently normalize missing acceptance data.  
**Reference**: existing frozen-artifact/contract patterns in `internal/artifacts`, `internal/contract`, `internal/intentevidence`.

## T150-11 TaskDAG
**Tests**: duplicate node, missing dependency, cycle, blocked→ready, multiple parallel-ready, illegal state transition, restart round-trip.  
**Contract**: activation validates the entire proposal; Active TaskDAG is sole Project Flow.  
**Implementation**: adjacency + deterministic topological/cycle validation; readiness derives from predecessor terminal states and node state, never LLM. Preserve input order only for stable display; selection sorting is deterministic.  
**Failure handling**: invalid proposal cannot activate; unknown node/dependency is explicit error.  
**Reference**: Go explicit-state-machine style already used across taskstate/closedloop; do not introduce graph framework.

## T150-12 Completion / Authority / Audit
**Tests**: missing artifact/evidence/check/record/authority/Git requirement blocks; rejected authority blocks; duplicate IDs fail closed; action records append.  
**Contract**: Complete = required state + artifacts + evidence + checks + records + authority + Git requirements.  
**Implementation**: CompletionFacts plus pure `Evaluate`; AuthorityRequirement/Decision keyed by stable IDs; ActionRecord immutable value with timestamp/actor/provenance.  
**Failure handling**: return structured missing requirements; no boolean-only VERIFIED.

## T150-20 IntelligenceProvider
**Tests**: Direct/AI Runtime adapters satisfy same Provider suite; provider unavailable/error returns error without state mutation; provider result cannot satisfy deterministic DAG/completion checks by itself.  
**Contract**: typed Request → Result{Proposal, Assessment, Recommendation, Provenance}.  
**Implementation**: small `Invoker` dependency adapters; provider ID in provenance only.  
**Failure handling**: preserve provider error classification/cancellation; no fallback that changes accepted state silently.  
**Reference**: existing `internal/llm` and `internal/airuntime` are transports/boundaries, not project controllers.

## T150-21/22 AgentExecutor × Environment
**Tests**: Local workspace resolution, capability mismatch, fake runtime execution, cancellation, interrupted result, resume unsupported, resume failure and restart with retained/lost workspace.  
**Contract**: AgentExecutor owns execute/cancel/result boundary only. Runtime remains `agentruntime`; environment is orthogonal.  
**Implementation**: `WorkspaceRef{Environment,Target,Identity,Path,BaseRevision}`; LocalEnvironment alone resolves `Path`; RuntimeExecutor converts to existing StartRequest at adapter edge and closes Session deterministically.  
**Failure handling**: unsupported environment/resume is explicit; context cancellation maps to interruption/cancel semantics, not candidate FAIL; no unbounded goroutine/retry.  
**Reference**: `internal/agentruntime/protocol.go` and runtime conformance tests.

## T150-30 Control Plane / Persistence
**Tests**: project lifecycle phase gating; cannot activate DAG without accepted Requirements/Plan/Architecture; persisted snapshot reload restores Active TaskDAG/readiness/audit; append-only record ordering.  
**Contract**: Project Harness owns lifecycle and DAG advancement; repository is a consumer-owned DB-neutral port.  
**Implementation**: atomic file snapshot compatibility adapter for v1.5.0 local/restart proof; persistence is replaceable in v1.5.1.  
**Failure handling**: corrupt snapshot fails closed; write temp + fsync/rename where supported; never create a second in-memory truth after repository save.  
**Reference**: existing append-only/evidence and Git CAS patterns; no schema/migration redesign.

## T150-31 Brownfield integration
**Tests**: ready node dispatches through AgentExecutor; Agent claim alone cannot advance; verified/accepted outcome plus CompletionContract can advance; closedloop/runtime errors leave DAG blocked/interrupted with audit evidence.  
**Contract**: controlplane calls narrow execution/verification ports; existing closedloop remains candidate authority.  
**Implementation**: adapter at composition root, additive v1.5 JSON/read-model endpoints; no existing route rewrite.  
**Failure handling**: distinguish operational/runtime/environment errors from verification failure.

## T150-40 Visible validation
Run explicit Local/Build Host commands only: focused packages, architecture tests, runtime conformance, existing closedloop/workflow/workspace/git safety, integration/E2E where dependencies exist. Record revision, environment, command and outcome. Never infer PASS.

## T150-41 Hidden Validation
Only after visible implementation/tests freeze. Cover ambiguous/conflicting requirements, missing AC/design artifact, cycle/missing predecessor/parallel-ready/restart, completion missing facts/authority/illegal transition, provider substitution/unavailable, Local/cancel/interruption/worktree retained/lost/resume unsupported/failed, Codex/DSH regression and single-runtime invariant.

## T150-42 Closure
Check source/docs consistency, all Issue/PR accounting, Full Regression, Hidden Validation, Critical Journeys, packaging if applicable and 20 Release Qualification assertions. Any required NOT_RUN/ENV-BLOCKED prevents READY unless the frozen release policy explicitly classifies it non-blocking.
