package web

import (
	"strings"
	"testing"
)

// 指纹条是纯函数:同输入同输出,跨调用确定。这是签名元素,用测试钉住(见 06-ui-spec)。

func TestFingerprintBarDeterministic(t *testing.T) {
	fp := "a3f9c2e1"
	first := FingerprintBar(fp)
	for i := 0; i < 100; i++ {
		if got := FingerprintBar(fp); got != first {
			t.Fatalf("同一指纹第 %d 次渲染变了: %v != %v", i, got, first)
		}
	}
}

func TestFingerprintBarSameInputSameBar(t *testing.T) {
	// 同一指纹字符串在任何页面都得到同一条。
	a := FingerprintBar("sha256:9f21ac")
	b := FingerprintBar("sha256:9f21ac")
	if a != b {
		t.Errorf("同指纹应同条: %v != %v", a, b)
	}
}

func TestFingerprintBarLengthAndPalette(t *testing.T) {
	bar := FingerprintBar("7b1e04")
	if len(bar) != 8 {
		t.Fatalf("指纹条应 8 块,得到 %d", len(bar))
	}
	allowed := map[string]bool{}
	for _, c := range fpPalette {
		allowed[c] = true
	}
	for i, c := range bar {
		if !allowed[c] {
			t.Errorf("第 %d 块颜色 %q 不在调色板内", i, c)
		}
	}
}

func TestFingerprintBarDiffersAcrossFingerprints(t *testing.T) {
	// 不同指纹通常应得到不同的条(不是严格断言单对,而是整体上不全相同)。
	seen := map[[8]string]bool{}
	for _, fp := range []string{"a3f9c2", "7b1e04", "c50d8a", "f21b76", "9f21ac", "4e80d1"} {
		seen[FingerprintBar(fp)] = true
	}
	if len(seen) < 2 {
		t.Errorf("6 个不同指纹应渲染出不止一种条,只见到 %d 种", len(seen))
	}
}

func TestFpBarHTMLStructure(t *testing.T) {
	html := string(fpBarHTML("a3f9c2e1"))
	if got := strings.Count(html, "<i "); got != 8 {
		t.Errorf("指纹条 HTML 应含 8 个 <i>,得到 %d: %s", got, html)
	}
	if !strings.Contains(html, `class="fp-strip"`) {
		t.Errorf("缺少 fp-strip 容器: %s", html)
	}
	// 短指纹进了 aria-label,且被转义(不含原始 < >)。
	if !strings.Contains(html, "a3f9c2") {
		t.Errorf("aria-label 应含短指纹: %s", html)
	}
}

func TestFpShort(t *testing.T) {
	if got := fpShort("sha256:abcdef1234"); got != "abcdef" {
		t.Errorf("fpShort = %q, 想要 abcdef", got)
	}
	if got := fpShort("abc"); got != "abc" {
		t.Errorf("短于 6 位应原样,得到 %q", got)
	}
}

// markdown 渲染必须安全:除白名单标签外一律转义。

func TestMarkdownHeadingsAndList(t *testing.T) {
	html := string(renderMarkdown("# 标题\n\n- 甲\n- 乙\n\n一段正文。"))
	for _, want := range []string{"<h1>标题</h1>", "<ul>", "<li>甲</li>", "<li>乙</li>", "<p>一段正文。</p>"} {
		if !strings.Contains(html, want) {
			t.Errorf("渲染缺少 %q: %s", want, html)
		}
	}
}

func TestMarkdownCodeFence(t *testing.T) {
	html := string(renderMarkdown("```\ngo build <pkg>\n```"))
	if !strings.Contains(html, "<pre><code>") {
		t.Errorf("应有代码块: %s", html)
	}
	// <pkg> 必须被转义,不能留下真标签。
	if strings.Contains(html, "<pkg>") {
		t.Errorf("代码块内容未转义: %s", html)
	}
	if !strings.Contains(html, "&lt;pkg&gt;") {
		t.Errorf("代码块内容应转义为 &lt;pkg&gt;: %s", html)
	}
}

func TestMarkdownEscapesRawHTML(t *testing.T) {
	html := string(renderMarkdown("你好 <script>alert(1)</script> <b>粗</b>"))
	if strings.Contains(html, "<script>") || strings.Contains(html, "<b>") {
		t.Errorf("原始 HTML 未被转义(XSS): %s", html)
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Errorf("<script> 应被转义: %s", html)
	}
}

func TestMarkdownInlineCodeAndBold(t *testing.T) {
	html := string(renderMarkdown("用 `go test` 跑 **测试**"))
	if !strings.Contains(html, "<code>go test</code>") {
		t.Errorf("行内 code 未渲染: %s", html)
	}
	if !strings.Contains(html, "<strong>测试</strong>") {
		t.Errorf("加粗未渲染: %s", html)
	}
}

func TestMarkdownLinkSanitized(t *testing.T) {
	// 正常链接保留。
	safe := string(renderMarkdown("[文档](https://example.com/x)"))
	if !strings.Contains(safe, `<a href="https://example.com/x">文档</a>`) {
		t.Errorf("正常链接应保留: %s", safe)
	}
	// javascript: 链接被拆成纯文本。
	xss := string(renderMarkdown("[点我](javascript:alert(1))"))
	if strings.Contains(xss, "javascript:") || strings.Contains(xss, "<a ") {
		t.Errorf("javascript: 链接应被拆成纯文本(XSS): %s", xss)
	}
}
