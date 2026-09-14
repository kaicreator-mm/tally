package harvest

import "fmt"

// KindDockerCompose 是 harvest 目前唯一产出的环境类型(与 internal/environment 对齐,不做第三种)。
const KindDockerCompose = "docker-compose"

// Definition 是一份待验证的服务定义。YAML 结构与 internal/environment.Definition 兼容
// (它是最终消费方:catalog 里的定义以后会被拷进某个项目去拉起),但 harvest 自带一份:
// 业务包互不导入,协作由上层(cmd/web)编排。verify 走注入的 Verifier,harvest 不 import environment。
type Definition struct {
	Name         string        `yaml:"name"`
	Project      string        `yaml:"project,omitempty"`
	Kind         string        `yaml:"kind"`
	ComposeFile  string        `yaml:"compose_file"`
	ProjectName  string        `yaml:"project_name,omitempty"`
	HealthChecks []HealthCheck `yaml:"health_checks"`
	Wait         WaitPolicy    `yaml:"wait"`

	// Compose 是 compose 文件的完整内容;Store 写成 ComposeFile 指向的文件,不入 definition.yaml。
	Compose string `yaml:"-"`
	// Source 记录来源(source:ref),用于 verified.json 的出处归属,不入 definition.yaml。
	Source string `yaml:"-"`
}

// HealthCheck 是一项健康检查定义。字段与 YAML 标签和 environment.HealthCheck 对齐;
// 同时带 JSON 标签——它是 LLM 提案结构的一部分(propose.go)。
type HealthCheck struct {
	Name string `json:"name" yaml:"name"`
	Type string `json:"type" yaml:"type"`

	// tcp
	Host string `json:"host,omitempty" yaml:"host,omitempty"`
	Port int    `json:"port,omitempty" yaml:"port,omitempty"`

	// http
	URL          string `json:"url,omitempty" yaml:"url,omitempty"`
	ExpectStatus int    `json:"expect_status,omitempty" yaml:"expect_status,omitempty"`
	ExpectBody   string `json:"expect_body,omitempty" yaml:"expect_body,omitempty"`

	// log-pattern
	Service string `json:"service,omitempty" yaml:"service,omitempty"`
	Pattern string `json:"pattern,omitempty" yaml:"pattern,omitempty"`

	Timeout string `json:"timeout,omitempty" yaml:"timeout,omitempty"` // "2s" 这类时长串
}

// WaitPolicy 是健康检查的等待重试策略(时长用 "60s" 串,environment 侧能解析)。
type WaitPolicy struct {
	Timeout  string `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Interval string `json:"interval,omitempty" yaml:"interval,omitempty"`
}

// 三种健康检查类型(与 environment 对齐)。
const (
	CheckTCP        = "tcp"
	CheckHTTP       = "http"
	CheckLogPattern = "log-pattern"
)

// validateCheck 校验单项健康检查的类型与必要字段(镜像 environment.validateCheck)。
func validateCheck(hc HealthCheck) error {
	switch hc.Type {
	case CheckTCP:
		if hc.Host == "" || hc.Port == 0 {
			return fmt.Errorf("健康检查 %q:tcp 需要 host 和 port", hc.Name)
		}
	case CheckHTTP:
		if hc.URL == "" {
			return fmt.Errorf("健康检查 %q:http 需要 url", hc.Name)
		}
	case CheckLogPattern:
		if hc.Pattern == "" {
			return fmt.Errorf("健康检查 %q:log-pattern 需要 pattern", hc.Name)
		}
	default:
		return fmt.Errorf("健康检查 %q:未知类型 %q(支持 tcp / http / log-pattern)", hc.Name, hc.Type)
	}
	return nil
}

// validateChecks 要求至少一项健康检查且每项合法。
// PRD 第一部分:健康检查是「这次拉起是不是真活了」的唯一机器判据,没有它的定义不许存在。
func validateChecks(checks []HealthCheck) error {
	if len(checks) == 0 {
		return ErrNoHealthCheck
	}
	for _, hc := range checks {
		if err := validateCheck(hc); err != nil {
			return err
		}
	}
	return nil
}
