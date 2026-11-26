# T37-R4 · 通用功能与业务功能的阶段性组合入口

#634 / #563；冻结输入 c301498b79bb320e5d610bac56dc53e4d500627b。[R3 独审](https://github.com/kaicreator-mm/tally/pull/633#issuecomment-7187606731) 与 [合入收据](https://github.com/kaicreator-mm/tally/issues/632#issuecomment-7187724951) ACCEPT_SCOPED 0/0/0。50条已接受引用、47条条件导航关系、46章/265发现定位。沿用 [语义合同](semantic-reference-contract.md)、[阶段消费设计](staged-consumption-design.md)、[R3](T37-r3-state-concurrency-references.md)。本批不是全库语义化完成、通用功能框架、生产Rule或实际消费者。

## 1. 两类新增引用与真实条件导航

[17条新增引用及39条条件导航](decision-references-r4.json) 绑定10个冻结文件：8个原公开教学组合的阅读入口，9个原历史Case阅读视图（覆盖18个既有Case身份，不重新计数为新Case）。COMPOSITION_REFERENCE 是实际组合的导航视图，不是新机制Owner；与 P36-RANGE/STAGING/CACHE/EXPORT 等重叠的范围是组合合同与单项机制的不同使用目的。CASE_REFERENCE 只提供实际已登记局部源码/文档观察及原创对照；其 owner 是本次阅读问题的使用责任（例如 CSV与搜索原各有K03/K02责任），不改变原Case/概念归属或资格。

| 局部阅读视图（P37R4-前缀） | 原Case/组合与阅读范围 | 明确限制 |
|---|---|---|
| G1/G2 | T35-1 L21–32 / L33–49 | 交互配置与整批导入、持久受理刷新为两种假定合同 |
| G3/G4 | T35-3 L5–44 / L45–72 | 实际HTTP/PG18条件与受保护披露；不得复制为所有产品合同 |
| B1/B2 | T35-2 L18–59 / L60–77 | 未发货单资源预留取消、固定审批和快照报表 |
| B3/B4 | T35-4 L5–47 / L48–83 | 部分履约退款；委派与长导出分别按子问题选择 |
| CONFIG-CASE | T05-C01 L5–13 | Viper查值优先级不证明运行激活 |
| UI-LIFECYCLE-CASE | T08-C04/C05 L29–44 | Redux通知、Home Assistant局部卸载；非完整资源回收证明 |
| TRANSFER-CASE | T06-C01 L5–12 | libcurl/aria2实际局部恢复合同，不同协议不可互换 |
| CSV-SEARCH-CASE | T06-C04/C05 L29–44 | CSV字段映射或搜索结果范围；按子问题选段 |
| JOB-CASE | T18-C01/C02/C03 L13–55 | Inline/Async/Solid Queue局部对照，非真实可靠性证明 |
| REFRESH-CASE | T19-C01 L13–78 | Miniflux v2.2.3单批完成点、调用关系/失败窗；未复现潜在风险 |
| AUTH-CASE | T07-C02/C03 L13–28 | 接口与租户路由不证明全局对象权限 |
| PAY-STOCK-CASE | T07-C04/C05/C06 L29–52 | 签名、差额/账务与在途库存局部观察，非完整财务流程 |
| FORM-REPORT-CASE | T07-C07/C08/C09 L53–76 | 表单、报表与审计/通知的不同保证，按实际子问题选择 |

上述局部键只引用原已接受正文/历史案例。需实际实现某个产品或SDK特性，必须另读原来源登记及当前实际版本/调用链。原 R2 BUSINESS-CASES/HISTORY-CASES 仍拥有其阅读入口。本轮未重新获取所有上游源码，不重建Case或提升Claim/Oracle资格。39条 CONTEXT_FOR 表达各组合到已有机制/局部历史参考的可选择导航；不是必选知识列表、可执行流程、自动关系闭包或新的原48概念关系。跨投影核各自 corpusCommit 与 priorReferences 的实际Blob，不用文件名相同替代来源身份。实际任务还可从46章、原165课程ID和T22索引发现未投影知识；未命中不是不存在。

## 2. 八组公开的阶段消费选择（NOT_RUN）

以下在已接受T35教学材料上演算选择与反转，**不增加新案例或秘密测试，不将原数字当当前产品事实**。简写 G/B、机制名、Case视图对应各已登记引用键；原P36/P37/R2/R3保留前缀和权威。各行先核需求事实，再形成最小候选；实施所需确切语言/平台/外部协议由真实任务另证；QA Expected独立来自有权合同，不从Builder选知列表产生义务。阶段可回访，未列机制不是排除其它合理来源。

| 组合 | L1 必须先确定的事实 | L2/L3 候选与实际知识选择 | 独立验收、维护反转及不加载 |
|---|---|---|---|
| G1 | 无配置可打开UI；明确激活目标才可写；有界UTF-8、唯一ID、整批原子 | G1→POLICY/TRANSACTION/CONSTRAINT，纯解析与领域校验/提交分离；确有多来源才CONFIG-CASE；实际资源切换才UI-LIFECYCLE-CASE C05；CSV需C04 | 原10条，3行第三非法仍10；只有另获权部分成功合同才可12；失败切换保留旧合法激活且不能向新旧目标混写。首次启动/实际安装用INSTALLED；无远端/独立插件不加载Kafka/分布式协议 |
| G2 | 受理是否必须持久、单批完成及逐项结果、租户权限与真实源读效果、容量 | G2→LIFETIME/COMMIT；JOB-CASE和REFRESH-CASE比较同步、内存异步及持久队列；确需派生投影才设计一致基线/应用进度；只有etcd v3.6才WATCH | 旧v7不能覆盖已晋升v8；受理后重启应可查询，收到未应用不前移checkpoint；单/批204并非同一完成；实际仅短同步且不承诺独立待办时不用持久Job，允许重算丢失时才选内存队列 |
| G3 | 固定获权源表示/目标generation、跨重启、全批或部分、资源与保留政策 | RANGE+DURABILITY+STAGING+COMMIT；TRANSFER-CASE按协议，CSV-SEARCH-CASE仅C04；TRANSACTION按实际同DB，实际parser/字节合同与完整消费者路径 | 400已验证/1000总长，206合法400–999需验证600；200不追加，416不等完成；c非法时g1保留，另获权partial(a,b,d)=3/1；崩溃后核真实file/receipt和DB checkpoint；短小可重下不要求持久staging，新parser不可套旧checkpoint |
| G4 | 当前scope、授权与输出准入、数据允许一代标陈旧而非权限、实际请求身份 | CACHE/QUERY；AUTH-CASE及CSV-SEARCH-CASE C05按搜索需要，UI-LIFECYCLE-CASE C04仅共享UI；无缓存先直接权威查，真实写用COMMIT；排序实际需要才PRECISION locale段 | R1 epoch7 P,Q→撤权epoch8→R2 gen21 P-only先到，R1不得新披露Q/旧填充/覆盖UI；合法已准入不可追回；真实许可不足直接拒绝或按获权查询；无缓存/UI仍保护披露，长流另选EXPORT；不从TTL、Vary、304推授权 |
| B1 | 共同权威、未发货、付款确认/无/未知、租户、数量与取消/退款政策 | TRANSACTION/CONSTRAINT/COMMIT；实际支付才EFFECT；R2 BUSINESS-CASES与PAY-STOCK-CASE提供状态/库存/外部接收对照；保留reserve/cancel不同业务动作，纯金额仅同语义复用 | A总5、B总8；并发3/4至多一成功且活动预留和式一致，B不变；R1超时不生成R2，取消已提交与退款待核分开；付款UNKNOWN不自动取消；无付款/远端不加载退款协议 |
| B2 | 当前有权非申请人审批；报告可见范围、快照/币种及陈旧政策 | POLICY/COMMIT分别判断审批，QUERY针对报告；FORM-REPORT-CASE C07/08和AUTH-CASE局部对照；真实单步骤直接实现 | A10+20=30，B40不混入；不同币种未经授权不能合计；旧报告不授予审批；两级或实时需求才重新选择流程/读取，通知与审计若需则分别核C09及EFFECT |
| B3 | 多资源全预留、部分发货/未发货取消、同capture额度、精确金额、退款授权 | TRANSACTION/COMMIT保护共同权威和式；PRECISION仅数值；EFFECT与实际Provider查询/幂等窗口；PAY-STOCK-CASE相应段，实际消息需P36-OFFSET等的协议前提才读 | X/Y初始5/4，预留3/2，发货1/1、取消1/1、再取消X1，最终(在手,预留,可用)X(4,0,4),Y(3,0,3)；第二SKU不足无首SKU残留；R1 1500 UNKNOWN保留P，R2 1000成功后S1000/P1500，核实R1后S2500/P0净1500；额外外部500差异冻结，不造账；仅全额未发货政策不需部分模型，跨库不能套共同事务 |
| B4 | 真实主体、半开grant有效期、阶段及revision；另核导出字段/快照/交付政策 | POLICY/COMMIT与实际权威业务时间（读T25b-2有效期相关段）；任务确有等待期限才DEADLINE；AUTH-CASE及FORM-REPORT-CASE对应段；长导出EXPORT/QUERY，真实恢复才LIFETIME/STAGING相应机制 | E11:59:59已提交合法历史保留，准备后12:00才提交拒绝；E不得兼第二阶段，v7 9000改v8 11000重启；S1两行12000非13000/19000；epoch5收窄后旧制品无新准入，已准入不可追回；无长流/委派不强制这些机制，实际输出保护不足不能假称撤权即时物理回收 |

B3、B4中所有金额/时间、身份/退款权及保留均是原公开教学假设，真实Stripe API v1/实际授权和当前Provider查询语义另核；不假定幂等key可永久重试或按key直接查询。B4业务有效期不是经过时间期限，原DEADLINE只帮助真实等待/时钟选择；若需日历/业务有效期细节另读 [T25b-2](../fine-grained/T25b-2-time-clock-calendar-semantics.md) L48–65，不将monotonic当权威日期。G4与B4不同于旧P2未闭状态；原两项修正已经限定关闭，仍须按实际披露范围核保护顺序，纯导出不用缓存UI。

## 3. 每次实际消费的最小交接与反例

- L1：真实任务目标、Owner、支持/禁止动作、状态身份、完成/失败合同及未知。例：同名“导入”未说明部分成功，不能直接选G1整批或G3逐批；先ASK或提供条件方案。例：“报告”未说明scope或快照，不能以B2/B4教学权限授予访问。
- L2：实际候选、权威边界与选择理由、必要条件、拒绝候选及反转。例：任务无实际broker而有外部邮件UNKNOWN，不因G2/B3相关而自动选Kafka；实际选择Kafka才读原协议（T27c-3），不能借消息可靠性保证外部支付。
- L3/实施：交接有权确定的机制与明确源码/版本，保留实际实现、命令、变更及失败证据。公开参考未发生任何实际代码、数据库、支付、外部请求或工具动作。标题索引不能代替读原文和反例。
- QA：独立权威前后态、实际完整义务、独立Oracle与实际执行。检查拒绝后无未授权效果、迟到/重复/丢ACK时实际状态、读结果的身份与范围；所有公开示例只作纸面期望，不能作为隐藏题、实际PASS或已证工程质量。
- 维护：只在真实合法变化触发重新选知，保留历史合法状态及消费者，验证受影响部分；记录全部失败、实际变更传播/成本及回归。未知政策不得由旧Case填补，真实的错误选择、知识缺口与实现缺陷分别反馈原Owner。

## 4. 本批与下一阶段边界

原50/47已接受，新增17/39候选。累计67引用/86条件关系须实际联合旧投影核唯一键、端点/basis、实际priorReferences、冻结source path/blob/完整行区间；没有新增canonical ID、Case、Claim或Rule。原46章/265发现定位、S51/D0/U5及历史165/48/142/143/164不改，不全量重审、不虚报语义覆盖。组合例尚未实际消费或工程执行；所有Oracle NOT_RUN。下一批R5优先综合检查知识定位覆盖、语义范围充分性、同义/重复、阶段选择与负向拒绝、跨版本及真实消费准入；对已接受知识与#563原L1/L2进行有界差距核对，避免无限增加引用。是否开展真实消费者或实验须另行获权，#565 R5全部2P1/2P2 OPEN/HOLD，#286 DEFERRED，A24/T4 NOT_RUN/HOLD，不冻结生产设计/ABI或dispatch。
