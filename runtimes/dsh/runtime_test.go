package dsh

// P4-04 的测试策略(对齐 runtimes/codex 的替身 CLI 模式):
//
//   - 全部进程面经「替身 dsh launcher」驱动:TestMain 从
//     testdata/fakedsh/main.go 现场构建替身二进制,适配器对它 spawn/读取/
//     取消/整树终止 —— 进程边界、headless 契约(stdout=最终回答、退出码、
//     MISSING_CREDENTIAL)、argv 组装、隔离与遥测环境强制、整树 teardown
//     都是真实经过(不是内存 mock)。
//   - 通用 conformance(PRD §35)对同一替身运行 runtimeconformance.Run 全套:
//     基础声明为空集 → 全部 cap:* 场景按门控不运行,events/capability_truth
//     走「未声明即拒绝」方向。
//   - 真实 dsh 表面由 TALLY_DSH_LIVE=1 门控(凭据/launcher 部署前提);
//     P4-03 spike 已对钉住版本完成 live 协议观测。

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

// fakeDSHBin 是 TestMain 构建的替身 dsh launcher 路径。
var fakeDSHBin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "tally-fake-dsh-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "dsh test: 建临时目录失败: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(dir)
	src, err := filepath.Abs(filepath.Join("testdata", "fakedsh", "main.go"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "dsh test: 解析替身源码路径失败: %v\n", err)
		os.Exit(1)
	}
	exe := filepath.Join(dir, "fakedsh.exe")
	if out, berr := exec.Command("go", "build", "-o", exe, src).CombinedOutput(); berr != nil {
		fmt.Fprintf(os.Stderr, "dsh test: 构建替身 dsh launcher 失败: %v\n%s\n", berr, out)
		os.Exit(1)
	}
	fakeDSHBin = exe
	os.Exit(m.Run())
}

// longMarker 是替身 launcher 识别「长运行指令」的标记:conformance 的
// LongRunInstruction 注入点(launcher spawn 子进程并等待,阻塞直到被
// Cancel/ctx/外部 kill 整树终止)。
const longMarker = "tally-conformance-long"

// ---- 测试基础设施 ------------------------------------------------------------

// fakeRecord 是替身 launcher 记录行的 Go 侧投影。
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

// waitForChildPID 轮询记录文件直到替身 launcher 记下子进程 PID(长任务已
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

// processAlive 报告 pid 是否仍是我们的替身 launcher 进程。Windows 的 pid 在
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
	return strings.EqualFold(strings.Trim(fields[0], "\" "), "fakedsh.exe")
}

// waitForGone 轮询直到进程消失(整树终止收敛)或超时。
func waitForGone(t *testing.T, pid int) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for processAlive(pid) {
		if time.Now().After(deadline) {
			t.Fatalf("子进程 %d 在整树终止后仍未消失(存在孤儿残留,R-1 回归)", pid)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// testEnv 返回带替身凭据门与记录面的基础注入环境。
func testEnv(record string) map[string]string {
	env := map[string]string{"DEEPSEEK_API_KEY": "test-key"}
	if record != "" {
		env["TALLY_FAKE_DSH_RECORD"] = record
	}
	return env
}

// startSession 构造挂在替身 launcher 上的会话(缺省配置 + 注入环境,
// workspace 为新建临时目录)。
func startSession(t *testing.T, extraEnv map[string]string, cfgMod func(*Config)) (*Runtime, *Session) {
	t.Helper()
	return startSessionIn(t, t.TempDir(), extraEnv, cfgMod)
}

// startSessionIn 是 startSession 的显式 workspace 变体(cwd 效果级验证用)。
func startSessionIn(t *testing.T, ws string, extraEnv map[string]string, cfgMod func(*Config)) (*Runtime, *Session) {
	t.Helper()
	cfg := Config{BinPath: fakeDSHBin, Env: testEnv("")}
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
		t.Fatalf("Start: want *dsh.Session, got %T", s)
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
	// headless 表面(spike 条件 5 的诚实清单):基础声明为空集 —— 协议词汇表
	// 中没有任何能力是 headless 一次性进程能真实兑现的。
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
			t.Fatalf("capability %s must stay undeclared by the dsh headless adapter", c)
		}
	}
	if len(d.Capabilities.List()) != 0 {
		t.Fatalf("基础声明应为空集, got %v", d.Capabilities.Sorted())
	}

	// session.isolation 证据门:未持 canary 证据(零值)不得声明;置位后且
	// 仅置位后出现(与 runtimes/codex 同一声明纪律)。
	if got := New(Config{}).capabilities(); got.Has(agentruntime.CapSessionIsolation) {
		t.Fatal("零值 Config 不得声明 session.isolation(声明以 canary 证据为门)")
	}
	gated := New(Config{SessionIsolationVerified: true}).capabilities()
	if !gated.Has(agentruntime.CapSessionIsolation) || len(gated.List()) != 1 {
		t.Fatalf("SessionIsolationVerified 置位后应且仅应声明 session.isolation, got %v", gated.Sorted())
	}
}

// 未声明的能力需求必须在 CheckStart 协商处被拒绝(spike 条件 5:
// requested ≠ observed;无 resume/model.select 面)。
func TestStartRejectsUndeclaredCapabilityRequirements(t *testing.T) {
	rt := New(Config{BinPath: fakeDSHBin})
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
		t.Fatal("带 ModelProfile 的 Start 应被拒绝(未声明 model.select:headless 无模型旗标且实际模型不可观察)")
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
	rt := New(Config{BinPath: fakeDSHBin})
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

// ---- AC:钉版身份闸(spike 条件 1)---------------------------------------------

func TestSnapshotObservesPinnedIdentity(t *testing.T) {
	rt := New(Config{BinPath: fakeDSHBin})
	snap, err := rt.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if snap.Name != DescriptorName || snap.AdapterVersion != DescriptorVersion {
		t.Fatalf("snapshot identity: %+v", snap)
	}
	// 版本来自现场 `--version`(替身输出钉住版本),不是硬编码常量冒充观察。
	if snap.CLIVersion != PinnedDSHVersion {
		t.Fatalf("CLIVersion: want %q, got %q", PinnedDSHVersion, snap.CLIVersion)
	}
	if snap.BinaryPath != fakeDSHBin {
		t.Fatalf("BinaryPath: want %q, got %q", fakeDSHBin, snap.BinaryPath)
	}
	if len(snap.Capabilities) != 0 {
		t.Fatalf("snapshot capabilities: want 空, got %v", snap.Capabilities)
	}

	// IdentityReporter:版本是已建立的观察;Name/Model/SessionID 未观察为空,
	// 且期望配置(Config.Model)绝不回填 Model。
	_, s := startSession(t, nil, func(c *Config) { c.Model = "deepseek-reasoner" })
	id := s.Identity()
	if id.RuntimeVersion != PinnedDSHVersion {
		t.Fatalf("Identity.RuntimeVersion: want %q, got %q", PinnedDSHVersion, id.RuntimeVersion)
	}
	if id.RuntimeName != "" || id.Model != "" || id.SessionID != "" {
		t.Fatalf("未观察字段必须为空(不得回填期望配置), got %+v", id)
	}
	if !id.Observed() {
		t.Fatal("版本已观察,Observed() 应为 true")
	}
}

func TestStartFailsOnVersionMismatch(t *testing.T) {
	// 替身自报错误版本 → 钉版闸 fail-closed(开发者预览表面的契约没有版本
	// 锚即不成立,spike 条件 1/R-2)。
	rt := New(Config{
		BinPath: fakeDSHBin,
		Env:     map[string]string{"TALLY_FAKE_DSH_VERSION": "0.2.0-beta.1"},
	})
	_, err := rt.Start(context.Background(), agentruntime.StartRequest{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("观察版本与钉住版本不一致时 Start 应拒绝")
	}
	if !strings.Contains(err.Error(), "0.2.0-beta.1") || !strings.Contains(err.Error(), PinnedDSHVersion) {
		t.Fatalf("错误应同时携带观察版本与钉住版本: %v", err)
	}
	if _, err := rt.Snapshot(context.Background()); err == nil {
		t.Fatal("Snapshot 同样受钉版闸约束")
	}
}

func TestStartFailsWhenVersionUnresolvable(t *testing.T) {
	rt := New(Config{
		BinPath: fakeDSHBin,
		Env:     map[string]string{"TALLY_FAKE_DSH_SCENARIO": "versionfail"},
	})
	if _, err := rt.Start(context.Background(), agentruntime.StartRequest{Workspace: t.TempDir()}); err == nil {
		t.Fatal("--version 不可观察时 Start 应拒绝(fail-closed,与 codex 的尽力而为探针是有意偏离)")
	}
}

func TestParseCLIIdentityAndVersionPin(t *testing.T) {
	// spike §3 探针 1 的观测形态:裸版本行。
	if got := parseCLIIdentity("0.1.5-rc.1\n"); got.Version != "0.1.5-rc.1" {
		t.Fatalf("parseCLIIdentity: want 0.1.5-rc.1, got %+v", got)
	}
	if got := parseCLIIdentity("\n  \n"); got.observed() {
		t.Fatalf("空输出应为零值观察, got %+v", got)
	}
	if err := checkVersionPin("", cliIdentity{Version: PinnedDSHVersion}); err != nil {
		t.Fatalf("钉住版本一致应通过: %v", err)
	}
	err := checkVersionPin("", cliIdentity{Version: "0.1.5-alpha.2"})
	if err == nil || !strings.Contains(err.Error(), PinnedDSHVersion) {
		t.Fatalf("不一致应拒绝且指明钉住版本: %v", err)
	}
}

// ---- 二进制发现 ----------------------------------------------------------------

func TestResolveBinaryPathPrecedence(t *testing.T) {
	t.Setenv(envDSHBinPath, fakeDSHBin)
	got, err := ResolveBinaryPath()
	if err != nil {
		t.Fatalf("ResolveBinaryPath: %v", err)
	}
	if got != fakeDSHBin {
		t.Fatalf("env DSH_BIN_PATH 应优先且直接采用: want %q, got %q", fakeDSHBin, got)
	}
	// env 指向缺失路径 → 显式报错(不静默回落)。
	t.Setenv(envDSHBinPath, filepath.Join(t.TempDir(), "no-such-dsh.exe"))
	if _, err := ResolveBinaryPath(); err == nil {
		t.Fatal("env 指向缺失路径应显式报错")
	}
}

func TestStartFailsExplicitlyWithoutBinary(t *testing.T) {
	t.Setenv(envDSHBinPath, "")
	if _, err := exec.LookPath("dsh"); err == nil {
		t.Skip("本机 PATH 上存在 dsh,无法构造「完全不可发现」场景")
	}
	rt := New(Config{})
	_, err := rt.Start(context.Background(), agentruntime.StartRequest{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("launcher 不可发现时 Start 应报错")
	}
}

// ---- AC:Start/execute/cwd/patch(经替身 launcher 的真实进程面)------------------

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
	// SessionID 是适配器自造的会话编号(headless 无协议句柄),非空以定位;
	// RunID 非空。
	if res.RunID == "" || res.SessionID == "" {
		t.Fatalf("claim 缺少 RunID/SessionID: %+v", res)
	}
	// Usage 不解析:headless 不自报用量,解析即发明(R-3;宁缺毋假)。
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
	// workspace 经 spawn cwd 钉定(headless 无 cwd 旗标,spike §2.2):
	// 替身在它的 cwd 里真实读到 marker 文件 = cwd 绑定的效果级证明
	//(spike §3 探针 6 的形态)。
	ws := t.TempDir()
	marker := filepath.Join(ws, "marker.txt")
	if err := os.WriteFile(marker, []byte("cwd-bound-answer"), 0o644); err != nil {
		t.Fatal(err)
	}
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	_, s := startSessionIn(t, ws, map[string]string{"TALLY_FAKE_DSH_RECORD": record}, nil)

	res, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "read marker.txt"})
	if err != nil || res.Status != agentruntime.RunCompleted {
		t.Fatalf("Execute: res=%+v err=%v", res, err)
	}
	if res.Summary != "cwd-bound-answer" {
		t.Fatalf("最终回答应来自替身在 cwd 内真实读到的 marker 文件: got %q", res.Summary)
	}
	rec := lastRecord(t, record)
	if rec.Cwd != s.Workspace() {
		t.Fatalf("替身观测到的进程 cwd %q 应是提供的 workspace %q(spike §2.2:workspaceRoot=process.cwd())", rec.Cwd, s.Workspace())
	}
	if rec.Task != "read marker.txt" {
		t.Fatalf("instruction 应原样作为位置参数传入: got %q", rec.Task)
	}
	// argv 形态:[bin, --profile, headless, <instruction>](无模型配置 = 零
	// overlay 零旗标)。
	wantArgv := append([]string{fakeDSHBin, "--profile", "headless"}, "read marker.txt")
	if !reflect.DeepEqual(rec.Argv, wantArgv) {
		t.Fatalf("argv 应为 %v, got %v", wantArgv, rec.Argv)
	}
	if rec.PatchPath != "" || rec.PatchContent != "" {
		t.Fatalf("未配置期望模型时不应有 --patch: %+v", rec)
	}
}

func TestModelProviderGeneratesPatchOverlay(t *testing.T) {
	// 期望模型/提供者只经生成的最小 --patch YAML overlay(spike 条件 6;
	// 无 CLI 模型旗标)。
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	_, s := startSession(t, map[string]string{"TALLY_FAKE_DSH_RECORD": record}, func(c *Config) {
		c.Model = "deepseek-reasoner"
		c.Provider = "deepseek-official"
	})
	res, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "add a test"})
	if err != nil || res.Status != agentruntime.RunCompleted {
		t.Fatalf("Execute: res=%+v err=%v", res, err)
	}
	rec := lastRecord(t, record)
	wantPatch := "- id: agent-default-model\n  config:\n    provider: deepseek-official\n    model: deepseek-reasoner\n"
	if rec.PatchContent != wantPatch {
		t.Fatalf("生成的 overlay 与期望不符:\n got %q\nwant %q", rec.PatchContent, wantPatch)
	}
	if rec.PatchPath == "" {
		t.Fatal("argv 应携带 --patch <file>")
	}
	// overlay 旗标必须在任务文本之前(launcher:首个未识别 token 起为 app args)。
	argv := rec.Argv
	idx := -1
	for i, a := range argv {
		if a == patchFlag {
			idx = i
			break
		}
	}
	if idx < 0 || idx+2 >= len(argv) || argv[len(argv)-1] != "add a test" {
		t.Fatalf("--patch 应在任务文本之前且 instruction 最后: %v", argv)
	}
}

func TestRenderPatchOverlay(t *testing.T) {
	// 零值 = 零 overlay。
	got, err := renderPatchOverlay("", "")
	if err != nil || got != "" {
		t.Fatalf("空配置应零 overlay: %q err=%v", got, err)
	}
	got, err = renderPatchOverlay("", "deepseek-chat")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got != "- id: agent-default-model\n  config:\n    model: deepseek-chat\n" {
		t.Fatalf("仅 model:\n got %q", got)
	}
	// 非法 token 显式拒绝(不做转义器)。
	for _, bad := range []string{"model with space", "a;b", "x\ny", "'quote'"} {
		if _, err := renderPatchOverlay(bad, ""); err == nil {
			t.Fatalf("非法 Provider %q 应被拒绝", bad)
		}
		if _, err := renderPatchOverlay("", bad); err == nil {
			t.Fatalf("非法 Model %q 应被拒绝", bad)
		}
	}
	// Start 前置校验同样拒绝(期望配置管道在构造期显式失败)。
	rt := New(Config{BinPath: fakeDSHBin, Model: "bad model"})
	if _, err := rt.Start(context.Background(), agentruntime.StartRequest{Workspace: t.TempDir()}); err == nil {
		t.Fatal("非法 Model 应让 Start 显式失败")
	}
}

// ---- AC:隔离与出口硬化(spike 条件 2/3)-----------------------------------------

func TestPerSessionHomeIsolationAndForcedEnv(t *testing.T) {
	// 模拟「继承环境试图恢复默认 home、开遥测、升级权限」的部署面:
	// 适配器必须强制 per-session home / DISABLED / 保守权限,压倒继承环境与
	// Config.Env 中的同名值。
	ownerHome := filepath.Join(t.TempDir(), "owner-dsh")
	if err := os.MkdirAll(ownerHome, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DSH_HOME", ownerHome)
	t.Setenv("DSH_TELEMETRY_MODE", "FEEDBACK_ONLY")
	t.Setenv("DSH_PERMISSION_MODE", "danger-full-access")

	record := filepath.Join(t.TempDir(), "calls.jsonl")
	rt, s1 := startSession(t, map[string]string{
		"TALLY_FAKE_DSH_RECORD": record,
		"DSH_TELEMETRY_MODE":    "FEEDBACK_ONLY", // Config.Env 中的同名值同样被压倒
	}, nil)
	if _, err := s1.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "x"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	rec := lastRecord(t, record)

	// per-session home:全新临时目录,绝非 owner home(spike 条件 2:默认
	// ~/.dsh 跨 run 持久化 sessions/storages,是 protocol-observable 泄漏面)。
	if rec.Home == "" || rec.Home == ownerHome {
		t.Fatalf("DSH_HOME 应为全新 per-session 目录, got %q(owner=%q)", rec.Home, ownerHome)
	}
	if _, err := os.Stat(rec.Home); err != nil {
		t.Fatalf("per-session home 应真实存在: %v", err)
	}
	// 遥测强制 DISABLED(spike 条件 3:默认 FEEDBACK_ONLY 送 DeepSeek OTLP)。
	if rec.Telemetry != telemetryDisabled {
		t.Fatalf("DSH_TELEMETRY_MODE 应被强制为 %s, got %q", telemetryDisabled, rec.Telemetry)
	}
	// 权限保守默认,压倒继承的 danger-full-access(升级只经显式 Config)。
	if rec.Permission != DefaultPermissionMode {
		t.Fatalf("DSH_PERMISSION_MODE 应被强制为 %s, got %q", DefaultPermissionMode, rec.Permission)
	}

	// 第二个会话获得不同的全新 home(会话边界,不共享)。
	record2 := filepath.Join(t.TempDir(), "calls2.jsonl")
	_, s2 := startSession(t, map[string]string{"TALLY_FAKE_DSH_RECORD": record2}, nil)
	if _, err := s2.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "x"}); err != nil {
		t.Fatalf("Execute(s2): %v", err)
	}
	rec2 := lastRecord(t, record2)
	if rec2.Home == "" || rec2.Home == rec.Home {
		t.Fatalf("两个会话应获得不同的全新 home: %q vs %q", rec.Home, rec2.Home)
	}

	// Close 删除本会话 home(尽力而为;等待短暂收敛后断言)。
	if err := s1.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(rec.Home); os.IsNotExist(err) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("Close 后 per-session home %s 未被删除", rec.Home)
		}
		time.Sleep(50 * time.Millisecond)
	}
	_ = rt
}

func TestPermissionModeConfiguredAndValidated(t *testing.T) {
	// 显式升级经 Config 传递(danger-full-access 是 spike §2.2 观测枚举)。
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	_, s := startSession(t, map[string]string{"TALLY_FAKE_DSH_RECORD": record}, func(c *Config) {
		c.PermissionMode = "danger-full-access"
	})
	if _, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "x"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if rec := lastRecord(t, record); rec.Permission != "danger-full-access" {
		t.Fatalf("显式配置的权限应生效: got %q", rec.Permission)
	}

	// 未观测取值在 Start 处显式拒绝(不发明枚举)。
	rt := New(Config{BinPath: fakeDSHBin, PermissionMode: "read-only"})
	if _, err := rt.Start(context.Background(), agentruntime.StartRequest{Workspace: t.TempDir()}); err == nil {
		t.Fatal("非法 PermissionMode 应让 Start 显式失败")
	}
}

// ---- 失败/取消/崩溃/关闭(claim 纪律 + R-1 teardown)-----------------------------

func TestMissingCredentialIsFailedClaimNotError(t *testing.T) {
	// spike §2.6 实测契约:缺 DEEPSEEK_API_KEY → exit 1 + stderr
	// MISSING_CREDENTIAL。适配器必须把它如实归类为 RunFailed claim(失败
	// 证据携带 stderr 原文),既不是成功也不是 Execute error。
	_, s := startSession(t, map[string]string{"DEEPSEEK_API_KEY": ""}, nil)
	res, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "reply ok"})
	if err != nil {
		t.Fatalf("凭据缺失是 run 级失败,不是 Execute error: %v", err)
	}
	if res.Status != agentruntime.RunFailed {
		t.Fatalf("Status: want failed, got %s summary=%q", res.Status, res.Summary)
	}
	if !strings.Contains(res.Summary, "MISSING_CREDENTIAL") {
		t.Fatalf("summary 应携带 stderr 结构化错误原文: %q", res.Summary)
	}
}

func TestNonZeroExitSurfacesAsFailedClaim(t *testing.T) {
	_, s := startSession(t, map[string]string{"TALLY_FAKE_DSH_SCENARIO": "exitfail"}, nil)
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
	_, s := startSession(t, map[string]string{"TALLY_FAKE_DSH_SCENARIO": "emptyok"}, nil)
	res, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "x"})
	if err != nil {
		t.Fatalf("契约不兑现是 RunFailed claim: %v", err)
	}
	if res.Status != agentruntime.RunFailed {
		t.Fatalf("Status: want failed, got %s summary=%q", res.Status, res.Summary)
	}
}

// TestCancelKillsWholeProcessTree 是 R-1 的 teardown 证明:Cancel/ctx 取消
// 必须整树终止 —— launcher 与其子进程全部消失,无孤儿残留。
//
// 双重证据:(a) WaitDelay 被刻意抬高到 30s —— 若子进程在整树终止后存活,
// 它持有的 stdout 管道会把 Execute 拖到 WaitDelay 兜底,waitForStatus 的
// 20s 上限必然先超时;快速收敛即证明持管道的子进程确实死了。
// (b) tasklist 镜像名核对确认子进程 pid 不再属于任何 fakedsh.exe。
func TestCancelKillsWholeProcessTree(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	_, s := startSession(t, map[string]string{"TALLY_FAKE_DSH_RECORD": record}, func(c *Config) { c.WaitDelay = 30 * time.Second })
	ctx := context.Background()

	outcome := make(chan executeOutcome, 1)
	go func() {
		res, err := s.Execute(ctx, agentruntime.ExecuteRequest{Instruction: longMarker + " run"})
		outcome <- executeOutcome{res, err}
	}()

	// 等 launcher 记下子进程 PID(spawn 完成后),再取消:确保整树里确实有
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

	// R-1 核心:子进程随 launcher 一并消失(整树终止),无孤儿。
	waitForGone(t, rec.ChildPID)
}

func TestExecuteContextCancelIsRunCancelled(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	_, s := startSession(t, map[string]string{"TALLY_FAKE_DSH_RECORD": record}, nil)
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
	// 清理可能的 launcher 孤儿子进程(ctx 取消同样走整树终止;此处兜底)。
	if rec := lastRecordFor(record); rec.ChildPID > 0 {
		waitForGone(t, rec.ChildPID)
	}
}

func TestProcessCrashHandledAndSessionResponsive(t *testing.T) {
	// PRD §35 crash 场景:外部单进程 kill launcher(非 Cancel、非整树)必须
	// 归为 RunFailed,且 Execute 必须收敛,会话不变成僵尸。kill 前先等
	// launcher 记下子进程 PID:保证孤儿真实存在并持有输出管道 —— 收敛只能
	// 经 WaitDelay 兜底(R-1 相邻硬化),不得挂死、不得伪装成取消或成功。
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	_, s := startSession(t, map[string]string{"TALLY_FAKE_DSH_RECORD": record}, func(c *Config) { c.WaitDelay = 2 * time.Second })
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
		t.Fatal("in-flight run 应可经 RunProcess 观测到底层 launcher 进程")
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
	_, s := startSession(t, map[string]string{"TALLY_FAKE_DSH_RECORD": record}, nil)
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

// TestDSHRuntimeConformance 对挂在替身 launcher 上的 DSHRuntime 跑完整
// runtimeconformance 套件:start / cwd / execute / result / events(未声明
// 拒绝方向)/ cancel(ctx 路径)/ crash / workspace / usage / capability_truth。
// 基础声明为空集 → 全部 cap:* 能力专项场景按门控不得运行(声明真实性)。
func TestDSHRuntimeConformance(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	rt := New(Config{
		BinPath:   fakeDSHBin,
		Env:       testEnv(record),
		WaitDelay: 2 * time.Second, // crash 注入面的孤儿管道兜底(测试用短值)
	})
	rep := runtimeconformance.Run(t, rt, runtimeconformance.Options{
		Probes: runtimeconformance.Probes{
			LongRunInstruction: longMarker + " run",
			CrashRun: func(ctx context.Context, sess agentruntime.Session, longRunInstruction string) (agentruntime.AgentRunResult, error) {
				ds, ok := sess.(*Session)
				if !ok {
					return agentruntime.AgentRunResult{}, fmt.Errorf("crash probe: want *dsh.Session, got %T", sess)
				}
				outcome := make(chan executeOutcome, 1)
				go func() {
					res, err := sess.Execute(ctx, agentruntime.ExecuteRequest{Instruction: longRunInstruction})
					outcome <- executeOutcome{res, err}
				}()
				// 等 launcher 记下子进程 PID:保证 kill 时孤儿真实存在并持有
				// 输出管道(WaitDelay 兜底路径被确定性覆盖)。
				waitForChildPID(t, record)
				// 未声明 event.streaming,订阅不到 run.started:经诊断缝定位。
				deadline := time.Now().Add(10 * time.Second)
				var id agentruntime.RunID
				for {
					got, ok := ds.InFlightRunID()
					if ok {
						id = got
						break
					}
					if time.Now().After(deadline) {
						return agentruntime.AgentRunResult{}, fmt.Errorf("crash probe: run 未进入 in-flight")
					}
					time.Sleep(10 * time.Millisecond)
				}
				proc, ok := ds.RunProcess(id)
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
				ds, ok := sess.(*Session)
				if !ok {
					return fmt.Errorf("workspace probe: want *dsh.Session, got %T", sess)
				}
				rec := lastRecordFor(record)
				if rec.Cwd != ds.Workspace() {
					return fmt.Errorf("替身观测到的进程 cwd %q 不是提供的 workspace %q(spike §2.2)", rec.Cwd, ds.Workspace())
				}
				return nil
			},
		},
	})
	if !rep.OK() {
		t.Fatalf("DSHRuntime conformance 未通过:\n%s", rep.FailureText())
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

// ---- 真实 dsh 表面(launcher 部署 + 凭据门控,环境前提)--------------------------

// TestLiveDSHSurface 在 TALLY_DSH_LIVE=1 时对真实 dsh launcher 验证三件事:
//  1. 钉版身份闸:Snapshot 现场观察 `--version` 并与 PinnedDSHVersion 比对
//     (版本不符即失败 —— 上游开发者预览,版本漂移必须显形);
//  2. keyless fail-safe:强制清空 DEEPSEEK_API_KEY 的回合必须以 RunFailed
//     claim 收场(不伪装成功、不挂死、不是 Execute error);
//  3. 若进程环境持有 DEEPSEEK_API_KEY:一次真实 headless 回合(指令为无害
//     回显,至多一个模型 turn)应 RunCompleted 且 stdout 非空。
//
// 未请求 live 时 SKIP;一旦 TALLY_DSH_LIVE=1 显式请求,launcher/key 缺失
// 都是 gate 配置失败而不是 SKIP。launcher 使用与生产 worker 相同的
// TALLY_DSH_BIN 配置面,避免测试通过另一套发现规则。
func TestLiveDSHSurface(t *testing.T) {
	if os.Getenv("TALLY_DSH_LIVE") == "" {
		t.Skip("TALLY_DSH_LIVE 未设置:跳过真实 dsh 表面(P4-03 spike 已有钉住版本 live 证据)")
	}
	bin := strings.TrimSpace(os.Getenv("TALLY_DSH_BIN"))
	if bin == "" {
		t.Fatal("TALLY_DSH_LIVE 已启用但缺 TALLY_DSH_BIN:live gate 必须使用生产 worker 同一 launcher 配置面")
	}
	if err := requireFile(bin, "env TALLY_DSH_BIN"); err != nil {
		t.Fatalf("TALLY_DSH_LIVE 已启用但 launcher 不可用: %v", err)
	}
	if strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY")) == "" {
		t.Fatal("TALLY_DSH_LIVE 已启用但缺 DEEPSEEK_API_KEY:显式 live gate 不允许 SKIP 真实完成回合")
	}

	// 1) 钉版身份闸(零模型成本)。
	rt := New(Config{BinPath: bin})
	snap, err := rt.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot(真实 launcher): %v", err)
	}
	t.Logf("真实 dsh 身份: version=%s bin=%s", snap.CLIVersion, snap.BinaryPath)

	// 2) keyless fail-safe(强制清空注入键;不消耗模型 turn 的失败路径)。
	keyless := New(Config{
		BinPath: bin,
		Env:     map[string]string{"DEEPSEEK_API_KEY": ""},
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
	s.Close(ctx)
	if res.Status != agentruntime.RunFailed {
		t.Fatalf("keyless 回合应 RunFailed, got %s summary=%q", res.Status, res.Summary)
	}
	t.Logf("keyless fail-safe: summary=%q", truncForLog(res.Summary))

	// 3) 一次真实回合(至多一个模型 turn)。到达这里时 live gate 已在
	// 入口强制验证 DEEPSEEK_API_KEY 存在。
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
	t.Logf("真实回合完成: summary=%q session=%s", truncForLog(res2.Summary), res2.SessionID)
}

// truncForLog 供日志截断。
func truncForLog(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return s
}

// ---- P4-05 生命周期/一致性矩阵(包级延伸) -----------------------------------

// P4-05 AC「no orphan process in cancellation/timeout tests」:ctx 超时(时限
// 到期触发 Cancel 路径)必须收敛 RunCancelled,且整树终止后子进程零残留
// (R-1 硬化的真实进程树证明)。收敛与孤儿清理都在同一测试内断言。
func TestP405ContextTimeoutLeavesNoOrphan(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	_, s := startSession(t, map[string]string{"TALLY_FAKE_DSH_RECORD": record}, func(c *Config) { c.WaitDelay = 2 * time.Second })

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
	// 孤儿零残留:launcher 子进程在整树终止后必须消失(真实进程树取证)。
	if rec.ChildPID > 0 {
		waitForGone(t, rec.ChildPID)
	}
}

// P4-05 AC「no owner-repo mutation outside assigned workspace」:workspace 之外
// 的目录(owner 主工作树的代理)在执行/取消/关闭全生命周期零变更 —— 真实
// 进程 cwd 钉定在 workspace,替身只在其 cwd 内活动;DSH_HOME 在 Close 后删除
// (per-session 隔离,无跨运行残留)。
func TestP405NoOwnerRepoMutation(t *testing.T) {
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
	_, s := startSessionIn(t, ws, map[string]string{"TALLY_FAKE_DSH_RECORD": record}, nil)

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
	// per-session DSH_HOME 已随 Close 删除(隔离无跨运行残留)。
	if _, err := os.Stat(s.home); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Close 后 per-session DSH_HOME 应已删除: %v", err)
	}
}

// Failure diagnostics cross a durable claim boundary. Secrets emitted by an
// external launcher must be redacted before they can enter AgentRunResult/log/UI.
func TestFailureSummaryRedactsSecrets(t *testing.T) {
	r0 := &run{id: "run-redact", status: agentruntime.RunRunning}
	s := &Session{session: "dsh-redact"}
	secret := "sk-abcdefghijklmnopqrstuvwxyz1234567890"
	res := s.finishRun(r0, agentruntime.RunFailed, "provider failed api_key="+secret)
	if strings.Contains(res.Summary, secret) {
		t.Fatalf("failure summary leaked secret: %q", res.Summary)
	}
	if !strings.Contains(res.Summary, "[REDACTED]") {
		t.Fatalf("failure summary should contain redaction marker: %q", res.Summary)
	}
}
