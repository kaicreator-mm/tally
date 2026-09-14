package logpipe

import (
	"strings"
	"testing"
)

// 该洗的必须洗掉。
func TestNormalizeWashes(t *testing.T) {
	tests := []struct{ name, in, gone string }{
		{"linux 家目录", "at /home/alice/proj/src/a.ts", "/home/alice"},
		{"macos 家目录", "at /Users/bob/proj/src/a.ts", "/Users/bob"},
		{"ISO 时间戳", "2026-07-21T10:33:02Z failed", "2026-07-21T10:33:02Z"},
		{"耗时", "took 4.21s", "4.21s"},
		{"pid", "pid=48122 crashed", "48122"},
		{"端口", "listen 127.0.0.1:53411", "53411"},
		{"内存地址", "at 0x7f3a2b crash", "0x7f3a2b"},
		{"uuid", "id 3f2b8c1a-9d4e-4f5a-8b6c-1a2b3c4d5e6f", "3f2b8c1a-9d4e-4f5a-8b6c-1a2b3c4d5e6f"},
		{"带前缀哈希", "sha256:abcdef0123456789", "abcdef0123456789"},
		{"临时构建目录", "using /tmp/build-xk29d/cache", "build-xk29d"},
	}
	for _, tt := range tests {
		if got := Normalize(tt.in); strings.Contains(got, tt.gone) {
			t.Errorf("%s: %q 未洗掉: %q -> %q", tt.name, tt.gone, tt.in, got)
		}
	}
}

// 绝对不能洗(洗了就失真,所有错误撞成一个,同样是静默故障)。
func TestNormalizeKeeps(t *testing.T) {
	tests := []struct{ name, in, keep string }{
		{"退出码 137", "exit status 137", "137"},
		{"退出码 1", "exit status 1", "1"},
		{"包版本", "lodash@4.17.21 not found", "4.17.21"},
		{"错误类型", "TypeError: x is not a function", "TypeError"},
		{"异常类名", "NullPointerException thrown", "NullPointerException"},
		{"行号(精确指纹保留)", "at /home/alice/proj/src/a.ts:12:5", ":12:5"},
	}
	for _, tt := range tests {
		if got := Normalize(tt.in); !strings.Contains(got, tt.keep) {
			t.Errorf("%s: %q 被误洗: %q -> %q", tt.name, tt.keep, tt.in, got)
		}
	}
}

// 同一个错误,只是时间/机器/路径前缀不同 → 规范化后完全一致。
func TestNormalizeCollapsesSameError(t *testing.T) {
	runs := []string{
		"2026-07-21T10:33:02Z ERROR /home/alice/proj/src/a.ts:12:5 pid=48122\nTypeError: x is not a function",
		"2026-07-22T08:15:44Z ERROR /home/bob/proj/src/a.ts:12:5 pid=1099\nTypeError: x is not a function",
		"2026-07-23T23:01:09Z ERROR /tmp/build-xk29d/proj/src/a.ts:12:5 pid=7\nTypeError: x is not a function",
	}
	first := Normalize(runs[0])
	for i, r := range runs[1:] {
		if got := Normalize(r); got != first {
			t.Errorf("第 %d 份日志规范化结果不同:\n基准: %s\n实际: %s", i+1, first, got)
		}
	}
}
