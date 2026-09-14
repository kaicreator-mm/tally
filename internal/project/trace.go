package project

import (
	"context"
	"fmt"
	"strconv"

	"tally/internal/store"
)

// 物化追溯图的节点类型。全部边从机器记录推导,无人工标注字段(INV-3/INV-7)。
//
// 链:条款哈希(clause) → 提交(commit) → 构建(build) → 制品(artifact) → 部署(deploy) → 失败指纹(fingerprint)。
// 构建失败直接挂在 build 上(deploy 失败挂在 deploy 上);环境来源的失败没有 commit 链,不进追溯图。
const (
	NodeClause      = "clause"
	NodeCommit      = "commit"
	NodeBuild       = "build"
	NodeArtifact    = "artifact"
	NodeDeploy      = "deploy"
	NodeFingerprint = "fingerprint"
)

// idKey 把自增 id 转成追溯边的字符串键。
func idKey(id int64) string { return strconv.FormatInt(id, 10) }

// globalNS 是「按全局唯一键锚定」的边所在的项目命名空间。
//
// 边身份自 0015 起带 project_id(§22:同一边 tuple 跨项目不合并),但只有 commit 挂钩行
// (commit_clause)自带归属项目;build/deploy 自增 id 全局唯一,制品摘要与失败指纹又只经
// 由这些 id 的边进入图中 —— 这些 tuple 任何两个项目都推不出同一条,天然不合并。它们的
// 归属项目在当前 store 链接合同(ListBuildRunLinks/ListDeployRunLinks/ListFailureLinks)
// 里取不到,这里不伪造归属,落空命名空间;项目归属由挂钩边(clause→commit)承载。
const globalNS = ""

// BuildEdges 从索引面(commit_clause)与事实面(build_run / deploy_run / failure_event)
// 推导全部追溯边。纯推导,不读任何人工字段;结果去重保序。
//
// 边身份 = (project_id, from, to):条款→commit 边按挂钩行的归属项目命名空间,
// 同一 commit SHA(甚至同一条款哈希)在两个项目是两个 commit 节点、两条边(§22)。
//
// 这是单向数据流的核心:trace_edge 的全部内容都能从上游机器记录重现,
// 所以可以随时 drop 重算(reindex),也能拿来 diff 检测有人绕过了单向数据流。
func BuildEdges(ctx context.Context, q *store.Queries) ([]store.TraceEdge, error) {
	commits, err := q.ListCommitClauses(ctx)
	if err != nil {
		return nil, fmt.Errorf("读 commit 挂钩: %w", err)
	}
	builds, err := q.ListBuildRunLinks(ctx)
	if err != nil {
		return nil, fmt.Errorf("读构建链: %w", err)
	}
	deploys, err := q.ListDeployRunLinks(ctx)
	if err != nil {
		return nil, fmt.Errorf("读部署链: %w", err)
	}
	failures, err := q.ListFailureLinks(ctx)
	if err != nil {
		return nil, fmt.Errorf("读失败链: %w", err)
	}

	seen := map[store.TraceEdge]bool{}
	var edges []store.TraceEdge
	add := func(project, fromType, fromKey, toType, toKey string) {
		e := store.TraceEdge{ProjectID: project, FromType: fromType, FromKey: fromKey, ToType: toType, ToKey: toKey}
		if !seen[e] {
			seen[e] = true
			edges = append(edges, e)
		}
	}

	for _, cc := range commits {
		add(cc.ProjectID, NodeClause, cc.ClauseContentHash, NodeCommit, cc.CommitSha)
	}
	for _, b := range builds {
		add(globalNS, NodeCommit, b.CommitSha, NodeBuild, idKey(b.ID))
		if b.ArtifactDigest.Valid {
			add(globalNS, NodeBuild, idKey(b.ID), NodeArtifact, b.ArtifactDigest.String)
		}
	}
	for _, d := range deploys {
		add(globalNS, NodeArtifact, d.ArtifactDigest, NodeDeploy, idKey(d.ID))
	}
	for _, f := range failures {
		if f.BuildRunID.Valid {
			add(globalNS, NodeBuild, idKey(f.BuildRunID.Int64), NodeFingerprint, f.FpExact)
		}
		if f.DeployRunID.Valid {
			add(globalNS, NodeDeploy, idKey(f.DeployRunID.Int64), NodeFingerprint, f.FpExact)
		}
	}
	return edges, nil
}

// RebuildTraceEdges 从机器记录重算物化追溯图,幂等:缺的边补上、陈旧的边删掉,
// 已有的边不重写(稳态下不写一行)。重算两次结果一致,这也是 reindex --check 能 diff 的前提。
// 返回当前重算出的边数(不是本次改动的行数)。
//
// 只碰索引面(trace_edge);事实面一张表都不读写之外的操作(INV-5)。
// 陈旧判定按完整边 tuple(含 project_id,§22)逐行比对:某条边的各列哪怕都还出现在
// 别的边里,只要整条 tuple 不在重算结果里,就得删(按列各自判存在会笛卡尔式误留)。
// 用「补 + 删陈旧」而非 ClearTraceEdges 全清重写:只读检查(VerifyIndex)在稳态下因此
// 不动全表,不再反复重铺,避免与共享库里并发写入互相污染(见 tasks/BACKLOG.md)。
// 调用方若需原子性,传入事务内的 *store.Queries。
func RebuildTraceEdges(ctx context.Context, q *store.Queries) (int, error) {
	edges, err := BuildEdges(ctx, q)
	if err != nil {
		return 0, err
	}
	projects := make([]string, 0, len(edges))
	fts := make([]string, 0, len(edges))
	fks := make([]string, 0, len(edges))
	tts := make([]string, 0, len(edges))
	tks := make([]string, 0, len(edges))
	for _, e := range edges {
		if err := q.InsertTraceEdge(ctx, store.InsertTraceEdgeParams(e)); err != nil {
			return 0, fmt.Errorf("写追溯边: %w", err)
		}
		projects = append(projects, e.ProjectID)
		fts = append(fts, e.FromType)
		fks = append(fks, e.FromKey)
		tts = append(tts, e.ToType)
		tks = append(tks, e.ToKey)
	}
	if err := q.DeleteStaleTraceEdges(ctx, store.DeleteStaleTraceEdgesParams{
		Column1: projects, Column2: fts, Column3: fks, Column4: tts, Column5: tks,
	}); err != nil {
		return 0, fmt.Errorf("清陈旧追溯边: %w", err)
	}
	return len(edges), nil
}
