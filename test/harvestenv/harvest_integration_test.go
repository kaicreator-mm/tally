//go:build integration

// Package harvestenv 是 harvest 的端到端集成测试:把 internal/environment 的拉起能力
// 经 harvest.Verifier 注入,用真实 docker + Postgres + git 跑「提案 → 实际拉起 → 健康检查 →
// 写 catalog」全流程。
//
// 放在 test/ 而非 internal/harvest 内,是为了让 internal/harvest 保持「不 import environment」
// (业务包互不导入,协作由上层编排)——这里的桥接正扮演那个「上层」角色。
package harvestenv

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"gopkg.in/yaml.v3"

	"tally/internal/airuntime"
	"tally/internal/environment"
	"tally/internal/harvest"
	"tally/internal/runner"
	"tally/internal/store"
)

// compose 夹具:alpine 起 sleep 容器,启动时打一个标记行,供 log-pattern 健康检查匹配。
const composeFixture = `services:
  sleeper:
    image: alpine:latest
    entrypoint: [""]
    command: ["sh", "-c", "echo harvest-ready-marker; sleep 300"]
`

// verifiedJSON 镜像 catalog/<svc>/verified.json 的结构(harvest 内部类型不导出)。
type verifiedJSON struct {
	VerifiedAt time.Time         `json:"verified_at"`
	Combo      map[string]string `json:"version_combo"`
	Record     string            `json:"record"`
	Service    string            `json:"service"`
}

// newFakeExecutor 返回预设 SUCCEEDED 的 AI Runtime 测试执行器(ADR-0002 之后
// 提案推理经 airuntime.Executor;与 internal/harvest 单测同构)。
func newFakeExecutor(out json.RawMessage) *airuntime.FakeExecutor {
	return &airuntime.FakeExecutor{
		Result: airuntime.Result{
			OperationID:       "op-harvest-integ",
			Status:            airuntime.StatusSucceeded,
			TerminationReason: airuntime.TerminationCompleted,
			Output:            out,
		},
	}
}

// envVerifier 把 environment.Manager 适配成 harvest.Verifier(与 cmd/tally 同形的桥接)。
type envVerifier struct{ mgr *environment.Manager }

func (v *envVerifier) Verify(ctx context.Context, def harvest.Definition) (harvest.VerifyResult, error) {
	dir, err := os.MkdirTemp("", "harvest-integ-*")
	if err != nil {
		return harvest.VerifyResult{}, err
	}
	defer os.RemoveAll(dir)
	composePath := filepath.Join(dir, "compose.yaml")
	if err := os.WriteFile(composePath, []byte(def.Compose), 0o600); err != nil {
		return harvest.VerifyResult{}, err
	}
	ydef := def
	ydef.ComposeFile = composePath
	if ydef.Project == "" {
		ydef.Project = "harvest"
	}
	ydef.ProjectName = "harvest-" + strings.ReplaceAll(strings.ToLower(def.Name), "_", "-")
	raw, err := yaml.Marshal(ydef)
	if err != nil {
		return harvest.VerifyResult{}, err
	}
	envDef, err := environment.ParseDefinitionBytes(raw)
	if err != nil {
		return harvest.VerifyResult{}, err
	}
	res, err := v.mgr.Up(ctx, envDef)
	if err != nil {
		return harvest.VerifyResult{}, err
	}
	defer func() { _ = v.mgr.Down(context.Background(), envDef) }() //nolint:contextcheck // 清理须独立于可能已取消的 ctx

	ok := 0
	for _, r := range res.Health.Results {
		if r.OK {
			ok++
		}
	}
	return harvest.VerifyResult{
		VerifiedAt: time.Now(),
		Combo:      res.Combo,
		Record:     fmt.Sprintf("health=%d/%dok", ok, len(res.Health.Results)),
	}, nil
}

// 验收(端到端):提案(含健康检查)→ 真实 alpine 拉起跑 log-pattern → 通过 → 写 catalog
// (definition.yaml + compose.yaml + verified.json + git commit),verified.json 含完整版本组合。
func TestHarvestEndToEndIntegration(t *testing.T) {
	ensureAlpine(t)
	st := openStore(t)
	repo := initRepo(t)
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	mgr := &environment.Manager{Docker: runner.NewDockerRunner(), Store: st.Queries()}

	composeJSON, _ := json.Marshal(composeFixture)
	proposal := json.RawMessage(fmt.Sprintf(`{
      "summary": "alpine sleep 测试服务",
      "compose": %s,
      "health_checks": [
        {"name":"sleeper-log","type":"log-pattern","service":"sleeper","pattern":"harvest-ready-marker"}
      ],
      "wait": {"timeout":"120s","interval":"1s"}
    }`, composeJSON))

	svcName := "sleeper-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	h := &harvest.Harvester{
		Executor: newFakeExecutor(proposal),
		Verifier: &envVerifier{mgr: mgr},
		Catalog: &harvest.GitCatalog{
			RepoPath: repo,
			Identity: harvest.Identity{Name: "tally-integ", Email: "integ@example.com"},
		},
		Sources: map[string]harvest.Source{
			"static": harvest.StaticSource{SourceName: "static", Content: map[string]string{svcName: "raw"}},
		},
	}

	out := h.Run(ctx, []harvest.Service{{Name: svcName, Source: "static", Ref: svcName}})
	if len(out) != 1 {
		t.Fatalf("应有一个结果, got %d", len(out))
	}
	if !out[0].Stored {
		t.Fatalf("alpine 应验证通过并写入 catalog: %+v", out[0])
	}

	// 三个文件都在。
	dir := filepath.Join(repo, "catalog", svcName)
	for _, f := range []string{"definition.yaml", "compose.yaml", "verified.json"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("应写入 %s: %v", f, err)
		}
	}

	// verified.json 含验证时间 + 完整版本组合(含 alpine:latest)。
	b, err := os.ReadFile(filepath.Join(dir, "verified.json"))
	if err != nil {
		t.Fatal(err)
	}
	var vj verifiedJSON
	if err := json.Unmarshal(b, &vj); err != nil {
		t.Fatalf("verified.json 非法: %v", err)
	}
	if vj.VerifiedAt.IsZero() {
		t.Error("verified.json 应含验证时间")
	}
	if len(vj.Combo) == 0 {
		t.Error("verified.json 应含版本组合")
	}
	if !comboHas(vj.Combo, "alpine:latest") {
		t.Errorf("版本组合应含镜像 tag alpine:latest: %v", vj.Combo)
	}

	// git 有提交。
	r, _ := git.PlainOpen(repo)
	if head, err := r.Head(); err != nil || head.Hash().IsZero() {
		t.Errorf("写 catalog 应产生 git 提交: %v", err)
	}
}

// 验收(端到端负例):健康检查一直不过 → 三轮后跳过,**不写 catalog**(INV-2)。
func TestHarvestUnverifiedNotWrittenIntegration(t *testing.T) {
	ensureAlpine(t)
	st := openStore(t)
	repo := initRepo(t)
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	mgr := &environment.Manager{Docker: runner.NewDockerRunner(), Store: st.Queries()}

	composeJSON, _ := json.Marshal(composeFixture)
	// pattern 永不出现 → 健康检查超时;wait 调短让三轮快点跑完。
	proposal := json.RawMessage(fmt.Sprintf(`{
      "summary": "健康检查永不通过的服务",
      "compose": %s,
      "health_checks": [
        {"name":"never","type":"log-pattern","service":"sleeper","pattern":"this-never-appears-xyz"}
      ],
      "wait": {"timeout":"4s","interval":"1s"}
    }`, composeJSON))

	svcName := "badsvc-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	h := &harvest.Harvester{
		Executor: newFakeExecutor(proposal),
		Verifier: &envVerifier{mgr: mgr},
		Catalog: &harvest.GitCatalog{
			RepoPath: repo,
			Identity: harvest.Identity{Name: "tally-integ", Email: "integ@example.com"},
		},
		Sources: map[string]harvest.Source{
			"static": harvest.StaticSource{SourceName: "static", Content: map[string]string{svcName: "raw"}},
		},
	}

	out := h.Run(ctx, []harvest.Service{{Name: svcName, Source: "static", Ref: svcName}})
	if !out[0].Skipped {
		t.Fatalf("健康检查不过应跳过: %+v", out[0])
	}
	// 未验证不得写 catalog。
	if _, err := os.Stat(filepath.Join(repo, "catalog", svcName)); !os.IsNotExist(err) {
		t.Errorf("未验证的服务不应出现在 catalog/: err=%v", err)
	}
}

func comboHas(vc map[string]string, tag string) bool {
	for _, v := range vc {
		if v == tag {
			return true
		}
	}
	return false
}

// —— 测试基建(与 environment 集成测试同法)——

func openStore(t *testing.T) *store.Store {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/tally?sslmode=disable"
	}
	st, err := store.Open(context.Background(), dsn)
	if err != nil {
		t.Skipf("打开 store 失败(数据库不可达?),跳过集成测试: %v", err)
	}
	t.Cleanup(st.Close)
	return st
}

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if _, err := git.PlainInit(dir, false); err != nil {
		t.Fatalf("git init: %v", err)
	}
	return dir
}

// ensureAlpine 保证 alpine:latest 可用。Docker Hub 受限时用本地 alpine 基础镜像打 tag 代替,
// 尽量让测试真跑而不是 skip;实在没有就 skip 并明确报告,别卡死。
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
	for _, base := range []string{"postgres:16-alpine", "redis:8.2", "minio/minio:latest"} {
		if !imageExists(base) {
			continue
		}
		if err := exec.Command("docker", "tag", base, "alpine:latest").Run(); err == nil {
			t.Cleanup(func() { _ = exec.Command("docker", "rmi", "alpine:latest").Run() })
			t.Logf("alpine:latest 拉取受限,用本地 %s 打 tag 代替", base)
			return
		}
	}
	t.Skip("无 alpine 镜像且无法拉取或打 tag,跳过集成测试(已明确报告)")
}

func imageExists(ref string) bool {
	return exec.Command("docker", "image", "inspect", ref).Run() == nil
}
