# K03 深层专业知识：并发线性化、幂等与崩溃重放的证伪方法

> [K03 课程](systematic-syllabus.md) · [第一卷：并发条件写](foundations-deep-dive-01.md) · [第二卷：取消与背压](foundations-deep-dive-02.md)
>
> **R5 ENGINEERING_THEORY_CANDIDATE / FORMAL_PROOF_NOT_RUN / SOURCE_CLAIM_NOT_QUALIFIED。** 教学用的状态转换/反例与实现选项比较，不是现有 Tally/DomainHarness 的实现或已认证交易语义。K01 定义业务约束、K02 定义系统 Owner、K05 验证、K10 负责未知远端效果恢复。

## 一、为什么“看起来串行”仍可能出错

Node.js 单进程 JS 事件循环避免某些数据竞争，但 `await` 可使两个逻辑操作在读写间交错；多个 Worker/进程/Host 或数据库连接更明显。若 `read(); await remote(); write();` 的中间状态未受合同保护，不能凭代码没有线程锁就断言原子性。

线性化（linearizability）要求每个操作可视为在**调用到返回之间某一点原子发生**，且保持真实时间顺序；事务可串行化强调并发事务结果等价于某个串行顺序，二者解决对象不同。数据库隔离、队列 ACK 和外部业务 Effect 的统一全局线性化尤其不能由单节点 CAS 推出。

## 二、最小资源模型及局部不变量

以可重复投递的支付申请为教学例子：状态 `REQUESTED / CLAIMED / EFFECT_UNKNOWN / APPLIED / ACKED`，持久身份 `businessKey, revision, attemptId`。安全性候选：

- `I1`：同一个 `businessKey` 不产生多个**被禁止的**外部资金效果；
- `I2`：仅持有当前合法状态和版本的 Worker 可提交相关本地状态跃迁；
- `I3`：`ACKED` 有相应的最终权威 Effect 事实或依法允许的无 Effect 结论；
- `I4`：旧 Attempt 或不同 Tenant 不可覆盖新的业务归属。

这些性质不是“任何 API 都天然 exactly-once”。对 I1，若第三方不支持受控幂等键、状态查询、事务或补偿，通常无法仅在本地证明远端未重复执行；此时需缩小保证或请求有权业务/架构决议。

局部条件更新的抽象伪代码：

```text
UPDATE job
SET state='CLAIMED', revision=revision+1, owner=attempt
WHERE key=businessKey AND revision=expectedRevision AND state='REQUESTED'
```

要求真实数据库在适当索引/事务下以原子语义评估条件；影响行数为零应由程序识别为冲突，不能静默承认成功。这仅保护受该行/条件约束的局部状态；跨行不变量还要看事务隔离与读写集，远端 Effect 要有独立机制。

## 三、崩溃点矩阵（候选反例，非运行日志）

| 失败点 | 本地持久态 | 远端结果 | 合法恢复依赖 |
| --- | --- | --- | --- |
| 请求尚未持久化 | 无 Job | 无已知发送 | 按真实客户端重试协议重新受理 |
| Job 已持久化、远端调用前崩溃 | REQUESTED/CLAIMED | 未证实执行 | 需要证明未发送或使用被批准幂等机制 |
| 调用已发出、响应前超时 | CLAIMED 或 UNKNOWN | 可能已提交 | 优先按 Effect ID 读权威结果；不能按 HTTP 超时推断未执行 |
| 远端已提交、本地标记前崩溃 | CLAIMED/UNKNOWN | 已提交 | 幂等键/查询/补偿策略；避免第二次不可逆效果 |
| 本地 APPLIED 已提交、ACK 丢失 | APPLIED | 已提交 | 重放 ACK/查询结果，不再次产生 Effect |
| 旧 Attempt 延迟写入 | 当前状态可能已前进 | 任何 | 条件版本/来源身份拒绝错误覆盖 |

错误工程假设是“收到超时就再发一次”。对任意 POST，HTTP method 定义的幂等性不能替业务副作用作保证；如果系统使用特定幂等 Key，其保留周期、唯一范围、参数冲突行为和失效后的重放窗口也要明确。

## 四、取消、重试、退避与容量是耦合的

Cancellation Request 只能表示调用方希望终止；如果 Worker 在远端提交前已收到取消，可能安全结束；如果远端已应用，取消不能凭 Signal 倒转外部业务事实。异步队列的终止状态要区分 `CANCELLED_WITHOUT_EFFECT` 与 `EFFECT_UNKNOWN`，具体状态名字为教学候选，并非 Kernel ABI。

重试若没有终止条件，会将故障放大为负载。设长期平均到达率 `λ`、可持续服务率 `μ`；当 `λ ≥ μ` 且队列无界时，简单模型下不能长期稳定维持；重试增加实际请求到达率，使本来短暂过载持续。要考虑限流、带 jitter 的退避、有界队列、去重与毒消息隔离。任何容量公式都依赖受控服务边界和采样，不证明真实部署已达到阈值。

## 五、用最小模型检查思维生成测试

把 `claim`、`sendRemote`、`remoteCommit`、`recordAck`、`crash`、`restart`、`redeliver` 视为可交错动作。尝试构造 **最短违反 I1 或 I2 的轨迹**，例如：

```text
A claims job -> A sends remote -> remote commits
A crashes before local APPLIED -> queue redelivers
B sees stale CLAIMED -> B sends remote again -> duplicate commit
```

若真实远端有稳定幂等合同，第二次可能不会产生第二笔效果；此时上面的反例失效，应在 Case 中明确该前提，而不是强行声称所有队列必然重复付款。Model-checking 的 Safety PASS 仅覆盖已编码的模型和搜索界限，活性还需公平/可用性前提。无 Spec、无实际工具执行时保持 `MODEL_NOT_RUN`。

## 六、区分性工程练习

给定“至少一次投递 + 数据库条件更新 + 远端接口没有声明幂等合同”，要求给出两个备选：在当前有权范围内隔离不可逆重试，或者协商远端业务身份查询/补偿。评价必须包含：
- 交错与崩溃点矩阵；
- 真正的 State Owner 和本地 commit 线性化点；
- 哪个不变量可由当前事实保证、哪个仍 UNKNOWN；
- 对偶重放/跨 Tenant/响应丢失反例；
- 真实 DB 与远端 Provider 可得时如何验证；
- 对不可证明的保证为何必须缩小承诺。

**来源线索**：[A50 TLA+](../external-source-catalog.md)、[A51 PostgreSQL 18 隔离](../external-source-catalog.md)、[A52 HTTP 语义](../external-source-catalog.md)，仅为公开专业基础参考，不是本任务已经形式证明的 Source Claim。
