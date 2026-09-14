package logpipe

import "tally/internal/core"

// 流水线编排:把六个纯函数阶段串成一条链。阶段本身各自可测,这里只负责
// 按正确顺序组合 —— 顺序是语义的一部分,不能乱(见 doc.go):
//
//	Collect 规范化 → Redact 粗 → Extract → Redact 细 → Normalize → Fingerprint
//
// 粗脱敏在抽取之前(原始内容不落盘就带密钥),细脱敏在抽取之后(片段小了再上更严的规则)。

// Result 是一次完整流水线的产出。
type Result struct {
	Excerpt    string           // 抽取 + 细脱敏后的关键片段,可安全落盘/展示
	Normalized string           // 规范化后文本
	Exact      core.Fingerprint // 精确指纹:只有它能关联已验证修复
	Fuzzy      core.Fingerprint // 模糊指纹:只能当召回线索
}

// Process 用通用抽取器跑完整流水线。确定性(INV-8)。
func Process(raw string) Result {
	return ProcessWith(raw, Extract)
}

// ProcessWith 跑完整流水线,抽取步骤由 extract 指定(通用 Extract,或 toolchain
// 包里某个工具链抽取器)。同一输入 byte 级相同输出。
func ProcessWith(raw string, extract func(string) string) Result {
	collected := CollectString(raw)
	coarse := RedactCoarse(collected)
	excerpt := RedactFine(extract(coarse))
	normalized := Normalize(excerpt)
	exact, fuzzy := Fingerprint(normalized)
	return Result{Excerpt: excerpt, Normalized: normalized, Exact: exact, Fuzzy: fuzzy}
}
