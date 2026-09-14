package environment

import (
	"errors"
	"fmt"
)

// ErrNoHealthCheck:环境定义缺少健康检查。注册时返回它并**拒绝注册**(不是警告)。
//
// 理由(PRD 第一部分):健康检查是「这次拉起是不是真活了」的唯一机器判据。
// 没有它,「拉起成功」的记录就可能是假的 —— 而这是自用工具,没人会来纠正。
var ErrNoHealthCheck = errors.New("环境定义缺少健康检查")

// Registry 校验并登记环境定义。注册是硬校验:不合格的定义直接拒绝,不收进来。
type Registry struct {
	defs map[string]Definition // key: project + "/" + name
}

// NewRegistry 返回空注册表。
func NewRegistry() *Registry {
	return &Registry{defs: map[string]Definition{}}
}

// Register 校验一份定义并登记。任一条不过都返回错误、不登记:
//   - 缺健康检查 → ErrNoHealthCheck(拒绝,不是警告)
//   - kind 未知 / 必要字段缺失 / 健康检查类型不合法 → 结构化错误
func (r *Registry) Register(def Definition) error {
	if err := Validate(def); err != nil {
		return err
	}
	r.defs[def.Project+"/"+def.Name] = def
	return nil
}

// Get 取已登记的定义。
func (r *Registry) Get(project, name string) (Definition, bool) {
	d, ok := r.defs[project+"/"+name]
	return d, ok
}

// Validate 是注册硬校验(不依赖 Registry,可独立调用)。
func Validate(def Definition) error {
	if len(def.HealthChecks) == 0 {
		return ErrNoHealthCheck
	}
	switch def.Kind {
	case KindDockerCompose:
		if def.ComposeFile == "" {
			return fmt.Errorf("docker-compose 环境 %q 缺 compose_file", def.Name)
		}
	case KindSSHScript:
		if def.Script == "" {
			return fmt.Errorf("ssh-script 环境 %q 缺 script", def.Name)
		}
		if def.SSH.Host == "" || def.SSH.User == "" {
			return fmt.Errorf("ssh-script 环境 %q 缺 ssh.host / ssh.user", def.Name)
		}
	default:
		return fmt.Errorf("环境 %q 的 kind %q 未知(支持 %s / %s)",
			def.Name, def.Kind, KindDockerCompose, KindSSHScript)
	}
	for _, hc := range def.HealthChecks {
		if err := validateCheck(hc); err != nil {
			return err
		}
	}
	return nil
}

// validateCheck 校验单项健康检查的类型与必要字段。
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
