package verification

import (
	"errors"
	"strings"
	"testing"
)

// 本文件覆盖 P1-26(PRD §8.7):
//   AC1 停止规则在首次执行前冻结(且此后无修改路径、不可扩大);
//   AC2 随机 seed 属于冻结 protocol;
//   AC3 语义环境变化改变 execution context 身份;
//   AC4 新旧环境的执行不能被静默聚合。

// execTestContractHash — 合法形态的冻结契约哈希占位。
var execTestContractHash = "sha256:" + strings.Repeat("0", 64)

// execEnvA 返回一个合法、可复用的环境身份。
func execEnvA() EnvironmentIdentity {
	return EnvironmentIdentity{
		OS:        "linux",
		Arch:      "amd64",
		Toolchain: "go1.23.4",
		Image:     "tally/verify:1",
		Tools:     []ToolVersion{{Name: "go", Version: "1.23.4"}},
		Env:       []string{"GOFLAGS=-count=1"},
		Annotations: map[string]string{
			"runner": "w1",
		},
	}
}

// execContext 返回一个合法 execution context:2 次 repetition、all_pass、
// 冻结 seed。
func execContext() ExecutionContext {
	return ExecutionContext{
		ContractRevision: 6,
		ContractHash:     execTestContractHash,
		BindingID:        "HOLDOUT-1",
		CandidateID:      "cs-0001",
		Protocol: ExecutionProtocol{
			Repetitions: 2,
			Aggregation: AggregationAllPass,
			Seed:        "2026-fixed-seed",
		},
		Environment: execEnvA(),
	}
}

// recordRun 向 session 记录一条合法 run;失败即测试失败。
func recordRun(t *testing.T, s *ExecutionSession, i int, st EvidenceStatus) {
	t.Helper()
	env := execEnvA()
	if err := s.Record(RunObservation{Index: i, Status: st, Seed: "2026-fixed-seed", Environment: env}); err != nil {
		t.Fatalf("Record(%d,%s): %v", i, st, err)
	}
}

// TestExecutionContextValidate — 构造前置校验:四元组缺一不可(§8.7.1)。
func TestExecutionContextValidate(t *testing.T) {
	if err := execContext().Validate(); err != nil {
		t.Fatalf("base context 应合法: %v", err)
	}
	cases := map[string]func(*ExecutionContext){
		"contract_hash 非哈希形态":    func(x *ExecutionContext) { x.ContractHash = "deadbeef" },
		"contract_revision 为零":   func(x *ExecutionContext) { x.ContractRevision = 0 },
		"binding 为空":             func(x *ExecutionContext) { x.BindingID = " " },
		"candidate 为空":           func(x *ExecutionContext) { x.CandidateID = "" },
		"repetitions 为零":         func(x *ExecutionContext) { x.Protocol.Repetitions = 0 },
		"aggregation 未声明":        func(x *ExecutionContext) { x.Protocol.Aggregation = "" },
		"aggregation 非 all_pass": func(x *ExecutionContext) { x.Protocol.Aggregation = "any_pass" },
		"命令 argv 为空":             func(x *ExecutionContext) { x.Protocol.Command = &CommandSpec{} },
		"环境无语义字段": func(x *ExecutionContext) {
			x.Environment = EnvironmentIdentity{Annotations: map[string]string{"note": "x"}}
		},
		"env 项缺 KEY": func(x *ExecutionContext) { x.Environment.Env = []string{"=novalue"} },
		"env KEY 重复": func(x *ExecutionContext) {
			x.Environment.Env = []string{"A=1", "A=2"}
		},
		"tool name 重复": func(x *ExecutionContext) {
			x.Environment.Tools = []ToolVersion{{Name: "go", Version: "1"}, {Name: "go", Version: "2"}}
		},
	}
	for name, mutate := range cases {
		x := execContext()
		mutate(&x)
		if err := x.Validate(); err == nil {
			t.Fatalf("%s: 期望校验失败,得到 nil", name)
		}
	}
}

// TestEnvironmentIdentityHash — 环境身份哈希与输入顺序无关、与语义字段精确
// 相关、与非语义标注无关(AC3 的基础)。
func TestEnvironmentIdentityHash(t *testing.T) {
	base, err := execEnvA().Hash()
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if !strings.HasPrefix(base, "sha256:") || len(base) != len("sha256:")+64 {
		t.Fatalf("hash 形态 = %q", base)
	}

	// 乱序重建(tools/env/annotations 顺序不同)→ 同一身份。
	shuffled := EnvironmentIdentity{
		Annotations: map[string]string{"runner": "w1"},
		Env:         []string{"GOFLAGS=-count=1"},
		Tools:       []ToolVersion{{Name: "go", Version: "1.23.4"}},
		Image:       "tally/verify:1",
		Toolchain:   "go1.23.4",
		Arch:        "amd64",
		OS:          "linux",
	}
	shuffledHash, err := shuffled.Hash()
	if err != nil {
		t.Fatalf("Hash(shuffled): %v", err)
	}
	if shuffledHash != base {
		t.Fatalf("输入顺序影响了身份哈希: %s != %s", shuffledHash, base)
	}

	// 仅改非语义标注 → 身份不变。
	annotated := execEnvA()
	annotated.Annotations = map[string]string{"runner": "w2", "note": "rebuilt"}
	annotatedHash, err := annotated.Hash()
	if err != nil {
		t.Fatalf("Hash(annotated): %v", err)
	}
	if annotatedHash != base {
		t.Fatalf("非语义标注改变了身份哈希: %s != %s", annotatedHash, base)
	}

	// 完全没有语义字段 → 拒绝(没有可绑定的环境身份)。
	empty := EnvironmentIdentity{}
	if err := empty.Validate(); err == nil {
		t.Fatalf("空语义字段环境应校验失败")
	}
	if _, err := empty.Hash(); err == nil {
		t.Fatalf("空语义字段环境应无法计算身份")
	}
}

// TestSemanticEnvironmentChangeChangesIdentity — AC3:任一语义字段(或四元组
// 其他成分)变化都改变 execution context 身份;仅非语义标注变化不改。
func TestSemanticEnvironmentChangeChangesIdentity(t *testing.T) {
	base, err := execContext().Identity()
	if err != nil {
		t.Fatalf("Identity: %v", err)
	}

	semantic := map[string]func(*ExecutionContext){
		"os 变化":        func(x *ExecutionContext) { x.Environment.OS = "windows" },
		"arch 变化":      func(x *ExecutionContext) { x.Environment.Arch = "arm64" },
		"toolchain 变化": func(x *ExecutionContext) { x.Environment.Toolchain = "go1.24.1" },
		"image 变化":     func(x *ExecutionContext) { x.Environment.Image = "tally/verify:2" },
		"tool 版本升级":    func(x *ExecutionContext) { x.Environment.Tools = []ToolVersion{{Name: "go", Version: "1.23.5"}} },
		"新增 tool": func(x *ExecutionContext) {
			x.Environment.Tools = append(x.Environment.Tools, ToolVersion{Name: "psql", Version: "16"})
		},
		"env 变量变化":             func(x *ExecutionContext) { x.Environment.Env = []string{"GOFLAGS=-count=2"} },
		"新增 env 变量":            func(x *ExecutionContext) { x.Environment.Env = append(x.Environment.Env, "CGO_ENABLED=0") },
		"seed 变化":              func(x *ExecutionContext) { x.Protocol.Seed = "other-seed" },
		"repetitions 变化":       func(x *ExecutionContext) { x.Protocol.Repetitions = 3 },
		"命令变化":                 func(x *ExecutionContext) { x.Protocol.Command = &CommandSpec{Argv: []string{"go", "test", "./..."}} },
		"candidate 变化":         func(x *ExecutionContext) { x.CandidateID = "cs-0002" },
		"binding 变化":           func(x *ExecutionContext) { x.BindingID = "HOLDOUT-2" },
		"contract hash 变化":     func(x *ExecutionContext) { x.ContractHash = "sha256:" + strings.Repeat("1", 64) },
		"contract revision 变化": func(x *ExecutionContext) { x.ContractRevision = 7 },
	}
	for name, mutate := range semantic {
		x := execContext()
		mutate(&x)
		id, err := x.Identity()
		if err != nil {
			t.Fatalf("%s: Identity: %v", name, err)
		}
		if id == base {
			t.Fatalf("%s: 语义变化未改变 execution context 身份", name)
		}
	}

	// 仅非语义标注变化 → 身份不变(不是环境变化,仍是同一次 protocol)。
	x := execContext()
	x.Environment.Annotations = map[string]string{"runner": "w9"}
	id, err := x.Identity()
	if err != nil {
		t.Fatalf("Identity(annotations): %v", err)
	}
	if id != base {
		t.Fatalf("非语义标注改变了 execution context 身份")
	}

	// 身份哈希可重复计算且稳定。
	again, err := execContext().Identity()
	if err != nil || again != base {
		t.Fatalf("身份哈希不稳定: %s != %s (err=%v)", again, base, err)
	}
}

// TestSeedBelongsToFrozenProtocol — AC2:seed 是冻结 protocol 的一部分;执行
// 观察携带不同 seed 时被拒绝且不进入聚合;冻结 seed 为空(声明无随机性)时,
// 携带 seed 的执行同样被拒绝。
func TestSeedBelongsToFrozenProtocol(t *testing.T) {
	withSeed, err := protocolHash(&ExecutionProtocol{Repetitions: 1, Aggregation: AggregationAllPass, Seed: "s1"})
	if err != nil {
		t.Fatalf("protocolHash: %v", err)
	}
	otherSeed, err := protocolHash(&ExecutionProtocol{Repetitions: 1, Aggregation: AggregationAllPass, Seed: "s2"})
	if err != nil {
		t.Fatalf("protocolHash: %v", err)
	}
	noSeed, err := protocolHash(&ExecutionProtocol{Repetitions: 1, Aggregation: AggregationAllPass})
	if err != nil {
		t.Fatalf("protocolHash: %v", err)
	}
	if withSeed == otherSeed || withSeed == noSeed || otherSeed == noSeed {
		t.Fatalf("seed 变化未改变冻结 protocol 身份")
	}

	run := func(seed string) RunObservation {
		return RunObservation{Index: 1, Status: EvidenceStatusPass, Seed: seed, Environment: execEnvA()}
	}

	s, err := NewExecutionSession(execContext())
	if err != nil {
		t.Fatalf("NewExecutionSession: %v", err)
	}
	if err := s.Record(run("different-seed")); !errors.Is(err, ErrSeedMismatch) {
		t.Fatalf("期望 ErrSeedMismatch,得到 %v", err)
	}
	if len(s.RecordedRuns()) != 0 {
		t.Fatalf("seed 不一致的执行不应被记录")
	}
	if err := s.Record(run("2026-fixed-seed")); err != nil {
		t.Fatalf("冻结 seed 的执行被拒绝: %v", err)
	}

	// 冻结 seed 为空 = protocol 声明无随机性。
	ctx := execContext()
	ctx.Protocol.Repetitions = 1
	ctx.Protocol.Seed = ""
	s2, err := NewExecutionSession(ctx)
	if err != nil {
		t.Fatalf("NewExecutionSession: %v", err)
	}
	if err := s2.Record(run("surprise-seed")); !errors.Is(err, ErrSeedMismatch) {
		t.Fatalf("期望 ErrSeedMismatch(冻结空 seed),得到 %v", err)
	}
	if err := s2.Record(run("")); err != nil {
		t.Fatalf("与冻结空 seed 一致的执行被拒绝: %v", err)
	}
}

// TestStoppingRuleFrozenBeforeFirstRun — AC1:停止规则随 session 构造冻结,
// 构造后调用方改动不生效;repetition 记录 append-only,超次数与同序号重跑均被
// 拒绝,verdict 不可能被 rerun-until-pass 改写。
func TestStoppingRuleFrozenBeforeFirstRun(t *testing.T) {
	ctx := execContext()
	ctx.Protocol.Repetitions = 1
	s, err := NewExecutionSession(ctx)
	if err != nil {
		t.Fatalf("NewExecutionSession: %v", err)
	}

	// 首次执行前冻结;构造后调用方试图扩大规则/换 seed/换环境全部无效。
	ctx.Protocol.Repetitions = 99
	ctx.Protocol.Seed = "evil-seed"
	ctx.Environment.Toolchain = "go1.99"
	rule := s.StoppingRule()
	if rule.Repetitions != 1 || rule.Aggregation != AggregationAllPass || rule.Seed != "2026-fixed-seed" {
		t.Fatalf("冻结停止规则被改动: %+v", rule)
	}

	recordRun(t, s, 1, EvidenceStatusPass)
	if !s.Complete() {
		t.Fatalf("1 次 repetition 的停止规则应已满足")
	}
	// 若调用方改动生效,这里应报 incomplete;冻结规则下应能产出 verdict。
	v, err := s.Verdict()
	if err != nil {
		t.Fatalf("Verdict: %v", err)
	}
	if v.Status != EvidenceStatusPass || v.Repetitions != 1 {
		t.Fatalf("verdict = %+v", v)
	}

	// 同序号重跑(pass 覆盖 fail)被拒绝;扩序号 rerun 也被拒绝。
	ctx2 := execContext()
	ctx2.Protocol.Repetitions = 1
	s2, err := NewExecutionSession(ctx2)
	if err != nil {
		t.Fatalf("NewExecutionSession: %v", err)
	}
	recordRun(t, s2, 1, EvidenceStatusFail)
	if err := s2.Record(RunObservation{Index: 1, Status: EvidenceStatusPass, Seed: "2026-fixed-seed", Environment: execEnvA()}); !errors.Is(err, ErrRunIndexAlreadyRecorded) {
		t.Fatalf("期望 ErrRunIndexAlreadyRecorded,得到 %v", err)
	}
	if err := s2.Record(RunObservation{Index: 2, Status: EvidenceStatusPass, Seed: "2026-fixed-seed", Environment: execEnvA()}); !errors.Is(err, ErrRunBeyondStoppingRule) {
		t.Fatalf("期望 ErrRunBeyondStoppingRule,得到 %v", err)
	}
	v2, err := s2.Verdict()
	if err != nil {
		t.Fatalf("Verdict: %v", err)
	}
	if v2.Status != EvidenceStatusFail || v2.Failures != 1 {
		t.Fatalf("FAIL 后 rerun 被拒,verdict 必须仍是 FAIL: %+v", v2)
	}
}

// TestAllPassAggregationRejectsCherryPick — §8.7.2:任一 repetition FAIL 即
// verdict FAIL;乱序记录不影响聚合;RunStatuses 按 repetition 序号排列。
func TestAllPassAggregationRejectsCherryPick(t *testing.T) {
	s, err := NewExecutionSession(execContext())
	if err != nil {
		t.Fatalf("NewExecutionSession: %v", err)
	}
	recordRun(t, s, 2, EvidenceStatusFail) // 乱序记录
	recordRun(t, s, 1, EvidenceStatusPass)

	// 越出冻结的 2 次 repetition 的第三次执行(cherry-pick 用)被拒绝。
	if err := s.Record(RunObservation{Index: 3, Status: EvidenceStatusPass, Seed: "2026-fixed-seed", Environment: execEnvA()}); !errors.Is(err, ErrRunBeyondStoppingRule) {
		t.Fatalf("期望 ErrRunBeyondStoppingRule,得到 %v", err)
	}

	v, err := s.Verdict()
	if err != nil {
		t.Fatalf("Verdict: %v", err)
	}
	if v.Status != EvidenceStatusFail || v.Failures != 1 || v.Repetitions != 2 {
		t.Fatalf("2 pass + 1 fail 被拒后,verdict 必须是 FAIL: %+v", v)
	}
	if len(v.RunStatuses) != 2 || v.RunStatuses[0] != EvidenceStatusPass || v.RunStatuses[1] != EvidenceStatusFail {
		t.Fatalf("RunStatuses 应按 repetition 序号排列: %v", v.RunStatuses)
	}
	if v.ContextHash != s.ContextHash() {
		t.Fatalf("verdict ContextHash %s != session %s", v.ContextHash, s.ContextHash())
	}
	if v.Aggregation != AggregationAllPass || v.BindingID != "HOLDOUT-1" || v.CandidateID != "cs-0001" {
		t.Fatalf("verdict 四元组字段缺失或不一致: %+v", v)
	}
}

// TestVerdictRequiresCompleteStoppingRule — 部分执行没有 verdict;补齐全部
// repetition 后才能聚合。
func TestVerdictRequiresCompleteStoppingRule(t *testing.T) {
	s, err := NewExecutionSession(execContext())
	if err != nil {
		t.Fatalf("NewExecutionSession: %v", err)
	}
	recordRun(t, s, 1, EvidenceStatusPass)
	if s.Complete() {
		t.Fatalf("还有未记录 repetition,不应报告 Complete")
	}
	if _, err := s.Verdict(); !errors.Is(err, ErrStoppingRuleIncomplete) {
		t.Fatalf("期望 ErrStoppingRuleIncomplete,得到 %v", err)
	}
	recordRun(t, s, 2, EvidenceStatusPass)
	if !s.Complete() {
		t.Fatalf("全部 repetition 已记录,应报告 Complete")
	}
	v, err := s.Verdict()
	if err != nil {
		t.Fatalf("Verdict: %v", err)
	}
	if v.Status != EvidenceStatusPass || v.Failures != 0 || len(v.RunStatuses) != 2 {
		t.Fatalf("verdict = %+v", v)
	}

	// 缺口执行(记录 1、3,缺 2)同样没有 verdict。
	ctx := execContext()
	ctx.Protocol.Repetitions = 3
	s2, err := NewExecutionSession(ctx)
	if err != nil {
		t.Fatalf("NewExecutionSession: %v", err)
	}
	recordRun(t, s2, 1, EvidenceStatusPass)
	recordRun(t, s2, 3, EvidenceStatusPass)
	if _, err := s2.Verdict(); !errors.Is(err, ErrStoppingRuleIncomplete) {
		t.Fatalf("缺口 repetition 应导致无 verdict,得到 %v", err)
	}
}

// TestForeignEnvironmentRunNotAggregated — AC4:语义环境不同的执行被显式拒绝、
// 不进入聚合,verdict 只可能来自冻结环境身份下的执行;仅非语义标注不同的执行
// 属于同一环境,允许并入。
func TestForeignEnvironmentRunNotAggregated(t *testing.T) {
	s, err := NewExecutionSession(execContext())
	if err != nil {
		t.Fatalf("NewExecutionSession: %v", err)
	}

	// 新 toolchain(语义变化)→ 拒绝且不留记录。
	foreign := execEnvA()
	foreign.Toolchain = "go1.24.1"
	err = s.Record(RunObservation{Index: 1, Status: EvidenceStatusPass, Seed: "2026-fixed-seed", Environment: foreign})
	if !errors.Is(err, ErrEnvironmentMismatch) {
		t.Fatalf("期望 ErrEnvironmentMismatch,得到 %v", err)
	}
	if len(s.RecordedRuns()) != 0 {
		t.Fatalf("外来环境的执行不能进入聚合")
	}

	// 新 image(语义变化)→ 同样拒绝。
	foreign2 := execEnvA()
	foreign2.Image = "tally/verify:2"
	err = s.Record(RunObservation{Index: 2, Status: EvidenceStatusPass, Seed: "2026-fixed-seed", Environment: foreign2})
	if !errors.Is(err, ErrEnvironmentMismatch) {
		t.Fatalf("期望 ErrEnvironmentMismatch,得到 %v", err)
	}

	// 仅非语义标注不同 → 同一环境身份,允许。
	annotated := execEnvA()
	annotated.Annotations = map[string]string{"runner": "w7"}
	if err := s.Record(RunObservation{Index: 1, Status: EvidenceStatusPass, Seed: "2026-fixed-seed", Environment: annotated}); err != nil {
		t.Fatalf("非语义标注差异被误判为环境变化: %v", err)
	}
	recordRun(t, s, 2, EvidenceStatusPass)

	v, err := s.Verdict()
	if err != nil {
		t.Fatalf("Verdict: %v", err)
	}
	envHash, err := execEnvA().Hash()
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if v.EnvironmentHash != envHash {
		t.Fatalf("verdict 环境身份 %s != 冻结环境身份 %s", v.EnvironmentHash, envHash)
	}
	if v.Status != EvidenceStatusPass || len(v.RunStatuses) != 2 {
		t.Fatalf("被拒绝的外来执行影响了 verdict: %+v", v)
	}
	if len(s.RecordedRuns()) != 2 {
		t.Fatalf("已记录 run 数应为 2,得到 %d", len(s.RecordedRuns()))
	}
}
