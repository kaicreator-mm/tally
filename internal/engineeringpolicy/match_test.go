package engineeringpolicy

import (
	"errors"
	"reflect"
	"testing"

	"tally/internal/risk"
)

// ---- 夹具 ---------------------------------------------------------------

// matchGatePolicy 构造一份合法的 gate 策略:trigger 与 gate 键均可指定。
func matchGatePolicy(id string, enforcement Enforcement, trigger Trigger) Policy {
	p := Policy{
		ID:          id,
		Enforcement: enforcement,
		Requires:    []string{"k_" + id},
		Trigger:     trigger,
	}
	if enforcement == EnforcementGate {
		p.GateCondition = GateCondition{AllOf: []string{"k_" + id}}
	}
	return p
}

// mustDerive 用 risk.Derive 构造派生风险集,失败即测试失败。
func mustDerive(t *testing.T, sigs ...risk.Signal) risk.Result {
	t.Helper()
	res, err := risk.Derive(sigs)
	if err != nil {
		t.Fatalf("risk.Derive: %v", err)
	}
	return res
}

func sig(id risk.SignalID, refs ...string) risk.Signal {
	if len(refs) == 0 {
		refs = []string{"intent:t1"}
	}
	return risk.Signal{ID: id, Basis: refs}
}

// ---- 命中与 basis(AC1)--------------------------------------------------

func TestMatchByEachTriggerAxis(t *testing.T) {
	riskSet := mustDerive(t, sig(risk.SignalSchemaMigration))
	cases := []struct {
		name         string
		policy       Policy
		input        MatchInput
		wantRiskAxis []risk.SignalID
		wantPaths    []string
		wantRefs     []string
	}{
		{
			name:         "风险信号轴",
			policy:       matchGatePolicy("P-RISK", EnforcementGate, Trigger{RiskSignals: []string{string(risk.SignalSchemaMigration)}}),
			input:        MatchInput{Risk: riskSet},
			wantRiskAxis: []risk.SignalID{risk.SignalSchemaMigration},
		},
		{
			name:      "路径轴",
			policy:    matchGatePolicy("P-PATH", EnforcementGate, Trigger{Paths: []string{"migrations/001_init.sql"}}),
			input:     MatchInput{ChangedPaths: []string{"cmd/x/main.go", "migrations/001_init.sql"}},
			wantPaths: []string{"migrations/001_init.sql"},
		},
		{
			name:     "需求引用轴",
			policy:   matchGatePolicy("P-REQ", EnforcementGate, Trigger{RequirementRefs: []string{".tally/requirements/REQ-018.md"}}),
			input:    MatchInput{RequirementRefs: []string{".tally/requirements/REQ-018.md"}},
			wantRefs: []string{".tally/requirements/REQ-018.md"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cov, err := Match([]Policy{tc.policy}, tc.input)
			if err != nil {
				t.Fatalf("Match: %v", err)
			}
			if len(cov.MatchedPolicies) != 1 {
				t.Fatalf("期望恰好 1 条命中,得到 %d", len(cov.MatchedPolicies))
			}
			basis := cov.MatchedPolicies[0].Basis
			if !basis.Matched() {
				t.Fatal("命中必须携带非空 basis(AC1)")
			}
			if !reflect.DeepEqual(basis.RiskSignals, tc.wantRiskAxis) {
				t.Fatalf("basis.RiskSignals = %v,期望 %v", basis.RiskSignals, tc.wantRiskAxis)
			}
			if !reflect.DeepEqual(basis.Paths, tc.wantPaths) {
				t.Fatalf("basis.Paths = %v,期望 %v", basis.Paths, tc.wantPaths)
			}
			if !reflect.DeepEqual(basis.RequirementRefs, tc.wantRefs) {
				t.Fatalf("basis.RequirementRefs = %v,期望 %v", basis.RequirementRefs, tc.wantRefs)
			}
		})
	}
}

func TestMatchAnyAxisFiresAndBasisRecordsAllHitAxes(t *testing.T) {
	// 三轴之间是「任一命中即命中」;全部命中时 basis 逐轴记录全部命中值。
	p := matchGatePolicy("P-ALL", EnforcementGate, Trigger{
		RiskSignals:     []string{"schema_migration"},
		Paths:           []string{"db/schema.sql", "unrelated/"},
		RequirementRefs: []string{".tally/requirements/REQ-018.md"},
	})
	cov, err := Match([]Policy{p}, MatchInput{
		Risk:            mustDerive(t, sig(risk.SignalSchemaMigration)),
		ChangedPaths:    []string{"db/schema.sql"},
		RequirementRefs: []string{".tally/requirements/REQ-018.md", "other.md"},
	})
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if len(cov.MatchedPolicies) != 1 {
		t.Fatalf("期望 1 条命中,得到 %d", len(cov.MatchedPolicies))
	}
	basis := cov.MatchedPolicies[0].Basis
	if !reflect.DeepEqual(basis.RiskSignals, []risk.SignalID{risk.SignalSchemaMigration}) ||
		!reflect.DeepEqual(basis.Paths, []string{"db/schema.sql"}) ||
		!reflect.DeepEqual(basis.RequirementRefs, []string{".tally/requirements/REQ-018.md"}) {
		t.Fatalf("basis 未逐轴记录命中值: %+v", basis)
	}
}

func TestMatchBasisOnlyContainsValuesPresentInInput(t *testing.T) {
	// AC1 的反面:basis 不允许出现输入中不存在的值(机器可读 = 可对账)。
	policies := []Policy{
		matchGatePolicy("P-A", EnforcementGate, Trigger{
			RiskSignals:     []string{"auth_symbol_change", "schema_migration"},
			Paths:           []string{"internal/auth/", "db/mig.sql"},
			RequirementRefs: []string{".tally/requirements/REQ-001.md"},
		}),
		matchGatePolicy("P-B", EnforcementGuidance, Trigger{RiskSignals: []string{"protected_surface_change"}}),
	}
	cov, err := Match(policies, MatchInput{
		Risk: mustDerive(t,
			sig(risk.SignalSchemaMigration),
			sig(risk.SignalProtectedSurfaceChange),
		),
		ChangedPaths:    []string{"db/mig.sql"},
		RequirementRefs: []string{".tally/requirements/REQ-001.md"},
	})
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	inRisk := map[string]bool{"schema_migration": true, "protected_surface_change": true}
	inPaths := map[string]bool{"db/mig.sql": true}
	inRefs := map[string]bool{".tally/requirements/REQ-001.md": true}
	if len(cov.MatchedPolicies) != 2 {
		t.Fatalf("期望 2 条命中,得到 %d", len(cov.MatchedPolicies))
	}
	for _, m := range cov.MatchedPolicies {
		if !m.Basis.Matched() {
			t.Fatalf("策略 %s 出现无依据命中(AC1 违例)", m.Policy.ID)
		}
		for _, id := range m.Basis.RiskSignals {
			if !inRisk[string(id)] {
				t.Fatalf("basis 风险信号 %q 不在输入中", string(id))
			}
		}
		for _, p := range m.Basis.Paths {
			if !inPaths[p] {
				t.Fatalf("basis 路径 %q 不在输入中", p)
			}
		}
		for _, r := range m.Basis.RequirementRefs {
			if !inRefs[r] {
				t.Fatalf("basis 需求引用 %q 不在输入中", r)
			}
		}
	}
}

// ---- 触发边界 ------------------------------------------------------------

func TestMatchZeroTriggerNeverMatches(t *testing.T) {
	// 空 Trigger 声明了零个条件:PRD §8.1 没有 match-all,不得猜。
	p := matchGatePolicy("P-EMPTY", EnforcementGate, Trigger{})
	cov, err := Match([]Policy{p}, MatchInput{
		Risk:            mustDerive(t, sig(risk.SignalSchemaMigration)),
		ChangedPaths:    []string{"anything/goes.sql"},
		RequirementRefs: []string{".tally/requirements/REQ-018.md"},
	})
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if len(cov.MatchedPolicies) != 0 || cov.Confidence != ConfidenceNone {
		t.Fatalf("零 Trigger 不得命中,得到 %+v", cov)
	}
	if !cov.HasUnmatchedRisk() || !reflect.DeepEqual(cov.UnmatchedRiskSignals, []risk.SignalID{risk.SignalSchemaMigration}) {
		t.Fatalf("唯一信号应保持未覆盖可见: %+v", cov)
	}
}

func TestMatchExactValueOnlyNoGlobNoRegex(t *testing.T) {
	// 匹配形态刻意最小:精确值相等,不做 glob/正则/前缀扩展。
	p := matchGatePolicy("P-EXACT", EnforcementGate, Trigger{Paths: []string{"migrations/"}})
	cov, err := Match([]Policy{p}, MatchInput{ChangedPaths: []string{"migrations/001_init.sql"}})
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if len(cov.MatchedPolicies) != 0 {
		t.Fatalf("前缀不得命中: %+v", cov.MatchedPolicies)
	}
}

func TestMatchGuidancePolicyMatchesWithoutBlockingPower(t *testing.T) {
	// 匹配对 enforcement 中立:guidance 可以命中并覆盖其声明的信号,
	// 但它得到的是事实记录,不是阻断权(阻断权仍归 gate.go 的 ErrNotGate 边界)。
	p := matchGatePolicy("P-GUIDE", EnforcementGuidance, Trigger{RiskSignals: []string{"auth_symbol_change"}})
	cov, err := Match([]Policy{p}, MatchInput{Risk: mustDerive(t, sig(risk.SignalAuthSymbolChange))})
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if len(cov.MatchedPolicies) != 1 || cov.Confidence != ConfidenceFull {
		t.Fatalf("guidance 应命中且覆盖其信号: %+v", cov)
	}
}

// ---- 未覆盖风险(INV-V14-09,AC2)----------------------------------------

func TestMatchUnmatchedRiskSignalsExplicit(t *testing.T) {
	// 策略经路径轴命中但只声明了 auth 信号:schema_migration 必须作为
	// 未覆盖信号显式露出,confidence 不得是 full。
	p := matchGatePolicy("P-PATH", EnforcementGate, Trigger{
		Paths:       []string{"db/mig.sql"},
		RiskSignals: []string{"auth_symbol_change"},
	})
	cov, err := Match([]Policy{p}, MatchInput{
		Risk: mustDerive(t,
			sig(risk.SignalSchemaMigration, "fact:e-1"),
			sig(risk.SignalAuthSymbolChange, "fact:e-2"),
		),
		ChangedPaths: []string{"db/mig.sql"},
	})
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if !reflect.DeepEqual(cov.DetectedRiskSignals, []risk.SignalID{risk.SignalAuthSymbolChange, risk.SignalSchemaMigration}) {
		t.Fatalf("DetectedRiskSignals 应升序全集: %v", cov.DetectedRiskSignals)
	}
	if !reflect.DeepEqual(cov.UnmatchedRiskSignals, []risk.SignalID{risk.SignalSchemaMigration}) {
		t.Fatalf("schema_migration 必须未覆盖可见: %v", cov.UnmatchedRiskSignals)
	}
	if !cov.HasUnmatchedRisk() {
		t.Fatal("HasUnmatchedRisk 应为 true")
	}
	if cov.Confidence != ConfidencePartial {
		t.Fatalf("存在未覆盖信号时 confidence 必须是 partial,得到 %q", cov.Confidence)
	}
}

func TestMatchUnknownSignalsRemainVisibleAndFailSafe(t *testing.T) {
	// INV-V14-09:未知信号不被丢弃、不被静默当作已覆盖。
	p := matchGatePolicy("P-KNOWN", EnforcementGate, Trigger{RiskSignals: []string{"schema_migration"}})
	cov, err := Match([]Policy{p}, MatchInput{
		Risk: mustDerive(t,
			sig(risk.SignalSchemaMigration),
			sig("custom_team_signal", "fact:e-9"),
		),
	})
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if !reflect.DeepEqual(cov.UnknownRiskSignals, []risk.SignalID{"custom_team_signal"}) {
		t.Fatalf("unknown 信号必须原样保真: %v", cov.UnknownRiskSignals)
	}
	if !reflect.DeepEqual(cov.UnmatchedRiskSignals, []risk.SignalID{"custom_team_signal"}) {
		t.Fatalf("未被声明的 unknown 信号必须未覆盖可见: %v", cov.UnmatchedRiskSignals)
	}
	if cov.Confidence != ConfidencePartial || !cov.HasUnmatchedRisk() {
		t.Fatalf("unknown 未覆盖必须压住 confidence: %+v", cov)
	}
}

func TestMatchUnknownSignalCoveredOnlyByExplicitTrigger(t *testing.T) {
	// 项目策略显式声明该信号名 ⇒ 显式覆盖(不是静默降级)。
	p := matchGatePolicy("P-DECLARED", EnforcementGate, Trigger{RiskSignals: []string{"custom_team_signal"}})
	cov, err := Match([]Policy{p}, MatchInput{Risk: mustDerive(t, sig("custom_team_signal", "fact:e-9"))})
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if len(cov.UnknownRiskSignals) != 1 {
		t.Fatalf("被覆盖不改变 unknown 的保真露出: %v", cov.UnknownRiskSignals)
	}
	if len(cov.UnmatchedRiskSignals) != 0 || cov.Confidence != ConfidenceFull {
		t.Fatalf("显式声明的 unknown 信号视为已覆盖: %+v", cov)
	}
}

func TestMatchConfidenceClassification(t *testing.T) {
	schemaPolicy := matchGatePolicy("P-SCHEMA", EnforcementGate, Trigger{RiskSignals: []string{"schema_migration"}})
	pathPolicy := matchGatePolicy("P-PATH", EnforcementGate, Trigger{Paths: []string{"a.sql"}})
	cases := []struct {
		name string
		in   MatchInput
		want MatchConfidence
	}{
		{"无命中", MatchInput{Risk: mustDerive(t, sig(risk.SignalSchemaMigration))}, ConfidenceNone},
		{"命中且全覆盖", MatchInput{Risk: mustDerive(t, sig(risk.SignalSchemaMigration))}, ConfidenceFull},
		{"无信号有命中", MatchInput{ChangedPaths: []string{"a.sql"}}, ConfidenceFull},
	}
	sets := [][]Policy{
		{pathPolicy},   // 无命中用例
		{schemaPolicy}, // 全覆盖用例
		{pathPolicy},   // 无信号有命中用例
	}
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cov, err := Match(sets[i], tc.in)
			if err != nil {
				t.Fatalf("Match: %v", err)
			}
			if cov.Confidence != tc.want {
				t.Fatalf("confidence = %q,期望 %q", cov.Confidence, tc.want)
			}
			if !cov.Confidence.IsValid() {
				t.Fatalf("confidence %q 必须是保留值", cov.Confidence)
			}
		})
	}
}

// ---- 确定性与纯函数边界(AC3/AC4)----------------------------------------

func TestMatchDeterministicAndOrderIndependent(t *testing.T) {
	build := func() []Policy {
		return []Policy{
			matchGatePolicy("P-A", EnforcementGate, Trigger{
				RiskSignals:     []string{"schema_migration"},
				Paths:           []string{"db/mig.sql", "db/other.sql"},
				RequirementRefs: []string{".tally/requirements/REQ-018.md"},
			}),
			matchGatePolicy("P-B", EnforcementGuidance, Trigger{RiskSignals: []string{"auth_symbol_change"}, Paths: []string{"internal/auth/"}}),
			matchGatePolicy("P-C", EnforcementGate, Trigger{Paths: []string{"nope/nope"}}),
			matchGatePolicy("P-D", EnforcementGate, Trigger{RequirementRefs: []string{".tally/requirements/REQ-007.md"}}),
		}
	}
	signals := []risk.Signal{
		sig(risk.SignalSchemaMigration, "fact:e-1"),
		sig(risk.SignalAuthSymbolChange, "fact:e-2"),
		sig("team_custom", "fact:e-3"),
	}
	base, err := Match(build(), MatchInput{
		Risk:            mustDerive(t, signals...),
		ChangedPaths:    []string{"db/mig.sql", "internal/auth/session.go"},
		RequirementRefs: []string{".tally/requirements/REQ-018.md", ".tally/requirements/REQ-007.md"},
	})
	if err != nil {
		t.Fatalf("Match: %v", err)
	}

	// 全部输入倒序重排(信号轴经 Derive 天然顺序无关;列表轴与策略集手动倒序)。
	rev := make([]risk.Signal, len(signals))
	for i, s := range signals {
		rev[len(signals)-1-i] = s
	}
	reversed := build()
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}
	again, err := Match(reversed, MatchInput{
		Risk:            mustDerive(t, rev...),
		ChangedPaths:    []string{"internal/auth/session.go", "db/mig.sql"},
		RequirementRefs: []string{".tally/requirements/REQ-007.md", ".tally/requirements/REQ-018.md"},
	})
	if err != nil {
		t.Fatalf("Match(reversed): %v", err)
	}
	if !reflect.DeepEqual(base, again) {
		t.Fatalf("输入顺序不得影响结果(AC4):\n首次 %+v\n重排 %+v", base, again)
	}
	var prev string
	for _, m := range base.MatchedPolicies {
		if prev != "" && m.Policy.ID < prev {
			t.Fatalf("MatchedPolicies 未按 id 升序:%s 在 %s 后", m.Policy.ID, prev)
		}
		prev = m.Policy.ID
	}
	// 重复求值必须逐位相同(确定性)。
	for i := 0; i < 3; i++ {
		replay, err := Match(build(), MatchInput{
			Risk:            mustDerive(t, signals...),
			ChangedPaths:    []string{"db/mig.sql", "internal/auth/session.go"},
			RequirementRefs: []string{".tally/requirements/REQ-018.md", ".tally/requirements/REQ-007.md"},
		})
		if err != nil {
			t.Fatalf("第 %d 次重放: %v", i, err)
		}
		if !reflect.DeepEqual(base, replay) {
			t.Fatalf("第 %d 次重放结果不同(AC4)", i)
		}
	}
}

func TestMatchDoesNotMutateInputs(t *testing.T) {
	// AC3 的纯函数面:匹配只读输入、只产出事实值,不写任何状态。
	policies := []Policy{
		matchGatePolicy("P-A", EnforcementGate, Trigger{RiskSignals: []string{"schema_migration"}, Paths: []string{"b.sql", "a.sql"}}),
		matchGatePolicy("P-B", EnforcementGate, Trigger{RequirementRefs: []string{".tally/requirements/REQ-018.md"}}),
	}
	policiesCopy := append([]Policy(nil), policies...)
	input := MatchInput{
		Risk:            mustDerive(t, sig(risk.SignalSchemaMigration)),
		ChangedPaths:    []string{"b.sql", "a.sql"},
		RequirementRefs: []string{".tally/requirements/REQ-018.md"},
	}
	inputPaths := append([]string(nil), input.ChangedPaths...)
	inputRefs := append([]string(nil), input.RequirementRefs...)

	if _, err := Match(policies, input); err != nil {
		t.Fatalf("Match: %v", err)
	}
	if !reflect.DeepEqual(policies, policiesCopy) {
		t.Fatal("策略集输入不得被修改")
	}
	if !reflect.DeepEqual(input.ChangedPaths, inputPaths) || !reflect.DeepEqual(input.RequirementRefs, inputRefs) {
		t.Fatal("输入列表不得被修改或重排")
	}
}

// ---- fail-safe 拒绝(假阴性防线)-----------------------------------------

func TestMatchRejectsInvalidPolicyInsteadOfSkipping(t *testing.T) {
	// 源事实:策略 false-negative 会让空 required 集看似完整。
	// 非法策略必须整体报错,静默跳过等于假阴性。
	policies := []Policy{
		matchGatePolicy("P-GOOD", EnforcementGate, Trigger{Paths: []string{"a.sql"}}),
		{ID: "P-BAD", Enforcement: EnforcementGate, GateCondition: GateCondition{AllOf: []string{"ghost"}}}, // 引用未声明键
	}
	_, err := Match(policies, MatchInput{ChangedPaths: []string{"a.sql"}})
	if err == nil {
		t.Fatal("非法策略不得被静默跳过")
	}
	if !errors.Is(err, ErrInvalidPolicy) {
		t.Fatalf("期望 ErrInvalidPolicy,得到: %v", err)
	}
}

func TestMatchRejectsDuplicatePolicyIDs(t *testing.T) {
	policies := []Policy{
		matchGatePolicy("P-DUP", EnforcementGate, Trigger{Paths: []string{"a.sql"}}),
		matchGatePolicy("P-DUP", EnforcementGate, Trigger{Paths: []string{"b.sql"}}),
	}
	_, err := Match(policies, MatchInput{})
	if !errors.Is(err, ErrInvalidMatchInput) {
		t.Fatalf("期望 ErrInvalidMatchInput,得到: %v", err)
	}
}

func TestMatchRejectsMalformedRiskSignals(t *testing.T) {
	cases := []struct {
		name string
		in   MatchInput
	}{
		{"detected 空白 id", MatchInput{Risk: risk.Result{Detected: []risk.Signal{{ID: "  ", Basis: []string{"x"}}}}}},
		{"unknown 带空白 id", MatchInput{Risk: risk.Result{Unknown: []risk.Signal{{ID: " sp ", Basis: []string{"x"}}}}}},
		{"detected 与 unknown 重复", MatchInput{Risk: risk.Result{
			Detected: []risk.Signal{{ID: risk.SignalSchemaMigration, Basis: []string{"x"}}},
			Unknown:  []risk.Signal{{ID: risk.SignalSchemaMigration, Basis: []string{"y"}}},
		}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Match(nil, tc.in)
			if !errors.Is(err, ErrInvalidMatchInput) {
				t.Fatalf("期望 ErrInvalidMatchInput,得到: %v", err)
			}
		})
	}
}

func TestMatchRejectsBlankInputEntries(t *testing.T) {
	p := matchGatePolicy("P-A", EnforcementGate, Trigger{Paths: []string{"a.sql"}})
	cases := []struct {
		name string
		in   MatchInput
	}{
		{"空白路径", MatchInput{ChangedPaths: []string{"a.sql", "   "}}},
		{"空白需求引用", MatchInput{RequirementRefs: []string{"\t"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Match([]Policy{p}, tc.in)
			if !errors.Is(err, ErrInvalidMatchInput) {
				t.Fatalf("期望 ErrInvalidMatchInput,得到: %v", err)
			}
		})
	}
}

// ---- 输入为空的合法形态 ---------------------------------------------------

func TestMatchEmptyEverything(t *testing.T) {
	cov, err := Match(nil, MatchInput{})
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if len(cov.MatchedPolicies) != 0 || cov.Confidence != ConfidenceNone ||
		len(cov.DetectedRiskSignals) != 0 || len(cov.UnknownRiskSignals) != 0 || len(cov.UnmatchedRiskSignals) != 0 {
		t.Fatalf("空输入是合法结果(全空),得到 %+v", cov)
	}
}
