package airuntime

import (
	"context"
	"errors"
)

// Validated 返回一个先强制 Request.Validate() 再委托 next 的 Executor:
// 意图闸、幂等键、预算、输入与脱敏闸不依赖每个传输实现方自觉。
// composition root 应该用 Validated 包装真实传输实现:
//
//	exec := airuntime.Validated(realTransport)
//
// 域侧机械闸由此收敛在 Tally(AC2),传输实现方只负责搬运。
func Validated(next Executor) Executor {
	return validatedExecutor{next: next}
}

type validatedExecutor struct {
	next Executor
}

// errNilExecutor 表示 Validated 包装了 nil Executor:这是装配错误,
// 在调用时报出来,而不是静默吞掉每次执行。
var errNilExecutor = errors.New("airuntime: Validated 包装了 nil Executor")

func (v validatedExecutor) Execute(ctx context.Context, req Request) (Result, error) {
	if v.next == nil {
		return Result{}, errNilExecutor
	}
	if err := req.Validate(); err != nil {
		return Result{}, err
	}
	return v.next.Execute(ctx, req)
}
