# K10 · 可运维性、可靠性与性能 — 软件工程领域知识库

> **Tally v2.1.2 软件工程知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #531](https://github.com/kaicreator-mm/tally/issues/531) · [总索引](../README.md) · ADS 正式基线 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。当前只完成文档化的知识设计，尚无正式 Source→Claim 资格或真实 Package 执行。

为不同应用形态设计可行动日志/错误、可靠失败与恢复，并将性能判断绑定真实基线。

**系统正文第一卷（R2 文档范围独立 ACCEPT_SCOPED @ ede9d646cb591b8df5179dc6bc11b60668fe968c）**：[学科深化 01：可靠性与未知副作用恢复](foundations-deep-dive-01.md)。此章解释术语背后的机制、适用条件、工程取舍与可证伪反例；仍为文档知识候选，不是已合格 Source Claim 或真实 Harness/Agent 功能。

**系统正文第二卷（R3 文档范围独立 ACCEPT_SCOPED @ a39a64a82beae103318da3e45bd09b5efed15ad1）**：[学科深化 02：Little 定律、尾延迟与容量](foundations-deep-dive-02.md)。本卷补充基础理论、适用条件、备选技术与可证伪反例；不宣称 165 点均已教材级展开，也不自动取得生产 Source/Consumer 资格。

**R4 逐点学科缺口补齐（独立 ACCEPT_SCOPED @ 42e7ba8bd883ec79736e25ff7c75bdbda1b738b3，旧 SHA 限定文档资格）**：[`K10.F5.1` 首次启动与运行后配置](focus-f5-1-first-run-post-start-configuration.md)。这些文件是对课程的定点专业深化，包含机制/条件/反例/STOP；并非获权 Source Claim、CI/Host/Package 或产品发布资格。

**R6 进阶原理与专业反例（旧 Exact SHA `83ebee092db412b5e38db0376562ee51b0d4f032` 已获独立 ACCEPT_SCOPED，文档范围）**：[SLI/SLO、可行动告警、未知 Effect 与容量](advanced-reliability-slo-effect-recovery-and-measurement.md)。对本模块的选定课程 ID 补充可验证的机制、适用前提、替代选项和否证轨迹；相应 [W06–W12 跨模块完整演算](../professional-reasoning-worked-traces-02.md) 均为教学而非实际运行。无 Source Claim、DomainHarness ABI 或生产 Agent/Host 权限资格。

**R9 专业基础及可证伪失效机制（新增待限定独立审核）**：[过载/背压、P95 归因和事故恢复 Oracle](advanced-overload-performance-inference-and-recovery-oracles.md)，对原系统课程中 **3 个历史高优先知识点**新增准确 A 专业正文关联；[W18–W22](../professional-reasoning-worked-traces-04.md) 是合成推理，[Source→Claim→Oracle 准入差距](../professional-source-to-claim-readiness-r9.md) 明确保留无真实权限/未运行证据。不是实际 Source Claim、Consumer ABI、Agent/Host/Release 资格。

**先了解整个专业学科，再按需选择问题**： [K10 系统课程：5 个知识群 / 15 个专业知识点](systematic-syllabus.md) 用于完整概念、原理和常见反例的持续积累；下表是每次任务的有界检索入口，并非该模块的全部知识。课程内容尚未得到 Source Claim/Domain Package 生产资格。

## 当前工程问题索引（优先入口）

先根据当前 Task 的**真实问题**命中下表最小 Topic，再阅读与之相关的 [专业方法](engineering-methods.md)、[Agent Skills](agent-skills.md)、[案例](casebook.md)、[来源](sources.md)；不要把九个专题文件全部塞进一次 Agent 上下文。**Topic ID 是文档检索标识，不是新增的 Tally Wire Schema/Task Status。** 跨模块列仅表示引用知识及交接位置，不改变原有工程权威。

| Topic ID | 当前问题/触发 | 最小必要输入 | 知识交接 | 必须拒绝的错误 |
| --- | --- | --- | --- | --- |
| `K10.FIRST_RUN` | 交互应用与 Headless 启动配置如何区分？ | 应用形态/业务授权/错误输出 | K10→K03/K08 | 不能所有 CLI 都弹设置 UI |
| [`K10.ACTIONABLE_ERROR`](topics/ACTIONABLE_ERROR.md) | 用户与维护者分别需要哪些错误信息？ | 失败类型/相关 ID/下一步 | K10→K03 | 日志不能泄漏密钥 |
| [`K10.UNKNOWN_EFFECT`](topics/UNKNOWN_EFFECT.md) | 超时却可能提交的业务操作如何恢复？ | 提交点/幂等键/状态回读 | K10→K06 | 超时≠未执行 |
| `K10.INCIDENT_RECOVERY` | 恢复事件如何保留故障历史和验证？ | Incident 事实/恢复/验证 | K10→K06 | Mitigation≠Verification |
| [`K10.PERFORMANCE_BASELINE`](topics/PERFORMANCE_BASELINE.md) | 没有同条件测量怎么判断优化有效？ | 负载/平台/缓存/指标 | K10→K11 | 无基线不能报 3 倍优化 |

**选用顺序：** 真实任务/版本/角色 → Topic → 相关 DP/Skill/正反例 → 当前来源/适用条件 → 原有 Validation/Task/Host 资格。文档只提供分析方法和停止条件。模块内容唯一维护在现有专题文件，Topic 行不复制其全文。

## 已深化的问题级专题

当前优先深化 3 个对真实工程结果影响较大的专题：[`K10.ACTIONABLE_ERROR`](topics/ACTIONABLE_ERROR.md)、[`K10.UNKNOWN_EFFECT`](topics/UNKNOWN_EFFECT.md)、[`K10.PERFORMANCE_BASELINE`](topics/PERFORMANCE_BASELINE.md)。每篇只解释当前问题的决策机制、反例和来源，其余方法继续直接使用已有九文件知识库；当前文档 Topic 不代表已激活生产知识。

## 目录和何时使用

| 知识资源 | 为什么要读 |
| --- | --- |
| [systematic-syllabus.md](systematic-syllabus.md) | 本模块系统专业学科：五个知识群、15 个概念/机制/反例 |
| [disciplinary-foundations.md](disciplinary-foundations.md) | 学科理论、可应用的工程方法与误推断 |
| [external-practices.md](external-practices.md) | 企业/GitHub 真实工程参考与可证伪案例 |
| [domain-model.md](domain-model.md) | 区分工程概念、事实来源及容易混淆的语义 |
| [engineering-methods.md](engineering-methods.md) | 根据具体条件选择方法、方案与有界 Decision Procedure |
| [agent-skills.md](agent-skills.md) | 供 Tally 选择的 Agent Skill、单次可复制提示词和拒绝条件 |
| [casebook.md](casebook.md) | 正反案例、变体、反模式和实际工程决策案例 |
| [validation.md](validation.md) | 负例、知识适用性训练、未来真实消费者效用验收 |
| [sources.md](sources.md) | ADS 具体条款、有效版本、候选/规范和权限边界 |

## 三个优先问题

1. 用户和维护者能否判断失败发生了什么、下一步怎么做？
2. 超时与重试是否可能产生重复业务副作用？
3. 有无同条件性能基线支持实际优化结论？

## 快速决策索引

| 触发 | 优先处理 | 期待输出 |
| --- | --- | --- |
| 交互桌面工具尚未设置仓库 | 允许启动配置 UI，禁止未授权业务操作 | 首次启动正负场景 |
| CLI/后台服务配置错误 | 按无头形态输出明确的配置诊断 | 退出码/日志或管理 API |
| 超时后状态未知 | 先查询结果并选择有界安全重试 | UNKNOWN/幂等/恢复合同 |
| 没有固定负载基线 | 先定义测量方法；否则 NOT_MEASURED | 性能测试计划 |

**选用规则**：不要求每次调用本模块的全部九份专题文档；依据当前任务所需决策选择最小知识切片。K09–K11 是条件性消费，K07/K08 在首批真实工程协作/交付需要时才选用。Tally 既有 Engineering Business/Host 事实源继续有效；本知识库为 `PROPOSED_NOT_CONNECTED`，不是新的知识服务/数据库或独立执行权限。