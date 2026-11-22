# K01.RULE_CONFLICT · 冲突需求如何定责、裁决与保留未知

> [K01 知识库](../README.md) · [专业方法](../engineering-methods.md) · [Agent Skill](../agent-skills.md) · [反例与案例](../casebook.md) · [ADS 来源与外部学科目录](../sources.md) · [外部资料总索引](../../external-source-catalog.md)。
>
> `KNOWLEDGE_TOPIC_CANDIDATE / ORIGINAL_EXPLANATION / NOT_CLAIM_QUALIFIED / PROPOSED_NOT_CONNECTED`。Topic 是问题驱动的知识切片，不改变 Workflow、Task、Reviewer、Host 和 Release 权限；下面案例为合成教学条件，不是实际 Host PASS。

## 1. 问题与应用触发

产品 Issue 说“发货后不可取消”，旧客服手册说“付款后均可取消”。两个来源是否真的冲突，还是角色、时间与商品条件不同？

**适用前置输入：**
1. 现行 Product/需求的权威入口及明确发布版本
2. 两条原始需求逐字出处、角色、状态、有效时间窗
3. 已接受业务不变量和不可逆副作用

**学科原理与为什么这样决策：** 软件需求应具有可识别来源、适用条件和独立可验证性。需求冲突不只是两段文字不同；要先把主体、业务事件、当前状态、版本和权限范围规格化，再判断是否能同时成立。来源的最近更新时间不足以单独证明更高权威。NASA 的需求验证材料重视需求清晰、一致、可检验；此处只借鉴方法，不套用其项目级强制流程。

## 2. 实际决策程序（按现有权限执行）

1. 把两条要求分别转成 (Actor, PreState, Event, Context) → ExpectedOutcome/SideEffect，缺失处标 UNKNOWN。
2. 做重叠性检查：两要求是否适用于同一个角色/时间/商品/业务状态？若不重叠，则不是同一条件上的逻辑矛盾。
3. 若确属同条件不兼容，展示两种具体候选政策及对状态机、API、测试的不同影响；只把决策交到合法 Product/Rule Owner。
4. 无冲突部分继续可观察验收准备；受冲突的单独 HOLD，并绑定最小的需要澄清事实。

## 3. 按条件选择，而不是背诵一种答案

| 场景条件 | 当前应作的判断 |
| --- | --- |
| 仅角色不同且来源确实分别授权 | 分别建规则/验收，不强制人工裁定。 |
| 相同条件同一业务身份却要求允许/拒绝 | 冲突，不任选更方便的政策。 |
| 无原始来源或已过期 | UNKNOWN/STALE，不补造业务规则。 |

## 4. 可复用的教学案例与负例

**合成正例：** 测试订单状态 PAID、运输状态 SHIPPED，普通消费者请求 CANCEL；现行规则若明确拒绝，则期望错误和原状态保持。另一位授权客服存在单独的售后补偿流程，不能据此直接覆盖消费者取消规则。

**失败/反模式：** 把旧客服资料的“均可取消”按同一 API 无条件实现，导致已出库订单重复退款。即使新代码通过编译，也不是业务验收通过。

**可证伪的结果 Oracle：** 输出矩阵至少展示相同条件上的冲突/无冲突分支，并标明谁有最终裁定权；模拟相同 Actor+State 的允许与拒绝源互斥时必须给出 HOLD，不能 ACCEPT 其中之一。

## 5. 最小输出、拒绝和跨模块交接

**Agent 最小输出**：当前事实/适用条件、被选知识及原因、候选/未选方案的权衡、具体正反 Oracle、缺失证据和 STOP、原始来源身份。不能取得真正 Product/Task/CI/Host 证据时要标 `UNKNOWN/NOT_RUN`，不能将本 Topic 引用的外部资料当作已发布 Tally 规则。

**交接关系：** 交给 K01.ACCEPTANCE 形成验收后再到 K02 接口/状态设计、K05 Oracle；不自行建立 Product Freeze。

**停止条件**：业务政策无权确定、错误可能影响不可逆外部副作用、需要超出合法写集的实施、无法取得必要证据时，输出受影响范围及最小合法下一步，而不是调用无界 Agent 循环。

## 6. 来源、版本与知识资格

- [ISO 29148:2018](https://www.iso.org/standard/72089.html)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。
- [NASA SWE-055 Requirements Validation](https://swehb.nasa.gov/spaces/SWEHBVD/pages/102695440/SWE-055%2B-%2BRequirements%2BValidation)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。
- [IREB CPRE](https://cpre.ireb.org/en/downloads-and-resources/downloads)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。

**ADS 约束基线**：`kaicreator-mm/ai-development-standard@933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；若与现行项目已生效规范冲突，应以具有权威的原规则为准并说明冲突。历史 ADS #1100、Draft #1384 的统一三阶段 DAG 观点不作为现行强制规则。
