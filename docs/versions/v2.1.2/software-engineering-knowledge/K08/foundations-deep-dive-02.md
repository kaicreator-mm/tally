# K08 · 系统学科深化 02：Hermetic Build、制品身份与缓存因果正确性

> [系统课程](systematic-syllabus.md) · [第一卷：构建/CI 执行事实](foundations-deep-dive-01.md) · [流水线阶段](topics/PIPELINE_STAGES.md)
>
> **R3 DOCUMENT_KNOWLEDGE_CANDIDATE / NOT_SOURCE_CLAIM_QUALIFIED / NO_BUILD_PERFORMED。** 深化 K08.F1.1–F1.3、K08.F2.1–F2.3、K08.F4.1–F4.3、K08.F5.1–F5.3，不修改既有 Woodpecker Pipeline 或额外强制 Gate。

## 一、可复现构建比“重复执行成功”要求更严格

一个项目运行两次 `npm run build` 都返回零，只能说明两次执行过程的命令结果。**Reproducible Build** 还要求在定义的源码、构建环境、构建指令相同时，指定制品的字节输出能够由参与者再现。它不是简单将锁文件提交、生成 hash 或获得 CI GREEN。

影响输出的输入除源码外还可能包括：编译器版本、系统库、依赖解析结果、生成器代码、环境变量、时间戳、文件排序、locale/timezone、外部网络下载与构建插件。哪些输入必须锁定，取决于真正的产物与目标平台。比对输出 hash 时需明确是安装包、可执行文件、映像还是源 map；某些带签名/构建时间的包装步骤不具字节确定性，需明确外围边界而非伪称“全产物可复现”。

## 二、缓存正确性需要内容依赖闭包

缓存键应代表**构建函数的有效输入**，而不是任意一个仓库 SHA 子串。内容寻址通常包括真实源、依赖/工具版本、影响生成的配置和目标平台。若共享缓存跨不同权限环境读写，还必须考虑缓存污染和不同构建变体意外交叉使用。

**可证伪例子**：Project A 修改数据库 schema，但生成的客户端缓存键只包含 `src/api/**`。第二次 CI 缓存命中、TypeScript 编译仍绿，发布产物却依赖旧 Schema；即使 SHA 是当前提交，该缓存仍不满足“基于全部材料性输入”的正确性。最小修复是修订真实缓存依赖与相应生成/消费者检查，而不是无界禁止所有缓存。

## 三、并行 CI 的一次 Checkout 是条件性优化

在一条 CI 中 19 个 workflow 分别 clone 会增加网络/IO 负担；然而一次 checkout 复用必须确认各 Workflow 的隔离与安全：是否真的共享 workspace、并行执行会不会相互修改输出、是否相同 Source SHA、插件是否自动再次 clone、缓存/工作树是否受污染。把 checkout 一次当成所有 CI 系统必须的规定同样是错误的。

性能研究输出应包括下载 Git 网络操作的次数、各 Clone 实际耗时、后续依赖安装/构建/测试耗时，以及所有失败/未运行阶段的身份。仅优化 Clone 后需要验证原始必要材料测试与制品没有退化；不能把 Provider 必要的 Required Check 当“多余重复”删掉。

## 四、制品与消费者组成完整的交付链

| 身份 | 最少要知道什么 | 不可代替什么 |
| --- | --- | --- |
| Source | commit SHA/必要源输入/许可证 | 已编译或可安装 |
| Toolchain | OS/架构/运行时/编译器版本 | 真实目标平台执行 |
| Dependencies | lock 与已解析的依赖 | 无运行时依赖缺失 |
| Build Artifact | 文件清单/目标平台/内容 hash | 用户已安装和启动 |
| Consumer | 公共入口/调用参数/实际环境 | Mock 的表面调用 |
| Release | 受权版本、发布地址、真实下载文件 | Build 成功就自动创建 Release |

部署中首次配置、升级与回滚属于实际可观察使用路径；如果项目要求启动后交互配置，不能仅交付环境变量说明就认定功能实现。K05 给测试充分性，K08 保留执行/制品事实，K09 给升级兼容边界。

## 五、专业练习：两次成功但二进制不同

同一源、同一可见命令构建两份包，内容 hash 不同。应按顺序比较真正环境与时间戳/文件次序/动态版本，再检查不同是否影响用户运行、构建可复现范围及修复成本。若无法确定生成产物范围，输出 NOT_ESTABLISHED 而非自动宣称供应链攻击或 Build FAIL。安全/供应链专业专项仍在本版本之外，但既有完整性义务不取消。

**来源线索**：[A20 Microsoft](../external-source-catalog.md)、[A23 GitLab](../external-source-catalog.md)、[A33 DORA](../external-source-catalog.md)、[A45 NASA Delivery](../external-source-catalog.md)、[A53 Reproducible Builds 定义](../external-source-catalog.md)。其专业定义不自动成为 ADS/Tally 所有 PR 的额外 Gate。