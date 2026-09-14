package failure

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"tally/internal/command"
	"tally/internal/core"
)

// —— 测试替身:确定性执行器(单测离线可跑,不依赖 docker/ssh)——

// scriptRunner 按 (是否带修复, 目标) 键回放预定红绿序列;序列放完后重复最后一个结果,
// 模拟「连跑 N 次」的确定性行为。
type scriptRunner struct {
	seqs  map[string][]RunResult
	pos   map[string]int
	calls []RunSpec
}

func newScriptRunner(seqs map[string][]RunResult) *scriptRunner {
	return &scriptRunner{seqs: seqs, pos: map[string]int{}}
}

func keyOf(s RunSpec) string {
	k := "baseline"
	if len(s.Fix) > 0 {
		k = "fixed"
	}
	return k + "|" + s.Target
}

func (r *scriptRunner) Run(_ context.Context, spec RunSpec) (RunResult, error) {
	r.calls = append(r.calls, spec)
	k := keyOf(spec)
	seq := r.seqs[k]
	if len(seq) == 0 {
		return RunResult{}, fmt.Errorf("没有脚本: %s", k)
	}
	p := r.pos[k]
	if p >= len(seq) {
		p = len(seq) - 1
	}
	r.pos[k] = p + 1
	return seq[p], nil
}

func (r *scriptRunner) count(prefix string) int {
	n := 0
	for _, c := range r.calls {
		if prefix == "" || (prefix == "baseline" && len(c.Fix) == 0) || (prefix == "fixed" && len(c.Fix) > 0) {
			n++
		}
	}
	return n
}

func reds(n int) []RunResult {
	out := make([]RunResult, n)
	return out // 零值 Green=false 即红
}

func greens(n int) []RunResult {
	out := make([]RunResult, n)
	for i := range out {
		out[i] = RunResult{Green: true}
	}
	return out
}

type errRunner struct{ err error }

func (r errRunner) Run(context.Context, RunSpec) (RunResult, error) { return RunResult{}, r.err }

var testChanges = []core.FileChange{
	{Path: "go.mod", Content: "module demo\n"},
	{Path: "docker-compose.yml", Content: "services: {}\n"},
}

var testCombo = core.VersionCombo{"go": "1.23.0"}

var fixedNow = time.Date(2026, 7, 22, 10, 0, 0, 0, time.UTC)

// —— G1:稳定复现 ——

// 验收:连跑 5 次全红才算稳定复现。
func TestConfirmReproAllRed(t *testing.T) {
	r := newScriptRunner(map[string][]RunResult{"baseline|": reds(5)})
	stable, err := ConfirmRepro(context.Background(), r, RunSpec{Command: command.CommandSpec{Argv: []string{"make", "build"}, Cwd: "/w"}})
	if err != nil {
		t.Fatalf("ConfirmRepro: %v", err)
	}
	if !stable {
		t.Error("5 次全红应判定稳定复现")
	}
	if n := r.count("baseline"); n != reproRuns {
		t.Errorf("应连跑 %d 次, got %d", reproRuns, n)
	}
	for _, c := range r.calls {
		if len(c.Fix) != 0 {
			t.Error("G1 复现不应带修复")
		}
	}
}

// 验收:5 红 4 红 1 绿 → 不稳定(flaky 的依据)。
func TestConfirmReproOneGreenIsUnstable(t *testing.T) {
	r := newScriptRunner(map[string][]RunResult{"baseline|": {
		{}, {}, {Green: true}, {}, {},
	}})
	stable, err := ConfirmRepro(context.Background(), r, RunSpec{Command: command.CommandSpec{Argv: []string{"make", "build"}}})
	if err != nil {
		t.Fatalf("ConfirmRepro: %v", err)
	}
	if stable {
		t.Error("4 红 1 绿不是稳定复现")
	}
}

func TestConfirmReproRunnerError(t *testing.T) {
	_, err := ConfirmRepro(context.Background(), errRunner{err: errors.New("执行器坏了")}, RunSpec{Command: command.CommandSpec{Argv: []string{"x"}}})
	if err == nil {
		t.Error("执行出错应向上返回")
	}
}

// —— G2 / G3 / 证据 ——

func TestVerifyCandidateAllPass(t *testing.T) {
	r := newScriptRunner(map[string][]RunResult{
		"fixed|":       greens(5),
		"fixed|other":  greens(1),
		"fixed|second": greens(1),
	})
	proof, err := VerifyCandidate(context.Background(), r, VerifyInput{
		ReproCommand: command.CommandSpec{Argv: []string{"make", "build"}, Cwd: "/w"},
		Candidate:    Candidate{Summary: "pin dep", Changes: testChanges},
		OtherTargets: []OtherTarget{
			{Target: "other", Command: command.CommandSpec{Argv: []string{"make", "other"}, Cwd: "/w"}},
			{Target: "second", Command: command.CommandSpec{Argv: []string{"make", "second"}, Cwd: "/w"}},
		},
		Combo: testCombo,
		Reds:  reproRuns,
		Now:   func() time.Time { return fixedNow },
	})
	if err != nil {
		t.Fatalf("VerifyCandidate: %v", err)
	}
	if !proof.Valid() {
		t.Error("四道闸全过的证据应有效")
	}
	if !proof.VerifiedAt().Equal(fixedNow) {
		t.Errorf("验证时刻应取自注入时钟: %v", proof.VerifiedAt())
	}
	if !reflect.DeepEqual(proof.VerifiedCombo(), testCombo) {
		t.Errorf("证据应带验证时版本组合: %v", proof.VerifiedCombo())
	}
	want := "g1=5/5red g2=5/5green g3=2/2targets g4=2files-ok"
	if proof.GateRecord() != want {
		t.Errorf("闸门记录 = %q, want %q", proof.GateRecord(), want)
	}
}

// 验收:4 绿 1 红 → 修复不过关,不入库。
func TestVerifyCandidateG2OneRed(t *testing.T) {
	r := newScriptRunner(map[string][]RunResult{
		"fixed|": {{Green: true}, {Green: true}, {Green: true}, {Green: true}, {}},
	})
	_, err := VerifyCandidate(context.Background(), r, VerifyInput{
		ReproCommand: command.CommandSpec{Argv: []string{"make", "build"}},
		Candidate:    Candidate{Changes: testChanges},
		Combo:        testCombo,
	})
	if !errors.Is(err, ErrFixNotGreen) {
		t.Errorf("4 绿 1 红应 ErrFixNotGreen, got %v", err)
	}
}

// 验收:其余构建目标受影响 → 不过关。
func TestVerifyCandidateG3Affected(t *testing.T) {
	r := newScriptRunner(map[string][]RunResult{
		"fixed|":      greens(5),
		"fixed|other": {{}}, // 其他目标挂了
	})
	_, err := VerifyCandidate(context.Background(), r, VerifyInput{
		ReproCommand: command.CommandSpec{Argv: []string{"make", "build"}},
		Candidate:    Candidate{Changes: testChanges},
		OtherTargets: []OtherTarget{{Target: "other", Command: command.CommandSpec{Argv: []string{"make", "other"}}}},
	})
	if !errors.Is(err, ErrOthersAffected) {
		t.Errorf("其他目标受影响应 ErrOthersAffected, got %v", err)
	}
}

// 验收:零值证据无效(只有闸流产出的才有效)。
func TestProofZeroInvalid(t *testing.T) {
	if (Proof{}).Valid() {
		t.Error("零值证据不应有效")
	}
}

// 验收(铁律 2 的编译期基础):证据类型字段全部不导出 ——
// failure 之外的包构造不出带内容的证据。
func TestProofFieldsUnexported(t *testing.T) {
	rt := reflect.TypeOf(Proof{})
	if rt.NumField() == 0 {
		t.Fatal("证据类型应携带验证时刻/版本组合/闸门记录")
	}
	for i := 0; i < rt.NumField(); i++ {
		if f := rt.Field(i); f.IsExported() {
			t.Errorf("证据字段 %s 导出了:包外可伪造证据,违反铁律 2", f.Name)
		}
	}
}

// Verify(四道闸连跑):G1 不稳 → ErrNotReproducible。
func TestVerifyG1Flaky(t *testing.T) {
	r := newScriptRunner(map[string][]RunResult{
		"baseline|": {{}, {Green: true}, {}, {}, {}},
		"fixed|":    greens(5),
	})
	_, err := Verify(context.Background(), r, VerifyInput{
		ReproCommand: command.CommandSpec{Argv: []string{"make", "build"}},
		Candidate:    Candidate{Changes: testChanges},
	})
	if !errors.Is(err, ErrNotReproducible) {
		t.Errorf("复现不稳应 ErrNotReproducible, got %v", err)
	}
}

func TestVerifyAllGatesPass(t *testing.T) {
	r := newScriptRunner(map[string][]RunResult{
		"baseline|": reds(5),
		"fixed|":    greens(5),
	})
	proof, err := Verify(context.Background(), r, VerifyInput{
		ReproCommand: command.CommandSpec{Argv: []string{"make", "build"}},
		Candidate:    Candidate{Changes: testChanges},
		Combo:        testCombo,
	})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !proof.Valid() {
		t.Error("全过证据应有效")
	}
}

// —— G4:白名单 ——

func TestCheckFixScopeAllWhitelisted(t *testing.T) {
	paths := []string{
		"go.mod", "go.sum", "package.json", "package-lock.json", "requirements.txt",
		"Dockerfile", "infra/Dockerfile.prod", "app.dockerfile",
		"docker-compose.yml", "docker-compose.dev.yml", "compose.yaml",
		".github/workflows/ci.yml", ".gitlab-ci.yml", "Jenkinsfile",
		"k8s/limits.yaml", "deploy/stack.yml", "config/deploy.yml",
		"scripts/healthcheck.sh", "ops/health-check.yml",
	}
	var changes []core.FileChange
	for _, p := range paths {
		changes = append(changes, core.FileChange{Path: p, Content: "x"})
	}
	if err := CheckFixScope(changes); err != nil {
		t.Errorf("白名单文件应全部放行: %v", err)
	}
}

// 验收:改到验证用的文件 → 直接拒绝(ErrFixTouchesTests),不是暂停。
func TestCheckFixScopeTouchesTests(t *testing.T) {
	testPaths := []string{
		"internal/app/handler_test.go",
		"tests/docker-compose.yml", // 测试目录里的 compose 也是验证设施
		"web/__tests__/login.test.ts",
		"testdata/fixtures/case.yml",
		"e2e/flow.spec.ts",
		"py/test_api.py",
		"spec/models_spec.rb",
	}
	for _, p := range testPaths {
		err := CheckFixScope([]core.FileChange{{Path: p, Content: "x"}})
		if !errors.Is(err, ErrFixTouchesTests) {
			t.Errorf("%s 应 ErrFixTouchesTests, got %v", p, err)
		}
	}
}

// 验收:改到应用源码 → 暂停等人确认(ErrFixTouchesSource)。
func TestCheckFixScopeTouchesSource(t *testing.T) {
	srcPaths := []string{"internal/app/handler.go", "src/main.ts", "lib/core.py"}
	for _, p := range srcPaths {
		err := CheckFixScope([]core.FileChange{{Path: p, Content: "x"}})
		if !errors.Is(err, ErrFixTouchesSource) {
			t.Errorf("%s 应 ErrFixTouchesSource, got %v", p, err)
		}
	}
}

// 测试文件与源码/白名单混杂时,拒绝(最强信号)优先于暂停。
func TestCheckFixScopeTestBeatsSource(t *testing.T) {
	err := CheckFixScope([]core.FileChange{
		{Path: "internal/app/handler.go"},
		{Path: "internal/app/handler_test.go"},
	})
	if !errors.Is(err, ErrFixTouchesTests) {
		t.Errorf("混改时测试文件拒绝应优先, got %v", err)
	}
}

func TestCheckFixScopeEmpty(t *testing.T) {
	if err := CheckFixScope(nil); !errors.Is(err, ErrEmptyFix) {
		t.Errorf("空修复应 ErrEmptyFix, got %v", err)
	}
}

func TestClassifyFile(t *testing.T) {
	cases := map[string]FileClass{
		"go.mod":                       FileWhitelisted,
		"Dockerfile":                   FileWhitelisted,
		"docker-compose.dev.yml":       FileWhitelisted,
		".github/workflows/ci.yml":     FileWhitelisted,
		"k8s/app.yaml":                 FileWhitelisted,
		"scripts/healthcheck.sh":       FileWhitelisted,
		"internal/app/handler.go":      FileSource,
		"cmd/server/main.go":           FileSource,
		"internal/app/handler_test.go": FileTest,
		"testdata/x.yml":               FileTest,
		"web/__tests__/a.test.ts":      FileTest,
		"tests/docker-compose.yml":     FileTest, // 白名单名字也挡不住测试目录
	}
	for path, want := range cases {
		if got := ClassifyFile(path); got != want {
			t.Errorf("ClassifyFile(%s) = %v, want %v", path, got, want)
		}
	}
}

// —— 结构化命令(P0-12:§21.6)——

// 验收(AC3):G3 跑各其余目标自己的结构化构建命令,Target 只是标签;
// 目标命令与复现命令互不窜扰,且带着修复跑(验证的是「修复不影响其余目标」)。
func TestVerifyCandidateG3RunsEachTargetCommand(t *testing.T) {
	r := newScriptRunner(map[string][]RunResult{
		"fixed|":      greens(5),
		"fixed|other": greens(1),
	})
	other := command.CommandSpec{Argv: []string{"make", "other"}, Cwd: "/w/other"}
	if _, err := VerifyCandidate(context.Background(), r, VerifyInput{
		ReproCommand: command.CommandSpec{Argv: []string{"make", "build"}, Cwd: "/w"},
		Candidate:    Candidate{Changes: testChanges},
		OtherTargets: []OtherTarget{{Target: "other", Command: other}},
		Combo:        testCombo,
		Reds:         reproRuns,
	}); err != nil {
		t.Fatalf("VerifyCandidate: %v", err)
	}
	var g3 []RunSpec
	for _, c := range r.calls {
		if c.Target == "other" {
			g3 = append(g3, c)
		}
	}
	if len(g3) != 1 {
		t.Fatalf("G3 应恰好跑一次其余目标, got %d", len(g3))
	}
	if !reflect.DeepEqual(g3[0].Command, other) {
		t.Errorf("G3 应原样跑该目标自己的命令: %+v", g3[0].Command)
	}
	if len(g3[0].Fix) != len(testChanges) {
		t.Error("G3 应带着修复跑")
	}
}

// 验收(AC1 的兼容面):0016 前存量行的旧文本两列按旧执行语义重建结构化命令 ——
// strings.Fields(repro) + Cwd=workdir 只活在这个明确命名的兼容装载器里。
func TestLegacyReproCommandRebuildsStructuredSpec(t *testing.T) {
	got := LegacyReproCommand("/tmp/demo", "make build")
	want := command.CommandSpec{Argv: []string{"make", "build"}, Cwd: "/tmp/demo"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("LegacyReproCommand = %+v, want %+v", got, want)
	}
}

// 验收:repro 为空(部署/环境失败本无可复现命令)→ Argv 为空的零值 spec,
// 调用方按「缺复现上下文」处理(D3),不得假装验证过。
func TestLegacyReproCommandEmptyReproYieldsEmptyArgv(t *testing.T) {
	got := LegacyReproCommand("/tmp/demo", "")
	if len(got.Argv) != 0 {
		t.Errorf("空 repro 应得空 Argv, got %+v", got.Argv)
	}
}
