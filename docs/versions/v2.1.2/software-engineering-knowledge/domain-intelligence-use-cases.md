# 软件工程领域智能：有界查询、推理与反馈示例

> [系统知识架构](systematic-knowledge-architecture.md) · [跨模块关系](cross-module-knowledge-relations.md) · [Harness 边界](domain-harness-consumption.md)

**状态：LOGICAL_USE_CASES_ONLY / REAL_AGENT_NOT_RUN / DOMAIN_PACKAGE_NOT_CONNECTED**。这些是 Tally 未来软件工程 Domain Intelligence 服务应能够回答的专业问题及正反预期，不是已存在的 public SDK 方法或 CI/Host 收据。

## 1. 一次领域问题的统一选择原则

输入必须包含当前 Goal、业务/代码事实、来源/角色、版本/环境及未知。先识别完整知识库的专业模块、知识群和概念，再按实际风险选 Topic/DecisionProcedure/Rule/Skill/Case；所有条件性结论说明支持/反对来源和 STOP。确定性知识足以解决时不必启动 LLM；只有未来获权且确有材料性未知时使用受限模型生成候选，并且不能替代真实 Host/Task/Review。

## 2. 六种跨学科查询及预期结果

### Q01：订单取消规则冲突

输入：用户希望已发货订单一键取消；现行政策 A 拒绝，旧客服 FAQ B 声称付款即可取消，但 B 的 Actor 与版本未知。

知识选择：[K01 系统课程](K01/systematic-syllabus.md) F2.2/F3.2/F5.1；[规则冲突](K01/topics/RULE_CONFLICT.md) 与 [验收](K01/topics/ACCEPTANCE.md)，有权决议后才到 K02/K05。

预期：对同一 Actor/State/Event/Context 比较来源。无法裁定时保持受影响行为 UNKNOWN/HOLD，而不是根据训练语料猜“合理政策”。错误答案是将任何状态都设计成可取消并宣布业务验收已通过。

### Q02：持久作业与恢复性架构

输入：用户要批量导入不阻塞 UI、重启后恢复进度；可能重复投递，负载与部署成本未知。

知识选择：[K02 架构课程](K02/systematic-syllabus.md)、[状态 Owner](K02/topics/STATE_OWNER.md)、[K03 幂等](K03/topics/ASYNC_IDEMPOTENCY.md)、K10 可靠性与 K11 经济决策。

预期：说明内存队列与持久 Job 的恢复差异、提交点、幂等要求与成本未知。提出有界重启+重复投递实验；不能仅凭无重启的 Mock Benchmark 宣称恢复已验证，或强制所有项目部署 Kafka。

### Q03：算法复杂度与真实性能

输入：有人建议“所有 Array 改为 Map，系统至少快 3 倍”；当前数组规模仅几十条、无实测基线。

知识选择：[K03 基础算法](K03/systematic-syllabus.md)、[算法选型](K03/topics/ALGORITHM_CHOICE.md)、[K10 性能基线](K10/topics/PERFORMANCE_BASELINE.md) 与 [K11 工程取舍](K11/topics/TRADEOFF.md)。

预期：比较数据量、时间/空间复杂度、维护/缓存成本，明确无基线时 NOT_MEASURED；不能把理论阶数直接宣传为整机端到端提效。

### Q04：CI 绿但材料测试未执行

输入：CI exit 0，但新的合同测试因 Runner glob 被遗漏；组件已导出但实际 UI 仍调用旧接口。

知识选择：[K05 测试全域](K05/systematic-syllabus.md)、[False PASS](K05/topics/FALSE_PASS.md)、[真实集成](K05/topics/INTEGRATION.md) 和 [K08 阶段](K08/topics/PIPELINE_STAGES.md)。

预期：报告真正已运行的测试范围，未执行材料目标保持 NOT_RUN；选真实消费者调用路径和最小必要负例。错误答案是因 CI 绿色/Mock 被调用就报 E2E PASS；也不应无条件重跑无关全矩阵。

### Q05：多 Agent 旧 ACK 与 Host 副作用未知

输入：Worker A 基于旧 HEAD=a111 修改，当前 HEAD=b222，A 返回成功 ACK 且远端操作可能已发生。

知识选择：[K07 协作系统课程](K07/systematic-syllabus.md)、[旧消息](K07/topics/STALE_RESULT.md)、[K10 未知副作用](K10/topics/UNKNOWN_EFFECT.md) 和 K08 来源证据。

预期：ACK 不等于 GitHub canonical Task/Claim/Review/Release，当前 HEAD 不能由旧消息晋升 PASS；Effect 未知时先合法状态读回，不做危险重试。错误答案是覆盖 b222 或以相同 GitHub 登录名否定一切独立 Reviewer。

### Q06：不可逆迁移与经济性的底线

输入：数据库迁移仅三行代码，旧数据含非法状态值，支持滚动升级；本轮有人提出跳过迁移测试。

知识选择：[K09 维护课程](K09/systematic-syllabus.md)、[数据迁移](K09/topics/DATA_MIGRATION.md)、K01 业务规则、K05 Oracle、[K11 必要保证](K11/topics/ASSURANCE_FLOOR.md)。

预期：对无合法映射的旧值保持 UNKNOWN、说明兼容与中断恢复以及不可逆清理需有权决议。错误答案是 Fresh install 建表通过就宣布真实旧客户升级全部完成，或因 LOC 少取消适用的恢复验证。

## 3. 正式消费前的实际评估维度

| 维度 | 应检查 | 不可代替的事实 |
| --- | --- | --- |
| 专业覆盖 | 能选对学科群/概念/方法，发现遗漏 | 关键词命中不是专业理解 |
| 适用性与冲突 | 角色/状态/版本差异能改变结论，缺失时 ASK/UNKNOWN | 模型自信不是 Rule |
| 原理和推理 | 提供实际机制、权衡和负例 | 长答案不是质量证明 |
| 真实使用 | 合法 Task/Host/CI 当前收据与用户结果 | 教学 Case 不是 PASS |
| 来源与权利 | 原始链接/版次/Clause/版权和适用界限 | 目录累计来源数不是 Claim 资格 |
| 工程收益 | 同任务基线与辅助组的正确性/返工/耗时和成本 | Token 用量和 PR 数不能代替业务结果 |

以上六例可成为未来在 #286 合法消费者版本中进行有限评估的候选数据。当前只能证明**知识选择与判断预期已被文档化**，不能证明 Tally/DomainHarness 当前代码真正能返回这些结果。
