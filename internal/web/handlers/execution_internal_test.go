package handlers

import (
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"tally/internal/core"
	"tally/internal/environment"
	"tally/internal/store"
)

// 执行引擎页面的推导逻辑全是纯函数,在此钉住(不碰网络与数据库)。

func TestEnvStatusView(t *testing.T) {
	tests := []struct {
		status    string
		wantText  string
		wantClass string
	}{
		{"up", "在跑", "st-ok"},
		{"failed", "上次拉起失败", "st-bad"},
		{"torn_down", "已停止", "st-off"},
		{"", "尚未拉起", "st-off"},
		{"别的", "尚未拉起", "st-off"},
	}
	for _, tt := range tests {
		text, class := envStatusView(tt.status)
		if text != tt.wantText || class != tt.wantClass {
			t.Errorf("envStatusView(%q) = (%q,%q), 想要 (%q,%q)",
				tt.status, text, class, tt.wantText, tt.wantClass)
		}
	}
}

func TestDeployStatusView(t *testing.T) {
	tests := []struct {
		status    string
		wantText  string
		wantClass string
	}{
		{"deployed", "部署成功", "st-ok"},
		{"failed", "部署失败", "st-bad"},
		{"rolled_back", "已回滚", "st-ok"},
		{"rollback_failed", "回滚失败", "st-bad"},
	}
	for _, tt := range tests {
		text, class := deployStatusView(tt.status)
		if text != tt.wantText || class != tt.wantClass {
			t.Errorf("deployStatusView(%q) = (%q,%q), 想要 (%q,%q)",
				tt.status, text, class, tt.wantText, tt.wantClass)
		}
	}
}

func TestClassificationView(t *testing.T) {
	tests := []struct {
		name      string
		status    string
		drifted   bool
		wantText  string
		wantClass string
	}{
		{"已验证", "fixed", false, "已验证修复", "ok"},
		{"已验证但漂移", "fixed", true, "修复可能过期", "warn"},
		{"偶发", "flaky", false, "偶发,不入库", ""},
		{"等上游", "upstream", false, "等上游", "warn"},
		{"基础设施", "infra", false, "基础设施", ""},
		{"需求缺陷", "spec_defect", false, "需求缺陷", "bad"},
		{"未诊断", "unresolved", false, "未诊断", ""},
	}
	for _, tt := range tests {
		text, class := classificationView(tt.status, tt.drifted)
		if text != tt.wantText || class != tt.wantClass {
			t.Errorf("%s: classificationView(%q,%v) = (%q,%q), 想要 (%q,%q)",
				tt.name, tt.status, tt.drifted, text, class, tt.wantText, tt.wantClass)
		}
	}
}

func TestRankValueIsCountTimesProjects(t *testing.T) {
	if got := rankValue(14, 1); got != 14 {
		t.Errorf("rankValue(14,1) = %d, 想要 14", got)
	}
	if got := rankValue(6, 4); got != 24 {
		t.Errorf("rankValue(6,4) = %d, 想要 24(出现次数 × 影响项目数)", got)
	}
	// 项目数为 0 时按 1 兜底,不让排行分归零。
	if got := rankValue(5, 0); got != 5 {
		t.Errorf("rankValue(5,0) = %d, 想要 5", got)
	}
}

func TestComboBrief(t *testing.T) {
	got := comboBrief([]byte(`{"docker":"27.0","compose":"2.29","image:0":"alpine:3.20"}`))
	want := "compose=2.29 docker=27.0 image:0=alpine:3.20" // 键排序
	if got != want {
		t.Errorf("comboBrief = %q, 想要 %q", got, want)
	}
	if got := comboBrief([]byte("不是 JSON")); got != "" {
		t.Errorf("坏 JSON 应落空,得到 %q", got)
	}
	if got := comboBrief(nil); got != "" {
		t.Errorf("空组合应落空,得到 %q", got)
	}
}

func TestComboText(t *testing.T) {
	// 与 comboBrief 同一契约(同源 formatComboMap):键排序 k=v 空格连接。
	got := comboText(core.VersionCombo{"docker": "27.0", "compose": "2.29", "image:0": "alpine:3.20"})
	want := "compose=2.29 docker=27.0 image:0=alpine:3.20"
	if got != want {
		t.Errorf("comboText = %q, 想要 %q", got, want)
	}
	if got := comboText(nil); got != "" {
		t.Errorf("空组合应落空,得到 %q", got)
	}
	// 超长截断:与 comboBrief 同一条规则,改一处两者同变。(len 按字节,「…」3 字节 → 80+3)
	long := comboText(core.VersionCombo{"k": strings.Repeat("v", 100)})
	if len(long) != 83 || !strings.HasSuffix(long, "…") {
		t.Errorf("超长应截断到 80 + 省略号,得到 %q(长 %d)", long, len(long))
	}
}

func TestExcerptFirstLine(t *testing.T) {
	if got := excerptFirstLine("第一行\n第二行"); got != "第一行" {
		t.Errorf("应取首行,得到 %q", got)
	}
	long := string(make([]byte, 200))
	if got := excerptFirstLine(long); len(got) <= 81 {
		t.Errorf("超长应截断,得到 %d 字节", len(got))
	}
}

func TestValidDefPath(t *testing.T) {
	valid := []string{".tally/environments/dev.yaml", "a/b/c.yml", "x.yaml"}
	invalid := []string{"", "/abs.yaml", "../evil.yaml", "a/../b.yaml", `a\b.yaml`, "a//b.yaml", "x.txt", "./x.yaml"}
	for _, s := range valid {
		if !validDefPath(s) {
			t.Errorf("validDefPath(%q) 应为 true", s)
		}
	}
	for _, s := range invalid {
		if validDefPath(s) {
			t.Errorf("validDefPath(%q) 应为 false", s)
		}
	}
}

func ts(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }

func TestMergeEnvRowNotLaunched(t *testing.T) {
	def := environment.Definition{Project: "forge", Name: "dev", Kind: environment.KindDockerCompose, Hash: "sha256:abcdef"}
	row := mergeEnvRow(def, ".tally/environments/dev.yaml", store.EnvironmentRun{})
	if row.StatusText != "尚未拉起" || row.StateClass != "st-off" {
		t.Errorf("无记录应「尚未拉起」: %+v", row)
	}
	if row.Def != ".tally/environments/dev.yaml" || row.DefHash != "abcdef" {
		t.Errorf("定义坐标应带出: %+v", row)
	}
}

func TestMergeEnvRowStatusAndDriftNote(t *testing.T) {
	def := environment.Definition{Project: "forge", Name: "dev", Hash: "sha256:新版"}
	run := store.EnvironmentRun{
		ID: 7, Status: "failed", DefinitionHash: "sha256:旧版",
		StartedAt: ts(time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)),
	}
	row := mergeEnvRow(def, "e.yaml", run)
	if row.StatusText != "上次拉起失败" || row.StateClass != "st-bad" {
		t.Errorf("failed 应「上次拉起失败」: %+v", row)
	}
	if row.Note != "上次拉起用的是旧版定义" {
		t.Errorf("定义哈希对不上应提示旧版定义: %+v", row)
	}
	// 哈希一致时不该有提示
	run.DefinitionHash = def.Hash
	if row2 := mergeEnvRow(def, "e.yaml", run); row2.Note != "" {
		t.Errorf("定义一致不应有提示: %+v", row2)
	}
}

func TestFailureRowsMergeAndDrift(t *testing.T) {
	fixAt := time.Date(2026, 6, 14, 9, 0, 0, 0, time.UTC)
	stats := []store.FingerprintStat{
		{
			FpExact: "a3f9c2", FpFuzzy: "fz1",
			OccurrenceCount: 6, AffectedProjects: []string{"forge", "hsk", "restudio", "orienta"},
			Status: "fixed", FixVerifiedAt: ts(fixAt),
			LastSeen: ts(fixAt.Add(48 * time.Hour)), // 修复验证后又出现 → 漂移
		},
		{
			FpExact: "7b1e04", FpFuzzy: "fz2",
			OccurrenceCount: 14, AffectedProjects: []string{"forge"},
			Status: "fixed", FixVerifiedAt: ts(fixAt),
			LastSeen: ts(fixAt.Add(-48 * time.Hour)), // 修复验证前的末次 → 未漂移
		},
	}
	excerpts := []store.ListLatestFailureExcerptsRow{
		{FpExact: "a3f9c2", Source: "build", Excerpt: pgtype.Text{String: "ABI 不匹配\n第二行", Valid: true}},
	}
	rows := failureRows(stats, excerpts)
	if len(rows) != 2 {
		t.Fatalf("应归并成 2 行,得到 %d", len(rows))
	}
	byFP := map[string]FailureRow{rows[0].Fp: rows[0], rows[1].Fp: rows[1]}

	drifted := byFP["a3f9c2"]
	if drifted.StatusText != "修复可能过期" || drifted.TagClass != "warn" {
		t.Errorf("漂移指纹应标「修复可能过期」: %+v", drifted)
	}
	if drifted.Rank != 24 { // 6 × 4
		t.Errorf("排行分应 24,得到 %d", drifted.Rank)
	}
	if drifted.Excerpt != "ABI 不匹配" {
		t.Errorf("摘要应取最近片段首行: %q", drifted.Excerpt)
	}
	if drifted.SourceText != "构建" {
		t.Errorf("来源应是「构建」: %q", drifted.SourceText)
	}

	fixed := byFP["7b1e04"]
	if fixed.StatusText != "已验证修复" || fixed.TagClass != "ok" {
		t.Errorf("未漂移的 fixed 应标「已验证修复」: %+v", fixed)
	}
	if fixed.Rank != 14 { // 14 × 1
		t.Errorf("排行分应 14,得到 %d", fixed.Rank)
	}
}

func TestTopRankingOrderAndCap(t *testing.T) {
	rows := []FailureRow{
		{Fp: "low", Rank: 2}, {Fp: "high", Rank: 24}, {Fp: "mid", Rank: 14}, {Fp: "tie", Rank: 24},
	}
	top := topRanking(rows, 3)
	if len(top) != 3 {
		t.Fatalf("应截到 3 行,得到 %d", len(top))
	}
	// 同分按指纹序:high < tie
	want := []string{"high", "tie", "mid"}
	for i, fp := range want {
		if top[i].Fp != fp {
			t.Errorf("第 %d 名应是 %s,得到 %s(%v)", i+1, fp, top[i].Fp, top)
		}
	}
}

func TestBuildTimelineAlignsFailuresWithDeps(t *testing.T) {
	now := time.Date(2026, 7, 21, 15, 0, 0, 0, time.UTC)
	dayA := time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)
	dayB := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)

	pgDay := func(d time.Time) pgtype.Date { return pgtype.Date{Time: d, Valid: true} }
	failures := []store.CountFailuresByDayRow{
		{Day: pgDay(dayA), Failures: 5},
		{Day: pgDay(dayB), Failures: 10}, // 峰值 → 100%
	}
	deps := []store.DependencyChange{
		{
			ProjectID: "forge", Package: "node", ToVersion: pgtype.Text{String: "22", Valid: true},
			ChangedAt: ts(dayB.Add(3 * time.Hour)), // 与失败峰值同一天 → 对齐
		},
	}

	tl := buildTimeline(failures, deps, now, 90)
	if len(tl) != 90 {
		t.Fatalf("应 90 天刻度,得到 %d", len(tl))
	}
	if tl[0].Full != "2026-04-23" || tl[89].Full != "2026-07-21" {
		t.Errorf("窗口应是最近 90 天:%s → %s", tl[0].Full, tl[89].Full)
	}

	byDay := map[string]TimelineDay{}
	for _, d := range tl {
		byDay[d.Full] = d
	}
	if b := byDay["2026-07-15"]; !b.HasDep || b.Pct != 100 || b.Failures != 10 {
		t.Errorf("峰值日应对齐依赖刻度且满柱: %+v", b)
	}
	if b := byDay["2026-07-15"]; b.DepTitle != "forge: node → 22" {
		t.Errorf("依赖刻度标题应含项目包版本: %q", b.DepTitle)
	}
	if a := byDay["2026-07-10"]; a.HasDep || a.Pct != 50 || a.Failures != 5 {
		t.Errorf("半高日应无依赖刻度: %+v", a)
	}
	if z := byDay["2026-07-01"]; z.Failures != 0 || z.Pct != 0 || z.HasDep {
		t.Errorf("空日应是零刻度: %+v", z)
	}
}
