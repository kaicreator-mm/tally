# K01 专业基础深化：需求可追溯、语义边界与变更影响证明

> [K01 系统课程](systematic-syllabus.md) · [R5 需求证据与业务合同](advanced-knowledge-from-evidence-to-contract.md) · [全库覆盖台账](../foundation-concept-coverage-ledger.md)
>
> **R7 ORIGINAL_THEORY_CANDIDATE / NO_REAL_REQUIREMENT_TRACE_VERIFIED / NO_POLICY_ADOPTION。** 主要针对原课程中尚无 A 进阶章节的 K01.F1.1、F1.2、F1.3、F3.1、F4.2、F5.3 六个概念，另连接 K01.F4.3。这里构造的是可审核的专业推理，不是要求所有 Tally Task 强制新建需求矩阵。

## 一、需求为何会在实现时丢失语义

从访谈得到“管理员可以重试付款”不是一个可以直接写代码的合同。“管理员”是否包括租户管理员？“重试”是再次读取结果还是再次触发付款？何种订单状态合法？外部支付平台是否提供幂等？如果不确认，系统实现者可能无意识地把 **业务目标**转换成一个风险极高的**实现手段**。

至少分清：
- Need/Goal：用户期望解决什么业务损失，例如“付款超时后能确认订单是否已经支付”；
- Stakeholder/Authority：谁提出观察、谁代表实际使用者、谁有权规定支付与退款；
- Business Rule：在具体 Actor/State/Event/Time/Scope 下允许/禁止什么；
- Quality Scenario：异常网络与目标恢复时间/用户体验如何被衡量；
- Solution Mechanism：远端状态查询、业务幂等键、补偿流程是否可用。

若最初需求只是“确认付款”，工程师擅自实现“失败就再次付款”就属于典型的**目标—方案混淆**。即使单元测试完美复现该实现，也不证明满足有权业务要求。

## 二、双向可追溯不是文件互相链接

教学跟踪：
```text
Business Goal G1: timeout 后不误报重复扣款
   -> Authorized Rule R1: 未确认远端结果不得盲目二次付款
   -> Software Behavior S1: EFFECT_UNKNOWN 触发只读查询或有权人工处理
   -> Design Decision D1: order state owner 与 provider lookup seam
   -> Implementation C1: check-before-retry branch
   -> Oracle T1: remote-committed + lost ACK + redelivery
   -> Evidence E1: exact-SHA real selected provider run (NOT_RUN)
```

这条链中，前面五种都是**知识/设计候选**；E1 在未执行前不能被标为 PASS。双向追溯的意义不只是从 Goal 往下找到实现，还包括：从某个代码/测试/缓存/Build 工作倒查“为什么要存在”和它的真实有权合同来源。**链接存在不等于被链接的材料都合法**。

NASA [SWE-052 Rev D](../external-source-catalog.md) 是高保证工程的原始参考，其明确的 NASA 项目适用性**不能**自动转为 Tally 所有任务的强制 Trace Gate。对小任务可以保留单条 Source/Rule→Code/Oracle 链，范围大时才需要完整的跨工件关系视图。

## 三、完整性、正确性与冗余区别

给定 Goal G1 和另外一个需求 G2“用户能查看当前订单状态”，若实现了 T1 并不能证明 G2；反之，一个无任何批准需求支持的 UI 显示“付款一定成功”，即使截图和前端测试都有，也可能是**无有权根据的多余或错误功能**。

专业区分：
1. 正向链追踪检查“有权需求是否被遗漏”；
2. 反向链追踪检查“代码/测试/声明是否有合法依据”；
3. 范围检查发现跨租户、不同发布版本、不同业务状态错用；
4. 反例检查验证“Oracle 是否真正能排除相反行为”。

不能把正反向链接数当作需求满足率。例如每个 Goal 都被标记到 `README.md` 也不能证明已执行或满足 Goal。

## 四、业务聚合/不变量与代码结构如何衔接

领域上的 `Order` 聚合候选应拥有订单状态跃迁的不变量，且更新点需要真实一致性机制。**聚合**不是“把所有与订单有关的表放一个对象”：关键是哪些业务事实在同一个一致性边界内必须同时成立。远端财务 Effect 不因被命名为 Order Aggregate 而加入本地事务。

如果产品具有多个 Tenant，需求的 Actor/Owner 语义必须影响读取、缓存和修改的权限检查；K02 拥有物理分解/信任边界，K03 拥有条件提交实现，K05 拥有独立测试。K01 提供的是**有权规则、领域语言和可验收事实**，不能依靠对象建模自动批准外部 Effect。

## 五、变更传播：先计算影响，不能无界全量复核

给出有权业务更改：R1 从“`PAID/PACKING` 可取消”缩小到“只有 `PAID` 可取消”。影响传播应至少识别：
- 规则作用域和有效日期；
- 当前仍允许 `PACKING` 的行为/服务端入口；
- 条件写、缓存、客户端按钮、外部副作用/补偿的相关路径；
- 明确引用 R1 的正向与反向 Oracle；
- 已按旧 SourceSHA 通过的受影响范围和仍可合法复用的无关历史范围。

`ChangeImpact(R1)` 并非仅看 git diff 中出现“PACKING”这个字符串。代码可能把状态转换封装成 enum/映射，语义依赖不能全部由字符串检索证明；另一方面无关的 README 修改也不应触发无限重跑所有真实 Host。

**区分性反例**：一个 Agent 仅修改前端按钮使 `PACKING` 隐藏“取消”，但旧 REST API 仍允许旧客户端发取消。这满足 UI 截图测试，不满足实际业务不变量。相反，如果完全不支持 `PACKING` 的实际 Consumer 只处理只读报表，不能凭同一个仓库就宣布该 Reader 受影响。

## 六、来源合法性与暂时假设

| 输入 | 可做的专业判断 | 不可做的资格晋升 |
| --- | --- | --- |
| 旧 FAQ 与新正式文档未确认生效期 | 定义冲突条件、列出 ASK 项 | 直接决定 R1 已替代 |
| 已有权政策与本地 Mock | 提出功能合同与负例 | 宣称真实远端支付安全 |
| 真实 API 文档但 Provider 幂等时限未知 | 区分合理架构备选与恢复风险 | 保证无限次重试都是 exactly-once |
| 已确认政策 + 当前 SHA 的适当 Oracle PASS | 报告受检功能/环境的结果 | 自动签发 Product/L2/Release |

**专业演习**：从“管理员付款重试”这一陈述导出两个互相竞争的业务目标（再次触发付款 vs 查询既有结果）；给出能区别目标的用户问题、一个会暴露重复扣款的最短失效轨迹、一个上下游双向追溯路径、一次有作用域的需求变更，并解释何时停止错误的实现提交、何时可继续其他任务。

**来源线索**：[A05 ISO/IEC/IEEE 29148](../external-source-catalog.md)、[A12 IREB](../external-source-catalog.md)、[A13 IIBA](../external-source-catalog.md)、[A44 NASA Requirements Validation](../external-source-catalog.md)、[A65 NASA SWE-052 Rev D](../external-source-catalog.md)。均为研究和合法参考，当前没有原始 Clause 签署、生产 Policy 或现实 Agent/Host 证据。
