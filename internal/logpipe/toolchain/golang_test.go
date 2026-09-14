package toolchain

import (
	"strings"
	"testing"
)

// 基于真实样本:go 编译错误与 vet 失败。行:列 必须保留(精确指纹据此区分位置)。
func TestGoExtract(t *testing.T) {
	g := Go{}

	compile := readSample(t, "../testdata/go/compile-error.log")
	out := g.Extract(compile)
	for _, frag := range []string{"handler.go:47:12:", "undefined: renderTemplate", "tally/internal/web"} {
		if !strings.Contains(out, frag) {
			t.Errorf("编译错误片段缺少 %q:\n%s", frag, out)
		}
	}
	if g.Extract(compile) != out {
		t.Error("Go.Extract 不确定:两次输出不同")
	}

	vet := readSample(t, "../testdata/go/vet-fail.log")
	vetOut := g.Extract(vet)
	for _, frag := range []string{"store.go:33:2:", "FAIL"} {
		if !strings.Contains(vetOut, frag) {
			t.Errorf("vet 片段缺少 %q:\n%s", frag, vetOut)
		}
	}
}
