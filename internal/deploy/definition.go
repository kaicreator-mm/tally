package deploy

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"tally/internal/core"
)

// Definition 是一份部署定义(.tally/deploys/*.yaml,和环境定义放一起,PRD 第二部分规则 ②)。
//
// 输入必须是已存在的制品 digest(规则 ①):部署的对象是一个镜像 digest 或打好的包,
// 不是「拉代码然后 build」。定义里出现任何拉源码/构建形态的字段 → 解析即报错。
type Definition struct {
	Project   string              `yaml:"project"`         // 项目 id
	TargetEnv string              `yaml:"target_env"`      // 目标环境名;重跑验证时覆盖成隔离环境
	Tool      string              `yaml:"tool"`            // kamal / ansible / compose
	Digest    core.ArtifactDigest `yaml:"artifact_digest"` // 必填:已存在制品的 digest
	BuildRun  int64               `yaml:"build_run_id"`    // 可空:产出该制品的 build_run(制品可能来自外部)
	Config    string              `yaml:"config"`          // 工具清单:kamal 配置 / ansible playbook / compose 文件
	Inventory string              `yaml:"inventory"`       // ansible 可选:inventory 文件

	// 禁止字段:「拉代码后构建」形态(规则 ①)。解析时任一非空即报错。
	Repo         string `yaml:"repo"`
	Git          string `yaml:"git"`
	Clone        string `yaml:"clone"`
	Build        string `yaml:"build"`
	BuildCommand string `yaml:"build_command"`
	BuildScript  string `yaml:"build_script"`

	// 非 YAML 字段,解析时填。
	Hash string `yaml:"-"` // 定义内容哈希(INV-4)
	Dir  string `yaml:"-"` // 定义文件所在目录,解析相对路径用
}

// 支持的工具(deploy_run.tool 的 CHECK 约束同此三者)。
const (
	ToolKamal   = "kamal"
	ToolAnsible = "ansible"
	ToolCompose = "compose"
)

// ParseDefinition 读并解析一份部署定义文件,计算内容哈希(INV-4)。
func ParseDefinition(path string) (Definition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Definition{}, fmt.Errorf("读部署定义 %s: %w", path, err)
	}
	def, err := ParseDefinitionBytes(data)
	if err != nil {
		return Definition{}, fmt.Errorf("解析部署定义 %s: %w", path, err)
	}
	def.Dir = filepath.Dir(path)
	return def, nil
}

// ParseDefinitionBytes 从字节解析并校验;definition hash = 整份内容的 sha256(INV-4)。
func ParseDefinitionBytes(data []byte) (Definition, error) {
	var def Definition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return Definition{}, fmt.Errorf("YAML 解析: %w", err)
	}
	sum := sha256.Sum256(data)
	def.Hash = "sha256:" + hex.EncodeToString(sum[:])
	return def, def.Validate()
}

// Validate 校验定义的硬约束。解析时调用;调用方覆盖字段(如 target_env 换成隔离环境)后再调一次。
func (d Definition) Validate() error {
	for _, f := range []struct{ name, val string }{
		{"repo", d.Repo}, {"git", d.Git}, {"clone", d.Clone},
		{"build", d.Build}, {"build_command", d.BuildCommand}, {"build_script", d.BuildScript},
	} {
		if f.val != "" {
			return fmt.Errorf("部署定义出现「拉代码后构建」形态(字段 %s=%q):部署只接受已存在的制品 digest,不接受在目标机上拉源码构建(PRD 第二部分规则 ①)", f.name, f.val)
		}
	}
	if d.Digest == "" {
		return fmt.Errorf("artifact_digest 必填:没有 digest 的部署不允许发生(PRD 第二部分规则 ①)")
	}
	switch d.Tool {
	case ToolKamal, ToolAnsible, ToolCompose:
	default:
		return fmt.Errorf("工具 %q 不支持:只支持 kamal / ansible / compose", d.Tool)
	}
	if d.Project == "" {
		return fmt.Errorf("project 必填")
	}
	if d.TargetEnv == "" {
		return fmt.Errorf("target_env 必填:部署必须指明目标环境")
	}
	if d.Config == "" {
		return fmt.Errorf("config 必填:kamal 配置文件 / ansible playbook / compose 文件")
	}
	return nil
}

// ConfigPath 返回工具清单路径(相对定义目录解析成绝对路径)。
func (d Definition) ConfigPath() string {
	if d.Config == "" || filepath.IsAbs(d.Config) || d.Dir == "" {
		return d.Config
	}
	return filepath.Join(d.Dir, d.Config)
}
