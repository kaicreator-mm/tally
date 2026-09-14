// Package knowledge 存储已验证的修复(「已验证修复库」层)。
//
// 存三元组:(指纹, 复现命令, 已验证修复 + 验证时的版本组合)。
//
// 不存:LLM 分析过程、故障描述、复盘散文、未通过验证的结论。
//
// 两种匹配待遇不同,且由 API 形状强制:
//
//	LookupExact 精确指纹命中 → 返回修复(core.FixHit)
//	LookupFuzzy 模糊指纹命中 → 只返回线索(core.Lead,签名里没有 Fix 字段)
//
// 写方法 Store 要求 core.VerificationProof 验证证据:该证据只由 internal/failure
// 的四道闸全部通过后产出(具体类型字段不导出、构造函数不导出),未验证的东西
// 进不来(铁律 2)。knowledge 不导入 failure —— 业务包互不导入,经 core 共享接口,
// 接线由上层(web/jobs)完成。
//
// 入库 = git commit:内容写 knowledge/<精确指纹>/(fix.md / repro.sh / verified.json),
// 不写数据库;索引面(fingerprint_stat)只用于模糊召回,不存修复内容。
//
// 老化:命中时比对版本组合,对不上或带老化标记(STALE.md)→ 标 stale 降级成线索;
// 记录保留不删。
//
// 不导入 llm(由 test/arch 强制)。
//
// 允许导入:core、store、go-git。
package knowledge
