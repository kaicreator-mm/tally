# K11 进阶工程经济：不确定估算、价值信息、风险后果和必要流程

> [K11 课程](systematic-syllabus.md) · [R5 Controlled evaluation](advanced-empirical-evaluation-and-value-of-knowledge.md) · [K07 协作对照](../K07/advanced-agent-role-context-and-outcome-evaluation.md)
>
> **R8 ACADEMIC_DOCUMENT_CANDIDATE / NO_FINANCIAL_MEASUREMENT / NO_ACTUAL_EXPERIMENT。** 精确深化旧课程 **K11.F2.1 估算不是精确承诺**、**K11.F3.1 风险识别与后果强度**、**K11.F5.1 最小必要流程**三个 ID。方法适用的关键不是任何通用数字阈值，而是有权目标、真实后果和可区分结果。

## 一、不同的工程问题需要不同的数字

假设五个任务存在并行依赖，给每个任务简单加一条“预计完成日期”不会自动得到可信交付时间。至少区分：
- **Work estimate**：真正需要多少工程人时，可能包括编译/集成/返工；
- **Calendar forecast**：考虑 Task DAG、有限并行 Agent 数、等待真实 Owner/Provider、共享状态互斥后多久结束；
- **Commitment**：项目有权主体愿意在特定条件与风险范围内承诺何种交付；
- **Outcome**：实际已经完成的版本/功能/验收证据。

K04 的教学 DAG A=2、B=4、C=3、D=2、E=1 小时，工作量之和为 12 小时，理想无限资源关键路径为 8 小时。若只有一位执行者，完全顺序的时间可能接近 12 小时；如 API 合同 A 变动、测试失败/资源占用，实际工期可超过这两个理想值。必须写明前提和区间，不把“8h”包装成 95% 可靠的具体承诺。

## 二、风险 = 事件概率 + 后果 + 证据不确定性，不只是频率

给两个教学风险：
- R1：CI 文档导航链接失效，假定概率 20%、影响成本 2 资源单位，模型期望损失 0.4；
- R2：远端付款 ACK 丢失导致重复财务 Effect，假定概率 1%、影响成本 500 单位，模型期望损失 5。
即使 R2 发生频率低，**在这套明确假设的期望损失模型中**其后果成本更高；严重风险是否允许自动处理还由不可逆后果、合规/授权和可撤销性约束决定，而不能只按期望数排序。真实概率均未测得，未知不意味着零；“极罕见”与“已实际确认不会发生”也不同。

公式 `ExpectedLoss = Σ p_i × L_i` 只在事件模型与后果定义可辩护时有意义；它不能将未知安全/法律后果“数值化成可以无条件批准”或替代用户的真实风险承受意愿。

## 三、调查成本和值得进一步提问的条件

**合成信息价值案例**：在相同假设下，不调查直接实施某方案，预计业务误判概率为 0.30，若错误损失 100，则期望误判损失是 30。花费 8 单位取得可信补充证据并据此调整决策，模型预测误判概率降至 0.10，故预期总成本为 `8 + 0.10×100 = 18`，比直接行动预计少 12。该 `12` 是**在调查真的能使后验/决策达到所设情景时**的预期净收益，绝不是实测的 Agent 提效或一个不需要验证的信息价值定理。

现实上，能否将误判概率从 0.30 降至 0.10、是否需要同步承受时间损失，必须用证据或敏感性分析判断。如果调查后仍无法识别真实 Source Authority，则不应把“做过调查”当成有权 Rule；信息价值可能低于成本，可以选择 ASK 关键 Owner、缩小范围、提供候选设计而不执行危险业务动作。

## 四、最小流程是保持独立风险底线后的最小充分证据

审核、测试与发布控制有成本，也有防错价值。若存在强制的 GitHub Required Check、真实 Host 验收或独立 Reviewer 资格，不能为了节省成本随意省略。对没有材料变化且等价受控环境已完成的测试，无意义重复 CI 可能没有额外信息增益。

教学判别：
- 文档拼写修正、未触及已测业务输入闭包：可能只需基本文档审阅与链接验证；
- 修改资金转移状态条件：必须核实有权 Rule、受影响提交与远端 Effect/失败路径；
- 改二进制打包或发布路径：必须区分编译、Target Installer 与实际 Release 授权；
- 外部不可逆操作缺少授权：不论验证成本多低，都应停在有权边界外。

优化问题可以理解为：
`min TotalDeliveryCost(implementation + rework + validation + coordination + failure loss)`
`subject to AuthorizedScope, required correctness, explicit host/provider constraints`。

这个符号不表示所有风险都能用货币换算，尤其不可逆后果、人格权或项目禁止的动作必须作为硬约束；流程优化的目标不是让模型自己取消外部授权。

## 五、实验不能只看完成样本

对比两个 Agent 策略时，先对全部预分派任务算合格完成率和总体成本：超时、NOT_RUN、失败、人工接管均保留，质量是否达标由独立 Oracle 判断。若只计算“成功任务平均 Token”，会在后处理结果上选择样本，形成选择偏差，让低成功率策略看似更省资源。

对反事实“更多 Agent 是否改善结果”，还需控制任务难度、工具/模型版本、项目规模、随机分派和观察窗口；小样本差异给出不确定范围，不能直接认定因果。参考 [K11 R5 比较实验](advanced-empirical-evaluation-and-value-of-knowledge.md)。

## 六、何时停止扩充知识库

追加“更专业的知识”也有成本。若一个已受审的 K04 Focus 已能准确解释 DONE≠ACCEPT，就不应为提高 A 链接数字重复写一篇同文。若真实任务暴露了此前遗漏的 Source/Owner/Effect 判别和高后果反例，则应增补最小必要章节并追溯它解决的失败假说。

**反例训练**：一名 Agent 以“3 Agent 比 1 Agent 更快交付 12 个 PR”为结论，但只展示成功 PR、忽略失败任务、人工审查/回滚成本且两组任务复杂度不同。合理答案必须要求同分母合格结果、完整资源与质量门槛，并对未知输入明确 `NOT_MEASURED`。有权 Review/Release 仍不能凭理论公式签发。

**参考来源**：[A04 ISO 12207 软件生命周期](../external-source-catalog.md)、[A10 ISO 15939 度量](../external-source-catalog.md)、[A33 DORA Research](../external-source-catalog.md)、[A40 SPACE](../external-source-catalog.md)、[A54 ACM SIGSOFT Empirical Standards](../external-source-catalog.md)、[A69 DORA 官方度量历史](../external-source-catalog.md)。均为学科参考，非实际成本核算或真实模型评估。