// Package web 提供 tally 的管理界面(v3:带写操作,非纯只读)。
//
// html/template + go:embed + HTMX,单二进制,无前端构建步骤。
// 模板与静态资源由 tally/web(web/embed.go)内嵌后导出,render.go 取用。
//
// 写边界(CLAUDE.md「界面的写边界」):写操作只有两种合法形态 ——
// 写 git 用户内容(项目/需求/文档/笔记)、入队 jobs 动作(T05b)。
// 绝不改状态字段(INV-3)、绝不写事实面(INV-5),由 test/arch 强制。
//
// 登录/CSRF:单用户 + 密码(bcrypt)+ session 表;写路由带 CSRF 双提交 token,
// 非公开路由一律过登录中间件(INV-9)。具体 CRUD handler 在 internal/web/handlers。
//
// 设计基准是 web/prototype/ 下的静态 HTML(只读参考),转模板时保持视觉一致。
//
// 允许导入:全部业务包、store、core、handlers。
package web
