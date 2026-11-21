# K06 · Domain Model / 核心概念和语义

> **v2.1.2 知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #527](https://github.com/kaicreator-mm/tally/issues/527) · [总索引](../README.md)。知识来源锁定 ADS v4.19.0 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；文档设计不代表已获知识资格或实际 Tally Runtime 消费。

## 概念定义、混淆与原文分析

## 2. 术语与领域模型

| 概念 | 具体含义 | 易错判断 |
| --- | --- | --- |
| Symptom | 用户或 Runner 实际观察到的错误表现 | 以 exit code 直接当作根因 |
| First Failing Step | 按真实执行时序第一个失败且有实际输出的命令/检查 | 只看 pipeline 最终总结消息 |
| Root Cause | 能够解释症状并可用反事实/修复证据验证的机制 | 出现频率高的猜测就等同确定因果 |
| Reproducer | 在受控输入/版本下可重复触发相同缺陷的最小执行 | 用不相关 Mock 演示替代真实触发 |
| Competing Hypothesis | 仍可能解释同一症状的其他原因 | 用一个猜想排除所有其他环境问题 |
| Repair Scope | 最小合法变更路径及应保持不变的业务/工程行为 | 故障排查时擅自重写整个项目 |
| Regression Oracle | 证明修复问题并保持已有行为的正负测试 | 仅旧错误消失但引入新的缺陷 |
| Attempt/UNKNOWN | 记录哪次 Agent/Host 检查实际运行，哪些结果未取得 | 工具未执行或权限缺失却写 FIXED |
| Oscillation | 不同修复在相同缺陷上反复回滚、补丁互相抵消 | 将多轮无进展当成持续推进 |

这些术语不是全局强制新增的 Tally Schema；仅当实际 Workflow/Business 的输入输出需要区分语义时，才形成有来源的可复用类型或术语。

## 与其他知识的语义边界

### 关系 1

Observed Symptom → First Failing Step → Competing Causes → Discriminating Probe → Root Cause → Patch → Regression Evidence。

### 关系 2

异常所属层级可能是 Product/Design/Code/Test/Dependency/Provider/Host；‘最后退出码’不是根因。

### 关系 3

K06 负责真实故障因果，K03 负责当前 Task 代码修复，K10 提供一般恢复/可靠性模式。

## 知识推断的可靠性分级

| 等级 | 可以断言的内容 | 不能断言的内容 |
| --- | --- | --- |
| Published ADS source | 当前适用源/版本中的明确规则、权威/权限边界 | 所有由本模块新增的便利推论都成为 MUST |
| Source-backed interpretation | 有清楚来源与适用范围的工程解释/方法 | 跨项目普遍效力或生产资格 |
| Teaching example | 在给定假设下的可证伪示例 | 真实 Agent/Host/项目结果 |
| UNKNOWN | 缺少权威来源、当前性或真实验证 | PASS、正式 Product/L2/Task/Release 状态 |

## Agent 使用时的五个提问

1. 哪条命令/调用最早失败？
2. 至少两种根因解释如何用最便宜实验区分？
3. 修复后哪个正负例证明根因已消除？

**边界：** 不额外展开代码库逆向分析或软件安全/供应链专项；按实际任务执行的基础源码定位和安全义务仍有效。