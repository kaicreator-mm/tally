//go:build integration

package project

import (
	"context"
	"strings"
	"testing"
	"time"

	"tally/internal/core"
	"tally/internal/store"
)

// TestEdgeLinesProjectNamespace:diff 渲染带项目命名空间 —— 仅 project 不同的两条边
// 渲染成不同行,sortedDiff 才能数清、人也看得出归属(纯函数,不连库)。
func TestEdgeLinesProjectNamespace(t *testing.T) {
	edges := []store.TraceEdge{
		{ProjectID: "pA", FromType: NodeClause, FromKey: "h", ToType: NodeCommit, ToKey: "sha"},
		{ProjectID: "pB", FromType: NodeClause, FromKey: "h", ToType: NodeCommit, ToKey: "sha"},
	}
	lines := edgeLines(edges)
	if len(lines) != 2 || lines[0] == lines[1] {
		t.Fatalf("仅项目不同的两条边应渲染成两行, got %v", lines)
	}
	if !strings.HasPrefix(lines[0], "pA|") || !strings.HasPrefix(lines[1], "pB|") {
		t.Errorf("渲染应带 project 前缀, got %v", lines)
	}
	// 少了 pB 那条,sortedDiff 必须报出差异(命名空间不再被渲染折叠)。
	diff := sortedDiff(lines, lines[:1])
	if len(diff) != 1 || !strings.HasPrefix(diff[0], "- pB|") {
		t.Errorf("sortedDiff 应检出 pB 侧缺失, got %v", diff)
	}
}

// TestReindexIdempotentFactUntouched:重算两次结果一致;事实面一条记录都不动(INV-5)。
func TestReindexIdempotentFactUntouched(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	q := st.Queries()
	run := runID(t)
	project := "proj-" + run
	h1 := "sha256:" + run + "-h1"
	commit1 := "c" + run
	art := "sha256:art-" + run

	seedClauseVersion(t, q, project, "REQ-1", h1, 1, base, true)
	if err := q.InsertCommitClause(ctx, store.InsertCommitClauseParams{ProjectID: project, CommitSha: commit1, ClauseContentHash: h1}); err != nil {
		t.Fatalf("写 commit_clause: %v", err)
	}
	b1 := seedBuild(t, q, project, commit1, art, base)
	d1 := seedDeploy(t, q, project, art, b1, base.Add(time.Minute))
	seedDeployFailure(t, q, project, d1, "fp-"+run, base.Add(2*time.Minute))

	beforeFacts := countFailureEvents(t, st, project)

	rep1, err := Reindex(ctx, st, ReindexOptions{}) // 无 git:重算 trace_edge + 指纹统计
	if err != nil {
		t.Fatalf("第一次 reindex: %v", err)
	}
	edges1, _ := q.ListTraceEdges(ctx)
	rep2, err := Reindex(ctx, st, ReindexOptions{})
	if err != nil {
		t.Fatalf("第二次 reindex: %v", err)
	}
	edges2, _ := q.ListTraceEdges(ctx)

	if rep1.TraceEdges != rep2.TraceEdges || rep1.TraceEdges == 0 {
		t.Errorf("重算应幂等且非空: %d vs %d", rep1.TraceEdges, rep2.TraceEdges)
	}
	if !equalSorted(edgeSet(edges1), edgeSet(edges2)) {
		t.Error("重算两次 trace_edge 应一致")
	}
	if after := countFailureEvents(t, st, project); after != beforeFacts {
		t.Errorf("reindex 不得碰事实面:失败事件 %d → %d", beforeFacts, after)
	}
}

// TestReindexWithGitRepoThenVerify:从 git 重挂 trailer + 注解,随后 VerifyIndex 应一致。
func TestReindexWithGitRepoThenVerify(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	q := st.Queries()
	run := runID(t)
	project := core.ProjectID("proj-" + run)
	h1 := "sha256:" + run + "-h1"
	seedClauseVersion(t, q, string(project), "REQ-1", h1, 1, base, true)

	dir, repo := initTestRepo(t)
	sha := writeCommit(t, repo, dir, "impl.go", "package x", "feat: 实现\n\nRefs: REQ-1", base.Add(1*time.Hour))
	writeCommit(t, repo, dir, "impl_test.go", "package x\n\n// @covers REQ-1\nfunc TestX(t *testing.T) {}\n",
		"test: 加测试", base.Add(2*time.Hour))

	rep, err := Reindex(ctx, st, ReindexOptions{RepoPath: dir, ProjectID: project})
	if err != nil {
		t.Fatalf("reindex(带 git): %v", err)
	}
	if rep.CommitHooks != 1 || rep.TestHooks != 1 {
		t.Errorf("应重挂 1 commit 挂钩 + 1 测试挂钩, got %d/%d", rep.CommitHooks, rep.TestHooks)
	}

	// 挂钩落库且绑到 h1;追溯图含 clause→commit。
	// 注:若这里失败且 commit_clause 里出现 project_id 为空的挂钩行,根因在
	// internal/project/trailer.go 的 BindCommit —— InsertCommitClauseParams 漏传
	// ProjectID(0015 起是必填身份列),不传静默落空项目。该文件不在 P0-11 允许路径内。
	ccs, _ := q.ListCommitClauses(ctx)
	foundCommit := false
	for _, c := range ccs {
		if c.ProjectID == string(project) && c.CommitSha == sha && c.ClauseContentHash == h1 {
			foundCommit = true
		}
	}
	if !foundCommit {
		t.Error("commit 挂钩应绑到 h1")
	}
	wantEdge := store.TraceEdge{ProjectID: string(project), FromType: NodeClause, FromKey: h1, ToType: NodeCommit, ToKey: sha}
	if !hasEdge(t, q, wantEdge) {
		t.Error("追溯图应含 clause→commit 边")
	}

	// 单向数据流自检:重挂+重算后与库一致。
	diff, err := VerifyIndex(ctx, st, ReindexOptions{RepoPath: dir, ProjectID: project})
	if err != nil {
		t.Fatalf("VerifyIndex: %v", err)
	}
	if !diff.Consistent() {
		t.Errorf("刚重算完应一致:\n%s", diff.String())
	}
}

// TestVerifyIndexDetectsHandEdit:有人手改索引表(绕过单向数据流),VerifyIndex 应报不一致。
func TestVerifyIndexDetectsHandEdit(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	q := st.Queries()
	run := runID(t)
	project := core.ProjectID("proj-" + run)
	h1 := "sha256:" + run + "-h1"
	seedClauseVersion(t, q, string(project), "REQ-1", h1, 1, base, true)

	dir, repo := initTestRepo(t)
	writeCommit(t, repo, dir, "impl.go", "package x", "feat: 实现\n\nRefs: REQ-1", base.Add(time.Hour))
	if _, err := Reindex(ctx, st, ReindexOptions{RepoPath: dir, ProjectID: project}); err != nil {
		t.Fatalf("reindex: %v", err)
	}

	// 干净状态下应一致。
	diff, err := VerifyIndex(ctx, st, ReindexOptions{RepoPath: dir, ProjectID: project})
	if err != nil {
		t.Fatalf("VerifyIndex(干净): %v", err)
	}
	if !diff.Consistent() {
		t.Fatalf("干净状态应一致:\n%s", diff.String())
	}

	// 模拟绕过单向数据流:手插一条假追溯边 + 一条假 commit 挂钩。
	bogusEdge := store.TraceEdge{ProjectID: "bogus-" + run, FromType: NodeClause, FromKey: "bogus" + run, ToType: NodeCommit, ToKey: "bogus" + run}
	if err := q.InsertTraceEdge(ctx, store.InsertTraceEdgeParams{
		ProjectID: bogusEdge.ProjectID,
		FromType:  bogusEdge.FromType, FromKey: bogusEdge.FromKey, ToType: bogusEdge.ToType, ToKey: bogusEdge.ToKey,
	}); err != nil {
		t.Fatalf("插假追溯边: %v", err)
	}
	bogusSha := "bogus" + run
	if err := q.InsertCommitClause(ctx, store.InsertCommitClauseParams{
		ProjectID:         string(project),
		CommitSha:         bogusSha,
		ClauseContentHash: h1,
	}); err != nil {
		t.Fatalf("插假挂钩: %v", err)
	}
	t.Cleanup(func() {
		_, _ = st.Pool().Exec(context.Background(), `DELETE FROM trace_edge WHERE from_key = $1`, "bogus"+run)
		_, _ = st.Pool().Exec(context.Background(), `DELETE FROM commit_clause WHERE commit_sha = $1`, bogusSha)
	})

	diff, err = VerifyIndex(ctx, st, ReindexOptions{RepoPath: dir, ProjectID: project})
	if err != nil {
		t.Fatalf("VerifyIndex(手改后): %v", err)
	}
	if diff.Consistent() {
		t.Error("手改了索引表,VerifyIndex 应报不一致")
	}
	if len(diff.TraceEdgeDiff) == 0 {
		t.Error("应检出假追溯边")
	}
	if len(diff.CommitClauseDiff) == 0 {
		t.Error("应检出假 commit 挂钩")
	}
}

// TestVerifyIndexDetectsForeignProjectEdge:别的项目命名空间里混进一条本推不出来的边
// (四列与真边全同,只有 project 不同),VerifyIndex 的 diff 必须能按命名空间指认,
// 而不是渲染折叠后看漏。检查性重算会把这条不可推导的边删掉(事务内,随即回滚),
// 所以它以「仅现有多余」(-)出现在 diff 里。
func TestVerifyIndexDetectsForeignProjectEdge(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	q := st.Queries()
	run := runID(t)
	project := core.ProjectID("proj-" + run)
	h1 := "sha256:" + run + "-h1"
	seedClauseVersion(t, q, string(project), "REQ-1", h1, 1, base, true)

	dir, repo := initTestRepo(t)
	sha := writeCommit(t, repo, dir, "impl.go", "package x", "feat: 实现\n\nRefs: REQ-1", base.Add(time.Hour))
	if _, err := Reindex(ctx, st, ReindexOptions{RepoPath: dir, ProjectID: project}); err != nil {
		t.Fatalf("reindex: %v", err)
	}

	// 同一条 clause→commit tuple,换个项目命名空间手插一条:四列与真边全同,只有 project 不同。
	foreign := store.TraceEdge{ProjectID: "elsewhere-" + run, FromType: NodeClause, FromKey: h1, ToType: NodeCommit, ToKey: sha}
	if err := q.InsertTraceEdge(ctx, store.InsertTraceEdgeParams(foreign)); err != nil {
		t.Fatalf("插外来项目边: %v", err)
	}
	t.Cleanup(func() {
		_, _ = st.Pool().Exec(context.Background(), `DELETE FROM trace_edge WHERE project_id = $1`, foreign.ProjectID)
	})

	diff, err := VerifyIndex(ctx, st, ReindexOptions{RepoPath: dir, ProjectID: project})
	if err != nil {
		t.Fatalf("VerifyIndex: %v", err)
	}
	if diff.Consistent() {
		t.Error("外来项目命名的边不在重算结果里,VerifyIndex 应报不一致")
	}
	want := "- " + foreign.ProjectID + "|" + foreign.FromType + ":" + foreign.FromKey + "→" + foreign.ToType + ":" + foreign.ToKey
	found := false
	for _, l := range diff.TraceEdgeDiff {
		if l == want {
			found = true
		}
	}
	if !found {
		t.Errorf("diff 应指认外来项目的边 %q, got %v", want, diff.TraceEdgeDiff)
	}
}

// hasEdge 判断追溯图里是否有某条边(完整身份,含 project)。
func hasEdge(t *testing.T, q *store.Queries, want store.TraceEdge) bool {
	t.Helper()
	edges, err := q.ListTraceEdges(context.Background())
	if err != nil {
		t.Fatalf("列追溯边: %v", err)
	}
	for _, e := range edges {
		if e == want {
			return true
		}
	}
	return false
}
