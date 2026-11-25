# T27c-3 · 可靠消息、分区进度、Outbox 与消费事务

#620 / R4-B / DA-05；K03 主责消息及实现，K02 权威/架构、K09 演进、K05 独立验证、K10 运行。承接 [T27c-2](T27c-2-distributed-consistency-coordination-recovery.md)、[CON](../../../v2.1.2/software-engineering-knowledge/K03/advanced-concurrency-idempotency-proof-sketch.md)、[T18](../casebook/waves/T18-job-architecture-contrasts.md)、[AR-012](../casebook/architecture-cases.md) 的局部提交/未知效果/持久队列，不重写通用幂等定义。新增具体 Kafka 4.1 / Java client 4.1.2 合同以及原创教学组合。所有教学与 Oracle NOT_RUN；非实际部署、运行保证或新 qualified Claim/Case。

## 1. 先选语义，再选消息基础设施

短任务可直接调用；同一 DB 的持久待办/投影已足够时不必引入 broker（[T26a-2](T26a-2-architecture-style-composition.md)）。当多个独立消费者需按自己的进度持续消费/重放、有明确保留与容量要求，分区日志才值得评估。消息身份、业务动作身份、传输记录地址、消费者组和处理尝试是不同对象。命令表示请求，事件表示已发生的事实；不得从通知推断业务已提交。

Kafka 4.1 Design 的投递语义区分生产记录可靠性和消费后结果可靠性。按所选故障模型、保留窗口和最终目的地分别裁定。Kafka 每个 partition 有自己的 offset 序列；同组当前分配给一个消费者的分区可因再平衡交给另一成员，不能把当前分配当外部数据库的写入锁。不同 group 有独立消费进度；不同分区的 offset 不能比较成全局顺序。为同一 tenant+aggregate 的事件选择稳定分区 key 并约束实际发布顺序；分区数量/路由改变、多个发布者交错或重新发布可能破坏所需业务顺序。即使同分区日志有序，异步业务执行仍可能乱序；用业务版本和当前有权合同决定接受、等待缺口或重建，不按到达时间覆盖。

| 可观察事实 | 证据及不推出的结论 |
|---|---|
| send 返回 Future | 客户端已接纳发送调用，不代表 broker 已保存 |
| producer ACK | 按实际 acks/复制合同完成；不代表外部业务处理 |
| poll 返回记录 / position 前移 | 已取到记录，不等于已处理或 offset 已提交 |
| consumer DB commit | 该目的地事务实际完成，不代表 Kafka offset 已提交 |
| group committed offset | 下次恢复位置；不代表所有外部 Effect 合法发生 |
| 业务成功 | 以相应权威状态/回执证明，不能由上面任一单项替代 |

实际 Kafka 4.1 配置：acks=0 不等待服务器确认，acks=1 仅 leader 写入确认，acks=all 等待 ISR 确认；仍须核实际复制因子、min.insync.replicas、可用副本、选主/故障模型，不承诺任意灾难无丢失。显式 enable.idempotence=true 要求 acks=all、retries>0、max.in.flight.requests.per.connection≤5；冲突配置会拒绝（未显式启用时可能被关闭）。生产者会在同一 session 内对其自动重试去重，但应用重新调用 send 或新会话重发同一业务事件不受该保证；业务 eventId 和消费去重另设。send 超时/失败不能一律断言未追加。无幂等且多在途重试可改变分区中追加顺序。上述为指定版本的配置及 API 合同，非所有 broker。

## 2. 纸面消费进度与并行间隙

Java KafkaConsumer 4.1.2 的 position 随 poll 取得记录推进，committed position 是持久恢复位置；offset 可以有间隙，不是记录数量。以 partition P 当前 committed=40，poll 实际返回 offset40、41、43，下一位置44（42 无可返回记录的合法间隙，此例已知）。若 40、43 的业务提交已完成而 41 未完成，安全恢复位置最多推进至41，不能提交44（也不能按完成数量40+2=42）。41 合法处理后，三条均持久完成，才能使用本批 nextOffsets()=44；实际接口按 nextRecordToBeProcessed.offset() 或已耗尽分区的 nextOffsets()，带适用 leader epoch。不要将其它分区已完成进度连带晋升。未读记录与实际日志间隙不同，不能凭数字跳过尚待处理的记录。

教学简化：关闭 auto commit，按分区顺序处理，每条/批真实 DB 事务完成再明确提交相应 next offset；若处理较慢，可暂停相应分区、保持 poll/组协调，按已连续完成的实际记录前缀提交，不用最高完成 offset。max.poll.records 限一次返回数量，不是系统总 backlog；max.poll.interval.ms 超期可能再平衡。旧消费者 commitSync 失败时不得伪造推进或继续无权写入目的地；重新分配后查询真实进度、恢复和当前应用写许可。KafkaConsumer API 并非任意线程可并发使用；实际设计须由合法 poll 所有者协调提交与关闭。

## 3. 同一业务事务 Outbox 与消费 Inbox（原创教学）

复用 T26a-2 的 D0，增加外部独立索引消费者的获权要求。权威发布和 event 记录同 DB 同事务；不需要外部消费者时直接读取变更记录即可。以下选轮询 relay + Kafka + 目标事务 DB，不声称已有真实项目实现。租户 A 文档 v7→v8 时，四项权威事实（内容、当前指针、任务成功 outcome、变更记录/outbox）同一事务。eventId=E8 稳定且唯一于获权生产者作用域，key=(tenant,docId)，version=8，事件携带约定 schemaVersion 和可验证的规范内容/引用；同 eventId 不同规范语义是冲突，非重复成功。relay 只读已提交 outbox，按真实权限发送；确认后标记送达 broker 的收据，不据此清除唯一恢复证据。应用层重发 E8 可产生不同 Kafka offset 的两条记录，必须容忍目标重复。保留策略须覆盖实际重放、滞后与灾难恢复，不能清理去重记录后仍承诺历史重放不重复。

目标 DB 以 (logicalConsumer, producerScope, eventId) 唯一键及内容/版本比对识别已处理事件。同一事务插入/确认 Inbox 与应用投影状态、必要结果；约束与实际隔离须保证并发只有一次有效变更，不是先查再无条件写。重复相同事件只确认已有事实；不同内容同 ID 拒绝/隔离；未知 schema 或缺必要版本不写业务成功。若事件是完整快照并获权可跳版本，可在当前版本条件下更新；若是依赖上一版本的 delta，则缺口不能用“最新 wins”跳过，先补齐或从有权一致快照重建。此例索引使用完整快照且只允许按较新版本晋升；旧 E7 不覆盖 v8，授权仍由当前权威核。Inbox 记录应区分 applied、already-applied、合法 stale/no-op 等明确定义状态；未决不能标已处理。若无权跨过坏消息，暂停/报告；若明确获权隔离，须有真实持久隔离/可恢复收据才按合同推进，不能因反序列化异常直接丢弃。

| 崩溃窗口 | 独立预期与恢复（NOT_RUN） |
|---|---|
| 源事务提交前崩溃 | 无 v8、无 E8；不发送未提交事件 |
| 源事务已提交但 relay 尚未发送 | v8 与 E8 均存在；重启读取 E8 |
| broker 追加 E8 后 ACK 丢失/relay 未记送达 | 本地保留未确认；按允许的可重复事件语义重送同 E8 或核真实结果，不创 E9；可能多条 broker 记录 |
| poll E8 后目标事务前/中崩溃 | 无有效 Inbox+投影提交；从未越过的 offset 重试 |
| 目标事务已提交，offset 尚未确认 | 重放 E8，核已有 Inbox/版本不再次应用；再按连续已处理前缀提交 |
| offset 提前提交后崩溃（错误候选） | 可能恢复到 E8 之后但目标仍 v7；这正是反例，禁止 |
| E8 两条不同 offset 并发，或旧 E7 晚到 | 只一个合法有效变更，v8 不倒退；同 ID 异内容不吞掉 |
| 分区转移后旧执行者恢复，当前文档版本仍匹配 | Kafka 组资格/offset fence 不是目标写 fence；须在目标权威资源落实当前应用执行许可或证明合法的同一幂等事务合同。旧许可不得导致未获权效果；参 T26a-2 两个专门反例 |
| 保留期结束、所需 offset 已不可读 | 不自动 earliest/latest 当成功；明确缺口，按获权一致快照/重建或人工接管，不声称完整重放 |

正向纸面核算：源 v8 成功一次；E8 可能出现在 P offset40、41，消费两次，但目标仅一份有效 Inbox/投影提交，完成两条后 next offset42（假定下一可读位置42）。非法实现把每次消费当新动作，重复会新增两份业务结果；仅观察最终 v8 可能漏掉重复副作用，核实际事务/效果记录。若外部目标是邮件或付款且不能与 Inbox 同事务，必须另以实际 Provider 的稳定业务身份、幂等窗口/权威读回或获权补偿处理 UNKNOWN；不得先写 Inbox 假装远端已成功，亦不得盲重试。此例无外部不可逆效果。

## 4. Kafka 内事务与外部事务不可混淆

在 Kafka topic→处理→Kafka topic 的选定场景，可用 transactional producer，消费输入、产生输出，将相应 group offsets 通过 sendOffsetsToTransaction 纳入同一 Kafka transaction；输出消费者使用 read_committed，故 abort 的输出不会作为已提交记录交付，offset 也不晋升。abort 后应用须重新获取并定位已提交进度，不假定本地 poll 游标自动倒退；处理异常按 API 当前事务状态、错误种类和组资格确定恢复，不一律 abort/retry。transactional.id 对逻辑生产者稳定且活跃实例不得任意共用；具体 fencing/超时/配置与实际版本核验。commitTransaction 超时不证明 broker 未提交。此机制并不包含独立 SQL/HTTP 效果。

外部事务 DB 可选择把投影结果和恢复 offset 同一目标事务存储（自管恢复位置与 seek/分配、fence、重建均须设计），或者本例的事务 Inbox + 后置 Kafka offset。前者不自动处理多组或同 ID 不同 offset；后者允许重复交付，但要求业务效果去重。两者均不由“Kafka exactly-once”名义证明。选用时按重放需求、保留/恢复、外部资源支持、吞吐/复杂度及实际故障边界评价；不强制所有项目使用 outbox 或 Kafka。

## 5. 验证与来源

独立 Oracle 取源事务前后、outbox eventId/key/version、实际 broker topic/partition/offset/确认、目标 Inbox/投影事务、group commit/重启位置；对照故障矩阵、两分区与不同 group、offset 间隙及先完成高 offset、不同业务版本乱序、重放超出保留、旧消费者、授权变化。实际负荷下核积压、处理速率、暂停及恢复，见 [T33b-1](T33b-1-performance-queueing-capacity.md)；未测不称可靠性 SLA。协议与 schema 演进另见 [T25c-3](T25c-3-protocol-capability-negotiation.md)。

2026-11-26 实际读取官方 [Kafka 4.1 Design](https://kafka.apache.org/41/design/design/) Message Delivery Semantics L321–350；[KafkaConsumer 4.1.2 API](https://kafka.apache.org/41/javadoc/org/apache/kafka/clients/consumer/KafkaConsumer.html) L25–51、78–123（position/group/poll/manual offsets）；[KafkaProducer 4.1.2 API](https://kafka.apache.org/41/javadoc/org/apache/kafka/clients/producer/KafkaProducer.html) L32–78、207–209 及 commitTransaction 方法说明；[Producer Configs 4.1](https://kafka.apache.org/41/configuration/producer-configs/) L199–202、676–709、1063–1065。以上为版本化官方文档实际阅读范围，不是源码全量核验。Debezium 3.4 文档打开返回403，不作为已读证据；Outbox/Inbox 及数值/失败矩阵为本文原创教学推导，不宣称 Debezium 实现或运行。所有 Oracle NOT_RUN；DA-05 扩展目标候选待独审；既有真实案例和资格不增加。
