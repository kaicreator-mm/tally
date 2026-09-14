package review

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"tally/internal/approval"
	"tally/internal/changeset"
	"tally/internal/evidence"
	"tally/internal/taskcontract"
	"tally/internal/verification"
)

// testDiff:8 个改动文件、每文件一个 hunk,覆盖全部确定性规则桶:
// 普通代码(unattributed)、required evidence、补充测试、forbidden、
// allowed 之外、migration 风险、docs、第二条款的 required evidence。
const testDiff = `diff --git a/internal/app/api.go b/internal/app/api.go
index 1111111..2222222 100644
--- a/internal/app/api.go
+++ b/internal/app/api.go
@@ -10,2 +10,3 @@ func Handle() {
 	ctx := background()
+	handle(ctx)
 }
diff --git a/internal/app/app_test.go b/internal/app/app_test.go
index 3333333..4444444 100644
--- a/internal/app/app_test.go
+++ b/internal/app/app_test.go
@@ -1,2 +1,2 @@
 ctx
-old
+new
diff --git a/internal/app/helper_test.go b/internal/app/helper_test.go
index 5555555..6666666 100644
--- a/internal/app/helper_test.go
+++ b/internal/app/helper_test.go
@@ -3,1 +3,2 @@
 ctx
+assert(t, ok)
diff --git a/vendor/x.go b/vendor/x.go
index 7777777..8888888 100644
--- a/vendor/x.go
+++ b/vendor/x.go
@@ -1,1 +1,1 @@
-a
+b
diff --git a/other/other.go b/other/other.go
index 9999999..aaaaaaa 100644
--- a/other/other.go
+++ b/other/other.go
@@ -1 +1 @@
-a
+b
diff --git a/migrations/0002_add.sql b/migrations/0002_add.sql
index bbbbbbb..ccccccc 100644
--- a/migrations/0002_add.sql
+++ b/migrations/0002_add.sql
@@ -0,0 +1,2 @@
+CREATE TABLE t(a);
+-- migration
diff --git a/docs/guide.md b/docs/guide.md
index ddddddd..eeeeeee 100644
--- a/docs/guide.md
+++ b/docs/guide.md
@@ -1,1 +1,2 @@
 title
+note
diff --git a/internal/app/api_test.go b/internal/app/api_test.go
index fffffff..1111112 100644
--- a/internal/app/api_test.go
+++ b/internal/app/api_test.go
@@ -5,1 +5,1 @@
-old
+new
`

// metaDiff:零 hunk 的文件条目(binary、纯 mode、纯 rename、空文件新增),
// INV-CHANGE-01:它们同样必须被记账,不允许从 review 中消失。
const metaDiff = `diff --git a/assets/logo.png b/assets/logo.png
index 1111111..2222222 100644
Binary files a/assets/logo.png and b/assets/logo.png differ
diff --git a/run.sh b/run.sh
old mode 100644
new mode 100755
diff --git a/docs/old.md b/docs/new.md
similarity index 100%
rename from docs/old.md
rename to docs/new.md
diff --git a/internal/app/empty.go b/internal/app/empty.go
new file mode 100644
index 0000000..e69de29
--- /dev/null
+++ b/internal/app/empty.go
`

// renameProtectedDiff:把 protected evidence 文件改名(从 protected 路径
// 改名出去仍然触及了受保护文件)。
const renameProtectedDiff = `diff --git a/internal/app/app_test.go b/internal/app/moved_test.go
similarity index 90%
rename from internal/app/app_test.go
rename to internal/app/moved_test.go
index 3333333..4444444
--- a/internal/app/app_test.go
+++ b/internal/app/moved_test.go
@@ -1,1 +1,1 @@
-old
+new
`

// deleteEvidenceDiff:删除一个 required evidence 文件 —— 仍必须被记账为
// acceptance_evidence,不允许「删了就消失」。
const deleteEvidenceDiff = `diff --git a/internal/app/api_test.go b/internal/app/api_test.go
deleted file mode 100644
index fffffff..0000000
--- a/internal/app/api_test.go
+++ /dev/null
@@ -1,1 +0,0 @@
-package api_test
`

var reviewScope = taskcontract.Scope{
	AllowedPaths:   []string{"internal/app", "docs", "migrations"},
	ForbiddenPaths: []string{"vendor"},
}

// reviewContract 构造冻结契约:AC-1(none)的 required evidence 是
// internal/app/app_test.go;AC-2(source)的 required evidence 是
// internal/app/api_test.go,声明 verification_authority basis。
func reviewContract(t *testing.T, mutate func(*verification.Contract)) *verification.Frozen {
	t.Helper()
	c := &verification.Contract{
		TaskID:   "TASK-R1",
		Revision: 1,
		Acceptances: []verification.Acceptance{
			{
				ID:        "AC-1",
				Statement: "AC-1 holds",
				EvidenceBindings: []verification.Binding{
					{
						ID:               "ev-1",
						AuthoringMode:    verification.AuthoringPreCommit,
						VisibilityPolicy: verification.VisibilityVisible,
						Evidence:         verification.EvidenceRef{Type: verification.EvidenceTest, Path: "internal/app/app_test.go"},
					},
				},
			},
			{
				ID:                   "AC-2",
				Statement:            "AC-2 holds",
				IndependenceRequired: verification.IndependenceSource,
				EvidenceBindings: []verification.Binding{
					{
						ID:               "ev-2",
						AuthoringMode:    verification.AuthoringPreCommit,
						VisibilityPolicy: verification.VisibilityVisible,
						Basis:            &verification.IndependenceBasis{Source: verification.SourceVerificationAuthority},
						Evidence:         verification.EvidenceRef{Type: verification.EvidenceTest, Path: "internal/app/api_test.go"},
					},
				},
			},
		},
	}
	if mutate != nil {
		mutate(c)
	}
	f, err := c.Freeze()
	if err != nil {
		t.Fatalf("Freeze: %v", err)
	}
	return f
}

func mustChangeSet(t *testing.T) *changeset.ChangeSet {
	t.Helper()
	cs, err := changeset.New(changeset.Spec{
		Project:           "proj",
		BaseRevision:      "11111111111",
		CandidateRevision: "22222222222",
		ProducerRun:       "run-1",
	})
	if err != nil {
		t.Fatalf("changeset.New: %v", err)
	}
	return cs
}

func mustInventory(t *testing.T, diff string) Inventory {
	t.Helper()
	inv, err := InventoryFromDiff(diff)
	if err != nil {
		t.Fatalf("InventoryFromDiff: %v", err)
	}
	return inv
}

// indepCtx 构造 independence 求值上下文:实现侧 provenance 已登记
// (否则 EffectiveSourceIndependence 保守取 dependent)。
func indepCtx(f *verification.Frozen, csID string) verification.IndependenceContext {
	return verification.IndependenceContext{
		Project:      "proj",
		Verification: evidence.VerificationRevision(f.Hash()),
		ChangeSet:    evidence.ChangeSetID(csID),
		Implementation: evidence.ImplementationProvenance{
			Contributors: []evidence.Contributor{{Name: "agent", Role: "coding_agent"}},
		},
	}
}

// passingVerdicts 为给定 binding 造一条 candidate 上的有效 PASS 判定。
func passingVerdicts(f *verification.Frozen, bindingID string) []evidence.VerdictRecord {
	return []evidence.VerdictRecord{{
		Binding: evidence.BindingRef{
			Project:      "proj",
			Verification: evidence.VerificationRevision(f.Hash()),
			BindingID:    bindingID,
		},
		ChangeSet:  "cs-1",
		Verdict:    evidence.VerdictPass,
		Valid:      true,
		RecordedAt: time.Now(),
	}}
}

// buildInput 组装完整的 AccountingInput。
func buildInput(t *testing.T, f *verification.Frozen, inv Inventory, indepRecs []verification.IndependenceEvidence, attrs []Attribution) AccountingInput {
	t.Helper()
	rules, err := RulesFor(reviewScope, f)
	if err != nil {
		t.Fatalf("RulesFor: %v", err)
	}
	functional, err := verification.DeriveFunctionalCoverage(f, nil)
	if err != nil {
		t.Fatalf("DeriveFunctionalCoverage: %v", err)
	}
	independence, err := verification.DeriveIndependenceCoverage(f, indepCtx(f, "cs-1"), indepRecs)
	if err != nil {
		t.Fatalf("DeriveIndependenceCoverage: %v", err)
	}
	return AccountingInput{
		ChangeSet:    mustChangeSet(t),
		Inventory:    inv,
		Rules:        rules,
		Frozen:       f,
		Functional:   functional,
		Independence: independence,
		Attributions: attrs,
	}
}

func mustBuild(t *testing.T, input AccountingInput) AccountingReport {
	t.Helper()
	rep, err := BuildAccounting(input)
	if err != nil {
		t.Fatalf("BuildAccounting: %v", err)
	}
	return rep
}

func entryByPath(t *testing.T, rep AccountingReport, path string) AccountingEntry {
	t.Helper()
	for _, e := range rep.Entries {
		if e.Path == path {
			return e
		}
	}
	t.Fatalf("报告缺少路径 %s 的记账条目", path)
	return AccountingEntry{}
}

func hunkIDByPath(t *testing.T, rep AccountingReport, path string) HunkID {
	t.Helper()
	for _, e := range rep.Entries {
		if e.Path == path && e.HunkID != "" {
			return e.HunkID
		}
	}
	t.Fatalf("路径 %s 下没有 hunk", path)
	return ""
}

func hasBlocker(rep AccountingReport, kind BlockerKind) bool {
	for _, b := range rep.Blockers {
		if b.Kind == kind {
			return true
		}
	}
	return false
}

// AC1:清单里每个 hunk 恰好产生一条记账条目(一一对应、ID 唯一、bucket
// 合法),且同输入重算必得同一报告(确定性,INV-STATE-01)。
func TestEveryHunkAccountedExactlyOnce(t *testing.T) {
	f := reviewContract(t, nil)
	inv := mustInventory(t, testDiff)
	input := buildInput(t, f, inv, nil, nil)
	rep := mustBuild(t, input)

	if got, want := len(rep.Entries), inv.HunkCount(); got != want {
		t.Fatalf("记账条目数 = %d, want %d(清单 hunk 数)", got, want)
	}
	seen := map[HunkID]int{}
	for _, e := range rep.Entries {
		if e.HunkID == "" {
			t.Errorf("%s: 有 hunk 的文件不应产生零 hunk 条目", e.Path)
		}
		seen[e.HunkID]++
		if !IsValidBucket(e.Bucket) {
			t.Errorf("%s: 非法 bucket %q", e.HunkID, e.Bucket)
		}
	}
	for id, n := range seen {
		if n != 1 {
			t.Errorf("hunk %s 被记账 %d 次, want 恰好 1 次(AC1)", id, n)
		}
	}

	rep2 := mustBuild(t, buildInput(t, f, inv, nil, nil))
	if !reflect.DeepEqual(rep, rep2) {
		t.Fatal("同输入两次归账结果不一致:归账不是确定性的")
	}

	// 报告身份来自 ChangeSet(§19:review 输入是固定 revision 的 ChangeSet)。
	if rep.ChangeSetID != input.ChangeSet.ID() || rep.Project != "proj" ||
		rep.BaseRevision != "11111111111" || rep.CandidateRevision != "22222222222" {
		t.Fatalf("报告身份字段不正确:%+v", rep)
	}
}

// AC1:零 hunk 的文件条目(binary/纯 mode/纯 rename/空文件新增)也必须
// 各得一条记账条目,任何改动不允许从 review 中消失。
func TestContentlessEntriesAccounted(t *testing.T) {
	f := reviewContract(t, nil)
	inv := mustInventory(t, metaDiff)
	if inv.HunkCount() != 0 {
		t.Fatalf("前置失败:metaDiff 应有 0 个 hunk,得到 %d", inv.HunkCount())
	}
	rep := mustBuild(t, buildInput(t, f, inv, nil, nil))

	if len(rep.Entries) != inv.FileCount() {
		t.Fatalf("记账条目数 = %d, want %d(每个零 hunk 文件条目一条)", len(rep.Entries), inv.FileCount())
	}
	paths := map[string]bool{}
	for _, e := range rep.Entries {
		if e.HunkID != "" {
			t.Errorf("%s: 零 hunk 文件条目不应带 hunk id", e.Path)
		}
		if paths[e.Path] {
			t.Errorf("路径 %s 被记账两次", e.Path)
		}
		paths[e.Path] = true
		if !IsValidBucket(e.Bucket) {
			t.Errorf("%s: 非法 bucket %q", e.Path, e.Bucket)
		}
	}
	for _, want := range []string{"assets/logo.png", "run.sh", "docs/new.md", "internal/app/empty.go"} {
		if !paths[want] {
			t.Errorf("缺少零 hunk 条目 %s(改动从 review 中消失)", want)
		}
	}
	// rename 目标 docs/new.md 仍在 docs 规则内;assets/run.sh 在边界外;
	// internal/app/empty.go 是规则无法归账的残差。
	if got := entryByPath(t, rep, "docs/new.md"); got.Bucket != BucketDocs {
		t.Errorf("docs/new.md bucket = %q, want %q", got.Bucket, BucketDocs)
	}
	if got := entryByPath(t, rep, "run.sh"); got.Bucket != BucketScopeViolation {
		t.Errorf("run.sh bucket = %q, want %q", got.Bucket, BucketScopeViolation)
	}
	if got := entryByPath(t, rep, "internal/app/empty.go"); got.Bucket != BucketUnattributed {
		t.Errorf("internal/app/empty.go bucket = %q, want %q", got.Bucket, BucketUnattributed)
	}
}

// AC2:scope/protected/evidence/风险/补充测试/docs 规则确定性生效,
// semantic tags 携带命中事实。
func TestDeterministicRuleBuckets(t *testing.T) {
	f := reviewContract(t, nil)
	rep := mustBuild(t, buildInput(t, f, mustInventory(t, testDiff), nil, nil))

	cases := []struct {
		path     string
		bucket   Bucket
		wantTags []string
	}{
		{"internal/app/api.go", BucketUnattributed, nil},
		{"internal/app/app_test.go", BucketAcceptanceEvidence, []string{TagBindingPrefix + "ev-1", TagACPrefix + "AC-1"}},
		{"internal/app/helper_test.go", BucketSupplementaryTest, []string{TagSupplementaryTest}},
		{"vendor/x.go", BucketScopeViolation, []string{TagScopeForbidden, TagScopeOutsideAllowed}},
		{"other/other.go", BucketScopeViolation, []string{TagScopeOutsideAllowed}},
		{"migrations/0002_add.sql", BucketRiskyChange, []string{TagRiskPrefix + "migration"}},
		{"docs/guide.md", BucketDocs, []string{TagDocs}},
		{"internal/app/api_test.go", BucketAcceptanceEvidence, []string{TagBindingPrefix + "ev-2", TagACPrefix + "AC-2"}},
	}
	for _, tc := range cases {
		e := entryByPath(t, rep, tc.path)
		if e.Bucket != tc.bucket {
			t.Errorf("%s: bucket = %q, want %q", tc.path, e.Bucket, tc.bucket)
		}
		if !e.ByRule {
			t.Errorf("%s: 规则归账条目 ByRule = false", tc.path)
		}
		for _, tag := range tc.wantTags {
			if !hasTag(e.Tags, tag) {
				t.Errorf("%s: 缺少 tag %q(tags=%v)", tc.path, tag, e.Tags)
			}
		}
	}
	// 确定性:直接重跑规则,结果一致。
	inv := mustInventory(t, testDiff)
	rules, err := RulesFor(reviewScope, f)
	if err != nil {
		t.Fatalf("RulesFor: %v", err)
	}
	for _, file := range inv.Files {
		b1, t1 := rules.ClassifyFile(file)
		b2, t2 := rules.ClassifyFile(file)
		if b1 != b2 || !reflect.DeepEqual(t1, t2) {
			t.Errorf("%s: 规则分类不确定:%q/%v vs %q/%v", file.Path, b1, t1, b2, t2)
		}
	}
}

// AC2:scope 模式语义(裸条目覆盖子树;* 单段;** 零或多段;段内 * 后缀)。
func TestScopePatternSemantics(t *testing.T) {
	cases := []struct {
		pattern, path string
		want          bool
	}{
		{"internal", "internal/app/x.go", true},
		{"internal", "internal", true},
		{"internal", "internalx/y.go", false},
		{"internal/**", "internal/x.go", true},
		{"internal/*/x.go", "internal/a/x.go", true},
		{"internal/*/x.go", "internal/a/b/x.go", false},
		{"**/*.md", "a/b/c.md", true},
		{"**/*.md", "c.md", true},
		{"**/*.md", "a/b/c.txt", false},
		{"*_test.go", "x_test.go", true},
		{"*_test.go", "test/x_test.go", false},
		{"vendor", "vendor/x.go", true},
		{"a//b", "a/b", false},
		{"/abs", "abs", false},
	}
	for _, tc := range cases {
		if got := matchPattern(tc.pattern, tc.path); got != tc.want {
			t.Errorf("matchPattern(%q, %q) = %v, want %v", tc.pattern, tc.path, got, tc.want)
		}
	}
}

// AC2/§19.3:删除 required evidence 文件仍被记账为 acceptance_evidence;
// protected 文件被改名(从 protected 路径改出)时 protected 事实优先,
// 且 ProtectedTouched 记录 MODIFIED 事实与 expected hash。
func TestProtectedAndDeletedEvidenceAccounted(t *testing.T) {
	hash := "sha256:" + strings.Repeat("ab", 32)
	f := reviewContract(t, func(c *verification.Contract) {
		acc := &c.Acceptances[0]
		b := &acc.EvidenceBindings[0]
		b.Protection = &verification.Protection{Mode: verification.ProtectionImmutable, BaselineHash: hash}
	})
	rules, err := RulesFor(reviewScope, f)
	if err != nil {
		t.Fatalf("RulesFor: %v", err)
	}
	if len(rules.Protected) != 1 || rules.Protected[0].Path != "internal/app/app_test.go" {
		t.Fatalf("protected 规则推导错误:%+v", rules.Protected)
	}
	if len(rules.Evidence) != 1 || rules.Evidence[0].Path != "internal/app/api_test.go" {
		t.Fatalf("evidence 规则推导错误:%+v", rules.Evidence)
	}

	rep := mustBuild(t, buildInput(t, f, mustInventory(t, renameProtectedDiff), nil, nil))
	e := entryByPath(t, rep, "internal/app/moved_test.go")
	if e.Bucket != BucketProtectedEvidence {
		t.Errorf("改名的 protected 文件 bucket = %q, want %q", e.Bucket, BucketProtectedEvidence)
	}
	if !hasTag(e.Tags, TagProtected) || !hasTag(e.Tags, TagBindingPrefix+"ev-1") {
		t.Errorf("改名 protected 文件缺少 protected/binding tags:%v", e.Tags)
	}
	if len(rep.ProtectedTouched) != 1 {
		t.Fatalf("ProtectedTouched = %+v, want 恰好 1 条", rep.ProtectedTouched)
	}
	pt := rep.ProtectedTouched[0]
	if pt.Path != "internal/app/app_test.go" || pt.BindingID != "ev-1" ||
		pt.AcceptanceID != "AC-1" || pt.ExpectedHash != hash {
		t.Errorf("ProtectedTouched = %+v, want app_test.go/ev-1/AC-1/%s", pt, hash)
	}

	// 删除 required evidence 文件仍是 acceptance_evidence(不是「消失」)。
	repDel := mustBuild(t, buildInput(t, f, mustInventory(t, deleteEvidenceDiff), nil, nil))
	if got := entryByPath(t, repDel, "internal/app/api_test.go"); got.Bucket != BucketAcceptanceEvidence {
		t.Errorf("被删除的 evidence 文件 bucket = %q, want %q", got.Bucket, BucketAcceptanceEvidence)
	}
}

// AC3:Unattributed 非空必须阻断 Approve;显式归属只能填残差,填满后
// 该阻断消失。
func TestUnattributedBlocksApprove(t *testing.T) {
	f := reviewContract(t, nil)
	// AC-2(source)由有效独立 PASS 覆盖,排除 independence 阻断的干扰。
	recs := []verification.IndependenceEvidence{
		{BindingID: "ev-2", Revision: f.Revision(), Verdicts: passingVerdicts(f, "ev-2")},
	}
	inv := mustInventory(t, testDiff)
	rep := mustBuild(t, buildInput(t, f, inv, recs, nil))

	apiHunk := hunkIDByPath(t, rep, "internal/app/api.go")
	if got := entryByPath(t, rep, "internal/app/api.go").Bucket; got != BucketUnattributed {
		t.Fatalf("api.go bucket = %q, want unattributed(规则无法归账)", got)
	}
	if rep.UnattributedCount() != 1 || len(rep.UnattributedEntries()) != 1 {
		t.Fatalf("UnattributedCount = %d/%d, want 1", rep.UnattributedCount(), len(rep.UnattributedEntries()))
	}
	if !rep.ApproveBlocked() {
		t.Fatal("Unattributed 非空时 ApproveBlocked = false(违反 §19.2)")
	}
	var found bool
	for _, b := range rep.Blockers {
		if b.Kind == BlockerUnattributed && b.HunkID == apiHunk && b.Path == "internal/app/api.go" {
			found = true
		}
	}
	if !found {
		t.Fatalf("阻断项缺少 api.go hunk(%s):%+v", apiHunk, rep.Blockers)
	}

	// 显式归属填残差:api.go 归因到 AC-1,阻断随之消失。
	attrs := []Attribution{{HunkID: apiHunk, AcceptanceIDs: []string{"AC-1"}, Tags: []string{"risk:auth"}}}
	rep2 := mustBuild(t, buildInput(t, f, inv, recs, attrs))
	if rep2.UnattributedCount() != 0 {
		t.Fatalf("归属后 UnattributedCount = %d, want 0", rep2.UnattributedCount())
	}
	if rep2.ApproveBlocked() {
		t.Fatalf("归属后仍有阻断:%+v", rep2.Blockers)
	}
	e := entryByPath(t, rep2, "internal/app/api.go")
	if e.Bucket != BucketAcceptanceCode || e.ByRule {
		t.Errorf("归属后 bucket/ByRule = %q/%v, want acceptance_code/false", e.Bucket, e.ByRule)
	}
	if !hasTag(e.Tags, TagACPrefix+"AC-1") || !hasTag(e.Tags, "risk:auth") {
		t.Errorf("归属后缺少 tags:%v", e.Tags)
	}
	for _, ar := range rep2.Acceptances {
		if ar.AcceptanceID != "AC-1" {
			continue
		}
		// AC-1 相关代码 = evidence 路径命中(app_test.go)+ 显式归属(api.go)。
		if len(ar.RelatedHunks) != 2 {
			t.Errorf("AC-1 RelatedHunks = %v, want 2 条(evidence 命中 + 显式归属)", ar.RelatedHunks)
		}
		found := false
		for _, id := range ar.RelatedHunks {
			if id == apiHunk {
				found = true
			}
		}
		if !found {
			t.Errorf("AC-1 RelatedHunks 缺少归属的 api.go hunk %s:%v", apiHunk, ar.RelatedHunks)
		}
	}
}

// 显式归属的边界:不能改写规则事实、不能指向不存在的目标/契约外条款、
// 不能重复归属或缺验收归因;零 hunk 条目可按路径归属。
func TestAttributionValidation(t *testing.T) {
	f := reviewContract(t, nil)
	inv := mustInventory(t, testDiff)
	base := buildInput(t, f, inv, nil, nil)
	var apiHunk HunkID
	for _, file := range inv.Files {
		if file.Path == "internal/app/api.go" {
			apiHunk = file.Hunks[0].ID
		}
	}

	cases := []struct {
		name string
		attr Attribution
	}{
		{"改写规则归账的 evidence 条目", Attribution{HunkID: hunkIDByPath(t, mustBuild(t, base), "internal/app/app_test.go"), AcceptanceIDs: []string{"AC-1"}}},
		{"改写规则归账的 scope 条目", Attribution{HunkID: hunkIDByPath(t, mustBuild(t, base), "vendor/x.go"), AcceptanceIDs: []string{"AC-1"}}},
		{"未知 hunk", Attribution{HunkID: "hunk-does-not-exist", AcceptanceIDs: []string{"AC-1"}}},
		{"契约外验收条款", Attribution{HunkID: HunkID(apiHunk), AcceptanceIDs: []string{"AC-99"}}},
		{"缺验收归因", Attribution{HunkID: HunkID(apiHunk)}},
	}
	for _, tc := range cases {
		input := base
		input.Attributions = []Attribution{tc.attr}
		if _, err := BuildAccounting(input); !errors.Is(err, ErrBadAttribution) {
			t.Errorf("%s: err = %v, want ErrBadAttribution", tc.name, err)
		}
	}

	// 重复归属同一目标。
	input := base
	input.Attributions = []Attribution{
		{HunkID: HunkID(apiHunk), AcceptanceIDs: []string{"AC-1"}},
		{HunkID: HunkID(apiHunk), AcceptanceIDs: []string{"AC-2"}},
	}
	if _, err := BuildAccounting(input); !errors.Is(err, ErrBadAttribution) {
		t.Errorf("重复归属: err = %v, want ErrBadAttribution", err)
	}

	// 零 hunk 条目按路径归属(空文件新增 → AC-2 的支撑资产)。
	metaBase := buildInput(t, f, mustInventory(t, metaDiff), nil, nil)
	metaBase.Attributions = []Attribution{{Path: "internal/app/empty.go", AcceptanceIDs: []string{"AC-2"}}}
	rep := mustBuild(t, metaBase)
	if rep.UnattributedCount() != 0 {
		t.Errorf("零 hunk 归属后 UnattributedCount = %d, want 0", rep.UnattributedCount())
	}
	if got := entryByPath(t, rep, "internal/app/empty.go"); got.Bucket != BucketAcceptanceCode || got.ByRule {
		t.Errorf("零 hunk 归属后 bucket/ByRule = %q/%v, want acceptance_code/false", got.Bucket, got.ByRule)
	}
}

// AC4:验收级 independence 缺口必须显式呈现 —— required 等级、覆盖结论、
// strict 状态与阻断项;覆盖满足时不产生缺口。
func TestIndependenceGapsExplicit(t *testing.T) {
	f := reviewContract(t, nil)
	inv := mustInventory(t, testDiff)

	// 无任何 independence 判定记录:AC-2(source)缺口显式存在。
	rep := mustBuild(t, buildInput(t, f, inv, nil, nil))
	var ac2 AcceptanceReview
	for _, ar := range rep.Acceptances {
		if ar.AcceptanceID == "AC-2" {
			ac2 = ar
		}
	}
	if ac2.IndependenceRequired != verification.IndependenceSource || ac2.IndependenceCovered {
		t.Fatalf("AC-2 independence = %q/covered=%v, want source/false", ac2.IndependenceRequired, ac2.IndependenceCovered)
	}
	if ac2.Strict != StrictUnmet {
		t.Errorf("AC-2 Strict = %q, want strict_unmet", ac2.Strict)
	}
	if !rep.ApproveBlocked() || !hasBlocker(rep, BlockerIndependenceGap) {
		t.Errorf("未豁免的 independence 缺口必须阻断:%v", rep.Blockers)
	}
	for _, b := range rep.Blockers {
		if b.Kind == BlockerIndependenceGap && b.AcceptanceID != "AC-2" {
			t.Errorf("缺口阻断指向错误条款 %q", b.AcceptanceID)
		}
	}
	if rep.StrictOutcome != "" || !errors.Is(rep.StrictResolution, approval.ErrWaiverRequired) {
		t.Errorf("无豁免时 StrictOutcome/StrictResolution = %q/%v, want 空值/ErrWaiverRequired", rep.StrictOutcome, rep.StrictResolution)
	}

	// AC-2 获得满足 source 独立性的有效 PASS:缺口消失,strict 满足。
	recs := []verification.IndependenceEvidence{
		{BindingID: "ev-2", Revision: f.Revision(), Verdicts: passingVerdicts(f, "ev-2")},
	}
	repOK := mustBuild(t, buildInput(t, f, inv, recs, nil))
	for _, ar := range repOK.Acceptances {
		if ar.AcceptanceID == "AC-2" {
			if !ar.IndependenceCovered || ar.Strict != StrictSatisfied {
				t.Errorf("AC-2 covered/Strict = %v/%q, want true/strict_satisfied", ar.IndependenceCovered, ar.Strict)
			}
			if !reflect.DeepEqual(ar.IndependenceSatisfiedBy, []string{"ev-2"}) {
				t.Errorf("AC-2 SatisfiedBy = %v, want [ev-2]", ar.IndependenceSatisfiedBy)
			}
		}
	}
	if hasBlocker(repOK, BlockerIndependenceGap) {
		t.Errorf("independence 已覆盖不应有缺口阻断:%+v", repOK.Blockers)
	}
	if repOK.StrictOutcome != approval.StrictSatisfied || repOK.StrictResolution != nil {
		t.Errorf("StrictOutcome/StrictResolution = %q/%v, want strict_satisfied/nil", repOK.StrictOutcome, repOK.StrictResolution)
	}
}

// AC5:strict_waived 绝不能显示为 strict_satisfied;豁免失效时按未满足
// 处理并阻断,绝不静默折算。
func TestStrictWaivedNeverSatisfied(t *testing.T) {
	f := reviewContract(t, nil)
	inv := mustInventory(t, testDiff)
	// 把唯一的残差 hunk 归因掉,使 independence 是唯一的阻断来源。
	attrs := []Attribution{{HunkID: hunkIDByPath(t, mustBuild(t, buildInput(t, f, inv, nil, nil)), "internal/app/api.go"), AcceptanceIDs: []string{"AC-1"}}}

	waiver, err := approval.NewWaiver(approval.WaiverSpec{
		Category:   "verification",
		Gaps:       []approval.CoverageGap{{AcceptanceID: "AC-2"}},
		ReasonCode: "owner_accepts_runtime_loss",
		AcceptedBy: "owner",
		AcceptedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("NewWaiver: %v", err)
	}

	// policy 允许:结论是 strict_waived,显示状态互斥于 strict_satisfied。
	input := buildInput(t, f, inv, nil, attrs)
	input.WaiverPolicy = approval.WaiverPolicy{Allowed: true}
	input.Waiver = waiver
	rep := mustBuild(t, input)
	if rep.StrictOutcome != approval.StrictWaived {
		t.Fatalf("StrictOutcome = %q, want strict_waived", rep.StrictOutcome)
	}
	for _, ar := range rep.Acceptances {
		if ar.AcceptanceID != "AC-2" {
			continue
		}
		if ar.Strict != StrictWaived {
			t.Errorf("AC-2 Strict = %q, want strict_waived", ar.Strict)
		}
		if ar.Strict == StrictSatisfied {
			t.Error("strict_waived 显示为 strict_satisfied(违反 AC5)")
		}
	}
	if rep.ApproveBlocked() {
		t.Fatalf("有效豁免后不应阻断:%+v", rep.Blockers)
	}

	// policy 不允许:豁免无效 → 未满足 + 阻断,不得显示为 satisfied/waived。
	inputDenied := buildInput(t, f, inv, nil, attrs)
	inputDenied.Waiver = waiver
	repDenied := mustBuild(t, inputDenied)
	if repDenied.StrictOutcome != "" || !errors.Is(repDenied.StrictResolution, approval.ErrWaiverNotPermitted) {
		t.Errorf("policy 拒绝时 StrictOutcome/StrictResolution = %q/%v", repDenied.StrictOutcome, repDenied.StrictResolution)
	}
	for _, ar := range repDenied.Acceptances {
		if ar.AcceptanceID == "AC-2" && ar.Strict != StrictUnmet {
			t.Errorf("policy 拒绝后 AC-2 Strict = %q, want strict_unmet", ar.Strict)
		}
	}
	if !hasBlocker(repDenied, BlockerIndependenceGap) || !hasBlocker(repDenied, BlockerStrictResolution) {
		t.Errorf("policy 拒绝后缺少阻断项:%+v", repDenied.Blockers)
	}

	// 豁免覆盖不完整:同样失效并阻断。
	emptyWaiver, err := approval.NewWaiver(approval.WaiverSpec{
		Category:   "verification",
		Gaps:       []approval.CoverageGap{{AcceptanceID: "AC-1"}},
		ReasonCode: "wrong_gap",
		AcceptedBy: "owner",
		AcceptedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("NewWaiver: %v", err)
	}
	inputWrong := buildInput(t, f, inv, nil, attrs)
	inputWrong.WaiverPolicy = approval.WaiverPolicy{Allowed: true}
	inputWrong.Waiver = emptyWaiver
	repWrong := mustBuild(t, inputWrong)
	if !errors.Is(repWrong.StrictResolution, approval.ErrWaiverIncomplete) {
		t.Errorf("覆盖不完整时 StrictResolution = %v, want ErrWaiverIncomplete", repWrong.StrictResolution)
	}
	for _, ar := range repWrong.Acceptances {
		if ar.AcceptanceID == "AC-2" && ar.Strict == StrictWaived {
			t.Error("失效豁免仍产生 strict_waived 显示")
		}
	}
	if !repWrong.ApproveBlocked() {
		t.Error("失效豁免必须阻断 Approve")
	}

	// 候选已 strict 满足却携带豁免:overreach 阻断,不得当作 satisfied 放行。
	recs := []verification.IndependenceEvidence{
		{BindingID: "ev-2", Revision: f.Revision(), Verdicts: passingVerdicts(f, "ev-2")},
	}
	inputOver := buildInput(t, f, inv, recs, attrs)
	inputOver.WaiverPolicy = approval.WaiverPolicy{Allowed: true}
	inputOver.Waiver = waiver
	repOver := mustBuild(t, inputOver)
	if !errors.Is(repOver.StrictResolution, approval.ErrWaiverOverreach) {
		t.Errorf("已满足却带豁免:StrictResolution = %v, want ErrWaiverOverreach", repOver.StrictResolution)
	}
	if !hasBlocker(repOver, BlockerStrictResolution) || !repOver.ApproveBlocked() {
		t.Errorf("overreach 未阻断:%+v", repOver.Blockers)
	}
	for _, ar := range repOver.Acceptances {
		if ar.AcceptanceID == "AC-2" && ar.Strict != StrictSatisfied {
			t.Errorf("已覆盖条款的 Strict = %q, want strict_satisfied(豁免不得改写覆盖事实)", ar.Strict)
		}
	}
}

// 输入形态校验:缺对象、规则非法、覆盖报告跨 revision 一律报错,
// 不产出静默残缺的报告。
func TestInputValidation(t *testing.T) {
	f := reviewContract(t, nil)
	inv := mustInventory(t, testDiff)

	if _, err := BuildAccounting(AccountingInput{Frozen: f}); !errors.Is(err, ErrNilChangeSet) {
		t.Errorf("缺 ChangeSet: err = %v, want ErrNilChangeSet", err)
	}
	input := buildInput(t, f, inv, nil, nil)
	cs := input.ChangeSet
	input.ChangeSet = nil
	if _, err := BuildAccounting(input); !errors.Is(err, ErrNilChangeSet) {
		t.Errorf("nil ChangeSet: err = %v, want ErrNilChangeSet", err)
	}
	input.ChangeSet = cs
	input.Frozen = nil
	if _, err := BuildAccounting(input); !errors.Is(err, ErrNilContract) {
		t.Errorf("nil 契约: err = %v, want ErrNilContract", err)
	}
	input.Frozen = f
	input.Rules = Rules{}
	if _, err := BuildAccounting(input); err == nil || strings.Contains(err.Error(), "review: 覆盖报告") {
		t.Errorf("空规则集应报 allowed_paths 错误,得到 %v", err)
	}
	validRules, err := RulesFor(reviewScope, f)
	if err != nil {
		t.Fatalf("RulesFor: %v", err)
	}
	input.Rules = validRules

	// 同一契约改一个 statement → 哈希不同 → 覆盖报告跨 revision。
	fOther := reviewContract(t, func(c *verification.Contract) {
		c.Acceptances[1].Statement = "AC-2 holds (v2)"
	})
	input.Frozen = fOther
	if _, err := BuildAccounting(input); !errors.Is(err, ErrReportMismatch) {
		t.Errorf("跨 revision 报告: err = %v, want ErrReportMismatch", err)
	}
}

// 空清单(候选无改动):没有任何条目与阻断;验收视图仍然完整呈现。
func TestEmptyInventory(t *testing.T) {
	f := reviewContract(t, nil)
	recs := []verification.IndependenceEvidence{
		{BindingID: "ev-2", Revision: f.Revision(), Verdicts: passingVerdicts(f, "ev-2")},
	}
	rep := mustBuild(t, buildInput(t, f, Inventory{}, recs, nil))
	if len(rep.Entries) != 0 || rep.UnattributedCount() != 0 || len(rep.ProtectedTouched) != 0 {
		t.Fatalf("空清单产生了条目:%+v", rep)
	}
	if len(rep.Acceptances) != 2 {
		t.Fatalf("验收视图条款数 = %d, want 2", len(rep.Acceptances))
	}
	if rep.ApproveBlocked() {
		t.Errorf("空清单且有覆盖时不应阻断:%+v", rep.Blockers)
	}
}

// RulesFor/Validate 的构造校验:缺 allowed、非法模式、具体 evidence 路径
// 带通配、重复条目都要报错。
func TestRulesValidation(t *testing.T) {
	f := reviewContract(t, nil)
	if _, err := RulesFor(taskcontract.Scope{}, f); err == nil {
		t.Error("空 scope 应报错(review 对象必须在合同边界内)")
	}
	r, err := RulesFor(reviewScope, f)
	if err != nil {
		t.Fatalf("RulesFor: %v", err)
	}
	bad := r
	bad.Allowed = nil
	if err := bad.Validate(); err == nil || !strings.Contains(err.Error(), "allowed_paths") {
		t.Errorf("空 allowed: err = %v", err)
	}
	bad = r
	bad.Allowed = []string{"/abs"}
	if err := bad.Validate(); err == nil {
		t.Error("绝对路径模式应报错")
	}
	bad = r
	bad.Evidence = append(bad.Evidence, EvidencePath{Path: "a/**/*.go", BindingID: "b1"})
	if err := bad.Validate(); err == nil {
		t.Error("evidence 路径带通配应报错")
	}
	bad = r
	bad.Evidence = append(bad.Evidence, EvidencePath{Path: "internal/app/api_test.go", BindingID: "ev-2"})
	if err := bad.Validate(); err == nil || !strings.Contains(err.Error(), "重复") {
		t.Errorf("重复 evidence 条目应报错: %v", err)
	}
	bad = r
	bad.Risky = append(bad.Risky, PathRule{Name: "x", Pattern: "a/**/b/**"})
	if err := bad.Validate(); err != nil {
		t.Errorf("多段 ** 合法模式不应报错: %v", err)
	}
}
