package workflow

// P1-39 内核测试(与 v13.go 同包)。
//
// 覆盖任务验收:
//   - AC1 单 Task 顺序生命周期:Run 顺序开跑/终结,sink 事件严格成序;
//   - AC2 一个 CodegenRun 恰好 0..1 个固定候选 ChangeSet(成功恰 1 个,
//     预算耗尽/升级/停止/取消的 Run 恒为 0);
//   - AC3 升级终止当前 Run(status: escalated)并开新 Run(记 escalated_from);
//     无目标/重复目标 → NEEDS_REVIEW,循环必须有限;
//   - AC4 无 DAG/并行调度/通用编排:Execute 一次性、顺序推进、对抗性升级
//     钩子也被链长上限终止。
//
// 单元层不需要数据库;RunStoreSink 的落库 round trip 由文件尾部的数据库
// 测试覆盖(库不可达即 SKIP,与 internal/runstore 同一隔离策略)。

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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
	"tally/internal/changeset"
	"tally/internal/policy"
	"tally/internal/runner"
	"tally/internal/runstore"
	"tally/internal/store"
	"tally/internal/taskcontract"
	"tally/internal/workspace"
	"tally/migrations"
	"tally/runtimes/fake"
)

var testAt = time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

// ---- 固件 ----

func testContract() taskcontract.TaskContract {
	return taskcontract.TaskContract{
		ID:        "TASK-WF39",
		Profile:   taskcontract.ProfileStandard,
		Title:     "Workflow kernel fixture",
		Objective: "Implement the widget scheduler",
		Scope:     taskcontract.Scope{AllowedPaths: []string{"internal/widget"}},
		Acceptance: []taskcontract.AcceptanceClause{
			{ID: "AC-01", Statement: "widget list renders", IndependenceRequired: taskcontract.IndependenceNone},
		},
		Hash: "sha256:wf39-task-revision",
	}
}

func testPolicy() policy.ExecutionPolicy {
	p := policy.DefaultExecutionPolicy()
	p.Retry.MaxIterations = 4
	p.Escalation.AfterFailedIterations = 2
	return p
}

// memSink 是 RunSink 的内存实现:记录严格顺序的事件日志与全部事实值。
type memSink struct {
	events     []string
	begins     map[RunID]RunBegin
	finishes   map[RunID]RunFinish
	changeSets map[RunID]*changeset.ChangeSet
	// failOn 非空时,对应阶段返回错误(基础设施故障路径测试)。
	failOn string
}

func newMemSink() *memSink {
	return &memSink{
		begins:     map[RunID]RunBegin{},
		finishes:   map[RunID]RunFinish{},
		changeSets: map[RunID]*changeset.ChangeSet{},
	}
}

func (s *memSink) log(ev string) { s.events = append(s.events, ev) }

func (s *memSink) BeginRun(_ context.Context, b RunBegin) (string, error) {
	if s.failOn == "begin" {
		return "", errors.New("sink begin failure")
	}
	s.log("begin:" + string(b.ID))
	s.begins[b.ID] = b
	return string(b.ID), nil
}

func (s *memSink) RecordProgress(_ context.Context, id RunID, p RunProgress) error {
	if s.failOn == "progress" {
		return errors.New("sink progress failure")
	}
	s.log(fmt.Sprintf("progress:%s:%d", id, p.Iterations))
	return nil
}

func (s *memSink) FinishRun(_ context.Context, f RunFinish) error {
	if s.failOn == "finish" {
		return errors.New("sink finish failure")
	}
	s.log(fmt.Sprintf("finish:%s:%s", f.ID, f.Status))
	s.finishes[f.ID] = f
	return nil
}

func (s *memSink) RecordChangeSet(_ context.Context, id RunID, cs *changeset.ChangeSet) error {
	if s.failOn == "changeset" {
		return errors.New("sink changeset failure")
	}
	s.log("changeset:" + string(id))
	s.changeSets[id] = cs
	return nil
}

func (s *memSink) indexOf(prefix string) int {
	for i, ev := range s.events {
		if strings.HasPrefix(ev, prefix) {
			return i
		}
	}
	return -1
}

// execFixture 汇齐一次测试执行:真实 git 仓库 + 一次性 worktree 表 + 内存
// sink + 计数的候选钩子。tweak 允许单测改写 spec。
type execFixture struct {
	exec       *TaskExecution
	sink       *memSink
	base       string
	repo       string
	fixCalls   int
	fixViews   []IterationView
	tweakCalls int
}

func newFixture(t *testing.T, tweak func(spec *TaskExecutionSpec, f *execFixture)) *execFixture {
	t.Helper()
	repo := newWFRepo(t)
	base := wfCommit(t, repo, "README.md", "base\n", "base")
	mgr, err := workspace.New(repo, filepath.Join(t.TempDir(), "wt"))
	if err != nil {
		t.Fatalf("workspace.New: %v", err)
	}

	f := &execFixture{sink: newMemSink(), base: base, repo: repo}
	spec := TaskExecutionSpec{
		Project:              "proj-39",
		Task:                 testContract(),
		Role:                 "coding",
		Policy:               testPolicy(),
		BaseRevision:         base,
		Profile:              "default-coding",
		Runtime:              fake.New(fake.Config{}),
		Model:                "test-model",
		BehaviorRevision:     "behavior-1",
		VerificationRevision: "verrev-1",
		ContextPackHash:      "ctxhash-1",
		Worktrees:            workspace.NewRunTable(mgr),
		Sink:                 f.sink,
		Hooks: Hooks{
			FixCandidate: func(_ context.Context, v IterationView) (CandidateSnapshot, bool, error) {
				f.fixCalls++
				f.fixViews = append(f.fixViews, v)
				return CandidateSnapshot{
					CandidateRevision: "cand-" + string(v.Run),
					Files: []changeset.ChangedFile{
						{Path: "internal/widget/" + string(v.Run) + ".go", Content: "package widget\n"},
					},
					Entities: []string{"widget"},
				}, true, nil
			},
		},
		Now: func() time.Time { return testAt },
	}
	if tweak != nil {
		f.tweakCalls++
		tweak(&spec, f)
	}
	exec, err := NewTaskExecution(spec)
	if err != nil {
		t.Fatalf("NewTaskExecution: %v", err)
	}
	f.exec = exec
	return f
}

// escalateTo 返回一个把升级指向 extra runtime 的解析钩子(记录每次入参)。
func escalateTo(f *execFixture, profile string, rt agentruntime.Runtime, calls *[][]string) EscalationResolveFunc {
	return func(_ context.Context, used []string) (EscalationTarget, bool, error) {
		cp := append([]string(nil), used...)
		*calls = append(*calls, cp)
		return EscalationTarget{
			Profile:          profile,
			Runtime:          rt,
			Model:            "escalated-model",
			BehaviorRevision: "behavior-2",
		}, true, nil
	}
}

// ---- AC1:单 Task 顺序生命周期 ----

func TestExecuteCompletesWithSingleCandidate(t *testing.T) {
	f := newFixture(t, nil)

	out, err := f.exec.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Status != TaskCompleted {
		t.Fatalf("Status = %q, want completed", out.Status)
	}
	if len(out.Runs) != 1 {
		t.Fatalf("Runs = %d, want 1(单 Task 顺序执行)", len(out.Runs))
	}
	rec := out.Runs[0]
	if rec.ID != "run-0001" || rec.Status != runstore.RunCompleted || rec.EscalatedFrom != "" {
		t.Fatalf("RunRecord = %+v, want run-0001/completed/无升级来源", rec)
	}
	if rec.Iterations != 1 {
		t.Fatalf("Iterations = %d, want 1", rec.Iterations)
	}
	if rec.WorkspacePath == "" || pathExists(rec.WorkspacePath) {
		t.Fatalf("worktree %q 未随 Run 终止销毁", rec.WorkspacePath)
	}

	// AC2:恰好一个固定候选,身份由内容推导并绑定 producer run。
	cs := out.ChangeSet
	if cs == nil {
		t.Fatal("ChangeSet = nil, want 固定候选")
	}
	if cs.ProducerRun() != string(rec.ID) || cs.BaseRevision() != f.base {
		t.Fatalf("ChangeSet producer/base = %s/%s, want run-0001/%s", cs.ProducerRun(), cs.BaseRevision(), f.base)
	}
	if cs.CandidateRevision() != "cand-run-0001" {
		t.Fatalf("CandidateRevision = %q", cs.CandidateRevision())
	}
	if rec.ChangeSetID != cs.ID() {
		t.Fatalf("record ChangeSetID = %s, want %s", rec.ChangeSetID, cs.ID())
	}

	// sink 事实:begin → progress → finish → changeset,严格成序。
	if got := f.sink.indexOf("begin:run-0001"); got == -1 ||
		got > f.sink.indexOf("progress:run-0001:1") ||
		f.sink.indexOf("progress:run-0001:1") > f.sink.indexOf("changeset:run-0001") ||
		f.sink.indexOf("changeset:run-0001") > f.sink.indexOf("finish:run-0001:completed") {
		t.Fatalf("sink 事件顺序错误: %v", f.sink.events)
	}

	// 开跑事实携带全部身份坐标(§15.2)。
	b := f.sink.begins["run-0001"]
	if b.Project != "proj-39" || b.TaskID != "TASK-WF39" || b.TaskRevision != testContract().Hash ||
		b.Role != "coding" || b.Profile != "default-coding" || b.RuntimeName != "fake" ||
		b.Model != "test-model" || b.BehaviorRevision != "behavior-1" ||
		b.BaseRevision != f.base || b.WorkspaceID != rec.WorkspacePath ||
		b.VerificationRevision != "verrev-1" || b.ContextPackHash != "ctxhash-1" {
		t.Fatalf("RunBegin = %+v", b)
	}
	// FakeRuntime 不实现 IdentityReporter:实际身份必须保持未观察零值(P0-01 AC3)。
	if b.ObservedIdentity.Observed() {
		t.Fatalf("未实现 IdentityReporter 的会话,观察身份必须为零值: %+v", b.ObservedIdentity)
	}
	var snap struct {
		Name         string   `json:"name"`
		Capabilities []string `json:"capabilities"`
	}
	if err := json.Unmarshal(b.CapabilitySnapshot, &snap); err != nil || snap.Name != "fake" || len(snap.Capabilities) == 0 {
		t.Fatalf("CapabilitySnapshot = %s (%v)", b.CapabilitySnapshot, err)
	}

	// 终态事实:usage 记账来自 runtime 自报(claim)。
	fin := f.sink.finishes["run-0001"]
	if fin.Status != runstore.RunCompleted || fin.ResultPatchHash != string(cs.ID()) {
		t.Fatalf("RunFinish = %+v", fin)
	}
	if fin.InputTokens == nil || *fin.InputTokens != 10 || fin.OutputTokens == nil || *fin.OutputTokens != 5 {
		t.Fatalf("usage = %v/%v, want fake 默认 10/5", fin.InputTokens, fin.OutputTokens)
	}
	if !strings.Contains(fin.AgentClaim, "completed") {
		t.Fatalf("AgentClaim = %q, want 原样保留的 runtime 自述", fin.AgentClaim)
	}
}

func TestRetryThenCompleteWithinOneRun(t *testing.T) {
	// 失败 1 轮 + 完成 1 轮,阈值 2 之内:同一 Run 内有限重试。
	rt := fake.New(fake.Config{Turns: []fake.TurnSpec{{Behavior: fake.TurnFail}, {}}})
	f := newFixture(t, func(spec *TaskExecutionSpec, f *execFixture) { spec.Runtime = rt })

	out, err := f.exec.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Status != TaskCompleted || len(out.Runs) != 1 {
		t.Fatalf("Status/Runs = %s/%d, want completed/1", out.Status, len(out.Runs))
	}
	if out.Runs[0].Iterations != 2 {
		t.Fatalf("Iterations = %d, want 2", out.Runs[0].Iterations)
	}
	if f.fixCalls != 1 {
		t.Fatalf("FixCandidate 调用次数 = %d, want 1", f.fixCalls)
	}
}

func TestInstructionCarriesContractAndRetryMark(t *testing.T) {
	first := instructionFor(testContract(), 1)
	if !strings.Contains(first, "TASK-WF39") || !strings.Contains(first, "Implement the widget scheduler") ||
		!strings.Contains(first, "AC-01: widget list renders") {
		t.Fatalf("首轮指令缺少合同内容: %q", first)
	}
	if strings.Contains(first, "retry iteration") {
		t.Fatalf("首轮指令不应带重试标记: %q", first)
	}
	retry := instructionFor(testContract(), 2)
	if !strings.Contains(retry, "retry iteration 2") {
		t.Fatalf("重试指令缺少标记: %q", retry)
	}
}

// ---- AC2:一个 Run 恰好 0..1 个固定候选 ChangeSet ----

func TestBudgetExhaustedRunProducesNoChangeSet(t *testing.T) {
	rt := fake.New(fake.Config{
		Turns: []fake.TurnSpec{{Usage: &agentruntime.Usage{InputTokens: 100, OutputTokens: 100}}},
	})
	f := newFixture(t, func(spec *TaskExecutionSpec, _ *execFixture) {
		spec.Runtime = rt
		spec.Policy.Budgets.MaxTokens = 10
	})

	out, err := f.exec.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	// §11.2.1:预算耗尽的 Run 不产生 ChangeSet;§29:进入 NEEDS_REVIEW。
	if out.Status != TaskNeedsReview || out.Reason != ReviewRunAborted {
		t.Fatalf("Status/Reason = %s/%s, want needs_review/run_aborted", out.Status, out.Reason)
	}
	rec := out.Runs[0]
	if rec.Status != runstore.RunFailed || rec.FailureReason != "token_budget_exhausted" {
		t.Fatalf("RunRecord = %+v", rec)
	}
	if rec.ChangeSetID != "" || out.ChangeSet != nil || f.fixCalls != 0 {
		t.Fatalf("预算耗尽 Run 不得产生候选: id=%s cs=%v fix=%d", rec.ChangeSetID, out.ChangeSet, f.fixCalls)
	}
	if fin := f.sink.finishes[rec.ID]; fin.ResultPatchHash != "" {
		t.Fatalf("ResultPatchHash = %q, want 空", fin.ResultPatchHash)
	}
}

func TestWallTimeExhaustedRunAborts(t *testing.T) {
	rt := fake.New(fake.Config{Turns: []fake.TurnSpec{{Behavior: fake.TurnWaitCancel}}})
	f := newFixture(t, func(spec *TaskExecutionSpec, _ *execFixture) {
		spec.Runtime = rt
		spec.Policy.Budgets.MaxWallTime = 50 * time.Millisecond
	})

	out, err := f.exec.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Status != TaskNeedsReview || out.Reason != ReviewRunAborted {
		t.Fatalf("Status/Reason = %s/%s, want needs_review/run_aborted", out.Status, out.Reason)
	}
	if rec := out.Runs[0]; rec.Status != runstore.RunFailed || rec.FailureReason != "wall_time_budget_exhausted" {
		t.Fatalf("RunRecord = %+v", rec)
	}
	if out.ChangeSet != nil {
		t.Fatal("墙钟耗尽 Run 不得产生候选")
	}
}

func TestCancelledRunProducesNoChangeSet(t *testing.T) {
	// 调用方在迭代中途取消:Run 记 cancelled,任务收束为 cancelled。
	// gateRuntime 保证在首个迭代真正开始后才取消,测试完全确定。
	rt := fake.New(fake.Config{Turns: []fake.TurnSpec{{Behavior: fake.TurnWaitCancel}}})
	started := make(chan struct{})
	f := newFixture(t, func(spec *TaskExecutionSpec, _ *execFixture) {
		spec.Runtime = &gateRuntime{inner: rt, started: started}
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type result struct {
		out Outcome
		err error
	}
	ch := make(chan result, 1)
	go func() {
		out, err := f.exec.Execute(ctx)
		ch <- result{out, err}
	}()
	<-started
	cancel()

	select {
	case r := <-ch:
		if r.err != nil {
			t.Fatalf("Execute: %v", r.err)
		}
		if r.out.Status != TaskCancelled {
			t.Fatalf("Status = %q, want cancelled", r.out.Status)
		}
		if rec := r.out.Runs[0]; rec.Status != runstore.RunCancelled || r.out.ChangeSet != nil {
			t.Fatalf("RunRecord/ChangeSet = %+v/%v", rec, r.out.ChangeSet)
		}
		if pathExists(r.out.Runs[0].WorkspacePath) {
			t.Fatal("取消后 worktree 未销毁")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Execute 未在取消后收束")
	}
}

func TestPreCancelledContextYieldsCancelledOutcomeWithoutRuns(t *testing.T) {
	f := newFixture(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	out, err := f.exec.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Status != TaskCancelled || len(out.Runs) != 0 {
		t.Fatalf("Status/Runs = %s/%d, want cancelled/0", out.Status, len(out.Runs))
	}
	if len(f.sink.events) != 0 {
		t.Fatalf("未开跑即取消不应有事实写入: %v", f.sink.events)
	}
}

// ---- AC3:升级终止当前 Run 并开新 Run ----

func TestEscalationTerminatesRunAndOpensEscalatedRun(t *testing.T) {
	weak := fake.New(fake.Config{DefaultTurn: fake.TurnSpec{Behavior: fake.TurnFail}})
	strong := fake.New(fake.Config{})
	f := newFixture(t, func(spec *TaskExecutionSpec, f *execFixture) {
		spec.Runtime = weak
		var calls [][]string
		spec.Hooks.ResolveEscalation = escalateTo(f, "strong-coding", strong, &calls)
		spec.Hooks.FixCandidate = func(_ context.Context, v IterationView) (CandidateSnapshot, bool, error) {
			if v.Profile != "strong-coding" {
				return CandidateSnapshot{}, false, nil // 弱 runtime 的迭代无可固定候选
			}
			f.fixCalls++
			return CandidateSnapshot{CandidateRevision: "cand-" + string(v.Run)}, true, nil
		}
	})

	out, err := f.exec.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Status != TaskCompleted {
		t.Fatalf("Status = %q, want completed", out.Status)
	}
	if len(out.Runs) != 2 {
		t.Fatalf("Runs = %d, want 2", len(out.Runs))
	}
	first, second := out.Runs[0], out.Runs[1]
	// §11.2.1:当前 Run status: escalated 并终止。
	if first.Status != runstore.RunEscalated || first.FailureReason != "escalated_to:strong-coding" {
		t.Fatalf("first = %+v, want escalated", first)
	}
	if first.ChangeSetID != "" {
		t.Fatalf("升级终止的 Run 不得产生候选: %s", first.ChangeSetID)
	}
	// 新 Run 记录 escalated_from,并以升级后的归因坐标运行。
	if second.EscalatedFrom != first.ID || second.Profile != "strong-coding" || second.Status != runstore.RunCompleted {
		t.Fatalf("second = %+v", second)
	}
	if b := f.sink.begins[second.ID]; b.EscalatedFrom != first.ID || b.Model != "escalated-model" || b.BehaviorRevision != "behavior-2" {
		t.Fatalf("升级 Run 的 BeginRun = %+v", b)
	}
	if b := f.sink.begins[first.ID]; b.EscalatedFrom != "" {
		t.Fatalf("首个 Run 不是升级 Run: %+v", b)
	}
	// 两个 Run 顺序执行:begin/finish 严格成序,worktree 各自独立且已销毁。
	order := []string{
		"begin:" + string(first.ID), "finish:" + string(first.ID) + ":escalated",
		"begin:" + string(second.ID), "finish:" + string(second.ID) + ":completed",
	}
	pos := 0
	for _, want := range order {
		idx := f.sink.indexOf(want)
		if idx < pos {
			t.Fatalf("事件 %q 未严格顺序出现: %v", want, f.sink.events)
		}
		pos = idx
	}
	if first.WorkspacePath == second.WorkspacePath {
		t.Fatal("两个 Run 不得共享 worktree")
	}
	if pathExists(first.WorkspacePath) || pathExists(second.WorkspacePath) {
		t.Fatal("两个 Run 的 worktree 都应已销毁")
	}
	// 升级 Run 与首个 Run 同基线(ChangeSet 不跨 Run 累积,§15.3)。
	if b := f.sink.begins[second.ID]; b.BaseRevision != f.base {
		t.Fatalf("升级 Run base = %q, want %q", b.BaseRevision, f.base)
	}
	// AC2:整条链上只有最终完成的 Run 固定了唯一候选。
	if out.ChangeSet == nil || out.ChangeSet.ProducerRun() != string(second.ID) || len(f.sink.changeSets) != 1 {
		t.Fatalf("候选固定错误: cs=%v sink=%d", out.ChangeSet, len(f.sink.changeSets))
	}
}

func TestEscalationUnavailableNeedsReview(t *testing.T) {
	weak := fake.New(fake.Config{DefaultTurn: fake.TurnSpec{Behavior: fake.TurnFail}})
	// 无升级钩子 = 无目标(§29:无 target → NEEDS_REVIEW)。
	f := newFixture(t, func(spec *TaskExecutionSpec, _ *execFixture) {
		spec.Runtime = weak
	})

	out, err := f.exec.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Status != TaskNeedsReview || out.Reason != ReviewEscalationUnavailable {
		t.Fatalf("Status/Reason = %s/%s, want needs_review/escalation_unavailable", out.Status, out.Reason)
	}
	if rec := out.Runs[0]; rec.Status != runstore.RunFailed || rec.FailureReason != "escalation_unavailable" || rec.ChangeSetID != "" {
		t.Fatalf("RunRecord = %+v", rec)
	}
	if len(out.Runs) != 1 {
		t.Fatalf("无目标时不得再开 Run: %d", len(out.Runs))
	}
}

func TestRepeatedEscalationTargetIsTreatedAsUnavailable(t *testing.T) {
	// 对抗性钩子永远返回同一个已用 profile:必须视为无目标,循环终止。
	weak := fake.New(fake.Config{DefaultTurn: fake.TurnSpec{Behavior: fake.TurnFail}})
	var calls int
	f := newFixture(t, func(spec *TaskExecutionSpec, _ *execFixture) {
		spec.Runtime = weak
		spec.Policy.Retry.MaxIterations = 2
		spec.Policy.Escalation.AfterFailedIterations = 1
		spec.Hooks.ResolveEscalation = func(_ context.Context, used []string) (EscalationTarget, bool, error) {
			calls++
			return EscalationTarget{
				Profile:          used[0], // 重复已用 profile
				Runtime:          weak,
				Model:            "m",
				BehaviorRevision: "b",
			}, true, nil
		}
	})

	out, err := f.exec.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Status != TaskNeedsReview || out.Reason != ReviewEscalationUnavailable {
		t.Fatalf("Status/Reason = %s/%s, want needs_review/escalation_unavailable", out.Status, out.Reason)
	}
	if len(out.Runs) != 1 || calls != 1 {
		t.Fatalf("Runs/Hook 调用 = %d/%d, want 1/1(重复目标一次即停)", len(out.Runs), calls)
	}
}

func TestAdversarialEscalationChainIsBounded(t *testing.T) {
	// 对抗性钩子不断产出新 profile:链长上限必须终止循环(§29 不得无限循环)。
	weak := fake.New(fake.Config{DefaultTurn: fake.TurnSpec{Behavior: fake.TurnFail}})
	f := newFixture(t, func(spec *TaskExecutionSpec, _ *execFixture) {
		spec.Runtime = weak
		spec.Policy.Retry.MaxIterations = 1
		spec.Policy.Escalation.AfterFailedIterations = 1
		spec.Hooks.ResolveEscalation = func(_ context.Context, used []string) (EscalationTarget, bool, error) {
			return EscalationTarget{
				Profile:          "esc-" + strconv.Itoa(len(used)), // 永远是新名字
				Runtime:          weak,
				Model:            "m",
				BehaviorRevision: "b",
			}, true, nil
		}
	})

	out, err := f.exec.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Status != TaskNeedsReview || out.Reason != ReviewEscalationUnavailable {
		t.Fatalf("Status/Reason = %s/%s", out.Status, out.Reason)
	}
	if len(out.Runs) != escalationCap {
		t.Fatalf("Runs = %d, want 链长上限 %d(有限终止)", len(out.Runs), escalationCap)
	}
	for i, rec := range out.Runs {
		want := runstore.RunEscalated
		if i == len(out.Runs)-1 {
			want = runstore.RunFailed
		}
		if rec.Status != want {
			t.Fatalf("Runs[%d].Status = %s, want %s", i, rec.Status, want)
		}
	}
}

func TestRuntimeCrashBelowBoundaryAborts(t *testing.T) {
	// crash 迭代以 RunFailed claim 计失败,下一轮对已关闭会话的调用按
	// runtime_error 再计一次;阈值取 3,使终止发生在升级边界之前:
	// 策略未到升级点,内核不越权升级,Run 记 failed + NEEDS_REVIEW。
	rt := fake.New(fake.Config{Turns: []fake.TurnSpec{{Behavior: fake.TurnCrash}}})
	f := newFixture(t, func(spec *TaskExecutionSpec, _ *execFixture) {
		spec.Runtime = rt
		spec.Policy.Escalation.AfterFailedIterations = 3
	})

	out, err := f.exec.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Status != TaskNeedsReview || out.Reason != ReviewRunAborted {
		t.Fatalf("Status/Reason = %s/%s, want needs_review/run_aborted", out.Status, out.Reason)
	}
	if rec := out.Runs[0]; rec.Status != runstore.RunFailed || rec.FailureReason != "runtime_error" || rec.Iterations != 2 {
		t.Fatalf("RunRecord = %+v", rec)
	}
}

func TestRuntimeCrashAtBoundaryEscalates(t *testing.T) {
	// crash 恰好跨过阈值(1):按 §11.2.1 记 escalated 并升级到新 runtime。
	weak := fake.New(fake.Config{Turns: []fake.TurnSpec{{Behavior: fake.TurnCrash}}})
	strong := fake.New(fake.Config{})
	var calls [][]string
	f := newFixture(t, func(spec *TaskExecutionSpec, f *execFixture) {
		spec.Runtime = weak
		spec.Policy.Escalation.AfterFailedIterations = 1
		spec.Hooks.ResolveEscalation = escalateTo(f, "strong-coding", strong, &calls)
	})

	out, err := f.exec.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Status != TaskCompleted || len(out.Runs) != 2 {
		t.Fatalf("Status/Runs = %s/%d, want completed/2", out.Status, len(out.Runs))
	}
	if out.Runs[0].Status != runstore.RunEscalated || out.Runs[0].FailureReason != "escalated_to:strong-coding" {
		t.Fatalf("first = %+v", out.Runs[0])
	}
	if len(calls) != 1 || !equalStrings(calls[0], []string{"default-coding"}) {
		t.Fatalf("升级钩子入参 = %v, want [default-coding] 且恰好一次", calls)
	}
}

// ---- 停止条件 ----

func TestStopConditionStopsRunImmediately(t *testing.T) {
	f := newFixture(t, func(spec *TaskExecutionSpec, _ *execFixture) {
		spec.Hooks.StopProbe = func(_ context.Context, v IterationView) ([]policy.StopCondition, error) {
			return []policy.StopCondition{policy.StopForbiddenPathModified}, nil
		}
	})

	out, err := f.exec.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Status != TaskNeedsReview || out.Reason != ReviewStopCondition {
		t.Fatalf("Status/Reason = %s/%s, want needs_review/stop_condition", out.Status, out.Reason)
	}
	if len(out.StopConditions) != 1 || out.StopConditions[0] != policy.StopForbiddenPathModified {
		t.Fatalf("StopConditions = %v", out.StopConditions)
	}
	rec := out.Runs[0]
	if rec.Status != runstore.RunFailed || rec.FailureReason != "stop_condition:forbidden_path_modified" {
		t.Fatalf("RunRecord = %+v", rec)
	}
	if rec.ChangeSetID != "" || out.ChangeSet != nil || f.fixCalls != 0 {
		t.Fatalf("停止的 Run 不得固定候选: id=%s fix=%d", rec.ChangeSetID, f.fixCalls)
	}
}

func TestUnregisteredStopConditionIsConfigurationError(t *testing.T) {
	f := newFixture(t, func(spec *TaskExecutionSpec, _ *execFixture) {
		spec.Hooks.StopProbe = func(_ context.Context, _ IterationView) ([]policy.StopCondition, error) {
			return []policy.StopCondition{"not_in_policy"}, nil
		}
	})

	_, err := f.exec.Execute(context.Background())
	if !errors.Is(err, ErrUnregisteredStopCondition) {
		t.Fatalf("err = %v, want ErrUnregisteredStopCondition", err)
	}
	// 治理优先:Run 仍以失败终态落库,不残留 running。
	if fin, ok := f.sink.finishes["run-0001"]; ok && fin.Status == runstore.RunRunning {
		t.Fatal("Run 不得停留在 running")
	}
}

// ---- AC4:无 DAG/并行调度/通用编排;一次性执行 ----

func TestExecuteIsOneShot(t *testing.T) {
	f := newFixture(t, nil)
	if _, err := f.exec.Execute(context.Background()); err != nil {
		t.Fatalf("第一次 Execute: %v", err)
	}
	if _, err := f.exec.Execute(context.Background()); !errors.Is(err, ErrAlreadyExecuted) {
		t.Fatalf("第二次 Execute err = %v, want ErrAlreadyExecuted", err)
	}
}

// ---- 构造校验 ----

func TestNewTaskExecutionRejectsInvalidSpecs(t *testing.T) {
	base := func() TaskExecutionSpec {
		f := newFixture(t, nil)
		// 从已通过的固件反提 spec 的最小合法形;逐字段破坏。
		return TaskExecutionSpec{
			Project:              "proj-39",
			Task:                 testContract(),
			Role:                 "coding",
			Policy:               testPolicy(),
			BaseRevision:         f.base,
			Profile:              "default-coding",
			Runtime:              fake.New(fake.Config{}),
			Model:                "m",
			BehaviorRevision:     "b",
			VerificationRevision: "v",
			ContextPackHash:      "c",
			Worktrees:            workspace.NewRunTable(mustManager(t, f.repo)),
			Sink:                 newMemSink(),
			Hooks: Hooks{FixCandidate: func(context.Context, IterationView) (CandidateSnapshot, bool, error) {
				return CandidateSnapshot{}, false, nil
			}},
		}
	}

	mutants := map[string]func(s *TaskExecutionSpec){
		"no project":               func(s *TaskExecutionSpec) { s.Project = "" },
		"no task":                  func(s *TaskExecutionSpec) { s.Task.ID = " " },
		"no role":                  func(s *TaskExecutionSpec) { s.Role = "" },
		"no base":                  func(s *TaskExecutionSpec) { s.BaseRevision = "" },
		"no profile":               func(s *TaskExecutionSpec) { s.Profile = "" },
		"no runtime":               func(s *TaskExecutionSpec) { s.Runtime = nil },
		"no model":                 func(s *TaskExecutionSpec) { s.Model = "" },
		"no behavior":              func(s *TaskExecutionSpec) { s.BehaviorRevision = " " },
		"no verification revision": func(s *TaskExecutionSpec) { s.VerificationRevision = "" },
		"no context pack hash":     func(s *TaskExecutionSpec) { s.ContextPackHash = "" },
		"no worktrees":             func(s *TaskExecutionSpec) { s.Worktrees = nil },
		"no sink":                  func(s *TaskExecutionSpec) { s.Sink = nil },
		"no fixer":                 func(s *TaskExecutionSpec) { s.Hooks.FixCandidate = nil },
		"invalid policy":           func(s *TaskExecutionSpec) { s.Policy.Retry.MaxIterations = 0 },
		"invalid task":             func(s *TaskExecutionSpec) { s.Task.Profile = "bogus" },
	}
	for name, mutate := range mutants {
		t.Run(name, func(t *testing.T) {
			spec := base()
			mutate(&spec)
			if _, err := NewTaskExecution(spec); !errors.Is(err, ErrInvalidSpec) {
				t.Fatalf("err = %v, want ErrInvalidSpec", err)
			}
		})
	}
}

func TestSinkFailurePropagatesAsError(t *testing.T) {
	for _, stage := range []string{"begin", "finish", "changeset", "progress"} {
		t.Run(stage, func(t *testing.T) {
			f := newFixture(t, func(spec *TaskExecutionSpec, f *execFixture) {
				f.sink.failOn = stage
			})
			if _, err := f.exec.Execute(context.Background()); err == nil {
				t.Fatalf("sink %s 失败应作为错误返回", stage)
			}
		})
	}
}

// ---- P0-01:期望 model 档位下发 + 实际身份观察 ---------------------------

func TestStartRequestCarriesTargetModelProfile(t *testing.T) {
	// AC1/AC4:声明 model.select 的 runtime 收到的 StartRequest 必须携带目标
	// model 档位,且与落库的期望 Model 完全一致 —— 记录值与请求值不得静默分叉。
	rec := &recordingRuntime{inner: fake.New(fake.Config{}), injectModelSelect: true}
	f := newFixture(t, func(spec *TaskExecutionSpec, _ *execFixture) { spec.Runtime = rec })

	out, err := f.exec.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Status != TaskCompleted || len(out.Runs) != 1 {
		t.Fatalf("Status/Runs = %s/%d, want completed/1", out.Status, len(out.Runs))
	}
	if len(rec.starts) != 1 {
		t.Fatalf("Start 调用数 = %d, want 1", len(rec.starts))
	}
	if got := rec.starts[0].ModelProfile; got != "test-model" {
		t.Fatalf("StartRequest.ModelProfile = %q, want test-model(目标档位必须下发)", got)
	}
	if rec.starts[0].Workspace == "" {
		t.Fatal("StartRequest.Workspace 不得因下发档位而丢失")
	}
	if b := f.sink.begins["run-0001"]; b.Model != rec.starts[0].ModelProfile {
		t.Fatalf("落库期望 Model %q 与 StartRequest 模型 %q 静默分叉", b.Model, rec.starts[0].ModelProfile)
	}
}

func TestEscalatedRunCarriesEscalatedModelProfile(t *testing.T) {
	// AC4(升级链):升级 Run 以 EscalationTarget.Model 开会话,且同样与该 Run
	// 落库的期望 Model 一致 —— 升级换模型不得只换记录、不换请求。
	weak := &recordingRuntime{
		inner:             fake.New(fake.Config{DefaultTurn: fake.TurnSpec{Behavior: fake.TurnFail}}),
		injectModelSelect: true,
	}
	strong := &recordingRuntime{inner: fake.New(fake.Config{}), injectModelSelect: true}
	f := newFixture(t, func(spec *TaskExecutionSpec, f *execFixture) {
		spec.Runtime = weak
		var calls [][]string
		spec.Hooks.ResolveEscalation = escalateTo(f, "strong-coding", strong, &calls)
		spec.Hooks.FixCandidate = func(_ context.Context, v IterationView) (CandidateSnapshot, bool, error) {
			if v.Profile != "strong-coding" {
				return CandidateSnapshot{}, false, nil // 弱 runtime 的迭代无可固定候选
			}
			return CandidateSnapshot{CandidateRevision: "cand-" + string(v.Run)}, true, nil
		}
	})

	out, err := f.exec.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Status != TaskCompleted || len(out.Runs) != 2 {
		t.Fatalf("Status/Runs = %s/%d, want completed/2", out.Status, len(out.Runs))
	}
	if len(weak.starts) != 1 || weak.starts[0].ModelProfile != "test-model" {
		t.Fatalf("首个 Run 的 StartRequest = %+v, want ModelProfile test-model", weak.starts)
	}
	if len(strong.starts) != 1 || strong.starts[0].ModelProfile != "escalated-model" {
		t.Fatalf("升级 Run 的 StartRequest = %+v, want ModelProfile escalated-model", strong.starts)
	}
	if b := f.sink.begins[out.Runs[1].ID]; b.Model != strong.starts[0].ModelProfile {
		t.Fatalf("升级 Run 落库 Model %q 与其 StartRequest 模型 %q 静默分叉", b.Model, strong.starts[0].ModelProfile)
	}
}

func TestRuntimeWithoutModelSelectStartsWithoutProfile(t *testing.T) {
	// AC3/能力安全降级:未声明 model.select 的 runtime(FakeRuntime 即如此)
	// 照常开跑 —— 请求不带 ModelProfile(带上会被 CheckStart 的能力协商拒绝),
	// 期望 Model 仍如实落库,实际身份保持未观察零值;降级可见,不伪造。
	rec := &recordingRuntime{inner: fake.New(fake.Config{})}
	f := newFixture(t, func(spec *TaskExecutionSpec, _ *execFixture) { spec.Runtime = rec })

	out, err := f.exec.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Status != TaskCompleted || len(out.Runs) != 1 {
		t.Fatalf("Status/Runs = %s/%d, want completed/1", out.Status, len(out.Runs))
	}
	if len(rec.starts) != 1 || rec.starts[0].ModelProfile != "" {
		t.Fatalf("未声明 model.select 时 StartRequest = %+v, want 不携带 ModelProfile", rec.starts)
	}
	b := f.sink.begins["run-0001"]
	if b.Model != "test-model" {
		t.Fatalf("期望 Model = %q, want test-model(降级不抹掉期望记录)", b.Model)
	}
	if b.ObservedIdentity.Observed() {
		t.Fatalf("会话未实现 IdentityReporter,观察身份必须为零值: %+v", b.ObservedIdentity)
	}
}

func TestObservedIdentityCarriedSeparatelyFromDesired(t *testing.T) {
	// Decision Lock 15:实际身份与期望配置分开携带 —— 观察到的模型可以与
	// 请求/落库的期望模型不同,两者并存且互不覆盖。
	observed := agentruntime.ObservedIdentity{
		RuntimeName:    "observed-cli",
		RuntimeVersion: "9.9.9",
		Model:          "actual-observed-model",
		SessionID:      "sess-observed-1",
	}
	rec := &recordingRuntime{inner: fake.New(fake.Config{}), reportIdentity: true, identity: observed}
	f := newFixture(t, func(spec *TaskExecutionSpec, _ *execFixture) { spec.Runtime = rec })

	out, err := f.exec.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Status != TaskCompleted {
		t.Fatalf("Status = %s, want completed", out.Status)
	}
	b := f.sink.begins["run-0001"]
	if b.ObservedIdentity != observed {
		t.Fatalf("RunBegin.ObservedIdentity = %+v, want 原样携带观察值 %+v", b.ObservedIdentity, observed)
	}
	if b.Model != "test-model" {
		t.Fatalf("期望 Model = %q, 不得被观察值覆盖", b.Model)
	}
	if rec.starts[0].ModelProfile != "" {
		t.Fatalf("未声明 model.select 时不得把期望档位当请求下发: %+v", rec.starts[0])
	}
}

// ---- 数据库层:RunStoreSink round trip(P1-33 适配;库不可达即 SKIP) ----

func TestRunStoreSinkRoundTrip(t *testing.T) {
	runs, pool := testWorkflowStore(t)
	_ = pool

	weak := fake.New(fake.Config{DefaultTurn: fake.TurnSpec{Behavior: fake.TurnFail}})
	// 升级 Run 的会话自报实际身份:观察到的 AgentSessionID 必须经
	// RunStoreSink 落到 codegen_run.agent_session_id(Decision Lock 15)。
	observed := agentruntime.ObservedIdentity{
		RuntimeName:    "fake",
		RuntimeVersion: "0.0.0-test",
		Model:          "escalated-model",
		SessionID:      "sess-strong-1",
	}
	strong := &recordingRuntime{inner: fake.New(fake.Config{}), reportIdentity: true, identity: observed}
	f := newFixture(t, func(spec *TaskExecutionSpec, f *execFixture) {
		spec.Project = "proj-db-39"
		spec.Runtime = weak
		spec.Sink = NewRunStoreSink(runs)
		var calls [][]string
		spec.Hooks.ResolveEscalation = escalateTo(f, "strong-coding", strong, &calls)
	})

	out, err := f.exec.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Status != TaskCompleted || len(out.Runs) != 2 {
		t.Fatalf("Status/Runs = %s/%d, want completed/2", out.Status, len(out.Runs))
	}

	timeline, err := runs.CodegenRunsForTask(context.Background(), "proj-db-39", "TASK-WF39")
	if err != nil {
		t.Fatalf("CodegenRunsForTask: %v", err)
	}
	if len(timeline) != 2 {
		t.Fatalf("落库 Run 数 = %d, want 2", len(timeline))
	}
	first, second := timeline[0], timeline[1]
	if first.Status != runstore.RunEscalated || second.Status != runstore.RunCompleted {
		t.Fatalf("落库状态 = %s/%s, want escalated/completed", first.Status, second.Status)
	}
	if second.EscalatedFrom != first.ID {
		t.Fatalf("落库 escalated_from = %d, want %d", second.EscalatedFrom, first.ID)
	}
	// P0-01:观察到的会话身份落库;未自报的 Run(weak 未实现 IdentityReporter)
	// 保持空,不得回填。
	if second.AgentSessionID != "sess-strong-1" {
		t.Fatalf("落库 agent_session_id = %q, want sess-strong-1", second.AgentSessionID)
	}
	if first.AgentSessionID != "" {
		t.Fatalf("未自报身份的 Run 落库 agent_session_id = %q, want 空", first.AgentSessionID)
	}
	if second.ResultPatchHash == "" || first.ResultPatchHash != "" {
		t.Fatalf("ResultPatchHash = %q/%q, want 仅完成的 Run 非空", first.ResultPatchHash, second.ResultPatchHash)
	}

	changeSets, err := runs.ChangeSetsForTask(context.Background(), "proj-db-39", "TASK-WF39")
	if err != nil {
		t.Fatalf("ChangeSetsForTask: %v", err)
	}
	if len(changeSets) != 1 {
		t.Fatalf("落库 ChangeSet 数 = %d, want 1", len(changeSets))
	}
	if changeSets[0].ID() != out.ChangeSet.ID() || changeSets[0].ProducerRun() != strconv.FormatInt(second.ID, 10) {
		t.Fatalf("落库候选与结果不符: %s/%s", changeSets[0].ID(), changeSets[0].ProducerRun())
	}
}

// gateRuntime 在首个迭代开始时关闭 started,再交给内层 runtime;供测试在
// 确定的执行点取消调用方 context(替代任意 sleep)。
type gateRuntime struct {
	inner   agentruntime.Runtime
	started chan struct{}
	once    sync.Once
}

func (g *gateRuntime) Descriptor() agentruntime.RuntimeDescriptor { return g.inner.Descriptor() }

func (g *gateRuntime) Start(ctx context.Context, req agentruntime.StartRequest) (agentruntime.Session, error) {
	s, err := g.inner.Start(ctx, req)
	if err != nil {
		return nil, err
	}
	return &gateSession{inner: s, g: g}, nil
}

type gateSession struct {
	inner agentruntime.Session
	g     *gateRuntime
}

func (s *gateSession) Execute(ctx context.Context, req agentruntime.ExecuteRequest) (agentruntime.AgentRunResult, error) {
	s.g.once.Do(func() { close(s.g.started) })
	return s.inner.Execute(ctx, req)
}

func (s *gateSession) Cancel(ctx context.Context, id agentruntime.RunID) error {
	return s.inner.Cancel(ctx, id)
}

func (s *gateSession) Events(ctx context.Context) (agentruntime.EventStream, error) {
	return s.inner.Events(ctx)
}

func (s *gateSession) Result(ctx context.Context, id agentruntime.RunID) (agentruntime.AgentRunResult, error) {
	return s.inner.Result(ctx, id)
}

func (s *gateSession) Close(ctx context.Context) error { return s.inner.Close(ctx) }

// recordingRuntime 记录每次 Start 收到的 StartRequest(P0-01 测试用):
// injectModelSelect 时在描述符中虚拟声明 model.select —— 内层 runtime 仍按
// 其自身能力约束运行,故把 ModelProfile 剥掉后再委托;reportIdentity 时其
// 会话额外实现 agentruntime.IdentityReporter 并上报 identity。
type recordingRuntime struct {
	inner             agentruntime.Runtime
	injectModelSelect bool
	reportIdentity    bool
	identity          agentruntime.ObservedIdentity

	mu     sync.Mutex
	starts []agentruntime.StartRequest
}

func (r *recordingRuntime) Descriptor() agentruntime.RuntimeDescriptor {
	d := r.inner.Descriptor()
	if r.injectModelSelect {
		d.Capabilities = agentruntime.NewCapabilitySet(
			append(d.Capabilities.List(), agentruntime.CapModelSelect)...)
	}
	return d
}

func (r *recordingRuntime) Start(ctx context.Context, req agentruntime.StartRequest) (agentruntime.Session, error) {
	r.mu.Lock()
	r.starts = append(r.starts, req)
	r.mu.Unlock()
	delegated := req
	if r.injectModelSelect {
		delegated.ModelProfile = "" // 内层未声明 model.select,剥掉后按其自身能力约束委托
	}
	s, err := r.inner.Start(ctx, delegated)
	if err != nil {
		return nil, err
	}
	if r.reportIdentity {
		return &recordingSession{Session: s, identity: r.identity}, nil
	}
	return s, nil
}

// recordingSession 内嵌核心 Session,可选自报固定观察身份。
type recordingSession struct {
	agentruntime.Session
	identity agentruntime.ObservedIdentity
}

func (s *recordingSession) Identity() agentruntime.ObservedIdentity { return s.identity }

// ---- git/worktree 固件(与 internal/workspace 测试同一隔离策略) ----

func gitTestEnv() []string {
	return []string{
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=" + os.DevNull,
		"GIT_CONFIG_SYSTEM=" + os.DevNull,
		"GIT_TERMINAL_PROMPT=0",
	}
}

func runGit(t *testing.T, dir string, argv ...string) string {
	t.Helper()
	local := runner.LocalRunner{}
	res, err := local.Run(context.Background(), runner.Command{
		Argv: append([]string{"git"}, argv...),
		Dir:  dir,
		Env:  gitTestEnv(),
	})
	if err != nil {
		t.Fatalf("git %v: %v", argv, err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("git %v: exit %d: %s", argv, res.ExitCode, res.Stderr)
	}
	return strings.TrimSpace(res.Stdout)
}

func newWFRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.name", "tally-test")
	runGit(t, dir, "config", "user.email", "tally-test@example.invalid")
	return dir
}

func wfCommit(t *testing.T, repo, name, content, msg string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repo, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", msg)
	return runGit(t, repo, "rev-parse", "HEAD")
}

func mustManager(t *testing.T, repo string) *workspace.Manager {
	t.Helper()
	m, err := workspace.New(repo, filepath.Join(t.TempDir(), "wt"))
	if err != nil {
		t.Fatalf("workspace.New: %v", err)
	}
	return m
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ---- 测试库:每测试独立 schema,自跑迁移,cleanup DROP;不可达即 SKIP ----

const workflowTestDSN = "postgres://postgres:postgres@localhost:5432/tally?sslmode=disable"

func testWorkflowStore(t *testing.T) (*runstore.Store, *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	base := os.Getenv("DATABASE_URL")
	if base == "" {
		base = workflowTestDSN
	}
	if !dbReachable(base) {
		t.Skipf("workflow 数据库层测试需要 PostgreSQL(%s 不可达),跳过", base)
	}
	schema := "tally_wf_" + strconv.FormatInt(time.Now().UnixNano(), 10)

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
		t.Fatalf("在测试 schema %s 跑迁移: %v", schema, err)
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
	t.Cleanup(func() {
		pool.Close()
		dropSchema(base, schema)
	})
	return runstore.New(store.New(pool)), pool
}

func dbReachable(base string) bool {
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

func dropSchema(base, schema string) {
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
