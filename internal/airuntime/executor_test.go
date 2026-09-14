package airuntime

import (
	"context"
	"errors"
	"testing"
)

func TestValidatedRejectsInvalidBeforeTransport(t *testing.T) {
	fake := &FakeExecutor{}
	exec := Validated(fake)

	req := validRequest()
	req.Intent.Role = "" // 未声明意图
	if _, err := exec.Execute(context.Background(), req); !errors.Is(err, ErrNoIntent) {
		t.Fatalf("want %v, got %v", ErrNoIntent, err)
	}
	if n := len(fake.Requests()); n != 0 {
		t.Fatalf("非法请求不应到达传输层,实际收到 %d 个", n)
	}
}

func TestValidatedDelegatesValidRequests(t *testing.T) {
	want := Result{
		OperationID:       "op-1",
		Status:            StatusSucceeded,
		TerminationReason: TerminationCompleted,
		Output:            []byte(`{"answer":"staging"}`),
		ProductionCostUSD: 0.01,
		Strategy:          StrategyBaseline,
	}
	fake := &FakeExecutor{Result: want}
	exec := Validated(fake)

	req := validRequest()
	got, err := exec.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("合法请求被拒: %v", err)
	}
	if got.OperationID != want.OperationID || got.Status != want.Status ||
		got.TerminationReason != want.TerminationReason || got.Strategy != want.Strategy {
		t.Fatalf("Result 透传漂移: got %+v", got)
	}
	if string(got.Output) != string(want.Output) {
		t.Fatalf("Output 漂移: got %s", got.Output)
	}
	if reqs := fake.Requests(); len(reqs) != 1 || reqs[0].IdempotencyKey != req.IdempotencyKey {
		t.Fatalf("传输层收到的请求不符: %+v", reqs)
	}
}

func TestValidatedNilExecutorFailsLoud(t *testing.T) {
	exec := Validated(nil)
	if _, err := exec.Execute(context.Background(), validRequest()); err == nil {
		t.Fatal("包装 nil Executor 必须报错,而不是静默")
	}
}

func TestFakeExecutorPresetErrAndResponder(t *testing.T) {
	// 预设传输错误优先于结果。
	fake := &FakeExecutor{Err: errors.New("dial tcp: refused")}
	if _, err := fake.Execute(context.Background(), validRequest()); err == nil {
		t.Fatal("预设 Err 未生效")
	}

	// 动态响应覆盖预设。
	calls := 0
	fake2 := &FakeExecutor{
		Result: Result{Status: StatusFailed},
		Responder: func(_ context.Context, _ Request) (Result, error) {
			calls++
			return Result{Status: StatusUncertain}, nil
		},
	}
	got, err := fake2.Execute(context.Background(), validRequest())
	if err != nil || got.Status != StatusUncertain {
		t.Fatalf("Responder 未生效: got %+v, err %v", got, err)
	}
	if calls != 1 || len(fake2.Requests()) != 1 {
		t.Fatalf("记录不符: calls=%d requests=%d", calls, len(fake2.Requests()))
	}
}

func TestFakeExecutorRecordsConcurrently(t *testing.T) {
	fake := &FakeExecutor{}
	const n = 8
	done := make(chan struct{})
	for i := 0; i < n; i++ {
		go func() {
			_, _ = fake.Execute(context.Background(), validRequest())
			done <- struct{}{}
		}()
	}
	for i := 0; i < n; i++ {
		<-done
	}
	if got := len(fake.Requests()); got != n {
		t.Fatalf("并发记录丢失: got %d, want %d", got, n)
	}
}
