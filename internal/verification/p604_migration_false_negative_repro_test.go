package verification

// p604_migration_false_negative_repro_test.go — P6-05(v1.4.1)迁移验证假阴性
// 根因修复的回归钉子(复现本体见 P6-04;本文件在修复后钉住**已修复**语义)。
//
// 历史事实(docs/evals/v1.4/results/raw/tally/db-0*.json、docs/tasks/v1.4/
// HANDOFF.md follow-up #1):5 个 database_migration 任务、14 次内部 verify
// 全部 FAIL(repair_exhausted 升级),而外部 judge 对同一最终候选 5/5 接受。
// P6-04 判别结论 = 触发器(组合层把冻结 gate key 原文当探针子命令 → 探针
// 文档化 infra 退出码 2)+ 放大器(executeVisibleBinding 把任何非零退出码
// 记为可信 EvidenceStatusFail,违反 AGENTS.md §5「verifier 错误不得伪造为
// 候选失败」)。
//
// P6-05 修复语义(本文件钉住):
//
//  1. 冻结协议通过 CommandSpec.InfraExitCodes 声明文档化 wiring/infra 退出码
//     (探针契约:exit 2);命中的执行 = 无 verdict 的执行错误(Err 非空),
//     结论 FinalNeedsReview,不落任何 evidence 事实(不伪造 verdict=fail)。
//  2. 负例守卫:未命中的退出码语义不变 —— 真实可信 FAIL(exit 1)仍必须是
//     verdict=fail + FinalFailed + 一条 fail 事实(修复不得削弱真负例)。
//  3. 结构守卫:0(pass)与 1(可信 FAIL)不可被声明为 infra —— Validate
//     拒绝,判定语义在类型层不可被削弱。
//
// 端到端回归(真实探针二进制 + 字节级正确 repair 候选)见
// test/e2e/v141_p604_migration_false_negative_repro_test.go。

import (
	"context"
	"strings"
	"testing"
	"time"

	"tally/internal/changeset"
	"tally/internal/core"
	"tally/internal/evidence"
	"tally/internal/policy"
	"tally/internal/runner"
)

// p604TrialShapeContract 返回 v1.4 trial 迁移绑定的最小同形契约:一条
// visible schema_check binding(pre_commit、all_pass、1 次 repetition、300s
// timeout、子命令 = 冻结 gate key 原文,即 P6-04 触发器的 argv 形状),并按
// P6-05 修复后的组合层形状声明探针文档化的 infra 退出码 {2}。replayExitCode
// 是 fakeRunner 逐字回放的探针观察(2 = unknown subcommand;1 = 可信 FAIL)。
func p604TrialShapeContract(t *testing.T) *Frozen {
	t.Helper()
	c := &Contract{
		TaskID:   "TASK-P6-04",
		Revision: 1,
		Acceptances: []Acceptance{
			{
				ID:                   "AC-01",
				Statement:            "fresh_database: the revision chain applies cleanly to an empty database",
				IndependenceRequired: IndependenceNone,
				EvidenceBindings: []Binding{{
					ID:               "fresh_database",
					AuthoringMode:    AuthoringPreCommit,
					VisibilityPolicy: VisibilityVisible,
					Basis:            &IndependenceBasis{Source: SourceVerificationAuthority},
					Evidence:         EvidenceRef{Type: EvidenceSchemaCheck},
					ExecutionProtocol: &ExecutionProtocol{
						Repetitions: 1,
						Aggregation: AggregationAllPass,
						Command: &CommandSpec{
							Argv:           []string{"v14tallyprobe.exe", "fresh_database", "-task", "db-01", "-repo", "C:/repo", "-dsn", "postgres://localhost:5433"},
							Timeout:        Duration(300 * time.Second),
							InfraExitCodes: []int{2},
						},
					},
				}},
			},
		},
	}
	f, err := FreezeContract(c, FreezePolicy{})
	if err != nil {
		t.Fatalf("FreezeContract: %v", err)
	}
	return f
}

// p604RunFixedSemantics 按修复后语义跑一次 trial 同形编排;candidate 文件 =
// 字节级正确的 repair 内容(v1.4 trial db-01 attempt 3 实况:候选与冻结
// reference repair 逐字节一致,仍曾被判 FAIL)。
func p604RunFixedSemantics(t *testing.T, fr *fakeRunner, sink *orchSink) *FinalOutcome {
	t.Helper()
	o, err := NewOrchestrator(OrchestratorConfig{
		Runner: fr,
		Policy: policy.DefaultExecutionPolicy(),
		Sink:   sink,
		Recorder: evidence.Recorder{
			Kind:  evidence.RecorderVerificationRun,
			Actor: "tally/p6-05-regression",
		},
	})
	if err != nil {
		t.Fatalf("NewOrchestrator: %v", err)
	}
	f := p604TrialShapeContract(t)
	cs, err := changeset.New(changeset.Spec{
		Project:      orchProject,
		BaseRevision: "base-p6-05",
		ProducerRun:  "run-p6-05",
		Files:        []changeset.ChangedFile{{Path: "migrations/0003.sql", Content: "ALTER TABLE orders ADD COLUMN total_cents BIGINT NOT NULL DEFAULT 0;\n"}},
	})
	if err != nil {
		t.Fatalf("changeset.New: %v", err)
	}
	out, err := o.Run(context.Background(), FinalRunRequest{
		Project:          core.ProjectID(orchProject),
		Contract:         f,
		ChangeSet:        cs,
		EnvironmentName:  "p6-05-regression",
		Environment:      orchEnv(),
		VersionCombo:     core.VersionCombo{"go": "p6-05"},
		Runtime:          "FakeRuntime",
		Workspace:        t.TempDir(),
		CandidateFixedAt: orchFixedPast,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return out
}

// TestP604InfraExitCode2YieldsNeedsReviewNotFabricatedFail — P6-04 假阴性的
// 放大器已修复:探针对 unknown subcommand 以文档化 infra 退出码 2 终止
// (fakeRunner 逐字回放该观察),且冻结协议声明了 {2} —— 编排器必须记为
// 无 verdict 的执行错误,结论 FinalNeedsReview,不落任何伪 fail 事实。
// 候选内容完全无关(P6-04 证据链第 4 条)。
func TestP604InfraExitCode2YieldsNeedsReviewNotFabricatedFail(t *testing.T) {
	fr := &fakeRunner{results: []runner.Result{{
		Stderr:   "unknown subcommand \"fresh_database\"\n\n",
		ExitCode: 2,
		Duration: 5 * time.Millisecond,
	}}}
	sink := &orchSink{}
	out := p604RunFixedSemantics(t, fr, sink)

	if len(out.Bindings) != 1 {
		t.Fatalf("bindings = %d, 期望 1", len(out.Bindings))
	}
	b := out.Bindings[0]
	if b.Err == "" || !strings.Contains(b.Err, "infra") {
		t.Fatalf("infra 退出码 2 必须产生无 verdict 的执行错误(Err 含 infra 说明),当前 %q", b.Err)
	}
	if b.ExitCode != 2 {
		t.Fatalf("binding 退出码 = %d, 期望 2(探针 unknown subcommand)", b.ExitCode)
	}
	if out.Status != FinalNeedsReview {
		t.Fatalf("最终结论 = %s, 修复后语义期望 needs_review(infra 错误不得伪造为候选失败)", out.Status)
	}
	if len(sink.facts) != 0 {
		t.Fatalf("落库 facts = %+v, 期望 0 条(无 verdict 的执行不产生事实)", sink.facts)
	}
	if len(out.FactIDs) != 0 {
		t.Fatalf("FactIDs = %v, 期望空", out.FactIDs)
	}
}

// TestP604TrustedExitCode1IsStillAFail — 负例守卫(不得削弱真负例):同一
// 冻结协议(声明 infra {2})下,真实可信 FAIL(exit 1)仍必须是 verdict=fail
// + FinalFailed + 恰一条 fail 事实。修复只通道化「文档化 infra 退出码」,
// 不抹掉任何候选失败。
func TestP604TrustedExitCode1IsStillAFail(t *testing.T) {
	fr := &fakeRunner{results: []runner.Result{{
		Stdout:   "probe fresh db-01: FAIL — seeded data was altered\n",
		ExitCode: 1,
		Duration: 5 * time.Millisecond,
	}}}
	sink := &orchSink{}
	out := p604RunFixedSemantics(t, fr, sink)

	if len(out.Bindings) != 1 {
		t.Fatalf("bindings = %d, 期望 1", len(out.Bindings))
	}
	b := out.Bindings[0]
	if b.Err != "" {
		t.Fatalf("可信 FAIL(exit 1)不应产生执行错误: %q", b.Err)
	}
	if b.Status != EvidenceStatusFail || b.ExitCode != 1 {
		t.Fatalf("binding = (%s, exit %d), 期望 (fail, 1)(真实验证失败仍是失败)", b.Status, b.ExitCode)
	}
	if out.Status != FinalFailed {
		t.Fatalf("最终结论 = %s, 期望 failed(确定性验证失败语义保持)", out.Status)
	}
	if len(sink.facts) != 1 || sink.facts[0].fact.Verdict != EvidenceStatusFail {
		t.Fatalf("落库 facts = %+v, 期望恰一条 verdict=fail 的事实", sink.facts)
	}
}

// TestP604InfraExitCodesCannotRedefinedPassOrFail — 结构守卫:0(pass)与
// 1(canonical trusted FAIL)不可被声明为 infra;Validate 必须拒绝,否则
// 判定语义可在冻结协议里被静默削弱。
func TestP604InfraExitCodesCannotRedefinedPassOrFail(t *testing.T) {
	for _, code := range []int{0, 1, -1} {
		s := &CommandSpec{Argv: []string{"probe"}, InfraExitCodes: []int{code}}
		if err := s.Validate(); err == nil {
			t.Fatalf("infra_exit_codes 声明 %d 必须被 Validate 拒绝", code)
		}
	}
	s := &CommandSpec{Argv: []string{"probe"}, InfraExitCodes: []int{2, 2}}
	if err := s.Validate(); err == nil {
		t.Fatal("infra_exit_codes 重复声明必须被 Validate 拒绝")
	}
	s = &CommandSpec{Argv: []string{"probe"}, InfraExitCodes: []int{2, 3}}
	if err := s.Validate(); err != nil {
		t.Fatalf("合法 infra_exit_codes {2,3} 不应被拒绝: %v", err)
	}
}
