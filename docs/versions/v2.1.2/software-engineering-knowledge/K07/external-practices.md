# K07 · 多 Agent 工程协作：权威实践与可证伪案例

> [K07 知识库](README.md) · [v2.1.2 #521](https://github.com/kaicreator-mm/tally/issues/521) · [外部来源总目录](../external-source-catalog.md)。**EXTERNAL_DISCIPLINARY_SOURCE_CANDIDATE / NOT_SOURCE_QUALIFIED / NOT_CONNECTED**；本文件为独立原创工程知识与实际方法的总结，非受版权保护的国际标准/教材原文复刻，不提升现有 Task/Release 权威。

## 从公开实践提炼有效技术，而非复制别人的流程

这里的实践案例是独立构造的**教学输入**，用外部权威/成熟项目的思想解释 Agent 应如何作决策。公开 GitHub 项目的原始能力受技术栈、依赖、执行环境和许可约束；其 README 或 Repo Star 不是验证当前 Tally Task 的权威。相应实例与成熟度都要逐项核对。

### P-K07-01 · 两会话 Reviewer 与单 Builder 交接

**合成任务输入：** 输入：Builder 在候选 SHA `abc123` 完成 API+相关测试，另一 ChatGPT 会话负责独立 Review。

**逐步操作：**
1.  Builder 持久化真实代码与测试，提交当前 HEAD。
2.  Reviewer 读取确切 Diff、来源需求和实际测试。
3.  审核报告记录与写者会话分离的事实。
4.  发现问题交还合法 Writer 修复生成新 SHA。
5.  只对影响范围进行必要复核。

**正反 Oracle：** 正确：Reviewer 与 Writer 的实际工作来源可区分，新 SHA 不继承旧 Reviewer PASS；错误：简单把同一对话输出标题改成‘独立审查’。

**出处与限制：** [A21](../external-source-catalog.md)、[A23](../external-source-catalog.md)。该示例是合成教学材料，并非宣称这些出版方已在对应项目上执行。真正可验的 Task 判断必须由受批准的业务 Oracle 和真实测试来源确定。

### P-K07-02 · IPC ACK 与迟到任务的对账

**合成任务输入：** 输入：远程 A17 根据旧 HEAD `aaa111` 处理，Task/PR 已更新到 `bbb222`，旧 Worker 返回 ACK SUCCESS。

**逐步操作：**
1.  ACK 仅作为消息事件记录。
2.  读取当前 native Task/Claim/PR 事实。
3.  比较返回 Artifact SourceSHA 与当前 SHA。
4.  STALE 结果保持候选隔离并核查是否可复用。
5.  外部副作用未知时读回，而不是重新执行。

**正反 Oracle：** 正确：现行 HEAD/Task 状态不受迟到旧 ACK 影响；错误：收到 SUCCESS 就直接把新 HEAD 标成 PASS。

**出处与限制：** [A23](../external-source-catalog.md)、[A29](../external-source-catalog.md)。该示例是合成教学材料，并非宣称这些出版方已在对应项目上执行。真正可验的 Task 判断必须由受批准的业务 Oracle 和真实测试来源确定。

### P-K07-03 · 有真实利益的 Agent 并行

**合成任务输入：** 输入：本次工作需三个独立模块分析，只有一个共享 Schema 文件可变更。

**逐步操作：**
1.  只读需求/测试/设计分析可多 Agent 并行。
2.  共享 Schema 单 Writer 完成合同与变更。
3.  模块 Builder 在稳定接口后写 disjoint scope。
4.  集成 Agent/合法 Owner 验证当前组合。
5.  记录总等待、Review、冲突修复成本。

**正反 Oracle：** 正确：有实际并行价值且冲突/接口一致；错误：按剩余 Token 而不是成果独立性派八个共享 Writer。

**出处与限制：** [A20](../external-source-catalog.md)、[A24](../external-source-catalog.md)、[A34](../external-source-catalog.md)。该示例是合成教学材料，并非宣称这些出版方已在对应项目上执行。真正可验的 Task 判断必须由受批准的业务 Oracle 和真实测试来源确定。

## 相邻知识如何复用

这些外部实践用于增强既有 [engineering-methods.md](engineering-methods.md) 的 3 个 Decision Procedures、[agent-skills.md](agent-skills.md) 的合法 Agent 输入输出、[casebook.md](casebook.md) 的正反案例和 [validation.md](validation.md) 的准确 Oracle。只按实际问题选择选定方法，不把研究 Source 直接提升为已采纳 MUST，不为每个练习引入新的 CI/Host 或独立 Review。
