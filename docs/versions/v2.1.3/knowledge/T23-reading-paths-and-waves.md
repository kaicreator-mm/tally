# T23-R1 · 实际知识阅读路径与下一批细分计划

冻结输入 `383358955d117d155e2b3b2fc4cd7886e77f71b9`，#584。与 [盘点合同](T23-inventory-contract.md)、[初始细节盘点](T23-detail-inventory.md)、[归属台账](T23-semantic-ownership.md) 配套。P2-PLAN-04 的目标是对人和 LLM 都可读的共享正文，而非新搜索引擎、Skill ABI 或两套语料。以下阅读演练是作者实际只读导航与原创问题示例，未运行任何模型/工具验证/外部项目，检索准确率与质量收益 NOT_MEASURED。

## 1. 阅读组织与实质检查

- 人类学习：从 [原课程及两卷地图](../../v2.1.2/software-engineering-knowledge/systematic-foundation-depth-map.md) 进入主题，先概念/前置、再机制/推导、例/反例、进一步深文及来源。以实际目标选深度，不要求所有章同样形式化。
- LLM/工程决策：从可辨识问题词（中/英文同义和含义区别）进入 [T22 当前入口](casebook/waves/T22-current-knowledge-entry.md)、[R10 选知手册](../../v2.1.2/software-engineering-knowledge/professional-knowledge-selection-playbook-r10.md) 或 T23 盘点组；读最少必要权威正文，再按事实补选择条件、真实版本、失败边界、Oracle 和未知。知识不授予业务/源码/执行权限。
- 术语导航非自动合并：幂等/idempotence 与去重/deduplication 不同；原子性/atomicity 与 durability 不同；timeout 与 failed 不同；回滚/rollback 与补偿/compensation 不同；测试预言/test oracle 与 observation 不同；build cache 与运行时 cache 不同。版本化细节追原规范或源；历史正文及独立资格通过固定 SHA 引用。以下四个路径均核实际段落，未做全文检索成功率试验。

## 2. 实际阅读路径

### N01 · 任务执行：同步、内存任务还是持久化队列？

- 人类：[K02 D1 模块与状态所有权 L13–23](https://github.com/kaicreator-mm/tally/blob/383358955d117d155e2b3b2fc4cd7886e77f71b9/docs/versions/v2.1.2/software-engineering-knowledge/K02/foundations-deep-dive-01.md#L13) → [K03 D2 背压 L30–37](https://github.com/kaicreator-mm/tally/blob/383358955d117d155e2b3b2fc4cd7886e77f71b9/docs/versions/v2.1.2/software-engineering-knowledge/K03/foundations-deep-dive-02.md#L30) → [T18-C01/C02/C03 L13–37](https://github.com/kaicreator-mm/tally/blob/383358955d117d155e2b3b2fc4cd7886e77f71b9/docs/versions/v2.1.3/knowledge/casebook/waves/T18-job-architecture-contrasts.md#L13) → [K10 D1 超时、恢复 L13–31](https://github.com/kaicreator-mm/tally/blob/383358955d117d155e2b3b2fc4cd7886e77f71b9/docs/versions/v2.1.2/software-engineering-knowledge/K10/foundations-deep-dive-01.md#L13)。实际读到直接函数/Inline 的完成区别、Async 默认无限 backlog、持久认领不等于业务效果原子性；T18 外部源行为 NOT_RUN，不从历史代码推出新任务运行资格。
- 决策入口：job queue / backlog / admission / durable acceptance / recovery；先问调用者需同步结果还是仅受理？允许丢失、重复、延期？重启恢复？负载与预算？无持久承诺、短且同步操作可直接函数；可丢失的异步可用进程内；确有持久待办和恢复合同才考虑持久队列。若可靠重放且禁止重复效果，需核稳定身份、远端幂等/可信读回与失效范围。反例：限制 worker 数但待办无限；已提交外部效果后崩溃。缺实际业务合同则 UNKNOWN，不选具体库或通用规则。Oracle 是崩溃前后可见状态与效果计数（拟议，NOT_RUN）。组 FM/DA/OP/RE，Owner K02/K03/K10/K05。

### N02 · API 超时：为什么不能直接重试？

- 人类：[K01 D1 意图、事件与授权 L13–36](https://github.com/kaicreator-mm/tally/blob/383358955d117d155e2b3b2fc4cd7886e77f71b9/docs/versions/v2.1.2/software-engineering-knowledge/K01/foundations-deep-dive-01.md#L13) → [K03 线性化与局部不变量 L7–32](https://github.com/kaicreator-mm/tally/blob/383358955d117d155e2b3b2fc4cd7886e77f71b9/docs/versions/v2.1.2/software-engineering-knowledge/K03/advanced-concurrency-idempotency-proof-sketch.md#L7) → [K10 D1 超时 L13–31](https://github.com/kaicreator-mm/tally/blob/383358955d117d155e2b3b2fc4cd7886e77f71b9/docs/versions/v2.1.2/software-engineering-knowledge/K10/foundations-deep-dive-01.md#L13) → [K05 D1 选例及性质 L13–32](https://github.com/kaicreator-mm/tally/blob/383358955d117d155e2b3b2fc4cd7886e77f71b9/docs/versions/v2.1.2/software-engineering-knowledge/K05/foundations-deep-dive-01.md#L13)。实际读到本地串行化/线性化对象不同、响应丢失可能已提交、重复/不变量测试需要业务范围。
- 决策入口：timeout / lost ACK / unknown effect / idempotency；先确定动作是否有外部效果、允许的重复/丢失合同、远端身份与查询有效期。未到达、已提交但 ACK 丢失、还在执行是竞争解释；只有真实远端合同和恢复策略允许才重放。若允许丢失但不允许重复，可拒绝自动重放；未知且不可逆须保留 UNKNOWN。反例：新 key 重试导致重复支付；本地状态 PAID 不是银行事实。按真实身份和外部提交读回制定 Oracle（NOT_RUN）。组 SY/DA/QA/OP。实际业务政策不来自教学示例。

### N03 · 首次启动与配置：桌面、CLI、服务是否应统一？

- 人类：[K10 Focus L7–27](https://github.com/kaicreator-mm/tally/blob/383358955d117d155e2b3b2fc4cd7886e77f71b9/docs/versions/v2.1.2/software-engineering-knowledge/K10/focus-f5-1-first-run-post-start-configuration.md#L7) → [K08 D2 制品与消费者 L25–37](https://github.com/kaicreator-mm/tally/blob/383358955d117d155e2b3b2fc4cd7886e77f71b9/docs/versions/v2.1.2/software-engineering-knowledge/K08/foundations-deep-dive-02.md#L25) → [K05 V&V 测试层 L16–29](https://github.com/kaicreator-mm/tally/blob/383358955d117d155e2b3b2fc4cd7886e77f71b9/docs/versions/v2.1.2/software-engineering-knowledge/K05/advanced-verification-validation-and-oracle-qualification.md#L16)。实际读到进程启动/配置/业务 READY 不同，桌面、CLI、headless 有不同配置方式；真实安装/消费者不能由文档或 Mock 代替。
- 决策入口：first-run / config / readiness / authorization；先问应用形态、是否交互、必要配置和业务权限、失败/撤权可见状态。已授权的特定交互式产品可先 Shell 后设置，headless 可合法外部配置并显式失败；非交互不能无限等待输入。反例：窗口显示等于已连接；无目标时默写默认远端。Oracle 是干净环境真实入口、合法设置和未授权副作用拒绝（NOT_RUN）；未给产品合同就不强制某一种。组 HC/OP，O01 权威引用候选。

### N04 · 软件重构：是否需要迁移旧数据和兼容旧版本？

- 人类：[K09 D1 重构与迁移 L7–37](https://github.com/kaicreator-mm/tally/blob/383358955d117d155e2b3b2fc4cd7886e77f71b9/docs/versions/v2.1.2/software-engineering-knowledge/K09/foundations-deep-dive-01.md#L7) → [K09 兼容方向及迁移 L7–29](https://github.com/kaicreator-mm/tally/blob/383358955d117d155e2b3b2fc4cd7886e77f71b9/docs/versions/v2.1.2/software-engineering-knowledge/K09/advanced-contract-evolution-migration-and-retirement.md#L7) → [K11 投资与风险 L25–29](https://github.com/kaicreator-mm/tally/blob/383358955d117d155e2b3b2fc4cd7886e77f71b9/docs/versions/v2.1.2/software-engineering-knowledge/K11/foundations-deep-dive-02.md#L25-L29) → [K05 V&V L16–29](https://github.com/kaicreator-mm/tally/blob/383358955d117d155e2b3b2fc4cd7886e77f71b9/docs/versions/v2.1.2/software-engineering-knowledge/K05/advanced-verification-validation-and-oracle-qualification.md#L16)。实际读到受支持消费者、非兼容重写例外、姓名拆分不可逆、旧状态不能自动变退款成功。原理并非 Tally 当前版本必须迁移所有历史接口。
- 决策入口：refactor / migration / compatibility / deprecation / rollback；区分保留行为、纠错、增强及明确重写；确定真实消费者、方向、数据历史、并存窗口与允许停机。无共存时可能直接迁移；有重叠时比较 expand/migrate/contract；不允许旧版兼容时只验证新获权合同，仍需评估真实数据处置。反例：代码回滚不恢复字段、状态重命名改变真实业务事实。Oracle 是旧/新支持路径和数据副作用（NOT_RUN）。缺真实授权或状态映射则 UNKNOWN。组 OP/RE/QA/FD，O03/O05。

四条路径均存在定义或机制、实际选择条件、反例、Owner 与固定正文；这是静态文档可达性样本，不代表所有问题可检索或 LLM 性能。下一次增加 HCI/数据/采购等薄弱组路径时要从实际新读正文生成，而非凭目录。

### N05 · 表单错误、焦点与文本输入（HCI）

人读：[T07-C07 真实字段/跨字段/业务验证](https://github.com/kaicreator-mm/tally/blob/383358955d117d155e2b3b2fc4cd7886e77f71b9/docs/versions/v2.1.3/knowledge/casebook/waves/T07-business-functions.md#L53-L60) → [K03 错误与安全下一步](https://github.com/kaicreator-mm/tally/blob/383358955d117d155e2b3b2fc4cd7886e77f71b9/docs/versions/v2.1.2/software-engineering-knowledge/K03/advanced-error-semantics-observability-and-validation-proof.md#L7-L17) → [WCAG 2.2 SC 3.3.1](https://www.w3.org/WAI/WCAG22/Understanding/error-identification) 与 [2.4.3](https://www.w3.org/WAI/WCAG22/Understanding/focus-order.html)（实际阅读 Intent） → [Unicode UAX29 rev49 §3](https://www.unicode.org/reports/tr29/tr29-49.html#Grapheme_Cluster_Boundaries)。既有表单分层不能代替用户实际理解/纠错。决策词：input validation / focus order / grapheme cluster / accessible error。假设用户填写多语言姓名，校验失败时只改变边框颜色、清空输入或把焦点跳到不相关按钮，都可能使纠错失败。先明确字段/跨字段/业务规则来源及真实使用者、输入法/语言/辅助技术；按平台规定设计错误文字、焦点顺序、保留合法输入与安全恢复。长度合同先区分字节、代码点、用户感知字符；不得任意改变身份语义。Oracle 候选：键盘操作、屏幕阅读、组合字符/本地化、提交失败后恢复（NOT_RUN）。K01 用户目标/K03 实现/K05 评价；HC/FD/QA。Web 规范不自动约束全部 CLI；未知目标平台需询问。

### N06 · 采购/自建/复用：供应商接口如何验收与退出？

人读：[K01 目标与追溯](https://github.com/kaicreator-mm/tally/blob/383358955d117d155e2b3b2fc4cd7886e77f71b9/docs/versions/v2.1.2/software-engineering-knowledge/K01/advanced-requirements-traceability-and-change-propagation.md#L7-L32) → [K11 工期、风险、信息价值](https://github.com/kaicreator-mm/tally/blob/383358955d117d155e2b3b2fc4cd7886e77f71b9/docs/versions/v2.1.2/software-engineering-knowledge/K11/advanced-decision-risk-value-of-information-and-process-cost.md#L7-L29) → [K02 边界与消费者](https://github.com/kaicreator-mm/tally/blob/383358955d117d155e2b3b2fc4cd7886e77f71b9/docs/versions/v2.1.2/software-engineering-knowledge/K02/advanced-extension-authority-state-and-consumer-boundaries.md#L7-L17) → [SEI 2005 HB-006 官方摘要](https://www.sei.cmu.edu/library/software-acquisition-planning-guidelines/)。实际 SEI 仅读摘要；NASA acquisition 正文直开 403。决策词：COTS / make-or-buy / supplier acceptance / exit。先问核心能力是否可由现有库/服务满足，谁负责数据、接口版本、合法使用范围、服务停止后的导出/迁移，何种验收可独立观察。教学例：采购通知服务仅比较月费，不检查失败/重复、导出、替换；低初始成本不保证生命周期低成本。小型稳定本地函数也不必采购。拟议 Oracle 为合同允许的用户路径、失败/停止及数据退出（NOT_RUN），非合同法律解释。K04 管理/K01 验收目标/K02 边界/K11 经济；SY/RE。

### N07 · 导入、查询、迁移：返回成功是否表示全部数据正确？

人读：[T05-C04 流解析](https://github.com/kaicreator-mm/tally/blob/383358955d117d155e2b3b2fc4cd7886e77f71b9/docs/versions/v2.1.3/knowledge/casebook/waves/T05-common-function-abstractions.md#L32-L40) → [T05-C07 查询迭代](https://github.com/kaicreator-mm/tally/blob/383358955d117d155e2b3b2fc4cd7886e77f71b9/docs/versions/v2.1.3/knowledge/casebook/waves/T05-common-function-abstractions.md#L59-L67) → [T19 批选取与部分结果风险](https://github.com/kaicreator-mm/tally/blob/383358955d117d155e2b3b2fc4cd7886e77f71b9/docs/versions/v2.1.3/knowledge/casebook/waves/T19-feed-refresh-architecture.md#L13-L17) → [K09 有损映射](https://github.com/kaicreator-mm/tally/blob/383358955d117d155e2b3b2fc4cd7886e77f71b9/docs/versions/v2.1.2/software-engineering-knowledge/K09/advanced-configuration-migration-and-evidence-impact-closure.md#L17-L27) → PostgreSQL 18 [约束 §5.5](https://www.postgresql.org/docs/18/ddl-constraints.html) / [隔离 §13.2](https://www.postgresql.org/docs/18/transaction-iso.html)。决策词：import/export / partial read / NULL / integrity / schema migration。先问整批原子或逐条、允许缺失/重复、失败后报告及重放、数据 Owner 与历史兼容；一个 JSON 值成功解码或 SQL 返回 Rows 不证明读取全部完成。具体 PostgreSQL FK 可空策略要核列约束，不能用一种 DB 行为代表全部引擎。原始字节与合法领域转换分开；迁移单名或 CANCELED→REFUNDED 不允许臆造事实。候选 Oracle：截断/尾随/迭代错误、NULL FK、旧值映射、提交中断、恢复（NOT_RUN）。本地文件 rename/断电持久性仍需目标 OS 原文，不能由上述路径证明。K03 实现/K09 演进/K02 合同/K05 验证；DA/OP。

### N08 · 工程报告与专业责任：发现风险时如何交接？

人读：[K01 陈述、观察、权限](https://github.com/kaicreator-mm/tally/blob/383358955d117d155e2b3b2fc4cd7886e77f71b9/docs/versions/v2.1.2/software-engineering-knowledge/K01/advanced-stakeholder-authority-and-triangulated-evidence.md#L7-L30) → [K07 独立性与充分上下文](https://github.com/kaicreator-mm/tally/blob/383358955d117d155e2b3b2fc4cd7886e77f71b9/docs/versions/v2.1.2/software-engineering-knowledge/K07/advanced-agent-role-context-and-outcome-evaluation.md#L7-L26) → [K03 安全行动](https://github.com/kaicreator-mm/tally/blob/383358955d117d155e2b3b2fc4cd7886e77f71b9/docs/versions/v2.1.2/software-engineering-knowledge/K03/advanced-error-semantics-observability-and-validation-proof.md#L7-L17) → [K11 后果与调查价值](https://github.com/kaicreator-mm/tally/blob/383358955d117d155e2b3b2fc4cd7886e77f71b9/docs/versions/v2.1.2/software-engineering-knowledge/K11/advanced-decision-risk-value-of-information-and-process-cost.md#L17-L29)。决策词：technical communication / risk disclosure / handoff / professional competence。教学情境：工程师收到疑似重复数据报告，不能因负责人希望按期发布就写“确定无问题”，也不能在无复现时断言根因。报告区分观察/解释、当前版本与未知、用户影响、风险/可执行下一步，交接给有权责任人；不泄露秘密，不将不具备专业能力的领域决策冒充批准。一般实践不强制每个项目使用 Exact SHA 或 P0/P1/P2；现有 K07 是部分前提，职业伦理、心理安全、许可和技术写作仍需专题建设。ACM HTML 403，未核其正式条款，不能据此声称完整职业标准。Oracle 为读者能否定位事实、限制和责任（仅拟议，未进行用户实验）。K07/K04/K01/K11，PR/QA。

## 3. T24–T36 子流与前置关系候选

T23 先对盘点表中 D/U 的高风险目标继续搜实际相关 Topic/Advanced/Case，判断哪些已有充分解释，哪些需补深；每个子流只冻结已核目标及真实 Owner，持续多批建设。下列编号是工作波次，不是知识分类或强制串行 DAG。任何“前置”仅在使用其尚未确定的具体产物时成立；可引用已有足够知识，不必等所有专题完成。T35 可在第一组足够机制后提供组合参考，T36 是阶段整合，不是长期建设终止。P-high/P-medium 是基于专业风险的建议，不是实测频率。

| 子流 | 细粒度主题及交付重心 | 优先/依赖条件 |
|---|---|---|
| T24a/b/c | a需求发现、规格及追溯；b业务规则、边界/状态/单位/有效时间；c HCI/产品交互、可用性与无障碍 | 高；先目标与观察，再选技术；依 SY/HC 实际缺口 |
| T25a/b/c/d | a离散数学、逻辑、算法与复杂度；b语言语义、数据表示/类型；c OS/网络/内存模型/并发和分布式基础；d概率统计、数值与工程测量 | 核心最低深度高，专项按需；已有 ALG 等可引用，避免与 T27 重复 |
| T26a/b/c | a架构方法/系统接口与视图；b单机/客户端/服务/分布式真实完整产品代表路径；c专项平台及产品组合（移动/嵌入式/实时/科学/图形/AI） | a高，b按真实范围，c专项；依已确定业务/质量场景 |
| T27a/b/c/d | a函数/模块/类型/抽象/资源；b数据模型、IO、事务、API协议；c并发/状态/故障机制；d版本化语言实现 | 高；基础机制引用 T25，不重复定义；真实案例与反例 |
| T28a/b/c | a测试设计与 Oracle；b静态、模型、形式化验证；c质量保证、产品质量与 HCI 验证 | 高；按缺陷风险而非测试量，T29 同步补真实失败 |
| T29 | 真实缺陷族：issue→原始版本/现象→根因候选→patch→回归；调试工具选择、错误修复和反例 | 高；使用实际历史，未运行 NOT_RUN，不按仓库刷案例 |
| T30a/b/c | a生命周期、管理/采购/供应商；b技术写作、人类专业实践/团队协作；c Agent 应用 profile、语境/交接 | a/b高，c已有较多正文；K11 成本共享，非 ADS 重写 |
| T31a/b/c | a配置/构建/依赖/版本；b集成、制品、安装/发布；c 平台运行入口 | 中高；明确工具版本和实际消费者，避免只讲 Woodpecker |
| T32a/b | a维护、重构、替换及产品线；b数据/接口迁移、共存、退役 | 高；依实际支持合同，复用 MIG/T20 |
| T33a/b/c | a故障、可靠性与恢复；b容量、性能、测量；c运营计划、事故、服务支持 | 中高；引用 K03 机制，K06 因果诊断 |
| T34a/b | a经济、估算、风险、采购/复用选择；b实证、度量、人因评价 | 中高；不以历史成本假设冒充实测 |
| T35 | 按通用功能族优先：配置/错误/文件/查询/任务/同步/交互，再有权业务功能族；最小实现、替代、合同、失败、Oracle 与真实例 | 高，按已足够主题滚动开展；仅组合引用，不复制机制或定义行业政策 |
| T36 | 综合入口、语义/来源去重、适用性/未知、实际阅读可达性、限定验收 | 本批整合；不宣称全域完整或产品运行 |

## 4. 当前边界

R1 历史四条样本保留（N04 成本定位修正）；R2 新增四条静态路径，合计八条。PLAN-01/02 已在 R1 限定关闭，PLAN-03/04 与 R1 新三 P2 待本次独审。实际目录 56 组仍为候选分组；[R2 风险交叉盘点](T23-R2-risk-crosswalk.md) 提出首批四个有限写作范围，后续是否批准由本次独审决定，不冻结全库。禁止自动派发 T24–36，禁止启动新代码、SDK、运行、CI 或 Agent 实验。#565 HOLD，#286 DEFERRED，T4/A24 NOT_RUN/HOLD。