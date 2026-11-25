# T18 · 同功能的不同架构：同步、进程内异步、数据库队列

[#574](https://github.com/kaicreator-mm/tally/issues/574) / #563；基线 0fff9c230c86f7e421b332b7d9815ad77825c367。三个真实实现路径（两个 L4、一个 L3），同一逻辑能力：接受任务参数，调用既定任务行为。参考 Rails Active Job v8.0.0 与 Solid Queue v1.0.0 的 adapter 及其执行路径；不声称两版本实际组合已通过兼容验证。直接函数是分析基线而非新增来源案例。比较不能消除合同差异。全部运行及反例验证 NOT_RUN，维护成本为定性工程推断，不是实测。

## 增量目录

| ID / 粒度 | 机制 | 原知识关系与限定理由 | 来源 |
|---|---|---|---|
| T18-C01 / L4 | Inline：当前调用栈内执行任务 | K02.F2.2 机制：支持即时 enqueue 而明确拒绝 enqueue_at；K02.F2.3 机制：相同任务入口选择执行适配器 | S01/S03 |
| T18-C02 / L4 | Async：进程内线程池及调度 | K03.F3.3 机制：线程/队列预算与调度；K02.F2.2 机制：提交与实际执行的合同差异 | S01/S02 |
| T18-C03 / L3 | Solid Queue：数据库任务记录、认领与工作池 | K02.F3.1 机制：认领归属与事务包围的状态变更；K03.F3.3 机制：工作池可用线程限制认领；K03.F3.2 条件：外部效果后的失败窗口提示幂等需求，非现成 exactly-once 机制 | S04–S10 |

## T18-C01 · 同步执行：最小合同不应先被复杂架构替代

**源码观察：** Active Job Enqueuing L81–88 实例化或复用 job 并 enqueue；L112–138 运行 enqueue callbacks、按 scheduled_at 分派 adapter 的 enqueue/enqueue_at，raw_enqueue 正常返回后标记 successfully_enqueued，捕获 ActiveJob::EnqueueError 并记录。InlineAdapter L13–20 在 enqueue 调用内直接 Base.execute(job.serialize)，enqueue_at 抛 NotImplementedError。即使有统一任务 API，Inline 的延时能力并不存在；不将 successfully_enqueued 等同业务副作用证实。Active Job 文件还描述事务提交后入队行为及 per-job 配置，但所读 raw_enqueue 并非该完整集成实现；不能外推所有调用总在事务内或提交后执行。

**工程选择（原创）：** 如果任务短、调用者必须知道完成情况、无独立调度或可靠恢复要求，直接业务函数通常已够；如果真实存在同一 job 在不同环境使用共同协议、序列化或框架 callbacks 的需求，Inline 可复用现有接口。不要为了潜在异步需求添加队列和 worker。当前调用中的 execute 返回不证明外部效果成功；异常行为仍由执行器与 job 合同决定。

**反例/Oracle：** 必须未来某时执行的任务不能改为立即执行以伪装支持；调用者要求真实返回结果而非 job 对象时也不能把 adapter 返回当业务返回。验证完成等待、异常、延期拒绝、回滚时调用时点；NOT_RUN。

## T18-C02 · 进程内异步：隔离请求等待与任务执行，但不自动提供恢复

**源码观察：** AsyncAdapter 的 enqueue/ enqueue_at 包装 job；JobWrapper 构造 provider UUID 并 serialize，perform 调 Base.execute；Scheduler 默认 min_threads 0、max_threads 取 RAILS_MAX_THREADS（默认 5）、max_queue 0 注释为 unlimited、fallback_policy caller_runs，允许覆盖。普通 enqueue 经 executor.post；未来时间且非 immediate 时用 ScheduledTask，否则立即提交给 executor；immediate 模式选择 ImmediateExecutor。shutdown 可等待线程池停止。源码注释明确进程内 pending jobs 重启丢失且共享线程池的长任务可阻塞短任务。此片段不证明 executor 内部丢弃/拒绝/取消全部行为、独立 queue_name 隔离、真实延迟或吞吐；queue_name 在此 scheduler 接收但未用于分池。

**工程选择（原创）：** 可容忍任务丢失、需要调用者快速返回、任务可独立执行的场景可用内存线程池；需要在完成前响应时，必须改为“已接收/待完成”合同，另定义结果或查询。默认无限 backlog 与有限并发是不同指标；任务堆积时应按真实负荷和失败合同设定容量、拒绝或背压，不因线程上限就认定内存有界。CPU/IO 类型、调度公平性、共享资源约束需具体评估。若任务不允许重启丢失，不能仅调大线程数解决。

**反例/Oracle：** 进程在任务排队后被终止；大量慢任务后提交短任务；最大线程数已满且提交持续增加；验证是否接受/拒绝、待办数量、执行时点和重启后的真实结果，而不只判断 submit 返回成功。NOT_RUN。

## T18-C03 · 持久队列：任务生命周期成为独立状态，但不等于业务事务

**源码观察（实际路径）：**
1. SolidQueueAdapter L10–25 委托 Job.enqueue/enqueue_all，未来时间传 scheduled_at，声明 enqueue_after_transaction_commit? true。此方法声明不证明 Rails v8.0.0 当前配置实际读取它，未审计跨版本集成。
2. Job L12–65：单任务 create! 保存 queue、active_job_id、priority、scheduled_at、class_name、序列化 arguments、concurrency_key，赋 provider_job_id，ActiveRecord 错误包装为 SolidQueue::Job::EnqueueError；批量有显式 transaction、创建并 prepare、标记成功。Executable L8–15/60–72/102–105 after_create prepare，按 due 选择 dispatch/schedule；dispatch 根据 concurrency lock 进入 ready 或 block，ready 创建/寻找 ReadyExecution。schedule/锁具体实现未读。
3. Worker L26–39 按 pool.idle_threads 请求 ReadyExecution.claim，再 post 各认领记录；ReadyExecution L10–43 按队列关系和剩余 limit，在 transaction 中选择 ordered/limited/non_blocking_lock 候选，调用 ClaimedExecution.claiming；后者插入 job_id/process_id，读取记录并回调，删除对应 ready 记录。可证认领事务及状态迁移写入；未核 DB schema/锁实现、隔离、唯一约束及所有并发路径，不能保证所有部署绝不重复。
4. Pool post 先减可用线程，Future 内调用 execution.perform，并 ensure 归还/唤醒；线程池 max_threads/max_queue 均为 size、fallback abort。ClaimedExecution L60–105 execute 调 ActiveJob::Base.execute(job.arguments)，正常返回或捕获 Exception 转 Result，再 finished 或 failed_with；各有 transaction 移除 claim，finished 调 job.finished!（Executable L78–84 依设置记 finished_at 或 destroy）。release 在 transaction 中再次 dispatch 并销毁 claim；另有 fail_all_with 与 orphaned scope，不能从存在这些方法推出自动崩溃恢复时限、必然重试或至少一次投递。成功执行返回与任务状态记账不是单个外部原子提交。

**工程选择（原创）：** 有真实的持久待办、独立 worker、延期或可查询恢复需求时，采用数据库队列是候选；要核数据库持久性配置、部署拓扑、恢复策略、任务积压、重复处理、版本兼容与生命周期。数据库入队不是外部效果完成；如果业务提交后再入队，需考虑两次提交之间的丢失窗口；若任务记录与业务更新需要同事务，必须核实实际连接、事务和提交时序，不能由 adapter 名称决定。外部支付/邮件等按实际允许丢失和重复的合同选择恢复策略：允许丢失且禁止重复时可不重放 UNKNOWN（仍须核实实际执行/重试路径）；要求可靠重放且不重复效果时须核实稳定业务身份、去重作用域/期限、原子条件提交或远端幂等/可信回读与恢复政策，不能靠本地 key 或认领事务实现全局 exactly-once；无法确认时保留 UNKNOWN、停止或人工处理。若只有本地短操作且不要求延迟/恢复，这些表、worker、迁移、告警及恢复流程是额外维护面。

**反例/Oracle：** worker 在认领后执行前崩溃；外部效果成功但 finished 记录前崩溃；业务事务提交而队列写入失败；两个 worker 同时认领同一候选；数据库连接丢失导致提交结果 UNKNOWN。按实际策略验证保留记录、可查状态、重试或人工处置、外部效果去重，不默认自动恢复或零重复。另检查 job 版本迁移与参数可反序列化；NOT_RUN。

## 统一需求下的选择表（原创综合，非额外案例）

情境：订单生成后需计算报表或发送通知，任务输入含业务 ID，明确调用者等待与失败责任。以下成本、替代方式和选择均为推导，不是项目性能实测。

| 维度 | 直接函数 | Inline | Async | Solid Queue |
|---|---|---|---|---|
| 调用边界 | 业务函数返回 | adapter 内执行 | 通常提交给 executor；immediate/回退可能同步 | 任务记录与 worker 执行分离 |
| 延期 | 需额外机制 | 明确不支持 | 进程内调度 | 接收 scheduled_at，准备路径区分 due；调度实现待核 |
| 待办恢复 | 无独立待办 | 无独立待办 | pending 重启丢失 | 数据库记录提供恢复基础；策略/保证待核 |
| 背压 | 调用者自身约束 | 同步调用者约束 | 默认无限队列，线程预算独立 | 按 idle 认领、工作池有限；数据库总 backlog 未由此限制 |
| 成功含义 | 按业务合同 | execute 返回不等于外部成功 | 提交不等于执行 | 入队/认领/执行/记账/外部效果分别 |
| 维护面 | 最少引入机制 | 序列化与框架合同 | 线程、排队、关停 | DB 表/迁移/worker/认领/监控/恢复 |

选取顺序：先明确任务完成与调用响应之间关系、允许丢失和重复的业务后果、等待期限和真实恢复需求；仅在需要多执行环境或独立调度时引入共同 job adapter；以负荷预算决定并发/背压，以持久性需求决定队列存储；对跨边界失败保留 UNKNOWN 并按实际业务证据对账。只有真实需求变化才升级架构，不能以队列/微服务的存在证明质量。示例：可重算的临时报表且允许丢失可考虑 in-process；必须保留已接受的报表请求则需要持久待办与恢复合同；通知若允许丢失且禁止重复，可采用不重放 UNKNOWN 的策略；若要求可靠重放且不重复效果，则需上述经验证的去重或回读等机制，不能仅由“持久”推出。无实际授权政策或负荷数据时不得宣称某方案最优。

## 来源登记（全部为本轮实际读取源码）

- **T18-S01** [rails/rails ActiveJob enqueuing.rb](https://github.com/rails/rails/blob/v8.0.0/activejob/lib/active_job/enqueuing.rb) · v8.0.0 · Blob a8f1a887139520195c2b81ec163d4af459933d9d · L43–88,112–138。
- **T18-S02** [rails/rails async_adapter.rb](https://github.com/rails/rails/blob/v8.0.0/activejob/lib/active_job/queue_adapters/async_adapter.rb) · v8.0.0 · Blob 54be09aefc4421ce3d7c02fe817809d80b2161fe · L12–32,33–112。
- **T18-S03** [rails/rails inline_adapter.rb](https://github.com/rails/rails/blob/v8.0.0/activejob/lib/active_job/queue_adapters/inline_adapter.rb) · v8.0.0 · Blob d75bab8cfa2bc233eb061bfc8323a42e728786ca · L13–21。
- **T18-S04** [rails/solid_queue solid_queue_adapter.rb](https://github.com/rails/solid_queue/blob/v1.0.0/lib/active_job/queue_adapters/solid_queue_adapter.rb) · v1.0.0 · Blob d30421947a8f8a1627cd9d7c84ef3e2a8149c00c · L10–25。
- **T18-S05** [rails/solid_queue job.rb](https://github.com/rails/solid_queue/blob/v1.0.0/app/models/solid_queue/job.rb) · v1.0.0 · Blob 8574c1ec9ab4b31efcab5a23a4390670e95c3f62 · L12–65。
- **T18-S06** [rails/solid_queue job/executable.rb](https://github.com/rails/solid_queue/blob/v1.0.0/app/models/solid_queue/job/executable.rb) · v1.0.0 · Blob e2146a67649ae5f6f84156ea5b9ef3de7c921b4b · L8–15,60–84,102–105。
- **T18-S07** [rails/solid_queue ready_execution.rb](https://github.com/rails/solid_queue/blob/v1.0.0/app/models/solid_queue/ready_execution.rb) · v1.0.0 · Blob 35a11292c43fa6a18f01020cb181967d293d1517 · L10–43。
- **T18-S08** [rails/solid_queue claimed_execution.rb](https://github.com/rails/solid_queue/blob/v1.0.0/app/models/solid_queue/claimed_execution.rb) · v1.0.0 · Blob d4abf45aba8d74fbb813f2923df5bde0480b6671 · L3–47,60–105。
- **T18-S09** [rails/solid_queue worker.rb](https://github.com/rails/solid_queue/blob/v1.0.0/lib/solid_queue/worker.rb) · v1.0.0 · Blob fc203774d6aa5cba7774af7ff71a81febb687858 · L12–19,26–47。
- **T18-S10** [rails/solid_queue pool.rb](https://github.com/rails/solid_queue/blob/v1.0.0/lib/solid_queue/pool.rb) · v1.0.0 · Blob c1bcf19507ceef81ee6359ec6fd01ba0abf315e6 · 全文件（post/idle_threads/executor）。

原定义：[K02.F2.2](https://github.com/kaicreator-mm/tally/blob/0fff9c230c86f7e421b332b7d9815ad77825c367/docs/versions/v2.1.2/software-engineering-knowledge/K02/systematic-syllabus.md#L48)、[K02.F2.3](https://github.com/kaicreator-mm/tally/blob/0fff9c230c86f7e421b332b7d9815ad77825c367/docs/versions/v2.1.2/software-engineering-knowledge/K02/systematic-syllabus.md#L56)、[K02.F3.1](https://github.com/kaicreator-mm/tally/blob/0fff9c230c86f7e421b332b7d9815ad77825c367/docs/versions/v2.1.2/software-engineering-knowledge/K02/systematic-syllabus.md#L67)、[K03.F3.2](https://github.com/kaicreator-mm/tally/blob/0fff9c230c86f7e421b332b7d9815ad77825c367/docs/versions/v2.1.2/software-engineering-knowledge/K03/systematic-syllabus.md#L75)、[K03.F3.3](https://github.com/kaicreator-mm/tally/blob/0fff9c230c86f7e421b332b7d9815ad77825c367/docs/versions/v2.1.2/software-engineering-knowledge/K03/systematic-syllabus.md#L83)。

反向增量：K02.F2.2→C01/C02 机制；K02.F2.3→C01 机制；K02.F3.1→C03 机制；K03.F3.3→C02/C03 机制；K03.F3.2→C03 条件启发。六机制/一条件；原165概念不变。十个新 Blob 在13份既有来源登记中均未出现。待独立限定复核；保留 T13/T15–T17；未执行项目、无基准测试/兼容测试/CI/Agent/新软件。
