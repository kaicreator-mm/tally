# K05：从不变量推导高区分力 Oracle、竞争形态与覆盖账本（R13 候选）

> [系统课程](systematic-syllabus.md) · [测试充分性](advanced-oracle-adequacy-and-test-selection.md) · [V&V 与真实执行](advanced-verification-validation-and-oracle-qualification.md) · [K03 文件写入提交边界](../K03/advanced-concurrent-local-file-write-convergence-r13.md) · [知识增量 #537](https://github.com/kaicreator-mm/tally/issues/537)
>
> **R13 REPORT_DERIVED_KNOWLEDGE_CANDIDATE / TESTS_NOT_EXECUTED / NOT_ORACLE_QUALIFIED。** 原归属 K05.F2.3 性质/组合测试、F5.1 风险优先、F4.1 实际发现/执行；跨 K01 验收与 K03/K10 恢复。#534 的 R1/R2 反馈指出「较多测试没有覆盖材料性错误形态」，但 #535 原始源码/QA 尚未独立复算；本章不为它们颁发技术验证资格。

## 1. 先从性质生成形态，不从实现代码生成断言

**反例导向选择**：给定实际有权的 Requirement/Invariant（如「同一作用域内，重复操作不得增加提交次数」），列出最小违反它的状态/事件交错，再找能**区分有缺陷实现和合法实现**的输入。衡量质量的是「新测试能推翻哪项错误机制」，不是「测试条数增加了多少」。

| 不变量 / 合法范围 | 高区分力事件形态（用于通用教学，不是某产品完整 QA） | 必须直接断言 |
| --- | --- | --- |
| 范围内身份唯一 | 两个写入者从未读到对象开始、同时创建且互相回读 | 最终身份数/内容摘要/业务 Effect 次数与失败原因 |
| API 返回有明确完成边界 | 在最后一次提交/响应之间打断并恢复 | 返回后状态是否符合合同，是否仍有被禁止的后台变更 |
| 旧版输入兼容 | 同一受权历史输入跨升级版本比较 | 旧行为（合法返回、错误、数据保留）的已冻结 Expected |
| 版本损坏 fail-closed | 缺/未知/未来版本、错误校验和、字段语义矛盾 | 原有字节保持不变；没有成功标志/不可逆副作用 |
| 状态机只允许合法跃迁 | CANCEL 与 COMMIT/重试并发、终态重复提交 | 一致的最终状态或受权 UNKNOWN，不能跳过非法边 |
| 非法读取不得泄漏资源存在性 | 对同一个不透明 ID 在不同作用域中查找 | 预先有权定义的错误/可见字段，拒绝越权读取 |
| Test Runner 的必要目标必须被执行 | 有一个已知应 FAIL 的负例但 runner glob 不选中它 | testId discovery/run/count/exit, 不允许仅凭总体 GREEN 认定 |

测试环境必须选择与风险等价的资源：进程内 Promise 并发、真正多进程并发、真实文件系统、模拟 crash、实际 OS kill/power loss 各自证明**不同**边界；不可把 `Promise.all` 测试当作对真实跨进程独占的充分证明。

## 2. Oracle 账本必须有明确的未验证面

对公开 Requirement/Acceptance 的每个材料条目写一行：

```text
Requirement_ID | Source/Owner/Applicability
Invariant | Oracle_ID | Positive/Negative/Boundary
Expected_source | Exact_source_SHA | Runner/environment
Discovered_testIDs | Executed_testIDs | Actual_assertions
PASS/FAIL/BLOCKED/NOT_RUN/NOT_APPLICABLE
Coverage=TESTED/PARTIAL/NOT_INDEPENDENTLY_VERIFIED
Limitations / mutation counterexample
```

**两种覆盖不同**：`7/7 selected tests PASS` 只说已实际选中的七个用例符合选定 Expected，不等于 `8/8 FR 已验`。当一个 FR 如「损坏版本」没被隐藏 QA 覆盖，就必须明确标记 `NOT_INDEPENDENTLY_VERIFIED`；本地自测通过可以作为另一组**限定**证据，不能篡改独立验证状态。

**断言应直接针对受权终态**：例如发生竞争时，检查最终存储快照、身份数、外部 effect、公开错误分类；如果只在清理阶段碰巧收到 `ENOTEMPTY`，它可提供故障症状但不足以判定具体业务不变量已经被违反。随后用最小探针区分「业务根因」与「测试 fixture/cleanup 不隔离」。

## 3. 性质、变形与公平的重复执行

- **性质型 Oracle**：`reapply(migrate(x))` 不应该再改变有权提交的逻辑状态（前提是合同定义该种幂等）；同一版本数据 roundtrip 的稳定性不能偷换成跨版本所有字段逐字节一致。
- **变形型 Oracle**：固定任务分布/输入/Runner/环境，只改变并发度，验证核心提交不变量在负载变化时仍成立；每次只声明观察到的差异，不能从性能变化直接推出知识收益。
- **反例差分**：同一缺陷在原实现存在、最小修复后消失，或对两种合法实现选择各自满足的相同公共合同。允许不同架构，但 Expected 必须共同且提前冻结。
- **并发稳定性**：竞态用例有时失败有时成功，保留**所有**执行历史、随机种子、环境与真实 testId；预登记重复次数及停止条件，不允许重试到 GREEN 后抹去红色结果。
- **正/负/边界补足**：不能只测有效输入，也不能把所有无效输入混成一种无差别的 NEGATIVE；为误区分高风险分支设计最小对照。

## 4. R1/R2 给我们的实验方法启示（未独立复核）

R1 报告涉及未冻结 legacy 输入行为；R2 报告涉及并发交错和标识符错误语义，同时有材料性 FR 未覆盖的隐藏 QA 缺口。由此可形成**候选审题规则**：

1. 把公开行为的正例、反例、**历史合法输入保持**与环境/故障边界分别列出；缺失不能从整体 PASS 中推断。
2. 每个高严重度 invariant 至少设计一个**能够使错误实现失败**的形态；若现有测试无法区分，补充针对性的 negative/mutation，而不是机械增加测试条数。
3. 每次 QA/独立 Reviewer 独立于被测 Builder，给出实际条件、Hash 和可恢复原始日志；若同一个协调者完成开发、QA、裁决，明确其独立性限制。
4. 验证结束后再对知识命中分级：`DIFFERENTIATING_SUPPORTED`、`REINFORCING`、`REDUNDANT_WITH_ADS_OR_TASK`、`INERT`、`NEGATIVE_TRANSFER`、`UNKNOWN`；**仅收到/引用知识不等于改善行为**。

**禁止类推**：「正确架构无需测试」不是从一次 23-test vs 46-test 的对照可以推出的规则。优秀的不变量设计可以令某些错误形态更难到达，但依然要用能反驳实现与环境假设的测试独立检验。

## 5. 来源与停止条件

- [NIST ACTS / Combinatorial Testing（机构入口）](https://csrc.nist.gov/projects/automated-combinatorial-testing-for-software)：参数交互测试与覆盖方法，不能代替选定高严重度时序不变量。
- [IEEE 1012/ISO 29119 在本知识库的来源适用目录](../external-source-catalog.md)：对专业 V&V 术语和测试活动提供来源候选，不代表全文复用/自动 conformance。
- [R2 逐项分析报告](https://github.com/kaicreator-mm/tally/issues/534#issuecomment-7070095204)：实验现象来源，尚非独立 QA 授权的通用事实。

**STOP**：Oracle Expected 来源不合法、未确认的业务规则、没有实际被发现/运行的测试、不完整跨 FR 覆盖或 Runner 与目标环境不等价时，不得宣称正式 PASS 或“所有风险已消除”。
