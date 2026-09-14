package intakefreeze

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"tally/internal/intake"
	"tally/internal/verification"
)

// planBase 返回与 confirmedBase() 两条验收条款一一对应的证据计划基线:每条
// 条款一条 visible test 声明(无 held-out,无 runtime 能力要求)。
func planBase() VerificationPlan {
	return VerificationPlan{
		Evidence: []EvidenceDeclaration{
			{AcceptanceID: "AC-01", Kind: verification.EvidenceTest,
				Path: "internal/session/session_test.go", Selector: "TestLogoutInvalidatesSession"},
			{AcceptanceID: "AC-02", Kind: verification.EvidenceTest,
				Path: "internal/session/store_test.go", Selector: "TestLogoutRemovesRecord"},
		},
	}
}

// 正路径(normal):确认草案 + 完整证据计划 → 三份冻结制品;验证契约的
// 作者化逐字段正确、与任务合同条款集 1:1、确定性(输入换序不影响哈希),
// 且与直接经 verification.FreezeContract 冻结同一内容的结果完全一致。
func TestFreezeHappyPathAuthorsVerification(t *testing.T) {
	in := confirmedBase()
	out, err := Freeze(in, planBase())
	if err != nil {
		t.Fatalf("Freeze: %v", err)
	}

	// 复用 P0-06 映射:Intent/Task 与 Map 直接产出完全一致(单一映射路径)。
	mapped, err := Map(in)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	if out.Intent.Revision() != mapped.Intent.Revision() || out.Task.Hash != mapped.Task.Hash {
		t.Fatalf("Freeze 的 Intent/Task 与 Map 产物不一致(revision/hash 漂移)")
	}
	if got := out.Verification.TaskID(); got != mapped.Task.ID {
		t.Errorf("Verification.TaskID = %q,与 Task.ID %q 不同一", got, mapped.Task.ID)
	}
	if out.Verification.Revision() != 1 {
		t.Errorf("Verification.Revision = %d,首次作者化应为 1", out.Verification.Revision())
	}

	// 作者化语义:条款集与任务合同 1:1,绑定 id 机械派生、provenance 固定
	// pre_commit、可见性默认 visible、证据定位逐字搬运。
	contract := out.Verification.Contract()
	if len(contract.Acceptances) != len(mapped.Task.Acceptance) {
		t.Fatalf("验证契约条款数 = %d,任务合同条款数 = %d", len(contract.Acceptances), len(mapped.Task.Acceptance))
	}
	seenClauses := map[string]bool{}
	for _, acc := range contract.Acceptances {
		seenClauses[acc.ID] = true
		if len(acc.EvidenceBindings) == 0 {
			t.Errorf("acceptance %s 没有任何 evidence binding(AC1:每条冻结验收必须有有效 planned binding)", acc.ID)
		}
		for _, b := range acc.EvidenceBindings {
			if !strings.HasPrefix(b.ID, acc.ID+"-EV") {
				t.Errorf("binding id %q 未由条款 %s 机械派生", b.ID, acc.ID)
			}
			if b.AuthoringMode != verification.AuthoringPreCommit {
				t.Errorf("binding %s authoring_mode = %q,Confirm→Freeze 边界只可作者化 pre_commit", b.ID, b.AuthoringMode)
			}
			if b.VisibilityPolicy != verification.VisibilityVisible {
				t.Errorf("binding %s visibility = %q,未声明 held-out 的计划应默认 visible", b.ID, b.VisibilityPolicy)
			}
		}
	}
	for _, a := range mapped.Task.Acceptance {
		if !seenClauses[a.ID] {
			t.Errorf("任务合同条款 %s 在验证契约中没有对应 acceptance(遗漏防护被破坏)", a.ID)
		}
	}
	if contract.RuntimeRequirements != (verification.RuntimeRequirements{}) {
		t.Errorf("纯 visible 契约不应携带 runtime 能力要求:%+v", contract.RuntimeRequirements)
	}

	// 与既有权威交叉金测试:同一内容手工构造后经 verification.FreezeContract
	// 冻结,哈希必须与准入入口产出一致(作者化不引入语义改写)。
	direct, err := verification.FreezeContract(&contract, verification.FreezePolicy{})
	if err != nil {
		t.Fatalf("verification.FreezeContract(作者化产物): %v", err)
	}
	if direct.Hash() != out.Verification.Hash() {
		t.Errorf("Verification.Hash = %s,直接冻结同一内容 = %s", out.Verification.Hash(), direct.Hash())
	}

	// 确定性:同一输入两次冻结,哈希一致;声明/条款换序是排版,哈希不变。
	again, err := Freeze(in, planBase())
	if err != nil {
		t.Fatalf("Freeze 第二次: %v", err)
	}
	if again.Verification.Hash() != out.Verification.Hash() {
		t.Errorf("同一输入两次冻结哈希不同:%s != %s", again.Verification.Hash(), out.Verification.Hash())
	}
	shuffled := confirmedBase()
	shuffled.Draft.Acceptance = []intake.AcceptanceClause{
		shuffled.Draft.Acceptance[1], shuffled.Draft.Acceptance[0],
	}
	plan := planBase()
	plan.Evidence = []EvidenceDeclaration{plan.Evidence[1], plan.Evidence[0]}
	permuted, err := Freeze(shuffled, plan)
	if err != nil {
		t.Fatalf("Freeze(换序输入): %v", err)
	}
	if permuted.Verification.Hash() != out.Verification.Hash() {
		t.Errorf("输入换序后 Verification.Hash 改变:%s != %s(顺序是排版不是内容)", permuted.Verification.Hash(), out.Verification.Hash())
	}
	if permuted.Task.Hash != out.Task.Hash {
		t.Errorf("输入换序后 Task.Hash 改变:%s != %s", permuted.Task.Hash, out.Task.Hash)
	}
}

// 遗漏防护(missing-evidence):任何验收条款缺 planned evidence 声明 →
// 类型化阻断并列出缺失条款;不产生任何冻结产物(AC2)。
func TestFreezeRejectsMissingEvidence(t *testing.T) {
	t.Run("缺一条", func(t *testing.T) {
		plan := planBase()
		plan.Evidence = plan.Evidence[1:] // AC-01 无证据
		out, err := Freeze(confirmedBase(), plan)
		if !errors.Is(err, ErrMissingEvidence) {
			t.Fatalf("err = %v, want ErrMissingEvidence", err)
		}
		if !strings.Contains(err.Error(), "AC-01") {
			t.Errorf("错误信息应点名缺失条款,got %q", err.Error())
		}
		if strings.Contains(err.Error(), "AC-02") {
			t.Errorf("不应把已有证据的条款列为缺失,got %q", err.Error())
		}
		if out.Intent.Revision() != "" || out.Task.Hash != "" || out.Verification.Hash() != "" {
			t.Fatalf("失败不得产出部分冻结产物:%+v", out)
		}
	})

	t.Run("空计划一次列全(条款 id 升序)", func(t *testing.T) {
		_, err := Freeze(confirmedBase(), VerificationPlan{})
		if !errors.Is(err, ErrMissingEvidence) {
			t.Fatalf("err = %v, want ErrMissingEvidence", err)
		}
		if !strings.Contains(err.Error(), "AC-01, AC-02") {
			t.Errorf("错误信息应按 id 升序列出全部缺失条款,got %q", err.Error())
		}
	})
}

// 发明防护:声明指向不存在的条款(含空 id 与纯空白 id)→ 类型化拒绝,
// 绝不改挂、绝不静默丢弃 —— Confirm→Freeze 不能发明证据绑定。
func TestFreezeRejectsUnknownEvidenceTarget(t *testing.T) {
	for _, id := range []string{"AC-09", "", "   ", "ac-01", "AC-1"} {
		plan := planBase()
		plan.Evidence = append(plan.Evidence, EvidenceDeclaration{
			AcceptanceID: id, Kind: verification.EvidenceTest, Path: "x_test.go",
		})
		out, err := Freeze(confirmedBase(), plan)
		if !errors.Is(err, ErrUnknownEvidenceTarget) {
			t.Errorf("孤儿声明 %q: err = %v, want ErrUnknownEvidenceTarget", id, err)
		}
		if out.Verification.Hash() != "" {
			t.Errorf("孤儿声明 %q: 失败不得产出冻结产物", id)
		}
	}
	// 首尾排版空白折叠后命中条款,合法(与 profile 同一排版纪律)。
	plan := planBase()
	plan.Evidence[0].AcceptanceID = " AC-01 "
	if _, err := Freeze(confirmedBase(), plan); err != nil {
		t.Errorf("首尾空白的条款路由键应按排版折叠: %v", err)
	}
}

// held-out 语义更严:held_out 只来自显式声明,且必须伴随完整的实现侧 runtime
// 隔离能力要求;缺席/不完整都类型化阻断,完整声明才可冻结并原样进入绑定。
func TestFreezeHeldOutExplicitDeclarationWithIsolationGate(t *testing.T) {
	heldPlan := func() VerificationPlan {
		plan := planBase()
		plan.Evidence = append(plan.Evidence, EvidenceDeclaration{
			AcceptanceID: "AC-02", Kind: verification.EvidenceTest,
			Store: "holdout", Ref: "objects/seeded-defect", HeldOut: true,
		})
		return plan
	}

	t.Run("缺 runtime 能力要求", func(t *testing.T) {
		_, err := Freeze(confirmedBase(), heldPlan())
		if !errors.Is(err, ErrHeldOutWithoutIsolation) {
			t.Fatalf("err = %v, want ErrHeldOutWithoutIsolation", err)
		}
	})
	t.Run("能力要求不完整", func(t *testing.T) {
		for _, rr := range []verification.RuntimeRequirements{
			{SessionIsolation: true, HeldOutUnreachable: false, ConformanceClass: "verification-isolation-v1"},
			{SessionIsolation: false, HeldOutUnreachable: true, ConformanceClass: "verification-isolation-v1"},
			{SessionIsolation: true, HeldOutUnreachable: true},
		} {
			plan := heldPlan()
			plan.RuntimeRequirements = &rr
			if _, err := Freeze(confirmedBase(), plan); !errors.Is(err, ErrHeldOutWithoutIsolation) {
				t.Errorf("rr %+v: err = %v, want ErrHeldOutWithoutIsolation", rr, err)
			}
		}
	})
	t.Run("完整声明可冻结且绑定原样承载", func(t *testing.T) {
		plan := heldPlan()
		plan.RuntimeRequirements = &verification.RuntimeRequirements{
			SessionIsolation: true, HeldOutUnreachable: true, ConformanceClass: "verification-isolation-v1",
		}
		out, err := Freeze(confirmedBase(), plan)
		if err != nil {
			t.Fatalf("Freeze: %v", err)
		}
		contract := out.Verification.Contract()
		var held *verification.Binding
		for i := range contract.Acceptances {
			for j := range contract.Acceptances[i].EvidenceBindings {
				if b := &contract.Acceptances[i].EvidenceBindings[j]; b.VisibilityPolicy == verification.VisibilityHeldOut {
					held = b
				} else if b.Evidence.Store != "" {
					t.Errorf("visible binding %s 携带 store 引用 %q", b.ID, b.Evidence.Store)
				}
			}
		}
		if held == nil {
			t.Fatalf("显式 held-out 声明未产生 held_out 绑定")
		}
		if held.Evidence.Path != "" || held.Evidence.Store != "holdout" || held.Evidence.Ref == "" {
			t.Errorf("held_out 绑定定位形状非法:%+v(§8.4.1 只允许 store/ref)", held.Evidence)
		}
		if out.Verification.Hash() == "" {
			t.Errorf("expected frozen hash")
		}
	})
	t.Run("未显式声明的绑定绝不自动成为 held-out", func(t *testing.T) {
		out, err := Freeze(confirmedBase(), planBase())
		if err != nil {
			t.Fatalf("Freeze: %v", err)
		}
		for _, acc := range out.Verification.Contract().Acceptances {
			for _, b := range acc.EvidenceBindings {
				if b.VisibilityPolicy == verification.VisibilityHeldOut {
					t.Errorf("binding %s 在无显式声明时成了 held_out(自动标注被禁止)", b.ID)
				}
			}
		}
	})
}

// 无效绑定(invalid binding):声明形状/词表违规由 verification 域校验兜底,
// 以 ErrInvalidVerification 类型化浮出;等值重复声明被拒绝而不是去重。
func TestFreezeRejectsInvalidBindings(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*VerificationPlan)
	}{
		{"未知证据类型", func(p *VerificationPlan) { p.Evidence[0].Kind = "tarot" }},
		{"缺失证据类型", func(p *VerificationPlan) { p.Evidence[0].Kind = "" }},
		{"visible 声明引用 holdout store", func(p *VerificationPlan) {
			p.Evidence[0].Store = "holdout"
			p.Evidence[0].Ref = "objects/x"
		}},
		{"等值重复声明", func(p *VerificationPlan) { p.Evidence[1] = p.Evidence[0] }},
		{"仅路由键不同的重复声明", func(p *VerificationPlan) {
			dup := p.Evidence[0]
			dup.AcceptanceID = " AC-01 " // 排版变体,内容等值
			p.Evidence = append(p.Evidence, dup)
		}},
	}
	for _, tc := range cases {
		plan := planBase()
		tc.mut(&plan)
		out, err := Freeze(confirmedBase(), plan)
		if !errors.Is(err, ErrInvalidVerification) {
			t.Errorf("%s: err = %v, want ErrInvalidVerification", tc.name, err)
		}
		if out.Verification.Hash() != "" {
			t.Errorf("%s: 失败不得产出冻结产物", tc.name)
		}
	}

	// held-out 形状违规(Path 暴露 repo 路径)由既有 §8.4.1 校验拒绝;
	// 隔离门先于它触发,补齐能力要求后应落到 ErrInvalidVerification。
	plan := planBase()
	plan.Evidence = append(plan.Evidence, EvidenceDeclaration{
		AcceptanceID: "AC-01", Kind: verification.EvidenceTest,
		Path: "internal/session/hidden_test.go", Store: "holdout", Ref: "objects/x", HeldOut: true,
	})
	plan.RuntimeRequirements = &verification.RuntimeRequirements{
		SessionIsolation: true, HeldOutUnreachable: true, ConformanceClass: "verification-isolation-v1",
	}
	if _, err := Freeze(confirmedBase(), plan); !errors.Is(err, ErrInvalidVerification) {
		t.Errorf("held-out 携带 repo path: err = %v, want ErrInvalidVerification", err)
	}
}

// light profile 且零验收条款:验证契约会是空集,verification 域拒绝
// (「acceptances 不能为空」)—— 没有验收就没有验证路径,不能执行就绪。
func TestFreezeRejectsAcceptanceFreezeWithoutClauses(t *testing.T) {
	in := confirmedBase()
	in.Profile = "light"
	in.Draft.Acceptance = nil
	_, err := Freeze(in, VerificationPlan{})
	if !errors.Is(err, ErrInvalidVerification) {
		t.Fatalf("err = %v, want ErrInvalidVerification", err)
	}
}

// Map 门原样生效且先行:治理输入/草案语义非法在验证作者化之前就被拒绝。
func TestFreezeReusesMapGatesFirst(t *testing.T) {
	in := confirmedBase()
	in.Profile = "hyper"
	if _, err := Freeze(in, planBase()); !errors.Is(err, ErrInvalidProfile) {
		t.Errorf("err = %v, want ErrInvalidProfile", err)
	}
	in = confirmedBase()
	in.Draft.NonScope = []string{"不改动导出 API"}
	if _, err := Freeze(in, planBase()); !errors.Is(err, ErrUnsupportedDraftSemantics) {
		t.Errorf("err = %v, want ErrUnsupportedDraftSemantics", err)
	}
}

// 输入隔离:Freeze 不修改确认草案与计划,也不保留其切片引用 —— 冻结之后
// 改动输入,冻结哈希纹丝不动。
func TestFreezeDoesNotMutateInputs(t *testing.T) {
	in := confirmedBase()
	plan := planBase()
	inSnapshot, planSnapshot := confirmedBase(), planBase()
	out, err := Freeze(in, plan)
	if err != nil {
		t.Fatalf("Freeze: %v", err)
	}
	wantHash := out.Verification.Hash()

	if !reflect.DeepEqual(in.Draft, inSnapshot.Draft) {
		t.Fatalf("Freeze 修改了输入草案本身")
	}
	if !reflect.DeepEqual(plan, planSnapshot) {
		t.Fatalf("Freeze 修改了输入计划本身")
	}

	in.Draft.Acceptance[0].Statement = "hijacked"
	plan.Evidence[0].Path = "hijacked_test.go"
	plan.Evidence[0].Selector = "TestHijacked"

	if out.Verification.Hash() != wantHash {
		t.Fatalf("冻结产物在输入被篡改后漂移:%s != %s", out.Verification.Hash(), wantHash)
	}
}
