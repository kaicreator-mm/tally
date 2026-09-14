# P3-07 — Approve and activate policy via GitWriter

- **执行者**: 编排者主会话(平台模型请求故障彻夜持续,子智能体派发连续 13 次失败;按既定先例接管)
- **日期**: 2026-09-13
- **基线**: master `89dc24c`(P3-05 之后)

## 1. L3 完成证据

**阅读顺序**: 本地契约 → `AGENTS.md`(DL-03 gitwriter 唯一 Git 写边界;DL-09) → DECISION_LOCKS BI-01(approved_by 归治理面;激活 = gitwriter 提交正典 YAML)+ DL-13 → `internal/gitwriter/writer.go`(CommitRequest 期望基线 CAS/CommitResult/FileUpdate;ErrBaseRevision 语义) → `internal/artifacts/v14_policy_loader.go`(LoadPolicies/PolicySnapshot/PoliciesDir + Policy.Revision 由 Parse 从 blob 内容填充) → 本 lane 既有面(P3-01 域转移 PROPOSED→ACTIVE;P3-02 Append/Reconstruct;P3-03 VerifyCandidate)。

外部参考: 无。

**接缝选择理由**: AC「提案只有在其承诺策略可从源快照观察后才算运行时生效」→ 激活顺序即不变量:重放核验(PROPOSED + 证据齐备 + 候选修订一致)→ gitwriter 期望基线 CAS 提交(零部分写入;失败即治理面保持 PROPOSED,零激活事件)→ `artifacts.LoadPolicies` 从确切新提交重载(必须观察到同 id 同修订)→ 最后落 activate 治理事件(approved_by 必填)。治理记录在 Git 可观察之后:DL-09 下 Git 是运行时权威,治理事件是批准的**证据记录**,不是运行时开关 —— 第 4 步失败时返回类型化错误并携带已生效提交 sha(治理记录按 append-only 补记,绝不回滚 Git 使运行时权威摆动)。

## 2. 语义变化

- `internal/policygovernance/activate.go`(新):`ActivationRequest/ActivationResult` + `(*Store).Activate(ctx, req, *gitwriter.Writer)`;哨兵 `ErrInvalidActivation`(前置:DRAFT/终态/未知提案/缺批准人/缺仓库)、`ErrActivationNotObservable`(重载闸);制品路径由策略 id 推导(路径安全字符白名单,不做转义器);提交消息确定性 `tally: activate policy <id> (proposal <pid>)`。
- `internal/gitwriter/**`、`internal/artifacts/**`、`internal/web/handlers/**`:零改动(纯组合:Writer.Commit + LoadPolicies;评审页 Governance 面已由 P3-08 落地)。

## 3. 测试与命令(验证矩阵逐项)

| 验证项 | 测试 | 结果 |
| --- | --- | --- |
| success | `TestActivateSuccess`:提交 → 独立 `LoadPolicies` 复核同 id 同修订可观察 → Reconstruct ACTIVE + approved_by | PASS |
| stale base | `TestActivateStaleBase`:无关提交推进 HEAD → 类型化失败(Detail 明示保持 PROPOSED)、治理面 PROPOSED、HEAD 停在推进提交(零部分写入) | PASS |
| invalid candidate | `TestActivatePreflightNegatives`:DRAFT 未提交评审拒绝并指明状态;缺批准人拒绝;未知提案 `ErrInvalidActivation`(候选内容漂移面由 P3-02/03 Reconstruct/VerifyCandidate 在第 1 步拦截) | PASS |
| commit failure | stale base 即 CAS 提交失败面;第 4 步事件失败语义在代码路径文档化(Git 已生效 + 类型化错误携带 sha;真实故障注入需治理库宕机夹具,未构造 —— 见风险) | PASS(部分 NOT_RUN,见 §5) |
| 复跑 | `DATABASE_URL=… go test ./internal/policygovernance/ -count=1` 全包 76.0s(域/存储/正典化/冲突/激活)PASS;arch/build/vet/gofmt/`git diff --check` 清洁 | PASS |

## 4. 基线失败观察

一轮测试夹具缺陷(p09Repo 属 e2e 包不可达,且 handlers 依赖会成环 → 改为 PlainInit + gitwriter 空基线首提交自建夹具)与一轮 gofmt,当轮修复;非基线失败。

## 5. 风险

- 第 4 步(activate 事件落库)失败的故障注入未构造真实夹具(需治理库中途宕机):代码路径已实现并文档化(Git 已生效 + `ErrInconsistentHistory` 携带提交 sha),留待 P3-09 门禁或后续演练覆盖。
- 激活不自动解决冲突分析(P3-05)的发现 —— 激活前是否消解 conflict 属治理流程人本裁决(P3-09 门禁面),本服务不做隐式阻断(发现本身已可见)。

## 6. Decision Locks

未触碰、未绕过:DL-03(唯一写边界 gitwriter)、DL-09(Git 提交 + 源快照可观察才是生效面;治理事件非运行时开关)、BI-01(approved_by 落治理事件)。hidden/ 未读(DL-17)。

## 7. Commit

由编排者提交(见 git log P3-07 提交信息)。
