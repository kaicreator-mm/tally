# Tally v2 — Engineering Business v08 ABI policy candidate
Parent [#105](https://github.com/kaicreator-mm/tally/issues/105), bounded [#108](https://github.com/kaicreator-mm/tally/issues/108).

A genuine `getPolicy()` Business module using the **released DomainHarness v0.10.2 v08 Business ABI**, derived from public authored example at tag `cee14399d9d0af4e20e34b9111861b16a465b690`. It owns *only* a Task-readiness **advisory** intent and a pure effect handler. **Do not treat it as verified BUILD-selected yet**. No Go bridge, private DomainHarness imports, duplicate Kernel, direct GitHub writes, tasks admitted, or source-of-truth SQLite.

The initial `assessTaskEvidence` merely checks source field forms and a supplied dependency indication; even positive output is `OBSERVED_INPUT_ONLY`, with `admitted=false,taskPass=false`. Actual native GitHub dependency/Claim reading and permission belong to C GitHub, and single authoritative Admission/Effect to selected K. It cannot be used to start an agent, merge PR or qualify Release.

Node24 scoped unit tests: `node --test test/business-policy.test.mjs`. These prove policy shape and negative semantics **only**, not real DomainHarness BUILD or chosen K/Host activation. T02 parent [#105](https://github.com/kaicreator-mm/tally/issues/105) will install released public tarball bytes, select B/K/S/C via `buildV10App`, and run positive and negative external consumer tests with actual Build Host exit before admission. Pending independent Product #77, ADS #85, L2 #82 and source code review. No main merge.
