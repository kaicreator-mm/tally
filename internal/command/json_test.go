package command

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"
)

// AC1 的 JSON 入口:shell=false 且 argv 缺省/空/null 时,Unmarshal 必须拒绝。
func TestUnmarshalRejectsEmptyArgv(t *testing.T) {
	cases := []string{
		`{"cwd":"."}`,
		`{"argv":[],"cwd":"."}`,
		`{"argv":null,"shell":false}`,
		`null`,
	}
	for i, in := range cases {
		var s CommandSpec
		if err := json.Unmarshal([]byte(in), &s); !errors.Is(err, ErrEmptyArgv) {
			t.Errorf("case %d (%s): 期望 ErrEmptyArgv, got %v", i, in, err)
		}
	}
}

// PRD §21.6 Structured Repro 的完整形态。
func TestUnmarshalFullShape(t *testing.T) {
	var s CommandSpec
	in := `{"argv":["go","test","./internal/jobs/..."],"cwd":".","env":{"GOFLAGS":"-mod=readonly"},"timeout":"120s","shell":false}`
	if err := json.Unmarshal([]byte(in), &s); err != nil {
		t.Fatal(err)
	}
	want := CommandSpec{
		Argv:    []string{"go", "test", "./internal/jobs/..."},
		Cwd:     ".",
		Env:     map[string]string{"GOFLAGS": "-mod=readonly"},
		Timeout: 120 * time.Second,
		Shell:   false,
	}
	if !reflect.DeepEqual(s, want) {
		t.Errorf("解析结果不对:\n got  %+v\n want %+v", s, want)
	}
}

// PRD §8.11 required gate 的形态:只有 argv/cwd/timeout,env/shell 缺省。
func TestUnmarshalGateShapeDefaults(t *testing.T) {
	var s CommandSpec
	if err := json.Unmarshal([]byte(`{"argv":["go","build","./..."],"cwd":".","timeout":"120s"}`), &s); err != nil {
		t.Fatal(err)
	}
	if s.Shell {
		t.Error("shell 缺省必须是 false")
	}
	if s.Env != nil {
		t.Errorf("env 缺省必须是 nil, got %v", s.Env)
	}
	if s.Timeout != 2*time.Minute {
		t.Errorf("timeout = %v, want 2m0s", s.Timeout)
	}
}

func TestUnmarshalUnknownFieldRejected(t *testing.T) {
	var s CommandSpec
	in := `{"argv":["go","build"],"argw":["typo"]}`
	if err := json.Unmarshal([]byte(in), &s); err == nil {
		t.Error("未知字段应报错(防拼写错误静默丢字段)")
	}
}

func TestUnmarshalTrailingContentRejected(t *testing.T) {
	var s CommandSpec
	in := `{"argv":["go","build"]} {"argv":["go","test"]}`
	if err := json.Unmarshal([]byte(in), &s); err == nil {
		t.Error("JSON 之后的多余内容应报错")
	}
}

func TestUnmarshalBadTimeoutRejected(t *testing.T) {
	cases := []string{
		`{"argv":["go"],"timeout":"abc"}`,
		`{"argv":["go"],"timeout":120}`,
	}
	for i, in := range cases {
		var s CommandSpec
		if err := json.Unmarshal([]byte(in), &s); err == nil {
			t.Errorf("case %d (%s): 非法 timeout 应报错", i, in)
		}
	}
}

func TestMarshalCanonicalForm(t *testing.T) {
	s := CommandSpec{
		Argv:    []string{"go", "test", "./..."},
		Cwd:     ".",
		Env:     map[string]string{"B": "2", "A": "1"}, // map 序不影响输出
		Timeout: 120 * time.Second,
	}
	got, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"argv":["go","test","./..."],"cwd":".","env":{"A":"1","B":"2"},"timeout":"2m0s","shell":false}`
	if string(got) != want {
		t.Errorf("规范输出不对:\n got  %s\n want %s", got, want)
	}
}

// 零值语义:env nil 输出 {},timeout 0 省略,shell false 恒显式输出。
func TestMarshalMinimalSpec(t *testing.T) {
	got, err := json.Marshal(CommandSpec{Argv: []string{"go", "build"}})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"argv":["go","build"],"cwd":"","env":{},"shell":false}`
	if string(got) != want {
		t.Errorf("最小输出不对:\n got  %s\n want %s", got, want)
	}
}

// AC3: JSON round-trip 稳定 —— 值相等且再次 Marshal 字节一致。
func TestRoundTripStable(t *testing.T) {
	specs := []CommandSpec{
		{Argv: []string{"go", "build", "./..."}},
		{Argv: []string{"go", "test", "./internal/jobs/..."}, Cwd: ".", Timeout: 90 * time.Second},
		{Argv: []string{"make", "check"}, Env: map[string]string{"CI": "1"}, Timeout: time.Hour},
		{Argv: []string{"sh", "-c", "echo hi"}, Shell: true},
	}
	for i, s := range specs {
		b1, err := json.Marshal(s)
		if err != nil {
			t.Fatalf("case %d: Marshal: %v", i, err)
		}
		var back CommandSpec
		if err := json.Unmarshal(b1, &back); err != nil {
			t.Fatalf("case %d: Unmarshal(%s): %v", i, b1, err)
		}
		if !reflect.DeepEqual(s, back) {
			t.Errorf("case %d: round-trip 值不相等:\n orig %+v\n back %+v", i, s, back)
		}
		b2, err := json.Marshal(back)
		if err != nil {
			t.Fatalf("case %d: re-Marshal: %v", i, err)
		}
		if string(b1) != string(b2) {
			t.Errorf("case %d: 二次 Marshal 字节不稳定:\n %s\n %s", i, b1, b2)
		}
	}
}

// env 空对象与缺省等价:Unmarshal 归一化为 nil,保证 round-trip 稳定。
func TestUnmarshalNormalizesEmptyEnv(t *testing.T) {
	var s CommandSpec
	if err := json.Unmarshal([]byte(`{"argv":["go"],"env":{},"shell":false}`), &s); err != nil {
		t.Fatal(err)
	}
	if s.Env != nil {
		t.Errorf("空 env 应归一化为 nil, got %v", s.Env)
	}
}
