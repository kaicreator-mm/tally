package closedloop

// repair_policy.go 测试(P2-07):
//
//   - AC1:MaxRepairAttempts 独立可配,且必须落在硬界内;
//   - AC2:默认 = 2(PRD §17.2;锁 23:策略默认,不是架构常量);
//   - AC3:零/负/超硬上限一律拒绝(零/负不表示无界);
//   - AC4:预算耗尽经 Engine 落显式 EscalationRepairExhausted 升级;
//   - 分离钉:与 internal/policy 的 ExecutionPolicy 词表零耦合(不导入、
//     不出现内层迭代词汇);预算对 (task, attempt_no) 位置式幂等,取消/
//     Inconclusive 结构上不消耗预算(与 Engine 落点对齐,不重复记账)。
//
// 测试固件一律以 rp 前缀命名,不与 engine_test/feedback_test 的固件共享,
// 也不依赖其实现细节。

import (
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	"tally/internal/changeset"
	"tally/internal/core"
)

// ---- 固件(rp 前缀;不跨测试文件共享) ----

// rpCandidate 构造一个内容不可变候选;marker 不同则身份不同。
func rpCandidate(t *testing.T, marker string) *changeset.ChangeSet {
	t.Helper()
	cs, err := changeset.New(changeset.Spec{
		Project:      core.ProjectID("proj-p2-07"),
		BaseRevision: "base-p2-07",
		Files:        []changeset.ChangedFile{{Path: "pkg/widget.go", Content: "marker: " + marker}},
		ProducerRun:  "codegen-run-" + marker,
	})
	if err != nil {
		t.Fatalf("构造候选 ChangeSet: %v", err)
	}
	return cs
}

// rpExecutor 按脚本逐次返回全新候选;脚本耗尽即报错(多出的尝试会显式失败,
// 绝不静默放行越界循环)。
type rpExecutor struct {
	steps []*changeset.ChangeSet
}

func (f *rpExecutor) ExecuteCandidate(_ context.Context, _ ExecuteRequest) (ExecuteOutcome, error) {
	if len(f.steps) == 0 {
		return ExecuteOutcome{}, errors.New("rpExecutor: 脚本耗尽")
	}
	cs := f.steps[0]
	f.steps = f.steps[1:]
	return ExecuteOutcome{Candidate: cs}, nil
}

// rpVerifier 按脚本逐次返回验证结论;脚本耗尽即报错。
type rpVerifier struct {
	steps []VerifyOutcome
}

func (f *rpVerifier) VerifyCandidate(_ context.Context, _ VerifyRequest) (VerifyOutcome, error) {
	if len(f.steps) == 0 {
		return VerifyOutcome{}, errors.New("rpVerifier: 脚本耗尽")
	}
	o := f.steps[0]
	f.steps = f.steps[1:]
	return o, nil
}

// rpLineage 内存取代事实出口(append-only)。
type rpLineage struct {
	facts []changeset.SupersededFact
}

func (m *rpLineage) RecordSuperseded(_ context.Context, fact changeset.SupersededFact) error {
	m.facts = append(m.facts, fact)
	return nil
}

// rpCountingBudget 记录全部预算询问(断言 Engine 只在修复裁决后咨询)。
type rpCountingBudget struct {
	policy RepairPolicy
	asked  []AttemptNo
}

func (b *rpCountingBudget) CanRepair(no AttemptNo) bool {
	b.asked = append(b.asked, no)
	return b.policy.CanRepair(no)
}

// rpAlwaysRepairableRouter:真实 failureRouter + 全部分类为 repairable 的解析器。
func rpAlwaysRepairableRouter() Router {
	return NewFailureRouter(func(VerifyFailure) FailureClass { return FailureRepairable })
}

func rpNewEngine(t *testing.T, exec *rpExecutor, verify *rpVerifier, budget RepairBudget, lineage *rpLineage) *Engine {
	t.Helper()
	e, err := NewEngine(Config{
		Executor:     exec,
		Verifier:     verify,
		Router:       rpAlwaysRepairableRouter(),
		RepairBudget: budget,
		LineageSink:  lineage,
		Now:          func() time.Time { return time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	return e
}

// ---- AC2:默认 2 ----

// TestRepairPolicyDefaultIsTwo:默认策略恰好 2 次修复(策略默认值,可配置,
// 但此处钉住出厂值与 PRD §17.2 一致)。
func TestRepairPolicyDefaultIsTwo(t *testing.T) {
	def := DefaultRepairPolicy()
	if def.MaxRepairAttempts != 2 {
		t.Errorf("DefaultRepairPolicy().MaxRepairAttempts = %d, 期望 2(PRD §17.2)", def.MaxRepairAttempts)
	}
	if def.MaxRepairAttempts != DefaultMaxRepairAttempts {
		t.Errorf("默认值与 DefaultMaxRepairAttempts(%d)不一致", DefaultMaxRepairAttempts)
	}
	if err := def.Validate(); err != nil {
		t.Errorf("默认策略应合法,得到 %v", err)
	}
}

// ---- AC1/AC3:硬界与拒绝面 ----

// TestRepairPolicyValidateHardBounds:界内独立可配(AC1);零/负/超硬上限一律
// ErrInvalidRepairPolicy(AC3;零/负不表示无界,math.MaxInt 这类近无界同样拒绝)。
func TestRepairPolicyValidateHardBounds(t *testing.T) {
	valid := []int{MinRepairAttempts, 2, 3, HardMaxRepairAttempts}
	for _, n := range valid {
		p, err := NewRepairPolicy(n)
		if err != nil {
			t.Errorf("NewRepairPolicy(%d) 不应拒绝(界内可配,AC1): %v", n, err)
			continue
		}
		if p.MaxRepairAttempts != n {
			t.Errorf("NewRepairPolicy(%d).MaxRepairAttempts = %d", n, p.MaxRepairAttempts)
		}
		if err := p.Validate(); err != nil {
			t.Errorf("Validate(%d) 不应报错: %v", n, err)
		}
	}

	invalid := []int{0, -1, -100, MinRepairAttempts - 1, HardMaxRepairAttempts + 1, 1000, math.MaxInt}
	for _, n := range invalid {
		_, err := NewRepairPolicy(n)
		if !errors.Is(err, ErrInvalidRepairPolicy) {
			t.Errorf("NewRepairPolicy(%d) 应拒绝(AC3),得到 %v", n, err)
		}
	}
	// 零值策略同样非法:零不表示「不限次」。
	if err := (RepairPolicy{}).Validate(); !errors.Is(err, ErrInvalidRepairPolicy) {
		t.Errorf("零值策略应 ErrInvalidRepairPolicy,得到 %v", err)
	}
}

// ---- 修复窗口(off-by-one)与 fail-safe ----

// TestRepairPolicyCanRepairWindow:第 no 次尝试是第 no-1 次修复;默认 2 允许
// 尝试 2、3、拒绝尝试 4 —— 耗尽点恰在第 2 次修复失败之后;初始尝试(1)与
// 非法序号永不归修复预算治理。
func TestRepairPolicyCanRepairWindow(t *testing.T) {
	def := DefaultRepairPolicy()
	want := map[AttemptNo]bool{
		0: false, 1: false,
		2: true, 3: true, // 两次修复都在预算内
		4: false, 5: false, // 第 3 次修复起耗尽
	}
	for no, wantOK := range want {
		if got := def.CanRepair(no); got != wantOK {
			t.Errorf("默认策略 CanRepair(%d) = %v, 期望 %v", no, got, wantOK)
		}
	}

	// 界内其它配置的窗口端点(AC1:独立可配)。
	single, err := NewRepairPolicy(1)
	if err != nil {
		t.Fatalf("NewRepairPolicy(1): %v", err)
	}
	if !single.CanRepair(2) || single.CanRepair(3) {
		t.Error("MaxRepairAttempts=1: 只允许尝试 2(第一次修复)")
	}
	maxed, err := NewRepairPolicy(HardMaxRepairAttempts)
	if err != nil {
		t.Fatalf("NewRepairPolicy(%d): %v", HardMaxRepairAttempts, err)
	}
	if !maxed.CanRepair(AttemptNo(HardMaxRepairAttempts + 1)) {
		t.Errorf("MaxRepairAttempts=%d: 尝试 %d 应允许(最后一次修复)", HardMaxRepairAttempts, HardMaxRepairAttempts+1)
	}
	if maxed.CanRepair(AttemptNo(HardMaxRepairAttempts + 2)) {
		t.Errorf("MaxRepairAttempts=%d: 尝试 %d 应耗尽", HardMaxRepairAttempts, HardMaxRepairAttempts+2)
	}

	// fail-safe:非法策略(零值/越界)拒绝一切修复,绝不猜测。
	denied := []RepairPolicy{
		{}, // 零值
		{MaxRepairAttempts: -3},
		{MaxRepairAttempts: HardMaxRepairAttempts + 1},
	}
	for _, p := range denied {
		if p.CanRepair(2) {
			t.Errorf("非法策略 %+v 不应允许任何修复(fail-safe)", p)
		}
	}
}

// TestRepairPolicyBudgetIsPurePositional:预算是尝试序号的纯函数 —— 没有累计
// 计数、没有调用顺序效应:被拒的查询不「扣减」任何东西,同序号重复询问恒定
// (对 (task, attempt_no) 幂等,ADR-0001 §2:崩溃恢复按键,不重复消耗)。
// 取消/Inconclusive 的尝试不推进序号,结构上就不消耗预算(Engine 落点已强制,
// 本测试钉住策略侧不引入任何隐性记账)。
func TestRepairPolicyBudgetIsPurePositional(t *testing.T) {
	a := DefaultRepairPolicy()
	if a.CanRepair(4) {
		t.Fatal("前置:默认策略应拒绝尝试 4")
	}
	// 拒绝性查询之后,预算内的窗口不受影响(没有 side-effect 消耗)。
	if !a.CanRepair(2) || !a.CanRepair(3) {
		t.Error("CanRepair(4) 的拒绝不得影响后续预算内询问(预算不是累计计数器)")
	}
	// 同序号重复询问恒定(幂等)。
	for i := 0; i < 3; i++ {
		if !a.CanRepair(3) {
			t.Fatalf("重复询问 CanRepair(3) 不应变假(无状态,第 %d 次)", i)
		}
	}
	// 任意询问顺序下答案一致(顺序无关)。
	b := DefaultRepairPolicy()
	sequence := []AttemptNo{3, 4, 2, 4, 3}
	want := []bool{true, false, true, false, true}
	for i, no := range sequence {
		if got := b.CanRepair(no); got != want[i] {
			t.Errorf("顺序询问 CanRepair(%d) = %v, 期望 %v", no, got, want[i])
		}
	}
}

// ---- 分离钉:与 ExecutionPolicy 零耦合(锁 7/23) ----

// TestRepairPolicyDecoupledFromExecutionPolicy:repair_policy.go 不 import
// internal/policy(声明面只允许 errors/fmt),且任何声明标识符都不出现
// ExecutionPolicy 的冻结词汇(与 test/arch 的锁 6 片段表同源)—— 内层迭代
// 预算在类型层结构性不可达。
func TestRepairPolicyDecoupledFromExecutionPolicy(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "repair_policy.go", nil, 0)
	if err != nil {
		t.Fatalf("解析 repair_policy.go: %v", err)
	}

	allowedImports := map[string]bool{"errors": true, "fmt": true}
	for _, imp := range file.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			t.Fatalf("导入路径 %s 不可解析", imp.Path.Value)
		}
		if !allowedImports[path] {
			t.Errorf("repair_policy.go 导入 %q 超出声明面:修复策略不得触及执行/传输/内层重试词汇", path)
		}
	}

	// 与 test/arch v14_runtime_boundary_test 的 ExecutionPolicy 冻结片段一致。
	fragments := []string{
		"executionpolicy", "networkpolicy", "networkmode", "allowedhosts",
		"shellpolicy", "allowshell", "credentialpolicy", "scopedsecrets",
		"retrypolicy", "maxiterations", "stopcondition", "escalationpolicy",
		"afterfailediterations", "targetprofile",
	}
	bad := func(name string) string {
		lower := strings.ToLower(name)
		for _, frag := range fragments {
			if strings.Contains(lower, frag) {
				return frag
			}
		}
		return ""
	}

	ast.Inspect(file, func(n ast.Node) bool {
		switch decl := n.(type) {
		case *ast.TypeSpec:
			if frag := bad(decl.Name.Name); frag != "" {
				t.Errorf("类型 %s 命中 ExecutionPolicy 词汇 %q:内外策略类型不得互相渗透(锁 7)", decl.Name.Name, frag)
			}
		case *ast.FuncDecl:
			if frag := bad(decl.Name.Name); frag != "" {
				t.Errorf("函数 %s 命中 ExecutionPolicy 词汇 %q(锁 7)", decl.Name.Name, frag)
			}
		case *ast.Field:
			if decl.Names == nil {
				return true
			}
			for _, id := range decl.Names {
				if frag := bad(id.Name); frag != "" {
					t.Errorf("字段 %s 命中 ExecutionPolicy 词汇 %q(锁 7)", id.Name, frag)
				}
			}
		case *ast.ValueSpec:
			for _, id := range decl.Names {
				if frag := bad(id.Name); frag != "" {
					t.Errorf("值/常量 %s 命中 ExecutionPolicy 词汇 %q(锁 7)", id.Name, frag)
				}
			}
		}
		return true
	})
}

// ---- AC4:耗尽 = 显式升级;终局尝试成功 = 接受 ----

// TestRepairPolicyDefaultExhaustionEscalates:默认预算下,初始尝试 + 2 次修复
// 全部可信 FAIL 后,第 3 次修复被拒 → Engine 落显式 EscalationRepairExhausted
// 终态(AC4),谱系完整、取代事实各就位、预算只被询问 [2 3 4]。
func TestRepairPolicyDefaultExhaustionEscalates(t *testing.T) {
	exec := &rpExecutor{steps: []*changeset.ChangeSet{
		rpCandidate(t, "rp-e1"), rpCandidate(t, "rp-e2"), rpCandidate(t, "rp-e3"),
	}} // 恰好 3 份脚本:若循环越界再执行,将显式报错而非静默通过
	verify := &rpVerifier{steps: []VerifyOutcome{
		{Verdict: VerdictFailed, RunRef: "rp-run-1"},
		{Verdict: VerdictFailed, RunRef: "rp-run-2"},
		{Verdict: VerdictFailed, RunRef: "rp-run-3"},
	}}
	lineage := &rpLineage{}
	budget := &rpCountingBudget{policy: DefaultRepairPolicy()}

	res, err := rpNewEngine(t, exec, verify, budget, lineage).Run(context.Background(), "T-P2-07")
	if err != nil {
		t.Fatalf("Engine.Run: %v", err)
	}
	if res.Phase != PhaseEscalated {
		t.Errorf("终态 = %s, 期望 escalated", res.Phase)
	}
	if res.Escalation == nil {
		t.Fatal("耗尽必须携带显式升级事实(AC4)")
	}
	if res.Escalation.Reason != EscalationRepairExhausted {
		t.Errorf("升级原因 = %q, 期望 %q(AC4:显式耗尽原因)", res.Escalation.Reason, EscalationRepairExhausted)
	}
	if res.Basis != nil {
		t.Error("升级终态不得携带 basis")
	}
	if len(res.Attempts) != 3 {
		t.Errorf("已完成尝试 = %d, 期望 3(初始 + 2 次修复)", len(res.Attempts))
	}
	for i, att := range res.Attempts {
		if want := AttemptNo(i + 1); att.No != want {
			t.Errorf("Attempts[%d].No = %d, 期望 %d", i, att.No, want)
		}
	}
	if got := res.Attempts[1].Supersedes; got != res.Attempts[0].ChangeSet {
		t.Errorf("第 1 次修复 Supersedes = %q, 期望 %q", got, res.Attempts[0].ChangeSet)
	}
	if got := res.Attempts[2].Supersedes; got != res.Attempts[1].ChangeSet {
		t.Errorf("第 2 次修复 Supersedes = %q, 期望 %q", got, res.Attempts[1].ChangeSet)
	}
	if len(lineage.facts) != 2 {
		t.Errorf("取代事实 = %d 条, 期望 2(每次修复一条,append-only)", len(lineage.facts))
	}
	if len(budget.asked) != 3 || budget.asked[0] != 2 || budget.asked[1] != 3 || budget.asked[2] != 4 {
		t.Errorf("预算询问 = %v, 期望 [2 3 4]", budget.asked)
	}
}

// TestRepairPolicyFinalAttemptSuccessAccepted:第 2 次(最后一次)修复通过验证
// → 必须接受,预算耗尽检查不得先于最终成功(耗尽语义是「失败后再无修复」,
// 不是「修复次数用满即弃」)。终局 basis 精确绑定第 3 次尝试的候选。
func TestRepairPolicyFinalAttemptSuccessAccepted(t *testing.T) {
	third := rpCandidate(t, "rp-f3")
	exec := &rpExecutor{steps: []*changeset.ChangeSet{
		rpCandidate(t, "rp-f1"), rpCandidate(t, "rp-f2"), third,
	}}
	verify := &rpVerifier{steps: []VerifyOutcome{
		{Verdict: VerdictFailed, RunRef: "rp-run-1"},
		{Verdict: VerdictFailed, RunRef: "rp-run-2"},
		{Verdict: VerdictVerified, RunRef: "rp-run-final"},
	}}
	lineage := &rpLineage{}
	budget := &rpCountingBudget{policy: DefaultRepairPolicy()}

	res, err := rpNewEngine(t, exec, verify, budget, lineage).Run(context.Background(), "T-P2-07")
	if err != nil {
		t.Fatalf("Engine.Run: %v", err)
	}
	if res.Phase != PhaseAccepted {
		t.Fatalf("终态 = %s, 期望 accepted(最后一次修复成功应被接受)", res.Phase)
	}
	if res.Escalation != nil {
		t.Errorf("接受终态不应携带升级载荷: %+v", res.Escalation)
	}
	if res.Basis == nil {
		t.Fatal("Accepted 必须携带 basis")
	}
	if res.Basis.Attempt != 3 {
		t.Errorf("basis.Attempt = %d, 期望 3", res.Basis.Attempt)
	}
	if want := exec.steps; len(want) != 0 {
		t.Errorf("executor 脚本应恰好用尽,剩余 %d", len(want))
	}
	if got := res.Basis.ChangeSet; got != third.ID() {
		t.Errorf("basis.ChangeSet = %q, 期望第 3 次尝试的候选身份", got)
	}
	if len(lineage.facts) != 2 {
		t.Errorf("取代事实 = %d 条, 期望 2", len(lineage.facts))
	}
	if len(budget.asked) != 2 || budget.asked[0] != 2 || budget.asked[1] != 3 {
		t.Errorf("预算询问 = %v, 期望 [2 3](第 4 次尝试不应发生)", budget.asked)
	}
}

// TestRepairPolicyFirstPassNeverConsultsBudget:首过接受不进入修复 —— 预算端口
// 零询问(初始尝试不归 RepairPolicy 治理,ADR-0001 §1)。
func TestRepairPolicyFirstPassNeverConsultsBudget(t *testing.T) {
	exec := &rpExecutor{steps: []*changeset.ChangeSet{rpCandidate(t, "rp-s1")}}
	verify := &rpVerifier{steps: []VerifyOutcome{{Verdict: VerdictVerified, RunRef: "rp-run-first"}}}
	budget := &rpCountingBudget{policy: DefaultRepairPolicy()}

	res, err := rpNewEngine(t, exec, verify, budget, &rpLineage{}).Run(context.Background(), "T-P2-07")
	if err != nil {
		t.Fatalf("Engine.Run: %v", err)
	}
	if res.Phase != PhaseAccepted || res.Basis == nil {
		t.Fatalf("首过应接受并携带 basis,得到 %s", res.Phase)
	}
	if res.Basis.Attempt != 1 {
		t.Errorf("basis.Attempt = %d, 期望 1", res.Basis.Attempt)
	}
	if len(budget.asked) != 0 {
		t.Errorf("首过接受不应咨询修复预算,预算被询问 %v", budget.asked)
	}
}
