# Tally PRD v1.5.5 — Re-plan & Engineering Learning

- Status: **FROZEN EXECUTABLE SCOPE**
- Depends on: v1.5.4

## Goal

Close the project-level learning loop: detect when the accepted plan is no longer valid, create a new planning proposal/revision safely, and promote evidence-backed engineering learning without giving AI unilateral authority.

## In scope

1. Re-plan triggers from requirement change, architecture invalidation, repeated task failure, blocked dependency, environment impossibility, verification evidence or explicit human request.
2. Current accepted facts/evidence are passed to Direct LLM or AI Runtime to generate `ReplanProposal`.
3. Engineering Harness performs impact analysis and validates the proposal.
4. Re-plan creates a new immutable PlanningRevision/TaskDAG baseline; history is never overwritten.
5. Preserve accepted work when compatible; explicitly supersede/invalidate affected Tasks when necessary.
6. Re-evaluate dependencies, readiness, parallelism, verification, Git mappings and environment constraints.
7. Policy determines AUTO, SUPERVISED or AUTHORITY_REQUIRED activation.
8. Outcome records feed engineering learning.
9. Candidate Engineering Rules cite supporting evidence/outcomes.
10. High-impact/shared rule promotion requires configured authority; AI proposes but does not self-authorize.

## Out of scope

Autonomous product-scope changes without authority, unrestricted self-modifying policy engine, hidden chain-of-thought learning and knowledge-platform replacement.

## Acceptance criteria

- Material failure can generate ReplanProposal through either Direct LLM or AI Runtime.
- Proposal cannot activate without Engineering Harness validation.
- New PlanningRevision never overwrites history.
- Unaffected accepted Tasks remain reusable and traceable.
- Invalidated/superseded Tasks are explicit.
- Policy can require human authority for material scope/architecture change.
- Engineering Rule proposals include evidence provenance and cannot bypass promotion authority.

## Hidden Validation

Requirement change, architecture invalidation, repeated repair failure, provider disagreement, re-plan cycle, accepted-task preservation, invalidated-task replacement, authority-required scope change and unsupported rule promotion.

## Release qualification

Re-plan critical journey PASS; revision/provenance integrity PASS; Direct LLM/AI Runtime substitution PASS; learning/rule promotion authority PASS.
