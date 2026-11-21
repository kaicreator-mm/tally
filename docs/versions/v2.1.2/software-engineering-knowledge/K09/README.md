# K09 · 软件演进与重构 — 软件工程领域知识库

> **Tally v2.1.2 软件工程知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #530](https://github.com/kaicreator-mm/tally/issues/530) · [总索引](../README.md) · ADS 正式基线 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。当前只完成文档化的知识设计，尚无正式 Source→Claim 资格或真实 Package 执行。

在维护已支持行为的前提下做小步重构、兼容接口升级和数据迁移，避免用新测试掩盖旧消费者回归。

**先了解整个专业学科，再按需选择问题**： [K09 系统课程：5 个知识群 / 15 个专业知识点](systematic-syllabus.md) 用于完整概念、原理和常见反例的持续积累；下表是每次任务的有界检索入口，并非该模块的全部知识。课程内容尚未得到 Source Claim/Domain Package 生产资格。

## 当前工程问题索引（优先入口）

先根据当前 Task 的**真实问题**命中下表最小 Topic，再阅读与之相关的 [专业方法](engineering-methods.md)、[Agent Skills](agent-skills.md)、[案例](casebook.md)、[来源](sources.md)；不要把九个专题文件全部塞进一次 Agent 上下文。**Topic ID 是文档检索标识，不是新增的 Tally Wire Schema/Task Status。** 跨模块列仅表示引用知识及交接位置，不改变原有工程权威。

| Topic ID | 当前问题/触发 | 最小必要输入 | 知识交接 | 必须拒绝的错误 |
| --- | --- | --- | --- | --- |
| `K09.REFACTOR_CONTRACT` | 结构重构是否保持外部行为？ | 旧接口/保留行为/测试 | K09→K05 | 更新 Golden 不掩盖回归 |
| [`K09.API_EVOLUTION`](topics/API_EVOLUTION.md) | 新 API 怎样避免破坏旧消费者？ | 兼容窗口/旧新客户 | K09→K02 | 不能无权宣告 Breaking Change |
| [`K09.DATA_MIGRATION`](topics/DATA_MIGRATION.md) | 数据迁移中断、重启、回滚怎么办？ | 旧数据/升级/恢复路径 | K09→K10 | Fresh install≠Upgrade |
| `K09.DEPRECATION` | 什么时候能合法移除旧能力？ | 消费者/支持版本/Owner 决议 | K09→K01 | 不假设永久兼容 |
| `K09.TECH_DEBT` | 技术债是否值得本轮偿还？ | 缺陷复发/演化成本/收益 | K09→K11 | 美观不足不能自动全仓重构 |

**选用顺序：** 真实任务/版本/角色 → Topic → 相关 DP/Skill/正反例 → 当前来源/适用条件 → 原有 Validation/Task/Host 资格。文档只提供分析方法和停止条件。模块内容唯一维护在现有专题文件，Topic 行不复制其全文。

## 已深化的问题级专题

当前优先深化 2 个对真实工程结果影响较大的专题：[`K09.API_EVOLUTION`](topics/API_EVOLUTION.md)、[`K09.DATA_MIGRATION`](topics/DATA_MIGRATION.md)。每篇只解释当前问题的决策机制、反例和来源，其余方法继续直接使用已有九文件知识库；当前文档 Topic 不代表已激活生产知识。

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

1. 当前变化属于 Bug/Refactor/Feature/Migration/Deprecation 哪一种？
2. 哪些旧用户、客户端或数据格式仍受支持？
3. 数据迁移中断/升级回滚后系统保持什么不变量？

## 快速决策索引

| 触发 | 优先处理 | 期待输出 |
| --- | --- | --- |
| 仅内部函数拆分，公开行为保持 | 先建立旧行为 Oracle 后做小步重构 | 前后兼容证据 |
| API 需要分页但旧客户端只认数组 | 用版本/可选字段或新端点保护旧合同 | 旧新 consumer 两臂测试 |
| 迁移涉及不可逆数据删除 | 准备备份、读回、补偿/受权决策 | 风险和恢复计划 |
| 旧代码只是风格不漂亮且无实际成本 | 允许延期，聚焦当前交付 | 债务候选及观察点 |

**选用规则**：不要求每次调用本模块的全部九份专题文档；依据当前任务所需决策选择最小知识切片。K09–K11 是条件性消费，K07/K08 在首批真实工程协作/交付需要时才选用。Tally 既有 Engineering Business/Host 事实源继续有效；本知识库为 `PROPOSED_NOT_CONNECTED`，不是新的知识服务/数据库或独立执行权限。