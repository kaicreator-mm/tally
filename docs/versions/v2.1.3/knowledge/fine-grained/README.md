# 第三批细粒度软件工程知识 · 专业正文入口

[#586](https://github.com/kaicreator-mm/tally/issues/586)，基线 `daba091d8d9249c8a2f0100c0bae2c5464929750`。T23 已限定接受 [#584 closeout](https://github.com/kaicreator-mm/tally/issues/584#issuecomment-7166217999)，四个有限写作范围已获准入。第一组四篇已在 [#586 closeout](https://github.com/kaicreator-mm/tally/issues/586#issuecomment-7166574037) 限定接受并合入（0/0/3）；第二组及三项限定修正已获 [#588 独审](https://github.com/kaicreator-mm/tally/issues/588#issuecomment-7167193694) 限定接受（0/0/1），实际 main 内容已核实，PR #589 元数据已恢复为 closed/merged（[#588 reconciliation](https://github.com/kaicreator-mm/tally/issues/588#issuecomment-7170371776)）；第三组已在 [#591 closeout](https://github.com/kaicreator-mm/tally/issues/591#issuecomment-7169778993) 限定接受并合入（0/0/1）；第四组已在 [#592 closeout](https://github.com/kaicreator-mm/tally/issues/592#issuecomment-7170367545) 限定接受并合入（0/0/2）；第五组已在 [#594 closeout](https://github.com/kaicreator-mm/tally/issues/594#issuecomment-7170892606) 限定接受并合入（0/0/2）；第六组已在 [#596 closeout](https://github.com/kaicreator-mm/tally/issues/596#issuecomment-7171454299) 限定接受并合入（0/0/1）；第七组已在 [#598 closeout](https://github.com/kaicreator-mm/tally/issues/598#issuecomment-7172122515) 限定接受并合入（0/0/1）；第八组已在 [#600 closeout](https://github.com/kaicreator-mm/tally/issues/600#issuecomment-7172639924) 限定接受并合入（0/0/1）；第九组已在 [#602 closeout](https://github.com/kaicreator-mm/tally/issues/602#issuecomment-7172876519) 限定接受并合入（0/0/1）；第十组已在 [#604 closeout](https://github.com/kaicreator-mm/tally/issues/604#issuecomment-7173126458) 限定接受并合入（0/0/1）；第十一组已在 [#606 closeout](https://github.com/kaicreator-mm/tally/issues/606#issuecomment-7173347143) 限定接受并合入（0/0/0）；第十二组已在 [#608 closeout](https://github.com/kaicreator-mm/tally/issues/608#issuecomment-7173839589) 限定接受并合入（0/0/0）；第十三组已在 [#610 closeout](https://github.com/kaicreator-mm/tally/issues/610#issuecomment-7174340906) 限定接受并合入（0/0/0）；第十四组已在 [#612 closeout](https://github.com/kaicreator-mm/tally/issues/612#issuecomment-7174596730) 限定接受并合入（0/0/0）；第十五组已在 [#614 closeout](https://github.com/kaicreator-mm/tally/issues/614#issuecomment-7174887803) 限定接受并合入（0/0/0）；不是新的平行课程体系或合格 Source Claim。保留 K01–K11、165 历史课程 ID、48 概念关系与 T22 142 Case/143 Blob/164 关联（129 M/35 C）原始身份及资格。

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

基线 cf76c2142e2bc390ed290c82deb2f366451c1897；[第八批独审](https://github.com/kaicreator-mm/tally/issues/600#issuecomment-7172549977)。FG12 在 T27d-1 §4.1 补活动 ID 唯一、Map/heap 双射 RI、任意位置删除两向修复、证明及独立 Oracle，已由第九批独审限定关闭。

| 子任务 / 归属 | 实际新增正文 | 阅读问题及边界 |
|---|---|---|
| T27c-2 K03/K10/K02/K05 | [分布式一致性、协调与恢复边界](T27c-2-distributed-consistency-coordination-recovery.md) | DA04/05；多数交集、读/Watch/revision、投影恢复、fencing；etcd v3.6 限定，不是 Raft 实现或证明 |
| T28a-2 K05/K08/K03/K02 | [真实消费者、接口合同与安装制品验证](T28a-2-consumer-contracts-installed-artifacts.md) | QA02/03、OP02；Pact 交互/功能分工、版本矩阵、Node v22.23.3 安装边界；非实际安装或合同执行 |

两篇获限定接受，FG12 已关闭；FG13 转入第十组。教学及 Oracle NOT_RUN，无新合格 Claim/Case。

## 第十组专业正文与 FG13 修订（#604）

基线 9cabcd61289a88cd794b78ad4b6236cbbabfa9b4；[第九批独审](https://github.com/kaicreator-mm/tally/issues/602#issuecomment-7172779103)。FG13 在 T27c-2 §3.1 补实际 Watch 分片、事件与响应 revision、固定分页及 checkpoint、单范围 watch 和多 watch 水位边界，已由第十批独审限定关闭。

| 子任务 / 归属 | 实际新增正文 | 阅读问题及边界 |
|---|---|---|
| T24b-1 K01/K02/K03/K05 | [业务规则、决策表与领域约束的可反驳合同](T24b-1-business-rule-decision-contracts.md) | 规则覆盖/冲突、UNKNOWN、单位/版本、提交边界；Camunda 8 hit policy 限定，非完整 DMN 或真实业务政策 |
| T28c-1 K05/K01/K02/K10/K11 | [产品质量目标、质量保证与评价范围](T28c-1-product-quality-assurance-evaluation.md) | 质量目标、保证与控制、分母、完整过程及证据聚合；ISO25010:2023 摘要/WCAG-EM 2.0 Group Note，非认证或真实测试 |

两篇获限定接受，FG13 已关闭；FG14 转入第十一组。教学与 Oracle NOT_RUN，无新合格 Claim/Case。

## 第十一组专业正文与 FG14 修订（#606）

基线 f0915fe5500b3ff0e78adc83fdc8be09cecc614c；[第十批独审](https://github.com/kaicreator-mm/tally/issues/604#issuecomment-7173019679)。FG14 在 T28c-1 §4.1 补固定 WCAG-EM 2.0 Note 结构化/随机样本反馈、默认及常用/关键过程分支、迭代与记录、明确符合性声明限制，已由第十一批独审限定关闭。

| 子任务 / 归属 | 实际新增正文 | 阅读问题及边界 |
|---|---|---|
| T33a-1 K10/K02/K05/K06/K09 | [可靠性模型、故障分析与业务恢复目标](T33a-1-reliability-fault-analysis-recovery.md) | DA07/OP06/08；任务可靠度/可用性、串并联和共同原因、FMEA/FTA割集、RTO/RPO及真实恢复；非正式安全分析或演练 |
| T30a-1 K04/K01/K02/K11/K05/K09 | [生命周期决策、软件获取及供应商退出](T30a-1-lifecycle-acquisition-supplier-exit.md) | SY05/PR01；生命周期反馈、自研/采购/复用/外包、SLA及验收/移交/退出；NASA Ver D限定参考，非合同范本 |

两篇获限定接受，FG14 已关闭；FG01–14 均已关闭，无遗留修订。教学与 Oracle NOT_RUN，无新合格 Claim/Case。

## 第十二组专业正文（#608）

基线 069bb5472d8e3ab6b057a6df5c0280bbe87a618b；[第十一批独审](https://github.com/kaicreator-mm/tally/issues/606#issuecomment-7173294338)。FG01–14 均限定关闭；本组无历史修订。

| 子任务 / 归属 | 实际新增正文 | 阅读问题及边界 |
|---|---|---|
| T34a-1 K11/K04/K01/K02/K05/K09 | [工程经济：可比较现金流、估算与决策敏感性](T34a-1-engineering-economics-estimation-sensitivity.md) | FD06/SY05/PR01；同目标成本/估算、现值/残值、转换值与敏感性、债务阈值；UK Green Book 2026 限定方法参考，非投资或实测 |
| T32a-1 K09/K03/K02/K05/K06/K11 | [软件维护、行为保持的重构与渐进替换](T32a-1-maintenance-refactoring-replacement.md) | RE01/PR01；维护诊断、观察合同、characterization/Oracle、Branch by Abstraction、写者/Effect、弃用与清理；非实际重构/迁移 |

两篇获限定接受，FG01–14 保持关闭，无新增整改。教学与 Oracle NOT_RUN，无新合格 Claim/Case。

## 第十三组专业正文（#610）

基线 5329233b26c884c0d96dad4296ceea18cc84ab67；[第十二批独审](https://github.com/kaicreator-mm/tally/issues/608#issuecomment-7173637222)。FG01–14 均限定关闭；本组无历史修订。

| 子任务 / 归属 | 实际新增正文 | 阅读问题及边界 |
|---|---|---|
| T33b-1 K10/K03/K02/K05/K11 | [性能工程：排队模型、容量规划与负载测量](T33b-1-performance-queueing-capacity.md) | OP06/QA04；M/M/1、资源需求、容量瓶颈、开放/闭环负载与遗漏、实测Oracle；MIT OCW/k6限定参考，非实测 |
| T33c-1 K10/K06/K08/K04/K05 | [运营准备、错误预算、事故协同与服务支持](T33c-1-operations-incident-service-support.md) | OP08；准备证据、变流量燃烧率、多窗口告警、事故协同、Runbook/支持/复盘；Google SRE限定参考，非实际运营 |

两篇获限定接受，FG01–14 保持关闭，无新增整改。教学与 Oracle NOT_RUN，无新合格 Claim/Case。

## 第十四组专业正文（#612）

基线 0e495c3c3bcc5937ae7f3d0a9752a7de0956f06d；[第十三批独审](https://github.com/kaicreator-mm/tally/issues/610#issuecomment-7174234980)。FG01–14 均限定关闭；本组无历史修订。

| 子任务 / 归属 | 实际新增正文 | 阅读问题及边界 |
|---|---|---|
| T34b-1 K11/K04/K05/K07/K03 | [实证软件工程：研究设计、因果识别与可复查证据](T34b-1-empirical-research-design-causal-evidence.md) | 研究方法、2×2干预/交互、阶段化知识、污染与效度、仓库挖掘及复制；SIGSOFT/LLM指南限定，非新实验 |
| T34b-2 K11/K07/K05/K03/K09 | [工程度量、人因与独立制品评价](T34b-2-engineering-measurement-human-factors.md) | SPACE维度、构念/代理、独立评分/一致性、首次及维护联合结局、人因数据；CACM 2021/LLM指南限定，非员工考核或实际收益 |

两篇获限定接受，FG01–14 保持关闭，无新增整改。教学与 Oracle NOT_RUN，无新合格 Claim/Case。

## 第十五组专业正文（#614）

基线 2fd1b346682f46fbc2b7e0cf90fe4b223e257ab7；[第十四批独审](https://github.com/kaicreator-mm/tally/issues/612#issuecomment-7174507376)。FG01–14 均限定关闭；本组无历史修订。

| 子任务 / 归属 | 实际新增正文 | 阅读问题及边界 |
|---|---|---|
| T35-1 K02/K03，跨 K01–K11 | [通用功能族的跨领域知识组合](T35-1-general-function-knowledge-composition.md) | RE02/FM02/04；7族检索入口、G1本地配置/导入、G2异步刷新/查询/同步、阶段性交接；历史T05/T08/T19/T20限定复用 |
| T35-2 K01/K02/K03/K05，跨 K04/K06–K11 | [业务功能族的跨领域知识组合](T35-2-business-function-knowledge-composition.md) | RE03/FM02/04；6族入口、B1订单/库存/退款、B2审批/报表、抽象候选与独立回归；历史T07/T17限定复用 |

两篇获限定接受，FG01–14 保持关闭。原创公开教学组合 G1/G2/B1/B2 与 Oracle NOT_RUN，既有142 Case等原身份不增加，不作为独立隐藏实验题。

## T23-R3 综合盘点（#616，分类已限定接受）

35篇已限定接受并合入。以冻结 main ded5b89f1c29081f53d6b0b60fe35c4f13a1e6e6 对照 [56组重新盘点](../T23-R3-detail-reassessment.md) 与 [剩余深度分流](../T23-R3-residual-depth-plan.md)，实际逐行判断为 S40/D11/U5；历史 S23/D28/U5 不回填。S 仅为限定问题充分；D 先定向检索再补深；U 含3专项、1暂缓、1排除。非全域完整度、非新合格 Claim/Case、非 LLM 消费成功证明。R4-A–G 是后续候选范围，不自动执行，T36 另行整合。#617 已合入 827bf0a2783ce5c0aedc0e599ebeb0ed64ad4e33（0/0/1），[收尾](https://github.com/kaicreator-mm/tally/issues/616#issuecomment-7175484402)。R3-RV-01 已在 [#616 closeout](https://github.com/kaicreator-mm/tally/issues/616#issuecomment-7175668874) 限定关闭，#616 已关闭；C/G 仍须独立的具体范围准入。原 S40/D11/U5 保持，历史 161/63 计数不回填。

## R4-A 架构组合及 R3-RV-01 复用修正（#618）

基线 827bf0a2783ce5c0aedc0e599ebeb0ed64ad4e33；[R3 独审](https://github.com/kaicreator-mm/tally/pull/617#issuecomment-7175399512)。R4-A 仅 FM-02 的架构风格组合目标，R3-RV-01 仅 DA-07/RE-02/RE-03 复用导航及残余准入；PR-04 表述澄清；分类保持不变。

| 子任务 / 归属 | 实际增量 | 阅读问题及边界 |
|---|---|---|
| R4-A / T26a-2 K02/K03/K01/K05/K10/K11 | [架构风格的组合、替代与边界变化](T26a-2-architecture-style-composition.md) | 同一 D0 合同；模块/处理流/运行连接/部署与扩展四维，五种候选及并发/故障/维护反转；既有案例复用，教学 NOT_RUN |
| R3-RV-01 | [56组台账补读](../T23-R3-detail-reassessment.md)；[C/G 修订计划](../T23-R3-residual-depth-plan.md) | T06/DE/T02/T03 精确复用，已充分子目标与 delta 分开，待独审；不提前关闭 P2 或写作 C/G |

R4-A 与 R3-RV-01 已在 [#618 closeout](https://github.com/kaicreator-mm/tally/issues/618#issuecomment-7175669731) 限定接受并合入（0/0/1 新 P2）。36篇已接受；FM-02 限定 D→S，当前派生 S41/D10/U5，历史 R3 S40/D11/U5 不回填。P2-R4A-01 已在 [#618 closeout](https://github.com/kaicreator-mm/tally/issues/618#issuecomment-7176228523) 限定关闭，#618 已关闭。FG01–14 保持关闭，历史资格不变。

## R4-B 可靠消息及协议协商（#620）

基线 d2fb562aa0ffcfced73080ed9c7b17d799d6b2d9；[R4-A 独审](https://github.com/kaicreator-mm/tally/pull/619#issuecomment-7175631747)。R4-B 限 DA-05/FM-07 扩展目标；同时 P2-R4A-01 两项独立 Oracle 修正。

| 子任务 / 归属 | 实际增量 | 阅读问题及边界 |
|---|---|---|
| DA-05 / T27c-3 K03/K02/K09/K05/K10 | [可靠消息、分区进度、Outbox 与消费事务](T27c-3-reliable-messaging-outbox-offsets.md) | Kafka 4.1 / Java client 4.1.2，offset 前缀、自动重试与应用重发、outbox/inbox 崩溃窗、Kafka 内事务 vs 外部效果 |
| FM-07 / T25c-3 K03/K02/K09/K01/K05 | [协议版本、能力协商与混合消费者](T25c-3-protocol-capability-negotiation.md) | Kafka ApiVersions、连接范围、认证与实际能力、业务 schema/未知必需字段/中转/离线重放 |
| P2-R4A-01 | T26a-2 两条限定 Oracle | 文档版本相同但旧执行许可、prepare 后权限撤销，真实权威前后状态；待独审关闭 |

R4-B 两篇已在 [#620 closeout](https://github.com/kaicreator-mm/tally/issues/620#issuecomment-7176229334) 限定接受并合入 d318c8b2e968355ad2ccc83133bc9a6c8782e775（0/0/1 新 P2）；DA-05/FM-07 限定 D→S，38篇已接受。原 R3 历史统计及身份不变；当前派生 S43/D8/U5。P2-R4B-01 已在 [#620 closeout](https://github.com/kaicreator-mm/tally/issues/620#issuecomment-7176664867) 限定关闭，#620 已关闭。所有教学及 Oracle NOT_RUN，无新合格 Claim/Case。C/G 复用整改已关闭但写作另准入。

## R4-C/D/E/F 批量专业深度建设（#622）

输入 d318c8b2e968355ad2ccc83133bc9a6c8782e775；[R4B 独审](https://github.com/kaicreator-mm/tally/pull/621#issuecomment-7176105161)，[关闭](https://github.com/kaicreator-mm/tally/issues/620#issuecomment-7176229334)。Owner 要求批量推进；按 R3-RV-01 已关闭后的实际复用边界，六个限定知识目标和 P2 分别裁定，同批有限独审。原 R3 盘点/计划为历史记录；其中 OPEN 措辞的实际处置以上述收据及 [#616 closeout](https://github.com/kaicreator-mm/tally/issues/616#issuecomment-7175668874) 为准，不回填历史。

| 主组 / 归属 | 新增正文 | 阅读问题与证据边界 |
|---|---|---|
| DA-07 / K02/K09/K10/K01/K05 | [T32b-2 数据生命周期、血缘、删除传播与离线冲突](T32b-2-data-lineage-conflict-tombstones.md) | CouchDB 3.5 局部冲突机制、OpenLineage object model；中心权威三方合并、generation/tombstone、恢复/质量证据；教学 |
| OP-02 / K08/K03/K09/K05 | [T31a-3 依赖约束、解析图与真实安装制品](T31a-3-dependency-resolution-installed-package.md) | npm v11 peer/optional/workspace/lock/pack；同宿主约束和真实隔离消费者 |
| OP-04 / K08/K09/K10/K05 | [T31a-4 渐进部署、版本路由、Feature Flag 与恢复](T31a-4-progressive-delivery-routing-flags.md) | Kubernetes v1.34/OpenFeature 在线规范限定，制品/路由/功能/数据，比例、停止、回退 |
| QA-06 / K06/K05/K03/K10 | [T29a-2 可区分的诊断探针与历史缺陷定位](T29a-2-diagnostic-probes-and-bisection.md) | 原 CPython 修复/测试差异复用，Git bisect 2.52 与 Go1.25 pprof，竞争根因与具体 probe |
| FM-05 / K03/K05/K02 | [T25b-3 跨语言值、所有权、泛型约束与失败语义](T25b-3-cross-language-ownership-generics.md) | Go spec 页面标 go1.27/Rust Book，T16 真实语义复用，alias/borrow/constraint/error/退出 |
| FD-02 / K03/K01/K05/K02 | [T25b-4 精确数值运算、Locale 排序与双向文本](T25b-4-exact-numbers-locale-bidi.md) | Python3.13.18 decimal、Unicode17 UTS10 rev53/UAX9 rev51；中间舍入、排序与身份、逻辑与显示 |
| P2-R4B-01 | T27c-3 §4 及独立负例 | read_committed 输入、禁止独立 offset commit、groupMetadata、commit 超时/中断、上游 abort 与错进度反例；待独审 |

R4-C/D/E/F 六篇及 P2-R4B-01 在 [#622 closeout](https://github.com/kaicreator-mm/tally/issues/622#issuecomment-7176665795) 限定接受并合入 5ac801aabca792dbcc468de46a2cfcf66ed8ef24（0/0/0）；44篇已接受，六个限定 D→S，当前派生 S49/D2/U5，原 R3 历史不回填。#620/#622 已关闭。所有教学与 Oracle NOT_RUN，无新合格 Claim/Case、实际部署、CI、Agent 或实验。RE02/RE03 为剩余 D；下一批按复用校准后的具体范围推进。T36 另行整合及消费设计。

## R4-G 复杂通用与业务功能组合（#624）

基线 5ac801aabca792dbcc468de46a2cfcf66ed8ef24；[R4-C/D/E/F 独审](https://github.com/kaicreator-mm/tally/pull/623#issuecomment-7176597497)。R3-RV-01 已限定关闭，按 [修订残余计划](../T23-R3-residual-depth-plan.md) 的 G3/G4/B3/B4 准入。现有 G1/G2/B1/B2、T06、DE001–008、T02-C06、T03-C03/C05、T07/T17/T19/T20 等保留语义归属及证据资格；本批仅深化选定组合。

| 子目标 / 归属 | 实际新增正文 | 阅读问题与证据边界 |
|---|---|---|
| RE-02 / G3、G4 K02/K03/K01/K05/K09/K10 | [T35-3 复杂通用功能组合](T35-3-complex-general-function-composition.md) | RFC9110 Range/If-Range、PG18 COPY、持久片段与 staging/checkpoint/原子发布 vs 部分成功；RFC9111 cache 与权限epoch/数据generation/UI sequence、撤权与迟到结果 |
| RE-03 / B3、B4 K01/K02/K03/K05/K09/K10 | [T35-4 复杂业务功能组合](T35-4-complex-business-function-composition.md) | 多SKU原子预留/部分履约与取消、Stripe API v1 部分退款/幂等窗口/UNKNOWN及对账；委派时段/实际actor/版本、长导出同快照与逐块披露准入 |

R4-G 四个具体子目标及 RE-02/RE-03 限定 D→S 已在 [#624 closeout](https://github.com/kaicreator-mm/tally/issues/624#issuecomment-7176986822) 限定接受并合入 7f5160b3ec4b29138d4f21e9bcc6ff21459090a3（0/0/0）。46篇已接受，当前派生 S51/D0/U5，原 R3 S40/D11/U5 历史不回填；非整个功能族完成。跨进程关闭/安全日志、其它下载协议/平台、预约冲突、跨支付方/结算及未选政策/产品变体保持 OPEN/UNKNOWN，专项 U 不晋升。全部为公开原创教学及 Oracle NOT_RUN，未运行或增加 qualified Claim/Case；不作为隐藏实验题，不自动产生业务 Rule。T36 已另行准入设计候选，见下节，仍未冻结。

## T36 语义整合与阶段消费设计（#626）

输入 7f5160b3ec4b29138d4f21e9bcc6ff21459090a3。沿用原165 ID、48关系、T22案例/来源与 #564 投影；46篇入口保留原语义拥有者。本批仅有界设计，非整库语义单元化。

| 阅读入口 | 作用 |
|---|---|
| [T36-A/B 语义引用与关系合同](../consumption/semantic-reference-contract.md) | 区分语义身份/定位/Owner/资格，关系方向及适用性，禁止候选自动晋升 Rule |
| [实际定位 pilot](../consumption/pilot-references.json) | 13条选定引用、12条候选导航关系、10个原文文件；3原ID+10局部导航键，非新全局图谱或 SDK ABI |
| [T36-C 阶段消费设计](../consumption/staged-consumption-design.md) | 发现、条件选择、事实与交接、预算/不完整、停止；原专业阶段及 Task 权威不变 |
| [T36-D 公开推演与独立评价](../consumption/evaluation-and-worked-traces.md) | 3组公开合成任务、8种变形、6层观察；全部 NOT_RUN，非隐藏题/独立执行报告 |

#565 R5 全部2P1/2P2 OPEN/HOLD，#286 DEFERRED，A24/T4 NOT_RUN/HOLD；不 dispatch 实施、Agent 或实验。T36 设计已在 [#626 收据](https://github.com/kaicreator-mm/tally/issues/626#issuecomment-7185974134) 限定接受并合入 f0358f8454a6f57b17e741ae5c03db760a4ee15e（0/0/2）；当时 P2-T36-01/02 OPEN_NONBLOCKING；后经 [#626 关闭收据](https://github.com/kaicreator-mm/tally/issues/626#issuecomment-7186888329) 限定关闭，#626 已关闭。设计接受不冻结生产 ABI，原资格/计数不变。

## T37-R1 · 发现入口及有限决策引用扩展（#628）

输入 f0358f8454a6f57b17e741ae5c03db760a4ee15e；[T36 独审](https://github.com/kaicreator-mm/tally/pull/627#issuecomment-7185602142)。[本批合同、范围与公开推演](../consumption/T37-r1-reference-expansion.md)：发现投影 [1–12](../consumption/chapter-discovery-01.json)、[13–24](../consumption/chapter-discovery-02.json)、[25–36](../consumption/chapter-discovery-03.json)、[37–46](../consumption/chapter-discovery-04.json) 覆盖46篇已接受正文及265个标题定位；逐文件实际获取并机械提取，不能把标题当自足语义。新增 [11条实质决策引用](../consumption/decision-references-r1.json)，每个Kxx一条限定入口、7条候选导航关系，均已在 [#628 收尾](https://github.com/kaicreator-mm/tally/issues/628#issuecomment-7186889556) 限定接受并合入 33406ba1f1081dd57593a0e645b81885fff55b3b（0/0/0），非新的概念/Claim/Case。原13条pilot引用保留，P2-T36-01/02 的限定修正见 [pilot](../consumption/pilot-references.json)、[阶段设计](../consumption/staged-consumption-design.md) 及 [公开反例](../consumption/evaluation-and-worked-traces.md)，已独立限定关闭。24条已接受引用不是全库语义覆盖；19条导航边非原48概念关系增量。无模型执行或产品接线，#565 HOLD、#286 DEFERRED、A24/T4 NOT_RUN/HOLD。

## T37-R2 · 工程抽象与合法变化选择（#630）

输入 33406ba1f1081dd57593a0e645b81885fff55b3b。[R2范围及公开推演](../consumption/T37-r2-abstraction-references.md)、[12条新增决策引用及12条导航边](../consumption/decision-references-r2.json)：结构坏味道、输入所有权、依赖/授权/状态/版本四边界、ADR、内聚及替换、错误/资源、变体、架构组合、包依赖、维护替换及 T17/T20 已有Case局部观察。原DP与所有权不变，候选研究反馈/历史Case资格不晋升；24条已接受+12条待独审，19条已接受+12条候选导航关系，非原48概念关系增量。四类公开推演 NOT_RUN；无实际消费者、模型、生产Rule/ABI或发布。#565 HOLD、#286 DEFERRED、A24/T4 NOT_RUN/HOLD。

## 阅读路径与边界

以任务检索：需求冲突/验收 → T24a；表单/焦点/输入/用户任务 → T24c；抽象/内聚/表示/alias/frame → T27a；外键/范式/NULL/join/isolation → T27b-1；绑定/数值/Unicode → T25b-1；跨语言所有权/泛型/错误 → T25b-3；精确数值/locale/BiDi → T25b-4；文件/流/持久性 → T27b-2；竞态/锁/准入/取消 → T27c-1；测试选择/模型/隔离 → T28a-1；控制流/抽象解释/形式验证 → T28b-1；真实缺陷/修复/跨组件回归 → T29a-1；bisect/heap/锁/trace 诊断 → T29a-2；进程/IPC/映射 → T25c-1；网络分帧/ACK/HTTP 条件 → T25c-2；API版本/能力协商/混合消费者 → T25c-3；时钟/时区/期限/生效 → T25b-2；架构场景/权衡/产品线 → T26a-1；架构风格组合/流水线/事件/插件/服务 → T26a-2；专业责任/写作/评审/交接 → T30b-1；生成/ABI/加载 → T31a-1；SCM/合并/基线 → T31a-2；依赖解析/真实包 → T31a-3；部署/路由/flag → T31a-4；迁移/混合版本/回退 → T32b-1；数据血缘/冲突/墓碑/恢复 → T32b-2；逻辑/关系/图/归纳 → T25a-1；数组/哈希/堆/图成本 → T27d-1；概率/测量/统计推断 → T25d-1；多数/分区/watch/fencing → T27c-2；消息投递/offset/outbox/inbox → T27c-3；真实消费者/合同/已安装制品 → T28a-2；规则表/冲突/版本/业务约束 → T24b-1；质量目标/保证/完整过程/评价 → T28c-1；可靠性/共同故障/割集/RTO/RPO → T33a-1；自研/采购/供应商验收/退出 → T30a-1；成本现值/估算/转换值 → T34a-1；维护/行为保持/替换/弃用 → T32a-1；排队/容量/负载模型/测量遗漏 → T33b-1；运营准备/燃烧率/事故协同/支持 → T33c-1；研究方法/因果/阶段化输入/污染 → T34b-1；工程度量/独立评价/维护及人因 → T34b-2；配置/导入/任务/查询/同步组合 → T35-1；复杂下载导入/权限缓存/异步交错 → T35-3；订单/库存/退款/审批/报表组合 → T35-2；多资源/部分效果/委派审批/长导出 → T35-4。需要先选业务目标和真实权限再选择技术；各文档有来源、反例与拟议 Oracle。实际原始资料定位见各章末尾；旧项目案例只复用历史证据，不重新宣称上游源码已核或已运行。新增示例均原创教学 NOT_RUN；外部规范/课程按版本与范围使用，不能转换为获权业务规则。T24c 使用实际读取的 UAX29 rev47 而非未可靠打开的 rev49；未验证任何运行时支持。T27b-1 仅 PostgreSQL 18 具体引擎行为；范式课程来源有实际阅读范围。未读或未验证部分明确保留。

T23 的 S23/D28/U5 是历史分组盘点，并非本次完成率；本轮不自动提升任何分组为全覆盖。专业正文可持续扩充，后续 T25b 其余语言/locale/时间尺度细节、T25c 其余 OS/分布式专题、T27c 其余专题、T28a/T28b/T28c 后续专题、T29 后续缺陷族、T30a/T30b 后续专业实践、T31a 后续生成及SCM专题、T32a/b 后续维护及迁移专题、T33a/b/c 后续性能/运营/可靠性专题、T34a/b 后续经济及实证专题等仍需另行有限准入。此次不增加运行时、SDK、CI、Agent、实验或产品发布。#565 HOLD、#286 DEFERRED。
