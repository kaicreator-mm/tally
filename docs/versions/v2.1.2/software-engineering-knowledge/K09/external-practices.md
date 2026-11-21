# K09 · 软件演进与重构：权威实践与可证伪案例

> [K09 知识库](README.md) · [v2.1.2 #521](https://github.com/kaicreator-mm/tally/issues/521) · [外部来源总目录](../external-source-catalog.md)。**EXTERNAL_DISCIPLINARY_SOURCE_CANDIDATE / NOT_SOURCE_QUALIFIED / NOT_CONNECTED**；本文件为独立原创工程知识与实际方法的总结，非受版权保护的国际标准/教材原文复刻，不提升现有 Task/Release 权威。

## 从公开实践提炼有效技术，而非复制别人的流程

这里的实践案例是独立构造的**教学输入**，用外部权威/成熟项目的思想解释 Agent 应如何作决策。公开 GitHub 项目的原始能力受技术栈、依赖、执行环境和许可约束；其 README 或 Repo Star 不是验证当前 Tally Task 的权威。相应实例与成熟度都要逐项核对。

### P-K09-01 · 双版本 API 平滑演进

**合成任务输入：** 输入：旧客户端期待订单数组，新客户端需要分页对象。

**逐步操作：**
1.  确认旧消费者受支持版本。
2.  设计新端点/可选参数/版本化响应候选。
3.  定义必须保留的旧 Oracle 与新功能 Acceptance。
4.  对两个实际消费者跑合同/集成测试。
5.  旧版本停用另需适用 Owner 决策。

**正反 Oracle：** 正确：旧客户端仍能读取受支持返回，新客户取得分页；错误：直接改成对象并把旧 Golden 更新为新值。

**出处与限制：** [A01](../external-source-catalog.md)、[A41](../external-source-catalog.md)。该示例是合成教学材料，并非宣称这些出版方已在对应项目上执行。真正可验的 Task 判断必须由受批准的业务 Oracle 和真实测试来源确定。

### P-K09-02 · 持久化迁移中断恢复

**合成任务输入：** 输入：需要把订单状态从字符串迁移为枚举，已有旧记录且允许滚动升级。

**逐步操作：**
1.  先分析实际旧数据分布与异常值。
2.  设计 expand/read-compat/backfill/contract 适用流程。
3.  迁移中途停止并重跑以验证幂等/恢复。
4.  检查旧实例/新实例读写合同。
5.  不可逆清理另求授权。

**正反 Oracle：** 正确：旧数据与中断阶段可按批准策略继续；错误：只测新库空表初次安装便宣称迁移完成。

**出处与限制：** [A04](../external-source-catalog.md)、[A35](../external-source-catalog.md)。该示例是合成教学材料，并非宣称这些出版方已在对应项目上执行。真正可验的 Task 判断必须由受批准的业务 Oracle 和真实测试来源确定。

### P-K09-03 · 局部重构与技术债比较

**合成任务输入：** 输入：一模块函数巨长但没有公开接口变化需求，历史缺陷集中在状态写入。

**逐步操作：**
1.  先定位有来源的真实痛点与用户结果。
2.  用保留行为测试保护现有合同。
3.  小步分离状态转换和 IO。
4.  复验旧用户路径、异常和性能。
5.  若收益不足及时停止扩大范围。

**正反 Oracle：** 正确：降低真实维护风险且旧行为不变；错误：为了美观全仓重写并顺带改变业务规则。

**出处与限制：** [A23](../external-source-catalog.md)、[A41](../external-source-catalog.md)。该示例是合成教学材料，并非宣称这些出版方已在对应项目上执行。真正可验的 Task 判断必须由受批准的业务 Oracle 和真实测试来源确定。

## 相邻知识如何复用

这些外部实践用于增强既有 [engineering-methods.md](engineering-methods.md) 的 3 个 Decision Procedures、[agent-skills.md](agent-skills.md) 的合法 Agent 输入输出、[casebook.md](casebook.md) 的正反案例和 [validation.md](validation.md) 的准确 Oracle。只按实际问题选择选定方法，不把研究 Source 直接提升为已采纳 MUST，不为每个练习引入新的 CI/Host 或独立 Review。
