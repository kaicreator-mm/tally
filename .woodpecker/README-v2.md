# Tally v2 — Woodpecker Node24 native source workflow

This directory contains two **separate, intentionally scoped** workflows:
- `verify.yaml` remains the inherited `main` push/manual Go maintenance verification.
- `v2-native-node24.yaml` adds a TypeScript/Node24-native GitHub C **source-only** check for pull requests targeting `version/v2`, pushes to `version/v2` and explicit manual runs.

The new workflow reuses the already configured `backend: local` ECF VM and Node24. **Security:** Woodpecker Local backend runs commands directly on the agent machine; permit PR-source execution only on a disposable, non-root, unprivileged VM with no Woodpecker server tokens/host secrets and no shared sensitive filesystem. If a trusted isolated VM is unavailable, do not enable PR runs; use reviewed source and manual verification instead. No GitHub write token or Task/Agent execution permission is required by the workflow.

The commands mirror the existing GitHub-native Node24 tests rather than introducing a new runner, Task authority, new package manager, retry loop, SDK or release gate. They install the existing Ajv/YAML/Octokit dependencies with ignored install scripts, run `node --test` across the native C test suite, and only print `TALLY_V2_WOODPECKER_NATIVE_NODE24_SOURCE_PASS` after literal test EXIT 0. The recorded `WOODPECKER_V2_SOURCE_HEAD` must match the intended reviewed PR subject. This is **source-only** evidence: no positive GitHub accepted CAS Claim, privileged Host writer proof, Kernel task effect, external Agent, Full/Hidden or Release.

## Bounded retry policy for intermittent self-hosted Woodpecker

When an attempt is a **proven infrastructure failure** (VM provisioning, agent disconnected, clone/network transient, no test steps), retry the **same pipeline and exact SHA** up to **two additional times**, keeping every earlier attempt's status/logs and exact provenance. Restart from the Woodpecker pipeline UI, not by making an empty code commit. If an actual Node24 test assertion fails or exact SHA changes, do **not** blind-retry: inspect the failure, fix source in a new SHA and rerun. After repeated infra failures, report `ENV_BLOCKED/NOT_RUN`, not PASS. Only a completed real Woodpecker job with source/Node24/test output and EXIT 0 is reusable for its exact SHA and test profile. Official API supports restarting an existing pipeline; using it requires an authenticated Woodpecker token and exact repository/pipeline identifier. Never paste tokens into Issue comments or this workflow.

**Required process:** project Woodpecker PR webhook and matching disposable local agent must actually be enabled; adding this YAML does not by itself prove server routing or make a running pipeline. Existing GitHub Actions failures at `runner_id=0` are distinct provenance and are **not** Woodpecker test results; follow ADS v4.18 frozen Product/L2 applicable CI policy before any version merge.

Refs: #125
Refs: #132
Refs: #160
Refs: #161
