//go:build integration

package project

// 集成测试用真实 Postgres(不 mock 数据库,见 docs/05-test-standards.md)与真实 git 仓库
// (go-git 在临时目录 init,不 mock git)。前置:已迁移(make migrate)。

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/jackc/pgx/v5/pgtype"

	"tally/internal/core"
	"tally/internal/store"
)

func openStore(t *testing.T) *store.Store {
	t.Helper()
	return store.TestDB(t) // 每测试独立 schema(docs/05-test-standards.md)
}

// runID 让每次运行的项目 id / 哈希 / 指纹都唯一:事实面只追加清不掉,索引面的条款哈希
// 也会在多次运行间残留,固定值会让断言被上一次运行污染。
func runID(t *testing.T) string {
	t.Helper()
	return "t08" + time.Now().Format("150405.000000000")
}

// seedClauseVersion 登记一个条款版本(索引面)。isCurrent 时顺带把它设为当前版。
func seedClauseVersion(t *testing.T, q *store.Queries, project, clauseID, hash string, versionNo int32, signedAt time.Time, isCurrent bool) {
	t.Helper()
	ctx := context.Background()
	if err := q.InsertClause(ctx, store.InsertClauseParams{ProjectID: project, ClauseID: clauseID}); err != nil {
		t.Fatalf("登记条款 %s: %v", clauseID, err)
	}
	err := q.InsertClauseVersion(ctx, store.InsertClauseVersionParams{
		ProjectID:   project,
		ClauseID:    clauseID,
		ContentHash: hash,
		VersionNo:   versionNo,
		SignedAt:    pgtype.Timestamptz{Time: signedAt, Valid: true},
		IsCurrent:   isCurrent,
	})
	if err != nil {
		t.Fatalf("登记条款版本 %s@%d: %v", clauseID, versionNo, err)
	}
	if isCurrent {
		err := q.SetClauseVersionCurrent(ctx, store.SetClauseVersionCurrentParams{
			ProjectID: project, ClauseID: clauseID, ContentHash: hash,
		})
		if err != nil {
			t.Fatalf("设当前版本 %s: %v", clauseID, err)
		}
	}
}

// seedBuild 记一次构建(事实面),返回 id。artifactDigest 为空表示构建失败无制品。
func seedBuild(t *testing.T, q *store.Queries, project, commitSha, artifactDigest string, at time.Time) int64 {
	t.Helper()
	combo, _ := json.Marshal(core.VersionCombo{"go": "1.23.0"})
	id, err := q.InsertBuildRun(context.Background(), store.InsertBuildRunParams{
		ProjectID:      project,
		CommitSha:      commitSha,
		StartedAt:      pgtype.Timestamptz{Time: at, Valid: true},
		Status:         "failed",
		ArtifactDigest: pgtype.Text{String: artifactDigest, Valid: artifactDigest != ""},
		VersionCombo:   combo,
	})
	if err != nil {
		t.Fatalf("记构建: %v", err)
	}
	return id
}

// seedDeploy 记一次部署(事实面),返回 id。
func seedDeploy(t *testing.T, q *store.Queries, project, artifactDigest string, buildRunID int64, at time.Time) int64 {
	t.Helper()
	combo, _ := json.Marshal(core.VersionCombo{"go": "1.23.0"})
	id, err := q.InsertDeployRun(context.Background(), store.InsertDeployRunParams{
		ProjectID:      project,
		ArtifactDigest: artifactDigest,
		BuildRunID:     pgtype.Int8{Int64: buildRunID, Valid: buildRunID != 0},
		TargetEnv:      "test",
		Tool:           "compose",
		StartedAt:      pgtype.Timestamptz{Time: at, Valid: true},
		Status:         "failed",
		VersionCombo:   combo,
	})
	if err != nil {
		t.Fatalf("记部署: %v", err)
	}
	return id
}

// seedBuildFailure 记一条构建来源的失败事件(事实面)。
func seedBuildFailure(t *testing.T, q *store.Queries, project string, buildRunID int64, fp string, at time.Time) {
	t.Helper()
	combo, _ := json.Marshal(core.VersionCombo{"go": "1.23.0"})
	_, err := q.InsertFailureEvent(context.Background(), store.InsertFailureEventParams{
		ProjectID:           project,
		Source:              "build",
		BuildRunID:          pgtype.Int8{Int64: buildRunID, Valid: true},
		FpExact:             fp,
		FpFuzzy:             fp + "f",
		PipelineVersion:     "1",
		OccurredAt:          pgtype.Timestamptz{Time: at, Valid: true},
		VersionCombo:        combo,
		ReproCommand:        []byte("null"),
		OtherTargetCommands: []byte("[]"),
	})
	if err != nil {
		t.Fatalf("记构建失败: %v", err)
	}
}

// seedDeployFailure 记一条部署来源的失败事件(事实面)。
func seedDeployFailure(t *testing.T, q *store.Queries, project string, deployRunID int64, fp string, at time.Time) {
	t.Helper()
	combo, _ := json.Marshal(core.VersionCombo{"go": "1.23.0"})
	_, err := q.InsertFailureEvent(context.Background(), store.InsertFailureEventParams{
		ProjectID:           project,
		Source:              "deploy",
		DeployRunID:         pgtype.Int8{Int64: deployRunID, Valid: true},
		FpExact:             fp,
		FpFuzzy:             fp + "f",
		PipelineVersion:     "1",
		OccurredAt:          pgtype.Timestamptz{Time: at, Valid: true},
		VersionCombo:        combo,
		ReproCommand:        []byte("null"),
		OtherTargetCommands: []byte("[]"),
	})
	if err != nil {
		t.Fatalf("记部署失败: %v", err)
	}
}

// countFailureEvents 数某项目的失败事件条数(验证 reindex 不碰事实面用)。
func countFailureEvents(t *testing.T, st *store.Store, project string) int {
	t.Helper()
	var n int
	err := st.Pool().QueryRow(context.Background(),
		`SELECT count(*) FROM failure_event WHERE project_id = $1`, project).Scan(&n)
	if err != nil {
		t.Fatalf("数失败事件: %v", err)
	}
	return n
}

// initTestRepo 在临时目录 init 一个真实 git 仓库。
func initTestRepo(t *testing.T) (string, *git.Repository) {
	t.Helper()
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init git 仓库: %v", err)
	}
	return dir, repo
}

// writeCommit 写一个文件并创建提交,返回提交 sha。when 控制作者时间(测「按当时版本绑定」用)。
func writeCommit(t *testing.T, repo *git.Repository, dir, name, content, msg string, when time.Time) string {
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
	h, err := wt.Commit(msg, &git.CommitOptions{Author: sig, Committer: sig, AllowEmptyCommits: true})
	if err != nil {
		t.Fatalf("创建提交: %v", err)
	}
	return h.String()
}
