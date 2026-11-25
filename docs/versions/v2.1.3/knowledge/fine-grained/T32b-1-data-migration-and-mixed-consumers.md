# T32b-1 · 数据迁移、混合版本消费者与兼容性边界

#598；K09 主责演进与迁移，K02 数据权威，K03 实现，K05 验证，K08 发布。复用 [K09 合同演进](../../../v2.1.2/software-engineering-knowledge/K09/advanced-contract-evolution-migration-and-retirement.md) 的 expand/migrate/contract 选择与完全重写例外、[配置迁移](../../../v2.1.2/software-engineering-knowledge/K09/advanced-configuration-migration-and-evidence-impact-closure.md) 的不可逆语义/证据影响、[T27b-1](T27b-1-relational-data-and-transactions.md) 关系约束与事务、[T27b-2](T27b-2-file-stream-and-durability.md) 导入导出。这里补并行写入、逐段验证、wire 演进与回退判断，不强制兼容无实际旧消费者的项目；例 NOT_RUN。

## 1. 先确定变换与真实消费者

迁移将旧状态按业务合同映射为新状态。应明确输入域、身份、单位、空值、状态、时间、权限、外部结果及历史要求；单射在其像集上存在数学逆映射，但未必覆盖新版本允许的全部值；实际恢复还需要可用的逆变换规则、版本及数据。若不同旧值映射同一新值，且没有另存区分信息，则不能仅从新值唯一恢复原值。不能用“程序成功”掩盖信息损失。教学：旧字段 duration_seconds 非负整数，新的 duration_ms 为非负整数；原值s映射1000s，在目标范围足够时可逆于旧像集。新写入1501ms无法无损回旧整数秒；采用截断或舍入需获权，不得伪造。若旧 null 表示未知，新字段0不能替代未知。

兼容矩阵应按 Reader/Writer 分别列：旧读旧写、旧读新写、新读旧写、新读新写，以及长寿命客户端、离线任务、持久消息、备份恢复、导出工具。兼容不是“新程序能打开旧数据库”一项。逐项决定仍受支持的方向、范围和退役时点。若允许一次停机且全部消费者可同步切换，单阶段受控升级可能更合适；若真实重叠，需设计混合状态和转移。不可逆变换的原始数据留存需同时满足保留期限、隐私、权限及可恢复性。

## 2. Expand–Migrate–Contract 的并发机制

教学 PostgreSQL 18 表 job(id, duration_seconds, row_version)，假定整数范围可用 bigint 毫秒表达，秒非负且非空，row_version 是本行权威版本（每个写入都维护）；目标增加 duration_ms。只有在旧、新 writer 都遵守迁移同步合同后才开始宣称两列一致。

| 阶段 | 机制及进入/退出条件 |
|---|---|
| Expand | 增加可空新列；旧写者若依赖 SELECT * 的列序等仍可能受影响，需真实旧消费者核验；保持旧字段权威 |
| 写入桥接 | 在同一权威 DB 事务/受控触发器内维护旧→新投影及版本，覆盖全部合法写路径。若有不能升级的旧 writer，先设计服务端桥接或停止其写入，不能假定它会双写 |
| Migrate | 分批回填、按稳定键遍历，读取当前权威值与版本；用条件更新/锁住同一行防止旧读覆盖更新，失败重新读取。记录范围与失败，重复执行不破坏新值 |
| Validate | 在同步合同下核所有记录、映射、异常、约束及旧/新消费者；新写能力暂限可表示旧语义的域 |
| Switch | 经获权验证切换读/写权威；仍支持旧者时明确反向投影及可表达值域，不能暗中开放1501ms |
| Contract | 全部已声明支持旧路径不再需要后，评估回退、备份/队列中旧数据、保留规则，方可删除旧字段/桥接 |

具体回填反例：读 row_version=7、duration_seconds=2；合法并发写把秒改为3、version=8，并同步 ms=3000。回填若无条件写2000即覆盖新事实。候选条件 UPDATE job SET duration_ms=2000 WHERE id=:id AND row_version=7 AND duration_ms IS NULL（仅填尚未初始化的投影；合法写入同步版本/新值），检查受影响行。若0行可能已被同步、已删或冲突，必须核当前权威状态，不能记为已迁移。若回填动作也改变被消费者依赖的版本语义，应按合同更新版本；本例假设它仅补内部投影、不改变逻辑业务版本。批次提交、断点与重新扫描需核并发新增/更新及删除，不以游标到末尾宣称全部完成。跨两个独立库的双写没有同事务保证，需要额外权威、可靠传播和对账设计，本文不新建分布式协议。

## 3. PostgreSQL 18 约束阶段与锁

在上述秒字段权威、范围检查及一致性写入桥接真实成立后，可按实际数据库能力逐步约束新列。教学 CHECK：
```sql
ALTER TABLE job ADD CONSTRAINT duration_ms_present
  CHECK (duration_ms IS NOT NULL) NOT VALID;
-- only after all ongoing writes meet this check;
-- correct existing missing rows according to authoritative mapping
ALTER TABLE job VALIDATE CONSTRAINT duration_ms_present;
ALTER TABLE job ALTER COLUMN duration_ms SET NOT NULL;
```
这不是可直接生产执行的脚本；实际 DDL 顺序、桥接与业务权限由迁移方案确定。PostgreSQL 18 NOT VALID 可用于该版本支持的外键、CHECK 和 not-null 约束，跳过初始全表验证但仍约束后续写入；VALIDATE CONSTRAINT 扫描既有行，验证使用 SHARE UPDATE EXCLUSIVE 锁，ADD CHECK 通常需 ACCESS EXCLUSIVE；存在可证明无 NULL 的有效 CHECK 时 SET NOT NULL 可省通常的扫描（同命令未删除该约束）。不能把“不扫描”当“不锁”或“无停机”；锁等待、表重写、WAL/磁盘、并发请求与回退均需按真实负载评估。只验证 NOT NULL 不证明数值映射正确；还需独立范围、单位与全量对应检查。若旧合法 writer 无法保证新列非空，不可提前引入此检查造成拒绝旧合同。删除旧列是实际数据/依赖变更，CASCADE 不能作为默认修复。

## 4. 消息与持久格式的版本兼容

Proto3 官方指南区分 binary wire-safe/unsafe/conditionally compatible。添加字段在二进制解析层可安全演进，但旧应用不会理解新的业务必需语义；新增 enum 可能破坏穷尽 switch 或旧业务拒绝策略。现有字段号不可任意重编号；删除后应保留号码避免复用，必要时保留名称（不同文本格式另核）。int32→int64 在二进制格式可解析却可能有损；旧 reader 若接收超过 INT32_MAX 的值可能截断高位。因此先部署足够兼容消费者，在仍有旧者期间限制实际写值域，再获权开放新范围。单位变化即使 wire 类型不变仍是语义破坏，例：旧秒字段同号改为毫秒会被旧读者误解，应另建字段/明确协议版本及映射。

Proto3 二进制 unknown fields 通常保留于消息解析/重新序列化；转换 ProtoJSON 或逐字段构造新消息可能丢未知字段。不能从“旧读者不报错”推断旧中转者能保留新数据。实际验证固定生成器/runtime 版本、二进制与文本/JSON 路径、旧→新→旧以及新→旧中转→新，比较域合同而不是要求序列化字节完全一致。解析合法、数据保真、业务行为、应用源/ABI 兼容是不同检查；详见 [T31a-1](T31a-1-compilation-linking-abi.md)。

## 5. 校验、恢复与退役

独立对账使用同一可解释观察边界或受控写停顿/版本水位；单纯分别 COUNT 两表可能在不同快照观察，数量相等也可能错配身份。核逐键映射、重复/缺失、范围/空值、业务聚合、未授权范围不变、异常原始值与处理状态。不要从被测变换直接生成 Expected。小例：旧 s=2 应独立算2000ms；旧 s=0 保留0，null 若合法仍标未知；若两记录误交换单位/身份，COUNT 不变但逐键 oracle 失败。摘要只能作辅助，定义排序、编码和字段域，并保留可定位异常。

回退决策分为停止切换、回退读取路径、回退程序、逆向数据迁移、从备份恢复或业务补偿。新写入1501ms时旧程序不能无损表示，直接部署旧代码或恢复旧备份可能丢掉之后已合法提交业务，必须按权威记录与恢复目标决策。备份存在不等于恢复可用；需要实际恢复演练与 RPO/RTO 证据（本轮未运行）。收缩前明确仍存在的旧消费方（包括延迟消息、导出、离线、外部服务）、有权退役记录、适用保留期限和不再可逆的条件；不强制保留所有历史兼容层。

## 6. Oracle 与资料边界

| 教学反例（NOT_RUN） | 独立预期 |
|---|---|
| 回填版本7后并发改8 | 不能覆盖3000ms，0行需读回或重试 |
| 旧 writer 绕过桥接/回填批次中断 | 拒绝提前切换；按持久记录恢复及对账 |
| NOT VALID 后旧行违规/新行违规 | 旧行尚未全局证明；新违规写受约束拒绝；验证全量及实际锁风险 |
| int64 大于旧范围、unknown field 经 JSON | 分别检查解析/值域/中转保真；不宣称自动兼容 |
| 1501ms 新写后代码回退 | 检查有权回退政策，不无损时拒绝自动逆变换 |

2026-11-26 实读 PostgreSQL [18 ALTER TABLE](https://www.postgresql.org/docs/18/sql-altertable.html) SET NOT NULL、ADD/VALIDATE CONSTRAINT/Notes 及 [5.7 Modifying Tables](https://www.postgresql.org/docs/18/ddl-alter.html) 列变更；[Protobuf proto3 Language Guide](https://protobuf.dev/programming-guides/proto3/) Updating A Message Type、Unknown Fields/Retaining Unknown Fields（在线指南，未验证具体 protoc/runtime 版本）。实际 DB/锁、消息编解码、迁移脚本与恢复均未执行；所有流程/SQL/反例为原创工程综合 DOCUMENT_CANDIDATE，不新增合格 Claim。具体 CDC、多库切换、线上迁移工具、备份恢复与数据保留政策留专题。
