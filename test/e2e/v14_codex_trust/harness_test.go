//go:build integration

// Package codextrust 承载 P0-04 门禁端到端:真实 Codex CLI(codex-cli 0.153.0,
// 经 ADR-0003 ProviderEndpoint 端点播种 + DECISION_LOCKS #22 auth.json 移交)
// 驱动的 standard Task + held-out 最终验证信任路径。
//
// 结构与 test/e2e 的 P1-47 纵向切片同构(FakeRuntime → 真实 CodexRuntime):
//
//	种子 fixture repo(微型 Go module + 一条可见失败测试)
//	→ 真实 codex 回合在工作流内核的一次性 worktree 内产出候选(无内容注入)
//	→ RunStoreSink 把 Run/ChangeSet 落进真实 PostgreSQL(每测试独立 schema、
//	  自跑 goose 迁移,store.TestDB)
//	→ 冻结 VerificationContract(visible go test + held_out git grep)经最终
//	  验证编排器在一次性验证 workspace / Tally 控制的 holdout 边界内执行
//	→ 会话隔离:共享 conformance Session Isolation Canary 先行(AC 前置),
//	  通过才允许 E2E 以 SessionIsolationVerified 声明 session.isolation。
//
// 环境前提(缺失一律 SKIP,不产出假失败):
//
//	TALLY_E2E_CODEX_AUTH_FILE  指向 {"OPENAI_API_KEY":"..."} 的路径
//	                           (凭据文件内容绝不打印、绝不入库);
//	DATABASE_URL               PostgreSQL DSN(默认本任务容器 5433);
//	TALLY_E2E_CODEX_BIN        可选,codex 二进制覆盖;
//	TALLY_E2E_CODEX_SANDBOX    可选,-s 取值覆盖(默认 workspace-write,
//	                           本机 2026-09-09 实测可用的最小特权模式)。
package codextrust

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"tally/internal/agentruntime"
	"tally/internal/runner"
	"tally/internal/runstore"
	"tally/internal/store"
	"tally/runtimes/codex"
	"tally/test/runtimeconformance"
)

const (
	p04DefaultDSN = "postgres://postgres:test@localhost:5433/postgres?sslmode=disable"
	// p04DefaultCodexBin 是本机 2026-09-09 验证过的 codex CLI 路径(App 发布
	// 形态,内容哈希目录会随更新变化;可用 TALLY_E2E_CODEX_BIN 覆盖)。
	p04DefaultCodexBin = `C:\Users\15549\AppData\Local\OpenAI\Codex\bin\9ba750cce02d5e5c\codex.exe`
	// p04DefaultSandbox 是本机(Windows)实测唯一能让 fixture 真实执行的 -s
	// 模式。实证记录(2026-09-09,codex-cli 0.153.0):workspace-write 下
	// codex 的 Windows 沙箱把 workspace 挂为只读并拒绝一切 exec_command
	//("rejected: blocked by policy" —— 连 `pwd`/`go version` 都被拒),
	// 只回话不执行命令的回合不受影响。本 fixture 需要真实写 workspace 与
	// 运行 go test,因此按任务契约落入「danger-full-access only if fixture
	// needs writes」的例外;缓解边界:workspace 本身是一次性 Git worktree
	// (INV-EXEC-01),会话 home 是 per-run 隔离 home(#22)。
	p04DefaultSandbox = "danger-full-access"
	// p04Model 是期望 model 档位(ADR-0003 实证形态)。它与「实际生效模型」
	// 是两个事实(Decision Lock 15):exec 表面不自报模型,observed = unknown。
	p04Model = "deepseek-v4-flash"
)

var (
	// p04CanaryOnce 保证一次测试进程内 canary 至多真实执行一轮(有界模型
	// 用量:注入 1 回合 + 检索 2 回合)。
	p04CanaryOnce sync.Once
	// p04CanaryReport 是 canary 的 typed 裁决(nil = 未运行)。
	p04CanaryReport *runtimeconformance.SessionIsolationReport
)

// ---- 环境解析(缺失即 SKIP)-------------------------------------------------

// p04CodexBinPath 解析 codex CLI 路径;不存在时 SKIP(环境前提,非代码缺陷)。
func p04CodexBinPath(t *testing.T) string {
	t.Helper()
	bin := os.Getenv("TALLY_E2E_CODEX_BIN")
	if bin == "" {
		bin = p04DefaultCodexBin
	}
	if _, err := os.Stat(bin); err != nil {
		t.Skipf("codex CLI 不可用(%s): %v", bin, err)
	}
	return bin
}

// p04OwnerHome 构建一个只含 auth.json 的临时 owner home(P0-03 移交源)。
// 凭据内容经 TALLY_E2E_CODEX_AUTH_FILE 指向的文件读取:只搬运字节,绝不打印、
// 绝不写入任何被跟踪文件。变量未设置时 SKIP(AC5 的 NOT_RUN 路径)。
func p04OwnerHome(t *testing.T) string {
	t.Helper()
	authFile := os.Getenv("TALLY_E2E_CODEX_AUTH_FILE")
	if strings.TrimSpace(authFile) == "" {
		t.Skip("TALLY_E2E_CODEX_AUTH_FILE 未设置:无凭据,跳过真实 codex 路径(NOT_RUN 环境)")
	}
	data, err := os.ReadFile(authFile)
	if err != nil {
		t.Fatalf("读取凭据文件 %s 失败: %v", authFile, err)
	}
	owner := t.TempDir()
	if werr := os.WriteFile(filepath.Join(owner, "auth.json"), data, 0o600); werr != nil {
		t.Fatalf("写 owner auth.json 失败: %v", werr)
	}
	return owner
}

// p04CodexConfig 组装本任务的 CodexRuntime 配置:真实二进制 + 最小特权沙箱 +
// owner home(仅 auth.json)+ ADR-0003 Provider 端点播种(DeepSeek responses
// 兼容端点,凭据仍只经 auth.json 移交)。verified 控制 session.isolation 声明
// 门:只有共享 canary 通过后才允许置位(DECISION_LOCKS #22)。
func p04CodexConfig(t *testing.T, verified bool) codex.Config {
	t.Helper()
	sandbox := os.Getenv("TALLY_E2E_CODEX_SANDBOX")
	if sandbox == "" {
		sandbox = p04DefaultSandbox
	}
	return codex.Config{
		BinPath: p04CodexBinPath(t),
		Sandbox: sandbox,
		Model:   p04Model,
		Env:     map[string]string{"CODEX_HOME": p04OwnerHome(t)},
		Provider: &codex.ProviderEndpoint{
			ID:                 "custom",
			Name:               "deepseek",
			BaseURL:            "https://api.deepseek.com",
			WireAPI:            "responses",
			RequiresOpenAIAuth: true,
		},
		SessionIsolationVerified: verified,
	}
}

// ---- PostgreSQL(每测试独立 schema,store.TestDB)--------------------------

// p04Store 返回接好独立 schema 的 runstore 适配器与裸连接池;数据库不可达时
// SKIP(不产出假失败)。默认 DSN 指向本任务的 5433 容器。
func p04Store(t *testing.T) (*runstore.Store, *pgxpool.Pool) {
	t.Helper()
	base := os.Getenv("DATABASE_URL")
	if base == "" {
		base = p04DefaultDSN
	}
	if !p04DBReachable(base) {
		t.Skipf("P0-04 端到端测试需要 PostgreSQL(%s 不可达),跳过", base)
	}
	t.Setenv("DATABASE_URL", base) // store.TestDB 读同一 DSN
	st := store.TestDB(t)
	return runstore.New(st.Queries()), st.Pool()
}

func p04DBReachable(base string) bool {
	u, err := url.Parse(base)
	if err != nil {
		return false
	}
	host := u.Hostname()
	if host == "" {
		host = "localhost"
	}
	port := u.Port()
	if port == "" {
		port = "5432"
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 2*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// ---- git 固件(结构化 argv)-------------------------------------------------

func p04GitEnv() []string {
	return []string{
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=" + os.DevNull,
		"GIT_CONFIG_SYSTEM=" + os.DevNull,
		"GIT_TERMINAL_PROMPT=0",
	}
}

// p04GitOut 是非测试上下文(hook)可用的 git 执行入口。返回值做了整体 Trim,
// 只用于取单值(rev-parse 等);porcelain 逐行解析必须用 p04GitRaw。
func p04GitOut(ctx context.Context, dir string, argv ...string) (string, error) {
	out, err := p04GitRaw(ctx, dir, argv...)
	return strings.TrimSpace(out), err
}

// p04GitRaw 返回未经裁剪的 stdout(porcelain 行首的 XY 状态列是位置敏感的)。
func p04GitRaw(ctx context.Context, dir string, argv ...string) (string, error) {
	res, err := (runner.LocalRunner{}).Run(ctx, runner.Command{
		Argv: append([]string{"git"}, argv...),
		Dir:  dir,
		Env:  p04GitEnv(),
	})
	if err != nil {
		return "", fmt.Errorf("git %v: %w", argv, err)
	}
	if res.ExitCode != 0 {
		return "", fmt.Errorf("git %v: exit %d: %s", argv, res.ExitCode, res.Stderr)
	}
	return res.Stdout, nil
}

func p04Git(t *testing.T, dir string, argv ...string) string {
	t.Helper()
	out, err := p04GitOut(context.Background(), dir, argv...)
	if err != nil {
		t.Fatalf("%v", err)
	}
	return out
}

func p04Commit(t *testing.T, repo, name, content, msg string) string {
	t.Helper()
	p := filepath.Join(repo, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("建目录 %s: %v", filepath.Dir(name), err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	p04Git(t, repo, "add", ".")
	p04Git(t, repo, "commit", "-m", msg)
	return p04Git(t, repo, "rev-parse", "HEAD")
}

// ---- 临时目录差集(处置证据)-------------------------------------------------

// p04TempSnapshot 记录 pattern 匹配的临时路径集合(基线),供运行后断言
// 「无新增残留」。
func p04TempSnapshot(t *testing.T, pattern string) map[string]bool {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(os.TempDir(), pattern))
	if err != nil {
		t.Fatalf("glob %s: %v", pattern, err)
	}
	out := make(map[string]bool, len(matches))
	for _, m := range matches {
		out[m] = true
	}
	return out
}

func p04TempNewAfter(t *testing.T, pattern string, before map[string]bool) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(os.TempDir(), pattern))
	if err != nil {
		t.Fatalf("glob %s: %v", pattern, err)
	}
	var fresh []string
	for _, m := range matches {
		if !before[m] {
			fresh = append(fresh, m)
		}
	}
	return fresh
}

// ---- 身份记录装饰器(捕获实际运行时/会话身份,Decision Lock 15)--------------

// p04RecordingRuntime 用委托包装真实 runtime,捕获每轮 claim 与轮次结束后的
// ObservedIdentity(快照:codex-cli / 版本 / thread id / observed model)。
// 它不做任何裁决,只记录事实。
type p04RecordingRuntime struct {
	inner agentruntime.Runtime

	mu       sync.Mutex
	claims   []agentruntime.AgentRunResult
	identity agentruntime.ObservedIdentity
}

func (r *p04RecordingRuntime) Descriptor() agentruntime.RuntimeDescriptor {
	return r.inner.Descriptor()
}

func (r *p04RecordingRuntime) Start(ctx context.Context, req agentruntime.StartRequest) (agentruntime.Session, error) {
	s, err := r.inner.Start(ctx, req)
	if err != nil {
		return nil, err
	}
	return &p04RecordingSession{inner: s, rec: r}, nil
}

func (r *p04RecordingRuntime) record(res agentruntime.AgentRunResult, inner agentruntime.Session) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.claims = append(r.claims, res)
	if rep, ok := inner.(agentruntime.IdentityReporter); ok {
		r.identity = rep.Identity()
	}
}

func (r *p04RecordingRuntime) lastClaim() (agentruntime.AgentRunResult, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.claims) == 0 {
		return agentruntime.AgentRunResult{}, false
	}
	return r.claims[len(r.claims)-1], true
}

type p04RecordingSession struct {
	inner agentruntime.Session
	rec   *p04RecordingRuntime
}

func (s *p04RecordingSession) Execute(ctx context.Context, req agentruntime.ExecuteRequest) (agentruntime.AgentRunResult, error) {
	res, err := s.inner.Execute(ctx, req)
	s.rec.record(res, s.inner)
	return res, err
}

func (s *p04RecordingSession) Cancel(ctx context.Context, id agentruntime.RunID) error {
	return s.inner.Cancel(ctx, id)
}

func (s *p04RecordingSession) Events(ctx context.Context) (agentruntime.EventStream, error) {
	return s.inner.Events(ctx)
}

func (s *p04RecordingSession) Result(ctx context.Context, id agentruntime.RunID) (agentruntime.AgentRunResult, error) {
	return s.inner.Result(ctx, id)
}

func (s *p04RecordingSession) Close(ctx context.Context) error { return s.inner.Close(ctx) }

// Identity 透传 IdentityReporter(会话开启时 thread 尚未观察,为零值;轮次
// 后经 rec.identity 捕获)。
func (s *p04RecordingSession) Identity() agentruntime.ObservedIdentity {
	if rep, ok := s.inner.(agentruntime.IdentityReporter); ok {
		return rep.Identity()
	}
	return agentruntime.ObservedIdentity{}
}
