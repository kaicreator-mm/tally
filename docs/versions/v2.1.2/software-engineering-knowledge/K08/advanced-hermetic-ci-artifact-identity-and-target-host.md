# K08 进阶交付：一次源码获取、密封构建输入与目标平台验收

> [K08 课程](systematic-syllabus.md) · [交付证据第一卷](foundations-deep-dive-01.md) · [Hermetic Build 第二卷](foundations-deep-dive-02.md)
>
> **R6 ORIGINAL_CI_ENGINEERING_KNOWLEDGE / DOCUMENT_ONLY / NO_WOODPECKER_PIPELINE_CHANGED。** 映射 K08.F1.1、F1.2、F2.1、F2.3、F3.1、F3.2、F3.3、F4.2。本文只阐明可复现构建和减少重复 Checkout 的专业决策方法，不暗示任何当前 pipeline 已符合目标。

## 一、构建的正确性依赖完整输入而非 CI 绿色

对给定编译动作，输出受以下因素影响：源码 SHA、直接及传递依赖、代码生成 Schema、工具链版本、操作系统/架构、对执行有影响的选项/环境、构建脚本与被合法读取的文件。概念上：
`Artifact = Build(Source, DeclaredInputs, Toolchain, Platform, RelevantConfig)`。
输出 Hash 如果建立在缺失的依赖上，即使缓存返回相同键也可能得到错误旧产物。

Hermetic 并非简单“离线执行”或“固定 Docker image”：如果生成器读取当前时间、未声明的系统头文件、网络响应或宿主路径，缓存可能非确定；另一方面某些构建因正式签名/时间戳会有预期的非确定字段，需要在相应发布合同中明确处理。参见 [Bazel Hermeticity 官方来源 A62](../external-source-catalog.md)。

## 二、为什么“一次流水线 N 个 Workflow 每个 clone”会放大成本

教学场景：19 个 Workflow，每个都有自己的完整 `clone-exact-sha`，每次 clone 平均传输 300MiB。那么单次管线名义传输可达约 5.7GiB，尚未计 Git retry、checkout 和文件系统 IO。**这是演算，不是当前 Woodpecker 真实日志与带宽测量**；如果 VCS Cache/Provider 已在底层复用对象，实际网络成本不能由步骤数量直接推断。

可能的优化设计：
- 构建系统/CI Provider 支持**共享可信工作区**时：一次获取精确 SHA 并将只读源码/签名 Workspace Artifact 分发给下游多个 Workflow；
- Provider 每 Workflow 必须隔离时：先做一个按 SHA 生成且验证 digest 的 Source Artifact，后续 Workflow 消费同一制品而非独立重新 clone；
- 对文档无关的编译/测试，可在已证实不会改变合法输入的情况下用受控路径过滤或缓存；如果文档参与 Package、代码生成或许可证材料，则不能盲目忽略。

要求先弄清楚 Provider 的真正隔离/工作目录语义；不要仅靠 YAML 看见一次 Clone 就保证后续每个独立容器真的可读同一源码。**单次 fetch 的优化必须保持 Exact SHA、来源证据、workspace 隔离、权限、Artifact 完整性**，不能靠信任一个可修改的临时目录替代可验证的 checkout。

## 三、缓存输入闭包的最短反例

假定 Generator 基于 `schema.json` 输出 `generated/client.ts`，但缓存 key 只包含 `src/*.ts`：
1. H1 Schema v1，缓存产物 G1；
2. H2 修改 schema v2，但 src 代码未变；
3. 缓存仍命中并拿到 G1；
4. 部分已有单测通过，真正消费者要求 v2 而失败。

对症修复是把 Schema/Generator/工具链纳入动作输入闭包，必要时使生成结果直接作为受内容寻址的下一步输入。**反例验证**须改变材料性 Schema 并检查对应产物变化、测试发现/执行与 Consumer 合同；一个“清空缓存后绿了”的观察不等于已经证明缓存根因。

类似地，`--filter`、浅 clone、子模块下载/依赖安装策略都会影响是否能从“只获取一次”安全生成可验证制品。它们只在满足构建实际输入、submodule/LFS/生成物/许可文件完整性条件时采用。

## 四、测试发现、执行和发布制品必须不同报告

| 阶段 | 可报告的事实 | 错误升级 |
| --- | --- | --- |
| Source checkout | 精确 commit、可信来源与读取路径 | 只看到 branch 名就称 Exact SHA |
| Static/Type/Compile | 当前输入下编译/校验过程 PASS | 编译成功就称 Unit/Package PASS |
| Test Discovery/Execution | 实际发现清单、受影响测试真实结果 | Runner exit 0 却遗漏新 `*.spec` |
| Packaging/Signing | 有相应平台/架构/版本的完整制品及 digest | 只有 source tarball 就称正式安装包 |
| Install/First launch | 在要求的真实 Host 环境中可安装启动/配置 | Mock CLI 冒充 Windows GUI 安装成功 |
| Release | 已满足有权项目发布流程的认可 | CI 绿即自动 Release Authority |

具体执行时应先做相关语法/类型检查和受影响测试，避免错误进入昂贵的打包/E2E，但不机械要求 Node24（仅当项目明确需要 Node24 特性才应提升 Runtime 下限）；也不重跑已在等价环境真实验证的无材料变化步骤，除非有明确 Required Check/Provider/Host 约束。

## 五、错误缓解与工程经济

仅优化 Git clone 时间可能把成本转移到跨 Workflow 下载/解包，或创建可信共享目录的复杂性。必须看至少三个指标：`source acquisition wall-clock`、`network bytes/retries`、`end-to-end successful pipeline wall-clock`，以及缓存一致性/故障概率。只改一条 YAML 参数、未读 CI Provider 执行事实不能当完成优化。

**条件性备选**：如果 19 个 Workflow 实际各有隔离 clone，考虑共用以 SHA 绑定的 Source Artifact；如果当前真实 clone 开销微小而 Provider 对共享 Artifact 额外打包昂贵，未必值得更改。按真实失败和测量选方案，不为“一次 Clone”口号牺牲可靠性。

## 六、练习与限制

给出 H1/H2 Schema 版本、一次绿色 CI 和没有当前 Windows 安装包的记录。输出必须区分已核实的 `SHA/Source`、发现了哪些测试、缓存有效性、是否已有真实 Package、目标平台是否真正验收，以及最小必要的修正/复测动作。即使多个 Workflow 的 Test 都是 PASS，也不能据此推断 Package 可安装。

K06 负责对现象提出可区分假设；K05 负责 Oracle；K09 负责 Consumer 演进；K10 负责首启及运维事实。K08 仅拥有构建/制品/CI 证据链的学科解释。

**来源线索**：[A20 Microsoft Engineering Playbook](../external-source-catalog.md)、[A31 SRE Workbook](../external-source-catalog.md)、[A45 NASA Delivery Verification](../external-source-catalog.md)、[A59 Git merge-base/diff](../external-source-catalog.md)、[A62 Bazel Hermeticity](../external-source-catalog.md)。本章无真实 Woodpecker 执行，不构成修改配置的任务单。