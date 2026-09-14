// failure.go 失败分类测试(P2-03)。
//
// 覆盖:
//   - 封闭词表:恰好 PRD §17.4 七类 + unknown,无自由字符串;
//   - AC1:每类恰好一个确定性动作(repair / escalate / retry-provider / stop);
//   - AC2:unknown / 非法值 / 解析器缺席一律 fail-safe 升级,绝不修复;
//   - AC3:分类只命名失败类 —— verdict/结论作为既成事实消费,不存在任何
//     重算或改写验收的输入;
//   - FAIL ≠ ERROR(INV-09/10):任何 error(含全部 verification 哨兵、
//     超时、取消)永不分类为 repairable;inconclusive 类即使被误送进
//     Router 也绝不产出修复裁决、不耗预算;
//   - Router 实现接线:类型化解析 → 裁决 → 驱动真实 Engine(修复路径与
//     升级路径,原因词面原样透传)。

package closedloop

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"tally/internal/verification"
)

// ---- 词表封闭性 ----

// 词表必须恰好是 PRD §17.4 的七类 + unknown;字符串值即 PRD 词面(经
// RouteEscalate 透传后成为 Escalation.Reason,必须稳定且人类可读)。
func TestFailureClassVocabularyIsClosed(t *testing.T) {
	want := map[FailureClass]bool{
		FailureRepairable:     true, // repairable
		FailureContractDefect: true, // contract_defect
		FailurePolicyConflict: true, // policy_conflict
		FailureEnvironment:    true, // environment_failure
		FailureProvider:       true, // provider_failure
		FailureInconclusive:   true, // verification_inconclusive
		FailureRequiresHuman:  true, // requires_human_judgment
		FailureUnknown:        true, // unknown
	}
	if len(allFailureClasses) != len(want) {
		t.Fatalf("词表元素数 = %d, 期望 %d(封闭集被扩大或收窄)", len(allFailureClasses), len(want))
	}
	for _, c := range allFailureClasses {
		if !want[c] {
			t.Errorf("意外词表元素: %q", string(c))
		}
		if !c.IsValid() {
			t.Errorf("%q 应为合法词表元素", string(c))
		}
	}
	for _, bad := range []FailureClass{"", "Repairable", "retry", "no_candidate", "failed"} {
		if bad.IsValid() {
			t.Errorf("非法值 %q 不应通过 IsValid", string(bad))
		}
	}
	// inconclusive 词面与 Engine 内部升级原因常量一致(ERROR 语义全链路同名)。
	if FailureInconclusive != FailureClass(EscalationInconclusive) {
		t.Errorf("FailureInconclusive = %q, 期望与 EscalationInconclusive(%q)同词面",
			string(FailureInconclusive), string(EscalationInconclusive))
	}
}

// ---- AC1:每类恰好一个确定性动作 ----

func TestFailureClassActionIsDeterministic(t *testing.T) {
	cases := []struct {
		cls  FailureClass
		want ClassAction
	}{
		{FailureRepairable, ActionRepair},
		{FailureContractDefect, ActionEscalate},
		{FailurePolicyConflict, ActionEscalate},
		{FailureEnvironment, ActionRetryProvider},
		{FailureProvider, ActionRetryProvider},
		{FailureInconclusive, ActionStop},
		{FailureRequiresHuman, ActionEscalate},
		{FailureUnknown, ActionEscalate},
	}
	for _, tc := range cases {
		if got := tc.cls.Action(); got != tc.want {
			t.Errorf("%s.Action() = %s, 期望 %s", tc.cls, got, tc.want)
		}
		// 确定性:同一输入重复求值结果恒等(纯函数,无时钟/随机)。
		if again := tc.cls.Action(); again != tc.want {
			t.Errorf("%s.Action() 二次求值 = %s, 期望 %s", tc.cls, again, tc.want)
		}
	}
	// AC2:词表之外的值没有第四种命运 —— fail-safe 升级,绝不修复。
	for _, bad := range []FailureClass{"", "bogus"} {
		if got := bad.Action(); got != ActionEscalate {
			t.Errorf("非法值 %q 的动作 = %s, 期望 fail-safe 升级", string(bad), got)
		}
	}
	// 动作词面与 AC1 一致(repair / escalate / retry-provider / stop)。
	names := map[ClassAction]string{
		ActionRepair:        "repair",
		ActionRetryProvider: "retry_provider",
		ActionEscalate:      "escalate",
		ActionStop:          "stop",
	}
	for a, want := range names {
		if got := a.String(); got != want {
			t.Errorf("ClassAction(%d).String() = %q, 期望 %q", a, got, want)
		}
	}
}

// ---- 错误分类:只经类型化判因,不做字符串嗅探 ----

func TestClassifyErrorTypedSentinels(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want FailureClass
	}{
		// environment_failure:环境/执行边界资源失败与超时。
		{"预算耗尽", fmt.Errorf("wrap: %w", verification.ErrBudgetExhausted), FailureEnvironment},
		{"workspace 缺失", fmt.Errorf("wrap: %w", verification.ErrNoWorkspace), FailureEnvironment},
		{"环境身份漂移", fmt.Errorf("wrap: %w", verification.ErrEnvironmentMismatch), FailureEnvironment},
		{"执行边界产出畸形", fmt.Errorf("wrap: %w", verification.ErrInvalidRunOutput), FailureEnvironment},
		{"超时", fmt.Errorf("wrap: %w", context.DeadlineExceeded), FailureEnvironment},
		// contract_defect:契约自身缺陷(漏声明)。
		{"契约漏声明 runtime 能力", fmt.Errorf("wrap: %w", verification.ErrRuntimeRequirementsUnmet), FailureContractDefect},
		// policy_conflict:§8.7.2 冻结纪律与决策锁 12 的违反。
		{"超停止规则重跑", fmt.Errorf("wrap: %w", verification.ErrRunBeyondStoppingRule), FailurePolicyConflict},
		{"同序号覆盖", fmt.Errorf("wrap: %w", verification.ErrRunIndexAlreadyRecorded), FailurePolicyConflict},
		{"seed 换用", fmt.Errorf("wrap: %w", verification.ErrSeedMismatch), FailurePolicyConflict},
		{"一次性证据再消耗", fmt.Errorf("wrap: %w", verification.ErrBindingAlreadyExecuted), FailurePolicyConflict},
		// verification_inconclusive:有协议纪律但无可用 verdict。
		{"停止规则未满足", fmt.Errorf("wrap: %w", verification.ErrStoppingRuleIncomplete), FailureInconclusive},
		// requires_human_judgment:保护证据链完整性被破坏(unmatched risk)。
		{"held-out 完整性", fmt.Errorf("wrap: %w", verification.ErrHoldoutIntegrity), FailureRequiresHuman},
		// unknown fail-safe:misuse/接线哨兵、取消、未知 error、nil。
		{"请求非法", fmt.Errorf("wrap: %w", verification.ErrInvalidRequest), FailureUnknown},
		{"held-out 通道缺失", fmt.Errorf("wrap: %w", verification.ErrHeldOutChannelMissing), FailureUnknown},
		{"依赖为 nil", fmt.Errorf("wrap: %w", verification.ErrNilDependency), FailureUnknown},
		{"协议配对不一致", fmt.Errorf("wrap: %w", verification.ErrProtocolMismatch), FailureUnknown},
		{"取消不是失败", context.Canceled, FailureUnknown},
		{"未知 error", errors.New("某个未分类错误"), FailureUnknown},
		{"nil 不是失败", nil, FailureUnknown},
	}
	for _, tc := range cases {
		if got := ClassifyError(tc.err); got != tc.want {
			t.Errorf("ClassifyError(%s) = %s, 期望 %s", tc.name, got, tc.want)
		}
	}
}

// FAIL ≠ ERROR(INV-09/10)的结构守卫:任何 error 都不构成候选失败 ——
// 分类永远不能把「无可信 verdict」的执行错误变成 repairable(进修复)。
func TestClassifyErrorNeverRepairable(t *testing.T) {
	errs := []error{
		verification.ErrInvalidRequest, verification.ErrHeldOutChannelMissing,
		verification.ErrHeldOutNotIncremental, verification.ErrUnknownBinding,
		verification.ErrNoWorkspace, verification.ErrBudgetExhausted,
		verification.ErrRunBeyondStoppingRule, verification.ErrRunIndexAlreadyRecorded,
		verification.ErrSeedMismatch, verification.ErrEnvironmentMismatch,
		verification.ErrStoppingRuleIncomplete,
		verification.ErrNilDependency, verification.ErrNoHeldOutBinding,
		verification.ErrRuntimeRequirementsUnmet, verification.ErrBindingNotFound,
		verification.ErrBindingNotHeldOut, verification.ErrExecutionContextMismatch,
		verification.ErrCandidateMismatch, verification.ErrProtocolMismatch,
		verification.ErrBindingAlreadyExecuted, verification.ErrHoldoutIntegrity,
		verification.ErrInvalidRunOutput,
		context.DeadlineExceeded, context.Canceled,
		errors.New("plain"),
		nil,
	}
	for _, err := range errs {
		if got := ClassifyError(err); got == FailureRepairable {
			t.Errorf("ClassifyError(%v) = repairable:error 绝不能伪装成候选失败", err)
		}
	}
}

// ---- AC3:分类只命名,不重释验收 ----
// FinalStatus 是编排器的既成结论;分类没有验收材料输入,不存在改写路径。
func TestClassifyFinalStatus(t *testing.T) {
	cases := []struct {
		status verification.FinalStatus
		want   FailureClass
	}{
		{verification.FinalFailed, FailureRepairable},
		{verification.FinalNeedsReview, FailureRequiresHuman},
		{verification.FinalVerified, FailureUnknown}, // verified 不是失败:分类不适用,fail-safe
		{verification.FinalStatus("bogus"), FailureUnknown},
	}
	for _, tc := range cases {
		if got := ClassifyFinalStatus(tc.status); got != tc.want {
			t.Errorf("ClassifyFinalStatus(%q) = %s, 期望 %s", tc.status, got, tc.want)
		}
	}
}

// ---- 裁决投影:仅 repairable 可产出修复 ----

func TestRulingForClass(t *testing.T) {
	for _, c := range allFailureClasses {
		r := RulingForClass(c)
		switch c {
		case FailureRepairable:
			if r.Route != RouteRepair || r.Reason != "" {
				t.Errorf("repairable 裁决 = %+v, 期望 RouteRepair 无原因", r)
			}
		default:
			if r.Route != RouteEscalate {
				t.Errorf("%s 的裁决路线 = %s, 期望升级(只有 repairable 进修复)", c, r.Route)
			}
			if r.Reason != EscalationReason(c) {
				t.Errorf("%s 的升级原因 = %q, 期望词面原样 %q", c, r.Reason, string(c))
			}
			if !r.Reason.IsValid() {
				t.Errorf("%s 的升级原因为空(Engine 契约:升级裁决必须带原因)", c)
			}
		}
	}
	// 非法值 → unknown 投影(fail-safe 升级,AC2)。
	r := RulingForClass("")
	if r.Route != RouteEscalate || r.Reason != EscalationReason(FailureUnknown) {
		t.Errorf("非法值裁决 = %+v, 期望升级且原因 unknown", r)
	}
}

// ---- Router 实现:解析 → 裁决,fail-safe ----

func TestFailureRouterRoutes(t *testing.T) {
	f := VerifyFailure{TaskID: "TASK-1", Attempt: 1, ChangeSet: "cs-1", RunRef: "vrun-1"}

	t.Run("nil 解析器 fail-safe 升级", func(t *testing.T) {
		ruling, err := NewFailureRouter(nil).RouteFailure(context.Background(), f)
		if err != nil {
			t.Fatalf("RouteFailure: %v", err)
		}
		if ruling.Route != RouteEscalate || ruling.Reason != EscalationReason(FailureUnknown) {
			t.Errorf("裁决 = %+v, 期望升级且原因 unknown", ruling)
		}
	})

	t.Run("repairable 判定才修复", func(t *testing.T) {
		var seen VerifyFailure
		router := NewFailureRouter(func(vf VerifyFailure) FailureClass {
			seen = vf
			return FailureRepairable
		})
		ruling, err := router.RouteFailure(context.Background(), f)
		if err != nil {
			t.Fatalf("RouteFailure: %v", err)
		}
		if ruling.Route != RouteRepair {
			t.Errorf("裁决 = %+v, 期望修复", ruling)
		}
		// 入参原样透传:只有 id 与 verdict 谱系,解析器不见任何 held-out 通道。
		if seen != f {
			t.Errorf("解析器入参 = %+v, 期望 %+v", seen, f)
		}
	})

	t.Run("非 repairable 带词面原因升级", func(t *testing.T) {
		router := NewFailureRouter(func(VerifyFailure) FailureClass { return FailureContractDefect })
		ruling, err := router.RouteFailure(context.Background(), f)
		if err != nil {
			t.Fatalf("RouteFailure: %v", err)
		}
		if ruling.Route != RouteEscalate || ruling.Reason != "contract_defect" {
			t.Errorf("裁决 = %+v, 期望升级且原因 contract_defect", ruling)
		}
	})

	t.Run("解析器返回非法值按 unknown 处理", func(t *testing.T) {
		router := NewFailureRouter(func(VerifyFailure) FailureClass { return FailureClass("") })
		ruling, err := router.RouteFailure(context.Background(), f)
		if err != nil {
			t.Fatalf("RouteFailure: %v", err)
		}
		if ruling.Route != RouteEscalate || ruling.Reason != EscalationReason(FailureUnknown) {
			t.Errorf("裁决 = %+v, 期望 fail-safe 升级且原因 unknown", ruling)
		}
	})
}

// ---- Router 实现 × 真实 Engine:分类裁决驱动闭环 ----

// 可信 FAIL 被解析为 repairable → 修复一次 → 复验通过接受;取代事实与
// basis 完整(与 P2-02 引擎不变量咬合)。
func TestFailureRouterDrivesEngineRepair(t *testing.T) {
	cs1 := candidate(t, "c1")
	cs2 := candidate(t, "c2")
	exec := &fakeExecutor{steps: []execStep{{cand: cs1}, {cand: cs2}}}
	verify := &fakeVerifier{steps: []verifyStep{
		{verdict: VerdictFailed, runRef: "vrun-a"},
		{verdict: VerdictVerified, runRef: "vrun-b"},
	}}
	var routed []VerifyFailure
	router := NewFailureRouter(func(vf VerifyFailure) FailureClass {
		routed = append(routed, vf)
		return FailureRepairable
	})
	budget := &fakeBudget{maxRepairNo: 2}
	lineage := &memLineage{}
	e := newEngine(t, Config{
		Executor: exec, Verifier: verify, Router: router,
		RepairBudget: budget, LineageSink: lineage,
	})

	res, err := e.Run(context.Background(), "TASK-1")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Phase != PhaseAccepted || res.Basis == nil {
		t.Fatalf("期望 Accepted 带 basis,得到 %s / %+v", res.Phase, res.Basis)
	}
	if res.Basis.ChangeSet != cs2.ID() || res.Basis.Attempt != 2 || res.Basis.VerifyRun != "vrun-b" {
		t.Errorf("Basis = %+v, 期望绑定第二个候选/第二次尝试/vrun-b", res.Basis)
	}
	if len(routed) != 1 || routed[0].ChangeSet != cs1.ID() || routed[0].RunRef != "vrun-a" {
		t.Errorf("Router 被咨询记录 = %+v, 期望恰一次且绑定 cs1/vrun-a", routed)
	}
	if len(lineage.facts) != 1 || lineage.facts[0].Old != cs1.ID() || lineage.facts[0].New != cs2.ID() {
		t.Errorf("取代事实 = %+v, 期望恰一条 cs1→cs2", lineage.facts)
	}
}

// 非 repairable 一律升级,原因词面原样成为 Escalation.Reason;绝不消耗
// 修复预算、绝不产生取代事实(AC2 / FAIL≠ERROR)。
func TestFailureRouterEscalatesNonRepairable(t *testing.T) {
	cases := []struct {
		name     string
		resolved FailureClass
		wantRe   EscalationReason
	}{
		{"contract_defect", FailureContractDefect, "contract_defect"},
		{"policy_conflict", FailurePolicyConflict, "policy_conflict"},
		{"requires_human_judgment", FailureRequiresHuman, "requires_human_judgment"},
		{"unknown", FailureUnknown, "unknown"},
		// 解析器把可信 FAIL 误标为 inconclusive:也不许修复 —— stop 语义在
		// Engine 词表内的落点是带同名原因的升级,绝不耗预算。
		{"inconclusive 误送", FailureInconclusive, "verification_inconclusive"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			exec := &fakeExecutor{steps: []execStep{{cand: candidate(t, "c1")}}}
			verify := &fakeVerifier{steps: []verifyStep{{verdict: VerdictFailed, runRef: "vrun-a"}}}
			router := NewFailureRouter(func(VerifyFailure) FailureClass { return tc.resolved })
			budget := &fakeBudget{maxRepairNo: 9}
			lineage := &memLineage{}
			e := newEngine(t, Config{
				Executor: exec, Verifier: verify, Router: router,
				RepairBudget: budget, LineageSink: lineage,
			})

			res, err := e.Run(context.Background(), "TASK-1")
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if res.Phase != PhaseEscalated || res.Basis != nil {
				t.Fatalf("期望 Escalated 且无 basis,得到 %s / %+v", res.Phase, res.Basis)
			}
			if res.Escalation == nil || res.Escalation.Reason != tc.wantRe {
				t.Errorf("升级原因 = %+v, 期望 %q", res.Escalation, tc.wantRe)
			}
			if len(budget.asked) != 0 {
				t.Errorf("修复预算被询问 = %+v, 非 repairable 绝不进修复", budget.asked)
			}
			if len(lineage.facts) != 0 {
				t.Errorf("取代事实 = %+v, 升级路径不得产生取代", lineage.facts)
			}
			if len(exec.calls) != 1 {
				t.Errorf("执行次数 = %d, 期望 1(不重试)", len(exec.calls))
			}
		})
	}
}
