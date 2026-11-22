# K08 · 构建、配置管理、集成与交付：系统学科知识纲要

> [本模块问题索引](README.md) · [学科基础](disciplinary-foundations.md) · [方法与 DP](engineering-methods.md) · [Agent Skills](agent-skills.md) · [案例与反例](casebook.md) · [来源对照](sources.md) · [全局知识架构](../systematic-knowledge-architecture.md)。
>
> `SYSTEMATIC_KNOWLEDGE_CANDIDATE / NOT_SOURCE_CLAIM_QUALIFIED / NOT_CONNECTED`。本纲要包含**5 个知识群 / 15 个原创专业知识点**，用于长期系统化软件工程领域智能；不会仅因为整理过学科概念就自动取得 Task、Claim、Release、真实 Agent 效果资格。

## 一、知识本体、工程推理与能力归属

构建、配置管理、集成与交付 不仅是若干执行提示词。它必须提供专业原理与假设、对真实输入的条件性判断、可反证的错误做法、跨学科连接与明确资格。**当前唯一知识 Owner：源码/依赖/构建/CI/制品/环境事实、实际执行与交付收据的唯一知识拥有者**。下面的主题是完整课程的骨架，实际深度 Topic 是对必要问题的解释与实践投影；现有 DP/Skill/Case/Source 仍为可被引用的专项资料，不强制复制 15 次。

## 1. 版本控制与软件配置管理

### K08.F1.1 — 源码/配置/制品多重身份

**专业知识及适用机制**：Commit HEAD、Base、Lockfile、Config SHA、Runner、Target 和 Artifact digest 共同决定执行对象，不能只保存分支名。

**错误外推/反例**：旧 commit CI 绿色用于新 commit Release 证明。

**对 Agent 的可检验要求**：面对具体问题时指出真实输入、可以使用本知识的前置条件、未选择替代方案的理由和关键缺失项；缺必要事实时输出 UNKNOWN 或停止而不是自报 Rule/Host PASS。

### K08.F1.2 — 变更控制与可复现构建

**专业知识及适用机制**：按当前项目环境/依赖锁和生成材料的权威入口重建，记录可重复但不推断供应链专项已验收。

**错误外推/反例**：某开发机有隐式全局工具，别人构建失败却被宣称可复现。

**对 Agent 的可检验要求**：面对具体问题时指出真实输入、可以使用本知识的前置条件、未选择替代方案的理由和关键缺失项；缺必要事实时输出 UNKNOWN 或停止而不是自报 Rule/Host PASS。

### K08.F1.3 — 资源/生成代码维护

**专业知识及适用机制**：在构建图内区分 Canonical Source、生成物、打包资产及依赖闭包；改源码要触发真实生成的相应验证。

**错误外推/反例**：只手改生成文件而下次 CI 自动覆盖。

**对 Agent 的可检验要求**：面对具体问题时指出真实输入、可以使用本知识的前置条件、未选择替代方案的理由和关键缺失项；缺必要事实时输出 UNKNOWN 或停止而不是自报 Rule/Host PASS。

## 2. 构建系统与依赖

### K08.F2.1 — 编译链和依赖图

**专业知识及适用机制**：明确 package graph、target、runtime/TypeScript/JDK 等工具链版本及语法/类型/Build 分层；版本要求来自实际特性而非统一偏好。

**错误外推/反例**：未使用 Node24 特性的项目被强制升级到 Node24。

**对 Agent 的可检验要求**：面对具体问题时指出真实输入、可以使用本知识的前置条件、未选择替代方案的理由和关键缺失项；缺必要事实时输出 UNKNOWN 或停止而不是自报 Rule/Host PASS。

### K08.F2.2 — 依赖安装 vs Build

**专业知识及适用机制**：安装依赖成功不证明源码通过类型/编译或应用入口加载；记录真实首个失败步骤。

**错误外推/反例**：npm ci PASS 就给版本生成 Installer 链接。

**对 Agent 的可检验要求**：面对具体问题时指出真实输入、可以使用本知识的前置条件、未选择替代方案的理由和关键缺失项；缺必要事实时输出 UNKNOWN 或停止而不是自报 Rule/Host PASS。

### K08.F2.3 — 增量构建与缓存正确性

**专业知识及适用机制**：缓存键包含能影响结果的源码、依赖、编译选项/平台/目标信息；命中缓存也需证明无过期构建污染。

**错误外推/反例**：用旧平台缓存生成一个表面成功但二进制不兼容的包。

**对 Agent 的可检验要求**：面对具体问题时指出真实输入、可以使用本知识的前置条件、未选择替代方案的理由和关键缺失项；缺必要事实时输出 UNKNOWN 或停止而不是自报 Rule/Host PASS。

## 3. CI 执行与有效验证

### K08.F3.1 — Provider 语义和工作区

**专业知识及适用机制**：Woodpecker/GitHub/本地各自的 Workflow 共享规则不同；一次 clone 的潜力须用真实 Runner 证据判断。

**错误外推/反例**：在所有 Workflow YAML 中只留一个 clone step，却因容器隔离导致其它 Workflow 没代码。

**对 Agent 的可检验要求**：面对具体问题时指出真实输入、可以使用本知识的前置条件、未选择替代方案的理由和关键缺失项；缺必要事实时输出 UNKNOWN 或停止而不是自报 Rule/Host PASS。

### K08.F3.2 — 最小且真实的测试顺序

**专业知识及适用机制**：便宜错误在前，再类型/受影响测试/集成/制品；真实 Host/独立 Provider 要求仍需按原权限执行。

**错误外推/反例**：为 Markdown 文案变更连续跑完整无关矩阵。

**对 Agent 的可检验要求**：面对具体问题时指出真实输入、可以使用本知识的前置条件、未选择替代方案的理由和关键缺失项；缺必要事实时输出 UNKNOWN 或停止而不是自报 Rule/Host PASS。

### K08.F3.3 — 执行阶段与证据分离

**专业知识及适用机制**：Clone/Install/Build/Test/Package/Install/Release 的状态各自报告，Runner 测试发现、SHA 和首次失败保留。

**错误外推/反例**：test 未运行却报告整个 Pipeline 所有业务行为验证通过。

**对 Agent 的可检验要求**：面对具体问题时指出真实输入、可以使用本知识的前置条件、未选择替代方案的理由和关键缺失项；缺必要事实时输出 UNKNOWN 或停止而不是自报 Rule/Host PASS。

## 4. 集成与交付制品

### K08.F4.1 — 组件/公共入口集成

**专业知识及适用机制**：不同 Package 的合同通过不代表正式用户路径已经选中代码；对材料变更需真实调用到对的 Function。

**错误外推/反例**：模块导出却应用总是使用旧路由，仍报告端到端 PASS。

**对 Agent 的可检验要求**：面对具体问题时指出真实输入、可以使用本知识的前置条件、未选择替代方案的理由和关键缺失项；缺必要事实时输出 UNKNOWN 或停止而不是自报 Rule/Host PASS。

### K08.F4.2 — Artifact 与平台目标

**专业知识及适用机制**：区分库包、容器镜像、桌面安装包、Web bundle 和架构/OS；源 commit、内容 digest 与目标平台安装行为必须相符。

**错误外推/反例**：只有 zip 文件名而没有真实可运行安装包。

**对 Agent 的可检验要求**：面对具体问题时指出真实输入、可以使用本知识的前置条件、未选择替代方案的理由和关键缺失项；缺必要事实时输出 UNKNOWN 或停止而不是自报 Rule/Host PASS。

### K08.F4.3 — 首次运行/配置与权限

**专业知识及适用机制**：交互式应用可以启动后设置，CLI/headless 可通过参数/服务配置；配置前未授权业务动作需阻止，错误应可行动。

**错误外推/反例**：强制所有 CLI 弹 GUI，或桌面程序只能靠未知环境变量启动。

**对 Agent 的可检验要求**：面对具体问题时指出真实输入、可以使用本知识的前置条件、未选择替代方案的理由和关键缺失项；缺必要事实时输出 UNKNOWN 或停止而不是自报 Rule/Host PASS。

## 5. 版本交付与可靠发布

### K08.F5.1 — 候选合并与 Release 权威

**专业知识及适用机制**：PR Merge、制品创建、安装验证、Release Qualification、正式 GitHub Release 是独立事实，仍由现有合法 Owner 决策。

**错误外推/反例**：以 Branch Merge 宣布 Binary Release 已完成。

**对 Agent 的可检验要求**：面对具体问题时指出真实输入、可以使用本知识的前置条件、未选择替代方案的理由和关键缺失项；缺必要事实时输出 UNKNOWN 或停止而不是自报 Rule/Host PASS。

### K08.F5.2 — 部署变更与回滚

**专业知识及适用机制**：考虑灰度、版本窗口、数据迁移和安全回退；对不可逆副作用不能假设代码回滚能恢复业务数据。

**错误外推/反例**：应用降级后认为数据库已自动退回旧 Schema。

**对 Agent 的可检验要求**：面对具体问题时指出真实输入、可以使用本知识的前置条件、未选择替代方案的理由和关键缺失项；缺必要事实时输出 UNKNOWN 或停止而不是自报 Rule/Host PASS。

### K08.F5.3 — 交付与故障复盘

**专业知识及适用机制**：用真实 pipeline/Host 记录缺失资格和复现方式，只增量验证有实质变化的组件与要求。

**错误外推/反例**：为凑绿色徽章重复 CI 却不检查真实目标测试。

**对 Agent 的可检验要求**：面对具体问题时指出真实输入、可以使用本知识的前置条件、未选择替代方案的理由和关键缺失项；缺必要事实时输出 UNKNOWN 或停止而不是自报 Rule/Host PASS。

## 六、知识复用、专业反例与跨模块推理

**所有权**：源码/依赖/构建/CI/制品/环境事实、实际执行与交付收据的唯一知识拥有者。K05 对真实选中的测试 Oracle/覆盖负责，K06 解释具体失败，K09 处理发布时旧数据/兼容，K10 处理首次运行/错误恢复，Release 交现有 Release Owner 而不是 K09。

**专业教学情境**：Woodpecker 多 Workflow 频繁 clone。先看每个 Workflow 实际容器/工作区和 Git 网络日志，再判断可共享 checkout/缓存与否，保持 exact SHA 和测试 fixture 完整。

**必须识别的反模式**：只按 YAML 的 19 个 step 预估网络节省 80%，却没有对照运行及真实平台支持。

**最小推理闭包**：Goal 与真实事实 → 本课程序列中的 Concept/前提 → Applicable/Excluded/UNKNOWN 判定 → [Decision Procedure](engineering-methods.md)/[Skill](agent-skills.md)/[Case](casebook.md) → 可观察 Oracle 与 STOP → 未来只有合法 Agent/Host/CI 真实运行时才得到相应验证结果。单独的系统课程文章永远不授予生产执行权。

**学科与工程来源候选**：[A01](../external-source-catalog.md)、[A04](../external-source-catalog.md)、[A09](../external-source-catalog.md)、[A20](../external-source-catalog.md)、[A23](../external-source-catalog.md)、[A31](../external-source-catalog.md)、[A33](../external-source-catalog.md)、[A37](../external-source-catalog.md)、[A45](../external-source-catalog.md)。来源目录是研究入口，不代表对全部理论主张完成 Clause、版权和正式 Claim 资格；适用的 ADS 正式规范和 [sources.md](sources.md) 仍须检查实际权限与当前版本。

## 七、从完整知识库继续增长的原则

- **系统完整性**：现有 15 个知识点为 5 个专业知识群的概念基础，可继续增加算法、理论/实证研究、不同规模/环境反例和相邻领域关联，而不是等待生产任务才允许编写知识。
- **实际使用深度**：优先把多次遇到的误判、复杂取舍和关键质量问题深化成独立 Topic，既有 Topic 通过 [README](README.md) 选择，不强制为每点创建独立文件/Issue。
- **专业准确性**：当某项资料仅列于来源目录或不能取得精确原文时，保持 `SOURCE_CANDIDATE` 并标明不能外推；后续选定 Task 需要时只审实际用到的主张与负例。
- **服务边界**：Tally 的选定领域资产可被未来 DomainHarness selected Element Content/Function 消费，但编译/运行/授权/反馈仍归 [Harness 边界](../domain-harness-consumption.md) 及原有 GitHub/Host/Release Owner；不创设第二知识 Runtime。
