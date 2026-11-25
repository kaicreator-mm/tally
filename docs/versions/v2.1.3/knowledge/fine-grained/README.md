# 第三批细粒度软件工程知识 · 专业正文入口

[#586](https://github.com/kaicreator-mm/tally/issues/586)，基线 `daba091d8d9249c8a2f0100c0bae2c5464929750`。T23 已限定接受 [#584 closeout](https://github.com/kaicreator-mm/tally/issues/584#issuecomment-7166217999)，四个有限写作范围已获准入。第一组四篇已在 [#586 closeout](https://github.com/kaicreator-mm/tally/issues/586#issuecomment-7166574037) 限定接受并合入（0/0/3）；第二组及三项限定修正已获 [#588 独审](https://github.com/kaicreator-mm/tally/issues/588#issuecomment-7167193694) 限定接受（0/0/1），实际 main 内容已核实，PR #589 元数据已恢复为 closed/merged（[#588 reconciliation](https://github.com/kaicreator-mm/tally/issues/588#issuecomment-7170371776)）；第三组已在 [#591 closeout](https://github.com/kaicreator-mm/tally/issues/591#issuecomment-7169778993) 限定接受并合入（0/0/1）；第四组已在 [#592 closeout](https://github.com/kaicreator-mm/tally/issues/592#issuecomment-7170367545) 限定接受并合入（0/0/2）；第五组已在 [#594 closeout](https://github.com/kaicreator-mm/tally/issues/594#issuecomment-7170892606) 限定接受并合入（0/0/2）；第六组已在 [#596 closeout](https://github.com/kaicreator-mm/tally/issues/596#issuecomment-7171454299) 限定接受并合入（0/0/1）；第七组已在 [#598 closeout](https://github.com/kaicreator-mm/tally/issues/598#issuecomment-7172122515) 限定接受并合入（0/0/1）；第八组已在 [#600 closeout](https://github.com/kaicreator-mm/tally/issues/600#issuecomment-7172639924) 限定接受并合入（0/0/1）；第九组为候选，待独审；不是新的平行课程体系或合格 Source Claim。保留 K01–K11、165 历史课程 ID、48 概念关系与 T22 142 Case/143 Blob/164 关联（129 M/35 C）原始身份及资格。

| 子任务 / 归属 | 正文及阅读问题 | 新增深化 | 既有知识复用 |
|---|---|---|---|
| T24a K01（K02/K05/K11） | [需求获取、协商与系统责任分配](T24a-requirements-discovery-negotiation.md)：如何选择调查方法、解决冲突、分配责任和验收？ | 访谈/观察/工作坊/文档/日志/原型、边界分配、可行性、优先级、离线库存教学反例 | SY02/03/04/05 接口部分；STK/REQ，K11 风险 |
| T24c K01（K02/K03/K05） | [用户任务、交互合同与可访问纠错](T24c-human-computer-interaction.md)：用户怎样理解状态和恢复错误？ | 任务/心理模型、信息组织、导航、状态反馈、可访问输入、Unicode 文本单位、用户观察 | HC01–04 CORE；T07-C07/ERR，N05 |
| T27a K03（K02/K05） | [详细设计：表示、修改边界、内聚与语义变化](T27a-detailed-design-representation.md)：什么必须被一个边界保护？ | RI/AF、表示泄漏、frame/别名、控制/数据流、资源及错误合同、区间集合纸面推导 | FM04–06；ABS/ALG、T17/T16 |
| T27b-1 K03.F4.1（K02.F3.3/F3.1/K05） | [关系数据设计、查询语义与事务隔离](T27b-1-relational-data-and-transactions.md)：约束、查询范围和并发正确性怎样保证？ | 函数依赖与范式、键/NULL/FK、JOIN/聚合、索引/计划、隔离/写偏斜 | DA01–03；CON、T05-C07/T19 |

## 第二组专业正文及限定修正（#588）

基线 0b760ec98f1f482a425674788931a52fdc8176a5；[首批审核](https://github.com/kaicreator-mm/tally/issues/586#issuecomment-7166522664)。P2-FG-01 去重复选择矩阵，-02 补无损分解/依赖保持推导，-03 区分低基数与谓词选择率，均已由第二批独审限定关闭；新增 P2-FG-04 流协议补充转入第三组。

| 子任务 / 归属 | 实际新增正文 | 阅读问题及边界 |
|---|---|---|
| T25b-1 K03/K01/K05 | [编程语言语义、数值和文本表示](T25b-1-language-value-and-text-semantics.md) | FM05/FD02/HC04；名称、绑定、闭包、别名、数值精度与字符单位、归一化；ECMAScript 2025/Python 3.11/Unicode17，非全语言及 locale 规范 |
| T27b-2 K03/K02/K09/K10/K05 | [文件、流、部分 IO、可见性及持久性](T27b-2-file-stream-and-durability.md) | DA06/RE02；路径/句柄、短读写、分帧、原子替换、sync 与恢复、导入导出；Linux/Node 22.17.0，不推广 Windows/远端 FS |

两篇均获限定接受；教学例与 Oracle NOT_RUN。原已接受四篇中的两篇仅改三项 P2，不重审全部；T24a/T24c 正文不变。

## 第三组专业正文及流协议修正（#591）

基线 d382b0091fee6989777421d5ce48280f11624a2f；[候选原稿](https://github.com/kaicreator-mm/tally/issues/563#issuecomment-7167352369)，[第二批独审](https://github.com/kaicreator-mm/tally/issues/588#issuecomment-7167193694)，[实际 main 内容核验](https://github.com/kaicreator-mm/tally/issues/588#issuecomment-7167297467)。本批限定写入 PR #590；此前新建 Task 接口失败，现已恢复并创建 [#591](https://github.com/kaicreator-mm/tally/issues/591)，关联 #563 原稿、#588 延续记录与现有 PR #590，不重复建 PR。P2-FG-04 已补入 T27b-2 §3.1（流背压/终止、严格增量解码、取消与 Oracle），经第三批独审限定关闭；新增 FG05 转入第四组。

| 子任务 / 归属 | 实际新增正文 | 阅读问题及边界 |
|---|---|---|
| T27c-1 K03/K02/K10/K05 | [并发、异步、状态所有权和任务生命周期](T27c-1-concurrency-task-lifetime.md) | FM06/DA03/05/OP06；同步关系、锁与状态、任务准入、容量、取消收敛；Go 内存模型/Python 3.11 结构化并发的限定例，不涵盖全部语言或分布式共识 |
| T28a-1 K05/K01/K03/K06 | [测试选择、状态模型、生成式验证与隔离](T28a-1-test-design-stateful-isolation.md) | QA02/03；缺陷模型、边界/状态、性质/变形/差分、Fuzz、Fixture 与可重复性；非完整质量体系/真实缺陷库 |

教学与 Oracle 全部 NOT_RUN；两篇获限定接受，不增加合格 Claim。

## 第四组专业正文及测试 Oracle 修正（#592）

基线 113dfa02a5c7fe1caa803af15c4ef98e6e4ace2d；[第三批独审](https://github.com/kaicreator-mm/tally/issues/591#issuecomment-7169113951)。FG05 在 T28a-1 §3.1 增加身份稳定、原始持久基数、未授权前后状态及 Effect 的独立观察，经第四批独审限定关闭。

| 子任务 / 归属 | 实际新增正文 | 阅读问题及边界 |
|---|---|---|
| T28b-1 K05/K03 | [静态分析、抽象解释及形式验证](T28b-1-static-analysis-and-verification.md) | FM08/QA04；CFG/数据流、格/固定点、路径敏感、SAT/SMT、有限模型；未运行分析器或证明器 |
| T29a-1 K06/K05/K03 | [真实缺陷与修复](T29a-1-historical-defect-repair-reasoning.md) | QA05/06；CPython 迭代语义、pytest/xdist 生命周期回归；真实 issue/commit/test diff，未复现/执行，非新增 T22 catalog Case |

两篇已获限定接受，FG05 已关闭；FG06/07 非阻断转入第五组。旧 T22 数据和 Claim 资格不变。

## 第五组专业正文与 FG06/07 修订（#594）

基线 e48f2d6b36cdaf7d9f240e1e8d89d3c801548793；[第四批独审](https://github.com/kaicreator-mm/tally/issues/592#issuecomment-7170213611)。FG06 在 T28b-1 §2 精确区分有限高度与有限元素数量、α/γ/join、不可达 transfer；FG07 在 T29a-1 精确补历史日期、完整 bisect 报告 SHA、测试断言及局部/全局停止范围。均经第五批独审限定关闭。#589 元数据已恢复，不重复合并。

| 子任务 / 归属 | 实际新增正文 | 阅读问题及边界 |
|---|---|---|
| T25c-1 K03/K10/K05 | [操作系统：进程、地址空间、IPC](T25c-1-os-process-memory-ipc.md) | FD03；Linux process/fork/exec/wait/pipe/mmap、继承与共享、驻留和成本；非完整 OS/Windows/容器规范 |
| T25c-2 K03/K02/K10/K05 | [网络传输与应用协议](T25c-2-network-transport-and-protocol.md) | FD03/DA05/FM07；TCP byte stream、分帧、ACK/业务效果、HTTP 方法/条件请求、期限及池；非完整网络栈/共识 |
| T25b-2 K03/K01/K02/K06/K10 | [时间、时钟、日历与生效](T25b-2-time-clock-calendar-semantics.md) | FD04；Linux clocks、Python3.11 zoneinfo/fold/gap、有效/记录时间、调度恢复；非完整时间尺度/分布式时钟规范 |

三篇获限定接受，FG06/07 已关闭；FG08/09 转入第六组。教学例及 Oracle NOT_RUN，不新增合格 Claim。

## 第六组专业正文与 FG08/09 修订（#596）

基线 71d6263bf7cd3098b7755e0baff0c2de7cc5ac5c；[第五批独审](https://github.com/kaicreator-mm/tally/issues/594#issuecomment-7170763805)。FG08 在 T25c-1 补多线程 fork/exec async-signal-safe、原子 CLOEXEC、独立启动错误通道、pipe EOF/原子写/回收及 Oracle；FG09 在 T25b-2 补期限控制流、重新计算预算、UTC 候选去重/gap/日历反例。均经第六批独审限定关闭。

| 子任务 / 归属 | 实际新增正文 | 阅读问题及边界 |
|---|---|---|
| T26a-1 K02/K01/K11/K05 | [架构场景、视图、质量权衡与产品线可变性](T26a-1-architecture-scenarios-views-variability.md) | SY05/06、FM04、RE01；实际场景、候选结构、跨视图映射、ATAM 启发评价、合法产品组合及共性资产治理；非正式 ATAM 或完整标准合规 |
| T30b-1 K07/K06/K01/K11 | [专业责任、技术沟通与知识交接](T30b-1-professional-communication-and-responsibility.md) | PR01/02/03；职业伦理、文档受众、事实与估计、同行评审/协商、接手独立验证及事故学习；非法律或组织强制规则 |

两篇获限定接受，FG08/09 已关闭；FG10 转入第七组。全部教学/Oracle NOT_RUN，无新合格 Claim。

## 第七组专业正文与 FG10 修订（#598）

基线 d25bbcd086caeb5490ba58472efb2038c7e50df8；[第六批独审](https://github.com/kaicreator-mm/tally/issues/596#issuecomment-7171229939)。FG10 在 T26a-1 补 Q3 跨网点权限、Q4 旧格式/消费者独立 Oracle 与逐变体质量验证，已由第七批独审限定关闭。

| 子任务 / 归属 | 实际新增正文 | 阅读问题及边界 |
|---|---|---|
| T31a-1 K08/K03/K02 | [编译、链接、加载、ABI 与依赖身份](T31a-1-compilation-linking-abi.md) | GCC14.2、Linux ELF、Node22.17.0；生成阶段/符号/ABI/真实依赖，非全平台 ABI |
| T31a-2 K08/K04/K09/K07 | [配置管理、基线与合并审核](T31a-2-configuration-management-baselines.md) | OP01；配置识别、变更、核算/审核，Git 对象/index/三方合并与语义冲突；非新项目强制流程 |
| T32b-1 K09/K02/K03/K05/K08 | [数据迁移与混合版本消费者](T32b-1-data-migration-and-mixed-consumers.md) | RE02/RE03；并发回填、PostgreSQL18 约束/锁、proto3 二进制演进、对账/回退/退役；非实际迁移执行 |

三篇获限定接受，FG10 已关闭；FG11 转入第八组。教学与 Oracle NOT_RUN，旧 Claim/Case 资格不变。

## 第八组专业正文与 FG11 修订（#600）

基线 b01223cffe8be7995c4df54df5a3dbc4e2c2aa1c；[第七批独审](https://github.com/kaicreator-mm/tally/issues/598#issuecomment-7171788140)。FG11 在 T32b-1 补 PostgreSQL18 DDL 分阶段事务提交、实际锁持续期、RC/RR/Serializable 回填区别及双会话 Oracle，已由第八批独审限定关闭。

| 子任务 / 归属 | 实际新增正文 | 阅读问题及边界 |
|---|---|---|
| T25a-1 K03/K04/K05 | [离散数学：逻辑、关系、图与归纳](T25a-1-discrete-logic-relations-graphs.md) | 量词/集合/等价及偏序/拓扑/归纳，基础纸面推导；非完整离散数学课程 |
| T27d-1 K03/K10/K05 | [数据结构、算法成本与实际工程选型](T27d-1-data-structures-cost-selection.md) | 数组摊还、哈希、堆双索引、图/外存成本，非特定 runtime 性能实测 |
| T25d-1 K11/K10/K05 | [概率、统计测量与工程证据](T25d-1-probability-statistics-engineering-measurement.md) | 概率/相关/样本分母/配对区间/区组、零失败上界，非真实实验或完整统计课程 |

三篇获限定接受，FG11 已关闭；FG12 转入第九组。所有教学及 Oracle NOT_RUN；旧 Claim/Case 资格不变。

## 第九组专业正文与 FG12 修订（#602）

基线 cf76c2142e2bc390ed290c82deb2f366451c1897；[第八批独审](https://github.com/kaicreator-mm/tally/issues/600#issuecomment-7172549977)。FG12 在 T27d-1 §4.1 补活动 ID 唯一、Map/heap 双射 RI、任意位置删除两向修复、证明及独立 Oracle，待独审。

| 子任务 / 归属 | 实际新增正文 | 阅读问题及边界 |
|---|---|---|
| T27c-2 K03/K10/K02/K05 | [分布式一致性、协调与恢复边界](T27c-2-distributed-consistency-coordination-recovery.md) | DA04/05；多数交集、读/Watch/revision、投影恢复、fencing；etcd v3.6 限定，不是 Raft 实现或证明 |
| T28a-2 K05/K08/K03/K02 | [真实消费者、接口合同与安装制品验证](T28a-2-consumer-contracts-installed-artifacts.md) | QA02/03、OP02；Pact 交互/功能分工、版本矩阵、Node v22.23.3 安装边界；非实际安装或合同执行 |

两篇候选与 FG12 一起独审。教学及 Oracle NOT_RUN，无新合格 Claim/Case。

## 阅读路径与边界

以任务检索：需求冲突/验收 → T24a；表单/焦点/输入/用户任务 → T24c；抽象/内聚/表示/alias/frame → T27a；外键/范式/NULL/join/isolation → T27b-1；绑定/数值/Unicode → T25b-1；文件/流/持久性 → T27b-2；竞态/锁/准入/取消 → T27c-1；测试选择/模型/隔离 → T28a-1；控制流/抽象解释/形式验证 → T28b-1；真实缺陷/修复/跨组件回归 → T29a-1；进程/IPC/映射 → T25c-1；网络分帧/ACK/HTTP 条件 → T25c-2；时钟/时区/期限/生效 → T25b-2；架构场景/权衡/产品线 → T26a-1；专业责任/写作/评审/交接 → T30b-1；生成/ABI/加载 → T31a-1；SCM/合并/基线 → T31a-2；迁移/混合版本/回退 → T32b-1；逻辑/关系/图/归纳 → T25a-1；数组/哈希/堆/图成本 → T27d-1；概率/测量/统计推断 → T25d-1；多数/分区/watch/fencing → T27c-2；真实消费者/合同/已安装制品 → T28a-2。需要先选业务目标和真实权限再选择技术；各文档有来源、反例与拟议 Oracle。实际原始资料定位见各章末尾；旧项目案例只复用历史证据，不重新宣称上游源码已核或已运行。新增示例均原创教学 NOT_RUN；外部规范/课程按版本与范围使用，不能转换为获权业务规则。T24c 使用实际读取的 UAX29 rev47 而非未可靠打开的 rev49；未验证任何运行时支持。T27b-1 仅 PostgreSQL 18 具体引擎行为；范式课程来源有实际阅读范围。未读或未验证部分明确保留。

T23 的 S23/D28/U5 是历史分组盘点，并非本次完成率；本轮不自动提升任何分组为全覆盖。专业正文可持续扩充，后续 T25b 其余语言/locale/时间尺度细节、T25c 其余 OS/分布式专题、T27c 其余专题、T28a/T28b 后续专题/T28c、T29 后续缺陷族、T30b 后续专业实践、T31a 后续生成及SCM专题、T32b 后续迁移专题等仍需另行有限准入。此次不增加运行时、SDK、CI、Agent、实验或产品发布。#565 HOLD、#286 DEFERRED。
