//go:build integration

package deploy

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"tally/internal/runner"
	"tally/internal/store"
)

// 集成测试:真实 docker compose + 真实 Postgres(不 mock,见 docs/05-test-standards.md)。
// 前置:已迁移。验证用的部署一律跑在隔离环境(独立项目名 + 独立 target_env),不碰共享测试环境。
//
// 测试镜像用自己的标签名 tally-t03-alpine:latest,不共享 environment 集成测试的
// alpine:latest —— go test 并行跑两个包,共享同一个 docker 标签会互相 rmi 打架。
const deployTestImage = "tally-t03-alpine:latest"

const deployComposeFixture = `services:
  app:
    image: ` + deployTestImage + `
    entrypoint: [""]
    command: ["sh", "-c", "echo deployed ${ARTIFACT_DIGEST:-none} on ${TARGET_ENV:-none}; sleep 60"]
`

const failingComposeFixture = `services:
  app:
    image: tally-t03-no-such-image:does-not-exist
`

func writeDeployFixture(t *testing.T, composeYAML, project, env, digest string) Definition {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "compose.yml"), []byte(composeYAML), 0o600); err != nil {
		t.Fatal(err)
	}
	defYAML := "project: " + project + "\n" +
		"target_env: " + env + "\n" +
		"tool: compose\n" +
		"artifact_digest: " + digest + "\n" +
		"config: ./compose.yml\n"
	path := filepath.Join(dir, "deploy.yaml")
	if err := os.WriteFile(path, []byte(defYAML), 0o600); err != nil {
		t.Fatal(err)
	}
	def, err := ParseDefinition(path)
	if err != nil {
		t.Fatalf("解析部署定义: %v", err)
	}
	return def
}

func openDeployStore(t *testing.T) *store.Store {
	t.Helper()
	return store.TestDB(t) // 每测试独立 schema(docs/05-test-standards.md)
}

// uniqueProject 每次运行独立项目名:事实面只追加清不掉,docker 资源也按它过滤残留。
func uniqueProject(t *testing.T, prefix string) string {
	t.Helper()
	return prefix + strconv.FormatInt(time.Now().UnixNano(), 10)
}

// ensureAlpineForDeploy 保证 deployTestImage 可用:本地有 alpine:latest 就直接打 tag(快,不走网络);
// 没有就先试着拉,拉不动再用本地 alpine 基础镜像打 tag。只动自己的标签名,尽量真跑而不是 skip。
func ensureAlpineForDeploy(t *testing.T) {
	t.Helper()
	if exec.Command("docker", "image", "inspect", deployTestImage).Run() == nil {
		return
	}
	if exec.Command("docker", "image", "inspect", "alpine:latest").Run() == nil {
		if err := exec.Command("docker", "tag", "alpine:latest", deployTestImage).Run(); err == nil {
			t.Cleanup(func() { _ = exec.Command("docker", "rmi", deployTestImage).Run() })
			return
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	if err := exec.CommandContext(ctx, "docker", "pull", "alpine:latest").Run(); err == nil {
		if err := exec.Command("docker", "tag", "alpine:latest", deployTestImage).Run(); err == nil {
			t.Cleanup(func() { _ = exec.Command("docker", "rmi", deployTestImage).Run() })
			return
		}
	}
	// 拉不动:用本地已有的 alpine 基础镜像打 tag(都自带 sh/sleep,够跑 sleep 容器)。
	for _, base := range []string{"postgres:16-alpine", "redis:8.2", "minio/minio:latest"} {
		if exec.Command("docker", "image", "inspect", base).Run() != nil {
			continue
		}
		if err := exec.Command("docker", "tag", base, deployTestImage).Run(); err == nil {
			t.Cleanup(func() { _ = exec.Command("docker", "rmi", deployTestImage).Run() })
			t.Logf("%s 用本地 %s 打 tag 代替(仍是 alpine 基础镜像)", deployTestImage, base)
			return
		}
	}
	t.Skip("无可用基础镜像且无法拉取,跳过集成测试(已明确报告)")
}

// composeDown 清掉某次部署的 compose 资源(测试收尾,不经 deploy 包:推倒不是部署动作)。
func composeDown(t *testing.T, def Definition) {
	t.Helper()
	_ = exec.Command("docker", "compose", "-f", def.ConfigPath(),
		"-p", def.Project+"-"+def.TargetEnv, "down", "--volumes", "--remove-orphans").Run()
}

// 验收:artifact_digest 非空约束在数据库层 —— 直接插 NULL 被拒。
func TestDeployRunArtifactDigestNotNullIntegration(t *testing.T) {
	st := openDeployStore(t)
	project := uniqueProject(t, "t03null")
	_, err := st.Pool().Exec(context.Background(),
		`INSERT INTO deploy_run (project_id, artifact_digest, target_env, tool, started_at, status, version_combo)
		 VALUES ($1, NULL, 'e', 'compose', now(), 'deployed', '{}')`, project)
	if err == nil {
		t.Fatal("数据库层应拒绝 artifact_digest 为 NULL 的 deploy_run")
	}
	if !strings.Contains(err.Error(), "artifact_digest") {
		t.Errorf("拒绝原因应指向 artifact_digest 非空约束: %v", err)
	}
}

// 验收:每次部署写一行,字段齐全(digest、目标、工具、结果、退出码、健康检查结果、版本组合)。
// 且验证用的重新部署跑在隔离环境:target_env 是独立值,compose 项目名随之隔离。
func TestDeployWritesFullRowIsolatedIntegration(t *testing.T) {
	ensureAlpineForDeploy(t)
	st := openDeployStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	project := uniqueProject(t, "t03dep")
	envName := "iso-" + strconv.FormatInt(time.Now().UnixNano(), 10) // 隔离环境,不是共享的 dev/test
	def := writeDeployFixture(t, deployComposeFixture, project, envName, "sha256:deadbeefcafe")
	d := &Deployer{Exec: runner.LocalRunner{}, Store: st.Queries()}
	t.Cleanup(func() { composeDown(t, def) })

	res, err := d.Deploy(ctx, def)
	if err != nil {
		t.Fatalf("部署应成功: %v", err)
	}
	if res.RunID == 0 || res.Status != StatusDeployed {
		t.Fatalf("应记一行 deployed: %+v", res)
	}

	var (
		digest, target, toolName, status string
		finished                         bool
		exitCode                         *int32
		hcOK                             bool
		combo                            []byte
	)
	err = st.Pool().QueryRow(ctx,
		`SELECT artifact_digest, target_env, tool, status, finished_at IS NOT NULL, exit_code,
		        COALESCE((health_check_result->>'ok')::bool, false), version_combo
		 FROM deploy_run WHERE id = $1`, res.RunID).
		Scan(&digest, &target, &toolName, &status, &finished, &exitCode, &hcOK, &combo)
	if err != nil {
		t.Fatalf("读回 deploy_run: %v", err)
	}
	if digest != "sha256:deadbeefcafe" || toolName != "compose" || status != "deployed" {
		t.Errorf("记账字段错: digest=%s tool=%s status=%s", digest, toolName, status)
	}
	if target != envName {
		t.Errorf("target_env 应是隔离环境 %s,实际 %s(不复用正在使用的测试环境)", envName, target)
	}
	if !finished {
		t.Error("成功部署应有 finished_at")
	}
	if exitCode == nil || *exitCode != 0 {
		t.Errorf("退出码应记 0: %v", exitCode)
	}
	if !hcOK {
		t.Error("成功时 health_check_result.ok 应是 true(工具退出码背书)")
	}
	var vc map[string]string
	if err := json.Unmarshal(combo, &vc); err != nil || len(vc) == 0 {
		t.Fatalf("version_combo 应是非空 JSON: %v", err)
	}
	if vc["artifact_digest"] != "sha256:deadbeefcafe" || vc["target_env"] != envName {
		t.Errorf("version_combo 应含制品 digest 与隔离目标: %v", vc)
	}

	// 隔离性:容器跑在 project-<isoenv> 名下,共享环境名下没有容器。
	iso := dockerNames(t, project+"-"+envName)
	if iso == "" {
		t.Error("隔离环境名下应有容器在跑")
	}
	if shared := dockerNames(t, project+"-main"); shared != "" {
		t.Errorf("共享环境名下不应有容器(隔离部署不碰它): %s", shared)
	}
}

// 验收:回滚记一行 rolled_back(调工具自带回滚,本包只转发)。
func TestRollbackWritesRowIntegration(t *testing.T) {
	ensureAlpineForDeploy(t)
	st := openDeployStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	project := uniqueProject(t, "t03rb")
	def := writeDeployFixture(t, deployComposeFixture, project,
		"iso-"+strconv.FormatInt(time.Now().UnixNano(), 10), "sha256:badbad")
	d := &Deployer{Exec: runner.LocalRunner{}, Store: st.Queries()}
	t.Cleanup(func() { composeDown(t, def) })

	if _, err := d.Deploy(ctx, def); err != nil {
		t.Fatalf("先部署: %v", err)
	}
	def.Digest = "sha256:previousgood" // 回滚到上一个能跑的制品
	res, err := d.Rollback(ctx, def)
	if err != nil {
		t.Fatalf("回滚应成功: %v", err)
	}

	var status string
	var comboDigest *string
	err = st.Pool().QueryRow(ctx,
		`SELECT status, version_combo->>'artifact_digest' FROM deploy_run WHERE id = $1`, res.RunID).
		Scan(&status, &comboDigest)
	if err != nil {
		t.Fatalf("读回回滚的 deploy_run: %v", err)
	}
	if status != "rolled_back" {
		t.Errorf("回滚成功应记 rolled_back: %s", status)
	}
	if comboDigest == nil || *comboDigest != "sha256:previousgood" {
		t.Errorf("回滚行的版本组合应指向回滚目标 digest: %v", comboDigest)
	}

	// 一行部署 + 一行回滚,两次事实都在(只追加)。
	var n int
	if err := st.Pool().QueryRow(ctx,
		`SELECT count(*) FROM deploy_run WHERE project_id = $1`, project).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("应有部署 + 回滚两行事实: got %d", n)
	}
}

// 验收:失败输出送 logpipe,source=deploy —— 记进 health_check_result.logpipe。
func TestDeployFailureFeedsLogpipeSourceDeployIntegration(t *testing.T) {
	st := openDeployStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	project := uniqueProject(t, "t03fail")
	def := writeDeployFixture(t, failingComposeFixture, project,
		"iso-"+strconv.FormatInt(time.Now().UnixNano(), 10), "sha256:whatever")
	d := &Deployer{Exec: runner.LocalRunner{}, Store: st.Queries()}
	t.Cleanup(func() { composeDown(t, def) })

	_, err := d.Deploy(ctx, def)
	if err == nil {
		t.Fatal("镜像不存在,部署应失败")
	}
	if !strings.Contains(err.Error(), "指纹") {
		t.Errorf("error 应带 logpipe 指纹: %v", err)
	}

	var status string
	var lpSource, exactFP *string
	err = st.Pool().QueryRow(ctx,
		`SELECT status, health_check_result->'logpipe'->>'source', health_check_result->'logpipe'->>'exact_fp'
		 FROM deploy_run WHERE project_id = $1 ORDER BY id DESC LIMIT 1`, project).
		Scan(&status, &lpSource, &exactFP)
	if err != nil {
		t.Fatalf("读回失败的 deploy_run: %v", err)
	}
	if status != "failed" {
		t.Errorf("status 应是 failed: %s", status)
	}
	if lpSource == nil || *lpSource != "deploy" {
		t.Errorf("失败输出的 logpipe 来源应标为 deploy: %v", lpSource)
	}
	if exactFP == nil || *exactFP == "" {
		t.Error("失败时应把 logpipe 精确指纹记进 health_check_result")
	}
}

func dockerNames(t *testing.T, composeProject string) string {
	t.Helper()
	out, err := exec.Command("docker", "ps", "--filter", "name="+composeProject,
		"--format", "{{.Names}}").Output()
	if err != nil {
		t.Fatalf("docker ps: %v", err)
	}
	return strings.TrimSpace(string(out))
}
