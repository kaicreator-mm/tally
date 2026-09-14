package arch

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// importGraph 返回 internal 每个顶层包导入了哪些 internal 顶层包。
func importGraph(t *testing.T) map[string]map[string]bool {
	t.Helper()
	entries, err := os.ReadDir(repoInternal)
	if err != nil {
		t.Fatalf("读 internal 目录: %v", err)
	}
	graph := map[string]map[string]bool{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pkg := e.Name()
		imports := map[string]bool{}
		for _, file := range goFiles(t, filepath.Join(repoInternal, pkg), true) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, file, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("解析 %s: %v", file, err)
			}
			for _, imp := range f.Imports {
				p := strings.Trim(imp.Path.Value, `"`)
				if strings.HasPrefix(p, modulePrefix) {
					rest := strings.TrimPrefix(p, modulePrefix)
					imports[strings.SplitN(rest, "/", 2)[0]] = true
				}
			}
		}
		graph[pkg] = imports
	}
	return graph
}

// INV-1:logpipe / knowledge / store 不允许导入 llm。
func TestLLMImportBoundary(t *testing.T) {
	graph := importGraph(t)
	for _, pkg := range []string{"logpipe", "knowledge", "store"} {
		if graph[pkg]["llm"] {
			t.Errorf("internal/%s 导入了 internal/llm,违反 INV-1(LLM 只能提案)", pkg)
		}
	}
}

// INV-1:core 不导入任何 internal 包。
func TestCoreImportsNothingInternal(t *testing.T) {
	graph := importGraph(t)
	for imp := range graph["core"] {
		t.Errorf("internal/core 导入了 internal/%s:core 不得导入任何 internal 包", imp)
	}
}
