//go:build integration

// P2-10 harness:FakeRuntime 闭环 E2E 矩阵的测试等价组合(test-equivalent
// composition)。cmd/worker 的 v14 路径是 package main,无法被导入;本文件按
// cmd/worker/v14.go runV14Task 的生产组成逐件复刻装配面 —— 全部域件都是生产
// 组件本身,测试只提供 FakeRuntime 的确定性脚本与候选落点:
//
//	Git HEAD 快照装载(artifacts.Load,inv-art-01)
//	→ workspace.New 一次性 worktree 根 + RunTable(生产 workspace 管理)
//	→ ContextPack(contextpack.Select,与 worker v13/v14 同一规则)
//	→ workflow.NewRunStoreSink(codegen 事实面 → PostgreSQL append-only)
//	→ candidateverify.New(生产候选验证服务:一次性验证 workspace + 冻结协议
//	  编排 + verification_run/evidence 落库)
//	→ closedloop.NewStoreLineageSink / NewStoreExposureSink(取代/暴露事实
//	  append-only 出口)
//	→ closedloop.NewLoop(P2-08 生产组合根:WorkspaceExecutor + ServiceVerifier
//	  + FailureRouter + FeedbackSource + RepairPolicy + P1-06 验收守卫)
//
// P1-08 编译权威对齐(执行前置序与 cmd/worker runV14Task 逐件一致):
//
//	artifacts.LoadBundle(P1-01 同源 bundle,DL-04 一次 revision 解析装齐
//	FrozenIntent/TaskContract/冻结 VerificationContract/策略快照)
//	→ bundle.CompileAuthority(P1-03 编译权威 = 执行的第一道门,先于任何
//	  runtime 会话与 workspace 创建;拒收策略/无效风险依据/来源不一致类型化
//	  上抛,不进闭环)
//	→ Compiled.Requires() 进 ContextPack(P1-05 Prevention,纳入 context_hash)
//	  与内层迭代指令的 Prevention 小节(与 worker v14Instruction 同渲染)
//	→ LoopConfig.Authority = Compiled(P1-06 验收守卫:FinalVerified 不再单独
//	  授权普通 Accept —— no-bypass 的 test/** 面)
//
// 与 cmd/worker 的差异(如实声明,均为测试脚本面而非被测行为):
//   - runtime 是 FakeRuntime(fake.New)的确定性脚本包装:每个 Execute 轮次
//     按脚本向会话的一次性 worktree 写入候选内容或注入 provider 失败;
//   - 内层 CodegenRunner 按 v14CodegenRunner 同型实现(本次测试允许路径内
//     复刻;其行为本体由 P2-09 的 cmd/worker 测试权威覆盖);
//   - 度量观测缝(measurement)缺席:用量观测归 P1-41/worker 测试,不在本
//     矩阵的裁决面上。
//
// 数据库不可达时 SKIP(不产出假失败)。
package e2e

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"tally/internal/agentruntime"
	"tally/internal/artifacts"
	"tally/internal/candidateverify"
	"tally/internal/changeset"
	"tally/internal/closedloop"
	"tally/internal/contextpack"
	"tally/internal/contract"
	"tally/internal/core"
	"tally/internal/evidence"
	"tally/internal/holdout"
	"tally/internal/policy"
	"tally/internal/runner"
	"tally/internal/runstore"
	"tally/internal/store"
	"tally/internal/taskcontract"
	"tally/internal/verification"
	"tally/internal/workflow"
	"tally/internal/workspace"
	"tally/runtimes/fake"
)

// ---- 常量与夹具标识 ----------------------------------------------------------

const (
	clProject = core.ProjectID("proj-p210")
	clTaskID  = "TASK-210"
	clAccID   = "AC-01"
	clVisID   = "VIS-1"
	clHoldID  = "HOLD-1"

	// clGreetingRel 是候选交付物(worker 脚本写入、visible 冻结命令 grep 的
	// 文件);base 里已提交,候选以修改形态进入快照。
	clGreetingRel = "internal/widget/greeting.go"

	// clPassToken 是冻结 visible 命令验收的行为 token:候选内容包含它才通过。
	clPassToken = "P210-GREETING-V2"

	// clHoldoutSecret 只存在于 held-out 内容(与 Tally 控制的 store 根、
	// 不透明 ref、内容摘要一起)构成泄漏扫描面:出现在实现可见通道即泄漏。
	clHoldoutSecret = "P210-HOLDOUT-SECRET-QUIET-HARBOR"

	clDSN = "postgres://postgres:test@localhost:5433/postgres?sslmode=disable"
)

// clContentBad / clContentGood 是脚本化候选内容:bad 不含验收 token(visible
// 冻结命令确定性 exit 1),good 含(exit 0)。内容驱动整条修复矩阵。
func clContentBad() string { return "package widget\n\nconst Greeting = \"v1\"\n" }
func clContentGood() string {
	return "package widget\n\nconst Greeting = \"v2\" // " + clPassToken + "\n"
}

const clProjectYAML = `project: proj-p210
entities:
  - id: widget
    type: go_package
    path: internal/widget
`

const clTaskYAML = `id: TASK-210
profile: standard
title: Closed-loop E2E fixture
objective: Deliver the widget greeting under internal/widget.
scope:
  allowed_paths:
    - internal/widget
acceptance:
  - id: AC-01
    statement: widget greeting delivers the frozen behavior.
    independence_required: none
`

// clVisibleVerificationYAML:单 visible binding,冻结命令 grep 候选内容里的
// 验收 token(离线、确定性;非零退出 = 可信 FAIL)。
const clVisibleVerificationYAML = `task_id: TASK-210
revision: 1
acceptances:
  - id: AC-01
    statement: widget greeting delivers the frozen behavior.
    independence_required: none
    evidence_bindings:
      - id: VIS-1
        authoring_mode: pre_commit
        visibility_policy: visible
        evidence:
          type: test
          path: internal/widget/greeting_test.go
        execution_protocol:
          repetitions: 1
          aggregation: all_pass
          command:
            argv: [git, grep, --no-index, -q, P210-GREETING-V2, internal/widget/greeting.go]
`

// clHeldVerificationYAML:visible(`go version`,恒过)+ held_out(grep
// holdout-input 是否包含验收 token;内容是否包含由夹具决定)双 binding。
// held-out FAIL 阻断验收(§8.8.1)并把结论推向 needs_review —— 实现侧自始
// 至终接触不到内容,只有 pass|fail 类别经 P2-04 投影可表达。
const clHeldVerificationYAML = `task_id: TASK-210
revision: 1
implementation_runtime_requirements:
  session_isolation: true
  held_out_unreachable: true
  conformance_class: verification-isolation-v1
acceptances:
  - id: AC-01
    statement: widget greeting delivers the frozen behavior.
    independence_required: none
    evidence_bindings:
      - id: VIS-1
        authoring_mode: pre_commit
        visibility_policy: visible
        evidence:
          type: test
          path: internal/widget/greeting_test.go
        execution_protocol:
          repetitions: 1
          aggregation: all_pass
          command:
            argv: [go, version]
      - id: HOLD-1
        authoring_mode: pre_commit
        visibility_policy: held_out
        evidence:
          type: test
          store: holdout
          ref: %s
        execution_protocol:
          repetitions: 1
          aggregation: all_pass
          command:
            argv: [git, grep, --no-index, -q, P210-GREETING-V2, holdout-input]
`

// ---- PostgreSQL(每测试独立 schema,store.TestDB)--------------------------

func clStore(t *testing.T) (*runstore.Store, *store.Store) {
	t.Helper()
	base := os.Getenv("DATABASE_URL")
	if base == "" {
		base = clDSN
	}
	if !e2eDBReachable(base) {
		t.Skipf("P2-10 端到端测试需要 PostgreSQL(%s 不可达),跳过", base)
	}
	t.Setenv("DATABASE_URL", base) // store.TestDB 读同一 DSN
	st := store.TestDB(t)
	return runstore.New(st.Queries()), st
}

// ---- git 固件(结构化 argv;与既有 e2e 同一隔离合同)------------------------

func clGit(t *testing.T, dir string, argv ...string) string {
	t.Helper()
	res, err := (runner.LocalRunner{}).Run(context.Background(), runner.Command{
		Argv: append([]string{"git"}, argv...),
		Dir:  dir,
		Env:  e2eGitEnv(),
	})
	if err != nil {
		t.Fatalf("git %v: %v", argv, err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("git %v: exit %d: %s", argv, res.ExitCode, res.Stderr)
	}
	return strings.TrimSpace(res.Stdout)
}

func clWriteFile(t *testing.T, repo, rel, content string) {
	t.Helper()
	p := filepath.Join(repo, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("建目录 %s: %v", rel, err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("写 %s: %v", rel, err)
	}
}

// clIntentYAML 渲染 TASK-210 的 FrozenIntent 制品文本(恰四字段冻结 schema,
// DL-06/P0-01;hints 非空时附 owner 确认的 risk_hints 原文)。同源 bundle
// 装载(P1-01 LoadBundle)要求同一任务身份的 FrozenIntent 在 pinned 提交里。
func clIntentYAML(taskType string, hints ...string) string {
	b := "task_id: " + clTaskID + "\ntask_type: " + taskType + "\nsource_ref: p2-10 e2e fixture issue\n"
	if len(hints) > 0 {
		b += "risk_hints:\n"
		for _, h := range hints {
			b += "- " + h + "\n"
		}
	}
	return b
}

// clNewRepo 建带 .tally 制品(含 FrozenIntent)与 base 文件的真 git 仓库
// (visible-only 契约;低风险 bug_fix 场景,intent 无风险提示)。
func clNewRepo(t *testing.T) string {
	return clNewFixtureRepo(t, "bug_fix", nil, nil)
}

// clNewFixtureRepo 是 clNewRepo 的参数化形态(P1-08 gate 场景):intent 的
// task_type/risk_hints 与 .tally/policies/ 下的策略制品(路径 → YAML)可配,
// 其余制品与 base 文件逐字相同。
func clNewFixtureRepo(t *testing.T, intentType string, hints []string, policies map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	clGit(t, dir, "init", "-b", "main")
	clGit(t, dir, "config", "user.name", "tally-p210-e2e")
	clGit(t, dir, "config", "user.email", "tally-p210-e2e@example.invalid")
	clWriteFile(t, dir, ".tally/project.yaml", clProjectYAML)
	clWriteFile(t, dir, ".tally/intents/"+clTaskID+".yaml", clIntentYAML(intentType, hints...))
	clWriteFile(t, dir, ".tally/tasks/"+clTaskID+".yaml", clTaskYAML)
	clWriteFile(t, dir, ".tally/verifications/"+clTaskID+".yaml", clVisibleVerificationYAML)
	for rel, content := range policies {
		clWriteFile(t, dir, rel, content)
	}
	clWriteFile(t, dir, clGreetingRel, "package widget\n\nconst Greeting = \"base\"\n")
	clWriteFile(t, dir, "internal/widget/greeting_test.go", "package widget\n")
	clGit(t, dir, "add", ".")
	clGit(t, dir, "commit", "-m", "p2-10 visible fixtures")
	return dir
}

// clNewHeldRepo 建含 held_out binding 的仓库:先在 Tally 控制的 store 根落
// held-out 内容取得不透明 ref,再把 ref 写进冻结契约 YAML。holdoutPass 决定
// held-out 冻结命令对内容的判定(内容含验收 token = pass;否则 fail)。
func clNewHeldRepo(t *testing.T, holdoutPass bool) (repo, holdoutRoot string) {
	t.Helper()
	repo = clNewRepo(t)
	holdoutRoot = filepath.Join(t.TempDir(), "holdout-store")
	// store 根必须解析在每个受管 root(repo / 一次性 worktree 根)之外;
	// 这里三者是互不相交的临时目录。
	probe, err := holdout.NewFilesystemStore(holdoutRoot, repo, filepath.Join(t.TempDir(), "wt"))
	if err != nil {
		t.Fatalf("holdout store: %v", err)
	}
	var b strings.Builder
	b.WriteString("P2-10 controlled held-out expectation; marker " + clHoldoutSecret + "\n")
	if holdoutPass {
		b.WriteString("the candidate must expose " + clPassToken + "\n")
	}
	ref, err := probe.Put(context.Background(), []byte(b.String()))
	if err != nil {
		t.Fatalf("Put held-out content: %v", err)
	}
	clWriteFile(t, repo, ".tally/verifications/"+clTaskID+".yaml",
		fmt.Sprintf(clHeldVerificationYAML, ref.String()))
	clGit(t, repo, "add", ".")
	clGit(t, repo, "commit", "-m", "p2-10 held-out fixtures")
	return repo, holdoutRoot
}

// clHoldoutLeakSurfaces 返回 held-out 泄漏扫描面:密标、store 根、不透明 ref、
// 内容摘要。任何一项出现在实现可见通道(编码指令)即泄漏。
func clHoldoutLeakSurfaces(t *testing.T, holdoutRoot string, content []byte) []string {
	t.Helper()
	digest := holdout.ContentDigest(content)
	refPrefix := "ho-"
	return []string{clHoldoutSecret, holdoutRoot, digest, refPrefix}
}

// ---- 配置 --------------------------------------------------------------------

// clConfig 是测试等价组合的配置面(字段语义与 v14Config.v13 一致)。
type clConfig struct {
	Repo         string
	WorktreeRoot string
	HoldoutRoot  string
	EnvName      string
	MaxRepair    int // 0 = 缺省 2(决策锁 23)
}

func clConfigFor(t *testing.T, repo string) clConfig {
	t.Helper()
	return clConfig{
		Repo:         repo,
		WorktreeRoot: filepath.Join(t.TempDir(), "wt"),
		EnvName:      "p210-e2e",
	}
}

// ---- 脚本化 FakeRuntime(确定性尝试脚本;claim 一律来自真 FakeRuntime)----

// clStep 是一次 Execute 轮次的脚本:write 非空则在会话的一次性 worktree 内
// 写入候选交付物;err 非空则模拟 provider 失败(Execute 返回 error,会话不
// 再可信);inspect 在写入前观察 workspace 与指令(修复基物化断言必须发生
// 在当下 —— worktree 随尝试结束销毁;断言用闭包捕获的 *testing.T)。
type clStep struct {
	write   string
	err     error
	inspect func(ws, instruction string)
}

// clScriptedRuntime 包装 FakeRuntime:每个 Start 产出一个会话,会话的每次
// Execute 弹出一条脚本,再委托真 FakeRuntime 产出 claim(事件/终态/用量)。
// runtime 归因与能力协商完全来自 FakeRuntime Descriptor(fake),控制流不按
// runtime 名字分支。
type clScriptedRuntime struct {
	inner agentruntime.Runtime

	mu     sync.Mutex
	steps  []clStep
	instr  []string
	starts int
}

func newClScriptedRuntime(steps ...clStep) *clScriptedRuntime {
	return &clScriptedRuntime{inner: fake.New(fake.Config{}), steps: steps}
}

func (r *clScriptedRuntime) Descriptor() agentruntime.RuntimeDescriptor { return r.inner.Descriptor() }

func (r *clScriptedRuntime) Start(ctx context.Context, req agentruntime.StartRequest) (agentruntime.Session, error) {
	r.mu.Lock()
	r.starts++
	r.mu.Unlock()
	s, err := r.inner.Start(ctx, req)
	if err != nil {
		return nil, err
	}
	return &clScriptedSession{rt: r, inner: s, ws: req.Workspace}, nil
}

// instructions 返回已观测到的迭代指令(按 Execute 顺序的拷贝)。
func (r *clScriptedRuntime) instructions() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.instr...)
}

// starts 返回已开启的 runtime 会话数(no-bypass 负例的观测面:编译权威
// 失败时必须为零 —— 任何会话开启都晚于编译成功,P1-04 执行前置序)。
func (r *clScriptedRuntime) startsCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.starts
}

type clScriptedSession struct {
	rt    *clScriptedRuntime
	inner agentruntime.Session
	ws    string
}

func (s *clScriptedSession) Execute(ctx context.Context, req agentruntime.ExecuteRequest) (agentruntime.AgentRunResult, error) {
	s.rt.mu.Lock()
	if len(s.rt.steps) == 0 {
		s.rt.mu.Unlock()
		return agentruntime.AgentRunResult{}, fmt.Errorf("clScriptedRuntime: 脚本耗尽")
	}
	step := s.rt.steps[0]
	s.rt.steps = s.rt.steps[1:]
	s.rt.instr = append(s.rt.instr, req.Instruction)
	s.rt.mu.Unlock()

	if step.inspect != nil {
		step.inspect(s.ws, req.Instruction)
	}
	if step.err != nil {
		return agentruntime.AgentRunResult{}, step.err
	}
	if step.write != "" {
		p := filepath.Join(s.ws, filepath.FromSlash(clGreetingRel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return agentruntime.AgentRunResult{}, err
		}
		if err := os.WriteFile(p, []byte(step.write), 0o644); err != nil {
			return agentruntime.AgentRunResult{}, err
		}
	}
	return s.inner.Execute(ctx, req)
}

func (s *clScriptedSession) Cancel(ctx context.Context, id agentruntime.RunID) error {
	return s.inner.Cancel(ctx, id)
}

func (s *clScriptedSession) Events(ctx context.Context) (agentruntime.EventStream, error) {
	return s.inner.Events(ctx)
}

func (s *clScriptedSession) Result(ctx context.Context, id agentruntime.RunID) (agentruntime.AgentRunResult, error) {
	return s.inner.Result(ctx, id)
}

func (s *clScriptedSession) Close(ctx context.Context) error { return s.inner.Close(ctx) }

// ---- 内层 CodegenRunner(v14CodegenRunner 同型;契约见 cmd/worker/v14.go)----

// clCodegenRunner 把一次外层尝试执行为一次全新 CodegenRun:runtime 会话绑定
// 本次尝试的一次性 worktree,attempt 推导的 RunID 是 CodegenRun 身份,内层
// 迭代预算来自 ExecutionPolicy.Retry(与外层 RepairPolicy 零耦合);候选固定
// 点按 worktree 事实(git status + 内容回读)收取,claim 只留档不作事实。
// expose 非空且本次是修复尝试时,在尝试开始经共享暴露账面记账一笔蓄意披露
// (P2-06:修复期间暴露的物理位置;锚点 = 冻结前候选)。
type clCodegenRunner struct {
	project  core.ProjectID
	task     taskcontract.TaskContract
	policy   policy.ExecutionPolicy
	profile  string
	runtime  agentruntime.Runtime
	model    string
	behavior string
	verifRev string
	packHash string
	// requires 是编译权威的执行前 requires 通知行(P1-05,与 worker
	// v14CodegenRunner 同型:Prevention 小节随每轮迭代告知实现权威)。
	requires []string
	sink     workflow.RunSink
	expose   func(ctx context.Context, req closedloop.CodegenRunRequest) error
	now      func() time.Time
}

func (r *clCodegenRunner) RunCodegen(ctx context.Context, req closedloop.CodegenRunRequest) (*changeset.ChangeSet, error) {
	runID := workflow.RunID(req.RunID)
	now := r.now
	if now == nil {
		now = time.Now
	}
	factCtx := context.WithoutCancel(ctx)

	if r.expose != nil && req.RepairBase != nil {
		if err := r.expose(ctx, req); err != nil {
			return nil, fmt.Errorf("cl: 蓄意披露记账: %w", err)
		}
	}

	desc := r.runtime.Descriptor()
	startReq := agentruntime.StartRequest{Workspace: req.WorkspacePath}
	if desc.Capabilities.Has(agentruntime.CapModelSelect) {
		startReq.ModelProfile = r.model
	}
	session, err := r.runtime.Start(ctx, startReq)
	if err != nil {
		return nil, fmt.Errorf("cl: %s 开启 runtime 会话: %w", req.RunID, err)
	}
	var observed agentruntime.ObservedIdentity
	if rep, ok := session.(agentruntime.IdentityReporter); ok {
		observed = rep.Identity()
	}

	capSnap, err := clCapabilitySnapshot(desc)
	if err != nil {
		_ = session.Close(context.WithoutCancel(ctx))
		return nil, fmt.Errorf("cl: %s 观测能力快照: %w", req.RunID, err)
	}
	runRef, err := r.sink.BeginRun(factCtx, workflow.RunBegin{
		ID:                   runID,
		EscalatedFrom:        workflow.RunID(req.ParentRun),
		Project:              r.project,
		TaskID:               r.task.ID,
		TaskRevision:         r.task.Hash,
		Role:                 "coding",
		Profile:              r.profile,
		RuntimeName:          desc.Name,
		RuntimeVersion:       desc.Version,
		Model:                r.model,
		BehaviorRevision:     r.behavior,
		ObservedIdentity:     observed,
		BaseRevision:         req.BaseRevision,
		WorkspaceID:          req.WorkspacePath,
		CapabilitySnapshot:   capSnap,
		StartedAt:            now(),
		VerificationRevision: r.verifRev,
		ContextPackHash:      r.packHash,
	})
	if err != nil {
		_ = session.Close(context.WithoutCancel(ctx))
		return nil, fmt.Errorf("cl: %s 记录开跑: %w", req.RunID, err)
	}
	if strings.TrimSpace(runRef) == "" {
		_ = session.Close(context.WithoutCancel(ctx))
		return nil, fmt.Errorf("cl: %s 的 sink 未返回持久化 Run 引用", req.RunID)
	}

	runCtx, cancelWall := context.WithTimeout(ctx, r.policy.Budgets.MaxWallTime)
	defer cancelWall()

	var (
		iterations int
		inTokens   int64
		outTokens  int64
		changeSet  *changeset.ChangeSet
		status     runstore.RunStatus
		why        string
		retErr     error
		lastClaim  agentruntime.AgentRunResult
	)

	for iterations < r.policy.Retry.MaxIterations {
		iterations++
		res, execErr := session.Execute(runCtx, agentruntime.ExecuteRequest{
			Instruction: clInstruction(r.task, iterations, req.Feedback, r.requires),
		})
		lastClaim = res
		inTokens += res.Usage.InputTokens
		outTokens += res.Usage.OutputTokens
		if pErr := r.sink.RecordProgress(factCtx, runID, workflow.RunProgress{
			Iterations:  int32(iterations),
			InputTokens: &inTokens, OutputTokens: &outTokens,
		}); pErr != nil {
			status, why, retErr = runstore.RunFailed, "record_progress_failed", pErr
			break
		}
		if ctx.Err() != nil {
			status, why, retErr = runstore.RunCancelled, "cancelled", ctx.Err()
			break
		}
		if runCtx.Err() != nil {
			status, why = runstore.RunFailed, "wall_time_budget_exhausted"
			break
		}
		if inTokens+outTokens > r.policy.Budgets.MaxTokens {
			status, why = runstore.RunFailed, "token_budget_exhausted"
			break
		}
		if execErr != nil {
			status, why, retErr = runstore.RunFailed, "runtime_error", execErr
			break
		}
		if res.Status != agentruntime.RunCompleted {
			continue
		}
		snap, ok, fixErr := clFixCandidateFromWorktree(ctx, req.WorkspacePath)
		if fixErr != nil {
			status, why, retErr = runstore.RunFailed, "fix_candidate_failed", fixErr
			break
		}
		if !ok {
			continue
		}
		cs, csErr := changeset.New(changeset.Spec{
			Project:           r.project,
			BaseRevision:      req.BaseRevision,
			CandidateRevision: snap.revision,
			Files:             snap.files,
			ProducerRun:       runRef,
			CreatedAt:         now(),
		})
		if csErr != nil {
			status, why, retErr = runstore.RunFailed, "changeset_failed", csErr
			break
		}
		if rErr := r.sink.RecordChangeSet(factCtx, runID, cs); rErr != nil {
			status, why, retErr = runstore.RunFailed, "record_changeset_failed", rErr
			break
		}
		changeSet = cs
		status = runstore.RunCompleted
		break
	}
	if status == "" {
		status, why = runstore.RunFailed, "iterations_exhausted"
	}

	if finErr := r.sink.FinishRun(factCtx, workflow.RunFinish{
		ID:              runID,
		Status:          status,
		Iterations:      int32(iterations),
		InputTokens:     &inTokens,
		OutputTokens:    &outTokens,
		ResultPatchHash: clChangeSetID(changeSet),
		AgentClaim:      lastClaim.Summary,
		FailureReason:   why,
		FinishedAt:      now(),
	}); finErr != nil && retErr == nil {
		_ = session.Close(context.WithoutCancel(ctx))
		return nil, fmt.Errorf("cl: %s 记录终态: %w", req.RunID, finErr)
	}
	_ = session.Close(context.WithoutCancel(ctx))
	if retErr != nil {
		return nil, retErr
	}
	return changeSet, nil
}

func clChangeSetID(cs *changeset.ChangeSet) string {
	if cs == nil {
		return ""
	}
	return string(cs.ID())
}

// clCandidateSnapshot 是从 worktree 事实收到的候选(等价 workflow 内核
// CandidateSnapshot 的最小投影)。
type clCandidateSnapshot struct {
	revision string
	files    []changeset.ChangedFile
}

// clFixCandidateFromWorktree 与 cmd/worker fixCandidateFromWorktree 同合同:
// 只读一次性 worktree(git status -z 机器格式,关 rename 检测,含全部未跟踪
// 文件),改动文件以改后完整内容收进快照;没有任何改动 = 无可固定候选。
func clFixCandidateFromWorktree(ctx context.Context, ws string) (clCandidateSnapshot, bool, error) {
	local := runner.LocalRunner{}
	res, err := local.Run(ctx, runner.Command{
		Argv: []string{"git", "-C", ws, "status", "--porcelain=v1", "-z", "--no-renames", "--untracked-files=all"},
		Env:  e2eGitEnv(),
	})
	if err != nil {
		return clCandidateSnapshot{}, false, fmt.Errorf("git status %s: %w", ws, err)
	}
	if res.ExitCode != 0 {
		return clCandidateSnapshot{}, false, fmt.Errorf("git status %s: %s", ws, strings.TrimSpace(res.Stderr))
	}
	var files []changeset.ChangedFile
	for _, rec := range strings.Split(res.Stdout, "\x00") {
		if rec == "" {
			continue
		}
		if len(rec) < 4 || rec[2] != ' ' {
			return clCandidateSnapshot{}, false, fmt.Errorf("cl: 无法解析 git status 记录 %q", rec)
		}
		code, path := rec[:2], rec[3:]
		if strings.Contains(code, "D") {
			return clCandidateSnapshot{}, false, fmt.Errorf("cl: 候选包含删除改动 %s,拒绝", path)
		}
		content, err := os.ReadFile(filepath.Join(ws, filepath.FromSlash(filepath.Clean(path))))
		if err != nil {
			return clCandidateSnapshot{}, false, fmt.Errorf("cl: 读取候选文件 %s: %w", path, err)
		}
		files = append(files, changeset.ChangedFile{Path: path, Content: string(content)})
	}
	if len(files) == 0 {
		return clCandidateSnapshot{}, false, nil
	}
	rev, err := local.Run(ctx, runner.Command{
		Argv: []string{"git", "-C", ws, "rev-parse", "HEAD"},
		Env:  e2eGitEnv(),
	})
	if err != nil || rev.ExitCode != 0 {
		return clCandidateSnapshot{}, false, fmt.Errorf("cl: 解析 worktree HEAD: %s", strings.TrimSpace(rev.Stderr))
	}
	return clCandidateSnapshot{revision: strings.TrimSpace(rev.Stdout), files: files}, true, nil
}

// clCapabilitySnapshot 把 runtime 自述能力编码成非空 JSON(观测面要求,与
// workflow 内核/worker 同形态)。
func clCapabilitySnapshot(d agentruntime.RuntimeDescriptor) (json.RawMessage, error) {
	caps := d.Capabilities.Sorted()
	names := make([]string, 0, len(caps))
	for _, c := range caps {
		names = append(names, string(c))
	}
	data, err := json.Marshal(struct {
		Name         string   `json:"name"`
		Version      string   `json:"version"`
		Capabilities []string `json:"capabilities"`
	}{Name: d.Name, Version: d.Version, Capabilities: names})
	if err != nil {
		return nil, fmt.Errorf("cl: 编码 runtime 能力快照: %w", err)
	}
	return data, nil
}

// clInstruction 与 cmd/worker v14Instruction 同渲染:合同目标 + 验收条款;
// requires 非空时附编译权威的防御上下文小节(P1-05 Prevention:每轮迭代都
// 被告知最终验收条件;行内容只来自命中策略的公开声明,held-out 验证材料不在
// 渲染输入里);修复尝试只附 P2-04 安全投影(held-out 的内容/定位/摘要在投影
// 类型上不存在,本渲染无从泄漏)。
func clInstruction(task taskcontract.TaskContract, iteration int, fb *closedloop.RepairFeedback, requires []string) string {
	var b strings.Builder
	b.WriteString("Task " + task.ID)
	if task.Title != "" {
		b.WriteString(": " + task.Title)
	}
	b.WriteString("\n\nObjective:\n" + task.Objective + "\n")
	if len(task.Acceptance) > 0 {
		b.WriteString("\nAcceptance:\n")
		for _, a := range task.Acceptance {
			b.WriteString("- " + a.ID + ": " + a.Statement + "\n")
		}
	}
	if len(requires) > 0 {
		b.WriteString("\nPrevention requirements (compiled contract):\n")
		for _, line := range requires {
			b.WriteString("- " + line + "\n")
		}
	}
	if fb != nil {
		b.WriteString("\nRepair context (verification feedback):\n")
		b.WriteString("- failed attempt: " + strconvItoa(int(fb.Attempt)) +
			" (run " + string(fb.RunID) + ")\n")
		if fb.Subject.AcceptanceID != "" {
			b.WriteString("- failed acceptance " + fb.Subject.AcceptanceID + ": " + fb.Subject.Statement + "\n")
		}
		if fb.Subject.Gate != "" {
			b.WriteString("- failed required gate: " + fb.Subject.Gate + "\n")
		}
		b.WriteString("- failure class: " + string(fb.Failure) + "\n")
		b.WriteString("- diagnostic summary: " + fb.DiagnosticSummary + "\n")
		for _, ev := range fb.Evidence {
			b.WriteString("- visible evidence " + ev.BindingID)
			if ev.Path != "" {
				b.WriteString(" at " + ev.Path)
			}
			if ev.Selector != "" {
				b.WriteString(" selector " + ev.Selector)
			}
			b.WriteString(" status " + string(ev.Status) + "\n")
		}
		b.WriteString("\nThis is repair attempt " + strconvItoa(iteration) +
			"; the previous candidate is already materialized in this workspace. " +
			"Re-work the change so the failed acceptance passes under the same frozen contract.\n")
	} else if iteration > 1 {
		b.WriteString("\nThis is retry iteration " + strconvItoa(iteration) +
			"; the previous iteration did not produce an acceptable candidate. Re-work the change in the same workspace.\n")
	}
	return b.String()
}

func strconvItoa(n int) string { return fmt.Sprintf("%d", n) }

// ---- ContextPack(P1-05 compiled-aware,worker v14ContextPack 同规则)--------

// clContextPack 组装 v14 路径的 ContextPack 并返回 compiled-aware context_hash:
// 候选面与 worker 同一规则(base 提交树里 scope.allowed_paths 之下的 tracked
// 文件 + visible test binding 指定的路径);requires 注入编译权威的执行前
// requires 上下文(PRD §8.3 Prevention),纳入 context_hash —— 同一任务在
// 策略换版后重跑/修复,即便候选面与合同内容不变,上下文坐标也必然改变。
func clContextPack(t *testing.T, repo, base string, task taskcontract.TaskContract, frozen *verification.Frozen, requires contract.RequiresContext) string {
	t.Helper()
	res, err := (runner.LocalRunner{}).Run(context.Background(), runner.Command{
		Argv: []string{"git", "-C", repo, "ls-tree", "-r", "--name-only", "-z", base},
		Env:  e2eGitEnv(),
	})
	if err != nil || res.ExitCode != 0 {
		t.Fatalf("git ls-tree %s: err=%v res=%+v", base, err, res)
	}
	seen := map[string]bool{}
	var candidates []contextpack.Candidate
	add := func(p string) {
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		candidates = append(candidates, contextpack.Candidate{Path: p})
	}
	for _, p := range strings.Split(res.Stdout, "\x00") {
		if p == "" {
			continue
		}
		if p == strings.TrimSuffix(p, "/") && (p == "internal/widget" || strings.HasPrefix(p, "internal/widget/")) {
			add(p)
		}
	}
	c := frozen.Contract()
	for _, acc := range c.Acceptances {
		for _, b := range acc.EvidenceBindings {
			if b.VisibilityPolicy == verification.VisibilityVisible &&
				b.Evidence.Type == verification.EvidenceTest && b.Evidence.Path != "" {
				add(b.Evidence.Path)
			}
		}
	}
	out, err := contextpack.Select(contextpack.SelectInput{
		Task:         task,
		Verification: frozen,
		Candidates:   candidates,
		Requires:     requires,
	})
	if err != nil {
		t.Fatalf("contextpack.Select: %v", err)
	}
	return out.Pack.Hash()
}

// ---- 测试等价组合(runV14Task 同型)------------------------------------------

// clDeps 是组合的依赖面:queries 为生产事实查询集;runtime 必须注入(本矩阵
// 的 FakeRuntime 脚本);verifyWrap 可在生产验证服务外套一层测试观察/注入缝;
// ExposeRepairReason 非空时,修复尝试开始经共享暴露账面蓄意披露一笔
// debug_disclosure(锚点 = 冻结前候选;P2-06 的「修复期间暴露」物理位置)。
type clDeps struct {
	cfg                clConfig
	queries            *store.Queries
	runtime            agentruntime.Runtime
	verifyWrap         func(closedloop.CandidateVerification) closedloop.CandidateVerification
	ExposeRepairReason string
}

// clObservation 是一次闭环执行的可观测面(Authority/Bundle 暴露编译权威与
// 同源 bundle,供 P1-08 gate 断言从同一 pinned 提交确定性复算)。
type clObservation struct {
	Result    closedloop.Result
	Frozen    *verification.Frozen
	Task      taskcontract.TaskContract
	Authority *artifacts.CompiledAuthority
	Bundle    *artifacts.ExecutionBundle
	Instrs    []string
}

// clPrepared 是闭环组合的产出:同源 bundle、编译权威、冻结契约与组装好的
// 生产组合根引擎(尚未执行)。
type clPrepared struct {
	Bundle    *artifacts.ExecutionBundle
	Authority *artifacts.CompiledAuthority
	Frozen    *verification.Frozen
	Task      taskcontract.TaskContract
	Loop      *closedloop.Engine
}

// clComposeTask 按 cmd/worker runV14Task 的生产组成与执行前置序(P1-04)装配
// 一次 v1.4 闭环:同源执行 bundle 装载(P1-01,DL-04:一次 revision 解析把
// FrozenIntent / TaskContract / 冻结 VerificationContract / 策略快照装到同一
// pinned 提交)→ **编译权威(P1-03 CompileAuthority,执行的第一道门,先于
// 任何 runtime 会话与一次性 workspace 创建)** → held-out 前提强制 → 一次性
// worktree 根 → compiled-aware ContextPack(P1-05:Requires 进 context_hash)
// → 生产事实面 → 生产候选验证服务 → closedloop.NewLoop(LoopConfig.Authority
// 必达:P1-06 验收守卫在测试等价组合同样生效 —— no-bypass 的 test/** 面)。
// 任何执行前置失败都以类型化 error 返回(与 worker 同裁决:编译/装载权威
// 失败不是候选失败,不进闭环、不伪装 verdict、不耗修复预算),绝不产出半成品
// 闭环;负例(bypass 夹具)经本缝断言类型化失败与零 runtime 会话。
func clComposeTask(t *testing.T, ctx context.Context, deps clDeps, taskID string) (*clPrepared, error) {
	cfg := deps.cfg

	bundle, err := artifacts.LoadBundle(cfg.Repo, "HEAD", taskID)
	if err != nil {
		return nil, fmt.Errorf("cl: 装载 %s 的同源执行 bundle(HEAD): %w", cfg.Repo, err)
	}
	if bundle.ProjectID == "" {
		return nil, fmt.Errorf("cl: 提交 %s 缺少 .tally/project.yaml(项目身份必须由 Git 注册表声明;与 worker 同一 fail-closed)", bundle.SourceCommit)
	}
	task := bundle.Task.Contract
	if task.Profile != taskcontract.ProfileStandard {
		return nil, fmt.Errorf("cl: task %s profile=%q,非 standard", taskID, task.Profile)
	}
	frozen := bundle.Verification.Frozen

	// 编译权威(P1-03/P1-04,PRD §16「Worker 不允许绕过 Compiler」):任一
	// fail-safe(拒收策略/无效风险依据/来源不一致)都不产出 Compiled,执行
	// 在任何 runtime 会话之前停止。
	authority, err := bundle.CompileAuthority()
	if err != nil {
		return nil, fmt.Errorf("cl: 编译 %s 的执行权威: %w", taskID, err)
	}

	// held-out 前提(与 worker enforceHeldOutPrerequisites 同一合同):通道
	// 接线 + runtime session.isolation 能力声明。
	if candidateverify.ContractHasHeldOut(frozen) {
		if strings.TrimSpace(cfg.HoldoutRoot) == "" {
			return nil, errors.New("cl: 契约含 held_out binding 但 holdout 通道未接线")
		}
		if !deps.runtime.Descriptor().Capabilities.Has(agentruntime.CapSessionIsolation) {
			return nil, errors.New("cl: runtime 未声明 session.isolation,拒绝执行含 held_out binding 的契约")
		}
	}

	mgr, err := workspace.New(cfg.Repo, cfg.WorktreeRoot)
	if err != nil {
		return nil, fmt.Errorf("cl: workspace.New: %w", err)
	}
	worktrees := workspace.NewRunTable(mgr)

	packHash := clContextPack(t, cfg.Repo, bundle.SourceCommit, task, frozen, authority.Compiled.Requires())

	runs := runstore.New(deps.queries)
	wfSink := workflow.NewRunStoreSink(runs)
	verify := closedloop.CandidateVerification(candidateverify.New(candidateverify.Config{
		Repo:         cfg.Repo,
		WorktreeRoot: cfg.WorktreeRoot,
		HoldoutRoot:  cfg.HoldoutRoot,
		EnvName:      cfg.EnvName,
		Recorder:     evidence.Recorder{Kind: evidence.RecorderVerificationRun, Actor: "tally-e2e/p2-10"},
	}, candidateverify.Deps{Queries: deps.queries}))
	if deps.verifyWrap != nil {
		verify = deps.verifyWrap(verify)
	}
	lineage, err := closedloop.NewStoreLineageSink(runs)
	if err != nil {
		return nil, fmt.Errorf("cl: NewStoreLineageSink: %w", err)
	}
	exposure, err := closedloop.NewStoreExposureSink(runs, frozen.Revision())
	if err != nil {
		return nil, fmt.Errorf("cl: NewStoreExposureSink: %w", err)
	}
	seeds, err := runs.ExposuresForTask(ctx, clProject, taskID)
	if err != nil {
		return nil, fmt.Errorf("cl: 读暴露事实种子: %w", err)
	}

	runner := &clCodegenRunner{
		project:  clProject,
		task:     task,
		policy:   policy.DefaultExecutionPolicy(),
		profile:  "default-coding",
		runtime:  deps.runtime,
		model:    "test-model",
		behavior: "behavior-1",
		verifRev: frozen.Hash(),
		packHash: packHash,
		requires: authority.Compiled.Requires().Lines(),
		sink:     wfSink,
	}

	repair := closedloop.DefaultRepairPolicy()
	if cfg.MaxRepair != 0 {
		repair, err = closedloop.NewRepairPolicy(cfg.MaxRepair)
		if err != nil {
			return nil, fmt.Errorf("cl: 修复预算非法: %w", err)
		}
	}
	loop, err := closedloop.NewLoop(closedloop.LoopConfig{
		Codegen:  runner,
		Verify:   verify,
		Contract: frozen,
		// 验收守卫(P1-06/P1-08,DL-07/A-03):编译权威在测试等价组合里同样是
		// accept 权威的输入 —— FinalVerified 不再单独授权普通 Accept,必需
		// 保证档未达成/未匹配风险 fail-safe 升级(assurance_insufficient)。
		Authority: authority.Compiled,
		Runtime:   deps.runtime.Descriptor().Name,
		Worktrees: worktrees,
		BaseRevision: func(context.Context, string) (string, error) {
			// 首次尝试基线 = bundle 来源提交(与编译权威同一 pinned 提交)。
			return bundle.SourceCommit, nil
		},
		Repair:    repair,
		Lineage:   lineage,
		Exposure:  exposure,
		Exposures: seeds,
	})
	if err != nil {
		return nil, fmt.Errorf("cl: 组装闭环: %w", err)
	}
	if deps.ExposeRepairReason != "" {
		ledger := loop.ExposureLedger()
		runner.expose = func(ctx context.Context, req closedloop.CodegenRunRequest) error {
			_, rErr := ledger.RecordExposure(ctx, closedloop.RepairExposureSpec{
				TaskID:     taskID,
				Binding:    evidence.BindingRef{Project: clProject, Verification: evidence.VerificationRevision(frozen.Hash()), BindingID: clVisID},
				Acceptance: evidence.AcceptanceID(clAccID),
				Anchor:     req.RepairBase.ID(),
				Channel:    evidence.ChannelDebugDisclosure,
				Reason:     deps.ExposeRepairReason,
				ExposedAt:  time.Now().UTC(),
			})
			return rErr
		}
	}
	return &clPrepared{Bundle: bundle, Authority: authority, Frozen: frozen, Task: task, Loop: loop}, nil
}

// clRunTask 组合并执行一个 standard TaskContract 的 v1.4 闭环(生产等价组成;
// 组合失败 = 执行前置失败,直接 Fatal —— 负例断言走 clComposeTask)。
func clRunTask(t *testing.T, ctx context.Context, deps clDeps, taskID string) clObservation {
	t.Helper()
	prep, err := clComposeTask(t, ctx, deps, taskID)
	if err != nil {
		t.Fatalf("cl: 组合失败(执行前置): %v", err)
	}
	res, err := prep.Loop.Run(ctx, taskID)
	if err != nil {
		t.Fatalf("闭环执行: %v", err)
	}
	return clObservation{
		Result:    res,
		Frozen:    prep.Frozen,
		Task:      prep.Task,
		Authority: prep.Authority,
		Bundle:    prep.Bundle,
		Instrs:    deps.runtime.(*clScriptedRuntime).instructions(),
	}
}

// ---- 共享断言 ----------------------------------------------------------------

// clAssertOwnerClean 断言 owner 主 worktree 零污染(INV-EXEC-01,AC7)。
func clAssertOwnerClean(t *testing.T, repo string) {
	t.Helper()
	if dirty := clGit(t, repo, "status", "--porcelain"); dirty != "" {
		t.Fatalf("owner 主 worktree 被污染:\n%s", dirty)
	}
}

// clAssertRootEmpty 断言一次性根已清空(尝试结束即丢弃沙箱)。
func clAssertRootEmpty(t *testing.T, root string) {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Fatalf("读一次性根 %s: %v", root, err)
	}
	if len(entries) != 0 {
		t.Fatalf("一次性根 %s 残留 %d 项,期望已清空", root, len(entries))
	}
}
