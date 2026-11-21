# K08 进阶交付：来源证明、Release 权威和回滚的不可逆边界

> [系统课程](systematic-syllabus.md) · [R6 Hermetic Build 与目标平台](advanced-hermetic-ci-artifact-identity-and-target-host.md) · [K09 迁移](../K09/advanced-contract-evolution-migration-and-retirement.md)
>
> **R8 DOCUMENT_CANDIDATE / NO_RELEASE_ISSUED / NO_CI_REPLAY / NO_PROVENANCE_GENERATED。** 精确深化 **K08.F5.1 候选合并与 Release 权威**、**K08.F5.2 部署变更与回滚**两个原课程 ID。既有 K08.F1.1 等 Source/Artifact 课题作为链接上下文，不重复宣称新 A 资格。

## 一、把“生成工件”拆成五种不同的可证明事实

同一个 GitHub PR 可以同时拥有以下证据，但没有任何一项自动推导下一项：
1. **源码身份**：`repository, commitSHA, protected-ref/merge` 的真实 GitHub 收据；同名 branch 在不同时间可能指向别的 SHA；
2. **Build 来源**：哪个 Builder 在什么输入/工具/受权配置下创建了哪些 Digest 所标识的 Artifact；
3. **验证证据**：对应 Source 与 Artifact 的真实编译、Test Discovery、受影响执行、特定目标平台 Installer/E2E；
4. **Release 权威**：项目规定的 Reviewer/Owner、Required Check、签名和正式审批、发布事务已经真实完成；
5. **交付后的运行与可恢复性**：安装后能否首次启动配置、真实 Consumer 是否接入、出现失败时是否能够安全回滚。

反例：CI 的若干 Workflow 绿了，但 Windows 安装包没有生成；不能宣称 Windows Release READY。另一个反例：有了可下载安装包和 SHA256，却没有经过可信发布流程；“制品存在”不代表“有权发布”。

## 二、Provenance 必须有可验证的信任根

以官方 [SLSA Build Provenance v1.2 (A68)](../external-source-catalog.md) 为专业参考。概念上，Provenance 可以关联 `subject artifact + digest`、`builder`、`build definition`、`input/resolved dependencies`。但三个不同的声明需分开：

| 问题 | 需要什么 | 容易误报 |
| --- | --- | --- |
| 这个记录说 Artifact X 源于 H1 吗？ | 元数据字段和 digest 对照 | 仅有文本标签就称可信证明 |
| 这个记录可信吗？ | 可验证签名/身份、受信 Builder、适当来源策略和信任根 | 任意 Writer 自填一份 JSON 就当受信 attest |
| 这个 Artifact 能在目标平台安装/运行吗？ | 真正的二进制/Installer 与真实 OS/权限/首次配置验收 | 即使 SLSA provenance 完整也自动称 Host E2E PASS |

因此不存在“获得某个 SLSA 标记即可免除所有用户环境和有权 Release 检查”的结论。相反，项目若从未采纳此类供应链级别规范，可以用最小 SHA/digest + 可信 Builder/验证记录完成已批准的项目需求，不强行新增一项 SLSA 等级 Gate。

## 三、确定性 Artifact 和 Release 审批之间的隐藏漂移

教学例子：源码 H1 通过 CI，签名后 Installer 的文件 Digest 为 D1，但 GitHub Release 上传的是本机临时重新打包的 D2。
- 假设 D1 所指内容有真实测试/版本证据；
- D2 即使同名、同 semver，也可能因为依赖/签名/打包输入不同而改变；
- 目标平台安装验收若仅针对 D1，就不能当作 D2 的安装证据。

正确对照应包括下载实际发布物后复核 D2 的 SHA/digest、Release tag/ref 与已批准 Build 来源，必要时针对 D2 做真实平台验收。**只在证明二进制/环境材料未变或存在可信等价性合同的范围内复用旧测试**，不能仅凭同一文件名/同一版本号认定相同。

## 四、回滚代码不等于撤销数据与业务 Effect

合成迁移：
```text
H1: users.full_name -> 支持旧 Reader
H2: 拆成 first_name/last_name, 丢弃无法无损拆分的历史字段
部署 H2: 新数据写入; 远端付款 Effect 已成功
执行 rollback: 容器镜像回 H1
```
即使 H1 进程启动成功，仍可能无法理解 H2 的新数据；丢弃的姓名信息无法凭镜像回滚恢复，远端已发生的付款也不能简单删日志来撤销。需要分别评估代码回滚、持久化数据恢复、消费者双向读取、外部 Effect/补偿、回滚观察窗口与有权手工介入。

**负例**：`kubectl rollout undo` 执行返回零，即宣称财务和消费者数据都恢复。正确结论只能是指定部署工作负载某部分回到了旧版本；业务用户 Journey 和不可逆 Effect 仍需要独立验证。

## 五、为什么 CI 绿色不是唯一 Release Truth

项目有权合同可以要求 GitHub Required Checks、特定 Provider、签名、独立 Review、目标 OS 的实际安装或部分应用首次运行。但这些必须由当前项目真实规则决定，不会因学科教程而统一增加流程。另一方面如果本地等价真实校验已经通过且没有合法的 Provider 特定强制要求，无意义重复运行并不会形成更强的证据。

“一个 Pipeline 内尽量一次获取 Exact SHA 源码”是优化备选，不是允许所有隔离 Workflow 共享不受控可变 Workspace。若采用 SHA/Digest 绑定的 Source Artifact 分发，仍必须证明每个 Worker 使用的是真正被验证的输入闭包，不能把克隆次数减少直接当作更强的交付可靠性。

## 六、最小可证伪验收路线

候选需求：“正式安装包可在 Windows x64 首次启动并在 UI 内选择多个项目进行配置。”
- **编译已执行**：只说明对应源码/工具的语法/类型/构建；
- **模拟 Provider**：可以验证交互/状态保护的 Mock 范围，不能保证真实 GitHub 权限；
- **正式 Installer**：应检查真实下载 digest、目标 OS 安装与启动、配置/授权前不能越权发出任务；
- **Release**：在有权规则确实满足时才能从 Candidate 晋升，未做到就保持 `NOT_RELEASED`；
- **真实 Provider/Host**：若用户尚未请求真实外部 Effect，不得用 Mock 结果冒充有权真值。

K05 提供独立 Oracle；K07 审查身份/来源；K09 定义迁移/退役；K10 运行可观测性；K11 评价额外验证成本。本章不执行任何步骤，不生成新的发布制品。

**来源**：[A53 Reproducible Builds](../external-source-catalog.md)、[A56 GitHub Branch Protection](../external-source-catalog.md)、[A57 Node22 Runner](../external-source-catalog.md)、[A62 Bazel Hermeticity](../external-source-catalog.md)、[A68 SLSA Provenance v1.2](../external-source-catalog.md)。参考性质不等于当前 Tally 已取得特定 SLSA Track/Level。