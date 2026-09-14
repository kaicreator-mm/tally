package fake

import (
	"testing"

	"tally/test/runtimeconformance"
)

// FakeRuntime 声明 session.isolation(PRD §10.4):每个 Session 是互相独立的
// 有界状态对象,不存在任何跨 session 的 Protocol-observable 检索路径。
// Session Isolation Canary(PRD §35,任务 P1-30)是该声明的 conformance
// 证据,也是 VerificationPlanner Role 资格的必要条件 —— 声明必须经 canary
// 验证为真(capability declaration truthful)。
func TestFakeRuntimePassesSessionIsolationCanary(t *testing.T) {
	rep := runtimeconformance.RunSessionIsolationCanary(t, New(Config{Now: fixedNow}), runtimeconformance.SessionIsolationOptions{})
	if !rep.CapabilityAvailable {
		t.Fatalf("FakeRuntime 应通过 Session Isolation Canary: %s", rep.FailureText())
	}
	if !rep.CapabilityDeclared {
		t.Error("FakeRuntime 声明了 session.isolation,报告应记录该声明")
	}
	if !rep.InjectionObserved {
		t.Errorf("注入应经协议可观测(FakeRuntime 在本会话事件中原样回显 instruction): %+v", rep)
	}
	if rep.ResidualRisk == "" {
		t.Error("报告必须携带 declared residual risk 声明(PRD §3.2.1:provider 内部不可观测状态不在 canary 证明范围)")
	}
	if len(rep.Failures) != 0 {
		t.Errorf("通过时不应有失败详情: %s", rep.FailureText())
	}
}
