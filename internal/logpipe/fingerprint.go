package logpipe

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"

	"tally/internal/core"
)

// Fingerprint 从规范化文本算出精确指纹与模糊指纹。RuleVersion 参与计算 ——
// 规则一升级,新旧指纹自然不混淆(对齐历史用原始日志重算)。
//
//   - 精确指纹:规范化后直接哈希(保留行号与具体版本)。只有它能关联已验证修复。
//   - 模糊指纹:再去掉行号与版本号后哈希。只能当线索,不能当答案(见 knowledge)。
func Fingerprint(normalized string) (exact, fuzzy core.Fingerprint) {
	return fingerprintWith(RuleVersion, normalized)
}

// fingerprintWith 用指定规则版本算指纹。拆出来是为了能测试「RuleVersion 参与指纹」:
// 版本不同,同一规范化文本须算出不同指纹。导出的 Fingerprint 固定用当前 RuleVersion。
func fingerprintWith(ruleVersion, normalized string) (exact, fuzzy core.Fingerprint) {
	exact = core.Fingerprint(digest(ruleVersion + "\x00" + normalized))
	fuzzy = core.Fingerprint(digest(ruleVersion + "\x00" + blur(normalized)))
	return exact, fuzzy
}

var (
	reLineCol = regexp.MustCompile(`:\d+:\d+\b`)               // 行:列
	reFileLn  = regexp.MustCompile(`([A-Za-z0-9_./\\]):\d+\b`) // 文件:行号
	reVersion = regexp.MustCompile(`\b\d+\.\d+\.\d+(?:[-+.\w]+)?\b`)
)

// blur 在规范化之上再去掉行号与版本号,供模糊指纹。
func blur(s string) string {
	s = reLineCol.ReplaceAllString(s, "")
	s = reFileLn.ReplaceAllString(s, "${1}")
	s = reVersion.ReplaceAllString(s, "<v>")
	return s
}

// digest 取 sha256 前 8 字节(16 位十六进制)。对此规模足够,且短便于界面展示。
func digest(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:8])
}
