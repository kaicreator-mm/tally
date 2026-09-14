package ingest

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"tally/internal/logpipe"
	"tally/internal/logpipe/toolchain"
)

// 单测离线跑:校验类错误都在碰库之前返回,store 传 nil 也安全
// (failure.RecordEvent 的来源外键计数检查同样先于数据库)。
// 真 Postgres 的接线在 ingest_integration_test.go(-tags=integration)。

// TestRecordFailureValidatesInput:Input 校验与来源唯一性(表驱动)。
func TestRecordFailureValidatesInput(t *testing.T) {
	at := time.Date(2026, 7, 28, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		in      Input
		wantSub string
	}{
		{
			name:    "缺 project_id",
			in:      Input{Source: "build", BuildRunID: 1, OccurredAt: at},
			wantSub: "project_id",
		},
		{
			name:    "未知来源",
			in:      Input{ProjectID: "demo", Source: "ci", BuildRunID: 1, OccurredAt: at},
			wantSub: "未知来源",
		},
		{
			name:    "零个来源外键",
			in:      Input{ProjectID: "demo", Source: "build", OccurredAt: at, Raw: "x"},
			wantSub: "恰好一个非空",
		},
		{
			name: "两个来源外键",
			in: Input{
				ProjectID: "demo", Source: "build",
				BuildRunID: 1, DeployRunID: 2, // 两个 → 拒绝
				OccurredAt: at, Raw: "x",
			},
			wantSub: "恰好一个非空",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := RecordFailure(context.Background(), nil, tt.in)
			if err == nil {
				t.Fatal("应报错")
			}
			if !strings.Contains(err.Error(), tt.wantSub) {
				t.Errorf("错误应含 %q,得到 %q", tt.wantSub, err.Error())
			}
		})
	}
}

// TestProcessSelectsExtractor:按名选工具链抽取器,空 / unknown 退回通用抽取;确定性。
func TestProcessSelectsExtractor(t *testing.T) {
	raw, err := os.ReadFile("../logpipe/testdata/go/compile-error.log")
	if err != nil {
		t.Skipf("缺 logpipe 测试样本: %v", err)
	}
	s := string(raw)

	e, ok := toolchain.ByName("go")
	if !ok {
		t.Fatal("go 抽取器应已注册")
	}
	if got := process(s, "go"); got != logpipe.ProcessWith(s, e.Extract) {
		t.Error("toolchain=go 应走 go 抽取器,与 logpipe.ProcessWith 同产出")
	}
	generic := logpipe.Process(s)
	if got := process(s, ""); got != generic {
		t.Error("空工具链应退回通用抽取")
	}
	if got := process(s, "不存在的工具链"); got != generic {
		t.Error("未知工具链应退回通用抽取")
	}
	// 确定性(INV-8):同输入 byte 级相同输出。
	if a, b := process(s, "go"), process(s, "go"); a != b {
		t.Error("同输入应产出相同指纹")
	}
	if generic.Exact == "" || process(s, "go").Exact == "" {
		t.Error("指纹不应为空")
	}
}

// TestIngestBuildValidates:构建录入的前置校验,都在写库之前。
func TestIngestBuildValidates(t *testing.T) {
	at := time.Date(2026, 7, 28, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		in      BuildInput
		wantSub string
	}{
		{
			name:    "缺 project_id",
			in:      BuildInput{CommitSha: "abc", Status: StatusFailed, StartedAt: at},
			wantSub: "project_id",
		},
		{
			name:    "缺 commit_sha",
			in:      BuildInput{Input: Input{ProjectID: "demo"}, Status: StatusFailed, StartedAt: at},
			wantSub: "commit_sha",
		},
		{
			name:    "status 非法",
			in:      BuildInput{Input: Input{ProjectID: "demo"}, CommitSha: "abc", Status: "ok", StartedAt: at},
			wantSub: "passed/failed",
		},
		{
			name:    "缺 started_at",
			in:      BuildInput{Input: Input{ProjectID: "demo"}, CommitSha: "abc", Status: StatusFailed},
			wantSub: "started_at",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := IngestBuild(context.Background(), nil, tt.in)
			if err == nil {
				t.Fatal("应报错")
			}
			if !strings.Contains(err.Error(), tt.wantSub) {
				t.Errorf("错误应含 %q,得到 %q", tt.wantSub, err.Error())
			}
		})
	}
}
