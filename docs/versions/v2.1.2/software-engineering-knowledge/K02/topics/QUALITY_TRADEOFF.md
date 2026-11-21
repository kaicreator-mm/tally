# K02.QUALITY_TRADEOFF · 以质量属性场景选择架构，而非凭技术潮流

> [K02 知识库](../README.md) · [专业方法](../engineering-methods.md) · [Agent Skill](../agent-skills.md) · [反例与案例](../casebook.md) · [ADS 来源与外部学科目录](../sources.md) · [外部资料总索引](../../external-source-catalog.md)。
>
> `KNOWLEDGE_TOPIC_CANDIDATE / ORIGINAL_EXPLANATION / NOT_CLAIM_QUALIFIED / PROPOSED_NOT_CONNECTED`。Topic 是问题驱动的知识切片，不改变 Workflow、Task、Reviewer、Host 和 Release 权限；下面案例为合成教学条件，不是实际 Host PASS。

## 1. 问题与应用触发

一项异步导入要求重启后保留状态、UI 不阻塞；要选择同步请求、进程内队列还是持久化任务系统？

**适用前置输入：**
1. 需求/质量目标的有效来源，真实部署形态与限制
2. 候选方案和当前工程依赖/团队维护能力
3. 测试平台、可控失败注入和成本信息的真实/未知部分

**学科原理与为什么这样决策：** 架构方案应该从有来源的质量属性场景推导，而不是从“微服务更先进”推导。架构关注的可观察结果要写出刺激（重启、并发、断连）、环境、响应与测量/证伪方式。SEI ATAM 可用于发现影响多个质量属性的敏感点及权衡点；此项目只有一项局部决策时可采用轻量表格而非完整多日 ATAM。

## 2. 实际决策程序（按现有权限执行）

1. 列出恢复性、交互响应、部署运维成本、持久化一致性等相关场景，不自动为每个项目添加全部 ISO 质量特征。
2. 对每个方案写出其能力的因果前提：内存队列进程重启后为何会丢失；持久队列需什么原子提交、重复领取和恢复合同。
3. 标出权衡点：可恢复性提升可能增加部署/一致性/延迟负担；响应性可能通过 Worker 线程而非独立服务实现。
4. 设计最小可证伪实验：启动作业、持久状态、模拟重启、重连查询；记录证据环境和现有 Product/L2 决策权。

## 3. 按条件选择，而不是背诵一种答案

| 场景条件 | 当前应作的判断 |
| --- | --- |
| 仅 UI 不阻塞，无恢复要求且单进程 | 进程内 Worker 可能足够。 |
| 恢复要求已授权且重启概率/后果材料性 | 不能仅靠内存 Map，需要持久化或可重建权威状态。 |
| 缺少队列部署/数据成本数据 | 列 ESTIMATED 与 UNKNOWN，不写确定优胜。 |

## 4. 可复用的教学案例与负例

**合成正例：** 内存队列在 1000 个任务中吞吐不错，但重启后任务状态全丢；持久队列吞吐较低但能够查询未完 Job。最终选择取决于真实恢复要求与运行成本，不能只对比单次吞吐。

**失败/反模式：** 在没有重启的 Mock Demo 中测得每秒处理数更大，于是宣布内存队列“兼顾可靠性和性能”，并冻结 L2。

**可证伪的结果 Oracle：** 同一场景必须列出一个实验可以推翻备选方案；没有实际重启验证时只允许 `DESIGN_HYPOTHESIS`，不能 `PRODUCTION_RELIABILITY_VERIFIED`。

## 5. 最小输出、拒绝和跨模块交接

**Agent 最小输出**：当前事实/适用条件、被选知识及原因、候选/未选方案的权衡、具体正反 Oracle、缺失证据和 STOP、原始来源身份。不能取得真正 Product/Task/CI/Host 证据时要标 `UNKNOWN/NOT_RUN`，不能将本 Topic 引用的外部资料当作已发布 Tally 规则。

**交接关系：** K10 提供可靠性/恢复知识，K11 比较总成本；K02 交出 ADR/合同，K05 设计正反验证。

**停止条件**：业务政策无权确定、错误可能影响不可逆外部副作用、需要超出合法写集的实施、无法取得必要证据时，输出受影响范围及最小合法下一步，而不是调用无界 Agent 循环。

## 6. 来源、版本与知识资格

- [ISO/IEC/IEEE 42010:2022](https://www.iso.org/standard/74393.html)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。
- [CMU SEI ATAM](https://www.sei.cmu.edu/library/architecture-tradeoff-analysis-method-collection/)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。
- [SEI Early Architecture Analysis](https://www.sei.cmu.edu/library/a-playbook-for-early-architecture-analysis/)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。

**ADS 约束基线**：`kaicreator-mm/ai-development-standard@933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；若与现行项目已生效规范冲突，应以具有权威的原规则为准并说明冲突。历史 ADS #1100、Draft #1384 的统一三阶段 DAG 观点不作为现行强制规则。
