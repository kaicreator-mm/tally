package toolchain

import (
	"strings"
	"testing"
)

// 基于真实样本:kamal 推镜像失败与健康等待超时。计时前缀的 ERROR 行须被抽到。
func TestKamalExtract(t *testing.T) {
	k := Kamal{}

	deploy := readSample(t, "../testdata/kamal/deploy-fail.log")
	out := k.Extract(deploy)
	for _, frag := range []string{"push access denied", "Deploy failed"} {
		if !strings.Contains(out, frag) {
			t.Errorf("推镜像失败片段缺少 %q:\n%s", frag, out)
		}
	}
	if k.Extract(deploy) != out {
		t.Error("Kamal.Extract 不确定:两次输出不同")
	}

	health := readSample(t, "../testdata/kamal/health-wait-timeout.log")
	healthOut := k.Extract(health)
	for _, frag := range []string{"Timed out waiting", "Health check failed"} {
		if !strings.Contains(healthOut, frag) {
			t.Errorf("健康超时片段缺少 %q:\n%s", frag, healthOut)
		}
	}
}
