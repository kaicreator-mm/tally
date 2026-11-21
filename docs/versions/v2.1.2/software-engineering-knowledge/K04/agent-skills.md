# K04 · Agent Skills / 操作技能与交接

> **v2.1.2 知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #525](https://github.com/kaicreator-mm/tally/issues/525) · [总索引](../README.md)。知识来源锁定 ADS v4.19.0 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；文档设计不代表已获知识资格或实际 Tally Runtime 消费。

## Tally Workflow 选择与实际消费者边界

## 5. Tally 消费路径与跨模块接口

**主要 Workflow：** 各阶段按需 DAG-P / DAG-A / Implementation DAG-I

| 工作环节 | 读取的当前事实 | 本模块的输出 | 原有权威/禁止推断 |
| --- | --- | --- | --- |
| L1 | User Need/冲突/研究 | DAG-P 或 INLINE/Worklist | 不冻结 Product |
| L2 | 被接受的需求、架构未知 | DAG-A/试验依赖 | 不冻结 Architecture |
| L3 | 已准入的实施目标 | DAG-I/真实 Native BLOCKED_BY | 不自动签 Claim |
| 交付 | 现有 Task/Review/Host 状态 | 最小关键路径与影响重规划 | 不自动完成 Release |

**跨模块组合：** K01 供给 Need/Requirement，K02 供给设计合同，K03/K05 形成原子实施与测试，K07 提供多 Agent 角色/写集准则，K11 用于估计拆分协调成本。K09 变更时调用重规划而非整图重做。

## 现有 Agent Skills 的输入输出与拒绝路径

## 6. Agent Skills（输入、输出和验证）

### SK-K04-01 · Phase Planner

- **调用时机：** 存在阶段不确定性或多项研究任务
- **最低输入：** 来源绑定的 Product/Design/Task 事实与真实需求
- **产出与格式：** 每层工作形态+真实节点和停止准则
- **自查及异常路线：** 没有不可替代前置则不能造 blocked-by
- **不能做：** 独立决定 Product/Architecture Freeze

### SK-K04-02 · Task DAG Designer

- **调用时机：** 已确认实现范围且有多 Task
- **最低输入：** Task Pack 候选、版本/写集、接口 seam、验证义务
- **产出与格式：** 验收单位、Owner、真实依赖、并行/串行理由
- **自查及异常路线：** 检查循环/虚假边/共享写冲突
- **不能做：** 把研究节点复用成源码 Task PASS

## 可直接作为单次 Agent 任务使用的提示词

> 注意：以下是**方法提示词**，不是写入许可、部署指令或平台强制格式；使用时必须先填写真实 Task/当前版本/输入来源。不得交给 Agent 无界持续执行。

```text
MODULE=K04
KNOWLEDGE_VERSION=TALLY_V2.1.2_SOURCE_CANDIDATE
ADS_EFFECTIVE_SOURCE=933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c
ROLE=SCOPED_ENGINEERING_ANALYST
TASK_SOURCE=<actual issue, current sha and approved scope>

你负责工程任务规划，只能给出候选工作结构，不能自行授予 Claim/Release。输入真实用户目标、L1/L2 当前来源、模块合同、执行环境和写集。先确定原子交付结果，逐条检验 blocked-by 是否真的不可缺少；按任务复杂度选最小 INLINE/WORKLIST/必要明确 DAG。分清研究证据与正式实施 Task，避免多 Agent 写入冲突；输出 Task/前置成果/Owner/验收/停止及不做事项。

输出时明确分隔 OBSERVED / SOURCE_BACKED / PROPOSED / UNKNOWN / NOT_RUN。
只要求本次问题所需内容；不自造 Product/L2/Claim/Review/Release 权威。
```

## 最小交接清单

- [ ] Task/Issue 与适用版本明确；信息来自可读的现行来源而不是聊天历史猜测。
- [ ] 具体问题、使用了哪些知识和适用条件明确，未加载整仓无关文档。
- [ ] 结果中的已观察事实、推断、建议、未知和测试未执行分开记录。
- [ ] 提交的 Artifact/测试/Review 指向准确 subject；没有合法执行环境时只给分析结果。
- [ ] 无界循环、未获权限的写入或虚构 PASS 被拒绝。