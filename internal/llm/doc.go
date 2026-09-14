// Package llm 封装模型调用。
//
// Deprecated:本包是遗留的直连供应商路径(自带 BaseURL/APIKey/model 配置)。
// 全部通用推理消费方已迁到公共 AI Runtime 边界 internal/airuntime
// (P0-09 边界、P0-10 requirement/harvest、P0-11 failure/exec,DECISION_LOCKS #17 /
// ADR-0002)。本包暂不删除:cmd/tally 与 cmd/worker 的 composition root 仍经
// llmExecutor 过渡适配器引用它搬运传输,直到真实 AI Runtime SDK 接线任务落地;
// 那时整体删除本包。不要再在新代码里导入本包 —— 新调用方一律走 airuntime 边界。
//
// 只导出一个接口:
//
//	type Proposer interface {
//	    Propose(ctx context.Context, prompt string, schema any) (json.RawMessage, error)
//	}
//
// 只有 Propose。没有 Judge、Verify、Score、Evaluate、Check、Validate。
// 接口形状本身就是约束 —— 方法不存在,就没人能调错。
// test/arch 会用反射检查这一点。
//
// 历史调用方(已迁走):requirement(问询)、failure(修复提案)、
// harvest(服务定义提案)。三处都是 LLM 提、机器判。
//
// 所有送进来的内容必须先过 logpipe.MustBeRedacted。
//
// 允许导入:core、logpipe、标准库 net/http。
package llm
