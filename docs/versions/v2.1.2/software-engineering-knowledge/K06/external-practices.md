# K06 · 调试与根因修复：权威实践项目与可证伪工作法

> [K06 知识库](README.md) · [v2.1.2 #521](https://github.com/kaicreator-mm/tally/issues/521) · [外部来源总目录](../external-source-catalog.md)。**EXTERNAL_DISCIPLINARY_SOURCE_CANDIDATE / NOT_SOURCE_QUALIFIED / NOT_CONNECTED**；独立原创方法总结，非 ISO/IEEE/书籍原文转录，不授予 Tally Package/Host/Task 资格。

## 为什么不直接复制 GitHub 仓库

开源项目里的代码/测试/流程是**在其原始平台和约束下成立的实例**。转成 Tally Knowledge 时必须提炼真实技术目标、输入/输出、适用性、成本、反例。实践可读或 Repo MIT 并不意味着第三方数据/模型/书籍也可被无限复制；此处仅记录原创分析及链接，不复制大段受版权保护的文本。

### P-K06-01 · 编译失败而非安装失败

**合成教学任务输入：** 输入：依赖安装成功，随后 tsc 指向 `class Error extends Error` 自继承。

**有界方法：**
1. 固定当前源/CI 运行 SHA。
2. 标注 install SUCCESS, build FAIL。
3. 在错误源行识别继承遮蔽。
4. 最小局部修复并执行项目构建/受影响测试。
5. 旧依赖或锁问题不作为当前根因。

**可证伪正/负结果：** 正确：旧失败未自动继承当前根因，build 新 SHA 真实 PASS 才可记录；错误：只升级所有依赖并反复重跑。

**适用性和出处：** [A01](../external-source-catalog.md)、[A28](../external-source-catalog.md)。这是将专业概念**重新表述成独立教学案例**，不代表该机构真实执行了本案例；正式 Task 的 Oracle 必须从用户批准的要求与现场测试得出。

### P-K06-02 · 超时后重复请求状态未知

**合成教学任务输入：** 输入：创建订单操作超时，客户端已发出可重复的请求但没有收到回包。

**有界方法：**
1. 区分连接超时与服务器已提交可能性。
2. 寻找原生幂等 token/请求 ID。
3. 查询有权状态事实。
4. 在可安全重放时采用有限重试。
5. 没有读回能力时保留 UNKNOWN。

**可证伪正/负结果：** 正确：不会重复创建多张订单；错误：每次超时后立即创建新订单并称这是恢复。

**适用性和出处：** [A31](../external-source-catalog.md)、[A32](../external-source-catalog.md)。这是将专业概念**重新表述成独立教学案例**，不代表该机构真实执行了本案例；正式 Task 的 Oracle 必须从用户批准的要求与现场测试得出。

### P-K06-03 · 仅测试发现错误而非业务逻辑错

**合成教学任务输入：** 输入：CI runner report 0 tests，任务宣称新负例已通过。

**有界方法：**
1. 检查 runner 匹配规则、后缀和目录。
2. 在相同源码/版本下证明新增测试被发现。
3. 运行原错误负例。
4. 若测试依然没运行则 NOT_RUN。
5. 不修改业务实现去配合空测试结果。

**可证伪正/负结果：** 正确：把问题归测试执行/发现，构建/发布状态单独判断；错误：把空测试 green 作为回归合格。

**适用性和出处：** [A14](../external-source-catalog.md)、[A27](../external-source-catalog.md)。这是将专业概念**重新表述成独立教学案例**，不代表该机构真实执行了本案例；正式 Task 的 Oracle 必须从用户批准的要求与现场测试得出。

## 与原有技能和案例的关系

现有 [agent-skills.md](agent-skills.md) 的 Skill 输入输出/权限边界、[engineering-methods.md](engineering-methods.md) 的 3 条有界 DP、[casebook.md](casebook.md) 的案例和 [validation.md](validation.md) 的负例保持主工作入口。本文件提供**外部学科/企业实践如何支持其技术选择**，不能重写已发布 ADS 权威。真实知识资格前先选少量有用主题通过 Source→Claim→Applicability→Counterexample→真实消费者验证，不批量激活 5 个主题和 3 个练习。