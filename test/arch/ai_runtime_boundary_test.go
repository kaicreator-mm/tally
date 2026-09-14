package arch

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// P0-09 / INV-V14-14 / PRD §28:internal/airuntime 是 Tally 域侧对公共 AI
// Runtime 的薄适配边界。这些测试把边界钉死成机器检查:
//
//	AC1 域只声明 operation/intent/input/budget —— 没有 provider/model/tier;
//	AC2 接口只有 Execute,判定永远在调用方;
//	AC3 Coding Agent 的 workspace/session/patch 语义不进入 AI Runtime 侧,
//	    反向 AgentRuntime/runtimes 也不经 airuntime 做推理;
//	AC4 传输(SDK/net/http)不在本包重建,由消费方 composition root 接线。

const airuntimeDir = "airuntime"

// parseAIRuntimeFiles 解析 internal/airuntime 的非测试 .go 文件。
func parseAIRuntimeFiles(t *testing.T) []*ast.File {
	t.Helper()
	dir := filepath.Join(repoInternal, airuntimeDir)
	files := goFiles(t, dir, false)
	if len(files) == 0 {
		t.Fatalf("%s 下没有非测试 .go 文件", dir)
	}
	var out []*ast.File
	for _, file := range files {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatalf("解析 %s: %v", file, err)
		}
		out = append(out, f)
	}
	return out
}

// AC3/AC4:airuntime 只准导入标准库(不含 net/http)与 tally/internal/logpipe
// (脱敏闸)。禁止:internal/llm(遗留直连供应商路径)、internal/agentruntime、
// runtimes/*、任何外部 SDK —— 传输属于公共 AI Runtime 的 SDK,
// 在消费方 composition root 接线,不在本包手写重建。
func TestAIRuntimeImportBoundary(t *testing.T) {
	for _, f := range parseAIRuntimeFiles(t) {
		for _, imp := range f.Imports {
			p := strings.Trim(imp.Path.Value, `"`)
			switch {
			case p == "tally/internal/logpipe":
				// 脱敏闸,允许。
			case p == "net/http":
				t.Errorf("%s 导入 net/http:传输不得在 airuntime 内重建(AC4),由消费方接 SDK", f.Name)
			case strings.HasPrefix(p, "tally/"):
				t.Errorf("%s 导入 %s:airuntime 不得依赖其他 internal/runtimes 包(AC3)", f.Name, p)
			case strings.Contains(strings.SplitN(p, "/", 2)[0], "."):
				t.Errorf("%s 导入外部包 %s:airuntime 不引入外部 SDK(AC4)", f.Name, p)
			}
		}
	}
}

// airuntimeIdentifiers 收包内全部标识符(类型/函数/方法/字段/常量/变量)
// 与 struct json tag —— 注释不算,文档里解释边界可以用禁词。
func airuntimeIdentifiers(t *testing.T) []string {
	t.Helper()
	var ids []string
	for _, f := range parseAIRuntimeFiles(t) {
		ast.Inspect(f, func(n ast.Node) bool {
			switch decl := n.(type) {
			case *ast.TypeSpec:
				ids = append(ids, decl.Name.Name)
			case *ast.FuncDecl:
				if decl.Recv != nil {
					ids = append(ids, decl.Name.Name)
				} else {
					ids = append(ids, decl.Name.Name)
				}
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

func TestNoForbiddenVocabulary(t *testing.T) {
	forbidden := map[string]string{
		// AC1:供应商/模型/tier 是 Runtime 侧的配置,不是域侧声明。
		"provider":  "provider 不得出现在 airuntime:路由属于公共 AI Runtime(AC1)",
		"model":     "model 不得出现在 airuntime:模型选择属于公共 AI Runtime(AC1)",
		"tier":      "tier 不得出现在 airuntime:tier 由 Runtime 从 rollout/policy 决定(AC1)",
		"baseurl":   "BaseURL 不得出现在 airuntime:端点是传输配置,不是域声明(AC1)",
		"apikey":    "APIKey 不得出现在 airuntime:凭证属于传输层(AC1)",
		"api_key":   "APIKey 不得出现在 airuntime:凭证属于传输层(AC1)",
		"openai":    "供应商名不得出现在 airuntime(AC1)",
		"anthropic": "供应商名不得出现在 airuntime(AC1)",
		// AC3:Coding Agent 的 workspace/session/patch 语义留在 AgentRuntime。
		"workspace": "workspace 语义不得进入 airuntime:它属于 AgentRuntime(AC3)",
		"session":   "session 语义不得进入 airuntime:它属于 AgentRuntime(AC3)",
		"worktree":  "worktree 语义不得进入 airuntime:它属于 AgentRuntime(AC3)",
		"codex":     "Codex 语义不得进入 airuntime:Coding Agent Runtime 与 AI Runtime 是两套系统(AC3)",
	}
	for _, id := range airuntimeIdentifiers(t) {
		lower := strings.ToLower(id)
		// 豁免:契约 TerminationReason 枚举(如 PROVIDER_FAILURE)是 Runtime
		// 报告的"为什么停"事实,镜像冻结契约,不是域侧路由声明(AC1 禁的是声明)。
		isTerminationEnum := strings.HasPrefix(lower, "termination")
		for frag, msg := range forbidden {
			if isTerminationEnum && frag == "provider" {
				continue
			}
			if strings.Contains(lower, frag) {
				t.Errorf("airuntime 标识符/标签 %q 命中禁词: %s", id, msg)
			}
		}
	}
}

// AC2(与 llm 的 INV-1 检查同款):airuntime 导出的接口只能 Execute。
// 出现 Judge/Verify/Score/Evaluate/Check/Validate 之类判定型方法 → 失败。
// Request.Validate 是结构体方法(域侧机械闸,AC2 要求留在 Tally),不算。
func TestAIRuntimeInterfacesOnlyExecute(t *testing.T) {
	forbidden := map[string]bool{
		"judge": true, "verify": true, "score": true,
		"evaluate": true, "check": true, "validate": true,
	}
	for _, f := range parseAIRuntimeFiles(t) {
		ast.Inspect(f, func(n ast.Node) bool {
			iface, ok := n.(*ast.InterfaceType)
			if !ok {
				return true
			}
			for _, m := range iface.Methods.List {
				for _, name := range m.Names {
					if forbidden[strings.ToLower(name.Name)] {
						t.Errorf("airuntime 接口出现判定型方法 %s:AI 产出永远是 Proposal(INV-1/AC2)", name.Name)
					}
				}
			}
			return true
		})
	}
}

// AC3 反向:Coding Agent Runtime(internal/agentruntime、runtimes/*)
// 不得导入 airuntime 做通用推理 —— 两套系统,PRD §28。
func TestAgentRuntimeDoesNotImportAIRuntime(t *testing.T) {
	dirs := []string{
		filepath.Join(repoInternal, "agentruntime"),
		filepath.Join("..", "..", "runtimes"),
	}
	for _, dir := range dirs {
		for _, file := range goFiles(t, dir, true) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, file, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("解析 %s: %v", file, err)
			}
			for _, imp := range f.Imports {
				if strings.Trim(imp.Path.Value, `"`) == "tally/internal/"+airuntimeDir {
					t.Errorf("%s 导入 internal/airuntime:AgentRuntime 与 AI Runtime 是两套系统(PRD §28/AC3)", file)
				}
			}
		}
	}
}
