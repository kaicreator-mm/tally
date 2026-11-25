# T04 · AI/Agent、检索、数据库、分布式及嵌入式系统案例

[#567](https://github.com/kaicreator-mm/tally/issues/567)；9 个案例、8 个实际来源文件、8 个项目。六类真实局部视图（AI 推理、Agent、搜索、数据库、分布式、RTOS/IoT）；所有案例 L2–L5，源文档/实际实现与本库推论区分。未运行任何外部项目、未编译/压测；验证意图 NOT_RUN，不复用代码或声称完整架构审计。

## T04-C01 · vLLM：以计算进度和资源预算组织推理调度

**分类：** AI 模型推理 / L2–L3 / SOURCE_OBSERVED_SCOPED。**功能/机制：** Token 预算、请求进度、调度与执行分离。**知识：** K03.F3.3。**来源：** T04-S01。**关系：** 机制关联；有界请求调度与预算。验证意图另见正文，NOT_RUN。

- **实际观察：** V1 Scheduler.schedule 的注释说明以已计算 token 与目标 token 数推进请求，而非在调度层固定 prefill/decode 两阶段；实际创建新/恢复/运行/抢占请求集合，设置 token 与 encoder budget，并先遍历运行请求直到预算不足。
- **职责、控制/数据流及状态/资源视图（分析）：** 请求状态→调度器分配本轮工作/缓存资源→执行侧（此处仅观察调度输入输出边界，未读执行器）→下一轮进度。统一的是进度和预算，而非假定所有模型计算、缓存及输出策略相同。
- **适用边界：** 多请求共享有限 GPU/缓存、请求长度不同；单请求本地推理或低负载 CLI 不一定需要连续批调度。
- **备选与反转：** 直接执行、静态 batch、动态有界调度；按真实内存、延迟、公平性、吞吐与取消条件选择，不能从源码推断绝对性能。
- **验证/反例（NOT_RUN）：** 超长请求、预算不足、新/恢复请求、抢占、取消与缓存回收；分别核对容量与请求身份，不能仅凭生成成功证明公平性。

## T04-C02 · LangGraph：状态图、局部更新及执行边界

**分类：** Agent/工作流 / L2–L3 / SOURCE_OBSERVED_SCOPED。**功能/机制：** 共享状态、reducer、编译、checkpoint 与中断。**知识：** K03.F3.1。**来源：** T04-S02。**关系：** 机制关联；状态图及局部状态合并。验证意图另见正文，NOT_RUN。

- **实际观察：** StateGraph 源文档定义节点读取 state、输出 partial state；各 key 可定义 reducer。compile 提供 checkpointer/store 和节点前后中断选项，并调用 validate；文档说明 checkpoint 可用于暂停、恢复、重放。
- **职责、控制/数据流及状态/资源视图（分析）：** 图定义、状态合并政策、执行与持久 checkpoint 分别表达。Reducer 的确定性/冲突规则归具体状态键；外部任务事实和作用结果需由相应权威确认，而不是由可重放状态自动赋权。
- **适用边界：** 有明确多步骤、条件转移、状态恢复与共享数据；简单固定任务用直接顺序调用或有限 worklist 更清晰。
- **备选与反转：** 普通函数组合、有限状态机、图工作流；真正有持久恢复才选择 checkpoint，需明确不可重放副作用。
- **验证/反例（NOT_RUN）：** 并行节点冲突、遗漏状态、恢复时外部副作用已经发生、节点中断；只观察 API/校验入口，不能证明分布式执行或 exactly-once。

## T04-C03 · Meilisearch：搜索入口的授权、索引与资源控制

**分类：** 搜索服务 / L2–L4 / SOURCE_OBSERVED_SCOPED。**功能/机制：** 查询参数、租户过滤、索引选择、并发许可。**知识：** K02.F4.3。**来源：** T04-S03。**关系：** 机制关联；租户查询范围与资源威胁。验证意图另见正文，NOT_RUN。

- **实际观察：** GET search 路径使用带 SEARCH action policy 的 index scheduler，转换 index UID 和参数，将租户 token 规则加入 query.filter，获取 index 和搜索类型，再申请 permit、spawn_blocking 执行搜索、释放许可并返回结果/错误。
- **职责、控制/数据流及状态/资源视图（分析）：** 请求边界处理参数、权限相关约束和索引身份；队列控制并发资源；搜索核心执行查询。授权筛选与排名/召回是不同机制，不能用搜索结果后置过滤代替真实执行范围。禁止披露/返回越权文档或使其占据授权 top-k；若业务明确要求跨租户排名统计、时延等强非干扰，另定义观察范围并验证索引/资源隔离。共享 BM25 统计可能改变合法分数，当前入口不证明强非干扰。
- **适用边界：** 有多索引、租户约束及并发资源竞争；少量公开本地文档可直接调用库。
- **备选与反转：** 嵌入搜索库、独立搜索服务、隔离索引/共享索引加明确过滤；依据真实授权、容量和部署要求。
- **验证/反例（NOT_RUN）：** 非法参数、跨租户规则、错误索引、permit 耗尽、执行失败后的释放；局部入口不证明所有权限及取消路径都正确。

## T04-C04 · Lucene：搜索算法与结果收集接口

**分类：** 全文搜索库 / L3–L5 / SOURCE_OBSERVED_SCOPED。**功能/机制：** Query、TopDocs、Collector、超时及并发。**知识：** K03.F1.2。**来源：** T04-S04。**关系：** 机制关联；索引、查询与结果收集。验证意图另见正文，NOT_RUN。

- **实际观察：** IndexSearcher 声明 query cache/policy、timeout、partialResult；search(query,n) 委托 searchAfter，另有 Collector 低层入口，源码注释推荐支持并发的 CollectorManager 版本。
- **职责、控制/数据流及状态/资源视图（分析）：** 查询表达、索引读取、匹配收集和结果输出可以分开；top-n 与遍历所有匹配不是同一合同，超时或部分结果不能当全量无命中。
- **适用边界：** 复杂排名、分页/聚合或并发查询；几十条内存数据可直接过滤排序。
- **备选与反转：** 直接扫描、索引 top-k、按用途专门收集器；公开接口应明确计数、排序、分页/范围和截断。
- **验证/反例（NOT_RUN）：** 并列分数、空查询、超时、限制 n、部分结果及 Collector 语义；未检查全部具体 partialResult 传播，也无实测速度。

## T04-C05 · PostgreSQL：用户事务、命令处理与底层事务

**分类：** 关系数据库 / L2–L3 / PROJECT_DOCUMENTED。**功能/机制：** 三层事务、savepoint、命令错误处理。**知识：** K03.F4.1。**来源：** T04-S05。**关系：** 机制关联；事务层与 savepoint。验证意图另见正文，NOT_RUN。

- **实际观察：** 内部 README 描述三层：底层事务/子事务、主循环控制、用户可见事务/savepoint；SQL BEGIN/COMMIT/ROLLBACK 等进入上层控制，按当前状态调用底层操作。
- **职责、控制/数据流及状态/资源视图（分析）：** 用户事务语义→命令状态协调→底层资源/事务机制。不同层抽象处理的状态不同；不能用单一 start/finish 函数省略用户事务语义。
- **适用边界：** 事务可跨多命令，存在 savepoint 及失败后状态；单个原子键更新不需要实现数据库事务内核。
- **备选与反转：** 数据库提供事务、应用层窄 transaction scope、跨系统业务协调；最后一种不自动继承 DB 原子性。
- **验证/反例（NOT_RUN）：** 显式/隐式事务、命令错误、savepoint、回滚后继续；文档不是所有隔离级别和并发故障的完整证明。

## T04-C06 · PostgreSQL：WAL 与页面持久化的时序合同

**分类：** 存储引擎/恢复 / L2–L4 / PROJECT_DOCUMENTED。**功能/机制：** 写前日志、LSN、持久化顺序、重放。**知识：** K03.F4.1。**来源：** T04-S05。**关系：** 机制关联；WAL 与数据库持久顺序。验证意图另见正文，NOT_RUN。

- **实际观察：** 内部文档说明 WAL 在相应数据页前到达稳定存储，页面 LSN 标记最新相关记录；shared-buffer manager 写脏页前确保 WAL 至少刷新到对应 LSN；重放比较页面 LSN 判断变更是否已应用，并指出原子页写假设的局限。
- **职责、控制/数据流及状态/资源视图（分析）：** 日志记录→持久化顺序约束→数据页写入→恢复读取/重放。日志、数据页面和可见业务提交不是一回事；幂等重放依赖具体 LSN 与记录语义。
- **适用边界：** 存储需崩溃恢复且有明确持久介质合同；不需要持久化的临时计算无需 WAL。
- **备选与反转：** 直接原子替换、小型持久日志、成熟数据库；优先复用真实数据库，不因案例自行实现不完整事务系统。
- **验证/反例（NOT_RUN）：** 在日志/页写不同阶段故障、重复重放、撕裂写；检查真实文件系统/硬件保证。未执行故障注入，不声称某配置断电安全。

## T04-C07 · etcd：共识消息的运输边界与成员身份

**分类：** 分布式系统 / L2–L3 / SOURCE_OBSERVED_SCOPED。**功能/机制：** Raft 消息、peer/remote、运输与共识分离。**知识：** K02.F2.2。**来源：** T04-S06。**关系：** 机制关联；共识消息运输接口与身份。验证意图另见正文，NOT_RUN。

- **实际观察：** Transport 有本地 member ID、连接相关 TLSInfo 和重试频率配置；Send 根据目标 ID，在读锁保护下取得 peer/remote 后分派；To=0 被忽略，未知目标记录 debug。
- **职责、控制/数据流及状态/资源视图（分析）：** 共识层生成消息→运输层按成员身份路由→对端处理（未读接收/共识状态机）。发送或传输 ACK 不等于日志已提交或业务操作已生效。
- **适用边界：** 多个节点需共识和成员生命周期；单机任务或无共享一致性需求不应用 Raft。
- **备选与反转：** 单机持久状态、中心服务、成熟共识库+独立运输；由故障模型和 quorum 需求决定，不照抄内部 transport。
- **验证/反例（NOT_RUN）：** 未知/已移除 peer、网络分区、重复/迟到消息与成员变更；此处仅运输分派，不证明 Raft 正确性、线性一致性或完整集群架构。

## T04-C08 · Zephyr：线程就绪状态与优先级调度

**分类：** 嵌入式/实时系统 / L2–L4 / PROJECT_DOCUMENTED。**功能/机制：** 线程、ISR、调度点、优先级、deadline。**知识：** K03.F3.3。**来源：** T04-S07。**关系：** 机制关联；线程就绪与优先级调度。验证意图另见正文，NOT_RUN。

- **实际观察：** 内核文档描述在等待/就绪/中断返回/yield 等重调度点选择线程，优先级高的 ready 线程先运行，同优先级按等待时间；可选 deadline 配置影响同静态优先级选择；ISR 可抢占线程。
- **职责、控制/数据流及状态/资源视图（分析）：** 事件/中断→内核调度状态→选择 ready 线程→运行/等待。就绪、当前执行、完成是不同事实；资源受限的实时系统需明确定时与抢占条件。
- **适用边界：** 存在多个任务、实时响应/资源限制；简单 MCU 主循环可用事件或明确状态机。
- **备选与反转：** 轮询、事件循环、RTOS 任务；安全关键控制须以专业实时/硬件约束单独分析，不能由 API 存在宣称满足 deadline。
- **验证/反例（NOT_RUN）：** 高优先级持续就绪、同优先级、等待/唤醒、ISR 干扰、实际时限；未运行目标板，不提供实时性保证。

## T04-C09 · Home Assistant：配置实体与集成激活状态

**分类：** IoT/设备平台 / L2–L4 / SOURCE_OBSERVED_SCOPED。**功能/机制：** 配置身份、运行数据、加载能力、状态和锁。**知识：** K03.F3.1。**来源：** T04-S08。**关系：** 机制关联；配置实体与激活状态。验证意图另见正文，NOT_RUN。

- **实际观察：** ConfigEntry 声明 entry_id/domain/data/options/runtime_data/state；async_setup 对忽略/禁用项返回，设置当前 entry 上下文并 finally 清除；内部 setup 对已 loaded/进行中及未持锁的主 integration 拒绝，设置进度状态并查询卸载/移除支持。
- **职责、控制/数据流及状态/资源视图（分析）：** 持久配置→集成查找及能力→受控激活→运行资源/状态。配置存在不等于已激活，运行数据不应混成可持久配置；卸载能力需单独确认。
- **适用边界：** 多个可选设备集成、可重载/停用并具有资源生命周期；简单单实例 CLI 可直接构造/关闭。
- **备选与反转：** 显式对象生命周期、静态模块组合、受控插件激活；只在真实扩展需求下支持动态加载。
- **验证/反例（NOT_RUN）：** 禁用项、重复 setup、缺锁、导入失败、卸载不支持和资源清理；未读取完整状态/退出实现，不承诺全部生命周期正确。

## 实际来源登记

- **T04-S01**：[vllm-project/vllm / vllm/v1/core/sched/scheduler.py](https://github.com/vllm-project/vllm/blob/v0.8.5/vllm/v1/core/sched/scheduler.py) · ref `v0.8.5` · Blob `60c6a0f00600ee24b5794c3e0c410f16f97b410b` · L35–44, 137–183。Scheduler 实际 token budget、请求进度、运行队列与 encoder budget。
- **T04-S02**：[langchain-ai/langgraph / libs/langgraph/langgraph/graph/state.py](https://github.com/langchain-ai/langgraph/blob/0.2.76/libs/langgraph/langgraph/graph/state.py) · ref `0.2.76` · Blob `a48356df5d6f9d5119ec8970dac45dada3019a8a` · L96–107, 506–545。StateGraph 源文档共享 state/reducer 与 compile/checkpointer/interrupt 选项、校验调用。
- **T04-S03**：[meilisearch/meilisearch / crates/meilisearch/src/routes/indexes/search.rs](https://github.com/meilisearch/meilisearch/blob/v1.13.0/crates/meilisearch/src/routes/indexes/search.rs) · ref `v1.13.0` · Blob `333ae19449213745c48be5fcaf200196e871ef0e` · L59–70, 318–365。搜索入口实际参数转换、租户规则、索引、并发许可与搜索执行。
- **T04-S04**：[apache/lucene / lucene/core/src/java/org/apache/lucene/search/IndexSearcher.java](https://github.com/apache/lucene/blob/releases/lucene/9.12.0/lucene/core/src/java/org/apache/lucene/search/IndexSearcher.java) · ref `releases/lucene/9.12.0` · Blob `9f0d5b71a08ec8c13de151210a51e8ad6978723a` · L79–90, 524–558。搜索器缓存/超时字段与 TopDocs、Collector 入口合同。
- **T04-S05**：[postgres/postgres / src/backend/access/transam/README](https://github.com/postgres/postgres/blob/REL_17_0/src/backend/access/transam/README) · ref `REL_17_0` · Blob `28d196cf62b33baa08b88004548d09d197aa9f47` · L3–40, 402–428。项目内部设计文档：事务层次及 WAL/LSN、持久顺序与故障假设。
- **T04-S06**：[etcd-io/etcd / server/etcdserver/api/rafthttp/transport.go](https://github.com/etcd-io/etcd/blob/v3.5.17/server/etcdserver/api/rafthttp/transport.go) · ref `v3.5.17` · Blob `a1969549a92c3a4f936e8e6d7e233d308ed81712` · L97–108, 175–209。Raft HTTP 运输层成员身份、peer 查找及 Send 分派。
- **T04-S07**：[zephyrproject-rtos/zephyr / doc/kernel/services/scheduling/index.rst](https://github.com/zephyrproject-rtos/zephyr/blob/v4.0.0/doc/kernel/services/scheduling/index.rst) · ref `v4.0.0` · Blob `87823ad40bf2930e0da966056b9f0a56f647314d` · L3–58, 146–156。项目调度文档：重调度点、ready/priority、ISR 与可选 deadline。
- **T04-S08**：[home-assistant/core / homeassistant/config_entries.py](https://github.com/home-assistant/core/blob/2025.1.0/homeassistant/config_entries.py) · ref `2025.1.0` · Blob `ade4cd855cac21c9e7f484adae4384f4aa4aaa04` · L316–325, 539–600。配置条目的持久数据与 runtime_data、异步 setup、状态/锁及能力检查。

## 交接及缺口

至少四类真实架构参考的限定范围已满足；模型完整执行器/多卡拓扑、Agent 持久恢复实际语义、搜索索引更新、数据库完整查询链、共识安全性与实时板级证据未系统审查。llama.cpp 候选 b4500 examples/server/README.md 未获得可读内容，未计入，vLLM 为实际替代；Zephyr 另一个线程文档只作候选发现，不计入本批 8 个实际引用文件。T04 DELIVERED_SCOPED，待 T14 独立审查。T05/T06/T08 可分别用于资源预算/缓存、搜索与持久化、生命周期比较，T10/T11 可结合真实故障模型与版本变化。