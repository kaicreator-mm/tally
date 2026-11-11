# Tally v2 Core MVP Scope Amendment R1

**Date:** 2026-11-12  
**Status:** OWNER_DIRECTED_SCOPE_AMENDMENT / INDEPENDENT_REVIEW_PENDING  
**Coordinator:** [Tally #259](https://github.com/kaicreator-mm/tally/issues/259)  
**Amendment record:** [Tally #392](https://github.com/kaicreator-mm/tally/issues/392)  
**Target:** `version/v2`  
**Source baseline:** `7ebb1cdd0a79fc82cc938e35f852a80dd8e15332`

## 1. Purpose

Shorten Tally v2 delivery by completing one genuinely usable, secure, end-to-end Agent-oriented ALM path and explicitly deferring marginal multi-Agent and measured-learning scenarios to v2.1.

This amendment is additive. It does not rewrite, delete, or backdate the frozen Product/L1 or L2 source artifacts. It changes the current v2 delivery acceptance scope through a new versioned record.

## 2. Frozen baselines — already complete

The previous controller report incorrectly stated that Product/L2 Freeze had not completed. Native GitHub records show:

| Layer | Frozen artifact | Exact source | Freeze record | Integration |
|---|---|---|---|---|
| Product/L1 | PR #157 | `93fe091aca5e8e367235f831bd416907bdf32d1d` | [#63 comment 6465782232](https://github.com/kaicreator-mm/tally/issues/63#issuecomment-6465782232) | merged; version merge `58dc8d931aeb7e5d46df100120438a53f26b5f19` |
| Semantic L2 | PR #158 | `1cd254694de9c6b36a1c66a193c10fe29901525f` | [#65 comment 6466641925](https://github.com/kaicreator-mm/tally/issues/65#issuecomment-6466641925) | merged into `version/v2` |

Product R9 and L2 R7 source reviews both returned P0/P1/P2=0/0/0. The Product baseline retains 24 P0, 82 vectors and 47 legacy dispositions. L2 retains DomainHarness Domain Package-first, one selected Kernel, B/S/C/Host boundaries and its negative semantic cases.

Some historical Markdown headers still show old `FREEZE_PENDING` / `L2_FREEZE_READY=NO` text. These are stale narrative fields, not the operative authority. The native Freeze records and merged PRs are authoritative. Do not repeat the full Product/L2 reviews or mutate the original frozen source to erase history.

## 3. V2 Core MVP — required

The v2 release candidate must demonstrate one complete, real lifecycle path:

1. **Project and requirements:** create/open a project and version; capture/revise requirements, Product and semantic L2; retain source-linked decisions and formal Freeze/readback semantics.
2. **Task planning:** create/read a version-qualified Task DAG; preserve exact Task identities and native dependencies; show live progress without treating ordinary Task state/updatedAt changes as structural DAG corruption.
3. **One real Agent provider:** configure and use exactly one real Builder provider. The current Tally source already implements a Codex CLI provider; use this path first. Do not require both Zcode and Codex, do not build a second provider adapter for v2. If an already-integrated Zcode path can satisfy the same exact acceptance, it may substitute; provider count remains one.
4. **One real Builder Task:** accepted Task scope, selected K/Host admission, isolated worktree, actual Agent execution, candidate source/commit/PR, durable GitHub readback and honest terminal status.
5. **Independent Review and Repair:** a Reviewer with a separate execution identity/source context checks the exact candidate; findings are repaired by the responsible Builder; repaired SHA receives fresh applicable verification. The Reviewer may be a human or independent Web Reviewer; it need not be a second Local Builder provider.
6. **CI and QA:** Node syntax/type preflight before applicable CI; real Woodpecker exact-SHA status and logs; defect/test-case/acceptance linkage; distinguish PASS, FAIL, UNKNOWN, STALE and NOT_RUN.
7. **Version delivery:** exact-SHA Release Qualification, current target verification, authorized version delivery and actual merge/tag readback. No Release PASS from a PR merge or unit tests alone.
8. **Basic maintenance:** preserve released history, accept a defect and prepare the next version without rewriting prior Release evidence.

### Non-negotiable safety and architecture

- True DomainHarness Domain Package-first implementation; one selected Kernel authority.
- GitHub/ADS native facts remain authoritative; SQLite is a derived projection/cache.
- Single-writer and exact write-set protections; reject overlapping or expired Claims.
- Builder receives no GitHub write credential; K/Host/C owns privileged GitHub writes.
- Session, repository, Origin/CSRF, protected-path and secret-redaction boundaries remain.
- Real independent Reviewer and applicable Hidden/held-out verification remain.
- Ambiguous external effects remain UNKNOWN and are reconciled without blind replay.
- task-meta remains optional/nonblocking. Missing telemetry is UNKNOWN, never Task PASS or Task BLOCK by itself.

## 4. Explicit deferrals to v2.1

| Original Product/L2 area | V2 disposition | Preserved in v2 |
|---|---|---|
| P0-20 / L2-N27 positive control: two simultaneously active independent Builders | DEFERRED_V2_1 | one real Builder; exact write-set, single-writer and overlap rejection |
| P0-19 full multi-Agent orchestration and Agent-to-Agent delegation | DEFERRED_V2_1 | one Builder, durable GitHub handoff and separate Reviewer |
| P0-12 multi-provider registry and advanced delegation | DEFERRED_V2_1 | one configured real provider with minimum role/scope controls |
| P0-23 measured Task Learning efficacy and automatic experience promotion | DEFERRED_V2_1 | immutable Task/Review/Defect history and optional source-linked advisory context; no quality-gain claim |
| Advanced telemetry coverage and experience-feedback analytics | DEFERRED_V2_1 | task-meta fail-open/nonblocking semantics and truthful UNKNOWN |
| Multi-project portfolio analytics, broad dashboards and cross-project optimization | DEFERRED_V2_1 | single-project/single-version core lifecycle |
| Runx/remote SSH broker as a core execution dependency | DEFERRED_V2_1 | one selected local/external Agent path |
| Generic BPM, recursive optimization and automatic Domain Package promotion | DEFERRED_V2_1 | bounded workflows and explicit human decisions |

The positive two-Builder physical acceptance is not a v2 release blocker after this amendment. Existing #176–#179 and #383 evidence remains preserved and reusable; it is not erased or retroactively reclassified. The negative overlap/conflict checks remain mandatory.

## 5. One-Agent provider decision

The current Tally implementation is specifically wired to the selected Local Codex provider:
- `experiments/v2/selected-business-build/local-codex-host.mjs`
- `experiments/v2/selected-business-build/local-codex-shim.mjs`
- `experiments/v2/selected-business-build/producer.mjs`

No Zcode provider implementation was found in the current Tally source path. Therefore:
1. Keep Codex as the one selected provider for the current implementation.
2. Do not add a Zcode adapter merely to have two providers.
3. The latest #207 backend differential proves MXC can initialize the app-server and complete one real 21-second subscription-backed attempt, including writing the candidate note. However, MXC runs as the real user with the Administrators group; the production `proveConfinement` gate correctly rejects that identity. Do not weaken the least-privilege contract to accept MXC.
4. The elevated backend has the correct `codexsandboxonline` principal, no Administrators group, and all five denied-read checks pass, but app-server initialization currently returns `os error 2` because the machine's elevated sandbox setup is incomplete. One owner-authorized local action is required: `codex sandbox setup --elevated`. After that, run one bounded production-chain attempt; do not repeat proxy/auth/ACL experiments.
5. The driver15 investigation also found that setting `TALLY_CODEX_HOME` in the process environment does not reach `defaultAlmConfig`; the controller accepts an explicit `codexHome` argument. The out-of-repository driver correction is not a product fix. Pass the per-attempt scratch home explicitly through the existing #207 write set (`web/v2/journey-controller.mjs`) and add a regression test before the final attempt.
6. The full Host→Sandbox→Codex production chain remains NOT_PROVEN until the elevated setup and explicit `codexHome` wiring both pass. If the bounded elevated route fails, do not silently switch to MXC or weaken identity rules; substitute Zcode only if an already-existing qualified Zcode path is available. Do not build a second provider adapter for v2.

## 6. Historical unsuccessful dispositions — #331 / #211 / #212, never PASS

The #331 historical recovery Task has two non-retroactive P1 findings: no globally atomic shared Host fence at the time of the one-shot effect, and no durable independent exact-source review witness before that effect. The independent historical review remains BLOCK.

The actual #203 terminal event and CAS rev4 `TERMINAL_VERIFIED` have already been independently verified read-only. #383 is closed with its physical shared Host-fence PASS. Therefore #331 no longer represents a missing current #203 terminality precondition; it represents an irreversible historical policy deviation.

The same truthful unsuccessful disposition is applied to #211/#212: their source PRs #215/#217 are already merged and exact-head CI evidence is preserved, but the historical Task FSW/Outcome evidence cannot be retroactively reconstructed. They are closed not_planned, not PASS. The fresh successor #377 owns the Request-input POST retry defect with its own exact two-path repair and fresh verification.

Per the explicit Owner instruction:
- #331, #211 and #212 are recorded as CLOSED_NOT_PLANNED / unsuccessful, not successful Task completion.
- `TASK_OUTCOME_ACCEPTANCE=NOT_ACCEPTED`, `TASK_PASS=NO`, `RELEASE_PASS=NO`; #331's two historical P1 findings and independent BLOCK verdict remain; #211/#212 source and CI evidence remain without a retroactive Task PASS.
- #284 may consume only #203 terminality and #383 physical-fence evidence, never #331 Task PASS. #377 may consume only the fresh successor scope and its own verification, never #211/#212 historical Claims/FSWs.
- Never replay #203, write its CAS again, rewrite its event, backdate review, seed a fake FSW or fabricate a successful closeout.
- The exception is Tally-project-specific and outside the current ADS v4.18 disposition semantics. Feedback is recorded at [ADS #1360 comment 6801481987](https://github.com/kaicreator-mm/ai-development-standard/issues/1360#issuecomment-6801481987); Tally execution does not wait for a future ADS release.

## 7. Current Task DAG refresh — no unnecessary dependency recreation

The historical R21.1 PackBlob `48298a5861cf7ad65c3bda915530d7be68dd1aa6` was observed at 2026-11-09 and is `CANDIDATE_NOT_FROZEN`. It contains 51 Task identities and 37 edges.

Fresh live GitHub API readback on 2026-11-12:
- 51 `type:task` Issues total.
- 29 closed / 22 open after #331/#211/#212 were closed `not_planned`.
- Of the 29 closed, 26 are other historical closed Issue states (their Task Outcome acceptance is not uniformly verified). #331/#211/#212 are unsuccessful `CLOSED_NOT_PLANNED` and must never count as Task PASS.
- 13/51 entries had state/updatedAt drift from the R21.1 snapshot, including five legitimate open→closed transitions before #331.
- All 37 candidate edges were checked against live GitHub `blocked_by` endpoints and matched. The R21.1 notes claiming only four native edges and thirty planned edges are stale.

Required successor Pack:
1. Preserve all 51 Task identities and all 37 existing native edges; do not recreate dependencies already present.
2. Re-read every live Task's state, updatedAt, labels, source refs and relevant write-set/Claim ownership.
3. Recompute and document exact current dependency list; correct the stale 4-native/30-planned narrative.
4. Add one explicit disposition per Task: `CORE_V2`, `DEFERRED_V2_1`, `CLOSED_HISTORICAL`, or `CLOSED_NOT_PLANNED`.
5. For mixed Tasks (#209/#219/#262/#276), retain only the minimal Core acceptance in v2 and move advanced learning/analytics sub-outcomes to v2.1 without deleting their source history.
6. Produce one fresh PackBlob from the live snapshot; independently review its exact blob, current native dependencies and scope partition; then publish the exact Owner-bound Freeze record and verify Host readback. R21.1 remains immutable history.

## 8. Task disposition guidance

### Core v2
- #196 / #207: integrated J04–J06 workspace and the one real Builder browser/Host journey.
- #200: minimum independent Reviewer Claim/Join identity for the single Builder path.
- #197 / #198: separate Builder A/B candidate Tasks deferred to v2.1; #199 advanced Reviewer re-Join deferred to v2.1.
- #202 / #203: dynamic binding correction and exact reviewed candidate publication.
- #207 / #208: real browser/Host evidence and exact-SHA Release Qualification.
- #209: minimal defect intake and next-version planning; defer measured-learning replay.
- #210 / #260 / #284 / #312: guided Product/L2/DAG, current Task Pack, selected K→Host freeze and Product ReviewWitness.
- #211 / #212 / #377: native read recovery and the Request-input POST retry safety defect.
- #219 / #262: minimal maintenance and source/audit readback; defer measured experience feedback.
- #261 / #263 / #264: one-Agent UI actions, minimal actionable ALM and final installed integration.
- #273 / #276: bounded replan and minimum true Domain Package Engineering B workflows.
- #331: CLOSED_NOT_PLANNED historical deviation, excluded from success.
- Reuse completed #274/#275/#319/#370/#383 evidence; do not rerun or reimplement their completed scope.

### Deferred v2.1
- #197 / #198: separate Builder A/B candidate experiments; the v2 single real Builder journey is proven through #196/#207.
- #199: advanced Reviewer re-Join/parallel coordination.
- Any remaining positive two-Builder concurrency / multi-Agent Join acceptance.
- Advanced measured Task Learning, feedback efficacy and automatic promotion.
- Additional Agent providers, multi-provider registry and advanced delegation.
- Advanced telemetry/analytics and cross-project portfolio scope.

This table is a proposed scope partition for the successor Pack; exact live write-set/Claim reconciliation remains part of the fresh DAG review.

## 9. Exact current implementation candidates

- A / #260: PR #389 HEAD `af433e9b5ada60f6ef1918df067323f9ef434655`; independent source review is reusable at the same exact tree; Woodpecker #555 is 5/5 SUCCESS.
- E0 / #284: PR #390 HEAD `32b5808673052b5117644e120479fad96a582800`; Woodpecker #610 is 8/8 SUCCESS. A fresh independent combined-tree Exact-SHA review remains required.
- PR #390 includes one `.woodpecker/v2-alm-workspace.yaml` path owned by E1/#264, outside #284's original 14-path FSW. Resolve by a one-time explicit E1 handoff or Owner-recorded scope exception before merge; do not silently relabel ownership.

## 10. Completion gates

1. Independent review of this exact additive Product/L2 scope amendment; integrate the approved amendment without changing original frozen sources.
2. Refresh and freeze the successor Task Pack from the live 51-task roster and 37 verified native edges with explicit Core/Deferred dispositions.
3. Read back the #284 root Claim through the selected Host route after #331's unsuccessful disposition and verify only #203/#383 predecessor facts are consumed.
4. Complete the exact combined #390 review and E1 CI-path ownership resolution; integrate #389/#390 in the correct serial order and requalify changed SHAs.
5. Complete one real Codex Builder Task through independent Review, Repair as needed, tests/CI and GitHub exact-source readback.
6. Complete minimal defect/QA/Release UI and one authorized exact-SHA release candidate qualification.
7. Report Core MVP completion separately from deferred v2.1 scope. Do not use issue/PR/Claim/Dispatch counts as product completion.
8. No main merge, tag, production deployment or Release PASS until all applicable Core MVP gates are proven.

---

**Owner decision:** proceed with the single-Agent Core MVP scope above. The original Product/L2 freezes and all historical evidence remain immutable; v2.1 receives the deferred scenarios.


### Existing Task IDs moved out of the v2 critical path

- #197 and #198 are the separate Builder A/B candidate lanes from the earlier two-Builder experiment. Neither is required to prove one real Builder through the integrated #196/#207 path; both are deferred.
- #199 is the advanced Reviewer re-Join/repair-rejoin lane. V2 retains the minimal independent Reviewer identity through #200 and the one review/repair loop in #196/#207; repeated Join/re-Join orchestration is deferred.
- #198/#197/#199 remain visible in the historical DAG and are not silently deleted or counted as successful Tasks.


### Current Codex verification boundary

The latest [#207 exact differential receipt](https://github.com/kaicreator-mm/tally/issues/207#issuecomment-6801090377) supersedes the earlier assumption that elevated app-server startup had already passed. It records: MXC physical execution works but fails the production identity contract; elevated identity/denied-read checks pass but machine setup is incomplete; and the driver must pass `codexHome` explicitly rather than relying on an ignored environment variable. These are the only remaining bounded Codex integration actions. The actual model call and MXC attempt are not Installed/J06 PASS.


### Fresh P1 successor implementation

[PR #394](https://github.com/kaicreator-mm/tally/pull/394) contains the exact two-path #377 repair candidate at HEAD `963295ef3c4c18b275a559046d6771f0681dd733`. It fixes Request-input method detection and adds POST/PATCH single-attempt, GET/HEAD retry and explicit method-override tests. Woodpecker pipeline #618 was pending at the latest readback; no PASS is claimed until exact-SHA statuses complete.
