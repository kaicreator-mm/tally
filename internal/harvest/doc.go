// Package harvest 从 Coolify 等来源抽取服务定义。
//
// 干什么:取来源材料 → LLM 产出候选定义(**必须含健康检查**)→ 经注入的 Verifier
// 实际拉起跑健康检查 → 通过的写进 catalog/(definition.yaml + verified.json + git commit)。
//
// 不干什么:不 fork Coolify;不依赖它的运行时(只读其公开的定义文件/文档);
// 不批量抽全部服务(清单显式有限,默认十来个,Coolify 全量有 362 个);不存未经验证的定义。
//
// 这是 llm 的三个合法调用方之一,但判定权在健康检查上,不在 LLM(铁律 1)。
// 未验证的定义进不了 catalog:写函数签名强制要 Proof,而 Proof 字段/构造都不导出(铁律 2)。
//
// 开工前置:catalog/LICENSE-NOTES.md 必须存在(Coolify 模板许可已查,见该文件)。
//
// 依赖注入,不 import 业务包:verify 的「实际拉起 + 健康检查」由上层(cmd/web)把
// internal/environment 的能力经 Verifier 接口注入;harvest 本身不 import environment。
// 允许导入:core、llm、logpipe。
package harvest
