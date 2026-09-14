package runner

import (
	"context"
	"fmt"
	"strings"
)

// DockerRunner 用 shell 调 `docker` / `docker compose` CLI(**不用 SDK**,见技术栈约束)。
//
// 每个操作都委托给底层 Exec(默认 LocalRunner),单测注入替身即可离线验证
// 组装出来的 docker 参数,不必真有 docker 守护进程。
type DockerRunner struct {
	Bin  string // docker 可执行文件名,空则 "docker"
	Exec Runner // 底层执行器,空则 LocalRunner{}
}

// NewDockerRunner 返回用本机 docker CLI 的执行器。
func NewDockerRunner() *DockerRunner {
	return &DockerRunner{Bin: "docker", Exec: LocalRunner{}}
}

func (d *DockerRunner) bin() string {
	if d.Bin == "" {
		return "docker"
	}
	return d.Bin
}

func (d *DockerRunner) exec() Runner {
	if d.Exec == nil {
		return LocalRunner{}
	}
	return d.Exec
}

// Run 执行一条 docker 子命令:cmd.Argv 是 `docker` 之后的参数
// (如 ["compose","-f","x.yml","up","-d"])。原始结果透传,非零退出码不作 error。
func (d *DockerRunner) Run(ctx context.Context, cmd Command) (Result, error) {
	return d.exec().Run(ctx, Command{
		Argv: append([]string{d.bin()}, cmd.Argv...),
		Dir:  cmd.Dir,
		Env:  cmd.Env,
	})
}

// composeArgs 组装 `compose -f <file> -p <project>` 的前缀参数。
func composeArgs(file, project string) []string {
	args := []string{"compose"}
	if file != "" {
		args = append(args, "-f", file)
	}
	if project != "" {
		args = append(args, "-p", project)
	}
	return args
}

// ComposeUp 后台拉起 compose 定义(--remove-orphans 清掉定义里已删除的服务)。
func (d *DockerRunner) ComposeUp(ctx context.Context, file, project string) (Result, error) {
	return d.Run(ctx, Command{Argv: append(composeArgs(file, project), "up", "-d", "--remove-orphans")})
}

// ComposeDown 推倒 compose 定义:--volumes 删命名卷、--remove-orphans 清孤儿,
// 网络随项目一并移除 —— 不残留卷和网络(验收要求)。
func (d *DockerRunner) ComposeDown(ctx context.Context, file, project string) (Result, error) {
	return d.Run(ctx, Command{Argv: append(composeArgs(file, project), "down", "--volumes", "--remove-orphans")})
}

// Pull 预拉镜像,让随后的 ComposeImages/inspect 读得到 tag。
func (d *DockerRunner) Pull(ctx context.Context, image string) (Result, error) {
	return d.Run(ctx, Command{Argv: []string{"pull", image}})
}

// ComposeImages 列出 compose 定义用到的镜像引用(每行一个,含 tag),供 version_combo 记录。
// 非零退出码视为失败上抛:拿不到镜像清单就没法记录版本组合。
func (d *DockerRunner) ComposeImages(ctx context.Context, file, project string) ([]string, error) {
	res, err := d.Run(ctx, Command{Argv: append(composeArgs(file, project), "config", "--images")})
	if err != nil {
		return nil, err
	}
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("docker compose config --images 退出码 %d: %s",
			res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	var imgs []string
	for _, line := range strings.Split(res.Stdout, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			imgs = append(imgs, line)
		}
	}
	return imgs, nil
}

// ComposeLogs 取 compose 某服务的日志(service 为空则全部)。log-pattern 健康检查用它。
func (d *DockerRunner) ComposeLogs(ctx context.Context, file, project, service string) (Result, error) {
	args := append(composeArgs(file, project), "logs", "--no-color")
	if service != "" {
		args = append(args, service)
	}
	return d.Run(ctx, Command{Argv: args})
}

// ServerVersion 返回 docker 服务端版本(运行时版本之一,记进 version_combo)。
func (d *DockerRunner) ServerVersion(ctx context.Context) (string, error) {
	res, err := d.Run(ctx, Command{Argv: []string{"version", "--format", "{{.Server.Version}}"}})
	return d.trimOutput(res, err, "docker version")
}

// ComposeVersion 返回 docker compose 插件版本。
func (d *DockerRunner) ComposeVersion(ctx context.Context) (string, error) {
	res, err := d.Run(ctx, Command{Argv: []string{"compose", "version", "--short"}})
	return d.trimOutput(res, err, "docker compose version")
}

// trimOutput 把 (Result, error) 收成「去空白后的 stdout」;非零退出码上抛。
func (d *DockerRunner) trimOutput(res Result, err error, what string) (string, error) {
	if err != nil {
		return "", err
	}
	if res.ExitCode != 0 {
		return "", fmt.Errorf("%s 退出码 %d: %s", what, res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	return strings.TrimSpace(res.Stdout), nil
}
