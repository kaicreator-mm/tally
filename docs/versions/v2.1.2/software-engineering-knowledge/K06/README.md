# K06 软件工程知识库：调试与根因修复

> **v2.1.2 知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #527](https://github.com/kaicreator-mm/tally/issues/527) · [总索引](../README.md)。知识来源锁定 ADS v4.19.0 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；文档设计不代表已获知识资格或实际 Tally Runtime 消费。

用最早可观察失败事实、竞争解释和最小复现定位因果根因，避免无界修复循环。

## 使用导航

| 文档 | 具体内容与消费时机 |
| --- | --- |
| [disciplinary-foundations.md](disciplinary-foundations.md) | SWEBOK/ISO/专业学科方法与具体 Agent 决策、适用反例 |
| [external-practices.md](external-practices.md) | 工程实践项目、操作步骤及可证伪任务样例 |
| [domain-model.md](domain-model.md) | 领域概念、不变量、语义关系与错误解释；第一次判断问题时读 |
| [engineering-methods.md](engineering-methods.md) | 工程方法与决策矩阵；需要比较方案/选择手段时读 |
| [agent-skills.md](agent-skills.md) | Agent Skills、输入输出合同与直接可用提示词；真实 Task 时读 |
| [casebook.md](casebook.md) | 贯穿案例、变体、反模式；遇到相似问题或错误时读 |
| [validation.md](validation.md) | 反例、测验与未来合格证据；需要证明适用性时读 |
| [sources.md](sources.md) | 知识→ADS 具体条款、规范/候选边界和接入条件；决定知识授权时读 |

## 能力检查问题

1. 哪条命令/调用最早失败？
2. 至少两种根因解释如何用最便宜实验区分？
3. 修复后哪个正负例证明根因已消除？

## 如何选择而非加载整个库

在已有 Tally Engineering Business/Task/Host 中，先用当前具体问题和相关源事实定位相应文档；如果现有答案足够，直接结束，不强制读取其他五份文件。示例/Decision Procedure 属 `PROPOSED_KNOWLEDGE`，不是新 schema 或 Gate；真实 Task/Claim/Review/Release 仍由当前权威处理。**特别注意：完整 DAG-P/A/I 与三阶段统一形态为候选，不是已发布 ADS v4.19 必经流程。**

## 快速能力索引

| 用户问题 | 优先知识 | 交付判断 |
| --- | --- | --- |
| 当前 install 成功，首次 build 出 TypeScript 错 | 定位编译器具体源码/合同问题 | root cause hypothesis + scope |
| 网络超时但业务是否提交未知 | 先读回状态和 Provider 证据 | UNKNOWN/恢复路径 |
| 测试用例根本没被发现 | 修复测试选择/布局，非业务逻辑 | test discovery 修复 |
| 多轮同类失败没有新证据 | 停止振荡，报告阻塞与最小新探针 | 有界升级记录 |

## 状态与版本

`DOCUMENTED_KNOWLEDGE_LIBRARY / NOT_QUALIFIED / PROPOSED_NOT_CONNECTED`。这是文档中可消费知识的逻辑库，并非独立数据库、运行时 SDK 或新执行引擎。