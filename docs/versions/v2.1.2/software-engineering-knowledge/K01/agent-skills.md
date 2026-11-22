# K01 · Agent Skills / 操作技能与交接

> **v2.1.2 知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #522](https://github.com/kaicreator-mm/tally/issues/522) · [总索引](../README.md)。知识来源锁定 ADS v4.19.0 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；文档设计不代表已获知识资格或实际 Tally Runtime 消费。

## Tally Workflow 选择与实际消费者边界

## 5. Tally Workflow 消费契约

**消费阶段：** L1 Requirements / DAG-P；使用 Tally 已有 Engineering Business、Task/Attempt 与 Agent/Host 事实源，新增的仅是领域知识与程序的调用关系。

| 环节 | 需要从上游获得 | 该模块实际提供 | 必须保留的判断 |
| --- | --- | --- | --- |
| 受理 | 用户原始描述和 Product 当前版本 | 目标/角色/范围及原文来源 | 用户意图不等于已冻结需求 |
| 分析 | 相关合法规则/证据 | 场景、规则、不变量和冲突/未知 | 证据不足时不能默认业务结果 |
| 规划 | 复杂度/依赖/权威 | INLINE/WORKLIST/DAG-P 选择 | DAG-P 不自动创建 L3 Task |
| 确认与交接 | Product Owner 的采纳/处置 | 准确版本的 Requirement/Acceptance | 研究审查成功不等于 Product Freeze |

**跨层交接/复用：** L2 消费被采纳的要求及适用的 UNKNOWN；L3 的测试 Oracle 追溯到 Requirement/Business Rule。K02/K05 共享同一需求语义，不能各自再解释并修改用户原意。

## 现有 Agent Skills 的输入输出与拒绝路径

## 6. Agent Skills 与任务输入输出

### SK-K01-01 · Requirements Clarifier

- **何时调用**：用户说法存在多个合理解释
- **最小输入**：原文与已确认 Product、目标场景和适用约束
- **输出结构**：澄清问题（按影响排序）、不确定项、建议选择与后果、现有证据
- **自检**：每个问题影响真实验收/设计；无须询问的事实已从允许的来源获得
- **禁止的捷径**：借澄清机会扩大功能和强行转成长访谈流程

### SK-K01-02 · Business Rule Modeler

- **何时调用**：新功能影响状态、角色或业务对象
- **最小输入**：可追溯的需求、场景、规则及已有对象
- **输出结构**：对象/关系/状态/约束矩阵、正常/非法转换、Acceptance 和仍未知条件
- **自检**：每条业务不变量都至少能找到一个正例和禁止行为
- **禁止的捷径**：未得业务授权就断言复杂聚合/权限策略是正式需求

## 可直接作为单次 Agent 任务使用的提示词

> 注意：以下是**方法提示词**，不是写入许可、部署指令或平台强制格式；使用时必须先填写真实 Task/当前版本/输入来源。不得交给 Agent 无界持续执行。

```text
MODULE=K01
KNOWLEDGE_VERSION=TALLY_V2.1.2_SOURCE_CANDIDATE
ADS_EFFECTIVE_SOURCE=933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c
ROLE=SCOPED_ENGINEERING_ANALYST
TASK_SOURCE=<actual issue, current sha and approved scope>

你负责当前任务的需求工程分析，不拥有 Product 批准权。仅使用提供的用户原文和可访问的有效业务来源。输出：目标/角色/场景、已确认事实、业务规则与不变量、可观察验收（含禁止路径）、假设和 UNKNOWN、影响评估及需要 Owner 裁定的最小问题。没有真实来源的业务决定一律不得写成 MUST；能 INLINE 完成时不要创建多余研究 DAG。

输出时明确分隔 OBSERVED / SOURCE_BACKED / PROPOSED / UNKNOWN / NOT_RUN。
只要求本次问题所需内容；不自造 Product/L2/Claim/Review/Release 权威。
```

## 最小交接清单

- [ ] Task/Issue 与适用版本明确；信息来自可读的现行来源而不是聊天历史猜测。
- [ ] 具体问题、使用了哪些知识和适用条件明确，未加载整仓无关文档。
- [ ] 结果中的已观察事实、推断、建议、未知和测试未执行分开记录。
- [ ] 提交的 Artifact/测试/Review 指向准确 subject；没有合法执行环境时只给分析结果。
- [ ] 无界循环、未获权限的写入或虚构 PASS 被拒绝。