package arch

import (
	"regexp"
	"strings"
	"testing"
)

// INV-4:挂钩记内容哈希,不记逻辑编号。
// 禁止边直接引用逻辑条款 id(条款一改,图还连着,实际全过期且不报错)。
func TestNoFKToLogicalClauseID(t *testing.T) {
	content := readAllMigrations(t)
	if regexp.MustCompile(`(?is)REFERENCES\s+clause\s*\(\s*id\s*\)`).MatchString(content) {
		t.Error("出现 REFERENCES clause(id):挂钩必须记内容哈希,不记逻辑编号(INV-4)")
	}
}

// INV-4:挂钩表(commit_clause / test_clause)的外键列必须以 _hash/_digest 结尾,且为字符型。
// 注:clause_version 指向父条款的复合外键(project_id, clause_id)是归属关系,不是挂钩,不在此列。
func TestEdgeFKsUseContentHash(t *testing.T) {
	content := readAllMigrations(t)
	for _, tbl := range []string{"commit_clause", "test_clause"} {
		body := createTableBody(t, content, tbl)
		for _, col := range fkColumns(body) {
			if !(strings.HasSuffix(col.name, "_hash") || strings.HasSuffix(col.name, "_digest")) {
				t.Errorf("%s 的外键列 %q 不以 _hash/_digest 结尾(INV-4)", tbl, col.name)
			}
			ltyp := strings.ToLower(col.typ)
			if !strings.Contains(ltyp, "text") && !strings.Contains(ltyp, "char") {
				t.Errorf("%s 的外键列 %q 类型 %q 非字符型(INV-4)", tbl, col.name, col.typ)
			}
		}
	}
}
