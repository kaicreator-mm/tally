# K01 · Domain Model / 核心概念和语义

> **v2.1.2 知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #522](https://github.com/kaicreator-mm/tally/issues/522) · [总索引](../README.md)。知识来源锁定 ADS v4.19.0 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；文档设计不代表已获知识资格或实际 Tally Runtime 消费。

## 概念定义、混淆与原文分析

## 2. 领域概念与边界

| 概念 | 精确定义及判断要点 | 常见误用 |
| --- | --- | --- |
| **Need / Goal** | 用户或业务追求的结果；可以源于明确的真实请求或正式业务来源 | 把某个按钮/表单当成最终业务目标 |
| **Stakeholder / Actor** | 具有不同目标、约束和权限的用户或系统角色 | 只考虑直接点击 UI 的用户 |
| **Requirement** | 对系统可观察行为或属性的约束，具备来源、范围、状态和优先级 | 把 Agent 提议当作用户批准的要求 |
| **Business Rule** | 在明确场景下决定允许、拒绝或计算的业务规则 | 将某个演示数据的偶然值写成通用规则 |
| **Invariant** | 在受约束过程或状态中必须持续成立的性质 | 把理想结果当成已由系统保证的事实 |
| **Acceptance Criterion** | 某个要求是否实现的可测试判据，含输入、前提、预期结果和边界 | 以“体验好”“看起来正确”代替判据 |
| **Assumption / UNKNOWN** | 已说明但尚无足够证据的假设，或未知的必要事实 | 补写一个默认业务决策掩盖 UNKNOWN |
| **Traceability** | Need→Requirement→Rule→Design→Task→Test 的证据链接 | 只列文档标题而无法说明来源和当前性 |

不同概念之间通过具体的来源/适用性/决策/消费者连接，不必为每一个词建立全新的数据库表。只有在真实 Tally 任务中被调用且能够改善工程判断时，才有必要形成可选的结构化资产。

## 与其他知识的语义边界

### 关系 1

User Goal → Scenario → Requirement → Business Rule/Invariant → Acceptance；Assumption 与 UNKNOWN 是独立分支，不能指向已接受 Rule。

### 关系 2

Actor/Role 与状态结合决定操作许可；同一操作在不同业务状态的期望结果应分别建模。

### 关系 3

任何 Requirement 的 Currentness 需要指出版本、生效范围、原始来源与审批身份。

## 知识推断的可靠性分级

| 等级 | 可以断言的内容 | 不能断言的内容 |
| --- | --- | --- |
| Published ADS source | 当前适用源/版本中的明确规则、权威/权限边界 | 所有由本模块新增的便利推论都成为 MUST |
| Source-backed interpretation | 有清楚来源与适用范围的工程解释/方法 | 跨项目普遍效力或生产资格 |
| Teaching example | 在给定假设下的可证伪示例 | 真实 Agent/Host/项目结果 |
| UNKNOWN | 缺少权威来源、当前性或真实验证 | PASS、正式 Product/L2/Task/Release 状态 |

## Agent 使用时的关键提问

1. 谁提出需求、由谁实际使用、真实成功条件是什么？
2. 哪些业务规则与不变量必须经 Product Owner 确认？
3. 什么时候能把研究交给 L2，而不是持续调研？

**边界：** 不额外展开代码库逆向分析或软件安全/供应链专项；按实际任务执行的基础源码定位和安全义务仍有效。