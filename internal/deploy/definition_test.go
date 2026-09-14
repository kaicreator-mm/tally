package deploy

import (
	"strings"
	"testing"
)

// 合法定义:已存在的制品 digest + 目标环境 + 工具。解析应成功,并算出内容哈希(INV-4)。
const validYAML = `project: myapp
target_env: staging
tool: kamal
artifact_digest: sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
config: deploy/kamal.yml
`

func TestParseDefinitionValid(t *testing.T) {
	def, err := ParseDefinitionBytes([]byte(validYAML))
	if err != nil {
		t.Fatalf("合法定义应解析成功: %v", err)
	}
	if def.Project != "myapp" || def.TargetEnv != "staging" || def.Tool != "kamal" {
		t.Errorf("字段解析错: %+v", def)
	}
	if def.Digest == "" {
		t.Error("artifact_digest 应解析出来")
	}
	if def.Config != "deploy/kamal.yml" {
		t.Errorf("config 应解析出来: %q", def.Config)
	}
	if !strings.HasPrefix(def.Hash, "sha256:") {
		t.Errorf("应算出内容哈希(INV-4): %q", def.Hash)
	}
}

// 验收:定义里出现「拉代码后构建」形态 → 解析时报错。部署只接受已存在的制品 digest。
func TestParseDefinitionRejectsBuildFromSource(t *testing.T) {
	cases := map[string]string{
		"repo+build": `project: p
target_env: e
tool: ansible
artifact_digest: sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
config: site.yml
repo: https://git.example.com/app.git
build: make release
`,
		"git 字段": `project: p
target_env: e
tool: kamal
artifact_digest: sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc
config: kamal.yml
git: git@git.example.com:app.git
`,
		"build_command 字段": `project: p
target_env: e
tool: compose
artifact_digest: sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd
config: compose.yml
build_command: docker build -t app .
`,
		"只有 digest 缺了也不行,拉代码形态必须报错": `project: p
target_env: e
tool: ansible
config: site.yml
repo: https://git.example.com/app.git
build: make release
`,
	}
	for name, yaml := range cases {
		if _, err := ParseDefinitionBytes([]byte(yaml)); err == nil {
			t.Errorf("%s: 「拉代码后构建」形态应解析报错", name)
		} else if !strings.Contains(err.Error(), "拉代码后构建") {
			t.Errorf("%s: 错误应点明是「拉代码后构建」: %v", name, err)
		}
	}
}

// 验收:输入必须是已存在的制品 digest —— digest 为空 → 解析报错。
func TestParseDefinitionRequiresDigest(t *testing.T) {
	yaml := `project: p
target_env: e
tool: kamal
config: kamal.yml
`
	if _, err := ParseDefinitionBytes([]byte(yaml)); err == nil {
		t.Fatal("缺 artifact_digest 应报错:没有 digest 的部署不允许发生")
	}
}

func TestParseDefinitionValidatesToolAndTarget(t *testing.T) {
	badTool := `project: p
target_env: e
tool: kubectl
artifact_digest: sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee
config: x.yml
`
	if _, err := ParseDefinitionBytes([]byte(badTool)); err == nil {
		t.Error("未知工具应报错(只支持 kamal/ansible/compose)")
	}
	noTarget := `project: p
tool: kamal
artifact_digest: sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff
config: x.yml
`
	if _, err := ParseDefinitionBytes([]byte(noTarget)); err == nil {
		t.Error("缺 target_env 应报错")
	}
	noConfig := `project: p
target_env: e
tool: kamal
artifact_digest: sha256:1111111111111111111111111111111111111111111111111111111111111111
`
	if _, err := ParseDefinitionBytes([]byte(noConfig)); err == nil {
		t.Error("缺 config(工具清单)应报错")
	}
}

// INV-4:哈希是内容的函数。改一个字节,哈希必须变。
func TestDefinitionHashChangesWithContent(t *testing.T) {
	a, err := ParseDefinitionBytes([]byte(validYAML))
	if err != nil {
		t.Fatal(err)
	}
	b, err := ParseDefinitionBytes([]byte(strings.Replace(validYAML, "staging", "prod", 1)))
	if err != nil {
		t.Fatal(err)
	}
	if a.Hash == b.Hash {
		t.Error("内容变了,定义哈希必须变(INV-4)")
	}
}

// 目标环境可覆盖:重跑验证部到隔离环境,不复用正在使用的测试环境。
func TestDefinitionTargetEnvOverride(t *testing.T) {
	def, err := ParseDefinitionBytes([]byte(validYAML))
	if err != nil {
		t.Fatal(err)
	}
	def.TargetEnv = "verify-iso-1"
	if err := def.Validate(); err != nil {
		t.Fatalf("覆盖 target_env 后仍应合法: %v", err)
	}
	if def.TargetEnv != "verify-iso-1" {
		t.Error("target_env 应能被调用方覆盖到隔离环境")
	}
}
