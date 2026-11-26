# T38-K3 · 任务生命周期、可靠效果及恢复知识的最小核验

#642 / #563；冻结输入 078078eac2ac7c14473ffb49310e26d58f7086cb；[Owner方向](https://github.com/kaicreator-mm/tally/issues/563#issuecomment-7190227558)。当前仍 KNOWLEDGE_CONSTRUCTION。承接 [K2独审](https://github.com/kaicreator-mm/tally/pull/641#issuecomment-7190897414) ACCEPT_SCOPED 0/0/1 与 [合入收据](https://github.com/kaicreator-mm/tally/issues/640#issuecomment-7191007924)，K1-RV-01 已关闭；K2-RV-01 待本批独审。验证针对知识命题而非产品实现，不增加新平台、Package、SDK 或LLM实验。

## 1. K2 P2 追加勘误：输入域与实际存储类型

保留 [原K2见证](T38-K2-witness.py) Blob d9704f0aab5fb68217432638aad0b7cb882793ba 和 [原收据](T38-K2-knowledge-validation.md) Blob 04f6a95c80e4a289e8ced40a85cfe499279aeacf，原观察不变。原A中CHECK/NOT NULL对正数量的结论仅限已声明合法数值域，C仅在有效非负数值库存、正数值请求和单权威行的合同下成立，若合同要求整数则须另核整数域。普通SQLite INTEGER是类型亲和性，并非严格类型约束。K2独审所述实际反例在本批A重现：'abc'存为text、1.5存为real，均通过普通INTEGER NOT NULL CHECK(q>0)。[SQLite datatype3](https://www.sqlite.org/datatype3.html) 为原review的官方限定参考；本批证据来自实际引擎，不重新宣称阅读全部上游规范。

对本教学要求“存储值必须为正整数”，实际SQLite3.46.1的STRICT INTEGER NOT NULL CHECK(q>0)拒绝本批'abc'/1.5/0/-1/NULL，接受1；另接受字符串'2'并存成integer 2。后者明确说明存储类型合同不等于输入参数原始类型/词法合同，若业务不允许字符串，需在实际输入边界验证。其它可行机制包括依实际版本使用typeof(q)='integer'与正值约束，未在本批执行。实际扣减另核参数正整数、库存整数非负及目标身份和提交；负数扣减增加库存，不在K2已声明正量合同中。本批没有执行新的并发扣减。原T27b-1（dfa2626af54fa2a77ab1e78e6e8d93e467ed2b7b L41–65、L89–103）为PG18限定正文，不以SQLite反例改写PG知识；K2-RV-01属于见证解释，不是原专业错误。历史字节和实际stdout不可改写。合并前仍OPEN_NONBLOCKING。

## 2. 冻结来源与实际最小观察

以下路径均相对 docs/versions/v2.1.3/knowledge/fine-grained/。本批实际读取所列冻结原文，未重新获取所有外部资料。原P37R3-LIFETIME/DEADLINE、P36-EFFECT/OFFSET/RANGE/DURABILITY/STAGING、P37-RECOVERY等仅作定位，不生成新的机制身份。

| ID | 原知识路径/Blob/范围 | 独立合同与实际观察、限制 |
|---|---|---|
| A | T27b-1-relational-data-and-transactions.md dfa2626af54fa2a77ab1e78e6e8d93e467ed2b7b L41–65、89–103 | 上节K2勘误，实际普通/STRICT存储类型对照。固定输入、同一SQLite引擎，非任意输入或PG18验证。 |
| B | T27c-1-concurrency-task-lifetime.md 8fb9513b0bca07e4f81bb0750ff905c3ca87c755 L34–49 | 明确合同：等待方可在未完成时返回；被等待任务未被取消且之后可完成；另一个TaskGroup内成员非取消异常应取消同组其他成员、等其清理后退出。实际Python3.13.5 asyncio.wait(timeout=0)返回done空、pending含task、未cancelled，随后释放gate并await得到'effect'（字符串是局部返回值，不是远端效果）。TaskGroup显式started gate保证兄弟进入后才抛ValueError，退出后child.cancelled且cleaned Event已设，捕获一个ValueError组。只覆盖实际3.13.5、该可协作取消的子任务及finally执行；不声称原3.11版本实测、清理任意资源、任意API超时行为、拒绝取消时终止期限或外部进程/事务停止。 |
| C | T27c-3-reliable-messaging-outbox-offsets.md 973fb2b186a5e76b5fe08dfa15e8c6b2544bcac8 L28–47 | 单逻辑consumer、单producer作用域，事件E8规范body=v8，同一权威DB内Inbox与effect记录同时提交；同ID同内容为合法重复，不产生新effect；同ID异内容为冲突。实际SQLite内存库默认legacy事务控制，在with db内两次INSERT后抛异常，两个表均为空；合法应用一次，重复一次、异内容一次均不增加/修改记录，最后两表仅(E8,v8)。这里只观察同连接顺序执行、Python异常导致回滚、DB行作为唯一局部effect；不是进程崩溃或重启、并发去重/抢占、broker/offset、外部邮件付款、持久介质或ACK丢失实测。脚本先查后写仅在本例单连接顺序及唯一约束合同下演示，真实并发需适当事务/隔离/唯一冲突处理及当期写许可；不能直接作为通用消费实现。 |

B 的合法反转：若实际等待API明确执行取消且目标协作并终止，则可在观察真实settled/资源后报告取消；仅请求取消、超时或取消等待者均不等价于业务效果未发生。C的合法反转：两个不同稳定业务身份即使body相同，若合同授权独立动作，也不能按内容直接去重；若是依赖前版本的delta，缺版本需补齐或有权重建，不以最新覆盖；完整快照是否可跳版本须获权。上述反转为纸面条件，不冒充执行。

## 3. 三组无需建设环境的独立推导（实际目标执行 NOT_RUN）

D 消息进度与效果（T27c-3 同Blob L5–58；特别L22–27、L28–47、L48–58）。单partition已提交位置40，实际返回40/41/43，42为已知合法空隙，批次下次位置44。40/43已持久处理而41未完成，恢复位置最多推进到41；不能按完成数+2得42、也不能提交44。41完成后才按实际合法nextOffsets推进44，不连带其它分区。若目标本地事务已提交、offset未确认，允许在同一业务身份/规范内容且去重窗口有效时重放，效果仅一次；若先提交offset再在业务事务前故障，则恢复可跳过尚未发生的效果。原C实际回滚/重复是局部事务见证，不证明这个broker故障历史。对于Kafka 4.1/Java 4.1.2 topic→topic，只处理已提交输入的合同需要输入read_committed、禁auto/独立offset commit、同producer事务和实际consumer.groupMetadata及合法连续完成前缀；输出read_committed。独立offset41先提交、输出abort可导致恢复漏处理40；输入read_uncommitted读取后来abort的E8并成功输出Y，即使输出read_committed也不能撤销Y。commitTransaction timeout/interruption不能推出未提交，按原API限定重试同一commit或关闭并按实际状态恢复，不盲切abort。独立SQL/HTTP不被该事务纳入；真实外部ACK未知需稳定身份、实际Provider幂等窗口、权威读回或获权补偿，禁止盲重发。消息/业务/传输/尝试身份不同；组资格非外部资源写fence。无broker、Kafka或目标API实际执行。

E 任务期限、容量与恢复（T25b-2-time-clock-calendar-semantics.md 195c4e65baadcaa404a34350a4f797ae02f65012 L5–47；T27c-1 L34–49；T33a-1-reliability-fault-analysis-recovery.md 16bcfa4b2cd718fbfc3a000cb9d3430f59091520 L32–37）。同一monotonic时钟域，预算5s、第一阶段3s，忽略其它开销第二阶段最多2s；若剩余0，不再启动下一阶段或正时长backoff。真实迟到成功可以记录结果，不能计为按期完成；Linux MONOTONIC不计suspend，业务含挂起则另选适用clock（例如核BOOTTIME），不以墙钟回拨延长经过时间，不把monotonic绝对值跨重启使用。C4+Q16只在两个互斥集合且超额准入前拒绝时约束控制器20，额外producer/retry不在此数内；已分配执行许可不等于仍排队或用户函数已进入，清空待办不撤销admitted。恢复例：10:00中断，目标RTO30分钟/RPO5分钟；10:12进程启动、10:34实际获权用户路径和所需对账完成，34分钟不满足RTO；09:53合格一致点差7分钟，不满足RPO，若真实连续日志恢复至09:58则差2分钟可满足该RPO。两目标独立，不以最近一行时间戳或进程启动证明恢复完整；若外部付款仍UNKNOWN且对账为恢复前提，不能报告已满足恢复结束条件。所有时间、容量为声明的教学数值，不是实测服务性能、计时或恢复。

F 文件/协议/导入恢复（T27b-2-file-stream-and-durability.md 9be743c9c6fe87e276b3867860fef9a28c217a79 L79–101；T35-3-complex-general-function-composition.md 6d3d81acd9fa952ebc3f06e1af9d5e4250822f89 L9–44）。同受控目录的Linux rename成功可使新名称可见，但目录sync前故障后持久性未获确认；旧已打开句柄仍可见旧内容。文件sync和目录sync角色不同，须真实FS/device合同，原子替换非CAS、非跨资源事务。固定representation A、可信块/文件digest、已持久验证0–399，总1000：合法206/400–999/1000且实际600字节校验后才可登记对应持久块；200全表示不追加，416即使本地长度1000也不证明身份或完整性；若只206/400–699不记全600字节完成，独立部分receipt另需合同。接收、校验、持久内容、receipt分别观察，内容先于receipt故障重核，receipt存在但缺字节撤回进度；不以HTTP头或UI进度保证耐久。CSV的字节块/物理行非逻辑记录，实际parser/dialect/encoding及记录身份决定恢复点。原教学a,b,c,d中c非法，整批发布g1保持，另获权部分政策才有3成功/1拒绝；staging持久不等于用户可见。真正同DB切换pointer/outcome/change且消费者仅查选定generation才是原子发布合同；ACK丢失读真实原任务，不另创任务。短小可重下输入不必持久staging，换parser不套旧checkpoint。未发送HTTP、未模拟/执行断电、PG18 COPY、真实文件持久化或发布。原文平台版本限定不提升。

## 4. 可重复实际收据

[T38-K3-witness.py](T38-K3-witness.py) 83行/3586字节，仅Python标准库asyncio/sqlite3，临时目录，无依赖安装、网络、产品源码。实际 Python3.13.5、SQLite3.46.1、sys.flags.optimize=0；`python -m py_compile T38-K3-witness.py`、`python T38-K3-witness.py` 均exit0。实际Git blob bdc0e68983eaa50949d57723418a7a60c5b50316与GitHub读取相同；SHA256 1c56ed95b871a88b4b6404d9d95946f159a9e2df714d828c5e7a1e66668be332。stdout：

```text
A loose accepts text/real; STRICT rejects selected invalid values; coerces numeric text
B wait timed out without cancellation; later result; group sibling cancelled and cleaned
C precommit exception rolls both back; repeat same identity no new effect; conflicting body rejected
```

A–C为实际固定见证，D–F为纸面推导，所有真实目标执行NOT_RUN。固定输入断言只检查明确合同，不以脚本自身定义知识正确。未发现所选原命题实质错误；K2勘误为输入域澄清。真实特定版本或跨资源未知时保持UNKNOWN/NOT_RUN，不用替代引擎或模拟冒充。独立审核从原文/合同重推预期并检查实际证据及反转，不必开发验证系统。无新增Case/Claim/Rule、实际LLM效果测量、全库验证或原资格晋升。67/86、46/265、S51/D0/U5及历史资格保持。#565 OPEN/HOLD、#286 DEFERRED、A24/T4 NOT_RUN/HOLD。后续优先实际业务合同与组合反例，重点检查政策与通用机制的分界，按疑点定向补知识，不将验证变成无限任务。
