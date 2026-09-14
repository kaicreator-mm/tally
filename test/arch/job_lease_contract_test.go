package arch

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// P0-01:job 租约/心跳契约(PRD §17.1 Job Lease / §37 P0 correctness)。
// 无真库可用,按本目录惯例做静态契约检查:直接断言 0014_job_lease.sql 与
// internal/store/queries/job.sql 的结构,把「活租约不可回收」这类铁律写进测试。

const (
	jobLeaseMigrationPath = "../../migrations/0014_job_lease.sql"
	jobLeaseQueriesPath   = "../../internal/store/queries/job.sql"
)

func readJobLeaseMigration(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.FromSlash(jobLeaseMigrationPath))
	if err != nil {
		t.Fatalf("读 %s: %v", jobLeaseMigrationPath, err)
	}
	return string(b)
}

func readJobLeaseQueries(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.FromSlash(jobLeaseQueriesPath))
	if err != nil {
		t.Fatalf("读 %s: %v", jobLeaseQueriesPath, err)
	}
	return string(b)
}

// sqlcQuery 按 `-- name: X :kind` 切分查询文件,返回指定查询的语句正文(找不到即 Fatal)。
// sqlc 的文档注释写在 name 行之后,先剥掉开头的注释/空行;语句正文不含 -- 注释,
// 再遇下一行注释即截断,避免混入下一条查询的文档注释。
func sqlcQuery(t *testing.T, content, name string) string {
	t.Helper()
	re := regexp.MustCompile(`(?m)^-- name: (\w+) :(\w+)\s*$`)
	locs := re.FindAllStringSubmatchIndex(content, -1)
	for i, loc := range locs {
		if content[loc[2]:loc[3]] != name {
			continue
		}
		body := content[loc[1]:]
		if i+1 < len(locs) {
			body = content[loc[1]:locs[i+1][0]]
		}
		lines := strings.Split(body, "\n")
		for len(lines) > 0 {
			trimmed := strings.TrimSpace(lines[0])
			if trimmed == "" || strings.HasPrefix(trimmed, "--") {
				lines = lines[1:]
				continue
			}
			break
		}
		body = strings.Join(lines, "\n")
		if idx := strings.Index(body, "\n--"); idx >= 0 {
			body = body[:idx]
		}
		return body
	}
	t.Fatalf("%s: 找不到查询 %s", jobLeaseQueriesPath, name)
	return ""
}

// normSQL 把 SQL 压成小写、单空格分隔,便于谓词匹配。
func normSQL(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

// AC1:job 行能记录租约持有人(worker_id)、到期时间(lease_until)与心跳(heartbeat_at)。
func TestJobLeaseSchemaRecordsLease(t *testing.T) {
	content := readJobLeaseMigration(t)
	if !strings.Contains(content, "-- +goose Up") || !strings.Contains(content, "-- +goose Down") {
		t.Errorf("0014_job_lease.sql 缺少 goose Up/Down 标记")
	}
	for _, col := range []struct{ name, typ string }{
		{"worker_id", "TEXT"},
		{"locked_at", "TIMESTAMPTZ"},
		{"lease_until", "TIMESTAMPTZ"},
		{"heartbeat_at", "TIMESTAMPTZ"},
	} {
		re := regexp.MustCompile(`(?is)ALTER\s+TABLE\s+job\b.*ADD\s+COLUMN\s+` + col.name + `\s+` + col.typ + `\b`)
		if !re.MatchString(content) {
			t.Errorf("迁移缺少 job.%s %s 列(AC1:租约持有人/到期/心跳)", col.name, col.typ)
		}
	}
}

// AC2:认领原语原子(SKIP LOCKED)且认领即写租约、递增 attempts、回带执行所需字段。
func TestJobLeaseClaimAtomicSkipLocked(t *testing.T) {
	claim := normSQL(sqlcQuery(t, readJobLeaseQueries(t), "ClaimNextJob"))
	for _, want := range []string{
		"update job set",
		"status = 'running'",
		"attempts = attempts + 1",
		"worker_id = $1",
		"lease_until = now() + $2::interval",
		"heartbeat_at = now()",
		"for update skip locked",
		"returning id, kind, payload, attempts",
	} {
		if !strings.Contains(claim, want) {
			t.Errorf("ClaimNextJob 缺少 %q(AC2:原子认领即续租)", want)
		}
	}
}

// AC2+AC3:回收原语原子(SKIP LOCKED)、真正退回 pending 并清空租约,
// 且谓词只触碰 stale 行:lease_until IS NULL OR lease_until <= now()。
// 活租约(lease_until > now())不满足该条件,不可能被回收。
func TestJobLeaseReclaimStaleOnly(t *testing.T) {
	content := readJobLeaseQueries(t)
	stale := regexp.MustCompile(`lease_until is null or lease_until <= now\(\)`)
	for _, name := range []string{"ClaimNextJob", "RequeueStaleJobs"} {
		q := normSQL(sqlcQuery(t, content, name))
		if !strings.Contains(q, "for update skip locked") {
			t.Errorf("%s 缺少 FOR UPDATE SKIP LOCKED(AC2:原子认领/回收)", name)
		}
		if !stale.MatchString(q) {
			t.Errorf("%s 的 running 谓词缺少 stale 条件 lease_until IS NULL OR lease_until <= now()(AC3:活租约不可回收)", name)
		}
	}
	requeue := normSQL(sqlcQuery(t, content, "RequeueStaleJobs"))
	for _, want := range []string{
		"status = 'pending'",
		"worker_id = null",
		"lease_until = null",
	} {
		if !strings.Contains(requeue, want) {
			t.Errorf("RequeueStaleJobs 缺少 %q(回收须释放租约归属)", want)
		}
	}
}

// AC3 双保险:活租约谓词(lease_until > now())只允许出现在心跳续租的 WHERE 里;
// 认领/回收谓词一旦出现它,活租约就可能被回收。心跳续租本身必须同时校验
// 属主(worker_id)、状态(running)与租约仍存活,过期后不允许复活。
func TestJobLeaseLiveLeaseOnlyInHeartbeat(t *testing.T) {
	content := readJobLeaseQueries(t)
	for _, name := range []string{"ClaimNextJob", "RequeueStaleJobs"} {
		if strings.Contains(normSQL(sqlcQuery(t, content, name)), "lease_until > now()") {
			t.Errorf("%s 出现 lease_until > now()(AC3:活租约不可被认领/回收谓词触碰)", name)
		}
	}
	hb := normSQL(sqlcQuery(t, content, "HeartbeatJob"))
	for _, want := range []string{
		"lease_until > now()",
		"worker_id = $2",
		"status = 'running'",
	} {
		if !strings.Contains(hb, want) {
			t.Errorf("HeartbeatJob 缺少 %q(续租只在活租约且属主匹配时发生)", want)
		}
	}
}
