# K11.EMPIRICAL_EVAL · 用真实开发任务评估 Agent 知识辅助的增益

> [K11 知识库](../README.md) · [工程方法](../engineering-methods.md) · [Agent Skills](../agent-skills.md) · [案例](../casebook.md) · [原始依据](../sources.md) · [外部资料目录](../../external-source-catalog.md)。
>
> `KNOWLEDGE_TOPIC_CANDIDATE / NOT_SOURCE_CLAIM_QUALIFIED / PROPOSED_NOT_CONNECTED`。本文件是原创工程问题知识切片，不是 Tally 已生效的机器 Schema，也不是当前 Task/Review/Host/Release 执行许可。案例仅为合成教学示例。

## 1. 工程问题和适用前提

Tally 增加一份知识文档后，7 道问答题得分提升，是否足以证明 Agent 真实软件工程能力提高？

**开始前必须确认：**
1. 固定的真实任务集与独立 Oracle/验收来源
2. 两种 Agent 策略与同等执行权限、预算、环境
3. 实际补丁/测试/Review、返工轮次、运行时间和资源数据

**学科知识与因果机制：** 语言问答的正确性与完整软件交付的正确性不是同一目标。SWE-bench 评估特定 Issue/Patch/Tests 的修复，而真实产品开发还涉及需求/架构/权限/构建/回归/用户路径。DORA 与 SPACE 提供不同维度的工程绩效/研究思路，但单一样本或工具调用次数不能证明普遍因果。需要固定真实任务、验收、环境和对照策略，同时记录失败和更多知识造成的上下文成本。

## 2. 可执行的有界判断过程

1. 优先选若干真实不同类型 Task：需求歧义、状态机负例、CI False PASS、根因诊断、用户安装/配置等；冻结版本、来源和验收。
2. 设置无知识辅助 baseline 与按 Topic 选择知识的 Agent，对齐工具、权限、预算，必要时同任务配对并区分任务难度。
3. 独立 Reviewer 按真正用户结果、缺陷/回归、验证有效性和实际持续修复次数评分，不能只看 Agent 自评。
4. 同时记录 token/CI/Review/交接成本、是否触发额外治理、失败及无法结论的场景；只有样本内观察时明确局限。

## 3. 场景分支、替代方案和不适用范围

| 条件 | 推荐的判断及范围 |
| --- | --- |
| 仅 7 道问答改善 | 只可说该题集的局部观察，非完整工程质量。 |
| 真实 Task 显示质量改善但成本明显更高 | 报告多维结果和适用范围，而非单一赢家。 |
| 评估资料/权限不公平 | 不能推断知识造成了差异，先修正设计。 |

## 4. 真实可证伪的教学情景

**受限教学输入及正确路线：** 两个策略在同一旧版本 API 修复任务上生成补丁；知识辅助 Agent 明确保护旧客户端并加负例，baseline 只改新测试。由独立 Reviewer 检查用户行为、测试充分性和实际返工。

**失败路线/反模式：** 给知识辅助组更多执行时间/更强工具，再用提高的 PR 数或自评分宣称知识必然带来 60% 生产率提升。

**最低可证伪 Oracle：** 有效报告必须区分实现质量、真实执行证据、成本和样本限制；若没有独立验收则不允许 `KNOWLEDGE_EFFECT_VERIFIED`。

## 5. 最小交接与明确的 STOP

**Agent 输出契约**：当前已观察事实/证据身份、选择的知识与前提、备选方案取舍、正反 Oracle、未得来源或环境、最小下一步。凡 Source Claim 未获得正式资格、Host/CI 未运行、Product/Task 未授权的内容，必须与可证实事实区分。

**跨模块交接**：K05 负责实际测试 Oracle，K08 提供 CI/Host 来源，K07 保证 Reviewer 来源，#286 负责未来正式消费者资格。

**停止规则**：需要越过合法写集、缺少有权业务决定、结果可能有不可逆副作用、外部状态不可确认或没有真实执行来源时，仅提出最小受控检查并保持 UNKNOWN/NOT_RUN，不通过重复调用或虚构状态推进。

## 6. 外部来源与可采纳边界

- [DORA Research](https://dora.dev/research/)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。
- [SPACE Productivity Framework](https://www.microsoft.com/en-us/research/publication/the-space-of-developer-productivity-theres-more-to-it-than-you-think/)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。
- [SWE-bench](https://github.com/SWE-bench/SWE-bench)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。

**已发布 ADS 约束**：`kaicreator-mm/ai-development-standard@933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。遇到差异应按当前有权规范与 Product/L2/Task/Host/Release 的真实作用域处理；未合并的三阶段 DAG 候选不自动变为正式工作图。
