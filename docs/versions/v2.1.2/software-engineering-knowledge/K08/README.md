# K08 · 构建、集成与交付 — 软件工程领域知识库

> **Tally v2.1.2 软件工程知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #529](https://github.com/kaicreator-mm/tally/issues/529) · [总索引](../README.md) · ADS 正式基线 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。当前只完成文档化的知识设计，尚无正式 Source→Claim 资格或真实 Package 执行。

让源码真正通过项目工具链、测试与集成入口形成可交付制品，区分各步骤的实际状态与版本。

**系统正文第一卷（R2 文档范围独立 ACCEPT_SCOPED @ ede9d646cb591b8df5179dc6bc11b60668fe968c）**：[学科深化 01：可复现构建与 CI 执行事实](foundations-deep-dive-01.md)。此章解释术语背后的机制、适用条件、工程取舍与可证伪反例；仍为文档知识候选，不是已合格 Source Claim 或真实 Harness/Agent 功能。

**系统正文第二卷（R3 文档范围独立 ACCEPT_SCOPED @ a39a64a82beae103318da3e45bd09b5efed15ad1）**：[学科深化 02：Hermetic Build 与缓存因果](foundations-deep-dive-02.md)。本卷补充基础理论、适用条件、备选技术与可证伪反例；不宣称 165 点均已教材级展开，也不自动取得生产 Source/Consumer 资格。

**R6 进阶原理与专业反例（旧 Exact SHA `83ebee092db412b5e38db0376562ee51b0d4f032` 已获独立 ACCEPT_SCOPED，文档范围）**：[Hermetic 构建输入、一次可信源码获取和平台制品](advanced-hermetic-ci-artifact-identity-and-target-host.md)。对本模块的选定课程 ID 补充可验证的机制、适用前提、替代选项和否证轨迹；相应 [W06–W12 跨模块完整演算](../professional-reasoning-worked-traces-02.md) 均为教学而非实际运行。无 Source Claim、DomainHarness ABI 或生产 Agent/Host 权限资格。

**R8 深化工程判断专业基础（旧 Exact SHA `258f5cd6ff8d34a62b99d1f81f5e5a8c0167062d` 已独立 ACCEPT_SCOPED，文档范围）**：[Artifact Provenance、正式 Release 和不可逆回滚](advanced-release-provenance-permission-and-rollback.md)，精确连接本模块 **2 个原有高优先级课程 ID**，明确 Source/范围、机制假设、负例与 STOP；[W13–W17](../professional-reasoning-worked-traces-03.md) 展示综合选择，[来源适用矩阵](../professional-source-applicability-r8.md) 仍是引用候选，不构成真实 Runtime/Agent/Host/Release 资格。

**先了解整个专业学科，再按需选择问题**： [K08 系统课程：5 个知识群 / 15 个专业知识点](systematic-syllabus.md) 用于完整概念、原理和常见反例的持续积累；下表是每次任务的有界检索入口，并非该模块的全部知识。课程内容尚未得到 Source Claim/Domain Package 生产资格。

## 当前工程问题索引（优先入口）

先根据当前 Task 的**真实问题**命中下表最小 Topic，再阅读与之相关的 [专业方法](engineering-methods.md)、[Agent Skills](agent-skills.md)、[案例](casebook.md)、[来源](sources.md)；不要把九个专题文件全部塞进一次 Agent 上下文。**Topic ID 是文档检索标识，不是新增的 Tally Wire Schema/Task Status。** 跨模块列仅表示引用知识及交接位置，不改变原有工程权威。

| Topic ID | 当前问题/触发 | 最小必要输入 | 知识交接 | 必须拒绝的错误 |
| --- | --- | --- | --- | --- |
| [`K08.PIPELINE_STAGES`](topics/PIPELINE_STAGES.md) | 依赖安装、Build、Test、打包结果如何分离？ | Runner 步骤/Exit/源码 SHA | K08→K05/K06 | Install PASS≠Build PASS |
| [`K08.CLONE_EFFICIENCY`](topics/CLONE_EFFICIENCY.md) | 如何避免多 Workflow 反复完整 Clone？ | Provider 隔离/真实网络日志 | K08→K11 | 只看 YAML 不认定节省比例 |
| `K08.SOURCE_PROVENANCE` | CI 和 Artifact 具体属于哪个 SHA/环境？ | 源/Config/Runner/Artifact digest | K08→K05 | 旧 CI PASS 不覆盖新 HEAD |
| [`K08.DELIVERY_READINESS`](topics/DELIVERY_READINESS.md) | 用户实际安装启动配置是否可用？ | 真实制品/平台/用户入口 | K08→K10 | PR merge 不等于 Release |
| `K08.RELEASE_BOUNDARY` | 哪个 Owner 允许冻结/发布制品？ | 当前授权/证据/版本合同 | K08 → **现有获授权 Release Owner / [ADS Release Standard](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/RELEASE_STANDARD.md)**；仅涉及旧版本兼容时咨询 K09 | Package Candidate≠Release |

**选用顺序：** 真实任务/版本/角色 → Topic → 相关 DP/Skill/正反例 → 当前来源/适用条件 → 原有 Validation/Task/Host 资格。文档只提供分析方法和停止条件。模块内容唯一维护在现有专题文件，Topic 行不复制其全文。

## 已深化的问题级专题

当前优先深化 3 个对真实工程结果影响较大的专题：[`K08.PIPELINE_STAGES`](topics/PIPELINE_STAGES.md)、[`K08.CLONE_EFFICIENCY`](topics/CLONE_EFFICIENCY.md)、[`K08.DELIVERY_READINESS`](topics/DELIVERY_READINESS.md)。每篇只解释当前问题的决策机制、反例和来源，其余方法继续直接使用已有九文件知识库；当前文档 Topic 不代表已激活生产知识。

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

1. 依赖安装、构建、测试、打包分别在何处执行且各自结果是什么？
2. 哪个准确源码和 Runner 配置对应本次验证收据？
3. 是否已证明真实用户能安装/启动/配置并使用目标功能？

## 快速决策索引

| 触发 | 优先处理 | 期待输出 |
| --- | --- | --- |
| npm install 成功而 tsc 失败 | 定位 tsc 首个失败类型和当前源码 | BUILD_FAIL 与根因信息 |
| 多个 workflow 重复 clone 同一 commit | 按真实 Provider 工作区/缓存机制判断能否复用 | 网络/墙钟证据与最小优化 |
| Unit PASS 但运行入口未加载修改模块 | K05 指定 public-path 检查，K08 安排真实集成 | 适用 Integration/Host 收据 |
| 发布制品没有存在/摘要/安装验证 | 将 Release Candidate 与可下载正式包分开 | artifact UNKNOWN/NOT_FOUND |

**选用规则**：不要求每次调用本模块的全部九份专题文档；依据当前任务所需决策选择最小知识切片。K09–K11 是条件性消费，K07/K08 在首批真实工程协作/交付需要时才选用。Tally 既有 Engineering Business/Host 事实源继续有效；本知识库为 `PROPOSED_NOT_CONNECTED`，不是新的知识服务/数据库或独立执行权限。