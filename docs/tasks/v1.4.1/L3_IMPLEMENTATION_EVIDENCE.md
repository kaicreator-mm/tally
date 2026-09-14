# Tally v1.4.1 — L3 Implementation Evidence Research

**Status:** COMPLETE  
**Source baseline:** `979a55ad3ba5e514bb80f0e012aa36e0e6577eba`  
**Tasks enriched:** 57  
**LOW execution:** 40  
**HIGH execution:** 17

## 1. Objective

L3 converts the L2 Task DAG into implementation-ready task packets for lower-cost coding models.

For every task, the reference order is:

> Tests → Contract / Interface → Core Implementation → Failure Handling → Examples / Docs

The goal is not to give a model more context. The goal is to give it the **minimum correct context** so that it copies Tally's existing architectural pattern instead of inventing a new one.

## 2. Main L3 findings

### 2.1 Low-cost models can execute most tasks

40/57 tasks are classified LOW execution after adding task-level references.

The reason is that Tally already has strong internal exemplars:

- Git snapshot loaders;
- strict parsers and content-derived revisions;
- append-only facts;
- runtime conformance tests;
- worktree isolation;
- expected-base Git CAS;
- held-out evidence protection;
- v1.4 read-model projections;
- architecture guard tests.

Lower-cost models should imitate these existing seams rather than design new abstractions.

### 2.2 Strong models are reserved for semantic risk

The 17 HIGH tasks concentrate on:

- authority-chain design;
- held-out/freeze semantics;
- risk derivation;
- acceptance/assurance safety;
- evidence-bounded policy replay;
- rapidly changing DeepSeek Harness integration;
- subprocess lifecycle;
- semantic Git integration/merge;
- Outcome acceptance transaction;
- migration verifier root cause;
- final adversarial release judgment.

### 2.3 DeepSeek Harness needs exact-version pinning

Current DeepSeek Harness `master` exposes provider/model/reasoning selection in the tool-subagent path and named subagent providers, but the project is explicitly developer-preview and current community reports show model-selection behavior has changed across recent builds.

Therefore P4-03 remains a mandatory HIGH-model spike:

- pin exact commit/package version;
- verify headless protocol;
- verify provider/model selection on that exact build;
- verify cancellation and process teardown;
- then implement the Tally adapter.

No implementation agent may code the production adapter from documentation assumptions alone.

### 2.4 Do not add JSON Schema dependency for Dynamic Forms by default

JSON Schema is used as a conceptual reference for bounded declarative validation. Tally already has an architectural dependency whitelist (`test/arch/deps_test.go`).

The intended implementation is a small Tally-owned typed form schema unless an explicit dependency ADR changes that decision.

### 2.5 Git merge implementation should preflight conflicts

For P5-03, current Git supports `git merge-tree --write-tree`, which performs a real merge calculation without touching the working tree/index and reports clean/conflicted status.

Use it as an implementation reference/preflight where the supported Git version permits it, while preserving Tally's expected-base and GitWriter authority.

## 3. Low-cost execution contract

A LOW-model agent must:

1. read its task file;
2. read Decision Locks / PRD Amendments;
3. read the listed tests first;
4. read only the listed contracts/core seams;
5. implement a minimal patch;
6. add/extend failure-path tests;
7. run focused tests + relevant architecture tests;
8. stop on source mismatch.

It must not:

- redesign architecture;
- add generic abstractions;
- add provider SDKs;
- add dependencies casually;
- broaden event enums;
- weaken fail-safe behavior;
- 'fix' unrelated code.

## 4. Strong-model review pattern

Some tasks are LOW execution but HIGH review.

This is intentional:

> cheap model implements deterministic code; strong model reviews semantic authority and boundary compliance.

This is the preferred cost/quality balance for v1.4.1.

## 5. Authority

Implementation authority order remains:

1. source invariants/tests;
2. `DECISION_LOCKS.md`;
3. `PRD_AMENDMENTS.md`;
4. L3 task reference pack;
5. L2 Task Pack;
6. original PRD.

External repositories never override Tally's internal contracts.
