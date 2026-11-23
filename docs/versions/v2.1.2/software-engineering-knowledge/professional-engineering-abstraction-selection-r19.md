# R19 · 工程抽象专业知识：按真实变化风险选择知识，区分原理与决策程序

> `DOCUMENT_CANDIDATE / KNOWLEDGE_SELECTION_VIEW / NO_EXPERIMENT_RUN` · [R19 Task DAG #554](https://github.com/kaicreator-mm/tally/issues/554) · [R18 既有风险选择](professional-quality-risk-knowledge-selection-r18.md) · [K02 变化边界](K02/advanced-engineering-abstraction-boundaries-and-variation-r19.md) · [K03 抽象层级](K03/advanced-abstraction-level-selection-and-semantic-reuse-r19.md) · [K09 演进检验](K09/advanced-abstraction-evolution-and-change-locality-r19.md)。
>
> **知识用途**：帮助 LLM 判断抽象的需求、真正的共性及停止条件，不让 LLM“越多接口越好”。本文件不创建新 Source Claim、Tally/DomainHarness Consumer、ADS 强制规则、Lint 或固定的领域架构。旧 K01–K11 165 课程知识点保持不变。

## 1. 三类知识对应不同决策节点

| 任务真实风险 | 使用的最小新知识 | 决策中的实际变化 | 不适用 |
| --- | --- | --- | --- |
| 模块职责、状态 Owner、边界/接口稳定性和扩展要求存在有权风险 | [K02 工程抽象/变化轴](K02/advanced-engineering-abstraction-boundaries-and-variation-r19.md) | 先识别稳定不变量、真实可独立变化的信息和要隐藏的设计决策，再比较候选边界 | 简单局部修改不应强制复杂模块化 |
| 重复代码、抽象模式/工具函数/模块拆分具有成本或错误语义风险 | [K03 抽象粒度和语义复用](K03/advanced-abstraction-level-selection-and-semantic-reuse-r19.md) | 决定 inline/helper/module/interface/Strategy 的最小适当级别，而非强制提取公共代码 | 语法相似不证明可以复用，抽象决策必须保持当前合同 |
| 真实新增需求、原 API/数据兼容、局部修改或坏抽象撤销 | [K09 变更隔离与维护](K09/advanced-abstraction-evolution-and-change-locality-r19.md) | 用真实陌生维护者的局部修改和旧新 Oracle 判断原抽象收益与成本 | 没有受权变更与合格 Oracle 不可宣称已证明“长期可维护” |

该路由是专业知识上的*建议选择视图*，不是每次调用都必须加载三个模块。如果任务已有清晰直接实现，`NONE` 是合理选择。需求/状态不清时先使用 K01 有权需求以及 `ASK/UNKNOWN/STOP`，新抽象知识不能赋予额外业务权限。

## 2. 必须明确区分 UNDER / WRONG / OVER / ENOUGH

- `UNDER_ABSTRACTION`：在同一稳定语义下重复维护相同机制，变化需要修改多个本不应重复的职责模块；**要有具体重复维护或变更传播证据**。
- `WRONG_ABSTRACTION`：因为表面代码相似，将**不等价的错误、权限、资源生命周期、事务或状态 Owner**强制统一；需要给同一抽象被反例推翻的真实行为。
- `OVER_ABSTRACTION`：为未经授权的扩展或不必要的层级增加真实导航、配置、维护与验证成本；不是仅有接口/类数过多的审美判断。
- `JUST_ENOUGH_ABSTRACTION`：必要事实隐藏、真实变化隔离、功能与旧行为正确且成本相称；**可以完全不创建新抽象**。
- `UNKNOWN/NOT_TESTED`：只有设计自述、没有现行有权变化或后续测试时不能给经验性“维护优胜”结论。

## 3. R19 三组实验的知识切片隔离建议（候选，不是已冻结实验）

为避免 B/C 拿到同样的方法导致处理差异不清，建议新实验 **正式冻结之前**，由独立非 Writer Reviewer 核准以下两个来自相同三篇专题的不同裁剪：

- **A — COMMON**：当前共同 ADS/受权需求/质量目标/工具/模型/预算；不额外给 R19 文章，不剥夺既有已发布 Kxx 的正常项目规范。
- **B — CONCEPT KNOWLEDGE**：A + K02/K03/K09 中的概念、学理、适用性、语义差异、正反例和 STOP 的*预登记知识切片*；**排除具有明确编号的逐步骤 DP 执行清单、步骤填写格式和强制设计检查动作**。B 仍可自行判断、采取类似方法，不能将其强行限制为不会抽象。
- **C — CONCEPT + EXPLICIT PROCEDURE**：与 B **完全同一概念性知识切片**，再加这三篇中可执行的 DP 具体步骤与可证伪产物；鼓励按步骤审查，但不额外给隐藏评分/维护答案、不同业务事实或更高工具权限。

**正式交付材料不是将整个公开仓库直接暴露给三组**。要在 G-B 实验冻结时导出并封存各组可见的*精确 UTF-8 内容字节*、SHA256、Git Blob/版本、分阶段交付时机与 Role ACL：如果 B 可访问包含 DP 的完整源文章，则 B/C 区别受污染，必须停止或明确 `CONTAMINATED/NOT_COMPARABLE`。此外，C 较长的提示是其处理成本，不能假称三组 Token 完全一样。

**实验命题不能沿用三篇知识文档中的教学示例、R18 的 F1/F2/F3 具体项目或秘密 QA**。抽象的好坏由独立冻结的有权需求与后续维护结果决定，不能把 Kxx 文章的术语原样出现在候选回复当作好设计。A/B/C 在首次 L2、首次 S5、陌生 Reader 和后续真正修改中都应以同一组评价、硬失败门槛和成本单位核算。

## 4. 为什么不是新规范或 LLM 培训证书

[R18 独立评分量表](professional-r18-quality-evaluator-rubric.md) 已明确“功能 QA 不等于架构/代码理解/维护能力”。R19 将*工程抽象判断*作为更聚焦的学科能力；两者可复用评判原则，但不能在 R18 原始实验中事后添加新治疗、权重、保密样本或编写对 B 有利的 Reviewer 答案。

**停止规则**：若真实代码中没有共同机制、没有现行需要隔离的变化或建立接口的授权依据，则让 LLM 继续采用直接实现；若后续实验缺少 Oracle 独立性、分组隐藏、对等权限或真正的维护修改，则该轮只能标 `NOT_ESTABLISHED`，不能宣称工程抽象知识已被验证有效。
