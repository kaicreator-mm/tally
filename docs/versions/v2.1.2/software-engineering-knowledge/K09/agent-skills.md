# K09 · Agent Skills / 任务级操作技能

> **Tally v2.1.2 软件工程知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #530](https://github.com/kaicreator-mm/tally/issues/530) · [总索引](../README.md) · ADS 正式基线 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。当前只完成文档化的知识设计，尚无正式 Source→Claim 资格或真实 Package 执行。

## Tally Workflow/Business 实际与拟议消费者

## 5. Tally Workflow 与其他知识模块的组合

**消费方（条件性拟议）：** `PROPOSED_NOT_CONNECTED`；Tally Engineering Business Maintenance/Change-intake 读者仅在兼容、数据迁移或重构需求真实发生时选用。输入：旧新版本、当前消费者合同、变更目标/数据；产出：保留行为清单、兼容决策和增量迁移/回归方案；L1/L2 与实际升级/Release 仍由原 Owner 授权。

| 环节 | 既有事实输入 | 使用该知识获得的输出 | 不能推断 |
| --- | --- | --- | --- |
| 维护受理 | 实际缺陷/新需求和版本记录 | 变更类别、范围和兼容性需求 | 不能直接改写历史 L1/L2 |
| 设计与任务规划 | 消费者/合同/迁移前提 | 有界迁移与重构计划 | 不是所有修复都重新冻结 |
| 实现验证 | 当前 Task/数据样本/旧行为 | 旧/新路径、失败/回退结果 | 新测试 PASS 不等于旧消费兼容 |
| 交付和经验 | 合格版本/真实回归与发布 | 知识案例及后续维护处置 | 不能自行生成 Release |

**与 K01–K11 的交叉关系：** K01/K02 负责真正的需求与结构变更，K03/K05 提供小步实现与保留行为测试，K06 区分修复与根因，K08 用于版本/制品兼容交付，K10 的运行恢复义务与 K11 的技术债经济决策按需求选用。

## 现有模块 Skill：触发、输入、结果与拒绝

## 6. Agent Skills：输入、输出、验证、失败路由

### SK-K09-01：Compatibility Impact Analyst

- **触发**：公开接口/数据/配置可能变化
- **需要的事实输入**：旧新合同、使用者、支持范围、已存在测试/版本
- **产生的工件或结论**：变更影响矩阵/破坏点/应保留行为与待授权决定
- **自我核对**：每一条“兼容”有真实消费者和测试覆盖
- **拒绝/升级**：不能只靠新版本编译证明兼容

### SK-K09-02：Refactoring & Migration Planner

- **触发**：结构改良或真实数据迁移有价值
- **需要的事实输入**：明确目标/数据事实、允许写集、旧用户基线
- **产生的工件或结论**：小步方案、验证/回退路径和失败停点
- **自我核对**：每一步能区分行为改变与纯结构整理
- **拒绝/升级**：不能凭口号强迫全部项目做 expand-contract

### SK-K09-03：Legacy Behaviour Verifier

- **触发**：修改老功能或关键公开合同
- **需要的事实输入**：被接受的旧版行为、当前新实现/测试
- **产生的工件或结论**：当前旧/新客户端/数据案例及残余 UNKNOWN
- **自我核对**：检测破坏后的旧调用和真实失败
- **拒绝/升级**：不应通过修改旧 Golden 掩盖非预期回归

## 单次 Agent Skill 提示词（直接文本，可按真实任务填写）

```text
OUTPUT_LABEL=TALLY_V212_K09_SCOPED_KNOWLEDGE_APPLICATION
VERSION_ISSUE=kaicreator-mm/tally#521
MODULE_ISSUE=kaicreator-mm/tally#530
STATUS=KNOWLEDGE_DESIGN_CANDIDATE_NOT_RUNTIME_AUTHORITY
ADS_SOURCE_SHA=933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c
TARGET_REPO=<actual repository>
TARGET_TASK=<actual issue/approved scope>
TARGET_HEAD=<actual exact source SHA>
ROLE=SCOPED_ENGINEERING_ANALYST

你是软件演进与兼容性分析 Agent。输入现行 Product/L2 版本、公开消费者、旧新 API/数据/配置合同、实际变更原因。区分 Refactor、Feature、Bug、Migration、Deprecation；列出允许变化与必须保留行为，选择最小兼容/迁移策略，给出旧新消费者正负例、部分失败/恢复计划。没有真实迁移不要造空流程，未获业务决策不得擅断支持范围；输出与现有 Owner 的受控交接。

输出按 OBSERVED / SOURCE_BACKED / PROPOSED / UNKNOWN / NOT_RUN 分类。
仅完成当前这一次有界任务，结束时输出完整结果和剩余阻塞。
```

## 结果结构的语义要求（不是新 Wire Schema）

- 事实与数据来自当前用户、真实 Issue、现行文档/源码/日志，且能复现来源版本。
- 选择的知识有适用性理由和必要反例，不凭文档标题或技能名称选择。
- 明确真实权威保留于 Task/Claim/Review/Host/Release 原 Owner；通信/技能建议不代授权。
- 未执行的代码验证写 `NOT_RUN`，证据不足写 `UNKNOWN`，不要编造 PASS 或时间/成本。