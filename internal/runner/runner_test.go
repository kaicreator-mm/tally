package runner

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"
)

// 本机没有保证存在的跨平台小程序(Windows 上 echo 是 shell 内建),
// 用标准做法:让测试二进制自己当子进程,靠环境变量决定行为。

func helperCommand(t *testing.T, env ...string) Command {
	t.Helper()
	return Command{
		Argv: []string{os.Args[0], "-test.run=TestHelperProcess"},
		Env:  append([]string{"GO_WANT_HELPER_PROCESS=1"}, env...),
	}
}

// TestHelperProcess 不是真正的测试:被当作子进程执行时按 HELPER_* 环境变量输出并退出。
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	if ms := os.Getenv("HELPER_SLEEP_MS"); ms != "" {
		d, _ := strconv.Atoi(ms)
		time.Sleep(time.Duration(d) * time.Millisecond)
	}
	fmt.Fprint(os.Stdout, os.Getenv("HELPER_STDOUT"))
	fmt.Fprint(os.Stderr, os.Getenv("HELPER_STDERR"))
	code, _ := strconv.Atoi(os.Getenv("HELPER_EXIT"))
	os.Exit(code)
}

func TestLocalRunnerCapturesStdoutStderr(t *testing.T) {
	res, err := LocalRunner{}.Run(context.Background(),
		helperCommand(t, "HELPER_STDOUT=out-text", "HELPER_STDERR=err-text", "HELPER_EXIT=0"))
	if err != nil {
		t.Fatalf("退出码 0 不该有 error: %v", err)
	}
	if res.Stdout != "out-text" || res.Stderr != "err-text" {
		t.Errorf("stdout/stderr 没捕获对: %q / %q", res.Stdout, res.Stderr)
	}
	if res.ExitCode != 0 {
		t.Errorf("退出码应是 0, got %d", res.ExitCode)
	}
}

// 契约核心:命令跑完但退出码非零,不是 error,落在 ExitCode 里(判断成败是调用方的事)。
func TestLocalRunnerNonZeroExitIsNotError(t *testing.T) {
	res, err := LocalRunner{}.Run(context.Background(),
		helperCommand(t, "HELPER_STDERR=boom", "HELPER_EXIT=3"))
	if err != nil {
		t.Fatalf("非零退出码不应返回 error: %v", err)
	}
	if res.ExitCode != 3 {
		t.Errorf("退出码应是 3, got %d", res.ExitCode)
	}
}

func TestLocalRunnerEmptyArgv(t *testing.T) {
	if _, err := (LocalRunner{}).Run(context.Background(), Command{}); err == nil {
		t.Error("空命令应返回 error")
	}
}

func TestLocalRunnerContextCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := LocalRunner{}.Run(ctx, helperCommand(t, "HELPER_SLEEP_MS=5000"))
	if err == nil {
		t.Error("ctx 超时应返回 error(没跑完)")
	}
}

// fakeRunner 记录收到的命令并回固定结果,给 DockerRunner 单测注入。
type fakeRunner struct {
	got Command
	res Result
	err error
}

func (f *fakeRunner) Run(_ context.Context, cmd Command) (Result, error) {
	f.got = cmd
	return f.res, f.err
}

func argvOf(r *fakeRunner) []string { return r.got.Argv }

func TestDockerRunnerRunPrependsBin(t *testing.T) {
	f := &fakeRunner{}
	d := &DockerRunner{Bin: "docker", Exec: f}
	if _, err := d.Run(context.Background(), Command{Argv: []string{"ps"}}); err != nil {
		t.Fatal(err)
	}
	if got := argvOf(f); len(got) != 2 || got[0] != "docker" || got[1] != "ps" {
		t.Errorf("应在参数前补 docker: %v", got)
	}
}

func TestDockerRunnerComposeUpArgs(t *testing.T) {
	f := &fakeRunner{}
	d := &DockerRunner{Exec: f}
	if _, err := d.ComposeUp(context.Background(), "c.yml", "proj"); err != nil {
		t.Fatal(err)
	}
	want := []string{"docker", "compose", "-f", "c.yml", "-p", "proj", "up", "-d", "--remove-orphans"}
	assertArgv(t, argvOf(f), want)
}

func TestDockerRunnerComposeDownRemovesVolumes(t *testing.T) {
	f := &fakeRunner{}
	d := &DockerRunner{Exec: f}
	if _, err := d.ComposeDown(context.Background(), "c.yml", "proj"); err != nil {
		t.Fatal(err)
	}
	// 验收:推倒不残留卷和网络 → 必须带 --volumes。
	want := []string{"docker", "compose", "-f", "c.yml", "-p", "proj", "down", "--volumes", "--remove-orphans"}
	assertArgv(t, argvOf(f), want)
}

func TestDockerRunnerComposeImages(t *testing.T) {
	f := &fakeRunner{res: Result{Stdout: "alpine:latest\nredis:8.2\n\n"}}
	d := &DockerRunner{Exec: f}
	imgs, err := d.ComposeImages(context.Background(), "c.yml", "proj")
	if err != nil {
		t.Fatal(err)
	}
	if len(imgs) != 2 || imgs[0] != "alpine:latest" || imgs[1] != "redis:8.2" {
		t.Errorf("镜像清单解析错: %v", imgs)
	}
}

func TestDockerRunnerComposeImagesNonZeroFails(t *testing.T) {
	f := &fakeRunner{res: Result{ExitCode: 1, Stderr: "bad compose file"}}
	d := &DockerRunner{Exec: f}
	if _, err := d.ComposeImages(context.Background(), "c.yml", "proj"); err == nil {
		t.Error("非零退出码应上抛 error")
	}
}

func TestDockerRunnerVersions(t *testing.T) {
	f := &fakeRunner{res: Result{Stdout: " 29.6.1\n"}}
	d := &DockerRunner{Exec: f}
	v, err := d.ServerVersion(context.Background())
	if err != nil || v != "29.6.1" {
		t.Errorf("ServerVersion = %q, %v", v, err)
	}
	f.res = Result{Stdout: "5.3.0\n"}
	cv, err := d.ComposeVersion(context.Background())
	if err != nil || cv != "5.3.0" {
		t.Errorf("ComposeVersion = %q, %v", cv, err)
	}
}

func TestSSHRunnerJoinsCommand(t *testing.T) {
	var gotCfg SSHConfig
	var gotCmd string
	r := &SSHRunner{
		Config: SSHConfig{Host: "h", User: "u", Password: "p"},
		RunRemote: func(_ context.Context, cfg SSHConfig, command string) (Result, error) {
			gotCfg, gotCmd = cfg, command
			return Result{Stdout: "ok"}, nil
		},
	}
	res, err := r.Run(context.Background(), Command{Argv: []string{"echo", "hi", "there"}})
	if err != nil {
		t.Fatal(err)
	}
	if gotCmd != "echo hi there" {
		t.Errorf("应拼成一条命令: %q", gotCmd)
	}
	if gotCfg.Host != "h" {
		t.Errorf("配置应透传: %+v", gotCfg)
	}
	if res.Stdout != "ok" {
		t.Errorf("结果应透传: %+v", res)
	}
}

func TestSSHAuthMethods(t *testing.T) {
	if _, err := authMethods(SSHConfig{}); err == nil {
		t.Error("无认证方式应报错")
	}
	if m, err := authMethods(SSHConfig{Password: "p"}); err != nil || len(m) != 1 {
		t.Errorf("口令应产出 1 个认证方式: %v %v", len(m), err)
	}
	if _, err := authMethods(SSHConfig{KeyFile: "/nonexistent/key"}); err == nil {
		t.Error("读不到私钥应报错")
	}
}

func assertArgv(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("argv 长度不对:\n got  %v\n want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("argv[%d] = %q, want %q\n got  %v\n want %v", i, got[i], want[i], got, want)
		}
	}
}
