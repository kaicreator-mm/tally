package deps

import (
	"context"
	"testing"

	"tally/internal/core"
)

// fakeReproRunner 返回固定 green;记录被调次数(断言连跑 5 次)。
type fakeReproRunner struct {
	green bool
	calls int
	err   error
}

func (f *fakeReproRunner) Run(_ context.Context, _, _ string) (bool, error) {
	f.calls++
	return f.green, f.err
}

// fakeRetirer 记录被退休的指纹。
type fakeRetirer struct {
	retired []core.Fingerprint
	err     error
}

func (f *fakeRetirer) Retire(_ context.Context, fp core.Fingerprint, _ string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	f.retired = append(f.retired, fp)
	return "commit-" + string(fp), nil
}

// fakeRetireStore 按指纹给复现上下文;记录被清索引的指纹。
type fakeRetireStore struct {
	ctx     map[core.Fingerprint][2]string // fp -> {workdir, repro}
	cleared []core.Fingerprint
}

func (f *fakeRetireStore) LoadReproContext(_ context.Context, _ string, fp core.Fingerprint) (string, string, error) {
	if c, ok := f.ctx[fp]; ok {
		return c[0], c[1], nil
	}
	return "", "", nil
}

func (f *fakeRetireStore) ClearFingerprintStatFix(_ context.Context, fp core.Fingerprint) error {
	f.cleared = append(f.cleared, fp)
	return nil
}

const pkgPGX = "github.com/jackc/pgx/v5"

func TestRetirePass(t *testing.T) {
	fp := core.Fingerprint("aa11bb22cc33dd44")

	tests := []struct {
		name    string
		changed []string
		fix     FixCandidate
		runner  *fakeReproRunner
		ctx     map[core.Fingerprint][2]string
		// 期望
		wantRetired    bool
		wantStillRepro bool
		wantSkipped    bool
		wantRunnerCall int
		wantCleared    bool
	}{
		{
			name:    "仍稳定复现不退休",
			changed: []string{pkgPGX},
			fix:     FixCandidate{Fingerprint: fp, VerifiedCombo: core.VersionCombo{pkgPGX: "v5.5.1", "go": "1.23.0"}},
			runner:  &fakeReproRunner{green: false}, // 全红 = 仍复现
			ctx:     map[core.Fingerprint][2]string{fp: {"/tmp/demo", "make build"}},
			// 全红 → stable → 不退休
			wantStillRepro: true,
			wantRunnerCall: reproRuns,
		},
		{
			name:           "不再复现退休",
			changed:        []string{pkgPGX},
			fix:            FixCandidate{Fingerprint: fp, VerifiedCombo: core.VersionCombo{pkgPGX: "v5.5.1", "go": "1.23.0"}},
			runner:         &fakeReproRunner{green: true}, // 有绿 = 不再复现
			ctx:            map[core.Fingerprint][2]string{fp: {"/tmp/demo", "make build"}},
			wantRetired:    true,
			wantRunnerCall: reproRuns,
			wantCleared:    true,
		},
		{
			name:        "缺复现上下文跳过",
			changed:     []string{pkgPGX},
			fix:         FixCandidate{Fingerprint: fp, VerifiedCombo: core.VersionCombo{pkgPGX: "v5.5.1"}},
			runner:      &fakeReproRunner{green: false},
			ctx:         nil, // 无上下文
			wantSkipped: true,
		},
		{
			name:    "验证组合未碰变更包无关",
			changed: []string{"golang.org/x/sync"},
			fix:     FixCandidate{Fingerprint: fp, VerifiedCombo: core.VersionCombo{pkgPGX: "v5.5.1"}},
			runner:  &fakeReproRunner{green: true},
			ctx:     map[core.Fingerprint][2]string{fp: {"/tmp/demo", "make build"}},
			// 无关 → 直接跳过,不跑复现
		},
		{
			name:        "已退休不重验",
			changed:     []string{pkgPGX},
			fix:         FixCandidate{Fingerprint: fp, VerifiedCombo: core.VersionCombo{pkgPGX: "v5.5.1"}, Retired: true},
			runner:      &fakeReproRunner{green: true},
			ctx:         map[core.Fingerprint][2]string{fp: {"/tmp/demo", "make build"}},
			wantSkipped: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retirer := &fakeRetirer{}
			st := &fakeRetireStore{ctx: tt.ctx}
			rep, err := RetirePass(context.Background(), RetireOptions{
				ProjectID: "demo",
				Changed:   tt.changed,
				Fixes:     []FixCandidate{tt.fix},
				Runner:    tt.runner,
				Retirer:   retirer,
				Store:     st,
			})
			if err != nil {
				t.Fatalf("RetirePass: %v", err)
			}

			if got := contains(rep.Retired, fp); got != tt.wantRetired {
				t.Errorf("Retired 含 %s = %v,想要 %v(rep=%+v)", fp, got, tt.wantRetired, rep)
			}
			if got := contains(rep.StillRepro, fp); got != tt.wantStillRepro {
				t.Errorf("StillRepro 含 %s = %v,想要 %v(rep=%+v)", fp, got, tt.wantStillRepro, rep)
			}
			if got := contains(rep.Skipped, fp); got != tt.wantSkipped {
				t.Errorf("Skipped 含 %s = %v,想要 %v(rep=%+v)", fp, got, tt.wantSkipped, rep)
			}
			if tt.runner.calls != tt.wantRunnerCall {
				t.Errorf("复现执行次数 = %d,想要 %d", tt.runner.calls, tt.wantRunnerCall)
			}
			if got := contains(retirer.retired, fp); got != tt.wantRetired {
				t.Errorf("Retirer 被调退休 %s = %v,想要 %v", fp, got, tt.wantRetired)
			}
			if got := contains(st.cleared, fp); got != tt.wantCleared {
				t.Errorf("索引面被清 %s = %v,想要 %v", fp, got, tt.wantCleared)
			}
		})
	}
}

// TestRetirePassNoChanges 无变更包 → 直接返回,什么都不查。
func TestRetirePassNoChanges(t *testing.T) {
	runner := &fakeReproRunner{green: true}
	rep, err := RetirePass(context.Background(), RetireOptions{
		ProjectID: "demo",
		Changed:   nil,
		Fixes:     []FixCandidate{{Fingerprint: "aa11bb22", VerifiedCombo: core.VersionCombo{pkgPGX: "v5.5.1"}}},
		Runner:    runner,
		Retirer:   &fakeRetirer{},
		Store:     &fakeRetireStore{},
	})
	if err != nil {
		t.Fatalf("RetirePass: %v", err)
	}
	if len(rep.Retired)+len(rep.StillRepro)+len(rep.Skipped) != 0 {
		t.Errorf("无变更应无动作,得到 %+v", rep)
	}
	if runner.calls != 0 {
		t.Errorf("无变更不应跑复现,跑了 %d 次", runner.calls)
	}
}

func contains(list []core.Fingerprint, fp core.Fingerprint) bool {
	for _, x := range list {
		if x == fp {
			return true
		}
	}
	return false
}
