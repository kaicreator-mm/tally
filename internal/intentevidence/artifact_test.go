package intentevidence

import (
	"bytes"
	"errors"
	"testing"
)

// TestArtifactPathAdjudication:P0-01 对 open seam「FrozenIntent exact file
// path / schema」的路径侧裁决 —— .tally/intents/<task_id>.yaml,扁平布局,
// 与 .tally/{tasks,verifications,policies} 同构。常量是唯一权威,不得散落
// 字面量;路径由身份派生,身份非法就不存在合法路径。
func TestArtifactPathAdjudication(t *testing.T) {
	if ArtifactDir != ".tally/intents" {
		t.Errorf("ArtifactDir = %q, 裁决值为 .tally/intents", ArtifactDir)
	}
	got, err := ArtifactPath("TASK-0023")
	if err != nil {
		t.Fatalf("ArtifactPath(TASK-0023): %v", err)
	}
	if got != ".tally/intents/TASK-0023.yaml" {
		t.Errorf("ArtifactPath = %q, 应为 .tally/intents/TASK-0023.yaml", got)
	}
	for _, id := range []string{"", "   ", "TASK 0023"} {
		if _, err := ArtifactPath(id); !errors.Is(err, ErrInvalidEvidence) {
			t.Errorf("ArtifactPath(%q) 未拒绝: err = %v(身份非法必须 fail closed)", id, err)
		}
	}
	// 首尾空白是排版(与 Freeze/Parse 同一规范化):折叠后是同一路径。
	padded, err := ArtifactPath("  TASK-0023 ")
	if err != nil || padded != ".tally/intents/TASK-0023.yaml" {
		t.Errorf("ArtifactPath(带排版空白) = (%q, %v), 应折叠为同一路径", padded, err)
	}
}

// TestRenderRoundTripDeterministic:验收标准 1 —— 往返确定性。
// Freeze → Render → Parse 恒等(逐字段与 Revision 都相同),且 Render 是
// 不动点:Parse 后再渲染字节相同。
func TestRenderRoundTripDeterministic(t *testing.T) {
	fi, err := Freeze(Facts{
		TaskID:    "TASK-0100",
		TaskType:  TaskTypeBugFix,
		SourceRef: "GitHub Issue #142", // 含 #,YAML 需引号才能无损往返
		RiskHints: []string{"protected_surface_change", "auth_symbol_change"},
	})
	if err != nil {
		t.Fatalf("Freeze: %v", err)
	}

	b1, err := fi.Render()
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	b1again, err := fi.Render()
	if err != nil {
		t.Fatalf("Render(第二次): %v", err)
	}
	if !bytes.Equal(b1, b1again) {
		t.Error("同一证据两次渲染字节不同(渲染不确定)")
	}

	parsed, err := Parse(b1)
	if err != nil {
		t.Fatalf("Parse(Render(fi)): %v", err)
	}
	if !parsed.Equal(fi) {
		t.Error("往返后内容不等(往返不确定)")
	}
	if parsed.Revision() != fi.Revision() {
		t.Errorf("往返后 Revision 不等: %s != %s", parsed.Revision(), fi.Revision())
	}

	b2, err := parsed.Render()
	if err != nil {
		t.Fatalf("Render(往返后): %v", err)
	}
	if !bytes.Equal(b1, b2) {
		t.Error("Render 不是不动点:Parse 再渲染字节不同")
	}
}

// TestParseFormattingVariantsSameRevision:YAML 键序、流式/块式列表、引号
// 与缩进差异都是排版 —— 同一内容必须得到同一 Revision(哈希与排版无关)。
func TestParseFormattingVariantsSameRevision(t *testing.T) {
	fi, err := Freeze(Facts{
		TaskID:    "TASK-0100",
		TaskType:  TaskTypeMigration,
		RiskHints: []string{"auth_symbol_change", "protected_surface_change"},
	})
	if err != nil {
		t.Fatalf("Freeze: %v", err)
	}

	block := []byte(`task_id: TASK-0100
task_type: db_migration
risk_hints:
    - auth_symbol_change
    - protected_surface_change
`)
	flow := []byte("risk_hints: [protected_surface_change, auth_symbol_change]\n" +
		"task_type: 'db_migration'\n" +
		`task_id: "TASK-0100"` + "\n")

	for name, doc := range map[string][]byte{"块式": block, "流式乱序": flow} {
		got, err := Parse(doc)
		if err != nil {
			t.Fatalf("Parse(%s): %v", name, err)
		}
		if !got.Equal(fi) {
			t.Errorf("%s:解析内容与冻结值不等(规范化不确定)", name)
		}
		if got.Revision() != fi.Revision() {
			t.Errorf("%s:Revision 随排版漂移: %s != %s", name, got.Revision(), fi.Revision())
		}
	}
}

// TestParseFailsClosed:验收标准 2 —— 非法/未知的权威字段一律 fail closed。
// 特别钉住:provider/model/推理 prose 没有任何字段可写(要求 3),声明出来的
// revision 按未知字段拒收(哈希只能从内容重算)。
func TestParseFailsClosed(t *testing.T) {
	skeleton := "task_id: TASK-0023\ntask_type: bug_fix\n"
	unknownFields := []string{
		"summary: 推理产出的摘要 prose",
		"objective: 推理产出的目标 prose",
		"model: gpt-x",
		"provider: openai",
		"prompt: do things",
		"raw_output: untrusted prose",
		"notes: human notes",
		"status: frozen",
		"frozen: true",
		"revision: sha256:deadbeef",
		"approved: true",
	}
	for _, field := range unknownFields {
		doc := skeleton + field + "\n"
		if _, err := Parse([]byte(doc)); !errors.Is(err, ErrInvalidEvidence) {
			t.Errorf("schema 外字段 %q 未被拒收: err = %v(必须 fail closed)", field, err)
		}
	}

	cases := []struct {
		name    string
		doc     string
		wantErr error
	}{
		{"空文档", "", ErrInvalidEvidence},
		{"多文档", skeleton + "---\ntask_id: TASK-0024\ntask_type: bug_fix\n", ErrInvalidEvidence},
		{"缺 task_id", "task_type: bug_fix\n", ErrInvalidEvidence},
		{"task_id 含内部空白", "task_id: TASK 0023\ntask_type: bug_fix\n", ErrInvalidEvidence},
		{"空白提示条目", skeleton + "risk_hints:\n  - \"  \"\n", ErrInvalidEvidence},
		{"task_type 词表外", "task_id: TASK-0023\ntask_type: refactor\n", ErrUnsupportedTaskType},
		{"task_type 大小写不折叠", "task_id: TASK-0023\ntask_type: Bug_Fix\n", ErrUnsupportedTaskType},
	}
	for _, tc := range cases {
		if _, err := Parse([]byte(tc.doc)); !errors.Is(err, tc.wantErr) {
			t.Errorf("%s: err = %v, 期望 %v(必须 fail closed)", tc.name, err, tc.wantErr)
		}
	}
}

// TestParseRejectsNonMapping:顶层不是映射(标量/序列)按结构非法拒收,
// 不做 best-effort 猜测。
func TestParseRejectsNonMapping(t *testing.T) {
	for _, doc := range []string{"hello\n", "- a\n- b\n"} {
		if _, err := Parse([]byte(doc)); !errors.Is(err, ErrInvalidEvidence) {
			t.Errorf("非映射文档 %q 未被拒收: err = %v", doc, err)
		}
	}
}

// TestRenderZeroValueFailsClosed:零值不是证据,渲染必须报错而不是产出
// 空制品(空制品若可解析成功,就等于存在「没有任何确认」的合法证据 ——
// 与「证据来自确认」矛盾)。
func TestRenderZeroValueFailsClosed(t *testing.T) {
	var zero FrozenIntent
	if !zero.IsZero() {
		t.Fatal("零值 IsZero = false")
	}
	if _, err := zero.Render(); !errors.Is(err, ErrInvalidEvidence) {
		t.Errorf("零值 Render: err = %v, 应 fail closed", err)
	}
}

// TestParseEqualsFreezeSameContent:同一确认内容从两条构造路径(Freeze /
// Parse)进入,得到完全相同的内容与 Revision —— 不存在声明/观测漂移通道。
func TestParseEqualsFreezeSameContent(t *testing.T) {
	facts := Facts{
		TaskID:    "TASK-0077",
		TaskType:  TaskTypeMigration,
		SourceRef: "REQ-018",
		RiskHints: []string{"schema_migration"},
	}
	fi, err := Freeze(facts)
	if err != nil {
		t.Fatalf("Freeze: %v", err)
	}
	b, err := fi.Render()
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	parsed, err := Parse(b)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !parsed.Equal(fi) || parsed.Revision() != fi.Revision() {
		t.Error("Freeze 与 Parse 两条构造路径对同一内容产出不同身份")
	}
}
