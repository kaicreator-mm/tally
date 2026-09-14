package environment

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"tally/internal/runner"
	"tally/internal/store"
)

// fakeDocker 实现 DockerRunner,回固定结果并记录调用。
type fakeDocker struct {
	upRes     runner.Result
	upErr     error
	downRes   runner.Result
	downErr   error
	images    []string
	imagesErr error
	logs      string
	serverV   string
	composeV  string
	calls     []string
}

func (f *fakeDocker) ComposeUp(_ context.Context, _, _ string) (runner.Result, error) {
	f.calls = append(f.calls, "up")
	return f.upRes, f.upErr
}
func (f *fakeDocker) ComposeDown(_ context.Context, _, _ string) (runner.Result, error) {
	f.calls = append(f.calls, "down")
	return f.downRes, f.downErr
}
func (f *fakeDocker) Pull(_ context.Context, _ string) (runner.Result, error) {
	f.calls = append(f.calls, "pull")
	return runner.Result{}, nil
}
func (f *fakeDocker) ComposeImages(_ context.Context, _, _ string) ([]string, error) {
	return f.images, f.imagesErr
}
func (f *fakeDocker) ComposeLogs(_ context.Context, _, _, _ string) (runner.Result, error) {
	return runner.Result{Stdout: f.logs}, nil
}
func (f *fakeDocker) ServerVersion(_ context.Context) (string, error)  { return f.serverV, nil }
func (f *fakeDocker) ComposeVersion(_ context.Context) (string, error) { return f.composeV, nil }

// fakeRecorder 实现 RunRecorder,记下写入的事实行。
type fakeRecorder struct {
	last store.InsertEnvironmentRunOutcomeParams
	n    int
	err  error
}

func (f *fakeRecorder) InsertEnvironmentRunOutcome(_ context.Context, arg store.InsertEnvironmentRunOutcomeParams) (int64, error) {
	f.last = arg
	f.n++
	return int64(100 + f.n), f.err
}

// fakeSSH 实现 SSHRunner,回固定结果。
type fakeSSH struct {
	runRes runner.Result
	runErr error
}

func (f *fakeSSH) Run(_ context.Context, _ runner.Command) (runner.Result, error) {
	return f.runRes, f.runErr
}

// sshDef 构造一个 ssh-script 环境定义(带 teardown 脚本)。
func sshDef(t *testing.T) Definition {
	t.Helper()
	return Definition{
		Name: "dev", Project: "demo", Kind: KindSSHScript,
		Dir: t.TempDir(), Hash: "sha256:deadbeef",
		SSH:          SSHDef{Host: "10.0.0.5", User: "deploy"},
		Script:       "echo up",
		Teardown:     "echo down",
		HealthChecks: []HealthCheck{{Name: "log", Type: CheckLogPattern, Pattern: "ready"}},
		Wait:         WaitPolicy{Timeout: duration(2 * time.Second), Interval: duration(10 * time.Millisecond)},
	}
}

func composeDef(t *testing.T) Definition {
	t.Helper()
	return Definition{
		Name: "dev", Project: "demo", Kind: KindDockerCompose,
		ComposeFile: "dc.yml", Dir: t.TempDir(), Hash: "sha256:deadbeef",
		HealthChecks: []HealthCheck{{Name: "log", Type: CheckLogPattern, Pattern: "ready"}},
		Wait:         WaitPolicy{Timeout: duration(2 * time.Second), Interval: duration(10 * time.Millisecond)},
	}
}

// 验收:拉起成功后 environment_run 有一行,version_combo 非空且含镜像 tag 与运行时版本。
func TestUpComposeSuccess(t *testing.T) {
	fd := &fakeDocker{
		upRes:    runner.Result{ExitCode: 0, Stdout: "Started"},
		images:   []string{"alpine:latest"},
		logs:     "sleeper | ready marker",
		serverV:  "29.6.1",
		composeV: "5.3.0",
	}
	rec := &fakeRecorder{}
	m := &Manager{Docker: fd, Store: rec, Now: func() time.Time { return time.Unix(1000, 0).UTC() }}

	res, err := m.Up(context.Background(), composeDef(t))
	if err != nil {
		t.Fatalf("健康应拉起成功: %v", err)
	}
	if res.RunID == 0 {
		t.Error("应返回记录 id")
	}
	if rec.n != 1 || rec.last.Status != StatusUp {
		t.Errorf("应写一行 status=up: n=%d status=%s", rec.n, rec.last.Status)
	}
	if rec.last.DefinitionHash != "sha256:deadbeef" {
		t.Errorf("definition_hash 应落库: %s", rec.last.DefinitionHash)
	}
	// version_combo:运行时版本 + 镜像 tag。
	if res.Combo["docker"] != "29.6.1" || res.Combo["compose"] != "5.3.0" {
		t.Errorf("version_combo 应含运行时版本: %+v", res.Combo)
	}
	if res.Combo["image:0"] != "alpine:latest" {
		t.Errorf("version_combo 应含镜像 tag: %+v", res.Combo)
	}
	// 落库的 version_combo JSON 也要能回读出镜像 tag。
	var combo map[string]string
	if err := json.Unmarshal(rec.last.VersionCombo, &combo); err != nil || combo["image:0"] != "alpine:latest" {
		t.Errorf("落库 version_combo 应含镜像 tag: %s", rec.last.VersionCombo)
	}
	if len(rec.last.VersionCombo) == 0 {
		t.Error("version_combo 不得为空")
	}
	if !rec.last.FinishedAt.Valid {
		t.Error("成功时应写 finished_at")
	}
}

// 验收:拉起失败把原始输出交给 logpipe 入口;失败也记一行(status=failed)。
func TestUpComposeFailureFeedsLogpipe(t *testing.T) {
	fd := &fakeDocker{
		upRes: runner.Result{ExitCode: 1, Stderr: "Error: service \"db\" failed to start: connection refused"},
	}
	rec := &fakeRecorder{}
	m := &Manager{Docker: fd, Store: rec, Now: func() time.Time { return time.Unix(1000, 0).UTC() }}

	_, err := m.Up(context.Background(), composeDef(t))
	if err == nil {
		t.Fatal("compose up 非零退出码应返回 error")
	}
	if !strings.Contains(err.Error(), "指纹") {
		t.Errorf("error 应带 logpipe 算出的指纹: %v", err)
	}
	if rec.n != 1 || rec.last.Status != StatusFailed {
		t.Fatalf("失败也应记一行 status=failed: n=%d status=%s", rec.n, rec.last.Status)
	}
	// health_check_result 里应有 logpipe 处理结果(脱敏片段 + 指纹)。
	var hr healthResultJSON
	if err := json.Unmarshal(rec.last.HealthCheckResult, &hr); err != nil {
		t.Fatalf("health_check_result 应是合法 JSON: %v", err)
	}
	if hr.OK {
		t.Error("失败时 health_check_result.ok 应是 false")
	}
	if hr.Logpipe == nil || hr.Logpipe.ExactFP == "" {
		t.Errorf("失败时应把 logpipe 产出(指纹)记进 health_check_result: %+v", hr.Logpipe)
	}
	// 结构化错误(T10b):带上 environment_run id 与同一份指纹,供编排层录入 failure_event。
	var uerr *UpError
	if !errors.As(err, &uerr) {
		t.Fatalf("失败应返回 *UpError: %T", err)
	}
	if uerr.RunID != 101 { // fakeRecorder 首次插入返回 100+1
		t.Errorf("UpError.RunID 应是刚记的 environment_run id: %d", uerr.RunID)
	}
	if string(uerr.Processed.Exact) != hr.Logpipe.ExactFP {
		t.Errorf("UpError 指纹应与 health_check_result 一致: %s vs %s", uerr.Processed.Exact, hr.Logpipe.ExactFP)
	}
	if errors.Unwrap(uerr) == nil {
		t.Error("UpError 应能 Unwrap 出原始 cause")
	}
}

// 健康检查不过也走失败路径(logpipe 同样接线)。
func TestUpComposeUnhealthy(t *testing.T) {
	fd := &fakeDocker{
		upRes: runner.Result{ExitCode: 0},
		logs:  "sleeper | still booting", // 不含 "ready"
	}
	rec := &fakeRecorder{}
	def := composeDef(t)
	def.Wait = WaitPolicy{Timeout: duration(60 * time.Millisecond), Interval: duration(15 * time.Millisecond)}
	m := &Manager{Docker: fd, Store: rec}

	if _, err := m.Up(context.Background(), def); err == nil {
		t.Fatal("健康检查不过应返回 error")
	}
	if rec.last.Status != StatusFailed {
		t.Errorf("健康检查不过应记 failed: %s", rec.last.Status)
	}
}

func TestDownCompose(t *testing.T) {
	fd := &fakeDocker{downRes: runner.Result{ExitCode: 0}}
	rec := &fakeRecorder{}
	m := &Manager{Docker: fd, Store: rec}
	if err := m.Down(context.Background(), composeDef(t)); err != nil {
		t.Fatalf("down: %v", err)
	}
	if len(fd.calls) != 1 || fd.calls[0] != "down" {
		t.Errorf("应调用 ComposeDown: %v", fd.calls)
	}
	// 推倒成功应记一行 status=torn_down。
	if rec.n != 1 || rec.last.Status != StatusTornDown {
		t.Errorf("推倒成功应记一行 torn_down: n=%d status=%s", rec.n, rec.last.Status)
	}
	if rec.last.DefinitionHash != "sha256:deadbeef" {
		t.Errorf("torn_down 行应带 definition_hash: %s", rec.last.DefinitionHash)
	}
	// version_combo 只记 kind(推倒时不采集镜像版本)。
	var combo map[string]string
	if err := json.Unmarshal(rec.last.VersionCombo, &combo); err != nil || combo["kind"] != string(KindDockerCompose) {
		t.Errorf("torn_down 的 version_combo 应含 kind: %s", rec.last.VersionCombo)
	}
	// health_check_result 为 null(推倒无健康检查)。
	if rec.last.HealthCheckResult != nil {
		t.Errorf("torn_down 的 health_check_result 应为 null: %s", rec.last.HealthCheckResult)
	}
}

func TestDownComposeNonZeroFails(t *testing.T) {
	fd := &fakeDocker{downRes: runner.Result{ExitCode: 1, Stderr: "boom"}}
	rec := &fakeRecorder{}
	m := &Manager{Docker: fd, Store: rec}
	if err := m.Down(context.Background(), composeDef(t)); err == nil {
		t.Error("down 非零退出码应返回 error")
	}
	// 推倒失败不记事实行。
	if rec.n != 0 {
		t.Errorf("推倒失败不应记事实行: n=%d", rec.n)
	}
}

func TestRebuildIsDownThenUp(t *testing.T) {
	fd := &fakeDocker{
		upRes: runner.Result{ExitCode: 0}, downRes: runner.Result{ExitCode: 0},
		logs: "ready", serverV: "29.6.1", composeV: "5.3.0",
	}
	rec := &fakeRecorder{}
	m := &Manager{Docker: fd, Store: rec}
	if _, err := m.Rebuild(context.Background(), composeDef(t)); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	if len(fd.calls) < 2 || fd.calls[0] != "down" {
		t.Errorf("rebuild 应先 down 再 up: %v", fd.calls)
	}
	// rebuild = down(记 torn_down)+ up(记 up);最后一行应是 up。
	if rec.last.Status != StatusUp {
		t.Errorf("rebuild 后最后一行应是 up: %s", rec.last.Status)
	}
	if rec.n != 2 {
		t.Errorf("rebuild 应记两行(torn_down + up): n=%d", rec.n)
	}
}

// TestDownSSHRecordsTornDown 验证 ssh-script 推倒(有 teardown 脚本)也记 torn_down。
func TestDownSSHRecordsTornDown(t *testing.T) {
	ssh := &fakeSSH{runRes: runner.Result{ExitCode: 0}}
	rec := &fakeRecorder{}
	def := sshDef(t)
	m := &Manager{SSH: ssh, Store: rec}
	if err := m.Down(context.Background(), def); err != nil {
		t.Fatalf("ssh down: %v", err)
	}
	if rec.n != 1 || rec.last.Status != StatusTornDown {
		t.Errorf("ssh 推倒应记 torn_down: n=%d status=%s", rec.n, rec.last.Status)
	}
}

// TestDownSSHNoTeardownStillRecords 验证无 teardown 脚本时仍记 torn_down(调用方意图即推倒)。
func TestDownSSHNoTeardownStillRecords(t *testing.T) {
	rec := &fakeRecorder{}
	def := sshDef(t)
	def.Teardown = ""
	m := &Manager{Store: rec} // SSH 为 nil,无 teardown 时不需要
	if err := m.Down(context.Background(), def); err != nil {
		t.Fatalf("无 teardown 的 ssh down 不应报错: %v", err)
	}
	if rec.n != 1 || rec.last.Status != StatusTornDown {
		t.Errorf("无 teardown 脚本仍应记 torn_down: n=%d status=%s", rec.n, rec.last.Status)
	}
}
