# Tally Project Overrides

## v1.5.0 verification carrier

For Tally v1.5.0, **do not use GitHub Actions and do not use any CI system**.

This explicit project requirement overrides generic workflow text that names CI as the validation carrier. It does not waive or weaken any test, review, closure, hidden-validation, critical-journey, packaging or release-qualification requirement.

Rules:
- Do not create, modify, enable, trigger, wait for or depend on `.github/workflows/**` for v1.5.0.
- Do not substitute hosted CI, self-hosted CI or another continuous-integration service.
- GitHub is used for repository state, Issues, branches, commits, PRs, reviews, merges, tags/releases and provenance.
- Focused tests, integration tests, full regression, Hidden Validation, Critical Journeys and packaging are executed explicitly in the developer/agent environment or a designated Local/Build Host.
- Verification evidence records environment, revision, command, result and relevant failure output.
- Status vocabulary: `PASS`, `FAIL`, `NOT_RUN(reason)`, `ENV-BLOCKED(reason/evidence)`.
- An unexecuted command is never PASS. Release Qualification is never inferred from PR merge.

Refs: #5

## Tally v2 — prospective ADS v4.17.0 project adoption surface

- v4.adoption_level: A1_MANUAL_PROTOCOL

**Prospective v2 scope only:** This is a project-level choice of **A1 manual durable GitHub/file protocol** as the smallest currently supported v4 adoption level. No v2 automated reducer/controller, A2+ machine gate, runnable Domain Package Host, or Tally real-runtime interoperability is claimed. Selecting A1 never relaxes security, independent review, required Validation or Release gates.

- `v4.compatibility_mode`: PROSPECTIVE_V1_LEGACY_TO_V4_NATIVE_DOMAIN_PACKAGE; project v2 implementation and qualified real-Host test pending.
- `v4.runtime`: APPLICABLE — future Tally engineering UI, selected DomainHarness application and Local Agent/Host need actual runtime evidence; currently NOT_RUN.
- `v4.incident`: APPLICABLE — implementation must manage real Task/Host failures and repair/evidence; production incident recovery NOT_RUN.
- `v4.maintenance`: APPLICABLE — Tally v2 version baseline, source-bound support and regression/backport policy pending Product/L2/Release qualification.

**Source and authorization:** Proposed v2 exact standard: ADS 4.17.0 at `a45644d734fdf94da3e6f9c3f90ffe4ce3c25fda`; requested adoption [Issue #80](https://github.com/kaicreator-mm/tally/issues/80), [Draft PR #81](https://github.com/kaicreator-mm/tally/pull/81). Upstream source RQ is CONDITIONAL P2×3, and ADS `standards/PROJECT_ADOPTION.md` §2.1C(2) requires READY for untagged manual adoption; **do not claim v2 adoption has become eligible or effective while this discrepancy remains unresolved by the authorized standards owner**. No Product Freeze from this override alone.

**Verification carrier:** The v1.5.0 prohibition on CI above belongs **only to v1.5.0**. For v2, choose valid optional GitHub PR checks and explicit Local/Build Host targeted tests under frozen v2 Product/L2. CI unavailability can be recorded as scoped waiver where ADS permits but cannot convert required execution into PASS. Specific Node24 and release test profile remains PHYSICAL_L2_PENDING; no CI lane is inferred active from this text.

**Evidence:** GitHub Issues/PRs/Actions/current SHA and ADS source-bound reports are external facts; K owns one app State/Admission/Journal/Effect. Tally v2 must be true DomainHarness selected B/S/K/C packages (no standalone TS server retrofitted), task-meta observer failure never blocks Task/Verifier, C GitHub-only reader never fetches private task-meta data, Local Codex baseline and runx optional, held-out verifier OS principal/ACL denial witnessed before Hidden PASS.

**Historical v1.5.0:** Above no-CI scope, old ADS 1.2.1 pin and existing v1.5.x source/Release evidence remain historic, without retrospective amendment. Task/Claim/review subjects accepted under old standard continue to use their immutable original effective source.
