# v1.5.4 L3 Implementation Evidence

## T154-01 — role contracts

**Tests:** role workflow unit tests and persisted snapshot validation.  
**Contract:** roles are engineering responsibilities; TaskDAG remains project-flow authority.  
**Implementation:** `internal/multiagent/types.go`; optional `MultiAgentPolicies`/`MultiAgentWorkflows` in Control Plane snapshot.  
**Failure handling:** invalid role/policy/state is rejected before execution or persistence.  
**Reference:** frozen v1.5.0 TaskDAG/CompletionContract and v1.5.4 PRD.

## T154-02 — independence and provenance

**Tests:** independent run violation, missing reviewer evidence, runtime-reference spoof rejection.  
**Contract:** reviewer/verifier cannot reuse candidate run/session; stricter runtime independence must represent a different actual runtime. Agent claim is never review fact.  
**Implementation:** pre/post role independence checks; `Finding.ProducerRole/ProducerRunID`; `RuntimeExecutor` verifies `RuntimeRef.ID == RuntimeDescriptor.Name`.  
**Failure handling:** impossible independence blocks fail-safe before acceptance; spoofed runtime reference is rejected before Agent session start.

## T154-03 — sequential collaboration

**Tests:** happy path, review finding → repair, repair provider failure, verifier disagreement, restart mid-review, bounded repair.  
**Contract:** Implement → Specialist(s) → Review → Repair when necessary → Review → Verify.  
**Implementation:** `internal/multiagent/engine.go` with durable checkpoints and fresh-role restart.  
**Failure handling:** infra/provider errors do not consume repair-quality budget; interrupted in-flight role is persisted and restarted without session/thought recovery.

## T154-04 — bounded scheduler

**Tests:** two ready non-conflicting tasks, dependency race, shared conflict key, workspace collision, concurrency limit, queued cancellation.  
**Contract:** scheduler may select only TaskDAG Ready + ParallelEligible tasks and cannot supersede TaskDAG state.  
**Implementation:** `internal/multiagent/scheduler.go` + engine semaphore.  
**Failure handling:** conflicting/dependency-blocked work stays Ready/Blocked and is not dispatched.

## T154-05 — Control Plane integration

**Tests:** existing CompletionContract acceptance, reviewer claim without evidence, parallel batch, configured conflict, legacy-running ownership, restart race.  
**Contract:** collaboration state is subordinate; CompletionContract owns final acceptance.  
**Implementation:** `internal/controlplane/multiagent.go`, optional HTTP routes and persisted snapshot fields.  
**Failure handling:** Running task recovery requires durable multi-agent ownership; RestartTask cancellation cannot be overwritten by stale late finalize.

## T154-06 — production composition

**Tests:** runtime configured/unconfigured production composition and concurrency-config bounds.  
**Contract:** reuse existing fake/codex/dsh Runtime + `taskexecution.RuntimeExecutor(LocalEnvironment{})`; no second runtime stack.  
**Implementation:** `cmd/server/v15.go`; Multi-Agent Engine and Brownfield Harness share the same executor.  
**Failure handling:** absent runtime keeps execution explicitly unavailable; invalid max-concurrency config rejects startup.

## T154-07/08 — validation and closure

Executable local/Build-Host entry points:

```bash
bash scripts/validate-v1.5.4.sh
bash scripts/hidden-validation-v1.5.4.sh
bash scripts/critical-journeys-v1.5.4.sh
bash scripts/full-regression-v1.5.4.sh
```

These scripts intentionally contain no GitHub Actions/CI dependency. Executed results must be copied into `VALIDATION_AND_CLOSURE.md` with revision, environment, command, timestamp and status. An Agent statement that tests would pass is not evidence.
