// feedback.go 安全 RepairFeedback 构造器测试(P2-04)。
//
// 覆盖:
//   - AC1:反馈可识别 acceptance/gate、类型化失败类与安全诊断摘要;
//     谱系(run/changeset id,修复尝试父链接)完整;
//   - AC2:held-out 材料(路径/内容/binding id/内容与输出摘要/oracle 词形)
//     在投影值面(JSON、%+v)与类型面上均不存在;held-out 只经
//     RedactHoldoutOutcome 组合进入,可见证据类型无 store/ref 字段;
//     sha256 摘要形态在散文字段中被显式标记替换,绝不透传;
//   - AC3:反馈规模有界 —— 散文按 rune 截断带显式 [truncated] 标记,
//     身份超长/条目超量一律类型化报错(身份绝不被截断改写);
//   - AC4:不合法/不安全输入(非 repairable 类、双重主体、缺摘要、未知
//     证据类型、非法 pass|fail 类别、任务身份矛盾)→ 类型化 error + 零值,
//     粘滞不吞错,绝不产出部分反馈;
//   - 确定性:相同输入恒得相同反馈(含 Build 幂等与输出隔离)。
package closedloop

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"tally/internal/changeset"
	"tally/internal/verification"
)

// ---- 固件 ----

const (
	fbTask     = "T-P2-04"
	fbCSFailed = changeset.ChangeSetID("cs-failed-1")
)

var (
	// 形如真实 held-out 判定记录里的敏感值(摘要 = sha256:<64hex>)。
	fbHiddenDigest    = "sha256:" + strings.Repeat("ab", 32)
	fbHiddenOutDigest = "sha256:" + strings.Repeat("cd", 32)
	fbContractHash    = "sha256:" + strings.Repeat("ef", 32)
)

// fbHoldoutRecord 构造一条含有全部敏感字段的 held-out 判定事实(Control/
// Review 通道词形);其投影经 RedactHoldoutRecord(真实出口)进入反馈。
func fbHoldoutRecord() verification.HoldoutVerdictRecord {
	return verification.HoldoutVerdictRecord{
		TaskID:           fbTask,
		ContractRevision: 1,
		ContractHash:     fbContractHash,
		AcceptanceID:     "AC-1",
		BindingID:        "b-hidden",
		CandidateID:      string(fbCSFailed),
		ContextHash:      "ctx-hash-should-not-leak",
		EvidenceType:     verification.EvidenceTest,
		ContentDigest:    fbHiddenDigest,
		Status:           verification.EvidenceStatusFail,
		Repetitions:      1,
		Failures:         1,
		OutputDigests:    []string{fbHiddenOutDigest},
		CompletedAt:      fixedNow,
	}
}

// fbAcceptanceFeedback 构造一份最小合法反馈(acceptance 主体)。
func fbAcceptanceFeedback(t *testing.T) RepairFeedback {
	t.Helper()
	fb, err := NewFeedbackBuilder(fbTask, 1, fbCSFailed, "verify-run-7").
		ForAcceptance("AC-1", "the widget counts occurrences").
		WithFailure(FailureRepairable).
		WithAffectedBehavior("counting drops occurrences above the batch limit").
		WithDiagnosticSummary("a visible assertion failed while counting").
		Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return fb
}

// ---- AC1:主体 / 失败类 / 摘要 / 谱系可识别 ----

func TestFeedbackIdentifiesSubjectClassSummaryAndLineage(t *testing.T) {
	fb := fbAcceptanceFeedback(t)

	if fb.TaskID != fbTask || fb.Attempt != 1 || fb.ChangeSet != fbCSFailed {
		t.Fatalf("谱系身份不符: %+v", fb)
	}
	if fb.RunID != RunIDFor(fbTask, 1) {
		t.Fatalf("RunID = %q, 期望确定性推导 %q", string(fb.RunID), string(RunIDFor(fbTask, 1)))
	}
	if fb.ParentRun != "" {
		t.Fatalf("首次尝试 ParentRun 应为空,得到 %q", string(fb.ParentRun))
	}
	if fb.VerifyRunRef != "verify-run-7" {
		t.Fatalf("VerifyRunRef = %q", fb.VerifyRunRef)
	}
	if fb.Subject.AcceptanceID != "AC-1" || fb.Subject.Gate != "" {
		t.Fatalf("主体应为 acceptance: %+v", fb.Subject)
	}
	if !strings.Contains(fb.Subject.Statement, "widget counts") {
		t.Fatalf("语义文本缺失: %+v", fb.Subject)
	}
	if fb.Failure != FailureRepairable {
		t.Fatalf("失败类 = %q, 期望 repairable", string(fb.Failure))
	}
	if fb.DiagnosticSummary == "" {
		t.Fatal("安全诊断摘要缺失")
	}

	// gate 主体:名称可识别,acceptance 侧为空。
	gb, err := NewFeedbackBuilder(fbTask, 1, fbCSFailed, "").
		ForGate("build").
		WithFailure(FailureRepairable).
		WithDiagnosticSummary("required gate build exited non-zero").
		Build()
	if err != nil {
		t.Fatalf("gate Build: %v", err)
	}
	if gb.Subject.Gate != "build" || gb.Subject.AcceptanceID != "" || gb.Subject.Statement != "" {
		t.Fatalf("gate 主体不符: %+v", gb.Subject)
	}

	// 修复尝试(≥2):父 Run 由 (task, attempt-1) 确定性推导(谱系父链接)。
	rb, err := NewFeedbackBuilder(fbTask, 2, fbCSFailed, "").
		ForGate("build").
		WithFailure(FailureRepairable).
		WithDiagnosticSummary("repair attempt failed the same gate").
		Build()
	if err != nil {
		t.Fatalf("repair Build: %v", err)
	}
	if rb.RunID != RunIDFor(fbTask, 2) || rb.ParentRun != RunIDFor(fbTask, 1) {
		t.Fatalf("修复尝试谱系不符: run=%q parent=%q", string(rb.RunID), string(rb.ParentRun))
	}
}

// ---- AC2:类型面 allowlist(反射守卫) ----

// TestFeedbackTypeSurfaceIsAllowlisted 守卫投影类型面:任何新增字段都必须
// 显式进入本测试(与 RedactHoldoutRecord 的 allowlist 纪律同构),且类型树
// 中不存在 map/chan/func/interface/unsafe 指针/字节切片等原始透传通道。
func TestFeedbackTypeSurfaceIsAllowlisted(t *testing.T) {
	fieldNames := func(typ reflect.Type) []string {
		names := make([]string, 0, typ.NumField())
		for i := 0; i < typ.NumField(); i++ {
			names = append(names, typ.Field(i).Name)
		}
		return names
	}
	sameNames := func(typ reflect.Type, want []string) {
		t.Helper()
		got := fieldNames(typ)
		if len(got) != len(want) {
			t.Fatalf("%s 字段数 = %d(%v), 期望 %v(allowlist 面被扩大或收窄)", typ.Name(), len(got), got, want)
		}
		for i, w := range want {
			if got[i] != w {
				t.Fatalf("%s 字段[%d] = %q, 期望 %q(全集 %v)", typ.Name(), i, got[i], w, want)
			}
		}
	}

	sameNames(reflect.TypeOf(RepairFeedback{}), []string{
		"TaskID", "Attempt", "RunID", "ParentRun", "ChangeSet", "VerifyRunRef",
		"Subject", "Failure",
		"AffectedBehavior", "DiagnosticSummary",
		"Evidence", "Holdout",
	})
	sameNames(reflect.TypeOf(FeedbackSubject{}), []string{"AcceptanceID", "Statement", "Gate"})
	sameNames(reflect.TypeOf(VisibleEvidence{}), []string{
		"AcceptanceID", "BindingID", "EvidenceType", "Path", "Selector", "Validator", "Status",
	})
	sameNames(reflect.TypeOf(HoldoutFailure{}), []string{"Outcome", "Status"})

	// 可见证据类型不得携带 held-out 定位(store/ref 在类型层不可表达)。
	ve := reflect.TypeOf(VisibleEvidence{})
	for i := 0; i < ve.NumField(); i++ {
		if name := ve.Field(i).Name; name == "Store" || name == "Ref" {
			t.Fatalf("VisibleEvidence 出现 held-out 定位字段 %q", name)
		}
	}

	// 类型树禁手:无 map/chan/func/interface/unsafe 指针/字节序列。
	var walk func(typ reflect.Type, path string)
	seen := map[reflect.Type]bool{}
	walk = func(typ reflect.Type, path string) {
		if typ == nil || seen[typ] {
			return
		}
		seen[typ] = true
		switch typ.Kind() {
		case reflect.Map, reflect.Chan, reflect.Func, reflect.Interface, reflect.UnsafePointer:
			t.Fatalf("%s 的类型 %v 是原始透传通道(kind %s)", path, typ, typ.Kind())
		case reflect.Slice, reflect.Array:
			if typ.Elem().Kind() == reflect.Uint8 {
				t.Fatalf("%s 的类型 %v 是字节序列([]byte 形态)", path, typ)
			}
			walk(typ.Elem(), path+"[]")
		case reflect.Struct:
			for i := 0; i < typ.NumField(); i++ {
				walk(typ.Field(i).Type, path+"."+typ.Field(i).Name)
			}
		}
	}
	walk(reflect.TypeOf(RepairFeedback{}), "RepairFeedback")
}

// ---- AC2:值面泄漏扫描(held-out 经真实投影出口组合) ----

func TestFeedbackHeldOutProjectionLeaksNothing(t *testing.T) {
	rec := fbHoldoutRecord()

	// 扫描有效性自检:敏感值确实存在于 Control/Review 侧原始记录中。
	raw, err := json.Marshal(rec)
	if err != nil {
		t.Fatalf("marshal record: %v", err)
	}
	for _, secret := range []string{"b-hidden", fbHiddenDigest, fbHiddenOutDigest, fbContractHash, "ctx-hash-should-not-leak"} {
		if !strings.Contains(string(raw), secret) {
			t.Fatalf("自检失败:原始记录不含 %q,泄漏扫描无意义", secret)
		}
	}

	fb, err := NewFeedbackBuilder(fbTask, 1, fbCSFailed, "verify-run-7").
		ForAcceptance("AC-1", "the widget counts occurrences").
		WithFailure(FailureRepairable).
		WithDiagnosticSummary("held-out category indicates the counting behavior regressed").
		AddHoldoutOutcome(verification.RedactHoldoutRecord(rec), verification.EvidenceStatusFail).
		AddVisibleEvidence(VisibleEvidence{
			AcceptanceID: "AC-1",
			BindingID:    "b-visible",
			EvidenceType: verification.EvidenceTest,
			Path:         "internal/widget/widget_test.go",
			Selector:     "TestWidgetCounts",
			Status:       verification.EvidenceStatusFail,
		}).
		Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	rendered := fmt.Sprintf("%+v", fb)
	jsonRaw, err := json.Marshal(fb)
	if err != nil {
		t.Fatalf("marshal feedback: %v", err)
	}
	jsonRendered := string(jsonRaw)

	// 结构性排除词形:binding id、hidden 内容/输出/契约摘要、context 哈希、
	// 原始记录字段名 —— 投影的 JSON 与文本形态都不得出现。
	for _, secret := range []string{
		"b-hidden", fbHiddenDigest, fbHiddenOutDigest, fbContractHash,
		"ctx-hash-should-not-leak",
		"ContentDigest", "OutputDigests", "ContextHash", "ContractHash",
	} {
		if strings.Contains(rendered, secret) {
			t.Fatalf("反馈文本形态泄漏 %q: %s", secret, rendered)
		}
		if strings.Contains(jsonRendered, secret) {
			t.Fatalf("反馈 JSON 泄漏 %q: %s", secret, jsonRendered)
		}
	}
	// held-out 条目只携带类别 + allowlist 投影:类别在,定位/摘要不在。
	if !strings.Contains(jsonRendered, `"Status":"fail"`) {
		t.Fatalf("held-out 类别缺失: %s", jsonRendered)
	}
	// 可见证据(本就实现侧可见)按 allowlist 在场 —— 证明扫描看得见字段。
	for _, allowed := range []string{"b-visible", "internal/widget/widget_test.go", "TestWidgetCounts", "AC-1"} {
		if !strings.Contains(jsonRendered, allowed) {
			t.Fatalf("可见证据 %q 应在反馈中: %s", allowed, jsonRendered)
		}
	}
}

// ---- AC2/AC3:散文管线 —— 摘要脱敏 + 界界截断,显式标记 ----

func TestFeedbackProseRedactsDigestsAndBounds(t *testing.T) {
	digest := "sha256:" + strings.Repeat("99", 32)
	b, err := NewFeedbackBuilder(fbTask, 1, fbCSFailed, "").
		ForAcceptance("AC-1", "statement digest sha256:"+strings.Repeat("77", 32)+" tail").
		WithFailure(FailureRepairable).
		WithAffectedBehavior("behavior digest " + digest + " end").
		WithDiagnosticSummary("summary digest " + digest + " end").
		Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	for name, field := range map[string]string{
		"statement":  b.Subject.Statement,
		"behavior":   b.AffectedBehavior,
		"diagnostic": b.DiagnosticSummary,
	} {
		if strings.Contains(field, digest) || strings.Contains(field, strings.Repeat("99", 32)) || strings.Contains(field, strings.Repeat("77", 32)) {
			t.Fatalf("%s 泄漏摘要形态: %q", name, field)
		}
		if !strings.Contains(field, feedbackDigestRedacted) {
			t.Fatalf("%s 缺少显式脱敏标记: %q", name, field)
		}
	}

	// 截断:超界散文按 rune 截断并追加显式标记(身份字段不适用截断)。
	long := strings.Repeat("x", 2*MaxFeedbackDiagnosticLen)
	c, err := NewFeedbackBuilder(fbTask, 1, fbCSFailed, "").
		ForGate("build").
		WithFailure(FailureRepairable).
		WithDiagnosticSummary(long).
		Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	wantLen := MaxFeedbackDiagnosticLen + len([]rune(feedbackTruncatedMark))
	if got := []rune(c.DiagnosticSummary); len(got) != wantLen {
		t.Fatalf("诊断摘要 rune 数 = %d, 期望 %d(有界)", len(got), wantLen)
	}
	if !strings.HasSuffix(c.DiagnosticSummary, feedbackTruncatedMark) {
		t.Fatalf("诊断摘要缺少截断标记: %q", c.DiagnosticSummary)
	}
	if !strings.HasPrefix(c.DiagnosticSummary, strings.Repeat("x", MaxFeedbackDiagnosticLen)) {
		t.Fatal("截断未保留确定性前缀")
	}
}

// ---- AC3:规模有界 —— 条目超量/身份超长显式报错 ----

func TestFeedbackSizeBounds(t *testing.T) {
	fact := VisibleEvidence{
		AcceptanceID: "AC-1",
		BindingID:    "b-visible",
		EvidenceType: verification.EvidenceTest,
		Status:       verification.EvidenceStatusFail,
	}
	// 恰好上限可构造;超界报错(可见证据)。
	b := NewFeedbackBuilder(fbTask, 1, fbCSFailed, "").
		ForAcceptance("AC-1", "statement").
		WithFailure(FailureRepairable).
		WithDiagnosticSummary("summary")
	for i := 0; i < MaxFeedbackEvidenceFacts; i++ {
		f := fact
		f.BindingID = fmt.Sprintf("b-visible-%02d", i)
		b.AddVisibleEvidence(f)
	}
	if _, err := b.Build(); err != nil {
		t.Fatalf("上限内构造失败: %v", err)
	}
	_, err := b.AddVisibleEvidence(fact).Build()
	if !errors.Is(err, ErrFeedbackBoundExceeded) {
		t.Fatalf("可见证据超界应报 ErrFeedbackBoundExceeded,得到 %v", err)
	}

	// held-out 投影条数有界。
	outcome := verification.RedactedHoldoutOutcome{
		TaskID:       fbTask,
		AcceptanceID: "AC-1",
		EvidenceType: verification.EvidenceTest,
		CompletedAt:  fixedNow,
	}
	h := NewFeedbackBuilder(fbTask, 1, fbCSFailed, "").
		ForAcceptance("AC-1", "statement").
		WithFailure(FailureRepairable).
		WithDiagnosticSummary("summary")
	for i := 0; i < MaxFeedbackHoldoutFacts; i++ {
		h.AddHoldoutOutcome(outcome, verification.EvidenceStatusFail)
	}
	if _, err := h.Build(); err != nil {
		t.Fatalf("held-out 上限内构造失败: %v", err)
	}
	_, err = h.AddHoldoutOutcome(outcome, verification.EvidenceStatusFail).Build()
	if !errors.Is(err, ErrFeedbackBoundExceeded) {
		t.Fatalf("held-out 超界应报 ErrFeedbackBoundExceeded,得到 %v", err)
	}

	// 身份超长显式报错,身份绝不被截断(acceptance id / gate 名 / 谱系)。
	long := strings.Repeat("i", MaxFeedbackIdentityLen+1)
	cases := []struct {
		name  string
		build func() *FeedbackBuilder
	}{
		{"acceptance id 超长", func() *FeedbackBuilder {
			return NewFeedbackBuilder(fbTask, 1, fbCSFailed, "").
				ForAcceptance(long, "statement")
		}},
		{"gate 名超长", func() *FeedbackBuilder {
			return NewFeedbackBuilder(fbTask, 1, fbCSFailed, "").ForGate(long)
		}},
		{"验证运行引用超长", func() *FeedbackBuilder {
			return NewFeedbackBuilder(fbTask, 1, fbCSFailed, long)
		}},
		{"可见证据身份超长", func() *FeedbackBuilder {
			f := fact
			f.BindingID = long
			return NewFeedbackBuilder(fbTask, 1, fbCSFailed, "").
				ForAcceptance("AC-1", "statement").
				AddVisibleEvidence(f)
		}},
	}
	for _, tc := range cases {
		_, err := tc.build().WithFailure(FailureRepairable).
			WithDiagnosticSummary("summary").Build()
		if !errors.Is(err, ErrFeedbackBoundExceeded) {
			t.Fatalf("%s:期望 ErrFeedbackBoundExceeded,得到 %v", tc.name, err)
		}
	}
}

// ---- AC4:不安全/不合法输入 fail-safe ----

func TestFeedbackUnsafeInputsFailSafe(t *testing.T) {
	// 只有 repairable 可构造修复反馈:其余全部类别(含非法值/缺席)一律拒绝
	// —— 升级与 infra 材料绝不从修复通道流出(PRD §17.4)。
	for _, cls := range allFailureClasses {
		if cls == FailureRepairable {
			continue
		}
		fb, err := NewFeedbackBuilder(fbTask, 1, fbCSFailed, "").
			ForGate("build").
			WithFailure(cls).
			WithDiagnosticSummary("summary").
			Build()
		if !errors.Is(err, ErrFailureNotRepairable) {
			t.Fatalf("失败类 %q 应报 ErrFailureNotRepairable,得到 %v", string(cls), err)
		}
		if fb.DiagnosticSummary != "" || len(fb.Evidence) != 0 || len(fb.Holdout) != 0 {
			t.Fatalf("失败构造不得产出部分反馈: %+v", fb)
		}
	}
	for _, cls := range []FailureClass{"", "bogus", "REPAIRABLE"} {
		_, err := NewFeedbackBuilder(fbTask, 1, fbCSFailed, "").
			ForGate("build").
			WithFailure(cls).
			WithDiagnosticSummary("summary").
			Build()
		if !errors.Is(err, ErrFailureNotRepairable) {
			t.Fatalf("非法失败类 %q 应报 ErrFailureNotRepairable,得到 %v", string(cls), err)
		}
	}

	base := func() *FeedbackBuilder {
		return NewFeedbackBuilder(fbTask, 1, fbCSFailed, "verify-run-7").
			ForAcceptance("AC-1", "statement").
			WithFailure(FailureRepairable).
			WithDiagnosticSummary("summary")
	}

	// 构造身份缺失。
	for _, tc := range []struct {
		name string
		call func() *FeedbackBuilder
	}{
		{"task id 为空", func() *FeedbackBuilder { return NewFeedbackBuilder("  ", 1, fbCSFailed, "") }},
		{"尝试序号非法", func() *FeedbackBuilder { return NewFeedbackBuilder(fbTask, 0, fbCSFailed, "") }},
		{"ChangeSet 为空", func() *FeedbackBuilder { return NewFeedbackBuilder(fbTask, 1, "", "") }},
	} {
		if _, err := tc.call().Build(); !errors.Is(err, ErrInvalidFeedback) {
			t.Fatalf("%s:期望 ErrInvalidFeedback,得到 %v", tc.name, err)
		}
	}

	// 主体缺一/双重/缺文本。
	for _, tc := range []struct {
		name string
		call func() *FeedbackBuilder
	}{
		{"缺少主体", func() *FeedbackBuilder {
			return NewFeedbackBuilder(fbTask, 1, fbCSFailed, "").
				WithFailure(FailureRepairable).
				WithDiagnosticSummary("summary")
		}},
		{"双重主体", func() *FeedbackBuilder {
			return NewFeedbackBuilder(fbTask, 1, fbCSFailed, "").
				ForAcceptance("AC-1", "statement").
				ForGate("build")
		}},
		{"acceptance 缺语义文本", func() *FeedbackBuilder {
			return NewFeedbackBuilder(fbTask, 1, fbCSFailed, "").ForAcceptance("AC-1", "  ")
		}},
		{"gate 名为空", func() *FeedbackBuilder {
			return NewFeedbackBuilder(fbTask, 1, fbCSFailed, "").ForGate("  ")
		}},
		{"缺少失败类", func() *FeedbackBuilder {
			return NewFeedbackBuilder(fbTask, 1, fbCSFailed, "").
				ForGate("build").
				WithDiagnosticSummary("summary")
		}},
		{"缺少诊断摘要", func() *FeedbackBuilder {
			return NewFeedbackBuilder(fbTask, 1, fbCSFailed, "").
				ForGate("build").
				WithFailure(FailureRepairable)
		}},
		{"诊断摘要为空白", func() *FeedbackBuilder {
			return NewFeedbackBuilder(fbTask, 1, fbCSFailed, "").
				ForGate("build").
				WithFailure(FailureRepairable).
				WithDiagnosticSummary("   ")
		}},
		{"失败类重复设置", func() *FeedbackBuilder {
			return NewFeedbackBuilder(fbTask, 1, fbCSFailed, "").
				ForGate("build").
				WithFailure(FailureRepairable).
				WithFailure(FailureRepairable)
		}},
	} {
		fb, err := tc.call().Build()
		if !errors.Is(err, ErrInvalidFeedback) && !errors.Is(err, ErrFailureNotRepairable) {
			t.Fatalf("%s:期望类型化错误,得到 %v", tc.name, err)
		}
		if fb.DiagnosticSummary != "" || fb.Subject.AcceptanceID != "" || fb.Subject.Gate != "" || len(fb.Evidence) != 0 || len(fb.Holdout) != 0 {
			t.Fatalf("%s:失败构造不得产出反馈: %+v", tc.name, fb)
		}
	}

	// 不安全证据材料:未知类型 / 非法类别 / 任务身份矛盾 → ErrUnsafeFeedback
	// 或 ErrInvalidFeedback,绝不静默改写或丢弃。
	_, err := base().AddVisibleEvidence(VisibleEvidence{
		BindingID:    "b",
		EvidenceType: verification.EvidenceType("mystery"),
		Status:       verification.EvidenceStatusFail,
	}).Build()
	if !errors.Is(err, ErrUnsafeFeedback) {
		t.Fatalf("未知证据类型应报 ErrUnsafeFeedback,得到 %v", err)
	}
	_, err = base().AddVisibleEvidence(VisibleEvidence{
		BindingID:    "b",
		EvidenceType: verification.EvidenceTest,
		Status:       verification.EvidenceStatus("error"),
	}).Build()
	if !errors.Is(err, ErrUnsafeFeedback) {
		t.Fatalf("可见证据非法类别应报 ErrUnsafeFeedback,得到 %v", err)
	}
	_, err = base().AddVisibleEvidence(VisibleEvidence{
		EvidenceType: verification.EvidenceTest,
		Status:       verification.EvidenceStatusFail,
	}).Build()
	if !errors.Is(err, ErrInvalidFeedback) {
		t.Fatalf("可见证据缺 binding id 应报 ErrInvalidFeedback,得到 %v", err)
	}
	_, err = base().AddHoldoutOutcome(verification.RedactedHoldoutOutcome{
		TaskID:       fbTask,
		AcceptanceID: "AC-1",
		EvidenceType: verification.EvidenceTest,
	}, verification.EvidenceStatus("inconclusive")).Build()
	if !errors.Is(err, ErrUnsafeFeedback) {
		t.Fatalf("held-out 非法类别应报 ErrUnsafeFeedback,得到 %v", err)
	}
	_, err = base().AddHoldoutOutcome(verification.RedactedHoldoutOutcome{
		TaskID:       "T-OTHER",
		AcceptanceID: "AC-1",
		EvidenceType: verification.EvidenceTest,
	}, verification.EvidenceStatusFail).Build()
	if !errors.Is(err, ErrInvalidFeedback) {
		t.Fatalf("held-out 任务身份矛盾应报 ErrInvalidFeedback,得到 %v", err)
	}
	_, err = base().AddHoldoutOutcome(verification.RedactedHoldoutOutcome{
		TaskID:       fbTask,
		AcceptanceID: " ",
		EvidenceType: verification.EvidenceTest,
	}, verification.EvidenceStatusFail).Build()
	if !errors.Is(err, ErrInvalidFeedback) {
		t.Fatalf("held-out 投影缺 acceptance id 应报 ErrInvalidFeedback,得到 %v", err)
	}

	// 粘滞错误:首个错误保留,后续调用 no-op,Build 原样返回(不吞错)。
	sticky := NewFeedbackBuilder("", 1, fbCSFailed, "")
	_, first := sticky.Build()
	sticky.ForGate("build").WithFailure(FailureRepairable).WithDiagnosticSummary("summary")
	fb2, second := sticky.Build()
	if first == nil || second == nil {
		t.Fatalf("粘滞错误不得为 nil: first=%v second=%v", first, second)
	}
	if first.Error() != second.Error() || !errors.Is(second, ErrInvalidFeedback) {
		t.Fatalf("粘滞错误应保持不变: first=%v second=%v", first, second)
	}
	if fb2.DiagnosticSummary != "" || len(fb2.Evidence) != 0 {
		t.Fatalf("粘滞失败后 Build 不得产出反馈: %+v", fb2)
	}
}

// ---- 确定性:相同输入恒得相同反馈;Build 幂等且输出隔离 ----

func TestFeedbackBuildIsDeterministic(t *testing.T) {
	rec := fbHoldoutRecord()
	build := func() RepairFeedback {
		fb, err := NewFeedbackBuilder(fbTask, 2, fbCSFailed, "verify-run-7").
			ForAcceptance("AC-1", "the widget counts occurrences").
			WithFailure(FailureRepairable).
			WithAffectedBehavior("counting drops occurrences above the batch limit").
			WithDiagnosticSummary("a visible assertion failed while counting").
			AddHoldoutOutcome(verification.RedactHoldoutRecord(rec), verification.EvidenceStatusFail).
			AddVisibleEvidence(VisibleEvidence{
				AcceptanceID: "AC-1",
				BindingID:    "b-visible",
				EvidenceType: verification.EvidenceTest,
				Path:         "internal/widget/widget_test.go",
				Status:       verification.EvidenceStatusFail,
			}).
			Build()
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		return fb
	}
	a, c := build(), build()
	if !reflect.DeepEqual(a, c) {
		t.Fatalf("相同输入产出不同反馈:\n%+v\n%+v", a, c)
	}
	if ja, err := json.Marshal(a); err != nil {
		t.Fatalf("marshal: %v", err)
	} else if jc, err := json.Marshal(c); err != nil {
		t.Fatalf("marshal: %v", err)
	} else if string(ja) != string(jc) {
		t.Fatalf("相同输入的 JSON 不一致:\n%s\n%s", ja, jc)
	}

	// 输出隔离:改写已产出反馈的切片不影响构造器的后续产出。
	a.Evidence = append(a.Evidence, VisibleEvidence{BindingID: "b-mutated"})
	d := build()
	if len(d.Evidence) != 1 || d.Evidence[0].BindingID != "b-visible" {
		t.Fatalf("Build 输出应与构造器状态隔离: %+v", d.Evidence)
	}
}
