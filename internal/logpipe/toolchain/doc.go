// Package toolchain 为各构建/部署工具链提供日志抽取器。
//
// 干什么:每个工具链(npm / go / docker / compose / kamal)一个 Extractor,
// 在 logpipe 的通用抽取之上补一层工具特异的关键标记 —— 让 `tally scan` 和
// 失败入库时能按 build_run.toolchain 选对抽取器,把几千行日志压到关键几十行。
//
// 不干什么:不脱敏、不规范化、不算指纹(那些是 logpipe 的阶段);抽取器只负责
// 「取片段」。每个 Extractor.Extract 是纯函数(INV-8)。
//
// 允许导入:logpipe、标准库。
package toolchain
