# K04 · 任务规划与分解：权威实践项目与可证伪工作法

> [K04 知识库](README.md) · [v2.1.2 #521](https://github.com/kaicreator-mm/tally/issues/521) · [外部来源总目录](../external-source-catalog.md)。**EXTERNAL_DISCIPLINARY_SOURCE_CANDIDATE / NOT_SOURCE_QUALIFIED / NOT_CONNECTED**；独立原创方法总结，非 ISO/IEEE/书籍原文转录，不授予 Tally Package/Host/Task 资格。

## 为什么不直接复制 GitHub 仓库

开源项目里的代码/测试/流程是**在其原始平台和约束下成立的实例**。转成 Tally Knowledge 时必须提炼真实技术目标、输入/输出、适用性、成本、反例。实践可读或 Repo MIT 并不意味着第三方数据/模型/书籍也可被无限复制；此处仅记录原创分析及链接，不复制大段受版权保护的文本。

### P-K04-01 · 从用户行为做垂直切片

**合成教学任务输入：** 输入：新增一条订单导出 API 与 UI 按钮，存在共享数据合同。

**有界方法：**
1. 固定用户验收。
2. 先定义公共响应/错误合同。
3. 若技术职责可独立验收，适度拆成 API/界面/集成任务。
4. 源码/测试同一真实 Owner。
5. 按真实依赖合并并验证用户路径。

**可证伪正/负结果：** 正确：有稳定 seam 和完整验收；错误：按文件层把 API/校验/测试分成六个不能独立完成的 Task。

**适用性和出处：** [A20](../external-source-catalog.md)、[A24](../external-source-catalog.md)。这是将专业概念**重新表述成独立教学案例**，不代表该机构真实执行了本案例；正式 Task 的 Oracle 必须从用户批准的要求与现场测试得出。

### P-K04-02 · 任务图反事实审查

**合成教学任务输入：** 输入：A 是新增非核心文档，B 是 API 合同测试，C 是 UI 功能。Planner 将所有任务串行依赖 A→B→C。

**有界方法：**
1. 为每条边列出 B/C 必须消费的真实产物。
2. A 若不影响 API 合同应删 A→B。
3. UI 只需稳定接口，可按实际契约决定 B/C 是否并行。
4. 对公共生成文件指定单 Writer。
5. 保留必要集成节点。

**可证伪正/负结果：** 正确：只剩有因果的依赖边；错误：因工作流‘规范完整’就保留无关阻塞。

**适用性和出处：** [A04](../external-source-catalog.md)、[A24](../external-source-catalog.md)。这是将专业概念**重新表述成独立教学案例**，不代表该机构真实执行了本案例；正式 Task 的 Oracle 必须从用户批准的要求与现场测试得出。

### P-K04-03 · 研究层与执行层分离

**合成教学任务输入：** 输入：方案仍需确认是否采用持久化队列，实施 Task 尚未确定。

**有界方法：**
1. 用现有 L2 Research/Demo Owner 完成可证伪评估。
2. 研究结论标来源与待选决策。
3. 合法设计确认后再组织实施 Task DAG。
4. 不从候选 DAG-A 生成自动 Claim。
5. 需要变更设计时只重规划受影响部分。

**可证伪正/负结果：** 正确：Demo 通过不等于 L2/Task Freeze；错误：技术研究 Issue 绿灯就派有写权的 Builder。

**适用性和出处：** [A01](../external-source-catalog.md)、[A04](../external-source-catalog.md)、[A39](../external-source-catalog.md)。这是将专业概念**重新表述成独立教学案例**，不代表该机构真实执行了本案例；正式 Task 的 Oracle 必须从用户批准的要求与现场测试得出。

## 与原有技能和案例的关系

现有 [agent-skills.md](agent-skills.md) 的 Skill 输入输出/权限边界、[engineering-methods.md](engineering-methods.md) 的 3 条有界 DP、[casebook.md](casebook.md) 的案例和 [validation.md](validation.md) 的负例保持主工作入口。本文件提供**外部学科/企业实践如何支持其技术选择**，不能重写已发布 ADS 权威。真实知识资格前先选少量有用主题通过 Source→Claim→Applicability→Counterexample→真实消费者验证，不批量激活 5 个主题和 3 个练习。