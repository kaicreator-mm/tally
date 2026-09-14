package intentevidence

import (
	"errors"
	"reflect"
	"testing"

	"tally/internal/risk"
)

// deriveFacts 构造一份冻结意图,失败即测试失败。
func deriveFacts(t *testing.T, taskType TaskType, hints ...string) FrozenIntent {
	t.Helper()
	fi, err := Freeze(Facts{TaskID: "TASK-0023", TaskType: taskType, RiskHints: hints})
	if err != nil {
		t.Fatalf("Freeze: %v", err)
	}
	return fi
}

// ---- 验收标准 1:已知 migration/auth/protected-surface 事实 ⇒ 确定性信号+basis

// TestDeriveRiskKnownFactsProduceDeterministicSignals 验收标准 1 的推导表:
// 已知迁移/认证/受保护面事实(确认提示与 task_type 确定性事实)确定性地产出
// 对应信号,且 basis 引用逐字保留出处。
func TestDeriveRiskKnownFactsProduceDeterministicSignals(t *testing.T) {
	cases := []struct {
		name         string
		intent       FrozenIntent
		wantDetected []risk.Signal
		wantUnknown  []risk.SignalID
		aboveMinimal bool
		hasUnknown   bool
	}{
		{
			name:   "bug_fix 无确认提示 ⇒ 空集(合法结果,不是错误)",
			intent: deriveFacts(t, TaskTypeBugFix),
		},
		{
			name:         "db_migration 任务类型 ⇒ 确定性 schema_migration(tasktype basis)",
			intent:       deriveFacts(t, TaskTypeMigration),
			wantDetected: []risk.Signal{{ID: risk.SignalSchemaMigration, Basis: []string{"tasktype:TASK-0023"}}},
			aboveMinimal: true,
		},
		{
			name:   "确认的 auth/protected 提示 ⇒ 对应信号(intent basis)",
			intent: deriveFacts(t, TaskTypeBugFix, "auth_symbol_change", "protected_surface_change"),
			wantDetected: []risk.Signal{
				{ID: risk.SignalAuthSymbolChange, Basis: []string{"intent:TASK-0023"}},
				{ID: risk.SignalProtectedSurfaceChange, Basis: []string{"intent:TASK-0023"}},
			},
			aboveMinimal: true,
		},
		{
			name:   "确认的 migration 提示 + db_migration 类型 ⇒ 同信号合并,basis 并集保两源出处",
			intent: deriveFacts(t, TaskTypeMigration, "schema_migration"),
			wantDetected: []risk.Signal{{
				ID:    risk.SignalSchemaMigration,
				Basis: []string{"intent:TASK-0023", "tasktype:TASK-0023"},
			}},
			aboveMinimal: true,
		},
		{
			name:   "db_migration + auth/protected 提示 ⇒ 三信号齐备",
			intent: deriveFacts(t, TaskTypeMigration, "auth_symbol_change", "protected_surface_change"),
			wantDetected: []risk.Signal{
				{ID: risk.SignalAuthSymbolChange, Basis: []string{"intent:TASK-0023"}},
				{ID: risk.SignalProtectedSurfaceChange, Basis: []string{"intent:TASK-0023"}},
				{ID: risk.SignalSchemaMigration, Basis: []string{"tasktype:TASK-0023"}},
			},
			aboveMinimal: true,
		},
		{
			name:         "枚举之外的确认提示 ⇒ Unknown 原样保真(INV-V14-09),不丢弃不降级",
			intent:       deriveFacts(t, TaskTypeBugFix, "payment_flow_change"),
			wantUnknown:  []risk.SignalID{"payment_flow_change"},
			aboveMinimal: true,
			hasUnknown:   true,
		},
	}
	for _, tc := range cases {
		res, err := tc.intent.DeriveRisk()
		if err != nil {
			t.Fatalf("%s: DeriveRisk: %v", tc.name, err)
		}
		if !reflect.DeepEqual(res.Detected, tc.wantDetected) {
			t.Errorf("%s: Detected = %+v, want %+v", tc.name, res.Detected, tc.wantDetected)
		}
		var gotUnknown []risk.SignalID
		for _, s := range res.Unknown {
			gotUnknown = append(gotUnknown, s.ID)
		}
		if !reflect.DeepEqual(gotUnknown, tc.wantUnknown) {
			t.Errorf("%s: Unknown = %v, want %v", tc.name, gotUnknown, tc.wantUnknown)
		}
		if res.AboveMinimal() != tc.aboveMinimal {
			t.Errorf("%s: AboveMinimal() = %v, want %v", tc.name, res.AboveMinimal(), tc.aboveMinimal)
		}
		if res.HasUnknown() != tc.hasUnknown {
			t.Errorf("%s: HasUnknown() = %v, want %v", tc.name, res.HasUnknown(), tc.hasUnknown)
		}
	}
}

// ---- 确定性(同一证据重算 ⇒ 逐字段相同)--------------------------------

// TestDeriveRiskDeterministicAcrossEqualEvidence:同一份事实的两种排版/顺序
// 冻结出同 Revision 证据,推导结果必须逐字段相同;重复重算也相同。
func TestDeriveRiskDeterministicAcrossEqualEvidence(t *testing.T) {
	base := deriveFacts(t, TaskTypeMigration, "auth_symbol_change", "schema_migration")
	shuffled, err := Freeze(Facts{
		TaskID:    "TASK-0023",
		TaskType:  TaskTypeMigration,
		RiskHints: []string{"schema_migration", "auth_symbol_change"},
	})
	if err != nil {
		t.Fatalf("Freeze(变体): %v", err)
	}
	if base.Revision() != shuffled.Revision() {
		t.Fatalf("排版变体冻结出不同 Revision: %s != %s", base.Revision(), shuffled.Revision())
	}
	want, err := base.DeriveRisk()
	if err != nil {
		t.Fatalf("DeriveRisk(base): %v", err)
	}
	for i := 0; i < 3; i++ {
		got, err := shuffled.DeriveRisk()
		if err != nil {
			t.Fatalf("DeriveRisk(shuffled): %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("同内容证据推导出不同结果: got %+v, want %+v", got, want)
		}
	}
}

// ---- 纯函数:不改输入、结果改动不影响重算(PRD §22)---------------------

func TestDeriveRiskIsPure(t *testing.T) {
	fi := deriveFacts(t, TaskTypeMigration, "auth_symbol_change")
	first, err := fi.DeriveRisk()
	if err != nil {
		t.Fatalf("DeriveRisk: %v", err)
	}
	// 调用方改动结果(含切片)不得影响重算。
	for i := range first.Detected {
		first.Detected[i].ID = "mutated"
		first.Detected[i].Basis[0] = "mutated"
	}
	again, err := fi.DeriveRisk()
	if err != nil {
		t.Fatalf("DeriveRisk 重算: %v", err)
	}
	fresh, err := deriveFacts(t, TaskTypeMigration, "auth_symbol_change").DeriveRisk()
	if err != nil {
		t.Fatalf("DeriveRisk(fresh): %v", err)
	}
	if !reflect.DeepEqual(again, fresh) {
		t.Fatalf("重算观察到先前结果的改动: got %+v, want %+v", again, fresh)
	}

	// 候选投影切片是调用方私有拷贝:改它不得影响证据或重算。
	cands := fi.RiskSignalCandidates()
	if len(cands) > 0 {
		cands[0].Basis[0] = "mutated"
	}
	if _, err := fi.DeriveRisk(); err != nil {
		t.Fatalf("候选切片改动后 DeriveRisk: %v", err)
	}
	if got := fi.RiskHints(); !reflect.DeepEqual(got, []string{"auth_symbol_change"}) {
		t.Fatalf("证据被候选切片改动穿透: %v", got)
	}
}

// ---- 验收标准 2(负例):缺 basis / 非证据按风险不变量拒收 ---------------

// TestDeriveRiskRejectsZeroValue:零值不是证据,拒绝从非证据推导出
// 「无风险」(那是静默低分类)。
func TestDeriveRiskRejectsZeroValue(t *testing.T) {
	var zero FrozenIntent
	if !zero.IsZero() {
		t.Fatal("夹具失真:零值应为 IsZero")
	}
	res, err := zero.DeriveRisk()
	if !errors.Is(err, ErrInvalidEvidence) {
		t.Fatalf("DeriveRisk(零值) err = %v, want ErrInvalidEvidence", err)
	}
	if res.Detected != nil || res.Unknown != nil {
		t.Fatalf("拒收时不得携带部分结果: %+v", res)
	}
}

// TestDeriveRiskChainRejectsMissingBasis 验收标准 2:推导链(risk.Derive 入口)
// 保持风险不变量 —— 缺 basis 的候选整体拒收为 ErrMalformedSignal,绝不产出
// 部分结果;新推导路径没有引入任何绕过通道。
func TestDeriveRiskChainRejectsMissingBasis(t *testing.T) {
	fi := deriveFacts(t, TaskTypeMigration, "auth_symbol_change")
	candidates := append(fi.RiskSignalCandidates(), fi.TaskTypeRiskCandidates()...)
	if len(candidates) != 2 {
		t.Fatalf("候选投影 = %d 条, want 2(1 提示 + 1 task_type)", len(candidates))
	}
	// 注入一条缺 basis 的候选:推导入口必须按风险不变量拒收。
	candidates = append(candidates, risk.Signal{ID: risk.SignalProtectedSurfaceChange})
	res, err := risk.Derive(candidates)
	if !errors.Is(err, risk.ErrMalformedSignal) {
		t.Fatalf("risk.Derive(缺 basis 候选) err = %v, want ErrMalformedSignal", err)
	}
	if res.Detected != nil || res.Unknown != nil {
		t.Fatalf("拒收时不得携带部分结果: %+v", res)
	}

	// 正向闭环:合法证据的 DeriveRisk 产物全部携带 basis —— 用推导自身输出
	// 重进 risk.Derive 不得报错,且重派生结果逐字段相同(集合语义稳定)。
	authentic, err := fi.DeriveRisk()
	if err != nil {
		t.Fatalf("DeriveRisk: %v", err)
	}
	if len(authentic.All()) != 2 {
		t.Fatalf("All() = %d 条, want 2", len(authentic.All()))
	}
	redone, err := risk.Derive(authentic.All())
	if err != nil {
		t.Fatalf("DeriveRisk 产物缺少 basis/重复 id: %v", err)
	}
	if !reflect.DeepEqual(redone, authentic) {
		t.Fatalf("产物重派生不一致: got %+v, want %+v", redone, authentic)
	}
}

// ---- 确定性事实投影(TaskTypeRiskCandidates)的边界 ----------------------

func TestTaskTypeRiskCandidatesBoundaries(t *testing.T) {
	if got := deriveFacts(t, TaskTypeBugFix).TaskTypeRiskCandidates(); got != nil {
		t.Errorf("bug_fix 应无确定性候选, got %+v", got)
	}
	var zero FrozenIntent
	if got := zero.TaskTypeRiskCandidates(); got != nil {
		t.Errorf("零值应无确定性候选, got %+v", got)
	}
	got := deriveFacts(t, TaskTypeMigration).TaskTypeRiskCandidates()
	want := []risk.Signal{{ID: risk.SignalSchemaMigration, Basis: []string{"tasktype:TASK-0023"}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("db_migration 候选 = %+v, want %+v", got, want)
	}
}
