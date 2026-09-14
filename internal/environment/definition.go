package environment

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Kind 是环境类型。只支持两种(PRD 第一部分:不做第三种)。
type Kind string

const (
	// KindDockerCompose 用 compose 文件拉起(本地或远程 docker 主机)。
	KindDockerCompose Kind = "docker-compose"
	// KindSSHScript 连到一台机器上按脚本执行。
	KindSSHScript Kind = "ssh-script"
)

// Definition 是一份环境定义(.tally/environments/*.yaml 解析而来)。
//
// Hash 是定义文件**内容**的 sha256(INV-4):定义一改,哈希就变,
// 新旧拉起记录自然分开,失效传播不需要额外机制。
type Definition struct {
	Name         string        `yaml:"name"`
	Project      string        `yaml:"project"`
	Kind         Kind          `yaml:"kind"`
	ComposeFile  string        `yaml:"compose_file"` // docker-compose:compose 文件(相对本定义目录)
	ProjectName  string        `yaml:"project_name"` // docker-compose:compose 项目名,缺省用 Name
	SSH          SSHDef        `yaml:"ssh"`          // ssh-script:连接信息
	Script       string        `yaml:"script"`       // ssh-script:拉起脚本
	Teardown     string        `yaml:"teardown"`     // ssh-script 可选:推倒脚本
	LogCommand   string        `yaml:"log_command"`  // ssh-script 可选:log-pattern 健康检查取日志的命令
	HealthChecks []HealthCheck `yaml:"health_checks"`
	Wait         WaitPolicy    `yaml:"wait"`

	// 非 YAML 字段,解析时填。
	Hash string `yaml:"-"` // 定义文件内容哈希(INV-4)
	Dir  string `yaml:"-"` // 定义文件所在目录,解析相对路径用
}

// SSHDef 是 ssh-script 环境的连接信息。
type SSHDef struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"` // 缺省 22
	User     string `yaml:"user"`
	Password string `yaml:"password"` // 与 KeyFile 二选一
	KeyFile  string `yaml:"key_file"`
}

// HealthCheck 是一项健康检查的定义。Type ∈ {tcp, http, log-pattern}(见 health.go)。
type HealthCheck struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`

	// tcp
	Host string `yaml:"host"`
	Port int    `yaml:"port"`

	// http
	URL          string `yaml:"url"`
	ExpectStatus int    `yaml:"expect_status"` // 0 = 任意 2xx
	ExpectBody   string `yaml:"expect_body"`   // 可选:响应体须含此子串

	// log-pattern
	Service string `yaml:"service"` // docker-compose 服务名,空 = 全部日志
	Pattern string `yaml:"pattern"` // 日志里须出现的子串

	Timeout duration `yaml:"timeout"` // 单次尝试超时,缺省 2s
}

// WaitPolicy 是健康检查的等待重试策略。
type WaitPolicy struct {
	Timeout  duration `yaml:"timeout"`  // 总等待上限,缺省 60s
	Interval duration `yaml:"interval"` // 轮询间隔,缺省 2s
}

// duration 让 YAML 里写 "60s" / "2s" 这类时长(也可写纯数字 = 秒)。
// yaml.v3 不直接认 time.Duration,包一层自定义解析。
type duration time.Duration

// UnmarshalYAML 实现 yaml.Unmarshaler。先按纯数字(秒)解,解不成再按 "60s" 时长串解 ——
// yaml.v3 会把整数标量也宽松地解成字符串,故数字分支必须在前。
func (d *duration) UnmarshalYAML(value *yaml.Node) error {
	var n int64
	if err := value.Decode(&n); err == nil {
		*d = duration(time.Duration(n) * time.Second)
		return nil
	}
	var s string
	if err := value.Decode(&s); err != nil {
		return fmt.Errorf("时长须是 \"60s\" 这类字符串或秒数")
	}
	parsed, perr := time.ParseDuration(s)
	if perr != nil {
		return fmt.Errorf("时长 %q 解析失败: %w", s, perr)
	}
	*d = duration(parsed)
	return nil
}

// D 转回 time.Duration。
func (d duration) D() time.Duration { return time.Duration(d) }

// timeout 单次健康检查尝试的超时,缺省 2s。
func (c HealthCheck) timeout() time.Duration {
	if d := c.Timeout.D(); d > 0 {
		return d
	}
	return 2 * time.Second
}

// totalTimeout 健康检查总等待上限,缺省 60s。
func (w WaitPolicy) totalTimeout() time.Duration {
	if d := w.Timeout.D(); d > 0 {
		return d
	}
	return 60 * time.Second
}

// interval 健康检查轮询间隔,缺省 2s。
func (w WaitPolicy) interval() time.Duration {
	if d := w.Interval.D(); d > 0 {
		return d
	}
	return 2 * time.Second
}

// ParseDefinition 读并解析一份环境定义文件,计算内容哈希(INV-4)。
func ParseDefinition(path string) (Definition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Definition{}, fmt.Errorf("读环境定义 %s: %w", path, err)
	}
	def, err := ParseDefinitionBytes(data)
	if err != nil {
		return Definition{}, fmt.Errorf("解析环境定义 %s: %w", path, err)
	}
	def.Dir = filepath.Dir(path)
	return def, nil
}

// ParseDefinitionBytes 从字节解析;definition_hash = 整份内容的 sha256(INV-4)。
// 记的是内容哈希:同一个文件换名字/位置哈希不变,内容改一个字节哈希就变。
func ParseDefinitionBytes(data []byte) (Definition, error) {
	var def Definition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return Definition{}, fmt.Errorf("YAML 解析: %w", err)
	}
	sum := sha256.Sum256(data)
	def.Hash = "sha256:" + hex.EncodeToString(sum[:])
	return def, nil
}

// ComposeFilePath 返回 compose 文件路径(相对定义目录解析成绝对路径)。
func (d Definition) ComposeFilePath() string {
	if d.ComposeFile == "" || filepath.IsAbs(d.ComposeFile) || d.Dir == "" {
		return d.ComposeFile
	}
	return filepath.Join(d.Dir, d.ComposeFile)
}

// ComposeProject 返回 compose 项目名,缺省用定义名。
func (d Definition) ComposeProject() string {
	if d.ProjectName != "" {
		return d.ProjectName
	}
	return d.Name
}
