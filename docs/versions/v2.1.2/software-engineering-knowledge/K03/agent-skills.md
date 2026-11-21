# K03 · Agent Skills / 操作技能与交接

> **v2.1.2 知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #524](https://github.com/kaicreator-mm/tally/issues/524) · [总索引](../README.md)。知识来源锁定 ADS v4.19.0 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；文档设计不代表已获知识资格或实际 Tally Runtime 消费。

## Tally Workflow 选择与实际消费者边界

## 5. Tally Workflow 消费契约

**消费阶段：** L3 Implementation Task DAG-I；使用 Tally 已有 Engineering Business、Task/Attempt 与 Agent/Host 事实源，新增的仅是领域知识与程序的调用关系。

| 环节 | 需要从上游获得 | 该模块实际提供 | 必须保留的判断 |
| --- | --- | --- | --- |
| 任务准入 | 当前合法 Task/写集/L2/来源 | 具体设计切片与局部测试计划 | 合法 Claim/FSW 仍由原有权威持有 |
| 实施 | 稳定接口与使用者 | 代码、错误处理、必要测试 | 不擅扩 Product/Architecture Scope |
| 验证 | 当前 workspace/命令和实际结果 | 语法/类型/构建/受影响行为收据 | 没有执行 = NOT_RUN |
| 集成交付 | 可审查 exact HEAD/真实项目入口 | 受控候选给 Review/Integration | Builder 声明不是 Release PASS |

**跨层交接/复用：** K04 负责 DAG-I 分解和共享写冲突，K05 提供规范测试设计与真实测试证据，K06 对失败进行根因修复，K07 的多 Agent 交接确保同一接口版本与 Review 来源。

## 现有 Agent Skills 的输入输出与拒绝路径

## 6. Agent Skills 与任务输入输出

### SK-K03-01 · Detailed Design Slice

- **何时调用**：Task 存在多个重要实现决策或故障路径
- **最小输入**：准确需求、架构、源码/技术栈和明确写集
- **输出结构**：接口/算法/状态/异常/并发、局部正反测试与集成条件
- **自检**：可让另一 Builder 独立实现而不会对核心状态的 Owner 产生歧义
- **禁止的捷径**：对已冻结设计自行重新定义业务规则

### SK-K03-02 · Bounded Code Builder

- **何时调用**：Task 已具有合法执行上下文和所需环境
- **最小输入**：Task Pack、允许写集、真实基础版本与设计/测试
- **输出结构**：合法代码+测试、执行结果、SHA/命令/错误日志、尚未覆盖行为
- **自检**：语法/类型检查、受影响测试是真实执行且覆盖当前目标
- **禁止的捷径**：靠未运行测试、伪造 Test PASS 或隐藏失败完成任务

### SK-K03-03 · Error & Logging Designer

- **何时调用**：对外错误和故障诊断属于本 Task 的行为合同
- **最小输入**：用户角色、错误分类、运行类型与敏感数据约束
- **输出结构**：错误分类表、可操作消息、必要相关 ID 与日志级别
- **自检**：用户能判断下一步动作；同一异常不会泄露令牌或敏感内容
- **禁止的捷径**：把堆栈打印量当作诊断质量

## 可直接作为单次 Agent 任务使用的提示词

> 注意：以下是**方法提示词**，不是写入许可、部署指令或平台强制格式；使用时必须先填写真实 Task/当前版本/输入来源。不得交给 Agent 无界持续执行。

```text
MODULE=K03
KNOWLEDGE_VERSION=TALLY_V2.1.2_SOURCE_CANDIDATE
ADS_EFFECTIVE_SOURCE=933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c
ROLE=SCOPED_ENGINEERING_ANALYST
TASK_SOURCE=<actual issue, current sha and approved scope>

你是受控 L3 Builder，只能改当前 Task 允许的文件。先读 Task、L1/L2 合同、测试布局；列出接口/状态/错误/并发约束，选择最简单可维护实现。实现必要源码和受影响测试后运行当前项目实际有效的语法/类型/build/测试命令，记录退出状态、发现的用例、精确 HEAD 与未验证事项。未经批准不扩大 Product/L2/Task 写集；本地 PASS 不等于 Host、Review 或 Release。

输出时明确分隔 OBSERVED / SOURCE_BACKED / PROPOSED / UNKNOWN / NOT_RUN。
只要求本次问题所需内容；不自造 Product/L2/Claim/Review/Release 权威。
```

## 最小交接清单

- [ ] Task/Issue 与适用版本明确；信息来自可读的现行来源而不是聊天历史猜测。
- [ ] 具体问题、使用了哪些知识和适用条件明确，未加载整仓无关文档。
- [ ] 结果中的已观察事实、推断、建议、未知和测试未执行分开记录。
- [ ] 提交的 Artifact/测试/Review 指向准确 subject；没有合法执行环境时只给分析结果。
- [ ] 无界循环、未获权限的写入或虚构 PASS 被拒绝。