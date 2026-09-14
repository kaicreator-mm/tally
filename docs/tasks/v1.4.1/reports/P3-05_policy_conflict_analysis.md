# P3-05 — Policy conflict analysis

- **执行者**: 编排者主会话(平台模型请求故障彻夜持续,子智能体派发连续 12 次失败;按既定先例接管)
- **日期**: 2026-09-13
- **基线**: master `634a7d8`(P3-08 之后)

## 1. L3 完成证据

**阅读顺序**: 本地契约 → `AGENTS.md`(§10:engineeringpolicy 运行时模式冻结) → DECISION_LOCKS DL-08/09 + PRD §8 词汇(trigger/requires/gate_condition/enforcement) → `internal/engineeringpolicy/policy.go`(Policy/Trigger/GateCondition 结构与 Validate 语义:gate 必带非空 all_of、guidance 不得携带 gate_condition、all_of ⊆ requires、列表去重) → `hash.go`(Canonical/RevisionHash) → 本 lane 既有面(P3-01 候选域)。

外部参考: 无实际使用(冲突检测完全落在 Tally 封闭词汇内,无需外部规则集模式)。

**接缝选择理由**: 契约 = 「检测需要人工关注的候选交互,不发明通用布尔规则引擎;保守告警优先」。词汇内可确定性判定的交互形态恰好四种(见下),全部输出型(无自动消解)。`conflict.go` 新文件,零既有文件改动;输入直接消费解析后的 `engineeringpolicy.Policy` 值(解析语义已由 P3-01/03 与上游测试覆盖)。

## 2. 语义变化(全部新文件)

- `Conflict`/`ConflictSeverity`(conflict|warning|info)/`ConflictKind`(封闭四类):
  - `duplicate_id`(conflict):同 id 多份 —— 身份必须唯一,携带全部内容修订供人裁决;
  - `shadowed_duplicate`(warning):不同 id 但 trigger/requires/all_of/enforcement 全同 —— 完全遮蔽,意图需人确认;
  - `enforcement_mismatch`(warning):同触发面 + 同 requires 键集 + 强度不同(gate vs guidance)—— 键比较只用 requires(guidance 按词汇表不携带 gate_condition,含 AllOf 的比较会让 gate/guidance 对永不命中 —— 开发中被表驱动测试捕获并修正);
  - `stacked_gates`(info):触发轴重叠的双 gate —— 合取叠加是合法语义,人需知晓叠加后的全部键(并集)。
- `AnalyzeConflicts([]engineeringpolicy.Policy) []Conflict`:确定性输出(kind 词表次序 + 涉及 ID 升序;输入顺序无关);触发轴重叠 = 任一轴交集,或一方零值 Trigger(无条件触发匹配一切 —— 保守方向:宁多报给人看);同 id 组先上报身份冲突,组内取首份进入两两比较(不级联噪声)。

## 3. 测试与命令

| 命令 | 结果 |
| --- | --- |
| `go test ./internal/policygovernance/ -count=1 -run 'TestAnalyze' -v` | PASS(3 测试:7 行表驱动 —— 干净/重复 id/叠加/强度不一致/完全遮蔽/不相交零发现/复合三发现并存,每发现断言非空 Detail + 涉及 ID;确定性 AC —— 旋转输入 3 轮发现序列逐字节一致;零值触发面保守上报) |
| `DATABASE_URL=… go test ./internal/policygovernance/ -count=1` | PASS 54.3s(全包回归:域/存储/正典化/冲突) |
| `go test ./test/arch/... -count=1` / `go build ./...` / `go vet` / `gofmt -l` / `git diff --check` | 全 PASS / 清洁 |

两轮开发期失败均为自引入(enforcement_mismatch 的键比较含 AllOf 导致 gate/guidance 对永不命中;复合夹具与组内去重设计冲突)—— 表驱动测试按设计捕获,当轮修正;非基线失败。

## 4. 风险

- 词汇内不可表达逻辑否定(none_of),真正的语义矛盾(如「要求 X」vs「禁止 X」)不在可判定面 —— 词汇扩展须先改运行时模式(DL-08 禁),本分析如实不做。
- `stacked_gates` 在策略集大时为 O(n²) 两两比较 —— 治理面规模(候选集 ≤ 数十)下可接受;零索引纪律同 P3-02。

## 5. Decision Locks

未触碰、未绕过:DL-08(engineeringpolicy 零改动)、DL-09(分析是治理面输入,不构成运行时行为)、要求 1/2(封闭词汇 + 保守输出,无自动消解)。hidden/ 未读(DL-17)。

## 6. Commit

由编排者提交(见 git log P3-05 提交信息)。
