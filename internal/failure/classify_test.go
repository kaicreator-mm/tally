package failure

import "testing"

// 验收:五种出口全部实现,且取值与 fingerprint_stat.status 枚举一致(归类直接写库)。
func TestOutcomeKindValues(t *testing.T) {
	cases := map[OutcomeKind]string{
		OutcomeFixed:      "fixed",
		OutcomeFlaky:      "flaky",
		OutcomeUpstream:   "upstream",
		OutcomeInfra:      "infra",
		OutcomeSpecDefect: "spec_defect",
	}
	for kind, want := range cases {
		if string(kind) != want {
			t.Errorf("出口取值 %q != 数据库枚举 %q", kind, want)
		}
		if !IsTerminal(want) {
			t.Errorf("%s 应是终态", want)
		}
	}
	if IsTerminal("unresolved") || IsTerminal("") {
		t.Error("unresolved/空 不是终态")
	}
}

// 验收:归类是确定性函数,顺序即权重。
func TestClassify(t *testing.T) {
	cases := []struct {
		name string
		ev   Evidence
		want OutcomeKind
	}{
		{"修复验证通过", Evidence{ReproStable: true, FixVerified: true}, OutcomeFixed},
		{"验证通过压过其他信号", Evidence{ReproStable: true, FixVerified: true, CrossProject: true, DepAligned: true}, OutcomeFixed},
		{"复现不稳 → flaky", Evidence{ReproStable: false}, OutcomeFlaky},
		{"不稳压过依赖对齐", Evidence{ReproStable: false, DepAligned: true}, OutcomeFlaky},
		{"需求缺陷成立", Evidence{ReproStable: true, SpecDefectConfirmed: true}, OutcomeSpecDefect},
		{"跨项目 → infra", Evidence{ReproStable: true, CrossProject: true}, OutcomeInfra},
		{"依赖升级对齐 → upstream", Evidence{ReproStable: true, DepAligned: true}, OutcomeUpstream},
		{"infra 压过 upstream", Evidence{ReproStable: true, CrossProject: true, DepAligned: true}, OutcomeInfra},
		{"候选失败无出口", Evidence{ReproStable: true}, ""},
	}
	for _, c := range cases {
		if got := Classify(c.ev); got != c.want {
			t.Errorf("%s: Classify = %q, want %q", c.name, got, c.want)
		}
	}
}
