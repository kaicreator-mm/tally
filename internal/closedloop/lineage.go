// lineage.go 实现 P2-05 的 LineageSink 持久化适配:把持有取代事实追加
// 入口的存储适配为闭环的 append-only 取代事实出口(ports.go 的 LineageSink)。
//
// 生产接线零桥接:internal/runstore 的 *Store 原生具备
// `AppendSupersededFact(ctx, changeset.SupersededFact) error` 形态
// (append-only:只 INSERT,重复事实幂等忽略,UPDATE/DELETE 由数据库
// 触发器拦截),因此 *Store 结构性满足本文件定义的消费面,组合根直接
// `NewStoreLineageSink(store)` 即完成闭环取代事实出口的持久化接线。
//
// 纪律(ADR-0001 §2;INV-V14-06/16):
//   - 只追加:适配器没有任何更新/删除/回滚方法,存储侧亦无;
//   - 透传不重写:事实原样交给存储,适配器不改写身份、时刻或原因;
//   - 时序归 Engine:修复候选的取代事实在复验之前落库(Engine 裁量点),
//     适配器不引入第二条写入时机;
//   - fail-safe:存储缺席时拒绝构造,绝不产出静默丢弃事实的出口。
package closedloop

import (
	"context"
	"errors"

	"tally/internal/changeset"
)

// ErrInvalidLineageSink:持久化出口构造依赖缺失(存储为 nil)。
// errors.Is 判因,不做字符串匹配。
var ErrInvalidLineageSink = errors.New("closedloop: LineageSink 持久化适配缺少取代事实存储")

// SupersededFactAppender 是本适配消费的最小持久化面(小接口由消费方定义):
// 追加一条取代事实。这是存储需要暴露的全部 —— 取代关系只增不改,持久化
// 面上不存在任何其他写入口。
type SupersededFactAppender interface {
	AppendSupersededFact(ctx context.Context, f changeset.SupersededFact) error
}

// NewStoreLineageSink 把取代事实存储适配为闭环 LineageSink(P2-05:取代
// 谱系的持久化出口)。存储缺席时拒绝构造(ErrInvalidLineageSink):没有
// 存储的出口会静默丢弃谱系事实,非法。
func NewStoreLineageSink(store SupersededFactAppender) (LineageSink, error) {
	if store == nil {
		return nil, ErrInvalidLineageSink
	}
	return storeLineageSink{store: store}, nil
}

// storeLineageSink 是 LineageSink 的持久化适配:唯一职责是把追加调用
// 原样透传给存储。没有更新/删除方法;存储错误向上返回,由 Engine 裁量为
// supersession_unavailable 升级(谱系断裂的修复不成立),绝不静默吞掉。
type storeLineageSink struct {
	store SupersededFactAppender
}

// RecordSuperseded 实现 LineageSink:事实原样透传(append-only,无改写)。
func (s storeLineageSink) RecordSuperseded(ctx context.Context, fact changeset.SupersededFact) error {
	return s.store.AppendSupersededFact(ctx, fact)
}
