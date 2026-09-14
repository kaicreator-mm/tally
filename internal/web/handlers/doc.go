// Package handlers 是 tally 界面的写路由层(T05a)。
//
// 写边界(CLAUDE.md「界面的写边界」):这里的写只有两种合法形态 ——
//
//   - 写用户内容:项目 / 需求 / 文档 / Wiki 笔记,落 git 并 commit(可 diff / 回滚);
//   - 触发动作:入队 jobs(T05a 尚无动作路由,入队 helper 给 T05b 留位)。
//
// 绝不改状态字段(INV-3)、绝不写事实面(INV-5)。任务状态从执行链算出,只读展示。
//
// 依赖方向:web → handlers →(project / requirement / store / core)。
// handlers 不反向导入 web:渲染经 PageRenderer 接口注入(*web.Renderer 满足它)。
//
// 允许导入:全部业务包、store、core。
package handlers
