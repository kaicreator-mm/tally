# Architecture Decision Records

This directory holds implementation decisions that are too detailed for the PRD.

## ADR authority

An ADR may refine *how* an approved product/architecture invariant is implemented. It may not silently redefine product authority, weaken a hard invariant, or move P2+ scope into P0/P1.

## Initial ADR backlog from v1.3

Expected topics include:

- Verification Independence and exposure calculation;
- Holdout Evidence Store and injection/isolation;
- Runtime isolation/conformance;
- ChangeSet/candidate identity;
- Verification environment identity;
- canonical failure fingerprint semantics;
- structured command persistence/execution;
- Git writer serialization.

Create ADRs when an implementation task reaches the decision, not pre-emptively as speculative architecture.
