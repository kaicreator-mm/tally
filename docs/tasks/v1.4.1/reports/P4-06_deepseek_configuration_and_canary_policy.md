# P4-06 — DeepSeek configuration and canary policy

- **执行者**: 编排者主会话(平台模型请求故障彻夜持续,子智能体派发连续 16 次失败;按既定先例接管)
- **日期**: 2026-09-13
- **基线**: master `369595e`(P5-05 之后)

## 1. L3 完成证据

**阅读顺序**: 本地契约 → `AGENTS.md`(§10:cmd/worker 组合属 P4-01 所有权已交付;Runtime lock) → DECISION_LOCKS DL-02/DL-15/DL-16 → `cmd/worker/runtimefactory.go`(P4-01 注册表:登记面即支持集合,构造失败 = 任务执行前 fail-closed)、`runtimes/dsh/runtime.go`(P4-04 Config/Start 校验面:permissionMode 枚举、renderPatchOverlay token、钉版身份闸) → `docs/guides/`(文档落点)。

外部参考: 无。

**接缝选择理由**: AC = 「无效配置在任务执行前失败;现有 Codex 缺省面不受影响」+ 要求「缺省禁用;canary 策略文档化;不向核心 TaskContract 加投机 provider 字段」。最窄接缝 = (a) `v13Config` 增 canary 配置面(4 字段,env 装载,§10 热点触碰已声明为最小 seam); (b) 注册表登记 `dsh` builder,**三重 canary 闸**(runtime=dsh + TALLY_DSH_CANARY=1 + 显式 TALLY_DSH_BIN)+ 构造期 `Config.Validate` 预检(权限枚举/overlay token 在最早闸拒绝); (c) canary 策略文档 `docs/guides/dsh-canary.md`。零 TaskContract 改动(要求 3)。

## 2. 语义变化

- `cmd/worker/runtimefactory.go`:登记 `dsh` builder —— canary 三闸 + `dsh.Config.Validate` 预检 + 显式 bin 路径要求(钉死版本面,不做隐式发现);`dsh.New` 产出钉版适配器(PinnedVersion=PinnedDSHVersion;Model/Provider 经 --patch overlay)。
- `cmd/worker/v13.go`(§10 声明式最小接缝):v13Config 增 DSHCanary/DSHBin/DSHModel/DSHProvider + env 装载(TALLY_DSH_CANARY/TALLY_DSH_BIN/TALLY_DSH_MODEL/TALLY_DSH_PROVIDER)。
- `runtimes/dsh/runtime.go`:`Config.Validate()` 预检方法(权限枚举/overlay token;不发起进程,钉版身份闸仍在 Start)。
- `docs/guides/dsh-canary.md`(新):启用三条件、绑定约束(钉版/隔离/kill-tree/诚实能力/认证)、bounded-task 边界。
- **零 TaskContract 字段**(要求 3);认证仅经继承环境(DEEPSEEK_API_KEY 不经配置面)。

## 3. 测试与命令

| 命令 | 结果 |
| --- | --- |
| `go test ./cmd/worker/ -count=1 -run 'TestDSH' -v` | PASS(3 测试:canary 缺省禁用 —— 5 组非法配置全部构造期类型化失败[无开关/无 bin/纯空白 bin/overlay 不安全 token 等];显式 canary 可构造;Codex 缺省面零影响[codex 显式构造照常、fake 缺省照常、未知 runtime 错误列出登记面含 dsh]) |
| `go test ./cmd/worker/ -count=1` / `./runtimes/dsh/` | PASS 26.4s / 9.3s(全包回归) |
| `go test ./test/arch/... -count=1` / `go build ./...` / `go vet` / `gofmt -l` / `git diff --check` | 全 PASS / 清洁 |

## 4. 基线失败观察

一轮构建错误(New 签名/未用变量/缺失 helper)与一轮测试失败(BAD MODEL 用例暴露 Start 期校验过晚)——后者直接催生 `Config.Validate` 预检设计;均为自引入,当轮修复。非基线失败。

## 5. 风险

- DSH 注册即进入登记面:未知 runtime 错误文案现在列出 dsh(登记键序派生,cosmetic;无 in-repo 字面断言依赖)。
- canary 开关是环境变量面而非 DB 配置:与 P4-03「显式配置」语义一致;组合根若未来需要 per-project canary 配比,归后续任务。

## 6. Decision Locks

未触碰、未绕过:DL-02(零 TaskContract 投机字段)、DL-15/DL-16(身份/能力诚实)、DL-08(零 EngineeringPolicy 字段)、Runtime lock(适配器经既有 agentruntime.Runtime 面)。hidden/ 未读(DL-17)。

## 7. Commit

由编排者提交(见 git log P4-06 提交信息)。
