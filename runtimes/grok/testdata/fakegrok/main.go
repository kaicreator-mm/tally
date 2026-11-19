// Command fakegrok 是 Grok 适配器测试的替身 grok CLI(仅标准库,由
// runtimes/grok 的 TestMain 按 go build 文件模式现场构建)。
//
// 它只模仿对官方 @xai-official/grok 1.0.87(原生二进制,commit
// 929e6dd187b1)观测到的表面,不发明任何旗标:
//
//   - `--version` → 首行 "grok 1.0.87 (0000000fakegrok)"(观测形态:
//     "grok 1.0.87 (929e6dd187b1)",exit 0);
//   - `--no-auto-update [-m <model>] [--permission-mode <mode>]
//     --single=<prompt>` → 一次性进程:stdout = 最终回答(plain 默认);
//     exit 0 = completed / 非 0 = 未完成;cwd = 进程 cwd(workspaceRoot =
//     process.cwd(),inspect 观测);缺 XAI_API_KEY → stderr
//     「Not signed in. ...」、exit 1(观测的 keyless 契约原文);
//   - 长任务(含 tally-conformance-long 标记)→ CLI 再 spawn 一个子进程
//     并等待(整树终止取证面):子进程 PID 记录进 record 文件;子进程继承
//     替身的 stdout/stderr,等待适配器整树终止或 WaitDelay 兜底。
//
// TALLY_FAKE_GROK_SCENARIO 注入异常路径(exitfail / emptyok / versionfail);
// TALLY_FAKE_GROK_VERSION 覆盖 --version 输出(钉版闸负例);
// TALLY_FAKE_GROK_RECORD 逐行 JSON 记录观测面(argv/cwd/GROK_HOME/认证 env
// 存在性/model/permission-mode/子进程 PID),供测试断言。记录面仅测试注入,
// 生产 runtime 永不设置这些变量。记录只含 env 存在性布尔,绝不含 env 值。
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// pinnedVersion 是替身自报的钉住版本(与 runtimes/grok.PinnedGrokVersion 一致)。
const pinnedVersion = "1.0.87"

// longMarker 是「长运行指令」标记(conformance 的 LongRunInstruction 注入点)。
const longMarker = "tally-conformance-long"

// fakeRecord 是一次 headless 调用的观测面记录(只含存在性,不含任何 env 值)。
type fakeRecord struct {
	Argv           []string `json:"argv"`
	Cwd            string   `json:"cwd"`
	GrokHome       string   `json:"grok_home"`
	XAIAPIKeySet   bool     `json:"xai_api_key_set"`
	DeepSeekKeySet bool     `json:"deepseek_key_set"`
	Model          string   `json:"model,omitempty"`
	PermissionMode string   `json:"permission_mode,omitempty"`
	ChildPID       int      `json:"child_pid,omitempty"`
	Prompt         string   `json:"prompt"`
}

func main() {
	// 子进程模式:睡到被整树终止/超时(整树终止的取证目标)。
	if os.Getenv("TALLY_FAKE_GROK_CHILD") == "1" {
		time.Sleep(10 * time.Minute)
		return
	}

	args := os.Args[1:]
	if len(args) == 0 {
		os.Exit(2)
	}
	if args[0] == "--version" {
		if v := os.Getenv("TALLY_FAKE_GROK_VERSION"); v != "" {
			fmt.Println(v)
			return
		}
		if os.Getenv("TALLY_FAKE_GROK_SCENARIO") == "versionfail" {
			fmt.Fprintln(os.Stderr, "fakegrok: --version refused")
			os.Exit(3)
		}
		fmt.Printf("grok %s (0000000fakegrok)\n", pinnedVersion)
		return
	}

	// 观测的调用形态:[bin] --no-auto-update [-m <m>] [--permission-mode <p>]
	// --single=<prompt>。替身按该形态解析,未知形态按 usage 错误拒绝
	// (观测:未知旗标 exit 2)。
	if args[0] != "--no-auto-update" {
		fmt.Fprintf(os.Stderr, "fakegrok: unexpected argument %q\n", args[0])
		os.Exit(2)
	}
	rest := args[1:]
	var model, permission string
	var prompt string
	for i := 0; i < len(rest); i++ {
		switch {
		case rest[i] == "-m":
			if i+1 >= len(rest) {
				os.Exit(2)
			}
			i++
			model = rest[i]
		case rest[i] == "--permission-mode":
			if i+1 >= len(rest) {
				os.Exit(2)
			}
			i++
			permission = rest[i]
		case strings.HasPrefix(rest[i], "--single="):
			prompt = strings.TrimPrefix(rest[i], "--single=")
		default:
			fmt.Fprintf(os.Stderr, "fakegrok: unexpected argument %q\n", rest[i])
			os.Exit(2)
		}
	}
	if prompt == "" {
		fmt.Fprintln(os.Stderr, "fakegrok: missing --single=<prompt>")
		os.Exit(2)
	}

	cwd, _ := os.Getwd()
	rec := fakeRecord{
		Argv:           os.Args,
		Cwd:            cwd,
		GrokHome:       os.Getenv("GROK_HOME"),
		XAIAPIKeySet:   os.Getenv("XAI_API_KEY") != "",
		DeepSeekKeySet: os.Getenv("DEEPSEEK_API_KEY") != "",
		Model:          model,
		PermissionMode: permission,
		Prompt:         prompt,
	}

	// 长任务:先 spawn 子进程(继承 stdout/stderr,模拟真实 CLI 子进程树的
	// 管道占用),PID 记录后等待 —— 被整树终止时父子同灭;被单进程 kill 时
	// 子进程成为持管道孤儿(适配器 WaitDelay 兜底面)。
	if strings.Contains(prompt, longMarker) {
		exe, err := os.Executable()
		if err != nil {
			os.Exit(3)
		}
		child := exec.Command(exe)
		child.Env = append(os.Environ(), "TALLY_FAKE_GROK_CHILD=1")
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

	// 真实契约(观测的 keyless 形态):缺 XAI_API_KEY → stderr「Not signed
	// in. ...」,exit 1,stdout 为空。
	if os.Getenv("XAI_API_KEY") == "" {
		fmt.Fprintln(os.Stderr, "Not signed in. To authenticate without a browser, run:\n  grok login --device-code\n\nAlternatively, set the XAI_API_KEY environment variable or run `grok login` on a machine with a browser.")
		fmt.Fprintln(os.Stderr, "Error: Not signed in. To authenticate without a browser, run:\n  grok login --device-code\n\nAlternatively, set the XAI_API_KEY environment variable or run `grok login` on a machine with a browser.")
		os.Exit(1)
	}

	switch os.Getenv("TALLY_FAKE_GROK_SCENARIO") {
	case "exitfail":
		// 部分回答后崩溃:非零退出,partial 文本不得成为最终回答。
		fmt.Println("partial answer before crash")
		os.Exit(7)
	case "emptyok":
		// exit 0 但无最终回答:契约未兑现,适配器必须 FAIL-SAFE 判败。
		os.Exit(0)
	}

	// cwd 绑定任务(inspect 观测的形态):`read <file>` → 输出 cwd 下
	// 该文件内容,证明 workspace 钉定经进程 cwd 生效。
	if strings.HasPrefix(prompt, "read ") {
		data, err := os.ReadFile(strings.TrimSpace(strings.TrimPrefix(prompt, "read ")))
		if err != nil {
			fmt.Fprintf(os.Stderr, "fakegrok: read failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(string(data))
		return
	}

	fmt.Println("ok")
}

// writeRecord 把观测面追加进 TALLY_FAKE_GROK_RECORD(未设置则丢弃)。
func writeRecord(rec fakeRecord) {
	path := os.Getenv("TALLY_FAKE_GROK_RECORD")
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
