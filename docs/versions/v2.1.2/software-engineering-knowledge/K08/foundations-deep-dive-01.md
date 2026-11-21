# K08 · 学科深化 01：可复现构建、依赖图与 CI 真实证明

> [系统课程](systematic-syllabus.md) · [流水线阶段](topics/PIPELINE_STAGES.md) · [Clone 效率](topics/CLONE_EFFICIENCY.md) · [真实交付](topics/DELIVERY_READINESS.md)
>
> **ORIGINAL_EXPLANATORY_KNOWLEDGE_CANDIDATE / NOT_SOURCE_CLAIM_QUALIFIED / CI_NOT_RUN**。深化 K08.F1.1/F1.2、K08.F2.1/F2.2/F2.3、K08.F3.1/F3.2/F3.3 和 K08.F4.1/F4.2。知识不修改现有 Woodpecker/Release 规则。

## 1. 不同成功阶段回答不同问题

`SourceSHA`、依赖锁版本、工具链/环境、构建命令、测试选择与制品摘要共同决定可追溯产物。依赖安装成功只表示依赖解析与获取满足该环境；语法/Typecheck 成功并不表示运行期行为正确；目标测试 PASS 需要真的发现和执行；生成 Artifact 不等于用户支持平台安装；Release 仍需有权 Owner 与真实发布记录。

必须区分 `NOT_RUN`、`FAILED`、`PASS_SCOPED` 的真实证据含义，不能用下游没执行的情况填 FAIL 或从上游 GREEN 推断完成。

## 2. 构建 DAG 和代码任务 DAG 不同

构建图的边是文件/模块/工具输出的**真实输入依赖**。缓存命中只有在影响输出的源文件、锁文件、工具版本、环境、生成脚本等关键输入都被精确匹配时才可复用；缓存命中不是任意 SourceSHA 都适用的无条件 PASS。构建 DAG 不应变成第二套 GitHub Task DAG 管理。

**实践反例**：仅对 `src/**` hash，忽略生成器配置或 schema；API 代码变了但缓存产物仍旧。即使 CI exit 0，实际消费者读到旧导出也不合格。需要针对缓存输入建立因果关系和最小真实消费者测试。

## 3. 一次流水线避免无关重复 clone

若一条 pipeline 下 19 个 workflow 各自完整 clone，网络/磁盘成本可能显著增加。若 Provider 允许共享可靠的 workspace 或一次 source checkout 供后续受控阶段使用，可降低重复 clone；但必须确认隔离/并发、不同 workflow 的容器生命周期、相同 Exact SHA、缓存读写安全性以及各任务所需源码范围。不能仅因“复用 workspace 更快”就让不同权限任务互相污染。

应先记录 `clone count`、每次实际 Git 网络操作和耗时、每个阶段真正需要的代码/依赖、CI 总墙钟时间，然后比较一条 pipeline 等价的选项。对无需源码的纯文档审核，没必要为了绿色 CI 徽章触发整套 Runtime 构建。

## 4. 可检验的阶段证据表

| 阶段 | 需要的证据 | 不能推出 |
| --- | --- | --- |
| Source checkout | repo/ref/精确 SHA/必要路径 | 已安装依赖 |
| Dependencies | 解析/锁/实际安装结果 | 编译成功 |
| Syntax/type/build | 工具链、命令、退出码和目标 | 运行时合同通过 |
| Selected test | suite discovery、实际断言、环境 | 未运行 E2E PASS |
| Integration | 公开消费者路径和依赖 | 发布安装包已存在 |
| Artifact | 平台/文件清单/完整性摘要 | GitHub Release 已创建 |
| Release | 受权 Release 记录、可取得产物 | 所有未来环境都可运行 |

K05 负责 Oracle 与测试充分性，K08 负责执行阶段与产物来源，K07 负责 Agent 交接/旧 SHA，K09 负责兼容升级。用户首次启动要求运行中配置时，应验证真正首次启动和合法持久化，而非以文档列环境变量冒充功能已支持。

## 5. 安全与止损

不要因重复拉取优化而绕过身份校验、共享不可信构建缓存或漏掉材料测试。优化前后在相同 commit 与足够同构的工作负载上比较执行次数/耗时/一致性；未测就输出 `NOT_MEASURED`。若项目当前存在 Required Checks 或独立特定 Provider，不能通过“本地貌似等价”删除它们。

**来源线索**：[A04 生命周期](../external-source-catalog.md)、[A20 Microsoft](../external-source-catalog.md)、[A23 GitLab](../external-source-catalog.md)、[A31 Google SRE](../external-source-catalog.md)、[A33 DORA](../external-source-catalog.md)、[A45 NASA 交付](../external-source-catalog.md)。