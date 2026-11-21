# K04 · Domain Model / 核心概念和语义

> **v2.1.2 知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #525](https://github.com/kaicreator-mm/tally/issues/525) · [总索引](../README.md)。知识来源锁定 ADS v4.19.0 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；文档设计不代表已获知识资格或实际 Tally Runtime 消费。

## 概念定义、混淆与原文分析

## 2. 术语与领域模型

| 概念 | 具体含义 | 易错判断 |
| --- | --- | --- |
| Work Outcome | 独立可验收的交付结果，能够说明用户/工程目标得到什么 | 只按代码文件或编辑动作数划分工作 |
| DAG-P | L1 需求/产品研究节点间实际前置依赖 | 把它当已经冻结的源码 Task DAG |
| DAG-A | L2 架构研究/合同/试验间实际前置依赖 | 一个架构讨论就创建正式 Task Claim |
| DAG-I | 既有实施 Task 与 GitHub native dependency 事实图 | 创建独立第二套实施 DAG 权威 |
| BLOCKED_BY | 前项未得到所需真实产物时后项不能合法完成 | 仅有关联/同一话题就标阻塞 |
| CONSUMES_EVIDENCE | 后项决策参考前项的来源、观察或结论 | 把引用自动当作 ready/PASS |
| Write Set / Owner | 某次 Builder 允许改哪些文件及共享资源谁持有写权 | 文件互不冲突便不需要接口和 Host 边界 |
| Work Shape | INLINE / BOUNDED_WORKLIST / EXPLICIT_DAG 按阶段单独决定 | 所有版本都使用同一种大 DAG |
| Critical Path | 由真实依赖制约最后交付的必要工作链 | 任务最多的路径或最复杂研究即关键路径 |

这些术语不是全局强制新增的 Tally Schema；仅当实际 Workflow/Business 的输入输出需要区分语义时，才形成有来源的可复用类型或术语。

## 与其他知识的语义边界

### 关系 1

Product Research/Evidence 与 Architecture Research/Decision 是不同上游事实；Implementation Task DAG 是正式工作执行依赖。

### 关系 2

`DAG-P/A/I` 统一命名与阶段工作形态属于 Tally 候选，不能写成已发布 ADS v4.19 正式三 DAG 权威。

### 关系 3

Task→Writer/WriteSet→Verification→Integration；CONSUMES_EVIDENCE 不等于 BLOCKED_BY。

## 知识推断的可靠性分级

| 等级 | 可以断言的内容 | 不能断言的内容 |
| --- | --- | --- |
| Published ADS source | 当前适用源/版本中的明确规则、权威/权限边界 | 所有由本模块新增的便利推论都成为 MUST |
| Source-backed interpretation | 有清楚来源与适用范围的工程解释/方法 | 跨项目普遍效力或生产资格 |
| Teaching example | 在给定假设下的可证伪示例 | 真实 Agent/Host/项目结果 |
| UNKNOWN | 缺少权威来源、当前性或真实验证 | PASS、正式 Product/L2/Task/Release 状态 |

## Agent 使用时的五个提问

1. 哪个成果可以独立验证且带完整失败处理与测试？
2. 哪条阻塞边具有真实不可替代前置条件？
3. 是不是正在为了更多 Agent 或 Issue 过度拆分？

**边界：** 不额外展开代码库逆向分析或软件安全/供应链专项；按实际任务执行的基础源码定位和安全义务仍有效。