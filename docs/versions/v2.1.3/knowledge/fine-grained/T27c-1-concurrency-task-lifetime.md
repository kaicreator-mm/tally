# T27c-1 · 并发、异步、状态所有权和任务生命周期

归属 K03.F3.1–3/F4.2；#563 有限第三批任务，原稿 #7167352369，#588 延续跟踪，K02 状态合同、K10 运行恢复、K05 验证。复用原 [CON](../../../v2.1.2/software-engineering-knowledge/K03/advanced-concurrency-idempotency-proof-sketch.md)（blob b4064f7dfbbc972e849293653ea5a47b68edb819）L7–32 的线性化、事务边界、局部 CAS 与远端 Effect；[T27a](T27a-detailed-design-representation.md) 表示/frame、[T27b-1](T27b-1-relational-data-and-transactions.md) DB 隔离、[T27b-2](T27b-2-file-stream-and-durability.md) 流协议已有权威正文。此处补同步关系、争用、队列准入及结构化生命周期；不重新定义远端幂等或引入运行时。

## 1. 并发和并行的不同问题
并发是多个任务的进展交错，并行是同时执行。单线程事件循环可能有逻辑竞态，多线程共享内存还有数据竞争/可见性；多进程、数据库与网络还需要各自的同步和提交合同。race condition 是结果依赖未受控时序；数据竞争是特定内存模型下未正确排序的冲突访问。没有数据竞争不保证业务没有竞态。例如两个逻辑任务先检查某项目尚有一个名额，分别 await 后写登记；即便所有读写都在单线程，仍可能超出配额。加锁若只包住最后写，不保护“检查与修改”的联合前提，仍不足。先选不变量、共享对象和实际操作边界，再选同步方法。

## 2. Happens-before 与发布
Go 官方 memory model（2022-06-06 文本）以 sequenced-before 和 synchronized-before 的传递闭包描述 happens-before；无数据竞争的 Go 程序满足其顺序一致性保证。原子操作、锁、channel 各有版本/模型语义，不是跨语言通用内存屏障。教学：
```go
// ready := make(chan struct{})
// value := 0
// one goroutine:
value = 42
close(ready)
// another:
<-ready
// read value here
```
在只有上述写者且接收因 ready 已关闭而返回的条件下，写在 close 前，close 与相应接收同步，读取在接收后；由传递关系可建立 value=42。若写者在 close 后继续无同步修改 value，证明不再成立。依靠 sleep、日志先后、观察到 goroutine 启动而无相应同步不能建立此保证。无缓冲 channel 与有缓冲 channel 的反向同步规则不同，不能随意将 0 改 1 并维持原 rendezvous 推理。Go 的 Mutex Unlock 与随后 Lock 返回有同步关系，锁同一对象的所有读写路径才可据此保护数据；不要将 Go 规则推广到 JS、C++、Python、远端数据。

## 3. 同步机制选择与故障模型
| 机制 | 适合的对象和限制 |
|---|---|
| 单 owner/消息传递 | 私有状态由一个执行者维护；消息可并发提交，必须规定准入、顺序、回复和退出；不得把外部共享对象作为消息继续无同步写 |
| Mutex | 明确短临界区内多字段/读改写；避免等待外部回调或持有锁无界 IO；全读写路径遵守同一协议 |
| RWMutex | 多读场景的候选，不保证更快或公平；读→写升级与锁顺序按实际 API |
| CAS/版本条件 | 单权威对象可原子比较更新；冲突需重新读、有限重试；旧值 A→B→A 可使仅值比较漏掉历史变化，若语义需要版本则检查版本及回绕合同 |
| Semaphore | 限制同时持有许可的资源数；保证所有获取成功路径最终释放，等待者总数/内存/速率另控 |
| DB 事务 | 涉及持久数据、多行不变量参见 T27b-1；本地锁不能跨进程保护独立写路径 |

死锁是进展受循环等待或其他阻塞依赖阻断，例如 A 持 L1 等 L2，B 持 L2 等 L1；建立统一获取顺序只排除该类反序环，不保证任何程序无死锁。livelock（不断重试却无有效推进）、starvation（个体长时间得不到资源）与 deadlock 不同。候选处理：减少共享状态，明确顺序，避免持锁调用未知回调，必要时超时/有限重试/退避、资源归还与可观察停止。超时只改变等待者策略，不一定能撤销他人已完成效果；不以死锁检测或 race detector 未报错证明全局正确。

## 4. 准入、容量和任务状态
任务控制层状态可分为 PENDING（已受理未获执行许可）、ADMITTED（许可已分配）、RUNNING（用户工作已进入）、CANCEL_REQUESTED、SETTLED（带 outcome）。这不是强制状态机，实际系统还需拒绝、失效、持久受理、重试等。SETTLED 描述本地任务终态，不是业务效果合同。[T20-C03](../casebook/waves/T20-historical-abstraction.md) 已接受的 clearQueue 对照：已出队获得许可但尚未调用用户函数的任务不同于仍排队项；取消待办不能假定停止所有 admitted 项。[T19](../casebook/waves/T19-feed-refresh-architecture.md) 的有限 worker 与无缓冲通道仍可能有多个被阻塞的 producer；参见 [T18](../casebook/waves/T18-job-architecture-contrasts.md) 内存/持久队列选择。若 worker 数 C=4、待办上限 Q=16，假定任务只存在于这两个互斥状态集且在准入前拒绝超额，则控制器最多持有 C+Q=20 个任务。若请求层先生成无界 promise/生产者、另有重试队列或外部缓存，该数不再是端到端上限。并发限制、每时间窗速率、内存和队列等待期限分别设置。选择阻塞/拒绝/丢弃/持久待办须满足调用者合同，不能静默丢失合法受理的任务。

## 5. 结构化生命周期、取消与收敛
父任务启动子任务时明确 owner、失败传播、取消和 join。Python3.11 asyncio.TaskGroup 是具体实例：正常退出等待成员结束；成员非 CancelledError 失败会取消其他成员、等待后汇总异常（KeyboardInterrupt/SystemExit 有特殊处理）。协程吞掉 CancelledError 可能破坏结构化协议；在 finally 释放已获得资源、按合同传播原异常。它不自动接管在组外创建的任务或外部进程/远端事务，也不能保证拒绝取消的工作在指定期限终止。Go 官方 pipeline 教学中下游提前退出需要向上游传播停止，单纯增加小缓冲可能只是掩盖被堵塞发送者。每阶段拥有其输出通道；多生产者应由协调方确认全部发送者结束后关闭，不由任意接收者提前关闭。

有界停止设计：停止新准入→按合同取消或处理已有待办→请求运行任务停止→等待实际 settled/资源释放→记录未完成和外部 UNKNOWN→退出或有权升级处置。若只有 Promise.race 超时而无取消能力，子工作可能继续修改状态；记录“等待已超时”而非“已取消工作”。当有强制隔离手段时，仍须评估数据与外部效果。不要在任务重试循环内重复不可逆外部效果；CON 与 T27b-1 已有合同。

## 6. 教学反例和 Oracle（NOT_RUN）
- 以两个 barrier 控制同一配额读、释放后并行写，独立预期不变量与合法拒绝，而非按 sleep 等待。
- Go 发送/关闭与接收顺序；无同步写、缓冲变化和共享引用反例，race detector 仅作补充。
- 两锁反序、取消等待许可后是否正确归还，记录真实持有计数；不得以最终任务数掩盖泄漏。
- C=4/Q=16，分别对待办、admitted 未执行、running 做取消；检查所有 promise/task 有合同终态、未执行项未调用用户函数。
- 子任务失败、另一子任务进入清理、拒绝取消、远端 ACK 丢失，分别检查局部结果、资源状态和外部 UNKNOWN。

资料实读：Go [Memory Model](https://go.dev/ref/mem) L127–157、216–264、280–284；[Pipelines and cancellation](https://go.dev/blog/pipelines) Stopping short/Explicit cancellation（历史教学文章）；Python [3.11 asyncio TaskGroup](https://docs.python.org/3.11/library/asyncio-task.html#task-groups) L245–271。上述设计、容量推导与测试均原创教学 DOCUMENT_CANDIDATE，NOT_RUN；非全部 Go/Python/JS 内存/线程规范，不宣称真实产品已无竞态。
