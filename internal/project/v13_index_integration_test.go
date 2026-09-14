//go:build integration

package project

// v13_index 的集成测试:真实 Postgres(store.TestDB,每测试独立 schema)+ 真实
// git 仓库。链路是 P1-32 的垂直切片:Git 快照(internal/artifacts 装载)→
// 项目身份 guard → task_index / verification_index 三步重建。
//
//	AC1 Git 是唯一来源(装载自提交快照,索引可从 Git 重derive);
//	AC2 无效制品不产生索引行;
//	AC3 重建幂等且可整体重建(清空后按同一快照重建结果一致);
//	AC4 注册表之外的项目(ghost)被拒收;
//	AC5 索引面不持有权威状态(重建只写索引表,事实面一字节不动)。

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"

	"tally/internal/artifacts"
	"tally/internal/store"
)

// v13TaskYAML 生成一份合法的 standard TaskContract(能过 Parse + ValidateStatic)。
func v13TaskYAML(id, objective string) string {
	return `id: ` + id + `
profile: standard
title: 索引测试任务
objective: ` + objective + `
product_refs:
  - .tally/product/PRD.md#index
requirement_refs:
  - .tally/requirements/REQ-0001.md
scope:
  allowed_paths:
    - src/**
  forbidden_paths:
    - vendor/**
acceptance:
  - id: AC-01
    statement: 索引测试验收
    independence_required: none
`
}

// v13BrokenTaskYAML 缺 objective:装载必拒(AC2 的无效制品)。
func v13BrokenTaskYAML(id string) string {
	return `id: ` + id + `
profile: light
scope:
  allowed_paths:
    - src/**
`
}

// v13VerificationYAML 生成一份可冻结的 VerificationContract(P1 standard path)。
func v13VerificationYAML(taskID string, revision int) string {
	return `task_id: ` + taskID + `
revision: ` + strconv.Itoa(revision) + `
acceptances:
  - id: AC-01
    statement: 索引测试验收
    independence_required: none
    evidence_bindings:
      - id: EV-AC01-01
        authoring_mode: pre_commit
        visibility_policy: visible
        evidence:
          type: validator
          validator: index-fixture-v1
        qualification:
          change_overlap: required
`
}

// v13IndexInput 从 git 仓库 HEAD 装载快照并适配成索引输入(组合方适配的样例)。
func v13IndexInput(t *testing.T, dir string) V13Snapshot {
	t.Helper()
	snap, err := artifacts.Load(dir, "HEAD")
	if err != nil {
		t.Fatalf("装载 git 快照: %v", err)
	}
	in := V13Snapshot{ProjectID: snap.ProjectID, SourceCommit: snap.SourceCommit}
	for _, a := range snap.Tasks {
		in.Tasks = append(in.Tasks, V13TaskRevision{
			TaskID:       a.Contract.ID,
			RevisionHash: a.Contract.Hash,
			Path:         a.Path,
		})
	}
	for _, v := range snap.Verifications {
		in.Verifications = append(in.Verifications, V13VerificationRevision{
			TaskID:     v.Frozen.TaskID(),
			Revision:   int32(v.Frozen.Revision()),
			FrozenHash: v.Frozen.Hash(),
			Path:       v.Path,
		})
	}
	return in
}

// v13TaskRowKey / v13VerificationRowKey 把索引行渲染成可比较的元组串
// (derived_at 是派生时刻,不参与内容比较)。
func v13TaskRowKey(r store.TaskIndex) string {
	return strings.Join([]string{r.ProjectID, r.TaskID, r.Revision, r.Path, strconv.FormatBool(r.IsCurrent), r.SourceCommit}, "|")
}

func v13VerificationRowKey(r store.VerificationIndex) string {
	return strings.Join([]string{r.ProjectID, r.TaskID, strconv.Itoa(int(r.Revision)), r.Hash, r.Path, strconv.FormatBool(r.IsCurrent), r.SourceCommit}, "|")
}

func v13TaskRowKeys(t *testing.T, st *store.Store, project string) []string {
	t.Helper()
	rows, err := st.Queries().ListTaskIndex(context.Background())
	if err != nil {
		t.Fatalf("列 task_index: %v", err)
	}
	var keys []string
	for _, r := range rows {
		if r.ProjectID == project {
			keys = append(keys, v13TaskRowKey(r))
		}
	}
	sort.Strings(keys)
	return keys
}

func v13VerificationRowKeys(t *testing.T, st *store.Store, project string) []string {
	t.Helper()
	rows, err := st.Queries().ListVerificationIndex(context.Background())
	if err != nil {
		t.Fatalf("列 verification_index: %v", err)
	}
	var keys []string
	for _, r := range rows {
		if r.ProjectID == project {
			keys = append(keys, v13VerificationRowKey(r))
		}
	}
	sort.Strings(keys)
	return keys
}

// removeGitFile 从工作区删除文件并提交删除(测「合同从快照消失」的重建)。
func removeGitFile(t *testing.T, repo *git.Repository, name, msg string, when time.Time) string {
	t.Helper()
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("取工作区: %v", err)
	}
	if _, err := wt.Remove(name); err != nil {
		t.Fatalf("删除 %s: %v", name, err)
	}
	sig := &object.Signature{Name: "test", Email: "test@example.com", When: when}
	h, err := wt.Commit(msg, &git.CommitOptions{Author: sig, Committer: sig, AllowEmptyCommits: true})
	if err != nil {
		t.Fatalf("提交删除: %v", err)
	}
	return h.String()
}

// TestRebuildV13IndexesSnapshotRebuildable:端到端垂直切片 ——
// 快照重建进索引、当前投影就位、无效制品不入索引;重建幂等;快照轮转后旧行
// 被收走;清空索引后按同一 Git 快照重建得回同一内容(AC1/AC2/AC3/AC5)。
func TestRebuildV13IndexesSnapshotRebuildable(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	q := st.Queries()
	run := runID(t)
	project := "proj-" + run

	// 注册表派生索引先收录项目(0017 的增量路径;guard 以它为准)。
	if err := q.UpsertProjectIndex(ctx, store.UpsertProjectIndexParams{ProjectID: project, SourceCommit: "seed-" + run}); err != nil {
		t.Fatalf("登记项目: %v", err)
	}

	// Git 侧:注册表 + 两份有效合同 + 一份冻结验证契约 + 一份坏合同。
	dir, repo := initTestRepo(t)
	writeCommit(t, repo, dir, ".tally/project.yaml", "project: "+project+"\n", "注册表", base)
	writeCommit(t, repo, dir, ".tally/tasks/TASK-0501.yaml", v13TaskYAML("TASK-0501", "目标 v1"), "任务 501 v1", base.Add(time.Minute))
	writeCommit(t, repo, dir, ".tally/tasks/TASK-0502.yaml", v13TaskYAML("TASK-0502", "任务二"), "任务 502", base.Add(2*time.Minute))
	writeCommit(t, repo, dir, ".tally/verifications/TASK-0501.yaml", v13VerificationYAML("TASK-0501", 3), "验证契约 501", base.Add(3*time.Minute))
	head := writeCommit(t, repo, dir, ".tally/tasks/TASK-0503.yaml", v13BrokenTaskYAML("TASK-0503"), "坏合同", base.Add(4*time.Minute))

	in := v13IndexInput(t, dir)
	if in.ProjectID != project || in.SourceCommit != head {
		t.Fatalf("快照身份/来源提交不对: %s@%s", in.ProjectID, in.SourceCommit)
	}
	if len(in.Tasks) != 2 || len(in.Verifications) != 1 {
		t.Fatalf("快照应有 2 份有效合同 + 1 份冻结契约, got %d/%d", len(in.Tasks), len(in.Verifications))
	}

	rep, err := RebuildV13Indexes(ctx, st, in)
	if err != nil {
		t.Fatalf("重建索引: %v", err)
	}
	if rep.TasksIndexed != 2 || rep.VerificationsIndexed != 1 {
		t.Errorf("重建报告应为 2/1, got %d/%d", rep.TasksIndexed, rep.VerificationsIndexed)
	}

	taskKeys := v13TaskRowKeys(t, st, project)
	if len(taskKeys) != 2 {
		t.Fatalf("task_index 应有 2 行, got %v", taskKeys)
	}
	var t501 store.TaskIndex
	for _, r := range mustTaskRows(t, st, project) {
		if r.TaskID == "TASK-0501" {
			t501 = r
		}
	}
	if t501.TaskID == "" {
		t.Fatal("task_index 应含 TASK-0501")
	}
	if !t501.IsCurrent || t501.Path != ".tally/tasks/TASK-0501.yaml" || t501.SourceCommit != head {
		t.Errorf("TASK-0501 当前投影不对: %+v", t501)
	}
	if t501.Revision != v13SnapshotRevision(t, in, "TASK-0501") {
		t.Errorf("revision 应为内容修订哈希 %s, got %s", v13SnapshotRevision(t, in, "TASK-0501"), t501.Revision)
	}
	// 无效制品(Task-0503 缺 objective)被拒于索引之外(AC2)。
	for _, k := range taskKeys {
		if strings.Contains(k, "TASK-0503") {
			t.Errorf("无效合同不应入索引: %s", k)
		}
	}

	vrows := mustVerificationRows(t, st, project)
	if len(vrows) != 1 {
		t.Fatalf("verification_index 应有 1 行, got %d", len(vrows))
	}
	if vrows[0].TaskID != "TASK-0501" || vrows[0].Revision != 3 || !vrows[0].IsCurrent ||
		vrows[0].Path != ".tally/verifications/TASK-0501.yaml" || vrows[0].SourceCommit != head {
		t.Errorf("验证契约当前投影不对: %+v", vrows[0])
	}
	if vrows[0].Hash != v13SnapshotHash(t, in, "TASK-0501") {
		t.Errorf("hash 应为冻结内容哈希 %s, got %s", v13SnapshotHash(t, in, "TASK-0501"), vrows[0].Hash)
	}

	// 幂等:同一快照再重建一次,内容一致(AC3)。
	if _, err := RebuildV13Indexes(ctx, st, in); err != nil {
		t.Fatalf("第二次重建: %v", err)
	}
	if got := v13TaskRowKeys(t, st, project); !equalStringSlices(taskKeys, got) {
		t.Errorf("重建应幂等:\nbefore %v\nafter  %v", taskKeys, got)
	}
	if got := v13VerificationRowKeys(t, st, project); len(got) != 1 {
		t.Errorf("验证索引应仍为 1 行, got %v", got)
	}

	// 轮转:TASK-0501 出新修订,TASK-0502 从快照删除 → 旧修订/消失合同被收走。
	writeCommit(t, repo, dir, ".tally/tasks/TASK-0501.yaml", v13TaskYAML("TASK-0501", "目标 v2"), "任务 501 v2", base.Add(5*time.Minute))
	head2 := removeGitFile(t, repo, ".tally/tasks/TASK-0502.yaml", "删除任务 502", base.Add(6*time.Minute))
	in2 := v13IndexInput(t, dir)
	if _, err := RebuildV13Indexes(ctx, st, in2); err != nil {
		t.Fatalf("轮转重建: %v", err)
	}
	rows2 := mustTaskRows(t, st, project)
	if len(rows2) != 1 {
		t.Fatalf("轮转后 task_index 应只剩 TASK-0501 新修订, got %v", rows2)
	}
	if rows2[0].Revision == t501.Revision || rows2[0].SourceCommit != head2 || !rows2[0].IsCurrent {
		t.Errorf("轮转后的当前投影不对: %+v", rows2[0])
	}

	// 可整体重建:清空两张索引表,按同一快照重建 → 与清空前完全一致(AC5:
	// 索引面不持有任何不可重derive的权威状态)。
	beforeWipe := v13TaskRowKeys(t, st, project)
	beforeWipeV := v13VerificationRowKeys(t, st, project)
	if _, err := st.Pool().Exec(ctx, `DELETE FROM task_index WHERE project_id = $1`, project); err != nil {
		t.Fatalf("清空 task_index: %v", err)
	}
	if _, err := st.Pool().Exec(ctx, `DELETE FROM verification_index WHERE project_id = $1`, project); err != nil {
		t.Fatalf("清空 verification_index: %v", err)
	}
	if _, err := RebuildV13Indexes(ctx, st, in2); err != nil {
		t.Fatalf("清空后重建: %v", err)
	}
	if got := v13TaskRowKeys(t, st, project); !equalStringSlices(beforeWipe, got) {
		t.Errorf("清空后重建应得回同一内容:\nbefore %v\nafter  %v", beforeWipe, got)
	}
	if got := v13VerificationRowKeys(t, st, project); !equalStringSlices(beforeWipeV, got) {
		t.Errorf("验证索引清空后重建应一致:\nbefore %v\nafter  %v", beforeWipeV, got)
	}
}

// TestRebuildV13IndexesGhostProjectGuard:注册表之外的项目一律拒收,一行索引
// 都不落(AC4,§33.3)。
func TestRebuildV13IndexesGhostProjectGuard(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	run := runID(t)
	ghost := "ghost-" + run // 故意不做 UpsertProjectIndex

	dir, repo := initTestRepo(t)
	writeCommit(t, repo, dir, ".tally/project.yaml", "project: "+ghost+"\n", "注册表", base)
	writeCommit(t, repo, dir, ".tally/tasks/TASK-0601.yaml", v13TaskYAML("TASK-0601", "幽灵任务"), "任务", base.Add(time.Minute))
	writeCommit(t, repo, dir, ".tally/verifications/TASK-0601.yaml", v13VerificationYAML("TASK-0601", 1), "契约", base.Add(2*time.Minute))

	in := v13IndexInput(t, dir)
	if _, err := RebuildV13Indexes(ctx, st, in); err == nil {
		t.Fatal("注册表之外的项目应被拒收")
	} else if !strings.Contains(err.Error(), ghost) {
		t.Errorf("错误应点名 ghost 项目, got %v", err)
	}
	if got := v13TaskRowKeys(t, st, ghost); len(got) != 0 {
		t.Errorf("ghost 项目不得有任何 task_index 行, got %v", got)
	}
	if got := v13VerificationRowKeys(t, st, ghost); len(got) != 0 {
		t.Errorf("ghost 项目不得有任何 verification_index 行, got %v", got)
	}
}

// TestRebuildV13IndexesFactsUntouched:重建只写两张索引表,事实面(与既有
// 索引面)一字节不动(AC5 的另一面)。
func TestRebuildV13IndexesFactsUntouched(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	q := st.Queries()
	run := runID(t)
	project := "proj-" + run
	h1 := "sha256:" + run + "-h1"
	seedClauseVersion(t, q, project, "REQ-1", h1, 1, base, true)

	dir, repo := initTestRepo(t)
	commit1 := writeCommit(t, repo, dir, "impl.go", "package x", "feat: 实现\n\nRefs: REQ-1", base.Add(time.Hour))
	art := "sha256:art-" + run
	b1 := seedBuild(t, q, project, commit1, art, base.Add(2*time.Hour))
	d1 := seedDeploy(t, q, project, art, b1, base.Add(2*time.Hour+time.Minute))
	seedDeployFailure(t, q, project, d1, "fp-"+run, base.Add(2*time.Hour+2*time.Minute))

	edgesBefore, err := q.ListTraceEdges(ctx)
	if err != nil {
		t.Fatalf("列追溯边: %v", err)
	}
	factsBefore := countFailureEvents(t, st, project)

	// v1.3 制品 + 一次重建。
	writeCommit(t, repo, dir, ".tally/project.yaml", "project: "+project+"\n", "注册表", base.Add(3*time.Hour))
	writeCommit(t, repo, dir, ".tally/tasks/TASK-0701.yaml", v13TaskYAML("TASK-0701", "事实面不受影响"), "任务", base.Add(3*time.Hour+time.Minute))
	if err := q.UpsertProjectIndex(ctx, store.UpsertProjectIndexParams{ProjectID: project, SourceCommit: "seed-" + run}); err != nil {
		t.Fatalf("登记项目: %v", err)
	}
	if _, err := RebuildV13Indexes(ctx, st, v13IndexInput(t, dir)); err != nil {
		t.Fatalf("重建索引: %v", err)
	}

	if factsAfter := countFailureEvents(t, st, project); factsAfter != factsBefore {
		t.Errorf("重建不得碰事实面:失败事件 %d → %d", factsBefore, factsAfter)
	}
	edgesAfter, err := q.ListTraceEdges(ctx)
	if err != nil {
		t.Fatalf("列追溯边: %v", err)
	}
	if len(edgesAfter) != len(edgesBefore) {
		t.Errorf("重建不得碰既有追溯图:追溯边 %d → %d", len(edgesBefore), len(edgesAfter))
	}
}

// TestRebuildV13IndexesRejectsBadInput:快照硬约束在写库前拦下 —— 身份缺失、
// 来源缺失、当前投影重复、revision 非法都是确定性错误。
func TestRebuildV13IndexesRejectsBadInput(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	q := st.Queries()
	run := runID(t)
	project := "proj-" + run
	if err := q.UpsertProjectIndex(ctx, store.UpsertProjectIndexParams{ProjectID: project, SourceCommit: "seed-" + run}); err != nil {
		t.Fatalf("登记项目: %v", err)
	}

	cases := []struct {
		name string
		snap V13Snapshot
	}{
		{"缺 project_id", V13Snapshot{SourceCommit: "c1"}},
		{"缺 source_commit", V13Snapshot{ProjectID: project}},
		{"task 行字段不完整", V13Snapshot{ProjectID: project, SourceCommit: "c1",
			Tasks: []V13TaskRevision{{TaskID: "TASK-0001"}}}},
		{"task 当前投影重复", V13Snapshot{ProjectID: project, SourceCommit: "c1",
			Tasks: []V13TaskRevision{
				{TaskID: "TASK-0001", RevisionHash: "sha256:a", Path: "p1"},
				{TaskID: "TASK-0001", RevisionHash: "sha256:b", Path: "p2"},
			}}},
		{"verification revision 非法", V13Snapshot{ProjectID: project, SourceCommit: "c1",
			Verifications: []V13VerificationRevision{{TaskID: "TASK-0001", Revision: 0, FrozenHash: "sha256:a", Path: "p"}}}},
		{"verification 当前投影重复", V13Snapshot{ProjectID: project, SourceCommit: "c1",
			Verifications: []V13VerificationRevision{
				{TaskID: "TASK-0001", Revision: 1, FrozenHash: "sha256:a", Path: "p1"},
				{TaskID: "TASK-0001", Revision: 2, FrozenHash: "sha256:b", Path: "p2"},
			}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := RebuildV13Indexes(ctx, st, tc.snap); err == nil {
				t.Fatal("坏快照应被拒收")
			}
		})
	}
}

// mustTaskRows / mustVerificationRows 读某项目的全部索引行(原始行,断言字段用)。
func mustTaskRows(t *testing.T, st *store.Store, project string) []store.TaskIndex {
	t.Helper()
	rows, err := st.Queries().ListTaskIndex(context.Background())
	if err != nil {
		t.Fatalf("列 task_index: %v", err)
	}
	var out []store.TaskIndex
	for _, r := range rows {
		if r.ProjectID == project {
			out = append(out, r)
		}
	}
	return out
}

func mustVerificationRows(t *testing.T, st *store.Store, project string) []store.VerificationIndex {
	t.Helper()
	rows, err := st.Queries().ListVerificationIndex(context.Background())
	if err != nil {
		t.Fatalf("列 verification_index: %v", err)
	}
	var out []store.VerificationIndex
	for _, r := range rows {
		if r.ProjectID == project {
			out = append(out, r)
		}
	}
	return out
}

func v13SnapshotRevision(t *testing.T, in V13Snapshot, taskID string) string {
	t.Helper()
	for _, r := range in.Tasks {
		if r.TaskID == taskID {
			return r.RevisionHash
		}
	}
	t.Fatalf("快照里没有任务 %s", taskID)
	return ""
}

func v13SnapshotHash(t *testing.T, in V13Snapshot, taskID string) string {
	t.Helper()
	for _, r := range in.Verifications {
		if r.TaskID == taskID {
			return r.FrozenHash
		}
	}
	t.Fatalf("快照里没有验证契约 %s", taskID)
	return ""
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
