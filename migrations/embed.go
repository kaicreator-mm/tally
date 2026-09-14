// Package migrations 持有 goose 迁移脚本,供 cmd/tally 以库方式执行(不依赖 goose CLI)。
package migrations

import "embed"

// FS 内嵌全部迁移脚本。goose 从该 FS 的根读取 *.sql。
//
//go:embed *.sql
var FS embed.FS
