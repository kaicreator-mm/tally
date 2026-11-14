# Tally v2.1.2 C1 Original Owner source amendment (R13) + R102 PDF Locator repair + R107 disposition contract

STATUS: ORIGINAL_SOURCE_OWNER_DECISIONS_2_OF_2 / SOURCE_OWNER_DISPOSITION_CONTRACT_RESOLVED / CLAIM_LIFECYCLE_PENDING / ORACLE_NOT_ATTESTED.
Owner confirmation: https://github.com/kaicreator-mm/tally/issues/398#issuecomment-6808543942
Immutable original commit: 3e94685ca57a4b7258988826697af1c1c3a6b3c8
Immutable original Graph Blob: 0f6007315fd7352fe6fd218effcfecf672cb4961
Single-Writer branch based on wip/v2.1.1-knowledge at 706a3ec33c2ba7ba3b60116471139113edb7a6cb.
Frozen `docs/versions/v2.1.1`, historical #299/#351 and frozen R8 are not modified (verified by Git Blob pins and PR diff scope).

## R102 SourceLocator P2 correction

Two incorrect frozen CS2023 PDF index references 174-176 are corrected in the new version only: REQ zero-based index 241/printed 242; Validation zero-based index 245/printed 246. New Locator record version 0.1.1; frozen v2.1.1 remains byte-identical, 106 PENDING Claims, no Oracle, SAP or R8 promotion. The canonical JSON-LD/Turtle/manifest and R100 bindings reflect the new source evidence.

## R107 SourceOwnerDisposition contract

`TALLY-C1-R100-SCHEMA-01` is resolved in this new version by two canonical
`se:SourceOwnerDisposition` records (`se:SOD-CS2023-REQ-R1` and
`se:SOD-CS2023-VV-R1`). The machine-readable
`model/source-owner-disposition.schema.json` and native
`tools/source-owner-disposition.mjs` bind each disposition to the exact
Claim/Evidence/original-Locator/candidate-Locator record versions and canonical
SHA-256 digests, Source Owner identity/authority, scoped outcome, rights,
positive/negative/UNKNOWN boundaries and candidate Element attribution.

The graph version is `0.2.0`. ACTIVE bindings fail closed when a target digest
or version becomes STALE; a superseding event must name `supersedes_ref`, increase
its record version and leave the prior event non-ACTIVE. SourceOwnerDisposition
is not ReviewDecision: both Claims remain `PENDING` / `PENDING_REVIEW`, Evidence
remains `EXTRACTED_PENDING_REVIEW`, and no record becomes ACCEPTED. Professional
Oracle remains NOT_ATTESTED and publication permission is not granted.

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
| `tools/source-owner-disposition.mjs` | pin/load, JSON-Schema-profile validation, canonical SourceOwnerDisposition producer and exact-binding/STALE checks |
| `tools/materialize-c1.mjs` | Owner-overlay + disposition candidate preview (overlay on the generated graph path) |
| `tools/build-c1-canonical.mjs` | build CLI: `--check` / `--write` / `--reproduce-v211` |
| `tools/build-c1-canonical.test.mjs` | native `node --test` suite: positive / negative / UNKNOWN / source-failure |
| `tools/make-parity-snapshot.py` | (pinned Python) regenerates `fixtures/c1-yaml-parity.sha256` |
| `tools/verify-c1-exports.py` | (pinned Python) independent rdflib parse-back verification |

Build gates (all enforced by `build-c1-canonical.mjs --check`):

- **R1** frozen-input Git Blob SHA-1 pins (10 lanes, registry, catalog, frozen
  graph `0f600731…`, updated R102 overlay Git Blob), plus v2.1.2 Owner
  disposition sidecar and extension-schema pins — drift or missing input fails closed.
- **R2** lane-integration fidelity: the overlay-free projection reproduces the
  frozen v2.1.1 `generated/graph.jsonld` and `generated/graph.ttl`
  **byte-for-byte** and the canonical N-Triples digest
  `2625a6b336a8a0b392e7eccd13f157e3fb9030e9262836a12ec8afc85eb2d2e8`.
- **R3** candidate invariants: 811 nodes; 106 PENDING KnowledgeClaims; two
  SourceOwnerDispositions; 0 ReviewDecision / Counterexample / ACCEPTED; OD05
  narrowing applied (no universal uniformity); GH-2018 separate locator +
  PENDING evidence + dropped-payload caveat; UNKNOWN conserved; all references
  resolve; originals preserved except the explicit `changes` and separately
  materialized disposition additions.
- **R4** the Owner-overlay candidate preview
  (`generated/c1-candidate.graph.jsonld`) is **byte-identical** with the
  canonical lane-integrated export `generated/graph.jsonld` — the two
  construction paths (overlay on generated graph vs. lane-level projection +
  overlay) converge.
- **R5** Turtle parse-back == JSON-LD expansion == internal canonical triple
  set (9,746 triples; canonical digest `dcbada98bc79753468622bf8a51b8c518a2de39ae616b57eae9552a6378ea100`).
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
node --check docs/versions/v2.1.2/knowledge/tools/source-owner-disposition.mjs
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
node docs/versions/v2.1.2/knowledge/tools/verify-c1-owner-claims-r100.mjs --check
```

`--write` regenerates `generated/{graph.jsonld,graph.ttl,manifest.sha256}`
and the preview; never run it blind — `--check` is the verification entry.

## Qualification boundary (unchanged)

Native execution and this canonical lane integration are **not** a semantic
qualification. Distinct Element/Global/Root semantic Oracle **NOT_ATTESTED**;
SAP 0/6; Root UNKNOWN; scoped EV16:D02 R8 remains 1/53 with zero increment
from this contract and formal full R8 remains 0/53; DIS CORE-002/005/016 PARTIAL; no
ReviewDecision exists; both new source claims stay PENDING. No ReviewDecision,
SAP, DIS Conformance, Product L2 or Release automatic qualification. Formal full
DKE R8 remains 0/53; a separately accountable semantic Oracle must test Element /
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
