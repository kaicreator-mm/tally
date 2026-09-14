# Deferred Roadmap — do not implement from this pack

The latest PRD defines P2–P6, but this pack intentionally does **not** atomize them into executable low-cost coding tasks yet. Their correct design depends on the P1 Trust Gate and real usage evidence.

## P2 — Engineering Automation & Product Value
Activate only after P1 review. Future epics: Triage/risk-aware intake, Light Fast Path, full profiles/governance monotonicity, Product/Planner/VerificationPlanner automation, Task DAG/ready selector, generalized workflow kernel, holdout recovery automation, docs co-evolution integration gate, Saved Views, lifecycle/value-stream comparison against bare agent.

## P3 — Runtime Portability
Pi/OpenCode adapters, formal capability registry hardening, at least two production runtimes.

## P4 — Operate & Failure
Release/Deployment separation, FailureEvent→Defect→Incident activation, regression/suspect change/known fix, DeliveryState and rollout/stability. Existing legacy failure code is retained as an asset until this stage.

## P5 — Adaptive Observe
BehaviorManifest observation, comparable run cohorts, golden cases, replay evaluation.

## P6 — Adaptive Evolve (conditional)
Only after readiness: candidate/replay/promotion/rollback. AdaptiveKit never owns hard-policy authority.

**Do not prebuild P2+ tables, UI or abstractions “for future compatibility” during P0/P1.**
