package harvest

import (
	"os"
	"strings"
	"testing"
)

// 硬节点:catalog/LICENSE-NOTES.md 必须存在、非空、如实记录了许可与需人工确认处。
// 没有它不允许开始抽取(T02 开工前置)。
func TestLicenseNotesExistAndHonest(t *testing.T) {
	b, err := os.ReadFile("../../catalog/LICENSE-NOTES.md")
	if err != nil {
		t.Fatalf("LICENSE-NOTES.md 必须存在(开工硬节点): %v", err)
	}
	s := string(b)
	if len(strings.TrimSpace(s)) == 0 {
		t.Fatal("LICENSE-NOTES.md 不得为空")
	}
	if !strings.Contains(s, "Apache") {
		t.Error("应如实记录 Coolify 的许可(Apache-2.0)")
	}
	if !strings.Contains(s, "需人工确认") {
		t.Error("查不清处应明确标注「需人工确认」,绝不臆断许可结论")
	}
}
