# K11 · R18 进阶：把首次实现、代码理解和后续维护放进同一真实成本账本

> `DOCUMENT_CANDIDATE / ENGINEERING_ECONOMICS_METHOD / NOT_A_COST_PREDICTOR` · [K11](README.md) · [既有风险与必要流程](advanced-decision-risk-value-of-information-and-process-cost.md) · [质量知识选择](../professional-quality-risk-knowledge-selection-r18.md) · [#551](https://github.com/kaicreator-mm/tally/issues/551)。
>
> 专业质量参考：[ISO/IEC 25010:2023](https://www.iso.org/standard/78176.html) 只提供产品质量分类；[SEI 架构权衡](https://www.sei.cmu.edu/library/architecture-tradeoff-analysis-method-collection/) 帮助分析不同质量目标的代价；[Google Code Review Practices](https://google.github.io/eng-practices/review/reviewer/looking-for.html) 提醒设计/复杂度/测试/文档是不同审查问题。下述成本模型是**分析建议**，不是来自 ISO 或 ADS 的规范公式。

## 1. 为什么只记录首次 Agent Token 会产生错误优化方向？

有的候选为迅速通过首次测试把职责集中到一个模块，第一次生成很便宜；但新增字段或修复一个异常需要修改多个不相关行为，并让维护者重新理解巨型状态机。另一方案开始时需要定义明确边界、写清错误语义，初始成本略高，后续变更更安全。只看首次 Token，容易选错架构。

`C_observed = C_initial + C_QA_and_review + C_rework + C_maintenance_read + C_maintenance_change + C_regression_repair + C_human_intervention + C_execution_resources`

上式是**成本账本的类别分解**，不是统一单位的数值公式。Token、秒、人民币/美元、人时、GPU/CI 用量不能不经单位与价格/估值约定直接相加。分别记录原始单位；如必须货币化，预先注册相同 Provider 价格、人工估算和真实不确定区间。

## 2. 一套比较公平的成本分母

| 阶段 | 应计成本 | 最常漏掉 |
| --- | --- | --- |
| 首次设计 L1/L2 | 研究/架构决策、澄清、Owner 反馈、失败重试 | 只统计最后被接受的设计 |
| 首次实现 L3/S5 | 模型输入/输出、实际工具、测试/构建、失败/修复 | A 失败后修复只算 A，但 B 的额外知识输入不计 |
| 审核/QA | 独立 Reviewer/Oracle 构造或执行、发现/复制错误、裁定 | Hidden QA 作业及人工判分被当作免费 |
| 陌生维护者接手 | 阅读/导航、解释错误、修改、旧/新行为测试、修复 | 只记实际输入代码的几分钟 |
| 风险与后果 | 严重合同违约、回归、数据安全/隔离失败、人工恢复 | 用平均“质量得分”抵消严重问题 |
| 基础设施 | Node 依赖/缓存、Agent Provider、CI/Host、重复仓库同步 | 过度执行全部 CI 被当作无成本 |

**共同基础设施/一次性的固定研究费用**（例如设计隐藏 QA），要单列为 `COMMON_EXPERIMENT_OVERHEAD`；若按每组分摊，必须先固定同一规则。不可把研究者事后额外给 A 的 Debug Hint 当免费资源，也不能把 B 的额外知识选择成本排除在总成本外。

## 3. 成本应与对应质量结果配对，而非追求最短时长

对每个任务同时保存 `INITIAL_QUALITY`、`FIRST_S5_FUNCTIONAL_GATE`、`MAINTENANCE_OLD_NEW_CONTRACT_GATE`、`TOTAL_COST_BY_UNIT`、`HARD_FAILURE` 和 `UNCERTAINTY`。只有在合同质量和关键安全下限可比较时，讨论在质量维度相似下哪个更省成本；若某组破坏有权隔离或数据完整性，即使最省 Token 也必须单列严重失败，而不是经济最优。

**帕累托判断**：一个候选在所有相关已授权质量维度和成本维度都不差，且至少一维严格改善，才可在样本范围内说它在这些指标上占优；不能默认“可读性好 10 分，性能差 10%”可以抵消，除非 Owner 已经明确接受这种权衡。在不同质量目标上互有优劣时，报告 `TRADEOFF_NOT_DOMINANCE`。

**重要因果边界**：R1 单文件修复 B 盲评较好但 Token +53%；R2 单次 A 15.56M/B 5.46M 包含 A 后置 QA 修复和流程不对称；R3 三组 12/12，B 6.83M/C 8.87M 但无统计分辨力；R4 原始 QA 部分无效。**这些不能推出一个通用 Token 节省率或“早注入最佳”**。如果当前账本缺某组独立真实用量，写 `COST_NOT_RECONSTRUCTABLE`，不要拿自述数做精确 ROI。

## 4. 反例与真实停止条件

**反例 A**：B 初始成本低 20%，但独立维护者花两倍时间定位错误且一个旧公开行为回归。若只看首次 Token 会称 B 优胜；正确结论是初始阶段便宜、维护性不利，并说明是否真正可比。

**反例 B**：A 有更多模块与测试，但这些是受权隔离边界和独立故障 Oracle 的必要成本。把它们全部算冗余，可能鼓励牺牲正确性。

**反例 C**：两个方案功能都通过，但一组运行在更快的 Provider、更大缓存或不同 Host；成本不具同等配置，必须标 `NOT_COMPARABLE`，而不是宣布模型更聪明。

**反例 D**：若研究样本只有每组两个，观察到一组平均便宜并不能推断未来任意任务的收益比例；要提供每一个样本的原始值、任务难度和变动范围，不能靠堆积 Token 总量造成假精度。

对于普通文档或低风险单文件改动，复杂生命周期成本计算可能不值得：只需记录有限受影响测试与明显的工时差异，保持投入产出匹配。无具体维护目标/现行价格/预算的场合，必须停止精确 ROI 声称。

## 5. 交接接口与资格

- K02 提供架构备选与质量属性权衡；K03 提供具体实现和可理解性证据；K05 提供独立功能和 Reviewer Oracle；K09 提供陌生维护者后续真实修改数据；K11 合并**同单位且有源**的比较，不替代上述质量判断。
- 最少提供 `Case_ID / H0 / Initial_HEAD / Maintenance_HEAD / Baseline_Authority / Sample_Inclusion / Provider_Profile / Cost_Source / Cost_Unit / Failures / Human_Handoff / Uncertainty / Reviewer_Result`。各分母包括超时、放弃、错误与返工。
- 无对等预算或隐藏 QA 泄漏时不把两组成本当公允对照；没有合理样本随机化时只做有界描述，不赋予知识 Source 资格。

本篇不部署成本追踪服务、不增加 ADS 检查或 Tally/DomainHarness Token 计费 API，不开启新的研究实验。
