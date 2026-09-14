package policy

import (
	"errors"
	"slices"
	"testing"
	"time"
)

func mustValid(t *testing.T, p ExecutionPolicy) ExecutionPolicy {
	t.Helper()
	if err := p.Validate(); err != nil {
		t.Fatalf("策略应合法: %v", err)
	}
	return p
}

// AC1/AC2/AC3:默认策略必须是「断网 + 结构化 shell + 有限预算」的保守形态。
func TestDefaultExecutionPolicyIsConservative(t *testing.T) {
	p := mustValid(t, DefaultExecutionPolicy())

	// AC1:默认编码运行网络 deny,且不携带任何放行主机。
	if p.Network.Mode != NetworkDeny {
		t.Errorf("默认 Network.Mode = %q, 期望 %q", p.Network.Mode, NetworkDeny)
	}
	if len(p.Network.AllowedHosts) != 0 {
		t.Errorf("默认策略不应放行任何主机: %v", p.Network.AllowedHosts)
	}

	// AC2:结构化命令是默认形态,shell 默认关闭。
	if p.Shell.AllowShell {
		t.Error("默认 ShellPolicy.AllowShell 必须是 false")
	}
	var zero ExecutionPolicy
	if zero.Shell.AllowShell {
		t.Error("零值 ExecutionPolicy 的 shell 必须是 false(默认关闭不能依赖构造函数)")
	}

	// AC3:retry/升级/预算全部有限。
	if p.Retry.MaxIterations != 4 {
		t.Errorf("默认 Retry.MaxIterations = %d, 期望 4", p.Retry.MaxIterations)
	}
	if p.Budgets.MaxTokens != 180000 {
		t.Errorf("默认 Budgets.MaxTokens = %d, 期望 180000", p.Budgets.MaxTokens)
	}
	if p.Budgets.MaxWallTime != 30*time.Minute {
		t.Errorf("默认 Budgets.MaxWallTime = %v, 期望 30m", p.Budgets.MaxWallTime)
	}
	if p.Escalation.AfterFailedIterations != 2 || p.Escalation.TargetProfile != "strong-coding" {
		t.Errorf("默认 Escalation = %+v, 期望 {2 strong-coding}", p.Escalation)
	}

	// 硬停止条件默认齐全。
	for _, hard := range hardStopConditions {
		if !slices.Contains(p.StopConditions, hard) {
			t.Errorf("默认策略缺少硬停止条件 %q", hard)
		}
	}

	// 默认不授予任何凭据。
	if len(p.Credentials.ScopedSecrets) != 0 {
		t.Errorf("默认策略不应授予凭据: %v", p.Credentials.ScopedSecrets)
	}
}

// AC3:零值策略没有有限预算,必须被 Validate 拒绝 —— 有效性不能靠默认值假设。
func TestZeroValuePolicyFailsValidation(t *testing.T) {
	var p ExecutionPolicy
	if err := p.Validate(); !errors.Is(err, ErrInvalidPolicy) {
		t.Errorf("零值策略应报 ErrInvalidPolicy, got %v", err)
	}
}

// AC4(硬下限):任何来源的策略都必须满足硬下限;逐一破坏每一项都应被拒绝。
func TestValidateEnforcesHardMinima(t *testing.T) {
	mutations := map[string]func(*ExecutionPolicy){
		"retry=0":          func(p *ExecutionPolicy) { p.Retry.MaxIterations = 0 },
		"retry<0":          func(p *ExecutionPolicy) { p.Retry.MaxIterations = -1 },
		"tokens=0":         func(p *ExecutionPolicy) { p.Budgets.MaxTokens = 0 },
		"walltime=0":       func(p *ExecutionPolicy) { p.Budgets.MaxWallTime = 0 },
		"escalation=0":     func(p *ExecutionPolicy) { p.Escalation.AfterFailedIterations = 0 },
		"escalation>retry": func(p *ExecutionPolicy) { p.Escalation.AfterFailedIterations = 5 },
		"target_profile为空": func(p *ExecutionPolicy) { p.Escalation.TargetProfile = "" },
		"allow_hosts但无主机":  func(p *ExecutionPolicy) { p.Network = NetworkPolicy{Mode: NetworkAllowHosts} },
		"deny却携带主机":        func(p *ExecutionPolicy) { p.Network.AllowedHosts = []string{"api.example.com"} },
		"未知network模式":      func(p *ExecutionPolicy) { p.Network.Mode = "open" },
		"空secret引用":        func(p *ExecutionPolicy) { p.Credentials.ScopedSecrets = []string{"a", ""} },
		"重复secret引用":       func(p *ExecutionPolicy) { p.Credentials.ScopedSecrets = []string{"a", "a"} },
		"缺硬停止条件":           func(p *ExecutionPolicy) { p.StopConditions = p.StopConditions[1:] },
		"重复停止条件": func(p *ExecutionPolicy) {
			p.StopConditions = append(slices.Clone(p.StopConditions), StopProtectedEvidenceModified)
		},
		"空停止条件": func(p *ExecutionPolicy) { p.StopConditions = append(p.StopConditions, "") },
	}

	for name, mutate := range mutations {
		p := DefaultExecutionPolicy()
		mutate(&p)
		if err := p.Validate(); !errors.Is(err, ErrInvalidPolicy) {
			t.Errorf("%s: 期望 ErrInvalidPolicy, got %v", name, err)
		}
	}
}

// AC4:空 proposal 是无操作,结果与原策略等价且仍合法。
func TestResolveEmptyProposalIsNoOp(t *testing.T) {
	base := mustValid(t, DefaultExecutionPolicy())
	got, err := base.Resolve(Proposal{})
	if err != nil {
		t.Fatalf("空 proposal 不该报错: %v", err)
	}
	if !slices.Equal(got.Network.AllowedHosts, base.Network.AllowedHosts) ||
		got.Shell != base.Shell ||
		!slices.Equal(got.Credentials.ScopedSecrets, base.Credentials.ScopedSecrets) ||
		got.Retry != base.Retry || got.Budgets != base.Budgets || got.Escalation != base.Escalation ||
		!slices.Equal(got.StopConditions, base.StopConditions) {
		t.Errorf("空 proposal 改动了策略: %+v", got)
	}
}

// AC4(允许收紧):owner 建立的宽松基础策略可以被 proposal 逐项收紧;
// 硬停止条件在收紧后仍然齐全。
func TestResolveTightensEveryDimension(t *testing.T) {
	base := DefaultExecutionPolicy()
	base.Network = NetworkPolicy{Mode: NetworkAllowHosts, AllowedHosts: []string{"proxy.internal:8081", "registry.internal:443"}}
	base.Shell = ShellPolicy{AllowShell: true}
	base.Credentials = CredentialPolicy{ScopedSecrets: []string{"pkg/token", "mirror/token"}}
	base = mustValid(t, base)

	got, err := base.Resolve(Proposal{
		Network:        &NetworkPolicy{Mode: NetworkDeny},
		Shell:          &ShellPolicy{AllowShell: false},
		Credentials:    &CredentialPolicy{ScopedSecrets: []string{"mirror/token"}},
		Retry:          &RetryPolicy{MaxIterations: 2},
		Budgets:        &Budgets{MaxTokens: 100000, MaxWallTime: 10 * time.Minute},
		Escalation:     &EscalationPolicy{AfterFailedIterations: 1, TargetProfile: "strong-coding"},
		StopConditions: []StopCondition{"budget_exhausted"},
	})
	if err != nil {
		t.Fatalf("全维度收紧不该报错: %v", err)
	}
	if got.Network.Mode != NetworkDeny {
		t.Errorf("网络应收紧为 deny, got %q", got.Network.Mode)
	}
	if got.Shell.AllowShell {
		t.Error("shell 应收紧为 false")
	}
	if !slices.Equal(got.Credentials.ScopedSecrets, []string{"mirror/token"}) {
		t.Errorf("凭据应收窄: %v", got.Credentials.ScopedSecrets)
	}
	if got.Retry.MaxIterations != 2 || got.Budgets.MaxTokens != 100000 || got.Budgets.MaxWallTime != 10*time.Minute {
		t.Errorf("预算应收紧: %+v %+v", got.Retry, got.Budgets)
	}
	if got.Escalation.AfterFailedIterations != 1 {
		t.Errorf("升级应收紧为 1 轮, got %d", got.Escalation.AfterFailedIterations)
	}
	for _, hard := range hardStopConditions {
		if !slices.Contains(got.StopConditions, hard) {
			t.Errorf("收紧后丢失硬停止条件 %q", hard)
		}
	}
	if !slices.Contains(got.StopConditions, StopCondition("budget_exhausted")) {
		t.Error("自定义停止条件应被追加")
	}
	if err := got.Validate(); err != nil {
		t.Errorf("收紧结果应合法: %v", err)
	}
}

// AC4(禁止放宽):proposal 的任何放宽方向都必须被拒绝,且返回零值策略。
func TestResolveRejectsWidening(t *testing.T) {
	base := DefaultExecutionPolicy()
	base.Network = NetworkPolicy{Mode: NetworkAllowHosts, AllowedHosts: []string{"proxy.internal:8081"}}
	base.Credentials = CredentialPolicy{ScopedSecrets: []string{"pkg/token"}}
	base = mustValid(t, base)

	cases := map[string]Proposal{
		"重试上调":    {Retry: &RetryPolicy{MaxIterations: 5}},
		"token上调": {Budgets: &Budgets{MaxTokens: 180001, MaxWallTime: 30 * time.Minute}},
		"墙钟上调":    {Budgets: &Budgets{MaxTokens: 180000, MaxWallTime: 31 * time.Minute}},
		"deny改放行": {Network: &NetworkPolicy{Mode: NetworkAllowHosts, AllowedHosts: []string{"api.example.com"}}},
		"放行新增主机":  {Network: &NetworkPolicy{Mode: NetworkAllowHosts, AllowedHosts: []string{"proxy.internal:8081", "extra.internal"}}},
		"shell开启": {Shell: &ShellPolicy{AllowShell: true}},
		"凭据集合外引用": {Credentials: &CredentialPolicy{ScopedSecrets: []string{"pkg/token", "prod/db"}}},
		"推迟升级":    {Escalation: &EscalationPolicy{AfterFailedIterations: 3, TargetProfile: "strong-coding"}},
	}

	for name, proposal := range cases {
		got, err := base.Resolve(proposal)
		if !errors.Is(err, ErrWidening) {
			t.Errorf("%s: 期望 ErrWidening, got %v", name, err)
		}
		if err == nil && (got.Retry.MaxIterations != 0 || got.Budgets.MaxTokens != 0 || got.Escalation.TargetProfile != "") {
			t.Errorf("%s: 被拒时应返回零值策略", name)
		}
	}
}

// proposal 子策略自身非法(非放宽,而是畸形)必须以 ErrInvalidPolicy 拒绝。
func TestResolveRejectsInvalidProposalValues(t *testing.T) {
	base := mustValid(t, DefaultExecutionPolicy())
	cases := map[string]Proposal{
		"retry=0":      {Retry: &RetryPolicy{MaxIterations: 0}},
		"tokens=0":     {Budgets: &Budgets{MaxTokens: 0, MaxWallTime: time.Minute}},
		"walltime<0":   {Budgets: &Budgets{MaxTokens: 1, MaxWallTime: -time.Minute}},
		"escalation=0": {Escalation: &EscalationPolicy{AfterFailedIterations: 0, TargetProfile: "strong-coding"}},
		"profile为空":    {Escalation: &EscalationPolicy{AfterFailedIterations: 1}},
		"未知网络模式":       {Network: &NetworkPolicy{Mode: "mesh"}},
		"空停止条件":        {StopConditions: []StopCondition{""}},
	}

	for name, proposal := range cases {
		if _, err := base.Resolve(proposal); !errors.Is(err, ErrInvalidPolicy) {
			t.Errorf("%s: 期望 ErrInvalidPolicy, got %v", name, err)
		}
	}
}

// 合并结果必须整体合法:收紧 retry 到 1 后,已有的 2 轮升级阈值永远无法触发,拒绝。
func TestResolveValidatesMergedResult(t *testing.T) {
	base := mustValid(t, DefaultExecutionPolicy())
	if _, err := base.Resolve(Proposal{Retry: &RetryPolicy{MaxIterations: 1}}); !errors.Is(err, ErrInvalidPolicy) {
		t.Errorf("合并结果非法应报 ErrInvalidPolicy, got %v", err)
	}
}

// 停止条件只增不减:proposal 只列自定义条件时,硬条件仍然保留。
func TestResolveStopConditionsAdditiveOnly(t *testing.T) {
	base := mustValid(t, DefaultExecutionPolicy())
	got, err := base.Resolve(Proposal{StopConditions: []StopCondition{"budget_exhausted", StopProtectedEvidenceModified}})
	if err != nil {
		t.Fatalf("追加停止条件不该报错: %v", err)
	}
	for _, hard := range hardStopConditions {
		if !slices.Contains(got.StopConditions, hard) {
			t.Errorf("硬停止条件 %q 被丢掉", hard)
		}
	}
	if got, want := len(got.StopConditions), 4; got != want {
		t.Errorf("停止条件数 = %d, 期望 %d(无重复追加)", got, want)
	}
}

// 基础策略非法时 Resolve 必须失败:合并不是洗白非法策略的通道。
func TestResolveOnInvalidBaseFails(t *testing.T) {
	var base ExecutionPolicy
	if _, err := base.Resolve(Proposal{}); !errors.Is(err, ErrInvalidPolicy) {
		t.Errorf("非法基础策略应报 ErrInvalidPolicy, got %v", err)
	}
}
