# Tally v2 W-Cx — real public SQLite CAS experiment (not production)

**GitHub owner:** [#100](https://github.com/kaicreator-mm/tally/issues/100), related W-C [#89](https://github.com/kaicreator-mm/tally/issues/89) and W-A [#87](https://github.com/kaicreator-mm/tally/issues/87). No Tally Kernel, Journal or business runtime is implemented here. This tests **only** the publicly exported Host-node physical `CasDocumentStore` SQLite adapter using actual installed code.

**Source pin:** DomainHarness v0.10.2 R01 candidate at `11ab7954f547fde44c003e2b227d672dc8af0a79` (not final RQ). Exact public type source: `packages/host-node/dist/sqlite-cas.d.ts` blob `daefc613c3644fe46ff1b1c162fec7ce5df62ebd` and `sqlite-adapter.d.ts` blob `95c08bb3052f6d914af6e30905bc6794281419b6`, confirmed GitHub readback. Exports: `openSqliteCasStore({path,busyTimeoutMs})`, `probeTrueCasStore`, typed `HOST_PUT_CONDITION_REQUIRED`, `HOST_KEY_EXISTS`, `HOST_CAS_CONFLICT`. Runtime package manifest `0.10.2-rc.0`; optional better-sqlite3 `^12` peer. Real GitHub source pin must be checked again if upstream version advances.

## Scope of real test

Creates a **temporary file-backed SQLite database**, performs create/CAS/reject-unconditional/reject-duplicate/reject-stale, races two independent **connections** to the same file (one winner), closes/reopens and checks revision/value preservation. Also runs the public `probeTrueCasStore` on the actual physical store. The temporary directory is removed after the test. The output contains OS/Node and a test revision but no secrets, GitHub API writes, Task acceptance or Hidden fixtures.

**Important exclusions:** two connections are not a cross-**process** locking witness; Host/CAS success alone is not K Journal/Effect/UNKNOWN, BUILD+selected K/S/B/C installation, real Codex, hidden verifier or Release PASS. Those are separate tests. No fabricated test result: `EXECUTION=NOT_RUN` until the operator uploads the actual test log/EXIT and package SHA256 digest.

## Reproducible CLI (Ubuntu or Git Bash on Windows; Node24)

Two genuine clean checkouts are required: (A) DomainHarness `11ab7954...` and (B) this Tally experiment PR exact SHA. Below use environment variables `DH` and `TALLY` pointing to those worktrees; **do not** perform production/native GitHub writes. Build/pack only the selected public packages; don't install unavailable registry `0.10.2-rc.0` by name.

```bash
set -euo pipefail
test "$(git -C "$DH" rev-parse HEAD)" = '11ab7954f547fde44c003e2b227d672dc8af0a79'
node --version  # must be v24.x
cd "$DH"
npm ci
npm run build
PACK="$(mktemp -d)"
npm pack --pack-destination "$PACK" -w @kaicreator/domain-harness-contracts
npm pack --pack-destination "$PACK" -w @kaicreator/domain-harness-microkernel
npm pack --pack-destination "$PACK" -w @kaicreator/domain-harness-host-node
sha256sum "$PACK"/*.tgz
CONSUMER="$(mktemp -d)"
cp "$TALLY/experiments/v2/host-sqlite-cas/host-cas.test.mjs" "$CONSUMER/"
cp "$TALLY/experiments/v2/host-sqlite-cas/package.json" "$CONSUMER/"
cd "$CONSUMER"
npm install --save-exact "$PACK"/kaicreator-domain-harness-contracts-*.tgz "$PACK"/kaicreator-domain-harness-microkernel-*.tgz "$PACK"/kaicreator-domain-harness-host-node-*.tgz 'better-sqlite3@^12'
node --test host-cas.test.mjs
echo "TEST_EXIT=$?"
```

**Evidence:** record (1) exact git SHAs/trees for both repos, (2) `node --version` and `npm --version`, (3) `npm ci` / `npm run build` / tarball install and test literal exit codes, (4) SHA256 tarball digests, (5) real node test summary and whether native better-sqlite3 actually loaded. Run on a separate clean consumer outside repository; do not publish an unqualified npm registry package or commit local `node_modules`.

**Failure dispositions:** `HOST_STORE_DRIVER_UNAVAILABLE` indicates optional peer missing/ABI install issue; CAS conflict code mismatches or second-writer success are physical FAIL, not a waiver. Node22 or unavailable install must be `ENV_BLOCKED`, never PASS; no switch to InMemoryCasStore to fake SQLite. Old DomainHarness T09 packed-consumer proof can be reused for identical package bytes but cannot be substituted for a Tally-specific exact-HEAD test claim.

**Workflow:** source-only experimental PR independent Review → optional real Node24 CLI → exact evidence writeback to #100 → re-pin to DH final RELEASE_READY when it exists → independent source+runtime suitability before production adapter admission. Version/main integration remains HOLD.
