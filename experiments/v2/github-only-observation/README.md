# Real GitHub-only task-meta observation

Tally Issue #62 consumes actual public ADS v5 reports through the selected native C reader. No task-meta API, DB, Manifest, private Trace, second parser or direct feedback channel is used.

## Actual public evidence

`kaicreator-mm/task-meta-e2e` Issue #9:

- ZCode `6381704604`: `capture_coverage=PARTIAL`, 9 observed metric fields.
- Claude Code `6381719186`: `capture_coverage=COMPLETE`, 12 metric fields; native comment edited later, so currentness remains STALE.
- Codex `6381725137`: `capture_coverage=COMPLETE`, 12 metric fields.

All three use `attempt_binding=COLLECTOR_ONLY`, `source_sha=null`, `publisher_source_sha=5b7072d78016052b1ea816b2b36322b946ef5c78`, `source_status=UNPINNED`, verifier `NOT_RUN` and cost `UNKNOWN`. The publisher commit is verified in `kaicreator-mm/task-meta`, not guessed in the subject repository. Each record remains `NATIVE_SOURCE_STALE` and advisory-only.

A fourth real report, `6382627467`, uses source SHA `1c8ec325c2d4df61246e02d3f01d28f31ea9d91a`. That commit is unavailable from the subject repository and commit search, so the consumer records `SOURCE_UNAVAILABLE / GITHUB_NATIVE_READ_FAILED` instead of bypassing source qualification.

## Validation

`consumer.mjs` reads only the three accepted collector comments through `readGitHubIssueReports`, with Host bindings for publisher repository/namespace/comment IDs. It checks native comment identity, publisher SHA, JCS digest presence, actor identity, source/capture/verifier/cost status and observed metric epistemic values.

`observation.test.mjs` covers duplicate identity, edited source, schema mismatch, missing publication, 429, incomplete collector evidence and Host comment allowlists. No private manifest is dereferenced.

Dedicated Woodpecker profile `v2-github-only-observation` runs both the real consumer and offline negatives using an unauthenticated public Octokit client, because all cited reports and commits are public. `TALLY_REAL_GITHUB_ONLY_OBSERVATION=PASS` means the actual public reports were consumed with their limits preserved. It does not mean TASK_PASS, HIDDEN_PASS, production adoption, Release Qualification or main integration.
