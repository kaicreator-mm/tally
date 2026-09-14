//go:build integration

package project

import (
	"context"
	"testing"
	"time"

	"tally/internal/core"
	"tally/internal/store"
)

// 基准时刻:条款 v1 签署于 base,升级 v2 于 base+2h;两个提交分别落在升级前后。
var base = time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

// TestBindCommitAtCurrentVersionThen:INV-4 的核心 —— 按「提交当时」的当前版本绑定,绑定后不变。
// 条款升级后,升级前的提交仍绑旧版本的哈希。
func TestBindCommitAtCurrentVersionThen(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	q := st.Queries()
	run := runID(t)
	project := core.ProjectID("proj-" + run)

	h1 := "sha256:" + run + "-v1" // REQ-12 第一版
	h2 := "sha256:" + run + "-v2" // REQ-12 升级后第二版
	t1 := base
	t2 := base.Add(2 * time.Hour)
	seedClauseVersion(t, q, string(project), "REQ-12", h1, 1, t1, true)
	seedClauseVersion(t, q, string(project), "REQ-12", h2, 2, t2, true) // 升级,v2 变当前

	// 提交 sha 也须每次运行唯一:commit_clause 是索引面会跨运行残留,而断言经
	// ListCommitClauses 全表读回按 commit_sha 归并 —— 固定 sha 会被上一次运行的行污染
	// (与 runID 的注释同一条理由)。
	before := "commit-before-" + run
	after := "commit-after-" + run
	// 升级前的提交(T1 之后、T2 之前):应绑 v1。
	if _, err := BindCommit(ctx, q, project, before, "feat: x\n\nRefs: REQ-12", base.Add(1*time.Hour)); err != nil {
		t.Fatalf("绑定升级前提交: %v", err)
	}
	// 升级后的提交:应绑 v2。
	if _, err := BindCommit(ctx, q, project, after, "feat: y\n\nRefs: REQ-12", base.Add(3*time.Hour)); err != nil {
		t.Fatalf("绑定升级后提交: %v", err)
	}

	got := map[string]string{}
	for _, cc := range mustListCommitClauses(t, q) {
		got[cc.CommitSha] = cc.ClauseContentHash
	}
	if got[before] != h1 {
		t.Errorf("升级前提交应绑旧哈希 %s, got %s(INV-4:绑定后不变)", h1, got[before])
	}
	if got[after] != h2 {
		t.Errorf("升级后提交应绑新哈希 %s, got %s", h2, got[after])
	}
}

// TestBindCommitSkipsUnsignedAtTime:引用了当时还没有签署版本的条款 → 跳过,不报错。
func TestBindCommitSkipsUnsignedAtTime(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	q := st.Queries()
	run := runID(t)
	project := core.ProjectID("proj-" + run)

	// 条款在 base+5h 才签署;提交在 base(签署之前)。
	seedClauseVersion(t, q, string(project), "REQ-1", "sha256:"+run+"-v1", 1, base.Add(5*time.Hour), true)
	n, err := BindCommit(ctx, q, project, "commit-early", "feat: x\n\nRefs: REQ-1", base)
	if err != nil {
		t.Fatalf("绑定应成功(跳过未签署条款), got err: %v", err)
	}
	if n != 0 {
		t.Errorf("签署前的提交应绑 0 条, got %d", n)
	}
	// 引用完全不存在的条款:同样跳过。
	n, err = BindCommit(ctx, q, project, "commit-none", "feat: x\n\nRefs: REQ-999", base.Add(6*time.Hour))
	if err != nil || n != 0 {
		t.Errorf("不存在的条款应跳过:n=%d err=%v", n, err)
	}
}

// TestIndexTrailersFromGitRepo:用 go-git 读真实仓库的全部提交,按各提交的作者时间绑定。
func TestIndexTrailersFromGitRepo(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	q := st.Queries()
	run := runID(t)
	project := core.ProjectID("proj-" + run)

	h1 := "sha256:" + run + "-v1"
	h2 := "sha256:" + run + "-v2"
	seedClauseVersion(t, q, string(project), "REQ-12", h1, 1, base, true)
	seedClauseVersion(t, q, string(project), "REQ-12", h2, 2, base.Add(2*time.Hour), true)

	dir, repo := initTestRepo(t)
	// 文件内容带上 run,让 git 提交 sha 每次运行唯一:commit_clause 跨运行残留,断言又经
	// 全表 ListCommitClauses 按 sha 归并,确定性 sha(内容/作者/时间全固定)会被上次运行污染。
	// 两个提交都引用 REQ-12:第一个在升级前(绑 h1),第二个在升级后(绑 h2)。
	sha1 := writeCommit(t, repo, dir, "a.go", "package a // "+run, "feat: 第一版实现\n\nRefs: REQ-12", base.Add(1*time.Hour))
	sha2 := writeCommit(t, repo, dir, "b.go", "package b // "+run, "feat: 升级后实现\n\nRefs: REQ-12", base.Add(3*time.Hour))
	// 一个无 Refs 的提交:不产生挂钩。
	writeCommit(t, repo, dir, "c.go", "package c // "+run, "chore: 无挂钩", base.Add(4*time.Hour))

	n, err := IndexTrailers(ctx, q, project, dir)
	if err != nil {
		t.Fatalf("IndexTrailers: %v", err)
	}
	if n != 2 {
		t.Errorf("应绑 2 条(两个有 Refs 的提交), got %d", n)
	}

	got := map[string]string{}
	for _, cc := range mustListCommitClauses(t, q) {
		got[cc.CommitSha] = cc.ClauseContentHash
	}
	if got[sha1] != h1 {
		t.Errorf("提交1应绑升级前哈希 %s, got %q", h1, got[sha1])
	}
	if got[sha2] != h2 {
		t.Errorf("提交2应绑升级后哈希 %s, got %q", h2, got[sha2])
	}
}

// mustListCommitClauses 读全部 commit_clause(测试辅助)。
func mustListCommitClauses(t *testing.T, q *store.Queries) []store.ListCommitClausesRow {
	t.Helper()
	rows, err := q.ListCommitClauses(context.Background())
	if err != nil {
		t.Fatalf("列 commit_clause: %v", err)
	}
	return rows
}
