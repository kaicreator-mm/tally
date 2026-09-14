package codex

// identity_test.go 验证 P0-02 的身份观察面:Session 经 IdentityReporter
// (P0-01 缝)自报的 ObservedIdentity 只含真实观察事实 ——
//
//   - CLI 版本来自现场 `codex --version` 探针(替身输出 "codex-cli
//     0.153.0-conformance"),绝不是适配器版本 DescriptorVersion("1");
//   - 会话身份来自 thread.started 观察的 codex thread id(resume 会话为
//     已建立的续用句柄);新会话首回合前如实未观察;
//   - 模型恒未观察:exec JSONL 表面(探针记录 §3)不自报模型,请求的
//     -m 档位绝不被当成实际生效模型上报(Decision Lock 15);
//   - 探针失败 = 未观察零值,不是 Start 失败,更不是伪造观察。

import (
	"context"
	"testing"

	"tally/internal/agentruntime"
)

// 编译期断言:*Session 实现 agentruntime.IdentityReporter(P0-01 可选缝)。
var _ agentruntime.IdentityReporter = (*Session)(nil)

func TestIdentityReportsActualCLIFactsNotAdapterConstants(t *testing.T) {
	rt := New(Config{BinPath: fakeCodexBin})
	ctx := context.Background()

	// 请求档位存在也绝不变成「实际生效模型」:期望值与观察值分开携带。
	s := startSessionModel(t, rt, t.TempDir(), "economical")
	id := s.Identity()

	// AC1:实际 CLI 版本来自现场探针,与适配器版本是两个可区分的事实。
	if id.RuntimeVersion != "0.153.0-conformance" {
		t.Fatalf("ObservedIdentity.RuntimeVersion = %q, want 现场探针值 0.153.0-conformance", id.RuntimeVersion)
	}
	if id.RuntimeVersion == DescriptorVersion || id.RuntimeVersion == rt.Descriptor().Version {
		t.Fatalf("CLI 版本被冒充为适配器版本 %q(Decision Lock 15)", DescriptorVersion)
	}
	if id.RuntimeName != cliSelfName {
		t.Fatalf("ObservedIdentity.RuntimeName = %q, want CLI 自报 %q", id.RuntimeName, cliSelfName)
	}
	// exec 表面不自报模型(探针记录 §3):未观察,不得回填请求档位。
	if id.Model != "" {
		t.Fatalf("ObservedIdentity.Model = %q, want 空(未观察 ≠ 请求值 economical)", id.Model)
	}
	// 新会话首回合前 thread id 尚未观察到。
	if id.SessionID != "" {
		t.Fatalf("新会话首回合前 SessionID 应未观察,得到 %q", id.SessionID)
	}
	if !id.Observed() {
		t.Fatalf("已有 CLI 身份观察,ObservedIdentity.Observed 应为真: %+v", id)
	}

	// 同一 Runtime 上的 Snapshot 仍区分两个版本事实(P1-37 契约不回归)。
	snap, err := rt.Snapshot(ctx)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if snap.AdapterVersion != DescriptorVersion || snap.CLIVersion != id.RuntimeVersion {
		t.Fatalf("AdapterVersion/CLIVersion 应区分: %+v vs observed %+v", snap, id)
	}
}

func TestIdentitySessionIDObservedAfterFirstTurn(t *testing.T) {
	rt := New(Config{BinPath: fakeCodexBin})
	ctx := context.Background()
	s := startSession(t, rt, t.TempDir())

	res, err := s.Execute(ctx, agentruntime.ExecuteRequest{Instruction: "first turn"})
	if err != nil || res.Status != agentruntime.RunCompleted {
		t.Fatalf("Execute: %+v err=%v", res, err)
	}
	id := s.Identity()
	if string(id.SessionID) != string(res.SessionID) || id.SessionID == "" {
		t.Fatalf("SessionID 观察 = %q, want thread.started 给出的 %q", id.SessionID, res.SessionID)
	}
}

func TestIdentityResumeReportsEstablishedThread(t *testing.T) {
	rt := New(Config{BinPath: fakeCodexBin})
	ctx := context.Background()
	ws := t.TempDir()

	s1 := startSession(t, rt, ws)
	r1, err := s1.Execute(ctx, agentruntime.ExecuteRequest{Instruction: "first"})
	if err != nil || r1.Status != agentruntime.RunCompleted {
		t.Fatalf("first turn: %+v err=%v", r1, err)
	}

	s2, err := rt.Start(ctx, agentruntime.StartRequest{Workspace: ws, ResumeOf: r1.SessionID})
	if err != nil {
		t.Fatalf("Start(ResumeOf): %v", err)
	}
	rep2, ok := s2.(agentruntime.IdentityReporter)
	if !ok {
		t.Fatal("resume 会话同样必须实现 IdentityReporter")
	}
	// resume 会话的身份是已建立的续用 thread 句柄(协议 SessionID = thread id)。
	if got := rep2.Identity().SessionID; got != r1.SessionID {
		t.Fatalf("resume 会话 SessionID = %q, want 已建立句柄 %q", got, r1.SessionID)
	}
}

func TestIdentityUnobservedWhenVersionProbeFails(t *testing.T) {
	// --version 探针失败:Start 不失败(会话开启不做进程健康门),身份事实
	// 如实未观察 —— 零值降级,绝不伪造(P0-01 能力安全语义)。
	rt := New(Config{
		BinPath: fakeCodexBin,
		Env:     map[string]string{"TALLY_FAKE_CODEX_SCENARIO": "versionfail"},
	})
	s, err := rt.Start(context.Background(), agentruntime.StartRequest{Workspace: t.TempDir()})
	if err != nil {
		t.Fatalf("探针失败不应导致 Start 失败: %v", err)
	}
	rep, ok := s.(agentruntime.IdentityReporter)
	if !ok {
		t.Fatal("*Session 必须实现 IdentityReporter")
	}
	if id := rep.Identity(); id.Observed() {
		t.Fatalf("探针失败且未执行回合:身份应完全未观察,得到 %+v", id)
	}
	// Snapshot 仍显式报错(P1-37:版本是它的契约),不降级为零值成功。
	if _, err := rt.Snapshot(context.Background()); err == nil {
		t.Fatal("Snapshot 在 --version 失败时应显式报错")
	}
}

func TestIdentityConcurrentSafeReadOnly(t *testing.T) {
	// Identity() 契约:只读、非阻塞、可并发;不在调用时发起新观察。
	rt := New(Config{BinPath: fakeCodexBin})
	s := startSession(t, rt, t.TempDir())
	first := s.Identity()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 50; i++ {
			if got := s.Identity(); got != first {
				t.Errorf("无新观察时 Identity 应稳定: %+v vs %+v", got, first)
				return
			}
		}
	}()
	<-done
}
