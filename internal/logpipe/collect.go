package logpipe

import (
	"fmt"
	"strings"
)

// 收集是流水线第 0 步:把外部来源的原始日志变成一份规范化的字符串。
//
// Collect 自己不碰网络、文件系统、时钟 —— 来源由调用方注入(CI API 响应体、
// runner 输出、容器日志流各自实现 Source)。这样 Collect 是纯函数,可独立测试,
// 也满足 INV-8:相同输入 byte 级相同输出。规范化只做字节层面的归一(编码、换行),
// 脱敏是 RedactCoarse 的职责,不在这里。

// bom 是 UTF-8 字节序标记(用十六进制转义写,避免源文件里出现不可见字符)。
const bom = "\xef\xbb\xbf"

// Source 提供一批原始日志。实现由调用方注入;Collect 本身不做 I/O。
type Source interface {
	RawLog() (string, error)
}

// SourceFunc 让一个普通函数充当 Source。
type SourceFunc func() (string, error)

// RawLog 调用底层函数。
func (f SourceFunc) RawLog() (string, error) { return f() }

// Collect 从注入的来源取原始日志并规范化字节。来源出错时带上下文返回。
func Collect(src Source) (string, error) {
	raw, err := src.RawLog()
	if err != nil {
		return "", fmt.Errorf("取原始日志: %w", err)
	}
	return CollectString(raw), nil
}

// CollectString 规范化原始日志字节:去 BOM、统一换行为 \n、去结尾多余空行。
// 纯函数,相同输入 byte 级相同输出(INV-8)。
func CollectString(raw string) string {
	s := strings.TrimPrefix(raw, bom)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n") // 孤立的 \r(老 mac / 某些进度条)
	return strings.TrimRight(s, "\n")
}
