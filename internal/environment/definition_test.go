package environment

import (
	"strings"
	"testing"
	"time"
)

const composeYAML = `name: dev
project: demo
kind: docker-compose
compose_file: ./dc.yml
project_name: demo-dev
health_checks:
  - name: web
    type: http
    url: http://localhost:8080/healthz
    expect_body: ok
wait:
  timeout: 30s
  interval: 1s
`

const sshYAML = `name: box
project: demo
kind: ssh-script
ssh:
  host: 10.0.0.5
  port: 2222
  user: deploy
  key_file: /tmp/key
script: |
  echo up
log_command: cat /var/log/app.log
teardown: |
  echo down
health_checks:
  - name: port
    type: tcp
    host: 10.0.0.5
    port: 5432
`

func TestParseComposeDefinition(t *testing.T) {
	def, err := ParseDefinitionBytes([]byte(composeYAML))
	if err != nil {
		t.Fatalf("解析: %v", err)
	}
	if def.Name != "dev" || def.Project != "demo" || def.Kind != KindDockerCompose {
		t.Errorf("基本字段解析错: %+v", def)
	}
	if def.ComposeFile != "./dc.yml" || def.ProjectName != "demo-dev" {
		t.Errorf("compose 字段解析错: %+v", def)
	}
	if len(def.HealthChecks) != 1 || def.HealthChecks[0].Type != CheckHTTP || def.HealthChecks[0].ExpectBody != "ok" {
		t.Errorf("健康检查解析错: %+v", def.HealthChecks)
	}
	if def.Wait.totalTimeout() != 30*time.Second || def.Wait.interval() != time.Second {
		t.Errorf("等待策略解析错: %v / %v", def.Wait.totalTimeout(), def.Wait.interval())
	}
	if !strings.HasPrefix(def.Hash, "sha256:") || len(def.Hash) != len("sha256:")+64 {
		t.Errorf("definition_hash 应是 sha256:<64hex>: %q", def.Hash)
	}
}

func TestParseSSHDefinition(t *testing.T) {
	def, err := ParseDefinitionBytes([]byte(sshYAML))
	if err != nil {
		t.Fatalf("解析: %v", err)
	}
	if def.Kind != KindSSHScript || def.SSH.Host != "10.0.0.5" || def.SSH.Port != 2222 || def.SSH.User != "deploy" {
		t.Errorf("ssh 字段解析错: %+v", def)
	}
	if !strings.Contains(def.Script, "echo up") || !strings.Contains(def.Teardown, "echo down") {
		t.Errorf("脚本解析错: %q / %q", def.Script, def.Teardown)
	}
}

// INV-4:definition_hash 是内容哈希 —— 内容变一个字节哈希就变,相同内容哈希不变。
func TestDefinitionHashIsContentHash(t *testing.T) {
	d1, err := ParseDefinitionBytes([]byte(composeYAML))
	if err != nil {
		t.Fatal(err)
	}
	d2, err := ParseDefinitionBytes([]byte(composeYAML))
	if err != nil {
		t.Fatal(err)
	}
	if d1.Hash != d2.Hash {
		t.Errorf("相同内容哈希应相同: %s vs %s", d1.Hash, d2.Hash)
	}
	// 改一个字节(项目名),哈希必须变。
	changed := strings.Replace(composeYAML, "project: demo", "project: dem0", 1)
	d3, err := ParseDefinitionBytes([]byte(changed))
	if err != nil {
		t.Fatal(err)
	}
	if d3.Hash == d1.Hash {
		t.Error("内容变了哈希必须变(INV-4)")
	}
}

func TestDurationDefaults(t *testing.T) {
	var def Definition // 零值
	if def.Wait.totalTimeout() != 60*time.Second {
		t.Errorf("默认总超时应 60s: %v", def.Wait.totalTimeout())
	}
	if def.Wait.interval() != 2*time.Second {
		t.Errorf("默认间隔应 2s: %v", def.Wait.interval())
	}
	hc := HealthCheck{}
	if hc.timeout() != 2*time.Second {
		t.Errorf("默认单次超时应 2s: %v", hc.timeout())
	}
}

func TestDurationParsesPlainSeconds(t *testing.T) {
	def, err := ParseDefinitionBytes([]byte("name: x\nkind: docker-compose\ncompose_file: c.yml\nwait:\n  timeout: 45\nhealth_checks:\n  - {name: a, type: tcp, host: h, port: 1}\n"))
	if err != nil {
		t.Fatal(err)
	}
	if def.Wait.totalTimeout() != 45*time.Second {
		t.Errorf("纯数字应按秒解析: %v", def.Wait.totalTimeout())
	}
}

func TestComposePathAndProject(t *testing.T) {
	def, _ := ParseDefinitionBytes([]byte(composeYAML))
	def.Dir = "/repo/.tally/environments"
	if got := def.ComposeFilePath(); got != "/repo/.tally/environments/dc.yml" && !strings.HasSuffix(got, "dc.yml") {
		t.Errorf("compose 路径应相对定义目录解析: %q", got)
	}
	if def.ComposeProject() != "demo-dev" {
		t.Errorf("项目名应优先用 project_name: %q", def.ComposeProject())
	}
	def.ProjectName = ""
	if def.ComposeProject() != "dev" {
		t.Errorf("缺省项目名应用定义名: %q", def.ComposeProject())
	}
}
