package v14assurance

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestCorpusFrozen 机械强制语料库冻结(AC1):语料任何改动都会使本测试
// 失败,防止「看完结果再调样本」(包 §N)。
func TestCorpusFrozen(t *testing.T) {
	got := FreezeHash()
	if frozenCorpusHash == "PENDING" {
		t.Skipf("冻结哈希尚未回填,当前计算值: %s", got)
	}
	if got != frozenCorpusHash {
		t.Fatalf("语料库已被修改,违反冻结: got %s, frozen %s", got, frozenCorpusHash)
	}
}

// TestBakeoffDeterministic 两次运行捕获结论一致(split/判定确定性,RQ-04)。
func TestBakeoffDeterministic(t *testing.T) {
	a := RunBakeoff(context.Background(), nil)
	b := RunBakeoff(context.Background(), nil)
	if len(a) != len(b) {
		t.Fatalf("结果数不一致: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i].Caught != b[i].Caught || a[i].DefectID != b[i].DefectID || a[i].Layer != b[i].Layer {
			t.Fatalf("第 %d 条结果不确定: %+v vs %+v", i, a[i], b[i])
		}
	}
}

// TestHeldOutOnlyDefectsExist 语料 sanity:每个任务至少一个缺陷仅 L3 可捕,
// 保证 L3\L1 差集非空、可观测。
func TestHeldOutOnlyDefectsExist(t *testing.T) {
	s := Summarize(RunBakeoff(context.Background(), nil))
	for _, task := range corpus() {
		found := false
		for _, d := range s.L3MinusL1 {
			for _, m := range task.Mutants {
				if m.DefectID == d {
					found = true
				}
			}
		}
		if !found {
			t.Errorf("任务 %s 没有仅 held-out 可捕的缺陷", task.ID)
		}
	}
}

// TestScriptedJudgePath 仅验证 judge 注入路径的机械正确性
// (harness self-test,不是 bake-off 证据;报告不得引用)。
func TestScriptedJudgePath(t *testing.T) {
	// scripted judge:对 t3-m2(visible 漏网的保序缺陷)报 suspect。
	judge := func(_ context.Context, _ string, _ []Case) (bool, string, error) {
		return true, "scripted", nil
	}
	task := corpus()[2]
	m := task.Mutants[1] // t3-m2-adjacent-only
	r1 := Evaluate(context.Background(), task, m, LayerL1, nil)
	r2 := Evaluate(context.Background(), task, m, LayerL2, judge)
	if r1.Caught {
		t.Fatal("前置假设破坏:t3-m2 应逃过 visible 检查")
	}
	if !r2.Caught || !r2.JudgeRan {
		t.Fatalf("judge 路径失效: %+v", r2)
	}
}

// TestNilJudgeRecorded 无 judge 时 L2/L3 必须显式记 not_run,不得冒充捕获。
func TestNilJudgeRecorded(t *testing.T) {
	for _, r := range RunBakeoff(context.Background(), nil) {
		if r.Layer == LayerL1 {
			continue
		}
		if r.JudgeRan {
			t.Errorf("无 judge 注入却 JudgeRan=true: %+v", r)
		}
	}
	s := Summarize(RunBakeoff(context.Background(), nil))
	if s.L2Complete {
		t.Error("judge 未运行时 L2Complete 必须为 false")
	}
	if s.JudgeNote == "" {
		t.Error("judge 未运行时必须携带 JudgeNote")
	}
}

// TestBakeoffWriteResults 在 TALLY_V14_BAKEOFF_WRITE=1 时把结果 JSONL 与
// 摘要 JSON 落到研究目录;默认只校验可序列化(不污染仓库)。
func TestBakeoffWriteResults(t *testing.T) {
	results := RunBakeoff(context.Background(), nil)
	if os.Getenv("TALLY_V14_BAKEOFF_WRITE") != "1" {
		if _, err := json.Marshal(results); err != nil {
			t.Fatalf("结果不可序列化: %v", err)
		}
		return
	}
	dir := filepath.Join("..", "..", "..", "docs", "tasks", "v1.4", "research")
	writeJSONL(t, filepath.Join(dir, "A0-03-results.jsonl"), results)
	s := Summarize(results)
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		t.Fatalf("摘要序列化失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "A0-03-summary.json"), append(b, '\n'), 0o644); err != nil {
		t.Fatalf("写摘要失败: %v", err)
	}
}

func writeJSONL(t *testing.T, path string, results []LayerResult) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("写结果失败: %v", err)
	}
	defer func() { _ = f.Close() }()
	enc := json.NewEncoder(f)
	for _, r := range results {
		if err := enc.Encode(r); err != nil {
			t.Fatalf("结果编码失败: %v", err)
		}
	}
}
