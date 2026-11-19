package grok

// 测试策略(对齐 runtimes/codex、runtimes/dsh 的替身 CLI 模式):
//
//   - 全部进程面经「替身 grok CLI」驱动:TestMain 从
//     testdata/fakegrok/main.go 现场构建替身二进制,适配器对它 spawn/读取/
//     取消/整树终止 —— 进程边界、headless 契约(stdout=最终回答、退出码、
//     keyless「Not signed in」)、argv 组装(--single= 附值形态)、白名单
//     环境与 GROK_HOME 隔离、整树 teardown 都是真实经过(不是内存 mock)。
//   - 通用 conformance(PRD §35)对同一替身运行 runtimeconformance.Run 全套:
//     基础声明为空集 → 全部 cap:* 场景按门控不运行,events/capability_truth
//     走「未声明即拒绝」方向。
//   - 真实 grok 表面的 live 探针不在包测试内(本机无凭据;观测事实见包注释,
//     真实执行收据由适配器交付流程另行记录)。

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

// fakeGrokBin 是 TestMain 构建的替身 grok CLI 路径。
var fakeGrokBin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "tally-fake-grok-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "grok test: 建临时目录失败: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(dir)
	src, err := filepath.Abs(filepath.Join("testdata", "fakegrok", "main.go"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "grok test: 解析替身源码路径失败: %v\n", err)
		os.Exit(1)
	}
	exe := filepath.Join(dir, "fakegrok.exe")
	if out, berr := exec.Command("go", "build", "-o", exe, src).CombinedOutput(); berr != nil {
		fmt.Fprintf(os.Stderr, "grok test: 构建替身 grok CLI 失败: %v\n%s\n", berr, out)
		os.Exit(1)
	}
	fakeGrokBin = exe
	os.Exit(m.Run())
}

// longMarker 是替身 CLI 识别「长运行指令」的标记:conformance 的
// LongRunInstruction 注入点(替身 spawn 子进程并等待,阻塞直到被
// Cancel/ctx/外部 kill 整树终止)。
const longMarker = "tally-conformance-long"

// ---- 测试基础设施 ------------------------------------------------------------

// fakeRecord 是替身 CLI 记录行的 Go 侧投影(只含 env 存在性,不含 env 值)。
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

// waitForChildPID 轮询记录文件直到替身 CLI 记下子进程 PID(长任务已
// 进入「spawn 完子进程并等待」阶段,整树终止的取证窗口已打开)。
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

// processAlive 报告 pid 是否仍是我们的替身 CLI 进程。Windows 的 pid 在
// 进程死后会被内核快速复用:单纯 FindProcess 成功可能只是复用了该 pid 的
// 别的进程(测试实测会造成假阳性),因此 Windows 侧用 tasklist 按 PID 过滤
// 并核对镜像名 —— 复用即视为「已消失」。
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
	return strings.EqualFold(strings.Trim(fields[0], "\" "), "fakegrok.exe")
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

// testEnv 返回带替身凭据门的基础注入环境(XAI_API_KEY 是观测的认证 env 名)。
func testEnv(record string) map[string]string {
	env := map[string]string{"XAI_API_KEY": "test-key"}
	if record != "" {
		env["TALLY_FAKE_GROK_RECORD"] = record
	}
	return env
}

// startSession 构造挂在替身 CLI 上的会话(缺省配置 + 注入环境,
// workspace 为新建临时目录)。
func startSession(t *testing.T, extraEnv map[string]string, cfgMod func(*Config)) (*Runtime, *Session) {
	t.Helper()
	return startSessionIn(t, t.TempDir(), extraEnv, cfgMod)
}

// startSessionIn 是 startSession 的显式 workspace 变体(cwd 效果级验证用)。
func startSessionIn(t *testing.T, ws string, extraEnv map[string]string, cfgMod func(*Config)) (*Runtime, *Session) {
	t.Helper()
	cfg := Config{BinPath: fakeGrokBin, Env: testEnv("")}
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
		t.Fatalf("Start: want *grok.Session, got %T", s)
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
	// --single 一次性表面(诚实清单):基础声明为空集 —— 协议词汇表中没有
	// 任何能力是该表面(在未做真实回合观测前)能真实兑现的。
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
			t.Fatalf("capability %s must stay undeclared by the grok headless adapter", c)
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

// 未声明的能力需求必须在 CheckStart 协商处被拒绝(无 resume/model.select 面)。
func TestStartRejectsUndeclaredCapabilityRequirements(t *testing.T) {
	rt := New(Config{BinPath: fakeGrokBin})
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
		t.Fatal("带 ModelProfile 的 Start 应被拒绝(未声明 model.select:实际生效模型不可观察)")
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
	rt := New(Config{BinPath: fakeGrokBin})
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

// ---- AC:Config.Validate(最早配置闸)-------------------------------------------

func TestConfigValidateRejectsInvalidConfiguration(t *testing.T) {
	// BinPath 显式必填(npm dist-tag latest 已观测失准,不做隐式发现)。
	if err := (Config{}).Validate(); err == nil {
		t.Fatal("空 BinPath 应被拒绝")
	}
	if err := (Config{BinPath: "   "}).Validate(); err == nil {
		t.Fatal("纯空白 BinPath 应被拒绝")
	}
	// 纯空白 Model 拒绝(空串 = 不传旗标,合法)。
	if err := (Config{BinPath: fakeGrokBin, Model: "   "}).Validate(); err == nil {
		t.Fatal("纯空白 Model 应被拒绝")
	}
	if err := (Config{BinPath: fakeGrokBin, Model: ""}).Validate(); err != nil {
		t.Fatalf("空 Model 应合法(不传 -m): %v", err)
	}
	// PermissionMode 只接受观测枚举(--help 逐字),其它值拒绝。
	for _, bad := range []string{"yolo", "workspace-write", "Default"} {
		if err := (Config{BinPath: fakeGrokBin, PermissionMode: bad}).Validate(); err == nil {
			t.Fatalf("非法 PermissionMode %q 应被拒绝", bad)
		}
	}
	for _, good := range []string{"", "default", "acceptEdits", "auto", "dontAsk", "bypassPermissions", "plan"} {
		if err := (Config{BinPath: fakeGrokBin, PermissionMode: good}).Validate(); err != nil {
			t.Fatalf("合法 PermissionMode %q 不应被拒绝: %v", good, err)
		}
	}
	// Validate 不发起进程:BinPath 指向不存在的路径时 Validate 仍通过
	// (路径存在性闸在 Start,见 TestStartFailsExplicitlyWithoutBinary)。
	if err := (Config{BinPath: filepath.Join(t.TempDir(), "no-such-grok.exe")}).Validate(); err != nil {
		t.Fatalf("Validate 不应做路径观察: %v", err)
	}
}

// ---- AC:钉版身份闸 ------------------------------------------------------------

func TestSnapshotObservesPinnedIdentity(t *testing.T) {
	rt := New(Config{BinPath: fakeGrokBin})
	snap, err := rt.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if snap.Name != DescriptorName || snap.AdapterVersion != DescriptorVersion {
		t.Fatalf("snapshot identity: %+v", snap)
	}
	// 版本来自现场 `--version`(替身输出钉住版本),不是硬编码常量冒充观察。
	if snap.CLIVersion != PinnedGrokVersion {
		t.Fatalf("CLIVersion: want %q, got %q", PinnedGrokVersion, snap.CLIVersion)
	}
	if snap.BinaryPath != fakeGrokBin {
		t.Fatalf("BinaryPath: want %q, got %q", fakeGrokBin, snap.BinaryPath)
	}
	if len(snap.Capabilities) != 0 {
		t.Fatalf("snapshot capabilities: want 空, got %v", snap.Capabilities)
	}

	// IdentityReporter:版本/名字是已建立的观察;Model/SessionID 未观察为空,
	// 且期望配置(Config.Model)绝不回填 Model(AGENTS.md §13)。
	_, s := startSession(t, nil, func(c *Config) { c.Model = "grok-4.7" })
	id := s.Identity()
	if id.RuntimeVersion != PinnedGrokVersion {
		t.Fatalf("Identity.RuntimeVersion: want %q, got %q", PinnedGrokVersion, id.RuntimeVersion)
	}
	if id.RuntimeName != "grok" {
		t.Fatalf("Identity.RuntimeName: want 自报的 grok, got %q", id.RuntimeName)
	}
	if id.Model != "" || id.SessionID != "" {
		t.Fatalf("未观察字段必须为空(不得回填期望配置), got %+v", id)
	}
	if !id.Observed() {
		t.Fatal("版本已观察,Observed() 应为 true")
	}
}

func TestStartFailsOnVersionMismatch(t *testing.T) {
	// 替身自报错误版本 → 钉版闸 fail-closed(快速演进表面的契约没有版本锚
	// 即不成立)。
	rt := New(Config{
		BinPath: fakeGrokBin,
		Env:     map[string]string{"TALLY_FAKE_GROK_VERSION": "grok 1.1.0 (ffffffffffff)"},
	})
	_, err := rt.Start(context.Background(), agentruntime.StartRequest{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("观察版本与钉住版本不一致时 Start 应拒绝")
	}
	if !strings.Contains(err.Error(), "1.1.0") || !strings.Contains(err.Error(), PinnedGrokVersion) {
		t.Fatalf("错误应同时携带观察版本与钉住版本: %v", err)
	}
	if _, err := rt.Snapshot(context.Background()); err == nil {
		t.Fatal("Snapshot 同样受钉版闸约束")
	}
}

func TestStartFailsWhenVersionUnresolvable(t *testing.T) {
	rt := New(Config{
		BinPath: fakeGrokBin,
		Env:     map[string]string{"TALLY_FAKE_GROK_SCENARIO": "versionfail"},
	})
	if _, err := rt.Start(context.Background(), agentruntime.StartRequest{Workspace: t.TempDir()}); err == nil {
		t.Fatal("--version 不可观察时 Start 应拒绝(fail-closed,与 codex 的尽力而为探针是有意偏离)")
	}
}

func TestParseCLIIdentityAndVersionPin(t *testing.T) {
	// 观测形态:两字段 + 括号 commit。
	got := parseCLIIdentity("grok 1.0.87 (929e6dd187b1)\n")
	if got.Name != "grok" || got.Version != "1.0.87" || got.Commit != "929e6dd187b1" {
		t.Fatalf("parseCLIIdentity: got %+v", got)
	}
	if !got.observed() {
		t.Fatal("观测形态应可辨认")
	}
	// 字段缺失时只填可辨认部分,不猜测。
	partial := parseCLIIdentity("grok 1.0.87\n")
	if partial.Version != "1.0.87" || partial.Commit != "" {
		t.Fatalf("部分形态应只填可辨认字段: %+v", partial)
	}
	if got := parseCLIIdentity("\n  \n"); got.observed() {
		t.Fatalf("空输出应为零值观察, got %+v", got)
	}
	if got := parseCLIIdentity("Grok Build TUI"); got.observed() {
		t.Fatalf("无可辨认版本的首行应为零值观察, got %+v", got)
	}
	if err := checkVersionPin("", cliIdentity{Version: PinnedGrokVersion}); err != nil {
		t.Fatalf("钉住版本一致应通过: %v", err)
	}
	err := checkVersionPin("", cliIdentity{Version: "1.1.0"})
	if err == nil || !strings.Contains(err.Error(), PinnedGrokVersion) {
		t.Fatalf("不一致应拒绝且指明钉住版本: %v", err)
	}
}

// ---- AC:二进制路径闸(BinPath 显式必填,缺失/错误即刻拒绝)---------------------

func TestStartFailsExplicitlyWithoutBinary(t *testing.T) {
	// BinPath 指向不存在的路径 → 类型化错误(errors.Is os.ErrNotExist),
	// 绝不开会话。
	missing := filepath.Join(t.TempDir(), "no-such-grok.exe")
	rt := New(Config{BinPath: missing})
	_, err := rt.Start(context.Background(), agentruntime.StartRequest{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("二进制缺失时 Start 应报错")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("缺失路径错误应可经 errors.Is(os.ErrNotExist) 辨认: %v", err)
	}
	if !strings.Contains(err.Error(), missing) {
		t.Fatalf("错误应携带缺失路径: %v", err)
	}
	// BinPath 指向目录 → 显式拒绝。
	dirRt := New(Config{BinPath: t.TempDir()})
	if _, err := dirRt.Start(context.Background(), agentruntime.StartRequest{Workspace: t.TempDir()}); err == nil {
		t.Fatal("BinPath 指向目录应报错")
	}
}

// ---- AC:Start/execute/cwd(经替身 CLI 的真实进程面)----------------------------

func TestExecuteHappyPathClaimAndResult(t *testing.T) {
	_, s := startSession(t, nil, nil)
	ctx := context.Background()

	res, err := s.Execute(ctx, agentruntime.ExecuteRequest{Instruction: "add a test"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.Status != agentruntime.RunCompleted {
		t.Fatalf("Status: want completed, got %s summary=%q", res.Status, res.Summary)
	}
	if res.Summary != "ok" {
		t.Fatalf("Summary 应来自 stdout 最终回答: got %q", res.Summary)
	}
	// SessionID 是适配器自造的会话编号(grok-session-NNNNNN),非空以定位;
	// RunID 非空。
	if res.RunID == "" || res.SessionID == "" {
		t.Fatalf("claim 缺少 RunID/SessionID: %+v", res)
	}
	if !strings.HasPrefix(string(res.SessionID), "grok-session-") {
		t.Fatalf("SessionID 应为适配器自造编号: got %q", res.SessionID)
	}
	// Usage 不解析:plain 表面不自报用量,schema 未观测,解析即发明(宁缺毋假)。
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
	if err := s.Cancel(ctx, "run-bogus"); !errors.Is(err, agentruntime.ErrRunNotFound) {
		t.Fatalf("Cancel(unknown): want ErrRunNotFound, got %v", err)
	}
	if _, err := s.Execute(ctx, agentruntime.ExecuteRequest{Instruction: ""}); !errors.Is(err, agentruntime.ErrNoInstruction) {
		t.Fatalf("空指令: want ErrNoInstruction, got %v", err)
	}
}

func TestExecutePinsWorkspaceViaProcessCwd(t *testing.T) {
	// workspace 经 spawn cwd 钉定(--cwd 旗标效果未观测,不使用;inspect
	// 观测 workspaceRoot = 进程 cwd):替身在它的 cwd 里真实读到 marker 文件
	// = cwd 绑定的效果级证明。
	ws := t.TempDir()
	marker := filepath.Join(ws, "marker.txt")
	if err := os.WriteFile(marker, []byte("cwd-bound-answer"), 0o644); err != nil {
		t.Fatal(err)
	}
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	_, s := startSessionIn(t, ws, map[string]string{"TALLY_FAKE_GROK_RECORD": record}, nil)

	res, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "read marker.txt"})
	if err != nil || res.Status != agentruntime.RunCompleted {
		t.Fatalf("Execute: res=%+v err=%v", res, err)
	}
	if res.Summary != "cwd-bound-answer" {
		t.Fatalf("最终回答应来自替身在 cwd 内真实读到的 marker 文件: got %q", res.Summary)
	}
	rec := lastRecord(t, record)
	if rec.Cwd != s.Workspace() {
		t.Fatalf("替身观测到的进程 cwd %q 应是提供的 workspace %q(workspaceRoot=process.cwd())", rec.Cwd, s.Workspace())
	}
	if rec.Prompt != "read marker.txt" {
		t.Fatalf("instruction 应原样作为 --single= 值传入: got %q", rec.Prompt)
	}
	// argv 形态:[bin, --no-auto-update, --single=<instruction>](零 overlay:
	// 未配置 model/permission 时不传对应旗标)。
	wantArgv := []string{fakeGrokBin, "--no-auto-update", "--single=read marker.txt"}
	if !reflect.DeepEqual(rec.Argv, wantArgv) {
		t.Fatalf("argv 应为 %v, got %v", wantArgv, rec.Argv)
	}
}

func TestModelAndPermissionModePassThroughObservedFlags(t *testing.T) {
	// 期望模型/审批模式只经观测旗标 -m / --permission-mode 透传;空 = 不传
	// 旗标(零 overlay)。
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	_, s := startSession(t, map[string]string{"TALLY_FAKE_GROK_RECORD": record}, func(c *Config) {
		c.Model = "grok-4.7"
		c.PermissionMode = "acceptEdits"
	})
	res, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "add a test"})
	if err != nil || res.Status != agentruntime.RunCompleted {
		t.Fatalf("Execute: res=%+v err=%v", res, err)
	}
	rec := lastRecord(t, record)
	if rec.Model != "grok-4.7" || rec.PermissionMode != "acceptEdits" {
		t.Fatalf("期望配置应经观测旗标生效: model=%q permission=%q", rec.Model, rec.PermissionMode)
	}
	wantArgv := []string{fakeGrokBin, "--no-auto-update", "-m", "grok-4.7", "--permission-mode", "acceptEdits", "--single=add a test"}
	if !reflect.DeepEqual(rec.Argv, wantArgv) {
		t.Fatalf("argv 应为 %v, got %v", wantArgv, rec.Argv)
	}

	// 以 "-" 开头的指令文本经 --single= 附值形态安全传入(观测:裸 -p 值形态
	// 会被 clap 误解析)。
	res2, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "-leading dash task"})
	if err != nil || res2.Status != agentruntime.RunCompleted {
		t.Fatalf("Execute(dash): res=%+v err=%v", res2, err)
	}
	rec2 := lastRecord(t, record)
	if rec2.Prompt != "-leading dash task" {
		t.Fatalf("以 - 开头的指令应原样到达替身: got %q", rec2.Prompt)
	}
}

// ---- AC:隔离与出口硬化 ---------------------------------------------------------

func TestPerSessionHomeIsolationAndWhitelistEnv(t *testing.T) {
	// 模拟「继承环境试图恢复默认 home、携带其他 provider 认证」的部署面:
	// 适配器必须强制 per-session GROK_HOME(压倒继承环境与 Config.Env 同名值),
	// 且 owner 环境中的其他 provider 认证不得进入子进程(白名单制)。
	ownerHome := filepath.Join(t.TempDir(), "owner-grok-home")
	if err := os.MkdirAll(ownerHome, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GROK_HOME", ownerHome)
	t.Setenv("DEEPSEEK_API_KEY", "owner-deepseek-secret-not-for-grok")

	record := filepath.Join(t.TempDir(), "calls.jsonl")
	rt, s1 := startSession(t, map[string]string{
		"TALLY_FAKE_GROK_RECORD": record,
		"GROK_HOME":              ownerHome, // Config.Env 中的同名值同样被压倒
	}, nil)
	if _, err := s1.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "x"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	rec := lastRecord(t, record)

	// per-session home:全新临时目录,绝非 owner home。
	if rec.GrokHome == "" || rec.GrokHome == ownerHome {
		t.Fatalf("GROK_HOME 应为全新 per-session 目录, got %q(owner=%q)", rec.GrokHome, ownerHome)
	}
	if _, err := os.Stat(rec.GrokHome); err != nil {
		t.Fatalf("per-session home 应真实存在: %v", err)
	}
	// 白名单:注入的 XAI_API_KEY 到达子进程;owner 的 DEEPSEEK_API_KEY 不到达
	// (结构性白名单,不靠覆盖)。
	if !rec.XAIAPIKeySet {
		t.Fatal("Config.Env 注入的 XAI_API_KEY 应到达子进程(观测的认证 env 名)")
	}
	if rec.DeepSeekKeySet {
		t.Fatal("owner 环境的 DEEPSEEK_API_KEY 不得进入子进程(spawn env 白名单)")
	}

	// 第二个会话获得不同的全新 home(会话边界,不共享)。
	record2 := filepath.Join(t.TempDir(), "calls2.jsonl")
	_, s2 := startSession(t, map[string]string{"TALLY_FAKE_GROK_RECORD": record2}, nil)
	if _, err := s2.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "x"}); err != nil {
		t.Fatalf("Execute(s2): %v", err)
	}
	rec2 := lastRecord(t, record2)
	if rec2.GrokHome == "" || rec2.GrokHome == rec.GrokHome {
		t.Fatalf("两个会话应获得不同的全新 home: %q vs %q", rec.GrokHome, rec2.GrokHome)
	}

	// Close 删除本会话 home(尽力而为;等待短暂收敛后断言)。
	if err := s1.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(rec.GrokHome); os.IsNotExist(err) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("Close 后 per-session home %s 未被删除", rec.GrokHome)
		}
		time.Sleep(50 * time.Millisecond)
	}
	_ = rt
}

// ---- 失败/取消/崩溃/关闭(claim 纪律 + 整树 teardown)---------------------------

func TestMissingCredentialIsFailedClaimNotError(t *testing.T) {
	// 观测契约:缺 XAI_API_KEY → exit 1 + stdout 空 + stderr「Not signed in. ...」。
	// 适配器必须把它如实归类为 RunFailed claim(失败证据携带 stderr 原文),
	// 既不是成功也不是 Execute error。
	_, s := startSession(t, map[string]string{"XAI_API_KEY": ""}, nil)
	res, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "reply ok"})
	if err != nil {
		t.Fatalf("凭据缺失是 run 级失败,不是 Execute error: %v", err)
	}
	if res.Status != agentruntime.RunFailed {
		t.Fatalf("Status: want failed, got %s summary=%q", res.Status, res.Summary)
	}
	if !strings.Contains(res.Summary, "Not signed in") {
		t.Fatalf("summary 应携带 stderr 错误原文: %q", res.Summary)
	}
}

func TestNonZeroExitSurfacesAsFailedClaim(t *testing.T) {
	_, s := startSession(t, map[string]string{"TALLY_FAKE_GROK_SCENARIO": "exitfail"}, nil)
	res, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "x"})
	if err != nil {
		t.Fatalf("非零退出是 RunFailed claim: %v", err)
	}
	if res.Status != agentruntime.RunFailed || !strings.Contains(res.Summary, "exit status 7") {
		t.Fatalf("want failed + 退出码, got %s/%q", res.Status, res.Summary)
	}
	// 崩溃前的部分输出不得冒充最终回答。
	if strings.Contains(res.Summary, "partial answer") {
		t.Fatalf("失败 claim 的 summary 不得携带 stdout 部分输出: %q", res.Summary)
	}
}

func TestExitZeroWithoutAnswerFailsSafe(t *testing.T) {
	// exit 0 但无最终回答:契约(stdout=最终回答)未兑现 → FAIL-SAFE 判败,
	// 不猜测成功。
	_, s := startSession(t, map[string]string{"TALLY_FAKE_GROK_SCENARIO": "emptyok"}, nil)
	res, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "x"})
	if err != nil {
		t.Fatalf("契约不兑现是 RunFailed claim: %v", err)
	}
	if res.Status != agentruntime.RunFailed {
		t.Fatalf("Status: want failed, got %s summary=%q", res.Status, res.Summary)
	}
}

// TestFailureSummaryRedactsSecrets:失败诊断跨持久的 claim 边界;外部 CLI
// 可能吐出含机密的文本,进入 AgentRunResult/log/UI 前必须脱敏;Config.Env
// 注入的凭据值也绝不出现在任何 claim/错误串里。
func TestFailureSummaryRedactsSecrets(t *testing.T) {
	r0 := &run{id: "run-redact", status: agentruntime.RunRunning}
	s := &Session{session: "grok-redact"}
	secret := "sk-abcdefghijklmnopqrstuvwxyz1234567890"
	res := s.finishRun(r0, agentruntime.RunFailed, "provider failed api_key="+secret)
	if strings.Contains(res.Summary, secret) {
		t.Fatalf("failure summary leaked secret: %q", res.Summary)
	}
	if !strings.Contains(res.Summary, "[REDACTED]") {
		t.Fatalf("failure summary should contain redaction marker: %q", res.Summary)
	}
}

func TestConfiguredSecretNeverEntersClaimOrError(t *testing.T) {
	// exitfail 路径:注入凭据值后触发失败 —— summary 与 error 都不得携带该值。
	secret := "xai-secret-probe-value-7f3a9"
	_, s := startSession(t, map[string]string{
		"XAI_API_KEY":              secret,
		"TALLY_FAKE_GROK_SCENARIO": "exitfail",
	}, nil)
	res, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "x"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.Status != agentruntime.RunFailed {
		t.Fatalf("Status: want failed, got %s", res.Status)
	}
	if strings.Contains(res.Summary, secret) {
		t.Fatalf("失败 claim 泄漏了 Config.Env 凭据值: %q", res.Summary)
	}
}

// TestCancelKillsWholeProcessTree 是整树终止的 teardown 证明:Cancel/ctx 取消
// 必须整树终止 —— 替身与其子进程全部消失,无孤儿残留。
//
// 双重证据:(a) WaitDelay 被刻意抬高到 30s —— 若子进程在整树终止后存活,
// 它持有的 stdout 管道会把 Execute 拖到 WaitDelay 兜底,waitForStatus 的
// 20s 上限必然先超时;快速收敛即证明持管道的子进程确实死了。
// (b) tasklist 镜像名核对确认子进程 pid 不再属于任何 fakegrok.exe。
func TestCancelKillsWholeProcessTree(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	_, s := startSession(t, map[string]string{"TALLY_FAKE_GROK_RECORD": record}, func(c *Config) { c.WaitDelay = 30 * time.Second })
	ctx := context.Background()

	outcome := make(chan executeOutcome, 1)
	go func() {
		res, err := s.Execute(ctx, agentruntime.ExecuteRequest{Instruction: longMarker + " run"})
		outcome <- executeOutcome{res, err}
	}()

	// 等替身记下子进程 PID(spawn 完成后),再取消:确保整树里确实有
	// 子进程可泄漏。
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
	_, s := startSession(t, map[string]string{"TALLY_FAKE_GROK_RECORD": record}, nil)
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

// AC「no orphan process in cancellation/timeout tests」:ctx 超时(时限到期
// 触发 Cancel 路径)必须收敛 RunCancelled,且整树终止后子进程零残留
// (真实进程树取证)。
func TestContextTimeoutLeavesNoOrphan(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	_, s := startSession(t, map[string]string{"TALLY_FAKE_GROK_RECORD": record}, func(c *Config) { c.WaitDelay = 2 * time.Second })

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
	// 孤儿零残留:替身子进程在整树终止后必须消失。
	if rec.ChildPID > 0 {
		waitForGone(t, rec.ChildPID)
	}
}

func TestProcessCrashHandledAndSessionResponsive(t *testing.T) {
	// PRD §35 crash 场景:外部单进程 kill 替身(非 Cancel、非整树)必须
	// 归为 RunFailed,且 Execute 必须收敛,会话不变成僵尸。kill 前先等
	// 替身记下子进程 PID:保证孤儿真实存在并持有输出管道 —— 收敛只能
	// 经 WaitDelay 兜底,不得挂死、不得伪装成取消或成功。
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	_, s := startSession(t, map[string]string{"TALLY_FAKE_GROK_RECORD": record}, func(c *Config) { c.WaitDelay = 2 * time.Second })
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
	_, s := startSession(t, map[string]string{"TALLY_FAKE_GROK_RECORD": record}, nil)
	ctx := context.Background()

	outcome := make(chan executeOutcome, 1)
	go func() {
		res, err := s.Execute(ctx, agentruntime.ExecuteRequest{Instruction: longMarker + " run"})
		outcome <- executeOutcome{res, err}
	}()
	waitForChildPID(t, record)

	home := s.home
	if home == "" {
		t.Fatal("会话应有 per-session home")
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
	// per-session home 被删除。
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(home); os.IsNotExist(err) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("Close 后 home %s 未被删除", home)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// ---- PRD §35 通用 conformance(全场景,零声明 → cap:* 全部不运行)---------------

// TestGrokRuntimeConformance 对挂在替身 CLI 上的 GrokRuntime 跑完整
// runtimeconformance 套件:start / cwd / execute / result / events(未声明
// 拒绝方向)/ cancel(ctx 路径)/ crash / workspace / usage / capability_truth。
// 基础声明为空集 → 全部 cap:* 能力专项场景按门控不得运行(声明真实性)。
func TestGrokRuntimeConformance(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	rt := New(Config{
		BinPath:   fakeGrokBin,
		Env:       testEnv(record),
		WaitDelay: 2 * time.Second, // crash 注入面的孤儿管道兜底(测试用短值)
	})
	rep := runtimeconformance.Run(t, rt, runtimeconformance.Options{
		Probes: runtimeconformance.Probes{
			LongRunInstruction: longMarker + " run",
			CrashRun: func(ctx context.Context, sess agentruntime.Session, longRunInstruction string) (agentruntime.AgentRunResult, error) {
				gs, ok := sess.(*Session)
				if !ok {
					return agentruntime.AgentRunResult{}, fmt.Errorf("crash probe: want *grok.Session, got %T", sess)
				}
				outcome := make(chan executeOutcome, 1)
				go func() {
					res, err := sess.Execute(ctx, agentruntime.ExecuteRequest{Instruction: longRunInstruction})
					outcome <- executeOutcome{res, err}
				}()
				// 等替身记下子进程 PID:保证 kill 时孤儿真实存在并持有
				// 输出管道(WaitDelay 兜底路径被确定性覆盖)。
				waitForChildPID(t, record)
				// 未声明 event.streaming,订阅不到 run.started:经诊断缝定位。
				deadline := time.Now().Add(10 * time.Second)
				var id agentruntime.RunID
				for {
					got, ok := gs.InFlightRunID()
					if ok {
						id = got
						break
					}
					if time.Now().After(deadline) {
						return agentruntime.AgentRunResult{}, fmt.Errorf("crash probe: run 未进入 in-flight")
					}
					time.Sleep(10 * time.Millisecond)
				}
				proc, ok := gs.RunProcess(id)
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
				gs, ok := sess.(*Session)
				if !ok {
					return fmt.Errorf("workspace probe: want *grok.Session, got %T", sess)
				}
				rec := lastRecordFor(record)
				if rec.Cwd != gs.Workspace() {
					return fmt.Errorf("替身观测到的进程 cwd %q 不是提供的 workspace %q(workspaceRoot=process.cwd())", rec.Cwd, gs.Workspace())
				}
				return nil
			},
		},
	})
	if !rep.OK() {
		t.Fatalf("GrokRuntime conformance 未通过:\n%s", rep.FailureText())
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

// ---- 真实 grok 表面(二进制 + 凭据门控,环境前提)---------------------------------

// TestLiveGrokSurface 在 TALLY_GROK_LIVE=1 时对真实 grok 1.0.87 二进制验证:
//  1. 钉版身份闸:Snapshot 现场观察 `--version` 并与 PinnedGrokVersion 比对
//     (版本不符即失败 —— 上游快速演进,版本漂移必须显形);不需要凭据;
//  2. keyless fail-safe:强制清空 XAI_API_KEY 的回合必须以 RunFailed claim
//     收场(不伪装成功、不挂死、不是 Execute error;观测的 supported 无凭据
//     形态本身也是 live 观测的一部分);不需要凭据;
//  3. 若进程环境持有 XAI_API_KEY:一次真实 headless 回合(指令为无害回显,
//     至多一个模型 turn)应 RunCompleted 且 stdout 非空。
//
// 未请求 live 时 SKIP;TALLY_GROK_LIVE=1 但二进制缺失是 gate 配置失败而不是
// SKIP。与 runtimes/dsh 的 live gate 有一处刻意差异:凭据缺失不 Fatal ——
// 本机无凭据时按交付纪律记 NOT_RUN(真实完成回合),keyless 失败路径仍完整
// 观测;持凭据的部署方设置 XAI_API_KEY 后即自动覆盖第 3 步。
func TestLiveGrokSurface(t *testing.T) {
	if os.Getenv("TALLY_GROK_LIVE") == "" {
		t.Skip("TALLY_GROK_LIVE 未设置:跳过真实 grok 表面(观测事实见包注释)")
	}
	bin := strings.TrimSpace(os.Getenv("TALLY_GROK_BIN"))
	if bin == "" {
		t.Fatal("TALLY_GROK_LIVE 已启用但缺 TALLY_GROK_BIN:live gate 必须显式指定二进制(钉死版本面,不做隐式发现)")
	}
	if err := requireFile(bin, "env TALLY_GROK_BIN"); err != nil {
		t.Fatalf("TALLY_GROK_LIVE 已启用但二进制不可用: %v", err)
	}

	// 1) 钉版身份闸(零模型成本)。
	rt := New(Config{BinPath: bin})
	snap, err := rt.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot(真实二进制): %v", err)
	}
	t.Logf("真实 grok 身份: version=%s bin=%s", snap.CLIVersion, snap.BinaryPath)

	// 2) keyless fail-safe(强制清空注入键;不消耗模型 turn 的失败路径)。
	keyless := New(Config{
		BinPath: bin,
		Env:     map[string]string{"XAI_API_KEY": ""},
	})
	s, err := keyless.Start(context.Background(), agentruntime.StartRequest{Workspace: t.TempDir()})
	if err != nil {
		t.Fatalf("Start(keyless): %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	res, err := s.Execute(ctx, agentruntime.ExecuteRequest{Instruction: "conformance: 请原样回复 ok"})
	if err != nil {
		t.Fatalf("keyless 回合不应是 Execute error(失败走 claim): %v", err)
	}
	_ = s.Close(ctx)
	if res.Status != agentruntime.RunFailed {
		t.Fatalf("keyless 回合应 RunFailed, got %s summary=%q", res.Status, truncForLog(res.Summary))
	}
	t.Logf("keyless fail-safe: status=%s summary=%q", res.Status, truncForLog(res.Summary))

	// 3) 一次真实回合(至多一个模型 turn)。本机无凭据时如实记 NOT_RUN。
	if strings.TrimSpace(os.Getenv("XAI_API_KEY")) == "" {
		t.Log("REAL_EXECUTION=NOT_RUN(XAI_API_KEY 未设置:本机无可用 Grok 凭据;真实完成回合待凭据部署)")
		return
	}
	live := New(Config{BinPath: bin})
	s2, err := live.Start(context.Background(), agentruntime.StartRequest{Workspace: t.TempDir()})
	if err != nil {
		t.Fatalf("Start(live): %v", err)
	}
	defer func() { _ = s2.Close(context.Background()) }()
	ctx2, cancel2 := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel2()
	res2, err := s2.Execute(ctx2, agentruntime.ExecuteRequest{Instruction: "conformance: 请原样回复 ok"})
	if err != nil {
		t.Fatalf("真实回合 Execute: %v", err)
	}
	if res2.Status != agentruntime.RunCompleted || strings.TrimSpace(res2.Summary) == "" {
		t.Fatalf("真实回合应完成且 stdout 非空: %s summary=%q", res2.Status, truncForLog(res2.Summary))
	}
	t.Logf("真实回合完成: status=%s summary=%q session=%s", res2.Status, truncForLog(res2.Summary), res2.SessionID)
}

// truncForLog 供日志截断。
func truncForLog(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return s
}
