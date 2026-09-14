package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"tally/internal/runner"
	"tally/internal/store"
)

// fakeRecorder 记录收到的 InsertDeployRun 参数。deploy 记账的单测替身。
type fakeRecorder struct {
	args []store.InsertDeployRunParams
	err  error
}

func (f *fakeRecorder) InsertDeployRun(_ context.Context, arg store.InsertDeployRunParams) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	f.args = append(f.args, arg)
	return int64(len(f.args)), nil
}

func newTestDeployer(exec *fakeExec, rec *fakeRecorder) *Deployer {
	return &Deployer{Exec: exec, Store: rec, Now: func() time.Time {
		return time.Date(2026, 7, 22, 10, 0, 0, 0, time.UTC)
	}}
}

// 验收:每次部署写一行 —— digest、目标、工具、结果、退出码、健康检查结果、版本组合,字段齐全。
func TestDeploySuccessRecordsFullRow(t *testing.T) {
	exec := &fakeExec{res: runner.Result{Stdout: "deployed ok", ExitCode: 0}}
	rec := &fakeRecorder{}
	d := newTestDeployer(exec, rec)

	res, err := d.Deploy(context.Background(), defFor(ToolKamal))
	if err != nil {
		t.Fatalf("退出码 0 应成功: %v", err)
	}
	if res.RunID != 1 || res.Status != StatusDeployed {
		t.Errorf("应记一行 status=deployed 并返回 run id: %+v", res)
	}
	if len(rec.args) != 1 {
		t.Fatalf("应恰好写一行 deploy_run,实际 %d", len(rec.args))
	}
	a := rec.args[0]
	if a.ProjectID != "myapp" || a.ArtifactDigest != "sha256:abc123" ||
		a.TargetEnv != "staging" || a.Tool != "kamal" || a.Status != "deployed" {
		t.Errorf("记账字段错: %+v", a)
	}
	if !a.StartedAt.Valid || !a.FinishedAt.Valid {
		t.Error("started_at / finished_at 应有值")
	}
	if !a.ExitCode.Valid || a.ExitCode.Int32 != 0 {
		t.Errorf("退出码应记 0: %+v", a.ExitCode)
	}
	// 健康检查结果:工具退出码为 0 → ok=true,标明由哪个工具的内置健康检查背书。
	var hc healthJSON
	if err := json.Unmarshal(a.HealthCheckResult, &hc); err != nil {
		t.Fatalf("health_check_result 应是合法 JSON: %v", err)
	}
	if !hc.OK || hc.By != "kamal" {
		t.Errorf("成功时健康检查结果应是 ok=true by=kamal: %+v", hc)
	}
	if hc.Logpipe != nil {
		t.Error("成功时不应有 logpipe 片段")
	}
	// 版本组合非空且含制品 digest。
	var combo map[string]string
	if err := json.Unmarshal(a.VersionCombo, &combo); err != nil || len(combo) == 0 {
		t.Fatalf("version_combo 应是非空 JSON 对象: %v", err)
	}
	if combo["artifact_digest"] != "sha256:abc123" || combo["target_env"] != "staging" {
		t.Errorf("version_combo 应含制品 digest 与目标环境: %v", combo)
	}
	// build_run_id 可空:定义没给 → NULL。
	if a.BuildRunID.Valid {
		t.Error("未给 build_run_id 时应是 NULL(制品可能来自外部)")
	}
}

// build_run_id 给了就记上(制品 → 部署的追溯坐标)。
func TestDeployRecordsBuildRunID(t *testing.T) {
	exec := &fakeExec{res: runner.Result{ExitCode: 0}}
	rec := &fakeRecorder{}
	d := newTestDeployer(exec, rec)
	def := defFor(ToolCompose)
	def.BuildRun = 42
	if _, err := d.Deploy(context.Background(), def); err != nil {
		t.Fatal(err)
	}
	if a := rec.args[0]; !a.BuildRunID.Valid || a.BuildRunID.Int64 != 42 {
		t.Errorf("build_run_id 应记 42: %+v", a.BuildRunID)
	}
}

// 验收:失败输出送 logpipe,source 标为 deploy。非零退出码 → 记 status=failed 的一行 + 指纹。
func TestDeployFailureFeedsLogpipeSourceDeploy(t *testing.T) {
	exec := &fakeExec{res: runner.Result{
		Stderr:   "ERROR: manifest unknown: connection refused to registry.example.com",
		ExitCode: 1,
	}}
	rec := &fakeRecorder{}
	d := newTestDeployer(exec, rec)

	_, err := d.Deploy(context.Background(), defFor(ToolCompose))
	if err == nil {
		t.Fatal("非零退出码应返回 error")
	}
	if !strings.Contains(err.Error(), "指纹") {
		t.Errorf("error 应带 logpipe 指纹: %v", err)
	}
	if len(rec.args) != 1 {
		t.Fatalf("失败也要记一行,实际 %d 行", len(rec.args))
	}
	a := rec.args[0]
	if a.Status != "failed" {
		t.Errorf("status 应是 failed: %s", a.Status)
	}
	if !a.ExitCode.Valid || a.ExitCode.Int32 != 1 {
		t.Errorf("应记退出码 1: %+v", a.ExitCode)
	}
	var hc healthJSON
	if err := json.Unmarshal(a.HealthCheckResult, &hc); err != nil {
		t.Fatal(err)
	}
	if hc.OK {
		t.Error("失败时健康检查结果 ok 应是 false")
	}
	if hc.Logpipe == nil {
		t.Fatal("失败输出应经 logpipe 处理并记录")
	}
	if hc.Logpipe.Source != "deploy" {
		t.Errorf("logpipe 来源应标为 deploy: %+v", hc.Logpipe)
	}
	if hc.Logpipe.ExactFP == "" || hc.Logpipe.Excerpt == "" {
		t.Errorf("logpipe 应产出指纹与脱敏片段: %+v", hc.Logpipe)
	}
	// 结构化错误(T10b):带上 deploy_run id 与同一份指纹,供编排层录入 failure_event。
	var ferr *FailureError
	if !errors.As(err, &ferr) {
		t.Fatalf("失败应返回 *FailureError: %T", err)
	}
	if ferr.RunID != 1 { // fakeRecorder 首次插入返回 len(args)=1
		t.Errorf("FailureError.RunID 应是刚记的 deploy_run id: %d", ferr.RunID)
	}
	if ferr.Status != StatusFailed {
		t.Errorf("FailureError.Status 应是 failed: %s", ferr.Status)
	}
	if string(ferr.Processed.Exact) != hc.Logpipe.ExactFP {
		t.Errorf("FailureError 指纹应与 health_check_result 一致: %s vs %s", ferr.Processed.Exact, hc.Logpipe.ExactFP)
	}
	if errors.Unwrap(ferr) == nil {
		t.Error("FailureError 应能 Unwrap 出原始 cause")
	}
}

// 工具没跑起来(runner 报 error)→ 记一行 failed,退出码 NULL,输出仍送 logpipe。
func TestDeployRunErrorRecordsNullExitCode(t *testing.T) {
	exec := &fakeExec{err: errors.New("exec: \"kamal\": executable file not found")}
	rec := &fakeRecorder{}
	d := newTestDeployer(exec, rec)

	if _, err := d.Deploy(context.Background(), defFor(ToolKamal)); err == nil {
		t.Fatal("没跑起来应返回 error")
	}
	a := rec.args[0]
	if a.Status != "failed" {
		t.Errorf("status 应是 failed: %s", a.Status)
	}
	if a.ExitCode.Valid {
		t.Errorf("没跑完不应有退出码(NULL): %+v", a.ExitCode)
	}
}

// 记账失败要上抛,不能被部署结果盖掉。
func TestDeployRecorderErrorPropagates(t *testing.T) {
	exec := &fakeExec{res: runner.Result{ExitCode: 0}}
	rec := &fakeRecorder{err: errors.New("connection refused")}
	d := newTestDeployer(exec, rec)
	if _, err := d.Deploy(context.Background(), defFor(ToolKamal)); err == nil {
		t.Fatal("记账失败应返回 error")
	}
}

// 验收:回滚调工具自带回滚命令,本包只转发;成功记 rolled_back,失败记 rollback_failed。
func TestRollbackRecordsAndForwards(t *testing.T) {
	exec := &fakeExec{res: runner.Result{ExitCode: 0}}
	rec := &fakeRecorder{}
	d := newTestDeployer(exec, rec)

	def := defFor(ToolKamal)
	def.Digest = "sha256:previous-good" // 回滚目标 = 上一个能跑的 digest
	res, err := d.Rollback(context.Background(), def)
	if err != nil {
		t.Fatalf("回滚应成功: %v", err)
	}
	if res.Status != StatusRolledBack || rec.args[0].Status != "rolled_back" {
		t.Errorf("回滚成功应记 rolled_back: %+v / %+v", res, rec.args[0])
	}
	cmd := exec.cmds[0].Argv
	if cmd[0] != "kamal" || cmd[1] != "rollback" {
		t.Errorf("应转发 kamal rollback: %v", cmd)
	}
	joined := strings.Join(cmd, " ")
	if !strings.Contains(joined, "sha256:previous-good") {
		t.Errorf("回滚命令应带目标 digest: %v", cmd)
	}

	// 回滚失败 → rollback_failed,输出同样送 logpipe(source=deploy)。
	exec2 := &fakeExec{res: runner.Result{Stderr: "rollback blew up", ExitCode: 3}}
	rec2 := &fakeRecorder{}
	d2 := newTestDeployer(exec2, rec2)
	if _, err := d2.Rollback(context.Background(), defFor(ToolAnsible)); err == nil {
		t.Fatal("回滚失败应返回 error")
	}
	a := rec2.args[0]
	if a.Status != "rollback_failed" {
		t.Errorf("回滚失败应记 rollback_failed: %s", a.Status)
	}
	var hc healthJSON
	if err := json.Unmarshal(a.HealthCheckResult, &hc); err != nil {
		t.Fatal(err)
	}
	if hc.Logpipe == nil || hc.Logpipe.Source != "deploy" {
		t.Errorf("回滚失败输出也应送 logpipe 且 source=deploy: %+v", hc.Logpipe)
	}
}

// 验收:隔离环境。覆盖 target_env 后,命令与记账都指向隔离环境,不复用原测试环境。
func TestDeployToIsolatedEnv(t *testing.T) {
	exec := &fakeExec{res: runner.Result{ExitCode: 0}}
	rec := &fakeRecorder{}
	d := newTestDeployer(exec, rec)

	def := defFor(ToolCompose)
	def.TargetEnv = "verify-iso-7" // 重跑验证:部到隔离环境,不碰 staging
	if _, err := d.Deploy(context.Background(), def); err != nil {
		t.Fatal(err)
	}
	if got := rec.args[0].TargetEnv; got != "verify-iso-7" {
		t.Errorf("记账的 target_env 应是隔离环境: %s", got)
	}
	joined := fmt.Sprint(exec.cmds[0].Argv) + fmt.Sprint(exec.cmds[0].Env)
	if !strings.Contains(joined, "verify-iso-7") {
		t.Errorf("命令应指向隔离环境: %v %v", exec.cmds[0].Argv, exec.cmds[0].Env)
	}
	if strings.Contains(joined, "staging") {
		t.Errorf("隔离部署不应再碰原环境 staging: %v", joined)
	}
}

// 防御:手搓的非法定义(空 digest)不进调度,DB 前就被拦下。
func TestDeployRejectsInvalidDefinition(t *testing.T) {
	exec := &fakeExec{res: runner.Result{ExitCode: 0}}
	rec := &fakeRecorder{}
	d := newTestDeployer(exec, rec)
	def := defFor(ToolKamal)
	def.Digest = ""
	if _, err := d.Deploy(context.Background(), def); err == nil {
		t.Fatal("空 digest 的定义应被拒绝")
	}
	if len(exec.cmds) != 0 || len(rec.args) != 0 {
		t.Error("非法定义不应执行工具,也不应记账")
	}
}
