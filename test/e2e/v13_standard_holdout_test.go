//go:build integration

// P1-47 端到端门禁(PRD §36.3 P1 Trust Gate):一个真实 P1 standard Task
// 带着 held-out 最终验证走完全程。
//
// 被测纵向切片与 P1-43 的 worker 接线(cmd/worker/v13.go)同构,但由本测试
// 直接从有界包组装(测试不能 import package main):
//
//	真实 Git repo → 一次性 worktree + FakeRuntime 单 Task 工作流内核(P1-39)
//	→ RunStoreSink 把 CodegenRun 与不可变 ChangeSet 落进真实 PostgreSQL
//	(每测试独立 schema、自跑 goose 迁移)→ 冻结 VerificationContract
//	(visible 增量检查 + held_out binding)经最终验证编排器(P1-38)执行:
//	visible 冻结协议命令在物化候选的一次性验证 workspace 内真实运行,
//	held-out 内容经 Tally 控制的 holdout 文件系统 Store(P1-27)与一次性
//	注入执行器(P1-29)进入最终验证 → VerificationRun / evidence 事实
//	append-only 落库 → TaskExecutionState(P1-22)只在证据齐备后推导为
//	VERIFIED。
//
// 验收映射(任务文件 AC1-AC5):
//
//	AC1 TaskContract / VerificationContract revision 可追溯到 Run/ChangeSet/Evidence;
//	AC2 实现上下文无法解析 held-out 内容;
//	AC3 visible 增量检查可运行;
//	AC4 held-out 最终 PASS 贡献 information independence;
//	AC5 派生状态只在必需证据之后到达 VERIFIED。
//
// 数据库不可达时 SKIP(不产出假失败)。验证命令均为真实进程执行:
// visible 检查 = go test(红转绿),held-out 检查 = git grep(内容 vs 冻结期望)。
package e2e

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"tally/internal/changeset"
	"tally/internal/command"
	"tally/internal/contextpack"
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

const (
	p47Project = core.ProjectID("proj-p1-47")
	p47TaskID  = "TASK-P1-47"
	// p47Greeting 是候选必须交付的行为:visible 测试在候选 workspace 内断言它,
	// held-out 内容把它固定为验证权威的期望。
	p47Greeting = "hello from P1-47"
	// p47HoldoutSecret 是只存在于 held-out 内容里的特征 token,AC2 泄漏扫描用。
	p47HoldoutSecret = "heldout-secret-047:TALLY-CONTROLLED-ONLY"
	p47Acceptance    = "AC-01"
	p47VisibleID     = "bnd-visible-greeting-test"
	p47HeldOutID     = "bnd-heldout-greeting-final"
	p47EnvName       = "p47-e2e-local"
)

const p47GoMod = "module tally-e2e/p47\n\ngo 1.23\n"

// p47WidgetTest 是 base 里的 visible 验收测试:引用候选才定义的 Greeting,
// 因此对 base 是红的、对候选是绿的 —— visible 增量检查因此真实有义。
const p47WidgetTest = `package widget

import "testing"

func TestGreeting(t *testing.T) {
	if Greeting != "` + p47Greeting + `" {
		t.Fatalf("Greeting = %q, want %q", Greeting, "` + p47Greeting + `")
	}
}
`

const p47BaseWidget = `package widget

// TASK-P1-47 之前的基线:Greeting 由候选交付。
`

const p47CandidateWidget = `package widget

// Greeting 是 TASK-P1-47 候选交付的行为。
const Greeting = "` + p47Greeting + `"
`

// p47HoldoutContent 是验证权威创作的 held-out 期望内容:Tally 控制的最终验证
// 把它注入一次性边界目录,冻结协议命令(git grep)核对期望确实声明了候选
// 必须满足的行为 —— 内容与期望不符时命令非零退出,检查真实可失败。
func p47HoldoutContent() []byte {
	return []byte("held-out expectation " + p47HoldoutSecret + "\n" +
		"the candidate must expose widget.Greeting == " + p47Greeting + "\n")
}

// ---- 合同固件 ---------------------------------------------------------------

func p47TaskContract() taskcontract.TaskContract {
	task := taskcontract.TaskContract{
		ID:        p47TaskID,
		Profile:   taskcontract.ProfileStandard,
		Title:     "P1-47 standard task end-to-end with held-out final verification",
		Objective: "Deliver widget.Greeting = " + p47Greeting,
		Scope:     taskcontract.Scope{AllowedPaths: []string{"internal/widget"}},
		Acceptance: []taskcontract.AcceptanceClause{{
			ID:                   p47Acceptance,
			Statement:            "widget exposes Greeting " + p47Greeting,
			IndependenceRequired: taskcontract.IndependenceInformation,
		}},
	}
	task.Hash = task.RevisionHash()
	return task
}

// p47FrozenContract 冻结 VerificationContract:同一 acceptance 两条 binding ——
// visible test binding(候选 workspace 内可执行的增量检查)与 held_out binding
// (只经 store/ref 引用,内容受 protection baseline 哈希保护)。independence
// 需求 information 属于 Acceptance Clause(Decision Lock 10),planned coverage
// 由 held_out binding + 非 implementation source basis 满足(§8.5.1)。
func p47FrozenContract(t *testing.T, ref holdout.Ref) *verification.Frozen {
	t.Helper()
	c := verification.Contract{
		TaskID:   p47TaskID,
		Revision: 1,
		Acceptances: []verification.Acceptance{{
			ID:                   p47Acceptance,
			Statement:            "widget exposes Greeting " + p47Greeting,
			IndependenceRequired: verification.IndependenceInformation,
			EvidenceBindings: []verification.Binding{
				{
					ID:               p47VisibleID,
					AuthoringMode:    verification.AuthoringPreCommit,
					VisibilityPolicy: verification.VisibilityVisible,
					Basis:            &verification.IndependenceBasis{Source: verification.SourceVerificationAuthority, Information: verification.InfoVisible},
					Evidence:         verification.EvidenceRef{Type: verification.EvidenceTest, Path: "internal/widget/widget_test.go", Selector: "TestGreeting"},
					ExecutionProtocol: &verification.ExecutionProtocol{
						Repetitions: 1,
						Aggregation: verification.AggregationAllPass,
						Command: &verification.CommandSpec{
							Argv:    []string{"go", "test", "./internal/widget/", "-run", "TestGreeting", "-count=1"},
							Timeout: verification.Duration(2 * time.Minute),
						},
					},
				},
				{
					ID:               p47HeldOutID,
					AuthoringMode:    verification.AuthoringPreCommit,
					VisibilityPolicy: verification.VisibilityHeldOut,
					Basis:            &verification.IndependenceBasis{Source: verification.SourceVerificationAuthority, Information: verification.InfoHeldOut},
					Evidence:         verification.EvidenceRef{Type: verification.EvidenceTest, Store: "holdout", Ref: ref.String()},
					Protection:       &verification.Protection{Mode: verification.ProtectionImmutable, BaselineHash: holdout.ContentDigest(p47HoldoutContent())},
					ExecutionProtocol: &verification.ExecutionProtocol{
						Repetitions: 1,
						Aggregation: verification.AggregationAllPass,
						Command:     &verification.CommandSpec{Argv: []string{"git", "grep", "--no-index", "-q", p47Greeting, "holdout-input"}},
					},
				},
			},
		}},
		RuntimeRequirements: verification.RuntimeRequirements{
			SessionIsolation:   true,
			HeldOutUnreachable: true,
			ConformanceClass:   "verification-isolation-v1",
		},
	}
	frozen, err := c.Freeze()
	if err != nil {
		t.Fatalf("冻结 VerificationContract: %v", err)
	}
	return frozen
}

// ---- 纵向切片驱动 ------------------------------------------------------------

// p47Scenario 是一次完整纵向切片的全部产物:codegen 侧事实 + 最终验证结论 +
// 关键身份坐标,供各验收测试断言。
type p47Scenario struct {
	runs          *runstore.Store
	pool          *pgxpool.Pool
	task          taskcontract.TaskContract
	frozen        *verification.Frozen
	pack          *contextpack.Pack
	base          string
	repo          string
	verifyWS      string
	holdoutRef    holdout.Ref
	holdoutDigest string
	codegen       workflow.Outcome
	codegenRunID  int64
	cs            *changeset.ChangeSet
	verify        *verification.FinalOutcome
	verifyRunID   int64
	envHash       string
}

// p47RunScenario 执行一次完整的标准 Task 端到端:真实 repo + FakeRuntime
// 工作流内核 + RunStoreSink 落库 + holdout Store 注入的最终验证编排。
func p47RunScenario(t *testing.T) p47Scenario {
	t.Helper()
	ctx := context.Background()
	runs, pool := e2eStore(t)
	s := p47Scenario{runs: runs, pool: pool, task: p47TaskContract()}
	if err := s.task.Validate(); err != nil {
		t.Fatalf("TaskContract 非法: %v", err)
	}

	// base repo:module + 候选位桩文件 + visible 验收测试(红)。
	s.repo = e2eNewRepo(t)
	s.base = p47Commit(t, s.repo, "go.mod", p47GoMod, "base: module")
	s.base = p47Commit(t, s.repo, "internal/widget/widget.go", p47BaseWidget, "base: widget stub")
	s.base = p47Commit(t, s.repo, "internal/widget/widget_test.go", p47WidgetTest, "base: visible acceptance test")

	// holdout Store:Tally 控制的文件系统根,机械上在每个受管 root 之外
	// (决策锁 11);验证权威写入期望内容,换取不透明 ref。
	storeRoot := filepath.Join(t.TempDir(), "holdout-store")
	wtRoot := filepath.Join(t.TempDir(), "codegen-worktrees")
	vRoot := filepath.Join(t.TempDir(), "verify-worktrees")
	hstore, err := holdout.NewFilesystemStore(storeRoot, s.repo, wtRoot, vRoot)
	if err != nil {
		t.Fatalf("打开 holdout store: %v", err)
	}
	s.holdoutRef, err = hstore.Put(ctx, p47HoldoutContent())
	if err != nil {
		t.Fatalf("写入 held-out 内容: %v", err)
	}
	s.holdoutDigest = holdout.ContentDigest(p47HoldoutContent())
	s.frozen = p47FrozenContract(t, s.holdoutRef)

	// ContextPack(P1-42 Select,与 worker 接线同一规则):context_hash 是
	// CodegenRun 的观测坐标,也是 AC2 的实现上下文本体。
	s.pack = p47ContextPack(t, s.repo, s.base, s.task, s.frozen)

	// ---- codegen:P1-39 单 Task 顺序内核,FakeRuntime 一轮完成声明固定候选 ----
	mgr, err := workspace.New(s.repo, wtRoot)
	if err != nil {
		t.Fatalf("workspace.New: %v", err)
	}
	texec, err := workflow.NewTaskExecution(workflow.TaskExecutionSpec{
		Project:              p47Project,
		Task:                 s.task,
		Role:                 "coding",
		Policy:               policy.DefaultExecutionPolicy(),
		BaseRevision:         s.base,
		Profile:              "default-coding",
		Runtime:              fake.New(fake.Config{}),
		Model:                "test-model",
		BehaviorRevision:     "behavior-1",
		VerificationRevision: s.frozen.Hash(),
		ContextPackHash:      s.pack.Hash(),
		Worktrees:            workspace.NewRunTable(mgr),
		Sink:                 workflow.NewRunStoreSink(runs),
		Hooks: workflow.Hooks{FixCandidate: func(_ context.Context, _ workflow.IterationView) (workflow.CandidateSnapshot, bool, error) {
			return workflow.CandidateSnapshot{
				CandidateRevision: s.base,
				Files:             []changeset.ChangedFile{{Path: "internal/widget/widget.go", Content: p47CandidateWidget}},
				Entities:          []string{"widget"},
			}, true, nil
		}},
	})
	if err != nil {
		t.Fatalf("NewTaskExecution: %v", err)
	}
	s.codegen, err = texec.Execute(ctx)
	if err != nil {
		t.Fatalf("codegen Execute: %v", err)
	}
	if s.codegen.Status != workflow.TaskCompleted || s.codegen.ChangeSet == nil {
		t.Fatalf("codegen status=%s changeset=%v, want completed + 候选", s.codegen.Status, s.codegen.ChangeSet)
	}
	s.cs = s.codegen.ChangeSet
	s.codegenRunID, err = strconv.ParseInt(s.cs.ProducerRun(), 10, 64)
	if err != nil || s.codegenRunID <= 0 {
		t.Fatalf("producer run 引用 %q 不是存储层 Run 主键", s.cs.ProducerRun())
	}

	// ---- 最终验证:P1-38 编排器 + holdout 注入(与 P1-43 接线同构) ----
	vmgr, err := workspace.New(s.repo, vRoot)
	if err != nil {
		t.Fatalf("workspace.New(verify): %v", err)
	}
	wt, err := vmgr.Create(ctx, s.cs.BaseRevision())
	if err != nil {
		t.Fatalf("建验证 workspace: %v", err)
	}
	t.Cleanup(func() { _ = vmgr.Cleanup(context.WithoutCancel(ctx), wt) })
	s.verifyWS = wt.Path
	// 候选物化:照写改后完整内容(与 cmd/worker materializeCandidate 同一约定)。
	for _, f := range s.cs.Files() {
		p := filepath.Join(wt.Path, filepath.FromSlash(filepath.Clean(f.Path)))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("物化候选(建目录)%s: %v", f.Path, err)
		}
		if err := os.WriteFile(p, []byte(f.Content), 0o644); err != nil {
			t.Fatalf("物化候选 %s: %v", f.Path, err)
		}
	}

	env := verification.EnvironmentIdentity{OS: runtime.GOOS, Arch: runtime.GOARCH, Toolchain: runtime.Version()}
	s.envHash, err = env.Hash()
	if err != nil {
		t.Fatalf("环境身份哈希: %v", err)
	}
	orch, err := verification.NewOrchestrator(verification.OrchestratorConfig{
		Runner:        runner.LocalRunner{},
		Policy:        policy.DefaultExecutionPolicy(),
		HoldoutReader: hstore,
		HoldoutRun:    p47HoldoutBoundary(runner.LocalRunner{}),
		Sink:          &p47VerifySink{runs: runs, revision: int32(s.frozen.Revision())},
		Recorder:      evidence.Recorder{Kind: evidence.RecorderVerificationRun, Actor: "tally-e2e/p1-47"},
	})
	if err != nil {
		t.Fatalf("构造最终验证编排器: %v", err)
	}
	s.verify, err = orch.Run(ctx, verification.FinalRunRequest{
		Project:          p47Project,
		Contract:         s.frozen,
		ChangeSet:        s.cs,
		EnvironmentName:  p47EnvName,
		Environment:      env,
		VersionCombo:     core.VersionCombo{"go": runtime.Version(), "runtime": "fake"},
		Runtime:          "fake",
		Workspace:        wt.Path,
		CandidateFixedAt: s.cs.CreatedAt(),
		Lineage:          []evidence.ChangeSetID{evidence.ChangeSetID(s.cs.ID())},
		Implementation: evidence.ImplementationProvenance{Contributors: []evidence.Contributor{
			{Name: "fake-session/p1-47", Role: "coding_agent"},
		}},
	})
	if err != nil {
		t.Fatalf("最终验证执行: %v", err)
	}
	s.verifyRunID = s.verify.RunID

	// Trust Gate 前提:整条流水线在 owner 主 worktree 零写入(§36.3)。
	if dirty := e2eGit(t, s.repo, "status", "--porcelain"); dirty != "" {
		t.Fatalf("owner 主 worktree 被污染:\n%s", dirty)
	}
	return s
}

// p47ContextPack 按 worker 接线同一规则组装实现上下文:base 树里 scope 之下的
// tracked 文件 + visible test binding 指定路径,交给 contextpack.Select 裁决。
func p47ContextPack(t *testing.T, repo, base string, task taskcontract.TaskContract, frozen *verification.Frozen) *contextpack.Pack {
	t.Helper()
	var candidates []contextpack.Candidate
	seen := map[string]bool{}
	add := func(p string) {
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		candidates = append(candidates, contextpack.Candidate{Path: p})
	}
	for _, p := range strings.Split(e2eGit(t, repo, "ls-tree", "-r", "--name-only", base), "\n") {
		if p = strings.TrimSpace(p); p47UnderRoots(task.Scope.AllowedPaths, p) {
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
	res, err := contextpack.Select(contextpack.SelectInput{Task: task, Verification: frozen, Candidates: candidates})
	if err != nil {
		t.Fatalf("contextpack.Select: %v", err)
	}
	return res.Pack
}

func p47UnderRoots(roots []string, path string) bool {
	for _, root := range roots {
		root = strings.TrimSuffix(root, "/")
		if path == root || strings.HasPrefix(path, root+"/") {
			return true
		}
	}
	return false
}

// ---- Tally 控制的 held-out 执行边界(P1-29 的控制面接线,与 P1-43 同构)----

func p47HoldoutBoundary(rt runner.Runner) verification.HoldoutRunFunc {
	return func(ctx context.Context, req verification.HoldoutRunRequest) (verification.HoldoutRunOutput, error) {
		if req.Command == nil {
			return verification.HoldoutRunOutput{}, errors.New("p47: held-out 冻结命令缺失")
		}
		// 一次性 Tally 控制目录:内容只以内存字节到达这里,物化后执行冻结命令;
		// 原始输出留在边界内,对外只有结构化 status + 摘要(§14.3)。
		dir, err := os.MkdirTemp("", "tally-p47-holdout-")
		if err != nil {
			return verification.HoldoutRunOutput{}, fmt.Errorf("p47: 建 holdout 边界目录: %w", err)
		}
		defer os.RemoveAll(dir)
		if err := os.WriteFile(filepath.Join(dir, "holdout-input"), req.Content, 0o600); err != nil {
			return verification.HoldoutRunOutput{}, fmt.Errorf("p47: 物化 held-out 内容: %w", err)
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

// ---- verification.RunSink → runstore 适配(P1-38 端口的集成接线,与 P1-43
// 的 v13VerificationSink 同一合同;测试不能 import package main,故在此重接)----

type p47VerifySink struct {
	runs     *runstore.Store
	revision int32
}

func (s *p47VerifySink) RecordRun(ctx context.Context, spec verification.RunSpec) (int64, error) {
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
		Status:            p47RunStatus(spec.Status),
	})
	if err != nil {
		return 0, fmt.Errorf("p47: 落 verification_run: %w", err)
	}
	return id, nil
}

func p47RunStatus(s verification.RunStatus) runstore.VerificationStatus {
	switch s {
	case verification.RunStatusPass:
		return runstore.VerificationPass
	case verification.RunStatusFail:
		return runstore.VerificationFail
	default:
		return runstore.VerificationError
	}
}

func (s *p47VerifySink) RecordFact(ctx context.Context, project core.ProjectID, fact verification.FactSpec) error {
	exit := int32(fact.ExitCode)
	if _, err := s.runs.RecordEvidence(ctx, project, runstore.EvidenceRecord{
		FactID:           fact.FactID,
		RunID:            fact.RunID,
		Task:             fact.Task,
		Acceptance:       fact.Acceptance,
		BindingID:        fact.BindingID,
		VisibilityPolicy: fact.VisibilityPolicy,
		Verdict:          evidence.Verdict(fact.Verdict),
		Recorder:         fact.Recorder,
		Command:          p47ToCommandSpec(fact.Command),
		ExitCode:         &exit,
		StdoutDigest:     fact.StdoutDigest,
		StderrDigest:     fact.StderrDigest,
		RecordedAt:       fact.RecordedAt,
	}); err != nil {
		return fmt.Errorf("p47: 落 evidence %s: %w", fact.FactID, err)
	}
	return nil
}

func (s *p47VerifySink) ValidApplicableFacts(ctx context.Context, scope evidence.Scope) ([]verification.PriorFact, error) {
	rows, err := s.runs.ApplicableValidEvidence(ctx, scope, s.revision)
	if err != nil {
		return nil, fmt.Errorf("p47: 读回既往 evidence: %w", err)
	}
	out := make([]verification.PriorFact, 0, len(rows))
	for _, r := range rows {
		switch evidence.Verdict(r.Verdict) {
		case evidence.VerdictPass:
			out = append(out, verification.PriorFact{BindingID: r.BindingID, Status: verification.EvidenceStatusPass, RecordedAt: r.RecordedAt})
		case evidence.VerdictFail:
			out = append(out, verification.PriorFact{BindingID: r.BindingID, Status: verification.EvidenceStatusFail, RecordedAt: r.RecordedAt})
		default:
			continue // error 判定无可用 verdict,不参与覆盖推导
		}
	}
	return out, nil
}

// p47ToCommandSpec 把冻结协议命令转成 command.CommandSpec(shell 恒为 false,
// 决策锁 6)。
func p47ToCommandSpec(spec *verification.CommandSpec) *command.CommandSpec {
	if spec == nil {
		return nil
	}
	var env map[string]string
	for _, kv := range spec.Env {
		if k, v, ok := strings.Cut(kv, "="); ok {
			if env == nil {
				env = make(map[string]string)
			}
			env[k] = v
		}
	}
	return &command.CommandSpec{
		Argv:    append([]string(nil), spec.Argv...),
		Cwd:     spec.Dir,
		Env:     env,
		Timeout: time.Duration(spec.Timeout),
		Shell:   false,
	}
}

// p47Commit 在 repo 内写入(可含子目录的)文件并提交,返回提交后 HEAD。
// 与 e2eCommit 同一语义,但先建父目录:base repo 的文件带 internal/widget/ 前缀。
func p47Commit(t *testing.T, repo, name, content, msg string) string {
	t.Helper()
	p := filepath.Join(repo, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("建目录 %s: %v", filepath.Dir(name), err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	e2eGit(t, repo, "add", ".")
	e2eGit(t, repo, "commit", "-m", msg)
	return e2eGit(t, repo, "rev-parse", "HEAD")
}

// ---- AC1:合同 revision 追溯到 Run / ChangeSet / Evidence --------------------

func TestV13StandardTaskRevisionsTraceToRunChangeSetEvidence(t *testing.T) {
	s := p47RunScenario(t)
	ctx := context.Background()

	// CodegenRun 行携带 TaskContract / VerificationContract / ContextPack 身份。
	run, err := s.runs.CodegenRun(ctx, s.codegenRunID)
	if err != nil {
		t.Fatalf("CodegenRun(%d): %v", s.codegenRunID, err)
	}
	if run.TaskRevision != s.task.Hash {
		t.Fatalf("codegen_run.task_revision = %s, want TaskContract revision %s(AC1)", run.TaskRevision, s.task.Hash)
	}
	if run.VerificationRevision != s.frozen.Hash() {
		t.Fatalf("codegen_run.verification_revision = %s, want 冻结契约 %s(AC1)", run.VerificationRevision, s.frozen.Hash())
	}
	if run.ContextPackHash != s.pack.Hash() {
		t.Fatalf("codegen_run.context_pack_hash = %s, want %s(AC1)", run.ContextPackHash, s.pack.Hash())
	}
	if run.Status != runstore.RunCompleted || run.ResultPatchHash != string(s.cs.ID()) {
		t.Fatalf("codegen_run status=%s result=%q, want completed + 快照身份 %s(AC1)", run.Status, run.ResultPatchHash, s.cs.ID())
	}

	// ChangeSet 回指产生它的持久化 Run 行(快照可追溯到过程记录)。
	if s.cs.ProducerRun() != strconv.FormatInt(s.codegenRunID, 10) {
		t.Fatalf("ChangeSet producer = %s, want Run 主键 %d(AC1)", s.cs.ProducerRun(), s.codegenRunID)
	}

	// VerificationRun 精确绑定冻结契约 revision + 候选身份(§8.13)。
	vr, err := s.runs.VerificationRun(ctx, p47Project, s.verifyRunID)
	if err != nil {
		t.Fatalf("VerificationRun(%d): %v", s.verifyRunID, err)
	}
	if vr.ContractRevision != int32(s.frozen.Revision()) || vr.ContractHash != s.frozen.Hash() || vr.ChangeSetID != s.cs.ID() {
		t.Fatalf("verification_run 绑定 rev=%d hash=%s cs=%s, want rev=%d hash=%s cs=%s(AC1)",
			vr.ContractRevision, vr.ContractHash, vr.ChangeSetID, s.frozen.Revision(), s.frozen.Hash(), s.cs.ID())
	}
	if vr.Status != runstore.VerificationPass || vr.EnvName != p47EnvName || vr.EnvDefinitionHash != s.envHash {
		t.Fatalf("verification_run status=%s env=%s/%s, want pass + %s/%s(AC1)", vr.Status, vr.EnvName, vr.EnvDefinitionHash, p47EnvName, s.envHash)
	}

	// Evidence 事实:每条 binding 恰好一条,scope 从 Run 行取得契约与候选身份。
	evs, err := s.runs.EvidenceForRun(ctx, p47Project, s.verifyRunID)
	if err != nil {
		t.Fatalf("EvidenceForRun: %v", err)
	}
	if len(evs) != 2 || len(s.verify.FactIDs) != 2 {
		t.Fatalf("evidence 行数=%d FactIDs=%d, want 2/2(AC1)", len(evs), len(s.verify.FactIDs))
	}
	byBinding := map[string]runstore.Evidence{}
	for _, e := range evs {
		byBinding[e.BindingID] = e
	}
	vis, ok := byBinding[p47VisibleID]
	if !ok || vis.VisibilityPolicy != verification.VisibilityVisible || vis.Verdict != evidence.VerdictPass {
		t.Fatalf("visible evidence = %+v ok=%v, want visible PASS(AC1)", vis, ok)
	}
	ho, ok := byBinding[p47HeldOutID]
	if !ok || ho.VisibilityPolicy != verification.VisibilityHeldOut || ho.Verdict != evidence.VerdictPass {
		t.Fatalf("held-out evidence = %+v ok=%v, want held_out PASS(AC1)", ho, ok)
	}
	for _, e := range []runstore.Evidence{vis, ho} {
		if e.Scope.Task != evidence.TaskID(p47TaskID) || e.Scope.Acceptance != evidence.AcceptanceID(p47Acceptance) ||
			e.Scope.Verification != evidence.VerificationRevision(s.frozen.Hash()) || e.Scope.ChangeSet != evidence.ChangeSetID(s.cs.ID()) {
			t.Fatalf("evidence %s scope = %+v, 未精确绑定 task/acceptance/contract/changeset(AC1)", e.FactID, e.Scope)
		}
	}
}

// ---- AC2:实现上下文无法解析 held-out 内容 ----------------------------------

func TestV13ImplementationContextCannotResolveHeldOutContent(t *testing.T) {
	s := p47RunScenario(t)

	// 1) ContextPack(实现权威的唯一上下文入口)的 redacted projection:
	// held_out binding 只暴露 evidence type,定位字段与 binding id 全部缺席(§8.4.3)。
	var heldOutProj, visibleProj *contextpack.EvidenceProjection
	for i := range s.pack.Verification {
		proj := &s.pack.Verification[i]
		if proj.ID != p47Acceptance {
			continue
		}
		for j := range proj.Evidence {
			switch proj.Evidence[j].Visibility {
			case verification.VisibilityHeldOut:
				heldOutProj = &proj.Evidence[j]
			case verification.VisibilityVisible:
				visibleProj = &proj.Evidence[j]
			}
		}
	}
	if heldOutProj == nil || visibleProj == nil {
		t.Fatalf("contextpack 缺 projection: held_out=%v visible=%v(AC2)", heldOutProj, visibleProj)
	}
	if heldOutProj.BindingID != "" || heldOutProj.Store != "" || heldOutProj.Ref != "" || heldOutProj.Path != "" {
		t.Fatalf("held-out projection 携带定位字段: %+v(AC2)", *heldOutProj)
	}
	if visibleProj.BindingID != p47VisibleID || visibleProj.Path != "internal/widget/widget_test.go" {
		t.Fatalf("visible projection = %+v, want binding/path 保留(AC2)", *visibleProj)
	}

	// 2) 整包序列化泄漏扫描:holdout ref 与内容 token 都不得出现在实现上下文里。
	data, err := json.Marshal(s.pack)
	if err != nil {
		t.Fatalf("序列化 contextpack: %v", err)
	}
	serialized := string(data)
	if strings.Contains(serialized, s.holdoutRef.String()) {
		t.Fatal("contextpack 泄漏 holdout ref(AC2)")
	}
	if strings.Contains(serialized, p47HoldoutSecret) || strings.Contains(serialized, s.holdoutDigest) {
		t.Fatal("contextpack 泄漏 held-out 内容/摘要(AC2)")
	}
	if !strings.Contains(serialized, "internal/widget/widget_test.go") || s.pack.Hash() == "" {
		t.Fatal("contextpack 缺 visible 测试或 context_hash(AC2 对照)")
	}

	// 3) Ref 不是路径:ref 词法拒绝路径分隔符 → 任何 held-out 引用都无法在
	// 实现 workspace 内寻址文件;当前 ref 在候选 workspace 上同样解析不出内容。
	if _, err := holdout.ParseRef("internal/widget/secret.txt"); !errors.Is(err, holdout.ErrInvalidRef) {
		t.Fatalf("路径形态 ref 被 ParseRef 接受: %v(AC2)", err)
	}
	ref, err := holdout.ParseRef(s.holdoutRef.String())
	if err != nil {
		t.Fatalf("holdout ref 非法: %v", err)
	}
	if strings.ContainsAny(ref.String(), `/\`) {
		t.Fatalf("holdout ref 含路径分隔符: %s(AC2)", ref)
	}
	if _, err := os.Stat(filepath.Join(s.verifyWS, ref.String())); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("holdout ref %s 在实现 workspace 内可解析: %v(AC2)", ref, err)
	}

	// 4) 实现侧可见信号(最终验证的唯一实现投影)不含 held-out 判定/内容锚点。
	sig := s.verify.RedactedSignal()
	if sig.Status != verification.FinalVerified || len(sig.Holdouts) != 1 || sig.Holdouts[0].AcceptanceID != p47Acceptance {
		t.Fatalf("redacted signal = %+v, want verified + 一条 AC-01 held-out 完成项(AC2)", sig)
	}
	projected := fmt.Sprint(sig.Holdouts[0])
	if strings.Contains(projected, s.holdoutRef.String()) || strings.Contains(projected, p47HoldoutSecret) ||
		strings.Contains(projected, s.holdoutDigest) || strings.Contains(projected, string(verification.EvidenceStatusPass)) {
		t.Fatalf("实现侧信号携带 held-out 锚点: %s(AC2)", projected)
	}

	// 5) 冻结契约本体:held_out binding 没有 repo path(§8.4.1 冻结校验)。
	for _, acc := range s.frozen.Contract().Acceptances {
		for _, b := range acc.EvidenceBindings {
			if b.VisibilityPolicy == verification.VisibilityHeldOut && b.Evidence.Path != "" {
				t.Fatalf("held-out binding %s 携带 repo path %s(AC2)", b.ID, b.Evidence.Path)
			}
		}
	}
}

// ---- AC3:visible 增量检查可运行 ---------------------------------------------

func TestV13VisibleIncrementalCheckRuns(t *testing.T) {
	s := p47RunScenario(t)
	ctx := context.Background()

	// 编排器真实执行了 visible 冻结协议:候选 workspace 内 go test 红转绿。
	var bo *verification.BindingOutcome
	for i := range s.verify.Bindings {
		if s.verify.Bindings[i].BindingID == p47VisibleID {
			bo = &s.verify.Bindings[i]
		}
	}
	if bo == nil {
		t.Fatal("visible binding 未被执行(AC3)")
	}
	if bo.Err != "" {
		t.Fatalf("visible binding 执行出错: %s(AC3)", bo.Err)
	}
	if bo.Status != verification.EvidenceStatusPass || bo.ExitCode != 0 || bo.Repetitions != 1 {
		t.Fatalf("visible binding status=%s exit=%d reps=%d, want pass/0/1(AC3)", bo.Status, bo.ExitCode, bo.Repetitions)
	}
	if bo.Visibility != verification.VisibilityVisible || bo.ContextHash == "" || bo.Command == nil {
		t.Fatalf("visible binding 可见性/context 身份/命令记录不完整: %+v(AC3)", bo)
	}

	// 证据行:结构化命令与退出码被原样落库(可追溯的观察事实)。
	e, err := s.runs.Evidence(ctx, p47Project, evidence.FactID(fmt.Sprintf("vrun-%d-%s", s.verifyRunID, p47VisibleID)))
	if err != nil {
		t.Fatalf("读 visible evidence: %v", err)
	}
	if e.Command == nil || len(e.Command.Argv) == 0 || e.Command.Argv[0] != "go" || e.Command.Shell {
		t.Fatalf("visible evidence 命令 = %+v, want 结构化非 shell go 命令(AC3)", e.Command)
	}
	if e.ExitCode == nil || *e.ExitCode != 0 || e.Verdict != evidence.VerdictPass {
		t.Fatalf("visible evidence exit/verdict = %v/%s, want 0/pass(AC3)", e.ExitCode, e.Verdict)
	}
}

// ---- AC4:held-out 最终 PASS 贡献 information independence -------------------

func p47IndependenceContext(s p47Scenario) verification.IndependenceContext {
	return verification.IndependenceContext{
		Project:          p47Project,
		Verification:     evidence.VerificationRevision(s.frozen.Hash()),
		ChangeSet:        evidence.ChangeSetID(s.cs.ID()),
		CandidateFixedAt: s.cs.CreatedAt(),
		Lineage:          []evidence.ChangeSetID{evidence.ChangeSetID(s.cs.ID())},
		Implementation: evidence.ImplementationProvenance{Contributors: []evidence.Contributor{
			{Name: "fake-session/p1-47", Role: "coding_agent"},
		}},
	}
}

func p47VerdictRecord(s p47Scenario, binding string, status verification.EvidenceStatus) evidence.VerdictRecord {
	return evidence.VerdictRecord{
		Binding: evidence.BindingRef{
			Project:      p47Project,
			Verification: evidence.VerificationRevision(s.frozen.Hash()),
			BindingID:    binding,
		},
		Task:       evidence.TaskID(p47TaskID),
		ChangeSet:  evidence.ChangeSetID(s.cs.ID()),
		Verdict:    evidence.Verdict(status),
		Valid:      true,
		RecordedAt: s.verify.FinishedAt,
	}
}

func TestV13HeldOutFinalPassContributesInformationIndependence(t *testing.T) {
	s := p47RunScenario(t)
	ctx := context.Background()

	// 判定事实(Control/Review 通道):held-out PASS 精确绑定候选与冻结契约,
	// 内容摘要与冻结 protection baseline 一致(INV-VER-03)。
	if len(s.verify.Holdout) != 1 {
		t.Fatalf("held-out 判定记录数 = %d, want 1(AC4)", len(s.verify.Holdout))
	}
	h := s.verify.Holdout[0]
	if h.BindingID != p47HeldOutID || h.Status != verification.EvidenceStatusPass {
		t.Fatalf("held-out 判定 = %s/%s, want %s/pass(AC4)", h.BindingID, h.Status, p47HeldOutID)
	}
	if h.CandidateID != string(s.cs.ID()) || h.ContractHash != s.frozen.Hash() || h.ContractRevision != s.frozen.Revision() {
		t.Fatalf("held-out 判定绑定 candidate=%s contract=%s/%d, want %s/%s/%d(AC4)",
			h.CandidateID, h.ContractHash, h.ContractRevision, s.cs.ID(), s.frozen.Hash(), s.frozen.Revision())
	}
	if h.ContentDigest != s.holdoutDigest {
		t.Fatalf("held-out 内容摘要 = %s, want %s(AC4)", h.ContentDigest, s.holdoutDigest)
	}

	ictx := p47IndependenceContext(s)

	// 反事实:只有 visible PASS 的世界 —— visible binding 恒无 information
	// independence(§8.4.1),acceptance 的 information 要求不被覆盖。
	visibleOnly, err := verification.DeriveIndependenceCoverage(s.frozen, ictx, []verification.IndependenceEvidence{{
		BindingID: p47VisibleID,
		Revision:  s.frozen.Revision(),
		Verdicts:  []evidence.VerdictRecord{p47VerdictRecord(s, p47VisibleID, verification.EvidenceStatusPass)},
	}})
	if err != nil {
		t.Fatalf("推导 visible-only 覆盖: %v", err)
	}
	if visibleOnly.Complete || visibleOnly.Acceptances[0].Covered {
		t.Fatalf("visible 单独即满足 information independence: %+v(AC4)", visibleOnly.Acceptances[0])
	}

	// 加入 held-out final PASS:恰好由该 binding 满足 information 要求。
	withHoldout, err := verification.DeriveIndependenceCoverage(s.frozen, ictx, []verification.IndependenceEvidence{
		{BindingID: p47VisibleID, Revision: s.frozen.Revision(), Verdicts: []evidence.VerdictRecord{p47VerdictRecord(s, p47VisibleID, verification.EvidenceStatusPass)}},
		{BindingID: p47HeldOutID, Revision: s.frozen.Revision(), Verdicts: []evidence.VerdictRecord{p47VerdictRecord(s, p47HeldOutID, verification.EvidenceStatusPass)}},
	})
	if err != nil {
		t.Fatalf("推导含 held-out 覆盖: %v", err)
	}
	if !withHoldout.Complete || !withHoldout.Acceptances[0].Covered {
		t.Fatalf("held-out PASS 未补全 independence 覆盖: complete=%v acc=%+v(AC4)", withHoldout.Complete, withHoldout.Acceptances[0])
	}
	satisfied := withHoldout.Acceptances[0].SatisfiedBy
	if len(satisfied) != 1 || satisfied[0] != p47HeldOutID {
		t.Fatalf("information independence 由 %v 满足, want [%s](AC4)", satisfied, p47HeldOutID)
	}
	for _, b := range withHoldout.Acceptances[0].Bindings {
		if b.BindingID == p47HeldOutID && (!b.Applicable || !b.SourceIndependent || !b.InformationIndependent || !b.Qualifies) {
			t.Fatalf("held-out binding 独立性推导 = %+v, want applicable+source+information(AC4)", b)
		}
	}

	// 编排器当轮结论与上述推导一致:整份契约 effective independence 完整。
	if !s.verify.Independence.Complete {
		t.Fatalf("编排器 independence 报告不完整: %+v(AC4)", s.verify.Independence.Acceptances)
	}

	// 事实面:held-out PASS 是 valid + applicable 的 gate 输入,visibility 恒为 held_out。
	rows, err := s.runs.ApplicableValidEvidence(ctx, evidence.Scope{
		Project:      p47Project,
		Task:         evidence.TaskID(p47TaskID),
		Acceptance:   evidence.AcceptanceID(p47Acceptance),
		Verification: evidence.VerificationRevision(s.frozen.Hash()),
		ChangeSet:    evidence.ChangeSetID(s.cs.ID()),
	}, int32(s.frozen.Revision()))
	if err != nil {
		t.Fatalf("ApplicableValidEvidence: %v", err)
	}
	heldOutFacts := 0
	for _, r := range rows {
		if r.BindingID == p47HeldOutID {
			heldOutFacts++
			if r.VisibilityPolicy != verification.VisibilityHeldOut || r.Verdict != evidence.VerdictPass {
				t.Fatalf("held-out 事实 = %s/%s, want held_out/pass(AC4)", r.VisibilityPolicy, r.Verdict)
			}
		}
	}
	if heldOutFacts != 1 {
		t.Fatalf("held-out 事实行数 = %d, want 1(AC4)", heldOutFacts)
	}
	if n := countRows(t, s.pool, "SELECT count(*) FROM verification_evidence WHERE project_id=$1 AND visibility_policy='held_out' AND verdict='pass'", string(p47Project)); n != 1 {
		t.Fatalf("数据库 held_out PASS 事实行数 = %d, want 1(AC4)", n)
	}
}

// ---- AC5:派生状态只在必需证据之后到达 VERIFIED ------------------------------

func TestV13DerivedStateReachesVerifiedOnlyAfterEvidence(t *testing.T) {
	s := p47RunScenario(t)

	human := evidence.Recorder{Kind: evidence.RecorderHuman, Actor: "owner"}
	rec := evidence.Recorder{Kind: evidence.RecorderVerificationRun, Actor: "tally-e2e/p1-47"}
	at := func(min int) time.Time { return time.Date(2026, 9, 6, 10, min, 0, 0, time.UTC) }
	runRef := strconv.FormatInt(s.codegenRunID, 10)
	facts := []taskstate.Fact{
		{ID: "p47-f1-planned", Kind: taskstate.FactTaskPlanned, At: at(0), Recorder: human, RevisionRef: s.task.Hash},
		{ID: "p47-f2-frozen", Kind: taskstate.FactVerificationFrozen, At: at(1), Recorder: human, RevisionRef: s.frozen.Hash()},
		{ID: "p47-f3-run", Kind: taskstate.FactCodegenRunStarted, At: at(2), Recorder: rec, RunID: runRef},
		{ID: "p47-f4-candidate", Kind: taskstate.FactChangeSetProduced, At: at(3), Recorder: rec, RunID: runRef, ChangeSet: evidence.ChangeSetID(s.cs.ID())},
	}
	verificationRunFact := taskstate.Fact{
		ID: "p47-f5-vrun", Kind: taskstate.FactVerificationRunStarted, At: at(4), Recorder: rec,
		RunID: strconv.FormatInt(s.verifyRunID, 10), ChangeSet: evidence.ChangeSetID(s.cs.ID()),
	}

	// (a) 候选已固定、验证证据未发生:CANDIDATE,不是 VERIFIED;缺口可追溯到
	// 功能覆盖 / independence 覆盖 / 验证 Run 三项门槛(INV-STATE-02)。
	resA, err := taskstate.Derive(taskstate.Snapshot{Contract: s.frozen, Facts: facts})
	if err != nil {
		t.Fatalf("推导(a): %v", err)
	}
	if resA.State != taskstate.StateCandidate || resA.VerificationClass != taskstate.ClassNone {
		t.Fatalf("(a) state=%s class=%s, want CANDIDATE/无(AC5)", resA.State, resA.VerificationClass)
	}
	requirements := map[taskstate.Requirement]bool{}
	for _, g := range resA.Gaps {
		requirements[g.Requirement] = true
	}
	for _, req := range []taskstate.Requirement{taskstate.ReqFunctionalCoverage, taskstate.ReqIndependenceCoverage, taskstate.ReqVerificationRun} {
		if !requirements[req] {
			t.Fatalf("(a) 缺口未列出 %s: %+v(AC5)", req, resA.Gaps)
		}
	}

	// (b) 验证 Run 事实出现、覆盖报告仍缺:VALIDATING,仍未 VERIFIED。
	withRun := append(append([]taskstate.Fact{}, facts...), verificationRunFact)
	resB, err := taskstate.Derive(taskstate.Snapshot{Contract: s.frozen, Facts: withRun})
	if err != nil {
		t.Fatalf("推导(b): %v", err)
	}
	if resB.State != taskstate.StateValidating {
		t.Fatalf("(b) state = %s, want VALIDATING(AC5)", resB.State)
	}

	// (c) 当前契约绑定的功能 + independence 覆盖报告(来自本 E2E 的最终验证
	// 事实)到位:VERIFIED,strict_satisfied,候选身份正确。
	resC, err := taskstate.Derive(taskstate.Snapshot{
		Contract:     s.frozen,
		Facts:        withRun,
		Functional:   &s.verify.Functional,
		Independence: &s.verify.Independence,
	})
	if err != nil {
		t.Fatalf("推导(c): %v", err)
	}
	if resC.State != taskstate.StateVerified {
		t.Fatalf("(c) state = %s gaps=%+v, want VERIFIED(AC5)", resC.State, resC.Gaps)
	}
	if resC.VerificationClass != taskstate.ClassStrictSatisfied {
		t.Fatalf("(c) verification class = %s, want strict_satisfied(AC5)", resC.VerificationClass)
	}
	if resC.CurrentCandidate != evidence.ChangeSetID(s.cs.ID()) || resC.ContractHash != s.frozen.Hash() {
		t.Fatalf("(c) candidate=%s contract=%s, want %s/%s(AC5)", resC.CurrentCandidate, resC.ContractHash, s.cs.ID(), s.frozen.Hash())
	}
	basis := strings.Join(resC.Basis, "\n")
	for _, want := range []string{
		"functional_report:" + s.frozen.Hash(),
		"independence_report:" + s.frozen.Hash(),
		"fact:p47-f5-vrun",
	} {
		if !strings.Contains(basis, want) {
			t.Fatalf("(c) 结论追溯缺 %s: %v(AC5)", want, resC.Basis)
		}
	}

	// (d) 绑定其它 revision 的覆盖报告对当前状态不可见(§8.8.2):推不到 VERIFIED。
	staleFunctional, staleIndependence := s.verify.Functional, s.verify.Independence
	staleFunctional.ContractHash = "sha256:stale-other-revision"
	staleIndependence.ContractHash = "sha256:stale-other-revision"
	resD, err := taskstate.Derive(taskstate.Snapshot{
		Contract:     s.frozen,
		Facts:        withRun,
		Functional:   &staleFunctional,
		Independence: &staleIndependence,
	})
	if err != nil {
		t.Fatalf("推导(d): %v", err)
	}
	if resD.State == taskstate.StateVerified {
		t.Fatal("(d) 过期 revision 的覆盖报告推出了 VERIFIED(AC5)")
	}
}
