package opencode

// OpenCode 适配器的测试策略(对齐 runtimes/dsh 的替身 CLI 模式):
//
//   - 全部进程面经「替身 opencode CLI」驱动:TestMain 从
//     testdata/fakeopencode/main.go 现场构建替身二进制,适配器对它 spawn/
//     读取/取消/整树终止 —— 进程边界、run 契约(--version、NDJSON 事件流、
//     退出码、降级路径)、argv 组装、XDG 隔离与强制环境、整树 teardown 都
//     是真实经过(不是内存 mock)。
//   - 通用 conformance(PRD §35)对同一替身运行 runtimeconformance.Run 全套:
//     基础声明为空集 → 全部 cap:* 场景按门控不运行,events/capability_truth
//     走「未声明即拒绝」方向。
//   - 真实 opencode 表面由 TALLY_OPENCODE_LIVE=1 门控(凭据/CLI 部署前提);
//     Phase A 已对钉住版本完成 live 表面观测(见包注释)。

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

// fakeOpencodeBin 是 TestMain 构建的替身 opencode CLI 路径。
var fakeOpencodeBin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "tally-fake-opencode-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "opencode test: 建临时目录失败: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(dir)
	src, err := filepath.Abs(filepath.Join("testdata", "fakeopencode", "main.go"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "opencode test: 解析替身源码路径失败: %v\n", err)
		os.Exit(1)
	}
	exe := filepath.Join(dir, "fakeopencode.exe")
	if out, berr := exec.Command("go", "build", "-o", exe, src).CombinedOutput(); berr != nil {
		fmt.Fprintf(os.Stderr, "opencode test: 构建替身 opencode CLI 失败: %v\n%s\n", berr, out)
		os.Exit(1)
	}
	fakeOpencodeBin = exe
	os.Exit(m.Run())
}

// longMarker 是替身 CLI 识别「长运行指令」的标记:conformance 的
// LongRunInstruction 注入点(替身 spawn 子进程并等待,阻塞直到被
// Cancel/ctx/外部 kill 整树终止)。
const longMarker = "tally-conformance-long"

// ---- 测试基础设施 ------------------------------------------------------------

// fakeRecord 是替身 CLI 记录行的 Go 侧投影。
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
	return strings.EqualFold(strings.Trim(fields[0], "\" "), "fakeopencode.exe")
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

// testEnv 返回带替身凭据与记录面的基础注入环境。
func testEnv(record string) map[string]string {
	env := map[string]string{"DEEPSEEK_API_KEY": "test-key"}
	if record != "" {
		env["TALLY_FAKE_OPENCODE_RECORD"] = record
	}
	return env
}

// startSession 构造挂在替身 CLI 上的会话(缺省配置 + 注入环境,workspace
// 为新建临时目录)。
func startSession(t *testing.T, extraEnv map[string]string, cfgMod func(*Config)) (*Runtime, *Session) {
	t.Helper()
	return startSessionIn(t, t.TempDir(), extraEnv, cfgMod)
}

// startSessionIn 是 startSession 的显式 workspace 变体(cwd 效果级验证用)。
func startSessionIn(t *testing.T, ws string, extraEnv map[string]string, cfgMod func(*Config)) (*Runtime, *Session) {
	t.Helper()
	cfg := Config{BinPath: fakeOpencodeBin, Env: testEnv("")}
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
		t.Fatalf("Start: want *opencode.Session, got %T", s)
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

// ---- 能力声明诚实(空基础集;session.isolation 证据门)------------------------

func TestDescriptorDeclaresOnlyHonestCapabilities(t *testing.T) {
	d := New(Config{}).Descriptor()
	if err := d.Validate(); err != nil {
		t.Fatalf("Descriptor.Validate: %v", err)
	}
	if d.Name != DescriptorName {
		t.Fatalf("Name: want %q, got %q", DescriptorName, d.Name)
	}
	if d.Version != DescriptorVersion {
		t.Fatalf("Version 应为适配器版本 %q, got %q", DescriptorVersion, d.Version)
	}
	// run 表面的诚实清单:基础声明为空集 —— 协议词汇表中没有任何能力是
	// 一次性进程 + 后验 stdout 事件流能真实兑现的(stdout 的 NDJSON 是
	// Execute 内消费的输出,不是可独立订阅的会话事件流)。
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
			t.Fatalf("capability %s must stay undeclared by the opencode run adapter", c)
		}
	}
	if len(d.Capabilities.List()) != 0 {
		t.Fatalf("基础声明应为空集, got %v", d.Capabilities.Sorted())
	}

	// session.isolation 证据门:未持 canary 证据(零值)不得声明;置位后且
	// 仅置位后出现(与 runtimes/codex、runtimes/dsh 同一声明纪律)。
	if got := New(Config{}).capabilities(); got.Has(agentruntime.CapSessionIsolation) {
		t.Fatal("零值 Config 不得声明 session.isolation(声明以 canary 证据为门)")
	}
	gated := New(Config{SessionIsolationVerified: true}).capabilities()
	if !gated.Has(agentruntime.CapSessionIsolation) || len(gated.List()) != 1 {
		t.Fatalf("SessionIsolationVerified 置位后应且仅应声明 session.isolation, got %v", gated.Sorted())
	}
}

// 未声明的能力需求必须在 CheckStart 协商处被拒绝(requested ≠ observed;
// 每 run 新会话无 resume 面,期望 model 不构成可验证的 model.select 面)。
func TestStartRejectsUndeclaredCapabilityRequirements(t *testing.T) {
	rt := New(Config{BinPath: fakeOpencodeBin})
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
		t.Fatal("带 ModelProfile 的 Start 应被拒绝(未声明 model.select:实际生效模型不可逐 run 观察)")
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
	rt := New(Config{BinPath: fakeOpencodeBin})
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

// ---- Config.Validate(BinPath 必填 / 期望旗标取值)-----------------------------

func TestConfigValidate(t *testing.T) {
	// BinPath 必填(canary 候选不做隐式发现)。
	err := (Config{}).Validate()
	if err == nil {
		t.Fatal("空 BinPath 应被拒绝")
	}
	if !strings.Contains(err.Error(), "BinPath") {
		t.Fatalf("错误应指明 BinPath: %v", err)
	}

	// Model 观测格式 provider/model:合法值放行,其余显式拒绝。
	for _, good := range []string{"deepseek/deepseek-v4-pro", "a/b", "openai/gpt-4.1-mini"} {
		if err := (Config{BinPath: "x", Model: good}).Validate(); err != nil {
			t.Fatalf("Model %q 应合法: %v", good, err)
		}
	}
	for _, bad := range []string{"no-slash", "a/b/c", "/leading", "trailing/", "bad model", "a/b c", "x\ny"} {
		if err := (Config{BinPath: "x", Model: bad}).Validate(); err == nil {
			t.Fatalf("非法 Model %q 应被拒绝", bad)
		}
	}

	// Agent 单 token:合法值放行,其余显式拒绝。
	for _, good := range []string{"build", "plan", "review_bot"} {
		if err := (Config{BinPath: "x", Agent: good}).Validate(); err != nil {
			t.Fatalf("Agent %q 应合法: %v", good, err)
		}
	}
	for _, bad := range []string{"a/b", "bad agent", "x\ny", "'quote'"} {
		if err := (Config{BinPath: "x", Agent: bad}).Validate(); err == nil {
			t.Fatalf("非法 Agent %q 应被拒绝", bad)
		}
	}

	// Validate 是最早闸:非法期望值在 Start 处显式失败(不发进程)。
	rt := New(Config{BinPath: fakeOpencodeBin, Model: "bad model"})
	if _, err := rt.Start(context.Background(), agentruntime.StartRequest{Workspace: t.TempDir()}); err == nil {
		t.Fatal("非法 Model 应让 Start 显式失败")
	}
}

// ---- 钉版身份闸 ---------------------------------------------------------------

func TestSnapshotObservesPinnedIdentity(t *testing.T) {
	rt := New(Config{BinPath: fakeOpencodeBin})
	snap, err := rt.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if snap.Name != DescriptorName || snap.AdapterVersion != DescriptorVersion {
		t.Fatalf("snapshot identity: %+v", snap)
	}
	// 版本来自现场 `--version`(替身输出钉住版本),不是硬编码常量冒充观察。
	if snap.CLIVersion != PinnedOpenCodeVersion {
		t.Fatalf("CLIVersion: want %q, got %q", PinnedOpenCodeVersion, snap.CLIVersion)
	}
	if snap.BinaryPath != fakeOpencodeBin {
		t.Fatalf("BinaryPath: want %q, got %q", fakeOpencodeBin, snap.BinaryPath)
	}
	if len(snap.Capabilities) != 0 {
		t.Fatalf("snapshot capabilities: want 空, got %v", snap.Capabilities)
	}

	// IdentityReporter:版本是已建立的观察;Name/Model/SessionID 未观察为空,
	// 且期望配置(Config.Model)绝不回填 Model。
	_, s := startSession(t, nil, func(c *Config) { c.Model = "deepseek/deepseek-v4-pro" })
	id := s.Identity()
	if id.RuntimeVersion != PinnedOpenCodeVersion {
		t.Fatalf("Identity.RuntimeVersion: want %q, got %q", PinnedOpenCodeVersion, id.RuntimeVersion)
	}
	if id.RuntimeName != "" || id.Model != "" || id.SessionID != "" {
		t.Fatalf("未观察字段必须为空(不得回填期望配置), got %+v", id)
	}
	if !id.Observed() {
		t.Fatal("版本已观察,Observed() 应为 true")
	}
}

func TestStartFailsOnVersionMismatch(t *testing.T) {
	// 替身自报错误版本 → 钉版闸 fail-closed(适配器契约没有版本锚即不成立;
	// run 表面旗标随版本增删,--standalone 的存在性即漂移实例)。
	rt := New(Config{
		BinPath: fakeOpencodeBin,
		Env:     map[string]string{"TALLY_FAKE_OPENCODE_VERSION": "0.2.0"},
	})
	_, err := rt.Start(context.Background(), agentruntime.StartRequest{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("观察版本与钉住版本不一致时 Start 应拒绝")
	}
	if !strings.Contains(err.Error(), "0.2.0") || !strings.Contains(err.Error(), PinnedOpenCodeVersion) {
		t.Fatalf("错误应同时携带观察版本与钉住版本: %v", err)
	}
	if _, err := rt.Snapshot(context.Background()); err == nil {
		t.Fatal("Snapshot 同样受钉版闸约束")
	}
}

func TestStartFailsWhenVersionUnresolvable(t *testing.T) {
	rt := New(Config{
		BinPath: fakeOpencodeBin,
		Env:     map[string]string{"TALLY_FAKE_OPENCODE_SCENARIO": "versionfail"},
	})
	if _, err := rt.Start(context.Background(), agentruntime.StartRequest{Workspace: t.TempDir()}); err == nil {
		t.Fatal("--version 不可观察时 Start 应拒绝(fail-closed,与 codex 的尽力而为探针是有意偏离)")
	}
}

func TestStartFailsExplicitlyWithoutBinary(t *testing.T) {
	// BinPath 指向缺失路径:显式类型化错误(路径携带在错误里),不静默回落。
	missing := filepath.Join(t.TempDir(), "no-such-opencode.exe")
	rt := New(Config{BinPath: missing})
	_, err := rt.Start(context.Background(), agentruntime.StartRequest{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("BinPath 不可用时 Start 应报错")
	}
	if !strings.Contains(err.Error(), missing) {
		t.Fatalf("错误应携带缺失路径: %v", err)
	}
}

func TestParseCLIIdentityAndVersionPin(t *testing.T) {
	// 观测形态:裸版本行。
	if got := parseCLIIdentity("1.19.6\n"); got.Version != "1.19.6" {
		t.Fatalf("parseCLIIdentity: want 1.19.6, got %+v", got)
	}
	if got := parseCLIIdentity("\n  \n"); got.observed() {
		t.Fatalf("空输出应为零值观察, got %+v", got)
	}
	if err := checkVersionPin("", cliIdentity{Version: PinnedOpenCodeVersion}); err != nil {
		t.Fatalf("钉住版本一致应通过: %v", err)
	}
	err := checkVersionPin("", cliIdentity{Version: "1.18.0"})
	if err == nil || !strings.Contains(err.Error(), PinnedOpenCodeVersion) {
		t.Fatalf("不一致应拒绝且指明钉住版本: %v", err)
	}
}

// ---- Start/execute/cwd/期望旗标(经替身 CLI 的真实进程面)-----------------------

func TestExecuteHappyPathClaimAndResult(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	_, s := startSession(t, map[string]string{"TALLY_FAKE_OPENCODE_RECORD": record}, nil)
	ctx := context.Background()

	res, err := s.Execute(ctx, agentruntime.ExecuteRequest{Instruction: "add a test"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.Status != agentruntime.RunCompleted {
		t.Fatalf("Status: want completed, got %s summary=%q", res.Status, res.Summary)
	}
	// Summary 来自 stdout 事件流最后一个 text 事件。
	if res.Summary != "ok" {
		t.Fatalf("Summary 应来自 text 事件: got %q", res.Summary)
	}
	// SessionID 是适配器自造的会话编号,非空以定位;RunID 非空。
	if res.RunID == "" || res.SessionID == "" {
		t.Fatalf("claim 缺少 RunID/SessionID: %+v", res)
	}
	if !strings.HasPrefix(string(res.SessionID), "opencode-session-") {
		t.Fatalf("SessionID 应为适配器自造编号: got %q", res.SessionID)
	}
	// Usage 来自 step_finish 事件 tokens 的求和(替身 emits input=38/output=2)。
	if res.Usage != (agentruntime.Usage{InputTokens: 38, OutputTokens: 2}) {
		t.Fatalf("Usage 应来自 step_finish tokens, got %+v", res.Usage)
	}
	// 注入的凭据不得出现在 claim 文本里。
	if strings.Contains(res.Summary, "test-key") {
		t.Fatalf("claim summary 泄漏注入凭据: %q", res.Summary)
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

func TestExecutePinsWorkspaceViaProcessCwd(t *testing.T) {
	// workspace 经 spawn cwd 钉定(不使用 --dir 旗标):替身在它的 cwd 里
	// 真实读到 marker 文件并以之作为回答文本 = cwd 绑定的效果级证明。
	ws := t.TempDir()
	marker := filepath.Join(ws, "marker.txt")
	if err := os.WriteFile(marker, []byte("cwd-bound-answer"), 0o644); err != nil {
		t.Fatal(err)
	}
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	_, s := startSessionIn(t, ws, map[string]string{"TALLY_FAKE_OPENCODE_RECORD": record}, nil)

	res, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "read marker.txt"})
	if err != nil || res.Status != agentruntime.RunCompleted {
		t.Fatalf("Execute: res=%+v err=%v", res, err)
	}
	if res.Summary != "cwd-bound-answer" {
		t.Fatalf("回答应来自替身在 cwd 内真实读到的 marker 文件: got %q", res.Summary)
	}
	rec := lastRecord(t, record)
	if rec.Cwd != s.Workspace() {
		t.Fatalf("替身观测到的进程 cwd %q 应是提供的 workspace %q", rec.Cwd, s.Workspace())
	}
	if rec.Message != "read marker.txt" {
		t.Fatalf("instruction 应原样作为位置参数传入: got %q", rec.Message)
	}
	// argv 形态:[bin, run, --format, json, --pure, <instruction>]
	//(无期望 model/agent 时无 --model/--agent 旗标)。
	wantArgv := append([]string{fakeOpencodeBin, "run", "--format", "json", "--pure"}, "read marker.txt")
	if !reflect.DeepEqual(rec.Argv, wantArgv) {
		t.Fatalf("argv 应为 %v, got %v", wantArgv, rec.Argv)
	}
}

func TestModelAgentFlagsPassThrough(t *testing.T) {
	// 期望 Model/Agent 只经观测旗标 --model provider/model / --agent 传入。
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	_, s := startSession(t, map[string]string{"TALLY_FAKE_OPENCODE_RECORD": record}, func(c *Config) {
		c.Model = "deepseek/deepseek-v4-pro"
		c.Agent = "build"
	})
	res, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "add a test"})
	if err != nil || res.Status != agentruntime.RunCompleted {
		t.Fatalf("Execute: res=%+v err=%v", res, err)
	}
	rec := lastRecord(t, record)
	if rec.Model != "deepseek/deepseek-v4-pro" || rec.Agent != "build" {
		t.Fatalf("期望 model/agent 应经旗标传入: model=%q agent=%q", rec.Model, rec.Agent)
	}
	// 旗标必须在 instruction 之前。
	argv := rec.Argv
	mi := -1
	for i, a := range argv {
		if a == "--model" {
			mi = i
			break
		}
	}
	if mi < 0 || argv[len(argv)-1] != "add a test" {
		t.Fatalf("旗标应在 instruction 之前且 instruction 最后: %v", argv)
	}
}

func TestPlainOutputDegradesWithoutPanic(t *testing.T) {
	// exit 0 但 stdout 是裸文本(非 JSON 事件流):降级为原文,Completed,
	// Usage 恒为零值(没有任何 step_finish tokens 观察面)。
	_, s := startSession(t, map[string]string{"TALLY_FAKE_OPENCODE_SCENARIO": "plainok"}, nil)
	res, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "x"})
	if err != nil {
		t.Fatalf("降级路径不应是 Execute error: %v", err)
	}
	if res.Status != agentruntime.RunCompleted || res.Summary != "plain-ok" {
		t.Fatalf("裸文本应降级为 Completed+原文: %s summary=%q", res.Status, res.Summary)
	}
	if res.Usage != (agentruntime.Usage{}) {
		t.Fatalf("降级路径 Usage 应为零值, got %+v", res.Usage)
	}
}

func TestParseRunOutput(t *testing.T) {
	// 观测 NDJSON:多 step(工具回合)形态 —— 回答取最后 text,Usage 求和。
	stream := strings.Join([]string{
		`{"type":"step_start","timestamp":1,"sessionID":"ses_a","part":{"type":"step-start"}}`,
		`{"type":"tool_use","timestamp":2,"sessionID":"ses_a","part":{"type":"tool","tool":"write"}}`,
		`{"type":"step_finish","timestamp":3,"sessionID":"ses_a","part":{"type":"step-finish","reason":"tool-calls","tokens":{"total":1,"input":47,"output":83}}}`,
		`{"type":"step_start","timestamp":4,"sessionID":"ses_a","part":{"type":"step-start"}}`,
		`{"type":"text","timestamp":5,"sessionID":"ses_a","part":{"type":"text","text":"earlier"}}`,
		`{"type":"text","timestamp":6,"sessionID":"ses_a","part":{"type":"text","text":"done"}}`,
		`{"type":"step_finish","timestamp":7,"sessionID":"ses_a","part":{"type":"step-finish","reason":"stop","tokens":{"total":2,"input":46,"output":2}}}`,
		"",                // 空行容忍
		`not json at all`, // 非 JSON 行容忍
	}, "\n")
	out, sawJSON := parseRunOutput(stream)
	if out.Answer != "done" {
		t.Fatalf("回答应为最后一个 text 事件: got %q", out.Answer)
	}
	if out.Usage != (agentruntime.Usage{InputTokens: 93, OutputTokens: 85}) {
		t.Fatalf("Usage 应为 step_finish tokens 求和: got %+v", out.Usage)
	}
	if !sawJSON {
		t.Fatal("应报告 sawJSON")
	}

	// 非 JSON 输出:零值 + sawJSON=false(降级判定的依据)。
	out, sawJSON = parseRunOutput("plain text answer")
	if out.Answer != "" || out.Usage != (agentruntime.Usage{}) || sawJSON {
		t.Fatalf("非 JSON 输出应零值: %+v sawJSON=%v", out, sawJSON)
	}
}

// ---- 失败分类(claim 纪律)------------------------------------------------------

func TestNonZeroExitSurfacesAsFailedClaim(t *testing.T) {
	_, s := startSession(t, map[string]string{"TALLY_FAKE_OPENCODE_SCENARIO": "exitfail"}, nil)
	res, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "x"})
	if err != nil {
		t.Fatalf("非零退出是 RunFailed claim: %v", err)
	}
	if res.Status != agentruntime.RunFailed || !strings.Contains(res.Summary, "exit status 7") {
		t.Fatalf("want failed + 退出码, got %s/%q", res.Status, res.Summary)
	}
	// 崩溃前事件流里的部分 text 不得冒充最终回答。
	if strings.Contains(res.Summary, "partial answer") {
		t.Fatalf("失败 claim 的 summary 不得携带 stdout 部分输出: %q", res.Summary)
	}
}

func TestExitZeroWithoutAnswerFailsSafe(t *testing.T) {
	// exit 0 但无任何输出:契约(stdout 携带回答)未兑现 → FAIL-SAFE 判败,
	// 不猜测成功。
	_, s := startSession(t, map[string]string{"TALLY_FAKE_OPENCODE_SCENARIO": "emptyok"}, nil)
	res, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "x"})
	if err != nil {
		t.Fatalf("契约不兑现是 RunFailed claim: %v", err)
	}
	if res.Status != agentruntime.RunFailed {
		t.Fatalf("Status: want failed, got %s summary=%q", res.Status, res.Summary)
	}
}

// ---- 隔离与出口硬化 --------------------------------------------------------------

func TestPerSessionDirsIsolationAndForcedEnv(t *testing.T) {
	// 模拟「继承环境携带 owner 的 XDG/temp 重定向、Config.Env 试图同名覆盖」
	// 的部署面:适配器必须强制 per-session 目录,压倒继承环境与 Config.Env。
	ownerCfg := filepath.Join(t.TempDir(), "owner-config")
	ownerData := filepath.Join(t.TempDir(), "owner-data")
	if err := os.MkdirAll(ownerCfg, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(ownerData, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", ownerCfg)
	t.Setenv("XDG_DATA_HOME", ownerData)
	t.Setenv("TMP", ownerData)
	// 模拟继承链携带的陈旧 PWD(真实观测面:go test/bash 的 PWD 与会话
	// workspace 不同,真实 opencode 按 PWD 解析项目目录 —— 必须被压倒)。
	t.Setenv("PWD", ownerCfg)
	t.Setenv("OLDPWD", ownerCfg)

	record := filepath.Join(t.TempDir(), "calls.jsonl")
	rt, s1 := startSession(t, map[string]string{
		"TALLY_FAKE_OPENCODE_RECORD": record,
		"XDG_DATA_HOME":              ownerData, // Config.Env 中的同名值同样被压倒
	}, nil)
	if _, err := s1.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "x"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	rec := lastRecord(t, record)

	// XDG 四根全部指向会话目录树,绝非 owner 目录。
	root := s1.dirs.root
	if root == "" || !strings.Contains(filepath.Base(root), "tally-opencode-session-") {
		t.Fatalf("会话应有 tally-opencode-session-* 隔离根: got %q", root)
	}
	for _, tc := range []struct{ name, got, owner, sub string }{
		{"XDG_CONFIG_HOME", rec.XDGConfig, ownerCfg, "config"},
		{"XDG_DATA_HOME", rec.XDGData, ownerData, "data"},
		{"XDG_CACHE_HOME", rec.XDGCache, "", "cache"},
		{"XDG_STATE_HOME", rec.XDGState, "", "state"},
	} {
		if tc.got != filepath.Join(root, tc.sub) {
			t.Fatalf("%s 应为会话目录 %s, got %q(owner=%q)", tc.name, filepath.Join(root, tc.sub), tc.got, tc.owner)
		}
		if tc.owner != "" && tc.got == tc.owner {
			t.Fatalf("%s 不得指向 owner 目录 %q", tc.name, tc.owner)
		}
	}
	// TMP/TEMP 强制指向会话 tmp 子目录(平台级 temp 重定向)。
	if rec.TMP != filepath.Join(root, "tmp") || rec.TEMP != filepath.Join(root, "tmp") {
		t.Fatalf("TMP/TEMP 应强制为会话 tmp 目录 %q, got TMP=%q TEMP=%q", filepath.Join(root, "tmp"), rec.TMP, rec.TEMP)
	}
	// PWD/OLDPWD 钉到 workspace(live 实测:真实 opencode 按 PWD 解析项目
	// 目录,进程 cwd 钉定必须配合 PWD 对齐才真实生效)。
	if rec.PWD != s1.Workspace() || rec.OLDPWD != s1.Workspace() {
		t.Fatalf("PWD/OLDPWD 应钉到 workspace %q, got PWD=%q OLDPWD=%q", s1.Workspace(), rec.PWD, rec.OLDPWD)
	}
	// 隔离目录树真实存在。
	if _, err := os.Stat(rec.XDGData); err != nil {
		t.Fatalf("per-session data 目录应真实存在: %v", err)
	}

	// 第二个会话获得不同的全新目录树(会话边界,不共享)。
	record2 := filepath.Join(t.TempDir(), "calls2.jsonl")
	_, s2 := startSession(t, map[string]string{"TALLY_FAKE_OPENCODE_RECORD": record2}, nil)
	if _, err := s2.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "x"}); err != nil {
		t.Fatalf("Execute(s2): %v", err)
	}
	rec2 := lastRecord(t, record2)
	if rec2.XDGData == "" || rec2.XDGData == rec.XDGData {
		t.Fatalf("两个会话应获得不同的全新目录树: %q vs %q", rec.XDGData, rec2.XDGData)
	}

	// Close 删除本会话整棵目录树(尽力而为;等待短暂收敛后断言)。
	if err := s1.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(root); os.IsNotExist(err) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("Close 后 per-session 目录树 %s 未被删除", root)
		}
		time.Sleep(50 * time.Millisecond)
	}
	_ = rt
}

func TestAuthEnvInjectedNotInLogs(t *testing.T) {
	// 凭据经 Config.Env 注入(providers list 观测的环境变量识别面),替身
	// 记录 APIKeySet;失败路径的 summary 携带 provider 错误原文时经
	// logpipe.RedactFine 脱敏,注入凭据不得出现在 claim。
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	_, s := startSession(t, map[string]string{"TALLY_FAKE_OPENCODE_RECORD": record}, nil)
	if _, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "x"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if rec := lastRecord(t, record); !rec.APIKeySet {
		t.Fatal("Config.Env 注入的 DEEPSEEK_API_KEY 应对子进程可见(APIKeySet=true)")
	}

	// 反面:未注入凭据时子进程不可见(隔离目录树内 auth.json 恒空,凭据
	// 只经显式 Config.Env)。中和测试进程继承环境中的同名变量(本机全局
	// 可能持有 DEEPSEEK_API_KEY;适配器语义是继承+注入,不清洗 —— 与
	// runtimes/dsh 同一语义,测试面负责把继承面归零)。
	t.Setenv("DEEPSEEK_API_KEY", "")
	record2 := filepath.Join(t.TempDir(), "calls2.jsonl")
	cfg := Config{BinPath: fakeOpencodeBin, Env: map[string]string{"TALLY_FAKE_OPENCODE_RECORD": record2}}
	rt := New(cfg)
	s2, err := rt.Start(context.Background(), agentruntime.StartRequest{Workspace: t.TempDir()})
	if err != nil {
		t.Fatalf("Start(keyless): %v", err)
	}
	defer func() { _ = s2.Close(context.Background()) }()
	if _, err := s2.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "x"}); err != nil {
		t.Fatalf("Execute(keyless): %v", err)
	}
	if rec := lastRecord(t, record2); rec.APIKeySet {
		t.Fatal("未注入凭据时子进程不得见到 DEEPSEEK_API_KEY")
	}
}

// Failure diagnostics cross a durable claim boundary. Secrets emitted by an
// external CLI must be redacted before they can enter AgentRunResult/log/UI.
func TestFailureSummaryRedactsSecrets(t *testing.T) {
	r0 := &run{id: "run-redact", status: agentruntime.RunRunning}
	s := &Session{session: "opencode-redact"}
	secret := "sk-abcdefghijklmnopqrstuvwxyz1234567890"
	res := s.finishRun(r0, agentruntime.RunFailed, "provider failed api_key="+secret, agentruntime.Usage{})
	if strings.Contains(res.Summary, secret) {
		t.Fatalf("failure summary leaked secret: %q", res.Summary)
	}
	if !strings.Contains(res.Summary, "[REDACTED]") {
		t.Fatalf("failure summary should contain redaction marker: %q", res.Summary)
	}
}

// ---- 失败/取消/崩溃/关闭(claim 纪律 + 整树 teardown)-----------------------------

// TestCancelKillsWholeProcessTree 是整树终止证明:Cancel/ctx 取消必须整树
// 终止 —— 替身与其子进程全部消失,无孤儿残留。
//
// 双重证据:(a) WaitDelay 被刻意抬高到 30s —— 若子进程在整树终止后存活,
// 它持有的 stdout 管道会把 Execute 拖到 WaitDelay 兜底,waitForStatus 的
// 20s 上限必然先超时;快速收敛即证明持管道的子进程确实死了。
// (b) tasklist 镜像名核对确认子进程 pid 不再属于任何 fakeopencode.exe。
func TestCancelKillsWholeProcessTree(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	_, s := startSession(t, map[string]string{"TALLY_FAKE_OPENCODE_RECORD": record}, func(c *Config) { c.WaitDelay = 30 * time.Second })
	ctx := context.Background()

	outcome := make(chan executeOutcome, 1)
	go func() {
		res, err := s.Execute(ctx, agentruntime.ExecuteRequest{Instruction: longMarker + " run"})
		outcome <- executeOutcome{res, err}
	}()

	// 等替身记下子进程 PID(spawn 完成后),再取消:确保整树里确实有子进程
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

	// 核心:子进程随替身一并消失(整树终止),无孤儿。
	waitForGone(t, rec.ChildPID)
}

func TestExecuteContextCancelIsRunCancelled(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	_, s := startSession(t, map[string]string{"TALLY_FAKE_OPENCODE_RECORD": record}, nil)
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
	// 清理可能的替身孤儿子进程(ctx 取消同样走整树终止;此处兜底)。
	if rec := lastRecordFor(record); rec.ChildPID > 0 {
		waitForGone(t, rec.ChildPID)
	}
}

// PRD §35「no orphan process in cancellation/timeout tests」:ctx 超时(时限
// 到期触发 Cancel 路径)必须收敛 RunCancelled,且整树终止后子进程零残留。
func TestContextTimeoutLeavesNoOrphan(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	_, s := startSession(t, map[string]string{"TALLY_FAKE_OPENCODE_RECORD": record}, func(c *Config) { c.WaitDelay = 2 * time.Second })

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
	// 孤儿零残留:替身子进程在整树终止后必须消失(真实进程树取证)。
	if rec.ChildPID > 0 {
		waitForGone(t, rec.ChildPID)
	}
}

func TestProcessCrashHandledAndSessionResponsive(t *testing.T) {
	// PRD §35 crash 场景:外部单进程 kill 替身(非 Cancel、非整树)必须归为
	// RunFailed,且 Execute 必须收敛,会话不变成僵尸。kill 前先等替身记下
	// 子进程 PID:保证孤儿真实存在并持有输出管道 —— 收敛只能经 WaitDelay
	// 兜底,不得挂死、不得伪装成取消或成功。
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	_, s := startSession(t, map[string]string{"TALLY_FAKE_OPENCODE_RECORD": record}, func(c *Config) { c.WaitDelay = 2 * time.Second })
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
		t.Fatal("in-flight run 应可经 RunProcess 观测到底层 CLI 进程")
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
	_, s := startSession(t, map[string]string{"TALLY_FAKE_OPENCODE_RECORD": record}, nil)
	ctx := context.Background()

	outcome := make(chan executeOutcome, 1)
	go func() {
		res, err := s.Execute(ctx, agentruntime.ExecuteRequest{Instruction: longMarker + " run"})
		outcome <- executeOutcome{res, err}
	}()
	waitForChildPID(t, record)

	root := s.dirs.root
	if root == "" {
		t.Fatal("会话应有 per-session 隔离目录树")
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
	// 隔离目录树被删除。
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(root); os.IsNotExist(err) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("Close 后目录树 %s 未被删除", root)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// ---- PRD §35 通用 conformance(全场景,零声明 → cap:* 全部不运行)---------------

// TestOpencodeRuntimeConformance 对挂在替身 CLI 上的 OpenCodeRuntime 跑完整
// runtimeconformance 套件:start / cwd / execute / result / events(未声明
// 拒绝方向)/ cancel(ctx 路径)/ crash / workspace / usage / capability_truth。
// 基础声明为空集 → 全部能力专项场景按门控不得运行(声明真实性)。
func TestOpencodeRuntimeConformance(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	rt := New(Config{
		BinPath:   fakeOpencodeBin,
		Env:       testEnv(record),
		WaitDelay: 2 * time.Second, // crash 注入面的孤儿管道兜底(测试用短值)
	})
	rep := runtimeconformance.Run(t, rt, runtimeconformance.Options{
		Probes: runtimeconformance.Probes{
			LongRunInstruction: longMarker + " run",
			CrashRun: func(ctx context.Context, sess agentruntime.Session, longRunInstruction string) (agentruntime.AgentRunResult, error) {
				oc, ok := sess.(*Session)
				if !ok {
					return agentruntime.AgentRunResult{}, fmt.Errorf("crash probe: want *opencode.Session, got %T", sess)
				}
				outcome := make(chan executeOutcome, 1)
				go func() {
					res, err := sess.Execute(ctx, agentruntime.ExecuteRequest{Instruction: longRunInstruction})
					outcome <- executeOutcome{res, err}
				}()
				// 等替身记下子进程 PID:保证 kill 时孤儿真实存在并持有输出
				// 管道(WaitDelay 兜底路径被确定性覆盖)。
				waitForChildPID(t, record)
				// 未声明 event.streaming,订阅不到 run.started:经诊断缝定位。
				deadline := time.Now().Add(10 * time.Second)
				var id agentruntime.RunID
				for {
					got, ok := oc.InFlightRunID()
					if ok {
						id = got
						break
					}
					if time.Now().After(deadline) {
						return agentruntime.AgentRunResult{}, fmt.Errorf("crash probe: run 未进入 in-flight")
					}
					time.Sleep(10 * time.Millisecond)
				}
				proc, ok := oc.RunProcess(id)
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
				oc, ok := sess.(*Session)
				if !ok {
					return fmt.Errorf("workspace probe: want *opencode.Session, got %T", sess)
				}
				rec := lastRecordFor(record)
				if rec.Cwd != oc.Workspace() {
					return fmt.Errorf("替身观测到的进程 cwd %q 不是提供的 workspace %q", rec.Cwd, oc.Workspace())
				}
				return nil
			},
		},
	})
	if !rep.OK() {
		t.Fatalf("OpenCodeRuntime conformance 未通过:\n%s", rep.FailureText())
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
