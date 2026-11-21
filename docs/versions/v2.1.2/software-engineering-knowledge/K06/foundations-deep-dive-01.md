# K06 · 学科深化 01：故障诊断作为可证伪的因果推理

> [系统课程](systematic-syllabus.md) · [首个失败](topics/FIRST_FAILURE.md) · [原因探针](topics/CAUSE_PROBE.md) · [诊断方法](engineering-methods.md)
>
> **ORIGINAL_EXPLANATORY_KNOWLEDGE_CANDIDATE / NOT_SOURCE_CLAIM_QUALIFIED / DIAGNOSTIC_RUN_NOT_PERFORMED**。深化 K06.F1.1/F1.2/F1.3、K06.F2.1/F2.2/F2.3 和 K06.F4.1/F4.2。

## 1. 分清症状、触发条件、直接失败与根因

一次 Build 失败可能在后续 Test、Artifact、Release 阶段产生大量衍生错误。应先定位本次执行 **第一个可观察的材料失败**，记录命令、退出码、时间、当前 Source SHA、运行环境和被读取的输入。首个错误未必就是根因：例如 `module not found` 可由漏安装、导出路径错误、缓存陈旧或错误工作区导致；需区分竞争解释。

不要把旧日志或另一条 pipeline 的错误强加给当前 HEAD，也不应对没有运行的下游阶段宣称 FAIL（应记录 `NOT_RUN`/blocked 的真实状态）。

## 2. 最小复现是控制变量，不是删除所有条件

复现单元需要保留导致现象的输入、版本、顺序、依赖、执行时环境。简化代码时若去掉竞争时序、网络副作用或旧数据快照，原问题可能消失，测试结论范围也改变。每次探针只应区分若干可竞争假设，而不是把所有环境参数都改一遍。

**可证伪假设示例**：
- H1：依赖未安装。预测从空依赖缓存按锁文件安装后可复现消失。
- H2：代码导出路径错误。预测 fresh install 后仍在相同 import path 失败。
- H3：CI Clone/工作区不同。预测在精确 Source SHA、相同工作目录与实际发布入口下出现/消失有差异。
一个观察可能只排除部分假设，不能因为某一次重跑 PASS 就直接认定 H1 已被证明。

## 3. 证据分级与小型诊断记录

| 记录 | 用途 | 常见误解 |
| --- | --- | --- |
| 原始 stderr/exit code | 直接失败观察 | 日志中的最终摘要就是首个原因 |
| 当前 SHA、环境、命令 | 复现身份 | 另一个 HEAD 绿也能证明本 HEAD |
| 最小稳定输入 | 触发条件与反例 | 简化后通过即根因已找到 |
| 竞争假设 + 区分性探针 | 因果候选排序 | 时间相关等于因果 |
| 修复前失败/修复后成功/回归 | 有界修复证据 | 回归全绿证明无限场景正确 |

生产环境的日志/Trace 读取需遵守最小权限、敏感数据脱敏与真实数据保密；K10 拥有可观测性总体合同，K06 选择当前问题的诊断探针，K03 负责合法 Task 内代码修复。

## 4. 故障类型改变验证方式

确定性语法错误可以用受影响编译器/类型校验直接区分；竞态与资源泄漏通常需要调度/生命周期反例；数据库迁移需要旧真实结构的隔离测试；外部 API 超时且 Effect 未知不能重复发不可逆请求，应先通过合法读回判断是否已生效，无法读回则记录 `UNKNOWN_EFFECT` 并寻求相应 Owner 决议。

“异常打印够详细”不等于错误处理正确；有用的日志应告诉操作者什么操作、哪个相关 ID、可重试性和下一步，同时避免记录 Token/密码/隐私内容。

## 5. 修复与学习的边界

先在合法写集内进行最小必要修复，重现原负例并检查受影响回归；若修复改变架构或业务政策，返回 K02/K01 有权来源。经验化记录“机制—触发条件—观察—失败假设—有效修复—不适用范围”，而不是通用指令“任何失败都重试三次”。无材料性新证据时停止无界调试循环。

**来源线索**：[A14 ISTQB](../external-source-catalog.md)、[A27 SWE-bench](../external-source-catalog.md)、[A28 SWE-agent](../external-source-catalog.md)、[A31 Google SRE](../external-source-catalog.md)、[A32 AWS Idempotent APIs](../external-source-catalog.md)。