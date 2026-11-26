# T45 · 动态工作流与联合资源约束有限收敛；T44 深度反馈

#662/#563，冻结 d7fd72a8bfbb8182ed7ad011c70701c617014a37。实际定向读取 T25a-1 的图和归纳、T40-R1 的组合及共同不变量、T40-R3 的分配与求解资格、T27c-1 的准入/生命周期、T35-4 审批版本、T35-5 的时段容量。保留既有专业 Owner 和核心定义，不重复普通 DAG 拓扑、匹配、预约或事务课程。

新增 [R1 动态工作流义务及版本](fine-grained/T45-R1-dynamic-workflow-obligations-and-revision.md)：区分历史、未来计划、节点/尝试/业务效果；以前后验证义务为中心分析新增/插入/跳过/替代、条件 join、局部图变更与并发 CAS、子图实例/出口义务、全实例有界动态生成。新增 [R2 联合资源调度及重规划](fine-grained/T45-R2-multiconstraint-scheduling-replanning.md)：有限整数槽、合法模式/时间选择、联合容量与前置、关键路径/共享资源下界、实际可行安排、硬期限及在线承诺、目标反转和独立 checker。二章为原创条件推导；不宣称实际工作流标准、SDK API、求解器实现或真实产品验证；不是全库缺失证明。单一固定过程、普通单资源检查仍使用已有知识，避免通用引擎强制抽象。

T44-RV-01 根据 [#661审核](https://github.com/kaicreator-mm/tally/pull/661#issuecomment-7198051774) 和 [合入收据](https://github.com/kaicreator-mm/tally/issues/660#issuecomment-7198110856)，T44-R1 §4 一段追加指定 CSSOM View WD L941–952 特殊规则及区分例，实际重新读取规范。原有两非退化矩形例和全部文字不改，候选修正待限定独审。T43-RV-01 已接受合入，不复审。

下一步 T46 选一个真实有权公开源码、冻结版本的完整代表性用户路径（而非产品实现），核必要实际源码/观察、局部设计到维护的组合；T47 关系/重复/冲突及索引收敛，T48 知识阶段最终盘点。三个剩余建议批次并非冻结数量承诺，避免无限追加。#653/#655 原 verdict 缺失仍独立门禁；本批不改共享索引/台账。60 正式接受章节，新增2候选，正式发现50/284、历史67/86、S51/D0/U5 不变；HC-04/DA-08 专项主行 U，RE-04 后续真实选题；RE-05 deferred/RE-06 excluded。#565 HOLD、#286 DEFERRED、A24/T4 NOT_RUN/HOLD。KNOWLEDGE_CONSTRUCTION，无产品/Package/DH/SDK/CI/Agent/LLM 或实际目标执行，无新 qualified Case/Claim/Rule。
