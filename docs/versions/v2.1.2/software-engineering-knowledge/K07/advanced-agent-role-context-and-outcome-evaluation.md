# K07 进阶协作知识：角色独立、知识上下文最小化与真实成果评估

> [课程](systematic-syllabus.md) · [R6 Attempt/Source 来源隔离](advanced-agent-concurrency-provenance-and-review-isolation.md) · [K11 工程经济](../K11/advanced-empirical-evaluation-and-value-of-knowledge.md)
>
> **R8 DOCUMENT_CANDIDATE / NO_ACTUAL_AGENTS_EXECUTED / EVALUATION_NOT_RUN。** 新增进阶正文精确针对 **K07.F2.1 Planner/Builder/Reviewer/Validator**、**K07.F3.3 上下文选择与精简**、**K07.F5.2 评估与协作反馈**三项。对 K07.F4.3 并行冲突仅作关联；不更改 Tally 真实 Session/Provider/Host ABI。

## 一、角色分离的实质是可反驳的证据，不是称号

Planner 提议任务依赖；Builder 产生写入与工件；Validator 运行指定 Oracle；Reviewer 独立解释 Diff、合同、实际证据并给出范围受限的结论。四个角色可以在某些小任务中由少数实际执行者承担，但**受项目规则明确要求的独立性**不能靠给同一写作会话换角色名称达成。

合成例子：Builder 基于 HEAD H1 修改路由、提交 PASS 声明；另一会话 Reviewer 基于 H1 检查并出具 `ACCEPT_SCOPED`；随后 Builder 改动授权检查到 H2。此前独立审查仍证明 H1 当时检查过什么，但不能证明 H2 的受影响权限入口同样正确。需要针对 H1→H2 的真实材料差异复核，而不是将 Reviewer 的新意见视作“否定过去所有未变内容”。

**错误推理**：“同一 GitHub 账号的另一独立对话不是独立 Reviewer”，如果没有项目有权合同要求不同账号，这个推断没有依据；正确审查维度是执行会话/是否写作、自身审核限制、Exact SHA 与可证伪证据。其他项目的身份政策可能不同，不能据此普遍断言相同账号总可接受。

## 二、选择上下文是一种带遗漏风险的检索决策

Agent 不能简单把 R7 阶段的 203 篇文档（或 R8 当前的 210 篇）全塞进当前推理就称完整，也不能只给一个 Topic 标题就声称拥有正确来源。需要从 Goal 及实际权限、Source 版本、知识 Owner 和当前 Oracle 推导**任务充分的最小证据集**。

例如“请确认下游重放任务会不会重复扣款”：
- 必要：K01 当前受权业务规则、K02 本地/远端状态边界、K03 幂等机制、K05 反例 Oracle、真实 Provider 外部合同及当前权限；
- 可能有用：K10 的超时/背压、K08 当前部署 Artifact 身份；
- 初始无材料作用：与该付款任务无关的 UI 颜色指南、所有 11 模块目录全文；
- 决定性 UNKNOWN：Provider 是否真的支持指定业务键的幂等/查询、其有效期和覆盖的 Actor/租户范围。

若恰好省略了外部 Provider 合同，LLM 可能“推理”出本地 Serializable 保证端到端不重复付款。**少上下文**会遗漏事实；**多上下文**也可能扩大检索噪声和成本，使先前已过期的候选 Rule 抢占有权新规则。真实答案要由出处与有效条件判定，而非由被检索次数投票。

## 三、Context 的充分性不能靠 Token 数量衡量

设 `C_min` 是支持正确结论的最小任务相关证据集合。它不是预先计算好的通用常数，也不等于向量检索 Top-5；其充要性应通过**删除单条关键证据后能否产生错误结论**和**添加大量无关材料是否改变有权结论**测试。

构造两组对照题：
1. 固定同一个 Goal、Source SHA 和真实 Oracle，给一组含现行业务 Rule、Provider 协议的完整最小上下文；
2. 删除当前 Rule 或 Provider 查询边界，保留大量相似旧 FAQ；
3. 比较 Agent 是否在需要时输出 `ASK/UNKNOWN`，而非自造确定业务规则；
4. 加入与任务无关的长篇资料，观察错误率、输出一致性和总成本是否恶化。

**结果限定**：这是一项未来可执行的实验设计，当前没有任何实际 Agent 对照运行，不报告改进百分比。省 Token 可以降低成本，但若合格任务完成率下降则不应以 Token 减少本身判胜。

## 四、质量底线之上的协作经济学

评估“两个 Agent 优于一个 Agent”不能只用生成内容或 commit 数量。建议预先定义每个分配任务的终点，包括失败、超时、人工接管、重复 Review、增量复验、API 费用、CI 资源、外部 Host 调用成本。按照 [K11 受控实验](../K11/advanced-empirical-evaluation-and-value-of-knowledge.md) 的质量底线与 all-assigned-task 分母分析，成功任务的平均延迟只是条件性描述，不能当整体协作因果提效。

例如同一组 20 个**假定**任务：
- 方案甲单 Agent：16/20 合格，用 100 资源单位；
- 方案乙三 Agent：18/20 合格，用 150 资源单位；
- 则不加权单位资源下甲 0.16 合格/资源，乙 0.12 合格/资源。乙质量完成比例更高但资源效率更低，不能不说明目标权重就宣布一个全局赢家；不同任务难度、资源含义与统计不确定性亦未被实际校准。
- 还要注意如果自动化把具有高失败风险的任务转交人工，漏计人工成本会虚增“Agent 效率”。

该演算是**原创合成数值**，没有真实 Agent benchmark 或生产统计依据。

## 五、协作反馈不能把一次失败固化为无条件命令

曾有 Agent 在 H1 检查通过、H2 路由变动后出现授权退化。可复用经验候选应是“在 Reviewer 证据绑定旧 SHA、且新提交触及有权授权入口的条件下，必须审当前受影响逻辑”。它不应变成“任何 Markdown 改动都必须全量跑所有 Windows Host 测试”。每条经验至少包含来源、触发条件、实际失败模式、反例和失效事件，未来由有权的 DomainHarness Experience/Evolution 版本决定是否以及如何消费。

不确定事实保持 UNKNOWN；无新材料时停止扩大 Agent 数量与 Review 阶段。K04 负责任务真实偏序，K05 负责 Oracle，K08 负责 Artifact/Release 事实，K11 负责经济对照。

**参考来源**：[A20 Microsoft Engineering Playbook](../external-source-catalog.md)、[A23 GitLab Workflow](../external-source-catalog.md)、[A28 SWE-agent](../external-source-catalog.md)、[A29 OpenHands SDK](../external-source-catalog.md)、[A40 SPACE](../external-source-catalog.md)、[A69 DORA 官方度量](../external-source-catalog.md)。没有调用这些工具运行测试，也没有创建生产 Agent 或自动化。