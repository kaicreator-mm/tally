# K08 · Domain Model / 领域概念及事实边界

> **Tally v2.1.2 软件工程知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #529](https://github.com/kaicreator-mm/tally/issues/529) · [总索引](../README.md) · ADS 正式基线 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。当前只完成文档化的知识设计，尚无正式 Source→Claim 资格或真实 Package 执行。

## 2. 概念、语义及常见混淆

| 概念 | 工程语义 | 容易出现的错误 |
| --- | --- | --- |
| Build / Compile | 从当前源与工具链形成可运行/可验证的目标产物 | 安装依赖成功便声称构建成功 |
| Dependency/Lockfile | 项目声明的依赖来源/版本及工具链约束 | 用开发机上存在的 Node 版本强制所有项目升级 |
| CI Provider | 实际执行命令的 Woodpecker/GitHub 等环境及其日志/状态 | 本地测试 PASS 冒充 Provider CI SUCCESS |
| Selected Consumer Path | 生产/测试入口真正调用了变更后的公开能力 | 模块文件存在就认为功能已生效 |
| Integration | 多个模块与合同在同一真实版本下运行并通过必要验证 | 各分支单独成功就认为最终组合正确 |
| Artifact Identity | 制品摘要、源/测试 SHA、目标平台和配置版本 | 用可变 latest 标签证明不可变交付内容 |
| Install/Startup | 用户真实启动、安装、配置及失败行为 | 只看本机开发命令就声称首次运行可用 |
| Release Qualification | 发布证据和资格由现有所有者受控决定 | 合并 PR 就自动产生正式 Release |
| Pipeline Efficiency | 最小必要 clone/步骤/验证与缓存策略 | 每个 workflow 都重复完整 clone 而无真实需要 |

此处是领域知识概念，而非要求另外建立一套消息数据库、Task 状态机或通用 Schema；必要时可映射到现有 Tally Engineering Business/Domain Data 结构。

## 必须区分的语义关系

### 关系 1

SourceSHA + WorkflowConfigSHA + ToolchainProfile + Environment + Commands + CIResult 是执行来源；ArtifactDigest + TargetPlatform 才说明具体制品。

### 关系 2

Dependency Install != Compile/Build != Tests != Selected Consumer != Package != GitHub Release。

### 关系 3

K05 决定测试充分性和 Oracle，K08 负责执行/集成/制品来源与诊断，不新造第二个 CI/Release Gate。

## 来源事实、方法建议与业务资格的区别

| 来源性质 | Agent 可以采用的方式 | 禁止推断 |
| --- | --- | --- |
| 现行 ADS v4.19 标准 | 在其适用范围内遵守发布的正式规则 | 本 KB 所有补充解释都拥有同等 MUST 权威 |
| Tally v2.1.2 设计方法 | 根据真实 Task 选择并与来源/案例验证 | 直接创建 Native Task/Claim/Review/Release |
| ADS #1100 / #1383 / Draft #1384 | 明确历史/候选身份，提供方法探索 | 当作已经发布的统一 DAG-P/A/I 强制流程 |
| 合成教学案例 | 说明在这些输入假设下应产生什么可观察结果 | 声称已有真实 Host/CI PASS |
| UNKNOWN | 明确缺失证据与下一项最小检查 | 通过猜测强行升格为已采纳 Rule |

## 领域实践中的三项核查

- 依赖安装、构建、测试、打包分别在何处执行且各自结果是什么？
- 哪个准确源码和 Runner 配置对应本次验证收据？
- 是否已证明真实用户能安装/启动/配置并使用目标功能？

本项目暂缓独立建设代码库逆向分析、软件安全和供应链专项；原有适用的受控权限、敏感数据与必要验证仍须执行。