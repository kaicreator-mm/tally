# K03 · 详细设计与代码实现：权威实践项目与可证伪工作法

> [K03 知识库](README.md) · [v2.1.2 #521](https://github.com/kaicreator-mm/tally/issues/521) · [外部来源总目录](../external-source-catalog.md)。**EXTERNAL_DISCIPLINARY_SOURCE_CANDIDATE / NOT_SOURCE_QUALIFIED / NOT_CONNECTED**；独立原创方法总结，非 ISO/IEEE/书籍原文转录，不授予 Tally Package/Host/Task 资格。

## 为什么不直接复制 GitHub 仓库

开源项目里的代码/测试/流程是**在其原始平台和约束下成立的实例**。转成 Tally Knowledge 时必须提炼真实技术目标、输入/输出、适用性、成本、反例。实践可读或 Repo MIT 并不意味着第三方数据/模型/书籍也可被无限复制；此处仅记录原创分析及链接，不复制大段受版权保护的文本。

### P-K03-01 · 实现一个合法状态转换

**合成教学任务输入：** 输入：业务已确认仅 OPEN 可转 CLOSED，其他状态应拒绝。

**有界方法：**
1. 定义输入/输出/错误及状态表。
2. 选择现有事务/并发控制，避免多处写入 Owner。
3. 实现成功与非法状态路径。
4. 加入重复调用/事务回归。
5. 在当前写集运行有效编译与行为测试。

**可证伪正/负结果：** 正确：重复和非法状态保持不变量；错误：测试只检查 handler 被调用，不核实真实持久化行为。

**适用性和出处：** [A01](../external-source-catalog.md)、[A20](../external-source-catalog.md)。这是将专业概念**重新表述成独立教学案例**，不代表该机构真实执行了本案例；正式 Task 的 Oracle 必须从用户批准的要求与现场测试得出。

### P-K03-02 · 可维护性与复杂抽象取舍

**合成教学任务输入：** 输入：两个模块有相同 7 行数据转换，已有稳定接口。

**有界方法：**
1. 比较复制维护风险、抽象引入额外耦合、未来变更频率。
2. 若共享语义稳定选择单一共享方法。
3. 若语义不同则保留局部实现。
4. 做当前行为回归。
5. 记录判断适用条件。

**可证伪正/负结果：** 正确：以语义稳定和未来改动为依据，而非单纯代码行数；错误：盲目创建通用跨应用插件体系。

**适用性和出处：** [A20](../external-source-catalog.md)、[A41](../external-source-catalog.md)。这是将专业概念**重新表述成独立教学案例**，不代表该机构真实执行了本案例；正式 Task 的 Oracle 必须从用户批准的要求与现场测试得出。

### P-K03-03 · CLI 和后台错误合同

**合成教学任务输入：** 输入：后台服务任务超时，但消费者不知道最终效果。

**有界方法：**
1. 区分业务/配置/外部暂时/内部错误。
2. 标出可能已提交的 UNKNOWN。
3. 日志有安全关联 ID 和可行动信息。
4. 按是否幂等有界重试/回读。
5. 验证错误格式与公开消费者。

**可证伪正/负结果：** 正确：有状态回读和可恢复错误，不泄漏令牌；错误：只打印堆栈或无条件重新执行。

**适用性和出处：** [A20](../external-source-catalog.md)、[A32](../external-source-catalog.md)。这是将专业概念**重新表述成独立教学案例**，不代表该机构真实执行了本案例；正式 Task 的 Oracle 必须从用户批准的要求与现场测试得出。

## 与原有技能和案例的关系

现有 [agent-skills.md](agent-skills.md) 的 Skill 输入输出/权限边界、[engineering-methods.md](engineering-methods.md) 的 3 条有界 DP、[casebook.md](casebook.md) 的案例和 [validation.md](validation.md) 的负例保持主工作入口。本文件提供**外部学科/企业实践如何支持其技术选择**，不能重写已发布 ADS 权威。真实知识资格前先选少量有用主题通过 Source→Claim→Applicability→Counterexample→真实消费者验证，不批量激活 5 个主题和 3 个练习。