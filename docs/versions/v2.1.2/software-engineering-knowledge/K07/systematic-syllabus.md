# K07 · 多 Agent 工程协作与专业实践：系统学科知识纲要

> [本模块问题索引](README.md) · [学科基础](disciplinary-foundations.md) · [方法与 DP](engineering-methods.md) · [Agent Skills](agent-skills.md) · [案例与反例](casebook.md) · [来源对照](sources.md) · [全局知识架构](../systematic-knowledge-architecture.md)。
>
> `SYSTEMATIC_KNOWLEDGE_CANDIDATE / NOT_SOURCE_CLAIM_QUALIFIED / NOT_CONNECTED`。本纲要包含**5 个知识群 / 15 个原创专业知识点**，用于长期系统化软件工程领域智能；不会仅因为整理过学科概念就自动取得 Task、Claim、Release、真实 Agent 效果资格。

## 一、知识本体、工程推理与能力归属

多 Agent 工程协作与专业实践 不仅是若干执行提示词。它必须提供专业原理与假设、对真实输入的条件性判断、可反证的错误做法、跨学科连接与明确资格。**当前唯一知识 Owner：工程 Agent 角色、独立会话、交互协议、共享工作写集和交接边界的唯一知识拥有者**。下面的主题是完整课程的骨架，实际深度 Topic 是对必要问题的解释与实践投影；现有 DP/Skill/Case/Source 仍为可被引用的专项资料，不强制复制 15 次。

## 1. 协作模型与组织工程

### K07.F1.1 — 协作目标与成果物

**专业知识及适用机制**：并行的目标是减少用户交付时间与错误，而非最大化 Agent 调用/消息数量；每次协作应有可检查的成果、依赖和 Owner。

**错误外推/反例**：十个 Agent 都输出“已完成”却没有真实合并成果。

**对 Agent 的可检验要求**：面对具体问题时指出真实输入、可以使用本知识的前置条件、未选择替代方案的理由和关键缺失项；缺必要事实时输出 UNKNOWN 或停止而不是自报 Rule/Host PASS。

### K07.F1.2 — 协调成本与瓶颈

**专业知识及适用机制**：跨角色上下文、等待、Review、合并和失败修复是工程成本；并行只在交接独立时净收益可观。

**错误外推/反例**：仅按订阅 Token 剩余额度决定同时启动的 Builder 数。

**对 Agent 的可检验要求**：面对具体问题时指出真实输入、可以使用本知识的前置条件、未选择替代方案的理由和关键缺失项；缺必要事实时输出 UNKNOWN 或停止而不是自报 Rule/Host PASS。

### K07.F1.3 — 人机责任边界

**专业知识及适用机制**：人/模型/CI/Host/合法 Owner 负责不同类型事实与决策，Agent 判断不能代替被保护的业务或发布操作。

**错误外推/反例**：把模型签名字符串直接当用户 Release 批准。

**对 Agent 的可检验要求**：面对具体问题时指出真实输入、可以使用本知识的前置条件、未选择替代方案的理由和关键缺失项；缺必要事实时输出 UNKNOWN 或停止而不是自报 Rule/Host PASS。

## 2. 角色、来源与权限

### K07.F2.1 — Planner/Builder/Reviewer/Validator

**专业知识及适用机制**：Planner 形成真正交付边界、Builder 写受控源、Reviewer 审当前差异、Validator 验证实际行为；同一人/账户可有不同逻辑会话但必须记录事实。

**错误外推/反例**：同一个写入会话换个 ROLE 标签就冒充独立 Reviewer。

**对 Agent 的可检验要求**：面对具体问题时指出真实输入、可以使用本知识的前置条件、未选择替代方案的理由和关键缺失项；缺必要事实时输出 UNKNOWN 或停止而不是自报 Rule/Host PASS。

### K07.F2.2 — Session/Attempt/Source 绑定

**专业知识及适用机制**：消息应关联 Task、真实逻辑会话、Attempt、SOURCE_SHA、ArtifactRef 和最终 GitHub 读回，避免结果漂移。

**错误外推/反例**：收到来自旧 HEAD 的 PASS 就覆盖新候选。

**对 Agent 的可检验要求**：面对具体问题时指出真实输入、可以使用本知识的前置条件、未选择替代方案的理由和关键缺失项；缺必要事实时输出 UNKNOWN 或停止而不是自报 Rule/Host PASS。

### K07.F2.3 — 受控执行和会话边界

**专业知识及适用机制**：有能力调用工具不意味着有权限写任意仓库/Host；边界以当前选定 Task、写集和已有 Agent Policy 为准。

**错误外推/反例**：只读分析者执行源码重构并把它称为“建议”。

**对 Agent 的可检验要求**：面对具体问题时指出真实输入、可以使用本知识的前置条件、未选择替代方案的理由和关键缺失项；缺必要事实时输出 UNKNOWN 或停止而不是自报 Rule/Host PASS。

## 3. 交互、消息与状态

### K07.F3.1 — 消息 vs canonical work facts

**专业知识及适用机制**：IPC/HTTP/queue/GitHub comment 可传输意图和结果；真正 Task/Claim/Review 当前性仍由合法 GitHub/Host 事实决定。

**错误外推/反例**：Tool ACK 被当作已提交合法 Claim/业务效果。

**对 Agent 的可检验要求**：面对具体问题时指出真实输入、可以使用本知识的前置条件、未选择替代方案的理由和关键缺失项；缺必要事实时输出 UNKNOWN 或停止而不是自报 Rule/Host PASS。

### K07.F3.2 — 超时/重复/迟到处理

**专业知识及适用机制**：ACK 丢失不证明外部 Effect 未发生，重复事件需关联稳定 Request/Attempt 并合法读回；STale 不能推进当前状态。

**错误外推/反例**：为修复一次超时无限重发不可逆操作。

**对 Agent 的可检验要求**：面对具体问题时指出真实输入、可以使用本知识的前置条件、未选择替代方案的理由和关键缺失项；缺必要事实时输出 UNKNOWN 或停止而不是自报 Rule/Host PASS。

### K07.F3.3 — 上下文选择与精简

**专业知识及适用机制**：每个 Agent 获取完成本次成果需要的最小知识闭包、源码/测试、历史证据与权限；不要读取所有 11 个知识库和全仓。

**错误外推/反例**：每次问一个 CI 错误就载入全部 150 篇知识文件。

**对 Agent 的可检验要求**：面对具体问题时指出真实输入、可以使用本知识的前置条件、未选择替代方案的理由和关键缺失项；缺必要事实时输出 UNKNOWN 或停止而不是自报 Rule/Host PASS。

## 4. 并行开发与集成

### K07.F4.1 — 独立写集与 Shared Owner

**专业知识及适用机制**：多个 Git worktree 只是物理隔离，Schema、lockfile、迁移、生成代码、公共 ABI 仍可发生语义冲突。

**错误外推/反例**：两个 Agent 同时修改 types.ts 后最后写入覆盖前者。

**对 Agent 的可检验要求**：面对具体问题时指出真实输入、可以使用本知识的前置条件、未选择替代方案的理由和关键缺失项；缺必要事实时输出 UNKNOWN 或停止而不是自报 Rule/Host PASS。

### K07.F4.2 — 稳定接口与受控合并

**专业知识及适用机制**：通过公共合同先行或单 Writer 串行共享变更，让互不冲突的实现/测试分析并行，最终在同 HEAD 验证集成。

**错误外推/反例**：三个分支各自 Unit PASS 就报告系统 E2E PASS。

**对 Agent 的可检验要求**：面对具体问题时指出真实输入、可以使用本知识的前置条件、未选择替代方案的理由和关键缺失项；缺必要事实时输出 UNKNOWN 或停止而不是自报 Rule/Host PASS。

### K07.F4.3 — 冲突/失败恢复

**专业知识及适用机制**：预定义谁接收冲突、哪些结果保留历史、何时停止多 Agent 循环，明确何种结果只能由有权 integrator 决定。

**错误外推/反例**：在遇到 Git 合并冲突时创建更多 Agent 继续写同一文件。

**对 Agent 的可检验要求**：面对具体问题时指出真实输入、可以使用本知识的前置条件、未选择替代方案的理由和关键缺失项；缺必要事实时输出 UNKNOWN 或停止而不是自报 Rule/Host PASS。

## 5. 审查、专业伦理与经验化

### K07.F5.1 — 可证伪独立代码审查

**专业知识及适用机制**：Reviewer 真实读取 Base→Exact HEAD Diff、需求和测试，给当前适用的 P0/P1/P2 及缺失项；报告不等于自动批准。

**错误外推/反例**：Review 只看 Writer 总结便签署 ACCEPT。

**对 Agent 的可检验要求**：面对具体问题时指出真实输入、可以使用本知识的前置条件、未选择替代方案的理由和关键缺失项；缺必要事实时输出 UNKNOWN 或停止而不是自报 Rule/Host PASS。

### K07.F5.2 — 评估与协作反馈

**专业知识及适用机制**：用正确交付/错误率/返工/总耗时/上下文成本衡量协作价值；不能凭 benchmark Stars/题目得分自证。

**错误外推/反例**：Agent 数量增加就等于团队效率提升。

**对 Agent 的可检验要求**：面对具体问题时指出真实输入、可以使用本知识的前置条件、未选择替代方案的理由和关键缺失项；缺必要事实时输出 UNKNOWN 或停止而不是自报 Rule/Host PASS。

### K07.F5.3 — 失误记录与可复用规则候选

**专业知识及适用机制**：保存真实同 SHA 证据、失败/成功条件与适用范围；经验可形成候选方法，但不能自动升级为生产 Rule。

**错误外推/反例**：单次 Agent 成功便自动改写所有未来工作流。

**对 Agent 的可检验要求**：面对具体问题时指出真实输入、可以使用本知识的前置条件、未选择替代方案的理由和关键缺失项；缺必要事实时输出 UNKNOWN 或停止而不是自报 Rule/Host PASS。

## 六、知识复用、专业反例与跨模块推理

**所有权**：工程 Agent 角色、独立会话、交互协议、共享工作写集和交接边界的唯一知识拥有者。K04 负责真正 Task/真实依赖，K07 负责协作和角色通信，K05 负责测试充分性，K08 负责 CI/制品来源，K11 评估协作经济性。

**专业教学情境**：过期 Agent ACK 与角色独立性。Builder 在旧 HEAD `a111` 修改代码，另一个非 Writer 会话审当时的差异；当前 HEAD 已是 `b222`，迟到 ACK/旧 Review 均不能直接更新当前 Task 结果。

**必须识别的反模式**：利用同一 GitHub 账号的登录名推断 reviewer 必然不独立，或把 Writer 的自述直接当审核完成。

**最小推理闭包**：Goal 与真实事实 → 本课程序列中的 Concept/前提 → Applicable/Excluded/UNKNOWN 判定 → [Decision Procedure](engineering-methods.md)/[Skill](agent-skills.md)/[Case](casebook.md) → 可观察 Oracle 与 STOP → 未来只有合法 Agent/Host/CI 真实运行时才得到相应验证结果。单独的系统课程文章永远不授予生产执行权。

**学科与工程来源候选**：[A01](../external-source-catalog.md)、[A20](../external-source-catalog.md)、[A21](../external-source-catalog.md)、[A23](../external-source-catalog.md)、[A24](../external-source-catalog.md)、[A28](../external-source-catalog.md)、[A29](../external-source-catalog.md)、[A30](../external-source-catalog.md)、[A40](../external-source-catalog.md)。来源目录是研究入口，不代表对全部理论主张完成 Clause、版权和正式 Claim 资格；适用的 ADS 正式规范和 [sources.md](sources.md) 仍须检查实际权限与当前版本。

## 七、从完整知识库继续增长的原则

- **系统完整性**：现有 15 个知识点为 5 个专业知识群的概念基础，可继续增加算法、理论/实证研究、不同规模/环境反例和相邻领域关联，而不是等待生产任务才允许编写知识。
- **实际使用深度**：优先把多次遇到的误判、复杂取舍和关键质量问题深化成独立 Topic，既有 Topic 通过 [README](README.md) 选择，不强制为每点创建独立文件/Issue。
- **专业准确性**：当某项资料仅列于来源目录或不能取得精确原文时，保持 `SOURCE_CANDIDATE` 并标明不能外推；后续选定 Task 需要时只审实际用到的主张与负例。
- **服务边界**：Tally 的选定领域资产可被未来 DomainHarness selected Element Content/Function 消费，但编译/运行/授权/反馈仍归 [Harness 边界](../domain-harness-consumption.md) 及原有 GitHub/Host/Release Owner；不创设第二知识 Runtime。
