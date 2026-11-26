# T46 · RE-04 真实产品用户路径范围与证据登记

#664/#563；Tally base 091d875fb97e2ca3239d9ac1ead60648d21e807c，miniflux/v2 tag 2.2.3 实际 fetch_commit 解析 da951164d52e7e4f987c50a64f1d1be01e6e90e5（release notes for version 2.2.3），随后以该 SHA 读取源码。原 [T19](casebook/waves/T19-feed-refresh-architecture.md) blob bae2f3113aadbd00f1072e548969e6905d7089cc 实际读，选刷新子系统而非整个产品；本次选择同一产品但新增真实官方 Go client 查询未读→设置 read→重新查询/单条读取的完整有限用户路径，覆盖消费者、传输、路由与认证、校验、数据库范围/事务、响应和实际已有局部测试。选择理由：能以有限可复查真实源码贯通需求至维护，避免重写合成示例或逐文件审全产品。实际 [R1 路径](fine-grained/T46-R1-miniflux-real-client-read-status-path.md) 与 [R2 维护推导](fine-grained/T46-R2-cross-layer-maintenance-contract-reasoning.md) 各自分开源码事实、条件推论和新政策假设。不是上游运行、完整 Web UI 或所有产品领域覆盖。

## 实际源码定位（18个不同文件，S01–S18 仅本批局部键）

所有路径均为上述固定上游 SHA，blob 为实际读取返回，行号对应实际读段；不将文件存在当成全部文件已证明。来源 S13 与 T19-S08 同 blob，范围不同，不重复历史合格 Case/Claim/Rule 或登记数量。其他来源不声称与全历史登记无重叠。

| 局部键及固定文件 | Blob | 实际定位 | 用途 |
|---|---|---|---|
| S01 [README.md](https://github.com/miniflux/v2/blob/da951164d52e7e4f987c50a64f1d1be01e6e90e5/README.md) | `2dd666949852df5220aad72f2df33e0aec97db05` | 1–55 | 产品自述及公开文档指引 |
| S02 [client/client.go](https://github.com/miniflux/v2/blob/da951164d52e7e4f987c50a64f1d1be01e6e90e5/client/client.go) | `a07c05525a17e15da92a10840f54c8ef47bdc9eb` | 1–41,470–550,657–741 | 客户端初始化、单条/列表/更新方法、筛选编码 |
| S03 [client/request.go](https://github.com/miniflux/v2/blob/da951164d52e7e4f987c50a64f1d1be01e6e90e5/client/request.go) | `47e1ff59cc5e8c5e9964103f7592a55584874460` | 18–30,37–170 | HTTP、凭证、编码、状态/错误及超时 |
| S04 [client/model.go](https://github.com/miniflux/v2/blob/da951164d52e7e4f987c50a64f1d1be01e6e90e5/client/model.go) | `0a6cd4d1016c132a6308a0ee011332bdf15986bb` | 271–297 | Filter 和 EntryResultSet |
| S05 [internal/api/api.go](https://github.com/miniflux/v2/blob/da951164d52e7e4f987c50a64f1d1be01e6e90e5/internal/api/api.go) | `c717e6cbeaa63329991b6796df823a90e8708948` | 22–75 | /v1 路由及中间件、handler 注入 |
| S06 [internal/api/middleware.go](https://github.com/miniflux/v2/blob/da951164d52e7e4f987c50a64f1d1be01e6e90e5/internal/api/middleware.go) | `eeadc3a03bcbb3bf105ded707cc70aec6e9fa7c2` | 37–164 | API key/Basic 认证、身份写入与错误 |
| S07 [internal/http/request/context.go](https://github.com/miniflux/v2/blob/da951164d52e7e4f987c50a64f1d1be01e6e90e5/internal/http/request/context.go) | `ba6ee40de970289a5cb807e18cd3674251eb7cc4` | 16–21,57–65,157–173 | 认证状态与 userID 的 context 读取 |
| S08 [internal/storage/user.go](https://github.com/miniflux/v2/blob/da951164d52e7e4f987c50a64f1d1be01e6e90e5/internal/storage/user.go) | `64b6d503d7db63c2f2f78a552e8aa51a562c4a97` | 461–525 | API key 查用户的实际 SQL 与读取调用；未完全读 fetchUser 尾部 |
| S09 [internal/api/entry.go](https://github.com/miniflux/v2/blob/da951164d52e7e4f987c50a64f1d1be01e6e90e5/internal/api/entry.go) | `508bbee451ca5a8cd6a2b8b28d2452c837a8a53f` | 24–41,65–188,343–390 | 单条/列表、状态写入及补充筛选 |
| S10 [internal/validator/entry.go](https://github.com/miniflux/v2/blob/da951164d52e7e4f987c50a64f1d1be01e6e90e5/internal/validator/entry.go) | `f4fdb1162199494b6f47d2cdf0c33f489f607ab7` | 12–40 | 非空、合法状态及排序白名单 |
| S11 [internal/validator/validator.go](https://github.com/miniflux/v2/blob/da951164d52e7e4f987c50a64f1d1be01e6e90e5/internal/validator/validator.go) | `9b3cfd908b37a5ebbd1351588a1c7675a9031753` | 15–38 | 范围与方向 |
| S12 [internal/storage/entry_query_builder.go](https://github.com/miniflux/v2/blob/da951164d52e7e4f987c50a64f1d1be01e6e90e5/internal/storage/entry_query_builder.go) | `1245e1d4cba00b99b3bdb7907d5d9e847f9628fa` | 62–75,90–239,240–398,439–475 | 实际条件、排序、分页、列表/总数/单条与 scope |
| S13 [internal/storage/entry.go](https://github.com/miniflux/v2/blob/da951164d52e7e4f987c50a64f1d1be01e6e90e5/internal/storage/entry.go) | `f22a424bc1825d7cdb4c24e5d245216d91abff43` | 369–387 | 状态写入/受影响行条件；复用 T19-S08 同 blob 新读范围 |
| S14 [internal/database/migrations.go](https://github.com/miniflux/v2/blob/da951164d52e7e4f987c50a64f1d1be01e6e90e5/internal/database/migrations.go) | `8285038efa62d60d7a8bdd0215026dfb5b261585` | 12–23,66–92,220–234,376–402,785–803 | 实际初始化与后续选定索引迁移；非最终 schema 审计 |
| S15 [internal/model/entry.go](https://github.com/miniflux/v2/blob/da951164d52e7e4f987c50a64f1d1be01e6e90e5/internal/model/entry.go) | `db4958ca5478d93f4708ef37894c5f617c0859a1` | 10–40,70–80 | 状态、默认排序、实体及更新请求 |
| S16 [internal/http/response/json/json.go](https://github.com/miniflux/v2/blob/da951164d52e7e4f987c50a64f1d1be01e6e90e5/internal/http/response/json/json.go) | `8e99681a623d2e945cd8ca8e52b5307ddbc11158` | 18–41,50–113,157–173 | JSON 响应、204、400/500/401 |
| S17 [internal/http/response/builder.go](https://github.com/miniflux/v2/blob/da951164d52e7e4f987c50a64f1d1be01e6e90e5/internal/http/response/builder.go) | `db89654a1b641b59a93be96e2821d1c8f75f6c36` | 23–49,76–110,147–150 | 响应状态/写 header，默认200 |
| S18 [internal/validator/entry_test.go](https://github.com/miniflux/v2/blob/da951164d52e7e4f987c50a64f1d1be01e6e90e5/internal/validator/entry_test.go) | `035b49cdee26610800fc6c8609bd04d58d9708f9` | 12–60 | 已有 validator 测试断言；只读未执行 |

## 专业组合与停止条件

基于实际源码证明路径可追踪性，未执行任何 Miniflux、HTTP 请求、DB、客户端、测试、UI、部署或性能试验。读取失败的猜测路径（包括 internal/ui/entry.go、internal/ui/unread.go、internal/ui/entry_status.go、internal/ui/unread_page.go、client/client_test.go、client/request_test.go、internal/api/entry_test.go、internal/http/request/query.go、internal/api/responses.go）不作为依据，不因 404 推断对应功能或测试不存在。实际测试 S18 已取得。未继续穷举启动、用户密码校验内部、query 参数 parser、media proxy、enclosure、迁移 runner、最终 schema、Web UI 等，明确未核范围；不对上游安全/性能/产品缺陷作全局裁定。

RE-04 有限代表性完整用户路径候选已交付，待独审；不自动将专项主行 U 晋升 S，不为其它每种平台继续写全量教程。DA-08/HC-04 overall U 保留，RE-05 deferred/RE-06 excluded。T45 两章及 T44-RV-01 已在 [#662 收尾](https://github.com/kaicreator-mm/tally/issues/662#issuecomment-7198666299) 限定接受合入，无新 finding；62篇实际接受正文，本批2篇候选。正式50/284发现、67/86、S51/D0/U5仍历史状态；#653/#655原审核回执未找到，必须在 T47 集成前处理（原 reviewer 回填或明确替代审核，不能伪造 verdict）。不修改共享索引/ledger，不新增 qualified Case/Claim/Rule。T47 以实际关系、索引、冲突、重复与范围收敛为主，T48 最终知识阶段验收；不以目标执行代替知识验证，不进行 Package/DH/SDK、CI、Agent、LLM、产品实现或效果实验。#565 HOLD、#286 DEFERRED、A24/T4 NOT_RUN/HOLD。
