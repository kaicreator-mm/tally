package logpipe

import (
	"errors"
	"testing"
)

// 规范化字节:去 BOM、统一换行为 \n、去结尾多余空行。这是流水线的第 0 步,
// 不碰脱敏(那是 RedactCoarse)。
func TestCanonicalize(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"CRLF 归一", "a\r\nb\r\nc", "a\nb\nc"},
		{"孤立 CR 归一", "a\rb\rc", "a\nb\nc"},
		{"去 UTF-8 BOM", "\xef\xbb\xbf" + "npm ERR! boom", "npm ERR! boom"},
		{"去结尾空行", "error: x\n\n\n", "error: x"},
		{"混合归一", "\xef\xbb\xbf" + "line1\r\nline2\r\n\n", "line1\nline2"},
		{"空串", "", ""},
	}
	for _, tt := range tests {
		if got := CollectString(tt.in); got != tt.want {
			t.Errorf("%s: CollectString(%q) = %q, 想要 %q", tt.name, tt.in, got, tt.want)
		}
	}
}

// Collect 从注入的来源取日志并规范化;来源出错要带上下文往上抛。
func TestCollectFromSource(t *testing.T) {
	src := SourceFunc(func() (string, error) { return "a\r\nb\r\n", nil })
	got, err := Collect(src)
	if err != nil {
		t.Fatalf("Collect 返回错误: %v", err)
	}
	if got != "a\nb" {
		t.Errorf("Collect = %q, 想要 %q", got, "a\nb")
	}

	boom := errors.New("连接被重置")
	_, err = Collect(SourceFunc(func() (string, error) { return "", boom }))
	if !errors.Is(err, boom) {
		t.Errorf("来源错误应被 wrap 后透传,得到: %v", err)
	}
}
