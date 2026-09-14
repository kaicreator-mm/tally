package closedloop

// lineage.go LineageSink 持久化适配测试(P2-05):append-only 透传、
// fail-safe 构造、错误原样上抛。存储侧行为(*runstore.Store 的
// AppendSupersededFact 与数据库触发器防线)由 internal/runstore 的测试
// 守护 —— 本文件只用 fake 验证适配器本身。

import (
	"context"
	"errors"
	"testing"
	"time"

	"tally/internal/changeset"
)

type fakeFactStore struct {
	got []changeset.SupersededFact
	err error
}

func (f *fakeFactStore) AppendSupersededFact(ctx context.Context, fact changeset.SupersededFact) error {
	if f.err != nil {
		return f.err
	}
	f.got = append(f.got, fact)
	return nil
}

// nil 存储拒绝构造:没有存储的出口会静默丢弃谱系事实,非法(fail-safe)。
func TestNewStoreLineageSinkRejectsNil(t *testing.T) {
	if _, err := NewStoreLineageSink(nil); !errors.Is(err, ErrInvalidLineageSink) {
		t.Fatalf("nil 存储应返回 ErrInvalidLineageSink,得到 %v", err)
	}
}

// 适配器把取代事实原样透传给存储:身份、原因、时刻一律不改写
// (append-only,无第二条改写路径)。
func TestStoreLineageSinkAppendsVerbatim(t *testing.T) {
	store := &fakeFactStore{}
	sink, err := NewStoreLineageSink(store)
	if err != nil {
		t.Fatalf("NewStoreLineageSink: %v", err)
	}
	at := time.Date(2026, 9, 9, 8, 0, 0, 0, time.UTC)
	fact, err := changeset.NewSupersededFact("sha256:old", "sha256:new", changeset.SupersedeRework, at)
	if err != nil {
		t.Fatalf("NewSupersededFact: %v", err)
	}
	if err := sink.RecordSuperseded(context.Background(), fact); err != nil {
		t.Fatalf("RecordSuperseded: %v", err)
	}
	if len(store.got) != 1 || store.got[0] != fact {
		t.Fatalf("存储收到 %+v, 期望原样一条 %+v", store.got, fact)
	}
}

// 存储错误原样上抛:Engine 据此落 supersession_unavailable 升级,适配器
// 绝不吞错或伪装(errors.Is 判因)。
func TestStoreLineageSinkPropagatesStoreError(t *testing.T) {
	boom := errors.New("store down")
	sink, err := NewStoreLineageSink(&fakeFactStore{err: boom})
	if err != nil {
		t.Fatalf("NewStoreLineageSink: %v", err)
	}
	fact, err := changeset.NewSupersededFact("sha256:old", "sha256:new", changeset.SupersedeRework, time.Now())
	if err != nil {
		t.Fatalf("NewSupersededFact: %v", err)
	}
	if err := sink.RecordSuperseded(context.Background(), fact); !errors.Is(err, boom) {
		t.Fatalf("存储错误应原样上抛,得到 %v", err)
	}
}
