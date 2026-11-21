# K03 · Domain Model / 核心概念和语义

> **v2.1.2 知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #524](https://github.com/kaicreator-mm/tally/issues/524) · [总索引](../README.md)。知识来源锁定 ADS v4.19.0 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；文档设计不代表已获知识资格或实际 Tally Runtime 消费。

## 概念定义、混淆与原文分析

## 2. 领域概念与边界

| 概念 | 精确定义及判断要点 | 常见误用 |
| --- | --- | --- |
| **Task-level Contract** | 从 L1/L2 得到的具体输入/输出、前置/后置、错误、允许代码范围 | 仅用 Issue 标题作为实现合同 |
| **Invariant / State Transition** | 改变状态时必须保持的约束和合法事件序列 | 只考虑成功路径，忽视部分失败 |
| **Algorithm / Complexity** | 为当前规模选择的数据结构及复杂度/内存/时间成本 | 将理论最优误认成总成本最低 |
| **Concurrency / Idempotency** | 处理并行、重复、迟到、超时或重试时保持语义正确 | 增加 retry 循环便自称可靠 |
| **Error Contract** | 可行动错误分类、因果信息、恢复/取消/重试能力 | 吞掉异常或只报“失败” |
| **Code Ownership / Write Set** | 当前 Task 合法可变更路径和共享文件边界 | 两个 Writer 修改同一生成物 |
| **Definition of Done** | 实际行为、构建/测试/集成和 Review 证据完整性 | 代码提交或 CI 绿灯等同交付完成 |
| **Source Currentness** | 实现/测试/Review 对应的准确版本和依赖状态 | 用旧的测试结果证明新 HEAD |

不同概念之间通过具体的来源/适用性/决策/消费者连接，不必为每一个词建立全新的数据库表。只有在真实 Tally 任务中被调用且能够改善工程判断时，才有必要形成可选的结构化资产。

## 与其他知识的语义边界

### 关系 1

Accepted Contract → Task Design → Implementation/Errors → Affected Test Oracle → Candidate SHA；没有可调用路径的源码文件不足以证明功能存在。

### 关系 2

业务非法状态与程序异常不同；超时表示未收到结果而不必然意味着业务没有提交。

### 关系 3

日志质量由应用形态和可行动诊断决定；K10 提供通用恢复模式，K03 负责当前代码落实。

## 知识推断的可靠性分级

| 等级 | 可以断言的内容 | 不能断言的内容 |
| --- | --- | --- |
| Published ADS source | 当前适用源/版本中的明确规则、权威/权限边界 | 所有由本模块新增的便利推论都成为 MUST |
| Source-backed interpretation | 有清楚来源与适用范围的工程解释/方法 | 跨项目普遍效力或生产资格 |
| Teaching example | 在给定假设下的可证伪示例 | 真实 Agent/Host/项目结果 |
| UNKNOWN | 缺少权威来源、当前性或真实验证 | PASS、正式 Product/L2/Task/Release 状态 |

## Agent 使用时的五个提问

1. 输入输出、状态、不变量和非法请求行为是什么？
2. 哪种算法和数据结构足以满足当前规模？
3. 当前修改是否有实际编译/类型/测试证据？

**边界：** 不额外展开代码库逆向分析或软件安全/供应链专项；按实际任务执行的基础源码定位和安全义务仍有效。