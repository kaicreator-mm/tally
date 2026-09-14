package toolchain

import (
	"regexp"

	"tally/internal/logpipe"
)

// kamal 特异标记:`HH.HHs ERROR` 时间前缀行、部署失败、健康等待超时、推镜像被拒。
var kamalMarkers = []*regexp.Regexp{
	regexp.MustCompile(`\d+\.\d+s\s+ERROR`), // kamal 的计时前缀错误行
	regexp.MustCompile(`(?i)deploy failed`),
	regexp.MustCompile(`(?i)timed out waiting`),
	regexp.MustCompile(`(?i)health check failed`),
	regexp.MustCompile(`(?i)push access denied`),
	regexp.MustCompile(`(?i)rolling back`),
}

// Kamal 抽取 kamal deploy 的失败日志。
type Kamal struct{}

// Name 返回 "kamal"。
func (Kamal) Name() string { return "kamal" }

// Extract 取 kamal 部署失败的关键片段(IP 等可变部分由后续规范化洗掉)。
func (Kamal) Extract(raw string) string { return logpipe.ExtractWith(raw, kamalMarkers) }
