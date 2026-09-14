package arch

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// INV-3:状态不给手动改。
// 事实面三张表的 status 不允许有 UPDATE(fingerprint_stat.status 是例外,
// 由 failure 的归类函数写,不由 HTTP 写,见 docs/03-data-model.md)。
func TestNoStatusUpdateQueries(t *testing.T) {
	re := regexp.MustCompile(`(?is)UPDATE\s+(build_run|deploy_run|environment_run)\b[\s\S]*?\bstatus\s*=`)
	for _, file := range queryFiles(t) {
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("读 %s: %v", file, err)
		}
		if re.Match(b) {
			t.Errorf("%s: 对事实面状态字段的 UPDATE 查询(INV-3)", file)
		}
	}
}

// INV-3:HTTP 路由里不允许出现改状态的写路由(形如 PUT/PATCH /tasks/{id}/status)。
func TestNoStatusRoutes(t *testing.T) {
	dir := filepath.Join(repoInternal, "web")
	mutating := map[string]bool{"Post": true, "Put": true, "Patch": true, "Delete": true}
	for _, file := range goFiles(t, dir, false) {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatalf("解析 %s: %v", file, err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || !mutating[sel.Sel.Name] || len(call.Args) == 0 {
				return true
			}
			if pat := litString(call.Args[0]); strings.Contains(pat, "status") {
				t.Errorf("%s: 写路由 %s 指向状态(INV-3:状态不给手动改)", file, pat)
			}
			return true
		})
	}
}

// litString 取字符串字面量的值;非字面量返回 ""。
func litString(expr ast.Expr) string {
	lit, ok := expr.(*ast.BasicLit)
	if !ok {
		return ""
	}
	return strings.Trim(lit.Value, "`\"")
}
