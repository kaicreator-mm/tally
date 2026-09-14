package failure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"tally/internal/command"
	"tally/internal/core"
)

// —— 知识库替身 ——

type storedFix struct {
	fp    core.Fingerprint
	fix   core.Fix
	proof core.VerificationProof
}

type fakeKB struct {
	exact     map[core.Fingerprint]core.FixHit
	leads     []core.Lead
	stored    []storedFix
	storeErr  error
	lookupErr error
}

func newFakeKB() *fakeKB { return &fakeKB{exact: map[core.Fingerprint]core.FixHit{}} }

func (f *fakeKB) LookupExact(_ context.Context, fp core.Fingerprint, _ core.VersionCombo) (core.FixHit, bool, error) {
	if f.lookupErr != nil {
		return core.FixHit{}, false, f.lookupErr
	}
	hit, ok := f.exact[fp]
	return hit, ok, nil
}

func (f *fakeKB) LookupFuzzy(context.Context, core.Fingerprint, core.VersionCombo) ([]core.Lead, error) {
	return f.leads, nil
}

func (f *fakeKB) Store(_ context.Context, fp core.Fingerprint, fix core.Fix, proof core.VerificationProof) (string, error) {
	f.stored = append(f.stored, storedFix{fp: fp, fix: fix, proof: proof})
	if f.storeErr != nil {
		return "", f.storeErr
	}
	return "ref-" + string(fp), nil
}

// —— spec_defect 回写替身 ——

type fakeSink struct {
	known   map[string]bool
	reports []string
}

func (f *fakeSink) ReportSpecDefect(_ context.Context, projectID, clauseID, _ string) error {
	if !f.known[clauseID] {
		return fmt.Errorf("条款 %s 不存在", clauseID)
	}
	f.reports = append(f.reports, projectID+"/"+clauseID)
	return nil
}

// —— 组装 ——

func testEvent() Event {
	return Event{
		ProjectID:  "demo",
		Source:     "build",
		BuildRunID: 1,
		Excerpt:    "make: *** [build] Error 1",
		Exact:      "a3f9c2e1b4d50607",
		Fuzzy:      "f00f00f00f00f00f",
		Combo:      core.VersionCombo{"go": "1.23.0"},
		OccurredAt: time.Date(2026, 7, 22, 9, 0, 0, 0, time.UTC),
		Repro:      command.CommandSpec{Argv: []string{"make", "build"}, Cwd: "/tmp/demo"},
		OtherTargets: []OtherTarget{
			{Target: "other", Command: command.CommandSpec{Argv: []string{"make", "other"}, Cwd: "/tmp/demo"}},
		},
	}
}

func allGreenRunner() *scriptRunner {
	return newScriptRunner(map[string][]RunResult{
		"baseline|":   reds(5),
		"fixed|":      greens(5),
		"fixed|other": greens(1),
	})
}

func deps(p *fakeExecutor, r Runner, kb *fakeKB, st *fakeStats, sink SpecDefectSink) Deps {
	return Deps{
		Executor: p,
		Runner:   r,
		KB:       kb,
		Stats:    st,
		SpecSink: sink,
		Now:      func() time.Time { return fixedNow },
	}
}

// 验收:精确指纹命中 → 直接返回修复,一次 LLM 都不调,执行器也不用跑。
func TestHandleExactHitZeroLLM(t *testing.T) {
	kb := newFakeKB()
	kb.exact[testEvent().Exact] = core.FixHit{
		Fix:        core.Fix{Repro: "make build", Summary: "锁定依赖"},
		VerifiedAt: fixedNow,
		Stale:      false,
	}
	p := newFakeExecutor(okCandidateJSON)
	r := allGreenRunner()

	got, err := Handle(context.Background(), deps(p, r, kb, newFakeStats(), nil), testEvent())
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if got.Kind != OutcomeFixed || got.Fix == nil || got.Fix.Repro != "make build" {
		t.Errorf("精确命中应直接返回修复: %+v", got)
	}
	if p.calls() != 0 {
		t.Errorf("精确命中必须零 LLM 调用, got %d", p.calls())
	}
	if len(r.calls) != 0 {
		t.Errorf("精确命中不需要跑执行器, got %d 次", len(r.calls))
	}
}

// 验收:命中但 stale → 降级成线索进提示词,走完诊断流程。
func TestHandleStaleExactDemotedToLead(t *testing.T) {
	ev := testEvent()
	kb := newFakeKB()
	kb.exact[ev.Exact] = core.FixHit{
		Fix:        core.Fix{Repro: "make build", Summary: "旧修复"},
		VerifiedAt: fixedNow,
		Stale:      true,
	}
	p := newFakeExecutor(okCandidateJSON)

	got, err := Handle(context.Background(), deps(p, allGreenRunner(), kb, newFakeStats(), nil), ev)
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if got.Kind != OutcomeFixed {
		t.Errorf("过期命中应走重新验证并 fixed: %+v", got)
	}
	if p.calls() != 1 {
		t.Fatalf("应调一次 LLM, got %d", p.calls())
	}
	if !strings.Contains(p.prompt(0), string(ev.Exact)) {
		t.Errorf("过期命中应作为线索进提示词: %q", p.prompt(0))
	}
}

// 验收:模糊命中 → 线索进提示词。
func TestHandleFuzzyLeadsInPrompt(t *testing.T) {
	kb := newFakeKB()
	kb.leads = []core.Lead{{ExactFP: "beef0000beef0000", Status: "fixed"}}
	p := newFakeExecutor(okCandidateJSON)

	if _, err := Handle(context.Background(), deps(p, allGreenRunner(), kb, newFakeStats(), nil), testEvent()); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(p.prompt(0), "beef0000beef0000") {
		t.Errorf("模糊线索应进提示词: %q", p.prompt(0))
	}
}

// 验收:复现不稳 → flaky 归类,不调 LLM(不给偶发故障花一分钱)。
func TestHandleFlakyNoLLM(t *testing.T) {
	r := newScriptRunner(map[string][]RunResult{
		"baseline|": {{}, {}, {Green: true}, {}, {}}, // 4 红 1 绿
	})
	p := newFakeExecutor(okCandidateJSON)
	st := newFakeStats()

	got, err := Handle(context.Background(), deps(p, r, newFakeKB(), st, nil), testEvent())
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if got.Kind != OutcomeFlaky {
		t.Errorf("4 红 1 绿应归类 flaky: %+v", got)
	}
	if p.calls() != 0 {
		t.Errorf("flaky 不该调 LLM, got %d", p.calls())
	}
	if len(st.classified) != 1 || st.classified[0] != OutcomeFlaky {
		t.Errorf("应写入 flaky 归类: %v", st.classified)
	}
}

// 验收:四道闸全过 → 入库(带有效证据)并回写索引。
func TestHandleFixedFlow(t *testing.T) {
	ev := testEvent()
	kb := newFakeKB()
	p := newFakeExecutor(okCandidateJSON)
	st := newFakeStats()

	got, err := Handle(context.Background(), deps(p, allGreenRunner(), kb, st, nil), ev)
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if got.Kind != OutcomeFixed || got.FixRef == "" {
		t.Errorf("应 fixed 且带引用: %+v", got)
	}
	if len(kb.stored) != 1 {
		t.Fatalf("应入库一次, got %d", len(kb.stored))
	}
	if !kb.stored[0].proof.Valid() {
		t.Error("入库证据必须有效(铁律 2)")
	}
	// 入库修复应带结构化复现命令:core.Fix.Repro 是 string(core 不属本任务范围),
	// 落的是 CommandSpec 规范 JSON,原样可解回同一命令。
	var storedRepro command.CommandSpec
	if err := json.Unmarshal([]byte(kb.stored[0].fix.Repro), &storedRepro); err != nil {
		t.Fatalf("入库复现命令应是 CommandSpec 规范 JSON: %v", err)
	}
	if !reflect.DeepEqual(storedRepro, ev.Repro) {
		t.Errorf("入库修复应带结构化复现命令: %+v", storedRepro)
	}
	if st.fixes[ev.Exact] != got.FixRef {
		t.Errorf("索引应回写 fix_ref: %v", st.fixes)
	}
	if p.calls() != 1 {
		t.Errorf("一轮应调一次 LLM, got %d", p.calls())
	}
}

// 验收:修复后 4 绿 1 红 → 候选失败,无出口,不入库(记一轮预算)。
func TestHandleG2FailNoOutcome(t *testing.T) {
	r := newScriptRunner(map[string][]RunResult{
		"baseline|": reds(5),
		"fixed|":    {{Green: true}, {Green: true}, {Green: true}, {Green: true}, {}},
	})
	p := newFakeExecutor(okCandidateJSON)
	kb := newFakeKB()
	st := newFakeStats()

	got, err := Handle(context.Background(), deps(p, r, kb, st, nil), testEvent())
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if got.Kind != "" || got.Archived || got.Paused {
		t.Errorf("候选失败应无出口: %+v", got)
	}
	if len(kb.stored) != 0 {
		t.Error("未验证不得入库")
	}
	if len(st.classified) != 0 {
		t.Error("候选失败不归类")
	}
	if st.bumps != 1 {
		t.Errorf("应记一轮预算, got %d", st.bumps)
	}
}

// 验收:诊断三轮不过 → 归档「暂无自动修复」;同指纹再命中不再调 LLM。
func TestHandleBudgetArchiveAndNoMoreLLM(t *testing.T) {
	r := newScriptRunner(map[string][]RunResult{
		"baseline|": reds(5),
		"fixed|":    {{Green: true}, {Green: true}, {Green: true}, {Green: true}, {}}, // 4 绿 1 红,轮轮失败
	})
	p := newFakeExecutor(okCandidateJSON)
	d := deps(p, r, newFakeKB(), newFakeStats(), nil)

	for i := 0; i < maxAttempts; i++ {
		got, err := Handle(context.Background(), d, testEvent())
		if err != nil {
			t.Fatalf("第 %d 轮: %v", i+1, err)
		}
		if got.Kind != "" || got.Archived {
			t.Fatalf("第 %d 轮应无出口未归档: %+v", i+1, got)
		}
	}
	if p.calls() != maxAttempts {
		t.Fatalf("三轮应各调一次 LLM, got %d", p.calls())
	}

	got, err := Handle(context.Background(), d, testEvent())
	if err != nil {
		t.Fatalf("第四轮: %v", err)
	}
	if !got.Archived {
		t.Errorf("三轮用尽应归档「暂无自动修复」: %+v", got)
	}
	if p.calls() != maxAttempts {
		t.Errorf("归档后同指纹再命中不得再调 LLM, got %d", p.calls())
	}
}

// 验收:终态指纹再命中 → 直接返回该结论,零 LLM,零执行。
func TestHandleTerminalShortCircuit(t *testing.T) {
	ev := testEvent()
	st := newFakeStats()
	st.stats[ev.Exact] = &Stat{Status: "flaky", DiagnoseAttempts: 1}
	p := newFakeExecutor(okCandidateJSON)
	r := allGreenRunner()

	got, err := Handle(context.Background(), deps(p, r, newFakeKB(), st, nil), ev)
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if got.Kind != OutcomeFlaky {
		t.Errorf("终态应直接返回结论: %+v", got)
	}
	if p.calls() != 0 || len(r.calls) != 0 {
		t.Errorf("终态短路:LLM %d 次、执行 %d 次,都应为 0", p.calls(), len(r.calls))
	}
}

// 验收:G4 改到验证文件 → 直接拒绝(ErrFixTouchesTests),不入库。
func TestHandleFixTouchesTests(t *testing.T) {
	r := newScriptRunner(map[string][]RunResult{"baseline|": reds(5)})
	p := newFakeExecutor(
		`{"summary":"放松检查","changes":[{"path":"internal/app/app_test.go","content":"// 全过"}]}`)
	kb := newFakeKB()

	_, err := Handle(context.Background(), deps(p, r, kb, newFakeStats(), nil), testEvent())
	if !errors.Is(err, ErrFixTouchesTests) {
		t.Errorf("改验证文件应 ErrFixTouchesTests, got %v", err)
	}
	if len(kb.stored) != 0 {
		t.Error("被拒修复不得入库")
	}
}

// 验收:G4 改到应用源码 → 暂停等人确认(不报错,不入库)。
func TestHandleFixTouchesSourcePaused(t *testing.T) {
	r := newScriptRunner(map[string][]RunResult{"baseline|": reds(5)})
	p := newFakeExecutor(
		`{"summary":"改源码","changes":[{"path":"internal/app/handler.go","content":"package app\n"}]}`)
	kb := newFakeKB()

	got, err := Handle(context.Background(), deps(p, r, kb, newFakeStats(), nil), testEvent())
	if err != nil {
		t.Fatalf("改源码是暂停不是错误: %v", err)
	}
	if !got.Paused {
		t.Errorf("应暂停等人确认: %+v", got)
	}
	if len(kb.stored) != 0 {
		t.Error("暂停的修复不得入库")
	}
}

// 验收:spec_defect 出口 —— LLM 提案 + 条款确实存在(确定性闸门)→ 归类并回写。
func TestHandleSpecDefectConfirmed(t *testing.T) {
	r := newScriptRunner(map[string][]RunResult{"baseline|": reds(5)})
	p := newFakeExecutor(
		`{"summary":"x","changes":[],"spec_defect":{"clause_id":"REQ-12","ambiguity":"磁盘写满时的行为没写清楚"}}`)
	st := newFakeStats()
	sink := &fakeSink{known: map[string]bool{"REQ-12": true}}

	got, err := Handle(context.Background(), deps(p, r, newFakeKB(), st, sink), testEvent())
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if got.Kind != OutcomeSpecDefect {
		t.Errorf("条款存在应归类 spec_defect: %+v", got)
	}
	if len(sink.reports) != 1 || sink.reports[0] != "demo/REQ-12" {
		t.Errorf("应回写 requirement: %v", sink.reports)
	}
	if len(st.classified) != 1 || st.classified[0] != OutcomeSpecDefect {
		t.Errorf("应写入 spec_defect 归类: %v", st.classified)
	}
}

// 验收:spec_defect 提案指向不存在的条款 → 确定性闸门挡下,继续尝试修复。
func TestHandleSpecDefectClauseMissingFallsThrough(t *testing.T) {
	p := newFakeExecutor(
		`{"summary":"锁定依赖","changes":[{"path":"go.mod","content":"module demo\n"}],"spec_defect":{"clause_id":"REQ-999","ambiguity":"瞎编的条款"}}`)
	sink := &fakeSink{known: map[string]bool{}} // 什么条款都不认识
	st := newFakeStats()

	got, err := Handle(context.Background(), deps(p, allGreenRunner(), newFakeKB(), st, sink), testEvent())
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if got.Kind != OutcomeFixed {
		t.Errorf("条款不存在,提案不成立,应继续验证修复并 fixed: %+v", got)
	}
	if len(sink.reports) != 0 {
		t.Error("不存在的条款不得回写")
	}
}

// 验收:同指纹跨项目 → infra 出口(确定性信号:受影响项目数)。
func TestHandleInfraWhenCrossProject(t *testing.T) {
	ev := testEvent()
	st := newFakeStats()
	st.stats[ev.Exact] = &Stat{Status: "unresolved", AffectedProjects: []string{"demo", "another"}}
	r := newScriptRunner(map[string][]RunResult{
		"baseline|": reds(5),
		"fixed|":    {{Green: true}, {Green: true}, {Green: true}, {Green: true}, {}}, // 修复不过
	})
	p := newFakeExecutor(okCandidateJSON)

	got, err := Handle(context.Background(), deps(p, r, newFakeKB(), st, nil), ev)
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if got.Kind != OutcomeInfra {
		t.Errorf("跨项目失败应归类 infra: %+v", got)
	}
}

// 验收:失败与依赖升级对齐 → upstream 出口(确定性信号:dependency_change 时间线)。
func TestHandleUpstreamWhenDepAligned(t *testing.T) {
	st := newFakeStats()
	st.depChange = true
	r := newScriptRunner(map[string][]RunResult{
		"baseline|": reds(5),
		"fixed|":    {{Green: true}, {Green: true}, {Green: true}, {Green: true}, {}},
	})
	p := newFakeExecutor(okCandidateJSON)

	got, err := Handle(context.Background(), deps(p, r, newFakeKB(), st, nil), testEvent())
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if got.Kind != OutcomeUpstream {
		t.Errorf("依赖升级对齐应归类 upstream: %+v", got)
	}
}

// 知识库查错应向上返回,不得吞掉。
func TestHandleLookupError(t *testing.T) {
	kb := newFakeKB()
	kb.lookupErr = errors.New("仓库坏了")
	p := newFakeExecutor(okCandidateJSON)
	if _, err := Handle(context.Background(), deps(p, allGreenRunner(), kb, newFakeStats(), nil), testEvent()); err == nil {
		t.Error("查库出错应返回")
	}
	if p.calls() != 0 {
		t.Error("查库出错时不得调 LLM")
	}
}

// 验收(D3):缺复现上下文(Repro 无 Argv)→ 跳过 G1/G2,本轮不提候选也不验证,
// 零 LLM、执行器不跑;次数照计(预算仍生效)。
func TestHandleNoReproContextSkipsGates(t *testing.T) {
	st := newFakeStats()
	r := newScriptRunner(nil) // 空脚本:被调到就会报错,用来证明没跑
	p := newFakeExecutor(okCandidateJSON)

	ev := testEvent()
	ev.Repro = command.CommandSpec{}
	got, err := Handle(context.Background(), deps(p, r, newFakeKB(), st, nil), ev)
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !got.SkippedRepro {
		t.Errorf("缺复现上下文应标 SkippedRepro: %+v", got)
	}
	if got.Kind != "" || got.Fix != nil || got.Archived || got.Paused {
		t.Errorf("本轮不应有出口/修复/归档/暂停: %+v", got)
	}
	if p.calls() != 0 {
		t.Errorf("跳过轮不得调 LLM: %d 次", p.calls())
	}
	if len(r.calls) != 0 {
		t.Errorf("跳过轮不得跑执行器: %+v", r.calls)
	}
	if st.bumps != 1 {
		t.Errorf("次数照计(预算仍生效): bumps=%d", st.bumps)
	}
}

// 验收(D3):缺复现上下文也不影响知识库精确命中短路 —— 命中即返回修复,零 LLM。
func TestHandleNoReproContextExactHitStillFixes(t *testing.T) {
	kb := newFakeKB()
	kb.exact[testEvent().Exact] = core.FixHit{
		Fix:        core.Fix{Repro: "make build", Summary: "锁定依赖"},
		VerifiedAt: fixedNow,
	}
	p := newFakeExecutor(okCandidateJSON)

	ev := testEvent()
	ev.Repro = command.CommandSpec{}
	got, err := Handle(context.Background(), deps(p, newScriptRunner(nil), kb, newFakeStats(), nil), ev)
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if got.Kind != OutcomeFixed || got.SkippedRepro {
		t.Errorf("精确命中应先于跳过逻辑短路返回修复: %+v", got)
	}
	if p.calls() != 0 {
		t.Errorf("精确命中不得调 LLM: %d 次", p.calls())
	}
}
