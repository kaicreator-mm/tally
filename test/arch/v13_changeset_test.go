// v1.3 ChangeSet 不变量测试(P1-45,PRD §3.2 稳定 ID)。
//
// 覆盖 P1 激活的变更域不变量:
//   - INV-CHANGE-01 · P1 — Every changed hunk must be accounted for before
//     approval(记账入口:确定性 Hunk Inventory + 确定性归账);
//   - INV-CHANGE-02 · P1 — ChangeSet is the immutable unit of verification,
//     review and integration。
package arch

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"tally/internal/changeset"
	"tally/internal/review"
	"tally/internal/verification"
)

// chgAt 是固定时钟:候选与取代事实的时刻全部确定。
var chgAt = time.Date(2026, 9, 6, 9, 0, 0, 0, time.UTC)

// chgChangeSet 构造一个带改动文件与 hunk 坐标的最小候选快照。
func chgChangeSet(t *testing.T, candidate, run string) *changeset.ChangeSet {
	t.Helper()
	cs, err := changeset.New(changeset.Spec{
		Project:           "proj-arch",
		BaseRevision:      "base-1",
		CandidateRevision: candidate,
		ProducerRun:       run,
		CreatedAt:         chgAt,
		Entities:          []string{"widget"},
		Files: []changeset.ChangedFile{{
			Path:    "internal/widget/widget.go",
			Content: "package widget\n",
			Hunks:   []changeset.Hunk{{OldStart: 1, OldLines: 1, NewStart: 1, NewLines: 2}},
		}},
	})
	if err != nil {
		t.Fatalf("changeset.New: %v", err)
	}
	return cs
}

// INV-CHANGE-02:快照构造之后 API 层面不可变 —— 取值返回副本,改返回值
// 不影响快照本体;身份由不可变输入推导,保持稳定。
func TestV13ChangeSetSnapshotImmutable(t *testing.T) {
	cs := chgChangeSet(t, "cand-1", "run-1")
	id, patch := cs.ID(), cs.PatchIdentity()

	files := cs.Files()
	files[0].Content = "hacked"
	files[0].Executable = true
	files[0].Hunks[0].NewStart = 99
	ents := cs.Entities()
	ents[0] = "hacked"

	got := cs.Files()
	if got[0].Content != "package widget\n" || got[0].Executable || got[0].Hunks[0].NewStart != 1 {
		t.Fatalf("修改 Files() 返回值影响了快照本体: %+v(INV-CHANGE-02)", got[0])
	}
	if cs.Entities()[0] != "widget" {
		t.Fatalf("修改 Entities() 返回值影响了快照本体: %q(INV-CHANGE-02)", cs.Entities()[0])
	}
	if cs.ID() != id || cs.PatchIdentity() != patch {
		t.Fatalf("取值路径改变了快照身份: id %s→%s patch %s→%s(INV-CHANGE-02)", id, cs.ID(), patch, cs.PatchIdentity())
	}
	// 可执行代码内容一变,内容身份必变:身份敏感于内容,不存在“原地换内容”。
	other, err := changeset.New(changeset.Spec{
		Project: "proj-arch", BaseRevision: "base-1", CandidateRevision: "cand-1",
		ProducerRun: "run-1", CreatedAt: chgAt,
		Files: []changeset.ChangedFile{{Path: "internal/widget/widget.go", Content: "package widget // mutated\n"}},
	})
	if err != nil {
		t.Fatalf("changeset.New(mutated): %v", err)
	}
	if other.ID() == cs.ID() || other.PatchIdentity() == cs.PatchIdentity() {
		t.Fatal("不同内容的候选得到同一身份(内容身份必须敏感于可执行内容,INV-CHANGE-02)")
	}
}

// INV-CHANGE-02 / INV-VER-04:取代关系是独立的 append-only 事实,不是快照上
// 的可变字段;重做(含 spec_defect 重做)永远产生新快照,绝不改写旧候选本体。
func TestV13ChangeSetSupersessionIsAppendOnlyFact(t *testing.T) {
	a := chgChangeSet(t, "cand-1", "run-1")
	b := chgChangeSet(t, "cand-2", "run-2")

	// 非法取代事实一律拒绝。
	if _, err := changeset.NewSupersededFact(a.ID(), a.ID(), changeset.SupersedeRework, chgAt); !errors.Is(err, changeset.ErrBadSupersede) {
		t.Fatalf("old == new 的取代事实: err = %v, want ErrBadSupersede", err)
	}
	if _, err := changeset.NewSupersededFact(a.ID(), b.ID(), changeset.SupersedeReason("retcon"), chgAt); !errors.Is(err, changeset.ErrBadSupersede) {
		t.Fatalf("词表外原因: err = %v, want ErrBadSupersede", err)
	}

	fact, err := changeset.NewSupersededFact(a.ID(), b.ID(), changeset.SupersedeSpecDefect, chgAt)
	if err != nil {
		t.Fatalf("NewSupersededFact(spec_defect): %v", err)
	}
	if fact.Old != a.ID() || fact.New != b.ID() || fact.Reason != changeset.SupersedeSpecDefect || !fact.At.Equal(chgAt.UTC()) {
		t.Fatalf("取代事实字段不一致: %+v", fact)
	}
	// 取代不修改被取代快照:A 的身份与内容身份原样不变。
	if a.ID() != chgChangeSet(t, "cand-1", "run-1").ID() {
		t.Fatal("取代构造后候选身份发生变化(INV-CHANGE-02:取代只能追加事实)")
	}
}

// chgMutatorRe 匹配“修改入口”形态的导出名(Set*/Update*/Apply*/Mutate*/…)。
var chgMutatorRe = regexp.MustCompile(`^(Set|Update|Apply|Mutate|Overwrite|Rewrite|Delete|Clear|Replace|Save|Write)[A-Z]`)

// INV-CHANGE-02(结构面):ChangeSet 类型字段全部不导出、没有任何修改入口
// —— 不可变是类型级保证,不依赖调用方自觉。
func TestV13ChangeSetTypeHasNoMutationPath(t *testing.T) {
	structs := map[string]*ast.StructType{}
	methods := map[string]map[string]bool{}
	for _, file := range goFiles(t, filepath.Join(repoInternal, "changeset"), false) {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatalf("解析 %s: %v", file, err)
		}
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if name := receiverTypeName(d); name != "" {
					if methods[name] == nil {
						methods[name] = map[string]bool{}
					}
					methods[name][d.Name.Name] = true
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok {
						if st, ok := ts.Type.(*ast.StructType); ok {
							structs[ts.Name.Name] = st
						}
					}
				}
			}
		}
	}

	csStruct, ok := structs["ChangeSet"]
	if !ok {
		t.Fatal("internal/changeset 缺少 ChangeSet 结构体(INV-CHANGE-02 的类型落点)")
	}
	for _, field := range csStruct.Fields.List {
		for _, name := range field.Names {
			if ast.IsExported(name.Name) {
				t.Errorf("ChangeSet 导出字段 %s:包外可原地改写快照(INV-CHANGE-02)", name.Name)
			}
		}
	}
	for name := range methods["ChangeSet"] {
		if chgMutatorRe.MatchString(name) {
			t.Errorf("ChangeSet 存在修改入口方法 %s(INV-CHANGE-02:快照只读)", name)
		}
	}
	if !methods["ChangeSet"]["ID"] || !methods["ChangeSet"]["Files"] {
		t.Fatal("internal/changeset 缺少 ChangeSet 的只读取值方法(ID/Files),断言对象可能已漂移")
	}
}

// chgFactTables 是 ChangeSet 域的持久化事实面:快照本体与取代事实都只追加。
var chgFactTables = []string{"changeset", "changeset_superseded_fact"}

// INV-CHANGE-02(持久化面):快照与取代事实在数据库层由 append-only 语句级
// 触发器兜底,sqlc 查询面没有 UPDATE/DELETE 路径。
func TestV13ChangeSetPersistenceAppendOnly(t *testing.T) {
	content := readAllMigrations(t)
	for _, tbl := range chgFactTables {
		re := regexp.MustCompile(`(?is)CREATE\s+TRIGGER\s+\S+\s+BEFORE\s+UPDATE\s+OR\s+DELETE\s+ON\s+` + tbl + `\b`)
		if !re.MatchString(content) {
			t.Errorf("事实表 %s 缺少阻止 UPDATE/DELETE 的触发器(INV-CHANGE-02 持久化面)", tbl)
		}
	}
	for _, file := range queryFiles(t) {
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("读 %s: %v", file, err)
		}
		for _, tbl := range chgFactTables {
			re := regexp.MustCompile(`(?is)\b(UPDATE\s+` + tbl + `|DELETE\s+FROM\s+` + tbl + `)\b`)
			if re.Match(b) {
				t.Errorf("%s: 对事实表 %s 的 UPDATE/DELETE 查询(INV-CHANGE-02:快照与取代事实只追加)", file, tbl)
			}
		}
	}
}

// chgAccountingDiff 是一次候选 diff:2 个文本 hunk 的实现文件、1 个文本 hunk
// 的文档、一个纯 rename 条目与一个 binary 条目(零 hunk 条目不允许从记账中消失)。
const chgAccountingDiff = `diff --git a/internal/widget/widget.go b/internal/widget/widget.go
--- a/internal/widget/widget.go
+++ b/internal/widget/widget.go
@@ -1,2 +1,3 @@
 package widget
+func A() {}
 func B() {}
@@ -10,2 +11,3 @@
 func C() {}
+func D() {}
 func E() {}
diff --git a/docs/notes.md b/docs/notes.md
--- a/docs/notes.md
+++ b/docs/notes.md
@@ -1 +1 @@
-old
+new
diff --git a/old.txt b/new.txt
similarity index 90%
rename from old.txt
rename to new.txt
diff --git a/assets/logo.png b/assets/logo.png
index 1111111..2222222 100644
Binary files a/assets/logo.png and b/assets/logo.png differ
`

// chgFrozen 是最小冻结契约:一条 independence=none 的验收条款。
func chgFrozen(t *testing.T) *verification.Frozen {
	t.Helper()
	contract := verification.Contract{
		TaskID:   "TASK-ARCH",
		Revision: 1,
		Acceptances: []verification.Acceptance{
			{ID: "AC-01", Statement: "widget renders"},
		},
	}
	frozen, err := contract.Freeze()
	if err != nil {
		t.Fatalf("契约冻结: %v", err)
	}
	return frozen
}

// chgAccountingInput 组装确定性归账的全部输入(清单 + 规则 + 冻结契约 +
// 与契约同 revision 的两份覆盖报告)。
func chgAccountingInput(t *testing.T) (review.AccountingInput, *changeset.ChangeSet, review.Inventory) {
	t.Helper()
	cs := chgChangeSet(t, "cand-1", "run-1")
	inv, err := review.InventoryFromDiff(chgAccountingDiff)
	if err != nil {
		t.Fatalf("InventoryFromDiff: %v", err)
	}
	rules := review.DefaultRules()
	rules.Allowed = []string{"**"}
	frozen := chgFrozen(t)
	input := review.AccountingInput{
		ChangeSet: cs,
		Inventory: inv,
		Rules:     rules,
		Frozen:    frozen,
		Functional: verification.FunctionalCoverageReport{
			TaskID: frozen.TaskID(), ContractRevision: frozen.Revision(), ContractHash: frozen.Hash(),
			Acceptances: []verification.AcceptanceCoverage{{AcceptanceID: "AC-01", Covered: true}},
			Complete:    true,
		},
		Independence: verification.IndependenceCoverageReport{
			TaskID: frozen.TaskID(), ContractRevision: frozen.Revision(), ContractHash: frozen.Hash(),
			Acceptances: []verification.AcceptanceIndependence{{AcceptanceID: "AC-01", Required: verification.IndependenceNone, Covered: true}},
			Complete:    true,
		},
	}
	return input, cs, inv
}

// INV-CHANGE-01(记账入口):确定性 Hunk Inventory 与确定性归账是 approval
// 前记账的两个入口 —— 每个 changed hunk 恰好一条记账;binary/rename/mode-only
// 等零 hunk 条目也必须有条目,不允许从 review 中消失;存在未归账条目时
// Owner 不得 Approve;显式归属只能填残差,不能改写规则事实。
func TestV13HunkAccountingEntryPointsCoverEveryHunk(t *testing.T) {
	input, _, inv := chgAccountingInput(t)

	if inv.HunkCount() != 3 || inv.FileCount() != 4 {
		t.Fatalf("清单 = %d 文件 / %d hunk, want 4 文件 / 3 hunk(入口 1:确定性清单)", inv.FileCount(), inv.HunkCount())
	}

	rep, err := review.BuildAccounting(input)
	if err != nil {
		t.Fatalf("BuildAccounting: %v", err)
	}
	// 每个清单 hunk 恰好一条记账(计数分母 = 清单)。
	counts := map[review.HunkID]int{}
	for _, e := range rep.Entries {
		if e.HunkID != "" {
			counts[e.HunkID]++
		}
	}
	for _, h := range inv.Hunks() {
		if counts[h.ID] != 1 {
			t.Fatalf("hunk %s(%s)记账 %d 次, want 恰好 1 次(INV-CHANGE-01)", h.ID, h.Path, counts[h.ID])
		}
	}
	// 零 hunk 文件条目(binary + 纯 rename)各有一条不消失的记账。
	contentless := map[string]bool{}
	for _, e := range rep.Entries {
		if e.HunkID == "" {
			contentless[e.Path] = true
		}
		if !review.IsValidBucket(e.Bucket) {
			t.Fatalf("条目 %s 落入词表外 bucket %q(INV-CHANGE-01:每个条目恰好一个 primary bucket)", e.Path, e.Bucket)
		}
	}
	for _, want := range []string{"assets/logo.png", "new.txt"} {
		if !contentless[want] {
			t.Fatalf("零 hunk 条目 %s 从记账中消失(INV-CHANGE-01)", want)
		}
	}
	if len(rep.Entries) != inv.HunkCount()+2 {
		t.Fatalf("记账条目 = %d, want 清单 hunk 数 3 + 零 hunk 条目 2(INV-CHANGE-01)", len(rep.Entries))
	}
	// 文档 hunk 被确定性规则归账。
	var docsSeen bool
	for _, e := range rep.Entries {
		if e.HunkID != "" && e.Path == "docs/notes.md" {
			docsSeen = e.Bucket == review.BucketDocs && e.ByRule
		}
	}
	if !docsSeen {
		t.Fatalf("docs hunk 未被规则归账为 docs(INV-CHANGE-01 的确定性规则面)")
	}
	// 未归账条目(2 个实现 hunk + binary + rename)阻断 Approve。
	if rep.UnattributedCount() != 4 || !rep.ApproveBlocked() {
		t.Fatalf("未归账 = %d, ApproveBlocked = %v, want 4/true(§19.2:Unattributed 非空不得 Approve)", rep.UnattributedCount(), rep.ApproveBlocked())
	}

	// 显式归属填残差:全部归账后阻断消除;归属条目带验收标签且 ByRule=false。
	widgetHunks := inv.Hunks()[:0]
	for _, h := range inv.Hunks() {
		if h.Path == "internal/widget/widget.go" {
			widgetHunks = append(widgetHunks, h)
		}
	}
	if len(widgetHunks) != 2 {
		t.Fatalf("实现文件 hunk 数 = %d, want 2", len(widgetHunks))
	}
	attributed := input
	attributed.Attributions = []review.Attribution{
		{HunkID: widgetHunks[0].ID, AcceptanceIDs: []string{"AC-01"}},
		{HunkID: widgetHunks[1].ID, AcceptanceIDs: []string{"AC-01"}},
		{Path: "assets/logo.png", AcceptanceIDs: []string{"AC-01"}},
		{Path: "new.txt", AcceptanceIDs: []string{"AC-01"}},
	}
	rep2, err := review.BuildAccounting(attributed)
	if err != nil {
		t.Fatalf("BuildAccounting(归属后): %v", err)
	}
	if rep2.UnattributedCount() != 0 || rep2.ApproveBlocked() {
		t.Fatalf("全部归属后仍未放行: unattributed=%d blocked=%v(INV-CHANGE-01:归属是唯一残差出口)", rep2.UnattributedCount(), rep2.ApproveBlocked())
	}
	for _, h := range widgetHunks {
		var related bool
		for _, acc := range rep2.Acceptances {
			if acc.AcceptanceID == "AC-01" {
				for _, id := range acc.RelatedHunks {
					if id == h.ID {
						related = true
					}
				}
			}
		}
		if !related {
			t.Fatalf("归属 hunk %s 未出现在 AC-01 的 RelatedHunks(INV-CHANGE-01 的验收映射)", h.ID)
		}
	}

	// 规则事实不可被改写:对已被规则归账的 docs hunk 提交归属 → 显式报错。
	// (提案侧分类是辅助,永远不能 relabel 确定性规则的事实。)
	var docsHunk review.HunkID
	for _, h := range inv.Hunks() {
		if h.Path == "docs/notes.md" {
			docsHunk = h.ID
		}
	}
	relabeled := input
	relabeled.Attributions = []review.Attribution{{HunkID: docsHunk, AcceptanceIDs: []string{"AC-01"}}}
	if _, err := review.BuildAccounting(relabeled); !errors.Is(err, review.ErrBadAttribution) {
		t.Fatalf("归属改写规则已归账条目: err = %v, want ErrBadAttribution(INV-CHANGE-01/INV-AUTH-02)", err)
	}

	// 报告是纯值投影:同输入重算必得同报告。
	repAgain, err := review.BuildAccounting(input)
	if err != nil {
		t.Fatalf("BuildAccounting(重算): %v", err)
	}
	if !reflect.DeepEqual(rep, repAgain) {
		t.Fatal("同输入两次归账结果不一致(报告必须可重建)")
	}
}

// INV-CHANGE-01(入口反面):解析失败的 diff 一律显式报错且不产出残缺清单
// —— 被丢掉的 hunk 会制造“所有改动都已看过”的假覆盖感。
func TestV13InventoryFromDiffRejectsMalformedDiff(t *testing.T) {
	cases := map[string]string{
		"无法识别的行":   "diff --git a/x.go b/x.go\n@@ garbage @@\n",
		"截断的 hunk": "diff --git a/x.go b/x.go\n--- a/x.go\n+++ b/x.go\n@@ -1,2 +1,2 @@\n-only\n",
	}
	for name, diff := range cases {
		inv, err := review.InventoryFromDiff(diff)
		if err == nil || !errors.Is(err, review.ErrBadDiff) {
			t.Fatalf("%s: err = %v, want ErrBadDiff(入口不得静默吞条目)", name, err)
		}
		if inv.FileCount() != 0 {
			t.Fatalf("%s: 解析失败仍产出 %d 个条目的残缺清单(INV-CHANGE-01)", name, inv.FileCount())
		}
	}
	// 空输入 = 无改动可记账,不是错误。
	if inv, err := review.InventoryFromDiff(""); err != nil || inv.FileCount() != 0 {
		t.Fatalf("空 diff: err = %v, inv = %+v, want 空清单", err, inv)
	}
	// 结构边界:清单与归账是纯确定性推导,不得调用 LLM。
	graph := importGraph(t)
	for _, pkg := range []string{"changeset", "review"} {
		if graph[pkg]["llm"] {
			t.Errorf("internal/%s 导入了 internal/llm:身份推导与 hunk 记账必须是确定性机械路径", pkg)
		}
	}
}

// 防御:确认 diff 常量仍覆盖四类条目(文本 hunk/文档/纯 rename/binary),
// 防止常量被改空后上面的断言失去意义。
func TestV13AccountingDiffFixtureShape(t *testing.T) {
	if strings.Count(chgAccountingDiff, "diff --git ") != 4 {
		t.Fatalf("记账 fixture 应含 4 个 diff 条目,得到 %d", strings.Count(chgAccountingDiff, "diff --git "))
	}
}
