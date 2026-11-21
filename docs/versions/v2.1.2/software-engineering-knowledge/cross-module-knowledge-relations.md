# 跨 K01–K11 的知识概念与关系网

> [系统知识架构](systematic-knowledge-architecture.md) · [11 模块课程](knowledge-coverage-and-growth.md) · [Tally/Harness 消费](domain-harness-consumption.md)

**范围：RELATIONSHIP_DOCUMENT_CANDIDATE / NOT_SOURCE_CLAIM_QUALIFIED / NOT_CONNECTED**。以下全部概念 ID 均对应各模块 systematic-syllabus.md 中的 F 系列知识点。关系是专业语义关联，不是新 Workflow、GitHub Native Dependency 或已执行的推断引擎。

## 一、关系的意义

跨模块的专业问题不能在单一 Topic 中完全解释。例如 K03 的幂等实现需要 K01 的业务规则、K02 的状态所有权、K05 的 Oracle 与 K10 的失败恢复语义；各模块保持单一事实/概念定义，其他模块通过关系引用。IS_A/REFINED_BY 表达分类与细化，REQUIRES 仅用于确有必要的推理前提，CONTEXT_FOR/SUPPORTS 表达有条件的专业背景，TRADEOFF_WITH 表达需比较的方案。**本表的两端均为课程 Concept ID，并无 Case/Oracle/执行证据节点，因此不以 TESTED_BY、FALSIFIED_BY、EXCLUDES_WHEN 断言测试已实现或条件已判真**。未来如引入明确 Case/Oracle/条件节点，才可按真实适用语义建立这些边；任何关系都不自动创建受权 Task/Host/Release 权威。

## 二、知识概念关系表

| 源知识点 | 关系 | 目标知识点 | 真实专业含义 |
| --- | --- | --- | --- |
| K01.F1.1 | CONTEXT_FOR | K02.F4.1 | 先确定真实用户 Need 和质量目标再选架构；技术偏好不是业务要求 |
| K01.F1.2 | CONTEXT_FOR | K02.F1.2 | 涉众角色/关切决定哪个视图和允许行为需要分析 |
| K01.F2.2 | CONTEXT_FOR | K07.F2.2 | 事实/假设/UNKNOWN 的来源判断与 Agent SourceSHA/Attempt 溯源互补；并非需求事实成立的必要条件 |
| K01.F3.1 | REFINED_BY | K02.F3.1 | 业务聚合与数据状态所有权相互约束 |
| K01.F3.2 | REFINED_BY | K03.F3.1 | 业务状态机约束并发条件写与实现状态变迁 |
| K01.F3.3 | SUPPORTS | K05.F1.2 | **已确认且适用**的业务不变量可作为 Oracle 的一种来源；获权行为/合同也可产生 Oracle |
| K01.F4.2 | SUPPORTS | K05.F1.3 | 需求追溯支撑 V&V 的合同/用户目标分层 |
| K01.F4.3 | TRADEOFF_WITH | K11.F1.2 | 质量承诺需考虑长期总成本，不只初次编程 |
| K01.F5.1 | CONTEXT_FOR | K03.F3.1 | 同条件政策冲突未决时，受影响的状态迁移实现要保持 HOLD；不是全部并发操作的通用排除 |
| K02.F2.1 | REFINED_BY | K03.F2.1 | 系统模块合同约束组件内部编码边界 |
| K02.F2.2 | CONTEXT_FOR | K05.F3.1 | 公开接口合同需要按所选消费者和验证层设计实际测试，概念链接不是已执行测试 |
| K02.F3.1 | CONTEXT_FOR | K10.F3.2 | 状态 Owner 与提交点是评估 **带副作用重试** 安全性的输入，不要求所有状态模型必须采用重试 |
| K02.F3.2 | REFINED_BY | K03.F3.2 | 一致性语义影响重复、取消和提交后的未知结果 |
| K02.F3.3 | CONTEXT_FOR | K09.F3.2 | 一致性/状态边界是规划旧数据迁移案例的输入；迁移机制概念本身不是 Test |
| K02.F4.2 | CONTEXT_FOR | K04.F4.1 | 可证伪架构 Demo 只用于会真正改变选型的未知 |
| K02.F5.1 | REFINED_BY | K09.F3.1 | 架构版本兼容落地为旧消费者行为保护 |
| K03.F1.1 | CONTEXT_FOR | K10.F4.3 | 算法复杂度推导指导性能基线设计，但没有真实负载数据便不能声称已测延迟 |
| K03.F1.2 | TRADEOFF_WITH | K10.F4.1 | 数据结构选择与真实工作负载、内存/IO 约束共同决定 |
| K03.F3.1 | CONTEXT_FOR | K05.F2.2 | 并发条件写的风险应指导状态转换正反选例；课程概念并非测试收据 |
| K03.F3.2 | CONTEXT_FOR | K10.F3.1 | 幂等协议设计需考虑重复请求/ACK 丢失和未知副作用，重试理论概念本身不是反例 |
| K03.F4.2 | CONTEXT_FOR | K06.F3.2 | 资源关闭和远端副作用 UNKNOWN 应分别取证 |
| K03.F5.2 | EXPLAINS | K10.F2.1 | K10 拥有一般日志政策，K03 负责当前任务真实实现 |
| K04.F2.1 | CONTEXT_FOR | K07.F1.1 | 真正可独立交付的 Task 才适合多 Agent 分工 |
| K04.F3.1 | CONTEXT_FOR | K07.F4.2 | 真实前置关系与稳定合同决定合理并行 |
| K04.F3.2 | CONTEXT_FOR | K07.F4.1 | 独立工作树仍可能共享 Schema/生成物写集 |
| K04.F3.3 | TRADEOFF_WITH | K11.F2.2 | CI/审查/集成等待决定多 Agent 是否更快 |
| K05.F1.2 | CONTEXT_FOR | K01.F3.3 | 当存在已接受的相关业务不变量时可以据此生成 Oracle；其他已批准业务行为也可作来源 |
| K05.F3.1 | CONTEXT_FOR | K08.F3.2 | K05 负责测试目标充分性，K08 负责真实执行来源 |
| K05.F4.1 | CONTEXT_FOR | K08.F3.3 | 测试发现结果的判定要结合 Runner/阶段执行证据；阶段概念本身不验证测试发现 |
| K05.F4.2 | CONTEXT_FOR | K09.F2.1 | Golden 变更时须对照适用的兼容行为与有权规格，并非任何快照都要求版本演进 |
| K06.F1.2 | CONTEXT_FOR | K08.F3.3 | 区分 Build 首个错误与下游 NOT_RUN |
| K06.F2.2 | CONTEXT_FOR | K10.F2.2 | 不同故障假设需通过可信日志/链路/状态做安全区分 |
| K06.F3.2 | CONTEXT_FOR | K10.F3.2 | 在存在可能不可逆副作用时，未知结果须先合法读回再决定重试；非所有诊断都要重试 |
| K07.F2.2 | CONTEXT_FOR | K08.F1.1 | 旧 Agent SourceSHA/Attempt 不覆盖当前 PR/制品事实 |
| K07.F3.1 | CONTEXT_FOR | K04.F2.3 | 消息 ACK 不可替代合法 Task 完成判定；Task 概念本身不是排除条件 |
| K07.F4.1 | TRADEOFF_WITH | K11.F5.2 | 共享写集冲突成本应计入并行收益 |
| K08.F2.3 | CONTEXT_FOR | K05.F4.1 | 缓存复用判定应结合实际测试发现、源码与依赖身份；非测试已执行证明 |
| K08.F4.1 | CONTEXT_FOR | K05.F3.1 | 制品集成应选择实际公开入口与消费者路径测试；测试层概念不是观察结果 |
| K08.F5.1 | CONTEXT_FOR | K09.F3.1 | 交付决策可咨询旧消费者兼容策略，但 Release Owner 的权限不由 K09 拥有 |
| K09.F3.2 | CONTEXT_FOR | K05.F3.2 | 迁移验证需要真实旧数据/隔离/恢复设计，环境隔离概念本身并非 Test |
| K10.F4.2 | TRADEOFF_WITH | K11.F1.2 | 性能微基准收益需与真实用户价值和维护成本比较 |
| K11.F4.3 | CONTEXT_FOR | K05.F1.3 | 比较实验需有独立业务 Oracle 与真实 V&V，不得由课程连接宣称实测改进 |

## R4 新增六条跨模块关系（概念语义，不是可执行观察）

以下边均连接原有课程 Concept ID。R4 深化了六项原先只具备课程解释的知识点，因此补充其与相邻学科的**有条件交接**。使用 `CONTEXT_FOR` 表示专业语境与输入，`SUPPORTS` 表示可提供论证依据；**不是**真实 Case、测试或权限的 Observation。

| 源知识点 | 关系 | 目标知识点 | 真实专业含义 |
| --- | --- | --- | --- |
| K02.F4.3 | CONTEXT_FOR | K05.F3.2 | 设计期识别的信任/租户边界可指导需要哪些环境隔离反例；仅有概念连接不等于实际测试运行 |
| K04.F2.3 | CONTEXT_FOR | K06.F4.3 | Task 的合法完工/Review 条件需要匹配补丁 Exact SHA 和材料性复核，不能由 Worker DONE 自我代签 |
| K05.F3.3 | CONTEXT_FOR | K08.F4.2 | Build/Package/Install 的验收目的指导哪种 Artifact/目标平台真实证据应收集；不能用 Node 单测冒称安装包已存在 |
| K05.F4.2 | CONTEXT_FOR | K06.F4.3 | Golden 变更与其有权 Expected、源版本影响补丁当前性判断；更新快照并不证明修复正确 |
| K06.F4.3 | CONTEXT_FOR | K07.F2.2 | 补丁当前性需要解释旧 Attempt/SourceSHA 消息如何与当前 Review HEAD 对齐；不是任何旧消息都必须被删除 |
| K10.F5.1 | CONTEXT_FOR | K08.F4.3 | GUI 首次运行后配置的业务状态与 headless 缺参处理会改变真实启动/权限验收，而不强迫所有应用使用相同配置 UI |

**本轮新增六边尚待 R4 独立审查**，保留上方 42 条 R1–R3 审定文档关系原文；全部链接仅为概念输入/推理条件，权限和实测仍留给当前有权系统。

## 三、从关系到有界专业推理

**案例 A：业务政策冲突**。K01.F1.1 确定 Need，K01.F5.1 依据角色/状态/版本判断有权 Rule 是否相斥。若关键政策没有来源，不能直接派生 K03.F3.1 实现；ASK/UNKNOWN 并确认业务决议。确认后，K05.F1.2 才能生成真正正反 Oracle。推理能提出任务建议，不能替代 Product Freeze。

**案例 B：性能算法选择**。K03.F1.1/F1.2 解释算法与结构代价，K10.F4.1/F4.3 明确真实指标、对照负载和测量误差，K05 保护已批准功能行为，K11.F1.2 评估开发与维护成本。没有实际 baseline 只能提交测量计划，不能宣称已实测提速。

**案例 C：测试与多 Agent 结果有效性**。K08.F3.3 保留当前 clone/install/build/test 的真实阶段，K05.F4.1 检查 Runner 实际收集测试，K06.F1.2 判断首个失败，K07.F2.2 对齐 Message/SourceSHA。旧 Agent ACK、Test 0 discovered 或仅 Mock 通过均不能使当前 HEAD Gate 自动 PASS。

## 四、如何维护关系而不制造新平台

- 原始概念定义唯一保存在相应模块的 systematic-syllabus.md；本文件只列依赖、备选和可证伪联系，不复制学科教程。
- Source/Claim 需要版本、出版方、适用性和合法权限，后续选用前独立核验；关系出现不代表已有 Source Claim 资格。
- 新知识和真实负例可继续扩充本表，并局部更新真正受影响的概念/Topic；不对无关文件全量重审。
- DomainHarness 的 selected Element/Knowledge/Rule/Reader 只有在未来实际获准并实现后，才可能把这些文档关系选成机器可执行内容。当前关系仅是模型与课程维护资产。
