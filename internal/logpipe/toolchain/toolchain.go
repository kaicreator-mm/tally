package toolchain

import "sort"

// Extractor 针对某一工具链的日志抽取关键片段。实现用 logpipe.ExtractWith
// 复用通用抽取的去 ANSI / 截断超长行 / 并行分组 / 整段取堆栈机制,只换标记集。
// Extract 是纯函数:相同输入 byte 级相同输出(INV-8)。
type Extractor interface {
	// Name 返回工具链名,与 build_run.toolchain 一致。
	Name() string
	// Extract 从(已粗脱敏的)原始日志取关键片段。
	Extract(raw string) string
}

// registry 是工具链名到抽取器的映射。build_run.toolchain 据此选抽取器。
var registry = map[string]Extractor{
	"npm":     Npm{},
	"go":      Go{},
	"docker":  Docker{},
	"compose": Compose{},
	"kamal":   Kamal{},
}

// ByName 按工具链名取抽取器。未知工具链返回 false,调用方应退回通用抽取。
func ByName(name string) (Extractor, bool) {
	e, ok := registry[name]
	return e, ok
}

// Names 返回全部已注册工具链名,排序保证输出确定。
func Names() []string {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
