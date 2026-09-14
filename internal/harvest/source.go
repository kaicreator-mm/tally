package harvest

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Service 是抽取清单里的一项。**清单是显式、有限的**(默认十来个),
// harvest 不提供遍历 Coolify 全部 362 个模板的入口(PRD 第一部分规则 ①:不批量抽,抽了不用的会过期)。
type Service struct {
	Name   string // catalog 目录名,如 "minio"
	Source string // 来源名:"coolify" / "upstream" / "docs"
	Ref    string // 来源内的引用:coolify 为模板名;upstream/docs 为 URL
	Hints  string // 给 LLM 的额外提示(端口、健康端点、依赖等)
}

// Material 是交给 LLM 的原始材料。
type Material struct {
	Service Service
	Raw     string // 原始定义/文档片段(只读自公开来源)
}

// Source 按服务取原始材料。**只读公开的定义文件/文档,不碰 Coolify 运行时**(铁律 6:不 fork、不依赖其运行时)。
type Source interface {
	Name() string
	Fetch(ctx context.Context, svc Service) (Material, error)
}

// Fetcher 取一段原始文本(如从 URL)。可注入:生产用 HTTPFetcher(只读),测试用 fake/httptest。
type Fetcher interface {
	Fetch(ctx context.Context, ref string) (string, error)
}

// DefaultServices 是默认抽取清单(~10 个作者实际要用的底座)。抽完即停,不扩展成全量。
// coolify 的 ref 是 templates/compose/<ref>.yaml;upstream 的 ref 是上游官方 URL。
var DefaultServices = []Service{
	{Name: "minio", Source: "coolify", Ref: "minio", Hints: "对象存储,S3 兼容;控制台端口 9001"},
	{Name: "rabbitmq", Source: "coolify", Ref: "rabbitmq", Hints: "消息队列;管理界面端口 15672"},
	{Name: "keycloak", Source: "coolify", Ref: "keycloak", Hints: "身份认证;自带内嵌 DB 或接 postgres,健康端点 /health/ready"},
	{Name: "n8n", Source: "coolify", Ref: "n8n", Hints: "工作流自动化;端口 5678"},
	{Name: "plausible", Source: "coolify", Ref: "plausible", Hints: "网站分析;依赖 clickhouse + postgres"},
	{Name: "vaultwarden", Source: "coolify", Ref: "vaultwarden", Hints: "密码管理;端口 80"},
	{Name: "grafana", Source: "coolify", Ref: "grafana", Hints: "监控面板;端口 3000,健康端点 /api/health"},
	{Name: "umami", Source: "coolify", Ref: "umami", Hints: "网站分析;依赖 postgres,端口 3000"},
	{Name: "uptime-kuma", Source: "coolify", Ref: "uptime-kuma", Hints: "拨测监控;端口 3001"},
	{Name: "redis", Source: "upstream", Ref: "https://raw.githubusercontent.com/redis/redis/unstable/docker-compose.yml", Hints: "缓存;Coolify 无独立模板,走上游官方 compose。注意 Redis 已改双许可(RSALv2+SSPLv1),要纯开源可换 Valkey(见 LICENSE-NOTES)"},
	{Name: "postgres", Source: "upstream", Ref: "https://raw.githubusercontent.com/docker-library/docs/master/postgres/README.md", Hints: "数据库;Coolify 作为数据库资源处理,参考上游官方文档写 compose,健康检查用 pg_isready"},
}

// defaultCoolifyBase 指向 Coolify 公开服务模板的原始文件目录(只读)。
const defaultCoolifyBase = "https://raw.githubusercontent.com/coollabsio/coolify/v4.x/templates/compose"

// CoolifySource 只读 Coolify 公开的服务模板(GitHub 原始文件),不依赖其运行时。
type CoolifySource struct {
	BaseURL string // 缺省 defaultCoolifyBase;可注入测试地址
	Fetcher Fetcher
}

// Name 实现 Source。
func (s CoolifySource) Name() string { return "coolify" }

func (s CoolifySource) base() string {
	if s.BaseURL != "" {
		return s.BaseURL
	}
	return defaultCoolifyBase
}

// Fetch 取 templates/compose/<ref>.yaml。
func (s CoolifySource) Fetch(ctx context.Context, svc Service) (Material, error) {
	raw, err := s.Fetcher.Fetch(ctx, s.base()+"/"+svc.Ref+".yaml")
	if err != nil {
		return Material{}, fmt.Errorf("取 coolify 模板 %s: %w", svc.Ref, err)
	}
	return Material{Service: svc, Raw: raw}, nil
}

// HTTPSource 是通用的「按 URL 只读取材料」来源,用于上游 compose 示例 / 官方文档。
// SourceName 区分用途("upstream" / "docs")。
type HTTPSource struct {
	SourceName string
	Fetcher    Fetcher
}

// Name 实现 Source。
func (s HTTPSource) Name() string { return s.SourceName }

// Fetch 直接取 svc.Ref(应为 URL)。
func (s HTTPSource) Fetch(ctx context.Context, svc Service) (Material, error) {
	raw, err := s.Fetcher.Fetch(ctx, svc.Ref)
	if err != nil {
		return Material{}, fmt.Errorf("取 %s 材料 %s: %w", s.SourceName, svc.Ref, err)
	}
	return Material{Service: svc, Raw: raw}, nil
}

// StaticSource 离线/测试用:按 ref 返回预置材料。生产不用。
type StaticSource struct {
	SourceName string
	Content    map[string]string // ref → 原始材料
}

// Name 实现 Source。
func (s StaticSource) Name() string { return s.SourceName }

// Fetch 返回预置材料,缺失则报错。
func (s StaticSource) Fetch(_ context.Context, svc Service) (Material, error) {
	raw, ok := s.Content[svc.Ref]
	if !ok {
		return Material{}, fmt.Errorf("static source 缺 %s 的材料", svc.Ref)
	}
	return Material{Service: svc, Raw: raw}, nil
}

// HTTPFetcher 用标准库 net/http 只读取文本(不引入额外依赖,技术栈:标准库直连)。
type HTTPFetcher struct {
	Client *http.Client // nil 时用带超时的默认客户端
}

// Fetch 实现 Fetcher:GET 该 URL,非 2xx 视为失败。
func (f HTTPFetcher) Fetch(ctx context.Context, url string) (string, error) {
	c := f.Client
	if c == nil {
		c = &http.Client{Timeout: 30 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("构造请求 %s: %w", url, err)
	}
	resp, err := c.Do(req)
	if err != nil {
		return "", fmt.Errorf("取 %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("取 %s: HTTP %d", url, resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", fmt.Errorf("读响应 %s: %w", url, err)
	}
	return string(b), nil
}
