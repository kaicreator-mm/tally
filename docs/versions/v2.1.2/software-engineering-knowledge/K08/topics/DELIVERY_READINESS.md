# K08.DELIVERY_READINESS · 从 Build 到可安装、可配置、可用的真实产品

> [K08 知识库](../README.md) · [工程方法](../engineering-methods.md) · [Agent Skills](../agent-skills.md) · [案例](../casebook.md) · [原始依据](../sources.md) · [外部资料目录](../../external-source-catalog.md)。
>
> `KNOWLEDGE_TOPIC_CANDIDATE / NOT_SOURCE_CLAIM_QUALIFIED / PROPOSED_NOT_CONNECTED`。本文件是原创工程问题知识切片，不是 Tally 已生效的机器 Schema，也不是当前 Task/Review/Host/Release 执行许可。案例仅为合成教学示例。

## 1. 工程问题和适用前提

源码 PR 已合并，Build 成功，但没有可下载安装包，也没验证首次启动后的仓库设置。是否可以称版本发布完成？

**开始前必须确认：**
1. 实际 Product 发行目标/支持 OS/运行时
2. 当前 Build/Tests/Host/Review 来源及 exact SHA
3. Artifact digest/安装命令或 UI 启动/配置步骤

**学科知识与因果机制：** 交付对象既有 Source SHA 又有实际 Artifact/平台与用户操作。只有 Build 成功不能自动证明正确打包、签名、可安装、首次配置、应用权限以及真实用户入口。对一段纯库 API，NPM 包和真实下游消费者是交付验证；桌面应用则需对应 Installer/Bundle 和目标系统启动。Release 决策由原有 Product/Release Owner 授权，文档不会自动创建发布资格。

## 2. 可执行的有界判断过程

1. 确认真正的交付物种类：软件包、容器、可执行文件或桌面 Installer，并列用户需要的最小安装/初始化路径。
2. 检查制品内容与源版本一致且存在可访问的真实产物标识；目标平台安装/首次运行按当前产品真实要求验证。
3. 在用户可见入口验证本次改动真实生效，未完成配置或授权时不能开放危险业务操作。
4. 汇总已运行/未运行事项给合法 Release Owner；无有权决议时只报告 candidate readiness，不创建 release URL。

## 3. 场景分支、替代方案和不适用范围

| 条件 | 推荐的判断及范围 |
| --- | --- |
| 只有 PR merge/Build PASS | 仍是交付候选，不是 Release。 |
| 目标产品为 CLI/headless | 可通过参数/配置服务初始化，不强制 GUI。 |
| 正式 Installer 存在且安装/用户路径真实通过 | 可形成限定验收候选并等待 Release 决定。 |

## 4. 真实可证伪的教学情景

**受限教学输入及正确路线：** 桌面版本支持启动后配置仓库，Installer 安装后打开设置页，未完成设置前业务同步入口被禁止；配置后才能执行真实目标动作。

**失败路线/反模式：** 构建一个 zip 后既不校验完整性也不尝试安装，却在说明中宣布“正式安装包已发布”。

**最低可证伪 Oracle：** 模拟缺失 Installer 或首次设置失败，仍须给出 NOT_DONE 而不是 Release SUCCESS；文档示例不得虚构已发布下载链接。

## 5. 最小交接与明确的 STOP

**Agent 输出契约**：当前已观察事实/证据身份、选择的知识与前提、备选方案取舍、正反 Oracle、未得来源或环境、最小下一步。凡 Source Claim 未获得正式资格、Host/CI 未运行、Product/Task 未授权的内容，必须与可证实事实区分。

**跨模块交接**：K10 负责配置错误与可运维性模式，K05 负责用户行为 Oracle，K09 负责兼容迁移。

**停止规则**：需要越过合法写集、缺少有权业务决定、结果可能有不可逆副作用、外部状态不可确认或没有真实执行来源时，仅提出最小受控检查并保持 UNKNOWN/NOT_RUN，不通过重复调用或虚构状态推进。

## 6. 外部来源与可采纳边界

- [NASA Delivery Requirements Verification](https://swehb.nasa.gov/spaces/SWEHBVD/pages/102695529/SWE-194%2B-%2BDelivery%2BRequirements%2BVerification)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。
- [DORA Research](https://dora.dev/research/)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。
- [Microsoft Engineering Playbook](https://github.com/microsoft/code-with-engineering-playbook)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。

**已发布 ADS 约束**：`kaicreator-mm/ai-development-standard@933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。遇到差异应按当前有权规范与 Product/L2/Task/Host/Release 的真实作用域处理；未合并的三阶段 DAG 候选不自动变为正式工作图。
