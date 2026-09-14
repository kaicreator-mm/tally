package failure

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"tally/internal/core"
)

// —— 预算判定(确定性,不触 LLM)——

func TestConsultBudget(t *testing.T) {
	cases := []struct {
		name      string
		found     bool
		stat      Stat
		proceed   bool
		archived  bool
		concluded OutcomeKind
	}{
		{"首次失败", false, Stat{}, true, false, ""},
		{"诊断中(未满三轮)", true, Stat{Status: "unresolved", DiagnoseAttempts: 2}, true, false, ""},
		{"三轮用尽 → 归档", true, Stat{Status: "unresolved", DiagnoseAttempts: 3}, false, true, ""},
		{"超过三轮也归档", true, Stat{Status: "unresolved", DiagnoseAttempts: 7}, false, true, ""},
		{"终态 flaky → 直接返回结论", true, Stat{Status: "flaky"}, false, false, OutcomeFlaky},
		{"终态 fixed → 直接返回结论", true, Stat{Status: "fixed", FixRef: "abc"}, false, false, OutcomeFixed},
		{"终态 spec_defect", true, Stat{Status: "spec_defect"}, false, false, OutcomeSpecDefect},
	}
	for _, c := range cases {
		dec := ConsultBudget(c.found, c.stat)
		if dec.Proceed != c.proceed || dec.Archived != c.archived {
			t.Errorf("%s: Proceed/Archived = %v/%v, want %v/%v",
				c.name, dec.Proceed, dec.Archived, c.proceed, c.archived)
		}
		if c.concluded == "" {
			if dec.Concluded != nil {
				t.Errorf("%s: 不应有结论, got %v", c.name, *dec.Concluded)
			}
		} else if dec.Concluded == nil || *dec.Concluded != c.concluded {
			t.Errorf("%s: 结论 = %v, want %v", c.name, dec.Concluded, c.concluded)
		}
	}
}

// —— 内存 StatStore 替身(预算逻辑离线可测)——

type fakeStats struct {
	stats      map[core.Fingerprint]*Stat
	bumps      int
	classified []OutcomeKind
	fixes      map[core.Fingerprint]string
	depChange  bool
}

func newFakeStats() *fakeStats {
	return &fakeStats{stats: map[core.Fingerprint]*Stat{}, fixes: map[core.Fingerprint]string{}}
}

func (f *fakeStats) row(fp core.Fingerprint) *Stat {
	s, ok := f.stats[fp]
	if !ok {
		s = &Stat{}
		f.stats[fp] = s
	}
	return s
}

func (f *fakeStats) GetStat(_ context.Context, fp core.Fingerprint) (Stat, error) {
	if s, ok := f.stats[fp]; ok {
		return *s, nil
	}
	return Stat{}, ErrNoStat
}

func (f *fakeStats) BumpAttempts(_ context.Context, fp core.Fingerprint) error {
	f.row(fp).DiagnoseAttempts++
	f.bumps++
	return nil
}

func (f *fakeStats) SetClassification(_ context.Context, fp core.Fingerprint, kind OutcomeKind) error {
	f.row(fp).Status = string(kind)
	f.classified = append(f.classified, kind)
	return nil
}

func (f *fakeStats) SetFix(_ context.Context, fp core.Fingerprint, fixRef string, _ time.Time, _ core.VersionCombo) error {
	f.fixes[fp] = fixRef
	s := f.row(fp)
	s.Status = "fixed"
	s.FixRef = fixRef
	return nil
}

func (f *fakeStats) HasRecentDepChange(context.Context, string, time.Time) (bool, error) {
	return f.depChange, nil
}

func TestFakeStatsSatisfiesInterface(t *testing.T) {
	var _ StatStore = (*fakeStats)(nil)
}

// —— 索引读写的真实接线(statStore 包装 store.Queries)在集成测试里覆盖;
// 这里只保证 ErrNoStat 哨兵语义稳定。

func TestErrNoStatSentinel(t *testing.T) {
	err := fmt.Errorf("包一层: %w", ErrNoStat)
	if !errors.Is(err, ErrNoStat) {
		t.Error("ErrNoStat 应可用 errors.Is 识别")
	}
}
