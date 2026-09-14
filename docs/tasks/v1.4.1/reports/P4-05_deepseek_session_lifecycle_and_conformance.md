# P4-05 — DeepSeek session lifecycle and conformance

- **执行者**: 编排者主会话(平台模型请求故障彻夜持续,子智能体派发连续 16 次失败;按既定先例接管)
- **日期**: 2026-09-13
- **基线**: master `fe9e76c`(P4-06 之后)

## 1. L3 完成证据

**阅读顺序**: 本地契约 → `AGENTS.md`(Runtime lock;DL-15/DL-16) → DECISION_LOCKS DL-14(取消/超时不伪装成候选失败) → 既有生命周期面(P4-04 交付):`runtimes/dsh/runtime_test.go`(既有矩阵:崩溃收敛+会话存活、ctx 取消 RunCancelled、Close 杀在途+拒绝后续操作+幂等、13 场景 conformance、能力诚实、身份闸、隔离强制)、`runtimes/dsh/{process,process_windows,process_unix}.go`(kill-tree + WaitDelay)、`testdata/fakedsh/main.go`(替身契约:记录面/子进程/长运行标记)。

外部参考: 无。

**接缝选择理由**: AC = 「取消/超时测试中零孤儿进程;owner 仓库在指派 workspace 之外零变更」。P4-04 已交付生命周期语义的主体(crash/cancel/close/幂等/13 场景 conformance);P4-05 的增量 = 矩阵中两个此前未显式覆盖的面,落在本包测试内(路径允许,夹具/助手同源):(a) ctx 超时路径的孤儿零残留(既有 cancel 测试未含真实进程树取证); (b) owner 仓库零变更 + per-session DSH_HOME 随 Close 删除(工作区钉定的效果级证明延伸到「workspace 之外零写入」)。

**被拒替代方案**: (a) 在 test/e2e 重写矩阵 —— 夹具/替身助手全在包内,复制=第二事实源;包内延伸 + conformance 引用即最小面。(b) 模拟执行结局 —— DL-10 禁。

## 2. 语义变化(仅测试;生产代码零改动)

- `TestP405ContextTimeoutLeavesNoOrphan`:ctx 时限到期触发 Cancel 路径 → 收敛 `RunCancelled`;launcher 子进程(真实进程树)在整树终止后经 `waitForGone` 取证零残留。既有既有断言不放宽。
- `TestP405NoOwnerRepoMutation`:owner 代理目录在 Execute/Close 全生命周期逐字节零变更(无新文件/无触碰);替身观测 cwd 钉定 workspace;per-session DSH_HOME 随 Close 删除(隔离零跨运行残留)。
- 生产代码零改动(P4-04 已交付全部语义;本任务为矩阵补全)。

## 3. 测试与命令

| 命令 | 结果 |
| --- | --- |
| `go test ./runtimes/dsh/ -count=1 -run 'TestP405' -v` | PASS ×2(超时收敛 RunCancelled + 真实进程树孤儿零残留;owner 目录哈希前后一致 + cwd 钉定 + DSH_HOME 已删除) |
| `go test ./runtimes/dsh/ -count=1`(全包) | PASS 9.6s(36 测试;既有崩溃/取消/关闭/一致性断言不变绿) |
| `go build ./...` / `go vet` / `gofmt -l` / `git diff --check` | 全 PASS / 清洁 |

一轮夹具缺失(替身 cwd 内 marker 文件未创建)当轮修复;非基线失败。

## 4. 风险

- 孤儿取证覆盖「取消/超时」路径(P5-04/P3-07 kill-tree + WaitDelay 的联合效果);「外部单进程 kill launcher」的孤儿面由 P4-04 崩溃测试覆盖 —— 两个方向合起来即生命周期全谱。
- Windows taskkill 依赖系统 PATH(见 P4-04 风险);缺失时 killTree 显式报错、WaitDelay 仍保收敛。

## 5. Decision Locks

未触碰、未绕过:DL-14(取消/超时不伪装成候选失败 —— RunCancelled/RunFailed 类型化映射测试钉死)、DL-15(身份观察)、Runtime lock(零协议扩张)。hidden/ 未读(DL-17)。

## 6. Commit

由编排者提交(见 git log P4-05 提交信息)。
