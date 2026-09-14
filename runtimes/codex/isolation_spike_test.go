// A0-02 Codex Session Isolation Spike(任务 docs/tasks/v1.4/A0/A0-02.md)。
//
// A0-02 已定案(DECISION_LOCKS #22):per-run CODEX_HOME + 仅 auth.json 移交。
// P0-03 据此实现了生产行为(isolation.go);本文件保留 spike 的机理证据:
//   - 用同一个 Session Isolation Canary 对比 Fake 与 Codex 的声明门控(AC1):
//     零值 Codex 配置未声明 → canary 不适用 —— 声明以真实 CLI 上的 canary
//     证据为门(isolation_test.go);
//   - 用 spike 替身 CLI 复现探针记录 §6 的共享 CODEX_HOME 泄漏机理
//     (rollout 落在 $CODEX_HOME/sessions 下,跨 run 可检索);
//   - 验证 per-run CODEX_HOME + 仅 auth.json 移交的候选方案在生产 runtime
//     上成立:认证可用 + 跨 run 不可检索(AC2/AC4)。
//
// 替身 CLI 的行为锚定 L3 Reference Pack(openai/codex@553df1c6,Apache-2.0):
//   - REF-CODEX-ROLLOUT:session/rollout 数据根在 CODEX_HOME/sessions;
//   - REF-CODEX-AUTH:file auth 在 CODEX_HOME/auth.json,keyring 存储的键
//     由 codex_home 路径派生 —— per-run CODEX_HOME 会改变认证行为;
//   - REF-CODEX-INSTR:全局 AGENTS.md 从 CODEX_HOME 加载,CODEX_HOME 是
//     可观测指令状态,不是纯缓存。
//
// 替身只复现上述已探明的状态布局语义,不复制外部实现代码。
package codex

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tally/internal/agentruntime"
	"tally/runtimes/fake"
	"tally/test/runtimeconformance"
)

// spikeRetrieveMark 是 spike 替身的「检索先前会话内容」指令前缀:替身把
// $CODEX_HOME/sessions/rollout.jsonl 的全部内容作为最终消息返回,模拟真实
// Codex 里 rollout 对后续 run protocol-observable 的检索面(探针记录 §6)。
const spikeRetrieveMark = "TALLY_SPIKE_RETRIEVE"

// spikeCLISource 是 spike 专用替身 CLI 源码(仅标准库)。与 runtime_test.go
// 的 conformance 替身不同,它按 REF-CODEX-ROLLOUT/REF-CODEX-AUTH 把状态落到
// CODEX_HOME 下:
//   - auth:无 $CODEX_HOME/auth.json 即按探针记录 §3 的 401 形态收场
//     (error 事件 + error item,exit 0);
//   - rollout:每次 exec 把指令追加到 $CODEX_HOME/sessions/rollout.jsonl;
//   - 检索:指令以 spikeRetrieveMark 开头时,最终消息 = rollout 全文。
const spikeCLISource = `package main

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

func main() {
	args := os.Args[1:]
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

	emit(map[string]any{"type": "thread.started", "thread_id": "spike-thread"})
	emit(map[string]any{"type": "turn.started"})

	home := os.Getenv("CODEX_HOME")
	if _, err := os.Stat(filepath.Join(home, "auth.json")); err != nil {
		// 探针记录 §3 的实测 401 形态:error 事件 + error item,exit 0。
		emit(map[string]any{"type": "error", "message": "Reconnecting... 1/5 (unexpected status 401 Unauthorized: Missing bearer or basic authentication in header)"})
		emit(map[string]any{"type": "item.completed", "item": map[string]any{"id": "item_0", "type": "error", "message": "unexpected status 401 Unauthorized: Missing bearer or basic authentication in header"}})
		return
	}

	sessDir := filepath.Join(home, "sessions")
	if err := os.MkdirAll(sessDir, 0o755); err != nil {
		os.Exit(1)
	}
	roll := filepath.Join(sessDir, "rollout.jsonl")
	f, err := os.OpenFile(roll, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		os.Exit(1)
	}
	_, _ = fmt.Fprintln(f, instr)
	_ = f.Close()

	final := "ok"
	if strings.HasPrefix(instr, "` + spikeRetrieveMark + `") {
		data, _ := os.ReadFile(roll)
		final = "RETRIEVED:\n" + string(data)
	}
	if outFile != "" {
		_ = os.WriteFile(outFile, []byte(final), 0o644)
	}
}
`

// buildSpikeCLI 现场构建 spike 替身 CLI(与 TestMain 构建 conformance 替身
// 同一模式:仅标准库,无网络依赖)。
func buildSpikeCLI(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	src := filepath.Join(dir, "main.go")
	if err := os.WriteFile(src, []byte(spikeCLISource), 0o644); err != nil {
		t.Fatalf("写 spike 替身源码失败: %v", err)
	}
	bin := filepath.Join(dir, "spike-codex.exe")
	out, err := exec.Command("go", "build", "-o", bin, src).CombinedOutput()
	if err != nil {
		t.Fatalf("构建 spike 替身失败: %v\n%s", err, out)
	}
	return bin
}

// newSpikeToken 生成一次性 canary 令牌。
func newSpikeToken(t *testing.T) string {
	t.Helper()
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("生成 canary 令牌失败: %v", err)
	}
	return "tally-spike-canary-" + hex.EncodeToString(b)
}

// writeSpikeAuth 在给定 home 下落一个最小 file auth(REF-CODEX-AUTH:
// auth.json 是 credential material,移交它 ≠ 移交会话/指令状态)。
func writeSpikeAuth(t *testing.T, home string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(home, "auth.json"), []byte(`{"tokens":{"access_token":"spike"}}`), 0o600); err != nil {
		t.Fatalf("写 auth.json 失败: %v", err)
	}
}

// startSpikeSession 在指定 CODEX_HOME 的 spike runtime 上开新会话。
func startSpikeSession(t *testing.T, bin, home, ws string) agentruntime.Session {
	t.Helper()
	rt := New(Config{BinPath: bin, Env: map[string]string{"CODEX_HOME": home}})
	s, err := rt.Start(context.Background(), agentruntime.StartRequest{Workspace: ws})
	if err != nil {
		t.Fatalf("Start 失败: %v", err)
	}
	return s
}

// TestSpikeCanaryFakeVsCodex 用同一个 Session Isolation Canary 对比两个
// runtime 的声明门控(AC1):Fake 声明并通过;Codex 零值配置未声明 → canary
// 按门控不适用,CapabilityAvailable=false。声明以真实 CLI 上的 canary 证据为
// 门(DECISION_LOCKS #22;经 verified 配置的 canary 通过见 isolation_test.go)。
func TestSpikeCanaryFakeVsCodex(t *testing.T) {
	ctx := context.Background()

	fakeRep := runtimeconformance.CheckSessionIsolation(ctx, fake.New(fake.Config{}), runtimeconformance.SessionIsolationOptions{})
	if !fakeRep.CapabilityDeclared {
		t.Error("FakeRuntime 应声明 session.isolation")
	}
	if !fakeRep.CapabilityAvailable {
		t.Errorf("FakeRuntime canary 应通过: %s", fakeRep.FailureText())
	}

	codexRep := runtimeconformance.CheckSessionIsolation(ctx, New(Config{BinPath: fakeCodexBin}), runtimeconformance.SessionIsolationOptions{})
	if codexRep.CapabilityDeclared {
		t.Error("零值配置不得声明 session.isolation(证据门控,DECISION_LOCKS #22)")
	}
	if codexRep.CapabilityAvailable {
		t.Error("未声明能力时 CapabilityAvailable 必须为 false")
	}
	if codexRep.UnavailableReason == "" {
		t.Error("未声明能力时应记录 UnavailableReason")
	}
	t.Logf("canary 对比: fake declared=%v available=%v; codex declared=%v available=%v reason=%q",
		fakeRep.CapabilityDeclared, fakeRep.CapabilityAvailable,
		codexRep.CapabilityDeclared, codexRep.CapabilityAvailable, codexRep.UnavailableReason)
}

// TestSpikeSharedHomeNegativeControl 负向对照(机理级):两个 run 共享一个
// CODEX_HOME 时,后一个 run 能检索到前一个 run 注入的 canary —— 证明泄漏机理
// 真实存在(探针记录 §6),这就是 Session Isolation Canary 判失败、per-run
// home 成为生产路径(A0-02/DECISION_LOCKS #22)的依据。生产 runtime 自 P0-03
// 起无条件强制 per-session home,共享 home 无法再经适配器配置产生,故本对照
// 直接驱动替身 CLI 复现机理 —— canary 对「跨 run 状态可观测即 FAIL」的裁决
// 方向由 isolation_test.go 的
// TestSessionIsolationCanaryFailsWhenCrossRunStateObservable 经共享 canary
// 入口验证。
func TestSpikeSharedHomeNegativeControl(t *testing.T) {
	bin := buildSpikeCLI(t)
	home := t.TempDir()
	writeSpikeAuth(t, home)
	ws := t.TempDir()
	token := newSpikeToken(t)

	// runSpikeDirect 以给定 CODEX_HOME 直接驱动一次替身 exec,返回最终消息。
	runSpikeDirect := func(instr string) string {
		out := filepath.Join(t.TempDir(), "last-message.txt")
		cmd := exec.Command(bin, "exec", "--json", "--skip-git-repo-check",
			"-C", ws, "-s", "workspace-write", "-o", out, instr)
		cmd.Env = append(os.Environ(), "CODEX_HOME="+home)
		if b, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("spike 替身运行失败: %v\n%s", err, b)
		}
		data, rerr := os.ReadFile(out)
		if rerr != nil {
			t.Fatalf("读最终消息失败: %v", rerr)
		}
		return string(data)
	}

	if got := runSpikeDirect("请记住机密令牌: " + token); got != "ok" {
		t.Fatalf("run A 注入失败: %q", got)
	}
	got := runSpikeDirect(spikeRetrieveMark)
	if !strings.Contains(got, token) {
		t.Fatal("负向对照不成立:共享 CODEX_HOME 下后一 run 未能检索前一 run 的 canary")
	}
	t.Logf("负向对照成立:共享 home 下后一 run 检索到前一 run 的 canary(泄漏机理复现)")
}

// TestSpikeIsolatedHomeCandidate 生产方案的运行时验证:run B 使用全新
// per-run CODEX_HOME(runtime 强制分配),只移交 auth.json(credential
// material),不移交 sessions/ 指令状态。预期:认证可用 + run A 的 canary
// 不可检索。
func TestSpikeIsolatedHomeCandidate(t *testing.T) {
	bin := buildSpikeCLI(t)
	ws := t.TempDir()
	token := newSpikeToken(t)

	homeA := t.TempDir()
	writeSpikeAuth(t, homeA)
	sA := startSpikeSession(t, bin, homeA, ws)
	defer func() { _ = sA.Close(context.Background()) }()
	if res, err := sA.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "请记住机密令牌: " + token}); err != nil || res.Status != agentruntime.RunCompleted {
		t.Fatalf("run A 注入失败: status=%s err=%v", res.Status, err)
	}

	homeB := t.TempDir()
	writeSpikeAuth(t, homeB) // 仅移交 auth.json;不复制 sessions/、不 symlink 整个 home
	sB := startSpikeSession(t, bin, homeB, ws)
	defer func() { _ = sB.Close(context.Background()) }()
	resB, err := sB.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: spikeRetrieveMark})
	if err != nil {
		t.Fatalf("run B 执行错误: %v", err)
	}
	if resB.Status != agentruntime.RunCompleted {
		t.Fatalf("隔离 home + auth.json 移交下认证应可用,得到 status=%s summary=%q", resB.Status, resB.Summary)
	}
	if strings.Contains(resB.Summary, token) {
		t.Fatal("隔离 home 下 run B 仍检索到 run A 的 canary:候选方案不成立")
	}
	t.Logf("生产方案成立:per-run CODEX_HOME + 仅 auth.json 移交 → 认证可用且跨 run 检索不到 canary")
}

// TestSpikeIsolatedHomeWithoutAuthHandoff 记录认证耦合事实(AC2):file
// auth 位于 CODEX_HOME/auth.json,per-run 隔离 home 若不移交 auth 材料,
// 新 run 立即 401 —— 「认证可用」与「会话/指令隔离」是两件事,隔离方案
// 必须显式处理 auth 移交;keyring 场景下键由 codex_home 路径派生
// (REF-CODEX-AUTH),换 home 同样改变认证行为。
func TestSpikeIsolatedHomeWithoutAuthHandoff(t *testing.T) {
	bin := buildSpikeCLI(t)
	ws := t.TempDir()

	homeA := t.TempDir()
	writeSpikeAuth(t, homeA)
	sA := startSpikeSession(t, bin, homeA, ws)
	defer func() { _ = sA.Close(context.Background()) }()
	if res, err := sA.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "请记住机密令牌: " + newSpikeToken(t)}); err != nil || res.Status != agentruntime.RunCompleted {
		t.Fatalf("run A 注入失败: status=%s err=%v", res.Status, err)
	}

	homeB := t.TempDir() // 不移交 auth.json
	sB := startSpikeSession(t, bin, homeB, ws)
	defer func() { _ = sB.Close(context.Background()) }()
	resB, err := sB.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: spikeRetrieveMark})
	if err != nil {
		t.Fatalf("run B 执行错误: %v", err)
	}
	if resB.Status != agentruntime.RunFailed || !strings.Contains(resB.Summary, "401") {
		t.Fatalf("无 auth 移交的隔离 home 应按 401 失败,得到 status=%s summary=%q", resB.Status, resB.Summary)
	}
	t.Logf("认证耦合记录:隔离 home 无 auth.json → 401(file auth 不随新 home 自动可用)")
}
