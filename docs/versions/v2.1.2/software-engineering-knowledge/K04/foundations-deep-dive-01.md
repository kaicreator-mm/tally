# K04 · 学科深化 01：依赖图、关键路径与可交付工作分解

> [系统课程](systematic-syllabus.md) · [依赖专题](topics/REAL_DEPENDENCY.md) · [并行写集](topics/PARALLEL_WRITES.md) · [方法](engineering-methods.md)
>
> **ORIGINAL_EXPLANATORY_KNOWLEDGE_CANDIDATE / NOT_SOURCE_CLAIM_QUALIFIED / NO_TASK_DISPATCH**。深化 K04.F2.1/F2.2、K04.F3.1/F3.2/F3.3、K04.F4.2 和 K04.F5.1。候选三视图 DAG-P/A/I 不是当前 ADS 已发布的统一强制协议；正式工作权威始终由当前 Task/Issue/Review Owner 管理。

## 1. Work Unit 应以可检验成果划分

任务边界首先是一个独立可判断的工程结果，不是“一位 Agent 工作一小时”或“一个目录”。输入包括准入来源、具体写入资源、外部依赖、Expected Outcome、验证方式与停止条件。拆分的价值是降低实际等待/冲突，过小的 Task 会增加交接、Review、CI 成本。

例：新增异步作业，包括 API 合同、持久模型、Runner、消费者路径。若持久模型的 schema 未稳定，依赖其字段语义的两个 Builder 不能通过在不同目录编写代码来保证可并行。反过来，文档修正与独立 UI 文案任务没有真实语义前置，就不应仅因计划图看起来整齐而串行。

## 2. 因果边与资源冲突是两种不同约束

**因果依赖** `A -> B`：B 的验收或正确性真正需要 A 的已确定输出。**资源排斥** `WriteSet(A) ∩ WriteSet(B) ≠ ∅`：即使逻辑上彼此独立，也不能在同一共享文件/生成目录中无协调同时写入。只有已批准合同、稳定导出/API、独立写集和明确集成责任时才安全并行。

图 `G=(V,E)` 若用于先后顺序，必须是有向无环图；出现环意味着任务合同或边的方向有问题，不应任意删掉一条边掩盖真实依赖。文档引用不是因果边，GitHub Issue URL 本身也不等于真实 Blocked-by。

## 3. 关键路径推导与错误并行示例

教学 DAG：`A(2) -> C(4) -> E(1)`，`B(3) -> D(2) -> E(1)`；括号是相同环境下的假定耗时单位。忽略资源冲突的理论路径分别为 7 和 6，**最低理论完工时间为 7 单位**，并非两个分支耗时简单相加。如果 C 和 D 争抢同一独占集成资源，真实时间可能增加；如果 A、B 输出存在隐含接口前置，初始图是错误的。

任务多 Agent 数量不是加速比；需计算 `Total = 关键路径执行 + 排队 + 集成冲突 + 审核/验证返工` 的观测成本（K11），而不是按 Agent 数直接除时间。估计值只是计划假设，必须按真实日志更新。

## 4. 有界选择表

| 条件 | 合理工作形态 | 失败条件 |
| --- | --- | --- |
| 一个文件、小范围、低风险且当前有权 | 一次受控改动 | 因“必须画图”延误交付 |
| 多文件但合同稳定、可顺序一次验证 | 有界工作清单 | 每步都开无收益的独立 Issue |
| 真正跨 Writer/版本/关键前置 | 显式依赖与有限并行 | 未声明共享 Schema Owner |
| 需求或 API 仍 UNKNOWN | 分离可证伪研究与受权实施 | 用未决设计生成硬性依赖 |
| 特定 CI/Host 平台不可替代 | 明示真实验证工作 | 用 Mock 冒充真实环境 PASS |

## 5. 验证与止损

正例：共同 Schema 经有权审定后，两个独立消费者在不冲突写集下并行，最后单一 Integrator 以当前 SHA 验证。反例：三个 Agent 同时修改锁文件，虽然各自测试绿色，最终合并产物缺失某个依赖，不能根据各分支 GREEN 宣称集成 PASS。遇到环、写集竞争、Contract 未冻结或旧 HEAD 结果，先重规划受影响部分并保留证据，不启动无界复跑。

**参考学科来源**：[A04 软件生命周期](../external-source-catalog.md)、[A20 工程实践](../external-source-catalog.md)、[A23 GitLab](../external-source-catalog.md)、[A37 CMMI](../external-source-catalog.md)、[A38 Scrum](../external-source-catalog.md)。这些是可选知识，不改变生效 ADS 原合同。