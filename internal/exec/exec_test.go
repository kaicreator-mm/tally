package exec

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"tally/internal/core"
	"tally/internal/jobs"
	"tally/internal/runner"
	"tally/internal/store"
)

/* ---------- 手写假实现(不引 mock 框架,见 docs/05-test-standards.md) ---------- */

// stubRecorder 记录写事实面的参数,离线断言「worker 执行动作才写事实面」。
type stubRecorder struct {
	mu      sync.Mutex
	envRuns []store.InsertEnvironmentRunOutcomeParams
	depRuns []store.InsertDeployRunParams
}

func (r *stubRecorder) InsertEnvironmentRunOutcome(_ context.Context, p store.InsertEnvironmentRunOutcomeParams) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.envRuns = append(r.envRuns, p)
	return int64(len(r.envRuns)), nil
}

func (r *stubRecorder) InsertDeployRun(_ context.Context, p store.InsertDeployRunParams) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.depRuns = append(r.depRuns, p)
	return int64(len(r.depRuns)), nil
}

func (r *stubRecorder) envCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.envRuns)
}

func (r *stubRecorder) depCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.depRuns)
}

// fakeDocker 是脚本化的 docker 执行器:全部命令成功,版本信息固定。
type fakeDocker struct {
	upExit   int
	downExit int
}

func (f fakeDocker) ComposeUp(context.Context, string, string) (runner.Result, error) {
	return runner.Result{ExitCode: f.upExit, Stdout: "container started"}, nil
}
func (f fakeDocker) ComposeDown(context.Context, string, string) (runner.Result, error) {
	return runner.Result{ExitCode: f.downExit}, nil
}
func (f fakeDocker) Pull(context.Context, string) (runner.Result, error) {
	return runner.Result{ExitCode: 0}, nil
}
func (f fakeDocker) ComposeImages(context.Context, string, string) ([]string, error) {
	return []string{"alpine:3.20"}, nil
}
func (f fakeDocker) ComposeLogs(context.Context, string, string, string) (runner.Result, error) {
	return runner.Result{}, nil
}
func (f fakeDocker) ServerVersion(context.Context) (string, error)  { return "27.0", nil }
func (f fakeDocker) ComposeVersion(context.Context) (string, error) { return "2.29", nil }

// fakeCmd 记录部署工具收到的命令,退出码可配。
type fakeCmd struct {
	mu   sync.Mutex
	cmds []runner.Command
	exit int
}

func (f *fakeCmd) Run(_ context.Context, cmd runner.Command) (runner.Result, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cmds = append(f.cmds, cmd)
	return runner.Result{ExitCode: f.exit, Stdout: "tool done"}, nil
}

/* ---------- 定义夹具 ---------- */

func writeEnvYAML(t *testing.T, root, rel, name, project, healthChecks string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := "name: " + name + "\nproject: " + project + "\nkind: docker-compose\n" +
		"compose_file: ./compose.yml\n" + healthChecks +
		"wait:\n  timeout: 2s\n  interval: 200ms\n"
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
}

func httpCheckYAML(url string) string {
	return "health_checks:\n  - name: api\n    type: http\n    url: " + url + "\n"
}

func writeDeployYAML(t *testing.T, root, rel, project, targetEnv string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := "project: " + project + "\ntarget_env: " + targetEnv + "\ntool: compose\n" +
		"artifact_digest: \"sha256:orig0000\"\nconfig: ./deploy-compose.yml\n"
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
}

func payload(t *testing.T, v any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

/* ---------- 环境动作:真正执行并写事实面 ---------- */

func TestHandlersEnvUpExecutesAndWritesFact(t *testing.T) {
	health := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(health.Close)

	root := t.TempDir()
	writeEnvYAML(t, root, "environments/dev.yaml", "dev", "forge", httpCheckYAML(health.URL))
	rec := &stubRecorder{}
	table := Handlers(Config{Root: root, Store: rec, Docker: fakeDocker{}})

	err := table[jobs.KindEnvironmentUp](context.Background(),
		payload(t, jobs.EnvActionPayload{Project: "forge", Name: "dev", Def: "environments/dev.yaml"}))
	if err != nil {
		t.Fatalf("拉起应成功: %v", err)
	}
	// 执行端写了一行事实:status=up,定义哈希与组合齐备。
	if rec.envCount() != 1 {
		t.Fatalf("应写 1 行 environment_run,写了 %d", rec.envCount())
	}
	p := rec.envRuns[0]
	if p.Status != "up" || p.ProjectID != "forge" || p.EnvName != "dev" {
		t.Errorf("事实行不对: %+v", p)
	}
	if !strings.HasPrefix(p.DefinitionHash, "sha256:") {
		t.Errorf("应记定义内容哈希(INV-4): %q", p.DefinitionHash)
	}
	combo := string(p.VersionCombo)
	if !strings.Contains(combo, `"docker":"27.0"`) || !strings.Contains(combo, "alpine:3.20") {
		t.Errorf("应记版本组合: %s", combo)
	}
}

func TestHandlersEnvUpFailedHealthWritesFailedFact(t *testing.T) {
	// 健康检查目标不可达 → 拉起失败:仍然写一行 status=failed(失败也是事实),handler 返回错误。
	health := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	healthURL := health.URL
	health.Close()

	root := t.TempDir()
	writeEnvYAML(t, root, "environments/dev.yaml", "dev", "forge", httpCheckYAML(healthURL))
	rec := &stubRecorder{}
	table := Handlers(Config{Root: root, Store: rec, Docker: fakeDocker{}})

	err := table[jobs.KindEnvironmentUp](context.Background(),
		payload(t, jobs.EnvActionPayload{Def: "environments/dev.yaml"}))
	if err == nil {
		t.Fatal("健康检查不过,拉起应报错")
	}
	if rec.envCount() != 1 {
		t.Fatalf("失败也应写 1 行 environment_run,写了 %d", rec.envCount())
	}
	if rec.envRuns[0].Status != "failed" {
		t.Errorf("失败行 status 应是 failed: %+v", rec.envRuns[0])
	}
}

func TestHandlersEnvDownExecutesWithoutFact(t *testing.T) {
	root := t.TempDir()
	writeEnvYAML(t, root, "environments/dev.yaml", "dev", "forge", httpCheckYAML("http://127.0.0.1:1"))
	rec := &stubRecorder{}
	table := Handlers(Config{Root: root, Store: rec, Docker: fakeDocker{}})

	err := table[jobs.KindEnvironmentDown](context.Background(),
		payload(t, jobs.EnvActionPayload{Def: "environments/dev.yaml"}))
	if err != nil {
		t.Fatalf("推倒应成功: %v", err)
	}
	// 推倒成功写一行 status=torn_down(INV-5 只追加)。
	if rec.envCount() != 1 || rec.envRuns[0].Status != "torn_down" {
		t.Errorf("推倒成功应写一行 torn_down: %+v", rec.envRuns)
	}
}

func TestHandlersEnvDownFailureReports(t *testing.T) {
	root := t.TempDir()
	writeEnvYAML(t, root, "environments/dev.yaml", "dev", "forge", httpCheckYAML("http://127.0.0.1:1"))
	table := Handlers(Config{Root: root, Store: &stubRecorder{}, Docker: fakeDocker{downExit: 1}})

	err := table[jobs.KindEnvironmentDown](context.Background(),
		payload(t, jobs.EnvActionPayload{Def: "environments/dev.yaml"}))
	if err == nil || !strings.Contains(err.Error(), "推倒环境") {
		t.Errorf("推倒失败应报清楚: %v", err)
	}
}

func TestHandlersEnvRebuildDownThenUp(t *testing.T) {
	health := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(health.Close)

	root := t.TempDir()
	writeEnvYAML(t, root, "environments/dev.yaml", "dev", "forge", httpCheckYAML(health.URL))
	rec := &stubRecorder{}
	table := Handlers(Config{Root: root, Store: rec, Docker: fakeDocker{}})

	err := table[jobs.KindEnvironmentRebuild](context.Background(),
		payload(t, jobs.EnvActionPayload{Def: "environments/dev.yaml"}))
	if err != nil {
		t.Fatalf("重建应成功: %v", err)
	}
	// rebuild = down(记 torn_down)+ up(记 up);共两行,最后一行是 up。
	if rec.envCount() != 2 || rec.envRuns[1].Status != "up" {
		t.Errorf("重建应写两行(torn_down + up): %+v", rec.envRuns)
	}
}

/* ---------- 部署:按定义调工具并写 deploy_run ---------- */

func TestHandlersDeployExecutesAndWritesFact(t *testing.T) {
	root := t.TempDir()
	writeDeployYAML(t, root, "deploys/forge-prod.yaml", "forge", "prod")
	rec := &stubRecorder{}
	cmd := &fakeCmd{exit: 0}
	table := Handlers(Config{Root: root, Store: rec, Docker: fakeDocker{}, Exec: cmd})

	err := table[jobs.KindDeploy](context.Background(), payload(t, jobs.DeployPayload{
		Project: "forge", Def: "deploys/forge-prod.yaml",
	}))
	if err != nil {
		t.Fatalf("部署应成功: %v", err)
	}
	// 工具真被调,且制品 digest 来自定义(compose 经 ARTIFACT_DIGEST 环境变量拿到制品)。
	if len(cmd.cmds) != 1 {
		t.Fatalf("应调一次部署工具,调了 %d 次", len(cmd.cmds))
	}
	joined := strings.Join(cmd.cmds[0].Argv, " ")
	if !strings.Contains(joined, "compose") {
		t.Errorf("应走 compose 工具: %s", joined)
	}
	if !slices.Contains(cmd.cmds[0].Env, "ARTIFACT_DIGEST=sha256:orig0000") {
		t.Errorf("工具命令应带定义里的制品 digest: %v", cmd.cmds[0].Env)
	}
	if rec.depCount() != 1 {
		t.Fatalf("应写 1 行 deploy_run,写了 %d", rec.depCount())
	}
	p := rec.depRuns[0]
	if p.Status != "deployed" || p.ArtifactDigest != "sha256:orig0000" || p.TargetEnv != "prod" {
		t.Errorf("部署事实行不对: %+v", p)
	}
}

func TestHandlersDeployFailureWritesFailedFact(t *testing.T) {
	root := t.TempDir()
	writeDeployYAML(t, root, "deploys/forge-prod.yaml", "forge", "prod")
	rec := &stubRecorder{}
	table := Handlers(Config{Root: root, Store: rec, Docker: fakeDocker{}, Exec: &fakeCmd{exit: 7}})

	err := table[jobs.KindDeploy](context.Background(), payload(t, jobs.DeployPayload{
		Def: "deploys/forge-prod.yaml",
	}))
	if err == nil {
		t.Fatal("工具退出码非零,部署应报错")
	}
	// 失败行已由 Deployer 记下:worker 执行动作写事实面,失败也写。
	if rec.depCount() != 1 || rec.depRuns[0].Status != "failed" {
		t.Errorf("失败也应写一行 failed: %+v", rec.depRuns)
	}
}

/* ---------- 回滚:调工具并写 deploy_run ---------- */

func TestHandlersRollbackExecutesAndWritesFact(t *testing.T) {
	root := t.TempDir()
	writeDeployYAML(t, root, "deploys/forge-prod.yaml", "forge", "prod")
	rec := &stubRecorder{}
	cmd := &fakeCmd{exit: 0}
	table := Handlers(Config{Root: root, Store: rec, Docker: fakeDocker{}, Exec: cmd})

	err := table[jobs.KindDeployRollback](context.Background(), payload(t, jobs.DeployRollbackPayload{
		Project: "forge", TargetEnv: "prod", Digest: "sha256:good1234", Def: "deploys/forge-prod.yaml",
	}))
	if err != nil {
		t.Fatalf("回滚应成功: %v", err)
	}
	// 工具真被调,且 digest / 目标来自 payload(覆盖定义里的值)。
	if len(cmd.cmds) != 1 {
		t.Fatalf("应调一次部署工具,调了 %d 次", len(cmd.cmds))
	}
	joined := strings.Join(cmd.cmds[0].Argv, " ")
	if !strings.Contains(joined, "compose") {
		t.Errorf("应走 compose 工具: %s", joined)
	}
	if rec.depCount() != 1 {
		t.Fatalf("应写 1 行 deploy_run,写了 %d", rec.depCount())
	}
	p := rec.depRuns[0]
	if p.Status != "rolled_back" || p.ArtifactDigest != "sha256:good1234" || p.TargetEnv != "prod" {
		t.Errorf("回滚事实行不对: %+v", p)
	}
}

func TestHandlersRollbackFailureWritesFailedFact(t *testing.T) {
	root := t.TempDir()
	writeDeployYAML(t, root, "deploys/forge-prod.yaml", "forge", "prod")
	rec := &stubRecorder{}
	table := Handlers(Config{Root: root, Store: rec, Docker: fakeDocker{}, Exec: &fakeCmd{exit: 3}})

	err := table[jobs.KindDeployRollback](context.Background(), payload(t, jobs.DeployRollbackPayload{
		TargetEnv: "prod", Digest: "sha256:good1234", Def: "deploys/forge-prod.yaml",
	}))
	if err == nil {
		t.Fatal("工具退出码非零,回滚应报错")
	}
	// 失败行已由 Deployer 记下:worker 执行动作写事实面,失败也写。
	if rec.depCount() != 1 || rec.depRuns[0].Status != "rollback_failed" {
		t.Errorf("失败也应写一行 rollback_failed: %+v", rec.depRuns)
	}
}

/* ---------- payload 与路径校验 ---------- */

func TestHandlersRejectBadInput(t *testing.T) {
	root := t.TempDir()
	writeEnvYAML(t, root, "environments/dev.yaml", "dev", "forge", httpCheckYAML("http://127.0.0.1:1"))
	table := Handlers(Config{Root: root, Store: &stubRecorder{}, Docker: fakeDocker{}})
	ctx := context.Background()

	tests := []struct {
		name    string
		kind    jobs.Kind
		raw     json.RawMessage
		wantSub string
	}{
		{"坏 JSON", jobs.KindEnvironmentUp, json.RawMessage("{坏"), "payload"},
		{"缺 def", jobs.KindEnvironmentUp, payload(t, jobs.EnvActionPayload{Project: "forge"}), "缺少 def"},
		{"路径穿越", jobs.KindEnvironmentUp, payload(t, jobs.EnvActionPayload{Def: "../evil.yaml"}), "越出内容仓库边界"},
		{"绝对路径", jobs.KindEnvironmentDown, payload(t, jobs.EnvActionPayload{Def: "/etc/passwd"}), "必须是仓库相对路径"},
		{"定义不存在", jobs.KindEnvironmentRebuild, payload(t, jobs.EnvActionPayload{Def: "environments/nope.yaml"}), ""},
		{"回滚缺 def", jobs.KindDeployRollback, payload(t, jobs.DeployRollbackPayload{Digest: "x"}), "缺少 def"},
		{"部署缺 def", jobs.KindDeploy, payload(t, jobs.DeployPayload{Project: "forge"}), "缺少 def"},
		{"部署路径穿越", jobs.KindDeploy, payload(t, jobs.DeployPayload{Def: "../evil.yaml"}), "越出内容仓库边界"},
	}
	for _, tt := range tests {
		err := table[tt.kind](ctx, tt.raw)
		if err == nil {
			t.Errorf("%s: 应报错", tt.name)
			continue
		}
		if tt.wantSub != "" && !strings.Contains(err.Error(), tt.wantSub) {
			t.Errorf("%s: 错误应含 %q,得到 %q", tt.name, tt.wantSub, err.Error())
		}
	}
}

func TestHandlersSSHEnvWithoutSSHCfgRefused(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "environments", "rem.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := "name: rem\nproject: forge\nkind: ssh-script\n" +
		"ssh:\n  host: example.com\n  user: root\nscript: echo up\n" +
		"health_checks:\n  - name: t\n    type: tcp\n    host: 127.0.0.1\n    port: 1\n"
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	table := Handlers(Config{Root: root, Store: &stubRecorder{}, Docker: fakeDocker{}}) // NewSSH 为 nil

	err := table[jobs.KindEnvironmentUp](context.Background(),
		payload(t, jobs.EnvActionPayload{Def: "environments/rem.yaml"}))
	if err == nil || !strings.Contains(err.Error(), "SSH") {
		t.Errorf("未配置 SSH 执行能力应拒绝 ssh-script 环境: %v", err)
	}
}

func TestHandlersTableCoversDeclaredKinds(t *testing.T) {
	table := Handlers(Config{})
	for _, kind := range []jobs.Kind{
		jobs.KindEnvironmentUp, jobs.KindEnvironmentDown, jobs.KindEnvironmentRebuild,
		jobs.KindDeploy, jobs.KindDeployRollback, jobs.KindDiagnose,
	} {
		if _, ok := table[kind]; !ok {
			t.Errorf("执行端应注册 %s 的处理器", kind)
		}
	}
	if len(table) != 6 {
		t.Errorf("执行端应注册声明的全部 6 种 kind(环境三件 + 部署 + 回滚 + 诊断),得到 %d", len(table))
	}
}

// 验收(T10c):诊断处理器在缺指纹或依赖未接线时报清楚的错,不空跑、不 panic。
func TestDiagnoseHandlerGuards(t *testing.T) {
	table := Handlers(Config{})
	ctx := context.Background()

	if err := table[jobs.KindDiagnose](ctx, payload(t, jobs.DiagnosePayload{})); err == nil {
		t.Error("缺 fingerprint 应报错")
	}
	if err := table[jobs.KindDiagnose](ctx, payload(t, jobs.DiagnosePayload{Fingerprint: "abc123def456"})); err == nil {
		t.Error("依赖未接线(DB/KB/Executor/Runner 缺)应报错")
	}
}

/* ---------- T11e:自动诊断触发开关 ---------- */

// fakeEnqueuer 是脚本化的诊断入队出口:记录入队的项目/指纹,出错可配。
type fakeEnqueuer struct {
	mu    sync.Mutex
	calls []string
	err   error
}

func (f *fakeEnqueuer) EnqueueDiagnose(_ context.Context, project, fingerprint string) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, project+"\t"+fingerprint)
	return int64(len(f.calls)), f.err
}

// TestMaybeAutoDiagnoseSwitches:自动诊断入队的守卫条件(D-e1 默认关)。
func TestMaybeAutoDiagnoseSwitches(t *testing.T) {
	ctx := context.Background()
	fp := core.Fingerprint("fp-abc123")

	t.Run("开关关不入队", func(t *testing.T) {
		enq := &fakeEnqueuer{}
		Config{DiagnoseAuto: false, DiagnoseQueue: enq}.maybeAutoDiagnose(ctx, "forge", fp)
		if len(enq.calls) != 0 {
			t.Errorf("默认关不应自动入队: %v", enq.calls)
		}
	})
	t.Run("无入队出口不 panic", func(t *testing.T) {
		Config{DiagnoseAuto: true}.maybeAutoDiagnose(ctx, "forge", fp) // DiagnoseQueue 为 nil
	})
	t.Run("空指纹不入队", func(t *testing.T) {
		enq := &fakeEnqueuer{}
		Config{DiagnoseAuto: true, DiagnoseQueue: enq}.maybeAutoDiagnose(ctx, "forge", "")
		if len(enq.calls) != 0 {
			t.Errorf("空指纹(成功路径)不应入队: %v", enq.calls)
		}
	})
	t.Run("开关开且指纹非空则入队", func(t *testing.T) {
		enq := &fakeEnqueuer{}
		Config{DiagnoseAuto: true, DiagnoseQueue: enq}.maybeAutoDiagnose(ctx, "forge", fp)
		if len(enq.calls) != 1 || enq.calls[0] != "forge\tfp-abc123" {
			t.Errorf("应带项目与精确指纹自动入队: %v", enq.calls)
		}
	})
	t.Run("入队出错不 panic", func(t *testing.T) {
		enq := &fakeEnqueuer{err: errors.New("queue down")}
		Config{DiagnoseAuto: true, DiagnoseQueue: enq}.maybeAutoDiagnose(ctx, "forge", fp)
	})
}
