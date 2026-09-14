package verification

// 本文件覆盖 P1-29(PRD §14.2/§14.3、INV-VER-05):
//   AC1 实现 workspace 无法读取 held-out 内容(窄能力注入 + 无路径/进程面);
//   AC2 实现 Agent 可见输出不暴露 path/selector/assertion/expected/actual,
//       也不暴露可识别 verdict(§8.4.1/§8.4.3);
//   AC3 FAIL 可记录并阻断 VERIFIED(经 DeriveFunctionalCoverage);
//   AC4 可执行代码任何变更都需要新 ChangeSet/新 verdict(§8.7.1);
//   AC5 不实现 P2 恢复自动化(一次性注入,无 rerun/retry/recovery API)。

import (
	"context"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"tally/internal/changeset"
	"tally/internal/holdout"
)

// ---- 测试夹具 ------------------------------------------------------------

const (
	holdoutTestSecret  = "TALLY-HELDOUT-SECRET-CONTENT-7f3a"
	holdoutTestRef     = "ho-3f2a9c1d5e4b6a7f8d0c1b2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a"
	holdoutTestTaskID  = "TASK-P1-29"
	holdoutTestSeed    = "2026-fixed-seed"
	holdoutTestEnvName = "tally/final-verify:1"
	// holdoutTestSelector 是「失败输出中的 selector 痕迹」标记(泄漏检查用)。
	holdoutTestSelector = "TestSecretSelectorName"
)

// holdoutTestEnv 返回合法环境身份(形态与 execution_context_test 一致)。
func holdoutTestEnv() EnvironmentIdentity {
	return EnvironmentIdentity{
		OS:        "linux",
		Arch:      "amd64",
		Toolchain: "go1.23.4",
		Image:     holdoutTestEnvName,
		Tools:     []ToolVersion{{Name: "go", Version: "1.23.4"}},
		Env:       []string{"GOFLAGS=-count=1"},
	}
}

// holdoutTestContract 返回含一条 held_out binding 的合法契约。
// baselineHash 非空时为 binding 声明 protection.baseline_hash(INV-VER-03)。
func holdoutTestContract(baselineHash string) *Contract {
	binding := Binding{
		ID:               "HOLDOUT-1",
		AuthoringMode:    AuthoringPreCommit,
		VisibilityPolicy: VisibilityHeldOut,
		Evidence:         EvidenceRef{Type: EvidenceTest, Store: "holdout", Ref: holdoutTestRef},
		ExecutionProtocol: &ExecutionProtocol{
			Repetitions: 2,
			Aggregation: AggregationAllPass,
			Seed:        holdoutTestSeed,
			Command:     &CommandSpec{Argv: []string{"tally-holdout-run"}, Dir: "/disposable/verify", Timeout: Duration(time.Minute)},
		},
	}
	if baselineHash != "" {
		binding.Protection = &Protection{Mode: ProtectionImmutable, BaselineHash: baselineHash}
	}
	return &Contract{
		TaskID:   holdoutTestTaskID,
		Revision: 3,
		Acceptances: []Acceptance{{
			ID:               "AC-01",
			Statement:        "held-out final gate holds",
			EvidenceBindings: []Binding{binding},
		}},
		RuntimeRequirements: RuntimeRequirements{
			SessionIsolation:   true,
			HeldOutUnreachable: true,
			ConformanceClass:   "verification-isolation-v1",
		},
	}
}

// frozenHoldoutContract 冻结测试契约;失败即测试失败。
func frozenHoldoutContract(t *testing.T, c *Contract) *Frozen {
	t.Helper()
	f, err := FreezeContract(c, FreezePolicy{})
	if err != nil {
		t.Fatalf("FreezeContract: %v", err)
	}
	return f
}

// frozenHoldoutBindingProtocol 返回夹具契约中 binding 的冻结执行协议。
func frozenHoldoutBindingProtocol(t *testing.T, f *Frozen) *ExecutionProtocol {
	t.Helper()
	p := f.Contract().Acceptances[0].EvidenceBindings[0].ExecutionProtocol
	if p == nil {
		t.Fatal("夹具契约缺少冻结协议")
	}
	return p
}

// holdoutTestChangeSet 构造以 content 为可执行文件内容的 immutable ChangeSet。
func holdoutTestChangeSet(t *testing.T, content string) *changeset.ChangeSet {
	t.Helper()
	cs, err := changeset.New(changeset.Spec{
		Project:      "proj-1",
		BaseRevision: "base-0001",
		ProducerRun:  "run-0001",
		Files:        []changeset.ChangedFile{{Path: "app/main.go", Content: content, Executable: true}},
	})
	if err != nil {
		t.Fatalf("changeset.New: %v", err)
	}
	return cs
}

// holdoutContext 返回与冻结契约 + ChangeSet 精确一致的 execution context。
func holdoutContext(t *testing.T, f *Frozen, cs *changeset.ChangeSet, protocol *ExecutionProtocol) ExecutionContext {
	t.Helper()
	return ExecutionContext{
		ContractRevision: f.Revision(),
		ContractHash:     f.Hash(),
		BindingID:        "HOLDOUT-1",
		CandidateID:      string(cs.ID()),
		Protocol:         *protocol,
		Environment:      holdoutTestEnv(),
	}
}

// fakeHoldoutReader 是测试用的 VerificationReader 窄能力实现。
type fakeHoldoutReader struct {
	content    []byte
	statDigest string // 覆盖 Stat 返回的摘要;空 = 用内容真实摘要
	readErr    error
	statErr    error
}

var _ holdout.VerificationReader = (*fakeHoldoutReader)(nil)

func (r *fakeHoldoutReader) Read(ctx context.Context, ref holdout.Ref) ([]byte, error) {
	if r.readErr != nil {
		return nil, r.readErr
	}
	return r.content, nil
}

func (r *fakeHoldoutReader) Stat(ctx context.Context, ref holdout.Ref) (holdout.Metadata, error) {
	if r.statErr != nil {
		return holdout.Metadata{}, r.statErr
	}
	d := r.statDigest
	if d == "" {
		d = holdout.ContentDigest(r.content)
	}
	return holdout.Metadata{Ref: ref, Size: int64(len(r.content)), Digest: d, CreatedAt: time.Now().UTC()}, nil
}

// recordingBoundary 是测试用的 Tally 控制执行边界:记录请求,按脚本回放观察。
type recordingBoundary struct {
	requests []HoldoutRunRequest
	script   []HoldoutRunOutput
	err      error // 非空时每次调用返回该错误
}

func (b *recordingBoundary) run(ctx context.Context, req HoldoutRunRequest) (HoldoutRunOutput, error) {
	b.requests = append(b.requests, req)
	if b.err != nil {
		return HoldoutRunOutput{}, b.err
	}
	i := len(b.requests) - 1
	if i >= len(b.script) {
		return HoldoutRunOutput{}, fmt.Errorf("boundary: 脚本耗尽(repetition %d)", i+1)
	}
	return b.script[i], nil
}

// boundaryFixture 组装「契约 + ChangeSet + reader + 边界」的完整执行器。
func boundaryFixture(t *testing.T, content, baselineHash string) (*HoldoutExecutor, *Frozen, *changeset.ChangeSet, *recordingBoundary, ExecutionContext) {
	t.Helper()
	f := frozenHoldoutContract(t, holdoutTestContract(baselineHash))
	cs := holdoutTestChangeSet(t, content)
	b := &recordingBoundary{}
	e, err := NewHoldoutExecutor(f, cs, &fakeHoldoutReader{content: []byte(content)}, b.run)
	if err != nil {
		t.Fatalf("NewHoldoutExecutor: %v", err)
	}
	x := holdoutContext(t, f, cs, frozenHoldoutBindingProtocol(t, f))
	return e, f, cs, b, x
}

// holdoutLeakMarkers 返回泄漏标记:实现侧可见输出出现任一串即视为泄漏。
func holdoutLeakMarkers() []string {
	return []string{
		holdoutTestSecret,
		holdoutTestRef,
		holdout.ContentDigest([]byte(holdoutTestSecret)),
		"secret/path_test.go",
		holdoutTestSelector,
		"expected: 42",
		"actual: 43",
		"assertion failed:",
	}
}

func containsAny(s string, needles []string) bool {
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

// ---- AC1:实现 workspace 无法读取 held-out 内容 ---------------------------

// TestHoldoutExecutorContentOnlyViaReaderCapability — 内容只经窄能力读取,只在
// 内存中进入 Tally 控制边界;判定事实与可见投影都不携带内容。
func TestHoldoutExecutorContentOnlyViaReaderCapability(t *testing.T) {
	e, _, _, b, x := boundaryFixture(t, holdoutTestSecret, "")
	b.script = []HoldoutRunOutput{
		{Status: EvidenceStatusPass, OutputDigest: holdout.ContentDigest([]byte("rep1"))},
		{Status: EvidenceStatusPass, OutputDigest: holdout.ContentDigest([]byte("rep2"))},
	}

	rec, pub, err := e.Execute(context.Background(), "HOLDOUT-1", x)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// 内容按 repetition、以内存字节交给 Tally 控制边界。
	if len(b.requests) != 2 {
		t.Fatalf("边界被调用 %d 次,期望 2 次(冻结 repetitions)", len(b.requests))
	}
	for i, req := range b.requests {
		if string(req.Content) != holdoutTestSecret {
			t.Errorf("repetition %d: 边界未按内存字节收到 held-out 内容", i+1)
		}
		if req.BindingID != "HOLDOUT-1" || req.Seed != holdoutTestSeed {
			t.Errorf("repetition %d: 请求标识/seed 不符: %+v", i+1, req)
		}
		if req.Command == nil || len(req.Command.Argv) != 1 || req.Command.Argv[0] != "tally-holdout-run" {
			t.Errorf("repetition %d: 冻结命令未被透传: %+v", i+1, req.Command)
		}
	}

	// 判定事实(仅 Control/Review 通道)不携带内容,只有摘要锚点。
	if rec.ContentDigest != holdout.ContentDigest([]byte(holdoutTestSecret)) {
		t.Errorf("判定事实内容摘要不符: %s", rec.ContentDigest)
	}
	if strings.Contains(fmt.Sprintf("%+v", rec), holdoutTestSecret) {
		t.Errorf("判定事实携带了 held-out 内容")
	}

	// 公开投影(实现通道)既无内容也无任何 held-out 痕迹。
	if containsAny(fmt.Sprintf("%+v", pub), holdoutLeakMarkers()) {
		t.Errorf("实现侧可见投影泄漏 held-out 痕迹: %+v", pub)
	}
}

// TestHoldoutExecutorRefusesVisibleAndUnknownBindings — 本注入点只接受
// held_out binding;visible binding 走增量验证通道,契约外 id 拒绝。
func TestHoldoutExecutorRefusesVisibleAndUnknownBindings(t *testing.T) {
	c := holdoutTestContract("")
	c.Acceptances[0].EvidenceBindings = append(c.Acceptances[0].EvidenceBindings, Binding{
		ID:               "VIS-1",
		AuthoringMode:    AuthoringPreCommit,
		VisibilityPolicy: VisibilityVisible,
		Evidence:         EvidenceRef{Type: EvidenceTest, Path: "app/main_test.go", Selector: "TestVisible"},
	})
	f := frozenHoldoutContract(t, c)
	cs := holdoutTestChangeSet(t, "v1")
	b := &recordingBoundary{}
	e, err := NewHoldoutExecutor(f, cs, &fakeHoldoutReader{content: []byte(holdoutTestSecret)}, b.run)
	if err != nil {
		t.Fatalf("NewHoldoutExecutor: %v", err)
	}

	x := holdoutContext(t, f, cs, frozenHoldoutBindingProtocol(t, f))
	if _, _, err := e.Execute(context.Background(), "VIS-1", x); !errors.Is(err, ErrBindingNotHeldOut) {
		t.Errorf("visible binding 应被拒绝(ErrBindingNotHeldOut),得到 %v", err)
	}
	if _, _, err := e.Execute(context.Background(), "NOT-IN-CONTRACT", x); !errors.Is(err, ErrBindingNotFound) {
		t.Errorf("契约外 binding 应被拒绝(ErrBindingNotFound),得到 %v", err)
	}
	if len(b.requests) != 0 {
		t.Errorf("被拒绝的注入不应到达执行边界")
	}
}

// TestHoldoutExecutorSourceImportsNoPathOrProcessSurface — 机械守卫 AC1:
// holdout_executor.go 不导入任何文件系统/进程能力;held-out 内容在本文件层
// 只能经 reader 窄能力进入,不存在自带的路径寻址或进程执行面。
func TestHoldoutExecutorSourceImportsNoPathOrProcessSurface(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("无法定位测试源文件")
	}
	src := filepath.Join(filepath.Dir(thisFile), "holdout_executor.go")
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, src, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("解析 holdout_executor.go: %v", err)
	}
	allowed := map[string]bool{
		"context":                  true,
		"errors":                   true,
		"fmt":                      true,
		"time":                     true,
		"tally/internal/changeset": true,
		"tally/internal/holdout":   true,
	}
	for _, imp := range f.Imports {
		p := strings.Trim(imp.Path.Value, `"`)
		if !allowed[p] {
			t.Errorf("holdout_executor.go 导入了 %q:held-out 注入点不得自带文件系统/进程能力(AC1)", p)
		}
	}
}

// ---- AC2:实现 Agent 可见输出不暴露细节与可识别 verdict --------------------

// TestRedactedOutcomeStructuralAllowlist — 可见投影的字段集合被锁死在
// §8.4.3 allowlist:结构上不存在能装 path/selector/assertion/expected/actual、
// ref/digest 或 verdict 的字段。
func TestRedactedOutcomeStructuralAllowlist(t *testing.T) {
	typ := reflect.TypeOf(RedactedHoldoutOutcome{})
	allowed := map[string]bool{
		"TaskID": true, "ContractRevision": true, "AcceptanceID": true,
		"EvidenceType": true, "CompletedAt": true,
	}
	for i := 0; i < typ.NumField(); i++ {
		if !allowed[typ.Field(i).Name] {
			t.Errorf("RedactedHoldoutOutcome 出现 allowlist 之外的字段 %q(AC2)", typ.Field(i).Name)
		}
	}
	if typ.NumField() != len(allowed) {
		t.Errorf("RedactedHoldoutOutcome 字段数 %d 与 allowlist %d 不一致", typ.NumField(), len(allowed))
	}
}

// TestRedactedOutcomeHidesVerdictAndDetails — FAIL 的可见投影与 PASS 逐字段
// 相同(可识别 verdict 不可见),且不泄漏任何 held-out 痕迹。
func TestRedactedOutcomeHidesVerdictAndDetails(t *testing.T) {
	at := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	base := HoldoutVerdictRecord{
		TaskID:           holdoutTestTaskID,
		ContractRevision: 3,
		ContractHash:     "sha256:" + strings.Repeat("a", 64),
		AcceptanceID:     "AC-01",
		BindingID:        "HOLDOUT-1",
		CandidateID:      "sha256:" + strings.Repeat("b", 64),
		ContextHash:      "sha256:" + strings.Repeat("c", 64),
		EvidenceType:     EvidenceTest,
		ContentDigest:    holdout.ContentDigest([]byte(holdoutTestSecret)),
		OutputDigests:    []string{holdout.ContentDigest([]byte("raw failure output: " + holdoutTestSecret))},
		CompletedAt:      at,
	}
	fail := base
	fail.Status = EvidenceStatusFail
	fail.Failures = 2
	pass := base
	pass.Status = EvidenceStatusPass

	pubFail, pubPass := RedactHoldoutRecord(fail), RedactHoldoutRecord(pass)
	if !reflect.DeepEqual(pubFail, pubPass) {
		t.Errorf("FAIL 与 PASS 的实现侧可见投影不同,泄露可识别 verdict:%+v vs %+v", pubFail, pubPass)
	}
	if containsAny(fmt.Sprintf("%+v", pubFail), holdoutLeakMarkers()) {
		t.Errorf("实现侧可见投影泄漏 held-out 痕迹: %+v", pubFail)
	}
}

// ---- AC3:FAIL 可记录并阻断 VERIFIED ---------------------------------------

// TestHoldoutFailRecordedAndBlocksVerified — 任一 repetition FAIL 即判定 FAIL;
// 判定事实进入功能覆盖推导后,该 acceptance 保持未覆盖(GapBlockingFailure),
// 报告不完整 → 候选不能 VERIFIED。
func TestHoldoutFailRecordedAndBlocksVerified(t *testing.T) {
	e, f, _, b, x := boundaryFixture(t, holdoutTestSecret, "")
	b.script = []HoldoutRunOutput{
		{Status: EvidenceStatusPass, OutputDigest: holdout.ContentDigest([]byte("rep1"))},
		{Status: EvidenceStatusFail, OutputDigest: holdout.ContentDigest([]byte("raw failure: expected: 42, actual: 43; assertion failed: " + holdoutTestSelector))},
	}

	rec, pub, err := e.Execute(context.Background(), "HOLDOUT-1", x)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if rec.Status != EvidenceStatusFail || rec.Failures != 1 || rec.Repetitions != 2 {
		t.Errorf("判定事实不符: %+v", rec)
	}
	if len(rec.OutputDigests) != 2 {
		t.Errorf("输出摘要应按 repetition 记录,得到 %v", rec.OutputDigests)
	}
	// 可见投影不携带 verdict,也不携带原始失败输出。
	if containsAny(fmt.Sprintf("%+v", pub), holdoutLeakMarkers()) {
		t.Errorf("实现侧可见投影泄漏失败细节: %+v", pub)
	}

	// FAIL 记录阻断 VERIFIED:功能覆盖推导保持未覆盖。
	rep, err := DeriveFunctionalCoverage(f, []EvidenceRecord{rec.EvidenceRecord()})
	if err != nil {
		t.Fatalf("DeriveFunctionalCoverage: %v", err)
	}
	if rep.Complete {
		t.Errorf("存在 valid+applicable FAIL 时覆盖不得完整(不能 VERIFIED)")
	}
	acc := rep.Acceptances[0]
	if acc.Covered || acc.Gap != GapBlockingFailure || len(acc.Blocking) != 1 || acc.Blocking[0] != "HOLDOUT-1" {
		t.Errorf("FAIL 应产生 blocking gap: %+v", acc)
	}

	// 记录是 append-only:Records 稳定保留历史 FAIL。
	if got := e.Records(); len(got) != 1 || got[0].Status != EvidenceStatusFail {
		t.Errorf("判定事实应可记录且不被抹掉: %+v", got)
	}
}

// ---- AC4:可执行代码任何变更都需要新 ChangeSet/新 verdict ------------------

// TestExecutableMutationRequiresNewChangeSetAndVerdict — ChangeSet 身份由内容
// 推导:可执行内容一变即新 ID;旧 execution context(绑定旧 candidate)被拒绝,
// 新 context 的四元组身份必变 → 旧 verdict 不能移植(§8.7.1)。
func TestExecutableMutationRequiresNewChangeSetAndVerdict(t *testing.T) {
	e, f, csA, b, xA := boundaryFixture(t, "v1", "")
	b.script = []HoldoutRunOutput{
		{Status: EvidenceStatusPass, OutputDigest: holdout.ContentDigest([]byte("rep1"))},
		{Status: EvidenceStatusPass, OutputDigest: holdout.ContentDigest([]byte("rep2"))},
	}

	recA, _, err := e.Execute(context.Background(), "HOLDOUT-1", xA)
	if err != nil {
		t.Fatalf("Execute(candidate A): %v", err)
	}
	if recA.CandidateID != string(csA.ID()) || recA.ContextHash == "" {
		t.Errorf("判定事实应绑定 candidate 与 context 身份: %+v", recA)
	}

	// 可执行代码变更:v1 → v2(同路径、可执行)。
	csB := holdoutTestChangeSet(t, "v2")
	if csA.ID() == csB.ID() {
		t.Fatalf("可执行内容变更必须改变 ChangeSet 身份")
	}

	// 旧 context 绑定旧 candidate:对新 ChangeSet 拒绝执行。
	eB, err := NewHoldoutExecutor(f, csB, &fakeHoldoutReader{content: []byte(holdoutTestSecret)}, b.run)
	if err != nil {
		t.Fatalf("NewHoldoutExecutor(candidate B): %v", err)
	}
	if _, _, err := eB.Execute(context.Background(), "HOLDOUT-1", xA); !errors.Is(err, ErrCandidateMismatch) {
		t.Errorf("旧 context 对新 ChangeSet 应被拒绝(ErrCandidateMismatch),得到 %v", err)
	}

	// 为新 ChangeSet 只能构造新 context,其四元组身份必变:旧 verdict 不适用。
	xB := holdoutContext(t, f, csB, frozenHoldoutBindingProtocol(t, f))
	idA, err := xA.Identity()
	if err != nil {
		t.Fatalf("Identity(A): %v", err)
	}
	idB, err := xB.Identity()
	if err != nil {
		t.Fatalf("Identity(B): %v", err)
	}
	if idA == idB || idA != recA.ContextHash {
		t.Errorf("candidate 变更必须改变 execution context 身份:idA=%s idB=%s rec=%s", idA, idB, recA.ContextHash)
	}
}

// ---- AC5:不实现 P2 恢复自动化 ---------------------------------------------

// TestNoRecoveryAutomationSurface — 执行器的导出方法面被锁死:不存在任何
// rerun/retry/recover/probe 形态的 API(决策锁 12)。
func TestNoRecoveryAutomationSurface(t *testing.T) {
	allowed := map[string]bool{
		"Execute": true, "Records": true, "HeldOutBindingIDs": true,
	}
	typ := reflect.TypeOf(&HoldoutExecutor{})
	for i := 0; i < typ.NumMethod(); i++ {
		name := typ.Method(i).Name
		if !allowed[name] {
			t.Errorf("HoldoutExecutor 出现 allowlist 之外的导出方法 %q:P1 不提供恢复/重试自动化(AC5)", name)
		}
	}
	if typ.NumMethod() != len(allowed) {
		t.Errorf("HoldoutExecutor 方法数 %d 与 allowlist %d 不一致", typ.NumMethod(), len(allowed))
	}
}

// TestHoldoutExecutionIsOneShotPerBinding — 注入一次性:binding 无论 PASS、FAIL
// 还是基础设施错误/取消,都已被消耗;本执行器上不存在第二次执行(AC5)。
func TestHoldoutExecutionIsOneShotPerBinding(t *testing.T) {
	// PASS 后再执行:拒绝。
	e, _, _, b, x := boundaryFixture(t, holdoutTestSecret, "")
	b.script = []HoldoutRunOutput{
		{Status: EvidenceStatusPass, OutputDigest: holdout.ContentDigest([]byte("r1"))},
		{Status: EvidenceStatusPass, OutputDigest: holdout.ContentDigest([]byte("r2"))},
	}
	if _, _, err := e.Execute(context.Background(), "HOLDOUT-1", x); err != nil {
		t.Fatalf("首次 Execute: %v", err)
	}
	if _, _, err := e.Execute(context.Background(), "HOLDOUT-1", x); !errors.Is(err, ErrBindingAlreadyExecuted) {
		t.Errorf("PASS 后重复执行应被拒绝(ErrBindingAlreadyExecuted),得到 %v", err)
	}

	// 边界错误后:无 verdict,但注入已消耗,同样拒绝重试。
	e2, _, _, b2, x2 := boundaryFixture(t, holdoutTestSecret, "")
	b2.err = errors.New("runner unavailable")
	if _, _, err := e2.Execute(context.Background(), "HOLDOUT-1", x2); err == nil || errors.Is(err, ErrBindingAlreadyExecuted) {
		t.Fatalf("边界错误应作为执行错误返回,得到 %v", err)
	}
	if len(e2.Records()) != 0 {
		t.Errorf("边界错误不得产生 verdict: %+v", e2.Records())
	}
	if _, _, err := e2.Execute(context.Background(), "HOLDOUT-1", x2); !errors.Is(err, ErrBindingAlreadyExecuted) {
		t.Errorf("失败后重试应被拒绝(P2 恢复自动化不存在),得到 %v", err)
	}

	// 取消后:同样一次性消耗。
	e3, _, _, _, x3 := boundaryFixture(t, holdoutTestSecret, "")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := e3.Execute(ctx, "HOLDOUT-1", x3); !errors.Is(err, context.Canceled) {
		t.Errorf("已取消 context 应返回取消错误,得到 %v", err)
	}
	if _, _, err := e3.Execute(context.Background(), "HOLDOUT-1", x3); !errors.Is(err, ErrBindingAlreadyExecuted) {
		t.Errorf("取消消耗注入后不得再次执行,得到 %v", err)
	}
}

// ---- 构造校验与接线错误(不消耗注入) --------------------------------------

// TestNewHoldoutExecutorValidation — 构造前置校验:依赖缺失与无注入对象都
// 拒绝构造。runtime 能力声明缺失由 Freeze 层强制(含 held_out binding 的契约
// 无法在缺声明下冻结),执行器内的重复检查是不可达的纵深防御。
func TestNewHoldoutExecutorValidation(t *testing.T) {
	f := frozenHoldoutContract(t, holdoutTestContract(""))
	cs := holdoutTestChangeSet(t, "v1")
	reader := &fakeHoldoutReader{content: []byte(holdoutTestSecret)}
	b := &recordingBoundary{}

	if _, err := NewHoldoutExecutor(nil, cs, reader, b.run); !errors.Is(err, ErrNilDependency) {
		t.Errorf("nil 契约应拒绝,得到 %v", err)
	}
	if _, err := NewHoldoutExecutor(f, nil, reader, b.run); !errors.Is(err, ErrNilDependency) {
		t.Errorf("nil ChangeSet 应拒绝,得到 %v", err)
	}
	if _, err := NewHoldoutExecutor(f, cs, nil, b.run); !errors.Is(err, ErrNilDependency) {
		t.Errorf("nil reader 能力应拒绝(AC1),得到 %v", err)
	}
	if _, err := NewHoldoutExecutor(f, cs, reader, nil); !errors.Is(err, ErrNilDependency) {
		t.Errorf("nil 执行边界应拒绝,得到 %v", err)
	}

	// 无 held_out binding 的契约:本执行器没有注入对象,拒绝构造。
	visibleOnly := holdoutTestContract("")
	visibleOnly.Acceptances[0].EvidenceBindings[0].VisibilityPolicy = VisibilityVisible
	visibleOnly.Acceptances[0].EvidenceBindings[0].Evidence = EvidenceRef{Type: EvidenceTest, Path: "app/main_test.go"}
	fv := frozenHoldoutContract(t, visibleOnly)
	if _, err := NewHoldoutExecutor(fv, cs, reader, b.run); !errors.Is(err, ErrNoHeldOutBinding) {
		t.Errorf("无 held_out binding 应拒绝构造(ErrNoHeldOutBinding),得到 %v", err)
	}
}

// TestExecuteRejectsContextMismatch — 接线错误在注入前拒绝:revision/哈希/
// binding 不一致 → ErrExecutionContextMismatch;现编协议(与契约冻结协议不一致)
// → ErrProtocolMismatch;被拒调用不消耗注入。
func TestExecuteRejectsContextMismatch(t *testing.T) {
	e, f, _, b, x := boundaryFixture(t, holdoutTestSecret, "")
	b.script = []HoldoutRunOutput{
		{Status: EvidenceStatusPass, OutputDigest: holdout.ContentDigest([]byte("r1"))},
		{Status: EvidenceStatusPass, OutputDigest: holdout.ContentDigest([]byte("r2"))},
	}

	bad := x
	bad.ContractRevision = f.Revision() + 1
	if _, _, err := e.Execute(context.Background(), "HOLDOUT-1", bad); !errors.Is(err, ErrExecutionContextMismatch) {
		t.Errorf("revision 不一致应拒绝,得到 %v", err)
	}
	bad = x
	bad.ContractHash = "sha256:" + strings.Repeat("9", 64)
	if _, _, err := e.Execute(context.Background(), "HOLDOUT-1", bad); !errors.Is(err, ErrExecutionContextMismatch) {
		t.Errorf("哈希不一致应拒绝,得到 %v", err)
	}
	bad = x
	bad.BindingID = "OTHER"
	if _, _, err := e.Execute(context.Background(), "HOLDOUT-1", bad); !errors.Is(err, ErrExecutionContextMismatch) {
		t.Errorf("binding 不一致应拒绝,得到 %v", err)
	}

	// 现编协议(次数/seed 与契约冻结协议不同)必须被拒绝。
	forged := x
	forged.Protocol.Repetitions = 5
	if _, _, err := e.Execute(context.Background(), "HOLDOUT-1", forged); !errors.Is(err, ErrProtocolMismatch) {
		t.Errorf("现编协议应拒绝(ErrProtocolMismatch),得到 %v", err)
	}
	forged = x
	forged.Protocol.Seed = "another-seed"
	if _, _, err := e.Execute(context.Background(), "HOLDOUT-1", forged); !errors.Is(err, ErrProtocolMismatch) {
		t.Errorf("改 seed 的协议应拒绝,得到 %v", err)
	}

	// 以上全是接线错误:不消耗注入;随后合法执行成功。
	if _, _, err := e.Execute(context.Background(), "HOLDOUT-1", x); err != nil {
		t.Fatalf("接线错误后合法执行应成功: %v", err)
	}
}

// TestHoldoutIntegrityChecks — 内容完整性:Read 内容与 Store 元数据摘要不符,
// 或与冻结 protection.baseline_hash 不符(INV-VER-03/§14.2),拒绝注入;
// 一致时放行。
func TestHoldoutIntegrityChecks(t *testing.T) {
	digest := holdout.ContentDigest([]byte(holdoutTestSecret))
	cs := holdoutTestChangeSet(t, "v1")

	// Store 元数据摘要与内容不符。
	f1 := frozenHoldoutContract(t, holdoutTestContract(""))
	x1 := holdoutContext(t, f1, cs, frozenHoldoutBindingProtocol(t, f1))
	e1, err := NewHoldoutExecutor(f1, cs, &fakeHoldoutReader{content: []byte(holdoutTestSecret), statDigest: "sha256:" + strings.Repeat("0", 64)}, (&recordingBoundary{}).run)
	if err != nil {
		t.Fatalf("NewHoldoutExecutor: %v", err)
	}
	if _, _, err := e1.Execute(context.Background(), "HOLDOUT-1", x1); !errors.Is(err, ErrHoldoutIntegrity) {
		t.Errorf("Store 摘要不符应拒绝(ErrHoldoutIntegrity),得到 %v", err)
	}

	// 冻结 protection.baseline_hash 与内容不符。
	f2 := frozenHoldoutContract(t, holdoutTestContract("sha256:"+strings.Repeat("1", 64)))
	x2 := holdoutContext(t, f2, cs, frozenHoldoutBindingProtocol(t, f2))
	e2, err := NewHoldoutExecutor(f2, cs, &fakeHoldoutReader{content: []byte(holdoutTestSecret)}, (&recordingBoundary{}).run)
	if err != nil {
		t.Fatalf("NewHoldoutExecutor: %v", err)
	}
	if _, _, err := e2.Execute(context.Background(), "HOLDOUT-1", x2); !errors.Is(err, ErrHoldoutIntegrity) {
		t.Errorf("protection 哈希不符应拒绝,得到 %v", err)
	}

	// 一致时放行。
	f3 := frozenHoldoutContract(t, holdoutTestContract(digest))
	x3 := holdoutContext(t, f3, cs, frozenHoldoutBindingProtocol(t, f3))
	b3 := &recordingBoundary{script: []HoldoutRunOutput{
		{Status: EvidenceStatusPass, OutputDigest: digest},
		{Status: EvidenceStatusPass, OutputDigest: digest},
	}}
	e3, err := NewHoldoutExecutor(f3, cs, &fakeHoldoutReader{content: []byte(holdoutTestSecret)}, b3.run)
	if err != nil {
		t.Fatalf("NewHoldoutExecutor: %v", err)
	}
	rec, _, err := e3.Execute(context.Background(), "HOLDOUT-1", x3)
	if err != nil {
		t.Fatalf("protection 哈希一致应通过: %v", err)
	}
	if rec.Status != EvidenceStatusPass {
		t.Errorf("判定应为 PASS: %+v", rec)
	}
}

// TestRecordsReturnsCopy — Records 返回副本:外部改写不影响 append-only 事实。
func TestRecordsReturnsCopy(t *testing.T) {
	e, _, _, b, x := boundaryFixture(t, holdoutTestSecret, "")
	b.script = []HoldoutRunOutput{
		{Status: EvidenceStatusPass, OutputDigest: holdout.ContentDigest([]byte("r1"))},
		{Status: EvidenceStatusPass, OutputDigest: holdout.ContentDigest([]byte("r2"))},
	}
	if _, _, err := e.Execute(context.Background(), "HOLDOUT-1", x); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	got := e.Records()
	got[0].Status = EvidenceStatusFail
	if e.Records()[0].Status != EvidenceStatusPass {
		t.Errorf("Records 必须返回副本,外部改写不得影响事实")
	}
}

// TestHeldOutBindingIDs — 控制面据此核对最终验证必须注入的完整集合。
func TestHeldOutBindingIDs(t *testing.T) {
	c := holdoutTestContract("")
	c.Acceptances[0].EvidenceBindings = append(c.Acceptances[0].EvidenceBindings, Binding{
		ID:               "HOLDOUT-2",
		AuthoringMode:    AuthoringPreCommit,
		VisibilityPolicy: VisibilityHeldOut,
		Evidence:         EvidenceRef{Type: EvidenceValidator, Store: "holdout", Ref: "ho-validator-ref", Validator: "holdout-validator"},
	})
	f := frozenHoldoutContract(t, c)
	cs := holdoutTestChangeSet(t, "v1")
	e, err := NewHoldoutExecutor(f, cs, &fakeHoldoutReader{content: []byte(holdoutTestSecret)}, (&recordingBoundary{}).run)
	if err != nil {
		t.Fatalf("NewHoldoutExecutor: %v", err)
	}
	got := e.HeldOutBindingIDs()
	if len(got) != 2 || got[0] != "HOLDOUT-1" || got[1] != "HOLDOUT-2" {
		t.Errorf("held-out binding 集合不符: %v", got)
	}
}
