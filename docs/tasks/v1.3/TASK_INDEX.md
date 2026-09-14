# Task Index

**Total atomic tasks: 75**

- P0: 23
- P1: 52

## Task types

- `contract`: 21
- `gate`: 7
- `integration`: 18
- `leaf`: 28
- `spike`: 1

## Tasks

| ID | Phase | Kind | Wave | Depends | Complexity | Task |
| --- | --- | --- | --- | --- | --- | --- |
| `P0-01` | P0 | contract | W00 | — | S | [Job lease / heartbeat schema and SQL contract](P0/P0-01.md) |
| `P0-02` | P0 | contract | W00 | — | M | [Trace tuple correctness and project namespace SQL contract](P0/P0-02.md) |
| `P0-03` | P0 | contract | W00 | — | S | [Structured CommandSpec core](P0/P0-03.md) |
| `P0-04` | P0 | contract | W01 | P0-03 | M | [Failure repro persistence: structured columns/query contract](P0/P0-04.md) |
| `P0-05` | P0 | leaf | W00 | — | M | [Disposable Git worktree manager](P0/P0-05.md) |
| `P0-06` | P0 | contract | W00 | — | M | [Single GitWriter lock foundation](P0/P0-06.md) |
| `P0-07` | P0 | leaf | W00 | — | S | [Role-aware LLM proposal client](P0/P0-07.md) |
| `P0-08` | P0 | contract | W00 | — | M | [Derived project registry / ghost-project guard SQL contract](P0/P0-08.md) |
| `P0-09` | P0 | integration | W02 | P0-01, P0-02, P0-04, P0-08 | S | [Regenerate sqlc for P0 contracts](P0/P0-09.md) |
| `P0-10` | P0 | leaf | W03 | P0-01, P0-09 | M | [Lease-aware Queue claim, heartbeat and stale reclaim](P0/P0-10.md) |
| `P0-11` | P0 | integration | W03 | P0-02, P0-09 | M | [Trace reindex adapts to project-namespaced identity](P0/P0-11.md) |
| `P0-12` | P0 | leaf | W03 | P0-03, P0-04, P0-09 | M | [Failure RunSpec uses structured commands end-to-end in domain layer](P0/P0-12.md) |
| `P0-13` | P0 | integration | W03 | P0-08, P0-09 | M | [Project identity validation at failure/environment/deploy ingest boundaries](P0/P0-13.md) |
| `P0-14` | P0 | leaf | W00 | — | M | [Production SpecDefect sink adapter](P0/P0-14.md) |
| `P0-15` | P0 | leaf | W01 | P0-03 | S | [Dependency command execution stops using strings.Fields](P0/P0-15.md) |
| `P0-16` | P0 | leaf | W01 | P0-06 | M | [ContentStore Git writes through GitWriter](P0/P0-16.md) |
| `P0-17` | P0 | leaf | W01 | P0-06 | M | [Requirement signing Git writes through GitWriter](P0/P0-17.md) |
| `P0-18` | P0 | leaf | W01 | P0-06 | M | [Harvest catalog Git writes through GitWriter](P0/P0-18.md) |
| `P0-19` | P0 | leaf | W01 | P0-06 | M | [Knowledge Store Git writes through GitWriter](P0/P0-19.md) |
| `P0-20` | P0 | integration | W00 | — | M | [Implement declared deploy job handler or retire unsupported declaration consistently](P0/P0-20.md) |
| `P0-21` | P0 | integration | W04 | P0-05, P0-07, P0-10, P0-12, P0-14 | L | [Worker P0 consolidated wiring: leases, structured failure runner, disposable workspace, SpecDefect, LLM roles](P0/P0-21.md) |
| `P0-22` | P0 | integration | W00 | — | M | [Retire legacy /tasks false-green semantics from authoritative Task language](P0/P0-22.md) |
| `P0-23` | P0 | gate | W05 | P0-11, P0-13, P0-15, P0-16, P0-17, P0-18, P0-19, P0-20, P0-21, P0-22 | M | [P0 correctness integration gate](P0/P0-23.md) |
| `P1-01` | P1 | contract | W06 | P0-23 | M | [TaskContract types, parsing and revision hash](P1/P1-01.md) |
| `P1-02` | P1 | contract | W06 | P0-23 | M | [VerificationContract types, binding provenance and frozen revision](P1/P1-02.md) |
| `P1-03` | P1 | contract | W06 | P0-23 | S | [ExecutionPolicy minimal P1 contract](P1/P1-03.md) |
| `P1-04` | P1 | contract | W06 | P0-23 | M | [Agent Runtime Protocol core types and capability registry](P1/P1-04.md) |
| `P1-05` | P1 | contract | W06 | P0-23 | M | [Immutable ChangeSet domain and identity](P1/P1-05.md) |
| `P1-06` | P1 | contract | W06 | P0-23 | M | [Evidence core: fact, validity, applicability and provenance](P1/P1-06.md) |
| `P1-07` | P1 | contract | W06 | P0-23 | M | [Deterministic Hunk Inventory](P1/P1-07.md) |
| `P1-08` | P1 | contract | W06 | P0-23 | S | [Measurement and cost-plane fact model](P1/P1-08.md) |
| `P1-09` | P1 | contract | W06 | P0-23 | S | [Holdout Evidence Store abstraction](P1/P1-09.md) |
| `P1-10` | P1 | contract | W06 | P0-23 | S | [Task/Verification derived-index SQL contract](P1/P1-10.md) |
| `P1-11` | P1 | contract | W06 | P0-23 | M | [CodegenRun / ChangeSet persistence SQL contract](P1/P1-11.md) |
| `P1-12` | P1 | contract | W06 | P0-23 | L | [VerificationRun / Evidence / Exposure persistence SQL contract](P1/P1-12.md) |
| `P1-13` | P1 | contract | W06 | P0-23 | M | [HumanApproval / review timing / LLM usage persistence SQL contract](P1/P1-13.md) |
| `P1-14` | P1 | leaf | W07 | P1-01 | M | [TaskContract static validator and scope rules](P1/P1-14.md) |
| `P1-15` | P1 | leaf | W07 | P1-01, P1-02, P1-06 | M | [Verification freeze validator and planned independence coverage](P1/P1-15.md) |
| `P1-16` | P1 | leaf | W07 | P1-02, P1-06 | M | [Evidence Qualification evaluator](P1/P1-16.md) |
| `P1-17` | P1 | leaf | W07 | P1-06 | M | [Effective source-independence derivation](P1/P1-17.md) |
| `P1-18` | P1 | leaf | W07 | P1-05, P1-06 | L | [Information independence: content exposure, selection exposure and applicability](P1/P1-18.md) |
| `P1-19` | P1 | leaf | W07 | P1-01, P1-06 | S | [Functional acceptance coverage](P1/P1-19.md) |
| `P1-20` | P1 | leaf | W08 | P1-17, P1-18, P1-19 | M | [Acceptance-level independence coverage](P1/P1-20.md) |
| `P1-21` | P1 | leaf | W07 | P1-01, P1-02, P1-04 | M | [ContextPack redacted projection and context hash](P1/P1-21.md) |
| `P1-22` | P1 | leaf | W09 | P1-19, P1-20 | M | [Evidence-derived Task state engine](P1/P1-22.md) |
| `P1-23` | P1 | leaf | W09 | P1-20 | M | [HumanApprovalEvidence and Independence Waiver domain](P1/P1-23.md) |
| `P1-24` | P1 | leaf | W07 | P1-04 | M | [FakeRuntime reference adapter](P1/P1-24.md) |
| `P1-25` | P1 | contract | W07 | P1-04 | M | [Runtime conformance harness](P1/P1-25.md) |
| `P1-26` | P1 | leaf | W07 | P1-02 | M | [Verification execution protocol and environment identity](P1/P1-26.md) |
| `P1-27` | P1 | leaf | W07 | P1-09 | M | [P1 filesystem Holdout Store implementation outside project worktree](P1/P1-27.md) |
| `P1-28` | P1 | leaf | W08 | P1-03, P1-16, P1-26 | M | [Visible incremental validation executor](P1/P1-28.md) |
| `P1-29` | P1 | leaf | W08 | P1-05, P1-26, P1-27 | L | [Held-out final validation executor](P1/P1-29.md) |
| `P1-30` | P1 | leaf | W08 | P1-24, P1-25 | M | [Session Isolation Canary](P1/P1-30.md) |
| `P1-31` | P1 | integration | W07 | P1-10, P1-11, P1-12, P1-13 | S | [Regenerate sqlc for P1 contracts](P1/P1-31.md) |
| `P1-32` | P1 | integration | W08 | P1-01, P1-02, P1-10, P1-14, P1-15, P1-31 | L | [Git artifact loader and task/verification derived indexing](P1/P1-32.md) |
| `P1-33` | P1 | integration | W08 | P1-05, P1-06, P1-11, P1-12, P1-31 | L | [Run/ChangeSet/Verification/Evidence persistence adapters](P1/P1-33.md) |
| `P1-34` | P1 | integration | W10 | P1-08, P1-13, P1-23, P1-31 | M | [Approval and measurement persistence adapters](P1/P1-34.md) |
| `P1-35` | P1 | leaf | W07 | P0-05, P1-03, P1-05 | M | [Codegen workspace session and dependency-cache boundary](P1/P1-35.md) |
| `P1-36` | P1 | spike | W07 | P1-04 | XS | [Codex runtime surface probe](P1/P1-36.md) |
| `P1-37` | P1 | leaf | W08 | P1-04, P1-25, P1-35, P1-36 | L | [CodexRuntime adapter against probed surface](P1/P1-37.md) |
| `P1-38` | P1 | integration | W09 | P1-16, P1-19, P1-20, P1-28, P1-29, P1-33 | L | [Verification orchestration for one immutable ChangeSet](P1/P1-38.md) |
| `P1-39` | P1 | integration | W09 | P1-03, P1-05, P1-24, P1-33, P1-35 | L | [P1 vertical workflow kernel](P1/P1-39.md) |
| `P1-40` | P1 | integration | W10 | P1-07, P1-19, P1-20, P1-23, P1-33 | L | [Patch Review accounting and acceptance mapping](P1/P1-40.md) |
| `P1-41` | P1 | integration | W11 | P1-08, P1-33, P1-34 | M | [Runtime usage and human-control measurement recorder](P1/P1-41.md) |
| `P1-42` | P1 | integration | W09 | P1-21, P1-32 | M | [P1 Context builder from refs and redacted verification projection](P1/P1-42.md) |
| `P1-43` | P1 | integration | W12 | P1-30, P1-37, P1-38, P1-39, P1-41, P1-42 | L | [Worker integration for v1.3 Codegen/Verify vertical path](P1/P1-43.md) |
| `P1-44` | P1 | integration | W11 | P1-22, P1-32, P1-33, P1-40 | L | [Minimal P1 Tasks/Changes web UI](P1/P1-44.md) |
| `P1-45` | P1 | contract | W11 | P1-15, P1-18, P1-22, P1-30, P1-40 | L | [v1.3 architecture invariant tests](P1/P1-45.md) |
| `P1-46` | P1 | integration | W13 | P1-43, P1-44, P1-45 | M | [Update implementation-facing project instructions to v1.3 baseline](P1/P1-46.md) |
| `P1-47` | P1 | gate | W13 | P1-43, P1-44, P1-45 | M | [E2E: standard Task with visible + held-out PASS](P1/P1-47.md) |
| `P1-48` | P1 | gate | W13 | P1-43, P1-45 | M | [E2E: held-out FAIL blocks VERIFIED](P1/P1-48.md) |
| `P1-49` | P1 | gate | W13 | P1-18, P1-43, P1-45 | M | [E2E: exposure fact flips information independence](P1/P1-49.md) |
| `P1-50` | P1 | gate | W10 | P1-05, P1-33, P1-39 | M | [E2E: immutable ChangeSet and supersession fact](P1/P1-50.md) |
| `P1-51` | P1 | gate | W14 | P1-41, P1-47, P1-48, P1-49 | S | [P1 metric baseline report generation](P1/P1-51.md) |
| `P1-52` | P1 | gate | W15 | P1-46, P1-47, P1-48, P1-49, P1-50, P1-51 | M | [P1 Vertical Trust Gate review](P1/P1-52.md) |
