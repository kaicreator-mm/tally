//go:build integration

package exec

// 集成测试:执行端接真 Postgres —— 给一个 job,真调 environment/deploy 并写事实面。
// docker/命令执行用假实现(执行逻辑的单测已覆盖命令构造),Postgres 是真的(不 mock 数据库)。
// 每个测试用 store.TestDB 拿独立 schema(见 docs/05-test-standards.md)。
//
// 项目身份(§33.3 ghost-project guard):动作写事实面前先查派生项目注册表 project_index。
// 正向用例先经 registerProject 登记(模拟 Git 注册表派生入库),未知项目用例断言动作
// 不执行、一行事实不落,注册表不可查时断言 fail-closed。

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tally/internal/jobs"
	"tally/internal/store"
)

func openStore(t *testing.T) *store.Store {
	t.Helper()
	return store.TestDB(t)
}

// registerProject 把项目登记进派生项目注册表(模拟注册表派生的增量入库路径)。
func registerProject(t *testing.T, st *store.Store, projectID string) {
	t.Helper()
	if err := st.Queries().UpsertProjectIndex(context.Background(),
		store.UpsertProjectIndexParams{ProjectID: projectID, SourceCommit: "itest-registry"}); err != nil {
		t.Fatalf("登记项目 %s 进派生注册表: %v", projectID, err)
	}
}

func TestEnvUpJobWritesRealFactRow(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()

	health := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(health.Close)

	project := "exec-itest-up" // 固定项目名:重跑时写新行,不影响断言(数增量)
	registerProject(t, st, project)
	root := t.TempDir()
	writeEnvYAML(t, root, "environments/dev.yaml", "dev", project, httpCheckYAML(health.URL))

	table := Handlers(Config{Root: root, Store: st.Queries(), DB: st, Docker: fakeDocker{}})

	var before int64
	if err := st.Pool().QueryRow(ctx,
		"SELECT count(*) FROM environment_run WHERE project_id = $1", project).Scan(&before); err != nil {
		t.Fatalf("数 environment_run: %v", err)
	}

	err := table[jobs.KindEnvironmentUp](ctx,
		payload(t, jobs.EnvActionPayload{Project: project, Name: "dev", Def: "environments/dev.yaml"}))
	if err != nil {
		t.Fatalf("拉起 job 应成功: %v", err)
	}

	var after int64
	if err := st.Pool().QueryRow(ctx,
		"SELECT count(*) FROM environment_run WHERE project_id = $1", project).Scan(&after); err != nil {
		t.Fatalf("数 environment_run: %v", err)
	}
	if after != before+1 {
		t.Fatalf("执行端应给真库写 1 行 environment_run: before=%d after=%d", before, after)
	}
	var status, defHash string
	if err := st.Pool().QueryRow(ctx,
		"SELECT status, definition_hash FROM environment_run WHERE project_id = $1 ORDER BY id DESC LIMIT 1",
		project).Scan(&status, &defHash); err != nil {
		t.Fatalf("读回事实行: %v", err)
	}
	if status != "up" || defHash == "" {
		t.Errorf("事实行应是 up 且带定义哈希: status=%q hash=%q", status, defHash)
	}
}

func TestRollbackJobWritesRealFactRow(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()

	project := "exec-itest-rb"
	registerProject(t, st, project)
	root := t.TempDir()
	writeDeployYAML(t, root, "deploys/prod.yaml", project, "prod")

	table := Handlers(Config{Root: root, Store: st.Queries(), DB: st, Docker: fakeDocker{}, Exec: &fakeCmd{exit: 0}})

	var before int64
	if err := st.Pool().QueryRow(ctx,
		"SELECT count(*) FROM deploy_run WHERE project_id = $1", project).Scan(&before); err != nil {
		t.Fatalf("数 deploy_run: %v", err)
	}

	err := table[jobs.KindDeployRollback](ctx, payload(t, jobs.DeployRollbackPayload{
		Project: project, TargetEnv: "prod", Digest: "sha256:good1234", Def: "deploys/prod.yaml",
	}))
	if err != nil {
		t.Fatalf("回滚 job 应成功: %v", err)
	}

	var after int64
	if err := st.Pool().QueryRow(ctx,
		"SELECT count(*) FROM deploy_run WHERE project_id = $1", project).Scan(&after); err != nil {
		t.Fatalf("数 deploy_run: %v", err)
	}
	if after != before+1 {
		t.Fatalf("执行端应给真库写 1 行 deploy_run: before=%d after=%d", before, after)
	}
	var status, digest string
	if err := st.Pool().QueryRow(ctx,
		"SELECT status, artifact_digest FROM deploy_run WHERE project_id = $1 ORDER BY id DESC LIMIT 1",
		project).Scan(&status, &digest); err != nil {
		t.Fatalf("读回事实行: %v", err)
	}
	if status != "rolled_back" || digest != "sha256:good1234" {
		t.Errorf("事实行应是 rolled_back 且 digest 来自 payload: status=%q digest=%q", status, digest)
	}
}

// TestEnvUpFailureIngestsFailureEvent(T10b):拉起失败 → exec 编排层经 ingest 录一行
// failure_event(source=environment, environment_run_id 指回失败行),指纹与
// environment_run.health_check_result 里 logpipe 的产出一致(同一次失败同一指纹)。
func TestEnvUpFailureIngestsFailureEvent(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()

	project := "exec-itest-upfail"
	registerProject(t, st, project)
	root := t.TempDir()
	writeEnvYAML(t, root, "environments/dev.yaml", "dev", project, httpCheckYAML("http://127.0.0.1:1/never"))

	table := Handlers(Config{Root: root, Store: st.Queries(), DB: st, Docker: fakeDocker{upExit: 1}})

	err := table[jobs.KindEnvironmentUp](ctx,
		payload(t, jobs.EnvActionPayload{Project: project, Name: "dev", Def: "environments/dev.yaml"}))
	if err == nil {
		t.Fatal("拉起失败时 job 应报错")
	}
	if !strings.Contains(err.Error(), "指纹") {
		t.Errorf("错误文本应保留 logpipe 指纹: %v", err)
	}

	// environment_run 失败行已记,取出它的 logpipe 指纹与行 id。
	var runID int64
	var envFP string
	if err := st.Pool().QueryRow(ctx,
		`SELECT id, health_check_result->'logpipe'->>'exact_fp'
		 FROM environment_run WHERE project_id = $1 ORDER BY id DESC LIMIT 1`, project,
	).Scan(&runID, &envFP); err != nil {
		t.Fatalf("读 environment_run 失败行: %v", err)
	}
	if envFP == "" {
		t.Fatal("environment_run.health_check_result 应记 logpipe 指纹")
	}

	// failure_event:source=environment,来源外键指回同一 run,指纹一致。
	var source string
	var feRunID int64
	var feFP string
	if err := st.Pool().QueryRow(ctx,
		"SELECT source, environment_run_id, fp_exact FROM failure_event WHERE project_id = $1", project,
	).Scan(&source, &feRunID, &feFP); err != nil {
		t.Fatalf("failure_event 应被 ingest 录入: %v", err)
	}
	if source != "environment" || feRunID != runID {
		t.Errorf("failure_event 应 source=environment 且 environment_run_id=%d: source=%q id=%d", runID, source, feRunID)
	}
	if feFP != envFP {
		t.Errorf("failure_event 指纹应与 environment_run 一致: fe=%s env=%s", feFP, envFP)
	}

	// T11e/D-e1:开关缺省关 → 失败录入后队列不出现 diagnose 任务。
	if n := countDiagnoseJobsFor(t, st, feFP); n != 0 {
		t.Errorf("开关默认关,不应自动入队诊断,有 %d 个", n)
	}
}

// TestEnvUpSuccessNoFailureEvent(T10b):成功拉起不录 failure_event(只录失败,PRD)。
func TestEnvUpSuccessNoFailureEvent(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()

	health := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(health.Close)

	project := "exec-itest-upok"
	registerProject(t, st, project)
	root := t.TempDir()
	writeEnvYAML(t, root, "environments/dev.yaml", "dev", project, httpCheckYAML(health.URL))

	table := Handlers(Config{Root: root, Store: st.Queries(), DB: st, Docker: fakeDocker{}})
	if err := table[jobs.KindEnvironmentUp](ctx,
		payload(t, jobs.EnvActionPayload{Project: project, Name: "dev", Def: "environments/dev.yaml"})); err != nil {
		t.Fatalf("拉起 job 应成功: %v", err)
	}

	var n int
	if err := st.Pool().QueryRow(ctx,
		"SELECT count(*) FROM failure_event WHERE project_id = $1", project).Scan(&n); err != nil {
		t.Fatalf("数 failure_event: %v", err)
	}
	if n != 0 {
		t.Errorf("成功拉起不应录 failure_event,有 %d 行", n)
	}
}

// TestRollbackFailureIngestsFailureEvent(T10b):回滚失败 → failure_event(source=deploy,
// deploy_run_id 指回 rollback_failed 行),指纹与 deploy_run.health_check_result 一致。
func TestRollbackFailureIngestsFailureEvent(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()

	project := "exec-itest-rbfail"
	registerProject(t, st, project)
	root := t.TempDir()
	writeDeployYAML(t, root, "deploys/prod.yaml", project, "prod")

	table := Handlers(Config{Root: root, Store: st.Queries(), DB: st, Docker: fakeDocker{}, Exec: &fakeCmd{exit: 1}})

	err := table[jobs.KindDeployRollback](ctx, payload(t, jobs.DeployRollbackPayload{
		Project: project, TargetEnv: "prod", Digest: "sha256:bad5678", Def: "deploys/prod.yaml",
	}))
	if err == nil {
		t.Fatal("回滚失败时 job 应报错")
	}

	var runID int64
	var deployFP, status string
	if err := st.Pool().QueryRow(ctx,
		`SELECT id, status, health_check_result->'logpipe'->>'exact_fp'
		 FROM deploy_run WHERE project_id = $1 ORDER BY id DESC LIMIT 1`, project,
	).Scan(&runID, &status, &deployFP); err != nil {
		t.Fatalf("读 deploy_run 失败行: %v", err)
	}
	if status != "rollback_failed" {
		t.Errorf("deploy_run 应是 rollback_failed: %q", status)
	}
	if deployFP == "" {
		t.Fatal("deploy_run.health_check_result 应记 logpipe 指纹")
	}

	var source string
	var feRunID int64
	var feFP string
	if err := st.Pool().QueryRow(ctx,
		"SELECT source, deploy_run_id, fp_exact FROM failure_event WHERE project_id = $1", project,
	).Scan(&source, &feRunID, &feFP); err != nil {
		t.Fatalf("failure_event 应被 ingest 录入: %v", err)
	}
	if source != "deploy" || feRunID != runID {
		t.Errorf("failure_event 应 source=deploy 且 deploy_run_id=%d: source=%q id=%d", runID, source, feRunID)
	}
	if feFP != deployFP {
		t.Errorf("failure_event 指纹应与 deploy_run 一致: fe=%s deploy=%s", feFP, deployFP)
	}
}

// TestEnvUpFailureAutoEnqueuesDiagnose(T11e):开关打开时,环境拉起失败录入后
// 按失败指纹自动入队一个 diagnose 任务(D-e4:与手动重诊断同一条队列入队路径)。
func TestEnvUpFailureAutoEnqueuesDiagnose(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()

	project := "exec-itest-autodiag"
	registerProject(t, st, project)
	root := t.TempDir()
	writeEnvYAML(t, root, "environments/dev.yaml", "dev", project, httpCheckYAML("http://127.0.0.1:1/never"))

	table := Handlers(Config{
		Root: root, Store: st.Queries(), DB: st, Docker: fakeDocker{upExit: 1},
		DiagnoseAuto: true, DiagnoseQueue: jobs.New(st, nil),
	})

	if err := table[jobs.KindEnvironmentUp](ctx,
		payload(t, jobs.EnvActionPayload{Project: project, Name: "dev", Def: "environments/dev.yaml"})); err == nil {
		t.Fatal("拉起失败时 job 应报错")
	}

	// 失败指纹以 failure_event 落库行为准;入队的 diagnose payload 应含同一指纹。
	var fp string
	if err := st.Pool().QueryRow(ctx,
		"SELECT fp_exact FROM failure_event WHERE project_id = $1", project).Scan(&fp); err != nil {
		t.Fatalf("failure_event 应被 ingest 录入: %v", err)
	}
	if n := countDiagnoseJobsFor(t, st, fp); n != 1 {
		t.Errorf("应按失败指纹自动入队 1 个 diagnose 任务,有 %d 个", n)
	}
}

// TestRollbackFailureAutoEnqueuesDiagnose(T11e):部署路径同样 —— 开关打开时,
// 回滚失败录入后按指纹自动入队诊断。
func TestRollbackFailureAutoEnqueuesDiagnose(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()

	project := "exec-itest-rb-auto"
	registerProject(t, st, project)
	root := t.TempDir()
	writeDeployYAML(t, root, "deploys/prod.yaml", project, "prod")

	table := Handlers(Config{
		Root: root, Store: st.Queries(), DB: st, Docker: fakeDocker{}, Exec: &fakeCmd{exit: 1},
		DiagnoseAuto: true, DiagnoseQueue: jobs.New(st, nil),
	})

	if err := table[jobs.KindDeployRollback](ctx, payload(t, jobs.DeployRollbackPayload{
		Project: project, TargetEnv: "prod", Digest: "sha256:bad5678", Def: "deploys/prod.yaml",
	})); err == nil {
		t.Fatal("回滚失败时 job 应报错")
	}

	var fp string
	if err := st.Pool().QueryRow(ctx,
		"SELECT fp_exact FROM failure_event WHERE project_id = $1", project).Scan(&fp); err != nil {
		t.Fatalf("failure_event 应被 ingest 录入: %v", err)
	}
	if n := countDiagnoseJobsFor(t, st, fp); n != 1 {
		t.Errorf("应按失败指纹自动入队 1 个 diagnose 任务,有 %d 个", n)
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

// TestUnknownProjectActionRejectedBeforeFact(§33.3 ghost-project guard 验收):
// 项目不在派生注册表 → 环境动作 / 部署动作直接拒绝,执行体根本不启动,
// environment_run / deploy_run 一行都不落(拒收发生在事实 INSERT 之前)。
func TestUnknownProjectActionRejectedBeforeFact(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()

	// 环境:拉起拒绝 → 不写 environment_run。
	envProject := "exec-itest-ghost-env" // 刻意不登记
	root := t.TempDir()
	writeEnvYAML(t, root, "environments/dev.yaml", "dev", envProject, httpCheckYAML("http://127.0.0.1:1/never"))
	rec := &stubRecorder{}
	table := Handlers(Config{Root: root, Store: rec, DB: st, Docker: fakeDocker{upExit: 1}})

	err := table[jobs.KindEnvironmentUp](ctx,
		payload(t, jobs.EnvActionPayload{Project: envProject, Name: "dev", Def: "environments/dev.yaml"}))
	if err == nil {
		t.Fatal("未知项目的环境动作应被拒绝")
	}
	if !strings.Contains(err.Error(), envProject) {
		t.Errorf("错误应点名未知项目 %q: %v", envProject, err)
	}
	if rec.envCount() != 0 {
		t.Errorf("未知项目应一行 environment_run 都不落,落了 %d 行", rec.envCount())
	}

	// 部署:拒绝且部署工具从未被调 → 不写 deploy_run。
	depProject := "exec-itest-ghost-dep" // 刻意不登记
	root = t.TempDir()
	writeDeployYAML(t, root, "deploys/prod.yaml", depProject, "prod")
	depRec := &stubRecorder{}
	cmd := &fakeCmd{exit: 0}
	table = Handlers(Config{Root: root, Store: depRec, DB: st, Docker: fakeDocker{}, Exec: cmd})

	err = table[jobs.KindDeploy](ctx, payload(t, jobs.DeployPayload{Project: depProject, Def: "deploys/prod.yaml"}))
	if err == nil {
		t.Fatal("未知项目的部署动作应被拒绝")
	}
	if !strings.Contains(err.Error(), depProject) {
		t.Errorf("错误应点名未知项目 %q: %v", depProject, err)
	}
	if len(cmd.cmds) != 0 {
		t.Errorf("未知项目不应启动部署工具,调了 %d 次", len(cmd.cmds))
	}
	if depRec.depCount() != 0 {
		t.Errorf("未知项目应一行 deploy_run 都不落,落了 %d 行", depRec.depCount())
	}

	// 已落库的历史事实不受 guard 影响(guard 只挡新事实,INV-5)。
	var total int64
	if err := st.Pool().QueryRow(ctx, "SELECT count(*) FROM environment_run").Scan(&total); err != nil {
		t.Fatalf("数 environment_run: %v", err)
	}
	if total != 0 {
		t.Errorf("guard 不应改动任何事实行,却有 %d 行 environment_run", total)
	}
}

// TestRegistryFailureFailClosed(§33.3 验收):注册表(project_index)不可查 →
// 动作 fail-closed 拒绝执行:执行体不启动,环境 / 部署事实一行不落。
func TestRegistryFailureFailClosed(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	st.Close() // 注册表不可查

	// 环境:拉起拒绝,stub 记录器零行。
	root := t.TempDir()
	writeEnvYAML(t, root, "environments/dev.yaml", "dev", "forge", httpCheckYAML("http://127.0.0.1:1/never"))
	rec := &stubRecorder{}
	table := Handlers(Config{Root: root, Store: rec, DB: st, Docker: fakeDocker{}})

	err := table[jobs.KindEnvironmentUp](ctx,
		payload(t, jobs.EnvActionPayload{Project: "forge", Name: "dev", Def: "environments/dev.yaml"}))
	if err == nil {
		t.Fatal("注册表不可查时应 fail-closed 拒绝执行")
	}
	if !strings.Contains(err.Error(), "fail-closed") {
		t.Errorf("错误应标明 fail-closed 拒绝: %v", err)
	}
	if rec.envCount() != 0 {
		t.Errorf("fail-closed 应零事实行,落了 %d 行 environment_run", rec.envCount())
	}

	// 部署:拒绝且工具从未被调。
	root = t.TempDir()
	writeDeployYAML(t, root, "deploys/prod.yaml", "forge", "prod")
	depRec := &stubRecorder{}
	cmd := &fakeCmd{exit: 0}
	table = Handlers(Config{Root: root, Store: depRec, DB: st, Docker: fakeDocker{}, Exec: cmd})

	err = table[jobs.KindDeploy](ctx, payload(t, jobs.DeployPayload{Def: "deploys/prod.yaml"}))
	if err == nil {
		t.Fatal("注册表不可查时应 fail-closed 拒绝执行")
	}
	if !strings.Contains(err.Error(), "fail-closed") {
		t.Errorf("错误应标明 fail-closed 拒绝: %v", err)
	}
	if len(cmd.cmds) != 0 {
		t.Errorf("fail-closed 不应启动部署工具,调了 %d 次", len(cmd.cmds))
	}
	if depRec.depCount() != 0 {
		t.Errorf("fail-closed 应零事实行,落了 %d 行 deploy_run", depRec.depCount())
	}
}
