# K09.DATA_MIGRATION · 数据迁移的真实升级、部分失败和恢复验证

> [K09 知识库](../README.md) · [工程方法](../engineering-methods.md) · [Agent Skills](../agent-skills.md) · [案例](../casebook.md) · [原始依据](../sources.md) · [外部资料目录](../../external-source-catalog.md)。
>
> `KNOWLEDGE_TOPIC_CANDIDATE / NOT_SOURCE_CLAIM_QUALIFIED / PROPOSED_NOT_CONNECTED`。本文件是原创工程问题知识切片，不是 Tally 已生效的机器 Schema，也不是当前 Task/Review/Host/Release 执行许可。案例仅为合成教学示例。

## 1. 工程问题和适用前提

已有数据库存储用户数据，现在要把状态字符串改为受限枚举，部署可能滚动，脚本中途失败怎么办？

**开始前必须确认：**
1. 真实旧版本 Schema/数据样本/异常值
2. 支持的客户端/服务版本、滚动或一次部署条件
3. 回滚/备份/数据恢复与不可逆副作用

**学科知识与因果机制：** Fresh Install、Existing Data Upgrade 和 Recovery 是不同的测试主体。演进式数据库设计倾向把 schema 扩展、旧新读写兼容、历史数据回填和最终收缩分开，适用于渐进发布，但并非每个项目都必须双写。`down migration` 不一定能恢复删除的信息；对于不可逆迁移，需要备份/补偿和明确有权风险决定。

## 2. 可执行的有界判断过程

1. 统计真实旧数据范围、空值/非法枚举值及是否有外部写入者，不以全新空库代替历史数据。
2. 按当前系统评估 expand→backfill→contract 或有界原子迁移，写清哪个版本同时读写哪些字段。
3. 在代表性旧数据上演练迁移中断、重复执行、重启恢复和版本回滚；记录需要的真实 DB/Host 能力。
4. 真实收据缺失则标 Upgrade/Recovery NOT_RUN，即使 fresh install PASS 也不能说整个迁移通过。

## 3. 场景分支、替代方案和不适用范围

| 条件 | 推荐的判断及范围 |
| --- | --- |
| 可在一个事务中完成且全量停机可接受 | 不强制多阶段双写。 |
| 滚动部署且旧新节点并存 | 需保护至少当前兼容窗口。 |
| 存在不可逆历史信息删除 | 不能假设 down migration 能恢复旧值；需要有权恢复方案。 |

## 4. 真实可证伪的教学情景

**受限教学输入及正确路线：** 旧状态 `CANCELLED_UNKNOWN` 不能直接映射新枚举，迁移先记录未知集供 Owner 裁定；已处理行有稳定完成标记，重新执行不产生重复写。

**失败路线/反模式：** 开发只在一个新建空库上验证 `CREATE TABLE` 成功，宣布历史真实用户升级与灾后恢复都通过。

**最低可证伪 Oracle：** 同一旧数据集在注入中途失败后重跑能保持已批准的不变量；旧消费者在明确兼容窗口仍能读取；不具备真实 DB 环境时不伪造验证。

## 5. 最小交接与明确的 STOP

**Agent 输出契约**：当前已观察事实/证据身份、选择的知识与前提、备选方案取舍、正反 Oracle、未得来源或环境、最小下一步。凡 Source Claim 未获得正式资格、Host/CI 未运行、Product/Task 未授权的内容，必须与可证实事实区分。

**跨模块交接**：K02 负责数据所有权/跨版本合同，K05 负责迁移测试 Oracle，K08 负责真实平台执行，K10 负责恢复语义。

**停止规则**：需要越过合法写集、缺少有权业务决定、结果可能有不可逆副作用、外部状态不可确认或没有真实执行来源时，仅提出最小受控检查并保持 UNKNOWN/NOT_RUN，不通过重复调用或虚构状态推进。

## 6. 外部来源与可采纳边界

- [Martin Fowler Evolutionary Database Design](https://martinfowler.com/articles/evodb.html)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。
- [Martin Fowler Strangler Fig Application](https://martinfowler.com/bliki/StranglerFigApplication.html)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。
- [ADS Data Migration Governance](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/DATA_MIGRATION_GOVERNANCE_STANDARD.md)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。

**已发布 ADS 约束**：`kaicreator-mm/ai-development-standard@933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。遇到差异应按当前有权规范与 Product/L2/Task/Host/Release 的真实作用域处理；未合并的三阶段 DAG 候选不自动变为正式工作图。
