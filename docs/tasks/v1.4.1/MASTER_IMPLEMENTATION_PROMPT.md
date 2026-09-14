# Tally v1.4.1 — Master Implementation Prompt

You are the implementation coordinator for Tally v1.4.1.

Your job is to complete the **entire visible v1.4.1 task program**, using low-cost models wherever the L3 pack says LOW and high-capability models only where it says HIGH or requires HIGH review.

Do not redesign the product. Execute the authorized brownfield plan.

## 0. Baseline

Expected starting HEAD:

`979a55ad3ba5e514bb80f0e012aa36e0e6577eba`

Before editing:
- verify repository identity and HEAD;
- read current working-tree state;
- never reset/clean unrelated user changes;
- run the validation preflight from `docs/tasks/v1.4.1/SOURCE_BASELINE.md`;
- if the real HEAD differs, inspect whether the v1.4.1 replacement docs were committed after this baseline; continue only if source semantics still match the task pack, otherwise report `BLOCKED_BY_SOURCE_FACT`.

## 1. Read authority in this order

1. `AGENTS.md`
2. `docs/product/Tally_PRD_v1.4.1_Final.md`
3. `docs/tasks/v1.4.1/DECISION_LOCKS.md`
4. assigned Task file
5. the Task's appended L3 Reference Pack
6. `docs/tasks/v1.4.1/TASK_DAG.md`
7. `docs/tasks/v1.4.1/PATH_OWNERSHIP.md`
8. `docs/tasks/v1.4.1/MODEL_ASSIGNMENT.md`
9. current source and executable tests as implementation facts
10. L2/L3 research evidence

Historical v1.4 documents are evidence, not current implementation authority.

## 2. Hard architecture locks

- Tally is brownfield. Do not rewrite it.
- `internal/closedloop` continues to own Verify / Repair / Accept / Escalate.
- `internal/agentruntime.Runtime/Session` is the **only core Coding Harness runtime protocol**.
- Do not create a new peer `ExecutionBackend` interface/lifecycle.
- Codex remains out-of-process through `runtimes/codex`.
- DeepSeek Harness must also be out-of-process through `runtimes/dsh`.
- If DSH requires an SDK, use a subprocess bridge; do not embed the Harness implementation into the Tally Go process.
- Runtime output is an Agent Claim, not Acceptance.
- Runtime has no durable Git authority.
- GitWriter/workspace/ChangeSet remain existing authorities.
- EngineeringPolicy Git content remains runtime policy authority; governance is a layer around it.
- Outcome keeps the existing five semantic kinds unless a separately authorized ADR changes it.
- Hidden/held-out verification material never enters implementation prompts.
- AI Runtime DIRECT/AUTO/ORCHESTRATED planning modes are outside Tally implementation scope.

## 3. Model routing

Read `docs/tasks/v1.4.1/MODEL_ASSIGNMENT.md`.

For each task:
- `Execute=LOW`: delegate implementation to the cheapest capable coding model.
- `Execute=HIGH`: use a high-capability reasoning/coding model.
- `Review=HIGH`: after implementation and focused tests, run an independent high-capability semantic review before merge/commit acceptance.

Do not promote every task to HIGH "for safety". The task pack was deliberately L3-enriched so most implementation can be LOW-cost.

## 4. Execution strategy

Follow `TASK_DAG.md` and `PARALLEL_EXECUTION_PLAN.md`.

Critical authority path:

A0 → P0 → P1.

Do not treat UI, Rules, DeepSeek Harness or Git integration as release-complete before P1-08 proves the authority chain.

Parallel work is allowed only when `PATH_OWNERSHIP.md` permits it.

One visible task should produce one reviewable ChangeSet/commit. The coordinator owns commits; delegated implementation agents do not commit independently.

## 5. Per-task protocol

For every task:

1. Read the exact Task file.
2. Read its L3 pack in this order:
   Tests → Contract/Interface → Core Implementation → Failure Handling → External reference.
3. Run the focused tests before changing code when practical.
4. Inspect the named implementation seam only; do not read/rewrite the whole repository by default.
5. Implement the smallest valid change.
6. Add at least one negative/failure-path test for new behavior.
7. Run focused tests.
8. Run relevant `test/arch` guards.
9. If the task requires HIGH review, perform it before acceptance.
10. Report:
   - status DONE/BLOCKED/PARTIAL;
   - changed files;
   - semantic changes;
   - exact commands/results;
   - baseline failures;
   - remaining risks;
   - Decision Lock touched or not;
   - resulting commit hash.

If the source contradicts the task pack, stop that task with `BLOCKED_BY_SOURCE_FACT`. Do not invent a replacement architecture.

## 6. Special task rules

### P0 / P1
These establish execution authority. Fail closed.
- Confirm→Freeze must be atomic and Git-backed.
- risk inputs must derive from immutable confirmed evidence.
- Contract compilation must occur before Runtime session start.
- required assurance / unmatched risk must be able to block Acceptance.

### P3 Rules
- do not create another policy engine;
- DB proposal state is not runtime activation;
- Approve & Activate becomes real only through canonical policy content committed to Git;
- historical replay cannot fabricate counterfactual outcomes.

### P4 DeepSeek Harness
P4-03 is a mandatory HIGH-model spike.
- pin exact DSH version/commit;
- verify real headless/subprocess/bridge protocol;
- verify workspace/cwd;
- verify provider/model route;
- verify Cancel/Close and process cleanup;
- return GO / GO_WITH_LIMITS / NO_GO.

If NO_GO:
- do not hack around upstream behavior;
- continue independent lanes;
- mark DSH production tasks honestly blocked/deferred;
- final gate must reflect the limitation.

If GO/GO_WITH_LIMITS:
- implement `runtimes/dsh` using existing `agentruntime.Runtime/Session`;
- DSH remains secondary/canary initially;
- no direct protected Git or Acceptance authority.

### P5 Git
v1.4.1 release scope is local L0–L3.
Do not expand into remote push/PR/MR unless explicitly authorized.
Conflicts and base drift are typed non-success outcomes; no force/history rewrite.

### P6 Verification / Outcome
- reproduce migration verification disagreement before changing verifier behavior;
- never weaken held-out or required gates to make a benchmark green;
- Outcome events attach to authoritative product outcomes, not generic workflow failures.

## 7. Visible release gates

Complete every visible gate honestly.

A visible command that cannot run must be recorded as:
`NOT_RUN(reason)` or `SKIP(reason)`

Never fabricate PASS.

Before final visible closeout, run in a dependency-ready environment:
- `go build ./...`
- `go vet ./...`
- `go test -count=1 ./...`
- required integration suites
- runtime conformance suites
- export deny-list validation

## 8. Hidden Validation — do not inspect early

The Hidden Validation Pack is distributed separately from the repository.

Expected SHA-256:

`67863529795f31b4a45848ee50e7746c49ef5ae91267b2ceb9392d8a400e53e0`

**Do not open, list, extract, search or summarize the Hidden Validation Pack during implementation.**

Only after:
- all visible tasks are terminal;
- all visible gates are complete;
- candidate HEAD is frozen;
- implementation work has stopped;

may the coordinator mount the external hidden pack.

Then use a fresh HIGH-capability validator and follow the hidden pack's own `HIDDEN_REVIEW_PROMPT.md`.

The implementation agents must not be given the hidden scenarios.

If Hidden Validation returns FAIL:
- do not rewrite the report;
- create a remediation cycle from the discovered defects;
- fix only the evidenced defects;
- rerun affected visible regressions;
- freeze a new candidate HEAD;
- rerun the full affected hidden scenario set.

## 9. Final output

At completion provide:

1. final HEAD;
2. task status 57/57 with commit mapping;
3. visible gate summary;
4. model-usage summary: LOW tasks / HIGH tasks / HIGH reviews;
5. Codex Runtime status;
6. DSH Runtime status and exact pinned version;
7. Contract/Risk/Assurance authority evidence;
8. Rule governance evidence;
9. Git L0–L3 evidence;
10. migration-verification result;
11. Outcome production evidence;
12. safe-export artifact audit;
13. Hidden Validation report and verdict;
14. remaining known limitations;
15. final verdict:
    `PASS | PASS_WITH_KNOWN_LIMITATIONS | FAIL`.

Do not claim v1.4.1 complete until the separate Hidden Validation stage has run.
