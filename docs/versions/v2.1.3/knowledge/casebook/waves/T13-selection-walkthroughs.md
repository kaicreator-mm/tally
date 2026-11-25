# T13 · Agent 选知演算（8 组实际只读阅读）

[#567](https://github.com/kaicreator-mm/tally/issues/567)；修正正文基线 41dad33035beab0eef1a1fffe3796d9fe71394db；目录读取基线 462133c8f8fcc48e77cf452084fd55c46d5d1919。原八组演算保持历史身份，本轮按 T14 两份独立意见复核并修正知识选择与验证意图。执行者为本轮知识整理者，在当前会话实际读取全目录与以下候选正文、T01/T10 理论索引，再进行人工工程推演。八组需求均为明确的教学约束，不是当前用户项目授权、实际 Agent 自动检索、LLM 因果试验、运行验证或最优性证明。结果为 READ_ONLY_SELECTION_COMPLETED / DESIGN_CANDIDATE；所有行为 Oracle NOT_RUN。每次只读有限相关知识，不要求全库注入。未执行任何外部程序或新增实现。

## 阅读方法（不是运行时算法）

先区分问题的软件类型、功能、抽象粒度与真实约束；用目录选候选；读原案例具体合同、源资格、限制和反例，而非只依据标题。需要专业理论时查原 Kxx 与 T01/T10；比较适用性、拒绝更复杂或不等价方案、说明何时反转；给出真正可判定的下一步验证。未知真实产品事实时保留条件候选。以下每组均有不采用方案及独立验证意图。

## W01 · 固定报表步骤是否需要工作流引擎

- **情境（教学假设）：** 本地 CLI 单次导出，读取→聚合→写文件；没有跨进程恢复要求，失败后不发布不完整输出。
- **入口轴：** 任务调度/恢复/复杂度，L2–L4。
- **实际读取案例：** [T12-C03](https://github.com/kaicreator-mm/tally/blob/41dad33035beab0eef1a1fffe3796d9fe71394db/docs/versions/v2.1.3/knowledge/casebook/waves/T12-comparative-design-cases.md#L23)、[T04-C02](https://github.com/kaicreator-mm/tally/blob/41dad33035beab0eef1a1fffe3796d9fe71394db/docs/versions/v2.1.3/knowledge/casebook/waves/T04-intelligence-and-infrastructure-architecture.md#L15)、[T05-C08](https://github.com/kaicreator-mm/tally/blob/41dad33035beab0eef1a1fffe3796d9fe71394db/docs/versions/v2.1.3/knowledge/casebook/waves/T05-common-function-abstractions.md#L68)。
- **专业知识：** T10 C3/C4；K04.F1.3、K04.F3.1（流程条件启发）、K03.F3.2（取消合同）（[T01](T01-theory-ab.md)、[T10](T10-theory-cd.md)、[原知识反向索引](T13-knowledge-and-coverage.md)）。
- **源观察到的区分（来自案例正文）：** T04-C02 的状态图具有局部更新、checkpoint 和中断能力，但当前没有恢复需求；T05-C08 明确取消信号不等待完成；T12-C03 提供直接调用与持久 DAG 的条件差异。
- **选择及拒绝（本轮推演）：** 采用直接顺序组合，写入临时文件后按产品合同提交；失败保留可行动错误，取消时等待所拥有资源清理。拒绝持久状态图和通用调度平台，因尚无实际需求。
- **反转及尚缺事实：** 若真实批量独立导出且共享容量受限，有限 worklist；若授权跨进程恢复再考虑持久 DAG，须先明确外部效果。
- **验证意图（NOT_RUN）：** 读取失败不进入写入；计算失败不发布；取消后检查实际终态；用户要求原子替换时须核实文件系统合同。

## W02 · 动态资源初始化与共享连接的清理边界

- **情境（教学假设）：** 桌面导入功能由用户选择任意数量输入文件，借用应用共享数据库连接池；第二个文件打开失败，其他窗口仍需数据库。
- **入口轴：** 资源/所有权/部分失败，L3–L5。
- **实际读取案例：** [T12-C06](https://github.com/kaicreator-mm/tally/blob/41dad33035beab0eef1a1fffe3796d9fe71394db/docs/versions/v2.1.3/knowledge/casebook/waves/T12-comparative-design-cases.md#L50)、[T05-C09](https://github.com/kaicreator-mm/tally/blob/41dad33035beab0eef1a1fffe3796d9fe71394db/docs/versions/v2.1.3/knowledge/casebook/waves/T05-common-function-abstractions.md#L77)、[T05-C10](https://github.com/kaicreator-mm/tally/blob/41dad33035beab0eef1a1fffe3796d9fe71394db/docs/versions/v2.1.3/knowledge/casebook/waves/T05-common-function-abstractions.md#L86)、[T08-C05](https://github.com/kaicreator-mm/tally/blob/41dad33035beab0eef1a1fffe3796d9fe71394db/docs/versions/v2.1.3/knowledge/casebook/waves/T08-platform-functions.md#L37)、[T09-C08](https://github.com/kaicreator-mm/tally/blob/41dad33035beab0eef1a1fffe3796d9fe71394db/docs/versions/v2.1.3/knowledge/casebook/waves/T09-code-abstraction-cases.md#L68)。
- **专业知识：** T10 C1/C2；K03.F4.2（资源关闭）、K03.F3.1（激活状态）（[T01](T01-theory-ab.md)、[T10](T10-theory-cd.md)、[原知识反向索引](T13-knowledge-and-coverage.md)）。
- **源观察到的区分（来自案例正文）：** ExitStack 动态登记、Go Closer 重复行为不普遍保证；对象池可丢弃项而连接池管理稀缺连接；Home Assistant 配置与加载状态独立；Once 的完成屏障不适合失败重试。
- **选择及拒绝（本轮推演）：** 动态输入清理登记在各次成功取得后；获取器内部部分成功但未交付，由获取器清理或明确移交句柄和责任。固定输出用作用域；借用 Conn/Rows 按合同 Close/归还，应用 DB pool 不关闭。若启动失败可重试，显式状态和实际资源记录，不套用 Once 永久完成或全局 Exit 强制去重。
- **反转及尚缺事实：** 如果只有固定单资源且无共享，直接语言作用域即可；真实多消费者独立生命周期才加 Owner。
- **验证意图（NOT_RUN）：** A 成功/B 失败须释放 A；清理错误保留；其他窗口仍能访问 pool；复试前确认上次实际资源状态。

## W03 · 支付通知丢失时如何抽象业务状态

- **情境（教学假设）：** 电商订单需实际支付确认才履约，webhook 可能重复或迟到；支付请求响应可能丢失。
- **入口轴：** 支付/状态/外部副作用，L3–L5。
- **实际读取案例：** [T03-C05](https://github.com/kaicreator-mm/tally/blob/41dad33035beab0eef1a1fffe3796d9fe71394db/docs/versions/v2.1.3/knowledge/casebook/waves/T03-business-and-collaboration-architecture.md#L45)、[T07-C04](https://github.com/kaicreator-mm/tally/blob/41dad33035beab0eef1a1fffe3796d9fe71394db/docs/versions/v2.1.3/knowledge/casebook/waves/T07-business-functions.md#L29)、[T03-C06](https://github.com/kaicreator-mm/tally/blob/41dad33035beab0eef1a1fffe3796d9fe71394db/docs/versions/v2.1.3/knowledge/casebook/waves/T03-business-and-collaboration-architecture.md#L55)、[T09-C10](https://github.com/kaicreator-mm/tally/blob/41dad33035beab0eef1a1fffe3796d9fe71394db/docs/versions/v2.1.3/knowledge/casebook/waves/T09-code-abstraction-cases.md#L86)、[T12-C10](https://github.com/kaicreator-mm/tally/blob/41dad33035beab0eef1a1fffe3796d9fe71394db/docs/versions/v2.1.3/knowledge/casebook/waves/T12-comparative-design-cases.md#L86)。
- **专业知识：** T01 B1/B2、T10 C3；K01.F3.2（业务状态）、K02.F3.2（事件与一致性）、K10.F3.2（恢复）（[T01](T01-theory-ab.md)、[T10](T10-theory-cd.md)、[原知识反向索引](T13-knowledge-and-coverage.md)）。
- **源观察到的区分（来自案例正文）：** Saleor 分列订单/授权/扣款状态，不证明任意组合均合法；Stripe webhook 先验签不证明业务记账；Mattermost 业务变更后发通知；MultiWriter 前一 sink 可能已写。
- **选择及拒绝（本轮推演）：** 订单 Owner 保留支付待确认与履约义务；签名和事件身份独立核实，按同一受权支付身份查询实际效果。确认已生效则读回不重复写；确认未生效、无在途且仍获权时可开始新 attempt；未知/在途效果仅在已证实身份、版本、payload、去重范围和有效期、并发和授权合同足以防重复时重放，否则对账、受权人工处置或停止。重试预算不是安全证明。拒绝用 timeout/事件接收/单 success 断言扣款或未扣款；可靠通知不是付款成立证据。
- **反转及尚缺事实：** 无外部支付的本地单步业务可单一状态；真实要求通知可靠才选择持久投递意图。支付对账 API、账务授权未给定，不设计具体重试/财务政策。
- **验证意图（NOT_RUN）：** 重复/迟到事件、错订单/金额、响应丢失、独立履约失败；实际远端状态无法核实则 UNKNOWN，不重复收费。

## W04 · 权限受限查询是否可以报告没有记录

- **情境（教学假设）：** 一个多租户搜索接口限制本轮最多读取一定文档、返回 top-k，查询可能超时；调用者请求判断整个授权范围内是否存在记录。
- **入口轴：** 搜索/授权/完整性，L3–L5。
- **实际读取案例：** [T12-C05](https://github.com/kaicreator-mm/tally/blob/41dad33035beab0eef1a1fffe3796d9fe71394db/docs/versions/v2.1.3/knowledge/casebook/waves/T12-comparative-design-cases.md#L41)、[T04-C03](https://github.com/kaicreator-mm/tally/blob/41dad33035beab0eef1a1fffe3796d9fe71394db/docs/versions/v2.1.3/knowledge/casebook/waves/T04-intelligence-and-infrastructure-architecture.md#L25)、[T04-C04](https://github.com/kaicreator-mm/tally/blob/41dad33035beab0eef1a1fffe3796d9fe71394db/docs/versions/v2.1.3/knowledge/casebook/waves/T04-intelligence-and-infrastructure-architecture.md#L35)、[T06-C05](https://github.com/kaicreator-mm/tally/blob/41dad33035beab0eef1a1fffe3796d9fe71394db/docs/versions/v2.1.3/knowledge/casebook/waves/T06-data-file-network-functions.md#L37)、[T07-C02](https://github.com/kaicreator-mm/tally/blob/41dad33035beab0eef1a1fffe3796d9fe71394db/docs/versions/v2.1.3/knowledge/casebook/waves/T07-business-functions.md#L13)。
- **专业知识：** T01 B1/B2、T10 C4；K05.F1.2（验证意图）、K02.F2.2（结果合同）、K02.F4.3（授权威胁）、K03.F1.2（索引）（[T01](T01-theory-ab.md)、[T10](T10-theory-cd.md)、[原知识反向索引](T13-knowledge-and-coverage.md)）。
- **源观察到的区分（来自案例正文）：** Meilisearch 将租户规则加入查询 filter，申请许可；Lucene top-n 和 collector 是不同合同，有 timeout/partial 字段；Django 权限接口带 user/perm/obj 但不保证对象级实现。
- **选择及拒绝（本轮推演）：** 在实际授权范围内过滤并区分返回集合、匹配集合、扫描/版本及截断。未完成扫描且无命中返回范围未知/不完整，不声称全域不存在；真正小范围可直接有界扫描。拒绝只在最终结果过滤越权项或 top-k 为空当全集空。
- **反转及尚缺事实：** 明确全文质量、容量、服务独立边界才选索引/独立服务；无多租户或排名需求时减复杂度。
- **验证意图（NOT_RUN）：** 越权文档不得被返回、披露或占据授权 top-k；正例为仅在受权集合中收集合法命中；反例为先全域 top-k 再过滤，可能丢掉本应返回的合法命中。共享 BM25 全局统计可能因其他租户文档改变而改变合法分数；若产品要求强非干扰，另定义排名统计、时延等观察及隔离条件，当前入口不能证明。k=0、部分结果、超时及截断前未命中均不等于没有；旧索引/排序并列/权限变化；搜索引擎全链安全与性能仍需实际验证。

## W05 · 新旧公共 API 能否共用一个验证入口

- **情境（教学假设）：** 两个真实支持的旧消费者接受非字典输入转换，新接口要求严格类型，尚未获权撤销旧支持。
- **入口轴：** 兼容/API/合法输入域，L5/L6。
- **实际读取案例：** [T12-C11](https://github.com/kaicreator-mm/tally/blob/41dad33035beab0eef1a1fffe3796d9fe71394db/docs/versions/v2.1.3/knowledge/casebook/waves/T12-comparative-design-cases.md#L95)、[T11-C05](https://github.com/kaicreator-mm/tally/blob/41dad33035beab0eef1a1fffe3796d9fe71394db/docs/versions/v2.1.3/knowledge/casebook/waves/T11-real-evolution-cases.md#L45)、[T11-C06](https://github.com/kaicreator-mm/tally/blob/41dad33035beab0eef1a1fffe3796d9fe71394db/docs/versions/v2.1.3/knowledge/casebook/waves/T11-real-evolution-cases.md#L55)、[T05-C04](https://github.com/kaicreator-mm/tally/blob/41dad33035beab0eef1a1fffe3796d9fe71394db/docs/versions/v2.1.3/knowledge/casebook/waves/T05-common-function-abstractions.md#L32)。
- **专业知识：** T01 A2、T10 D1/D2；K09.F3.1（版本兼容）、K09.F4.1（真实受支持消费者，条件）、K03.F2.2（输入验证）（[T01](T01-theory-ab.md)、[T10](T10-theory-cd.md)、[原知识反向索引](T13-knowledge-and-coverage.md)）。
- **源观察到的区分（来自案例正文）：** Pydantic 的旧 parse_obj 在 V2 转发新入口且迁移指南列相等语义变化；Go ServeMux 隔离旧路由，用启动级开关；Go 流解码与结构政策分离。
- **选择及拒绝（本轮推演）：** 先复核两个消费者实际合法输入与结果，能等价适配的共同纯机制可共用；若新旧政策不等价，保持窄版本入口和政策边界，不宣称简单 wrapper 完成兼容。禁止未经授权删除旧合同。
- **反转及尚缺事实：** 若确定旧消费者退役且有权完全重写，移除旧分支；不为不存在的消费者建立双路径。
- **验证意图（NOT_RUN）：** 旧有效输入、严格拒绝、相等及错误、数据版本、真实旧消费者调用；当前没有具体程序/调用记录，迁移设计仅候选，不能判定可合并。

## W06 · 两个内置格式转换是否需要插件平台

- **情境（教学假设）：** 一个 CLI 有固定 CSV/JSON 输出，转换是纯函数，当前没有第三方、独立发布、动态加载或资源访问需求。
- **入口轴：** 接口/复用/扩展，L3–L5。
- **实际读取案例：** [T12-C12](https://github.com/kaicreator-mm/tally/blob/41dad33035beab0eef1a1fffe3796d9fe71394db/docs/versions/v2.1.3/knowledge/casebook/waves/T12-comparative-design-cases.md#L104)、[T09-C07](https://github.com/kaicreator-mm/tally/blob/41dad33035beab0eef1a1fffe3796d9fe71394db/docs/versions/v2.1.3/knowledge/casebook/waves/T09-code-abstraction-cases.md#L59)、[T08-C02](https://github.com/kaicreator-mm/tally/blob/41dad33035beab0eef1a1fffe3796d9fe71394db/docs/versions/v2.1.3/knowledge/casebook/waves/T08-platform-functions.md#L13)、[T02-C02](https://github.com/kaicreator-mm/tally/blob/41dad33035beab0eef1a1fffe3796d9fe71394db/docs/versions/v2.1.3/knowledge/casebook/waves/T02-interactive-tools-and-computation.md#L15)。
- **专业知识：** T01 A1/A2、T10 D2；K02.F2.3、K03.F2.1、K02.F2.2（[T01](T01-theory-ab.md)、[T10](T10-theory-cd.md)、[原知识反向索引](T13-knowledge-and-coverage.md)）。
- **源观察到的区分（来自案例正文）：** Go HandlerFunc 表明小函数可适配真实窄接口；Python EntryPoint 动态导入有真实加载行为；Now in Android 的 api/impl 对应真实跨模块导航，不是全部私有函数公共化。
- **选择及拒绝（本轮推演）：** 固定格式直接分派，只有共同输入/输出语义的纯机制才提取私有 helper；不引入注册器、依赖注入、动态插件或独立公开接口。
- **反转及尚缺事实：** 有独立第三方作者、真实稳定能力/版本合同再设计窄公共接口；需动态选择与隔离再研究 plugin host；进程分离不证明沙箱安全，实际 OS 权限、IPC 授权与资源限制另验。
- **验证意图（NOT_RUN）：** 两种格式边界、错误与不可变输入；已有消费者旧合同；若未来第三方授权出现重新评估，不宣称当前维护收益。

## W07 · 桌面多窗口是否应各自持久化业务状态

- **情境（教学假设）：** 同一文档的列表/详情/第二窗口允许未提交草稿，业务内容只有一个本地文件/数据库权威；不需要设备离线写入。
- **入口轴：** GUI/状态/视图/所有权，L2–L4。
- **实际读取案例：** [T12-C02](https://github.com/kaicreator-mm/tally/blob/41dad33035beab0eef1a1fffe3796d9fe71394db/docs/versions/v2.1.3/knowledge/casebook/waves/T12-comparative-design-cases.md#L14)、[T02-C03](https://github.com/kaicreator-mm/tally/blob/41dad33035beab0eef1a1fffe3796d9fe71394db/docs/versions/v2.1.3/knowledge/casebook/waves/T02-interactive-tools-and-computation.md#L25)、[T02-C01](https://github.com/kaicreator-mm/tally/blob/41dad33035beab0eef1a1fffe3796d9fe71394db/docs/versions/v2.1.3/knowledge/casebook/waves/T02-interactive-tools-and-computation.md#L5)、[T08-C04](https://github.com/kaicreator-mm/tally/blob/41dad33035beab0eef1a1fffe3796d9fe71394db/docs/versions/v2.1.3/knowledge/casebook/waves/T08-platform-functions.md#L29)。
- **专业知识：** T01 A1/A3；K02.F3.1（业务模型 Owner）、K03.F3.1（状态转移）（[T01](T01-theory-ab.md)、[T10](T10-theory-cd.md)、[原知识反向索引](T13-knowledge-and-coverage.md)）。
- **源观察到的区分（来自案例正文）：** FreeCAD GUI Document 持有 App Document 并响应信号；Android 架构文档有本地持久数据源与派生 UI；Redux reducer 更新后通知 listener，不能当事务账本。
- **选择及拒绝（本轮推演）：** 共享文档 Owner，窗口保留局部草稿与派生展示，真实提交后通知所有窗口；简单直接观察即可，不强制 Redux 或离线双写数据库。
- **反转及尚缺事实：** 窗口极少且同步直接读取足够可更简化；跨设备离线写入才增版本与冲突策略。
- **验证意图（NOT_RUN）：** 两窗口同一对象、未提交草稿取消、旧响应、关闭窗口不删除模型、并发更新。FreeCAD 整体 undo 与同步语义未由头文件证明。

## W08 · 结构验证能否直接授权更新订单

- **情境（教学假设）：** HTTP 导入接口接受版本化 JSON，用户可更新自己的租户订单，跨字段数量政策和库存必须依据当前业务状态。
- **入口轴：** 输入校验/资源授权/状态，L4–L5。
- **实际读取案例：** [T12-C07](https://github.com/kaicreator-mm/tally/blob/41dad33035beab0eef1a1fffe3796d9fe71394db/docs/versions/v2.1.3/knowledge/casebook/waves/T12-comparative-design-cases.md#L59)、[T05-C04](https://github.com/kaicreator-mm/tally/blob/41dad33035beab0eef1a1fffe3796d9fe71394db/docs/versions/v2.1.3/knowledge/casebook/waves/T05-common-function-abstractions.md#L32)、[T07-C02](https://github.com/kaicreator-mm/tally/blob/41dad33035beab0eef1a1fffe3796d9fe71394db/docs/versions/v2.1.3/knowledge/casebook/waves/T07-business-functions.md#L13)、[T03-C05](https://github.com/kaicreator-mm/tally/blob/41dad33035beab0eef1a1fffe3796d9fe71394db/docs/versions/v2.1.3/knowledge/casebook/waves/T03-business-and-collaboration-architecture.md#L45)。
- **专业知识：** T01 A2/B1；K03.F2.2（类型与结构）、K02.F4.3（实际授权边界）、K05.F1.2（验证意图）（[T01](T01-theory-ab.md)、[T10](T10-theory-cd.md)、[原知识反向索引](T13-knowledge-and-coverage.md)）。
- **源观察到的区分（来自案例正文）：** Go Decoder 只读取下一值且可拒未知字段；Django 权限接口的 obj 参数不保证后端实现；Saleor 独立订单和支付状态提醒实际业务不变量。
- **选择及拒绝（本轮推演）：** 解析与完整输入/形状合同在入口；真实主体资源绑定和当前订单政策由受权业务执行点负责，必要时事务内重新确认。共享校验机制不能统一订单合法性和授权政策。
- **反转及尚缺事实：** 可信内部参数可减少重复结构检查；若真正公共 API 有版本变动需要明确未知字段策略，不能静默扩大写权限。
- **验证意图（NOT_RUN）：** 尾随 JSON、旧/新字段、同角色异租户、合法结构但库存不足、并发状态改变。未知业务政策停止相关决策，不构造通用强制规则。


## T14 修正后的四组只读正负选知回放

本轮实际根据修正目录和原 syllabus 机制人工核对下列正负关系。只证明选知候选定位与作者语义判断，不是自动检索实验或独立 Reviewer 通过；所有功能 Oracle NOT_RUN。

| 问题 | 查询原知识 | 正向候选 | 排除直接关联 | 判断理由 |
|---|---|---|---|---|
| DB 事务 vs 通用资源 | K03.F4.1 | AR-001、AB-014、T04-C05、T06-C03 | T05-C10、T06-C06、T12-C06 | K03.F4.1 按业务不变量选择 DB 事务及持久读回；后三条改按 K03.F4.2 资源释放查找，不能从清理推断事务。 |
| 首次配置 vs 普通激活/租户 | K10.F5.1 | DE-006 | T04-C09、T08-C05、T07-C03 | 只有首次启动、配置与授权条件启发。普通加载为 K03.F3.1，租户选择为 K02.F4.3；配置记录存在不证明首次使用门禁。 |
| 结构验证 vs 异常分类 | K03.F2.2 | T05-C04、T05-C05、T07-C07、T11-C04、T12-C07 | T05-C03、T12-C08 | 外部 JSON、动态类型与跨字段结构必须运行时验证；错误包装按 K03.F5.1 查询，不因都返回错误等同验证。 |
| 版本兼容 vs 渐进迁移 | K09.F3.1 | T11-C02、T11-C03、T11-C05、T11-C06、T12-C11 | T10-D2 | API 新旧合同与消费者是兼容问题；D2 的一般影响集合/经济比较不标 K09.F2.2，真正 Strangler 需实际旧新入口、切换和退役条件，不由版本变化自动推出。 |

## 结论与边界

八组实际完成有限只读选知，涉及工作流、资源、支付、检索、兼容、插件、UI、验证；不是八条新增案例，不改变 125 条计数。不报告检索精度、Agent 能力提升或方案执行通过；没有真实产品代码或独立读者对照。实际研究 R6 另见原证据文档。T14 应重点确认本次选择链条、候选适用条件、引用真实可达性和限制；如发现错误知识，修正对应正文和索引后再限定复审。