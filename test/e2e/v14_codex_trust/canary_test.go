//go:build integration

// Session Isolation Canary 对真实 codex CLI(codex-cli 0.153.0,经生产适配器
// 与 P0-03 per-run home 隔离)的先行验证(P0-04 AC 前置,DECISION_LOCKS #22):
//
//	canary 只可能对「声明了 session.isolation」的 runtime 运行,而声明又以
//	canary 证据为门 —— 因此本测试以 SessionIsolationVerified 置位构造一个
//	仅用于取证的 runtime,运行共享 conformance 的 CheckSessionIsolation:
//	  - 通过 → 证据成立,E2E 才允许以该配置声明 session.isolation;
//	  - 失败 → 本测试显形失败,生产配置保持未声明,不得削弱 canary。
//
// 裁决经 sync.Once 在一次测试进程内至多真实执行一轮(有界模型用量),结果
// 由 standard + held-out 测试共享引用。

package codextrust

import (
	"context"
	"testing"
	"time"

	"tally/runtimes/codex"
	"tally/test/runtimeconformance"
)

// p04RunCanaryOnce 对真实 codex runtime(声明门仅为本探针置位)运行共享
// conformance Session Isolation Canary,一次进程至多一轮。返回 typed 裁决。
func p04RunCanaryOnce(t *testing.T) *runtimeconformance.SessionIsolationReport {
	t.Helper()
	cfg := p04CodexConfig(t, true) // 取证专用:声明门置位才能运行 canary
	p04CanaryOnce.Do(func() {
		rt := codex.New(cfg)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()
		p04CanaryReport = runtimeconformance.CheckSessionIsolation(ctx, rt,
			runtimeconformance.SessionIsolationOptions{Timeout: 14 * time.Minute})
	})
	if p04CanaryReport == nil {
		t.Fatal("Session Isolation Canary 未运行")
	}
	return p04CanaryReport
}

// TestP04SessionIsolationCanaryOnRealCodex 是 canary 证据的显形入口:
// 通过 = CapabilityAvailable 且无失败详情;失败 = 真实失败(AC 前置不成立),
// 上层配置必须保持 session.isolation 未声明。
func TestP04SessionIsolationCanaryOnRealCodex(t *testing.T) {
	rep := p04RunCanaryOnce(t)

	// 证据记录(裁决字段原样进测试日志,供 P0_CODEX_TRUST 报告引用)。
	t.Logf("canary runtime=%q CapabilityDeclared=%v CapabilityAvailable=%v InjectionObserved=%v UnavailableReason=%q ResidualRisk=%q",
		rep.RuntimeName, rep.CapabilityDeclared, rep.CapabilityAvailable, rep.InjectionObserved, rep.UnavailableReason, rep.ResidualRisk)

	if !rep.CapabilityDeclared {
		t.Fatalf("取证配置声明了 session.isolation,CapabilityDeclared 应为 true: %+v", rep)
	}
	if !rep.CapabilityAvailable {
		t.Fatalf("Session Isolation Canary 对真实 codex CLI 未通过 —— session.isolation 证据不成立,生产配置不得声明:\n%s",
			rep.FailureText())
	}
	if len(rep.Failures) != 0 || rep.UnavailableReason != "" {
		t.Fatalf("通过时不应有失败/不可用记录: %s %q", rep.FailureText(), rep.UnavailableReason)
	}
	t.Logf("canary 注入/检索会话均经真实 codex 回合完成;隔离证据成立(canary token 前缀 %q… 不落任何被跟踪文件)",
		safePrefix(rep.Canary))
}

func safePrefix(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	return s
}
