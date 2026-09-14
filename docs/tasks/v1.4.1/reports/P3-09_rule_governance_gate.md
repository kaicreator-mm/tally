# P3-09 — Rule-governance gate

- **执行者**: 编排者主会话(平台模型请求故障彻夜持续,子智能体派发连续 15 次失败;按既定先例接管)
- **日期**: 2026-09-13
- **基线**: master `8a91e1c`(P3-04 之后)

## 1. 门禁裁决

**PASS** —— 四条 AC 全部在真实临时内容仓库 + 真实治理事件库上端到端证明(P3-01..P3-08 全部交付面原样组合,零生产代码改动;门禁路径纪律 `test/**` + 本报告)。

## 2. 被测链(P3 E2E:proposal → replay → human decision → GitWriter → reload)

`test/e2e/v141_p309_rule_governance_gate_test.go::TestP309RuleGovernanceGate`(DB-gated,8.1s,双跑确定性 PASS):

1. **提案**:`NewCandidate` 出生正典校验 + draft/submit 事件(submit 必须携带证据引用,真实治理库每测试独立 schema)。
2. **重放**:`ReplayCandidate`(P3-04)—— 候选命中 schema_migration 录得轴;录得键 `fresh_database=passed` → `would_have_passed`;`n1_upgrade` 无录得 → `insufficient_evidence` **一等输出**(AC2,DL-10)。
3. **人本批准 → GitWriter → 重载**(P3-07 服务):期望基线 CAS 提交正典 YAML → `artifacts.LoadPolicies(HEAD)` 观察到同 id 同修订策略 → 治理面 ACTIVE + approved_by(AC1)。
4. **分裂脑刻意注入**:第二个提案只落治理 ACTIVE 事件、不做 Git 提交 → 治理面记 ACTIVE,而 HEAD 快照观察不到该策略(AC1 对立面:运行时权威跟随 Git,治理面不是第二权威,DL-09/BI-01)。
5. **陈旧激活**(AC4):第三个提案基于过期基线激活 → 类型化拒绝、治理面保持 PROPOSED、零激活事件。
6. **治理字段**(AC3):携带 `governance_status` 的策略制品经运行时装载器整体拒绝(进入 `Rejected`,永不进生效集),同快照的合法策略不受影响;开发期一轮失败为投毒夹具与合法策略同 id 触发了「同 id 歧义整体拒绝」语义(另一条正确的守卫)—— 投毒夹具改用独立 id 使 AC3 断言精确。
7. **P3-08 复用确认**:评审页 propose-rule 动作写入的就是本链消费的治理事件面(同一 append-only 存储)。

## 3. 验证命令

| 命令 | 结果 |
| --- | --- |
| `DATABASE_URL=… go test -tags=integration ./test/e2e/ -run 'TestP309RuleGovernanceGate' -count=1 -v` | PASS 8.14s |
| 复跑 `-count=1` | PASS(确定性) |
| `go test ./test/arch/... -count=1` | PASS(P3 lane 既有守卫:v14 运行时边界/策略治理字段排除/Outcome kind 集) |
| `go build ./...` / `gofmt -l` / `git diff --check` | PASS / 清洁 |

## 4. 基线失败观察

两轮开发期失败均为门禁测试自身夹具问题(同 id 歧义语义;一处断言方向写反),当轮修正;非基线失败,P3 lane 全部既有测试不变绿。

## 5. 风险

- 分裂脑注入采用「治理事件先行、Git 缺席」方向;反向(治理面缺记录、Git 已生效)由 P3-07 报告风险 §5 文档化(激活事件落库失败的类型化错误携带生效 sha),两者合起来覆盖 split-brain 的两个方向,其中 Git 永远是运行时权威。
- 治理库宕机时的激活第 4 步故障注入夹具仍未构造(需中途故障注入设施)—— 留待后续演练,P3-09 门禁以 AC1/AC3 的权威面判定为准。

## 6. Decision Locks

未触碰、未绕过:DL-09(Git 快照保持运行时权威 —— AC1 分裂脑注入的结构性证明)、DL-10(insufficient_evidence 一等 —— AC2)、DL-08(治理字段混入即拒 —— AC3)、DL-03(唯一写边界 gitwriter)、BI-01(approved_by 落治理事件)。hidden/ 未读(DL-17)。

## 7. Commit

由编排者提交(见 git log P3-09 提交信息)。
