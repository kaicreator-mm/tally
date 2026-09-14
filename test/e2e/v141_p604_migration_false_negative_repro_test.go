//go:build integration

// P6-05 — v1.4 迁移验证假阴性根因修复的端到端回归钉子(复现本体见 P6-04;
// 本文件在修复后钉住**已修复**语义)。
//
// 历史事实(docs/evals/v1.4/results/raw/tally/db-0*.json、docs/tasks/v1.4/
// HANDOFF.md follow-up #1):5 个 database_migration 任务、14 次内部 verify
// 全部 FAIL(repair_exhausted 升级),外部 judge 对同一最终候选 5/5 接受
// (db-01 最终候选与冻结 reference repair 字节一致)。
//
// P6-04 判别:触发器 = 组合层把冻结 gate key 原文当探针子命令(compose.go
// migration 分支)→ 探针只接受 fresh/n1/integrity → 文档化 infra 退出码 2;
// 放大器 = 生产编排器把任何非零退出码记为可信 FAIL。P6-05 修复双面:
// compose.go 映射 gate key → 探针子命令;CommandSpec.InfraExitCodes 声明
// 文档化 wiring 错误(探针契约 exit 2)→ 无 verdict、needs_review。
//
// 本测试在同一进程内钉住修复后的完整因果面:
//
//	A(触发器残留面,无 DB 依赖):gate key 原文 argv 仍被探针以文档化 infra
//	  退出码 2 + "unknown subcommand" 拒绝(探针子命令契约不变 —— 修复在
//	  组合层映射,不在探针放宽)。
//	B1(放大器已修,无 DB 依赖):生产编排器(LocalRunner)按 trial 形冻结
//	  协议(子命令 = gate key 原文 + infra_exit_codes {2})驱动**字节级正确
//	  的 repair 候选** → 三条 binding 全部无 verdict(Err 含 infra)、最终
//	  FinalNeedsReview、零落库事实 —— 不再伪造可信 FAIL。
//	B2(负例守卫,无 DB 依赖):同一探针、真实可信 FAIL 面(fresh 子命令 +
//	  不可达 DSN → 探针按自身契约 exit 1)→ verdict=fail + FinalFailed
//	  (真实验证失败仍是失败,修复不得削弱真负例)。
//	C(对照,需 PostgreSQL,不可达即 skip):同一候选、同一探针、同一 DSN
//	  面,子命令 = fresh/n1/integrity(即 P6-05 后组合层实际发出的形状)→
//	  三条全绿 —— 外部 judge 接受同一候选的等价证据面。
//
// 判别结论见 docs/tasks/v1.4.1/reports/P6-04_migration_false_negative_reproduction.md;
// 修复报告见 docs/tasks/v1.4.1/reports/P6-05_fix_migration_verification_root_cause.md。
package e2e

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"
	"time"

	"tally/internal/changeset"
	"tally/internal/core"
	"tally/internal/evidence"
	"tally/internal/policy"
	"tally/internal/runner"
	"tally/internal/verification"
)

// p604ProbeKeysToSubcommands 是探针接受的迁移子命令(冻结 gate key → 语义)。
// 与 scripts/v14_trial/tally/compose.go 的 migrationProbeSubcommands 同一映射
// (P6-05 修复面 1):本表钉住该映射的右侧值即探针子命令契约。
var p604ProbeKeysToSubcommands = map[string]string{
	"fresh_database":    "fresh",
	"n_minus_1_upgrade": "n1",
	"data_integrity":    "integrity",
}

// p604BuildTrialProbe 编译 v1.4 trial 的真实验证探针(组合层 argv 指向的
// 同一二进制),返回绝对路径。
func p604BuildTrialProbe(t *testing.T) string {
	t.Helper()
	out := p604TempFile(t, "v14tallyprobe")
	res, err := (runner.LocalRunner{}).Run(context.Background(), runner.Command{
		Argv: []string{"go", "build", "-o", out, "./scripts/v14_trial/tally/probe"},
		Dir:  migRepoRoot(t),
	})
	if err != nil || res.ExitCode != 0 {
		t.Fatalf("编译 trial 探针: err=%v exit=%d stderr=%s", err, res.ExitCode, res.Stderr)
	}
	return out
}

func p604TempFile(t *testing.T, name string) string {
	t.Helper()
	if goruntime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(t.TempDir(), name)
}

// p604MaintenanceDSN 把 DATABASE_URL 归一为 trial 组合层 maintenanceURI 的
// 语义(去 database 名与 query;探针 dbConn 按需重设 path)。
func p604MaintenanceDSN(t *testing.T, base string) string {
	t.Helper()
	u, err := url.Parse(base)
	if err != nil {
		t.Fatalf("解析 DSN: %v", err)
	}
	u.Path = ""
	u.RawQuery = ""
	return u.String()
}

// p604CandidateWorkspace 物化 v1.4 trial 验证 workspace 的候选面:冻结迁移链
// 布局 + 字节级正确的 repair 候选(0003 = repair 夹具字节 —— 实验 db-01
// attempt 3 的实况),返回 workspace 绝对路径。
func p604CandidateWorkspace(t *testing.T) string {
	t.Helper()
	ws := t.TempDir()
	rel := map[string]string{
		"test/fixtures/v14/migration/migrations/0001_init.sql":                  migReadFixture(t, "migrations", "0001_init.sql"),
		"test/fixtures/v14/migration/migrations/0002_add_orders.sql":            migReadFixture(t, "migrations", "0002_add_orders.sql"),
		"test/fixtures/v14/migration/migrations/0003_orders_total_not_null.sql": migReadFixture(t, "repair", "0003_orders_total_not_null.sql"),
		"test/fixtures/v14/migration/seed/n_minus_1.sql":                        migReadFixture(t, "seed", "n_minus_1.sql"),
	}
	for path, content := range rel {
		p := filepath.Join(ws, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("建目录 %s: %v", p, err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatalf("写候选文件 %s: %v", p, err)
		}
	}
	return ws
}

// TestP604MigrationFalseNegativeReproE2E — 修复后完整因果面(A + B1 + B2 + C)。
func TestP604MigrationFalseNegativeReproE2E(t *testing.T) {
	probe := p604BuildTrialProbe(t)
	ws := p604CandidateWorkspace(t)
	base := migBaseDSN()
	dsn := p604MaintenanceDSN(t, base)
	root := migRepoRoot(t)

	// 冻结 gate keys 以 db-01 冻结验收文档为准(组合层消费同一文档)。
	acc := migReadRepoFile(t, "test/eval/v14_product/testdata/tasks/db-01-orders-total-backfill/acceptance.yaml")
	for key := range p604ProbeKeysToSubcommands {
		if !strings.Contains(acc, key) {
			t.Fatalf("冻结验收文档缺少 gate key %s(组合层输入漂移)", key)
		}
	}

	// ---- A. 触发器残留面:gate key 原文 argv 仍 → 文档化 infra exit 2 ------
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	res, err := (runner.LocalRunner{}).Run(ctx, runner.Command{
		Argv: []string{probe, "fresh_database", "-task", "db-01-orders-total-backfill", "-repo", root, "-dsn", dsn},
		Dir:  ws,
	})
	if err != nil {
		t.Fatalf("执行探针: %v", err)
	}
	if res.ExitCode != 2 || !strings.Contains(res.Stderr, `unknown subcommand "fresh_database"`) {
		t.Fatalf("A 触发器面未保持: exit=%d stderr=%q(期望 exit 2 + unknown subcommand \"fresh_database\" —— 探针子命令契约不可放宽)", res.ExitCode, res.Stderr)
	}

	// ---- B1. 放大器已修:infra exit 2 → needs_review,不伪造可信 FAIL ------
	sink := &p604MemSink{}
	out := p604RunOrchestratorTrialShaped(t, probe, ws, root, dsn, func(key string) string { return key }, sink)
	if out.Status != verification.FinalNeedsReview {
		t.Fatalf("B1 修复语义未成立: 最终结论 = %s, 期望 needs_review(infra 错误不得伪造为候选失败)", out.Status)
	}
	if len(out.Bindings) != len(p604ProbeKeysToSubcommands) {
		t.Fatalf("bindings = %d, 期望 %d", len(out.Bindings), len(p604ProbeKeysToSubcommands))
	}
	for _, b := range out.Bindings {
		if b.Err == "" || !strings.Contains(b.Err, "infra") {
			t.Fatalf("binding %s: infra 退出码 2 必须产生无 verdict 执行错误,当前 Err=%q", b.BindingID, b.Err)
		}
		if b.ExitCode != 2 {
			t.Fatalf("binding %s exit = %d, 期望 2", b.BindingID, b.ExitCode)
		}
	}
	if len(sink.facts) != 0 || len(out.FactIDs) != 0 {
		t.Fatalf("B1 不应落任何事实: sink=%d FactIDs=%v(无 verdict 的执行不产生事实)", len(sink.facts), out.FactIDs)
	}

	// ---- B2. 负例守卫:真实可信 FAIL(exit 1)仍是 fail + FinalFailed -------
	// fresh 是合法子命令;DSN 指向必然拒绝连接的端口 → 探针按自身契约以
	// exit 1(可信 FAIL)终止 —— 与 B1 同一编排器、同一 infra 声明。
	badDSN := "postgres://postgres:postgres@127.0.0.1:1/postgres"
	sink2 := &p604MemSink{}
	out2 := p604RunOrchestratorTrialShaped(t, probe, ws, root, badDSN, func(string) string { return "fresh" }, sink2)
	if out2.Status != verification.FinalFailed {
		t.Fatalf("B2 负例守卫未成立: 最终结论 = %s, 期望 failed(exit 1 是可信 FAIL,不得被抹成 error)", out2.Status)
	}
	for _, b := range out2.Bindings {
		if b.Err != "" || b.Status != verification.EvidenceStatusFail || b.ExitCode != 1 {
			t.Fatalf("B2 binding %s = (status %s, exit %d, Err %q), 期望 (fail, 1, 无 Err)", b.BindingID, b.Status, b.ExitCode, b.Err)
		}
	}
	if len(sink2.facts) != len(p604ProbeKeysToSubcommands) {
		t.Fatalf("B2 落库 facts = %d, 期望 %d(每条 binding 一条 fail 事实)", len(sink2.facts), len(p604ProbeKeysToSubcommands))
	}

	// ---- C. 对照:同一候选、同一探针、子命令 = 组合层修复后形状 → 全绿 ----
	if !e2eDBReachable(base) {
		t.Skipf("C 对照面需要 PostgreSQL(%s 不可达),A+B1+B2 已确定性钉住,跳过对照", base)
	}
	for key, sub := range p604ProbeKeysToSubcommands {
		cctx, ccancel := context.WithTimeout(context.Background(), 4*time.Minute)
		cres, cerr := (runner.LocalRunner{}).Run(cctx, runner.Command{
			Argv: []string{probe, sub, "-task", "db-01-orders-total-backfill", "-repo", root, "-dsn", dsn},
			Dir:  ws,
		})
		ccancel()
		if cerr != nil || cres.ExitCode != 0 {
			t.Fatalf("C 对照 %s(%s)未通过: err=%v exit=%d stderr=%q stdout=%q —— 候选本应可验",
				key, sub, cerr, cres.ExitCode, cres.Stderr, cres.Stdout)
		}
	}
}

// p604RunOrchestratorTrialShaped 用生产编排器(LocalRunner,真实子进程)按
// trial 形冻结协议(三条 visible schema_check binding,300s timeout,声明
// infra_exit_codes {2} —— P6-05 修复后组合层发出的协议形状)验证一个字节级
// 正确的候选 ChangeSet。subcommandFor 决定 argv 子命令:返回 key 本身 =
// 历史 trial 触发器形状(P6-04 A/B 面);返回映射值 = 修复后组合层形状。
func p604RunOrchestratorTrialShaped(t *testing.T, probe, ws, repoRoot, dsn string, subcommandFor func(key string) string, sink *p604MemSink) *verification.FinalOutcome {
	t.Helper()
	c := &verification.Contract{
		TaskID:   "TASK-P6-04",
		Revision: 1,
	}
	for _, key := range []string{"fresh_database", "n_minus_1_upgrade", "data_integrity"} {
		c.Acceptances = append(c.Acceptances, verification.Acceptance{
			ID:                   "AC-" + key,
			Statement:            key + ": frozen trial acceptance clause",
			IndependenceRequired: verification.IndependenceNone,
			EvidenceBindings: []verification.Binding{{
				ID:               key,
				AuthoringMode:    verification.AuthoringPreCommit,
				VisibilityPolicy: verification.VisibilityVisible,
				Basis:            &verification.IndependenceBasis{Source: verification.SourceVerificationAuthority},
				Evidence:         verification.EvidenceRef{Type: verification.EvidenceSchemaCheck},
				ExecutionProtocol: &verification.ExecutionProtocol{
					Repetitions: 1,
					Aggregation: verification.AggregationAllPass,
					Command: &verification.CommandSpec{
						Argv:           []string{probe, subcommandFor(key), "-task", "db-01-orders-total-backfill", "-repo", repoRoot, "-dsn", dsn},
						Timeout:        verification.Duration(300 * time.Second),
						InfraExitCodes: []int{2},
					},
				},
			}},
		})
	}
	frozen, err := verification.FreezeContract(c, verification.FreezePolicy{})
	if err != nil {
		t.Fatalf("FreezeContract: %v", err)
	}
	cs, err := changeset.New(changeset.Spec{
		Project:      core.ProjectID("proj-p6-04"),
		BaseRevision: "base-p6-04",
		ProducerRun:  "run-p6-04",
		Files: []changeset.ChangedFile{{
			Path:    "test/fixtures/v14/migration/migrations/0003_orders_total_not_null.sql",
			Content: migReadFixture(t, "repair", "0003_orders_total_not_null.sql"),
		}},
	})
	if err != nil {
		t.Fatalf("changeset.New: %v", err)
	}
	orch, err := verification.NewOrchestrator(verification.OrchestratorConfig{
		Runner: runner.LocalRunner{},
		Policy: policy.DefaultExecutionPolicy(),
		Sink:   sink,
		Recorder: evidence.Recorder{
			Kind:  evidence.RecorderVerificationRun,
			Actor: "tally-e2e/p6-05",
		},
	})
	if err != nil {
		t.Fatalf("NewOrchestrator: %v", err)
	}
	out, err := orch.Run(context.Background(), verification.FinalRunRequest{
		Project:         core.ProjectID("proj-p6-04"),
		Contract:        frozen,
		ChangeSet:       cs,
		EnvironmentName: "p6-05-regression",
		Environment: verification.EnvironmentIdentity{
			OS:        "linux",
			Arch:      "amd64",
			Toolchain: "go-p6-05",
		},
		VersionCombo:     core.VersionCombo{"go": "p6-05"},
		Runtime:          "P6-05Regression",
		Workspace:        ws,
		CandidateFixedAt: time.Now().UTC().Add(-time.Minute),
	})
	if err != nil {
		t.Fatalf("orchestrator.Run: %v", err)
	}
	return out
}

// p604MemSink 是 RunSink 的内存假实现(与 orchestrator 单测同面)。
type p604MemSink struct {
	nextID int64
	facts  []verification.FactSpec
}

func (s *p604MemSink) RecordRun(ctx context.Context, spec verification.RunSpec) (int64, error) {
	s.nextID++
	return s.nextID, nil
}

func (s *p604MemSink) RecordFact(ctx context.Context, project core.ProjectID, fact verification.FactSpec) error {
	s.facts = append(s.facts, fact)
	return nil
}

func (s *p604MemSink) ValidApplicableFacts(ctx context.Context, scope evidence.Scope) ([]verification.PriorFact, error) {
	return nil, nil
}
