# K08 · Casebook / 贯穿案例及反例库

> **Tally v2.1.2 软件工程知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #529](https://github.com/kaicreator-mm/tally/issues/529) · [总索引](../README.md) · ADS 正式基线 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。当前只完成文档化的知识设计，尚无正式 Source→Claim 资格或真实 Package 执行。

## 已有贯穿案例与正反推断

## 7. 贯穿示例：正常路线、故意错误路线和适用差异

**情景（教学模拟）：** 一个 TypeScript 项目的 Woodpecker pipeline 含多个 workflow，每个都重新完整 clone；npm install 成功后某共享包 tsc 报错，最终用户还未收到构建好的桌面安装包。

**正确推进：**
1. 识别同一 pipeline 中能否复用一次 exact SHA checkout 及依赖闭包，在真实隔离条件不允许时才保留必要重复 clone。
2. 分清安装成功、tsc 失败、目标受影响测试未执行、桌面包未生成四种不同状态。
3. 依据编译失败文件/类型错误交 K06 局部根因修复，而不是更新所有锁文件。
4. 在新 HEAD 上执行所需类型检查、受影响测试和打包构建，记录真正的 Provider CI 结果。
5. 有权限和环境时运行安装包首次启动/配置与关键用户路径，否则清楚报 Host/Release 未完成。

**有效的验收证据：** 选定流程真实使用正确源 SHA、相关 TypeScript 构建和测试通过、可交付制品在目标平台运行且来源可追溯。

**错误实现/决策：** 流水线所有 clone 都执行成功就宣称优化完成；安装依赖绿灯直接合并 Release；没有发布包也报告正式 MVP 已可下载。

**条件变化如何影响选择：** 若每个 workflow 确实运行在相互隔离且不能安全共享工作区的远程 Host，重复 clone 的优化必须在提供者能力约束内实施，不能假设单个 Docker Volume 总是共享。

## 既有反模式和失效机制

## 8. 反例库与适用性边界

1. **Clone Explosion**：默认每个工作流都重新完整拉无关源码而无必要
2. **Install=Build**：成功安装包依赖就跳过实际类型/源码构建
3. **Mock=Installed**：测试环境模拟启动成功就宣称真实安装包可用
4. **Stale Release**：打包与审查 SHA 不同仍当作同一 Candidate
5. **CI Overreach**：无影响改动触发昂贵完整矩阵，阻碍交付
6. **Silent Failure**：只提供流水线 exit code 不暴露首个可操作错误

## 新增三种反事实场景

### C-K08-01 · 单仓 19 个 workflow clone

证明真实执行次数和复用可能性后优化，而非凭 YAML 行数断言完成。

**Agent 工作法：** 明确与当前 Task 相关的行为/接口/版本/失败条件，引用 [工程方法矩阵](engineering-methods.md) 选择最小充分 Procedure。若此场景改变了前提、角色或执行环境，就要重新核实有效来源；不将旧正例强行外推。

**预期反例：** 删除必要的来源、引入互相矛盾的状态/权限、或仅给 Mock 结果时，合格 Agent 应指出不能作出的结论并保留 UNKNOWN/NOT_RUN。

### C-K08-02 · 首次启动需配置

交互应用允许启动后配置，headless CLI 采用适合其形态的策略并给错误诊断。

**Agent 工作法：** 明确与当前 Task 相关的行为/接口/版本/失败条件，引用 [工程方法矩阵](engineering-methods.md) 选择最小充分 Procedure。若此场景改变了前提、角色或执行环境，就要重新核实有效来源；不将旧正例强行外推。

**预期反例：** 删除必要的来源、引入互相矛盾的状态/权限、或仅给 Mock 结果时，合格 Agent 应指出不能作出的结论并保留 UNKNOWN/NOT_RUN。

### C-K08-03 · 跨模块同 SHA 集成

确保模块/测试/构建/打包同 HEAD 并覆盖真实消费入口。

**Agent 工作法：** 明确与当前 Task 相关的行为/接口/版本/失败条件，引用 [工程方法矩阵](engineering-methods.md) 选择最小充分 Procedure。若此场景改变了前提、角色或执行环境，就要重新核实有效来源；不将旧正例强行外推。

**预期反例：** 删除必要的来源、引入互相矛盾的状态/权限、或仅给 Mock 结果时，合格 Agent 应指出不能作出的结论并保留 UNKNOWN/NOT_RUN。

## 使用教学案例的资格边界

本文所有新增 C-K08 案例均为 `SYNTHETIC_TEACHING_CASE`。真实生产资格需要来源明确的项目实例及正反/失败/当前版本证据；不能将教学案例当真实事故统计或 Agent 历史成功率。