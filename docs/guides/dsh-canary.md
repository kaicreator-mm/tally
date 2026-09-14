# DSH Canary 策略(P4-06)

DeepSeek Harness(`dsh`,developer-preview)经 `runtimes/dsh` 适配器接入 Tally worker。**缺省禁用**:不显式配置,worker 永不选择 dsh —— 现有 Codex 缺省面零影响。

## 启用(canary/bounded-task 策略)

三个条件同时满足才可构造(任一缺席在任务执行前类型化失败,绝不开 runtime 会话/建 worktree/落 run 事实):

| 环境变量 | 含义 |
| --- | --- |
| `TALLY_V13_RUNTIME=dsh` | 显式选择 dsh 适配器 |
| `TALLY_DSH_CANARY=1` | canary 显式开关(缺省禁用的结构闸) |
| `TALLY_DSH_BIN=<path>` | dsh launcher 二进制的显式路径(钉死 developer-preview 版本面,不做隐式发现) |

可选期望配置(写入生成的 `--patch` YAML overlay;与实际生效模型是两个事实,headless 不可观察):

| 环境变量 | 含义 |
| --- | --- |
| `TALLY_DSH_MODEL` | 期望模型(如 `deepseek-chat`;token 仅允许字母数字与 `._-/`) |
| `TALLY_DSH_PROVIDER` | 期望提供者(如 `deepseek-official`) |

## 绑定约束(来自 P4-03 spike 裁决,不可协商)

- **钉版**:`@deepseek-ai/dsh@0.1.5-rc.1`;Start 现场观察 `--version` 与钉住值不一致即 fail-closed(会话拒绝,无任务执行)。
- **隔离**:每会话全新临时 `DSH_HOME`(无移交,Close 删除);强制 `DSH_TELEMETRY_MODE=DISABLED`(默认遥测外送 DeepSeek OTLP);`DSH_PERMISSION_MODE` 强制为适配器配置值,继承环境的 `danger-full-access` 不得静默升级。
- **kill-tree**:取消/关闭走 `taskkill /T /F`(Windows)或进程组 SIGKILL(POSIX);`WaitDelay` 兜底孤儿管道。
- **诚实能力集**:headless 无 events/resume/fork/structured-output/usage/实际模型观测 —— 能力协商拒绝一切此类请求;usage 恒为零值(宁缺毋假)。
- **认证**:仅经 `DEEPSEEK_API_KEY` 环境变量(继承注入;适配器不读、不存凭据文件)。

## bounded-task 边界

- canary 面向**有界任务**:单仓、单任务、一次性 worktree;不用于长会话或交互式流程(headless 形态本身即一次性)。
- 失败映射:进程异常退出(含崩溃)→ RunFailed;ctx 取消/Close → RunCancelled;成功退出但无最终回答 → RunFailed(契约未兑现,fail-safe 不猜测成功)。
- 已知缺口见 `docs/tasks/v1.4.1/reports/P4-04_deepseek_harness_agentruntime_adapter_skeleton.md`(R-1~R-5)。

## Live readiness gate

真实外部验证使用与生产 worker 相同的 launcher 配置面：

```bash
TALLY_DSH_LIVE=1 \
TALLY_DSH_BIN=/absolute/path/to/dsh \
DEEPSEEK_API_KEY=... \
go test ./runtimes/dsh -run TestLiveDSHSurface -count=1
```

`TALLY_DSH_LIVE` 未设置时该测试是环境性 `SKIP`。一旦显式设置，缺
`TALLY_DSH_BIN`、launcher 不可用、缺 `DEEPSEEK_API_KEY` 或真实回合失败均为
**FAIL**，不能再降级为 SKIP；这样 `Strong`/release readiness 不会出现假绿。
