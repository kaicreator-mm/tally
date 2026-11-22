# K04.PARALLEL_WRITES · 并行 Builder 的写集划分和接口稳定性判定

> [K04 知识库](../README.md) · [专业方法](../engineering-methods.md) · [Agent Skill](../agent-skills.md) · [反例与案例](../casebook.md) · [ADS 来源与外部学科目录](../sources.md) · [外部资料总索引](../../external-source-catalog.md)。
>
> `KNOWLEDGE_TOPIC_CANDIDATE / ORIGINAL_EXPLANATION / NOT_CLAIM_QUALIFIED / PROPOSED_NOT_CONNECTED`。Topic 是问题驱动的知识切片，不改变 Workflow、Task、Reviewer、Host 和 Release 权限；下面案例为合成教学条件，不是实际 Host PASS。

## 1. 问题与应用触发

三个 Agent 同时做 API、UI 和 DB Schema；每个人都有独立工作树，是否意味着互不冲突？

**适用前置输入：**
1. 各 Task 可独立验收的成果与目录/生成物写集
2. 公共 API/Schema/锁文件/版本迁移 Owner
3. 合法 Source SHA 与当前 Review/集成计划

**学科原理与为什么这样决策：** 工作树隔离只防止同一文件系统内直接覆盖，不意味着源逻辑、Schema、生成代码、数据库迁移或锁文件没有交叉写入。并行成立需要独立可验收结果、稳定接口合同和可集成顺序。过于横向的任务拆分把本应一次完成的行为/测试分开，反而增加审查、等待和回滚成本。

## 2. 实际决策程序（按现有权限执行）

1. 先列全部共享可变资产，包括 API 类型、数据库 Schema、迁移顺序、package lock、生成代码和测试 fixture。
2. 对每项共享资产指定唯一 Writer，或定义明确受控的串行合并阶段；公共合同先确定最小稳定版本。
3. 为每个 Agent 给不重叠的可操作写集、当前 Source SHA 和单次验收标准；必要时只读并行，代码实施串行。
4. 集成时以当前同一 HEAD 执行跨组件/消费者测试，不能将三个分支各自 Unit PASS 合成为系统 PASS。

## 3. 按条件选择，而不是背诵一种答案

| 场景条件 | 当前应作的判断 |
| --- | --- |
| 只读设计/测试分析 | 通常可并行，交接候选而非直接改源码。 |
| 互不交叉的实现与稳定接口 | 可有界并行。 |
| 共享 Schema 未确认或互斥写集冲突 | 先合同/单 Writer，停止冲突并行。 |

## 4. 可复用的教学案例与负例

**合成正例：** API Builder 和 UI Builder 都改 `types.ts`，DB Builder 改同一迁移文件；即使 git 分支独立，合并时出现数据/类型冲突。调整为 Schema 单 Writer，另两个模块只消费稳定合同。

**失败/反模式：** 仅凭三台 Agent 的 workspace 目录不同就宣布写集独立，并由最后一个提交覆盖其他模块的类型修订。

**可证伪的结果 Oracle：** 计划必须标记每项共享生成/迁移/锁文件的唯一 Owner；合并前有同一 HEAD 的真实公开消费路径验证，缺少则 `INTEGRATION_NOT_RUN`。

## 5. 最小输出、拒绝和跨模块交接

**Agent 最小输出**：当前事实/适用条件、被选知识及原因、候选/未选方案的权衡、具体正反 Oracle、缺失证据和 STOP、原始来源身份。不能取得真正 Product/Task/CI/Host 证据时要标 `UNKNOWN/NOT_RUN`，不能将本 Topic 引用的外部资料当作已发布 Tally 规则。

**交接关系：** K07 负责交接 Role/Attempt、Review 独立性；K08 负责集成/CI 来源收据。

**停止条件**：业务政策无权确定、错误可能影响不可逆外部副作用、需要超出合法写集的实施、无法取得必要证据时，输出受影响范围及最小合法下一步，而不是调用无界 Agent 循环。

## 6. 来源、版本与知识资格

- [Microsoft Engineering Playbook](https://github.com/microsoft/code-with-engineering-playbook)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。
- [GitLab Modular Code](https://handbook.gitlab.com/handbook/engineering/development/growth/modular_code/)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。
- [ADS Task Decomposition](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/TASK_DECOMPOSITION_STANDARD.md)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。

**ADS 约束基线**：`kaicreator-mm/ai-development-standard@933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；若与现行项目已生效规范冲突，应以具有权威的原规则为准并说明冲突。历史 ADS #1100、Draft #1384 的统一三阶段 DAG 观点不作为现行强制规则。
