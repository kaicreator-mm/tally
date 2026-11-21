# K09 · 学科深化 01：接口与数据迁移中的兼容不变量

> [系统课程](systematic-syllabus.md) · [API 演进](topics/API_EVOLUTION.md) · [数据迁移](topics/DATA_MIGRATION.md) · [维护方法](engineering-methods.md)
>
> **ORIGINAL_EXPLANATORY_KNOWLEDGE_CANDIDATE / NOT_SOURCE_CLAIM_QUALIFIED / MIGRATION_NOT_RUN**。深化 K09.F1.1/F1.2、K09.F2.1/F2.2、K09.F3.1/F3.2/F3.3、K09.F4.1/F4.2 和 K09.F5.1。

## 1. “重构”与“业务变更”有不同验收要求

重构目标是在明确的支持范围内保留对外可观察行为，内部结构变化不能偷偷修改业务规则；功能增强/纠错则可能合法改变行为，必须识别 Product/消费者是否授权。重要的是明确**支持的旧版本、真实消费者及观察合同**，不能抽象声称“任何版本永远兼容”。

函数签名没有变，仍可能通过错误类型、排序、幂等语义或持久化状态改变而破坏兼容。K01 决定业务需要保留/改变什么，K02 决定接口/状态合同，K05 对旧消费者真实路径设 Oracle。

## 2. Expand–Migrate–Contract 是可选策略，不是万能顺序

对数据库列改名，从 `old_name` 迁移到 `new_name`，在滚动升级且旧进程仍运行时，直接 DROP 旧列会破坏旧应用。常见候选：
1. **Expand**：增加新列/读取兼容，旧进程仍可正常运行。
2. **Migrate**：按有界批次回填或双写，核实源与目标字段映射、失败恢复、重放安全。
3. **Contract**：确认所有受支持消费者均已切换且旧数据已处理，才考虑移除旧结构。

如果无滚动升级、多版本共存或迁移成本很小，可以评估更简单路线。两种方式都需具体恢复/备份与真实旧数据反例；不能把迁移方案命名成 Expand–Contract 就宣称验证通过。

## 3. 状态迁移的不可逆风险

示例：旧状态 `PENDING, DONE, UNKNOWN_LEGACY`，新状态仅 `READY, FINISHED`。前两个也许有获权映射，`UNKNOWN_LEGACY` 没有批准规则，不能任意替换成 `READY` 以便 Schema 验证成功。应明确保留/隔离/询问/人工处理路径，直到真实 Owner 裁定；不允许无来源默认映射。

不可逆删除、重新编码或外部副作用要从可恢复性角度审查。Rollback 代码能执行不意味着曾被删除的数据能恢复。若只完成 Fresh install，不可把结果标作旧用户数据升级 PASS。

## 4. 迁移知识的风险矩阵

| 条件 | 必须评估的影响 | 禁止推断 |
| --- | --- | --- |
| 新旧应用共存 | 读写格式、字段可空、旧 Consumer | 新代码绿色意味着旧客户端兼容 |
| 大表迁移 | 锁、批次、暂停/接管、进度记录 | 三行 DDL 必然瞬间完成 |
| 非法历史值 | 无合法映射的数据隔离/裁决 | 默认填充某个枚举即合格 |
| 外部 Effect | 重试/补偿/人工核实 | 本地事务回滚撤销远端动作 |
| 停用旧接口 | 用户声明的支持期限与日志证据 | Version 大一号就可立即移除 |

## 5. 专业停止条件与知识演化

缺合法历史值映射时，保留 UNKNOWN，不执行不可逆清理；没有真实受支持消费者范围时，不宣称完整兼容；缺部署权限时只输出有界候选计划。旧版本、源、迁移方法和失败反例需保留，后续发生知识过期时只局部复审受影响 Claim/Topic。

**来源线索**：[A04 生命周期](../external-source-catalog.md)、[A35 Fowler 数据库演进](../external-source-catalog.md)、[A41 Software Engineering at Google](../external-source-catalog.md)、[A47 Strangler Fig](../external-source-catalog.md)。引用仅作原理研究，不重新发布受保护原文。