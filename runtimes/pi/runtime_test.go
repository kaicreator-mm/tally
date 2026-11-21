package pi

// pi 适配器的测试策略(对齐 runtimes/dsh、runtimes/codex 的替身 CLI 模式):
//
//   - 全部进程面经「替身 pi CLI」驱动:TestMain 从 testdata/fakepi/main.go
//     现场构建替身二进制,适配器对它 spawn/读取/取消/整树终止 —— 进程边界、
//     print 契约(json NDJSON 流 / text 最终回答、退出码、keyless)、argv
//     组装、隔离与强制环境、整树 teardown 都是真实经过(不是内存 mock)。
//   - 通用 conformance(PRD §35)对同一替身运行 runtimeconformance.Run 全套:
//     基础声明为空集 → 全部 cap:* 场景按门控不运行,events/capability_truth
//     走「未声明即拒绝」方向。
//   - 真实 pi 表面由 TALLY_PI_LIVE=1 门控(凭据/CLI 部署前提);本机 0.85.1
//     观测证据见包注释,真跑收据由部署方另行留档。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"tally/internal/agentruntime"
	"tally/test/runtimeconformance"
)

// fakePIBin 是 TestMain 构建的替身 pi CLI 路径。
var fakePIBin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "tally-fake-pi-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "pi test: 建临时目录失败: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(dir)
	src, err := filepath.Abs(filepath.Join("testdata", "fakepi", "main.go"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "pi test: 解析替身源码路径失败: %v\n", err)
		os.Exit(1)
	}
	exe := filepath.Join(dir, "fakepi.exe")
	if out, berr := exec.Command("go", "build", "-o", exe, src).CombinedOutput(); berr != nil {
		fmt.Fprintf(os.Stderr, "pi test: 构建替身 pi CLI 失败: %v\n%s\n", berr, out)
		os.Exit(1)
	}
	fakePIBin = exe
	os.Exit(m.Run())
}

// ---- 测试基础设施 ------------------------------------------------------------

// fakeSessionID / fakeModel 是替身在 json 流里固定自报的会话 UUID 与模型名
// (必须与 testdata/fakepi/main.go 的常量一致;替身是独立 main 包,常量在此
// 镜像)。刻意与任何合理期望配置不同,用于证明观察值来自流而非期望旗标。
const (
	fakeSessionID = "f0000000-0000-4000-8000-000000000001"
	fakeModel     = "fake-model"
)

// longMarker 是替身 CLI 识别「长运行指令」的标记:conformance 的
// LongRunInstruction 注入点(替身 spawn 子进程并等待,阻塞直到被 Cancel/
// ctx/外部 kill 整树终止;真实 pi 不识别它 —— 它只是普通指令文本)。
const longMarker = "tally-conformance-long"

// fakeRecord 是替身 CLI 记录行的 Go 侧投影。
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
	PiSessionDirEnv string   `json:"pi_session_dir_env"`
	Offline         string   `json:"pi_offline"`
	Telemetry       string   `json:"pi_telemetry"`
	DeepseekKeySet  bool     `json:"deepseek_key_set"`
	AnthropicKeySet bool     `json:"anthropic_key_set"`
	OpenaiKeySet    bool     `json:"openai_key_set"`
	ChildPID        int      `json:"child_pid,omitempty"`
	Instruction     string   `json:"instruction"`
}

// lastRecord 读取替身记录文件的最后一条(最后一次调用)。
func lastRecord(t *testing.T, path string) fakeRecord {
	t.Helper()
	lines := readRecordLines(t, path)
	var rec fakeRecord
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &rec); err != nil {
		t.Fatalf("解析替身记录: %v", err)
	}
	return rec
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

// readRecordLines 读取非空的替身记录行(容忍子进程写入中的半行:轮询面用)。
func readRecordLines(t *testing.T, path string) []string {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		data, err := os.ReadFile(path)
		if err == nil {
			var lines []string
			for _, l := range strings.Split(string(data), "\n") {
				if strings.TrimSpace(l) != "" {
					lines = append(lines, l)
				}
			}
			if len(lines) > 0 {
				// 最后一行必须是完整 JSON(未写完则继续等)。
				var probe map[string]any
				if json.Unmarshal([]byte(lines[len(lines)-1]), &probe) == nil {
					return lines
				}
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("替身记录 %s 未在时限内产生完整行", path)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// waitForChildPID 轮询记录文件直到替身 CLI 记下子进程 PID(长任务已进入
// 「spawn 完子进程并等待」阶段,整树终止的取证窗口已打开)。
func waitForChildPID(t *testing.T, path string) fakeRecord {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		lines := readRecordLines(t, path)
		var rec fakeRecord
		if json.Unmarshal([]byte(lines[len(lines)-1]), &rec) == nil && rec.ChildPID > 0 {
			return rec
		}
		if time.Now().After(deadline) {
			t.Fatalf("替身记录未在时限内记录子进程 PID: %s", path)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// processAlive 报告 pid 是否仍是我们的替身 CLI 进程。Windows 的 pid 在进程
// 死后会被内核快速复用:单纯 FindProcess 成功可能只是复用了该 pid 的别的
// 进程,因此 Windows 侧用 tasklist 按 PID 过滤并核对镜像名 —— 复用即视为
// 「已消失」。
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	if runtime.GOOS != "windows" {
		p, err := os.FindProcess(pid)
		if err != nil {
			return false
		}
		return p.Signal(syscall.Signal(0)) == nil
	}
	out, err := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV", "/NH").Output() //nolint:gosec // 固定工具名 + 数字 pid,无 shell
	if err != nil {
		// 无匹配时部分系统以非零退出并输出本地化 INFO 行:按「已消失」处理。
		return false
	}
	line := strings.TrimSpace(string(out))
	if line == "" {
		return false
	}
	fields := strings.Split(line, ",")
	if len(fields) == 0 {
		return false
	}
	return strings.EqualFold(strings.Trim(fields[0], "\" "), "fakepi.exe")
}

// waitForGone 轮询直到进程消失(整树终止收敛)或超时。
func waitForGone(t *testing.T, pid int) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for processAlive(pid) {
		if time.Now().After(deadline) {
			t.Fatalf("子进程 %d 在整树终止后仍未消失(存在孤儿残留)", pid)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// testEnv 返回带替身凭据门与记录面的基础注入环境。
func testEnv(record string) map[string]string {
	env := map[string]string{"DEEPSEEK_API_KEY": "test-key"}
	if record != "" {
		env["TALLY_FAKE_PI_RECORD"] = record
	}
	return env
}

// startSession 构造挂在替身 CLI 上的会话(缺省配置 + 注入环境,workspace 为
// 新建临时目录)。
func startSession(t *testing.T, extraEnv map[string]string, cfgMod func(*Config)) (*Runtime, *Session) {
	t.Helper()
	return startSessionIn(t, t.TempDir(), extraEnv, cfgMod)
}

// startSessionIn 是 startSession 的显式 workspace 变体(cwd 效果级验证用)。
func startSessionIn(t *testing.T, ws string, extraEnv map[string]string, cfgMod func(*Config)) (*Runtime, *Session) {
	t.Helper()
	cfg := Config{BinPath: fakePIBin, Env: testEnv("")}
	for k, v := range extraEnv {
		cfg.Env[k] = v
	}
	if cfgMod != nil {
		cfgMod(&cfg)
	}
	rt := New(cfg)
	s, err := rt.Start(context.Background(), agentruntime.StartRequest{Workspace: ws})
	if err != nil {
		t.Fatalf("Start: unexpected error: %v", err)
	}
	c, ok := s.(*Session)
	if !ok {
		t.Fatalf("Start: want *pi.Session, got %T", s)
	}
	t.Cleanup(func() { _ = s.Close(context.Background()) })
	return rt, c
}

type executeOutcome struct {
	res agentruntime.AgentRunResult
	err error
}

// waitForStatus 在超时内等待 Execute 收敛并返回 claim。
func waitForStatus(t *testing.T, outcome <-chan executeOutcome) executeOutcome {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	select {
	case out := <-outcome:
		return out
	case <-ctx.Done():
		t.Fatalf("Execute 未在时限内收敛: %v", ctx.Err())
		return executeOutcome{}
	}
}

// ---- AC:能力声明诚实(空基础集;session.isolation 证据门)--------------------

func TestDescriptorDeclaresOnlyHonestCapabilities(t *testing.T) {
	d := New(Config{BinPath: fakePIBin}).Descriptor()
	if err := d.Validate(); err != nil {
		t.Fatalf("Descriptor.Validate: %v", err)
	}
	if d.Name != DescriptorName {
		t.Fatalf("Name: want %q, got %q", DescriptorName, d.Name)
	}
	if d.Version != DescriptorVersion {
		t.Fatalf("Version 应为适配器版本 %q, got %q", DescriptorVersion, d.Version)
	}
	// 一次性执行适配器的诚实清单:基础声明为空集 —— 协议词汇表中没有任何
	// 能力是本适配器的一次性同步进程形态真实兑现的(pi CLI 自身的 resume/
	// fork 等旗标不是本适配器兑现面,不声明)。
	all := []agentruntime.Capability{
		agentruntime.CapSessionResume,
		agentruntime.CapSessionFork,
		agentruntime.CapSessionIsolation,
		agentruntime.CapTurnSteer,
		agentruntime.CapTurnInterrupt,
		agentruntime.CapEventStreaming,
		agentruntime.CapOutputStructured,
		agentruntime.CapSandboxNative,
		agentruntime.CapToolApproval,
		agentruntime.CapToolsCustom,
		agentruntime.CapModelSelect,
		agentruntime.CapSessionPersist,
		agentruntime.CapMemoryDeclared,
	}
	for _, c := range all {
		if d.Capabilities.Has(c) {
			t.Fatalf("capability %s must stay undeclared by the pi one-shot adapter", c)
		}
	}
	if len(d.Capabilities.List()) != 0 {
		t.Fatalf("基础声明应为空集, got %v", d.Capabilities.Sorted())
	}

	// session.isolation 证据门:未持 canary 证据(零值)不得声明;置位后且
	// 仅置位后出现(与 runtimes/codex、runtimes/dsh 同一声明纪律)。
	if got := New(Config{BinPath: fakePIBin}).capabilities(); got.Has(agentruntime.CapSessionIsolation) {
		t.Fatal("零值 Config 不得声明 session.isolation(声明以 canary 证据为门)")
	}
	gated := New(Config{BinPath: fakePIBin, SessionIsolationVerified: true}).capabilities()
	if !gated.Has(agentruntime.CapSessionIsolation) || len(gated.List()) != 1 {
		t.Fatalf("SessionIsolationVerified 置位后应且仅应声明 session.isolation, got %v", gated.Sorted())
	}
}

// 未声明的能力需求必须在 CheckStart 协商处被拒绝(requested ≠ observed;
// 本适配器无 resume/model.select 兑现面)。
func TestStartRejectsUndeclaredCapabilityRequirements(t *testing.T) {
	rt := New(Config{BinPath: fakePIBin})
	ctx := context.Background()
	ws := t.TempDir()

	if _, err := rt.Start(ctx, agentruntime.StartRequest{Workspace: ws, ResumeOf: "not-a-real-handle"}); err == nil {
		t.Fatal("带 ResumeOf 的 Start 应被拒绝(未声明 session.resume)")
	} else {
		var capErr *agentruntime.CapabilityError
		if !errors.As(err, &capErr) {
			t.Fatalf("拒绝应是 *CapabilityError(经 CheckStart 协商), got %v", err)
		}
		if len(capErr.Missing) != 1 || capErr.Missing[0] != agentruntime.CapSessionResume {
			t.Fatalf("缺失能力应为 session.resume, got %v", capErr.Missing)
		}
	}
	if _, err := rt.Start(ctx, agentruntime.StartRequest{Workspace: ws, ModelProfile: "economical"}); err == nil {
		t.Fatal("带 ModelProfile 的 Start 应被拒绝(未声明 model.select)")
	} else {
		var capErr *agentruntime.CapabilityError
		if !errors.As(err, &capErr) {
			t.Fatalf("拒绝应是 *CapabilityError, got %v", err)
		}
		if len(capErr.Missing) != 1 || capErr.Missing[0] != agentruntime.CapModelSelect {
			t.Fatalf("缺失能力应为 model.select, got %v", capErr.Missing)
		}
	}
}

// 缺 workspace 的 Start 被协议闸拒绝(INV-EXEC-01)。
func TestStartRequiresWorkspace(t *testing.T) {
	rt := New(Config{BinPath: fakePIBin})
	if _, err := rt.Start(context.Background(), agentruntime.StartRequest{}); !errors.Is(err, agentruntime.ErrNoWorkspace) {
		t.Fatalf("Start without workspace: want ErrNoWorkspace, got %v", err)
	}
}

// Events 被未声明拒绝(诚实能力集的方向)。
func TestEventsRefusedWithoutStreamingCapability(t *testing.T) {
	_, s := startSession(t, nil, nil)
	if _, err := s.Events(context.Background()); !errors.Is(err, agentruntime.ErrStreamingNotDeclared) {
		t.Fatalf("Events: want ErrStreamingNotDeclared, got %v", err)
	}
}

// ---- AC:Config.Validate(执行前拒绝非法配置)---------------------------------

func TestValidateRejectsInvalidConfig(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
	}{
		{"BinPath 为空", Config{}},
		{"BinPath 空白", Config{BinPath: "   "}},
		{"OutputMode 未观测值 rpc", Config{BinPath: fakePIBin, OutputMode: "rpc"}},
		{"OutputMode 拼写漂移", Config{BinPath: fakePIBin, OutputMode: "JSON"}},
		{"ThinkingLevel 未观测值", Config{BinPath: fakePIBin, ThinkingLevel: "ultra"}},
		{"Provider 含空格", Config{BinPath: fakePIBin, Provider: "deepseek official"}},
		{"Provider 含分号", Config{BinPath: fakePIBin, Provider: "a;b"}},
		{"Model 含换行", Config{BinPath: fakePIBin, Model: "x\ny"}},
		{"Model 含引号", Config{BinPath: fakePIBin, Model: "'quote'"}},
	}
	for _, tc := range cases {
		if err := tc.cfg.Validate(); err == nil {
			t.Fatalf("%s: Validate 应拒绝", tc.name)
		}
	}
	// 合法面:默认(json)+ 显式 text + 全枚举 ThinkingLevel + 观测形态 token。
	for _, tc := range []Config{
		{BinPath: fakePIBin},
		{BinPath: fakePIBin, OutputMode: "text"},
		{BinPath: fakePIBin, OutputMode: "json", Provider: "deepseek", Model: "deepseek-v4-flash"},
		{BinPath: fakePIBin, Model: "anthropic/claude-sonnet:high"},
		{BinPath: fakePIBin, ThinkingLevel: "off"},
		{BinPath: fakePIBin, ThinkingLevel: "max"},
	} {
		if err := tc.Validate(); err != nil {
			t.Fatalf("Validate(%+v) 应通过: %v", tc, err)
		}
	}
	// 非法配置在 Start 处同样被拒(执行前的最早闸)。
	if _, err := New(Config{OutputMode: "rpc", BinPath: fakePIBin}).Start(context.Background(), agentruntime.StartRequest{Workspace: t.TempDir()}); err == nil {
		t.Fatal("非法 OutputMode 应让 Start 显式失败")
	}
}

// ---- AC:二进制缺失 → 类型化错误;钉版身份闸 ---------------------------------

func TestStartFailsWithTypeErrorWhenBinaryMissing(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-pi.exe")
	rt := New(Config{BinPath: missing})
	_, err := rt.Start(context.Background(), agentruntime.StartRequest{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("二进制缺失时 Start 应报错")
	}
	if !errors.Is(err, ErrBinaryMissing) {
		t.Fatalf("二进制缺失应是类型化错误 ErrBinaryMissing: %v", err)
	}
	// BinPath 指向目录同样类型化拒绝。
	rt2 := New(Config{BinPath: t.TempDir()})
	if _, err := rt2.Start(context.Background(), agentruntime.StartRequest{Workspace: t.TempDir()}); !errors.Is(err, ErrBinaryMissing) {
		t.Fatalf("BinPath 为目录应 ErrBinaryMissing: %v", err)
	}
}

func TestSnapshotObservesPinnedIdentity(t *testing.T) {
	rt := New(Config{BinPath: fakePIBin})
	snap, err := rt.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if snap.Name != DescriptorName || snap.AdapterVersion != DescriptorVersion {
		t.Fatalf("snapshot identity: %+v", snap)
	}
	// 版本来自现场 `--version`(替身输出钉住版本),不是硬编码常量冒充观察。
	if snap.CLIVersion != PinnedPiVersion {
		t.Fatalf("CLIVersion: want %q, got %q", PinnedPiVersion, snap.CLIVersion)
	}
	if snap.BinaryPath != fakePIBin {
		t.Fatalf("BinaryPath: want %q, got %q", fakePIBin, snap.BinaryPath)
	}
	if len(snap.Capabilities) != 0 {
		t.Fatalf("snapshot capabilities: want 空, got %v", snap.Capabilities)
	}
}

func TestStartFailsOnVersionMismatch(t *testing.T) {
	// 替身自报错误版本 → 钉版闸 fail-closed(pi 迭代活跃,版本漂移必须显形)。
	rt := New(Config{
		BinPath: fakePIBin,
		Env:     map[string]string{"TALLY_FAKE_PI_VERSION": "0.86.0"},
	})
	_, err := rt.Start(context.Background(), agentruntime.StartRequest{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("观察版本与钉住版本不一致时 Start 应拒绝")
	}
	if !errors.Is(err, ErrVersionPinMismatch) {
		t.Fatalf("版本不符应是类型化错误 ErrVersionPinMismatch: %v", err)
	}
	if !strings.Contains(err.Error(), "0.86.0") || !strings.Contains(err.Error(), PinnedPiVersion) {
		t.Fatalf("错误应同时携带观察版本与钉住版本: %v", err)
	}
	if _, err := rt.Snapshot(context.Background()); !errors.Is(err, ErrVersionPinMismatch) {
		t.Fatalf("Snapshot 同样受钉版闸约束: %v", err)
	}
}

func TestStartFailsWhenVersionUnresolvable(t *testing.T) {
	rt := New(Config{
		BinPath: fakePIBin,
		Env:     map[string]string{"TALLY_FAKE_PI_SCENARIO": "versionfail"},
	})
	if _, err := rt.Start(context.Background(), agentruntime.StartRequest{Workspace: t.TempDir()}); err == nil {
		t.Fatal("--version 不可观察时 Start 应拒绝(fail-closed)")
	}
}

func TestParseCLIIdentityAndVersionPin(t *testing.T) {
	// 观测形态:裸版本行。
	if got := parseCLIIdentity("0.85.1\n"); got.Version != "0.85.1" {
		t.Fatalf("parseCLIIdentity: want 0.85.1, got %+v", got)
	}
	if got := parseCLIIdentity("\n  \n"); got.observed() {
		t.Fatalf("空输出应为零值观察, got %+v", got)
	}
	if err := checkVersionPin("", cliIdentity{Version: PinnedPiVersion}); err != nil {
		t.Fatalf("钉住版本一致应通过: %v", err)
	}
	err := checkVersionPin("", cliIdentity{Version: "0.84.0"})
	if err == nil || !errors.Is(err, ErrVersionPinMismatch) || !strings.Contains(err.Error(), PinnedPiVersion) {
		t.Fatalf("不一致应拒绝且指明钉住版本: %v", err)
	}
}

// ---- AC:Execute(json/text)/Result/claim 纪律 --------------------------------

func TestExecuteHappyPathJSONClaimAndResult(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	_, s := startSession(t, map[string]string{"TALLY_FAKE_PI_RECORD": record}, nil)
	ctx := context.Background()

	res, err := s.Execute(ctx, agentruntime.ExecuteRequest{Instruction: "add a test"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.Status != agentruntime.RunCompleted {
		t.Fatalf("Status: want completed, got %s summary=%q", res.Status, res.Summary)
	}
	if res.Summary != "ok" {
		t.Fatalf("Summary 应来自 assistant message_end 的 text content: got %q", res.Summary)
	}
	// Usage 取自 assistant message_end 的 usage(替身固定自报 11/3)。
	if res.Usage != (agentruntime.Usage{InputTokens: 11, OutputTokens: 3}) {
		t.Fatalf("Usage 应为替身自报值, got %+v", res.Usage)
	}
	if res.RunID == "" || res.SessionID == "" {
		t.Fatalf("claim 缺少 RunID/SessionID: %+v", res)
	}
	if !strings.HasPrefix(string(res.SessionID), "pi-session-") {
		t.Fatalf("SessionID 应为适配器自造编号 pi-session-NNNNNN, got %q", res.SessionID)
	}

	snap, err := s.Result(ctx, res.RunID)
	if err != nil || snap != res {
		t.Fatalf("Result: want %+v, got %+v err=%v", res, snap, err)
	}
	if _, err := s.Result(ctx, "run-bogus"); !errors.Is(err, agentruntime.ErrRunNotFound) {
		t.Fatalf("Result(unknown): want ErrRunNotFound, got %v", err)
	}
	if err := s.Cancel(ctx, "run-bogus"); !errors.Is(err, agentruntime.ErrRunNotFound) {
		t.Fatalf("Cancel(unknown): want ErrRunNotFound, got %v", err)
	}
	if _, err := s.Execute(ctx, agentruntime.ExecuteRequest{Instruction: ""}); !errors.Is(err, agentruntime.ErrNoInstruction) {
		t.Fatalf("空指令: want ErrNoInstruction, got %v", err)
	}
}

func TestExecuteTextModeFallsBackToRawStdout(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	_, s := startSession(t, map[string]string{"TALLY_FAKE_PI_RECORD": record}, func(c *Config) {
		c.OutputMode = "text"
	})
	res, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "add a test"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.Status != agentruntime.RunCompleted || res.Summary != "ok" {
		t.Fatalf("text 模式 Summary 应为 stdout 原文: status=%s summary=%q", res.Status, res.Summary)
	}
	// text 模式 stdout 无用量自报:Usage 恒零值(宁缺毋假)。
	if res.Usage != (agentruntime.Usage{}) {
		t.Fatalf("text 模式 Usage 应为零值, got %+v", res.Usage)
	}
}

// ObservedIdentity:版本来自 --version 探针;Model/SessionID 是 json 流内
// 的观察值(替身固定自报 fake-model / 固定 UUID,刻意 ≠ 任何期望配置);
// RuntimeName 未观察为空;期望旗标绝不回填观察字段。
func TestObservedIdentityFromStreamNotFromConfig(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	_, s := startSession(t, map[string]string{"TALLY_FAKE_PI_RECORD": record}, func(c *Config) {
		c.Provider = "deepseek"
		c.Model = "deepseek-v4-flash"
	})
	id := s.Identity()
	if id.RuntimeVersion != PinnedPiVersion {
		t.Fatalf("Identity.RuntimeVersion: want %q, got %q", PinnedPiVersion, id.RuntimeVersion)
	}
	if id.RuntimeName != "" {
		t.Fatalf("RuntimeName 未观察应为空, got %q", id.RuntimeName)
	}
	if id.Model != "" || id.SessionID != "" {
		t.Fatalf("run 前 Model/SessionID 应未观察为空, got %+v", id)
	}
	res, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "add a test"})
	if err != nil || res.Status != agentruntime.RunCompleted {
		t.Fatalf("Execute: res=%+v err=%v", res, err)
	}
	id = s.Identity()
	if id.Model != fakeModel || id.SessionID != agentruntime.SessionID(fakeSessionID) {
		t.Fatalf("Identity 应为流内观察值(model=%q session=%q), got %+v", fakeModel, fakeSessionID, id)
	}
	if id.Model == "deepseek-v4-flash" {
		t.Fatal("期望旗标不得回填 Model(伪造观察)")
	}
	if !id.Observed() {
		t.Fatal("已建立观察后 Observed() 应为 true")
	}
}

func TestExecutePinsWorkspaceViaProcessCwd(t *testing.T) {
	// workspace 经 spawn cwd 钉定(pi 无 cwd 旗标):替身在它的 cwd 里真实
	// 读到 marker 文件 = cwd 绑定的效果级证明。
	ws := t.TempDir()
	marker := filepath.Join(ws, "marker.txt")
	if err := os.WriteFile(marker, []byte("cwd-bound-answer"), 0o644); err != nil {
		t.Fatal(err)
	}
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	_, s := startSessionIn(t, ws, map[string]string{"TALLY_FAKE_PI_RECORD": record}, nil)

	res, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "read marker.txt"})
	if err != nil || res.Status != agentruntime.RunCompleted {
		t.Fatalf("Execute: res=%+v err=%v", res, err)
	}
	if res.Summary != "cwd-bound-answer" {
		t.Fatalf("最终回答应来自替身在 cwd 内真实读到的 marker 文件: got %q", res.Summary)
	}
	rec := lastRecord(t, record)
	if rec.Cwd != s.Workspace() {
		t.Fatalf("替身观测到的进程 cwd %q 应是提供的 workspace %q", rec.Cwd, s.Workspace())
	}
	// argv 形态:[bin, -p, --mode, json, --no-session, --session-dir, <dir>,
	// --no-extensions, --no-skills, --no-prompt-templates, --no-themes,
	// --no-context-files, --no-approve, --, instruction](零期望配置 = 零
	// provider/model/thinking 旗标)。
	wantArgv := []string{
		fakePIBin, "-p", "--mode", "json",
		"--no-session", "--session-dir", s.dirs.sessions,
		"--no-extensions", "--no-skills", "--no-prompt-templates",
		"--no-themes", "--no-context-files", "--no-approve",
		"--", "read marker.txt",
	}
	if !reflect.DeepEqual(rec.Argv, wantArgv) {
		t.Fatalf("argv 应为 %v, got %v", wantArgv, rec.Argv)
	}
}

func TestExpectedModelProviderFlagsPassedNotObserved(t *testing.T) {
	// 期望 Provider/Model/ThinkingLevel 只经旗标传递,且与流内观察值分清。
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	_, s := startSession(t, map[string]string{"TALLY_FAKE_PI_RECORD": record}, func(c *Config) {
		c.Provider = "deepseek"
		c.Model = "deepseek-v4-flash"
		c.ThinkingLevel = "low"
	})
	res, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "x"})
	if err != nil || res.Status != agentruntime.RunCompleted {
		t.Fatalf("Execute: res=%+v err=%v", res, err)
	}
	rec := lastRecord(t, record)
	if rec.Provider != "deepseek" || rec.Model != "deepseek-v4-flash" || rec.Thinking != "low" {
		t.Fatalf("期望旗标应原样传递: provider=%q model=%q thinking=%q", rec.Provider, rec.Model, rec.Thinking)
	}
	// 旗标必须在 -- 之前(选项区),instruction 恒为最后位置参数。
	argv := rec.Argv
	sep := -1
	for i, a := range argv {
		if a == "--" {
			sep = i
			break
		}
	}
	if sep < 0 || argv[len(argv)-1] != "x" {
		t.Fatalf("-- 应存在且 instruction 恒为最后参数: %v", argv)
	}
}

// ---- AC:隔离与 spawn 环境强制面 ----------------------------------------------

func TestPerSessionIsolationAndForcedEnv(t *testing.T) {
	// 模拟「operator 环境带 owner pi 配置目录、HOME 与他家 provider 凭据、
	// 关 offline 开遥测」的部署面:适配器必须强制 per-session 目录组与
	// 剥离/硬化面,压倒继承环境与 Config.Env 中的同名值。
	ownerRoot := t.TempDir()
	ownerAgentDir := filepath.Join(ownerRoot, "owner-pi-agent")
	ownerHome := filepath.Join(ownerRoot, "owner-home")
	for _, d := range []string{ownerAgentDir, ownerHome} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PI_CODING_AGENT_DIR", ownerAgentDir)
	t.Setenv("PI_CODING_AGENT_SESSION_DIR", filepath.Join(ownerRoot, "owner-sessions"))
	t.Setenv("HOME", ownerHome)
	t.Setenv("USERPROFILE", ownerHome)
	t.Setenv("ANTHROPIC_API_KEY", "operator-anthropic-key")
	t.Setenv("OPENAI_API_KEY", "operator-openai-key")
	t.Setenv("PI_OFFLINE", "0")
	t.Setenv("PI_TELEMETRY", "1")

	record := filepath.Join(t.TempDir(), "calls.jsonl")
	_, s1 := startSession(t, map[string]string{
		"TALLY_FAKE_PI_RECORD": record,
		"DEEPSEEK_API_KEY":     "test-key",    // 显式注入(认证透传通道)
		"PI_OFFLINE":           "0",           // Config.Env 同名值必须被强制键压倒
		"PI_CODING_AGENT_DIR":  ownerAgentDir, // Config.Env 同名值必须被强制键压倒
	}, nil)
	if _, err := s1.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "x"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	rec := lastRecord(t, record)

	// per-session 目录组:全部落在会话根之下,绝不指向 owner 面。
	root := s1.dirs.root
	if rec.PiAgentDir == "" || rec.PiAgentDir == ownerAgentDir || !strings.HasPrefix(rec.PiAgentDir, root) {
		t.Fatalf("PI_CODING_AGENT_DIR 应为 per-session 目录(=%s,root=%s,owner=%s)", rec.PiAgentDir, root, ownerAgentDir)
	}
	if rec.PiSessionDirEnv == "" || !strings.HasPrefix(rec.PiSessionDirEnv, root) {
		t.Fatalf("PI_CODING_AGENT_SESSION_DIR 应为 per-session 目录, got %q", rec.PiSessionDirEnv)
	}
	if rec.Home == "" || rec.Home == ownerHome || !strings.HasPrefix(rec.Home, root) {
		t.Fatalf("HOME 应为 per-session 目录, got %q(owner=%q)", rec.Home, ownerHome)
	}
	if rec.UserProfile != rec.Home {
		t.Fatalf("USERPROFILE 应与 HOME 同源重定向, got %q vs %q", rec.UserProfile, rec.Home)
	}
	if rec.SessionDir == "" || !strings.HasPrefix(rec.SessionDir, root) {
		t.Fatalf("--session-dir 应为 per-session 目录, got %q", rec.SessionDir)
	}
	// --no-session 必传(观测:不落会话文件)。
	if !rec.NoSession {
		t.Fatal("argv 必须携带 --no-session")
	}
	// 发现面全关 + --no-approve。
	if !rec.NoExtensions || !rec.NoSkills || !rec.NoPromptTmpl || !rec.NoThemes || !rec.NoContextFiles || !rec.NoApprove {
		t.Fatalf("隔离旗标必须全开: %+v", rec)
	}
	// 出口硬化强制(压倒继承与 Config.Env)。
	if rec.Offline != offlineForced {
		t.Fatalf("PI_OFFLINE 应被强制为 %s, got %q", offlineForced, rec.Offline)
	}
	if rec.Telemetry != telemetryDisabled {
		t.Fatalf("PI_TELEMETRY 应被强制为 %s, got %q", telemetryDisabled, rec.Telemetry)
	}
	// 凭据剥离面:operator 环境的他家 provider key 不得流入。
	if rec.AnthropicKeySet || rec.OpenaiKeySet {
		t.Fatalf("operator 的 ANTHROPIC/OPENAI key 应被剥离: anthropic=%v openai=%v", rec.AnthropicKeySet, rec.OpenaiKeySet)
	}
	// 显式注入经 Config.Env 生效(认证透传通道)。
	if !rec.DeepseekKeySet {
		t.Fatal("Config.Env 注入的 DEEPSEEK_API_KEY 应流入 spawn 环境")
	}

	// 第二个会话获得不同的全新目录组(会话边界,不共享)。
	record2 := filepath.Join(t.TempDir(), "calls2.jsonl")
	_, s2 := startSession(t, map[string]string{"TALLY_FAKE_PI_RECORD": record2}, nil)
	if _, err := s2.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "x"}); err != nil {
		t.Fatalf("Execute(s2): %v", err)
	}
	rec2 := lastRecord(t, record2)
	if rec2.PiAgentDir == "" || rec2.PiAgentDir == rec.PiAgentDir {
		t.Fatalf("两个会话应获得不同的全新目录组: %q vs %q", rec.PiAgentDir, rec2.PiAgentDir)
	}

	// Close 删除本会话隔离目录组(尽力而为;等待短暂收敛后断言)。
	if err := s1.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(root); os.IsNotExist(err) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("Close 后 per-session 目录组 %s 未被删除", root)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestSpawnEnvForcesAndStrips(t *testing.T) {
	// spawnEnv 纯函数面:剥离 pi 凭据/配置键(大小写不敏感)、强制隔离键
	// (压倒 Config.Env 同名值)、保留 OS 功能面、Config.Env 显式注入。
	ownerRoot := t.TempDir()
	t.Setenv("DEEPSEEK_API_KEY", "operator-deepseek")
	t.Setenv("ANTHROPIC_API_KEY", "operator-anthropic")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "operator-aws")
	t.Setenv("PI_CODING_AGENT_DIR", filepath.Join(ownerRoot, "owner-agent"))
	t.Setenv("PI_PACKAGE_DIR", filepath.Join(ownerRoot, "owner-packages"))
	t.Setenv("PI_OFFLINE", "0")
	t.Setenv("PI_TELEMETRY", "1")
	t.Setenv("PATH", "/kept/bin") // OS 功能面代表:不在剥离面,必须保留

	dirs, err := newSessionDirs()
	if err != nil {
		t.Fatalf("newSessionDirs: %v", err)
	}
	defer func() { _ = dirs.Close() }()

	env := spawnEnv(map[string]string{
		"DEEPSEEK_API_KEY": "explicit-key", // 唯一显式凭据通道
		"home":             "must-drop",    // 强制键的异大小写形态:压倒
		"pi_offline":       "0",            // 同上
		"CUSTOM_FLAG":      "kept",
	}, dirs)

	if got := env["DEEPSEEK_API_KEY"]; got != "explicit-key" {
		t.Fatalf("Config.Env 显式凭据应覆盖剥离面: got %q", got)
	}
	for _, k := range []string{"ANTHROPIC_API_KEY", "AWS_SECRET_ACCESS_KEY", "ANTHROPIC_API_TOKEN", "OPENAI_API_KEY", "GEMINI_API_KEY"} {
		if _, ok := env[k]; ok {
			t.Fatalf("operator 凭据 %s 应被剥离", k)
		}
	}
	if got := env[envPiAgentDir]; got != dirs.agentConfig {
		t.Fatalf("PI_CODING_AGENT_DIR 应强制 per-session 值, got %q(want %q)", got, dirs.agentConfig)
	}
	if got := env[envPiSessionDir]; got != dirs.sessions {
		t.Fatalf("PI_CODING_AGENT_SESSION_DIR 应强制 per-session 值, got %q", got)
	}
	if got := env[envHome]; got != dirs.home {
		t.Fatalf("HOME 应强制 per-session 值, got %q", got)
	}
	if got := env[envUserProfile]; got != dirs.home {
		t.Fatalf("USERPROFILE 应强制 per-session 值, got %q", got)
	}
	if got := env[envPiOffline]; got != offlineForced {
		t.Fatalf("PI_OFFLINE 应强制 %s, got %q", offlineForced, got)
	}
	if got := env[envPiTelemetry]; got != telemetryDisabled {
		t.Fatalf("PI_TELEMETRY 应强制 %s, got %q", telemetryDisabled, got)
	}
	if got := env["PATH"]; got != "/kept/bin" {
		t.Fatalf("OS 功能面 PATH 应保留, got %q", got)
	}
	if got := env["CUSTOM_FLAG"]; got != "kept" {
		t.Fatalf("非剥离/非强制的 Config.Env 键应透传, got %q", got)
	}
	// 异大小写形态不得共存(Windows 环境变量名不区分大小写)。
	for k := range env {
		if strings.EqualFold(k, "HOME") && k != envHome {
			t.Fatalf("HOME 的异大小写残留: %s", k)
		}
	}
}

// ---- 失败/取消/崩溃/关闭(claim 纪律 + 整树 teardown)-----------------------------

func TestKeylessIsFailedClaimNotError(t *testing.T) {
	// 观测的 keyless 契约(0.85.1):缺 provider key → exit 1 + stderr
	// "No API key found..."。替身经显式门控启用;同时证明 operator 环境的
	// DEEPSEEK_API_KEY 被剥离(未显式注入 → 替身看不到 key)。注意覆盖
	// Config.Env:helper 缺省注入的 DEEPSEEK_API_KEY 必须去掉。
	t.Setenv("DEEPSEEK_API_KEY", "operator-key-that-must-be-stripped")
	_, s := startSession(t, nil, func(c *Config) {
		c.Env = map[string]string{"TALLY_FAKE_PI_REQUIRE_KEY": "1"}
	})
	res, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "reply ok"})
	if err != nil {
		t.Fatalf("凭据缺失是 run 级失败,不是 Execute error: %v", err)
	}
	if res.Status != agentruntime.RunFailed {
		t.Fatalf("Status: want failed, got %s summary=%q", res.Status, res.Summary)
	}
	if !strings.Contains(res.Summary, "No API key found for deepseek.") {
		t.Fatalf("summary 应携带 stderr 错误原文: %q", res.Summary)
	}
}

func TestNonZeroExitSurfacesAsFailedClaim(t *testing.T) {
	_, s := startSession(t, map[string]string{"TALLY_FAKE_PI_SCENARIO": "exitfail"}, nil)
	res, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "x"})
	if err != nil {
		t.Fatalf("非零退出是 RunFailed claim: %v", err)
	}
	if res.Status != agentruntime.RunFailed || !strings.Contains(res.Summary, "exit status 7") {
		t.Fatalf("want failed + 退出码, got %s/%q", res.Status, res.Summary)
	}
	// 崩溃前的部分输出不得冒充最终回答。
	if strings.Contains(res.Summary, "partial") {
		t.Fatalf("失败 claim 的 summary 不得携带 stdout 部分输出: %q", res.Summary)
	}
}

func TestExitZeroWithoutAnswerFailsSafe(t *testing.T) {
	// exit 0 但无可解析回答:契约未兑现 → FAIL-SAFE 判败,不猜测成功。
	_, s := startSession(t, map[string]string{"TALLY_FAKE_PI_SCENARIO": "emptyok"}, nil)
	res, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "x"})
	if err != nil {
		t.Fatalf("契约不兑现是 RunFailed claim: %v", err)
	}
	if res.Status != agentruntime.RunFailed {
		t.Fatalf("Status: want failed, got %s summary=%q", res.Status, res.Summary)
	}
}

// Failure diagnostics cross a durable claim boundary. Secrets emitted by an
// external CLI must be redacted before they can enter AgentRunResult/log/UI.
func TestFailureSummaryRedactsSecrets(t *testing.T) {
	_, s := startSession(t, map[string]string{"TALLY_FAKE_PI_SCENARIO": "leakstderr"}, nil)
	res, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "x"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.Status != agentruntime.RunFailed {
		t.Fatalf("Status: want failed, got %s", res.Status)
	}
	secret := "sk-fake-secret-1234567890abcdef"
	if strings.Contains(res.Summary, secret) {
		t.Fatalf("failure summary leaked secret: %q", res.Summary)
	}
	if !strings.Contains(res.Summary, "[REDACTED]") {
		t.Fatalf("failure summary should contain redaction marker: %q", res.Summary)
	}
}

// TestCancelKillsWholeProcessTree 是整树终止证明:Cancel/ctx 取消必须整树
// 终止 —— pi 进程与其子进程全部消失,无孤儿残留。
//
// 双重证据:(a) WaitDelay 被刻意抬高到 30s —— 若子进程在整树终止后存活,
// 它持有的 stdout 管道会把 Execute 拖到 WaitDelay 兜底,waitForStatus 的
// 20s 上限必然先超时;快速收敛即证明持管道的子进程确实死了。
// (b) tasklist 镜像名核对确认子进程 pid 不再属于任何 fakepi.exe。
func TestCancelKillsWholeProcessTree(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	_, s := startSession(t, map[string]string{"TALLY_FAKE_PI_RECORD": record}, func(c *Config) { c.WaitDelay = 30 * time.Second })
	ctx := context.Background()

	outcome := make(chan executeOutcome, 1)
	go func() {
		res, err := s.Execute(ctx, agentruntime.ExecuteRequest{Instruction: longMarker + " run"})
		outcome <- executeOutcome{res, err}
	}()

	// 等 CLI 记下子进程 PID(spawn 完成后),再取消:确保整树里确实有子进程
	// 可泄漏。
	rec := waitForChildPID(t, record)
	if rec.ChildPID <= 0 || !processAlive(rec.ChildPID) {
		t.Fatalf("替身子进程应在运行: pid=%d", rec.ChildPID)
	}

	if err := s.Cancel(ctx, "run-bogus"); !errors.Is(err, agentruntime.ErrRunNotFound) {
		t.Fatalf("Cancel(unknown): want ErrRunNotFound, got %v", err)
	}
	id, ok := s.InFlightRunID()
	if !ok {
		t.Fatal("应存在 in-flight run")
	}
	if err := s.Cancel(ctx, id); err != nil {
		t.Fatalf("Cancel(in-flight): %v", err)
	}

	got := waitForStatus(t, outcome)
	if got.err != nil {
		t.Fatalf("被 Cancel 的回合是正常终态: %v", got.err)
	}
	if got.res.Status != agentruntime.RunCancelled {
		t.Fatalf("Status: want cancelled, got %s summary=%q", got.res.Status, got.res.Summary)
	}
	snap, err := s.Result(ctx, id)
	if err != nil || snap != got.res {
		t.Fatalf("Result after cancel: want %+v, got %+v err=%v", got.res, snap, err)
	}
	// 对已到终态的 Cancel 幂等成功。
	if err := s.Cancel(ctx, id); err != nil {
		t.Fatalf("Cancel(terminal run) 应幂等成功: %v", err)
	}

	// 整树核心:子进程随 pi 进程一并消失,无孤儿。
	waitForGone(t, rec.ChildPID)
}

func TestExecuteContextCancelIsRunCancelled(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	_, s := startSession(t, map[string]string{"TALLY_FAKE_PI_RECORD": record}, nil)
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
	// 清理可能的 CLI 孤儿子进程(ctx 取消同样走整树终止;此处兜底)。
	if rec := lastRecordFor(record); rec.ChildPID > 0 {
		waitForGone(t, rec.ChildPID)
	}
}

// 生命周期 AC「no orphan process in cancellation/timeout tests」:ctx 超时
// (时限到期触发 Cancel 路径)必须收敛 RunCancelled,且整树终止后子进程
// 零残留(真实进程树取证)。
func TestContextTimeoutLeavesNoOrphan(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	_, s := startSession(t, map[string]string{"TALLY_FAKE_PI_RECORD": record}, func(c *Config) { c.WaitDelay = 2 * time.Second })

	ctx, cancel := context.WithCancel(context.Background())
	outcome := make(chan executeOutcome, 1)
	go func() {
		res, err := s.Execute(ctx, agentruntime.ExecuteRequest{Instruction: longMarker + " run"})
		outcome <- executeOutcome{res, err}
	}()
	rec := waitForChildPID(t, record)

	// 时限到期(以 ctx 取消表达):收敛必须是 RunCancelled 终态。
	cancel()
	got := waitForStatus(t, outcome)
	if got.err != nil || got.res.Status != agentruntime.RunCancelled {
		t.Fatalf("超时取消应 RunCancelled: %+v err=%v", got.res, got.err)
	}
	// 孤儿零残留:子进程在整树终止后必须消失。
	if rec.ChildPID > 0 {
		waitForGone(t, rec.ChildPID)
	}
}

func TestProcessCrashHandledAndSessionResponsive(t *testing.T) {
	// crash 场景:外部单进程 kill pi 进程(非 Cancel、非整树)必须归为
	// RunFailed,且 Execute 必须收敛,会话不变成僵尸。kill 前先等 CLI 记下
	// 子进程 PID:保证孤儿真实存在并持有输出管道 —— 收敛只能经 WaitDelay
	// 兜底,不得挂死、不得伪装成取消或成功。
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	_, s := startSession(t, map[string]string{"TALLY_FAKE_PI_RECORD": record}, func(c *Config) { c.WaitDelay = 2 * time.Second })
	ctx := context.Background()

	outcome := make(chan executeOutcome, 1)
	go func() {
		res, err := s.Execute(ctx, agentruntime.ExecuteRequest{Instruction: longMarker + " run"})
		outcome <- executeOutcome{res, err}
	}()
	waitForChildPID(t, record)

	deadline := time.Now().Add(10 * time.Second)
	var id agentruntime.RunID
	for {
		got, ok := s.InFlightRunID()
		if ok {
			id = got
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("run 未进入 in-flight")
		}
		time.Sleep(10 * time.Millisecond)
	}
	proc, ok := s.RunProcess(id)
	if !ok {
		t.Fatal("in-flight run 应可经 RunProcess 观测到底层进程")
	}
	if err := proc.Kill(); err != nil {
		t.Fatalf("外部 kill: %v", err)
	}

	got := waitForStatus(t, outcome)
	if got.err != nil {
		t.Fatalf("崩溃应以 RunFailed claim 上报并收敛(不挂死): %v", got.err)
	}
	if got.res.Status != agentruntime.RunFailed {
		t.Fatalf("Status: want failed(外部 kill 非 Cancel), got %s summary=%q", got.res.Status, got.res.Summary)
	}
	// 崩溃后协议哨兵仍响应(会话未成僵尸)。
	if _, err := s.Result(ctx, "run-bogus"); !errors.Is(err, agentruntime.ErrRunNotFound) {
		t.Fatalf("崩溃后 Result(unknown): want ErrRunNotFound, got %v", err)
	}
	// 清理被单进程 kill 留下的孤儿子进程(取证面:正是 WaitDelay 兜住的那种)。
	if rec := lastRecordFor(record); rec.ChildPID > 0 {
		if p, err := os.FindProcess(rec.ChildPID); err == nil {
			_ = killTree(p)
		}
		waitForGone(t, rec.ChildPID)
	}
}

func TestCloseKillsInFlightAndRejectsFurtherOps(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	_, s := startSession(t, map[string]string{"TALLY_FAKE_PI_RECORD": record}, nil)
	ctx := context.Background()

	outcome := make(chan executeOutcome, 1)
	go func() {
		res, err := s.Execute(ctx, agentruntime.ExecuteRequest{Instruction: longMarker + " run"})
		outcome <- executeOutcome{res, err}
	}()
	waitForChildPID(t, record)

	root := s.dirs.root
	if root == "" {
		t.Fatal("会话应有 per-session 目录组")
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
	if err := s.Cancel(ctx, got.res.RunID); !errors.Is(err, agentruntime.ErrSessionClosed) {
		t.Fatalf("Cancel after Close: want ErrSessionClosed, got %v", err)
	}
	if err := s.Close(ctx); err != nil {
		t.Fatalf("Close 应幂等: %v", err)
	}
	// 子进程不残留(整树终止)。
	if rec := lastRecordFor(record); rec.ChildPID > 0 {
		waitForGone(t, rec.ChildPID)
	}
	// per-session 目录组被删除。
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(root); os.IsNotExist(err) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("Close 后目录组 %s 未被删除", root)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// ---- json NDJSON 流解析(纯函数面)---------------------------------------------

func TestParseJSONStreamFailSafeOnPartialStream(t *testing.T) {
	// 观测:取消/崩溃时 json 流在部分输出后被截断(尾行可能不完整)。解析
	// 必须跳过截断行、保留 session 行观察,且不得凭 session 行猜测成功。
	partial := `{"type":"session","version":3,"id":"` + fakeSessionID + `","cwd":"C:\\ws"}
{"type":"agent_start"}
{"type":"message_update","assistantMessageEvent":{"delta":"trunc`
	obs, usage := parseJSONStream(partial)
	if obs.sessionHandle != fakeSessionID {
		t.Fatalf("session 行观察应保留, got %q", obs.sessionHandle)
	}
	if obs.hasAssistant {
		t.Fatal("截断流不得被当成有 assistant 回答(FAIL-SAFE)")
	}
	if usage != (agentruntime.Usage{}) {
		t.Fatalf("截断流 Usage 应为零值, got %+v", usage)
	}

	// 完整流:取最后一条带文本 assistant message_end。
	full := `{"type":"session","version":3,"id":"` + fakeSessionID + `","cwd":"C:\\ws"}
{"type":"message_end","message":{"role":"user","content":[{"type":"text","text":"q"}]}}
{"type":"message_end","message":{"role":"assistant","content":[{"type":"text","text":"part1"}],"provider":"p1","model":"m1","usage":{"input":1,"output":2},"stopReason":"stop"}}
{"type":"message_end","message":{"role":"assistant","content":[{"type":"text","text":"final"},{"type":"text","text":"part2"}],"provider":"p2","model":"m2","usage":{"input":5,"output":7},"stopReason":"stop"}}
{"type":"agent_settled"}`
	obs, usage = parseJSONStream(full)
	if !obs.hasAssistant || obs.text != "final\npart2" {
		t.Fatalf("应取最后一条带文本 assistant message_end: %+v", obs)
	}
	if obs.model != "m2" || obs.provider != "p2" {
		t.Fatalf("观察应来自流内自报: model=%q provider=%q", obs.model, obs.provider)
	}
	if usage != (agentruntime.Usage{InputTokens: 5, OutputTokens: 7}) {
		t.Fatalf("Usage 应取自同一条 message_end, got %+v", usage)
	}
}

// ---- 生命周期 AC:no owner mutation -------------------------------------------

// 「no owner-repo mutation outside assigned workspace」:workspace 之外的目录
// (owner 主工作树的代理)在执行/取消/关闭全生命周期零变更 —— 真实进程 cwd
// 钉定在 workspace,替身只在其 cwd 内活动;per-session 目录组在 Close 后删除
// (隔离无跨运行残留)。
func TestNoOwnerRepoMutation(t *testing.T) {
	owner := t.TempDir()
	ownerFile := filepath.Join(owner, "owned.txt")
	if err := os.WriteFile(ownerFile, []byte("owner worktree proxy"), 0o644); err != nil {
		t.Fatal(err)
	}
	dirHash := func() string {
		entries, err := os.ReadDir(owner)
		if err != nil {
			t.Fatalf("ReadDir(owner): %v", err)
		}
		var sb strings.Builder
		for _, e := range entries {
			sb.WriteString(e.Name())
			sb.WriteString("|")
			if info, err := e.Info(); err == nil {
				sb.WriteString(info.ModTime().UTC().Format(time.RFC3339Nano))
			}
			sb.WriteString(";")
		}
		return sb.String()
	}
	before := dirHash()

	ws := t.TempDir()
	marker := filepath.Join(ws, "marker.txt")
	if err := os.WriteFile(marker, []byte("cwd-bound-answer"), 0o644); err != nil {
		t.Fatal(err)
	}
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	_, s := startSessionIn(t, ws, map[string]string{"TALLY_FAKE_PI_RECORD": record}, nil)

	res, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "read marker.txt"})
	if err != nil || res.Status != agentruntime.RunCompleted {
		t.Fatalf("Execute: res=%+v err=%v", res, err)
	}
	// 替身观测到的 cwd 是 workspace,不是 owner。
	if rec := lastRecord(t, record); rec.Cwd != s.Workspace() {
		t.Fatalf("进程 cwd %q 应钉定 workspace %q", rec.Cwd, s.Workspace())
	}
	if err := s.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
	// owner 目录零变更(无新文件、无改名、无触碰)。
	if after := dirHash(); after != before {
		t.Fatalf("owner 目录在执行/关闭后被变更: before=%q after=%q", before, after)
	}
	// per-session 目录组已随 Close 删除(隔离无跨运行残留)。
	if _, err := os.Stat(s.dirs.root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Close 后 per-session 目录组应已删除: %v", err)
	}
}

// ---- PRD §35 通用 conformance(全场景,零声明 → cap:* 全部不运行)---------------

// TestPIRuntimeConformance 对挂在替身 CLI 上的 PiRuntime 跑完整
// runtimeconformance 套件:start / cwd / execute / result / events(未声明
// 拒绝方向)/ cancel(ctx 路径)/ crash / workspace / usage / capability_truth。
// 基础声明为空集 → 全部能力专项场景按门控不得运行(声明真实性)。
func TestPIRuntimeConformance(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	rt := New(Config{
		BinPath:   fakePIBin,
		Env:       testEnv(record),
		WaitDelay: 2 * time.Second, // crash 注入面的孤儿管道兜底(测试用短值)
	})
	rep := runtimeconformance.Run(t, rt, runtimeconformance.Options{
		Probes: runtimeconformance.Probes{
			LongRunInstruction: longMarker + " run",
			CrashRun: func(ctx context.Context, sess agentruntime.Session, longRunInstruction string) (agentruntime.AgentRunResult, error) {
				ps, ok := sess.(*Session)
				if !ok {
					return agentruntime.AgentRunResult{}, fmt.Errorf("crash probe: want *pi.Session, got %T", sess)
				}
				outcome := make(chan executeOutcome, 1)
				go func() {
					res, err := sess.Execute(ctx, agentruntime.ExecuteRequest{Instruction: longRunInstruction})
					outcome <- executeOutcome{res, err}
				}()
				// 等 CLI 记下子进程 PID:保证 kill 时孤儿真实存在并持有输出
				// 管道(WaitDelay 兜底路径被确定性覆盖)。
				waitForChildPID(t, record)
				// 未声明 event.streaming,订阅不到 run.started:经诊断缝定位。
				deadline := time.Now().Add(10 * time.Second)
				var id agentruntime.RunID
				for {
					got, ok := ps.InFlightRunID()
					if ok {
						id = got
						break
					}
					if time.Now().After(deadline) {
						return agentruntime.AgentRunResult{}, fmt.Errorf("crash probe: run 未进入 in-flight")
					}
					time.Sleep(10 * time.Millisecond)
				}
				proc, ok := ps.RunProcess(id)
				if !ok {
					return agentruntime.AgentRunResult{}, fmt.Errorf("crash probe: in-flight run 无底层进程可注入崩溃")
				}
				if err := proc.Kill(); err != nil {
					return agentruntime.AgentRunResult{}, fmt.Errorf("注入进程崩溃: %w", err)
				}
				select {
				case out := <-outcome:
					// 清理单进程 kill 留下的孤儿子进程(避免污染测试机)。
					if rec := lastRecordFor(record); rec.ChildPID > 0 {
						if p, err := os.FindProcess(rec.ChildPID); err == nil {
							_ = killTree(p)
						}
					}
					return out.res, out.err
				case <-ctx.Done():
					return agentruntime.AgentRunResult{}, ctx.Err()
				}
			},
			WorkspaceEffect: func(ctx context.Context, sess agentruntime.Session) error {
				ps, ok := sess.(*Session)
				if !ok {
					return fmt.Errorf("workspace probe: want *pi.Session, got %T", sess)
				}
				rec := lastRecordFor(record)
				if rec.Cwd != ps.Workspace() {
					return fmt.Errorf("替身观测到的进程 cwd %q 不是提供的 workspace %q", rec.Cwd, ps.Workspace())
				}
				return nil
			},
		},
	})
	if !rep.OK() {
		t.Fatalf("PiRuntime conformance 未通过:\n%s", rep.FailureText())
	}
	// 核心场景必须真的跑过。
	for _, want := range []string{
		"start", "cwd", "execute", "result", "events",
		"cancel", "crash", "workspace", "usage", "capability_truth",
	} {
		if !rep.Ran(want) {
			t.Fatalf("场景 %s 未运行", want)
		}
	}
	// 零能力声明 → 全部能力专项场景不得运行(声明真实性:未声明即未验证)。
	for _, capped := range []string{
		"cap:session.resume", "cap:session.fork", "cap:output.structured",
		"cap:tool.approval", "cap:session.isolation",
	} {
		if rep.Ran(capped) {
			t.Fatalf("场景 %s 不应运行(基础声明为空集)", capped)
		}
	}
}

// ---- 真实 pi 表面(CLI 部署 + 凭据门控,环境前提)------------------------------

// TestLivePISurface 在 TALLY_PI_LIVE=1 时对真实 pi CLI 验证:钉版身份闸 +
// 一次真实回合(指令为无害回显,至多一个模型 turn)。未请求 live 时 SKIP;
// 一旦显式请求,CLI/key 缺失都是 gate 配置失败而不是 SKIP。CLI 路径经
// TALLY_PI_BIN(与生产 worker 同一显式配置面);凭据经 TALLY_PI_LIVE_KEY_VAR
// 指名的环境变量注入(缺省 DEEPSEEK_API_KEY)。
func TestLivePISurface(t *testing.T) {
	if os.Getenv("TALLY_PI_LIVE") == "" {
		t.Skip("TALLY_PI_LIVE 未设置:跳过真实 pi 表面(0.85.1 观测证据见包注释,真跑收据由部署方留档)")
	}
	bin := strings.TrimSpace(os.Getenv("TALLY_PI_BIN"))
	if bin == "" {
		t.Fatal("TALLY_PI_LIVE 已启用但缺 TALLY_PI_BIN:live gate 必须使用显式路径(钉版面不做隐式发现)")
	}
	keyVar := strings.TrimSpace(os.Getenv("TALLY_PI_LIVE_KEY_VAR"))
	if keyVar == "" {
		keyVar = "DEEPSEEK_API_KEY"
	}
	if strings.TrimSpace(os.Getenv(keyVar)) == "" {
		t.Fatalf("TALLY_PI_LIVE 已启用但缺 %s:显式 live gate 不允许 SKIP 真实完成回合", keyVar)
	}

	// 1) 钉版身份闸(零模型成本)。
	rt := New(Config{BinPath: bin, Provider: "deepseek", Env: map[string]string{keyVar: os.Getenv(keyVar)}})
	snap, err := rt.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot(真实 CLI): %v", err)
	}
	t.Logf("真实 pi 身份: version=%s bin=%s", snap.CLIVersion, snap.BinaryPath)

	// 2) 一次真实回合(至多一个模型 turn,无害回显指令)。
	live := New(Config{
		BinPath:  bin,
		Provider: "deepseek",
		Env:      map[string]string{keyVar: os.Getenv(keyVar)},
	})
	s, err := live.Start(context.Background(), agentruntime.StartRequest{Workspace: t.TempDir()})
	if err != nil {
		t.Fatalf("Start(live): %v", err)
	}
	defer func() { _ = s.Close(context.Background()) }()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	res, err := s.Execute(ctx, agentruntime.ExecuteRequest{Instruction: "conformance: 请原样回复 ok"})
	if err != nil {
		t.Fatalf("真实回合 Execute: %v", err)
	}
	if res.Status != agentruntime.RunCompleted || strings.TrimSpace(res.Summary) == "" {
		t.Fatalf("真实回合应完成且 summary 非空: %s summary=%q", res.Status, truncForLog(res.Summary))
	}
	// IdentityReporter 是可选缝:具体 *Session 实现它(接口值需断言)。
	ps, ok := s.(*Session)
	if !ok {
		t.Fatalf("Start(live): want *pi.Session, got %T", s)
	}
	id := ps.Identity()
	t.Logf("真实回合完成: summary=%q observed_model=%q observed_session=%q", truncForLog(res.Summary), id.Model, id.SessionID)
}

// truncForLog 供日志截断。
func truncForLog(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return s
}
