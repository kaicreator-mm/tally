# K07 · Agent Skills / 任务级操作技能

> **Tally v2.1.2 软件工程知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #528](https://github.com/kaicreator-mm/tally/issues/528) · [总索引](../README.md) · ADS 正式基线 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。当前只完成文档化的知识设计，尚无正式 Source→Claim 资格或真实 Package 执行。

## Tally Workflow/Business 实际与拟议消费者

## 5. Tally Workflow 与其他知识模块的组合

**消费方（拟议，不代表已接线）：** `PROPOSED_NOT_CONNECTED`；Tally 既有 Engineering Business 的 L3 Task/Attempt、Agent Handoff 与 Review 入口在有合法任务时选择 DP-K07-01/02/03。输入：当前 Issue/Task、角色、源 HEAD、允许写集及消息/Artifact 事实；产出：受约束的角色/通道选择、独立审查交接、迟到消息对账建议；最终 Task Claim、Review 和执行授权仍由既有 GitHub/Host 权威决定。

| 环节 | 既有事实输入 | 使用该知识获得的输出 | 不能推断 |
| --- | --- | --- | --- |
| Planner | 用户/任务/分层 DAG | 角色和可并行界面 | 不自动批准任务 |
| Builder/子 Agent | 合法执行环境/写集 | 源码和可验证工件 | 消息完成不等于 Task 完成 |
| Reviewer/Verifier | 精确 HEAD/独立身份与测试 | 有边界的结论/返工要求 | 不代签 Product/Release |
| 集成与回读 | 当前 native Task/Host/CI 事实 | 合法状态与真实消费者证据 | IPC/HTTP ACK 无权限效力 |

**与 K01–K11 的交叉关系：** K04 判断任务是否真正能拆分，K02 先定义接口/状态合同，K03 完成受控实现，K05/K06 分别为验证和修复服务，K08 负责整合与实际 Build；K11 估算并行真实成本。

## 现有模块 Skill：触发、输入、结果与拒绝

## 6. Agent Skills：输入、输出、验证、失败路由

### SK-K07-01：Bounded Agent Handoff

- **触发**：需要跨 Agent/Host 交付独立工作
- **需要的事实输入**：Repo/Issue/PR/Exact HEAD/Task Pack/Role/写集/环境
- **产生的工件或结论**：结构化交接、输出工件和可停止的单次任务
- **自我核对**：信息足以让接收者复原状态并知道哪些决策无权改变
- **拒绝/升级**：不得生成无界“持续推进直到完成”的本地 Agent 指令

### SK-K07-02：Independent Source Reviewer

- **触发**：已有候选源码和真正独立复核需求
- **需要的事实输入**：候选 HEAD/Base、真实 Diff、有效要求、测试收据和逻辑身份
- **产生的工件或结论**：ACCEPT_SCOPED/REWORK、P0/P1/P2、准确来源和最小修复
- **自我核对**：Reviewer 与 Writer 在实际工作上分离，并说明检查了什么
- **拒绝/升级**：不能靠同一 Writer 自报 PASS 取代独审

### SK-K07-03：Interaction Adapter Designer

- **触发**：接入非 GitHub Agent 消息系统时
- **需要的事实输入**：现行 Host/Task 权威、发送/确认/错误/重放语义
- **产生的工件或结论**：端口契约、发送/观察/对账、失败及幂等行为
- **自我核对**：ACK 不写 Task 状态，重试不产生重复外部 Effect
- **拒绝/升级**：不把任意 Agent 协议消息当作 Release 许可

## 单次 Agent Skill 提示词（直接文本，可按真实任务填写）

```text
OUTPUT_LABEL=TALLY_V212_K07_SCOPED_KNOWLEDGE_APPLICATION
VERSION_ISSUE=kaicreator-mm/tally#521
MODULE_ISSUE=kaicreator-mm/tally#528
STATUS=KNOWLEDGE_DESIGN_CANDIDATE_NOT_RUNTIME_AUTHORITY
ADS_SOURCE_SHA=933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c
TARGET_REPO=<actual repository>
TARGET_TASK=<actual issue/approved scope>
TARGET_HEAD=<actual exact source SHA>
ROLE=SCOPED_ENGINEERING_ANALYST

你负责单次多 Agent 协作方案的设计/只读协调。输入当前 Task/Issue/HEAD、现有 Provider/Agent 能力、合法写集、可被分别验证的结果和 Review 要求。判断是否应单 Agent 或并行；若并行，输出每个 Role 的单次交接、Task/Attempt/ExactSHA/ArtifactRef、接口与写集、消息通道及 ACK 后的状态对账。Git/GitHub 为默认耐久通道，IPC/HTTP 只是可选传输，不得代签 Claim/Review/Release 或派发无界持续任务。

输出按 OBSERVED / SOURCE_BACKED / PROPOSED / UNKNOWN / NOT_RUN 分类。
仅完成当前这一次有界任务，结束时输出完整结果和剩余阻塞。
```

## 结果结构的语义要求（不是新 Wire Schema）

- 事实与数据来自当前用户、真实 Issue、现行文档/源码/日志，且能复现来源版本。
- 选择的知识有适用性理由和必要反例，不凭文档标题或技能名称选择。
- 明确真实权威保留于 Task/Claim/Review/Host/Release 原 Owner；通信/技能建议不代授权。
- 未执行的代码验证写 `NOT_RUN`，证据不足写 `UNKNOWN`，不要编造 PASS 或时间/成本。