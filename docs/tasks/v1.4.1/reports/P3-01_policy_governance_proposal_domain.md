# P3-01 — Policy governance proposal domain

- **执行者**: 编排者主会话（平台模型请求故障持续,子智能体派发连续 8 次失败;按 P4-04/P2-04/P2-05 先例接管）
- **日期**: 2026-09-13
- **基线**: master `b3b5c83`（P2-05 之后）

## 1. L3 完成证据

**阅读顺序**: 本地契约 → `AGENTS.md`（§10 共享热点:engineeringpolicy 运行时模式冻结,P3 消费不扩展）→ `DECISION_LOCKS.md` DL-08/09/10 + **BI-01**（治理生命周期元数据的封闭词表与归属裁定 —— 本任务的设计权威）→ engineeringpolicy 源码（`parse.go` KnownFields 严格模式 + forbiddenFields 执行策略字段拒绝、`policy.go` Policy 窄模式 + Validate、`hash.go` RevisionHash/Canonical/GateKeys）→ 参照测试 `engineeringpolicy/{parse_test,policy_test,hash_test}.go`、`test/arch/v14_runtime_boundary_test.go::TestV14EngineeringPolicyRejectsExecutionPolicyFields`（只读,确认既有守卫面）。

外部参考: 未实际使用（EXT-GITHUB-RULESETS / EXT-GITLAB-APPROVAL 为模式参考;本域面足够窄,无需引入外部模式）。

**接缝选择理由**: 契约 = 「表示提案内容/证据引用/治理身份,不改运行时 EngineeringPolicy 模式」。最窄接缝 = 新包 `internal/policygovernance` 的值域（Candidate + Proposal + EvidenceRef + Status）,候选侧组合 `engineeringpolicy.Parse`（正典解析复用,零平行解析器 —— 不建第二 DSL/引擎,Don't #1）,治理侧用 BI-01 指定词表。零持久化（P3-02）、零 Git 写（P3-07）、零 UI（P3-06）。

## 2. 语义变化（全部新文件,零既有文件改动）

- `Candidate`: 不可变候选（raw 深拷贝 + 正典 `engineeringpolicy.Policy` + 内容修订;构造即拒绝不可解析内容 —— DL-09 下它永远进不了运行时,早拒优于晚炸）。
- `EvidenceRef` + 封闭 kind 词表 `outcome|changeset|verification|task`: 证据只指向既有事实面。
- `Proposal`: BI-01 元数据词表（status `DRAFT/PROPOSED/ACTIVE/SUPERSEDED/REJECTED`、evidence_refs、approved_by、supersedes、scope）+ 四个具名封闭转移（`Submit`/`Activate`/`Reject`/`Supersede`）;失败时原样返回接收者;`REJECTED/SUPERSEDED` 终态;supersession append-only。
- 哨兵: `ErrInvalidProposal` / `ErrInvalidTransition`（errors.Is 判别）。

## 3. 测试与命令

| 命令 | 结果 |
| --- | --- |
| `go test ./internal/policygovernance/ -count=1 -v` | PASS 0.687s(5 测试:AC 正面 / 不可变性 / 治理字段污染负例 / 构造负路径 / 生命周期全表) |
| `go vet ./internal/policygovernance/` / `gofmt -l` | PASS / 清洁 |
| `go build ./...` | PASS |
| `go test ./test/arch/... -count=1` | PASS(1.773s;既有 runtime-boundary 守卫全绿) |

一轮构建失败（strings 未用 import）与一轮测试失败（转移失败返回零值导致调用方丢失原提案）均为本任务自引入,当轮修复;非基线失败。

## 4. 负测试（新增）

- 治理字段污染的候选 YAML（`governance_status`/`approved_by`）在 `NewCandidate` 出生即拒绝 —— 要求 2/DL-08 的结构性证明（Parse KnownFields 严格模式）。
- 不可解析候选 / 空候选 / 零证据提案 / 空 ID / 含空白 ID / 空提案人 / 词表外证据 kind / 空白引用串。
- 9 组非法生命周期转移（DRAFT 直接激活/否决/取代、PROPOSED 取代、ACTIVE 再激活/否决、两个终态再转移）—— 全部 `ErrInvalidTransition` 且原提案状态不变;Activate 空批准人拒绝;Submit 零证据拒绝。

## 5. 风险与不确定性

- StatusActive 的语义边界:本包记录治理事实,不构成运行时开关(DL-09) —— P3-07 落地激活提交时,若 store 侧（P3-02）需要额外的激活一致性证明（治理状态 ACTIVE ⇔ Git 面存在对应正典 YAML）,属其契约范围。
- `EvidenceRef` 引用串只验证形态（非空/无空白/无控制字符）,不验证所指事实存在 —— 存在性验证归消费方（P3-04 replay 的 evidence-bounded 纪律）。
- `Supersedes` 只是指针;前任提案的实际转移动作由 store/激活链发起,域层不隐式级联。

## 6. Decision Locks

未触碰、未绕过:DL-08（零治理字段进 Policy —— 结构性负测试）、DL-09（运行时只消费 pinned Git 内容 —— 域层零激活语义）、DL-10 边界（本包不做 replay）。hidden/ 未读（DL-17）。

## 7. Commit

由编排者提交(见 git log P3-01 提交信息)。
