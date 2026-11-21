# K03 · 学科深化 01：算法、数据结构与实现正确性的联合选型

> [15 点系统课程](systematic-syllabus.md) · [算法专题](topics/ALGORITHM_CHOICE.md) · [实现方法](engineering-methods.md) · [测试](../K05/systematic-syllabus.md)
>
> **状态：ORIGINAL_EXPLANATORY_KNOWLEDGE_CANDIDATE / NOT_SOURCE_CLAIM_QUALIFIED / NOT_BENCHMARKED。** 深化 K03.F1.1/F1.2/F1.3、K03.F2.2、K03.F3.1 与 K03.F4.1。知识分析不授予 Builder 源码写权限。

## 1. 成本模型至少包含正确性、规模和环境

复杂度 `T(n)` 描述输入规模增长下的运算成本阶，不是具体系统端到端耗时。数据库/网络访问、CPU 缓存局部性、GC、分配和 IO 往往支配小规模工作负载。应同时记录输入规模范围、操作组合、内存预算、数据变化频率和响应目标。

示例：列表线性扫描单次查找 `O(n)`；Hash Map 期望查找常可近似 `O(1)`，但不保证所有输入、所有碰撞情况都为常数，构建索引、内存和序列化同样付成本。对 20 个元素的一次查找，构建 Map 未必更快；对数十万条、多次查找且键稳定，Map 值得实际比较。

## 2. 正确性必须在性能之前明确

先定义函数的前置条件、不变量、输出合同和错误输入。排序算法应说明是否需要稳定排序、如何处理重复键及不同 locale；去重应定义业务相等性和顺序保持；字符串归一化不可随便替代领域 ID 比较。若键是可变对象或者序列化不稳定，索引可能查不到或产生错误匹配。

伪代码思路：
```text
For each event:
  verify validated identity and version
  compare current revision with expected revision
  if mismatch -> CONFLICT (no state mutation)
  else perform one authorized transition
```
这是并发保护策略的抽象，不是特定数据库 API。是否必须实现 compare-and-swap 取决于实际冲突模型；异常重试还需考虑副作用和幂等身份（K02/K10）。

## 3. 选数据结构时的实际权衡

| 需求 | 候选 | 必须检查的代价/反例 |
| --- | --- | --- |
| 少量、只读、顺序扫描 | Array/List | 不应只因 O(n) 就重写 |
| 大量、频繁键查找 | Map/Hash Index | 构建开销、碰撞、键相等性、内存 |
| 区间/排序查询 | 排序数组/B-tree 等 | 更新频率、页 IO、排序合同 |
| 先进先出任务 | Queue/持久 Job Store | 重启丢失、回压、重复投递 |
| 高并发共享状态 | 条件写/事务 | 隔离级别、冲突重试、业务副作用 |

算法选型不能忽略数据库查询计划；将所有记录读取到内存后使用 O(1) Map，可能比有索引的数据库范围查询昂贵得多。

## 4. 形式化的边界与最小反例

证明 `O(n log n)` 并不证明排序结果正确。教学反例： comparator 对相等键返回非零，导致顺序不稳定或合同不一致；Map 用未经规范化的复合键导致“逻辑同一个订单”重复存在。可提出性质：
- 输出中的每个元素都来自原输入且次数守恒（如果合同是排列而非去重）。
- 已排序邻接元素满足 comparator 的非降序。
- 重复请求有稳定 ID 时，同一操作结果的外部副作用数符合**明确已授权**的幂等合同。

这些性质须与 K05 的业务 Oracle 和真实测试执行分开；“性质能写出来”并不代表测试已运行。

## 5. 选择与停止规则

小数据基线没有瓶颈时默认保持简单实现；真实热点与正确性风险都明确时，先用等价输入比较候选，记录 CPU/内存/IO 与端到端延迟的实际测量；未测则标 `NOT_MEASURED`。一旦缺关键业务相等性或持久一致性合同，应先回 K01/K02 澄清，不凭算法偏好创造业务规则。

**来源线索（非条款资格）**：[A01 SWEBOK](../external-source-catalog.md)、[A20 Engineering Playbook](../external-source-catalog.md)、[A22 Google Style Guides](../external-source-catalog.md)、[A49 CS2023](../external-source-catalog.md)。