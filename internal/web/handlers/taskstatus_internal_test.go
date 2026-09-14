package handlers

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"tally/internal/store"
)

// computeTaskRows 的纯逻辑单测:摘要行五栏从执行链怎么推、措辞边界是什么,
// 不碰数据库(P0-22:这是执行/项目摘要,不是 Task 完成判定;INV 状态可重建)。

func int4(v int32) pgtype.Int4 { return pgtype.Int4{Int32: v, Valid: true} }

func rowByProject(rows []TaskRow, p string) (TaskRow, bool) {
	for _, r := range rows {
		if r.Project == p {
			return r, true
		}
	}
	return TaskRow{}, false
}

func TestComputeTaskRowsFullPath(t *testing.T) {
	builds := []store.TaskBuildSummaryRow{
		{ProjectID: "forge", CommitSha: "c1", Status: "passed", ExitCode: int4(0)},
		{ProjectID: "hsk", CommitSha: "c2", Status: "failed", ExitCode: int4(1)},
	}
	stabs := []store.TaskStabilityRow{
		{ProjectID: "forge", LastDeployStatus: "deployed", HasPostDeployFailure: false},
		{ProjectID: "hsk", LastDeployStatus: "failed", HasPostDeployFailure: true},
	}
	clauseCovered := map[string]bool{"forge": true}

	rows := computeTaskRows(builds, stabs, nil, clauseCovered)

	forge, ok := rowByProject(rows, "forge")
	if !ok {
		t.Fatal("forge 应在摘要行里")
	}
	// 有构建记录 ⟹ 构建记录栏「有记录」(不再写「已合并」);条款被覆盖 ⟹ 「有覆盖」(不再写「通过」)。
	if forge.Merged != "有记录" || forge.Built != "通过" || forge.Tested != "有覆盖" ||
		forge.Deployed != "成功" || forge.Stable != "稳定" {
		t.Errorf("forge 五栏不对: %+v", forge)
	}

	hsk, ok := rowByProject(rows, "hsk")
	if !ok {
		t.Fatal("hsk 应在摘要行里")
	}
	if hsk.Merged != "有记录" || hsk.Built != "失败" || hsk.Tested != "无记录" ||
		hsk.Deployed != "失败" || hsk.Stable != "有复发" {
		t.Errorf("hsk 五栏不对: %+v", hsk)
	}
}

func TestComputeTaskRowsRequirementOnlyProject(t *testing.T) {
	// 只有需求层、还没构建/部署的项目:除条款覆盖栏外全「无记录」。
	rows := computeTaskRows(nil, nil, []string{"idea"}, map[string]bool{"idea": true})
	idea, ok := rowByProject(rows, "idea")
	if !ok {
		t.Fatal("仅有需求层的项目也应进表(已立项未动工)")
	}
	if idea.Merged != "无记录" || idea.Built != "无记录" || idea.Deployed != "无记录" || idea.Stable != "无记录" {
		t.Errorf("无执行记录应为「无记录」: %+v", idea)
	}
	if idea.Tested != "有覆盖" {
		t.Errorf("条款被测试引用应为「有覆盖」: %+v", idea)
	}
}

func TestComputeTaskRowsNoRequirementNoCoverage(t *testing.T) {
	// 有构建但无需求层:条款覆盖栏落「无记录」(没有覆盖的机器记录)。
	builds := []store.TaskBuildSummaryRow{{ProjectID: "solo", Status: "passed", ExitCode: int4(0)}}
	rows := computeTaskRows(builds, nil, nil, nil)
	solo, _ := rowByProject(rows, "solo")
	if solo.Tested != "无记录" {
		t.Errorf("无需求层条款覆盖栏应「无记录」: %+v", solo)
	}
}

func TestComputeTaskRowsSortedByProject(t *testing.T) {
	builds := []store.TaskBuildSummaryRow{
		{ProjectID: "zeta", Status: "passed", ExitCode: int4(0)},
		{ProjectID: "alpha", Status: "passed", ExitCode: int4(0)},
		{ProjectID: "mid", Status: "passed", ExitCode: int4(0)},
	}
	rows := computeTaskRows(builds, nil, nil, nil)
	if len(rows) != 3 || rows[0].Project != "alpha" || rows[1].Project != "mid" || rows[2].Project != "zeta" {
		t.Errorf("摘要行应按项目名排序: %+v", rows)
	}
}

// TestTaskRowsWordingRetiresFalseGreen 钉死 P0-22 的措辞边界(P0-22 AC2):
// 摘要行是执行/项目摘要,任何推导路径都不得再产出可被误读为
// Task 完成 / VERIFIED / APPROVED 的旧措辞 —— 合并栏不得出现「已合并」,
// 条款覆盖栏不得出现「通过」;取值只能来自记录式词表。
func TestTaskRowsWordingRetiresFalseGreen(t *testing.T) {
	builds := []store.TaskBuildSummaryRow{
		{ProjectID: "full", CommitSha: "c1", Status: "passed", ExitCode: int4(0)},       // 全链路通过
		{ProjectID: "broken", CommitSha: "c2", Status: "failed", ExitCode: int4(1)},     // 构建/部署失败
		{ProjectID: "ghost", CommitSha: "c3", Status: "weird", ExitCode: pgtype.Int4{}}, // 状态不认得
	}
	stabs := []store.TaskStabilityRow{
		{ProjectID: "full", LastDeployStatus: "deployed", HasPostDeployFailure: false},
		{ProjectID: "broken", LastDeployStatus: "failed", HasPostDeployFailure: true},
		{ProjectID: "ghost", LastDeployStatus: "mystery", HasPostDeployFailure: false},
	}
	rows := computeTaskRows(builds, stabs, []string{"idea"}, map[string]bool{"full": true, "idea": true})

	// 各栏允许的取值词表:只描述「存在什么记录」,不描述完成判定。
	allowedMerged := map[string]bool{"有记录": true, "无记录": true}
	allowedTested := map[string]bool{"有覆盖": true, "无记录": true}
	allowedBuilt := map[string]bool{"通过": true, "失败": true, "无记录": true}
	allowedDeployed := map[string]bool{"成功": true, "失败": true, "无记录": true}
	allowedStable := map[string]bool{"稳定": true, "有复发": true, "无记录": true}

	for _, r := range rows {
		if !allowedMerged[r.Merged] {
			t.Errorf("项目 %s 构建记录栏出现词表外的值 %q(不得再宣称「已合并」)", r.Project, r.Merged)
		}
		if !allowedTested[r.Tested] {
			t.Errorf("项目 %s 条款覆盖栏出现词表外的值 %q(不得再宣称「测试通过」)", r.Project, r.Tested)
		}
		if !allowedBuilt[r.Built] {
			t.Errorf("项目 %s 末次构建栏出现词表外的值 %q", r.Project, r.Built)
		}
		if !allowedDeployed[r.Deployed] {
			t.Errorf("项目 %s 末次部署栏出现词表外的值 %q", r.Project, r.Deployed)
		}
		if !allowedStable[r.Stable] {
			t.Errorf("项目 %s 部署后栏出现词表外的值 %q", r.Project, r.Stable)
		}
	}

	// 就是最顺利的全链路项目,也只是「有记录 / 有覆盖」,不是完成判定。
	full, ok := rowByProject(rows, "full")
	if !ok {
		t.Fatal("full 应在摘要行里")
	}
	if full.Merged != "有记录" || full.Tested != "有覆盖" {
		t.Errorf("全链路项目应为记录式表述(有记录/有覆盖): %+v", full)
	}
}

func TestBuildOutcome(t *testing.T) {
	tests := []struct {
		name   string
		status string
		exit   pgtype.Int4
		want   string
	}{
		{"退出码 0", "whatever", int4(0), "通过"},
		{"退出码非 0", "whatever", int4(2), "失败"},
		{"无退出码 状态 passed", "passed", pgtype.Int4{}, "通过"},
		{"无退出码 状态 failed", "failed", pgtype.Int4{}, "失败"},
		{"无退出码 状态不认得", "weird", pgtype.Int4{}, "无记录"},
	}
	for _, tt := range tests {
		if got := buildOutcome(tt.status, tt.exit); got != tt.want {
			t.Errorf("%s: buildOutcome(%q) = %q, want %q", tt.name, tt.status, got, tt.want)
		}
	}
}

func TestDeployOutcome(t *testing.T) {
	tests := []struct{ status, want string }{
		{"deployed", "成功"},
		{"rolled_back", "成功"},
		{"failed", "失败"},
		{"rollback_failed", "失败"},
		{"unknown", "无记录"},
	}
	for _, tt := range tests {
		if got := deployOutcome(tt.status); got != tt.want {
			t.Errorf("deployOutcome(%q) = %q, want %q", tt.status, got, tt.want)
		}
	}
}
