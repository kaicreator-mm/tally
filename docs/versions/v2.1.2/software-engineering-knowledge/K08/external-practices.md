# K08 · 构建、集成与交付：权威实践与可证伪案例

> [K08 知识库](README.md) · [v2.1.2 #521](https://github.com/kaicreator-mm/tally/issues/521) · [外部来源总目录](../external-source-catalog.md)。**EXTERNAL_DISCIPLINARY_SOURCE_CANDIDATE / NOT_SOURCE_QUALIFIED / NOT_CONNECTED**；本文件为独立原创工程知识与实际方法的总结，非受版权保护的国际标准/教材原文复刻，不提升现有 Task/Release 权威。

## 从公开实践提炼有效技术，而非复制别人的流程

这里的实践案例是独立构造的**教学输入**，用外部权威/成熟项目的思想解释 Agent 应如何作决策。公开 GitHub 项目的原始能力受技术栈、依赖、执行环境和许可约束；其 README 或 Repo Star 不是验证当前 Tally Task 的权威。相应实例与成熟度都要逐项核对。

### P-K08-01 · 一条流水线的状态分层

**合成任务输入：** 输入：clone 成功、npm install 成功、tsc 在 shared package 报 TS2506，后续 tests 未执行。

**逐步操作：**
1.  按真实执行先后记录各 step 和 source SHA。
2.  Install=PASS 与 Build=FAIL 分别记录。
3.  未执行 Test=NOT_RUN。
4.  K06 定位 Typescript 源问题而非盲目改 lock。
5.  仅在修复后重验所影响的 build/tests。

**正反 Oracle：** 正确：第一失败来源清楚、无虚假 E2E；错误：因 install 成功将整个 CI 标 SUCCESS。

**出处与限制：** [A20](../external-source-catalog.md)、[A33](../external-source-catalog.md)。该示例是合成教学材料，并非宣称这些出版方已在对应项目上执行。真正可验的 Task 判断必须由受批准的业务 Oracle 和真实测试来源确定。

### P-K08-02 · CI Clone 次数优化

**合成任务输入：** 输入：单一 PR 构建含 19 个 Workflow，均声明 clone step，运行时间过长。

**逐步操作：**
1.  检查 Provider 每个容器是否真实独立。
2.  实际日志统计 clone 网络次数/墙钟耗时。
3.  选共享工作区/镜像或有界缓存的方案。
4.  对准确 SHA、测试 fixture 和构建范围做回归。
5.  对比前后真实 CI 开销。

**正反 Oracle：** 正确：在真实 Provider 约束下减少网络/时间且保留正确构建；错误：看见 19 个 YAML 节点就直接宣称已经网络 clone 19 次并节省固定百分比。

**出处与限制：** [A23](../external-source-catalog.md)、[A33](../external-source-catalog.md)。该示例是合成教学材料，并非宣称这些出版方已在对应项目上执行。真正可验的 Task 判断必须由受批准的业务 Oracle 和真实测试来源确定。

### P-K08-03 · 真正可安装的版本交付

**合成任务输入：** 输入：桌面应用 TypeScript 包构建成功，但没有 Installer、首次设置路径或 Release 页面。

**逐步操作：**
1.  确定目标平台与产物版本。
2.  构建实际可分发 Installer/Bundle。
3.  按应用形态验证首次启动配置和受影响用户路径。
4.  关联 artifact digest/source SHA/CI/审查。
5.  交授权 Release Owner 创建发行记录或报告尚未授权。

**正反 Oracle：** 正确：完整已测制品才能声称本范围可用；错误：仅以 PR merge 或源 build 就宣称 Release 已发布。

**出处与限制：** [A04](../external-source-catalog.md)、[A20](../external-source-catalog.md)、[A33](../external-source-catalog.md)。该示例是合成教学材料，并非宣称这些出版方已在对应项目上执行。真正可验的 Task 判断必须由受批准的业务 Oracle 和真实测试来源确定。

## 相邻知识如何复用

这些外部实践用于增强既有 [engineering-methods.md](engineering-methods.md) 的 3 个 Decision Procedures、[agent-skills.md](agent-skills.md) 的合法 Agent 输入输出、[casebook.md](casebook.md) 的正反案例和 [validation.md](validation.md) 的准确 Oracle。只按实际问题选择选定方法，不把研究 Source 直接提升为已采纳 MUST，不为每个练习引入新的 CI/Host 或独立 Review。
