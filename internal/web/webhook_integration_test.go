//go:build integration

package web

// 集成测试:真 Postgres(不 mock 数据库,docs/05-test-standards.md)。
// webhook 走完整 NewRouter(连路由注册一起验):合法请求落 build_run + failure_event,
// 原始日志落盘且 raw_log_ref 指向它;成功构建只落 build_run 不录失败。

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"tally/internal/command"
	"tally/internal/failure"
	"testing"
	"time"

	"tally/internal/jobs"
	"tally/internal/logpipe"
	"tally/internal/logpipe/toolchain"
	"tally/internal/store"
)

const itestWebhookToken = "itest-token"

// webhookTestServer 起带 webhook 配置的完整路由。

// seedProjectIndex 把项目种进派生项目注册表(P0-13 ghost-project guard):新事实
// 只对注册表中的项目放行,webhook 录入测试必须先注册项目。
func seedProjectIndex(t *testing.T, st *store.Store, project string) {
	t.Helper()
	if err := st.Queries().UpsertProjectIndex(context.Background(), store.UpsertProjectIndexParams{
		ProjectID: project,
	}); err != nil {
		t.Fatalf("种 project_index: %v", err)
	}
}

func webhookTestServer(t *testing.T, st *store.Store, logDir string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(NewRouter(Deps{
		WebhookToken: itestWebhookToken,
		LogDir:       logDir,
		IngestStore:  st,
	}))
	t.Cleanup(srv.Close)
	return srv
}

// postWebhook 带 token POST 一个 webhook 请求,返回响应与解码后的结果。
func postWebhook(t *testing.T, srv *httptest.Server, payload buildPayload) (*http.Response, webhookResponse) {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("序列化请求体: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/webhook/build", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("构造请求: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+itestWebhookToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST webhook: %v", err)
	}
	defer resp.Body.Close()
	var out webhookResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("解码响应(状态 %d): %v", resp.StatusCode, err)
	}
	return resp, out
}

// goSampleLog 读 logpipe 的 go 编译失败样本(与 ingest 集成测试同源)。
func goSampleLog(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("../logpipe/testdata/go/compile-error.log")
	if err != nil {
		t.Skipf("缺 logpipe 测试样本: %v", err)
	}
	return string(b)
}

// TestWebhookBuildFailedIntegration:失败构建 → build_run + failure_event 落库,
// 日志文件落盘且 raw_log_ref 指向它;复现上下文与版本组合原样持久化;
// 指纹与直接跑 go 抽取器一致(同源同指纹,INV-8)。
func TestWebhookBuildFailedIntegration(t *testing.T) {
	st := store.TestDB(t)
	ctx := context.Background()
	logDir := t.TempDir()
	srv := webhookTestServer(t, st, logDir)

	project := "wh-itest-" + fmt.Sprintf("%d", time.Now().UnixNano())
	seedProjectIndex(t, st, project)
	raw := goSampleLog(t)
	commit := "abcdef1234567890abcdef1234567890abcdef12"

	resp, out := postWebhook(t, srv, buildPayload{
		Project:      project,
		Commit:       commit,
		Toolchain:    "go",
		Status:       "failed",
		ExitCode:     ptrInt(2),
		CiURL:        "https://ci.example/run/1",
		Combo:        map[string]string{"go": "1.23.0"},
		Workdir:      "/tmp/demo",
		Repro:        "make build",
		OtherTargets: []string{"make other-a", "make other-b"},
		Log:          raw,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("状态码 = %d,想要 200(响应 %+v)", resp.StatusCode, out)
	}
	if out.BuildRun == 0 || out.FailureEvent == 0 {
		t.Fatalf("失败构建应同时返回 build_run 与 failure_event id,得到 %+v", out)
	}

	// 原始日志落盘,raw_log_ref 指向它,内容 byte 级一致(D-d4)。
	if !strings.HasPrefix(out.RawLogRef, logDir) {
		t.Errorf("raw_log_ref = %q,应落在日志目录 %q 下", out.RawLogRef, logDir)
	}
	saved, err := os.ReadFile(out.RawLogRef)
	if err != nil {
		t.Fatalf("读留存日志: %v", err)
	}
	if string(saved) != raw {
		t.Errorf("留存日志与请求日志不一致(长度 %d vs %d)", len(saved), len(raw))
	}

	// failure_event 落库:raw_log_ref 指向留存文件,复现上下文原样持久化。
	events, err := st.Queries().ListFailureEventsWithRaw(ctx)
	if err != nil {
		t.Fatalf("读 failure_event: %v", err)
	}
	var mine []store.ListFailureEventsWithRawRow
	for _, e := range events {
		if e.ProjectID == project {
			mine = append(mine, e)
		}
	}
	if len(mine) != 1 {
		t.Fatalf("failure_event 行数 = %d,想要 1", len(mine))
	}
	if !mine[0].RawLogRef.Valid || mine[0].RawLogRef.String != out.RawLogRef {
		t.Errorf("failure_event.raw_log_ref = %v,想要 %q", mine[0].RawLogRef, out.RawLogRef)
	}
	if mine[0].Source != "build" {
		t.Errorf("failure_event.source = %q,想要 build", mine[0].Source)
	}

	// 指纹与直接跑 go 抽取器一致(录入复用 logpipe,INV-8)。
	ext, ok := toolchain.ByName("go")
	if !ok {
		t.Fatal("toolchain 缺 go 抽取器")
	}
	want := logpipe.ProcessWith(raw, ext.Extract)
	if mine[0].FpExact != string(want.Exact) {
		t.Errorf("fp_exact = %q,想要 %q", mine[0].FpExact, want.Exact)
	}

	// 按指纹装载完整事件:build_run 外键配对,复现上下文三字段齐全(T11b 链路)。
	ev, err := st.Queries().GetLatestFailureEventByFingerprint(ctx, mine[0].FpExact)
	if err != nil {
		t.Fatalf("按指纹读 failure_event: %v", err)
	}
	if !ev.BuildRunID.Valid || ev.BuildRunID.Int64 != out.BuildRun {
		t.Errorf("build_run_id = %v,想要 %d", ev.BuildRunID, out.BuildRun)
	}
	// 0016 双轨合同:新写入走结构化列(repro_command/other_target_commands),
	// 旧文本三列在新行上恒空(仅存量行兼容读取)。
	var repro command.CommandSpec
	if err := json.Unmarshal(ev.ReproCommand, &repro); err != nil {
		t.Fatalf("解 repro_command: %v", err)
	}
	if repro.Cwd != "/tmp/demo" || strings.Join(repro.Argv, " ") != "make build" {
		t.Errorf("复现上下文 = %+v,想要 (cwd=/tmp/demo, argv=make build)", repro)
	}
	if ev.Workdir != "" || ev.Repro != "" {
		t.Errorf("新行旧文本列应恒空: (%q, %q)", ev.Workdir, ev.Repro)
	}
	var targets []failure.OtherTarget
	if err := json.Unmarshal(ev.OtherTargetCommands, &targets); err != nil {
		t.Fatalf("解 other_target_commands: %v", err)
	}
	if len(targets) != 2 || strings.Join(targets[0].Command.Argv, " ") != "make other-a" ||
		strings.Join(targets[1].Command.Argv, " ") != "make other-b" {
		t.Errorf("other_target_commands = %+v,想要 [make other-a make other-b]", targets)
	}

	// build_run 落库:状态/退出码/commit 原样。
	sums, err := st.Queries().TaskBuildSummary(ctx)
	if err != nil {
		t.Fatalf("读 build_run 汇总: %v", err)
	}
	found := false
	for _, s := range sums {
		if s.ProjectID != project {
			continue
		}
		found = true
		if s.Status != "failed" || s.CommitSha != commit || s.ExitCode.Int32 != 2 {
			t.Errorf("build_run = (%q, %q, %v),想要 (failed, %s, 2)", s.Status, s.CommitSha, s.ExitCode, commit)
		}
	}
	if !found {
		t.Errorf("build_run 汇总里没有项目 %s", project)
	}

	// T11e/D-e1:开关默认关 → 失败录入后队里没有 diagnose 任务。
	if n := countDiagnoseJobsFor(t, st, mine[0].FpExact); n != 0 {
		t.Errorf("开关默认关,不应自动入队诊断,有 %d 个", n)
	}
}

// TestWebhookBuildPassedIntegration:成功构建 → 只落 build_run,不录失败事件(D-d5)。
func TestWebhookBuildPassedIntegration(t *testing.T) {
	st := store.TestDB(t)
	ctx := context.Background()
	srv := webhookTestServer(t, st, t.TempDir())

	project := "wh-itest-pass-" + fmt.Sprintf("%d", time.Now().UnixNano())
	resp, out := postWebhook(t, srv, buildPayload{
		Project:  project,
		Commit:   "fedcba0987654321",
		Status:   "passed",
		Artifact: "sha256:deadbeef",
		Log:      "build ok",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("状态码 = %d,想要 200(响应 %+v)", resp.StatusCode, out)
	}
	if out.BuildRun == 0 {
		t.Fatal("成功构建也应返回 build_run id")
	}
	if out.FailureEvent != 0 {
		t.Errorf("成功构建不应录失败事件,得到 failure_event=%d", out.FailureEvent)
	}

	events, err := st.Queries().ListFailureEventsWithRaw(ctx)
	if err != nil {
		t.Fatalf("读 failure_event: %v", err)
	}
	for _, e := range events {
		if e.ProjectID == project {
			t.Errorf("成功构建不应有 failure_event,得到 %+v", e)
		}
	}
}

// TestWebhookBuildAutoDiagnoseEnqueues(T11e):开关打开时,webhook 录入失败构建后
// 按失败指纹自动入队一个 diagnose 任务;成功构建不入队(D-e1/e4)。
func TestWebhookBuildAutoDiagnoseEnqueues(t *testing.T) {
	st := store.TestDB(t)
	ctx := context.Background()
	srv := httptest.NewServer(NewRouter(Deps{
		WebhookToken:  itestWebhookToken,
		LogDir:        t.TempDir(),
		IngestStore:   st,
		DiagnoseAuto:  true,
		DiagnoseQueue: jobs.New(st, nil), // 与手动重诊断同一条入队路径(D-e4)
	}))
	t.Cleanup(srv.Close)

	project := "wh-itest-auto-" + fmt.Sprintf("%d", time.Now().UnixNano())
	seedProjectIndex(t, st, project)
	resp, out := postWebhook(t, srv, buildPayload{
		Project: project, Commit: "abc123456789", Toolchain: "go", Status: "failed", Log: goSampleLog(t),
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("状态码 = %d,想要 200(响应 %+v)", resp.StatusCode, out)
	}

	// 失败指纹以落库行为准;入队的 diagnose payload 应含同一指纹。
	events, err := st.Queries().ListFailureEventsWithRaw(ctx)
	if err != nil {
		t.Fatalf("读 failure_event: %v", err)
	}
	var fp string
	for _, e := range events {
		if e.ProjectID == project {
			fp = e.FpExact
		}
	}
	if fp == "" {
		t.Fatal("failure_event 应被录入")
	}
	if n := countDiagnoseJobsFor(t, st, fp); n != 1 {
		t.Errorf("应按失败指纹自动入队 1 个 diagnose 任务,有 %d 个", n)
	}

	// 成功构建:不录失败事件 → 不再入队 diagnose。
	resp, out = postWebhook(t, srv, buildPayload{
		Project: project + "-ok", Commit: "fedcba987654", Status: "passed", Log: "build ok",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("成功构建状态码 = %d,想要 200(响应 %+v)", resp.StatusCode, out)
	}
	if n := countDiagnoseJobsFor(t, st, fp); n != 1 {
		t.Errorf("成功构建不应再入队 diagnose,仍应是 1 个,有 %d 个", n)
	}
}

// countDiagnoseJobsFor 数队列里 payload 含给定指纹的 diagnose 任务数。
func countDiagnoseJobsFor(t *testing.T, st *store.Store, fp string) int {
	t.Helper()
	var n int
	if err := st.Pool().QueryRow(context.Background(),
		"SELECT count(*) FROM job WHERE kind = $1 AND strpos(payload::text, $2) > 0",
		string(jobs.KindDiagnose), fp).Scan(&n); err != nil {
		t.Fatalf("数 diagnose 任务: %v", err)
	}
	return n
}

// ptrInt 是指针辅助:exit_code 可空,缺省由录入方按 status 推导。
func ptrInt(v int) *int { return &v }
