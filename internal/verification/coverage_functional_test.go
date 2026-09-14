package verification

import (
	"strings"
	"testing"
)

// frozenStdContract 返回 P1 standard path 冻结的 §8.11 示例契约:
// revision 6,AC-01→EV-AC01-01、AC-02→EV-AC02-H1、AC-03→EV-AC03-01。
func frozenStdContract(t *testing.T) *Frozen {
	t.Helper()
	frozen, err := FreezeContract(mustParse(t, fullContractYAML), FreezePolicy{})
	if err != nil {
		t.Fatalf("FreezeContract: %v", err)
	}
	return frozen
}

// passRec / failRec 构造指定 binding 在指定契约 revision 上的观察记录。
func passRec(binding string, revision int) EvidenceRecord {
	return EvidenceRecord{BindingID: binding, Revision: revision, Status: EvidenceStatusPass}
}

func failRec(binding string, revision int) EvidenceRecord {
	return EvidenceRecord{BindingID: binding, Revision: revision, Status: EvidenceStatusFail}
}

// 全部 acceptance 各有一条有效 applicable 的 PASS → 全覆盖;报告可追溯到契约
// 身份与满足 binding(INV-STATE-02);同 binding 的重复 PASS 不改变覆盖
// (覆盖按 acceptance 判定,不按 evidence 数量,AC1)。
func TestDeriveFunctionalCoverageAllCovered(t *testing.T) {
	frozen := frozenStdContract(t)

	rep, err := DeriveFunctionalCoverage(frozen, []EvidenceRecord{
		passRec("EV-AC01-01", 6),
		passRec("EV-AC02-H1", 6),
		passRec("EV-AC03-01", 6),
	})
	if err != nil {
		t.Fatalf("DeriveFunctionalCoverage: %v", err)
	}

	if rep.TaskID != "TASK-023" || rep.ContractRevision != 6 || rep.ContractHash != frozen.Hash() {
		t.Fatalf("report identity = %s/%d/%s, want TASK-023/6/%s", rep.TaskID, rep.ContractRevision, rep.ContractHash, frozen.Hash())
	}
	if !rep.Complete {
		t.Fatalf("expected Complete=true")
	}
	if len(rep.Acceptances) != 3 {
		t.Fatalf("acceptances = %d, want 3", len(rep.Acceptances))
	}
	wantBy := map[string]string{
		"AC-01": "EV-AC01-01",
		"AC-02": "EV-AC02-H1",
		"AC-03": "EV-AC03-01",
	}
	for i, ac := range rep.Acceptances {
		if !ac.Covered || ac.Gap != GapNone {
			t.Fatalf("acceptances[%d] %s covered=%v gap=%q, want covered/GapNone", i, ac.AcceptanceID, ac.Covered, ac.Gap)
		}
		if len(ac.SatisfiedBy) != 1 || ac.SatisfiedBy[0] != wantBy[ac.AcceptanceID] {
			t.Fatalf("acceptance %s SatisfiedBy = %v, want [%s]", ac.AcceptanceID, ac.SatisfiedBy, wantBy[ac.AcceptanceID])
		}
		if len(ac.Blocking) != 0 {
			t.Fatalf("acceptance %s Blocking = %v, want empty", ac.AcceptanceID, ac.Blocking)
		}
	}

	// 追加同一条 binding 的第二条 PASS(append-only 事实):覆盖结论与
	// SatisfiedBy 不变 —— 不按 evidence 数量计数(AC1)。
	rep2, err := DeriveFunctionalCoverage(frozen, []EvidenceRecord{
		passRec("EV-AC01-01", 6),
		passRec("EV-AC01-01", 6),
		passRec("EV-AC02-H1", 6),
		passRec("EV-AC03-01", 6),
	})
	if err != nil {
		t.Fatalf("DeriveFunctionalCoverage(duplicate): %v", err)
	}
	if !rep2.Complete || len(rep2.Acceptances[0].SatisfiedBy) != 1 {
		t.Fatalf("duplicate pass changed coverage: complete=%v satisfiedBy=%v", rep2.Complete, rep2.Acceptances[0].SatisfiedBy)
	}
}

// AC1:覆盖按 Acceptance Clause 判定。只有 AC-01 有有效 PASS 时,其余
// acceptance 不因任何「证据数量」而被覆盖(存在性,不是计数)。
func TestDeriveFunctionalCoveragePerAcceptanceNotByCount(t *testing.T) {
	frozen := frozenStdContract(t)

	// AC-01 的 binding 被多条有效 PASS 覆盖,其他 acceptance 一条记录都没有。
	rep, err := DeriveFunctionalCoverage(frozen, []EvidenceRecord{
		passRec("EV-AC01-01", 6),
		passRec("EV-AC01-01", 6),
		passRec("EV-AC01-01", 6),
	})
	if err != nil {
		t.Fatalf("DeriveFunctionalCoverage: %v", err)
	}
	if rep.Complete {
		t.Fatalf("expected Complete=false")
	}
	ac01, ac02, ac03 := rep.Acceptances[0], rep.Acceptances[1], rep.Acceptances[2]
	if ac01.AcceptanceID != "AC-01" || ac02.AcceptanceID != "AC-02" || ac03.AcceptanceID != "AC-03" {
		t.Fatalf("acceptance order changed: %s/%s/%s", ac01.AcceptanceID, ac02.AcceptanceID, ac03.AcceptanceID)
	}
	if !ac01.Covered || len(ac01.SatisfiedBy) != 1 {
		t.Fatalf("AC-01 covered=%v satisfiedBy=%v, want covered by one binding", ac01.Covered, ac01.SatisfiedBy)
	}
	for _, ac := range []AcceptanceCoverage{ac02, ac03} {
		if ac.Covered || ac.Gap != GapNoApplicablePass || len(ac.SatisfiedBy) != 0 {
			t.Fatalf("acceptance %s covered=%v gap=%q, want uncovered/%s", ac.AcceptanceID, ac.Covered, ac.Gap, GapNoApplicablePass)
		}
	}
}

// AC2:invalid 或 inapplicable 的 evidence 不能满足当前 revision(§8.8.2)。
func TestDeriveFunctionalCoverageInvalidOrInapplicablePassCannotSatisfy(t *testing.T) {
	cases := map[string][]EvidenceRecord{
		"invalidated pass": {func() EvidenceRecord {
			r := passRec("EV-AC01-01", 6)
			r.Validity = EvidenceValidityInvalidated
			return r
		}()},
		"pass bound to older revision": {passRec("EV-AC01-01", 5)},
	}
	for name, records := range cases {
		t.Run(name, func(t *testing.T) {
			frozen := frozenStdContract(t)
			rep, err := DeriveFunctionalCoverage(frozen, records)
			if err != nil {
				t.Fatalf("DeriveFunctionalCoverage: %v", err)
			}
			ac01 := rep.Acceptances[0]
			if ac01.Covered || ac01.Gap != GapNoApplicablePass || len(ac01.SatisfiedBy) != 0 {
				t.Fatalf("AC-01 covered=%v gap=%q satisfiedBy=%v, want uncovered/%s with no satisfying evidence",
					ac01.Covered, ac01.Gap, ac01.SatisfiedBy, GapNoApplicablePass)
			}
			if rep.Complete {
				t.Fatalf("expected Complete=false")
			}
		})
	}

	// invalid PASS 不能满足,但同 binding 的有效 PASS 可以(§8.8.1:invalid
	// 记录只是自身失效,不否定其他合法事实)。
	frozen := frozenStdContract(t)
	invalidated := passRec("EV-AC01-01", 6)
	invalidated.Validity = EvidenceValidityInvalidated
	rep, err := DeriveFunctionalCoverage(frozen, []EvidenceRecord{invalidated, passRec("EV-AC01-01", 6)})
	if err != nil {
		t.Fatalf("DeriveFunctionalCoverage: %v", err)
	}
	if !rep.Acceptances[0].Covered || len(rep.Acceptances[0].SatisfiedBy) != 1 {
		t.Fatalf("AC-01 covered=%v satisfiedBy=%v, want covered by the valid pass", rep.Acceptances[0].Covered, rep.Acceptances[0].SatisfiedBy)
	}
}

// AC3:一条 valid + applicable 的 FAIL 使该 acceptance 保持未覆盖(候选非
// VERIFIED),即使同 acceptance 的其他 binding 已 PASS;只有正式 invalidated
// 或不再 applicable 的 FAIL 才解除阻断(§8.8.1/§8.8.2)。
func TestDeriveFunctionalCoverageBlockingFailure(t *testing.T) {
	frozen := frozenStdContract(t)

	// AC-01 的 binding FAIL,AC-02/AC-03 PASS:FAIL 不被其他 PASS 抹掉。
	rep, err := DeriveFunctionalCoverage(frozen, []EvidenceRecord{
		failRec("EV-AC01-01", 6),
		passRec("EV-AC02-H1", 6),
		passRec("EV-AC03-01", 6),
	})
	if err != nil {
		t.Fatalf("DeriveFunctionalCoverage: %v", err)
	}
	ac01 := rep.Acceptances[0]
	if ac01.Covered || ac01.Gap != GapBlockingFailure {
		t.Fatalf("AC-01 covered=%v gap=%q, want uncovered/%s", ac01.Covered, ac01.Gap, GapBlockingFailure)
	}
	if len(ac01.Blocking) != 1 || ac01.Blocking[0] != "EV-AC01-01" {
		t.Fatalf("AC-01 Blocking = %v, want [EV-AC01-01]", ac01.Blocking)
	}
	if rep.Complete {
		t.Fatalf("valid blocking FAIL must keep candidate non-VERIFIED (Complete=false)")
	}

	// 同一 binding FAIL 后重跑 PASS(append-only 事实):保守合并,FAIL 仍阻断,
	// 禁止 cherry-pick(§8.8.1、§8.7.2)。
	rep, err = DeriveFunctionalCoverage(frozen, []EvidenceRecord{
		failRec("EV-AC01-01", 6),
		passRec("EV-AC01-01", 6),
		passRec("EV-AC02-H1", 6),
		passRec("EV-AC03-01", 6),
	})
	if err != nil {
		t.Fatalf("DeriveFunctionalCoverage(rerun): %v", err)
	}
	ac01 = rep.Acceptances[0]
	if ac01.Covered || ac01.Gap != GapBlockingFailure || len(ac01.Blocking) != 1 {
		t.Fatalf("FAIL then PASS: covered=%v gap=%q blocking=%v, want still blocked by EV-AC01-01", ac01.Covered, ac01.Gap, ac01.Blocking)
	}

	// 正式 invalidated 的 FAIL 解除阻断;binding 的有效 PASS 覆盖 acceptance。
	invalidatedFail := failRec("EV-AC01-01", 6)
	invalidatedFail.Validity = EvidenceValidityInvalidated
	rep, err = DeriveFunctionalCoverage(frozen, []EvidenceRecord{
		invalidatedFail,
		passRec("EV-AC01-01", 6),
		passRec("EV-AC02-H1", 6),
		passRec("EV-AC03-01", 6),
	})
	if err != nil {
		t.Fatalf("DeriveFunctionalCoverage(invalidated fail): %v", err)
	}
	ac01 = rep.Acceptances[0]
	if !ac01.Covered || ac01.Gap != GapNone || len(ac01.Blocking) != 0 || len(ac01.SatisfiedBy) != 1 {
		t.Fatalf("invalidated FAIL: covered=%v gap=%q blocking=%v satisfiedBy=%v, want covered",
			ac01.Covered, ac01.Gap, ac01.Blocking, ac01.SatisfiedBy)
	}
	if !rep.Complete {
		t.Fatalf("expected Complete=true after FAIL invalidation")
	}

	// 旧 revision 的 FAIL 不适用于当前 revision(§8.8.2):不阻断当前状态。
	rep, err = DeriveFunctionalCoverage(frozen, []EvidenceRecord{
		failRec("EV-AC01-01", 5),
		passRec("EV-AC01-01", 6),
		passRec("EV-AC02-H1", 6),
		passRec("EV-AC03-01", 6),
	})
	if err != nil {
		t.Fatalf("DeriveFunctionalCoverage(old-revision fail): %v", err)
	}
	if !rep.Complete || len(rep.Acceptances[0].Blocking) != 0 {
		t.Fatalf("inapplicable FAIL blocked current revision: complete=%v blocking=%v", rep.Complete, rep.Acceptances[0].Blocking)
	}
}

// 契约外 binding id 的记录(supplementary / 来源不明,§8.12)不参与覆盖:
// 其 FAIL 不阻断,其 PASS 不满足;但要在 Unmatched 中可追溯(INV-STATE-02)。
func TestDeriveFunctionalCoverageIgnoresRecordsOutsideContract(t *testing.T) {
	frozen := frozenStdContract(t)

	rep, err := DeriveFunctionalCoverage(frozen, []EvidenceRecord{
		failRec("EV-SUPP-01", 6),
		passRec("EV-SUPP-01", 6),
		passRec("EV-AC01-01", 6),
		passRec("EV-AC02-H1", 6),
		passRec("EV-AC03-01", 6),
	})
	if err != nil {
		t.Fatalf("DeriveFunctionalCoverage: %v", err)
	}
	if len(rep.Unmatched) != 1 || rep.Unmatched[0] != "EV-SUPP-01" {
		t.Fatalf("Unmatched = %v, want [EV-SUPP-01]", rep.Unmatched)
	}
	if !rep.Complete {
		t.Fatalf("supplementary records must not affect coverage; acceptances=%+v", rep.Acceptances)
	}
}

// 零 binding 的 acceptance(经最小 Freeze 入口,绕过 FreezeContract 的
// evidence-plan 检查)→ GapNoEvidence(§8.1 Completeness 兜底)。
func TestDeriveFunctionalCoverageZeroBindingAcceptance(t *testing.T) {
	c := mustParse(t, `
task_id: TASK-COV
revision: 1
acceptances:
  - id: AC-PLAIN
    statement: no evidence plan (minimal Freeze entry only)
  - id: AC-OK
    statement: covered by validator
    evidence_bindings:
      - id: EV-ACOK-01
        authoring_mode: pre_commit
        visibility_policy: visible
        evidence:
          type: validator
          validator: retry-budget-v1
`)
	frozen, err := c.Freeze()
	if err != nil {
		t.Fatalf("Contract.Freeze: %v", err)
	}

	rep, err := DeriveFunctionalCoverage(frozen, []EvidenceRecord{passRec("EV-ACOK-01", 1)})
	if err != nil {
		t.Fatalf("DeriveFunctionalCoverage: %v", err)
	}
	if rep.Acceptances[0].AcceptanceID != "AC-PLAIN" || rep.Acceptances[0].Covered || rep.Acceptances[0].Gap != GapNoEvidence {
		t.Fatalf("AC-PLAIN = %+v, want uncovered/%s", rep.Acceptances[0], GapNoEvidence)
	}
	if !rep.Acceptances[1].Covered || rep.Complete {
		t.Fatalf("AC-OK covered=%v complete=%v, want covered but Complete=false", rep.Acceptances[1].Covered, rep.Complete)
	}
}

// 输入形态非法时显式报错:调用方错误不静默修正。
func TestDeriveFunctionalCoverageInputErrors(t *testing.T) {
	frozen := frozenStdContract(t)

	if _, err := DeriveFunctionalCoverage(nil, nil); err == nil {
		t.Fatalf("nil contract: expected error, got nil")
	}

	cases := map[string]EvidenceRecord{
		"empty binding id": {},
		"unknown status":   {BindingID: "EV-AC01-01", Revision: 6, Status: EvidenceStatus("green")},
		"unknown validity": {BindingID: "EV-AC01-01", Revision: 6, Status: EvidenceStatusPass, Validity: EvidenceValidity("maybe")},
		"zero revision":    {BindingID: "EV-AC01-01", Revision: 0, Status: EvidenceStatusPass},
	}
	for name, r := range cases {
		t.Run(name, func(t *testing.T) {
			rep, err := DeriveFunctionalCoverage(frozen, []EvidenceRecord{r})
			if err == nil {
				t.Fatalf("expected error, got report %+v", rep)
			}
			if !strings.Contains(err.Error(), "EV-AC01-01") && !strings.Contains(err.Error(), "binding id") {
				t.Fatalf("error = %q, want binding/record context", err)
			}
		})
	}
}
