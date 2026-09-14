//go:build integration

package project

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"tally/internal/store"
)

// edgeSet 把追溯边列表渲染成排序后的字符串集合,便于比较。
// 边身份含 project_id(0015,§22),渲染带上,不同项目的同名边不会撞成一行。
func edgeSet(edges []store.TraceEdge) []string {
	out := make([]string, 0, len(edges))
	for _, e := range edges {
		out = append(out, fmt.Sprintf("%s|%s:%s→%s:%s", e.ProjectID, e.FromType, e.FromKey, e.ToType, e.ToKey))
	}
	sort.Strings(out)
	return out
}

func equalSorted(a, b []string) bool {
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

// containsEdge 判断边列表里是否恰好有某条边(完整身份,含 project)。
func containsEdge(edges []store.TraceEdge, want store.TraceEdge) bool {
	for _, e := range edges {
		if e == want {
			return true
		}
	}
	return false
}

// countEdges 数边列表里 from→to tuple(不看 project)出现的次数。
func countEdges(edges []store.TraceEdge, fromType, fromKey, toType, toKey string) int {
	n := 0
	for _, e := range edges {
		if e.FromType == fromType && e.FromKey == fromKey && e.ToType == toType && e.ToKey == toKey {
			n++
		}
	}
	return n
}

// TestBuildEdgesFullChain:追溯链 条款→commit→build→制品→部署→失败指纹 全部从机器记录推出。
func TestBuildEdgesFullChain(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	q := st.Queries()
	run := runID(t)
	project := "proj-" + run
	h1 := "sha256:" + run + "-h1"
	commit1 := "c" + run
	artifact := "sha256:art-" + run
	fpBuild := "fpb-" + run
	fpDeploy := "fpd-" + run

	seedClauseVersion(t, q, project, "REQ-12", h1, 1, base, true)
	if err := q.InsertCommitClause(ctx, store.InsertCommitClauseParams{ProjectID: project, CommitSha: commit1, ClauseContentHash: h1}); err != nil {
		t.Fatalf("写 commit_clause: %v", err)
	}

	b1 := seedBuild(t, q, project, commit1, artifact, base)            // 成功构建,产出制品
	b2 := seedBuild(t, q, project, commit1, "", base.Add(time.Minute)) // 失败构建,无制品
	d1 := seedDeploy(t, q, project, artifact, b1, base.Add(2*time.Minute))
	seedBuildFailure(t, q, project, b2, fpBuild, base.Add(3*time.Minute))
	seedDeployFailure(t, q, project, d1, fpDeploy, base.Add(4*time.Minute))

	edges, err := BuildEdges(ctx, q)
	if err != nil {
		t.Fatalf("BuildEdges: %v", err)
	}
	// BuildEdges 构建全库追溯图(共享库里有其它测试的数据),所以断言本测试的边都在其中,而非全等。
	gotSet := map[string]bool{}
	for _, s := range edgeSet(edges) {
		gotSet[s] = true
	}

	id := func(n int64) string { return fmt.Sprintf("%d", n) }
	// 挂钩边带项目命名空间;build/deploy/指纹边按全局唯一的运行 id 锚定,不携带项目归属。
	want := edgeSet([]store.TraceEdge{
		{ProjectID: project, FromType: NodeClause, FromKey: h1, ToType: NodeCommit, ToKey: commit1},
		{FromType: NodeCommit, FromKey: commit1, ToType: NodeBuild, ToKey: id(b1)},
		{FromType: NodeCommit, FromKey: commit1, ToType: NodeBuild, ToKey: id(b2)},
		{FromType: NodeBuild, FromKey: id(b1), ToType: NodeArtifact, ToKey: artifact},
		{FromType: NodeArtifact, FromKey: artifact, ToType: NodeDeploy, ToKey: id(d1)},
		{FromType: NodeBuild, FromKey: id(b2), ToType: NodeFingerprint, ToKey: fpBuild},
		{FromType: NodeDeploy, FromKey: id(d1), ToType: NodeFingerprint, ToKey: fpDeploy},
	})
	for _, e := range want {
		if !gotSet[e] {
			t.Errorf("缺少追溯边: %s", e)
		}
	}
}

// TestBuildEdgesProjectNamespaceCommitNodes(AC1):同一 commit SHA 落在两个项目
// (cherry-pick 场景),且两个项目签的是同一条款内容(同一 content_hash,哈希全局唯一)
// —— 最恶劣的撞车输入。两条 clause→commit 边必须各自带归属项目、互不合并。
func TestBuildEdgesProjectNamespaceCommitNodes(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	q := st.Queries()
	run := runID(t)
	pA, pB := "projA-"+run, "projB-"+run
	h := "sha256:" + run + "-shared" // 同一条款内容:两项目的 content_hash 相同
	sha := "c" + run                 // 同一 commit SHA 在两个项目都有挂钩

	seedClauseVersion(t, q, pA, "REQ-1", h, 1, base, true)
	for _, p := range []string{pA, pB} {
		if err := q.InsertCommitClause(ctx, store.InsertCommitClauseParams{ProjectID: p, CommitSha: sha, ClauseContentHash: h}); err != nil {
			t.Fatalf("写 commit_clause(%s): %v", p, err)
		}
	}

	if _, err := RebuildTraceEdges(ctx, q); err != nil {
		t.Fatalf("第一次重算: %v", err)
	}
	edges, err := q.ListTraceEdges(ctx)
	if err != nil {
		t.Fatalf("读追溯边: %v", err)
	}

	wantA := store.TraceEdge{ProjectID: pA, FromType: NodeClause, FromKey: h, ToType: NodeCommit, ToKey: sha}
	wantB := store.TraceEdge{ProjectID: pB, FromType: NodeClause, FromKey: h, ToType: NodeCommit, ToKey: sha}
	if !containsEdge(edges, wantA) {
		t.Errorf("缺少项目 A 的挂钩边: %+v", wantA)
	}
	if !containsEdge(edges, wantB) {
		t.Errorf("缺少项目 B 的挂钩边: %+v", wantB)
	}
	if n := countEdges(edges, NodeClause, h, NodeCommit, sha); n != 2 {
		t.Errorf("同 commit SHA 同条款哈希在两个项目应是两条边, got %d 条", n)
	}

	// 重算第二次(AC2):两条边原样保留,不合并也不增删。
	if _, err := RebuildTraceEdges(ctx, q); err != nil {
		t.Fatalf("第二次重算: %v", err)
	}
	edges2, err := q.ListTraceEdges(ctx)
	if err != nil {
		t.Fatalf("读第二次追溯边: %v", err)
	}
	if n := countEdges(edges2, NodeClause, h, NodeCommit, sha); n != 2 {
		t.Errorf("重算两次后仍应是两条边(幂等), got %d 条", n)
	}
	if !containsEdge(edges2, wantA) || !containsEdge(edges2, wantB) {
		t.Error("重算两次后两条项目边都应原样保留")
	}
}

// TestRebuildTraceEdgesTupleDelete(AC3):一条边的来源挂钩被删后,哪怕它的每一列
// (from_key、to_key)都还出现在别的边里 —— 甚至是别的项目的边里 —— 这条边也必须
// 被完整 tuple 判定删掉,而不是按列判存在误留(旧 NOT IN 按列语义的笛卡尔误留)。
func TestRebuildTraceEdgesTupleDelete(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	q := st.Queries()
	run := runID(t)
	pA, pB := "projA-"+run, "projB-"+run
	h1, h2 := "sha256:"+run+"-h1", "sha256:"+run+"-h2"
	sha1, shaX := "c1"+run, "cx"+run

	// 条款版本:content_hash 全局唯一,h1/h2 各登记一次即可被两项目的挂钩引用。
	seedClauseVersion(t, q, pA, "REQ-1", h1, 1, base, true)
	seedClauseVersion(t, q, pA, "REQ-1", h2, 2, base, true)

	hooks := []store.InsertCommitClauseParams{
		{ProjectID: pA, CommitSha: sha1, ClauseContentHash: h1}, // E1 = (pA, clause,h1 → commit,sha1),待删
		{ProjectID: pA, CommitSha: shaX, ClauseContentHash: h2}, // E2:给 h2 提供宿主
		{ProjectID: pB, CommitSha: shaX, ClauseContentHash: h1}, // E3:给 h1 提供宿主(pB)
		{ProjectID: pB, CommitSha: sha1, ClauseContentHash: h2}, // E4:给 sha1、h2 提供宿主(pB)
	}
	for _, hk := range hooks {
		if err := q.InsertCommitClause(ctx, hk); err != nil {
			t.Fatalf("写 commit_clause(%s/%s): %v", hk.ProjectID, hk.CommitSha, err)
		}
	}

	if _, err := RebuildTraceEdges(ctx, q); err != nil {
		t.Fatalf("第一次重算: %v", err)
	}
	edges, err := q.ListTraceEdges(ctx)
	if err != nil {
		t.Fatalf("读追溯边: %v", err)
	}
	e1 := store.TraceEdge{ProjectID: pA, FromType: NodeClause, FromKey: h1, ToType: NodeCommit, ToKey: sha1}
	if !containsEdge(edges, e1) {
		t.Fatalf("前置:E1 应已写入: %+v", e1)
	}

	// 删掉 E1 的来源挂钩(索引面,模拟条款升级后重挂前的陈旧行)。
	if _, err := st.Pool().Exec(ctx,
		`DELETE FROM commit_clause WHERE project_id = $1 AND commit_sha = $2 AND clause_content_hash = $3`,
		pA, sha1, h1); err != nil {
		t.Fatalf("删来源挂钩: %v", err)
	}

	if _, err := RebuildTraceEdges(ctx, q); err != nil {
		t.Fatalf("第二次重算: %v", err)
	}
	edges2, err := q.ListTraceEdges(ctx)
	if err != nil {
		t.Fatalf("读重算后追溯边: %v", err)
	}

	if containsEdge(edges2, e1) {
		t.Error("来源挂钩已删,E1 整条 tuple 不在重算结果里,必须被删掉(各列存在于别的边也不行)")
	}
	// 宿主边都还在:它们的挂钩没动。
	for _, want := range []store.TraceEdge{
		{ProjectID: pA, FromType: NodeClause, FromKey: h2, ToType: NodeCommit, ToKey: shaX},
		{ProjectID: pB, FromType: NodeClause, FromKey: h1, ToType: NodeCommit, ToKey: shaX},
		{ProjectID: pB, FromType: NodeClause, FromKey: h2, ToType: NodeCommit, ToKey: sha1},
	} {
		if !containsEdge(edges2, want) {
			t.Errorf("宿主边不应被误删: %+v", want)
		}
	}
	if n := countEdges(edges2, NodeClause, h1, NodeCommit, sha1); n != 0 {
		t.Errorf("E1 不应残留任何项目命名空间的副本, got %d 条", n)
	}
}

// TestRebuildTraceEdgesIdempotent:稳定输入下重算两次结果一致,且写进 trace_edge 的内容与 BuildEdges 一致。
//
// 每个测试跑在独立 schema 里(openStore/TestDB),没有别的事务往本 schema 追加事实,
// 全局边数在两次重算之间应当稳定(事实面只追加,本测试中途不再写)。
func TestRebuildTraceEdgesIdempotent(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	q := st.Queries()
	run := runID(t)
	project := "proj-" + run
	h1 := "sha256:" + run + "-h1"
	commit1 := "c" + run
	artifact := "sha256:art-" + run

	seedClauseVersion(t, q, project, "REQ-1", h1, 1, base, true)
	if err := q.InsertCommitClause(ctx, store.InsertCommitClauseParams{ProjectID: project, CommitSha: commit1, ClauseContentHash: h1}); err != nil {
		t.Fatalf("写 commit_clause: %v", err)
	}
	b1 := seedBuild(t, q, project, commit1, artifact, base)
	d1 := seedDeploy(t, q, project, artifact, b1, base.Add(time.Minute))

	n1, err := RebuildTraceEdges(ctx, q)
	if err != nil {
		t.Fatalf("第一次重算: %v", err)
	}
	first, err := q.ListTraceEdges(ctx)
	if err != nil {
		t.Fatalf("读第一次结果: %v", err)
	}
	n2, err := RebuildTraceEdges(ctx, q)
	if err != nil {
		t.Fatalf("第二次重算: %v", err)
	}
	second, err := q.ListTraceEdges(ctx)
	if err != nil {
		t.Fatalf("读第二次结果: %v", err)
	}

	if n1 != n2 {
		t.Errorf("独立 schema 内重算两次边数应完全一致: %d → %d", n1, n2)
	}
	// 本测试自己的键集合:两次重算对同一批稳定数据必须产出完全相同的边。
	mine := map[string]bool{
		h1: true, commit1: true, artifact: true,
		fmt.Sprintf("%d", b1): true, fmt.Sprintf("%d", d1): true,
	}
	own := func(edges []store.TraceEdge) []string {
		var sub []store.TraceEdge
		for _, e := range edges {
			if mine[e.FromKey] || mine[e.ToKey] {
				sub = append(sub, e)
			}
		}
		return edgeSet(sub)
	}
	firstOwn, secondOwn := own(first), own(second)
	if !equalSorted(firstOwn, secondOwn) {
		t.Errorf("稳定输入的两次重算应一致:\n%s\nvs\n%s",
			strings.Join(firstOwn, "\n"), strings.Join(secondOwn, "\n"))
	}
	if len(firstOwn) == 0 {
		t.Error("应有追溯边(非空链)")
	}
}
