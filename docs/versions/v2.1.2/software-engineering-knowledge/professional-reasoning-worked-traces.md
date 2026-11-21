# 软件工程领域智能：五条可反驳的完整知识推理演算

> [知识库总索引](README.md) · [165 点学科台账](foundation-concept-coverage-ledger.md) · [18 组短案例](domain-intelligence-use-cases.md) · [跨模块关系](cross-module-knowledge-relations.md)
>
> **R5 ORIGINAL_WORKED_REASONING_CANDIDATE / SYNTHETIC / NO_AGENT_RUNTIME / NO_TEST_OR_PRODUCT_QUALIFICATION。** 与现有 Q01–Q18 的快速“输入/正确/错误”教学示例不同，本文完整演示**一个真实软件工程问题应如何选择并推导知识**。所有角色、案例数据、状态和文中可观察结果均为**教学假设，不是已运行的 Tally/DomainHarness Consumer**。不提出新的 Graph ABI、Rule Schema 或 GitHub Task 权威。

## 1. 知识推理的可审核结构：反事实与证据同等重要

一个 Domain Intelligence 学习性回答应具备以下自然语言部分，而不只是给出检索链接：

| 部分 | 需要说明 | 明确不能替代 |
| --- | --- | --- |
| 需求问题 | 这个任务究竟要求什么可观察结果 | Agent 自己构造的业务目标 |
| 事实/来源 | 谁/何时/何版本观察或授权该条件 | URL 列表等于已接受 Source Claim |
| 知识路由 | 为什么选择特定 Kxx、Concept、Topic/方法 | 全部 165 条知识无差别灌入上下文 |
| 机制与备选 | 因果如何成立、何种输入会改变选型 | 模型喜好/框架流行程度 |
| 可证伪反例 | 什么反例会使当前推荐错误 | 只列正例或复述实现逻辑 |
| Oracle/执行界限 | 哪个事实属于教学 Expected，哪个是实际执行 | 未运行的 CI/Host/用户 E2E PASS |
| 决议/未知 | 此刻能建议什么、需要谁进一步裁定什么 | 自动代签 Product/Task/Release |

推荐把“需要当前答案的最少必要专业知识”与“整个学科书架”分开：可在问题发生时按 Concept ID 调到相邻章，最终响应只引用足以支撑决定的几个来源和一个最能检验错误假设的反例。

## 2. 演算 W01：已发货取消政策冲突 + 竞争写入

**受影响问题**：Web UI 能让用户在发货后取消；客服旧 FAQ 提示可以取消，但正式规则可能禁止；两个 Worker 可能同时发货和取消。

**候选输入事实**：

| 事实 ID | 内容 | 状态 |
| --- | --- | --- |
| F1 | **教学假设**的有权政策：普通用户在 `PAID`/`PACKING` 可按条件取消，`SHIPPED` 提交后禁止普通取消 | 仅为合成教学假定 `AUTHORIZED`，不代表真实政策 |
| F2 | 旧 FAQ 声称付款后均可取消，生效时间/适用角色未知 | `STALE_OR_UNKNOWN` |
| F3 | 数据库存储 `state, revision`，两个 Worker 可能从同一版本读取 | 候选实现假定 |
| F4 | UI 有“取消”按钮 | 可观察界面，不等于政策 |

**路由与判断**：
1. [K01 契约推理](K01/advanced-knowledge-from-evidence-to-contract.md) 比较角色、状态、事件、有效时间。F2 与 F1 只有真正相同条件才可能冲突；F4 不拥有授权地位。
2. [K02 状态所有权](K02/foundations-deep-dive-01.md) 确认合法状态迁移的单一权威提交点；不能由 UI 或未经审查的并行 Agent 自行保管。
3. [K03 条件写](K03/advanced-concurrency-idempotency-proof-sketch.md) 分析 A/B 两个 Worker 的版本竞争，找出可发生的 `SHIPPED -> CANCELED` 不允许跃迁。
4. [K05 独立 Oracle](K05/advanced-oracle-adequacy-and-test-selection.md) 要求正常、越权、交错、过期 Policy 等区别性测试。
5. [K06 补丁当前性](K06/focus-f4-3-patch-currentness-and-review.md) 检查哪个精确提交修复上述错误。

**强反例**：A 在 `PACKING` 读取并按 F1 合法准备取消，B 先把 `SHIPPED` 成功提交，A 没有在提交点重新核对版本/状态，仍用旧结果写入 `CANCELED`。A 最初的判断合法；违反 F1 的原因仅是后续竞争交错和缺少有效条件写。若取消先于发货提交，则发货也应依状态合同拒绝，二者不能都被承认为有效。若未使用有效并发控制，即使页面层 Actor 检查全绿，也能违反 F1。反过来，若最新有权政策允许撤回已发货商品且具合法补偿流程，禁止所有取消也可能错误，不能静态宣称这个演算永远适用。

**最小产出**：规则来源/冲突范围、一个受权状态转移表候选、并发反例和未验证的真实 DB/Host 前提；如果 F1 并非真实授权，输出 `ASK_POLICY_OWNER` 而非自动修改代码。教学演算没有运行 DB/CI。

## 3. 演算 W02：重复投递、ACK 丢失与未知资金效果

**受影响问题**：后台任务采用至少一次队列；远端扣款提交后响应超时，本地仍处于 CLAIMED。用户提出“超时后立即重试直到成功”。

**路由与机制**：
- K01 给出不可重复付款、对账/补偿政策；K02 给出 Job 状态 Owner 和提交责任；
- [K03 详细崩溃点矩阵](K03/advanced-concurrency-idempotency-proof-sketch.md) 说明本地 revision 只能保护部分状态，并不回滚远端 Effect；
- [K10 UNKNOWN_EFFECT](K10/foundations-deep-dive-01.md) 限制重复不可逆动作；
- [K05 模型/状态测试](K05/foundations-deep-dive-02.md) 构造 `remoteCommit -> crash -> redeliver`；
- [K11 价值/风险](K11/advanced-empirical-evaluation-and-value-of-knowledge.md) 不允许为了节省 100ms 以未授权二次扣费代价优化吞吐。

**两个真实可选前提**：如果远端批准以稳定业务 ID 在足够的时间窗内去重、并能读回权威结果，则可以使用该合同设计安全恢复；如果外部供应商不提供去重/查询，可能必须隔离未决付款、使用人工/有权补偿路径或缩小自动重试保证。选择依据是**外部真实接口语义**而非“HTTP POST 后超时”。

**最短反例**：远端已扣款→本地超时→自动重试→远端第二次扣款。若去重接口真实有效，这条反例的最后一步可被拒绝，故所有验证都必须注明这项前提。

**推断边界**：不记录实际 Effect 查询或 Provider 回执时，正确结果是 `UNKNOWN_EFFECT / DO_NOT_BLIND_RETRY`；知识库不能把这称为已实现 exactly-once。

## 4. 演算 W03：CI 绿色但 Package 不可使用

**受影响问题**：当前 Node JS suite exit 0；新合同测试由于 glob 未被发现。项目尚未建立 Windows 安装包，PR 报告“全平台 Release Ready”。

**知识路由**：
1. [K05 对 Oracle 与 Test Discovery 的分析](K05/advanced-oracle-adequacy-and-test-selection.md)：明确真实发现到的测试集合与材料测试的 NOT_RUN。
2. [K05 Build/Package/Install/Launch](K05/focus-f3-3-build-package-platform-verification.md)：区分编译成功、目标制品存在、安装成功、可运行的真实用户路径。
3. [K08 构建与内容输入](K08/foundations-deep-dive-02.md)：标记当前源 SHA、工具链、依赖、Package hash、真实 Runner/Host 的结果；如果生成缓存键不包含 Schema 输入，CI GREEN 仍可能使用过期制品。
4. [K06 失败因果](K06/foundations-deep-dive-02.md)：若构建阶段失败，下游 Test/Package 不能冒充执行过；若 Runner 漏测，不能把它写成“全部测试 PASS”。
5. [K10 首次启动和配置](K10/focus-f5-1-first-run-post-start-configuration.md)：GUI 可在 UNCONFIGURED 状态进入配置，不代表能在没有授权项目时执行远端任务。

**区分性探针**：在指定 Node 版本下读回发现清单；检查实际可下载的 Windows 安装包是否在当前 SHA 的制品目录；在支持的 Windows Host 做最小安装/启动/配置路径；对不需真实特定 Provider 的材料性测试可在等价环境复用，但不能将 Mock 顶替特定 Host 收据。

**反例**：`node --test` 运行 100 个旧测试通过，真正材料性 `contract.spec.mjs` 未运行，仍宣传“所有测试通过、Windows 安装包已验证”。正确产出是分层状态与最小补测建议，不强制跑所有无关历史测试。

## 5. 演算 W04：同一任务多 Agent 与旧 Review 的错误晋升

**受影响问题**：两个 Agent 并行修改独立文件，但一个共享生成 Schema 随后被改；旧 Agent 返回 DONE，旧 Reviewer 已接受 H1，当前 PR 为 H2。

- [K04 任务授权知识](K04/focus-f2-3-task-acceptance-and-authority.md) 区分接收结果与正式当前完工决议。
- [K07 会话/Attempt 与协作](K07/foundations-deep-dive-02.md) 判断迟到消息只能对应原 SourceSHA，不能覆盖当前 GitHub 权威。
- [K06 精确影响闭包](K06/focus-f4-3-patch-currentness-and-review.md) 比较 H1→H2 的生成物/依赖，并保持受影响子范围的增量复核。
- [K05 Oracle 与 K08 执行证据](K05/foundations-deep-dive-01.md) 说明旧测试仍是历史事实，但新受影响调用必须在当前输入下被真正验证。

**反例**：H1 合法修复了 Auth，H2 更新共享 Router 造成绕过，两个 Agent 的原本各自单测均绿。只按文件名“Router != Auth”判无关会漏掉缺陷；反之 H2 仅改与 Auth 无关的说明文字时，也不需要重跑所有实际宿主 E2E。

**结论分类**：`VALID_HISTORICAL_SCOPE`、`CURRENT_CHANGED_SCOPE_REQUIRES_EVIDENCE`、`REVIEW_OWNER_DECISION_PENDING`。这些是教学分类词，并非新 GitHub Task Status。

## 6. 演算 W05：知识辅助 Agent 是否带来真实收益

**受影响问题**：团队宣称知识 Agent 在 10 个问题通过 8 个，对照 Agent 在 10 个问题通过 6 个，要求立即认定知识库提升 33%。

根据 [K11 实证方法](K11/advanced-empirical-evaluation-and-value-of-knowledge.md)，观察到的比例分别 80%/60%，绝对差异 20 个百分点；“提升 33%”也只是这组分母下的**相对观察差异**，不是知识导致的因果提升。如果两个组用了不同模型、不同任务难度、权限或预算，混杂更加明显。

**最小严格反事实**：以相同的任务级独立 Oracle、相近预算与工具设置比较“有当前相关知识”与“没有这些材料”；记录错误/越权、人工修复、整体墙钟时间和最终可维护性。不能因为 Token 消耗降低而忽略新增审核和返工成本，也不能只保留成功数据。K05 保护实际正确性底线，K07 控制 Agent 执行环境和旧消息偏差，K04 保护任务的可比来源。

**反例**：知识组成功率较高，但全部项目均在旧 SHA 上测试并且有一次严重跨租户数据泄露，按有权质量底线不能宣布优于正确的基线实现。若任务太少、未做真实实验，应报告 `STUDY_NOT_RUN/INSUFFICIENT_CAUSAL_EVIDENCE`。

## 7. R5 本轮的准确资格边界

本文件完成的是**原创的多学科知识选择、推理与可证伪教学样例**。它没有执行五个 Agent 任务，没有查询真实付款 Effect，没有生成真正发布制品，没有计算用户真实提升比例，也不预设 DomainHarness 0.11.x 的知识 ABI。被例子引用的 Source/规则若未来需要用于产品运行，仍应按具体版本、许可、授权、实际消费者独立取得资格。

**专业完整性不是只数段落**：可复核的完整推理需要 Source 范围、明确的业务/技术机制、足以区分方案的条件、可被现实推翻的反例、可行动的不确定性和明确的 Owner 边界。当前 165 个学科课程点与 61 项发现来源仍属于文档层积累，不能充当 165 项已在生产 Agent 中证明的能力。

**来源路由**：上述五演算使用原有 [61 项来源目录](external-source-catalog.md) 中的知识线索，优先从相应 Kxx 原课程/专业深文推导，不增加无真实需要的额外标准、任务 Gate 或知识服务层。
