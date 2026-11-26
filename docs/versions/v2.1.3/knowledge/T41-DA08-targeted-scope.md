# T41 · DA-08 有限选题与增量范围

#654 / #563。冻结 5ddd7163f1c8e2393426af843a2d237cd1f1d8ee。原 [T23-R3 detail reassessment](T23-R3-detail-reassessment.md) Blob 2dbfba9e4722e0dcefd20d55e37c5e5658273cd2 L38 明列流式窗口、迟到、水位、存储模型尚未整体裁定；[residual plan](T23-R3-residual-depth-plan.md) 26418040c8c32afb77907895ba38a0e5036ffe6f L40 要求先确定实际工作负载。T41 选择官方公开文档描述的两个工作负载，而非构造所谓实际用户产品：Power BI/Fabric 订单行分析、库存快照；Beam Mobile Gaming LeaderBoard 每小时团队积分及离线迟到。实际读取原始资料范围见各章。可用来深化选择与推理，不能晋升为已核真实产品端到端 Case。

旧知识实际复用：K02 foundations-01 a43f8659a5edbc99b69d031d45dfece7d3d50603 L13–25 与 K03 foundations-01 03da0bf6a7cdc6f46ffd01dd6b5b26d2f9eff1ca L27–40 已有权威、持久任务和数据结构；T19 bae2f3113aadbd00f1072e548969e6905d7089cc L11–38 已有实际批刷新/迭代完整性与工作容量；T25b-2、T27b-1、T27c-3、T32b-2、T35-4/8 的实际冻结范围已读，详见章节。未穷尽其它旧专文，不称全库缺失。

| 有限选题 | 新增专业判断 | 保持原归属 |
|---|---|---|
| [R1 分析事实粒度与聚合](fine-grained/T41-R1-analytical-grain-aggregation-contracts.md) | 不同事实联结倍增、非可加指标、版本化维度、负载选型 | T27b-1 关系语义、T40-R2 表示、T35-8 对账 |
| [R2 流窗口及迟到](fine-grained/T41-R2-stream-event-time-watermark-consistency.md) | 事件时间窗口、水位/迟到/触发、pane 消费合同、恢复和关闭边界 | T25b-2 时间、T27c-3 可靠消息、T32b-2 血缘 |

同一指标如果需要历史完整日报，可使用 R1 的有限快照/聚合；确需连续暂定小时值及修订才用 R2。源事件和计算输出的身份、单位/指标版本、覆盖、权限必须同时满足。R2 的累积输出并非可加事实，正是 R1 的粒度/聚合条件在流式结果上的反例。两章纸面推导，无实际目标执行，不加产品/Package/DH/SDK/CI/Agent/LLM 实验。DA-08 仍 SPECIALIST U（不自动重判整体）；其余搜索、专用存储引擎、复杂会话、全量数据平台/真实性能专项未选，RE-04 未自动准入、RE-05 DEFERRED、RE-06 EXCLUDED。原 67/86、S51/D0/U5 及资格保持；正文候选未纳入已接受数量。
