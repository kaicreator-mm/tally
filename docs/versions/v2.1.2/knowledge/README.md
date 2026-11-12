# Tally v2.1.2 C1 Original Owner source amendment (R13)

STATUS: ORIGINAL_SOURCE_OWNER_DECISIONS_2_OF_2 / SOURCE_CANDIDATE_CANONICAL_LANE_INTEGRATED / ORACLE_NOT_ATTESTED.
Owner confirmation: https://github.com/kaicreator-mm/tally/issues/398#issuecomment-6808543942
Immutable original commit: 3e94685ca57a4b7258988826697af1c1c3a6b3c8
Immutable original Graph Blob: 0f6007315fd7352fe6fd218effcfecf672cb4961
Single-Writer branch based on wip/v2.1.1-knowledge at 706a3ec33c2ba7ba3b60116471139113edb7a6cb.
Frozen `docs/versions/v2.1.1`, historical #299/#351 and frozen R8 are not modified (verified by Git Blob pins and PR diff scope).

## Source record

`source/C1_OWNER_REVISION_R13.json` — Owner-authorized overlay (schema
`tally.dke.c1-owner-authorized-overlay/1`, Git Blob pinned by the builder):
replaces the unsupported uniformity constraint in the DecisionModel and its
associated Pattern (OD05=NARROW_CLAIM, CS2023 at-least-one educational scope
preserved), and adds a distinct first-party 2018-10-22 16:24→23:03 UTC GitHub
incident locator plus separate PENDING evidence with the ~200,000 dropped
TTL-expired webhook payloads caveat (GH-2018=ACCEPT_PATCH). The original 16:24
topology locator remains unchanged. Both new records are additions; the
Evidence stays PENDING_REVIEW / EXTRACTED_PENDING_REVIEW; publisher raw HTML
SHA256 remains UNKNOWN. Originals are changed only by the explicit
`changes` entries.

## Canonical lane-integrated build (native Node, zero dependencies)

The v2.1.2 candidate is produced by the same canonical source lane as
v2.1.1: the ten frozen lane bundles + `SOURCE_REGISTRY.yaml` +
`RELATIONSHIP_CATALOG.yaml` are parsed with a strict YAML-subset parser,
projected per the documented v2.1.1 semantics, and the Owner overlay is
applied post-projection. Tools:

| Tool | Role |
|---|---|
| `tools/yaml-min.mjs` | strict YAML-subset parser (PyYAML-1.1-compatible on this corpus; duplicate keys fail closed; parity-proven below) |
| `tools/rdf-min.mjs` | deterministic JSON-LD/Turtle serialization, canonical N-Triples + SHA-256 digest, strict Turtle parse-back |
| `tools/project-c1.mjs` | lane assembly, source derivation, typed-edge materialization, overlay application, invariants (Git Blob pins for every frozen input) |
| `tools/materialize-c1.mjs` | Owner-overlay candidate preview (overlay on the generated graph path) |
| `tools/build-c1-canonical.mjs` | build CLI: `--check` / `--write` / `--reproduce-v211` |
| `tools/build-c1-canonical.test.mjs` | native `node --test` suite: positive / negative / UNKNOWN / source-failure |
| `tools/make-parity-snapshot.py` | (pinned Python) regenerates `fixtures/c1-yaml-parity.sha256` |
| `tools/verify-c1-exports.py` | (pinned Python) independent rdflib parse-back verification |

Build gates (all enforced by `build-c1-canonical.mjs --check`):

- **R1** frozen-input Git Blob SHA-1 pins (10 lanes, registry, catalog, frozen
  graph `0f600731…`, overlay `d7900570…`) — drift or missing input fails closed.
- **R2** lane-integration fidelity: the overlay-free projection reproduces the
  frozen v2.1.1 `generated/graph.jsonld` and `generated/graph.ttl`
  **byte-for-byte** and the canonical N-Triples digest
  `2625a6b336a8a0b392e7eccd13f157e3fb9030e9262836a12ec8afc85eb2d2e8`.
- **R3** candidate invariants: 809 nodes; 106 PENDING KnowledgeClaims; 0
  ReviewDecision / Counterexample / ACCEPTED; OD05 narrowing applied (no
  universal uniformity); GH-2018 separate locator + PENDING evidence +
  dropped-payload caveat; UNKNOWN conserved; all references resolve; originals
  preserved except the explicit `changes`.
- **R4** the Owner-overlay candidate preview
  (`generated/c1-candidate.graph.jsonld`) is **byte-identical** with the
  canonical lane-integrated export `generated/graph.jsonld` — the two
  construction paths (overlay on generated graph vs. lane-level projection +
  overlay) converge.
- **R5** Turtle parse-back == JSON-LD expansion == internal canonical triple
  set (9,678 triples; canonical digest `2e4fbab6e056137317032ce15cebe1a41e22ea8955835250e06c721987f0c8fc`).
- **R6** determinism: independent rebuilds byte-identical.
- **R7** `generated/manifest.sha256` digests match the committed bytes.

Parity with PyYAML (two independent parsers over the whole frozen corpus) is
asserted by the native suite against `fixtures/c1-yaml-parity.sha256`
(12 inputs; ~10.5k scalar leaves). Independent rdflib verification
(`tools/verify-c1-exports.py`, Python 3.14.7 + rdflib 7.6.0, the pinned
SEK-08/09H environment) parses both exports back and confirms TTL ≡ JSON-LD,
counts and manifests — for the v2.1.2 candidate **and** the frozen v2.1.1
pair (`--also-v211`).

## Native execution (repository root)

```sh
node --check docs/versions/v2.1.2/knowledge/tools/materialize-c1.mjs
node --check docs/versions/v2.1.2/knowledge/tools/materialize-c1.test.mjs
node --check docs/versions/v2.1.2/knowledge/tools/yaml-min.mjs
node --check docs/versions/v2.1.2/knowledge/tools/rdf-min.mjs
node --check docs/versions/v2.1.2/knowledge/tools/project-c1.mjs
node --check docs/versions/v2.1.2/knowledge/tools/build-c1-canonical.mjs
node --check docs/versions/v2.1.2/knowledge/tools/build-c1-canonical.test.mjs
node --test docs/versions/v2.1.2/knowledge/tools/materialize-c1.test.mjs
node --test docs/versions/v2.1.2/knowledge/tools/build-c1-canonical.test.mjs
node docs/versions/v2.1.2/knowledge/tools/materialize-c1.mjs --check
node docs/versions/v2.1.2/knowledge/tools/build-c1-canonical.mjs --reproduce-v211
node docs/versions/v2.1.2/knowledge/tools/build-c1-canonical.mjs --check
```

`--write` regenerates `generated/{graph.jsonld,graph.ttl,manifest.sha256}`
and the preview; never run it blind — `--check` is the verification entry.

## Qualification boundary (unchanged)

Native execution and this canonical lane integration are **not** a semantic
qualification. Distinct Element/Global/Root semantic Oracle **NOT_ATTESTED**;
SAP 0/6; Root UNKNOWN; frozen DKE R8 0/53; DIS CORE-002/005/016 PARTIAL; no
ReviewDecision exists; both new source claims stay PENDING. No ReviewDecision,
SAP, DIS Conformance, Product L2 or Release automatic qualification. DKE R8
remains 0/53; a separately accountable semantic Oracle must test Element /
five Global / eight Root / six SAP and original R8 before any downstream DIS
conformance.

## CI coverage note

The Woodpecker v2 carrier (`.woodpecker/v2-*.yaml`) triggers only on
`version/v2` (legacy `verify.yaml` on `main`); no profile covers the
`wip/v2.1.1-knowledge` base branch or the `docs/versions/v2.1.2/**` paths of
this PR. CI status for this branch is therefore **NOT_TRIGGERED by
configuration, not by failure**; the native exact-SHA commands above are the
executable verification evidence for this candidate, and the version/v2
pipelines remain the carrier for any future version-branch integration.
