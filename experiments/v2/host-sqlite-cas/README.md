# Tally v2 W-Cx — real public SQLite CAS experiment (not production)

**GitHub owner:** [#100](https://github.com/kaicreator-mm/tally/issues/100), related W-C [#89](https://github.com/kaicreator-mm/tally/issues/89) and W-A [#87](https://github.com/kaicreator-mm/tally/issues/87). No Tally Kernel, Journal or business runtime is implemented here. This tests **only** the publicly exported Host-node physical `CasDocumentStore` SQLite adapter using actual installed code.

**Source pin:** DomainHarness v0.10.2 R01 candidate at `11ab7954f547fde44c003e2b227d672dc8af0a79` (not final RQ). Exact public type source: `packages/host-node/dist/sqlite-cas.d.ts` blob `daefc613c3644fe46ff1b1c162fec7ce5df62ebd` and `sqlite-adapter.d.ts` blob `95c08bb3052f6d914af6e30905bc6794281419b6`, confirmed GitHub readback. Exports: `openSqliteCasStore({path,busyTimeoutMs})`, `probeTrueCasStore`, typed `HOST_PUT_CONDITION_REQUIRED`, `HOST_KEY_EXISTS`, `HOST_CAS_CONFLICT`. Runtime package manifest `0.10.2-rc.0`; optional better-sqlite3 `^12` peer. Real GitHub source pin must be checked again if upstream version advances.

## Scope of real test

Creates a **temporary file-backed SQLite database**, performs create/CAS/reject-unconditional/reject-duplicate/reject-stale, races two independent **connections** to the same file (one winner), closes/reopens and checks revision/value preservation. Also runs the public `probeTrueCasStore` on the actual physical store. The temporary directory is removed after the test. The output contains OS/Node and a test revision but no secrets, GitHub API writes, Task acceptance or Hidden fixtures.

**Coverage boundary:** the default test's two connections are **not** a cross-process locking witness. The **optional** `WCX_WAL_STRESS=1` path uses four actual independent Node child **processes** per fresh SQLite database to exercise public Host WAL initialization/concurrent open; it does **not** prove cross-process same-key CAS, K Journal/Effect/UNKNOWN, selected Domain App/BUILD, real Codex, Hidden verifier or Release PASS. It is **not** the separate legacy `NodeSqliteRuntimeStore` regression suite for [DomainHarness #1385](https://github.com/kaicreator-mm/domain-harness/issues/1385). Both `EXECUTION=NOT_RUN` and `STRESS=NOT_RUN` remain until literal Node24 results, exit and tarball SHA256 receipts exist.

## Reproducible CLI (Ubuntu or Git Bash on Windows; Node24)

Use the **versioned `run-clean.sh` in this PR** rather than copying/adapting a shell glob by hand. It refuses the wrong DH candidate SHA and non-Node24, normalizes both checkout paths, packs the three public dependency tarballs, checks SHA256, installs them with the optional native `better-sqlite3` peer into an independent temporary consumer, runs the actual test, and **always prints `WCX_EXPERIMENT_EXIT=<actual process exit>`**, including on failure. The native Node test suite also enforces the Node24 major version and source-typed CAS denials.

Two *genuine* clean worktrees are required: DomainHarness `11ab7954f547fde44c003e2b227d672dc8af0a79` (still a release candidate), and the latest reviewed Tally PR #101 HEAD. On Ubuntu or Git Bash:

```bash
export DH=/absolute/path/to/clean/domain-harness
export TALLY=/absolute/path/to/clean/tally-pr101
git -C "$DH" status --porcelain
git -C "$TALLY" rev-parse HEAD
git -C "$TALLY" status --porcelain
KEEP_WCX_WORK=1 bash "$TALLY/experiments/v2/host-sqlite-cas/run-clean.sh" 2>&1 | tee "$HOME/tally-wcx-evidence.log"
test ${PIPESTATUS[0]} -eq 0
```

**Important:** `KEEP_WCX_WORK=1` retains temporary consumer+packed artifacts for independent digest/lockfile inspection; remove that experimental directory when evidence has been captured. A terminal log containing text `PUBLIC_HOST_CAS_SMOKE=PASS` without Node test process **EXIT 0** is insufficient. If the test fails, both `WCX_EXPERIMENT_EXIT` and the calling shell's `PIPESTATUS[0]` preserve failure; do not describe it as PASS. All package tarballs are pinned to actual source and SHA256, with no registry assumption for private `0.10.2-rc.0` package names.

### Optional cross-process WAL initialization stress

On the **same pinned R01 source checkout** and separate installed consumer, enable the separate test:

```bash
WCX_WAL_STRESS=1 WCX_WAL_ROUNDS=15 KEEP_WCX_WORK=1 bash "$TALLY/experiments/v2/host-sqlite-cas/run-clean.sh" 2>&1 | tee "$HOME/tally-wcx-wal-evidence.log"
test ${PIPESTATUS[0]} -eq 0
```

The optional path creates **15 fresh SQLite databases**, spawning **4 independent Node child processes** per database. It captures each failing worker's exit and a bounded stderr excerpt without needing GitHub credentials. Any failed child causes the test and parent runner to exit nonzero; no retry masks a failure. Each child writes a distinct key: this covers concurrent *WAL initialization*, not cross-process same-key CAS. WCX_WAL_ROUNDS is bounded 1–30 (default 5 if enabled).

This determines whether the public Tally-selected openSqliteCasStore interface shares startup exposure with upstream [DH #1385](https://github.com/kaicreator-mm/domain-harness/issues/1385), which specifically concerns the **different legacy NodeSqliteRuntimeStore** constructor. Even a clean stress result cannot close #1385, qualify DomainHarness R01, replace its private Hidden 7/7 evidence, or act as Tally Release PASS. Report observed discrepancies on #100 with exact source/OS/Node/logs. Run the normal CAS smoke first.

**Evidence:** record (1) exact git SHAs/trees for both repos, (2) `node --version` and `npm --version`, (3) `npm ci` / `npm run build` / tarball install and test literal exit codes, (4) SHA256 tarball digests, (5) real node test summary and whether native better-sqlite3 actually loaded. Run on a separate clean consumer outside repository; do not publish an unqualified npm registry package or commit local `node_modules`.

**Failure dispositions:** `HOST_STORE_DRIVER_UNAVAILABLE` indicates optional peer missing/ABI install issue; CAS conflict code mismatches or second-writer success are physical FAIL, not a waiver. Node22 or unavailable install must be `ENV_BLOCKED`, never PASS; no switch to InMemoryCasStore to fake SQLite. Old DomainHarness T09 packed-consumer proof can be reused for identical package bytes but cannot be substituted for a Tally-specific exact-HEAD test claim.

**Workflow:** source-only experimental PR independent Review → optional real Node24 CLI → exact evidence writeback to #100 → re-pin to DH final RELEASE_READY when it exists → independent source+runtime suitability before production adapter admission. Version/main integration remains HOLD.
