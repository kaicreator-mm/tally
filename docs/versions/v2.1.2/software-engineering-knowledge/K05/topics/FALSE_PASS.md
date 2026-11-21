# K05.FALSE_PASS · 测试假阳性取证：空测试、Mock 绿灯与过期来源

> [K05 知识库](../README.md) · [专业方法](../engineering-methods.md) · [Agent Skill](../agent-skills.md) · [反例与案例](../casebook.md) · [ADS 来源与外部学科目录](../sources.md) · [外部资料总索引](../../external-source-catalog.md)。
>
> `KNOWLEDGE_TOPIC_CANDIDATE / ORIGINAL_EXPLANATION / NOT_CLAIM_QUALIFIED / PROPOSED_NOT_CONNECTED`。Topic 是问题驱动的知识切片，不改变 Workflow、Task、Reviewer、Host 和 Release 权限；下面案例为合成教学条件，不是实际 Host PASS。

## 1. 问题与应用触发

CI 状态 SUCCESS，但变更代码没有进入真实构建，或者新测试没被 Runner 发现；怎样知道是否真的证明业务行为？

**适用前置输入：**
1. 当前候选和测试 HEAD、CI 配置/Runner/完整步骤
2. 受影响的公开入口、实际构建产物及测试发现日志
3. 已批准行为 Oracle 与必要真实依赖

**学科原理与为什么这样决策：** 一个 Pipeline 成功状态只说明这个 Pipeline 在其实际执行的命令和环境中成功。它可能没有发现目标测试、执行了旧构建、只覆盖 Mock 或对照了错误 Oracle。证据强度必须绑定源码 SHA、测试选择与发现、真实调用路径、被观察的业务结果以及 Provider/Host 作用范围。SWE-bench 的 Issue→Patch→Test 成绩不能自动等同整个应用的用户交付。

## 2. 实际决策程序（按现有权限执行）

1. 先核 CI 报告是否属于本次准确 HEAD，区分 install/build/test/集成阶段。
2. 确认目标测试是否被 Runner 发现；exit 0 但目标用例数 0 时不能采用为该目标的 PASS。
3. 检查修改模块是否在选定真实入口的依赖闭包中；若仅 Mock，明确测试模型未覆盖的实际状态/接口。
4. 据缺口只补一个最小会在错误实现下失败的契约/集成/公开路径负例；真实 Provider/Artifact 执行交 K08。

## 3. 按条件选择，而不是背诵一种答案

| 场景条件 | 当前应作的判断 |
| --- | --- |
| 目标测试已发现、Oracle 正确且实际被调用 | 可按具体测试层报告范围内 PASS。 |
| 0 tests / STALE HEAD / 只有 Mock | 只报告对应不足，不能声称目标行为有效。 |
| 外部 Host 未运行 | 按现行验证权限保留 NOT_RUN/BLOCKED。 |

## 4. 可复用的教学案例与负例

**合成正例：** 木啄鸟报告 `node --test` exit 0，但由于 glob 排除 `new-contract.test.mjs`，目标测试计数 0；Task 里的新 API 仍可能错误。

**失败/反模式：** 把 0 个目标测试执行的 pipeline SUCCESS 自动升级为 E2E PASS 和正式用户功能完成。

**可证伪的结果 Oracle：** 人工将对应测试文件重命名到 Runner 无法发现的位置，Agent 必须识别未覆盖范围并拒绝 `TARGET_BEHAVIOR_PASS`；不能仅引用 SUCCESS 徽章。

## 5. 最小输出、拒绝和跨模块交接

**Agent 最小输出**：当前事实/适用条件、被选知识及原因、候选/未选方案的权衡、具体正反 Oracle、缺失证据和 STOP、原始来源身份。不能取得真正 Product/Task/CI/Host 证据时要标 `UNKNOWN/NOT_RUN`，不能将本 Topic 引用的外部资料当作已发布 Tally 规则。

**交接关系：** K08 拥有 CI/Runner/Artifact 实际来源，K05 拥有测试充分性和 Oracle；不产生新的 Validation Gate 状态。

**停止条件**：业务政策无权确定、错误可能影响不可逆外部副作用、需要超出合法写集的实施、无法取得必要证据时，输出受影响范围及最小合法下一步，而不是调用无界 Agent 循环。

## 6. 来源、版本与知识资格

- [ADS Testing Standard](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/TESTING_STANDARD.md)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。
- [ISTQB CTFL v4](https://istqb.org/certifications/certified-tester-foundation-level-ctfl-v4-0/)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。
- [SWE-bench](https://github.com/SWE-bench/SWE-bench)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。

**ADS 约束基线**：`kaicreator-mm/ai-development-standard@933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；若与现行项目已生效规范冲突，应以具有权威的原规则为准并说明冲突。历史 ADS #1100、Draft #1384 的统一三阶段 DAG 观点不作为现行强制规则。
