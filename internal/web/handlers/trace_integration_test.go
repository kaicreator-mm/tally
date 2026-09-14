//go:build integration

package handlers_test

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"tally/internal/command"
	"tally/internal/core"
	"tally/internal/failure"
	"tally/internal/jobs"
	"tally/internal/knowledge"
	"tally/internal/store"
)

/* ---------- T05c 集成:任务算出状态、视图 A/B、覆盖率、知识库、重诊断(真 Postgres + 真 git) ----------
   每测试 store.TestDB 拿独立 schema。项目名 / 指纹仍用每次运行唯一值,只为让页面文本断言
   能锁定本测试种的那一行(同库内还有别的种子数据),不是防它包并行污染。 */

// uniqueProject 生成本次运行唯一的项目名,让页面文本断言锁定本测试种的那行。
func uniqueProject(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// countJobsFor 数某 kind 且 payload 含给定串(项目名/指纹)的 job 数(payload 自带坐标)。
func countJobsFor(t *testing.T, st *store.Store, kind, needle string) int64 {
	t.Helper()
	var n int64
	if err := st.Pool().QueryRow(context.Background(),
		"SELECT count(*) FROM job WHERE kind = $1 AND strpos(payload::text, $2) > 0",
		kind, needle).Scan(&n); err != nil {
		t.Fatalf("数 job(%s, %s): %v", kind, needle, err)
	}
	return n
}

// greenRunner 是四道闸用的确定性执行器:G1 基线复现(前 reproRuns 次)全红,其后 G2/G3 全绿。
type greenRunner struct{ calls int }

func (r *greenRunner) Run(_ context.Context, _ failure.RunSpec) (failure.RunResult, error) {
	r.calls++
	if r.calls <= 5 { // reproRuns = 5:G1 基线复现全红(稳定复现)
		return failure.RunResult{Green: false, Output: "red"}, nil
	}
	return failure.RunResult{Green: true, Output: "green"}, nil // G2/G3:绿
}

// makeProof 经 failure.Verify 产一个真实有效证据(假 Runner:G1 红、其后全绿)。
func makeProof(t *testing.T) failure.Proof {
	t.Helper()
	proof, err := failure.Verify(context.Background(), &greenRunner{}, failure.VerifyInput{
		ReproCommand: command.CommandSpec{Argv: []string{"go", "test", "./..."}, Cwd: t.TempDir()},
		Candidate: failure.Candidate{
			Summary: "升级 node 到 22",
			Changes: []core.FileChange{{Path: "Dockerfile", Content: "FROM node:22"}},
		},
		Combo: core.VersionCombo{"node": "22.3.0"},
	})
	if err != nil {
		t.Fatalf("产验证证据: %v", err)
	}
	return proof
}

// seedBuildAndDeploy 种一组「构建 + 部署 + 可选部署后失败」的最小事实链。
func seedBuildAndDeploy(t *testing.T, q *store.Queries, project string, exit int32, depStatus string, postFail bool) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	buildStatus := "passed"
	if exit != 0 {
		buildStatus = "failed"
	}
	if _, err := q.InsertBuildRun(ctx, store.InsertBuildRunParams{
		ProjectID: project, CommitSha: "commit-" + project, Status: buildStatus,
		StartedAt:    pgtype.Timestamptz{Time: now.Add(-3 * time.Hour), Valid: true},
		ExitCode:     pgtype.Int4{Int32: exit, Valid: true},
		VersionCombo: []byte(`{"go":"1.23"}`),
	}); err != nil {
		t.Fatalf("种 build_run: %v", err)
	}
	depID, err := q.InsertDeployRun(ctx, store.InsertDeployRunParams{
		ProjectID: project, ArtifactDigest: "sha256:" + project, TargetEnv: "prod", Tool: "compose",
		Status:       depStatus,
		StartedAt:    pgtype.Timestamptz{Time: now.Add(-2 * time.Hour), Valid: true},
		VersionCombo: []byte(`{"go":"1.23"}`),
	})
	if err != nil {
		t.Fatalf("种 deploy_run: %v", err)
	}
	if postFail {
		fp := "2b" + fmt.Sprintf("%010x", now.UnixNano()&0xffffffffffff)
		if _, err := q.InsertFailureEvent(ctx, store.InsertFailureEventParams{
			ProjectID: project, Source: "deploy", DeployRunID: pgtype.Int8{Int64: depID, Valid: true},
			Excerpt: pgtype.Text{String: "健康检查超时", Valid: true}, Normalized: pgtype.Text{String: "健康检查超时", Valid: true},
			FpExact: fp, FpFuzzy: "fz-" + fp, PipelineVersion: "v4",
			OccurredAt: pgtype.Timestamptz{Time: now.Add(-time.Hour), Valid: true}, VersionCombo: []byte(`{"go":"1.23"}`),
			OtherTargetCommands: []byte("[]"),
			ReproCommand:        []byte("null"),
		}); err != nil {
			t.Fatalf("种 failure_event: %v", err)
		}
	}
}

// 任务页从真执行链算出五栏(INV-3),且无任何手动状态控件。
func TestTaskStatusComputedFromRealChain(t *testing.T) {
	st := openExecStore(t)
	srv, _, client := newExecServerReal(t, st)
	project := uniqueProject("task-ok")
	failProj := uniqueProject("task-bad")

	seedBuildAndDeploy(t, st.Queries(), project, 0, "deployed", false) // 通过/成功/稳定
	seedBuildAndDeploy(t, st.Queries(), failProj, 1, "failed", true)   // 失败/有复发

	resp, err := client.Get(srv.URL + "/tasks")
	if err != nil {
		t.Fatalf("GET /tasks: %v", err)
	}
	body := bodyString(t, resp)

	okSeg := segmentForProject(body, project)
	if !strings.Contains(okSeg, "有记录") || !strings.Contains(okSeg, "通过") ||
		!strings.Contains(okSeg, "成功") || !strings.Contains(okSeg, "稳定") {
		t.Errorf("%s 应为 有记录/通过/成功/稳定: %s", project, okSeg)
	}
	badSeg := segmentForProject(body, failProj)
	if !strings.Contains(badSeg, "失败") || !strings.Contains(badSeg, "有复发") {
		t.Errorf("%s 应为 失败/有复发: %s", failProj, badSeg)
	}
	for _, banned := range []string{`type="checkbox"`, `<select`, `name="status"`, `type="radio"`} {
		if strings.Contains(body, banned) {
			t.Errorf("任务页不应出现状态控件 %q(INV-3)", banned)
		}
	}
}

// 视图 B:无需求层时关联条款列为空,但影响面照常(LEFT JOIN)。
func TestViewBLeftJoinWithoutRequirementLayer(t *testing.T) {
	st := openExecStore(t)
	srv, _, client := newExecServerReal(t, st)
	ctx := context.Background()
	q := st.Queries()
	project := uniqueProject("viewb")
	fp := "0f" + fmt.Sprintf("%010x", time.Now().UnixNano()&0xffffffffffff)

	// 只种指纹索引,不种任何需求层 → LEFT JOIN 无匹配。
	if _, err := q.UpsertFingerprintStat(ctx, store.UpsertFingerprintStatParams{
		FpExact: fp, FpFuzzy: "fz-" + fp,
		FirstSeen: pgtype.Timestamptz{Time: time.Now().Add(-time.Hour).UTC(), Valid: true}, Project: project,
	}); err != nil {
		t.Fatalf("种 fingerprint_stat: %v", err)
	}

	resp, err := client.Get(srv.URL + "/views/b?fp=" + fp)
	if err != nil {
		t.Fatalf("GET /views/b: %v", err)
	}
	body := bodyString(t, resp)
	if strings.Contains(body, "还没有索引进库") {
		t.Fatalf("已种索引的指纹不应落空状态")
	}
	if !strings.Contains(body, "影响面") || !strings.Contains(body, project) {
		t.Errorf("视图 B 无需求层时影响面应照常(含影响面与项目)")
	}
	if !strings.Contains(body, "关联条款") || !strings.Contains(body, "没有需求层") {
		t.Errorf("视图 B 应渲染关联条款区并说明无需求层(LEFT JOIN)")
	}
}

// 视图 A:种需求层 + 失败链,断言五个字段都出来。
func TestViewAFieldsFromRealChain(t *testing.T) {
	st := openExecStore(t)
	srv, _, client := newExecServerReal(t, st)
	ctx := context.Background()
	q := st.Queries()
	now := time.Now().UTC()
	project := uniqueProject("viewa")
	clauseID := "REQ-1"

	hashV1 := "hash-" + project + "-v1"
	hashV2 := "hash-" + project + "-v2"
	if err := q.InsertClause(ctx, store.InsertClauseParams{ProjectID: project, ClauseID: clauseID}); err != nil {
		t.Fatalf("种 clause: %v", err)
	}
	for i, h := range []string{hashV1, hashV2} {
		if err := q.InsertClauseVersion(ctx, store.InsertClauseVersionParams{
			ProjectID: project, ClauseID: clauseID, ContentHash: h, VersionNo: int32(i + 1),
			SignedAt:  pgtype.Timestamptz{Time: now.Add(-time.Duration(10-i) * 24 * time.Hour), Valid: true},
			IsCurrent: i == 1,
		}); err != nil {
			t.Fatalf("种 clause_version: %v", err)
		}
	}
	commitSha := "commit-" + project
	if err := q.InsertCommitClause(ctx, store.InsertCommitClauseParams{ProjectID: project, CommitSha: commitSha, ClauseContentHash: hashV1}); err != nil {
		t.Fatalf("种 commit_clause: %v", err)
	}
	buildID, err := q.InsertBuildRun(ctx, store.InsertBuildRunParams{
		ProjectID: project, CommitSha: commitSha, Status: "failed",
		StartedAt:    pgtype.Timestamptz{Time: now.Add(-time.Hour), Valid: true},
		ExitCode:     pgtype.Int4{Int32: 1, Valid: true},
		VersionCombo: []byte(`{"go":"1.23"}`),
	})
	if err != nil {
		t.Fatalf("种 build_run: %v", err)
	}
	fp := "1a" + fmt.Sprintf("%010x", time.Now().UnixNano()&0xffffffffffff)
	for i := 0; i < 2; i++ {
		if _, err := q.InsertFailureEvent(ctx, store.InsertFailureEventParams{
			ProjectID: project, Source: "build", BuildRunID: pgtype.Int8{Int64: buildID, Valid: true},
			Excerpt: pgtype.Text{String: "编译错误", Valid: true}, Normalized: pgtype.Text{String: "编译错误", Valid: true},
			FpExact: fp, FpFuzzy: "fz-" + fp, PipelineVersion: "v4",
			OccurredAt: pgtype.Timestamptz{Time: now.Add(-time.Duration(i+1) * time.Minute), Valid: true}, VersionCombo: []byte(`{"go":"1.23"}`),
			OtherTargetCommands: []byte("[]"),
			ReproCommand:        []byte("null"),
		}); err != nil {
			t.Fatalf("种 failure_event: %v", err)
		}
	}

	resp, err := client.Get(srv.URL + "/views/a?project=" + project + "&clause=" + clauseID)
	if err != nil {
		t.Fatalf("GET /views/a: %v", err)
	}
	body := bodyString(t, resp)
	if strings.Contains(body, "找不到这个条款") {
		t.Fatalf("已种条款不应落空状态")
	}
	for _, label := range []string{"实现提交", "构建失败", "部署失败", "同一指纹", "变更次数", "条款年龄"} {
		if !strings.Contains(body, label) {
			t.Errorf("视图 A 字段标签应含 %q", label)
		}
	}
	if !strings.Contains(body, ">1<") {
		t.Errorf("实现提交数应为 1")
	}
	if !strings.Contains(body, fp) {
		t.Errorf("失败构成应列出指纹 %s", fp)
	}
}

// 覆盖率 + 依据已变更(INV-4):条款改动后,引用旧哈希的下游挂钩标「依据已变更」。
func TestCoverageChangedReferenceMarked(t *testing.T) {
	st := openExecStore(t)
	srv, _, client := newExecServerReal(t, st)
	ctx := context.Background()
	q := st.Queries()
	now := time.Now().UTC()
	project := uniqueProject("cov")
	clauseID := "REQ-9"

	hashV1 := "hash-" + project + "-v1"
	hashV2 := "hash-" + project + "-v2"
	if err := q.InsertClause(ctx, store.InsertClauseParams{ProjectID: project, ClauseID: clauseID}); err != nil {
		t.Fatalf("种 clause: %v", err)
	}
	for i, h := range []string{hashV1, hashV2} {
		if err := q.InsertClauseVersion(ctx, store.InsertClauseVersionParams{
			ProjectID: project, ClauseID: clauseID, ContentHash: h, VersionNo: int32(i + 1),
			SignedAt:  pgtype.Timestamptz{Time: now.Add(-time.Duration(5-i) * 24 * time.Hour), Valid: true},
			IsCurrent: i == 1,
		}); err != nil {
			t.Fatalf("种 clause_version: %v", err)
		}
	}
	if err := q.InsertCommitClause(ctx, store.InsertCommitClauseParams{ProjectID: project, CommitSha: "commit-" + project, ClauseContentHash: hashV1}); err != nil {
		t.Fatalf("种 commit_clause: %v", err)
	}
	if err := q.InsertTestClause(ctx, store.InsertTestClauseParams{ProjectID: project, TestName: "test/" + project, ClauseContentHash: hashV2}); err != nil {
		t.Fatalf("种 test_clause: %v", err)
	}

	resp, err := client.Get(srv.URL + "/coverage?project=" + project)
	if err != nil {
		t.Fatalf("GET /coverage: %v", err)
	}
	body := bodyString(t, resp)
	if !strings.Contains(body, "100%") {
		t.Errorf("条款被测试 @covers 当前版,覆盖率应 100%%")
	}
	if !strings.Contains(body, "依据已变更") {
		t.Errorf("引用旧版哈希的 commit 挂钩应标「依据已变更」(INV-4)")
	}
	if !strings.Contains(body, clauseID) {
		t.Errorf("依据已变更区应列出条款 %s", clauseID)
	}
}

// 知识库浏览:从真 git 库读已验证修复(只读),无编辑入口。
func TestKnowledgeBrowsesRealGitLibrary(t *testing.T) {
	st := openExecStore(t)
	srv, repo, client := newExecServerReal(t, st)
	ctx := context.Background()

	fp := "abcd1234" + fmt.Sprintf("%08x", time.Now().UnixNano()&0xffffffff)
	ks := knowledge.NewStore(repo, st.Queries())
	fix := core.Fix{
		Repro:   "go test ./...",
		Summary: "升级 node 到 22",
		Changes: []core.FileChange{{Path: "Dockerfile", Content: "FROM node:22"}},
	}
	if _, err := ks.Store(ctx, core.Fingerprint(fp), fix, makeProof(t)); err != nil {
		t.Fatalf("种已验证修复: %v", err)
	}

	resp, err := client.Get(srv.URL + "/knowledge")
	if err != nil {
		t.Fatalf("GET /knowledge: %v", err)
	}
	body := bodyString(t, resp)
	if !strings.Contains(body, "badge-verified") || !strings.Contains(body, "已验证") {
		t.Errorf("验证库页应带 badge-verified「已验证」标")
	}
	if !strings.Contains(body, fp[:8]) {
		t.Errorf("验证库应列出条目 %s", fp)
	}
	if !strings.Contains(body, "go test ./...") {
		t.Errorf("验证库应展示复现命令")
	}
	if idx := strings.Index(body, "<main>"); idx >= 0 && strings.Contains(body[idx:], `<form`) {
		t.Errorf("验证库内容区不应出现表单(INV-2)")
	}
}

// 重诊断入队走真队列 + 三轮上限由后端守(真索引行)。
func TestDiagnoseTriggerRealQueueAndBackendCap(t *testing.T) {
	st := openExecStore(t)
	srv, _, client := newExecServerReal(t, st)
	ctx := context.Background()
	q := st.Queries()
	project := uniqueProject("diag")

	fpCap := "dc" + fmt.Sprintf("%010x", time.Now().UnixNano()&0xffffffffffff)
	if _, err := q.UpsertFingerprintStat(ctx, store.UpsertFingerprintStatParams{
		FpExact: fpCap, FpFuzzy: "fz-" + fpCap,
		FirstSeen: pgtype.Timestamptz{Time: time.Now().Add(-time.Hour).UTC(), Valid: true}, Project: project,
	}); err != nil {
		t.Fatalf("种 fingerprint_stat: %v", err)
	}
	for i := 0; i < 3; i++ {
		if err := q.BumpDiagnoseAttempts(ctx, fpCap); err != nil {
			t.Fatalf("累加诊断次数: %v", err)
		}
	}

	jobBefore := countJobsFor(t, st, string(jobs.KindDiagnose), fpCap)
	resp := postForm(t, client, srv.URL, "/diagnose?fp="+fpCap, "/diagnose", url.Values{"fp": {fpCap}, "project": {project}})
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if !strings.Contains(loc, "kind=warn") {
		t.Errorf("三轮用尽 POST 应带警告重定向,得到 %q", loc)
	}
	if got := countJobsFor(t, st, string(jobs.KindDiagnose), fpCap); got != jobBefore {
		t.Errorf("三轮用尽不应入队: before=%d after=%d(后端没守住)", jobBefore, got)
	}

	fpOk := "d0" + fmt.Sprintf("%010x", time.Now().UnixNano()&0xffffffffffff)
	if _, err := q.UpsertFingerprintStat(ctx, store.UpsertFingerprintStatParams{
		FpExact: fpOk, FpFuzzy: "fz-" + fpOk,
		FirstSeen: pgtype.Timestamptz{Time: time.Now().Add(-time.Hour).UTC(), Valid: true}, Project: project,
	}); err != nil {
		t.Fatalf("种 fingerprint_stat(ok): %v", err)
	}
	okBefore := countJobsFor(t, st, string(jobs.KindDiagnose), fpOk)
	resp = postForm(t, client, srv.URL, "/diagnose?fp="+fpOk, "/diagnose", url.Values{"fp": {fpOk}, "project": {project}})
	loc = resp.Header.Get("Location")
	resp.Body.Close()
	if !strings.Contains(loc, "kind=ok") {
		t.Errorf("有预算 POST 应带「已入队」重定向,得到 %q", loc)
	}
	if got := countJobsFor(t, st, string(jobs.KindDiagnose), fpOk); got != okBefore+1 {
		t.Errorf("应入队一个 diagnose job: before=%d after=%d", okBefore, got)
	}
}

/* ---------- 辅助 ---------- */

// segmentForProject 取页面里某项目名之后的一段(到行尾),断言收窄到本项目用。
func segmentForProject(body, project string) string {
	idx := strings.Index(body, project)
	if idx < 0 {
		return ""
	}
	seg := body[idx:]
	if end := strings.Index(seg, "</tr>"); end > 0 {
		seg = seg[:end]
	}
	return seg
}
