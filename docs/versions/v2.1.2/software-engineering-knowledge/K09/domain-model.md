# K09 · Domain Model / 领域概念及事实边界

> **Tally v2.1.2 软件工程知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #530](https://github.com/kaicreator-mm/tally/issues/530) · [总索引](../README.md) · ADS 正式基线 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。当前只完成文档化的知识设计，尚无正式 Source→Claim 资格或真实 Package 执行。

## 2. 概念、语义及常见混淆

| 概念 | 工程语义 | 容易出现的错误 |
| --- | --- | --- |
| Refactoring | 外部观察行为预期保持的内部结构改进 | 换了 API 语义却叫重构 |
| Compatibility Contract | 被真实消费者依赖的接口/版本/配置/数据格式和错误语义 | 只检查新客户端正常便认为兼容 |
| Breaking Change | 破坏已有承诺或消费者可观察行为的改变 | 没有声明就静默删除旧接口 |
| Migration | 现有数据/状态/配置逐步变成新格式和新行为的过程 | 仅定义最终 schema 而不考虑老数据 |
| Expand/Contract | 先扩展兼容范围、迁移消费者，再收紧旧合同的演进路线 | 直接发布不可回滚的高风险切换 |
| Technical Debt | 当前权衡导致未来真实维护/测试成本的负担 | 任何旧代码都称技术债 |
| Preserved Behavior Oracle | 已接受的既有行为及其对应测试/消费者证据 | 只把新需求的测试当完整回归 |
| Rollback/Compensation | 变更失败时返回可接受状态的方式，可能是补偿而非直接反转 | 假设所有数据库迁移都能简单逆转 |

此处是领域知识概念，而非要求另外建立一套消息数据库、Task 状态机或通用 Schema；必要时可映射到现有 Tally Engineering Business/Domain Data 结构。

## 必须区分的语义关系

### 关系 1

ChangeIntent → ExistingConsumer/CompatibilityWindow → Alternative Migration/Refactor → Retained Oracle → Verified Result。

### 关系 2

Refactor 的外部行为应保留；Feature 可以有经批准的新行为；Breaking Change 需要真实版本/范围决定。

### 关系 3

Fresh Install/Upgrade/Recovery 是不同主体；Migration 脚本存在不等于执行通过，down migration 并非通用恢复方法。

## 来源事实、方法建议与业务资格的区别

| 来源性质 | Agent 可以采用的方式 | 禁止推断 |
| --- | --- | --- |
| 现行 ADS v4.19 标准 | 在其适用范围内遵守发布的正式规则 | 本 KB 所有补充解释都拥有同等 MUST 权威 |
| Tally v2.1.2 设计方法 | 根据真实 Task 选择并与来源/案例验证 | 直接创建 Native Task/Claim/Review/Release |
| ADS #1100 / #1383 / Draft #1384 | 明确历史/候选身份，提供方法探索 | 当作已经发布的统一 DAG-P/A/I 强制流程 |
| 合成教学案例 | 说明在这些输入假设下应产生什么可观察结果 | 声称已有真实 Host/CI PASS |
| UNKNOWN | 明确缺失证据与下一项最小检查 | 通过猜测强行升格为已采纳 Rule |

## 领域实践中的三项核查

- 当前变化属于 Bug/Refactor/Feature/Migration/Deprecation 哪一种？
- 哪些旧用户、客户端或数据格式仍受支持？
- 数据迁移中断/升级回滚后系统保持什么不变量？

本项目暂缓独立建设代码库逆向分析、软件安全和供应链专项；原有适用的受控权限、敏感数据与必要验证仍须执行。