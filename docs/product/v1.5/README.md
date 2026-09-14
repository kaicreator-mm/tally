# Tally v1.5 PRD Set

Status: **FROZEN**  
Frozen on: 2026-09-15  
Implementation baseline: Tally v1.4.1 Strong Development Baseline.

This directory contains the frozen v1.5 architecture baseline and the executable delivery slices derived from it.

## Authority

- `Tally_PRD_v1.5_FROZEN.md` is the architecture and product-direction authority for the v1.5 line.
- Each version PRD freezes the delivery scope for that version.
- The v1.5 documents do not retroactively change the released v1.4.1 baseline.
- Implementation starts per version through the project development standard: L2 Architecture Evidence → Task DAG → L3 Implementation Evidence → GitHub Issues/PR → CI → Version Closure → Hidden Validation / Critical Journeys / Packaging → Release Qualification.

## Version map

| Version | Scope | Dependency |
|---|---|---|
| v1.5.0 | Engineering Control Plane Core | v1.4.1 |
| v1.5.1 | Persistence & Execution Simplification | v1.5.0 |
| v1.5.2 | Execution Environments | v1.5.1 |
| v1.5.3 | Git Engineering Authority | v1.5.2 |
| v1.5.4 | Multi-Agent Engineering Workflow | v1.5.3 |
| v1.5.5 | Re-plan & Engineering Learning | v1.5.4 |

The versions are intentionally small, cumulative, and independently closable. Later versions may begin design work earlier, but Release Qualification is cumulative in the order above.
