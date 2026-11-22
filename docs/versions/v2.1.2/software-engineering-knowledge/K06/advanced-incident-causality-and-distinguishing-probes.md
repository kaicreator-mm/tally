# K06 进阶故障诊断：因果时序、可区分假设和反事实回归

> [K06 课程](systematic-syllabus.md) · [第一失败点/诊断假设第一卷](foundations-deep-dive-01.md) · [跨进程时序第二卷](foundations-deep-dive-02.md) · [补丁当前性 Focus](focus-f4-3-patch-currentness-and-review.md)
>
> **R6 DOCUMENTARY_PROFESSIONAL_CANDIDATE / NO_ACTUAL_INCIDENT_READ / NO_CI_LOG_EVALUATED。** 重点 K06.F1.1、F1.2、F1.3、F2.2、F3.1、F4.2、F5.2。这里的错误日志/时间戳/代码均为合成教学输入。K06 负责诊断假说与反驳，不代替 K10 的实际运行 SLI 或 K05 的正式 Oracle。

## 一、观察、第一失败与根因是不同的层级

Woodpecker 类流水线教学日志：
```text
10:00:01 clone-exact-sha: completed
10:00:04 generate-schema: skipped (cache hit)
10:00:05 compile: success
10:00:06 unit-test: 60 tests passed
10:00:07 contract-e2e: consumer rejected field version 3
10:00:08 workflow: failed
```
最后一行 `workflow failed` 是结果，不是根因；`contract-e2e` 是第一个可见失败断言，不保证缺陷就在 consumer；更早的 `generate-schema cache hit` 是候选的**上游致因事件**，还需要证据证明旧 schema 是否被缓存复用。

区分三种假设：
- H1：编译器实现生成的 v3 schema 确实错误；
- H2：构建缓存未将 schema 源内容/生成器工具版本纳入有效键，实际执行了旧编译产物；
- H3：consumer 合同已变，测试 Expected 未获最新授权。

三者都能导致相同的下游日志，不能见到一次 cache hit 就宣布 H2 实锤。

## 二、最小区分性探针的排序

一个有信息价值的诊断序列：

| 探针 | H1 预期 | H2 预期 | H3 预期 | 外部权限/代价 |
| --- | --- | --- | --- | --- |
| 读取当前 Exact SHA、Schema 输入 digest、Generator/Toolchain digest | 生成输入一致也可能失败 | 缓存键可能遗漏输入 | 可能与错误无关 | 静态只读，低成本 |
| 不复用缓存，在同一 SHA/工具链下生成并比较 Artifact hash | 错误继续可复现 | Artifact 可能改变，测试恢复 | 即使新产物仍可能不符合最新合同 | 需独立环境、源不可变 |
| 核对当前有权 Consumer 合同版本与测试 Expected 来源 | 仍可能合法预期 v3 | 取决于旧缓存 | 最新政策/接口可能与测试不一致 | 需要合同 Owner |
| 修改一处受影响字段并运行针对性 Oracle | 可能明确触发 | 可判断输入是否正确失效 | 需要先明确 Expected 是否有权 | 不应对生产 Host 写入 |

如果冷构建通过，说明缓存路径存在重要差异，但还不能证明具体是缓存**错误**：也可能不同环境变量、工具链或网络读入导致。下一步应比较**动作输入闭包**和执行环境，而不是粗暴删掉所有 cache 并认定已修复。

## 三、时序与因果不能仅凭墙钟排序

并行 Workflow 的日志可能有时钟偏差，真实顺序应由同一 Job/Attempt/Source SHA、上游/下游 Artifact digest 和已确认的调用-响应关系建立。A 事件时间 10:00:04 早于 B 的 10:00:05，并不足以证明 A 导致 B。所谓第一失败是当前可观测链路中的第一个失败，并不保证它是完整系统中最早产生错误状态的动作。

同时要区分：确定性源码 Bug、依赖/CI Provider/网络不稳定、真正的资源耗尽、未得证实的远端 Effect，以及拥有同样症状的上游业务来源不一致。仅靠重跑一次失败流水线 PASS，不能消除可重复的竞态错误。

## 四、反事实补丁与回归证据

针对 H2 的候选修复：把生成器版本、schema 输入、编译选项及影响输出的环境纳入缓存的有效输入闭包。需要针对同一 SHA/工具链构造至少两个反事实：
1. 改变一个**有材料性影响**的 Schema 输入，确认 Artifact key/output 对应变化并触发正确测试；
2. 改变一个真正无关的说明文件，确认可以合法复用对应缓存且没有遗漏必要验证。

如果修复后两项均通过，可限定支持“缓存输入有效性”的相关条件，而不是声明“所有 CI 永不失败”。目标平台安装和真实 Agent/Host 仍各自有独立 Oracle。

诊断因果与补丁影响要和 [K08 缓存机制](../K08/foundations-deep-dive-02.md)、[K05 Oracle](../K05/advanced-oracle-adequacy-and-test-selection.md) 交接；Exact SHA 的受影响审核由 [K06 既有 Focus](focus-f4-3-patch-currentness-and-review.md) 定义，不复制旧 Reviewer 的结论来覆盖新的错误区间。

## 五、事后经验如何正确抽象

不要把每一个偶发网络 502 固化为“总是重试三次”。可复用经验应包含 `symptom, competing hypotheses, differentiating probe, confirmed root cause, scope, counterexample, regression, expiry`。如果没有根因收据，标签应为 `UNRESOLVED_HYPOTHESIS`，不能把想象填入正式方法库。更不能把 AI 自生成的一句话“缓存老了”代替真实流水线日志。

**停止条件**：材料问题被最小探针区分且修复/验证已完成时停止增加无意义探针；如有远端 Effect 仍 UNKNOWN，则只暂停受影响的有危险操作，不要无限调查并行无关模块。

## 六、反例训练

“同样 SHA 的重复流水线第二次通过，所以第一次必为偶发网络故障”是错误推断：首次可能受并发资源影响、缓存热化、非确定性测试、缺少真正执行的用例或远端网络。应先对比测试发现清单、输入 hash、事件顺序/Attempt，且只宣称比较过的实际数据。诊断 **不是**为了挑一个最讨喜解释，而是让竞争假设在指定观测下产生不同的可反驳预测。

**参考线索**：[A14 ISTQB](../external-source-catalog.md)、[A27 SWE-bench](../external-source-catalog.md)、[A28 SWE-agent](../external-source-catalog.md)、[A31 Google SRE](../external-source-catalog.md)；本章引用专业思路并公开承认未实际读取当前任何 Woodpecker Logs。