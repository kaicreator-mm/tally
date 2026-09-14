package toolchain

import (
	"regexp"

	"tally/internal/logpipe"
)

// docker 特异标记:buildkit 步骤失败、failed to solve、拉镜像 access denied、退出码。
var dockerMarkers = []*regexp.Regexp{
	regexp.MustCompile(`(?i)did not complete successfully`),
	regexp.MustCompile(`(?i)failed to solve`),
	regexp.MustCompile(`(?i)pull access denied`),
	regexp.MustCompile(`(?i)load metadata for`),
	regexp.MustCompile(`ERROR: process`),
	regexp.MustCompile(`exit code: [1-9]\d*`),
}

// Docker 抽取 docker build / 拉镜像的失败日志。
type Docker struct{}

// Name 返回 "docker"。
func (Docker) Name() string { return "docker" }

// Extract 取 docker 失败的关键片段(保留退出码)。
func (Docker) Extract(raw string) string { return logpipe.ExtractWith(raw, dockerMarkers) }
