# K11 · 工程经济学与决策 — 软件工程领域知识库

> **Tally v2.1.2 软件工程知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #532](https://github.com/kaicreator-mm/tally/issues/532) · [总索引](../README.md) · ADS 正式基线 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。当前只完成文档化的知识设计，尚无正式 Source→Claim 资格或真实 Package 执行。

以真实用户交付与质量目标衡量开发活动，选择最小充分研究、任务拆分、Agent 并行和测试验证。

**系统正文第一卷（R2 文档范围独立 ACCEPT_SCOPED @ ede9d646cb591b8df5179dc6bc11b60668fe968c）**：[学科深化 01：工程经济与实证效用评估](foundations-deep-dive-01.md)。本章解释原理、适用条件、工程取舍与可证伪反例；仍为文档知识候选，不是已合格 Source Claim 或真实 Harness/Agent 功能。

**系统正文第二卷（R3 新增、待增量独立审核）**：[学科深化 02：实验设计与统计证据](foundations-deep-dive-02.md)。本卷补充基础理论、适用条件、备选技术与可证伪反例；不宣称 165 点均已教材级展开，也不自动取得生产 Source/Consumer 资格。

**先了解整个专业学科，再按需选择问题**： [K11 系统课程：5 个知识群 / 15 个专业知识点](systematic-syllabus.md) 用于完整概念、原理和常见反例的持续积累；下表是每次任务的有界检索入口，并非该模块的全部知识。课程内容尚未得到 Source Claim/Domain Package 生产资格。

## 当前工程问题索引（优先入口）

先根据当前 Task 的**真实问题**命中下表最小 Topic，再阅读与之相关的 [专业方法](engineering-methods.md)、[Agent Skills](agent-skills.md)、[案例](casebook.md)、[来源](sources.md)；不要把九个专题文件全部塞进一次 Agent 上下文。**Topic ID 是文档检索标识，不是新增的 Tally Wire Schema/Task Status。** 跨模块列仅表示引用知识及交接位置，不改变原有工程权威。

| Topic ID | 当前问题/触发 | 最小必要输入 | 知识交接 | 必须拒绝的错误 |
| --- | --- | --- | --- | --- |
| [`K11.TRADEOFF`](topics/TRADEOFF.md) | 多个方案如何比较价值/成本/风险？ | 用户结果/备选/真实后果 | K11→K02 | 最少 Token 不等于最优 |
| [`K11.ASSURANCE_FLOOR`](topics/ASSURANCE_FLOOR.md) | 小 diff 是否仍需高风险测试？ | 影响/不可逆性/现行保证 | K11→K05 | 省成本不能取消强制保护 |
| `K11.STOP_WORK` | 什么时候停止研究、重复 Review/CI？ | 当前成果/实质新证据 | K11→K04 | 不能无界延长版本 |
| `K11.PARALLEL_ROI` | 并行 Agent 的协作成本是否高于收益？ | 各写集/合并/Reviewer/时序 | K11→K07 | Agent 数越多未必更快 |
| [`K11.EMPIRICAL_EVAL`](topics/EMPIRICAL_EVAL.md) | 知识辅助 Agent 是否真的提高软件开发质量？ | 同任务/基线/独立验收/成本 | K11→K05 | 问答胜率≠真实工程提升 |

**选用顺序：** 真实任务/版本/角色 → Topic → 相关 DP/Skill/正反例 → 当前来源/适用条件 → 原有 Validation/Task/Host 资格。文档只提供分析方法和停止条件。模块内容唯一维护在现有专题文件，Topic 行不复制其全文。

## 已深化的问题级专题

当前优先深化 3 个对真实工程结果影响较大的专题：[`K11.TRADEOFF`](topics/TRADEOFF.md)、[`K11.ASSURANCE_FLOOR`](topics/ASSURANCE_FLOOR.md)、[`K11.EMPIRICAL_EVAL`](topics/EMPIRICAL_EVAL.md)。每篇只解释当前问题的决策机制、反例和来源，其余方法继续直接使用已有九文件知识库；当前文档 Topic 不代表已激活生产知识。

## 目录和何时使用

| 知识资源 | 为什么要读 |
| --- | --- |
| [systematic-syllabus.md](systematic-syllabus.md) | 本模块系统专业学科：五个知识群、15 个概念/机制/反例 |
| [disciplinary-foundations.md](disciplinary-foundations.md) | 学科理论、可应用的工程方法与误推断 |
| [external-practices.md](external-practices.md) | 企业/GitHub 真实工程参考与可证伪案例 |
| [domain-model.md](domain-model.md) | 区分工程概念、事实来源及容易混淆的语义 |
| [engineering-methods.md](engineering-methods.md) | 根据具体条件选择方法、方案与有界 Decision Procedure |
| [agent-skills.md](agent-skills.md) | 供 Tally 选择的 Agent Skill、单次可复制提示词和拒绝条件 |
| [casebook.md](casebook.md) | 正反案例、变体、反模式和实际工程决策案例 |
| [validation.md](validation.md) | 负例、知识适用性训练、未来真实消费者效用验收 |
| [sources.md](sources.md) | ADS 具体条款、有效版本、候选/规范和权限边界 |

## 三个优先问题

1. 额外研究/审查/CI 是否影响关键决策或真实保护底线？
2. 方案的总实施/集成/返工/维护成本是多少，哪些只是估算？
3. Agent 知识辅助是否经真实开发任务公平对照证明有益？

## 快速决策索引

| 触发 | 优先处理 | 期待输出 |
| --- | --- | --- |
| 单一 UI 文案变更，无高风险 | INLINE 一次修复+相关检查 | 最小成本完成 |
| 公共接口具有不可逆迁移风险 | 保留必要设计/负例/恢复/Host 验证 | 风险与保证方案 |
| 多 Agent 并行需共享同一中心文件 | 优先单 Writer/合同后并行只读 | 总集成成本对比 |
| Tally 知识试验只在 7 组问答得分好 | 限定知识问题观察，另设计真实编码任务对照 | 研究级别效用报告 |

**选用规则**：不要求每次调用本模块的全部九份专题文档；依据当前任务所需决策选择最小知识切片。K09–K11 是条件性消费，K07/K08 在首批真实工程协作/交付需要时才选用。Tally 既有 Engineering Business/Host 事实源继续有效；本知识库为 `PROPOSED_NOT_CONNECTED`，不是新的知识服务/数据库或独立执行权限。