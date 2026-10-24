# Tally v2 ADS v4.17.0 exact-source adoption

**State:** PROJECT_V2_PIN_UPDATED_ON_VERSION_STAGING_BRANCH / INDEPENDENT_ADOPTION_REVIEW_PENDING / PRODUCT_FREEZE_NOT_GRANTED / MAIN_UNCHANGED.
**Issue:** [Tally #80](https://github.com/kaicreator-mm/tally/issues/80). **One concern:** v2 standards adoption only; no Product/L2 freeze or production runtime claim.

## New effective v2 standard source

| Field | Exact value |
|---|---|
| Version pin | 4.17.0 |
| Source repository | https://github.com/kaicreator-mm/ai-development-standard |
| Immutable source | `a45644d734fdf94da3e6f9c3f90ffe4ce3c25fda` (actual ADS v4.17 merged main) |
| Qualified Git tree | `d1059bef9d5f74ad607c68c0a08d5fb9836c6fa5` |
| Upstream frozen candidate | `a4e38259945118b868eff5aebd5e0a045f696f2c` (content tree equivalent) |
| Project scope | `tally-v2` only |
| Metadata changed | `.dev-standard/VERSION`: version/commit/adopted_for; retains `source` URI |
| Evidence | [ADS #1311 independent RQ](https://github.com/kaicreator-mm/ai-development-standard/issues/1311#issuecomment-6381396628); [final integration exact tree](https://github.com/kaicreator-mm/ai-development-standard/issues/1311#issuecomment-6381478181) |
| RQ | `CONDITIONAL` P0=0/P1=0/P2=3 for standards-source only; not a runtime or external integration PASS |

## Conditions carried into this adoption

1. Legacy CPython 3.14 pinned-dependency portability on an older ADS profile is an unclosed nonblocking P2; not rewritten as PASS.
2. v4.17 new source test / planning docs manifest coverage is a bounded P2; project implementation must still check applicable source/profile and tests.
3. **External runtime integration remains NOT_PROVEN:** task-meta → GitHub ADS Report → Tally GitHub-only reader/feedback, native runtime Claims/Release, Host effects and real human DENY. These must be independently exercised when a qualified DomainHarness Domain Package is available; **do not count ADS source RQ as Tally Release PASS**.

## Historical Tally v1.5.0 protection

The original `.dev-standard/VERSION` at Tally main `fab321845a0eb630d50eb5791f3de54b6870f5fb` was:

```yaml
version: 1.2.1
source: https://github.com/kaicreator-mm/ai-development-standard
commit: e836f7a4e1f1c7fcf4530a811af235c8c8dba8a1
adopted_for: tally-v1.5.0
```

This original immutable Git revision remains the truth for **historic v1.5.0 and v1.5.5 release evidence**. The new working-tree pin is prospective for `tally-v2`; it is **not** retroactive validation or a migration of Go v1.5 behavior. The old `AGENTS.md` and `.dev-standard/PROJECT_OVERRIDES.md` explicitly scope their Go/CI rules to v1.5.0, not future TypeScript Domain Package v2.

## Acceptance and execution limits

- The selected source is the actual exact upstream main revision at inspection, not a floating `main` or invented ADS release tag. The normal Tally version-stage reviewer must re-read source SHA/tree and independent upstream qualification before admitting this pin.
- A linked source-specific **Tally v2 Product Freeze** still needs an independent Product Review at exact Tally candidate head, legally applicable GitHub ADS artifact/current readback, plus independently accepted v2 adoption. Modifying this YAML alone is **not** that Freeze.
- `Task DAG`, L3 and production B/S/K/C/Host remain blocked by Tally Product/L2 gates and actual DH 0.10.2 public Package/BUILD/Host/external-producer proof. Local and runx integration tests are NOT_RUN.
- Keep the one-concern adoption PR **draft and unmerged** until independent review. Target an existing v2 source branch; do not merge an unfinished v2 into historical main.
- The product intent is complete minimum ADS lifecycle with **Native Domain Packages, local Codex, task-meta fail-open, real GitHub evidence**. ADS v4.17 selected Contract/report dialect must be exact and separately proven when a real C GitHub Reader exists.

**Validation scope this change:** GitHub source blob and exact SHA/tree checks only. No Python, Node, Go, CI, Hidden or external runtime tests were run for Tally. Status `ADOPTION_CHANGE=COMMITTED_TO_STAGING_BRANCH; INDEPENDENT_REVIEW=NOT_RUN; EFFECTIVE_MAIN_PIN=1.2.1; V2_RELEASE_PASS=NO`.