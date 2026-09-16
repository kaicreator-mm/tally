# Tally v1.5.5 Validation & Closure Ledger

Version: **v1.5.5 — Re-plan & Engineering Learning**

Frozen product baseline: v1.5.0 shared contracts at `d0c0f72390fb04d5891cfc82937f977729488f25`.

Development was merged through `de5915e1fe20e5ff80086bc831b47cdea8f7ddfd` (Re-plan) and `689d56d4b726b4fa29b2ba913b5fe17766e63d8c` (Engineering Learning). Closure compatibility is re-evaluated against current main, which additionally contains the independent v1.5.1 SQLite-first slice. v1.5.5 does not acquire a hard dependency on v1.5.1 internals.

This ledger records development closure without converting unexecuted checks into PASS. Validation carrier is Local / Build Host only. GitHub Actions and CI are prohibited for this version.

## Development task status

| Task | Scope | Development state | Evidence |
|---|---|---|---|
| T155-01 | fact-backed Replan trigger/domain/dispositions | CODE-COMPLETE | `internal/replan/types.go` + deterministic tests |
| T155-02 | semantic fingerprint / bounded repeat-cycle policy | CODE-COMPLETE | `internal/replan/validate.go` + cycle/fingerprint tests |
| T155-03 | impact/dependency/accepted-work preservation | CODE-COMPLETE | Re-plan validator; explicit preserve/invalidate semantics |
| T155-04 | Direct LLM / AI Runtime proposal adapter | CODE-COMPLETE | existing `intelligence.Provider`; provider substitution/disagreement tests |
| T155-05 | atomic Control Plane activation / immutable history | CODE-COMPLETE | separate `ActivateReplan`; stale parent, authority and history tests |
| T155-06 | EngineeringRuleProposal domain | CODE-COMPLETE | `internal/engineeringlearning` |
| T155-07 | Outcome/evidence/conflict validation | CODE-COMPLETE | structured Outcome facts + evidence/conflict validation |
| T155-08 | authority-governed promotion | CODE-COMPLETE | shared/high-impact/AI authority gate; provider self-promotion rejected; handoff remains PROPOSED + append-only submit event until existing Git-backed `policygovernance.Store.Activate` succeeds |
| T155-09 | bounded integration/API surface | CODE-COMPLETE | additive `/api/v15/projects/{project}/replan` delegates to `Service.ActivateReplan`; SQLite ReplanHistory restart compatibility test |
| T155-10 | focused + full visible validation carrier | CARRIER-COMPLETE / ENV-BLOCKED | `scripts/validate-v1.5.5.sh`; not executed in current ChatGPT Web environment |
| T155-11 | held-out Hidden Validation carrier | CARRIER-COMPLETE / ENV-BLOCKED | `scripts/run-v1.5.5-hidden.sh`; external held-out pack unavailable here |
| T155-12 | Critical Journey / Closure / Release Qualification | NOT_RUN | journey definitions are frozen below; empirical predecessor gates are not PASS |

`CODE-COMPLETE` and `CARRIER-COMPLETE` are implementation states, not validation states and not Release PASS.

## Compatibility review against current main

Current main's v1.5.1 SQLite adapter persists the complete `controlplane.ProjectSnapshot` as JSON and validates it again on load. Therefore the additive v1.5.5 `ReplanHistory` field is structurally carried by the adapter rather than projected into a lossy v1.5.1-specific schema. The v1.5.5 closure patch adds an executable restart test with a non-empty ReplanHistory to turn this structural observation into Local/Build Host evidence when validation is run.

The v1.5.5 HTTP endpoint is additive and does not modify the frozen v1.5.0 `ControlPlane` interface: it uses a separate `ReplanControlPlane` capability assertion and then invokes the existing `Service.ActivateReplan`. Existing initial `AcceptPlan` / `ActivateDAG` immutability remains unchanged. `ActiveTaskDAG` remains the sole Project Flow.

Engineering Learning continues to reuse the existing Outcome facts, `intelligence.Provider`, `policygovernance` candidate/lifecycle, and Git-backed runtime policy authority. Closure review corrected the handoff so Authority approval produces only a `PROPOSED` governance value plus an append-only `submit` event; Learning no longer fabricates an in-memory `ACTIVE` state. Only the existing `policygovernance.Store.Activate` may cross the runtime activation boundary through Git CAS, exact pinned-snapshot reload and the final activate event. No second knowledge store, provider contract, rule engine, or Project Flow is introduced.

## Current validation ledger

| Gate | Status | Basis / blocker |
|---|---|---|
| v1.5.0 PRD/shared-contract compatibility review | PASS | additive Re-plan path; frozen initial lifecycle/provider/shared engineering contracts remain unchanged |
| Current-main cross-slice static review | PASS | v1.5.1 persistence stores full ProjectSnapshot JSON; bounded v1.5.5 compatibility test added; no hard dependency introduced |
| PR/static source review | PASS | implementation diffs reviewed; invalid output-shape escaping, incomplete Outcome facts, premature Learning ACTIVE handoff and missing T155-09 route were corrected before closure |
| Focused Go tests | ENV-BLOCKED | current ChatGPT Web environment has no executable private-repository checkout / Go build host |
| Re-plan HTTP integration test | ENV-BLOCKED | test carrier exists; requires Local/Build Host execution |
| SQLite ReplanHistory restart compatibility | ENV-BLOCKED | test carrier exists; requires Local/Build Host execution |
| Full visible regression | ENV-BLOCKED | requires Local/Build Host |
| Legacy Integration/E2E compatibility | ENV-BLOCKED | requires Local/Build Host and applicable brownfield prerequisites such as `DATABASE_URL` |
| Independent Review | NOT_RUN | no separate independent reviewer execution/evidence is attached to the frozen candidate |
| Hidden Validation | ENV-BLOCKED | external held-out pack/root is unavailable in the current environment |
| Critical Journeys | ENV-BLOCKED | executable candidate environment unavailable; journey definitions are frozen below |
| Release Qualification | NOT_RUN | mandatory empirical predecessor gates are not PASS |

## Local / Build Host validation

Run from a clean, frozen checkout of the candidate revision:

```bash
bash scripts/validate-v1.5.5.sh
```

If the brownfield integration surface is available:

```bash
DATABASE_URL='postgres://...' bash scripts/validate-v1.5.5.sh
```

Run the externally held-out pack only after visible implementation and visible validation are frozen:

```bash
TALLY_V155_HIDDEN_ROOT=/outside/repo/v1.5.5-hidden \
  bash scripts/run-v1.5.5-hidden.sh
```

Each carrier records revision, host/environment, command/result and UTC timestamp. A missing prerequisite must remain `ENV-BLOCKED`; it is never equivalent to PASS.

## Critical Journeys

CJ-155-01 — **Fact-backed proposal without direct activation**: start from one accepted v1.5.0 plan/DAG; introduce a supported failure/constraint/human trigger; generate a ReplanProposal through Direct LLM or AI Runtime; verify provider output alone changes no ProjectSnapshot authority.

CJ-155-02 — **Provider substitution and disagreement**: feed equivalent accepted facts to Direct LLM and AI Runtime; verify the same typed proposal contract; when semantic fingerprints disagree, require escalation/evidence rather than silently selecting one result.

CJ-155-03 — **Governed bounded HTTP activation**: submit a valid non-material ReplanProposal through `POST /api/v15/projects/{project}/replan`; verify Plan and ActiveTaskDAG advance together, exactly one immutable history entry is retained, and replaying the stale parent is rejected.

CJ-155-04 — **Accepted-work invalidation requires independent authority**: mark previously accepted work invalidated/superseded and declare material impact; verify missing authority blocks activation, provider self-authorization is rejected, and a valid independent human AuthorityDecision allows activation only after validation.

CJ-155-05 — **Restart/history continuity under current SQLite-first main**: activate a re-plan, close the SQLite store/process, reopen it, and verify the active Plan/DAG plus previous and activated revision ancestry survive unchanged.

CJ-155-06 — **Engineering Learning is governed, not self-promoting**: provide validated Outcome facts/evidence; generate an EngineeringRuleProposal; reject missing evidence/invalid confidence/unresolved conflicts; require independent Authority for AI/shared/high-impact proposals; verify promotion yields a `PROPOSED` governance handoff and append-only submit event, never an `ACTIVE` shortcut; after persistence, verify only existing `policygovernance.Store.Activate` can make the candidate runtime-observable through the pinned Git policy path.

CJ-155-07 — **Cycle budget fail-safe**: repeatedly propose semantically equivalent re-plans with new proposal/provider identities; verify semantic fingerprint repeat budget and total re-plan budget stop the loop.

CJ-155-08 — **v1.5.0 lifecycle remains frozen**: after a v1.5.5 re-plan, attempt to reuse initial `AcceptPlan` or initial `ActivateDAG`; verify immutable-stage protection remains intact and no second Project Flow can be created.

## Hidden Validation contract

The held-out pack must live outside the implementation repository and should cover at least:

- normal fact-backed Re-plan proposal → validation → activation;
- malformed/stale proposals, missing dispositions, DAG cycles and missing dependencies;
- retained accepted work compatibility and explicit invalidation/supersession;
- missing/rejected/self Authority cases;
- equivalent-proposal cycle/repeat escape attempts using changed IDs/timestamps/providers;
- Direct LLM / AI Runtime substitution and disagreement;
- ProjectSnapshot restart/history preservation including SQLite current-main compatibility;
- Outcome/evidence/provenance/conflict/low-confidence Engineering Learning cases;
- AI self-promotion attempts, shared/high-impact rule authority, and attempts to bypass `policygovernance.Store.Activate` with a premature ACTIVE value;
- negative checks proving no second Project Flow/provider/rule engine/runtime authority was introduced.

Only sanitized status/evidence may be copied into release records. Hidden fixtures/oracles remain outside the repository.

## Release qualification rule

Do **not** tag or publish v1.5.5 while any mandatory gate is `FAIL`, `NOT_RUN`, or `ENV-BLOCKED`. PR merge, static review, CODE-COMPLETE or carrier existence are explicitly insufficient.

Release Qualification may be changed to PASS only after the frozen candidate has revision-pinned evidence for focused tests, full regression, applicable Integration/E2E, independent review, held-out Hidden Validation and all v1.5.5 Critical Journeys. Only then may Version Closure be finalized and a v1.5.5 tag/release be created.
