# Tally v2 — ADS v4.17.0 version migration and qualification ledger

**Status:** STAGED_PIN_REPAIRED / STANDARD_CONFORMANCE_REVIEW_PENDING / RQ_READY_CONFLICT_OPEN / PR #81 DRAFT_UNMERGED / EFFECTIVE_TALLY_MAIN_PIN_STILL_LEGACY / PRODUCT_FROZEN=NO. These facts are not runtime qualification.

## Exact old vs proposed identities

| Scope | Identity | Truth state |
| --- | --- | --- |
| Tally historical v1.5.0 main | ADS 1.2.1 `e836f7a4e1f1c7fcf4530a811af235c8c8dba8a1`, YAML old syntax, project `main@fab321845a0eb630d50eb5791f3de54b6870f5fb` | HISTORICAL; no retrospective mutation |
| Proposed v2 root pin (this PR) | `repository=kaicreator-mm/ai-development-standard`, `version=4.17.0`, `revision=a45644d734fdf94da3e6f9c3f90ffe4ce3c25fda` | STAGED / unmerged, not actual main |
| ADS frozen source candidate | `version/v4.17@a4e38259945118b868eff5aebd5e0a045f696f2c`, tree `d1059bef9d5f74ad607c68c0a08d5fb9836c6fa5` | FROZEN / independently conditionally qualified |
| ADS integrated source | `main@a45644d734fdf94da3e6f9c3f90ffe4ce3c25fda`, tree `d1059bef9d5f74ad607c68c0a08d5fb9836c6fa5`, VERSION=4.17.0 | INTEGRATED / tree matches qualified source |
| Tally version-update checkpoint | [Issue #80](https://github.com/kaicreator-mm/tally/issues/80) and [Draft PR #81](https://github.com/kaicreator-mm/tally/pull/81) | REVIEW_REQUIRED / not merged; exact HEAD must be refreshed |

References: [independent ADS v4.17 RQ](https://github.com/kaicreator-mm/ai-development-standard/issues/1311#issuecomment-6381396628) (`CONDITIONAL`, P0/P1/P2=0/0/3), [exact main integration](https://github.com/kaicreator-mm/ai-development-standard/issues/1311#issuecomment-6381478181). Git tag/public Release optional, not present; no invented Tag.

## Normative format and adoption profile remediation

At the pinned ADS revision, `standards/PROJECT_ADOPTION.md` §2 requires `.dev-standard/VERSION` canonical `repository=...`, `version=...`, `revision=<immutable SHA>`; historical v1.5 YAML is invalid as a v4 pin. [PR #81 old-head reviewer BLOCK](https://github.com/kaicreator-mm/tally/pull/81#issuecomment-6382939716) found P1-01 for YAML and P1-02 for missing level.

This candidate now uses only the three canonical keys (no `adopted_for` inside VERSION) and records `tally-v2` scope here and `.dev-standard/PROJECT_OVERRIDES.md`. **Exactly one operative `v4.adoption_level: A1_MANUAL_PROTOCOL`** is present in project root overrides, outside a fenced code block. A1 honestly describes durable GitHub/file manual source protocols before Tally implements real v4 machine reducers/controllers. A1 changes implementation surface, **not required gate/assurance/validation predicates**. Future A2–A4 requires actual tested capability and a new reviewed adoption.

These edits are made in the draft source branch; previous reviewer tested and saw FAIL at old HEAD. **No current-HEAD pinned Python verifier has been executed in this conversation.** Thus current parser/immutable-resolution results remain NOT_RUN until a verifier with real project Git root and exact ADS Git repository runs `python -I <ads-source>/scripts/verify_project_standard.py <tally-project-root> --standard-repo <ads-source> --require-resolution`. Do not promote the author's source edits into an executed PASS. Reviewer should record actual exit code/stdout/environment.

## Standards-source RQ constraint — must resolve before actual adoption

**P1 external admission ambiguity:** ADS `standards/PROJECT_ADOPTION.md` §2.1C(2) expressly requires a distinct Release Qualification verdict **READY** for untagged-manual adoption. The same pinned source's `standards/RELEASE_STANDARD.md` §6 allows **CONDITIONAL** for source Release Qualification when mandatory gates pass and P2 are explicitly accepted. Actual v4.17 independent RQ is `CONDITIONAL`, not literal `READY`. **The author cannot treat CONDITIONAL as READY or create a waiver.** This discrepancy is filed at [ADS prospective normative Issue #1318](https://github.com/kaicreator-mm/ai-development-standard/issues/1318). Pending an independently authorized binding interpretation or successor RQ satisfying the adoption predicate, `V2_ADOPTION_ELIGIBLE=NO / ADOPTION_NOT_EFFECTIVE`; keep this pin PR Draft/unmerged even if canonical file parser passes. [#80](https://github.com/kaicreator-mm/tally/issues/80) owns the explicit disposition; ADS standards owner may need to publish a source-bound clarification or qualified successor receipt.

The v4.17 residual limitations must stay visible: P2-01 legacy cp314 pinned dependency portability failed on a historical profile, P2-02 source manifest coverage of newly added docs/tests, P2-03 real task-meta→GitHub→Tally reader/Host-native Claim/Release/human DENY E2E **NOT_PROVEN**. For Tally v2 source-only Product review, these do not assert the runtime has run; for Tally v2 release they remain required physical applicability checks. This ledger does not independently accept a normative exception.

## Compatibility matrix for prospective migration (not guessed green)

| Concern | Current evidence and decision | Status before adoption |
| --- | --- | --- |
| Identity/source | Exact upstream integrated 4.17 SHA/tree/source evidence; canonical pin changed in PR; actual project ROOT verifier not yet re-run | SOURCE_CONFIRMED; LOCAL_VERIFIER_NOT_RUN |
| v1.5.0 vs v2 authority | Old YAML and no-CI restriction are explicitly v1.5.0 only; prospective v2 uses DomainHarness B/S/K/C/Host and real GitHub facts | SCOPED_DESIGN; RUNTIME_NOT_RUN |
| Adoption level | Root project A1_MANUAL_PROTOCOL; no A2+ reducers or orchestration asserted | DECLARED_SOURCE / RUNTIME_UNPROVEN |
| CI/Validation | v1.5.0 remains explicit Local/Build Host no CI; v2 CI profile not frozen yet, may accept scoped waiver under normative policy only | V2_CI_PROFILE_PENDING; BASELINE_FAILURES_NOT_WAIVED |
| Review/Freeze | Product source candidate PR #77 not independently Frozen; B/K draft cannot replace actual GitHub ADS Freeze receipt | NOT_FROZEN |
| Report protocol | GitHub-only Task report reader, strictly versioned v4/v5 dialect planned; task-meta collector hook failure never blocks actual Task | DESIGN_ONLY; CONFORMANCE_NOT_RUN |
| Security/permissions/external effects | Builder lacks GitHub write credential by Product design; K-governed GitHub publisher, OS-account Hidden isolation require actual denial witnesses | REAL_DENY_NOT_RUN |
| Host/Domain Package | DomainHarness v0.10.2 external producer/BUILD/Node24 K/Host selected package proof required | BLOCKED_PENDING_DH #1330/#1331 |
| In-flight Claims/Tasks | Historic accepted subject immutable; v2 source has no qualified implementation Task claims evidenced by this adoption PR | NO_REPIN; LIVE_TASK_AUDIT_PENDING |
| Rollback | Do not overwrite legacy historical SHA/status; reversing new v2 effective pin requires separately authorized source-bound PR and affected-gate reconciliation | SPEC_DEFINED; NOT_EXECUTED |

## Readback, release boundaries, next independent tasks

After the last change, re-read exact PR HEAD and all three touched standards/migration files. Do not merge PR #81 into version branch until an independent reviewer provides P0/P1/P2 with source compatibility and **the adoption READY/CONDITIONAL conflict has an authorized disposition**; no main merge while Tally v2 is incomplete. If the Product branch advanced while PR #81 was authored, ensure the stacked diff is applied to its latest head without overwriting Product fixes.

Physical tests not performed here. No v2 Product Freeze, L2 Freeze, Task DAG admission, actual K/C Effect/Hidden E2E, Full/Hidden/Critical Journey, Release RQ or Tag can be asserted merely from this standards-only source patch.

**Canonical adoption status:** `V2_ADS_VERSION_FILE=4.17.0_STAGED_CANONICAL; V4_ADOPTION_LEVEL=A1_MANUAL_PROTOCOL_STAGED; FORMAL_INDEPENDENT_REVIEW=PENDING; RQ_LITERAL_READY_CONFLICT=OPEN; V2_EFFECTIVE_VERSION_UNCHANGED; TESTS=NOT_RUN`.
