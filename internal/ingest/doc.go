// Package ingest 是统一的失败录入编排:把三类失败(构建 / 部署 / 环境)接进
// failure_event + fingerprint_stat,补上了 PRD 第四部分生产链上「失败已算出指纹,
// 却没人写 failure_event」的断点。
//
// 编排顺序:原始日志 → logpipe(按工具链选抽取器)→ failure.RecordEvent
// (同事务写事实面 + 累加索引面)。构建失败先写一行真实 build_run(来源外键约束),
// 再录 failure_event;部署 / 环境失败的接入点在编排层(exec / CLI),业务包不导入本包。
//
// 只 INSERT 事实面(INV-5),不碰任何 UPDATE;只录失败,成功构建只写 build_run。
//
// 允许导入:core、store、logpipe(含 toolchain)、failure。
// 不反向依赖:failure 不导入 ingest(方向是 ingest → failure)。
package ingest
