package toolchain

import (
	"strings"
	"testing"
)

// 基于真实样本:compose 健康检查失败(并行日志按服务分组)与容器内 Python 崩溃(整段取堆栈)。
func TestComposeExtract(t *testing.T) {
	c := Compose{}

	hc := readSample(t, "../testdata/compose/healthcheck-timeout.log")
	out := c.Extract(hc)
	if !strings.Contains(out, "[app_1]") {
		t.Errorf("并行日志未按失败服务分组:\n%s", out)
	}
	if !strings.Contains(out, "health check failed") {
		t.Errorf("未抽到失败服务的错误:\n%s", out)
	}
	// 健康的服务组不应出现在片段里
	if strings.Contains(out, "[web_1]") || strings.Contains(out, "[db_1]") {
		t.Errorf("不应输出无失败的服务组:\n%s", out)
	}
	if c.Extract(hc) != out {
		t.Error("Compose.Extract 不确定:两次输出不同")
	}

	exit := readSample(t, "../testdata/compose/container-exit.log")
	exitOut := c.Extract(exit)
	for _, frag := range []string{
		"[worker_1]",
		"Traceback (most recent call last):",
		"conn = psycopg2.connect(dsn)", // 堆栈中间行,检验整段取
		"psycopg2.OperationalError",
	} {
		if !strings.Contains(exitOut, frag) {
			t.Errorf("容器崩溃片段缺少 %q(堆栈可能被切断):\n%s", frag, exitOut)
		}
	}
}
