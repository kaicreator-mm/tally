# P6-04 Execution Report — Migration false-negative reproduction

- **Status:** DONE(确定性复现成功,根因已判别)
- **Date:** 2026-09-12
- **Baseline:** HEAD `bb6db68`(master);并行 lane(P2-01/P4-01/P5-02)在途编辑未触碰
- **Kind:** spike — 根因诊断;**未修复任何语义**(修复归 P6-05)

## 0. 根因判别结论(verdict line)

**主类:fixture(实验组合层错位)+ 语义放大器:verifier semantics(exit-code 通道缺失)。**

- **触发器(fixture / trial tooling,P6-05 修复面在 `scripts/v14_trial/tally/compose.go`)**:migration 分支把冻结 gate key 原文当作探针子命令写进 binding argv(`compose.go` L278:`[]string{probePath, key, "-task", …}`),而探针 `scripts/v14_trial/tally/probe/main.go` 的 switch 只接受 `fresh` / `n1` / `integrity` —— 每条迁移 binding 对任意候选(包括字节级正确修复)确定性 `exit 2` + `unknown subcommand "fresh_database"`。bug_fix 分支显式映射了子命令(`bug_repro_failed_before → repro`,else → `after`),因此只有 db 任务全军覆没 —— 与实验 14/14 FAIL 仅出现在 db 任务的观测完全一致。
- **放大器(verifier semantics,修复面在 `internal/verification/orchestrator.go`)**:`executeVisibleBinding` 把**任何**非零退出码记为 `EvidenceStatusFail`(orchestrator.go L602-608:`st := EvidenceStatusFail; if raw.ExitCode == 0 { st = EvidenceStatusPass }`),连 `Err` 都不产生 —— 探针文档化的 infra 退出码 2("wiring/infra error, recorded, not fabricated")被伪造为**可信的候选 FAIL**,结论 `FinalFailed`(而非 needs_review)。这违反信任不变量「verifier 错误不得伪造为候选失败」(AGENTS.md §5)。
- **排除项**:非 DB lifecycle(探针在首次 DB 访问前即终止)、非 environment(DSN/sslmode 面对内部 verify 无害;judge 的 maintenance-URI query-string bug 是独立且已修复的问题,只影响 judge 的 NOT_RUN,不产生内部 FAIL)、非 oracle(oracle.json 判定本身正确)、非 timeout、非 setup/teardown、非 state leakage(每 binding 一次性 database,从未被触达)。

**决定性证据链**:

1. 保存的冻结契约(`/tmp/tally-v14-trial/tally/db-01-orders-total-backfill/repo/.tally/verifications/TASK-9106.yaml`,commit `7f40403`)逐字记录 `argv: [...v14tallyprobe.exe, fresh_database, ...]`(三条 binding 同形)。
2. 探针 `probe/main.go` L58-76:subcommand switch 无 `fresh_database` 分支 → `default: unknown subcommand → os.Exit(2)`。
3. 原始实验行(`docs/evals/v1.4/results/raw/tally/db-01-orders-total-backfill.json`):三次 attempt 的 `verification_lineage.status` 全部 `failed`(不是 error)——与「exit 2 → verdict fail → FinalFailed」的投影吻合;若 binding 无 verdict 会是 needs_review/`error` 行。
4. attempt 3 候选与冻结 reference repair 字节一致(agent_claim 自证 + judge acceptance),仍 FAIL —— 候选内容无关性得证。
5. 本任务新增的确定性复现(§3)在当前 HEAD 上按需重建 A(trigger)→ B(false negative)→ C(control 三绿)全链。

## 1. Changed files(全部在 Allowed Paths)

- `internal/verification/p604_migration_false_negative_repro_test.go`(新)— 单元级钉子:fakeRunner 逐字回放「exit 2 + unknown subcommand」观察,生产编排器对 trial 同形契约产出 `FinalFailed` + 一条 `verdict=fail` 伪事实。**钉住当前缺陷语义**,文件头注明 P6-05 修复后必须翻转期望并保留 exit-1 可信 FAIL 负例。
- `test/e2e/v141_p604_migration_false_negative_repro_test.go`(新,integration)— 端到端复现:编译真实 `scripts/v14_trial/tally/probe` 二进制,物化字节级正确 repair 候选 workspace,三步因果链:
  - **A**(无 DB 依赖):trial 组合形 argv → `exit 2` + `unknown subcommand "fresh_database"`;
  - **B**(无 DB 依赖):生产 `verification.Orchestrator` + `LocalRunner` + trial 形冻结契约 → `FinalFailed`,三 binding 全部 `(fail, exit 2)`;
  - **C**(需 PostgreSQL,不可达即显式 skip):同一候选、同一探针、同一 DSN 面,仅子命令换成 `fresh/n1/integrity` → 三条全绿 —— 即外部 judge 接受同一候选的等价证据面,证明候选本可验证。
- `docs/tasks/v1.4.1/reports/P6-04_migration_false_negative_reproduction.md`(本报告)。

零生产代码改动(诊断任务);`scripts/v14_trial/**` 只读(不在 Allowed Paths,错位修复归 P6-05 + PATH_OWNERSHIP 裁量)。

## 2. Semantic changes

None(纯诊断 + 复现测试)。复现测试对生产语义的断言 = 当前缺陷面的事实钉住,不是认可。

## 3. Tests / commands executed(exact)

| 命令 | 结果 |
|---|---|
| `go test ./internal/verification/ -run TestP604 -count=2` | PASS ×2(确定性,~1s/次) |
| `go test ./internal/verification/... -count=1` | PASS(全套,1.2s) |
| `DATABASE_URL=postgresql://postgres:postgres@localhost:15432/postgres go test -tags=integration ./test/e2e/ -run TestP604MigrationFalseNegativeReproE2E -count=1 -v` | **PASS ×2(44.5s / 44.1s,两次独立运行)**;A/B 确定性复现,C 对照在共享 PostgreSQL 上三绿 |
| `DATABASE_URL=… go test -tags=integration ./test/e2e/ -run "TestV14Migration" -count=1` | PASS(79.2s;既有迁移闭环 + 探针套件不回退) |
| `go vet -tags=integration ./test/e2e/` | PASS |
| `gofmt -l`(两个触碰文件) | clean |
| `git diff --check` | clean |
| 全套 integration / make check | NOT_RUN(orchestrator 规则:宽套件归 post-commit;Git Bash 无 make) |

## 4. Baseline failures observed

None。`internal/verification` 全套与既有 `TestV14Migration` 套件在新增文件在场时全 PASS。

## 5. Reproduction matrix

| 变量 | 取值 | 结果 |
|---|---|---|
| 候选内容 | 缺陷 0003 / repair 0003(字节级) | 无关 —— 两种均 `FinalFailed`(假阴性定义) |
| 子命令 | gate key 原文(trial 形) | exit 2,触发 |
| 子命令 | fresh/n1/integrity(探针契约) | 三绿(C 对照) |
| DB 可达性 | 可达 / 不可达 | A+B 不依赖 DB(探针先于任何连接终止);C skip 并显式记录 |
| 重复运行 | ×2(单元与 e2e 各两次) | 逐字节一致 |

## 6. Remaining risks / P6-05 行动面

- **修复面 1(fixture,先修)**:`scripts/v14_trial/tally/compose.go` migration 分支把 gate key 映射到探针子命令(`fresh_database→fresh`,`n_minus_1_upgrade→n1`,`data_integrity→integrity`)。注意 `scripts/**` 不在 P6-05 的 Allowed Paths(`internal/verification/**`、`test/**`、`docs/**`)—— 需 PATH_OWNERSHIP 裁量或 orchestrator 授权;不改它,重跑实验仍会 14/14 FAIL。
- **修复面 2(verifier semantics,生产面)**:`internal/verification/orchestrator.go` 为 visible binding 引入「可信 FAIL vs infra error」的退出码判别通道(探针已在 header 文档化 exit 0/1/2 契约)。最小缝候选:binding 冻结协议/CommandSpec 声明 infra 退出码集合(如 `{2}`),命集中 → `Err`(无 verdict,needs_review,不落伪事实);**不得**把 exit 1 一并抹成 error(钉子测试的负例守卫)。涉及冻结契约 schema 语义,P6-05 须按 Authority order 校验 PRD/Decision Locks(尤其 fail-safe 与 append-only 事实面)。
- 两个修复面**都**要动才能让「内部 verify vs 外部 judge」在重跑实验时一致;只修 compose.go 会让 exit-2 类 infra 故障继续伪装成可信 FAIL(fail-safe 缺口仍在)。
- e2e 复现的 B 面用内存 RunSink(同 orchestrator 单测口径),未驱动 runstore 落库行 —— 实验行上的 `verification_lineage.status=="failed"` 投影已由保存的原始 JSON 直接佐证,无需复刻落库。

## 7. Decision Locks

Not bypassed, none modified。未读/未挂载 hidden(§3 lock);未改任何共享热点;DECISION_LOCKS/PRD/STATUS/TASK_DAG 等零触碰;DEEPSEEK_API_KEY 未使用(复现无需真实模型调用)。

## 8. Completion report(task contract format)

1. **status:** DONE
2. **changed files:** §1(两个新测试文件 + 本报告)
3. **semantic changes:** none(§2)
4. **tests/commands:** §3(全部 PASS/NOT_RUN 如实记录)
5. **baseline failures:** none(§4)
6. **remaining risks:** §6(P6-05 双修复面 + scripts/** 路径授权)
7. **Decision Locks touched:** none(§7)
8. **commit:** not committed(orchestration 规则:本 agent 非 committer)

## 9. L3 completion evidence

- **exact source/tests read:** 任务文件全文;`docs/tasks/v1.4.1/DECISION_LOCKS.md` 相关行、`docs/tasks/v1.4/HANDOFF.md`(follow-up #1)、`docs/evals/v1.4/results/raw/tally/db-01-orders-total-backfill.json` 全文、`docs/evals/v1.4/results/raw/tally/` 清单;保存的 trial 工件(`/tmp/tally-v14-trial/tally/db-01-…/{repo/.tally/verifications/TASK-9106.yaml, events.jsonl, judge-result*.json}`);`scripts/v14_trial/tally/{trialrun.go, compose.go, probe/main.go, probe/pg.go, probe/exec.go}`、`scripts/v14_trial/bare/judge.go`(子命令面)、`scripts/v14_trial/tally/psqlsh`;`internal/candidateverify/{service.go, holdout.go}`;`internal/verification/{orchestrator.go, orchestrator_test.go, visible_executor_test.go, freeze.go, types.go}`;`test/e2e/{v14_migration_closedloop_test.go, v14_migration_harness_test.go, v14_migration_probe/main.go}`;`test/eval/v14_product/testdata/tasks/db-01-…/{oracle/oracle.json, acceptance.yaml}`;`internal/changeset/types.go`;P1-08 报告(结构对齐)。
- **external reference used:** none(L3 Pack 明示不需要)。
- **narrowest seam 论证:** 诊断任务的合法最小缝 = 复现工件本身:一个包内单元钉子(fakeRunner 回放,零新依赖、零生产改动)+ 一个 integration 复现(真实探针 + 生产编排器 + 冻结夹具字节)。未发明平行诊断子系统,未触碰 held-out/守卫语义。
- **negative tests added:** 复现本身就是负路径(exit 2 → 伪 FAIL 钉住);e2e C 面构成正例对照;P6-05 需补的「exit 1 仍是可信 FAIL」负例已在钉子测试注释中显式预留。
- **known uncertainty / version sensitivity:** 复现依赖 `scripts/v14_trial/tally/probe` 的子命令面与 exit-code 契约(两者都在 trial 工具内,HEAD 上未变);保存的 `/tmp` trial 工件为本地未提交证据(报告引用其内容与路径,重装机器后以仓库内原始 JSON + 源码即可独立重建同一结论)。
- **Decision Locks not bypassed:** 确认(§7)。
