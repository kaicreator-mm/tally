# K05 · Domain Model / 核心概念和语义

> **v2.1.2 知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #526](https://github.com/kaicreator-mm/tally/issues/526) · [总索引](../README.md)。知识来源锁定 ADS v4.19.0 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；文档设计不代表已获知识资格或实际 Tally Runtime 消费。

## 概念定义、混淆与原文分析

## 2. 术语与领域模型

| 概念 | 具体含义 | 易错判断 |
| --- | --- | --- |
| Test Oracle | 从已确认需求、合同或不变量导出的预期行为/不允许结果 | 用实现者本人猜想作为唯一正确答案 |
| Positive / Negative | 代表正常条件与应当拒绝或失败的条件 | 只追求所有测试通过而不设计失败场景 |
| Boundary Value | 输入规模、时间、状态、类型等切换边界 | 只测平均输入/常规状态 |
| Unit / Contract / Integration | 分别检查局部行为、公开接口合同和真实组件组合 | Mock 测试通过就当真实数据库 E2E |
| User Critical Journey | 对用户有意义的真实业务闭环 | 把一堆不相关 API 调用拼成大测试 |
| Test Discovery | 测试运行器实际选择并执行了哪些用例 | 命令 exit 0 就猜测相关文件被执行 |
| Regression Test | 锁定既有接受行为或历史已修复问题 | 修改期望值以配合错误实现 |
| Source/Tested Ref | 精确代码/测试版本和环境 | 用过去的 CI 结果证明新 HEAD |
| False Pass | 测试或证据显示绿，但关键行为未被测试或真实入口未调用 | 把该现象当作没有缺陷 |

这些术语不是全局强制新增的 Tally Schema；仅当实际 Workflow/Business 的输入输出需要区分语义时，才形成有来源的可复用类型或术语。

## 与其他知识的语义边界

### 关系 1

Requirement/Invariant → Oracle → Positive/Negative/Boundary Test → Actual Execution → Evidence Scope；存在测试文件不等于被 runner 发现。

### 关系 2

Unit/Mock、Contract、Integration、Critical Journey、Host/Release 是不同范围的证据。

### 关系 3

K05 定义‘应测试什么、怎样判断充分’，K08 负责实际 CI/Build/Artifact 运行和来源收据。

## 知识推断的可靠性分级

| 等级 | 可以断言的内容 | 不能断言的内容 |
| --- | --- | --- |
| Published ADS source | 当前适用源/版本中的明确规则、权威/权限边界 | 所有由本模块新增的便利推论都成为 MUST |
| Source-backed interpretation | 有清楚来源与适用范围的工程解释/方法 | 跨项目普遍效力或生产资格 |
| Teaching example | 在给定假设下的可证伪示例 | 真实 Agent/Host/项目结果 |
| UNKNOWN | 缺少权威来源、当前性或真实验证 | PASS、正式 Product/L2/Task/Release 状态 |

## Agent 使用时的关键提问

1. 哪个来源决定 expected 行为？
2. 哪些负例和真实消费者能证明目标不变量？
3. CI GREEN 是否实际上覆盖当前受影响代码？

**边界：** 不额外展开代码库逆向分析或软件安全/供应链专项；按实际任务执行的基础源码定位和安全义务仍有效。