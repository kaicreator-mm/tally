# 软件工程系统知识库：领域知识架构（v2.1.2 候选）
> [知识库索引](README.md) · [18 KA 学科覆盖](disciplinary-coverage.md) · [11 模块系统课程](knowledge-coverage-and-growth.md) · [Tally/Harness 消费边界](domain-harness-consumption.md) · [来源目录](external-source-catalog.md)
>
> **性质**：Tally 软件工程 Domain App 的**系统性知识资产设计**，非已经通过验证的生产 Domain Package。ADS 正式规则、GitHub Task/Claim/Review/Release 与真实 Host 的权限均不受影响；不会据此生成第二套 Workflow Engine、Knowledge Server 或 DomainHarness 新 ABI。基线为 `ADS@933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`，外部源应按实际版本和许可核实。

## 1. 产品目标：领域知识 ≠ 高价值 Topic 列表
Tally 需要对软件工程具有**系统的专业认知基础**，而不是只在 4 个 Source Claim 上进行“提示词检索”。知识体系必须支持“完整专业概念与学科覆盖 → 可用问题与方法 → 受权任务中的适用判断 → 真实执行/验证 → 经验反馈”。这五类内容各有职责：

| 视角 | 必须包含什么 | 允许先文档建设吗 | 绝不凭文档证明 |
| --- | --- | --- | --- |
| **BOK / 学科广度** | 术语、概念、理论、子学科、关联学科、关键问题、反例、外部出处 | 可以，形成明确覆盖和空白 | 全部世界知识已吸收、专业知识已证实 |
| **Domain Intelligence / 专业理解** | 对象/关系/状态/条件、因果机制、模型与不变量、方案权衡、适用边界、冲突/UNKNOWN | 可以设计模型、示例和可证伪方法 | 已实现确定性推断或模型正确率 |
| **Knowledge Assets / 可复用资产** | Claim、Principle、Method、DecisionProcedure、Pattern、Rule、Skill、Case、Oracle、Template、Provenance | 可以编写候选并与当前权限来源绑定 | 规则已自动采纳、实际 Agent 已消费 |
| **Workflow / Agent Harness** | Intent/Goal、Task/阶段、上下文选用、受权 Function/Tool、执行/停止、反馈 | 只描述已有/候选消费者契约 | DomainHarness 0.11.2/0.11.5 已发布，Tool/Effect 已授权 |
| **Qualification / Feedback** | 来源准确性、版权与范围、反例、真实任务试验、反馈、版本与失效 | 可以规划/准备选定范围 | 未运行的评估被标为 PASS |

**覆盖目标**：沿用现有 K01–K11 作为业务能力目录；借助 [SWEBOK 18 个 KA](https://www.computer.org/education/bodies-of-knowledge/software-engineering/topics)、[SEBoK](https://sebokwiki.org/wiki/Main_Page)、[ISO/IEC/IEEE 12207:2026](https://www.iso.org/standard/90219.html)、[SFIA 9 与 SWEBOK 的能力映射](https://sfia-online.org/en/tools-and-resources/bodies-of-knowledge/swebok-software-engineering-body-of-knowledge)、IREB/ISTQB 等标注专业覆盖。学科标准是覆盖和方法的依据，不自动成为 Tally 的生命周期或任意任务强制 Gate。软件安全/供应链和代码库逆向仍不单独开展专项；已有安全责任不失效。

## 2. 组织层次和唯一语义拥有者

```text
SoftwareEngineeringDomain
├── KnowledgeArea（18 个学科轴，另有计算机/数学/系统工程交叉轴）
├── CapabilityModule（K01–K11，固定业务能力与主要 Owner）
│   ├── Subject/KnowledgeCluster（完整学科目录，系统性覆盖）
│   │   ├── Concept / Object / Property / Relation / Invariant
│   │   ├── Principle / Assumption / Applicability / Claim / Source
│   │   ├── Model / Technique / DecisionProcedure / Pattern
│   │   ├── Case / Counterexample / Oracle / Validation
│   │   └── Related module / Dependency / KnownLimit
│   └── Topic（56 个高频问题入口，26 篇已深化文章；允许继续增长）
├── Workflow / Engineering Business references（由现有 Tally 业务拥有）
├── Package selection / Element references（待合法 ABI/Consumer 决策）
└── Evidence / Feedback（真实任务/Host/独立审核；不得自签）
```

**Concept** 解释“是什么”，**Principle/Claim** 说明“为什么与在什么条件下成立”，**Procedure/Skill** 教会 Agent “在有限步骤中如何判断/操作”，**Rule** 只有来自足够权限和资格的确定性约束才可在真实系统中执行，**Workflow** 组织任务活动而不复制知识，**Case/Oracle** 解释什么结果能推翻知识的误用。Module/Subject 负责**广度**，Topic/DP 负责**适时深化**，不是彼此替代。

概念可在不同模块使用，但**定义只有一个权威所属**。例如“状态所有权”由 K02 解释，K03 说明某 Task 如何编码，K05 对行为设 Oracle，K08 说明实际测试/制品收据；不能各自生成矛盾的状态定义。**K11 经济选择和 K04 任务分解/K07 协作是横向能力**，不应被 P0/P1 开发阶段线性排列。

## 3. 知识关系：为了推理，而不是为了好看的知识图谱

| 关系 | 正确的作用 | 例子 | 绝对不能据此提升 |
| --- | --- | --- | --- |
| `IS_A / PART_OF` | 分类与组成 | Requirement → functional / quality; Task → WorkUnit | 层级存在 ≠ 有权执行 |
| `DEPENDS_ON / PREREQUISITE` | 真实前提与逻辑依赖 | 已接受 Rule 是生成 Oracle 的前提 | 文档引用 ≠ GitHub `BLOCKED_BY` |
| `APPLIES_WHEN / EXCLUDES_WHEN` | 适用范围/反例 | 幂等重试需合同、稳定请求身份 | 所有 timeout 都能重试 |
| `DERIVED_FROM / SUPPORTED_BY` | Claim 来源与出处 | 测试发现主张来自 ADS Testing §6.1 | 链接存在 ≠ Clause 已验收 |
| `CONTRADICTS / SUPERSEDES` | 明示来源冲突/版本演进 | 两个同条件 CANCEL 政策相斥 | 版本晚 ≠ 自动业务权威更高 |
| `IMPLEMENTS / INVOKES` | 方法/能力关联 | K05.ORACLE → DP-K05；Builder 调用选定 Function | Skill 文案 ≠ 工具调用许可 |
| `TESTED_BY / FALSIFIED_BY` | 正例和负例的可检验性 | 测试缺席→不得声称合同 PASS | 合成示例 ≠ 真实执行证明 |
| `ALTERNATIVE_TO / TRADEOFF_WITH` | 技术选型与质量冲突 | 内存 Queue ↔ 持久 Job | 质量未测 ≠ 自动选择某框架 |
| `CONSUMED_BY / OBSERVED_IN` | 选定消费者、应用证据 | K08 Stage 原理 → 选定真实 Task 日志 | 预备消费 ≠ Package 已激活 |
| `LEARNED_FROM / CANDIDATE_FOR` | 经验候选与未来受控进化 | CI false green 变为新负例 | 单次 Agent 自报成功 ≠ Rule 晋升 |

**支持冲突与未知的一等语义**：每个主张必须标明适用主体、时效、输入事实、权限/版本、正反预期。多个可信来源在相同适用条件下冲突时输出 `CONFLICT/UNKNOWN` 与具体待判项；缺原文或过期时 `STALE/UNKNOWN`；不适用输出解释而非强行答复。无符合条件的 Rule 时可选择合格的 Procedure/问询，只有将来实际获权且需要时才进入 LLM 有界推理。

## 4. 可维护知识单元：文档约定，不是新的 Wire Schema

为避免每篇文章都复制 9 个基础文件，保持现有文档的唯一内容拥有者。知识单元可在 Markdown 中使用如下**语义字段**；不要求当前 140 个 Markdown 全量补字段，也不声明这是 DomainHarness 0.11.x 的最终机器 ABI：

| 字段 | 最小回答的问题 | 示例 |
| --- | --- | --- |
| Identity | 哪个 Domain/Module/Subject/Topic，谁负责定义？ | `K05 / Verification / FALSE_PASS` |
| Kind | Concept、Principle、Claim、Pattern、Procedure、RuleCandidate、Skill、Case？ | `PRINCIPLE_CANDIDATE` |
| Meaning | 精确定义/因果解释与可选择方案？ | `CI success does not prove omitted suite` |
| Applicability | 输入、事实、范围、角色、平台和失效条件？ | 受影响材料合同、真正的 test runner |
| Relations | 依赖/反例/跨模块关系是什么？ | `REQUIRES:K01.ACCEPTANCE; TESTED_BY:...` |
| Provenance | 源标题、版本、完整 URL/immutable ref、节和许可？ | `ADS Testing §6.1 / candidate SHOULD` |
| Qualification | 哪些只是研究，哪些经过内容/源/真实效果验证？ | `SOURCE_CANDIDATE`、`NOT_RUN` |
| Procedure & Outcome | 被选中时产出什么，何时 STOP？ | “确认测试发现；未知则 NOT_RUN” |
| Evolution | 原始解释、较新版本替代、旧案例与失效？ | 来源发生变更时选定受影响 Claim 复核 |

**资格分开记录，不造新 Gate**：文档质量经独立审核 `ACCEPT_SCOPED` 不等于 Source Claim 接受；Source Claim 接受不等于选定 Package/Runtime 正确；真实运行无证据时使用原有效 Validation 的 `NOT_RUN/BLOCKED`，不是自造 `MOCK_SCOPED` 等 Gate。外部教材/开源仓库常只适合形成**方法候选**，不能当当前 ADS 项目的 `MUST`。

## 5. 同一个问题怎样跨知识而仍保持最小上下文

**示例：软件项目要求新增重启可恢复的批量订单导出，第一次构建失败，另有旧 Agent 迟到消息。**

1. **K01（业务语义）**：确定角色、允许导出订单范围、重复请求/失败后状态等有权规则；不清楚的事务政策标 UNKNOWN。
2. **K02（架构模型）**：根据业务恢复需求和真实部署约束选内存任务/持久任务，状态 Owner 明确；实验用进程重启推翻错误设计。
3. **K03（实现）**：在合法 Task/写集内实现幂等 ID/事务/资源清理与具体错误。
4. **K04（实施规划）**：仅为必要 Schema 和共享写集建立真实前置，避免无关串行 Task。
5. **K05（验证）**：来源 Oracle、重启/重复/非法状态的负例；识别测试 Runner 0 发现的假阳性。
6. **K06（诊断）**：若 build 失败，获取首个真实失败而非在旧错误上反复重跑。
7. **K07（协作）**：迟到 Agent 结果绑定旧 HEAD，不影响当前 Task/Review。
8. **K08（交付）**：分离 install/build/tests/制品/用户首次启动证据，只有真实执行才能报告 PASS。
9. **K09/K10（演进/运行）**：若涉及旧用户的数据迁移及运行期超时恢复则按需调用，否则不额外加载。
10. **K11（决策）**：比较真实总成本和保证下限，决定何时研究够了、何时停止重试。

Agent 不需要一次载入全部 K01–K11；通过目标、真实条件和跨模块关系选出最小知识闭包。**系统知识库的广度与每次 Agent 上下文的长度是两个不同指标。**

## 6. 版本演化与维护原则

- **学科来源层**：公开目录包含版本/出版方和技术范围；新资料可加入学科目录，但专业观点要在具体 Concept/Claim 的边界上纠错，避免“官网仍可访问”就宣称内容都有效。
- **工程实践层**：ADS 的实验成功、失败和反例有独立原始日志和相应实施环境；将经验与适用条件归属单一知识主题，跨模块引用而非复制。
- **模型层**：每次源更新只传播到有实际 `DERIVED_FROM/SUPPORTED_BY` 依赖的知识资产；不重跑 11 个模块全部考试。
- **消费者层**：Tally 未来选定 Package/Agent 时需要精确版本、Rule/Knowledge ID、真实场景/来源的最小读回；变更相应来源后检验受影响 Consumer。
- **缺口层**：详见 [knowledge-coverage-and-growth.md](knowledge-coverage-and-growth.md)，按学科重要性和实际需求补齐，不以文件数量或 Token 用量定义“知识合格”。

## 7. 本版本不声称什么

目前无已验证的 18 KA 每个细目“全部掌握”证明、无将 49 项外部来源变为 47 条已审 Claim 的过程、无 DomainHarness 0.11.2 编译执行、无真实 selected Domain Package、无模型准确率/提效实测。**系统性知识覆盖设计与专业知识生产可以同步推进，但真实 Runtime 资格仍按合法接口、Source、消费者 Task 和实际观察逐级取得。**
