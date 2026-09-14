package main

// P1-43 的垂直路径接线测试。
//
// 覆盖任务验收:
//   - AC1 单个 approved standard TaskContract 走完 CodegenRun→ChangeSet→Verification
//     (真 git 仓库 + 真 git worktree + FakeRuntime;事实面用内存 sink,数据库
//     round trip 由文件尾部的数据库测试覆盖,库不可达即 SKIP,与
//     internal/workflow 同一隔离策略);
//   - AC2 legacy job 注册不受影响(main.go 的 handlers 表仍含全部 exec handler);
//   - AC3 Codex/Fake 选择是配置(workerRuntimes 注册表的映射面,P4-01),
//     不是内核条件逻辑;
//   - AC4 主 worktree 零污染、一次性 worktree 全部回收;
//   - AC5 held-out 前提强制:holdout 通道未接线或 runtime 未声明
//     session.isolation 一律拒绝执行。
//
// 候选命令与 visible binding 命令用 `go version`(离线、确定性 exit 0),全程
// 无 shell 字符串。

import (
	"context"
	"database/sql"
	"encoding/json"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"tally/internal/agentruntime"
	"tally/internal/candidateverify"
	"tally/internal/changeset"
	"tally/internal/core"
	"tally/internal/evidence"
	"tally/internal/exec"
	"tally/internal/jobs"
	"tally/internal/runner"
	"tally/internal/runstore"
	"tally/internal/store"
	"tally/internal/verification"
	"tally/internal/workflow"
	"tally/internal/workspace"
	"tally/migrations"
	"tally/runtimes/fake"
)

// ---- 固件:.tally 制品仓库 ----

const v13ProjectYAML = `project: proj-v13
entities:
  - id: widget
    type: go_package
    path: internal/widget
`

const v13TaskYAML = `id: TASK-1043
profile: standard
title: Vertical path fixture
objective: Add a widget greeting under internal/widget.
scope:
  allowed_paths:
    - internal/widget
acceptance:
  - id: AC-01
    statement: widget greets.
    independence_required: none
`

const v13VerificationYAML = `task_id: TASK-1043
revision: 1
acceptances:
  - id: AC-01
    statement: widget greets.
    independence_required: none
    evidence_bindings:
      - id: VIS-1
        authoring_mode: pre_commit
        visibility_policy: visible
        evidence:
          type: test
          path: internal/widget/widget_test.go
        execution_protocol:
          repetitions: 1
          aggregation: all_pass
          command:
            argv: [go, version]
`

// light 合同:P1 垂直路径只执行 standard(AC1),light 应被拒绝。
const v13LightTaskYAML = `id: TASK-1051
profile: light
title: Light fixture
objective: Tidy the widget docs.
scope:
  allowed_paths:
    - internal/widget
`

// held-out 契约:含 held_out binding,冻结必须声明 INV-VER-05 runtime 能力。
const v13HeldTaskYAML = `id: TASK-1050
profile: standard
title: Held-out fixture
objective: Hidden property holds.
scope:
  allowed_paths:
    - internal/widget
acceptance:
  - id: AC-01
    statement: hidden property holds.
    independence_required: none
`

const v13HeldVerificationYAML = `task_id: TASK-1050
revision: 1
implementation_runtime_requirements:
  session_isolation: true
  held_out_unreachable: true
  conformance_class: verification-isolation-v1
acceptances:
  - id: AC-01
    statement: hidden property holds.
    independence_required: none
    evidence_bindings:
      - id: HOLD-1
        authoring_mode: pre_commit
        visibility_policy: held_out
        evidence:
          type: test
          store: holdout
          ref: hold-1
        execution_protocol:
          repetitions: 1
          aggregation: all_pass
          command:
            argv: [go, version]
`

// newV13Repo 建一个带 .tally 制品(TASK-1043 standard + 冻结验证契约)的真 git 仓库。
func newV13Repo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.name", "tally-test")
	gitRun(t, dir, "config", "user.email", "tally-test@example.invalid")
	v13WriteRepoFile(t, dir, ".tally/project.yaml", v13ProjectYAML)
	v13WriteRepoFile(t, dir, ".tally/tasks/TASK-1043.yaml", v13TaskYAML)
	v13WriteRepoFile(t, dir, ".tally/verifications/TASK-1043.yaml", v13VerificationYAML)
	v13WriteRepoFile(t, dir, "internal/widget/widget_test.go", "package widget\n")
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-m", "v13 fixtures")
	return dir
}

// newV13HeldRepo 建一个含 held_out 验证契约(TASK-1050)的仓库。
func newV13HeldRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.name", "tally-test")
	gitRun(t, dir, "config", "user.email", "tally-test@example.invalid")
	v13WriteRepoFile(t, dir, ".tally/project.yaml", v13ProjectYAML)
	v13WriteRepoFile(t, dir, ".tally/tasks/TASK-1050.yaml", v13HeldTaskYAML)
	v13WriteRepoFile(t, dir, ".tally/verifications/TASK-1050.yaml", v13HeldVerificationYAML)
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-m", "held-out fixtures")
	return dir
}

func v13WriteRepoFile(t *testing.T, repo, rel, content string) {
	t.Helper()
	p := filepath.Join(repo, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("建目录 %s: %v", rel, err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("写 %s: %v", rel, err)
	}
}

// v13GitOut 跑一条只读 git 查询并返回 stdout(结构化 argv)。
func v13GitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	res, err := (runner.LocalRunner{}).Run(context.Background(), runner.Command{
		Argv: append([]string{"git", "-C", dir}, args...),
	})
	if err != nil || res.ExitCode != 0 {
		t.Fatalf("git %v: err=%v res=%+v", args, err, res)
	}
	return strings.TrimSpace(res.Stdout)
}

func v13ConfigFor(t *testing.T, repo string) v13Config {
	return v13Config{
		Repo:         repo,
		WorktreeRoot: filepath.Join(t.TempDir(), "wt"),
		Runtime:      "fake",
		Profile:      "default-coding",
		Model:        "test-model",
		Behavior:     "behavior-1",
		EnvName:      "test-env",
	}
}

// ---- 测试注入替身 ----

// v13WritingRuntime 包一层 fake runtime:首个 turn 在会话 workspace 内写一个
// scope 内文件再完成 —— 模拟实现产出,让候选固定钩子有真改动可收。
type v13WritingRuntime struct {
	inner   agentruntime.Runtime
	relPath string
	content string

	once     sync.Once
	writeErr error
}

func newWritingRuntime(relPath, content string) *v13WritingRuntime {
	return &v13WritingRuntime{inner: fake.New(fake.Config{}), relPath: relPath, content: content}
}

func (r *v13WritingRuntime) Descriptor() agentruntime.RuntimeDescriptor { return r.inner.Descriptor() }

func (r *v13WritingRuntime) Start(ctx context.Context, req agentruntime.StartRequest) (agentruntime.Session, error) {
	s, err := r.inner.Start(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v13WritingSession{rt: r, inner: s, ws: req.Workspace}, nil
}

type v13WritingSession struct {
	rt    *v13WritingRuntime
	inner agentruntime.Session
	ws    string
}

func (s *v13WritingSession) Execute(ctx context.Context, req agentruntime.ExecuteRequest) (agentruntime.AgentRunResult, error) {
	s.rt.once.Do(func() { s.rt.writeErr = s.rt.write(s.ws) })
	if s.rt.writeErr != nil {
		return agentruntime.AgentRunResult{}, s.rt.writeErr
	}
	return s.inner.Execute(ctx, req)
}

func (r *v13WritingRuntime) write(ws string) error {
	p := filepath.Join(ws, filepath.FromSlash(r.relPath))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(r.content), 0o644)
}

func (s *v13WritingSession) Cancel(ctx context.Context, id agentruntime.RunID) error {
	return s.inner.Cancel(ctx, id)
}

func (s *v13WritingSession) Events(ctx context.Context) (agentruntime.EventStream, error) {
	return s.inner.Events(ctx)
}

func (s *v13WritingSession) Result(ctx context.Context, id agentruntime.RunID) (agentruntime.AgentRunResult, error) {
	return s.inner.Result(ctx, id)
}

func (s *v13WritingSession) Close(ctx context.Context) error { return s.inner.Close(ctx) }

// v13MemRunSink 是 workflow.RunSink 的内存实现(严格记录事件值)。
type v13MemRunSink struct {
	mu         sync.Mutex
	seq        int
	begins     []workflow.RunBegin
	progresses []workflow.RunProgress
	finishes   []workflow.RunFinish
	changeSets []*changeset.ChangeSet
}

func (s *v13MemRunSink) BeginRun(_ context.Context, b workflow.RunBegin) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	s.begins = append(s.begins, b)
	return strconv.Itoa(s.seq), nil
}

func (s *v13MemRunSink) RecordProgress(_ context.Context, _ workflow.RunID, p workflow.RunProgress) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.progresses = append(s.progresses, p)
	return nil
}

func (s *v13MemRunSink) FinishRun(_ context.Context, f workflow.RunFinish) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.finishes = append(s.finishes, f)
	return nil
}

func (s *v13MemRunSink) RecordChangeSet(_ context.Context, _ workflow.RunID, cs *changeset.ChangeSet) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.changeSets = append(s.changeSets, cs)
	return nil
}

// v13MemVerifySink 是 verification.RunSink 的内存实现。
type v13MemVerifySink struct {
	mu    sync.Mutex
	next  int64
	runs  []verification.RunSpec
	facts []verification.FactSpec
}

func (s *v13MemVerifySink) RecordRun(_ context.Context, spec verification.RunSpec) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	s.runs = append(s.runs, spec)
	return s.next, nil
}

func (s *v13MemVerifySink) RecordFact(_ context.Context, _ core.ProjectID, fact verification.FactSpec) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.facts = append(s.facts, fact)
	return nil
}

func (s *v13MemVerifySink) ValidApplicableFacts(context.Context, evidence.Scope) ([]verification.PriorFact, error) {
	return nil, nil
}

// ---- AC1/AC4:垂直路径端到端(内存事实面) ----

func TestV13VerticalPathEndToEnd(t *testing.T) {
	repo := newV13Repo(t)
	cfg := v13ConfigFor(t, repo)
	wf := &v13MemRunSink{}
	vs := &v13MemVerifySink{}

	deps := v13Deps{
		cfg:          cfg,
		runtime:      newWritingRuntime("internal/widget/greeting.go", "package widget\n"),
		workflowSink: wf,
		verifySink:   vs,
	}
	res, err := runV13Task(context.Background(), deps, "TASK-1043")
	if err != nil {
		t.Fatalf("runV13Task: %v", err)
	}

	// AC1:走完 CodegenRun → ChangeSet → Verification。
	if res.Codegen.Status != workflow.TaskCompleted || len(res.Codegen.Runs) != 1 {
		t.Fatalf("codegen 侧终态 = %s/%d run, want completed/1", res.Codegen.Status, len(res.Codegen.Runs))
	}
	cs := res.Codegen.ChangeSet
	if cs == nil {
		t.Fatal("ChangeSet = nil, want 固定候选")
	}
	if cs.Project() != "proj-v13" || cs.BaseRevision() == "" {
		t.Fatalf("ChangeSet project/base = %s/%s", cs.Project(), cs.BaseRevision())
	}
	if cs.ProducerRun() != "1" {
		t.Fatalf("ChangeSet producer run = %q, want sink 返回的持久化引用 1", cs.ProducerRun())
	}
	found := false
	for _, f := range cs.Files() {
		if f.Path == "internal/widget/greeting.go" && strings.Contains(f.Content, "package widget") {
			found = true
		}
	}
	if !found {
		t.Fatalf("候选缺少实现产出: %v", cs.Files())
	}

	// run 事实:开跑坐标齐全,终态 completed。
	if len(wf.begins) != 1 || len(wf.progresses) != 1 || len(wf.finishes) != 1 || len(wf.changeSets) != 1 {
		t.Fatalf("sink 事件数 = %d/%d/%d/%d, want 1/1/1/1",
			len(wf.begins), len(wf.progresses), len(wf.finishes), len(wf.changeSets))
	}
	b := wf.begins[0]
	if b.Project != "proj-v13" || b.TaskID != "TASK-1043" || b.Role != "coding" ||
		b.RuntimeName != "fake" || b.Model != "test-model" || b.BehaviorRevision != "behavior-1" ||
		b.BaseRevision == "" || b.WorkspaceID == "" ||
		!strings.HasPrefix(b.VerificationRevision, "sha256:") ||
		!strings.HasPrefix(b.ContextPackHash, "sha256:") {
		t.Fatalf("RunBegin = %+v", b)
	}
	if wf.finishes[0].Status != runstore.RunCompleted || wf.finishes[0].ResultPatchHash == "" {
		t.Fatalf("RunFinish = %+v", wf.finishes[0])
	}

	// Verification:visible binding(go version)通过 → verified;事实落 sink。
	if res.Verify == nil {
		t.Fatal("Verify = nil, want 最终验证结论")
	}
	if res.Verify.Status != verification.FinalVerified {
		t.Fatalf("Verification = %s, want verified: %+v", res.Verify.Status, res.Verify.Bindings)
	}
	if len(vs.runs) != 1 || vs.runs[0].Status != verification.RunStatusPass {
		t.Fatalf("VerificationRun 落库 = %+v", vs.runs)
	}
	if len(vs.facts) != 1 {
		t.Fatalf("evidence 事实数 = %d, want 1", len(vs.facts))
	}
	fact := vs.facts[0]
	if fact.BindingID != "VIS-1" || fact.Acceptance != "AC-01" ||
		fact.Verdict != verification.EvidenceStatusPass ||
		fact.Recorder.Kind != evidence.RecorderVerificationRun ||
		fact.Command == nil || fact.Command.Argv[0] != "go" {
		t.Fatalf("evidence 事实 = %+v", fact)
	}

	// AC4:owner 主 worktree 零污染;一次性 worktree 全部回收。
	if dirty := v13GitOut(t, repo, "status", "--porcelain"); dirty != "" {
		t.Errorf("主 worktree 被污染: %q", dirty)
	}
	entries, err := os.ReadDir(cfg.WorktreeRoot)
	if err != nil || len(entries) != 0 {
		t.Errorf("一次性根目录应清空: entries=%d err=%v", len(entries), err)
	}
	if _, err := os.Stat(res.Codegen.Runs[0].WorkspacePath); !os.IsNotExist(err) {
		t.Errorf("run 的一次性 worktree 应已销毁: %v", err)
	}
}

// ---- AC3:runtime 选择是配置(workerRuntimes 注册表,P4-01)----

func TestV13RuntimeSelectionIsConfiguration(t *testing.T) {
	reg := newRuntimeRegistry()
	for _, tc := range []struct {
		name string
		val  string
		want string
	}{
		{"缺省即 fake(决策锁 13)", "", "fake"},
		{"显式 fake", "fake", "fake"},
		{"显式 codex", "codex", "codex"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rt, err := reg.build(v13Config{Runtime: tc.val})
			if err != nil {
				t.Fatalf("build(%q): %v", tc.val, err)
			}
			if got := rt.Descriptor().Name; got != tc.want {
				t.Fatalf("runtime = %s, want %s", got, tc.want)
			}
		})
	}
	_, err := reg.build(v13Config{Runtime: "magic"})
	if err == nil {
		t.Fatal("未知 runtime 应拒绝")
	}
	// 选择失败的错误由注册表自述支持集合(键序推导)。
	if !strings.Contains(err.Error(), "fake") || !strings.Contains(err.Error(), "codex") {
		t.Fatalf("错误应列出注册的支持集: %v", err)
	}
}

// ---- AC1:P1 垂直路径只执行 standard TaskContract ----

func TestV13RefusesNonStandardTask(t *testing.T) {
	repo := newV13Repo(t)
	v13WriteRepoFile(t, repo, ".tally/tasks/TASK-1051.yaml", v13LightTaskYAML)
	gitRun(t, repo, "add", ".")
	gitRun(t, repo, "commit", "-m", "light task")

	_, err := runV13Task(context.Background(), v13Deps{
		cfg:          v13ConfigFor(t, repo),
		workflowSink: &v13MemRunSink{},
		verifySink:   &v13MemVerifySink{},
	}, "TASK-1051")
	if err == nil || !strings.Contains(err.Error(), "standard") {
		t.Fatalf("light 合同应被拒绝(P1 只执行 standard): %v", err)
	}
}

// ---- AC5:held-out 前提强制 ----

func TestV13HeldOutPrerequisitesEnforced(t *testing.T) {
	repo := newV13HeldRepo(t)
	ctx := context.Background()

	// (1) holdout 通道未接线 → 拒绝执行,绝不降级为只跑 visible(INV-VER-05)。
	cfg := v13ConfigFor(t, repo)
	if _, err := runV13Task(ctx, v13Deps{cfg: cfg}, "TASK-1050"); err == nil || !strings.Contains(err.Error(), "held_out") {
		t.Fatalf("holdout 通道未接线应拒绝执行: %v", err)
	}

	// (2) 通道已接线但 runtime 未声明 session.isolation(Session Isolation
	// Canary 前提不满足,CodexRuntime v1 即此形态)→ 拒绝执行。
	cfg.HoldoutRoot = t.TempDir()
	cfg.Runtime = "codex"
	if _, err := runV13Task(ctx, v13Deps{cfg: cfg}, "TASK-1050"); err == nil || !strings.Contains(err.Error(), "session.isolation") {
		t.Fatalf("runtime 缺 session.isolation 声明应拒绝执行: %v", err)
	}
}

// ---- 候选固定钩子(一次性 worktree 只读快照) ----

func TestFixCandidateFromWorktree(t *testing.T) {
	repo := newV13Repo(t)
	base := v13GitOut(t, repo, "rev-parse", "HEAD")
	mgr, err := workspace.New(repo, filepath.Join(t.TempDir(), "wt"))
	if err != nil {
		t.Fatalf("workspace.New: %v", err)
	}
	ctx := context.Background()
	wt, err := mgr.Create(ctx, base)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer mgr.Cleanup(ctx, wt)

	// 干净 worktree:没有可固定候选。
	snap, ok, err := fixCandidateFromWorktree(ctx, workflow.IterationView{WorkspacePath: wt.Path})
	if err != nil || ok {
		t.Fatalf("干净 worktree 应无可固定候选: ok=%v err=%v", ok, err)
	}

	// 写入改动 → 恰好收进快照,内容与 revision 正确(runtime 未提交 → HEAD 仍是 base)。
	p := filepath.Join(wt.Path, "internal", "widget", "greeting.go")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("建目录: %v", err)
	}
	if err := os.WriteFile(p, []byte("package widget\n"), 0o644); err != nil {
		t.Fatalf("写候选文件: %v", err)
	}
	snap, ok, err = fixCandidateFromWorktree(ctx, workflow.IterationView{WorkspacePath: wt.Path})
	if err != nil || !ok {
		t.Fatalf("有改动应可固定候选: ok=%v err=%v", ok, err)
	}
	if snap.CandidateRevision != base {
		t.Fatalf("CandidateRevision = %s, want %s", snap.CandidateRevision, base)
	}
	if len(snap.Files) != 1 || snap.Files[0].Path != "internal/widget/greeting.go" ||
		snap.Files[0].Content != "package widget\n" {
		t.Fatalf("候选文件 = %+v", snap.Files)
	}

	// 删除改动:P1 候选固定只收改后完整内容,删除文件确定性拒绝
	// (删的是 tracked 文件,git status 才会报 D)。
	if err := os.Remove(filepath.Join(wt.Path, "internal", "widget", "widget_test.go")); err != nil {
		t.Fatalf("删 tracked 文件: %v", err)
	}
	if _, _, err := fixCandidateFromWorktree(ctx, workflow.IterationView{WorkspacePath: wt.Path}); err == nil || !strings.Contains(err.Error(), "删除") {
		t.Fatalf("删除改动应被拒绝: %v", err)
	}
}

func TestParseGitStatusZ(t *testing.T) {
	entries, err := parseGitStatusZ(" M a.go\x00?? b/c.go\x00")
	if err != nil {
		t.Fatalf("parseGitStatusZ: %v", err)
	}
	if len(entries) != 2 || entries[0].code != " M" || entries[0].path != "a.go" ||
		entries[1].code != "??" || entries[1].path != "b/c.go" {
		t.Fatalf("entries = %+v", entries)
	}
	if _, err := parseGitStatusZ("bogus"); err == nil {
		t.Fatal("坏记录应报错")
	}
}

// ---- handler 入队面(payload 校验) ----

func TestV13HandlerPayloadValidation(t *testing.T) {
	kind, h := v13Handler(v13Config{}, nil)
	if kind != KindV13Task {
		t.Fatalf("kind = %s, want %s", kind, KindV13Task)
	}
	if err := h(context.Background(), json.RawMessage(`not-json`)); err == nil {
		t.Fatal("坏 JSON 应报错")
	}
	if err := h(context.Background(), json.RawMessage(`{"task":"  "}`)); err == nil {
		t.Fatal("缺 task 应报错")
	}
}

// AC2:legacy job 类型全部保留,v13 是新增的独立 kind,不覆盖任何 legacy kind。
func TestLegacyKindsRemainRegistered(t *testing.T) {
	legacy := map[jobs.Kind]bool{}
	for kind := range exec.Handlers(exec.Config{}) {
		legacy[kind] = true
	}
	for _, k := range []jobs.Kind{
		jobs.KindEnvironmentUp,
		jobs.KindEnvironmentDown,
		jobs.KindEnvironmentRebuild,
		jobs.KindDeploy,
		jobs.KindDeployRollback,
		jobs.KindDiagnose,
	} {
		if !legacy[k] {
			t.Errorf("legacy job %s 不在注册表", k)
		}
		if k == KindV13Task {
			t.Errorf("v13 kind 不得与 legacy kind 重叠")
		}
	}
}

// ---- 数据库层:每测试独立 schema,自跑迁移,cleanup DROP;不可达即 SKIP ----
// (与 internal/workflow 同一隔离策略;DATABASE_URL 可覆盖缺省 DSN。)

const v13TestDSN = "postgres://postgres:postgres@localhost:5432/tally?sslmode=disable"

func v13DBReachable(base string) bool {
	u, err := url.Parse(base)
	if err != nil {
		return false
	}
	host := u.Hostname()
	if host == "" {
		host = "localhost"
	}
	port := u.Port()
	if port == "" {
		port = "5432"
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 2*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func v13DropSchema(base, schema string) {
	u, err := url.Parse(base)
	if err != nil {
		return
	}
	q := u.Query()
	q.Del("search_path")
	u.RawQuery = q.Encode()
	db, err := sql.Open("pgx", u.String())
	if err != nil {
		return
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, _ = db.ExecContext(ctx, "DROP SCHEMA IF EXISTS "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
}

func v13TestStore(t *testing.T) (*runstore.Store, *store.Queries) {
	t.Helper()
	ctx := context.Background()
	base := os.Getenv("DATABASE_URL")
	if base == "" {
		base = v13TestDSN
	}
	if !v13DBReachable(base) {
		t.Skipf("P1-43 数据库层测试需要 PostgreSQL(%s 不可达),跳过", base)
	}
	schema := "tally_v13_" + strconv.FormatInt(time.Now().UnixNano(), 10)

	setSearchPath := func(ctx context.Context, c *pgconn.PgConn) error {
		_, err := c.Exec(ctx, "SET search_path = "+pgx.Identifier{schema}.Sanitize()).ReadAll()
		return err
	}

	boot, err := pgxpool.ParseConfig(base)
	if err != nil {
		t.Fatalf("解析 DATABASE_URL: %v", err)
	}
	boot.MaxConns = 1
	boot.ConnConfig.Config.AfterConnect = setSearchPath
	bootDB := stdlib.OpenDB(*boot.ConnConfig)
	if _, err := bootDB.ExecContext(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		bootDB.Close()
		t.Fatalf("建测试 schema %s: %v", schema, err)
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, bootDB, migrations.FS)
	if err != nil {
		bootDB.Close()
		t.Fatalf("创建 goose provider: %v", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		bootDB.Close()
		t.Fatalf("在测试 schema 跑迁移: %v", err)
	}
	bootDB.Close()

	cfg, err := pgxpool.ParseConfig(base)
	if err != nil {
		t.Fatalf("解析 DATABASE_URL(连接池): %v", err)
	}
	cfg.ConnConfig.Config.AfterConnect = setSearchPath
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("建立测试连接池: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("ping 测试库: %v", err)
	}
	st := store.New(pool)
	t.Cleanup(func() {
		pool.Close()
		v13DropSchema(base, schema)
	})
	return runstore.New(st), st
}

// 验证 sink 的落库 round trip(P2-01 起适配器在 internal/candidateverify):
// RecordRun / RecordFact 只写,ValidApplicableFacts 按 scope 精确读回
// (applicability 精确到 ChangeSet)。
func TestV13VerificationSinkRoundTrip(t *testing.T) {
	runs, _ := v13TestStore(t)
	ctx := context.Background()
	s := candidateverify.NewRunStoreSink(runs, 3)

	id, err := s.RecordRun(ctx, verification.RunSpec{
		Project:          "proj-sink",
		Task:             "TASK-SINK",
		ContractRevision: 3,
		ContractHash:     "sha256:abc",
		ChangeSetID:      "cs-1",
		EnvironmentName:  "test-env",
		EnvironmentHash:  "sha256:def",
		VersionCombo:     core.VersionCombo{"go": "test"},
		Runtime:          "fake",
		StartedAt:        time.Now().Add(-time.Minute),
		FinishedAt:       time.Now(),
		Status:           verification.RunStatusPass,
	})
	if err != nil {
		t.Fatalf("RecordRun: %v", err)
	}
	if id <= 0 {
		t.Fatalf("run id = %d", id)
	}

	err = s.RecordFact(ctx, "proj-sink", verification.FactSpec{
		FactID:           "vrun-1-VIS-1",
		RunID:            id,
		Task:             "TASK-SINK",
		Acceptance:       "AC-01",
		BindingID:        "VIS-1",
		VisibilityPolicy: verification.VisibilityVisible,
		Verdict:          verification.EvidenceStatusPass,
		Recorder:         evidence.Recorder{Kind: evidence.RecorderVerificationRun, Actor: "test"},
		Command:          &verification.CommandSpec{Argv: []string{"go", "version"}, Env: []string{"TALLY_TEST=1"}},
		ExitCode:         0,
		RecordedAt:       time.Now(),
	})
	if err != nil {
		t.Fatalf("RecordFact: %v", err)
	}

	priors, err := s.ValidApplicableFacts(ctx, evidence.Scope{
		Project:      "proj-sink",
		Task:         "TASK-SINK",
		Acceptance:   "AC-01",
		Verification: "sha256:abc",
		ChangeSet:    "cs-1",
	})
	if err != nil {
		t.Fatalf("ValidApplicableFacts: %v", err)
	}
	if len(priors) != 1 || priors[0].BindingID != "VIS-1" || priors[0].Status != verification.EvidenceStatusPass {
		t.Fatalf("priors = %+v", priors)
	}

	// scope 不匹配 → 无适用既往事实(精确 applicability,§8.8.2)。
	other, err := s.ValidApplicableFacts(ctx, evidence.Scope{
		Project:      "proj-sink",
		Task:         "TASK-SINK",
		Acceptance:   "AC-01",
		Verification: "sha256:abc",
		ChangeSet:    "cs-other",
	})
	if err != nil || len(other) != 0 {
		t.Fatalf("scope 精确性破坏: %+v %v", other, err)
	}
}

// AC1(持久化面):垂直路径端到端把 CodegenRun / ChangeSet / VerificationRun /
// evidence 全部落成 append-only 事实,且彼此可追溯(§36.3 traceability)。
func TestV13VerticalPathPersistsFacts(t *testing.T) {
	runs, st := v13TestStore(t)
	repo := newV13Repo(t)
	cfg := v13ConfigFor(t, repo)

	res, err := runV13Task(context.Background(), v13Deps{
		cfg:     cfg,
		runtime: newWritingRuntime("internal/widget/greeting.go", "package widget\n"),
		queries: st,
	}, "TASK-1043")
	if err != nil {
		t.Fatalf("runV13Task: %v", err)
	}
	if res.Codegen.Status != workflow.TaskCompleted || res.Verify == nil ||
		res.Verify.Status != verification.FinalVerified {
		t.Fatalf("垂直路径结果: codegen=%s verify=%v", res.Codegen.Status, res.Verify)
	}

	ctx := context.Background()
	timeline, err := runs.CodegenRunsForTask(ctx, "proj-v13", "TASK-1043")
	if err != nil {
		t.Fatalf("CodegenRunsForTask: %v", err)
	}
	if len(timeline) != 1 || timeline[0].Status != runstore.RunCompleted || timeline[0].ResultPatchHash == "" {
		t.Fatalf("codegen_run = %+v", timeline)
	}
	if timeline[0].ContextPackHash == "" || timeline[0].VerificationRevision == "" ||
		timeline[0].WorkspaceID == "" {
		t.Fatalf("codegen_run 观测坐标缺失: %+v", timeline[0])
	}

	css, err := runs.ChangeSetsForTask(ctx, "proj-v13", "TASK-1043")
	if err != nil || len(css) != 1 || css[0].ID() != res.Codegen.ChangeSet.ID() {
		t.Fatalf("落库 ChangeSet = %+v err=%v", css, err)
	}
	if css[0].ProducerRun() != strconv.FormatInt(timeline[0].ID, 10) {
		t.Fatalf("ChangeSet producer run = %s, want codegen_run id %d", css[0].ProducerRun(), timeline[0].ID)
	}

	vruns, err := runs.VerificationRunsForTask(ctx, "proj-v13", "TASK-1043")
	if err != nil || len(vruns) != 1 || vruns[0].Status != runstore.VerificationPass {
		t.Fatalf("verification_run = %+v err=%v", vruns, err)
	}
	if vruns[0].ContractHash != res.Verify.ContractHash ||
		vruns[0].ChangeSetID != res.Codegen.ChangeSet.ID() ||
		vruns[0].EnvDefinitionHash == "" {
		t.Fatalf("verification_run 身份绑定 = %+v", vruns[0])
	}

	evs, err := runs.EvidenceForRun(ctx, "proj-v13", vruns[0].ID)
	if err != nil || len(evs) != 1 {
		t.Fatalf("evidence = %+v err=%v", evs, err)
	}
	if evs[0].BindingID != "VIS-1" || evs[0].Verdict != evidence.VerdictPass ||
		evs[0].Recorder.Kind != evidence.RecorderVerificationRun {
		t.Fatalf("evidence = %+v", evs[0])
	}

	// AC4:主 worktree 零污染。
	if dirty := v13GitOut(t, repo, "status", "--porcelain"); dirty != "" {
		t.Errorf("主 worktree 被污染: %q", dirty)
	}
}
