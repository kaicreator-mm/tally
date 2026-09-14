package handlers_test

// P3-09 的 Attention 投影页测试(PRD §26.4)。
//
// 两层,与 v14_launch_test.go 同一模式:
//   - 纯 HTTP 层(不需要数据库):数据源未接入空状态;
//   - 全链路层(临时 Git 制品仓库 + 独立 schema 的真实 Postgres,不可达即
//     SKIP):主线推进中的任务零条目(Attention 不是任务列表);终态失败
//     Run 与在账暴露事实各产出一条目(只含合法阻断来源,AC4);边界声明
//     (dismiss/snooze 未接线、生命周期状态不复制,AC5)可见。
//
// 全部断言只针对派生投影:页面不落库、没有可写状态。

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"tally/internal/core"
	"tally/internal/evidence"
	"tally/internal/measurement"
	"tally/internal/runstore"
)

func TestV14AttentionUnavailableWithoutSources(t *testing.T) {
	srv := v14NewServer(t, nil, "", nil)
	status, body := v13Get(t, srv.URL, "/v14/attention")
	if status != http.StatusOK {
		t.Fatalf("GET /v14/attention = %d, want 200", status)
	}
	if !strings.Contains(body, "未接入") {
		t.Errorf("无数据源时应渲染未接入空状态")
	}
}

func TestV14AttentionListsBlockingFactsOnly(t *testing.T) {
	ctx := context.Background()
	st := v13TestDB(t)
	repo, base, docs, code := v13Repo(t)
	frozenHash := v13FrozenHash(t, repo, base)
	rs := runstore.New(st.Queries())

	// 干净主线(docs/code 候选 + 验证 Run,VALIDATING 推进中):零条目 ——
	// Attention 只显示真正需要人的事项,不是任务列表。
	_, codeCS, _ := v13Seed(t, ctx, st.Queries(), repo, base, docs, code)
	srv := v14NewServer(t, st.Queries(), repo, nil)

	status, body := v13Get(t, srv.URL, "/v14/attention")
	if status != http.StatusOK {
		t.Fatalf("GET /v14/attention = %d, want 200", status)
	}
	for _, want := range []string{
		"扫描 1 个 standard TaskContract", // HEAD 快照只有 TASK-0001
		"当前没有需要人工介入的事项",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("干净主线应渲染无事项空状态:%q,body:\n%s", want, body)
		}
	}

	// 合法阻断来源 1:最近一次实现 Run 终态失败(晚于全部既有事实 → 派生
	// FAILED;失败原因是观测值,原样呈现)。
	runF, err := rs.StartCodegenRun(ctx, runstore.StartRunSpec{
		Project: "demo-proj", Task: "TASK-0001", Role: "coding",
		Runtime: "fake", RuntimeVersion: "v0", Model: "demo-model", BehaviorRevision: "b1",
		CostPlane: measurement.CostPlaneExecution, BaseCommit: base,
		TaskRevision: "sha256:taskrev", VerificationRevision: frozenHash, ContextPackHash: "sha256:cp",
		WorkspaceID:               "ws-p309-attention",
		RuntimeCapabilitySnapshot: []byte(`{"runtime":"fake"}`),
		StartedAt:                 time.Now().Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("StartCodegenRun(failed): %v", err)
	}
	if _, err := rs.FinishCodegenRun(ctx, runstore.FinishRunSpec{
		ID: runF, Status: runstore.RunFailed, Iterations: 2,
		FailureReason: "iterations_exhausted", FinishedAt: time.Now().Add(3 * time.Minute),
	}); err != nil {
		t.Fatalf("FinishCodegenRun(failed): %v", err)
	}

	// 合法阻断来源 2:在账暴露事实(exposure_compromise 的账面形态)。
	if _, err := rs.RecordExposure(ctx, runstore.ExposureRecord{
		Task:             "TASK-0001",
		ContractRevision: 1,
		Binding: evidence.BindingRef{
			Project:      core.ProjectID("demo-proj"),
			Verification: evidence.VerificationRevision(frozenHash),
			BindingID:    "EV-AC01-01",
		},
		Acceptance: evidence.AcceptanceID("AC-01"),
		ChangeSet:  evidence.ChangeSetID(codeCS),
		Channel:    evidence.ExposureChannel("repair_workspace"),
		Reason:     "P3-09 attention 投影测试",
		ExposedAt:  time.Now().Add(time.Minute),
	}); err != nil {
		t.Fatalf("RecordExposure: %v", err)
	}

	status, body = v13Get(t, srv.URL, "/v14/attention")
	if status != http.StatusOK {
		t.Fatalf("GET /v14/attention(注入后)= %d, want 200", status)
	}
	for _, want := range []string{
		"实现 Run 终态失败",                 // failed_run 条目
		"iterations_exhausted",        // 失败原因原样呈现
		"暴露事实在账",                      // exposure_compromise 条目
		"exposure_compromise",         // 机器词面
		"exposure:",                   // 依据引用落到暴露事实 id
		"fact:v13art:task_planned",    // 派生条目携带生命周期事实引用
		`href="/v14/tasks/TASK-0001"`, // 链到闭环任务详情
		// 边界声明(AC4/AC5):合法来源、投影不落库、dismiss/snooze 未接线。
		"只包含合法阻断来源",
		"repair_exhausted",
		"投影",
		"生命周期状态不从本页复制",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("Attention 应包含 %q", want)
		}
	}
	// 无中生有的来源不得出现:没有 review 请求事实 → 没有 needs_review 条目;
	// 投影没有失败 → 没有 projection_error 条目(条目标签「投影失败」;边界
	// 声明里对词面本身的提及不算条目)。
	for _, banned := range []string{"需要人工评审", "投影失败"} {
		if strings.Contains(body, banned) {
			t.Errorf("Attention 不应包含未发生的来源 %q", banned)
		}
	}
}

func TestV14AttentionReviewItemsCarryReasonBasisAndAction(t *testing.T) {
	// P2-04 AC:每条评审项必须同时携带 原因(Detail)/ 依据(Refs)/ 目标动作
	// (Action);页面聚合小计按封闭类型词表统计非零类别。投影测试三面:
	// no-review(干净主线零条目、零动作字面)/ exception(终态失败 Run)/
	// high-risk(在账暴露事实)。
	ctx := context.Background()
	st := v13TestDB(t)
	repo, base, docs, code := v13Repo(t)
	frozenHash := v13FrozenHash(t, repo, base)
	rs := runstore.New(st.Queries())

	_, codeCS, _ := v13Seed(t, ctx, st.Queries(), repo, base, docs, code)
	srv := v14NewServer(t, st.Queries(), repo, nil)

	// no-review 面:干净主线 → 空状态,且不得出现任何目标动作字面(动作只
	// 随条目出现,不预渲染、不无中生有)。
	status, body := v13Get(t, srv.URL, "/v14/attention")
	if status != http.StatusOK {
		t.Fatalf("GET /v14/attention = %d, want 200", status)
	}
	for _, banned := range []string{
		"目标动作",        // 动作列随条目渲染,空状态无表头
		"人工裁决后续",      // failed_run 动作
		"人工裁决 strict", // exposure 动作
		"项需人工事项",      // 聚合小计行随条目渲染
	} {
		if strings.Contains(body, banned) {
			t.Errorf("干净主线不应出现动作/聚合字面 %q", banned)
		}
	}

	// exception 面:最近一次实现 Run 终态失败(失败原因原样呈现)。
	runF, err := rs.StartCodegenRun(ctx, runstore.StartRunSpec{
		Project: "demo-proj", Task: "TASK-0001", Role: "coding",
		Runtime: "fake", RuntimeVersion: "v0", Model: "demo-model", BehaviorRevision: "b1",
		CostPlane: measurement.CostPlaneExecution, BaseCommit: base,
		TaskRevision: "sha256:taskrev", VerificationRevision: frozenHash, ContextPackHash: "sha256:cp",
		WorkspaceID:               "ws-p204-review-exception",
		RuntimeCapabilitySnapshot: []byte(`{"runtime":"fake"}`),
		StartedAt:                 time.Now().Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("StartCodegenRun(failed): %v", err)
	}
	if _, err := rs.FinishCodegenRun(ctx, runstore.FinishRunSpec{
		ID: runF, Status: runstore.RunFailed, Iterations: 3,
		FailureReason: "assurance_insufficient", FinishedAt: time.Now().Add(3 * time.Minute),
	}); err != nil {
		t.Fatalf("FinishCodegenRun(failed): %v", err)
	}

	// high-risk 面:在账暴露事实(strict 主张需人工裁决)。
	if _, err := rs.RecordExposure(ctx, runstore.ExposureRecord{
		Task:             "TASK-0001",
		ContractRevision: 1,
		Binding: evidence.BindingRef{
			Project:      core.ProjectID("demo-proj"),
			Verification: evidence.VerificationRevision(frozenHash),
			BindingID:    "EV-AC01-01",
		},
		Acceptance: evidence.AcceptanceID("AC-01"),
		ChangeSet:  evidence.ChangeSetID(codeCS),
		Channel:    evidence.ExposureChannel("repair_workspace"),
		Reason:     "P2-04 评审投影测试",
		ExposedAt:  time.Now().Add(time.Minute),
	}); err != nil {
		t.Fatalf("RecordExposure: %v", err)
	}

	status, body = v13Get(t, srv.URL, "/v14/attention")
	if status != http.StatusOK {
		t.Fatalf("GET /v14/attention(注入后)= %d, want 200", status)
	}
	for _, want := range []string{
		"目标动作", // 动作列头
		"人工裁决后续:修复重试 / 换基线 / 放弃(不自动重试)。",    // failed_run 的固定目标动作
		"人工裁决 strict 主张:waiver / 换基线 / 放弃。", // exposure 的固定目标动作
		"聚合 2 项需人工事项",                       // 聚合小计
		"实现 Run 终态失败 ×1",                    // 小计按封闭类型词表计数
		"暴露事实在账 ×1",
		"fact:v13art:task_planned", // 依据引用仍在(原因/依据/动作三者并存)
		"exposure:",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("评审投影应包含 %q,body:\n%s", want, body)
		}
	}
	// 未发生的类别不得出现在小计或条目中(封闭词表,无中生有即违规)。
	for _, banned := range []string{"需要人工评审 ×", "被显式阻塞 ×", "投影失败 ×"} {
		if strings.Contains(body, banned) {
			t.Errorf("未发生的类别不应出现在聚合小计 %q", banned)
		}
	}
}
