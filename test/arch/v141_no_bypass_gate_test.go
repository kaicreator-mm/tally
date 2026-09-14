package arch

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// P1-08 / DECISION_LOCKS「Machine enforcement index」:repo-wide compiled-
// authority no-bypass 的机器守卫。P1-06 把验收守卫接进闭环组合根
// (`closedloop.NewLoop`:凡 `LoopConfig.Authority` 非 nil 即恒接线
// CompiledAcceptanceGuard,DL-07/A-03),生产组合根(cmd/worker)自 P1-04 起
// 编译权威必达。本文件把「不存在绕过编译权威的闭环组合」钉成机器检查:
//
//	v141-NB1 仓库内每一个 closedloop.LoopConfig 组合字面量都必须显式设置
//	         Authority 字段 —— 缺席即「验收守卫未绑定」= FinalVerified 单独
//	         授权普通 Accept 的 v1.4 旁路。测试等价组合(test/e2e harness,
//	         P1-08 已对齐 worker 生产组成)与生产组合根一律在守卫范围内;
//	         例外只允许显式列出(文件 + 理由),且例外文件必须真实包含
//	         LoopConfig 字面量(例外清单不得腐化成永久豁免)。
//	v141-NB2 守卫自身必须会响:合成缺 Authority 的组合字面量必须被识别,
//	         带 Authority 的不得误报(负例/正例对照)。
//
// 静态面的边界(如实声明):本守卫钉「组合面声明了权威」;接线级事实
// (编译先于任何 runtime 会话、守卫真的在 accept 转移裁决)由 cmd/worker
// 的生产路径测试(P1-04/P1-06)与本文件配套的 e2e gate
// (test/e2e/v141_no_bypass_gate_test.go)以运行时证据覆盖。

// v141LoopAuthorityAllowistPrefix 是守卫扫描的仓库根(test/arch 相对路径):
// 控制面(internal)、组合根(cmd)、脚本工具(scripts)与测试面(test)。
var v141LoopAuthorityRoots = []string{
	filepath.Join("..", "..", "cmd"),
	filepath.Join("..", "..", "internal"),
	filepath.Join("..", "..", "scripts"),
	filepath.Join("..", "..", "test"),
}

// v141LoopAuthorityAllowlist 是唯一允许 Authority 缺席的文件清单(仓库根相对
// POSIX 路径 → 理由)。新增例外 = 修改本清单并在对应任务报告里给出裁决:
// 静默新增 nil-authority 组合会被 NB1 拒绝。
var v141LoopAuthorityAllowlist = map[string]string{
	// P1-06 守卫自身的语义钉:pin「nil-authority 组合保持 v1.4 验收行为」
	// (TestLoopWithoutAuthorityKeepsLegacyAccept)与 guard 组合负例 ——
	// nil 语义的权威测试就在机制旁边,不属于闭环业务组合。
	"internal/closedloop/acceptguard_test.go": "P1-06 验收守卫机制的 nil-authority 语义钉(守卫自身单元/组合负例)",
	// P2-08 引擎域集成夹具(组合根守卫落地之前的域测试面);nil 语义由
	// acceptguard_test 钉,业务闭环组合一律走 NewLoop + Authority。
	"internal/closedloop/integration_test.go": "P2-08 引擎域集成夹具(域层测试,非业务闭环组合)",
	// P4-03 DeepSeek Harness trial 工具:手工 trial 驱动器,不在 v1.4.1
	// 发布路径上;P1-08 Allowed Paths(test/**、reports/**)不覆盖 scripts/**,
	// 其 Authority 对齐归 P4-03/P4-07(DSH limited-readiness gate)所有。
	"scripts/v14_trial/tally/trialrun.go": "P4-03 DSH trial 工具(P1-08 Allowed Paths 之外;验收守卫对齐归 P4-03/P4-07 gate)",
}

// v141LoopConfigMissingAuthority 解析一个源文件,返回缺 Authority 的
// closedloop.LoopConfig 组合字面量个数与总数(守卫检测器;NB2 对它做负例/正例
// 对照)。匹配口径:选择器名 LoopConfig(导入别名不构成绕过:换个包名造第二个
// 组合根同样会被本守卫点名),外加 closedloop 包内文件的裸标识符形态
// (`LoopConfig{...}` —— 包内测试/composition 与类型同包,无选择器)。
func v141LoopConfigMissingAuthority(f *ast.File, inClosedloopPackage bool) (missing, total int) {
	isLoopConfigType := func(expr ast.Expr) bool {
		switch t := expr.(type) {
		case *ast.SelectorExpr:
			return t.Sel.Name == "LoopConfig"
		case *ast.Ident:
			return inClosedloopPackage && t.Name == "LoopConfig"
		}
		return false
	}
	ast.Inspect(f, func(n ast.Node) bool {
		lit, ok := n.(*ast.CompositeLit)
		if !ok || !isLoopConfigType(lit.Type) {
			return true
		}
		total++
		for _, elt := range lit.Elts {
			if kv, ok := elt.(*ast.KeyValueExpr); ok {
				if id, ok := kv.Key.(*ast.Ident); ok && id.Name == "Authority" {
					return true
				}
			}
		}
		missing++
		return true
	})
	return missing, total
}

// v141RepoRel 把文件路径折算成仓库根相对 POSIX 路径(allowlist 键形态)。
func v141RepoRel(path string) string {
	return strings.TrimPrefix(filepath.ToSlash(path), "../../")
}

// v141-NB1:全仓库扫描 —— 每个 closedloop.LoopConfig 字面量要么显式绑定
// Authority,要么其文件在显式例外清单里(理由在案);例外文件必须真实包含
// LoopConfig 字面量(清单不腐化)。
func TestV141EveryLoopCompositionBindsCompiledAuthority(t *testing.T) {
	type hit struct {
		path    string
		missing int
		total   int
	}
	allowlisted := map[string]bool{}
	seen := map[string]int{} // path → LoopConfig 字面量总数(例外清单活性检查用)

	for _, root := range v141LoopAuthorityRoots {
		files := goFiles(t, root, true)
		if len(files) == 0 {
			t.Fatalf("扫描范围 %s 下没有 .go 文件(夹具失真)", root)
		}
		for _, path := range files {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				t.Fatalf("解析 %s: %v", path, err)
			}
			rel := v141RepoRel(path)
			missing, total := v141LoopConfigMissingAuthority(f, strings.HasPrefix(rel, "internal/closedloop/"))
			seen[rel] = total
			if total == 0 {
				continue
			}
			reason, ok := v141LoopAuthorityAllowlist[rel]
			if !ok {
				if missing > 0 {
					t.Errorf("%s 的 closedloop.LoopConfig 组合缺 Authority(nozzle-bypass,P1-06/DL-07): %d/%d 个字面量未绑定编译权威;新增组合必须显式绑定,例外交由 %s 清单裁决",
						rel, missing, total, "v141LoopAuthorityAllowlist")
				}
				continue
			}
			_ = reason
			allowlisted[rel] = true
		}
	}

	// 例外清单不得腐化:清单里每个文件都必须真实存在且包含 LoopConfig 字面量
	//(文件删除/重构后遗留的死例外会掩盖新的旁路)。
	for rel := range v141LoopAuthorityAllowlist {
		if !allowlisted[rel] {
			if seen[rel] == 0 {
				t.Errorf("例外清单条目 %s 已不再包含任何 closedloop.LoopConfig 字面量(死例外,从清单移除)", rel)
			} else {
				t.Errorf("例外清单条目 %s 在扫描范围内未被命中(路径漂移?)", rel)
			}
		}
	}
}

// v141-NB2:守卫必须会响 —— 合成的缺 Authority 组合被识别,带 Authority 的
// 组合不误报;别名导入与包内裸标识符形态都不构成绕过。
func TestV141LoopAuthorityGuardDetectsViolations(t *testing.T) {
	parse := func(src string) *ast.File {
		t.Helper()
		f, err := parser.ParseFile(token.NewFileSet(), "synthetic.go", src, 0)
		if err != nil {
			t.Fatalf("解析合成源码: %v", err)
		}
		return f
	}

	bypass := parse(`package p
func f() {
	_ = closedloop.LoopConfig{Codegen: nil, Verify: nil, Contract: nil, Runtime: "x"}
}
`)
	missing, total := v141LoopConfigMissingAuthority(bypass, false)
	if missing != 1 || total != 1 {
		t.Fatalf("守卫未识别缺 Authority 的组合字面量: missing=%d total=%d(负例:守卫必须会响)", missing, total)
	}

	wired := parse(`package p
func f() {
	_ = closedloop.LoopConfig{Codegen: nil, Verify: nil, Contract: nil, Runtime: "x", Authority: compiled}
}
`)
	if missing, total = v141LoopConfigMissingAuthority(wired, false); missing != 0 || total != 1 {
		t.Fatalf("守卫误报带 Authority 的组合: missing=%d total=%d", missing, total)
	}

	// 换导入别名不构成绕过:选择器名 LoopConfig 一律纳入。
	aliased := parse(`package p
func f() {
	_ = cl.LoopConfig{Runtime: "x"}
}
`)
	if missing, total = v141LoopConfigMissingAuthority(aliased, false); missing != 1 || total != 1 {
		t.Fatalf("别名导入绕过守卫: missing=%d total=%d(换名不换形也算旁路)", missing, total)
	}

	// closedloop 包内文件的裸标识符形态(`LoopConfig{...}`,与类型同包)同样
	// 纳入 —— 包内组合不得借无选择器形态绕过。
	inPackage := parse(`package closedloop
func f() {
	_, _ = NewLoop(LoopConfig{Runtime: "x"})
}
`)
	if missing, total = v141LoopConfigMissingAuthority(inPackage, true); missing != 1 || total != 1 {
		t.Fatalf("包内裸标识符形态绕过守卫: missing=%d total=%d", missing, total)
	}
	if missing, total = v141LoopConfigMissingAuthority(inPackage, false); missing != 0 || total != 0 {
		t.Fatalf("包外文件误报裸 LoopConfig 本地类型: missing=%d total=%d(裸形态只在 closedloop 包内计入)", missing, total)
	}
}
