package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"tally/internal/command"
	"tally/internal/runner"
)

// fakeRunner 捕获执行器收到的结构化命令,按预设退出码返回(退出码 0 = 绿)。
type fakeRunner struct {
	got  runner.Command
	code int
}

func (f *fakeRunner) Run(ctx context.Context, cmd runner.Command) (runner.Result, error) {
	f.got = cmd
	return runner.Result{ExitCode: f.code}, nil
}

// P0-15 AC1:历史 TEXT repro 的 argv 拆分不再按空白直切(shell 字符串拆分已移除)。
// 只认空白分隔 + 成对引号;引号内空白保持字面,引号本身不进参数。
func TestLegacyReproArgv(t *testing.T) {
	cases := []struct {
		name  string
		repro string
		want  []string
	}{
		{"普通空白", "go test ./internal/deps/...", []string{"go", "test", "./internal/deps/..."}},
		{"连续空白与制表符", "  go\t test  ./... \n", []string{"go", "test", "./..."}},
		{"双引号含空格", `go test -run "X Y" ./...`, []string{"go", "test", "-run", "X Y", "./..."}},
		{"单引号含空格", `echo 'a b'`, []string{"echo", "a b"}},
		{"引号贴参", `go test -run="X"./...`, []string{`go`, `test`, `-run=X./...`}},
		{"空引号保留空参数", `prog "" x`, []string{"prog", "", "x"}},
		{"空串", "", nil},
		{"纯空白", " \t\n ", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := legacyReproArgv(c.repro)
			if err != nil {
				t.Fatalf("legacyReproArgv(%q): %v", c.repro, err)
			}
			if len(got) != len(c.want) {
				t.Fatalf("legacyReproArgv(%q) = %q, 期望 %q", c.repro, got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Fatalf("legacyReproArgv(%q) = %q, 期望 %q", c.repro, got, c.want)
				}
			}
		})
	}
}

// 引号不闭合是数据缺陷:报错,不猜。
func TestLegacyReproArgvUnterminatedQuote(t *testing.T) {
	if _, err := legacyReproArgv(`go test -run "X Y`); err == nil {
		t.Fatal("引号未闭合应报错")
	}
}

// P0-15 AC2:repro 装配成结构化 CommandSpec —— argv/cwd 就位,shell 恒 false,timeout 未设。
func TestReproSpecStructured(t *testing.T) {
	spec, err := reproSpec("C:\\work", `go test -run "X Y" ./...`)
	if err != nil {
		t.Fatalf("reproSpec: %v", err)
	}
	want := command.CommandSpec{
		Argv:  []string{"go", "test", "-run", "X Y", "./..."},
		Cwd:   "C:\\work",
		Shell: false,
	}
	if spec.Cwd != want.Cwd || spec.Shell != want.Shell || spec.Timeout != 0 {
		t.Fatalf("spec 字段不符: %+v", spec)
	}
	if len(spec.Argv) != len(want.Argv) {
		t.Fatalf("argv = %q, 期望 %q", spec.Argv, want.Argv)
	}
	for i := range spec.Argv {
		if spec.Argv[i] != want.Argv[i] {
			t.Fatalf("argv = %q, 期望 %q", spec.Argv, want.Argv)
		}
	}
	if err := spec.Validate(); err != nil {
		t.Fatalf("spec 应通过 Validate: %v", err)
	}
}

// 空/纯空白 repro 拆不出合法 argv:拒绝,不静默执行空命令(错误链含 ErrEmptyArgv)。
func TestReproSpecRejectsEmptyRepro(t *testing.T) {
	if _, err := reproSpec("C:\\work", "   "); !errors.Is(err, command.ErrEmptyArgv) {
		t.Fatalf("期望 ErrEmptyArgv, got %v", err)
	}
}

// P0-15 AC3:退休语义不变 —— 退出码 0 = 绿(不再复现),非 0 = 红(仍复现);
// 执行器收到的是结构化 argv(引号已拆)与 cwd,不带任何 shell 字符串。
func TestLocalReproRunnerExecutesSpec(t *testing.T) {
	t.Run("红=仍复现", func(t *testing.T) {
		fr := &fakeRunner{code: 1}
		green, err := (localReproRunner{local: fr}).Run(context.Background(), "C:\\work", "go test ./...")
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		if green {
			t.Fatal("退出码 1 应为红(仍复现)")
		}
	})
	t.Run("绿=不再复现", func(t *testing.T) {
		fr := &fakeRunner{code: 0}
		green, err := (localReproRunner{local: fr}).Run(context.Background(), "C:\\work", `go test -run "X Y" ./...`)
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		if !green {
			t.Fatal("退出码 0 应为绿(不再复现)")
		}
		if len(fr.got.Argv) != 5 || fr.got.Argv[3] != "X Y" {
			t.Fatalf("执行器应收拆好的 argv, got %q", fr.got.Argv)
		}
		if fr.got.Dir != "C:\\work" {
			t.Fatalf("Dir 应为 workdir, got %q", fr.got.Dir)
		}
		if len(fr.got.Env) != 0 {
			t.Fatalf("默认不追加环境变量, got %q", fr.got.Env)
		}
	})
}

// toRunnerCommand 映射:argv→Argv、cwd→Dir、env→追加环境变量(键序固定)。
func TestToRunnerCommandEnvSortedKeys(t *testing.T) {
	spec := command.CommandSpec{
		Argv:    []string{"go", "test"},
		Cwd:     "C:\\work",
		Env:     map[string]string{"B": "2", "A": "1"},
		Timeout: 30 * time.Second,
	}
	cmd := toRunnerCommand(spec)
	if len(cmd.Argv) != 2 || cmd.Argv[0] != "go" || cmd.Argv[1] != "test" {
		t.Fatalf("Argv 映射不符: %q", cmd.Argv)
	}
	if cmd.Dir != "C:\\work" {
		t.Fatalf("Dir 映射不符: %q", cmd.Dir)
	}
	wantEnv := []string{"A=1", "B=2"}
	if len(cmd.Env) != len(wantEnv) {
		t.Fatalf("Env = %q, 期望 %q", cmd.Env, wantEnv)
	}
	for i := range wantEnv {
		if cmd.Env[i] != wantEnv[i] {
			t.Fatalf("Env = %q, 期望 %q(键序应固定)", cmd.Env, wantEnv)
		}
	}
}
