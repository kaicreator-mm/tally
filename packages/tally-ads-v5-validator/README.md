# Tally v2 T03a — pinned official ADS v4.18 report Schema validator

Parent [#106](https://github.com/kaicreator-mm/tally/issues/106); separate subconcern from GitHub API pagination/writer and [T01 #104](https://github.com/kaicreator-mm/tally/pull/104), which is under independent Claude review.

* Upstream source: `kaicreator-mm/ai-development-standard@2c395c55569fe2edf3f4135f12d3d069e526d0b7`
* Exact canonical JSON Schema: `schemas/github/v4.16/report.schema.json`, Git blob `7478313cc006162432649d734e59c5cd77db5106`.
* Vendored bytes are checked against the exact **Git blob SHA** at module load, not a descriptive filename alone.
* Draft2020-12 validation with Ajv 8.17.1; `additionalProperties: false` and conditional/oneOf validation stay active. The validator neither coerces values, mutates data nor silently accepts historical `ads/github/v4` as v5.
* Official v5 `report_kind` values: `EXECUTION_SUMMARY`, `EXPERIENCE_USE_FEEDBACK`, `EXECUTION_CORRECTION`. **Do not** use invented lowercase short names.
* Fixtures are *illustrative offline adaptations* of the upstream prior v4.15 example records, with new mandatory subject/metric/verifier/publication fields populated. They are not real GitHub reports, and schema acceptance does **not** certify native API readback, RFC8785 JCS publication digest, signing, GitHub permission, Source Claim, K admission, Task PASS, Release PASS or original private manifest.
* `validateAdsV5Report` accepts a pre-parsed JSON object and refuses literal YAML/Markdown; the transport must implement a separate correct canonical GitHub marker/parser, defend duplicate JSON keys and bind Issue/comment/source/revision. Transport + JCS digest is **T03b**.
* This is a standalone Tally C validator module (pure data validation), not another Kernel or Task state store. Only after independent review should the real GitHub C reader feed its success into T01. It cannot grant permissions itself.

Run `npm install --ignore-scripts --no-audit --no-fund && npm test` from this package using Node24. Package lock is not yet source-qualified; the first CI task intentionally tests a pinned direct Ajv 8.17.1 version and must record actual installed transitive closure, then commit audited package-lock before production integration. No tests claimed until actual CI/real Node run.
