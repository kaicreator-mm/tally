# K11 · Agent Skills / 任务级操作技能

> **Tally v2.1.2 软件工程知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #532](https://github.com/kaicreator-mm/tally/issues/532) · [总索引](../README.md) · ADS 正式基线 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。当前只完成文档化的知识设计，尚无正式 Source→Claim 资格或真实 Package 执行。

## Tally Workflow/Business 实际与拟议消费者

## 5. Tally 工程 Workflow 消费关系

**主要消费场景（条件性拟议）：** `PROPOSED_NOT_CONNECTED`；Tally Engineering Business 的 Product/Design/Task 规划与 Experience/Feedback 读者在真实方案取舍、并行成本或重复验证问题时选用。输入：实际任务、备选方案、已测/估算成本、质量下限及真实结果；产出：最低充分方案、停止准则、有限任务对照建议；Product/L2/Task/Release 权威不由经济评价替代。

| 时刻 | 既有来源输入 | 该知识提供 | 不得取代的工程权威 |
| --- | --- | --- | --- |
| Product/L1 | 真实用户价值和不确定性 | 调研深度、先做/不做判断 | 不能代替 Product Owner |
| L2 | 架构候选/后果与演进 | 方案权衡与最小必要 Demo | 不能代替 L2 Freeze |
| L3/DAG | 任务独立性/写集/实际资源 | 合理 Task/Agent/CI 组合 | 不能自动 Claim/Release |
| Feedback | 实际缺陷/返工/成本与交付 | 经验候选和投资优先级 | 未经验证不宣称普遍收益 |

**跨模块关系：** K01/K02 产生候选决策和业务/质量驱动，K04 应用复杂度自适应 Task 组织，K07 评估多 Agent 协同成本，K05/K08 判断哪些验证是必要而非多余，K06/K09/K10 提供真实长期返工、运行质量和维护成本来源。

## 现有模块 Skill：触发、输入、结果与拒绝

## 6. Agent Skills 与输入输出契约

### SK-K11-01 · Engineering Trade-off Analyst

- **调用条件**：至少两种可行路线且区别会影响工程质量/总成本
- **输入**：真实目标、可行候选、现行保证、环境与资源证据
- **输出**：同一维度的方案矩阵、推荐与替代方案、适用范围/待测变量
- **核验要点**：估计与实测分开，无法核实数字不写为事实
- **拒绝或升级条件**：不能把单一 Token 成本压到最低作为唯一准则

### SK-K11-02 · Adaptive Work Stop Planner

- **调用条件**：任务过度拆分、重复研究/CI 或版本久未收尾
- **输入**：当前真正未完成结果、真实依赖、已合格/缺失证据和价值
- **输出**：最小下一步或应当停止的条件、受影响验证范围
- **核验要点**：没有额外研究收益时明确结束准备工作
- **拒绝或升级条件**：不允许反向规避高风险真实 Host/独立审查

### SK-K11-03 · Task Outcome Evaluator

- **调用条件**：ADS 实践或 Tally 已收集足够真实工程结果
- **输入**：固定任务集/验收标准、实际源/Review/CI/Agent 消耗
- **输出**：对照正确性、返工、缺陷、时间、成本和局限的效用报告
- **核验要点**：如样本数少、不是公平对照就明确不可外推
- **拒绝或升级条件**：不能修改原始收据或自评成果取代独立验证

## 单次 Agent Skill 提示词（直接文本，可按真实任务填写）

```text
OUTPUT_LABEL=TALLY_V212_K11_SCOPED_KNOWLEDGE_APPLICATION
VERSION_ISSUE=kaicreator-mm/tally#521
MODULE_ISSUE=kaicreator-mm/tally#532
STATUS=KNOWLEDGE_DESIGN_CANDIDATE_NOT_RUNTIME_AUTHORITY
ADS_SOURCE_SHA=933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c
TARGET_REPO=<actual repository>
TARGET_TASK=<actual issue/approved scope>
TARGET_HEAD=<actual exact source SHA>
ROLE=SCOPED_ENGINEERING_ANALYST

你是受约束的软件工程方案分析 Agent。输入真实目标、候选工程路径、当前要求的质量/权限下限、已观察 CI/Agent/Review/维护成本。比较同一目标下的总工程成本、交付时间、变更风险与可逆性，将观测值、估计和 UNKNOWN 分开；建议最小充分实施/验证深度及停止点。只有真正独立工作才并行，多余 CI/Review 可省但既有强制 Gate 不可跳过。不得通过 Issue/Token 数量证明业务价值。

输出按 OBSERVED / SOURCE_BACKED / PROPOSED / UNKNOWN / NOT_RUN 分类。
仅完成当前这一次有界任务，结束时输出完整结果和剩余阻塞。
```

## 结果结构的语义要求（不是新 Wire Schema）

- 事实与数据来自当前用户、真实 Issue、现行文档/源码/日志，且能复现来源版本。
- 选择的知识有适用性理由和必要反例，不凭文档标题或技能名称选择。
- 明确真实权威保留于 Task/Claim/Review/Host/Release 原 Owner；通信/技能建议不代授权。
- 未执行的代码验证写 `NOT_RUN`，证据不足写 `UNKNOWN`，不要编造 PASS 或时间/成本。