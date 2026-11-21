# K08.PIPELINE_STAGES · 构建流水线各阶段的可观察结果与失败隔离

> [K08 知识库](../README.md) · [工程方法](../engineering-methods.md) · [Agent Skills](../agent-skills.md) · [案例](../casebook.md) · [原始依据](../sources.md) · [外部资料目录](../../external-source-catalog.md)。
>
> `KNOWLEDGE_TOPIC_CANDIDATE / NOT_SOURCE_CLAIM_QUALIFIED / PROPOSED_NOT_CONNECTED`。本文件是原创工程问题知识切片，不是 Tally 已生效的机器 Schema，也不是当前 Task/Review/Host/Release 执行许可。案例仅为合成教学示例。

## 1. 工程问题和适用前提

同一 CI 中 clone/install 成功、TypeScript 编译失败、测试未运行、Installer 未产生。应怎样报告当前交付状态？

**开始前必须确认：**
1. 真实 CI Workflow/Config SHA、Source SHA、Runner/OS/工具链
2. 按实际时间执行的命令、退出码和失败日志
3. 目标 Artifact/消费者/安装平台及当前 Release 要求

**学科知识与因果机制：** Source checkout、依赖安装、语法/类型、Build、测试发现和执行、公开入口集成、打包、安装、发布分别证明不同性质的结果。单个 CI 的最终绿色/红色徽章不足以替代阶段证据；NASA 的交付验证工程手册强调交付目标与验证关系，但属于任务/机构特定高保证参考，不能将其所有流程移植到普通项目。

## 2. 可执行的有界判断过程

1. 形成 Source/Config/Environment→Stage→Command→Exit/FirstError 的可追溯表。
2. 对已执行的步骤记录真实 PASS/FAIL，对没执行的记录 NOT_RUN 而非推断。
3. K06 诊断首个失败根因，K05 确定受影响测试和 Oracle，K08 选择当前 Provider 上必要的重新构建/验证。
4. 若目标是真正用户可安装软件，另要求真实制品摘要、目标 OS 启动/配置及合法 Release 决议；项目暂未发布时如实标尚无制品。

## 3. 场景分支、替代方案和不适用范围

| 条件 | 推荐的判断及范围 |
| --- | --- |
| 安装成功而 Build 失败 | Install PASS、Build FAIL、Tests NOT_RUN。 |
| 编译成功但 test discovery 为零 | Build PASS，目标行为验证不足。 |
| 打包未运行 | 不声称有正式 Installer，不能提供虚构下载地址。 |

## 4. 真实可证伪的教学情景

**受限教学输入及正确路线：** Node 项目 npm ci 顺利完成，`tsc` 报 TS2506，后续 `node --test` 与 Bundle 未被调度。当前摘要只能是安装成功、编译失败、测试/制品未完成。

**失败路线/反模式：** 把 `npm ci` 安装成功写成“CI 已通过”，并告诉用户 Release 安装包可下载。

**最低可证伪 Oracle：** 故意让 build step exit 2 且后续 step 0 次执行，结果模型必须保留各阶段分离，不能输出 Release PASS。

## 5. 最小交接与明确的 STOP

**Agent 输出契约**：当前已观察事实/证据身份、选择的知识与前提、备选方案取舍、正反 Oracle、未得来源或环境、最小下一步。凡 Source Claim 未获得正式资格、Host/CI 未运行、Product/Task 未授权的内容，必须与可证实事实区分。

**跨模块交接**：K05 Oracle/测试选取、K06 根因，K10 首次运行配置，K11 重复 CI 成本；Release Owner 仍是现行授权人。

**停止规则**：需要越过合法写集、缺少有权业务决定、结果可能有不可逆副作用、外部状态不可确认或没有真实执行来源时，仅提出最小受控检查并保持 UNKNOWN/NOT_RUN，不通过重复调用或虚构状态推进。

## 6. 外部来源与可采纳边界

- [Microsoft Engineering Playbook](https://github.com/microsoft/code-with-engineering-playbook)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。
- [GitLab Engineering Workflow](https://handbook.gitlab.com/handbook/engineering/workflow/)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。
- [NASA Delivery Requirements Verification](https://swehb.nasa.gov/spaces/SWEHBVD/pages/102695529/SWE-194%2B-%2BDelivery%2BRequirements%2BVerification)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。

**已发布 ADS 约束**：`kaicreator-mm/ai-development-standard@933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。遇到差异应按当前有权规范与 Product/L2/Task/Host/Release 的真实作用域处理；未合并的三阶段 DAG 候选不自动变为正式工作图。
