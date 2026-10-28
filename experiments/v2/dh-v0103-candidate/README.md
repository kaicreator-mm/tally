# Tally v2 × DomainHarness v0.10.3 — exact tagged-source qualification

## Source and package identity (2026-10-29)

- Upstream Git tag [`v0.10.3`](https://github.com/kaicreator-mm/domain-harness/tree/v0.10.3) **exists** and points to exact **`061376f4a023070f215f9feffad5c24d96d503cb`**, which also equals upstream `main` at source verification. This closes the previous missing-tag gate; current upstream RQ [#1419](https://github.com/kaicreator-mm/domain-harness/issues/1419) is closed. Tag identity is about **source**, not a GitHub Release object or npm package version.
- Public Contracts/Microkernel/Host Node `package.json` still declare **`0.10.2-rc.0`**, despite the new v0.10.3 source tag. Use **one immutable Git checkout** and `npm pack` from its real source/exports; do not install an invented `@kaicreator/*@0.10.3` package, mix versions or silently rewrite manifests. Node engine `>=24.0.0`; this fixed Node24 verification job is only one tested tuple.
- `version/v2` baseline is `0cbe2a55cf2ebf8472c82fbae0de31e7b6e165f6` at this task intake. Historical v0.10.2 smoke evidence remains valid *for its original tested SHA only*.

## Actual ABI impact and scoped tests

- Compared released v0.10.2 `cee14399d9d0af4e20e34b9111861b16a465b690` vs tagged v0.10.3: 22 changed public SDK source/dist/manifest paths, principally new Contracts neutral LLM generate types and Host Provider integrity/client/deployment; **Microkernel, selected Kernel/Standard Domain Packages, public BUILD, SQLite CAS core** remain unchanged. This is **not** the old candidate runtime-equality claim.
- The existing GitHub Actions workflow consumes the exact tagged SDK source with two independent Node24 profiles: selected native GitHub C → B/BUILD → K → SQLite, and real public Host SQLite CAS/WAL. It source-checks the unchanged K/BUILD/SQLite seams; verifies the new LLM/provider files exist and runs the upstream v0.10.3 contract and verified-provider-bundle negative tests. One temporary script copy replaces only the already-existing old SHA guard. Existing v0.10.2 script/source/receipts are not changed.
- Local Windows evidence from [Tally #125](https://github.com/kaicreator-mm/tally/issues/125#issuecomment-6477907842) independently exercised earlier v0.10.3 candidate (`21aba1aa…`), real Host CAS PASS, WAL 15×4 PASS×2, read-only canonical sandbox Task PASS; **not a run on the tagged SHA**, not proof of native accepted Claim or K/Agent effect.
- Ubuntu/WSL optional simultaneous first-WAL-open still has a real SQLITE_BUSY negative in [DomainHarness #1452](https://github.com/kaicreator-mm/domain-harness/issues/1452); does not undo the one-single-writer CAS PASS, and cannot be relabeled fixed by different Windows run.

## Gates

1. Real current SHA GitHub Actions check for this PR must finish SUCCESS, or an equivalent independently qualified installed-package CI/Host run must be recorded with exact input SHA and literal EXIT. Do not fabricate log counts from provider statuses.
2. Independent ADS v4.18 #1241 exact-head R1 review: exact source tag, scope of changed SDK APIs, selected K/SQLite ABI stability, correct source-only provenance, legitimate Provider-integrity denial coverage, no unauthorized GitHub/Agent writes. Merge **version/v2 only** after PASS and applicable CI. A successful SDK tag or this suite does not authorize a real GitHub Task Claim or Tally main release.
3. Real Owner-controlled Host source admission + actual ADS native Claim + selected Kernel Agent Task permission remain separate Tally [#125](https://github.com/kaicreator-mm/tally/issues/125) / [#62](https://github.com/kaicreator-mm/tally/issues/62) P0 work, not granted by a package pin.

```ini
DOMAINHARNESS_V0103_TAG=061376f4a023070f215f9feffad5c24d96d503cb
V0103_GIT_SOURCE_TAG=YES
PUBLIC_NPM_PACKAGE_SEMVER_0103=NOT_DECLARED
RELEASED_SDK_SOURCE_ABI_IMPACT=CHANGED_CONTRACTS_AND_HOST_PROVIDER
SELECTED_K_SQLITE_BUILD_SEAMS=SOURCE_UNCHANGED
TALLY_VERSION_UPGRADE=PR_CI_AND_INDEPENDENT_REVIEW_REQUIRED
REAL_ACCEPTED_TASK_CLAIM=NOT_RUN
MAIN=HOLD
```
