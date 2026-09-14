package contract

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"tally/internal/assurance"
	"tally/internal/engineeringpolicy"
	"tally/internal/risk"
	"tally/internal/verification"
)

// bindTaskYAML 编译器用的任务合同:带 requirement_refs(匹配的任务输入轴)。
const bindTaskYAML = `
id: TASK-201
profile: standard
title: contract bind
objective: bind matched policies, required evidence and requires context to the pinned contracts
requirement_refs:
  - .tally/requirements/REQ-042.md
scope:
  allowed_paths:
    - internal/contract/**
acceptance:
  - id: AC-01
    statement: migration upgrade path verified on a fresh database
    independence_required: none
  - id: AC-02
    statement: no unmatched risk silently weakens assurance
    independence_required: none
`

// bindGatePolicyYAML 一条 gate 策略:requires 含三个键,gate_condition 只要求
// 其中两个 —— requires[](Prevention 全量)与 gate keys(Assurance 子集)的
// 差异是 PRD §8.3 双重用途测试的核心。
const bindGatePolicyYAML = `
id: policy-migration
enforcement: gate
rationale: migration-class changes must prove upgrade path and data integrity on a fresh database
trigger:
  risk_signals:
    - schema_migration
requires:
  - fresh_database
  - n_minus_1_upgrade
  - data_integrity
gate_condition:
  all_of:
    - data_integrity
    - fresh_database
`

// bindGuidancePolicyYAML 一条 guidance 策略:有 requires、没有阻断权
// (engineeringpolicy.Validate 禁止 guidance 携带 gate_condition)。
const bindGuidancePolicyYAML = `
id: policy-auth-guidance
enforcement: guidance
rationale: auth changes need a self-review of token storage approach
trigger:
  risk_signals:
    - auth_symbol_change
requires:
  - token_storage_self_review
`

// held-out 标记内容:出现在冻结契约里,但绝不许出现在执行前 requires 上下文。
const (
	bindHeldOutStore = "holdout"
	bindHeldOutRef   = "corpus/secret-12.json"
	bindHeldOutBind  = "B-HOLD"
	bindVisibleBind  = "B-BUILD"
)

// mustFrozenBind 一份含 visible + held_out 绑定与 required/非 required gate 的
// 冻结验证契约(TASK-201)。
func mustFrozenBind(t *testing.T, taskID string) *verification.Frozen {
	t.Helper()
	c := &verification.Contract{
		TaskID:   taskID,
		Revision: 2,
		Acceptances: []verification.Acceptance{
			{
				ID:        "AC-01",
				Statement: "build gate passes",
				EvidenceBindings: []verification.Binding{{
					ID:               bindVisibleBind,
					AuthoringMode:    verification.AuthoringPreCommit,
					VisibilityPolicy: verification.VisibilityVisible,
					Evidence:         verification.EvidenceRef{Type: verification.EvidenceBuild, Path: "Makefile"},
				}},
			},
			{
				ID:        "AC-02",
				Statement: "held-out verifier passes",
				EvidenceBindings: []verification.Binding{{
					ID:               bindHeldOutBind,
					AuthoringMode:    verification.AuthoringPreCommit,
					VisibilityPolicy: verification.VisibilityHeldOut,
					Evidence:         verification.EvidenceRef{Type: verification.EvidenceTest, Store: bindHeldOutStore, Ref: bindHeldOutRef},
				}},
			},
		},
		RuntimeRequirements: verification.RuntimeRequirements{
			SessionIsolation:   true,
			HeldOutUnreachable: true,
			ConformanceClass:   "verification-isolation-v1",
		},
		RequiredGates: map[string]verification.Gate{
			"build": {Required: true, Command: &verification.CommandSpec{Argv: []string{"go", "build", "./..."}}},
			"lint":  {Required: false, Command: &verification.CommandSpec{Argv: []string{"go", "vet", "./..."}}},
		},
	}
	frozen, err := verification.FreezeContract(c, verification.FreezePolicy{})
	if err != nil {
		t.Fatalf("FreezeContract(%s): %v", taskID, err)
	}
	return frozen
}

// mustPolicy 解析工程策略 YAML(填充内容修订 pin),失败即测试失败。
func mustPolicy(t *testing.T, data string) engineeringpolicy.Policy {
	t.Helper()
	p, err := engineeringpolicy.Parse([]byte(data))
	if err != nil {
		t.Fatalf("engineeringpolicy.Parse: %v", err)
	}
	return p
}

// mustRisk 从信号 id 派生风险集(带 basis ref),失败即测试失败。
func mustRisk(t *testing.T, ids ...risk.SignalID) risk.Result {
	t.Helper()
	sigs := make([]risk.Signal, 0, len(ids))
	for _, id := range ids {
		sigs = append(sigs, risk.Signal{ID: id, Basis: []string{"intent:TASK-201"}})
	}
	r, err := risk.Derive(sigs)
	if err != nil {
		t.Fatalf("risk.Derive: %v", err)
	}
	return r
}

// mustBind 组装标准编译夹具:snapshot + 两条命中策略 + 风险事实
// (schema_migration 与 auth_symbol_change,分别被两条策略声明覆盖)。
func mustBind(t *testing.T) (*Compiled, *Snapshot) {
	t.Helper()
	snap, err := Compile(mustTask(t, bindTaskYAML), mustFrozenBind(t, "TASK-201"))
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	policies := []engineeringpolicy.Policy{
		mustPolicy(t, bindGatePolicyYAML),
		mustPolicy(t, bindGuidancePolicyYAML),
	}
	c, err := Bind(snap, policies, mustRisk(t, risk.SignalSchemaMigration, risk.SignalAuthSymbolChange))
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}
	return c, snap
}

// AC1:编译器钉住确切的目标修订 —— Pins 原样来自 snapshot,身份哈希是
// snapshot 身份 + 全部绑定内容的确定性函数;同一输入(含切片顺序)重复编译
// 得到同一 Compiled(AC4)。
func TestBindPinsSnapshotAndIsDeterministic(t *testing.T) {
	c, snap := mustBind(t)

	if c.Snapshot() != snap {
		t.Fatalf("Snapshot() 不是被编译的 snapshot")
	}
	if c.Pins() != snap.Pins() {
		t.Fatalf("Pins() = %+v, want snapshot pins %+v(编译器不改写 pin)", c.Pins(), snap.Pins())
	}
	if c.TaskID() != "TASK-201" {
		t.Fatalf("TaskID() = %q", c.TaskID())
	}
	if !strings.HasPrefix(c.Hash(), "sha256:") || len(c.Hash()) != len("sha256:")+64 {
		t.Fatalf("hash 格式 = %q", c.Hash())
	}

	// 同一输入,策略集与风险信号换序:同哈希、同投影(AC4)。
	policies := []engineeringpolicy.Policy{
		mustPolicy(t, bindGuidancePolicyYAML),
		mustPolicy(t, bindGatePolicyYAML),
	}
	c2, err := Bind(snap, policies, mustRisk(t, risk.SignalAuthSymbolChange, risk.SignalSchemaMigration))
	if err != nil {
		t.Fatalf("Bind(reordered): %v", err)
	}
	if c2.Hash() != c.Hash() {
		t.Fatalf("同一输入两次编译哈希不同:%s != %s", c2.Hash(), c.Hash())
	}
	if !reflect.DeepEqual(c2.Coverage(), c.Coverage()) {
		t.Fatalf("同一输入两次编译 coverage 不同")
	}
	if !reflect.DeepEqual(c2.Clauses(), c.Clauses()) {
		t.Fatalf("同一输入两次编译 clauses 不同")
	}
	if !reflect.DeepEqual(c2.Requires(), c.Requires()) {
		t.Fatalf("同一输入两次编译 requires 不同")
	}
	if !reflect.DeepEqual(c2.GateKeys(), c.GateKeys()) {
		t.Fatalf("同一输入两次编译 gate keys 不同")
	}
	if !reflect.DeepEqual(c2.FrozenGates(), c.FrozenGates()) {
		t.Fatalf("同一输入两次编译 frozen gates 不同")
	}

	// 风险事实变化(未声明信号 → fail-safe)→ 哈希必变。
	c3, err := Bind(snap, policies, mustRisk(t, "novel_signal"))
	if err != nil {
		t.Fatalf("Bind(other risk): %v", err)
	}
	if c3.Hash() == c.Hash() {
		t.Fatalf("风险事实变化后哈希未变:%s", c.Hash())
	}
}

// PRD §8.3 requires[] 双重用途:同一份 requires 同时编译成
//   - Prevention:RequiresContext(全量 requires,含 guidance 策略);
//   - Assurance:GateKeys()(仅 gate 策略的 gate_condition.all_of)。
func TestBindCompilesRequiresDoubleDuty(t *testing.T) {
	c, _ := mustBind(t)

	cov := c.Coverage()
	if len(cov.MatchedPolicies) != 2 {
		t.Fatalf("matched policies = %d, want 2", len(cov.MatchedPolicies))
	}
	if cov.Confidence != engineeringpolicy.ConfidenceFull {
		t.Fatalf("confidence = %q, want full(两个信号都被命中策略声明)", cov.Confidence)
	}
	// match basis 原样来自 P1-05 seam(按 Policy.ID 升序)。
	if got := cov.MatchedPolicies[0].Policy.ID; got != "policy-auth-guidance" {
		t.Fatalf("first matched policy = %q", got)
	}
	if got := cov.MatchedPolicies[1].Policy.ID; got != "policy-migration" {
		t.Fatalf("second matched policy = %q", got)
	}
	if want := []risk.SignalID{"schema_migration"}; !reflect.DeepEqual(cov.MatchedPolicies[1].Basis.RiskSignals, want) {
		t.Fatalf("migration basis risk signals = %v, want %v", cov.MatchedPolicies[1].Basis.RiskSignals, want)
	}

	// Assurance 面:只有 gate 策略的 gate_condition 键;guidance 的 requires
	// 不进阻断门禁。
	if want := []string{"data_integrity", "fresh_database"}; !reflect.DeepEqual(c.GateKeys(), want) {
		t.Fatalf("GateKeys() = %v, want %v", c.GateKeys(), want)
	}

	// Prevention 面:每条命中策略一条,requires 全量(gate 策略的三个键都在,
	// 不裁剪到 gate_condition 子集);按 PolicyID 升序。
	req := c.Requires()
	if len(req.Items) != 2 {
		t.Fatalf("requires items = %d, want 2", len(req.Items))
	}
	mig := req.Items[1]
	if mig.PolicyID != "policy-migration" || mig.Enforcement != engineeringpolicy.EnforcementGate {
		t.Fatalf("migration item = %+v", mig)
	}
	if want := []string{"data_integrity", "fresh_database", "n_minus_1_upgrade"}; !reflect.DeepEqual(mig.Keys, want) {
		t.Fatalf("migration requires = %v, want %v(Prevention 面不裁剪 requires)", mig.Keys, want)
	}
	guid := req.Items[0]
	if guid.PolicyID != "policy-auth-guidance" || guid.Enforcement != engineeringpolicy.EnforcementGuidance {
		t.Fatalf("guidance item = %+v", guid)
	}
	if want := []string{"token_storage_self_review"}; !reflect.DeepEqual(guid.Keys, want) {
		t.Fatalf("guidance requires = %v, want %v", guid.Keys, want)
	}

	// 人类可读:固定句式 + 策略理由,确定性渲染。
	lines := req.Lines()
	if len(lines) != 2 {
		t.Fatalf("requires lines = %d, want 2", len(lines))
	}
	joined := strings.Join(lines, "\n")
	for _, want := range []string{
		"policy policy-migration (gate): 最终会按这些条件验收: data_integrity, fresh_database, n_minus_1_upgrade",
		"migration-class changes must prove upgrade path",
		"policy policy-auth-guidance (guidance): 最终会按这些条件验收: token_storage_self_review",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("requires 行缺少 %q:\n%s", want, joined)
		}
	}
}

// AC2:required evidence 不得弱于冻结 VerificationContract —— 逐条款全量保留
// 冻结绑定(含 held_out 的 store/ref,这是验证权威面;实现侧 redacted 投影归
// contextpack),required gate 只保留 required=true 的。
func TestBindRequiredEvidenceNotWeakerThanFrozen(t *testing.T) {
	c, _ := mustBind(t)

	clauses := c.Clauses()
	if len(clauses) != 2 {
		t.Fatalf("clauses = %d, want 2", len(clauses))
	}
	if clauses[0].AcceptanceID != "AC-01" || clauses[1].AcceptanceID != "AC-02" {
		t.Fatalf("clause ids = %s/%s, want AC-01/AC-02(按 acceptance id 升序)", clauses[0].AcceptanceID, clauses[1].AcceptanceID)
	}

	ev1 := clauses[0].Evidence
	if len(ev1) != 1 || ev1[0].BindingID != bindVisibleBind || ev1[0].Visibility != verification.VisibilityVisible {
		t.Fatalf("AC-01 evidence = %+v", ev1)
	}
	if ev1[0].Evidence.Path != "Makefile" || ev1[0].Evidence.Type != verification.EvidenceBuild {
		t.Fatalf("AC-01 evidence ref = %+v", ev1[0].Evidence)
	}

	// held_out 绑定全量保留:store/ref 不丢失(丢掉即弱于冻结契约)。
	ev2 := clauses[1].Evidence
	if len(ev2) != 1 || ev2[0].BindingID != bindHeldOutBind || ev2[0].Visibility != verification.VisibilityHeldOut {
		t.Fatalf("AC-02 evidence = %+v", ev2)
	}
	if ev2[0].Evidence.Store != bindHeldOutStore || ev2[0].Evidence.Ref != bindHeldOutRef {
		t.Fatalf("AC-02 held_out 定位丢失:%+v(AC2:不得弱于冻结契约)", ev2[0].Evidence)
	}

	// 冻结契约的 required gates:只收 required=true;命令规格保留。
	gates := c.FrozenGates()
	if len(gates) != 1 || gates[0].Name != "build" {
		t.Fatalf("frozen gates = %+v, want only required build gate", gates)
	}
	if gates[0].Command == nil || !reflect.DeepEqual(gates[0].Command.Argv, []string{"go", "build", "./..."}) {
		t.Fatalf("build gate command = %+v", gates[0].Command)
	}

	// 必需保证档:两个信号都被命中策略声明覆盖 → 常规风险上调(elevated → L3),
	// 决策链记录 uplift 理由,而非 fail-safe。
	for _, cl := range clauses {
		if cl.Assurance.Required != assurance.LevelL3 {
			t.Fatalf("clause %s required = %s, want L3(风险上调)", cl.AcceptanceID, cl.Assurance.Required)
		}
		if cl.Assurance.Basis.RiskUnknown {
			t.Fatalf("clause %s 不应触发 fail-safe(信号全部已声明覆盖)", cl.AcceptanceID)
		}
		if !reflect.DeepEqual(cl.Assurance.Basis.Reasons, []assurance.Reason{assurance.ReasonDefaultFloor, assurance.ReasonRiskUplift}) {
			t.Fatalf("clause %s reasons = %v", cl.AcceptanceID, cl.Assurance.Basis.Reasons)
		}
	}
}

// 最小风险夹具:无任何风险信号 → 维持 Lock #21 的 L1 默认地板。
func TestBindMinimalRiskKeepsL1Floor(t *testing.T) {
	snap, err := Compile(mustTask(t, bindTaskYAML), mustFrozenBind(t, "TASK-201"))
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	policies := []engineeringpolicy.Policy{mustPolicy(t, bindGatePolicyYAML)}
	c, err := Bind(snap, policies, risk.Result{})
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}
	for _, cl := range c.Clauses() {
		if cl.Assurance.Required != assurance.LevelL1 {
			t.Fatalf("clause %s required = %s, want L1(默认地板,Lock #21)", cl.AcceptanceID, cl.Assurance.Required)
		}
		if !reflect.DeepEqual(cl.Assurance.Basis.Reasons, []assurance.Reason{assurance.ReasonDefaultFloor}) {
			t.Fatalf("clause %s reasons = %v", cl.AcceptanceID, cl.Assurance.Basis.Reasons)
		}
	}
}

// INV-V14-09 / PRD §13.4:未被任何命中策略声明的风险信号(含 unknown)绝不静默
// —— 经 Coverage() 保持可见,并把全部条款的必需档 fail-safe 上调到 L3
// (ReasonUnmatchedRiskFailsafe),绝不落到 L0/L1。
func TestBindUnmatchedRiskFailsafeVisible(t *testing.T) {
	snap, err := Compile(mustTask(t, bindTaskYAML), mustFrozenBind(t, "TASK-201"))
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	policies := []engineeringpolicy.Policy{mustPolicy(t, bindGatePolicyYAML)}
	c, err := Bind(snap, policies, mustRisk(t, risk.SignalSchemaMigration, "novel_signal"))
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}

	cov := c.Coverage()
	if !cov.HasUnmatchedRisk() {
		t.Fatalf("HasUnmatchedRisk() = false, want true")
	}
	if want := []risk.SignalID{"novel_signal"}; !reflect.DeepEqual(cov.UnmatchedRiskSignals, want) {
		t.Fatalf("unmatched = %v, want %v", cov.UnmatchedRiskSignals, want)
	}
	if want := []risk.SignalID{"novel_signal"}; !reflect.DeepEqual(cov.UnknownRiskSignals, want) {
		t.Fatalf("unknown = %v, want %v", cov.UnknownRiskSignals, want)
	}
	if cov.Confidence != engineeringpolicy.ConfidencePartial {
		t.Fatalf("confidence = %q, want partial", cov.Confidence)
	}

	for _, cl := range c.Clauses() {
		if cl.Assurance.Required != assurance.LevelL3 {
			t.Fatalf("clause %s required = %s, want L3(unmatched fail-safe,绝不最弱档)", cl.AcceptanceID, cl.Assurance.Required)
		}
		if !cl.Assurance.Basis.RiskUnknown {
			t.Fatalf("clause %s RiskUnknown = false, want true", cl.AcceptanceID)
		}
		last := cl.Assurance.Basis.Reasons[len(cl.Assurance.Basis.Reasons)-1]
		if last != assurance.ReasonUnmatchedRiskFailsafe {
			t.Fatalf("clause %s reasons = %v, want 末条为 ReasonUnmatchedRiskFailsafe", cl.AcceptanceID, cl.Assurance.Basis.Reasons)
		}
	}
}

// AC3 / INV-V14-07:执行前 requires 上下文人类可读,但绝不携带冻结验证契约的
// held_out 定位/内容;也不携带任何验证器定位(requires 上下文只来自策略本身的
// 公开声明;held-out 的 redacted 投影归 contextpack)。
func TestBindRequiresContextExcludesHeldOut(t *testing.T) {
	c, _ := mustBind(t)

	req := c.Requires()
	rendered := strings.Join(req.Lines(), "\n")
	if rendered == "" {
		t.Fatalf("requires 上下文为空")
	}
	for _, secret := range []string{bindHeldOutStore, bindHeldOutRef, bindHeldOutBind, bindVisibleBind, "Makefile", "secret-12"} {
		if strings.Contains(rendered, secret) {
			t.Fatalf("执行前 requires 上下文泄漏 %q:\n%s", secret, rendered)
		}
	}
}

// Bind 拒绝: nil snapshot、缺少修订 pin 的策略、非法策略集。错误用 errors.Is 判类。
func TestBindRejectsInvalidInputs(t *testing.T) {
	snap, err := Compile(mustTask(t, bindTaskYAML), mustFrozenBind(t, "TASK-201"))
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	policies := []engineeringpolicy.Policy{mustPolicy(t, bindGatePolicyYAML)}

	// nil snapshot。
	if _, err := Bind(nil, policies, risk.Result{}); !errors.Is(err, ErrNilSnapshot) {
		t.Fatalf("Bind(nil snapshot) = %v, want ErrNilSnapshot", err)
	}

	// 未填充内容修订的策略不可钉,拒绝绑定。
	unpinned := mustPolicy(t, bindGuidancePolicyYAML)
	unpinned.Revision = ""
	if _, err := Bind(snap, []engineeringpolicy.Policy{unpinned}, risk.Result{}); !errors.Is(err, ErrUnpinnedPolicy) {
		t.Fatalf("Bind(unpinned policy) = %v, want ErrUnpinnedPolicy", err)
	}

	// 策略集非法(重复 id):透传 P1-05 哨兵。
	dup := mustPolicy(t, bindGatePolicyYAML)
	if _, err := Bind(snap, []engineeringpolicy.Policy{dup, dup}, risk.Result{}); !errors.Is(err, engineeringpolicy.ErrInvalidMatchInput) {
		t.Fatalf("Bind(duplicate ids) = %v, want ErrInvalidMatchInput", err)
	}

	// 策略自身非法(guidance 携带 gate_condition):透传 ErrInvalidPolicy。
	bad := mustPolicy(t, bindGuidancePolicyYAML)
	bad.GateCondition.AllOf = []string{"token_storage_self_review"}
	if _, err := Bind(snap, []engineeringpolicy.Policy{bad}, risk.Result{}); !errors.Is(err, engineeringpolicy.ErrInvalidPolicy) {
		t.Fatalf("Bind(invalid policy) = %v, want ErrInvalidPolicy", err)
	}
}

// 冻结输入纪律(INV-V14-02):编译不改 snapshot;实现权威/调用方改动自己的
// 输入(策略切片,含触发列表底层数组)或改动访问器返回的投影,都影响不到
// 已编译合同与其身份哈希。
func TestBindFrozenInputDiscipline(t *testing.T) {
	c, snap := mustBind(t)
	wantSnapHash := snap.Hash()
	wantHash := c.Hash()
	wantClauses := c.Clauses()
	wantReq := c.Requires()

	// 编译不改 snapshot(P1-01 snapshot 不可变,编译器不得改写)。
	if snap.Hash() != wantSnapHash {
		t.Fatalf("Bind 后 snapshot 哈希变化")
	}

	// 改动调用方手里的策略切片(值字段 + 触发列表元素):编译产物不受影响。
	policies := []engineeringpolicy.Policy{mustPolicy(t, bindGatePolicyYAML), mustPolicy(t, bindGuidancePolicyYAML)}
	c2, err := Bind(snap, policies, mustRisk(t, risk.SignalSchemaMigration, risk.SignalAuthSymbolChange))
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}
	policies[0].Rationale = "mutated after bind"
	policies[0].Trigger.RiskSignals[0] = "mutated_signal"
	policies[0].Requires[0] = "mutated_key"
	policies[0].GateCondition.AllOf[0] = "mutated_key"
	if c2.Hash() != wantHash {
		t.Fatalf("调用方改动策略后 Compiled 哈希变化:%s != %s", c2.Hash(), wantHash)
	}
	if !reflect.DeepEqual(c2.Coverage(), c.Coverage()) {
		t.Fatalf("调用方改动策略后 coverage 变化")
	}
	if !reflect.DeepEqual(c2.Requires(), wantReq) {
		t.Fatalf("调用方改动策略后 requires 变化")
	}

	// 改动访问器返回的投影:不影响已编译合同。
	clauses := c.Clauses()
	clauses[0].Evidence[0].BindingID = "MUTATED"
	clauses[0].Assurance.Basis.Reasons[0] = "mutated"
	if !reflect.DeepEqual(c.Clauses(), wantClauses) {
		t.Fatalf("通过 Clauses() 的改动泄漏进 Compiled")
	}
	req := c.Requires()
	req.Items[0].Keys[0] = "MUTATED"
	req.Items[0].Rationale = "mutated"
	if !reflect.DeepEqual(c.Requires(), wantReq) {
		t.Fatalf("通过 Requires() 的改动泄漏进 Compiled")
	}
	gates := c.FrozenGates()
	gates[0].Name = "MUTATED"
	gates[0].Command.Argv[0] = "MUTATED"
	if got := c.FrozenGates()[0].Name; got == "MUTATED" {
		t.Fatalf("通过 FrozenGates() 的改动泄漏进 Compiled")
	}
	if got := c.FrozenGates()[0].Command.Argv[0]; got == "MUTATED" {
		t.Fatalf("通过 FrozenGates().Command 的改动泄漏进 Compiled")
	}
	if c.Hash() != wantHash {
		t.Fatalf("访问器改动后 Compiled 哈希变化")
	}
}
