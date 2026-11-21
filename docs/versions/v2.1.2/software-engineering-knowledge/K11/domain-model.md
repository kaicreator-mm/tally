# K11 · Domain Model / 领域概念及事实边界

> **Tally v2.1.2 软件工程知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #532](https://github.com/kaicreator-mm/tally/issues/532) · [总索引](../README.md) · ADS 正式基线 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。当前只完成文档化的知识设计，尚无正式 Source→Claim 资格或真实 Package 执行。

## 2. 术语表、关系与边界误区

| 概念 | 可使用的工程定义 | 必须避免的推断 |
| --- | --- | --- |
| **Value / Outcome** | 交付后用户/业务/工程系统真实获得的结果 | 以 commit、PR、Issue 数量替代交付收益 |
| **Opportunity Cost** | 资源用在此任务时无法用于其他更重要任务的代价 | 只考虑直接 Token 成本 |
| **Total Engineering Cost** | 研究、Agent 运行、Review、CI、集成、故障修复与维护的总成本 | 把某个阶段最便宜当全生命周期最便宜 |
| **Marginal Value** | 额外研究/审查/工具给当前目标增加的实际收益 | 所有检查做得越多越好 |
| **Risk / Assurance Floor** | 失败后果和现行权限/测试最低要求 | “小改动”就省去高风险必须验证 |
| **Reversibility** | 错误决策修复/回滚的难易和后果 | 所有方案都能无代价回退 |
| **Evidence vs Estimate** | 实测资源/时间与基于假设的成本模型 | 把未经测量的估算写成 observed |
| **Work Shape Choice** | INLINE/BOUNDED_WORKLIST/EXPLICIT_DAG 以实际依赖和不确定性选择 | 所有版本一律用大型 DAG |
| **Outcome Evaluation** | 真实任务完成、缺陷、返工、延迟、资源的可比观察 | 单个 LLM answer 得分证明软件工程普遍改善 |

这些概念可以作为 Domain Concepts/Knowledge 与 Tally Workflow 的语义输入，但**不是**新增通用数据中心或自建 Task/Claim/Release 权限状态机的理由。

## 必须区分的语义关系

### 关系 1

Goal/Outcome → Alternatives → Observed Cost/Estimated Cost/Unknown → Risk/Assurance Floor → Decision → Measured Outcome。

### 关系 2

局部 Token 或 PR 数不是用户价值，独立审查/CI 必要性由实际风险和权限约束决定。

### 关系 3

直接 Agent 与 Tally+Knowledge 比较必须固定任务/环境/验收，评估遗漏、回归、返工、成本与负结果。

## 来源事实、方法建议与业务资格的区别

| 来源性质 | Agent 可以采用的方式 | 禁止推断 |
| --- | --- | --- |
| 现行 ADS v4.19 标准 | 在其适用范围内遵守发布的正式规则 | 本 KB 所有补充解释都拥有同等 MUST 权威 |
| Tally v2.1.2 设计方法 | 根据真实 Task 选择并与来源/案例验证 | 直接创建 Native Task/Claim/Review/Release |
| ADS #1100 / #1383 / Draft #1384 | 明确历史/候选身份，提供方法探索 | 当作已经发布的统一 DAG-P/A/I 强制流程 |
| 合成教学案例 | 说明在这些输入假设下应产生什么可观察结果 | 声称已有真实 Host/CI PASS |
| UNKNOWN | 明确缺失证据与下一项最小检查 | 通过猜测强行升格为已采纳 Rule |

## 领域实践中的三项核查

- 额外研究/审查/CI 是否影响关键决策或真实保护底线？
- 方案的总实施/集成/返工/维护成本是多少，哪些只是估算？
- Agent 知识辅助是否经真实开发任务公平对照证明有益？

本项目暂缓独立建设代码库逆向分析、软件安全和供应链专项；原有适用的受控权限、敏感数据与必要验证仍须执行。