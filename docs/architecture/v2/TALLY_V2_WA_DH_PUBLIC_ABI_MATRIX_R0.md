# Tally v2 W-A — selected DomainHarness v0.10.2 public Package/BUILD/Host ABI matrix R0

**Owner:** [#87](https://github.com/kaicreator-mm/tally/issues/87). **Source identity for this inspection:** DomainHarness **version-branch** `v0.10.2@7c24f2ad0f75929e2dffbe256afb4ce9d42c49a7`, **not** `main` or v0.10.2 Release/Tag. DH [R01 #1331](https://github.com/kaicreator-mm/domain-harness/issues/1331) is still OPEN: current R01 final Full/Hidden/Critical/Packaging/Release NOT_QUALIFIED. This matrix is GitHub exact-blob source inspection, **not physical Node24 tests**.

## Actual external public package closure

| Package | Exact manifest blob at pinned version source | Public export | Uses |
|---|---|---|---|
| `@kaicreator/domain-harness-contracts` `0.10.2-rc.0` | `b67314e1604ee6c07605b0b2bea26d7aca99aabb` | package root `.` → `dist/index.js`, `.d.ts` | Common typed payloads, digests, errors; no runtime K |
| `@kaicreator/domain-harness-build` `0.10.2-rc.0` | `ef9d71dd789206da076db7598f0abd3e60aa86cc` at `tooling/build/package.json` | root `.` → `dist/index.js`, `.d.ts` | **Only external Producer process** builds sealed selected K/S/B/C plain-data App |
| `@kaicreator/domain-harness-microkernel` `0.10.2-rc.0` | `b36bf56f1ea87ba7b116a17acb8861317fd8a81e` | root `.` and `./host-seam` | Runtime `load(app, options)` yields selected `{send,query,subscribe,stop}` facade; `./host-seam` for trusted Host only |
| `@kaicreator/domain-harness-host-node` `0.10.2-rc.0` | `72242fbe446dafc2ff08fff6da581d875766b0c2` | root `.` only | Rich trusted Host, `openSqliteCasStore`, governed activation, scoped memory/durable controls; optional `better-sqlite3` peer |

**Physical package reality caveat:** package `version=0.10.2-rc.0`, Node engine `>=24.0.0` on these four manifests; registry publication is not inferred by manifest. Actual external Consumer must install **exact tarballs with captured SHA256 and matching lockfile**, not float npm latest or import source `src/**` / unexported `dist/**`/private Compiler. Runtime closure excludes BUILD/Compiler and unselected packages; producer closure contains BUILD+Contracts+selected payloads, no runtime selection changes.

## Exact root exported APIs (source inspection only)

- Microkernel public type root Git blob `78471f3de149b4582da28260a67507956d89d6dc`: `load`, `DomainApp`, `LoadAppOptions`, `V10BuiltApp`, `V10DocStore`, typed `V10Error`, `V10BuildError`, `V10ActivationError`, `V10ExternalEffectError`, plus `V10PortDeniedError`, `V10MultiBQueryError`. The core never imports BUILD, even type-only.
- Contracts type root `8ded97571ca352362883de872a4c1bb014199336` exports digest, known error codes, identity/json types, BUILD types and activation types.
- Host-node public type root Git blob `2990cfbc4eb853085834bcef03f02f9bdbfa30c8`: `CasDocumentStore`, `CasPutConditions`, `SqliteCasStoreOptions`, `openSqliteCasStore`, `SqliteCasStore`, `probeTrueCasStore`, `InMemoryCasStore`, `activateGovernedV10Build`, `V10DeploymentError` and durable coordination types. Optional peer `better-sqlite3 ^12`, dynamic import by Host, not a hard dependency of the portable K or Tiny.
- Legacy `@kaicreator/domain-harness-compiler` manifest is `private:true` / version `0.2.0`, and old compiler `/v10build` public import path is expressly retired. Tally v2 is **Domain Package-first** from selected public BUILD, never a standalone TS service wrapped afterward.

## Boundaries and proposed acceptance negative matrix

| ID | Scope | Current source-aligned expectation |
|---|---|---|
| DH-A01 | External BUILD producer clean-root only | Real packed BUILD+Contracts, sealed selected K/S/B/C; output trusted plain-data App; no production effects |
| DH-A02 | Tiny runtime installed closure | Microkernel+Contracts+selected packages only, no BUILD/Compiler/Host-node/better-sqlite3 |
| DH-A03 | Tiny runtime events | real send/query/replay/changed-input typed conflict/subscribe-unsubscribe-stop; redacted bounded event |
| DH-A04 | Rich Host root import | `openSqliteCasStore` real file-backed CAS via optional peer present; missing peer denial typed, not memory fallback |
| DH-A05 | Rich file restart/UNKNOWN | durable CAS/recovery/Effect reconciliation verified by actual Node24 clean-consumer receipt |
| DH-A06 | External Producer→Tiny/Rich | third-party sealed App accepted by PUBLIC build + runtime; no `src/**` imports or legacy Compiler public path |
| DH-A07 | App activation authority | one selected K owns State/Admission/Journal/Effect, Host only physical persistence; never second Tally K |
| DH-A08 | Multi-B | explicit authorized Business selection on query/send; cross-B read/write denied, typed |
| DH-A09 | Build/runtime import closure | no runtime link to BUILD/Compiler, even type-only; no deep unexported path or private registry tarball |
| DH-A10 | Scoped observed evidence | stop/revoke/subscribe and no K key/receipt secrets in public events |
| DH-A11 | Node24 toolchain | actual `node --version` 24.x; exact lockfile, tarball hashes, CLI command + exit |
| DH-A12 | Release vs scoped T09 | T09 evidence may be reused byte-for-byte if unchanged, but R01 whole-version gates remain independent NOT_RUN until actual receipt |

## Workfront gate graph and target local task

**Source verification now (Web-first)**: exact package manifests and declared public exports were inspected in GitHub at the pinned version HEAD. The v0.10.2 final-Release source may advance after this pin, requiring a new Source Delta review; do not silently upgrade.

**Minimal optional experimental Local task** (NOT production admission, no GitHub writes): on a clean Node24 directory pack the 4 public packages from exact DH candidate source, save tarball SHA256, install selected dependencies into three isolated producer/Tiny/Rich roots (separate lockfiles), execute Tiny and Rich positive+negative tests, verify `node_modules` graph/import closure, record literal command/EXIT and evidence SHA. Prefer reusing independently qualified DH T09 Node24 receipts and perform only missing Tally-relevant proof. If `R01` has not qualified, label `EXPERIMENTAL_PUBLIC_ABI_SPIKE`; never Release PASS.

**Later implementation prerequisite:** accepted Tally v2 Product and ADS v4.18 exact pin [#85](https://github.com/kaicreator-mm/tally/pull/85), independent L2 physical conformance/review, DomainHarness R01 qualified source, actual native Task DAG Issues+dependencies+Claim and per-concern L3. Public SDK imports only; SQLite-first [#89](https://github.com/kaicreator-mm/tally/issues/89) must reuse Host public store, not fork K.

**Reviewer output:** P0/P1/P2, factual name/entrypoint/digest accuracy at exact DH source, local-tests NOT_RUN vs reusable T09 evidence, source ABI matrix READY|BLOCK, release/implementation admission separately. One concern/PR, no merge.
