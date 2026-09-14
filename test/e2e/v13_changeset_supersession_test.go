//go:build integration

// Package e2e 承载 P1 v1.3 门禁端到端测试(P1-50,INV-CHANGE-02)。
//
// 被测纵向切片跨真实边界组装:P1-39 单 Task 工作流内核驱动 FakeRuntime 与
// 一次性 Git worktree,经 RunStoreSink(P1-33 适配)把 Run 生命周期与候选
// ChangeSet 事实落进真实 PostgreSQL(每测试独立 schema、自跑 goose 迁移);
// 随后用 SupersededFact 表达取代、用 verification_run/evidence 绑定验证。
//
// 证明目标(任务验收,PRD §15.3 / INV-CHANGE-02):
//
//	AC1 没有固定 candidate 的 Run 不产生 ChangeSet;
//	AC2 新候选 A、B 是两个互不相同的不可变快照记录;
//	AC3 取代(A→B)不改动 A 本体 —— 快照间的唯一关系是 append-only 取代事实;
//	AC4 验证精确绑定在快照身份上,不跨快照继承。
//
// 数据库不可达时 SKIP(不产出假失败);数据库侧不可变性由 migrations/0019、
// 0020 的 append-only 语句级触发器兜底,这里直接断言其拒绝路径。
package e2e

import (
	"context"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"tally/internal/changeset"
	"tally/internal/core"
	"tally/internal/evidence"
	"tally/internal/policy"
	"tally/internal/runner"
	"tally/internal/runstore"
	"tally/internal/store"
	"tally/internal/taskcontract"
	"tally/internal/verification"
	"tally/internal/workflow"
	"tally/internal/workspace"
	"tally/runtimes/fake"
)

const e2eDSN = "postgres://postgres:postgres@localhost:5432/tally?sslmode=disable"

var (
	// e2eAt / e2eAt2 固定时钟:候选 A、B 与取代事实的时刻全部确定,
	// created_at 定序因此确定,断言不依赖真实时间。
	e2eAt  = time.Date(2026, 9, 6, 8, 0, 0, 0, time.UTC)
	e2eAt2 = e2eAt.Add(time.Hour)
)

const (
	e2eProject = core.ProjectID("proj-p1-50")
	e2eTask    = "TASK-P1-50"
	// e2eContractHash 是冻结 VerificationContract 内容哈希(e2e 常量,
	// verification_run/evidence 的 scope 维度)。
	e2eContractHash = "sha256:p1-50-frozen-contract"
)

// ---- 固件 ------------------------------------------------------------------

func e2eTaskContract() taskcontract.TaskContract {
	return taskcontract.TaskContract{
		ID:        e2eTask,
		Profile:   taskcontract.ProfileStandard,
		Title:     "P1-50 ChangeSet supersession gate",
		Objective: "Prove immutable ChangeSet and append-only supersession",
		Scope:     taskcontract.Scope{AllowedPaths: []string{"internal/widget"}},
		Acceptance: []taskcontract.AcceptanceClause{
			{ID: "AC-01", Statement: "widget list renders", IndependenceRequired: taskcontract.IndependenceNone},
		},
		Hash: "sha256:p1-50-task-revision",
	}
}

func e2ePolicy() policy.ExecutionPolicy {
	p := policy.DefaultExecutionPolicy()
	p.Retry.MaxIterations = 3
	p.Escalation.AfterFailedIterations = 2
	return p
}

// e2eStore 返回接好独立 schema 的 runstore 适配器与裸连接池(触发器拒绝
// 路径断言用)。数据库不可达时 SKIP。
func e2eStore(t *testing.T) (*runstore.Store, *pgxpool.Pool) {
	t.Helper()
	base := os.Getenv("DATABASE_URL")
	if base == "" {
		base = e2eDSN
	}
	if !e2eDBReachable(base) {
		t.Skipf("P1-50 端到端测试需要 PostgreSQL(%s 不可达),跳过", base)
	}
	st := store.TestDB(t)
	return runstore.New(st.Queries()), st.Pool()
}

func e2eDBReachable(base string) bool {
	u, err := url.Parse(base)
	if err != nil {
		return false
	}
	host := u.Hostname()
	if host == "" {
		host = "localhost"
	}
	port := u.Port()
	if port == "" {
		port = "5432"
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 2*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// ---- git / worktree 固件(结构化 argv,与 internal/workflow 测试同一策略) ----

func e2eGitEnv() []string {
	return []string{
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=" + os.DevNull,
		"GIT_CONFIG_SYSTEM=" + os.DevNull,
		"GIT_TERMINAL_PROMPT=0",
	}
}

func e2eGit(t *testing.T, dir string, argv ...string) string {
	t.Helper()
	res, err := (runner.LocalRunner{}).Run(context.Background(), runner.Command{
		Argv: append([]string{"git"}, argv...),
		Dir:  dir,
		Env:  e2eGitEnv(),
	})
	if err != nil {
		t.Fatalf("git %v: %v", argv, err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("git %v: exit %d: %s", argv, res.ExitCode, res.Stderr)
	}
	return strings.TrimSpace(res.Stdout)
}

func e2eNewRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	e2eGit(t, dir, "init", "-b", "main")
	e2eGit(t, dir, "config", "user.name", "tally-e2e")
	e2eGit(t, dir, "config", "user.email", "tally-e2e@example.invalid")
	return dir
}

func e2eCommit(t *testing.T, repo, name, content, msg string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repo, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	e2eGit(t, repo, "add", ".")
	e2eGit(t, repo, "commit", "-m", msg)
	return e2eGit(t, repo, "rev-parse", "HEAD")
}

// runCandidate 驱动一次完整的单 Task 工作流执行(真实 worktree + FakeRuntime +
// RunStoreSink 落库)。at 是本次执行的固定时钟;mark 同时决定候选内容与
// candidate revision:不同 mark 的执行产出内容身份不同的候选。
func runCandidate(t *testing.T, runs *runstore.Store, repo, base, mark string, at time.Time, noCandidate bool) workflow.Outcome {
	t.Helper()
	mgr, err := workspace.New(repo, filepath.Join(t.TempDir(), "wt"))
	if err != nil {
		t.Fatalf("workspace.New: %v", err)
	}
	fix := workflow.Hooks{FixCandidate: func(_ context.Context, _ workflow.IterationView) (workflow.CandidateSnapshot, bool, error) {
		if noCandidate {
			return workflow.CandidateSnapshot{}, false, nil
		}
		return workflow.CandidateSnapshot{
			CandidateRevision: "cand-" + mark,
			Files: []changeset.ChangedFile{{
				Path:    "internal/widget/widget.go",
				Content: "package widget // candidate " + mark + "\n",
			}},
			Entities: []string{"widget"},
		}, true, nil
	}}
	exec, err := workflow.NewTaskExecution(workflow.TaskExecutionSpec{
		Project:              e2eProject,
		Task:                 e2eTaskContract(),
		Role:                 "coding",
		Policy:               e2ePolicy(),
		BaseRevision:         base,
		Profile:              "default-coding",
		Runtime:              fake.New(fake.Config{}),
		Model:                "test-model",
		BehaviorRevision:     "behavior-1",
		VerificationRevision: e2eContractHash,
		ContextPackHash:      "sha256:p1-50-context-pack",
		Worktrees:            workspace.NewRunTable(mgr),
		Sink:                 workflow.NewRunStoreSink(runs),
		Hooks:                fix,
		Now:                  func() time.Time { return at },
	})
	if err != nil {
		t.Fatalf("NewTaskExecution: %v", err)
	}
	out, err := exec.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	return out
}

// runTwoCandidates 在同一 base 上顺序产出候选 A(mark v1)与返工候选 B(mark v2):
// 两次独立的工作流执行、两个不同的 producer Run,正是「重做 = 新快照」的纵向路径。
func runTwoCandidates(t *testing.T, runs *runstore.Store) (workflow.Outcome, workflow.Outcome, string) {
	t.Helper()
	repo := e2eNewRepo(t)
	base := e2eCommit(t, repo, "README.md", "base\n", "base")
	outA := runCandidate(t, runs, repo, base, "v1", e2eAt, false)
	outB := runCandidate(t, runs, repo, base, "v2", e2eAt2, false)
	return outA, outB, base
}

func countRows(t *testing.T, pool *pgxpool.Pool, query string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := pool.QueryRow(context.Background(), query, args...).Scan(&n); err != nil {
		t.Fatalf("count %q: %v", query, err)
	}
	return n
}

// mustModifyRejected 断言一条改写/删除语句被 append-only 触发器拒绝
// (migrations/0019、0020 的数据库侧防线)。
func mustModifyRejected(t *testing.T, pool *pgxpool.Pool, query string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), query, args...); err == nil {
		t.Fatalf("append-only 触发器未拦截: %s", query)
	}
}

// ---- AC1:没有 candidate 的 Run 不产生 ChangeSet ----------------------------

func TestV13RunWithoutCandidateCreatesNoChangeSet(t *testing.T) {
	runs, pool := e2eStore(t)
	repo := e2eNewRepo(t)
	base := e2eCommit(t, repo, "README.md", "base\n", "base")

	out := runCandidate(t, runs, repo, base, "v1", e2eAt, true)

	// 执行落点:候选固定点始终落空 → 阶梯耗尽 → Run failed,任务 NEEDS_REVIEW,
	// 结果值不带任何 ChangeSet(§11.2.1:没有固定 candidate 的 Run 不产生快照)。
	if out.Status != workflow.TaskNeedsReview {
		t.Fatalf("Status = %q, want needs_review", out.Status)
	}
	if out.ChangeSet != nil {
		t.Fatalf("无候选 Run 产生了 ChangeSet %s(AC1)", out.ChangeSet.ID())
	}
	if len(out.Runs) != 1 || out.Runs[0].Status != runstore.RunFailed {
		t.Fatalf("Runs = %+v, want 单条 failed Run", out.Runs)
	}
	if out.Runs[0].ChangeSetID != "" {
		t.Fatalf("RunRecord.ChangeSetID = %s, want 空(AC1)", out.Runs[0].ChangeSetID)
	}

	// 事实面:该 Task 的 Run 时间线存在且无产出候选,changeset 表零行(AC1)。
	timeline, err := runs.CodegenRunsForTask(context.Background(), e2eProject, e2eTask)
	if err != nil {
		t.Fatalf("CodegenRunsForTask: %v", err)
	}
	if len(timeline) != 1 || timeline[0].Status != runstore.RunFailed || timeline[0].ResultPatchHash != "" {
		t.Fatalf("落库 Run = %+v, want failed 且 result_patch_hash 为空(AC1)", timeline)
	}
	if n := countRows(t, pool, "SELECT count(*) FROM changeset WHERE project_id=$1", string(e2eProject)); n != 0 {
		t.Fatalf("无候选执行落了 %d 行 changeset,应为 0(AC1)", n)
	}
	if n := countRows(t, pool, "SELECT count(*) FROM changeset_superseded_fact"); n != 0 {
		t.Fatalf("无候选执行落了 %d 行取代事实,应为 0", n)
	}
}

// ---- AC2:候选 A 与 B 是互不相同的不可变记录 --------------------------------

func TestV13CandidatesAreDistinctImmutableRecords(t *testing.T) {
	runs, pool := e2eStore(t)
	outA, outB, _ := runTwoCandidates(t, runs)

	if outA.Status != workflow.TaskCompleted || outB.Status != workflow.TaskCompleted {
		t.Fatalf("Status = %s/%s, want completed/completed", outA.Status, outB.Status)
	}
	csA, csB := outA.ChangeSet, outB.ChangeSet
	if csA == nil || csB == nil {
		t.Fatalf("ChangeSet = %v/%v, want 两个固定候选", csA, csB)
	}
	// 新候选 = 新快照:身份不同、内容身份(patch)不同,绝不聚合进旧记录。
	if csA.ID() == csB.ID() {
		t.Fatalf("候选 A 与 B 身份相同 %s(AC2)", csA.ID())
	}
	if csA.PatchIdentity() == csB.PatchIdentity() {
		t.Fatalf("候选 A 与 B patch 身份相同 %s(AC2)", csA.PatchIdentity())
	}
	if csA.ProducerRun() == csB.ProducerRun() {
		t.Fatalf("候选 A 与 B producer run 相同 %s(AC2)", csA.ProducerRun())
	}

	ctx := context.Background()
	// A、B 各自成行且全字段保真(经 changeset.New 重推身份对账的读路径)。
	gotA, err := runs.ChangeSet(ctx, csA.ID())
	if err != nil {
		t.Fatalf("ChangeSet(A): %v", err)
	}
	gotB, err := runs.ChangeSet(ctx, csB.ID())
	if err != nil {
		t.Fatalf("ChangeSet(B): %v", err)
	}
	for _, pair := range [][2]*changeset.ChangeSet{{csA, gotA}, {csB, gotB}} {
		want, got := pair[0], pair[1]
		if got.ID() != want.ID() || got.BaseRevision() != want.BaseRevision() ||
			got.CandidateRevision() != want.CandidateRevision() || got.PatchIdentity() != want.PatchIdentity() ||
			got.ProducerRun() != want.ProducerRun() || !got.CreatedAt().Equal(want.CreatedAt()) {
			t.Fatalf("快照 %s 读回不一致(AC2): got id=%s base=%s run=%s",
				want.ID(), got.ID(), got.BaseRevision(), got.ProducerRun())
		}
		gf, wf := got.Files(), want.Files()
		if len(gf) != len(wf) || len(gf) != 1 || gf[0].Path != wf[0].Path || gf[0].Content != wf[0].Content {
			t.Fatalf("快照 %s 改动文件读回不一致(AC2): %+v != %+v", want.ID(), gf, wf)
		}
	}
	// B 的内容不覆盖 A:A 的行仍是 v1 内容。
	if gotA.Files()[0].Content != "package widget // candidate v1\n" {
		t.Fatalf("B 落库后 A 的内容被改写(AC2): %q", gotA.Files()[0].Content)
	}

	// Task 聚合点恰好两个快照,身份各就各位。
	list, err := runs.ChangeSetsForTask(ctx, e2eProject, e2eTask)
	if err != nil {
		t.Fatalf("ChangeSetsForTask: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("落库 ChangeSet 数 = %d, want 2(AC2)", len(list))
	}
	ids := map[changeset.ChangeSetID]bool{list[0].ID(): true, list[1].ID(): true}
	if !ids[csA.ID()] || !ids[csB.ID()] {
		t.Fatalf("落库快照身份 = %v, want {A,B}(AC2)", ids)
	}

	// 不可变:重复保存同一快照幂等,不产生第三行;数据库层任何 UPDATE/DELETE
	// 都被触发器拒绝(AC2 的第二道防线)。
	if err := runs.SaveChangeSet(ctx, csA); err != nil {
		t.Fatalf("重复 SaveChangeSet(A) 应幂等: %v", err)
	}
	if n := countRows(t, pool, "SELECT count(*) FROM changeset WHERE project_id=$1", string(e2eProject)); n != 2 {
		t.Fatalf("重复保存后 changeset 行数 = %d, want 2(AC2)", n)
	}
	mustModifyRejected(t, pool, "UPDATE changeset SET base_commit='evil' WHERE id=$1", string(csA.ID()))
	mustModifyRejected(t, pool, "DELETE FROM changeset WHERE id=$1", string(csA.ID()))
	mustModifyRejected(t, pool, "UPDATE changeset SET changed_files='[]'::jsonb WHERE id=$1", string(csB.ID()))
	mustModifyRejected(t, pool, "DELETE FROM changeset WHERE id=$1", string(csB.ID()))
}

// ---- AC3:取代事实 append-only,不改动被取代快照 ----------------------------

// rawChangeSetRow 是 changeset 行的裸投影:取代前后逐列对比,证明快照本体
// 没有被任何路径改写(AC3)。
type rawChangeSetRow struct {
	baseCommit      string
	candidateCommit *string
	patchHash       *string
	changedFiles    []byte
	entities        []byte
	producerRunID   int64
	createdAt       time.Time
}

func rawChangeSet(t *testing.T, pool *pgxpool.Pool, id changeset.ChangeSetID) rawChangeSetRow {
	t.Helper()
	var r rawChangeSetRow
	err := pool.QueryRow(context.Background(),
		"SELECT base_commit, candidate_commit, patch_hash, changed_files, entities, producer_run_id, created_at FROM changeset WHERE id=$1",
		string(id)).Scan(&r.baseCommit, &r.candidateCommit, &r.patchHash, &r.changedFiles, &r.entities, &r.producerRunID, &r.createdAt)
	if err != nil {
		t.Fatalf("读 raw changeset %s: %v", id, err)
	}
	return r
}

func sameChangeSetRow(a, b rawChangeSetRow) bool {
	return a.baseCommit == b.baseCommit &&
		((a.candidateCommit == nil) == (b.candidateCommit == nil)) &&
		(a.candidateCommit == nil || *a.candidateCommit == *b.candidateCommit) &&
		((a.patchHash == nil) == (b.patchHash == nil)) &&
		(a.patchHash == nil || *a.patchHash == *b.patchHash) &&
		string(a.changedFiles) == string(b.changedFiles) &&
		string(a.entities) == string(b.entities) &&
		a.producerRunID == b.producerRunID &&
		a.createdAt.Equal(b.createdAt)
}

func TestV13SupersessionDoesNotMutatePriorSnapshot(t *testing.T) {
	runs, pool := e2eStore(t)
	outA, outB, _ := runTwoCandidates(t, runs)
	csA, csB := outA.ChangeSet, outB.ChangeSet

	// 取代前的 A 裸行快照:取代关系的表达不得触碰它(AC3)。
	before := rawChangeSet(t, pool, csA.ID())
	// 取代前不存在任何关系事实。
	if n := countRows(t, pool, "SELECT count(*) FROM changeset_superseded_fact"); n != 0 {
		t.Fatalf("取代前已有 %d 行取代事实", n)
	}

	fact, err := changeset.NewSupersededFact(csA.ID(), csB.ID(), changeset.SupersedeRework, e2eAt2)
	if err != nil {
		t.Fatalf("NewSupersededFact: %v", err)
	}
	if err := runs.AppendSupersededFact(context.Background(), fact); err != nil {
		t.Fatalf("AppendSupersededFact: %v", err)
	}

	// A 的裸行逐列不变:取代不修改、不删行、不回填任何 superseded 字段。
	after := rawChangeSet(t, pool, csA.ID())
	if !sameChangeSetRow(before, after) {
		t.Fatalf("取代后 A 的行被改写(AC3): %+v != %+v", before, after)
	}
	// 领域读路径同样保真(身份重推对账仍成立)。
	gotA, err := runs.ChangeSet(context.Background(), csA.ID())
	if err != nil || gotA.ID() != csA.ID() || !gotA.CreatedAt().Equal(csA.CreatedAt()) {
		t.Fatalf("取代后 A 读回不一致: %v %s", err, gotA.ID())
	}

	// 快照间唯一的联系是这条 append-only 事实:old/new/reason/时刻各就各位,
	// 从 old 与 new 两侧投影均可查,全量投影恰好一条。
	got, err := runs.ChangeSetSupersessions(context.Background(), csA.ID())
	if err != nil || len(got) != 1 {
		t.Fatalf("ChangeSetSupersessions(A): %v %d", err, len(got))
	}
	if got[0].Old != csA.ID() || got[0].New != csB.ID() || got[0].Reason != changeset.SupersedeRework || !got[0].At.Equal(e2eAt2) {
		t.Fatalf("取代事实重建不一致(AC3): %+v", got[0])
	}
	if fromB, err := runs.ChangeSetSupersessions(context.Background(), csB.ID()); err != nil || len(fromB) != 1 {
		t.Fatalf("new 侧取代事实: %v %d", err, len(fromB))
	}
	if all, err := runs.AllChangeSetSupersessions(context.Background()); err != nil || len(all) != 1 {
		t.Fatalf("AllChangeSetSupersessions: %v %d", err, len(all))
	}

	// 后续返工的正确形态 = 新快照 + 新事实,而不是改写 A 或旧事实:
	// 事实行 append-only(UPDATE/DELETE 被触发器拒绝),A 本体同样仍只读。
	mustModifyRejected(t, pool, "UPDATE changeset_superseded_fact SET reason='escalation'")
	mustModifyRejected(t, pool, "DELETE FROM changeset_superseded_fact")
	mustModifyRejected(t, pool, "UPDATE changeset SET candidate_commit='rewritten' WHERE id=$1", string(csA.ID()))
	mustModifyRejected(t, pool, "DELETE FROM changeset WHERE id=$1", string(csA.ID()))
	// 事实仍在、仍精确一条:拒绝路径没有吞掉或变形既有事实。
	if n := countRows(t, pool, "SELECT count(*) FROM changeset_superseded_fact"); n != 1 {
		t.Fatalf("拒绝改写后取代事实行数 = %d, want 1(AC3)", n)
	}
}

// ---- AC4:验证精确绑定在快照身份上 ------------------------------------------

func TestV13VerificationBoundToExactSnapshot(t *testing.T) {
	runs, pool := e2eStore(t)
	outA, outB, _ := runTwoCandidates(t, runs)
	csA, csB := outA.ChangeSet, outB.ChangeSet
	ctx := context.Background()

	// 对候选 A 记一次通过执行的 verification_run:行绑定 A 的精确快照身份。
	vrA, err := runs.RecordVerificationRun(ctx, runstore.VerificationRunSpec{
		Project:           e2eProject,
		Task:              e2eTask,
		ContractRevision:  1,
		ContractHash:      e2eContractHash,
		ChangeSetID:       csA.ID(),
		EnvName:           "ci",
		EnvDefinitionHash: "sha256:p1-50-envdef",
		VersionCombo:      core.VersionCombo{"go": "1.27", "os": "windows"},
		Runtime:           "fake",
		StartedAt:         e2eAt,
		FinishedAt:        e2eAt.Add(time.Minute),
		Status:            runstore.VerificationPass,
	})
	if err != nil {
		t.Fatalf("RecordVerificationRun(A): %v", err)
	}
	got, err := runs.VerificationRun(ctx, e2eProject, vrA)
	if err != nil {
		t.Fatalf("VerificationRun: %v", err)
	}
	if got.ChangeSetID != csA.ID() {
		t.Fatalf("verification_run 绑定 %s, want 精确快照 %s(AC4)", got.ChangeSetID, csA.ID())
	}

	// 证据事实经 Run 行取得 scope:ChangeSet 维度就是 A 的快照身份。
	evID, err := runs.RecordEvidence(ctx, e2eProject, runstore.EvidenceRecord{
		FactID:           "fact-p1-50-a",
		RunID:            vrA,
		Task:             e2eTask,
		Acceptance:       "AC-01",
		BindingID:        "v1",
		VisibilityPolicy: verification.VisibilityVisible,
		Verdict:          evidence.VerdictPass,
		Recorder:         evidence.Recorder{Kind: evidence.RecorderVerificationRun, Actor: "tally/verification"},
		RecordedAt:       e2eAt.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("RecordEvidence: %v", err)
	}
	if evID <= 0 {
		t.Fatalf("evidence 行 id = %d", evID)
	}

	// A 的 scope 精确命中;把 ChangeSet 维度换成 B 的其余四维全同 → 零命中:
	// 验证与证据不跨快照继承,新候选必须有自己的验证(AC4)。
	scopeA := evidence.Scope{
		Project:      e2eProject,
		Task:         evidence.TaskID(e2eTask),
		Acceptance:   "AC-01",
		Verification: evidence.VerificationRevision(e2eContractHash),
		ChangeSet:    evidence.ChangeSetID(csA.ID()),
	}
	hitsA, err := runs.ApplicableValidEvidence(ctx, scopeA, 1)
	if err != nil {
		t.Fatalf("ApplicableValidEvidence(A): %v", err)
	}
	if len(hitsA) != 1 || hitsA[0].FactID != "fact-p1-50-a" || hitsA[0].Scope.ChangeSet != evidence.ChangeSetID(csA.ID()) {
		t.Fatalf("A scope 命中不一致(AC4): %+v", hitsA)
	}
	scopeB := scopeA
	scopeB.ChangeSet = evidence.ChangeSetID(csB.ID())
	hitsB, err := runs.ApplicableValidEvidence(ctx, scopeB, 1)
	if err != nil {
		t.Fatalf("ApplicableValidEvidence(B): %v", err)
	}
	if len(hitsB) != 0 {
		t.Fatalf("B 的 scope 命中了 %d 条 A 的证据,验证未精确绑定快照(AC4)", len(hitsB))
	}

	// 历史投影按快照身份分流:A 的 Run 只在 A 名下。
	forA, err := runs.VerificationRunsForChangeSet(ctx, e2eProject, csA.ID())
	if err != nil || len(forA) != 1 || forA[0].ID != vrA {
		t.Fatalf("VerificationRunsForChangeSet(A): %v %d", err, len(forA))
	}
	forB, err := runs.VerificationRunsForChangeSet(ctx, e2eProject, csB.ID())
	if err != nil || len(forB) != 0 {
		t.Fatalf("B 名下出现 %d 条 verification_run(AC4)", len(forB))
	}

	// 取代(A→B)之后:B 作为新候选拿到自己的独立验证记录;两条记录各自成行,
	// A 的历史验证不被改写、不被挪用。
	if err := runs.AppendSupersededFact(ctx, mustSupersededFact(t, csA.ID(), csB.ID())); err != nil {
		t.Fatalf("AppendSupersededFact: %v", err)
	}
	vrB, err := runs.RecordVerificationRun(ctx, runstore.VerificationRunSpec{
		Project:           e2eProject,
		Task:              e2eTask,
		ContractRevision:  1,
		ContractHash:      e2eContractHash,
		ChangeSetID:       csB.ID(),
		EnvName:           "ci",
		EnvDefinitionHash: "sha256:p1-50-envdef",
		VersionCombo:      core.VersionCombo{"go": "1.27", "os": "windows"},
		Runtime:           "fake",
		StartedAt:         e2eAt2,
		FinishedAt:        e2eAt2.Add(time.Minute),
		Status:            runstore.VerificationPass,
	})
	if err != nil {
		t.Fatalf("RecordVerificationRun(B): %v", err)
	}
	if vrB == vrA {
		t.Fatal("B 的验证不应复用 A 的 Run 行(AC4)")
	}
	if forA, err = runs.VerificationRunsForChangeSet(ctx, e2eProject, csA.ID()); err != nil || len(forA) != 1 || forA[0].ID != vrA {
		t.Fatalf("取代后 A 的验证历史被改写(AC4): %v %d", err, len(forA))
	}
	if forB, err = runs.VerificationRunsForChangeSet(ctx, e2eProject, csB.ID()); err != nil || len(forB) != 1 || forB[0].ID != vrB {
		t.Fatalf("B 的验证历史不一致(AC4): %v %d", err, len(forB))
	}

	// 绑定本身不可变:verification_run 是 append-only 事实,不能把 A 的验证
	// 改挂到 B(AC4 的数据库侧防线)。
	mustModifyRejected(t, pool, "UPDATE verification_run SET changeset_id=$1 WHERE id=$2", string(csB.ID()), vrA)
	mustModifyRejected(t, pool, "DELETE FROM verification_run WHERE id=$1", vrA)
}

// mustSupersededFact 构造取代事实(A→B,rework),独立小助手仅为让 AC4 用例
// 的失败信息聚焦。
func mustSupersededFact(t *testing.T, oldID, newID changeset.ChangeSetID) changeset.SupersededFact {
	t.Helper()
	f, err := changeset.NewSupersededFact(oldID, newID, changeset.SupersedeRework, e2eAt2)
	if err != nil {
		t.Fatalf("NewSupersededFact: %v", err)
	}
	return f
}

// 哨兵引用检查:runstore 适配器的 producer run 引用必须是持久化 Run 主键的
// 十进制串(workflow.RunStoreSink 合同)。这里机械核对 A 的快照确实回指其
// 产生 Run 的行 —— 保持快照可追溯到过程记录(PRD §15.3)。
func TestV13ChangeSetProducerRunReferencesStoredRun(t *testing.T) {
	runs, _ := e2eStore(t)
	outA, _, _ := runTwoCandidates(t, runs)

	runID, err := strconv.ParseInt(outA.ChangeSet.ProducerRun(), 10, 64)
	if err != nil || runID <= 0 {
		t.Fatalf("producer run 引用 %q 不是存储层 Run 主键", outA.ChangeSet.ProducerRun())
	}
	run, err := runs.CodegenRun(context.Background(), runID)
	if err != nil {
		t.Fatalf("CodegenRun(%d): %v", runID, err)
	}
	if run.ResultPatchHash != string(outA.ChangeSet.ID()) {
		t.Fatalf("Run %d 的 result_patch_hash = %q, want 快照身份 %s", runID, run.ResultPatchHash, outA.ChangeSet.ID())
	}
}

// ---- P0-07:祖先暴露 lineage 投影(取代不能掩盖历史泄漏) --------------------

// TestV13AncestorExposureStaysVisibleAcrossSupersession 证明 P0-07:读取模型
// 沿 append-only 取代事实确定性展开祖先候选,并把暴露投影到整条 lineage 上
// —— 锚定祖先的 held-out 暴露在当前候选自身干净(候选局部查询为空)时也不
// 消失(AC1/AC3);投影是纯读取,不新增任何可变暴露状态(AC4)。推导侧的
// lineage 语义(祖先暴露翻 independence)已由 v13_exposure_test.go 覆盖。
func TestV13AncestorExposureStaysVisibleAcrossSupersession(t *testing.T) {
	runs, pool := e2eStore(t)
	ctx := context.Background()
	repo := e2eNewRepo(t)
	base := e2eCommit(t, repo, "README.md", "base\n", "base")

	e2eAt3 := e2eAt2.Add(time.Hour)
	outA := runCandidate(t, runs, repo, base, "v1", e2eAt, false)
	outB := runCandidate(t, runs, repo, base, "v2", e2eAt2, false)
	outC := runCandidate(t, runs, repo, base, "v3", e2eAt3, false)
	csA, csB, csC := outA.ChangeSet, outB.ChangeSet, outC.ChangeSet
	if csA == nil || csB == nil || csC == nil {
		t.Fatalf("三代候选未全部固定: A=%v B=%v C=%v", csA, csB, csC)
	}

	// 三代取代链:A→B→B→C(append-only 事实;重做 = 新快照 + 新事实)。
	for _, pair := range []struct {
		old, new *changeset.ChangeSet
		at       time.Time
	}{
		{csA, csB, e2eAt2},
		{csB, csC, e2eAt3},
	} {
		fact, err := changeset.NewSupersededFact(pair.old.ID(), pair.new.ID(), changeset.SupersedeRework, pair.at)
		if err != nil {
			t.Fatalf("NewSupersededFact: %v", err)
		}
		if err := runs.AppendSupersededFact(ctx, fact); err != nil {
			t.Fatalf("AppendSupersededFact: %v", err)
		}
	}

	// AC1:祖先展开确定性 —— C → [B, A](最近在前,层内按身份排序),重复
	// 调用同输出;B → [A];A → []。
	ancC, err := runs.ChangeSetAncestors(ctx, csC.ID())
	if err != nil {
		t.Fatalf("ChangeSetAncestors(C): %v", err)
	}
	if len(ancC) != 2 || ancC[0] != csB.ID() || ancC[1] != csA.ID() {
		t.Fatalf("ancestors(C) = %v, want [B, A]", ancC)
	}
	ancC2, err := runs.ChangeSetAncestors(ctx, csC.ID())
	if err != nil || len(ancC2) != 2 || ancC2[0] != ancC[0] || ancC2[1] != ancC[1] {
		t.Fatalf("ancestors(C) 非确定: %v != %v(%v)", ancC, ancC2, err)
	}
	if ancB, err := runs.ChangeSetAncestors(ctx, csB.ID()); err != nil || len(ancB) != 1 || ancB[0] != csA.ID() {
		t.Fatalf("ancestors(B) = %v(%v), want [A]", ancB, err)
	}
	if ancA, err := runs.ChangeSetAncestors(ctx, csA.ID()); err != nil || len(ancA) != 0 {
		t.Fatalf("ancestors(A) = %v(%v), want 空", ancA, err)
	}

	// held-out 暴露锚定在最老祖先 A(lineage 局部通道;Reason 只是追溯
	// provenance,不携带 held-out 内容,INV-VER-05)。
	expA, err := runs.RecordExposure(ctx, runstore.ExposureRecord{
		Task:             e2eTask,
		ContractRevision: 1,
		Binding:          evidence.BindingRef{Project: e2eProject, Verification: evidence.VerificationRevision(e2eContractHash), BindingID: "v1-held"},
		Acceptance:       "AC-01",
		ChangeSet:        evidence.ChangeSetID(csA.ID()),
		Channel:          evidence.ChannelDebugDisclosure,
		Reason:           "held-out diagnostic surfaced during first attempt",
		ExposedAt:        e2eAt2.Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("RecordExposure(A): %v", err)
	}

	// AC3:当前候选 B、C 自身干净(候选局部投影为空),祖先暴露经 lineage
	// 投影仍然可见 —— 取代不重置暴露账,修复不能掩盖既有信息泄漏。
	for _, cs := range []*changeset.ChangeSet{csB, csC} {
		local, err := runs.ExposuresForChangeSet(ctx, e2eProject, cs.ID())
		if err != nil {
			t.Fatalf("ExposuresForChangeSet(%s): %v", cs.ID(), err)
		}
		if len(local) != 0 {
			t.Fatalf("候选 %s 的局部暴露应空(自身干净),得到 %d 条", cs.ID(), len(local))
		}
	}
	lineageB, err := runs.ExposuresForChangeSetLineage(ctx, e2eProject, csB.ID())
	if err != nil {
		t.Fatalf("ExposuresForChangeSetLineage(B): %v", err)
	}
	if len(lineageB) != 1 || lineageB[0].Exposure.ID() != expA.ID() ||
		lineageB[0].ChangeSet != csA.ID() || lineageB[0].AncestorDepth != 1 {
		t.Fatalf("lineage(B) = %+v, want 祖先 A 暴露(depth 1)仍可见", lineageB)
	}
	lineageC, err := runs.ExposuresForChangeSetLineage(ctx, e2eProject, csC.ID())
	if err != nil {
		t.Fatalf("ExposuresForChangeSetLineage(C): %v", err)
	}
	if len(lineageC) != 1 || lineageC[0].Exposure.ID() != expA.ID() ||
		lineageC[0].ChangeSet != csA.ID() || lineageC[0].AncestorDepth != 2 {
		t.Fatalf("lineage(C) = %+v, want 祖先 A 暴露(depth 2)仍可见(AC3)", lineageC)
	}

	// AC4:投影零写副作用 —— 暴露事实面恰 1 行、原样未改(append-only 触发器
	// 拒绝改写;不存在任何可变暴露状态字段)。
	if n := countRows(t, pool, "SELECT count(*) FROM evidence_exposure"); n != 1 {
		t.Fatalf("lineage 投影后暴露行数 = %d, want 1(投影零写副作用)", n)
	}
	mustModifyRejected(t, pool, "UPDATE evidence_exposure SET changeset_id=$1 WHERE id=$2", string(csC.ID()), string(expA.ID()))
	if still, err := runs.ExposuresForChangeSetLineage(ctx, e2eProject, csC.ID()); err != nil ||
		len(still) != 1 || still[0].Exposure.ID() != expA.ID() || still[0].ChangeSet != csA.ID() || still[0].AncestorDepth != 2 {
		t.Fatalf("拒绝改写后 lineage 投影不一致: %v %+v", err, still)
	}
}
