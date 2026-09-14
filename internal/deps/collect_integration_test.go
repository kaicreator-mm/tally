//go:build integration

package deps

// 集成测试用真实 Postgres(不 mock 数据库,见 docs/05-test-standards.md)与真实 git 仓库
// (go-git 在临时目录 init,不 mock git)。

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"

	"tally/internal/store"
)

const goModV1 = `module example.com/demo

go 1.23

require (
	github.com/jackc/pgx/v5 v5.5.1
	golang.org/x/sync v0.10.0
)
`

const goModV2 = `module example.com/demo

go 1.23

require (
	github.com/jackc/pgx/v5 v5.6.0
	golang.org/x/sync v0.10.0
	golang.org/x/crypto v0.31.0
)
`

// initRepo 在临时目录 init 一个真实 git 仓库。
func initRepo(t *testing.T) (string, *git.Repository) {
	t.Helper()
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init git 仓库: %v", err)
	}
	return dir, repo
}

// commitFile 写一个文件并创建提交,返回提交 sha;when 控制提交者时间(测 changed_at 用)。
func commitFile(t *testing.T, repo *git.Repository, dir, name, content, msg string, when time.Time) string {
	t.Helper()
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("取工作区: %v", err)
	}
	path := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("建目录: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("写文件: %v", err)
	}
	if _, err := wt.Add(name); err != nil {
		t.Fatalf("暂存: %v", err)
	}
	sig := &object.Signature{Name: "test", Email: "test@example.com", When: when}
	h, err := wt.Commit(msg, &git.CommitOptions{Author: sig, Committer: sig})
	if err != nil {
		t.Fatalf("创建提交: %v", err)
	}
	return h.String()
}

// rows 读某项目的全部 dependency_change(断言用)。
func rows(t *testing.T, st *store.Store, project string) []store.DependencyChange {
	t.Helper()
	rs, err := st.Queries().ListDependencyChangesRecent(context.Background())
	if err != nil {
		t.Fatalf("读依赖变更: %v", err)
	}
	var out []store.DependencyChange
	for _, r := range rs {
		if r.ProjectID == project {
			out = append(out, r)
		}
	}
	return out
}

func TestCollectGoModUpgrade(t *testing.T) {
	st := store.TestDB(t)
	dir, repo := initRepo(t)
	project := "deps-itest-" + fmt.Sprintf("%d", time.Now().UnixNano())

	t0 := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	t1 := t0.Add(24 * time.Hour)
	commitFile(t, repo, dir, "go.mod", goModV1, "init", t0)
	sha := commitFile(t, repo, dir, "go.mod", goModV2, "upgrade pgx, add crypto", t1)

	changed, n, err := Collect(context.Background(), st, Options{ProjectID: project, RepoPath: dir})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if n != 2 { // pgx 升级 + crypto 新增
		t.Fatalf("新行数 = %d,想要 2", n)
	}
	// 变更包列表 = 退休检查的触发集:本次实际新写入的包。
	if len(changed) != 2 || changed[0] != "github.com/jackc/pgx/v5" || changed[1] != "golang.org/x/crypto" {
		t.Errorf("变更包 = %v,想要 [pgx crypto](排序)", changed)
	}

	rs := rows(t, st, project)
	if len(rs) != 2 {
		t.Fatalf("dependency_change 行数 = %d,想要 2", len(rs))
	}
	byPkg := map[string]store.DependencyChange{}
	for _, r := range rs {
		byPkg[r.Package] = r
	}
	pgx, ok := byPkg["github.com/jackc/pgx/v5"]
	if !ok {
		t.Fatalf("缺 pgx 行: %v", rs)
	}
	if !pgx.FromVersion.Valid || pgx.FromVersion.String != "v5.5.1" ||
		!pgx.ToVersion.Valid || pgx.ToVersion.String != "v5.6.0" {
		t.Errorf("pgx 版本 = %v → %v,想要 v5.5.1 → v5.6.0", pgx.FromVersion, pgx.ToVersion)
	}
	if pgx.CommitSha != sha {
		t.Errorf("commit_sha = %s,想要 %s", pgx.CommitSha, sha)
	}
	if !pgx.ChangedAt.Valid || !pgx.ChangedAt.Time.Equal(t1) {
		t.Errorf("changed_at = %v,想要提交时间 %v", pgx.ChangedAt, t1)
	}
	crypto := byPkg["golang.org/x/crypto"]
	if crypto.FromVersion.Valid {
		t.Errorf("新增依赖的 from_version 应为 NULL,得到 %v", crypto.FromVersion)
	}
	if !crypto.ToVersion.Valid || crypto.ToVersion.String != "v0.31.0" {
		t.Errorf("crypto to_version = %v,想要 v0.31.0", crypto.ToVersion)
	}
}

func TestCollectRerunDedup(t *testing.T) {
	st := store.TestDB(t)
	dir, repo := initRepo(t)
	project := "deps-itest-" + fmt.Sprintf("%d", time.Now().UnixNano())

	t0 := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	commitFile(t, repo, dir, "go.mod", goModV1, "init", t0)
	commitFile(t, repo, dir, "go.mod", goModV2, "upgrade", t0.Add(time.Hour))

	if _, _, err := Collect(context.Background(), st, Options{ProjectID: project, RepoPath: dir}); err != nil {
		t.Fatalf("首次 Collect: %v", err)
	}
	// 重跑同一区间(全量):唯一索引去重,不新增行,变更包也为空(不重复触发退休检查)。
	changed, n, err := Collect(context.Background(), st, Options{ProjectID: project, RepoPath: dir})
	if err != nil {
		t.Fatalf("重跑 Collect: %v", err)
	}
	if n != 0 {
		t.Errorf("重跑新行数 = %d,想要 0(去重)", n)
	}
	if len(changed) != 0 {
		t.Errorf("重跑变更包 = %v,想要空(去重后不重复触发)", changed)
	}
	if len(rows(t, st, project)) != 2 {
		t.Errorf("重跑后行数 = %d,想要 2(不变)", len(rows(t, st, project)))
	}
}

func TestCollectFromExclusive(t *testing.T) {
	st := store.TestDB(t)
	dir, repo := initRepo(t)
	project := "deps-itest-" + fmt.Sprintf("%d", time.Now().UnixNano())

	t0 := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	commitFile(t, repo, dir, "go.mod", goModV1, "init", t0)
	from := commitFile(t, repo, dir, "go.mod", goModV2, "upgrade", t0.Add(time.Hour))
	// From 之后再来一次变更:只有它该被采到。
	const goModV3 = `module example.com/demo

go 1.23

require (
	github.com/jackc/pgx/v5 v5.7.0
	golang.org/x/sync v0.10.0
	golang.org/x/crypto v0.31.0
)
`
	commitFile(t, repo, dir, "go.mod", goModV3, "upgrade pgx again", t0.Add(2*time.Hour))

	changed, n, err := Collect(context.Background(), st, Options{ProjectID: project, RepoPath: dir, From: from})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if n != 1 {
		t.Fatalf("新行数 = %d,想要 1(只采 From 之后)", n)
	}
	if len(changed) != 1 || changed[0] != "github.com/jackc/pgx/v5" {
		t.Errorf("变更包 = %v,想要 [pgx]", changed)
	}
	rs := rows(t, st, project)
	if len(rs) != 1 {
		t.Fatalf("行数 = %d,想要 1", len(rs))
	}
	r := rs[0]
	if r.Package != "github.com/jackc/pgx/v5" || r.ToVersion.String != "v5.7.0" {
		t.Errorf("行 = %s → %v,想要 pgx → v5.7.0", r.Package, r.ToVersion)
	}
}

func TestCollectPackageLock(t *testing.T) {
	st := store.TestDB(t)
	dir, repo := initRepo(t)
	project := "deps-itest-" + fmt.Sprintf("%d", time.Now().UnixNano())

	lockV1 := `{
  "name": "demo",
  "lockfileVersion": 3,
  "packages": {
    "": {"name": "demo", "version": "0.0.1"},
    "node_modules/left-pad": {"version": "1.3.0"}
  }
}`
	lockV2 := `{
  "name": "demo",
  "lockfileVersion": 3,
  "packages": {
    "": {"name": "demo", "version": "0.0.1"},
    "node_modules/left-pad": {"version": "1.4.0"}
  }
}`
	t0 := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	commitFile(t, repo, dir, "package-lock.json", lockV1, "init", t0)
	commitFile(t, repo, dir, "package-lock.json", lockV2, "bump left-pad", t0.Add(time.Hour))

	changed, n, err := Collect(context.Background(), st, Options{ProjectID: project, RepoPath: dir})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if n != 1 {
		t.Fatalf("新行数 = %d,想要 1", n)
	}
	if len(changed) != 1 || changed[0] != "left-pad" {
		t.Errorf("变更包 = %v,想要 [left-pad]", changed)
	}
	rs := rows(t, st, project)
	if len(rs) != 1 || rs[0].Package != "left-pad" ||
		rs[0].FromVersion.String != "1.3.0" || rs[0].ToVersion.String != "1.4.0" {
		t.Errorf("行 = %+v,想要 left-pad 1.3.0 → 1.4.0", rs)
	}
}

func TestCollectNoLockfileChange(t *testing.T) {
	st := store.TestDB(t)
	dir, repo := initRepo(t)
	project := "deps-itest-" + fmt.Sprintf("%d", time.Now().UnixNano())

	t0 := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	commitFile(t, repo, dir, "go.mod", goModV1, "init", t0)
	// 改普通文件,锁文件不变 → 无依赖变更。
	commitFile(t, repo, dir, "main.go", "package main\n", "touch main", t0.Add(time.Hour))

	changed, n, err := Collect(context.Background(), st, Options{ProjectID: project, RepoPath: dir})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if n != 0 {
		t.Errorf("新行数 = %d,想要 0(锁文件未变)", n)
	}
	if len(changed) != 0 {
		t.Errorf("变更包 = %v,想要空", changed)
	}
}

func TestCollectMissingArgs(t *testing.T) {
	st := store.TestDB(t)
	if _, _, err := Collect(context.Background(), st, Options{RepoPath: t.TempDir()}); err == nil {
		t.Error("缺 ProjectID 应报错")
	}
	if _, _, err := Collect(context.Background(), st, Options{ProjectID: "p"}); err == nil {
		t.Error("缺 RepoPath 应报错")
	}
}
