# T27c-2 · 分布式一致性、协调与恢复边界

#602；K03 主责，K02 合同、K10 恢复、K05 验证。深化 DA04/05；复用 [CON](../../../v2.1.2/software-engineering-knowledge/K03/advanced-concurrency-idempotency-proof-sketch.md)、[T27c-1](T27c-1-concurrency-task-lifetime.md)、[T27b-1](T27b-1-relational-data-and-transactions.md)、[T25c-2](T25c-2-network-transport-and-protocol.md)、[T25a-1](T25a-1-discrete-logic-relations-graphs.md)。以下是固定成员、非 Byzantine、受控故障的教学机制；具体 API 例限定 etcd v3.6；全部例及 Oracle NOT_RUN。

## 1. 从可观察合同选择一致性，而非从拓扑选择

线性化要求单个抽象操作存在于调用与响应之间的合法线性化点，并保持非重叠操作的实时先后；串行化只要求事务历史等价于某串行历史，不单独要求实时先后；严格串行化把两者结合到事务历史。并发操作允许多个合法次序，不要求读取得响应瞬间的全局最新值。最终一致性需声明写停止、通信/修复继续及冲突解决等条件；不是有界陈旧、read-your-writes 或单调读的同义词。例：W(x=7) 已响应后才开始 R，且无其它写，R=6 违反线性化；若 R 与 W 重叠，则 6/7 均可能合法（初值6）。两个键各自的读可能跨越同一事务的版本边界；分别正确并不能证明多键快照。UI 展示可接受陈旧值的合同，不自动授权库存扣减使用同一陈旧值。

分区时两个隔离侧若都接受互相冲突的操作且要求立即成功，同时又要求全局唯一名额，则没有通信无法确认另一侧是否已授出名额。可选等待/拒绝、事先划分可独立分配的额度、或放宽语义并定义冲突处置。额度分配自身需要正确约束；不能把拒绝服务称为业务可用，也不能把 CAP 口诀当平时延迟推导。单节点与单数据库事务足够时，复制协调不是必选。

## 2. 多数集合与复制状态机

固定 N 个投票成员，q=floor(N/2)+1，任意两个大小至少 q 的集合交集至少 2q−N>0（由并集不超过 N）。N=3 时 q=2，可失去1个；N=4 时 q=3，也只能失去1个；N=5 时 q=3，可失去2个。这里假定余下多数能互通且服务/持久存储正常，并不保证必在给定时间内完成。5节点分为3+2时只有3侧有多数；分为2+2+1时均没有多数。相交本身不能证明共识安全：交点若可对冲突值双重承诺、丢失持久记录，或成员集合被未经协议改变，推导即不足。不能把 quorum ACK 当任意副本已经应用。

复制状态机概念：协议先决定日志位置与命令顺序，副本按已提交前缀应用确定性状态转移；随机数/当前时间等非确定性输入若影响结果，应成为已决定的命令事实或有明确确定性机制。领导者/任期、投票限制、日志匹配、提交规则和持久化需要具体协议保证；多数副本存有某条目不是 Raft 任意旧任期条目的完整提交规则。本章不实现或证明 Raft。etcd v3.6 故障说明区分已提交写与旧领导者未提交写；选举过程可能使客户端超时，不能由超时推出未提交。其多数丢失停止接受写；分区恢复及成员改变依赖协议，不是任意重建副本。

## 3. 读、Watch 与版本化投影

etcd v3.6 默认 KV 操作的实时顺序与持久承诺见 API guarantees；可选 serializable Range 允许读到较旧的已持久状态，不能把它等同 PostgreSQL SERIALIZABLE。Watch 在可用历史窗口内按 revision 提供有序、完整 revision 的事件和恢复机制，但不是线性化读，也无确定交付期限。多个键同 revision 可能来自一次事务；按单键事件立刻发布跨键视图，可能暴露部分更新。嵌套事务内事件顺序不能假定等于请求排列。

教学投影合同：先取得同一 scope 的一致快照及响应 revision r（分页要固定同一历史 revision；若历史已压缩则重建），原子发布全量基线；再从 r+1 建立 watch。按完整 revision 更新投影，与最后已应用 revision 一并原子持久化；断线从该持久位置+1 恢复而非最后已接收位置，避免“收到但未应用”丢失；重放按已应用身份处理。只监听部分键不能以 revision 不连续直接认定漏事件，因为其它键也能增加 revision。需要进度时使用具体 watch 的进度通知并限定 scope，不能把空闲时间当已追平。若需要当下权威决定则直接合适的线性化读/事务，不依据 watch 缓存的最近接收时间。

例：r=10 快照，r=11 同时改 A、B，r=12 只改其它 scope。应用11的一半后崩溃，若已写 checkpoint=11，恢复从12将漏另一半。原子投影+checkpoint 或可独立证明的幂等重放修复这一类失败。若 r+1 已压缩，重新获得完整一致快照及新 revision 再追增量；不把缺口当没有变更。恢复灾难快照是新逻辑集群，不能沿用旧集群 revision 身份；etcd v3.6 文档提供 revision bump/mark-compacted 使旧 watch 失效的恢复方式，实际 bump 必须覆盖已暴露的最大 revision 范围；它不重建遗失写入，不证明业务外部 Effect 一致。

### 3.1 实际消息、分页与 checkpoint 边界（FG13）

限定一个 watch 覆盖整个投影的键范围，不滤除投影所需的 PUT/DELETE，且禁用 raw WatchCreateRequest.fragment（或使用已经按实际协议完整重组并验证的客户端）。第一份默认线性化 Range 取得 r 与 cluster_id；分页使用相同 cluster_id、固定原始 r、同一 range_end 和按字节键升序的独占续页起点（例如最后键后追加 0x00 字节作为下一包含下界），检查 more、次序和范围；后续请求明确带 revision=r。ResponseHeader.revision 是生成响应时的 store revision，可能大于历史查询 r，绝不能用它更新基线。所有页完整才原子发布；遇 ErrCompacted 或身份变化废弃未完成基线，重建，不拼接不同快照。新快照发布后以 r+1 inclusive 建立 watch，期间历史压缩则同样重建。

对绑定同一 cluster/stream/watch_id 的正常事件批，按 Event.kv.mod_revision（DELETE 为删除 revision，而非 prev_kv 的旧 revision）划分 revision，完整组保持事件顺序并原子应用该组及持久 checkpoint；不能用普通响应 header 当前 revision 代替已应用组的 revision。created ACK 仅建立 watch 身份，canceled/compaction 是控制状态，不是已应用历史或进度。真正的进度通知仅在确认同一 watcher 的所有截至该 revision 的必要事件已完整处理并持久化后才可原子推进无变更 checkpoint；不把所有空响应当进度。若允许 fragment，必须按该 watcher 及客户端协议完成重组后才分组提交，不将单个 raw response 当完整 revision；中途断线丢弃未完成组并从最后持久 checkpoint+1 重连，不跳到 fragment header。本文不规定某 SDK 的重组实现；未验证的 raw 分片路径应拒绝或禁用。

纸面反例与 Oracle（NOT_RUN）：
- r=10，创建 watch 时 header=14，但历史11..14尚未应用；若 ACK 后记 checkpoint14 并断线，重连15会漏历史。正确保持10，重放11起；以权威 r14 快照核整个投影及持久 checkpoint，而非回调计数。
- revision11同时写A、B；只收到A的分片即断线，不能发布半组或记11；重连11取得完整组后原子应用，核A/B同时可见及最终 checkpoint。
- r10中键A、C，第一页A；此后r11插入B且修改C，第二页仍以r10读C旧值、无B，即使header=11；随后watch从11得到B及C更新。若第二页改查r11却记基线11，会混合版本并跳过A的潜在更新。
- 若另行设计多个watch，A侧进度到14、B侧仅11，不能把14作为整个投影已应用边界；需分别记录同scope进度、证明各流覆盖与无缺口，并缓冲至共同完整水位才发布跨范围原子视图。共同水位本身不是聚合完整性的全部证明；本章不实现多watch合并器。

实读 etcd v3.6 [API](https://etcd.io/docs/v3.6/learning/api/) ResponseHeader/KeyValue/Range/Watch（L850–889、928–951、1105–1150）与 [API reference](https://etcd.io/docs/v3.6/dev-guide/api_reference_v3/) WatchCreateRequest fragment、WatchResponse、Event（L1554、1575–1588）；原 API guarantees 的完整 revision 保证须结合实际消息边界。以上为限定投影合同和原创反例，非已运行 etcd/SDK。

## 4. 租约、fencing 与远端效果

租约的失效由协调服务判定，不会撤销暂停中的旧进程已经持有的连接或即将发送的请求。教学：A 获 token=41 后暂停，租约失效，B 获42并向受保护资源写入；A 恢复。仅检查 A 的本地时间或协调服务中的新 owner，无法与远端最终写原子绑定。可选 fencing：资源以持久高水位拒绝小于已接受 token 的写；同一 token 的重放仍需操作身份及效果合同。token 必须来自可信、单调且适用同一资源/epoch 的发放者，资源在同一原子操作内检查并更新高水位和数据。只有 B 已在资源上建立42的 fence 后才能保证 A 的41在该资源被拒绝；若 B 尚未到达，单靠高水位不保证租约刚失效即撤权。严格实时撤权需要更强的共同权威合同。灾难恢复若重置 token 序列，需新 epoch 及有权资源切换，不可直接重复41。不可校验 token 的外部 API 不获得此保证；参见 CON 的未知效果与授权重放。租约不等于永久身份、恰好一次或分布式事务。

## 5. 有界故障验证

| 教学故障（NOT_RUN） | 独立 Oracle / 不足边界 |
|---|---|
| 5节点3+2/2+2+1分区 | 固定成员、注入网络隔离，记录调用/响应及独立权威读；禁止两侧各提交互斥名额；不得把超时请求数当未提交数 |
| 旧领导者响应丢失 | 真实请求身份、确认点和后续权威读取；不能重试为新意图；不强行判定每个未响应调用失败 |
| 暂停旧 owner，建立新 fence 后恢复旧 owner | 同一资源先接受42再尝试41，独立读取业务状态/Effect 与高水位；另测新 owner 未到资源的边界 |
| 多键同 revision、崩溃及历史压缩 | 投影与固定 revision 权威快照比较；应用/检查点一致；压缩后完整重建；不能只测 watch callback 次数 |
| 灾难恢复 | 新旧集群身份、旧客户端隔离、恢复点与已确认写差异、缓存失效、独立外部效果对账；不报 RPO=0 |

资料实读 2026-11-26：etcd v3.6 [API guarantees](https://etcd.io/docs/v3.6/learning/api_guarantees/) KV/Watch/Lease/Operation completed/Revision（L825–867），[Failure modes](https://etcd.io/docs/v3.6/op-guide/failures/) leader/majority/partition（L815–829），[Disaster recovery](https://etcd.io/docs/v3.6/op-guide/recovery/) restore/revision bump（L834–849）。多数交集证明、fencing 合同、投影算法及反例是原创教学推导，不声称来自 etcd 的业务 API 保证；未读完整 Raft 论文、未执行集群或验证生产性能。DOCUMENT_CANDIDATE，无新合格 Claim。
