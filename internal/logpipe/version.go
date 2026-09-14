package logpipe

// RuleVersion 是抽取 + 规范化规则的版本号,参与指纹计算。
//
// 规则一改,指纹随之变化,新旧天然不混淆;要对齐历史数据,用留存的原始日志
// 重算(cmd/tally reindex-fingerprints)。这就是为什么要留原始日志。
//
// 改规则 = 升版本号(单调递增,不复用)。没有这套机制,每优化一次抽取规则,
// 历史知识库就作废一次。
const RuleVersion = "1"
