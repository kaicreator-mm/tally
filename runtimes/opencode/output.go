package opencode

// output.go 拥有 `opencode run --format json` stdout 的逐行 JSON 事件解析:
// 从事件流提取最终回答(Summary)与 token 用量(Usage)。
//
// 观测契约(对 opencode-ai@1.19.6 的真实 run 取样:纯文本回合与带工具回合
// 各一,NDJSON 逐行一事件):
//
//	{"type":"step_start","timestamp":<ms>,"sessionID":"ses_...","part":{...,"type":"step-start"}}
//	{"type":"tool_use", "...","part":{"type":"tool","tool":"write","state":{...}}}
//	{"type":"text","...","part":{"type":"text","text":"<回答文本>","time":{...}}}
//	{"type":"step_finish","...","part":{"type":"step-finish","reason":"stop"|"tool-calls",
//	                                   "tokens":{"total":N,"input":N,"output":N,
//	                                             "reasoning":N,"cache":{...}},"cost":N}}
//
// 观测到的事件顺序(带工具回合):step_start → tool_use → step_finish →
// step_start → text → step_finish —— 一个 run 可含多个 step;回答取**最后**
// 一个 text 事件的 part.text;Usage 取全部 step_finish 事件 tokens.input/
// output 的**逐步求和**(input/output 是每 step 的 prompt/completion token,
// 求和即 run 级总量;tokens.total 语义未观测清楚 —— 疑似含 cache read 的
// 上下文量,刻意不使用)。
//
// 解析纪律:逐行解析,单行失败静默跳过(不 panic;半行/日志混入容忍);
// 只认观测过的 envelope type(text / step_finish),未知类型(未来版本新增)
// 忽略不影响回答提取。**任何解析失败都降级,绝不 panic**;降级路径(stdout
// 原文作为回答)由 classifyRun 处理。

import (
	"encoding/json"
	"strings"

	"tally/internal/agentruntime"
)

// runOutput 是一次 run stdout 的解析产物。
type runOutput struct {
	// Answer 是事件流中最后一个 text 事件的 part.text(观测:最终回答)。
	Answer string
	// Usage 是全部 step_finish 事件 tokens.input/output 的逐步求和;
	// 未观察到任何 step_finish 时为零值。
	Usage agentruntime.Usage
}

// opencodeEvent 是观测 envelope 的最小解码投影:只携带本适配器消费的字段,
// 其余(envelope 的 timestamp/sessionID、part 的其余形态)不解释。
type opencodeEvent struct {
	Type string `json:"type"`
	Part struct {
		Type string `json:"type"`
		Text string `json:"text"`
		// Tokens 只在 envelope type=step_finish 的 part(step-finish)中观测到。
		Tokens *struct {
			Input  int64 `json:"input"`
			Output int64 `json:"output"`
		} `json:"tokens"`
	} `json:"part"`
}

// parseRunOutput 解析 `--format json` 的 stdout(逐行 NDJSON)。返回事件流
// 提取的回答与用量;sawJSON 报告是否至少成功解析出一行观测形态的事件
// (供降级判定:整段 stdout 无一行可解析 = CLI 未按 JSON 形态输出)。
func parseRunOutput(stdout string) (out runOutput, sawJSON bool) {
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}
		var ev opencodeEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		switch ev.Type {
		case "text":
			// 观测:回答 = 最后一个 text 事件的 part.text(后到者覆盖)。
			out.Answer = ev.Part.Text
			sawJSON = true
		case "step_finish":
			if ev.Part.Tokens != nil {
				out.Usage.InputTokens += ev.Part.Tokens.Input
				out.Usage.OutputTokens += ev.Part.Tokens.Output
			}
			sawJSON = true
		default:
			// 观测过的其余类型(step_start/tool_use)与未观测类型都不携带
			// 回答/用量,跳过。但「合法 JSON 事件行」这一事实成立(降级判定
			// 区分「JSON 流无 text」与「根本不是 JSON 流」)。
			sawJSON = true
		}
	}
	return out, sawJSON
}
