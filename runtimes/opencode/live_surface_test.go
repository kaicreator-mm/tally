package opencode

// 真实 opencode 表面(CLI 部署 + 凭据门控,环境前提)。
//
// TestLiveOpenCodeSurface 在 TALLY_OPENCODE_LIVE=1 时对真实 opencode CLI
// 验证两件事:
//  1. 钉版身份闸:Snapshot 现场观察 `--version` 并与 PinnedOpenCodeVersion
//     比对(版本不符即失败 —— 适配器契约按精确版本钉住,版本漂移必须显形);
//  2. 一次真实回合(至多少量模型 turn):指令要求在 workspace 内创建
//     opencode-adapter-smoke.txt(内容 OPENCODE_ADAPTER_OK),断言
//     RunCompleted 且文件真实落地 —— cwd 钉定与 agent 产出的效果级证据。
//
// 未请求 live 时 SKIP;一旦 TALLY_OPENCODE_LIVE=1 显式请求,CLI/key 缺失
// 都是 gate 配置失败而不是 SKIP。CLI 使用与生产 candidate 相同的显式
// BinPath 配置面(TALLY_OPENCODE_BIN),不做隐式发现。
//
// workspace 可经 TALLY_OPENCODE_LIVE_WORKSPACE 覆盖为调用方准备的 git 仓库
// (证据收集场景:git diff 取证);缺省自建 t.TempDir() + git init。
// 认证走环境变量识别面(providers list 观测):进程环境或 Config.Env 持有
// DEEPSEEK_API_KEY 即可;无凭据时 CLI 可能落免凭据默认模型(观测行为),
// 因此本测试不假设「无 key 必败」。

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tally/internal/agentruntime"
)

func TestLiveOpenCodeSurface(t *testing.T) {
	if os.Getenv("TALLY_OPENCODE_LIVE") == "" {
		t.Skip("TALLY_OPENCODE_LIVE 未设置:跳过真实 opencode 表面(包注释记录了钉住版本的 Phase A live 观测)")
	}
	bin := strings.TrimSpace(os.Getenv("TALLY_OPENCODE_BIN"))
	if bin == "" {
		t.Fatal("TALLY_OPENCODE_LIVE 已启用但缺 TALLY_OPENCODE_BIN:live gate 必须使用显式二进制路径(canary 候选不做隐式发现)")
	}
	if err := requireFile(bin, "env TALLY_OPENCODE_BIN"); err != nil {
		t.Fatalf("TALLY_OPENCODE_LIVE 已启用但 CLI 不可用: %v", err)
	}

	// 1) 钉版身份闸(零模型成本)。
	rt := New(Config{BinPath: bin})
	snap, err := rt.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot(真实 CLI): %v", err)
	}
	t.Logf("真实 opencode 身份: version=%s bin=%s", snap.CLIVersion, snap.BinaryPath)

	// 2) 一次真实回合:workspace 内创建 smoke 文件。
	ws := strings.TrimSpace(os.Getenv("TALLY_OPENCODE_LIVE_WORKSPACE"))
	if ws == "" {
		ws = t.TempDir()
		if out, gerr := exec.Command("git", "init", ws).CombinedOutput(); gerr != nil {
			t.Fatalf("git init workspace: %v: %s", gerr, string(out))
		}
	}
	t.Logf("live workspace: %s", ws)

	smoke := "Create a file named opencode-adapter-smoke.txt containing exactly OPENCODE_ADAPTER_OK"
	s, err := rt.Start(context.Background(), agentruntime.StartRequest{Workspace: ws})
	if err != nil {
		t.Fatalf("Start(真实 CLI): %v", err)
	}
	defer func() { _ = s.Close(context.Background()) }()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	res, err := s.Execute(ctx, agentruntime.ExecuteRequest{Instruction: smoke})
	if err != nil {
		t.Fatalf("真实回合 Execute: %v", err)
	}
	if res.Status != agentruntime.RunCompleted || strings.TrimSpace(res.Summary) == "" {
		t.Fatalf("真实回合应完成且回答非空: %s summary=%q", res.Status, truncate(res.Summary, 300))
	}
	t.Logf("真实回合完成: summary=%q usage=%+v session=%s", truncForLog(res.Summary), res.Usage, res.SessionID)
	// IdentityReporter 是 Session 的可选扩展(类型断言获取)。
	oc, ok := s.(*Session)
	if !ok {
		t.Fatalf("Start 应返回 *opencode.Session, got %T", s)
	}
	if id := oc.Identity(); id.RuntimeVersion != PinnedOpenCodeVersion {
		t.Fatalf("Identity.RuntimeVersion 应为钉住版本: got %+v", id)
	}

	// 效果级证据:agent 真实创建了文件且内容正确(经 spawn cwd 钉定生效)。
	data, rerr := os.ReadFile(filepath.Join(ws, "opencode-adapter-smoke.txt"))
	if rerr != nil {
		t.Fatalf("agent 应在 workspace 内创建 smoke 文件: %v", rerr)
	}
	if got := strings.TrimSpace(string(data)); got != "OPENCODE_ADAPTER_OK" {
		t.Fatalf("smoke 文件内容不符: got %q", got)
	}
	t.Logf("smoke 文件效果级验证通过: %s", filepath.Join(ws, "opencode-adapter-smoke.txt"))
}

// truncForLog 供日志截断。
func truncForLog(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return s
}
