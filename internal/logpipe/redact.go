package logpipe

import (
	"fmt"
	"regexp"
	"strings"
)

// 脱敏把疑似密钥替换成占位符。这是安全边界(docs/04-code-standards.md):
// 所有要送进 llm 包的内容必须先脱敏。**宁可误伤,不可漏一个** ——
// 漏一个 token,它就同时躺在数据库、git 和别人的服务器上。
//
// 做两遍,先粗后细:
//   - RedactCoarse —— 收集时,防原始日志落盘就带密钥(只打最明显的 token 格式)
//   - RedactFine   —— 抽取后,片段小了,加 key=value 与长 base64 等更严的规则

const redacted = "[REDACTED]"

var (
	// 常见 token 格式(粗筛即打)
	reGitHub = regexp.MustCompile(`\b(?:gh[pousr]_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,})`)
	reSKKey  = regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{20,}`)
	reAWS    = regexp.MustCompile(`\b(?:AKIA|ASIA)[0-9A-Z]{16}\b`)
	reGitLab = regexp.MustCompile(`\bglpat-[A-Za-z0-9_-]{20,}`)
	reSlack  = regexp.MustCompile(`\bxox[baprs]-[A-Za-z0-9-]{10,}`)
	reJWT    = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}`)

	// 头部与连接串(粗筛即打)
	reAuthHeader = regexp.MustCompile(`(?i)\b(Authorization|Proxy-Authorization)\s*[:=]\s*(?:Bearer\s+)?\S+`)
	reUserPass   = regexp.MustCompile(`\b([a-z][a-z0-9+.-]*)://[^:/\s@]+:[^@\s/]+@`)

	// 细筛:更多 secret key=value 与超长 base64
	reSecretKV = regexp.MustCompile(`(?i)\b(password|passwd|pwd|secret|token|api[_-]?key|access[_-]?key|private[_-]?key|client[_-]?secret)\s*[:=]\s*(?:"[^"]*"|'[^']*'|\S+)`)
	reLongB64  = regexp.MustCompile(`\b[A-Za-z0-9+/]{40,}={0,2}\b`)

	// 「关键字 + 占位符」的已脱敏形态。MustBeRedacted 检查前先剥掉这些,
	// 避免 "Authorization: [REDACTED]" 被重新判定为含密钥(RE2 不支持负向前瞻)。
	reAuthRedacted     = regexp.MustCompile(`(?i)\b(?:Authorization|Proxy-Authorization)\s*[:=]\s*(?:Bearer\s+)?` + regexp.QuoteMeta(redacted))
	reUserPassRedacted = regexp.MustCompile(`\b[a-z][a-z0-9+.-]*://` + regexp.QuoteMeta(redacted) + `@`)
	reSecretKVRedacted = regexp.MustCompile(`(?i)\b(?:password|passwd|pwd|secret|token|api[_-]?key|access[_-]?key|private[_-]?key|client[_-]?secret)\s*[:=]\s*` + regexp.QuoteMeta(redacted))
)

// RedactCoarse 粗脱敏:明显的 token 格式、鉴权头、连接串口令。用于原始日志落盘前。
func RedactCoarse(s string) string {
	s = reGitHub.ReplaceAllString(s, redacted)
	s = reSKKey.ReplaceAllString(s, redacted)
	s = reAWS.ReplaceAllString(s, redacted)
	s = reGitLab.ReplaceAllString(s, redacted)
	s = reSlack.ReplaceAllString(s, redacted)
	s = reJWT.ReplaceAllString(s, redacted)
	s = reAuthHeader.ReplaceAllString(s, "${1}: "+redacted)
	s = reUserPass.ReplaceAllString(s, "${1}://"+redacted+"@")
	return s
}

// RedactFine 细脱敏:在粗脱敏之上,加 key=value 形式的口令与超长 base64。用于抽取之后。
func RedactFine(s string) string {
	s = RedactCoarse(s)
	s = reSecretKV.ReplaceAllString(s, "${1}="+redacted)
	s = reLongB64.ReplaceAllString(s, redacted)
	return s
}

// MustBeRedacted 检查 s 是否仍含疑似密钥模式;含则返回错误,列出命中的模式。
// 在 llm 包入口调用:脱敏漏了就拒绝发送。
//
// 检查前先剥掉「关键字 + 占位符」的已脱敏形态,避免误判已脱敏文本。
func MustBeRedacted(s string) error {
	cleaned := reAuthRedacted.ReplaceAllString(s, "")
	cleaned = reUserPassRedacted.ReplaceAllString(cleaned, "")
	cleaned = reSecretKVRedacted.ReplaceAllString(cleaned, "")
	var hits []string
	for _, c := range []struct {
		name string
		re   *regexp.Regexp
	}{
		{"github-token", reGitHub},
		{"sk-key", reSKKey},
		{"aws-key", reAWS},
		{"gitlab-token", reGitLab},
		{"slack-token", reSlack},
		{"jwt", reJWT},
		{"authorization", reAuthHeader},
		{"user:pass@", reUserPass},
		{"secret-key=value", reSecretKV},
	} {
		if c.re.MatchString(cleaned) {
			hits = append(hits, c.name)
		}
	}
	if len(hits) > 0 {
		return fmt.Errorf("内容疑似含密钥,拒绝发送(命中: %s)", strings.Join(hits, ", "))
	}
	return nil
}
