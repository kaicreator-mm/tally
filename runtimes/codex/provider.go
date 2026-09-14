package codex

// provider.go 拥有 ADR-0003 的有界缝(docs/architecture/adr/
// ADR-0003-codex-provider-endpoint-seeding.md,DECISION_LOCKS #24,实现任务
// P0-04):自定义 provider 部署下,会话构造时向 per-run CODEX_HOME 播种一份
// Tally 生成的最小 config.toml。
//
// 边界(改动前必读,ADR-0003 §2):
//
//   - 这是期望配置管道(与 `-m <model>` 同类),不是状态移交:内容全部由
//     类型化的 Config.Provider 字段生成,没有任何文件从 owner home 复制。
//   - 结构上不可能携带指令/会话状态:结构体没有 instructions/notify/
//     mcp_servers/profiles/history 的字段,渲染器也绝不发明。
//   - 凭据材料仍然只经 #22 的 auth.json 移交(isolation.go 不变),或
//     RequiresOpenAIAuth=false 时经 Config.Env 的环境变量透传;播种本身
//     绝不写任何机密。
//   - 零值 Provider(nil)= 零播种,行为与 P0-03 完全一致。
//   - 不改隔离语义、argv、能力门控或任何其它 runtime 行为:本文件只在
//     Start/Fork 建好 per-run home 之后多写一个生成文件,失败即回滚该 home
//     并让 Start/Fork 显式失败(与 newSessionHome 同一失败纪律)。

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// configTomlName 是 per-run CODEX_HOME 内被播种的文件名。
const configTomlName = "config.toml"

// ProviderEndpoint 是一次显式、类型化的自定义 provider 端点声明(ADR-0003
// §1)。字段集合是封闭的:任何新键都必须先改本结构 —— 这正是「指令/会话
// 状态结构性不可播种」的保证。
type ProviderEndpoint struct {
	// ID 是写入 run home config 的 model_provider id(如 "custom")。
	ID string
	// Name 是 provider 展示名(如 "deepseek")。
	Name string
	// BaseURL 是 provider 的 API 基址(如 "https://api.deepseek.com")。
	BaseURL string
	// WireAPI 是 provider 协议面:"responses" 或 "chat"。
	WireAPI string
	// RequiresOpenAIAuth 为 true 表示 key 经 auth.json 移交抵达
	//(DECISION_LOCKS #22,不变);false 时凭据经 EnvKey 命名的环境变量。
	RequiresOpenAIAuth bool
	// EnvKey 是 RequiresOpenAIAuth=false 时携带 API key 的环境变量名;
	// 只写名字进 config,绝不写值(值经 Config.Env 透传)。
	EnvKey string
}

// providerWireAPIs 是 wire_api 的全部合法取值(codex config 的观测枚举)。
var providerWireAPIs = map[string]bool{
	"responses": true,
	"chat":      true,
}

// validate 校验端点声明的最小完整性。只在播种路径调用:零值 Provider 永远
// 不进入本函数,官方 provider 部署不受任何影响。
func (p *ProviderEndpoint) validate() error {
	if p == nil {
		return nil
	}
	switch {
	case strings.TrimSpace(p.ID) == "":
		return fmt.Errorf("codex: Provider.ID 缺失(ADR-0003:model_provider id 必填)")
	case strings.TrimSpace(p.Name) == "":
		return fmt.Errorf("codex: Provider.Name 缺失(ADR-0003)")
	case strings.TrimSpace(p.BaseURL) == "":
		return fmt.Errorf("codex: Provider.BaseURL 缺失(ADR-0003)")
	case !providerWireAPIs[p.WireAPI]:
		return fmt.Errorf("codex: Provider.WireAPI %q 非法(合法值: responses | chat)", p.WireAPI)
	}
	return nil
}

// renderProviderConfigTOML 渲染播种内容:恰好是 ADR-0003 §1 的最小 config,
// 一行不多 —— model_provider、(Model 非空时的)model、[model_providers.<ID>]
// 节的 name/base_url/wire_api/requires_openai_auth 与(非空时的)env_key。
func renderProviderConfigTOML(p *ProviderEndpoint, model string) (string, error) {
	if err := p.validate(); err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("model_provider = " + tomlQuoteString(p.ID) + "\n")
	if strings.TrimSpace(model) != "" {
		b.WriteString("model = " + tomlQuoteString(model) + "\n")
	}
	b.WriteString("\n[model_providers." + p.ID + "]\n")
	b.WriteString("name = " + tomlQuoteString(p.Name) + "\n")
	b.WriteString("base_url = " + tomlQuoteString(p.BaseURL) + "\n")
	b.WriteString("wire_api = " + tomlQuoteString(p.WireAPI) + "\n")
	b.WriteString("requires_openai_auth = " + fmt.Sprintf("%v", p.RequiresOpenAIAuth) + "\n")
	if p.EnvKey != "" {
		b.WriteString("env_key = " + tomlQuoteString(p.EnvKey) + "\n")
	}
	return b.String(), nil
}

// tomlQuoteString 渲染 TOML 基本字符串:转义反斜杠、双引号与显式控制字符。
func tomlQuoteString(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// seedSessionHome 在 per-run home 内播种生成的 config.toml(仅当
// Config.Provider 置位)。零值 Provider 是 no-op:官方 provider 部署的 run
// home 与 P0-03 逐字节一致。
func seedSessionHome(home string, cfg Config) error {
	if cfg.Provider == nil {
		return nil
	}
	content, err := renderProviderConfigTOML(cfg.Provider, cfg.Model)
	if err != nil {
		return fmt.Errorf("codex: 渲染 per-run config.toml 失败: %w", err)
	}
	path := filepath.Join(home, configTomlName)
	if werr := os.WriteFile(path, []byte(content), 0o644); werr != nil {
		return fmt.Errorf("codex: 写 per-run config.toml 失败: %w", werr)
	}
	return nil
}

// newSeededSessionHome 组合「建隔离 home(仅 auth.json 移交)+ 播种 provider
// 端点」。任一步失败都回收整个 home 并显式报错 —— 不留半初始化状态,与
// newSessionHome 同一纪律。Provider 未置位时等价于 newSessionHome。
func newSeededSessionHome(cfg Config) (string, error) {
	home, err := newSessionHome(cfg.Env)
	if err != nil {
		return "", err
	}
	if err := seedSessionHome(home, cfg); err != nil {
		_ = os.RemoveAll(home)
		return "", err
	}
	return home, nil
}
