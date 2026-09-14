package toolchain

import (
	"regexp"

	"tally/internal/logpipe"
)

// go 特异标记:包头(# pkg)、file:line:col 错误位、FAIL 结尾、undefined/cannot use。
var goMarkers = []*regexp.Regexp{
	regexp.MustCompile(`^#\s+\S+`),      // 包头: "# tally/internal/web"
	regexp.MustCompile(`\.go:\d+:\d+:`), // 错误定位: internal/x/a.go:47:12:
	regexp.MustCompile(`^FAIL\b`),       // 收尾: "FAIL\tpkg [build failed]"
	regexp.MustCompile(`\bundefined: \w+`),
	regexp.MustCompile(`\bcannot use \b`),
}

// Go 抽取 go build / go vet / go test 的失败日志。
type Go struct{}

// Name 返回 "go"。
func (Go) Name() string { return "go" }

// Extract 取 go 工具链失败的关键片段(保留 行:列,精确指纹据此区分不同位置)。
func (Go) Extract(raw string) string { return logpipe.ExtractWith(raw, goMarkers) }
