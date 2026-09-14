# P3-04 — Evidence-bounded historical replay

- **执行者**: 编排者主会话(平台模型请求故障彻夜持续,子智能体派发连续 15 次失败;按既定先例接管;HIGH 任务的接缝判定与被拒替代方案见 §2)
- **日期**: 2026-09-13
- **基线**: master `2b23a0e`(P3-06 之后)

## 1. L3 完成证据

**阅读顺序**: 本地契约 → `AGENTS.md`(§11 值语义/封闭转移;DL-10) → DECISION_LOCKS DL-10(重放证据边界,不可发明反事实结局)+ DL-09 → 参照测试 `engineeringpolicy/match_test.go`(MatchInput/risk.Result 构造惯例)、`internal/risk/derive.go`(Derive:detected/unknown 分类,unknown 原样保真)、`contract/compile_test.go` 与 `taskstate/basis_test.go`(投影惯例,只读消费)、`outcome/store_test.go`(五类事实面,只读) → 既有接缝 `engineeringpolicy/{match,hash}.go`(Match/PolicyMatchCoverage:matched basis/unknown/unmatched 信号保真/Confidence)、`assurance/select.go`(Select:条款选择;见 §2 裁定)、本 lane P3-05 AnalyzeConflicts。

外部参考: 无实际使用。

## 2. 接缝判定(HIGH 语义裁决)与被拒替代方案

** chosen seam**: 纯函数 `ReplayCandidate(ReplayInput) ReplayResult`(policygovernance/replay.go 新文件,零既有文件改动)—— 输入 = 证据快照(候选 + 当时策略集 + 显式声明的录得触发输入 + 录得的门禁键结局),输出 = 匹配判定/差量/逐键结局/冲突/信号保真。匹配复用 `engineeringpolicy.Match`(零第二 DSL);冲突复用 P3-05 `AnalyzeConflicts`;全部列表确定性排序。

**被拒替代方案**:
1. *重放直接查询 outcome/taskstate/store 事实面*:拒绝 —— 重放必须对「证据快照」纯函数化(确定性 AC 与可测试性);事实面 → 快照的投影由调用方完成,重放对来源不敏感(同一快照 = 同一结论,来源无关)。
2. *从任务状态推导历史 achieved assurance 并比较*:拒绝 —— taskstate 的 achieved 投影依赖运行时覆盖报告,P3-04 的治理词汇内政策不携带 assurance 等级(DL-08);把「要求键集差量」如实作为 assurance 差量的治理面表达,不伪造等级比较。
3. *为「历史代码会不会过」建立模拟执行*:拒绝 —— 任何未录得结局的推断都违反 DL-10;结局只从录得的验证绑定结果(键 → passed/failed)确定,缺席键 = `insufficient_evidence` 一等输出。
4. *Match 覆盖率之外的自行触发判定*:拒绝 —— 触发匹配 100% 复用 `engineeringpolicy.Match`(含 unknown/unmatched 信号保真),零第二判定器。

## 3. 语义变化(新文件)

- `ReplayMatchDecision`(matched|unmatched|**insufficient_evidence** —— 一等结论):触发轴未录得(`TriggerRecorded=false`)时强制 INSUFFICIENT,不猜 matched/unmatched。
- `ReplayGateStatus`(would_have_passed|would_have_failed|**insufficient_evidence**):逐差量键,只认 `RecordedGateResults` 录得事实(验证绑定结果按 gate 键的投影)。
- 差量:`NewlyRequiredKeys`(候选 requires − 当时命中历史集的 requires 并集)、`NewlyGatedKeys`(all_of 同理)、`UnchangedKeys`;assurance 差量的治理面表达 = 要求键集差量(§2.2 裁定)。
- `Conflicts`:候选加入历史集后的 P3-05 发现(仅涉及候选者)。
- `Unknown/UnmatchedRiskSignals` 原样保真(INV-V14-09)。
- 哨兵 `ErrInvalidReplayInput`;候选 Validate fail-closed;历史集合非法由 Match 类型化透传。

## 4. 测试与命令

| 命令 | 结果 |
| --- | --- |
| `go test ./internal/policygovernance/ -count=1 -run 'TestReplay' -count=2` | PASS(确定性 AC:同快照双跑恒等;8 测试) |
| `DATABASE_URL=… go test ./internal/policygovernance/ -count=1` | PASS 76.0s(全包回归:域/存储/正典化/冲突/激活/重放) |
| `go test ./internal/engineeringpolicy/... ./internal/outcome/... ./internal/taskstate/... -count=1` | PASS(三邻包零改动回归) |
| `go test ./test/arch/... -count=1` / `go build ./...` / `go vet` / `gofmt -l` / `git diff --check` | 全 PASS / 清洁 |

表驱动/负测试:matched+差量+passed、k9 无录得 → insufficient_evidence 一等、录得 failed → would_have_failed、触发轴未录得 → 匹配 INSUFFICIENT 且零差量输出(证据边界:未录得时连差量都不给)、unmatched(录得轴与候选不交)、unknown 信号保真(枚举外 id 原样输出)、同 id 冲突可见、确定性双跑、无效候选拒绝。开发期三轮失败均为夹具/断言错误(非被测代码缺陷),当轮修正。

## 5. 风险与不确定性

- `RecordedGateResults` 的键必须与策略 requires 键同一命名空间(fresh_database 等)—— 投影由调用方负责;P6-04 诊断确认 trial 验证绑定确实按 gate 键记录,生产验证面同构。
- 历史策略集来自 pinned 快照(调用方提供):快照选取的时点正确性是调用方契约,重放对快照内容不敏感但结论以快照为界。

## 6. Decision Locks

未触碰、未绕过:DL-10(insufficient_evidence 一等 + 零反事实推断 —— 结构性)、DL-08(engineeringpolicy 零改动)、DL-09(Git 面照旧)、DL-12(outcome 只读)、INV-V14-09(unknown 保真)。hidden/ 未读(DL-17)。

## 7. Commit

由编排者提交(见 git log P3-04 提交信息)。
