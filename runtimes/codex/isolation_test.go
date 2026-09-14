// P0-03 per-run session/home 隔离的验证(A0-02 决策 / DECISION_LOCKS #22)。
//
// 替身 CLI(canaryCLISource)按 L3 Reference Pack 锚定的真实状态布局建模:
//   - REF-CODEX-AUTH:file auth 在 CODEX_HOME/auth.json,无 auth.json 即按
//     探针记录 §3 的 401 形态收场(error 事件 + error item,exit 0);
//   - REF-CODEX-ROLLOUT:session/rollout 数据根在 CODEX_HOME/sessions;
//   - REF-CODEX-INSTR:CODEX_HOME 是可观测指令状态,不是纯缓存。
//
// 与 runtime_test.go 的 conformance 替身(恒答 "ok")不同,本替身把「本
// run 所见的一切 CODEX_HOME 先前状态」(home 路径 + 先前 rollout 内容)作为
// 最终消息返回 —— 模拟真实 Codex 下 CODEX_HOME 状态对 run 的 protocol-
// observable 检索面(探针记录 §6)。这样 Session Isolation Canary 的裁决就
// 不再是空洞通过:若跨 run 状态可观测,canary 令牌必然出现在检索会话的
// claim/事件里;隔离成立时则不可检索。TALLY_SPIKE_LEAK_HOME 是仅测试注入的
// 共享状态通道(生产 runtime 永不设置),用于验证 canary 的失败方向(AC4)。
package codex

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tally/internal/agentruntime"
	"tally/test/runtimeconformance"
)

// canaryCLISource 是隔离验证专用替身 CLI 源码(仅标准库)。行为锚定见文件头。
const canaryCLISource = `package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func emit(v map[string]any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	fmt.Println(string(b))
}

func readFileOrEmpty(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func appendLine(path, line string) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	_, _ = fmt.Fprintln(f, line)
}

func main() {
	args := os.Args[1:]
	if len(args) == 1 && args[0] == "--version" {
		fmt.Println("codex-cli 0.153.0-canary")
		return
	}
	if len(args) == 0 || args[0] != "exec" {
		os.Exit(2)
	}
	rest := args[1:]
	if len(rest) > 0 && (rest[0] == "resume" || rest[0] == "fork") {
		rest = rest[1:]
		if len(rest) > 0 && !strings.HasPrefix(rest[0], "-") {
			rest = rest[1:]
		}
	}
	var outFile string
	var prompt []string
	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case "--json", "--skip-git-repo-check":
		case "-C", "--cd", "-s", "--sandbox", "-a", "--ask-for-approval",
			"-m", "--model", "--output-schema", "-o", "--output-last-message":
			if i+1 >= len(rest) {
				os.Exit(2)
			}
			if rest[i] == "-o" || rest[i] == "--output-last-message" {
				outFile = rest[i+1]
			}
			i++
		default:
			prompt = append(prompt, rest[i])
		}
	}
	instr := strings.Join(prompt, " ")

	emit(map[string]any{"type": "thread.started", "thread_id": "canary-thread"})
	emit(map[string]any{"type": "turn.started"})

	home := os.Getenv("CODEX_HOME")
	if _, err := os.Stat(filepath.Join(home, "auth.json")); err != nil {
		// 探针记录 §3 的实测 401 形态:error 事件 + error item,exit 0。
		emit(map[string]any{"type": "error", "message": "unexpected status 401 Unauthorized: Missing bearer or basic authentication in header"})
		emit(map[string]any{"type": "item.completed", "item": map[string]any{"id": "item_0", "type": "error", "message": "unexpected status 401 Unauthorized: Missing bearer or basic authentication in header"}})
		return
	}

	// 隔离面:本会话 home 的 rollout,先读后写 —— 只暴露先前 run 的状态。
	roll := filepath.Join(home, "sessions", "rollout.jsonl")
	prior := readFileOrEmpty(roll)
	_ = os.MkdirAll(filepath.Join(home, "sessions"), 0o755)
	appendLine(roll, instr)

	// 泄漏面(仅测试注入):TALLY_SPIKE_LEAK_HOME 模拟「隔离缺失/共享状态
	// 通道」。生产 runtime 不设置该变量;它只用于验证 canary 对跨 run 可观测
	// 状态的裁决方向(AC4:可观测即 FAIL)。
	final := "home=" + home + "\n-- prior rollout --\n" + prior
	if leak := os.Getenv("TALLY_SPIKE_LEAK_HOME"); leak != "" {
		lroll := filepath.Join(leak, "rollout.jsonl")
		final += "\n-- leak rollout --\n" + readFileOrEmpty(lroll)
		_ = os.MkdirAll(leak, 0o755)
		appendLine(lroll, instr)
	}
	if outFile != "" {
		_ = os.WriteFile(outFile, []byte(final), 0o644)
	}
}
`

// buildCanaryCLI 现场构建隔离验证替身 CLI(与 TestMain 构建 conformance 替身
// 同一模式:仅标准库,无网络依赖)。
func buildCanaryCLI(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	src := filepath.Join(dir, "main.go")
	if err := os.WriteFile(src, []byte(canaryCLISource), 0o644); err != nil {
		t.Fatalf("写 canary 替身源码失败: %v", err)
	}
	bin := filepath.Join(dir, "canary-codex.exe")
	out, err := exec.Command("go", "build", "-o", bin, src).CombinedOutput()
	if err != nil {
		t.Fatalf("构建 canary 替身失败: %v\n%s", err, out)
	}
	return bin
}

// writeOwnerAuth 在 owner home 下落最小 file auth(REF-CODEX-AUTH:auth.json
// 是 credential material,移交它 ≠ 移交会话/指令状态)。
func writeOwnerAuth(t *testing.T, home, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(home, "auth.json"), []byte(content), 0o600); err != nil {
		t.Fatalf("写 owner auth.json 失败: %v", err)
	}
}

// TestSessionIsolationDeclarationIsEvidenceGated(AC3):per-run home 隔离是
// 无条件生产行为,但 session.isolation 声明只跟证据走 —— 零值配置保持未声明
// (DECISION_LOCKS #22:canary 对真实 CLI 通过之前不声明);显式置位
// SessionIsolationVerified 才声明。Descriptor 与 Snapshot 一致。
func TestSessionIsolationDeclarationIsEvidenceGated(t *testing.T) {
	base := []agentruntime.Capability{
		agentruntime.CapEventStreaming,
		agentruntime.CapModelSelect,
		agentruntime.CapOutputStructured,
		agentruntime.CapSandboxNative,
		agentruntime.CapSessionFork,
		agentruntime.CapSessionPersist,
		agentruntime.CapSessionResume,
	}

	withheld := New(Config{BinPath: fakeCodexBin})
	if d := withheld.Descriptor(); d.Capabilities.Has(agentruntime.CapSessionIsolation) {
		t.Fatal("零值配置必须保持 session.isolation 未声明(真实 CLI canary 证据不存在)")
	} else if got, want := d.Capabilities.Sorted(), agentruntime.SortedCapabilities(base); len(got) != len(want) {
		t.Fatalf("基础声明不应变化: want %v, got %v", want, got)
	}
	if snap, err := withheld.Snapshot(context.Background()); err != nil {
		t.Fatalf("Snapshot: %v", err)
	} else if snap.Capabilities == nil || (agentruntime.NewCapabilitySet(snap.Capabilities...).Has(agentruntime.CapSessionIsolation)) {
		t.Fatalf("Snapshot 也不得声明 session.isolation: %v", snap.Capabilities)
	}

	declared := New(Config{BinPath: fakeCodexBin, SessionIsolationVerified: true})
	d := declared.Descriptor()
	if !d.Capabilities.Has(agentruntime.CapSessionIsolation) {
		t.Fatal("SessionIsolationVerified 置位后应声明 session.isolation")
	}
	snap, err := declared.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if !agentruntime.NewCapabilitySet(snap.Capabilities...).Has(agentruntime.CapSessionIsolation) {
		t.Fatalf("Snapshot 应携带 session.isolation 声明: %v", snap.Capabilities)
	}
}

// TestPerSessionHomeBoundaryAndAuthHandoff(AC1/AC2):同一 runtime 上的每个
// 会话拿到彼此独立的全新 home;跨 home 只移交 auth.json(复制而非链接);
// owner home 的 config.toml/AGENTS.md/sessions 一概不流入(REF-CODEX-INSTR:
// 指令状态不随会话携带);home 生命周期跟随会话,Close 即回收,且不影响
// 其它会话的 home。
func TestPerSessionHomeBoundaryAndAuthHandoff(t *testing.T) {
	owner := t.TempDir()
	const authContent = `{"tokens":{"access_token":"owner"}}`
	writeOwnerAuth(t, owner, authContent)
	// owner home 的其它状态:指令/配置/既有会话 rollout —— 都不得流入。
	for rel, content := range map[string]string{
		"config.toml":            "[model]\nname = \"gpt-5\"",
		"AGENTS.md":              "owner-global-instruction",
		"sessions/rollout.jsonl": "prior-owner-run-state",
	} {
		p := filepath.Join(owner, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("建 owner 状态目录失败: %v", err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatalf("写 owner 状态失败: %v", err)
		}
	}

	rt := New(Config{BinPath: fakeCodexBin, Env: map[string]string{"CODEX_HOME": owner}})
	ws := t.TempDir()
	s1 := startSession(t, rt, ws)
	s2 := startSession(t, rt, ws)
	if s1.home == "" || s2.home == "" {
		t.Fatalf("会话应有 per-run home: %q %q", s1.home, s2.home)
	}
	if s1.home == s2.home {
		t.Fatalf("不同会话必须持有不同 home(隔离边界): %q", s1.home)
	}

	for _, s := range []*Session{s1, s2} {
		entries, err := os.ReadDir(s.home)
		if err != nil {
			t.Fatalf("读会话 home 失败: %v", err)
		}
		if len(entries) != 1 || entries[0].Name() != "auth.json" {
			t.Fatalf("会话 home 只应含移交的 auth.json,得到 %v", entries)
		}
		info, err := entries[0].Info()
		if err != nil {
			t.Fatalf("stat auth.json: %v", err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			t.Fatal("auth 移交必须是复制,不是 symlink")
		}
		data, err := os.ReadFile(filepath.Join(s.home, "auth.json"))
		if err != nil {
			t.Fatalf("读移交的 auth.json 失败: %v", err)
		}
		if string(data) != authContent {
			t.Fatalf("移交内容应原样: %q", data)
		}
	}

	ctx := context.Background()
	if err := s1.Close(ctx); err != nil {
		t.Fatalf("Close(s1): %v", err)
	}
	if _, err := os.Stat(s1.home); !os.IsNotExist(err) {
		t.Fatalf("Close 后会话 home 应回收: %v", err)
	}
	if _, err := os.Stat(s2.home); err != nil {
		t.Fatalf("其它会话的 home 不受影响: %v", err)
	}
	if data, err := os.ReadFile(filepath.Join(owner, "auth.json")); err != nil || string(data) != authContent {
		t.Fatalf("owner home 不得被改动: %v %q", err, data)
	}
	if err := s2.Close(ctx); err != nil {
		t.Fatalf("Close(s2): %v", err)
	}
}

// TestSpawnUsesSessionHomeAndContinuityWithinSession(AC1/AC2):被 spawn 进程
// 的 CODEX_HOME 是本会话隔离 home(压倒 Config.Env 与继承 env);同一会话的
// 多个 Execute 共享该 home —— 会话内先前 turn 的指令状态对后续 turn 可观测
// (thread 连续性),这正是 per-session 而非 per-Execute 边界的语义。
func TestSpawnUsesSessionHomeAndContinuityWithinSession(t *testing.T) {
	owner := t.TempDir()
	writeOwnerAuth(t, owner, `{"tokens":{"access_token":"owner"}}`)
	rt := New(Config{
		BinPath: buildCanaryCLI(t),
		Env:     map[string]string{"CODEX_HOME": owner, "TALLY_SPIKE_LEAK_HOME": ""},
	})
	ws := t.TempDir()
	s := startSession(t, rt, ws)
	defer func() { _ = s.Close(context.Background()) }()

	ctx := context.Background()
	res1, err := s.Execute(ctx, agentruntime.ExecuteRequest{Instruction: "first-turn-marker"})
	if err != nil || res1.Status != agentruntime.RunCompleted {
		t.Fatalf("turn1 应完成(认证经 auth.json 移交可用): status=%s err=%v summary=%q", res1.Status, err, res1.Summary)
	}
	if !strings.Contains(res1.Summary, "home="+s.home) {
		t.Fatalf("spawn 的 CODEX_HOME 应是本会话隔离 home: summary=%q home=%q", res1.Summary, s.home)
	}
	if strings.Contains(res1.Summary, "first-turn-marker") {
		t.Fatalf("turn1 不应看到自己写入后的 rollout(先读后写): %q", res1.Summary)
	}

	res2, err := s.Execute(ctx, agentruntime.ExecuteRequest{Instruction: "second-turn-marker"})
	if err != nil || res2.Status != agentruntime.RunCompleted {
		t.Fatalf("turn2 应完成: status=%s err=%v", res2.Status, err)
	}
	if !strings.Contains(res2.Summary, "home="+s.home) || !strings.Contains(res2.Summary, "first-turn-marker") {
		t.Fatalf("同会话后续 turn 应共享同一 home 并观测到先前 turn(会话内连续性): %q", res2.Summary)
	}
}

// TestSessionIsolationCanaryPassesOnCodex(AC1/AC2):共享 conformance
// Session Isolation Canary 对 CodexRuntime(替身 CLI,声明经
// SessionIsolationVerified 置位)通过 —— 注入会话的 canary 对全新检索会话
// protocol-observable 不可检索。canary 全程完成同时证明 auth.json 移交使认证
// 在隔离 home 下可用(替身无 auth 即 401 失败)。
func TestSessionIsolationCanaryPassesOnCodex(t *testing.T) {
	owner := t.TempDir()
	writeOwnerAuth(t, owner, `{"tokens":{"access_token":"owner"}}`)
	rt := New(Config{
		BinPath:                  buildCanaryCLI(t),
		Env:                      map[string]string{"CODEX_HOME": owner},
		SessionIsolationVerified: true,
	})
	rep := runtimeconformance.CheckSessionIsolation(context.Background(), rt,
		runtimeconformance.SessionIsolationOptions{Timeout: 30 * time.Second})
	if !rep.CapabilityDeclared {
		t.Fatalf("声明的 runtime:CapabilityDeclared 应为 true: %+v", rep)
	}
	if !rep.CapabilityAvailable {
		t.Fatalf("per-run home 隔离下 canary 应通过(AC1): %s", rep.FailureText())
	}
	if len(rep.Failures) != 0 || rep.UnavailableReason != "" {
		t.Fatalf("通过时不应有失败/不可用记录: %s %q", rep.FailureText(), rep.UnavailableReason)
	}
}

// TestSessionIsolationCanaryWithheldWithoutEvidence(AC3):没有真实 CLI canary
// 证据(SessionIsolationVerified 零值)时,声明保持收回,共享 canary 按门控
// 不运行 —— 实现 存在 ≠ 证据 存在。
func TestSessionIsolationCanaryWithheldWithoutEvidence(t *testing.T) {
	owner := t.TempDir()
	writeOwnerAuth(t, owner, `{"tokens":{"access_token":"owner"}}`)
	rt := New(Config{
		BinPath: buildCanaryCLI(t),
		Env:     map[string]string{"CODEX_HOME": owner},
	})
	rep := runtimeconformance.CheckSessionIsolation(context.Background(), rt,
		runtimeconformance.SessionIsolationOptions{Timeout: 30 * time.Second})
	if rep.CapabilityDeclared || rep.CapabilityAvailable {
		t.Fatalf("无证据配置不得声明/ qualify: %+v", rep)
	}
	if rep.UnavailableReason == "" {
		t.Fatal("应记录声明收回的原因")
	}
	if len(rep.Failures) != 0 {
		t.Fatalf("探针未运行,不应产生失败详情: %s", rep.FailureText())
	}
}

// TestSessionIsolationCanaryFailsWhenCrossRunStateObservable(AC4):跨 run
// 状态一旦 protocol-observable(经 TALLY_SPIKE_LEAK_HOME 注入的共享状态通道
// 模拟隔离缺失/回归),共享 canary 必须对 CodexRuntime 判 FAIL —— 泄漏令牌
// 经 claim summary 可检索即 capability unavailable。
func TestSessionIsolationCanaryFailsWhenCrossRunStateObservable(t *testing.T) {
	owner := t.TempDir()
	writeOwnerAuth(t, owner, `{"tokens":{"access_token":"owner"}}`)
	leak := t.TempDir() // 模拟共享 home:注入会话状态对后续会话可观测
	rt := New(Config{
		BinPath: buildCanaryCLI(t),
		Env: map[string]string{
			"CODEX_HOME":            owner,
			"TALLY_SPIKE_LEAK_HOME": leak,
		},
		SessionIsolationVerified: true,
	})
	rep := runtimeconformance.CheckSessionIsolation(context.Background(), rt,
		runtimeconformance.SessionIsolationOptions{Timeout: 30 * time.Second})
	if rep.CapabilityAvailable {
		t.Fatalf("跨 run 状态可观测时 canary 必须判 FAIL(AC4): %+v", rep)
	}
	if rep.UnavailableReason != "" {
		t.Fatalf("探针已运行,不可用应由 Failures 承载: %q", rep.UnavailableReason)
	}
	if txt := rep.FailureText(); !strings.Contains(txt, "claim summary") {
		t.Fatalf("失败详情应指明泄漏通道: %s", txt)
	}
}

// TestNoAuthHandoffSurfacesAsClaim(AC2 的另一面):owner home 无 auth.json
// (如 keyring 部署,REF-CODEX-AUTH:键由 codex_home 路径派生,与 per-run
// home 不兼容)时不伪造认证 —— 隔离照常成立,认证缺失以 401 RunFailed claim
// 显形,不被隐藏。
func TestNoAuthHandoffSurfacesAsClaim(t *testing.T) {
	owner := t.TempDir() // 无 auth.json
	rt := New(Config{
		BinPath: buildCanaryCLI(t),
		Env:     map[string]string{"CODEX_HOME": owner},
	})
	ws := t.TempDir()
	s := startSession(t, rt, ws)
	defer func() { _ = s.Close(context.Background()) }()

	res, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "x"})
	if err != nil {
		t.Fatalf("认证失败是 claim 不是 error: %v", err)
	}
	if res.Status != agentruntime.RunFailed || !strings.Contains(res.Summary, "401") {
		t.Fatalf("无 auth 移交的隔离 home 应以 401 claim 显形: status=%s summary=%q", res.Status, res.Summary)
	}
}

// TestOwnerCodexHomePrecedence:auth 移交来源(owner home)的解析优先级
// Config.Env > 进程 env > ~/.codex。
func TestOwnerCodexHomePrecedence(t *testing.T) {
	t.Setenv("CODEX_HOME", `C:\proc-env-codex`)
	if got := ownerCodexHome(map[string]string{"CODEX_HOME": `C:\cfg-codex`}); got != `C:\cfg-codex` {
		t.Fatalf("Config.Env 应优先: got %q", got)
	}
	if got := ownerCodexHome(map[string]string{}); got != `C:\proc-env-codex` {
		t.Fatalf("空 Config.Env 应回落进程 env: got %q", got)
	}
	t.Setenv("CODEX_HOME", "")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("UserHomeDir 不可用: %v", err)
	}
	if got, want := ownerCodexHome(nil), filepath.Join(home, ".codex"); got != want {
		t.Fatalf("最终回落 ~/.codex: want %q, got %q", want, got)
	}
}
