# P4-03 — DeepSeek Harness integration spike

- **Phase:** P4
- **Kind:** spike
- **Wave:** W25
- **Complexity:** XL
- **Execution model:** HIGH
- **Depends on:** A0-05
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-13) — 裁决 **GO-WITH-CONDITIONS**：针对钉死版本 `@deepseek-ai/dsh@0.1.5-rc.1` 的 10 次活体探针（2 轮模型对话）证明无头进程外接缝可行 —— `--profile headless "<task>"` 一次性执行、stdout=最终答案、exit 0/1；cwd=进程 cwd（config dump `workspaceRoot: process.cwd()` + 活体 cwd 文件读取双证）；无凭证 fail-safe（`MISSING_CREDENTIAL`，exit 1，指名 `DEEPSEEK_API_KEY`）；SIGTERM kill→exit 143；`sdk`/`acp` stdio profile 注册在案但外部未文档化、TS SDK 未发布（npm 404）——现为协议发明，禁用。**无需 SDK 桥**：与 Runtime lock（`Tally Go → runtimes/dsh → external DSH process`）及已证 CodexRuntime `codex exec` 模式 1:1 对应（一次 Execute=一进程、cwd 钉死工作区、env 认证、kill 即取消）。
- P4-04 绑定条件：钉死精确版本；每运行 `DSH_HOME` 隔离（默认 `~/.dsh` 跨运行残留会话态）；强制 `DSH_TELEMETRY_MODE=DISABLED`（默认向 DeepSeek OTLP 端点发遥测）；kill-tree 加固（Windows Job Objects + teardown 测试，R-1）；诚实能力集（headless 不暴露 events/resume/structured-output/actual-model 观测 —— 适配器不得声明，R-3）；期望模型仅经生成的 `--patch` YAML 配置覆盖（无 CLI model flag）。风险 R-2（developer-preview 易变，`dsh run` 已移除；运行时 `--version` 身份探针）、R-4（npm 工件不可观测上游 commit hash）、R-5（内部 `session.v3.jsonl.zstd` 永不作为契约解析）。
- Verification：活体探针 + `bash tools/dsh-spike/probe-headless.sh` 可复现探针脚本 PASS（key 仅经 env 读取，构建期零依赖）；`git diff --check` 清洁。Full evidence: `spikes/P4-03_dsh_protocol_spike.md` + `reports/P4-03_deepseek_harness_integration_spike.md`。PRODUCTION_REACHABILITY: N/A(reason: spike；生产接线归 P4-04)。

## Goal

Establish the actual supported headless integration protocol/lifecycle for current DeepSeek Harness before production adapter code.


## Allowed / primary paths

- `docs/tasks/v1.4.1/spikes/**`
- `tools/**`


## Implementation requirements

1. Determine CLI/SDK/process protocol, cwd semantics, output/result, cancel/kill, provider/model config, auth shape.
2. Test keyless/local/fake route if possible.
3. Record upstream version/commit and breaking-change risk.
4. No production runtime adapter until GO verdict.


## Acceptance criteria

- [ ] Report returns GO / GO_WITH_LIMITS / NO_GO and a minimal protocol transcript.


## Verification

- Reproducible spike commands; process cleanup confirmed.

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

- **Execution:** `HIGH`
- **Review:** `HIGH`
- **Reason:** External runtime is developer-preview and rapidly changing; integration protocol must be verified against the exact version.
- **Review rule:** HIGH review required because the task touches an authority boundary, release gate, or high-blast-radius projection even when implementation is deterministic.

## Reference order — mandatory

Use evidence in this order. Do not jump directly to an external example before reading Tally's tests/contracts.
1. **Tests:** Tests/evidence first: build a keyless minimal subprocess probe and pin exact DSH version/commit
2. **Contract / Interface:** Interface target: Tally `agentruntime.Runtime`; inspect DSH headless/subagent/process seams only after pinning
3. **Core Implementation:** Core external source: DSH subagent registry, tool-subagent route selection, Codex backend ownership note
4. **Failure Handling:** Failure: verify cancellation/process teardown, cwd, provider/model selection on the exact installed build; developer-preview breakage is expected

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
- **EXT-DSH-CODEX — DeepSeek Harness Codex/Claude Code backend design note**  
  https://github.com/deepseek-ai/deepseek-harness/blob/master/.agents/notes/implemented/feature/2026-08-04-claude-code-and-codex-subagent-backends.md  
  License/version: MIT; master.  
  Use: Reference ownership boundaries for task text, cwd, cancellation, result settlement and process teardown.

## Do

- Implement only the runtime adapter seam; Tally keeps workspace, Git, verification and acceptance.
- Start by running the listed focused tests or the closest available subset.
- Make the smallest semantic change that satisfies this task; keep public surface area narrow.
- Add at least one negative/failure-path test for new behavior.

## Don't

- Do not add a second runtime lifecycle protocol or grant a runtime durable Git/acceptance authority.
- Do not copy external code wholesale. Extract the pattern and adapt it to Tally's existing interfaces and dependency whitelist.
- Do not add a new dependency unless `test/arch/deps_test.go` and the dependency policy are intentionally updated by an approved architecture change.

## High-model execution requirement

This task should be performed by a high-capability model because success requires semantic judgment rather than mechanical editing. The model must explain the chosen seam, enumerate rejected alternatives, and demonstrate that fail-safe behavior remains intact before coding.

## L3 completion evidence required from implementing agent

- exact source/tests read;
- any external reference actually used;
- why the chosen seam is the narrowest valid seam;
- changed files;
- exact test commands/results;
- negative tests added;
- known uncertainty or version sensitivity;
- confirmation that Decision Locks were not bypassed.
