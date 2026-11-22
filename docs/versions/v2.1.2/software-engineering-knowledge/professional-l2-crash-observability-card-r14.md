# R14 · L2 决策时最小知识卡：持久状态可观察性

> **PROPOSED / DOCUMENT_CANDIDATE / NOT A RULE / NO RUNTIME QUALIFICATION。** 唯一详细依据：[R14 跨模块论证](professional-crash-observability-projection-r14.md)；学科 Owner 分别为 [K02](K02/advanced-architecture-invariants-and-failure-tradeoffs.md)、[K03](K03/advanced-concurrent-local-file-write-convergence-r13.md)、[K05](K05/advanced-invariant-derived-discriminating-oracles-r13.md)。R3 来源为 [独立复核 #539](https://github.com/kaicreator-mm/tally/issues/539#issuecomment-7076957905)，仅支持可复算的事实与机制假说，不支持知识收益的因果断言。本卡在 R14 独立接受前不应用作正式冻结的实验包。

## 何时读取？

任务实际存在**有权的持久状态变更**，且有并发、进程崩溃、重放或提交 ACK 丢失导致的状态不确定性。纯只读、无持久化影响、无已授权恢复要求时：`NOT_APPLICABLE`。先记录 Scope/业务身份、数据与状态 Owner、OS/介质/文件系统或 DB、承诺的故障范围；未知条件留 `UNKNOWN`，不能自行创造权限。

## L2：在编码前回答

1. **不变量**：谁能写？原始数据何时必须保持不变？逻辑身份如何跨重放保持？
2. **提交与 ACK**：精确哪个持久事实赋予 `COMMITTED`？对外成功是否保证已耐久？不要把 `rename`、EEXIST、局部 DB Commit 或超时自动等同整个业务操作完成。
3. **写入顺序**：按真实介质写出 Intent→Data/Effect→Commit 证据→ACK 的操作顺序；逐个核对每个命名状态在崩溃切点*实际能否被持久观察*。如果某状态只能在对应事件**之后**写入，这个状态不能用来区别发生于两者之间的崩溃。
4. **恢复真值**：在 Intent 前/后、发布前/后、Commit 前/后、ACK 前/后逐处截断，靠确实保存的版本/Scope/摘要/日志/事务记录区分 `NOT_COMMITTED`、`COMMITTED`、`UNKNOWN`、`CORRUPT`。若事实不足，保留 `UNKNOWN`，不得猜测重试成成功。
5. **技术备选**：比较单事务 DB、条件创建+可恢复记录、临时文件+有条件原子替换；先看保证与成本，再选技术。文件 `wx` 是条件创建，不是多文件原子事务；SQLite 事务也不覆盖未声明的外部 Effect。
6. **反驳**：让一个 writer 在“数据已变但 Commit 证据未保存”处崩溃；另一个读者只能以持久事实判定。写下它应拒绝的错误决定。若无法指定独立 Oracle 或目标 OS/FS 条件，记录 `NOT_VERIFIED`。

## 交接与验证

**K02 输出**：不变量、存储与 Owner、实际写序列/提交点、切点恢复表、备选与未决前提。

**K03 接手**：只对已选方案选取可验证的持久/锁/同步原语；并发竞争者不得把“目标存在”视为“记录已提交”，临时资源须有 Owner。

**K05 验证**：冻结一个能使错误状态模型失败的测试；直接断言文件 bytes、身份/错误码/公开结果，记录真实测试发现、执行、退出及所有未覆盖的故障面。模拟 crash 不等于 OS Kill 或掉电证明。绿色用例数不等于风险覆盖。

若发现**真正旧版迁移**，再选择 K09；若有**外部未知 Effect、运行恢复或过载**，再选择 K10。不要为凑知识包大小无条件注入这些章节。全过程仍遵循实际发布 ADS 与已授权任务流程；这是一张候选专业知识卡，不新增任何强制 Gate 或实现许可。
