# Tally v2 W-B — GitHub C native authority and ADS v4.18 report contract R0

**Owner:** [#88](https://github.com/kaicreator-mm/tally/issues/88). **Status:** SOURCE_CONTRACT_CANDIDATE / IMPLEMENTATION_NOT_ADMITTED / RUNTIME_TESTS_NOT_RUN / NO_PRIVILEGED_WRITE. Intended Tally ADS source: `4.18.0@2c395c55569fe2edf3f4135f12d3d069e526d0b7`; adoption [#85](https://github.com/kaicreator-mm/tally/pull/85) has genuine pinned verifier PASS on previous SHA but v2 custom CI required-check disposition remains blocked. Pin is **target** not effective main.

## Public wire owner — no invented Tally protocol

- ADS canonical reference [GitHub Structured Interaction](https://github.com/kaicreator-mm/ai-development-standard/blob/2c395c55569fe2edf3f4135f12d3d069e526d0b7/references/GITHUB_STRUCTURED_INTERACTION_REFERENCE.md): v3 `work_item` / `intent` / `event` remain the canonical non-report document kinds; current report family is **`schema: ads/github/v5`, `kind: report`** at the 4.18 source, via exact `schemas/github/v4.16/report.schema.json` Git blob `7478313cc006162432649d734e59c5cd77db5106`. v4.15 `ads/github/v4` report is a separately version-qualified prior shape, not silently coerced. Historical v3 Report is read-only.
- Three strict v5 report `data` profiles: `summary`, `feedback`, `correction`; schema validation is **SHAPE_VALID_ONLY**, never K Admission/engineering PASS. `report_id` plus `revision` and kind are part of the report identity; `publisher_source_sha` and `ads_revision` are independent of observational `subject.source_sha` (which can be null/UNKNOWN for bounded collector-only summary). Do not impute unknown token/cost or unauthenticated verifier success.
- Public digest is `SHA256(RFC8785_JCS(validated data))`, computed after validation in external readback receipt/index; **do not embed own digest or guessed future comment URL in the report `data`**. Private task-meta manifest digest is separate, and Tally must **never dereference** private manifest or task-meta API/DB/object storage.

## Semantic responsibility / public C boundary

| Concern | Primary owner | Input | Safe result / denial |
|---|---|---|---|
| GitHub native Issue/PR/Review/CI/Tag/Dependencies/Claim | C GitHub reader observes, B product evaluates, K guards app state | repo, object identifiers, current ETag/page/commit SHA | source-bound fact or `UNKNOWN/PARTIAL/STALE`; not synthetic state from SQLite |
| ADS report read | C GitHub validates exact current ADS schema/actual comment+revision, B forms advisory analysis | owning Issue, expected namespace/report_id/revision, source and publisher refs | `OBSERVED` with coverage / `INSUFFICIENT_EVIDENCE` / `UNSUPPORTED_SCHEMA` / `UNKNOWN` |
| Governed GitHub write | B proposes target and expected currentness; K authorizes; C GitHub executes actual GitHub request/readback | accepted actor role/scope, repository grant, exact target SHA, immutable idempotency identity | effect receipt only if actual readback matches; timeout/ACK loss reconciled before retry |
| Source-qualified Task feedback | B produces bounded use feedback; K→C GitHub publishes v5 `feedback` after legal Claim/permission | no task-meta private references; source report and exact procedure/run | real native comment receipt; correction is **not** authorization to cancel task |
| Read-side SQLite projection | selected derived cache package, NOT K and NOT GitHub authority | source SHA/report schema/comment revision/page completeness | rebuildable indices only, never a permission or Release PASS decision |

## Typed failure/coverage contract to freeze at L2 (no unverified SDK class names)

Read outcomes: `COMPLETE_CURRENT`, `PARTIAL`, `STALE`, `UNKNOWN`, `UNSUPPORTED_SCHEMA`, `INSUFFICIENT_EVIDENCE`, `PERMISSION_DENIED`, `RETRYABLE_UNAVAILABLE`; read outcomes are not K state transitions. Actual effect/write outcomes: `DENIED`, `CONFLICT`, `COMMITTED_READBACK_VERIFIED`, `UNKNOWN_EFFECT` (after network timeout; re-query before any repeat), with source/provider receipt. A valid schema or HTTP 202 alone is never `COMMITTED_READBACK_VERIFIED`. These are **Tally design vocabulary only** until reviewed and mapped to genuine DH public C/Host types.

## Source negative fixtures to implement independently of production Host

| ID | Case | Expected authority-safe result |
|---|---|---|
| GH-N01 | GitHub 403 / revoked repo grant | PERMISSION_DENIED; previous cache cannot approve |
| GH-N02 | 429 with retry-after exceeding bounded budget | UNKNOWN; no false complete read |
| GH-N03 | Only first page returned, remaining pages unobserved | PARTIAL even if valid records exist |
| GH-N04 | Stale Issue/PR revision or force-pushed HEAD | STALE; affected derived recommendation invalid |
| GH-N05 | v5 report under wrong effective ADS pin v4.15 | UNSUPPORTED_SCHEMA; never downcast v5 |
| GH-N06 | v4/v3 historical Report read by v4.18 code | historical-only qualified profile; no new v3 emit |
| GH-N07 | Summary `subject.source_sha=null`, epistemic UNKNOWN | preserve UNKNOWN; never infer verified run |
| GH-N08 | Valid envelope missing GitHub Claim/permission | SHAPE_VALID_ONLY, not ADMITTED or Task PASS |
| GH-N09 | Publisher SHA and Task source SHA differ | distinct identities retained; no silent rebinding |
| GH-N10 | Comment edited but same report_id/revision conflicts | FAIL_CLOSED evidence conflict; cannot choose newest arbitrary |
| GH-N11 | Correction references another Issue/report/namespace | reject cross-target correction despite correct schema |
| GH-N12 | Timeout after GitHub accepted write | read native source by idempotency identity first, avoid duplicate |
| GH-N13 | Two concurrent source-changing intents both schema valid | K serial permit + C expected revision; one rejected/stale |
| GH-N14 | Task-meta collector stopped or failed to publish | Task/Verifier actual status unaffected, telemetry UNKNOWN |
| GH-N15 | Malicious YAML inside human Markdown discussion | not a canonical first-line marker, ignore as data |
| GH-N16 | Private task-meta manifest URI in summary | retain opaque locator, **never fetch** |
| GH-N17 | Report unknown cost or quality metric | null/UNKNOWN, not 0 |
| GH-N18 | GitHub write requested by Local Builder using gh/GCM/SSH agent | `UNGOVERNED_EFFECT`; only K→C approved publisher may write |
| GH-N19 | Same transport login, different logical publisher | preserve actor self-declaration and review provenance; no impersonation PASS |
| GH-N20 | Report correction received while cache is stale | mark stale and re-read current GitHub native comment/currentness before reanalysis |

## Narrow validation and implementation DAG

1. Source fixtures: use pinned ADS v4.18 schema/reference and offline fixture object IDs; run *actual* v4.18 shape verifier if used and report exit; source-only consumer conformance can be reviewed before DH R01.
2. Accepted public ABI from [#87](https://github.com/kaicreator-mm/tally/issues/87) + independent Product/L2/ADS pin gates → one C GitHub reader Package (no private task-meta calls), a separate governed C GitHub writer concern, and derived cache [#89](https://github.com/kaicreator-mm/tally/issues/89). Split Contract/Core/Integration PRs; never deliver one giant C service.
3. Real integration: GitHub test repo with actual Issue/PR/report, 403/partial/revocation, K Effect/source SHA and readback, observer fail-open. Actual tests/CI EXIT preserved. No live effect or Release PASS is authorized merely by this R0 contract.

**Review request:** exact Git blob/HEAD; `P0/P1/P2`, source-schema compatibility, ownership boundaries, negative fixture omissions, scope-only readiness. No physical Test PASS/Claim/merge.
