# P4-07 — DeepSeek limited-readiness gate

- **执行者**: 编排者主会话(平台模型请求故障彻夜持续,子智能体派发连续 16 次失败;按既定先例接管)
- **日期**: 2026-09-13
- **基线**: master `3c98d7d`(P4-05 之后)

## 1. 门禁裁决

**LIMITED_READY** —— 有界候选流在真实仓库 + Tally 自有隔离上端到端证明(确定性、无钥、可复现);显式限制见 §3,升级到完全 READY 的条件见 §4。P4 lane 至此全部落地;P4-07 发布阻塞按 LIMITED_READY 裁决关闭(有限范围 release:DSH 仅 canary/bounded-task 可用,见 docs/guides/dsh-canary.md)。

## 2. 被测链(`test/e2e/v141_p407_dsh_limited_readiness_gate_test.go::TestP407DSHLimitedReadinessGate`,2.1s,双跑确定性 PASS)

真实临时 Git 仓库(PlainInit + bootstrap 提交)+ 真实进程树(替身 launcher 经 go build 现场构建,与 P4-04 包矩阵同一替身):

1. **有界任务**:headless 一次性任务在真实仓库 workspace 内读取任务输入并确定性完成(RunCompleted,最终回答 = 任务输入原文);**候选 ChangeSet 经既有闭环桥**返回(changeset.New 从 workspace 差量构造,身份非空)。
2. **负面权限**:ResumeOf 与 ModelProfile 请求在能力协商处被拒(诚实能力集 —— headless 无 resume/模型选择;P4-04 空能力声明的结构后果)。
3. **负面取消**:长运行任务(真实子进程)Cancel → `RunCancelled` + 整树终止后子进程零残留(`p407WaitGone` 进程取证)。
4. **隔离**:cwd 钉 workspace、per-session DSH_HOME、遥测禁用(P4-04 包矩阵逐项核对;本门禁经真实进程树复核取消/孤儿面;替身记录面 dump 与 P4-03 spike 观测同源)。

## 3. 显式限制(LIMITED_READY 的边界,如实声明)

1. **live 内容产出未验证**:真实 dsh 二进制的 live 运行(模型真实写文件、真实模型轮次)不在本门禁内 —— 门禁以确定性替身保持可复现、无钥;live 连通性证据由 P4-03 spike 的活体探针承担(10 探针含 2 轮真实模型对话),live 候选流归后续 live 演练。
2. **能力面受限(R-3)**:headless 不暴露 events/resume/结构化输出/实际模型观测 —— 能力协商拒绝一切此类请求;usage 恒零值。
3. **上游易变(R-2)**:developer-preview 表面;钉版闸(0.1.5-rc.1 + `--version` 观测)使任何漂移以 Start 显式失败显形。
4. **治理面**:DSH 缺省禁用(P4-06 三重 canary 闸);运行时权威永不经 DSH(DL-09;集成分支/合并由 P5 lane 具名操作承载)。

## 4. 升级到完全 READY 的条件

1. live dsh 二进制 + 真钥的真实仓库候选流演练通过(内容产出 → ChangeSet → 验证全链)。
2. Session Isolation Canary 对真实 dsh 运行并通过(`SessionIsolationVerified` 声明门置位)。
3. 上游版本变更时重跑 P4-03 spike 验证 overlay/钉版语义。

## 5. 验证命令

| 命令 | 结果 |
| --- | --- |
| `DATABASE_URL=… go test -tags=integration ./test/e2e/ -run 'TestP407DSHLimitedReadinessGate' -count=1 -v` | PASS 2.12s |
| 复跑 `-count=1` | PASS(确定性) |
| `go test ./test/arch/... -count=1` / `go build ./...` / `gofmt -l` / `git diff --check` | 全 PASS / 清洁 |

## 6. 基线失败观察

三轮开发期夹具修正(Windows .exe 后缀;workspace bootstrap 提交使 base/candidate 差量有定义;具体类型断言取 InFlightRunID)——当轮修复;非基线失败。

## 7. Decision Locks

未触碰、未绕过:DL-09(DSH 零 Git 持久权限 —— 门禁内无任何 Git 写经 runtime)、DL-10(无反事实推断)、DL-14(取消/超时类型化映射)、DL-15(身份诚实)、DL-16(能力诚实)、Runtime lock(零协议扩张)、INV-EXEC-01(workspace 可丢弃、owner 零变更)。hidden/ 未读(DL-17)。

## 8. Commit

由编排者提交(见 git log P4-07 提交信息)。
