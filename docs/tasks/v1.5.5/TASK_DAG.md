# Tally v1.5.5 Task DAG

Base: `d0c0f72390fb04d5891cfc82937f977729488f25`

```text
T155-01 Replan domain + triggers
   ├── T155-02 Proposal fingerprint / cycle control
   ├── T155-03 Impact + accepted-work preservation validator
   └── T155-04 Intelligence proposal adapter

T155-01 + T155-03
   └── T155-05 Control Plane replan activation + immutable history

T155-06 Engineering learning proposal domain
   ├── T155-07 Outcome/evidence + conflict validation
   └── T155-08 Authority-governed policygovernance adapter

T155-04 + T155-05 + T155-08
   └── T155-09 Integration/API surface

T155-01..09
   ├── T155-10 Focused + regression validation carrier
   ├── T155-11 Hidden Validation carrier
   └── T155-12 Critical Journey / Closure / Release Qualification
```

## Tasks

| ID | Concern | Dependencies | Parallel | Target model | Completion |
|---|---|---|---:|---|---|
| T155-01 | Trigger facts, dispositions, ReplanProposal typed domain | v1.5.0 contracts | yes | low | typed validation tests |
| T155-02 | Canonical fingerprint + bounded repeat/cycle policy | T155-01 | yes | low | deterministic repeat tests |
| T155-03 | Impact/dependency/accepted-work preservation validation | T155-01 | yes | high | preserve/invalidate/dependency tests |
| T155-04 | Direct/AI Runtime proposal parsing through existing Provider | T155-01 | yes | medium | substitution/disagreement tests |
| T155-05 | Atomic Control Plane replan activation + history | T155-03 | no | high | stale-parent/authority/history tests |
| T155-06 | EngineeringRuleProposal domain | v1.5.0 evidence + existing policy governance | yes | low | evidence/provenance tests |
| T155-07 | Outcome/evidence/conflict validation | T155-06 | yes | medium | missing/conflicting evidence tests |
| T155-08 | Promotion authority + existing policygovernance adapter | T155-06,07 | yes | high | self-authorize/authority tests |
| T155-09 | Bounded Control Plane/API integration | T155-04,05,08 | no | high | integration route/service tests |
| T155-10 | visible validation + full regression carrier | T155-01..09 | no | medium | evidence matrix |
| T155-11 | held-out Hidden Validation carrier | T155-01..09 | no | high | external hidden protocol |
| T155-12 | Critical Journeys + Closure + Release Qualification | T155-10,11 | no | high | explicit qualification verdict |

## Parallel lanes

Lane A — Replan domain: T155-01 → T155-02/T155-03/T155-04.

Lane B — Learning: T155-06 → T155-07/T155-08.

Lane C — Integration: T155-05 + T155-09 after the relevant domain contracts stabilize.

Lane D — Qualification: T155-10/T155-11 → T155-12.

No task may require v1.5.1 persistence internals, v1.5.2 SSH/Docker, v1.5.3 GitHub automation, or v1.5.4 multi-Agent implementation.