# 第三批细粒度软件工程知识 · 专业正文入口

[#586](https://github.com/kaicreator-mm/tally/issues/586)，基线 `daba091d8d9249c8a2f0100c0bae2c5464929750`。T23 已限定接受 [#584 closeout](https://github.com/kaicreator-mm/tally/issues/584#issuecomment-7166217999)，四个有限写作范围已获准入。第一组四篇已在 [#586 closeout](https://github.com/kaicreator-mm/tally/issues/586#issuecomment-7166574037) 限定接受并合入（0/0/3）；第二组及三项限定修正已获 [#588 独审](https://github.com/kaicreator-mm/tally/issues/588#issuecomment-7167193694) 限定接受（0/0/1），实际 main 内容已核实，PR #589 元数据仍待一致性恢复；第三组为新增候选，待独审；不是新的平行课程体系或合格 Source Claim。保留 K01–K11、165 历史课程 ID、48 概念关系与 T22 142 Case/143 Blob/164 关联（129 M/35 C）原始身份及资格。

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

## 第三组专业正文及流协议修正（#563/#588 延续）

基线 d382b0091fee6989777421d5ce48280f11624a2f；[候选原稿](https://github.com/kaicreator-mm/tally/issues/563#issuecomment-7167352369)，[第二批独审](https://github.com/kaicreator-mm/tally/issues/588#issuecomment-7167193694)，[实际 main 内容核验](https://github.com/kaicreator-mm/tally/issues/588#issuecomment-7167297467)。本批限定写入 PR #590；未创建新的独立 Task（接口持续失败），使用 #563 任务记录与 #588 延续追踪。P2-FG-04 已补入 T27b-2 §3.1（流背压/终止、严格增量解码、取消与 Oracle），待独审，不提前关闭。

| 子任务 / 归属 | 实际新增正文 | 阅读问题及边界 |
|---|---|---|
| T27c-1 K03/K02/K10/K05 | [并发、异步、状态所有权和任务生命周期](T27c-1-concurrency-task-lifetime.md) | FM06/DA03/05/OP06；同步关系、锁与状态、任务准入、容量、取消收敛；Go 内存模型/Python 3.11 结构化并发的限定例，不涵盖全部语言或分布式共识 |
| T28a-1 K05/K01/K03/K06 | [测试选择、状态模型、生成式验证与隔离](T28a-1-test-design-stateful-isolation.md) | QA02/03；缺陷模型、边界/状态、性质/变形/差分、Fuzz、Fixture 与可重复性；非完整质量体系/真实缺陷库 |

教学与 Oracle 全部 NOT_RUN；新正文未独审，不增加合格 Claim。

## 阅读路径与边界

以任务检索：需求冲突/验收 → T24a；表单/焦点/输入/用户任务 → T24c；抽象/内聚/表示/alias/frame → T27a；外键/范式/NULL/join/isolation → T27b-1；绑定/数值/Unicode → T25b-1；文件/流/持久性 → T27b-2；竞态/锁/准入/取消 → T27c-1；测试选择/模型/隔离 → T28a-1。需要先选业务目标和真实权限再选择技术；各文档有来源、反例与拟议 Oracle。实际原始资料定位见各章末尾；旧项目案例只复用历史证据，不重新宣称上游源码已核或已运行。新增示例均原创教学 NOT_RUN；外部规范/课程按版本与范围使用，不能转换为获权业务规则。T24c 使用实际读取的 UAX29 rev47 而非未可靠打开的 rev49；未验证任何运行时支持。T27b-1 仅 PostgreSQL 18 具体引擎行为；范式课程来源有实际阅读范围。未读或未验证部分明确保留。

T23 的 S23/D28/U5 是历史分组盘点，并非本次完成率；本轮不自动提升任何分组为全覆盖。专业正文可持续扩充，后续 T25b 其余语言/时间/locale 细节、T27c 其余专题、T28a-2/T28b/c、T29、T30b 等仍需另行有限准入。此次不增加运行时、SDK、CI、Agent、实验或产品发布。#565 HOLD、#286 DEFERRED。
