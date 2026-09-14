package requirement

import (
	"testing"

	"tally/internal/core"
)

// CheckReference:哈希一致 = 有效;不一致 = 依据已变更。
func TestCheckReference(t *testing.T) {
	h := core.ClauseHash("sha256:abc")
	if CheckReference(h, h) != RefValid {
		t.Error("同一哈希应为依据有效")
	}
	if CheckReference(core.ClauseHash("sha256:old"), h) != RefChanged {
		t.Error("旧哈希应为依据已变更")
	}
	if RefChanged.String() != "依据已变更" || RefValid.String() != "依据有效" {
		t.Errorf("状态文案: %q / %q", RefValid, RefChanged)
	}
}

// 条款内容一变,引用旧哈希的下游自动"依据已变更"(INV-4,无额外机制)。
func TestInvalidateOnContentChange(t *testing.T) {
	v1 := Clause{ID: "REQ-1", Statement: "系统 shall A", Mechanism: MechAutoTest, Criteria: []string{"x"}}
	oldHash := ContentHash(v1)

	// 条款内容变更 → 新哈希
	v2 := v1
	v2.Statement = "系统 shall B(改了)"
	newHash := ContentHash(v2)
	if newHash == oldHash {
		t.Fatal("内容变更后哈希应改变")
	}

	// 旧挂钩(指向 oldHash)对照当前版(newHash)→ 依据已变更
	if CheckReference(oldHash, newHash) != RefChanged {
		t.Error("指向旧版的挂钩应显示依据已变更")
	}
	// 按新版重新挂钩 → 有效
	if CheckReference(newHash, newHash) != RefValid {
		t.Error("指向当前版的挂钩应有效")
	}
}

// ClassifyReferences:逐条对照,删除的条款也算依据已变更。
func TestClassifyReferences(t *testing.T) {
	doc := cleanDoc()
	cur := ContentHash(doc.Clauses[0])

	refs := []Reference{
		{ClauseID: "REQ-1", Hash: cur},                // 有效
		{ClauseID: "REQ-1", Hash: "sha256:stale"},     // 已变更
		{ClauseID: "REQ-99", Hash: "sha256:anything"}, // 条款已不存在
	}
	reports := ClassifyReferences(refs, doc)
	want := []RefStatus{RefValid, RefChanged, RefChanged}
	for i, rp := range reports {
		if rp.Status != want[i] {
			t.Errorf("引用[%d] 状态 = %v, 期望 %v", i, rp.Status, want[i])
		}
	}
}
