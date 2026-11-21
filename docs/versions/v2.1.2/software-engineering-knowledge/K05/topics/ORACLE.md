# K05.ORACLE · 由需求与不变量推导测试 Oracle，防止以实现自证

> [K05 知识库](../README.md) · [专业方法](../engineering-methods.md) · [Agent Skill](../agent-skills.md) · [反例与案例](../casebook.md) · [ADS 来源与外部学科目录](../sources.md) · [外部资料总索引](../../external-source-catalog.md)。
>
> `KNOWLEDGE_TOPIC_CANDIDATE / ORIGINAL_EXPLANATION / NOT_CLAIM_QUALIFIED / PROPOSED_NOT_CONNECTED`。Topic 是问题驱动的知识切片，不改变 Workflow、Task、Reviewer、Host 和 Release 权限；下面案例为合成教学条件，不是实际 Host PASS。

## 1. 问题与应用触发

新实现产生的输出能否直接作为 Expected Snapshot？如果用户与设计来源没有明确写出一个状态转移怎么办？

**适用前置输入：**
1. 真实接受的 Requirement/Rule/Invariant 与版本/角色
2. 实际公开输入/输出及持久副作用
3. 正常、拒绝、边界、重复或竞态条件

**学科原理与为什么这样决策：** 测试 Oracle 必须从有来源的业务规则、合同或可辩护的不变量推导，而不是把当前实现的行为抄成期望。ISTQB 的决策表、状态转换、边界值技术帮助穷举重要差异；NASA 需求验证强调需求的可核对性与一致性，但其高保证流程只作为专业参考，不能直接移植到所有项目。

## 2. 实际决策程序（按现有权限执行）

1. 先写业务真值表：角色、前置状态、操作和期望结果，缺少有权政策的单元格标 UNKNOWN。
2. 选择能区分错误实现的正/反案例：正常变更、非法状态拒绝、数据库无副作用、重复请求。
3. 确认断言观察的是实际业务状态/消费者接口，不只 Mock 函数被调用；若需真实依赖，指定 K08 执行环境。
4. 故意考虑一个错误实现能否在当前测试下仍通过；如果能，补一个最小分离反例而不是增加十个同质 happy-path。

## 3. 按条件选择，而不是背诵一种答案

| 场景条件 | 当前应作的判断 |
| --- | --- |
| 有批准规则 | 可产生具有正反例的候选 Test/Acceptance。 |
| 只有新实现输出而无来源 | Oracle 不成立，应返回 K01 澄清或保留 UNKNOWN。 |
| 纯静态/语法要求 | 可复用已有类型/构建，但不能宣称业务 Oracle 已被验证。 |

## 4. 可复用的教学案例与负例

**合成正例：** Product 明确只有 OPEN 订单允许 CLOSE，关闭 CLOSED 应返回业务冲突且状态不变。负例执行真实 API→DB 发现重复关闭错误；只记录 `handler_called=true` 无法验证状态。

**失败/反模式：** 开发者同步更新 Golden 为候选实现的错误响应，从而让测试 PASS，却违反已批准业务不变量。

**可证伪的结果 Oracle：** 若故意移除非法状态守卫，至少一条负例必须 FAIL；否则 `ORACLE_INSUFFICIENT`（知识描述，非新增 Gate）。

## 5. 最小输出、拒绝和跨模块交接

**Agent 最小输出**：当前事实/适用条件、被选知识及原因、候选/未选方案的权衡、具体正反 Oracle、缺失证据和 STOP、原始来源身份。不能取得真正 Product/Task/CI/Host 证据时要标 `UNKNOWN/NOT_RUN`，不能将本 Topic 引用的外部资料当作已发布 Tally 规则。

**交接关系：** K01 确认业务政策，K08 运行 Provider/Host，K03 修复代码。

**停止条件**：业务政策无权确定、错误可能影响不可逆外部副作用、需要超出合法写集的实施、无法取得必要证据时，输出受影响范围及最小合法下一步，而不是调用无界 Agent 循环。

## 6. 来源、版本与知识资格

- [ISTQB CTFL v4.0.1](https://istqb.org/certifications/certified-tester-foundation-level-ctfl-v4-0/)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。
- [NASA SWE-055 Requirements Validation](https://swehb.nasa.gov/spaces/SWEHBVD/pages/102695440/SWE-055%2B-%2BRequirements%2BValidation)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。
- [ADS Testing Standard](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/TESTING_STANDARD.md)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。

**ADS 约束基线**：`kaicreator-mm/ai-development-standard@933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；若与现行项目已生效规范冲突，应以具有权威的原规则为准并说明冲突。历史 ADS #1100、Draft #1384 的统一三阶段 DAG 观点不作为现行强制规则。
