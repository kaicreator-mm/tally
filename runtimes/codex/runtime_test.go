package codex

// CodexRuntime(P1-37)的测试策略,对应任务验收:
//
//   - 本机 codex 凭据缺失(P1-36 探针记录 §5 是部署前提,不是代码缺陷),
//     真实模型回合必然 401。因此模型无关的协议面全部经「fake codex 二进制」
//     驱动:TestMain 现场构建一个只实现探针记录 §2.1/§3 观测表面的替身 CLI,
//     适配器对它 spawn/解析/取消/分类 —— 进程边界、JSONL 协议、argv 组装、
//     崩溃与取消语义都是真实经过(不是内存 mock)。
//   - 通用 conformance(PRD §35)对同一替身运行 runtimeconformance.Run 全套
//     (能力专项按声明门控:resume/fork/output.structured 运行,
//     session.isolation 未声明故 canary 不运行 —— 探针记录 §6)。
//   - 真实二进制面由 TALLY_CODEX_LIVE=1 门控的测试覆盖:验证 401 认证门被
//     如实归类为 RunFailed claim(环境前提:无凭据;见包注释与报告)。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"tally/internal/agentruntime"
	"tally/test/runtimeconformance"
)

// fakeCodexBin 是 TestMain 构建的替身 codex CLI 路径。
var fakeCodexBin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "tally-fake-codex-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "codex test: 建临时目录失败: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(dir)
	src := filepath.Join(dir, "fake_codex_main.go")
	if werr := os.WriteFile(src, []byte(fakeCodexSource), 0o644); werr != nil {
		fmt.Fprintf(os.Stderr, "codex test: 写替身源码失败: %v\n", werr)
		os.Exit(1)
	}
	exe := filepath.Join(dir, "fakecodex.exe")
	if out, berr := exec.Command("go", "build", "-o", exe, src).CombinedOutput(); berr != nil {
		fmt.Fprintf(os.Stderr, "codex test: 构建替身 codex CLI 失败: %v\n%s\n", berr, out)
		os.Exit(1)
	}
	fakeCodexBin = exe
	os.Exit(m.Run())
}

// longMarker 是替身 CLI 识别「长运行指令」的标记:conformance 的
// LongRunInstruction 注入点(阻塞直到被 Cancel/ctx/外部 kill 终止)。
const longMarker = "tally-conformance-long"

// fakeCodexSource 是替身 codex CLI 的完整源码。它只模仿实测表面(P1-36 探针
// 记录 + P0_CODEX_TRUST §6 于 2026-09-09 对 codex-cli 0.153.0 的现场探针):
// --version、exec [OPTIONS] [PROMPT]、exec [OPTIONS] resume|fork [SESSION_ID]
// [PROMPT](旗标必须在子命令之前;resume/fork 不收 -C/-s,收到即如 clap 一样
// exit 2);JSONL 词汇表 = thread.started/turn.started/turn.completed/error/
// item.completed。TALLY_FAKE_CODEX_SCENARIO 注入正常/异常路径
// (warnok/warnauthfail/authfail/badjson/exitfail);
// TALLY_FAKE_CODEX_RECORD 逐行记录收到的调用,供测试断言 argv 组装。
const fakeCodexSource = `package main

import (
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

const versionLine = "codex-cli 0.153.0-conformance"

const longMarker = "tally-conformance-long"

type fakeRecord struct {
	Mode    string   ` + "`json:\"mode\"`" + `
	Thread  string   ` + "`json:\"thread\"`" + `
	Cwd     string   ` + "`json:\"cwd\"`" + `
	CwdFlag string   ` + "`json:\"cwd_flag\"`" + `
	Sandbox string   ` + "`json:\"sandbox\"`" + `
	Model   string   ` + "`json:\"model\"`" + `
	Schema  string   ` + "`json:\"schema\"`" + `
	Out     string   ` + "`json:\"out\"`" + `
	Prompt  string   ` + "`json:\"prompt\"`" + `
	Argv    []string ` + "`json:\"argv\"`" + `
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		os.Exit(2)
	}
	if args[0] == "--version" {
		if os.Getenv("TALLY_FAKE_CODEX_SCENARIO") == "versionfail" {
			// P0-02:--version 探针失败面(身份不可观察;exec 表面不受影响)。
			fmt.Fprintln(os.Stderr, "fake-codex: --version refused")
			os.Exit(3)
		}
		fmt.Println(versionLine)
		return
	}
	if args[0] != "exec" {
		fmt.Fprintf(os.Stderr, "fake-codex: 不支持的子命令 %q\n", args[0])
		os.Exit(2)
	}
	rest := args[1:]
	mode, threadArg := "new", ""
	var cwdFlag, sandbox, model, schemaFile, outFile string
	var prompt []string
	// 实测表面(2026-09-09 探针,P0_CODEX_TRUST §6):codex exec [OPTIONS]
	// <COMMAND> [ARGS] —— 旗标只在 exec 层(子命令之前)被全部接受;进入
	// resume/fork 子命令后,-C/--cd 与 -s/--sandbox 不在其 OPTIONS 中,如
	// clap 一样拒绝(exit 2)。替身按同一结构解析,让 argv 顺序回归有进程级
	// 复现面。
	inSub := false
	for i := 0; i < len(rest); i++ {
		tok := rest[i]
		if !inSub && (tok == "resume" || tok == "fork") {
			mode = tok
			inSub = true
			continue
		}
		if strings.HasPrefix(tok, "-") && len(tok) > 1 {
			takesValue := false
			switch tok {
			case "--json", "--skip-git-repo-check":
			case "-C", "--cd", "-s", "--sandbox", "-m", "--model", "--output-schema", "-o", "--output-last-message":
				takesValue = true
			default:
				// 实测:未知旗标被 clap 拒绝(exit 2)。
				fmt.Fprintf(os.Stderr, "error: unexpected argument '%s' found\n", tok)
				os.Exit(2)
			}
			if takesValue {
				if i+1 >= len(rest) {
					os.Exit(2)
				}
				i++
				v := rest[i]
				switch rest[i-1] {
				case "-C", "--cd":
					cwdFlag = v
				case "-s", "--sandbox":
					sandbox = v
				case "-m", "--model":
					model = v
				case "--output-schema":
					schemaFile = v
				case "-o", "--output-last-message":
					outFile = v
				}
				if inSub && (rest[i-1] == "-C" || rest[i-1] == "--cd" || rest[i-1] == "-s" || rest[i-1] == "--sandbox") {
					fmt.Fprintf(os.Stderr, "error: unexpected argument '%s' found\n\nUsage: codex exec %s [OPTIONS] [SESSION_ID] [PROMPT]\n", rest[i-1], mode)
					os.Exit(2)
				}
			}
			continue
		}
		if inSub {
			if threadArg == "" {
				threadArg = tok
			} else {
				prompt = append(prompt, tok)
			}
		} else {
			prompt = append(prompt, tok)
		}
	}
	instr := strings.Join(prompt, " ")
	recordCall(mode, threadArg, cwdFlag, sandbox, model, schemaFile, outFile, instr)

	var threadID string
	switch mode {
	case "resume":
		threadID = threadArg // resume 落回原 thread(会话身份延续)
	default:
		threadID = newThreadID() // new 与 fork 都落新 thread(fork = 派生新会话)
	}
	emit(map[string]any{"type": "thread.started", "thread_id": threadID})
	scenario := os.Getenv("TALLY_FAKE_CODEX_SCENARIO")
	if scenario == "warnok" || scenario == "warnauthfail" {
		// P0_CODEX_TRUST §6.1 实测(2026-09-09,codex-cli 0.153.0):未知模型名
		// 的非致命元数据警告以 item.completed + item.type=="error" 形态在
		// turn.started 之前到达,随后 turn 照常进行。
		emit(map[string]any{"type": "item.completed", "item": map[string]any{"id": "item_0", "type": "error", "message": "Model metadata for deepseek-v4-flash not found. Defaulting to fallback metadata; this can degrade performance and cause issues."}})
	}
	emit(map[string]any{"type": "turn.started"})

	switch scenario {
	case "authfail":
		// P1-36 探针记录 §3 的实测 401 形态:error 事件 + error item,exit 0。
		emit(map[string]any{"type": "error", "message": "Reconnecting... 1/5 (unexpected status 401 Unauthorized: Missing bearer or basic authentication in header, url: wss://api.openai.com/v1/responses)"})
		emit(map[string]any{"type": "item.completed", "item": map[string]any{"id": "item_0", "type": "error", "message": "Falling back from WebSockets to HTTPS transport. unexpected status 401 Unauthorized: Missing bearer or basic authentication in header"}})
		return
	case "warnauthfail":
		// P0_CODEX_TRUST §6.1 实测的同一流形态:警告之后跟真实 401 失败
		// (error 事件 + error item,无 last-message 产物)。真实失败必须仍判
		// 失败 —— 警告权威化只豁免「事实完成」的 turn。
		emit(map[string]any{"type": "error", "message": "Reconnecting... 1/5 (unexpected status 401 Unauthorized: Missing bearer or basic authentication in header, url: wss://api.openai.com/v1/responses)"})
		emit(map[string]any{"type": "item.completed", "item": map[string]any{"id": "item_1", "type": "error", "message": "Falling back from WebSockets to HTTPS transport. unexpected status 401 Unauthorized: Missing bearer or basic authentication in header"}})
		return
	case "warnok":
		// 警告后 turn 正常完成(实测:turn.completed + -o 产物,exit 0)。
		emit(map[string]any{"type": "turn.completed", "usage": map[string]any{"input_tokens": 10, "output_tokens": 2}})
		writeLastMessage(outFile, "ok")
		return
	case "badjson":
		fmt.Println("<<not json>>")
		writeLastMessage(outFile, "ok")
		return
	case "exitfail":
		writeLastMessage(outFile, "partial work before crash")
		os.Exit(7)
	}

	final := "ok"
	if schemaFile != "" {
		final = ` + "`" + `{"answer":"ok"}` + "`" + `
		if err := checkMinimalSchema(schemaFile, final); err != nil {
			emit(map[string]any{"type": "item.completed", "item": map[string]any{"id": "item_1", "type": "error", "message": err.Error()}})
			os.Exit(1)
		}
	}
	if strings.Contains(instr, longMarker) {
		time.Sleep(10 * time.Minute) // 由适配器 Cancel/ctx 或外部 kill 终止
		return
	}
	writeLastMessage(outFile, final)
}

func emit(v map[string]any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	fmt.Println(string(b))
}

func writeLastMessage(path, msg string) {
	if path == "" {
		return
	}
	_ = os.WriteFile(path, []byte(msg), 0o644)
}

func recordCall(mode, threadArg, cwdFlag, sandbox, model, schemaFile, outFile, instr string) {
	path := os.Getenv("TALLY_FAKE_CODEX_RECORD")
	if path == "" {
		return
	}
	cwd, _ := os.Getwd()
	b, err := json.Marshal(fakeRecord{
		Mode: mode, Thread: threadArg, Cwd: cwd, CwdFlag: cwdFlag,
		Sandbox: sandbox, Model: model, Schema: schemaFile, Out: outFile,
		Prompt: instr, Argv: os.Args,
	})
	if err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(b, '\n'))
}

// checkMinimalSchema 模仿 codex 的 --output-schema 强制行为(探针记录 §2.1:
// "JSON Schema file constraining the model's final response"):按 schema 的
// required 顶层属性校验最终消息,违约即报错(进程非零退出)。
func checkMinimalSchema(schemaPath, final string) error {
	b, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("fake-codex: 读 schema 失败: %w", err)
	}
	var schema struct {
		Required []string ` + "`json:\"required\"`" + `
	}
	if err := json.Unmarshal(b, &schema); err != nil {
		return fmt.Errorf("fake-codex: schema 不是合法 JSON: %w", err)
	}
	var msg map[string]any
	if err := json.Unmarshal([]byte(final), &msg); err != nil {
		return fmt.Errorf("fake-codex: 最终消息不是 JSON 对象")
	}
	for _, key := range schema.Required {
		if _, ok := msg[key]; !ok {
			return fmt.Errorf("fake-codex: 最终消息违反 output schema: 缺少 required 属性 %q", key)
		}
	}
	return nil
}

func newThreadID() string {
	var b [16]byte
	if _, err := crand.Read(b[:]); err != nil {
		return fmt.Sprintf("00000000-0000-0000-0000-%012d", time.Now().UnixNano()%1000000000000)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b[:])
	return fmt.Sprintf("%s-%s-%s-%s-%s", h[0:8], h[8:12], h[12:16], h[16:20], h[20:32])
}
`

// ---- 测试基础设施 ------------------------------------------------------------

// fakeRecord 是替身 CLI 记录行的 Go 侧投影。
type fakeRecord struct {
	Mode    string   `json:"mode"`
	Thread  string   `json:"thread"`
	Cwd     string   `json:"cwd"`
	CwdFlag string   `json:"cwd_flag"`
	Sandbox string   `json:"sandbox"`
	Model   string   `json:"model"`
	Schema  string   `json:"schema"`
	Out     string   `json:"out"`
	Prompt  string   `json:"prompt"`
	Argv    []string `json:"argv"`
}

// lastRecord 读取替身 CLI 记录文件的最后一条(最后一次调用)。
func lastRecord(t *testing.T, path string) fakeRecord {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取替身记录 %s: %v", path, err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[len(lines)-1]) == "" {
		t.Fatalf("替身记录为空: %s", path)
	}
	var rec fakeRecord
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &rec); err != nil {
		t.Fatalf("解析替身记录: %v", err)
	}
	return rec
}

// newConformanceRuntime 构造挂在替身 CLI 上的 CodexRuntime,记录每次调用的
// argv 供 WorkspaceEffect 深度校验。
func newConformanceRuntime(t *testing.T) (*Runtime, string) {
	t.Helper()
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	rt := New(Config{
		BinPath: fakeCodexBin,
		Env:     map[string]string{"TALLY_FAKE_CODEX_RECORD": record},
	})
	return rt, record
}

func startSession(t *testing.T, rt *Runtime, ws string) *Session {
	t.Helper()
	s, err := rt.Start(context.Background(), agentruntime.StartRequest{Workspace: ws})
	if err != nil {
		t.Fatalf("Start: unexpected error: %v", err)
	}
	c, ok := s.(*Session)
	if !ok {
		t.Fatalf("Start: want *codex.Session, got %T", s)
	}
	return c
}

// nextEvent 在超时保护下读取下一个事件;失败直接 FailNow。
func nextEvent(t *testing.T, st agentruntime.EventStream) agentruntime.Event {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ev, err := st.Next(ctx)
	if err != nil {
		t.Fatalf("EventStream.Next: unexpected error: %v", err)
	}
	return ev
}

// drainEvents 读取事件直到流关闭,返回已收到的事件。
func drainEvents(t *testing.T, st agentruntime.EventStream) []agentruntime.Event {
	t.Helper()
	var out []agentruntime.Event
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for {
		ev, err := st.Next(ctx)
		if err != nil {
			if !errors.Is(err, agentruntime.ErrEventStreamClosed) {
				t.Fatalf("EventStream.Next: expected ErrEventStreamClosed, got %v", err)
			}
			return out
		}
		out = append(out, ev)
	}
}

// waitForStatus 在超时内等待 Execute 收敛并返回 claim。
func waitForStatus(t *testing.T, outcome <-chan executeOutcome) executeOutcome {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	select {
	case out := <-outcome:
		return out
	case <-ctx.Done():
		t.Fatalf("Execute 未在时限内收敛: %v", ctx.Err())
		return executeOutcome{}
	}
}

type executeOutcome struct {
	res agentruntime.AgentRunResult
	err error
}

// ---- AC4:能力声明显式且真实(探针记录 §8 清单)------------------------------

func TestDescriptorDeclaresExactlyProbedCapabilities(t *testing.T) {
	d := New(Config{}).Descriptor()
	if err := d.Validate(); err != nil {
		t.Fatalf("Descriptor.Validate: %v", err)
	}
	if d.Name != DescriptorName {
		t.Fatalf("Name: want %q, got %q", DescriptorName, d.Name)
	}
	got := d.Capabilities.Sorted()
	want := []agentruntime.Capability{
		agentruntime.CapEventStreaming,
		agentruntime.CapModelSelect,
		agentruntime.CapOutputStructured,
		agentruntime.CapSandboxNative,
		agentruntime.CapSessionFork,
		agentruntime.CapSessionPersist,
		agentruntime.CapSessionResume,
	}
	if len(got) != len(want) {
		t.Fatalf("declared capabilities: want exactly %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("declared capabilities: want %v, got %v", want, got)
		}
	}
	// 未被探针记录支持的能力必须全部缺席(AC4:不广告未支持能力):
	//   - session.isolation:P0-03 起 per-run home 隔离是无条件生产行为
	//     (isolation.go),但声明以真实 CLI 上的 Session Isolation Canary
	//     证据为门(Config.SessionIsolationVerified,DECISION_LOCKS #22);
	//     零值配置 → 未声明;
	//   - turn.steer / turn.interrupt:exec 表面无 steer,中断=杀进程(核心 Cancel);
	//   - tool.approval:exec 无 headless 审批通道(探针记录 §7);
	//   - tools.custom:exec 无注入旗标;
	//   - memory.declared:无证据(探针记录 §7)。
	for _, absent := range []agentruntime.Capability{
		agentruntime.CapSessionIsolation,
		agentruntime.CapTurnSteer,
		agentruntime.CapTurnInterrupt,
		agentruntime.CapToolApproval,
		agentruntime.CapToolsCustom,
		agentruntime.CapMemoryDeclared,
	} {
		if d.Capabilities.Has(absent) {
			t.Fatalf("capability %s must stay undeclared by CodexRuntime v1", absent)
		}
	}
}

func TestStartEnforcesProtocolGates(t *testing.T) {
	rt := New(Config{BinPath: fakeCodexBin})
	ctx := context.Background()
	// 缺 workspace → ErrNoWorkspace(INV-EXEC-01)。
	if _, err := rt.Start(ctx, agentruntime.StartRequest{}); !errors.Is(err, agentruntime.ErrNoWorkspace) {
		t.Fatalf("Start without workspace: want ErrNoWorkspace, got %v", err)
	}
	// ResumeOf / ModelProfile 对应能力已声明 → 协商通过。
	ws := t.TempDir()
	if _, err := rt.Start(ctx, agentruntime.StartRequest{Workspace: ws, ResumeOf: "01a07368-d440-7912-b025-44811ad11414"}); err != nil {
		t.Fatalf("Start(ResumeOf): unexpected error: %v", err)
	}
	if _, err := rt.Start(ctx, agentruntime.StartRequest{Workspace: ws, ModelProfile: "economical"}); err != nil {
		t.Fatalf("Start(ModelProfile): unexpected error: %v", err)
	}
}

// ---- AC3:身份/版本/能力快照可记录 -------------------------------------------

func TestSnapshotRecordsIdentityVersionCapabilities(t *testing.T) {
	rt := New(Config{BinPath: fakeCodexBin})
	snap, err := rt.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if snap.Name != DescriptorName || snap.AdapterVersion != DescriptorVersion {
		t.Fatalf("snapshot identity: %+v", snap)
	}
	// 版本来自现场 `--version`(替身输出 "codex-cli 0.153.0-conformance"),
	// 不是硬编码常量。
	if snap.CLIVersion != "0.153.0-conformance" {
		t.Fatalf("CLIVersion: want 0.153.0-conformance, got %q", snap.CLIVersion)
	}
	if snap.BinaryPath != fakeCodexBin {
		t.Fatalf("BinaryPath: want %q, got %q", fakeCodexBin, snap.BinaryPath)
	}
	if len(snap.Capabilities) != 7 {
		t.Fatalf("snapshot capabilities: want 7, got %v", snap.Capabilities)
	}
	if rt.Descriptor().Version != DescriptorVersion {
		t.Fatalf("Descriptor.Version 应为适配器版本 %q, got %q", DescriptorVersion, rt.Descriptor().Version)
	}
}

func TestParseCLIVersion(t *testing.T) {
	// 探针记录 §1 的实测输出形态。
	got := parseCLIIdentity("codex-cli 0.153.0\n")
	if got.Name != "codex-cli" || got.Version != "0.153.0" {
		t.Fatalf("parseCLIIdentity: want codex-cli/0.153.0, got %+v", got)
	}
	// 非实测形态:不发明标识;版本回退整行原文(与 parseCLIVersion 时代一致)。
	if got := parseCLIIdentity("something else"); got.Name != "" || got.Version != "something else" {
		t.Fatalf("parseCLIIdentity fallback: got %+v", got)
	}
}

// ---- 二进制发现(探针记录 §1)-------------------------------------------------

func TestResolveBinaryPathEnvPrecedence(t *testing.T) {
	t.Setenv(envCodexCLIPath, fakeCodexBin)
	got, err := ResolveBinaryPath()
	if err != nil {
		t.Fatalf("ResolveBinaryPath: %v", err)
	}
	if got != fakeCodexBin {
		t.Fatalf("env CODEX_CLI_PATH 应优先且直接采用: want %q, got %q", fakeCodexBin, got)
	}
}

func TestResolveBinaryPathFromConfigToml(t *testing.T) {
	// 观测形态(P1-36 §1):桌面 App 在 [mcp_servers.<name>.env] 内以字面量
	// 字符串发布 CODEX_CLI_PATH;无关节/无关键必须被忽略。
	home := t.TempDir()
	t.Setenv("CODEX_HOME", home)
	t.Setenv(envCodexCLIPath, "")
	cfg := "[mcp_servers]\n\n" +
		"[mcp_servers.node_repl]\n" +
		"type = \"stdio\"\n" +
		"command = 'C:\\App\\node_repl.exe'\n" +
		"\n" +
		"[mcp_servers.node_repl.env]\n" +
		"BROWSER_USE_CODEX_APP_VERSION = \"26.901.22334\"\n" +
		"CODEX_CLI_PATH = '" + fakeCodexBin + "' # trailing comment\n" +
		"CODEX_HOME = 'C:\\Users\\x\\.codex'\n"
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveBinaryPath()
	if err != nil {
		t.Fatalf("ResolveBinaryPath: %v", err)
	}
	if got != fakeCodexBin {
		t.Fatalf("config.toml 发现: want %q, got %q", fakeCodexBin, got)
	}
}

func TestResolveBinaryPathConfigWinsWithExplicitError(t *testing.T) {
	// config.toml 指向已消失的哈希路径(App 更新后的观测风险,探针记录 §1):
	// 显式报错,静默回落到 PATH 上的错误二进制更危险。
	home := t.TempDir()
	t.Setenv("CODEX_HOME", home)
	t.Setenv(envCodexCLIPath, "")
	missing := filepath.Join(home, "no-such-hash-dir", "codex.exe")
	cfg := "[mcp_servers.node_repl.env]\nCODEX_CLI_PATH = '" + missing + "'\n"
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ResolveBinaryPath()
	if err == nil {
		t.Fatal("config 指向缺失路径应显式报错")
	}
	if !strings.Contains(err.Error(), missing) {
		t.Fatalf("错误应指明缺失路径 %q: %v", missing, err)
	}
}

func TestStartFailsExplicitlyWithoutBinary(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CODEX_HOME", home)
	t.Setenv(envCodexCLIPath, "")
	// PATH 上找不到 codex 且无 config/env 线索 → Start 显式失败。
	if _, err := exec.LookPath("codex"); err == nil {
		t.Skip("本机 PATH 上存在 codex,无法构造「完全不可发现」场景")
	}
	rt := New(Config{})
	_, err := rt.Start(context.Background(), agentruntime.StartRequest{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("二进制不可发现时 Start 应报错")
	}
}

// ---- AC1:Start/execute/cwd/events/result(经替身 CLI 的真实进程面)-----------

func TestExecuteHappyPathClaimEventsAndResult(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	rt := New(Config{
		BinPath: fakeCodexBin,
		Env:     map[string]string{"TALLY_FAKE_CODEX_RECORD": record},
	})
	ws := t.TempDir()
	s := startSession(t, rt, ws)
	ctx := context.Background()

	st, err := s.Events(ctx)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}
	res, err := s.Execute(ctx, agentruntime.ExecuteRequest{Instruction: "add a test"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.Status != agentruntime.RunCompleted {
		t.Fatalf("Status: want completed, got %s summary=%q", res.Status, res.Summary)
	}
	if res.RunID == "" || res.SessionID == "" {
		t.Fatalf("claim 缺少 RunID/SessionID: %+v", res)
	}
	if res.Summary != "ok" {
		t.Fatalf("Summary 应来自 -o last-message 产物: got %q", res.Summary)
	}
	// Usage 不解析:探针记录未给出 exec 表面用量事件形态,解析即发明
	// (INV-AUTH-01:宁缺毋假)。
	if res.Usage != (agentruntime.Usage{}) {
		t.Fatalf("Usage 应为零值(unavailable),got %+v", res.Usage)
	}

	snap, err := s.Result(ctx, res.RunID)
	if err != nil || snap != res {
		t.Fatalf("Result: want %+v, got %+v err=%v", res, snap, err)
	}
	if _, err := s.Result(ctx, "run-bogus"); !errors.Is(err, agentruntime.ErrRunNotFound) {
		t.Fatalf("Result(unknown): want ErrRunNotFound, got %v", err)
	}

	events := drainEvents(t, st)
	wantKinds := []agentruntime.EventKind{
		agentruntime.EventRunStarted,
		agentruntime.EventAgentMessage,
		agentruntime.EventRunCompleted,
	}
	if len(events) != len(wantKinds) {
		t.Fatalf("event count: want %d, got %d: %+v", len(wantKinds), len(events), events)
	}
	var prevSeq int64
	for i, ev := range events {
		if ev.Kind != wantKinds[i] {
			t.Fatalf("event[%d]: want %s, got %s", i, wantKinds[i], ev.Kind)
		}
		if ev.SessionID != res.SessionID || ev.RunID != res.RunID {
			t.Fatalf("event[%d] 归属错误: %+v", i, ev)
		}
		if i > 0 && ev.Seq <= prevSeq {
			t.Fatalf("event[%d]: Seq 不单调: %d <= %d", i, ev.Seq, prevSeq)
		}
		prevSeq = ev.Seq
	}
	if events[0].Seq != 1 {
		t.Fatalf("新会话首事件 Seq 应为 1, got %d", events[0].Seq)
	}
	if events[1].Text != "ok" {
		t.Fatalf("agent.message text: want ok, got %q", events[1].Text)
	}
}

func TestExecutePassesProbedFlagsViaStructuredArgv(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	rt := New(Config{
		BinPath: fakeCodexBin,
		Env:     map[string]string{"TALLY_FAKE_CODEX_RECORD": record},
	})
	ws := t.TempDir()
	s := startSession(t, rt, ws)
	ctx := context.Background()

	res, err := s.Execute(ctx, agentruntime.ExecuteRequest{Instruction: "do the thing"})
	if err != nil || res.Status != agentruntime.RunCompleted {
		t.Fatalf("Execute: res=%+v err=%v", res, err)
	}
	rec := lastRecord(t, record)
	if rec.Mode != "new" {
		t.Fatalf("Mode: want new, got %q", rec.Mode)
	}
	// 探针记录 §2.1/§8:--json JSONL、-C 钉 workspace、结构化 argv。
	// 审批旗标:P1-37 现场复核实测真实 codex exec 拒绝 -a/--ask-for-approval
	// (与探针记录 §2.1 抄录不符,以实测为准),适配器不传任何审批旗标。
	for _, flag := range []string{"--json", "--skip-git-repo-check"} {
		if !argvHas(rec.Argv, flag) {
			t.Fatalf("argv 缺少 %s: %v", flag, rec.Argv)
		}
	}
	if argvHas(rec.Argv, "-a") || argvHas(rec.Argv, "--ask-for-approval") {
		t.Fatalf("真实表面已拒绝审批旗标,argv 不应携带: %v", rec.Argv)
	}
	if !argvHasPair(rec.Argv, "-C", ws) {
		t.Fatalf("argv 缺少 -C <%s>: %v", ws, rec.Argv)
	}
	if !argvHasPair(rec.Argv, "-s", "workspace-write") {
		t.Fatalf("argv 缺少 -s workspace-write(默认沙箱): %v", rec.Argv)
	}
	if argvHas(rec.Argv, "-m") {
		t.Fatalf("未请求模型档位时不应传 -m: %v", rec.Argv)
	}
	if rec.Cwd != ws || rec.CwdFlag != ws {
		t.Fatalf("进程 cwd 与 -C 都应是 workspace: cwd=%q cwdFlag=%q ws=%q", rec.Cwd, rec.CwdFlag, ws)
	}
	if rec.Prompt != "do the thing" {
		t.Fatalf("instruction 应原样作为位置参数传入: got %q", rec.Prompt)
	}
	if rec.Out == "" {
		t.Fatalf("argv 缺少 -o last-message 产物路径: %v", rec.Argv)
	}
	if rec.Schema != "" {
		t.Fatalf("未设置 schema 时不应传 --output-schema: %v", rec.Argv)
	}
}

func argvHas(argv []string, flag string) bool {
	for _, a := range argv {
		if a == flag {
			return true
		}
	}
	return false
}

// TestBuildExecSpecPlacesFlagsBeforeSubcommand 锁定实测 CLI 表面(P0_CODEX_TRUST
// §6.2,2026-09-09 对 codex-cli 0.153.0 的现场探针):
//   - `codex exec resume <id> --json ... -C ...`(旧顺序)被 clap 拒绝(exit 2,
//     "error: unexpected argument '-C' found")—— resume/fork 子命令的 OPTIONS
//     不含 -C/-s;
//   - 用法面 `codex exec [OPTIONS] <COMMAND> [ARGS]` 只在 exec 层(子命令之前)
//     接受全部旗标;实测 `exec --json --skip-git-repo-check -C <ws> -s read-only
//     -m <model> -o <file> resume <uuid> <prompt>` 与 fork 同形 argv 均通过 clap
//     进入运行时层(报 "no rollout found",exit 1 —— 非 clap 的 exit 2)。
//
// instruction 恒为最后一个位置参数(子命令形态下是子命令的 [PROMPT])。
func TestBuildExecSpecPlacesFlagsBeforeSubcommand(t *testing.T) {
	ws := t.TempDir()
	s := &Session{
		runtime:   New(Config{}),
		bin:       "codex-fake-bin",
		sandbox:   "read-only",
		model:     "deepseek-v4-flash",
		workspace: ws,
	}
	lastMsg := filepath.Join(ws, "last-message.txt")

	// resume:全部旗标在 resume 子命令之前,<id> 紧随其后,instruction 最后。
	spec, err := s.buildExecSpec(turnMode{sub: "resume", threadID: "01a0826a-aa7d-7f00-8000-000000000000"}, "second turn", "", lastMsg)
	if err != nil {
		t.Fatalf("buildExecSpec(resume): %v", err)
	}
	wantResume := []string{
		"codex-fake-bin", "exec",
		"--json", "--skip-git-repo-check",
		"-C", ws, "-s", "read-only",
		"-m", "deepseek-v4-flash",
		"-o", lastMsg,
		"resume", "01a0826a-aa7d-7f00-8000-000000000000",
		"second turn",
	}
	if !reflect.DeepEqual(spec.Argv, wantResume) {
		t.Fatalf("resume argv 顺序错误(旗标必须在子命令之前):\n got %v\nwant %v", spec.Argv, wantResume)
	}

	// fork:同形(同一 buildExecSpec 代码路径,同一实测表面)。
	specFork, err := s.buildExecSpec(turnMode{sub: "fork", threadID: "01a0826a-aa7d-7f00-8000-000000000000"}, "derived turn", "", lastMsg)
	if err != nil {
		t.Fatalf("buildExecSpec(fork): %v", err)
	}
	wantFork := []string{
		"codex-fake-bin", "exec",
		"--json", "--skip-git-repo-check",
		"-C", ws, "-s", "read-only",
		"-m", "deepseek-v4-flash",
		"-o", lastMsg,
		"fork", "01a0826a-aa7d-7f00-8000-000000000000",
		"derived turn",
	}
	if !reflect.DeepEqual(specFork.Argv, wantFork) {
		t.Fatalf("fork argv 顺序错误:\n got %v\nwant %v", specFork.Argv, wantFork)
	}

	// 新 thread(无子命令):旗标顺序与既有行为一致,instruction 最后。
	sNoModel := &Session{
		runtime:   New(Config{}),
		bin:       "codex-fake-bin",
		sandbox:   "workspace-write",
		workspace: ws,
	}
	specNew, err := sNoModel.buildExecSpec(turnMode{}, "first turn", "", lastMsg)
	if err != nil {
		t.Fatalf("buildExecSpec(new): %v", err)
	}
	wantNew := []string{
		"codex-fake-bin", "exec",
		"--json", "--skip-git-repo-check",
		"-C", ws, "-s", "workspace-write",
		"-o", lastMsg,
		"first turn",
	}
	if !reflect.DeepEqual(specNew.Argv, wantNew) {
		t.Fatalf("new-thread argv 顺序错误:\n got %v\nwant %v", specNew.Argv, wantNew)
	}
}

func argvHasPair(argv []string, flag, value string) bool {
	for i := 0; i+1 < len(argv); i++ {
		if argv[i] == flag && argv[i+1] == value {
			return true
		}
	}
	return false
}

func TestResumeUsesExecResumeWithSameThreadID(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	rt := New(Config{
		BinPath: fakeCodexBin,
		Env:     map[string]string{"TALLY_FAKE_CODEX_RECORD": record},
	})
	ws := t.TempDir()
	ctx := context.Background()

	s1 := startSession(t, rt, ws)
	r1, err := s1.Execute(ctx, agentruntime.ExecuteRequest{Instruction: "first"})
	if err != nil || r1.Status != agentruntime.RunCompleted {
		t.Fatalf("first turn: %+v err=%v", r1, err)
	}
	if err := s1.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}

	s2, err := rt.Start(ctx, agentruntime.StartRequest{Workspace: ws, ResumeOf: r1.SessionID})
	if err != nil {
		t.Fatalf("Start(ResumeOf) 已声明 session.resume,应成功: %v", err)
	}
	c2 := s2.(*Session)
	r2, err := s2.Execute(ctx, agentruntime.ExecuteRequest{Instruction: "second"})
	if err != nil {
		t.Fatalf("resume turn: %v", err)
	}
	if r2.SessionID != r1.SessionID {
		t.Fatalf("resume 应续用同一会话: %s vs %s", r1.SessionID, r2.SessionID)
	}
	rec := lastRecord(t, record)
	if rec.Mode != "resume" || rec.Thread != string(r1.SessionID) {
		t.Fatalf("应执行 `exec resume <threadID>`: mode=%q thread=%q", rec.Mode, rec.Thread)
	}
	_ = c2
}

func TestForkDerivesNewSessionID(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	rt := New(Config{
		BinPath: fakeCodexBin,
		Env:     map[string]string{"TALLY_FAKE_CODEX_RECORD": record},
	})
	ws := t.TempDir()
	ctx := context.Background()

	parent := startSession(t, rt, ws)
	pres, err := parent.Execute(ctx, agentruntime.ExecuteRequest{Instruction: "parent"})
	if err != nil || pres.Status != agentruntime.RunCompleted {
		t.Fatalf("parent turn: %+v err=%v", pres, err)
	}

	child, err := rt.Fork(ctx, parent)
	if err != nil {
		t.Fatalf("Fork: %v", err)
	}
	cres, err := child.Execute(ctx, agentruntime.ExecuteRequest{Instruction: "child"})
	if err != nil || cres.Status != agentruntime.RunCompleted {
		t.Fatalf("child turn: %+v err=%v", cres, err)
	}
	if cres.SessionID == pres.SessionID {
		t.Fatalf("fork 应产生新会话: 子会话与父会话同为 %s", pres.SessionID)
	}
	rec := lastRecord(t, record)
	if rec.Mode != "fork" || rec.Thread != string(pres.SessionID) {
		t.Fatalf("应执行 `exec fork <parentThreadID>`: mode=%q thread=%q", rec.Mode, rec.Thread)
	}
	// fork 只作用于首回合:子会话第二个 turn 应转为 resume。
	if _, err := child.Execute(ctx, agentruntime.ExecuteRequest{Instruction: "child2"}); err != nil {
		t.Fatalf("child turn 2: %v", err)
	}
	rec = lastRecord(t, record)
	if rec.Mode != "resume" {
		t.Fatalf("fork 后续 turn 应为 resume: got %q", rec.Mode)
	}
	// 父会话派生后仍可用(PRD §35: parent still usable)。
	if _, err := parent.Execute(ctx, agentruntime.ExecuteRequest{Instruction: "parent2"}); err != nil {
		t.Fatalf("parent turn 2: %v", err)
	}
}

func TestModelProfileSelectsModelFlag(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	rt := New(Config{
		BinPath: fakeCodexBin,
		Env:     map[string]string{"TALLY_FAKE_CODEX_RECORD": record},
	})
	s := startSessionModel(t, rt, t.TempDir(), "economical")
	if _, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "x"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	rec := lastRecord(t, record)
	if !argvHasPair(rec.Argv, "-m", "economical") {
		t.Fatalf("argv 缺少 -m economical: %v", rec.Argv)
	}
}

func startSessionModel(t *testing.T, rt *Runtime, ws, profile string) *Session {
	t.Helper()
	s, err := rt.Start(context.Background(), agentruntime.StartRequest{Workspace: ws, ModelProfile: profile})
	if err != nil {
		t.Fatalf("Start(ModelProfile): %v", err)
	}
	return s.(*Session)
}

// ---- 失败/崩溃/取消(AC2,PRD §35)---------------------------------------------

func TestAuthFailureSurfacesAsFailedClaimNotError(t *testing.T) {
	// P1-36 探针记录 §3:401 后 exec 以 exit 0 + error item 收场。适配器必须
	// 把它归类为 RunFailed claim(失败证据),既不是成功也不是 Execute error。
	rt := New(Config{
		BinPath: fakeCodexBin,
		Env:     map[string]string{"TALLY_FAKE_CODEX_SCENARIO": "authfail"},
	})
	s := startSession(t, rt, t.TempDir())
	res, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "reply ok"})
	if err != nil {
		t.Fatalf("认证失败是 run 级失败,不是 Execute error: %v", err)
	}
	if res.Status != agentruntime.RunFailed {
		t.Fatalf("Status: want failed, got %s", res.Status)
	}
	if !strings.Contains(res.Summary, "401") {
		t.Fatalf("summary 应携带 error 事件原文(含 401): %q", res.Summary)
	}
	// 失败不产生 agent.message(失败文本属于 run.failed summary)。
	if _, err := s.Result(context.Background(), "run-bogus"); !errors.Is(err, agentruntime.ErrRunNotFound) {
		t.Fatalf("失败后会话应仍可响应: %v", err)
	}
}

func TestNonFatalMetadataWarningDoesNotFailCompletedTurn(t *testing.T) {
	// P0_CODEX_TRUST §6.1(实测 codex-cli 0.153.0,2026-09-09):未知模型名
	// (custom provider 模型如 deepseek-v4-flash)在每个 turn 起始发非致命
	// item.type=="error" 元数据警告("Model metadata ... not found. Defaulting
	// to fallback metadata ..."),随后 turn 正常完成(exit 0 + -o last-message
	// 产物)。完成证据(进程成功退出 + 产出最终消息)权威于该警告:claim 必须
	// 是 completed 且 summary 来自 last-message 产物,不得误报 RunFailed。
	rt := New(Config{
		BinPath: fakeCodexBin,
		Env:     map[string]string{"TALLY_FAKE_CODEX_SCENARIO": "warnok"},
	})
	s := startSession(t, rt, t.TempDir())
	res, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "reply ok"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.Status != agentruntime.RunCompleted {
		t.Fatalf("非致命元数据警告不得把事实完成的 turn 判为失败: %s summary=%q", res.Status, res.Summary)
	}
	if res.Summary != "ok" {
		t.Fatalf("Summary 应来自 -o last-message 产物: got %q", res.Summary)
	}
}

func TestMetadataWarningDoesNotMaskGenuineFailure(t *testing.T) {
	// FAIL-SAFE 伴随测试(P0_CODEX_TRUST §6.1 实测同一流形态):元数据警告之后
	// 跟着真实的 401 失败(error 事件 + error item,无 last-message 产物)时,
	// 必须仍判 RunFailed,summary 携带真实失败原文(而非警告文本)。
	// 实测判据:turn 没有产出最终消息(无 -o 产物)时,流内 error 事件仍是
	// 权威失败信号 —— 401 两代实测形态(exit 0 + error item;exit 1 +
	// turn.failed)在此分支全部如实失败。
	rt := New(Config{
		BinPath: fakeCodexBin,
		Env:     map[string]string{"TALLY_FAKE_CODEX_SCENARIO": "warnauthfail"},
	})
	s := startSession(t, rt, t.TempDir())
	res, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "reply ok"})
	if err != nil {
		t.Fatalf("真实失败是 RunFailed claim,不是 Execute error: %v", err)
	}
	if res.Status != agentruntime.RunFailed {
		t.Fatalf("Status: want failed, got %s summary=%q", res.Status, res.Summary)
	}
	if !strings.Contains(res.Summary, "401") {
		t.Fatalf("summary 应携带真实失败原文(401)而非警告文本: %q", res.Summary)
	}
}

func TestMalformedJSONLIsExplicitFailure(t *testing.T) {
	// 探针记录 §8 前提:JSONL 解析错误是显式错误,不是可忽略噪音。
	rt := New(Config{
		BinPath: fakeCodexBin,
		Env:     map[string]string{"TALLY_FAKE_CODEX_SCENARIO": "badjson"},
	})
	s := startSession(t, rt, t.TempDir())
	res, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "x"})
	if err != nil {
		t.Fatalf("解析失败是 RunFailed claim,不是 Execute error: %v", err)
	}
	if res.Status != agentruntime.RunFailed || !strings.Contains(res.Summary, "JSONL") {
		t.Fatalf("want failed + JSONL 解析说明, got %s/%q", res.Status, res.Summary)
	}
}

func TestNonZeroExitSurfacesStderrAsFailedClaim(t *testing.T) {
	rt := New(Config{
		BinPath: fakeCodexBin,
		Env:     map[string]string{"TALLY_FAKE_CODEX_SCENARIO": "exitfail"},
	})
	s := startSession(t, rt, t.TempDir())
	res, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "x"})
	if err != nil {
		t.Fatalf("非零退出是 RunFailed claim: %v", err)
	}
	if res.Status != agentruntime.RunFailed || !strings.Contains(res.Summary, "exit status 7") {
		t.Fatalf("want failed + 退出码, got %s/%q", res.Status, res.Summary)
	}
}

func TestCancelKillsInFlightRun(t *testing.T) {
	rt := New(Config{BinPath: fakeCodexBin})
	s := startSession(t, rt, t.TempDir())
	ctx := context.Background()
	st, err := s.Events(ctx)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	outcome := make(chan executeOutcome, 1)
	go func() {
		res, err := s.Execute(ctx, agentruntime.ExecuteRequest{Instruction: longMarker + " run"})
		outcome <- executeOutcome{res, err}
	}()

	started := nextEvent(t, st)
	if started.Kind != agentruntime.EventRunStarted {
		t.Fatalf("首事件应为 run.started, got %s", started.Kind)
	}
	if err := s.Cancel(ctx, started.RunID); err != nil {
		t.Fatalf("Cancel(in-flight): %v", err)
	}

	got := waitForStatus(t, outcome)
	if got.err != nil {
		t.Fatalf("被 Cancel 的回合是正常终态: %v", got.err)
	}
	if got.res.Status != agentruntime.RunCancelled {
		t.Fatalf("Status: want cancelled, got %s", got.res.Status)
	}
	snap, err := s.Result(ctx, started.RunID)
	if err != nil || snap != got.res {
		t.Fatalf("Result after cancel: want %+v, got %+v err=%v", got.res, snap, err)
	}
	// 幂等 + 未知 run 哨兵。
	if err := s.Cancel(ctx, started.RunID); err != nil {
		t.Fatalf("Cancel(terminal run) 应幂等成功: %v", err)
	}
	if err := s.Cancel(ctx, "run-bogus"); !errors.Is(err, agentruntime.ErrRunNotFound) {
		t.Fatalf("Cancel(unknown): want ErrRunNotFound, got %v", err)
	}
	events := drainEvents(t, st)
	if len(events) != 1 || events[0].Kind != agentruntime.EventRunCancelled {
		t.Fatalf("剩余事件应为 [run.cancelled]: %+v", events)
	}
}

func TestExecuteContextCancelIsRunCancelled(t *testing.T) {
	rt := New(Config{BinPath: fakeCodexBin})
	s := startSession(t, rt, t.TempDir())
	runCtx, cancel := context.WithCancel(context.Background())

	outcome := make(chan executeOutcome, 1)
	go func() {
		res, err := s.Execute(runCtx, agentruntime.ExecuteRequest{Instruction: longMarker + " run"})
		outcome <- executeOutcome{res, err}
	}()
	cancel()
	got := waitForStatus(t, outcome)
	if got.err != nil || got.res.Status != agentruntime.RunCancelled {
		t.Fatalf("ctx 取消应为 RunCancelled 正常终态: %+v err=%v", got.res, got.err)
	}
}

func TestProcessCrashHandledAndSessionResponsive(t *testing.T) {
	// PRD §35 crash 场景的注入面:外部 kill(非 Cancel)必须归为 RunFailed,
	// 且会话不变成僵尸。外部 kill = 直接打 RunProcess 拿到的进程句柄。
	rt := New(Config{BinPath: fakeCodexBin})
	s := startSession(t, rt, t.TempDir())
	ctx := context.Background()
	st, err := s.Events(ctx)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	outcome := make(chan executeOutcome, 1)
	go func() {
		res, err := s.Execute(ctx, agentruntime.ExecuteRequest{Instruction: longMarker + " run"})
		outcome <- executeOutcome{res, err}
	}()

	started := nextEvent(t, st)
	proc, ok := s.RunProcess(started.RunID)
	if !ok {
		t.Fatalf("in-flight run 应可经 RunProcess 观测到底层进程")
	}
	if err := proc.Kill(); err != nil {
		t.Fatalf("外部 kill: %v", err)
	}

	got := waitForStatus(t, outcome)
	if got.err != nil {
		t.Fatalf("崩溃应以 RunFailed claim 上报,不是挂死: %v", got.err)
	}
	if got.res.Status != agentruntime.RunFailed {
		t.Fatalf("Status: want failed(外部 kill 非 Cancel), got %s summary=%q", got.res.Status, got.res.Summary)
	}
	// 崩溃后协议哨兵仍响应(会话未成僵尸)。
	if _, err := s.Result(ctx, "run-bogus"); !errors.Is(err, agentruntime.ErrRunNotFound) {
		t.Fatalf("崩溃后 Result(unknown): want ErrRunNotFound, got %v", err)
	}
	drainEvents(t, st) // 排干并确认流以终态收尾
}

func TestCloseKillsInFlightAndRejectsFurtherOps(t *testing.T) {
	rt := New(Config{BinPath: fakeCodexBin})
	s := startSession(t, rt, t.TempDir())
	ctx := context.Background()
	st, err := s.Events(ctx)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}
	outcome := make(chan executeOutcome, 1)
	go func() {
		res, err := s.Execute(ctx, agentruntime.ExecuteRequest{Instruction: longMarker + " run"})
		outcome <- executeOutcome{res, err}
	}()
	if ev := nextEvent(t, st); ev.Kind != agentruntime.EventRunStarted {
		t.Fatalf("首事件应为 run.started, got %s", ev.Kind)
	}
	if err := s.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}
	got := waitForStatus(t, outcome)
	if got.err != nil || got.res.Status != agentruntime.RunCancelled {
		t.Fatalf("Close 应以 RunCancelled 收尾 in-flight run: %+v err=%v", got.res, got.err)
	}
	if _, err := s.Execute(ctx, agentruntime.ExecuteRequest{Instruction: "x"}); !errors.Is(err, agentruntime.ErrSessionClosed) {
		t.Fatalf("Execute after Close: want ErrSessionClosed, got %v", err)
	}
	if _, err := s.Result(ctx, got.res.RunID); !errors.Is(err, agentruntime.ErrSessionClosed) {
		t.Fatalf("Result after Close: want ErrSessionClosed, got %v", err)
	}
	if err := s.Close(ctx); err != nil {
		t.Fatalf("Close 应幂等: %v", err)
	}
}

// ---- output.structured(--output-schema 旗标落地 + 违约可见)-------------------

func TestSetOutputSchemaRejectsInvalidJSON(t *testing.T) {
	rt := New(Config{BinPath: fakeCodexBin})
	s := startSession(t, rt, t.TempDir())
	if err := s.SetOutputSchema("{not json"); err == nil {
		t.Fatal("非法 JSON schema 应被拒绝")
	}
}

func TestStructuredOutputEnforcedThroughAdapter(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	rt := New(Config{
		BinPath: fakeCodexBin,
		Env:     map[string]string{"TALLY_FAKE_CODEX_RECORD": record},
	})
	s := startSession(t, rt, t.TempDir())
	ctx := context.Background()

	// 正例:schema 覆盖替身 CLI 的最终消息 → 完成,summary 是符合 schema 的 JSON。
	if err := s.SetOutputSchema(`{"required":["answer"]}`); err != nil {
		t.Fatalf("SetOutputSchema: %v", err)
	}
	res, err := s.Execute(ctx, agentruntime.ExecuteRequest{Instruction: "x"})
	if err != nil || res.Status != agentruntime.RunCompleted {
		t.Fatalf("符合 schema 的回合应完成: %+v err=%v", res, err)
	}
	var parsed map[string]any
	if jerr := json.Unmarshal([]byte(res.Summary), &parsed); jerr != nil || parsed["answer"] != "ok" {
		t.Fatalf("summary 应为符合 schema 的 JSON: %q", res.Summary)
	}
	rec := lastRecord(t, record)
	if rec.Schema == "" || !argvHasPair(rec.Argv, "--output-schema", rec.Schema) {
		t.Fatalf("argv 应携带 --output-schema <file>: %v", rec.Argv)
	}

	// 反例:schema 要求替身消息不具备的属性 → CLI 违约退出非零 → RunFailed。
	// (schema 本体由 codex CLI 强制 —— 探针记录 §2.1 的旗标语义;本测试证明
	// 适配器把违约如实暴露为失败 claim,而不是当成成功。)
	if err := s.SetOutputSchema(`{"required":["status"]}`); err != nil {
		t.Fatalf("SetOutputSchema: %v", err)
	}
	res2, err := s.Execute(ctx, agentruntime.ExecuteRequest{Instruction: "x"})
	if err != nil {
		t.Fatalf("schema 违约是 RunFailed claim: %v", err)
	}
	if res2.Status != agentruntime.RunFailed {
		t.Fatalf("schema 违约应被判失败: %+v", res2)
	}
}

// ---- PRD §35 通用 conformance(全场景,能力专项按声明门控)----------------------

// TestCodexRuntimeConformance 对挂在替身 CLI 上的 CodexRuntime 跑完整
// runtimeconformance 套件: start / cwd / execute / result / events / cancel /
// crash / workspace / usage / capability_truth,以及已声明能力专项
// cap:session.resume / cap:session.fork / cap:output.structured。
// session.isolation 未声明 → canary 场景按门控不运行(探针记录 §6)。
//
// 覆盖边界(如实声明):模型无关的协议面全部真实经过进程边界;真实 codex 的
// 模型回合受凭据门控(探针记录 §5),由 TALLY_CODEX_LIVE 门控的测试另行覆盖。
func TestCodexRuntimeConformance(t *testing.T) {
	rt, record := newConformanceRuntime(t)
	rep := runtimeconformance.Run(t, rt, runtimeconformance.Options{
		Probes: runtimeconformance.Probes{
			LongRunInstruction: longMarker + " run",
			CrashRun: func(ctx context.Context, sess agentruntime.Session, longRunInstruction string) (agentruntime.AgentRunResult, error) {
				cs, ok := sess.(*Session)
				if !ok {
					return agentruntime.AgentRunResult{}, fmt.Errorf("crash probe: want *codex.Session, got %T", sess)
				}
				st, err := sess.Events(ctx)
				if err != nil {
					return agentruntime.AgentRunResult{}, err
				}
				defer func() { _ = st.Close() }()
				outcome := make(chan executeOutcome, 1)
				go func() {
					res, err := sess.Execute(ctx, agentruntime.ExecuteRequest{Instruction: longRunInstruction})
					outcome <- executeOutcome{res, err}
				}()
				ev, err := st.Next(ctx)
				if err != nil {
					return agentruntime.AgentRunResult{}, fmt.Errorf("等待 run.started: %w", err)
				}
				if ev.Kind != agentruntime.EventRunStarted {
					return agentruntime.AgentRunResult{}, fmt.Errorf("首事件应为 run.started, got %s", ev.Kind)
				}
				proc, ok := cs.RunProcess(ev.RunID)
				if !ok {
					return agentruntime.AgentRunResult{}, fmt.Errorf("in-flight run 无底层进程可注入崩溃")
				}
				if err := proc.Kill(); err != nil {
					return agentruntime.AgentRunResult{}, fmt.Errorf("注入进程崩溃: %w", err)
				}
				select {
				case out := <-outcome:
					return out.res, out.err
				case <-ctx.Done():
					return agentruntime.AgentRunResult{}, ctx.Err()
				}
			},
			Fork: func(ctx context.Context, parent agentruntime.Session) (agentruntime.Session, error) {
				rt := New(Config{BinPath: fakeCodexBin})
				return rt.Fork(ctx, parent)
			},
			StructuredOutput: func(ctx context.Context, sess agentruntime.Session) error {
				cs, ok := sess.(*Session)
				if !ok {
					return fmt.Errorf("structured probe: want *codex.Session, got %T", sess)
				}
				if err := cs.SetOutputSchema(`{"required":["answer"]}`); err != nil {
					return err
				}
				res, err := sess.Execute(ctx, agentruntime.ExecuteRequest{Instruction: "conformance: 请原样回复 ok"})
				if err != nil {
					return err
				}
				if res.Status != agentruntime.RunCompleted {
					return fmt.Errorf("符合 schema 的回合应完成: %s/%q", res.Status, res.Summary)
				}
				var parsed map[string]any
				if jerr := json.Unmarshal([]byte(res.Summary), &parsed); jerr != nil {
					return fmt.Errorf("最终消息不是 JSON: %q (%v)", res.Summary, jerr)
				}
				if _, ok := parsed["answer"]; !ok {
					return fmt.Errorf("最终消息缺少 schema required 属性 answer: %q", res.Summary)
				}
				return nil
			},
			WorkspaceEffect: func(ctx context.Context, sess agentruntime.Session) error {
				cs, ok := sess.(*Session)
				if !ok {
					return fmt.Errorf("workspace probe: want *codex.Session, got %T", sess)
				}
				rec := lastRecordFor(record)
				if rec.CwdFlag != cs.Workspace() || rec.Cwd != cs.Workspace() {
					return fmt.Errorf("替身 CLI 观测到的执行目录 (-C=%q, cwd=%q) 不是提供的 workspace %q", rec.CwdFlag, rec.Cwd, cs.Workspace())
				}
				return nil
			},
		},
	})
	if !rep.OK() {
		t.Fatalf("CodexRuntime conformance 未通过:\n%s", rep.FailureText())
	}
	// 能力专项必须真的跑过(声明即验证,PRD §35):
	for _, scenario := range []string{
		"cap:session.resume", "cap:session.fork", "cap:output.structured",
	} {
		if !rep.Ran(scenario) {
			t.Fatalf("场景 %s 未运行(声明了对应能力,必须验证)", scenario)
		}
	}
	// 未声明 session.isolation → canary 不运行(声明真实性,探针记录 §6)。
	if rep.Ran("cap:session.isolation") {
		t.Fatal("未声明 session.isolation,canary 场景不应运行")
	}
}

// lastRecordFor 是 lastRecord 的非 Fatal 版(供 harness 探针闭包使用)。
func lastRecordFor(path string) fakeRecord {
	data, err := os.ReadFile(path)
	if err != nil {
		return fakeRecord{}
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 {
		return fakeRecord{}
	}
	var rec fakeRecord
	_ = json.Unmarshal([]byte(lines[len(lines)-1]), &rec)
	return rec
}

// ---- 真实 codex 表面(凭据门控,环境前提)--------------------------------------

// TestLiveCodexSurfaceAuthGate 在 TALLY_CODEX_LIVE=1 时对真实 codex CLI 执行
// 一个最小回合,验证适配器在真实表面上行为正确:
//   - 凭据可用(部署前提已满足):回合完成,claim 为 RunCompleted;
//   - 凭据缺失(本机现状,P1-36 探针记录 §5):401 被如实归类为 RunFailed
//     claim,summary 携带认证错误原文 —— 不伪装成功,不挂死,不误报 adapter 崩溃。
//
// 环境变量 CODEX_HOME 被重定向到临时目录:不污染真实 CODEX_HOME,等价于
// 「无凭据」前提。二进制不可发现时跳过(非 NOT_RUN:本测试自身声明了前提)。
func TestLiveCodexSurfaceAuthGate(t *testing.T) {
	if os.Getenv("TALLY_CODEX_LIVE") == "" {
		t.Skip("TALLY_CODEX_LIVE 未设置:跳过真实 codex 表面(本机凭据缺失,见 P1-36 探针记录 §5)")
	}
	bin, err := ResolveBinaryPath()
	if err != nil {
		t.Skipf("真实 codex CLI 不可发现: %v", err)
	}
	home := t.TempDir()
	rt := New(Config{
		BinPath: bin,
		Env:     map[string]string{"CODEX_HOME": home},
	})
	ws := t.TempDir()
	s, err := rt.Start(context.Background(), agentruntime.StartRequest{Workspace: ws})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	res, err := s.Execute(ctx, agentruntime.ExecuteRequest{Instruction: "conformance: 请原样回复 ok"})
	if err != nil {
		t.Fatalf("真实表面 Execute 不应返回 error(失败走 claim): %v", err)
	}
	switch res.Status {
	case agentruntime.RunCompleted:
		t.Logf("凭据可用:真实回合完成 summary=%q session=%s", res.Summary, res.SessionID)
	case agentruntime.RunFailed:
		// 探针记录 §3 实测 401 文案:"unexpected status 401 Unauthorized:
		// Missing bearer or basic authentication in header"。标记刻意收窄,
		// 避免误匹配(如路径中的 "Auth" 字样)。
		lower := strings.ToLower(res.Summary)
		for _, marker := range []string{"401", "unauthorized", "not logged in", "no credential", "api key", "missing bearer"} {
			if strings.Contains(lower, marker) {
				t.Logf("环境前提成立(无凭据):失败被如实归类为 RunFailed claim,summary=%q", res.Summary)
				return
			}
		}
		t.Fatalf("真实回合失败但 summary 不含认证错误标记: %q", res.Summary)
	default:
		t.Fatalf("真实回合意外终态: %+v", res)
	}
}

// Failure diagnostics cross a durable claim boundary. Secrets emitted by an
// external CLI must be redacted before they can enter AgentRunResult/log/UI.
func TestFailureSummaryRedactsSecrets(t *testing.T) {
	r0 := &run{id: "run-redact", status: agentruntime.RunRunning}
	s := &Session{}
	secret := "ghp_abcdefghijklmnopqrstuvwxyz1234567890"
	res := s.finishRun(r0, agentruntime.RunFailed, "provider failed token="+secret, false)
	if strings.Contains(res.Summary, secret) {
		t.Fatalf("failure summary leaked secret: %q", res.Summary)
	}
	if !strings.Contains(res.Summary, "[REDACTED]") {
		t.Fatalf("failure summary should contain redaction marker: %q", res.Summary)
	}
}
