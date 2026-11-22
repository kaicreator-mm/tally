# K06.FIRST_FAILURE · 找首个失败阶段，而非围绕最后错误反复修复

> [K06 知识库](../README.md) · [专业方法](../engineering-methods.md) · [Agent Skill](../agent-skills.md) · [反例与案例](../casebook.md) · [ADS 来源与外部学科目录](../sources.md) · [外部资料总索引](../../external-source-catalog.md)。
>
> `KNOWLEDGE_TOPIC_CANDIDATE / ORIGINAL_EXPLANATION / NOT_CLAIM_QUALIFIED / PROPOSED_NOT_CONNECTED`。Topic 是问题驱动的知识切片，不改变 Workflow、Task、Reviewer、Host 和 Release 权限；下面案例为合成教学条件，不是实际 Host PASS。

## 1. 问题与应用触发

一个流水线曾经 npm ci 失败；这次 npm ci 成功、TypeScript build 失败。Agent 应如何确定当前根因而不是复用旧结论？

**适用前置输入：**
1. 真实流水线/命令的步骤与 Source/Config SHA
2. 第一条实际失败位置和日志上下文
3. 哪些前序步骤已 PASS、哪些后续步骤 NOT_RUN

**学科原理与为什么这样决策：** 故障诊断从实际可观察时序开始：主体的准确源码 SHA、配置和执行环境、最早失败命令/调用及原始错误。后来出现的“所有下游失败”和最终退出码通常只是症状传播。历史环境问题只有在被当前证据支持时才应保留为竞争假设。SRE 的事故事实时间线强调检测/缓解与验证分离。

## 2. 实际决策程序（按现有权限执行）

1. 按时间顺序形成 Stage→Status→Duration→FirstError，不因最后一步 exit 2 而假设最终命令是唯一原因。
2. 将已确实成功的上游步骤从当前最可能根因中有证据地排除，留下不确定环境因素时明确保留。
3. 生成至少两个可区分解释，并选择只涉及失败阶段的最小代码/环境探针。
4. 必要时由合法 Builder 作有界修复，回测原失败命令与受影响行为；独立 Host/Release 未运行则保持未验证。

## 3. 按条件选择，而不是背诵一种答案

| 场景条件 | 当前应作的判断 |
| --- | --- |
| npm ci 成功而 tsc 报自继承语义错误 | 优先源码符号/继承语义。 |
| 不同 Host 在相同 SHA 出现网络失败 | 评估 Provider 环境与依赖，不能忽略网络因素。 |
| 没有步骤日志 | 保留不足并请求最小证据，而非猜根因。 |

## 4. 可复用的教学案例与负例

**合成正例：** `packages/contracts/src/errors.ts:4` 声明 `class Error extends Error`，导致 TS2506/TS2339；修复应明确所需全局 Error 与本地命名冲突，而不是改全部依赖版本。

**失败/反模式：** 每次流水线最终 exit 2 就自动 rm lockfile、升级 npm 包、运行 CI 直到状态绿色，完全未处理实际编译错误。

**可证伪的结果 Oracle：** 具有相同源码 SHA、npm ci PASS、tsc FAIL 的输入必须优先输出 build 首个失败和具体源码/编译错误；若只输出依赖安装损坏，判为错误归因。

## 5. 最小输出、拒绝和跨模块交接

**Agent 最小输出**：当前事实/适用条件、被选知识及原因、候选/未选方案的权衡、具体正反 Oracle、缺失证据和 STOP、原始来源身份。不能取得真正 Product/Task/CI/Host 证据时要标 `UNKNOWN/NOT_RUN`，不能将本 Topic 引用的外部资料当作已发布 Tally 规则。

**交接关系：** K08 提供 CI 执行和克隆来源，K03 修复具体源码，K11 控制重复无新证据的尝试。

**停止条件**：业务政策无权确定、错误可能影响不可逆外部副作用、需要超出合法写集的实施、无法取得必要证据时，输出受影响范围及最小合法下一步，而不是调用无界 Agent 循环。

## 6. 来源、版本与知识资格

- [Google SRE Incident Management](https://sre.google/workbook/incident-response/)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。
- [ADS Incident Recovery](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/INCIDENT_RECOVERY_FEEDBACK_STANDARD.md)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。
- [SWE-agent](https://github.com/SWE-agent/SWE-agent)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。

**ADS 约束基线**：`kaicreator-mm/ai-development-standard@933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；若与现行项目已生效规范冲突，应以具有权威的原规则为准并说明冲突。历史 ADS #1100、Draft #1384 的统一三阶段 DAG 观点不作为现行强制规则。
