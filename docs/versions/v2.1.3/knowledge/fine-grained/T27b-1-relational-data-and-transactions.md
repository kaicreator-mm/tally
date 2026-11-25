# T27b-1 · 关系数据设计、查询语义与事务隔离

K03.F4.1 主责，K02.F3.3/F3.1 状态/数据合同，K05 验证；#586 DA01/02/03。复用 [K03 并发及幂等](../../../v2.1.2/software-engineering-knowledge/K03/advanced-concurrency-idempotency-proof-sketch.md)、[T05-C07 查询](../casebook/waves/T05-common-function-abstractions.md)、[T19 批查询](../casebook/waves/T19-feed-refresh-architecture.md)。本文独立教学 SQL、数据和并发时序均 NOT_RUN；PostgreSQL 18 作为具体引擎，不宣称所有 SQL 实现相同。文件/ETL/共识/完整迁移另任务。

## 1. 从概念到逻辑再到物理模型

概念模型描述业务对象、关系、唯一身份、状态与约束；逻辑模型将它们表示为关系、列、键及依赖；物理模型决定存储、索引、分区、访问计划。订单 ID 是标识，不能用客户姓名替代；业务自然键与代理键可同时需要。一个代理键不自动防止重复业务记录。

教学例：设备登记与授权项目。规则候选：一个项目可多设备，同一设备可加入多个项目；每个组合只登记一次。设备序列号在厂商范围唯一（假定已由业务确认）。逻辑表：
- Project(project_id, name)
- Device(device_id, vendor_id, serial, model_id)
- Model(model_id, model_name)
- ProjectDevice(project_id, device_id, role)
主键描述行身份，外键描述引用，关联表的组合键约束登记唯一。若角色可以同时多个，组合键需改变；若历史登记要分期，另定义有效期与身份，不能凭名字猜。物理层是否为关联表建立反向索引，取决于查询、删除和写入成本，而非逻辑规则自动决定。

## 2. 函数依赖、范式与数据冗余

函数依赖 X→Y 表示在所有合法关系状态中，相同 X 必须对应相同 Y。不是一次样本刚好无重复就可推导；语义来自业务事实。候选键为最小超键，主键是所选择的候选键。教学登记表 R(project_id, device_id, project_name, model_id, model_name, role)，假设键 (project_id,device_id)，且：
```text
project_id -> project_name
device_id -> model_id
model_id -> model_name
(project_id,device_id) -> role
```
单表会重复 project_name/model_name，更新模型名可能需要改许多行；删除最后一条登记可能丢失设备或项目事实，不能独立录入尚未登记设备。

1NF 关注每个属性的适当原子域与关系表示，不等于禁止所有数组类型（SQL 物理/类型设计另议）。2NF 排除非主属性依赖候选键的一部分；3NF 对每个非平凡 X→a，要求 X 是超键或 a 是某候选键组成属性；BCNF 更强，要求非平凡决定因素 X 为超键。以上为一般关系理论定义及原创建模推导，并不表示 PostgreSQL 自动检查范式。先拆出 Project、Device、Model 与 ProjectDevice；按键连接可重建合法登记关系，且上述依赖局部可表达。一般分解须检查无损连接与依赖保持，不能按“每字段一表”机械拆分。

反范式是有意保存派生值/快照：例如历史订单上的商品名/成交价可能是不可变历史事实，不等同当前商品名；若只是缓存当前值，则必须规定来源、刷新/写入责任和不一致窗口。冗余不一定错误，是否保留取决于真实语义与实际读写成本。

## 3. 约束不是仅在 UI 校验

PostgreSQL 18 教学 DDL（只演示上述模型的登记/身份约束子集，省略 Model/model_id；vendor_id 非空，但厂商表暂未建，实际引用须加 FK）：
```sql
CREATE TABLE project (
  project_id bigint PRIMARY KEY,
  name text NOT NULL
);
CREATE TABLE device (
  device_id bigint PRIMARY KEY,
  vendor_id bigint NOT NULL,
  serial text NOT NULL,
  UNIQUE(vendor_id, serial)
);
CREATE TABLE project_device (
  project_id bigint REFERENCES project(project_id) NOT NULL,
  device_id bigint REFERENCES device(device_id) NOT NULL,
  role text NOT NULL CHECK (role IN ('reader','operator')),
  PRIMARY KEY(project_id, device_id)
);
```
这里只示范约束，role 是假定领域规则，不是权限执行器。关系表键防重复关联，不能证明请求者有权新增；跨租户需用权威租户范围与适当组合约束/访问控制（非本轮安全专项）。名称可重复，序列号唯一范围按业务决定。

PostgreSQL 18 具体行为：CHECK 表达式为真或 null 时不拒绝，若数量必须正且存在，要同时 NOT NULL 与 CHECK；UNIQUE 默认将 null 视为可彼此不同，需把缺失当一个唯一值时可以显式 NULLS NOT DISTINCT（支持版本依引擎）；PK 强制唯一且非空。FK 默认 MATCH SIMPLE 下引用列任一 null 可免匹配；MATCH FULL 只允许全 null 作为免匹配，混合 null/非 null 不满足。若必须有父记录，使用适当 NOT NULL；FK 引用目标键有约束，但 PostgreSQL 不自动为引用方创建索引。删除动作 RESTRICT/NO ACTION/CASCADE/SET NULL 含不同后果；选用前明确实际业务删除政策与可逆性，不能以级联替代授权。CHECK 不适合依赖其他行当前值的任意跨行约束；需具体唯一/排斥、事务/锁等机制。

## 4. NULL、连接与结果合同

NULL 表示缺失/未知标记，不是 0、空字符串或业务“未知效果”状态本身。SQL 谓词三值逻辑：普通比较任一操作数 null 得 UNKNOWN，WHERE 只保留 TRUE。使用 IS NULL、IS NOT NULL；确需 null-safe 比较时用 IS [NOT] DISTINCT FROM（核目标引擎）。反例：NOT (x=3) 在 x=NULL 时仍 UNKNOWN，不能自动包含缺失项。NOT IN 子查询若含 null 可能产生 UNKNOWN；用适当 NOT EXISTS/显式 null 合同解决，不机械替换导致语义变化。

教学数据：Project A、B；A 有 reader 设备 d1、operator d2，B 无设备。
```sql
SELECT p.project_id, pd.device_id
FROM project p
LEFT JOIN project_device pd
 ON p.project_id = pd.project_id AND pd.role = 'reader';
```
预期 A,d1 与 B,NULL。若将 pd.role='reader' 改放 WHERE，B 的 null 扩展行被移除；A 若无 reader 也会被移除。这不是优化等价变化。JOIN 一对多会增加行数，多次一对多连接可能相乘；求每项目设备数应按真实粒度分组，不能靠 DISTINCT 掩盖错误关联。SQL 无 ORDER BY 不保证输出次序；分页要有稳定且足够唯一的排序，及并发数据变化合同，具体留分页专题。

聚合区分 COUNT(*)（结果行）与 COUNT(column)（非 null 值）。在上述左连接中，COUNT(*) 对 B 为 1，COUNT(pd.device_id) 为 0；若统计 reader 数量，后者满足假定目标。查询返回 Rows 只是获得读取入口；既有 T05/T19 说明需核迭代完成与错误，不能忽略终止错误并报全部成功。

## 5. 索引与查询计划

逻辑正确性与物理成本分开。B-tree 常适合有序比较、范围、相等访问，但引擎会根据选择率、行数、成本估计、数据分布及统计等选择扫描/连接方式；索引增加空间、写入/更新维护成本。唯一索引可作为数据库约束的一部分，但普通索引不是业务唯一保证。

例：ProjectDevice 主键 (project_id, device_id) 便于按 project_id 查找；如果经常按 device_id 查其项目或删除设备，候选另建 device_id 索引。不能仅凭“复合索引第一列规则”断言所有版本绝不使用后列或必然快；实测目标版本查询计划。按 role 这种低选择率字段未必有益，组合索引、部分索引的适用性取决于条件与数据。

计划阅读顺序：核 SQL 与结果范围→估计/实际行数（统计与分布）→扫描方法、过滤位置、连接方法→排序/内存/IO→比较相同负荷的尾延迟及写入。EXPLAIN 给计划及估算，EXPLAIN ANALYZE 会实际执行；涉及修改时不要在生产或无权限对象上直接运行，即使回滚也不保证外部效果撤销。未运行此例，不给性能数字。

## 6. 事务合同与并发异常

事务是选择的提交边界；ACID 中原子性、约束一致性、隔离、持久性分别回答不同问题。业务不变量需设计正确，数据库事务不会自动理解任意业务语义；持久性还取决于引擎配置和故障模型。外部请求不因在数据库事务里调用就自动参与该事务。复用 K03 CON 对线性化与可串行化区别，不重复定义。

PostgreSQL 18 文档：Read Uncommitted 实际按 Read Committed；Read Committed 普通 SELECT 每语句取得可见快照，后续语句可能见新提交；Repeatable Read 使用首个非事务控制语句起的稳定快照（自身修改可见），比标准最低要求强但可能有序列化异常；Serializable 检测不符合串行执行的冲突并可拒绝提交，需要按合同重试整个事务。具体语句锁行为与其他引擎不同，不能推广。并发选择：
- 单行扣减：使用原子条件更新与受约束受影响行检查，可防特定单行不变量破坏；读后无条件写可能产生覆盖。
- 唯一插入：依数据库唯一约束和冲突处理，不能只 SELECT 不存在再 INSERT。
- 跨行不变量：明确读取谓词和写集，考虑锁共同权威记录、串行化或其他可证明机制；锁单独修改行不一定够。
- 只读多查询一致快照：按实际需要选隔离；更强隔离有重试/等待/容量成本。禁止把一次快照证明成最新远端事实。

教学写偏斜：两台设备 X、Y 均 enabled；规则“至少一台启用”。两个事务都读取两台，分别仅关闭 X 或 Y；在各自快照都看到另一台启用且修改不同记录，可能同时提交导致零台启用。使用 PostgreSQL 18 Serializable 且所有相关写入遵守同一合同，可检测这类不符合任一串行顺序的执行，至少需要处理失败/重新读；替代是锁共同项目状态行，随后按当前状态判断并更新（必须所有相关写路径同协议，且隔离/快照适配）。单纯给每台设备加 UNIQUE、各自锁行不能保证该跨行规则。验证两连接控制交错、最终可观察状态以及重试前重新检查业务前提；NOT_RUN。

事务重试只在错误语义、业务合同及预算允许时，重跑读取、判断与写入，而非仅重试最后语句；不要在可能被重试的事务中盲目重复邮件、支付等外部副作用。若提交回执丢失需核权威事实，不能直接假设回滚。此处不新增远端效果机制。

## 7. 独立验证矩阵

| 输入/交错（NOT_RUN） | 独立预期（按上述教学合同） |
|---|---|
| 同厂商重复 serial、不同厂商同 serial | 前者拒绝、后者允许 |
| 未存在父 project、缺失 device_id | FK/NOT NULL 拒绝 |
| 数量 CHECK>0 的 null 输入 | 只有 CHECK 时 null 可通过；组合 NOT NULL 拒绝 |
| B 无 reader，ON 与 WHERE 对照 | ON 保留 B，WHERE 排除；COUNT 差异如上 |
| 迭代中途失败 | 不能报完整成功，按部分/全有全无合同处理 |
| 两连接写偏斜/重复插入 | 按选用隔离或约束、真实提交与最终读回验证 |
| 有无候选索引 | 结果语义相同才比较实际计划/负载，未测不宣称性能改善 |

## 8. 来源与证据限制

实读 2026-11-25 PostgreSQL 18 官方 [§5.5 Constraints](https://www.postgresql.org/docs/18/ddl-constraints.html)、[§9.1 Logical Operators](https://www.postgresql.org/docs/18/functions-logical.html)、[§9.2 Comparison](https://www.postgresql.org/docs/18/functions-comparison.html)、[§7.2 Table Expressions](https://www.postgresql.org/docs/18/queries-table-expressions.html)（qualified joins / WHERE）、[§14.1 Using EXPLAIN](https://www.postgresql.org/docs/18/using-explain.html)（计划结点、估算及实际观察）、[§13.2 Transaction Isolation](https://www.postgresql.org/docs/18/transaction-iso.html)（Read Committed/Repeatable Read/Serializable）。范式理论参考 University of Wollongong CSCI235 Autumn 2026 [Database normalization](https://documents.uow.edu.au/~jrg/.snapshot/daily.2026-11-04_0010/235/slides/03databasenormalization/03databasenormalization.html)（实际读 2NF L322–324、3NF L724–731、BCNF 分解相关段；原创建模例与一般无损/依赖保持判断为教学推导）。Wilfrid Laurier University Normalization 页面直开限校园访问，不算已核正文。索引具体版本选型依据实际计划，未连接数据库。所有示例、选择与反例为 DOCUMENT_CANDIDATE，非新 source-qualified Claim 或产品测试。
