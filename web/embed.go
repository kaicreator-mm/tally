// Package web 内嵌界面资源:html/template 模板(web/templates)与静态文件(web/static)。
//
// 单独成包是因为 go:embed 只能内嵌与 .go 文件同目录或更深的文件 ——
// internal/web 无法直接 embed 到 ../web,故由本包持有 FS 再导出,
// internal/web/render.go 以别名导入取用(与 migrations/embed.go 同一模式)。
//
// 只有 templates 与 static 参与构建;web/prototype 是只读设计基准,不内嵌。
package web

import "embed"

// TemplateFS 内嵌全部页面模板(web/templates/*.tmpl)。
//
//go:embed templates/*.tmpl
var TemplateFS embed.FS

// StaticFS 内嵌静态资源(web/static:tally.css + vendored htmx.min.js)。
// all: 前缀确保不以字母数字开头的文件(若有)也被纳入。
//
//go:embed all:static
var StaticFS embed.FS
