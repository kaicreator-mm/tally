package codex

// provider_test.go 验证 ADR-0003 有界缝(DECISION_LOCKS #24):
//
//   - 播种内容逐字节等于 ADR §1 的最小 config(键集合封闭,Nothing else);
//   - 条件键的条件性:model 非空才写、env_key 非空才写;
//   - 零值 Provider = 零播种(P0-03 行为逐字节不变);
//   - Start/Fork 会话 home 真实被播种(auth.json 移交不受影响,owner home
//     不被改动);播种失败回收整个 home,Start 显式失败。
//
// 只测播种本身:隔离语义/argv/能力门控不属于本缝(AC 边界见 ADR §2)。

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tally/internal/agentruntime"
)

// p04Provider 是环境事实里的工作 provider 形态(DeepSeek responses 兼容端点)。
func p04Provider() *ProviderEndpoint {
	return &ProviderEndpoint{
		ID:                 "custom",
		Name:               "deepseek",
		BaseURL:            "https://api.deepseek.com",
		WireAPI:            "responses",
		RequiresOpenAIAuth: true,
	}
}

func TestRenderProviderConfigTOMLMinimalExact(t *testing.T) {
	got, err := renderProviderConfigTOML(p04Provider(), "")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	want := "model_provider = \"custom\"\n" +
		"\n[model_providers.custom]\n" +
		"name = \"deepseek\"\n" +
		"base_url = \"https://api.deepseek.com\"\n" +
		"wire_api = \"responses\"\n" +
		"requires_openai_auth = true\n"
	if got != want {
		t.Fatalf("渲染内容不是 ADR-0003 最小形态:\n got: %q\nwant: %q", got, want)
	}
	for _, banned := range []string{"model ", "env_key", "instructions", "profile", "mcp_servers", "notify"} {
		if strings.Contains(got, banned) {
			t.Fatalf("最小 config 不应包含 %q:\n%s", banned, got)
		}
	}
}

func TestRenderProviderConfigTOMLConditionalKeys(t *testing.T) {
	got, err := renderProviderConfigTOML(&ProviderEndpoint{
		ID:                 "custom",
		Name:               "deepseek",
		BaseURL:            "https://api.deepseek.com",
		WireAPI:            "chat",
		RequiresOpenAIAuth: false,
		EnvKey:             "DEEPSEEK_API_KEY",
	}, "deepseek-v4-flash")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	want := "model_provider = \"custom\"\n" +
		"model = \"deepseek-v4-flash\"\n" +
		"\n[model_providers.custom]\n" +
		"name = \"deepseek\"\n" +
		"base_url = \"https://api.deepseek.com\"\n" +
		"wire_api = \"chat\"\n" +
		"requires_openai_auth = false\n" +
		"env_key = \"DEEPSEEK_API_KEY\"\n"
	if got != want {
		t.Fatalf("渲染内容与 ADR-0003 形态不符:\n got: %q\nwant: %q", got, want)
	}
}

func TestRenderProviderConfigTOMLEscapesBasicStrings(t *testing.T) {
	got, err := renderProviderConfigTOML(&ProviderEndpoint{
		ID:      "custom",
		Name:    `say "hi" \ now`,
		BaseURL: "https://example.invalid/a\\b",
		WireAPI: "responses",
	}, "")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(got, `name = "say \"hi\" \\ now"`) {
		t.Fatalf("name 未按 TOML 基本字符串转义:\n%s", got)
	}
	if !strings.Contains(got, `base_url = "https://example.invalid/a\\b"`) {
		t.Fatalf("base_url 未按 TOML 基本字符串转义:\n%s", got)
	}
}

func TestProviderEndpointValidateRejectsIncomplete(t *testing.T) {
	cases := map[string]*ProviderEndpoint{
		"缺 ID":       {Name: "n", BaseURL: "https://x", WireAPI: "responses"},
		"缺 Name":     {ID: "custom", BaseURL: "https://x", WireAPI: "responses"},
		"缺 BaseURL":  {ID: "custom", Name: "n", WireAPI: "responses"},
		"非法 WireAPI": {ID: "custom", Name: "n", BaseURL: "https://x", WireAPI: "grpc"},
		"空 WireAPI":  {ID: "custom", Name: "n", BaseURL: "https://x"},
	}
	for name, p := range cases {
		if err := p.validate(); err == nil {
			t.Fatalf("%s: 应显式拒绝", name)
		}
		if _, err := renderProviderConfigTOML(p, ""); err == nil {
			t.Fatalf("%s: 渲染路径同样应显式拒绝", name)
		}
	}
}

// TestZeroProviderKeepsHomeAuthOnly(向后不变性):Provider 零值时 run home
// 与 P0-03 逐字节一致 —— 只有移交的 auth.json,没有 config.toml。
func TestZeroProviderKeepsHomeAuthOnly(t *testing.T) {
	owner := t.TempDir()
	writeOwnerAuth(t, owner, `{"tokens":{"access_token":"owner"}}`)
	rt := New(Config{BinPath: fakeCodexBin, Env: map[string]string{"CODEX_HOME": owner}})
	s := startSession(t, rt, t.TempDir())
	defer func() { _ = s.Close(context.Background()) }()
	entries, err := os.ReadDir(s.home)
	if err != nil {
		t.Fatalf("读 home: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != authFileName {
		t.Fatalf("零值 Provider 不得播种: home = %v", entries)
	}
}

// TestStartSeedsProviderConfigIntoSessionHome(AC:播种发生在会话构造):
// Start 建好的 per-run home 恰好含 auth.json + 生成的 config.toml,内容为
// ADR 最小形态;owner home 不被改动。
func TestStartSeedsProviderConfigIntoSessionHome(t *testing.T) {
	owner := t.TempDir()
	writeOwnerAuth(t, owner, `{"tokens":{"access_token":"owner"}}`)
	rt := New(Config{
		BinPath:  fakeCodexBin,
		Model:    "deepseek-v4-flash",
		Env:      map[string]string{"CODEX_HOME": owner},
		Provider: p04Provider(),
	})
	ws := t.TempDir()
	s := startSession(t, rt, ws)
	defer func() { _ = s.Close(context.Background()) }()

	entries, err := os.ReadDir(s.home)
	if err != nil {
		t.Fatalf("读 home: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("播种后的 home 应恰含 auth.json + config.toml: %v", entries)
	}
	data, err := os.ReadFile(filepath.Join(s.home, configTomlName))
	if err != nil {
		t.Fatalf("读 config.toml: %v", err)
	}
	// Config.Model 置位 → model 行出现;RequiresOpenAIAuth=true → 无 env_key。
	want := "model_provider = \"custom\"\n" +
		"model = \"deepseek-v4-flash\"\n" +
		"\n[model_providers.custom]\n" +
		"name = \"deepseek\"\n" +
		"base_url = \"https://api.deepseek.com\"\n" +
		"wire_api = \"responses\"\n" +
		"requires_openai_auth = true\n"
	if string(data) != want {
		t.Fatalf("播种内容不符:\n got: %q\nwant: %q", data, want)
	}
	if _, err := os.Stat(filepath.Join(s.home, authFileName)); err != nil {
		t.Fatalf("auth.json 移交不应受播种影响: %v", err)
	}
	if entries0 := dirNames(t, s.home); len(entries0) != 2 {
		t.Fatalf("home 内容应恰为两项: %v", entries0)
	}
	if data, err := os.ReadFile(filepath.Join(owner, authFileName)); err != nil || string(data) != `{"tokens":{"access_token":"owner"}}` {
		t.Fatalf("owner home 不得被播种改动: %v %q", err, data)
	}
}

// TestForkSeedsProviderConfigToo(ADR-0003:会话构造 = Start/Fork)。
func TestForkSeedsProviderConfigToo(t *testing.T) {
	owner := t.TempDir()
	writeOwnerAuth(t, owner, `{"tokens":{"access_token":"owner"}}`)
	rt := New(Config{
		BinPath:  fakeCodexBin,
		Env:      map[string]string{"CODEX_HOME": owner},
		Provider: p04Provider(),
	})
	ctx := context.Background()
	parent := startSession(t, rt, t.TempDir())
	defer func() { _ = parent.Close(ctx) }()
	if _, err := parent.Execute(ctx, agentruntime.ExecuteRequest{Instruction: "x"}); err != nil {
		t.Fatalf("parent turn: %v", err)
	}
	child, err := rt.Fork(ctx, parent)
	if err != nil {
		t.Fatalf("Fork: %v", err)
	}
	cs := child.(*Session)
	defer func() { _ = cs.Close(ctx) }()
	if cs.home == parent.home {
		t.Fatalf("fork 会话应有独立 home")
	}
	data, err := os.ReadFile(filepath.Join(cs.home, configTomlName))
	if err != nil {
		t.Fatalf("fork 会话 home 应被播种: %v", err)
	}
	// Config.Model 未置位 → 无 model 行(条件键的条件性,经真实构造路径)。
	if strings.Contains(string(data), "model =") {
		t.Fatalf("Config.Model 为空时不得写 model 行:\n%s", data)
	}
	if !strings.Contains(string(data), `model_provider = "custom"`) {
		t.Fatalf("播种缺 model_provider:\n%s", data)
	}
}

// TestSeedFailureRejectsStartAndReclaimsHome(失败纪律):非法 Provider 声明
// 让 Start 显式失败,半初始化 home 被回收。
func TestSeedFailureRejectsStartAndReclaimsHome(t *testing.T) {
	owner := t.TempDir()
	writeOwnerAuth(t, owner, `{"tokens":{"access_token":"owner"}}`)
	rt := New(Config{
		BinPath: fakeCodexBin,
		Env:     map[string]string{"CODEX_HOME": owner},
		Provider: &ProviderEndpoint{
			ID: "custom", Name: "deepseek", BaseURL: "https://api.deepseek.com",
			WireAPI: "grpc", // 非法
		},
	})
	before := map[string]bool{}
	for _, p := range homeLeakPaths(t) {
		before[p] = true
	}
	_, err := rt.Start(context.Background(), agentruntime.StartRequest{Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("非法 Provider 声明应让 Start 显式失败")
	}
	if !strings.Contains(err.Error(), "WireAPI") {
		t.Fatalf("错误应指明非法字段: %v", err)
	}
	for _, p := range homeLeakPaths(t) {
		if !before[p] {
			t.Fatalf("失败的会话构造不得留下新的 tally-codex-home-* 目录: %s", p)
		}
	}
}

func dirNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("读目录 %s: %v", dir, err)
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}

// homeLeakPaths 列出系统临时目录下的 tally-codex-home-* 路径。
func homeLeakPaths(t *testing.T) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(os.TempDir(), "tally-codex-home-*"))
	if err != nil {
		t.Fatalf("glob 临时目录: %v", err)
	}
	return matches
}
