# K07 · 系统学科深化 02：多 Agent 人因、协议契约与独立判断

> [系统课程](systematic-syllabus.md) · [第一卷：消息与来源身份](foundations-deep-dive-01.md) · [角色分离](topics/ROLE_SEPARATION.md)
>
> **R3 DOCUMENT_KNOWLEDGE_CANDIDATE / NOT_SOURCE_CLAIM_QUALIFIED / NO_REAL_COLLABORATION_EVAL。** 深化 K07.F1.1–F1.3、K07.F3.2/F3.3、K07.F4.2/F4.3、K07.F5.1–F5.3。此章是社会技术与协议工程知识，不实现新消息总线或 Task Authority。

## 一、多 Agent 协作是人机组织系统，不是把线程数扩大

任务适合独立 Agent 的必要条件至少包括：目标与验收可分别定义、接口已足够稳定、相互写入或决策不会未经协调相互覆盖、实际等待成本可因并行降低。人和 Agent 都存在注意力、语境迁移、确认偏差和沟通延迟；更多通信不一定减少误解。

**协作拓扑备选**：单 Builder 贯穿端到端；多 Builder 按独立写集并行；一个 Integrator 集中归并；独立 Reviewer 只评估精确输入而不写源。不同拓扑的优劣取决于真实边界，而非固定“多 Agent 一定更专业”的假设。

## 二、交接契约应让接收方能够拒绝不安全输入

一个合法交接不仅要能阅读消息，还应检查：
1. 任务来源/目标（哪个 Issue/Task/Contract）是否仍有效。
2. 输入 Artifact、SourceSHA、预期基线和输出可观测证据是否齐全。
3. 当前 Worker 的写权限、资源范围与工具 Effect 是否确实授权。
4. 会话/Attempt 是否过期、是否来自本次目标、是否发生重试或重复。
5. 交接缺少材料事实时要输出什么 STOP/UNKNOWN，而不是生成一个新的任务真相。

**示例**：Reviewer 收到 `WriterDone(A, sha=old)`，但当前 PR HEAD 为 `new`。他可以保留旧报告，并要求限定增量差异；不能以该报告作为新 HEAD 无条件通过的证据。若 Reviewer 与 Writer 使用同一登录账号但不同对话且满足真实独立身份要求，不应仅因账户相同拒绝独立性；反过来，换了账号但复制 Writer 的自评也不形成真正独审。

## 三、上下文越长不必然推理越可靠

对一个测试修复 Agent，相关信息可能是：错误 Oracle、代码入口、当前 SHA、具体日志、受影响文件及授权边界。额外装入所有 168+ 篇学科材料可能引入版本冲突、旧过程以及不适用的术语，使选择结果变差。

**有界知识选择**应返回 Concept ID、前置条件、必要来源、可用方法、一个足以反驳错误外推的 Case，以及缺少什么；其余知识通过精确链接引用。若检索到相互矛盾的文档，要求列出来源/范围差异，不能只根据相似度分数裁定谁拥有权限。

## 四、冲突恢复与独立审查不是一个 PASS 状态

| 协作事件 | 谁拥有实际事实 | 错误推断 |
| --- | --- | --- |
| 共享 Schema 冲突 | Integrator 与正式源/Task 合同 | 两支单独构建都绿即最终兼容 |
| Reviewer 提出修复 | Reviewer 给出限定意见，Builder 修复 | Reviewer 可以暗中替 Builder 修改并仍称独立 |
| 旧 Agent 返回新消息 | Source/Attempt 记录与当前 GitHub HEAD | 最新时间戳自动覆盖当前 |
| Queue/IPC 已送达 | Transport 层只拥有接收事件 | 消息送达可以创建 Release |
| 多 Agent 总成本更低 | K11 根据真实结果对照 | 仅观察 Agent Token 用量即可证明 |

审查者应提出可证伪缺陷与必要最小修正，并区分新的 P0/P1 与不影响接受范围的 P2；如无材料新证据，不应无限循环打开同样审核。用户/Owner 的正式 Product/L2/Release 决策仍然独立存在。

## 五、如何评价协作专业能力

实验需使用同一批可判断的工程任务、相当工具与预算、相同代码起点与有效 Oracle，对比一次完成率、复审工作量、因共享资源导致的冲突、整体墙钟时间、最终维护风险和总成本。既保留正收益，也记录并行造成的失败或负收益。不要将跨组织 DORA 相关结果宣称为当前协作方式的因果效果。

**教学反例**：同时启用八个 Agent 修改同一生成 Schema；每个都报告完成，合并后 API 在真实消费者路径中失效。正确的经验不是“让更多 Agent 修复”，而是先识别共享写集与唯一合同 Owner，必要时恢复受控串行。

**来源线索**：[A20 Microsoft Playbook](../external-source-catalog.md)、[A21 Google 归档工程实践](../external-source-catalog.md)、[A28 SWE-agent](../external-source-catalog.md)、[A29 OpenHands](../external-source-catalog.md)、[A40 SPACE](../external-source-catalog.md)、[A54 ACM SIGSOFT](../external-source-catalog.md)。