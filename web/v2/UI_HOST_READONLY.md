# Tally v2 native read-only Host — operator instructions

This is an **installed read-only adapter**, not a bypass of DomainHarness Kernel
or ADS Task CAS. The selected C binds one explicitly configured GitHub repository,
one version branch, and one exact Git SHA. Issue→version membership comes ONLY
from the Host-supplied pinned Task roster. There is no "scan all open Issues and
pretend every Issue belongs to the version" fallback.

Host environment (only the Host process, never browser or ADS report):

- `TALLY_V2_REPOSITORY=owner/repo`
- `TALLY_V2_VERSION_REF=version/v2`
- `TALLY_V2_VERSION_SHA=<exact 40-hex SHA from selected version branch>`
- `TALLY_GITHUB_READ_TOKEN=<read-only scoped credential>`
- `TALLY_V2_TASK_ISSUES_JSON=[101,102,...]` (optional for catalog-only; **required** for Task/DAG)
- `TALLY_V2_DH_PATH`, `TALLY_V2_WORK_ROOT` (local paths optional for read-only bootstrap).

Node>=24: install the existing `packages/tally-github-native-c` dependency
tree, and the `apps/tally-web` build dependencies. Run `node
web/v2/serve.mjs` on a trusted loopback Host; the printed path ends in
`/app`. No browser credentials are needed.

The bootstrap deliberately injects a rejecting legacy Agent runner; merely
starting the UI never authorizes execution or release. The read Host provides
`catalog`, `tasks`, `dag` only when exact source/branch/Task roster/native
dependency evidence is verified. No Agent, Report, metadata, QA, Workflow,
BUILD activation, LLM or config-write producer is fabricated; those menus
remain NOT_CONNECTED until independently qualified selected C/B/Host adapters
are provided. Missing or out-of-roster native blockers, stale branch SHA,
invalid Task state or missing credentials fail closed.

`connected:true` certifies only **source read qualification**, NOT a CAS
Claim, trusted Kernel admission, Task PASS or Release RQ. Release permissions
remain separately gated. See `docs/implementation/v2/UI_WEBFIRST_R1_ACCEPTANCE.md`.
