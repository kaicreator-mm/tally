# K10 · Agent Skills / 任务级操作技能

> **Tally v2.1.2 软件工程知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #531](https://github.com/kaicreator-mm/tally/issues/531) · [总索引](../README.md) · ADS 正式基线 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。当前只完成文档化的知识设计，尚无正式 Source→Claim 资格或真实 Package 执行。

## Tally Workflow/Business 实际与拟议消费者

## 5. Tally 工程 Workflow 消费关系

**主要消费场景（条件性拟议）：** `PROPOSED_NOT_CONNECTED`；现有 Engineering Business L1/L2/L3 的运行质量约束选择和 Maintenance/Incident 反馈读者按实际应用形态选用。输入：应用形态、质量目标、错误/运行收据、当前配置与真实负载；产出：可行动错误合同、适用重试/恢复场景或可复验性能计划；业务设计、Host/Incident/Release 仍由其原权威裁定。

| 时刻 | 既有来源输入 | 该知识提供 | 不得取代的工程权威 |
| --- | --- | --- | --- |
| 需求与架构 | 质量属性/用户支持范围 | 日志/重试/配置/性能质量契约 | 不能自行发明强制 SLO |
| 代码实施 | Task/真实错误与环境 | 可行动错误/有界恢复/必要日志 | 日志不是 Test PASS |
| 真实运维 | 运行/事故/Host 证据 | 定位故障与恢复验证 | 恢复不改写先前失败历史 |
| 性能改善 | 固定输入/基线/对照 | 可重复性能变化报告 | 不能凭印象宣称收益 |

**跨模块关系：** K01 确定用户可见的可靠性与配置要求，K02 处理跨边界一致性和部署策略，K03 实现日志/错误/恢复，K05 设计故障注入和质量测试，K06 定位故障根因，K08 验证真实安装和配置，K11 衡量收益是否值得持续维护成本。

## 现有模块 Skill：触发、输入、结果与拒绝

## 6. Agent Skills 与输入输出契约

### SK-K10-01 · Error and Log Contract Designer

- **调用条件**：UI/CLI/后台服务需改善诊断体验
- **输入**：真实用户角色、错误类型、应用形态与数据保护规范
- **输出**：错误码/原因/行动/日志字段/示例及最小测试
- **核验要点**：用户无需查看源码就能理解下一步，不泄露敏感信息
- **拒绝或升级条件**：不能对所有产品输出完整堆栈/环境变量

### SK-K10-02 · Reliability Scenario Builder

- **调用条件**：超时、取消、重试或恢复是实际产品要求
- **输入**：当前提交点/存储/外部 Provider、状态机/异常语义
- **输出**：正常、重复、超时、部分成功、迟到、恢复的可证伪案例
- **核验要点**：UNKNOWN 与已提交结果被准确区分，重试有限
- **拒绝或升级条件**：不得“重试直到成功”或保证没证实的 exactly-once

### SK-K10-03 · Performance Evaluation Planner

- **调用条件**：性能要求/投诉有具体来源
- **输入**：目标指标、输入数据、设备/平台和可重复基线
- **输出**：profiling/对照/正确性回归和测量报告
- **核验要点**：测量可复验且可说明条件差异
- **拒绝或升级条件**：没有基线不得在文档里写数字改善百分比

## 单次 Agent Skill 提示词（直接文本，可按真实任务填写）

```text
OUTPUT_LABEL=TALLY_V212_K10_SCOPED_KNOWLEDGE_APPLICATION
VERSION_ISSUE=kaicreator-mm/tally#521
MODULE_ISSUE=kaicreator-mm/tally#531
STATUS=KNOWLEDGE_DESIGN_CANDIDATE_NOT_RUNTIME_AUTHORITY
ADS_SOURCE_SHA=933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c
TARGET_REPO=<actual repository>
TARGET_TASK=<actual issue/approved scope>
TARGET_HEAD=<actual exact source SHA>
ROLE=SCOPED_ENGINEERING_ANALYST

你是工程运行质量 Agent。输入真实用户/服务运行形态、质量属性来源、具体日志/错误/超时收据、执行平台。先区分用户可行动错误与技术诊断并检查敏感信息边界；针对可能重复副作用的操作定义有限重试、幂等/读回、取消及 UNKNOWN；性能问题要求同条件基线、真实 profiling 与回归。输出适用案例/负例和未验证事项。不新增通用 APM 平台，也不将 Runtime Observed 自动作为 Release PASS。

输出按 OBSERVED / SOURCE_BACKED / PROPOSED / UNKNOWN / NOT_RUN 分类。
仅完成当前这一次有界任务，结束时输出完整结果和剩余阻塞。
```

## 结果结构的语义要求（不是新 Wire Schema）

- 事实与数据来自当前用户、真实 Issue、现行文档/源码/日志，且能复现来源版本。
- 选择的知识有适用性理由和必要反例，不凭文档标题或技能名称选择。
- 明确真实权威保留于 Task/Claim/Review/Host/Release 原 Owner；通信/技能建议不代授权。
- 未执行的代码验证写 `NOT_RUN`，证据不足写 `UNKNOWN`，不要编造 PASS 或时间/成本。