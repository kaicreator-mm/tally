// Package closedloop 实现 P2-02:v1.4 外层闭环编排的**领域骨架**(PRD §5 最小
// 产品闭环的 Execute → Verify → Repair → Accept/Escalate 段;PRD §17 Repair、
// §18 Accept/Escalate;ADR-0001 冻结语义;DECISION_LOCKS 7/8/23)。
//
// v1.3 的 codegen 内循环(internal/workflow,保留不动)与独立验证
// (internal/verification)是本包消费的两个**注入服务**:外层循环只看
// 「一次尝试产出了哪个候选快照」「独立验证给出了哪个 verdict」,内层迭代
// 预算对它完全不可见(ADR-0001 §1:inner retry ≠ outer Repair;本包任何
// 类型、任何端口都不出现 ExecutionPolicy.Retry,内层预算不外泄)。
//
// # 骨架范围(Kind: domain)
//
//   - phase.go:外层循环的类型化状态机(Pending → Executing → Verifying →
//     Repairing → Accepted/Escalated/Cancelled),确定性转移表是唯一权威,
//     没有任何通用 workflow DSL、没有配置驱动图、没有运行期自改规则;
//   - result.go:尝试谱系(Attempt,append-only)与终态结果载体(Result /
//     Basis / Escalation);
//   - ports.go:消费侧窄端口 —— Executor(不透明 codegen 服务)、Verifier
//     (独立验证服务)、Router(类型化失败路由缝,P2-03 提供实现)、
//     RepairBudget(外层修复预算缝,P2-07 提供实现)、LineageSink
//     (append-only 取代事实出口,P2-05/P2-08 提供适配);
//   - engine.go:确定性驱动器 Engine:按转移表推进,终态完整(每次 Run
//     恰好落入一个终态),取消是一等终态,unknown/无法裁决一律 fail-safe
//     升级(INV-V14-09:绝不静默选最弱保证)。
//
// # 结构性编码的 ADR-0001 不变量
//
//   - 每次外层 Repair = 新 CodegenRun + 新一次性 worktree + 新不可变
//     ChangeSet + reason=rework 的追加式取代事实 + 全新独立验证(§2):
//     前两者与最后一个由端口契约约束(适配器负责),ChangeSet 身份新鲜性
//     与取代事实由 Engine 机械强制(重复候选身份 → fail-safe 升级);
//   - 修复从冻结的前候选继续(§2,P2-05):修复尝试的 ExecuteRequest
//     携带最新未被取代候选的内容快照 RepairBase(仅候选内容,无任何
//     验收/保证材料 —— 契约 pin 逐字不变);适配器在全新一次性 worktree
//     内以其基线检出并确定性物化;前候选与其工作区绝不被改写;
//   - 已验证/已失败的候选绝不原地修改:候选只经 changeset.ChangeSet
//     (不可变快照)进入循环,循环没有任何改写路径;取代关系只经
//     LineageSink 追加,不做反向操作;
//   - FAIL 与 Inconclusive 端到端不同(§3):只有可信 FAIL 会被交给
//     Router 裁决;Inconclusive(验证器无法给出可信 verdict)在结构上
//     不可 repair、不可 accept,fail-safe 升级,不算候选失败、不耗预算;
//   - 取消是一等终态(§3):取消的尝试不产生 verdict、不消耗修复预算、
//     不产生尝试谱系记录(半途工作随一次性 worktree 丢弃);
//   - 修复预算是独立类型(§1/§6):Engine 只经 RepairBudget 端口问
//     「第 N 次修复能否开始」,永不读内层迭代预算;端口缺席 = 永不修复
//     (fail-safe),预算拒绝 = 显式 repair_exhausted 升级;
//   - 每个可接受结果必须携带 basis:Accepted 终态的 Basis 由 Engine 从
//     独立验证的运行引用构造,验证服务不报运行引用的 verified 视为
//     invalid_outcome,fail-safe 升级,绝不伪造 basis。
//
// # 后续任务的归属边界(本骨架刻意不做)
//
//   - P2-03(failure*.go):失败分类封闭集与 Router 实现;Engine 只执行
//     Router 的裁决,不解释失败内容;
//   - P2-04(feedback*.go):安全 RepairFeedback 投影;循环内不存在任何
//     失败细节通道,Router 的入参只有 id 与 verdict 词表;
//   - P2-05:修复从既有候选继续(前候选物化)+ LineageSink 持久化适配;
//     已交付为 lineage.go(SupersededFactAppender → LineageSink,组合根以
//     *runstore.Store 直接接线)与 ExecuteRequest.RepairBase(Engine 把
//     冻结前候选交给修复尝试;其内容在全新一次性 worktree 内的确定性
//     物化由 internal/workspace 的 RunSources.Materialize 提供);
//   - P2-06(exposure.go + feedback/ports 缝):held-out 修复安全边界 ——
//     ExecuteRequest.Feedback 是修复尝试的唯一失败材料通道(FeedbackSource
//     构造失败一律 feedback_unavailable 升级);ExposureSink/ExposureLedger
//     蓄意披露记账(append-only,谱系单调),accept 守卫把 compromise 当前
//     候选谱系的暴露变成显式 exposure_compromise 升级,strict 主张绝不
//     带病存活(INV-V14-07/08);
//   - P2-07(repair_policy*.go):RepairBudget 的策略实现(默认 2 次);
//   - P2-08:用真实 TaskExecution 与验证服务组成引擎(Executor/Verifier
//     的生产适配器、基础设施类失败的 infra/provider 重试路径);
//   - P2-09:worker 作业接线(cmd/worker 不在本包)。
//
// 骨架对基础设施类端口错误的落点是 fail-safe 升级
// (EscalationInfrastructure):端口错误绝不伪装成候选失败,也绝不静默
// 重试;P2-03/P2-08 的 infra/provider 路由在 Router/组合层细化。
//
// 本包不做进程执行、不发网络请求、不持有持久化;除 changeset(候选身份
// 词表)外不依赖任何实现包。
package closedloop
