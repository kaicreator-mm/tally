# P5-06 — Local Git control-plane gate

- **执行者**: 编排者主会话(平台模型请求故障彻夜持续,子智能体派发连续 16 次失败;按既定先例接管)
- **日期**: 2026-09-13
- **基线**: master `369595e`(P5-05 之后)

## 1. 门禁裁决

**PASS** —— P5 lane 全矩阵(集成/漂移/冲突族/受保护分支/并发写/运行时天花板/审计列举)在真实临时仓库上端到端证明,零生产代码改动,零远程凭证/零 push(P5-01..05 交付面原样组合)。

## 2. 被测矩阵(`test/e2e/v141_p506_git_control_plane_gate_test.go::TestP506LocalGitControlPlaneGate`,1.6s,双跑确定性 PASS)

| 条件 | 验证 | 结果 |
| --- | --- | --- |
| 集成主链 | EnsureIntegrationBranch(确定性分支,CAS create,分支令牌)→ Integrate(候选内容经唯一写边界 CAS 提交到分支)→ Merge(确定性合并提交,目标 ref CAS 推进,受保护 main 经批准令牌) | PASS |
| 漂移 | 持过期基线的 Integrate → `ErrStaleBase`,HEAD 不动(败者零部分写入;预检 + gitwriter 锁内二次校验双层 CAS) | PASS |
| 并发写 | 败者写者持过期基线失败即 CAS 语义;gitwriter 跨进程 per-repo 锁串行化(既有 0011/0019/0020 同型纪律) | PASS |
| 冲突族 | 已合并重放 → `ErrAlreadyMerged`;分支漂移 → `ErrIntegrationBranchExists`(P5-02 CAS create 使覆盖结构上不可能);真三方可检冲突由 merge_test.go 包级 12 测试覆盖(本门禁组合 e2e 面,报告如实声明) | PASS |
| 受保护分支 | Decider 声明 main:无批准 → `ErrProtectedTargetDenied`;带批准 → 令牌签发 | PASS |
| 运行时天花板 | RuntimeCaller 请求 L3 → `ErrRuntimeAuthorityExceeded`(即使携带有效批准;P5-04 结构执行点) | PASS |
| 审计列举 | ListIntegrationRefs(P5-05)可见治理命名空间分支 | PASS |
| AC1 零远程 | 仓库无任何 remote 配置(git.Remotes 断言空);全部操作离线 | PASS |
| AC2 并发 CAS | 见漂移/并发行 | PASS |

## 3. 验证命令

| 命令 | 结果 |
| --- | --- |
| `DATABASE_URL=… go test -tags=integration ./test/e2e/ -run 'TestP506LocalGitControlPlaneGate' -count=1 -v` | PASS 1.61s |
| 复跑 `-count=1` | PASS(确定性) |
| `go test ./test/arch/... -count=1` / `go build ./...` / `gofmt -l` / `git diff --check` | 全 PASS / 清洁 |

## 4. 基线失败观察

两轮开发期修正:PlainInit 默认分支名不定 → 显式创建 refs/heads/main(合并走 ref 级 CAS 无需检出);分支编排(集成分支检出后 Integrate 落分支,合并才有内容)。均为夹具问题,非基线失败;P5 lane 全部既有测试不变绿。

## 5. 风险

- 真三方可检冲突(merge-tree 冲突路径)在包级 merge_test.go 覆盖(12 测试:clean true merge/conflict/target drift/branch drift/replay/checked-out 等),本门禁的 e2e 组合面覆盖 already-merged 与 branch-drift 两个类型化冲突 —— 三层组合的证据面完整。
- 并发写采用「败者持过期基线」的确定性 CAS 面而非真并发线程竞态(P5-03 已注记:CAS 终局竞态分类为代码评审覆盖,无注入钩子不可确定性测试)。

## 6. Decision Locks

未触碰、未绕过:DL-03(gitwriter 唯一写边界)、DL-13(集成分支/合并提交 append-only)、DL-14 同型(CAS + 锁)、P5-04 权限阶梯(运行时天花板结构性保持)。hidden/ 未读(DL-17)。

## 7. Commit

由编排者提交(见 git log P5-06 提交信息)。
