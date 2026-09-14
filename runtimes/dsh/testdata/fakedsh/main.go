// Command fakedsh 是 DSH 适配器测试的替身 dsh launcher(仅标准库,由
// runtimes/dsh 的 TestMain 按 go build 文件模式现场构建)。
//
// 它只模仿 P4-03 spike(docs/tasks/v1.4.1/spikes/P4-03_dsh_protocol_spike.md)
// 对 @deepseek-ai/dsh@0.1.5-rc.1 观测到的表面,不发明任何旗标:
//
//   - `--version` → 裸版本行 "0.1.5-rc.1"(spike §3 探针 1 的观测形态);
//   - `--profile headless [--patch <path>] <task>` → 一次性进程:
//     stdout = 最终回答(纯文本);exit 0 = completed / 非 0 = 未完成;
//     cwd = 进程 cwd(workspaceRoot = process.cwd(),spike §2.2);
//     缺 DEEPSEEK_API_KEY → stderr MISSING_CREDENTIAL、exit 1(spike §2.6);
//   - 长任务(含 tally-conformance-long 标记)→ launcher 再 spawn 一个子进程
//     并等待(spike R-1 取证面):验证适配器整树终止 —— 子进程 PID 记录进
//     record 文件;子进程继承 launcher 的 stdout/stderr(真实 Node 子进程的
//     管道继承形态,等待适配器 WaitDelay 兜底)。
//
// TALLY_FAKE_DSH_SCENARIO 注入异常路径(exitfail / emptyok / versionfail);
// TALLY_FAKE_DSH_VERSION 覆盖 --version 输出(钉版闸负例);
// TALLY_FAKE_DSH_RECORD 逐行 JSON 记录观测面(argv/cwd/隔离与遥测环境/
// patch 内容/子进程 PID),供测试断言。记录面仅测试注入,生产 runtime 永不
// 设置这些变量。
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// pinnedVersion 是替身自报的钉住版本(与 runtimes/dsh.PinnedDSHVersion 一致)。
const pinnedVersion = "0.1.5-rc.1"

// longMarker 是「长运行指令」标记(conformance 的 LongRunInstruction 注入点)。
const longMarker = "tally-conformance-long"

// fakeRecord 是一次 headless 调用的观测面记录。
type fakeRecord struct {
	Argv         []string `json:"argv"`
	Cwd          string   `json:"cwd"`
	Home         string   `json:"dsh_home"`
	Telemetry    string   `json:"telemetry_mode"`
	Permission   string   `json:"permission_mode"`
	APIKeySet    bool     `json:"api_key_set"`
	PatchPath    string   `json:"patch_path,omitempty"`
	PatchContent string   `json:"patch_content,omitempty"`
	ChildPID     int      `json:"child_pid,omitempty"`
	Task         string   `json:"task"`
}

func main() {
	// 子进程模式:睡到被整树终止/超时(整树终止的取证目标)。
	if os.Getenv("TALLY_FAKE_DSH_CHILD") == "1" {
		time.Sleep(10 * time.Minute)
		return
	}

	args := os.Args[1:]
	if len(args) == 0 {
		os.Exit(2)
	}
	if args[0] == "--version" {
		if v := os.Getenv("TALLY_FAKE_DSH_VERSION"); v != "" {
			fmt.Println(v)
			return
		}
		if os.Getenv("TALLY_FAKE_DSH_SCENARIO") == "versionfail" {
			fmt.Fprintln(os.Stderr, "fakedsh: --version refused")
			os.Exit(3)
		}
		fmt.Println(pinnedVersion)
		return
	}
	// 观测表面只有 --profile <name>(spike §2.1);未知形态按拒绝处理。
	if args[0] != "--profile" || len(args) < 2 {
		fmt.Fprintf(os.Stderr, "fakedsh: unexpected argument %q\n", args[0])
		os.Exit(2)
	}
	if args[1] != "headless" {
		fmt.Fprintf(os.Stderr, "fakedsh: unsupported profile %q\n", args[1])
		os.Exit(2)
	}

	rest := args[2:]
	var patchPath string
	var task []string
	for i := 0; i < len(rest); i++ {
		if rest[i] == "--patch" {
			if i+1 >= len(rest) {
				os.Exit(2)
			}
			i++
			patchPath = rest[i]
			continue
		}
		task = append(task, rest[i])
	}
	instr := strings.Join(task, " ")

	cwd, _ := os.Getwd()
	rec := fakeRecord{
		Argv:       os.Args,
		Cwd:        cwd,
		Home:       os.Getenv("DSH_HOME"),
		Telemetry:  os.Getenv("DSH_TELEMETRY_MODE"),
		Permission: os.Getenv("DSH_PERMISSION_MODE"),
		APIKeySet:  os.Getenv("DEEPSEEK_API_KEY") != "",
		PatchPath:  patchPath,
		Task:       instr,
	}
	if patchPath != "" {
		if b, err := os.ReadFile(patchPath); err == nil {
			rec.PatchContent = string(b)
		}
	}

	// 长任务:先 spawn 子进程(继承 stdout/stderr,模拟真实 Node 子进程树的
	// 管道占用),PID 记录后等待 —— 被整树终止时父子同灭;被单进程 kill 时
	// 子进程成为持管道孤儿(适配器 WaitDelay 兜底面)。
	if strings.Contains(instr, longMarker) {
		exe, err := os.Executable()
		if err != nil {
			os.Exit(3)
		}
		child := exec.Command(exe)
		child.Env = append(os.Environ(), "TALLY_FAKE_DSH_CHILD=1")
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

	// 真实契约(spike §2.6):缺 DEEPSEEK_API_KEY → 结构化 MISSING_CREDENTIAL,
	// exit 1,stdout 为空。
	if os.Getenv("DEEPSEEK_API_KEY") == "" {
		fmt.Fprintln(os.Stderr, `dsh: MISSING_CREDENTIAL: llm-deepseek: no API key for provider route "deepseek-official"; store DEEPSEEK_API_KEY through the credentials service (the web Models page writes it), or export DEEPSEEK_API_KEY in the launching environment`)
		os.Exit(1)
	}

	switch os.Getenv("TALLY_FAKE_DSH_SCENARIO") {
	case "exitfail":
		// 部分回答后崩溃:非零退出,partial 文本不得成为最终回答。
		fmt.Println("partial answer before crash")
		os.Exit(7)
	case "emptyok":
		// exit 0 但无最终回答:契约未兑现,适配器必须 FAIL-SAFE 判败。
		os.Exit(0)
	}

	// cwd 绑定任务(spike §3 探针 6 的形态):`read <file>` → 输出 cwd 下
	// 该文件内容,证明 workspace 钉定经进程 cwd 生效。
	if strings.HasPrefix(instr, "read ") {
		data, err := os.ReadFile(strings.TrimSpace(strings.TrimPrefix(instr, "read ")))
		if err != nil {
			fmt.Fprintf(os.Stderr, "fakedsh: read failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(string(data))
		return
	}

	fmt.Println("ok")
}

// writeRecord 把观测面追加进 TALLY_FAKE_DSH_RECORD(未设置则丢弃)。
func writeRecord(rec fakeRecord) {
	path := os.Getenv("TALLY_FAKE_DSH_RECORD")
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
