package environment

import (
	"errors"
	"testing"
)

// 验收核心:缺健康检查的定义,注册时返回 ErrNoHealthCheck,拒绝注册(不是警告)。
func TestRegisterRejectsNoHealthCheck(t *testing.T) {
	def, err := ParseDefinitionBytes([]byte("name: dev\nproject: demo\nkind: docker-compose\ncompose_file: c.yml\n"))
	if err != nil {
		t.Fatal(err)
	}
	r := NewRegistry()
	err = r.Register(def)
	if !errors.Is(err, ErrNoHealthCheck) {
		t.Fatalf("缺健康检查应返回 ErrNoHealthCheck, got %v", err)
	}
	if _, ok := r.Get("demo", "dev"); ok {
		t.Error("拒绝注册的定义不应被登记")
	}
}

func TestRegisterAcceptsValid(t *testing.T) {
	def, err := ParseDefinitionBytes([]byte(composeYAML))
	if err != nil {
		t.Fatal(err)
	}
	r := NewRegistry()
	if err := r.Register(def); err != nil {
		t.Fatalf("合法定义应注册成功: %v", err)
	}
	if _, ok := r.Get("demo", "dev"); !ok {
		t.Error("注册后应能取到")
	}
}

func TestRegisterValidatesKindAndFields(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{"未知 kind", "name: x\nproject: p\nkind: k8s\ncompose_file: c.yml\nhealth_checks:\n  - {name: a, type: tcp, host: h, port: 1}\n"},
		{"compose 缺 compose_file", "name: x\nproject: p\nkind: docker-compose\nhealth_checks:\n  - {name: a, type: tcp, host: h, port: 1}\n"},
		{"ssh 缺 script", "name: x\nproject: p\nkind: ssh-script\nssh: {host: h, user: u}\nhealth_checks:\n  - {name: a, type: tcp, host: h, port: 1}\n"},
		{"ssh 缺 host", "name: x\nproject: p\nkind: ssh-script\nssh: {user: u}\nscript: echo\nhealth_checks:\n  - {name: a, type: tcp, host: h, port: 1}\n"},
		{"健康检查类型未知", "name: x\nproject: p\nkind: docker-compose\ncompose_file: c.yml\nhealth_checks:\n  - {name: a, type: grpc, host: h, port: 1}\n"},
		{"tcp 缺 port", "name: x\nproject: p\nkind: docker-compose\ncompose_file: c.yml\nhealth_checks:\n  - {name: a, type: tcp, host: h}\n"},
		{"http 缺 url", "name: x\nproject: p\nkind: docker-compose\ncompose_file: c.yml\nhealth_checks:\n  - {name: a, type: http}\n"},
		{"log-pattern 缺 pattern", "name: x\nproject: p\nkind: docker-compose\ncompose_file: c.yml\nhealth_checks:\n  - {name: a, type: log-pattern}\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			def, err := ParseDefinitionBytes([]byte(tc.yaml))
			if err != nil {
				t.Fatalf("YAML 本身应能解析: %v", err)
			}
			if err := NewRegistry().Register(def); err == nil {
				t.Error("应拒绝注册")
			}
		})
	}
}

// Validate 可独立于 Registry 使用(harvest 抽定义时先过这道)。
func TestValidateStandalone(t *testing.T) {
	def, _ := ParseDefinitionBytes([]byte(composeYAML))
	if err := Validate(def); err != nil {
		t.Errorf("合法定义 Validate 应过: %v", err)
	}
}
