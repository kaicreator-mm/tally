# R20 · Decision-time knowledge consumption reference

Status: RESEARCH_REFERENCE / T1_T2_BOUNDED_CALIBRATION / NO_PRODUCT_INTEGRATION. See [#561](https://github.com/kaicreator-mm/tally/issues/561).

This small, offline Node.js reference consumes caller-assessed decision facts, not natural-language risk recognition. It materializes exact advisory excerpts, not authorized business rules or proof of Agent comprehension. It is not a Tally/DH runtime, SDK, agent loop, or second knowledge owner. No production import or packaging configuration is changed.

## Run

Node.js 22.16.0 used, no npm installation. From this directory:

```sh
node --check selector.mjs
node --check selector.test.mjs
node --test selector.test.mjs
node selector.mjs example.json
```

The example is synthetic, not an actual authorized task. An array of distinct decisions is also accepted.

## Input and output contract

Each decision has decisionId, issue, sourceSha (40 lower-case hex characters), stage (L1/L2/L3/QA/MAINTENANCE), question, contract (CONFIRMED/MISSING/CONFLICT), contractEvidence, and risks: [{id, applicability: YES/NO/UNKNOWN, evidence}]. The caller identifies risk and supplies source evidence; the selector checks syntax and consistency, **not** whether that evidence is true, current or authorized on GitHub. A missing/conflicting contract returns ASK; an unknown/unsupported or out-of-stage relevant risk returns UNKNOWN. An empty or explicitly inapplicable set returns NONE. Explicit NO is excluded, with reason; this is not proof that the caller identified all risks.

Supported initial IDs and eligible stages: boundary (L2), alias (L2/L3), abstraction (L2/L3), precision (L2/L3/QA), oracle (QA), maintenance (MAINTENANCE), effects (L2/L3/QA), cost (any). These are reference routing labels, not new Kxx IDs or an exhaustive risk taxonomy. Effects points to a cross-module route, not a full solution. The K01 contract-gap path is ASK, not a K01 excerpt; other modules and unindexed risks remain outside this small snapshot.

Default budget is 3 pieces / 6144 UTF-8 excerpt bytes, excluding metadata and model prompt/token cost. The caller may explicitly enlarge it. If relevant pieces do not all fit, UNKNOWN rather than partial success; no arbitrary truncation. There is no semantic ranking, automatic neighbor expansion or LLM risk classifier in this first reference.

`select` returns SELECTED metadata only; `deliver` materializes selected text and returns a deterministic receipt digest. DELIVERED means MATERIALIZED_NOT_READ; not network delivery or proof a model saw it. `recordApplication` binds caller-reported before/after decision, reason, falsifying probe and outcome APPLIED/ALREADY_ALIGNED/REJECTED/UNKNOWN to the receipt; self-reported, not verified. Equality/change is textual, not semantic. A receipt digest is an integrity identifier, not a signature, security boundary, or evidence of chronology. Actual first code SHA, independent behavior, QA, maintenance/cost and Agent consumption are T3/T4 work, not inferred here.

## Exact source projection

The knowledge source is main commit 874702df43a7563b624b270fb2389c368050408b. Catalog paths are relative to docs/versions/v2.1.2/software-engineering-knowledge/ at that commit. Each entry records Git blob, one-based inclusive line range, exact UTF-8 text (LF-joined lines, with no additional terminal LF beyond the specified range) and SHA256. Catalog canonical JSON serialization is separately pinned in selector.mjs. This is a versioned research projection, not a source replacement. Catalog bytes were independently reconstructed from GitHub's exact commit contents via the connector and matched the locally tested Git blob. A caller who has a full checkout can compare each line range with git show at the pinned commit. No full clone was available in the execution environment; source pinning does not validate current task facts.

## T2 calibration evidence

Node v22.16.0, Linux, offline. Both node --check calls passed; 14/14 node:test cases passed after anchoring the full catalog identity; CLI example DELIVERED alias only, 869 excerpt bytes, MATERIALIZED_NOT_READ. Cases cover eight routes, same text/different declared facts, NONE, ASK, unresolved/stage/unsupported risk, stale and tampered catalog (including rehash), UTF-8 budgets, explicit exclusion, exact delivery, multiple decisions, unchanged/changed self-report, invalid inputs. All task contexts in these tests are synthetic; no real Agent or product execution and no knowledge effectiveness claim.

Local tested file Git blob IDs:
- selector.mjs: e79114b54c4e1d26f6d7dee9d30b40d2089ca107
- selector.test.mjs: a603ae6cc33325ce06adcfacf0ab8db6ea68c012
- catalog.json: 28c76328d2b590154f73708c3d830e92318271f2
- example.json: a35555522125cafd109b71960ef8df3007fc2b8f

No production integration, CI or Windows runtime tests run. T3 must validate a genuine bounded authorized code decision using source facts, exact task baseline, first code SHA, authorized falsifier and actual execution; do not repeat R19-F2.
