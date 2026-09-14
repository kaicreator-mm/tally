// v1.3 状态与事实不变量测试(P1-45,PRD §3.2 稳定 ID)。
//
// 覆盖 P1 激活的状态/事实域不变量:
//   - INV-FACT-01 · P1 — Machine facts are append-only whenever practical;
//   - INV-STATE-01 · P1 — Derived state must be rebuildable(状态只能推导,
//     没有“直接写入权威状态”的入口);
//   - INV-STATE-02 · P1 — Every factual lifecycle state is evidence-traceable;
//   - INV-STATE-03 · P1(dev) — Desired intent and observed evidence remain
//     separately traceable(P1 只建模 dev 侧;P4 delivery 侧不激活,不写
//     无主语断言)。
//
// P2/P5 未激活不变量(INV-VIEW-01、INV-GOV-01/02)不在本文件(PRD §3.2,
// 任务 AC2)。
package arch

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"regexp"
	"testing"
	"time"

	"tally/internal/evidence"
	"tally/internal/taskstate"
	"tally/internal/verification"
)

// stateAt 起始固定时钟;后续事实逐秒递增,规范排序确定。
var stateAt = time.Date(2026, 9, 6, 11, 0, 0, 0, time.UTC)

func stateRec(kind evidence.RecorderKind, actor string) evidence.Recorder {
	return evidence.Recorder{Kind: kind, Actor: actor}
}

// stateFrozen 冻结当前 desired VerificationContract(INV-STATE-03 的 desired 侧)。
func stateFrozen(t *testing.T) *verification.Frozen {
	t.Helper()
	contract := verification.Contract{
		TaskID:      "TASK-ARCH",
		Revision:    1,
		Acceptances: []verification.Acceptance{{ID: "AC-01", Statement: "widget renders"}},
	}
	frozen, err := contract.Freeze()
	if err != nil {
		t.Fatalf("契约冻结: %v", err)
	}
	return frozen
}

// stateFacts 是一条完整主线事实链:planned → frozen → run → candidate →
// verification run。全部经 fact-capable 通道记录。
func stateFacts(frozen *verification.Frozen) []taskstate.Fact {
	valid := stateRec(evidence.RecorderValidator, "tally")
	return []taskstate.Fact{
		{ID: "f-planned", Kind: taskstate.FactTaskPlanned, At: stateAt, Recorder: valid, RevisionRef: "sha256:task-rev-1"},
		{ID: "f-frozen", Kind: taskstate.FactVerificationFrozen, At: stateAt.Add(time.Second), Recorder: valid, RevisionRef: frozen.Hash()},
		{ID: "f-run", Kind: taskstate.FactCodegenRunStarted, At: stateAt.Add(2 * time.Second), Recorder: valid, RunID: "run-1"},
		{ID: "f-cand", Kind: taskstate.FactChangeSetProduced, At: stateAt.Add(3 * time.Second), Recorder: valid, RunID: "run-1", ChangeSet: "sha256:cs-1"},
		{ID: "f-vrun", Kind: taskstate.FactVerificationRunStarted, At: stateAt.Add(4 * time.Second), Recorder: valid, RunID: "run-1", ChangeSet: "sha256:cs-1"},
	}
}

// stateReports 返回与冻结契约同 revision 的两份覆盖报告(observed 侧)。
func stateReports(frozen *verification.Frozen) (*verification.FunctionalCoverageReport, *verification.IndependenceCoverageReport) {
	functional := &verification.FunctionalCoverageReport{
		TaskID: frozen.TaskID(), ContractRevision: frozen.Revision(), ContractHash: frozen.Hash(),
		Acceptances: []verification.AcceptanceCoverage{{AcceptanceID: "AC-01", Covered: true}},
		Complete:    true,
	}
	independence := &verification.IndependenceCoverageReport{
		TaskID: frozen.TaskID(), ContractRevision: frozen.Revision(), ContractHash: frozen.Hash(),
		Acceptances: []verification.AcceptanceIndependence{{AcceptanceID: "AC-01", Required: verification.IndependenceNone, Covered: true}},
		Complete:    true,
	}
	return functional, independence
}

// stateVerifiedSnapshot 组装可推导 APPROVED 的完整快照(desired 侧契约 +
// observed 侧事实/报告/人类批准,两侧分别可追溯)。
func stateVerifiedSnapshot(t *testing.T) taskstate.Snapshot {
	t.Helper()
	frozen := stateFrozen(t)
	functional, independence := stateReports(frozen)
	return taskstate.Snapshot{
		Contract:     frozen,
		Facts:        stateFacts(frozen),
		Functional:   functional,
		Independence: independence,
		Approvals: []taskstate.ApprovalEvidence{{
			ID: "appr-1", Decision: taskstate.DecisionApprove,
			ChangeSet: "sha256:cs-1", Recorder: stateRec(evidence.RecorderHuman, "owner"), At: stateAt.Add(5 * time.Second),
		}},
	}
}

// INV-STATE-01:状态是纯函数推导 —— 同一事实集的任意输入顺序得到完全相同
// 的结论;推导可整体重建。状态域没有任何“直接写入权威状态”的入口(另见
// TestV13StateHasNoDirectWritePath 的结构面)。
func TestV13StateDerivationIsRebuildable(t *testing.T) {
	snap := stateVerifiedSnapshot(t)
	res1, err := taskstate.Derive(snap)
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if res1.State != taskstate.StateApproved {
		t.Fatalf("完整事实链推导出 %s, want APPROVED(fixture 失真)", res1.State)
	}

	// 事实输入顺序不是语义:逆序重推导必得同一结论(INV-STATE-01)。
	reversed := make([]taskstate.Fact, len(snap.Facts))
	for i, f := range snap.Facts {
		reversed[len(snap.Facts)-1-i] = f
	}
	shuffled := snap
	shuffled.Facts = reversed
	res2, err := taskstate.Derive(shuffled)
	if err != nil {
		t.Fatalf("Derive(乱序): %v", err)
	}
	if !reflect.DeepEqual(res1, res2) {
		t.Fatalf("事实输入顺序影响推导结论(INV-STATE-01):\n%+v\n%+v", res1, res2)
	}

	// 同一快照重算必等:投影可整体重建。
	res3, err := taskstate.Derive(snap)
	if err != nil {
		t.Fatalf("Derive(重算): %v", err)
	}
	if !reflect.DeepEqual(res1, res3) {
		t.Fatal("同快照两次推导结论不一致(状态不可重建,INV-STATE-01)")
	}
}

// INV-STATE-02:每一条结论性状态都追溯得到具体事实/证据 —— 没有创建事实
// 就没有状态;结论携带 Basis;失败证据阻断 VERIFIED 时缺口精确指向证据。
func TestV13StateConclusionTraceableToEvidence(t *testing.T) {
	// 缺少 task_planned:没有创建事实,一切状态推导都无事实基础。
	frozen := stateFrozen(t)
	facts := stateFacts(frozen)
	if _, err := taskstate.Derive(taskstate.Snapshot{Facts: facts[2:]}); !errors.Is(err, taskstate.ErrNoTaskFact) {
		t.Fatalf("无 task_planned 事实: err = %v, want ErrNoTaskFact(INV-STATE-02)", err)
	}

	// APPROVED 结论可追溯到计划事实、批准证据与两份覆盖报告。
	snap := stateVerifiedSnapshot(t)
	approved, err := taskstate.Derive(snap)
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if len(approved.Basis) == 0 {
		t.Fatal("结论没有携带 Basis(状态结论必须可追溯,INV-STATE-02)")
	}
	wantRefs := []string{
		"fact:f-planned", "approval:appr-1",
		"functional_report:" + snap.Contract.Hash(),
		"independence_report:" + snap.Contract.Hash(),
	}
	for _, want := range wantRefs {
		var found bool
		for _, ref := range approved.Basis {
			if ref == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("APPROVED 的 Basis 缺少 %q: %v(INV-STATE-02)", want, approved.Basis)
		}
	}
	if approved.CurrentCandidate != "sha256:cs-1" || approved.CurrentRun != "run-1" {
		t.Fatalf("当前指针不可追溯: run=%q candidate=%q(INV-STATE-02)", approved.CurrentRun, approved.CurrentCandidate)
	}

	// applicable 未裁定 FAIL 阻断 VERIFIED,缺口精确指向该证据;
	// 新 PASS 不抹掉合法 FAIL,这里没有再追加 PASS,阻断必须仍在。
	withFailure := snap
	withFailure.Approvals = nil
	withFailure.Failures = []taskstate.ApplicableFailure{{
		FactID: "f-fail", BindingID: "b-1", ChangeSet: "sha256:cs-1",
		Recorder: stateRec(evidence.RecorderVerificationRun, "ci"), At: stateAt.Add(6 * time.Second),
	}}
	blocked, err := taskstate.Derive(withFailure)
	if err != nil {
		t.Fatalf("Derive(failure): %v", err)
	}
	if blocked.State == taskstate.StateVerified || blocked.State == taskstate.StateApproved {
		t.Fatalf("存在 applicable unresolved FAIL 仍推导出 %s(INV-STATE-02)", blocked.State)
	}
	var gapFound bool
	for _, g := range blocked.Gaps {
		if g.Requirement == taskstate.ReqNoApplicableFailures {
			gapFound = true
			var refFound bool
			for _, ref := range g.Refs {
				if ref == "f-fail" {
					refFound = true
				}
			}
			if !refFound {
				t.Fatalf("失败缺口未指向证据 f-fail: %v(INV-STATE-02)", g.Refs)
			}
		}
	}
	if !gapFound {
		t.Fatalf("缺口缺少 applicable_failures: %+v(INV-STATE-02)", blocked.Gaps)
	}
}

// INV-STATE-03:desired intent 与 observed evidence 分别可追溯 —— desired
// 侧是冻结契约身份;observed 侧报告必须绑定该身份才被消费,绑定其它
// revision 的历史事实对当前状态不可见(validity ≠ applicability)。
func TestV13DesiredAndObservedSeparatelyTraceable(t *testing.T) {
	frozen := stateFrozen(t)
	functional, independence := stateReports(frozen)

	// desired 侧身份单独可追溯:结论携带当前冻结契约内容哈希。
	ok, err := taskstate.Derive(taskstate.Snapshot{
		Contract: frozen, Facts: stateFacts(frozen),
		Functional: functional, Independence: independence,
		Approvals: []taskstate.ApprovalEvidence{{
			ID: "appr-1", Decision: taskstate.DecisionApprove,
			ChangeSet: "sha256:cs-1", Recorder: stateRec(evidence.RecorderHuman, "owner"), At: stateAt.Add(5 * time.Second),
		}},
	})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if ok.ContractHash != frozen.Hash() {
		t.Fatalf("结论未携带当前冻结契约哈希(INV-STATE-03 desired 侧): %q", ok.ContractHash)
	}

	// observed 侧报告绑定陈旧 revision → 对当前状态不可见,VERIFIED 不可达;
	// desired 侧哈希不受影响。
	stale := *functional
	stale.ContractHash = "sha256:stale-contract"
	blocked, err := taskstate.Derive(taskstate.Snapshot{
		Contract: frozen, Facts: stateFacts(frozen),
		Functional: &stale, Independence: independence,
	})
	if err != nil {
		t.Fatalf("Derive(stale): %v", err)
	}
	if blocked.State == taskstate.StateVerified || blocked.State == taskstate.StateApproved {
		t.Fatalf("陈旧 revision 的功能覆盖报告仍放行 %s(Validity ≠ Applicability,INV-STATE-03)", blocked.State)
	}
	if blocked.ContractHash != frozen.Hash() {
		t.Fatalf("陈旧报告污染了 desired 侧身份: %q(INV-STATE-03)", blocked.ContractHash)
	}
	var hasFunctionalGap, hasIndependenceGap bool
	for _, g := range blocked.Gaps {
		switch g.Requirement {
		case taskstate.ReqFunctionalCoverage:
			hasFunctionalGap = true
		case taskstate.ReqIndependenceCoverage:
			hasIndependenceGap = true
		}
	}
	if !hasFunctionalGap || hasIndependenceGap {
		t.Fatalf("缺口投影错误: functional=%v independence=%v(只有绑定期满的报告不可见)", hasFunctionalGap, hasIndependenceGap)
	}
	for _, ref := range blocked.Basis {
		if ref == "functional_report:sha256:stale-contract" {
			t.Fatal("陈旧报告进入了结论 Basis(历史事实不得 gate 当前状态,INV-STATE-03)")
		}
	}
}

// INV-FACT-01(领域面):事实账本只追加 —— 重复记录绝不是更新;裁定只追加;
// 没有任何删除/改写入口(结构面断言在文件尾部)。
func TestV13FactLedgerAppendOnly(t *testing.T) {
	l := evidence.NewLedger()
	rec := stateRec(evidence.RecorderValidator, "tally")
	f1, err := evidence.NewFact("fact-1", authScope(), evidence.VerdictPass, stateAt, rec)
	if err != nil {
		t.Fatalf("NewFact: %v", err)
	}
	if err := l.Record(f1); err != nil {
		t.Fatalf("Record: %v", err)
	}

	// 同 id 再记录是错误,绝不是更新:第一条事实原样保留。
	f1b, err := evidence.NewFact("fact-1", authScope(), evidence.VerdictFail, stateAt, rec)
	if err != nil {
		t.Fatalf("NewFact(重复构造): %v", err)
	}
	if err := l.Record(f1b); !errors.Is(err, evidence.ErrDuplicateFact) {
		t.Fatalf("重复记录: err = %v, want ErrDuplicateFact(INV-FACT-01:事实身份唯一)", err)
	}
	got, ok := l.Fact("fact-1")
	if !ok || got.Verdict() != evidence.VerdictPass || len(l.Facts()) != 1 {
		t.Fatalf("重复记录改写了既有事实: verdict=%s facts=%d(INV-FACT-01)", got.Verdict(), len(l.Facts()))
	}

	// 有效性只经追加裁定移除;事实本体(观察结论)永不改写。
	ruling, err := evidence.NewRuling(evidence.RulingInvalidEvidence, "execution invalid", stateAt.Add(time.Minute), stateRec(evidence.RecorderHuman, "owner"))
	if err != nil {
		t.Fatalf("NewRuling: %v", err)
	}
	updated, err := l.AppendRuling("fact-1", ruling)
	if err != nil {
		t.Fatalf("AppendRuling: %v", err)
	}
	if updated.IsValid() {
		t.Fatal("裁定后事实仍有效(ruling 必须移除 gating 效力)")
	}
	got2, _ := l.Fact("fact-1")
	if got2.Verdict() != evidence.VerdictPass || len(got2.Rulings()) != 1 {
		t.Fatalf("裁定改写了观察结论或丢失裁定: verdict=%s rulings=%d(裁定只追加,INV-FACT-01)", got2.Verdict(), len(got2.Rulings()))
	}
	if len(l.Facts()) != 1 {
		t.Fatalf("裁定后账本行数 = %d, want 1(append-only:裁定不产生新事实行)", len(l.Facts()))
	}
}

// stateFactTables 是状态/度量域的持久化机器事实面(P1-12 契约):只追加。
var stateFactTables = []string{"evidence_exposure", "human_approval", "llm_usage", "human_effort"}

// INV-FACT-01(持久化面):机器事实表在数据库层由 append-only 语句级触发器
// 兜底,sqlc 查询面没有 UPDATE/DELETE 路径。
func TestV13MachineFactTablesAppendOnly(t *testing.T) {
	content := readAllMigrations(t)
	for _, tbl := range stateFactTables {
		re := regexp.MustCompile(`(?is)CREATE\s+TRIGGER\s+\S+\s+BEFORE\s+UPDATE\s+OR\s+DELETE\s+ON\s+` + tbl + `\b`)
		if !re.MatchString(content) {
			t.Errorf("机器事实表 %s 缺少阻止 UPDATE/DELETE 的触发器(INV-FACT-01 持久化面)", tbl)
		}
	}
}

// stateWriterRe 匹配“直接写入状态”形态的导出名。
var stateWriterRe = regexp.MustCompile(`^(Set|Write|Apply|Save|Store|Update|Insert|Mutate|Overwrite|Rewrite)[A-Z]`)

// 状态面没有“直接写入权威状态”的入口(AC1):taskstate 包唯一的状态产出
// 路径是纯函数 Derive,不存在 Set*/Apply*/Write* 形态的导出入口;且该包
// 不得持有持久化/写通道依赖(只依赖 evidence 与 verification)。
func TestV13StateHasNoDirectWritePath(t *testing.T) {
	var deriveFound bool
	fset := token.NewFileSet()
	for _, file := range goFiles(t, filepath.Join(repoInternal, "taskstate"), false) {
		f, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatalf("解析 %s: %v", file, err)
		}
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if fd.Recv == nil {
				if fd.Name.Name == "Derive" {
					deriveFound = true
					continue
				}
				if ast.IsExported(fd.Name.Name) && stateWriterRe.MatchString(fd.Name.Name) {
					t.Errorf("taskstate 导出函数 %s:状态不得有直接写入口(INV-STATE-01)", fd.Name.Name)
				}
				continue
			}
			if ast.IsExported(fd.Name.Name) && stateWriterRe.MatchString(fd.Name.Name) {
				t.Errorf("taskstate 导出方法 %s:状态不得有直接写入口(INV-STATE-01)", fd.Name.Name)
			}
		}
	}
	if !deriveFound {
		t.Fatal("internal/taskstate 缺少导出的 Derive(状态只能由事实推导产出,INV-STATE-01)")
	}

	graph := importGraph(t)
	// v1.4(Decision Lock 12/P1-08):VerifiedBasis 派生投影允许 taskstate 消费
	// 纯域接缝(contract/assurance/engineeringpolicy/risk —— 均无持久化/写通道,
	// 由 test/arch/v14_lifecycle_state_test.go 的 TestV14TaskstateStaysPureDerivation
	// 以同清单双重钉住)。不变量本身不变:状态推导面仍不得持有持久化/写通道依赖。
	allowed := map[string]bool{
		"evidence": true, "verification": true,
		"contract": true, "assurance": true, "engineeringpolicy": true, "risk": true,
		"taskcontract": true, // 仅测试夹具(importGraph 含 _test.go);taskcontract 是纯域类型包
	}
	for imp := range graph["taskstate"] {
		if !allowed[imp] {
			t.Errorf("internal/taskstate 导入了 internal/%s:状态推导面不得持有持久化/写通道依赖(INV-STATE-01)", imp)
		}
	}
}
