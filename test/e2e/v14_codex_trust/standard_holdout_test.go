//go:build integration

// P0-04 门禁端到端主体:真实 codex standard Task + held-out 最终验证信任路径。
//
// 结构与 test/e2e 的 P1-47 纵向切片同构,唯一本质差异是 codegen 侧的 runtime:
// FakeRuntime 的固定候选 → 真实 CodexRuntime(deepseek-v4-flash,经 ADR-0003
// ProviderEndpoint 播种 + auth.json 移交)在一次性 worktree 内真实产出候选。
//
// 编排说明(为什么不走 P1-39 工作流内核,如实记录的两个真实表面事实):
//
//  1. codex-cli 0.153.0 对所有「未知模型名」的每个 turn 都先发一条
//     item.type=="error" 的非致命警告 item("Model metadata for
//     `deepseek-v4-flash` not found ...",实测 turn 仍正常完成、exit 0),
//     而适配器的冻结分类逻辑(process.go classifyTurn,P0-03 所有)把任何
//     error item 一律归为 RunFailed claim —— 内核因此永远到不了候选固定点;
//  2. 同一会话的后续 turn 走 `codex exec resume <id>`,而 0.153.0 的 resume
//     子命令拒绝其后的旗标(clap: unexpected argument '-C';argv 面同样
//     冻结),第二轮必然 exit 2。
//
// 两者都不属于本任务可改的 allowed path。故本测试以生产原语直连:
// 真实适配器会话(workspace.Manager 的一次性 worktree)→ 真实 claim 原样
// 记录(claim ≠ Delivery Fact,INV-AUTH-01;候选从不取自 claim)→ 候选从
// worktree 事实(git status + 内容回读,零注入)固定 → changeset/runstore
// 生产存储接口落库(含 observed_identity 实际身份)→ 冻结验证编排器(visible
// go test + held_out git grep)裁决。候选是否可接受完全由验证决定(锁 9)。
//
// 会话隔离前提:Session Isolation Canary 先行(见 canary_test.go);只有
// canary 通过才允许以 SessionIsolationVerified=true 构造生产 runtime 并断言
// session.isolation 声明(DECISION_LOCKS #22/#24)。
//
// 模型用量有界:本测试一个真实标准回合(策略性不重试;两个冻结事实使内核
// 式重试必然无效),held-out 最终验证是确定性命令,不消耗模型;canary 全进程
// 至多一轮。
package codextrust

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

	"tally/internal/agentruntime"
	"tally/internal/changeset"
	"tally/internal/command"
	"tally/internal/contextpack"
	"tally/internal/core"
	"tally/internal/evidence"
	"tally/internal/holdout"
	"tally/internal/measurement"
	"tally/internal/policy"
	"tally/internal/runner"
	"tally/internal/runstore"
	"tally/internal/taskcontract"
	"tally/internal/verification"
	"tally/internal/workspace"
	"tally/runtimes/codex"
)

const (
	p04Project = core.ProjectID("proj-p0-04")
	p04TaskID  = "TASK-P0-04"
	// p04Greeting 是候选必须交付的行为:visible 测试在候选 workspace 内断言
	// 它,held-out 内容把它固定为验证权威的期望。
	p04Greeting = "hello from P0-04"
	// p04HoldoutSecret 是只存在于 held-out 内容里的特征 token,泄漏扫描用。
	p04HoldoutSecret = "heldout-secret-p04:TALLY-CONTROLLED-ONLY"
	p04Acceptance    = "AC-01"
	p04VisibleID     = "bnd-visible-greeting-test"
	p04HeldOutID     = "bnd-heldout-greeting-final"
	p04EnvName       = "p04-e2e-local"
)

const p04GoMod = "module tally-e2e/p04\n\ngo 1.23\n"

// p04GreetingTest 是 base 里的 visible 验收测试:引用候选才定义的 Greeting,
// 对 base 是红的(编译失败)、对候选是绿的 —— visible 增量检查真实有义。
const p04GreetingTest = `package greeting

import "testing"

func TestGreeting(t *testing.T) {
	if Greeting != "` + p04Greeting + `" {
		t.Fatalf("Greeting = %q, want %q", Greeting, "` + p04Greeting + `")
	}
}
`

const p04BaseGreeting = `package greeting

// TASK-P0-04 之前的基线:Greeting 由候选交付。
`

// p04Instruction 与工作流内核 instructionFor 的产物逐字同构(合同目标 +
// 验收条款)。
func p04Instruction(task taskcontract.TaskContract) string {
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
	return b.String()
}

// p04HoldoutContent 是验证权威创作的 held-out 期望内容:最终验证把它注入
// Tally 控制的一次性边界目录,冻结协议命令(git grep)核对候选必须满足的
// 行为 —— 内容与期望不符时命令非零退出,检查真实可失败。
func p04HoldoutContent() []byte {
	return []byte("held-out expectation " + p04HoldoutSecret + "\n" +
		"the candidate must expose greeting.Greeting == " + p04Greeting + "\n")
}

// ---- 合同固件 ---------------------------------------------------------------

func p04TaskContract() taskcontract.TaskContract {
	task := taskcontract.TaskContract{
		ID:      p04TaskID,
		Profile: taskcontract.ProfileStandard,
		Title:   "P0-04 real codex standard task with held-out final verification",
		// Objective 直接可执行:真实 codex 回合据此完成实现。验收裁决不在
		// instruction 里 —— 候选是否可接受由冻结的 visible/held-out 验证决定。
		Objective: "Make `go test ./greeting/` pass in this workspace: the greeting package must define `const Greeting = \"" + p04Greeting + "\"`. Do not modify greeting/greeting_test.go; work only inside the greeting/ directory.",
		Scope:     taskcontract.Scope{AllowedPaths: []string{"greeting"}},
		Acceptance: []taskcontract.AcceptanceClause{{
			ID:                   p04Acceptance,
			Statement:            "greeting exposes Greeting " + p04Greeting,
			IndependenceRequired: taskcontract.IndependenceInformation,
		}},
	}
	task.Hash = task.RevisionHash()
	return task
}

// p04FrozenContract 冻结 VerificationContract:同一 acceptance 两条 binding ——
// visible test binding(候选 workspace 内可执行的增量检查)与 held_out binding
// (只经 store/ref 引用,内容受 protection baseline 哈希保护)。
func p04FrozenContract(t *testing.T, ref holdout.Ref) *verification.Frozen {
	t.Helper()
	c := verification.Contract{
		TaskID:   p04TaskID,
		Revision: 1,
		Acceptances: []verification.Acceptance{{
			ID:                   p04Acceptance,
			Statement:            "greeting exposes Greeting " + p04Greeting,
			IndependenceRequired: verification.IndependenceInformation,
			EvidenceBindings: []verification.Binding{
				{
					ID:               p04VisibleID,
					AuthoringMode:    verification.AuthoringPreCommit,
					VisibilityPolicy: verification.VisibilityVisible,
					Basis:            &verification.IndependenceBasis{Source: verification.SourceVerificationAuthority, Information: verification.InfoVisible},
					Evidence:         verification.EvidenceRef{Type: verification.EvidenceTest, Path: "greeting/greeting_test.go", Selector: "TestGreeting"},
					ExecutionProtocol: &verification.ExecutionProtocol{
						Repetitions: 1,
						Aggregation: verification.AggregationAllPass,
						Command: &verification.CommandSpec{
							Argv:    []string{"go", "test", "./greeting/", "-run", "TestGreeting", "-count=1"},
							Timeout: verification.Duration(3 * time.Minute),
						},
					},
				},
				{
					ID:               p04HeldOutID,
					AuthoringMode:    verification.AuthoringPreCommit,
					VisibilityPolicy: verification.VisibilityHeldOut,
					Basis:            &verification.IndependenceBasis{Source: verification.SourceVerificationAuthority, Information: verification.InfoHeldOut},
					Evidence:         verification.EvidenceRef{Type: verification.EvidenceTest, Store: "holdout", Ref: ref.String()},
					Protection:       &verification.Protection{Mode: verification.ProtectionImmutable, BaselineHash: holdout.ContentDigest(p04HoldoutContent())},
					ExecutionProtocol: &verification.ExecutionProtocol{
						Repetitions: 1,
						Aggregation: verification.AggregationAllPass,
						Command:     &verification.CommandSpec{Argv: []string{"git", "grep", "--no-index", "-q", p04Greeting, "holdout-input"}},
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

// p04ContextPack 按 worker 接线同一规则组装实现上下文:base 树里 scope 之下的
// tracked 文件 + visible test binding 指定路径,交给 contextpack.Select 裁决。
func p04ContextPack(t *testing.T, repo, base string, task taskcontract.TaskContract, frozen *verification.Frozen) *contextpack.Pack {
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
	for _, p := range strings.Split(p04Git(t, repo, "ls-tree", "-r", "--name-only", base), "\n") {
		if p = strings.TrimSpace(p); p04UnderRoots(task.Scope.AllowedPaths, p) {
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

func p04UnderRoots(roots []string, path string) bool {
	for _, root := range roots {
		root = strings.TrimSuffix(root, "/")
		if path == root || strings.HasPrefix(path, root+"/") {
			return true
		}
	}
	return false
}

// ---- 真实候选固定(只读 worktree,不注入内容)---------------------------------

// p04CandidateFromWorktree 读取一次性 worktree 的真实文件变化(git status
// --porcelain + 内容回读),有变化才有候选。它与 claim 无关:候选从不取自
// agent 自述(INV-AUTH-01),这里也没有任何期望内容可以注入。
func p04CandidateFromWorktree(ctx context.Context, ws string) (workflowCandidate, bool, error) {
	status, err := p04GitRaw(ctx, ws, "status", "--porcelain")
	if err != nil {
		return workflowCandidate{}, false, fmt.Errorf("p04: 读 worktree 状态: %w", err)
	}
	var files []changeset.ChangedFile
	for _, line := range strings.Split(status, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" || len(line) < 4 {
			continue
		}
		code, path := line[:2], strings.TrimSpace(line[3:])
		if code == "!!" {
			continue // ignored:不属于候选
		}
		if strings.Contains(path, "->") {
			return workflowCandidate{}, false, fmt.Errorf("p04: rename 形态变化超出本 fixture 支持面: %q", line)
		}
		if strings.HasPrefix(code, "D") || strings.HasSuffix(code, "D") {
			return workflowCandidate{}, false, fmt.Errorf("p04: 删除形态变化超出本 fixture 支持面: %q", line)
		}
		data, rerr := os.ReadFile(filepath.Join(ws, filepath.FromSlash(path)))
		if rerr != nil {
			return workflowCandidate{}, false, fmt.Errorf("p04: 回读改动 %s: %w", path, rerr)
		}
		files = append(files, changeset.ChangedFile{Path: path, Content: string(data)})
	}
	if len(files) == 0 {
		return workflowCandidate{}, false, nil
	}
	head, err := p04GitOut(ctx, ws, "rev-parse", "HEAD")
	if err != nil {
		return workflowCandidate{}, false, fmt.Errorf("p04: 读 worktree HEAD: %w", err)
	}
	return workflowCandidate{candidateRevision: head, files: files, entities: []string{"greeting"}}, true, nil
}

// workflowCandidate 是从 worktree 事实读出的候选内容(等价于内核
// CandidateSnapshot 的最小投影,测试本地持有)。
type workflowCandidate struct {
	candidateRevision string
	files             []changeset.ChangedFile
	entities          []string
}

// ---- held-out 注入边界(P1-29 控制面接线,与 P1-43 同构)----------------------

func p04HoldoutBoundary(rt runner.Runner) verification.HoldoutRunFunc {
	return func(ctx context.Context, req verification.HoldoutRunRequest) (verification.HoldoutRunOutput, error) {
		if req.Command == nil {
			return verification.HoldoutRunOutput{}, errors.New("p04: held-out 冻结命令缺失")
		}
		// 一次性 Tally 控制目录:内容只以内存字节到达这里,物化后执行冻结命令;
		// 原始输出留在边界内,对外只有结构化 status + 摘要(§14.3)。
		dir, err := os.MkdirTemp("", "tally-p04-holdout-")
		if err != nil {
			return verification.HoldoutRunOutput{}, fmt.Errorf("p04: 建 holdout 边界目录: %w", err)
		}
		defer os.RemoveAll(dir)
		if err := os.WriteFile(filepath.Join(dir, "holdout-input"), req.Content, 0o600); err != nil {
			return verification.HoldoutRunOutput{}, fmt.Errorf("p04: 物化 held-out 内容: %w", err)
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
		st := verification.EvidenceStatusFail
		if res.ExitCode == 0 {
			st = verification.EvidenceStatusPass
		}
		return verification.HoldoutRunOutput{
			Status:       st,
			OutputDigest: holdout.ContentDigest([]byte(res.Stdout + "\n" + res.Stderr)),
		}, nil
	}
}

// ---- verification.RunSink → runstore 适配(与 P1-43 接线同一合同)------------

type p04VerifySink struct {
	runs     *runstore.Store
	revision int32
}

func (s *p04VerifySink) RecordRun(ctx context.Context, spec verification.RunSpec) (int64, error) {
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
		Status:            p04RunStatus(spec.Status),
	})
	if err != nil {
		return 0, fmt.Errorf("p04: 落 verification_run: %w", err)
	}
	return id, nil
}

func p04RunStatus(s verification.RunStatus) runstore.VerificationStatus {
	switch s {
	case verification.RunStatusPass:
		return runstore.VerificationPass
	case verification.RunStatusFail:
		return runstore.VerificationFail
	default:
		return runstore.VerificationError
	}
}

func (s *p04VerifySink) RecordFact(ctx context.Context, project core.ProjectID, fact verification.FactSpec) error {
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
		Command:          p04ToCommandSpec(fact.Command),
		ExitCode:         &exit,
		StdoutDigest:     fact.StdoutDigest,
		StderrDigest:     fact.StderrDigest,
		RecordedAt:       fact.RecordedAt,
	}); err != nil {
		return fmt.Errorf("p04: 落 evidence %s: %w", fact.FactID, err)
	}
	return nil
}

func (s *p04VerifySink) ValidApplicableFacts(ctx context.Context, scope evidence.Scope) ([]verification.PriorFact, error) {
	rows, err := s.runs.ApplicableValidEvidence(ctx, scope, s.revision)
	if err != nil {
		return nil, fmt.Errorf("p04: 读回既往 evidence: %w", err)
	}
	out := make([]verification.PriorFact, 0, len(rows))
	for _, r := range rows {
		switch evidence.Verdict(r.Verdict) {
		case evidence.VerdictPass:
			out = append(out, verification.PriorFact{BindingID: r.BindingID, Status: verification.EvidenceStatusPass, RecordedAt: r.RecordedAt})
		case evidence.VerdictFail:
			out = append(out, verification.PriorFact{BindingID: r.BindingID, Status: verification.EvidenceStatusFail, RecordedAt: r.RecordedAt})
		default:
			continue
		}
	}
	return out, nil
}

func p04ToCommandSpec(spec *verification.CommandSpec) *command.CommandSpec {
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

func p04CountRows(t *testing.T, pool *pgxpool.Pool, query string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := pool.QueryRow(context.Background(), query, args...).Scan(&n); err != nil {
		t.Fatalf("count %q: %v", query, err)
	}
	return n
}

// ---- 主体测试 -----------------------------------------------------------------

func TestP04RealCodexStandardHeldOutTrustPath(t *testing.T) {
	ctx := context.Background()

	// 0) Canary 先行:证据门(DECISION_LOCKS #22)。失败时生产配置保持未声明,
	// canary 测试另行显形失败;本测试继续(隔离行为本身仍是 per-run home)。
	canary := p04RunCanaryOnce(t)
	verified := canary.CapabilityAvailable

	runs, pool := p04Store(t)
	task := p04TaskContract()
	if err := task.Validate(); err != nil {
		t.Fatalf("TaskContract 非法: %v", err)
	}

	// 1) 种子 fixture repo:微型 Go module + 一条对 base 可见的失败测试。
	repo := t.TempDir()
	p04Git(t, repo, "init", "-b", "main")
	p04Git(t, repo, "config", "user.name", "tally-p04-e2e")
	p04Git(t, repo, "config", "user.email", "tally-p04-e2e@example.invalid")
	base := p04Commit(t, repo, "go.mod", p04GoMod, "base: module")
	base = p04Commit(t, repo, "greeting/greeting.go", p04BaseGreeting, "base: greeting stub")
	base = p04Commit(t, repo, "greeting/greeting_test.go", p04GreetingTest, "base: visible acceptance test (red)")

	// 2) holdout Store:Tally 控制的文件系统根,机械上在每个受管 root 之外。
	storeRoot := filepath.Join(t.TempDir(), "holdout-store")
	wtRoot := filepath.Join(t.TempDir(), "codegen-worktrees")
	vRoot := filepath.Join(t.TempDir(), "verify-worktrees")
	hstore, err := holdout.NewFilesystemStore(storeRoot, repo, wtRoot, vRoot)
	if err != nil {
		t.Fatalf("打开 holdout store: %v", err)
	}
	holdoutRef, err := hstore.Put(ctx, p04HoldoutContent())
	if err != nil {
		t.Fatalf("写入 held-out 内容: %v", err)
	}
	holdoutDigest := holdout.ContentDigest(p04HoldoutContent())
	frozen := p04FrozenContract(t, holdoutRef)
	pack := p04ContextPack(t, repo, base, task, frozen)

	// 3) 真实 CodexRuntime:ADR-0003 Provider 播种 + auth.json 移交;声明门以
	// canary 证据为准。
	cfg := p04CodexConfig(t, verified)
	rtInner := codex.New(cfg)
	desc := rtInner.Descriptor()
	if desc.Capabilities.Has(agentruntime.CapSessionIsolation) != verified {
		t.Fatalf("session.isolation 声明与 canary 证据不符: verified=%v caps=%v", verified, desc.Capabilities.Sorted())
	}
	if verified {
		t.Logf("canary 通过:生产配置声明 session.isolation")
	}
	homeBefore := p04TempSnapshot(t, "tally-codex-home-*")
	holdoutDirBefore := p04TempSnapshot(t, "tally-p04-holdout-*")

	// 4) codegen:真实 codex 回合在工作流内核同款的一次性 worktree 内执行。
	//     身份证据(AC3)在此全程捕获:Snapshot(现场 --version)+ 会话
	//     ObservedIdentity(thread id;model 不自报 → unknown)。
	snap, err := rtInner.Snapshot(ctx)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if snap.BinaryPath != cfg.BinPath || snap.CLIVersion == "" {
		t.Fatalf("Snapshot 身份不完整: %+v", snap)
	}
	hasIso := false
	for _, c := range snap.Capabilities {
		if c == agentruntime.CapSessionIsolation {
			hasIso = true
		}
	}
	if hasIso != verified {
		t.Fatalf("Snapshot 的 session.isolation 声明与 canary 证据不符: %v", snap.Capabilities)
	}

	mgr, err := workspace.New(repo, wtRoot)
	if err != nil {
		t.Fatalf("workspace.New: %v", err)
	}
	runTable := workspace.NewRunTable(mgr)
	sources, err := runTable.Open(ctx, workspace.RunID("run-0001"), base)
	if err != nil {
		t.Fatalf("打开一次性 codegen worktree: %v", err)
	}
	codegenWS := sources.Path()
	if !strings.Contains(filepath.ToSlash(codegenWS), filepath.ToSlash(wtRoot)) {
		t.Fatalf("codegen workspace %s 不在受管一次性根 %s 下(INV-EXEC-01)", codegenWS, wtRoot)
	}

	session, err := rtInner.Start(ctx, agentruntime.StartRequest{
		Workspace:    codegenWS,
		ModelProfile: p04Model, // codex 声明 model.select;期望档位随会话请求下发
	})
	if err != nil {
		t.Fatalf("Start(真实 codex 会话): %v", err)
	}

	turnCtx, cancelTurn := context.WithTimeout(ctx, 10*time.Minute)
	claim, execErr := session.Execute(turnCtx, agentruntime.ExecuteRequest{
		Instruction: p04Instruction(task),
	})
	cancelTurn()
	if execErr != nil {
		t.Fatalf("Execute(真实 codex 回合) 不应返回 error(失败走 claim): %v", execErr)
	}

	// claim 原样是 claim(可与事实相反,INV-AUTH-01):真实表面事实是适配器把
	// codex 对未知模型名的非致命 metadata 警告 item 归为 RunFailed;候选与验收
	// 都不取自它。这里只要求会话身份已观察(thread.started 已到)。
	observed := agentruntime.ObservedIdentity{}
	if rep, ok := session.(agentruntime.IdentityReporter); ok {
		observed = rep.Identity()
	}
	if claim.RunID == "" || claim.SessionID == "" || observed.SessionID != claim.SessionID {
		t.Fatalf("claim/会话身份不完整: claim=%+v observed=%+v", claim, observed)
	}
	t.Logf("真实 codex 回合 claim(原样): status=%s session=%s summary=%q",
		claim.Status, claim.SessionID, truncateForLog(claim.Summary, 300))
	t.Logf("身份证据:binary=%s cli_version=%s adapter_version=%s requested_model=%s observed_model=unknown(不自报,不回填)",
		snap.BinaryPath, snap.CLIVersion, snap.AdapterVersion, p04Model)
	t.Logf("用量:adapter 不解析 exec 用量事件 → usage unavailable(零值,非伪造 reported usage)")

	// 候选从 worktree 事实固定(零注入)。
	cand, ok, err := p04CandidateFromWorktree(ctx, codegenWS)
	if err != nil {
		t.Fatalf("读真实候选: %v", err)
	}
	if !ok {
		t.Fatalf("真实 codex 回合未在一次性 worktree 内留下任何候选改动(status=%s)", claim.Status)
	}
	// 一次性 worktree 随 Run 终止销毁(P1-35)。
	if err := runTable.Terminate(ctx, workspace.RunID("run-0001")); err != nil {
		t.Fatalf("销毁 codegen worktree: %v", err)
	}
	if _, err := os.Stat(codegenWS); !os.IsNotExist(err) {
		t.Fatalf("codegen 一次性 worktree 未销毁: %s (%v)", codegenWS, err)
	}
	if err := session.Close(ctx); err != nil {
		t.Fatalf("Close(真实 codex 会话): %v", err)
	}

	var greetingFile *changeset.ChangedFile
	for i := range cand.files {
		if cand.files[i].Path == "greeting/greeting.go" {
			greetingFile = &cand.files[i]
		}
	}
	if greetingFile == nil || !strings.Contains(greetingFile.Content, p04Greeting) {
		t.Fatalf("候选缺真实交付的 greeting 实现: files=%v", cand.files)
	}

	// 5) 落库(生产存储接口):Run 行带期望坐标与 observed_identity 实际身份,
	//     AgentClaim 原样保留;候选 ChangeSet 回指该 Run。
	capsJSON := mustJSON(t, struct {
		Name         string   `json:"name"`
		Version      string   `json:"version"`
		Capabilities []string `json:"capabilities"`
	}{Name: desc.Name, Version: desc.Version, Capabilities: capabilityNames(t, desc)})
	codegenRunID, err := runs.StartCodegenRun(ctx, runstore.StartRunSpec{
		Project:                   p04Project,
		Task:                      task.ID,
		Role:                      "coding",
		Runtime:                   desc.Name,
		RuntimeVersion:            desc.Version,
		Model:                     p04Model, // 期望档位(与实际身份分列,Decision Lock 15)
		BehaviorRevision:          "behavior-1",
		CostPlane:                 measurement.CostPlaneExecution,
		BaseCommit:                base,
		TaskRevision:              task.Hash,
		VerificationRevision:      frozen.Hash(),
		ContextPackHash:           pack.Hash(),
		WorkspaceID:               codegenWS,
		AgentSessionID:            string(observed.SessionID),
		ObservedRuntimeName:       observed.RuntimeName,
		ObservedRuntimeVersion:    observed.RuntimeVersion,
		ObservedModel:             observed.Model, // 空 = 未观察(exec 表面不自报)
		RuntimeCapabilitySnapshot: capsJSON,
	})
	if err != nil {
		t.Fatalf("StartCodegenRun: %v", err)
	}
	cs, err := changeset.New(changeset.Spec{
		Project:           p04Project,
		BaseRevision:      base,
		CandidateRevision: cand.candidateRevision,
		Files:             cand.files,
		Entities:          cand.entities,
		ProducerRun:       strconv.FormatInt(codegenRunID, 10),
	})
	if err != nil {
		t.Fatalf("构造 ChangeSet: %v", err)
	}
	if err := runs.SaveChangeSet(ctx, cs); err != nil {
		t.Fatalf("SaveChangeSet: %v", err)
	}
	if ok, err := runs.FinishCodegenRun(ctx, runstore.FinishRunSpec{
		ID:              codegenRunID,
		Status:          runstore.RunCompleted, // Run 事实:候选已固定(验证随后裁决)
		Iterations:      1,
		InputTokens:     nil, // 未上报(usage unavailable),不伪造零值当 reported
		OutputTokens:    nil,
		ResultPatchHash: string(cs.ID()),
		AgentClaim:      claim.Summary, // claim 原样留档(含其 failed 分类)
		FinishedAt:      time.Now(),
	}); err != nil || !ok {
		t.Fatalf("FinishCodegenRun: ok=%v err=%v", ok, err)
	}

	// 6) AC1 断言:Run/ChangeSet 行真实、可追溯、产自一次性 worktree。
	run, err := runs.CodegenRun(ctx, codegenRunID)
	if err != nil {
		t.Fatalf("CodegenRun(%d): %v", codegenRunID, err)
	}
	if run.Runtime != codex.DescriptorName || run.Model != p04Model || run.Status != runstore.RunCompleted {
		t.Fatalf("codegen_run runtime/model/status = %s/%s/%s, want codex/%s/completed(AC1)",
			run.Runtime, run.Model, run.Status, p04Model)
	}
	if run.AgentSessionID != string(observed.SessionID) || run.AgentSessionID == "" {
		t.Fatalf("codegen_run.agent_session_id = %q, want 观察到的 thread id %q(AC1/AC3)", run.AgentSessionID, observed.SessionID)
	}
	if run.TaskRevision != task.Hash || run.VerificationRevision != frozen.Hash() || run.ContextPackHash != pack.Hash() {
		t.Fatalf("codegen_run 合同身份不齐: task=%s ver=%s pack=%s", run.TaskRevision, run.VerificationRevision, run.ContextPackHash)
	}
	if run.ResultPatchHash != string(cs.ID()) {
		t.Fatalf("codegen_run.result = %q, want ChangeSet 身份 %s", run.ResultPatchHash, cs.ID())
	}
	if !strings.Contains(string(run.RuntimeCapabilitySnapshot), "observed_identity") ||
		!strings.Contains(string(run.RuntimeCapabilitySnapshot), "codex-cli") {
		t.Fatalf("codegen_run 快照缺 observed_identity 实际身份: %s", truncateForLog(string(run.RuntimeCapabilitySnapshot), 300))
	}
	t.Logf("真实候选:changeset=%s candidate_rev=%s files=%d codegen_run=%d",
		cs.ID(), cs.CandidateRevision(), len(cs.Files()), codegenRunID)

	// 7) 最终验证:一次性验证 workspace 内物化候选,编排器执行冻结验证。
	vmgr, err := workspace.New(repo, vRoot)
	if err != nil {
		t.Fatalf("workspace.New(verify): %v", err)
	}
	wt, err := vmgr.Create(ctx, cs.BaseRevision())
	if err != nil {
		t.Fatalf("建验证 workspace: %v", err)
	}
	var vmgrCleaned bool
	cleanupVerify := func() {
		if vmgrCleaned {
			return
		}
		vmgrCleaned = true
		_ = vmgr.Cleanup(context.WithoutCancel(ctx), wt)
	}
	t.Cleanup(cleanupVerify)
	verifyWS := wt.Path
	for _, f := range cs.Files() {
		p := filepath.Join(verifyWS, filepath.FromSlash(filepath.Clean(f.Path)))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("物化候选(建目录)%s: %v", f.Path, err)
		}
		if err := os.WriteFile(p, []byte(f.Content), 0o644); err != nil {
			t.Fatalf("物化候选 %s: %v", f.Path, err)
		}
	}

	env := verification.EnvironmentIdentity{OS: runtime.GOOS, Arch: runtime.GOARCH, Toolchain: runtime.Version()}
	envHash, err := env.Hash()
	if err != nil {
		t.Fatalf("环境身份哈希: %v", err)
	}
	orch, err := verification.NewOrchestrator(verification.OrchestratorConfig{
		Runner:        runner.LocalRunner{},
		Policy:        policy.DefaultExecutionPolicy(),
		HoldoutReader: hstore,
		HoldoutRun:    p04HoldoutBoundary(runner.LocalRunner{}),
		Sink:          &p04VerifySink{runs: runs, revision: int32(frozen.Revision())},
		Recorder:      evidence.Recorder{Kind: evidence.RecorderVerificationRun, Actor: "tally-e2e/p0-04"},
	})
	if err != nil {
		t.Fatalf("构造最终验证编排器: %v", err)
	}
	verify, err := orch.Run(ctx, verification.FinalRunRequest{
		Project:          p04Project,
		Contract:         frozen,
		ChangeSet:        cs,
		EnvironmentName:  p04EnvName,
		Environment:      env,
		VersionCombo:     core.VersionCombo{"go": runtime.Version(), "runtime": "codex"},
		Runtime:          "codex",
		Workspace:        wt.Path,
		CandidateFixedAt: cs.CreatedAt(),
		Lineage:          []evidence.ChangeSetID{evidence.ChangeSetID(cs.ID())},
		Implementation: evidence.ImplementationProvenance{Contributors: []evidence.Contributor{
			{Name: "codex-session/" + string(observed.SessionID), Role: "coding_agent"},
		}},
	})
	if err != nil {
		t.Fatalf("最终验证执行: %v", err)
	}
	if verify.Status != verification.FinalVerified {
		for _, b := range verify.Bindings {
			t.Logf("binding %s status=%s exit=%d err=%q", b.BindingID, b.Status, b.ExitCode, b.Err)
		}
		t.Fatalf("最终验证结论 = %s, want VERIFIED", verify.Status)
	}

	// 8) AC2:验证 Run 精确绑定冻结契约 + 候选身份 + 环境身份。
	vr, err := runs.VerificationRun(ctx, p04Project, verify.RunID)
	if err != nil {
		t.Fatalf("VerificationRun(%d): %v", verify.RunID, err)
	}
	if vr.ContractRevision != int32(frozen.Revision()) || vr.ContractHash != frozen.Hash() ||
		vr.ChangeSetID != cs.ID() || vr.EnvDefinitionHash != envHash || vr.Status != runstore.VerificationPass {
		t.Fatalf("verification_run 绑定不精确: rev=%d hash=%s cs=%s env=%s status=%s",
			vr.ContractRevision, vr.ContractHash, vr.ChangeSetID, vr.EnvDefinitionHash, vr.Status)
	}

	// AC2:held-out 验证在 Tally 控制边界内通过,且贡献 information independence。
	if !verify.Independence.Complete {
		t.Fatalf("independence 覆盖不完整: %+v", verify.Independence)
	}
	var visibleBO *verification.BindingOutcome
	for i := range verify.Bindings {
		if verify.Bindings[i].BindingID == p04VisibleID {
			visibleBO = &verify.Bindings[i]
		}
	}
	if visibleBO == nil || visibleBO.Status != verification.EvidenceStatusPass || visibleBO.ExitCode != 0 {
		t.Fatalf("visible binding 未通过: %+v", visibleBO)
	}
	// held-out 判定事实只走 Control/Review 通道(verify.Holdout),不进 Bindings。
	if len(verify.Holdout) != 1 {
		t.Fatalf("held-out 判定记录数 = %d, want 1", len(verify.Holdout))
	}
	h := verify.Holdout[0]
	if h.BindingID != p04HeldOutID || h.Status != verification.EvidenceStatusPass {
		t.Fatalf("held-out 判定 = %s/%s, want %s/pass", h.BindingID, h.Status, p04HeldOutID)
	}
	if h.ContentDigest != holdoutDigest ||
		h.CandidateID != string(cs.ID()) || h.ContractHash != frozen.Hash() {
		t.Fatalf("held-out 判定绑定不精确: %+v", h)
	}

	// AC2 的隔离面:实现上下文(contextpack)无法解析 held-out 内容。
	var heldOutProj *contextpack.EvidenceProjection
	for i := range pack.Verification {
		if pack.Verification[i].ID != p04Acceptance {
			continue
		}
		for j := range pack.Verification[i].Evidence {
			if pack.Verification[i].Evidence[j].Visibility == verification.VisibilityHeldOut {
				heldOutProj = &pack.Verification[i].Evidence[j]
			}
		}
	}
	if heldOutProj == nil {
		t.Fatal("contextpack 缺 held-out projection")
	}
	if heldOutProj.BindingID != "" || heldOutProj.Store != "" || heldOutProj.Ref != "" || heldOutProj.Path != "" {
		t.Fatalf("held-out projection 携带定位字段: %+v", *heldOutProj)
	}
	serialized := string(mustJSON(t, pack))
	if strings.Contains(serialized, holdoutRef.String()) ||
		strings.Contains(serialized, p04HoldoutSecret) ||
		strings.Contains(serialized, holdoutDigest) {
		t.Fatal("contextpack 泄漏 held-out 引用/内容/摘要(AC2)")
	}
	// 实现侧 redacted signal 不携带 held-out 判定锚点。
	sig := verify.RedactedSignal()
	projected := fmt.Sprint(sig.Holdouts)
	if strings.Contains(projected, holdoutRef.String()) || strings.Contains(projected, p04HoldoutSecret) ||
		strings.Contains(projected, holdoutDigest) || strings.Contains(projected, string(verification.EvidenceStatusPass)) {
		t.Fatalf("实现侧信号携带 held-out 锚点: %s", projected)
	}

	// 事实面:held-out PASS 以 held_out 可见性落库,恰好一条。
	if n := p04CountRows(t, pool, "SELECT count(*) FROM verification_evidence WHERE project_id=$1 AND visibility_policy='held_out' AND verdict='pass'", string(p04Project)); n != 1 {
		t.Fatalf("数据库 held_out PASS 事实行数 = %d, want 1(AC2)", n)
	}

	// 9) 处置证据:验证 workspace 回收;无新增 per-run home / holdout 边界残留。
	cleanupVerify()
	if _, err := os.Stat(verifyWS); !os.IsNotExist(err) {
		t.Fatalf("验证 workspace 未回收: %s (%v)", verifyWS, err)
	}
	if fresh := p04TempNewAfter(t, "tally-codex-home-*", homeBefore); len(fresh) != 0 {
		t.Fatalf("真实 codex 会话 home 未随 Close 全部回收: %v", fresh)
	}
	if fresh := p04TempNewAfter(t, "tally-p04-holdout-*", holdoutDirBefore); len(fresh) != 0 {
		t.Fatalf("held-out 边界目录未回收: %v", fresh)
	}

	// 10) AC4:owner fixture repo 全程零污染(实现从未在 owner worktree 执行)。
	if dirty := p04Git(t, repo, "status", "--porcelain"); dirty != "" {
		t.Fatalf("owner fixture worktree 被污染:\n%s", dirty)
	}

	t.Logf("P0-04 信任路径完成:canary=%v codex=%s verified=%s visible=%s heldout=%s",
		verified, snap.CLIVersion, verify.Status, visibleBO.Status, h.Status)
}

func capabilityNames(t *testing.T, d agentruntime.RuntimeDescriptor) []string {
	t.Helper()
	list := d.Capabilities.Sorted()
	out := make([]string, 0, len(list))
	for _, c := range list {
		out = append(out, string(c))
	}
	return out
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("序列化 %T: %v", v, err)
	}
	return data
}

func truncateForLog(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return string(r)
	}
	return string(r[:max]) + "..."
}
