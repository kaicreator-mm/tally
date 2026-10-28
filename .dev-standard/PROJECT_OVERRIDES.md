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

## Tally v2 — version-stage ADS v4.18.0 project adoption (effective on version/v2 only)

- v4.adoption_level: A1_MANUAL_PROTOCOL

**v2 version branch only:** The immutable ADS v4.18.0 source pin and one **A1 manual durable GitHub/file protocol** are effective in `version/v2` after [qualified PR #85](https://github.com/kaicreator-mm/tally/pull/85), actual merge `a2877bbd7d1ce87e755a01a25eb2341fc9467a78`. Historic `main` v1.5 ADS 1.2.1 and in-flight accepted legacy claims keep their original effective standard. No v2 automated reducer/controller, A2+ machine gate, runnable Domain Package Host, or Tally real-runtime interoperability is claimed. Selecting A1 never relaxes security, independent review, required Validation or Release gates.

- `v4.compatibility_mode`: PROSPECTIVE_V1_LEGACY_TO_V4_NATIVE_DOMAIN_PACKAGE; project v2 implementation and qualified real-Host test pending.
- `v4.runtime`: APPLICABLE — future Tally engineering UI, selected DomainHarness application and Local Agent/Host need actual runtime evidence; currently NOT_RUN.
- `v4.incident`: APPLICABLE — implementation must manage real Task/Host failures and repair/evidence; production incident recovery NOT_RUN.
- `v4.maintenance`: APPLICABLE — Tally v2 version baseline, source-bound support and regression/backport policy pending Product/L2/Release qualification.

**Source and authorization:** Prospective Tally v2 source standard is **ADS 4.18.0** at immutable integrated `main@2c395c55569fe2edf3f4135f12d3d069e526d0b7` (tree `cdf032f629668008a940993713a2bfaa219f5653`), with independently qualified frozen candidate `ac7e7e51de3a7ac27343f16a96568c0b0f2c5349` and [RQ READY P0/P1/P2=0/0/0](https://github.com/kaicreator-mm/ai-development-standard/issues/1327#issuecomment-6401840670); [upstream integration closure](https://github.com/kaicreator-mm/ai-development-standard/issues/1322#issuecomment-6402117697). This **satisfies the source-version choice's literal RQ READY** unlike earlier ADS v4.17 CONDITIONAL proposal #81, and Tally's project-local strict Python 3.14 verifier **PASSED** at approved PR #85 exact head `4331221d24d2cfc5e9b6a7d9bec126e6b4e47d43` ([run #40645893856](https://github.com/kaicreator-mm/tally/actions/runs/40645893856)), independent R6 SOURCE PASS (P0/P1/P2=0/0/1), then version-only integration read back. This **adopts the source standard for future v2 version-scoped work** without certifying runtime Tally/TaskMeta/Host/Human E2E or requalifying previously accepted claims. Project adoption [Tally #80](https://github.com/kaicreator-mm/tally/issues/80); previous draft #81 is historical/superseded candidate, not automatically merged. No Product Freeze or real runtime qualification from this override alone.

**v2 CI profile under ADS v4.18 §3.2: `custom`.** This is an **effective v2 version-scoped profile with a prospective implementation applicability matrix (Issue #138, review pending)**; v1.5.0 Local/Build Host-only history remains unchanged. Rationale: v2 uses Node24/DomainHarness selected Domain Packages and SQLite P0, not the inherited Go+Postgres runtime. A v2 `custom` profile cannot weaken existing branch protection or independently applicable validation gates.

**Required checks, by independently inspected change scope:**
- **Every v2 PR:** actual current-HEAD GitHub `trailer-check` (`Refs:` for all non-merge commits), exact SHA/tree/base-currentness, applicable independent source/security Review, source-qualified GitHub receipt. Failed or unexecuted check is never PASS.
- **ADS-version / docs-only Product/L2/DAG changes:** full clean Tally Git root verified using the **exact pinned ADS revision** `scripts/verify_project_standard.py <tally-root> --standard-repo <ads-root> --require-resolution` (real exit code required); relevant markdown/links/requirements/P0/Fxx/Vxx/negative-fixture completeness and legal source/Review/permission compatibility. Real pinned 4.18 check on PR #85 final source `4331221d24d2cfc5e9b6a7d9bec126e6b4e47d43` **PASS/EXIT0**, Python3.14.8 Ubuntu [run #40645893856](https://github.com/kaicreator-mm/tally/actions/runs/40645893856). On future Product/L2/DAG changes the now installed `.github/workflows/v2-ads-pinned-project-verifier.yml` has path triggers and must show *its own* exact applicable head PASS. Historical PR #85 PASS never transfers to another HEAD.
- **Native v2 TypeScript/Host/Domain Package source:** Node24 installed public DomainHarness packed ABI/BUILD smoke, typecheck, bounded test contract and explicit Host SQLite CAS/Journal/Effect, recovery and GitHub source readback; a Local Agent/Hidden privilege-change additionally requires actual Windows/Ubuntu OS-principal denied probes. These checks are NOT_RUN on docs-only PRs, not fake PASS.
- **Release candidate:** real exact-SHA Full/Hidden/Critical Journeys/Packaging/independent RQ where required, separate from per-PR CI. No repeated Full per leaf by default.
- **PostgreSQL:** not part of v2 P0. Optional Postgres P1 adapter requires its own real integration/transaction suite if shipped, never implies legacy Go Postgres checks PASS.

**CI execution semantics (§3.3):** GitHub Actions PR runs on `ubuntu-latest`, with actual checkout version/source verified from each run (do not assume merge-ref equals exact PR HEAD); the inherited `.github/workflows/quality.yml` currently uses `actions/checkout@v4`, `setup-go@v5`, Go 1.23 and Postgres 16, and is not a Node24 v2 conformance lane. Current native v2 Node24/SQLite GitHub Actions workflows **exist**: `.github/workflows/v2-native-claim-events.yml`, `v2-dh-native-github-sqlite.yml`, `v2-dh-selected-business-build.yml`, `v2-business-policy.yml`, `v2-github-native-readonly-c.yml`, `v2-native-task-workitem-source.yml`, and concern-specific consumers; check individual `pull_request.paths` and actual exact-head provider results. They do not form one universal automatically required check; source-only PASS is not Host/Agent/Release PASS. Windows/Ubuntu Local/Build Host carries physical ACL/Agent/Human-deny tests when a hosted runner cannot execute them. Record workflow path/entrypoint/shell/runner/runtime/exit and any merge-ref difference. One scoped rerun for transient failures, never reusing a prior SHA as current PASS.

### Prospective v2 executable-source CI selection — owner approval requested in #138

**Not yet a binding new required-check rule:** the table below is a **prospective** extension to the currently effective v2 `custom` profile, derived from actual currently installed workflows. #138's independent CI/Project Overrides owner must review/accept **before** applying it to a future code merge. Do not use it retrospectively to turn an existing red check green, to bypass native GitHub-required branch protection, or to make already-reviewed v2 code PRs automatically merge eligible.

| Source/diff scope | Existing provider to require prospectively | Actual selected proof and limitation |
| --- | --- | --- |
| All v2 changes | `trailer-check`, authored exact SHA and independent ADS #1241 review | One SHA, one scope; no inherited green from a different PR/head |
| Product/L2/Task DAG and standards docs | `v2-ads-pinned-project-verifier` + structural/semantic source review | Exact immutable ADS v4.18 Python3.14 verifier; no installed app proof |
| `packages/tally-github-native-c/**` | `v2-native-claim-events` + `v2-native-task-workitem-source` and `v2-github-native-readonly-c` | Node24 native Claim/dependency source and adversarial tests; *only* observed Claim, no CAS acceptance |
| `packages/tally-engineering-business/**` | `v2-business-policy` + `v2-dh-selected-business-build` | Actual Business policy and public selected K/B/C BUILD; no external Agent effect |
| `experiments/v2/selected-business-build/**` | `v2-dh-selected-business-build` + `v2-dh-native-github-sqlite` and concern-specific public Host workflow when introduced | Public installed DH v0.10.2, genuine GitHub read, selected SQLite Host, replay; inspection K-admitted does **not** authorize Task |
| New isolated v0.10.3 contract-first Tally source (`packages/tally-guided-intake/**` or `packages/tally-llm-contract/**`) | Its own exact-source Node24 schema/contract workflow and pinned frozen v0.10.3 schema/L2 blob equality | Current scoped 9/9 and 13/13 source tests, **not** v0.10.3 public installed runtime/Host/model call. Require separate public ABI/Host test when physically integrated |
| Go code/module/Go workflow or source renamed in/out of Go paths | Keep actual legacy `source-quality` and `postgres-integration` as affected, plus relevant v2 tests if mixed | Failed Go jobs are **FAIL** until fixed; not neutralized by TS migration |
| CI profile/helper/runner paths | Its own focused Node24/real-Git or pinned root verifier; independent CI owner approval of provider/applicability semantics | PR #155's advisory `v2-ci-scope-diagnostic` never grants required-check applicability or waiver |
| Actual selected Host Task/Agent/credential/Hidden source changes | Selected public K/Host/SQLite end-to-end **and** OS-principal credential deny/hidden tests where permission/physical behavior changes | Hosted source mocks or CI-only K read admission insufficient; physical evidence can remain `NOT_RUN` but then production/Task authorization blocked |
| Release candidate | Exact frozen Full/Hidden/Critical Journeys/Packaging and independent RQ per authorized release protocol | May re-use unchanged qualified evidence only at exact provenance, no source-only Release PASS |

**Fail closed:** if an expected workflow does not trigger, provider is absent, source SHA changed, independent reviewer disagrees, a native required rule remains red, or an app Host test did not run, record `TARGET_PROFILE_NOT_RUN/UNKNOWN` and withhold merge/production authorization. The old `quality` provider's historical failures remain exact **FAIL**; the independent #83 finding applies only to reviewed ADS/docs changes, **not** to TypeScript, runtime or CI-code PRs. Until a distinct #138 prospective decision, code PRs #149/#151/#153/#154/#155 remain held on applicability even with scoped green. This table does not request a new platform, gate, scheduler or CI runner.

**Inherited CI exact-source disposition:** The observed `source-quality=FAIL` (Go gofmt) and `postgres-integration=FAIL` (Go integration) remain literal `FAIL + BASELINE_PREEXISTING` on the Product baseline and this docs-only proposal. [Independent authorized #83 scoped ruling](https://github.com/kaicreator-mm/tally/issues/83#issuecomment-6408620557) verified `protected=false` for inspected branches and GraphQL `CheckRun.isRequired=false` for all three observed contexts on #82/#85; ruleset-list GET was **PLAN_LIMITED_403**, not evidence of an enumerated empty list. The ruling sets **`CI_APPLICABILITY=DECIDED_FOR_V2_DOCS_AND_ADS_ONLY`**: real trailer, exact source/root verifier, applicable independent review remain required; red legacy Go jobs are informational *only* for the checked docs/ADS-only write set, **not PASS, not an all-project waiver, not disabled**. For v2 executable code, change scope and security/full Release gates require new independent applicability and real Node24/Host/SQLite tests. This source observation **does not independently authorize ADS adoption or merge** until current exact HEAD and other project gates pass.

**Assurance:** Any scope-limited CI choice leaves Product/Architecture Freeze, real security/human consent, accepted Claims, exact Task validation and Release Qualification unchanged. Current unfinished checks remain `NOT_RUN`, current red checks remain `FAIL`, and project-wide v2 version pin activation waits for distinct source/CI review.

**Evidence:** GitHub Issues/PRs/Actions/current SHA and ADS source-bound reports are external facts; K owns one app State/Admission/Journal/Effect. Tally v2 must be true DomainHarness selected B/S/K/C packages (no standalone TS server retrofitted), task-meta observer failure never blocks Task/Verifier, C GitHub-only reader never fetches private task-meta data, Local Codex baseline and runx optional, held-out verifier OS principal/ACL denial witnessed before Hidden PASS.

**Historical v1.5.0:** Above no-CI scope, old ADS 1.2.1 pin and existing v1.5.x source/Release evidence remain historic, without retrospective amendment. Task/Claim/review subjects accepted under old standard continue to use their immutable original effective source.
