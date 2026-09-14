// Package arch 是架构约束测试(七条铁律 + INV-8/INV-9 的机器检查)。
//
// 这些测试解析源码、迁移与 go.mod,断言结构性约束 —— 写在文档里的规矩会被忘,
// 写成测试的不会。它们先于单元测试运行(make check)。见 docs/02-invariants.md。
package arch

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const (
	repoInternal   = "../../internal"
	repoMigrations = "../../migrations"
	modulePrefix   = "tally/internal/"
)

// readAllMigrations 拼接全部迁移 .sql 的内容。
func readAllMigrations(t *testing.T) string {
	t.Helper()
	entries, err := os.ReadDir(repoMigrations)
	if err != nil {
		t.Fatalf("读 migrations 目录: %v", err)
	}
	var sb strings.Builder
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(repoMigrations, e.Name()))
		if err != nil {
			t.Fatalf("读 %s: %v", e.Name(), err)
		}
		sb.WriteString("\n")
		sb.Write(b)
	}
	return sb.String()
}

// queryFiles 返回 sqlc 查询文件列表(目录不存在时为空)。
func queryFiles(t *testing.T) []string {
	t.Helper()
	dir := filepath.Join(repoInternal, "store", "queries")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	return files
}

// createTableBody 提取 CREATE TABLE <name> (...) 括号内的列定义块。
func createTableBody(t *testing.T, content, table string) string {
	t.Helper()
	re := regexp.MustCompile(`(?is)CREATE\s+TABLE\s+` + table + `\s*\((.*?)\n\);`)
	m := re.FindStringSubmatch(content)
	if m == nil {
		t.Fatalf("迁移里找不到 CREATE TABLE %s", table)
	}
	return m[1]
}

type column struct{ name, typ string }

// fkColumns 解析列定义块中带 REFERENCES 的单列外键(跳过复合 FOREIGN KEY)。
func fkColumns(body string) []column {
	var cols []column
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		upper := strings.ToUpper(line)
		if !strings.Contains(upper, "REFERENCES") || strings.HasPrefix(upper, "FOREIGN KEY") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		cols = append(cols, column{name: strings.Trim(fields[0], `"`), typ: fields[1]})
	}
	return cols
}

// goFiles 递归收集 dir 下的 .go 文件;includeTests=false 时排除 _test.go。
func goFiles(t *testing.T, dir string, includeTests bool) []string {
	t.Helper()
	var files []string
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		if !includeTests && strings.HasSuffix(path, "_test.go") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files
}
