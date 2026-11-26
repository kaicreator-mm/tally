# T39-R1–R4 · 定向专业知识批次及范围台账

#646 / #563；基线 8b4594b1a27852c294a379e345bf6f59451c7545，KNOWLEDGE_CONSTRUCTION。T39 是从 T38 的有限知识见证返回实际知识建设，不是新增验证系统。四章作为一批候选；原46篇/265发现、67引用/86边、S51/D0/U5 均为旧已接受有限口径，不因写作自动更新。R5 仅限下面的局部剩余目标分流，未重盘全库。

| 增量 | 已有资料明确的未选目标 | 本批内容与候选Blob | 剩余边界 |
|---|---|---|---|
| R1 B5 | T35-2 L64、T35-4 L86：预约冲突 | [T35-5](fine-grained/T35-5-appointment-interval-capacity-composition.md) 707be88972f10c52b4786dfb881cba8584eef490 | 复杂排班优化、系列预约、跨资源实际分配另需合同 |
| R2 G5 | T35-3 L75：跨进程关闭 | [T35-6](fine-grained/T35-6-process-supervision-and-shutdown-composition.md) 4fd8d18b8821a7d4c396c4f132fc00000ca5fa01 | 实际平台创建期归属、权限、监督失效、回收时限与真实测试仍未知 |
| R3 G6 | T35-3 L75：异常及安全日志 | [T35-7](fine-grained/T35-7-diagnostic-logging-audit-composition.md) 20fe268593a718dbf65f21acc7d32ae04919fe00 | 行业审计和保留政策、真实日志系统另核；不重开RE-05 |
| R4 B6 | T35-4 L86：跨多个支付方/结算 | [T35-8](fine-grained/T35-8-multi-provider-settlement-reconciliation.md) 6f8b1eebc04ed8cc223c5cac66eec3757870490e | 真实渠道/汇率/会计、结算/对账运行仍未知 |

原章节的机制归属不变。新增章用于明确问题的阅读组合，不产生新的强制业务政策或新 qualified Case/Claim/Rule。所有目标执行 NOT_RUN。B5 PostgreSQL18范围与排斥约束官方资料已在本批补查（https://www.postgresql.org/docs/18/rangetypes.html 与 https://www.postgresql.org/docs/18/sql-createtable.html）；其它引用见各章。只核局部来源，不声称穷尽全库或全上游。R5：本批四个明确目标已有候选，未选的协议/UI平台、排班优化、实际完整代表性产品（RE-04）、ETL/分析工作负载（DA-08）等保持原专项/未决状态，后续按获权工程问题定向查；RE-05 deferred、RE-06 excluded 不重开。不得将整个 RE-02/03 的局部 S 当全域完成，亦不盲目增加章节或验证轮次。

## 审核与集成

一次独立审核覆盖四章内容正确性、来源版本、业务教学假设、反例、合法反转和复用边界。仅定向查源与独立推导，若发现重大命题疑点才使用最小见证；不得要求真实产品、Package、SDK、CI、Agent或LLM实验。#644/#645 K4 exact a0896ba62b484e64266d3c9aba9f97f3268840be 正式审核仍未恢复、未写回，合并门禁不得绕过；T39 只依赖已接受 main，不依赖 K4 候选。公共索引及 Output Ledger 的共享变更暂缓到 K4 门禁处置后，在最终集成前同步，避免两批并发覆盖。此文与 #646/#563 收据记录本批成果；不将其误称为已更新公共 ledger。#565 HOLD #286 DEFERRED A24/T4 NOT_RUN/HOLD。
