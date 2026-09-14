package command

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
)

// wireSpec 是 CommandSpec 的 JSON 线上形态,字段名与 PRD §21.6/§8.11 一致:
//
//	{"argv":["go","test","./..."],"cwd":".","env":{},"timeout":"120s","shell":false}
//
// timeout 用 duration 字符串(如 "120s"),不用纳秒整数;shell 恒显式输出,
// 缺省即 false,保证 shell 语义在 JSON 里始终可见。
type wireSpec struct {
	Argv    []string          `json:"argv"`
	Cwd     string            `json:"cwd"`
	Env     map[string]string `json:"env"`
	Timeout string            `json:"timeout,omitempty"`
	Shell   bool              `json:"shell"`
}

// MarshalJSON 输出 CommandSpec 的规范 JSON,对同一值字节稳定:
// env 恒为 JSON 对象(nil 输出 {}),timeout 为 0 时省略,shell 恒输出。
func (s CommandSpec) MarshalJSON() ([]byte, error) {
	w := wireSpec{Argv: s.Argv, Cwd: s.Cwd, Env: s.Env, Shell: s.Shell}
	if w.Env == nil {
		w.Env = map[string]string{}
	}
	if s.Timeout != 0 {
		w.Timeout = s.Timeout.String()
	}
	return json.Marshal(w)
}

// UnmarshalJSON 解析规范 JSON 形态。缺省字段取零值(shell 缺省即 false);
// argv/env 为空时归一化为 nil;解析成功后再 Validate——非法 spec 无法从 JSON 构造。
func (s *CommandSpec) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var w wireSpec
	if err := dec.Decode(&w); err != nil {
		return fmt.Errorf("command: 解析 CommandSpec JSON: %w", err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("command: CommandSpec JSON 之后有多余内容")
	}

	spec := CommandSpec{Cwd: w.Cwd, Shell: w.Shell}
	if len(w.Argv) > 0 {
		spec.Argv = w.Argv
	}
	if len(w.Env) > 0 {
		spec.Env = w.Env
	}
	if w.Timeout != "" {
		d, err := time.ParseDuration(w.Timeout)
		if err != nil {
			return fmt.Errorf("command: timeout %q 不是合法 duration: %w", w.Timeout, err)
		}
		spec.Timeout = d
	}
	if err := spec.Validate(); err != nil {
		return err
	}
	*s = spec
	return nil
}
