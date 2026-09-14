package intentevidence

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"tally/internal/intake"
	"tally/internal/risk"
)

// 好事实基线:各测试在其上做单点变异。
func goodFacts() Facts {
	return Facts{
		TaskID:    "TASK-0023",
		TaskType:  TaskTypeBugFix,
		SourceRef: "GitHub Issue #142",
		RiskHints: []string{"schema_migration", "auth_symbol_change"},
	}
}

// TestFreezeCanonicalizesDeterministically:同一份事实无论提示顺序、重复、
// 空白排版如何,冻结结果逐字段相同、Revision 相同(要求 2:确定性规范化)。
func TestFreezeCanonicalizesDeterministically(t *testing.T) {
	base, err := Freeze(goodFacts())
	if err != nil {
		t.Fatalf("Freeze(好事实): %v", err)
	}
	if base.IsZero() {
		t.Fatal("冻结结果为零值")
	}
	if !strings.HasPrefix(base.Revision(), "sha256:") {
		t.Fatalf("Revision %q 不是 sha256:<hex> 形式", base.Revision())
	}

	shuffled := Facts{
		TaskID:    "  TASK-0023", // 首尾空白是排版
		TaskType:  TaskTypeBugFix,
		SourceRef: " GitHub   Issue #142 ",                                                  // 内部连续空白是排版
		RiskHints: []string{"auth_symbol_change", "schema_migration", "auth_symbol_change"}, // 乱序 + 重复
	}
	other, err := Freeze(shuffled)
	if err != nil {
		t.Fatalf("Freeze(变体排版): %v", err)
	}
	if !base.Equal(other) {
		t.Error("排版/顺序/重复变体冻结出了不同内容(规范化不确定)")
	}
	if base.Revision() != other.Revision() {
		t.Errorf("Revision 不稳定: %s != %s", base.Revision(), other.Revision())
	}
	if got := base.RiskHints(); !reflect.DeepEqual(got, []string{"auth_symbol_change", "schema_migration"}) {
		t.Errorf("RiskHints 未排序去重: %v", got)
	}
	if got := base.SourceRef(); got != "GitHub Issue #142" {
		t.Errorf("SourceRef 空白未折叠: %q", got)
	}
}

// TestFreezeFailsClosed:结构性非法与词表外的输入一律类型化报错,
// 绝不产出「部分冻结」的中间态(验收标准 2 的 Freeze 侧)。
func TestFreezeFailsClosed(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Facts)
		wantErr error
	}{
		{"缺 task_id", func(f *Facts) { f.TaskID = "" }, ErrInvalidEvidence},
		{"task_id 纯空白", func(f *Facts) { f.TaskID = "   " }, ErrInvalidEvidence},
		{"task_id 含内部空白", func(f *Facts) { f.TaskID = "TASK 0023" }, ErrInvalidEvidence},
		{"task_type 空", func(f *Facts) { f.TaskType = "" }, ErrUnsupportedTaskType},
		{"task_type 词表外", func(f *Facts) { f.TaskType = "refactor" }, ErrUnsupportedTaskType},
		{"task_type 大小写不折叠", func(f *Facts) { f.TaskType = "BUG_FIX" }, ErrUnsupportedTaskType},
		{"空白提示条目", func(f *Facts) { f.RiskHints = []string{"schema_migration", "  "} }, ErrInvalidEvidence},
		{"全空白提示条目", func(f *Facts) { f.RiskHints = []string{"\t\n"} }, ErrInvalidEvidence},
	}
	for _, tc := range cases {
		facts := goodFacts()
		tc.mutate(&facts)
		fi, err := Freeze(facts)
		if !errors.Is(err, tc.wantErr) {
			t.Errorf("%s: err = %v, 期望 %v(必须 fail closed)", tc.name, err, tc.wantErr)
		}
		if !fi.IsZero() {
			t.Errorf("%s: 失败路径产出了非零值(部分冻结中间态)", tc.name)
		}
	}
}

// TestFrozenIntentImmutableByType:不可变语义在类型层面的落点 ——
//
//	(a) 字段全部非导出,包外不存在任何写通道;
//	(b) 没有 Set 类方法,且全部方法都是值接收者(指针类型方法集不多出任何
//	    方法,不存在经指针改写内部状态的方法);
//	(c) 集合取值返回副本,外部改动不穿透回证据。
func TestFrozenIntentImmutableByType(t *testing.T) {
	rt := reflect.TypeOf(FrozenIntent{})
	for i := 0; i < rt.NumField(); i++ {
		if rt.Field(i).IsExported() {
			t.Errorf("FrozenIntent.%s 是导出字段:不可变证据不得有包外写通道", rt.Field(i).Name)
		}
	}
	valueMethods := rt.NumMethod()
	ptrMethods := reflect.PointerTo(rt).NumMethod()
	if ptrMethods != valueMethods {
		t.Errorf("*FrozenIntent 方法集(%d)多于 FrozenIntent(%d):存在指针接收者方法,不可变类型只允许值接收者", ptrMethods, valueMethods)
	}
	for i := 0; i < valueMethods; i++ {
		if name := rt.Method(i).Name; strings.HasPrefix(strings.ToLower(name), "set") {
			t.Errorf("FrozenIntent 存在 setter 类方法 %s(冻结后不可变)", name)
		}
	}

	fi, err := Freeze(goodFacts())
	if err != nil {
		t.Fatalf("Freeze: %v", err)
	}
	hints := fi.RiskHints()
	hints[0] = "tampered"
	if got := fi.RiskHints(); reflect.DeepEqual(got, hints) {
		t.Error("RiskHints 返回了内部切片:外部改动穿透了不可变证据")
	}
}

// TestFreezeIsolatesInputFacts:冻结后改动输入 Facts 不影响已冻结证据
// (冻结是一次性快照,不是对输入的引用)。
func TestFreezeIsolatesInputFacts(t *testing.T) {
	facts := goodFacts()
	fi, err := Freeze(facts)
	if err != nil {
		t.Fatalf("Freeze: %v", err)
	}
	before := fi.Revision()
	facts.RiskHints[0] = "mutated_after_freeze"
	facts.SourceRef = "mutated"
	if fi.Revision() != before {
		t.Error("冻结后改动输入 Facts 改变了证据(冻结不是快照)")
	}
	if got := fi.RiskHints(); got[0] == "mutated_after_freeze" {
		t.Error("冻结后改动输入 Facts 改变了证据内容")
	}
}

// TestRevisionChangesOnlyWithContent:任何实质内容差异都改变 Revision;
// 只有排版差异不改变(哈希是内容的身份,不是文档的身份)。
func TestRevisionChangesOnlyWithContent(t *testing.T) {
	base, err := Freeze(goodFacts())
	if err != nil {
		t.Fatalf("Freeze: %v", err)
	}
	mustRevision := func(f Facts) string {
		t.Helper()
		fi, err := Freeze(f)
		if err != nil {
			t.Fatalf("Freeze(%+v): %v", f, err)
		}
		return fi.Revision()
	}
	if rev := mustRevision(goodFacts()); rev != base.Revision() {
		t.Error("相同内容两次冻结得到不同 Revision")
	}
	for _, tc := range []struct {
		name   string
		mutate func(*Facts)
	}{
		{"task_id 变", func(f *Facts) { f.TaskID = "TASK-0024" }},
		{"task_type 变", func(f *Facts) { f.TaskType = TaskTypeMigration }},
		{"source_ref 变", func(f *Facts) { f.SourceRef = "GitHub Issue #143" }},
		{"source_ref 移除", func(f *Facts) { f.SourceRef = "" }},
		{"新增提示", func(f *Facts) { f.RiskHints = append(f.RiskHints, "protected_surface_change") }},
		{"移除提示", func(f *Facts) { f.RiskHints = f.RiskHints[:1] }},
	} {
		facts := goodFacts()
		tc.mutate(&facts)
		if rev := mustRevision(facts); rev == base.Revision() {
			t.Errorf("%s:Revision 未变化(哈希对内容不敏感)", tc.name)
		}
	}
}

// TestRiskSignalCandidatesPreserveUnknown:确认提示逐条进入推导候选,_basis
// 格式统一为 intent:<task-id>;枚举之外的提示经 risk.Derive 保持 Unknown 可见
// (INV-V14-09 fail-safe),绝不静默消失或被当作低风险(DL-05:归类归派生层)。
func TestRiskSignalCandidatesPreserveUnknown(t *testing.T) {
	fi, err := Freeze(Facts{
		TaskID:    "TASK-0009",
		TaskType:  TaskTypeMigration,
		RiskHints: []string{"schema_migration", "totally_unknown_signal"},
	})
	if err != nil {
		t.Fatalf("Freeze: %v", err)
	}
	cands := fi.RiskSignalCandidates()
	if len(cands) != 2 {
		t.Fatalf("候选数 = %d, 应为 2(逐条保留,不丢弃)", len(cands))
	}
	for _, c := range cands {
		if !reflect.DeepEqual(c.Basis, []string{"intent:TASK-0009"}) {
			t.Errorf("候选 %s 的 basis = %v, 应为 [intent:TASK-0009]", c.ID, c.Basis)
		}
	}

	res, err := risk.Derive(cands)
	if err != nil {
		t.Fatalf("risk.Derive: %v", err)
	}
	if len(res.Detected) != 1 || res.Detected[0].ID != risk.SignalSchemaMigration {
		t.Errorf("已知提示未被检出: %+v", res.Detected)
	}
	if len(res.Unknown) != 1 || res.Unknown[0].ID != risk.SignalID("totally_unknown_signal") {
		t.Errorf("未知提示未原样保留为 Unknown: %+v", res.Unknown)
	}
	if !res.HasUnknown() || !res.AboveMinimal() {
		t.Error("未知提示必须把派生集保持在最小风险地板之上(fail safe)")
	}
}

// TestRiskSignalCandidatesEmpty:确认零条提示是合法证据,投影为空候选,
// 推导得到空集(合法结果,不是错误)。
func TestRiskSignalCandidatesEmpty(t *testing.T) {
	fi, err := Freeze(Facts{TaskID: "TASK-0001", TaskType: TaskTypeBugFix})
	if err != nil {
		t.Fatalf("Freeze(零提示): %v", err)
	}
	if cands := fi.RiskSignalCandidates(); cands != nil {
		t.Errorf("零提示投影 = %v, 应为 nil", cands)
	}
}

// TestCanonicalPinsFormat:规范文本格式钉死(版本前缀 + 固定字段序),
// 格式漂移必须显式改 canonicalVersion,不允许静默换哈希空间。
func TestCanonicalPinsFormat(t *testing.T) {
	fi, err := Freeze(Facts{
		TaskID:    "TASK-0023",
		TaskType:  TaskTypeBugFix,
		SourceRef: "GitHub Issue #142",
		RiskHints: []string{"schema_migration", "auth_symbol_change"},
	})
	if err != nil {
		t.Fatalf("Freeze: %v", err)
	}
	want := "intent/v1\ntask:TASK-0023\ntype:bug_fix\nsource:GitHub Issue #142\nhint:auth_symbol_change\nhint:schema_migration\n"
	if got := fi.Canonical(); got != want {
		t.Errorf("Canonical 格式漂移:\n got = %q\nwant = %q", got, want)
	}
	if canonicalVersion != "intent/v1" {
		t.Errorf("canonicalVersion = %q, 与钉住的格式不符", canonicalVersion)
	}
}

// TestTaskTypeVocabularyMatchesIntake:冻结词表与 intake 的 MVP 词表逐值
// 一致(同一产品事实,两侧不得漂移);本包生产代码不依赖 intake,一致性由
// 本测试钉住。
func TestTaskTypeVocabularyMatchesIntake(t *testing.T) {
	if string(TaskTypeBugFix) != string(intake.TaskTypeBugFix) {
		t.Errorf("TaskTypeBugFix = %q, intake 侧 = %q", TaskTypeBugFix, intake.TaskTypeBugFix)
	}
	if string(TaskTypeMigration) != string(intake.TaskTypeMigration) {
		t.Errorf("TaskTypeMigration = %q, intake 侧 = %q", TaskTypeMigration, intake.TaskTypeMigration)
	}
	if TaskType("refactor").IsValid() {
		t.Error("词表外类型被判定合法(必须 fail closed)")
	}
}
