# DomainHarness v0.1 — Tally external-domain validation

This directory is validation-only and does not replace Tally's Active TaskDAG, CompletionContract, verification, or authority model.

## Primitive coverage

- Skill: `assess_task`
- Tool: host-injected `tally_verify_completion_contract`
- Expr: normalization and child authority projection
- Child Workflow: `authority_check`
- Waiting event: `approve` / `reject` with JSON Schema validation

The Tool is deliberately host-owned: its implementation must call or adapt Tally-owned deterministic verification/authority code. Runtime receives only JSON input/output and does not own Tally semantics.

## Required exact-SHA execution

Run this Harness against the DomainHarness v0.1 candidate package on a Build Host. Record both repository SHAs, package/tarball SHA, lifecycle trace, persisted run/step evidence, and proof that no DomainHarness public contract change was required.
