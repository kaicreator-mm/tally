# K08 · 构建、集成与交付 — 软件工程领域知识库

> **Tally v2.1.2 软件工程知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #529](https://github.com/kaicreator-mm/tally/issues/529) · [总索引](../README.md) · ADS 正式基线 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。当前只完成文档化的知识设计，尚无正式 Source→Claim 资格或真实 Package 执行。

让源码真正通过项目工具链、测试与集成入口形成可交付制品，区分各步骤的实际状态与版本。

## 目录和何时使用

| 知识资源 | 为什么要读 |
| --- | --- |
| [disciplinary-foundations.md](disciplinary-foundations.md) | 学科理论、可应用的工程方法与误推断 |
| [external-practices.md](external-practices.md) | 企业/GitHub 真实工程参考与可证伪案例 |
| [domain-model.md](domain-model.md) | 区分工程概念、事实来源及容易混淆的语义 |
| [engineering-methods.md](engineering-methods.md) | 根据具体条件选择方法、方案与有界 Decision Procedure |
| [agent-skills.md](agent-skills.md) | 供 Tally 选择的 Agent Skill、单次可复制提示词和拒绝条件 |
| [casebook.md](casebook.md) | 正反案例、变体、反模式和实际工程决策案例 |
| [validation.md](validation.md) | 负例、知识适用性训练、未来真实消费者效用验收 |
| [sources.md](sources.md) | ADS 具体条款、有效版本、候选/规范和权限边界 |

## 三个优先问题

1. 依赖安装、构建、测试、打包分别在何处执行且各自结果是什么？
2. 哪个准确源码和 Runner 配置对应本次验证收据？
3. 是否已证明真实用户能安装/启动/配置并使用目标功能？

## 快速决策索引

| 触发 | 优先处理 | 期待输出 |
| --- | --- | --- |
| npm install 成功而 tsc 失败 | 定位 tsc 首个失败类型和当前源码 | BUILD_FAIL 与根因信息 |
| 多个 workflow 重复 clone 同一 commit | 按真实 Provider 工作区/缓存机制判断能否复用 | 网络/墙钟证据与最小优化 |
| Unit PASS 但运行入口未加载修改模块 | K05 指定 public-path 检查，K08 安排真实集成 | 适用 Integration/Host 收据 |
| 发布制品没有存在/摘要/安装验证 | 将 Release Candidate 与可下载正式包分开 | artifact UNKNOWN/NOT_FOUND |

**选用规则**：不要求每次调用全部 6 篇文档；依据当前任务所需决策选择最小知识切片。K09–K11 是条件性消费，K07/K08 在首批真实工程协作/交付需要时才选用。Tally 既有 Engineering Business/Host 事实源继续有效；本知识库为 `PROPOSED_NOT_CONNECTED`，不是新的知识服务/数据库或独立执行权限。