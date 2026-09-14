package web

import (
	"html/template"
	"regexp"
	"strconv"
	"strings"
)

// renderMarkdown 把 markdown 渲染成安全 HTML,不引入第三方库(依赖白名单里没有 markdown 库,INV-6)。
//
// 只支持文档/笔记够用的子集:标题、段落、无序列表、围栏代码块、行内 code/加粗/链接。
// 安全约束:除本函数显式生成的白名单标签外,所有文本一律经 template.HTMLEscapeString 转义;
// 链接 URL 过 safeURL 白名单(拒绝 javascript:/data: 等)。输出因此可安全作为 template.HTML 注入。
func renderMarkdown(src string) template.HTML {
	lines := strings.Split(src, "\n")
	var out strings.Builder
	var para []string

	flushPara := func() {
		if len(para) == 0 {
			return
		}
		out.WriteString("<p>")
		out.WriteString(inlineMD(strings.Join(para, " ")))
		out.WriteString("</p>")
		para = nil
	}

	i := 0
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])

		// 围栏代码块:``` 之间原样转义输出,不做行内格式化。
		if strings.HasPrefix(trimmed, "```") {
			flushPara()
			var code []string
			i++
			for i < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
				code = append(code, lines[i])
				i++
			}
			i++ // 跳过收尾的 ```(到文件尾也安全)
			out.WriteString("<pre><code>")
			out.WriteString(template.HTMLEscapeString(strings.Join(code, "\n")))
			out.WriteString("</code></pre>")
			continue
		}

		if trimmed == "" {
			flushPara()
			i++
			continue
		}

		if level := headingLevel(trimmed); level > 0 {
			flushPara()
			text := strings.TrimSpace(trimmed[level:])
			out.WriteString("<h")
			out.WriteByte('0' + byte(level))
			out.WriteString(">")
			out.WriteString(inlineMD(text))
			out.WriteString("</h")
			out.WriteByte('0' + byte(level))
			out.WriteString(">")
			i++
			continue
		}

		if isBullet(trimmed) {
			flushPara()
			out.WriteString("<ul>")
			for i < len(lines) {
				t := strings.TrimSpace(lines[i])
				if !isBullet(t) {
					break
				}
				out.WriteString("<li>")
				out.WriteString(inlineMD(bulletText(t)))
				out.WriteString("</li>")
				i++
			}
			out.WriteString("</ul>")
			continue
		}

		para = append(para, trimmed)
		i++
	}
	flushPara()
	return template.HTML(out.String())
}

// headingLevel 返回行首 # 的个数(1–3),非标题返回 0。要求 # 后跟空格。
func headingLevel(s string) int {
	n := 0
	for n < len(s) && s[n] == '#' {
		n++
	}
	if n == 0 || n > 3 || n >= len(s) || s[n] != ' ' {
		return 0
	}
	return n
}

// isBullet 判断是否无序列表项(- 或 * 开头,后跟空格)。
func isBullet(s string) bool {
	return (strings.HasPrefix(s, "- ") || strings.HasPrefix(s, "* "))
}

// bulletText 去掉列表项前缀 "- " / "* "。
func bulletText(s string) string { return strings.TrimSpace(s[2:]) }

var (
	reInlineCode = regexp.MustCompile("`([^`]+)`")
	reBold       = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	reLink       = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
)

// inlineMD 渲染行内格式。先把整段转义,再在已转义文本上套白名单标签 ——
// 被包裹的内容此前已转义,故注入的 <code>/<strong>/<a> 不会带进原始 HTML。
func inlineMD(s string) string {
	esc := strings.ReplaceAll(template.HTMLEscapeString(s), "\x00", "")

	// 行内 code:先替换,用 NUL 占位符保护其内容不再被加粗/链接规则触碰。
	var codes []string
	esc = reInlineCode.ReplaceAllStringFunc(esc, func(m string) string {
		inner := reInlineCode.FindStringSubmatch(m)[1]
		codes = append(codes, "<code>"+inner+"</code>")
		return "\x00C" + strconv.Itoa(len(codes)-1) + "\x00"
	})

	esc = reBold.ReplaceAllString(esc, "<strong>$1</strong>")

	esc = reLink.ReplaceAllStringFunc(esc, func(m string) string {
		parts := reLink.FindStringSubmatch(m)
		text, url := parts[1], parts[2]
		safe := safeURL(url)
		if safe == "" {
			return text
		}
		return `<a href="` + template.HTMLEscapeString(safe) + `">` + text + `</a>`
	})

	// 还原 code 占位符。
	for idx, c := range codes {
		esc = strings.Replace(esc, "\x00C"+strconv.Itoa(idx)+"\x00", c, 1)
	}
	return esc
}

// safeURL 校验链接目标:拒绝 javascript:/data:/vbscript: 与含空白/控制字符的 URL,
// 其余(相对路径或常规 scheme)放行。空返回值表示「不成链接,只留文字」。
func safeURL(u string) string {
	t := strings.TrimSpace(u)
	if t == "" || strings.ContainsAny(t, " \t\r\n\x00-\x1f") {
		return ""
	}
	low := strings.ToLower(t)
	for _, bad := range []string{"javascript:", "data:", "vbscript:", "file:"} {
		if strings.HasPrefix(low, bad) {
			return ""
		}
	}
	return t
}
