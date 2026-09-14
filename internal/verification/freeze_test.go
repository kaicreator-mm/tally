package verification

import (
	"strings"
	"testing"
)

// freezeStdContract 返回一个能通过 P1 standard path freeze 的完整契约
// (复用 types_test.go 的 fullContractYAML:全部 binding pre_commit、
// protection 均带 baseline_hash、AC-01 source / AC-02 information 的
// planned coverage 均可满足)。
func freezeStdContract(t *testing.T) *Contract {
	t.Helper()
	return mustParse(t, fullContractYAML)
}

// freeze 入口与类型层 Contract.Freeze 对同一内容产生完全一致的冻结结果;
// 且冻结视图与作者侧对象脱钩(P1 standard path,零值策略)。
func TestFreezeContractStandardPath(t *testing.T) {
	c := freezeStdContract(t)

	frozen, err := FreezeContract(c, FreezePolicy{})
	if err != nil {
		t.Fatalf("FreezeContract: %v", err)
	}
	if frozen.TaskID() != "TASK-023" || frozen.Revision() != 6 {
		t.Fatalf("frozen task/revision = %s/%d", frozen.TaskID(), frozen.Revision())
	}
	if h := frozen.Hash(); !strings.HasPrefix(h, "sha256:") || len(h) != len("sha256:")+64 {
		t.Fatalf("hash format = %q", h)
	}

	// 与 Contract.Freeze 的结果一致(同内容 → 同哈希)。
	cf, err := c.Freeze()
	if err != nil {
		t.Fatalf("Contract.Freeze: %v", err)
	}
	if cf.Hash() != frozen.Hash() {
		t.Fatalf("FreezeContract hash %s != Contract.Freeze hash %s", frozen.Hash(), cf.Hash())
	}

	// 冻结后改动作者侧不影响冻结视图。
	c.Revision = 99
	if frozen.Revision() != 6 || frozen.Hash() != cf.Hash() {
		t.Fatalf("frozen view changed after mutating author-side contract")
	}
}

// AC1:每个 required acceptance 必须有功能验证计划(至少一条 planned evidence
// binding);没有任何 binding 的 acceptance 不能通过 freeze。
func TestFreezeContractRequiresFunctionalEvidencePlan(t *testing.T) {
	c := freezeStdContract(t)
	// 仅当 independence coverage 不先失败时才能定位到 evidence plan 缺失,
	// 因此把 AC-01 降为 none 再清空它的 bindings。
	c.Acceptances[0].IndependenceRequired = IndependenceNone
	c.Acceptances[0].EvidenceBindings = nil

	err := ValidateFreeze(c, FreezePolicy{})
	if err == nil {
		t.Fatalf("ValidateFreeze: expected evidence-plan error, got nil")
	}
	if !strings.Contains(err.Error(), "AC-01") || !strings.Contains(err.Error(), "planned evidence binding") {
		t.Fatalf("ValidateFreeze error = %q, want acceptance id + planned evidence binding", err)
	}

	if _, err := FreezeContract(c, FreezePolicy{}); err == nil {
		t.Fatalf("FreezeContract: expected error, got nil")
	} else if !strings.Contains(err.Error(), "planned evidence binding") {
		t.Fatalf("FreezeContract error = %q", err)
	}
}

// AC2:source/information independence 的 planned coverage(PRD §8.5.1)。
// 设计阶段缺失 → FREEZE FAIL,无 Waiver;Waiver 只属于运行期 independence loss。
func TestFreezeContractPlannedIndependenceCoverage(t *testing.T) {
	t.Run("source without capable binding basis", func(t *testing.T) {
		for name, mutate := range map[string]func(*Contract){
			"basis is implementation_authority": func(c *Contract) {
				c.Acceptances[0].EvidenceBindings[0].Basis.Source = SourceImplementationAuthority
			},
			"basis missing": func(c *Contract) {
				c.Acceptances[0].EvidenceBindings[0].Basis = nil
			},
		} {
			t.Run(name, func(t *testing.T) {
				c := freezeStdContract(t)
				mutate(c)
				err := ValidateFreeze(c, FreezePolicy{})
				if err == nil {
					t.Fatalf("expected FREEZE FAIL, got nil")
				}
				if !strings.Contains(err.Error(), "AC-01") || !strings.Contains(err.Error(), "FREEZE FAIL") {
					t.Fatalf("error = %q, want AC-01 planned coverage FREEZE FAIL", err)
				}
				if _, err := FreezeContract(c, FreezePolicy{}); err == nil {
					t.Fatalf("FreezeContract: expected FREEZE FAIL, got nil")
				}
			})
		}
	})

	t.Run("information without held_out binding", func(t *testing.T) {
		c := freezeStdContract(t)
		// 把 AC-02 唯一的 held_out binding 改成 visible repo 测试:
		// source basis 仍声明,但不再可能满足 information independence。
		b := &c.Acceptances[1].EvidenceBindings[0]
		b.VisibilityPolicy = VisibilityVisible
		b.Evidence = EvidenceRef{Type: EvidenceTest, Path: "internal/jobs/lease_safety_test.go", Selector: "TestNeverReclaimLiveLease"}
		b.Basis = &IndependenceBasis{Source: SourcePreExisting, Information: InfoVisible}

		err := ValidateFreeze(c, FreezePolicy{})
		if err == nil {
			t.Fatalf("expected FREEZE FAIL, got nil")
		}
		if !strings.Contains(err.Error(), "AC-02") || !strings.Contains(err.Error(), "FREEZE FAIL") {
			t.Fatalf("error = %q, want AC-02 planned coverage FREEZE FAIL", err)
		}
		if _, err := FreezeContract(c, FreezePolicy{}); err == nil {
			t.Fatalf("FreezeContract: expected FREEZE FAIL, got nil")
		}
	})

	t.Run("satisfied coverage freezes cleanly", func(t *testing.T) {
		// fullContractYAML 的正向情形:AC-01(source)由 verification_authority
		// basis 的 visible binding 覆盖;AC-02(information)由 held_out binding 覆盖。
		c := freezeStdContract(t)
		if _, err := FreezeContract(c, FreezePolicy{}); err != nil {
			t.Fatalf("FreezeContract: %v", err)
		}
	})
}

// AC3:post_hoc 在 P1 standard path 默认拒绝;只有 FreezePolicy 显式允许才通过。
func TestFreezeContractRejectsPostHocByDefault(t *testing.T) {
	c := freezeStdContract(t)
	c.Acceptances[2].EvidenceBindings[0].AuthoringMode = AuthoringPostHoc

	if err := ValidateFreeze(c, FreezePolicy{}); err == nil {
		t.Fatalf("ValidateFreeze: expected post_hoc rejection, got nil")
	} else if !strings.Contains(err.Error(), "post_hoc") {
		t.Fatalf("ValidateFreeze error = %q, want post_hoc", err)
	}
	if _, err := FreezeContract(c, FreezePolicy{}); err == nil {
		t.Fatalf("FreezeContract: expected post_hoc rejection, got nil")
	}

	// 显式策略放行后同一契约可通过(治理升级流程本身的 candidate-blind/
	// explicit review 约束归 planner/review 任务,不在契约形态层)。
	allowed := freezeStdContract(t)
	allowed.Acceptances[2].EvidenceBindings[0].AuthoringMode = AuthoringPostHoc
	frozen, err := FreezeContract(allowed, FreezePolicy{AllowPostHoc: true})
	if err != nil {
		t.Fatalf("FreezeContract with AllowPostHoc: %v", err)
	}
	if frozen.Hash() == "" {
		t.Fatalf("expected frozen hash")
	}
}

// AC4:声明 protection 的 binding 必须带 baseline_hash;未声明 protection 的
// binding 不受此约束(where applicable)。
func TestFreezeContractRequiresProtectedHash(t *testing.T) {
	c := freezeStdContract(t)
	c.Acceptances[0].EvidenceBindings[0].Protection.BaselineHash = ""

	if err := ValidateFreeze(c, FreezePolicy{}); err == nil {
		t.Fatalf("ValidateFreeze: expected protected-hash error, got nil")
	} else if !strings.Contains(err.Error(), "baseline_hash") || !strings.Contains(err.Error(), "EV-AC01-01") {
		t.Fatalf("ValidateFreeze error = %q, want binding id + baseline_hash", err)
	}

	if _, err := FreezeContract(c, FreezePolicy{}); err == nil {
		t.Fatalf("FreezeContract: expected protected-hash error, got nil")
	}

	// protection 未声明 → 不适用(where applicable)。
	noProtection := freezeStdContract(t)
	noProtection.Acceptances[0].EvidenceBindings[0].Protection = nil
	if _, err := FreezeContract(noProtection, FreezePolicy{}); err != nil {
		t.Fatalf("FreezeContract without protection declaration: %v", err)
	}
}

func TestFreezeValidatorsRejectNilContract(t *testing.T) {
	if err := ValidateFreeze(nil, FreezePolicy{}); err == nil {
		t.Fatalf("ValidateFreeze(nil): expected error, got nil")
	}
	if _, err := FreezeContract(nil, FreezePolicy{}); err == nil {
		t.Fatalf("FreezeContract(nil): expected error, got nil")
	}
}
