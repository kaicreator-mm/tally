# Tally v2.1.2 — C1 original-Source-Owner revision (single-Writer)

**Status:** DRAFT / EXPERIMENTAL / HUMAN_REVIEW_PENDING — nothing here is ACCEPTED;
independent Element/Global/Root semantic Oracle: **NOT_ATTESTED**; no qualification
promotion of any kind.

**Authorizing decision:** [Tally #398 R13](https://github.com/kaicreator-mm/tally/issues/398#issuecomment-6808543942)
— the original Tally Source Owner (authenticated GitHub account `kaicreator-mm`,
user id `274507369`, matching the original source commit author of the pinned
snapshot `3e94685ca57a4b7258988826697af1c1c3a6b3c8`) explicitly confirmed:

1. **OD05 = NARROW_CLAIM** — `se:DM-REQ-FUNCTIONAL-DOC-METHOD`: only the CS2023
   SE-Requirements *Product Requirements* KA Core 1a–1c supported *at-least-one*
   documentation/structuring method (use cases / user stories as named examples) with
   consideration of how the method supports design and implementation and of
   approach-specific strengths/weaknesses. The universal one-method-uniformity
   assertion is removed; selection rationale is constrained to curriculum scope.
2. **GH-2018 = ACCEPT_PATCH** — `se:CASE-09F-GH-DB-FAILOVER`: keep the original
   `se:LOC-09F-GH-REPLICATION` (ends 2018-10-22 16:24 UTC) for replication/topology
   assertions and add a distinct time-scoped first-party locator plus PENDING evidence
   covering backlog/recovery outcome toward 23:03 UTC
   ([GitHub October 21 post-incident analysis](https://github.blog/news-insights/company-news/oct21-post-incident-analysis/)),
   explicitly preserving the approximately 200,000 TTL-expired dropped-payload caveat.

## What this version contains

Derived from the v2.1.1 knowledge work at WIP `706a3ec3` (whose
`docs/versions/v2.1.1/knowledge/generated/graph.jsonld` is byte-identical to the
original pinned Blob `0f6007315fd7352fe6fd218effcfecf672cb4961`; 807 `@graph`
objects, 106 PENDING KnowledgeClaims, 0 ReviewDecision). Exactly two authorized
source changes are applied; the deterministic RDF projection is rebuilt:

- `authority/` — unchanged copy of v2.1.1 (registry, research framework,
  source reviews). No new source is registered: the new locator cites the already
  registered `SE-SRC-09F-GH-POSTMORTEM-2018`.
- `knowledge/model/`, `knowledge/case-intake/`, `knowledge/vertical-slice/` —
  model/catalog/validator and lane layout copied; two lane files edited
  (`requirements`: OD05 narrowing; `reliability-performance`: GH patch).
- `knowledge/generated/` — rebuilt projection: 808 records, 809 `@graph` objects,
  9,680 triples, canonical digest
  `e56bed65b7f90d4bff1e6efdadd55abad0de30837e12e3e30be4c54e5687eac6`.
- `knowledge/HANDBOOK.md`, `knowledge/generated/README.md`,
  `knowledge/generated/COVERAGE_REPORT.md` — counts and derivation updated.
- `methods/observations/C1-R13-REVISION.md` — decision record, first-party
  readbacks, gate receipts (validator / LinkML / determinism / equivalence /
  negative fixtures / C1 positive-negative-UNKNOWN), exact file digests.

## What is deliberately excluded and unchanged

- `docs/versions/v2.1.1/**` — **byte-identical to the WIP baseline in this branch**;
  the frozen historical v2.1.1 graph, the pinned snapshot
  `3e94685ca57a4b7258988826697af1c1c3a6b3c8`, historical #299/#351 evidence and
  frozen R8 are not rewritten and not silently altered.
- v2.1.1 `experiments/` (SEK-09G paired-trial runs, linkml-pof) and
  `methods/observations/SEK-*` — historical method evidence of v2.1.1; not copied,
  so v2.1.2 does not imply re-running those experiments. The v2.1.1 tree remains the
  authoritative location for that history.
- v2.1.1 `PILOT_CHARTER.md` — pilot-specific; this targeted revision is scoped by
  this README instead.

## Qualification accounting (unchanged by this revision)

- Original Source Owner decisions: **2/2 CONFIRMED** (#398 R13) and now executed as
  source bytes at this version.
- Independent semantic Oracle (Element cohesion/split/merge, Global invariants, Root
  composition): **NOT_ATTESTED** — no self-signing by the Writer/Owner.
- SAP: **0/6**; Root: **UNKNOWN**; DKE frozen R8: **0/53** qualified.
- DIS CORE-002/005/016: remain PARTIAL pending the independent Oracle; no Core
  promotion is claimed by this version.
- All 106 knowledge claims stay `PENDING`; both cases stay `PENDING_REVIEW`; the
  revised DecisionModel stays `PROPOSED` (record_version 0.2.0).
