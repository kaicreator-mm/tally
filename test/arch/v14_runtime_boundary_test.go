package arch

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"tally/internal/engineeringpolicy"
)

// P1-09 / DECISION_LOCKS 17 / PRD §28:L2 要求边界不变量成为机器规则。本文件
// 把两套 runtime 的边界与 EngineeringPolicy 的执行边界钉成机器检查(AC2+AC3):
//
//	v14-AC2a AgentRuntime 与 AI Runtime 互相隔离:internal/airuntime(AI Runtime
//	         侧)不依赖 Coding Agent 侧(agentruntime/runtimes/llm);反向
//	         internal/agentruntime 与 runtimes/** 不导入 airuntime,也不直连
//	         遗留 provider 路径 internal/llm 做通用推理 —— 两套系统,各走各边;
//	v14-AC2b v1.4 trust-core 基础包(contract/engineeringpolicy/assurance/risk/
//	         taskstate)与任何 runtime 无关:信任核心不依赖 agentruntime、
//	         airuntime、llm 或 runtures/*;
//	v14-AC3a EngineeringPolicy 不导入 runtime/provider 执行:导入面只允许
//	         标准库 + YAML 解析 + internal/risk(风险信号模型);禁止
//	         internal/policy(ExecutionPolicy)、agentruntime、airuntime、llm、
//	         runtures/*、net/http、os/exec 与一切外部 SDK;
//	v14-AC3b ExecutionPolicy != EngineeringPolicy(锁 6):两个包互不导入、
//	         不共享任何导出类型名;EngineeringPolicy 的导出类型面不得出现
//	         ExecutionPolicy 的类型/字段词汇;EngineeringPolicy 解析器在
//	         行为面拒绝一切 ExecutionPolicy 字段(runtime/model/retry/网络/
//	         shell/凭据/预算/stop_conditions 不得从后门进入验收策略)。

// v14PkgImports 返回 dir 下非测试 .go 文件的导入路径全集(生产依赖面,
// 不含 _test.go:测试对边界的引用不属于生产边界)。
func v14PkgImports(t *testing.T, dir string) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	for _, file := range goFiles(t, dir, false) {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("解析 %s: %v", file, err)
		}
		for _, imp := range f.Imports {
			out[strings.Trim(imp.Path.Value, `"`)] = true
		}
	}
	return out
}

// v14-AC2a:AgentRuntime 与 AI Runtime 两套系统互不依赖(DECISION_LOCKS 17)。
func TestV14AgentRuntimeAIRuntimeMutuallyIsolated(t *testing.T) {
	forbiddenForAIRuntime := func(p string) bool {
		return p == "tally/internal/agentruntime" ||
			p == "tally/internal/llm" ||
			strings.HasPrefix(p, "tally/runtimes/")
	}
	for p := range v14PkgImports(t, filepath.Join(repoInternal, "airuntime")) {
		if forbiddenForAIRuntime(p) {
			t.Errorf("internal/airuntime 导入 %s:AI Runtime 侧不得依赖 Coding Agent runtime 或直连 provider 路径(AC2/锁 17)", p)
		}
	}

	codingAgentDirs := []string{
		filepath.Join(repoInternal, "agentruntime"),
		filepath.Join("..", "..", "runtimes"),
	}
	for _, dir := range codingAgentDirs {
		for _, file := range goFiles(t, dir, false) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, file, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("解析 %s: %v", file, err)
			}
			for _, imp := range f.Imports {
				switch p := strings.Trim(imp.Path.Value, `"`); p {
				case "tally/internal/airuntime":
					t.Errorf("%s 导入 internal/airuntime:Coding Agent runtime 不得借 AI Runtime 做通用推理(AC2/PRD §28)", file)
				case "tally/internal/llm":
					t.Errorf("%s 导入 internal/llm:Coding Agent runtime 不得直连遗留 provider 推理路径(AC2/锁 17)", file)
				}
			}
		}
	}
}

// v14FoundationPackages 是 v1.4 trust-core 基础包(本任务钉住的各领域包):
// 它们必须与 runtime 选择无关 —— 信任原语的定义路径上不出现任何 runtime。
var v14FoundationPackages = []string{"contract", "engineeringpolicy", "assurance", "risk", "taskstate"}

// v14-AC2b:trust-core 基础包不依赖任何 runtime(两套系统的哪一侧都不依赖)。
func TestV14FoundationPackagesAreRuntimeNeutral(t *testing.T) {
	for _, pkg := range v14FoundationPackages {
		for p := range v14PkgImports(t, filepath.Join(repoInternal, pkg)) {
			switch {
			case p == "tally/internal/agentruntime":
				t.Errorf("internal/%s 导入 internal/agentruntime:trust-core 不得依赖 Coding Agent runtime(AC2)", pkg)
			case p == "tally/internal/airuntime":
				t.Errorf("internal/%s 导入 internal/airuntime:trust-core 不得依赖 AI Runtime 适配层(AC2)", pkg)
			case p == "tally/internal/llm":
				t.Errorf("internal/%s 导入 internal/llm:trust-core 不得依赖遗留 provider 路径(AC2)", pkg)
			case strings.HasPrefix(p, "tally/runtimes/"):
				t.Errorf("internal/%s 导入 %s:trust-core 不得依赖具体 runtime 实现(AC2)", pkg, p)
			}
		}
	}
}

// v14-AC3a:EngineeringPolicy 的导入面只允许 标准库 + yaml + risk。
// 验收/门禁策略是纯值 + 纯函数:解析、哈希与确定性求值,不执行进程、
// 不发网络请求、不触碰任何 runtime/provider(AC3)。
func TestV14EngineeringPolicyImportBoundary(t *testing.T) {
	allowedInternal := map[string]bool{"tally/internal/risk": true}
	allowedExternal := map[string]bool{"gopkg.in/yaml.v3": true}
	for p := range v14PkgImports(t, filepath.Join(repoInternal, "engineeringpolicy")) {
		if p == "tally/internal/policy" {
			t.Errorf("internal/engineeringpolicy 导入 internal/policy:ExecutionPolicy 不得进入 EngineeringPolicy(AC3/锁 6)")
			continue
		}
		firstSeg := strings.SplitN(p, "/", 2)[0]
		isStd := !strings.Contains(firstSeg, ".")
		switch {
		case isStd:
			// 标准库允许;但 runtime/provider 执行形态仍单独点名(防御性):
			if p == "net/http" || p == "os/exec" {
				t.Errorf("internal/engineeringpolicy 导入 %s:验收策略不得自带执行/传输通道(AC3)", p)
			}
		case allowedExternal[p]:
			// YAML 解析是策略格式的组成部分,允许。
		case allowedInternal[p]:
			// 风险信号模型(P1-04)是匹配语义的输入,允许。
		default:
			t.Errorf("internal/engineeringpolicy 导入 %s:超出 标准库+yaml+risk 的边界(AC3:不导入 runtime/provider 执行)", p)
		}
	}
}

// v14-AC3b(结构面):两个策略包互不导入,且不共享任何导出类型名 ——
// 锁 6 的类型层落点。
func TestV14PolicyPackagesShareNoTypes(t *testing.T) {
	epImports := v14PkgImports(t, filepath.Join(repoInternal, "engineeringpolicy"))
	if epImports["tally/internal/policy"] {
		t.Errorf("engineeringpolicy 导入 policy:ExecutionPolicy != EngineeringPolicy(锁 6)")
	}
	polImports := v14PkgImports(t, filepath.Join(repoInternal, "policy"))
	if polImports["tally/internal/engineeringpolicy"] {
		t.Errorf("policy 导入 engineeringpolicy:两个策略包互不依赖(锁 6)")
	}

	exportedTypes := func(dir string) map[string]bool {
		out := map[string]bool{}
		for _, f := range v14ParsePackage(t, dir) {
			ast.Inspect(f, func(n ast.Node) bool {
				if ts, ok := n.(*ast.TypeSpec); ok && ast.IsExported(ts.Name.Name) {
					out[ts.Name.Name] = true
				}
				return true
			})
		}
		return out
	}
	epTypes := exportedTypes(filepath.Join(repoInternal, "engineeringpolicy"))
	polTypes := exportedTypes(filepath.Join(repoInternal, "policy"))
	for name := range epTypes {
		if polTypes[name] {
			t.Errorf("两个策略包共享导出类型名 %s:锁 6 要求两套类型互不复用", name)
		}
	}
}

// v14ExecutionPolicyVocabulary 是 ExecutionPolicy 导出类型/字段的词汇表
// (internal/policy 的声明面)。EngineeringPolicy 的导出类型面出现这些词,
// 即是把运行时执行边界从后门塞进验收策略(锁 6)。
var v14ExecutionPolicyVocabulary = []string{
	"executionpolicy", "networkpolicy", "networkmode", "allowedhosts",
	"shellpolicy", "allowshell",
	"credentialpolicy", "scopedsecrets",
	"retrypolicy", "maxiterations",
	"budgets", "maxtokens", "maxwalltime",
	"escalationpolicy", "afterfailediterations", "targetprofile",
	"stopcondition", "proposal",
}

// v14-AC3b(声明面):EngineeringPolicy 的导出类型与其字段不得复用
// ExecutionPolicy 的类型/字段词汇。解析器的「出现即拒绝」脚手架
// (policyFile/forbiddenFields,均为未导出)刻意不在此列:那是锁 6 的
// 执行点,不是渗透。
func TestV14EngineeringPolicyDeclaresNoExecutionPolicySurface(t *testing.T) {
	files := v14ParsePackage(t, filepath.Join(repoInternal, "engineeringpolicy"))
	for _, f := range files {
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts, _ := spec.(*ast.TypeSpec)
				if ts == nil || !ast.IsExported(ts.Name.Name) {
					continue
				}
				lowerType := strings.ToLower(ts.Name.Name)
				for _, frag := range v14ExecutionPolicyVocabulary {
					if strings.Contains(lowerType, frag) {
						t.Errorf("engineeringpolicy 导出类型 %s 命中 ExecutionPolicy 词汇 %q:锁 6 禁止两套策略类型互相渗透(AC3)", ts.Name.Name, frag)
					}
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}
				for _, field := range st.Fields.List {
					for _, name := range field.Names {
						lowerField := strings.ToLower(name.Name)
						for _, frag := range v14ExecutionPolicyVocabulary {
							if strings.Contains(lowerField, frag) {
								t.Errorf("engineeringpolicy 导出类型 %s 的字段 %s 命中 ExecutionPolicy 词汇 %q(AC3/锁 6)", ts.Name.Name, name.Name, frag)
							}
						}
					}
				}
			}
		}
	}
}

// v14-AC3b(行为面):EngineeringPolicy 解析器必须拒绝一切 ExecutionPolicy
// 字段 —— 运行时资源/网络/shell/retry 语义只能从 internal/policy 的
// ExecutionPolicy 进入系统,不得经策略 YAML 从后门进入验收面(锁 6)。
func TestV14EngineeringPolicyRejectsExecutionPolicyFields(t *testing.T) {
	// 每个字段配一份「其余部分合法」的 gate 策略骨架,单独注入被禁字段。
	inject := func(field string) string {
		return `
id: policy-p109-leak
enforcement: gate
rationale: probe
trigger:
  risk_signals:
    - schema_migration
requires:
  - fresh_database
gate_condition:
  all_of:
    - fresh_database
` + field + "\n"
	}
	// 与 internal/policy 的 ExecutionPolicy 声明面一一对应(决策锁 6 冻结集)。
	fields := []string{
		"max_iterations: 4",
		"token_budget: 180000",
		"wall_budget: 30m",
		"timeout: 30m",
		"network: deny",
		"shell: {allow_shell: true}",
		"credentials: []",
		"retry: {max_iterations: 4}",
		"runtime: codex",
		"model: strong-coding",
		"stop_conditions: [protected_evidence_modified]",
	}
	for _, field := range fields {
		_, err := engineeringpolicy.Parse([]byte(inject(field)))
		if !errors.Is(err, engineeringpolicy.ErrInvalidPolicy) {
			t.Errorf("策略 YAML 携带 ExecutionPolicy 字段 %q 未被拒绝: err = %v(锁 6:字段不得进入验收策略)", field, err)
		}
	}
}
