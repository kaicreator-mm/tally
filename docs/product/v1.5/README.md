# Tally v1.5 PRD Set

Status: **FROZEN**  
Frozen on: 2026-09-15  
Implementation baseline: Tally v1.4.1 Strong Development Baseline.

This directory contains the frozen v1.5 architecture baseline and executable delivery slices.

## Authority

- `Tally_PRD_v1.5_FROZEN.md` is the architecture/product-direction authority for the v1.5 line.
- Each version PRD freezes the delivery scope for that capability slice.
- v1.5.0 is the shared Engineering Control Plane Foundation and freezes the contracts required by all later slices.
- Freezing these documents does not claim implementation or Release Qualification.

## Parallel development model

```text
                         v1.5.0
             Engineering Control Plane Foundation
                         │
        ┌────────────────┼────────────────┬────────────────┬────────────────┬────────────────┐
        ▼                ▼                ▼                ▼                ▼
     v1.5.1           v1.5.2           v1.5.3           v1.5.4           v1.5.5
 Persistence &      Execution        Git Engineering    Multi-Agent       Re-plan &
 Execution          Environments    Authority          Workflow          Learning
 Simplification
```

**Implementation rule:** after v1.5.0 Foundation contracts are frozen and integrated, v1.5.1 through v1.5.5 may run L2/L3, implementation and local validation in parallel. They must not introduce incompatible replacements for v1.5.0 contracts.

**Integration/release rule:** PRs remain one concern each and merge when locally safe. Version Closure and Release Qualification remain ordered so `main`, tags and release provenance are monotonic and reproducible.

## Version map

| Version | Scope | Implementation dependency | Release baseline dependency |
|---|---|---|---|
| v1.5.0 | Engineering Control Plane Foundation | v1.4.1 | v1.4.1 |
| v1.5.1 | Persistence & Execution Simplification | v1.5.0 Foundation contracts | v1.5.0 |
| v1.5.2 | Execution Environments | v1.5.0 execution contracts | v1.5.1 |
| v1.5.3 | Git Engineering Authority | v1.5.0 node/authority/contracts | v1.5.2 |
| v1.5.4 | Multi-Agent Engineering Workflow | v1.5.0 TaskDAG/execution/contracts | v1.5.3 |
| v1.5.5 | Re-plan & Engineering Learning | v1.5.0 planning/DAG/contracts | v1.5.4 |

The release-baseline column does **not** mean implementation must wait for the preceding minor version. It only defines the cumulative baseline used when qualifying/tagging that numbered release.

## Shared-contract rule

The following are owned/frozen by v1.5.0 and are not redefined by parallel slices: Requirements/Planning/Architecture domain contracts, `TaskDAG`, `EngineeringNode`, `CompletionContract`, Artifact/Evidence/Audit references, Authority modes, `IntelligenceProvider`, `AgentExecutor`, Agent Runtime boundary, `ExecutionEnvironment` boundary, `ExecutionTarget`, `WorkspaceRef`, capability requirements, execution request/result and interruption/restart semantics.

Implementation starts per the project development standard: L2 Architecture Evidence → Task DAG → L3 Implementation Evidence → GitHub Issues/PR → CI → Version Closure → Hidden Validation / Critical Journeys / Packaging → Release Qualification.