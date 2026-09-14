package deploy

import (
	"context"
	"reflect"
	"testing"

	"tally/internal/runner"
)

// 适配器验收:只转发命令、读退出码。用 fake 执行器断言组装出来的 argv,不真跑工具。

func defFor(tool string) Definition {
	return Definition{
		Project:   "myapp",
		TargetEnv: "staging",
		Tool:      tool,
		Digest:    "sha256:abc123",
		Config:    cfgName(tool),
		Inventory: "inv.ini",
	}
}

func cfgName(tool string) string {
	switch tool {
	case ToolKamal:
		return "kamal.yml"
	case ToolAnsible:
		return "site.yml"
	default:
		return "compose.yml"
	}
}

func TestKamalCommands(t *testing.T) {
	tl, err := newTool(defFor(ToolKamal))
	if err != nil {
		t.Fatal(err)
	}
	wantDeploy := []string{"kamal", "deploy", "-c", "kamal.yml", "-v", "sha256:abc123", "-d", "staging", "-y"}
	if got := tl.deployCmd(defFor(ToolKamal)).Argv; !reflect.DeepEqual(got, wantDeploy) {
		t.Errorf("kamal deploy 命令错:\n got %v\nwant %v", got, wantDeploy)
	}
	wantRollback := []string{"kamal", "rollback", "-c", "kamal.yml", "-v", "sha256:abc123", "-d", "staging", "-y"}
	if got := tl.rollbackCmd(defFor(ToolKamal)).Argv; !reflect.DeepEqual(got, wantRollback) {
		t.Errorf("kamal rollback 命令错:\n got %v\nwant %v", got, wantRollback)
	}
}

func TestAnsibleCommands(t *testing.T) {
	tl, err := newTool(defFor(ToolAnsible))
	if err != nil {
		t.Fatal(err)
	}
	def := defFor(ToolAnsible)
	wantDeploy := []string{"ansible-playbook", "-i", "inv.ini",
		"-e", "artifact_digest=sha256:abc123", "-e", "target_env=staging", "site.yml"}
	if got := tl.deployCmd(def).Argv; !reflect.DeepEqual(got, wantDeploy) {
		t.Errorf("ansible deploy 命令错:\n got %v\nwant %v", got, wantDeploy)
	}
	wantRollback := []string{"ansible-playbook", "-i", "inv.ini", "--tags", "rollback",
		"-e", "artifact_digest=sha256:abc123", "-e", "target_env=staging", "site.yml"}
	if got := tl.rollbackCmd(def).Argv; !reflect.DeepEqual(got, wantRollback) {
		t.Errorf("ansible rollback 命令错:\n got %v\nwant %v", got, wantRollback)
	}

	// inventory 可空:不带 -i。
	noInv := def
	noInv.Inventory = ""
	got := tl.deployCmd(noInv).Argv
	if got[0] != "ansible-playbook" || got[1] == "-i" {
		t.Errorf("无 inventory 时不应带 -i: %v", got)
	}
}

func TestComposeCommands(t *testing.T) {
	tl, err := newTool(defFor(ToolCompose))
	if err != nil {
		t.Fatal(err)
	}
	def := defFor(ToolCompose)
	cmd := tl.deployCmd(def)
	// compose 项目名含目标环境 → 与「正在使用的测试环境」天然隔离。
	wantArgv := []string{"docker", "compose", "-f", "compose.yml", "-p", "myapp-staging",
		"up", "-d", "--remove-orphans"}
	if !reflect.DeepEqual(cmd.Argv, wantArgv) {
		t.Errorf("compose deploy 命令错:\n got %v\nwant %v", cmd.Argv, wantArgv)
	}
	wantEnv := []string{"ARTIFACT_DIGEST=sha256:abc123", "TARGET_ENV=staging"}
	if !reflect.DeepEqual(cmd.Env, wantEnv) {
		t.Errorf("compose 应经环境变量把 digest 传给清单:\n got %v\nwant %v", cmd.Env, wantEnv)
	}
	// compose 无内置回滚命令:回滚 = 用目标 digest 再部一次(转发,不是引擎)。
	if rb := tl.rollbackCmd(def).Argv; !reflect.DeepEqual(rb, wantArgv) {
		t.Errorf("compose rollback 应等价于再部署目标 digest:\n got %v\nwant %v", rb, wantArgv)
	}
}

// 验收:隔离环境。target_env 一换,命令里的目标随之变(不复用原环境)。
func TestToolCommandFollowsTargetEnv(t *testing.T) {
	def := defFor(ToolKamal)
	def.TargetEnv = "verify-iso-42"
	tl, _ := newTool(def)
	cmd := tl.deployCmd(def)
	found := false
	for _, a := range cmd.Argv {
		if a == "verify-iso-42" {
			found = true
		}
	}
	if !found {
		t.Errorf("换到隔离环境后,命令应指向 verify-iso-42: %v", cmd.Argv)
	}
}

// runner.LocalRunner 结构上满足 CommandRunner(接口在使用方定义,实现由 runner 包提供)。
var _ CommandRunner = runner.LocalRunner{}

// newTool 拒绝未知工具。
func TestNewToolRejectsUnknown(t *testing.T) {
	def := defFor(ToolKamal)
	def.Tool = "kubectl"
	if _, err := newTool(def); err == nil {
		t.Error("未知工具应报错")
	}
}

// fakeExec 记录收到的命令,返回预设结果。deploy 的单测替身。
type fakeExec struct {
	cmds []runner.Command
	res  runner.Result
	err  error
}

func (f *fakeExec) Run(_ context.Context, cmd runner.Command) (runner.Result, error) {
	f.cmds = append(f.cmds, cmd)
	return f.res, f.err
}
