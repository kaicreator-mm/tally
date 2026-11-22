# K05.INTEGRATION · 契约测试、真实依赖与用户公开路径的证据分层

> [K05 知识库](../README.md) · [专业方法](../engineering-methods.md) · [Agent Skill](../agent-skills.md) · [反例与案例](../casebook.md) · [ADS 来源与外部学科目录](../sources.md) · [外部资料总索引](../../external-source-catalog.md)。
>
> `KNOWLEDGE_TOPIC_CANDIDATE / ORIGINAL_EXPLANATION / NOT_CLAIM_QUALIFIED / PROPOSED_NOT_CONNECTED`。Topic 是问题驱动的知识切片，不改变 Workflow、Task、Reviewer、Host 和 Release 权限；下面案例为合成教学条件，不是实际 Host PASS。

## 1. 问题与应用触发

API Consumer Contract 通过、真实数据库集成通过，却没有打开产品 UI；现在能否宣称用户完整体验 PASS？

**适用前置输入：**
1. 合同参与者、版本和合法消费者
2. 真实 DB/消息队列等依赖和目标平台
3. 用户 Critical Journey、Host 和 Release 实际运行证据

**学科原理与为什么这样决策：** Consumer Contract 验证特定生产者/消费者之间约定的交互；Testcontainers 等真实依赖集成测试验证当前运行环境下的数据库/中间件行为，但这两种范围都未必覆盖 UI、真实授权、首次配置或实际安装包。IEEE V&V 强调产物契约与用户期望的验证侧重点，不同证据应保持相应结论。

## 2. 实际决策程序（按现有权限执行）

1. 列层次：Unit/Mock、Contract Consumer/Provider、真实依赖集成、选定公开入口、最终用户 Journey/安装。
2. 对每一层明确实际执行哪些组件、哪些调用仍被替换、可观察什么业务不变量。
3. 必要时用真实 DB 验证非法状态不写入、重复请求幂等、事务回滚，但不要强制所有小变更启容器。
4. 仅当要求真实 UI/Host/部署时验证对应入口，并由原有 Owner 判定是否满足 Release 的独立条件。

## 3. 按条件选择，而不是背诵一种答案

| 场景条件 | 当前应作的判断 |
| --- | --- |
| 仅涉及协议字段 | 消费者合同可能足够作为受影响范围检查。 |
| 涉及 DB 写入/事务 | 考虑真实依赖集成。 |
| 需证明用户端到端 | 必须有选定真实 UI/应用运行，不可从前两层推断。 |

## 4. 可复用的教学案例与负例

**合成正例：** 订单 API 的 Pact 断言 `409` 错误码，Testcontainers 真实 DB 验证订单状态不变；但安装版 UI 将按钮接到了旧接口，用户路径仍失败。

**失败/反模式：** 依赖两个层面的单测绿灯，却从未安装/启动新版 UI，报告全流程业务交付。

**可证伪的结果 Oracle：** 同一错误 UI 绑定不影响 Pact/DB Test PASS，却应使真实用户 Journey FAIL；若未运行 Journey，则必须明确 `JOURNEY_NOT_RUN`。

## 5. 最小输出、拒绝和跨模块交接

**Agent 最小输出**：当前事实/适用条件、被选知识及原因、候选/未选方案的权衡、具体正反 Oracle、缺失证据和 STOP、原始来源身份。不能取得真正 Product/Task/CI/Host 证据时要标 `UNKNOWN/NOT_RUN`，不能将本 Topic 引用的外部资料当作已发布 Tally 规则。

**交接关系：** K02 管 API 接口合同，K08 管 CI/Host/Artifact 实际运行，K05 判断每层证据充分性。

**停止条件**：业务政策无权确定、错误可能影响不可逆外部副作用、需要超出合法写集的实施、无法取得必要证据时，输出受影响范围及最小合法下一步，而不是调用无界 Agent 循环。

## 6. 来源、版本与知识资格

- [Pact Specification](https://github.com/pact-foundation/pact-specification)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。
- [Testcontainers](https://github.com/testcontainers/testcontainers-java)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。
- [IEEE 1012 V&V](https://standards.ieee.org/ieee/1012/7324/)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。

**ADS 约束基线**：`kaicreator-mm/ai-development-standard@933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；若与现行项目已生效规范冲突，应以具有权威的原规则为准并说明冲突。历史 ADS #1100、Draft #1384 的统一三阶段 DAG 观点不作为现行强制规则。
