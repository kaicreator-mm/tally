# K02 · Agent Skills / 操作技能与交接

> **v2.1.2 知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #523](https://github.com/kaicreator-mm/tally/issues/523) · [总索引](../README.md)。知识来源锁定 ADS v4.19.0 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；文档设计不代表已获知识资格或实际 Tally Runtime 消费。

## Tally Workflow 选择与实际消费者边界

## 5. Tally Workflow 消费契约

**消费阶段：** L2 Solution Design / DAG-A；使用 Tally 已有 Engineering Business、Task/Attempt 与 Agent/Host 事实源，新增的仅是领域知识与程序的调用关系。

| 环节 | 需要从上游获得 | 该模块实际提供 | 必须保留的判断 |
| --- | --- | --- | --- |
| 输入 | 被确认的 L1 要求与有效版本 | 驱动/质量属性/约束表 | L2 不能擅改 Product Scope |
| 研究 | 现有方案与关键未知 | 实证与可证伪实验结果 | 只有真正影响结构的未知才 Demo |
| 设计 | 决策依据与约束 | 模块/数据/接口/状态合同 | Demo 不能替代真实实现验收 |
| 冻结/移交 | 原有 L2 Authority | ADR/契约/Task seam/验证义务 | 设计文档完成不等于 L2 Freeze |

**跨层交接/复用：** K03 用设计合同进行详细实现，K04 把真实可并行 seam 放进 DAG-I，K05 由设计不变量导出集成/负例。K09/10 在兼容/可靠性材料存在时分别提供专项分析。

## 现有 Agent Skills 的输入输出与拒绝路径

## 6. Agent Skills 与任务输入输出

### SK-K02-01 · Architecture Decision Writer

- **何时调用**：多个实现方案会导致显著不同的长期成本或质量
- **最小输入**：准确 Product 需求、系统约束、已有方案及来源
- **输出结构**：ADR 的背景、备选方案、驱动矩阵、选定理由、风险/未知、验证和退出路线
- **自检**：结论可从驱动证据推导，未选方案有实质反例
- **禁止的捷径**：只列软件行业流行词或无数据编造 SLO/吞吐数字

### SK-K02-02 · Interface/State Designer

- **何时调用**：数据所有权、接口或异步状态是关键设计变量
- **最小输入**：当前系统权限/状态/事务与外部调用边界
- **输出结构**：责任图、接口 schema/示例、状态转移/重复与故障语义、合同测试种子
- **自检**：读写权、正常与异常的同一事实源一致
- **禁止的捷径**：在同一字段上赋予两个相互竞争的变更 Owner

### SK-K02-03 · Architecture Experiment Planner

- **何时调用**：至少一个可测技术未知会改变架构
- **最小输入**：反事实假设、候选方案、环境/成本和可用工具
- **输出结构**：最小实验、证伪阈值、输入版本、真实测量、明确外推限制
- **自检**：Demo 只说明它实际测试的事实
- **禁止的捷径**：把实验型 Mock PASS 写成部署生产资格

## 可直接作为单次 Agent 任务使用的提示词

> 注意：以下是**方法提示词**，不是写入许可、部署指令或平台强制格式；使用时必须先填写真实 Task/当前版本/输入来源。不得交给 Agent 无界持续执行。

```text
MODULE=K02
KNOWLEDGE_VERSION=TALLY_V2.1.2_SOURCE_CANDIDATE
ADS_EFFECTIVE_SOURCE=933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c
ROLE=SCOPED_ENGINEERING_ANALYST
TASK_SOURCE=<actual issue, current sha and approved scope>

你负责 L2 架构决策候选，不能改动已批准的 Product 范围。输入为已确认需求/验收、当前架构约束、实际运行环境。先列驱动和 UNKNOWN，比较两个真实可行方案（若现有方案足够直接复用则说明），明确模块/数据 Owner、API 成功/失败、状态/超时、兼容和可证伪验证。输出 ADR 与 Task seam。未验证的 Demo、成本或性能只能标范围内观察；统一 DAG-A 只是未采纳候选，不能新增任务权限。

输出时明确分隔 OBSERVED / SOURCE_BACKED / PROPOSED / UNKNOWN / NOT_RUN。
只要求本次问题所需内容；不自造 Product/L2/Claim/Review/Release 权威。
```

## 最小交接清单

- [ ] Task/Issue 与适用版本明确；信息来自可读的现行来源而不是聊天历史猜测。
- [ ] 具体问题、使用了哪些知识和适用条件明确，未加载整仓无关文档。
- [ ] 结果中的已观察事实、推断、建议、未知和测试未执行分开记录。
- [ ] 提交的 Artifact/测试/Review 指向准确 subject；没有合法执行环境时只给分析结果。
- [ ] 无界循环、未获权限的写入或虚构 PASS 被拒绝。