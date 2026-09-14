// v1.3 验证权威不变量测试(P1-45,PRD §3.2 稳定 ID)。
//
// 覆盖 P1 激活的验证域不变量:
//   - INV-VER-01 · P1 — required verification authority 必须声明 authoring
//     provenance;
//   - INV-VER-02 · P1 — Implementation authority cannot redefine verification
//     authority(冻结契约只读);
//   - INV-VER-03 · P1 — Protected verification evidence is immutable to the
//     authorized implementation run;
//   - INV-VER-04 · P0/P1 — Spec defects enter review instead of being bypassed;
//   - INV-VER-05 · P1 — Held-out evidence is mechanically unreachable within
//     Tally-controlled and Runtime-Protocol-observable Implementation Authority
//     channels(含 §3.2.1 的可证明边界:第三方 opaque retention 属 declared
//     residual risk,不做无主语断言)。
package arch

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"tally/internal/agentruntime"
	"tally/internal/changeset"
	"tally/internal/core"
	"tally/internal/evidence"
	"tally/internal/holdout"
	"tally/internal/taskstate"
	"tally/internal/verification"
)

// verAt 是固定时钟。
var verAt = time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC)

// verBinding 构造一个 binding,可选字段经 mut 调整。
func verBinding(mut func(*verification.Binding)) verification.Binding {
	b := verification.Binding{
		ID:               "b-1",
		AuthoringMode:    verification.AuthoringPreCommit,
		VisibilityPolicy: verification.VisibilityVisible,
		Evidence: verification.EvidenceRef{
			Type: verification.EvidenceTest,
			Path: "internal/widget/widget_test.go",
		},
	}
	if mut != nil {
		mut(&b)
	}
	return b
}

// verFreeze 把单验收条款的契约冻结为校验入口;失败即测试失败。
func verFreeze(t *testing.T, bindings []verification.Binding, reqs verification.RuntimeRequirements) *verification.Frozen {
	t.Helper()
	contract := verification.Contract{
		TaskID:              "TASK-ARCH",
		Revision:            1,
		Acceptances:         []verification.Acceptance{{ID: "AC-01", Statement: "widget renders", EvidenceBindings: bindings}},
		RuntimeRequirements: reqs,
	}
	frozen, err := contract.Freeze()
	if err != nil {
		t.Fatalf("契约冻结: %v", err)
	}
	return frozen
}

// INV-VER-01:required verification authority 必须声明 authoring provenance
// —— 没有声明 authoring_mode 的 binding 无法冻结;pre_commit 与 post_hoc
// 都是合法声明(post_hoc 的流程约束属 P2,声明本身是 P1 必需)。
func TestV13Ver01BindingRequiresAuthoringProvenance(t *testing.T) {
	missing := []verification.Binding{verBinding(func(b *verification.Binding) { b.AuthoringMode = "" })}
	contract := verification.Contract{
		TaskID:      "TASK-ARCH",
		Revision:    1,
		Acceptances: []verification.Acceptance{{ID: "AC-01", Statement: "widget renders", EvidenceBindings: missing}},
	}
	if _, err := contract.Freeze(); err == nil {
		t.Fatal("缺少 authoring provenance 的 binding 被冻结(INV-VER-01)")
	}
	verFreeze(t, []verification.Binding{verBinding(func(b *verification.Binding) { b.AuthoringMode = verification.AuthoringPreCommit })}, verification.RuntimeRequirements{})
	verFreeze(t, []verification.Binding{verBinding(func(b *verification.Binding) { b.AuthoringMode = verification.AuthoringPostHoc })}, verification.RuntimeRequirements{})
}

// INV-VER-02:冻结后的 VerificationContract 没有任何修改路径 —— 作者侧对象
// 的后续改动不影响冻结内容;Frozen 类型字段全部不导出且无修改入口。实现
// 权威因此没有可用的“重定义验证权威”的类型通道。
func TestV13Ver02FrozenContractImmutable(t *testing.T) {
	contract := verification.Contract{
		TaskID:              "TASK-ARCH",
		Revision:            1,
		Acceptances:         []verification.Acceptance{{ID: "AC-01", Statement: "widget renders", EvidenceBindings: []verification.Binding{verBinding(nil)}}},
		RuntimeRequirements: verification.RuntimeRequirements{},
	}
	frozen, err := contract.Freeze()
	if err != nil {
		t.Fatalf("契约冻结: %v", err)
	}
	hash := frozen.Hash()

	// 作者侧对象原地“重定义”:改任务、改陈述、改 binding 身份。
	contract.TaskID = "TASK-EVIL"
	contract.Acceptances[0].Statement = "evil statement"
	contract.Acceptances[0].EvidenceBindings[0].ID = "b-evil"
	contract.RuntimeRequirements = verification.RuntimeRequirements{ConformanceClass: "evil"}

	if frozen.TaskID() != "TASK-ARCH" {
		t.Fatalf("冻结契约 TaskID 被作者侧改动影响: %s(INV-VER-02)", frozen.TaskID())
	}
	if frozen.Hash() != hash {
		t.Fatal("冻结契约哈希被作者侧改动影响(INV-VER-02)")
	}
	got := frozen.Contract()
	if got.TaskID != "TASK-ARCH" || got.Acceptances[0].Statement != "widget renders" || got.Acceptances[0].EvidenceBindings[0].ID != "b-1" {
		t.Fatalf("冻结内容读回被污染: %+v(INV-VER-02)", got.Acceptances[0])
	}
	// 类型面:Frozen 无导出字段、无修改入口方法。
	fset := token.NewFileSet()
	var frozenStruct *ast.StructType
	mutators := map[string]bool{}
	for _, file := range goFiles(t, filepath.Join(repoInternal, "verification"), false) {
		f, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatalf("解析 %s: %v", file, err)
		}
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if name := receiverTypeName(d); name == "Frozen" && chgMutatorRe.MatchString(d.Name.Name) {
					mutators[d.Name.Name] = true
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok && ts.Name.Name == "Frozen" {
						if st, ok := ts.Type.(*ast.StructType); ok {
							frozenStruct = st
						}
					}
				}
			}
		}
	}
	if frozenStruct == nil {
		t.Fatal("internal/verification 缺少 Frozen 结构体(INV-VER-02 的类型落点)")
	}
	for _, field := range frozenStruct.Fields.List {
		for _, name := range field.Names {
			if ast.IsExported(name.Name) {
				t.Errorf("Frozen 导出字段 %s:包外可原地改写验证权威(INV-VER-02)", name.Name)
			}
		}
	}
	for name := range mutators {
		t.Errorf("Frozen 存在修改入口方法 %s(INV-VER-02:冻结契约只读)", name)
	}
}

// verFactTables 是验证域的持久化事实面:验证 Run 与证据(含裁定)只追加,
// 这是 INV-VER-03 在数据库侧的兜底防线。
var verFactTables = []string{"verification_run", "verification_evidence", "verification_evidence_ruling"}

// INV-VER-03:protected evidence 的 P1 唯一合法保护模式是 immutable,基线
// 摘要必须是 sha256 形态;验证事实面在数据库层由 append-only 触发器兜底,
// sqlc 查询面没有 UPDATE/DELETE 路径 —— 被授权的实现 Run 没有改写受保护
// 证据的任何通道。
func TestV13Ver03ProtectedEvidenceImmutable(t *testing.T) {
	badMode := []verification.Binding{verBinding(func(b *verification.Binding) {
		b.Protection = &verification.Protection{Mode: verification.ProtectionMode("read_only")}
	})}
	contract := verification.Contract{
		TaskID:      "TASK-ARCH",
		Revision:    1,
		Acceptances: []verification.Acceptance{{ID: "AC-01", Statement: "widget renders", EvidenceBindings: badMode}},
	}
	if _, err := contract.Freeze(); err == nil {
		t.Fatal("非 immutable 的保护模式被冻结(INV-VER-03:P1 唯一合法值是 immutable)")
	}

	badHash := []verification.Binding{verBinding(func(b *verification.Binding) {
		b.Protection = &verification.Protection{Mode: verification.ProtectionImmutable, BaselineHash: "md5:deadbeef"}
	})}
	contract.Acceptances[0].EvidenceBindings = badHash
	if _, err := contract.Freeze(); err == nil {
		t.Fatal("非法 baseline_hash 格式被冻结(必须形如 sha256:<64 位十六进制>)")
	}

	goodHash := []verification.Binding{verBinding(func(b *verification.Binding) {
		b.Protection = &verification.Protection{Mode: verification.ProtectionImmutable, BaselineHash: "sha256:" + strings.Repeat("a", 64)}
	})}
	verFreeze(t, goodHash, verification.RuntimeRequirements{})

	content := readAllMigrations(t)
	for _, tbl := range verFactTables {
		re := regexp.MustCompile(`(?is)CREATE\s+TRIGGER\s+\S+\s+BEFORE\s+UPDATE\s+OR\s+DELETE\s+ON\s+` + tbl + `\b`)
		if !re.MatchString(content) {
			t.Errorf("验证事实表 %s 缺少阻止 UPDATE/DELETE 的触发器(INV-VER-03 持久化面)", tbl)
		}
	}
}

// INV-VER-04:规格缺陷走正式通道进入 review/重做,而不是被绕过 ——
// spec_defect 是合法的取代原因(重做 = 新快照 + append-only 取代事实);
// 规格缺陷裁定是 append-only ruling(历史失败证据不被抹掉);review 请求
// 事实强制 NEEDS_REVIEW,任何验证结论都不能绕过它。
func TestV13Ver04SpecDefectEntersReview(t *testing.T) {
	// (1) 规格缺陷重做 = 新快照事实,不是改写旧候选。
	a := chgChangeSet(t, "cand-1", "run-1")
	b := chgChangeSet(t, "cand-2", "run-2")
	fact, err := changeset.NewSupersededFact(a.ID(), b.ID(), changeset.SupersedeSpecDefect, verAt)
	if err != nil {
		t.Fatalf("spec_defect 取代事实被拒绝: %v(INV-VER-04:重做必须可达)", err)
	}
	if fact.Reason != changeset.SupersedeSpecDefect {
		t.Fatalf("取代原因 = %s, want spec_defect", fact.Reason)
	}

	// (2) 规格缺陷裁定 = append-only ruling;裁定通道必须 fact-capable。
	recorder := verAt2Recorder()
	bad, err := evidence.NewFact("fact-spec-fail", authScope(), evidence.VerdictFail, verAt, recorder)
	if err != nil {
		t.Fatalf("NewFact: %v", err)
	}
	ruling, err := evidence.NewRuling(evidence.RulingInvalidSpecDefect, "acceptance clause ambiguous", verAt.Add(time.Minute), recorder)
	if err != nil {
		t.Fatalf("spec_defect 裁定被拒绝: %v(INV-VER-04)", err)
	}
	ruled := bad.WithRuling(ruling)
	if ruled.IsValid() {
		t.Fatal("裁定后事实仍有效(裁定必须移除 gating 效力,INV-VER-04)")
	}
	if len(ruled.Rulings()) != 1 || ruled.Rulings()[0].Kind != evidence.RulingInvalidSpecDefect {
		t.Fatalf("裁定未保留在事实历史上: %+v(历史失败证据不被抹掉)", ruled.Rulings())
	}
	agent := evidence.Recorder{Kind: evidence.RecorderAgent, Actor: "coding-agent"}
	if _, err := evidence.NewRuling(evidence.RulingInvalidSpecDefect, "self-ruling", verAt, agent); !errors.Is(err, evidence.ErrRecorderCannotRecordFact) {
		t.Fatalf("Agent 通道做裁定: err = %v, want ErrRecorderCannotRecordFact(INV-AUTH-01/INV-VER-04)", err)
	}

	// (3) review 请求强制 NEEDS_REVIEW:主线程验证结论不能绕过 review。
	valid := evidence.Recorder{Kind: evidence.RecorderValidator, Actor: "tally"}
	facts := []taskstate.Fact{
		{ID: "f-planned", Kind: taskstate.FactTaskPlanned, At: verAt, Recorder: valid, RevisionRef: "sha256:task-rev-1"},
		{ID: "f-review", Kind: taskstate.FactReviewRequested, At: verAt.Add(time.Second), Recorder: valid},
	}
	res, err := taskstate.Derive(taskstate.Snapshot{Facts: facts})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if res.State != taskstate.StateNeedsReview {
		t.Fatalf("存在 review 请求时状态 = %s, want NEEDS_REVIEW(INV-VER-04:缺陷进入 review,不可绕过)", res.State)
	}
	var traced bool
	for _, ref := range res.Basis {
		if ref == "fact:f-review" {
			traced = true
		}
	}
	if !traced {
		t.Fatalf("NEEDS_REVIEW 结论未追溯到 review 请求事实: %v(INV-STATE-02)", res.Basis)
	}
}

func verAt2Recorder() evidence.Recorder {
	return evidence.Recorder{Kind: evidence.RecorderHuman, Actor: "owner"}
}

// INV-VER-05:held-out 证据在 Tally 控制面与 Runtime 协议可观察面上机械
// 不可达 —— 契约级强制能力声明、holdout 引用拒绝一切路径语义、可见面不得
// 引用 holdout store、暴露通道词表覆盖两类可观察面且未知通道保守按 persistent、
// 实现权威侧包在结构上无法触达 holdout。
func TestV13Ver05HeldOutBoundaryAtObservableChannels(t *testing.T) {
	// (1) 契约含 held_out binding 时必须声明会话隔离、held-out 不可达与
	// conformance class,否则冻结失败。
	heldOut := []verification.Binding{verBinding(func(b *verification.Binding) {
		b.VisibilityPolicy = verification.VisibilityHeldOut
		b.Evidence = verification.EvidenceRef{Type: verification.EvidenceTest, Store: "holdout", Ref: "case-1"}
	})}
	contract := verification.Contract{
		TaskID:      "TASK-ARCH",
		Revision:    1,
		Acceptances: []verification.Acceptance{{ID: "AC-01", Statement: "widget renders", EvidenceBindings: heldOut}},
	}
	if _, err := contract.Freeze(); err == nil {
		t.Fatal("held_out 契约未声明 runtime 能力即被冻结(INV-VER-05)")
	}
	reqs := verification.RuntimeRequirements{SessionIsolation: true, HeldOutUnreachable: true, ConformanceClass: "verification-isolation-v1"}
	verFreeze(t, heldOut, reqs)

	// (2) held_out 内容不得以 repo path 形式暴露(普通 tracked 目录 +
	// forbidden_paths 不构成隐藏机制)。
	asPath := []verification.Binding{verBinding(func(b *verification.Binding) {
		b.VisibilityPolicy = verification.VisibilityHeldOut
		b.Evidence = verification.EvidenceRef{Type: verification.EvidenceTest, Path: "internal/hidden/case_test.go"}
	})}
	contract.Acceptances[0].EvidenceBindings = asPath
	if _, err := contract.Freeze(); err == nil {
		t.Fatal("held_out binding 以 repo path 引用被冻结(INV-VER-05/§8.4.1)")
	}

	// (3) visible binding 不得引用 holdout store(store 对实现侧不可达)。
	visibleHoldout := []verification.Binding{verBinding(func(b *verification.Binding) {
		b.Evidence = verification.EvidenceRef{Type: verification.EvidenceTest, Store: "holdout", Ref: "case-1"}
	})}
	contract.Acceptances[0].EvidenceBindings = visibleHoldout
	if _, err := contract.Freeze(); err == nil {
		t.Fatal("visible binding 引用 holdout store 被冻结(INV-VER-05)")
	}

	// (4) 合法 holdout ref 无法表达任何文件系统路径:实现 workspace 内的
	// 位置在引用层面不可达。
	for _, bad := range []string{"internal/hidden/case-1", `win\path`, "..", "."} {
		if _, err := holdout.ParseRef(bad); err == nil {
			t.Fatalf("ref %q 被接受:holdout 引用不得携带路径语义(INV-VER-05)", bad)
		}
	}
	if _, err := holdout.ParseRef("case-1"); err != nil {
		t.Fatalf("合法不透明 ref 被拒绝: %v", err)
	}

	// (5) 暴露通道词表覆盖 Tally 控制面(knowledge_index/validation_log)与
	// Runtime 协议可观察面(session_leak);未知/空通道保守按 persistent ——
	// 已发生的暴露不能被 relabel 成局部(§3.2.1 / §8.6.1)。
	if evidence.ChannelScopeOf(evidence.ChannelSessionLeak) != evidence.ChannelScopeLineage {
		t.Fatal("session 泄漏未按 lineage 局部分类(§8.6.1 通道语义漂移)")
	}
	for _, ch := range []evidence.ExposureChannel{evidence.ChannelKnowledgeIndex, evidence.ChannelValidationLog, evidence.ExposureChannel("mystery"), evidence.ExposureChannel("")} {
		if evidence.ChannelScopeOf(ch) != evidence.ChannelScopePersistent {
			t.Fatalf("通道 %q 未保守按 persistent 分类:暴露可以被 relabel 成局部(INV-VER-05)", ch)
		}
	}
	exp, err := evidence.NewExposure(evidence.ExposureSpec{
		ID:        "exp-1",
		Binding:   evidence.BindingRef{Project: core.ProjectID("proj-arch"), Verification: "sha256:contract-1", BindingID: "b-held"},
		Channel:   evidence.ExposureChannel("mystery"),
		ExposedAt: verAt,
	})
	if err != nil {
		t.Fatalf("NewExposure(未知通道应被接受并保守分类): %v", err)
	}
	if !exp.BindingWide() {
		t.Fatal("未知通道暴露未破坏整个 binding 的信息独立性(保守默认失效,INV-VER-05)")
	}

	// (6) Runtime-Protocol-observable 通道是协议稳定词表的一部分:session
	// 隔离能力与“已声明记忆”能力必须存在于协商词表中,否则 holdout 审计
	// 无从覆盖 memory/scratchpad 通道(PRD §10.3/§3.2.1)。
	if agentruntime.CapSessionIsolation != agentruntime.Capability("session.isolation") {
		t.Fatalf("CapSessionIsolation = %q, want session.isolation(协议能力字符串即契约)", agentruntime.CapSessionIsolation)
	}
	if agentruntime.CapMemoryDeclared != agentruntime.Capability("memory.declared") {
		t.Fatalf("CapMemoryDeclared = %q, want memory.declared(可观察通道缺失)", agentruntime.CapMemoryDeclared)
	}

	// (7) 结构边界:实现权威侧与期望制品/事实面包不导入 holdout;holdout
	// 只能被 Tally 控制的验证执行面(verification)触达。
	graph := importGraph(t)
	for _, pkg := range []string{"agentruntime", "workspace", "contextpack", "changeset", "taskcontract", "evidence"} {
		if graph[pkg]["holdout"] {
			t.Errorf("internal/%s 导入了 internal/holdout:实现权威侧/期望制品面不得触达 holdout(INV-VER-05)", pkg)
		}
	}
}
