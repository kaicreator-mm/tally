// Package store 是数据访问层。
//
// sqlc 生成的查询 + 薄仓储 + 事务辅助。
//
// 不放业务逻辑。
//
// 事实面表(build_run / deploy_run / environment_run / failure_event /
// dependency_change)只暴露 Insert 和 Select,不暴露 Update / Delete。
// 数据库层另有触发器兜底。
//
// 不导入 llm(由 test/arch 强制)。
//
// 允许导入:core、pgx。
package store
