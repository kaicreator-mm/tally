# Tally v1.5.0 Hidden Validation Protocol

Hidden Validation is a **release-level held-out gate** for the frozen v1.5.0 candidate. It is not an implementation aid and its fixtures/oracles must not be committed to this repository.

## Preconditions

Hidden Validation may start only after all visible implementation work for the candidate is frozen:

1. the candidate revision is a concrete Git commit, not a moving branch expression;
2. the implementation checkout is clean;
3. visible source/tests/docs for the candidate are frozen;
4. the held-out pack lives outside the implementation repository;
5. the selected Build Host/runtime/environment is recorded;
6. any held-out runtime role that requires `session.isolation` has independent conformance evidence; a runtime declaration alone is insufficient.

The repository wrapper is:

```bash
TALLY_V15_HIDDEN_ROOT=/secure/held-out/tally-v1.5.0 \
  bash scripts/run-v1.5-hidden.sh
```

The wrapper refuses to run if the held-out root is inside the implementation repository or if the candidate has uncommitted changes.

## External pack contract

`TALLY_V15_HIDDEN_ROOT` must contain at least:

```text
manifest.json
run.sh
```

`run.sh` receives:

```text
--repo <absolute implementation checkout>
--revision <exact frozen commit>
--result-dir <external/sanitized result directory>
```

The held-out pack is responsible for checking that its manifest is compatible with the requested product/version/revision and for preventing raw fixture/oracle leakage into the implementation workspace.

Exit code contract used by the repository wrapper:

- `0` — PASS;
- `3` — ENV-BLOCKED (required external dependency/runtime/environment unavailable);
- any other non-zero value — FAIL.

## Information-separation rule

Implementation authority may receive only the already-approved safe repair projection exposed by the existing brownfield closed loop. Hidden Validation must not expose to coding/repair agents:

- protected test source or path when path identity itself is protected;
- exact hidden assertions or expected values;
- hidden fixtures/oracle implementation;
- raw held-out stdout/stderr when it would reveal protected material;
- hidden asset content or content digests;
- richer verdict detail than the existing allowlisted safe projection.

The held-out runner may publish sanitized aggregate evidence such as scenario ID, category, `PASS|FAIL|ENV-BLOCKED`, duration, and an opaque verification-run reference. It must not copy raw protected material into repository files, PR comments, Agent prompts, or implementation logs.

## Required hidden scenario families

The exact stimuli remain external. The manifest must cover at least these behavior families.

### HV-REQ — Requirements / architecture

- ambiguous requirement is not silently normalized into accepted truth;
- conflicting requirements remain explicit/governed rather than guessed;
- missing acceptance criterion is rejected;
- missing architecture/design artifact blocks the lifecycle stage;
- architecture constraint violation cannot be hidden by an Agent claim.

### HV-DAG — TaskDAG / Project Flow

- duplicate node rejection;
- missing predecessor rejection;
- self dependency / duplicate dependency rejection;
- cycle rejection;
- blocked → ready only when dependency facts permit it;
- two independent nodes can be parallel-ready without inventing a second Project Flow;
- restart/reload preserves the Active TaskDAG and readiness state.

### HV-COMP — Completion / authority / audit

- missing required artifact blocks completion;
- missing or non-PASS evidence blocks completion;
- missing/non-PASS deterministic check blocks completion;
- missing action record blocks completion;
- required authority pending/rejected blocks completion;
- required Git operation absence blocks completion;
- illegal lifecycle transition is rejected;
- duplicate/contradictory project facts do not silently overwrite append-only evidence.

### HV-INT — Intelligence providers

- Direct and AI Runtime adapters satisfy the same typed provider contract;
- provider unavailable/error does not mutate Project Flow;
- Proposal/Assessment/Recommendation cannot activate a DAG or satisfy CompletionContract by itself;
- provider substitution changes provenance, not deterministic project authority.

### HV-EXEC — Local task execution

- Local execution resolves the environment-aware WorkspaceRef correctly;
- cancellation/interruption is distinct from candidate verification failure;
- retained workspace restart does not require Agent session resume;
- lost workspace restart begins a governed fresh task attempt;
- resume unsupported with allowed fresh fallback does not attempt an invalid resume first;
- resume failure with allowed fallback performs at most the bounded fresh fallback;
- owner primary checkout is rejected as executable candidate workspace;
- no-change Agent completion produces no immutable candidate;
- candidate deletion unsupported by the current full-content ChangeSet model fails closed.

### HV-BROWNFIELD — Existing authority reuse

- production execution uses the existing `internal/agentruntime` protocol, not a second runtime abstraction;
- production verify/repair/accept uses the existing `internal/closedloop` + `candidateverify` authority;
- Agent `completed` remains a claim and cannot directly become evidence/acceptance;
- each executable attempt runs in a Tally-owned disposable Git worktree;
- immutable ChangeSet is durably associated with a real CodegenRun before verification evidence is recorded;
- repair candidate supersession lineage remains append-only;
- held-out execution fails closed when required isolation evidence is absent;
- Codex/DSH use the same runtime protocol and do not gain runtime-name-specific project authority.

## Result record

A release evidence record must include at minimum:

- frozen candidate commit;
- hidden-pack manifest/version identifier (not hidden contents);
- Build Host identifier;
- runtime adapter/version/model coordinates used where applicable;
- start/end time;
- aggregate scenario count by category;
- final status `PASS|FAIL|ENV-BLOCKED`;
- opaque result directory / verification-run references.

## Current status

`PASS` — executed on Build Host `Nilar` (Windows 10.0.26200 x64, Go 1.27.1) against frozen candidate `0a5f7307ec5263240d726cd49b00232c39ebf684` at UTC 2026-09-15T23:43:47Z via:

```bash
TALLY_V15_HIDDEN_ROOT=C:/Dev/vCode/hidden_validation/tally/v1.5.0 TALLY_V15_HIDDEN_OUT=C:/Dev/xCode/tally-validation/v1.5.0-hidden   bash scripts/run-v1.5-hidden.sh
```

Result: 41/41 held-out scenarios PASS across all required families (HV-REQ, HV-DAG, HV-COMP, HV-INT, HV-EXEC, HV-BROWNFIELD); sanitized aggregate evidence `C:/Dev/xCode/hidden_validation` → `C:/Dev/xCode/tally-validation/v1.5.0-hidden/20260915T234347Z/` (summary.txt, scenarios.tsv, aggregate.txt); candidate checkout verified clean before and after the run; no GitHub Actions / no CI.

Pack provenance: authored 2026-09-16 by the local Build-Host validation agent (which did not author the v1.5.0 implementation) black-box from this protocol's required scenario families after the candidate froze; fixtures/oracles remain outside the implementation repository at `C:/Dev/vCode/hidden_validation/tally/v1.5.0/`. A future independently-authored pack can re-run the same wrapper as an additional gate.
