package arch

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// P0-09/P0-10/P0-11 / DECISION_LOCKS #17 / ADR-0002 / PRD §28:通用推理的消费方
// 迁移完成后,把迁移成果钉成机器检查 —— 直连供应商路径(internal/llm 与任何
// provider/model 路由声明)不得回流到 Domain 包:
//
//	AC1 internal/** 一律不导入 internal/llm(遗留直连供应商路径;
//	    composition root 的过渡适配器是唯一豁免,见下一条);
//	AC2 legacy llm 只能出现在 cmd/** composition root,且必须与 airuntime
//	    适配同现(即只作为被适配的传输,不是独立推理入口);
//	AC3 AI Runtime 消费域包(requirement/harvest/failure/exec)的声明面
//	    (类型/函数/变量/字段/struct tag)不得出现 provider/model/tier/凭证/
//	    供应商名 —— 路由属于公共 AI Runtime,域只声明 operation/intent/
//	    input/budget(ai_runtime_boundary_test.go 反向钉住 airuntime 本包)。

// TestLegacyProviderPathNotImportedByInternalPackages(AC1):internal 下除
// internal/llm 自身外,任何包(含其测试)不得导入遗留直连供应商路径。
func TestLegacyProviderPathNotImportedByInternalPackages(t *testing.T) {
	graph := importGraph(t)
	for pkg, imports := range graph {
		if pkg == "llm" {
			continue // 遗留路径自身;过渡适配器退场后随包一起删除(P0-11 / AC4)
		}
		if imports["llm"] {
			t.Errorf("internal/%s 导入 internal/llm:直连供应商路径不得回流 Domain 包(DECISION_LOCKS #17 / PRD §28)", pkg)
		}
	}
}

// TestLegacyLLMOnlyAtCompositionRootAdapters(AC2):cmd/** composition root
// 里凡是导入 legacy llm 的 main 包,必须同时导入 airuntime —— 即 legacy 路径
// 只允许以「被适配成 airuntime.Executor 的过渡传输」(llmExecutor,ADR-0002
// SDK 接线缺口)存在,不得绕过边界直接发起推理。
func TestLegacyLLMOnlyAtCompositionRootAdapters(t *testing.T) {
	want := filepath.Clean("../../cmd/internal/airuntimelegacy/adapter.go")
	var importers []string
	for _, file := range goFiles(t, "../../cmd", true) {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("解析 %s: %v", file, err)
		}
		for _, imp := range f.Imports {
			if strings.Trim(imp.Path.Value, `"`) == "tally/internal/llm" {
				importers = append(importers, filepath.Clean(file))
			}
		}
	}
	if len(importers) != 1 || importers[0] != want {
		t.Fatalf("legacy internal/llm 必须只由唯一过渡适配器 %s 导入;实际=%v", want, importers)
	}

	// 唯一 legacy adapter 本身必须同时走 provider-neutral airuntime 边界。
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, want, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("解析 %s: %v", want, err)
	}
	hasAIRuntime := false
	for _, imp := range f.Imports {
		if strings.Trim(imp.Path.Value, `"`) == "tally/internal/airuntime" {
			hasAIRuntime = true
		}
	}
	if !hasAIRuntime {
		t.Fatalf("%s 导入 legacy llm 却未导入 airuntime", want)
	}
}

// aiRuntimeConsumerDomains 是已迁到 AI Runtime 边界的通用推理消费域包。
var aiRuntimeConsumerDomains = []string{"requirement", "harvest", "failure", "exec"}

// domainIdentifiers 收一个域包内全部标识符(类型/函数/方法/字段/常量/变量)
// 与 struct tag —— 注释不算,文档里解释边界可以用禁词(仿 airuntimeIdentifiers)。
func domainIdentifiers(t *testing.T, dir string) []string {
	t.Helper()
	var ids []string
	for _, file := range goFiles(t, dir, false) {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatalf("解析 %s: %v", file, err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch decl := n.(type) {
			case *ast.TypeSpec:
				ids = append(ids, decl.Name.Name)
			case *ast.FuncDecl:
				ids = append(ids, decl.Name.Name)
			case *ast.ValueSpec:
				for _, name := range decl.Names {
					ids = append(ids, name.Name)
				}
			case *ast.Field:
				for _, name := range decl.Names {
					ids = append(ids, name.Name)
				}
				if decl.Tag != nil {
					ids = append(ids, strings.Trim(decl.Tag.Value, "`\""))
				}
			}
			return true
		})
	}
	return ids
}

// TestAIConsumerDomainsDeclareNoProviderRouting(AC2):消费域包的声明面不得
// 出现供应商/模型/tier/凭证词汇,也不得再引用 legacy llm 的类型 —— 那意味着
// provider/model 路由回流到了 Domain 包。
func TestAIConsumerDomainsDeclareNoProviderRouting(t *testing.T) {
	forbidden := map[string]string{
		// 供应商/模型/tier/凭证是公共 AI Runtime 侧的路由与传输配置,
		// 不是域侧声明(Request 只有 operation/intent/input/budget + 幂等键)。
		"provider":  "provider 路由不得回到消费域包:属于公共 AI Runtime(AC2)",
		"model":     "model 选择不得回到消费域包:属于公共 AI Runtime(AC2)",
		"tier":      "tier 不得回到消费域包:Runtime 从 rollout/policy 决定(AC2)",
		"apikey":    "凭证不得进入消费域包:凭证属于传输层(AC2)",
		"api_key":   "凭证不得进入消费域包:凭证属于传输层(AC2)",
		"openai":    "供应商名不得进入消费域包(AC2)",
		"anthropic": "供应商名不得进入消费域包(AC2)",
		// 遗留直连路径的类型/接口名:迁移后域内不得再引用。
		"httpclient": "legacy llm 客户端类型不得回到消费域包:一律经 airuntime.Executor 边界(P0-11)",
		"proposer":   "legacy llm.Proposer 不得回到消费域包:一律 airuntime.Executor(P0-11)",
	}
	for _, pkg := range aiRuntimeConsumerDomains {
		dir := filepath.Join(repoInternal, pkg)
		for _, id := range domainIdentifiers(t, dir) {
			lower := strings.ToLower(id)
			for frag, msg := range forbidden {
				if strings.Contains(lower, frag) {
					t.Errorf("internal/%s 标识符/标签 %q 命中禁词: %s", pkg, id, msg)
				}
			}
		}
	}
}
