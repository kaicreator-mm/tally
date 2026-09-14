# Tally v1.4.1 Task Index — L3 Enriched

Total tasks: **57**  
LOW execution: **40**  
HIGH execution: **17**

| ID | Phase | Wave | Task | Execute Model | Review Model | L3 | Depends |
|---|---|---|---|---|---|---|---|
| A0-01 | A0 | W00 | Current source-to-PRD authority map | **LOW** | HIGH | COMPLETE | - |
| A0-02 | A0 | W00 | Freeze v1.4.1 architecture decision locks | **HIGH** | HIGH | COMPLETE | A0-01 |
| A0-03 | A0 | W01 | Validation and dependency preflight | **LOW** | LOW | COMPLETE | - |
| A0-04 | A0 | W01 | Architecture guard tests | **LOW** | HIGH | COMPLETE | A0-02 |
| A0-05 | A0 | W02 | Architecture readiness gate | **LOW** | HIGH | COMPLETE | A0-02, A0-03, A0-04 |
| P0-01 | P0 | W03 | Define immutable FrozenIntent evidence | **LOW** | HIGH | COMPLETE | A0-05 |
| P0-02 | P0 | W04 | FrozenIntent Git artifact loader | **LOW** | LOW | COMPLETE | P0-01 |
| P0-03 | P0 | W03 | Bounded dynamic clarification schema | **LOW** | LOW | COMPLETE | A0-05 |
| P0-04 | P0 | W05 | Render dynamic forms in v1.4 Launch UI | **LOW** | LOW | COMPLETE | P0-03 |
| P0-05 | P0 | W04 | Wire production Intent AI executor | **HIGH** | HIGH | COMPLETE | A0-05 |
| P0-06 | P0 | W05 | Draft-to-frozen domain mapper | **LOW** | HIGH | COMPLETE | P0-01, P0-03 |
| P0-07 | P0 | W05 | Verification authoring and freeze admission | **HIGH** | HIGH | COMPLETE | P0-03, P0-06 |
| P0-08 | P0 | W06 | Atomic Confirm-to-Freeze Git transaction | **HIGH** | HIGH | COMPLETE | P0-02, P0-04, P0-05, P0-06, P0-07 |
| P0-09 | P0 | W07 | Intent-to-Freeze E2E gate | **LOW** | HIGH | COMPLETE | P0-08 |
| P1-01 | P1 | W08 | Same-source execution bundle loader | **LOW** | HIGH | COMPLETE | P0-09 |
| P1-02 | P1 | W09 | Authoritative risk input derivation | **HIGH** | HIGH | COMPLETE | P1-01 |
| P1-03 | P1 | W10 | Production contract compilation assembler | **HIGH** | HIGH | COMPLETE | P1-02 |
| P1-04 | P1 | W11 | Wire worker to compiled authority | **LOW** | HIGH | COMPLETE | P1-03 |
| P1-05 | P1 | W12 | Compiled prevention context projection | **LOW** | HIGH | COMPLETE | P1-04 |
| P1-06 | P1 | W12 | Risk-adjusted acceptance guard | **HIGH** | HIGH | COMPLETE | P1-04 |
| P1-07 | P1 | W13 | Compiled basis in task/read projections | **LOW** | HIGH | COMPLETE | P1-04, P1-06 |
| P1-08 | P1 | W14 | Compiled-authority no-bypass gate | **LOW** | HIGH | COMPLETE | P1-05, P1-06, P1-07 |
| P2-01 | P2 | W15 | Human-centered navigation projection | **LOW** | LOW | COMPLETE | P1-08 |
| P2-02 | P2 | W16 | Work list and task detail projection | **LOW** | LOW | COMPLETE | P2-01 |
| P2-03 | P2 | W16 | Runs projection | **LOW** | LOW | COMPLETE | P2-01 |
| P2-04 | P2 | W17 | Reviews projection with human-needed reason | **LOW** | LOW | COMPLETE | P2-01, P2-02 |
| P2-05 | P2 | W18 | Human-centered UX gate | **LOW** | HIGH | COMPLETE | P2-02, P2-03, P2-04 |
| P3-01 | P3 | W19 | Policy governance proposal domain | **LOW** | HIGH | COMPLETE | P2-05 |
| P3-02 | P3 | W20 | Append-only policy governance store | **LOW** | LOW | COMPLETE | P3-01 |
| P3-03 | P3 | W20 | Policy candidate canonicalization and validation | **LOW** | HIGH | COMPLETE | P3-01 |
| P3-04 | P3 | W21 | Evidence-bounded historical replay | **HIGH** | HIGH | COMPLETE | P3-02, P3-03 |
| P3-05 | P3 | W21 | Policy conflict analysis | **HIGH** | HIGH | COMPLETE | P3-03 |
| P3-06 | P3 | W22 | Rules UI projections | **LOW** | LOW | COMPLETE | P3-02, P3-04, P3-05 |
| P3-07 | P3 | W23 | Approve and activate policy via GitWriter | **LOW** | HIGH | COMPLETE | P3-03, P3-06 |
| P3-08 | P3 | W23 | Apply Once vs Propose Rule review action | **LOW** | LOW | COMPLETE | P2-04, P3-01 |
| P3-09 | P3 | W24 | Rule-governance gate | **LOW** | HIGH | COMPLETE | P3-04, P3-05, P3-07, P3-08 |
| P4-01 | P4 | W25 | Runtime composition registry/factory | **LOW** | HIGH | COMPLETE | P1-08 |
| P4-02 | P4 | W26 | Codex runtime regression through factory | **LOW** | HIGH | COMPLETE | P4-01 |
| P4-03 | P4 | W25 | DeepSeek Harness integration spike | **HIGH** | HIGH | COMPLETE | A0-05 |
| P4-04 | P4 | W27 | DeepSeek Harness agentruntime adapter skeleton | **LOW** | HIGH | COMPLETE | P4-01, P4-03 |
| P4-05 | P4 | W28 | DeepSeek session lifecycle and conformance | **HIGH** | HIGH | COMPLETE | P4-04 |
| P4-06 | P4 | W28 | DeepSeek configuration and canary policy | **LOW** | HIGH | COMPLETE | P4-04 |
| P4-07 | P4 | W29 | DeepSeek limited-readiness gate | **LOW** | HIGH | COMPLETE | P4-02, P4-05, P4-06 |
| P5-01 | P5 | W30 | Semantic local Git integration service | **HIGH** | HIGH | COMPLETE | P1-08 |
| P5-02 | P5 | W31 | Controlled integration branch creation | **LOW** | LOW | COMPLETE | P5-01 |
| P5-03 | P5 | W32 | Safe local merge integration | **HIGH** | HIGH | COMPLETE | P5-02 |
| P5-04 | P5 | W32 | Protected-branch and Git permission policy L0-L3 | **LOW** | HIGH | COMPLETE | P5-01 |
| P5-05 | P5 | W33 | Integration audit/read projection | **LOW** | LOW | COMPLETE | P5-03, P5-04 |
| P5-06 | P5 | W34 | Local Git control-plane gate | **LOW** | HIGH | COMPLETE | P5-03, P5-04, P5-05 |
| P6-01 | P6 | W35 | Accepted Outcome production writer | **HIGH** | HIGH | COMPLETE | P1-08 |
| P6-02 | P6 | W36 | Remaining existing Outcome event capture | **LOW** | HIGH | COMPLETE | P6-01 |
| P6-03 | P6 | W37 | Outcome evidence projection for Rules | **LOW** | LOW | COMPLETE | P3-01, P6-02 |
| P6-04 | P6 | W35 | Migration false-negative reproduction | **HIGH** | HIGH | COMPLETE | A0-05 |
| P6-05 | P6 | W36 | Fix migration verification root cause | **HIGH** | HIGH | COMPLETE | P6-04 |
| P6-06 | P6 | W35 | Safe project export | **LOW** | LOW | COMPLETE | A0-05 |
| P6-07 | P6 | W38 | Current documentation synchronization | **LOW** | LOW | COMPLETE | P3-09, P4-07, P5-06, P6-02, P6-05, P6-06 |
| P6-08 | P6 | W39 | Final hidden validation and release gate | **HIGH** | HIGH | COMPLETE | P3-09, P4-07, P5-06, P6-03, P6-05, P6-07 |
