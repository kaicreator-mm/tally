package deploy

import (
	"context"
	"fmt"

	"tally/internal/runner"
)

// CommandRunner 是 deploy 需要的执行能力:跑外部工具的 CLI。
//
// 在使用方定义(见 docs/04-code-standards.md「接口」):runner 包的
// runner.LocalRunner / *runner.SSHRunner 结构上满足;单测注入替身离线验证命令构造。
type CommandRunner interface {
	Run(ctx context.Context, cmd runner.Command) (runner.Result, error)
}

// tool 是一个部署工具的适配器:把定义翻译成一条工具命令。
//
// 三个适配器都只做一件事 —— 组装 argv 交给 CommandRunner,然后读退出码。
// 零停机、健康等待、滚动更新、回滚逻辑全在工具里(铁律 6:不写别人写好的东西)。
type tool interface {
	name() string
	deployCmd(def Definition) runner.Command
	rollbackCmd(def Definition) runner.Command
}

// newTool 按定义选适配器。
func newTool(def Definition) (tool, error) {
	switch def.Tool {
	case ToolKamal:
		return kamalTool{}, nil
	case ToolAnsible:
		return ansibleTool{}, nil
	case ToolCompose:
		return composeTool{}, nil
	default:
		return nil, fmt.Errorf("工具 %q 不支持:只支持 kamal / ansible / compose", def.Tool)
	}
}

// kamalTool 转发 `kamal deploy / rollback`。-y 免交互确认(CLI 触发无人应答)。
// 版本用 -v 传制品 digest;目标环境用 kamal 的 destination(-d)。
type kamalTool struct{}

func (kamalTool) name() string { return ToolKamal }

func (kamalTool) deployCmd(def Definition) runner.Command {
	return kamalCmd("deploy", def)
}

func (kamalTool) rollbackCmd(def Definition) runner.Command {
	return kamalCmd("rollback", def)
}

func kamalCmd(action string, def Definition) runner.Command {
	return runner.Command{Argv: []string{
		"kamal", action, "-c", def.ConfigPath(), "-v", string(def.Digest), "-d", def.TargetEnv, "-y",
	}}
}

// ansibleTool 转发 `ansible-playbook`。digest 与目标环境经 -e 传进 playbook;
// 回滚走 --tags rollback(回滚逻辑写在 playbook 里,不在这里)。
type ansibleTool struct{}

func (ansibleTool) name() string { return ToolAnsible }

func (ansibleTool) deployCmd(def Definition) runner.Command {
	return ansibleCmd(def, nil)
}

func (ansibleTool) rollbackCmd(def Definition) runner.Command {
	return ansibleCmd(def, []string{"--tags", "rollback"})
}

func ansibleCmd(def Definition, extra []string) runner.Command {
	args := []string{"ansible-playbook"}
	if def.Inventory != "" {
		args = append(args, "-i", def.Inventory)
	}
	args = append(args, extra...)
	args = append(args,
		"-e", "artifact_digest="+string(def.Digest),
		"-e", "target_env="+def.TargetEnv,
		def.ConfigPath(),
	)
	return runner.Command{Argv: args}
}

// composeTool 转发 `docker compose up`。digest 经环境变量 ARTIFACT_DIGEST 交给清单引用;
// compose 项目名 = project-target_env → 部到隔离环境时与正在使用的环境天然分开。
// compose 没有内置回滚命令:回滚就是把目标 digest 再部一次(本包只转发)。
type composeTool struct{}

func (composeTool) name() string { return ToolCompose }

func (composeTool) deployCmd(def Definition) runner.Command {
	return runner.Command{
		Argv: []string{"docker", "compose", "-f", def.ConfigPath(), "-p", composeProject(def),
			"up", "-d", "--remove-orphans"},
		Env: []string{"ARTIFACT_DIGEST=" + string(def.Digest), "TARGET_ENV=" + def.TargetEnv},
	}
}

func (c composeTool) rollbackCmd(def Definition) runner.Command {
	return c.deployCmd(def)
}

func composeProject(def Definition) string {
	return def.Project + "-" + def.TargetEnv
}
