package arch

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// P0-08 合同的机器检查(纯静态,不连库):
//
//	§33.3 Derived Index / §6.1 Git-first Artifact:project_index 是从 Git 项目注册表
//	(.tally/project.yaml)派生的索引面,可整体重建(INV-STATE-01),不是作者源;
//	ghost-project guard:查询合同提供 upsert/rebuild/list/exists,ingest 以 exists
//	判定拒收注册表之外的项目;事实面五张表保持只追加(INV-5),不因本索引变得可改。
//
// 约束落在两处:migrations/0017_project_index.sql(派生索引 schema)与
// internal/store/queries/project_index.sql(派生/守卫读写合同)。

const (
	projectIndexMigrationFile = "0017_project_index.sql"
	projectIndexQueriesPath   = "../../internal/store/queries/project_index.sql"
)

func readProjectIndexQueries(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.FromSlash(projectIndexQueriesPath))
	if err != nil {
		t.Fatalf("读 %s: %v", projectIndexQueriesPath, err)
	}
	return string(b)
}

// AC1:project_index 是派生索引面,不是作者源 ——
// 身份以 project_id 为主键,每行携带派生依据(source_commit:派生自注册表哪个 commit);
// 全库没有任何 append-only 触发器挂在它上面:派生态必须可改可重建(INV-STATE-01),
// 与事实面的只追加约束(0001/0011)刻意相反。
func TestProjectIndexSchemaIsDerived(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoMigrations, projectIndexMigrationFile))
	if err != nil {
		t.Fatalf("读 %s: %v", projectIndexMigrationFile, err)
	}
	content := string(b)
	if !strings.Contains(content, "-- +goose Up") || !strings.Contains(content, "-- +goose Down") {
		t.Errorf("%s 缺少 goose Up/Down 标记", projectIndexMigrationFile)
	}

	body := latestCreateTableBody(t, "project_index")
	if !regexp.MustCompile(`(?im)^\s*project_id\s+TEXT\s+PRIMARY\s+KEY\b`).MatchString(body) {
		t.Error("project_index 缺少 project_id TEXT PRIMARY KEY(注册表声明的项目身份即主键)")
	}
	if !regexp.MustCompile(`(?im)^\s*source_commit\s+TEXT\s+NOT\s+NULL\b`).MatchString(body) {
		t.Error("project_index 缺少 source_commit TEXT NOT NULL(每行必须带派生自注册表 commit 的依据,派生而非作者源)")
	}
	if !regexp.MustCompile(`(?im)^\s*derived_at\s+TIMESTAMPTZ\s+NOT\s+NULL\b`).MatchString(body) {
		t.Error("project_index 缺少 derived_at TIMESTAMPTZ NOT NULL(重建/刷新可追溯)")
	}

	// 派生索引不做事实面那套 append-only 双保险:挂上就不可重建了。
	all := readAllMigrations(t)
	re := regexp.MustCompile(`(?is)CREATE\s+TRIGGER\s+\S+\s+BEFORE\s+UPDATE\s+OR\s+DELETE\s+ON\s+project_index\b`)
	if re.MatchString(all) {
		t.Error("project_index 被挂了 append-only 触发器(派生索引必须可 UPDATE/DELETE 重建,INV-STATE-01)")
	}
}

// AC3:0017 迁移只建派生索引,不碰任何事实面表 ——
// 不出现事实表名(无 ALTER/DROP/改数据),不禁用/删除任何触发器
// (事实面只追加的双保险原样保留)。
func TestProjectIndexMigrationLeavesFactTablesAlone(t *testing.T) {
	up := strings.ToLower(gooseUpSection(t, projectIndexMigrationFile))
	for _, tbl := range factTables {
		if strings.Contains(up, tbl) {
			t.Errorf("0017 迁移的 Up 段出现事实面表 %s(只允许建 project_index,事实面不可变,INV-5)", tbl)
		}
	}
	for _, banned := range []string{"drop trigger", "disable trigger", "enable trigger"} {
		if strings.Contains(up, banned) {
			t.Errorf("0017 迁移的 Up 段出现 %s(事实面 append-only 双保险不可拆)", banned)
		}
	}
}

// AC2:查询合同支持 upsert / rebuild / list / exists 四类语义。
func TestProjectIndexQueriesSupportUpsertRebuildListExists(t *testing.T) {
	content := readProjectIndexQueries(t)

	upsert := normSQL(sqlcQuery(t, content, "UpsertProjectIndex"))
	for _, want := range []string{
		"insert into project_index",
		"on conflict (project_id) do update set",
		"source_commit = excluded.source_commit",
	} {
		if !strings.Contains(upsert, want) {
			t.Errorf("UpsertProjectIndex 缺少 %q(AC2:单项目派生 upsert)", want)
		}
	}

	rebuild := normSQL(sqlcQuery(t, content, "RebuildProjectIndex"))
	for _, want := range []string{
		"insert into project_index",
		"unnest($1::text[])",
		"unnest($2::text[])",
		"on conflict (project_id) do update set",
	} {
		if !strings.Contains(rebuild, want) {
			t.Errorf("RebuildProjectIndex 缺少 %q(AC2:按注册表快照批量重铺)", want)
		}
	}

	// 重建的清陈旧一步:按快照整体比对删除,注册表里移除的项目行必须能清掉,
	// 否则「重建后内容 = 注册表快照」不成立(INV-STATE-01)。
	stale := normSQL(sqlcQuery(t, content, "DeleteStaleProjectIndex"))
	for _, want := range []string{
		"delete from project_index",
		"not exists",
		"unnest($1::text[])",
	} {
		if !strings.Contains(stale, want) {
			t.Errorf("DeleteStaleProjectIndex 缺少 %q(AC2:重建须可清快照之外的行)", want)
		}
	}

	list := normSQL(sqlcQuery(t, content, "ListProjectIndex"))
	if !strings.HasPrefix(list, "select project_id, source_commit, derived_at from project_index") {
		t.Errorf("ListProjectIndex 未读回派生索引列(AC2:list 语义): %q", list)
	}
	if !strings.Contains(list, "order by project_id") {
		t.Error("ListProjectIndex 缺少 ORDER BY project_id(列表序必须确定)")
	}
}

// ghost-project guard(§33.3):exists 原语返回布尔判定,ingest 据此拒收未知项目;
// 查询合同只落在索引面上,任何事实面表都不出现在语句里(AC3:事实面不被改写)。
func TestProjectIndexQueriesGuardTouchesOnlyIndexFace(t *testing.T) {
	content := readProjectIndexQueries(t)

	exists := normSQL(sqlcQuery(t, content, "ProjectIndexExists"))
	for _, want := range []string{
		"select exists",
		"from project_index",
		"where project_id = $1",
	} {
		if !strings.Contains(exists, want) {
			t.Errorf("ProjectIndexExists 缺少 %q(guard 原语:按 project_id 的布尔存在判定)", want)
		}
	}

	body := stripSQLComments(content)
	for _, tbl := range factTables {
		re := regexp.MustCompile(`(?i)\b` + tbl + `\b`)
		if re.MatchString(body) {
			t.Errorf("project_index.sql 的语句出现事实面表 %s(guard 只读派生索引,不改写事实,INV-5)", tbl)
		}
	}
}
