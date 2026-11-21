// Command fakeopencode 是 OpenCode 适配器测试的替身 opencode CLI(仅标准库,
// 由 runtimes/opencode 的 TestMain 按 go build 文件模式现场构建)。
//
// 它只模仿对 opencode-ai@1.19.6 观测到的表面,不发明任何旗标:
//
//   - `--version` → 裸版本行 "1.19.6"(观测形态;exit 0);
//   - `run --format json --pure [--model <p/m>] [--agent <a>] <message>` →
//     一次性进程:stdout = 逐行 JSON 事件流(step_start / text /
//     step_finish,观测 envelope 形态);exit 0 = completed / 非 0 = 未完成;
//     cwd = 进程 cwd(workspace 经 spawn cwd 钉定);配置/数据目录完全跟随
//     XDG_CONFIG_HOME / XDG_DATA_HOME / XDG_CACHE_HOME / XDG_STATE_HOME
//     (isolation 观测);无凭据不必然失败(观测:CLI 落免凭据默认模型,
//     本替身不实现凭据缺失失败分支);
//   - 长任务(含 tally-conformance-long 标记)→ 替身再 spawn 一个子进程
//     并等待(整树终止取证面):子进程 PID 记录进 record 文件;子进程继承
//     替身的 stdout/stderr(持管道孤儿形态,等待适配器 WaitDelay 兜底)。
//
// TALLY_FAKE_OPENCODE_SCENARIO 注入异常路径(exitfail / emptyok / plainok /
// versionfail);TALLY_FAKE_OPENCODE_VERSION 覆盖 --version 输出(钉版闸
// 负例);TALLY_FAKE_OPENCODE_RECORD 逐行 JSON 记录观测面(argv/cwd/XDG 与
// temp 环境/凭据存在性/model/agent/message/子进程 PID),供测试断言。记录面
// 仅测试注入,生产 runtime 永不设置这些变量。
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// pinnedVersion 是替身自报的钉住版本(与 runtimes/opencode.PinnedOpenCodeVersion 一致)。
const pinnedVersion = "1.19.6"

// longMarker 是「长运行指令」标记(conformance 的 LongRunInstruction 注入点)。
const longMarker = "tally-conformance-long"

// fakeRecord 是一次 run 调用的观测面记录。
type fakeRecord struct {
	Argv      []string `json:"argv"`
	Cwd       string   `json:"cwd"`
	PWD       string   `json:"pwd_env"`
	OLDPWD    string   `json:"oldpwd_env"`
	XDGConfig string   `json:"xdg_config"`
	XDGData   string   `json:"xdg_data"`
	XDGCache  string   `json:"xdg_cache"`
	XDGState  string   `json:"xdg_state"`
	TMP       string   `json:"tmp"`
	TEMP      string   `json:"temp"`
	APIKeySet bool     `json:"api_key_set"`
	Model     string   `json:"model,omitempty"`
	Agent     string   `json:"agent,omitempty"`
	Message   string   `json:"message"`
	ChildPID  int      `json:"child_pid,omitempty"`
}

// emitEvent 按观测 envelope 形态输出一行 JSON 事件。
func emitEvent(envType, partType, text string, tokens *[2]int64) {
	ts := time.Now().UnixMilli()
	part := map[string]any{"type": partType, "id": "prt_fake", "sessionID": "ses_fake", "messageID": "msg_fake"}
	if text != "" {
		part["text"] = text
	}
	if tokens != nil {
		part["reason"] = "stop"
		part["tokens"] = map[string]any{"total": tokens[0] + tokens[1], "input": tokens[0], "output": tokens[1], "reasoning": 0, "cache": map[string]int{"write": 0, "read": 0}}
		part["cost"] = 0.0001
	}
	b, err := json.Marshal(map[string]any{"type": envType, "timestamp": ts, "sessionID": "ses_fake", "part": part})
	if err != nil {
		return
	}
	fmt.Println(string(b))
}

func main() {
	// 子进程模式:睡到被整树终止/超时(整树终止的取证目标)。
	if os.Getenv("TALLY_FAKE_OPENCODE_CHILD") == "1" {
		time.Sleep(10 * time.Minute)
		return
	}

	args := os.Args[1:]
	if len(args) == 0 {
		os.Exit(2)
	}
	if args[0] == "--version" {
		if v := os.Getenv("TALLY_FAKE_OPENCODE_VERSION"); v != "" {
			fmt.Println(v)
			return
		}
		if os.Getenv("TALLY_FAKE_OPENCODE_SCENARIO") == "versionfail" {
			fmt.Fprintln(os.Stderr, "fakeopencode: --version refused")
			os.Exit(3)
		}
		fmt.Println(pinnedVersion)
		return
	}
	// 观测表面只有 run 子命令;未知形态按拒绝处理。
	if args[0] != "run" {
		fmt.Fprintf(os.Stderr, "fakeopencode: unexpected argument %q\n", args[0])
		os.Exit(2)
	}

	rest := args[1:]
	var model, agent string
	var msg []string
	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case "--format":
			if i+1 >= len(rest) {
				os.Exit(2)
			}
			i++
			if rest[i] != "json" {
				fmt.Fprintf(os.Stderr, "fakeopencode: unsupported format %q\n", rest[i])
				os.Exit(2)
			}
		case "--pure":
			// 布尔旗标,无值(观测形态)。
		case "--model":
			if i+1 >= len(rest) {
				os.Exit(2)
			}
			i++
			model = rest[i]
		case "--agent":
			if i+1 >= len(rest) {
				os.Exit(2)
			}
			i++
			agent = rest[i]
		default:
			msg = append(msg, rest[i])
		}
	}
	instr := strings.Join(msg, " ")

	cwd, _ := os.Getwd()
	rec := fakeRecord{
		Argv:      os.Args,
		Cwd:       cwd,
		PWD:       os.Getenv("PWD"),
		OLDPWD:    os.Getenv("OLDPWD"),
		XDGConfig: os.Getenv("XDG_CONFIG_HOME"),
		XDGData:   os.Getenv("XDG_DATA_HOME"),
		XDGCache:  os.Getenv("XDG_CACHE_HOME"),
		XDGState:  os.Getenv("XDG_STATE_HOME"),
		TMP:       os.Getenv("TMP"),
		TEMP:      os.Getenv("TEMP"),
		APIKeySet: os.Getenv("DEEPSEEK_API_KEY") != "",
		Model:     model,
		Agent:     agent,
		Message:   instr,
	}

	// 长任务:先 spawn 子进程(继承 stdout/stderr,模拟真实子进程树的管道
	// 占用),PID 记录后等待 —— 被整树终止时父子同灭;被单进程 kill 时子
	// 进程成为持管道孤儿(适配器 WaitDelay 兜底面)。
	if strings.Contains(instr, longMarker) {
		exe, err := os.Executable()
		if err != nil {
			os.Exit(3)
		}
		child := exec.Command(exe)
		child.Env = append(os.Environ(), "TALLY_FAKE_OPENCODE_CHILD=1")
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

	switch os.Getenv("TALLY_FAKE_OPENCODE_SCENARIO") {
	case "exitfail":
		// 部分事件后崩溃:非零退出,部分事件中的 text 不得成为最终回答。
		emitEvent("step_start", "step-start", "", nil)
		emitEvent("text", "text", "partial answer before crash", nil)
		fmt.Fprintln(os.Stderr, `{"name":"UnknownError","data":{"message":"Unexpected server error. Check server logs for details.","ref":"err_fake"}}`)
		os.Exit(7)
	case "emptyok":
		// exit 0 但无任何输出:契约未兑现,适配器必须 FAIL-SAFE 判败。
		os.Exit(0)
	case "plainok":
		// exit 0 但 stdout 是裸文本(非 JSON 事件流):解析失败降级原文,
		// 不得 panic、不得判败。
		fmt.Println("plain-ok")
		return
	}

	// cwd 绑定任务(`read <file>` → 输出 cwd 下该文件内容作为回答文本,
	// 证明 workspace 钉定经进程 cwd 生效)。
	answer := "ok"
	if strings.HasPrefix(instr, "read ") {
		data, err := os.ReadFile(strings.TrimSpace(strings.TrimPrefix(instr, "read ")))
		if err != nil {
			fmt.Fprintf(os.Stderr, "fakeopencode: read failed: %v\n", err)
			os.Exit(1)
		}
		answer = string(data)
	}

	// happy path:观测的 NDJSON 事件形态(step_start → text → step_finish,
	// tokens 携带 input/output)。
	tokens := [2]int64{38, 2}
	emitEvent("step_start", "step-start", "", nil)
	emitEvent("text", "text", answer, nil)
	emitEvent("step_finish", "step-finish", "", &tokens)
}

// writeRecord 把观测面追加进 TALLY_FAKE_OPENCODE_RECORD(未设置则丢弃)。
func writeRecord(rec fakeRecord) {
	path := os.Getenv("TALLY_FAKE_OPENCODE_RECORD")
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
