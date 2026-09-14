package closedloop

// store_lineage_test.go 固化 runstore 持久化面与闭环出口的适配(P2-05;
// 原钉子位于 runstore 内部测试,因 P2-08 exposure_store 反向导入 runstore
// 形成测试环而迁到消费方一侧 —— 本包生产代码已单向导入 runstore,无环)。

import (
	"errors"
	"testing"

	"tally/internal/runstore"
)

// 编译期固化:*runstore.Store 原生满足取代事实出口的消费面。组合根无需任何
// 桥接代码,直接 NewStoreLineageSink(st) 即完成 append-only 取代事实出口的
// 持久化接线。
var _ SupersededFactAppender = (*runstore.Store)(nil)

// TestStoreLineageSinkWiring 构造 LineageSink 持久化适配并验证 fail-safe:
// nil 存储拒绝构造;合法存储产出一个只追加的出口(类型上没有更新/删除方法,
// 与 runstore 侧 TestStoreAPIHasNoFactMutators 的 API 面固化互为防线)。
func TestStoreLineageSinkWiring(t *testing.T) {
	sink, err := NewStoreLineageSink(runstore.New(nil))
	if err != nil {
		t.Fatalf("NewStoreLineageSink(*runstore.Store): %v", err)
	}
	if sink == nil {
		t.Fatal("适配器不应为 nil")
	}
	if _, err := NewStoreLineageSink(nil); !errors.Is(err, ErrInvalidLineageSink) {
		t.Fatalf("nil 存储应拒绝构造(ErrInvalidLineageSink),得到 %v", err)
	}
}
