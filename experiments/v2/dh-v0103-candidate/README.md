# Tally v2 × DomainHarness v0.10.3 — candidate compatibility (NOT production upgrade)

**Status as checked 2026-10-28**: The DomainHarness **version branch** `version/v0.10.3` exists at exact `e86f2c9450b962812fc03e1494aaad402490258e`, but a GitHub `v0.10.3` tag / published Release does **not** yet exist. This revision is a planning / governance + documentation candidate. At this SHA, the public `packages/contracts`, `packages/microkernel`, `packages/host-node`, `tooling/build`, and selected basic Domain Package *implementation/manifest trees* are the same bytes as released `v0.10.2@cee14399d9d0af4e20e34b9111861b16a465b690`. Public Host/Contracts/Microkernel `package.json` still report `0.10.2-rc.0`. **This is not an installed v0.10.3 ABI** and proves nothing about unfinished 0.10.3 features.

## Pins and separate release gates

| Purpose | Source of truth | Qualified status |
| --- | --- | --- |
| Tally current version-stage runtime | `kaicreator-mm/domain-harness@cee14399d9d0af4e20e34b9111861b16a465b690` | Released v0.10.2, R01 CONDITIONAL; already used by current Tally BUILD/Host CI |
| Candidate compatibility probe only | `kaicreator-mm/domain-harness@e86f2c9450b962812fc03e1494aaad402490258e` | `version/v0.10.3` branch snapshot, **not published** |
| Actual future upgrade | TBD real immutable `v0.10.3` release tag, package manifest revision and qualified Release RQ | BLOCKED on upstream implementation/qualification |

The new `.github/workflows/v2-dh-v0103-candidate-compatibility.yml` is purposely **non-authoritative for Release**. Its checkout and source-tree equality guard prevent claiming the in-progress candidate is a new runtime. Node24 then runs the existing **real Tally public BUILD → selected K/B/Native GitHub C → SQLite** and **physical Host SQLite CAS / WAL** suites from the Tally checkout against the candidate exact SHA. Existing stable v0.10.2 jobs are unchanged and remain the rollback baseline. No new K, no proprietary Host API or fake public npm 0.10.3 version.

### Release adoption trigger / actual migration after upstream RQ

1. Confirm exact immutable `v0.10.3` tag and independent DomainHarness [version RQ #1419](https://github.com/kaicreator-mm/domain-harness/issues/1419) / [shared integration #1418](https://github.com/kaicreator-mm/domain-harness/issues/1418) / [Provider integrity #1413](https://github.com/kaicreator-mm/domain-harness/issues/1413) facts. Do not infer release from a version branch or an Issue CLOSED alone.
2. Replace candidate SHA in this isolated verification with the **actually published** tag target; verify package export versions, exact supported B/C/Host/BUILD APIs and lockfile/integrity. The present **ABI-equality assertion must be rewritten as an impact assessment** when actual v0.10.3 runtime bytes change (otherwise it correctly fails).
3. Rerun packed clean-consumer Node24 BUILD/selected Domain App, native GitHub C → K → SQLite restart/replay, **real selected Host CAS**, all existing Tally report/Claim negative suites and legitimate execution authority gates. Repeat the Tally tampering negatives (Provider swap; Provider + matching mutable embedded digest; stale trusted App/subjectDigest); exercise new opt-in verified-executable profile only if actually exported in the released SDK. Preserve Tally C output allowlist and deny untrusted Provider exceptions. Real positive Task CAS writer/accepted event/Kernel admission is separately governed in [Tally #125](https://github.com/kaicreator-mm/tally/issues/125), **not solved by a DomainHarness version bump**.
4. Independent exact-SHA source/compatibility review under ADS v4.18, then guarded `version/v2` integration; main remains HOLD pending Tally Product #77, L2 #82, ADS #85, RQ. Do not mass-replace immutable historical `0.10.2` references or v0.10.2 CI evidence with claims of newly tested 0.10.3.

References: [Tally v2 controller #86](https://github.com/kaicreator-mm/tally/issues/86), [v0.10.3 intake #1392](https://github.com/kaicreator-mm/domain-harness/issues/1392), [Provider integrity #1393](https://github.com/kaicreator-mm/domain-harness/issues/1393).

**Gate flags:** `UPSTREAM_V0103_RELEASED=NO` · `V0103_REAL_RUNTIME_ABI=NOT_AVAILABLE` · `CANDIDATE_V0102_COMPAT_TEST=CI_PENDING` · `TALLY_VERSION_UPGRADE=HOLD` · `MAIN_INTEGRATION_AUTHORIZED=NO`.
