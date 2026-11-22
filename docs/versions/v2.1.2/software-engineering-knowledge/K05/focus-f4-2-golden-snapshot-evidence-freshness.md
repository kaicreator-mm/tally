# K05.F4.2｜Golden/Snapshot Oracle 的合法更新与证据时效

> [K05 课程](systematic-syllabus.md) · [第一卷](foundations-deep-dive-01.md) · [独立 Oracle](topics/FALSE_PASS.md)
>
> **R4 ORIGINAL_KNOWLEDGE_CANDIDATE / NOT_SOURCE_CLAIM_QUALIFIED / NO_SNAPSHOT_TEST_RUN。** 深化 K05.F4.2。快照是有适用条件的回归工具，不能自行产生 Expected 行为或继承旧 SHA 的全部测试权威。

## 1. Golden 从哪里获得权威？

Golden Master/Snapshot 将一次被选定的输出用作下一次的比较基线；真正难点是该基线为什么代表**经授权的正确行为**。它可以由经过检查的旧版本观察产生、由批准的例子手工给定，也可能只是错误实现首次写出的内容。工具生成快照的动作并不证明其正确。

`Observed=Expected` 可能因为错误地更新 Expected 而成立。对业务权限、不可逆 Effect 或跨租户隔离等高风险性质，需要独立于当前实现的 Oracle：批准的规则表、具备权限的决议、可反驳的不变量、或受控的真实负例。不应用一次 Snapshot 批量更新覆盖所有负例。

## 2. 黄金文件适用输入的身份

一个可追溯的 Snapshot/Golden 至少要知道：
- **Scope**：哪个行为、Actor/State/Context 和受支持的平台。
- **Input**：测试/fixture、配置、依赖、语言区域、日期/时间或排序规则。
- **Expected Owner**：谁/什么正式授权这个预期输出（可能是有权规范或经过审核的历史观察）。
- **Version**：产生 Expected 的源版本以及当前运行的源码和测试版本。
- **Transformation**：如果比较前进行规范化，哪些字段被排除及排除是否真的不影响行为。

过度宽泛的 Snapshot 既会对无害时间戳抖动报错，也会因 Reviewer 不易阅读而掩盖敏感字段变化。建议用明确字段和可读 diff 来降低审查成本，不能因为 Snapshot 很大就放弃检查。

## 3. 精确的“过期”判断必须区分三个场景

| 情境 | 允许的专业判断 | 禁止的推断 |
| --- | --- | --- |
| 当前源码 H2 完全未影响 Snapshot 输入或业务语义 | H1 的独立结论可作为不受影响历史证据复用，是否满足正式 Gate 看当前规则 | 所有旧证据无条件失效 |
| H2 修改本测试覆盖的关键行为 | 以 H2 真实跑受影响测试、人工对照有权 Expected | 复制 H1 的绿色结果 |
| H2 只修改 Snapshot 文本或生成器 | 判断 Expected 修改依据和对应负例；必要时更新语义 Oracle | 更新 Golden 后 GREEN 就证明行为正确 |

GitHub 的保护分支旧审批失效规则取决于实际设置；专业测试证据的材料当前性也取决于输入影响集。**不是把 GitHub 的某一可配置规则误写成所有知识任务的法定 Gate。**

## 4. 可证伪教学案例

给定支付状态查询 JSON：H1 返回 `{id, status:"PAID", tenantId:"A"}`，H2 在租户 B 请求时错误重用了 A 的缓存。Writer 因快照失败把 Expected 改为新的输出 `tenantId:"A"`，测试变绿。

正确的测试设计必须通过独立租户归属合同识别新快照不合法；更新 Expected 不应消除跨租户负例。相反，如果 H2 只是新增获批准的无语义日志字段，并且比较约束明确排除它，受控 Snapshot 调整可能是合法的。

对 Agent 的练习输出：原 Golden 的权威、H1/H2 实际 diff、合法变化类别、不合法变化类别、已执行 suite 与未执行受影响 suite，最后才能给限定结论。

## 5. 研究边界与来源

没有受权 Oracle、无法确认 Golden 产生时的环境、输入/测试被新 commit 改动但当前 suite 未执行时，对对应范围输出 `UNKNOWN/NOT_RUN/STOP`；不因为 Snapshot Framework 存在就宣称兼容或 DomainHarness 消费真实验证。

**公开来源线索**：[Jest Snapshot Testing](https://jestjs.io/docs/snapshot-testing)（官方明确建议与代码一起审查 Snapshot 变更），[GitHub stale review semantics](https://docs.github.com/en/pull-requests/how-tos/review-pull-requests/reviewing-proposed-changes-in-a-pull-request)，[知识目录](../external-source-catalog.md)。独立措辞、无授权 Clause 转录。
