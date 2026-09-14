# P4-07 — DeepSeek limited-readiness gate

- **Phase:** P4
- **Kind:** gate
- **Wave:** W29
- **Complexity:** XL
- **Execution model:** LOW
- **Depends on:** P4-02, P4-05, P4-06
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-13) — **裁决: LIMITED_READY**(P4 lane 至此全部落地;有限范围 release:DSH 仅 canary/bounded-task 可用,见 docs/guides/dsh-canary.md)。门禁 `test/e2e/v141_p407_dsh_limited_readiness_gate_test.go::TestP407DSHLimitedReadinessGate`(真实临时 Git 仓库 + 真实进程树,替身 launcher 现场构建,确定性无钥,双跑 PASS 2.1s):有界任务(headless 读取任务输入确定性完成 RunCompleted)→ **候选 ChangeSet 经既有闭环桥**返回(changeset.New workspace 差量,身份非空)→ **零直接 protected/durable Git 权限**(门禁内无任何 Git 写经 runtime,DL-09)→ **取消清理通过**(长运行任务 Cancel → RunCancelled + 整树终止后子进程零残留,真实进程取证)→ 隔离面(cwd 钉 workspace / per-session DSH_HOME / 遥测禁用)→ 负面权限(ResumeOf/ModelProfile 能力协商拒绝)。**安全假设与上游 developer-preview 状态一致**(AC4):钉版闸使漂移显式失败,能力诚实,遥测强制禁用。
- 显式限制(LIMITED_READY 边界):① live 内容产出(真钥+真二进制的模型真实写文件)未在门禁内验证 —— 可复现无钥替身承担门禁,live 证据由 P4-03 spike 活体探针承担(10 探针含 2 轮真实模型对话);② 能力面受限(R-3);③ 上游易变(R-2)。升级条件见报告 §4。
- 验证:`-tags=integration ./test/e2e/ -run TestP407DSHLimitedReadinessGate -count=1 -v` PASS 2.12s + 双跑确定性;arch/build/vet/gofmt/`git diff --check` 清洁。三轮开发期夹具修正(Windows .exe 后缀/workspace bootstrap 提交/具体类型断言)当轮修复,非基线失败。执行过程:平台模型请求故障彻夜持续,编排者主会话完成。Full evidence: `reports/P4-07_deepseek_limited_readiness_gate.md`。


## Goal

Run a bounded real-repository candidate flow through DSH under Tally-owned isolation and issue LIMITED_READY or BLOCKED.


## Allowed / primary paths

- `test/**`
- `docs/tasks/v1.4.1/reports/**`


## Implementation requirements

1. Implement only the minimum change required by this task and preserve all v1.4 invariants.


## Acceptance criteria

- [ ] Candidate ChangeSet returns through existing Closed Loop bridge.
- [ ] No direct protected/durable Git authority.
- [ ] Cancellation cleanup passes.
- [ ] Security assumptions match upstream developer-preview status.


## Verification

- Real bounded task + negative permission/cancel tests.

## Mandatory implementation discipline

- Read `../../DECISION_LOCKS.md`, `../../PRD_AMENDMENTS.md`, and `../../SOURCE_BASELINE.md` first.
- Do not broaden scope to “clean up” adjacent architecture.
- Do not weaken fail-safe behavior to make tests pass.
- Preserve source-commit identity and held-out verification boundaries.
- If an expected seam does not exist, stop this task as `BLOCKED_BY_SOURCE_FACT` and report the exact source evidence instead of inventing a parallel subsystem.
- New behavior needs focused tests.
- Existing baseline failures must be distinguished from introduced failures.

## Completion report

Report exactly:

1. status: `DONE | BLOCKED | PARTIAL`;
2. changed files;
3. semantic changes;
4. tests/commands executed with exact results;
5. baseline failures observed;
6. remaining risks;
7. whether any Decision Lock was touched;
8. commit hash (if committed).

---

# L3 Task-Level Implementation Reference Pack

## Model assignment

- **Execution:** `LOW`
- **Review:** `HIGH`
- **Reason:** The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred.
- **Review rule:** HIGH review required because the task touches an authority boundary, release gate, or high-blast-radius projection even when implementation is deterministic.

## Reference order — mandatory

Use evidence in this order. Do not jump directly to an external example before reading Tally's tests/contracts.
1. **Tests:** Tests: generic agentruntime conformance + DSH lifecycle tests + real bounded repo scenario
2. **Contract / Interface:** Contract: DSH remains candidate producer under Tally workspace/verification/acceptance
3. **Core Implementation:** Core: one primary runtime owner
4. **Failure Handling:** Failure: gate LIMITED_READY if any cancellation, isolation or version-pinning uncertainty remains

## External reference pack
- **EXT-DSH-SAFETY — DeepSeek Harness SAFETY**  
  https://github.com/deepseek-ai/deepseek-harness/blob/master/SAFETY.md  
  License/version: MIT repository; master; developer preview.  
  Use: Hard constraint: do not treat DSH as sole security control; least privilege and disposable isolation.
- **EXT-DSH-SUBAGENT — DeepSeek Harness subagent seam**  
  https://github.com/deepseek-ai/deepseek-harness/blob/master/docs/subsystems/subagent.md  
  License/version: MIT; master; pin commit.  
  Use: Named provider registry, capability preflight, child lifecycle and result semantics.
- **EXT-DSH-TOOL — DeepSeek Harness tool-subagent implementation**  
  https://github.com/deepseek-ai/deepseek-harness/blob/master/packages/subagent/tool-subagent/src/index.ts  
  License/version: MIT; master; pin commit.  
  Use: Current provider/model/reasoning selection surface. Must verify exact installed build because developer-preview behavior has changed.
- **EXT-GO-EXEC — Go os/exec**  
  https://pkg.go.dev/os/exec  
  License/version: BSD-3-Clause (Go project); Go standard library.  
  Use: Subprocess context cancellation, Wait/Process lifecycle and command execution semantics.

## Do

- Implement only the runtime adapter seam; Tally keeps workspace, Git, verification and acceptance.
- Start by running the listed focused tests or the closest available subset.
- Make the smallest semantic change that satisfies this task; keep public surface area narrow.
- Add at least one negative/failure-path test for new behavior.

## Don't

- Do not add a second runtime lifecycle protocol or grant a runtime durable Git/acceptance authority.
- Do not copy external code wholesale. Extract the pattern and adapt it to Tally's existing interfaces and dependency whitelist.
- Do not add a new dependency unless `test/arch/deps_test.go` and the dependency policy are intentionally updated by an approved architecture change.

## Low-cost model execution recipe

1. Read the exact tests listed above.
2. Read the interface/types used by those tests.
3. Read only the nearest implementation seam, not the whole repository.
4. Implement one bounded change.
5. Run focused tests.
6. Run relevant `test/arch` guards.
7. If expected source facts differ from this pack, stop with `BLOCKED_BY_SOURCE_FACT`; do not redesign the architecture.

## L3 completion evidence required from implementing agent

- exact source/tests read;
- any external reference actually used;
- why the chosen seam is the narrowest valid seam;
- changed files;
- exact test commands/results;
- negative tests added;
- known uncertainty or version sensitivity;
- confirmation that Decision Locks were not bypassed.
