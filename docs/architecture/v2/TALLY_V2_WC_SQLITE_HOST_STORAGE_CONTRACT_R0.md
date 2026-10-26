# Tally v2 W-C — SQLite-first F40 dual-storage contract R0

**Owner:** [#89](https://github.com/kaicreator-mm/tally/issues/89), Product decision [#84](https://github.com/kaicreator-mm/tally/issues/84), L2 [#78](https://github.com/kaicreator-mm/tally/issues/78). **State:** SOURCE_CONTRACT_ONLY / PHYSICAL_TESTS_NOT_RUN / DH_RELEASE_R01_PENDING. SQLite is P0 mandatory; PostgreSQL is an optional future provider, **not** default CI/service prerequisite.

## Read-backed DomainHarness public Host seam (exact candidate)

DomainHarness version branch source at `v0.10.2@7c24f2ad0f75929e2dffbe256afb4ce9d42c49a7` (release candidate, not qualified R01). The actual public `@kaicreator/domain-harness-host-node` package `package.json` Git blob `72242fbe446dafc2ff08fff6da581d875766b0c2` declares Node `>=24.0.0`, version `0.10.2-rc.0`, **root import** from `./dist/index.js` and optional `better-sqlite3` peer `^12.0.0`; no magic inherited private DB dependency. Its actual published type root `packages/host-node/dist/index.d.ts` Git blob `2990cfbc4eb853085834bcef03f02f9bdbfa30c8` exports `CasDocumentStore`, `CasPutConditions`, `SqliteCasStoreOptions`, `openSqliteCasStore`, `SqliteCasStore`, `InMemoryCasStore`, `probeTrueCasStore`, `validateWriteCondition`, `HostBoundaryError`, `activateGovernedV10Build`, K-selected/deployment IDs. **These are root public exports** at candidate source; final R01 may modify exact artifact bytes, so re-pin before production.

Public README shape (NOT execution evidence):

```js
import { openSqliteCasStore, activateGovernedV10Build } from '@kaicreator/domain-harness-host-node';
const store = await openSqliteCasStore({path:'app.sqlite'});
const governed = await activateGovernedV10Build(builtApp,{store,deployment,decision,authorization});
const receipt = await governed.send({intentType,instanceKey,messageId,input});
governed.close();
```

Do **not** import BUILD from Host, import `src/**`/private `dist/**` paths, install obsolete Compiler v10 public exports, mirror K document key grammar, or build a second Tally state machine. For the clean Node24 consumer proof use the **sealed installable packages** and valid BUILD produced by qualified DH source rather than just README snippets.

## Two different persistence responsibilities, one SQLite transport default

| Responsibility | Sole logical owner | Store semantics | Disallowed shortcut |
|---|---|---|---|
| K app State, Admission, CAS, Journal/Effect, replay/UNKNOWN | selected DomainHarness K Primary governs; public Host persists | Real durable CAS document revisions, append/replay/effect reconciliation and restart | A Tally-owned second Journal/Effect/Kernel with its own authoritative mutable state |
| B-domain useful execution intent/derived UI projection | Tally Business B, selected K records admitted app facts | Source-qualified K snapshot/query facade; do not write protected K docs directly | Direct SQLite UPDATE used to override K Task acceptance/Release |
| GitHub Issue/PR/Review/CI/ADS Report source | GitHub actual native API + adopted ADS dialect | Native readback, permission and SHA/currentness, completeness across pages | Treating local cache/SQLite as GitHub fact or signed permission |
| GitHub-derived read model/cache | Tally C GitHub Reader + bounded cache service, transport via SQLite | separate SQLite namespace/tables/schema, repo+source SHA+comment revision+ADS dialect, TTL/currentness/coverage, rebuild safely | Stale/partial cache authorizes PR merge, Task PASS or Release READY |

**Host transport boundary:** If the real selected public Host uses one SQLite file for both concerns, retain distinct scoped namespaces/tables/transaction owners and never imply atomic cross-store transaction with a GitHub network write. If separate files are simpler and safer, prefer two file paths. GitHub external write uses K permit + C readback/reconciliation, not a SQLite transaction pretending to commit remote state.

## Minimum semantic operations (design, not invented DH symbols)

- **K/Host:** use *actual* `CasDocumentStore` and `openSqliteCasStore` public types/implementation; verify `get`/compare-write semantics, conflict, restart, lock, fault/UNKNOWN via the installed public ABI. Do not assert method signatures until physical TypeScript import/compilation pins the final candidate `.d.ts` and actual runtime.
- **Tally derived projection:** a project-local `observeGitHubSource(repo, ref, sourceSha, adsDialect, coverage, revision, payload)`, `queryBounded`, `invalidate`, `rebuild` **proposal** can be adjusted at L2. No new generic ORM or DB-wide dialect abstraction for P0. Enforce `repo + Issue + schema + SHA + revision` currentness, actual permission before any external native effect.
- **Generic failure vocabulary:** `CONFLICT`, `NOT_FOUND`, `UNAVAILABLE`, `READ_ONLY`, `PARTIAL`, `STALE`, `UNKNOWN_EFFECT`, `UNQUALIFIED_SOURCE`; error translation must not convert observed conflict into accepted success. Use actual DH typed errors where available; proposed names are not library exports.
- **SQLite physical profile:** WAL, prepared parameterized queries, bounded transactions/timeout, durable recovery, one active executing Task Attempt in P0; additional cross-process writer CAS negative must be real. File path isolation and backups under Host ownership; no accidental sensitive Trace export in GitHub cache.
- **Postgres P1:** no required service, migration framework, Postgres-specific lock or test in v2 P0. Only when a Postgres Host provider truly ships, prove same *behavioral* contract via physical Postgres tests and adapter-specific isolation. Legacy Go `postgres-integration` CI actual failures remain separately open [#83](https://github.com/kaicreator-mm/tally/issues/83), not waived.

## Negative and physical test mapping

| ID | Observation | Expected valid result |
|---|---|---|
| ST-N01 | two write requests with same `expectedRevision` | exactly one CAS winner; conflict typed; no double Effect |
| ST-N02 | app process crash after durable Journal but before remote GitHub receipt | restart/recover `UNKNOWN_EFFECT`; query GitHub before replay |
| ST-N03 | process restart after committed K state | correct replay and no fabricated double Task transition |
| ST-N04 | SQLite busy/lock timeout under two OS processes | bounded typed conflict/unavailable; no deadlock/pass by default |
| ST-N05 | Host file read-only/full disk/IO error | real FAIL/UNKNOWN; no silent in-memory fallback for mandatory durable mode |
| ST-N06 | schema migration interrupted | transaction rollback or recoverable known state; no destructive data loss |
| ST-N07 | GitHub report edited/corrected, cache has old revision | source cache STALE, refetch native; no Task acceptance from cached report |
| ST-N08 | GitHub 403/revoked repo grant but old cache complete | derived display may show stale with caveat; effects denied, not current/authorized |
| ST-N09 | first page cache success but more pages missing | PARTIAL/UNKNOWN; no 'all Issues complete' assertion |
| ST-N10 | old ADS v4/v3 report treated as current v5 | reject unsupported dialect or explicitly historical read-only |
| ST-N11 | K CAS store file and derived cache share path/key mistakenly | detect namespace collision, fail closed |
| ST-N12 | optional better-sqlite3 peer absent for required Rich SQLite | deterministic unavailable, do not silently claim SQLite P0 PASS |
| ST-N13 | Tiny selected without Host-node SQLite | Tiny is supported Tiny behavior, **not** Tally Rich SQLite E2E PASS |
| ST-N14 | Tally attempts to direct-write K Journal via cache SQL | wrong-store write forbidden; K owner must record effect |
| ST-N15 | task-meta publisher offline and no GitHub report | actual task allowed by K, derived metrics UNKNOWN, no zero |
| ST-N16 | read-only operator wants project list without Codex/SQLite-Host active Task | UI safely reads scoped source facts without false production status |

## Parallel scope and admissibility

Source-only exact ABI mapping and failure fixtures can be independently reviewed NOW; W-A [#87](https://github.com/kaicreator-mm/tally/issues/87) must verify installed public package closure/real Node24 and selected SQLite Host after DomainHarness R01 #1331 current candidate qualification. W-B [#88](https://github.com/kaicreator-mm/tally/issues/88) independently owns GitHub-native Report readers, not database authority. W-D #90 owns process/OS credential and held-out isolation. The production SQLite Host Adapter Task requires actual accepted Product/L2/ADS current pin and valid L3 + native dependency/Claim source before any implementation write.

**Review checklist:** check public export names against latest exact DH SHA, verify K-vs-cache separation, no production lease/Effect lie, SQLite-first floor, correct optional Postgres semantics, negative completeness, REAL_HOST_TESTS_NOT_RUN.
