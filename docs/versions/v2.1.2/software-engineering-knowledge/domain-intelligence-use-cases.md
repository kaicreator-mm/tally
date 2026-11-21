# 软件工程领域智能：有界查询、推理与反馈示例

> [系统知识架构](systematic-knowledge-architecture.md) · [跨模块关系](cross-module-knowledge-relations.md) · [Harness 边界](domain-harness-consumption.md)

**状态：LOGICAL_USE_CASES_ONLY / REAL_AGENT_NOT_RUN / DOMAIN_PACKAGE_NOT_CONNECTED**。这些是 Tally 未来软件工程 Domain Intelligence 服务应能够回答的专业问题及正反预期，不是已存在的 public SDK 方法或 CI/Host 收据。

## 1. 一次领域问题的统一选择原则

输入必须包含当前 Goal、业务/代码事实、来源/角色、版本/环境及未知。先识别完整知识库的专业模块、知识群和概念，再按实际风险选 Topic/DecisionProcedure/Rule/Skill/Case；所有条件性结论说明支持/反对来源和 STOP。确定性知识足以解决时不必启动 LLM；只有未来获权且确有材料性未知时使用受限模型生成候选，并且不能替代真实 Host/Task/Review。

## 2. 十八种跨学科查询及预期结果（Q01–Q06 早期已审、Q07–Q12 为 R3 已审、Q13–Q18 为 R4 候选）

### Q01：订单取消规则冲突

输入：用户希望已发货订单一键取消；现行政策 A 拒绝，旧客服 FAQ B 声称付款即可取消，但 B 的 Actor 与版本未知。

知识选择：[K01 系统课程](K01/systematic-syllabus.md) F2.2/F3.2/F5.1；[规则冲突](K01/topics/RULE_CONFLICT.md) 与 [验收](K01/topics/ACCEPTANCE.md)，有权决议后才到 K02/K05。

预期：对同一 Actor/State/Event/Context 比较来源。无法裁定时保持受影响行为 UNKNOWN/HOLD，而不是根据训练语料猜“合理政策”。错误答案是将任何状态都设计成可取消并宣布业务验收已通过。

### Q02：持久作业与恢复性架构

输入：用户要批量导入不阻塞 UI、重启后恢复进度；可能重复投递，负载与部署成本未知。

知识选择：[K02 架构课程](K02/systematic-syllabus.md)、[状态 Owner](K02/topics/STATE_OWNER.md)、[K03 幂等](K03/topics/ASYNC_IDEMPOTENCY.md)、K10 可靠性与 K11 经济决策。

预期：说明内存队列与持久 Job 的恢复差异、提交点、幂等要求与成本未知。提出有界重启+重复投递实验；不能仅凭无重启的 Mock Benchmark 宣称恢复已验证，或强制所有项目部署 Kafka。

### Q03：算法复杂度与真实性能

输入：有人建议“所有 Array 改为 Map，系统至少快 3 倍”；当前数组规模仅几十条、无实测基线。

知识选择：[K03 基础算法](K03/systematic-syllabus.md)、[算法选型](K03/topics/ALGORITHM_CHOICE.md)、[K10 性能基线](K10/topics/PERFORMANCE_BASELINE.md) 与 [K11 工程取舍](K11/topics/TRADEOFF.md)。

预期：比较数据量、时间/空间复杂度、维护/缓存成本，明确无基线时 NOT_MEASURED；不能把理论阶数直接宣传为整机端到端提效。

### Q04：CI 绿但材料测试未执行

输入：CI exit 0，但新的合同测试因 Runner glob 被遗漏；组件已导出但实际 UI 仍调用旧接口。

知识选择：[K05 测试全域](K05/systematic-syllabus.md)、[False PASS](K05/topics/FALSE_PASS.md)、[真实集成](K05/topics/INTEGRATION.md) 和 [K08 阶段](K08/topics/PIPELINE_STAGES.md)。

预期：报告真正已运行的测试范围，未执行材料目标保持 NOT_RUN；选真实消费者调用路径和最小必要负例。错误答案是因 CI 绿色/Mock 被调用就报 E2E PASS；也不应无条件重跑无关全矩阵。

### Q05：多 Agent 旧 ACK 与 Host 副作用未知

输入：Worker A 基于旧 HEAD=a111 修改，当前 HEAD=b222，A 返回成功 ACK 且远端操作可能已发生。

知识选择：[K07 协作系统课程](K07/systematic-syllabus.md)、[旧消息](K07/topics/STALE_RESULT.md)、[K10 未知副作用](K10/topics/UNKNOWN_EFFECT.md) 和 K08 来源证据。

预期：ACK 不等于 GitHub canonical Task/Claim/Review/Release，当前 HEAD 不能由旧消息晋升 PASS；Effect 未知时先合法状态读回，不做危险重试。错误答案是覆盖 b222 或以相同 GitHub 登录名否定一切独立 Reviewer。

### Q06：不可逆迁移与经济性的底线

输入：数据库迁移仅三行代码，旧数据含非法状态值，支持滚动升级；本轮有人提出跳过迁移测试。

知识选择：[K09 维护课程](K09/systematic-syllabus.md)、[数据迁移](K09/topics/DATA_MIGRATION.md)、K01 业务规则、K05 Oracle、[K11 必要保证](K11/topics/ASSURANCE_FLOOR.md)。

预期：对无合法映射的旧值保持 UNKNOWN、说明兼容与中断恢复以及不可逆清理需有权决议。错误答案是 Fresh install 建表通过就宣布真实旧客户升级全部完成，或因 LOC 少取消适用的恢复验证。

## 2.1 R3 新增六种基础学科推理场景（教学 Case，不是真实 Agent E2E）

### Q07：不完整用户访谈带来的架构过度设计

**输入**：用户说“系统不能慢”，需求访谈只覆盖高频用户；部分用户在低网络质量环境下工作，实际数据量、服务等级与恢复需求未知。架构师建议直接部署跨区域数据库。

**专业路由**：[K01 第二卷](K01/foundations-deep-dive-02.md) 的观察偏差/可测质量场景 → [K02 第二卷](K02/foundations-deep-dive-02.md) 的边界/选型 → K11 价值成本权衡。

**预期**：区分观测、口头诉求、真实授权与未知；提出需要澄清的刺激/环境/响应度量，比较简单设计与跨区域替代。**反例**：把“不能慢”当成已批准 99.99% SLA 并宣称异地多活已获权。

### Q08：事务隔离与外部副作用的错误外推

**输入**：两个 Worker 并发修改支付 Job，数据库使用 PostgreSQL Repeatable Read；一次外部付款调用在远端完成后响应丢失，应用重启。

**专业路由**：[K02 第二卷](K02/foundations-deep-dive-02.md) 一致性/故障模型 → [K03 第二卷](K03/foundations-deep-dive-02.md) 条件写/资源 → [K10 第一卷](K10/foundations-deep-dive-01.md) UNKNOWN_EFFECT → K05 状态测试。

**预期**：明确数据库隔离不能回滚远端扣款，读回有权状态、检验稳定业务请求身份，必要时保留 UNKNOWN。**反例**：以 Repeatable Read 名称宣称整个外部流程全局 exactly-once，直接重试付款。

### Q09：Pairwise 测试错过三元权限条件

**输入**：Actor=admin/user、State=OPEN/CLOSED、Action=READ/CANCEL 的测试矩阵做了 Pairwise，业务禁止普通用户 CANCEL 已 CLOSED 的订单，但该三元组合未执行。

**专业路由**：[K05 第二卷](K05/foundations-deep-dive-02.md) 组合强度 → [K01 第一卷](K01/foundations-deep-dive-01.md) 受权政策 → K08 真实 Runner Discovery。

**预期**：说明 2-way 并不保证具体三元组合；新增必要的非法状态 Oracle，当前未执行仍 NOT_RUN。**反例**：以 Pairwise 100% 自动宣传完整业务权限覆盖。

### Q10：两次 Build GREEN 却输出不同文件

**输入**：同一可见 SourceSHA，两次 CI 均 exit 0，包 hash 不同；构建依赖时间戳和外部未固定下载；另有一个新 API 真实消费者仍调用旧导出。

**专业路由**：[K08 第二卷](K08/foundations-deep-dive-02.md) 可复现环境/缓存 → [K05 集成 Topic](K05/topics/INTEGRATION.md) → [K09 第二卷](K09/foundations-deep-dive-02.md) 旧消费者合同。

**预期**：将 Build success、可复现性和真实消费者兼容分开评估，收集具体制品差异与依赖输入，恢复最小受影响真实测试。**反例**：任意哈希不同都断言安全攻击，或两次命令 GREEN 就宣称 bit-for-bit 可复现。

### Q11：平均性能改善但尾延迟恶化

**输入**：一个优化使平均延迟从 100ms 降到 80ms，却在同一负载环境下把 P99 从 400ms 提高到 2s；P99 采样只有 30 个请求，整体样本和环境差异没有记录。

**专业路由**：[K10 第二卷](K10/foundations-deep-dive-02.md) 尾延迟/采样 → [K03 算法 Topic](K03/topics/ALGORITHM_CHOICE.md) → [K11 第二卷](K11/foundations-deep-dive-02.md) 实证统计。

**预期**：确认哪个指标受业务认可、分位数采样是否可信、错误/正确性是否改变；不可直接承诺优化更好或变差，提出等价真实路径的重复/分层测量。**反例**：只引用平均下降 20% 宣称 SLO 一定改善。

### Q12：知识辅助 Agent 的错误因果评价

**输入**：实验组使用新知识库和更强模型、更多 Token/工具权限，在 10 个任务中通过 8 个；对照组使用旧模型，在 10 个不同任务中通过 6 个。

**专业路由**：[K11 第二卷](K11/foundations-deep-dive-02.md) 实验设计与统计 → [K07 第二卷](K07/foundations-deep-dive-02.md) Agent 协作/偏差 → K05 独立 Oracle 与真实执行范围。

**预期**：明确观察差异 ≠ 知识导致的因果提升，要求冻结可比任务、Source SHA、工具权限/模型、独立 Oracle、复现/置信边界；负收益要记录。**反例**：用 8/10 vs 6/10 直接写“软件工程能力提高 33%”，忽略混杂因素、样本误差和不同任务。

以上六个案例全部为 **TEACHING_SYNTHETIC / NOT_AGENT_RUN / NOT_PRODUCTION_EVALUATION**；不能依它们授予任何 Source Claim、Rule、Product/L2 或 DomainHarness 消费资格。

## 2.2 R4 六种专业补缺的跨模块推理反例（合成教学，Q13–Q18）

### Q13：租户共享缓存的结构性隔离缺陷

**输入**：两个租户存在同名 Task ID，后台通过 `cache[taskId]` 获取结果；单租户测试绿，跨租户顺序访问结果混淆。**知识选择**：[K02.F4.3](K02/focus-f4-3-minimal-failure-and-trust-boundary.md) 信任/状态边界 → K01 Actor/Rule → K05 跨租户 Oracle → K10 隔离恢复。**预期**：说明 ID 身份/缓存作用域、受权主体和可能数据泄露，提出 A/B 顺序和交错反例；缺实际身份合同则 UNKNOWN。**禁止**：以 TypeScript 类型检查成功或 UI 不显示其他租户认定安全。

### Q14：Worker DONE 与合法任务接受不同

**输入**：Agent A 发 DONE，PR 仍 OPEN，受影响合同测试未执行；Reviewer 旧 SHA 的结论无法覆盖当前修改。**知识选择**：[K04.F2.3](K04/focus-f2-3-task-acceptance-and-authority.md) → K07 ACK/Attempt → K05 正反验证 → K06 精确差异。**预期**：区分 Worker 报告、交付、实证与有权决定，保留历史 ACCEPT_SCOPED，当前受影响范围 WAIT/NOT_RUN。**禁止**：把 Agent DONE 或旧 Review 直接写成当前 Release。

### Q15：跨 OS 安装包存在性被 Node 测试误证明

**输入**：Linux 下 Node 单元测试全部通过，Windows 打包步骤未执行，用户请求下载安装器。**知识选择**：[K05.F3.3](K05/focus-f3-3-build-package-platform-verification.md) → K08 Build/Artifact 身份 → K10 首配合同。**预期**：单元/打包/安装/首次运行各自单列证据，对 Windows 包保持 NOT_BUILT/NOT_RUN。**禁止**：以跨平台 JavaScript 测试绿声称 Windows 安装文件可下载。

### Q16：Snapshot 更新将错误输出洗成绿色

**输入**：H2 引入跨租户数据混淆，使 H1 Golden 测试失败；Writer 更新 Golden 后报告测试绿色。**知识选择**：[K05.F4.2](K05/focus-f4-2-golden-snapshot-evidence-freshness.md) → K01 权限规则 → K02 隔离架构。**预期**：识别 Expected Owner 不可由当前错误实现自动赋予，恢复独立租户负例；对确实合法的无语义输出变更可以受控调整。**禁止**：批量更新 Snapshot 就认定回归已修复。

### Q17：修补日志后重新引入旧补丁的错误

**输入**：H1 已针对请求 ID 固定幂等，H2 改变队列的 ID 生成位置；H1 审核与 CI 通过但 H2 真实请求重复扣费可能发生。**知识选择**：[K06.F4.3](K06/focus-f4-3-patch-currentness-and-review.md) → K03 幂等 → K05 当前 Oracle → K08 真实执行记录。**预期**：分析 H1→H2 行为影响闭包，保留不受影响旧证据，对关键新路径安排小范围复测；Effect 未知先合法读回。**禁止**：以“改的是日志”推断无语义影响或复制 H1 PASS。

### Q18：必须设置环境变量才能打开桌面项目

**输入**：桌面工具启动前要求用户手写唯一仓库环境变量，用户希望多项目管理、首次运行后向导、无配置也能打开安全页面；Headless 运行不允许挂起等待 UI。**知识选择**：[K10.F5.1](K10/focus-f5-1-first-run-post-start-configuration.md) → K01 产品入口要求 → K05 真实启动/负例 → K08 目标平台制品。**预期**：区分 GUI 进程启动、配置/授权、业务 READY、Headless 的错误退出；在不同项目 Profile 间安全隔离。**禁止**：以 Twelve-Factor 的服务 env 方法一刀切要求所有桌面 UI 启动前配置，或未配置时默认远端仓库执行 Effect。

**R4 六例均为 SYNTHETIC_TEACHING_CASE_ONLY / REAL_AGENT_HOST_NOT_RUN。** 提供正确推理与错误外推的判别框架，不是 6 次真实领域模型测试，也不授权生产消费。

## 3. 正式消费前的实际评估维度

| 维度 | 应检查 | 不可代替的事实 |
| --- | --- | --- |
| 专业覆盖 | 能选对学科群/概念/方法，发现遗漏 | 关键词命中不是专业理解 |
| 适用性与冲突 | 角色/状态/版本差异能改变结论，缺失时 ASK/UNKNOWN | 模型自信不是 Rule |
| 原理和推理 | 提供实际机制、权衡和负例 | 长答案不是质量证明 |
| 真实使用 | 合法 Task/Host/CI 当前收据与用户结果 | 教学 Case 不是 PASS |
| 来源与权利 | 原始链接/版次/Clause/版权和适用界限 | 目录累计来源数不是 Claim 资格 |
| 工程收益 | 同任务基线与辅助组的正确性/返工/耗时和成本 | Token 用量和 PR 数不能代替业务结果 |

以上 Q01–Q18 共十八例可成为未来在 #286 合法消费者版本中进行有限评估的候选数据。当前只能证明**知识选择与判断预期已被文档化**，不能证明 Tally/DomainHarness 当前代码真正能返回这些结果。
