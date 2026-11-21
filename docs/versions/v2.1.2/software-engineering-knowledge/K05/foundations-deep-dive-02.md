# K05 · 系统学科深化 02：组合覆盖、变形关系与状态空间探索

> [系统课程](systematic-syllabus.md) · [第一卷：Oracle/性质/变异](foundations-deep-dive-01.md) · [测试方法](engineering-methods.md)
>
> **R3 DOCUMENT_KNOWLEDGE_CANDIDATE / NOT_SOURCE_CLAIM_QUALIFIED / NO_TEST_EXECUTED。** 深化 K05.F2.1–F2.3、K05.F3.1–F3.2、K05.F4.3、K05.F5.1–F5.3。测试设计候选不能冒充被真实 Runner 发现的测试。

## 一、组合爆炸与覆盖强度是不同概念

若 Actor 有 3 种、状态有 4 种、命令有 5 种、环境有 3 种，则完全笛卡尔积为 3×4×5×3=180 个候选组合，还未计时间序列。**覆盖阵列（Covering Array）**可在明确交互强度下选取较少测试，例如 Pairwise 只保证每对参数值至少被覆盖一次，**并不保证三元或四元故障被发现**；若业务合同有角色×状态×命令三元条件，则 Pairwise 可能遗漏关键失败。

可行做法：先把已知业务风险（非法付款、重复提交、权限跃迁）作为强制列出的可反例，再对其余组合使用 Pairwise/边界抽样。若必要值在某些条件下不合法，约束需显式写入生成器，不可悄悄删除所有“难处理”组合并声称全覆盖。

## 二、缺少完整 Oracle 时可考虑变形测试

某些计算没有容易手写的 Expected（如大型排序、解析、搜索排序的全量输出），但可以写**变形关系**：
- 排序输出再次排序后应按已批准相等性保持一致；
- 去重后再去重，其可观察集合不变；
- 在不影响语义的字段顺序置换下，某些 JSON 解析的合法业务对象可能保持等价。

这些关系依赖前提：排序 comparator 必须满足所需一致性，输入操作确实保持业务含义；例如“任意金额加倍时结果也加倍”对带阶梯折扣的产品就是错误变形关系。变形测试可减轻 Oracle 问题，但**不证明没有未知错误**。

## 三、状态空间探索：安全性、活性与抽象边界

对业务 Job 规定离散状态和可用动作：READY、CLAIMED、APPLIED、ACKED、EXPIRED；选择关键操作重复、超时、崩溃、Worker 交错的短序列。针对 safety 写出“同一合法 Job ID 不产生重复不可逆 Effect”；针对 liveness 写出在明确公平性/依赖可用的假设下最终达到终止或可恢复状态。

实际使用 TLA+ / 模型检验时，记录 Spec 版本、状态界限、对称性简化、搜索限制及反例轨迹；有限状态模型 PASS 只能说明**被编码的抽象与选中属性**在假设下未找到反例，不能自动证明所有真实系统行为。没有执行模型检验就保持 NOT_RUN。

## 四、Flaky 问题是推断问题，不是无界重试

同一测试有时成功、有时失败，可能来自调度竞争、时钟、共享环境、外部网络或被测试代码的非确定性。重跑可用于判断变动，但要记录次数、失败率、环境和源码是否保持一致，不能用最后一次通过覆盖历史失败。

**教学对照**：
- A：测试依赖真实时间窗口，时钟扰动会系统性改变结果。
- B：不同线程对同一临时数据库写入造成污染。
- C：CI Runner glob 使目标测试从未发现。
A/B 是可能的 flaky 机制；C 不是“偶发不稳定”，而是材料测试的 NOT_RUN。修复方式和验证结果都不应混为一谈。

## 五、测试组合的真实专业验证框架

| 要回答的工程问题 | 合理证据 | 典型错误晋升 |
| --- | --- | --- |
| 业务不变量是否有覆盖？ | 有权 Rule 与正反 Oracle | 用行覆盖率代替行为覆盖 |
| 组合交互错误是否被发现？ | 明确 t-way 强度与未覆盖例 | Pairwise = 完整功能测试 |
| 状态迁移正确吗？ | 模型/实现对照、故障交错 | 抽象模型 PASS = Host E2E |
| 测试能检测回归吗？ | 变异体/故意错误被识别 | Mutation Score = 全面正确性 |
| 测试真的运行了吗？ | Runner discovery、当前 SHA、执行清单 | 零发现还称 PASS |
| 优化是否值得？ | 风险和额外运行成本对比 | 所有提交强制跑所有方法 |

K05 拥有测试目的和 Oracle，K08 只负责真实 Provider/Runner/Artifact 执行事实；所有新增专业测评仍基于现有 Task/Review/Release 权威，不创造新的 CI Gate。

**来源线索**：[A08 ISO 29119](../external-source-catalog.md)、[A14 ISTQB](../external-source-catalog.md)、[A25 Testcontainers](../external-source-catalog.md)、[A50 TLA+ Tools](../external-source-catalog.md)、[A54 ACM SIGSOFT Empirical Standards](../external-source-catalog.md)。