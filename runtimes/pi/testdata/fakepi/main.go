// Command fakepi 是 pi 适配器测试的替身 pi CLI(仅标准库,由 runtimes/pi 的
// TestMain 按 go build 文件模式现场构建)。
//
// 它只模仿本机对 pi 0.85.1 观测到的表面,不发明任何旗标:
//
//   - `--version` → 裸版本行 "0.85.1"(观测形态);
//   - `-p --mode <m> --no-session --session-dir <d> --no-extensions
//     --no-skills --no-prompt-templates --no-themes --no-context-files
//     --no-approve [--provider <p>] [--model <m>] [--thinking <t>] --
//     <instruction>` → 一次性进程:json 模式 stdout = NDJSON 事件流(session
//     行 + assistant message_end 携带固定自报 provider/model/usage;text
//     模式 stdout = 最终回答纯文本);exit 0 = 成功 / 非 0 = 未完成;cwd =
//     进程 cwd(workspace 钉定的效果级取证);`read <file>` 指令输出 cwd 下
//     该文件内容;`tally-conformance-long` 指令 spawn 子进程并等待(整树终止
//     取证面,形态对齐 fakedsh)。
//
// 场景注入(仅测试设置,生产 runtime 永不设置):
//   - TALLY_FAKE_PI_VERSION 覆盖 --version 输出(钉版闸负例);
//   - TALLY_FAKE_PI_SCENARIO=versionfail(--version 退出 3)/ exitfail(部分
//     输出后退出 7)/ emptyok(exit 0 无输出)/ leakstderr(stderr 夹带密钥
//     形态文本后退出 1,脱敏取证);
//   - TALLY_FAKE_PI_REQUIRE_KEY=1 时缺 DEEPSEEK_API_KEY → stderr
//     "No API key found for deepseek."、exit 1(观测的 keyless 契约形态);
//   - TALLY_FAKE_PI_RECORD 逐行 JSON 记录观测面(argv/cwd/mode/隔离与强制
//     环境值/provider key 注入与剥离/子进程 PID),供测试断言。
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// pinnedVersion 是替身自报的钉住版本(与 runtimes/pi.PinnedPiVersion 一致)。
const pinnedVersion = "0.85.1"

// longMarker 是「长运行指令」标记(conformance 的 LongRunInstruction 注入点)。
const longMarker = "tally-conformance-long"

// fakeSessionID 是替身自报的会话 UUID(观测形态:type=session 行,id 字段;
// 固定值即可 —— 测试只断言「适配器观察到的 == 替身自报的」)。
const fakeSessionID = "f0000000-0000-4000-8000-000000000001"

// fakeProvider / fakeModel / fakeUsage 是替身在 assistant message_end 里
// 自报的固定值:刻意与任何合理期望配置不同,证明适配器读的是流内观察值
// 而不是回填期望旗标。
const (
	fakeProvider = "fake-provider"
	fakeModel    = "fake-model"
)

// fakeRecord 是一次 print 调用的观测面记录。
type fakeRecord struct {
	Argv            []string `json:"argv"`
	Cwd             string   `json:"cwd"`
	Mode            string   `json:"mode"`
	SessionDir      string   `json:"session_dir"`
	Provider        string   `json:"provider,omitempty"`
	Model           string   `json:"model,omitempty"`
	Thinking        string   `json:"thinking,omitempty"`
	NoSession       bool     `json:"no_session"`
	NoExtensions    bool     `json:"no_extensions"`
	NoSkills        bool     `json:"no_skills"`
	NoPromptTmpl    bool     `json:"no_prompt_templates"`
	NoThemes        bool     `json:"no_themes"`
	NoContextFiles  bool     `json:"no_context_files"`
	NoApprove       bool     `json:"no_approve"`
	Home            string   `json:"home"`
	UserProfile     string   `json:"userprofile"`
	PiAgentDir      string   `json:"pi_agent_dir"`
	PiSessionDir    string   `json:"pi_session_dir_env"`
	Offline         string   `json:"pi_offline"`
	Telemetry       string   `json:"pi_telemetry"`
	DeepseekKeySet  bool     `json:"deepseek_key_set"`
	AnthropicKeySet bool     `json:"anthropic_key_set"`
	OpenaiKeySet    bool     `json:"openai_key_set"`
	ChildPID        int      `json:"child_pid,omitempty"`
	Instruction     string   `json:"instruction"`
}

func main() {
	// 子进程模式:睡到被整树终止/超时(整树终止的取证目标)。
	if os.Getenv("TALLY_FAKE_PI_CHILD") == "1" {
		time.Sleep(10 * time.Minute)
		return
	}

	args := os.Args[1:]
	if len(args) == 0 {
		os.Exit(2)
	}
	if args[0] == "--version" || args[0] == "-v" {
		if os.Getenv("TALLY_FAKE_PI_SCENARIO") == "versionfail" {
			fmt.Fprintln(os.Stderr, "fakepi: --version refused")
			os.Exit(3)
		}
		if v := os.Getenv("TALLY_FAKE_PI_VERSION"); v != "" {
			fmt.Println(v)
			return
		}
		fmt.Println(pinnedVersion)
		return
	}

	var mode, sessionDir, provider, model, thinking string
	var instruction string
	sawSeparator := false
	flags := map[string]bool{}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case sawSeparator:
			instruction = a // 替身只支持单位置参数(适配器契约:-- 后单个 instruction)
		case a == "--":
			sawSeparator = true
		case a == "-p" || a == "--print":
			// 非交互 print 模式:替身恒按一次性进程处理。
		case a == "--mode" && i+1 < len(args):
			i++
			mode = args[i]
		case a == "--session-dir" && i+1 < len(args):
			i++
			sessionDir = args[i]
		case a == "--provider" && i+1 < len(args):
			i++
			provider = args[i]
		case a == "--model" && i+1 < len(args):
			i++
			model = args[i]
		case a == "--thinking" && i+1 < len(args):
			i++
			thinking = args[i]
		case strings.HasPrefix(a, "--no-"):
			flags[a] = true
		default:
			fmt.Fprintf(os.Stderr, "fakepi: unexpected argument %q\n", a)
			os.Exit(2)
		}
	}
	if mode == "" {
		mode = "text" // pi 帮助文本观测:text 是默认输出模式
	}

	cwd, _ := os.Getwd()
	rec := fakeRecord{
		Argv:            os.Args,
		Cwd:             cwd,
		Mode:            mode,
		SessionDir:      sessionDir,
		Provider:        provider,
		Model:           model,
		Thinking:        thinking,
		NoSession:       flags["--no-session"],
		NoExtensions:    flags["--no-extensions"],
		NoSkills:        flags["--no-skills"],
		NoPromptTmpl:    flags["--no-prompt-templates"],
		NoThemes:        flags["--no-themes"],
		NoContextFiles:  flags["--no-context-files"],
		NoApprove:       flags["--no-approve"],
		Home:            os.Getenv("HOME"),
		UserProfile:     os.Getenv("USERPROFILE"),
		PiAgentDir:      os.Getenv("PI_CODING_AGENT_DIR"),
		PiSessionDir:    os.Getenv("PI_CODING_AGENT_SESSION_DIR"),
		Offline:         os.Getenv("PI_OFFLINE"),
		Telemetry:       os.Getenv("PI_TELEMETRY"),
		DeepseekKeySet:  os.Getenv("DEEPSEEK_API_KEY") != "",
		AnthropicKeySet: os.Getenv("ANTHROPIC_API_KEY") != "",
		OpenaiKeySet:    os.Getenv("OPENAI_API_KEY") != "",
		Instruction:     instruction,
	}

	// 长任务:先 spawn 子进程(继承 stdout/stderr,模拟真实子进程树的管道
	// 占用),PID 记录后等待 —— 被整树终止时父子同灭;被单进程 kill 时子进程
	// 成为持管道孤儿(适配器 WaitDelay 兜底面)。
	if strings.Contains(instruction, longMarker) {
		exe, err := os.Executable()
		if err != nil {
			os.Exit(3)
		}
		child := exec.Command(exe)
		child.Env = append(os.Environ(), "TALLY_FAKE_PI_CHILD=1")
		child.Stdout = os.Stdout
		child.Stderr = os.Stderr
		if err := child.Start(); err != nil {
			os.Exit(3)
		}
		rec.ChildPID = child.Process.Pid
		writeRecord(rec)
		_ = child.Wait()
		return
	}

	writeRecord(rec)

	// 观测的 keyless 契约(0.85.1 实测):缺 provider key → stderr 结构化
	// 错误文本,exit 1,stdout 为空。替身经显式门控启用(生产适配器永不设置
	// 该变量);凭据剥离面测试用它证明 operator 环境的 key 被剥离。
	if os.Getenv("TALLY_FAKE_PI_REQUIRE_KEY") == "1" && os.Getenv("DEEPSEEK_API_KEY") == "" {
		fmt.Fprintln(os.Stderr, "No API key found for deepseek.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Use /login to log into a provider via OAuth or API key.")
		os.Exit(1)
	}

	switch os.Getenv("TALLY_FAKE_PI_SCENARIO") {
	case "exitfail":
		// 部分输出后崩溃:非零退出,partial 不得成为最终回答(json 模式给出
		// 一行合法 session 行 + 一行截断尾行,覆盖「截断行被跳过、无 assistant
		// message_end 即失败」的解析纪律)。
		if mode == "json" {
			fmt.Printf("{\"type\":\"session\",\"version\":3,\"id\":%q,\"cwd\":%q}\n", fakeSessionID, cwd)
			fmt.Printf("{\"type\":\"message_update\",\"assistantMessageEvent\":{\"delta\":\"partial")
		} else {
			fmt.Println("partial answer before crash")
		}
		os.Exit(7)
	case "emptyok":
		// exit 0 但无输出:契约未兑现,适配器必须 FAIL-SAFE 判败。
		os.Exit(0)
	case "leakstderr":
		// 失败输出夹带密钥形态文本:非完成态 summary 必须先脱敏再进 claim。
		fmt.Fprintln(os.Stderr, "provider error DEEPSEEK_API_KEY=sk-fake-secret-1234567890abcdef")
		os.Exit(1)
	}

	// cwd 绑定任务(观测:pi 的工具在进程 cwd 内活动):`read <file>` → 输出
	// cwd 下该文件内容,证明 workspace 钉定经进程 cwd 生效。
	if strings.HasPrefix(instruction, "read ") {
		data, err := os.ReadFile(strings.TrimSpace(strings.TrimPrefix(instruction, "read ")))
		if err != nil {
			fmt.Fprintf(os.Stderr, "fakepi: read failed: %v\n", err)
			os.Exit(1)
		}
		if mode == "json" {
			emitJSONStream(cwd, strings.TrimRight(string(data), "\n"))
		} else {
			fmt.Print(string(data))
		}
		return
	}

	if mode == "json" {
		emitJSONStream(cwd, "ok")
		return
	}
	fmt.Println("ok")
}

// emitJSONStream 输出替身的 NDJSON 事件流(观测形态的最小子集:session 行 +
// agent_start + 带文本/自报 provider/model/usage 的 assistant message_end +
// turn_end/agent_end/agent_settled)。
func emitJSONStream(cwd, text string) {
	fmt.Printf("{\"type\":\"session\",\"version\":3,\"id\":%q,\"timestamp\":\"2026-11-19T00:00:00.000Z\",\"cwd\":%q}\n", fakeSessionID, cwd)
	fmt.Println(`{"type":"agent_start"}`)
	fmt.Printf("{\"type\":\"message_end\",\"message\":{\"role\":\"assistant\",\"content\":[{\"type\":\"text\",\"text\":%s}],\"api\":\"openai-completions\",\"provider\":%q,\"model\":%q,\"usage\":{\"input\":11,\"output\":3,\"cacheRead\":0,\"cacheWrite\":0,\"totalTokens\":14},\"stopReason\":\"stop\"}}\n",
		mustJSON(text), fakeProvider, fakeModel)
	fmt.Println(`{"type":"turn_end"}`)
	fmt.Println(`{"type":"agent_end"}`)
	fmt.Println(`{"type":"agent_settled"}`)
}

// mustJSON 把字符串编码为 JSON 字符串字面量(替身输出构造用)。
func mustJSON(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return `""`
	}
	return string(b)
}

// writeRecord 把观测面追加进 TALLY_FAKE_PI_RECORD(未设置则丢弃)。
func writeRecord(rec fakeRecord) {
	path := os.Getenv("TALLY_FAKE_PI_RECORD")
	if path == "" {
		return
	}
	b, err := json.Marshal(rec)
	if err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	_, _ = f.Write(append(b, '\n'))
}
