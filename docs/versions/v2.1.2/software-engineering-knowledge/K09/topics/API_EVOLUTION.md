# K09.API_EVOLUTION · 旧消费者保护、版本窗口与渐进接口演化

> [K09 知识库](../README.md) · [工程方法](../engineering-methods.md) · [Agent Skills](../agent-skills.md) · [案例](../casebook.md) · [原始依据](../sources.md) · [外部资料目录](../../external-source-catalog.md)。
>
> `KNOWLEDGE_TOPIC_CANDIDATE / NOT_SOURCE_CLAIM_QUALIFIED / PROPOSED_NOT_CONNECTED`。本文件是原创工程问题知识切片，不是 Tally 已生效的机器 Schema，也不是当前 Task/Review/Host/Release 执行许可。案例仅为合成教学示例。

## 1. 工程问题和适用前提

旧客户端把查询结果当数组，新客户端要求分页对象；如何演进接口而不是悄悄破坏所有消费者？

**开始前必须确认：**
1. 已支持的客户端/版本与真实 API 合同
2. 变更的业务目的、返回数据/错误/性能约束
3. 允许的迁移窗口、消费者测试与 Release 计划

**学科知识与因果机制：** 演进应区分 Refactoring（保持现有公开行为）、Feature Change（受权新行为）和 Breaking Change（需明确版本/消费者政策）。兼容不仅涉及类型，还涉及错误、排序、空值、分页语义、版本窗口和部署顺序。增量扩展可以先增加可选字段或新端点，待真实消费者迁移完成再由有权 Owner 取消旧合同；不要求永久支持所有历史消费者。

## 2. 可执行的有界判断过程

1. 列 Consumer×Version×Old/Proposed Behavior 矩阵，判断哪些行为必须保留、哪些允许变化以及是否已批准。
2. 比较新端点、参数 opt-in、版本化响应或一次性 Breaking Change；逐项写出客户端升级、文档和测试成本。
3. 为旧新 Consumer 分别定义正反合同，并通过真实接口/Consumer 测试，不以更新 Golden 隐藏旧行为退化。
4. 迁移/停用旧接口仅在政策/真实消费者状态确定后推进；缺少支持窗口事实时 UNKNOWN。

## 3. 场景分支、替代方案和不适用范围

| 条件 | 推荐的判断及范围 |
| --- | --- |
| 受支持旧客户端依然读取旧数组 | 不得直接将全部响应替换成分页对象。 |
| 正式 major version 已批准不兼容 | 可定义清楚迁移路径和受影响证明。 |
| 没有实际消费者但旧接口在测试中有历史引用 | 先核真实支持范围，不自动要求永久兼容。 |

## 4. 真实可证伪的教学情景

**受限教学输入及正确路线：** `GET /orders` 保留数组，新增 `GET /orders/page?page=2` 返回分页对象，并让旧新客户端真实调用各自端点；后续是否废弃旧接口由发布策略决定。

**失败路线/反模式：** 直接修改旧 `GET /orders` 的返回形状，再把旧快照测试更新成新期望，导致老桌面客户端运行时读取失败。

**最低可证伪 Oracle：** 当旧客户端仍被支持时，变更之后旧版本合同仍 PASS；如果某被支持客户端读取失败，应为 REWORK 而非被新测试忽略。

## 5. 最小交接与明确的 STOP

**Agent 输出契约**：当前已观察事实/证据身份、选择的知识与前提、备选方案取舍、正反 Oracle、未得来源或环境、最小下一步。凡 Source Claim 未获得正式资格、Host/CI 未运行、Product/Task 未授权的内容，必须与可证实事实区分。

**跨模块交接**：K01 决定用户兼容承诺，K02 设计合同，K05 验证消费者，K11 评估长期成本。

**停止规则**：需要越过合法写集、缺少有权业务决定、结果可能有不可逆副作用、外部状态不可确认或没有真实执行来源时，仅提出最小受控检查并保持 UNKNOWN/NOT_RUN，不通过重复调用或虚构状态推进。

## 6. 外部来源与可采纳边界

- [Martin Fowler Evolutionary Database Design](https://martinfowler.com/articles/evodb.html)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。
- [ISO/IEC/IEEE 12207:2026](https://www.iso.org/standard/90219.html)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。
- [Google Software Engineering Book](https://abseil.io/resources/swe-book)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。

**已发布 ADS 约束**：`kaicreator-mm/ai-development-standard@933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。遇到差异应按当前有权规范与 Product/L2/Task/Host/Release 的真实作用域处理；未合并的三阶段 DAG 候选不自动变为正式工作图。
