# Tally v2.1.3 · 首个软件工程领域智能纵向切片（L1/L2 候选）

状态：PRODUCT_L1_L2_CANDIDATE / NOT_FROZEN / NO_PRODUCTION_DISPATCH。Issue #563；Tally main@3c9241d7a08a39b8a9865b7b7d221339a0f145e3。仅设计及 SDK 可行性实测，不声称本切片已运行。#286 未解除 deferred。

## 1. 实际需求与非目标

开发者在已有受权代码任务的一个真实设计决策处，需要判断是否保持内联、抽取私有共享机制、调整模块边界、提供稳定公共契约或可替换策略。领域能力应确定性排除**已经由事实证伪**的候选，提出有条件建议、材料性未知及可复核反例。它不是根据语法相似、文件数或 LLM 打分自动重构。需求、当前源码、消费者约束、验收合同依旧由原有权来源决定。选择结果不授予写权限、不认定 Review/QA/Release PASS。

继承 #563 A1–A7；不新建知识引擎、编译器、通用 Rule VM、图数据库、向量库、Agent/LLM Loop、Workflow runtime，不强制每个小改动走本能力。R20 历史证据和 v2.1.1 原图不可修改。

## 2. 知识资产与语义唯一性

[projection](abstraction-knowledge-projection.json) 是供这一切片读取的**候选索引投影**，不是新通用 schema 或第二知识图谱。现有 K01/K02/K03/K05 概念 ID 与定义仍在 v2.1.2 原文；投影只保存名称、Owner、精确文件/Blob/行区间与候选资格。16 个原 ID、3 条两端均在子集的原关系、两个原 DP 和一个 Oracle 知识入口；省略的边不是不存在。K05.ORACLE 在旧目录已有 Topic ID；此处指向相关 R18 专业深化而不重定义其整个 Topic。三条边仅用于相关语义导航，不充当事实蕴含或代码执行证明；不自动遍历补全。旧 graph 精确 807 节点/106 Claim/271 RelationshipAssertion，四个概念只建立 RELATED_CONTEXT_NOT_EQUIVALENCE 映射（实际 PROPOSED/PENDING_REVIEW），不把 CS2023 学科层级与模块粒度机械等同、不提升 Claim。K01 需求事实，K02 职责/变化边界，K03 实现机制，K05 Oracle 资格分别有原 Owner。无新增业务 Rule，rules=[]。源知识外部资格仍未取得；允许本研究选取 advisory procedure，非通用强制规则。

## 3. 实施位置与真实 SDK 合同

先在同一 Git monorepo 建隔离的 TypeScript 工程目录（建议 research/v213/engineering-decision 或经审核确认的相同职责目录），不是在旧 B package 添加另一套 ABI。首个候选 Package 为软件工程领域 Business：一个领域决策 Element 持有/读取只读知识投影，普通私有 TypeScript 函数完成事实核对、决策和 K05 证据验证。实际消费时一个新的原生 SDK-only Domain App；后续 Tally 正式迁移由 #286 的版本设计决定，不在此阶段创建完整 App/KDP/Host 或 shim。只有确实需要生命周期的资源才加 Init/Exit；无状态判断不需要资源。Package 普通多文件，由人或 Agent 编写，生成期负责正确性，SDK 不扫描 Package 静态正确性。具体选定 Package 元数据与显式依赖应遵守固定 0.11.2 规范，不复制旧 FunctionRef/BUILD/Lock/Pin 模式。无 Forge 前置。

已验收内部 SDK：DH #1612/#1718、reviewed 566cd3129a6ef51bfc40e7d4a018530cabd965a8，train merge 18384259e36ba56d937416ed5262bac1de6689ae；[SDK public README](https://github.com/kaicreator-mm/domain-harness/blob/18384259e36ba56d937416ed5262bac1de6689ae/packages/domain-harness/README.md)；[真实领域示例](https://github.com/kaicreator-mm/domain-harness/blob/18384259e36ba56d937416ed5262bac1de6689ae/examples/native-domain-app/combined/book-catalog.ts)。单一 @kaicreator/domain-harness 根入口，TS→ESM Node >=22.12；不得引入第二种运行时、SDK 私有路径或旧 ABI 兼容。现有 Tally B index.mjs 是 v08-business-abi/1 (Blob 278847a16a303788ebf7ae3254ffd286cd1c7479)，保留不改。

- `BoundedQueryElement.select(source,predicate,{maxVisited,maxReturned})` 用于已选定只读知识记录查找；业务提供 predicate、稳定源、适用性/版本。complete=false → UNKNOWN，不返回完备候选，不将缺命中作否定。对小定长投影可按实际记录长度设扫描上限，返回上限应大于全部可能匹配数；不复制 SDK 查询算法。
- `StandardWorkflowSession(limits).run(root=>root.invokeOutcome(...))` 组织一个实际有界决策调用，需要证据验证时在同一根额度内顺序调用，非 COMPLETE 停止。不通过 invoke 包装业务 Outcome，不重新创建满额度 Session 绕过根预算。
- 仅在真实需要时使用 GraphQueryElement/BoundedExpressionElement；此切片无需数据库、Vector、图遍历或表达式 DSL。领域比较用普通 TS 函数。Query/Workflow 的通用基础由 SDK 实现，候选知识与结论由 Tally 实现。

## 4. L2 输入合同（业务类型候选，不是 SDK 新接口）

一个 `AbstractionRequest` 绑定 repo、issue、sourceSha、decisionId；声明当前合同条目（稳定 contractId、来源路径/Issue locator、版本、Owner 授权状态 CONFIRMED/MISSING/CONFLICT/STALE）和事实来源。实际候选及维度可由人/LLM 从真实代码与合同提取；这一切片只消费已结构化输入，不认证 GitHub 或自动推断 AST。调用者不能凭字符串 SHA 或 CONFIRMED 自称真实授权；适用证据必须由现有有权读回/人工验收确认，缺资格则 UNKNOWN。任何后续真实集成需把这些事实绑定原权威。
事实为 YES/NO/UNKNOWN/CONFLICT + evidenceRefs，区分已证明否定与未证；每条证据包含对应 subject（候选/维度/契约/证据范围）、当前 sourceSha 与 locator。候选类别 INLINE / PRIVATE_SHARED / DOMAIN_MODULE / PUBLIC_CONTRACT / STRATEGY；同类别可有多个设计，以独立 candidateId 区分。明确候选共享的是业务行为还是纯机制，不把每个 helper 当统一业务语义。

**R2 确定性输入准入合同（本切片业务约束，不是 SDK 限额）：**
- 单次 AbstractionRequest 最多 5 个候选；每候选 9 个不同语义维度（仅下列枚举）；整个请求最多 64 个事实记录（含候选维度、适用性与全局事实记录），128 个 evidence 记录、128 次 evidenceRef 引用（重复引用也计数）。单次 Validator 请求最多 64 个合同义务、128 个 observation 记录、128 个 evidence 记录及 128 次 evidenceRef 引用。均为包含上限；不将超限截断后继续判断。缺少可选事实不构成非法类型，但不能由缺失默认 NO。
- 固定准入顺序（第一类失败即停止）：必需顶层身份/容器结构非法或 SHA 格式非法 → REJECT / INVALID_INPUT；超过上述任一业务上限 → REJECT / INPUT_LIMIT_EXCEEDED（先核对集合长度及累计引用数，不作部分业务评价）；上限内记录字段缺失或类型非法 → REJECT / INVALID_INPUT；不支持的枚举（含维度）→ REJECT / UNSUPPORTED_ENUM；重复定义的 ID 或同一候选同维度重复定义 → REJECT / DUPLICATE_ID（不同观察 ID 对同一主体报告冲突不是重复 ID）；引用悬空、已知证据主体/候选/合同归属不匹配 → REJECT / EVIDENCE_REFERENCE_INVALID。此处针对已解析业务数据，不声称本切片是任意网络 payload 的资源安全解析器。
- 引用结构正确、同主体但证据是旧版本或旧 SHA → 入口 UNKNOWN / STALE_EVIDENCE；Validator 逐项 STALE（不能用于当前观察）。有权合同明确缺失 → ASK / CONTRACT_REQUIRED；合同确认声称无资格、材料事实证据缺失或仅未核验自报 → UNKNOWN / EVIDENCE_UNQUALIFIED（候选局部则 NEEDS_EVIDENCE）；已确认同范围材料冲突 → UNKNOWN / MATERIAL_CONFLICT（Validator 逐项 CONFLICT）。入口语义检查顺序：已确认材料冲突、过期、合同缺失、证据不合格；同一优先级保留全部相应原因。证据源缺失并非输入结构错误；不把无证据默认 NO 或把冲突以最后一条覆盖。SDK 查询正常执行但 complete=false → UNKNOWN / QUERY_INCOMPLETE；SDK Workflow 调用预算耗尽 → FAILED / WORKFLOW_CALL_BUDGET，保留 SDK 原因，不重置根预算。

核心事实：
- contractKnown / materialConflict / currentEvidence 由 K01 准入；缺有权合同 ASK，冲突/过期/证据不全 UNKNOWN，输入类型非法 REJECT；不推断权限。
- 合并候选的 PRECONDITION、POSTCONDITION、INVARIANT、ERROR_SET、SIDE_EFFECT、STATE_OWNER、CLEANUP、VERSIONED_CONSUMER、CHANGE_AXIS 各自证据：`compatible: YES|NO|UNKNOWN|CONFLICT`，只针对该候选宣称保持的语义范围。纯 helper 仅核对其真实输入/输出、不变量、资源语义等；不继承调用者业务授权，调用者须分别保护差异。不可省略适用维度以伪造同语义；不适用需说明和证据（事实附带 applicability）；材料性 UNKNOWN → 保留候选待证。
- 已存在共同维护位置/同一 Owner/同一不变量/共同变化方向、实际独立变化/资源或发布边界、真实异构消费者/稳定替换需求、已获权动态扩展、公共兼容义务、已知维护和间接性成本。不能由文件数、代码长度、future-maybe、同正则推断任何一个 YES。

## 5. 有界决策与输出

`assessAbstraction(request)` 采用普通业务函数，返回 `StandardWorkflowOutcome<DecisionAssessment>`，无需所有内部 helper 统一 Outcome。
- 入口事实资格/contract 确认不完整先 ASK/UNKNOWN，不继续给无条件推荐。语义事实在候选内缺失则标该候选 NEEDS_EVIDENCE；冲突阻断受影响决策，不一律否定其他明确不相关候选。结构错误 REJECT；SDK budget 或异常 FAILED。WAIT 保留 SDK 语义，此纯计算不主动产生 WAIT。
- 按候选实际语义，已确认某材料维度不兼容且候选会强行合并为单一语义 → EXCLUDED（排除这一**提出的行为合并**，不是禁止任何共享底层函数）。候选若保持两种语义并共享可证明纯机制，另行评估。
- 私有共享的正向条件：多个实际维护位置，同 Owner、共同不变量及变化机制、相关语义相容，未破坏已有公共合同。均具证据时标 SUPPORTED，不宣布唯一最优。若只有语法相同或尚未证明共享语义，不能标 SUPPORTED。
- PUBLIC_CONTRACT：已有真实异构消费者需要稳定共同语义、独立发布/兼容边界或已证实稳定替换边界，且拟议公开契约保持所有已授权行为，才可 SUPPORTED；并非要求动态换实现。STRATEGY：必须另外证明实际且获权的多实现替换/动态变化需求，并确认这些实现满足同一稳定合同（包括错误、权限、状态及清理等适用语义）；仅多个消费者、稳定接口或独立发布不足以支持 Strategy。任一材料条件未知 → NEEDS_EVIDENCE；已经证实无相应需求且无既有兼容义务 → NOT_SUPPORTED（不是强制 Rule 拒绝）。已证实候选违背现有合同或强行合并不相容语义 → EXCLUDED。现有 API/Strategy 的兼容义务不因没有新增需求消失；保留原行为与新增扩展点是不同决策。DOMAIN_MODULE 以真正职责/变化和资源边界为依据，INLINE 以局部无需共享/独立变化为依据。
- 每候选输出 disposition（SUPPORTED / EXCLUDED / NEEDS_EVIDENCE / NOT_SUPPORTED）、具体 evidenceRefs、所用概念/DP ID 与精确来源、拒绝/保留原因、待验证项和针对实际合同的反证建议（待运行）。不为建议自动生成新的授权事实。已知无实质未决问题且原行为/契约保持可观察，返回 KEEP_EXISTING；多合理候选返回 TRADEOFF；没有证据支持任何选择返回 UNKNOWN。即使只有一个 SUPPORTED 也只能 advisory candidate，不能保证全局最优，亦不因其他未知方案被排除而伪造唯一性。建议级结果不等于通过工程合同。
- 根 Workflow COMPLETE 表示**本次有界评估完成**，不等于候选质量 PASS。决定过程按输入稳定顺序和固定理由，不依赖时间/随机或网络；source/knowledge identity 随结果附回。报告不包含隐藏模型思维过程，仅事实和可检验依据。

## 6. K05 合同证据 Validator（独立于设计推荐）

`assessContractEvidence(request, observations)` 验证冻结任务的合同义务、输入域和观察条件，不从任意调用者提交的 tests 列表推断全部合同。**合格证据与自报的信任边界：**
1. 由既有受权 GitHub/任务合同读回或人工核对，向领域调用提供已确认的 obligation universe（repo、Issue、合同来源 locator/版本、当前 candidate SHA、完整义务 ID 与每项适用范围、预期和观察条件）。需明确本次声明覆盖范围的义务集合已完整核对。调用者不得用任意自报的 obligationIds 缩小该集合；即使全部自报测试绿，也不能得出聚合 PASS。对仅部分确认的义务可记录限定观察，未确认集合整体 UNKNOWN，不允许空集合或删去遗漏项制造 PASS。
2. Oracle 资格：DECLARED 必须绑定受权条款及预期；DERIVED 必须绑定受权条款与已经核对的推导；ENGINEERING_HYPOTHESIS、未核对推导或未经授权的预期一律不能决定主 PASS/FAIL。
3. 每条观察记录绑定 obligationId、candidateId、repo/Issue、sourceSha、目标/环境、实际执行身份（run/test/probe ID）、可读回收据 locator、发现/执行事实、实际结果、Oracle 来源与观察覆盖。证据资格只分 VERIFIED_OBSERVATION（有权读回或人工实际核对原始执行证据/直接可复现观察，确认上述绑定和内容）、REPORTED_ONLY（仅自报或未核对）与 UNRESOLVED（缺失、无法核对或冲突）。VERIFIED_OBSERVATION 由**调用边界外的既有受权确认者**提供：领域函数仅消费可信入口已核对的记录，不接受任意 request/observation 字段中的自称 VERIFIED、SHA、链接、签名样字符串或 CONFIRMED 当作确认。实现将确认者提供的上下文与普通 request/observation 分开；缺该可信上下文时降为未核验，不模拟验证或发明认证服务。测试的可信上下文可以由测试明确构造用于验证领域判定，不能冒充真实任务读回。未来真实接线必须另行绑定 GitHub/Host/Reviewer 原权威。
4. 只有受权义务、合格 Oracle、相同主体/当前版本、VERIFIED_OBSERVATION 且证据实际确认目标已执行并观察到通过、覆盖足以支持该义务**本次声明的范围**，才可 OBSERVED_PASS_SCOPED。已证实适用反例同样要求受权义务/Oracle、当前主体及 VERIFIED_OBSERVATION；一个已确认反例即可 OBSERVED_FAIL，无须通过测试覆盖整个输入域。仅自报成功或失败、看似完整但未核验的收据、无法判断是否执行、局部通过而覆盖不足均 UNKNOWN；已核对目标未执行则 NOT_RUN；同主体旧 SHA/版本证据且无合格当前观察为 STALE。不同主体的引用不作该义务观察；显式错误绑定按输入准入 REJECT。相同范围下相互矛盾且无法消歧的**合格**观察 → CONFLICT，不以最后一条、通过数或自报覆盖；适用域中另一独立有效反例不因其他范围通过而消失。保留原证据资格与逐项未决原因。

逐项状态：OBSERVED_PASS_SCOPED / OBSERVED_FAIL / NOT_RUN / UNKNOWN / CONFLICT / STALE。聚合固定优先级：任何独立、未被同范围冲突阻断的合格 OBSERVED_FAIL → PUBLIC_CONTRACT_PARTIAL_OR_FAILED；否则任一 CONFLICT/UNKNOWN、义务集合未确认完整或为空 → UNKNOWN；否则任一 NOT_RUN/STALE → INCOMPLETE；否则非空、完整且已核对的义务集合中每项均为 OBSERVED_PASS_SCOPED → TESTED_CONTRACTS_PASS_SCOPED。缺失义务观察（没有已核对未执行证明）逐项 UNKNOWN，不凭空认定 NOT_RUN。记录未核验观察但不让它单独推翻已核对证据。聚合是证据评估值，不是 SDK Workflow Outcome，也不产生 Review/QA/Release 权限；永不宣称所有输入或系统质量 PASS。测试总数仅辅证，exit 0 不能证明某项目标被执行。

R20 历史回放验收：原短 Provider 191 bytes（163 header+28 body）完整 body 解码，违反已冻结 TASK L9–10。原 public probes 8/8 保持，Validator 在该证据下必须区分 tested oracles PASS 与 PUBLIC_CONTRACT_PARTIAL_OR_FAILED；metadata MISMATCH/INVALID 解释仍待 Owner，不用此歧义增加确定的权限/安全失败。不能修改历史或把已归档研究叫作生产缺陷修复。真实来源案例仅作为已有证据输入，未运行新的工程 Agent。

## 7. 执行及资格门

[acceptance-cases.json](acceptance-cases.json) 是前瞻用例意图，不是测试报告；具体可执行 tests 必须在 L2 受审并冻结后实现，尤其语义维度独立反例、遗漏证据、跨候选/跨 SHA、根预算截断和合法可替代方案不能被错误排除。先用原生 SDK 真实调用与 TS typecheck、Node 测试、必要反例验证；本地已有 R63 制品与依赖，详见 [sdk-preflight.md](sdk-preflight.md)，不要求另一个 Agent 或远程 CI 代替当前可执行验证。若取得的 artifact/依赖身份不匹配则停止，不 Mock SDK。

先冻结本 L1/L2 候选；T2 写实际领域代码与 tests，T3 完成同 SHA验证及一次综合独立审查（有问题最小修复）；T4 选择实际有权新代码决策并验证行为，视真实价值扩展 K01/K05 等，不以另一套 A/B 实验或文档数量当目标。#286 的生产迁移、完整 BOK/正式 Source Claim 资格、安装发布及其他阶段不在本 PR。
