package verification

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"tally/internal/policy"
	"tally/internal/runner"
)

// ---- 测试基础设施 ---------------------------------------------------------

// recordedCall 记录执行器真正下发给 runner 的一次命令与调用时的 ctx deadline。
type recordedCall struct {
	Argv     []string
	Dir      string
	Env      []string
	Deadline time.Duration // 调用时刻距 ctx deadline 的剩余时间;无 deadline 为 0
}

// fakeRunner 是 runner.Runner 的确定性假实现:不启动任何进程,按调用序返回
// 预设结果;results/errs 耗尽后重复最后一项。
type fakeRunner struct {
	results []runner.Result
	errs    []error
	calls   []recordedCall
}

func (f *fakeRunner) Run(ctx context.Context, cmd runner.Command) (runner.Result, error) {
	i := len(f.calls)
	rec := recordedCall{Argv: append([]string(nil), cmd.Argv...), Dir: cmd.Dir, Env: append([]string(nil), cmd.Env...)}
	if dl, ok := ctx.Deadline(); ok {
		rec.Deadline = time.Until(dl)
	}
	f.calls = append(f.calls, rec)
	if i < len(f.errs) && f.errs[i] != nil {
		return runner.Result{}, f.errs[i]
	}
	if len(f.results) == 0 {
		return runner.Result{}, nil
	}
	res := f.results[min(i, len(f.results)-1)]
	return res, nil
}

// execTestContract 返回一个含 visible(protected)+ held-out binding 的冻结契约:
//   - AC-1/b-vis:visible、protected(immutable)、带冻结结构化命令;
//   - AC-2/b-hold:held_out,store/ref 只经 final executor(P1-29)可达。
func execTestContract(t *testing.T) *Frozen {
	t.Helper()
	c := &Contract{
		TaskID:   "TASK-P1-28",
		Revision: 3,
		Acceptances: []Acceptance{
			{
				ID:        "AC-1",
				Statement: "visible unit tests pass during iteration",
				EvidenceBindings: []Binding{{
					ID:               "b-vis",
					AuthoringMode:    AuthoringPreCommit,
					VisibilityPolicy: VisibilityVisible,
					Evidence:         EvidenceRef{Type: EvidenceTest, Path: "example", Selector: "TestExample"},
					Protection:       &Protection{Mode: ProtectionImmutable, BaselineHash: "sha256:" + strings.Repeat("a", 64)},
					ExecutionProtocol: &ExecutionProtocol{
						Repetitions: 1,
						Aggregation: AggregationAllPass,
						Command: &CommandSpec{
							Argv:    []string{"go", "test", "./example/..."},
							Dir:     "example",
							Env:     []string{"GOFLAGS=-count=1"},
							Timeout: Duration(90 * time.Second),
						},
					},
				}},
			},
			{
				ID:        "AC-2",
				Statement: "held-out scenario only runs at final verification",
				EvidenceBindings: []Binding{{
					ID:               "b-hold",
					AuthoringMode:    AuthoringPreCommit,
					VisibilityPolicy: VisibilityHeldOut,
					Evidence:         EvidenceRef{Type: EvidenceTest, Store: "holdout-store", Ref: "ho-secret-ref-0001"},
				}},
			},
		},
		RuntimeRequirements: RuntimeRequirements{SessionIsolation: true, HeldOutUnreachable: true, ConformanceClass: "verification-isolation-v1"},
	}
	f, err := c.Freeze()
	if err != nil {
		t.Fatalf("Freeze: %v", err)
	}
	return f
}

func mustExecutor(t *testing.T, f *fakeRunner) *VisibleExecutor {
	t.Helper()
	x, err := NewVisibleExecutor(f, policy.DefaultExecutionPolicy())
	if err != nil {
		t.Fatalf("NewVisibleExecutor: %v", err)
	}
	return x
}

const leakRef = "ho-secret-ref-0001"
const leakStore = "holdout-store"

// assertNoHeldOutLeak 断言序列化后的 Agent 可见输出不含任何 held-out 可识别信息
// (AC4)。
func assertNoHeldOutLeak(t *testing.T, data []byte) {
	t.Helper()
	s := string(data)
	for _, leak := range []string{"b-hold", leakRef, leakStore} {
		if strings.Contains(s, leak) {
			t.Fatalf("Agent 可见输出泄露 held-out 信息 %q: %s", leak, s)
		}
	}
}

// ---- AC1:只有 visible binding 被执行 --------------------------------------

func TestVisibleExecutorRunsOnlyVisibleBindings(t *testing.T) {
	fr := &fakeRunner{results: []runner.Result{{Stdout: "ok 0.1s", ExitCode: 0, Duration: 100 * time.Millisecond}}}
	x := mustExecutor(t, fr)
	f := execTestContract(t)

	rep, err := x.Run(context.Background(), IncrementalRequest{Contract: f, Workspace: t.TempDir()})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// 只有 visible binding 执行;held-out 只以计数出现。
	if len(fr.calls) != 1 {
		t.Fatalf("runner 调用次数 = %d, want 1(只有 visible binding 执行)", len(fr.calls))
	}
	if rep.SkippedHeldOut != 1 {
		t.Fatalf("SkippedHeldOut = %d, want 1", rep.SkippedHeldOut)
	}
	if len(rep.Results) != 1 || rep.Results[0].BindingID != "b-vis" || rep.Results[0].Status != VisibleStatusPass {
		t.Fatalf("Results = %+v, want 单条 b-vis pass", rep.Results)
	}
	if rep.TaskID != "TASK-P1-28" || rep.ContractRevision != 3 {
		t.Fatalf("report 头部错误: task=%s rev=%d", rep.TaskID, rep.ContractRevision)
	}
	if rep.ContractHash != f.Hash() {
		t.Fatalf("report ContractHash = %q, want 冻结哈希 %q", rep.ContractHash, f.Hash())
	}

	// AC4:报告整体(JSON 投影)不含 held-out id/store/ref。
	data, err := json.Marshal(rep)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	assertNoHeldOutLeak(t, data)
}

// ---- AC1/AC4:focus 选择与拒绝 ---------------------------------------------

func TestVisibleExecutorFocusSelection(t *testing.T) {
	t.Run("focus 只执行点名的 visible binding", func(t *testing.T) {
		fr := &fakeRunner{results: []runner.Result{{ExitCode: 0}}}
		x := mustExecutor(t, fr)
		f := execTestContract(t)

		rep, err := x.Run(context.Background(), IncrementalRequest{Contract: f, Workspace: t.TempDir(), Focus: []string{"b-vis"}})
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		if len(rep.Results) != 1 || rep.Results[0].BindingID != "b-vis" {
			t.Fatalf("Results = %+v, want 只含 b-vis", rep.Results)
		}
		if rep.SkippedHeldOut != 0 {
			t.Fatalf("focus 模式 SkippedHeldOut = %d, want 0(未点名的不属于本轮)", rep.SkippedHeldOut)
		}
	})

	t.Run("focus 点名 held-out binding 被整体拒绝且不泄露 id", func(t *testing.T) {
		fr := &fakeRunner{}
		x := mustExecutor(t, fr)
		f := execTestContract(t)

		rep, err := x.Run(context.Background(), IncrementalRequest{Contract: f, Workspace: t.TempDir(), Focus: []string{"b-vis", "b-hold"}})
		if !errors.Is(err, ErrHeldOutNotIncremental) {
			t.Fatalf("err = %v, want ErrHeldOutNotIncremental", err)
		}
		if rep.Results != nil || len(fr.calls) != 0 {
			t.Fatalf("选择错误时不得执行任何命令: calls=%d results=%+v", len(fr.calls), rep.Results)
		}
		assertNoHeldOutLeak(t, []byte(err.Error()))
	})

	t.Run("focus 点名未知 binding 报 ErrUnknownBinding", func(t *testing.T) {
		fr := &fakeRunner{}
		x := mustExecutor(t, fr)
		f := execTestContract(t)

		_, err := x.Run(context.Background(), IncrementalRequest{Contract: f, Workspace: t.TempDir(), Focus: []string{"no-such-binding"}})
		if !errors.Is(err, ErrUnknownBinding) || !strings.Contains(err.Error(), "no-such-binding") {
			t.Fatalf("err = %v, want ErrUnknownBinding 携带未知 id", err)
		}
		if len(fr.calls) != 0 {
			t.Fatalf("选择错误时不得执行任何命令")
		}
	})
}

// ---- AC2:命令遵守 ExecutionPolicy 与结构化 CommandSpec ---------------------

func TestVisibleExecutorStructuredCommandAndWorkspace(t *testing.T) {
	ws := t.TempDir()
	fr := &fakeRunner{results: []runner.Result{{ExitCode: 0}}}
	x := mustExecutor(t, fr)
	f := execTestContract(t)

	rep, err := x.Run(context.Background(), IncrementalRequest{Contract: f, Workspace: ws})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.Results[0].Status != VisibleStatusPass {
		t.Fatalf("status = %q (%s), want pass", rep.Results[0].Status, rep.Results[0].Err)
	}

	// 结构化 argv 原样下发(无 shell 包装),相对 cwd 解析在 workspace 内,
	// env 为追加的 KEY=VALUE。
	got := fr.calls[0]
	wantDir := filepath.Join(ws, "example")
	if !reflect.DeepEqual(got.Argv, []string{"go", "test", "./example/..."}) {
		t.Fatalf("Argv = %v, want 结构化 argv 原样", got.Argv)
	}
	if got.Dir != wantDir {
		t.Fatalf("Dir = %q, want %q", got.Dir, wantDir)
	}
	if !reflect.DeepEqual(got.Env, []string{"GOFLAGS=-count=1"}) {
		t.Fatalf("Env = %v, want 追加 env 原样", got.Env)
	}
}

func TestVisibleExecutorRejectsUnsafeCommandSpecs(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*CommandSpec)
		wantErr string
	}{
		{
			name:    "cwd 逃出 workspace",
			mutate:  func(s *CommandSpec) { s.Dir = "../outside" },
			wantErr: "逃出 candidate workspace",
		},
		{
			name:    "cwd 绝对路径",
			mutate:  func(s *CommandSpec) { s.Dir = t.TempDir() },
			wantErr: "workspace 内相对路径",
		},
		{
			name:    "env 缺少 KEY=VALUE",
			mutate:  func(s *CommandSpec) { s.Env = []string{"NOVALUE"} },
			wantErr: "KEY=VALUE",
		},
		{
			name:    "env KEY 重复",
			mutate:  func(s *CommandSpec) { s.Env = []string{"A=1", "A=2"} },
			wantErr: "KEY 重复",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fr := &fakeRunner{results: []runner.Result{{ExitCode: 0}}}
			x := mustExecutor(t, fr)
			c := &Contract{
				TaskID:   "TASK-P1-28",
				Revision: 1,
				Acceptances: []Acceptance{{
					ID:        "AC-1",
					Statement: "s",
					EvidenceBindings: []Binding{{
						ID:               "b-bad",
						AuthoringMode:    AuthoringPreCommit,
						VisibilityPolicy: VisibilityVisible,
						Evidence:         EvidenceRef{Type: EvidenceTest, Path: "p"},
						ExecutionProtocol: &ExecutionProtocol{
							Repetitions: 1, Aggregation: AggregationAllPass,
							Command: func() *CommandSpec {
								s := &CommandSpec{Argv: []string{"go", "test"}, Timeout: Duration(time.Second)}
								tt.mutate(s)
								return s
							}(),
						},
					}},
				}},
			}
			f, err := c.Freeze()
			if err != nil {
				t.Fatalf("Freeze: %v", err)
			}

			rep, err := x.Run(context.Background(), IncrementalRequest{Contract: f, Workspace: t.TempDir()})
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			res := rep.Results[0]
			if res.Status != VisibleStatusError || !strings.Contains(res.Err, tt.wantErr) {
				t.Fatalf("result = %q/%q, want error 含 %q", res.Status, res.Err, tt.wantErr)
			}
			if len(fr.calls) != 0 {
				t.Fatalf("非法命令不得下发给 runner")
			}
			// error 不是 verdict,不得映射为证据事实。
			if _, err := res.EvidenceRecord(f.Revision()); err == nil {
				t.Fatalf("error 结果不得映射为 EvidenceRecord")
			}
		})
	}
}

// 空 argv 的防御层:Freeze 会在契约侧拒绝它(types.go),但执行器对冻结内容
// 仍保留二次校验 —— 白盒直接构造 Frozen 绕过 Freeze,验证执行层兜底。
func TestVisibleExecutorRejectsEmptyArgvFrozenDirectly(t *testing.T) {
	fr := &fakeRunner{}
	x := mustExecutor(t, fr)
	c := &Contract{
		TaskID:   "TASK-P1-28",
		Revision: 1,
		Acceptances: []Acceptance{{
			ID:        "AC-1",
			Statement: "s",
			EvidenceBindings: []Binding{{
				ID: "b-empty", AuthoringMode: AuthoringPreCommit, VisibilityPolicy: VisibilityVisible,
				Evidence: EvidenceRef{Type: EvidenceTest, Path: "p"},
				ExecutionProtocol: &ExecutionProtocol{
					Repetitions: 1, Aggregation: AggregationAllPass,
					Command: &CommandSpec{Argv: []string{""}},
				},
			}},
		}},
	}
	// 绕过 Freeze 的直接构造(仅白盒测试允许);Run 只消费 TaskID/Revision/Hash
	// 与 binding,哈希用占位形态值即可。
	f := &Frozen{c: *c, hash: "sha256:" + strings.Repeat("0", 64)}

	rep, err := x.Run(context.Background(), IncrementalRequest{Contract: f, Workspace: t.TempDir()})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.Results[0].Status != VisibleStatusError || !strings.Contains(rep.Results[0].Err, "argv") {
		t.Fatalf("result = %q/%q, want argv error", rep.Results[0].Status, rep.Results[0].Err)
	}
	if len(fr.calls) != 0 {
		t.Fatalf("空 argv 不得下发给 runner")
	}
}

func TestVisibleExecutorHonorsPolicyBudgetDeadline(t *testing.T) {
	tests := []struct {
		name    string
		budget  time.Duration
		spec    time.Duration // binding 声明的命令超时;0 = 未声明
		wantMin time.Duration // 调用时刻 ctx 剩余 deadline 的合法下限(留误差)
		wantMax time.Duration // 合法上限
	}{
		{name: "binding 超时小于预算", budget: 30 * time.Minute, spec: 90 * time.Second, wantMin: 80 * time.Second, wantMax: 91 * time.Second},
		{name: "binding 超时大于预算被截断", budget: 30 * time.Minute, spec: 2 * time.Hour, wantMin: 29 * time.Minute, wantMax: 30 * time.Minute},
		{name: "binding 未声明超时用剩余预算", budget: 45 * time.Minute, spec: 0, wantMin: 44 * time.Minute, wantMax: 45 * time.Minute},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fr := &fakeRunner{results: []runner.Result{{ExitCode: 0}}}
			p := policy.DefaultExecutionPolicy()
			p.Budgets.MaxWallTime = tt.budget
			x, err := NewVisibleExecutor(fr, p)
			if err != nil {
				t.Fatalf("NewVisibleExecutor: %v", err)
			}
			c := &Contract{
				TaskID:   "TASK-P1-28",
				Revision: 1,
				Acceptances: []Acceptance{{
					ID:        "AC-1",
					Statement: "s",
					EvidenceBindings: []Binding{{
						ID:               "b-t",
						AuthoringMode:    AuthoringPreCommit,
						VisibilityPolicy: VisibilityVisible,
						Evidence:         EvidenceRef{Type: EvidenceTest, Path: "p"},
						ExecutionProtocol: &ExecutionProtocol{
							Repetitions: 1, Aggregation: AggregationAllPass,
							Command: &CommandSpec{Argv: []string{"true"}, Timeout: Duration(tt.spec)},
						},
					}},
				}},
			}
			f, err := c.Freeze()
			if err != nil {
				t.Fatalf("Freeze: %v", err)
			}

			if _, err := x.Run(context.Background(), IncrementalRequest{Contract: f, Workspace: t.TempDir()}); err != nil {
				t.Fatalf("Run: %v", err)
			}
			dl := fr.calls[0].Deadline
			if dl <= 0 || dl < tt.wantMin || dl > tt.wantMax {
				t.Fatalf("ctx 剩余 deadline = %v, want (%v, %v]", dl, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestVisibleExecutorBudgetExhaustedStopsFurtherRuns(t *testing.T) {
	fr := &fakeRunner{results: []runner.Result{{ExitCode: 0, Duration: 30 * time.Minute}}}
	p := policy.DefaultExecutionPolicy()
	p.Budgets.MaxWallTime = 30 * time.Minute
	x, err := NewVisibleExecutor(fr, p)
	if err != nil {
		t.Fatalf("NewVisibleExecutor: %v", err)
	}

	// 两条 visible binding;第一条耗尽全部预算,第二条不得再执行。
	c := &Contract{
		TaskID:   "TASK-P1-28",
		Revision: 1,
		Acceptances: []Acceptance{
			{ID: "AC-1", Statement: "s1", EvidenceBindings: []Binding{visBinding("b-1")}},
			{ID: "AC-2", Statement: "s2", EvidenceBindings: []Binding{visBinding("b-2")}},
		},
	}
	f, err := c.Freeze()
	if err != nil {
		t.Fatalf("Freeze: %v", err)
	}

	rep, err := x.Run(context.Background(), IncrementalRequest{Contract: f, Workspace: t.TempDir()})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(fr.calls) != 1 {
		t.Fatalf("runner 调用次数 = %d, want 1(预算用尽后不再执行)", len(fr.calls))
	}
	if !rep.BudgetExhausted {
		t.Fatalf("BudgetExhausted = false, want true")
	}
	if len(rep.Results) != 2 {
		t.Fatalf("Results = %d 条, want 2(第二条以 error 记账)", len(rep.Results))
	}
	second := rep.Results[1]
	if second.Status != VisibleStatusError || !strings.Contains(second.Err, ErrBudgetExhausted.Error()) {
		t.Fatalf("第二条 = %q/%q, want 预算用尽 error", second.Status, second.Err)
	}
	// 预算用尽不是验证失败:不得产生 FAIL 事实。
	if rec, err := second.EvidenceRecord(f.Revision()); err == nil {
		t.Fatalf("预算用尽 error 不得映射为 EvidenceRecord: %+v", rec)
	}
}

func visBinding(id string) Binding {
	return Binding{
		ID:               id,
		AuthoringMode:    AuthoringPreCommit,
		VisibilityPolicy: VisibilityVisible,
		Evidence:         EvidenceRef{Type: EvidenceTest, Path: "p"},
		ExecutionProtocol: &ExecutionProtocol{
			Repetitions: 1, Aggregation: AggregationAllPass,
			Command: &CommandSpec{Argv: []string{"true"}},
		},
	}
}

func TestNewVisibleExecutorRejectsInvalidPolicyAndRunner(t *testing.T) {
	if _, err := NewVisibleExecutor(nil, policy.DefaultExecutionPolicy()); err == nil {
		t.Fatalf("nil runner 必须拒绝")
	}
	// 零值策略缺预算、缺硬停止条件、网络模式非法 —— 必须整体拒绝创建。
	if _, err := NewVisibleExecutor(&fakeRunner{}, policy.ExecutionPolicy{}); !errors.Is(err, policy.ErrInvalidPolicy) {
		t.Fatalf("err = %v, want policy.ErrInvalidPolicy", err)
	}
}

// ---- AC3:结果可记录,但不直接设置 Task 状态 --------------------------------

func TestVisibleExecutorResultsAreRecordableNotStateful(t *testing.T) {
	fr := &fakeRunner{results: []runner.Result{{Stdout: "ok", Stderr: "warn", ExitCode: 0, Duration: time.Second}}}
	x := mustExecutor(t, fr)
	f := execTestContract(t)
	hashBefore := f.Hash()

	rep, err := x.Run(context.Background(), IncrementalRequest{Contract: f, Workspace: t.TempDir()})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if f.Hash() != hashBefore {
		t.Fatalf("Run 改变了冻结契约哈希:执行器不得有任何契约副作用")
	}

	// digest 独立于实现计算(不重用包内 helper,避免自证)。
	sum := sha256.Sum256([]byte("ok"))
	wantDigest := "sha256:" + hex.EncodeToString(sum[:])
	if rep.Results[0].StdoutDigest != wantDigest {
		t.Fatalf("StdoutDigest = %q, want %q", rep.Results[0].StdoutDigest, wantDigest)
	}

	// 结果经 EvidenceRecord 映射后进入功能覆盖推导 —— 这是「可记录」的落点;
	// 执行器本身没有任何写 Task/VERIFIED 状态的出口。
	rec, err := rep.Results[0].EvidenceRecord(f.Revision())
	if err != nil {
		t.Fatalf("EvidenceRecord: %v", err)
	}
	if rec.BindingID != "b-vis" || rec.Revision != 3 || rec.Status != EvidenceStatusPass {
		t.Fatalf("record = %+v, want b-vis/rev3/pass", rec)
	}
	cov, err := DeriveFunctionalCoverage(f, []EvidenceRecord{rec})
	if err != nil {
		t.Fatalf("DeriveFunctionalCoverage: %v", err)
	}
	if !cov.Acceptances[0].Covered {
		t.Fatalf("AC-1 应被可见 PASS 覆盖: %+v", cov.Acceptances[0])
	}
	// AC-2(held-out)不被增量结果覆盖:增量验证不运行 held-out,就不产生它的
	// 任何证据 —— 最终覆盖归 final executor(§14.2)。
	if cov.Acceptances[1].Covered || cov.Acceptances[1].Gap != GapNoApplicablePass {
		t.Fatalf("AC-2 不应被增量验证覆盖: %+v", cov.Acceptances[1])
	}
}

func TestVisibleExecutorFailAndErrorStatuses(t *testing.T) {
	t.Run("非零退出码是 FAIL 且可记录", func(t *testing.T) {
		fr := &fakeRunner{results: []runner.Result{{Stderr: "FAIL: TestExample", ExitCode: 3}}}
		x := mustExecutor(t, fr)
		f := execTestContract(t)

		rep, err := x.Run(context.Background(), IncrementalRequest{Contract: f, Workspace: t.TempDir()})
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		res := rep.Results[0]
		if res.Status != VisibleStatusFail || res.ExitCode != 3 {
			t.Fatalf("result = %+v, want fail/exit 3", res)
		}
		rec, err := res.EvidenceRecord(f.Revision())
		if err != nil || rec.Status != EvidenceStatusFail {
			t.Fatalf("FAIL 记录 = %+v/%v, want fail", rec, err)
		}
		// 记录后的 FAIL 阻断覆盖(§8.8.1:失败事实不被增量上下文抹掉)。
		cov, err := DeriveFunctionalCoverage(f, []EvidenceRecord{rec})
		if err != nil {
			t.Fatalf("DeriveFunctionalCoverage: %v", err)
		}
		if cov.Acceptances[0].Covered || cov.Acceptances[0].Gap != GapBlockingFailure {
			t.Fatalf("FAIL 应阻断 AC-1 覆盖: %+v", cov.Acceptances[0])
		}
	})

	t.Run("命令起不来是 error 不是 FAIL", func(t *testing.T) {
		fr := &fakeRunner{errs: []error{fmt.Errorf("执行 %q: 找不到程序", "go")}}
		x := mustExecutor(t, fr)
		f := execTestContract(t)

		rep, err := x.Run(context.Background(), IncrementalRequest{Contract: f, Workspace: t.TempDir()})
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		res := rep.Results[0]
		if res.Status != VisibleStatusError || res.ExitCode != 0 || res.Err == "" {
			t.Fatalf("result = %+v, want error 带原因", res)
		}
		if _, err := res.EvidenceRecord(f.Revision()); err == nil {
			t.Fatalf("error 结果不得映射为 EvidenceRecord")
		}
	})
}

func TestVisibleEvidenceRecordRevisionValidation(t *testing.T) {
	res := VisibleBindingResult{BindingID: "b", Status: VisibleStatusPass}
	if _, err := res.EvidenceRecord(0); err == nil {
		t.Fatalf("revision=0 必须拒绝")
	}
}

// ---- 输入校验与执行顺序 ----------------------------------------------------

func TestVisibleExecutorInputValidation(t *testing.T) {
	fr := &fakeRunner{}
	x := mustExecutor(t, fr)
	f := execTestContract(t)

	if _, err := x.Run(context.Background(), IncrementalRequest{Workspace: t.TempDir()}); err == nil {
		t.Fatalf("nil contract 必须拒绝")
	}
	if _, err := x.Run(context.Background(), IncrementalRequest{Contract: f}); !errors.Is(err, ErrNoWorkspace) {
		t.Fatalf("err = %v, want ErrNoWorkspace", err)
	}
	if _, err := x.Run(context.Background(), IncrementalRequest{Contract: f, Workspace: "relative/path"}); err == nil {
		t.Fatalf("相对路径 workspace 必须拒绝")
	}
	if len(fr.calls) != 0 {
		t.Fatalf("输入非法时不得执行任何命令")
	}
}

func TestVisibleExecutorBindingWithoutCommand(t *testing.T) {
	fr := &fakeRunner{}
	x := mustExecutor(t, fr)
	c := &Contract{
		TaskID:   "TASK-P1-28",
		Revision: 1,
		Acceptances: []Acceptance{{
			ID:        "AC-1",
			Statement: "s",
			EvidenceBindings: []Binding{{
				ID: "b-nocmd", AuthoringMode: AuthoringPreCommit, VisibilityPolicy: VisibilityVisible,
				Evidence: EvidenceRef{Type: EvidenceTest, Path: "p"},
			}},
		}},
	}
	f, err := c.Freeze()
	if err != nil {
		t.Fatalf("Freeze: %v", err)
	}

	rep, err := x.Run(context.Background(), IncrementalRequest{Contract: f, Workspace: t.TempDir()})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	res := rep.Results[0]
	if res.Status != VisibleStatusError || !strings.Contains(res.Err, "没有冻结的可执行命令") {
		t.Fatalf("result = %q/%q, want 缺命令 error", res.Status, res.Err)
	}
	if len(fr.calls) != 0 {
		t.Fatalf("无命令 binding 不得触发 runner")
	}
}

func TestVisibleExecutorPreservesContractOrder(t *testing.T) {
	fr := &fakeRunner{results: []runner.Result{{ExitCode: 0}}}
	x := mustExecutor(t, fr)
	c := &Contract{
		TaskID:   "TASK-P1-28",
		Revision: 1,
		Acceptances: []Acceptance{
			{ID: "AC-1", Statement: "s1", EvidenceBindings: []Binding{visBinding("b-1"), visBinding("b-2")}},
			{ID: "AC-2", Statement: "s2", EvidenceBindings: []Binding{visBinding("b-3")}},
		},
	}
	f, err := c.Freeze()
	if err != nil {
		t.Fatalf("Freeze: %v", err)
	}

	rep, err := x.Run(context.Background(), IncrementalRequest{Contract: f, Workspace: t.TempDir()})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got := make([]string, 0, len(rep.Results))
	for _, r := range rep.Results {
		got = append(got, r.BindingID)
	}
	want := []string{"b-1", "b-2", "b-3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("结果顺序 = %v, want 契约声明顺序 %v", got, want)
	}
}
