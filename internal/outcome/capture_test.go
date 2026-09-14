package outcome

// capture.go(P6-02)显式捕获 API 测试。
//
// 两层(与 store_test.go 同一惯例):
//   - 纯单元层:事实校验的负路径、捕获面不暴露 accepted 写入方(不触库),
//     无 DB 全绿;
//   - 数据库层:四类事件经捕获入口的写入/读回往返、重放幂等、不同发生追加、
//     非法事实零落行。数据库不可达时 SKIP,不产出假失败。

import (
	"context"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---- 纯单元层(无 DB) ----

// 事实校验的负路径(fail-safe):完全无归属、纯空白坐标、缺发生时刻一律
// 拒绝,绝不静默收下不可审计的捕获事实;合法的部分绑定(task-only /
// changeset-only)放行(§23:采集点先于归档完善)。
func TestCaptureFactValidate(t *testing.T) {
	ok := []interface{ Validate() error }{
		RevertFact{TaskID: "T-1", ChangesetID: "cs-1", RecordedAt: testTime},
		RevertFact{ChangesetID: "cs-1", RecordedAt: testTime},
		HotfixFact{ChangesetID: "cs-1", RecordedAt: testTime}, // 带外热修:仅候选绑定
		HotfixFact{TaskID: "T-1", ChangesetID: "cs-1", RecordedAt: testTime},
		RegressionFact{TaskID: "T-1", ChangesetID: "cs-1", RecordedAt: testTime},
		RegressionFact{TaskID: "T-1", RecordedAt: testTime},
		HumanCorrectionFact{TaskID: "T-1", ChangesetID: "cs-1", RecordedAt: testTime},
	}
	for i, f := range ok {
		if err := f.Validate(); err != nil {
			t.Errorf("ok[%d] 合法捕获事实被拒绝: %v", i, err)
		}
	}

	bad := []struct {
		name string
		f    interface{ Validate() error }
	}{
		{"revert 双空绑定", RevertFact{RecordedAt: testTime}},
		{"revert task 纯空白", RevertFact{TaskID: "   ", ChangesetID: "cs-1", RecordedAt: testTime}},
		{"revert 缺发生时刻", RevertFact{TaskID: "T-1", ChangesetID: "cs-1"}},
		{"hotfix 双空绑定", HotfixFact{RecordedAt: testTime}},
		{"hotfix changeset 纯空白", HotfixFact{ChangesetID: "  ", RecordedAt: testTime}},
		{"hotfix 缺发生时刻", HotfixFact{TaskID: "T-1", ChangesetID: "cs-1"}},
		{"regression 双空绑定", RegressionFact{RecordedAt: testTime}},
		{"regression 缺发生时刻", RegressionFact{TaskID: "T-1", ChangesetID: "cs-1"}},
		{"human_correction 双空绑定", HumanCorrectionFact{RecordedAt: testTime}},
		{"human_correction 缺发生时刻", HumanCorrectionFact{TaskID: "T-1", ChangesetID: "cs-1"}},
	}
	for _, tc := range bad {
		if err := tc.f.Validate(); err == nil {
			t.Errorf("%s: 非法捕获事实被接受", tc.name)
		}
	}
}

// AC「每个受支持事件有被记录的权威生产者」的 API 面钉住:四类既有事件各有
// 一个专用捕获入口;accepted 刻意不暴露 —— 它的唯一写入方是闭环 accept 缝
// (P6-01),本面绝不提供第二个验收落账权威。
func TestCaptureAPINeverExposesAccepted(t *testing.T) {
	want := map[string]bool{
		"RecordReverted":        true,
		"RecordHotfixed":        true,
		"RecordRegressionFound": true,
		"RecordHumanCorrection": true,
	}
	typ := reflect.TypeOf(&Store{})
	got := map[string]bool{}
	for i := 0; i < typ.NumMethod(); i++ {
		name := typ.Method(i).Name
		got[name] = true
		if strings.Contains(name, "Accepted") {
			t.Errorf("捕获面出现 accepted 写入方 %s:accepted 的唯一权威是闭环 accept 缝(P6-01)", name)
		}
	}
	for name := range want {
		if !got[name] {
			t.Errorf("捕获面缺少 %s(四类既有事件必须各有显式捕获入口)", name)
		}
	}
}

// ---- 数据库层(不可达即 SKIP) ----

// 四类事件经捕获入口写入/读回往返:kind 由入口固定,坐标与发生时刻一次落成;
// reverted/hotfixed/regression_found 归 guardrail/rework 读侧投影,human_
// correction 归介入投影(§31 词表不变,DL-12 五类封闭枚举不扩)。
func TestCaptureRoundtripEachKind(t *testing.T) {
	st, pool := testOutcomeStore(t)
	ctx := context.Background()

	if err := st.RecordReverted(ctx, RevertFact{TaskID: "T-1", ChangesetID: "cs-1", RecordedAt: testTime}); err != nil {
		t.Fatalf("捕获 reverted: %v", err)
	}
	if err := st.RecordHotfixed(ctx, HotfixFact{ChangesetID: "cs-hotfix", RecordedAt: testTime.Add(time.Minute)}); err != nil {
		t.Fatalf("捕获 hotfixed(带外仅候选绑定): %v", err)
	}
	if err := st.RecordRegressionFound(ctx, RegressionFact{TaskID: "T-1", ChangesetID: "cs-1", RecordedAt: testTime.Add(2 * time.Minute)}); err != nil {
		t.Fatalf("捕获 regression_found: %v", err)
	}
	if err := st.RecordHumanCorrection(ctx, HumanCorrectionFact{TaskID: "T-1", ChangesetID: "cs-2", RecordedAt: testTime.Add(3 * time.Minute)}); err != nil {
		t.Fatalf("捕获 human_correction: %v", err)
	}

	events, err := st.ForTask(ctx, "T-1")
	if err != nil {
		t.Fatalf("读回任务事件: %v", err)
	}
	wantKinds := []Kind{KindReverted, KindRegressionFound, KindHumanCorrection}
	if len(events) != len(wantKinds) {
		t.Fatalf("T-1 应有 %d 行,得到 %d: %+v", len(wantKinds), len(events), events)
	}
	for i, e := range events {
		if e.Kind != wantKinds[i] {
			t.Errorf("events[%d].Kind = %s, want %s(旧→新)", i, e.Kind, wantKinds[i])
		}
		if e.TaskID != "T-1" {
			t.Errorf("events[%d].TaskID = %q, want T-1", i, e.TaskID)
		}
	}
	if events[0].ChangesetID != "cs-1" || !events[0].RecordedAt.Equal(testTime) {
		t.Errorf("reverted 往返不一致: %+v", events[0])
	}

	hotfixes, err := st.ForChangeset(ctx, "cs-hotfix")
	if err != nil || len(hotfixes) != 1 {
		t.Fatalf("读回热修事件: %v %+v", err, hotfixes)
	}
	if hotfixes[0].Kind != KindHotfixed || hotfixes[0].TaskID != "" || hotfixes[0].ChangesetID != "cs-hotfix" {
		t.Errorf("hotfixed 往返不一致: %+v", hotfixes[0])
	}

	if n := countOutcomeRows(t, pool); n != 4 {
		t.Fatalf("四类捕获应恰落 4 行,得到 %d", n)
	}
}

// 重放幂等:同一捕获事实(坐标+发生时刻完全相同)重复投递只落一行 —— 重试/
// 重放不产生重复语义事件(P6-01 幂等口径;时刻经时区换算后仍匹配)。
func TestCaptureReplayIsIdempotent(t *testing.T) {
	st, pool := testOutcomeStore(t)
	ctx := context.Background()

	local := testTime.In(time.FixedZone("UTC+8", 8*3600))
	f := RevertFact{TaskID: "T-2", ChangesetID: "cs-replay", RecordedAt: local}
	if err := st.RecordReverted(ctx, f); err != nil {
		t.Fatalf("首次捕获 reverted: %v", err)
	}
	// 同一事实的另一种时区表达:同一时刻,必须幂等而不是追加。
	if err := st.RecordReverted(ctx, RevertFact{TaskID: "T-2", ChangesetID: "cs-replay", RecordedAt: testTime}); err != nil {
		t.Fatalf("重放捕获 reverted: %v", err)
	}
	if n := countOutcomeRows(t, pool); n != 1 {
		t.Fatalf("同一事实重放应幂等为 1 行,得到 %d", n)
	}

	events, err := st.ForTask(ctx, "T-2")
	if err != nil || len(events) != 1 || !events[0].RecordedAt.Equal(testTime) {
		t.Fatalf("幂等后事件应恰一行且时刻不变: %v %+v", err, events)
	}
}

// 并发重放幂等:多个 writer 同时捕获同一事实时只能有一个数据库 winner。
// 这是 0025 唯一约束 + ON CONFLICT 原子路径的回归测试;旧的先查再写实现会
// 在并发窗口内偶发重复。
func TestCaptureConcurrentReplayIsIdempotent(t *testing.T) {
	st, pool := testOutcomeStore(t)
	ctx := context.Background()
	f := RevertFact{TaskID: "T-concurrent", ChangesetID: "cs-concurrent", RecordedAt: testTime}

	const writers = 12
	start := make(chan struct{})
	errCh := make(chan error, writers)
	var wg sync.WaitGroup
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errCh <- st.RecordReverted(ctx, f)
		}()
	}
	close(start)
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("并发重放应全部按幂等成功收敛: %v", err)
		}
	}
	if n := countOutcomeRows(t, pool); n != 1 {
		t.Fatalf("%d 个并发 writer 捕获同一事实后应恰有 1 行,得到 %d", writers, n)
	}
}

// 不同发生必追加:同一坐标、不同发生时刻 = 另一次真实发生(同一候选的多次
// 回归是多行事实,不是可变状态);事件面没有载荷区分实例,时刻就是实例坐标。
func TestCaptureDistinctOccurrenceAppends(t *testing.T) {
	st, pool := testOutcomeStore(t)
	ctx := context.Background()

	first := RegressionFact{TaskID: "T-3", ChangesetID: "cs-reg", RecordedAt: testTime}
	second := RegressionFact{TaskID: "T-3", ChangesetID: "cs-reg", RecordedAt: testTime.Add(time.Hour)}
	if err := st.RecordRegressionFound(ctx, first); err != nil {
		t.Fatalf("首次捕获 regression_found: %v", err)
	}
	if err := st.RecordRegressionFound(ctx, second); err != nil {
		t.Fatalf("第二次捕获 regression_found: %v", err)
	}
	if n := countOutcomeRows(t, pool); n != 2 {
		t.Fatalf("两次不同发生应落 2 行,得到 %d", n)
	}

	events, err := st.ForTask(ctx, "T-3")
	if err != nil || len(events) != 2 {
		t.Fatalf("读回两次回归: %v %+v", err, events)
	}
	if !events[0].RecordedAt.Before(events[1].RecordedAt) {
		t.Errorf("事件应按发生时刻定序: %+v", events)
	}
}

// 负路径/失败路径:非法事实在校验层拒绝、零落行(不触库);捕获 API 绝不把
// 非法输入静默折成空成功 —— 调用方拿到的是显式错误,自行裁决(事实未落成
// 对生产者可见,绝不伪装成已捕获)。
func TestCaptureRejectsInvalidFactWithoutRows(t *testing.T) {
	st, pool := testOutcomeStore(t)
	ctx := context.Background()

	cases := []struct {
		name string
		cap  func() error
	}{
		{"reverted 无绑定", func() error { return st.RecordReverted(ctx, RevertFact{RecordedAt: testTime}) }},
		{"reverted 缺时刻", func() error {
			return st.RecordReverted(ctx, RevertFact{TaskID: "T-4", ChangesetID: "cs-4"})
		}},
		{"hotfixed 无绑定", func() error { return st.RecordHotfixed(ctx, HotfixFact{RecordedAt: testTime}) }},
		{"regression_found 纯空白坐标", func() error {
			return st.RecordRegressionFound(ctx, RegressionFact{TaskID: "  ", ChangesetID: "cs-4", RecordedAt: testTime})
		}},
		{"human_correction 无绑定", func() error {
			return st.RecordHumanCorrection(ctx, HumanCorrectionFact{RecordedAt: testTime})
		}},
	}
	for _, tc := range cases {
		if err := tc.cap(); err == nil {
			t.Errorf("%s: 非法捕获事实被接受", tc.name)
		}
	}
	if n := countOutcomeRows(t, pool); n != 0 {
		t.Fatalf("非法事实不得落行,得到 %d 行", n)
	}
}
