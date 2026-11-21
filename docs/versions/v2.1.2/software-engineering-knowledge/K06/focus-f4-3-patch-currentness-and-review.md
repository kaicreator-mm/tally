# K06.F4.3｜补丁当前性、诊断证据与增量复核

> [K06 课程](systematic-syllabus.md) · [第一卷：因果诊断](foundations-deep-dive-01.md) · [第二卷：时序证据](foundations-deep-dive-02.md)
>
> **R4 ORIGINAL_KNOWLEDGE_CANDIDATE / NOT_SOURCE_CLAIM_QUALIFIED / NO_PATCH_REVIEW_PERFORMED。** 深化 K06.F4.3。K06 负责判定修复是否解决指定故障与反例，K05 负责行为 Oracle，K08 负责实际运行/制品身份；本文不建立第二套 Issue、审核和 Release 权威。

## 1. 精确修复对象是函数：错误症状×源版本×环境

一个修复报告如果只写“已解决”，通常不够可证伪。应明确故障在何处被观察，重现需要什么输入、状态、环境、工具版本，哪一个实际提交包含更改，哪个当前测试覆盖该故障。

**证据三元组**示意：`<SourceSHA, Test/Oracle identity, Environment>`。当任一项变化，原测试结论仍是历史事实，但不能无条件晋升为新上下文的执行事实；新旧之间能否复用取决于改变是否影响该测试的输入闭包和所需授权。更严格的生产发布规则以实际仓库现行配置为准。

## 2. Diff 决定需要复核的材料范围，但不是唯一线索

对一个补丁至少比较原失败分支、修复路径、调用者、共享数据/生成物及被影响的失败模式。Git `merge-base` 可以定位两个历史的共同祖先；`git diff` 给出内容差异；它们都不提供自动“语义影响”证明：调用是动态的、生成物可能不在 diff、配置与 Provider 环境也能改变行为。

**受影响闭包**可由以下关系约束：被修改的函数/规则 → 使用该合同的真实消费者 → 对应行为 Oracle → 实际执行 Runner/环境 → 必要独立审核。未变化的课程章节/不受影响的历史测试可以复用；这不是任意一处改动都让整仓失效。

## 3. 问题复现、补丁修复与回归不是一个事实

| 事件 | 允许结论 | 不允许结论 |
| --- | --- | --- |
| 旧 HEAD H0 上的复现探针触发故障 | 明确旧症状可观察 | H1 已修复 |
| H1 源码包含补丁 | 有精确修复候选 | 当前业务 Oracle PASS |
| H1 对应正反测试真正执行 | 在已测环境/范围内有验证结果 | 所有目标平台、所有消费者均已验证 |
| Reviewer 审查 H1 diff | 对 H1 受影响范围的独立审查事实 | H2 新更改自动通过 |
| H2 仅是无关文案变化 | 可复用未受影响 H1 的证据，按当前需要做增量检查 | 旧审批必然满足 H2 的 GitHub 必需规则 |
| H2 修改共享状态契约 | 应重审受影响调用路径和负例 | 只看文件名不同便宣称无影响 |

**教学例子**：H0 的登录权限缺陷在 H1 被修复，Reviewer ACCEPTED；H2 为修复日志重复把认证中间件移到另一位置，重新出现绕过。即使 H2 的改动文件名是“logging”，也不能据此排除安全/权限影响。反过来，另一模块的非语义 README 更新一般不需要重跑认证 Host E2E。

## 4. 避免两种相反的错误：过度晋升与过度重验证

- **过度晋升**：复制绿灯、历史测试数字或审查结论，未记录当前 SourceSHA/输入差异就称全部完成。
- **过度重验**：每次任何文档、格式或互不相关代码改变都要求从零做架构全审、完全 CI 和全部平台安装；会增加成本与人为疲劳，且不提升材料正确性。
- **正确的折中**：给出可检查的语义影响图，明确真正需要当前证据的范围；凡原审查的输入未变可复用为限定依据，真正改变的合同需要增量核验，受权发布条件仍按当时配置执行。

如果无权确定变化是否影响高风险业务操作，不应该以“只改三行”直接跳过；也不必凭不确定性无界增加复杂流程。可先做最小区分性探针再确定 Scope。

## 5. 两次补丁的可证伪反例练习

输入：H0 的异步重试存在重复付款；H1 引入业务 ID 幂等；H2 修改队列绑定使请求 ID 重新生成。旧 CI 对 H1 的所有单元测试通过，但 H2 测试未发现材料 E2E。要求判断：

1. H1 的哪些成功证据保留为历史事实？
2. H2 恢复出的违反不变量序列是什么？
3. 需要哪个现行外部 Effect/Oracle 才能判断幂等？
4. 能否仅根据 GitHub Review 文字或 H1 的 exit=0 给 H2 ACCEPT？
5. 当 Provider/环境不可访问时如何诚实记录 UNKNOWN/NOT_RUN？

合法输出必须逐一说清 SourceSHA、受影响项和 STOP 条件；不要伪造 H2 Host 成功。

**公开来源线索**：[Git merge-base](https://git-scm.com/docs/git-merge-base)、[Git diff](https://git-scm.com/docs/git-diff)、[GitHub PR Review](https://docs.github.com/en/pull-requests/how-tos/review-pull-requests/reviewing-proposed-changes-in-a-pull-request)、[GitHub Required Checks](https://docs.github.com/en/pull-requests/reference/status-checks)。这些说明工具语义，不自动代表当前仓库的已生效权限策略。
