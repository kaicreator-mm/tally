# P6-05 Execution Report — Fix migration verification root cause

- **Status:** DONE(双修复面落地,真负例语义保持,e2e 回归 ×2 确定性 PASS)
- **Date:** 2026-09-12
- **Baseline:** HEAD `33197fe`(master);并行 lane(P2-02/P6-01/P4-04/P5-03/P6-06)在途编辑未触碰
- **Depends on:** P6-04(commit `8a98a2c` 的根因判别,本报告按其证据链实现)

## 0. 修复内容(verdict line)

**按 P6-04 判别结论落地双修复面:触发器(组合层映射)+ 放大器(infra 退出码通道);exit-1 可信 FAIL 语义原样保持。**

- **修复面 1(触发器,`scripts/v14_trial/tally/compose.go` — ORCHESTRATOR AUTHORIZATION)**:本路径不在任务文件列出的 Allowed Paths,由 orchestrator 在任务派发中**显式授权最小修复**("map gate keys → probe subcommands",已记录于 P6-04 commit message 与 STATUS.md)。新增 `migrationProbeSubcommands` 全集映射(`fresh_database→fresh`、`n_minus_1_upgrade→n1`、`data_integrity→integrity`),migration 分支的 binding argv 改用映射后的子命令;未映射 key 立即报错(frozen-set 漂移 fail loudly),组合层再也无法静默产出不可执行的 binding 命令。**未触碰 scripts/ 下任何其他文件**(closedloop.LoopConfig 组合属 P1-08 exception/P4-07 owner)。
- **修复面 2(放大器,`internal/verification`)**:
  - `types.go`:`CommandSpec` 新增 `InfraExitCodes []int`(yaml `infra_exit_codes`,omitempty)—— 冻结协议声明的「文档化 wiring/infra 错误」退出码通道。`Validate` 强制只能声明 **>= 2** 的退出码(0=pass、1=canonical trusted FAIL 不可重定义为 infra,重复声明拒绝):判定语义在类型层不可被削弱。`cloneCommandSpec` 深拷贝该切片(冻结隔离)。
  - `orchestrator.go` `executeVisibleBinding`:命中声明的 infra 退出码 → `Err`(无 verdict)、不 `session.Record`、不落事实 → bindingErr → **FinalNeedsReview**;未命中退出码语义不变(0=pass,其余含 1=可信 FAIL)。`executeGates` 同通道:命中 → `VisibleStatusError`(needs_review),不伪造 required-gate FAIL。该通道为 opt-in(不声明 `InfraExitCodes` 的既有契约行为逐字节不变)。
  - `compose.go` 两个分支(bug_fix 与 migration)的组合协议都声明 `InfraExitCodes: [2]`(探针 main.go 头注文档化的 exit 0/1/2 契约),使重跑 v1.4 实验时「内部 verify vs 外部 judge」在两个修复面上同时一致。
- **保持不变(负例守卫)**:exit 1 → verdict=fail → FinalFailed + 恰一条 fail 事实;既有 fail-safe(unknown/unmatched 风险取最强面、无 verdict 执行不落库、append-only)全部未动。

## 1. Changed files(全部在 Allowed Paths + 单点授权的 compose.go)

- `internal/verification/types.go` — CommandSpec.InfraExitCodes 通道 + 校验 + 深拷贝(修复面 2)。
- `internal/verification/orchestrator.go` — executeVisibleBinding / executeGates 消费 infra 通道(修复面 2)。
- `scripts/v14_trial/tally/compose.go` — gate key → 探针子命令映射 + 协议声明 infra {2}(修复面 1;orchestrator 授权的最小修复,未触碰 scripts/ 其他文件)。
- `internal/verification/p604_migration_false_negative_repro_test.go` — P6-04 钉子测试**翻转为修复后语义** + 新增两个守卫(详见 §2 测试语义变更)。
- `test/e2e/v141_p604_migration_false_negative_repro_test.go` — e2e 复现演化为修复后因果面 A+B1+B2+C(详见 §2)。
- `docs/tasks/v1.4.1/reports/P6-05_fix_migration_verification_root_cause.md`(本报告)。

零改动:internal/closedloop/**、cmd/**、internal/outcome/**、internal/web/**、runtimes/**、migrations、sqlc 生成物、DECISION_LOCKS/PRD/STATUS/TASK_DAG 等治理文件。

## 2. 测试语义变更(§16:逐条声明,无静默削弱)

| 测试 | P6-04 语义(缺陷钉住) | P6-05 语义(修复后) | 依据 |
|---|---|---|---|
| 单元 `TestP604InfraExitCode2IsFabricatedAsTrustedFail` | exit 2 → fail 事实 + FinalFailed(钉住缺陷) | 更名 `TestP604InfraExitCode2YieldsNeedsReviewNotFabricatedFail`:exit 2(协议声明 {2})→ Err 含 infra、FinalNeedsReview、**零落库事实** | 信任不变量「verifier 错误不得伪造为候选失败」(AGENTS.md §5);P6-04 报告 §6 预留的翻转点 |
| 单元(新增)`TestP604TrustedExitCode1IsStillAFail` | 不存在 | exit 1 → verdict=fail + FinalFailed + 恰一条 fail 事实(**负例守卫:修复不得抹掉真负例**) | P6-04 钉子文件头显式预留的负例 |
| 单元(新增)`TestP604InfraExitCodesCannotRedefinedPassOrFail` | 不存在 | 声明 {0}/{1}/{-1}/重复 → Validate 拒绝(**结构守卫:0/1 不可重定义为 infra**) | 削弱阻断的类型层落点 |
| e2e B 面 | trial 形契约(无 infra 声明)→ 三条 (fail,2)+ FinalFailed | 拆为 B1(同形协议 + infra {2} → 三条 Err + needs_review + 零事实)与 B2(合法子命令 fresh + 不可达 DSN → 真实探针按自身契约 exit 1 → fail + FinalFailed + 每条一事实) | B2 是**经真实探针二进制**的 exit-1 可信 FAIL 端到端负例 |
| e2e A / C 面 | 触发器复现 / 三绿对照 | 语义不变:A 仍钉「gate key 原文 → exit 2」(探针子命令契约**未**放宽,修复在组合层);C 子命令 = `p604ProbeKeysToSubcommands` 映射右侧值,与 compose.go 新映射同一表(映射面钉住) | 探针契约是修复的依据而非修改对象 |

新增负路径测试(P6-04 报告 §9 要求):B2 + 单元 exit-1 守卫 + Validate 拒绝面。

## 3. Tests / commands executed(exact)

| 命令 | 结果 |
|---|---|
| `go build ./scripts/...` | PASS(compose.go 编译) |
| `go vet ./internal/verification/` | PASS |
| `go vet -tags=integration ./test/e2e/` | PASS |
| `go test ./internal/verification/... -count=1` | PASS(全套 1.2s,无回退) |
| `go test ./internal/verification/ -run TestP604 -count=2` | PASS ×2(确定性) |
| `DATABASE_URL=postgresql://postgres:postgres@localhost:15432/postgres go test -tags=integration ./test/e2e/ -run 'TestP604\|TestV14Migration' -count=1 -v` | **PASS**(TestP604 43.4s;TestV14Migration×2 = 37.0s + 40.6s;total 121.3s;C 对照在共享 PostgreSQL 上三绿) |
| 同上 `-run TestP604` 第二次独立运行 | PASS ×2(44.0s,确定性) |
| `git diff --check` | clean |
| `gofmt -l`(触碰文件) | 内容格式 clean;`gofmt -l` 因工作树 CRLF(core.autocrlf=true)列出厂内全部文件 —— 预先存在的仓库级现象,`.gitattributes`(eol=lf)在提交时归一,未做整文件行尾归一(§16) |
| 全套 integration / make check | NOT_RUN(orchestrator 规则:宽套件归 post-commit;Git Bash 无 make) |

## 4. Baseline failures observed

None。`internal/verification` 全套与既有 `TestV14Migration` 套件在本任务改动在场时全 PASS。

## 5. 三个 verdict 面如何映射 P6-04 证据链

1. **触发器修复 ↔ P6-04 证据链 1+2**:保存的冻结契约 TASK-9106.yaml 逐字记录 `argv: [...probe.exe, fresh_database, ...]`;probe/main.go switch 无该分支 → default exit 2。compose.go 的映射使重跑实验的冻结协议 argv 子命令 = fresh/n1/integrity,触发器消除;e2e C(映射右侧值)三绿 = P6-04 C 对照的等价证据面。
2. **放大器修复 ↔ P6-04 证据链 3+§0 放大器段**:实验行 `verification_lineage.status == "failed"`(不是 error)源于 orchestrator L602-608 把 exit 2 记为 `EvidenceStatusFail`。infra 通道使同类观察落为 Err/needs_review/零事实 —— 与「binding 无 verdict 会是 needs_review/error 行」的正确投影吻合,不再产生伪 fail 事实(单测钉住 sink.facts 为空)。
3. **保持 exit-1 可信 FAIL ↔ P6-04 §6 负例守卫要求**:「不得把 exit 1 一并抹成 error」。单元守卫(fakeRunner exit 1)+ e2e B2(真实探针 fresh + 拒连 DSN → exit 1)双重钉住 fail/FinalFailed/fail 事实;Validate 的 >= 2 约束使该语义在冻结协议层不可被重定义削弱。

## 6. Remaining risks

- 冻结契约 schema 新增 `infra_exit_codes`(omitempty、可选):既有契约解析/哈希不受影响(不声明则字段缺省、行为逐字节不变);但**新字段的消费目前只在 orchestrator 的 visible binding 与 gate 执行面**,若未来有别的 CommandSpec 消费者(如 holdout 执行面)需要同一通道,须由其 owner 任务另行接线 —— 本任务按最小缝不扩大。
- 重跑完整 v1.4 实验(14 迁移 binding)验证两修复面联合效果,归实验 owner(P6-06/后续 trial 重跑),不在本任务验证面;e2e 已在单任务(db-01)上证明联合因果面。
- `gofmt -l` 的 CRLF 现象为仓库工作树级(见 §3),提交时由 `.gitattributes` 归一 —— 提醒 committer 勿混入行尾归一 pass(§16)。

## 7. Decision Locks

Not bypassed, none modified。DECISION_LOCKS/PRD/STATUS/TASK_DAG/TASK_MANIFEST 等零触碰;hidden validation pack 未搜索/未挂载;共享热点零触碰;`scripts/v14_trial/tally/compose.go` 的改动为 orchestrator 显式授权的最小修复(授权已在 P6-04 commit message 与 STATUS.md 记录,本报告 §0 引用);DEEPSEEK_API_KEY 未使用。

## 8. Completion report(task contract format)

1. **status:** DONE
2. **changed files:** §1
3. **semantic changes:** §0(infra 退出码通道 + gate key 映射;exit-1/0 语义与既有 fail-safe 不变)
4. **tests/commands:** §3(全部 PASS / NOT_RUN 如实记录)
5. **baseline failures:** none(§4)
6. **remaining risks:** §6
7. **Decision Locks touched:** none(§7)
8. **commit:** not committed(orchestration 规则:本 agent 非 committer)

## 9. L3 completion evidence

- **exact source/tests read:** 任务文件全文(含 L3 Pack);AGENTS.md(信任不变量 §5/§16);P6-04 报告全文;`internal/verification/{orchestrator.go, types.go, freeze.go, orchestrator_test.go(helper 面)}`;`scripts/v14_trial/tally/{compose.go, probe/main.go}`;P6-04 的两个复现测试文件(本任务演化)。
- **external reference used:** none(L3 Pack 明示不需要;探针 exit-code 契约以 probe/main.go 头注为权威)。
- **narrowest seam 论证:** 放大器的最窄缝 = 冻结协议已有的 CommandSpec 上新增**可选、强校验(>= 2)** 的文档化 infra 退出码声明,消费点仅在判定退出码语义的两处(executeVisibleBinding / executeGates);未发明平行错误通道、未改结论判定顺序、未动 append-only 事实面。被拒替代方案:(a) 按退出码数值硬编码 exit 2 → error —— 把探针私有契约烧进生产编排器,且无法拒绝其他工具的 2 含义;(b) 任何非零退出码产生 stderr 关键字嗅探 —— 字符串匹配域错误,AGENTS.md §11 明令避免;(c) 只修 compose.go —— P6-04 报告 §6 已论证 fail-safe 缺口仍在。
- **negative tests added:** 单元 exit-1 可信 FAIL 守卫;单元 Validate 拒绝 {0,1,-1,重复};e2e B2(真实探针 exit 1 → FinalFailed)。
- **known uncertainty / version sensitivity:** 通道效果依赖冻结协议**声明** infra 退出码 —— 未声明的旧契约里 exit 2 仍按原语义记为可信 FAIL(这是有意的:语义改写必须经新 revision 显式冻结,INV-VER-02);trial 组合层已声明 {2},与探针文档契约一致。
- **Decision Locks not bypassed:** 确认(§7)。
