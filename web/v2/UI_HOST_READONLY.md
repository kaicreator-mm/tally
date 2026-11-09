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
tree, the `packages/tally-engineering-business` runtime dependency (`ajv@8.17.1`), and the `apps/tally-web` build dependencies. The engineering Business package must be installed explicitly; installing the Native C package and web app alone does not provide its Ajv dependency. Run `node
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

## Optional ADS v5 report-only source

The Host may additionally configure **both** variables; neither may be supplied
by browser query, dynamic form, or Agent output:

- `TALLY_V2_ADS_PUBLISHER_BINDINGS_JSON={"ads/task-meta":["trusted-publisher-login"]}`
  — Host-selected binding of publisher namespaces to the actual GitHub
  comment transport author; no publisher is inferred from report text.
- `TALLY_V2_ADS_ACCEPTED_COMMENTS_JSON={"210":[123456789]}`
  — Host-selected, positive native GitHub Issue comment ID witness per Task
  in the bounded roster. A missing mapping does **not** mean no reports exist.

This composes `/api/v2/metadata` using the already implemented read-only
GitHub Native C ADS v5 schema/source/publisher/JCS/correction checks. Only
CURRENT_OBSERVED reports with exact comment/source match are returned, and
all are **SELF_REPORTED** until an independent Reviewer explicitly witnesses
them. No Task PASS, Claim, release permission, private task-meta or Agent
execution grant is implied.

Telemetry, Task Attempt, lineage and learning feeds are **not** sourced by
this adapter: the UI marks their coverage UNKNOWN, even if report rows are
visible. The frozen Task roster coverage remains UNVERIFIED. A verified
Publisher ID and comment ID are not a guarantee of complete report coverage.
Omit the two variables to keep metadata NOT_CONNECTED, not a fake empty set.

To validate in a real Host environment, use Node >=24 and the existing
`web/v2/ui-report-host.test.mjs` and
`web/v2/ui-report-host-http.test.mjs` before supplying actual publisher
bindings. The application must not log or persist credential values.

## Optional ADS v3 Claim candidate Audit (read-only)

`TALLY_V2_AUDIT_CLAIM_CANDIDATES=read-only` enables the selected GitHub C
WorkItem + ADS Claim event **source pair** over the configured Task roster.
Requires the same pinned version SHA and Host-only GitHub credential.
This is a potentially larger paginated scan; omit the flag to leave the
Audit source NOT_CONNECTED. Invalid flag value and missing Task roster fail
at Host startup, never silently enable a scanner.

The Governance/Audit menu displays one source observation per selected Task,
with an optional count of `DISPATCH_CLAIMED` comments. A valid ADS v3 event
is a **candidate**, not a durable accepted CAS Claim, Agent Attempt, Reviewer
PASS, Kernel permit or Task closure. The UI does not see raw dispatch IDs,
session refs, GitHub comment bodies, model credentials or operator details.
If any selected Issue cannot be read or fails the source-pair witness, the
whole audit source returns NOT_CONNECTED; no partial positive ledger.

Native Claim candidate comments under `ads/github/v3` are distinct from
ADS v5 `kind: report` comments. Neither type substitutes for the other.
Run `web/v2/ui-audit-host.test.mjs` and
`web/v2/ui-audit-host-http.test.mjs` with Node >=24 before enabling
production audit reads.

## Optional selected GitHub C Requirement/Story roster

The Host can opt into the read-only Requirements view using
`TALLY_V2_REQUIREMENT_ISSUES_JSON`, a bounded JSON list (up to 16 entries)
of `{"number":77,"kind":"requirement","parent":null}` and optional nested
`kind: "epic" | "story"` records referring to other list members.
No browser is allowed to configure this list. Cycles, foreign parents,
duplicate IDs and invalid category names are refused.

The already-selected native Octokit reads each Issue via the existing
`packages/tally-github-c/src/project-intake.mjs` source port and compares
the exact Issue title, number and canonical URL with a second GitHub read,
with the selected `version/v2` branch verified both before and after.
Only source-qualified Issue titles become connected. The category and parent
pointers are **Host-declared metadata**, not canonical frozen Product or L2
source; they render as `REQUIREMENT_STRUCTURE=HOST_DECLARED_UNVERIFIED`.
Product Revision, acceptance references, L2 decisions and Freeze remain
`null`/`UNKNOWN`, never inferred from the Issue title or closed state.
No effect/Approval/PRD/Owner grant is created.

If the roster environment is absent, Requirements stay NOT_CONNECTED.
The existing `ui-requirements-host-http.test.mjs`, native C tests and
source-level scope tests must pass on Node>=24.

## Optional exact-SHA GitHub CI status for QA (reported only)

`TALLY_V2_QA_STATUS_CONTEXT=ci/woodpecker/push/v2-alm-workspace`
(or an explicitly selected `ci/woodpecker/pr/...` context) enables
a read-only GitHub Native C commit-status observation on the pinned version SHA.
All returned status pages must be complete and match the version branch;
only a literal `https://ci.kaicreator.com/repos/<id>/pipeline/<id>/<id>`
evidence URL is eligible. This projects exactly one
`fullRegression` `REPORTED` record, **not** a full independent QA
qualification. HiddenValidation, criticalJourneys, installedAcceptance,
Owner/RQ remain UNKNOWN or NOT_RUN. A missing CI status is a verified
absence in this scope, not a passed test.

Run `web/v2/ui-qa-host-http.test.mjs` and the Native C status tests on
Node>=24. Only a real selected Host may set this context; it is not
computed from or accepted from browser input.

## Source-qualified GitHub-only Settings inventory

The read-only Host supplies `/api/v2/settings` from two consecutive,
exact-version Native C catalog readbacks. Only the selected repository,
`version/v2` reference and current pinned SHA are qualified GitHub facts.
The response marks `SETTINGS_COVERAGE=PARTIAL`. Separate coverage for
Agent providers, Tally LLM, Woodpecker webhook, OAuth scopes, storage,
security and deployment is `NOT_CONNECTED`, **not** "no providers", "LLM
disabled", or "nothing deployed". UI sections lacking a trusted selected
configuration source render an explicit unconnected message instead of
rendering empty placeholder arrays. No token, credential target, Host
principal or dynamic execution authority is disclosed to browser; settings
writes remain disabled. Tests: `web/v2/ui-settings-host.test.mjs` and
`web/v2/ui-settings-host-http.test.mjs`.

## Optional GitHub-native blocked-Task workbench (partial source)

The Host-only `TALLY_V2_WORKBENCH_BLOCKERS=read-only` option composes
`web/v2/ui-workbench-host.mjs` with the selected GitHub Native C
`version()` reader. It requires an explicit `TALLY_V2_TASK_ISSUES_JSON`
roster, exact version SHA and current catalog double readback.
The Native C Task projection preserves **openBlockerCount** and
**dependencyState** from independently verified GitHub blocked_by edges;
only Tasks positively observed with open native blockers produce advisory
`BLOCKER` rows. They are UNASSIGNED and do not authorize any operation.
Unverified Task roster membership, cross-repo edges, stale SHA, unaccepted
Claim states or inconsistent dependency counts fail closed.

The Workbench emits `queueCoverage=PARTIAL_UNVERIFIED_ROSTER` and
`approvalCoverage=NOT_CONNECTED`; a zero blocked-row count is only a
bounded-source observation, never proof of a complete empty human inbox
or zero approvals. The UI visibly warns about limited Task coverage.
Without the Host-only opt-in, Workbench stays NOT_CONNECTED.
Run `web/v2/ui-workbench-host.test.mjs` and
`web/v2/ui-workbench-host-http.test.mjs` with Node >=24.
