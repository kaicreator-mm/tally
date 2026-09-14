package logpipe

import (
	"strings"
	"testing"
)

// 整套流水线(收集→粗脱敏→抽取→细脱敏→规范化→指纹)必须确定性:
// 同一输入连跑两次,逐字段 byte 级相同(INV-8)。
func TestProcessDeterminism(t *testing.T) {
	raw := "2026-07-21T10:33:02Z ERROR /home/alice/proj/src/a.ts:12:5 pid=48122\n" +
		"TypeError: x is not a function\r\n" +
		"exit status 1\n"
	a := Process(raw)
	b := Process(raw)
	if a != b {
		t.Errorf("Process 不确定:\n%+v\nvs\n%+v", a, b)
	}
}

// 粗脱敏在抽取之前,所以落盘的 Excerpt 不得含密钥;规范化不得含可变路径。
func TestProcessRedactsAndNormalizes(t *testing.T) {
	raw := "clone with ghp_aB3dE5gH7jK9mN1pQ3sT5vW7xY9z\n" +
		"npm ERR! code ERESOLVE\n" +
		"npm ERR! 404 Not Found - GET https://x/npm\n" +
		"error: Cannot find module 'lodash' at /home/alice/proj/node_modules\n"
	r := Process(raw)
	if strings.Contains(r.Excerpt, "ghp_aB3dE5gH7jK9mN1pQ3sT5vW7xY9z") {
		t.Errorf("Excerpt 含密钥(粗脱敏失效): %q", r.Excerpt)
	}
	if strings.Contains(r.Normalized, "/home/alice") {
		t.Errorf("Normalized 未洗掉路径: %q", r.Normalized)
	}
	if !strings.Contains(r.Normalized, "ERESOLVE") {
		t.Errorf("Normalized 丢失错误码: %q", r.Normalized)
	}
	if r.Exact == "" || r.Fuzzy == "" {
		t.Error("指纹为空")
	}
}

// ProcessWith 的抽取步骤可注入:换抽取器,产出的 Excerpt 随之改变。
func TestProcessWithCustomExtract(t *testing.T) {
	raw := "error: boom happened here\n"
	custom := func(string) string { return "CUSTOM-FRAGMENT" }
	r := ProcessWith(raw, custom)
	if !strings.Contains(r.Excerpt, "CUSTOM-FRAGMENT") {
		t.Errorf("未使用注入的抽取器: %q", r.Excerpt)
	}
}
