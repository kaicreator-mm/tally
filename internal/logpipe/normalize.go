package logpipe

import "regexp"

// 规范化:让同一个错误每次算出同一个指纹。这一步决定整个知识库有没有用 ——
// 洗不干净,指纹永不重复,知识库永远查不到东西,而表现出来只是「这些错误好像各不相同」。
//
// 必须洗掉:绝对路径、时间戳、耗时、PID/线程/端口、内存地址、UUID、内容哈希、临时目录名。
// 绝对不能洗:退出码(137 是内存超限,1 是普通失败)、包名与版本号(指纹最主要的信息)、错误类型。
//
// 顺序有讲究:先洗具体的(UUID、带前缀哈希、时间戳),再洗通用的(路径、长 hex),
// 避免通用规则破坏具体模式的结构。全部是确定性纯函数(INV-8)。
var normalizeRules = []struct {
	re   *regexp.Regexp
	repl string
}{
	// UUID
	{regexp.MustCompile(`\b[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\b`), "<uuid>"},
	// 带前缀的内容哈希
	{regexp.MustCompile(`\b(?:sha256|sha512|sha1|md5):[0-9a-fA-F]{8,}`), "<hash>"},
	// 内存地址
	{regexp.MustCompile(`\b0x[0-9a-fA-F]+\b`), "<addr>"},
	// 时间戳:ISO、common-log、纯日期、纯时间
	{regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?\b`), "<ts>"},
	{regexp.MustCompile(`\b\d{2}/[A-Za-z]{3}/\d{4}:\d{2}:\d{2}:\d{2}\b`), "<ts>"},
	{regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}\b`), "<ts>"},
	{regexp.MustCompile(`\b\d{2}:\d{2}:\d{2}(?:\.\d+)?\b`), "<ts>"},
	// 耗时
	{regexp.MustCompile(`\b\d+(?:\.\d+)?(?:ns|us|µs|ms|s|m|h)\b`), "<dur>"},
	// 进程/线程/goroutine/行编号
	{regexp.MustCompile(`(?i)\b(pid|tid)[=:\s]+\d+`), "${1}=<num>"},
	{regexp.MustCompile(`\bgoroutine \d+\b`), "goroutine <num>"},
	{regexp.MustCompile(`(?i)\bthread \d+\b`), "thread <num>"},
	{regexp.MustCompile(`(?i)\bline \d+\b`), "line <num>"},
	// IPv4
	{regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`), "<ip>"},
	// 端口:冒号后 4–5 位数字(行号通常 1–3 位,不受影响,精确指纹保留行号)
	{regexp.MustCompile(`:\d{4,5}\b`), ":<port>"},
	// 临时目录与用户目录前缀统一归一成 <root>:CI 的临时构建目录(/tmp/build-xk29d/proj/…)
	// 与开发机的 home 路径(/home/alice/proj/…)指同一个项目,必须撞到一起。
	{regexp.MustCompile(`/(?:private/)?tmp/[^/\s]+`), "<root>"},
	{regexp.MustCompile(`/var/folders/[^\s:]+`), "<root>"},
	{regexp.MustCompile(`(?i)[a-z]:\\users\\[^\\\s]+`), "<root>"},
	{regexp.MustCompile(`/home/[^/\s]+`), "<root>"},
	{regexp.MustCompile(`/Users/[^/\s]+`), "<root>"},
	{regexp.MustCompile(`\b/root\b`), "<root>"},
	// 裸长 hex(12+ 位:commit sha、内容哈希;版本号带点不是纯 hex,退出码太短)
	{regexp.MustCompile(`\b[0-9a-fA-F]{12,}\b`), "<hash>"},
}

// Normalize 洗掉可变部分,保留退出码、包名与版本、错误类型。
func Normalize(s string) string {
	for _, r := range normalizeRules {
		s = r.re.ReplaceAllString(s, r.repl)
	}
	return s
}
