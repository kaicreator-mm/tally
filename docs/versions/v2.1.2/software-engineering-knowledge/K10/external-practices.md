# K10 · 可运维性、可靠性与性能：权威实践与可证伪案例

> [K10 知识库](README.md) · [v2.1.2 #521](https://github.com/kaicreator-mm/tally/issues/521) · [外部来源总目录](../external-source-catalog.md)。**EXTERNAL_DISCIPLINARY_SOURCE_CANDIDATE / NOT_SOURCE_QUALIFIED / NOT_CONNECTED**；本文件为独立原创工程知识与实际方法的总结，非受版权保护的国际标准/教材原文复刻，不提升现有 Task/Release 权威。

## 从公开实践提炼有效技术，而非复制别人的流程

这里的实践案例是独立构造的**教学输入**，用外部权威/成熟项目的思想解释 Agent 应如何作决策。公开 GitHub 项目的原始能力受技术栈、依赖、执行环境和许可约束；其 README 或 Repo Star 不是验证当前 Tally Task 的权威。相应实例与成熟度都要逐项核对。

### P-K10-01 · 首次启动后的应用配置与错误

**合成任务输入：** 输入：交互桌面应用还没有选择仓库目录，CLI 后台脚本则通过参数配置。

**逐步操作：**
1.  根据产品形态分别规划设置入口与无头诊断。
2.  桌面首次运行允许设置但阻止未经授权业务动作。
3.  CLI 对缺失参数输出明确可行动错误。
4.  不泄漏密钥。
5.  测试错误配置、取消设置与实际恢复路径。

**正反 Oracle：** 正确：两类产品遵循不同启动体验而功能合法；错误：要求所有 CLI 必须先启动 GUI 或要求桌面应用只能设环境变量。

**出处与限制：** [A01](../external-source-catalog.md)、[A31](../external-source-catalog.md)。该示例是合成教学材料，并非宣称这些出版方已在对应项目上执行。真正可验的 Task 判断必须由受批准的业务 Oracle 和真实测试来源确定。

### P-K10-02 · 超时且结果未知的支付/订单操作

**合成任务输入：** 输入：客户端 POST 已发送，连接超时后返回 UNKNOWN，外部服务可能已提交。

**逐步操作：**
1.  记录请求 ID 与提交边界。
2.  查询可读回的真实状态。
3.  有安全幂等合同时才按次数/截止重试。
4.  若无法确认保留 UNKNOWN 并指引用户。
5.  验证重复调用不会额外产生副作用。

**正反 Oracle：** 正确：已提交只产生一次业务效果，未知时不伪造失败；错误：每次超时都强制重发创建请求。

**出处与限制：** [A32](../external-source-catalog.md)、[A31](../external-source-catalog.md)。该示例是合成教学材料，并非宣称这些出版方已在对应项目上执行。真正可验的 Task 判断必须由受批准的业务 Oracle 和真实测试来源确定。

### P-K10-03 · 性能优化与 SLO 的有界评估

**合成任务输入：** 输入：用户抱怨导出大数据时 UI 无响应，但没有现成吞吐/延迟基线。

**逐步操作：**
1.  确认真正用户目标是响应性还是整体吞吐。
2.  选对应可观察指标和代表性负载。
3.  同平台收集 baseline/候选 profile。
4.  用必要正确性回归排除副作用。
5.  报告测量误差/不适用外推与未知。

**正反 Oracle：** 正确：有条件和基线的局部改进观察；错误：基于某次无对照微基准写出‘系统整体快 300%’。

**出处与限制：** [A07](../external-source-catalog.md)、[A31](../external-source-catalog.md)、[A33](../external-source-catalog.md)。该示例是合成教学材料，并非宣称这些出版方已在对应项目上执行。真正可验的 Task 判断必须由受批准的业务 Oracle 和真实测试来源确定。

## 相邻知识如何复用

这些外部实践用于增强既有 [engineering-methods.md](engineering-methods.md) 的 3 个 Decision Procedures、[agent-skills.md](agent-skills.md) 的合法 Agent 输入输出、[casebook.md](casebook.md) 的正反案例和 [validation.md](validation.md) 的准确 Oracle。只按实际问题选择选定方法，不把研究 Source 直接提升为已采纳 MUST，不为每个练习引入新的 CI/Host 或独立 Review。
