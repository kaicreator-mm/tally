package toolchain

import (
	"regexp"

	"tally/internal/logpipe"
)

// compose 特异标记:健康检查失败、容器退出、Python Traceback、compose 顶层 ERROR。
// 带 `svc_1  |` 前缀的并行日志由 logpipe 的分组机制先按服务拆开,再抽失败的那组。
var composeMarkers = []*regexp.Regexp{
	regexp.MustCompile(`Traceback \(most recent call last\)`),
	regexp.MustCompile(`(?i)health ?check failed`),
	regexp.MustCompile(`(?i)did not become healthy`),
	regexp.MustCompile(`exited with status [1-9]\d*`),
	regexp.MustCompile(`^ERROR: for \S+`),
	regexp.MustCompile(`(?i)encountered errors while bringing up`),
}

// Compose 抽取 docker compose 的失败日志(并行服务 + 容器内崩溃)。
type Compose struct{}

// Name 返回 "compose"。
func (Compose) Name() string { return "compose" }

// Extract 取 compose 失败的关键片段;并行日志按服务分组后只取失败服务。
func (Compose) Extract(raw string) string { return logpipe.ExtractWith(raw, composeMarkers) }
