package candidateverify

// P2-01 的候选验证服务测试。
//
// 覆盖任务验收:
//   - 服务接受精确的不可变上下文(冻结 VerificationContract + immutable
//     ChangeSet),缺一或缺少 runtime 身份即拒绝(AC2);
//   - 候选物化进一次性 workspace:内容照写、越出路径拒绝(INV-EXEC-01)、
//     用毕全部回收(owner 主 worktree 零污染,一次性根清空);
//   - FAIL ≠ ERROR:确定性验证失败是领域结论(FinalFailed + append-only 事实
//     落 sink),绝不是基础设施 error;
//   - held-out 边界:内容只进 Tally 控制的一次性目录,cwd 越出即拒绝。
//
// 命令用 `go version` / `git`(离线、确定性);事实面用内存 sink,数据库
// round trip 由 cmd/worker 的 TestV13VerificationSinkRoundTrip 覆盖(库不可达
// 即 SKIP,同一隔离策略)。

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"tally/internal/changeset"
	"tally/internal/core"
	"tally/internal/evidence"
	"tally/internal/runner"
	"tally/internal/runstore"
	"tally/internal/verification"
	"tally/internal/workspace"
)

// ---- 固件:真 git 仓库 + 冻结契约 + 候选 ----

func cvGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	res, err := (runner.LocalRunner{}).Run(context.Background(), runner.Command{
		Argv: append([]string{"git", "-C", dir}, args...),
	})
	if err != nil || res.ExitCode != 0 {
		t.Fatalf("git %v: err=%v res=%+v", args, err, res)
	}
}

func cvGitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	res, err := (runner.LocalRunner{}).Run(context.Background(), runner.Command{
		Argv: append([]string{"git", "-C", dir}, args...),
	})
	if err != nil || res.ExitCode != 0 {
		t.Fatalf("git %v: err=%v res=%+v", args, err, res)
	}
	return strings.TrimSpace(res.Stdout)
}

func cvWriteRepoFile(t *testing.T, repo, rel, content string) {
	t.Helper()
	p := filepath.Join(repo, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("建目录 %s: %v", rel, err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("写 %s: %v", rel, err)
	}
}

// newCVRepo 建一个带 tracked 测试文件的真 git 仓库,返回仓库根与 base 提交。
func newCVRepo(t *testing.T) (repo, base string) {
	t.Helper()
	repo = t.TempDir()
	cvGit(t, repo, "init")
	cvGit(t, repo, "config", "user.name", "tally-test")
	cvGit(t, repo, "config", "user.email", "tally-test@example.invalid")
	cvWriteRepoFile(t, repo, "internal/widget/widget_test.go", "package widget\n")
	cvGit(t, repo, "add", ".")
	cvGit(t, repo, "commit", "-m", "cv fixtures")
	return repo, cvGitOut(t, repo, "rev-parse", "HEAD")
}

// frozenVisibleContract 冻结一个单 visible binding 契约,协议命令由测试给定。
func frozenVisibleContract(t *testing.T, argv []string) *verification.Frozen {
	t.Helper()
	c := verification.Contract{
		TaskID:   "TASK-CV-1",
		Revision: 1,
		Acceptances: []verification.Acceptance{{
			ID:                   "AC-01",
			Statement:            "widget greets.",
			IndependenceRequired: verification.IndependenceNone,
			EvidenceBindings: []verification.Binding{{
				ID:               "VIS-1",
				AuthoringMode:    verification.AuthoringPreCommit,
				VisibilityPolicy: verification.VisibilityVisible,
				Evidence:         verification.EvidenceRef{Type: verification.EvidenceTest, Path: "internal/widget/widget_test.go"},
				ExecutionProtocol: &verification.ExecutionProtocol{
					Repetitions: 1,
					Aggregation: verification.AggregationAllPass,
					Command:     &verification.CommandSpec{Argv: argv},
				},
			}},
		}},
	}
	f, err := verification.FreezeContract(&c, verification.FreezePolicy{})
	if err != nil {
		t.Fatalf("冻结契约: %v", err)
	}
	return f
}

// frozenHeldOutContract 冻结一个含 held_out binding 的契约(与 worker held
// fixture 同形:声明 INV-VER-05 runtime 能力要求)。
func frozenHeldOutContract(t *testing.T) *verification.Frozen {
	t.Helper()
	c := verification.Contract{
		TaskID:              "TASK-CV-2",
		Revision:            1,
		RuntimeRequirements: verification.RuntimeRequirements{SessionIsolation: true, HeldOutUnreachable: true, ConformanceClass: "verification-isolation-v1"},
		Acceptances: []verification.Acceptance{{
			ID:                   "AC-01",
			Statement:            "hidden property holds.",
			IndependenceRequired: verification.IndependenceNone,
			EvidenceBindings: []verification.Binding{{
				ID:               "HOLD-1",
				AuthoringMode:    verification.AuthoringPreCommit,
				VisibilityPolicy: verification.VisibilityHeldOut,
				Evidence:         verification.EvidenceRef{Type: verification.EvidenceTest, Store: "holdout", Ref: "hold-1"},
				ExecutionProtocol: &verification.ExecutionProtocol{
					Repetitions: 1,
					Aggregation: verification.AggregationAllPass,
					Command:     &verification.CommandSpec{Argv: []string{"go", "version"}},
				},
			}},
		}},
	}
	f, err := verification.FreezeContract(&c, verification.FreezePolicy{})
	if err != nil {
		t.Fatalf("冻结 held-out 契约: %v", err)
	}
	return f
}

func cvCandidate(t *testing.T, base string, files ...changeset.ChangedFile) *changeset.ChangeSet {
	t.Helper()
	cs, err := changeset.New(changeset.Spec{
		Project:      "proj-cv",
		BaseRevision: base,
		Files:        files,
		ProducerRun:  "run-cv-1",
	})
	if err != nil {
		t.Fatalf("构造 ChangeSet: %v", err)
	}
	return cs
}

// cvService 装配一次性根在测试临时区的服务。
func cvService(t *testing.T, repo string, sink verification.RunSink) *Service {
	t.Helper()
	return New(Config{
		Repo:         repo,
		WorktreeRoot: filepath.Join(t.TempDir(), "wt"),
		EnvName:      "test-env",
		Recorder:     evidence.Recorder{Kind: evidence.RecorderVerificationRun, Actor: "cv-test"},
	}, Deps{Sink: sink})
}

// ---- 测试注入替身 ----

// memVerifySink 是 verification.RunSink 的内存实现(严格记录事件值)。
type memVerifySink struct {
	mu    sync.Mutex
	next  int64
	runs  []verification.RunSpec
	facts []verification.FactSpec
}

func (s *memVerifySink) RecordRun(_ context.Context, spec verification.RunSpec) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	s.runs = append(s.runs, spec)
	return s.next, nil
}

func (s *memVerifySink) RecordFact(_ context.Context, _ core.ProjectID, fact verification.FactSpec) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.facts = append(s.facts, fact)
	return nil
}

func (s *memVerifySink) ValidApplicableFacts(context.Context, evidence.Scope) ([]verification.PriorFact, error) {
	return nil, nil
}

// ---- 请求校验(AC2:精确不可变上下文) ----

func TestVerifyRequestValidation(t *testing.T) {
	repo, base := newCVRepo(t)
	frozen := frozenVisibleContract(t, []string{"go", "version"})
	cs := cvCandidate(t, base, changeset.ChangedFile{Path: "internal/widget/greeting.go", Content: "package widget\n"})
	ctx := context.Background()

	if _, err := cvService(t, repo, &memVerifySink{}).Verify(ctx, Request{ChangeSet: cs, Runtime: "fake"}); err == nil {
		t.Fatal("缺冻结契约应拒绝")
	}
	if _, err := cvService(t, repo, &memVerifySink{}).Verify(ctx, Request{Contract: frozen, Runtime: "fake"}); err == nil {
		t.Fatal("缺 ChangeSet 应拒绝")
	}
	if _, err := cvService(t, repo, &memVerifySink{}).Verify(ctx, Request{Contract: frozen, ChangeSet: cs}); err == nil {
		t.Fatal("缺 runtime 身份应拒绝")
	}
	// 事实面装配:Queries 与 Sink 皆缺 = 无处落库,拒绝执行。
	svc := New(Config{
		Repo:         repo,
		WorktreeRoot: filepath.Join(t.TempDir(), "wt"),
		EnvName:      "e",
		Recorder:     evidence.Recorder{Kind: evidence.RecorderVerificationRun, Actor: "cv-test"},
	}, Deps{})
	if _, err := svc.Verify(ctx, Request{Contract: frozen, ChangeSet: cs, Runtime: "fake"}); err == nil || !strings.Contains(err.Error(), "缺少事实存储") {
		t.Fatalf("无事实存储应拒绝: %v", err)
	}
}

// ---- visible 全链:pass 是 verified;fail 是 FinalFailed 领域结论(FAIL ≠ ERROR) ----

func TestVerifyVisiblePass(t *testing.T) {
	repo, base := newCVRepo(t)
	frozen := frozenVisibleContract(t, []string{"go", "version"})
	cs := cvCandidate(t, base, changeset.ChangedFile{Path: "internal/widget/greeting.go", Content: "package widget\n"})
	worktreeRoot := filepath.Join(t.TempDir(), "wt")
	sink := &memVerifySink{}
	svc := New(Config{
		Repo:         repo,
		WorktreeRoot: worktreeRoot,
		EnvName:      "test-env",
		Recorder:     evidence.Recorder{Kind: evidence.RecorderVerificationRun, Actor: "cv-test"},
	}, Deps{Sink: sink})

	out, err := svc.Verify(context.Background(), Request{Contract: frozen, ChangeSet: cs, Runtime: "fake"})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if out.Status != verification.FinalVerified {
		t.Fatalf("结论 = %s, want verified: bindings=%+v", out.Status, out.Bindings)
	}
	if out.TaskID != "TASK-CV-1" || out.ChangeSetID != cs.ID() || out.ContractHash != frozen.Hash() {
		t.Fatalf("结论身份绑定 = %+v", out)
	}
	if len(sink.runs) != 1 || sink.runs[0].Status != verification.RunStatusPass {
		t.Fatalf("VerificationRun 落库 = %+v", sink.runs)
	}
	if len(sink.facts) != 1 {
		t.Fatalf("evidence 事实数 = %d, want 1", len(sink.facts))
	}
	fact := sink.facts[0]
	if fact.BindingID != "VIS-1" || fact.Acceptance != "AC-01" ||
		fact.Verdict != verification.EvidenceStatusPass ||
		fact.Recorder.Kind != evidence.RecorderVerificationRun || fact.Recorder.Actor != "cv-test" ||
		fact.Command == nil || fact.Command.Argv[0] != "go" {
		t.Fatalf("evidence 事实 = %+v", fact)
	}
	// 用毕回收:一次性根应清空(INV-EXEC-01 的另一半)。
	entries, err := os.ReadDir(worktreeRoot)
	if err != nil || len(entries) != 0 {
		t.Errorf("一次性根目录应清空: entries=%d err=%v", len(entries), err)
	}
	// owner 主 worktree 零污染。
	if dirty := cvGitOut(t, repo, "status", "--porcelain"); dirty != "" {
		t.Errorf("主 worktree 被污染: %q", dirty)
	}
}

func TestVerifyVisibleFailIsDomainResultNotError(t *testing.T) {
	repo, base := newCVRepo(t)
	// 确定性失败:解析一个不存在的 ref,exit 非 0,离线稳定。
	frozen := frozenVisibleContract(t, []string{"git", "rev-parse", "--verify", "refs/heads/cv-definitely-missing"})
	cs := cvCandidate(t, base, changeset.ChangedFile{Path: "internal/widget/greeting.go", Content: "package widget\n"})
	worktreeRoot := filepath.Join(t.TempDir(), "wt")
	sink := &memVerifySink{}
	svc := New(Config{
		Repo:         repo,
		WorktreeRoot: worktreeRoot,
		EnvName:      "test-env",
		Recorder:     evidence.Recorder{Kind: evidence.RecorderVerificationRun, Actor: "cv-test"},
	}, Deps{Sink: sink})

	out, err := svc.Verify(context.Background(), Request{Contract: frozen, ChangeSet: cs, Runtime: "fake"})
	if err != nil {
		t.Fatalf("确定性 FAIL 不得作为 error 冒泡: %v", err)
	}
	if out.Status != verification.FinalFailed {
		t.Fatalf("结论 = %s, want failed", out.Status)
	}
	if len(sink.runs) != 1 || sink.runs[0].Status != verification.RunStatusFail {
		t.Fatalf("VerificationRun 落库 = %+v", sink.runs)
	}
	if len(sink.facts) != 1 || sink.facts[0].Verdict != verification.EvidenceStatusFail {
		t.Fatalf("失败 evidence 事实 = %+v", sink.facts)
	}
	if entries, err := os.ReadDir(worktreeRoot); err != nil || len(entries) != 0 {
		t.Errorf("失败后一次性根目录应清空: entries=%d err=%v", len(entries), err)
	}
}

// ---- 候选物化:照写改后完整内容;越出路径拒绝(INV-EXEC-01) ----

func TestMaterializeCandidate(t *testing.T) {
	repo, base := newCVRepo(t)
	ctx := context.Background()
	mgr, err := workspace.New(repo, filepath.Join(t.TempDir(), "wt"))
	if err != nil {
		t.Fatalf("workspace.New: %v", err)
	}
	cs := cvCandidate(t, base,
		changeset.ChangedFile{Path: "internal/widget/greeting.go", Content: "package widget\n\nfunc Hi() {}\n"},
		changeset.ChangedFile{Path: "docs/note.md", Content: "note\n"})
	wt, err := materializeCandidate(ctx, mgr, cs)
	if err != nil {
		t.Fatalf("materializeCandidate: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(wt.Path, "internal", "widget", "greeting.go"))
	if err != nil || string(got) != "package widget\n\nfunc Hi() {}\n" {
		t.Fatalf("物化内容 = %q err=%v", got, err)
	}
	if _, err := os.Stat(filepath.Join(wt.Path, "docs", "note.md")); err != nil {
		t.Fatalf("第二改动文件未物化: %v", err)
	}
	if err := mgr.Cleanup(ctx, wt); err != nil {
		t.Fatalf("清理: %v", err)
	}
}

func TestMaterializeCandidateRejectsEscapePath(t *testing.T) {
	repo, base := newCVRepo(t)
	ctx := context.Background()
	mgr, err := workspace.New(repo, filepath.Join(t.TempDir(), "wt"))
	if err != nil {
		t.Fatalf("workspace.New: %v", err)
	}
	cs := cvCandidate(t, base, changeset.ChangedFile{Path: "../evil.txt", Content: "no\n"})
	if _, err := materializeCandidate(ctx, mgr, cs); err == nil || !strings.Contains(err.Error(), "越出") {
		t.Fatalf("越出 worktree 的候选路径应拒绝: %v", err)
	}
}

// ---- held-out 边界 ----

func TestHoldoutBoundary(t *testing.T) {
	b := holdoutBoundary(runner.LocalRunner{})
	ctx := context.Background()

	// 正常路径:内容物化为边界目录下的 holdout-input,命令在边界内执行,
	// 返回结构化 status + 输出摘要(原始输出不出边界)。
	out, err := b(ctx, verification.HoldoutRunRequest{
		Content: []byte("hidden-bytes"),
		Command: &verification.CommandSpec{Argv: []string{"git", "hash-object", "holdout-input"}},
	})
	if err != nil || out.Status != verification.EvidenceStatusPass {
		t.Fatalf("held-out 执行 = %+v err=%v", out, err)
	}
	if !strings.HasPrefix(out.OutputDigest, "sha256:") {
		t.Fatalf("输出摘要 = %q", out.OutputDigest)
	}

	// 失败路径:非零 exit = fail(不是 error)。
	out, err = b(ctx, verification.HoldoutRunRequest{
		Content: []byte("x"),
		Command: &verification.CommandSpec{Argv: []string{"git", "rev-parse", "--verify", "refs/heads/cv-missing"}},
	})
	if err != nil || out.Status != verification.EvidenceStatusFail {
		t.Fatalf("非零 exit 应是 fail: %+v err=%v", out, err)
	}

	// nil 冻结命令拒绝。
	if _, err := b(ctx, verification.HoldoutRunRequest{Content: []byte("x")}); err == nil {
		t.Fatal("缺冻结命令应拒绝")
	}
	// cwd 越出边界目录拒绝(相对 ..、浅层 ..、深层回穿、OS 绝对路径)。
	for _, d := range []string{"../outside", "..", "sub/../../outside", t.TempDir()} {
		if _, err := b(ctx, verification.HoldoutRunRequest{
			Content: []byte("x"),
			Command: &verification.CommandSpec{Argv: []string{"go", "version"}, Dir: d},
		}); err == nil || !strings.Contains(err.Error(), "越出") {
			t.Errorf("cwd %q 越出应拒绝: %v", d, err)
		}
	}
}

// ---- ContractHasHeldOut ----

func TestContractHasHeldOut(t *testing.T) {
	if ContractHasHeldOut(frozenVisibleContract(t, []string{"go", "version"})) {
		t.Fatal("visible 契约不应含 held_out")
	}
	if !ContractHasHeldOut(frozenHeldOutContract(t)) {
		t.Fatal("held-out 契约应被识别")
	}
}

// ---- RunSink→runstore 适配的纯转换面 ----

func TestCommandSpecConversion(t *testing.T) {
	if commandSpec(nil) != nil {
		t.Fatal("nil 命令应转 nil")
	}
	got := commandSpec(&verification.CommandSpec{
		Argv:    []string{"go", "test", "./..."},
		Dir:     "internal/widget",
		Env:     []string{"A=1", "B=2", "bad", "C=3"},
		Timeout: verification.Duration(1_000_000_000),
	})
	if got.Argv[0] != "go" || got.Cwd != "internal/widget" || got.Shell || got.Timeout != 1_000_000_000 ||
		got.Env["A"] != "1" || got.Env["B"] != "2" || got.Env["C"] != "3" || len(got.Env) != 3 {
		t.Fatalf("commandSpec = %+v", got)
	}
}

func TestRunstoreStatusMapping(t *testing.T) {
	for _, tc := range []struct {
		in   verification.RunStatus
		want runstore.VerificationStatus
	}{
		{verification.RunStatusPass, runstore.VerificationPass},
		{verification.RunStatusFail, runstore.VerificationFail},
		{verification.RunStatus("weird"), runstore.VerificationError},
	} {
		if got := runstoreStatus(tc.in); got != tc.want {
			t.Errorf("runstoreStatus(%s) = %s, want %s", tc.in, got, tc.want)
		}
	}
}
