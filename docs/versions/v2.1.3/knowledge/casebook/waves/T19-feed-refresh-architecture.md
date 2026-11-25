# T19 · 完整子系统架构：Miniflux 订阅刷新

[#576](https://github.com/kaicreator-mm/tally/issues/576) / #563；基线 0516b64b17da66f3029dbd34664b1328bf254a54。一个 L3 集成源码案例 T19-C01，九个新增固定来源。研究 Miniflux v2 2.2.3 已存在的订阅刷新子系统：从手动 API 请求、任务选择及工作池到共同刷新流程、条目处理、实际持久更新和外部集成边界。多个视图及反例不是额外案例。不是全产品架构审计；周期调度入口/启动装配、路由授权中间件、解析器/HTTP 实现内部、DB schema/隔离和外部集成内部不在本次核实范围。所有运行/Oracle NOT_RUN。

## 增量目录

| 案例 | 真实机制 | 关联及限定理由 | 来源 |
|---|---|---|---|
| T19-C01 / L3 | 多入口共享刷新编排，独立查询/处理/持久化/异步边界 | K02.F2.1 机制：真实 package 职责和调用边界；K02.F2.2 机制：单条与批量 HTTP 返回的不同完成合同；K02.F3.1 机制：条目逐条事务与订阅另行写入；K03.F3.3 机制：无缓冲通道及 worker 数量；K03.F3.2 条件：部分提交、并发刷新及外部效果失败窗口提示重试合同，非幂等保证 | T19-S01–S09 |

## T19-C01 · 从产品能力到结构

### 1. 功能、数据和接口视图（源码观察）

单条 refreshFeed 从请求读取 userID/feedID，检查 FeedExists，再直接调用 reader/handler.RefreshFeed；失败返回 ServerError，正常返回 NoContent。批量 refreshAllFeeds 获取 userID，构造 BatchBuilder，传入错误阈值（仅正值启用）、排除 disabled、next_check 已到期、用户条件，FetchJobs 后启动 goroutine 执行 pool.Push 并返回 NoContent（S01 L53–95）。同一 HTTP 204 对应不同完成点：前者等待 RefreshFeed 返回；后者仅证明代码已启动异步推送路径，不证明每个 job 已被 worker 接收、成功刷新或持久排队。名称 all 并非全部订阅无条件刷新。userID 取自 request 的方式和路由鉴权不是本次证明对象。

BatchBuilder 实际 SELECT id,user_id FROM feeds；条件拼接并用参数绑定用户、错误阈值等；仅 limit>0 时 ORDER BY next_check_at ASC 与 LIMIT。上述批量入口未设置 batch size，不能用 builder 支持 limit 推断该入口有总待办上限。FetchJobs 将结果追加 JobList（S02 L14–89）；Query 或 Scan 错误显式返回，但 rows.Next 结束后未检查 rows.Err，若迭代中途发生终止错误，可能返回部分列表及 nil error，继而触发异步推送并返回 204；未复现实例，不以 nil error 单独证明扫描完整。Pool 内含无缓冲 chan model.Job，NewPool 启动指定数量 goroutine，Push 逐项发送（S03 L12–35）。Worker.Run 循环接收，调用同一 RefreshFeed，按结果记录可选指标和错误日志（S04 L18–56）。无缓冲 channel 可使发送者等待接收，但异步入口可产生多个阻塞的 producer；并发 worker 上限不等于总排队/内存/网络请求上限。未核全局关停、取消、重试、跨进程调度，亦不声称并发刷新已去重。

### 2. 组件与依赖视图（真实调用关系，非建议分布式拓扑）

```text
API refreshFeed -----------------------------------+
API refreshAllFeeds -> storage.BatchBuilder        |
                    -> goroutine -> Pool -> Worker|
                                                   v
                              reader/handler.RefreshFeed
                   +--------------+-------+-----------+---------+
                   v              v       v           v         v
               storage          model   fetcher     parser   processor
             read/write       timing    HTTP       parse     entry rules
                   ^                                          |
                   +------------------------------------------+
                   |
            separate integration lookup -> goroutine PushEntries
```

图中 integration lookup/PushEntries 由 RefreshFeed 调用（不是 storage 自行发送），processor 也调用 storage 读取条目相关信息。依赖方向是源码调用，不等同纯 DDD、六边形或独立部署；handler 与 processor 都依赖具体 Storage，不能宣称 domain core 已与数据库完全解耦。模块边界有现实工作职责：入口翻译请求，查询选择待办，worker 管并发与执行，handler 编排，model 算时间，processor 执行条目规则，storage 持久化。只需要单入口时可以更少模块；实际重复入口/变化轴出现才复用相同业务功能，不按每个表建立微服务。

### 3. 业务流程与状态视图（源码观察）

S05 RefreshFeed L191–367：读取 user、按 userID/feedID 获取原订阅（不存在即错误）；若按条目频率调度则查询周条目数；更新 CheckedAt 与下一次检查时间；建立 HTTP request builder，按配置设置 timeout/代理等，不强制刷新且不忽略缓存时带 ETag/Last-Modified。ExecuteRequest 后 defer responseHandler.Close；限流时读取 retry delay（秒转分钟），重算下次检查。LocalizedError 或重复 URL 等分支写错误并返回。忽略缓存或 IsModified 时 ReadBody（传最大 body size）、ParseFeed，解析失败写错误；成功用 TTL 重算下次检查，赋新 entries 并调用 ProcessFeedEntries，再 RefreshFeedEntries。未修改分支不走条目处理，必要时更新 LastModified；之后 reset error、UpdateFeed，成功返回 nil。IsModified/ReadBody 等内部合同未核，不能把条件等同仅 HTTP 304。

S09 L93–137：Feed model 维护错误计数、checked_at、next_check；ScheduleNextCheck 默认取全局 round-robin 最小间隔，条目频率策略依据周数和配置 min/max，若 TTL/Retry-After 换算分钟数高于计算间隔则再抬高；以当前时间加间隔。此方法计算状态，不代表实际定时器唤醒/准时完成，也不验证全部配置有效性。

S06 L31–128：processor 从 slice 尾向前处理；按 blocked/allowed/recent 筛选，清理/重写 URL，查新条目；crawler 且新条目或强刷时尝试抓取正文，抓取错误记录日志而非替换内容；随后 rewrite，确定 base URL，调用 sanitizer，再更新时间估计，生成 filteredEntries。实际顺序支持“修改内容后再清理”的边界选择，但未读 sanitizer 内部，不能证明所有 XSS/SSRF 安全、时间排序正确或所有内容已验证。业务筛选、网络抓取、内容转换仍组合在同一 processor；有真实变化/独立测试需求时才进一步拆分。

### 4. 事务、外部效果与失败视图（源码观察与限定推论）

| 实际路径 | 观察到的写入或返回 | 不能推出 |
|---|---|---|
| 单条刷新 | API 等待 handler，nil 后 NoContent | 所有异步集成成功 |
| 批量刷新 | FetchJobs 后 goroutine Push，NoContent | 全部 job 已接收、耐久或完成 |
| 修改内容 | RefreshFeedEntries 后查集成、可能启动 PushEntries，随后 UpdateFeed | 条目、缓存元数据与外部效果同一事务 |
| 条目存储 | 每条 entry Begin，查存在，按参数 update 或 create，Commit；末尾 goroutine cleanup | 整批 all-or-nothing、严格一次写入或可靠 cleanup |
| 错误写入 | 若干失败分支调用 UpdateFeedError，忽略其返回；ReadBody 失败直接返回 | 每次错误必然落盘或统一重试 |
| 订阅元数据 | UpdateFeed SQL WHERE id AND user_id，另有 UpdateFeedError 仅写错误/检查字段 | 可见 SQL 路径具备 CAS 或全系统无竞争 |

S08 L270–327 的每条事务可能已经成功提交，后续条目失败时不会自动回滚前条（从循环与 commit 位置推导）；返回 nil,error 丢弃本次 newEntries 返回值，不能据此断言数据库无新增。cleanupEntries（L253–268）针对 removed 且不在 hashes 中的条目，异步调用。S05 L324–333 查询集成失败只日志并继续，存在集成和新条目时异步 PushEntries；发生在最终 UpdateFeed 之前。S07 L317–425 显示分别执行更新 SQL，未见 version 条件，也没有读取 affected rows；这里只提出竞争/零行等验证意图，不认定现有项目整体有缺陷。S05 L198–210 / S07 L198–211 使用 user+feed 查询构建器，具体 builder/认证未审。外部调用前后失败与数据库提交结果未知必须区别；本案例不是全面事务审计。

### 5. 从此架构导出设计判断（原创分析，不是项目事实）

先确定需求合同：单用户要求等待一次刷新，批量触发允许稍后完成；是否要持久待办、用户可见进度、丢失/重复容忍、每条或全量原子更新、外部推送策略均由实际业务决定。最简单方案是直接调用共同刷新函数；有真实批量并发时添加 bounded worker/producer 责任，若可按数据库订阅状态再计算到期工作，不能仅为异步就断言需要持久队列；若接受后不得丢失，则需真实持久接收与恢复。当前源码并不证明某种可靠性合同已满足。

把 HTTP 读取与内容处理、业务刷新与持久化、任务提交与完成状态分开验证。若条目独立增量允许部分成功，逐条事务可能合适；若整个订阅快照必须原子发布，则需要不同事务/版本方案，不能把当前 per-entry 机制照搬。并发刷新若会覆盖用户修改，考虑单订阅串行、字段级写入、版本条件或其他经验证机制；不默认每个模块都需要 repository 接口、消息总线或分布式锁。外部推送需要按允许丢失/重复选择策略，复用 T18 P2 澄清：允许丢失且不重放 UNKNOWN 可选择 at-most-once 尝试政策，但仍要核实实际重试路径；可靠重放且禁止重复效果需稳定身份、去重/条件提交或可信远端回读等能力，不假设本地 key 就能控制远端。

### 6. 可判别验证计划（全部 NOT_RUN）

| 场景 | 需要观察的证据 |
|---|---|
| 单条成功、批量立即返回 | HTTP 完成点与 worker 真正完成分别记录；批量排除未到期/disabled，正错误阈值才排除达到或超过阈值的订阅；对照零/负值与正值；模拟 rows.Next 中途失败，核实列表、HTTP 结果与 worker 实际处理，不以 nil error 证明完整 |
| 并发 producer 与有限 worker | 测在 worker 忙时 Push 阻塞、总 producer 数及待办内存，非只测线程数 |
| 内容未修改与强制刷新 | 观察是否执行 parse/processor/entry write；下一次检查时间及缓存头变化 |
| 条目 1 提交，条目 2 写失败 | 查实际 DB 条目、handler 错误与元数据；不可用 error 推断零写入 |
| 条目写成功但 UpdateFeed 失败 | 核对缓存元数据、错误记录、集成效果与再刷新政策 |
| 两次刷新或用户修改交错 | 比较陈旧读取与最后写入；根据业务合同裁定，而非从未见锁直接判系统故障 |
| 抓取正文失败、恶意内容、限流 | 原内容是否保留、sanitizer 实际输出、下一次检查与错误处理；安全验证需另行完整边界审计 |
| 进程停止或远端结果 UNKNOWN | 核实真实恢复、丢失/重复容忍与外部回执；不把返回 204 当完成 |

## 来源登记（实际读取 miniflux/v2 ref 2.2.3）

- **T19-S01** [internal/api/feed.go](https://github.com/miniflux/v2/blob/2.2.3/internal/api/feed.go) · Blob 3bcc2edccf8c873fb3a7f19b39f18c25d1b65c20 · L53–95。
- **T19-S02** [internal/storage/batch.go](https://github.com/miniflux/v2/blob/2.2.3/internal/storage/batch.go) · Blob bf6a0d249857d6b672c873474a64ebb2ad8d5533 · L14–89。
- **T19-S03** [internal/worker/pool.go](https://github.com/miniflux/v2/blob/2.2.3/internal/worker/pool.go) · Blob 44c899c4e2affdc06ea8d828ecb6462614ce43e3 · L12–35。
- **T19-S04** [internal/worker/worker.go](https://github.com/miniflux/v2/blob/2.2.3/internal/worker/worker.go) · Blob 0ef0b01ff9d8ec957b32cd5041cf11f2e1232606 · L18–56。
- **T19-S05** [internal/reader/handler/handler.go](https://github.com/miniflux/v2/blob/2.2.3/internal/reader/handler/handler.go) · Blob 2e992b6771138c89572b94b849b53e362298f98e · L191–367。
- **T19-S06** [internal/reader/processor/processor.go](https://github.com/miniflux/v2/blob/2.2.3/internal/reader/processor/processor.go) · Blob 12ab44611e465b3f7c5a5acec73d6483d73df76c · L31–128。
- **T19-S07** [internal/storage/feed.go](https://github.com/miniflux/v2/blob/2.2.3/internal/storage/feed.go) · Blob decb4638ec3b93e8ba11b00f8817221b283f57be · L198–211,317–425。
- **T19-S08** [internal/storage/entry.go](https://github.com/miniflux/v2/blob/2.2.3/internal/storage/entry.go) · Blob f22a424bc1825d7cdb4c24e5d245216d91abff43 · L253–327。
- **T19-S09** [internal/model/feed.go](https://github.com/miniflux/v2/blob/2.2.3/internal/model/feed.go) · Blob 9f1de1eb50ddfa00f286635084a6fc56a5954907 · L93–137。

原知识定义：[K02.F2.1/F2.2](https://github.com/kaicreator-mm/tally/blob/0516b64b17da66f3029dbd34664b1328bf254a54/docs/versions/v2.1.2/software-engineering-knowledge/K02/systematic-syllabus.md#L40-L54)、[K02.F3.1](https://github.com/kaicreator-mm/tally/blob/0516b64b17da66f3029dbd34664b1328bf254a54/docs/versions/v2.1.2/software-engineering-knowledge/K02/systematic-syllabus.md#L67-L73)、[K03.F3.2/F3.3](https://github.com/kaicreator-mm/tally/blob/0516b64b17da66f3029dbd34664b1328bf254a54/docs/versions/v2.1.2/software-engineering-knowledge/K03/systematic-syllabus.md#L75-L89)。

反向增量：K02.F2.1、K02.F2.2、K02.F3.1、K03.F3.3→T19-C01（4 机制）；K03.F3.2→T19-C01（1 条件启发）。九个新 Blob 与前14份历史登记无重复。原165概念保持；不重复计数多个视图。候选 scheduler、batch_builder 路径不可读，周期启动/调度不纳入观察范围；实际 batch.go 和 handler.go 已取得。T19 不宣称整个 Miniflux 产品或运行验收；待独立审核。
