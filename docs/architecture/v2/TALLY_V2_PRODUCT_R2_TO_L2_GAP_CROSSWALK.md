# Tally v2 Product R2 → semantic L2 delta crosswalk (provisional, no physical ABI)

**Source identity:** Product [PR #77](https://github.com/kaicreator-mm/tally/pull/77), latest inspected author HEAD `62a5c3793e052bbeacb46870888964657893123b` as of 2026-10-24; Product PRD blob after R2 refinements `db69a996149020e71a0ab30b66f302f2f32d9597` (recheck current SHA before L2 review). Exactly 17 Product P0, 47 legacy functional domains and 67 Product acceptance specifications, **all physical tests NOT_RUN**.

**Review and version status:** Actual Claude R2 repeated-model delta [#6382926918](https://github.com/kaicreator-mm/tally/pull/77#issuecomment-6382926918) returned `P0=0/P1=0/P2=4 CONDITIONAL`, but **does not satisfy the distinct formal independent Product reviewer requirement**. Author has fixed four P2 concerns as source candidate only. [ADS v4.17 adoption PR #81](https://github.com/kaicreator-mm/tally/pull/81) has canonical root `repository/version/revision` and A1 adoption level on a **separate draft staging branch**, but **not merged or effective**. Pinned ADS `standards/PROJECT_ADOPTION.md` §2.1C(2) literal `RQ READY` conflicts with integrated v4.17 `CONDITIONAL`; source authority question [ADS #1318](https://github.com/kaicreator-mm/ai-development-standard/issues/1318) under [Tally #80](https://github.com/kaicreator-mm/tally/issues/80). No Product/L2 Freeze or implementation Task admission.

## Exact semantic allocation — carry forward only after Product reviewer disposition

| Product contract/gate | B/S/K/C/Host duty | Real authoritative subject | L2 mandatory negative evidence | Physical binding state |
|---|---|---|---|---|
| P0-01 Package-first | Engineering B packaged as real selected DH Domain Package(s), selected S/one K/C/Host public installed BUILD | Qualified DH public Package Manifest/ABI/BUILD revision | Any standalone TS business state engine or private DH src import ⇒ BLOCK | PENDING_DH_0_10_2 |
| P0-03/04 Freeze truth | B proposes draft/review; K stores only app intent; C GitHub publishes and reads **actual applicable ADS review/freeze evidence** | GitHub commit/Issue/Review/accepted source & adoption | K State/SQLite showing FROZEN without current qualified GitHub artifact ⇒ UNKNOWN/BLOCK | PENDING_STANDARD_PIN_AND_PRODUCT_R1 |
| P0-05 L3 execution admission | B enforces TaskContract Tests/Contract/Implementation plan/Failure Handling/Reference; K admits effect; C GitHub readback | Current GitHub Issue/L3 source SHA, selected ADS dialect | Ready DAG but stale/incomplete L3 ⇒ L3_NOT_ADMITTED | SOURCE_SPEC_ONLY |
| P0-06/F17 real Agent | Mandatory real installed/authenticated Codex CLI C Local; B selects role/provider; K owns start/cancel/terminal/replay admission, Host owns process receipts | Actual Agent exit/source/workspace/receipt | Fake/echo-only Agent cannot satisfy CJ03; accepted not verified terminal | PENDING_PUBLIC_HOST_C |
| P0-16/F21/F29 workspace safety | C Local creates per-Attempt clean exact-base disposable Git worktree; K authorizes non-overlapping write scope; freeze resulting candidate commit/tree before verifier | Real Git worktree root/source SHA/tree/attempt | Dirty base, wrong parent, two writers same checkout, stale candidate ⇒ BLOCK | SOURCE_SPEC_ONLY |
| P0-15/P0-16 GitHub write security | **Choose Model A**: Builder OS process has no GitHub write credential/agent/key/GCM token; B issues scoped publish intent; K permits; C GitHub sole native write issuer with immutable receipt/readback | Native GitHub fact and real K Journal/Effect receipt | Builder direct git push/gh merge/force/alternate store/ssh-agent/sudo-admin ⇒ UNGOVERNED_EFFECT; not Task PASS | PENDING_REAL_OS_DENY |
| P0-17/F24 Hidden isolation | Local default separate verifier OS principal/UID/Windows account with ACL-restricted fixtures, staged only after candidate freeze; B enforces assurance, K/C Verifier executes | Actual OS ACL plus **Builder deny-read witness** with escalation paths checked | Same OS user readable verifier fixture, sudo NOPASSWD/backup privilege escape ⇒ HIDDEN_NOT_QUALIFIED; missing verifier ⇒ HIDDEN_VERIFIER_UNAVAILABLE | PENDING_REAL_OS_DENY |
| P0-10/F39 task-meta failure isolation | task-meta observational Publisher nonblocking; bounded timeout and disable failing in-process hooks; B projects UNKNOWN metrics, K/Task/Verifier truth unchanged | Real Agent result; independently published GitHub ADS task-meta report when available | Hook exit nonzero/hung/publisher unavailable ⇒ same actual Task outcome, no awaited telemetry | SOURCE_SPEC_ONLY |
| P0-11 Release | B composes Full/Hidden/CJ/RQ proposal; native GitHub Reviewer/CI/commit/tree/tag readback and K/C effects only if eligible | Exact qualified source/tree and actual main merged tree/tag target | stale main/changed tree/Hidden not qualified ⇒ BLOCK/re-validate/RQ | PENDING_PRODUCT_AND_PHYSICAL |
| P0-08/09 GitHub cache | C GitHub is source reader; SQLite indexes source-qualified complete/current revisions, B business projection; no cached authority bypass | Native GitHub and ADS Report edit/source identity | 403/pagination incomplete/correction invalidates derived advice; Task telemetry outage not engineering failure | SOURCE_SPEC_ONLY |
| P0-13 UI | UI shell only performs form/display/command projection of actual B/K, never business decision state; support no raw JSON forced | B/K source-qualified actual current facts | Shell itself making Task acceptance or bypassing K/C ⇒ BLOCK | SOURCE_SPEC_ONLY |

## Product vector groups driving L2 fixtures

- **Lifecycle**: V01/V02/V31/V32/V46/V47/V50/V51/V65 — source/Freeze currentness, adopted exact standard or conditional status not FROZEN.
- **DAG/L3/authority**: V03–V06/V33/V34/V44/V45/V48/V49 — selected ADS dialect and pre-admission evidence vs post-run Implementation receipts.
- **Local/Agent/worktree/Effect**: V07–V12/V26/V30/V52–V56/V61–V63/V66 — Codex real provider, selected K, actual Host receipts/unknown, no direct Builder GitHub writes, single active Task attempt.
- **Hidden/security**: V13/V28/V35–V41/V57/V58/V67 — held-out verifier ACL principal, privilege escalation and credential-store negative evidence, repo grants/origin/CSRF.
- **Task telemetry and GitHub reader**: V18–V22/V27/V43/V59/V60 — collector/hook fail-open, current report correction/unsupported schema/derived advice invalidation.
- **Release and App shell**: V23–V25/V29/V42/V64 — physical Domain Package import closure and independently qualified Version Tag; UI shell never second B/K state machine.

## Strict dependency and source authority

1. Independent Product Reviewer **distinct from prior reviewer** must deliver an exact-HEAD PASS/CONDITIONAL verdict with P0/P1/P2 and legal Product Freeze conditions; prior repeated Claude verdict is preliminary review evidence, not formal completion.
2. ADS #1318 authoritative source interpretation or successor RQ must resolve untagged manual adoption READY-vs-CONDITIONAL; until resolved, #81 stays Draft. A1 override declaration is **not** a gate waiver.
3. Refresh/rebase this L2 branch onto the **actual accepted** Product/standards baseline before any independent semantic L2 R1. Since PR #79 was branched from a much older Product head, its file/source references are not proof of integration.
4. Public DomainHarness v0.10.2 real packed Consumer/Producer and Node24 BUILD/Host (T09 #1330 / R01 #1331) required before physical ABI Freeze or production TS Package integration.
5. Real `BUILDER_CONFINEMENT_UNAVAILABLE` vs `HIDDEN_VERIFIER_UNAVAILABLE` separately projected; only evidence-backed denial and requisite platform permissions can qualify selected Local journey; no Docker/runx requirement imposed.

**Disposition:** `L2_CROSSWALK_SOURCE_PROPOSED`, `SEMANTIC_L2_REVIEW=NOT_RUN`, `PHYSICAL_L2_FREEZE=NO`, `ADS_V2_EFFECTIVE_PIN_PENDING`, `TALLY_RUNTIME_TESTS=NOT_RUN`, `MERGE_ELIGIBLE=NO`.
