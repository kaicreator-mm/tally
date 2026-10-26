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

## Tally v2 — prospective ADS v4.18.0 project adoption surface

- v4.adoption_level: A1_MANUAL_PROTOCOL

**Prospective v2 scope only:** This is a project-level choice of **A1 manual durable GitHub/file protocol** as the smallest currently supported v4 adoption level. No v2 automated reducer/controller, A2+ machine gate, runnable Domain Package Host, or Tally real-runtime interoperability is claimed. Selecting A1 never relaxes security, independent review, required Validation or Release gates.

- `v4.compatibility_mode`: PROSPECTIVE_V1_LEGACY_TO_V4_NATIVE_DOMAIN_PACKAGE; project v2 implementation and qualified real-Host test pending.
- `v4.runtime`: APPLICABLE — future Tally engineering UI, selected DomainHarness application and Local Agent/Host need actual runtime evidence; currently NOT_RUN.
- `v4.incident`: APPLICABLE — implementation must manage real Task/Host failures and repair/evidence; production incident recovery NOT_RUN.
- `v4.maintenance`: APPLICABLE — Tally v2 version baseline, source-bound support and regression/backport policy pending Product/L2/Release qualification.

**Source and authorization:** Prospective Tally v2 source standard is **ADS 4.18.0** at immutable integrated `main@2c395c55569fe2edf3f4135f12d3d069e526d0b7` (tree `cdf032f629668008a940993713a2bfaa219f5653`), with independently qualified frozen candidate `ac7e7e51de3a7ac27343f16a96568c0b0f2c5349` and [RQ READY P0/P1/P2=0/0/0](https://github.com/kaicreator-mm/ai-development-standard/issues/1327#issuecomment-6401840670); [upstream integration closure](https://github.com/kaicreator-mm/ai-development-standard/issues/1322#issuecomment-6402117697). This **satisfies the source-version choice's literal RQ READY** unlike earlier ADS v4.17 CONDITIONAL proposal #81, but project-local compatibility, immutable-version verifier, current Task/Claim pin safety and independent v2 adoption review still must complete *before activation*. Project v2 adoption [Tally #80](https://github.com/kaicreator-mm/tally/issues/80); previous draft #81 is historical/superseded candidate, not automatically merged. No Product Freeze or real runtime qualification from this override alone.

**v2 CI profile under ADS v4.18 §3.2: `custom`.** This is a **prospective v2-specific execution profile**; v1.5.0 Local/Build Host-only history remains unchanged. Rationale: v2 uses Node24/DomainHarness selected Domain Packages and SQLite P0, not the inherited Go+Postgres runtime. A v2 `custom` profile cannot weaken existing branch protection or independently applicable validation gates.

**Required checks, by independently inspected change scope:**
- **Every v2 PR:** actual current-HEAD GitHub `trailer-check` (`Refs:` for all non-merge commits), exact SHA/tree/base-currentness, applicable independent source/security Review, source-qualified GitHub receipt. Failed or unexecuted check is never PASS.
- **ADS-version / docs-only Product/L2/DAG changes:** full clean Tally Git root verified using the **exact pinned ADS revision** `scripts/verify_project_standard.py <tally-root> --standard-repo <ads-root> --require-resolution` (real exit code required); relevant markdown/links/requirements/P0/Fxx/Vxx/negative-fixture completeness and legal source/Review/permission compatibility. Real pinned 4.18 check on prior unchanged PR #85 HEAD `bee79439` **PASS/EXIT0** with Windows Python 3.14.7 [receipt](https://github.com/kaicreator-mm/tally/pull/85#issuecomment-6407643797); **new HEAD not yet run**.
- **Native v2 TypeScript/Host/Domain Package source:** Node24 installed public DomainHarness packed ABI/BUILD smoke, typecheck, bounded test contract and explicit Host SQLite CAS/Journal/Effect, recovery and GitHub source readback; a Local Agent/Hidden privilege-change additionally requires actual Windows/Ubuntu OS-principal denied probes. These checks are NOT_RUN on docs-only PRs, not fake PASS.
- **Release candidate:** real exact-SHA Full/Hidden/Critical Journeys/Packaging/independent RQ where required, separate from per-PR CI. No repeated Full per leaf by default.
- **PostgreSQL:** not part of v2 P0. Optional Postgres P1 adapter requires its own real integration/transaction suite if shipped, never implies legacy Go Postgres checks PASS.

**CI execution semantics (§3.3):** GitHub Actions PR runs on `ubuntu-latest`, with actual checkout version/source verified from each run (do not assume merge-ref equals exact PR HEAD); the inherited `.github/workflows/quality.yml` currently uses `actions/checkout@v4`, `setup-go@v5`, Go 1.23 and Postgres 16, and is not a Node24 v2 conformance lane. A v2-specific Node24/SQLite GitHub Actions workflow is **PLANNED, NOT_YET_INSTALLED**: fresh clean checkout at exact SHA, `node --version` checked for Node24, sealed DH package digest pinned, bounded deterministic checks. Windows/Ubuntu Local/Build Host carries physical ACL/Agent/Human-deny tests when a hosted runner cannot execute them. Record workflow path/entrypoint/shell/runner/runtime/exit and any merge-ref difference. One scoped rerun for transient failures, never reusing a prior SHA as current PASS.

**Current inherited CI failures and merge gate:** Existing `source-quality=FAIL` (Go `gofmt -l .`) and `postgres-integration=FAIL` (legacy Go tests) are real on both Product baseline and this doc-only candidate, whose diff changes no Go code. These are **BASELINE_PREEXISTING** by scope, **not waived/not PASS**. [CI owner #83](https://github.com/kaicreator-mm/tally/issues/83) must independently review the actual failed IDs, native required-check configuration and issue an explicit project-authorized applicability decision or route genuine Go repairs to a separate legacy maintenance concern. **Until then, `CI_APPLICABILITY=BLOCKED`, `MERGE_ELIGIBLE=NO`.** A project-local `custom` declaration cannot override a required red GitHub check or higher-authority rule.

**Assurance:** Any scope-limited CI choice leaves Product/Architecture Freeze, real security/human consent, accepted Claims, exact Task validation and Release Qualification unchanged. Current unfinished checks remain `NOT_RUN`, current red checks remain `FAIL`, and project-wide v2 version pin activation waits for distinct source/CI review.

**Evidence:** GitHub Issues/PRs/Actions/current SHA and ADS source-bound reports are external facts; K owns one app State/Admission/Journal/Effect. Tally v2 must be true DomainHarness selected B/S/K/C packages (no standalone TS server retrofitted), task-meta observer failure never blocks Task/Verifier, C GitHub-only reader never fetches private task-meta data, Local Codex baseline and runx optional, held-out verifier OS principal/ACL denial witnessed before Hidden PASS.

**Historical v1.5.0:** Above no-CI scope, old ADS 1.2.1 pin and existing v1.5.x source/Release evidence remain historic, without retrospective amendment. Task/Claim/review subjects accepted under old standard continue to use their immutable original effective source.
