# SEK-09F — first-party incident case source review (candidate data)

**Parent:** #351 / #299 · **Task:** #365 · **Corpus WIP pinned HEAD:** `55ab06a8a09b8aa850e1c75743d66420d8c57b56` · **Authoring schema:** 0.3.0 · **Status:** TWO_REAL_CASE_CANDIDATES / NOT_GRAPH_MERGED / NOT_ACCEPTED.

## Verified source-specific facts

| Case ID | Official publisher first-party sources | Verified scope | Rights / uncertainty |
|---|---|---|---|
| `se:CASE-09F-CF-KV-DEPENDENCY` | [Cloudflare June 12, 2025 incident](https://blog.cloudflare.com/cloudflare-service-outage-june-12-2025/); [August 8, 2025 KV redesign](https://blog.cloudflare.com/rearchitecting-workers-kv-for-redundancy/) | Workers KV backing failure and downstream availability impact; Cloudflare-reported 2h28m and no loss; service response and design follow-up | Publisher self-report; HTML content version/hash not frozen; remediation benefit unmeasured |
| `se:CASE-09F-GH-DB-FAILOVER` | [GitHub October 21 postmortem](https://github.blog/news-insights/company-news/oct21-post-incident-analysis/) | Network partition, database leader failover, divergent writes, safety-first manual recovery | Firsthand but historical/MySQL-topology-specific; content revision/hash not frozen |

**Intake method:** targeted reading of the three official publisher-owned public web pages on 2026-11-08; original short paraphrases only. Source title, dates and reported incident findings are first-party assertions, not independently adjudicated root causes. No PDF/HTML snapshot or copyrighted full article is checked in. No content hash or source immutability is claimed; independent source-reviewer may inspect archived/current revisions and refine uncertainty within this exact task-owned write set.

**Model shape:** machine-readable `SEK-09F-INCIDENT-CASES.yaml` uses the JSON subset of YAML (so PyYAML accepts it) with exact 0.3.0 typed field names: 3 Source, 4 SourceLocator, 5 Evidence, 2 Case. Graph-wide ID uniqueness and two case-to-evidence-to-locator-to-source closure are required. This is an **intake sidecar**, not a new authoritatively included lane and not a rebuilt `graph.jsonld`. No Case or Counterexample is yet present in the integrated 793-node graph.

## Case vs Counterexample boundary

The incidents illustrate that shared dependencies and replication divergence can defeat simplistic architectural assumptions; **neither directly refutes** the current qualified `se:PRI-07D-PLAN-FOR-FAILURE` or `se:PAT-07D-CONTEXT-DEPENDENT-FAULT-RESPONSE` records, whose scopes already account for failure context. Therefore do not invent a `Counterexample.counterexample_to_refs` edge by mislabeling these supportive cases as contradictions. `counterexamples: []` records the outstanding schema/usefulness gap without faking a counterclaim. Any genuine counterexample requires a specific, evidence-backed target proposition and scoped contrary observation, reviewed before graph merge.

## Validation and next legal integration

- Initial bounded structural preflight: 14 unique se: IDs; 3 Source references, 5 Evidence locator references and 5 Case evidence references resolve in the same sidecar; both Case states PENDING_REVIEW with synthetic=false; 0 ReviewDecision, 0 Counterexample, 0 ACCEPTED.
- The source content, title/section locators and rights must receive **independent exact-SHA review**; do not confuse this Builder preflight with LinkML/PyYAML/graph validator PASS (those **NOT_RUN** for this sidecar).
- Separate authorized graph Writer must register sources and merge cases into canonical lane/graph, run full schema/relationship/JSON-LD/Turtle/manifest rebuild, update truthful coverage, and obtain independent review. Do not update historical 29 OPEN gaps in this intake PR.
- G4 human review and G5 controlled model scoring remain NOT_CONDUCTED/NOT_RUN; this file is not a release or DomainForge promotion.
