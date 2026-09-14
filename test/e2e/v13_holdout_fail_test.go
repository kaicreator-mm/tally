//go:build integration

// Package e2e 承载 P1-48 门禁端到端测试:构造一次受控的 held-out 失败,证明它
// 不能静默变成 VERIFIED(PRD §36.3 Trust Gate「至少一次构造 holdout failure 阻断
// VERIFIED」、INV-VER-05;§8.9:P1 只证明 FAIL 阻断,Recovery Ladder 自动化属 P2,
// 决策锁 12)。
//
// 被测纵向切片跨真实边界组装(与 cmd/worker v13 垂直路径同一组合,全部经可导入
// 的包边界):一次性 Git worktree 上跑单 Task 工作流内核(P1-39,FakeRuntime,
// RunStoreSink 把 Run 与候选 ChangeSet 事实落进真实 PostgreSQL,每测试独立
// schema、自跑 goose 迁移)→ 对固定候选做最终验证编排(P1-38:visible 冻结协议 +
// held-out 注入),VerificationRun / evidence 事实经 verification.RunSink →
// runstore 适配落库 → taskstate 纯函数推导 Task 生命周期状态(§18.1)。
//
// 受控失败的设计:同一 acceptance 挂两条 binding —— VIS-1(visible,`go version`,
// 必过)与 HOLD-1(held_out,`go version -m holdout-input`,对 held-out 内容的
// 「是 Go 可执行文件」性质检查确定性失败)。可见面全绿而 held-out FAIL:
// 阻断必须来自 held-out 事实本身,而不是任何可见失败。
//
// 证明目标(任务验收):
//
//	AC1 held-out 失败细节只留在控制面:实现侧唯一合法投影(RedactedSignal)
//	    结构上不含内容/ref/digest/verdict/binding 身份;evidence 行不落原始
//	    输出;held-out 内容只存在于 Tally 控制的 store 之内。
//	AC2 Task 推导 NEEDS_REVIEW / 非 VERIFIED:验证门结论 needs_review(行上
//	    映射为 error,绝不是 pass);taskstate 推导不出 VERIFIED,缺口可追溯到
//	    该条 held-out FAIL 事实;控制面记入 review_requested 后即 NEEDS_REVIEW。
//	AC3 没有自动 Rung-1/Rung-2 恢复:阻断之后没有第二次 codegen / 验证
//	    Run / held-out 重注入,唯一动作是「留下事实并阻断」。
//	AC4 失败证据保持 append-only:evidence / verification_run 行的 UPDATE /
//	    DELETE 被语句级触发器拒绝;事实 id 重复是错误不是更新;无自动失效裁定;
//	    visible PASS 与 held-out FAIL 并存时 FAIL 仍阻断覆盖(§8.8.1)。
//
// 数据库不可达时 SKIP(不产出假失败)。
package e2e

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"tally/internal/agentruntime"
	"tally/internal/changeset"
	"tally/internal/command"
	"tally/internal/core"
	"tally/internal/evidence"
	"tally/internal/holdout"
	"tally/internal/policy"
	"tally/internal/runner"
	"tally/internal/runstore"
	"tally/internal/taskcontract"
	"tally/internal/taskstate"
	"tally/internal/verification"
	"tally/internal/workflow"
	"tally/internal/workspace"
	"tally/runtimes/fake"
)

// hfSecretMarker 是 held-out 内容中的唯一标记串:实现侧任何通道出现它即为泄漏。
const hfSecretMarker = "EXPECT-BRIGHT-LAKE-4848"

// hfHoldoutContent 是受控 held-out 证据内容:一条「widget 满足隐藏性质」的
// 权威检查材料。冻结协议对它做的性质检查(`go version -m`)确定性失败。
const hfHoldoutContent = "P1-48 controlled held-out evidence: hidden property material, marker " + hfSecretMarker + "\n"

const (
	hfProject = core.ProjectID("proj-p1-48")
	hfTask    = "TASK-P1-48"
	hfRelPath = "internal/widget/widget.go"
)

// hfTaskContract 是被测 standard TaskContract(工作流内核输入)。
func hfTaskContract() taskcontract.TaskContract {
	return taskcontract.TaskContract{
		ID:        hfTask,
		Profile:   taskcontract.ProfileStandard,
		Title:     "P1-48 held-out failure gate",
		Objective: "Prove one controlled held-out failure blocks VERIFIED",
		Scope:     taskcontract.Scope{AllowedPaths: []string{"internal/widget"}},
		Acceptance: []taskcontract.AcceptanceClause{
			{ID: "AC-01", Statement: "widget greets", IndependenceRequired: taskcontract.IndependenceNone},
		},
		Hash: "sha256:p1-48-task-revision",
	}
}

// hfVerification 声明并冻结验证契约:AC-01 同时挂可见 PASS binding 与 held-out
// binding(内容经真实 FilesystemStore 落盘后取得不透明 ref)。ref 由调用方传入。
func hfVerification(t *testing.T, ref holdout.Ref) *verification.Frozen {
	t.Helper()
	c := verification.Contract{
		TaskID:              hfTask,
		Revision:            1,
		RuntimeRequirements: verification.RuntimeRequirements{SessionIsolation: true, HeldOutUnreachable: true, ConformanceClass: "verification-isolation-v1"},
		Acceptances: []verification.Acceptance{{
			ID:                   "AC-01",
			Statement:            "widget greets",
			IndependenceRequired: verification.IndependenceNone,
			EvidenceBindings: []verification.Binding{
				{
					ID:               "VIS-1",
					AuthoringMode:    verification.AuthoringPreCommit,
					VisibilityPolicy: verification.VisibilityVisible,
					Basis:            &verification.IndependenceBasis{Source: verification.SourcePreExisting, Information: verification.InfoVisible},
					Evidence:         verification.EvidenceRef{Type: verification.EvidenceTest, Path: "internal/widget/widget_test.go"},
					ExecutionProtocol: &verification.ExecutionProtocol{
						Repetitions: 1, Aggregation: verification.AggregationAllPass,
						Command: &verification.CommandSpec{Argv: []string{"go", "version"}},
					},
				},
				{
					ID:               "HOLD-1",
					AuthoringMode:    verification.AuthoringPreCommit,
					VisibilityPolicy: verification.VisibilityHeldOut,
					Basis:            &verification.IndependenceBasis{Source: verification.SourceVerificationAuthority, Information: verification.InfoHeldOut},
					Evidence:         verification.EvidenceRef{Type: verification.EvidenceTest, Store: "holdout", Ref: ref.String()},
					ExecutionProtocol: &verification.ExecutionProtocol{
						Repetitions: 1, Aggregation: verification.AggregationAllPass,
						Command: &verification.CommandSpec{Argv: []string{"go", "version", "-m", "holdout-input"}},
					},
				},
			},
		}},
	}
	frozen, err := c.Freeze()
	if err != nil {
		t.Fatalf("冻结验证契约: %v", err)
	}
	return frozen
}

// hfScenario 是一次完整纵向执行的事实集合:codegen(真 worktree + FakeRuntime)+
// 最终验证(真 holdout store + Tally 控制边界 + 落库 sink)+ 全部落库事实。
type hfScenario struct {
	runs   *runstore.Store
	pool   *pgxpool.Pool
	repo   string
	base   string
	frozen *verification.Frozen

	codegen workflow.Outcome
	cs      *changeset.ChangeSet
	verify  *verification.FinalOutcome

	store         *holdout.FilesystemStore
	holdoutRef    holdout.Ref
	contentDigest string
	outputDigest  string // 末次 held-out repetition 的输出摘要(控制面可见)

	holdoutFactID string // "vrun-<run>-HOLD-1"
	runID         int64
}

// hfPutHoldout 把受控失败材料落入真实 Tally 控制 FilesystemStore(Tally 拥有的
// root,机械上在 repo 与一次性 worktree 根之外,决策锁 11)。
func hfPutHoldout(t *testing.T, repo, wtRoot string) (*holdout.FilesystemStore, holdout.Ref, string) {
	t.Helper()
	st, err := holdout.NewFilesystemStore(filepath.Join(t.TempDir(), "holdout"), repo, wtRoot)
	if err != nil {
		t.Fatalf("建 holdout store: %v", err)
	}
	ref, err := st.Put(context.Background(), []byte(hfHoldoutContent))
	if err != nil {
		t.Fatalf("Put held-out 内容: %v", err)
	}
	return st, ref, holdout.ContentDigest([]byte(hfHoldoutContent))
}

// hfWritingRuntime 包一层 FakeRuntime:首个 turn 在会话 workspace 内写一个
// scope 内文件再完成 —— 模拟实现产出,让候选固定钩子有真改动可收。
type hfWritingRuntime struct {
	inner            agentruntime.Runtime
	relPath, content string

	once     sync.Once
	writeErr error
}

func (r *hfWritingRuntime) Descriptor() agentruntime.RuntimeDescriptor { return r.inner.Descriptor() }

func (r *hfWritingRuntime) Start(ctx context.Context, req agentruntime.StartRequest) (agentruntime.Session, error) {
	s, err := r.inner.Start(ctx, req)
	if err != nil {
		return nil, err
	}
	return &hfWritingSession{rt: r, inner: s, ws: req.Workspace}, nil
}

type hfWritingSession struct {
	rt    *hfWritingRuntime
	inner agentruntime.Session
	ws    string
}

func (s *hfWritingSession) Execute(ctx context.Context, req agentruntime.ExecuteRequest) (agentruntime.AgentRunResult, error) {
	s.rt.once.Do(func() {
		p := filepath.Join(s.ws, filepath.FromSlash(s.rt.relPath))
		s.rt.writeErr = errors.Join(os.MkdirAll(filepath.Dir(p), 0o755), os.WriteFile(p, []byte(s.rt.content), 0o644))
	})
	if s.rt.writeErr != nil {
		return agentruntime.AgentRunResult{}, s.rt.writeErr
	}
	return s.inner.Execute(ctx, req)
}

func (s *hfWritingSession) Cancel(ctx context.Context, id agentruntime.RunID) error {
	return s.inner.Cancel(ctx, id)
}

func (s *hfWritingSession) Events(ctx context.Context) (agentruntime.EventStream, error) {
	return s.inner.Events(ctx)
}

func (s *hfWritingSession) Result(ctx context.Context, id agentruntime.RunID) (agentruntime.AgentRunResult, error) {
	return s.inner.Result(ctx, id)
}

func (s *hfWritingSession) Close(ctx context.Context) error { return s.inner.Close(ctx) }

// hfFixCandidate 从一次性 worktree 读取实现产出的文件固定候选(与生产
// fixCandidateFromWorktree 同一语义的窄化版:只认 hfRelPath 这一个改动)。
func hfFixCandidate(rel string) workflow.CandidateFixFunc {
	return func(_ context.Context, v workflow.IterationView) (workflow.CandidateSnapshot, bool, error) {
		b, err := os.ReadFile(filepath.Join(v.WorkspacePath, filepath.FromSlash(rel)))
		if errors.Is(err, os.ErrNotExist) {
			return workflow.CandidateSnapshot{}, false, nil
		}
		if err != nil {
			return workflow.CandidateSnapshot{}, false, err
		}
		return workflow.CandidateSnapshot{
			CandidateRevision: v.BaseRevision,
			Files:             []changeset.ChangedFile{{Path: rel, Content: string(b)}},
			Entities:          []string{"widget"},
		}, true, nil
	}
}

// hfHoldoutBoundary 是 Tally 控制的 held-out 执行边界(与 cmd/worker 的
// holdoutBoundary 同一合同):held-out 内容只以内存字节到达这里,物化进一次性
// 目录执行冻结命令;原始输出只在边界内消化,对外只返回结构化 status + 输出摘要。
func hfHoldoutBoundary(rt runner.Runner) verification.HoldoutRunFunc {
	return func(ctx context.Context, req verification.HoldoutRunRequest) (verification.HoldoutRunOutput, error) {
		if req.Command == nil {
			return verification.HoldoutRunOutput{}, errors.New("p1-48: held-out 冻结命令缺失")
		}
		dir, err := os.MkdirTemp("", "tally-holdout-p148-")
		if err != nil {
			return verification.HoldoutRunOutput{}, err
		}
		defer os.RemoveAll(dir)
		if err := os.WriteFile(filepath.Join(dir, "holdout-input"), req.Content, 0o600); err != nil {
			return verification.HoldoutRunOutput{}, err
		}
		if timeout := time.Duration(req.Command.Timeout); timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}
		res, err := rt.Run(ctx, runner.Command{
			Argv: append([]string(nil), req.Command.Argv...),
			Dir:  dir,
			Env:  append([]string(nil), req.Command.Env...),
		})
		if err != nil {
			return verification.HoldoutRunOutput{}, err
		}
		status := verification.EvidenceStatusFail
		if res.ExitCode == 0 {
			status = verification.EvidenceStatusPass
		}
		return verification.HoldoutRunOutput{
			Status:       status,
			OutputDigest: holdout.ContentDigest([]byte(res.Stdout + "\n" + res.Stderr)),
		}, nil
	}
}

// hfVerifySink 把 verification.RunSink 窄端口接到 runstore 适配器(与 cmd/worker
// 的 v13VerificationSink 同一接线的测试内副本:internal/runstore 反向依赖
// verification 包,适配归集成接线)。append-only:只写与只读,端口上不存在
// 任何更新/删除/裁定语义(INV-FACT-01)。
type hfVerifySink struct {
	runs     *runstore.Store
	revision int32
}

func hfNewVerifySink(runs *runstore.Store, revision int) *hfVerifySink {
	return &hfVerifySink{runs: runs, revision: int32(revision)}
}

func (s *hfVerifySink) RecordRun(ctx context.Context, spec verification.RunSpec) (int64, error) {
	id, err := s.runs.RecordVerificationRun(ctx, runstore.VerificationRunSpec{
		Project:           spec.Project,
		Task:              spec.Task,
		ContractRevision:  int32(spec.ContractRevision),
		ContractHash:      spec.ContractHash,
		ChangeSetID:       spec.ChangeSetID,
		EnvName:           spec.EnvironmentName,
		EnvDefinitionHash: spec.EnvironmentHash,
		VersionCombo:      spec.VersionCombo,
		Runtime:           spec.Runtime,
		StartedAt:         spec.StartedAt,
		FinishedAt:        spec.FinishedAt,
		Status:            hfRunStatus(spec.Status),
	})
	if err != nil {
		return 0, fmt.Errorf("p1-48: 落 verification_run: %w", err)
	}
	return id, nil
}

func hfRunStatus(s verification.RunStatus) runstore.VerificationStatus {
	switch s {
	case verification.RunStatusPass:
		return runstore.VerificationPass
	case verification.RunStatusFail:
		return runstore.VerificationFail
	default:
		return runstore.VerificationError
	}
}

func (s *hfVerifySink) RecordFact(ctx context.Context, project core.ProjectID, fact verification.FactSpec) error {
	exit := int32(fact.ExitCode)
	rec := runstore.EvidenceRecord{
		FactID:           fact.FactID,
		RunID:            fact.RunID,
		Task:             fact.Task,
		Acceptance:       fact.Acceptance,
		BindingID:        fact.BindingID,
		VisibilityPolicy: fact.VisibilityPolicy,
		Verdict:          evidence.Verdict(fact.Verdict),
		Recorder:         fact.Recorder,
		Command:          hfCommandSpec(fact.Command),
		ExitCode:         &exit,
		StdoutDigest:     fact.StdoutDigest,
		StderrDigest:     fact.StderrDigest,
		RecordedAt:       fact.RecordedAt,
	}
	if _, err := s.runs.RecordEvidence(ctx, project, rec); err != nil {
		return fmt.Errorf("p1-48: 落 evidence %s: %w", fact.FactID, err)
	}
	return nil
}

// hfCommandSpec 把冻结协议命令转成 command.CommandSpec(shell 恒为 false,决策锁 6)。
func hfCommandSpec(spec *verification.CommandSpec) *command.CommandSpec {
	if spec == nil {
		return nil
	}
	var env map[string]string
	for _, kv := range spec.Env {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		if env == nil {
			env = make(map[string]string)
		}
		env[k] = v
	}
	return &command.CommandSpec{
		Argv:    append([]string(nil), spec.Argv...),
		Cwd:     spec.Dir,
		Env:     env,
		Timeout: time.Duration(spec.Timeout),
		Shell:   false,
	}
}

func (s *hfVerifySink) ValidApplicableFacts(ctx context.Context, scope evidence.Scope) ([]verification.PriorFact, error) {
	rows, err := s.runs.ApplicableValidEvidence(ctx, scope, s.revision)
	if err != nil {
		return nil, fmt.Errorf("p1-48: 读回既往 evidence: %w", err)
	}
	out := make([]verification.PriorFact, 0, len(rows))
	for _, r := range rows {
		switch evidence.Verdict(r.Verdict) {
		case evidence.VerdictPass:
			out = append(out, verification.PriorFact{BindingID: r.BindingID, Status: verification.EvidenceStatusPass, RecordedAt: r.RecordedAt})
		case evidence.VerdictFail:
			out = append(out, verification.PriorFact{BindingID: r.BindingID, Status: verification.EvidenceStatusFail, RecordedAt: r.RecordedAt})
		}
	}
	return out, nil
}

// hfScenarioRun 驱动一次完整纵向执行(codegen → 最终验证,全部事实落库)。
func hfScenarioRun(t *testing.T) hfScenario {
	t.Helper()
	runs, pool := e2eStore(t) // DB 不可达时 SKIP
	ctx := context.Background()

	repo := e2eNewRepo(t)
	hfCommit(t, repo, "internal/widget/widget_test.go", "package widget\n", "base")
	base := e2eGit(t, repo, "rev-parse", "HEAD")
	wtRoot := filepath.Join(t.TempDir(), "wt")

	// authoring:held-out 内容先落 Tally 控制 store,契约引用不透明 ref。
	hs, ref, contentDigest := hfPutHoldout(t, repo, wtRoot)
	frozen := hfVerification(t, ref)

	// 1. Codegen:一次性 worktree + FakeRuntime + RunStoreSink 落库。
	codegenMgr, err := workspace.New(repo, wtRoot)
	if err != nil {
		t.Fatalf("workspace.New: %v", err)
	}
	texec, err := workflow.NewTaskExecution(workflow.TaskExecutionSpec{
		Project:              hfProject,
		Task:                 hfTaskContract(),
		Role:                 "coding",
		Policy:               policy.DefaultExecutionPolicy(),
		BaseRevision:         base,
		Profile:              "default-coding",
		Runtime:              &hfWritingRuntime{inner: fake.New(fake.Config{}), relPath: hfRelPath, content: "package widget // p1-48 candidate\n"},
		Model:                "test-model",
		BehaviorRevision:     "behavior-1",
		VerificationRevision: frozen.Hash(),
		ContextPackHash:      "sha256:p1-48-context-pack",
		Worktrees:            workspace.NewRunTable(codegenMgr),
		Sink:                 workflow.NewRunStoreSink(runs),
		Hooks:                workflow.Hooks{FixCandidate: hfFixCandidate(hfRelPath)},
	})
	if err != nil {
		t.Fatalf("NewTaskExecution: %v", err)
	}
	out, err := texec.Execute(ctx)
	if err != nil {
		t.Fatalf("codegen Execute: %v", err)
	}
	if out.Status != workflow.TaskCompleted || out.ChangeSet == nil {
		t.Fatalf("codegen 未固定候选: status=%s reason=%s", out.Status, out.Reason)
	}
	cs := out.ChangeSet

	// 2. 最终验证:候选物化进一次性验证 workspace,held-out 经真实 store + 边界注入。
	vmgr, err := workspace.New(repo, wtRoot)
	if err != nil {
		t.Fatalf("verification workspace.New: %v", err)
	}
	vw, err := vmgr.Create(ctx, cs.BaseRevision())
	if err != nil {
		t.Fatalf("建验证 workspace: %v", err)
	}
	defer func() {
		if err := vmgr.Cleanup(context.WithoutCancel(ctx), vw); err != nil {
			t.Errorf("清理验证 workspace: %v", err)
		}
	}()
	for _, f := range cs.Files() {
		p := filepath.Join(vw.Path, filepath.FromSlash(filepath.Clean(f.Path)))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("物化候选目录: %v", err)
		}
		if err := os.WriteFile(p, []byte(f.Content), 0o644); err != nil {
			t.Fatalf("物化候选 %s: %v", f.Path, err)
		}
	}

	orch, err := verification.NewOrchestrator(verification.OrchestratorConfig{
		Runner:        runner.LocalRunner{},
		Policy:        policy.DefaultExecutionPolicy(),
		HoldoutReader: hs,
		HoldoutRun:    hfHoldoutBoundary(runner.LocalRunner{}),
		Sink:          hfNewVerifySink(runs, frozen.Revision()),
		Recorder:      evidence.Recorder{Kind: evidence.RecorderVerificationRun, Actor: "tally-e2e/p1-48"},
	})
	if err != nil {
		t.Fatalf("构造最终验证编排器: %v", err)
	}
	outcome, err := orch.Run(ctx, verification.FinalRunRequest{
		Project:         hfProject,
		Contract:        frozen,
		ChangeSet:       cs,
		EnvironmentName: "p1-48-e2e-env",
		Environment: verification.EnvironmentIdentity{
			OS: goruntime.GOOS, Arch: goruntime.GOARCH, Toolchain: goruntime.Version(),
		},
		VersionCombo:     core.VersionCombo{"go": goruntime.Version(), "runtime": "fake"},
		Runtime:          "fake",
		Workspace:        vw.Path,
		CandidateFixedAt: cs.CreatedAt(),
	})
	if err != nil {
		t.Fatalf("最终验证执行: %v", err)
	}
	if len(outcome.Holdout) != 1 || len(outcome.Holdout[0].OutputDigests) != 1 {
		t.Fatalf("held-out 判定记录不完整: %+v", outcome.Holdout)
	}

	return hfScenario{
		runs: runs, pool: pool, repo: repo, base: base, frozen: frozen,
		codegen: out, cs: cs, verify: outcome,
		store: hs, holdoutRef: ref, contentDigest: contentDigest,
		outputDigest:  outcome.Holdout[0].OutputDigests[len(outcome.Holdout[0].OutputDigests)-1],
		holdoutFactID: fmt.Sprintf("vrun-%d-HOLD-1", outcome.RunID),
		runID:         outcome.RunID,
	}
}

// hfCommit 在 owner repo 里提交一个文件(目录自动创建;结构化 git argv)。
func hfCommit(t *testing.T, repo, name, content, msg string) {
	t.Helper()
	p := filepath.Join(repo, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("建目录 %s: %v", name, err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("写 %s: %v", name, err)
	}
	e2eGit(t, repo, "add", ".")
	e2eGit(t, repo, "commit", "-m", msg)
}

// hfEvidenceRowJSON 读一条 evidence 行的完整投影(黑盒泄漏扫描用)。
func hfEvidenceRowJSON(t *testing.T, pool *pgxpool.Pool, factID string) string {
	t.Helper()
	var raw []byte
	if err := pool.QueryRow(context.Background(),
		"SELECT to_jsonb(ve.*) FROM verification_evidence ve WHERE fact_id=$1", factID).Scan(&raw); err != nil {
		t.Fatalf("读 evidence 行 %s: %v", factID, err)
	}
	return string(raw)
}

// ---- AC2 + AC3:held-out FAIL 阻断 VERIFIED,且没有自动恢复 -----------------

func TestV13HeldOutFailureBlocksVerified(t *testing.T) {
	scn := hfScenarioRun(t)
	ctx := context.Background()
	outcome := scn.verify

	// 可见面全绿:visible binding PASS(阻断只能来自 held-out 事实本身)。
	var vis *verification.BindingOutcome
	for i := range outcome.Bindings {
		if outcome.Bindings[i].BindingID == "VIS-1" {
			vis = &outcome.Bindings[i]
		}
	}
	if vis == nil || vis.Err != "" || vis.Status != verification.EvidenceStatusPass {
		t.Fatalf("visible binding 应 PASS: %+v", outcome.Bindings)
	}

	// AC2(门):结论 needs_review —— 不是 verified,也不是 failed。
	if outcome.Status != verification.FinalNeedsReview {
		t.Fatalf("验证结论 = %s, want needs_review(held-out FAIL 阻断,§36.3)", outcome.Status)
	}
	// AC4(覆盖面):acceptance 有 passing binding(VIS-1)但被 HOLD-1 的
	// valid+applicable FAIL 阻断 —— 新 PASS 不抹掉合法 FAIL(§8.8.1)。
	acc := outcome.Functional.Acceptances
	if len(acc) != 1 || acc[0].Covered || acc[0].Gap != verification.GapBlockingFailure {
		t.Fatalf("功能覆盖 = %+v, want AC-01 被 blocking failure 阻断", acc)
	}
	if len(acc[0].Blocking) != 1 || acc[0].Blocking[0] != "HOLD-1" {
		t.Fatalf("阻断 binding = %v, want [HOLD-1]", acc[0].Blocking)
	}
	if len(acc[0].SatisfiedBy) == 0 || acc[0].SatisfiedBy[0] != "VIS-1" {
		t.Fatalf("满足 binding = %v, want 含 VIS-1(PASS 在场也阻断)", acc[0].SatisfiedBy)
	}
	if outcome.Functional.Complete {
		t.Fatal("功能覆盖 Complete=true,held-out FAIL 未阻断(AC4)")
	}

	// AC3(无自动恢复):阻断之后系统没有第二次动作 —— 恰好 1 个 codegen Run、
	// 恰好 1 行 verification_run、held-out binding 恰好注入一次;没有 Rung-1
	// 重执行、没有 Rung-2 换约重验(§8.9:P1 只证明阻断,恢复自动化属 P2)。
	if n := len(scn.codegen.Runs); n != 1 {
		t.Fatalf("codegen Run 数 = %d, want 1(无自动返工迭代,AC3)", n)
	}
	timeline, err := scn.runs.CodegenRunsForTask(ctx, hfProject, hfTask)
	if err != nil || len(timeline) != 1 || timeline[0].Status != runstore.RunCompleted {
		t.Fatalf("codegen_run 时间线 = %+v err=%v, want 恰好 1 条 completed(AC3)", timeline, err)
	}
	vruns, err := scn.runs.VerificationRunsForTask(ctx, hfProject, hfTask)
	if err != nil || len(vruns) != 1 {
		t.Fatalf("verification_run 行数 = %d err=%v, want 恰好 1 行(无自动重验,AC3)", len(vruns), err)
	}
	run := vruns[0]
	if run.ID != scn.runID {
		t.Fatalf("verification_run id = %d, want %d", run.ID, scn.runID)
	}
	// AC2(事实面):needs_review 在 verification_run 行上映射为 error ——
	// 运行行绝不是 pass,不存在可被读成 VERIFIED 的 pass 事实。
	if run.Status != runstore.VerificationError {
		t.Fatalf("verification_run status = %s, want error(needs_review 映射;绝不 pass,AC2)", run.Status)
	}
	if run.ChangeSetID != scn.cs.ID() || run.ContractHash != scn.frozen.Hash() {
		t.Fatalf("verification_run 绑定 = %s/%s, want 精确候选与冻结契约", run.ChangeSetID, run.ContractHash)
	}
	evs, err := scn.runs.EvidenceForRun(ctx, hfProject, scn.runID)
	if err != nil || len(evs) != 2 {
		t.Fatalf("evidence 行数 = %d err=%v, want 恰好 2(VIS-1 + HOLD-1,无恢复产生的事实,AC3)", len(evs), err)
	}
	if len(outcome.FactIDs) != 2 {
		t.Fatalf("落库事实 id = %v, want 恰好 2 条(AC3)", outcome.FactIDs)
	}

	// AC2(状态面):从落库事实推导 —— 推导不出 VERIFIED,缺口精确追溯到该条
	// held-out FAIL;控制面记入 review_requested(非 VERIFIED 信号的标准后果,
	// §29/§8.9)后即 NEEDS_REVIEW。见 TestV13HeldOutFailureDerivesNonVerifiedTaskState。
}

// ---- AC1:held-out 失败细节只在控制面 ---------------------------------------

func TestV13HeldOutFailureStaysControlPlaneOnly(t *testing.T) {
	scn := hfScenarioRun(t)

	// 正向对照:控制面(FinalOutcome)确实拿到了完整失败判定 —— 若控制面也没有
	// verdict,阻断就无从证明。
	h := scn.verify.Holdout[0]
	if h.Status != verification.EvidenceStatusFail || h.BindingID != "HOLD-1" || h.AcceptanceID != "AC-01" {
		t.Fatalf("控制面 held-out 判定 = %+v, want HOLD-1 FAIL", h)
	}
	if h.ContentDigest != scn.contentDigest {
		t.Fatalf("控制面内容摘要 = %s, want %s", h.ContentDigest, scn.contentDigest)
	}
	if h.OutputDigests[0] != scn.outputDigest {
		t.Fatalf("控制面输出摘要漂移: %s != %s", h.OutputDigests[0], scn.outputDigest)
	}

	// 实现侧唯一合法投影:allowlist 逐字段相等;结构上不存在
	// ref/digest/verdict/binding 身份(§8.4.3,P1-29 的 RedactHoldoutRecord)。
	sig := scn.verify.RedactedSignal()
	if sig.Status != verification.FinalNeedsReview {
		t.Fatalf("实现侧信号 status = %s, want needs_review(非 VERIFIED 信号本身可见)", sig.Status)
	}
	if len(sig.Holdouts) != 1 {
		t.Fatalf("实现侧 held-out 投影数 = %d, want 1", len(sig.Holdouts))
	}
	want := verification.RedactedHoldoutOutcome{
		TaskID:           hfTask,
		ContractRevision: scn.frozen.Revision(),
		AcceptanceID:     "AC-01",
		EvidenceType:     verification.EvidenceTest,
		CompletedAt:      h.CompletedAt,
	}
	if sig.Holdouts[0] != want {
		t.Fatalf("实现侧投影 = %+v, want allowlist 投影 %+v", sig.Holdouts[0], want)
	}
	// 黑盒泄漏扫描:渲染整份实现侧信号,内容标记/ref/内容摘要/输出摘要/binding
	// 身份一个都不许出现。
	rendered := fmt.Sprintf("%+v", sig)
	for _, leak := range []string{hfSecretMarker, scn.holdoutRef.String(), scn.contentDigest, scn.outputDigest, "HOLD-1", "ho-"} {
		if strings.Contains(rendered, leak) {
			t.Fatalf("实现侧信号泄漏 %q:%s", leak, rendered)
		}
	}

	// 落库 evidence 行是控制面事实(Review 可见 verdict),但绝不携带原始失败
	// 输出(raw_output_ref 恒空)与 held-out 内容;黑盒扫描整行投影。
	if raw := hfEvidenceRowJSON(t, scn.pool, scn.holdoutFactID); strings.Contains(raw, hfSecretMarker) {
		t.Fatalf("evidence 行泄漏 held-out 内容:%s", raw)
	}
	ev, err := scn.runs.Evidence(context.Background(), hfProject, evidence.FactID(scn.holdoutFactID))
	if err != nil {
		t.Fatalf("读回 held-out evidence: %v", err)
	}
	if ev.RawOutputRef != "" {
		t.Fatalf("held-out evidence 行携带 raw_output_ref=%q(§14.3:原始输出不得进入可投影行)", ev.RawOutputRef)
	}
	if ev.Command != nil {
		t.Fatalf("held-out evidence 行携带 command(冻结协议命令不进 evidence 投影): %+v", ev.Command)
	}
	if ev.VisibilityPolicy != verification.VisibilityHeldOut || ev.Verdict != evidence.VerdictFail {
		t.Fatalf("held-out evidence 行 = visibility=%s verdict=%s, want held_out/fail", ev.VisibilityPolicy, ev.Verdict)
	}

	// held-out 内容自始至终只在 Tally 控制 store 内:owner 主 worktree 零污染
	// (INV-EXEC-01),store 内容摘要不变(无恢复路径改写材料)。
	if dirty := e2eGit(t, scn.repo, "status", "--porcelain"); dirty != "" {
		t.Fatalf("owner 主 worktree 被污染: %q", dirty)
	}
	meta, err := scn.store.Stat(context.Background(), scn.holdoutRef)
	if err != nil || meta.Digest != scn.contentDigest {
		t.Fatalf("holdout store 内容漂移: %+v err=%v", meta, err)
	}
	if n := countRows(t, scn.pool, "SELECT count(*) FROM evidence_exposure WHERE project_id=$1", string(hfProject)); n != 0 {
		t.Fatalf("出现 %d 条暴露事实:实现侧通道出现了 held-out 内容(AC1)", n)
	}
}

// ---- AC4:失败证据保持 append-only -------------------------------------------

func TestV13HeldOutFailureEvidenceAppendOnly(t *testing.T) {
	scn := hfScenarioRun(t)
	ctx := context.Background()

	// 失败事实在场且无任何失效裁定:失效只能经带 provenance 的正式裁定,
	// 系统没有自动裁定路径(§8.8.1)。
	ev, err := scn.runs.Evidence(ctx, hfProject, evidence.FactID(scn.holdoutFactID))
	if err != nil {
		t.Fatalf("读回 held-out 失败事实: %v", err)
	}
	rulings, err := scn.runs.EvidenceRulings(ctx, hfProject, ev.ID)
	if err != nil || len(rulings) != 0 {
		t.Fatalf("失败事实已有 %d 条裁定 err=%v:出现了自动失效路径(AC4)", len(rulings), err)
	}

	// 数据库侧防线:失败事实与 Run 行的一切改写/删除都被语句级触发器拒绝 ——
	// 包括把 verdict 翻成 pass、把 run 翻成 pass 的「静默变 VERIFIED」路径
	// (migrations/0020,INV-FACT-01)。
	mustModifyRejected(t, scn.pool, "UPDATE verification_evidence SET verdict='pass' WHERE id=$1", ev.ID)
	mustModifyRejected(t, scn.pool, "UPDATE verification_evidence SET visibility_policy='visible' WHERE id=$1", ev.ID)
	mustModifyRejected(t, scn.pool, "DELETE FROM verification_evidence WHERE id=$1", ev.ID)
	mustModifyRejected(t, scn.pool, "UPDATE verification_run SET status='pass' WHERE id=$1", scn.runID)
	mustModifyRejected(t, scn.pool, "DELETE FROM verification_run WHERE id=$1", scn.runID)

	// 事实身份唯一且永不复用:同一 fact_id 二次记录是约束错误,不是更新。
	dup := runstore.EvidenceRecord{
		FactID:           evidence.FactID(scn.holdoutFactID),
		RunID:            scn.runID,
		Task:             hfTask,
		Acceptance:       "AC-01",
		BindingID:        "HOLD-1",
		VisibilityPolicy: verification.VisibilityHeldOut,
		Verdict:          evidence.VerdictPass, // 试图用同 id 的 PASS 覆盖 FAIL
		Recorder:         evidence.Recorder{Kind: evidence.RecorderVerificationRun, Actor: "tally-e2e/p1-48"},
		RecordedAt:       time.Now(),
	}
	if _, err := scn.runs.RecordEvidence(ctx, hfProject, dup); err == nil {
		t.Fatal("同 fact_id 二次记录应报错(事实身份不复用,INV-FACT-01)")
	}
	after, err := scn.runs.Evidence(ctx, hfProject, evidence.FactID(scn.holdoutFactID))
	if err != nil || after.Verdict != evidence.VerdictFail {
		t.Fatalf("重复记录后事实被改写: %+v err=%v(AC4)", after, err)
	}

	// 拒绝路径没有吞掉既有事实:失败事实仍然 valid + applicable,继续 gate
	// 当前候选(§8.8.1/8.8.2)。
	scope := evidence.Scope{
		Project:      hfProject,
		Task:         evidence.TaskID(hfTask),
		Acceptance:   "AC-01",
		Verification: evidence.VerificationRevision(scn.frozen.Hash()),
		ChangeSet:    evidence.ChangeSetID(scn.cs.ID()),
	}
	hits, err := scn.runs.ApplicableValidEvidence(ctx, scope, int32(scn.frozen.Revision()))
	if err != nil {
		t.Fatalf("ApplicableValidEvidence: %v", err)
	}
	verdicts := map[string]evidence.Verdict{}
	for _, h := range hits {
		verdicts[h.BindingID] = h.Verdict
	}
	if verdicts["HOLD-1"] != evidence.VerdictFail || verdicts["VIS-1"] != evidence.VerdictPass {
		t.Fatalf("适用事实投影 = %v, want HOLD-1 fail 仍 valid+applicable(AC4)", verdicts)
	}
	if n := countRows(t, scn.pool, "SELECT count(*) FROM verification_evidence WHERE binding_id='HOLD-1' AND project_id=$1", string(hfProject)); n != 1 {
		t.Fatalf("HOLD-1 失败事实行数 = %d, want 恰好 1(无恢复/重注入产生新行,AC3/AC4)", n)
	}
}

// ---- AC2:TaskExecutionState 推导非 VERIFIED / NEEDS_REVIEW ------------------

func TestV13HeldOutFailureDerivesNonVerifiedTaskState(t *testing.T) {
	scn := hfScenarioRun(t)

	// 事实投影:垂直路径落库的生命周期事实(全部经 fact-capable 通道)。
	at := func(i int) time.Time { return time.Date(2026, 9, 6, 12, i, 0, 0, time.UTC) }
	rec := evidence.Recorder{Kind: evidence.RecorderVerificationRun, Actor: "tally-e2e/p1-48"}
	facts := []taskstate.Fact{
		{ID: "hf-fact-planned", Kind: taskstate.FactTaskPlanned, At: at(1), Recorder: rec, RevisionRef: hfTaskContract().Hash},
		{ID: "hf-fact-frozen", Kind: taskstate.FactVerificationFrozen, At: at(2), Recorder: rec, RevisionRef: scn.frozen.Hash()},
		{ID: "hf-fact-run", Kind: taskstate.FactCodegenRunStarted, At: at(3), Recorder: rec, RunID: string(scn.codegen.Runs[0].ID)},
		{ID: "hf-fact-candidate", Kind: taskstate.FactChangeSetProduced, At: at(4), Recorder: rec, RunID: string(scn.codegen.Runs[0].ID), ChangeSet: evidence.ChangeSetID(scn.cs.ID())},
		{ID: "hf-fact-vrun", Kind: taskstate.FactVerificationRunStarted, At: at(5), Recorder: rec, RunID: strconv.FormatInt(scn.runID, 10), ChangeSet: evidence.ChangeSetID(scn.cs.ID())},
	}
	// held-out FAIL 作为 valid 且 applicable 的失败事实(控制面投影)。
	failures := []taskstate.ApplicableFailure{{
		FactID:    evidence.FactID(scn.holdoutFactID),
		BindingID: "HOLD-1",
		ChangeSet: evidence.ChangeSetID(scn.cs.ID()),
		Recorder:  rec,
		At:        at(6),
	}}

	// (1) 无 review 请求:验证门已定 needs_review 但控制面还没记入 review 事实
	// —— 任务停在 VALIDATING,推导不出 VERIFIED/APPROVED/MERGED,缺口可追溯到
	// held-out FAIL 事实与 AC-01(INV-STATE-02)。
	res, err := taskstate.Derive(taskstate.Snapshot{
		Contract:     scn.frozen,
		Facts:        facts,
		Functional:   &scn.verify.Functional,
		Independence: &scn.verify.Independence,
		Failures:     failures,
	})
	if err != nil {
		t.Fatalf("taskstate.Derive: %v", err)
	}
	if res.State == taskstate.StateVerified || res.State == taskstate.StateApproved || res.State == taskstate.StateMerged {
		t.Fatalf("状态 = %s:held-out FAIL 未阻断生命周期(AC2)", res.State)
	}
	if res.VerificationClass != taskstate.ClassNone {
		t.Fatalf("VerificationClass = %s, want 空(未到 VERIFIED,AC2)", res.VerificationClass)
	}
	has := func(req taskstate.Requirement) bool {
		for _, g := range res.Gaps {
			if g.Requirement == req {
				return true
			}
		}
		return false
	}
	if !has(taskstate.ReqFunctionalCoverage) || !has(taskstate.ReqNoApplicableFailures) {
		t.Fatalf("缺口 = %+v, want 功能覆盖与 applicable-failures 两个缺口(AC2)", res.Gaps)
	}

	// (2) 控制面把非 VERIFIED 信号记入 review_requested(§29 的标准后果):
	// Task 状态即 NEEDS_REVIEW —— 不是任何可继续推进的主线位置,更不是 VERIFIED。
	facts = append(facts, taskstate.Fact{ID: "hf-fact-review", Kind: taskstate.FactReviewRequested, At: at(7), Recorder: rec})
	res2, err := taskstate.Derive(taskstate.Snapshot{
		Contract:     scn.frozen,
		Facts:        facts,
		Functional:   &scn.verify.Functional,
		Independence: &scn.verify.Independence,
		Failures:     failures,
	})
	if err != nil {
		t.Fatalf("taskstate.Derive(review): %v", err)
	}
	if res2.State != taskstate.StateNeedsReview {
		t.Fatalf("状态 = %s, want NEEDS_REVIEW(AC2)", res2.State)
	}
	if res2.VerificationClass != taskstate.ClassNone {
		t.Fatalf("NEEDS_REVIEW 下 VerificationClass = %s, want 空(AC2)", res2.VerificationClass)
	}
}
