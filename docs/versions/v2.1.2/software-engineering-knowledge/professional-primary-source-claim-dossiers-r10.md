# R10 软件工程专业知识：五项原始来源→命题候选→资格差距档案

> [来源目录](external-source-catalog.md) · [R9 13 项知识准入差距](professional-source-to-claim-readiness-r9.md) · [R10 评价案例](professional-knowledge-evaluation-fixtures-r10.md)
>
> **R10 PRIMARY_SOURCE_PARAPHRASE_ONLY / FIVE_CANDIDATE_DOSSIERS / QUALIFIED_CLAIMS=0 / REAL_SOURCE_DISPATCH=NO。** 本文由 Writer 对五个已编目原始公开来源作**版次、位置、命题、反例和边界的分离**。它们不是用户业务政策、DomainHarness 可执行 Rule、许可证批准或已经经实际 Provider/Host 证明的事实。只有未来具有实际消费目标的独立 Source Owner 才能授予限定资格；本文件的“技术命题”不构成产品规范。

## 一、资格问题：学术正确性和实际消费权力是正交维度

`Reference ↛ Qualified Claim ↛ Authorized Business Rule ↛ Verified Runtime Behavior`。即使某条公开技术规范的术语可核对，仍可能缺少 (1) 某产品为何适用它的有权决策，(2) 正确 Publisher 版次和条文，(3) 许可/复制范围，(4) 可能使当前结论失效的反例，(5) Consumer 所需的实际 Oracle。这里不为每个外部参考引入新的 Task；仅记录五个将来更容易被实际工程目标调用的样本。

## C10-01 — HTTP Problem Details 不是业务幂等保证

- **原始出版来源**：[IETF RFC 9457](https://www.rfc-editor.org/rfc/rfc9457.html)，*Problem Details for HTTP APIs*，July 2023，§3、§3.1、§5；历史 [A70](external-source-catalog.md)。
- **待复核的准确学科命题（原创概述）**：该 RFC 为 HTTP 响应中的机器可读问题信息定义可互操作的表示，包括 `type`、`status`、`title`、`detail`、`instance` 等成员；提供错误结构不等于**告诉客户端远端外部副作用是否实际提交**。
- **作用条件**：仅在被选定 API 真实采用 RFC 9457、其 Content-Type/字段/响应状态与当前 Provider 合同可核实的范围里推断格式。业务可重试性取决于**具体操作和 Provider 的幂等/结果查询协议**，与 RFC 9457 成为两层独立证据。
- **区分性负例**：返回结构化 `application/problem+json` + `status=504`，但下游已经扣款、ACK 丢失；结构化响应不能给出 `NOT_CHARGED` 的证明。`detail` 若含 Bearer Token 则另有敏感数据问题。
- **待有权验证**：实际选定 API 的公开合同/运行版本、对应用例和真实返回内容，授权的 Provider 结果查询；未核实前仅 `REFERENCE_ONLY`。
- **最小 Oracle 候选**：受权环境中对选定 API 的错误形状与实际生效业务身份分开测试；如果真实 Effect 仍不可见，结果必须保留 `OUTCOME_UNKNOWN`。**当前未执行**。

## C10-02 — Node Test Runner 发现/执行条件限制了绿色结论

- **原始出版来源**：[Node.js v22.16.0 Test Runner Documentation](https://nodejs.org/download/release/v22.16.0/docs/api/test.html)，*Running tests from the command line / Test runner execution model / Filtering tests*；历史 [A57](external-source-catalog.md)。
- **待复核命题**：Node 的 test runner 有可指定的运行范围、命名/文件选择和执行行为；某测试文件**未被该次命令发现或运行**时，其他测试的绿色退出码不构成该目标测试的 PASS。
- **边界**：版本固定 v22.16.0；实际采用的 Glob/CLI flags、包装脚本、Subtests/Skip 与 CI 的真实 Test IDs 都需看当前实际日志。不能仅凭文档宣称某项目测试已漏选，或推断 Node24 必要。
- **负例**：选定目标 `contract.spec.mjs`，作业仅发现并实际运行 `*.test.mjs`、40 个通过；新目标的准确状态是 `NOT_RUN`。若确实发现并执行目标而失败，才可按真实 Test Result 记录。
- **待有权验证**：当前项目的测试调用、选用的 Node/Runner 版本、SourceSHA、Test Discovery/Run Manifest、独立合法 Expected；一项都没有真实收据时 `NOT_RUN` 不可写成已观察的项目事件。
- **最小 Oracle 候选**：人为设定带两种扩展名的**隔离合成测试目录**并对比实际发现集合；教学设计未执行，真实用户项目未经授权不能擅自触发 CI。

## C10-03 — PostgreSQL 数据库隔离保证有明确事务边界

- **原始出版来源**：[PostgreSQL 18 文档，§13.2 Transaction Isolation](https://www.postgresql.org/docs/18/transaction-iso.html)，含 §13.2.1–13.2.3 的不同级别语义；历史 [A51](external-source-catalog.md)。
- **待复核命题**：PostgreSQL Serializable 规定数据库并发事务执行的可串行化语义；这一限定与应用向第三方发送的付款 API Effect 不是同一个事务提交对象。该结论是“数据库合同有边界”的推导，不是假定全部网络端到端事务可以原子提交。
- **边界**：准确写明 PostgreSQL 18，不能拿 18 的具体实现行为冒充其他数据库/旧版；Serializable 的回滚/重试条件以及具体读写冲突也应以真实事务/驱动环境为准，**不能把数据库内部并发正确性推导成外部 Effect exactly-once**。
- **负例**：本地 Job/Intent 已在数据库事务提交，第三方财务 Provider 完成扣款却丢失 ACK；本地 Serializable 不会撤销外部扣款。
- **待有权验证**：是否实际使用 PG18、真实事务隔离级别/提交/回执、外部 Provider 幂等/查询和处理权限。
- **最小 Oracle 候选**：**仅隔离模拟**本地 Commit、外部已 Commit、ACK 丢失的交错，要求 `EFFECT_UNKNOWN` 且禁止盲重试。当前没有运行数据库/付款系统。

## C10-04 — SLSA Build Provenance 结构不能替代信任根与发布授权

- **原始出版来源**：[SLSA Build Provenance v1.2](https://slsa.dev/spec/v1.2/build-provenance)，*Purpose / Model / Schema / Verification*；[v1.2 Tracks](https://slsa.dev/spec/v1.2/tracks)；历史 [A68](external-source-catalog.md)。
- **待复核命题**：Provenance 可关联制品 Subject/Digest、Builder、Build Definition/Resolved Dependencies 等出处信息，并有独立验证要求。**任意作者手写一个结构正确的 JSON 不是可信 Builder 证明**；更不能替代实际安装、业务验证或有权 Release。
- **边界**：来源页面是 v1.2，但规范中的 `predicateType` URI 是另有定义的稳定语义标识，不能把网页版本号直接当成可编程 Predicate URI。项目如未合法采用 SLSA 级别合同，不能在知识层强制所有 PR 产生对应证明。
- **负例**：D1 被构建且做过测试，正式上传的 Installer digest 为 D2；一份 Writer 自报的 JSON 把 D2 写成受信发布，不足以满足真实 Builder 信任、实际制品同一性、Host 验证与 Release Owner 决议。
- **待有权验证**：真实 Build System 和可信发行者、证明 Signature/Identity/Policy、下载制品 digest、目标环境及项目 Required Checks/审批；均未取得实际结果。
- **最小 Oracle 候选**：比较两个静态合成 Artifact Digest、证据来源与 Authority；对未经验证 attestation 不授予 `RELEASED`。无真实 SLSA 认证/发布。

## C10-05 — Git Diff 定位材料变更不等于证明消费者没有影响

- **原始出版来源**：[Git 官方 git-diff 文档](https://git-scm.com/docs/git-diff) / [git-merge-base](https://git-scm.com/docs/git-merge-base)；历史 [A59](external-source-catalog.md)。
- **待复核命题**：Git 可以针对确定的 revision 范围比较文件快照差异及其内容；文件级 Diff 是工程影响分析的**输入**，而不是业务语义影响的充分证明。
- **边界**：必须明确比较基点、Head、Path/Ref 和工作树/生成物是否属于选定代码仓库；`git diff A B` 与合并基点/三点语法在具体选择上并非无条件等价，使用时须确认工具真实参数。
- **负例**：H2 只改生成器输入，打包后的消费接口/动态调用语义变化但生成输出未提交；“PR 没改 src/generated”不能推出当前 Consumer 测试可复用。反过来 README 标点变化也不必使全部无关真实 E2E 失效。
- **待有权验证**：真实 SourceSHA、生成规则/构建输入闭包、消费者和测试选择；旧 Review 的 Exact SHA 适用性与当前行为相交的部分。
- **最小 Oracle 候选**：将受影响的生成器/Consumer/Oracle 依赖图与真实 H1/H2 Diff 对齐，只复测有材料变化且无等价证明的路径；当前为纸面分析。

## 二、五档独立复核视图（**不是产品 Enum**）

| 视图 | 本轮实际状态 | 不能由它自动推出 |
| --- | --- | --- |
| Publisher/version/section 定位 | 五个原始出版方入口和章节候选已写入档案，供 Reviewer 复核 | 原文逐句权利、Source Claim 资格 |
| 专业命题与适用范围 | 五项原创技术解释/反例已写入 | 当前具体用户/仓库有该技术约束 |
| 精确条款和许可 | `UNQUALIFIED/RIGHTS_UNCHECKED` | 获准复制原始规范全文 |
| 业务和消费者授权 | `NOT_FROZEN`（#286） | 可以编译/发布选定 DomainHarness Package |
| 独立实际 Oracle/运行收据 | `NOT_RUN` | 实际 Agent/CI/Host/Effect 或产品能力 PASS |

**本轮最多可请求**其他独立对话对“原始出版/准确专业概述/不同版本适用”作有限文档审核；不得由此宣称“已形成 5 个生产合格 Claims”。其余更详细的规则可根据真正 Consumer Profile/DomainHarness ABI 的有权版本按需资格化，避免批量升格 72 条目录参考。
