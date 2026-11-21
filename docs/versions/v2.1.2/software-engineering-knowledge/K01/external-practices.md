# K01 · 需求工程与业务建模：权威实践项目与可证伪工作法

> [K01 知识库](README.md) · [v2.1.2 #521](https://github.com/kaicreator-mm/tally/issues/521) · [外部来源总目录](../external-source-catalog.md)。**EXTERNAL_DISCIPLINARY_SOURCE_CANDIDATE / NOT_SOURCE_QUALIFIED / NOT_CONNECTED**；独立原创方法总结，非 ISO/IEEE/书籍原文转录，不授予 Tally Package/Host/Task 资格。

## 为什么不直接复制 GitHub 仓库

开源项目里的代码/测试/流程是**在其原始平台和约束下成立的实例**。转成 Tally Knowledge 时必须提炼真实技术目标、输入/输出、适用性、成本、反例。实践可读或 Repo MIT 并不意味着第三方数据/模型/书籍也可被无限复制；此处仅记录原创分析及链接，不复制大段受版权保护的文本。

### P-K01-01 · 需求描述到可观察验收

**合成教学任务输入：** 输入：用户说“管理员可以批量关闭工单”，未说明部分失败；当前已批准状态为 OPEN/ACTIVE/CLOSED。

**有界方法：**
1. 从需求确认操作角色与范围。
2. 提议 CLOSED 后不可再关闭但标明待批准。
3. 设计单条与批量的正常/失败/非法状态。
4. 将部分成功/整批原子性列为必须决策。
5. 关联 L2 数据一致性策略。

**可证伪正/负结果：** 正确：有来源的角色/状态与一条禁止路径，未知批处理语义保持 UNKNOWN。错误：直接实现全部原子成功且声称用户已同意。

**适用性和出处：** [A05](../external-source-catalog.md)、[A11](../external-source-catalog.md)。这是将专业概念**重新表述成独立教学案例**，不代表该机构真实执行了本案例；正式 Task 的 Oracle 必须从用户批准的要求与现场测试得出。

### P-K01-02 · 质量场景驱动非功能需求

**合成教学任务输入：** 输入：用户说“导入不能卡住 UI”，没有给吞吐数字。

**有界方法：**
1. 将“卡住”拆为界面响应和作业可取消/进度。
2. 给出负载场景候选，明确未经验证的时间指标。
3. 把 UI/Worker 边界变为 L2 驱动。
4. 验证用户能在任务运行时继续操作界面。
5. 保留真实性能基线未采集。

**可证伪正/负结果：** 正确：可观察的 UI 响应行为和必要 Demo；错误：编造吞吐/延迟 SLA 并要求生产服务架构。

**适用性和出处：** [A07](../external-source-catalog.md)、[A16](../external-source-catalog.md)。这是将专业概念**重新表述成独立教学案例**，不代表该机构真实执行了本案例；正式 Task 的 Oracle 必须从用户批准的要求与现场测试得出。

### P-K01-03 · 相互冲突的业务资料

**合成教学任务输入：** 输入：旧手册说 PENDING 可取消，新版 Product Issue 说“发货后不可取消”，而发货状态定义没有关联。

**有界方法：**
1. 固定来源版本、角色、订单状态与商品物流边界。
2. 判断是否适用不同时间窗。
3. 只有不能分解的冲突才交 Owner。
4. 为合法和非法路径创建验收候选。
5. 不把旧资料删除。

**可证伪正/负结果：** 正确：有权来源覆盖范围可核，无法确定时 HOLD 对应决议；错误：因新版日期较新便不检查适用范围，直接改所有订单。

**适用性和出处：** [A03](../external-source-catalog.md)、[A05](../external-source-catalog.md)、[A12](../external-source-catalog.md)。这是将专业概念**重新表述成独立教学案例**，不代表该机构真实执行了本案例；正式 Task 的 Oracle 必须从用户批准的要求与现场测试得出。

## 与原有技能和案例的关系

现有 [agent-skills.md](agent-skills.md) 的 Skill 输入输出/权限边界、[engineering-methods.md](engineering-methods.md) 的 3 条有界 DP、[casebook.md](casebook.md) 的案例和 [validation.md](validation.md) 的负例保持主工作入口。本文件提供**外部学科/企业实践如何支持其技术选择**，不能重写已发布 ADS 权威。真实知识资格前先选少量有用主题通过 Source→Claim→Applicability→Counterexample→真实消费者验证，不批量激活 5 个主题和 3 个练习。