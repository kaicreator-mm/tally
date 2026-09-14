package toolchain

import (
	"strings"
	"testing"
)

// 基于真实样本:npm 依赖冲突与原生模块 ABI 失败,关键行都要被抽出来。
func TestNpmExtract(t *testing.T) {
	npm := Npm{}

	eresolve := readSample(t, "../testdata/npm/eresolve.log")
	out := npm.Extract(eresolve)
	for _, frag := range []string{"ERESOLVE", "npm ERR!", "react@18.2.0", "peer react@"} {
		if !strings.Contains(out, frag) {
			t.Errorf("ERESOLVE 片段缺少 %q:\n%s", frag, out)
		}
	}
	if npm.Extract(eresolve) != out {
		t.Error("Npm.Extract 不确定:两次输出不同")
	}

	abi := readSample(t, "../testdata/npm/native-module-abi/run-1.log")
	abiOut := npm.Extract(abi)
	for _, frag := range []string{"NODE_MODULE_VERSION", "bcrypt@5.1.1", "npm ERR!"} {
		if !strings.Contains(abiOut, frag) {
			t.Errorf("原生模块片段缺少 %q:\n%s", frag, abiOut)
		}
	}
}
