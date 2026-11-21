# K08 · Agent Skills / 任务级操作技能

> **Tally v2.1.2 软件工程知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #529](https://github.com/kaicreator-mm/tally/issues/529) · [总索引](../README.md) · ADS 正式基线 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。当前只完成文档化的知识设计，尚无正式 Source→Claim 资格或真实 Package 执行。

## Tally Workflow/Business 实际与拟议消费者

## 5. Tally Workflow 与其他知识模块的组合

**消费方（拟议，不代表已接线）：** `PROPOSED_NOT_CONNECTED`；Tally Engineering Business 的 Task 验证、Integration/Delivery 与 Release-readiness 读者在出现变更代码、CI/Host 或制品时选用。输入：当前 PR/HEAD、项目 build/test 配置、实际 CI 和制品收据；产出：最小有效 CI/集成/安装验证计划及明确 `NOT_RUN` 缺口；原有 Validation、CI、Release Owner 分别作权威判断。

| 环节 | 既有事实输入 | 使用该知识获得的输出 | 不能推断 |
| --- | --- | --- | --- |
| 代码候选 | Task/PR exact HEAD | 已选 Build/CI 目标 | 提交不等于 build |
| Provider CI | 真实依赖和测试命令 | 分层通过/失败收据 | Provider 未跑即 NOT_RUN |
| 集成 | 公开合同与多个模块版本 | 组合/真实入口行为 | 单模块 PASS 不等于集成 |
| 制品与交付 | 构建包、目标环境、Release 规则 | 安装/启动/配置和发行收据 | 仅候选不自动 Release |

**与 K01–K11 的交叉关系：** K03 交付候选代码和编译行为，K05 提供真实测试策略，K06 分析 CI/安装失败根因，K07 负责多个 Agent/PR 的交接，K09 提供升级兼容性判断，K10 提供首次配置、日志/可靠性边界，K11 评估 CI/Review 总成本。

## 现有模块 Skill：触发、输入、结果与拒绝

## 6. Agent Skills：输入、输出、验证、失败路由

### SK-K08-01：Build & CI Planner

- **触发**：任务已有合法源候选和当前工具链
- **需要的事实输入**：manifest/lock/build scripts、changed paths、需要验证的合同
- **产生的工件或结论**：一次最小有效命令及平台矩阵、测试复用理由和执行输出要求
- **自我核对**：检查语法/测试确实选择目标源，真实 Provider/Host 条件可核实
- **拒绝/升级**：不得将 Agent 主机版本当作通用项目要求

### SK-K08-02：Integration & Artifact Verifier

- **触发**：多个模块合并或用户可运行制品需要证明
- **需要的事实输入**：当前集成 HEAD、真实调用入口、模块合同、制品摘要和目标平台
- **产生的工件或结论**：核心调用路径/安装/启动/失败的执行与回读收据
- **自我核对**：确认测试、运行和制品指向同一实际版本
- **拒绝/升级**：不能用 Mock 或源文件存在来说明用户功能生效

### SK-K08-03：Release Evidence Composer

- **触发**：有实际 Release Candidate 和资格材料
- **需要的事实输入**：目标版本、构建/测试/Host/Review/安装证据、残余 UNKNOWN
- **产生的工件或结论**：可审阅的 Release 证据和未完成条件列表
- **自我核对**：不将未知条件冒充满足，不代签发布
- **拒绝/升级**：不得自动发 Tag/Release 或把 PR Merge 当正式发行

## 单次 Agent Skill 提示词（直接文本，可按真实任务填写）

```text
OUTPUT_LABEL=TALLY_V212_K08_SCOPED_KNOWLEDGE_APPLICATION
VERSION_ISSUE=kaicreator-mm/tally#521
MODULE_ISSUE=kaicreator-mm/tally#529
STATUS=KNOWLEDGE_DESIGN_CANDIDATE_NOT_RUNTIME_AUTHORITY
ADS_SOURCE_SHA=933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c
TARGET_REPO=<actual repository>
TARGET_TASK=<actual issue/approved scope>
TARGET_HEAD=<actual exact source SHA>
ROLE=SCOPED_ENGINEERING_ANALYST

你是构建、CI、集成与交付分析 Agent。读取当前项目真实 manifest/lock、Woodpecker/Provider workflow、Task/PR Source SHA 和现有测试方案。先划分 clone/安装/编译/类型/测试/集成/打包/安装每个阶段，选择最小有效执行路线；记录 Runner 环境、配置来源、用例发现与制品身份。只有真实执行步骤才能写 PASS，缺少 Host/Release 证据标 NOT_RUN/BLOCKED；不能从依赖安装或 PR merge 推导正式 Release。

输出按 OBSERVED / SOURCE_BACKED / PROPOSED / UNKNOWN / NOT_RUN 分类。
仅完成当前这一次有界任务，结束时输出完整结果和剩余阻塞。
```

## 结果结构的语义要求（不是新 Wire Schema）

- 事实与数据来自当前用户、真实 Issue、现行文档/源码/日志，且能复现来源版本。
- 选择的知识有适用性理由和必要反例，不凭文档标题或技能名称选择。
- 明确真实权威保留于 Task/Claim/Review/Host/Release 原 Owner；通信/技能建议不代授权。
- 未执行的代码验证写 `NOT_RUN`，证据不足写 `UNKNOWN`，不要编造 PASS 或时间/成本。