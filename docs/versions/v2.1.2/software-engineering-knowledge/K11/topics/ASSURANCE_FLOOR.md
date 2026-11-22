# K11.ASSURANCE_FLOOR · 避免过度测试同时不降低真正必要的质量保证

> [K11 知识库](../README.md) · [工程方法](../engineering-methods.md) · [Agent Skills](../agent-skills.md) · [案例](../casebook.md) · [原始依据](../sources.md) · [外部资料目录](../../external-source-catalog.md)。
>
> `KNOWLEDGE_TOPIC_CANDIDATE / NOT_SOURCE_CLAIM_QUALIFIED / PROPOSED_NOT_CONNECTED`。本文件是原创工程问题知识切片，不是 Tally 已生效的机器 Schema，也不是当前 Task/Review/Host/Release 执行许可。案例仅为合成教学示例。

## 1. 工程问题和适用前提

只改三行数据库迁移是否可以用 Fast Path 跳过升级/恢复测试？反过来，纯文档变更是否需要完整 Host CI？

**开始前必须确认：**
1. 变更对象和用户/数据/权限影响
2. 已有合法验证证据、Source SHA、Provider 环境
3. 受影响的原始合同、测试/部署/独立 Review 要求

**学科知识与因果机制：** 测试/验证强度取决于真实行为影响、失败后果、现有权威要求和实际新证据，而不仅是代码行数。必要保证下限（assurance floor）不因快速交付而下降；已经合格且相同适用范围/Exact SHA 的证据可合理复用，但不存在的特定平台/Host/独立性要求不能由别处的 Unit PASS 替代。12207 生命周期标准并不要求一种固定流程。

## 2. 可执行的有界判断过程

1. 按失效后果而不是 diff 大小评估不可逆数据、权限、公共合同、首次运行/配置等真实风险。
2. 核已通过的相同源/环境/对象证据能否在本次要求范围内复用；逐项说明未受影响项为何可跳过。
3. 对真正受影响行为设计最小正反 Oracle；在必须真实 Host/Provider/independent Review 时仅安排对应验证。
4. 新证据证明充分后收敛，不能为追求完美数字而循环增加 CI；无真实环境则保留 NOT_RUN。

## 3. 场景分支、替代方案和不适用范围

| 条件 | 推荐的判断及范围 |
| --- | --- |
| 纯 Markdown 导航改动且不影响执行 | 通常文档检查足够，无需运行全项目。 |
| 数据库迁移即使改动三行 | 必须覆盖实际历史数据/恢复等适用风险。 |
| 已有旧 SHA 通过 | 不可直接用于新源码/新测试证明，除非范围合法且不变。 |

## 4. 真实可证伪的教学情景

**受限教学输入及正确路线：** 只改安装说明里的导航文本，不动程序/CI；应检查链接和文档，不为此启动全量端到端测试。另一个三行迁移会永久删除旧字段，必须优先验证真实旧数据和补救能力。

**失败路线/反模式：** 为了“节省一次 CI”把不可逆迁移的真实升级测试关闭，或者为 Markdown 改字强制执行几十个全量 Build。

**最低可证伪 Oracle：** 给出仅以代码行数为风险指标的方案时应拒绝；给出实际失效后果和源证据后应选择最小必要验证，不制造重复 Gate。

## 5. 最小交接与明确的 STOP

**Agent 输出契约**：当前已观察事实/证据身份、选择的知识与前提、备选方案取舍、正反 Oracle、未得来源或环境、最小下一步。凡 Source Claim 未获得正式资格、Host/CI 未运行、Product/Task 未授权的内容，必须与可证实事实区分。

**跨模块交接**：K05 设计 Oracle，K08 执行 CI/Host，K09 处理数据迁移，K04 调整实际 Task 计划。

**停止规则**：需要越过合法写集、缺少有权业务决定、结果可能有不可逆副作用、外部状态不可确认或没有真实执行来源时，仅提出最小受控检查并保持 UNKNOWN/NOT_RUN，不通过重复调用或虚构状态推进。

## 6. 外部来源与可采纳边界

- [ADS Assurance Plan v4.19.0](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/ASSURANCE_PLAN_STANDARD.md)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。
- [ISO/IEC/IEEE 12207:2026](https://www.iso.org/standard/90219.html)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。
- [ISTQB CTFL](https://istqb.org/certifications/certified-tester-foundation-level-ctfl-v4-0/)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。

**已发布 ADS 约束**：`kaicreator-mm/ai-development-standard@933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。遇到差异应按当前有权规范与 Product/L2/Task/Host/Release 的真实作用域处理；未合并的三阶段 DAG 候选不自动变为正式工作图。
