# K02 · Domain Model / 核心概念和语义

> **v2.1.2 知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #523](https://github.com/kaicreator-mm/tally/issues/523) · [总索引](../README.md)。知识来源锁定 ADS v4.19.0 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；文档设计不代表已获知识资格或实际 Tally Runtime 消费。

## 概念定义、混淆与原文分析

## 2. 领域概念与边界

| 概念 | 精确定义及判断要点 | 常见误用 |
| --- | --- | --- |
| **Architectural Driver** | 有来源的功能/质量/部署/成本约束，足以影响结构决策 | 拿某个技术潮流当作驱动 |
| **Boundary / Component** | 具有清晰责任和依赖方向的模块或子系统边界 | 按代码目录名称机械划分服务 |
| **Interface Contract** | 调用者与实现共同遵循的输入输出、错误、版本和行为语义 | 仅列函数签名而忽略失败情况 |
| **Data/State Ownership** | 谁具有状态变更权、事务范围与最终真相 | 两个模块可各自任意修改同一份状态 |
| **Quality Attribute** | 可靠性、性能、演进、可测试性等项目真正关心的系统属性 | 不带目标/场景就宣称“高性能” |
| **ADR / Trade-off** | 决策原因、备选方案、代价、限制与适用期的持久记录 | 只有结论，没有候选和权衡 |
| **Research Demo** | 针对关键未知假设设计的有界可证伪试验 | Demo PASS 等同 Host/Release PASS |
| **Architecture Freeze** | 经现有 L2 Authority 确认的设计基线 | Agent 写了 ADR 就默认冻结 |

不同概念之间通过具体的来源/适用性/决策/消费者连接，不必为每一个词建立全新的数据库表。只有在真实 Tally 任务中被调用且能够改善工程判断时，才有必要形成可选的结构化资产。

## 与其他知识的语义边界

### 关系 1

Requirement/Quality Attribute → Architecture Driver → Alternatives → ADR → Contract/State Ownership → Validation；缺少驱动的模式选择为无源建议。

### 关系 2

模块边界不是目录名字；Bounded Context、部署单元与事务一致性边界应分开。

### 关系 3

Architecture Demo 收据按 E1/E2/E3 限定强度，不能跨越到真实 User Journey/Release。

## 知识推断的可靠性分级

| 等级 | 可以断言的内容 | 不能断言的内容 |
| --- | --- | --- |
| Published ADS source | 当前适用源/版本中的明确规则、权威/权限边界 | 所有由本模块新增的便利推论都成为 MUST |
| Source-backed interpretation | 有清楚来源与适用范围的工程解释/方法 | 跨项目普遍效力或生产资格 |
| Teaching example | 在给定假设下的可证伪示例 | 真实 Agent/Host/项目结果 |
| UNKNOWN | 缺少权威来源、当前性或真实验证 | PASS、正式 Product/L2/Task/Release 状态 |

## Agent 使用时的五个提问

1. 真实架构驱动是否支持新增模块或服务？
2. 接口、数据与状态的唯一写入责任如何定义？
3. 需要何种最小可证伪 Demo 才能关闭关键未知？

**边界：** 不额外展开代码库逆向分析或软件安全/供应链专项；按实际任务执行的基础源码定位和安全义务仍有效。