package arch

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"
)

// INV-2 的机器检查(docs/02-invariants.md):未验证的东西进不了知识库。
//
// 两条断言:
//
//  1. failure 里产生验证证据的具体类型,字段全部不导出 ——
//     产生者之外的包构造不出带内容的证据(编译期保证)。
//  2. knowledge 的写库方法签名要求 VerificationProof 参数 ——
//     没有证据的调用编不过。

// proofMethodSet 是验证证据类型必须具备的方法(与 core.VerificationProof 同形)。
var proofMethodSet = []string{"VerifiedAt", "VerifiedCombo", "GateRecord", "Valid"}

// TestProofFieldsUnexported 找出 failure 里实现验证证据的类型(带全部证据方法),
// 断言它是结构体且字段全部不导出。字段一旦导出,包外就能伪造「已验证」证据,
// 铁律 2 失效 —— 所以这是架构级断言。
func TestProofFieldsUnexported(t *testing.T) {
	dir := filepath.Join(repoInternal, "failure")
	methods := map[string]map[string]bool{}
	structs := map[string]*ast.StructType{}
	for _, file := range goFiles(t, dir, false) {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatalf("解析 %s: %v", file, err)
		}
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				name := receiverTypeName(d)
				if name == "" {
					continue
				}
				if methods[name] == nil {
					methods[name] = map[string]bool{}
				}
				methods[name][d.Name.Name] = true
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					if st, ok := ts.Type.(*ast.StructType); ok {
						structs[ts.Name.Name] = st
					}
				}
			}
		}
	}

	var proofTypes []string
	for typ, ms := range methods {
		ok := true
		for _, want := range proofMethodSet {
			if !ms[want] {
				ok = false
				break
			}
		}
		if ok {
			proofTypes = append(proofTypes, typ)
		}
	}
	if len(proofTypes) == 0 {
		t.Fatal("internal/failure 缺少验证证据类型(应有 VerifiedAt/VerifiedCombo/GateRecord/Valid 方法)")
	}
	for _, typ := range proofTypes {
		st, ok := structs[typ]
		if !ok {
			t.Errorf("证据类型 %s 不是结构体", typ)
			continue
		}
		for _, field := range st.Fields.List {
			for _, name := range field.Names {
				if ast.IsExported(name.Name) {
					t.Errorf("证据类型 %s 的字段 %s 导出了:包外可伪造证据(INV-2 铁律 2)", typ, name.Name)
				}
			}
		}
	}
}

// receiverTypeName 取方法接收者类型名(支持 T 与 *T);非方法或复杂接收者返回 ""。
func receiverTypeName(fd *ast.FuncDecl) string {
	if fd.Recv == nil || len(fd.Recv.List) != 1 {
		return ""
	}
	switch t := fd.Recv.List[0].Type.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return id.Name
		}
	}
	return ""
}

// TestKnowledgeStoreRequiresProof 断言 knowledge 的写库方法带 VerificationProof 参数。
func TestKnowledgeStoreRequiresProof(t *testing.T) {
	dir := filepath.Join(repoInternal, "knowledge")
	found := false
	for _, file := range goFiles(t, dir, false) {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatalf("解析 %s: %v", file, err)
		}
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Recv == nil || fd.Name.Name != "Store" {
				continue
			}
			for _, param := range fd.Type.Params.List {
				if typeMentions(param.Type, "VerificationProof") {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("internal/knowledge 的写库方法缺少 VerificationProof 参数:未验证的东西进得了库(INV-2 铁律 2)")
	}
}

// typeMentions 判断类型表达式是否提到 name(标识符或选择器尾段)。
func typeMentions(expr ast.Expr, name string) bool {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name == name
	case *ast.SelectorExpr:
		return t.Sel.Name == name
	case *ast.StarExpr:
		return typeMentions(t.X, name)
	}
	return false
}
