# Tally v1.4 Task DAG

Architecture is `READY_AFTER_FIXES`; A0 resolves/fixes freeze prerequisites, then P0 closes v1.3 trust gaps before v1.4 foundation and loop work.

```text
A0 Evidence/Spikes -> A0 Freeze
                     |
                     v
              P0 Trust Closeout
                     |
                     v
        P1 Contract/Policy/Assurance
                     |
                     v
             P2 Closed Repair Loop
                     |
                     v
          P3 MVP Harness + Surface
                     |
                     v
             P4 Product Validation
```

## W00
- `A0-01` Source baseline and v1.4 Architecture Claim Register — deps: none
- `A0-02` Codex Session Isolation Spike — deps: none
- `A0-03` Assurance L1/L2/L3 Bake-off — deps: none
- `A0-04` Closed-loop Repair and held-out feedback ADR — deps: none
- `A0-05` Product value experiment preregistration — deps: none

## W01
- `A0-06` Architecture Freeze Gate — deps: A0-01, A0-02, A0-03, A0-04, A0-05

## W02
- `P0-01` Propagate requested ModelProfile and capture generic runtime identity — deps: A0-06
- `P0-05` Measurement SQL contract for unknown usage calls — deps: A0-06
- `P0-07` Ancestor exposure lineage projection and review visibility — deps: A0-06
- `P0-08` Operationalize baseline report on real history — deps: A0-06
- `P0-09` Common AI Runtime domain adapter boundary — deps: A0-06
- `P0-12` Repository EOL hygiene and v1.4 source baseline docs — deps: A0-06

## W03
- `P0-02` Persist actual Codex runtime/model/session snapshot — deps: P0-01
- `P0-06` Measurement recorder/store correctness and sqlc regeneration — deps: P0-05
- `P0-10` Migrate requirement and harvest generic LLM consumers to AI Runtime — deps: P0-09

## W04
- `P0-03` Implement Codex per-run session/home isolation — deps: A0-02, P0-02
- `P0-11` Migrate failure/diagnose/worker generic LLM consumers and retire direct provider path — deps: P0-10

## W05
- `P0-04` Real Codex standard + held-out trust E2E — deps: P0-03

## W06
- `P0-13` v1.3.1 Trust Closeout Gate — deps: P0-04, P0-06, P0-07, P0-08, P0-11, P0-12

## W07
- `P1-01` ContractSnapshot aggregate over existing Task + Verification contracts — deps: P0-13
- `P1-02` EngineeringPolicy schema, parser and deterministic gate validation — deps: P0-13
- `P1-04` Derived RiskSignals model — deps: P0-13
- `P1-06` Assurance model and selector from bake-off decision — deps: P0-13, A0-03

## W08
- `P1-03` Git artifact loading for EngineeringPolicy — deps: P1-02

## W09
- `P1-05` Policy matching and PolicyMatchCoverage — deps: P1-02, P1-04

## W10
- `P1-07` Contract compiler: policy bindings, required evidence and pre-execution requires context — deps: P1-01, P1-03, P1-05, P1-06

## W11
- `P1-08` Extend taskstate with v1.4 VerifiedBasis projection — deps: P1-06, P1-07
- `P1-09` v1.4 foundation architecture invariants — deps: P1-05, P1-07, P1-08

## W12
- `P1-10` Contract/Policy/Assurance Foundation Gate — deps: P1-08, P1-09

## W13
- `P2-01` Extract reusable candidate verification service from worker v1.3 composition — deps: P1-10
- `P2-02` Closed-loop orchestrator domain skeleton — deps: P1-10
- `P2-03` Verification failure classification — deps: P1-10

## W14
- `P2-04` Structured safe RepairFeedback builder — deps: P2-03
- `P2-05` Repair continuation from prior candidate with immutable supersession lineage — deps: P2-02, P2-03
- `P2-07` Separate bounded outer RepairPolicy from ExecutionPolicy retry — deps: P2-02, P2-03

## W15
- `P2-06` Held-out Repair safety and exposure accounting — deps: A0-04, P2-04, P2-05, P0-07

## W16
- `P2-08` Integrate Execute -> Verify -> Repair closed-loop engine — deps: P2-01, P2-04, P2-05, P2-06, P2-07

## W17
- `P2-09` Worker v1.4 closed-loop job integration — deps: P2-08

## W18
- `P2-10` FakeRuntime closed-loop E2E matrix — deps: P2-09

## W19
- `P2-11` Closed Loop Gate — deps: P2-10

## W20
- `P3-01` Built-in Bug Fix EngineeringPolicy bundle and fixtures — deps: P2-11
- `P3-02` Built-in Database Migration EngineeringPolicy bundle and fixtures — deps: P2-11
- `P3-05` Append-only OutcomeEvent SQL/store — deps: P2-11
- `P3-07` Minimal intent intake and Contract proposal via AI Runtime — deps: P1-07, P0-11

## W21
- `P3-03` Bug Fix closed-loop E2E — deps: P3-01, P2-11
- `P3-04` Database Migration closed-loop E2E — deps: P3-02, P2-11
- `P3-06` Closed-loop product metric projections — deps: P3-05, P2-11

## W22
- `P3-08` Minimal Launch and Closed Loop task detail UI — deps: P3-07, P2-09
- `P3-09` Change Detail + minimal Attention projection — deps: P1-08, P2-09, P0-07

## W23
- `P3-10` MVP Product Gate — deps: P3-03, P3-04, P3-06, P3-08, P3-09

## W24
- `P4-01` Freeze dogfood evaluation task set and fixtures — deps: P3-10, A0-05

## W25
- `P4-02` Bare-Agent control runner with parity controls — deps: P4-01

## W26
- `P4-03` Real Codex Tally closed-loop comparative trial — deps: P4-02, P0-04

## W27
- `P4-04` v1.4 Comparative Product Value Report and kill/pivot decision — deps: P4-03

## W28
- `P4-05` v1.4 Final Closeout and next-scope promotion gate — deps: P4-04
