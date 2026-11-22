# K11.TRADEOFF · 工程方案的总成本、质量下限和可逆性权衡

> [K11 知识库](../README.md) · [工程方法](../engineering-methods.md) · [Agent Skills](../agent-skills.md) · [案例](../casebook.md) · [原始依据](../sources.md) · [外部资料目录](../../external-source-catalog.md)。
>
> `KNOWLEDGE_TOPIC_CANDIDATE / NOT_SOURCE_CLAIM_QUALIFIED / PROPOSED_NOT_CONNECTED`。本文件是原创工程问题知识切片，不是 Tally 已生效的机器 Schema，也不是当前 Task/Review/Host/Release 执行许可。案例仅为合成教学示例。

## 1. 工程问题和适用前提

实现同一用户需求可以复用现有模块、引入外部依赖或自建框架；如何合理比较，而不是只看初始 Token 或代码行数？

**开始前必须确认：**
1. 一致的用户结果/必要功能和质量下限
2. 至少两个真正可行的方案与环境/依赖前提
3. 已有实测/估计的开发、测试、集成、修复与维护成本

**学科知识与因果机制：** 工程经济性依赖完成结果、总生命周期成本、风险/失败后果、机会成本和可逆性。真实成本既有开发与 CI/Agent 时间，也有集成、Review、Bug 返工、依赖维护及用户环境成本。SPACE 研究提醒不能用一个活动指标代表开发者生产率；没有财务数据时应把估计和 UNKNOWN 分开，避免伪精确。

## 2. 可执行的有界判断过程

1. 先排除不能满足当前业务/权限/质量保证底线的方案，降低成本不能用来豁免合法 Review/Host。
2. 按同一维度列复用、引入依赖、自研的时间/资源/升级/兼容/风险与可逆性，区分 OBSERVED、ESTIMATED、UNKNOWN。
3. 识别材料性未知：一次小型 Demo 能否改变决策？若不能，则不为得到漂亮报告重复试验。
4. 选在真实目标下最小充分且可解释的方案，定义停止条件/替代方案和需关注的后续指标。

## 3. 场景分支、替代方案和不适用范围

| 条件 | 推荐的判断及范围 |
| --- | --- |
| 复用现有接口可完整实现 | 无需新框架，优先验证兼容和受影响测试。 |
| 引入依赖真实有长期维护风险 | 评估许可证/平台与替代成本，不按下载量下判断。 |
| 重要数据不可逆且测试要求高 | 即使只有三行变更也保留必要迁移/恢复验证。 |

## 4. 真实可证伪的教学情景

**受限教学输入及正确路线：** 一个只需添加 CSV 字段的小需求，由现有导出模块增量修改可在一个 Task 完成；新建通用报表引擎引入多个部署/兼容/Review 成本。

**失败路线/反模式：** 仅因团队 Agent Token 即将失效，就将局部需求拆成 12 个 Task 和 8 个交叉写入的 Builder，以 PR 数量展示速度。

**最低可证伪 Oracle：** 方案报告必须明确共同的需求目标和不可降低的验证范围；如果只给“Token 最少所以质量最好”，判定证据不足。

## 5. 最小交接与明确的 STOP

**Agent 输出契约**：当前已观察事实/证据身份、选择的知识与前提、备选方案取舍、正反 Oracle、未得来源或环境、最小下一步。凡 Source Claim 未获得正式资格、Host/CI 未运行、Product/Task 未授权的内容，必须与可证实事实区分。

**跨模块交接**：K04 负责 Task 划分，K07 负责协作成本，K08 负责实际构建成本，K05 负责质量保证必要性。

**停止规则**：需要越过合法写集、缺少有权业务决定、结果可能有不可逆副作用、外部状态不可确认或没有真实执行来源时，仅提出最小受控检查并保持 UNKNOWN/NOT_RUN，不通过重复调用或虚构状态推进。

## 6. 外部来源与可采纳边界

- [SWEBOK Software Engineering Economics](https://www.computer.org/education/bodies-of-knowledge/software-engineering/topics)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。
- [SPACE Developer Productivity](https://www.microsoft.com/en-us/research/publication/the-space-of-developer-productivity-theres-more-to-it-than-you-think/)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。
- [ISO/IEC/IEEE 15939:2017](https://www.iso.org/standard/71197.html)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。

**已发布 ADS 约束**：`kaicreator-mm/ai-development-standard@933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。遇到差异应按当前有权规范与 Product/L2/Task/Host/Release 的真实作用域处理；未合并的三阶段 DAG 候选不自动变为正式工作图。
