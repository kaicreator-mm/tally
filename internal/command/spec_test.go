package command

import (
	"errors"
	"testing"
	"time"
)

// AC2: shell 语义显式且默认拒绝 —— 零值 Shell 必须是 false。
func TestZeroValueShellDefaultsFalse(t *testing.T) {
	var s CommandSpec
	if s.Shell {
		t.Error("零值 CommandSpec.Shell 必须是 false")
	}
}

// AC1: shell=false 时拒绝空 argv。零值、显式空切片都算空。
func TestValidateRejectsEmptyArgvWhenShellFalse(t *testing.T) {
	cases := []CommandSpec{
		{},
		{Argv: []string{}},
		{Argv: nil, Cwd: ".", Timeout: time.Minute},
	}
	for i, s := range cases {
		if err := s.Validate(); !errors.Is(err, ErrEmptyArgv) {
			t.Errorf("case %d: 期望 ErrEmptyArgv, got %v", i, err)
		}
	}
}

func TestValidateAcceptsNonEmptyArgv(t *testing.T) {
	s := CommandSpec{Argv: []string{"go", "test", "./internal/jobs/..."}}
	if err := s.Validate(); err != nil {
		t.Errorf("非空 argv 不该报错: %v", err)
	}
}

// shell=true 是显式 opt-in;AC 只约束 shell=false 的路径,这里锁住边界不被悄悄加严。
func TestValidateExplicitShellEmptyArgvAllowed(t *testing.T) {
	if err := (CommandSpec{Shell: true}).Validate(); err != nil {
		t.Errorf("shell=true 时不按空 argv 拒绝: %v", err)
	}
}
