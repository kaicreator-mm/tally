package arch

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// INV-1 辅助:llm 包导出的接口只能有 Propose。
// 出现 Judge/Verify/Score/Evaluate/Check/Validate 之类判定型方法名 → 失败。
// 接口形状是最便宜的约束:方法不存在,就没人能调错。
func TestLLMInterfaceOnlyProposes(t *testing.T) {
	forbidden := map[string]bool{
		"judge": true, "verify": true, "score": true,
		"evaluate": true, "check": true, "validate": true,
	}
	dir := filepath.Join(repoInternal, "llm")
	for _, file := range goFiles(t, dir, false) {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatalf("解析 %s: %v", file, err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			iface, ok := n.(*ast.InterfaceType)
			if !ok {
				return true
			}
			for _, m := range iface.Methods.List {
				for _, name := range m.Names {
					if forbidden[strings.ToLower(name.Name)] {
						t.Errorf("llm 接口出现判定型方法 %s:LLM 只能提案(INV-1)", name.Name)
					}
				}
			}
			return true
		})
	}
}
