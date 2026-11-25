# T35-3 · 复杂通用功能组合：可恢复下载与导入、权限缓存与异步交互

#624 / R4-G / RE-02；K02/K03 结构与实现，K01 目标和授权，K05 独立验收，K09 变更，K10 运行。承接 [T35-1](T35-1-general-function-knowledge-composition.md) G1/G2。复用 [T06](../casebook/waves/T06-data-file-network-functions.md) C01/C04–06、[DE](../casebook/derived-design-cases.md) 001/003/005–008、[T02-C06](../casebook/waves/T02-interactive-tools-and-computation.md) 缓存失效、[T19](../casebook/waves/T19-feed-refresh-architecture.md)、[T20](../casebook/waves/T20-historical-abstraction.md) 的实际局部观察/教学参考，不重写 DownloadJob、基本批处理、缓存及取消机制。R3-RV-01 已限定关闭。本篇两组具体组合使用明确教学合同，全部 NOT_RUN，不新增 qualified Claim/Case。

## 1. G3：可恢复的大文件下载与导入

教学需求：租户 A 从获权 HTTP 源导入一个 UTF-8 CSV，允许暂停、跨进程重启恢复，下载字节有界缓存，源可能换版本。目标权威 PostgreSQL18 数据集当前 generation g1；新批次必须全部验证后原子切换到 g2，旧 g1 在有权保留期内可读，禁止半批可见。另一产品明确允许逐行部分成功，作为独立变体。任务 identity=(tenant,operationId,sourceIdentity,expectedTargetGeneration,contractVersion) 的绑定不得被重试改写；同 operationId 异输入拒绝。用户只获本租户导入许可，开始、执行许可/目标切换、完成读回分别检查；源访问合法不授予目标写权限。源改变后不得把旧片段混入新版本；允许重新下载须由原合同明确允许或新获权任务决定，不能悄悄改目标快照。

### 1.1 实际协议与下载状态

采用 RFC 9110 HTTP 语义中的单一 byte range。强 ETag 是表示验证器，不必是哈希；URI 相同、长度相同或弱 ETag 均不足证明可拼接。固定所选 representation（源资源、获权访问范围、内容协商及 content coding 等）；本例只接收 identity coding，若服务不满足则拒绝续传或获权从头获取并使用正确字节域。RFC §14.1.2 的 range 是编码后表示字节，不是解压后偏移。Accept-Ranges 仅提供建议；服务器可不返回部分内容。If-Range 采用强验证器，匹配失败应忽略 Range。没有足够验证器时不跨请求复用片段；如另有获权可信固定内容清单，可依该具体合同验证，不假设任何 HTTP 源都有清单。

教学固定源 total=1000，强 ETag "A"，可信清单给两个块各自 digest 及全文件 digest；已持久验证区间 [0,399]，请求 Range: bytes=400-999 与 If-Range: "A"。清单及字节身份均需从有权独立来源得到，本例不是 ETag 自带密码学保证。

| 响应/事实 | 合法处理 |
|---|---|
| 206，Content-Range bytes 400-999/1000 | 检查单段、字节域、范围与表示身份、实际内容600字节及完整性；源验证器依据 If-Range 及实际响应合同核对（不要求所有合法206重复全部表示头）；全部校验才进入持久片段 |
| 206 仅满足400-699，或长度/范围不符 | 对所选完整块合同，不能将300字节记为600字节块完成；保留已独立验证部分须有对应清单/记录，否则丢弃未验证内容、重新请求缺口；非法 Content-Range 不拼接 |
| 200 全表示（源变"B"或忽略Range） | 绝不追加到原400字节；原任务如固定"A"则拒绝/等待授权；若仍为合法同源版本，可全量重新验证；获权新版本使用新 staging identity |
| 416，Content-Range bytes */1000 | 仅表示当前长度/范围条件；即使本地长度1000也须核身份、全量完整性及持久事实；不能直接报完成 |
| 连接中止、无验证器、解码/摘要失败 | 未验证部分不晋升；关闭所拥有资源，保留已有合法块或按合同从头恢复；不把100%传输进度当成功 |

采用有界片段与独立清单，不引入多段协议处理；若要并行分块，另核服务容量及覆盖/重叠/重复的实际合同。接收与持久事实分离：写入隔离临时文件、验证块、按实际文件系统持久化内容及必要元数据后才登记 durable range receipt；以 [T27b-2](T27b-2-file-stream-and-durability.md) 的平台原子/持久化边界为准，不声称文件 rename 与 DB checkpoint 是一个事务。内容落盘后 receipt 前崩溃，恢复时重新校验已存在文件/片段或丢弃；receipt 存在但文件缺失/损坏时撤回相应已验证进度、重新获取或阻断，不直接读未验证区域。只有完整覆盖、长度与独立 digest 均满足且任务许可有效，才能将 staging 对象标为 READY_TO_PARSE；不对用户公开临时路径或未验证内容。

### 1.2 数据导入：块进度不是记录进度

CSV 引号内可含换行，UTF-8 字符可能跨字节片段；下载块边界不是 CSV 记录边界。最小候选是下载完成后流式解析，用确定 parser/dialect/encoding 及完整记录序号标识记录；恢复可从头验证并跳过已登记的规范记录，或在确实可重建 parser 状态的记录边界存 checkpoint，不把物理行号或字节偏移任意当合法续点。PG18 COPY FROM 的转换错误策略 stop/ignore 和 REJECT_LIMIT 只针对规定转换错误，不能代替领域校验或完整错误清单；COPY FROM 会触发目标表触发器/check，不支持启用 RLS 的表。选择获权隔离 staging（不暴露其它租户或允许任意 server file/PROGRAM），可用 COPY FROM STDIN 仅在实际适用时导入受限 staging，再由保护的业务事务写目标；若需要 RLS 写路径使用合法 INSERT 等。业务唯一性、单位、字段语义、错误记录及租户由真实合同约束，不把 COPY count/NOTICE 当验收。

教学文件四条逻辑记录 a,b,c,d，c 数量非法，批1包含 a,b，批2包含 c,d。staging 每批按 (jobId,recordOrdinal) 唯一登记原身份、规范值/拒绝理由与 checkpoint，在同一 PG 事务提交；重复相同批核对摘要/版本而非再写；同序号异内容拒绝。原子发布合同：c 非法则不切换 g1；即使 a,b 已持久 staging，也不属于当前数据集。另一有权部分成功产品：a,b,d 合法逐批提交，c 持久拒绝，终态 PARTIAL(3,1)，重试不再新增 a,b；独立 Expected 逐记录且同数据范围核对，不用总数代替。部分成功变体不能用来放宽原子合同。

原子版本发布选择：在 staging 中构造独立不可见 g2，核全文件、完整记录、领域约束、实际当前 g1 和授权/执行许可；同一权威 DB 事务更新当前 generation pointer、任务 outcome 及必要 change（参 [T26a-2](T26a-2-architecture-style-composition.md)、[T27c-3](T27c-3-reliable-messaging-outbox-offsets.md)），所有消费者必须只读选定 generation；若旧入口直接扫描所有 staging/新旧记录，则该结构不能实现原子可见性，需修改/验证实际消费者或采用满足规模的整批事务。保留历史按有权策略，不以清空 g1 当回滚。外部文件若作为业务权威则另需跨资源恢复合同，本例只有同DB权威切换。

| 故障窗 | 独立期望与恢复（NOT_RUN） |
|---|---|
| 下载收到后未持久；持久后 receipt 前；receipt 与文件不一致 | 只以实际已验证字节登记恢复；重新核摘要、范围与文件身份；不越过缺口 |
| 批写过程中崩溃 | 同一事务的数据和 checkpoint 均未晋升；重试相同记录，不漏/重 |
| 批提交后响应丢失 | 从真实 staging+checkpoint 读回；不同输入同 identity 拒绝；部分成功变体同样核实际结果 |
| g2 已备好、切换前崩溃或撤权/旧 Worker | g1 仍唯一当前，任务未成功；当前 grant 和 expected g1 不满足时拒绝，即使全部校验成功 |
| 原子切换提交后 ACK 丢失 | 真实 g2 pointer/outcome/change 同时存在，查原任务，不能新开任务重复发布 |
| 第三条 c 非法 / 截断 / 超资源预算 | 原子合同保留 g1；合法部分变体只对已确定的完整有效记录承诺结果，截断剩余 UNKNOWN/FAILED 不称完整 |
| 新增一种合法 CSV 方言 | 保持原 parser 合同与旧记录身份，显式 version；旧 checkpoint 不套新 parser；旧格式回归、跨引号和编码块边界、恶意超长记录等独立负例 |

短小且可重下的文件可直接一次读取/事务，不要求上述 staging 和恢复状态；若需求变更为大流、跨重启则增加最小持久边界。独立观测源固定 bytes/摘要、文件/receipt、真实 DB g1/g2、每批输入输出及当前授权，不依赖被测进度 UI。资源预算/保留/具体文件系统能力须实际获权，未测不声称高吞吐。

## 2. G4：权限、缓存失效和异步 UI 的交错

教学合同：租户 A 的搜索面板显示当前有权项目的资料；数据允许最多一代明确标陈旧的快照（非权限陈旧），切换租户/查询必须以最新 UI 请求身份决定呈现。撤权后禁止新披露及写入；已合法交付的字节不能远程追回。服务端的披露准入必须与权威授权变化有明确顺序，本例选择权威在同一受保护执行边界签发当前披露许可并在实际输出交付点验证，不能仅用早期读取/旧 token；若使用独立服务无法保证该顺序，则拒绝高敏路径或获权不同合同，不声称零竞态。客户端在接收到撤权/会话失效时清除当前敏感视图及相关缓存，服务器不依赖 UI 来执行权限。服务端长流输出合同另见 T35-4 B4。

### 2.1 HTTP 缓存与应用缓存的不同合同

RFC9111 §3.5 对带 Authorization 的请求限制 shared cache 复用（有明确允许指令且满足相应要求才例外）；不将它概括为所有私人内容绝对不会缓存。Vary 影响匹配请求的表示字段，不证明业务资源授权。no-cache 允许存储但要求复用前验证；no-store 限制 HTTP 缓存存储/复用，不是保证隐私、清除所有历史或控制任意应用内存。敏感搜索教学选择不使用 shared response cache，HTTP 响应 no-store；服务器内部只允许经权威授权控制的派生查询缓存，不等于 HTTP cache。优先无缓存直接查询；有实际成本/延迟需求才考虑以下内部缓存。已有 [T27b-1](T27b-1-relational-data-and-transactions.md)、[T32b-2](T32b-2-data-lineage-conflict-tombstones.md) 的范围与版本语义保持原归属。

内部缓存键至少绑定 (tenant,authorizedScopeIdentity,authorityEpoch,query/filter,locale/sort,representationVersion,dataGeneration)，必要时含 actor/session 或不共享。scope identity 与 epoch 必须来自当前受信权威，不信客户端自报角色或 tenant；同一角色名并不保证相同对象权限。当前授权必须在 hit/miss 和返回时都保护；缓存只能选择当前获权数据的可接受版本，不负责授予权利。缓存构建时从确定快照记录 generation；写回采用受保护的版本匹配或不可复用的旧版本分区；删除缓存并不阻止在途旧请求重新填充。若 epoch/数据水位无法确认则直接权威查询或拒绝，不使用过期权限缓存。搜索在授权范围内筛选后排序/分页/聚合，后置 UI 隐藏或仅过滤 top-k 不保证有权结果完整，参 T06-C05。

### 2.2 同一交错中区分三种版本

纸面：A 的 actor U 在权限 epoch7 可查项目 P,Q；数据 gen20。请求 R1 查询 all，开始构造结果 P,Q。权威先删除 Q 的可见权限（epoch8），随后 P 更新 gen21。UI 切换到查询 P-only，发 R2，捕获新 requestSequence=2、session/tenant。R2 完成显示获权 P gen21。R1 后返回：
- 服务端不能把 epoch7 结果作为 epoch8 新披露；若旧结果构造已完成但撤权先于输出准入，必须重新按当前 scope 查询/安全拒绝，不能只丢掉 Q 后沿用旧排名/计数。旧 cache fill 不得在新 key 中重建旧结果。若 R1 已合法交付于撤权前，客户端不能撤销网络历史，但不得在收到撤权/切换后重新呈现。
- UI 仅接受当前 (session,tenant,requestSequence,适用查询 identity) 匹配的结果；R1 晚于 R2 不应把 UI 改回 all 或将旧错误覆盖新成功。UI requestSequence 不是 DB 版本/权限 token。数据 gen20 仅在当前 scope 合法且陈旧合同允许时可显示并标陈旧，不能发起写入依据。
- 写操作以权威当前权限及业务版本执行（例如 expected gen21），不信缓存/浏览器 state。取消请求只停止等待或按实际取消合同终止执行，不保证后端未完成；任何有持久效果的任务须以原 operationId 查询。

若用户切换到租户 B 且 query 相同，不重用 A 的结果/请求/分页 cursor；若网络断开，公共已缓存信息按有权规则显示，敏感视图按合同隐藏/拒绝，不能说“离线所有授权仍有效”。若服务只支持单用户本地无权限变化，无需上述多租户缓存机制。若新增需求为实时协同实时权限，重新评价版本水位、披露时序及真实服务能力，不能把 TTL=0 当强一致授权。

| 独立正反 Oracle（NOT_RUN） | 区分性结果 |
|---|---|
| A/B 同名项目和相同 query；不同 actor 同 role | 实际有权集合、总数、分页、错误及命中缓存数据不得越界；以独立权威记录构造 Expected |
| epoch7 构建、epoch8 撤权后输出/填充 | 无新 Q 披露，旧填充不污染 epoch8；若 R1 先合法交付则单列不可追溯撤销；验证真实授权顺序 |
| gen20 构建、gen21 失效、gen20 迟到 | 新版本 key/水位不倒退；获权陈旧允许条件与错误拒绝分别观察 |
| R1→R2，R2 先成功，R1 后成功或报错 | 最新 request 的数据与状态保留；切换租户、会话失效后旧请求不能渲染；不能以 AbortController 已调用替代 |
| 缓存不可用、304 或授权服务失联 | 不从旧授权推断合法；304 不证明有权内容未变，实际先核当前 scope 再采用缓存；按当前合同直接查询/拒绝 |
| 合法新增 locale/filter 或授权收窄 | 真实 key/排序/分页变化、旧缓存与 cursor 失效、旧/新消费者和可观察行为分别回归；不凭接口数量评价抽象 |

## 3. 来源、归属与剩余范围

实际读取 2026-11-26 [RFC9110](https://www.rfc-editor.org/rfc/rfc9110.html) §13.1.5 L2864–2885、§14.1.2 L2986–2988、§14.2–4 L3062–3141、§15.3.7 L3358–3373；[RFC9111](https://www.rfc-editor.org/rfc/rfc9111.html) §3.5 L296–308、§4.1 L327–338、§5.2.2 L594–623；[PostgreSQL18 COPY](https://www.postgresql.org/docs/18/sql-copy.html) L145–162、190–204。引用实际协议/命令局部合同；未运行 HTTP/PG/缓存或具体 UI，不宣称实现支持全部特性。T35-1、T06、DE、T02、T19/T20 为已接受原观察/教学；本篇 G3/G4 只填 R3-RV-01 残余组合，不新增真实案例身份。RE-02 仅两个选定复杂组合候选；跨进程关闭、异常与安全日志、其它下载协议与 UI 平台的未选变体继续 OPEN/UNKNOWN。所有政策、数值、交错及 Oracle 为原创公开教学 NOT_RUN；不得作为隐藏实验题或直接成为项目 Rule。
