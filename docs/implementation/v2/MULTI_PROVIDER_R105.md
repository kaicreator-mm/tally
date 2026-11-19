# MULTI_PROVIDER_R105 — 多 Provider 接入 v2 Task 执行链（统一契约层）

状态：`v2.1 candidate / NOT for current v2 MVP`（#392 修正案将 dual-provider 推迟到 v2.1；本文档不改任何 MVP 完成条件）。
基准：`version/v2` @ `dff5329e`（本 PR 基点 exact SHA）。
产品决议：一个 Task 的一次 Attempt 只用一个 Provider；互不冲突且就绪的 Task 可并行；默认 Provider=codex（现有路径字节不变）；其余 Provider 显式选择、默认 DISABLED，probe 验证通过才可启用；无失败自动回退。

## 交付物（本 PR：Phase 2+3 共享层，全部新文件）

| 文件 | 职责 |
| --- | --- |
| `web/v2/provider-contract.mjs` | 统一 Provider 接口：输入/输出校验器、状态词汇表、evidence-only 守卫、准入期选择策略 |
| `web/v2/provider-registry.mjs` | 可选 Provider 注册表（结构校验/enabled 行/probe 缓存/AVAILABLE·DISABLED·UNKNOWN 描述符）；codex 不入注册表 |
| `web/v2/task-provider-dispatch.mjs` | Task 级派发：可信上下文 providerId 戳记、每 Task 单活槽位（原子预约）、五路并行准入、UNKNOWN 不盲重启 |

测试：`web/v2/{provider-contract,provider-registry,task-provider-dispatch}.test.mjs`（26 用例，node:test + fake 注入，与 zcode-provider.test.mjs 同模式）。

## 统一 TaskProvider 接口（五成员）

```
{providerId, probe(config), execute(input), cancel({controller}), readback(config, candidate)}
```

- `probe`：分级 VERSION→CREDENTIALS→CANARY；任一级不过 ⇒ `available:false`，永不 spawn。
- `execute(input)`：统一输入 `{providerId, taskId, attemptId, sourceSha(40hex), worktreePath, allowedWriteSet[], taskPrompt, issueRef?, hostAuthorization{}, providerConfig{}, signal?, timeoutMs?}`。
- `cancel({controller})`：AbortController（调用方持有）是唯一取消句柄；AbortSignal 自身无法被外部 abort。host 在 execute 时监听 `signal` 的 abort 杀进程树。
- `readback`：独立 git 回读（parent/tree/paths 全匹配才 verified），agent 自报永不是证据。

统一输出 `{provider, providerId, taskId?, attemptId, sourceSha, state, reason, agentLaunch, process{status,exitCode,pid,startedAt,terminatedAt}, candidateTree/Commit/Ref/Paths?, error?, taskPass:false, releasePass:false}`；
`state ∈ {LOCAL_CANDIDATE_READY, LOCAL_ATTEMPT_FAILED, LOCAL_ATTEMPT_UNKNOWN, LOCAL_ATTEMPT_CANCELLED, AGENT_NOT_EXECUTED, NOT_ADMITTED}`。
**taskPass/releasePass 恒 false**：接口只返回执行证据，不授予 Task PASS / Review PASS / Release PASS（`OUTCOME_GRANTS_PASS` 校验拒绝越权）。

## 派发纪律（task-provider-dispatch）

1. providerId 只来自**可信 Attempt 上下文**（Host/Claim 戳记）；浏览器字段仅 advisory，与可信戳不一致 ⇒ `PROVIDER_MISMATCH_REQUEST_VS_TRUSTED` 拒绝。
2. Claim 未准入 ⇒ 拒绝；准入端口缺失 ⇒ fail-closed（`*_NOT_CONNECTED`）。
3. codex 路由返回 `route:'EXISTING_CHAIN'`，本层**不执行** codex——现有链路与默认行为不变。
4. 每 Task 原子预约单活槽位（同步 check-and-set，先于一切异步端口；等价 local-codex host 用 CAS grant 关闭的跨 attempt 竞态——见 dispatch 测试「same task cannot run two attempts at once」）。
5. 并行五条件（全部注入端口、fail-closed）：DAG 前置满足 / Claim 有效 / WriteSet 无冲突 / worktree+ref 独立 / Provider 容量允许。
6. 冲突、取消、超时、UNKNOWN 复用 Host 机制；UNKNOWN 保持槽位占用（操作员对账，不盲重启）；终态释放槽位。失败不切换 Provider（测试「provider failure stays on the same provider — no codex fallback」）。

## 显式请求不可用 Provider = 拒绝（非静默回退）

`resolveProviderSelection` 对显式非 codex 请求：未注册/未 enabled/probe 不可用 ⇒ `admitted:false`（`PROVIDER_NOT_ENABLED` / `PROVIDER_CAPABILITY_UNVERIFIED`）。与 zcode-provider 内 journey 接缝的 `resolveZcodeProviderSelection`（回落 codex）语义不同：那是 journey 层安全默认，这是 Task 派发层 fail-closed。两者并存是刻意的。

## Provider 盘点（as-of 本提交；真实执行状态见各 Provider PR 收据）

| Provider | 载体 | 默认 | 真实执行 |
| --- | --- | --- | --- |
| codex | 现有 local-codex-host（不在注册表） | 默认 | 已有真实收据（既有链路） |
| zcode | PR #470 `web/v2/zcode-provider.mjs` + 统一 facade | DISABLED | 真实收据已录（#470，candidate 434ac166） |
| dsh (DeepSeek) | PR #467 `packages/provider-dsh` + 统一 facade | DISABLED | 本 PR 未跑真实 API（容量由 controller 统一安排） |
| opencode | R105 `web/v2/opencode-provider.mjs`（契约源 PR #471 Go） | DISABLED | 见该 PR 收据 |
| pi | R105 `web/v2/pi-provider.mjs`（契约源 PR #469 Go） | DISABLED | 见该 PR 收据 |
| grok | R105 `web/v2/grok-provider.mjs`（契约源 PR #468 Go） | DISABLED | 无真实凭据 ⇒ 仅契约验证，保持 DISABLED |

## REMAINING_INTEGRATION（Phase 4，阻塞于 R102 写集释放）

以下接线需要修改 R102 活动写集文件，在 `TALLY_V2_R102_THREE_SEAM_GUARDED_INTEGRATION` 释放共享源码写集之前**不动**：
- `web/v2/journey-server.mjs`：journey 执行入口注入 registry/dispatch（provider 选择 UI → 受信 admission）。
- `web/v2/serve.mjs`：Host 组装处挂载 Provider Registry 与 Task Dispatch 端口（DAG/Claim/写集/容量端口接线）。
- `experiments/v2/selected-business-build/{bundle-local-codex.mjs,local-codex-shim.mjs,producer.mjs}`：B→K→Host/C 链上把 providerId 戳进可信 Attempt 上下文。
新文件先行部分（不冲突）：本契约层 + 各 Provider 适配器/facade + 注入式端口定义；接线时只需在各热点文件做一次注入调用。

## 验证状态（诚实记录）

- 本层 26 用例 node:test 全绿（Windows Node v24.21.0；真实退出码 EXIT=0）。
- `find web/v2 -name '*.test.mjs'` 全 lane 本地预跑：仅 1 例环境型失败（`version-task-dag-host-http` 需 lane 前置 `npm --prefix packages/tally-github-native-c install`，按 CI 同款命令补装后 6/6 绿——非本 diff 引入）。
- 真实并行正例（≥2 Provider 两个不冲突 Task）、真实 Candidate 生成与 git 回读：NOT_RUN（Phase 5 由 controller 统一执行后在本文件补记）。
- Go Worker 测试 PASS 不作为 v2 TypeScript Host PASS（本层无 Go 依赖）。

Refs: #64
