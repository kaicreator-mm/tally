package arch

import (
	"errors"
	"go/ast"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"tally/internal/contract"
	"tally/internal/engineeringpolicy"
	"tally/internal/risk"
	"tally/internal/taskcontract"
	"tally/internal/verification"
)

// P1-09 / PRD §7 / DECISION_LOCKS 5/9:L2 要求不变量成为机器检查。本文件把
// 「Contract 聚合不得取代/覆写底层权威」钉成机器检查(AC4):
//
//	v14-AC4a 聚合是只读 facade:Snapshot/Compiled/Frozen 没有导出字段,导出
//	         方法全部是零参只读访问器(没有 Set*/Replace*/Update* 写入口),
//	         且聚合必须原样透出底层权威(Task/Verification/Contract 可达);
//	v14-AC4b pin 从内容推导,不来自调用方断言:Open 对照内容复核,伪造 pin、
//	         内容漂移、revision 漂移一律拒绝(ErrPinMismatch/ErrTaskMismatch);
//	         同一内容对 ⇒ 同一 snapshot 身份;内容变 ⇒ 身份变;
//	v14-AC4c 聚合透出的底层合同与源合同逐字段一致,且访问器返回深拷贝:
//	         调用方改动手里的拷贝不得影响已钉住的 snapshot(修订 = 新 snapshot);
//	v14-AC4d 编译器不改写权威:Compiled.Pins() == snapshot.Pins(),Bind 前后
//	         snapshot 身份不变,重复 Bind 同一输入得到同一 Compiled 哈希;
//	v14-AC4e requires[] 双重用途一致(PRD §8.3):命中 gate 策略在执行前
//	         RequiresContext 通告的键与执行后 GateKeys() 门禁键同源一致;
//	         guidance 策略的 requires 只出现在通告面,永远进不了门禁面。

// ---- 共享夹具(本文件定义,v14_* 系列测试共用)--------------------------------

// v14TaskYAML 一份能通过 taskcontract.Parse 的标准任务合同。
const v14TaskYAML = `
id: TASK-P109
profile: standard
title: v1.4 foundation invariants
objective: pin the v1.4 foundation architecture invariants as executable tests
requirement_refs:
  - .tally/requirements/REQ-007.md
scope:
  allowed_paths:
    - test/arch/v14_*.go
acceptance:
  - id: AC-01
    statement: lifecycle state is derived only
  - id: AC-02
    statement: unmatched risk fails safe
`

// v14MustTask 解析任务合同 YAML,失败即测试失败。
func v14MustTask(t *testing.T, data string) taskcontract.TaskContract {
	t.Helper()
	c, err := taskcontract.Parse([]byte(data))
	if err != nil {
		t.Fatalf("taskcontract.Parse: %v", err)
	}
	return c
}

// v14MustFrozenSimple 冻结一份最小验证契约(单条款 visible build gate)。
func v14MustFrozenSimple(t *testing.T, taskID string, revision int) *verification.Frozen {
	t.Helper()
	c := &verification.Contract{
		TaskID:   taskID,
		Revision: revision,
		Acceptances: []verification.Acceptance{{
			ID:        "AC-01",
			Statement: "visible build gate passes",
			EvidenceBindings: []verification.Binding{{
				ID:               "B-1",
				AuthoringMode:    verification.AuthoringPreCommit,
				VisibilityPolicy: verification.VisibilityVisible,
				Evidence:         verification.EvidenceRef{Type: verification.EvidenceBuild, Path: "Makefile"},
			}},
		}},
	}
	frozen, err := verification.FreezeContract(c, verification.FreezePolicy{})
	if err != nil {
		t.Fatalf("FreezeContract(%s r%d): %v", taskID, revision, err)
	}
	return frozen
}

// v14MustPolicy 解析 EngineeringPolicy YAML(Parse 会填充内容修订 pin),
// 失败即测试失败。
func v14MustPolicy(t *testing.T, data string) engineeringpolicy.Policy {
	t.Helper()
	p, err := engineeringpolicy.Parse([]byte(data))
	if err != nil {
		t.Fatalf("engineeringpolicy.Parse: %v", err)
	}
	if p.Revision == "" {
		t.Fatal("Parse 未填充内容修订 pin(P1-02 契约)")
	}
	return p
}

// v14MustRisk 从信号 id 构造派生风险集(P1-04 seam)。
func v14MustRisk(t *testing.T, ids ...risk.SignalID) risk.Result {
	t.Helper()
	sigs := make([]risk.Signal, 0, len(ids))
	for _, id := range ids {
		sigs = append(sigs, risk.Signal{ID: id, Basis: []string{"intent:TASK-P109"}})
	}
	r, err := risk.Derive(sigs)
	if err != nil {
		t.Fatalf("risk.Derive: %v", err)
	}
	return r
}

// v14Compile 编译最小已编译合同:snapshot + 策略集 + 风险事实(P1-07 seam)。
func v14Compile(t *testing.T, frozen *verification.Frozen, policies []engineeringpolicy.Policy, r risk.Result) *contract.Compiled {
	t.Helper()
	snap, err := contract.Compile(v14MustTask(t, v14TaskYAML), frozen)
	if err != nil {
		t.Fatalf("contract.Compile: %v", err)
	}
	c, err := contract.Bind(snap, policies, r)
	if err != nil {
		t.Fatalf("contract.Bind: %v", err)
	}
	return c
}

// ---- 静态结构面 ---------------------------------------------------------------

// v14MutatorNameRe 匹配「写入权威」形态的方法名。
var v14MutatorNameRe = regexp.MustCompile(`^(Set|Write|Apply|Save|Store|Update|Insert|Mutate|Overwrite|Rewrite|Amend|Replace)[A-Z]`)

// v14RecvTypeName 取方法接收者的类型名;无接收者返回空串。
func v14RecvTypeName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	switch e := recv.List[0].Type.(type) {
	case *ast.StarExpr:
		if id, ok := e.X.(*ast.Ident); ok {
			return id.Name
		}
	case *ast.Ident:
		return e.Name
	}
	return ""
}

// v14ReadOnlyAuthorityCheck 断言 typeName:无导出字段,导出方法全部零参且
// 不带写入动词,且至少暴露 requiredAccessors 列出的只读访问器(聚合必须
// 原样透出底层权威,不得把它藏成有损投影)。
func v14ReadOnlyAuthorityCheck(t *testing.T, dir, typeName string, requiredAccessors []string) {
	t.Helper()
	files := v14ParsePackage(t, dir)
	typeFound := false
	seenAccessors := map[string]bool{}
	for _, f := range files {
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if v14RecvTypeName(d.Recv) != typeName {
					continue
				}
				name := d.Name.Name
				if !ast.IsExported(name) {
					continue
				}
				if v14MutatorNameRe.MatchString(name) {
					t.Errorf("%s.%s 是写入形态的方法:%s 不得有改写入口(AC4:聚合不改写底层权威)", typeName, name, typeName)
				}
				if d.Type != nil && len(d.Type.Params.List) > 0 {
					t.Errorf("%s.%s 带参数:%s 的导出方法必须是零参只读访问器(AC4)", typeName, name, typeName)
				}
				seenAccessors[name] = true
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok || ts.Name.Name != typeName {
						continue
					}
					st, ok := ts.Type.(*ast.StructType)
					if !ok {
						t.Fatalf("%s 不是 struct,夹具失真", typeName)
					}
					typeFound = true
					for _, field := range st.Fields.List {
						for _, name := range field.Names {
							if ast.IsExported(name.Name) {
								t.Errorf("%s.%s 是导出字段:%s 只能经构造入口与只读访问器暴露(AC4)", typeName, name.Name, typeName)
							}
						}
					}
				}
			}
		}
	}
	if !typeFound {
		t.Fatalf("%s 下找不到类型 %s", dir, typeName)
	}
	for _, want := range requiredAccessors {
		if !seenAccessors[want] {
			t.Errorf("%s 缺少只读访问器 %s:聚合必须原样透出底层权威(AC4)", typeName, want)
		}
	}
}

// v14-AC4a:Contract 聚合(Snapshot/Compiled)与冻结验证契约(Frozen)都是
// 只读权威面:没有导出字段、没有写入口、底层合同可达。
func TestV14ContractAuthoritySurfacesAreReadOnly(t *testing.T) {
	v14ReadOnlyAuthorityCheck(t, filepath.Join(repoInternal, "contract"), "Snapshot",
		[]string{"Task", "TaskID", "Verification", "Pins", "Hash"})
	v14ReadOnlyAuthorityCheck(t, filepath.Join(repoInternal, "contract"), "Compiled",
		[]string{"Snapshot", "Pins", "Hash"})
	v14ReadOnlyAuthorityCheck(t, filepath.Join(repoInternal, "verification"), "Frozen",
		[]string{"Contract", "Hash"})
}

// ---- 行为面:pin 从内容推导,内容是唯一权威 -----------------------------------

// v14-AC4b:Compile 从合同内容重算 pin(不信任调用方断言);同一内容对得到
// 同一 snapshot 身份;Open 拒绝伪造 pin、内容漂移与 revision 漂移。
func TestV14ContractPinsDeriveFromContent(t *testing.T) {
	tc := v14MustTask(t, v14TaskYAML)
	frozen := v14MustFrozenSimple(t, "TASK-P109", 3)

	s, err := contract.Compile(tc, frozen)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	pins := s.Pins()
	if pins.Task.TaskID != tc.ID || pins.Task.RevisionHash != tc.RevisionHash() {
		t.Fatalf("task pin %+v 与内容重算结果不符(AC4:pin 只能从内容推导)", pins.Task)
	}
	if pins.Verification.Hash != frozen.Hash() || pins.Verification.Revision != frozen.Revision() {
		t.Fatalf("verification pin %+v 与冻结内容不符(AC4)", pins.Verification)
	}

	// 同一内容重复编译 ⇒ 同一身份(身份是内容的确定性纯函数)。
	again, err := contract.Compile(v14MustTask(t, v14TaskYAML), v14MustFrozenSimple(t, "TASK-P109", 3))
	if err != nil {
		t.Fatalf("Compile(重放): %v", err)
	}
	if again.Hash() != s.Hash() {
		t.Fatalf("同一内容对编译出不同身份 %q vs %q(AC4)", again.Hash(), s.Hash())
	}

	// 内容漂移:pin 不变,内容变了 —— Open 必须以 ErrPinMismatch 拒绝,
	// 调用方不能把 pin 重新锚到漂移后的内容上。
	drifted := v14MustFrozenSimple(t, "TASK-P109", 4)
	if _, err := contract.Open(pins, tc, drifted); !errors.Is(err, contract.ErrPinMismatch) {
		t.Fatalf("Open(内容漂移): err = %v, want ErrPinMismatch(AC4)", err)
	}

	// 伪造 pin:声明一个不存在的修订哈希 —— Open 对照实际内容拒绝。
	forged := pins
	forged.Task.RevisionHash = "sha256:" + strings.Repeat("0", 64)
	if _, err := contract.Open(forged, tc, frozen); !errors.Is(err, contract.ErrPinMismatch) {
		t.Fatalf("Open(伪造 task pin): err = %v, want ErrPinMismatch(AC4)", err)
	}

	// task id 不一致:两个 pin 各说各话 —— 拒绝。
	if _, err := contract.Open(pins, v14MustTask(t, v14TaskYAML), v14MustFrozenSimple(t, "TASK-OTHER", 3)); !errors.Is(err, contract.ErrTaskMismatch) {
		t.Fatalf("Open(task id 不一致): err = %v, want ErrTaskMismatch(AC4)", err)
	}
}

// v14-AC4c:聚合透出的底层合同与源合同逐字段一致;访问器返回深拷贝,调用方
// 改动拷贝不影响已钉住的 snapshot(修订 = 新 snapshot,不改写权威)。
func TestV14ContractAggregateExposesUnderlyingAuthorityUnchanged(t *testing.T) {
	tc := v14MustTask(t, v14TaskYAML)
	frozen := v14MustFrozenSimple(t, "TASK-P109", 3)
	s, err := contract.Compile(tc, frozen)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	// 底层合同原样透出,不裁剪、不改写。
	if got := s.Task(); !reflect.DeepEqual(got, tc) {
		t.Fatalf("Snapshot.Task() 与源 TaskContract 不一致(AC4:聚合不覆盖底层权威)")
	}
	if got := s.Verification().Hash(); got != frozen.Hash() {
		t.Fatalf("Snapshot.Verification() 哈希 %q 与冻结契约 %q 不一致(AC4)", got, frozen.Hash())
	}

	// 改动透出拷贝:snapshot 身份与 pin 不动(深拷贝纪律)。
	mutated := s.Task()
	mutated.Objective = "tampered objective"
	mutated.Scope.AllowedPaths = append(mutated.Scope.AllowedPaths, "cmd/evil/**")
	if len(mutated.Acceptance) > 0 {
		mutated.Acceptance[0].Statement = "tampered statement"
	}
	if reflect.DeepEqual(s.Task(), mutated) {
		t.Fatal("Snapshot.Task() 返回了可变共享引用(AC4:访问器必须深拷贝)")
	}
	if s.Task().Objective == "tampered objective" || s.Hash() != hashOfSnapshot(t, tc, frozen) {
		t.Fatalf("透过拷贝改写了 snapshot(AC4:修订 = 新 snapshot)")
	}
	frozenCopy := s.Verification().Contract()
	if len(frozenCopy.Acceptances) > 0 {
		frozenCopy.Acceptances[0].Statement = "tampered binding"
	}
	if s.Verification().Contract().Acceptances[0].Statement == "tampered binding" {
		t.Fatal("透过 Frozen.Contract() 拷贝改写了冻结契约(AC4)")
	}
}

// hashOfSnapshot 独立重编译同一内容对,取 snapshot 身份(不读私有字段)。
func hashOfSnapshot(t *testing.T, tc taskcontract.TaskContract, frozen *verification.Frozen) string {
	t.Helper()
	s, err := contract.Compile(tc, frozen)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	return s.Hash()
}

// v14-AC4d:编译器不改写权威 —— Compiled 原样携带 snapshot 的 pin,Bind 前后
// snapshot 身份不变;同一输入重复 Bind 得到同一 Compiled 身份。
func TestV14CompiledContractDoesNotRewriteAuthority(t *testing.T) {
	tc := v14MustTask(t, v14TaskYAML)
	frozen := v14MustFrozenSimple(t, "TASK-P109", 3)
	snap, err := contract.Compile(tc, frozen)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	policies := []engineeringpolicy.Policy{
		v14MustPolicy(t, `
id: policy-migration
enforcement: gate
rationale: migration-class changes must prove upgrade path on a fresh database
trigger:
  risk_signals:
    - schema_migration
requires:
  - fresh_database
gate_condition:
  all_of:
    - fresh_database
`),
	}
	r := v14MustRisk(t, risk.SignalSchemaMigration)

	beforePins, beforeHash := snap.Pins(), snap.Hash()
	c, err := contract.Bind(snap, policies, r)
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}
	if snap.Pins() != beforePins || snap.Hash() != beforeHash {
		t.Fatal("Bind 改动了 snapshot 的 pin 或身份(AC4:编译是纯投影)")
	}
	if c.Pins() != snap.Pins() {
		t.Fatalf("Compiled.Pins() %v != snapshot pin %v(AC4:编译器不改写 pin)", c.Pins(), snap.Pins())
	}
	if c.Snapshot() != snap {
		t.Fatal("Compiled 未只读引用被编译的 snapshot(AC4)")
	}

	// 同一输入(新切片)重复编译 ⇒ 同一身份(确定性,与输入切片无关)。
	rc, err := contract.Bind(snap, []engineeringpolicy.Policy{policies[0]}, r)
	if err != nil {
		t.Fatalf("Bind(重放): %v", err)
	}
	if rc.Hash() != c.Hash() {
		t.Fatalf("同一输入编译出不同身份 %q vs %q(AC4)", rc.Hash(), c.Hash())
	}
}

// v14-AC4e:requires[] 双重用途一致(PRD §8.3)—— 命中 gate 策略执行前通告的
// requires 键与执行后门禁键同源;guidance 的 requires 只通告、永不阻断。
func TestV14RequiresDoubleDutyPreAndPost(t *testing.T) {
	gatePolicy := v14MustPolicy(t, `
id: policy-migration
enforcement: gate
rationale: migration-class changes must prove upgrade path on a fresh database
trigger:
  risk_signals:
    - schema_migration
requires:
  - fresh_database
gate_condition:
  all_of:
    - fresh_database
`)
	guidancePolicy := v14MustPolicy(t, `
id: policy-migration-notes
enforcement: guidance
rationale: guidance only, no blocking power in v1.4
trigger:
  requirement_refs:
    - .tally/requirements/REQ-007.md
requires:
  - clear_migration_notes
`)
	c := v14Compile(t, v14MustFrozenSimple(t, "TASK-P109", 1),
		[]engineeringpolicy.Policy{gatePolicy, guidancePolicy}, v14MustRisk(t, risk.SignalSchemaMigration))

	// 执行前通告面:每个命中策略一条,Keys 与策略 requires 同源。
	requires := c.Requires()
	keysByPolicy := map[string][]string{}
	for _, item := range requires.Items {
		keysByPolicy[item.PolicyID] = item.Keys
	}
	for _, p := range []engineeringpolicy.Policy{gatePolicy, guidancePolicy} {
		got, ok := keysByPolicy[p.ID]
		if !ok {
			t.Fatalf("命中策略 %s 缺少执行前 requires 通告(PRD §8.3 Prevention)", p.ID)
		}
		if !reflect.DeepEqual(got, p.Requires) {
			t.Fatalf("策略 %s 通告键 %v != requires %v(双重用途必须同源)", p.ID, got, p.Requires)
		}
	}

	// 执行后门禁面:gate 策略的通告键就是门禁键(全部进入 all_of 求值)。
	gateKeys := c.GateKeys()
	if !reflect.DeepEqual(gateKeys, gatePolicy.GateCondition.AllOf) {
		t.Fatalf("GateKeys %v != gate 策略 gate_condition %v(执行后门禁与执行前通告脱钩)", gateKeys, gatePolicy.GateCondition.AllOf)
	}
	for _, key := range gateKeys {
		if !containsStr(gatePolicy.Requires, key) {
			t.Fatalf("门禁键 %s 不在该策略执行前通告的 requires 中(Agent 被要求未告知的条件,PRD §8.3)", key)
		}
	}

	// guidance 无阻断权:其 requires 出现在通告面,绝不进入门禁面。
	for _, key := range guidancePolicy.Requires {
		if containsStr(gateKeys, key) {
			t.Fatalf("guidance 策略的 requires 键 %s 混入门禁面(guidance 在 v1.4 拿不到阻断权)", key)
		}
	}
}

func containsStr(vals []string, v string) bool {
	for _, x := range vals {
		if x == v {
			return true
		}
	}
	return false
}
