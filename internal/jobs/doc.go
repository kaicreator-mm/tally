// Package jobs 提供任务队列。
//
// Postgres 表 + 进程内 worker pool,默认并发 2。
//
// 任务类型是固定枚举,不支持运行时注册 —— 流水线图在设计期就定死了
// (收集 → 清洗 → 指纹 → 查库 → 提案 → 验证),不需要动态编排。
//
// 不用 Redis,不用 Temporal。引入编排引擎是过度设计。
//
// 允许导入:core、store。
package jobs
