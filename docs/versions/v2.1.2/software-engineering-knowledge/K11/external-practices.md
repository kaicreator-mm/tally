# K11 · 工程经济学与决策：权威实践与可证伪案例

> [K11 知识库](README.md) · [v2.1.2 #521](https://github.com/kaicreator-mm/tally/issues/521) · [外部来源总目录](../external-source-catalog.md)。**EXTERNAL_DISCIPLINARY_SOURCE_CANDIDATE / NOT_SOURCE_QUALIFIED / NOT_CONNECTED**；本文件为独立原创工程知识与实际方法的总结，非受版权保护的国际标准/教材原文复刻，不提升现有 Task/Release 权威。

## 从公开实践提炼有效技术，而非复制别人的流程

这里的实践案例是独立构造的**教学输入**，用外部权威/成熟项目的思想解释 Agent 应如何作决策。公开 GitHub 项目的原始能力受技术栈、依赖、执行环境和许可约束；其 README 或 Repo Star 不是验证当前 Tally Task 的权威。相应实例与成熟度都要逐项核对。

### P-K11-01 · 小变更是否有必要多 Agent

**合成任务输入：** 输入：单个 UI 文案变更，现有受影响测试已经可运行且没有架构改动。

**逐步操作：**
1.  明确用户结果和风险底线。
2.  比较 1 个 Builder INLINE 与 6 个 Agent 多 Task 方案。
3.  计算新增交接/Review/集成/CI 成本。
4.  验证必要实际测试。
5.  没有收益则停止额外任务。

**正反 Oracle：** 正确：交付快、真实验收且无不必要并行；错误：把完成六个 Issue 当成更高价值。

**出处与限制：** [A20](../external-source-catalog.md)、[A40](../external-source-catalog.md)。该示例是合成教学材料，并非宣称这些出版方已在对应项目上执行。真正可验的 Task 判断必须由受批准的业务 Oracle 和真实测试来源确定。

### P-K11-02 · 高风险小改动保留保证

**合成任务输入：** 输入：三行数据库迁移涉及不可逆删除旧数据。

**逐步操作：**
1.  识别不可逆后果和受影响消费者。
2.  确认最小有权 L2/迁移决策。
3.  准备备份/恢复/失败验证（按实际系统）。
4.  测试真实受影响升级路径。
5.  以真实收据而非 LOC 判断充分。

**正反 Oracle：** 正确：低代码行数不降低实际保障；错误：以‘变化只有三行’跳过迁移测试和 Owner 权限。

**出处与限制：** [A04](../external-source-catalog.md)、[A10](../external-source-catalog.md)、[A37](../external-source-catalog.md)。该示例是合成教学材料，并非宣称这些出版方已在对应项目上执行。真正可验的 Task 判断必须由受批准的业务 Oracle 和真实测试来源确定。

### P-K11-03 · Agent 知识介入效果的公平对照

**合成任务输入：** 输入：选择五个真实小任务，其中两个含状态机负例，Tally 候选知识库尚未通过生产资格。

**逐步操作：**
1.  冻结任务、测试/验收和允许工具。
2.  两种 Agent 策略随机或配对分组，保持环境一致。
3.  独立 Reviewer 看真实代码/缺陷/正确性。
4.  收集 Token/CI/Review/修复成本与实际时间。
5.  报告成功和失败、样本限制，不扩大为总体结论。

**正反 Oracle：** 正确：仅报告当前样本的可信对比，知识没合格不自动生产使用；错误：模型问答提升便宣称软件工程质量普遍提高。

**出处与限制：** [A27](../external-source-catalog.md)、[A33](../external-source-catalog.md)、[A34](../external-source-catalog.md)、[A40](../external-source-catalog.md)。该示例是合成教学材料，并非宣称这些出版方已在对应项目上执行。真正可验的 Task 判断必须由受批准的业务 Oracle 和真实测试来源确定。

## 相邻知识如何复用

这些外部实践用于增强既有 [engineering-methods.md](engineering-methods.md) 的 3 个 Decision Procedures、[agent-skills.md](agent-skills.md) 的合法 Agent 输入输出、[casebook.md](casebook.md) 的正反案例和 [validation.md](validation.md) 的准确 Oracle。只按实际问题选择选定方法，不把研究 Source 直接提升为已采纳 MUST，不为每个练习引入新的 CI/Host 或独立 Review。
