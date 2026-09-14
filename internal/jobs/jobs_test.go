package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"tally/internal/store"
)

// —— 内存查询面(按 migrations/0014_job_lease.sql 与 queries/job.sql 的谓词建模)——
//
// 离线可测 Queue 的认领/心跳/终态语义,不依赖真库。建模口径与 SQL 严格一致:
//   - 认领只取 pending,或 stale 的 running(running 且 lease_until 为空/已过期),
//     认领即换主:写 owner、累加 attempts、按入参 interval 续新租约窗口;
//   - 心跳只在「行仍归该 worker、仍 running、租约仍存活(lease_until > now)」时续租,
//     否则命中 0 行(fencing);
//   - 终态写入按 SQL 原样只按 id 覆盖(MarkJobDone/MarkJobFailed 无 owner 谓词)
//     —— 正因如此,Queue 侧必须先做租约闸门再落终态,由 lease_test.go 验证。
type fakeQueries struct {
	mu  sync.Mutex
	now func() time.Time // 可控时钟,默认 time.Now

	jobs []*fakeJob
	seq  int64

	// 调用记录(断言用)
	claims     []store.ClaimNextJobParams
	heartbeats []store.HeartbeatJobParams
	doneIDs    []int64
	failed     []store.MarkJobFailedParams
	inserts    []store.InsertJobParams

	// 故障注入:claimErr 让认领报错;hbScript 非空时心跳逐次取用其元素作为
	// RowsAffected 返回值(取尽后回真实谓词),0 即模拟租约丢失;hbErr 让心跳报错。
	claimErr error
	hbErr    error
	hbScript []int64
}

// fakeJob 是 job 行的内存映像;时间字段零值 = SQL NULL。
type fakeJob struct {
	id         int64
	kind       string
	payload    []byte
	status     string
	attempts   int32
	workerID   pgtype.Text
	leaseUntil time.Time // 零值 = NULL
	heartbeat  time.Time
	createdAt  time.Time
	lastError  pgtype.Text
}

const (
	statusPending = "pending"
	statusRunning = "running"
	statusDone    = "done"
	statusFailed  = "failed"
)

var _ querier = (*fakeQueries)(nil)

func newFakeQueries() *fakeQueries {
	return &fakeQueries{now: func() time.Time { return time.Now() }}
}

// seedPending 入一个 pending 任务,返回 job id。
func (f *fakeQueries) seedPending(kind string, payload []byte) int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	f.jobs = append(f.jobs, &fakeJob{id: f.seq, kind: kind, payload: payload, status: statusPending, createdAt: f.now()})
	return f.seq
}

// seedRunning 放一个已被某 worker 认领的 running 行(模拟他机执行中 / worker 崩溃遗留)。
// leaseUntil 传零值 = 迁移前 lease_until 为 NULL 的存量卡死行。
func (f *fakeQueries) seedRunning(kind, worker string, leaseUntil time.Time, attempts int32) int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	f.jobs = append(f.jobs, &fakeJob{
		id:         f.seq,
		kind:       kind,
		status:     statusRunning,
		attempts:   attempts,
		workerID:   pgtype.Text{String: worker, Valid: worker != ""},
		leaseUntil: leaseUntil,
		createdAt:  f.now(),
	})
	return f.seq
}

// job 返回行快照(断言用)。
func (f *fakeQueries) job(id int64) fakeJob {
	f.mu.Lock()
	defer f.mu.Unlock()
	j := f.byIDLocked(id)
	if j == nil {
		return fakeJob{}
	}
	return *j
}

func (f *fakeQueries) byIDLocked(id int64) *fakeJob {
	for _, j := range f.jobs {
		if j.id == id {
			return j
		}
	}
	return nil
}

// renewLocked 是 HeartbeatJob 的真实谓词(调用方已持锁):仅当行仍归该 worker、
// 仍 running 且租约仍存活时续租,返回受影响行数。过期后不允许复活。
func (f *fakeQueries) renewLocked(arg store.HeartbeatJobParams) int64 {
	now := f.now()
	j := f.byIDLocked(arg.ID)
	if j == nil || j.status != statusRunning || j.workerID != arg.WorkerID {
		return 0
	}
	if j.leaseUntil.IsZero() || !j.leaseUntil.After(now) {
		return 0
	}
	j.heartbeat = now
	j.leaseUntil = now.Add(intervalDur(arg.Column3))
	return 1
}

func (f *fakeQueries) ClaimNextJob(_ context.Context, arg store.ClaimNextJobParams) (store.ClaimNextJobRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.claims = append(f.claims, arg)
	if f.claimErr != nil {
		return store.ClaimNextJobRow{}, f.claimErr
	}
	now := f.now()
	for _, j := range f.jobs {
		stale := j.leaseUntil.IsZero() || !j.leaseUntil.After(now)
		if j.status != statusPending && !(j.status == statusRunning && stale) {
			continue
		}
		j.status = statusRunning
		j.attempts++
		j.workerID = arg.WorkerID
		j.leaseUntil = now.Add(intervalDur(arg.Column2))
		j.heartbeat = now
		return store.ClaimNextJobRow{ID: j.id, Kind: j.kind, Payload: j.payload, Attempts: j.attempts}, nil
	}
	return store.ClaimNextJobRow{}, pgx.ErrNoRows
}

func (f *fakeQueries) HeartbeatJob(_ context.Context, arg store.HeartbeatJobParams) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.heartbeats = append(f.heartbeats, arg)
	if f.hbErr != nil {
		return 0, f.hbErr
	}
	if len(f.hbScript) > 0 {
		rows := f.hbScript[0]
		f.hbScript = f.hbScript[1:]
		if rows == 1 {
			f.renewLocked(arg)
		}
		return rows, nil
	}
	return f.renewLocked(arg), nil
}

func (f *fakeQueries) InsertJob(_ context.Context, arg store.InsertJobParams) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.inserts = append(f.inserts, arg)
	f.seq++
	f.jobs = append(f.jobs, &fakeJob{id: f.seq, kind: arg.Kind, payload: arg.Payload, status: statusPending, createdAt: f.now()})
	return f.seq, nil
}

func (f *fakeQueries) MarkJobDone(_ context.Context, id int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.doneIDs = append(f.doneIDs, id)
	if j := f.byIDLocked(id); j != nil {
		j.status = statusDone
	}
	return nil
}

func (f *fakeQueries) MarkJobFailed(_ context.Context, arg store.MarkJobFailedParams) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failed = append(f.failed, arg)
	if j := f.byIDLocked(arg.ID); j != nil {
		j.status = statusFailed
		j.lastError = arg.LastError
	}
	return nil
}

// intervalDur 把 Postgres interval 入参还原成时长(fake 记账用)。
func intervalDur(i pgtype.Interval) time.Duration {
	return time.Duration(i.Microseconds) * time.Microsecond
}

// —— 队列主干:入队 → 认领 → 执行 → 终态 ——

func TestEnqueueMarshalsPayloadViaInsertJob(t *testing.T) {
	f := newFakeQueries()
	q := newQueue(f, nil)

	id, err := q.Enqueue(context.Background(), KindDeploy, DeployPayload{Project: "forge", Def: ".tally/deploy.yaml"})
	if err != nil {
		t.Fatalf("入队: %v", err)
	}
	if id != 1 {
		t.Errorf("job id = %d, want 1", id)
	}
	if len(f.inserts) != 1 {
		t.Fatalf("InsertJob 调用次数 = %d, want 1", len(f.inserts))
	}
	if got := f.inserts[0].Kind; got != string(KindDeploy) {
		t.Errorf("InsertJob kind = %q, want %q", got, KindDeploy)
	}
	var payload DeployPayload
	if err := json.Unmarshal(f.inserts[0].Payload, &payload); err != nil {
		t.Fatalf("payload 不是合法 JSON: %v", err)
	}
	if payload.Project != "forge" || payload.Def != ".tally/deploy.yaml" {
		t.Errorf("payload 内容不对: %+v", payload)
	}
}

// 认领 → 执行 → 成功落完成,是队列的主干回路;同时核对认领入参
// (worker 带非空 owner 标识、租约窗口 = 队列租约)。
func TestTakeOneClaimsRunsAndMarksDone(t *testing.T) {
	f := newFakeQueries()
	var inHandler fakeJob
	var gotPayload json.RawMessage
	handlerCalled := false
	q := newQueue(f, map[Kind]Handler{
		KindDiagnose: func(_ context.Context, payload json.RawMessage) error {
			handlerCalled = true
			gotPayload = payload
			inHandler = f.job(1) // 执行中的行状态
			return nil
		},
	}, WithLease(30*time.Second))

	f.seedPending(string(KindDiagnose), []byte(`{"fingerprint":"fp-1"}`))
	took, err := q.takeOne(context.Background())
	if err != nil {
		t.Fatalf("takeOne: %v", err)
	}
	if !took {
		t.Fatal("有 pending 任务,应被认领")
	}
	if !handlerCalled {
		t.Fatal("handler 未被执行")
	}
	if string(gotPayload) != `{"fingerprint":"fp-1"}` {
		t.Errorf("handler 收到的 payload = %s", gotPayload)
	}

	// 认领入参:owner 标识 + 租约窗口。
	if len(f.claims) != 1 {
		t.Fatalf("ClaimNextJob 调用次数 = %d, want 1", len(f.claims))
	}
	arg := f.claims[0]
	if !arg.WorkerID.Valid || arg.WorkerID.String == "" {
		t.Error("认领应携带非空的 worker 标识(owner)")
	}
	if arg.WorkerID.String != q.workerID {
		t.Errorf("认领 owner = %q, want 本队列的 %q", arg.WorkerID.String, q.workerID)
	}
	if arg.Column2.Microseconds != (30 * time.Second).Microseconds() {
		t.Errorf("认领租约窗口 = %dµs, want %dµs", arg.Column2.Microseconds, (30 * time.Second).Microseconds())
	}

	// 执行中:行已 running、归属本 worker、attempts 递增(认领即写租约)。
	if inHandler.status != statusRunning {
		t.Errorf("执行中 status = %q, want running", inHandler.status)
	}
	if inHandler.attempts != 1 {
		t.Errorf("执行中 attempts = %d, want 1(认领即递增)", inHandler.attempts)
	}
	if inHandler.workerID.String != q.workerID || !inHandler.workerID.Valid {
		t.Errorf("执行中 worker_id = %+v, want %q", inHandler.workerID, q.workerID)
	}

	// 成功:落完成。
	if f.doneIDs == nil || len(f.doneIDs) != 1 || f.doneIDs[0] != 1 {
		t.Errorf("MarkJobDone 记录 = %v, want [1]", f.doneIDs)
	}
	if len(f.failed) != 0 {
		t.Errorf("成功路径不应落失败: %+v", f.failed)
	}
}

func TestTakeOneNoClaimableJobReturnsFalse(t *testing.T) {
	f := newFakeQueries()
	q := newQueue(f, nil)

	took, err := q.takeOne(context.Background())
	if err != nil {
		t.Fatalf("空队列 takeOne 不应报错: %v", err)
	}
	if took {
		t.Error("空队列不应认领到任务")
	}
}

func TestTakeOneUnregisteredKindMarksFailed(t *testing.T) {
	f := newFakeQueries()
	q := newQueue(f, nil) // 空注册表

	f.seedPending("no_such_kind", []byte(`{}`))
	took, err := q.takeOne(context.Background())
	if err != nil || !took {
		t.Fatalf("took/err = %v/%v, want true/nil", took, err)
	}
	if len(f.failed) != 1 {
		t.Fatalf("MarkJobFailed 调用次数 = %d, want 1", len(f.failed))
	}
	if f.failed[0].ID != 1 || f.failed[0].LastError.String != "未注册的任务类型: no_such_kind" {
		t.Errorf("失败记录 = %+v", f.failed[0])
	}
	if got := f.job(1).status; got != statusFailed {
		t.Errorf("status = %q, want failed", got)
	}
}

func TestHandlerErrorMarksFailedWithMessage(t *testing.T) {
	f := newFakeQueries()
	q := newQueue(f, map[Kind]Handler{
		KindDiagnose: func(context.Context, json.RawMessage) error { return errors.New("boom") },
	})

	f.seedPending(string(KindDiagnose), []byte(`{}`))
	if _, err := q.takeOne(context.Background()); err != nil {
		t.Fatalf("takeOne: %v", err)
	}
	if len(f.failed) != 1 || f.failed[0].ID != 1 || f.failed[0].LastError.String != "boom" {
		t.Errorf("失败记录 = %+v, want id=1 last_error=boom", f.failed)
	}
	if len(f.doneIDs) != 0 {
		t.Error("失败路径不应落完成")
	}
}

// 队列关闭:Run 应随 ctx 取消返回,不遗留 goroutine 阻塞。
func TestRunStopsOnContextCancel(t *testing.T) {
	f := newFakeQueries()
	q := newQueue(f, nil, WithPollDelay(time.Millisecond))

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err := q.Run(ctx)
	if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Run 返回 %v, want 上下文取消", err)
	}
}
