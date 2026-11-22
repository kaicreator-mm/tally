# K04 进阶任务工程：真实偏序、写集冲突、关键路径与交付边界

> [K04 课程](systematic-syllabus.md) · [真实依赖 DAG 第一卷](foundations-deep-dive-01.md) · [估算反馈第二卷](foundations-deep-dive-02.md) · [接受/权威 Focus](focus-f2-3-task-acceptance-and-authority.md)
>
> **R6 ORIGINAL_ENGINEERING_THEORY_CANDIDATE / NO_ACTUAL_TASK_DISPATCH / NO_NATIVE_STATUS_CREATED。** 重点 K04.F2.1、F3.1、F3.2、F3.3、F4.2、F5.1、F5.3。DAG 是可选的项目计划表示；不能将“两个任务需要相同库”直接当成因果边，也不必把每项工作无限细分。

## 一、Task 图不是任务名称的序号

计划是偏序（partial order）：`A → B` 意味着 B 必须消费 A 完成后的明确成果、先决合同或资格。可行顺序必须满足**实际前提**，与 issue 创建顺序、模块号、谁先提交没有必然关系。Task 的基本候选信息至少应有：
`Goal, Input/SourceSHA, Outputs, WriteSet, ReadSet, AcceptanceOracle, Dependencies, CurrentOwner`。这些字段是分析维度，不是新 Tally Task Schema。

**图的错误例子**：“K01 必须先于 K02，K02 必须先于 K03”，会把11模块强制成串行；另一种错误是“所有 Writer 可以同时改共享 schema”，忽视真实竞争。K04 的职责是从读/写集、接口版本和消费者事实推导依赖，而非从学科分类推导审批顺序。

## 二、极小 DAG 演算与关键路径

示例任务（时长为教学假定，不是测得的流水线）：

| Task | 内容 | 必需前件 | 估时 |
| --- | --- | --- | --- |
| A | 明确 API contract | 无 | 2h |
| B | Implement server | A | 4h |
| C | Implement client | A | 3h |
| D | Contract integration test | B、C | 2h |
| E | 独立文档草案 | 无 | 1h |

**依赖偏序**：A→B→D 与 A→C→D，E 可在不触碰 D 的真实业务合同条件下并行。理想无限资源的最短关键链是 A+B+D=8h，不是所有时长 12h 相加。B/C 并行、E 并行，才可能达到 8h；若只有一个 Builder 能写 B 与 C，共享执行资源可能要求顺序，实际用时大于理论下界。

教材式最早完成计算：`EF(task) = duration(task) + max(EF(predecessors))`，空前件的 `max` 取零。**注意**：该公式需要静态估时、固定图、无资源抢占/互斥，不能将其运行在多 Agent 不稳定耗时上就称为真实交付预测。

## 三、读写集比“文件名不重复”更接近真实冲突

假设 B 改服务端 schema，C 改依赖该 schema 的客户端。即使它们修改不同文件，也有**语义冲突**：C 的编译和测试必须消费一致接口版本。解决方式之一是在 A 冻结最小接口合同后 B/C 并行，D 消费同一 Exact SHA 下集成后的接口；或由共享 Owner 在 D 统一仲裁版本，不是用不同分支直接互相覆盖。

共享写集举例：两个 Agent 都修改根 `package.json`，即使代码路径不同，Git 合并容易冲突；可以将版本变更拆给单一负责者，同时让两个实现者以声明的依赖版本工作。**没有共享文件**也不意味着没有事务、部署配置、消费者或权限语义冲突。

## 四、不确定性、重规划与停止过度规划

如果 A 的合同仍是 UNKNOWN，可以允许 B/C 在 Mock 契约或明确的暂定接口上进行可逆原型，但不能把这当“最终集成通过”。计划应写出：
- 当前事实：哪些输入已获授权，哪些只能作假设；
- 依赖类型：实质数据依赖、合法权限依赖、共享状态/写集、资源互斥；
- 决策点：如果 A 变化，哪几个 Task/Oracle 受影响；
- 停止条件：何时继续细化成本超过新信息价值，转为最小实现与真实反馈。

关键路径随 B 的实际运行时长、测试失败及共享资源变化而变化。将每个临时失败自动转成新增十个审批/Issue 不会提高交付质量；可先在当前 Task 完成有边界的修复、验证与合并，只有真实独立交付/权限冲突才拆分。

## 五、Task DONE、验证收据和有权接受的不同事实

Worker 消息 `DONE` 只证明 Worker 声称结束；GitHub 具体文件/commit 证明提交物存在；CI或本地受影响测试的实际执行证明相应范围；正式 Task/Review/Release 是否接受，必须服从当前项目的有权流程。旧 `H1` 的独立审查不会因 H2 只改无关说明文字自动失效，但 H2 触及共享接口时，应该从读/写集确定**材料影响闭包**，做必要增量验证。

一个项目已在等价环境做完受影响测试，不必为了制造“CI 二次通过”重复无意义执行；如果真正强制的 GitHub Check/Provider/Host 环境存在，仍需满足。K04 不把所有建议推广成无条件额外 Gate。

## 六、工程判断练习

题目：A 固定合同；B 服务端；C 客户端；D 集成；E 文档；两个 Agent 声称“所有文件不冲突，所以五个任务可全并行”。回答应明确：
1. **必须保留** A→B、A→C、B→D、C→D 的因果边，以及 D 的当前合同/源码身份；
2. E 与哪些任务能并行取决于 E 是否消费已冻结合同，而非仅取决于文档后缀；
3. 如果 B/C 使用不同版本 A，至少 D 不能被宣称通过；
4. 若某个 Agent 在旧 HEAD 迟到，ACK 不晋升当前 Task/Review；
5. 即使资源可并行也要区分下界 8h 与实际墙钟时间。

**STOP**：未明确合同变化时只暂停受影响交付资格，而不是暂停全部无关工作。图中时长无任何真实执行数据，`NOT_MEASURED`。

**参考线索**：[A04 ISO 12207](../external-source-catalog.md)、[A20 Microsoft Engineering Playbook](../external-source-catalog.md)、[A23 GitLab Engineering](../external-source-catalog.md)、[A24 Modular Code](../external-source-catalog.md)、[A38 Scrum Guide](../external-source-catalog.md)。这些只是适用比较来源，非新项目规范。