package arch

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// P0-02 合同的机器检查(纯静态,不连库):
//
//	§22:所有 project-specific node 必须带 project_id namespace;trace 陈旧边清理
//	必须按完整 tuple 判断,不允许按分列存在性误判。
//
// 约束落在两处:migrations/0015_trace_project_namespace.sql(身份 schema)与
// internal/store/queries/trace.sql(身份读写合同)。

const (
	namespaceMigration = "0015_trace_project_namespace.sql"
)

var (
	// commit_clause / test_clause / trace_edge 的完整身份列(主键序)。
	commitClauseIdentity = []string{"project_id", "commit_sha", "clause_content_hash"}
	testClauseIdentity   = []string{"project_id", "test_name", "clause_content_hash"}
	traceEdgeIdentity    = []string{"project_id", "from_type", "from_key", "to_type", "to_key"}
)

// gooseUpSection 取迁移文件的 +goose Up 段(Down 段合法地包含旧身份的还原语句,不参与断言)。
func gooseUpSection(t *testing.T, file string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoMigrations, file))
	if err != nil {
		t.Fatalf("读 %s: %v", file, err)
	}
	re := regexp.MustCompile(`(?is)--\s*\+goose\s+Up\b(.*?)(?:--\s*\+goose\s+Down|\z)`)
	m := re.FindStringSubmatch(string(b))
	if m == nil {
		t.Fatalf("%s 缺少 +goose Up 段", file)
	}
	return m[1]
}

// latestCreateTableBody 取全部迁移里最后一次 CREATE TABLE <table> 的列定义块
// (派生表会被后续迁移重建,如 0015 重建 trace_edge,必须看最终形状而不是 0002)。
// 只看 +goose Up 段:Down 段合法地还原旧形状,描述的是回滚而不是最终状态。
func latestCreateTableBody(t *testing.T, table string) string {
	t.Helper()
	entries, err := os.ReadDir(repoMigrations)
	if err != nil {
		t.Fatalf("读 migrations 目录: %v", err)
	}
	re := regexp.MustCompile(`(?is)CREATE\s+TABLE\s+` + table + `\s*\((.*?)\n\);`)
	down := regexp.MustCompile(`(?is)--\s*\+goose\s+Down\b.*\z`)
	found := ""
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(repoMigrations, e.Name()))
		if err != nil {
			t.Fatalf("读 %s: %v", e.Name(), err)
		}
		for _, m := range re.FindAllStringSubmatch(down.ReplaceAllLiteralString(string(b), ""), -1) {
			found = m[1]
		}
	}
	if found == "" {
		t.Fatalf("迁移里找不到 CREATE TABLE %s", table)
	}
	return found
}

// namedQueryBody 取 sqlc 查询文件里 -- name: <name> 查询的正文(含注释,到下一个 name 为止)。
func namedQueryBody(t *testing.T, path, name string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读 %s: %v", path, err)
	}
	re := regexp.MustCompile(`(?is)--\s*name:\s*` + name + `\s*:[a-z]+(.*?)(?:--\s*name:|\z)`)
	m := re.FindStringSubmatch(string(b))
	if m == nil {
		t.Fatalf("%s 里找不到查询 %s", path, name)
	}
	return m[1]
}

// stripSQLComments 去掉整行 -- 注释,断言只看真正的 SQL(注释里提到 NOT IN 不算数)。
func stripSQLComments(s string) string {
	var sb strings.Builder
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "--") {
			continue
		}
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	return sb.String()
}

// splitPKColumns 拆主键列清单("a, b, c" → [a b c])。
func splitPKColumns(list string) []string {
	var cols []string
	for _, c := range strings.Split(list, ",") {
		if c = strings.TrimSpace(c); c != "" {
			cols = append(cols, strings.Trim(c, `"`))
		}
	}
	return cols
}

// sameColumns 列清单逐位相等(主键序有意义:project_id 必须打头)。
func sameColumns(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if !strings.EqualFold(got[i], want[i]) {
			return false
		}
	}
	return true
}

// traceQueryPath 返回 internal/store/queries/trace.sql 的路径。
func traceQueryPath(t *testing.T) string {
	t.Helper()
	for _, f := range queryFiles(t) {
		if strings.HasSuffix(f, "trace.sql") {
			return f
		}
	}
	t.Fatal("找不到 internal/store/queries/trace.sql")
	return ""
}

// §22:project-specific node 必须带 project_id namespace(P0-02 AC1)。
// 三张表的最终身份都必须以 project_id 打头:同 commit_sha / test_name / 边 tuple
// 在不同项目不得合并成一行。
func TestTraceIdentityNamespacedByProject(t *testing.T) {
	up := gooseUpSection(t, namespaceMigration)

	// 挂钩表身份经 ALTER 换新:先丢旧主键,再以 project_id 打头重建。
	for _, tbl := range []string{"commit_clause", "test_clause"} {
		if !regexp.MustCompile(`(?is)ALTER\s+TABLE\s+` + tbl + `\s+DROP\s+CONSTRAINT\s+` + tbl + `_pkey\b`).MatchString(up) {
			t.Errorf("%s 旧主键未被替换(缺少 DROP CONSTRAINT %s_pkey)", tbl, tbl)
		}
		re := regexp.MustCompile(`(?is)ALTER\s+TABLE\s+` + tbl + `\s+ADD\s+PRIMARY\s+KEY\s*\(([^)]*)\)`)
		m := re.FindStringSubmatch(up)
		if m == nil {
			t.Fatalf("%s 缺少项目命名空间的新主键", tbl)
		}
		want := commitClauseIdentity
		if tbl == "test_clause" {
			want = testClauseIdentity
		}
		if cols := splitPKColumns(m[1]); !sameColumns(cols, want) {
			t.Errorf("%s 新主键是 (%s),必须是 (%s)(§22 项目命名空间)", tbl, strings.Join(cols, ", "), strings.Join(want, ", "))
		}
	}

	// trace_edge:取最终(0015 重建后)的建表形状。
	body := latestCreateTableBody(t, "trace_edge")
	if !regexp.MustCompile(`(?im)^\s*project_id\s+TEXT\s+NOT\s+NULL\b`).MatchString(body) {
		t.Errorf("trace_edge 缺少 project_id TEXT NOT NULL 列(§22 项目命名空间)")
	}
	pk := regexp.MustCompile(`(?is)PRIMARY\s+KEY\s*\(([^)]*)\)`).FindStringSubmatch(body)
	if pk == nil {
		t.Fatal("trace_edge 找不到 PRIMARY KEY")
	}
	if cols := splitPKColumns(pk[1]); !sameColumns(cols, traceEdgeIdentity) {
		t.Errorf("trace_edge 主键是 (%s),必须是 (%s)(同边 tuple 跨项目不得合并)", strings.Join(cols, ", "), strings.Join(traceEdgeIdentity, ", "))
	}
}

// §22:trace 陈旧边清理必须按完整 tuple 判断,不允许按分列存在性误判(P0-02 AC2)。
// 分列 NOT IN 的存在性来自不同的行,会笛卡尔式误判:该删的边因各列恰好在别的边里出现而被留下。
func TestStaleTraceEdgeDeletionIsTupleBased(t *testing.T) {
	body := stripSQLComments(namedQueryBody(t, traceQueryPath(t), "DeleteStaleTraceEdges"))
	if !regexp.MustCompile(`(?i)\bNOT\s+EXISTS\b`).MatchString(body) {
		t.Error("DeleteStaleTraceEdges 未用 NOT EXISTS 按完整 tuple 比对")
	}
	if regexp.MustCompile(`(?i)\bNOT\s+IN\b`).MatchString(body) {
		t.Error("DeleteStaleTraceEdges 仍含 NOT IN 分列判定(按列存在性误判)")
	}
	for _, col := range traceEdgeIdentity {
		re := regexp.MustCompile(`(?i)\bwant\.` + col + `\s*=\s*te\.` + col + `\b`)
		if !re.MatchString(body) {
			t.Errorf("DeleteStaleTraceEdges 的 tuple 比对缺少 %s(必须整行比对,含 project_id)", col)
		}
	}
	// 身份五列各对应一个数组参数:项目命名空间后是 5 个 text[],不是原来的 4 个。
	arrays := regexp.MustCompile(`\$[0-9]+::text\[\]`).FindAllString(body, -1)
	if len(arrays) != len(traceEdgeIdentity) {
		t.Errorf("DeleteStaleTraceEdges 应接收 %d 个身份列数组,实为 %d 个", len(traceEdgeIdentity), len(arrays))
	}
}

// trace.sql 的读写合同与新身份一致:插入列与冲突目标带全 project_id,
// 读取列把 project_id 读回(P0-02 AC1 的合同面,P0-09 据此再生成 sqlc)。
func TestTraceQueriesMatchNamespacedIdentity(t *testing.T) {
	path := traceQueryPath(t)
	inserts := map[string][]string{
		"InsertCommitClause": commitClauseIdentity,
		"InsertTestClause":   testClauseIdentity,
		"InsertTraceEdge":    traceEdgeIdentity,
	}
	for name, identity := range inserts {
		body := stripSQLComments(namedQueryBody(t, path, name))
		for _, col := range identity {
			re := regexp.MustCompile(`(?i)\b` + col + `\b`)
			if !re.MatchString(body) {
				t.Errorf("%s 缺少身份列 %s", name, col)
			}
		}
		m := regexp.MustCompile(`(?is)ON\s+CONFLICT\s*\(([^)]*)\)`)
		cm := m.FindStringSubmatch(body)
		if cm == nil {
			t.Fatalf("%s 缺少 ON CONFLICT(幂等写入合同)", name)
		}
		if cols := splitPKColumns(cm[1]); !sameColumns(cols, identity) {
			t.Errorf("%s 的冲突目标是 (%s),必须是完整命名空间身份 (%s)", name, strings.Join(cols, ", "), strings.Join(identity, ", "))
		}
	}

	// 读回合同:命名空间身份必须可读回,否则调用方(P0-11)无法构建带项目的边。
	for _, name := range []string{"ListCommitClauses", "ListTestClauses", "ListTraceEdges"} {
		body := stripSQLComments(namedQueryBody(t, path, name))
		m := regexp.MustCompile(`(?is)SELECT\s+(.*?)\bFROM\b`).FindStringSubmatch(body)
		if m == nil {
			t.Fatalf("%s 找不到 SELECT 列表", name)
		}
		if !regexp.MustCompile(`(?i)\bproject_id\b`).MatchString(m[1]) {
			t.Errorf("%s 的 SELECT 列表缺少 project_id(命名空间身份必须可读回)", name)
		}
	}
}

// P0-02 AC3:迁移保留/重建派生索引语义。
// 挂钩表原地回填(数据保留,归属不了则迁移显式失败);trace_edge 整表重建,
// 内容由 reindex 从机器记录重铺(INV-STATE-01 可重建),不在迁移里伪造归属。
func TestNamespaceMigrationPreservesDerivedSemantics(t *testing.T) {
	up := gooseUpSection(t, namespaceMigration)

	dropIdx := strings.Index(up, "DROP TABLE trace_edge")
	createIdx := strings.Index(strings.ToLower(up), "create table trace_edge")
	if dropIdx < 0 || createIdx < 0 || dropIdx > createIdx {
		t.Error("trace_edge 应按新身份整表重建(DROP 在前、CREATE 在后),重建不丢索引语义(仅主键换新)")
	}

	for _, tbl := range []string{"commit_clause", "test_clause"} {
		addIdx := strings.Index(up, "ALTER TABLE "+tbl+" ADD COLUMN project_id")
		updIdx := strings.Index(up, "UPDATE "+tbl)
		nnIdx := strings.Index(up, "ALTER TABLE "+tbl+" ALTER COLUMN project_id SET NOT NULL")
		if addIdx < 0 || updIdx < 0 || nnIdx < 0 || !(addIdx < updIdx && updIdx < nnIdx) {
			t.Errorf("%s 必须按 加列 → UPDATE 回填 → SET NOT NULL 顺序迁移(数据保留,NOT NULL 兜底归属完整性)", tbl)
			continue
		}
		if backfill := up[addIdx:nnIdx]; !strings.Contains(backfill, "clause_version") {
			t.Errorf("%s 的 project_id 回填必须来自 clause_version(content_hash 全局唯一 → 归属唯一)", tbl)
		}
	}
}
