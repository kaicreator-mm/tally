# K01.ACCEPTANCE · 从需求到可证伪业务验收与双向追溯

> [K01 知识库](../README.md) · [专业方法](../engineering-methods.md) · [Agent Skill](../agent-skills.md) · [反例与案例](../casebook.md) · [ADS 来源与外部学科目录](../sources.md) · [外部资料总索引](../../external-source-catalog.md)。
>
> `KNOWLEDGE_TOPIC_CANDIDATE / ORIGINAL_EXPLANATION / NOT_CLAIM_QUALIFIED / PROPOSED_NOT_CONNECTED`。Topic 是问题驱动的知识切片，不改变 Workflow、Task、Reviewer、Host 和 Release 权限；下面案例为合成教学条件，不是实际 Host PASS。

## 1. 问题与应用触发

用户要求“批量关闭工单”，开发者应该怎样确定什么是成功、什么是非法，尤其遇到部分成功、重复执行和角色差异？

**适用前置输入：**
1. 明确的业务 Actor/Role 与可操作资源
2. 每个状态对应允许/禁止操作的已批准规则
3. 用户能观察的响应、数据状态、副作用与失败体验

**学科原理与为什么这样决策：** 验收不是实现后的快照，而是需求来源到可观察结果的可证伪合同。需求可采用角色×状态×动作的决策表，明确前置条件、正常输出、非法状态拒绝、外部副作用和不应变化的不变量。NASA 对任务级需求追溯的高保证做法可作为质量参考，普通项目只保留真正需要的来源链，不强制生成完整认证矩阵。

## 2. 实际决策程序（按现有权限执行）

1. 识别正常与拒绝边界，并写出至少一对在同一控制变量下不同结果的正/负 Case。
2. 将预期写成环境无关的判断：业务状态/输出/副作用而非某个函数 mock 被调用。
3. 把部分成功、幂等、取消和异步结果分别标为批准/未知；未知业务语义不能直接进入正式 Oracle。
4. 按 Requirement→Rule/Invariant→Acceptance→候选 Test 局部建立可反查关系，接口/实现改变只通知相关消费者。

## 3. 按条件选择，而不是背诵一种答案

| 场景条件 | 当前应作的判断 |
| --- | --- |
| 业务规则有权且可检查 | 形成验收并交 K05 测试设计。 |
| 需要产品决定是否全批原子 | 保留 UNKNOWN，不擅定事务模型。 |
| 只有 UI 文案而无行为变化 | 验收只覆盖可观察用户差异，不制造多层契约实验。 |

## 4. 可复用的教学案例与负例

**合成正例：** 管理员对三个工单批量 CLOSE；两条 OPEN、一条 CLOSED。若产品决定部分成功，期望两条成功、一条结构化拒绝，且已经 CLOSED 的状态和副作用无新变化。若产品决定全批原子，同样输入应全部回滚。两者必须通过真实有权决议选择。

**失败/反模式：** 测试代码只是 `expect(closeSpy).toHaveBeenCalledTimes(3)`；即便数据库实际将三个任务全部重复关闭也可能测试通过。

**可证伪的结果 Oracle：** 给两个可区分结果的业务分支和最低一条错误实现应失败的负例；没有经过正式选定的部分成功政策时不得出具 `ACCEPTANCE_FROZEN`。

## 5. 最小输出、拒绝和跨模块交接

**Agent 最小输出**：当前事实/适用条件、被选知识及原因、候选/未选方案的权衡、具体正反 Oracle、缺失证据和 STOP、原始来源身份。不能取得真正 Product/Task/CI/Host 证据时要标 `UNKNOWN/NOT_RUN`，不能将本 Topic 引用的外部资料当作已发布 Tally 规则。

**交接关系：** K05 负责测试 Oracle/充分性，K02 负责事务边界，K03 实现当前受权行为。

**停止条件**：业务政策无权确定、错误可能影响不可逆外部副作用、需要超出合法写集的实施、无法取得必要证据时，输出受影响范围及最小合法下一步，而不是调用无界 Agent 循环。

## 6. 来源、版本与知识资格

- [IREB CPRE syllabus](https://cpre.ireb.org/en/downloads-and-resources/downloads)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。
- [NASA SWE-050 Software Requirements](https://swehb.nasa.gov/spaces/SWEHBVB/pages/32604503/SWE-050%2B-%2BSoftware%2BRequirements)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。
- [IEEE SWEBOK V4 Topics](https://www.computer.org/education/bodies-of-knowledge/software-engineering/topics)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。

**ADS 约束基线**：`kaicreator-mm/ai-development-standard@933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；若与现行项目已生效规范冲突，应以具有权威的原规则为准并说明冲突。历史 ADS #1100、Draft #1384 的统一三阶段 DAG 观点不作为现行强制规则。
