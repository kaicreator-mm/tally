# Tally v1.5.4 Engineering Task Pack

Version: **v1.5.4 — Multi-Agent Engineering Workflow**  
Frozen product scope: `docs/product/v1.5/Tally_PRD_v1.5.4_Multi_Agent_Engineering_Workflow.md`  
Implementation dependency: v1.5.0 frozen TaskDAG / AgentExecutor / EngineeringNode contracts.  
Release baseline dependency: **v1.5.3**.

This task pack follows the repository development-standard override for this version: **no GitHub Actions and no CI**. Validation is executed only on an authorized Local/Build Host and evidence must use `PASS | FAIL | NOT_RUN(reason) | ENV-BLOCKED(reason + evidence)`.

## Documents

- `L2_ARCHITECTURE_EVIDENCE.md` — compatibility and architecture evidence.
- `DECISION_LOCKS.md` — frozen implementation decisions.
- `TASK_DAG.md` — implementation DAG and Issue mapping.
- `L3_IMPLEMENTATION_EVIDENCE.md` — implementation references, contracts and failure semantics.
- `VALIDATION_AND_CLOSURE.md` — validation matrix, Critical Journeys, blockers and Release Qualification.

## GitHub work items

- #39 — role workflow, independence and evidence provenance.
- #40 — bounded parallel scheduler and Control Plane integration.
- #42 — validation, Critical Journeys and Release Closure.

Implementation branch: `dev/v1.5.4-multi-agent-workflow-r3`.

The v1.5.4 core is intentionally layered above the frozen v1.5.0 contracts. Active TaskDAG remains the only Project Flow authority, Agent output remains a claim until Tally evidence/gates accept it, and no generic workflow engine, persistent thought graph, distributed Job Queue or remote Agent fleet is introduced.
