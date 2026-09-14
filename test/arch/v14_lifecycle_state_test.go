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

// P1-09 / DECISION_LOCKS 12 / PRD §14/§22 / L2 Phase 12:v1.4 的生命周期状态面
// 是纯派生读模型,没有任何「直接写状态」的权威入口。本文件把 AC1(_tests
// prevent direct lifecycle state writes)钉成机器检查:
//
//	v14-AC1a taskstate 没有状态写入面:导出函数/方法不得是 Set*/Apply*/Write*
//	         形态;VerifiedBasis 的唯一生产路径是 Derive(经未导出的
//	         projectVerifiedBasis);VerifiedBasis 类型本身没有任何方法面;
//	v14-AC1b 生命周期事实种类里不存在可直接宣告终态的事实(没有
//	         verified/approved 形态的 FactKind):VERIFIED 只能由覆盖门槛推导,
//	         APPROVED 只能由人类审批证据推导(P1-08 结构保证的延续);
//	v14-AC1c 没有第二状态权威:数据库迁移里不存在可变生命周期状态表
//	         (task_execution_state / verified_basis / risk_profile 等),
//	         sqlc 查询面没有任何针对它们的路径(Decision Lock 12 / PRD §22:
//	         risk 与 verified 结论都是派生投影,不是持久化业务对象);
//	v14-AC1d taskstate 保持纯推导:生产依赖只有领域 seam(evidence/verification
//	         与 P1-05/06/07 的 contract/assurance/engineeringpolicy/risk),
//	         不持有持久化、写通道或 runtime 依赖。
//
// 注:v13_state_test.go 的 TestV13StateHasNoDirectWritePath 仍是 P1-08 之前的旧
// allowlist(只允许 evidence/verification),与 Decision Lock 12 冻结的「扩展现有
// taskstate 读模型」冲突;本文件 v14-AC1d 是 v1.4 基线的修正钉。旧清单的修正属
// v13 文件所有者,不在本任务 Allowed paths 内。

// v14ParsePackage 解析 dir 下全部非测试 .go 文件为完整 AST。
func v14ParsePackage(t *testing.T, dir string) []*ast.File {
	t.Helper()
	var out []*ast.File
	for _, file := range goFiles(t, dir, false) {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatalf("解析 %s: %v", file, err)
		}
		out = append(out, f)
	}
	if len(out) == 0 {
		t.Fatalf("%s 下没有非测试 .go 文件", dir)
	}
	return out
}

// v14-AC1a:taskstate 包没有状态写入口;VerifiedBasis 只有一个构造点且只被
// Derive 消费(状态结论只能从事实推导,PRD §18.1 / §14)。
func TestV14LifecycleStateHasNoWriteAuthority(t *testing.T) {
	files := v14ParsePackage(t, filepath.Join(repoInternal, "taskstate"))

	var mutatorRe = regexp.MustCompile(`^(Set|Write|Apply|Save|Store|Update|Insert|Mutate|Overwrite|Rewrite)[A-Z]`)
	var (
		basisConstructorFound bool // projectVerifiedBasis 的定义存在
		deriveConsumesFound   bool // Derive 调用该构造函数
		deriveFound           bool // Derive 本身存在
	)
	for _, f := range files {
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if fd.Recv == nil && fd.Name.Name == "Derive" {
				deriveFound = true
			}
			if fd.Recv == nil && ast.IsExported(fd.Name.Name) && mutatorRe.MatchString(fd.Name.Name) {
				t.Errorf("taskstate 导出函数 %s:生命周期状态不得有直接写入口(AC1/INV-STATE-01)", fd.Name.Name)
			}
			if fd.Recv != nil && ast.IsExported(fd.Name.Name) && mutatorRe.MatchString(fd.Name.Name) {
				t.Errorf("taskstate 导出方法 %s:生命周期状态不得有直接写入口(AC1/INV-STATE-01)", fd.Name.Name)
			}

			// VerifiedBasis 的构造点与消费点审计。
			if fd.Recv == nil && fd.Name.Name == "projectVerifiedBasis" {
				basisConstructorFound = true
			}
			body := fd.Body
			if body == nil {
				continue
			}
			isConstructor := fd.Recv == nil && fd.Name.Name == "projectVerifiedBasis"
			ast.Inspect(body, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.CompositeLit:
					if id, ok := x.Type.(*ast.Ident); ok && id.Name == "VerifiedBasis" && !isConstructor {
						t.Errorf("函数 %s 直接构造 VerifiedBasis:唯一构造点是 projectVerifiedBasis,唯一消费路径是 Derive(AC1/Decision Lock 12)", fd.Name.Name)
					}
				case *ast.CallExpr:
					if id, ok := x.Fun.(*ast.Ident); ok && id.Name == "projectVerifiedBasis" {
						if fd.Name.Name != "Derive" {
							t.Errorf("函数 %s 调用 projectVerifiedBasis:VerifiedBasis 只能由 Derive 产出(AC1/Decision Lock 12)", fd.Name.Name)
						} else {
							deriveConsumesFound = true
						}
					}
				}
				return true
			})
		}
		// VerifiedBasis 类型本身不得有任何方法(没有 setter 面,连只读辅助
		// 也挂在 EvidenceCoverage 等值类型上;投影是纯数据)。
		ast.Inspect(f, func(n ast.Node) bool {
			fd, ok := n.(*ast.FuncDecl)
			if !ok || fd.Recv == nil || len(fd.Recv.List) == 0 {
				return true
			}
			if v14RecvTypeName(fd.Recv) == "VerifiedBasis" {
				t.Errorf("VerifiedBasis 声明了方法 %s:投影是纯派生只读数据,不得携带任何方法面(AC1)", fd.Name.Name)
			}
			return true
		})
	}
	if !basisConstructorFound || !deriveConsumesFound || !deriveFound {
		t.Fatalf("Derive=%v projectVerifiedBasis=%v Derive 内消费=%v:VerifiedBasis 的唯一生产路径断了(AC1)", deriveFound, basisConstructorFound, deriveConsumesFound)
	}
}

// v14-AC1b:FactKind 枚举里不存在可直接宣告终态的事实种类 —— 没有
// verified/approved 形态的生命周期事实,Agent/调用方就没有可写的终态旁路。
func TestV14NoTerminalStateFactKinds(t *testing.T) {
	files := v14ParsePackage(t, filepath.Join(repoInternal, "taskstate"))
	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			spec, ok := n.(*ast.ValueSpec)
			if !ok || spec.Type == nil {
				return true
			}
			id, ok := spec.Type.(*ast.Ident)
			if !ok || id.Name != "FactKind" {
				return true
			}
			for i, name := range spec.Names {
				lowerName := strings.ToLower(name.Name)
				var value string
				if i < len(spec.Values) {
					if lit, ok := spec.Values[i].(*ast.BasicLit); ok {
						value = strings.Trim(lit.Value, `"`)
					}
				}
				for _, frag := range []string{"verifie", "approv"} {
					if strings.Contains(lowerName, frag) || strings.Contains(strings.ToLower(value), frag) {
						t.Errorf("生命周期事实种类 %s=%q 含终态词根 %q:终态只能推导,不能作为可写入事实(AC1)", name.Name, value, frag)
					}
				}
			}
			return true
		})
	}
}

// v14StateAuthorityTables 是被禁止持久化为「可变生命周期状态」的表名:
// 状态/验证结论/风险都是派生投影(Decision Lock 12 / PRD §22),不存在
// 第二状态权威表。
var v14StateAuthorityTables = []string{
	"task_execution_state",
	"task_state",
	"verified_basis",
	"risk_profile",
	"risk_signal",
	"lifecycle_state",
}

// v14-AC1c:迁移里没有状态权威表,sqlc 查询面没有任何针对它们的路径。
func TestV14NoSecondStateAuthorityTable(t *testing.T) {
	all := readAllMigrations(t)
	re := regexp.MustCompile(`(?is)CREATE\s+TABLE\s+(IF\s+NOT\s+EXISTS\s+)?(` + strings.Join(v14StateAuthorityTables, "|") + `)\b`)
	if re.MatchString(all) {
		t.Errorf("迁移中出现可变生命周期状态表 %v:状态/结论/风险是派生投影,不得落成第二状态权威(AC1/Decision Lock 12)", v14StateAuthorityTables)
	}
	for _, file := range queryFiles(t) {
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("读 %s: %v", file, err)
		}
		lower := strings.ToLower(string(b))
		for _, tbl := range v14StateAuthorityTables {
			if strings.Contains(lower, tbl) {
				t.Errorf("%s 出现状态权威表 %s 的查询路径:派生投影不经数据库读写(AC1/Decision Lock 12)", file, tbl)
			}
		}
	}
}

// v14TaskstateAllowedImports 是 taskstate 的生产依赖白名单:证据/冻结契约
// 两个 v1.3 权威面,加 P1-05/06/07 的领域 seam(Decision Lock 12:扩展现有
// 读模型)。出现白名单之外的依赖(持久化、写通道、runtime、LLM)即失败。
var v14TaskstateAllowedImports = map[string]bool{
	"evidence":          true,
	"verification":      true,
	"contract":          true,
	"assurance":         true,
	"engineeringpolicy": true,
	"risk":              true,
}

// v14-AC1d:taskstate 只消费领域 seam,不持有持久化/写通道依赖 —— 推导面
// 与任何第二状态存储在依赖图上隔离。标准库不限(纯推导本就只用标准库);
// internal 依赖逐一点名。
func TestV14TaskstateStaysPureDerivation(t *testing.T) {
	for imp := range v14PkgImports(t, filepath.Join(repoInternal, "taskstate")) {
		if !strings.HasPrefix(imp, modulePrefix) {
			continue // 标准库与外部模块不在本检查范围(纯推导不碰持久化驱动)
		}
		pkg := strings.TrimPrefix(imp, modulePrefix)
		pkg = strings.SplitN(pkg, "/", 2)[0]
		if !v14TaskstateAllowedImports[pkg] {
			t.Errorf("internal/taskstate 导入了 internal/%s:状态推导面只允许领域 seam %v(AC1/Decision Lock 12)", pkg, v14TaskstateAllowedImports)
		}
	}
}
