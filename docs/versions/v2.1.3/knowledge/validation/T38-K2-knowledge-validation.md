# T38-K2 · 关系、事务与一致性知识的最小核验

#640 / #563；冻结输入 a0e9900a873bf0ab1b88d41ef12fa60ff90af048；[Owner方向](https://github.com/kaicreator-mm/tally/issues/563#issuecomment-7190227558)。当前仍 KNOWLEDGE_CONSTRUCTION。目标是验证知识本身在明确条件下是否正确合理，而非构建数据库产品或验证框架。承接 [K1独审](https://github.com/kaicreator-mm/tally/pull/639#issuecomment-7190379741) ACCEPT_SCOPED 0/0/1；[合入收据](https://github.com/kaicreator-mm/tally/issues/638#issuecomment-7190543552)。#638 仅保留一项P2待关闭。

## 1. K1 P2 术语勘误（T38-K1-RV-01）

原 [K1见证](T38-K1-witness.mjs) Blob eda008c46cf42ccca41f907e2c335e6a3e3ee99e 与 [原收据](T38-K1-knowledge-validation.md) Blob fe406c1b39cce66daaf6140eef80d7e2ca9c1a1b 原样保存，不重写实际输出。原T27a-detailed-design-representation.md（0e1e0bd6db29c7d73626d1a502a08f0ca845e83a L18–28）定义 AF 的域为合法表示。K1 C 的注释“same abstract value”和输出“AF equal”应读作：**raw union equal; RI canonical=true split=false**。函数expand对非法表示仍可计算原始数学并集，但它不是原合同的AF在非法表示上的定义。canonical的AF为{1,2,3,7,8}；split的raw union同此集合，却不在原AF定义域中，不能声称两者AF相等。K1原断言、原观察和限定结论保持；原专业正文无需修改。本篇为明确的后续勘误及收据解释，而非声称K1历史实际输出已变化；P2仍待独审确认及合并后关闭。

## 2. 冻结来源与有限验证问题

实际读取冻结原文两个文件：`docs/versions/v2.1.3/knowledge/fine-grained/T27b-1-relational-data-and-transactions.md`（dfa2626af54fa2a77ab1e78e6e8d93e467ed2b7b）L5–114；`docs/versions/v2.1.3/knowledge/fine-grained/T27c-2-distributed-consistency-coordination-recovery.md`（941bb0e11a631bcd5b06860c03e0225be40a03d2）L5–16、L39–42。原RELATIONAL/CONSTRAINT/QUERY/TRANSACTION/CONSISTENCY/FENCE仅为阅读入口。按源语义决定验证方式，原文引擎限定不替换；当前环境未发现postgres/psql命令（仅PATH检查，不代表任何地方不存在PG）。不下载或安装新数据库，不将SQLite当PostgreSQL的替代验证。

| ID | 原文 | 明确的独立教学合同、正反例 | 本批证据资格 |
|---|---|---|---|
| A | T27b-1 L41–65 | 厂商内serial唯一，跨厂商可重复；必须存在父项目且设备非空；q存在且正需两个约束；可空唯一键按本引擎默认语义。 | 实际SQLite 3.46.1隔离内存库，PRAGMA foreign_keys=ON实际读回1，跨厂商同serial插入成功；同厂商拒绝、未知项目及NULL设备拒绝；CHECK单独允许NULL、拒绝0、NOT NULL组合拒绝NULL，UNIQUE允许2条NULL。不是PG18执行，未覆盖MATCH FULL、NULLS NOT DISTINCT、级联及所有引擎语义。键/约束不提供调用者授权；例如同一合法键值的请求可由未授权主体提出，DDL本身不包含该事实。 |
| B | T27b-1 L66–88 | 结果必须保留所有项目并统计reader设备；A有reader1、operator2，B无设备。WHERE只接受TRUE。 | 实际SQLite：ON得(A,1),(B,NULL)，WHERE得(A,1)，A计数1/1、B计数1/0；NOT(NULL=3)和1 NOT IN(2,NULL)为NULL（SQL UNKNOWN）。若真实合同只要存在reader的项目，WHERE排除B可以合法；不能宣称条件位置总错误。未验证索引、计划、性能、分页或PG特有行为。 |
| C | T27b-1 L89–103 | 一个库存行q=5，两次有权申请正量3、4，条件更新和受影响行表示成功或不成功。 | 实际SQLite同连接顺序执行，两次受影响行1/0，提交前读回2并实际commit成功；未作重启持久性或并发测试。若两个原子操作在同一权威行串行生效，4先成功则剩1、3失败（纸面反转）；若容量7且相同正量合同，两次可成功，最后0（纸面）。不能把顺序见证称为隔离证明，不能用无行身份谓词的单行教学SQL更新真实多行表；负扣减须由合同/约束阻止。 |
| D | T27b-1 L5–40 | 函数依赖来自合法关系全集；无损分解与依赖保持分别成立或不成立。 | 纸面独立证明与构造反例，见§3；NOT_RUN。 |
| E | T27b-1 L89–114 | 至少一台X/Y启用的共同谓词；两个事务各读两行、只关不同一行。 | 纸面历史反证，见§3；非PG18两连接运行。原PG18具体保证仍为原有文献限定，未重新实测。 |
| F | T27c-2 L5–16、L39–42 | 实时读写、一致性语义、固定成员多数与资源高水位各自有条件。 | 纸面历史/集合/状态推导，见§3；不运行etcd/Raft、真实远端或模拟服务。 |

A–C 只是共同SQL语义在一个实际引擎/固定输入的可观察见证，不能因此将原 PostgreSQL18 专业来源资格、隔离行为或其它数据库合同晋升。各例没有真实业务授权，教学合同独立给定；正确性Oracle从表中声明的预期推导，而不是执行返回值决定预期。

## 3. 不需要代码的三个知识检验

D（关系理论）。原登记关系R(p,d,pn,m,mn,role)的给定FD为p→pn、d→m、m→mn、pd→role。闭包pd+=全部6属性；没有FD引入p或d，因此任意超键含p,d，唯一候选键pd。依次拆Model(m,mn)、Device(d,m)、Project(p,pn)，每一步交集m/d/p决定拆出的一侧，逐步无损；每条原FD在某子关系内，投影FD并集蕴含原FD。反例L(c,s,z)，cs→z,z→c：cs+=csz,zs+=csz，两个候选键cs/zs；z+=zc缺s，故z非超键；所有单属性均为主属性，满足3NF而非BCNF。Z(z,c)和S(s,z)交集z决定Z，分解无损。局部合法Z={(z1,c),(z2,c)},S={(s,z1),(s,z2)}连接得(c,s,z1),(c,s,z2)，违反cs→z；这不是来自满足原FD的L，故不反驳无损定理。局部FD z→c及S无非平凡原投影FD不足以保持cs→z。可保留3NF、另核跨表约束，或确证实际依赖变化；不能仅因BCNF就必选。历史成交值若为真实事件事实，当前名称变化不能重写历史。上述为数学/条件推导，NOT_RUN，不是随机样本证明FD。

E（事务）。初始(X,Y)=(on,on)，T1读均on、拟关X；T2读均on、拟关Y；在允许两个快照写不同记录而均提交的历史H中最终(off,off)。串行T1→T2时T2应见X off，拒绝关闭Y；T2→T1反之，故H在此合同下与任何串行合法历史不等价。原PG18 Repeatable Read可能有此序列化异常，Serializable检测并可拒绝，需真实引擎/全部相关事务和重试合同；不能用这一纸面历史当PG真实复现。共同项目权威锁方案只有所有相关写路径遵守同协议、按适用隔离/快照在持锁后取得正确当前状态并完成判断和写入才成立；锁各自设备不足。对真正单行条件更新可不采用全局串行化，但同一事务内顺序见证C不证明多连接行为。若业务明确允许全部关闭，则原反例不再违反该业务谓词，仍须另核隔离合同。序列化错误的获权重试应重新读/判断/写，若邮件或支付效果可能已发生，不能当作数据库回滚自动撤销；提交ACK未知也不直接判未提交。没有调用外部效果或模拟出真实事务保证。

F（一致性及fencing）。初值x=6，W(7)完成后R才开始且无其它写，R=6不存在合法线性化点；R与W重叠时可在W之前读6或之后读7。两个键各自合法单读不保证跨键一致快照。若原(x,y)=(0,0)，原子更新(1,1)，读x在更新前、读y在更新后得(0,1)，各读可合法但该组合不属于任一原子版本。固定5个投票成员多数3，任意两个三元子集至少共享1（3+3-5），3+2分区仅一侧有多数，2+2+1无多数；交点可双重承诺时，例如{a,b,c}与{c,d,e}分别接受冲突值，共享c仍不阻止冲突，所以交集不是共识证明。活性还需通信/协议/存储等真实条件。单资源高水位41，若B已在同一原子保护操作建立42并写入，A41随后小于42拒绝；若B只在协调器取得42尚未到资源，资源仍41，仅“小于高水位”规则允许A41（假设无其它拒绝条件），说明fence不等于即时撤权。要求可信发放、同资源/epoch、资源原子检查更新、实际重放身份；远端不可核token不获保证。同token重复是否可接受由实际操作身份/效果决定；独立合法历史不得被事后撤权自动取消。无真实共识、网络、时间保证或远端执行，NOT_RUN。三种纸面推导均可由独审另行重算，非真实运行Oracle。

## 4. 实际最小执行收据

[T38-K2-witness.py](T38-K2-witness.py) 53行、2759 bytes；Python 3.13.5、SQLite 3.46.1；仅Python标准库sqlite3，独立内存数据库，无SDK、Package、第三方依赖或网络。临时目录执行 `python -m py_compile T38-K2-witness.py`、`python T38-K2-witness.py`，均exit0；随后 `git hash-object`、`sha256sum`、`wc -l -c`。实际Git blob d9704f0aab5fb68217432638aad0b7cb882793ba 与GitHub提交读回一致，SHA256 7f9461b92a71d7b32d45db90a769bdf74a123b44712a8943cc0884b6ef653387。执行时未启用Python -O（断言有效）。实际stdout：

```text
SQLite 3.46.1
A scoped uniqueness, FK, CHECK NULL, NOT NULL, UNIQUE NULL observed
B ON retains B; WHERE removes B; B counts 1/0; predicates UNKNOWN
C sequential conditional debit 3/4 from 5: affected 1/0; final 2
```

A–C的断言为明确教学合同中的固定有限输入，D–F为独立条件推导；不合并成“6个真实数据库/分布式测试通过”。没有新Case/Claim/Rule、全库语义/来源复核、实际LLM消费或工程效用统计。

## 5. 知识反馈及后续

本轮所选源命题与观察/条件推导一致，尚无证据要求修改原专业正文。已修正K1的术语解释（待独审），原始证据保留。未来真实任务涉及特定引擎隔离、持久化/复制/远端结果，应优先读相应原文及版本限定，必要时用已经具备的实际环境最小核验；没有条件时留NOT_RUN/UNKNOWN，不建设平台。下一批优先任务状态、异步失败及实际消息/协议语义疑点，仍按问题选最小推导或执行。不以每个引用写脚本、数量或通过率为目标；67/86、46/265、S51/D0/U5与历史资格保持。#565 2P1/2P2 OPEN/HOLD，#286 DEFERRED，A24/T4 NOT_RUN/HOLD；当前知识建设，不恢复产品实施。
