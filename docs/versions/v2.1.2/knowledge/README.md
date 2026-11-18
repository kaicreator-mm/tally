# Tally v2.1.2 C1 Original Owner source amendment (R13) + R102 PDF Locator repair + R107 disposition contract + R137 selected-CQ two-leg source binding + R180 AI cross-model review ledger

STATUS: ORIGINAL_SOURCE_OWNER_DECISIONS_2_OF_2 / SOURCE_OWNER_DISPOSITION_CONTRACT_RESOLVED / SELECTED_CQ_TWO_LEG_SOURCE_BOUND_CONSUMED / AI_CROSS_MODEL_REVIEW_LEDGER_RECORDED_SCOPED / CLAIM_LIFECYCLE_PENDING / ORACLE_NOT_ATTESTED.
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

SourceOwnerDisposition is not ReviewDecision: both Claims remain `PENDING` /
`PENDING_REVIEW`, Evidence remains `EXTRACTED_PENDING_REVIEW`, and no record
becomes ACCEPTED. Professional Oracle remains NOT_ATTESTED and publication
permission is not granted.

## R137 selected-CQ two-leg source binding (p246 SECTION + p248 LO2)

The selected candidate VV CQ (`CS2023_VV_TEACHING`, research candidate only)
consumes **both** publisher facts executably:

- **Section-start leg (p246)**: `se:LOC-CS2023-SE-VALIDATION-KU` rv0.1.1
  (`locator_type=SECTION`, PDF zero-based index 245 / printed 246) stays the
  shared anchor of exactly 8 Evidence (`EVD-TVV-0003…0009,0014`) and 5 direct
  Claims (`CLM-TVV-0001/0002/0003/0006/0007`) — unchanged, byte-identical
  originals, no 8/5 fan-out damage.
- **Outcome-pinpoint leg (p248)**: new versioned addition
  `se:LOC-CS2023-SE-VALIDATION-LO2` rv0.1.0 (`locator_type=PAGE`, PDF
  zero-based index 247 / printed 248, CS Core illustrative learning outcome 2
  "Distinguish between program validation and verification.", verified by
  full-text extraction of the official publisher PDF on 2026-11-15), consumed
  by new separate pending Evidence `se:EVD-TVV-0016` and linked to the
  selected Claim through `se:REL-TVV-0024` (catalog-legal overlay
  RelationshipAssertion; like all overlay additions it does not materialize a
  typed edge onto the locked original Claim).

The executable binding is `se:SCB-CS2023-VV-R1`
(`se:SelectedCqSourceBinding`, sidecar
`elementAttribution.selectedCqSourceBinding`, machine schema
`model/selected-cq-source-binding.schema.json`, native producer/validator
`tools/selected-cq-source-binding.mjs`): both legs carry exact record
versions + canonical SHA-256 digests recomputed from the graph (STALE fails
closed), distinct locators and distinct page anchors (a p246-as-LO2 disguise
fails closed), the claim-support assertion and both new records must actually
exist in the graph (orphans fail closed), the 8/5 shared fan-out and the
single-consumer property of the p248 locator are asserted on every build, and
Claim lifecycle/Oracle/publication boundaries are conserved
(`SELECTED_CQ_RESEARCH_CANDIDATE_ONLY_NOT_FROZEN_R8_CELL`). The original
Claim `se:CLM-TVV-0001` keeps exactly its original single `se:SUPPORTED_BY`
edge. Authority record: DIE DKE #2 R137
(https://github.com/kaicreator-mm/domain-intelligence-engineering/issues/2#issuecomment-6890002084).
This binding is scoped to the selected research candidate CQ; a formal frozen
R8 Cell LO2 dependency has NOT been established and is not widened here.

## R180 AI non-human cross-model review ledger (GPT / GLM / Qwen)

The R180 integrator convergence (three actual distinct model families, blind
first passes, P0/P1=0, unanimous `ACCEPT_SCOPED` on the two selected original
Claims) is recorded by the original Writer as a **versioned sidecar ledger**,
not as graph content: `source/C1_AI_CROSS_MODEL_REVIEW_DECISIONS_R180.json`
(schema `tally.dke.c1-ai-cross-model-review-decisions/1`, machine schema
`model/ai-cross-model-review-decision.schema.json`, native validator
`tools/ai-cross-model-review-decision.mjs`, harness
`tools/verify-c1-ai-cross-model-decisions-r180.mjs --check`). Integration
decision record: [#398 R180](https://github.com/kaicreator-mm/tally/issues/398#issuecomment-6983286610);
the three receipt comments (GPT 6976173032 / GLM 6976329069 / Qwen 6976586466)
are bound as three separate provenance references with code-pinned identity
(roster drift, duplicate model families and forged family labels — e.g. GLM
presented as Gemini — all fail closed; the Gemini lane itself remains NOT_RUN).

Hard properties enforced on every run: the reviewed candidate graph blob stays
exactly `a49ddf58…` and frozen v2.1.1 stays `0f600731…` (807 nodes) — the
ledger mutates nothing; both decisions exact-bind Claim/Evidence/Locator (and
the p248 pending leg) record versions + canonical SHA-256 digests; all 106
KnowledgeClaims stay `PENDING`, 0 in-graph `ReviewDecision`, no `se:AIM-*`
nodes are materialized; the p248 support leg
(`REL-TVV-0024`/`EVD-TVV-0016`/`LOC-…-LO2`) remains `PENDING_REVIEW` and is
never retroactively accepted; the 104 non-selected claims are untouched; P2
review observations are recorded with `PRESERVED_NOT_AUTO_APPLIED` only;
human/professional Oracle stays NOT_ATTESTED, publication permission stays
NOT_GRANTED, and formal SAP 0/6 / Global 0/5 / Parent/Root UNKNOWN /
scoped R8 1/53 / Effect/Release unauthorized are conserved. This ledger is
AI non-human review evidence only — it is not a human professional
certification — and authorizes no Forge #208 Cell by itself; the Forge #208
single Writer decides that separately with its own exact provenance.

## Source record

`source/C1_OWNER_REVISION_R13.json` — Owner-authorized overlay (schema
`tally.dke.c1-owner-authorized-overlay/1`, Git Blob pinned by the builder):
replaces the unsupported uniformity constraint in the DecisionModel and its
associated Pattern (OD05=NARROW_CLAIM, CS2023 at-least-one educational scope
preserved), and adds a distinct first-party 2018-10-22 16:24→23:03 UTC GitHub
incident locator plus separate PENDING evidence with the ~200,000 dropped
TTL-expired webhook payloads caveat (GH-2018=ACCEPT_PATCH). The original 16:24
topology locator remains unchanged. R137 additions (`LOC-…-LO2`,
`EVD-TVV-0016`, `REL-TVV-0024`) follow the same additions channel. All new
records are additions; the Evidence stays PENDING_REVIEW /
EXTRACTED_PENDING_REVIEW; publisher raw PDF/HTML SHA256 remains UNKNOWN.
Originals are changed only by the explicit `changes` entries.

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
| `tools/selected-cq-source-binding.mjs` | R137 SelectedCqSourceBinding producer + two-leg consumption/fan-out/STALE validator |
| `tools/materialize-c1.mjs` | Owner-overlay + disposition + binding candidate preview (overlay on the generated graph path) |
| `tools/build-c1-canonical.mjs` | build CLI: `--check` / `--write` / `--reproduce-v211` |
| `tools/build-c1-canonical.test.mjs` | native `node --test` suite: positive / negative / UNKNOWN / source-failure |
| `tools/make-parity-snapshot.py` | (pinned Python) regenerates `fixtures/c1-yaml-parity.sha256` |
| `tools/verify-c1-exports.py` | (pinned Python) independent rdflib parse-back verification |

Build gates (all enforced by `build-c1-canonical.mjs --check`):

- **R1** frozen-input Git Blob SHA-1 pins (10 lanes, registry, catalog, frozen
  graph `0f600731…`, updated R137 overlay Git Blob), plus v2.1.2 Owner
  disposition sidecar, extension-schema and binding-schema pins — drift or
  missing input fails closed.
- **R2** lane-integration fidelity: the overlay-free projection reproduces the
  frozen v2.1.1 `generated/graph.jsonld` and `generated/graph.ttl`
  **byte-for-byte** and the canonical N-Triples digest
  `2625a6b336a8a0b392e7eccd13f157e3fb9030e9262836a12ec8afc85eb2d2e8`.
- **R3** candidate invariants: 815 nodes; 106 PENDING KnowledgeClaims; two
  SourceOwnerDispositions; one consumed SelectedCqSourceBinding; 0
  ReviewDecision / Counterexample / ACCEPTED; OD05 narrowing applied (no
  universal uniformity); GH-2018 separate locator + PENDING evidence +
  dropped-payload caveat; UNKNOWN conserved; all references resolve;
  originals preserved except the explicit `changes` and separately
  materialized disposition/binding additions.
- **R4** the Owner-overlay candidate preview
  (`generated/c1-candidate.graph.jsonld`) is **byte-identical** with the
  canonical lane-integrated export `generated/graph.jsonld` — the two
  construction paths (overlay on generated graph vs. lane-level projection +
  overlay) converge.
- **R5** Turtle parse-back == JSON-LD expansion == internal canonical triple
  set (9,812 triples; canonical digest `84e840cbd43a15fb8e41e959d18f1f49fcf37e79fbd03f6adc63c16857cf2034`).
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
node --check docs/versions/v2.1.2/knowledge/tools/selected-cq-source-binding.mjs
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
python docs/versions/v2.1.2/knowledge/tools/verify-c1-exports.py . --also-v211
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
conformance. The R137 two-leg binding covers the selected research candidate
CQ only and does not widen the frozen R8 gate.

## CI coverage note

The Woodpecker v2 carrier (`.woodpecker/v2-*.yaml`) triggers only on
`version/v2` (legacy `verify.yaml` on `main`); no profile covers the
`wip/v2.1.1-knowledge` base branch or the `docs/versions/v2.1.2/**` paths of
this PR. CI status for this branch is therefore **NOT_TRIGGERED by
configuration, not by failure**; the native exact-SHA commands above are the
executable verification evidence for this candidate, and the version/v2
pipelines remain the carrier for any future version-branch integration.
