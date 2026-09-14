//go:build integration

package environment

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

// 集成测试:用 alpine 起 sleep 容器,走 拉起 → 健康检查(log-pattern)→ 记录 → 推倒 全流程。
// 用真实 docker 与真实 Postgres(不 mock,见 docs/05-test-standards.md)。前置:已迁移。

const composeFixture = `services:
  sleeper:
    image: alpine:latest
    entrypoint: [""]
    command: ["sh", "-c", "echo integ-ready-marker; sleep 300"]
    volumes:
      - scratch:/data
volumes:
  scratch:
`

// writeFixture 在临时目录写 compose 文件与环境定义,返回解析好的 Definition。
func writeFixture(t *testing.T, project, logPattern string, waitTimeout string) Definition {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "compose.yml"), []byte(composeFixture), 0o600); err != nil {
		t.Fatal(err)
	}
	defYAML := "name: dev\n" +
		"project: " + project + "\n" +
		"kind: docker-compose\n" +
		"compose_file: ./compose.yml\n" +
		"project_name: " + project + "\n" +
		"health_checks:\n" +
		"  - name: sleeper-log\n" +
		"    type: log-pattern\n" +
		"    service: sleeper\n" +
		"    pattern: " + logPattern + "\n" +
		"wait:\n" +
		"  timeout: " + waitTimeout + "\n" +
		"  interval: 1s\n"
	path := filepath.Join(dir, "dev.yaml")
	if err := os.WriteFile(path, []byte(defYAML), 0o600); err != nil {
		t.Fatal(err)
	}
	def, err := ParseDefinition(path)
	if err != nil {
		t.Fatalf("解析定义: %v", err)
	}
	return def
}

func openEnvStore(t *testing.T) *store.Store {
	t.Helper()
	return store.TestDB(t) // 每测试独立 schema(docs/05-test-standards.md)
}

// uniqueProject 每次运行用独立项目名:事实面只追加清不掉,docker 资源也按它过滤残留。
// 名字须是合法 compose 项目名(小写字母数字与连字符):故用 UnixNano 纯数字后缀,不带点。
func uniqueProject(t *testing.T) string {
	t.Helper()
	return "t01env-" + strconv.FormatInt(time.Now().UnixNano(), 10)
}

// ensureAlpine 保证 alpine:latest 可用。Docker Hub 受限时用本地 alpine 基础镜像打 tag 代替,
// 尽量让测试真跑而不是 skip。
func ensureAlpine(t *testing.T) {
	t.Helper()
	if imageExists("alpine:latest") {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	if err := exec.CommandContext(ctx, "docker", "pull", "alpine:latest").Run(); err == nil {
		return
	}
	// 拉不动:用一个本地已有的 alpine 基础镜像打 tag(都自带 sh/sleep,够跑 sleep 容器)。
	for _, base := range []string{"postgres:16-alpine", "redis:8.2", "minio/minio:latest"} {
		if !imageExists(base) {
			continue
		}
		if err := exec.Command("docker", "tag", base, "alpine:latest").Run(); err == nil {
			t.Cleanup(func() { _ = exec.Command("docker", "rmi", "alpine:latest").Run() })
			t.Logf("alpine:latest 拉取受网络限制,用本地 %s 打 tag 代替(仍是 alpine 基础镜像)", base)
			return
		}
	}
	t.Skip("无 alpine 镜像且无法拉取或打 tag,跳过集成测试(已明确报告)")
}

func imageExists(ref string) bool {
	return exec.Command("docker", "image", "inspect", ref).Run() == nil
}

// assertNoResidue 断言推倒后该项目不残留卷、网络、容器。
func assertNoResidue(t *testing.T, project string) {
	t.Helper()
	checks := []struct {
		kind string
		args []string
	}{
		{"volume", []string{"volume", "ls", "--filter", "name=" + project, "--format", "{{.Name}}"}},
		{"network", []string{"network", "ls", "--filter", "name=" + project, "--format", "{{.Name}}"}},
		{"container", []string{"ps", "-a", "--filter", "name=" + project, "--format", "{{.Names}}"}},
	}
	for _, c := range checks {
		out, err := exec.Command("docker", c.args...).Output()
		if err != nil {
			t.Fatalf("docker %s ls: %v", c.kind, err)
		}
		if s := strings.TrimSpace(string(out)); s != "" {
			t.Errorf("推倒后残留 %s: %s", c.kind, s)
		}
	}
}

// 验收主流程:alpine 起 sleep 容器 → 拉起 → log-pattern 健康检查 → 记录 environment_run → 推倒不残留。
func TestEnvironmentLifecycleIntegration(t *testing.T) {
	ensureAlpine(t)
	st := openEnvStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	project := uniqueProject(t)
	def := writeFixture(t, project, "integ-ready-marker", "120s")
	m := &Manager{Docker: runner.NewDockerRunner(), Store: st.Queries()}

	// —— 拉起 ——
	res, err := m.Up(ctx, def)
	if err != nil {
		t.Fatalf("拉起应成功: %v", err)
	}
	if res.RunID == 0 {
		t.Fatal("应返回 environment_run id")
	}
	t.Cleanup(func() { _ = m.Down(context.Background(), def) }) // 兜底清理,别让容器泄漏

	// —— 记录:status=up,finished_at,health ok,version_combo 含镜像 tag + 运行时版本 ——
	var (
		status   string
		finished bool
		hcOK     bool
		defHash  string
		combo    []byte
	)
	err = st.Pool().QueryRow(ctx,
		`SELECT status, finished_at IS NOT NULL, COALESCE((health_check_result->>'ok')::bool, false),
		        definition_hash, version_combo
		 FROM environment_run WHERE id = $1`, res.RunID).
		Scan(&status, &finished, &hcOK, &defHash, &combo)
	if err != nil {
		t.Fatalf("读回 environment_run: %v", err)
	}
	if status != "up" {
		t.Errorf("status 应是 up: %s", status)
	}
	if !finished {
		t.Error("成功拉起应有 finished_at")
	}
	if !hcOK {
		t.Error("health_check_result.ok 应是 true")
	}
	// definition_hash = 定义文件内容哈希(INV-4)。
	if defHash != def.Hash {
		t.Errorf("落库 definition_hash 应等于定义内容哈希: %s vs %s", defHash, def.Hash)
	}
	var vc map[string]string
	if err := json.Unmarshal(combo, &vc); err != nil {
		t.Fatalf("version_combo 应是合法 JSON: %v", err)
	}
	if len(vc) == 0 {
		t.Fatal("version_combo 不得为空")
	}
	if vc["docker"] == "" {
		t.Errorf("version_combo 应含 docker 运行时版本: %+v", vc)
	}
	if !comboHasImageTag(vc, "alpine:latest") {
		t.Errorf("version_combo 应含镜像 tag alpine:latest: %+v", vc)
	}

	// —— definition_hash 随内容变化(INV-4):改一个字节,哈希必须不同 ——
	def2 := writeFixture(t, project, "integ-ready-marker-CHANGED", "120s")
	if def2.Hash == def.Hash {
		t.Error("改了定义内容,definition_hash 必须变(INV-4)")
	}

	// —— 推倒:不残留卷/网络/容器 ——
	if err := m.Down(ctx, def); err != nil {
		t.Fatalf("推倒: %v", err)
	}
	assertNoResidue(t, project)
}

// 推倒重建:Down 再 Up,还能起来,且不残留。
func TestEnvironmentRebuildIntegration(t *testing.T) {
	ensureAlpine(t)
	st := openEnvStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	project := uniqueProject(t)
	def := writeFixture(t, project, "integ-ready-marker", "120s")
	m := &Manager{Docker: runner.NewDockerRunner(), Store: st.Queries()}

	if _, err := m.Rebuild(ctx, def); err != nil {
		t.Fatalf("首次重建(= 拉起): %v", err)
	}
	t.Cleanup(func() { _ = m.Down(context.Background(), def) })
	// 再来一次:先推倒已有的,再拉起。
	res, err := m.Rebuild(ctx, def)
	if err != nil {
		t.Fatalf("二次重建应成功: %v", err)
	}
	if res.RunID == 0 {
		t.Error("重建应产生新的 environment_run id")
	}
	if err := m.Down(ctx, def); err != nil {
		t.Fatalf("收尾推倒: %v", err)
	}
	assertNoResidue(t, project)
}

// 失败路径:健康检查一直不过 → Up 返回 error,记一行 status=failed,
// 且原始输出过了 logpipe(health_check_result 里有指纹)。
func TestEnvironmentFailureFeedsLogpipeIntegration(t *testing.T) {
	ensureAlpine(t)
	st := openEnvStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	project := uniqueProject(t)
	// pattern 永远匹配不到 → 健康检查超时失败。
	def := writeFixture(t, project, "this-pattern-never-appears-xyz", "5s")
	m := &Manager{Docker: runner.NewDockerRunner(), Store: st.Queries()}

	_, err := m.Up(ctx, def)
	t.Cleanup(func() { _ = m.Down(context.Background(), def) })
	if err == nil {
		t.Fatal("健康检查不过应返回 error")
	}
	if !strings.Contains(err.Error(), "指纹") {
		t.Errorf("error 应带 logpipe 指纹: %v", err)
	}

	var status string
	var exactFP *string
	err = st.Pool().QueryRow(ctx,
		`SELECT status, health_check_result->'logpipe'->>'exact_fp'
		 FROM environment_run WHERE project_id = $1 ORDER BY id DESC LIMIT 1`, project).
		Scan(&status, &exactFP)
	if err != nil {
		t.Fatalf("读回失败的 environment_run: %v", err)
	}
	if status != "failed" {
		t.Errorf("status 应是 failed: %s", status)
	}
	if exactFP == nil || *exactFP == "" {
		t.Error("失败时应把 logpipe 指纹记进 health_check_result")
	}
}

// comboHasImageTag 判断 version_combo 里是否记录了某镜像 tag(值相等即可,键名不拘)。
func comboHasImageTag(vc map[string]string, tag string) bool {
	for _, v := range vc {
		if v == tag {
			return true
		}
	}
	return false
}
