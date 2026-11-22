# K07 进阶多 Agent 协作：来源身份、交付协议与独立判断

> [K07 课程](systematic-syllabus.md) · [会话来源第一卷](foundations-deep-dive-01.md) · [人因/审查第二卷](foundations-deep-dive-02.md)
>
> **R6 ORIGINAL_PROFESSIONAL_KNOWLEDGE_CANDIDATE / NO_AGENTS_DISPATCHED / NO_ROLE_AUTHORITY_CREATED。** 映射 K07.F1.2、F2.2、F3.1、F3.2、F4.1、F4.2、F5.1。讨论受控协作机制，不假定 Tally 当前的真实 Provider/Host 已实现以下教学协议。

## 一、分工结构不等于行为证据

多个 Agent 各自对工作报 `DONE`，只说明多个消息存在，**不说明**当前 GitHub HEAD 下可集成成果完整，也不说明任何 Reviewer 已形成独立判断。建立三个不同问题：
1. **任务能力**：谁能读哪些 Input、写哪些 Output、调用哪个 Provider/Host；
2. **并发正确性**：不同执行是否读取同一个受权 SourceSHA、共享哪些写集；
3. **证据与授权**：谁提供的材料可供复查，谁有权宣称某范围 ACCEPT。

同一 GitHub 账号的不同 ChatGPT 对话可以在项目允许的前提下扮演独立 Reviewer；关键是**不能由同一写作执行会话自我给出独立审核**。身份/独立性要求要遵从真实规则，不应凭一个工具调用自行发明“不同账号”强制门槛。

## 二、四种标识不可以互相代替

教学合同：
- `TaskId`：工作语义单元（可跨多次尝试）；
- `AttemptId`：某次具体执行尝试，必须可区分重复/迟到；
- `SourceSHA`：读取的确切版本，旧提交可能有效但仅对旧范围；
- `ArtifactDigest`：交付内容身份，单一 SourceSHA 可有多个可验证输出；
- `Actor/Role/Scope`：消息来源与可执行权限，并不因持有 Artifact 自动增加。

消息包候选结构：`{task, attempt, sourceSHA, outputs, evidenceRefs, actor, claims, timestamp}`。这里只是**学术性分析字段**，不等于现有 Tally GitHub Wire Schema 或 DomainHarness 0.11.x Kernel Port ABI。

**迟到反例**：A@H1 运行较慢；B@H2 已提交受影响 API 修复；A 返回 `DONE` 带 H1 测试 PASS。若工作台把最后到达的消息当“最新结果”，H1 的证据可能错误覆盖 H2 的当前事实。正确做法是保留 A@H1 历史结果及其实际可复用范围，受 H2 影响部分仍由当前身份/Oracle 重新判断。

## 三、共享写集如何影响并行收益

两名 Agent 分别修改 `src/auth.ts` 与 `src/router.ts`，表面路径互不冲突，但若 Router 引入新路径绕过 Auth，语义依赖使原 Auth 审查不再完整。另一组 Agent 修改完全独立的帮助文档，语义交叉几乎无，则不应以“同一个仓库”为由串行。

候选并行判据：`SafeParallel(A,B)` 需要同时讨论 `WriteSet` 交叉、`ReadSet` 被改变风险、共享接口/生成物、资源互斥、外部不可逆效果以及权威提交顺序。文件写集无交集只是初筛，并非充分证明。

**协调策略比较**：
| 方案 | 何时合理 | 失败/代价 |
| --- | --- | --- |
| 一名 Builder 端到端完成 | 小且耦合紧的任务 | 并行度低，但集成风险小 |
| 冻结共享合同后并行，最后有界集成 | 不同实现边界、合同已足够明确 | 合同变更触发受影响复查 |
| 多名 Builder 同改一处，由 Shared Owner 仲裁 | 更复杂/确需同时研究备选 | Merge 代价与权威冲突升高，通常不适于简单工作 |

多 Agent 应以**总体合格交付耗时/错误率和协作成本**评价，不以生成 Token 或同时启动的 Agent 数量评价。

## 四、Review 独立性与 Reviewer 自身的证据边界

独立 Reviewer 不应该只是询问 Writer“你修好了吗”，而应基于确定的 HEAD 和变更范围，追溯真实 Diff、合同、必要的正反例与已执行/未执行的测试。若 R1 只审过 H1，Writer 在 H2 新增了代码，不能把 R1 ACCEPT 直接变成 H2 ACCEPT。但 H2 未触及的旧部分，也不必再做一次全文重审；**受影响范围可证明时有界复用**。

一个被批准的流程仍需区分审查意见、正式代码合并、Product/L2 发布、真实 Host/Effect 权限。Reviewer 可以确认文档专业准确性而**不能凭此**把文档 Skill 推成生产 Rule。

## 五、消息故障与恢复

| 故障 | 错误处理 | 更安全的条件性处理 |
| --- | --- | --- |
| Worker 消息重复 | 创建两个互斥“当前完成”结果 | 以 Attempt 和真实提交身份分类重复 |
| Worker 中断/超时 | 立即认为外部 Effect 未执行 | 查权威 Effect；未知保持 UNKNOWN |
| Agent 改写了共享配置 | 因文件可合并而跳过受影响测试 | 查询 Consumer/Build 输入影响闭包 |
| Reviewer 被要求给尚不存在的 SHA 放行 | 按预计将生成的 Diff 预签 | 只报告实际可见 SHA 与对应限定范围 |
| Agent 在旧分支持续运行 | 无限分派“直到完成” | 有界任务及停止条件；剩余必要事宜重新提交 |

**最小反事实测试**：让两个 Agent 分别基于 H1/H2 修复相关业务合同，再交换到达顺序。任何仅因消息先后就反转“当前 HEAD 与权威决定”的方案都不满足来源约束。要区分“旧证据仍为历史事实”和“旧证据满足当前受影响资格”。

## 六、从协作反馈到可信经验

对协作实践的改善，应记录一次冲突为什么发生、共享接口/写集在哪里、什么可证伪观测区分“模型不会做”与“任务计划错误”、修复是否实际减少重工。若缺独立基线，不应宣称增加第三个 Provider 就提升交付成功率。

**专业交接**：K04 定义任务因果偏序与必要 Owner，K06 分析失败，K08 提供真实代码/Artifact/CI 来源，K05 给出独立 Oracle，K11 衡量投入产出。K07 拥有协作/消息语义但不接管其他模块的正式权威。**STOP**：只有真正不可核实的受影响提交/权限需要等待，其他可独立推进。

**来源线索**：[A20 Microsoft Engineering Playbook](../external-source-catalog.md)、[A23 GitLab Workflow](../external-source-catalog.md)、[A28 SWE-agent](../external-source-catalog.md)、[A29 OpenHands SDK](../external-source-catalog.md)、[A30 Agent benchmarks](../external-source-catalog.md)、[A41 SWE at Google](../external-source-catalog.md)；来源可用于学习而非当前 Task/Review 状态机证明。