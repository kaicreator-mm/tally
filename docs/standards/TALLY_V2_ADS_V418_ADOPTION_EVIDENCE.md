# Tally v2 prospective ADS v4.18.0 adoption — exact source and compatibility

**Status:** PROPOSED_SOURCE_PIN / LOCAL_PROJECT_ADOPTION_NOT_YET_REVIEWED / EFFECTIVE_MAIN_PIN_HISTORIC_1.2.1 / PRODUCT_FROZEN=NO / PHYSICAL_TALLY_RUNTIME_NOT_RUN.

## Immutable release subjects

| Field | Exact evidence |
|---|---|
| Proposed source | `kaicreator-mm/ai-development-standard` v4.18.0 integrated `main@2c395c55569fe2edf3f4135f12d3d069e526d0b7` |
| Frozen qualified candidate | `version/v4.18@ac7e7e51de3a7ac27343f16a96568c0b0f2c5349` |
| Both Git trees | `cdf032f629668008a940993713a2bfaa219f5653` |
| Independent standards-source RQ | [#1327 RQ READY](https://github.com/kaicreator-mm/ai-development-standard/issues/1327#issuecomment-6401840670), P0/P1/P2=0/0/0, scoped to standards Git-source |
| Main integration | [PR #1335](https://github.com/kaicreator-mm/ai-development-standard/pull/1335) merged via a guarded two-parent merge, identical qualified Git tree; [ADS #1322 source closeout](https://github.com/kaicreator-mm/ai-development-standard/issues/1322#issuecomment-6402117697) |
| Post-main CI | [planning Linux+Windows](https://github.com/kaicreator-mm/ai-development-standard/actions/runs/40225708568), [standard](https://github.com/kaicreator-mm/ai-development-standard/actions/runs/40225708591), [GitHub structured](https://github.com/kaicreator-mm/ai-development-standard/actions/runs/40225708561): 3/3 observed SUCCESS |
| Tag/GitHub public Release | NOT_CREATED — NOT REQUIRED for independently qualified **manual untagged source adoption**, if project-local predicates verified |
| Historical Tally main | `fab321845a0eb630d50eb5791f3de54b6870f5fb`; original ADS 1.2.1 `e836f7a4e1f1c7fcf4530a811af235c8c8dba8a1` bound to v1.5.0 |

## Correct v4 project-root carrier

Prospective `.dev-standard/VERSION` must contain exactly:

```text
repository=kaicreator-mm/ai-development-standard
version=4.18.0
revision=2c395c55569fe2edf3f4135f12d3d069e526d0b7
```

Project-root `.dev-standard/PROJECT_OVERRIDES.md` declares exactly one `v4.adoption_level: A1_MANUAL_PROTOCOL`, preserving manual GitHub/file fact obligations without inventing automation. Current root **structural verifier on this new PR HEAD has NOT_RUN**; run the **pinned v4.18** `scripts/verify_project_standard.py <tally-project-root> --standard-repo <ads-clone> --require-resolution` with real exact source/exit logs before candidate admission. The previous v4.17 PR #81 Python3.14 structural PASS was for a **different exact source**, not reusable as a new 4.18 verification PASS.

## Material project compatibility and safe activation

| Dimension | Project-specific assessment / remaining obligation |
|---|---|
| Source/protocol | Tally v2 native DomainHarness B/S/one K/C/Host design, never standalone Go/TS shim. ADS v4.18 adds Task decomposition/scaffold/parallel workfront source guidance; no new automatic live scheduler or extra Kernel source authority. **Design CHECKED; runtime NOT_RUN** |
| Task DAG/Scaffold | Separate readiness/topology/fit and native GitHub Issue Dependencies/accepted Claim; don't substitute document edge for live dependency. v2 DAG/L3 not executable admitted yet. **Review pending** |
| Actor/authorization | v4.18 preserves declared execution roles, independent Review, real external GitHub Effect permission and denial. Actual Host/Repo identities not tested. **SOURCE_CHECKED / PHYSICAL_NOT_RUN** |
| Validation/CI | v1.5 historical no-CI override only scoped to v1.5; v2 needs version-applicable Node24/SQLite/Host/Local/Hidden/CJ/Release evidence. Existing Go check failures are inherited, not PASS. **CI profile #83 pending** |
| Task Report dialect / task-meta | v2 C GitHub-only source reader; no task-meta private API/DB/traces. Actual compatible ADS GitHub report dialect and live task-meta→GitHub→Tally feedback **NOT_RUN**, must qualify before claiming that vertical slice |
| Kernel/SQLite/Host | v2 default SQLite-first #84 with separate K Host CAS/Journal/Effect and Tally GitHub-derived projection; Postgres optional. DomainHarness 0.10.2 T09 #1330 version-merged, R01 #1331 source Release still open; no published platform Runtime Release PASS |
| In-flight Claims | Historical v1.5/v1.5.5 source subjects and accepted Claims remain pinned to their old effective standard. Before v2 activation, current native Issue/Claim/DAG/frozen Product/PR evidence must be independently inspected; **current-work currentness NOT_RUN** |

### Previous candidate and normative issue scope

Old Tally [Draft PR #81](https://github.com/kaicreator-mm/tally/pull/81) proposed ADS v4.17.0 on source RQ CONDITIONAL P2×3, while [ADS §2.1C(2)](https://github.com/kaicreator-mm/ai-development-standard/blob/2c395c55569fe2edf3f4135f12d3d069e526d0b7/standards/PROJECT_ADOPTION.md) literally requires RQ READY for untagged manual adoption. [ADS #1318](https://github.com/kaicreator-mm/ai-development-standard/issues/1318) may remain open as a historical normative issue; **the proposed v4.18 has an actual distinct READY RQ**, avoiding the specific old contradiction without a retrospective waiver. This is not an automatic endorsement of Tally's project compatibility or a retroactive claim that v4.17 was READY.

## Required independent review and activation result

1. Verify immutable candidate/source/main SHA/tree and RQ READY independently, confirm exact root standard `VERSION`, one A1 adoption level, previous effective pin and current no-Tally-native-Task Claim state.
2. Execute pinned v4.18 Python project-root verifier and `--require-resolution` at the exact proposed project HEAD; preserve literal command/environment/EXIT.
3. Independent Tally Product standards adoption Review P0/P1/P2 plus all applicable user/runtime/CI/permissions/currentness, then legally merge to **v2 version-stage branch only**, not historical `main`, with remote readback.
4. Actual GitHub Product Freeze still requires independent Product Review and source-qualified Freeze receipt. DomainHarness 0.10.2 release and Tally physical conformance remain independently pending.

**No local tests executed while drafting this GitHub-only source artifact.** `PIN_SOURCE_CANDIDATE=YES; V2_ADOPTED=NO; PREVIOUS_TASK_PINS_UNCHANGED; MERGE_ELIGIBLE=NO; NO_MAIN_MERGE`.
