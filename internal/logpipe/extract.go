package logpipe

import (
	"regexp"
	"strconv"
	"strings"
)

// 抽取:几千行压到几十行。做砸了 LLM 又贵又抓不到重点。
//
// 顺序(不能乱):
//  1. 去 ANSI 颜色码与控制字符 —— 必须在任何处理之前,不去掉后面全乱
//  2. 单行超长先截断(有的日志一行几 MB:base64、内嵌二进制)
//  3. 并行日志先按服务/step 分组,再按组抽取 —— 否则不同任务的输出混进同一片段
//  4. 按标记定位,取标记行前后若干行;堆栈跟踪整段取,不按行切

const (
	contextLines = 3    // 标记行前后各取几行
	maxLineLen   = 2000 // 单行超过就截断
	fallbackK    = 20   // 兜底:头尾各取几行
)

// 标记:命中即视为关键行。按工具链共性给一份,工具链特异模式在 toolchain/ 里补。
var markerREs = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\berror\b[: ]`),
	regexp.MustCompile(`\bERROR\b`),
	regexp.MustCompile(`\bFAILED\b`),
	regexp.MustCompile(`(?i)\bfatal\b[: ]`),
	regexp.MustCompile(`^npm ERR!`),
	regexp.MustCompile(`Traceback \(most recent call last\)`),
	regexp.MustCompile(`\bpanic:`),
	regexp.MustCompile(`(?i)\bexception\b`),
	regexp.MustCompile(`\b[A-Z]\w*(?:Error|Exception)\b`), // 异常类名:TypeError、ZeroDivisionError 等
	regexp.MustCompile(`(?i)exit(?:ed)? (?:with )?(?:status|code) [1-9]\d*`),
	regexp.MustCompile(`(?i)(?:command|build|compilation|step) (?:failed|did not complete)`),
	regexp.MustCompile(`(?i)cannot |could not |unable to `),
}

var (
	// ANSI:CSI 序列、OSC 序列、其余转义;再清掉其它控制字符(保留 \t \n \r)。
	reANSI = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]|\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)|\x1b[@-_]`)
	reCtrl = regexp.MustCompile(`[\x00-\x08\x0b\x0c\x0e-\x1f\x7f]`)

	// docker-compose 风格的服务前缀:`svc_1  | <内容>`
	reServicePrefix = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9_.-]{0,40})\s+\|\s?(.*)$`)
)

// StripANSI 去掉 ANSI 颜色码、转义序列与控制字符。必须在其它处理之前调用。
func StripANSI(s string) string {
	s = reANSI.ReplaceAllString(s, "")
	s = reCtrl.ReplaceAllString(s, "")
	return s
}

// Extract 从原始日志抽取关键片段(用通用标记集)。输入应是已粗脱敏的原始日志。
func Extract(raw string) string {
	return ExtractWith(raw, nil)
}

// ExtractWith 用「通用标记 + 额外标记」抽取,供 toolchain 包复用同一套
// 去 ANSI / 截断超长行 / 并行分组 / 整段取堆栈的机制,只换标记集。
// extra 为 nil 时与 Extract 完全一致。
func ExtractWith(raw string, extra []*regexp.Regexp) string {
	s := StripANSI(raw)      // 1. 去 ANSI,最先
	s = truncateLongLines(s) // 2. 截断超长行
	markers := markerREs
	if len(extra) > 0 {
		// 新建切片再追加,避免 append 别名污染包级 markerREs
		markers = make([]*regexp.Regexp, 0, len(markerREs)+len(extra))
		markers = append(markers, markerREs...)
		markers = append(markers, extra...)
	}
	for _, g := range groupByService(s) { // 3. 并行分组后按组抽取
		frag := extractMarkersWith(g.body, markers)
		if frag == "" {
			continue
		}
		if g.name != "" {
			frag = "[" + g.name + "]\n" + frag
		}
		return frag // 多组时只取第一个命中组(最常见的失败源),避免片段互相污染
	}
	return fallbackHeadTail(s) // 4. 兜底:头 + 尾
}

// truncateLongLines 把超长行截成 头 + 省略标记 + 尾,按 rune 切避免切碎多字节字符。
func truncateLongLines(s string) string {
	lines := strings.Split(s, "\n")
	for i, ln := range lines {
		runes := []rune(ln)
		if len(runes) <= maxLineLen {
			continue
		}
		head := string(runes[:maxLineLen/2])
		tail := string(runes[len(runes)-maxLineLen/4:])
		lines[i] = head + " …[行过长已截断]… " + tail
	}
	return strings.Join(lines, "\n")
}

type group struct{ name, body string }

// groupByService 把 docker-compose 风格的并行日志按服务前缀分组。
// 只有大多数行都带前缀才分组,避免把普通日志误拆。
func groupByService(s string) []group {
	lines := strings.Split(s, "\n")
	byName := map[string][]string{}
	var order []string
	matched := 0
	for _, ln := range lines {
		m := reServicePrefix.FindStringSubmatch(ln)
		if m == nil {
			continue
		}
		matched++
		if _, ok := byName[m[1]]; !ok {
			order = append(order, m[1])
		}
		byName[m[1]] = append(byName[m[1]], m[2])
	}
	if matched == 0 || matched*2 < len(lines) {
		return []group{{body: s}}
	}
	groups := make([]group, 0, len(order))
	for _, name := range order {
		groups = append(groups, group{name: name, body: strings.Join(byName[name], "\n")})
	}
	return groups
}

// extractMarkersWith 取所有标记行及其上下文;堆栈跟踪整段取。
func extractMarkersWith(s string, markers []*regexp.Regexp) string {
	lines := strings.Split(s, "\n")
	n := len(lines)
	keep := make([]bool, n)
	for i, ln := range lines {
		if !isMarker(ln, markers) {
			continue
		}
		for j := max(0, i-contextLines); j <= min(n-1, i+contextLines); j++ {
			keep[j] = true
		}
		if isStackStart(ln) { // 堆栈整段取,不按行切
			for j := i + 1; j < n && isStackContinuation(lines[j]); j++ {
				keep[j] = true
			}
		}
	}
	var out []string
	for i, ln := range lines {
		if keep[i] {
			out = append(out, ln)
		}
	}
	return strings.Join(out, "\n")
}

func isMarker(ln string, markers []*regexp.Regexp) bool {
	for _, re := range markers {
		if re.MatchString(ln) {
			return true
		}
	}
	return false
}

// isStackStart 判断是否一段堆栈跟踪的起点。
func isStackStart(ln string) bool {
	t := strings.TrimSpace(ln)
	return strings.HasPrefix(t, "Traceback (most recent call last)") ||
		strings.HasPrefix(t, "panic:") ||
		strings.HasPrefix(t, "goroutine ") ||
		strings.HasPrefix(t, "Exception in thread")
}

// isStackContinuation 判断某行是否堆栈的延续(缩进行 / `at` 行);空行或非缩进则止。
func isStackContinuation(ln string) bool {
	if strings.TrimSpace(ln) == "" {
		return false
	}
	if strings.HasPrefix(ln, " ") || strings.HasPrefix(ln, "\t") {
		return true
	}
	t := strings.TrimPrefix(ln, " ")
	return strings.HasPrefix(t, "at ") || strings.HasPrefix(t, "...")
}

// fallbackHeadTail 没有标记时的兜底:取头部与尾部各若干行。
func fallbackHeadTail(s string) string {
	lines := strings.Split(s, "\n")
	n := len(lines)
	if n <= 2*fallbackK {
		return s
	}
	return strings.Join(lines[:fallbackK], "\n") +
		"\n…[中间省略 " + strconv.Itoa(n-2*fallbackK) + " 行]…\n" +
		strings.Join(lines[n-fallbackK:], "\n")
}
