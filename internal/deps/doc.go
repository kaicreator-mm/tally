// Package deps 采集依赖升级事实:解析项目 git 仓库里的锁文件 diff,
// 写进事实面 dependency_change(只 INSERT,INV-5)。
//
// 干什么(PRD 第三部分「从哪收」、第四部分「界面上要有什么」):
// 失败次数时间线要和依赖升级事件对齐 —— 哪根失败柱子旁边有升级刻度,
// 哪次升级就可能是元凶;归类「上游的问题」也吃这个确定性信号
// (failure 的 HasRecentDepChange)。没有录入方,那些消费端就永远空转。
//
// 采集 = 逐提交与父提交比锁文件内容(go.mod / package-lock.json),
// 解析成「包 → 版本」映射后 diff,变更落库;唯一索引保证 CLI 重跑幂等。
//
// 允许导入:store(写事实面)。不导入 llm(采集是确定性解析,INV-1)。
package deps
