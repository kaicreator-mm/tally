# K04.REAL_DEPENDENCY · 真假依赖的反事实判定和 Task DAG 最小化

> [K04 知识库](../README.md) · [专业方法](../engineering-methods.md) · [Agent Skill](../agent-skills.md) · [反例与案例](../casebook.md) · [ADS 来源与外部学科目录](../sources.md) · [外部资料总索引](../../external-source-catalog.md)。
>
> `KNOWLEDGE_TOPIC_CANDIDATE / ORIGINAL_EXPLANATION / NOT_CLAIM_QUALIFIED / PROPOSED_NOT_CONNECTED`。Topic 是问题驱动的知识切片，不改变 Workflow、Task、Reviewer、Host 和 Release 权限；下面案例为合成教学条件，不是实际 Host PASS。

## 1. 问题与应用触发

A 是补充文档，B 是真实接口变更，C 是 UI 功能；Planner 把 A→B→C 全部标硬性 BLOCKED_BY，是否合理？

**适用前置输入：**
1. 每个任务实际可独立验收的交付产物
2. 后续节点必须消费的合同/数据/权限/环境
3. 共享写集、集成点以及当前已有权依赖

**学科原理与为什么这样决策：** 执行依赖必须由后续任务实际消费的前置产物决定，而不是由“流程次序”或拥有相同父 Issue 决定。`CONSUMES_EVIDENCE` 只是参考，不一定是执行阻塞。研究/设计层可以有有界证据依赖，但不能让拟议 DAG-P/A 图自动成为实施 Task 的 Native Dependency；生命周期标准不规定统一三图流程。

## 2. 实际决策程序（按现有权限执行）

1. 对每条边写具体因果句：后项没有哪份文件/接口/决策/证据就不能合法执行？
2. 做反事实移除实验：删除该边，后续任务是否仍能在当前允许写集内完成并通过必要验证？若能，则删除硬边或改为参考关系。
3. 识别环路、共享 Owner、集成后才能发现的缺陷；把真正共享接口作为一个受控前置成果。
4. 仅将合法、获权的实施依赖映射到实际 GitHub Native Task 关系；旧结论仅在同版本/范围有效。

## 3. 按条件选择，而不是背诵一种答案

| 场景条件 | 当前应作的判断 |
| --- | --- |
| A 文档无关 B 的接口/权限 | 不设 A→B 硬阻塞。 |
| B 真正依赖尚未确定的 Schema | 前置合同材料性，保留 B 等待它。 |
| C 可基于受控 Mock 合同并行开发 | 允许并行探索，但实际集成/验收仍必须消费真实 B。 |

## 4. 可复用的教学案例与负例

**合成正例：** A 是 Release Notes 文本，B 是订单 API，C 是 UI 消费端；API 与 UI 可以通过先明确 Response Contract 分离写集，最终真实集成；A 完成与否不影响 B 的源码实现。

**失败/反模式：** 把无关文档作为所有实施任务共同先决条件，导致 CI 和开发队列长时间空闲却没有新增质量收益。

**可证伪的结果 Oracle：** 对 A→B 和 B→C 两条边分别写反事实证明；如果没有能具体指认的前置产物，则不应出现 Native `BLOCKED_BY`。

## 5. 最小输出、拒绝和跨模块交接

**Agent 最小输出**：当前事实/适用条件、被选知识及原因、候选/未选方案的权衡、具体正反 Oracle、缺失证据和 STOP、原始来源身份。不能取得真正 Product/Task/CI/Host 证据时要标 `UNKNOWN/NOT_RUN`，不能将本 Topic 引用的外部资料当作已发布 Tally 规则。

**交接关系：** K07 处理 Agent 并行交接，K11 做成本/收益，正式 Task 由现有 GitHub 权威操作。

**停止条件**：业务政策无权确定、错误可能影响不可逆外部副作用、需要超出合法写集的实施、无法取得必要证据时，输出受影响范围及最小合法下一步，而不是调用无界 Agent 循环。

## 6. 来源、版本与知识资格

- [ADS Task Decomposition v4.19.0](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/TASK_DECOMPOSITION_STANDARD.md)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。
- [GitLab Engineering Workflow](https://handbook.gitlab.com/handbook/engineering/workflow/)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。
- [ISO/IEC/IEEE 12207:2026](https://www.iso.org/standard/90219.html)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。

**ADS 约束基线**：`kaicreator-mm/ai-development-standard@933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；若与现行项目已生效规范冲突，应以具有权威的原规则为准并说明冲突。历史 ADS #1100、Draft #1384 的统一三阶段 DAG 观点不作为现行强制规则。
