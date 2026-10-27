# Tally v2 — DOMAIN_PACKAGE_READY 外部验收门（R0）

**状态：** PREPARATION_ONLY / DH_0_10_2_CONDITIONAL_RELEASED / TALLY_RUNTIME_PRODUCTION_NOT_ADMITTED。  
本清单是 Tally 的**外部依赖准入合同**，并非 DomainHarness normative API、发布声明、Tally 代码或真实 Node24 PASS。对现有平台不新增功能需求，先验证其公开实现是否满足目标再提出真正缺口。

## A. 门禁定义

| Gate | 条件及可复验来源 | 验证方式（未来） | 当前 |
|---|---|---|---|
| DP-01 Contracts | 第三方可消费的独立安装 Contracts，唯一 Error/identity 实例，不依赖私有 core/compiler 路径 | npm pack/install + Node24 esm/require identity，类型/ABI negatives | PENDING：DH [T00 #1319](https://github.com/kaicreator-mm/domain-harness/issues/1319) |
| DP-02 Microkernel | 独立安装的 Microkernel public Facade 加载选中 BUILD，不自有 K business Effect 机制 | outside-repo Application 启动+合法 K permit/deny | PENDING：[T04 #1324](https://github.com/kaicreator-mm/domain-harness/issues/1324) |
| DP-03 K/S/Business package closure | 生产 K Primary、可选 S、Business、Capability Package 的独立源/manifest/ABI 和选中闭包，经干净外部 producer 编译 | clean npm pack/install，chosen digest，禁止 deep import/private compiler | PENDING：[T09 #1330](https://github.com/kaicreator-mm/domain-harness/issues/1330) |
| DP-04 Host Node | Node24 Host 真实 SQLite CAS/持久化/恢复与 selected K 接线 | process restart, UNKNOWN, CAS conflict, readback | PENDING：[T03 #1325](https://github.com/kaicreator-mm/domain-harness/issues/1325) |
| DP-05 BUILD boundary | BUILD 工具与 Runtime Package 分离；外部第三方 Business/Capability Manifest 被批准和选择 | external producer + tiny runtime graph excludes compiler, fixed pin | PENDING：[T05 #1323](https://github.com/kaicreator-mm/domain-harness/issues/1323), [T09](https://github.com/kaicreator-mm/domain-harness/issues/1330) |
| DP-06 C Local admission | 一个实际外部 Local Tool Effect 由 K Permit，ACK 丢失进入 UNKNOWN，恢复不重复动作 | external C fixture + controlled lost ACK negative | PENDING：Tally #62 + DH K/Host |
| DP-07 Event lifecycle | 如 UI/Agent 需要的 public subscribe/stop，必须符合 governed active currentness/redaction | selected public facade event smoke + cancellation stale check | PENDING：[T06 #1326](https://github.com/kaicreator-mm/domain-harness/issues/1326) |
| DP-08 Qualified source | 使用唯一 exact SHA / pack digest / independent review + Platform final RQ | check version RQ exact source, no stale PR Gate | DH R01 已 CLOSED/CONDITIONAL，**Tally 独立项目资格仍待验**：[最终回执](https://github.com/kaicreator-mm/domain-harness/issues/1331#issuecomment-6416079327) |
| DP-09 Tally Product/L2 | Tally Product frozen、semantic+physical L2 independent approved、合法 ADS exact adopted dialect | L1/L2 reviewer comments and version pins | OPEN：Tally #76/#78 |
| DP-10 Tally runtime evidence | 第一个安装的 Tally Engineering B/C Domain Package 和真实 Local Task 及 GitHub/Verifier 闭环 | qualified Domain App selected package, local only, telemetry killed negative | NOT_RUN：Tally #62 |

**分阶段含义：**
- `DOMAIN_PACKAGE_READY_DESIGN`：公共 Contracts/Manifest/Host/BUILD 设计有固定版本文档可审查，允许准备 L2，**不等于可以实施**。
- `DOMAIN_PACKAGE_READY_EXPERIMENTAL`：经本地真实安装的 public RC/producer，可能允许独立批准的隔离实验；不转化为正式 v2 生产实现或 RQ。
- `DOMAIN_PACKAGE_READY_PRODUCTION`：DP-01…DP-09 均有有效的已审查证据和项目授权，方可准入真实 Tally Production B/C/UI 工作。DP-10 属于后续实现验收，不是生产代码开始的前置循环依赖。
- 完整 Tally Release PASS 要包括 DP-10 加 Tally 自身版本 Full Regression/Hidden/CJ/packaging/RQ，不能用 DH RQ 代替。

## B. 当前依赖路径：不是时间预测

DH [A2 #1341](https://github.com/kaicreator-mm/domain-harness/issues/1341) 的历史 R1 曾给 early T00 write-set `BLOCK P1=1`，但 [PR #1342 独立 R2](https://github.com/kaicreator-mm/domain-harness/pull/1342#issuecomment-6381135926) 已在精确 HEAD `77f2f3fcb3ec964a474fa6770617573deaaadd64` 给出 **PASS P0/P1/P2=0/0/0、A2_FREEZE_READY=YES**；PR #1342 已合并，merge commit `cfa3beddf79996a624b6b265e6f1034e8998e30f`。这证明 **A2 设计差异**，不是 Node24 物理安装、T00/T07a3 实测或 Public Package 发布资格。上述 T00/T04/T05/T03/T06/T09 为 v0.10.2 历史 Task；**当前**版本已在 [R01 终结回执](https://github.com/kaicreator-mm/domain-harness/issues/1331#issuecomment-6416079327)以一项明确审核的 OS Full 例外 **CONDITIONAL** 完成发布：精确 tag `cee14399d9d0af4e20e34b9111861b16a465b690`、main 同树、Hidden 7/7 PASS；Full 原始 EXIT1（3,979/3,986 workspace PASS），当前 root 测试 NOT_RUN。过去 A2 的设计 PASS 并不单独等于 Tally 的 Host/Domain Package 可交付；Tally version/v2 #101/#112/#113/#116/#118 已提供独立限定的真实公开 SDK/Host/Reader 验证，仍不构成全部 DP-01…DP-10 或 Tally Release PASS。

只在证据可验证时关闭该条；旧的 draft/PR HEAD 改变应重新审查。平台内部 ABI 如发生允许的 v0.10.2 修订，Tally 语义 L2 可保持稳定，但物理导入/manifest 必须重新 pin 后才能冻结。

## C. Tally 隔离实验最小接受条件（若用户另外批准）

1. 必须消耗独立第三方生产者构建的**实际 public** DH artifact，不得通过 ../packages 内部源码/私有 alias 获得假成功。
2. 清洁 Node24 沙盒使用 exact package tarballs/digest，禁止依赖本地 monorepo 隐式环境。
3. 只用一个最小工程 Business/Capability 包和一个无生产 GitHub 权限的工具效果实验；不发布真实合并/Release。
4. 负例至少包括 unauthorized K effect、UNKNOWN ACK、duplicate effect、stale source、Package update older pin、package closure private-import rejection。
5. 结果只能标 `EXPERIMENTAL_PASSED_ON_PIN`，不能声称 v2 功能实现通过、ADS Release PASS 或 Product/L2 Freeze。

**当前实验：NOT_RUN。当前计划：先 Product 和语义 L2 审查，不立即开实验。**

## D. Tally 上游要求的范围界限

如果真实公共包安装证明 DP-01…DP-08 本身存在缺口：按具体 exact SHA/失败日志明确 `TRUE_DOMAIN_HARNESS_GAP`，在 DH 新增独立 Issue 并依据其版本门禁处理；如果只缺具体 Tally 工程规则、C GitHub/Local 行为、UI、源缓存，则属于 **Tally 自己的 B/C/App 实现**，不要推给 DH 微内核。未来 runx 是可选 Host/Capability，不是 DomainHarness 核心 SDK 责任。

**结论：**本门禁可以提前作为产品/架构验收材料固化；未获取 DP-01…DP-09 证据前，Tally 正式 Domain Package 实施状态为 `NOT_ADMITTED`。

## E. Version-current Product and ADS activation prerequisites (source-only)

- Parent Product PR #77 exact synced source `0d1f1a2e15581e300500bada6da54b2c9bcef75f`, P0×17, Fxx×47, Vxx×67, including Builder GitHub credential deny and Local separate OS-principal Hidden verifier with escalation-negative fixtures.
- **ADS target is now v4.18.0** (independently [RQ READY on ADS #1327](https://github.com/kaicreator-mm/ai-development-standard/issues/1327#issuecomment-6401840670), main exact `2c395c55569fe2edf3f4135f12d3d069e526d0b7`), not historical v4.17. Tally [prospective PR #85](https://github.com/kaicreator-mm/tally/pull/85) has an old-H actual strict Windows Python3.14.7 `EXIT=0`; [CI scope #83](https://github.com/kaicreator-mm/tally/issues/83#issuecomment-6408620557) independently DECIDED for docs-only, but current-source independent adoption and Freeze are still pending. The earlier `RQ READY`/`CONDITIONAL` tension for unmerged v4.17 #81 remains an old, separate issue, not today's standard selection.
- A distinct independent Product Reviewer and eligible native GitHub Product Freeze record must precede admitted **executable** implementation Task DAG and qualified L2 Freeze. DomainHarness v0.10.2 T09 #1330 与 [R01 #1331 最终回执](https://github.com/kaicreator-mm/domain-harness/issues/1331#issuecomment-6416079327)已结束：正式 tag/main 完成，R01 为 **CONDITIONAL 而非全量测试 PASS**；真实 Hidden 7/7 PASS，Full EXIT1，root NOT_RUN，单次例外不得移作 Tally 例外。Tally 已合并 #101/#112/#113/#116/#118 等受限源码和 Node24 Host/BUILD/SQLite/Reader 实证，#120 真实集成还需最新 exact-SHA 独立合并审核；不能把局部阶段 CI 当作完整 Tally Release。
- The L2 branch sync is **not** successful production Domain Package installation, a completed hidden test, or Release PASS.

## F. Product P0 owner-completeness and negative-fixture readback (semantic only)

Before semantic L2 freeze, independently check [P0-01…P0-17 explicit coverage](./TALLY_V2_PRODUCT_R2_TO_L2_GAP_CROSSWALK.md), including P0-02 Requirements UI, P0-07 protected-path Repair/independent verification, P0-12 declared-role scoped delegation, P0-14 cross-session GitHub durability, and P0-15 four-layer Host/App Shell session/Origin+CSRF → B repo/resource authorization → K permit → C actual provider grant with redaction. GitHub publication is not accepted without native readback/source SHA.

P0 physical max simultaneously EXECUTING Task Attempts = **1**, even if DAG has multiple Ready items and worktrees differ. **Three typed causes stay distinct:** no real installed/authenticated Codex CLI or only Fake/echo ⇒ `LOCAL_AGENT_UNAVAILABLE` (do not dispatch; ordinary authorized Product/read/edit UI still available); real Provider is present but Builder credential/access isolation cannot be witnessed ⇒ `BUILDER_CONFINEMENT_UNAVAILABLE` (do not dispatch); independent verifier OS account/ACL missing ⇒ `HIDDEN_VERIFIER_UNAVAILABLE` (no Hidden qualification). Real Codex CLI Provider model/session credentials must be passed by Host to the least-privilege Builder principal independently of the operator's GitHub GCM/gh/SSH write credentials. Sharing credential storage, passwordless elevation or insufficient deny witness blocks with `BUILDER_CONFINEMENT_UNAVAILABLE`; missing protected verifier principal blocks Hidden with `HIDDEN_VERIFIER_UNAVAILABLE`.

Semantic negatives L2-N17…N30 include expired Host session, CSRF/cross-origin attempt, protected diff, secret spill, unauthorized delegation, no durable GitHub artifact, concurrent second executing attempt and credential-store overlap. They are **SPEC_ONLY/NOT_RUN**, not a qualification of real DomainHarness public SDK/Host. **Effective Tally ADS v4.18 adoption requires #85 exact-head source Review and strict verifier**, regardless of upstream RQ READY; old [#81](https://github.com/kaicreator-mm/tally/pull/81) is superseded source history. The [#83 CI scope decision](https://github.com/kaicreator-mm/tally/issues/83#issuecomment-6408620557) keeps legacy Go FAIL unchanged; future native Tally executable tests require Node24/selected Host/SQLite.
