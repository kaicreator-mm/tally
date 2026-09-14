# Tally Documentation & Implementation Status

**Current program:** v1.4.1 closure-repaired development baseline
**Updated:** 2026-09-15
**Development mode:** Brownfield evolution

## Executive status

```text
v1.4 implementation                    COMPLETE — historical baseline
v1.4.1 visible implementation          IMPLEMENTED
v1.4.1 closure repair                  IMPLEMENTED — validation evidence recorded separately
v1.4.1 Safe Export                     REPAIRED / gated on exact committed candidate
v1.4.1 Current-doc consistency         GATED by scripts/docscheck
v1.4.1 Hidden Validation (P6-08)       PENDING_EXTERNAL
release qualification                  NOT COMPLETE until required external gates run
```

The live task authority remains `docs/tasks/v1.4.1/STATUS.md`. Historical task reports are evidence of the original implementation program; the closure-repair baseline additionally fixes issues found by post-program code audit.

## Current implementation facts

The source now contains the v1.4.1 product chain rather than merely an authorized task pack:

- Intent proposal → bounded dynamic form → Confirm → atomic Freeze;
- production Contract / Risk / Assurance authority and risk-adjusted Acceptance;
- human-centered Work / Review / Rules / Integration projections;
- Rule-first EngineeringPolicy governance;
- Codex plus canary-gated DeepSeek Harness Runtime Adapters behind one `agentruntime.Runtime` protocol;
- local Git Control Plane L0–L3;
- append-only Outcome facts with database-atomic replay idempotency;
- deterministic Safe Export and repository hygiene gates.

## Runtime integration lock

```text
Tally domain
  ↓
internal/airuntime.Executor             provider-neutral AI operation boundary
  ↓
cmd/internal/airuntimelegacy            single transitional transport adapter
  ↓
internal/llm                            legacy OpenAI-compatible transport

Coding Task execution:
Tally Go
  ↓
internal/agentruntime.Runtime
  ├─ runtimes/codex → external Codex CLI
  └─ runtimes/dsh   → external DSH launcher (canary; live evidence required)
```

The public AI Runtime service transport is not falsely claimed as complete: only the domain boundary and centralized transitional adapter are present today.

## Strong / external-evidence boundary

See `docs/releases/v1.4.1_STRONG_BASELINE.md` for the capability matrix. DSH real-provider validation and P6-08 Hidden Validation remain external evidence; when unavailable they are recorded as `SKIP/NOT_RUN(environment)`, never inherited PASS.

## Hidden Validation

P6-08 is intentionally external and withheld from the implementation repository. It must run against the frozen candidate before release qualification. Current status: **PENDING_EXTERNAL**.
