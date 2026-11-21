# 软件工程系统知识覆盖与滚动完善计划（11 个模块，不只是精选 Topic）
> [知识库总索引](README.md) · [语义架构](systematic-knowledge-architecture.md) · [SWEBOK 18 KA 交叉映射](disciplinary-coverage.md) · [Harness 消费](domain-harness-consumption.md)
>
> **目标**：把 K01–K11 形成可长期研究、复用、解释和验证的**系统性软件工程知识库**。以下是**学科知识框架和工作分层**，不是已经阅读全部标准正文、没有知识空白、已经完成生产 Package 的认证。现有 56 路由/26 Topic 是首批可用入口，不限制 BOK 的理论深度与未来主题数。

## 1. 完整性与可用性的两个维度必须同时衡量

| 维度 | 判断依据 | 当前已有 | 尚需继续 |
| --- | --- | --- | --- |
| **学科广度** | SWEBOK 18 KA 对照 + 关联学科；每个模块拥有系统课程和必要核心概念 | [18 KA 对照](disciplinary-coverage.md)、K01–K11 文件 | 分层子主题、基础技术和理论关联扩充 |
| **概念一致性** | 概念/对象/不变量拥有唯一解释与清楚跨模块引用 | 模块边界/基础 Domain Model | 关键术语歧义消除、概念关系正反例 |
| **技术深度** | 原理、因果、可证伪条件、替代策略、实现手段 | 26 个深入 Topic/33 DP 候选 + 22 篇两卷学科正文与 6 篇 R4 聚焦正文（旧 SHA 已独立限定接受）+ 4 篇 R5 进阶推理正文与 5 条完整演算（新增待审） | 继续遍历基础学科群，不只针对这 26 个问题 |
| **权威质量** | 版本、Clause/Source、版权、规范/建议/研究适用性与反例 | 61 外部资料目录（A55–A61 为 R4 新增七项专业发现线索）、4 条源条款预核查候选 | 按实际重要性逐条资格验收；不将目录总数等同 Claim |
| **服务正确性** | Goal/事实→知识选择→合法方法→结果来源→缺口/STOP | 逻辑设计及合成反例 | 未来 #286 真正 Agent/Harness 用户路径、对照评估 |
| **可维护性** | 新知识能快速归属主题、定位冲突和旧版本失效 | 11 Owner 模块、56 路由、Git 版本化 | 增量语义变更/失效影响与持续知识回流 |

**R2/R3 两卷系统深化**：R2 的 [11 篇第一卷学科正文](systematic-foundation-depth-map.md) 已由独立 Reviewer 在旧 Exact HEAD 接受文档范围；R3 新增 11 篇第二卷专业正文，涵盖需求样本/质量场景、架构依赖/状态、类型/背压、估算、组合/模型测试、分布式因果、人因协作、可复现构建、技术债、排队/尾延迟、实证统计；**R3 第二卷独立审核已限定接受，旧 SHA 证据可复用**。R4 为六项原先仅 S 的课程点各新增一篇聚焦文档，详见 [165 点逐项课程—深文章节覆盖台账](foundation-concept-coverage-ledger.md)：165/165 已有课程解释，其中 159/165 有两卷主题关联，另 6/165 有 R4 一对一 F 聚焦正文（R4 已限定审核接受）。**全部有深入文档入口不等于 165 条来源资格，也不等于每点独立教材完成。** 两卷共 22 篇，外加 6 篇已审补缺与 4 篇新进阶理论、5 条新完整演算；R5 进阶章节额外映射 25 个原课程 ID，但不得因此宣称 165 个概念已获得同等深度。不替代现有 26 个问题 Topic。

**R5 的深度评价转向**：从“每模块有两篇正文”进一步要求对已选概念给出可逐步核查的**输入事实→假设→机制→失败轨迹→独立 Oracle→适用边界→UNKNOWN**。见 [四篇进阶专业演算与五条端到端 W 案例](systematic-foundation-depth-map.md#r5四篇深层专业推理与五条完整演算待增量审核) 和 [五条完整版](professional-reasoning-worked-traces.md)。R5 为 25 个原知识 ID 增加 A 深层章节路由，但其他 140 个仍需逐学科持续研究，不将一项 Knowledge Link 充当原始已合格标准条款。

## 2. 三个深度级别不是三套独立产物

- **学科底座（Systematic Foundation）**：各 Kxx 的 [systematic-syllabus.md] 解释完整知识主干、每个子域的核心概念、机制、选型、反例及相邻学科；**不要求每个子主题立刻变成单独文件**。
- **高价值深化（Problem/Method Knowledge）**：现有 Topic/DecisionProcedure/Skill/Case，是基础理论的按问题可执行投影。优先深化严重影响正确性、容易 Agent 猜测、跨模块复用频繁的场景。
- **生产选用（Selected Qualified Asset）**：具有来源资格、真实应用需求、已获授权的消费者以及可观察负例验证的最小闭包。它是学科库的一个可更新子集，**不会反过来删掉暂未被选中的基础知识**。

从原理到真实使用的例子：K03 学科底座解释并发的线性化/竞争、资源生命周期、复杂度；K03.ASYNC_IDEMPOTENCY 教 Agent 判断重试；仅当某合法 Task 真的需要选定幂等 Rule/Skill 时，才做当前业务 Source/Claim/Host 资格。

## 3. 每个 Kxx 的系统性学科骨架和优先边界

| Module | 系统学科主干（详见各模块 systematic-syllabus） | 关联 SWEBOK / 相关学科 | 当前深化范围的局限 |
| --- | --- | --- | --- |
| [K01](K01/systematic-syllabus.md) | 需求获取/分析/规格/管理、业务过程、规则、状态与验收、涉众/质量需求 | Requirements、Business Analysis、Models | 不只处理 CANCEL 与验收 |
| [K02](K02/systematic-syllabus.md) | 架构原理、质量场景、模块/数据/接口、分布式语义、评估/演进 | Architecture、Design、Systems Engineering | 不只排队系统权衡 |
| [K03](K03/systematic-syllabus.md) | 算法/数据结构、编码范式、类型、状态/错误、并发/存储、可维护实现 | Construction、Design、Computing/Math | 不只幂等和 Big-O |
| [K04](K04/systematic-syllabus.md) | 生命周期/迭代、任务建模、依赖图/关键路径、并行协作、计划反馈 | Process、Management、Economics | 不只 BLOCKED_BY |
| [K05](K05/systematic-syllabus.md) | 测试设计理论、Oracle/测试层、属性/状态/模型测试、覆盖/变异、可靠验证 | Testing、Quality、Mathematical foundations | 不只 CI False PASS |
| [K06](K06/systematic-syllabus.md) | 诊断科学、可观测事实、复现/归因、定位/修复、回归、事故学习 | Maintenance、Operations、Engineering foundations | 不只首个报错 |
| [K07](K07/systematic-syllabus.md) | 角色/协作、组织协调、人机协议、接口/消息语义、Context/冲突/审核 | Professional Practice、Management、Human Factors | 不只 ACK/Reviewer |
| [K08](K08/systematic-syllabus.md) | 配置/版本、依赖/构建、测试/CI、制品/部署、集成/交付、真实环境 | Configuration Management、Operations、Process | 不只 clone 成本 |
| [K09](K09/systematic-syllabus.md) | 生命周期维护、修改/兼容、迁移/重构、版本停用、技术债、系统演进 | Maintenance、Configuration、Lifecycle | 不只 API/DB 迁移 |
| [K10](K10/systematic-syllabus.md) | 可靠性、配置、错误/日志、弹性/容量、性能分析、故障恢复与运营 | Quality、Operations、SRE/Systems | 不只 timeout/logging |
| [K11](K11/systematic-syllabus.md) | 工程价值/成本、估算/不确定性、风险、度量/实证、经济取舍/优化反馈 | Economics、Management、Measurement/Statistics | 不只 Agent Token 成本 |

**基础支柱必须横向复用**：计算机基础（数据结构、算法、操作系统、网络、数据库）主要归 K03/K02/K08；数学/逻辑/统计归 K02/K05/K11；系统工程/人因归 K01/K02/K07/K09/K10。不能只做管理流程，也不能只做编程模式；两者需要通过共同模型和来源关联。

## 3.1 关联知识能支持跨专业推理

目前已建立 [48 条跨模块概念关系](cross-module-knowledge-relations.md)，连接需求规则、架构合同、任务实现、测试、维护与运维等；并有 [18 组领域智能查询教学输入](domain-intelligence-use-cases.md)，明确哪种输入必须 ASK/UNKNOWN/STOP。两者是**系统知识本体和可证伪消费设计**，不是已部署 Harness Agent 服务或可执行规则。

## 4. 知识空白如何真实处理

**Coverage 状态**使用普通知识维护语言，不是新增 Validation Gate：
- `INVENTORIED`：只有题目/概念/关联，明确未深写或未核原文。
- `EXPLAINED_CANDIDATE`：已有原创技术原理、适用条件、反例/替代方案。
- `METHOD_CASE_READY`：能就指定场景给具体 DP/Skill/Oracle（现有 26 Topic 多在该文档层级，未因此生产合格）。
- `SOURCE_CLAIM_CHECKED`：条款与适用性经过真实审查，不假设自动获得消费者资格。
- `CONSUMER_VERIFIED`：经过合法 Task/Package/Host 等真实消费者证明，仅代表明确 Scope。

**每个子主题必须能回答**：术语/对象是什么、为何重要、前置事实、不同策略如何权衡、常见错误和负例、相关课程/跨模块术语、原始来源、当前缺口。理论学科目录可以留有 `INVENTORIED` 未完善项，避免伪造专业掌握/学科证书。没用到的高成本专题可保持文献线索和最小概念解释；真实 Consumer 需要时再深化并审查。

## 5. 面向 Tally 领域智能的进度推进顺序（不再以四条候选封顶）

1. **先固定系统学科地图**：本章与各模块课程让 K01–K11 都形成学科结构、专业核心原理、跨学科关系、反模式和已有 Topic 归属，不必马上转换成机器知识库。
2. **补实际内容**：依专业系统性缺口逐模块完善，例如 K03 数据结构/并发/算法/持久化；K05 形式化 Oracle/性质测试/边界/状态测试；K02 系统架构/领域/接口一致性；K07 通信与多 Agent 互斥。**不仅是“高价值候选”**，也包括基础理论和通用方法。
3. **来源核查常态化但不制造 Gate**：每次新增核心概念记录适用出处与不能外推的范围；需要生产 Rule 或任务决定时才做材料性 Clause、版权与真实反例核验，不要求每篇人类阅读概论走与 Release 同级审查。
4. **支撑语义和推理**：完善概念关系、条件、竞争方案、UNKNOWN 传播、Problem→Topic→Method 选择，随专业知识自然扩展。
5. **限定的消费者验收**：等 #286 未来编号 Consumer Product/L2 确认，选择覆盖不同技能的少量真实开发任务，比较无知识与系统知识辅助的设计/修复/验收和成本，记录负收益。
6. **持续演化**：将真实失败、缺知识、错误推断归到原 Subject，修复内容/适用范围/负例，并按真正受影响关系传播，不做无界自动训练和不受权晋升。

**明确不做**：为了完成系统课程创建新的 18 个 GitHub Issue、要求每项知识固定 9 个文件、复制受版权保护的全部 SWEBOK/ISO 内容、建设第二个编译器/RAG/Kernel，或用自评报告宣称整个知识体系已获得生产 Qualification。
