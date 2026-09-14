//go:build integration

package ingest

// 集成测试:录入编排接真 Postgres(不 mock 数据库,docs/05-test-standards.md)。
// 每个测试用 store.TestDB 拿独立 schema,断言 build_run / failure_event / fingerprint_stat
// 三面齐落,且指纹与直接跑 logpipe 的产出一致。
//
// 项目身份(§33.3 ghost-project guard):新事实落库前先查派生项目注册表 project_index。
// 正向用例先经 registerProject 登记(模拟 Git 注册表派生入库),未知项目用例断言拒收
// 发生在事实 INSERT 之前,注册表不可查时断言 fail-closed。

import (
	"context"
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"tally/internal/command"
	"tally/internal/failure"
	"tally/internal/logpipe"
	"tally/internal/logpipe/toolchain"
	"tally/internal/store"

	"github.com/jackc/pgx/v5/pgtype"
)

func openStore(t *testing.T) *store.Store {
	t.Helper()
	return store.TestDB(t)
}

// registerProject 把项目登记进派生项目注册表(模拟注册表派生的增量入库路径,
// UpsertProjectIndex 幂等可反复调)。
func registerProject(t *testing.T, st *store.Store, projectID string) {
	t.Helper()
	if err := st.Queries().UpsertProjectIndex(context.Background(),
		store.UpsertProjectIndexParams{ProjectID: projectID, SourceCommit: "itest-registry"}); err != nil {
		t.Fatalf("登记项目 %s 进派生注册表: %v", projectID, err)
	}
}

// goBuildLog 读真实构建失败样本(已脱敏),断言「构建失败 → 同一指纹撞库」。
func goBuildLog(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("../logpipe/testdata/go/compile-error.log")
	if err != nil {
		t.Skipf("缺 logpipe 测试样本: %v", err)
	}
	return string(b)
}

// TestIngestBuildFailureWritesAllRows:一次失败构建录入 →
// build_run、failure_event(build_run_id=runID, source=build)、fingerprint_stat 各有对应行,
// 指纹与直接跑 go 抽取器一致。
func TestIngestBuildFailureWritesAllRows(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	raw := goBuildLog(t)
	at := time.Now().UTC()

	project := "ingest-itest-fail"
	registerProject(t, st, project)

	runID, eventID, fp, err := IngestBuild(ctx, st, BuildInput{
		Input: Input{
			ProjectID:  project,
			Raw:        raw,
			Toolchain:  "go",
			Combo:      map[string]string{"go": "1.23.0"},
			OccurredAt: at,
		},
		CommitSha:  "abc123def456",
		Status:     StatusFailed,
		ExitCode:   2,
		StartedAt:  at.Add(-time.Minute),
		FinishedAt: at,
	})
	if err != nil {
		t.Fatalf("IngestBuild: %v", err)
	}
	if runID == 0 || eventID == 0 {
		t.Fatalf("失败构建应同时返回 build_run 与 failure_event 的 id: run=%d event=%d", runID, eventID)
	}

	// build_run 行:commit / status / 工具链齐。
	var commit, status string
	var tool pgtype.Text
	if err := st.Pool().QueryRow(ctx,
		"SELECT commit_sha, status, toolchain FROM build_run WHERE id = $1", runID,
	).Scan(&commit, &status, &tool); err != nil {
		t.Fatalf("读 build_run: %v", err)
	}
	if commit != "abc123def456" || status != StatusFailed || tool.String != "go" {
		t.Errorf("build_run 行不对: commit=%q status=%q toolchain=%q", commit, status, tool.String)
	}

	// failure_event 行:来源外键指回 build_run,source=build,原始字节数对得上。
	var source string
	var buildRunID pgtype.Int8
	var fpExact string
	var rawBytes pgtype.Int8
	var pipelineVersion string
	if err := st.Pool().QueryRow(ctx,
		`SELECT source, build_run_id, fp_exact, raw_log_bytes, pipeline_version
		 FROM failure_event WHERE id = $1`, eventID,
	).Scan(&source, &buildRunID, &fpExact, &rawBytes, &pipelineVersion); err != nil {
		t.Fatalf("读 failure_event: %v", err)
	}
	if source != "build" || !buildRunID.Valid || buildRunID.Int64 != runID {
		t.Errorf("failure_event 应 source=build 且 build_run_id=%d: source=%q id=%+v", runID, source, buildRunID)
	}
	if rawBytes.Int64 != int64(len(raw)) {
		t.Errorf("raw_log_bytes 应是 %d,得到 %d", len(raw), rawBytes.Int64)
	}
	if pipelineVersion != logpipe.RuleVersion {
		t.Errorf("pipeline_version 应是 %q,得到 %q", logpipe.RuleVersion, pipelineVersion)
	}

	// 指纹与直接跑 go 抽取器一致(同日志同指纹,INV-8)。
	e, _ := toolchain.ByName("go")
	want := logpipe.ProcessWith(raw, e.Extract)
	if fpExact != string(want.Exact) {
		t.Errorf("fp_exact 应与 go 抽取器一致:\n got %s\nwant %s", fpExact, want.Exact)
	}
	// 返回的精确指纹与落库一致(T11e/D-e3:调用方拿它入队诊断)。
	if fp != want.Exact || string(fp) != fpExact {
		t.Errorf("返回指纹应与落库 fp_exact 一致: got %s want %s", fp, want.Exact)
	}

	// fingerprint_stat 与事件同事务而生,次数 1。
	var occ int64
	if err := st.Pool().QueryRow(ctx,
		"SELECT occurrence_count FROM fingerprint_stat WHERE fp_exact = $1", fpExact,
	).Scan(&occ); err != nil {
		t.Fatalf("fingerprint_stat 应与事件同事务写入: %v", err)
	}
	if occ != 1 {
		t.Errorf("新指纹应是 1 次,得到 %d", occ)
	}
}

// TestIngestBuildSuccessNoFailureEvent:成功构建只落 build_run,不落 failure_event。
func TestIngestBuildSuccessNoFailureEvent(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	at := time.Now().UTC()

	project := "ingest-itest-ok"
	registerProject(t, st, project)

	runID, eventID, fp, err := IngestBuild(ctx, st, BuildInput{
		Input: Input{
			ProjectID:  project,
			Raw:        "all good\nok  \tdemo\t0.3s",
			Toolchain:  "go",
			OccurredAt: at,
		},
		CommitSha:      "fff000",
		Status:         StatusPassed,
		ExitCode:       0,
		StartedAt:      at.Add(-time.Minute),
		FinishedAt:     at,
		ArtifactDigest: "sha256:art0000",
	})
	if err != nil {
		t.Fatalf("IngestBuild: %v", err)
	}
	if eventID != 0 {
		t.Errorf("成功构建不应录 failure_event,得到 event=%d", eventID)
	}
	if fp != "" {
		t.Errorf("成功构建不录失败事件,返回指纹应为空,得到 %s", fp)
	}

	var digest pgtype.Text
	if err := st.Pool().QueryRow(ctx,
		"SELECT artifact_digest FROM build_run WHERE id = $1", runID,
	).Scan(&digest); err != nil {
		t.Fatalf("读 build_run: %v", err)
	}
	if digest.String != "sha256:art0000" {
		t.Errorf("成功构建应记制品 digest: %q", digest.String)
	}

	var n int
	if err := st.Pool().QueryRow(ctx,
		"SELECT count(*) FROM failure_event WHERE project_id = $1", project,
	).Scan(&n); err != nil {
		t.Fatalf("数 failure_event: %v", err)
	}
	if n != 0 {
		t.Errorf("成功构建不应有 failure_event,有 %d 行", n)
	}
}

// TestRecordFailurePassthroughProcessed:业务包已算好 logpipe 产出(部署 / 环境的接入形状)
// → 透传指纹,不重复处理。
func TestRecordFailurePassthroughProcessed(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	at := time.Now().UTC()

	project := "ingest-itest-env"
	registerProject(t, st, project)

	// 来源外键需要真实行:先记一行 environment_run。
	envID, err := st.Queries().InsertEnvironmentRun(ctx, store.InsertEnvironmentRunParams{
		ProjectID:      project,
		EnvName:        "dev",
		DefinitionHash: "sha256:def",
		StartedAt:      pgtype.Timestamptz{Time: at, Valid: true},
		Status:         "failed",
		VersionCombo:   []byte(`{"kind":"docker-compose"}`),
	})
	if err != nil {
		t.Fatalf("记 environment_run: %v", err)
	}

	processed := logpipe.Process("health check failed: connection refused")
	eventID, fp, err := RecordFailure(ctx, st, Input{
		ProjectID:        project,
		Source:           "environment",
		EnvironmentRunID: envID,
		Processed:        &processed,
		RawLogBytes:      1234,
		OccurredAt:       at,
	})
	if err != nil {
		t.Fatalf("RecordFailure: %v", err)
	}
	if fp != processed.Exact {
		t.Errorf("返回指纹应是 Processed 的精确指纹: got %s want %s", fp, processed.Exact)
	}

	var dbFP string
	var rawBytes int64
	if err := st.Pool().QueryRow(ctx,
		"SELECT fp_exact, raw_log_bytes FROM failure_event WHERE id = $1", eventID,
	).Scan(&dbFP, &rawBytes); err != nil {
		t.Fatalf("读 failure_event: %v", err)
	}
	if dbFP != string(processed.Exact) {
		t.Errorf("应透传 Processed 的指纹: got %s want %s", dbFP, processed.Exact)
	}
	if rawBytes != 1234 {
		t.Errorf("应透传调用方给的 raw_log_bytes: %d", rawBytes)
	}
}

// TestIngestBuildPersistsReproContext(T11b 验收):ingest-build 携带的复现上下文
// (-workdir/-repro/-targets)按 0016 的结构化合同落库可查:repro_command 是
// CommandSpec 规范 JSON,other_target_commands 是 [{target,command}] 数组;
// 旧文本三列不再写入(存量行兼容读取来源,新行取默认空串)。
func TestIngestBuildPersistsReproContext(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	raw := goBuildLog(t)
	at := time.Now().UTC()

	project := "ingest-itest-repro"
	registerProject(t, st, project)

	_, eventID, _, err := IngestBuild(ctx, st, BuildInput{
		Input: Input{
			ProjectID:    project,
			Raw:          raw,
			Toolchain:    "go",
			OccurredAt:   at,
			Workdir:      "/srv/demo",
			Repro:        "go build ./...",
			OtherTargets: []string{"go build ./cmd/a", "go build ./cmd/b"},
		},
		CommitSha:  "abc900",
		Status:     StatusFailed,
		ExitCode:   1,
		StartedAt:  at.Add(-time.Minute),
		FinishedAt: at,
	})
	if err != nil {
		t.Fatalf("IngestBuild: %v", err)
	}

	var reproRaw, targetsRaw []byte
	if err := st.Pool().QueryRow(ctx,
		`SELECT repro_command, other_target_commands FROM failure_event WHERE id = $1`, eventID,
	).Scan(&reproRaw, &targetsRaw); err != nil {
		t.Fatalf("读 failure_event 结构化复现上下文: %v", err)
	}

	// repro_command:按旧执行语义重建的结构化命令(argv=strings.Fields,cwd=workdir,shell=false)。
	var repro command.CommandSpec
	if err := json.Unmarshal(reproRaw, &repro); err != nil {
		t.Fatalf("repro_command 应是 CommandSpec 规范 JSON: %v (%s)", err, reproRaw)
	}
	if !slices.Equal(repro.Argv, []string{"go", "build", "./..."}) || repro.Cwd != "/srv/demo" || repro.Shell {
		t.Errorf("repro_command 不对: %+v", repro)
	}

	// other_target_commands:每个旧文本目标 → {target=原文, command=同源重建的 spec}。
	var targets []failure.OtherTarget
	if err := json.Unmarshal(targetsRaw, &targets); err != nil {
		t.Fatalf("other_target_commands 应是 [{target,command}] 数组: %v (%s)", err, targetsRaw)
	}
	wantCmds := []string{"go build ./cmd/a", "go build ./cmd/b"}
	if len(targets) != len(wantCmds) {
		t.Fatalf("应落 2 个结构化目标,得到 %d: %s", len(targets), targetsRaw)
	}
	for i, w := range targets {
		if w.Target != wantCmds[i] {
			t.Errorf("目标 %d 的 Target 应用原文作标签: got %q want %q", i, w.Target, wantCmds[i])
		}
		if !slices.Equal(w.Command.Argv, strings.Fields(wantCmds[i])) || w.Command.Cwd != "/srv/demo" || w.Command.Shell {
			t.Errorf("目标 %d 的 Command 应按旧执行语义重建: %+v", i, w.Command)
		}
	}

	// 旧文本三列不再写入(0016 合同):新行取默认空串,只有结构化两列承载复现上下文。
	var workdir, reproText string
	if err := st.Pool().QueryRow(ctx,
		`SELECT workdir, repro FROM failure_event WHERE id = $1`, eventID,
	).Scan(&workdir, &reproText); err != nil {
		t.Fatalf("读 failure_event 旧文本列: %v", err)
	}
	if workdir != "" || reproText != "" {
		t.Errorf("新写入不应再落旧文本列: workdir=%q repro=%q", workdir, reproText)
	}
}

// TestRecordFailureRejectsUnknownProject(§33.3 ghost-project guard 验收):
// 项目不在派生注册表 → 事实拒收,且拒收发生在事实 INSERT 之前 —— failure_event /
// fingerprint_stat 都不出现该项目的任何行(已落库的历史行也不被改动,INV-5)。
func TestRecordFailureRejectsUnknownProject(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	at := time.Now().UTC()

	project := "ingest-itest-ghost" // 刻意不登记进 project_index
	// 来源外键需要真实行:ghost 事实的典型来历就是先有一行挂在该项目名下的执行记录。
	envID, err := st.Queries().InsertEnvironmentRun(ctx, store.InsertEnvironmentRunParams{
		ProjectID:      project,
		EnvName:        "dev",
		DefinitionHash: "sha256:def",
		StartedAt:      pgtype.Timestamptz{Time: at, Valid: true},
		Status:         "failed",
		VersionCombo:   []byte(`{"kind":"docker-compose"}`),
	})
	if err != nil {
		t.Fatalf("记 environment_run: %v", err)
	}

	processed := logpipe.Process("health check failed: connection refused")
	_, _, err = RecordFailure(ctx, st, Input{
		ProjectID:        project,
		Source:           "environment",
		EnvironmentRunID: envID,
		Processed:        &processed,
		RawLogBytes:      1234,
		OccurredAt:       at,
	})
	if err == nil {
		t.Fatal("未知项目的事实应被拒收")
	}
	if !strings.Contains(err.Error(), project) {
		t.Errorf("错误应点名未知项目 %q: %v", project, err)
	}

	var n int
	if err := st.Pool().QueryRow(ctx,
		"SELECT count(*) FROM failure_event WHERE project_id = $1", project,
	).Scan(&n); err != nil {
		t.Fatalf("数 failure_event: %v", err)
	}
	if n != 0 {
		t.Errorf("未知项目应在事实 INSERT 之前被拒收,却有 %d 行 failure_event", n)
	}
	var stat int64
	if err := st.Pool().QueryRow(ctx,
		"SELECT count(*) FROM fingerprint_stat").Scan(&stat); err != nil {
		t.Fatalf("数 fingerprint_stat: %v", err)
	}
	if stat != 0 {
		t.Errorf("拒收不应累加指纹索引面,却有 %d 行 fingerprint_stat", stat)
	}
}

// TestRecordFailureRegistryFailureFailClosed(§33.3 验收):注册表(project_index)
// 不可查 → fail-closed 拒收新事实,绝不放行(宁可少录,不写来历不明的事实)。
func TestRecordFailureRegistryFailureFailClosed(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	at := time.Now().UTC()

	project := "ingest-itest-closed"
	// 来源外键给真实行,证明错误来自注册表判定面而不是外键缺行。
	envID, err := st.Queries().InsertEnvironmentRun(ctx, store.InsertEnvironmentRunParams{
		ProjectID:      project,
		EnvName:        "dev",
		DefinitionHash: "sha256:def",
		StartedAt:      pgtype.Timestamptz{Time: at, Valid: true},
		Status:         "failed",
		VersionCombo:   []byte(`{"kind":"docker-compose"}`),
	})
	if err != nil {
		t.Fatalf("记 environment_run: %v", err)
	}

	st.Close() // 注册表不可查

	processed := logpipe.Process("health check failed: connection refused")
	_, _, err = RecordFailure(ctx, st, Input{
		ProjectID:        project,
		Source:           "environment",
		EnvironmentRunID: envID,
		Processed:        &processed,
		RawLogBytes:      1234,
		OccurredAt:       at,
	})
	if err == nil {
		t.Fatal("注册表不可查时应 fail-closed 拒收新事实")
	}
	if !strings.Contains(err.Error(), "fail-closed") {
		t.Errorf("错误应标明 fail-closed 拒收: %v", err)
	}
}
