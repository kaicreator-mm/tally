# K05 · 软件测试与质量验证：权威实践项目与可证伪工作法

> [K05 知识库](README.md) · [v2.1.2 #521](https://github.com/kaicreator-mm/tally/issues/521) · [外部来源总目录](../external-source-catalog.md)。**EXTERNAL_DISCIPLINARY_SOURCE_CANDIDATE / NOT_SOURCE_QUALIFIED / NOT_CONNECTED**；独立原创方法总结，非 ISO/IEEE/书籍原文转录，不授予 Tally Package/Host/Task 资格。

## 为什么不直接复制 GitHub 仓库

开源项目里的代码/测试/流程是**在其原始平台和约束下成立的实例**。转成 Tally Knowledge 时必须提炼真实技术目标、输入/输出、适用性、成本、反例。实践可读或 Repo MIT 并不意味着第三方数据/模型/书籍也可被无限复制；此处仅记录原创分析及链接，不复制大段受版权保护的文本。

### P-K05-01 · 状态机与决策表设计

**合成教学任务输入：** 输入：单据根据角色/状态允许或拒绝执行取消操作。

**有界方法：**
1. 固定批准的角色/状态/动作规则。
2. 为条件组合形成决策表。
3. 选择至少一个允许和一个不允许的状态转移。
4. 通过公开路径验证业务结果及无副作用。
5. 对重复请求/超时增加必要负例。

**可证伪正/负结果：** 正确：错误实现真的让至少一个负例 FAIL；错误：只检查 Mock 函数被调用。

**适用性和出处：** [A08](../external-source-catalog.md)、[A14](../external-source-catalog.md)。这是将专业概念**重新表述成独立教学案例**，不代表该机构真实执行了本案例；正式 Task 的 Oracle 必须从用户批准的要求与现场测试得出。

### P-K05-02 · 消费者合同与真实数据库分层

**合成教学任务输入：** 输入：API 返回对象在新版本增加字段，旧 Web 客户端依赖旧 JSON。

**有界方法：**
1. Consumer contract 检查旧字段和错误码。
2. 实际 DB 读写路径用真实依赖做必要集成。
3. 测试 runner 证明相关用例被发现。
4. Selected Public Path 进一步核对真实用户调用。
5. 按各层证据分别记录。

**可证伪正/负结果：** 正确：Unit/Contract/Integration/Host Scope 分明；错误：只跑一个 mock test 就标 E2E SUCCESS。

**适用性和出处：** [A25](../external-source-catalog.md)、[A26](../external-source-catalog.md)。这是将专业概念**重新表述成独立教学案例**，不代表该机构真实执行了本案例；正式 Task 的 Oracle 必须从用户批准的要求与现场测试得出。

### P-K05-03 · Agent 补丁基准双层评价

**合成教学任务输入：** 输入：两个 Agent 对同一个源码 Issue 生成 Patch，一个只依据 prompt，另一个参考选中 K05 知识。

**有界方法：**
1. 固定 repo/base commit/issue/test harness。
2. 两组预算和权限一致。
3. 用同样的实际 tests 对比修复效果。
4. 独立 Reviewer 检查新增回归、Oracle 真伪与实际消费者。
5. 报告未测/负结果与外推范围。

**可证伪正/负结果：** 正确：有样本内修复结果但不声称已具备完整 Release 能力；错误：只有一个 hidden test 绿便宣传全领域提升。

**适用性和出处：** [A09](../external-source-catalog.md)、[A27](../external-source-catalog.md)。这是将专业概念**重新表述成独立教学案例**，不代表该机构真实执行了本案例；正式 Task 的 Oracle 必须从用户批准的要求与现场测试得出。

## 与原有技能和案例的关系

现有 [agent-skills.md](agent-skills.md) 的 Skill 输入输出/权限边界、[engineering-methods.md](engineering-methods.md) 的 3 条有界 DP、[casebook.md](casebook.md) 的案例和 [validation.md](validation.md) 的负例保持主工作入口。本文件提供**外部学科/企业实践如何支持其技术选择**，不能重写已发布 ADS 权威。真实知识资格前先选少量有用主题通过 Source→Claim→Applicability→Counterexample→真实消费者验证，不批量激活 5 个主题和 3 个练习。