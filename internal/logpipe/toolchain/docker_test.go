package toolchain

import (
	"strings"
	"testing"
)

// 基于真实样本:docker buildkit 构建失败与拉镜像失败。退出码须保留。
func TestDockerExtract(t *testing.T) {
	d := Docker{}

	build := readSample(t, "../testdata/docker/build-fail.log")
	out := d.Extract(build)
	for _, frag := range []string{"did not complete successfully", "exit code: 1", "no required module provides package"} {
		if !strings.Contains(out, frag) {
			t.Errorf("构建失败片段缺少 %q:\n%s", frag, out)
		}
	}
	if d.Extract(build) != out {
		t.Error("Docker.Extract 不确定:两次输出不同")
	}

	pull := readSample(t, "../testdata/docker/pull-fail.log")
	pullOut := d.Extract(pull)
	for _, frag := range []string{"pull access denied", "node:22-alpine"} {
		if !strings.Contains(pullOut, frag) {
			t.Errorf("拉镜像片段缺少 %q:\n%s", frag, pullOut)
		}
	}
}
