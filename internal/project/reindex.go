package project

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"

	"tally/internal/core"
	"tally/internal/store"
)

// reindex 只重算索引面(派生数据),绝不碰事实面(INV-5):
// 事实表只由系统在执行动作时 INSERT,reindex 读它们来推导,不改一个字节。
// 重算是幂等的:同一输入重算两次结果一致,这也是 CI 定期 diff 的前提。

// ReindexOptions 是一次 reindex 的输入。
type ReindexOptions struct {
	RepoPath  string         // 可选;给了就从该 git 仓库重挂 trailer + 测试注解
	ProjectID core.ProjectID // RepoPath 给了时必填:重挂归属的项目
}

// ReindexReport 是一次 reindex 的产出统计。
type ReindexReport struct {
	CommitHooks int // 重挂的 commit 挂钩数(RepoPath 给了才有)
	TestHooks   int // 重挂的测试挂钩数(RepoPath 给了才有)
	TraceEdges  int // 重算出的追溯边数
}

// reindexWrites 是重算的全部写步骤(在调用方提供的事务内执行)。
// 抽出来让 Reindex(提交)与 VerifyIndex(回滚)共用同一套逻辑,保证「检查用的重算」
// 和「真正执行的重算」永远一致。
func reindexWrites(ctx context.Context, q *store.Queries, opts ReindexOptions) (ReindexReport, error) {
	var rep ReindexReport
	if opts.RepoPath != "" {
		if err := q.DeleteCommitClausesForProject(ctx, string(opts.ProjectID)); err != nil {
			return rep, fmt.Errorf("清 commit 挂钩: %w", err)
		}
		n, err := IndexTrailers(ctx, q, opts.ProjectID, opts.RepoPath)
		if err != nil {
			return rep, err
		}
		rep.CommitHooks = n

		if err := q.DeleteTestClausesForProject(ctx, string(opts.ProjectID)); err != nil {
			return rep, fmt.Errorf("清测试挂钩: %w", err)
		}
		m, err := ScanAnnotations(ctx, q, opts.ProjectID, opts.RepoPath)
		if err != nil {
			return rep, err
		}
		rep.TestHooks = m
	}

	edges, err := RebuildTraceEdges(ctx, q)
	if err != nil {
		return rep, err
	}
	rep.TraceEdges = edges

	// 指纹派生统计从 failure_event 重算;只动统计列,归类/修复列不在其中(见 RebuildFingerprintStats)。
	if err := q.RebuildFingerprintStats(ctx); err != nil {
		return rep, fmt.Errorf("重算指纹统计: %w", err)
	}
	return rep, nil
}

// Reindex 从 git + 事实面重算索引面,全程一个事务:要么全部成功,要么回滚不留半截状态。
// 见 reindexWrites 的步骤。不碰事实面(INV-5)。
func Reindex(ctx context.Context, st *store.Store, opts ReindexOptions) (ReindexReport, error) {
	var rep ReindexReport
	err := st.InTx(ctx, func(q *store.Queries) error {
		r, err := reindexWrites(ctx, q, opts)
		rep = r
		return err
	})
	if err != nil {
		return rep, fmt.Errorf("重算索引面: %w", err)
	}
	return rep, nil
}

// DiffReport 是 reindex --check 的结果:重算会与现有索引数据有哪些不同。
// Consistent() 为 true = 这些派生表与机器记录一致,没人绕过单向数据流。
//
// 只查「纯派生」的表:trace_edge(从索引面+事实面全量重算)与 commit/test 挂钩(从 git 重挂)。
// 不查 fingerprint_stat —— 它带着 failure 包归类写入的 status/fix(非事实面可推导),
// 不是干净的 drop 重算,拿来 diff 会误报;reindex 仍会刷新它的统计列,只是不做一致性断言。
type DiffReport struct {
	TraceEdgeDiff    []string
	CommitClauseDiff []string
	TestClauseDiff   []string
}

// Consistent 报告索引面是否与重算结果完全一致。
func (d DiffReport) Consistent() bool {
	return len(d.TraceEdgeDiff) == 0 && len(d.CommitClauseDiff) == 0 && len(d.TestClauseDiff) == 0
}

// String 把差异渲染成给人看的清单(一致时说明一致)。
func (d DiffReport) String() string {
	if d.Consistent() {
		return "索引面与机器记录一致"
	}
	var b strings.Builder
	for _, section := range []struct {
		name  string
		lines []string
	}{
		{"trace_edge", d.TraceEdgeDiff},
		{"commit_clause", d.CommitClauseDiff},
		{"test_clause", d.TestClauseDiff},
	} {
		if len(section.lines) == 0 {
			continue
		}
		fmt.Fprintf(&b, "%s 不一致(- 现有多余 / + 重算应有):\n", section.name)
		for _, l := range section.lines {
			fmt.Fprintf(&b, "  %s\n", l)
		}
	}
	return b.String()
}

// VerifyIndex 在一个「永不提交」的事务里跑一遍重算,对比重算前后的纯派生表,报告差异。
// 用于 CI 定期检查:有人手改了 trace_edge 或挂钩(绕过单向数据流),重算就对不上。
//
// 不写库(deferred 回滚)。trace_edge 总是检查;commit/test 挂钩仅在给了 RepoPath 时检查
// (那需要 git 才能重算)。fingerprint_stat 不参与 diff(见 DiffReport 说明)。
func VerifyIndex(ctx context.Context, st *store.Store, opts ReindexOptions) (DiffReport, error) {
	var diff DiffReport

	// RepeatableRead 而非默认 read committed:否则每个语句都重读最新已提交的行,
	// 并发事务(共享库并行测试,见 tasks/BACKLOG.md)往 commit/test 挂钩写入时,
	// before 快照 / 重算删除 / after 快照会截到不一致的横截面,造成幻影 diff 误报。
	// 可重复读把整个检查钉在事务开始的快照上,并发提交进不来。trace_edge 不受影响
	// (reindex 总是全量清重建,外来行两边相消);挂钩臂也天然不受本检查自身删除的影响
	// (DeleteCommitClausesForProject 只删本项目 content_hash 的挂钩,外来行碰不到)。
	tx, err := st.Pool().BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return diff, fmt.Errorf("开检查事务: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // 检查从不提交
	q := st.Queries().WithTx(tx)

	beforeEdges, err := q.ListTraceEdges(ctx)
	if err != nil {
		return diff, fmt.Errorf("读现有追溯边: %w", err)
	}
	beforeCommit, beforeTest, err := hookSnapshots(ctx, q, string(opts.ProjectID))
	if err != nil {
		return diff, err
	}

	if _, err := reindexWrites(ctx, q, opts); err != nil {
		return diff, fmt.Errorf("检查性重算: %w", err)
	}

	afterEdges, err := q.ListTraceEdges(ctx)
	if err != nil {
		return diff, fmt.Errorf("读重算追溯边: %w", err)
	}
	afterCommit, afterTest, err := hookSnapshots(ctx, q, string(opts.ProjectID))
	if err != nil {
		return diff, err
	}

	diff.TraceEdgeDiff = sortedDiff(edgeLines(beforeEdges), edgeLines(afterEdges))
	if opts.RepoPath != "" { // 没给 git 就不评挂钩(无从重算,评了是噪声)
		diff.CommitClauseDiff = sortedDiff(beforeCommit, afterCommit)
		diff.TestClauseDiff = sortedDiff(beforeTest, afterTest)
	}
	return diff, nil
}

// hookSnapshots 读某项目的 commit/test 挂钩,渲染成排序无关的字符串行。
// 收窄到项目:挂钩表不记 project_id(INV-4 记内容哈希),借 clause_version 的
// content_hash 归属收窄,让并行共享库里别项目的挂钩不进入本项目的一致性 diff
// (共享库污染,见 tasks/BACKLOG.md);Reindex 的删除本就按项目收窄,快照同口径才对得上。
func hookSnapshots(ctx context.Context, q *store.Queries, projectID string) (commit, test []string, err error) {
	commits, err := q.ListCommitClausesForProject(ctx, projectID)
	if err != nil {
		return nil, nil, fmt.Errorf("读 commit 挂钩: %w", err)
	}
	for _, c := range commits {
		commit = append(commit, c.CommitSha+"→"+c.ClauseContentHash)
	}
	tests, err := q.ListTestClausesForProject(ctx, projectID)
	if err != nil {
		return nil, nil, fmt.Errorf("读测试挂钩: %w", err)
	}
	for _, t := range tests {
		test = append(test, t.TestName+"→"+t.ClauseContentHash)
	}
	return commit, test, nil
}

// edgeLines 把追溯边渲染成字符串行。边身份含 project_id(0015,§22),渲染带上,
// 否则不同项目里仅命名空间不同的两条边会渲染成同一行,diff 数不对、人也看不出归属。
func edgeLines(edges []store.TraceEdge) []string {
	lines := make([]string, 0, len(edges))
	for _, e := range edges {
		lines = append(lines, e.ProjectID+"|"+e.FromType+":"+e.FromKey+"→"+e.ToType+":"+e.ToKey)
	}
	return lines
}

// sortedDiff 返回两个字符串多重集的对称差,前缀 - (仅 before 有) / + (仅 after 有),排序输出。
func sortedDiff(before, after []string) []string {
	bSet := countSet(before)
	aSet := countSet(after)
	var diffs []string
	for s, n := range bSet {
		for i := aSet[s]; i < n; i++ {
			diffs = append(diffs, "- "+s)
		}
	}
	for s, n := range aSet {
		for i := bSet[s]; i < n; i++ {
			diffs = append(diffs, "+ "+s)
		}
	}
	sort.Strings(diffs)
	return diffs
}

// countSet 统计每个元素出现次数(处理多重集,虽然这些表里元素都唯一)。
func countSet(items []string) map[string]int {
	m := make(map[string]int, len(items))
	for _, s := range items {
		m[s]++
	}
	return m
}
