# K02 · 架构与系统设计：权威实践项目与可证伪工作法

> [K02 知识库](README.md) · [v2.1.2 #521](https://github.com/kaicreator-mm/tally/issues/521) · [外部来源总目录](../external-source-catalog.md)。**EXTERNAL_DISCIPLINARY_SOURCE_CANDIDATE / NOT_SOURCE_QUALIFIED / NOT_CONNECTED**；独立原创方法总结，非 ISO/IEEE/书籍原文转录，不授予 Tally Package/Host/Task 资格。

## 为什么不直接复制 GitHub 仓库

开源项目里的代码/测试/流程是**在其原始平台和约束下成立的实例**。转成 Tally Knowledge 时必须提炼真实技术目标、输入/输出、适用性、成本、反例。实践可读或 Repo MIT 并不意味着第三方数据/模型/书籍也可被无限复制；此处仅记录原创分析及链接，不复制大段受版权保护的文本。

### P-K02-01 · ATAM 风格的轻量取舍

**合成教学任务输入：** 输入：多租户异步导入任务，重启后不能丢作业。

**有界方法：**
1. 列性能/可靠性/隔离关键质量场景。
2. 比较进程内队列与持久化队列。
3. 指出重启恢复是选型敏感点。
4. 设计断电/重复投递负例。
5. 用 ADR 记录理由和未测范围。

**可证伪正/负结果：** 正确：进程内 Map Demo 不能证明恢复，持久队列还要评估副作用幂等和运维成本；错误：吞吐快便等于可靠。

**适用性和出处：** [A06](../external-source-catalog.md)、[A15](../external-source-catalog.md)、[A16](../external-source-catalog.md)。这是将专业概念**重新表述成独立教学案例**，不代表该机构真实执行了本案例；正式 Task 的 Oracle 必须从用户批准的要求与现场测试得出。

### P-K02-02 · C4 与接口合同联合交接

**合成教学任务输入：** 输入：API、Worker、Web UI 三个模块由不同 Builder 实施。

**有界方法：**
1. 画 Context/Container 只明确边界与调用者。
2. 定义实际的 Job 状态 Owner。
3. 单独写 API/消息/错误/版本合同。
4. 冻结消费者所需的最低接口。
5. 合入后实测公开路径。

**可证伪正/负结果：** 正确：视图与接口/状态 Owner 一致，且三者可测试；错误：绘制 Container 图就认为数据迁移和错误合同无需定义。

**适用性和出处：** [A06](../external-source-catalog.md)、[A19](../external-source-catalog.md)、[A18](../external-source-catalog.md)。这是将专业概念**重新表述成独立教学案例**，不代表该机构真实执行了本案例；正式 Task 的 Oracle 必须从用户批准的要求与现场测试得出。

### P-K02-03 · 研究 Demo 的外推边界

**合成教学任务输入：** 输入：本地 Mock 消息服务吞吐达到目标，真实网络和持久 DB 尚未测。

**有界方法：**
1. 在 Demo 开始前列可证伪的候选假设。
2. 记录完整环境/负载与测量。
3. 将成功归类为该 Demo 范围内成立。
4. 针对持久化/断网/多实例列追加验证是否影响方案选择。
5. 交回真实 L2 Owner。

**可证伪正/负结果：** 正确：研究通过但真实 Host/Release NOT_RUN；错误：将 Mock 指标直接当正式生产质量。

**适用性和出处：** [A15](../external-source-catalog.md)、[A16](../external-source-catalog.md)、[A17](../external-source-catalog.md)。这是将专业概念**重新表述成独立教学案例**，不代表该机构真实执行了本案例；正式 Task 的 Oracle 必须从用户批准的要求与现场测试得出。

## 与原有技能和案例的关系

现有 [agent-skills.md](agent-skills.md) 的 Skill 输入输出/权限边界、[engineering-methods.md](engineering-methods.md) 的 3 条有界 DP、[casebook.md](casebook.md) 的案例和 [validation.md](validation.md) 的负例保持主工作入口。本文件提供**外部学科/企业实践如何支持其技术选择**，不能重写已发布 ADS 权威。真实知识资格前先选少量有用主题通过 Source→Claim→Applicability→Counterexample→真实消费者验证，不批量激活 5 个主题和 3 个练习。