package outcome

import (
	"errors"
	"testing"
	"time"
)

var testTime = time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)

// validEvent 返回一条通过校验的最小 outcome 事件(任务 + 候选双绑定)。
func validEvent() Event {
	return Event{
		TaskID:      "T-1",
		ChangesetID: "cs-abc",
		Kind:        KindAccepted,
		RecordedAt:  testTime,
	}
}

// AC1:kind 是封闭枚举,只有 PRD §23 的五个合法值;空串/大小写/未知值都非法。
func TestKindClosedEnum(t *testing.T) {
	valid := map[Kind]bool{
		KindAccepted:        true,
		KindReverted:        true,
		KindHotfixed:        true,
		KindRegressionFound: true,
		KindHumanCorrection: true,
		"":                  false,
		"Accepted":          false,
		"accepted ":         false,
		"shipped":           false,
		"regression":        false,
	}
	for k, want := range valid {
		if got := k.IsValid(); got != want {
			t.Errorf("Kind(%q).IsValid() = %v, want %v", string(k), got, want)
		}
	}
}

// AC1:未知 kind 的解析 fail-safe —— 返回 errors.Is 可判的 ErrUnknownKind,
// 绝不静默映射到某个已知 kind。
func TestParseKindUnknownFailsSafe(t *testing.T) {
	for _, ok := range []string{"accepted", "reverted", "hotfixed", "regression_found", "human_correction"} {
		k, err := ParseKind(ok)
		if err != nil || string(k) != ok {
			t.Errorf("ParseKind(%q) = %v, %v; want %s, nil", ok, k, err, ok)
		}
	}
	if _, err := ParseKind("merged"); err == nil {
		t.Fatal("未知 kind 被静默接受")
	} else if !errors.Is(err, ErrUnknownKind) {
		t.Errorf("错误未包装 ErrUnknownKind(errors.Is 不可判): %v", err)
	}
	if blank, err := ParseKind("  "); err == nil || blank != "" {
		t.Errorf("空白输入应报错且返回零值: %q %v", blank, err)
	}
}

// AC2:事件校验 —— kind 合法、时间显式、至少绑定 task/change 其一
// (绑定可得时必填;两个坐标同时为空等于完全无归属,写入路径拒绝)。
func TestEventValidate(t *testing.T) {
	ok := validEvent()
	if err := ok.Validate(); err != nil {
		t.Fatalf("合法事件被拒绝: %v", err)
	}

	// 上下文外事件(事故热修):只有 changeset,没有任务 —— 合法(AC2「可得时必填」)。
	hotfix := validEvent()
	hotfix.TaskID = ""
	hotfix.Kind = KindHotfixed
	if err := hotfix.Validate(); err != nil {
		t.Fatalf("仅 changeset 绑定的事件被拒绝: %v", err)
	}

	// 只有 task、没有候选:同样合法(部分绑定先落事实)。
	taskOnly := validEvent()
	taskOnly.ChangesetID = ""
	if err := taskOnly.Validate(); err != nil {
		t.Fatalf("仅 task 绑定的事件被拒绝: %v", err)
	}

	mutate := func(name string, f func(*Event)) {
		e := validEvent()
		f(&e)
		if err := e.Validate(); err == nil {
			t.Errorf("%s: 非法事件被接受", name)
		}
	}
	mutate("kind 缺失(空)", func(e *Event) { e.Kind = "" })
	mutate("kind 未知", func(e *Event) { e.Kind = "shipped" })
	mutate("recorded_at 缺失", func(e *Event) { e.RecordedAt = time.Time{} })
	mutate("task 与 changeset 双空", func(e *Event) { e.TaskID, e.ChangesetID = "", "" })
	mutate("task 是纯空白", func(e *Event) { e.TaskID = "   " })
	mutate("changeset 是纯空白", func(e *Event) { e.ChangesetID = "  " })
}
