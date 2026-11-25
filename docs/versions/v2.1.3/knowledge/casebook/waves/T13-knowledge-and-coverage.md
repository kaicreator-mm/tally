# T13 · 专业知识反向索引与覆盖台账

[#567](https://github.com/kaicreator-mm/tally/issues/567)；修正正文基线 41dad33035beab0eef1a1fffe3796d9fe71394db；[全量目录](T13-case-catalog.md)（实际读取 Blob c3053eee4096e6042709c9900f77806016ceca4f）。原 syllabus 165 项不变。125 个案例的保留关联均有一句具体理由，区分机制关联与条件启发；机制关联涉及 26 个原概念，仅条件启发 6 个，其余 133 个无当前案例标引。不是知识库完整率或评价分数。106 个来源登记及 49 项目不变；所有外部运行 NOT_RUN。T14 两份独立报告各自读取 34 个来源，样本有重叠但不假定相同；报告未宣称独立覆盖其余全部来源。在线背景另读不等于旧版本证据。Writer 本轮重审关系而不冒称重跑或全量独立源审。

## 1. 按原专业知识选择参考

原 syllabus 每个概念的真实标题及固定基线位置见下表；案例 ID 在 [目录](T13-case-catalog.md) 查精确内容、约束、来源及资格。标引是阅读关系而非语义等价；机制关联指实际观察中的特定机制，条件启发只提供类比/教学选择；验证意图不构成运行证据。删除非必要或语义不适切的旧关联不代表删除原理论。

### K01 · 机制关联 3/15；仅条件启发 0/15

| 原概念 | 案例 ID |
|---|---|
| [K01.F1.1 Need/Goal 与 Solution](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K01/systematic-syllabus.md#L13) | 未直接标引；保留原理论 |
| [K01.F1.2 Stakeholder 与利益冲突](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K01/systematic-syllabus.md#L21) | 未直接标引；保留原理论 |
| [K01.F1.3 业务边界与领域语言](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K01/systematic-syllabus.md#L29) | 未直接标引；保留原理论 |
| [K01.F2.1 访谈/观察/文档三角验证](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K01/systematic-syllabus.md#L40) | 未直接标引；保留原理论 |
| [K01.F2.2 事实/假设/未知分类](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K01/systematic-syllabus.md#L48) | 未直接标引；保留原理论 |
| [K01.F2.3 需求非功能影响](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K01/systematic-syllabus.md#L56) | 未直接标引；保留原理论 |
| [K01.F3.1 实体/值对象/聚合边界](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K01/systematic-syllabus.md#L67) | 机制关联：T03-C04 |
| [K01.F3.2 状态机/业务事件](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K01/systematic-syllabus.md#L75) | 机制关联：T03-C02、T03-C05、T07-C06 |
| [K01.F3.3 业务不变量与可推理约束](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K01/systematic-syllabus.md#L83) | 机制关联：AB-011、T07-C05；条件启发：DE-004 |
| [K01.F4.1 功能合同与边界用例](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K01/systematic-syllabus.md#L94) | 未直接标引；保留原理论 |
| [K01.F4.2 需求可追溯链](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K01/systematic-syllabus.md#L102) | 未直接标引；保留原理论 |
| [K01.F4.3 质量场景/优先级](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K01/systematic-syllabus.md#L110) | 未直接标引；保留原理论 |
| [K01.F5.1 同条件冲突判定](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K01/systematic-syllabus.md#L121) | 未直接标引；保留原理论 |
| [K01.F5.2 验收与反例](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K01/systematic-syllabus.md#L129) | 未直接标引；保留原理论 |
| [K01.F5.3 变更影响和冻结权威](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K01/systematic-syllabus.md#L137) | 未直接标引；保留原理论 |

### K02 · 机制关联 7/15；仅条件启发 0/15

| 原概念 | 案例 ID |
|---|---|
| [K02.F1.1 架构与实现的不同抽象层](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K02/systematic-syllabus.md#L13) | 未直接标引；保留原理论 |
| [K02.F1.2 Stakeholder / Concern / Viewpoint](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K02/systematic-syllabus.md#L21) | 未直接标引；保留原理论 |
| [K02.F1.3 架构决策记录 ADR](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K02/systematic-syllabus.md#L29) | 未直接标引；保留原理论 |
| [K02.F2.1 模块/组件边界](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K02/systematic-syllabus.md#L40) | 机制关联：AR-006、AB-007、T02-C04、T02-C08、T03-C01、T07-C08；条件启发：DE-011、T12-C01 |
| [K02.F2.2 接口契约与错误语义](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K02/systematic-syllabus.md#L48) | 机制关联：AB-004、AB-008、AB-009、T03-C07、T04-C07、T06-C05、T08-C06、T09-C10；条件启发：DE-001、DE-002、DE-007 |
| [K02.F2.3 依赖方向与扩展点](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K02/systematic-syllabus.md#L56) | 机制关联：AR-005、AR-010、AR-011、AB-012、AB-016、T02-C02、T03-C08、T08-C02、T09-C05；条件启发：DE-009、T12-C12 |
| [K02.F3.1 状态所有权与事务](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K02/systematic-syllabus.md#L67) | 机制关联：AR-002、AB-013、T02-C01、T02-C03、T03-C03、T05-C01、T09-C09；条件启发：T12-C02 |
| [K02.F3.2 一致性、事件和幂等](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K02/systematic-syllabus.md#L75) | 机制关联：T03-C06、T07-C04、T07-C09 |
| [K02.F3.3 持久化和数据模型](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K02/systematic-syllabus.md#L83) | 机制关联：T06-C02 |
| [K02.F4.1 质量属性场景](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K02/systematic-syllabus.md#L94) | 未直接标引；保留原理论 |
| [K02.F4.2 架构权衡与可证伪 Demo](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K02/systematic-syllabus.md#L102) | 未直接标引；保留原理论 |
| [K02.F4.3 威胁/失败模式的最小考虑](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K02/systematic-syllabus.md#L110) | 机制关联：T04-C03、T07-C02、T07-C03；条件启发：DE-005 |
| [K02.F5.1 版本/兼容边界](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K02/systematic-syllabus.md#L121) | 未直接标引；保留原理论 |
| [K02.F5.2 结构化可维护架构说明](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K02/systematic-syllabus.md#L129) | 未直接标引；保留原理论 |
| [K02.F5.3 架构适用性失效与回评](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K02/systematic-syllabus.md#L137) | 未直接标引；保留原理论 |

### K03 · 机制关联 12/15；仅条件启发 0/15

| 原概念 | 案例 ID |
|---|---|
| [K03.F1.1 复杂度/成本模型](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K03/systematic-syllabus.md#L13) | 机制关联：T02-C06 |
| [K03.F1.2 数据结构与索引](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K03/systematic-syllabus.md#L21) | 机制关联：T02-C07、T04-C04、T09-C02、T09-C06；条件启发：T12-C05、T12-C09 |
| [K03.F1.3 算法正确性与边界](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K03/systematic-syllabus.md#L29) | 机制关联：T09-C01、T09-C03 |
| [K03.F2.1 模块/接口/内聚](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K03/systematic-syllabus.md#L40) | 机制关联：T02-C05、T09-C04、T09-C07 |
| [K03.F2.2 类型系统与数据验证](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K03/systematic-syllabus.md#L48) | 机制关联：T05-C04、T05-C05、T06-C04、T07-C07、T11-C04；条件启发：T12-C07 |
| [K03.F2.3 可读性与可维护性](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K03/systematic-syllabus.md#L56) | 未直接标引；保留原理论 |
| [K03.F3.1 状态机/条件更新](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K03/systematic-syllabus.md#L67) | 机制关联：AR-003、AB-002、AB-003、T04-C02、T04-C09、T05-C06、T07-C01、T08-C04、T08-C05、T09-C08；条件启发：DE-010、T12-C04 |
| [K03.F3.2 幂等/重试/取消](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K03/systematic-syllabus.md#L75) | 机制关联：AR-012、AB-005、T05-C08、T06-C01 |
| [K03.F3.3 异步任务与背压](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K03/systematic-syllabus.md#L83) | 机制关联：AR-004、AB-010、AB-015、T04-C01、T04-C08、T08-C01 |
| [K03.F4.1 数据库和事务实现](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K03/systematic-syllabus.md#L94) | 机制关联：AR-001、AB-001、AB-014、T04-C05、T04-C06、T05-C07、T06-C03；条件启发：DE-003 |
| [K03.F4.2 文件/网络/进程资源](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K03/systematic-syllabus.md#L102) | 机制关联：AR-008、T05-C09、T05-C10、T06-C06、T08-C03；条件启发：T12-C06 |
| [K03.F4.3 依赖/生成代码边界](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K03/systematic-syllabus.md#L110) | 未直接标引；保留原理论 |
| [K03.F5.1 异常分类和可行动错误](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K03/systematic-syllabus.md#L121) | 机制关联：T05-C03；条件启发：T12-C08 |
| [K03.F5.2 日志与敏感性](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K03/systematic-syllabus.md#L129) | 机制关联：AB-006、T05-C02；条件启发：DE-008 |
| [K03.F5.3 真正的编译与受影响测试](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K03/systematic-syllabus.md#L137) | 未直接标引；保留原理论 |

### K04 · 机制关联 0/15；仅条件启发 2/15

| 原概念 | 案例 ID |
|---|---|
| [K04.F1.1 生命周期过程](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K04/systematic-syllabus.md#L13) | 未直接标引；保留原理论 |
| [K04.F1.2 迭代/增量与反馈](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K04/systematic-syllabus.md#L21) | 未直接标引；保留原理论 |
| [K04.F1.3 复杂度与风险驱动流程](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K04/systematic-syllabus.md#L29) | 条件启发：T12-C03 |
| [K04.F2.1 Work Unit/Task Outcome](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K04/systematic-syllabus.md#L40) | 未直接标引；保留原理论 |
| [K04.F2.2 问题分解与组合](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K04/systematic-syllabus.md#L48) | 未直接标引；保留原理论 |
| [K04.F2.3 验收与授权](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K04/systematic-syllabus.md#L56) | 未直接标引；保留原理论 |
| [K04.F3.1 真实因果依赖](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K04/systematic-syllabus.md#L67) | 条件启发：AR-009、T12-C03 |
| [K04.F3.2 共享写集/冲突控制](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K04/systematic-syllabus.md#L75) | 未直接标引；保留原理论 |
| [K04.F3.3 关键路径与资源约束](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K04/systematic-syllabus.md#L83) | 未直接标引；保留原理论 |
| [K04.F4.1 事实缺口/研究目标](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K04/systematic-syllabus.md#L94) | 未直接标引；保留原理论 |
| [K04.F4.2 变更影响与重规划](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K04/systematic-syllabus.md#L102) | 未直接标引；保留原理论 |
| [K04.F4.3 计划中的风险预算](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K04/systematic-syllabus.md#L110) | 未直接标引；保留原理论 |
| [K04.F5.1 实际进度的事实来源](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K04/systematic-syllabus.md#L121) | 未直接标引；保留原理论 |
| [K04.F5.2 复盘与实验对比](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K04/systematic-syllabus.md#L129) | 未直接标引；保留原理论 |
| [K04.F5.3 停止策略与交付首位](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K04/systematic-syllabus.md#L137) | 未直接标引；保留原理论 |

### K05 · 机制关联 0/15；仅条件启发 0/15

| 原概念 | 案例 ID |
|---|---|
| [K05.F1.1 测试与证明的范围](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K05/systematic-syllabus.md#L13) | 未直接标引；保留原理论 |
| [K05.F1.2 业务来源驱动 Oracle](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K05/systematic-syllabus.md#L21) | 未直接标引；保留原理论 |
| [K05.F1.3 验证与确认 V&V](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K05/systematic-syllabus.md#L29) | 未直接标引；保留原理论 |
| [K05.F2.1 等价类与边界值](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K05/systematic-syllabus.md#L40) | 未直接标引；保留原理论 |
| [K05.F2.2 决策表与状态转换](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K05/systematic-syllabus.md#L48) | 未直接标引；保留原理论 |
| [K05.F2.3 性质/组合/变异测试](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K05/systematic-syllabus.md#L56) | 未直接标引；保留原理论 |
| [K05.F3.1 Unit/Contract/Integration/E2E](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K05/systematic-syllabus.md#L67) | 未直接标引；保留原理论 |
| [K05.F3.2 依赖环境与隔离](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K05/systematic-syllabus.md#L75) | 未直接标引；保留原理论 |
| [K05.F3.3 Build/Package 与平台](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K05/systematic-syllabus.md#L83) | 未直接标引；保留原理论 |
| [K05.F4.1 实际发现/执行](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K05/systematic-syllabus.md#L94) | 未直接标引；保留原理论 |
| [K05.F4.2 Snapshot/Golden 和过期证据](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K05/systematic-syllabus.md#L102) | 未直接标引；保留原理论 |
| [K05.F4.3 Flaky 测试与诊断](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K05/systematic-syllabus.md#L110) | 未直接标引；保留原理论 |
| [K05.F5.1 风险驱动优先级](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K05/systematic-syllabus.md#L121) | 未直接标引；保留原理论 |
| [K05.F5.2 覆盖与变异有效性](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K05/systematic-syllabus.md#L129) | 未直接标引；保留原理论 |
| [K05.F5.3 测试结果与权威](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K05/systematic-syllabus.md#L137) | 未直接标引；保留原理论 |

### K06 · 机制关联 0/15；仅条件启发 0/15

| 原概念 | 案例 ID |
|---|---|
| [K06.F1.1 现象与原因分离](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K06/systematic-syllabus.md#L13) | 未直接标引；保留原理论 |
| [K06.F1.2 第一失败点与失败传播](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K06/systematic-syllabus.md#L21) | 未直接标引；保留原理论 |
| [K06.F1.3 环境/版本同一性](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K06/systematic-syllabus.md#L29) | 未直接标引；保留原理论 |
| [K06.F2.1 最小复现](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K06/systematic-syllabus.md#L40) | 未直接标引；保留原理论 |
| [K06.F2.2 竞争假设区分](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K06/systematic-syllabus.md#L48) | 未直接标引；保留原理论 |
| [K06.F2.3 调试工具与观测](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K06/systematic-syllabus.md#L56) | 未直接标引；保留原理论 |
| [K06.F3.1 确定性 Bug 与基础设施缺陷](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K06/systematic-syllabus.md#L67) | 未直接标引；保留原理论 |
| [K06.F3.2 超时/未知副作用](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K06/systematic-syllabus.md#L75) | 未直接标引；保留原理论 |
| [K06.F3.3 跨模块/集成缺陷](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K06/systematic-syllabus.md#L83) | 未直接标引；保留原理论 |
| [K06.F4.1 最小合法改动](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K06/systematic-syllabus.md#L94) | 未直接标引；保留原理论 |
| [K06.F4.2 回归与反事实](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K06/systematic-syllabus.md#L102) | 未直接标引；保留原理论 |
| [K06.F4.3 补丁当前性与复核](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K06/systematic-syllabus.md#L110) | 未直接标引；保留原理论 |
| [K06.F5.1 临时缓解与永久修复](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K06/systematic-syllabus.md#L121) | 未直接标引；保留原理论 |
| [K06.F5.2 事故时序与影响](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K06/systematic-syllabus.md#L129) | 未直接标引；保留原理论 |
| [K06.F5.3 可复用经验候选](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K06/systematic-syllabus.md#L137) | 未直接标引；保留原理论 |

### K07 · 机制关联 0/15；仅条件启发 0/15

| 原概念 | 案例 ID |
|---|---|
| [K07.F1.1 协作目标与成果物](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K07/systematic-syllabus.md#L13) | 未直接标引；保留原理论 |
| [K07.F1.2 协调成本与瓶颈](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K07/systematic-syllabus.md#L21) | 未直接标引；保留原理论 |
| [K07.F1.3 人机责任边界](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K07/systematic-syllabus.md#L29) | 未直接标引；保留原理论 |
| [K07.F2.1 Planner/Builder/Reviewer/Validator](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K07/systematic-syllabus.md#L39) | 未直接标引；保留原理论 |
| [K07.F2.2 Session/Attempt/Source 绑定](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K07/systematic-syllabus.md#L47) | 未直接标引；保留原理论 |
| [K07.F2.3 受控执行和会话边界](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K07/systematic-syllabus.md#L55) | 未直接标引；保留原理论 |
| [K07.F3.1 消息 vs canonical work facts](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K07/systematic-syllabus.md#L65) | 未直接标引；保留原理论 |
| [K07.F3.2 超时/重复/迟到处理](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K07/systematic-syllabus.md#L73) | 未直接标引；保留原理论 |
| [K07.F3.3 上下文选择与精简](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K07/systematic-syllabus.md#L81) | 未直接标引；保留原理论 |
| [K07.F4.1 独立写集与 Shared Owner](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K07/systematic-syllabus.md#L91) | 未直接标引；保留原理论 |
| [K07.F4.2 稳定接口与受控合并](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K07/systematic-syllabus.md#L99) | 未直接标引；保留原理论 |
| [K07.F4.3 冲突/失败恢复](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K07/systematic-syllabus.md#L107) | 未直接标引；保留原理论 |
| [K07.F5.1 可证伪独立代码审查](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K07/systematic-syllabus.md#L117) | 未直接标引；保留原理论 |
| [K07.F5.2 评估与协作反馈](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K07/systematic-syllabus.md#L125) | 未直接标引；保留原理论 |
| [K07.F5.3 失误记录与可复用规则候选](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K07/systematic-syllabus.md#L133) | 未直接标引；保留原理论 |

### K08 · 机制关联 0/15；仅条件启发 1/15

| 原概念 | 案例 ID |
|---|---|
| [K08.F1.1 源码/配置/制品多重身份](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K08/systematic-syllabus.md#L13) | 未直接标引；保留原理论 |
| [K08.F1.2 变更控制与可复现构建](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K08/systematic-syllabus.md#L21) | 未直接标引；保留原理论 |
| [K08.F1.3 资源/生成代码维护](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K08/systematic-syllabus.md#L29) | 未直接标引；保留原理论 |
| [K08.F2.1 编译链和依赖图](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K08/systematic-syllabus.md#L39) | 未直接标引；保留原理论 |
| [K08.F2.2 依赖安装 vs Build](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K08/systematic-syllabus.md#L47) | 未直接标引；保留原理论 |
| [K08.F2.3 增量构建与缓存正确性](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K08/systematic-syllabus.md#L55) | 未直接标引；保留原理论 |
| [K08.F3.1 Provider 语义和工作区](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K08/systematic-syllabus.md#L65) | 未直接标引；保留原理论 |
| [K08.F3.2 最小且真实的测试顺序](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K08/systematic-syllabus.md#L73) | 未直接标引；保留原理论 |
| [K08.F3.3 执行阶段与证据分离](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K08/systematic-syllabus.md#L81) | 未直接标引；保留原理论 |
| [K08.F4.1 组件/公共入口集成](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K08/systematic-syllabus.md#L91) | 未直接标引；保留原理论 |
| [K08.F4.2 Artifact 与平台目标](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K08/systematic-syllabus.md#L99) | 未直接标引；保留原理论 |
| [K08.F4.3 首次运行/配置与权限](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K08/systematic-syllabus.md#L107) | 条件启发：DE-006 |
| [K08.F5.1 候选合并与 Release 权威](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K08/systematic-syllabus.md#L117) | 未直接标引；保留原理论 |
| [K08.F5.2 部署变更与回滚](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K08/systematic-syllabus.md#L125) | 未直接标引；保留原理论 |
| [K08.F5.3 交付与故障复盘](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K08/systematic-syllabus.md#L133) | 未直接标引；保留原理论 |

### K09 · 机制关联 3/15；仅条件启发 1/15

| 原概念 | 案例 ID |
|---|---|
| [K09.F1.1 修复/增强/重构/适配](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K09/systematic-syllabus.md#L13) | 机制关联：T11-C01 |
| [K09.F1.2 维护优先级与长期成本](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K09/systematic-syllabus.md#L21) | 未直接标引；保留原理论 |
| [K09.F1.3 系统生命期和退役](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K09/systematic-syllabus.md#L29) | 未直接标引；保留原理论 |
| [K09.F2.1 行为保留的重构 Oracle](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K09/systematic-syllabus.md#L39) | 条件启发：DE-012 |
| [K09.F2.2 逐步替换与 Strangler 模式](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K09/systematic-syllabus.md#L47) | 未直接标引；保留原理论 |
| [K09.F2.3 可测试性与依赖切割](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K09/systematic-syllabus.md#L55) | 未直接标引；保留原理论 |
| [K09.F3.1 API 兼容版本](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K09/systematic-syllabus.md#L65) | 机制关联：T11-C02、T11-C03、T11-C05、T11-C06；条件启发：T12-C11 |
| [K09.F3.2 演进式数据库设计](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K09/systematic-syllabus.md#L73) | 机制关联：T08-C07 |
| [K09.F3.3 配置和数据语义迁移](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K09/systematic-syllabus.md#L81) | 未直接标引；保留原理论 |
| [K09.F4.1 受支持消费者及契约](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K09/systematic-syllabus.md#L91) | 未直接标引；保留原理论 |
| [K09.F4.2 渐进部署与回滚](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K09/systematic-syllabus.md#L99) | 未直接标引；保留原理论 |
| [K09.F4.3 回归传播和证据复用](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K09/systematic-syllabus.md#L107) | 未直接标引；保留原理论 |
| [K09.F5.1 技术债成因与利息](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K09/systematic-syllabus.md#L117) | 未直接标引；保留原理论 |
| [K09.F5.2 替换与保留的经济评估](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K09/systematic-syllabus.md#L125) | 未直接标引；保留原理论 |
| [K09.F5.3 受证据驱动的停止条件](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K09/systematic-syllabus.md#L133) | 未直接标引；保留原理论 |

### K10 · 机制关联 1/15；仅条件启发 2/15

| 原概念 | 案例 ID |
|---|---|
| [K10.F1.1 质量特征与业务场景](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K10/systematic-syllabus.md#L13) | 未直接标引；保留原理论 |
| [K10.F1.2 故障/缺陷/失效区分](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K10/systematic-syllabus.md#L21) | 未直接标引；保留原理论 |
| [K10.F1.3 SLO/SLI 与错误预算](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K10/systematic-syllabus.md#L29) | 未直接标引；保留原理论 |
| [K10.F2.1 结构化诊断与关联 ID](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K10/systematic-syllabus.md#L39) | 未直接标引；保留原理论 |
| [K10.F2.2 Logs/Metrics/Traces](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K10/systematic-syllabus.md#L47) | 机制关联：AR-007 |
| [K10.F2.3 监控与告警噪声](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K10/systematic-syllabus.md#L55) | 未直接标引；保留原理论 |
| [K10.F3.1 Timeout/重试/退避](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K10/systematic-syllabus.md#L65) | 未直接标引；保留原理论 |
| [K10.F3.2 幂等与恢复一致性](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K10/systematic-syllabus.md#L73) | 条件启发：T12-C10 |
| [K10.F3.3 隔离/熔断/背压](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K10/systematic-syllabus.md#L81) | 未直接标引；保留原理论 |
| [K10.F4.1 用户指标与分层定位](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K10/systematic-syllabus.md#L91) | 未直接标引；保留原理论 |
| [K10.F4.2 火焰图/Profiling 和瓶颈](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K10/systematic-syllabus.md#L99) | 未直接标引；保留原理论 |
| [K10.F4.3 实验对照和误差](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K10/systematic-syllabus.md#L107) | 未直接标引；保留原理论 |
| [K10.F5.1 首次运行和配置形态](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K10/systematic-syllabus.md#L117) | 条件启发：DE-006 |
| [K10.F5.2 事故响应与恢复验证](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K10/systematic-syllabus.md#L125) | 未直接标引；保留原理论 |
| [K10.F5.3 运行质量经验回流](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K10/systematic-syllabus.md#L133) | 未直接标引；保留原理论 |

### K11 · 机制关联 0/15；仅条件启发 0/15

| 原概念 | 案例 ID |
|---|---|
| [K11.F1.1 目标、用户价值与效用](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K11/systematic-syllabus.md#L13) | 未直接标引；保留原理论 |
| [K11.F1.2 总成本与生命周期](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K11/systematic-syllabus.md#L21) | 未直接标引；保留原理论 |
| [K11.F1.3 机会成本与可逆性](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K11/systematic-syllabus.md#L29) | 未直接标引；保留原理论 |
| [K11.F2.1 估算不是精确承诺](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K11/systematic-syllabus.md#L39) | 未直接标引；保留原理论 |
| [K11.F2.2 复杂度与依赖成本](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K11/systematic-syllabus.md#L47) | 未直接标引；保留原理论 |
| [K11.F2.3 研究价值与停止规则](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K11/systematic-syllabus.md#L55) | 未直接标引；保留原理论 |
| [K11.F3.1 风险识别与后果强度](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K11/systematic-syllabus.md#L65) | 未直接标引；保留原理论 |
| [K11.F3.2 质量保证下限](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K11/systematic-syllabus.md#L73) | 未直接标引；保留原理论 |
| [K11.F3.3 风险缓解选项](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K11/systematic-syllabus.md#L81) | 未直接标引；保留原理论 |
| [K11.F4.1 信息需求驱动的度量](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K11/systematic-syllabus.md#L91) | 未直接标引；保留原理论 |
| [K11.F4.2 SPACE / DORA 的解释边界](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K11/systematic-syllabus.md#L99) | 未直接标引；保留原理论 |
| [K11.F4.3 实验和对照设计](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K11/systematic-syllabus.md#L107) | 未直接标引；保留原理论 |
| [K11.F5.1 最小必要流程](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K11/systematic-syllabus.md#L117) | 未直接标引；保留原理论 |
| [K11.F5.2 多 Agent 协作经济学](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K11/systematic-syllabus.md#L125) | 未直接标引；保留原理论 |
| [K11.F5.3 知识维护/经验化投资](https://github.com/kaicreator-mm/tally/blob/7d2192b04080385cd147de84aee30b99ef14797a/docs/versions/v2.1.2/software-engineering-knowledge/K11/systematic-syllabus.md#L133) | 未直接标引；保留原理论 |

## 2. 理论补深与真实案例的选择关系

| 专业主题 | 先读理论 | 代表性参考 |
|---|---|---|
| 边界、替换与表示 | [T01 A1–A3](T01-theory-ab.md) | T02-C02/C03/C07/C08、T09-C01/C02/C07/C09、T12-C01/C12 |
| 合同义务、证据、可证伪 | [T01 B1–B3](T01-theory-ab.md)、[R6 实际研究](../../evidence-boundary-case-r6.md) | T06-C05、T07-C02/C04、T09-C10、T12-C07/C10 |
| 资源生命周期和并发 | [T10 C1–C2](T10-theory-cd.md) | T05-C08/C09/C10、T09-C08、T11-C02、T12-C04/C06 |
| 恢复、有界工作与外部效果 | [T10 C3–C4](T10-theory-cd.md) | T04-C01/C03/C05/C06、AB-005、T06-C01/C03、T12-C03/C05/C10 |
| 兼容演进与成本 | [T10 D1–D2](T10-theory-cd.md) | T11-C01–06、T08-C07、T12-C11/C12 |

## 3. 覆盖/缺口台账（T00 → T13）

| 分区 | 已具备的实际材料 | 仍有实质缺口；处理 |
|---|---|---|
| T01/T10 专业理论 | 12 个机制补深，165 个原概念保留 | 非穷尽深文审核；缺少原始论文/标准进一步交叉核验，真实合同与工程经济性待具体任务观察 |
| T02 软件架构 | 六类、8 案例 | 跨平台移动、CAD 全几何链、实际游戏应用、完整编译器前后端，未来批次 |
| T03 业务系统 | 八例，Web/ERP/电商/协同/CMS | CRM、完整 SaaS 租户架构及部署、实际库存/财务事务，局部观察不冒称完整产品 |
| T04 AI/基础设施 | 九例，推理、状态图、检索、事务、共识运输、实时/设备 | 模型执行器、多卡、Agent 持久恢复、共识正确性及板级实时性 UNKNOWN |
| T05 共用功能 | 10 条、九族，多约束设计参考 | 部分第二方案仅为设计参考；跨语言异常/序列化/查询及真实性能待补 |
| T06 数据网络 | 六族、六案例 | HTTP Range/ETag、同步冲突、快照恢复、事务/查询实际运行未验证 |
| T07 业务功能 | 九例，身份、权限、租户、支付、库存、表单、报表、审计 | 完整租户隔离链、财务对账与合规、可靠投递、权限端到端未知；国际化与独立账户生命周期仍薄 |
| T08 平台功能 | 七族、七例 | 原生 GUI/移动组件、真实硬件、跨平台安装升级、插件完整安全/兼容待补 |
| T09 代码抽象 | 十例，有正反例 | Go/Python 集中；TypeScript/Java/C++/Rust 等更多语言与真实业务函数对照待补 |
| T11 历史 | 六项实际前后版本差异 | 五个项目，只有一例实际读取新测试源码；未执行；缺大规模业务模块迁移与实测维护证据 |
| T12 原创方案 | 十二组，每组多个选择与反转 | 教学假设，NOT_RUN，不等于实际产品设计验收 |
| 初始 AR/AB/DE | 40 条保持原 ID；六项 AR 的正文/目录统一为 L2，DE-012 演进教学归 L5 | AR 在线背景未固定版本证据；T14 已实际读取五份在线背景，仍未绑定历史版本；正文与目录同步修正，保留旧 commit 历史 |

## 4. 明确资格与下一批建设门槛

- 不补造全覆盖。125 条中 101 条真实来源参考（含六条真实前后版本观察），24 条原创，不是 125 套运行验证；106 个文件 Blob 来自 49 个项目。R6 的内部实际运行另行保留，不混计。
- 统一目录索引不代替原文。来源不具备足够上下文、原作者未说明动机、未实测或业务授权不明确时，不输出确定最优方案；必要时降粒度或标 UNKNOWN。
- 原 22 类软件候选中已有若干真实局部来源，但完整产品级、多语言/跨平台、业务案例深度与真实重构演进仍有差距。此表是首期剩余缺口，不新立持续工作任务。
- T14 修正后限定独立复核需核实语义正确、版本文件可读、定位范围、案例资格及目录标签，尤其 AR 背景、DE 的新增知识关联、历史变更与演进反例。没有独立接受前不合并。
