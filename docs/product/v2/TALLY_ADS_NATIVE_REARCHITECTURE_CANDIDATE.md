# Tally v2 — ADS-native Engineering Control Plane (Product / L2 Candidate)

> **User-directed redesign, 2026-10-16; CANDIDATE ONLY.** Re-architect Tally wholly around the pinned `kaicreator-mm/ai-development-standard` (ADS) contract and move **runtime ADS Toolkit/devkit implementation** to Tally. Historical frozen v1.5.x releases are not edited, requalified or re-labeled. This is **not an executable Task DAG, code implementation, or freeze**.
>
> Version label `v2` is a proposed **new major architecture line**, not proof of released v2. Owner planning source: [Tally #55](https://github.com/kaicreator-mm/tally/issues/55); standard version Product input [ADS #1147](https://github.com/kaicreator-mm/ai-development-standard/issues/1147).

## 1. Goal and product non-negotiables

**Tally is the executable organization-level engineering platform; ADS is its internal, independently consumable engineering standard.** No Tally-specific normative development workflow that contradicts current effective ADS version. ADS remains implementable by Codex/Claude/other agents without Tally; Tally must support local CLI independently of its server.

- **ADS owns** L1→PRD→L2→Task DAG→Task/Review/Validation→Closure→Release semantics, current normative schemas/phase conditions, actor declaration semantics, organization development environment profile schema and per-project immutable ADS version/adoption rules.
- **Tally owns** Project Portfolio, per-project operational orchestration state, Agent execution & parallel scheduling, lifecycle of local/remote execution and workspaces, real validation/Review invocation, GitHub SCM actions and observed outcomes, operational agent evidence storage, project change proposals, and actual opted-in cross-project updater.
- **GitHub owns** actual Git repository objects, native Issues/Issue Dependencies, accepted Claim events, PRs, merge/Review/Actions state and immutable release subjects for ADS-native GitHub projects. Tally stores an indexed/materialized execution projection and local Attempt/Session/Host state. Do not create a second independently authoritative GitHub task state.
- **LLM/Agent claims** are proposals until accepted through actual current ADS-mandated proof. **GitHub login is transport identity**, not logical actor identity: trust self-declared `operator_kind/operator_id/role/session_ref` for logical role attribution and existing policy, while platform grants, real effects, executed tests, exact Sha/currentness and actual external human rights remain independent facts. Do not add Owner credential gate.

## 2. Why a domain-model rewrite (not an indiscriminate code rewrite)

Actual Tally main has merged v1.5.2 SSH/Docker environment [#49](https://github.com/kaicreator-mm/tally/pull/49), v1.5.3 Git/GitHub operations [#44](https://github.com/kaicreator-mm/tally/pull/44), v1.5.4 multi-Agent workflow/scheduler [#52](https://github.com/kaicreator-mm/tally/pull/52), v1.5.5 learning/replan [#53](https://github.com/kaicreator-mm/tally/pull/53); preserve tested algorithmic/transport components **only behind new ADS-native contracts**, after targeted source/behavior audit.

| Existing Tally source/domain | v2 disposition | Reason |
|---|---|---|
| Active TaskDAG as sole internally authoritative Project Flow | **REMODEL** as ADS/GitHub native task/dependency projection + separately owned execution attempts/ready queue | Eliminates conflict with ADS's native Issue Dependencies and durable Claim/PR facts |
| RequirementsBaseline / PlanningRevision / ArchitectureRevision | **MAP** to ADS L1/PRD/L2 immutable artifacts and exact source/pins | Existing Tally planning records cannot supersede frozen ADS artifact authority |
| Tally TaskContract/CompletionContract, Policy/Human Authority | **REMODEL** from resolved exact ADS version + project overrides, preserving useful deterministic validation code | One effective policy, no contradictory Owner-signature or blanket human gate |
| `internal/agentruntime` Codex/DSH adapters and IntelligenceProvider | **REUSE** narrow provider/runtime ports, no second Agent Runtime | ADS does not own model orchestration |
| Local/SSH/Docker execution, cancellation, capabilities | **REUSE/HARDEN** | Existing real implementation, no need for new environment framework |
| SQLite-first stores + optional PostgreSQL, append-only evidence/audit | **REUSE SCOPED STORAGE**, define new materialized project/attempt records and prospective migrations | No new organization-level central database by default, SQLite default |
| GitHub provider, Git actions, ChangeSet, candidateverify, closedloop | **REUSE** transport and proof algorithms, remove duplicated authority decisions | Must check exact native target HEAD, PR, actual tests and compliance |
| v1.5.4 role scheduler, restart, conflict suppression | **ADAPT** to ADS task/role/Claim semantics | Multi-Agent work exists; role attribution must follow ADS v4.12+ |
| v1.5.5 planning/replan and engineering learning | **ADAPT** proposal/impact mechanisms into ADS versioned flow; suggestions not self-changing normative standard | Prevent second Product Freeze/Task DAG/Replan authority |
| `devkit` scripts and candidate `mise/Worktrunk` support | **IMPORT INTO TALLY TOOLKIT** with bounded repair and providers | ADS standards repo should not own persistent org execution/host orchestration code |

Current Tally `.dev-standard/VERSION` is a v1.2.1 **YAML** format, while v4.12+ ADS describes a `repository/version/revision` exact pin. Tally v2 **explicitly adopts a qualified immutable ADS successor**, but freezes existing Tally history under its original pin and never guesses source compatibility. Migration uses a separately approved project-local update operation and readback, not a bulk rewrite of frozen v1.5.x runs. Verify real v4.12.1 and v4.13 qualification at the time of admission; draft planning is not a release.

## 3. New conceptual architecture

```text
Tally v2 Server / Console       Tally v2 CLI (headless, server optional)
          |                                    |
          +------------ Tally Core ------------+
                           |
              ADS Revision / Policy Resolver
    [ADS repo immutable revision + project override + source facts]
                           |
          Project Portfolio / GitHub Projection
        [GitHub native Issue/Deps/PR/Claim/Git records]
                           |
              Work Planning / Execution
    [Tally runtime DAG *projection* + scheduler/Attempt states]
                           |
                 ADS Task Contract
                   /          \
      AgentRuntime x Host    Toolkit provider execution
      (local/SSH/Docker)     (mise, native Git/Worktrunk,
                             bare mirror, worktree, caches)
                   \          /
           Immutable ChangeSet / real validation
                           |
       ADS Review / Gate policy / exact evidence mapper
                           |
             GitHub write/readback + release facts
                           |
           Project Portfolio + bounded learning
```

The Project Portfolio controls **cross-project dispatch/resource budgets**, not a new normative workflow. Project Issue dependencies remain GitHub-native facts; a local snapshot/ready queue is **only a verifiable derived execution projection**, with explicit stale/unknown outcomes. Runtime Attempt lifecycle, host health and Process/Session state legitimately live in Tally; those are **not GitHub Issue States**.

## 4. ADS Toolkit becomes Tally DevKit implementation

Package under Tally (final module paths chosen at L2):
- `tally dev env explain|doctor` — source-attributed ADS organization/project/host toolchain resolution; show conflicts and actual installed state.
- `tally dev bootstrap --plan|--apply` — idempotent host setup, preferred mise provider, explicit fallback, only needed tools.
- `tally dev repo ensure|mirror sync` — retain devkit Git bare mirror but not duplicate remote refresh; offline object identity distinct from remote freshness.
- `tally dev workspace ensure|resume|inspect|cleanup` — linked worktrees, optional Worktrunk provider, safe native-Git fallback; exact task base; no automatic destructive cleanup.
- `tally dev validate --scope affected|candidate` — execute project-selected commands and actual required assertions, local/CI/candidate dedup; test NOT_RUN != PASS.
- `tally dev conformance check` — resolve effective immutable ADS schemas and deterministic contract checks; schema PASS != GitHub/Release authorization.
- `tally ads upgrades plan|prepare|apply|rollback` — project-opted-in real ADS compatible update PR from qualified immutable version, approval/checkpoint and readback; **one writer inside Tally**, not a second ADS updater.

**CLI must operate without Tally daemon/DB/login** for local setup/workspace/validate, using only project/organization config, Git and local credentials where required. Server calls **the same core library**, while optional Agent Skills/MCP are only interfaces to that library; no second CLI reimplementation in the ADS standard repo. Organization profile and schema remain in ADS; Tally stores machine-specific path/proxy/secret refs outside committed organization policy.

External tools: mise preferred but not mandated/authority resolver; Worktrunk is a candidate UX provider only after Windows/Ubuntu+mirror native proof, native Git mandatory fallback; workmux/devcontainer/devenv optional, never mandatory.

## 5. One canonical cross-repo ADS Task identity & state contract

Canonical identity (candidate, final schema after L2):

```text
ProjectIdentity = (repository, integration_ref, effective_ads_sha, policy_source_ref)
TaskIdentity    = (repository, github_issue_number, task_contract_revision)
TaskGraph       = read_only_reconcile(native_issue_dependencies, frozen_Task_DAG)
Execution       = (task_id, attempt_id, host_ref, agent_operator_id, exact_base_sha)
Candidate       = (changeset_sha/tree, source_branch, actual_validation_tuple)
Outcome         = (native_claim_ref, actual_pr_and_review, merge_readback, closure_ref)
```

- Tally can predict readiness from cached snapshot but **MUST revalidate fresh GitHub-native dependency and effective Claim/current SHA before consequential dispatch/merge**. Source conflict, missing permission or unknown state => hold affected operation; other independent projects may continue.
- No blanket "any open Issue/PR blocks all work" rule. Distinguish unrelated work from conflicting Claim/source/freeze; request bounded evidence, never assert safe without it.
- Source-writing Work Item follows existing ADS rule/Claim acceptance. GitHub username never substitutes self-declared actor role or independently executed validation result.
- Frozen legacy v1.5 Tally records remain immutable historical evidence; new v2 effective ADS revision adopted per project, not per Agent session.
- Direct LLM and AI Runtime remain optional planning providers, not authority. An Agent changing its model/Session does not create a new Task or invalidate reliable exact evidence.

## 6. Product modes

1. **Standalone:** Tally CLI can bootstrap/inspect/validate a repository using its exact ADS pin and project overrides, no persistent server.
2. **Single-project ADS-native:** Tally monitors one GitHub repo and runs eligible Task DAG with multiple Agent roles, exact immutable change proof and PR/Review.
3. **Organization Portfolio:** Tally shows multiple repositories, versions, blocked/ready/review/CI/Release status, host capability/cost; scheduling is bounded and based on source-checked project policy.
4. **ADS upgrade:** A qualified ADS released candidate is first prospective, then one scoped project update PR, then authorized activation for new/checkpointed Task; retain old active Tasks.
5. **Offline/degraded:** local plan/diff/check may work against proven exact Git objects, but lack of remote Claim/Issue/PR currentness blocks actions requiring live GitHub proof; does not fabricate PASS.

## 7. Proposed engineering sequence (not frozen Task DAG)

- **P0 — L1 Product & source comparison:** current ADS qualified pin and current Tally source; semantic conflict map; decide one canonical project task authority, prospective legacy migration, Toolkit ownership and package boundary.
- **P1 — L2 contracts / compatibility:** ADS policy resolver, canonical GitHub task/Claim/Review/Release projection, Tally runtime Attempt state separation, version compatibility and fail-closed invalidation; re-use verified exact source.
- **P2 — Core single-project read-only shadow:** resolve one adopter repo/ADS pin and project overrides, GitHub Issue Dependencies/PR/Claim source, show traceable status with no writes.
- **P3 — CLI Toolkit standalone:** minimal org/host profile & safe workspace/Bootstrap/validation with native Git fallback; one real Windows/Ubuntu scenario when relevant to shipped support.
- **P4 — One actual ADS Task end-to-end:** frozen Task/Claim + true base -> existing AgentRunner/environment -> exact worktree/ChangeSet -> focused real checks and selected Review -> actual GitHub PR/merge readback -> legitimate Task completion.
- **P5 — Multi-Agent / Portfolio:** adapt existing scheduler, role flow and persisted attempts; prove 2 Agents in distinct worktrees, 2 separate repos, independent source states, restart, conflict/refusal.
- **P6 — Version updater/learning:** Tally executes opted-in v4.13 release update PR and rollback; retain bounded, evidence-backed next-ADS suggestions.
- **Closure:** one exact frozen Tally v2 candidate; actual applicable CLI/server/buildhost/OS/critical journeys; source review according to ADS effective policy, no duplicate full-suite loops; Release Qualification separate from PR PASS. Actual multi-project scope claims ONLY after executed pilot.

Task DAG implementation: **reuse one concern per PR**, favor parallel and staged Contract → Core → Integration → UI → Validation. Do not create a separate Issue for each field/provider/test example, no new scheduler beyond existing Tally.

## 8. Acceptance: real critical journeys and falsifiers

| ID | Desired result | False/unsafe result |
|---|---|---|
| A01 | Tally resolves two projects' distinct immutable ADS pins, loads correct referenced policy | Reads mutable ADS `main/latest` or unverified v1.2 YAML as v4.12 INI |
| A02 | GitHub Issue dependency causes Tally Ready/Hold projection; after source readback it advances | Persisted independent Tally DAG overrides native GitHub blocker |
| A03 | Local `tally dev bootstrap/doctor/workspace` works without daemon | Tally server/DB required just to open task worktree |
| A04 | Two Agents run concurrently in separate source-pinned worktrees, restart safe | Wrong-base/dirty `--force` deletion or same branch double-writer |
| A05 | Agent self-declared role is consumed as logical identity | GitHub login used as Agent identity or compulsory Owner credential |
| A06 | Reviewer/Validator/Claim/Release are based on actual current-source evidence and authority | Agent summary/mock/old SHA fabricated as PASS or Release |
| A07 | Two repos shown in Portfolio with per-repo readiness and real host status | Portfolio progress fabricated from task labels or unrelated open PRs |
| A08 | Qualified compatible ADS release creates a targeted opt-in update PR and later lawful adoption | Upgrade silently mutates active Task/Claim effective pin |
| A09 | Local and CI selected by evidence/execution-placement; one final candidate closure | Repeated full root suite per tiny repair or skipped check == PASS |
| A10 | Unavailable GitHub/host leaves explicit BLOCKED/NOT_RUN while nondependent work continues | Broad global deadlock or fake GATE pass |
| A11 | Historical v1.5 frozen record remains readable/in its actual qualified state | Retroactive mutation or false migration of historical evidence |

## 9. Explicit non-goals

No from-scratch rewrite of validated Git/process adapters, no new Agent thought-state persistence, generic BPM, cloud Kubernetes Agent fleet, universal Nix/docker requirement, obligatory always-on MCP, centralized new knowledge DB, reimplemented GitHub Issue dependency store, artificial Owner auth, new ADS normative versions auto-accepted without source/qualification, or claiming multi-project Release qualification from static contracts.

**Status: PRODUCT_DIRECTION_RECORDED; DESIGN_CANDIDATE_ONLY; PRD_FREEZE=NO; L2_FREEZE=NO; TASK_DAG_FREEZE=NO; IMPLEMENTATION=NOT_STARTED; REAL_HOST_TESTS=NOT_RUN.**
