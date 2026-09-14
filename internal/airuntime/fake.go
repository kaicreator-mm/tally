package airuntime

import (
	"context"
	"sync"
)

// FakeExecutor 是确定性的 Executor 测试替身(P0-10/P0-11 消费方迁移测试用):
// 记录收到的每个请求,返回预设结果/错误;不做任何传输,不做任何判定。
// 预设 Err 模拟传输层失败(优先于 Result);Responder 非 nil 时改用其动态响应
// (例如按请求顺序返回不同结果)。它自己不做 Validate —— 要测域侧闸就用
// Validated(&FakeExecutor{...}) 组合。
type FakeExecutor struct {
	mu sync.Mutex

	// requests 是收到的请求记录(Requests 返回其拷贝)。
	requests []Request

	// Result 是默认返回的预设结果(Responder 非 nil 时忽略)。
	Result Result
	// Err 是预设的传输层错误(优先于 Result)。
	Err error
	// Responder 是可选的动态响应函数;nil 表示用 Result/Err。
	Responder func(ctx context.Context, req Request) (Result, error)
}

var _ Executor = (*FakeExecutor)(nil)

// Execute 记录请求并返回预设响应。
func (f *FakeExecutor) Execute(ctx context.Context, req Request) (Result, error) {
	f.mu.Lock()
	f.requests = append(f.requests, req)
	responder := f.Responder
	result, err := f.Result, f.Err
	f.mu.Unlock()

	if responder != nil {
		return responder(ctx, req)
	}
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

// Requests 返回至今收到的全部请求(拷贝)。
func (f *FakeExecutor) Requests() []Request {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]Request, len(f.requests))
	copy(out, f.requests)
	return out
}
