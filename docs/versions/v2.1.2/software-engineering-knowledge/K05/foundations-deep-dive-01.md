# K05 · 学科深化 01：测试 Oracle、状态模型与性质检验的知识基础

> [系统课程](systematic-syllabus.md) · [Oracle 专题](topics/ORACLE.md) · [False PASS](topics/FALSE_PASS.md) · [真实集成](topics/INTEGRATION.md)
>
> **ORIGINAL_EXPLANATORY_KNOWLEDGE_CANDIDATE / NOT_SOURCE_CLAIM_QUALIFIED / TEST_NOT_RUN**。深化 K05.F1.1/F1.2/F1.3、K05.F2.1/F2.2/F2.3、K05.F4.1 和 K05.F5.2。教学测试设计不是实际的 Host/CI 结果。

## 1. Oracle 为什么比测试数量重要

测试结果的核心不是 `exit=0`，而是 **选中的受测对象在指定条件下与独立期望相比较**。Oracle 来源可来自获权业务规则、接口合同、不变量或已认可的性质；不能以实现代码当前输出作为它自己的正确性证明。即使运行了几千个测试，未检验关键行为也无法证明产品目标。

V&V 区分：Verification 检查对给定合同的符合性，Validation 检查产品在使用情境是否满足真实目标。它们需要相应输入/观察，代码编译成功并不自动提供真实用户路径证据。

## 2. 选例要能区分相互竞争的错误实现

假设已批准业务行为是 `OPEN -> CLOSED` 成功、`CLOSED -> CLOSED` 重复请求不得重复结算。测试设计不仅覆盖一个成功 CLOSE，还应有非法角色、旧状态版本、重复请求、失败中断和副作用次数。若重复请求策略在合同里未知，相关 Expected 标为 UNKNOWN。

| 测试技术 | 解决的问题 | 不能据此推断 |
| --- | --- | --- |
| 等价分区/边界 | 不同输入类别和上下界是否改变行为 | 每个具体输入均已运行 |
| 决策表 | Actor×State×Action 的组合及互斥政策 | 任意未批准政策已被业务接受 |
| 状态转换 | 非法边、重复事件、故障时状态 | 只测最终字段就验证外部副作用 |
| 性质/变形 | 任意输入的守恒、单调、幂等等性质 | Property 无论 Generator 如何均有效 |
| 变异测试 | 故意改变实现，看测试能否杀死错误变异 | Mutation Score=100% 就完成产品验收 |

**性质示例**：按合法合同“批处理可重试但不重复写”设计 `apply(apply(s, r), r)` 与 `apply(s, r)` 的可观察持久效果相同。若允许计费、通知等外部 Effect，必须明确它们是否也属于该幂等合同；不然性质过强或过弱。

## 3. 属性测试不是随机打印大量输入

Generator 应覆盖真正危险的输入结构：空集、单元素、重复身份、边界状态、交错调度、非法字段。Shrinking 有助于定位最小反例，但它不独立保证期望正确。可将模型状态机作为独立 Oracle，把随机操作序列投到实现；模型本身依然需要业务来源审查。计算密集的性质测试按风险和成本选用，不成为每次改文案都必跑的检查。

**教学反例**：测试生成器只产生 `OPEN` 且用户角色恒为 admin，于是 10 万次测试从未触达“无权限取消”；测试数量增长不意味着覆盖到关键政策。

## 4. 防止不存在的测试被报告 PASS

受影响测试识别须记录：目标 suite 的匹配规则、discovered test 列表、执行环境/Source SHA、实际断言以及材料行为路径。Runner exit 0 只证明该命令的执行状态；若发现 0 个目标测试，即使退出 0 仍是目标 `NOT_RUN`。K05 判断测试充分性，K08 记录 Runner/Build/制品的执行来源，K07 判断不同 Agent 结果是否属于当前 HEAD。

在真实集成验证中，Mock 只证明调用符合模拟合同，不能证明真实 DB/HTTP/UI 公共入口工作。既有等价验证可复用，但不能拿一个旧 SHA 的 Mock 代替明确要求当前真实 Host 的测试。

## 5. 限定评估的正反观察

正例：先保留被批准的非法状态 Oracle，修复 Runner glob 后真实发现目标合同测试，执行并产生可读回结果；只在真实范围报告 PASS。负例：更新 Snapshot 迎合错误实现、跳过未发现测试、无视跨版本真实消费者，均不是有效验收。

**来源线索**：[A08 29119](../external-source-catalog.md)、[A09 IEEE 1012](../external-source-catalog.md)、[A14 ISTQB](../external-source-catalog.md)、[A25 Testcontainers](../external-source-catalog.md)、[A26 Pact](../external-source-catalog.md)。均为原理/实践参考，不是当前项目新增强制测试政策。