# T46-R1 · 真实产品的完整有限用户路径：Miniflux 阅读状态

#664/#563；K01 用户目标与接口合同，K02 结构/权威，K03 数据/算法/实现，K05 验证，K06 诊断，K09 维护，K10 运行。冻结 Tally 091d875fb97e2ca3239d9ac1ead60648d21e807c；上游 miniflux/v2 tag 2.2.3 实际解析 commit da951164d52e7e4f987c50a64f1d1be01e6e90e5。以官方 Go client 为实际使用入口，选择“查询本人未读文章 → 标记所选文章已读 → 再查未读列表及指定文章”这一完整有限 API 路径；不是 Web UI、全产品或启动/运维审计，不运行上游。所有源定位见 [T46 证据登记](../T46-real-product-path-assessment.md)；S01–S18 是本批局部来源引用，不是新增 qualified Case/Claim/Rule。既有 T19-C01 只研究刷新子系统，以下新增真实消费者及另一条读写闭环，不重复计数 T19。

## 1. 产品需求和架构的不同证据

S01 README 自述 Go、PostgreSQL、无 ORM、原生 JS、单二进制等产品选择；这里只确认公开说明，不确认实际构建、部署、吞吐或全部依赖。S02–S05 实际客户端和 /v1 路由、S06–S08 认证及请求身份、S09–S18 handler/校验/存储/数据迁移/响应/测试提供具体路径源码。用户任务是从这些真实能力抽象出的有限使用目标；不宣称已访谈真实用户或获知所有产品政策。

实际跨层调用（箭头是读取到的调用关系，非拆分服务或部署图）：

    Go client.Entries(Filter) / UpdateEntries(ids,status) / Entry(id)
        -> client.request (HTTP GET/PUT, credentials, JSON)
        -> /v1 router -> CORS middleware -> API key auth -> basic auth
        -> API handler (query/JSON input, validator, scoped builder or storage)
        -> PostgreSQL query builder / Storage.SetEntriesStatus
        -> model + JSON response builder -> client error or decoded result

用户身份从认证上下文到查询和更新 WHERE，文章身份从查询返回到后续写命令，状态从请求/校验到 SQL 和查询表示。worker pool 虽被注入 handler，但选定的查询、状态更新方法直接调用 storage，不经过 T19 的异步刷新队列；不因同项目有 worker 就引入异步任务。没有从 package 名推出独立领域层、微服务或完整的安全边界。

## 2. 正常查询：调用参数到实际数据库和消费结果

S02 L486–501/657–741：client.Entries 将 Filter 构造为查询参数，发 GET /v1/entries，成功读取并解码 JSON，defer 关闭 body；S04 的 EntryResultSet 有 Total 与 Entries。Filter 允许 status、statuses、方向/顺序、limit/offset、时间/ID 范围及搜索等；本例显式传 Status=unread、Order=id、Direction=asc、Limit=2、Offset=0，不依赖隐含默认。S03 的 request.execute 构造 URL、method、headers、凭证和 HTTP 客户端（默认 timeout 常量80秒），按实际状态码处理并返回 body/error；未核 Go 标准库的所有重定向、取消或传输重试行为。

S05 /v1 GET /entries → getEntries → findEntries（S09 L83–167）。S09 读取状态列表并校验每个状态、排序字段、方向、offset/limit 范围；取 request.UserID，正数 feed/category 参数按该用户检查存在；创建用户限定 builder，设置条件、排序、分页、tags、enclosures 及其它筛选。S10 状态/排序白名单，S11 范围拒绝负数、方向仅 asc/desc。S12 constructor 初始化参数 userID 与 e.user_id=$1；非空 statuses 用参数数组匹配，conditions 用 AND 连接；排序表达式拼接，故入口验证排序字段/方向有真实意义，不能只见参数绑定就声称所有表达式安全。S12 buildSorting 仅在正 limit/offset 时附加子句。S09 实际 GetEntries 后另调用 CountEntries，再重写内容中的媒体代理 URL，json.OK 返回 total 与 entries；S16/S17 实际响应 builder 默认200、设置 JSON 类型及写响应。S12 GetEntries 关联 feed/category/icon/user，逐行构造 model，按需查 enclosures，转换时间并追加结果；CountEntries 用同条件但不带排序/分页，另执行 count(*) 查询。读取了调用和局部 SQL，未读 URL rewrite、enclosure 查询、所有 query 参数解析或完整匿名路径，不声称所有表示/授权已审。

有限查询表示：文章 ID 是数据库行身份，user_id 是作用域，feed_id/hash 是另一来源关联身份；status 为 unread/read/removed，starred 为独立表示。S14 初始迁移声明 entry_status enum、id 主键、(feed_id,hash) 唯一及 user/feed FK；后续有 user/status 与按 published_at、created_at、changed_at 相关索引。仅确认这些迁移片段，不称最终安装数据库 schema、全部约束或具体查询计划已证实；不能用 id 唯一代替 user scope，亦不从 FK 单独推出所有跨实体业务授权。S15 model 对应序列化字段，client model/服务端 model 属各自消费者与提供者合同。

## 3. 身份与变更：同步提交及其真实结果资格

S02 L540–549：UpdateEntries(ids,status) 编码 entry_ids/status，PUT /v1/entries，返回 request error，不返回逐项结果。S03 发 JSON、API token header 或 Basic auth；S05 路由使用认证中间件。S06 API token 非空时查 UserByAPIKey（S08 使用 token 参数查询关联 user）；无 token 转后续 Basic；找不到用户拒绝，读取错误返回服务器错误；成功把真实 user.ID、认证状态等写入请求 context。Basic 分支若已认证直接通过，否则核凭证、查用户、写上下文；S07 UserID 从 context 取值。这里观察认证和用户身份传播，并不证明 token 唯一性、认证撤销与实际写入的线性化或全产品安全；认证本身还包含 last-login/API-key-used 时间写入，不把 GET 的全部系统效果说成纯读。

S09 L169–187 handler 解码请求；S10 验证非空 ID 列表与三种 status；错误走 BadRequest。合法后 storage.SetEntriesStatus 以当前请求用户和 ID 数组进行单条 UPDATE，设置 status 与 changed_at=now()；S13 L369–387 检查 Exec、RowsAffected 错误及 count==0，非零即 nil。S09 nil 后 json.NoContent；S16/S17 设204并写 headers；S03 204 关闭 body 并返回 nil,nil；client 最终 nil。这里成功完成点是数据库更新调用及非零受影响行返回后同步响应，不是待办受理。单语句没有逐项结果或 expected version；不能把此响应提升为“每个请求 ID 都获准且已更新”“状态永远保持”“严格一次副作用”或“同历史响应”。对无匹配 count=0 走存储错误→ServerError，非不存在逐项404。对非空混合 ID 列表，至少一个本用户匹配即可能204；其它用户不匹配，非授权 ID 不在该 WHERE 的写集内。重复 ID 在 ANY 中不会使一行成为多行；真实 DB 执行/trigger 等另核，以下纸面例假定该语句成功且无其它影响。

## 4. 从实际机制推导完整有限用户任务（原创固定数据，NOT_RUN）

教学初态 A 用户拥有文章 11、12（均 unread），B 拥有 21（unread），A 无 99；均有正常关联 feed/category/user，ID 唯一，不涉 enclosures 或并发。A client 查询 status=unread, order=id asc, limit=2, offset=0，按已观察 SQL/正常响应的独立预期为 [11,12]、total2；PUT [11] read 得204，然后再次查询同条件应 [12]、total1，直接查 11 应 read。S02 Entry(id) L470–483 走 GET /v1/entries/{id}；S09 L24–41/65–71 以当前 user 建 builder，S12 L115–122/240–257 使用指定非零ID及 limit1、GetEntries、附件读取，正常返回实体；无记录走 NotFound。这个完整用户路径的结束条件是权威再次观察状态，而非仅客户端按钮变化或204。

区分反例（固定相同初态，独立于实现返回）：
| 输入/历史 | 由实际局部机制推导的观察候选 |
|---|---|
| PUT [] read / PUT [11] invalid | 400；请求校验阶段不调用 SetEntriesStatus |
| PUT [11,21,99] read | 仅 A 的11匹配，非零 count 候选204；B 的21不变，99不存在；不能当全部三个 ID 成功 |
| PUT [21,99] read | 零匹配 → 错误候选500，不是逐项404；B 的21不变 |
| PUT [11,11] read | 仍为一行匹配，不是两条独立效果 |
| 先成功 read，稍后相同请求再来 | status 值仍 read，但语句仍写 changed_at；不同时间可产生不同可观察历史，不证明全效果幂等 |
| 查询列表后、count 前另一请求修改状态 | 两次 SQL 读取没有本路径可见的共同事务；不能保证 total 与返回列表同一观察快照 |
| A 的未读 [11,12,13,14]，页1=[11,12] 后这两条改 read，再以 offset2 获取未读页 | 新结果 [13,14] 长度2，offset2 跳过两条；对照重新从头或有明确键/快照合同的遍历 |

上述是源码与固定前提下的可反驳推论，不是上游运行或缺陷复现；没有从不符合更强假设合同就认定项目有缺陷。T42-R1 分页、T40-R1 组合、T27b-1 事务、T25b-2 时间、T24c 任务观察可直接消费这些真实机制。

## 5. 真实验证边界与未核范围

S18 实际已有 validator 单元测试：合法 IDs/read、空列表、非法状态、状态枚举及部分排序字段/非法值；本轮只读测试，不执行，不能说这些测试覆盖认证、跨用户 SQL、并发或整个消费路径。GetEntries 实际循环结束直接返回 entries,nil（S12 L320–398），未见 rows.Err；因此可提出迭代中途异常可能使结果部分返回的独立验证意图，非已发生实例。客户端更新只取 error，request helper 非上述显式错误且状态码≤400（除400处理）可返回 body,nil；对真实204正常关闭，不能据此把任意响应等同实际服务成功。查询默认 published_at/asc（S15 L10–16），本路径显式 id 排序，未声明所有排序均稳定。Limit 0 由真实校验允许且 builder 不加 LIMIT；不能把 client Filter 默认数值或100默认请求参数自动当资源上限，实际参数解析/代理约束待核。真实产品未启动，所有目标执行、性能、全安全及用户效果 NOT_RUN；下一章由实际源码约束做维护决策推导，不升级原有案例资格。
