//go:build integration

package exec

// T10c 集成测试:诊断执行体接真 Postgres + 真知识库(git 仓库)。
// 覆盖三条链路:知识库精确命中(零 LLM)、缺复现上下文跳 G1/G2 并走完三轮预算、未知指纹报错。

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"tally/internal/airuntime"
	"tally/internal/command"
	"tally/internal/core"
	"tally/internal/failure"
	"tally/internal/jobs"
	"tally/internal/knowledge"
	"tally/internal/store"
)

// noInference 是从未被调用的推理执行器替身:这些测试里任何一轮都不该调到它
// (真被调到会拿到空终态 → 诊断 job 报错,零调用断言也会失败)。
func noInference() *airuntime.FakeExecutor { return &airuntime.FakeExecutor{} }

// gateRunner 是给 failure.Verify 用的脚本执行器:基线全红、应用修复后全绿。
type gateRunner struct {
	seqs map[string][]failure.RunResult
	pos  map[string]int
}

func (r *gateRunner) Run(_ context.Context, spec failure.RunSpec) (failure.RunResult, error) {
	k := "baseline"
	if len(spec.Fix) > 0 {
		k = "fixed"
	}
	key := k + "|" + spec.Target
	seq := r.seqs[key]
	if len(seq) == 0 {
		return failure.RunResult{}, fmt.Errorf("没有脚本: %s", key)
	}
	p := r.pos[k]
	if p >= len(seq) {
		p = len(seq) - 1
	}
	r.pos[k] = p + 1
	return seq[p], nil
}

func reds5() []failure.RunResult {
	out := make([]failure.RunResult, 5)
	for i := range out {
		out[i] = failure.RunResult{Green: false, Output: "make: *** Error 1"}
	}
	return out
}

func greens5() []failure.RunResult {
	out := make([]failure.RunResult, 5)
	for i := range out {
		out[i] = failure.RunResult{Green: true, Output: "ok"}
	}
	return out
}

const (
	diagFP    = core.Fingerprint("a3f9c2e1b4d50607")
	diagFuzzy = core.Fingerprint("f00f00f00f00f00f")
)

func diagCombo() core.VersionCombo { return core.VersionCombo{"go": "1.23.0"} }

// seedBuildRun 记一行失败的 build_run,供 failure.RecordEvent 挂来源外键(事实面只追加)。
func seedBuildRun(t *testing.T, st *store.Store, project string) int64 {
	t.Helper()
	id, err := st.Queries().InsertBuildRun(context.Background(), store.InsertBuildRunParams{
		ProjectID:    project,
		CommitSha:    "0123456789abcdef",
		StartedAt:    pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		Status:       "failed",
		VersionCombo: []byte("{}"),
	})
	if err != nil {
		t.Fatalf("seed build_run: %v", err)
	}
	return id
}

// TestDiagnoseKBHitZeroLLM:该指纹已有验证过的修复(上一轮诊断经四道闸入库、
// 索引面已 SetFix,正是生产里 KB 有条目的唯一来路)→ 新事件诊断时精确命中,
// 零 LLM、执行器一次不跑,索引面的 fixed + fix_ref 原样保留。
func TestDiagnoseKBHitZeroLLM(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	project := "exec-itest-diag-hit"
	combo := diagCombo()

	// —— 事实面先行:一行 build_run + 一条 failure_event(RecordEvent 建好索引行)——
	buildRunID := seedBuildRun(t, st, project)
	if _, err := failure.RecordEvent(ctx, st, failure.Event{
		ProjectID:       project,
		Source:          "build",
		BuildRunID:      buildRunID,
		Excerpt:         "make: *** [build] Error 1",
		Normalized:      "make: *** [build] Error N",
		PipelineVersion: "1",
		Exact:           diagFP,
		Fuzzy:           diagFuzzy,
		Combo:           combo,
		OccurredAt:      time.Now().UTC(),
	}); err != nil {
		t.Fatalf("记 failure_event: %v", err)
	}

	// —— 知识库:真跑四道闸产出证据入库,再像 Handle 验证路径那样 SetFix 索引面 ——
	repo := t.TempDir()
	if _, err := git.PlainInit(repo, false); err != nil {
		t.Fatalf("初始化知识库仓库: %v", err)
	}
	kb := knowledge.NewStore(repo, st.Queries())

	seed := &gateRunner{
		seqs: map[string][]failure.RunResult{"baseline|": reds5(), "fixed|": greens5()},
		pos:  map[string]int{},
	}
	proof, err := failure.Verify(ctx, seed, failure.VerifyInput{
		ReproCommand: command.CommandSpec{Argv: []string{"make", "build"}, Cwd: t.TempDir()},
		Candidate: failure.Candidate{
			Summary: "锁定依赖版本",
			Changes: []core.FileChange{{Path: "go.mod", Content: "module demo\n"}},
		},
		Combo: combo,
	})
	if err != nil {
		t.Fatalf("四道闸应全过: %v", err)
	}
	fix := core.Fix{
		Repro:   "make build",
		Summary: "锁定依赖版本",
		Changes: []core.FileChange{{Path: "go.mod", Content: "module demo\n"}},
	}
	ref, err := kb.Store(ctx, diagFP, fix, proof)
	if err != nil {
		t.Fatalf("已验证修复入库: %v", err)
	}
	if err := failure.NewStatStore(st.Queries()).SetFix(ctx, diagFP, ref,
		proof.VerifiedAt(), proof.VerifiedCombo()); err != nil {
		t.Fatalf("索引面标记 fixed: %v", err)
	}

	// —— 跑诊断 job:精确命中,短路返回 ——
	p := noInference()
	table := Handlers(Config{
		Root:  t.TempDir(),
		Store: st.Queries(),
		DB:    st,
		Diagnose: DiagnoseDeps{
			Executor: p,
			Runner:   seed, // 非 nil 即可;命中短路不会跑到
			KB:       kb,
		},
	})
	if err := table[jobs.KindDiagnose](ctx,
		payload(t, jobs.DiagnosePayload{Fingerprint: string(diagFP), Project: project})); err != nil {
		t.Fatalf("诊断 job 应成功: %v", err)
	}
	if len(p.Requests()) != 0 {
		t.Errorf("知识库精确命中不得调推理: %d 次", len(p.Requests()))
	}

	// 命中路径不碰索引面:上一轮验证写下的 fixed + fix_ref 原样保留。
	var status, fixRef string
	var attempts int
	if err := st.Pool().QueryRow(ctx,
		"SELECT status, fix_ref, diagnose_attempts FROM fingerprint_stat WHERE fp_exact = $1", string(diagFP),
	).Scan(&status, &fixRef, &attempts); err != nil {
		t.Fatalf("读 fingerprint_stat: %v", err)
	}
	if status != "fixed" || fixRef != ref {
		t.Errorf("索引面应保持 fixed 与入库 ref: status=%s fix_ref=%s want=%s", status, fixRef, ref)
	}
	if attempts != 0 {
		t.Errorf("命中短路在预算闸门之前,不该计次: %d", attempts)
	}
}

// TestDiagnoseSkipAndBudgetArchive:缺复现上下文(D3)→ 每轮跳 G1/G2,零 LLM;
// 次数照计,三轮后预算归档(第四轮不再消耗次数、仍不调 LLM)。
func TestDiagnoseSkipAndBudgetArchive(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	project := "exec-itest-diag-skip"
	fp := core.Fingerprint("b4e8d3f2c5a61718")

	buildRunID := seedBuildRun(t, st, project)
	if _, err := failure.RecordEvent(ctx, st, failure.Event{
		ProjectID:       project,
		Source:          "build",
		BuildRunID:      buildRunID,
		Excerpt:         "panic: runtime error",
		Normalized:      "panic: runtime error",
		PipelineVersion: "1",
		Exact:           fp,
		Fuzzy:           diagFuzzy,
		Combo:           diagCombo(),
		OccurredAt:      time.Now().UTC(),
	}); err != nil {
		t.Fatalf("记 failure_event: %v", err)
	}

	p := noInference()
	table := Handlers(Config{
		Root:  t.TempDir(),
		Store: st.Queries(),
		DB:    st,
		Diagnose: DiagnoseDeps{
			Executor: p,
			Runner:   &gateRunner{seqs: map[string][]failure.RunResult{}, pos: map[string]int{}}, // 被调到就报错
			KB:       knowledge.NewStore(t.TempDir(), st.Queries()),                              // 空库:必然未命中
		},
	})
	run := func() error {
		return table[jobs.KindDiagnose](ctx,
			payload(t, jobs.DiagnosePayload{Fingerprint: string(fp), Project: project}))
	}

	for i := 0; i < 3; i++ {
		if err := run(); err != nil {
			t.Fatalf("第 %d 轮诊断(跳过)应成功: %v", i+1, err)
		}
	}
	var attempts int
	if err := st.Pool().QueryRow(ctx,
		"SELECT diagnose_attempts FROM fingerprint_stat WHERE fp_exact = $1", string(fp)).Scan(&attempts); err != nil {
		t.Fatalf("读 diagnose_attempts: %v", err)
	}
	if attempts != 3 {
		t.Errorf("三轮跳过各计一次,应是 3: %d", attempts)
	}

	// 第四轮:预算归档,短路在计次之前(attempts 不再涨),仍零 LLM。
	if err := run(); err != nil {
		t.Fatalf("归档轮应成功返回: %v", err)
	}
	if err := st.Pool().QueryRow(ctx,
		"SELECT diagnose_attempts FROM fingerprint_stat WHERE fp_exact = $1", string(fp)).Scan(&attempts); err != nil {
		t.Fatalf("读 diagnose_attempts: %v", err)
	}
	if attempts != 3 {
		t.Errorf("归档轮不应再计次: %d", attempts)
	}
	if len(p.Requests()) != 0 {
		t.Errorf("全程零推理(跳过轮不提、归档轮短路): %d 次", len(p.Requests()))
	}
	var status string
	if err := st.Pool().QueryRow(ctx,
		"SELECT status FROM fingerprint_stat WHERE fp_exact = $1", string(fp)).Scan(&status); err != nil {
		t.Fatalf("读 status: %v", err)
	}
	if status != "unresolved" {
		t.Errorf("无出口无归类,status 应仍 unresolved: %s", status)
	}
}

// TestDiagnoseUnknownFingerprintErrors:没有对应 failure_event 的指纹不得凭空诊断。
func TestDiagnoseUnknownFingerprintErrors(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()

	table := Handlers(Config{
		Root:  t.TempDir(),
		Store: st.Queries(),
		DB:    st,
		Diagnose: DiagnoseDeps{
			Executor: noInference(),
			Runner:   &gateRunner{seqs: map[string][]failure.RunResult{}, pos: map[string]int{}},
			KB:       knowledge.NewStore(t.TempDir(), st.Queries()),
		},
	})
	err := table[jobs.KindDiagnose](ctx,
		payload(t, jobs.DiagnosePayload{Fingerprint: "deadbeefdeadbeef", Project: "x"}))
	if err == nil {
		t.Fatal("未知指纹应报错")
	}
	if !strings.Contains(err.Error(), "无 failure_event") {
		t.Errorf("错误应说明先录入再诊断: %v", err)
	}
}

// countingGateRunner 在 gateRunner 之上记调用次数:G1 连跑 5 次,用计数证明真跑了。
type countingGateRunner struct {
	gateRunner
	calls int
}

func (r *countingGateRunner) Run(ctx context.Context, spec failure.RunSpec) (failure.RunResult, error) {
	r.calls++
	return r.gateRunner.Run(ctx, spec)
}

// TestDiagnoseWithReproContextRunsG1(T11b 验收):带 repro 录入的失败,诊断时装载出
// 非空 Repro,Handle 真跑 G1(fake Runner 断言被调)。对照 TestDiagnoseSkipAndBudgetArchive
// (缺 repro 跳 G1/G2)——同样的链路,只因为录入时带了复现上下文就能往下验证。
func TestDiagnoseWithReproContextRunsG1(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	project := "exec-itest-diag-repro"
	fp := core.Fingerprint("c5f7e1a9d2b83459")

	buildRunID := seedBuildRun(t, st, project)
	if _, err := failure.RecordEvent(ctx, st, failure.Event{
		ProjectID:       project,
		Source:          "build",
		BuildRunID:      buildRunID,
		Excerpt:         "make: *** [build] Error 1",
		Normalized:      "make: *** [build] Error N",
		PipelineVersion: "1",
		Exact:           fp,
		Fuzzy:           diagFuzzy,
		Combo:           diagCombo(),
		OccurredAt:      time.Now().UTC(),
		// 复现上下文(T11b):录入时携带,随失败持久化(0016 结构化合同)。
		Repro:        command.CommandSpec{Argv: []string{"make", "build"}, Cwd: "/tmp/repro-demo"},
		OtherTargets: []failure.OtherTarget{{Target: "other", Command: command.CommandSpec{Argv: []string{"make", "other"}}}},
	}); err != nil {
		t.Fatalf("记 failure_event: %v", err)
	}

	// 装载出的事件应带上复现上下文三列。
	ev, err := Config{DB: st}.loadDiagnoseEvent(ctx, fp)
	if err != nil {
		t.Fatalf("loadDiagnoseEvent: %v", err)
	}
	if ev.Repro.Cwd != "/tmp/repro-demo" || strings.Join(ev.Repro.Argv, " ") != "make build" {
		t.Errorf("应装载出复现上下文: repro=%+v", ev.Repro)
	}
	if len(ev.OtherTargets) != 1 || strings.Join(ev.OtherTargets[0].Command.Argv, " ") != "make other" {
		t.Errorf("应装载出其余构建目标: %v", ev.OtherTargets)
	}

	// G1 连跑 5 次;4 红 1 绿 → 复现不稳,归类 flaky,不调 LLM。
	// 用「复现不稳」收尾是为了把断言集中在 G1 真被跑到,不牵涉后续 LLM/G2/G3。
	r := &countingGateRunner{gateRunner: gateRunner{
		seqs: map[string][]failure.RunResult{
			"baseline|": {{}, {}, {Green: true}, {}, {}}, // 4 红 1 绿
		},
		pos: map[string]int{},
	}}
	p := noInference()
	table := Handlers(Config{
		Root:  t.TempDir(),
		Store: st.Queries(),
		DB:    st,
		Diagnose: DiagnoseDeps{
			Executor: p,
			Runner:   r,
			KB:       knowledge.NewStore(t.TempDir(), st.Queries()), // 空库:必然未命中
		},
	})
	if err := table[jobs.KindDiagnose](ctx,
		payload(t, jobs.DiagnosePayload{Fingerprint: string(fp), Project: project})); err != nil {
		t.Fatalf("诊断 job 应成功: %v", err)
	}
	if r.calls != 5 {
		t.Errorf("G1 应连跑 5 次复现: %d 次", r.calls)
	}
	if len(p.Requests()) != 0 {
		t.Errorf("复现不稳不得调推理: %d 次", len(p.Requests()))
	}
	var status string
	if err := st.Pool().QueryRow(ctx,
		"SELECT status FROM fingerprint_stat WHERE fp_exact = $1", string(fp)).Scan(&status); err != nil {
		t.Fatalf("读 status: %v", err)
	}
	if status != string(failure.OutcomeFlaky) {
		t.Errorf("复现不稳应归类 flaky: %s", status)
	}
}
