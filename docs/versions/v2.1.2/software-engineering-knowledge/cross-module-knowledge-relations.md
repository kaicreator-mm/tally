# 跨 K01–K11 的知识概念与关系网

> [系统知识架构](systematic-knowledge-architecture.md) · [11 模块课程](knowledge-coverage-and-growth.md) · [Tally/Harness 消费](domain-harness-consumption.md)

**范围：RELATIONSHIP_DOCUMENT_CANDIDATE / NOT_SOURCE_CLAIM_QUALIFIED / NOT_CONNECTED**。以下全部概念 ID 均对应各模块 systematic-syllabus.md 中的 F 系列知识点。关系是专业语义关联，不是新 Workflow、GitHub Native Dependency 或已执行的推断引擎。

## 一、关系的意义

跨模块的专业问题不能在单一 Topic 中完全解释。例如 K03 的幂等实现需要 K01 的业务规则、K02 的状态所有权、K05 的 Oracle 与 K10 的失败恢复语义；各模块保持单一事实/概念定义，其他模块通过关系引用。IS_A/REFINED_BY 表达分类、REQUIRES 表达推理前提、TESTED_BY/FALSIFIED_BY 表达可能的可观察证据、TRADEOFF_WITH 表达条件性备选；任何关系都不自动创建受权 Task/Host/Release 权威。

## 二、知识概念关系表

| 源知识点 | 关系 | 目标知识点 | 真实专业含义 |
| --- | --- | --- | --- |
| K01.F1.1 | CONTEXT_FOR | K02.F4.1 | 先确定真实用户 Need 和质量目标再选架构；技术偏好不是业务要求 |
| K01.F1.2 | CONTEXT_FOR | K02.F1.2 | 涉众角色/关切决定哪个视图和允许行为需要分析 |
| K01.F2.2 | REQUIRES | K07.F2.2 | 事实/假设/UNKNOWN 与 Agent 的 SourceSHA/Attempt 溯源分离 |
| K01.F3.1 | REFINED_BY | K02.F3.1 | 业务聚合与数据状态所有权相互约束 |
| K01.F3.2 | REFINED_BY | K03.F3.1 | 业务状态机约束并发条件写与实现状态变迁 |
| K01.F3.3 | TESTED_BY | K05.F1.2 | 业务不变量是 Test Oracle 的语义来源 |
| K01.F4.2 | SUPPORTS | K05.F1.3 | 需求追溯支撑 V&V 的合同/用户目标分层 |
| K01.F4.3 | TRADEOFF_WITH | K11.F1.2 | 质量承诺需考虑长期总成本，不只初次编程 |
| K01.F5.1 | EXCLUDES_WHEN | K03.F3.1 | 同条件业务政策冲突未决时禁止默默实施新状态迁移 |
| K02.F2.1 | REFINED_BY | K03.F2.1 | 系统模块合同约束组件内部编码边界 |
| K02.F2.2 | TESTED_BY | K05.F3.1 | 公开接口必须由实际消费者/合同验证，而不只类型检查 |
| K02.F3.1 | REQUIRES | K10.F3.2 | 确认提交点与状态 Owner 才能设计安全重试 |
| K02.F3.2 | REFINED_BY | K03.F3.2 | 一致性语义影响重复、取消和提交后的未知结果 |
| K02.F3.3 | TESTED_BY | K09.F3.2 | 真实数据库迁移需要旧数据和版本兼容案例 |
| K02.F4.2 | CONTEXT_FOR | K04.F4.1 | 可证伪架构 Demo 只用于会真正改变选型的未知 |
| K02.F5.1 | REFINED_BY | K09.F3.1 | 架构版本兼容落地为旧消费者行为保护 |
| K03.F1.1 | TESTED_BY | K10.F4.3 | Big-O 与实测延迟不等同，需同条件基线 |
| K03.F1.2 | TRADEOFF_WITH | K10.F4.1 | 数据结构选择与真实工作负载、内存/IO 约束共同决定 |
| K03.F3.1 | TESTED_BY | K05.F2.2 | 并发状态更新通过正反状态测试验证 |
| K03.F3.2 | FALSIFIED_BY | K10.F3.1 | 重复请求和 ACK 丢失能暴露错误幂等假设 |
| K03.F4.2 | OBSERVED_IN | K06.F3.2 | 资源关闭和远端副作用 UNKNOWN 应分别取证 |
| K03.F5.2 | EXPLAINS | K10.F2.1 | K10 拥有一般日志政策，K03 负责当前任务真实实现 |
| K04.F2.1 | CONTEXT_FOR | K07.F1.1 | 真正可独立交付的 Task 才适合多 Agent 分工 |
| K04.F3.1 | CONTEXT_FOR | K07.F4.2 | 真实前置关系与稳定合同决定合理并行 |
| K04.F3.2 | CONTEXT_FOR | K07.F4.1 | 独立工作树仍可能共享 Schema/生成物写集 |
| K04.F3.3 | TRADEOFF_WITH | K11.F2.2 | CI/审查/集成等待决定多 Agent 是否更快 |
| K05.F1.2 | REQUIRES | K01.F3.3 | 没有有权 Rule 的业务预期不能作为正式 Oracle |
| K05.F3.1 | CONTEXT_FOR | K08.F3.2 | K05 负责测试目标充分性，K08 负责真实执行来源 |
| K05.F4.1 | TESTED_BY | K08.F3.3 | Runner exit 0 与零材料测试不能合成目标 PASS |
| K05.F4.2 | REQUIRES | K09.F2.1 | 旧 Golden 是否仍有效需参照必须保留的业务行为 |
| K06.F1.2 | OBSERVED_IN | K08.F3.3 | 区分 Build 首个错误与下游 NOT_RUN |
| K06.F2.2 | CONTEXT_FOR | K10.F2.2 | 不同故障假设需通过可信日志/链路/状态做安全区分 |
| K06.F3.2 | REQUIRES | K10.F3.2 | 副作用未知时先回读，不自动重发不可逆操作 |
| K07.F2.2 | OBSERVED_IN | K08.F1.1 | 旧 Agent SourceSHA/Attempt 不覆盖当前 PR/制品事实 |
| K07.F3.1 | EXCLUDES_WHEN | K04.F2.3 | 通信 ACK/工具返回不是合法 Claim/任务最终状态 |
| K07.F4.1 | TRADEOFF_WITH | K11.F5.2 | 共享写集冲突成本应计入并行收益 |
| K08.F2.3 | TESTED_BY | K05.F4.1 | 缓存命中须对应当前源码依赖与真实受影响测试 |
| K08.F4.1 | TESTED_BY | K05.F3.1 | 只有实际选中公开入口才能证明集成有效 |
| K08.F5.1 | EXCLUDES_WHEN | K09.F3.1 | K09 仅提供兼容建议，不能成为 Release Owner |
| K09.F3.2 | TESTED_BY | K05.F3.2 | 新空库 PASS 不证明旧数据迁移/恢复 |
| K10.F4.2 | TRADEOFF_WITH | K11.F1.2 | 性能微基准收益需与真实用户价值和维护成本比较 |
| K11.F4.3 | TESTED_BY | K05.F1.3 | 知识辅助 Agent 的实际改进需独立业务 Oracle 而非问答评分 |

## 三、从关系到有界专业推理

**案例 A：业务政策冲突**。K01.F1.1 确定 Need，K01.F5.1 依据角色/状态/版本判断有权 Rule 是否相斥。若关键政策没有来源，不能直接派生 K03.F3.1 实现；ASK/UNKNOWN 并确认业务决议。确认后，K05.F1.2 才能生成真正正反 Oracle。推理能提出任务建议，不能替代 Product Freeze。

**案例 B：性能算法选择**。K03.F1.1/F1.2 解释算法与结构代价，K10.F4.1/F4.3 明确真实指标、对照负载和测量误差，K05 保护已批准功能行为，K11.F1.2 评估开发与维护成本。没有实际 baseline 只能提交测量计划，不能宣称已实测提速。

**案例 C：测试与多 Agent 结果有效性**。K08.F3.3 保留当前 clone/install/build/test 的真实阶段，K05.F4.1 检查 Runner 实际收集测试，K06.F1.2 判断首个失败，K07.F2.2 对齐 Message/SourceSHA。旧 Agent ACK、Test 0 discovered 或仅 Mock 通过均不能使当前 HEAD Gate 自动 PASS。

## 四、如何维护关系而不制造新平台

- 原始概念定义唯一保存在相应模块的 systematic-syllabus.md；本文件只列依赖、备选和可证伪联系，不复制学科教程。
- Source/Claim 需要版本、出版方、适用性和合法权限，后续选用前独立核验；关系出现不代表已有 Source Claim 资格。
- 新知识和真实负例可继续扩充本表，并局部更新真正受影响的概念/Topic；不对无关文件全量重审。
- DomainHarness 的 selected Element/Knowledge/Rule/Reader 只有在未来实际获准并实现后，才可能把这些文档关系选成机器可执行内容。当前关系仅是模型与课程维护资产。
