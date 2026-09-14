package toolchain

import (
	"os"
	"path/filepath"
	"testing"
)

// readSample 读取 testdata 下的一个样本(toolchain 目录在 logpipe 下,故用 ../testdata)。
func readSample(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Clean(rel))
	if err != nil {
		t.Fatalf("读样本 %s: %v", rel, err)
	}
	return string(b)
}

func TestRegistry(t *testing.T) {
	want := map[string]bool{"npm": true, "go": true, "docker": true, "compose": true, "kamal": true}
	names := Names()
	if len(names) != len(want) {
		t.Errorf("Names() = %v, 期望 %d 个工具链", names, len(want))
	}
	for _, n := range names {
		if !want[n] {
			t.Errorf("多出未预期的工具链 %q", n)
		}
		e, ok := ByName(n)
		if !ok {
			t.Errorf("ByName(%q) 未注册", n)
			continue
		}
		if e.Name() != n {
			t.Errorf("ByName(%q).Name() = %q, 不一致", n, e.Name())
		}
	}
	if _, ok := ByName("cargo"); ok {
		t.Error("未注册的工具链不应命中")
	}
}
