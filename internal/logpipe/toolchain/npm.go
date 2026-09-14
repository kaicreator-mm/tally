package toolchain

import (
	"regexp"

	"tally/internal/logpipe"
)

// npm 特异标记:npm ERR! 块与错误码(ERESOLVE/ETARGET/ELIFECYCLE 等)、原生模块 ABI。
var npmMarkers = []*regexp.Regexp{
	regexp.MustCompile(`^npm ERR!`),
	regexp.MustCompile(`\bERESOLVE\b`),
	regexp.MustCompile(`\bcode E[A-Z0-9]+\b`),
	regexp.MustCompile(`\bnotarget\b`),
	regexp.MustCompile(`NODE_MODULE_VERSION \d+`),
	regexp.MustCompile(`peer \S+ from`),
}

// Npm 抽取 npm install / run 的失败日志。
type Npm struct{}

// Name 返回 "npm"。
func (Npm) Name() string { return "npm" }

// Extract 取 npm 失败的关键片段。
func (Npm) Extract(raw string) string { return logpipe.ExtractWith(raw, npmMarkers) }
