//go:build integration

package jobs

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"tally/internal/store"
)

// TestQueueEnqueueAndProcess 验证入队 → worker 取出 → 执行 → 标记完成 的完整回路。
// 需要真 Postgres(每测试 store.TestDB 拿独立 schema,docs/05-test-standards.md)。
func TestQueueEnqueueAndProcess(t *testing.T) {
	ctx := context.Background()
	st := store.TestDB(t)

	done := make(chan struct{})
	q := New(st, map[Kind]Handler{
		"itest": func(ctx context.Context, payload json.RawMessage) error {
			var got map[string]string
			if err := json.Unmarshal(payload, &got); err != nil {
				return err
			}
			if got["hello"] != "world" {
				t.Errorf("payload 不对: %v", got)
			}
			close(done)
			return nil
		},
	}, WithConcurrency(1))

	id, err := q.Enqueue(ctx, "itest", map[string]string{"hello": "world"})
	if err != nil {
		t.Fatalf("入队: %v", err)
	}
	if id == 0 {
		t.Fatal("job id 应非零")
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() { _ = q.Run(runCtx) }()

	select {
	case <-done:
		cancel()
	case <-time.After(10 * time.Second):
		cancel()
		t.Fatal("任务超时未被处理")
	}
}
