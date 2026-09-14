package logpipe

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestStripANSI(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"去颜色码", "\x1b[31merror: boom\x1b[0m", "error: boom"},
		{"保留退出码", "exit status 137", "exit status 137"},
		{"去控制字符", "a\x00b\x07c", "abc"},
		{"保留制表换行", "a\tb\nc", "a\tb\nc"},
	}
	for _, tt := range tests {
		if got := StripANSI(tt.in); got != tt.want {
			t.Errorf("%s: StripANSI(%q) = %q, 想要 %q", tt.name, tt.in, got, tt.want)
		}
	}
}

func TestExtractTruncatesLongLine(t *testing.T) {
	long := strings.Repeat("x", 5000)
	in := "error: start\n" + long + "\nerror: end"
	out := Extract(in)
	for _, ln := range strings.Split(out, "\n") {
		if len([]rune(ln)) > maxLineLen+100 {
			t.Errorf("超长行未被截断(长度 %d)", len([]rune(ln)))
		}
	}
	if !strings.Contains(out, "已截断") {
		t.Error("缺少截断标记")
	}
}

func TestExtractCapturesMarkerContext(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 50; i++ {
		b.WriteString("leading noise " + strconv.Itoa(i) + "\n")
	}
	b.WriteString("prep done\n")
	b.WriteString("Error: cannot resolve dependency foo\n") // 标记行
	b.WriteString("aftermath line\n")
	for i := 0; i < 50; i++ {
		b.WriteString("trailing noise " + strconv.Itoa(i) + "\n")
	}
	out := Extract(b.String())
	if !strings.Contains(out, "cannot resolve dependency foo") {
		t.Error("未捕获标记行")
	}
	if !strings.Contains(out, "prep done") {
		t.Error("未捕获标记行前的上下文")
	}
	if strings.Contains(out, "leading noise 5") || strings.Contains(out, "trailing noise 30") {
		t.Errorf("捕获了远处的噪声:\n%s", out)
	}
}

// 堆栈跟踪必须整段取,不能切一半。
func TestExtractWholeStack(t *testing.T) {
	in := "running\n" +
		"Traceback (most recent call last):\n" +
		"  File \"/app/main.py\", line 10, in <module>\n" +
		"    do_thing()\n" +
		"  File \"/app/lib.py\", line 22, in do_thing\n" +
		"    return 1/0\n" +
		"ZeroDivisionError: division by zero\n" +
		"done\n"
	out := Extract(in)
	for _, frag := range []string{
		"Traceback (most recent call last):",
		"do_thing()",
		"return 1/0",
		"ZeroDivisionError: division by zero",
	} {
		if !strings.Contains(out, frag) {
			t.Errorf("堆栈被切断,缺少: %q\n输出:\n%s", frag, out)
		}
	}
}

// 并行日志先按服务分组,避免不同服务输出混进同一片段。
func TestExtractGroupsParallel(t *testing.T) {
	in := "db_1  | ready for connections\n" +
		"app_1 | npm ERR! code ECONNREFUSED\n" +
		"app_1 | npm ERR! errno ECONNREFUSED\n" +
		"web_1 | listening normally\n"
	out := Extract(in)
	if !strings.Contains(out, "[app_1]") {
		t.Errorf("未按服务分组:\n%s", out)
	}
	if !strings.Contains(out, "npm ERR!") {
		t.Errorf("未抽取到失败服务的内容:\n%s", out)
	}
	if strings.Contains(out, "[web_1]") || strings.Contains(out, "[db_1]") {
		t.Errorf("不应输出无失败的服务组:\n%s", out)
	}
}

// 无任何标记时兜底取头尾。
func TestExtractFallback(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 100; i++ {
		b.WriteString("line " + string(rune('0'+i%10)) + " all good\n")
	}
	in := b.String()
	out := Extract(in)
	if !strings.Contains(out, "省略") {
		t.Errorf("无标记时应兜底取头尾并标注省略:\n%s", out)
	}
}

// 工具链抽取器靠额外标记复用同一套分组/截断/整段取堆栈机制:
// 额外标记能命中的行,通用抽取不认(落到兜底取头尾,中间行被略过)。
func TestExtractWithExtraMarkers(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 40; i++ {
		b.WriteString("noise line " + strconv.Itoa(i) + "\n")
	}
	b.WriteString("CUSTOM_SIGNAL disk full imminent\n") // 非通用标记,在中间
	for i := 0; i < 40; i++ {
		b.WriteString("noise after " + strconv.Itoa(i) + "\n")
	}
	raw := b.String()

	extra := []*regexp.Regexp{regexp.MustCompile(`CUSTOM_SIGNAL`)}
	if got := ExtractWith(raw, extra); !strings.Contains(got, "CUSTOM_SIGNAL") {
		t.Errorf("额外标记未生效:\n%s", got)
	}
	if got := Extract(raw); strings.Contains(got, "CUSTOM_SIGNAL") {
		t.Errorf("通用抽取不应命中额外标记:\n%s", got)
	}
	// 不传额外标记时与原 Extract 行为一致
	if a, c := ExtractWith(raw, nil), Extract(raw); a != c {
		t.Error("ExtractWith(raw, nil) 应与 Extract(raw) 一致")
	}
}
