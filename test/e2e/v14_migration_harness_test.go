//go:build integration

// P3-04 harness:Database Migration 闭环 E2E 的迁移执行面(PRD §24.2)。
// P3-02 的单元测试把夹具 SQL 的真实执行明确留给「harness 运行时任务」;本文件
// 就是该执行面 —— 以真实 PostgreSQL(per-test 一次性 database)驱动
// test/fixtures/v14/migration/** 的迁移链,并按
// catalog/v14/policies/database_migration.yaml 的门禁键
// (fresh_database / n_minus_1_upgrade / data_integrity)裁决。
//
// 组成(与 v14_closedloop_harness_test.go 的 P2-10 模式同型,按 P2-09 的
// cmd/worker/v14.go 生产组成逐件复刻;全部域件都是生产组件):
//
//	迁移探针(test/e2e/v14_migration_probe,冻结 binding 的 argv 命令)
//	  → 真实执行夹具 SQL:fresh 路径 / N-1 seed 升级路径 / 数据完整性断言
//	→ artifacts.Load(Git HEAD 快照装载)+ artifacts.LoadPolicies(策略束装载)
//	→ risk.Derive(夹具 basis)→ contract.Compile + contract.Bind(P1-07)
//	→ contextpack.Select(含 policy requires 的 Prevention 上下文)
//	→ workflow.NewRunStoreSink / closedloop 事实出口(真实 PostgreSQL append-only)
//	→ candidateverify.New(生产候选验证服务:一次性验证 workspace + 冻结协议编排
//	  + verification_run/evidence 落库;binding 命令 = 迁移探针)
//	→ closedloop.NewLoop(P2-08 生产组合根)
//
// 数据库不可达时 SKIP(不产出假失败)。
package e2e

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"tally/internal/agentruntime"
	"tally/internal/artifacts"
	"tally/internal/candidateverify"
	"tally/internal/closedloop"
	"tally/internal/contextpack"
	"tally/internal/contract"
	"tally/internal/core"
	"tally/internal/engineeringpolicy"
	"tally/internal/evidence"
	"tally/internal/policy"
	"tally/internal/risk"
	"tally/internal/runner"
	"tally/internal/runstore"
	"tally/internal/store"
	"tally/internal/taskcontract"
	"tally/internal/verification"
	"tally/internal/workflow"
	"tally/internal/workspace"
	"tally/runtimes/fake"
)

// ---- 常量与夹具标识 ----------------------------------------------------------

const (
	migProject  = core.ProjectID("proj-p304")
	migTaskID   = "TASK-304"
	migAccID    = "AC-304"
	migTaskPath = "migrations/0003_orders_total_not_null.sql"

	// migPolicyRel / migFixtureRel 定位被驱动的真实权威文件(只读):
	// 内置迁移策略束与 P3-02 迁移夹具。
	migPolicyRel  = "catalog/v14/policies/database_migration.yaml"
	migFixtureRel = "test/fixtures/v14/migration"
)

// migRepoRoot 定位被测仓库根(测试运行目录 = test/e2e)。
func migRepoRoot(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("定位测试目录: %v", err)
	}
	root := filepath.Clean(filepath.Join(cwd, "..", ".."))
	if fi, err := os.Stat(filepath.Join(root, migPolicyRel)); err != nil || fi.IsDir() {
		t.Fatalf("仓库根 %s 下找不到 %s: %v", root, migPolicyRel, err)
	}
	return root
}

// migFixturePath 返回 P3-02 迁移夹具的绝对路径。
func migFixturePath(t *testing.T, rel ...string) string {
	t.Helper()
	return filepath.Join(append([]string{migRepoRoot(t), migFixtureRel}, rel...)...)
}

// migReadFixture 读一份 P3-02 夹具字节(策略束 / 迁移链 / seed / 证据集)。
func migReadFixture(t *testing.T, rel ...string) string {
	t.Helper()
	data, err := os.ReadFile(migFixturePath(t, rel...))
	if err != nil {
		t.Fatalf("读夹具 %v: %v", rel, err)
	}
	return string(data)
}

// migReadRepoFile 读一份仓库根相对路径的权威文件(如内置策略束)。
func migReadRepoFile(t *testing.T, rel ...string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(append([]string{migRepoRoot(t)}, rel...)...))
	if err != nil {
		t.Fatalf("读仓库文件 %v: %v", rel, err)
	}
	return string(data)
}

// ---- PostgreSQL 探针 DSN 与一次性 database -----------------------------------

// migBaseDSN 返回宿主 PostgreSQL 的管理 DSN(与 clStore 同一缺省面)。
func migBaseDSN() string {
	if base := os.Getenv("DATABASE_URL"); base != "" {
		return base
	}
	return clDSN
}

// migDSNFor 把管理 DSN 的 database 替换为 name。
func migDSNFor(t *testing.T, base, name string) string {
	t.Helper()
	u, err := url.Parse(base)
	if err != nil {
		t.Fatalf("解析 DSN: %v", err)
	}
	u.Path = "/" + name
	return u.String()
}

// migCreateDB 建一个一次性 database(P3-04 的迁移沙箱,与 tally schema 完全
// 隔离),返回其 DSN;t.Cleanup 里强制删除。
func migCreateDB(t *testing.T, base, tag string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, base)
	if err != nil {
		t.Fatalf("连管理库: %v", err)
	}
	name := fmt.Sprintf("migp304_%s_%d", tag, time.Now().UnixNano())
	if _, err := conn.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
		_ = conn.Close(ctx)
		t.Fatalf("建一次性 database %s: %v", name, err)
	}
	if err := conn.Close(ctx); err != nil {
		t.Fatalf("关闭管理连接: %v", err)
	}
	t.Cleanup(func() {
		dctx, dcancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer dcancel()
		dconn, err := pgx.Connect(dctx, base)
		if err != nil {
			return
		}
		defer dconn.Close(dctx)
		// 先断开残留连接,FORCE 删除兜底(PostgreSQL 13+)。
		_, _ = dconn.Exec(dctx,
			"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname=$1 AND pid <> pg_backend_pid()", name)
		_, _ = dconn.Exec(dctx, "DROP DATABASE IF EXISTS "+pgx.Identifier{name}.Sanitize()+" WITH (FORCE)")
	})
	return migDSNFor(t, base, name)
}

// ---- 迁移探针(frozen binding 的 argv 命令)-----------------------------------

// migBuildProbe 编译迁移探针(结构化 argv 编译命令;模块上下文 = 仓库根),
// 返回探针绝对路径。go build 走缓存,逐测试编译成本可忽略。
func migBuildProbe(t *testing.T) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "v14migprobe.exe")
	res, err := (runner.LocalRunner{}).Run(context.Background(), runner.Command{
		Argv: []string{
			"go", "build", "-tags=integration", "-o", out,
			"./test/e2e/v14_migration_probe",
		},
		Dir: migRepoRoot(t),
	})
	if err != nil || res.ExitCode != 0 {
		t.Fatalf("编译迁移探针: err=%v exit=%d stderr=%s", err, res.ExitCode, res.Stderr)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("探针未产出: %v", err)
	}
	return out
}

// migProbeResult 是一次探针执行的观察(确定性判定面)。
type migProbeResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// migRunProbe 以结构化 argv 执行一次探针子命令(与冻结 binding 命令同形;
// 超时由 context 界定,不依赖外部 shell)。
func migRunProbe(t *testing.T, probe, sub, dsn, migrationsDir, seed string) migProbeResult {
	t.Helper()
	argv := []string{probe, sub, "-dsn", dsn, "-dir", migrationsDir, "-timeout", "120s"}
	if seed != "" {
		argv = append(argv, "-seed", seed)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	res, err := (runner.LocalRunner{}).Run(ctx, runner.Command{
		Argv: argv,
		Dir:  migRepoRoot(t),
	})
	if err != nil {
		t.Fatalf("探针 %s 执行失败: %v", sub, err)
	}
	return migProbeResult{ExitCode: res.ExitCode, Stdout: res.Stdout, Stderr: res.Stderr}
}

// migRepairChain 把修复候选(新 ChangeSet 内容)与夹具基础修订并排组合成一个
// 迁移链目录 —— 决策锁 8/23:缺陷文件不被改写,修复以新内容进入同一修订号位。
func migRepairChain(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "migrations")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("建修复链目录: %v", err)
	}
	files := map[string]string{
		"0001_init.sql":                  migReadFixture(t, "migrations", "0001_init.sql"),
		"0002_add_orders.sql":            migReadFixture(t, "migrations", "0002_add_orders.sql"),
		"0003_orders_total_not_null.sql": migReadFixture(t, "repair", "0003_orders_total_not_null.sql"),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("写修复链 %s: %v", name, err)
		}
	}
	return dir
}

// migFulfillments 编译一次真实执行的门禁 fulfillments(bundle requires 键 →
// 是否被真实迁移执行满足)。
func migFulfillments(fresh, n1, integrity bool) []engineeringpolicy.RequirementFulfillment {
	return []engineeringpolicy.RequirementFulfillment{
		{Key: "data_integrity", Satisfied: integrity},
		{Key: "fresh_database", Satisfied: fresh},
		{Key: "n_minus_1_upgrade", Satisfied: n1},
	}
}

// ---- 真实 SQL 执行腿:夹具迁移链的 fresh / N-1 / 修复三路径 -------------------

// TestV14MigrationE2EFixtureSQLRealExecution 真实执行 P3-02 夹具迁移链:
//
//	AC1 fresh 路径:缺陷候选在空库上"通过"——PRD §24.2 的假置信绿;
//	N-1 路径:seed 后升级,PostgreSQL 确定性失败(SQLSTATE 23502,
//	'column "total_cents" of relation "orders" contains null values');
//	AC2/AC3 修复候选(NOT NULL DEFAULT 0)让 N-1 升级通过,seed 行的身份/
//	链路/时间戳逐字保留,total_cents 全量回填 0;
//	门禁(bundle gate_condition.all_of):真实执行的 fulfillments 对缺陷谱系
//	不放行(fresh 单独是假置信)、对修复谱系放行,且与 P3-02 证据夹具
//	(evidence/pre_repair、evidence/post_repair)逐键一致。
func TestV14MigrationE2EFixtureSQLRealExecution(t *testing.T) {
	base := migBaseDSN()
	if !e2eDBReachable(base) {
		t.Skipf("P3-04 迁移 E2E 需要 PostgreSQL(%s 不可达),跳过", base)
	}
	probe := migBuildProbe(t)
	defectDir := migFixturePath(t, "migrations")
	repairDir := migRepairChain(t)
	seed := migFixturePath(t, "seed", "n_minus_1.sql")

	// ---- 缺陷候选:fresh 绿(假置信),N-1 与完整性确定性失败 ----------------
	freshDefect := migRunProbe(t, probe, "fresh", migCreateDB(t, base, "fresh_defect"), defectDir, "")
	if freshDefect.ExitCode != 0 {
		t.Fatalf("缺陷候选 fresh 路径 exit=%d stderr=%s,期望通过(假置信绿必须可复现)", freshDefect.ExitCode, freshDefect.Stderr)
	}
	n1Defect := migRunProbe(t, probe, "n1", migCreateDB(t, base, "n1_defect"), defectDir, seed)
	if n1Defect.ExitCode != 1 {
		t.Fatalf("缺陷候选 N-1 路径 exit=%d stdout=%s,期望确定性失败", n1Defect.ExitCode, n1Defect.Stdout)
	}
	// 确定性迁移失败签名(P3-02 夹具注释钉死的 PostgreSQL 判定)。
	for _, want := range []string{"23502", `column "total_cents" of relation "orders" contains null values`} {
		if !strings.Contains(n1Defect.Stderr, want) {
			t.Fatalf("N-1 失败签名缺 %q: stderr=%s", want, n1Defect.Stderr)
		}
	}
	integrityDefect := migRunProbe(t, probe, "integrity", migCreateDB(t, base, "integrity_defect"), defectDir, seed)
	if integrityDefect.ExitCode != 1 {
		t.Fatalf("缺陷候选完整性路径 exit=%d stdout=%s,期望失败(升级失败使断言不可达)", integrityDefect.ExitCode, integrityDefect.Stdout)
	}

	// ---- 修复候选:N-1 升级通过,数据完整性逐字成立 -------------------------
	freshRepair := migRunProbe(t, probe, "fresh", migCreateDB(t, base, "fresh_repair"), repairDir, "")
	if freshRepair.ExitCode != 0 {
		t.Fatalf("修复候选 fresh 路径 exit=%d stderr=%s,期望通过", freshRepair.ExitCode, freshRepair.Stderr)
	}
	n1Repair := migRunProbe(t, probe, "n1", migCreateDB(t, base, "n1_repair"), repairDir, seed)
	if n1Repair.ExitCode != 0 {
		t.Fatalf("修复候选 N-1 路径 exit=%d stderr=%s,期望通过", n1Repair.ExitCode, n1Repair.Stderr)
	}
	integrityRepair := migRunProbe(t, probe, "integrity", migCreateDB(t, base, "integrity_repair"), repairDir, seed)
	if integrityRepair.ExitCode != 0 {
		t.Fatalf("修复候选数据完整性 exit=%d stderr=%s,期望通过(seed 行保留 + total_cents 回填)", integrityRepair.ExitCode, integrityRepair.Stderr)
	}

	// ---- 门禁(bundle gate_condition.all_of)对真实执行结果求值 --------------
	bundle, err := engineeringpolicy.Parse([]byte(migReadRepoFile(t, migPolicyRel)))
	if err != nil {
		t.Fatalf("内置迁移策略束必须通过解析: %v", err)
	}
	pre, err := bundle.EvaluateGate(migFulfillments(
		freshDefect.ExitCode == 0, n1Defect.ExitCode == 0, integrityDefect.ExitCode == 0))
	if err != nil {
		t.Fatalf("EvaluateGate(缺陷谱系): %v", err)
	}
	if pre.Satisfied {
		t.Fatal("缺陷候选(fresh 单独绿)门禁不得放行:PRD §24.2 fresh-only 假置信")
	}
	if want := []string{"data_integrity", "n_minus_1_upgrade"}; strings.Join(pre.UnmetKeys, ",") != strings.Join(want, ",") {
		t.Fatalf("缺陷谱系未满足键 = %v,期望 %v", pre.UnmetKeys, want)
	}
	post, err := bundle.EvaluateGate(migFulfillments(
		freshRepair.ExitCode == 0, n1Repair.ExitCode == 0, integrityRepair.ExitCode == 0))
	if err != nil {
		t.Fatalf("EvaluateGate(修复谱系): %v", err)
	}
	if !post.Satisfied || len(post.UnmetKeys) != 0 {
		t.Fatalf("修复谱系三类证据齐备必须放行: satisfied=%v unmet=%v", post.Satisfied, post.UnmetKeys)
	}
	// 夹具证据集与真实执行结论逐键一致(P3-02 夹具契约的运行时确认)。
	migAssertEvidenceFixture(t, "pre_repair", map[string]bool{
		"fresh_database": true, "n_minus_1_upgrade": false, "data_integrity": false,
	})
	migAssertEvidenceFixture(t, "post_repair", map[string]bool{
		"fresh_database": true, "n_minus_1_upgrade": true, "data_integrity": true,
	})
}

// migAssertEvidenceFixture 读 evidence/<set>/<key>.txt(satisfied|unsatisfied)
// 并与真实执行结论对照。
func migAssertEvidenceFixture(t *testing.T, set string, want map[string]bool) {
	t.Helper()
	for key, satisfied := range want {
		fact := strings.TrimSpace(migReadFixture(t, "evidence", set, key+".txt"))
		got := fact == "satisfied"
		if fact != "satisfied" && fact != "unsatisfied" {
			t.Fatalf("证据夹具 %s/%s 内容 %q 非法", set, key, fact)
		}
		if got != satisfied {
			t.Fatalf("证据夹具 %s/%s = %s,与真实执行结论 %v 不一致", set, key, fact, satisfied)
		}
	}
}

// ---- 闭环 E2E 的脚本化 FakeRuntime(候选 = 迁移文件内容)----------------------

// migStep 是一次 Execute 轮次的脚本:files 非空则在会话的一次性 worktree 内
// 照写候选文件(repo 相对路径 → 内容);err 非空则模拟 provider 失败;inspect
// 在写入前观察 workspace 与指令(修复基物化断言必须发生在当下 —— worktree 随
// 尝试结束销毁;断言用闭包捕获的 *testing.T)。
type migStep struct {
	files   map[string]string
	err     error
	inspect func(ws, instruction string)
}

// migScriptedRuntime 包装真 FakeRuntime:每个 Start 产出一个会话,会话的每次
// Execute 弹出一条脚本,再委托真 FakeRuntime 产出 claim。runtime 归因与能力
// 协商完全来自 FakeRuntime Descriptor(fake),控制流不按 runtime 名字分支。
type migScriptedRuntime struct {
	inner agentruntime.Runtime

	mu    sync.Mutex
	steps []migStep
	instr []string
}

func newMigScriptedRuntime(steps ...migStep) *migScriptedRuntime {
	return &migScriptedRuntime{inner: fake.New(fake.Config{}), steps: steps}
}

func (r *migScriptedRuntime) Descriptor() agentruntime.RuntimeDescriptor { return r.inner.Descriptor() }

func (r *migScriptedRuntime) Start(ctx context.Context, req agentruntime.StartRequest) (agentruntime.Session, error) {
	s, err := r.inner.Start(ctx, req)
	if err != nil {
		return nil, err
	}
	return &migScriptedSession{rt: r, inner: s, ws: req.Workspace}, nil
}

// instructions 返回已观测到的迭代指令(按 Execute 顺序的拷贝)。
func (r *migScriptedRuntime) instructions() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.instr...)
}

type migScriptedSession struct {
	rt    *migScriptedRuntime
	inner agentruntime.Session
	ws    string
}

func (s *migScriptedSession) Execute(ctx context.Context, req agentruntime.ExecuteRequest) (agentruntime.AgentRunResult, error) {
	s.rt.mu.Lock()
	if len(s.rt.steps) == 0 {
		s.rt.mu.Unlock()
		return agentruntime.AgentRunResult{}, fmt.Errorf("migScriptedRuntime: 脚本耗尽")
	}
	step := s.rt.steps[0]
	s.rt.steps = s.rt.steps[1:]
	s.rt.instr = append(s.rt.instr, req.Instruction)
	s.rt.mu.Unlock()

	if step.inspect != nil {
		step.inspect(s.ws, req.Instruction)
	}
	if step.err != nil {
		return agentruntime.AgentRunResult{}, step.err
	}
	for rel, content := range step.files {
		p := filepath.Join(s.ws, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return agentruntime.AgentRunResult{}, err
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			return agentruntime.AgentRunResult{}, err
		}
	}
	return s.inner.Execute(ctx, req)
}

func (s *migScriptedSession) Cancel(ctx context.Context, id agentruntime.RunID) error {
	return s.inner.Cancel(ctx, id)
}

func (s *migScriptedSession) Events(ctx context.Context) (agentruntime.EventStream, error) {
	return s.inner.Events(ctx)
}

func (s *migScriptedSession) Result(ctx context.Context, id agentruntime.RunID) (agentruntime.AgentRunResult, error) {
	return s.inner.Result(ctx, id)
}

func (s *migScriptedSession) Close(ctx context.Context) error { return s.inner.Close(ctx) }

// ---- 配置 --------------------------------------------------------------------

// migE2EConfig 是迁移闭环的测试组合配置面(clConfig 同型):owner 仓库、一次性
// 执行/验证 workspace 根与外层修复预算(0 = 缺省 2,决策锁 23)。
type migE2EConfig struct {
	Repo               string
	WorktreeRoot       string // 一次性执行 worktree 根
	VerifyWorktreeRoot string // 一次性验证 workspace 根
	MaxRepair          int    // 0 = 缺省 2(决策锁 23)
}

// migConfigFor 为仓库装配独立的临时 workspace 根。
func migConfigFor(t *testing.T, repo string) migE2EConfig {
	t.Helper()
	return migE2EConfig{
		Repo:               repo,
		WorktreeRoot:       filepath.Join(t.TempDir(), "wt"),
		VerifyWorktreeRoot: filepath.Join(t.TempDir(), "verify-wt"),
	}
}

// ---- 模拟项目仓库(P3-02 夹具为迁移链基线)------------------------------------

const migProjectYAML = `project: proj-p304
entities:
  - id: migrations
    type: db_migration
    path: migrations
`

const migTaskYAML = `id: TASK-304
profile: standard
title: Database migration 0003 (orders.total_cents)
objective: >
  Ship revision 0003 adding orders.total_cents under migrations without losing
  data on upgraded databases.
scope:
  allowed_paths:
    - migrations
acceptance:
  - id: AC-304
    statement: >
      Revision 0003 adds orders.total_cents; acceptance follows policy
      database_migration evidence keys fresh_database, n_minus_1_upgrade and
      data_integrity (fresh DB alone is not acceptance).
    independence_required: none
`

// migVerificationYAML 是冻结 VerificationContract 模板:三条 visible binding 的
// id 恰为策略束 requires[]/gate_condition.all_of 的三类证据键,frozen 命令 =
// 迁移探针(真实 SQL 执行;per-binding 一次性 database DSN 注入 argv)。
const migVerificationYAML = `task_id: TASK-304
revision: 1
acceptances:
  - id: AC-304
    statement: >
      Revision 0003 adds orders.total_cents; acceptance follows policy
      database_migration evidence keys fresh_database, n_minus_1_upgrade and
      data_integrity.
    independence_required: none
    evidence_bindings:
      - id: fresh_database
        authoring_mode: pre_commit
        visibility_policy: visible
        evidence:
          type: schema_check
        execution_protocol:
          repetitions: 1
          aggregation: all_pass
          command:
            argv: [%q, fresh, -dsn, %q, -dir, migrations, -timeout, 120s]
            timeout: 180s
      - id: n_minus_1_upgrade
        authoring_mode: pre_commit
        visibility_policy: visible
        evidence:
          type: schema_check
        execution_protocol:
          repetitions: 1
          aggregation: all_pass
          command:
            argv: [%q, n1, -dsn, %q, -dir, migrations, -seed, %q, -timeout, 120s]
            timeout: 180s
      - id: data_integrity
        authoring_mode: pre_commit
        visibility_policy: visible
        evidence:
          type: schema_check
        execution_protocol:
          repetitions: 1
          aggregation: all_pass
          command:
            argv: [%q, integrity, -dsn, %q, -dir, migrations, -seed, %q, -timeout, 120s]
            timeout: 180s
`

// migNewRepo 建带 .tally 制品(含 FrozenIntent:db_migration 任务类型是
// P1-02 权威风险输入的确定性事实 ⇒ schema_migration 信号)、真实策略束副本
// 与迁移链基线修订 0001/0002 的真 git 仓库。修订 0003 不在基线里 —— 它是
// 被验候选(缺陷或修复)。
func migNewRepo(t *testing.T, probe string, freshDSN, n1DSN, integrityDSN, seed string) string {
	t.Helper()
	dir := t.TempDir()
	clGit(t, dir, "init", "-b", "main")
	clGit(t, dir, "config", "user.name", "tally-p304-e2e")
	clGit(t, dir, "config", "user.email", "tally-p304-e2e@example.invalid")
	clWriteFile(t, dir, ".tally/project.yaml", migProjectYAML)
	clWriteFile(t, dir, ".tally/intents/"+migTaskID+".yaml",
		"task_id: "+migTaskID+"\ntask_type: db_migration\nsource_ref: P3-04 migration harness fixture\n")
	clWriteFile(t, dir, ".tally/tasks/"+migTaskID+".yaml", migTaskYAML)
	clWriteFile(t, dir, ".tally/verifications/"+migTaskID+".yaml", fmt.Sprintf(
		migVerificationYAML,
		probe, freshDSN,
		probe, n1DSN, seed,
		probe, integrityDSN, seed,
	))
	// 内置迁移策略束:从真实 catalog 原样复制进生产装载目录(P1-03 Git 装载)。
	clWriteFile(t, dir, ".tally/policies/database_migration.yaml", migReadRepoFile(t, migPolicyRel))
	// 迁移链基线:夹具修订 0001/0002 逐字。
	clWriteFile(t, dir, "migrations/0001_init.sql", migReadFixture(t, "migrations", "0001_init.sql"))
	clWriteFile(t, dir, "migrations/0002_add_orders.sql", migReadFixture(t, "migrations", "0002_add_orders.sql"))
	clGit(t, dir, "add", ".")
	clGit(t, dir, "commit", "-m", "p3-04 migration fixtures (base chain 0001-0002)")
	return dir
}

// ---- 验证结论记录缝(观察 FinalOutcome;行为不改生产验证服务)-----------------

// migOutcomeRecorder 包装生产候选验证服务,记录每次 FinalOutcome(供闭环后的
// taskstate 投影消费),不改变任何验证行为。
type migOutcomeRecorder struct {
	inner closedloop.CandidateVerification

	mu       sync.Mutex
	outcomes []*verification.FinalOutcome
}

func (r *migOutcomeRecorder) Verify(ctx context.Context, req candidateverify.Request) (*verification.FinalOutcome, error) {
	out, err := r.inner.Verify(ctx, req)
	if err == nil && out != nil {
		r.mu.Lock()
		r.outcomes = append(r.outcomes, out)
		r.mu.Unlock()
	}
	return out, err
}

func (r *migOutcomeRecorder) recorded() []*verification.FinalOutcome {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]*verification.FinalOutcome(nil), r.outcomes...)
}

// ---- 闭环组合(clRunTask 同型;策略束/风险/合同编译为迁移面增量)---------------

// migObservation 是一次迁移闭环执行的可观测面。
type migObservation struct {
	Result   closedloop.Result
	Frozen   *verification.Frozen
	Task     taskcontract.TaskContract
	Compiled *contract.Compiled
	Bundle   engineeringpolicy.Policy // 装载的 database_migration 束(Git 生产装载)
	Instrs   []string
	Outcomes []*verification.FinalOutcome
}

// migRunTask 按 cmd/worker runV14Task 的生产组成执行迁移任务的 v1.4 闭环
// (P1-08 编译权威对齐):同源 bundle 装载(P1-01,DL-04:任务 + 冻结验证
// 契约 + FrozenIntent + 策略快照一次 revision 解析装齐)→ 编译权威(P1-03
// CompileAuthority = 执行的第一道门;权威风险输入由 FrozenIntent 的
// db_migration 任务事实确定性派生,P1-02)→ 一次性 worktree 根 →
// compiled-aware ContextPack(含 policy requires 的 Prevention 上下文)→
// 真实事实面(RunStoreSink / Lineage / Exposure)→ 生产候选验证服务
// (binding 命令 = 迁移探针)→ closedloop.NewLoop(LoopConfig.Authority 必达:
// P1-06 验收守卫在测试等价组合同样生效)→ Engine.Run。
func migRunTask(t *testing.T, ctx context.Context, queries *store.Queries, runtime agentruntime.Runtime, cfg migE2EConfig) migObservation {
	t.Helper()
	repo := cfg.Repo

	bundle, err := artifacts.LoadBundle(repo, "HEAD", migTaskID)
	if err != nil {
		t.Fatalf("装载 %s 的同源执行 bundle(HEAD): %v", repo, err)
	}
	if bundle.ProjectID == "" {
		t.Fatalf("提交 %s 缺少 .tally/project.yaml(项目身份必须由 Git 注册表声明)", bundle.SourceCommit)
	}
	task := bundle.Task.Contract
	if task.Profile != taskcontract.ProfileStandard {
		t.Fatalf("task %s profile=%q,非 standard", migTaskID, task.Profile)
	}
	frozen := bundle.Verification.Frozen

	// 策略快照(bundle 原样携带同一提交的裁决):恰为内置迁移束、零拒收。
	if len(bundle.Policies.Rejected) != 0 {
		t.Fatalf("内置迁移策略束不得被拒收: %+v", bundle.Policies.Rejected)
	}
	if len(bundle.Policies.Policies) != 1 || bundle.Policies.Policies[0].Policy.ID != "database_migration" {
		t.Fatalf("装载策略 = %+v,期望恰为 database_migration", bundle.Policies.Policies)
	}

	// 编译权威(P1-03,先于任何 runtime 会话与 workspace 创建):db_migration
	// 任务事实 ⇒ 确定性 schema_migration 风险信号(P1-02 DeriveRisk),
	// 命中迁移束(P1-03/04 匹配)。
	authority, err := bundle.CompileAuthority()
	if err != nil {
		t.Fatalf("编译 %s 的执行权威: %v", migTaskID, err)
	}
	compiled := authority.Compiled
	found := false
	for _, sig := range authority.Risk.Detected {
		if sig.ID == risk.SignalSchemaMigration {
			found = true
		}
	}
	if !found {
		t.Fatalf("权威风险输入 = %+v,期望 db_migration 任务事实派生 schema_migration 信号(P1-02)", authority.Risk.Detected)
	}
	// Contract-as-prevention 前提:命中迁移策略、门禁键恰为三类证据键。
	if len(compiled.Coverage().MatchedPolicies) != 1 ||
		compiled.Coverage().MatchedPolicies[0].Policy.ID != "database_migration" {
		t.Fatalf("schema_migration 风险事实必须命中迁移策略: %+v", compiled.Coverage().MatchedPolicies)
	}
	gotKeys := compiled.GateKeys()
	if len(gotKeys) != 3 || gotKeys[0] != "data_integrity" || gotKeys[1] != "fresh_database" || gotKeys[2] != "n_minus_1_upgrade" {
		t.Fatalf("编译门禁键 = %v,期望三类证据键", gotKeys)
	}

	mgr, err := workspace.New(repo, cfg.WorktreeRoot)
	if err != nil {
		t.Fatalf("workspace.New: %v", err)
	}
	worktrees := workspace.NewRunTable(mgr)

	packHash := migContextPack(t, repo, bundle.SourceCommit, task, frozen, compiled)

	runs := runstore.New(queries)
	wfSink := workflow.NewRunStoreSink(runs)
	recorder := &migOutcomeRecorder{}
	verify := closedloop.CandidateVerification(recorder)
	verifyImpl := candidateverify.New(candidateverify.Config{
		Repo:         repo,
		WorktreeRoot: cfg.VerifyWorktreeRoot,
		EnvName:      "p304-e2e",
		Recorder:     evidence.Recorder{Kind: evidence.RecorderVerificationRun, Actor: "tally-e2e/p3-04"},
	}, candidateverify.Deps{Queries: queries})
	recorder.inner = verifyImpl
	lineage, err := closedloop.NewStoreLineageSink(runs)
	if err != nil {
		t.Fatalf("NewStoreLineageSink: %v", err)
	}
	exposure, err := closedloop.NewStoreExposureSink(runs, frozen.Revision())
	if err != nil {
		t.Fatalf("NewStoreExposureSink: %v", err)
	}
	seeds, err := runs.ExposuresForTask(ctx, migProject, migTaskID)
	if err != nil {
		t.Fatalf("读暴露事实种子: %v", err)
	}

	codegen := &clCodegenRunner{
		project:  migProject,
		task:     task,
		policy:   policy.DefaultExecutionPolicy(),
		profile:  "default-coding",
		runtime:  runtime,
		model:    "test-model",
		behavior: "behavior-1",
		verifRev: frozen.Hash(),
		packHash: packHash,
		requires: compiled.Requires().Lines(),
		sink:     wfSink,
	}

	repair := closedloop.DefaultRepairPolicy()
	if cfg.MaxRepair != 0 {
		repair, err = closedloop.NewRepairPolicy(cfg.MaxRepair)
		if err != nil {
			t.Fatalf("修复预算非法: %v", err)
		}
	}
	loop, err := closedloop.NewLoop(closedloop.LoopConfig{
		Codegen:  codegen,
		Verify:   verify,
		Contract: frozen,
		// 验收守卫(P1-06/P1-08,DL-07/A-03):编译权威是 accept 权威的输入 ——
		// 迁移风险上调 L3 后,仅 visible 确定性证据(achieved L1)不再授权
		// 普通 Accept。
		Authority: authority.Compiled,
		Runtime:   runtime.Descriptor().Name,
		Worktrees: worktrees,
		BaseRevision: func(context.Context, string) (string, error) {
			return bundle.SourceCommit, nil
		},
		Repair:    repair,
		Lineage:   lineage,
		Exposure:  exposure,
		Exposures: seeds,
	})
	if err != nil {
		t.Fatalf("组装闭环: %v", err)
	}
	res, err := loop.Run(ctx, migTaskID)
	if err != nil {
		t.Fatalf("闭环执行: %v", err)
	}
	return migObservation{
		Result:   res,
		Frozen:   frozen,
		Task:     task,
		Compiled: compiled,
		Bundle:   bundle.Policies.Policies[0].Policy,
		Instrs:   runtime.(*migScriptedRuntime).instructions(),
		Outcomes: recorder.recorded(),
	}
}

// migContextPack 与 worker v13/v14 同规则:Git 树候选(migrations 域)经
// contextpack.Select;requires 上下文(策略束 Prevention 面)随包编译。
func migContextPack(t *testing.T, repo, base string, task taskcontract.TaskContract, frozen *verification.Frozen, compiled *contract.Compiled) string {
	t.Helper()
	res, err := (runner.LocalRunner{}).Run(context.Background(), runner.Command{
		Argv: []string{"git", "-C", repo, "ls-tree", "-r", "--name-only", "-z", base},
		Env:  e2eGitEnv(),
	})
	if err != nil || res.ExitCode != 0 {
		t.Fatalf("git ls-tree %s: err=%v res=%+v", base, err, res)
	}
	seen := map[string]bool{}
	var candidates []contextpack.Candidate
	for _, p := range strings.Split(res.Stdout, "\x00") {
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		if p == "migrations" || strings.HasPrefix(p, "migrations/") {
			candidates = append(candidates, contextpack.Candidate{Path: p})
		}
	}
	out, err := contextpack.Select(contextpack.SelectInput{
		Task:         task,
		Verification: frozen,
		Candidates:   candidates,
		Requires:     compiled.Requires(),
	})
	if err != nil {
		t.Fatalf("contextpack.Select: %v", err)
	}
	return out.Pack.Hash()
}
