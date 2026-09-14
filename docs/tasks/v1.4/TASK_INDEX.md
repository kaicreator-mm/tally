# Tally v1.4 Task Index

**Task count:** 55

| ID | Title | Phase | Wave | Complexity | Dependencies | L3 |
|---|---|---|---|---|---|---|
| [A0-01](A0/A0-01.md) | Source baseline and v1.4 Architecture Claim Register | A0 | W00 | S | — | no |
| [A0-02](A0/A0-02.md) | Codex Session Isolation Spike | A0 | W00 | M | — | yes |
| [A0-03](A0/A0-03.md) | Assurance L1/L2/L3 Bake-off | A0 | W00 | L | — | yes |
| [A0-04](A0/A0-04.md) | Closed-loop Repair and held-out feedback ADR | A0 | W00 | M | — | yes |
| [A0-05](A0/A0-05.md) | Product value experiment preregistration | A0 | W00 | M | — | no |
| [A0-06](A0/A0-06.md) | Architecture Freeze Gate | A0 | W01 | S | A0-01, A0-02, A0-03, A0-04, A0-05 | no |
| [P0-01](P0/P0-01.md) | Propagate requested ModelProfile and capture generic runtime identity | P0 | W02 | M | A0-06 | yes |
| [P0-02](P0/P0-02.md) | Persist actual Codex runtime/model/session snapshot | P0 | W03 | M | P0-01 | yes |
| [P0-03](P0/P0-03.md) | Implement Codex per-run session/home isolation | P0 | W04 | L | A0-02, P0-02 | yes |
| [P0-04](P0/P0-04.md) | Real Codex standard + held-out trust E2E | P0 | W05 | M | P0-03 | no |
| [P0-05](P0/P0-05.md) | Measurement SQL contract for unknown usage calls | P0 | W02 | M | A0-06 | yes |
| [P0-06](P0/P0-06.md) | Measurement recorder/store correctness and sqlc regeneration | P0 | W03 | M | P0-05 | yes |
| [P0-07](P0/P0-07.md) | Ancestor exposure lineage projection and review visibility | P0 | W02 | M | A0-06 | optional |
| [P0-08](P0/P0-08.md) | Operationalize baseline report on real history | P0 | W02 | S | A0-06 | no |
| [P0-09](P0/P0-09.md) | Common AI Runtime domain adapter boundary | P0 | W02 | M | A0-06 | yes |
| [P0-10](P0/P0-10.md) | Migrate requirement and harvest generic LLM consumers to AI Runtime | P0 | W03 | M | P0-09 | yes |
| [P0-11](P0/P0-11.md) | Migrate failure/diagnose/worker generic LLM consumers and retire direct provider path | P0 | W04 | L | P0-10 | yes |
| [P0-12](P0/P0-12.md) | Repository EOL hygiene and v1.4 source baseline docs | P0 | W02 | S | A0-06 | no |
| [P0-13](P0/P0-13.md) | v1.3.1 Trust Closeout Gate | P0 | W06 | M | P0-04, P0-06, P0-07, P0-08, P0-11, P0-12 | no |
| [P1-01](P1/P1-01.md) | ContractSnapshot aggregate over existing Task + Verification contracts | P1 | W07 | M | P0-13 | optional |
| [P1-02](P1/P1-02.md) | EngineeringPolicy schema, parser and deterministic gate validation | P1 | W07 | M | P0-13 | yes |
| [P1-03](P1/P1-03.md) | Git artifact loading for EngineeringPolicy | P1 | W08 | M | P1-02 | optional |
| [P1-04](P1/P1-04.md) | Derived RiskSignals model | P1 | W07 | S | P0-13 | yes |
| [P1-05](P1/P1-05.md) | Policy matching and PolicyMatchCoverage | P1 | W09 | M | P1-02, P1-04 | yes |
| [P1-06](P1/P1-06.md) | Assurance model and selector from bake-off decision | P1 | W07 | M | P0-13, A0-03 | yes |
| [P1-07](P1/P1-07.md) | Contract compiler: policy bindings, required evidence and pre-execution requires context | P1 | W10 | L | P1-01, P1-03, P1-05, P1-06 | yes |
| [P1-08](P1/P1-08.md) | Extend taskstate with v1.4 VerifiedBasis projection | P1 | W11 | M | P1-06, P1-07 | optional |
| [P1-09](P1/P1-09.md) | v1.4 foundation architecture invariants | P1 | W11 | M | P1-05, P1-07, P1-08 | no |
| [P1-10](P1/P1-10.md) | Contract/Policy/Assurance Foundation Gate | P1 | W12 | S | P1-08, P1-09 | no |
| [P2-01](P2/P2-01.md) | Extract reusable candidate verification service from worker v1.3 composition | P2 | W13 | L | P1-10 | optional |
| [P2-02](P2/P2-02.md) | Closed-loop orchestrator domain skeleton | P2 | W13 | M | P1-10 | yes |
| [P2-03](P2/P2-03.md) | Verification failure classification | P2 | W13 | M | P1-10 | yes |
| [P2-04](P2/P2-04.md) | Structured safe RepairFeedback builder | P2 | W14 | M | P2-03 | yes |
| [P2-05](P2/P2-05.md) | Repair continuation from prior candidate with immutable supersession lineage | P2 | W14 | L | P2-02, P2-03 | yes |
| [P2-06](P2/P2-06.md) | Held-out Repair safety and exposure accounting | P2 | W15 | L | A0-04, P2-04, P2-05, P0-07 | yes |
| [P2-07](P2/P2-07.md) | Separate bounded outer RepairPolicy from ExecutionPolicy retry | P2 | W14 | M | P2-02, P2-03 | optional |
| [P2-08](P2/P2-08.md) | Integrate Execute -> Verify -> Repair closed-loop engine | P2 | W16 | L | P2-01, P2-04, P2-05, P2-06, P2-07 | yes |
| [P2-09](P2/P2-09.md) | Worker v1.4 closed-loop job integration | P2 | W17 | L | P2-08 | optional |
| [P2-10](P2/P2-10.md) | FakeRuntime closed-loop E2E matrix | P2 | W18 | L | P2-09 | no |
| [P2-11](P2/P2-11.md) | Closed Loop Gate | P2 | W19 | M | P2-10 | no |
| [P3-01](P3/P3-01.md) | Built-in Bug Fix EngineeringPolicy bundle and fixtures | P3 | W20 | M | P2-11 | yes |
| [P3-02](P3/P3-02.md) | Built-in Database Migration EngineeringPolicy bundle and fixtures | P3 | W20 | M | P2-11 | yes |
| [P3-03](P3/P3-03.md) | Bug Fix closed-loop E2E | P3 | W21 | M | P3-01, P2-11 | yes |
| [P3-04](P3/P3-04.md) | Database Migration closed-loop E2E | P3 | W21 | L | P3-02, P2-11 | yes |
| [P3-05](P3/P3-05.md) | Append-only OutcomeEvent SQL/store | P3 | W20 | M | P2-11 | optional |
| [P3-06](P3/P3-06.md) | Closed-loop product metric projections | P3 | W21 | M | P3-05, P2-11 | optional |
| [P3-07](P3/P3-07.md) | Minimal intent intake and Contract proposal via AI Runtime | P3 | W20 | L | P1-07, P0-11 | yes |
| [P3-08](P3/P3-08.md) | Minimal Launch and Closed Loop task detail UI | P3 | W22 | L | P3-07, P2-09 | optional |
| [P3-09](P3/P3-09.md) | Change Detail + minimal Attention projection | P3 | W22 | L | P1-08, P2-09, P0-07 | optional |
| [P3-10](P3/P3-10.md) | MVP Product Gate | P3 | W23 | M | P3-03, P3-04, P3-06, P3-08, P3-09 | no |
| [P4-01](P4/P4-01.md) | Freeze dogfood evaluation task set and fixtures | P4 | W24 | M | P3-10, A0-05 | yes |
| [P4-02](P4/P4-02.md) | Bare-Agent control runner with parity controls | P4 | W25 | M | P4-01 | optional |
| [P4-03](P4/P4-03.md) | Real Codex Tally closed-loop comparative trial | P4 | W26 | L | P4-02, P0-04 | no |
| [P4-04](P4/P4-04.md) | v1.4 Comparative Product Value Report and kill/pivot decision | P4 | W27 | M | P4-03 | no |
| [P4-05](P4/P4-05.md) | v1.4 Final Closeout and next-scope promotion gate | P4 | W28 | M | P4-04 | no |
