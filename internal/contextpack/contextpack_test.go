package contextpack

import (
	"errors"
	"regexp"
	"strings"
	"testing"

	"tally/internal/taskcontract"
	"tally/internal/verification"

	"gopkg.in/yaml.v3"
)

// 测试夹具:一份合法 TaskContract + 一份含 visible/held_out binding 的冻结
// VerificationContract。held_out binding 携带 store/ref 与 baseline 细节,
// 作为「不得出现在实现侧包中」的标记内容。
const (
	secretStore       = "holdout"
	secretRef         = "impl-ref-0001"
	secretFingerprint = "fp-secret-0001"
	secretBindingID   = "b-heldout"
)

func testTask() taskcontract.TaskContract {
	return taskcontract.TaskContract{
		ID:        "TASK-001",
		Profile:   taskcontract.ProfileStandard,
		Objective: "为导出 API 提供 redacted ContextPack 投影",
		Scope:     taskcontract.Scope{AllowedPaths: []string{"internal/foo"}},
		Acceptance: []taskcontract.AcceptanceClause{
			{ID: "AC-1", Statement: "导出行为符合规格"},
		},
	}
}

// frozenContract 返回测试用冻结 VerificationContract;mutate 可在 Freeze 前
// 调整契约内容。
func frozenContract(t *testing.T, mutate func(c *verification.Contract)) *verification.Frozen {
	t.Helper()
	c := &verification.Contract{
		TaskID:   "TASK-001",
		Revision: 1,
		Acceptances: []verification.Acceptance{
			{
				ID:        "AC-1",
				Statement: "导出行为符合规格",
				EvidenceBindings: []verification.Binding{
					{
						ID:               "b-visible",
						AuthoringMode:    verification.AuthoringPreCommit,
						VisibilityPolicy: verification.VisibilityVisible,
						Evidence: verification.EvidenceRef{
							Type:     verification.EvidenceTest,
							Path:     "internal/foo/foo_test.go",
							Selector: "TestFoo",
						},
					},
					{
						ID:               secretBindingID,
						AuthoringMode:    verification.AuthoringPreCommit,
						VisibilityPolicy: verification.VisibilityHeldOut,
						Evidence: verification.EvidenceRef{
							Type:  verification.EvidenceTest,
							Store: secretStore,
							Ref:   secretRef,
						},
						Baseline: &verification.Baseline{
							Expectation: &verification.BaselineExpectation{Status: "fail"},
							Observed: &verification.BaselineObserved{
								Status:      "fail",
								Class:       "assertion",
								Fingerprint: secretFingerprint,
							},
						},
					},
				},
			},
		},
		RuntimeRequirements: verification.RuntimeRequirements{
			SessionIsolation:   true,
			HeldOutUnreachable: true,
			ConformanceClass:   "verification-isolation-v1",
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

func buildPack(t *testing.T, in Input) *Pack {
	t.Helper()
	p, err := Build(in)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return p
}

// marshalPack 序列化整个包,用于断言泄漏面:任何不应出现的字符串都不得出现。
func marshalPack(t *testing.T, p *Pack) string {
	t.Helper()
	data, err := yaml.Marshal(p)
	if err != nil {
		t.Fatalf("marshal pack: %v", err)
	}
	return string(data)
}

func assertAbsent(t *testing.T, rendered string, secrets ...string) {
	t.Helper()
	for _, s := range secrets {
		if strings.Contains(rendered, s) {
			t.Errorf("实现侧包泄漏了受控内容 %q:\n%s", s, rendered)
		}
	}
}

// AC1:held-out projection 只暴露 acceptance statement 与 evidence type;
// store/ref、selector、baseline 细节、binding id 一律不得出现。
func TestHeldOutProjectionExposesStatementAndTypeOnly(t *testing.T) {
	p := buildPack(t, Input{Task: testTask(), Verification: frozenContract(t, nil)})

	if len(p.Verification) != 1 {
		t.Fatalf("projection acceptance 数 = %d, want 1", len(p.Verification))
	}
	acc := p.Verification[0]
	if acc.ID != "AC-1" || acc.Statement != "导出行为符合规格" {
		t.Errorf("acceptance 语义丢失: %+v", acc)
	}
	if len(acc.Evidence) != 2 {
		t.Fatalf("evidence 投影数 = %d, want 2", len(acc.Evidence))
	}

	var heldOut *EvidenceProjection
	for i := range acc.Evidence {
		ev := &acc.Evidence[i]
		if ev.Visibility != verification.VisibilityHeldOut {
			continue
		}
		heldOut = ev
		if ev.Type != verification.EvidenceTest {
			t.Errorf("held_out 投影 evidence type = %q, want %q", ev.Type, verification.EvidenceTest)
		}
		// §8.4.3 allowlist 之外的字段必须全部为零值。
		if ev.BindingID != "" || ev.Path != "" || ev.Selector != "" || ev.Validator != "" || ev.Store != "" || ev.Ref != "" {
			t.Errorf("held_out 投影暴露了定位信息: %+v", ev)
		}
	}
	if heldOut == nil {
		t.Fatalf("projection 中没有 held_out evidence 条目")
	}

	rendered := marshalPack(t, p)
	assertAbsent(t, rendered, secretStore, secretRef, secretFingerprint, secretBindingID)
}

// AC2:visible binding 按 visibility policy 保持可用(标识与定位信息完整)。
func TestVisibleEvidenceRemainsAvailable(t *testing.T) {
	p := buildPack(t, Input{Task: testTask(), Verification: frozenContract(t, nil)})

	acc := p.Verification[0]
	var visible *EvidenceProjection
	for i := range acc.Evidence {
		if acc.Evidence[i].Visibility == verification.VisibilityVisible {
			visible = &acc.Evidence[i]
		}
	}
	if visible == nil {
		t.Fatalf("projection 中没有 visible evidence 条目")
	}
	if visible.BindingID != "b-visible" {
		t.Errorf("visible binding id = %q, want b-visible", visible.BindingID)
	}
	if visible.Path != "internal/foo/foo_test.go" || visible.Selector != "TestFoo" {
		t.Errorf("visible 定位信息丢失: %+v", visible)
	}

	found := false
	for _, s := range p.Included {
		if s.Source == SectionVerification && s.Ref == "AC-1/b-visible" && s.Reason == ReasonVisibleBinding {
			found = true
		}
	}
	if !found {
		t.Errorf("Included 记录缺少 visible binding 条目: %+v", p.Included)
	}
}

// AC3:历史失败/知识输入按 binding provenance 过滤 —— held_out provenance 与
// 无法证实 provenance 的条目被剔除;记录面不携带被剔除条目的 ref,也不回泄内容。
func TestNotesFilteredByBindingProvenance(t *testing.T) {
	in := Input{
		Task:         testTask(),
		Verification: frozenContract(t, nil),
		Failures: []FailureInput{
			{Ref: "FE-1", Summary: "held-out baseline 失败细节 secret-detail", Provenance: Provenance{BindingID: secretBindingID}},
			{Ref: "FE-2", Summary: "visible 测试在 CI 上抖动", Provenance: Provenance{BindingID: "b-visible"}},
			{Ref: "FE-3", Summary: "与验证证据无关的一般部署问题"},
			{Ref: "FE-4", Summary: "声称来自未知 binding", Provenance: Provenance{BindingID: "b-nope"}},
		},
		Knowledge: []KnowledgeInput{
			{Ref: "KN-1", Summary: "连接池大小经验", Prov: Provenance{BindingID: "b-visible"}},
			{Ref: "KN-2", Summary: "held-out 推导线索 secret-knowledge", Prov: Provenance{BindingID: secretBindingID}},
		},
	}
	p := buildPack(t, in)

	wantFailures := []string{"FE-2", "FE-3"}
	if len(p.HistoricalFailures) != len(wantFailures) {
		t.Fatalf("HistoricalFailures = %+v, want refs %v", p.HistoricalFailures, wantFailures)
	}
	for i, ref := range wantFailures {
		if p.HistoricalFailures[i].Ref != ref {
			t.Errorf("HistoricalFailures[%d].Ref = %q, want %q", i, p.HistoricalFailures[i].Ref, ref)
		}
	}
	if len(p.Knowledge) != 1 || p.Knowledge[0].Ref != "KN-1" {
		t.Fatalf("Knowledge = %+v, want 仅 KN-1", p.Knowledge)
	}

	// 排除记录:held_out / 未证 provenance 一律不携带 ref。
	for _, s := range p.Excluded {
		if s.Source != SectionHistoricalFailure && s.Source != SectionKnowledge {
			continue
		}
		if s.Ref != "" {
			t.Errorf("被排除条目的记录携带 ref %q(泄漏面): %+v", s.Ref, s)
		}
	}
	var heldOutExcluded, unresolvedExcluded bool
	for _, s := range p.Excluded {
		switch s.Reason {
		case ReasonHeldOutProvenance:
			heldOutExcluded = true
		case ReasonUnresolvedProvenance:
			unresolvedExcluded = true
		}
	}
	if !heldOutExcluded || !unresolvedExcluded {
		t.Errorf("Excluded 记录不全: %+v", p.Excluded)
	}

	rendered := marshalPack(t, p)
	assertAbsent(t, rendered,
		"secret-detail", "secret-knowledge",
		"FE-1", "FE-4", "KN-2",
		secretRef, secretFingerprint,
	)
}

// AC4:context_hash 在 redaction/filter 之后计算 —— held-out 内容(store/ref、
// baseline 指纹)变化不影响哈希;实现侧可见内容变化必须改变哈希;同一输入
// 哈希确定。
func TestContextHashComputedAfterRedaction(t *testing.T) {
	hashRe := regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

	base := buildPack(t, Input{Task: testTask(), Verification: frozenContract(t, nil)})
	if !hashRe.MatchString(base.Hash()) {
		t.Fatalf("context_hash 形态非法: %q", base.Hash())
	}

	// 同一输入重复 Build:哈希稳定。
	again := buildPack(t, Input{Task: testTask(), Verification: frozenContract(t, nil)})
	if again.Hash() != base.Hash() {
		t.Errorf("同一输入哈希不稳定: %q vs %q", base.Hash(), again.Hash())
	}

	// 引用输入顺序不同(排序规范化后内容相同):哈希相同。
	shuffled := buildPack(t, Input{
		Task:         testTask(),
		Verification: frozenContract(t, nil),
		ProductDocs:  []string{"docs/b.md", "docs/a.md"},
	})
	ordered := buildPack(t, Input{
		Task:         testTask(),
		Verification: frozenContract(t, nil),
		ProductDocs:  []string{"docs/a.md", "docs/b.md"},
	})
	if shuffled.Hash() != ordered.Hash() {
		t.Errorf("输入顺序影响哈希: %q vs %q", shuffled.Hash(), ordered.Hash())
	}

	// held-out 内容变化(store/ref、baseline 指纹):哈希必须不变。
	heldOutChanged := buildPack(t, Input{Task: testTask(), Verification: frozenContract(t, func(c *verification.Contract) {
		b := &c.Acceptances[0].EvidenceBindings[1]
		b.Evidence.Ref = "impl-ref-9999"
		b.Baseline.Observed.Fingerprint = "fp-secret-9999"
	})})
	if heldOutChanged.Hash() != base.Hash() {
		t.Errorf("held-out 内容变化影响了 context_hash(哈希未在 redaction 后计算): %q vs %q", base.Hash(), heldOutChanged.Hash())
	}

	// 实现侧可见内容变化(acceptance statement):哈希必须变化。
	visibleChanged := buildPack(t, Input{Task: testTask(), Verification: frozenContract(t, func(c *verification.Contract) {
		c.Acceptances[0].Statement = "导出行为符合修订后的规格"
	})})
	if visibleChanged.Hash() == base.Hash() {
		t.Errorf("可见内容变化未改变 context_hash")
	}

	// 可见小节内容变化(文档引用):哈希必须变化。
	withDoc := buildPack(t, Input{Task: testTask(), Verification: frozenContract(t, nil), ProductDocs: []string{"docs/x.md"}})
	if withDoc.Hash() == base.Hash() {
		t.Errorf("可见文档小节变化未改变 context_hash")
	}
}

// Build 输入校验:契约缺失/任务不一致/合同非法/条目缺字段都确定性报错。
func TestBuildInputValidation(t *testing.T) {
	if _, err := Build(Input{Task: testTask()}); !errors.Is(err, ErrNilContract) {
		t.Errorf("nil 契约应返回 ErrNilContract, got %v", err)
	}
	if _, err := RedactVerification(nil); !errors.Is(err, ErrNilContract) {
		t.Errorf("RedactVerification(nil) 应返回 ErrNilContract, got %v", err)
	}

	mismatched := testTask()
	mismatched.ID = "TASK-002"
	if _, err := Build(Input{Task: mismatched, Verification: frozenContract(t, nil)}); err == nil {
		t.Errorf("任务 id 不一致应报错")
	}

	invalid := testTask()
	invalid.Objective = ""
	if _, err := Build(Input{Task: invalid, Verification: frozenContract(t, nil)}); err == nil {
		t.Errorf("非法 TaskContract 应报错")
	}

	if _, err := Build(Input{Task: testTask(), Verification: frozenContract(t, nil),
		Failures: []FailureInput{{Ref: "FE-1"}}}); err == nil {
		t.Errorf("缺少 summary 的失败条目应报错")
	}
	if _, err := Build(Input{Task: testTask(), Verification: frozenContract(t, nil),
		Knowledge: []KnowledgeInput{{Ref: "KN-1", Summary: "ok", Prov: Provenance{BindingID: "b-visible"}}}}); err != nil {
		t.Errorf("合法知识条目不应报错: %v", err)
	}
}

// 预算:每节条目数上限生效,超限条目登记 Excluded 并携带原 ref(非 held-out
// 排除,记录面允许保留引用)。
func TestBudgetTruncatesSectionsWithRecords(t *testing.T) {
	in := Input{
		Task:         testTask(),
		Verification: frozenContract(t, nil),
		Budget:       Budget{MaxItemsPerSection: 2},
		ProductDocs:  []string{"docs/1.md", "docs/2.md", "docs/3.md"},
		Failures: []FailureInput{
			{Ref: "FE-A", Summary: "a"},
			{Ref: "FE-B", Summary: "b"},
			{Ref: "FE-C", Summary: "c"},
		},
	}
	p := buildPack(t, in)

	if len(p.ProductDocs) != 2 || p.ProductDocs[0] != "docs/1.md" || p.ProductDocs[1] != "docs/2.md" {
		t.Errorf("ProductDocs 预算截断错误: %v", p.ProductDocs)
	}
	if len(p.HistoricalFailures) != 2 {
		t.Errorf("HistoricalFailures 预算截断错误: %v", p.HistoricalFailures)
	}
	budgetExcluded := 0
	for _, s := range p.Excluded {
		if s.Reason == ReasonBudget {
			budgetExcluded++
			if s.Ref == "" {
				t.Errorf("预算排除记录缺少 ref(非 held-out 排除应可引用): %+v", s)
			}
		}
	}
	if budgetExcluded != 2 {
		t.Errorf("预算排除记录数 = %d, want 2(每节各一): %+v", budgetExcluded, p.Excluded)
	}
	if p.Budget != (Budget{MaxItemsPerSection: 2}) {
		t.Errorf("Budget 未记录到包中: %+v", p.Budget)
	}
}
