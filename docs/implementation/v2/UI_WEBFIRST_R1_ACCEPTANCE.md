# Tally v2 Web-first UI R1 — source/authority/acceptance matrix

Source owner: ChatGPT (source + unit tests). Local Agent: environment-only Node >=24, self-hosted Woodpecker, real selected DomainHarness Host/C and installed Chromium. Parent: #227, #244, #256; consolidated draft PR #258. This **is not** RQ or authorization. Existing Product/L2 freeze and ADS v4.18 #1241 govern all actions.

## Authority rule

Source order: GitHub native Task/Issue/Dependencies + immutable ADS v5 report published from task-meta → selected Engineering Business projections → one DomainHarness K → trusted Host/GitHub C/SQLite. UI is a **read-and-request** facade, never a second scheduler, GitHub writer, credential store, Domain Package hot-swap authority, or substitute independent Reviewer/Owner. No auth from planned DAG edge, self-reported report, local toggle, browser button, or empty response. Actual source inaccessible = `NOT_CONNECTED` / `UNKNOWN`, **not zero, PASS, or empty verified history**.

## Page contract

| Menu/view | UI read model | Authority / permitted action | Current source implementation | Acceptance |
|---|---|---|---|---|
| Header Project + Version | `/api/v2/catalog`, trusted catalog | Selection only, synchronized hash/Host membership | Selected `v2ReadModelHost.catalog` required | Reload, cross-project membership, no left-sidebar duplicate |
| 工作台 / 待办、审批、阻塞、概览 | `/api/v2/workbench` | Same Host human-action inbox; action proposal only | Source-verified read port; provider not installed => NOT_CONNECTED | One queue in 4 routes + header bell, no fake approvals |
| 项目研发 / 需求 | `/api/v2/requirements` | Product/L2 facts and approved freeze only | Version-scoped read port exists; provider required | Real Requirement→Epic→Story with acceptance and frozen rev |
| 项目研发 / Product 引导 | GET/POST `/api/v2/guided/{step,submit}` | Selected Host `readCurrentStep/dispatchStep` only; exact step/sha/CSRF | Port implemented; selected Host not yet composed | Invalid field/stale/foreign/cross-origin deny, no phantom freeze |
| 项目研发 / Task 列表、看板、详情 | `/api/v2/tasks` | Native WorkItem/Claim/Attempt with source SHA | Scoped selected Host port, no synthetic Issue-open=Task | Pagination, completed tasks, right project/version, deep links |
| 项目研发 / 版本 DAG | `/api/v2/dag` | Native blocked_by = evidence-backed, PLANNED = dashed | Host port+source guard; native reader still needed | Every version has its own immutable source/as-of graph |
| 项目研发 / Agent | `/api/v2/agents` | selected Host attempts, no Agent rankings | Scoped read port; no pretending [] is verified | Attempt, role, telemetry coverage with UNKNOWN |
| 项目研发 / QA | `/api/v2/qa` | Full/Hidden/Critical/Installed distinct; independent witness | Scoped read port; no fake PASS | Real QA layered source and gate judgments |
| 项目研发 / Release | `/api/v2/release` | Independent RQ + Owner + all gates + current exact permit + real selected K Host dispatch | Read + fail-closed button, **no React dispatch bound** | No-op is disabled; installed qualified effect required |
| 项目研发 / Deployment | `/api/v2/deployment` | Environment-specific Host deployment, separate from Release | Host read port; UI route still NOT_CONNECTED | Per-env receipts/health/rollback, never infer Release success |
| 项目研发 / Maintenance | `/api/v2/maintenance` | Native defect and immutable Casebook evidence | Scoped read port; no invented zero history | Defect→Repair/next-version linkage |
| 研发治理 / 元数据 | `/api/v2/metadata` | task-meta→ADS v5 published, no raw private data | Read port, missing Task denominator => UNKNOWN | Actual Task/Attempt/report/usage/as-of with source provenance |
| 研发治理 / Workflow | `/api/v2/workflow` | selected Business/Host workflow definitions + instances (not a second scheduler) | read-only governance ledger; producer needed | Definition revision, current instance, audit evidence |
| 研发治理 / Domain Packages | `/api/v2/domain-packages` | Exact BUILD selected K/S/B/C manifests, activation read-only | read-only ledger; manifest producer needed | ABI/hash/source and authorized future BUILD only |
| 研发治理 / 经验 | `/api/v2/experience` | Proposed/reviewed/attested learning only, no autonomous Meta adoption | read-only ledger; ADS/Meta producer needed | Evidence→proposal→review→future BUILD |
| 研发治理 / Audit | `/api/v2/audit` | Verified Claim/PR/CI/RQ ledger, never source-free PASS | read-only ledger; producer needed | Native source ref + SHA + event role coverage |
| 配置中心 / Agent | `/api/v2/settings` | Actual selected BUILD/Host provider; Host-only secrets | read UI exists; no Host config mutation port | No fake Build-selected switch or browser key |
| 配置中心 / Tally LLM | `/api/v2/settings` | Separate from executor Agent; optional provider selected in BUILD | read UI; edit disabled without trusted setter | Provider & prompt data minimization; no key in browser |
| 配置中心 / Git/ADS、Woodpecker | `/api/v2/settings` | Real Git scopes and self-hosted CI readbacks | read UI; CI Test disabled until Host callback | No GitHub Actions; missing OAuth != empty permissions |
| 配置中心 / Host、Storage、安全、部署、集成 | `/api/v2/settings` | Host-only trusted changes, scope cascades, deployment authorization | unsupported sections show NOT_CONNECTED, not fake settings | Settings validation + verified write receipts before enabling |

## R1 exact candidate testing

Unit-test source includes: latest-request generation/source identity, trusted catalog URL and Task identity, source/redaction/cross-version guards, release RQ/Owner/current permit negatives, Workflow/Package proposal distinctions, dynamic step revision and body whitelist, same-origin HTTP submission denial, and source-specific Task/QA/metadata coverage. Model tests are not installed End-to-End acceptance.

**Critical remaining blockers for version-only completion:**
1. Source-qualified **installed** selected Host adapters must be composed for catalog/native task dependencies/ADS metadata/Agent/QA/Workflow/packages; mere JavaScript port is not a connected service.
2. Woodpecker must be SUCCESS on the **combined exact HEAD**, with full Node24 tests/typecheck/Vite build; previous failure pipelines are not overridden by new pendings.
3. Browser route/scope and true J01–J08 complete user journey must be independently observed on Node24 Windows/real Host with negative test cases and no hidden data disclosure.
4. Accepted native CAS Claims and independent exact-SHA review must be read back; PR descriptions are not independent review evidence.
5. Release RQ/Owner grant, main/tag and deployed receipts stay separate and are NOT satisfied by UI source PR merges.

## Current policy

All R1 source work is **draft PRs**. `version/v2` base remains pinned at original source until accepted review. No GitHub Actions, no author-conducted self-approval, no main merge/tag/release inferred.
