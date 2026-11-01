# Tally v2 — Woodpecker-only CI validation carrier (#173)

Self-hosted Woodpecker is the **sole** CI validation carrier for `version/v2`
(Owner direction 2026-10-29, reaffirmed on #86/#125/#166; integrated by #173).
No GitHub Actions workflow is triggered, required, retried or used as v2
acceptance evidence: every `.github/workflows/v2-*.yml` and the generic
`trailer-check` exclude `version/v2` from their PR/push execution paths and are
retained with their historical results as provenance only. Legacy `main`/`master`
v1.5 maintenance is unchanged (`quality.yml`, `trailer-check` for all other base
branches, `reindex-check.yml`, and `verify.yaml` below).

This directory contains the following **separate, intentionally scoped** workflows:
- `verify.yaml` remains the inherited `main` push/manual Go maintenance verification.
- `v2-native-node24.yaml` — Node24 native C source-only check. Asserts the exact
  authored SHA (`CI_COMMIT_SHA` detached), enforces `Refs:` on all non-merge
  commits of the PR range via `scripts/check-trailers.sh` semantics (base derived
  from `CI_COMMIT_TARGET_BRANCH`, head = `CI_COMMIT_SHA`; push/manual events check
  the HEAD commit), then runs the native C test suite. Paths:
  `packages/tally-github-native-c/**`, `packages/tally-ads-v5-validator/**`, the
  profile itself.
- `v2-t06j-host-admission.yaml` — Engineering B + native C Node24 tests, immutable
  DomainHarness `v0.10.3@061376f4a023070f215f9feffad5c24d96d503cb` cloned from
  Git source, built and packed, then public Host admission and single-writer
  integration tests against the freshly packed SDK. Path-scoped to Engineering B /
  native C / ADS validator / `experiments/v2/t06d-host-claim-publication/**` on
  `version/v2`; the historical `task/tally-i166-zcode` replay branch is preserved.
- `v2-t06l-kernel-admission.yaml` — historical T06l accepted-Claim replay. It
  **intentionally keeps DomainHarness v0.10.2**
  (`cee14399d9d0af4e20e34b9111861b16a465b690`) as the recorded DH0.10.2 proof and
  does not move it to v0.10.3; new/common SDK qualification uses DH v0.10.3 via
  the T06j profile. Path-scoped to `experiments/v2/selected-business-build/**`
  and the replay helpers on `version/v2`; the historical
  `task/tally-t125-kernel-zcode` replay branch is preserved.
- `v2-ads-project-verifier.yaml` — pinned ADS v4.18 full-root project verifier for
  ADS-version / Product / L2 / DAG / CI-policy scopes. Clones
  `ai-development-standard` at the immutable pin
  `2c395c55569fe2edf3f4135f12d3d069e526d0b7`, requires **Python 3.14** (fails
  closed otherwise) and runs
  `scripts/verify_project_standard.py <tally-root> --standard-repo <ads-root> --require-resolution`.

The future T06m Local Agent profile (`v2-t06m-local-agent.yaml`) belongs
exclusively to #172 and is not created here.

All v2 profiles run on the already configured `backend: local` ECF VM (the local
Docker Woodpecker Agent is the actual runner) with Node24 and Python available on
the host. **Security:** Woodpecker Local backend runs commands directly on the
agent machine; permit PR-source execution only on a disposable, non-root,
unprivileged VM with no Woodpecker server tokens/host secrets and no shared
sensitive filesystem. If a trusted isolated VM is unavailable, do not enable PR
runs; use reviewed source and manual verification instead. No GitHub write token
or Task/Agent execution permission is required by any v2 profile.

Every profile: checks out `CI_COMMIT_SHA` detached and asserts exact HEAD, emits
pipeline number/event (`CI_PIPELINE_NUMBER`/`CI_PIPELINE_EVENT`), agent ID
(`CI_AGENT_ID`), step name/number and `CI_PIPELINE_URL`, records Node/Python
versions, and propagates literal command exit codes. On `pull_request` events
every applicable profile also enforces `Refs:` on all non-merge commits of the
PR range via `scripts/check-trailers.sh` (base = merge-base of the fetched
`CI_COMMIT_TARGET_BRANCH`, head = `CI_COMMIT_SHA`); push/manual events check
the HEAD commit. Woodpecker records step-level state/exit per step; test counts
are recorded only from separately executed exact-SHA runs and are never inferred
from pipeline success. Container tests are never called Windows OS-principal
isolation tests; real Windows OS principal, Codex authentication and
Builder/Verifier credential isolation are qualified on the local Windows Host,
not fabricated by a container CI fixture.

## Bounded retry policy for intermittent self-hosted Woodpecker

When an attempt is a **proven infrastructure failure** (VM provisioning, agent
disconnected, clone/network transient, no test steps), retry the **same pipeline
and exact SHA** up to **two additional times**, keeping every earlier attempt's
status/logs and exact provenance. Restart from the Woodpecker pipeline UI, not by
making an empty code commit. If an actual Node24/Python/test assertion fails or
exact SHA changes, do **not** blind-retry: inspect the failure, fix source in a new
SHA and rerun. After repeated infra failures, report `ENV_BLOCKED/NOT_RUN`, not
PASS. Only a completed real Woodpecker job with source/runtime/test output and
EXIT 0 is reusable for its exact SHA and test profile. Official API supports
restarting an existing pipeline; using it requires an authenticated Woodpecker
token and exact repository/pipeline identifier. Never paste tokens into Issue
comments or this workflow.

**Required process:** project Woodpecker PR webhook and matching disposable local
agent must actually be enabled; adding these YAML files does not by itself prove
server routing or make a running pipeline. Existing GitHub Actions failures at
`runner_id=0` are distinct provenance and are **not** Woodpecker test results;
follow ADS v4.18 frozen Product/L2 applicable CI policy before any version merge.

Refs: #125
Refs: #132
Refs: #160
Refs: #161
Refs: #166
Refs: #171
Refs: #172
Refs: #173
