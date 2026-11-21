# K04 专业深化：风险驱动流程、信息缺口与有界任务决议

> [K04 系统课程](systematic-syllabus.md) · [R6 任务偏序与关键路径](advanced-task-graph-critical-path-and-bounded-delivery.md) · [R4 验收与授权 Focus](focus-f2-3-acceptance-and-authorization.md)
>
> **R9 DOCUMENT_CANDIDATE / NO_LIVE_TASKS_OR_AUTHORITIES_CREATED。** 精确深化 **K04.F1.3 复杂度与风险驱动流程**和 **K04.F4.1 事实缺口/研究目标** 两个尚无 A 的课程 ID。**K04.F2.3 验收与授权已有独立接受的 R4 Focus**，这里只交叉引用和构造材料性反例，不为了增加链接重写它。

## 一、工作流应该由需要降低的未知和风险决定，而不是由固定仪式数量决定

软件工程过程包括发现问题、分析、设计、实施、验证、验收及反馈，但不是每次改一行文档都必须建立独立 Planner/Builder/Reviewer/Release 子 DAG。专业流程应至少由以下现实变量决定其深度：

`Risk = (scope and reversibility, failure severity, uncertainty, verification cost, evidence owner, external effect exposure)`。

这些量不是可直接相加的统一分数。**授权与不可逆操作是硬约束**，不能以低经济风险或高 Token 成本折抵。普通文档目录错链可直接定位并做链接校验；支付状态幂等异常即便只改一行，也可能要求明确 Provider 的 Effect 事实、业务 Rule、授权 Actor 与故障交错的独立 Oracle。

### 三个条件化最小流程

| 任务情境 | 有材料价值的最小工作和证据 | 常见无效扩张 |
| --- | --- | --- |
| 修正文档锚点 | 对真实 Markdown anchor、链接和受影响导航做核对，记录 Exact SHA | 默认强制 Windows Installer E2E、全量单元测试 |
| 修改只读 API 的数据转换 | 类型/Contract、生成代码与真实选定 Consumer 受影响测试；必要时限定集成 | 编译通过就宣称全部 Consumer 验证通过 |
| 修改资金业务 Effect | 权威规则、状态 Owner、Provider idempotency/查询合同、超时/重复负例、真实授权与适当 Oracle | 把远端超时自动判未扣款，或让一个 Builder 自己给自己 Release |

有意义的是证明目标**被许可地解决了问题**，不是证明走过最多阶段。真实 GitHub Required Check、合同约定的独立审查或目标 Provider 验证仍必须执行；“等价环境优先”不能用于跳过不可替代的强制验证。

## 二、将任务拆分为必要事实决议、实施与观察

候选需求：“修复某跨租户缓存泄漏并发布。”至少存在：
- Rule：Tenant 隔离的业务/访问授权是否已由有权 Owner 确认？
- Cause：跨 Tenant 返回同一结果是否源于 Cache Key 遗漏 Tenant，还是复用了有问题的 Session？
- Change：需编辑的准确组件及其语义读写集合是什么？
- Oracle：能否在两个 Tenant 共用实体 ID 的场景下观察到**真实隔离**？
- Release：当前 Exact SHA、实际 Artifact 与所需 Required Checks 是否满足有权发布条件？

这些是**依赖事实**，不自动等于五个需要不同 Agent 的 Issue。若一个 Builder 可以在有权事实成立后完成局部修复并真实验证，额外调度可能只增加等待、上下文重述和工作区冲突。

K04 的 DAG 仅画**真实先后约束和资源互斥**。例如“先拿到有权 Tenant 权限合同再把缓存修改合入生产”的偏序不可由“把更多 Agent 并行起来”消除。文档处理与只读验证在不依赖合同时则可独立推进。

## 三、事实缺口要区别于实现缺陷

给出合成任务：
```text
Source A: 未注明生效日的旧产品 FAQ
Source B: 一份未签署的当前策略草案
Source C: 某日服务端日志中的已提交状态
Goal: 立即将付款重试改为自动双次提交
```
缺少的是**有权业务规则/Provider 幂等和状态读取合同**，并不必然意味着现有代码已经存在某个可修复 bug。此时派十个 Builder 修改源码不构成风险降低；正确下一个步骤是提一个有辨识力的问题给真实 Owner，或在严格 Mock/只读范围内分析安全备选。

“已发现问题”也有三个层次：
1. **Known fact**：有当前来源/版本/范围的证据；
2. **Hypothesis**：在有限观察下提出可被新事实推翻的解释；
3. **Unknown/ASK**：缺有权 Source、必要系统输入或没有权限查询的事实。

把 UNKNOWN 自动写成假设通过会污染后续 Task DAG。把所有小的事实不确定都提交 Product Owner 又会产生不必要的人为阻塞；能从当前授权 GitHub/源码/文档直接核实的事实应直接核实。

## 四、信息价值与停止/推进条件

可以对调研候选建立简单**价值信息假说**：
`ExpectedBenefitOfProbe ≈ expected avoided wrong-decision loss − probe/time/opportunity cost`。
但概率/损失若无实测就只能标为教学假设。对不可逆 Effect，即使数字“有利”，未获授权也不可执行。

示例（合成）：查找目标 Runner 的实际 Test Discovery manifest 可在几分钟内证伪“测试已经执行”的说法，比无关地重复所有 CI 更有信息增益。反例是必须完成真实 OS Installer 的发布任务：无论对 Runner manifest 做多少静态核查，也无法替代真实安装/首次启动的能力验收。

最小决策记录可限定于：
```text
Goal/Owner -> Scope/Source SHA -> Known/Unknown -> Candidate mechanisms ->
Falsifier/Oracle -> Permission -> STOP or bounded next action
```
该概念记录不是新的正式 Task Wire Schema，也不要求每次都造一个独立审批文件。信息已足够且安全可行动时继续；不能跨越的业务权限或外部 Effect 仍保持 STOP。

## 五、反证：流程越多可能使错误更难发现

一个流程在同一任务上执行四轮“Review PASS”，但四轮都引用来自 Writer 自己写的相同错误 Expected。如果没有独立合法 Oracle，Review 数量增加不能改善正确性。另一个流程只做一轮针对真正受影响的消费者合同测试，若已经覆盖权威规则并保留当前 SHA/真实执行记录，可能比许多模拟验证更有意义。

然而不能由此反推“独立 Review 没意义”；当项目要求独立 Reviewer 时，必须留下真实可反驳的独立结论。专业优化关注**每个必要活动是否增加不同且合法的证据**，不能机械按活动数压缩至零。

## 六、Owner 和跨模块交接

K01 对目标/规范有效性负责；K02 对真实状态/架构边界负责；K03 负责实现；K05 定义独立 Oracle；K06 调查根因；K07 管理 Agent 身份与消息；K08 建立制品/平台证据；K11 对风险成本和信息价值作研究。K04 不创建第二个 Source Owner 或强制本轮新 Gate。

专业来源：[A04 ISO 12207 生命周期](../external-source-catalog.md)、[A10 ISO 15939 度量](../external-source-catalog.md)、[A37 CMMI](../external-source-catalog.md)、[A54 ACM SIGSOFT Empirical Standards](../external-source-catalog.md)。本章没有实际 Task/Agent/CI/Host 执行，不把经济性假说写成测量结论。